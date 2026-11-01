package handlers

import (
	"encoding/json"
	"net"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v2"
	gws "github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"gorm.io/gorm"

	"sigap-backend/middleware"
	"sigap-backend/models"
	"sigap-backend/utils"
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

// TestGetAllMaintenanceLogs_SiteScoped memastikan filter kawasan mengikuti
// relasi maintenance_logs.cctv_id -> cctvs.site_id.
func TestGetAllMaintenanceLogs_SiteScoped(t *testing.T) {
	_, mock := setupMockGORM(t)
	siteID := uuid.New()

	mock.ExpectQuery(`SELECT count\(\*\) FROM "maintenance_logs" WHERE \(EXISTS \(SELECT 1 FROM cctvs WHERE cctvs\.id = maintenance_logs\.cctv_id AND cctvs\.site_id = \$1 AND cctvs\.deleted_at IS NULL\)\)`).
		WithArgs(siteID.String()).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`SELECT \* FROM "maintenance_logs" WHERE \(EXISTS \(SELECT 1 FROM cctvs WHERE cctvs\.id = maintenance_logs\.cctv_id AND cctvs\.site_id = \$1 AND cctvs\.deleted_at IS NULL\)\)`).
		WithArgs(siteID.String(), 10).
		WillReturnRows(maintenanceMockRow())
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kode_cctv", "nama", "status"}))
	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "role"}))

	app := newTestApp()
	app.Get("/maintenance-logs", func(c *fiber.Ctx) error {
		c.Locals("site_id", siteID.String())
		return GetAllMaintenanceLogs(c)
	})

	resp, err := app.Test(httptest.NewRequest("GET", "/maintenance-logs?page=1&limit=10", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
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

	// Nama CCTV untuk pesan notifikasi ke teknisi yang ditugaskan
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kode_cctv", "nama", "status"}).
			AddRow(cctvID, "CCTV-01", "Kamera Gerbang", "aktif"))

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

// TestAssignMaintenanceLog_CompletedRejected memastikan log berstatus
// "completed" (atau "failed") tidak dapat dialihkan — hanya "scheduled" dan
// "ongoing" yang menerima penugasan (Fase 5.3).
func TestAssignMaintenanceLog_CompletedRejected(t *testing.T) {
	_, mock := setupMockGORM(t)
	id, techID := uuid.New(), uuid.New()

	// Log sudah berstatus completed — tidak boleh di-assign/dialihkan
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(maintenanceMockRow())

	app := newTestApp()
	app.Put("/maintenance-logs/:id/assign", withUserLocals(uuid.New().String(), "admin"), AssignMaintenanceLog)

	req := httptest.NewRequest("PUT", "/maintenance-logs/"+id.String()+"/assign", strings.NewReader(`{"technician_id":"`+techID.String()+`"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestAssignMaintenanceLog_ReassignSuccess — re-assign log berstatus "ongoing"
// ke teknisi lain: teknisi lama dilepas, teknisi baru ditugaskan, status tetap
// "ongoing" (Fase 5.3).
func TestAssignMaintenanceLog_ReassignSuccess(t *testing.T) {
	_, mock := setupMockGORM(t)
	id, cctvID, oldTechID, newTechID := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	// Log berstatus ongoing dengan teknisi lama
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(knownMaintenanceRows(id, oldTechID, cctvID))
	// Teknisi baru (user tujuan) valid berperan teknisi
	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnRows(userRow(newTechID, "teknisi2", "teknisi"))
	// Teknisi lama dimuat untuk audit & notifikasi "dialihkan"
	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnRows(userRow(oldTechID, "teknisi1", "teknisi"))
	mock.ExpectExec(`^UPDATE "maintenance_logs" SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Nama CCTV untuk pesan notifikasi re-assign
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kode_cctv", "nama", "status"}).
			AddRow(cctvID, "CCTV-01", "Kamera Gerbang", "aktif"))

	// Reload dengan relasi — teknisi kini teknisi baru, status tetap ongoing
	reload := sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"cctv_id", "technician_id", "action_taken", "notes", "status",
		"scheduled_date", "completed_at", "image_url",
	}).AddRow(id, time.Now(), time.Now(), nil, cctvID, newTechID, "Bersihkan lensa kamera", "Debu", "ongoing", nil, nil, "")
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(reload)
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kode_cctv", "nama", "status"}))
	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "role"}))

	app := newTestApp()
	app.Put("/maintenance-logs/:id/assign", withUserLocals(uuid.New().String(), "admin"), AssignMaintenanceLog)

	req := httptest.NewRequest("PUT", "/maintenance-logs/"+id.String()+"/assign", strings.NewReader(`{"technician_id":"`+newTechID.String()+`"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	data := result["data"].(map[string]interface{})
	assert.Equal(t, "ongoing", data["status"])
	assert.Equal(t, newTechID.String(), data["technician_id"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestAssignMaintenanceLog_ReassignSameTechnician — re-assign ke teknisi yang
// sama dengan yang sedang ditugaskan ditolak dengan 400 (Fase 5.3).
func TestAssignMaintenanceLog_ReassignSameTechnician(t *testing.T) {
	_, mock := setupMockGORM(t)
	id, cctvID, techID := uuid.New(), uuid.New(), uuid.New()

	// Log berstatus ongoing dengan teknisi lama = user tujuan
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(knownMaintenanceRows(id, techID, cctvID))
	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnRows(userRow(techID, "teknisi1", "teknisi"))

	app := newTestApp()
	app.Put("/maintenance-logs/:id/assign", withUserLocals(uuid.New().String(), "admin"), AssignMaintenanceLog)

	req := httptest.NewRequest("PUT", "/maintenance-logs/"+id.String()+"/assign", strings.NewReader(`{"technician_id":"`+techID.String()+`"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "Teknisi sudah ditugaskan pada tugas ini", result["message"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestAssignMaintenanceLog_ReassignRoleForbidden — re-assign hanya boleh
// dilakukan admin; user berperan teknisi ditolak middleware RequireRole (403).
func TestAssignMaintenanceLog_ReassignRoleForbidden(t *testing.T) {
	app := newTestApp()
	app.Put("/maintenance-logs/:id/assign",
		withUserLocals(uuid.New().String(), "teknisi"),
		middleware.RequireRole("admin"),
		AssignMaintenanceLog,
	)

	req := httptest.NewRequest("PUT", "/maintenance-logs/"+uuid.New().String()+"/assign", strings.NewReader(`{"technician_id":"`+uuid.New().String()+`"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "error", result["status"])
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

// completeChecks membangun payload checklist 9 item resmi; "allOK=false" membuat salah satu item not_ok.
func completeChecks(allOK bool) string {
	items := []string{
		"Pengecekan jaringan", "Pengecekan pasokan listrik", "Pembersihan lensa", "Pemeriksaan fisik kamera",
		"Pengecekan kualitas gambar", "Pengujian perekaman", "Pengecekan akses jarak jauh", "Pemeriksaan kabel & konektor", "Pengecekan lingkungan sekitar",
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
	expectHealthScoreRecalculation(mock)

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
	expectHealthScoreRecalculation(mock)

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
	expectHealthScoreRecalculation(mock)

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

// TestCompleteMaintenanceLog_SyncCCTV_LegacyUppercaseUntouched memastikan
// baris legacy berstatus huruf besar Inggris ('INACTIVE', 'RUSAK', 'BROKEN')
// tetap dilindungi oleh aturan konservatif — sinkronisasi otomatis tidak boleh
// menimpanya (regresi yang ditemukan saat uji manual 2026-08-05: 'INACTIVE'
// sempat diubah otomatis menjadi 'aktif').
func TestCompleteMaintenanceLog_SyncCCTV_LegacyUppercaseUntouched(t *testing.T) {
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

	// CCTV legacy berstatus "INACTIVE" (huruf besar Inggris) — hasil 9/9 ok
	// tetap tidak boleh mengubahnya; hanya baca status, tanpa aksi update.
	legacyRows, _ := cctvMockRowWithStatus("INACTIVE")
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(legacyRows)
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
	expectHealthScoreRecalculation(mock)

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
	expectHealthScoreRecalculation(mock)

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

// H06: Memastikan broadcastMaintenanceLogEvent hanya menyiarkan payload minimal
// (tanpa notes rahasia atau rtsp_url) ke admin dan teknisi pemilik di site terkait,
// serta menolak teknisi lain dalam site yang sama maupun teknisi di site berbeda.
func TestBroadcastMaintenanceLogEvent_MinimalPayloadAndRecipientIsolation(t *testing.T) {
	app := fiber.New()
	app.Get("/ws", gws.New(func(c *gws.Conn) {
		var sites []string
		if raw := c.Query("sites"); raw != "" {
			sites = strings.Split(raw, ",")
		}
		utils.Hub.Register(c, c.Query("user"), c.Query("role"), sites)
		defer utils.Hub.Unregister(c)
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
		}
	}))

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = listener.Close() }()
	go func() { _ = fasthttp.Serve(listener, app.Handler()) }()

	dial := func(query string) *websocket.Conn {
		conn, _, err := websocket.DefaultDialer.Dial("ws://"+listener.Addr().String()+"/ws?"+query, nil)
		require.NoError(t, err)
		return conn
	}

	siteA := uuid.New()
	siteB := uuid.New()
	ownerTech := uuid.New()
	otherSiteATech := uuid.New()
	siteBTech := uuid.New()
	adminID := uuid.New()

	adminConn := dial("user=" + adminID.String() + "&role=admin")
	defer func() { _ = adminConn.Close() }()
	ownerConn := dial("user=" + ownerTech.String() + "&role=teknisi&sites=" + siteA.String())
	defer func() { _ = ownerConn.Close() }()
	otherSiteAConn := dial("user=" + otherSiteATech.String() + "&role=teknisi&sites=" + siteA.String())
	defer func() { _ = otherSiteAConn.Close() }()
	siteBConn := dial("user=" + siteBTech.String() + "&role=teknisi&sites=" + siteB.String())
	defer func() { _ = siteBConn.Close() }()

	require.Eventually(t, func() bool {
		return utils.Hub.ActiveConnections() >= 4
	}, time.Second, 10*time.Millisecond)

	logID := uuid.New()
	cctvID := uuid.New()
	logEntry := models.MaintenanceLog{
		Base:         models.Base{ID: logID},
		CCTVID:       cctvID,
		TechnicianID: &ownerTech,
		ActionTaken:  "Penggantian konektor RJ45",
		Notes:        "CATATAN RAHASIA INTERNAL SITE A",
		Status:       "ongoing",
		CCTV: models.CCTV{
			Base:    models.Base{ID: cctvID},
			SiteID:  &siteA,
			RTSPURL: "enc:v1:rahasia-rtsp",
		},
	}

	broadcastMaintenanceLogEvent(nil, "MAINTENANCE_UPDATED", logEntry)

	readRaw := func(conn *websocket.Conn) string {
		require.NoError(t, conn.SetReadDeadline(time.Now().Add(time.Second)))
		_, raw, err := conn.ReadMessage()
		require.NoError(t, err)
		return string(raw)
	}

	adminBody := readRaw(adminConn)
	ownerBody := readRaw(ownerConn)

	for _, body := range []string{adminBody, ownerBody} {
		assert.Contains(t, body, `"event":"MAINTENANCE_UPDATED"`)
		assert.Contains(t, body, logID.String())
		assert.Contains(t, body, siteA.String())
		assert.Contains(t, body, ownerTech.String())
		assert.NotContains(t, body, "CATATAN RAHASIA INTERNAL SITE A")
		assert.NotContains(t, body, `"notes"`)
		assert.NotContains(t, body, "enc:v1:rahasia-rtsp")
	}

	require.NoError(t, otherSiteAConn.SetReadDeadline(time.Now().Add(200*time.Millisecond)))
	_, _, err = otherSiteAConn.ReadMessage()
	assert.Error(t, err, "teknisi lain di site-a tidak boleh menerima event tugas teknisi pemilik")

	require.NoError(t, siteBConn.SetReadDeadline(time.Now().Add(200*time.Millisecond)))
	_, _, err = siteBConn.ReadMessage()
	assert.Error(t, err, "teknisi site-b tidak boleh menerima event tugas site-a")
}

// H07: Reproduksi eksploitasi dari laporan audit — penyelesaian tugas dengan
// item checklist karangan dan foto bukti fiktif wajib ditolak dengan 400.
func TestAudit_CompletionMustRejectInventedChecklist(t *testing.T) {
	_, mock := setupMockGORM(t)
	id, techID, cctvID := uuid.New(), uuid.New(), uuid.New()

	app := newTestApp()
	app.Post("/maintenance-logs/:id/complete", withUserLocals(techID.String(), "teknisi"), CompleteMaintenanceLog)

	// 1. Foto bukti fiktif + checklist karangan -> 400
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(knownMaintenanceRows(id, techID, cctvID))
	fakePhotoBody := `{"checks":[{"name":"This is not an official checklist item","result":"ok","note":""}],"images":[{"image_url":"/api/uploads/does-not-exist.jpg"}],"notes":"Completed without active template checks"}`
	reqFakePhoto := httptest.NewRequest("POST", "/maintenance-logs/"+id.String()+"/complete", strings.NewReader(fakePhotoBody))
	reqFakePhoto.Header.Set("Content-Type", "application/json")
	respFakePhoto, err := app.Test(reqFakePhoto)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, respFakePhoto.StatusCode)

	// 2. Foto bukti valid tetapi checklist karangan / tidak mencakup seluruh template wajib -> 400
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(knownMaintenanceRows(id, techID, cctvID))
	mock.ExpectQuery(`^SELECT \* FROM "checklist_templates"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "nama_item", "urutan", "wajib", "aktif"}))
	inventedCheckBody := `{"checks":[{"name":"This is not an official checklist item","result":"ok","note":""}],"images":[{"image_url":"/api/uploads/a.jpg"}],"notes":"Completed without active template checks"}`
	reqInvented := httptest.NewRequest("POST", "/maintenance-logs/"+id.String()+"/complete", strings.NewReader(inventedCheckBody))
	reqInvented.Header.Set("Content-Type", "application/json")
	respInvented, err := app.Test(reqInvented)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, respInvented.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// H07: Menguji validasi template dinamis (jumlah item bervariasi, kombinasi wajib=true & wajib=false),
// penolakan item wajib yang absen, penolakan duplikat item, serta pencocokan template_id.
func TestCompleteMaintenanceLog_DynamicChecklistTemplatesValidation(t *testing.T) {
	_, mock := setupMockGORM(t)
	id, techID, cctvID := uuid.New(), uuid.New(), uuid.New()
	tpl1ID, tpl2ID, tplOptID := uuid.New(), uuid.New(), uuid.New()

	templateRows := func() *sqlmock.Rows {
		return sqlmock.NewRows([]string{"id", "nama_item", "urutan", "wajib", "aktif"}).
			AddRow(tpl1ID, "Pengecekan PoE Switch", 1, true, true).
			AddRow(tpl2ID, "Pembersihan Housing Outdoor", 2, true, true).
			AddRow(tplOptID, "Pengecekan IR Illuminator Tambahan", 3, false, true)
	}

	app := newTestApp()
	app.Post("/maintenance-logs/:id/complete", withUserLocals(techID.String(), "teknisi"), CompleteMaintenanceLog)

	// Kasus 1: Hanya mengirim 1 dari 2 item wajib -> 400
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(knownMaintenanceRows(id, techID, cctvID))
	mock.ExpectQuery(`^SELECT \* FROM "checklist_templates"`).
		WillReturnRows(templateRows())

	missingMandatory := `{"checks":[{"name":"Pengecekan PoE Switch","result":"ok"}],"images":[{"image_url":"/api/uploads/a.jpg"}]}`
	reqMissing := httptest.NewRequest("POST", "/maintenance-logs/"+id.String()+"/complete", strings.NewReader(missingMandatory))
	reqMissing.Header.Set("Content-Type", "application/json")
	respMissing, err := app.Test(reqMissing)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, respMissing.StatusCode)

	// Kasus 2: Mengirim duplikat item wajib -> 400
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(knownMaintenanceRows(id, techID, cctvID))
	mock.ExpectQuery(`^SELECT \* FROM "checklist_templates"`).
		WillReturnRows(templateRows())

	duplicateItem := `{"checks":[{"name":"Pengecekan PoE Switch","result":"ok"},{"name":"Pengecekan PoE Switch","result":"ok"},{"name":"Pembersihan Housing Outdoor","result":"ok"}],"images":[{"image_url":"/api/uploads/a.jpg"}]}`
	reqDup := httptest.NewRequest("POST", "/maintenance-logs/"+id.String()+"/complete", strings.NewReader(duplicateItem))
	reqDup.Header.Set("Content-Type", "application/json")
	respDup, err := app.Test(reqDup)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, respDup.StatusCode)

	// Kasus 3: Mengirim seluruh item wajib (2 item) tanpa item opsional, menggunakan template_id -> 200 OK
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(knownMaintenanceRows(id, techID, cctvID))
	mock.ExpectQuery(`^SELECT \* FROM "checklist_templates"`).
		WillReturnRows(templateRows())
	mock.ExpectBegin()
	mock.ExpectExec(`^INSERT INTO "field_maintenances"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`^INSERT INTO "field_maintenance_checks"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`^INSERT INTO "field_maintenance_checks"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`^INSERT INTO "field_maintenance_images"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`^UPDATE "maintenance_logs" SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	syncRows, _ := cctvMockRowWithStatus("pemeliharaan")
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(syncRows)
	mock.ExpectExec(`^UPDATE "cctvs" SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	reload := sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"cctv_id", "technician_id", "action_taken", "notes", "status",
		"scheduled_date", "completed_at", "image_url",
	}).AddRow(id, time.Now(), time.Now(), nil, cctvID, techID, "Bersihkan lensa kamera", "Normal", "completed", nil, time.Now(), "")
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(reload)
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kode_cctv", "nama", "status"}))
	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "role"}))
	expectHealthScoreRecalculation(mock)

	validDynamic := `{"checks":[{"template_id":"` + tpl1ID.String() + `","name":"Pengecekan PoE Switch","result":"ok"},{"template_id":"` + tpl2ID.String() + `","name":"Pembersihan Housing Outdoor","result":"ok"}],"images":[{"image_url":"/api/uploads/a.jpg"}]}`
	reqValid := httptest.NewRequest("POST", "/maintenance-logs/"+id.String()+"/complete", strings.NewReader(validDynamic))
	reqValid.Header.Set("Content-Type", "application/json")
	respValid, err := app.Test(reqValid)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, respValid.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}
