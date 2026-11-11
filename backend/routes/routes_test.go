package routes

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"sigap-backend/config"
	"sigap-backend/middleware"
	"sigap-backend/utils"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		{"GET", "/cctv/coordinate-quality"},
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
	mock.ExpectQuery(`^SELECT "token_version",\s*"must_change_password" FROM "users"|^SELECT "token_version".*FROM "users"`).
		WithArgs(userID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"token_version", "must_change_password"}).AddRow(1, false))
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
	mock.ExpectQuery(`^SELECT "token_version",\s*"must_change_password" FROM "users"|^SELECT "token_version".*FROM "users"`).
		WithArgs(userID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"token_version", "must_change_password"}).AddRow(1, false))
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
	mock.ExpectQuery(`^SELECT "token_version",\s*"must_change_password" FROM "users"|^SELECT "token_version".*FROM "users"`).
		WithArgs(userID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"token_version", "must_change_password"}).AddRow(1, false))
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
	mock.ExpectQuery(`^SELECT "token_version",\s*"must_change_password" FROM "users"|^SELECT "token_version".*FROM "users"`).
		WithArgs(userID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"token_version", "must_change_password"}).AddRow(1, false))
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

// TestSetupCCTVRoutes_TechnicianCCTVLogsNoPrefixLeak memastikan teknisi yang
// ditugaskan ke sebuah site dapat mengakses GET /api/cctv-logs dan GET /api/cctv/logs
// tanpa diblokir oleh middleware RequireRole("admin") milik mutasi CCTV (mencegah
// bug Fiber v2 Use() prefix collision /cctv vs /cctv-logs), sementara mutasi admin
// pada /api/cctv/ tetap menolak teknisi dengan 403.
func TestSetupCCTVRoutes_TechnicianCCTVLogsNoPrefixLeak(t *testing.T) {
	t.Setenv("JWT_SECRET", "sigap_test_jwt_secret_key_2026")
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
	require.NoError(t, err)
	prevDB := config.DB
	config.DB = gormDB
	t.Cleanup(func() { config.DB = prevDB })

	techID := uuid.NewString()
	siteID := uuid.New()
	token, err := utils.GenerateToken(techID, "teknisi", 1)
	require.NoError(t, err)

	app := fiber.New()
	api := app.Group("/api")
	SetupCCTVRoutes(api)

	for _, path := range []string{"/api/cctv-logs", "/api/cctv/logs"} {
		mock.ExpectQuery(`^SELECT "token_version",\s*"must_change_password" FROM "users"|^SELECT "token_version".*FROM "users"`).
			WithArgs(techID, 1).
			WillReturnRows(sqlmock.NewRows([]string{"token_version", "must_change_password"}).AddRow(1, false))
		mock.ExpectQuery(`FROM "sites" JOIN user_sites`).
			WithArgs(techID, 1).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "created_at", "updated_at", "deleted_at",
				"nama", "kode", "alamat", "koordinat_lat", "koordinat_lng", "is_default",
			}).AddRow(siteID, time.Now(), time.Now(), nil, "Site A", "SITE-A", "", nil, nil, true))

		mock.ExpectQuery(`^SELECT count\(\*\) FROM "cctv_stream_logs" WHERE .*EXISTS .*cctvs\.site_id = \$1`).
			WithArgs(siteID.String()).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectQuery(`^SELECT \* FROM "cctv_stream_logs" WHERE .*EXISTS .*cctvs\.site_id = \$1`).
			WithArgs(siteID.String(), 20).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "cctv_id", "event_type"}))
		mock.ExpectQuery(`^SELECT count\(\*\) FROM "cctv_stream_logs" WHERE .*EXISTS .*cctvs\.site_id = \$1`).
			WithArgs(siteID.String(), sqlmock.AnyArg(), sqlmock.AnyArg(), "online").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectQuery(`^SELECT count\(\*\) FROM "cctv_stream_logs" WHERE .*EXISTS .*cctvs\.site_id = \$1`).
			WithArgs(siteID.String(), sqlmock.AnyArg(), sqlmock.AnyArg(), "offline").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		req := httptest.NewRequest("GET", path, nil)
		req.AddCookie(&http.Cookie{Name: utils.SessionCookieName, Value: token})
		resp, err := app.Test(req)
		require.NoError(t, err, path)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode, path)
	}

	// Mutasi admin CCTV masih wajib menolak teknisi dengan 403 error peran.
	mock.ExpectQuery(`^SELECT "token_version",\s*"must_change_password" FROM "users"|^SELECT "token_version".*FROM "users"`).
		WithArgs(techID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"token_version", "must_change_password"}).AddRow(1, false))

	adminMutReq := httptest.NewRequest("POST", "/api/cctv/", nil)
	adminMutReq.AddCookie(&http.Cookie{Name: utils.SessionCookieName, Value: token})
	adminMutResp, err := app.Test(adminMutReq)
	require.NoError(t, err)
	bodyBytes, err := io.ReadAll(adminMutResp.Body)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, adminMutResp.StatusCode)
	assert.Contains(t, string(bodyBytes), "Peran Anda tidak memiliki izin")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestSetupCCTVRoutes_CoordinateQualityStaticRouteAndIsolation memverifikasi
// rute statis /cctv/coordinate-quality terdaftar sebelum /:id (tidak dianggap
// UUID invalid) dan menerapkan fail-closed 403 bagi teknisi tanpa site.
func TestSetupCCTVRoutes_CoordinateQualityStaticRouteAndIsolation(t *testing.T) {
	t.Setenv("JWT_SECRET", "sigap_test_jwt_secret_key_2026")
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
	require.NoError(t, err)
	prevDB := config.DB
	config.DB = gormDB
	t.Cleanup(func() { config.DB = prevDB })

	techID := uuid.NewString()
	token, err := utils.GenerateToken(techID, "teknisi", 1)
	require.NoError(t, err)

	mock.ExpectQuery(`^SELECT "token_version",\s*"must_change_password" FROM "users"|^SELECT "token_version".*FROM "users"`).
		WithArgs(techID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"token_version", "must_change_password"}).AddRow(1, false))
	mock.ExpectQuery(`FROM "sites" JOIN user_sites`).
		WithArgs(techID, 1).
		WillReturnError(gorm.ErrRecordNotFound)

	app := fiber.New()
	api := app.Group("/api")
	SetupCCTVRoutes(api)

	req := httptest.NewRequest("GET", "/api/cctv/coordinate-quality", nil)
	req.AddCookie(&http.Cookie{Name: utils.SessionCookieName, Value: token})
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}
