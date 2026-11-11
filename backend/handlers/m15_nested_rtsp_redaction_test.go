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
	"github.com/stretchr/testify/require"

	"sigap-backend/models"
	"sigap-backend/utils"
)

func TestAudit_MaintenanceListMustRedactNestedRTSP(t *testing.T) {
	t.Setenv("AES_SECRET_KEY", "sigap_m15_regression_secret_key_2026")

	plainSecret := "super-secret-plain-pw"
	encSecret := "super-secret-enc-pw"
	rawPlainURL := "rtsp://operator:" + plainSecret + "@10.10.1.50:554/Streaming/channels/101"
	rawEncURL := "rtsp://admin:" + encSecret + "@10.10.1.51:554/Streaming/channels/201"
	encryptedURL, err := utils.EncryptRTSPURL(rawEncURL)
	require.NoError(t, err)

	rtspPwSecret := "raw-password-column-secret"

	assertRedactedBody := func(t *testing.T, body string, expectedSecrets ...string) {
		t.Helper()
		for _, secret := range expectedSecrets {
			assert.NotContains(t, body, secret, "nested CCTV response leaked RTSP credential")
		}
		assert.NotContains(t, body, `"rtsp_url":`, "raw rtsp_url field must never serialize in JSON")
		assert.NotContains(t, body, `"rtsp_password":`, "rtsp_password field must never serialize in JSON")
		assert.Contains(t, body, `"rtsp_url_configured":true`)
		assert.Contains(t, body, `:xxxxx@`)
	}

	t.Run("model json marshal redacts RTSPURL and RTSPPassword even without handler call", func(t *testing.T) {
		payload, err := json.Marshal(models.CCTV{
			KodeCCTV:     "CAM-DEFENSE",
			RTSPURL:      rawPlainURL,
			RTSPPassword: rtspPwSecret,
		})
		require.NoError(t, err)
		body := string(payload)
		assert.NotContains(t, body, plainSecret)
		assert.NotContains(t, body, rtspPwSecret)
		assert.NotContains(t, body, `"rtsp_url"`)
		assert.NotContains(t, body, `"rtsp_password"`)
	})

	t.Run("GET /maintenance-logs and /maintenance-logs/:id redact encrypted & legacy RTSP", func(t *testing.T) {
		_, mock := setupMockGORM(t)
		adminID := uuid.New()
		log1ID, log2ID := uuid.New(), uuid.New()
		cctv1ID, cctv2ID := uuid.New(), uuid.New()
		techID := uuid.New()

		mock.ExpectQuery(`^SELECT count\(\*\) FROM "maintenance_logs"`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
		mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "created_at", "updated_at", "deleted_at",
				"cctv_id", "technician_id", "action_taken", "notes", "status",
			}).
				AddRow(log1ID, time.Now(), time.Now(), nil, cctv1ID, techID, "Cek kamera 1", "Catatan", "scheduled").
				AddRow(log2ID, time.Now(), time.Now(), nil, cctv2ID, techID, "Cek kamera 2", "Catatan", "completed"))
		mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "kode_cctv", "nama", "status", "rtsp_url", "rtsp_password",
			}).
				AddRow(cctv1ID, "CAM-01", "Kamera Plain", "aktif", rawPlainURL, rtspPwSecret).
				AddRow(cctv2ID, "CAM-02", "Kamera Enc", "aktif", encryptedURL, rtspPwSecret))
		mock.ExpectQuery(`^SELECT \* FROM "users"`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "username", "role"}).
				AddRow(techID, "teknisi1", "teknisi"))

		app := newTestApp()
		app.Get("/maintenance-logs", withUserLocals(adminID.String(), "admin"), GetAllMaintenanceLogs)

		resp, err := app.Test(httptest.NewRequest("GET", "/maintenance-logs", nil))
		require.NoError(t, err)
		require.Equal(t, fiber.StatusOK, resp.StatusCode)

		var rawBody strings.Builder
		buf := make([]byte, 4096)
		for {
			n, readErr := resp.Body.Read(buf)
			if n > 0 {
				rawBody.Write(buf[:n])
			}
			if readErr != nil {
				break
			}
		}
		assertRedactedBody(t, rawBody.String(), plainSecret, encSecret, encryptedURL, rtspPwSecret)
		assert.NoError(t, mock.ExpectationsWereMet())

		// Detail endpoint
		mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "created_at", "updated_at", "deleted_at",
				"cctv_id", "technician_id", "action_taken", "notes", "status",
			}).AddRow(log2ID, time.Now(), time.Now(), nil, cctv2ID, techID, "Cek kamera 2", "Catatan", "completed"))
		mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "kode_cctv", "nama", "status", "rtsp_url", "rtsp_password",
			}).AddRow(cctv2ID, "CAM-02", "Kamera Enc", "aktif", encryptedURL, rtspPwSecret))
		mock.ExpectQuery(`^SELECT \* FROM "users"`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "username", "role"}).
				AddRow(techID, "teknisi1", "teknisi"))

		app.Get("/maintenance-logs/:id", withUserLocals(adminID.String(), "admin"), GetMaintenanceLogByID)
		respDetail, err := app.Test(httptest.NewRequest("GET", "/maintenance-logs/"+log2ID.String(), nil))
		require.NoError(t, err)
		require.Equal(t, fiber.StatusOK, respDetail.StatusCode)

		var detailBody strings.Builder
		for {
			n, readErr := respDetail.Body.Read(buf)
			if n > 0 {
				detailBody.Write(buf[:n])
			}
			if readErr != nil {
				break
			}
		}
		assertRedactedBody(t, detailBody.String(), encSecret, encryptedURL, rtspPwSecret)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GET /field-maintenance and /field-maintenance/:id redact nested RTSP", func(t *testing.T) {
		_, mock := setupMockGORM(t)
		adminID := uuid.New()
		fmID, cctvID, techID := uuid.New(), uuid.New(), uuid.New()

		mock.ExpectQuery(`^SELECT count\(\*\) FROM "field_maintenances"`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`^SELECT \* FROM "field_maintenances"`).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "created_at", "updated_at", "deleted_at",
				"cctv_id", "technician_id", "overall_status", "notes",
			}).AddRow(fmID, time.Now(), time.Now(), nil, cctvID, techID, "ok", "Normal"))
		mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "kode_cctv", "nama", "status", "rtsp_url", "rtsp_password",
			}).AddRow(cctvID, "CAM-FM", "Kamera Lapangan", "aktif", encryptedURL, rtspPwSecret))
		mock.ExpectQuery(`^SELECT \* FROM "field_maintenance_checks"`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "field_maintenance_id", "name", "result"}))
		mock.ExpectQuery(`^SELECT \* FROM "field_maintenance_images"`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "field_maintenance_id", "image_url"}))
		mock.ExpectQuery(`^SELECT \* FROM "users"`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "username", "role"}).AddRow(techID, "tek1", "teknisi"))

		app := newTestApp()
		app.Get("/field-maintenance", withUserLocals(adminID.String(), "admin"), GetAllFieldMaintenances)
		app.Get("/field-maintenance/:id", withUserLocals(adminID.String(), "admin"), GetFieldMaintenanceByID)

		resp, err := app.Test(httptest.NewRequest("GET", "/field-maintenance", nil))
		require.NoError(t, err)
		require.Equal(t, fiber.StatusOK, resp.StatusCode)
		var decoded map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&decoded))
		encoded, err := json.Marshal(decoded)
		require.NoError(t, err)
		assertRedactedBody(t, string(encoded), encSecret, encryptedURL, rtspPwSecret)

		mock.ExpectQuery(`^SELECT \* FROM "field_maintenances"`).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "created_at", "updated_at", "deleted_at",
				"cctv_id", "technician_id", "overall_status", "notes",
			}).AddRow(fmID, time.Now(), time.Now(), nil, cctvID, techID, "ok", "Normal"))
		mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "kode_cctv", "nama", "status", "rtsp_url", "rtsp_password",
			}).AddRow(cctvID, "CAM-FM", "Kamera Lapangan", "aktif", rawPlainURL, rtspPwSecret))
		mock.ExpectQuery(`^SELECT \* FROM "field_maintenance_checks"`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "field_maintenance_id", "name", "result"}))
		mock.ExpectQuery(`^SELECT \* FROM "field_maintenance_images"`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "field_maintenance_id", "image_url"}))
		mock.ExpectQuery(`^SELECT \* FROM "users"`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "username", "role"}).AddRow(techID, "tek1", "teknisi"))

		respDetail, err := app.Test(httptest.NewRequest("GET", "/field-maintenance/"+fmID.String(), nil))
		require.NoError(t, err)
		require.Equal(t, fiber.StatusOK, respDetail.StatusCode)
		var decodedDetail map[string]interface{}
		require.NoError(t, json.NewDecoder(respDetail.Body).Decode(&decodedDetail))
		encodedDetail, err := json.Marshal(decodedDetail)
		require.NoError(t, err)
		assertRedactedBody(t, string(encodedDetail), plainSecret, rtspPwSecret)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GET /maintenance-schedules, /maintenance-schedules/:id, and /zones/:id redact nested RTSP", func(t *testing.T) {
		_, mock := setupMockGORM(t)
		adminID := uuid.New()
		schedID, cctvID, zoneID := uuid.New(), uuid.New(), uuid.New()

		mock.ExpectQuery(`^SELECT count\(\*\) FROM "maintenance_schedules"`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`^SELECT \* FROM "maintenance_schedules"`).
			WillReturnRows(maintenanceScheduleRows(schedID, cctvID, nil, "daily", true))
		mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "kode_cctv", "nama", "status", "rtsp_url", "rtsp_password",
			}).AddRow(cctvID, "CAM-SCH", "Kamera Jadwal", "aktif", encryptedURL, rtspPwSecret))

		app := newTestApp()
		app.Get("/maintenance-schedules", withUserLocals(adminID.String(), "admin"), GetAllMaintenanceSchedules)
		app.Get("/maintenance-schedules/:id", withUserLocals(adminID.String(), "admin"), GetMaintenanceScheduleByID)
		app.Get("/zones/:id", withUserLocals(adminID.String(), "admin"), GetZoneByID)

		respScheds, err := app.Test(httptest.NewRequest("GET", "/maintenance-schedules", nil))
		require.NoError(t, err)
		require.Equal(t, fiber.StatusOK, respScheds.StatusCode)
		var sMap map[string]interface{}
		require.NoError(t, json.NewDecoder(respScheds.Body).Decode(&sMap))
		sBytes, err := json.Marshal(sMap)
		require.NoError(t, err)
		assertRedactedBody(t, string(sBytes), encSecret, encryptedURL, rtspPwSecret)

		mock.ExpectQuery(`^SELECT \* FROM "maintenance_schedules"`).
			WillReturnRows(maintenanceScheduleRows(schedID, cctvID, nil, "daily", true))
		mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "kode_cctv", "nama", "status", "rtsp_url", "rtsp_password",
			}).AddRow(cctvID, "CAM-SCH", "Kamera Jadwal", "aktif", rawPlainURL, rtspPwSecret))

		respSchedDetail, err := app.Test(httptest.NewRequest("GET", "/maintenance-schedules/"+schedID.String(), nil))
		require.NoError(t, err)
		require.Equal(t, fiber.StatusOK, respSchedDetail.StatusCode)
		var sdMap map[string]interface{}
		require.NoError(t, json.NewDecoder(respSchedDetail.Body).Decode(&sdMap))
		sdBytes, err := json.Marshal(sdMap)
		require.NoError(t, err)
		assertRedactedBody(t, string(sdBytes), plainSecret, rtspPwSecret)

		mock.ExpectQuery(`SELECT \* FROM "zones"`).
			WillReturnRows(zoneRows(zoneID, "Zona Inti", "KIPP"))
		mock.ExpectQuery(`SELECT .* FROM "cctvs"`).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "zone_id", "kode_cctv", "nama", "status", "rtsp_url", "rtsp_password",
			}).AddRow(cctvID, zoneID, "CAM-ZONE", "Kamera Zona", "aktif", encryptedURL, rtspPwSecret))
		mock.ExpectQuery(`SELECT .* FROM "user_zones"`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "zone_id", "user_id"}))

		respZone, err := app.Test(httptest.NewRequest("GET", "/zones/"+zoneID.String(), nil))
		require.NoError(t, err)
		require.Equal(t, fiber.StatusOK, respZone.StatusCode)
		var zMap map[string]interface{}
		require.NoError(t, json.NewDecoder(respZone.Body).Decode(&zMap))
		zBytes, err := json.Marshal(zMap)
		require.NoError(t, err)
		assertRedactedBody(t, string(zBytes), encSecret, encryptedURL, rtspPwSecret)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
