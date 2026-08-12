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

// siteRows membangun baris tabel sites untuk mock sqlmock.
func siteRows(id uuid.UUID, nama, kode string, isDefault bool) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at", "nama", "kode", "alamat",
		"koordinat_lat", "koordinat_lng", "is_default",
	}).AddRow(id, time.Now(), time.Now(), nil, nama, kode, "", nil, nil, isDefault)
}

// TestCreateSite_Success menguji pembuatan site baru (LogAudit async tidak di-mock).
func TestCreateSite_Success(t *testing.T) {
	_, mock := setupMockGORM(t)

	// Tidak ada kode duplikat.
	mock.ExpectQuery(`^SELECT \* FROM "sites"`).
		WillReturnError(gorm.ErrRecordNotFound)
	mock.ExpectExec(`^INSERT INTO "sites"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	app := newTestApp()
	app.Post("/sites", withUserLocals(uuid.New().String(), "admin"), CreateSite)

	body := `{"nama":"Site Batam","kode":"BTM","alamat":"Batam"}`
	req := httptest.NewRequest("POST", "/sites", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCreateSite_DuplicateKode menguji kode site yang sudah dipakai → 400.
func TestCreateSite_DuplicateKode(t *testing.T) {
	_, mock := setupMockGORM(t)
	id := uuid.New()

	// Kode sudah dipakai oleh data yang tidak di-soft-delete → 400.
	mock.ExpectQuery(`^SELECT \* FROM "sites"`).
		WillReturnRows(siteRows(id, "Kawasan OIKN", "OIKN", true))

	app := newTestApp()
	app.Post("/sites", withUserLocals(uuid.New().String(), "admin"), CreateSite)

	body := `{"nama":"Site Lain","kode":"OIKN"}`
	req := httptest.NewRequest("POST", "/sites", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "Kode site sudah digunakan oleh site lain", result["message"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCreateSite_EmptyNama menguji site tanpa nama → 400 (tanpa query DB).
func TestCreateSite_EmptyNama(t *testing.T) {
	_, mock := setupMockGORM(t)

	app := newTestApp()
	app.Post("/sites", withUserLocals(uuid.New().String(), "admin"), CreateSite)

	req := httptest.NewRequest("POST", "/sites", strings.NewReader(`{"nama":"","kode":"XYZ"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetAllSites_Success menguji daftar site terpaginasi dengan jumlah user.
func TestGetAllSites_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	id := uuid.New()

	mock.ExpectQuery(`^SELECT count\(\*\) FROM "sites"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	// SELECT utama sites (order nama asc, limit, offset).
	mock.ExpectQuery(`SELECT \* FROM "sites"`).
		WillReturnRows(siteRows(id, "Kawasan OIKN", "OIKN", true))
	// Hitung user_sites per site.
	mock.ExpectQuery(`^SELECT count\(\*\) FROM "user_sites"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	app := newTestApp()
	app.Get("/sites", withUserLocals(uuid.New().String(), "admin"), GetAllSites)

	req := httptest.NewRequest("GET", "/sites?page=1&limit=10", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	data := result["data"].(map[string]interface{})
	rows := data["rows"].([]interface{})
	assert.Equal(t, float64(2), rows[0].(map[string]interface{})["user_count"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestUpdateSite_Success menguji pembaruan site.
func TestUpdateSite_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	id := uuid.New()

	mock.ExpectQuery(`SELECT \* FROM "sites"`).
		WillReturnRows(siteRows(id, "Kawasan OIKN", "OIKN", true))
	mock.ExpectExec(`^UPDATE "sites" SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	app := newTestApp()
	app.Put("/sites/:id", withUserLocals(uuid.New().String(), "admin"), UpdateSite)

	body := `{"nama":"Kawasan OIKN Baru"}`
	req := httptest.NewRequest("PUT", "/sites/"+id.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestDeleteSite_Success menguji soft delete site (bukan default).
func TestDeleteSite_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	id := uuid.New()

	mock.ExpectQuery(`SELECT \* FROM "sites"`).
		WillReturnRows(siteRows(id, "Site Batam", "BTM", false))
	mock.ExpectExec(`^UPDATE "sites" SET "deleted_at"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	app := newTestApp()
	app.Delete("/sites/:id", withUserLocals(uuid.New().String(), "admin"), DeleteSite)

	req := httptest.NewRequest("DELETE", "/sites/"+id.String(), nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestDeleteSite_Default menguji penolakan hapus site default → 400.
func TestDeleteSite_Default(t *testing.T) {
	_, mock := setupMockGORM(t)
	id := uuid.New()

	mock.ExpectQuery(`SELECT \* FROM "sites"`).
		WillReturnRows(siteRows(id, "Kawasan OIKN", "OIKN", true))

	app := newTestApp()
	app.Delete("/sites/:id", withUserLocals(uuid.New().String(), "admin"), DeleteSite)

	req := httptest.NewRequest("DELETE", "/sites/"+id.String(), nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "Tidak dapat menghapus site default", result["message"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestAssignUsersToSite_Success menguji assign teknisi ke site.
func TestAssignUsersToSite_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	siteID, userID := uuid.New(), uuid.New()

	// Muat site.
	mock.ExpectQuery(`SELECT \* FROM "sites"`).
		WillReturnRows(siteRows(siteID, "Site Batam", "BTM", false))
	// Hitung jumlah user teknisi pada daftar.
	mock.ExpectQuery(`^SELECT count\(\*\) FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	// Transaksi: hapus penugasan lama + sisipkan baru.
	mock.ExpectBegin()
	mock.ExpectExec(`^DELETE FROM "user_sites"`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`^INSERT INTO "user_sites"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	app := newTestApp()
	app.Post("/sites/:id/assign-users", withUserLocals(uuid.New().String(), "admin"), AssignUsersToSite)

	body := `{"user_ids":["` + userID.String() + `"]}`
	req := httptest.NewRequest("POST", "/sites/"+siteID.String()+"/assign-users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestAssignUsersToSite_InvalidUser menguji user bukan teknisi ditolak → 400.
func TestAssignUsersToSite_InvalidUser(t *testing.T) {
	_, mock := setupMockGORM(t)
	siteID := uuid.New()

	mock.ExpectQuery(`SELECT \* FROM "sites"`).
		WillReturnRows(siteRows(siteID, "Site Batam", "BTM", false))
	mock.ExpectQuery(`^SELECT count\(\*\) FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	app := newTestApp()
	app.Post("/sites/:id/assign-users", withUserLocals(uuid.New().String(), "admin"), AssignUsersToSite)

	body := `{"user_ids":["` + uuid.New().String() + `"]}`
	req := httptest.NewRequest("POST", "/sites/"+siteID.String()+"/assign-users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetAllSitesForSelect_Success menguji daftar site tanpa paginasi.
func TestGetAllSitesForSelect_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	id := uuid.New()

	mock.ExpectQuery(`SELECT \* FROM "sites"`).
		WillReturnRows(siteRows(id, "Kawasan OIKN", "OIKN", true))

	app := newTestApp()
	app.Get("/sites/all", withUserLocals(uuid.New().String(), "teknisi"), GetAllSitesForSelect)

	req := httptest.NewRequest("GET", "/sites/all", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}
