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
