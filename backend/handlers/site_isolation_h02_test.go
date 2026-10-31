package handlers

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"sigap-backend/utils"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func cctvRowForSite(id, siteID uuid.UUID) *sqlmock.Rows {
	return cctvRowForSiteWithPayload(id, siteID, "encrypted-payload")
}

func cctvRowForSiteWithPayload(id, siteID uuid.UUID, qrPayload string) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"kode_cctv", "id_channel", "nama", "lokasi_area",
		"longitude", "latitude", "jenis_cctv", "sumber_listrik",
		"tipe_koneksi", "indoor_outdoor", "pemilik", "status",
		"ip_nvr", "ip_camera", "keterangan", "qr_payload", "health_score",
		"channel_port", "rtsp_password", "stream_enabled", "site_id",
	}).AddRow(
		id, time.Now(), time.Now(), nil,
		"CCTV-SITE-B", "CH-02", "Kamera Kawasan B", "Area B",
		117.2, -0.6, "Bullet", "PLN",
		"Fiber", "outdoor", "OIKN", "aktif",
		"192.168.2.10", "192.168.2.11", "Terbatas", qrPayload, 100,
		20, "encrypted-rtsp", true, siteID,
	)
}

func zoneRowForSite(id, siteID uuid.UUID, name string) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"nama", "deskripsi", "site_id",
	}).AddRow(id, time.Now(), time.Now(), nil, name, "Zona Kawasan B", siteID)
}

// H02: GetCCTVByID wajib menolak akses lintas site ketika pengguna terikat ke site A
// tetapi meminta objek CCTV milik site B.
func TestAudit_CCTVDetailMustDenyOtherSite(t *testing.T) {
	_, mock := setupMockGORM(t)
	siteA := uuid.New()
	siteB := uuid.New()
	cctvID := uuid.New()

	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(cctvRowForSite(cctvID, siteB))

	app := newTestApp()
	app.Get("/cctv/:id", withUserSiteLocals(uuid.New().String(), "teknisi", siteA.String()), GetCCTVByID)

	resp, err := app.Test(httptest.NewRequest("GET", "/cctv/"+cctvID.String(), nil))
	require.NoError(t, err)
	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode, "detail CCTV site lain harus ditolak 403")
	assert.NotContains(t, string(bodyBytes), "Kamera Kawasan B", "respons 403 tidak boleh membocorkan detail CCTV site lain")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// H02: CreateMaintenanceLog wajib menolak pembuatan log untuk CCTV di site lain.
