package middleware_test

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

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

func setupTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{SkipDefaultTransaction: true})
	require.NoError(t, err)
	return gormDB, mock
}

// TestAudit_ForcedPasswordMustRestrictOperationalAPI mereproduksi skenario audit M12
// dari laporan audit SIGAP: pengguna dengan must_change_password=true harus dibatasi
// dan ditolak 403 pada endpoint operasional.
func TestAudit_ForcedPasswordMustRestrictOperationalAPI(t *testing.T) {
	gormDB, mock := setupTestDB(t)
	config.DB = gormDB
	t.Setenv("JWT_SECRET", "audit-only-not-a-real-application-secret")

	uid := uuid.NewString()
	token, err := utils.GenerateToken(uid, "admin", 1)
	require.NoError(t, err)

	mock.ExpectQuery(`^SELECT "token_version",\s*"must_change_password" FROM "users"|^SELECT "token_version".*FROM "users"`).
		WithArgs(uid, 1).
		WillReturnRows(sqlmock.NewRows([]string{"token_version", "must_change_password"}).AddRow(1, true))

	app := fiber.New()
	app.Get("/operational", middleware.Protected(), middleware.RequireRole("admin"), func(c *fiber.Ctx) error {
		return c.SendStatus(200)
	})

	req := httptest.NewRequest("GET", "/operational", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("must_change_password=true: expected restricted operational API (403), actual=%d", resp.StatusCode)
	}

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "error", body["status"])
	assert.Equal(t, "Anda wajib mengganti password sebelum dapat mengakses resource ini", body["message"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestM12_OperationalAPIRestrictedWhenMustChangePassword memverifikasi bahwa
// berbagai endpoint operasional ditolak 403 ketika pengguna belum mengganti sandi.
func TestM12_OperationalAPIRestrictedWhenMustChangePassword(t *testing.T) {
	gormDB, mock := setupTestDB(t)
	config.DB = gormDB
	t.Setenv("JWT_SECRET", "m12-test-secret-key-32bytes-auth!")

	operationalPaths := []string{
		"/api/operational",
		"/api/cctvs",
		"/api/maintenance-logs",
		"/api/auth/users",
		"/api/dashboard/summary",
		"/api/reports",
	}

	for _, path := range operationalPaths {
		uid := uuid.NewString()
		token, err := utils.GenerateToken(uid, "admin", 1)
		require.NoError(t, err)

		mock.ExpectQuery(`^SELECT "token_version",\s*"must_change_password" FROM "users"|^SELECT "token_version".*FROM "users"`).
			WithArgs(uid, 1).
			WillReturnRows(sqlmock.NewRows([]string{"token_version", "must_change_password"}).AddRow(1, true))

		app := fiber.New()
		app.All(path, middleware.Protected(), func(c *fiber.Ctx) error {
			return c.SendStatus(fiber.StatusOK)
		})

		req := httptest.NewRequest("GET", path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusForbidden, resp.StatusCode, "path operasional %s harus ditolak 403", path)

		var res map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&res))
		assert.Equal(t, "error", res["status"])
		assert.Equal(t, "Anda wajib mengganti password sebelum dapat mengakses resource ini", res["message"])
	}

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestM12_AllowedEndpointsPassWhenMustChangePassword memverifikasi bahwa endpoint
// yang diperlukan untuk profil, ganti kata sandi, dan logout tetap dapat diakses
// oleh pengguna yang berstatus must_change_password=true.
func TestM12_AllowedEndpointsPassWhenMustChangePassword(t *testing.T) {
	gormDB, mock := setupTestDB(t)
	config.DB = gormDB
	t.Setenv("JWT_SECRET", "m12-test-secret-key-32bytes-auth!")

	allowedCases := []struct {
		method string
		path   string
	}{
		{"GET", "/api/auth/me"},
		{"GET", "/auth/me"},
		{"POST", "/api/auth/change-password"},
		{"POST", "/auth/change-password"},
		{"POST", "/api/auth/logout"},
		{"POST", "/auth/logout"},
		{"PUT", "/api/auth/profile"},
		{"PUT", "/auth/profile"},
	}

	for _, tc := range allowedCases {
		uid := uuid.NewString()
		token, err := utils.GenerateToken(uid, "teknisi", 1)
		require.NoError(t, err)

		mock.ExpectQuery(`^SELECT "token_version",\s*"must_change_password" FROM "users"|^SELECT "token_version".*FROM "users"`).
			WithArgs(uid, 1).
			WillReturnRows(sqlmock.NewRows([]string{"token_version", "must_change_password"}).AddRow(1, true))

		app := fiber.New()
		app.Add(tc.method, tc.path, middleware.Protected(), func(c *fiber.Ctx) error {
			return c.JSON(fiber.Map{"status": "success", "endpoint": tc.path})
		})

		req := httptest.NewRequest(tc.method, tc.path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode, "path %s %s harus diizinkan 200", tc.method, tc.path)

		var res map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&res))
		assert.Equal(t, "success", res["status"])
	}

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestM12_NormalUserUnrestricted memverifikasi bahwa pengguna dengan
// must_change_password=false tidak dibatasi pada endpoint operasional.
func TestM12_NormalUserUnrestricted(t *testing.T) {
	gormDB, mock := setupTestDB(t)
	config.DB = gormDB
	t.Setenv("JWT_SECRET", "m12-test-secret-key-32bytes-auth!")

	uid := uuid.NewString()
	token, err := utils.GenerateToken(uid, "admin", 1)
	require.NoError(t, err)

	mock.ExpectQuery(`^SELECT "token_version",\s*"must_change_password" FROM "users"|^SELECT "token_version".*FROM "users"`).
		WithArgs(uid, 1).
		WillReturnRows(sqlmock.NewRows([]string{"token_version", "must_change_password"}).AddRow(1, false))

	app := fiber.New()
	app.Get("/api/cctvs", middleware.Protected(), func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "success", "data": "cctv_list"})
	})

	req := httptest.NewRequest("GET", "/api/cctvs", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var res map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&res))
	assert.Equal(t, "success", res["status"])

	assert.NoError(t, mock.ExpectationsWereMet())
}
