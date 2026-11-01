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
	mock.MatchExpectationsInOrder(false)
	zoneID, userID := uuid.New(), uuid.New()

	// Muat zona.
	mock.ExpectQuery(`SELECT \* FROM "zones"`).
		WillReturnRows(zoneRows(zoneID, "Zona Utara", "Wilayah utara"))
	// Hitung jumlah user teknisi pada daftar.
	mock.ExpectQuery(`^SELECT count\(\*\) FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	// Ambil penugasan lama agar notifikasi hanya dikirim ke teknisi baru.
	mock.ExpectQuery(`^SELECT \* FROM "user_zones" WHERE zone_id = \$1`).
		WithArgs(zoneID).
		WillReturnRows(sqlmock.NewRows([]string{"zone_id", "user_id"}))
	// Transaksi: hapus penugasan lama + sisipkan baru.
	mock.ExpectBegin()
	mock.ExpectExec(`^DELETE FROM "user_zones"`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`^INSERT INTO "user_zones"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	// Notifikasi in-app disimpan sinkron sebelum respons sukses dikirim.
	mock.ExpectExec(`^INSERT INTO "notifications"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

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

// TestAudit_ZoneCreateMustPersistSelectedSite menguji persistensi site_id saat
// pembuatan zona (baik melalui konteks/header site aktif maupun body site_id),
// visibilitas pada list/dropdown site tersebut, serta konsistensi site antara
// zona dan CCTV pada penugasan maupun pembuatan CCTV (Issue #177 / M04).
func TestAudit_ZoneCreateMustPersistSelectedSite(t *testing.T) {
	t.Run("create_zone_persists_active_site_and_appears_in_site_list_and_dropdown", func(t *testing.T) {
		_, mock := setupMockGORM(t)
		siteID := uuid.New()
		zoneID := uuid.New()

		mock.ExpectExec(`^INSERT INTO "zones"`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), nil, "Zona Inti KIPP", "Kawasan Inti", siteID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// GET /zones dengan site aktif yang sama
		mock.ExpectQuery(`^SELECT count\(\*\) FROM "zones" WHERE site_id = \$1`).
			WithArgs(siteID.String()).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`SELECT \* FROM "zones" WHERE site_id = \$1`).
			WithArgs(siteID.String(), 10).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "created_at", "updated_at", "deleted_at", "nama", "deskripsi", "site_id",
			}).AddRow(zoneID, time.Now(), time.Now(), nil, "Zona Inti KIPP", "Kawasan Inti", siteID))
		mock.ExpectQuery(`^SELECT count\(\*\) FROM "cctvs"`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectQuery(`^SELECT count\(\*\) FROM "user_zones"`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		// GET /zones/all dengan header X-Site-ID tanpa middleware SiteContext
		mock.ExpectQuery(`SELECT \* FROM "zones" WHERE site_id = \$1`).
			WithArgs(siteID.String()).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "created_at", "updated_at", "deleted_at", "nama", "deskripsi", "site_id",
			}).AddRow(zoneID, time.Now(), time.Now(), nil, "Zona Inti KIPP", "Kawasan Inti", siteID))

		app := newTestApp()
		adminID := uuid.New().String()
		app.Post("/zones", withUserSiteLocals(adminID, "admin", siteID.String()), CreateZone)
		app.Get("/zones", withUserSiteLocals(adminID, "admin", siteID.String()), GetAllZones)
		app.Get("/zones/all", withUserLocals(adminID, "admin"), GetAllZonesForSelect)

		createReq := httptest.NewRequest("POST", "/zones", strings.NewReader(`{"nama":"Zona Inti KIPP","deskripsi":"Kawasan Inti"}`))
		createReq.Header.Set("Content-Type", "application/json")
		createResp, err := app.Test(createReq)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, createResp.StatusCode)

		var createBody map[string]interface{}
		assert.NoError(t, json.NewDecoder(createResp.Body).Decode(&createBody))
		createdData := createBody["data"].(map[string]interface{})
		assert.Equal(t, siteID.String(), createdData["site_id"])

		listReq := httptest.NewRequest("GET", "/zones?page=1&limit=10", nil)
		listResp, err := app.Test(listReq)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, listResp.StatusCode)

		var listBody map[string]interface{}
		assert.NoError(t, json.NewDecoder(listResp.Body).Decode(&listBody))
		rows := listBody["data"].(map[string]interface{})["rows"].([]interface{})
		assert.Len(t, rows, 1)
		assert.Equal(t, siteID.String(), rows[0].(map[string]interface{})["site_id"])

		selectReq := httptest.NewRequest("GET", "/zones/all", nil)
		selectReq.Header.Set("X-Site-ID", siteID.String())
		selectResp, err := app.Test(selectReq)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, selectResp.StatusCode)

		var selectBody map[string]interface{}
		assert.NoError(t, json.NewDecoder(selectResp.Body).Decode(&selectBody))
		selectData := selectBody["data"].([]interface{})
		assert.Len(t, selectData, 1)
		assert.Equal(t, siteID.String(), selectData[0].(map[string]interface{})["site_id"])
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("global_admin_can_specify_site_id_in_body_and_conflicting_site_is_rejected", func(t *testing.T) {
		_, mock := setupMockGORM(t)
		siteA := uuid.New()
		siteB := uuid.New()

		mock.ExpectExec(`^INSERT INTO "zones"`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), nil, "Zona Timur", "", siteA).
			WillReturnResult(sqlmock.NewResult(1, 1))

		app := newTestApp()
		adminID := uuid.New().String()
		app.Post("/zones-global", withUserLocals(adminID, "admin"), CreateZone)
		app.Post("/zones-scoped", withUserSiteLocals(adminID, "admin", siteA.String()), CreateZone)

		// Global admin mengirim site_id di body -> tersimpan dengan siteA
		req1 := httptest.NewRequest("POST", "/zones-global", strings.NewReader(`{"nama":"Zona Timur","site_id":"`+siteA.String()+`"}`))
		req1.Header.Set("Content-Type", "application/json")
		resp1, err := app.Test(req1)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, resp1.StatusCode)

		var body1 map[string]interface{}
		assert.NoError(t, json.NewDecoder(resp1.Body).Decode(&body1))
		assert.Equal(t, siteA.String(), body1["data"].(map[string]interface{})["site_id"])

		// Scoped admin (siteA) mengirim site_id berbeda (siteB) di body -> 400
		req2 := httptest.NewRequest("POST", "/zones-scoped", strings.NewReader(`{"nama":"Zona Konflik","site_id":"`+siteB.String()+`"}`))
		req2.Header.Set("Content-Type", "application/json")
		resp2, err := app.Test(req2)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp2.StatusCode)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("assign_cctv_and_create_cctv_enforce_zone_site_consistency", func(t *testing.T) {
		_, mock := setupMockGORM(t)
		siteA := uuid.New()
		siteB := uuid.New()
		zoneA := uuid.New()
		cctvInSiteB := uuid.New()

		// 1) AssignCCTVToZone oleh global admin ke zoneA (milik siteA) dengan CCTV dari siteB -> ditolak 400
		mock.ExpectQuery(`SELECT \* FROM "zones"`).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "created_at", "updated_at", "deleted_at", "nama", "deskripsi", "site_id",
			}).AddRow(zoneA, time.Now(), time.Now(), nil, "Zona A", "Kawasan A", siteA))
		mock.ExpectQuery(`^SELECT count\(\*\) FROM "cctvs" WHERE \(id IN \(\$1\) AND deleted_at IS NULL\) AND site_id = \$2`).
			WithArgs(cctvInSiteB, siteA).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		// 2) CreateCCTV pada siteB tetapi memilih zoneA (milik siteA) -> ditolak 400
		mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
			WillReturnError(gorm.ErrRecordNotFound)
		mock.ExpectQuery(`^SELECT \* FROM "zones"`).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "created_at", "updated_at", "deleted_at", "nama", "deskripsi", "site_id",
			}).AddRow(zoneA, time.Now(), time.Now(), nil, "Zona A", "Kawasan A", siteA))

		app := newTestApp()
		adminID := uuid.New().String()
		app.Post("/zones/:id/assign-cctvs", withUserLocals(adminID, "admin"), AssignCCTVToZone)
		app.Post("/cctv", withUserLocals(adminID, "admin"), CreateCCTV)

		assignReq := httptest.NewRequest("POST", "/zones/"+zoneA.String()+"/assign-cctvs",
			strings.NewReader(`{"cctv_ids":["`+cctvInSiteB.String()+`"]}`))
		assignReq.Header.Set("Content-Type", "application/json")
		assignResp, err := app.Test(assignReq)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, assignResp.StatusCode)

		createCCTVBody := `{"kode_cctv":"CAM-CROSS-01","nama":"Kamera Lintas Site","status":"aktif","site_id":"` + siteB.String() + `","zone_id":"` + zoneA.String() + `"}`
		createCCTVReq := httptest.NewRequest("POST", "/cctv", strings.NewReader(createCCTVBody))
		createCCTVReq.Header.Set("Content-Type", "application/json")
		createCCTVResp, err := app.Test(createCCTVReq)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, createCCTVResp.StatusCode)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