func TestAudit_MaintenanceCreateMustDenyOtherSite(t *testing.T) {
	_, mock := setupMockGORM(t)
	siteA := uuid.New()
	siteB := uuid.New()
	cctvID := uuid.New()

	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(cctvRowForSite(cctvID, siteB))

	app := newTestApp()
	app.Post("/maintenance-logs", withUserSiteLocals(uuid.New().String(), "teknisi", siteA.String()), CreateMaintenanceLog)

	payload := `{"cctv_id":"` + cctvID.String() + `","action_taken":"Bersihkan lensa kamera","status":"scheduled"}`
	req := httptest.NewRequest("POST", "/maintenance-logs", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode, "pembuatan log pemeliharaan untuk CCTV site lain harus ditolak 403")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// H02 kontrol: CreateFieldMaintenance menolak site lain dengan 403.
func TestAudit_FieldCheckMustDenyOtherSite_Control(t *testing.T) {
	_, mock := setupMockGORM(t)
	siteA := uuid.New()
	siteB := uuid.New()
	cctvID := uuid.New()

	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(cctvRowForSite(cctvID, siteB))

	app := newTestApp()
	app.Post("/field-maintenance", withUserSiteLocals(uuid.New().String(), "teknisi", siteA.String()), CreateFieldMaintenance)

	payload := `{"cctv_id":"` + cctvID.String() + `","checks":[{"name":"Jaringan internet kamera","result":"ok","note":""}],"images":[{"image_url":"/api/uploads/foto.jpg"}]}`
	req := httptest.NewRequest("POST", "/field-maintenance", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// H02: Seluruh endpoint turunan dan penulisan objek CCTV (QR, scan, snapshot,
// snapshot-status, timeline, health-score, health-history, update, delete, rotate-qr)
// wajib menolak akses lintas site dengan 403 tanpa membocorkan data.
func TestSiteIsolation_CCTVObjectAccessAndDerivativesDenyOtherSite(t *testing.T) {
	t.Setenv("AES_SECRET_KEY", "sigap_aes_test_secret_2026")
	siteA := uuid.New()
	siteB := uuid.New()
	cctvID := uuid.New()
	qrPayload, err := utils.EncryptAES256(cctvID.String())
	require.NoError(t, err)

	t.Run("GetCCTVQRCode", func(t *testing.T) {
		_, mock := setupMockGORM(t)
		mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
			WillReturnRows(cctvRowForSiteWithPayload(cctvID, siteB, qrPayload))

		app := newTestApp()
		app.Get("/cctv/:id/qrcode", withUserSiteLocals(uuid.New().String(), "teknisi", siteA.String()), GetCCTVQRCode)

		resp, err := app.Test(httptest.NewRequest("GET", "/cctv/"+cctvID.String()+"/qrcode", nil))
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
		assert.NotEqual(t, "image/png", resp.Header.Get("Content-Type"))
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("ScanCCTV", func(t *testing.T) {
		_, mock := setupMockGORM(t)
		mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
			WillReturnRows(cctvRowForSiteWithPayload(cctvID, siteB, qrPayload))

		app := newTestApp()
		app.Post("/cctv/scan", withUserSiteLocals(uuid.New().String(), "teknisi", siteA.String()), ScanCCTV)

		req := httptest.NewRequest("POST", "/cctv/scan", strings.NewReader(`{"qr_payload":"`+qrPayload+`"}`))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		bodyBytes, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
		assert.NotContains(t, string(bodyBytes), "Kamera Kawasan B")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetCCTVSnapshot", func(t *testing.T) {
		_, mock := setupMockGORM(t)
		mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
			WillReturnRows(cctvRowForSite(cctvID, siteB))

		app := newTestApp()
		app.Get("/cctv/:id/snapshot", withUserSiteLocals(uuid.New().String(), "teknisi", siteA.String()), GetCCTVSnapshot)

		resp, err := app.Test(httptest.NewRequest("GET", "/cctv/"+cctvID.String()+"/snapshot", nil))
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetCCTVSnapshotStatus", func(t *testing.T) {
		_, mock := setupMockGORM(t)
		mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
			WillReturnRows(cctvRowForSite(cctvID, siteB))

		app := newTestApp()
		app.Get("/cctv/:id/snapshot-status", withUserSiteLocals(uuid.New().String(), "teknisi", siteA.String()), GetCCTVSnapshotStatus)

		resp, err := app.Test(httptest.NewRequest("GET", "/cctv/"+cctvID.String()+"/snapshot-status", nil))
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetCCTVTimeline", func(t *testing.T) {
		_, mock := setupMockGORM(t)
		mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
			WillReturnRows(cctvRowForSite(cctvID, siteB))

		app := newTestApp()
		app.Get("/cctv/:id/timeline", withUserSiteLocals(uuid.New().String(), "teknisi", siteA.String()), GetCCTVTimeline)

		resp, err := app.Test(httptest.NewRequest("GET", "/cctv/"+cctvID.String()+"/timeline", nil))
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetCCTVHealthScore", func(t *testing.T) {
		_, mock := setupMockGORM(t)
		mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
			WillReturnRows(cctvRowForSite(cctvID, siteB))

		app := newTestApp()
		app.Get("/cctv/:id/health-score", withUserSiteLocals(uuid.New().String(), "teknisi", siteA.String()), GetCCTVHealthScore)

		resp, err := app.Test(httptest.NewRequest("GET", "/cctv/"+cctvID.String()+"/health-score", nil))
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetCCTVHealthHistory", func(t *testing.T) {
		_, mock := setupMockGORM(t)
		mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
			WillReturnRows(cctvRowForSite(cctvID, siteB))

		app := newTestApp()
		app.Get("/cctv/:id/health-history", withUserSiteLocals(uuid.New().String(), "teknisi", siteA.String()), GetCCTVHealthHistory)

		resp, err := app.Test(httptest.NewRequest("GET", "/cctv/"+cctvID.String()+"/health-history", nil))
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("UpdateCCTV", func(t *testing.T) {
		_, mock := setupMockGORM(t)
		mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
			WillReturnRows(cctvRowForSite(cctvID, siteB))

		app := newTestApp()
		app.Put("/cctv/:id", withUserSiteLocals(uuid.New().String(), "admin", siteA.String()), UpdateCCTV)

		req := httptest.NewRequest("PUT", "/cctv/"+cctvID.String(), strings.NewReader(`{"nama":"Ubah Lintas Site"}`))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("DeleteCCTV", func(t *testing.T) {
		_, mock := setupMockGORM(t)
		mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
			WillReturnRows(cctvRowForSite(cctvID, siteB))

		app := newTestApp()
		app.Delete("/cctv/:id", withUserSiteLocals(uuid.New().String(), "admin", siteA.String()), DeleteCCTV)

		resp, err := app.Test(httptest.NewRequest("DELETE", "/cctv/"+cctvID.String(), nil))
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("RotateQRCode", func(t *testing.T) {
		_, mock := setupMockGORM(t)
		mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
			WillReturnRows(cctvRowForSite(cctvID, siteB))

		app := newTestApp()
		app.Post("/cctv/:id/rotate-qr", withUserSiteLocals(uuid.New().String(), "admin", siteA.String()), RotateQRCode)

		resp, err := app.Test(httptest.NewRequest("POST", "/cctv/"+cctvID.String()+"/rotate-qr", nil))
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// H02: Ketika CCTV dipindahkan dari site A ke site B, teknisi site A yang sebelumnya
// ditugaskan pada log pemeliharaan tersebut (ownership cocok) tetap harus ditolak (403)
// saat mengakses detail, mengubah, atau menyelesaikan log.
func TestSiteIsolation_MaintenanceDeniesAccessAfterCCTVMovedToOtherSite(t *testing.T) {
	siteA := uuid.New()
	siteB := uuid.New()
	techID := uuid.New()
	cctvID := uuid.New()
	logID := uuid.New()

	t.Run("GetMaintenanceLogByID_AfterCCTVMoved", func(t *testing.T) {
		_, mock := setupMockGORM(t)
		mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
			WillReturnRows(knownMaintenanceRows(logID, techID, cctvID))
		mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
			WillReturnRows(cctvRowForSite(cctvID, siteB))
		mock.ExpectQuery(`^SELECT \* FROM "users"`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "username", "role"}).AddRow(techID, "tek1", "teknisi"))

		app := newTestApp()
		app.Get("/maintenance-logs/:id", withUserSiteLocals(techID.String(), "teknisi", siteA.String()), GetMaintenanceLogByID)

		resp, err := app.Test(httptest.NewRequest("GET", "/maintenance-logs/"+logID.String(), nil))
		require.NoError(t, err)
		bodyBytes, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
		assert.NotContains(t, string(bodyBytes), "Bersihkan lensa kamera")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("UpdateMaintenanceLog_AfterCCTVMoved", func(t *testing.T) {
		_, mock := setupMockGORM(t)
		mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
			WillReturnRows(knownMaintenanceRows(logID, techID, cctvID))
		mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
			WillReturnRows(cctvRowForSite(cctvID, siteB))

		app := newTestApp()
		app.Put("/maintenance-logs/:id", withUserSiteLocals(techID.String(), "teknisi", siteA.String()), UpdateMaintenanceLog)

		req := httptest.NewRequest("PUT", "/maintenance-logs/"+logID.String(), strings.NewReader(`{"notes":"Ubah setelah pindah site"}`))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("CompleteMaintenanceLog_AfterCCTVMoved", func(t *testing.T) {
		_, mock := setupMockGORM(t)
		mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
			WillReturnRows(knownMaintenanceRows(logID, techID, cctvID))
		mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
			WillReturnRows(cctvRowForSite(cctvID, siteB))

		app := newTestApp()
		app.Post("/maintenance-logs/:id/complete", withUserSiteLocals(techID.String(), "teknisi", siteA.String()), CompleteMaintenanceLog)

		payload := `{` + completeChecks(true) + `,"images":[{"image_url":"/api/uploads/a.jpg"}]}`
		req := httptest.NewRequest("POST", "/maintenance-logs/"+logID.String()+"/complete", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// H02: Detail dan operasi tulis zona wajib menolak akses bila zona berada di site lain.
func TestSiteIsolation_ZoneDetailAndWriteDenyOtherSite(t *testing.T) {
	siteA := uuid.New()
	siteB := uuid.New()
	zoneID := uuid.New()

	t.Run("GetZoneByID", func(t *testing.T) {
		_, mock := setupMockGORM(t)
		mock.ExpectQuery(`^SELECT \* FROM "zones"`).
			WillReturnRows(zoneRowForSite(zoneID, siteB, "Zona Rahasia Site B"))

		app := newTestApp()
		app.Get("/zones/:id", withUserSiteLocals(uuid.New().String(), "teknisi", siteA.String()), GetZoneByID)

		resp, err := app.Test(httptest.NewRequest("GET", "/zones/"+zoneID.String(), nil))
		require.NoError(t, err)
		bodyBytes, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
		assert.NotContains(t, string(bodyBytes), "Zona Rahasia Site B")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("UpdateZone", func(t *testing.T) {
		_, mock := setupMockGORM(t)
		mock.ExpectQuery(`^SELECT \* FROM "zones"`).
			WillReturnRows(zoneRowForSite(zoneID, siteB, "Zona Rahasia Site B"))

		app := newTestApp()
		app.Put("/zones/:id", withUserSiteLocals(uuid.New().String(), "admin", siteA.String()), UpdateZone)

		req := httptest.NewRequest("PUT", "/zones/"+zoneID.String(), strings.NewReader(`{"nama":"Zona Diubah"}`))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("DeleteZone", func(t *testing.T) {
		_, mock := setupMockGORM(t)
		mock.ExpectQuery(`^SELECT \* FROM "zones"`).
			WillReturnRows(zoneRowForSite(zoneID, siteB, "Zona Rahasia Site B"))

		app := newTestApp()
		app.Delete("/zones/:id", withUserSiteLocals(uuid.New().String(), "admin", siteA.String()), DeleteZone)

		resp, err := app.Test(httptest.NewRequest("DELETE", "/zones/"+zoneID.String(), nil))
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
