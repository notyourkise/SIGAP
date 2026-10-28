package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"sigap-backend/config"
	"sigap-backend/middleware"
	"sigap-backend/utils"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestCCTVRoutesRequireAuthentication(t *testing.T) {
	app := fiber.New()
	SetupCCTVRoutes(app)

	resp, err := app.Test(httptest.NewRequest("GET", "/cctv/", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

// TestCCTVRoutesRegisterProtectedPaths memastikan restrukturisasi grup
// middleware (auth -> SiteContext) tidak menghilangkan rute protected/admin.
func TestCCTVRoutesRegisterProtectedPaths(t *testing.T) {
	app := fiber.New()
	SetupCCTVRoutes(app)

	const id = "00000000-0000-0000-0000-000000000000"
	cases := []struct {
		method string
		path   string
	}{
		{"GET", "/cctv/template"},
		{"POST", "/cctv/scan"},
		{"GET", "/cctv/" + id + "/qrcode"},
		{"GET", "/cctv/" + id + "/health-score"},
		{"GET", "/cctv/" + id + "/snapshot-status"},
		{"GET", "/cctv/" + id + "/snapshot"},
		{"GET", "/cctv/" + id + "/timeline"},
		{"GET", "/cctv/" + id + "/health-history"},
		{"GET", "/cctv/" + id},
		{"POST", "/cctv/"},
		{"POST", "/cctv/import"},
		{"POST", "/cctv/bulk-delete"},
		{"POST", "/cctv/" + id + "/rotate-qr"},
		{"PUT", "/cctv/" + id},
		{"DELETE", "/cctv/" + id},
	}

	for _, tc := range cases {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		resp, err := app.Test(req)
		assert.NoError(t, err, "%s %s", tc.method, tc.path)
		assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode, "%s %s", tc.method, tc.path)
	}
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

func TestReportRoutesRejectTechnician(t *testing.T) {
	t.Setenv("JWT_SECRET", "sigap_test_jwt_secret_key_2026")
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
	assert.NoError(t, err)
	config.DB = gormDB

	userID := "technician-report-test"
	mock.ExpectQuery(`^SELECT "token_version" FROM "users"`).
		WithArgs(userID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"token_version"}).AddRow(1))
	token, err := utils.GenerateToken(userID, "teknisi", 1)
	assert.NoError(t, err)

	app := fiber.New()
	SetupReportRoutes(app)
	req := httptest.NewRequest("POST", "/reports/generate", nil)
	req.AddCookie(&http.Cookie{Name: utils.SessionCookieName, Value: token})
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSparePartInventoryRoutesRejectTechnician(t *testing.T) {
	t.Setenv("JWT_SECRET", "sigap_test_jwt_secret_key_2026")
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
	assert.NoError(t, err)
	config.DB = gormDB

	userID := "technician-spare-part-test"
	mock.ExpectQuery(`^SELECT "token_version" FROM "users"`).
		WithArgs(userID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"token_version"}).AddRow(1))
	token, err := utils.GenerateToken(userID, "teknisi", 1)
	assert.NoError(t, err)

	app := fiber.New()
	SetupSparePartRoutes(app)
	req := httptest.NewRequest("GET", "/spare-parts/", nil)
	req.AddCookie(&http.Cookie{Name: utils.SessionCookieName, Value: token})
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDashboardDigestRouteRequiresAuthentication(t *testing.T) {
	app := fiber.New()
	SetupDashboardRoutes(app)

	resp, err := app.Test(httptest.NewRequest("GET", "/dashboard/digest", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestDashboardSummaryRejectsUnassignedTechnician(t *testing.T) {
	t.Setenv("JWT_SECRET", "sigap_test_jwt_secret_key_2026")
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
	assert.NoError(t, err)
	config.DB = gormDB

	userID := uuid.NewString()
	mock.ExpectQuery(`^SELECT "token_version" FROM "users"`).
		WithArgs(userID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"token_version"}).AddRow(1))
	mock.ExpectQuery(`FROM "sites" JOIN user_sites`).
		WillReturnError(gorm.ErrRecordNotFound)
	token, err := utils.GenerateToken(userID, "teknisi", 1)
	assert.NoError(t, err)

	app := fiber.New()
	SetupDashboardRoutes(app)
	req := httptest.NewRequest("GET", "/dashboard/summary", nil)
	req.AddCookie(&http.Cookie{Name: utils.SessionCookieName, Value: token})
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFieldMaintenanceRejectsUnassignedTechnician(t *testing.T) {
	t.Setenv("JWT_SECRET", "sigap_test_jwt_secret_key_2026")
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
	assert.NoError(t, err)
	config.DB = gormDB

	userID := uuid.NewString()
	mock.ExpectQuery(`^SELECT "token_version" FROM "users"`).
		WithArgs(userID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"token_version"}).AddRow(1))
	mock.ExpectQuery(`FROM "sites" JOIN user_sites`).
		WillReturnError(gorm.ErrRecordNotFound)
	token, err := utils.GenerateToken(userID, "teknisi", 1)
	assert.NoError(t, err)

	app := fiber.New()
	SetupFieldMaintenanceRoutes(app)
	req := httptest.NewRequest("GET", "/field-maintenance/", nil)
	req.AddCookie(&http.Cookie{Name: utils.SessionCookieName, Value: token})
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
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
