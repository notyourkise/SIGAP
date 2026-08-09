package handlers

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// TestGenerateReport_CSV_Success menguji pembuatan laporan CSV dari entitas CCTV.
func TestGenerateReport_CSV_Success(t *testing.T) {
	_, mock := setupMockGORM(t)

	// Query data CCTV.
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "created_at", "updated_at", "deleted_at",
			"kode_cctv", "id_channel", "nama", "lokasi_area",
			"longitude", "latitude", "jenis_cctv", "sumber_listrik",
			"tipe_koneksi", "indoor_outdoor", "pemilik", "status",
			"ip_nvr", "ip_camera", "keterangan", "qr_payload", "health_score",
			"channel_port", "rtsp_password", "stream_enabled",
		}).AddRow(uuid.New(), time.Now(), time.Now(), nil, "CCTV-001", "CH-01", "Kamera Depan", "Area Utama",
			"117.1", "-0.5", "Dome", "PLN", "Fiber", "outdoor", "Pemkot", "aktif",
			"192.168.1.10", "192.168.1.11", "Aman", "encrypted", 100, 19, "encrypted-rtsp", true))

	app := newTestApp()
	app.Post("/reports/generate", withUserLocals(uuid.New().String(), "admin"), GenerateReport)

	body := `{"entity":"cctv","columns":["kode_cctv","nama","status"],"format":"csv"}`
	req := httptest.NewRequest("POST", "/reports/generate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/csv; charset=UTF-8", resp.Header.Get("Content-Type"))
	assert.Contains(t, resp.Header.Get("Content-Disposition"), "laporan_")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGenerateReport_InvalidEntity memastikan entitas tak dikenal ditolak.
func TestGenerateReport_InvalidEntity(t *testing.T) {
	_, mock := setupMockGORM(t)

	app := newTestApp()
	app.Post("/reports/generate", withUserLocals(uuid.New().String(), "admin"), GenerateReport)

	body := `{"entity":"assets","columns":["nama"],"format":"csv"}`
	req := httptest.NewRequest("POST", "/reports/generate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGenerateReport_NoValidColumns memastikan kolom tidak valid ditolak.
func TestGenerateReport_NoValidColumns(t *testing.T) {
	_, mock := setupMockGORM(t)

	app := newTestApp()
	app.Post("/reports/generate", withUserLocals(uuid.New().String(), "admin"), GenerateReport)

	body := `{"entity":"cctv","columns":["nonexistent"],"format":"csv"}`
	req := httptest.NewRequest("POST", "/reports/generate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestSaveReportTemplate_Success menguji penyimpanan template laporan.
func TestSaveReportTemplate_Success(t *testing.T) {
	_, mock := setupMockGORM(t)

	mock.ExpectExec(`^INSERT INTO "report_templates"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	app := newTestApp()
	app.Post("/reports/templates", withUserLocals(uuid.New().String(), "admin"), SaveReportTemplate)

	body := `{"nama":"Laporan CCTV Bulanan","config":"{\"entity\":\"cctv\",\"columns\":[\"nama\"],\"format\":\"csv\"}"}`
	req := httptest.NewRequest("POST", "/reports/templates", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestSaveReportTemplate_InvalidConfig memastikan config JSON valid.
func TestSaveReportTemplate_InvalidConfig(t *testing.T) {
	_, mock := setupMockGORM(t)

	app := newTestApp()
	app.Post("/reports/templates", withUserLocals(uuid.New().String(), "admin"), SaveReportTemplate)

	body := `{"nama":"Template","config":"not-json"}`
	req := httptest.NewRequest("POST", "/reports/templates", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestListReportTemplates_Success menguji daftar template.
func TestListReportTemplates_Success(t *testing.T) {
	_, mock := setupMockGORM(t)

	mock.ExpectQuery(`^SELECT \* FROM "report_templates"`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "created_at", "updated_at", "deleted_at", "nama", "config", "created_by",
		}).AddRow(uuid.New(), time.Now(), time.Now(), nil, "Laporan CCTV", `{"entity":"cctv"}`, uuid.New()))

	app := newTestApp()
	app.Get("/reports/templates", withUserLocals(uuid.New().String(), "admin"), ListReportTemplates)

	req := httptest.NewRequest("GET", "/reports/templates", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestDeleteReportTemplate_Success menguji soft-delete template.
func TestDeleteReportTemplate_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	id := uuid.New()

	mock.ExpectQuery(`^SELECT \* FROM "report_templates"`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "created_at", "updated_at", "deleted_at", "nama", "config", "created_by",
		}).AddRow(id, time.Now(), time.Now(), nil, "Laporan CCTV", `{"entity":"cctv"}`, uuid.New()))
	mock.ExpectExec(`^UPDATE "report_templates" SET "deleted_at"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	app := newTestApp()
	app.Delete("/reports/templates/:id", withUserLocals(uuid.New().String(), "admin"), DeleteReportTemplate)

	req := httptest.NewRequest("DELETE", "/reports/templates/"+id.String(), nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestDeleteReportTemplate_NotFound memastikan 404 untuk template tak ada.
func TestDeleteReportTemplate_NotFound(t *testing.T) {
	_, mock := setupMockGORM(t)

	mock.ExpectQuery(`^SELECT \* FROM "report_templates"`).
		WillReturnError(gorm.ErrRecordNotFound)

	app := newTestApp()
	app.Delete("/reports/templates/:id", withUserLocals(uuid.New().String(), "admin"), DeleteReportTemplate)

	req := httptest.NewRequest("DELETE", "/reports/templates/"+uuid.New().String(), nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}
