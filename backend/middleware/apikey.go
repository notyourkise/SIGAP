package middleware

import (
	"context"
	"strings"
	"time"

	"sigap-backend/config"
	"sigap-backend/models"
	"sigap-backend/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

// HeaderAPIKey adalah nama header untuk API key.
const HeaderAPIKey = "X-API-Key"

// APIKeyRateLimit adalah batas request per menit per API key (Task 12.1).
const APIKeyRateLimit = 100

// apiKeyRateLimiter memakai counter Redis INCR/EXPIRE (sama seperti limiter.go)
// dengan key "apikey-limiter:{keyID}:{minute}". Fallback in-memory bila Redis
// tidak tersedia.
type apiKeyRateLimiter struct {
	client *redis.Client
}

func (r *apiKeyRateLimiter) Increment(key string, window time.Duration) (int64, error) {
	ctx := context.Background()
	n, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	if n == 1 {
		_ = r.client.Expire(ctx, key, window).Err()
	}
	return n, nil
}

// memoryAPIKeyLimiter adalah fallback in-memory per-proses.
type memoryAPIKeyLimiter struct {
	mu       chan struct{}
	counters map[string]*memoryCounterEntry
}

func newMemoryAPIKeyLimiter() *memoryAPIKeyLimiter {
	return &memoryAPIKeyLimiter{
		mu:       make(chan struct{}, 1),
		counters: make(map[string]*memoryCounterEntry),
	}
}

func (m *memoryAPIKeyLimiter) Increment(key string, window time.Duration) (int64, error) {
	m.mu <- struct{}{}
	defer func() { <-m.mu }()
	now := time.Now()
	entry, exists := m.counters[key]
	if !exists || now.After(entry.expiresAt) {
		entry = &memoryCounterEntry{count: 0, expiresAt: now.Add(window)}
		m.counters[key] = entry
	}
	entry.count++
	return entry.count, nil
}

// apiKeyLimiter memilih backend limiter API key (Redis atau in-memory) saat
// pertama kali digunakan. Redis dipakai bila tersedia (shared multi-instance).
var apiKeyLimiterMem *memoryAPIKeyLimiter

// incrementAPIKeyLimiter menaikkan counter rate limit API key dan mengembalikan
// jumlah request pada window berjalan.
func incrementAPIKeyLimiter(keyID string) (int64, error) {
	windowKey := "apikey-limiter:" + keyID + ":" + time.Now().Format("2006-01-02T15:04")
	if config.RedisEnabled && config.RedisClient != nil {
		r := &apiKeyRateLimiter{client: config.RedisClient}
		return r.Increment(windowKey, time.Minute)
	}
	if apiKeyLimiterMem == nil {
		apiKeyLimiterMem = newMemoryAPIKeyLimiter()
	}
	return apiKeyLimiterMem.Increment(windowKey, time.Minute)
}

// authenticateAPIKey memvalidasi key mentah dari header X-API-Key, memuat
// record APIKey dari DB, mengecek revoked/expired, scope, dan rate limit.
// Bila valid, mengisi c.Locals dengan identitas API key dan melanjutkan.
func authenticateAPIKey(c *fiber.Ctx, requiredScope string) error {
	raw := strings.TrimSpace(c.Get(HeaderAPIKey))
	if raw == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Header X-API-Key tidak ditemukan",
		})
	}

	// Bcrypt hash tidak bisa di-query langsung, jadi muat kandidat (non-revoked,
	// non-expired) lalu bandingkan hash.
	var keys []models.APIKey
	now := time.Now()
	if err := config.DB.Where("revoked_at IS NULL AND (expires_at IS NULL OR expires_at > ?)", now).
		Find(&keys).Error; err != nil {
		utils.Log.Errorf("❌ Gagal memuat API key: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal memverifikasi API key",
		})
	}

	var matched *models.APIKey
	for i := range keys {
		if bcrypt.CompareHashAndPassword([]byte(keys[i].KeyHash), []byte(raw)) == nil {
			matched = &keys[i]
			break
		}
	}
	if matched == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "API key tidak valid",
		})
	}

	// Cek scope bila disyaratkan.
	if requiredScope != "" {
		hasScope := false
		for _, s := range matched.Scopes {
			if s == requiredScope {
				hasScope = true
				break
			}
		}
		if !hasScope {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"status":  "error",
				"message": "API key tidak memiliki izin (scope) untuk resource ini",
			})
		}
	}

	// Rate limit per key.
	count, err := incrementAPIKeyLimiter(matched.ID.String())
	if err != nil {
		// Fail-open ringan: kegagalan counter tidak memblokir layanan.
		utils.Log.Warnf("⚠️ API key limiter gagal INCR: %v", err)
	} else if count > APIKeyRateLimit {
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"status":  "error",
			"message": "API key melebihi batas permintaan (100/menit)",
		})
	}

	// Catat last_used_at. Dilakukan sinkron (bukan goroutine) untuk menghindari
	// race/panic bila koneksi DB berubah antara pemanggilan handler dan eksekusi
	// goroutine (mis. di lingkungan uji). Kegagalan update tidak memblokir layanan.
	if config.DB != nil {
		if err := config.DB.Model(&models.APIKey{}).Where("id = ?", matched.ID).
			Update("last_used_at", time.Now()).Error; err != nil {
			utils.Log.Warnf("⚠️ Gagal memperbarui last_used_at API key: %v", err)
		}
	}

	// Set identitas untuk handler.
	c.Locals("api_key_id", matched.ID.String())
	c.Locals("api_key_name", matched.Name)
	c.Locals("auth_method", "api_key")
	return c.Next()
}

// APIKeyAuth memvalidasi request melalui API key (strict). Tanpa key valid →
// 401. Dipakai endpoint yang hanya bisa diakses via API key.
func APIKeyAuth(requiredScope string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return authenticateAPIKey(c, requiredScope)
	}
}

// APIKeyOrProtected mengizinkan akses via API key ATAU sesi JWT (web).
// Bila header X-API-Key ada, autentikasi via API key; selain itu gunakan
// Protected (JWT). Dipakai endpoint yang mendukung kedua jalur (GET /api/cctv).
func APIKeyOrProtected(requiredScope string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		raw := strings.TrimSpace(c.Get(HeaderAPIKey))
		if raw != "" {
			return authenticateAPIKey(c, requiredScope)
		}
		return Protected()(c)
	}
}
