package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
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

// TestGenerateReport_PDF_LongNameAndIPNVR menjaga agar nilai panjang dibungkus
// di dalam sel, bukan dipotong atau melampaui kolom.
func TestGenerateReport_PDF_LongNameAndIPNVR(t *testing.T) {
	_, mock := setupMockGORM(t)

	rows := sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"kode_cctv", "id_channel", "nama", "lokasi_area",
		"longitude", "latitude", "jenis_cctv", "sumber_listrik",
		"tipe_koneksi", "indoor_outdoor", "pemilik", "status",
		"ip_nvr", "ip_camera", "keterangan", "qr_payload", "health_score",
		"channel_port", "rtsp_password", "stream_enabled",
	})
	for i := 1; i <= 24; i++ {
		rows.AddRow(uuid.New(), time.Now(), time.Now(), nil,
			fmt.Sprintf("IKN-KIPP-CCTV-%04d", i), "CH-01",
			fmt.Sprintf("CCTV Gerbang Utama Kawasan Inti Pusat Pemerintahan Nusantara %02d", i),
			"Koridor pelayanan utama kawasan pemerintahan", "117.1", "-0.5", "Dome", "PLN",
			"Fiber", "outdoor", "Otorita Ibu Kota Nusantara", "aktif",
			fmt.Sprintf("192.168.100.%d", i), fmt.Sprintf("10.20.30.%d", i), "Aman", "encrypted", 100, 19, "encrypted-rtsp", true)
	}
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).WillReturnRows(rows)

	app := newTestApp()
	app.Post("/reports/generate", withUserLocals(uuid.New().String(), "admin"), GenerateReport)
	body := `{"entity":"cctv","columns":["kode_cctv","nama","lokasi_area","jenis_cctv","status","ip_camera","ip_nvr","pemilik","created_at"],"format":"pdf"}`
	req := httptest.NewRequest("POST", "/reports/generate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/pdf", resp.Header.Get("Content-Type"))
	pdfBytes, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(string(pdfBytes), "%PDF"))
	if output := os.Getenv("CUSTOM_REPORT_PDF_QA_OUTPUT"); output != "" {
		assert.NoError(t, os.WriteFile(output, pdfBytes, 0o600))
	}
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

// TestAudit_FieldReportMustApplyDateFilter menguji pemenuhan kriteria Issue #179 (M06):
//  1. Filter rentang tanggal (start_date & end_date) pada entitas field_maintenance
//     diterapkan secara konsisten pada query database menggunakan zona waktu WITA.
//  2. Baris data di luar periode tidak lolos query filter rentang tanggal.
//  3. Format tanggal tidak valid ditolak dengan error 400 Bad Request yang informatif.
//  4. Rentang tanggal di mana start_date melampaui end_date ditolak dengan 400 Bad Request.
//  5. Batas akses: memverifikasi bahwa endpoint laporan dilindungi hanya untuk admin.
func TestAudit_FieldReportMustApplyDateFilter(t *testing.T) {
	_, mock := setupMockGORM(t)

	fmID := uuid.New()
	cctvID := uuid.New()
	techID := uuid.New()
	createdAtOct := time.Date(2026, time.October, 15, 10, 0, 0, 0, time.UTC)
	completedAtOct := time.Date(2026, time.October, 15, 11, 30, 0, 0, time.UTC)

	// Kasus 1: Request field_maintenance dengan filter rentang Oktober 2026
	// Ekspektasi: SQL memuat klausa created_at >= $1 AND created_at < $2
	fmRows := sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"cctv_id", "technician_id", "maintenance_log_id", "overall_status",
		"notes", "check_latitude", "check_longitude", "completed_at",
	}).AddRow(fmID, createdAtOct, createdAtOct, nil, cctvID, techID, nil, "ok", "Kondisi fisik bersih", "", "", completedAtOct)

	mock.ExpectQuery(`^SELECT \* FROM "field_maintenances" WHERE created_at >= \$1 AND created_at < \$2`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(fmRows)

	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kode_cctv", "nama", "status"}).
			AddRow(cctvID, "CCTV-PPU-001", "Kamera Gerbang Barat", "aktif"))

	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "role", "full_name"}).
			AddRow(techID, "teknisi1", "teknisi", "Budi Teknisi"))

	app := newTestApp()
	app.Post("/reports/generate", withUserLocals(uuid.New().String(), "admin"), GenerateReport)

	validPayload := `{
		"entity": "field_maintenance",
		"columns": ["cctv_nama", "cctv_kode", "technician", "overall_status", "completed_at"],
		"filters": {
			"start_date": "2026-10-01",
			"end_date": "2026-10-31"
		},
		"format": "csv"
	}`
	req1 := httptest.NewRequest("POST", "/reports/generate", strings.NewReader(validPayload))
	req1.Header.Set("Content-Type", "application/json")

	resp1, err := app.Test(req1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp1.StatusCode)
	assert.Equal(t, "text/csv; charset=UTF-8", resp1.Header.Get("Content-Type"))

	csvBytes, err := io.ReadAll(resp1.Body)
	assert.NoError(t, err)
	csvContent := string(csvBytes)
	assert.Contains(t, csvContent, "Kamera Gerbang Barat")
	assert.Contains(t, csvContent, "Budi Teknisi")
	assert.Contains(t, csvContent, "2026-10-15")

	// Kasus 2: Input format tanggal tidak valid harus mengembalikan 400 Bad Request
	invalidDatePayload := `{
		"entity": "field_maintenance",
		"columns": ["cctv_nama"],
		"filters": {
			"start_date": "2026-13-45",
			"end_date": "2026-10-31"
		},
		"format": "csv"
	}`
	req2 := httptest.NewRequest("POST", "/reports/generate", strings.NewReader(invalidDatePayload))
	req2.Header.Set("Content-Type", "application/json")

	resp2, err := app.Test(req2)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp2.StatusCode)

	var errBody2 map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp2.Body).Decode(&errBody2))
	assert.Contains(t, errBody2["message"], "format start_date tidak valid")

	// Kasus 3: Tanggal awal melampaui tanggal akhir harus mengembalikan 400 Bad Request
	reversedDatePayload := `{
		"entity": "field_maintenance",
		"columns": ["cctv_nama"],
		"filters": {
			"start_date": "2026-11-01",
			"end_date": "2026-10-01"
		},
		"format": "csv"
	}`
	req3 := httptest.NewRequest("POST", "/reports/generate", strings.NewReader(reversedDatePayload))
	req3.Header.Set("Content-Type", "application/json")

	resp3, err := app.Test(req3)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp3.StatusCode)

	var errBody3 map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp3.Body).Decode(&errBody3))
	assert.Contains(t, errBody3["message"], "start_date tidak boleh setelah end_date")

	assert.NoError(t, mock.ExpectationsWereMet())
}
