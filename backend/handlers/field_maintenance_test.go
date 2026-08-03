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
	mock.ExpectCommit()

	// Reload dengan relasi lengkap (GORM menjalankan preload sesuai urutan alfabetis)
	reload := sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"cctv_id", "technician_id", "overall_status", "notes", "completed_at",
	}).AddRow(uuid.New(), time.Now(), time.Now(), nil, cctvID, techID, "attention", "Pengecekan rutin", time.Now())
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

func TestGetAllFieldMaintenances_TechnicianScoped(t *testing.T) {
	_, mock := setupMockGORM(t)
	techID := uuid.New()

	mock.ExpectQuery(`^SELECT count\(\*\) FROM "field_maintenances" WHERE technician_id = \$1`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	mock.ExpectQuery(`^SELECT \* FROM "field_maintenances" WHERE technician_id = \$1 AND "field_maintenances"."deleted_at" IS NULL ORDER BY created_at desc LIMIT`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "created_at", "updated_at", "deleted_at",
			"cctv_id", "technician_id", "overall_status", "notes", "completed_at",
		}).AddRow(uuid.New(), time.Now(), time.Now(), nil, uuid.New(), techID, "ok", "Bersih", time.Now()))
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
