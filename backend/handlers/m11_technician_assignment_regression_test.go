package handlers

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"sigap-backend/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestM11_AssignMaintenanceLog_CrossSite_Rejected memastikan penugasan teknisi yang bukan
// anggota site CCTV ditolak dengan HTTP 400 (Issue #184 / M11).
func TestM11_AssignMaintenanceLog_CrossSite_Rejected(t *testing.T) {
	_, mock := setupMockGORM(t)
	mock.MatchExpectationsInOrder(false)

	logID, cctvID, siteA, techID := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	// 1. Muat log pemeliharaan terjadwal
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(scheduledMaintenanceRows(logID, cctvID))

	// 2. Muat user tujuan (berperan teknisi)
	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnRows(userRow(techID, "teknisi_b", "teknisi"))

	// 3. Muat CCTV (berada di siteA)
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kode_cctv", "nama", "status", "site_id"}).
			AddRow(cctvID, "CCTV-01", "Kamera Site A", "aktif", siteA))

	// 4. Verifikasi user_sites: teknisi TIDAK terdaftar di siteA (count 0)
	mock.ExpectQuery(`^SELECT count\(\*\) FROM "user_sites" WHERE site_id = \$1 AND user_id = \$2`).
		WithArgs(siteA, techID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	app := newTestApp()
	app.Put("/maintenance-logs/:id/assign", withUserLocals(uuid.New().String(), "admin"), AssignMaintenanceLog)

	body := `{"technician_id":"` + techID.String() + `"}`
	req := httptest.NewRequest("PUT", "/maintenance-logs/"+logID.String()+"/assign", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "error", result["status"])
	assert.Contains(t, result["message"], "Teknisi tidak ditugaskan ke site tempat CCTV berada")
}

