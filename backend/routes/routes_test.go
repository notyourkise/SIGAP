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

func TestFieldMaintenanceRoutesRequireAuthentication(t *testing.T) {
	app := fiber.New()
	SetupFieldMaintenanceRoutes(app)

	resp, err := app.Test(httptest.NewRequest("GET", "/field-maintenance/"+"00000000-0000-0000-0000-000000000000"+"/report-pdf", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestDashboardDigestRouteRequiresAuthentication(t *testing.T) {
	app := fiber.New()
	SetupDashboardRoutes(app)

	resp, err := app.Test(httptest.NewRequest("GET", "/dashboard/digest", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestSetupAllRoutesRegistersProtectedEndpoints(t *testing.T) {
	app := fiber.New()
	api := app.Group("/api", middleware.GlobalLimiter(), middleware.CSRFProtection())
	SetupAuthRoutes(api)
	SetupMaintenanceRoutes(api)
	SetupFieldMaintenanceRoutes(api)
	SetupUserRoutes(api)
	SetupCCTVRoutes(api)
	SetupUploadRoutes(api)
	SetupAuditRoutes(api)
	SetupWebSocketRoutes(api)
	SetupDashboardRoutes(api)
	SetupZoneRoutes(api)
	SetupSLARoutes(api)
	SetupAPIKeyRoutes(api)
	SetupWebhookRoutes(api)

	resp, err := app.Test(httptest.NewRequest("GET", "/api/dashboard/summary", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestAPIKeyRoutesRequireAuthentication(t *testing.T) {
	app := fiber.New()
	SetupAPIKeyRoutes(app)

	resp, err := app.Test(httptest.NewRequest("GET", "/api-keys/", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestWebhookRoutesRequireAuthentication(t *testing.T) {
	app := fiber.New()
	SetupWebhookRoutes(app)

	resp, err := app.Test(httptest.NewRequest("GET", "/webhooks/", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestZoneRoutesRequireAuthentication(t *testing.T) {
	app := fiber.New()
	SetupZoneRoutes(app)

	resp, err := app.Test(httptest.NewRequest("GET", "/zones/", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestSiteRoutesRequireAuthentication(t *testing.T) {
	app := fiber.New()
	SetupSiteRoutes(app)

	resp, err := app.Test(httptest.NewRequest("GET", "/sites/", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestSLAPoliciesRoutesRequireAuthentication(t *testing.T) {
	app := fiber.New()
	SetupSLARoutes(app)

	resp, err := app.Test(httptest.NewRequest("GET", "/sla-policies/", nil))
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
