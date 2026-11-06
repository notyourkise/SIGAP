package handlers

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"sigap-backend/models"
	"sigap-backend/utils"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAuditActor_RecordSparePartUsage memastikan LogAudit menggunakan UserID aktor dari Locals,
// bukan logID maintenance log (Issue #183 / M10).
func TestAuditActor_RecordSparePartUsage(t *testing.T) {
	_, mock := setupMockGORM(t)
	mock.MatchExpectationsInOrder(false)

	actorID := uuid.New()
	logID := uuid.New()
	partID := uuid.New()
	cctvID := uuid.New()

	var capturedLog models.AuditLog
	cleanup := utils.SetAuditHookForTest(func(l models.AuditLog) {
		capturedLog = l
	})
	defer cleanup()

	// 1. Validasi maintenance log (status harus completed)
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(maintenanceLogCompletedRow(logID, cctvID))

	// 2. Transaksi: kurangi stok suku cadang
	mock.ExpectBegin()
	mock.ExpectQuery(`^SELECT \* FROM "spare_parts"`).
		WillReturnRows(sparePartRows(partID, "Kabel UTP Cat6", "SP-LAN-01", "Kabel", 10, 2))
	mock.ExpectExec(`^UPDATE "spare_parts" SET "stok"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`^INSERT INTO "spare_part_usages"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	// 3. Preload relasi untuk respon
	mock.ExpectQuery(`^SELECT \* FROM "spare_part_usages"`).
		WillReturnRows(sparePartUsageRows(uuid.New(), logID, partID, 2))
	mock.ExpectQuery(`^SELECT \* FROM "spare_parts"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "nama", "sku"}).AddRow(partID, "Kabel UTP Cat6", "SP-LAN-01"))

	app := newTestApp()
	app.Post("/maintenance-logs/:id/spare-parts", withUserLocals(actorID.String(), "admin"), RecordSparePartUsage)

	body, err := json.Marshal(map[string]interface{}{
		"spare_part_id": partID,
		"jumlah":        2,
	})
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/maintenance-logs/"+logID.String()+"/spare-parts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	// Verifikasi audit log
	require.NotNil(t, capturedLog.UserID, "UserID audit log tidak boleh nil")
	assert.Equal(t, actorID, *capturedLog.UserID, "UserID audit log HARUS aktor penanggung jawab (bukan logID)")
	assert.NotEqual(t, logID, *capturedLog.UserID, "UserID audit log TIDAK BOLEH objek logID")
	assert.Equal(t, "RECORD_SPARE_PART_USAGE", capturedLog.Action)
	assert.Equal(t, "INVENTORY", capturedLog.Module)
	assert.Contains(t, capturedLog.Description, logID.String(), "Deskripsi harus memuat ID maintenance log")
	assert.Contains(t, capturedLog.Description, "SP-LAN-01", "Deskripsi harus memuat SKU spare part")
}

// TestAuditActor_AssignUsersToSite memastikan LogAudit menggunakan UserID aktor dari Locals,
// bukan siteID (Issue #183 / M10).
func TestAuditActor_AssignUsersToSite(t *testing.T) {
	_, mock := setupMockGORM(t)
	mock.MatchExpectationsInOrder(false)

	actorID := uuid.New()
	siteID := uuid.New()
	techID := uuid.New()

	var capturedLog models.AuditLog
	cleanup := utils.SetAuditHookForTest(func(l models.AuditLog) {
		capturedLog = l
	})
	defer cleanup()

	// 1. Query site ada
	mock.ExpectQuery(`SELECT \* FROM "sites"`).
		WillReturnRows(siteRows(siteID, "Site Nusantara", "IKN-01", false))

	// 2. Validasi user adalah teknisi
	mock.ExpectQuery(`^SELECT count\(\*\) FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	// 3. Transaction delete + insert user_sites
	mock.ExpectBegin()
	mock.ExpectQuery(`^SELECT "user_id" FROM "user_sites"`).
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}))
	mock.ExpectExec(`^DELETE FROM "user_sites"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`^INSERT INTO "user_sites"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	app := newTestApp()
	app.Post("/sites/:id/assign-users", withUserLocals(actorID.String(), "admin"), AssignUsersToSite)

	body, err := json.Marshal(map[string]interface{}{
		"user_ids": []string{techID.String()},
	})
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/sites/"+siteID.String()+"/assign-users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	// Verifikasi audit log
	require.NotNil(t, capturedLog.UserID, "UserID audit log tidak boleh nil")
	assert.Equal(t, actorID, *capturedLog.UserID, "UserID audit log HARUS aktor admin (bukan siteID)")
	assert.NotEqual(t, siteID, *capturedLog.UserID, "UserID audit log TIDAK BOLEH objek siteID")
	assert.Equal(t, "ASSIGN_USER_SITE", capturedLog.Action)
	assert.Equal(t, "SITE", capturedLog.Module)
	assert.Contains(t, capturedLog.Description, siteID.String(), "Deskripsi harus memuat ID site")
	assert.Contains(t, capturedLog.Description, "Site Nusantara", "Deskripsi harus memuat nama site")
}

