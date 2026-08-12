package handlers

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"sigap-backend/middleware"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// checklistTemplateRows membangun baris tabel checklist_templates untuk mock
// sqlmock.
func checklistTemplateRows(id uuid.UUID, nama, deskripsi string, urutan int, wajib, aktif bool) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"nama_item", "deskripsi", "urutan", "wajib", "aktif",
	}).AddRow(id, time.Now(), time.Now(), nil, nama, deskripsi, urutan, wajib, aktif)
}

// TestCreateChecklistTemplate_Success menguji pembuatan item template checklist.
func TestCreateChecklistTemplate_Success(t *testing.T) {
	_, mock := setupMockGORM(t)

	mock.ExpectExec(`^INSERT INTO "checklist_templates"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	app := newTestApp()
	app.Post("/checklist-templates", withUserLocals(uuid.New().String(), "admin"), CreateChecklistTemplate)

	body := `{"nama_item":"Pengecekan jaringan","deskripsi":"Pastikan koneksi jaringan normal.","urutan":1}`
	req := httptest.NewRequest("POST", "/checklist-templates", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCreateChecklistTemplate_ValidationError memastikan nama_item kosong
// ditolak dengan 400.
func TestCreateChecklistTemplate_ValidationError(t *testing.T) {
	app := newTestApp()
	app.Post("/checklist-templates", withUserLocals(uuid.New().String(), "admin"), CreateChecklistTemplate)

	body := `{"nama_item":""}`
	req := httptest.NewRequest("POST", "/checklist-templates", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "error", result["status"])
}

// TestGetAllChecklistTemplates_Success menguji daftar template terpaginasi.
func TestGetAllChecklistTemplates_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	id := uuid.New()

	mock.ExpectQuery(`^SELECT count\(\*\) FROM "checklist_templates"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`^SELECT \* FROM "checklist_templates"`).
		WillReturnRows(checklistTemplateRows(id, "Pengecekan jaringan", "Cek jaringan", 1, true, true))

	app := newTestApp()
	app.Get("/checklist-templates", withUserLocals(uuid.New().String(), "admin"), GetAllChecklistTemplates)

	req := httptest.NewRequest("GET", "/checklist-templates?page=1&limit=10", nil)

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

// TestGetAllChecklistTemplates_AktifFilter menguji filter ?aktif=true.
func TestGetAllChecklistTemplates_AktifFilter(t *testing.T) {
	_, mock := setupMockGORM(t)
	id := uuid.New()

	mock.ExpectQuery(`^SELECT count\(\*\) FROM "checklist_templates" WHERE aktif = \$1`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`^SELECT \* FROM "checklist_templates" WHERE aktif = \$1`).
		WillReturnRows(checklistTemplateRows(id, "Pengecekan jaringan", "Cek jaringan", 1, true, true))

	app := newTestApp()
	app.Get("/checklist-templates", withUserLocals(uuid.New().String(), "teknisi"), GetAllChecklistTemplates)

	req := httptest.NewRequest("GET", "/checklist-templates?aktif=true", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestUpdateChecklistTemplate_Success menguji pembaruan item template checklist.
func TestUpdateChecklistTemplate_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	id := uuid.New()

	mock.ExpectQuery(`^SELECT \* FROM "checklist_templates"`).
		WillReturnRows(checklistTemplateRows(id, "Pengecekan jaringan", "Cek jaringan", 1, true, true))
	mock.ExpectExec(`^UPDATE "checklist_templates" SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	app := newTestApp()
	app.Put("/checklist-templates/:id", withUserLocals(uuid.New().String(), "admin"), UpdateChecklistTemplate)

	body := `{"nama_item":"Pengecekan jaringan","urutan":2}`
	req := httptest.NewRequest("PUT", "/checklist-templates/"+id.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestUpdateChecklistTemplate_NotFound memastikan 404 untuk id yang tidak ada.
func TestUpdateChecklistTemplate_NotFound(t *testing.T) {
	_, mock := setupMockGORM(t)

	mock.ExpectQuery(`^SELECT \* FROM "checklist_templates"`).
		WillReturnError(gorm.ErrRecordNotFound)

	app := newTestApp()
	app.Put("/checklist-templates/:id", withUserLocals(uuid.New().String(), "admin"), UpdateChecklistTemplate)

	req := httptest.NewRequest("PUT", "/checklist-templates/"+uuid.New().String(),
		strings.NewReader(`{"nama_item":"Test"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestDeleteChecklistTemplate_Success menguji soft delete (aktif = false).
func TestDeleteChecklistTemplate_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	id := uuid.New()

	mock.ExpectQuery(`^SELECT \* FROM "checklist_templates"`).
		WillReturnRows(checklistTemplateRows(id, "Pengecekan jaringan", "Cek jaringan", 1, true, true))
	mock.ExpectExec(`^UPDATE "checklist_templates" SET "aktif"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	app := newTestApp()
	app.Delete("/checklist-templates/:id", withUserLocals(uuid.New().String(), "admin"), DeleteChecklistTemplate)

	req := httptest.NewRequest("DELETE", "/checklist-templates/"+id.String(), nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestDeleteChecklistTemplate_NotFound memastikan 404 untuk id yang tidak ada.
func TestDeleteChecklistTemplate_NotFound(t *testing.T) {
	_, mock := setupMockGORM(t)

	mock.ExpectQuery(`^SELECT \* FROM "checklist_templates"`).
		WillReturnError(gorm.ErrRecordNotFound)

	app := newTestApp()
	app.Delete("/checklist-templates/:id", withUserLocals(uuid.New().String(), "admin"), DeleteChecklistTemplate)

	req := httptest.NewRequest("DELETE", "/checklist-templates/"+uuid.New().String(), nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestReorderChecklistTemplates_Success menguji batch update urutan dalam
// transaksi.
func TestReorderChecklistTemplates_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	id1, id2 := uuid.New(), uuid.New()

	mock.ExpectBegin()
	mock.ExpectExec(`^UPDATE "checklist_templates" SET "urutan"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`^UPDATE "checklist_templates" SET "urutan"`).
		WillReturnResult(sqlmock.NewResult(2, 1))
	mock.ExpectCommit()

	app := newTestApp()
	app.Put("/checklist-templates/reorder", withUserLocals(uuid.New().String(), "admin"), ReorderChecklistTemplates)

	body := `{"items":[{"id":"` + id1.String() + `","urutan":2},{"id":"` + id2.String() + `","urutan":1}]}`
	req := httptest.NewRequest("PUT", "/checklist-templates/reorder", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestReorderChecklistTemplates_ValidationError memastikan items kosong
// ditolak dengan 400.
func TestReorderChecklistTemplates_ValidationError(t *testing.T) {
	app := newTestApp()
	app.Put("/checklist-templates/reorder", withUserLocals(uuid.New().String(), "admin"), ReorderChecklistTemplates)

	body := `{"items":[]}`
	req := httptest.NewRequest("PUT", "/checklist-templates/reorder", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "error", result["status"])
}

// TestCreateChecklistTemplate_RequireRoleForbidsNonAdmin menguji guard admin:
// role non-admin yang mencoba membuat template harus ditolak 403.
func TestCreateChecklistTemplate_RequireRoleForbidsNonAdmin(t *testing.T) {
	app := newTestApp()
	app.Post("/checklist-templates",
		withUserLocals(uuid.New().String(), "teknisi"),
		middleware.RequireRole("admin"),
		CreateChecklistTemplate)

	body := `{"nama_item":"Pengecekan jaringan"}`
	req := httptest.NewRequest("POST", "/checklist-templates", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "error", result["status"])
}

// TestCreateChecklistTemplate_ProtectedUnauthorized menguji middleware
// Protected: tanpa identitas user, akses ditolak 401.
func TestCreateChecklistTemplate_ProtectedUnauthorized(t *testing.T) {
	app := newTestApp()
	app.Post("/checklist-templates", middleware.Protected(), CreateChecklistTemplate)

	req := httptest.NewRequest("POST", "/checklist-templates", strings.NewReader(`{"nama_item":"Test"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}
