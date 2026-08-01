package handlers

import (
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestGenerateMaintenanceReportPDF_Success(t *testing.T) {
	_, mock := setupMockGORM(t)

	// Tidak ada log pada periode → laporan tetap terbit dengan catatan kosong
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "created_at", "updated_at", "deleted_at",
			"cctv_id", "technician_id", "action_taken", "notes", "status",
			"scheduled_date", "completed_at", "image_url",
		}))

	app := newTestApp()
	app.Get("/maintenance-logs/report-pdf", withUserLocals(uuid.New().String(), "admin"), GenerateMaintenanceReportPDF)

	resp, err := app.Test(httptest.NewRequest("GET", "/maintenance-logs/report-pdf", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/pdf", resp.Header.Get("Content-Type"))

	buf := make([]byte, 4)
	n, _ := resp.Body.Read(buf)
	assert.Equal(t, "%PDF", string(buf[:n]), "body harus berupa file PDF valid")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGenerateMaintenanceReportPDF_TechnicianScoped(t *testing.T) {
	_, mock := setupMockGORM(t)

	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "created_at", "updated_at", "deleted_at",
			"cctv_id", "technician_id", "action_taken", "notes", "status",
			"scheduled_date", "completed_at", "image_url",
		}))

	app := newTestApp()
	app.Get("/maintenance-logs/report-pdf", withUserLocals(uuid.New().String(), "teknisi"), GenerateMaintenanceReportPDF)

	resp, err := app.Test(httptest.NewRequest("GET", "/maintenance-logs/report-pdf?status=completed", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}
