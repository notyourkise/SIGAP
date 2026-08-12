package handlers

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func expectTechnicianPerformanceQuery(mock sqlmock.Sqlmock, rows *sqlmock.Rows) {
	mock.ExpectQuery(`^SELECT\s+u\.id\s+AS user_id`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(rows)
}

func TestTechnicianPerformance_ResponseStructure(t *testing.T) {
	_, mock := setupMockGORM(t)
	firstID := uuid.New()
	secondID := uuid.New()

	expectTechnicianPerformanceQuery(mock, sqlmock.NewRows([]string{
		"user_id", "full_name", "username", "total_tasks", "completed", "failed", "ongoing",
		"avg_completion_hours", "overdue_count", "field_check_count", "ok_count",
	}).
		AddRow(firstID.String(), "Teknisi Satu", "teknisi1", 10, 8, 1, 1, 24.56, 2, 5, 4).
		AddRow(secondID.String(), "Teknisi Dua", "teknisi2", 4, 2, 0, 2, 8.04, 0, 0, 0))

	app := newTestApp()
	app.Get("/dashboard/technician-performance", withUserLocals(uuid.New().String(), "admin"), GetTechnicianPerformance)

	resp, err := app.Test(httptest.NewRequest("GET", "/dashboard/technician-performance", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])

	data, ok := result["data"].([]interface{})
	if assert.True(t, ok) {
		assert.Len(t, data, 2)
		first := data[0].(map[string]interface{})
		assert.Equal(t, firstID.String(), first["user_id"])
		assert.Equal(t, "Teknisi Satu", first["full_name"])
		assert.Equal(t, float64(10), first["total_tasks"])
		assert.Equal(t, float64(8), first["completed"])
		assert.Equal(t, float64(80), first["completion_rate"])
		assert.Equal(t, float64(24.6), first["avg_completion_hours"])
		assert.Equal(t, float64(5), first["field_check_count"])
		assert.Equal(t, float64(80), first["quality_score"])
	}
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTechnicianPerformance_DateFilter(t *testing.T) {
	_, mock := setupMockGORM(t)
	startDate := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`^SELECT\s+u\.id\s+AS user_id`).
		WithArgs(sqlmock.AnyArg(), startDate, endDate, startDate, endDate).
		WillReturnRows(sqlmock.NewRows([]string{
			"user_id", "full_name", "username", "total_tasks", "completed", "failed", "ongoing",
			"avg_completion_hours", "overdue_count", "field_check_count", "ok_count",
		}))

	app := newTestApp()
	app.Get("/dashboard/technician-performance", withUserLocals(uuid.New().String(), "admin"), GetTechnicianPerformance)

	request := httptest.NewRequest(
		"GET",
		"/dashboard/technician-performance?start_date=2026-07-01&end_date=2026-07-31",
		nil,
	)
	resp, err := app.Test(request)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	assert.Empty(t, result["data"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTechnicianPerformanceWithPeriod(t *testing.T) {
	_, mock := setupMockGORM(t)

	// `period=7d` harus menghasilkan query dengan 5 argumen (now + dua pasang
	// startDate/endDate) seperti query kinerja pada umumnya. Nilai tanggal
	// tepatnya memakai `now` saat handler berjalan sehingga tidak bisa
	// diprediksi eksak; AnyArg memastikan struktur query tetap terjaga.
	mock.ExpectQuery(`^SELECT\s+u\.id\s+AS user_id`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"user_id", "full_name", "username", "total_tasks", "completed", "failed", "ongoing",
			"avg_completion_hours", "overdue_count", "field_check_count", "ok_count",
		}))

	app := newTestApp()
	app.Get("/dashboard/technician-performance", withUserLocals(uuid.New().String(), "admin"), GetTechnicianPerformance)

	request := httptest.NewRequest("GET", "/dashboard/technician-performance?period=7d", nil)
	resp, err := app.Test(request)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTechnicianPerformance_NonAdminForbidden(t *testing.T) {
	app := newTestApp()
	app.Get("/dashboard/technician-performance", withUserLocals(uuid.New().String(), "teknisi"), GetTechnicianPerformance)

	resp, err := app.Test(httptest.NewRequest("GET", "/dashboard/technician-performance", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}
