package handlers

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestHealthCheck_Connected(t *testing.T) {
	setupMockGORM(t)
	app := newTestApp()
	app.Get("/health", HealthCheck)

	resp, err := app.Test(httptest.NewRequest("GET", "/health", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "Healthy", result["api_status"])
	assert.Equal(t, "Connected", result["db_status"])
}
