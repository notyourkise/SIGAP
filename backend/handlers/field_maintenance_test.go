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

func fieldMaintenanceValidPayload(cctvID string) string {
	return `{"cctv_id":"` + cctvID + `","notes":"Pengecekan rutin",
		"checks":[{"name":"Pengecekan jaringan","result":"ok","note":""},{"name":"Pengecekan listrik","result":"not_ok","note":"Adaptor longgar"}],
		"images":[{"image_url":"/api/uploads/bukti1.jpg"}]}`
}

// fieldMaintenanceOkPayload menghasilkan payload pengecekan dengan seluruh item
// checklist berstatus "ok" (hasil keseluruhan menjadi "ok").
func fieldMaintenanceOkPayload(cctvID string) string {
	return `{"cctv_id":"` + cctvID + `","notes":"Semua item normal",
		"checks":[{"name":"Pengecekan jaringan","result":"ok","note":""},{"name":"Pengecekan listrik","result":"ok","note":""}],
		"images":[{"image_url":"/api/uploads/bukti1.jpg"}]}`
}

func TestCreateFieldMaintenance_ValidationError(t *testing.T) {
	app := newTestApp()
	app.Post("/field-maintenance", withUserLocals(uuid.New().String(), "teknisi"), CreateFieldMaintenance)

	// Tanpa checks & images
	payload := `{"cctv_id":"` + uuid.New().String() + `","notes":"kosong"}`
	req := httptest.NewRequest("POST", "/field-maintenance", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestCreateFieldMaintenance_NoImages(t *testing.T) {
	app := newTestApp()
	app.Post("/field-maintenance", withUserLocals(uuid.New().String(), "teknisi"), CreateFieldMaintenance)

	payload := `{"cctv_id":"` + uuid.New().String() + `",
		"checks":[{"name":"Pengecekan jaringan","result":"ok"}],"images":[]}`
	req := httptest.NewRequest("POST", "/field-maintenance", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestCreateFieldMaintenance_CCTVNotFound(t *testing.T) {
	_, mock := setupMockGORM(t)

	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnError(gorm.ErrRecordNotFound)

	app := newTestApp()
	app.Post("/field-maintenance", withUserLocals(uuid.New().String(), "teknisi"), CreateFieldMaintenance)

	payload := fieldMaintenanceValidPayload(uuid.New().String())
	req := httptest.NewRequest("POST", "/field-maintenance", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestCreateFieldMaintenance_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	cctvID := uuid.New()
	techID := uuid.New()

	cctvRows, _ := cctvMockRow()
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(cctvRows)

	// Transaksi: header + checks + images
	mock.ExpectBegin()
	mock.ExpectExec(`^INSERT INTO "field_maintenances"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`^INSERT INTO "field_maintenance_checks"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`^INSERT INTO "field_maintenance_checks"`).
		WillReturnResult(sqlmock.NewResult(2, 1))
	mock.ExpectExec(`^INSERT INTO "field_maintenance_images"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Fase 5.2: sinkronisasi status otomatis — CCTV berstatus "aktif" dengan
	// hasil "attention" (ada item tidak ok) dipindahkan ke "pemeliharaan".
	syncRows, _ := cctvMockRow()
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(syncRows)
	mock.ExpectExec(`^UPDATE "cctvs" SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	// Reload dengan relasi lengkap (GORM menjalankan preload sesuai urutan alfabetis)
	reload := sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"cctv_id", "technician_id", "overall_status", "notes", "check_latitude", "check_longitude", "completed_at",
	}).AddRow(uuid.New(), time.Now(), time.Now(), nil, cctvID, techID, "attention", "Pengecekan rutin", "", "", time.Now())
	mock.ExpectQuery(`^SELECT \* FROM "field_maintenances"`).
		WillReturnRows(reload)
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kode_cctv", "nama", "status"}))
	mock.ExpectQuery(`^SELECT \* FROM "field_maintenance_checks"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "field_maintenance_id", "name", "result", "note"}))
	mock.ExpectQuery(`^SELECT \* FROM "field_maintenance_images"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "field_maintenance_id", "image_url"}))
	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "role"}))
	expectHealthScoreRecalculation(mock)

	app := newTestApp()
	app.Post("/field-maintenance", withUserLocals(techID.String(), "teknisi"), CreateFieldMaintenance)

	payload := fieldMaintenanceValidPayload(cctvID.String())
	req := httptest.NewRequest("POST", "/field-maintenance", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateFieldMaintenance_SyncCCTV_Attention(t *testing.T) {
	// Fase 5.2: hasil pengecekan "attention" (ada item not_ok) memindahkan CCTV
	// berstatus "aktif" menjadi "pemeliharaan" dalam transaksi yang sama.
	_, mock := setupMockGORM(t)
	cctvID := uuid.New()
	techID := uuid.New()

	cctvRows, _ := cctvMockRow()
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(cctvRows)

	// Transaksi: header + checks + images
	mock.ExpectBegin()
	mock.ExpectExec(`^INSERT INTO "field_maintenances"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`^INSERT INTO "field_maintenance_checks"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`^INSERT INTO "field_maintenance_checks"`).
		WillReturnResult(sqlmock.NewResult(2, 1))
	mock.ExpectExec(`^INSERT INTO "field_maintenance_images"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Sinkronisasi otomatis: baca status saat ini ("aktif") lalu update →
	// "pemeliharaan" karena hasil pengecekan memerlukan perhatian.
	syncRows, _ := cctvMockRow()
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(syncRows)
	mock.ExpectExec(`^UPDATE "cctvs" SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	// Reload dengan relasi lengkap
	reload := sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"cctv_id", "technician_id", "overall_status", "notes", "check_latitude", "check_longitude", "completed_at",
	}).AddRow(uuid.New(), time.Now(), time.Now(), nil, cctvID, techID, "attention", "Pengecekan rutin", "", "", time.Now())
	mock.ExpectQuery(`^SELECT \* FROM "field_maintenances"`).
		WillReturnRows(reload)
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kode_cctv", "nama", "status"}))
	mock.ExpectQuery(`^SELECT \* FROM "field_maintenance_checks"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "field_maintenance_id", "name", "result", "note"}))
	mock.ExpectQuery(`^SELECT \* FROM "field_maintenance_images"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "field_maintenance_id", "image_url"}))
	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "role"}))
	expectHealthScoreRecalculation(mock)

	app := newTestApp()
	app.Post("/field-maintenance", withUserLocals(techID.String(), "teknisi"), CreateFieldMaintenance)

	payload := fieldMaintenanceValidPayload(cctvID.String())
	req := httptest.NewRequest("POST", "/field-maintenance", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateFieldMaintenance_SyncCCTV_Ok(t *testing.T) {
	// Fase 5.2: hasil pengecekan "ok" (seluruh item ok) mengembalikan CCTV
	// berstatus "pemeliharaan" menjadi "aktif" dalam transaksi yang sama.
	_, mock := setupMockGORM(t)
	cctvID := uuid.New()
	techID := uuid.New()

	cctvRows, _ := cctvMockRowWithStatus("pemeliharaan")
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(cctvRows)

	// Transaksi: header + checks + images
	mock.ExpectBegin()
	mock.ExpectExec(`^INSERT INTO "field_maintenances"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`^INSERT INTO "field_maintenance_checks"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`^INSERT INTO "field_maintenance_checks"`).
		WillReturnResult(sqlmock.NewResult(2, 1))
	mock.ExpectExec(`^INSERT INTO "field_maintenance_images"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Sinkronisasi otomatis: baca status saat ini ("pemeliharaan") lalu update
	// → "aktif" karena semua item checklist berstatus ok.
	syncRows, _ := cctvMockRowWithStatus("pemeliharaan")
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(syncRows)
	mock.ExpectExec(`^UPDATE "cctvs" SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	// Reload dengan relasi lengkap
	reload := sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"cctv_id", "technician_id", "overall_status", "notes", "check_latitude", "check_longitude", "completed_at",
	}).AddRow(uuid.New(), time.Now(), time.Now(), nil, cctvID, techID, "ok", "Semua item normal", "", "", time.Now())
	mock.ExpectQuery(`^SELECT \* FROM "field_maintenances"`).
		WillReturnRows(reload)
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kode_cctv", "nama", "status"}))
	mock.ExpectQuery(`^SELECT \* FROM "field_maintenance_checks"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "field_maintenance_id", "name", "result", "note"}))
	mock.ExpectQuery(`^SELECT \* FROM "field_maintenance_images"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "field_maintenance_id", "image_url"}))
	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "role"}))
	expectHealthScoreRecalculation(mock)

	app := newTestApp()
	app.Post("/field-maintenance", withUserLocals(techID.String(), "teknisi"), CreateFieldMaintenance)

	payload := fieldMaintenanceOkPayload(cctvID.String())
	req := httptest.NewRequest("POST", "/field-maintenance", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateFieldMaintenance_InvalidLatitude(t *testing.T) {
	// check_latitude di luar rentang [-90,90] harus ditolak dengan 400.
	app := newTestApp()
	app.Post("/field-maintenance", withUserLocals(uuid.New().String(), "teknisi"), CreateFieldMaintenance)

	payload := `{"cctv_id":"` + uuid.New().String() + `","notes":"Pengecekan",
		"check_latitude":"999","check_longitude":"116.7083",
		"checks":[{"name":"Pengecekan jaringan","result":"ok"}],
		"images":[{"image_url":"/api/uploads/bukti1.jpg"}]}`
	req := httptest.NewRequest("POST", "/field-maintenance", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestCreateFieldMaintenance_InvalidLongitude(t *testing.T) {
	// check_longitude bukan angka valid harus ditolak dengan 400.
	app := newTestApp()
	app.Post("/field-maintenance", withUserLocals(uuid.New().String(), "teknisi"), CreateFieldMaintenance)

	payload := `{"cctv_id":"` + uuid.New().String() + `","notes":"Pengecekan",
		"check_latitude":"-0.9723","check_longitude":"abc",
		"checks":[{"name":"Pengecekan jaringan","result":"ok"}],
		"images":[{"image_url":"/api/uploads/bukti1.jpg"}]}`
	req := httptest.NewRequest("POST", "/field-maintenance", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestCreateFieldMaintenance_WithValidGeotag(t *testing.T) {
	// Payload dengan koordinat geotag valid harus berhasil (201) dan koordinat
	// disertakan saat insert ke tabel field_maintenances.
	_, mock := setupMockGORM(t)
	cctvID := uuid.New()
	techID := uuid.New()

	cctvRows, _ := cctvMockRow()
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(cctvRows)

	// Transaksi: header (dengan geotag) + checks + images
	mock.ExpectBegin()
	mock.ExpectExec(`^INSERT INTO "field_maintenances"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`^INSERT INTO "field_maintenance_checks"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`^INSERT INTO "field_maintenance_checks"`).
		WillReturnResult(sqlmock.NewResult(2, 1))
	mock.ExpectExec(`^INSERT INTO "field_maintenance_images"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Sinkronisasi otomatis: baca status ("aktif") lalu update karena hasil
	// pengecekan "attention" (ada item tidak ok) → "pemeliharaan".
	syncRows, _ := cctvMockRow()
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(syncRows)
	mock.ExpectExec(`^UPDATE "cctvs" SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	// Reload dengan relasi lengkap
	reload := sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"cctv_id", "technician_id", "overall_status", "notes", "check_latitude", "check_longitude", "completed_at",
	}).AddRow(uuid.New(), time.Now(), time.Now(), nil, cctvID, techID, "attention", "Pengecekan", "-0.9723", "116.7083", time.Now())
	mock.ExpectQuery(`^SELECT \* FROM "field_maintenances"`).
		WillReturnRows(reload)
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kode_cctv", "nama", "status"}))
	mock.ExpectQuery(`^SELECT \* FROM "field_maintenance_checks"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "field_maintenance_id", "name", "result", "note"}))
	mock.ExpectQuery(`^SELECT \* FROM "field_maintenance_images"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "field_maintenance_id", "image_url"}))
	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "role"}))
	expectHealthScoreRecalculation(mock)

	app := newTestApp()
	app.Post("/field-maintenance", withUserLocals(techID.String(), "teknisi"), CreateFieldMaintenance)

	payload := `{"cctv_id":"` + cctvID.String() + `","notes":"Pengecekan",
		"check_latitude":"-0.9723","check_longitude":"116.7083",
		"checks":[{"name":"Pengecekan jaringan","result":"ok"},{"name":"Pengecekan listrik","result":"not_ok","note":"Adaptor longgar"}],
		"images":[{"image_url":"/api/uploads/bukti1.jpg"}]}`
	req := httptest.NewRequest("POST", "/field-maintenance", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAllFieldMaintenances_TechnicianScoped(t *testing.T) {
	_, mock := setupMockGORM(t)
	techID := uuid.New()

	mock.ExpectQuery(`^SELECT count\(\*\) FROM "field_maintenances" WHERE technician_id = \$1`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	mock.ExpectQuery(`^SELECT \* FROM "field_maintenances" WHERE technician_id = \$1 AND "field_maintenances"."deleted_at" IS NULL ORDER BY created_at desc LIMIT`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "created_at", "updated_at", "deleted_at",
			"cctv_id", "technician_id", "overall_status", "notes", "check_latitude", "check_longitude", "completed_at",
		}).AddRow(uuid.New(), time.Now(), time.Now(), nil, uuid.New(), techID, "ok", "Bersih", "", "", time.Now()))
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kode_cctv", "nama", "status"}))
	mock.ExpectQuery(`^SELECT \* FROM "field_maintenance_checks"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "field_maintenance_id", "name", "result", "note"}))
	mock.ExpectQuery(`^SELECT \* FROM "field_maintenance_images"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "field_maintenance_id", "image_url"}))
	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "role"}))

	app := newTestApp()
	app.Get("/field-maintenance", withUserLocals(techID.String(), "teknisi"), GetAllFieldMaintenances)

	resp, err := app.Test(httptest.NewRequest("GET", "/field-maintenance", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetFieldMaintenanceByID_InvalidUUID(t *testing.T) {
	app := newTestApp()
	app.Get("/field-maintenance/:id", GetFieldMaintenanceByID)

	resp, err := app.Test(httptest.NewRequest("GET", "/field-maintenance/not-a-uuid", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestGetFieldMaintenanceByID_NotFound(t *testing.T) {
	_, mock := setupMockGORM(t)

	mock.ExpectQuery(`^SELECT \* FROM "field_maintenances"`).
		WillReturnError(gorm.ErrRecordNotFound)

	app := newTestApp()
	app.Get("/field-maintenance/:id", withUserLocals(uuid.New().String(), "teknisi"), GetFieldMaintenanceByID)

	req := httptest.NewRequest("GET", "/field-maintenance/"+uuid.New().String(), nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func expectFieldMaintenanceReportQueries(mock sqlmock.Sqlmock, reportID, cctvID, techID uuid.UUID) {
	now := time.Now()
	mock.ExpectQuery(`^SELECT \* FROM "field_maintenances"`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "created_at", "updated_at", "deleted_at",
			"cctv_id", "technician_id", "maintenance_log_id", "overall_status", "notes", "check_latitude", "check_longitude", "completed_at",
		}).AddRow(reportID, now, now, nil, cctvID, techID, nil, "ok", "Semua item normal", "", "", now))
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "created_at", "updated_at", "deleted_at",
			"kode_cctv", "id_channel", "nama", "lokasi_area",
			"longitude", "latitude", "jenis_cctv", "sumber_listrik",
			"tipe_koneksi", "indoor_outdoor", "pemilik", "status",
			"ip_nvr", "ip_camera", "keterangan", "qr_payload", "health_score",
		}).AddRow(cctvID, now, now, nil, "CCTV-001", "CH-01", "Kamera Utama", "Plaza Seremoni",
			"117.1", "-0.5", "Dome", "PLN", "Fiber", "outdoor", "IKN", "aktif",
			"192.168.1.10", "192.168.1.11", "Aman", "payload", 100))
	mock.ExpectQuery(`^SELECT \* FROM "field_maintenance_checks"`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "created_at", "updated_at", "deleted_at",
			"field_maintenance_id", "name", "result", "note",
		}).AddRow(uuid.New(), now, now, nil, reportID, "Pengecekan jaringan", "ok", ""))
	mock.ExpectQuery(`^SELECT \* FROM "field_maintenance_images"`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "created_at", "updated_at", "deleted_at",
			"field_maintenance_id", "image_url",
		}).AddRow(uuid.New(), now, now, nil, reportID, "/api/uploads/bukti.jpg"))
	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "full_name", "username", "role"}).
			AddRow(techID, "Teknisi Lapangan", "teknisi.ikn", "teknisi"))
}

