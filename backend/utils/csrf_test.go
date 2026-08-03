package utils

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestGenerateCSRFToken_LengthAndUniqueness(t *testing.T) {
	t1 := GenerateCSRFToken()
	t2 := GenerateCSRFToken()

	// 32 byte hex = 64 karakter
	assert.Len(t, t1, 64)
	assert.Len(t, t2, 64)
	assert.NotEqual(t, t1, t2, "token CSRF harus acak/unik")
}

func TestExtractTokenString_CookiePriority(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"token": ExtractTokenString(c)})
	})

	// Cookie sesi dan Bearer header sama-sama ada → cookie yang diprioritaskan
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "cookie-token"})
	req.Header.Set("Authorization", "Bearer header-token")

	resp, err := app.Test(req)
	assert.NoError(t, err)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "cookie-token", result["token"])
}

func TestExtractTokenString_BearerFallback(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"token": ExtractTokenString(c)})
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer header-token")

	resp, err := app.Test(req)
	assert.NoError(t, err)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "header-token", result["token"])
}

func TestExtractTokenString_NoToken(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"token": ExtractTokenString(c)})
	})

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "", result["token"])
}

func TestExtractTokenString_DuplicateCookiesPrefersLast(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"token": ExtractTokenString(c)})
	})

	// Browser dapat mengirim dua cookie sigap_session dengan Path berbeda:
	// cookie stale warisan versi lama di /api (revoked) dikirim LEBIH DULU
	// (RFC 6265 §5.4 — Path terpanjang pertama), cookie sesi aktif di "/"
	// paling akhir. Token yang dipilih harus yang TERAKHIR (sesi aktif).
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Cookie", "sigap_session=stale-revoked-token; sigap_session=active-token; sigap_csrf=abc")

	resp, err := app.Test(req)
	assert.NoError(t, err)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "active-token", result["token"])
}

func TestExtractTokenString_NoSessionCookieUsesBearer(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"token": ExtractTokenString(c)})
	})

	// Tanpa cookie sesi, header Bearer dipakai sebagai fallback.
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Cookie", "sigap_csrf=abc")
	req.Header.Set("Authorization", "Bearer header-token")

	resp, err := app.Test(req)
	assert.NoError(t, err)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "header-token", result["token"])
}
