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

// zoneRows membangun baris tabel zones untuk mock sqlmock.
func zoneRows(id uuid.UUID, nama, deskripsi string) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at", "nama", "deskripsi",
	}).AddRow(id, time.Now(), time.Now(), nil, nama, deskripsi)
}

// TestCreateZone_Success menguji pembuatan zona baru. Catatan: LogAudit berjalan
// async di goroutine sehingga query audit tidak di-mock; respons handler tetap
// diverifikasi. (Pola sama dengan test CRUD modul lain di codebase.)
func TestCreateZone_Success(t *testing.T) {
	_, mock := setupMockGORM(t)

	mock.ExpectExec(`^INSERT INTO "zones"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	app := newTestApp()
	app.Post("/zones", withUserLocals(uuid.New().String(), "admin"), CreateZone)

	body := `{"nama":"Zona Utara","deskripsi":"Wilayah utara kawasan IKN"}`
	req := httptest.NewRequest("POST", "/zones", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
}

// TestCreateZone_ValidationError memastikan zona tanpa nama ditolak (400).
func TestCreateZone_ValidationError(t *testing.T) {
	_, mock := setupMockGORM(t)

	app := newTestApp()
	app.Post("/zones", withUserLocals(uuid.New().String(), "admin"), CreateZone)

	req := httptest.NewRequest("POST", "/zones", strings.NewReader(`{"nama":""}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetAllZones_Success menguji daftar zona terpaginasi dengan jumlah CCTV/teknisi.
func TestGetAllZones_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	id := uuid.New()

	mock.ExpectQuery(`^SELECT count\(\*\) FROM "zones"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	// SELECT utama zones.
	mock.ExpectQuery(`SELECT \* FROM "zones"`).
		WillReturnRows(zoneRows(id, "Zona Utara", "Wilayah utara"))
	// Hitung CCTV per zona.
	mock.ExpectQuery(`^SELECT count\(\*\) FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	// Hitung user_zones per zona.
	mock.ExpectQuery(`^SELECT count\(\*\) FROM "user_zones"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	app := newTestApp()
	app.Get("/zones", withUserLocals(uuid.New().String(), "admin"), GetAllZones)

	req := httptest.NewRequest("GET", "/zones?page=1&limit=10", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	data := result["data"].(map[string]interface{})
	rows := data["rows"].([]interface{})
	assert.Equal(t, float64(2), rows[0].(map[string]interface{})["cctv_count"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetZoneByID_Success menguji detail zona (termasuk preload relasi).
func TestGetZoneByID_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	id := uuid.New()

	mock.ExpectQuery(`SELECT \* FROM "zones"`).
		WillReturnRows(zoneRows(id, "Zona Utara", "Wilayah utara"))
	// Preload CCTVs (one-to-many).
	mock.ExpectQuery(`SELECT .* FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "zone_id", "kode_cctv", "nama", "lokasi_area", "status"}))
	// Preload Users (many2many): ambil id user_zones lalu detail users.
	mock.ExpectQuery(`SELECT .* FROM "user_zones"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "zone_id", "user_id"}))
	mock.ExpectQuery(`SELECT .* FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "full_name", "username", "email", "role"}))

	app := newTestApp()
	app.Get("/zones/:id", withUserLocals(uuid.New().String(), "admin"), GetZoneByID)

	req := httptest.NewRequest("GET", "/zones/"+id.String(), nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
}