func TestGenerateFieldMaintenanceReportPDF_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	reportID := uuid.New()
	cctvID := uuid.New()
	techID := uuid.New()
	expectFieldMaintenanceReportQueries(mock, reportID, cctvID, techID)

	app := newTestApp()
	app.Get("/field-maintenance/:id/report-pdf", withUserLocals(uuid.New().String(), "admin"), GenerateFieldMaintenanceReportPDF)

	resp, err := app.Test(httptest.NewRequest("GET", "/field-maintenance/"+reportID.String()+"/report-pdf", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/pdf", resp.Header.Get("Content-Type"))
	assert.Contains(t, resp.Header.Get("Content-Disposition"), "laporan_pengecekan_lapangan_")

	prefix := make([]byte, 4)
	n, _ := resp.Body.Read(prefix)
	assert.Equal(t, "%PDF", string(prefix[:n]), "body harus berupa file PDF valid")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGenerateFieldMaintenanceReportPDF_TechnicianOwner(t *testing.T) {
	_, mock := setupMockGORM(t)
	reportID := uuid.New()
	cctvID := uuid.New()
	techID := uuid.New()
	expectFieldMaintenanceReportQueries(mock, reportID, cctvID, techID)

	app := newTestApp()
	app.Get("/field-maintenance/:id/report-pdf", withUserLocals(techID.String(), "teknisi"), GenerateFieldMaintenanceReportPDF)

	resp, err := app.Test(httptest.NewRequest("GET", "/field-maintenance/"+reportID.String()+"/report-pdf", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/pdf", resp.Header.Get("Content-Type"))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGenerateFieldMaintenanceReportPDF_TechnicianOwnershipDenied(t *testing.T) {
	_, mock := setupMockGORM(t)
	reportID := uuid.New()
	cctvID := uuid.New()
	techID := uuid.New()
	expectFieldMaintenanceReportQueries(mock, reportID, cctvID, techID)

	app := newTestApp()
	app.Get("/field-maintenance/:id/report-pdf", withUserLocals(uuid.New().String(), "teknisi"), GenerateFieldMaintenanceReportPDF)

	resp, err := app.Test(httptest.NewRequest("GET", "/field-maintenance/"+reportID.String()+"/report-pdf", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGenerateFieldMaintenanceReportPDF_NotFound(t *testing.T) {
	_, mock := setupMockGORM(t)
	mock.ExpectQuery(`^SELECT \* FROM "field_maintenances"`).
		WillReturnError(gorm.ErrRecordNotFound)

	app := newTestApp()
	app.Get("/field-maintenance/:id/report-pdf", withUserLocals(uuid.New().String(), "admin"), GenerateFieldMaintenanceReportPDF)

	resp, err := app.Test(httptest.NewRequest("GET", "/field-maintenance/"+uuid.New().String()+"/report-pdf", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteFieldMaintenance_NotFound(t *testing.T) {
	_, mock := setupMockGORM(t)

	mock.ExpectQuery(`^SELECT \* FROM "field_maintenances"`).
		WillReturnError(gorm.ErrRecordNotFound)

	app := newTestApp()
	app.Delete("/field-maintenance/:id", DeleteFieldMaintenance)

	req := httptest.NewRequest("DELETE", "/field-maintenance/"+uuid.New().String(), nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}
