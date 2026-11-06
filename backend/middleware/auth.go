package middleware

import (
	"strings"

	"sigap-backend/config"
	"sigap-backend/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// isAllowedWhenMustChangePassword memeriksa apakah path request diizinkan untuk
// pengguna dengan status must_change_password = true.
// Pengguna hanya diizinkan mengakses profil (/auth/me, /auth/profile),
// penggantian kata sandi (/auth/change-password), dan logout (/auth/logout).
func isAllowedWhenMustChangePassword(path string) bool {
	cleanPath := strings.TrimRight(path, "/")
	if cleanPath == "" {
		cleanPath = "/"
	}

	allowedPaths := []string{
		"/auth/me",
		"/api/auth/me",
		"/auth/change-password",
		"/api/auth/change-password",
		"/auth/logout",
		"/api/auth/logout",
		"/auth/profile",
		"/api/auth/profile",
	}

	for _, allowed := range allowedPaths {
		if cleanPath == allowed {
			return true
		}
	}
	return false
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
		mustChangePassword, err := utils.ValidateTokenVersionAndMustChange(config.DB, token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "Sesi telah dicabut. Silakan login kembali.",
			})
		}

		// Simpan data user ke Locals context agar bisa dibaca di handler berikutnya
		c.Locals("user_id", claims["user_id"])
		c.Locals("role", claims["role"])
		c.Locals("must_change_password", mustChangePassword)

		// SECURITY: Pengguna yang berstatus must_change_password wajib mengganti sandi
		// terlebih dahulu sebelum dapat mengakses API operasional.
		if mustChangePassword && !isAllowedWhenMustChangePassword(c.Path()) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"status":  "error",
				"message": "Anda wajib mengganti password sebelum dapat mengakses resource ini",
			})
		}

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
