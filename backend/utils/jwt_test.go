package utils

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func TestGenerateAndValidateToken_Success(t *testing.T) {
	t.Setenv("JWT_SECRET", "sigap_test_jwt_secret_key_2026")
	userID := "12345-abcde"
	role := "admin"

	// 1. Uji Pembuatan Token
	tokenString, err := GenerateToken(userID, role)
	assert.NoError(t, err)
	assert.NotEmpty(t, tokenString)

	// 2. Uji Validasi Token
	token, err := ValidateToken(tokenString)
	assert.NoError(t, err)
	assert.True(t, token.Valid)

	// 3. Uji Kesesuaian Klaim
	claims, ok := token.Claims.(jwt.MapClaims)
	assert.True(t, ok)
	assert.Equal(t, userID, claims["user_id"])
	assert.Equal(t, role, claims["role"])
	// Versi token default adalah 1
	assert.Equal(t, float64(1), claims["tv"])
}

func TestGenerateToken_WithTokenVersion(t *testing.T) {
	t.Setenv("JWT_SECRET", "sigap_test_jwt_secret_key_2026")
	tokenString, err := GenerateToken("user-1", "teknisi", 5)
	assert.NoError(t, err)

	token, err := ValidateToken(tokenString)
	assert.NoError(t, err)
	claims, ok := token.Claims.(jwt.MapClaims)
	assert.True(t, ok)
	assert.Equal(t, float64(5), claims["tv"])
}

func TestValidateToken_InvalidSignature(t *testing.T) {
	t.Setenv("JWT_SECRET", "sigap_test_jwt_secret_key_2026")
	// Token buatan manual dengan secret key yang salah
	claims := jwt.MapClaims{
		"user_id": "test-user",
		"role":    "teknisi",
		"exp":     time.Now().Add(time.Hour).Unix(),
	}
	badToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := badToken.SignedString([]byte("wrong_secret_key"))
	assert.NoError(t, err)

	// Validasi token yang dibuat dengan secret key salah
	parsedToken, err := ValidateToken(tokenString)
	assert.Error(t, err)
	assert.False(t, parsedToken.Valid)
}

func TestValidateToken_Expired(t *testing.T) {
	t.Setenv("JWT_SECRET", "sigap_test_jwt_secret_key_2026")
	// Token yang sudah kadaluarsa (exp set di masa lalu)
	claims := jwt.MapClaims{
		"user_id": "test-user",
		"role":    "teknisi",
		"exp":     time.Now().Add(-time.Hour).Unix(), // -1 jam
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(getJWTSecret())
	assert.NoError(t, err)

	// Validasi token yang kadaluarsa
	parsedToken, err := ValidateToken(tokenString)
	assert.Error(t, err) // Harus mengembalikan error expired
	if parsedToken != nil {
		assert.False(t, parsedToken.Valid)
	}
}