// TestAuditActor_AssignCCTVsToZone memastikan LogAudit menggunakan UserID aktor dari Locals,
// bukan zoneID (Issue #183 / M10).
func TestAuditActor_AssignCCTVsToZone(t *testing.T) {
	_, mock := setupMockGORM(t)
	mock.MatchExpectationsInOrder(false)

	actorID := uuid.New()
	zoneID := uuid.New()
	cctvID := uuid.New()

	var capturedLog models.AuditLog
	cleanup := utils.SetAuditHookForTest(func(l models.AuditLog) {
		capturedLog = l
	})
	defer cleanup()

	// 1. Query zona ada
	mock.ExpectQuery(`SELECT \* FROM "zones"`).
		WillReturnRows(zoneRows(zoneID, "Zona Presidensial", "Wilayah inti"))

	// 2. Query validasi CCTV ada
	mock.ExpectQuery(`^SELECT count\(\*\) FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	// 3. Transaksi update zone_id pada cctvs
	mock.ExpectBegin()
	mock.ExpectExec(`^UPDATE "cctvs" SET "zone_id"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	app := newTestApp()
	app.Post("/zones/:id/assign-cctvs", withUserLocals(actorID.String(), "admin"), AssignCCTVToZone)

	body, err := json.Marshal(map[string]interface{}{
		"cctv_ids": []string{cctvID.String()},
	})
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/zones/"+zoneID.String()+"/assign-cctvs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	// Verifikasi audit log
	require.NotNil(t, capturedLog.UserID, "UserID audit log tidak boleh nil")
	assert.Equal(t, actorID, *capturedLog.UserID, "UserID audit log HARUS aktor admin (bukan zoneID)")
	assert.NotEqual(t, zoneID, *capturedLog.UserID, "UserID audit log TIDAK BOLEH objek zoneID")
	assert.Equal(t, "ASSIGN_CCTV_ZONE", capturedLog.Action)
	assert.Equal(t, "ZONE", capturedLog.Module)
	assert.Contains(t, capturedLog.Description, zoneID.String(), "Deskripsi harus memuat ID zone")
	assert.Contains(t, capturedLog.Description, "Zona Presidensial", "Deskripsi harus memuat nama zona")
}

// TestAuditActor_AssignUsersToZone memastikan LogAudit menggunakan UserID aktor dari Locals,
// bukan zoneID (Issue #183 / M10).
func TestAuditActor_AssignUsersToZone(t *testing.T) {
	_, mock := setupMockGORM(t)
	mock.MatchExpectationsInOrder(false)

	actorID := uuid.New()
	zoneID := uuid.New()
	techID := uuid.New()

	var capturedLog models.AuditLog
	cleanup := utils.SetAuditHookForTest(func(l models.AuditLog) {
		capturedLog = l
	})
	defer cleanup()

	// 1. Query zona ada
	mock.ExpectQuery(`SELECT \* FROM "zones"`).
		WillReturnRows(zoneRows(zoneID, "Zona Presidensial", "Wilayah inti"))

	// 2. Query validasi teknisi
	mock.ExpectQuery(`^SELECT count\(\*\) FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	// 3. Query penugasan user_zones lama
	mock.ExpectQuery(`^SELECT \* FROM "user_zones" WHERE zone_id = \$1`).
		WithArgs(zoneID).
		WillReturnRows(sqlmock.NewRows([]string{"zone_id", "user_id"}))

	// 4. Transaksi: hapus penugasan lama + sisipkan baru
	mock.ExpectBegin()
	mock.ExpectExec(`^DELETE FROM "user_zones"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`^INSERT INTO "user_zones"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	// 5. Notifikasi teknisi baru
	mock.ExpectExec(`^INSERT INTO "notifications"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	app := newTestApp()
	app.Post("/zones/:id/assign-users", withUserLocals(actorID.String(), "admin"), AssignUsersToZone)

	body, err := json.Marshal(map[string]interface{}{
		"user_ids": []string{techID.String()},
	})
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/zones/"+zoneID.String()+"/assign-users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	// Verifikasi audit log
	require.NotNil(t, capturedLog.UserID, "UserID audit log tidak boleh nil")
	assert.Equal(t, actorID, *capturedLog.UserID, "UserID audit log HARUS aktor admin (bukan zoneID)")
	assert.NotEqual(t, zoneID, *capturedLog.UserID, "UserID audit log TIDAK BOLEH objek zoneID")
	assert.Equal(t, "ASSIGN_USER_ZONE", capturedLog.Action)
	assert.Equal(t, "ZONE", capturedLog.Module)
	assert.Contains(t, capturedLog.Description, zoneID.String(), "Deskripsi harus memuat ID zone")
	assert.Contains(t, capturedLog.Description, "Zona Presidensial", "Deskripsi harus memuat nama zona")
}
