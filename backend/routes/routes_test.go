package routes

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestCCTVRoutesRequireAuthentication(t *testing.T) {
	app := fiber.New()
	SetupCCTVRoutes(app)

	resp, err := app.Test(httptest.NewRequest("GET", "/cctv/", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestMaintenanceRoutesRequireAuthentication(t *testing.T) {
	app := fiber.New()
	SetupMaintenanceRoutes(app)

	resp, err := app.Test(httptest.NewRequest("GET", "/maintenance-logs/", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}
