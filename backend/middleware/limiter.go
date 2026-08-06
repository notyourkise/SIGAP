package middleware

import (
	"context"
	"sync"
	"time"

	"sigap-backend/config"
	"sigap-backend/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

// getClientIP mengembalikan IP klien yang sebenarnya.
// Hanya membaca header proxy (X-Real-IP / X-Forwarded-For) bila koneksi datang
// dari trusted proxy yang dikonfigurasi di main.go (app.SetTrustedProxies).
// Tanpa trusted proxy, c.IP() mengembalikan IP peer langsung sehingga header
// yang dipalsukan klien tidak berpengaruh (anti-spoofing).
func getClientIP(c *fiber.Ctx) string {
	return c.IP()
}

// limiterKey menyusun key rate limit dengan format "limiter:{ip}:{endpoint}".
// Endpoint memakai pola route (mis. /api/auth/login) agar seluruh request ke
// satu endpoint dihitung dalam satu bucket; fallback ke URL path aktual.
func limiterKey(c *fiber.Ctx) string {
	endpoint := c.Path()
	if r := c.Route(); r != nil && r.Path != "" {
		endpoint = r.Path
	}
	return "limiter:" + getClientIP(c) + ":" + endpoint
}

// rateLimitCounter adalah kontrak counter rate limit per key.
// Redis: INCR + EXPIRE (window). In-memory: peta + mutex (fallback).
type rateLimitCounter interface {
	// Increment menaikkan counter key dan mengembalikan nilai setelah INCR.
	// Window diterapkan sebagai TTL bila key baru dibuat.
	Increment(key string, window time.Duration) (int64, error)
}

// redisRateLimitCounter memakai operasi Redis INCR/EXPIRE — counter dibagikan
// antar instance API (multi-instance support).
type redisRateLimitCounter struct {
	client *redis.Client
}

func (r *redisRateLimitCounter) Increment(key string, window time.Duration) (int64, error) {
	ctx := context.Background()
	n, err := r.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, err
	}
	// Key baru (INCR pertama) langsung diberi TTL = window rate limit.
	if n == 1 {
		_ = r.client.Expire(ctx, key, window).Err()
	}
	return n, nil
}

// memoryCounterEntry menyimpan nilai counter + waktu reset window.
type memoryCounterEntry struct {
	count     int64
	expiresAt time.Time
}

// memoryRateLimitCounter adalah fallback in-memory per-proses (satu instance).
type memoryRateLimitCounter struct {
	mu       sync.Mutex
	counters map[string]*memoryCounterEntry
}

func newMemoryRateLimitCounter() *memoryRateLimitCounter {
	return &memoryRateLimitCounter{counters: make(map[string]*memoryCounterEntry)}
}

func (m *memoryRateLimitCounter) Increment(key string, window time.Duration) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	entry, exists := m.counters[key]
	if !exists || now.After(entry.expiresAt) {
		entry = &memoryCounterEntry{count: 0, expiresAt: now.Add(window)}
		m.counters[key] = entry
	}
	entry.count++
	return entry.count, nil
}

// limiterStore adalah backend counter aktif. Dipilih sekali saat pertama kali
// digunakan: Redis bila tersedia (shared multi-instance), else in-memory.
var (
	limiterStore     rateLimitCounter
	limiterStoreOnce sync.Once
)

func getLimiterStore() rateLimitCounter {
	limiterStoreOnce.Do(func() {
		if config.RedisEnabled && config.RedisClient != nil {
			limiterStore = &redisRateLimitCounter{client: config.RedisClient}
			utils.Log.Info("⚡ Rate limiter memakai Redis (shared lintas instance)")
		} else {
			limiterStore = newMemoryRateLimitCounter()
			utils.Log.Warn("⚠️ Rate limiter memakai in-memory (Redis tidak tersedia)")
		}
	})
	return limiterStore
}

// newFixedWindowLimiter membangun handler rate limit fixed-window berbasis
// counter per key "limiter:{ip}:{endpoint}".
func newFixedWindowLimiter(max int, window time.Duration, message string) fiber.Handler {
	store := getLimiterStore()
	return func(c *fiber.Ctx) error {
		count, err := store.Increment(limiterKey(c), window)
		if err != nil {
			// Kegagalan counter tidak boleh memblokir layanan (fail-open ringan);
			// cukup dicatat agar masalah infra terlihat.
			utils.Log.Warnf("⚠️ Rate limiter gagal INCR key %s: %v", limiterKey(c), err)
			return c.Next()
		}
		if count > int64(max) {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"status":  "error",
				"message": message,
			})
		}
		return c.Next()
	}
}

// GlobalLimiter membatasi maksimal 100 request per menit per IP per endpoint
// (key "limiter:{ip}:{endpoint}").
func GlobalLimiter() fiber.Handler {
	return newFixedWindowLimiter(100, 1*time.Minute, "Terlalu banyak permintaan. Silakan coba lagi beberapa saat lagi.")
}

// AuthLimiter membatasi maksimal 5 request per menit per IP khusus rute login
// (key "limiter:{ip}:{endpoint}" → endpoint /api/auth/login).
// Melindungi dari serangan brute force percobaan kata sandi.
func AuthLimiter() fiber.Handler {
	return newFixedWindowLimiter(5, 1*time.Minute, "Terlalu banyak percobaan login. Silakan coba lagi setelah 1 menit.")
}
