package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestBlacklist_RevokeAndCheck memverifikasi bahwa token yang di-revoke
// terdeteksi oleh IsTokenRevoked, sementara token lain tidak terpengaruh.
func TestBlacklist_RevokeAndCheck(t *testing.T) {
	tokenA := "token-aaa"
	tokenB := "token-bbb"

	assert.False(t, IsTokenRevoked(tokenA))
	assert.False(t, IsTokenRevoked(tokenB))

	RevokeToken(tokenA, time.Now().Add(24*time.Hour))

	assert.True(t, IsTokenRevoked(tokenA))
	assert.False(t, IsTokenRevoked(tokenB))
}

func TestBlacklist_CleanupExpiredTokens(t *testing.T) {
	expiredToken := "token-expired"
	validToken := "token-valid"

	// Revoke dengan expiry di masa lalu → harus dibersihkan
	RevokeToken(expiredToken, time.Now().Add(-time.Minute))
	RevokeToken(validToken, time.Now().Add(24*time.Hour))

	assert.True(t, IsTokenRevoked(expiredToken))

	cleanupExpiredTokens()

	assert.False(t, IsTokenRevoked(expiredToken), "token yang sudah expire harus dibersihkan")
	assert.True(t, IsTokenRevoked(validToken), "token yang belum expire harus tetap ada")
}
