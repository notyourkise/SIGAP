package handlers

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"sigap-backend/config"
	"sigap-backend/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// sparePartRows membangun baris tabel spare_parts untuk mock sqlmock.
func sparePartRows(id uuid.UUID, nama, sku, kategori string, stok, minOrder int) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"nama", "sku", "kategori", "stok", "minimum_order", "satuan", "harga_satuan", "keterangan",
	}).AddRow(id, time.Now(), time.Now(), nil, nama, sku, kategori, stok, minOrder, "pcs", 15000, "Kabel UTP Cat6")
}

// sparePartUsageRows membangun baris tabel spare_part_usages untuk mock.
func sparePartUsageRows(usageID, logID, spareID uuid.UUID, jumlah int) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"maintenance_log_id", "spare_part_id", "jumlah", "catatan",
	}).AddRow(usageID, time.Now(), time.Now(), nil, logID, spareID, jumlah, "Penggantian kabel")
}

// maintenanceLogCompletedRow membangun baris maintenance_logs berstatus completed.
func maintenanceLogCompletedRow(id, cctvID uuid.UUID) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"cctv_id", "technician_id", "action_taken", "notes", "status",
		"scheduled_date", "completed_at", "image_url",
	}).AddRow(id, time.Now(), time.Now(), nil, cctvID, nil, "Bersihkan lensa", "Selesai", "completed", time.Now(), time.Now(), "")
}

