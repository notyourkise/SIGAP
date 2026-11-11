package handlers

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"sigap-backend/config"
	"sigap-backend/middleware"
	"sigap-backend/models"
	"sigap-backend/utils"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func expectProtectedUserCheck(mock sqlmock.Sqlmock, userID string, tokenVersion int64, mustChange bool) {
	mock.ExpectQuery(`^SELECT "token_version",\s*"must_change_password" FROM "users"|^SELECT "token_version".*FROM "users"`).
		WithArgs(userID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"token_version", "must_change_password"}).AddRow(tokenVersion, mustChange))
}

// TestOpsRegression_SiteIsolationFullMiddlewareStack memverifikasi isolasi multi-site
// secara end-to-end melewati middleware Protected() + SiteContext() + RequireRole()
// hingga handler /api/cctv-logs dan /api/streams/diagnostics.
func TestOpsRegression_SiteIsolationFullMiddlewareStack(t *testing.T) {
	t.Setenv("JWT_SECRET", "sigap_ops_regression_jwt_secret_2026")
	t.Setenv("APP_TIMEZONE", "Asia/Makassar")
	t.Setenv("CCTV_METRO_E_DIRECT_PROBE", "false")

	_, mock := setupMockGORM(t)
	injectMetroEProbeResult(t, func(string) bool { return false })
	CheckMetroEReachabilityAtStartup(nil)

	mtxClient, closeSrv := startMTXStateServer(t, nil, nil)
	defer closeSrv()
	injectStreamClient(t, mtxClient)

	app := fiber.New(config.NewFiberConfig(nil))
	app.Get("/api/cctv-logs", middleware.Protected(), middleware.SiteContext(), GetCCTVStreamLogs)
	app.Get("/api/streams/diagnostics", middleware.Protected(), middleware.RequireRole("admin"), middleware.SiteContext(), GetStreamDiagnostics)

	siteA := uuid.New()
	siteB := uuid.New()
	cctvA := uuid.New()
	techID := uuid.NewString()
	adminID := uuid.NewString()

	techToken, err := utils.GenerateToken(techID, "teknisi", 1)
	require.NoError(t, err)
	adminToken, err := utils.GenerateToken(adminID, "admin", 1)
	require.NoError(t, err)

	// 1. Teknisi site A mencoba mengakses log site B via X-Site-ID -> 403 Forbidden.
	expectProtectedUserCheck(mock, techID, 1, false)
	mock.ExpectQuery(`^SELECT \* FROM "sites" WHERE id = \$1`).
		WithArgs(siteB, 1).
		WillReturnRows(siteRows(siteB, "Site B", "SITE-B", false))
	mock.ExpectQuery(`^SELECT count\(\*\) FROM "user_sites" WHERE site_id = \$1 AND user_id = \$2`).
		WithArgs(siteB, techID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	reqCrossSite := httptest.NewRequest("GET", "/api/cctv-logs", nil)
	reqCrossSite.AddCookie(&http.Cookie{Name: utils.SessionCookieName, Value: techToken})
	reqCrossSite.Header.Set("X-Site-ID", siteB.String())
	respCrossSite, err := app.Test(reqCrossSite)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, respCrossSite.StatusCode)

	// 2. Teknisi yang belum ditugaskan ke site mana pun -> fail closed 403 Forbidden.
	expectProtectedUserCheck(mock, techID, 1, false)
	mock.ExpectQuery(`FROM "sites" JOIN user_sites`).
		WithArgs(techID, 1).
		WillReturnError(gorm.ErrRecordNotFound)

	reqUnassigned := httptest.NewRequest("GET", "/api/cctv-logs", nil)
	reqUnassigned.AddCookie(&http.Cookie{Name: utils.SessionCookieName, Value: techToken})
	respUnassigned, err := app.Test(reqUnassigned)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, respUnassigned.StatusCode)

	// 3. Teknisi site A tanpa X-Site-ID otomatis ter-scope ke default site A.
	expectProtectedUserCheck(mock, techID, 1, false)
	mock.ExpectQuery(`FROM "sites" JOIN user_sites`).
		WithArgs(techID, 1).
		WillReturnRows(siteRows(siteA, "Site A", "SITE-A", true))
	expectCCTVStreamLogForSite(mock, siteA.String(), cctvA, "Kamera Site A", "SITE-A-01")

	reqTechOwn := httptest.NewRequest("GET", "/api/cctv-logs", nil)
	reqTechOwn.AddCookie(&http.Cookie{Name: utils.SessionCookieName, Value: techToken})
	respTechOwn, err := app.Test(reqTechOwn)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, respTechOwn.StatusCode)

	var techBody map[string]interface{}
	require.NoError(t, json.NewDecoder(respTechOwn.Body).Decode(&techBody))
	techRows := techBody["data"].(map[string]interface{})["rows"].([]interface{})
	require.Len(t, techRows, 1)
	assert.Equal(t, cctvA.String(), techRows[0].(map[string]interface{})["cctv_id"])

	// 4. Teknisi dilarang mengakses endpoint diagnostik admin (/api/streams/diagnostics -> 403).
	expectProtectedUserCheck(mock, techID, 1, false)
	reqTechDiag := httptest.NewRequest("GET", "/api/streams/diagnostics", nil)
	reqTechDiag.AddCookie(&http.Cookie{Name: utils.SessionCookieName, Value: techToken})
	respTechDiag, err := app.Test(reqTechDiag)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, respTechDiag.StatusCode)

	// 5. Admin tanpa X-Site-ID berhasil mengakses /api/streams/diagnostics seluruh site.
	expectProtectedUserCheck(mock, adminID, 1, false)
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(probeCCTVRow(cctvA, "10.20.105.62", false, nil, nil, nil, models.StreamStateIdle))

	reqAdminDiag := httptest.NewRequest("GET", "/api/streams/diagnostics", nil)
	reqAdminDiag.AddCookie(&http.Cookie{Name: utils.SessionCookieName, Value: adminToken})
	respAdminDiag, err := app.Test(reqAdminDiag)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, respAdminDiag.StatusCode)

	var diagBody map[string]interface{}
	require.NoError(t, json.NewDecoder(respAdminDiag.Body).Decode(&diagBody))
	diagData := diagBody["data"].(map[string]interface{})
	obs := diagData["observability"].(map[string]interface{})
	assert.Equal(t, "Asia/Makassar", obs["timezone"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestOpsRegression_ProxySpoofingFullStack memverifikasi bahwa header X-Forwarded-For
// dan X-Real-IP palsu diabaikan pada audit log & peer detection ketika TRUSTED_PROXY_IPS
// kosong/tidak cocok, dan hanya dipercaya bila peer termasuk trusted proxy yang valid.
func TestOpsRegression_ProxySpoofingFullStack(t *testing.T) {
	t.Run("untrusted peer ignores spoofed XFF in login audit log", func(t *testing.T) {
		_, mock := setupMockGORM(t)

		capturedCh := make(chan models.AuditLog, 1)
		cleanup := utils.SetAuditHookForTest(func(entry models.AuditLog) {
			capturedCh <- entry
		})
		defer cleanup()

		mock.ExpectQuery(`^SELECT \* FROM "users"`).
			WillReturnError(gorm.ErrRecordNotFound)
		mock.ExpectExec(`^INSERT INTO "audit_logs"`).
			WillReturnResult(sqlmock.NewResult(1, 1))

		app := fiber.New(config.NewFiberConfig(config.ParseTrustedProxies("")))
		app.Post("/api/auth/login", middleware.AuthLimiter(), Login)

		payload := `{"username":"spoof_probe_untrusted","password":"WrongPassword123!"}`
		req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(fiber.HeaderXForwardedFor, "198.51.100.77")
		req.Header.Set("X-Real-IP", "198.51.100.77")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)

		select {
		case entry := <-capturedCh:
			assert.Equal(t, "LOGIN_FAILED", entry.Action)
			assert.NotEqual(t, "198.51.100.77", entry.IPAddress, "header XFF palsu tidak boleh masuk audit log")
			assert.Equal(t, "0.0.0.0", entry.IPAddress)
		case <-time.After(1 * time.Second):
			t.Fatal("timeout menunggu hook AuditLog")
		}

		assert.Eventually(t, func() bool {
			return mock.ExpectationsWereMet() == nil
		}, 1*time.Second, 20*time.Millisecond)
	})

	t.Run("trusted proxy honors valid client IP and skips invalid XFF token", func(t *testing.T) {
		_, mock := setupMockGORM(t)

		capturedCh := make(chan models.AuditLog, 1)
		cleanup := utils.SetAuditHookForTest(func(entry models.AuditLog) {
			capturedCh <- entry
		})
		defer cleanup()

		mock.ExpectQuery(`^SELECT \* FROM "users"`).
			WillReturnError(gorm.ErrRecordNotFound)
		mock.ExpectExec(`^INSERT INTO "audit_logs"`).
			WillReturnResult(sqlmock.NewResult(1, 1))

		trusted := config.ParseTrustedProxies("0.0.0.0/32, invalid-proxy-entry")
		require.Equal(t, []string{"0.0.0.0/32"}, trusted)

		app := fiber.New(config.NewFiberConfig(trusted))
		app.Post("/api/auth/login", middleware.AuthLimiter(), Login)

		payload := `{"username":"spoof_probe_trusted","password":"WrongPassword123!"}`
		req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(fiber.HeaderXForwardedFor, "not-an-ip, 203.0.113.55")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)

		select {
		case entry := <-capturedCh:
			assert.Equal(t, "LOGIN_FAILED", entry.Action)
			assert.Equal(t, "203.0.113.55", entry.IPAddress)
		case <-time.After(1 * time.Second):
			t.Fatal("timeout menunggu hook AuditLog")
		}

		assert.Eventually(t, func() bool {
			return mock.ExpectationsWereMet() == nil
		}, 1*time.Second, 20*time.Millisecond)
	})
}

