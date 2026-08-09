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

// slaPolicyRows membangun baris tabel sla_policies untuk mock sqlmock.
func slaPolicyRows(id uuid.UUID, nama, jenis string, respon, selesai int) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"nama", "jenis_tugas", "target_jam_respon", "target_jam_selesai",
	}).AddRow(id, time.Now(), time.Now(), nil, nama, jenis, respon, selesai)
}

// TestCreateSLAPolicy_Success menguji pembuatan kebijakan SLA.
func TestCreateSLAPolicy_Success(t *testing.T) {
	_, mock := setupMockGORM(t)

	mock.ExpectExec(`^INSERT INTO "sla_policies"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	app := newTestApp()
	app.Post("/sla-policies", withUserLocals(uuid.New().String(), "admin"), CreateSLAPolicy)

	body := `{"nama":"SLA Korektif Prioritas Tinggi","jenis_tugas":"corrective","target_jam_respon":1,"target_jam_selesai":8}`
	req := httptest.NewRequest("POST", "/sla-policies", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
}

// TestCreateSLAPolicy_InvalidJenis memastikan jenis tugas di luar enum ditolak.
func TestCreateSLAPolicy_InvalidJenis(t *testing.T) {
	_, mock := setupMockGORM(t)

	app := newTestApp()
	app.Post("/sla-policies", withUserLocals(uuid.New().String(), "admin"), CreateSLAPolicy)

	body := `{"nama":"SLA","jenis_tugas":"invalid","target_jam_respon":1,"target_jam_selesai":8}`
	req := httptest.NewRequest("POST", "/sla-policies", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetSLAPolicies_Success menguji daftar kebijakan SLA terpaginasi.
func TestGetSLAPolicies_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	id := uuid.New()

	mock.ExpectQuery(`^SELECT count\(\*\) FROM "sla_policies"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`SELECT \* FROM "sla_policies"`).
		WillReturnRows(slaPolicyRows(id, "SLA Korektif", "corrective", 1, 8))

	app := newTestApp()
	app.Get("/sla-policies", withUserLocals(uuid.New().String(), "admin"), GetSLAPolicies)

	req := httptest.NewRequest("GET", "/sla-policies?page=1&limit=10", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetSLAPolicyByID_Success menguji detail kebijakan SLA.
func TestGetSLAPolicyByID_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	id := uuid.New()

	mock.ExpectQuery(`SELECT \* FROM "sla_policies"`).
		WillReturnRows(slaPolicyRows(id, "SLA Korektif", "corrective", 1, 8))

	app := newTestApp()
	app.Get("/sla-policies/:id", withUserLocals(uuid.New().String(), "admin"), GetSLAPolicyByID)

	req := httptest.NewRequest("GET", "/sla-policies/"+id.String(), nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	data := result["data"].(map[string]interface{})
	assert.Equal(t, "SLA Korektif", data["nama"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetSLAPolicyByID_NotFound memastikan 404 untuk id tidak ada.
func TestGetSLAPolicyByID_NotFound(t *testing.T) {
	_, mock := setupMockGORM(t)

	mock.ExpectQuery(`SELECT \* FROM "sla_policies"`).
		WillReturnError(gorm.ErrRecordNotFound)

	app := newTestApp()
	app.Get("/sla-policies/:id", withUserLocals(uuid.New().String(), "admin"), GetSLAPolicyByID)

	req := httptest.NewRequest("GET", "/sla-policies/"+uuid.New().String(), nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestUpdateSLAPolicy_Success menguji pembaruan kebijakan SLA.
func TestUpdateSLAPolicy_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	id := uuid.New()

	mock.ExpectQuery(`SELECT \* FROM "sla_policies"`).
		WillReturnRows(slaPolicyRows(id, "SLA Korektif", "corrective", 1, 8))
	mock.ExpectExec(`^UPDATE "sla_policies" SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	app := newTestApp()
	app.Put("/sla-policies/:id", withUserLocals(uuid.New().String(), "admin"), UpdateSLAPolicy)

	body := `{"target_jam_selesai":4}`
	req := httptest.NewRequest("PUT", "/sla-policies/"+id.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
}

// TestDeleteSLAPolicy_Success menguji soft delete kebijakan SLA.
func TestDeleteSLAPolicy_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	id := uuid.New()

	mock.ExpectQuery(`SELECT \* FROM "sla_policies"`).
		WillReturnRows(slaPolicyRows(id, "SLA Korektif", "corrective", 1, 8))
	mock.ExpectExec(`^UPDATE "sla_policies" SET "deleted_at"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	app := newTestApp()
	app.Delete("/sla-policies/:id", withUserLocals(uuid.New().String(), "admin"), DeleteSLAPolicy)

	req := httptest.NewRequest("DELETE", "/sla-policies/"+id.String(), nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetDashboardSLA_Success menguji dashboard SLA admin.
func TestGetDashboardSLA_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	logID, cctvID, techID := uuid.New(), uuid.New(), uuid.New()

	// Query agregat SLA (JOIN maintenance_logs → maintenance_schedules → sla_policies).
	rows := sqlmock.NewRows([]string{
		"id", "cctv_id", "cctv_name", "cctv_code", "technician_id", "technician_name",
		"action_taken", "status", "created_at", "completed_at", "target_jam_selesai", "jenis_tugas",
	}).AddRow(
		logID.String(), cctvID.String(), "CCTV 1", "CCTV-001", techID.String(), "Teknisi A",
		"Perbaikan lensa", "completed", time.Now().Add(-2*time.Hour), time.Now(), 8, "corrective",
	)
	mock.ExpectQuery(`SELECT ml\.id, ml\.cctv_id.*FROM maintenance_logs AS ml`).
		WillReturnRows(rows)

	app := newTestApp()
	app.Get("/dashboard/sla", withUserLocals(uuid.New().String(), "admin"), GetDashboardSLA)

	req := httptest.NewRequest("GET", "/dashboard/sla", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	data := result["data"].(map[string]interface{})
	assert.NotNil(t, data["monthly_compliance"])
	assert.NotNil(t, data["violations"])
	assert.NotNil(t, data["technician_ranking"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetDashboardSLA_Forbidden memastikan teknisi ditolak (403).
func TestGetDashboardSLA_Forbidden(t *testing.T) {
	_, mock := setupMockGORM(t)

	app := newTestApp()
	app.Get("/dashboard/sla", withUserLocals(uuid.New().String(), "teknisi"), GetDashboardSLA)

	req := httptest.NewRequest("GET", "/dashboard/sla", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCheckSLADeadlines_Success menguji goroutine pemeriksa deadline SLA tanpa
// panic ketika tidak ada tugas.
func TestCheckSLADeadlines_Success(t *testing.T) {
	db, mock := setupMockGORM(t)

	mock.ExpectQuery(`SELECT ml\.id, ml\.cctv_id.*FROM maintenance_logs AS ml`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "cctv_id", "cctv_name", "action_taken", "technician_id",
			"created_at", "target_jam_selesai", "jenis_tugas",
		}))

	// Jalankan tanpa panic; tidak ada tugas → tidak ada broadcast.
	CheckSLADeadlines(db)
	assert.NoError(t, mock.ExpectationsWereMet())
}
