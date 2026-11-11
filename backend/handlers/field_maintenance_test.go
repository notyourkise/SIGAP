package handlers

import (
	"encoding/json"
	"errors"
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

func expectTwoActiveChecklistTemplates(mock sqlmock.Sqlmock) {
	mock.ExpectQuery(`^SELECT \* FROM "checklist_templates"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "nama_item", "urutan", "wajib", "aktif"}).
			AddRow(uuid.New(), "Pengecekan jaringan", 1, true, true).
			AddRow(uuid.New(), "Pengecekan listrik", 2, true, true))
}

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
	expectTwoActiveChecklistTemplates(mock)

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
	expectTwoActiveChecklistTemplates(mock)

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
	expectTwoActiveChecklistTemplates(mock)

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
	expectTwoActiveChecklistTemplates(mock)

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

func TestCreateFieldMaintenance_SiteMismatch(t *testing.T) {
	_, mock := setupMockGORM(t)
	cctvID := uuid.New()
	techID := uuid.New()
	activeSite := uuid.New()
	otherSite := uuid.New()

	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kode_cctv", "nama", "status", "site_id"}).
			AddRow(cctvID, "CCTV-001", "Kamera", "aktif", otherSite))

	app := newTestApp()
	app.Post("/field-maintenance", withUserSiteLocals(techID.String(), "teknisi", activeSite.String()), CreateFieldMaintenance)

	payload := fieldMaintenanceValidPayload(cctvID.String())
	req := httptest.NewRequest("POST", "/field-maintenance", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAllFieldMaintenances_SiteScoped(t *testing.T) {
	_, mock := setupMockGORM(t)
	techID := uuid.New()
	siteID := uuid.New()

	mock.ExpectQuery(`^SELECT count\(\*\) FROM "field_maintenances" WHERE .*site_id = \$1.*technician_id = \$2`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`^SELECT \* FROM "field_maintenances" WHERE .*site_id = \$1.*technician_id = \$2.*ORDER BY created_at desc LIMIT`).
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
	app.Get("/field-maintenance", withUserSiteLocals(techID.String(), "teknisi", siteID.String()), GetAllFieldMaintenances)

	resp, err := app.Test(httptest.NewRequest("GET", "/field-maintenance", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetFieldMaintenanceByID_SiteMismatch(t *testing.T) {
	_, mock := setupMockGORM(t)
	reportID := uuid.New()
	cctvID := uuid.New()
	techID := uuid.New()
	activeSite := uuid.New()
	otherSite := uuid.New()
	now := time.Now()

	mock.ExpectQuery(`^SELECT \* FROM "field_maintenances"`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "created_at", "updated_at", "deleted_at",
			"cctv_id", "technician_id", "overall_status", "notes", "check_latitude", "check_longitude", "completed_at",
		}).AddRow(reportID, now, now, nil, cctvID, techID, "ok", "Bersih", "", "", now))
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "kode_cctv", "nama", "status", "site_id"}).
			AddRow(cctvID, now, now, nil, "CCTV-001", "Kamera", "aktif", otherSite))
	mock.ExpectQuery(`^SELECT \* FROM "field_maintenance_checks"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "field_maintenance_id", "name", "result", "note"}))
	mock.ExpectQuery(`^SELECT \* FROM "field_maintenance_images"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "field_maintenance_id", "image_url"}))
	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "role"}))

	app := newTestApp()
	app.Get("/field-maintenance/:id", withUserSiteLocals(techID.String(), "teknisi", activeSite.String()), GetFieldMaintenanceByID)

	resp, err := app.Test(httptest.NewRequest("GET", "/field-maintenance/"+reportID.String(), nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
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

// H07: Memastikan CreateFieldMaintenance juga menolak checklist karangan dan foto bukti fiktif.
func TestAudit_FieldMaintenanceMustRejectInventedChecklistAndFakePhoto(t *testing.T) {
	_, mock := setupMockGORM(t)
	cctvID := uuid.New()
	techID := uuid.New()

	app := newTestApp()
	app.Post("/field-maintenance", withUserLocals(techID.String(), "teknisi"), CreateFieldMaintenance)

	// 1. Foto bukti fiktif (tidak ada di folder uploads) ditolak sebelum menyentuh DB
	fakePhoto := `{"cctv_id":"` + cctvID.String() + `","checks":[{"name":"Pengecekan jaringan","result":"ok"}],"images":[{"image_url":"/api/uploads/does-not-exist.jpg"}]}`
	reqFake := httptest.NewRequest("POST", "/field-maintenance", strings.NewReader(fakePhoto))
	reqFake.Header.Set("Content-Type", "application/json")
	respFake, err := app.Test(reqFake)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, respFake.StatusCode)

	// 2. Foto bukti valid tetapi checklist karangan / tidak mencakup seluruh template wajib -> 400
	cctvRows, _ := cctvMockRow()
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(cctvRows)
	expectTwoActiveChecklistTemplates(mock)

	inventedChecks := `{"cctv_id":"` + cctvID.String() + `","checks":[{"name":"Item Karangan","result":"ok"}],"images":[{"image_url":"/api/uploads/bukti1.jpg"}]}`
	reqInvented := httptest.NewRequest("POST", "/field-maintenance", strings.NewReader(inventedChecks))
	reqInvented.Header.Set("Content-Type", "application/json")
	respInvented, err := app.Test(reqInvented)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, respInvented.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// L02 (#190): Memastikan validasi koordinat lapangan menolak NaN, Inf, dan -Inf.
func TestAudit_FieldCoordinatesMustRejectNaN(t *testing.T) {
	assert.Error(t, validateCoordinates("NaN", "116.7"))
	assert.Error(t, validateCoordinates("nan", "116.7"))
	assert.Error(t, validateCoordinates("-0.97", "Inf"))
	assert.Error(t, validateCoordinates("-0.97", "+Inf"))
	assert.Error(t, validateCoordinates("-Inf", "116.7"))
	assert.Error(t, validateCoordinates("-90.1", "116.7"))
	assert.Error(t, validateCoordinates("-0.97", "180.1"))
	assert.NoError(t, validateCoordinates("", ""))
	assert.NoError(t, validateCoordinates("-90", "180"))

	_, mock := setupMockGORM(t)
	cctvID := uuid.New()
	techID := uuid.New()

	app := newTestApp()
	app.Post("/field-maintenance", withUserLocals(techID.String(), "teknisi"), CreateFieldMaintenance)

	badCoords := []struct {
		lat string
		lng string
	}{
		{"NaN", "116.7092"},
		{"-0.9721", "nan"},
		{"Inf", "116.7092"},
		{"-0.9721", "-Inf"},
		{"+Inf", "116.7092"},
	}
	for _, tc := range badCoords {
		payload := `{"cctv_id":"` + cctvID.String() + `","check_latitude":"` + tc.lat + `","check_longitude":"` + tc.lng + `",
			"checks":[{"name":"Pengecekan jaringan","result":"ok"}],
			"images":[{"image_url":"/api/uploads/bukti1.jpg"}]}`
		req := httptest.NewRequest("POST", "/field-maintenance", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	}
	assert.NoError(t, mock.ExpectationsWereMet())
}

// M14 (#187): Replay client_request_id mengembalikan catatan yang sudah ada dengan HTTP 200 idempotent:true
// serta meredaksi metadata RTSP bersarang tanpa membuat insert ganda.
func TestCreateFieldMaintenance_IdempotencyReplay(t *testing.T) {
	_, mock := setupMockGORM(t)
	cctvID := uuid.New()
	techID := uuid.New()
	entryID := uuid.New()
	reqID := "req-offline-abc-123"

	cctvRows := cctvMockRowWithIdentity(cctvID, "encrypted-qr", "aktif")
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).WillReturnRows(cctvRows)

	existingRows := sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"cctv_id", "technician_id", "overall_status", "notes", "check_latitude", "check_longitude", "client_request_id", "completed_at",
	}).AddRow(entryID, time.Now(), time.Now(), nil, cctvID, techID, "ok", "Semua item normal", "-0.97", "116.7", reqID, time.Now())
	mock.ExpectQuery(`^SELECT \* FROM "field_maintenances"`).WillReturnRows(existingRows)
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kode_cctv", "nama", "status", "rtsp_url", "rtsp_password"}).
			AddRow(cctvID, "CAM-01", "Kamera Utama", "aktif", "rtsp://admin:replay-secret@10.0.0.1:554/ch1", "pw-secret"))
	mock.ExpectQuery(`^SELECT \* FROM "field_maintenance_checks"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "field_maintenance_id", "name", "result"}))
	mock.ExpectQuery(`^SELECT \* FROM "field_maintenance_images"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "field_maintenance_id", "image_url"}))
	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "role"}).AddRow(techID, "teknisi1", "teknisi"))

	app := newTestApp()
	app.Post("/field-maintenance", withUserLocals(techID.String(), "teknisi"), CreateFieldMaintenance)

	payload := `{"cctv_id":"` + cctvID.String() + `","client_request_id":"` + reqID + `","notes":"Semua item normal",
		"checks":[{"name":"Pengecekan jaringan","result":"ok"}],
		"images":[{"image_url":"/api/uploads/bukti1.jpg"}]}`
	req := httptest.NewRequest("POST", "/field-maintenance", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "success", body["status"])
	assert.Equal(t, true, body["idempotent"])

	encoded, err := json.Marshal(body)
	assert.NoError(t, err)
	assert.NotContains(t, string(encoded), "replay-secret")
	assert.NotContains(t, string(encoded), "pw-secret")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// M14 (#187): Penggunaan ulang client_request_id untuk CCTV yang berbeda ditolak HTTP 409 Conflict.
func TestCreateFieldMaintenance_IdempotencyConflictDifferentCCTV(t *testing.T) {
	_, mock := setupMockGORM(t)
	cctvID := uuid.New()
	otherCCTVID := uuid.New()
	techID := uuid.New()
	entryID := uuid.New()
	reqID := "req-offline-reuse"

	cctvRows := cctvMockRowWithIdentity(cctvID, "encrypted-qr", "aktif")
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).WillReturnRows(cctvRows)

	existingRows := sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"cctv_id", "technician_id", "overall_status", "notes", "client_request_id",
	}).AddRow(entryID, time.Now(), time.Now(), nil, otherCCTVID, techID, "ok", "Kamera lain", reqID)
	mock.ExpectQuery(`^SELECT \* FROM "field_maintenances"`).WillReturnRows(existingRows)
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kode_cctv", "nama", "status"}))
	mock.ExpectQuery(`^SELECT \* FROM "field_maintenance_checks"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "field_maintenance_id", "name", "result"}))
	mock.ExpectQuery(`^SELECT \* FROM "field_maintenance_images"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "field_maintenance_id", "image_url"}))
	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "role"}))

	app := newTestApp()
	app.Post("/field-maintenance", withUserLocals(techID.String(), "teknisi"), CreateFieldMaintenance)

	payload := `{"cctv_id":"` + cctvID.String() + `","client_request_id":"` + reqID + `",
		"checks":[{"name":"Pengecekan jaringan","result":"ok"}],
		"images":[{"image_url":"/api/uploads/bukti1.jpg"}]}`
	req := httptest.NewRequest("POST", "/field-maintenance", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusConflict, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// M14 (#187): Race condition concurrent submit ditangani melalui recovery SQLSTATE 23505 menjadi HTTP 200 idempotent:true.
func TestCreateFieldMaintenance_IdempotencyUniqueViolationRace(t *testing.T) {
	_, mock := setupMockGORM(t)
	cctvID := uuid.New()
	techID := uuid.New()
	entryID := uuid.New()
	reqID := "req-offline-race-23505"

	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(cctvMockRowWithIdentity(cctvID, "encrypted-qr", "aktif"))
	mock.ExpectQuery(`^SELECT \* FROM "field_maintenances"`).
		WillReturnError(gorm.ErrRecordNotFound)
	expectTwoActiveChecklistTemplates(mock)

	mock.ExpectBegin()
	mock.ExpectExec(`^INSERT INTO "field_maintenances"`).
		WillReturnError(errors.New(`ERROR: duplicate key value violates unique constraint "uq_field_maintenances_tech_client_req" (SQLSTATE 23505)`))
	mock.ExpectRollback()

	existingRows := sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"cctv_id", "technician_id", "overall_status", "notes", "client_request_id", "completed_at",
	}).AddRow(entryID, time.Now(), time.Now(), nil, cctvID, techID, "ok", "Concurrent OK", reqID, time.Now())
	mock.ExpectQuery(`^SELECT \* FROM "field_maintenances"`).WillReturnRows(existingRows)
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kode_cctv", "nama", "status", "rtsp_url", "rtsp_password"}).
			AddRow(cctvID, "CAM-01", "Kamera Race", "aktif", "rtsp://admin:race-secret@10.0.0.2:554/ch1", "pw-race-secret"))
	mock.ExpectQuery(`^SELECT \* FROM "field_maintenance_checks"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "field_maintenance_id", "name", "result"}))
	mock.ExpectQuery(`^SELECT \* FROM "field_maintenance_images"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "field_maintenance_id", "image_url"}))
	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "role"}).AddRow(techID, "teknisi1", "teknisi"))

	app := newTestApp()
	app.Post("/field-maintenance", withUserLocals(techID.String(), "teknisi"), CreateFieldMaintenance)
	payload := `{"cctv_id":"` + cctvID.String() + `","client_request_id":"` + reqID + `",
		"checks":[{"name":"Pengecekan jaringan","result":"ok"},{"name":"Pengecekan listrik","result":"ok"}],
		"images":[{"image_url":"/api/uploads/bukti1.jpg"}]}`
	req := httptest.NewRequest("POST", "/field-maintenance", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "success", body["status"])
	assert.Equal(t, true, body["idempotent"])

	encoded, err := json.Marshal(body)
	assert.NoError(t, err)
	assert.NotContains(t, string(encoded), "race-secret")
	assert.NotContains(t, string(encoded), "pw-race-secret")
	assert.NoError(t, mock.ExpectationsWereMet())
}
