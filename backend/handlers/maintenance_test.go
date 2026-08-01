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
	payload := `{"cctv_id":"` + uuid.New().String() + `","action_taken":"x","status":"completed"}`
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

	payload := `{"cctv_id":"` + cctvID.String() + `","action_taken":"Bersihkan lensa","status":"completed"}`
	req := httptest.NewRequest("POST", "/maintenance-logs", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestCreateMaintenanceLog_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	cctvID := uuid.New()
	techID := uuid.New()

	cctvRows, _ := cctvMockRow()
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(cctvRows)
	mock.ExpectExec(`^INSERT INTO "maintenance_logs"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Reload dengan relasi (Preload CCTV & User)
	reload := sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"cctv_id", "technician_id", "action_taken", "notes", "status",
		"scheduled_date", "completed_at", "image_url",
	}).AddRow(uuid.New(), time.Now(), time.Now(), nil, cctvID, techID, "Bersihkan lensa", "Debu", "completed", nil, nil, "")
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(reload)
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kode_cctv", "nama", "status"}))
	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "role"}))

	app := newTestApp()
	app.Post("/maintenance-logs", withUserLocals(techID.String(), "teknisi"), CreateMaintenanceLog)

	payload := `{"cctv_id":"` + cctvID.String() + `","action_taken":"Bersihkan lensa kamera","status":"completed","notes":"Debu"}`
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

	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(maintenanceMockRow())
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kode_cctv", "nama", "status"}))
	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "role"}))

	app := newTestApp()
	app.Get("/maintenance-logs/:id", GetMaintenanceLogByID)

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
