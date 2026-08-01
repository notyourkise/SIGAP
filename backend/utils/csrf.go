package utils

import (
	"crypto/rand"
	"encoding/hex"
	"strings"

	"github.com/gofiber/fiber/v2"
)

const (
	// SessionCookieName nama cookie httpOnly yang menyimpan JWT sesi.
	// Token tidak lagi diekspos ke JavaScript (migrasi dari localStorage).
	SessionCookieName = "sigap_session"
	// CSRFCookieName nama cookie CSRF (non-httpOnly) untuk pola double-submit.
	CSRFCookieName = "sigap_csrf"
	// CSRFHeaderName nama header untuk mengirim token CSRF.
	CSRFHeaderName = "X-CSRF-Token"
)

// GenerateCSRFToken menghasilkan token CSRF acak 32-byte (hex).
func GenerateCSRFToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

// ExtractTokenString mengambil token JWT dari cookie sesi terlebih dahulu,
// lalu fallback ke header Authorization Bearer (untuk client API/Swagger).
func ExtractTokenString(c *fiber.Ctx) string {
	if cookieToken := c.Cookies(SessionCookieName); cookieToken != "" {
		return cookieToken
	}
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return ""
	}
	parts := strings.Split(authHeader, " ")
	if len(parts) == 2 && parts[0] == "Bearer" {
		return parts[1]
	}
	return ""
}