// TestM11_AssignMaintenanceLog_SameSite_Success memastikan penugasan teknisi yang terdaftar
// pada site CCTV berhasil dengan HTTP 200 (Issue #184 / M11).
func TestM11_AssignMaintenanceLog_SameSite_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	mock.MatchExpectationsInOrder(false)

	logID, cctvID, siteA, techID := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	// 1. Muat log pemeliharaan terjadwal
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(scheduledMaintenanceRows(logID, cctvID))

	// 2. Muat user tujuan (berperan teknisi)
	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnRows(userRow(techID, "teknisi_a", "teknisi"))

	// 3. Muat CCTV (berada di siteA)
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kode_cctv", "nama", "status", "site_id"}).
			AddRow(cctvID, "CCTV-01", "Kamera Site A", "aktif", siteA))

	// 4. Verifikasi user_sites: teknisi terdaftar di siteA (count 1)
	mock.ExpectQuery(`^SELECT count\(\*\) FROM "user_sites" WHERE site_id = \$1 AND user_id = \$2`).
		WithArgs(siteA, techID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	// 5. Simpan log pemeliharaan berstatus ongoing
	mock.ExpectExec(`^UPDATE "maintenance_logs" SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// 6. Preload reload untuk respon
	reload := sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"cctv_id", "technician_id", "action_taken", "notes", "status",
		"scheduled_date", "completed_at", "image_url", "site_id",
	}).AddRow(logID, time.Now(), time.Now(), nil, cctvID, techID, "Pembersihan", "", "ongoing", nil, nil, "", siteA)
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(reload)
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kode_cctv", "nama", "status"}))
	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "role"}))

	app := newTestApp()
	app.Put("/maintenance-logs/:id/assign", withUserLocals(uuid.New().String(), "admin"), AssignMaintenanceLog)

	body := `{"technician_id":"` + techID.String() + `"}`
	req := httptest.NewRequest("PUT", "/maintenance-logs/"+logID.String()+"/assign", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	data := result["data"].(map[string]interface{})
	assert.Equal(t, "ongoing", data["status"])
}

// TestM11_CreateMaintenanceSchedule_CrossSite_Rejected memastikan pembuatan jadwal dengan
// teknisi dari site lain ditolak dengan HTTP 400 (Issue #184 / M11).
func TestM11_CreateMaintenanceSchedule_CrossSite_Rejected(t *testing.T) {
	_, mock := setupMockGORM(t)
	mock.MatchExpectationsInOrder(false)

	cctvID, siteA, techID := uuid.New(), uuid.New(), uuid.New()

	// 1. Muat CCTV (berada di siteA)
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kode_cctv", "nama", "status", "site_id"}).
			AddRow(cctvID, "CCTV-01", "Kamera Site A", "aktif", siteA))

	// 2. Muat teknisi tujuan
	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnRows(userRow(techID, "teknisi_b", "teknisi"))

	// 3. Verifikasi user_sites: bukan anggota siteA
	mock.ExpectQuery(`^SELECT count\(\*\) FROM "user_sites" WHERE site_id = \$1 AND user_id = \$2`).
		WithArgs(siteA, techID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	app := newTestApp()
	app.Post("/maintenance-schedules", withUserLocals(uuid.New().String(), "admin"), CreateMaintenanceSchedule)

	body := `{"cctv_id":"` + cctvID.String() + `","technician_id":"` + techID.String() +
		`","action_taken":"Pemeriksaan berkala","frequency":"weekly"}`
	req := httptest.NewRequest("POST", "/maintenance-schedules", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "error", result["status"])
	assert.Contains(t, result["message"], "Teknisi tidak ditugaskan ke site tempat CCTV berada")
}

// TestM11_UpdateMaintenanceSchedule_CrossSite_Rejected memastikan update teknisi jadwal
// ke teknisi di luar site ditolak dengan HTTP 400 (Issue #184 / M11).
func TestM11_UpdateMaintenanceSchedule_CrossSite_Rejected(t *testing.T) {
	_, mock := setupMockGORM(t)
	mock.MatchExpectationsInOrder(false)

	schedID, cctvID, siteA, techID := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	// 1. Muat jadwal saat ini
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_schedules"`).
		WillReturnRows(maintenanceScheduleRows(schedID, cctvID, nil, "weekly", true))

	// 2. Muat teknisi tujuan
	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnRows(userRow(techID, "teknisi_b", "teknisi"))

	// 3. Muat CCTV jadwal
	mock.ExpectQuery(`^SELECT "id",\s*"?site_id"? FROM "cctvs"|^SELECT .* FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "site_id"}).AddRow(cctvID, siteA))

	// 4. Verifikasi membership siteA: tidak terdaftar
	mock.ExpectQuery(`^SELECT count\(\*\) FROM "user_sites" WHERE site_id = \$1 AND user_id = \$2`).
		WithArgs(siteA, techID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	app := newTestApp()
	app.Put("/maintenance-schedules/:id", withUserLocals(uuid.New().String(), "admin"), UpdateMaintenanceSchedule)

	body := `{"technician_id":"` + techID.String() + `"}`
	req := httptest.NewRequest("PUT", "/maintenance-schedules/"+schedID.String(), bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "error", result["status"])
	assert.Contains(t, result["message"], "Teknisi tidak ditugaskan ke site tempat CCTV berada")
}

// TestM11_AssignUsersToSite_ReconcilesRemovedTechnicians memastikan pencabutan teknisi dari site
// merekonsiliasi tugas aktif & jadwal pemeliharaan (Issue #184 / M11).
func TestM11_AssignUsersToSite_ReconcilesRemovedTechnicians(t *testing.T) {
	_, mock := setupMockGORM(t)
	mock.MatchExpectationsInOrder(false)

	siteID := uuid.New()
	techStay := uuid.New()
	techRemoved := uuid.New()
	cctvID := uuid.New()

	// 1. Muat site
	mock.ExpectQuery(`SELECT \* FROM "sites"`).
		WillReturnRows(siteRows(siteID, "Site Nusantara", "IKN-01", false))

	// 2. Validasi teknisi baru (hanya techStay yang didaftarkan kembali)
	mock.ExpectQuery(`^SELECT count\(\*\) FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	// 3. Transaksi: muat penugasan lama (memuat techStay dan techRemoved)
	mock.ExpectBegin()
	mock.ExpectQuery(`^SELECT "user_id" FROM "user_sites" WHERE site_id = \$1`).
		WithArgs(siteID).
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow(techStay).AddRow(techRemoved))

	// 4. Hapus penugasan lama & insert penugasan baru
	mock.ExpectExec(`^DELETE FROM "user_sites" WHERE site_id = \$1`).
		WithArgs(siteID).
		WillReturnResult(sqlmock.NewResult(1, 2))
	mock.ExpectExec(`^INSERT INTO "user_sites"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// 5. Rekonsiliasi: ambil CCTV pada site ini
	mock.ExpectQuery(`^SELECT "id" FROM "cctvs" WHERE site_id = \$1`).
		WithArgs(siteID).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(cctvID))

	// 6. Rekonsiliasi: unassign log aktif milik techRemoved
	mock.ExpectExec(`^UPDATE "maintenance_logs" SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// 7. Rekonsiliasi: unassign jadwal berulang milik techRemoved
	mock.ExpectExec(`^UPDATE "maintenance_schedules" SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	app := newTestApp()
	app.Post("/sites/:id/assign-users", withUserLocals(uuid.New().String(), "admin"), AssignUsersToSite)

	body := `{"user_ids":["` + techStay.String() + `"]}`
	req := httptest.NewRequest("POST", "/sites/"+siteID.String()+"/assign-users", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
}

// TestM11_UpdateUser_RoleChangeFromTechnician_ReconcilesTasks memastikan pergantian role
// dari teknisi ke role lain merekonsiliasi tugas aktif & jadwal berulang (Issue #184 / M11).
func TestM11_UpdateUser_RoleChangeFromTechnician_ReconcilesTasks(t *testing.T) {
	_, mock := setupMockGORM(t)
	mock.MatchExpectationsInOrder(false)

	techID := uuid.New()
	techRow := modelsUserRow{
		ID:        techID,
		Username:  "teknisi_promo",
		Hash:      "hash",
		FullName:  "Teknisi Senior",
		Email:     "promo@sigap.local",
		Role:      "teknisi",
		TokenVer:  1,
		CreatedAt: time.Now(),
	}

	// 1. Muat data user lama
	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnRows(newUserMockRows([]modelsUserRow{techRow}))

	// 2. Transaksi update role ke admin + rekonsiliasi
	mock.ExpectBegin()
	mock.ExpectExec(`^UPDATE "users" SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Rekonsiliasi tugas aktif
	mock.ExpectExec(`^UPDATE "maintenance_logs" SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Rekonsiliasi jadwal aktif
	mock.ExpectExec(`^UPDATE "maintenance_schedules" SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Bersihkan user_sites & user_zones
	mock.ExpectExec(`^DELETE FROM "user_sites" WHERE user_id = \$1`).
		WithArgs(techID).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`^DELETE FROM "user_zones" WHERE user_id = \$1`).
		WithArgs(techID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	app := newTestApp()
	app.Put("/users/:id", withUserLocals(uuid.New().String(), "admin"), UpdateUser)

	body := `{"role":"admin"}`
	req := httptest.NewRequest("PUT", "/users/"+techID.String(), bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	var updated models.User
	dataBytes, _ := json.Marshal(result["data"])
	require.NoError(t, json.Unmarshal(dataBytes, &updated))
	assert.Equal(t, "admin", updated.Role)
}

// TestM11_DeleteUser_Technician_ReconcilesTasks memastikan penghapusan teknisi
// merekonsiliasi tugas aktif & jadwal berulang (Issue #184 / M11).
func TestM11_DeleteUser_Technician_ReconcilesTasks(t *testing.T) {
	_, mock := setupMockGORM(t)
	mock.MatchExpectationsInOrder(false)

	techID := uuid.New()
	adminID := uuid.New()
	techRow := modelsUserRow{
		ID:        techID,
		Username:  "teknisi_hapus",
		Hash:      "hash",
		FullName:  "Teknisi Dihapus",
		Email:     "hapus@sigap.local",
		Role:      "teknisi",
		TokenVer:  1,
		CreatedAt: time.Now(),
	}

	// 1. Muat data user
	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnRows(newUserMockRows([]modelsUserRow{techRow}))

	// 2. Transaksi delete user + rekonsiliasi
	mock.ExpectBegin()
	mock.ExpectExec(`^UPDATE "users" SET "deleted_at"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Rekonsiliasi tugas aktif
	mock.ExpectExec(`^UPDATE "maintenance_logs" SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Rekonsiliasi jadwal aktif
	mock.ExpectExec(`^UPDATE "maintenance_schedules" SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Bersihkan user_sites & user_zones
	mock.ExpectExec(`^DELETE FROM "user_sites" WHERE user_id = \$1`).
		WithArgs(techID).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`^DELETE FROM "user_zones" WHERE user_id = \$1`).
		WithArgs(techID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	app := newTestApp()
	app.Delete("/users/:id", withUserLocals(adminID.String(), "admin"), DeleteUser)

	req := httptest.NewRequest("DELETE", "/users/"+techID.String(), nil)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
}
