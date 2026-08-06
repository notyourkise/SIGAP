package utils

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const tokenBlacklistKey = "sigap:blacklist:jwt"

// TokenBlacklist adalah kontrak penyimpanan token yang telah dicabut (revoked).
// Diimplementasikan dua varian:
//   - RedisTokenBlacklist — memakai Redis SET (SADD/SISMEMBER/SREM) dengan TTL
//     otomatis sesuai sisa masa hidup token (persisten lintas restart &
//     multi-instance). Cleanup goroutine tidak diperlukan.
//   - MemoryTokenBlacklist — peta in-memory + mutex; fallback saat Redis tidak
//     tersedia (development tanpa Docker).
//
// Interface dibuat agar mudah di-mock pada unit test (task verifikasi).
type TokenBlacklist interface {
	// Add menambahkan token ke daftar terlarang dengan waktu kedaluwarsa.
	Add(token string, expiry time.Time) error
	// IsRevoked memeriksa apakah token sudah dicabut.
	IsRevoked(token string) (bool, error)
	// Remove menghapus token dari daftar (mis. saat sesi dipulihkan kembali).
	Remove(token string) error
	// CleanupExpired membersihkan entri yang sudah kedaluwarsa (no-op saat TTL
	// Redis mengelola siklus hidup — hanya relevan untuk varian in-memory).
	CleanupExpired()
}

// redisCommander membatasi API go-redis yang dipakai blacklist agar mudah
// diganti mock pada unit test.
type redisCommander interface {
	SAdd(ctx context.Context, key string, members ...interface{}) *redis.IntCmd
	SIsMember(ctx context.Context, key string, member interface{}) *redis.BoolCmd
	SRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd
	TTL(ctx context.Context, key string) *redis.DurationCmd
	Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd
}

// RedisTokenBlacklist menyimpan token ter-revoke di Redis SET. TTL key mengikuti
// token dengan sisa masa hidup terpanjang agar tidak ada token aktif yang
// hilang lebih awal dari jadwalnya.
type RedisTokenBlacklist struct {
	client redisCommander
}

// NewRedisTokenBlacklist membuat blacklist berbasis Redis.
func NewRedisTokenBlacklist(client *redis.Client) *RedisTokenBlacklist {
	return &RedisTokenBlacklist{client: client}
}

func (b *RedisTokenBlacklist) Add(token string, expiry time.Time) error {
	ctx := context.Background()
	if err := b.client.SAdd(ctx, tokenBlacklistKey, token).Err(); err != nil {
		return fmt.Errorf("menambahkan token ke blacklist Redis: %w", err)
	}

	// TTL tombol mengikuti sisa masa hidup token TERPANJANG. Menambah token baru
	// tidak boleh mempersingkat TTL token lain yang sudah ada di dalam set.
	ttl := time.Until(expiry)
	if ttl <= 0 {
		return nil
	}
	current, err := b.client.TTL(ctx, tokenBlacklistKey).Result()
	if err != nil {
		return fmt.Errorf("membaca TTL blacklist Redis: %w", err)
	}
	if current < ttl {
		if err := b.client.Expire(ctx, tokenBlacklistKey, ttl).Err(); err != nil {
			return fmt.Errorf("mengatur TTL blacklist Redis: %w", err)
		}
	}
	return nil
}

func (b *RedisTokenBlacklist) IsRevoked(token string) (bool, error) {
	ok, err := b.client.SIsMember(context.Background(), tokenBlacklistKey, token).Result()
	if err != nil {
		return false, fmt.Errorf("memeriksa blacklist Redis: %w", err)
	}
	return ok, nil
}

func (b *RedisTokenBlacklist) Remove(token string) error {
	if err := b.client.SRem(context.Background(), tokenBlacklistKey, token).Err(); err != nil {
		return fmt.Errorf("menghapus token dari blacklist Redis: %w", err)
	}
	return nil
}

// CleanupExpired tidak melakukan apa-apa — Redis EXPIRE menangani TTL otomatis.
func (b *RedisTokenBlacklist) CleanupExpired() {}

// memoryTokenBlacklist adalah fallback in-memory (per-proses) memakai
// sync.RWMutex untuk keamanan concurrent access.
type memoryTokenBlacklist struct {
	mu     sync.RWMutex
	tokens map[string]time.Time // token -> waktu expiry
}

func newMemoryTokenBlacklist() *memoryTokenBlacklist {
	return &memoryTokenBlacklist{tokens: make(map[string]time.Time)}
}

func (b *memoryTokenBlacklist) Add(token string, expiry time.Time) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.tokens[token] = expiry
	return nil
}

func (b *memoryTokenBlacklist) IsRevoked(token string) (bool, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	_, exists := b.tokens[token]
	return exists, nil
}

func (b *memoryTokenBlacklist) Remove(token string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.tokens, token)
	return nil
}

func (b *memoryTokenBlacklist) CleanupExpired() {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	for token, expiry := range b.tokens {
		if now.After(expiry) {
			delete(b.tokens, token)
		}
	}
}

// blacklistStore adalah backend aktif yang dipakai seluruh modul. Default
// memakai in-memory; saat startup main.go menggantinya dengan varian Redis
// bila koneksi tersedia (config.ConnectRedis → utils.SetTokenBlacklist).
var blacklistStore TokenBlacklist = newMemoryTokenBlacklist()

// SetTokenBlacklist mengganti backend blacklist (dipanggil dari main.go).
func SetTokenBlacklist(b TokenBlacklist) {
	blacklistStore = b
}

// RevokeToken menambahkan token ke blacklist dengan waktu expiry-nya.
func RevokeToken(tokenString string, expiry time.Time) {
	if err := blacklistStore.Add(tokenString, expiry); err != nil {
		Log.Warnf("⚠️ Gagal menambahkan token ke blacklist: %v", err)
	}
}

// IsTokenRevoked memeriksa apakah token sudah direvoke/blacklisted.
func IsTokenRevoked(tokenString string) bool {
	revoked, err := blacklistStore.IsRevoked(tokenString)
	if err != nil {
		Log.Warnf("⚠️ Gagal memeriksa status blacklist: %v", err)
		return false
	}
	return revoked
}

// StartBlacklistCleanup menjalankan goroutine pembersihan token expired.
// Relevan hanya untuk backend in-memory; saat Redis aktif CleanupExpired
// adalah no-op karena TTL otomatis yang mengelola siklus hidup token.
func StartBlacklistCleanup(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			cleanupExpiredTokens()
		}
	}()
}

// cleanupExpiredTokens membersihkan token kadaluarsa pada backend aktif.
func cleanupExpiredTokens() {
	blacklistStore.CleanupExpired()
}
