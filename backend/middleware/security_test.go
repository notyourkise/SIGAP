package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestSecurityHeaders_Development(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	app := fiber.New()
	app.Use(SecurityHeaders())
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	resp, err := app.Test(httptest.NewRequest("GET", "/", nil))
	assert.NoError(t, err)

	// Header keamanan dasar selalu ada
	assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", resp.Header.Get("X-Frame-Options"))
	assert.Equal(t, "1; mode=block", resp.Header.Get("X-XSS-Protection"))
	assert.Equal(t, "SIGAP", resp.Header.Get("Server"))
	assert.Equal(t, "strict-origin-when-cross-origin", resp.Header.Get("Referrer-Policy"))
	assert.Equal(t, "none", resp.Header.Get("X-Permitted-Cross-Domain-Policies"))

	// CSP strict & HSTS hanya untuk produksi
	assert.Empty(t, resp.Header.Get("Content-Security-Policy"))
	assert.Empty(t, resp.Header.Get("Strict-Transport-Security"))
}

func TestSecurityHeaders_Production(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	app := fiber.New()
	app.Use(SecurityHeaders())
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	resp, err := app.Test(httptest.NewRequest("GET", "/", nil))
	assert.NoError(t, err)

	assert.NotEmpty(t, resp.Header.Get("Content-Security-Policy"))
	assert.Equal(t, "max-age=31536000; includeSubDomains", resp.Header.Get("Strict-Transport-Security"))
}
