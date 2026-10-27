package middleware

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"sigap-backend/config"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestGlobalLimiter_AllowsUnderLimit(t *testing.T) {
	app := fiber.New()
	app.Use(GlobalLimiter())
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	// Sejumlah request di bawah batas 100/menit harus lolos
	for i := 0; i < 10; i++ {
		resp, err := app.Test(httptest.NewRequest("GET", "/", nil))
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	}
}

func TestGlobalLimiter_RejectsOverLimit(t *testing.T) {
	t.Setenv("RATE_LIMIT_GLOBAL_MAX", "100")
	app := fiber.New()
	app.Use(GlobalLimiter())
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	// Melebihi batas 100 request/menit → 429 Too Many Requests
	var lastStatus int
	for i := 0; i < 105; i++ {
		resp, err := app.Test(httptest.NewRequest("GET", "/", nil))
		assert.NoError(t, err)
		lastStatus = resp.StatusCode
		if lastStatus == fiber.StatusTooManyRequests {
			break
		}
	}
	assert.Equal(t, fiber.StatusTooManyRequests, lastStatus)

	// Body error limiter
	resp, _ := app.Test(httptest.NewRequest("GET", "/", nil))
	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "error", result["status"])
}

// TestLimiterKey_Format memverifikasi format key "limiter:{ip}:{endpoint}".
func TestLimiterKey_Format(t *testing.T) {
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		key := limiterKey(c)
		assert.Regexp(t, `^limiter:[^:]+:(/|/api/.*)$`, key, "format key harus limiter:{ip}:{endpoint}")
		return c.SendStatus(fiber.StatusOK)
	})
	app.Get("/api/login", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	resp, err := app.Test(httptest.NewRequest("GET", "/api/login", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

// TestLimiter_DirectClientCannotBypassWithSpoofedXFF memastikan klien yang
// mengakses API tanpa trusted proxy tidak bisa memecah bucket rate limit dengan
// mengganti-ganti header X-Forwarded-For.
func TestLimiter_DirectClientCannotBypassWithSpoofedXFF(t *testing.T) {
	const route = "/api/limiter-spoof-direct"
	app := fiber.New(config.NewFiberConfig(nil))
	app.Get(route, newFixedWindowLimiter(2, time.Minute, "Terlalu banyak permintaan."), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	spoofed := []string{"198.51.100.1", "198.51.100.2", "198.51.100.3"}
	for i, ip := range spoofed {
		req := httptest.NewRequest("GET", route, nil)
		req.Header.Set(fiber.HeaderXForwardedFor, ip)
		req.Header.Set("X-Real-IP", ip)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		if i < 2 {
			assert.Equal(t, fiber.StatusOK, resp.StatusCode)
		} else {
			assert.Equal(t, fiber.StatusTooManyRequests, resp.StatusCode,
				"IP palsu tidak boleh membuat bucket rate limit baru")
		}
	}
}

// TestLimiter_TrustedProxyUsesValidatedClientIP memastikan hanya koneksi dari
// trusted proxy yang boleh memakai X-Forwarded-For, dan entri tidak valid
// dilewati sehingga rate limit tetap memakai IP klien yang valid.
func TestLimiter_TrustedProxyUsesValidatedClientIP(t *testing.T) {
	const route = "/api/limiter-trusted-proxy"
	app := fiber.New(config.NewFiberConfig([]string{"0.0.0.0"}))
	app.Get(route, newFixedWindowLimiter(1, time.Minute, "Terlalu banyak permintaan."), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	request := func(forwardedFor string) int {
		req := httptest.NewRequest("GET", route, nil)
		req.Header.Set(fiber.HeaderXForwardedFor, forwardedFor)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		return resp.StatusCode
	}

	assert.Equal(t, fiber.StatusOK, request("203.0.113.10"))
	assert.Equal(t, fiber.StatusTooManyRequests, request("203.0.113.10"),
		"IP klien yang sama harus memakai bucket yang sama")
	assert.Equal(t, fiber.StatusOK, request("203.0.113.11"),
		"IP klien berbeda harus memakai bucket terpisah")
	assert.Equal(t, fiber.StatusOK, request("bukan-ip, 203.0.113.12"),
		"entri X-Forwarded-For tidak valid harus dilewati")
	assert.Equal(t, fiber.StatusTooManyRequests, request("bukan-ip, 203.0.113.12"),
		"IP valid pertama tetap menjadi identitas klien")
}

// TestLimiterMemoryCounter_WindowReset memverifikasi counter in-memory di-reset
// setelah window rate limit berakhir (fallback tanpa Redis).
func TestLimiterMemoryCounter_WindowReset(t *testing.T) {
	counter := newMemoryRateLimitCounter()

	// Isi counter melewati batas window sangat pendek
	window := 50 * time.Millisecond
	count, err := counter.Increment("limiter:ip:/api/x", window)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// Tunggu window habis → counter harus kembali ke 1 (reset window baru)
	time.Sleep(80 * time.Millisecond)
	count, err = counter.Increment("limiter:ip:/api/x", window)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count, "counter harus di-reset setelah window berakhir")
}

// TestLimiterMemoryCounter_PerKeyIsolation memverifikasi counter per key
// terpisah satu sama lain.
func TestLimiterMemoryCounter_PerKeyIsolation(t *testing.T) {
	counter := newMemoryRateLimitCounter()
	window := time.Minute

	for i := 0; i < 3; i++ {
		_, _ = counter.Increment("limiter:ip-a:/api/x", window)
		_, _ = counter.Increment("limiter:ip-b:/api/x", window)
	}

	countA, _ := counter.Increment("limiter:ip-a:/api/x", window)
	countB, _ := counter.Increment("limiter:ip-b:/api/x", window)
	assert.Equal(t, int64(4), countA)
	assert.Equal(t, int64(4), countB)

	// Key endpoint lain tetap terisolasi
	countC, _ := counter.Increment("limiter:ip-a:/api/other", window)
	assert.Equal(t, int64(1), countC)
}
