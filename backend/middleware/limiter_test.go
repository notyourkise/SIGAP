package middleware

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

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
