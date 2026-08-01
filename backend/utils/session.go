package utils

import (
	"os"

	"github.com/gofiber/fiber/v2"
)

// SetSessionCookie memasang cookie httpOnly yang menyimpan token JWT sesi.
// Dipakai oleh Login, ChangePassword, dan middleware Protected (sliding session).
func SetSessionCookie(c *fiber.Ctx, token string) {
	c.Cookie(&fiber.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		HTTPOnly: true,
		Secure:   os.Getenv("APP_ENV") == "production",
		SameSite: "Lax",
		Path:     "/",
		MaxAge:   24 * 60 * 60,
	})
}

// SetCSRFCookie memasang cookie CSRF (non-httpOnly) untuk pola double-submit.
func SetCSRFCookie(c *fiber.Ctx, token string) {
	c.Cookie(&fiber.Cookie{
		Name:     CSRFCookieName,
		Value:    token,
		HTTPOnly: false,
		Secure:   os.Getenv("APP_ENV") == "production",
		SameSite: "Lax",
		Path:     "/",
		MaxAge:   24 * 60 * 60,
	})
}

// ClearSessionCookies menghapus cookie sesi & CSRF dari browser klien.
func ClearSessionCookies(c *fiber.Ctx) {
	c.ClearCookie(SessionCookieName, CSRFCookieName)
}
