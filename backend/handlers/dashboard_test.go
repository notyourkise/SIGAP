package handlers

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestDashboardTasks_NoUserIdentity(t *testing.T) {
	app := newTestApp()
	app.Get("/dashboard/tasks", DashboardTasks)

	resp, err := app.Test(httptest.NewRequest("GET", "/dashboard/tasks", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestDashboardTasks_Technician(t *testing.T) {
	_, mock := setupMockGORM(t)
	techID := uuid.New()

	// Teknisi → hanya tugas miliknya (filter technician_id)
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "cctv_id", "technician_id", "action_taken", "status"}))

	app := newTestApp()
	app.Get("/dashboard/tasks", withUserLocals(techID.String(), "teknisi"), DashboardTasks)

	resp, err := app.Test(httptest.NewRequest("GET", "/dashboard/tasks", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	data := result["data"].(map[string]interface{})
	assert.Equal(t, float64(0), data["pending_count"])
	assert.Equal(t, float64(0), data["overdue_count"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDashboardTasks_Admin(t *testing.T) {
	_, mock := setupMockGORM(t)
	adminID := uuid.New()

	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "cctv_id", "technician_id", "action_taken", "status"}))

	app := newTestApp()
	app.Get("/dashboard/tasks", withUserLocals(adminID.String(), "admin"), DashboardTasks)

	resp, err := app.Test(httptest.NewRequest("GET", "/dashboard/tasks", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDashboardSummary_SiteScoped(t *testing.T) {
	_, mock := setupMockGORM(t)
	siteID := uuid.New()

	// Seluruh query dashboard wajib membawa filter site aktif.
	mock.ExpectQuery(`^SELECT status, count\(\*\) as count FROM "cctvs" WHERE site_id = \$[0-9]+`).
		WillReturnRows(sqlmock.NewRows([]string{"status", "count"}).AddRow("aktif", 1))
	mock.ExpectQuery(`^SELECT status, created_at FROM "maintenance_logs" WHERE created_at >= \$[0-9]+ AND \(EXISTS`).
		WillReturnRows(sqlmock.NewRows([]string{"status", "created_at"}))
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs" WHERE .*EXISTS .*site_id = \$[0-9]+`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "cctv_id", "technician_id", "action_taken", "status", "scheduled_date"}))

	app := newTestApp()
	app.Get("/dashboard/summary", withUserSiteLocals(uuid.New().String(), "teknisi", siteID.String()), DashboardSummary)

	resp, err := app.Test(httptest.NewRequest("GET", "/dashboard/summary", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDashboardTasks_SiteScoped(t *testing.T) {
	_, mock := setupMockGORM(t)
	techID := uuid.New()
	siteID := uuid.New()

	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs" WHERE .*EXISTS .*technician_id = \$[0-9]+`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "cctv_id", "technician_id", "action_taken", "status"}))

	app := newTestApp()
	app.Get("/dashboard/tasks", withUserSiteLocals(techID.String(), "teknisi", siteID.String()), DashboardTasks)

	resp, err := app.Test(httptest.NewRequest("GET", "/dashboard/tasks", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDashboardSummary_Success(t *testing.T) {
	_, mock := setupMockGORM(t)

	// 1. Distribusi status CCTV
	mock.ExpectQuery(`^SELECT status, count\(\*\) as count FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{"status", "count"}).
			AddRow("aktif", 2).
			AddRow("nonaktif", 1))

	// 2. Tren pemeliharaan 6 bulan
	mock.ExpectQuery(`^SELECT status, created_at FROM "maintenance_logs"`).
		WillReturnRows(sqlmock.NewRows([]string{"status", "created_at"}))

	// 3. Overdue (kosong → tanpa preload)
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "cctv_id", "technician_id", "action_taken", "status", "scheduled_date"}))

	app := newTestApp()
	app.Get("/dashboard/summary", DashboardSummary)

	resp, err := app.Test(httptest.NewRequest("GET", "/dashboard/summary", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	data := result["data"].(map[string]interface{})
	statusCounts := data["cctv_status"].([]interface{})
	assert.Len(t, statusCounts, 2)

	trend := data["maintenance_trend"].([]interface{})
	// Selalu 6 bulan
	assert.Len(t, trend, 6)

	overdue := data["overdue"].([]interface{})
	assert.Len(t, overdue, 0)
	assert.NoError(t, mock.ExpectationsWereMet())
}
