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

func maintenanceMockRow() *sqlmock.Rows {
	id := uuid.New()
	cctvID := uuid.New()
	techID := uuid.New()
	return sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"cctv_id", "technician_id", "action_taken", "notes", "status",
		"scheduled_date", "completed_at", "image_url",
	}).AddRow(id, time.Now(), time.Now(), nil, cctvID, techID, "Bersihkan lensa kamera", "Debu", "completed", nil, time.Now(), "/api/uploads/foto.jpg")
}

func TestCreateMaintenanceLog_ValidationError(t *testing.T) {
	app := newTestApp()
	app.Post("/maintenance-logs", withUserLocals(uuid.New().String(), "teknisi"), CreateMaintenanceLog)

	// ActionTaken kurang dari 5 karakter
	payload := `{"cctv_id":"` + uuid.New().String() + `","action_taken":"x","status":"scheduled"}`
	req := httptest.NewRequest("POST", "/maintenance-logs", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestCreateMaintenanceLog_StatusForbidden(t *testing.T) {
	app := newTestApp()
	app.Post("/maintenance-logs", withUserLocals(uuid.New().String(), "teknisi"), CreateMaintenanceLog)

	// Status selain "scheduled" tidak diizinkan — alur status dikelola sistem
	payload := `{"cctv_id":"` + uuid.New().String() + `","action_taken":"Bersihkan lensa kamera","status":"completed"}`
	req := httptest.NewRequest("POST", "/maintenance-logs", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestCreateMaintenanceLog_CCTVNotFound(t *testing.T) {
	_, mock := setupMockGORM(t)
	cctvID := uuid.New()

	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnError(gorm.ErrRecordNotFound)

	app := newTestApp()
	app.Post("/maintenance-logs", withUserLocals(uuid.New().String(), "teknisi"), CreateMaintenanceLog)

	payload := `{"cctv_id":"` + cctvID.String() + `","action_taken":"Bersihkan lensa","status":"scheduled"}`
	req := httptest.NewRequest("POST", "/maintenance-logs", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestCreateMaintenanceLog_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	cctvID := uuid.New()

	cctvRows, _ := cctvMockRow()
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(cctvRows)
	mock.ExpectExec(`^INSERT INTO "maintenance_logs"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Reload dengan relasi (Preload CCTV & User) — log baru tanpa teknisi (nil),
	// sehingga GORM melewati query preload User (FK nil).
	reload := sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"cctv_id", "technician_id", "action_taken", "notes", "status",
		"scheduled_date", "completed_at", "image_url",
	}).AddRow(uuid.New(), time.Now(), time.Now(), nil, cctvID, nil, "Bersihkan lensa", "Debu", "scheduled", nil, nil, "")
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(reload)
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kode_cctv", "nama", "status"}))

	app := newTestApp()
	app.Post("/maintenance-logs", withUserLocals(uuid.New().String(), "teknisi"), CreateMaintenanceLog)

	payload := `{"cctv_id":"` + cctvID.String() + `","action_taken":"Bersihkan lensa kamera","status":"scheduled","notes":"Debu"}`
	req := httptest.NewRequest("POST", "/maintenance-logs", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMaintenanceLogByID_InvalidUUID(t *testing.T) {
	app := newTestApp()
	app.Get("/maintenance-logs/:id", GetMaintenanceLogByID)

	resp, err := app.Test(httptest.NewRequest("GET", "/maintenance-logs/not-a-uuid", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestGetMaintenanceLogByID_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	techID := uuid.New()

	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(maintenanceMockRow())
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kode_cctv", "nama", "status"}))
	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "role"}))

	app := newTestApp()
	app.Get("/maintenance-logs/:id", withUserLocals(techID.String(), "admin"), GetMaintenanceLogByID)

	resp, err := app.Test(httptest.NewRequest("GET", "/maintenance-logs/"+uuid.New().String(), nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	data := result["data"].(map[string]interface{})
	assert.Equal(t, "Bersihkan lensa kamera", data["action_taken"])
	assert.Equal(t, "completed", data["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMaintenanceLogByID_OwnershipDenied(t *testing.T) {
	_, mock := setupMockGORM(t)

	// Teknisi A mencoba melihat log milik teknisi lain (technician_id berbeda)
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(maintenanceMockRow())
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kode_cctv", "nama", "status"}))
	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "role"}))

	app := newTestApp()
	app.Get("/maintenance-logs/:id", withUserLocals(uuid.New().String(), "teknisi"), GetMaintenanceLogByID)

	resp, err := app.Test(httptest.NewRequest("GET", "/maintenance-logs/"+uuid.New().String(), nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}

func TestGetAllMaintenanceLogs_Success(t *testing.T) {
	_, mock := setupMockGORM(t)

	mock.ExpectQuery(`^SELECT count\(\*\) FROM "maintenance_logs"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(maintenanceMockRow())
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kode_cctv", "nama", "status"}))
	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "role"}))

	app := newTestApp()
	app.Get("/maintenance-logs", GetAllMaintenanceLogs)

	resp, err := app.Test(httptest.NewRequest("GET", "/maintenance-logs", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	data := result["data"].(map[string]interface{})
	pagination := data["pagination"].(map[string]interface{})
	assert.Equal(t, float64(1), pagination["total_rows"])
	rows := data["rows"].([]interface{})
	assert.Len(t, rows, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateMaintenanceLog_OwnershipDenied(t *testing.T) {
	_, mock := setupMockGORM(t)

	// Log milik teknisi lain
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(maintenanceMockRow())

	app := newTestApp()
	app.Put("/maintenance-logs/:id", withUserLocals(uuid.New().String(), "teknisi"), UpdateMaintenanceLog)

	req := httptest.NewRequest("PUT", "/maintenance-logs/"+uuid.New().String(), strings.NewReader(`{"notes":"ubah"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "error", result["status"])
}

func TestUpdateMaintenanceLog_InvalidUUID(t *testing.T) {
	app := newTestApp()
	app.Put("/maintenance-logs/:id", UpdateMaintenanceLog)

	resp, err := app.Test(httptest.NewRequest("PUT", "/maintenance-logs/not-a-uuid", strings.NewReader(`{"notes":"ubah"}`)))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestDeleteMaintenanceLog_InvalidUUID(t *testing.T) {
	app := newTestApp()
	app.Delete("/maintenance-logs/:id", DeleteMaintenanceLog)

	resp, err := app.Test(httptest.NewRequest("DELETE", "/maintenance-logs/not-a-uuid", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestUpdateMaintenanceLog_NotFound(t *testing.T) {
	_, mock := setupMockGORM(t)
	id := uuid.New()
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).WillReturnError(gorm.ErrRecordNotFound)

	app := newTestApp()
	app.Put("/maintenance-logs/:id", UpdateMaintenanceLog)
	resp, err := app.Test(httptest.NewRequest("PUT", "/maintenance-logs/"+id.String(), strings.NewReader(`{"notes":"ubah"}`)))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteMaintenanceLog_NotFound(t *testing.T) {
	_, mock := setupMockGORM(t)
	id := uuid.New()
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).WillReturnError(gorm.ErrRecordNotFound)

	app := newTestApp()
	app.Delete("/maintenance-logs/:id", DeleteMaintenanceLog)
	resp, err := app.Test(httptest.NewRequest("DELETE", "/maintenance-logs/"+id.String(), nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func knownMaintenanceRows(id, technicianID, cctvID uuid.UUID) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"cctv_id", "technician_id", "action_taken", "notes", "status",
		"scheduled_date", "completed_at", "image_url",
	}).AddRow(id, time.Now(), time.Now(), nil, cctvID, technicianID, "Bersihkan lensa kamera", "Debu", "ongoing", nil, nil, "")
}

func TestUpdateMaintenanceLog_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	id, technicianID, cctvID := uuid.New(), uuid.New(), uuid.New()
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).WillReturnRows(knownMaintenanceRows(id, technicianID, cctvID))
	mock.ExpectExec(`^UPDATE "maintenance_logs" SET`).WillReturnResult(sqlmock.NewResult(1, 1))

	app := newTestApp()
	app.Put("/maintenance-logs/:id", withUserLocals(technicianID.String(), "teknisi"), UpdateMaintenanceLog)
	req := httptest.NewRequest("PUT", "/maintenance-logs/"+id.String(), strings.NewReader(`{"notes":"Selesai dibersihkan","status":"completed"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteMaintenanceLog_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	id, technicianID, cctvID := uuid.New(), uuid.New(), uuid.New()
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).WillReturnRows(knownMaintenanceRows(id, technicianID, cctvID))
	mock.ExpectExec(`^UPDATE "maintenance_logs" SET "deleted_at"`).WillReturnResult(sqlmock.NewResult(1, 1))

	app := newTestApp()
	app.Delete("/maintenance-logs/:id", DeleteMaintenanceLog)
	resp, err := app.Test(httptest.NewRequest("DELETE", "/maintenance-logs/"+id.String(), nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// scheduledMaintenanceRows menghasilkan baris log berstatus "scheduled" (belum ditugaskan).
func scheduledMaintenanceRows(id, cctvID uuid.UUID) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"cctv_id", "technician_id", "action_taken", "notes", "status",
		"scheduled_date", "completed_at", "image_url",
	}).AddRow(id, time.Now(), time.Now(), nil, cctvID, nil, "Bersihkan lensa kamera", "Debu", "scheduled", nil, nil, "")
}

// userRow menghasilkan baris user (untuk cek peran teknisi saat assign).
func userRow(id uuid.UUID, username, role string) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "username", "password_hash", "full_name", "email", "role", "token_version", "must_change_password"}).
		AddRow(id, time.Now(), time.Now(), nil, username, "hash", "Teknisi Satu", "tek@sigap.local", role, 1, false)
}

func TestAssignMaintenanceLog_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	id, cctvID, techID := uuid.New(), uuid.New(), uuid.New()

	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(scheduledMaintenanceRows(id, cctvID))
	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnRows(userRow(techID, "teknisi1", "teknisi"))
	mock.ExpectExec(`^UPDATE "maintenance_logs" SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Reload dengan relasi (Preload CCTV & User)
	reload := sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"cctv_id", "technician_id", "action_taken", "notes", "status",
		"scheduled_date", "completed_at", "image_url",
	}).AddRow(id, time.Now(), time.Now(), nil, cctvID, techID, "Bersihkan lensa kamera", "Debu", "ongoing", nil, nil, "")
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(reload)
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kode_cctv", "nama", "status"}))
	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "role"}))

	app := newTestApp()
	app.Put("/maintenance-logs/:id/assign", withUserLocals(uuid.New().String(), "admin"), AssignMaintenanceLog)

	req := httptest.NewRequest("PUT", "/maintenance-logs/"+id.String()+"/assign", strings.NewReader(`{"technician_id":"`+techID.String()+`"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	data := result["data"].(map[string]interface{})
	assert.Equal(t, "ongoing", data["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAssignMaintenanceLog_NotScheduled(t *testing.T) {
	_, mock := setupMockGORM(t)
	id, techID, cctvID := uuid.New(), uuid.New(), uuid.New()

	// Log sudah berstatus ongoing — tidak boleh di-assign ulang
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(knownMaintenanceRows(id, techID, cctvID))

	app := newTestApp()
	app.Put("/maintenance-logs/:id/assign", withUserLocals(uuid.New().String(), "admin"), AssignMaintenanceLog)

	req := httptest.NewRequest("PUT", "/maintenance-logs/"+id.String()+"/assign", strings.NewReader(`{"technician_id":"`+techID.String()+`"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestAssignMaintenanceLog_NotTechnician(t *testing.T) {
	_, mock := setupMockGORM(t)
	id, cctvID, adminID := uuid.New(), uuid.New(), uuid.New()

	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(scheduledMaintenanceRows(id, cctvID))
	// User tujuan berperan admin — bukan teknisi
	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnRows(userRow(adminID, "admin1", "admin"))

	app := newTestApp()
	app.Put("/maintenance-logs/:id/assign", withUserLocals(uuid.New().String(), "admin"), AssignMaintenanceLog)

	req := httptest.NewRequest("PUT", "/maintenance-logs/"+id.String()+"/assign", strings.NewReader(`{"technician_id":"`+adminID.String()+`"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestAssignMaintenanceLog_InvalidUUID(t *testing.T) {
	app := newTestApp()
	app.Put("/maintenance-logs/:id/assign", withUserLocals(uuid.New().String(), "admin"), AssignMaintenanceLog)

	req := httptest.NewRequest("PUT", "/maintenance-logs/not-a-uuid/assign", strings.NewReader(`{"technician_id":"`+uuid.New().String()+`"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

// completeChecks membangun payload checklist 9 item; "allOK=false" membuat salah satu item not_ok.
func completeChecks(allOK bool) string {
	items := []string{
		"Jaringan internet kamera", "Pasokan listrik", "Kebersihan lensa", "Kondisi fisik kamera",
		"Kualitas gambar", "Fungsi perekaman", "Akses jarak jauh", "Kabel & konektor", "Lingkungan sekitar",
	}
	checks := make([]string, 0, 9)
	for i, name := range items {
		result := "ok"
		if !allOK && i == 4 {
			result = "not_ok"
		}
		checks = append(checks, `{"name":"`+name+`","result":"`+result+`","note":""}`)
	}
	return `"checks":[` + strings.Join(checks, ",") + `]`
}

func TestCompleteMaintenanceLog_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	id, techID, cctvID := uuid.New(), uuid.New(), uuid.New()

	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(knownMaintenanceRows(id, techID, cctvID))
	mock.ExpectBegin()
	mock.ExpectExec(`^INSERT INTO "field_maintenances"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	for i := 0; i < 9; i++ {
		mock.ExpectExec(`^INSERT INTO "field_maintenance_checks"`).
			WillReturnResult(sqlmock.NewResult(1, 1))
	}
	for i := 0; i < 2; i++ {
		mock.ExpectExec(`^INSERT INTO "field_maintenance_images"`).
			WillReturnResult(sqlmock.NewResult(1, 1))
	}
	mock.ExpectExec(`^UPDATE "maintenance_logs" SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Fase 5.2: sinkronisasi status otomatis — CCTV berstatus "pemeliharaan"
	// dan hasil 9/9 ok mengembalikannya menjadi "aktif".
	syncRows, _ := cctvMockRowWithStatus("pemeliharaan")
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(syncRows)
	mock.ExpectExec(`^UPDATE "cctvs" SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	// Reload dengan relasi
	reload := sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"cctv_id", "technician_id", "action_taken", "notes", "status",
		"scheduled_date", "completed_at", "image_url",
	}).AddRow(id, time.Now(), time.Now(), nil, cctvID, techID, "Bersihkan lensa kamera", "Debu", "completed", nil, time.Now(), "")
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(reload)
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kode_cctv", "nama", "status"}))
	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "role"}))

	app := newTestApp()
	app.Post("/maintenance-logs/:id/complete", withUserLocals(techID.String(), "teknisi"), CompleteMaintenanceLog)

	payload := `{` + completeChecks(true) + `,"images":[{"image_url":"/api/uploads/a.jpg"},{"image_url":"/api/uploads/b.jpg"}],"notes":"Semua normal"}`
	req := httptest.NewRequest("POST", "/maintenance-logs/"+id.String()+"/complete", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	data := result["data"].(map[string]interface{})
	assert.Equal(t, "completed", data["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCompleteMaintenanceLog_SyncCCTV_RusakUntouched(t *testing.T) {
	_, mock := setupMockGORM(t)
	id, techID, cctvID := uuid.New(), uuid.New(), uuid.New()

	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(knownMaintenanceRows(id, techID, cctvID))
	mock.ExpectBegin()
	mock.ExpectExec(`^INSERT INTO "field_maintenances"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	for i := 0; i < 9; i++ {
		mock.ExpectExec(`^INSERT INTO "field_maintenance_checks"`).
			WillReturnResult(sqlmock.NewResult(1, 1))
	}
	for i := 0; i < 2; i++ {
		mock.ExpectExec(`^INSERT INTO "field_maintenance_images"`).
			WillReturnResult(sqlmock.NewResult(1, 1))
	}
	mock.ExpectExec(`^UPDATE "maintenance_logs" SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Fase 5.2: CCTV berstatus "rusak" — sinkronisasi otomatis dipaksa skip
	// (tidak ada aksi update status), hanya membaca status saat ini.
	rusakRows, _ := cctvMockRowWithStatus("rusak")
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(rusakRows)
	mock.ExpectCommit()

	reload := sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"cctv_id", "technician_id", "action_taken", "notes", "status",
		"scheduled_date", "completed_at", "image_url",
	}).AddRow(id, time.Now(), time.Now(), nil, cctvID, techID, "Bersihkan lensa kamera", "Debu", "completed", nil, time.Now(), "")
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(reload)
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kode_cctv", "nama", "status"}))
	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "role"}))

	app := newTestApp()
	app.Post("/maintenance-logs/:id/complete", withUserLocals(techID.String(), "teknisi"), CompleteMaintenanceLog)

	payload := `{` + completeChecks(true) + `,"images":[{"image_url":"/api/uploads/a.jpg"},{"image_url":"/api/uploads/b.jpg"}],"notes":"Semua normal"}`
	req := httptest.NewRequest("POST", "/maintenance-logs/"+id.String()+"/complete", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	data := result["data"].(map[string]interface{})
	assert.Equal(t, "completed", data["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCompleteMaintenanceLog_SyncCCTV_NonaktifUntouched(t *testing.T) {
	_, mock := setupMockGORM(t)
	id, techID, cctvID := uuid.New(), uuid.New(), uuid.New()

	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(knownMaintenanceRows(id, techID, cctvID))
	mock.ExpectBegin()
	mock.ExpectExec(`^INSERT INTO "field_maintenances"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	for i := 0; i < 9; i++ {
		mock.ExpectExec(`^INSERT INTO "field_maintenance_checks"`).
			WillReturnResult(sqlmock.NewResult(1, 1))
	}
	mock.ExpectExec(`^INSERT INTO "field_maintenance_images"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`^UPDATE "maintenance_logs" SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Fase 5.2: CCTV berstatus "nonaktif" dengan hasil tidak semua ok →
	// sinkronisasi otomatis dipaksa skip (status manual admin tidak diubah).
	nonaktifRows, _ := cctvMockRowWithStatus("nonaktif")
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(nonaktifRows)
	mock.ExpectCommit()

	reload := sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"cctv_id", "technician_id", "action_taken", "notes", "status",
		"scheduled_date", "completed_at", "image_url",
	}).AddRow(id, time.Now(), time.Now(), nil, cctvID, techID, "Bersihkan lensa kamera", "Debu", "failed", nil, time.Now(), "")
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(reload)
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kode_cctv", "nama", "status"}))
	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "role"}))

	app := newTestApp()
	app.Post("/maintenance-logs/:id/complete", withUserLocals(techID.String(), "teknisi"), CompleteMaintenanceLog)

	payload := `{` + completeChecks(false) + `,"images":[{"image_url":"/api/uploads/a.jpg"}],"notes":"Kualitas gambar buram"}`
	req := httptest.NewRequest("POST", "/maintenance-logs/"+id.String()+"/complete", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	data := result["data"].(map[string]interface{})
	assert.Equal(t, "failed", data["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCompleteMaintenanceLog_NotAllOk(t *testing.T) {
	_, mock := setupMockGORM(t)
	id, techID, cctvID := uuid.New(), uuid.New(), uuid.New()

	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(knownMaintenanceRows(id, techID, cctvID))
	mock.ExpectBegin()
	mock.ExpectExec(`^INSERT INTO "field_maintenances"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	for i := 0; i < 9; i++ {
		mock.ExpectExec(`^INSERT INTO "field_maintenance_checks"`).
			WillReturnResult(sqlmock.NewResult(1, 1))
	}
	mock.ExpectExec(`^INSERT INTO "field_maintenance_images"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`^UPDATE "maintenance_logs" SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Fase 5.2: sinkronisasi status otomatis — CCTV berstatus "aktif" dengan
	// hasil ada item tidak ok dipindahkan menjadi "pemeliharaan".
	aktifRows, _ := cctvMockRow()
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(aktifRows)
	mock.ExpectExec(`^UPDATE "cctvs" SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	reload := sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"cctv_id", "technician_id", "action_taken", "notes", "status",
		"scheduled_date", "completed_at", "image_url",
	}).AddRow(id, time.Now(), time.Now(), nil, cctvID, techID, "Bersihkan lensa kamera", "Debu", "failed", nil, time.Now(), "")
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(reload)
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kode_cctv", "nama", "status"}))
	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "role"}))

	app := newTestApp()
	app.Post("/maintenance-logs/:id/complete", withUserLocals(techID.String(), "teknisi"), CompleteMaintenanceLog)

	payload := `{` + completeChecks(false) + `,"images":[{"image_url":"/api/uploads/a.jpg"}],"notes":"Kualitas gambar buram"}`
	req := httptest.NewRequest("POST", "/maintenance-logs/"+id.String()+"/complete", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	data := result["data"].(map[string]interface{})
	assert.Equal(t, "failed", data["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCompleteMaintenanceLog_OwnershipDenied(t *testing.T) {
	_, mock := setupMockGORM(t)
	id, techID, cctvID := uuid.New(), uuid.New(), uuid.New()

	// Teknisi lain mencoba menyelesaikan tugas milik teknisi lain
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(knownMaintenanceRows(id, techID, cctvID))

	app := newTestApp()
	app.Post("/maintenance-logs/:id/complete", withUserLocals(uuid.New().String(), "teknisi"), CompleteMaintenanceLog)

	payload := `{` + completeChecks(true) + `,"images":[{"image_url":"/api/uploads/a.jpg"}]}`
	req := httptest.NewRequest("POST", "/maintenance-logs/"+id.String()+"/complete", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}

func TestCompleteMaintenanceLog_NotOngoing(t *testing.T) {
	_, mock := setupMockGORM(t)
	id, cctvID := uuid.New(), uuid.New()

	// Log berstatus scheduled — belum ditugaskan, tidak bisa diselesaikan
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(scheduledMaintenanceRows(id, cctvID))

	app := newTestApp()
	app.Post("/maintenance-logs/:id/complete", withUserLocals(uuid.New().String(), "admin"), CompleteMaintenanceLog)

	payload := `{` + completeChecks(true) + `,"images":[{"image_url":"/api/uploads/a.jpg"}]}`
	req := httptest.NewRequest("POST", "/maintenance-logs/"+id.String()+"/complete", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}