// TestGetZoneByID_NotFound memastikan 404 untuk zona tidak ada.
func TestGetZoneByID_NotFound(t *testing.T) {
	_, mock := setupMockGORM(t)

	mock.ExpectQuery(`SELECT .* FROM "zones"`).
		WillReturnError(gorm.ErrRecordNotFound)

	app := newTestApp()
	app.Get("/zones/:id", withUserLocals(uuid.New().String(), "admin"), GetZoneByID)

	req := httptest.NewRequest("GET", "/zones/"+uuid.New().String(), nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestUpdateZone_Success menguji pembaruan zona.
func TestUpdateZone_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	id := uuid.New()

	mock.ExpectQuery(`SELECT \* FROM "zones"`).
		WillReturnRows(zoneRows(id, "Zona Utara", "Wilayah utara"))
	mock.ExpectExec(`^UPDATE "zones" SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	app := newTestApp()
	app.Put("/zones/:id", withUserLocals(uuid.New().String(), "admin"), UpdateZone)

	body := `{"nama":"Zona Utara Baru"}`
	req := httptest.NewRequest("PUT", "/zones/"+id.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
}

// TestDeleteZone_Success menguji soft delete zona.
func TestDeleteZone_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	id := uuid.New()

	mock.ExpectQuery(`SELECT \* FROM "zones"`).
		WillReturnRows(zoneRows(id, "Zona Utara", "Wilayah utara"))
	mock.ExpectExec(`^UPDATE "zones" SET "deleted_at"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	app := newTestApp()
	app.Delete("/zones/:id", withUserLocals(uuid.New().String(), "admin"), DeleteZone)

	req := httptest.NewRequest("DELETE", "/zones/"+id.String(), nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
}

// TestAssignCCTVToZone_Success menguji bulk-assign CCTV ke zona.
func TestAssignCCTVToZone_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	zoneID, cctvID1, cctvID2 := uuid.New(), uuid.New(), uuid.New()

	// Muat zona.
	mock.ExpectQuery(`SELECT \* FROM "zones"`).
		WillReturnRows(zoneRows(zoneID, "Zona Utara", "Wilayah utara"))
	// Hitung jumlah CCTV pada daftar.
	mock.ExpectQuery(`^SELECT count\(\*\) FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	// Transaksi update zone_id.
	mock.ExpectBegin()
	mock.ExpectExec(`^UPDATE "cctvs" SET "zone_id"`).
		WillReturnResult(sqlmock.NewResult(1, 2))
	mock.ExpectCommit()

	app := newTestApp()
	app.Post("/zones/:id/assign-cctvs", withUserLocals(uuid.New().String(), "admin"), AssignCCTVToZone)

	body := `{"cctv_ids":["` + cctvID1.String() + `","` + cctvID2.String() + `"]}`
	req := httptest.NewRequest("POST", "/zones/"+zoneID.String()+"/assign-cctvs", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
}

// TestAssignUsersToZone_Success menguji assign teknisi ke zona.
func TestAssignUsersToZone_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	zoneID, userID := uuid.New(), uuid.New()

	// Muat zona.
	mock.ExpectQuery(`SELECT \* FROM "zones"`).
		WillReturnRows(zoneRows(zoneID, "Zona Utara", "Wilayah utara"))
	// Hitung jumlah user teknisi pada daftar.
	mock.ExpectQuery(`^SELECT count\(\*\) FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	// Transaksi: hapus penugasan lama + sisipkan baru.
	mock.ExpectBegin()
	mock.ExpectExec(`^DELETE FROM "user_zones"`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`^INSERT INTO "user_zones"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	app := newTestApp()
	app.Post("/zones/:id/assign-users", withUserLocals(uuid.New().String(), "admin"), AssignUsersToZone)

	body := `{"user_ids":["` + userID.String() + `"]}`
	req := httptest.NewRequest("POST", "/zones/"+zoneID.String()+"/assign-users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
}

// TestAssignUsersToZone_InvalidUser memastikan user bukan teknisi ditolak.
func TestAssignUsersToZone_InvalidUser(t *testing.T) {
	_, mock := setupMockGORM(t)
	zoneID := uuid.New()

	mock.ExpectQuery(`SELECT \* FROM "zones"`).
		WillReturnRows(zoneRows(zoneID, "Zona Utara", "Wilayah utara"))
	mock.ExpectQuery(`^SELECT count\(\*\) FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	app := newTestApp()
	app.Post("/zones/:id/assign-users", withUserLocals(uuid.New().String(), "admin"), AssignUsersToZone)

	body := `{"user_ids":["` + uuid.New().String() + `"]}`
	req := httptest.NewRequest("POST", "/zones/"+zoneID.String()+"/assign-users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetAllZonesForSelect_Success menguji daftar zona tanpa paginasi.
func TestGetAllZonesForSelect_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	id := uuid.New()

	mock.ExpectQuery(`SELECT \* FROM "zones"`).
		WillReturnRows(zoneRows(id, "Zona Utara", "Wilayah utara"))

	app := newTestApp()
	app.Get("/zones/all", withUserLocals(uuid.New().String(), "teknisi"), GetAllZonesForSelect)

	req := httptest.NewRequest("GET", "/zones/all", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}
