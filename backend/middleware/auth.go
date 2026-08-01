package middleware

import (
	"sigap-backend/config"
	"sigap-backend/utils"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// slidingSessionThreshold menentukan berapa lama sisa masa berlaku token yang
// memicu perpanjangan otomatis (sliding session). Bila tersisa kurang dari
// 1 jam, token & cookie sesi diperbarui sehingga pengguna aktif tidak keluar.
const slidingSessionThreshold = 1 * time.Hour

// extendSessionIfNearExpiry memperbarui sesi (sliding session) bila token
// mendekati kedaluwarsa. Token baru memakai versi token yang sama agar token
// lama yang beredar tetap valid hingga kedaluwarsanya.
func extendSessionIfNearExpiry(c *fiber.Ctx, claims jwt.MapClaims) {
	expFloat, ok := claims["exp"].(float64)
	if !ok {
		return
	}
	if remaining := time.Until(time.Unix(int64(expFloat), 0)); remaining > slidingSessionThreshold {
		return
	}

	userID, _ := claims["user_id"].(string)
	role, _ := claims["role"].(string)
	var tokenVersion int64 = 1
	if tv, ok := claims["tv"].(float64); ok {
		tokenVersion = int64(tv)
	}

	newToken, err := utils.GenerateToken(userID, role, tokenVersion)
	if err != nil {
		return
	}
	utils.SetSessionCookie(c, newToken)
}

// Protected adalah middleware untuk memvalidasi token JWT.
// Token diambil dari cookie httpOnly sigap_session (mode web), dengan
// fallback ke header Authorization Bearer (client API / Swagger).
func Protected() fiber.Handler {
	return func(c *fiber.Ctx) error {
		tokenString := utils.ExtractTokenString(c)
		if tokenString == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "Token otentikasi tidak ditemukan",
			})
		}

		// SECURITY: Periksa apakah token sudah direvoke (user sudah logout)
		if utils.IsTokenRevoked(tokenString) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "Token telah dicabut. Silakan login kembali.",
			})
		}

		token, err := utils.ValidateToken(tokenString)
		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "Token tidak valid atau telah kedaluwarsa",
			})
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "Klaim token tidak valid",
			})
		}

		// SECURITY: Pastikan versi token cocok dengan versi token terbaru pengguna.
		// Versi token dinaikkan saat password di-reset atau role diubah, sehingga
		// seluruh token lama langsung dicabut.
		if err := utils.ValidateTokenVersion(config.DB, token); err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "Sesi telah dicabut. Silakan login kembali.",
			})
		}

		// Fase 3.2 — Sliding session: perpanjang otomatis bila token mendekati
		// kedaluwarsa (sisa < 1 jam) sehingga pengguna aktif tidak keluar paksa.
		extendSessionIfNearExpiry(c, claims)

		// Simpan data user ke Locals context agar bisa dibaca di handler berikutnya
		c.Locals("user_id", claims["user_id"])
		c.Locals("role", claims["role"])

		return c.Next()
	}
}

// RequireRole adalah middleware untuk membatasi akses berdasarkan peran pengguna
func RequireRole(allowedRoles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userRole, ok := c.Locals("role").(string)
		if !ok || userRole == "" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"status":  "error",
				"message": "Akses ditolak: Identitas peran tidak ditemukan",
			})
		}

		// Periksa apakah role user terdaftar dalam role yang diizinkan
		roleAllowed := false
		for _, role := range allowedRoles {
			if userRole == role {
				roleAllowed = true
				break
			}
		}

		if !roleAllowed {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"status":  "error",
				"message": "Akses ditolak: Peran Anda tidak memiliki izin untuk mengakses resource ini",
			})
		}

		return c.Next()
	}
}