// TestCreateSparePart_Success menguji pembuatan suku cadang baru.
func TestCreateSparePart_Success(t *testing.T) {
	_, mock := setupMockGORM(t)

	// SKU belum dipakai (tidak ada baris yang cocok) → 404 dari query Unscoped.
	mock.ExpectQuery(`^SELECT \* FROM "spare_parts"`).
		WillReturnError(gorm.ErrRecordNotFound)
	mock.ExpectExec(`^INSERT INTO "spare_parts"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	app := newTestApp()
	app.Post("/spare-parts", withUserLocals(uuid.New().String(), "admin"), CreateSparePart)

	body := `{"nama":"Kabel UTP Cat6 5m","sku":"KBL-UTP-005","kategori":"Kabel","stok":20,"minimum_order":5,"satuan":"pcs","harga_satuan":15000,"keterangan":"Kabel UTP Cat6"}`
	req := httptest.NewRequest("POST", "/spare-parts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCreateSparePart_DuplicateSKU memastikan SKU duplikat ditolak (400).
func TestCreateSparePart_DuplicateSKU(t *testing.T) {
	_, mock := setupMockGORM(t)
	id := uuid.New()

	mock.ExpectQuery(`^SELECT \* FROM "spare_parts"`).
		WillReturnRows(sparePartRows(id, "Kabel UTP", "KBL-UTP-005", "Kabel", 10, 5))

	app := newTestApp()
	app.Post("/spare-parts", withUserLocals(uuid.New().String(), "admin"), CreateSparePart)

	req := httptest.NewRequest("POST", "/spare-parts",
		strings.NewReader(`{"nama":"Kabel UTP Cat6","sku":"KBL-UTP-005","stok":10,"minimum_order":5}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetAllSpareParts_Success menguji daftar suku cadang terpaginasi.
func TestGetAllSpareParts_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	id := uuid.New()

	mock.ExpectQuery(`^SELECT count\(\*\) FROM "spare_parts"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`^SELECT \* FROM "spare_parts"`).
		WillReturnRows(sparePartRows(id, "Kabel UTP Cat6", "KBL-UTP-005", "Kabel", 20, 5))

	app := newTestApp()
	app.Get("/spare-parts", withUserLocals(uuid.New().String(), "admin"), GetAllSpareParts)

	req := httptest.NewRequest("GET", "/spare-parts?page=1&limit=10", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	data := result["data"].(map[string]interface{})
	pagination := data["pagination"].(map[string]interface{})
	assert.Equal(t, float64(1), pagination["total_rows"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetAllSpareParts_FilterSite memastikan filter kawasan diterapkan pada
// query count dan daftar sehingga pagination serta baris selalu konsisten.
func TestGetAllSpareParts_FilterSite(t *testing.T) {
	_, mock := setupMockGORM(t)
	itemID, siteID := uuid.New(), uuid.New()

	mock.ExpectQuery(`^SELECT count\(\*\) FROM "spare_parts" WHERE site_id = \$1`).
		WithArgs(siteID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`^SELECT \* FROM "spare_parts" WHERE site_id = \$1`).
		WithArgs(siteID, 10).
		WillReturnRows(sparePartRows(itemID, "Kabel Zona", "KBL-ZONA", "Kabel", 5, 2))

	app := newTestApp()
	app.Get("/spare-parts", withUserLocals(uuid.New().String(), "admin"), GetAllSpareParts)

	req := httptest.NewRequest("GET", "/spare-parts?page=1&limit=10&site_id="+siteID.String(), nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetSparePartByID_Success menguji pengambilan detail suku cadang.
func TestGetSparePartByID_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	id := uuid.New()

	mock.ExpectQuery(`^SELECT \* FROM "spare_parts"`).
		WillReturnRows(sparePartRows(id, "Konektor RJ45", "KNT-RJ45", "Konektor", 50, 10))

	app := newTestApp()
	app.Get("/spare-parts/:id", withUserLocals(uuid.New().String(), "admin"), GetSparePartByID)

	req := httptest.NewRequest("GET", "/spare-parts/"+id.String(), nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	data := result["data"].(map[string]interface{})
	assert.Equal(t, "Konektor RJ45", data["nama"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetSparePartByID_NotFound memastikan 404 untuk id yang tidak ada.
func TestGetSparePartByID_NotFound(t *testing.T) {
	_, mock := setupMockGORM(t)

	mock.ExpectQuery(`^SELECT \* FROM "spare_parts"`).
		WillReturnError(gorm.ErrRecordNotFound)

	app := newTestApp()
	app.Get("/spare-parts/:id", withUserLocals(uuid.New().String(), "admin"), GetSparePartByID)

	req := httptest.NewRequest("GET", "/spare-parts/"+uuid.New().String(), nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestUpdateSparePart_Success menguji pembaruan suku cadang (stok).
func TestUpdateSparePart_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	id := uuid.New()

	mock.ExpectQuery(`^SELECT \* FROM "spare_parts"`).
		WillReturnRows(sparePartRows(id, "Kabel UTP Cat6", "KBL-UTP-005", "Kabel", 20, 5))
	mock.ExpectExec(`^UPDATE "spare_parts" SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	app := newTestApp()
	app.Put("/spare-parts/:id", withUserLocals(uuid.New().String(), "admin"), UpdateSparePart)

	req := httptest.NewRequest("PUT", "/spare-parts/"+id.String(),
		strings.NewReader(`{"stok":30,"keterangan":"Stok diperbarui"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestDeleteSparePart_Success menguji soft delete suku cadang.
func TestDeleteSparePart_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	id := uuid.New()

	mock.ExpectQuery(`^SELECT \* FROM "spare_parts"`).
		WillReturnRows(sparePartRows(id, "Kabel UTP Cat6", "KBL-UTP-005", "Kabel", 20, 5))
	mock.ExpectExec(`^UPDATE "spare_parts" SET "deleted_at"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	app := newTestApp()
	app.Delete("/spare-parts/:id", withUserLocals(uuid.New().String(), "admin"), DeleteSparePart)

	req := httptest.NewRequest("DELETE", "/spare-parts/"+id.String(), nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestRecordSparePartUsage_Success menguji pencatatan penggunaan yang
// mengurangi stok otomatis dalam transaksi.
func TestRecordSparePartUsage_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	logID, cctvID, spareID := uuid.New(), uuid.New(), uuid.New()

	// Validasi maintenance log (status completed).
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(maintenanceLogCompletedRow(logID, cctvID))
	// Transaksi: muat suku cadang.
	mock.ExpectBegin()
	mock.ExpectQuery(`^SELECT \* FROM "spare_parts"`).
		WillReturnRows(sparePartRows(spareID, "Kabel UTP Cat6", "KBL-UTP-005", "Kabel", 10, 5))
	// Kurangi stok.
	mock.ExpectExec(`^UPDATE "spare_parts" SET "stok"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	// Buat catatan penggunaan.
	mock.ExpectExec(`^INSERT INTO "spare_part_usages"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	// Muat ulang dengan preload suku cadang.
	mock.ExpectQuery(`^SELECT \* FROM "spare_part_usages"`).
		WillReturnRows(sparePartUsageRows(uuid.New(), logID, spareID, 2))
	mock.ExpectQuery(`^SELECT \* FROM "spare_parts"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "nama", "sku"}).AddRow(spareID, "Kabel UTP Cat6", "KBL-UTP-005"))

	app := newTestApp()
	app.Post("/maintenance-logs/:id/spare-parts", withUserLocals(uuid.New().String(), "admin"), RecordSparePartUsage)

	body := `{"spare_part_id":"` + spareID.String() + `","jumlah":2,"catatan":"Penggantian kabel"}`
	req := httptest.NewRequest("POST", "/maintenance-logs/"+logID.String()+"/spare-parts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestRecordSparePartUsage_InsufficientStock memastikan stok tidak cukup
// ditolak dengan 400 tanpa mengurangi stok.
func TestRecordSparePartUsage_InsufficientStock(t *testing.T) {
	_, mock := setupMockGORM(t)
	logID, cctvID, spareID := uuid.New(), uuid.New(), uuid.New()

	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(maintenanceLogCompletedRow(logID, cctvID))
	mock.ExpectBegin()
	mock.ExpectQuery(`^SELECT \* FROM "spare_parts"`).
		WillReturnRows(sparePartRows(spareID, "Kabel UTP Cat6", "KBL-UTP-005", "Kabel", 1, 5))
	// Stok tidak cukup → transaksi di-rollback.
	mock.ExpectRollback()

	app := newTestApp()
	app.Post("/maintenance-logs/:id/spare-parts", withUserLocals(uuid.New().String(), "admin"), RecordSparePartUsage)

	body := `{"spare_part_id":"` + spareID.String() + `","jumlah":5}`
	req := httptest.NewRequest("POST", "/maintenance-logs/"+logID.String()+"/spare-parts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "error", result["status"])
	assert.Contains(t, result["message"], "Stok tidak mencukupi")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestRecordSparePartUsage_LogNotCompleted memastikan penggunaan hanya
// dicatat pada log berstatus completed (409).
func TestRecordSparePartUsage_LogNotCompleted(t *testing.T) {
	_, mock := setupMockGORM(t)
	logID, cctvID := uuid.New(), uuid.New()

	// Log berstatus "ongoing" — bukan completed.
	rows := sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"cctv_id", "technician_id", "action_taken", "notes", "status",
		"scheduled_date", "completed_at", "image_url",
	}).AddRow(logID, time.Now(), time.Now(), nil, cctvID, nil, "Bersihkan", "Proses", "ongoing", time.Now(), nil, "")
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).WillReturnRows(rows)

	app := newTestApp()
	app.Post("/maintenance-logs/:id/spare-parts", withUserLocals(uuid.New().String(), "admin"), RecordSparePartUsage)

	body := `{"spare_part_id":"` + uuid.New().String() + `","jumlah":1}`
	req := httptest.NewRequest("POST", "/maintenance-logs/"+logID.String()+"/spare-parts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusConflict, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetSparePartUsageByLog_Success menguji daftar penggunaan per log.
func TestGetSparePartUsageByLog_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	logID, spareID := uuid.New(), uuid.New()

	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(maintenanceLogCompletedRow(logID, uuid.New()))
	mock.ExpectQuery(`^SELECT \* FROM "spare_part_usages"`).
		WillReturnRows(sparePartUsageRows(uuid.New(), logID, spareID, 2))
	mock.ExpectQuery(`^SELECT \* FROM "spare_parts"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "nama", "sku"}).AddRow(spareID, "Kabel UTP Cat6", "KBL-UTP-005"))

	app := newTestApp()
	app.Get("/maintenance-logs/:id/spare-parts", withUserLocals(uuid.New().String(), "admin"), GetSparePartUsageByLog)

	req := httptest.NewRequest("GET", "/maintenance-logs/"+logID.String()+"/spare-parts", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetLowStockSpareParts_Success menguji daftar suku cadang stok rendah.
func TestGetLowStockSpareParts_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	id := uuid.New()

	mock.ExpectQuery(`^SELECT \* FROM "spare_parts"`).
		WillReturnRows(sparePartRows(id, "Power Supply 12V", "PSU-12V", "Power Supply", 2, 5))

	app := newTestApp()
	app.Get("/spare-parts/low-stock", withUserLocals(uuid.New().String(), "admin"), GetLowStockSpareParts)

	req := httptest.NewRequest("GET", "/spare-parts/low-stock", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func sparePartRowForSite(id, siteID uuid.UUID, nama, sku, kategori string, stok, minOrder int) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"site_id", "nama", "sku", "kategori", "stok", "minimum_order", "satuan", "harga_satuan", "keterangan",
	}).AddRow(id, time.Now(), time.Now(), nil, siteID, nama, sku, kategori, stok, minOrder, "pcs", 15000, "Kabel UTP Cat6")
}

func maintenanceLogCompletedRowForTechnician(id, cctvID, techID uuid.UUID) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"cctv_id", "technician_id", "action_taken", "notes", "status",
		"scheduled_date", "completed_at", "image_url",
	}).AddRow(id, time.Now(), time.Now(), nil, cctvID, techID, "Bersihkan lensa", "Selesai", "completed", time.Now(), time.Now(), "")
}

// H03: Teknisi yang bukan pemilik tugas pemeliharaan wajib ditolak (403)
// saat mencoba mencatat pemakaian suku cadang tanpa mengurangi stok.
func TestAudit_SparePartWriteMustDenyOtherTechnician(t *testing.T) {
	_, mock := setupMockGORM(t)
	logID, cctvID, spareID, ownerTechID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	otherTechID := uuid.New()
	siteID := uuid.New()

	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(maintenanceLogCompletedRowForTechnician(logID, cctvID, ownerTechID))

	app := newTestApp()
	app.Post("/maintenance-logs/:id/spare-parts",
		withUserSiteLocals(otherTechID.String(), "teknisi", siteID.String()),
		RecordSparePartUsage,
	)

	body := `{"spare_part_id":"` + spareID.String() + `","jumlah":2}`
	req := httptest.NewRequest("POST", "/maintenance-logs/"+logID.String()+"/spare-parts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// H03: Teknisi yang bukan pemilik tugas pemeliharaan wajib ditolak (403)
// saat mencoba membaca daftar pemakaian suku cadang pada tugas teknisi lain.
func TestAudit_SparePartReadMustDenyOtherTechnician(t *testing.T) {
	_, mock := setupMockGORM(t)
	logID, cctvID, ownerTechID := uuid.New(), uuid.New(), uuid.New()
	otherTechID := uuid.New()
	siteID := uuid.New()

	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(maintenanceLogCompletedRowForTechnician(logID, cctvID, ownerTechID))

	app := newTestApp()
	app.Get("/maintenance-logs/:id/spare-parts",
		withUserSiteLocals(otherTechID.String(), "teknisi", siteID.String()),
		GetSparePartUsageByLog,
	)

	req := httptest.NewRequest("GET", "/maintenance-logs/"+logID.String()+"/spare-parts", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// H03: Teknisi pemilik tugas di site yang sama diizinkan mencatat & membaca penggunaan
// suku cadang dari site yang sesuai, sementara akses lintas site pada log maupun
// suku cadang ditolak dengan 403.
func TestSparePartUsage_SiteAndOwnershipBoundaries(t *testing.T) {
	siteA := uuid.New()
	siteB := uuid.New()
	ownerTechID := uuid.New()
	cctvID := uuid.New()
	logID := uuid.New()
	spareID := uuid.New()

	t.Run("OwnerTechnicianAllowedWhenLogAndPartMatchSite", func(t *testing.T) {
		_, mock := setupMockGORM(t)

		mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
			WillReturnRows(maintenanceLogCompletedRowForTechnician(logID, cctvID, ownerTechID))
		mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
			WillReturnRows(cctvRowForSite(cctvID, siteA))
		mock.ExpectBegin()
		mock.ExpectQuery(`^SELECT \* FROM "spare_parts"`).
			WillReturnRows(sparePartRowForSite(spareID, siteA, "Kabel UTP Site A", "KBL-A", "Kabel", 10, 2))
		mock.ExpectExec(`^UPDATE "spare_parts" SET "stok"`).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec(`^INSERT INTO "spare_part_usages"`).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()
		mock.ExpectQuery(`^SELECT \* FROM "spare_part_usages"`).
			WillReturnRows(sparePartUsageRows(uuid.New(), logID, spareID, 2))
		mock.ExpectQuery(`^SELECT \* FROM "spare_parts"`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "nama", "sku"}).AddRow(spareID, "Kabel UTP Site A", "KBL-A"))

		app := newTestApp()
		app.Post("/maintenance-logs/:id/spare-parts",
			withUserSiteLocals(ownerTechID.String(), "teknisi", siteA.String()),
			RecordSparePartUsage,
		)

		body := `{"spare_part_id":"` + spareID.String() + `","jumlah":2}`
		req := httptest.NewRequest("POST", "/maintenance-logs/"+logID.String()+"/spare-parts", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("DenyWhenLogCCTVInOtherSiteEvenIfOwner", func(t *testing.T) {
		_, mock := setupMockGORM(t)

		mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
			WillReturnRows(maintenanceLogCompletedRowForTechnician(logID, cctvID, ownerTechID))
		mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
			WillReturnRows(cctvRowForSite(cctvID, siteB))

		app := newTestApp()
		app.Post("/maintenance-logs/:id/spare-parts",
			withUserSiteLocals(ownerTechID.String(), "teknisi", siteA.String()),
			RecordSparePartUsage,
		)

		body := `{"spare_part_id":"` + spareID.String() + `","jumlah":1}`
		req := httptest.NewRequest("POST", "/maintenance-logs/"+logID.String()+"/spare-parts", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("DenyWhenSparePartBelongsToOtherSite", func(t *testing.T) {
		_, mock := setupMockGORM(t)

		mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
			WillReturnRows(maintenanceLogCompletedRowForTechnician(logID, cctvID, ownerTechID))
		mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
			WillReturnRows(cctvRowForSite(cctvID, siteA))
		mock.ExpectBegin()
		mock.ExpectQuery(`^SELECT \* FROM "spare_parts"`).
			WillReturnRows(sparePartRowForSite(spareID, siteB, "Kabel UTP Site B", "KBL-B", "Kabel", 10, 2))
		mock.ExpectRollback()

		app := newTestApp()
		app.Post("/maintenance-logs/:id/spare-parts",
			withUserSiteLocals(ownerTechID.String(), "teknisi", siteA.String()),
			RecordSparePartUsage,
		)

		body := `{"spare_part_id":"` + spareID.String() + `","jumlah":1}`
		req := httptest.NewRequest("POST", "/maintenance-logs/"+logID.String()+"/spare-parts", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("DenyAdminWhenSparePartSiteMismatchesTaskCCTVSite", func(t *testing.T) {
		_, mock := setupMockGORM(t)

		mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
			WillReturnRows(maintenanceLogCompletedRow(logID, cctvID))
		mock.ExpectBegin()
		mock.ExpectQuery(`^SELECT \* FROM "spare_parts"`).
			WillReturnRows(sparePartRowForSite(spareID, siteB, "Kabel UTP Site B", "KBL-B", "Kabel", 10, 2))
		mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
			WillReturnRows(cctvRowForSite(cctvID, siteA))
		mock.ExpectRollback()

		app := newTestApp()
		app.Post("/maintenance-logs/:id/spare-parts",
			withUserLocals(uuid.New().String(), "admin"),
			RecordSparePartUsage,
		)

		body := `{"spare_part_id":"` + spareID.String() + `","jumlah":1}`
		req := httptest.NewRequest("POST", "/maintenance-logs/"+logID.String()+"/spare-parts", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("ReadUsageDeniedWhenLogCCTVInOtherSiteEvenIfOwner", func(t *testing.T) {
		_, mock := setupMockGORM(t)

		mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
			WillReturnRows(maintenanceLogCompletedRowForTechnician(logID, cctvID, ownerTechID))
		mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
			WillReturnRows(cctvRowForSite(cctvID, siteB))

		app := newTestApp()
		app.Get("/maintenance-logs/:id/spare-parts",
			withUserSiteLocals(ownerTechID.String(), "teknisi", siteA.String()),
			GetSparePartUsageByLog,
		)

		req := httptest.NewRequest("GET", "/maintenance-logs/"+logID.String()+"/spare-parts", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

type h05StockDecrementCapture struct {
	mu           *sync.Mutex
	matched      *bool
	currentStock *int64
	history      *[]int64
}

func (c h05StockDecrementCapture) Match(value driver.Value) bool {
	delta, ok := value.(int64)
	if !ok || delta <= 0 {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if *c.matched {
		return true
	}
	if *c.currentStock < delta {
		return false
	}
	*c.matched = true
	*c.currentStock -= delta
	*c.history = append(*c.history, *c.currentStock)
	return true
}

// H05: Dua pemakaian suku cadang bersamaan masing-masing 4 dari stok awal 10
// wajib mengurangi stok secara berurutan/atomik menjadi 2 dengan 2 catatan penggunaan.
func TestAudit_ConcurrentStockUsageMustDeductBothRequests(t *testing.T) {
	_, mock := setupMockGORM(t)
	mock.MatchExpectationsInOrder(false)
	logID, cctvID, spareID := uuid.New(), uuid.New(), uuid.New()

	var mu sync.Mutex
	currentStock := int64(10)
	var balances []int64

	// Transaksi 1 (membaca stok 10 dengan FOR UPDATE -> dikurangi 4 -> sisa 6).
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(maintenanceLogCompletedRow(logID, cctvID))
	mock.ExpectBegin()
	mock.ExpectQuery(`^SELECT \* FROM "spare_parts".*FOR UPDATE`).
		WillDelayFor(30 * time.Millisecond).
		WillReturnRows(sparePartRows(spareID, "Kabel UTP Audit", "AUDIT-H05", "Kabel", 10, 0))
	mock.ExpectExec(`^UPDATE "spare_parts" SET "stok"=stok - \$1`).
		WithArgs(h05StockDecrementCapture{mu: &mu, matched: new(bool), currentStock: &currentStock, history: &balances}, sqlmock.AnyArg(), spareID, 4).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`^INSERT INTO "spare_part_usages"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	mock.ExpectQuery(`^SELECT \* FROM "spare_part_usages"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	// Transaksi 2 (terserialisasi oleh lock & FOR UPDATE -> membaca stok 6 -> dikurangi 4 -> sisa 2).
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(maintenanceLogCompletedRow(logID, cctvID))
	mock.ExpectBegin()
	mock.ExpectQuery(`^SELECT \* FROM "spare_parts".*FOR UPDATE`).
		WillDelayFor(30 * time.Millisecond).
		WillReturnRows(sparePartRows(spareID, "Kabel UTP Audit", "AUDIT-H05", "Kabel", 6, 0))
	mock.ExpectExec(`^UPDATE "spare_parts" SET "stok"=stok - \$1`).
		WithArgs(h05StockDecrementCapture{mu: &mu, matched: new(bool), currentStock: &currentStock, history: &balances}, sqlmock.AnyArg(), spareID, 4).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`^INSERT INTO "spare_part_usages"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	mock.ExpectQuery(`^SELECT \* FROM "spare_part_usages"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	app := newTestApp()
	app.Post("/maintenance-logs/:id/spare-parts", withUserLocals(uuid.NewString(), "admin"), RecordSparePartUsage)

	statuses := make(chan int, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			body := `{"spare_part_id":"` + spareID.String() + `","jumlah":4}`
			req := httptest.NewRequest("POST", "/maintenance-logs/"+logID.String()+"/spare-parts", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req, -1)
			if err != nil {
				statuses <- 0
				return
			}
			statuses <- resp.StatusCode
		}()
	}
	wg.Wait()
	close(statuses)

	for status := range statuses {
		assert.Equal(t, fiber.StatusCreated, status)
	}
	assert.NoError(t, mock.ExpectationsWereMet())
	assert.Equal(t, int64(2), currentStock, "stok awal 10 dikurangi dua pemakaian @4 wajib menyisakan 2")
	assert.Equal(t, []int64{6, 2}, balances)
}

// H05: Pemakaian total yang melebihi stok wajib ditolak secara atomik (400 + Rollback),
// baik ketika terserialisasi maupun ketika UPDATE bersyarat stok >= jumlah menghasilkan RowsAffected == 0.
func TestAudit_ConcurrentStockUsageRejectsOverdrawAtomically(t *testing.T) {
	t.Run("RejectWhenConditionalUpdateAffectsZeroRows", func(t *testing.T) {
		_, mock := setupMockGORM(t)
		logID, cctvID, spareID := uuid.New(), uuid.New(), uuid.New()

		mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
			WillReturnRows(maintenanceLogCompletedRow(logID, cctvID))
		mock.ExpectBegin()
		mock.ExpectQuery(`^SELECT \* FROM "spare_parts".*FOR UPDATE`).
			WillReturnRows(sparePartRows(spareID, "Kabel UTP Audit", "AUDIT-H05", "Kabel", 10, 0))
		// Simulasi kondisi di mana stok di DB sudah terkuras oleh koneksi lain sehingga WHERE stok >= 6 mempengaruhi 0 baris.
		mock.ExpectExec(`^UPDATE "spare_parts" SET "stok"=stok - \$1`).
			WithArgs(6, sqlmock.AnyArg(), spareID, 6).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectRollback()

		app := newTestApp()
		app.Post("/maintenance-logs/:id/spare-parts", withUserLocals(uuid.NewString(), "admin"), RecordSparePartUsage)

		body := `{"spare_part_id":"` + spareID.String() + `","jumlah":6}`
		req := httptest.NewRequest("POST", "/maintenance-logs/"+logID.String()+"/spare-parts", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req, -1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("TwoConcurrentRequestsTotalExceedingStock", func(t *testing.T) {
		_, mock := setupMockGORM(t)
		mock.MatchExpectationsInOrder(false)
		logID, cctvID, spareID := uuid.New(), uuid.New(), uuid.New()

		// Request pertama: stok 10 dikurangi 6 -> sisa 4 (berhasil 201).
		mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
			WillReturnRows(maintenanceLogCompletedRow(logID, cctvID))
		mock.ExpectBegin()
		mock.ExpectQuery(`^SELECT \* FROM "spare_parts".*FOR UPDATE`).
			WillReturnRows(sparePartRows(spareID, "Kabel UTP Audit", "AUDIT-H05", "Kabel", 10, 0))
		mock.ExpectExec(`^UPDATE "spare_parts" SET "stok"=stok - \$1`).
			WithArgs(6, sqlmock.AnyArg(), spareID, 6).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec(`^INSERT INTO "spare_part_usages"`).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()
		mock.ExpectQuery(`^SELECT \* FROM "spare_part_usages"`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}))

		// Request kedua: membaca sisa stok 4 (< 6) -> ditolak 400 dan di-rollback.
		mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
			WillReturnRows(maintenanceLogCompletedRow(logID, cctvID))
		mock.ExpectBegin()
		mock.ExpectQuery(`^SELECT \* FROM "spare_parts".*FOR UPDATE`).
			WillReturnRows(sparePartRows(spareID, "Kabel UTP Audit", "AUDIT-H05", "Kabel", 4, 0))
		mock.ExpectRollback()

		app := newTestApp()
		app.Post("/maintenance-logs/:id/spare-parts", withUserLocals(uuid.NewString(), "admin"), RecordSparePartUsage)

		statuses := make(chan int, 2)
		var wg sync.WaitGroup
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				body := `{"spare_part_id":"` + spareID.String() + `","jumlah":6}`
				req := httptest.NewRequest("POST", "/maintenance-logs/"+logID.String()+"/spare-parts", strings.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				resp, err := app.Test(req, -1)
				if err != nil {
					statuses <- 0
					return
				}
				statuses <- resp.StatusCode
			}()
		}
		wg.Wait()
		close(statuses)

		createdCount := 0
		badRequestCount := 0
		for status := range statuses {
			switch status {
			case fiber.StatusCreated:
				createdCount++
			case fiber.StatusBadRequest:
				badRequestCount++
			}
		}
		assert.Equal(t, 1, createdCount, "hanya satu pemakaian @6 dari stok 10 yang boleh berhasil")
		assert.Equal(t, 1, badRequestCount, "pemakaian kedua @6 yang melebihi sisa stok 4 wajib ditolak 400")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// H05: Verifikasi transaksi terkontrol pada PostgreSQL nyata (bila SIGAP_TEST_POSTGRES_DSN diset).
// Menggunakan skema sementara terisolasi yang dihapus otomatis pada t.Cleanup sehingga tidak menyentuh data operasional.
func TestAudit_PostgreSQLConcurrentStockUsageContention(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("SIGAP_TEST_POSTGRES_DSN"))
	if dsn == "" {
		t.Skip("SIGAP_TEST_POSTGRES_DSN tidak diset; lewati uji kontensi PostgreSQL langsung")
	}

	schemaName := fmt.Sprintf("h05_test_%s", strings.ReplaceAll(uuid.NewString()[:8], "-", ""))
	adminDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("gagal membuka koneksi PostgreSQL: %v", err)
	}
	if err := adminDB.Exec("CREATE SCHEMA " + schemaName).Error; err != nil {
		t.Fatalf("gagal membuat skema uji sementara: %v", err)
	}
	t.Cleanup(func() {
		_ = adminDB.Exec("DROP SCHEMA IF EXISTS " + schemaName + " CASCADE").Error
		if sqlDB, err := adminDB.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})

	scopedDSN := dsn + " search_path=" + schemaName
	pgDB, err := gorm.Open(postgres.Open(scopedDSN), &gorm.Config{})
	if err != nil {
		t.Fatalf("gagal membuka koneksi PostgreSQL dengan search_path: %v", err)
	}
	sqlDB, err := pgDB.DB()
	if err != nil {
		t.Fatalf("gagal mengambil sql.DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(5)
	t.Cleanup(func() { _ = sqlDB.Close() })

	ddl := `
		CREATE TABLE maintenance_logs (
			id uuid PRIMARY KEY,
			created_at timestamptz NOT NULL,
			updated_at timestamptz NOT NULL,
			deleted_at timestamptz NULL,
			cctv_id uuid NOT NULL,
			technician_id uuid NULL,
			action_taken text NULL,
			notes text NULL,
			status varchar(20) NOT NULL,
			scheduled_date timestamptz NULL,
			completed_at timestamptz NULL,
			image_url text NULL
		);
		CREATE TABLE spare_parts (
			id uuid PRIMARY KEY,
			created_at timestamptz NOT NULL,
			updated_at timestamptz NOT NULL,
			deleted_at timestamptz NULL,
			site_id uuid NULL,
			nama varchar(100) NOT NULL,
			sku varchar(50) NOT NULL UNIQUE,
			kategori varchar(50) NULL,
			stok integer NOT NULL DEFAULT 0 CHECK (stok >= 0),
			minimum_order integer NOT NULL DEFAULT 5,
			satuan varchar(20) NOT NULL DEFAULT 'pcs',
			harga_satuan numeric(12,2) DEFAULT 0,
			keterangan text NULL
		);
		CREATE TABLE spare_part_usages (
			id uuid PRIMARY KEY,
			created_at timestamptz NOT NULL,
			updated_at timestamptz NOT NULL,
			deleted_at timestamptz NULL,
			maintenance_log_id uuid NOT NULL REFERENCES maintenance_logs(id) ON DELETE CASCADE,
			spare_part_id uuid NOT NULL REFERENCES spare_parts(id) ON DELETE CASCADE,
			jumlah integer NOT NULL CHECK (jumlah > 0),
			catatan text NULL
		);
	`
	if err := pgDB.Exec(ddl).Error; err != nil {
		t.Fatalf("gagal membuat tabel uji di skema sementara: %v", err)
	}

	prevDB := config.DB
	config.DB = pgDB
	t.Cleanup(func() { config.DB = prevDB })

	app := newTestApp()
	app.Post("/maintenance-logs/:id/spare-parts", withUserLocals(uuid.NewString(), "admin"), RecordSparePartUsage)

	t.Run("TwoConcurrentUsesOf4From10YieldStock2AndTwoRecords", func(t *testing.T) {
		logID, cctvID, partID := uuid.New(), uuid.New(), uuid.New()
		now := time.Now()
		assert.NoError(t, pgDB.Exec(
			`INSERT INTO maintenance_logs (id, created_at, updated_at, cctv_id, status, completed_at) VALUES (?, ?, ?, ?, 'completed', ?)`,
			logID, now, now, cctvID, now,
		).Error)
		assert.NoError(t, pgDB.Exec(
			`INSERT INTO spare_parts (id, created_at, updated_at, nama, sku, kategori, stok, minimum_order, satuan) VALUES (?, ?, ?, 'Kabel UTP PG', ?, 'Kabel', 10, 2, 'pcs')`,
			partID, now, now, "SKU-"+partID.String()[:8],
		).Error)

		statuses := make(chan int, 2)
		var wg sync.WaitGroup
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				body := `{"spare_part_id":"` + partID.String() + `","jumlah":4}`
				req := httptest.NewRequest("POST", "/maintenance-logs/"+logID.String()+"/spare-parts", strings.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				resp, err := app.Test(req, -1)
				if err != nil {
					statuses <- 0
					return
				}
				statuses <- resp.StatusCode
			}()
		}
		wg.Wait()
		close(statuses)

		for st := range statuses {
			assert.Equal(t, fiber.StatusCreated, st)
		}

		var part models.SparePart
		assert.NoError(t, pgDB.First(&part, "id = ?", partID).Error)
		assert.Equal(t, 2, part.Stok)

		var usageCount int64
		assert.NoError(t, pgDB.Model(&models.SparePartUsage{}).Where("spare_part_id = ?", partID).Count(&usageCount).Error)
		assert.Equal(t, int64(2), usageCount)
	})

	t.Run("ConcurrentTotalExceedingStockRejectedAtomically", func(t *testing.T) {
		logID, cctvID, partID := uuid.New(), uuid.New(), uuid.New()
		now := time.Now()
		assert.NoError(t, pgDB.Exec(
			`INSERT INTO maintenance_logs (id, created_at, updated_at, cctv_id, status, completed_at) VALUES (?, ?, ?, ?, 'completed', ?)`,
			logID, now, now, cctvID, now,
		).Error)
		assert.NoError(t, pgDB.Exec(
			`INSERT INTO spare_parts (id, created_at, updated_at, nama, sku, kategori, stok, minimum_order, satuan) VALUES (?, ?, ?, 'Konektor RJ45 PG', ?, 'Konektor', 10, 2, 'pcs')`,
			partID, now, now, "SKU-"+partID.String()[:8],
		).Error)

		statuses := make(chan int, 2)
		var wg sync.WaitGroup
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				body := `{"spare_part_id":"` + partID.String() + `","jumlah":6}`
				req := httptest.NewRequest("POST", "/maintenance-logs/"+logID.String()+"/spare-parts", strings.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				resp, err := app.Test(req, -1)
				if err != nil {
					statuses <- 0
					return
				}
				statuses <- resp.StatusCode
			}()
		}
		wg.Wait()
		close(statuses)

		createdCount := 0
		badRequestCount := 0
		for st := range statuses {
			switch st {
			case fiber.StatusCreated:
				createdCount++
			case fiber.StatusBadRequest:
				badRequestCount++
			}
		}
		assert.Equal(t, 1, createdCount)
		assert.Equal(t, 1, badRequestCount)

		var part models.SparePart
		assert.NoError(t, pgDB.First(&part, "id = ?", partID).Error)
		assert.Equal(t, 4, part.Stok)

		var usageCount int64
		assert.NoError(t, pgDB.Model(&models.SparePartUsage{}).Where("spare_part_id = ?", partID).Count(&usageCount).Error)
		assert.Equal(t, int64(1), usageCount)
	})
}
