package utils

import (
	"errors"
	"time"

	"sigap-backend/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// getJWTSecret mengembalikan kunci JWT dari Docker secret (file /run/secrets/jwt_secret)
// dengan fallback ke environment variable JWT_SECRET.
// Tidak ada fallback hardcoded: kunci wajib disediakan (divalidasi di startup oleh RequireSecret).
// Mengembalikan slice kosong jika tidak diset agar signing/validasi gagal alih-alih memakai kunci publik yang diketahui.
func getJWTSecret() []byte {
	return []byte(GetSecret("JWT_SECRET", "/run/secrets/jwt_secret"))
}

// GenerateToken membuat token JWT baru berisi user_id, role, dan versi token.
// tokenVersion bersifat opsional; bila tidak diberikan, default 1.
func GenerateToken(userID string, role string, tokenVersion ...int64) (string, error) {
	version := int64(1)
	if len(tokenVersion) > 0 {
		version = tokenVersion[0]
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"user_id": userID,
		"role":    role,
		"tv":      version,
		"jti":     uuid.NewString(),
		"iat":     now.Unix(),
		"exp":     now.Add(time.Hour * 5).Unix(), // Berlaku 5 jam kerja
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(getJWTSecret())
}

// ValidateToken memvalidasi token string dan mengembalikan parsed Token
func ValidateToken(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validasi metode signing
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("metode signing token tidak valid")
		}
		return getJWTSecret(), nil
	})
}

// ValidateTokenVersion memastikan versi token pada klaim ("tv") cocok dengan
// versi token pengguna di database. Jika tidak cocok, token dianggap dicabut
// (mis. saat password di-reset atau role diubah oleh Admin).
func ValidateTokenVersion(db *gorm.DB, token *jwt.Token) error {
	_, err := ValidateTokenVersionAndMustChange(db, token)
	return err
}

// ValidateTokenVersionAndMustChange memastikan versi token pada klaim ("tv") cocok dengan
// versi token pengguna di database dan mengembalikan status must_change_password.
func ValidateTokenVersionAndMustChange(db *gorm.DB, token *jwt.Token) (bool, error) {
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return false, errors.New("klaim token tidak valid")
	}

	userID, ok := claims["user_id"].(string)
	if !ok || userID == "" {
		return false, errors.New("user_id tidak ditemukan dalam token")
	}

	tokenVersion, _ := claims["tv"].(float64)

	var user models.User
	if err := db.Select("token_version", "must_change_password").First(&user, "id = ?", userID).Error; err != nil {
		return false, errors.New("pengguna tidak ditemukan")
	}

	if int64(tokenVersion) != user.TokenVersion {
		return false, errors.New("token telah dicabut")
	}

	return user.MustChangePassword, nil
}
