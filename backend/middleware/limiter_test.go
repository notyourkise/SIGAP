package middleware

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

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