type witaMidnightMatcher struct {
	loc      *time.Location
	captured *time.Time
}

func (m witaMidnightMatcher) Match(v driver.Value) bool {
	ts, ok := v.(time.Time)
	if !ok {
		return false
	}
	local := ts.In(m.loc)
	if local.Hour() != 0 || local.Minute() != 0 || local.Second() != 0 || local.Nanosecond() != 0 {
		return false
	}
	if m.captured != nil {
		*m.captured = ts
	}
	return true
}

// TestOpsRegression_TimezoneWITAFullRequest memverifikasi /api/cctv-logs mengirim
// batas hari tepat [00:00 WITA, +24h) dalam UTC ke SQL dan melaporkan Asia/Makassar.
func TestOpsRegression_TimezoneWITAFullRequest(t *testing.T) {
	t.Setenv("JWT_SECRET", "sigap_ops_regression_tz_secret_2026")
	t.Setenv("APP_TIMEZONE", "Asia/Makassar")

	_, mock := setupMockGORM(t)
	adminID := uuid.NewString()
	cctvID := uuid.New()
	adminToken, err := utils.GenerateToken(adminID, "admin", 1)
	require.NoError(t, err)

	now := time.Now()
	loc := mustWITALocation(t)
	var onlineStart, onlineEnd, offlineStart, offlineEnd time.Time

	expectProtectedUserCheck(mock, adminID, 1, false)
	mock.ExpectQuery(`^SELECT count\(\*\) FROM "cctv_stream_logs"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`^SELECT \* FROM "cctv_stream_logs"`).
		WithArgs(20).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "created_at", "cctv_id", "event_type", "previous_state",
			"current_state", "trigger_source", "network_type", "latency_ms", "error_detail", "message",
		}).AddRow(uuid.New(), now, cctvID, "online", "offline", "live", "verify", "metro-e", nil, "", "Online"))
	mock.ExpectQuery(`^SELECT id, nama, kode_cctv, lokasi_area, ip_camera, status FROM "cctvs"`).
		WithArgs(cctvID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "nama", "kode_cctv", "lokasi_area", "ip_camera", "status",
		}).AddRow(cctvID, "Kamera WITA", "WITA-01", "KIPP", "10.20.105.5", "aktif"))
	mock.ExpectQuery(`^SELECT count\(\*\) FROM "cctv_stream_logs" WHERE cctv_stream_logs\.created_at >= \$1 AND cctv_stream_logs\.created_at < \$2 AND event_type = \$3`).
		WithArgs(witaMidnightMatcher{loc: loc, captured: &onlineStart}, witaMidnightMatcher{loc: loc, captured: &onlineEnd}, "online").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`^SELECT count\(\*\) FROM "cctv_stream_logs" WHERE cctv_stream_logs\.created_at >= \$1 AND cctv_stream_logs\.created_at < \$2 AND event_type = \$3`).
		WithArgs(witaMidnightMatcher{loc: loc, captured: &offlineStart}, witaMidnightMatcher{loc: loc, captured: &offlineEnd}, "offline").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	app := fiber.New(config.NewFiberConfig(nil))
	app.Get("/api/cctv-logs", middleware.Protected(), middleware.SiteContext(), GetCCTVStreamLogs)

	req := httptest.NewRequest("GET", "/api/cctv-logs", nil)
	req.AddCookie(&http.Cookie{Name: utils.SessionCookieName, Value: adminToken})
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	summary := body["data"].(map[string]interface{})["summary"].(map[string]interface{})
	assert.Equal(t, "Asia/Makassar", summary["timezone"])
	assert.Equal(t, 24*time.Hour, onlineEnd.Sub(onlineStart))
	assert.Equal(t, 24*time.Hour, offlineEnd.Sub(offlineStart))
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestOpsRegression_RetentionConfigAndCleanup memverifikasi integrasi konfigurasi
// CCTV_STREAM_LOG_RETENTION_DAYS ke eksekusi cleanup bertahap dan idempoten.
func TestOpsRegression_RetentionConfigAndCleanup(t *testing.T) {
	t.Setenv("CCTV_STREAM_LOG_RETENTION_DAYS", "30")
	days := config.CCTVStreamLogRetentionDays()
	require.Equal(t, 30, days)

	db, mock := setupMockGORM(t)
	mock.ExpectExec(deleteCCTVStreamLogsBatchRegex).
		WithArgs(sqlmock.AnyArg(), cctvStreamLogCleanupBatchSize).
		WillReturnResult(sqlmock.NewResult(0, 1000))
	mock.ExpectExec(deleteCCTVStreamLogsBatchRegex).
		WithArgs(sqlmock.AnyArg(), cctvStreamLogCleanupBatchSize).
		WillReturnResult(sqlmock.NewResult(0, 42))
	mock.ExpectExec(deleteCCTVStreamLogsBatchRegex).
		WithArgs(sqlmock.AnyArg(), cctvStreamLogCleanupBatchSize).
		WillReturnResult(sqlmock.NewResult(0, 0))

	deleted, err := CleanupCCTVStreamLogs(db, days)
	require.NoError(t, err)
	assert.Equal(t, int64(1042), deleted)

	deletedAgain, err := CleanupCCTVStreamLogs(db, days)
	require.NoError(t, err)
	assert.Zero(t, deletedAgain)

	// Nilai env tidak valid wajib fallback aman ke default 90 hari agar cleanup
	// tidak menghapus seluruh histori log secara tidak sengaja.
	t.Setenv("CCTV_STREAM_LOG_RETENTION_DAYS", "-5")
	assert.Equal(t, config.DefaultCCTVStreamLogRetentionDays, config.CCTVStreamLogRetentionDays())
	assert.NoError(t, mock.ExpectationsWereMet())
}
