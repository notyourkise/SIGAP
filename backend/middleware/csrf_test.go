package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"sigap-backend/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

// csrfTestApp membuat app Fiber dengan middleware CSRFProtection dan handler
// yang hanya bisa dipanggil bila middleware meloloskan request.
func csrfTestApp() *fiber.App {
	app := fiber.New()
	app.Use(CSRFProtection())
	app.Post("/mutate", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})
	app.Post("/api/auth/login", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})
	app.Post("/api/auth/logout", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})
	app.Get("/read", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})
	app.Options("/read", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})
	return app
}

func TestCSRF_SafeMethodsPass(t *testing.T) {
	app := csrfTestApp()

	for _, method := range []string{"GET", "HEAD", "OPTIONS"} {
		resp, err := app.Test(httptest.NewRequest(method, "/read", nil))
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode, "metode %s harus dilewati CSRF", method)
	}
}

func TestCSRF_BearerTokenPass(t *testing.T) {
	app := csrfTestApp()

	// Client API memakai Bearer token → dilewati tanpa token CSRF
	req := httptest.NewRequest("POST", "/mutate", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer some-jwt-token")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestCSRF_NoSessionCookiePass(t *testing.T) {
	app := csrfTestApp()

	// Request publik (belum login) tanpa cookie sesi → tidak perlu token CSRF
	req := httptest.NewRequest("POST", "/mutate", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestCSRF_LoginEndpointPassWithStaleSessionCookie(t *testing.T) {
	app := csrfTestApp()

	// Login bersifat publik: sisa cookie sesi lama dari sesi sebelumnya
	// TIDAK boleh memicu pengecekan CSRF pada percobaan login.
	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: utils.SessionCookieName, Value: "stale-session"})

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestCSRF_LogoutEndpointPassWithStaleSessionCookie(t *testing.T) {
	app := csrfTestApp()

	// Logout harus tetap dapat membersihkan sesi ketika cookie CSRF stale/hilang.
	req := httptest.NewRequest("POST", "/api/auth/logout", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: utils.SessionCookieName, Value: "stale-session"})

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestCSRF_MismatchTokenForbidden(t *testing.T) {
	app := csrfTestApp()

	req := httptest.NewRequest("POST", "/mutate", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: utils.SessionCookieName, Value: "valid-session"})
	req.AddCookie(&http.Cookie{Name: utils.CSRFCookieName, Value: "abc123"})
	req.Header.Set(utils.CSRFHeaderName, "def456")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "error", result["status"])
}

func TestCSRF_MissingTokenForbidden(t *testing.T) {
	app := csrfTestApp()

	req := httptest.NewRequest("POST", "/mutate", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: utils.SessionCookieName, Value: "valid-session"})

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}

func TestCSRF_MatchingTokenPass(t *testing.T) {
	app := csrfTestApp()

	req := httptest.NewRequest("POST", "/mutate", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: utils.SessionCookieName, Value: "valid-session"})
	req.AddCookie(&http.Cookie{Name: utils.CSRFCookieName, Value: "same-token"})
	req.Header.Set(utils.CSRFHeaderName, "same-token")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}
