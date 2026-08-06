package config

import (
	"context"
	"os"
	"time"

	"sigap-backend/utils"

	"github.com/redis/go-redis/v9"
)

// RedisClient adalah client go-redis global. Nil bila Redis tidak tersedia
// (fallback in-memory untuk rate-limit & blacklist — development tanpa Docker).
var RedisClient *redis.Client

// RedisEnabled menandakan apakah Redis aktif (shared store multi-instance).
var RedisEnabled bool

// ConnectRedis membangun koneksi ke Redis dari environment variables
// REDIS_HOST / REDIS_PORT / REDIS_PASSWORD (password opsional). Default host
// adalah "redis" (nama service di docker-compose) dan port 6379.
//
// Graceful fallback: bila Redis tidak dapat dijangkau (mis. development tanpa
// Docker), koneksi gagal hanya menimbulkan LOG WARNING — aplikasi tetap jalan
// memakai in-memory store untuk rate-limit & blacklist (jangan fail-fast).
func ConnectRedis() {
	host := os.Getenv("REDIS_HOST")
	if host == "" {
		host = "redis"
	}
	port := os.Getenv("REDIS_PORT")
	if port == "" {
		port = "6379"
	}
	addr := host + ":" + port

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: os.Getenv("REDIS_PASSWORD"), // opsional; kosong = tanpa AUTH
		DB:       0,
	})

	// Verifikasi koneksi dengan Ping ber-timeout singkat agar startup tidak
	// terblokir lama ketika Redis tidak tersedia.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		utils.Log.Warnf("⚠️ Redis tidak tersedia di %s: %v — fallback in-memory untuk rate-limit & blacklist", addr, err)
		_ = client.Close()
		RedisClient = nil
		RedisEnabled = false
		return
	}

	RedisClient = client
	RedisEnabled = true
	utils.Log.Infof("🔄 Koneksi Redis berhasil (%s)", addr)
}
