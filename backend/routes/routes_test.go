package routes

import (
	"net/http/httptest"
	"testing"

	"sigap-backend/middleware"

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

func TestSetupAllRoutesRegistersProtectedEndpoints(t *testing.T) {
	app := fiber.New()
	api := app.Group("/api", middleware.GlobalLimiter(), middleware.CSRFProtection())
	SetupAuthRoutes(api)
	SetupMaintenanceRoutes(api)
	SetupUserRoutes(api)
	SetupCCTVRoutes(api)
	SetupUploadRoutes(api)
	SetupAuditRoutes(api)
	SetupWebSocketRoutes(api)
	SetupDashboardRoutes(api)

	resp, err := app.Test(httptest.NewRequest("GET", "/api/dashboard/summary", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestLogoutRouteRemainsAvailableWithoutSession(t *testing.T) {
	app := fiber.New()
	api := app.Group("/api", middleware.GlobalLimiter(), middleware.CSRFProtection())
	SetupAuthRoutes(api)

	resp, err := app.Test(httptest.NewRequest("POST", "/api/auth/logout", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}
