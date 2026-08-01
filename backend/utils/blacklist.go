package utils

import (
	"sync"
	"time"
)

// tokenBlacklist menyimpan token yang sudah di-revoke beserta waktu expiry-nya.
// Menggunakan sync.RWMutex untuk keamanan concurrent access.
// Catatan: blacklist ini bersifat in-memory dan akan reset saat server restart.
// Untuk deployment multi-instance, pertimbangkan Redis sebagai shared store.
type tokenBlacklistStore struct {
	mu     sync.RWMutex
	tokens map[string]time.Time // token -> waktu expiry
}

var blacklist = &tokenBlacklistStore{
	tokens: make(map[string]time.Time),
}

// RevokeToken menambahkan token ke blacklist dengan waktu expiry-nya.
// Token yang sudah expire akan dibersihkan secara otomatis oleh cleanup goroutine.
func RevokeToken(tokenString string, expiry time.Time) {
	blacklist.mu.Lock()
	defer blacklist.mu.Unlock()
	blacklist.tokens[tokenString] = expiry
}

// IsTokenRevoked memeriksa apakah token sudah direvoke/blacklisted.
func IsTokenRevoked(tokenString string) bool {
	blacklist.mu.RLock()
	defer blacklist.mu.RUnlock()
	_, exists := blacklist.tokens[tokenString]
	return exists
}

// StartBlacklistCleanup menjalankan goroutine latar belakang yang membersihkan
// token expired dari blacklist setiap interval tertentu untuk mencegah memory leak.
func StartBlacklistCleanup(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			cleanupExpiredTokens()
		}
	}()
}

// cleanupExpiredTokens menghapus semua token yang sudah melewati waktu expiry-nya.
func cleanupExpiredTokens() {
	blacklist.mu.Lock()
	defer blacklist.mu.Unlock()
	now := time.Now()
	for token, expiry := range blacklist.tokens {
		if now.After(expiry) {
			delete(blacklist.tokens, token)
		}
	}
}
