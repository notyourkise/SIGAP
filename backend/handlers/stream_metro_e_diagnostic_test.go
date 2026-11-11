package handlers

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"sigap-backend/models"
	"sigap-backend/utils"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func captureZapLogs(t *testing.T) *observer.ObservedLogs {
	t.Helper()
	core, logs := observer.New(zap.DebugLevel)
	prev := utils.Log
	utils.Log = zap.New(core).Sugar()
	t.Cleanup(func() {
		utils.Log = prev
	})
	return logs
}

func injectMetroEProbeResult(t *testing.T, fn func(host string) bool) {
	t.Helper()
	prev := metroEReachabilityProbeFunc
	metroEReachabilityProbeFunc = fn
	resetMetroEDiagnosticStateForTest()
	t.Cleanup(func() {
		metroEReachabilityProbeFunc = prev
		resetMetroEDiagnosticStateForTest()
	})
}

func TestCheckMetroEReachabilityAtStartup_UnreachableTriggersPassiveFallbackAndWarning(t *testing.T) {
	t.Setenv("CCTV_METRO_E_DIRECT_PROBE", "true")
	t.Setenv("CCTV_METRO_E_PROBE_TARGET", "")

	db, mock := setupMockGORM(t)
	logs := captureZapLogs(t)
	injectMetroEProbeResult(t, func(host string) bool {
		return false
	})

	mock.ExpectQuery(`SELECT "ip_camera" FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{"ip_camera"}).AddRow("10.20.105.62"))

	snap := CheckMetroEReachabilityAtStartup(db)
	assert.True(t, snap.ConfiguredDirectProbe)
	assert.False(t, snap.EffectiveDirectProbe)
	assert.True(t, snap.PassiveFallbackActive)
	assert.Equal(t, metroEModePassiveFallback, snap.Mode)
	assert.Equal(t, []string{"10.20.105.62"}, snap.ProbeTargetsTested)
	assert.Equal(t, 0, snap.ReachableTargets)
	assert.Equal(t, 1, snap.UnreachableTargets)
	assert.False(t, effectiveMetroEDirectProbe())

	warnLogs := logs.FilterLevelExact(zap.WarnLevel).All()
	require.NotEmpty(t, warnLogs, "startup fallback harus mencatat log warning eksplisit")
	assert.Contains(t, warnLogs[0].Message, "10.20.105.x")

	// Setelah fallback aktif, siklus probe terhadap kamera 10.20.105.x memakai
	// mode pasif MediaMTX (idle), BUKAN offline palsu.
	camID := uuid.New()
	path := utils.BuildStreamPath(camID.String())
	client, closeSrv := startMTXStateServer(t, nil, []utils.PathConfig{{Name: path, Source: "rtsp://nvr/channel"}})
	defer closeSrv()
	injectStreamClient(t, client)

	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WithArgs(true).
		WillReturnRows(probeCCTVRow(camID, "10.20.105.62", false, nil, nil, nil, models.StreamStateUnknown))
	mock.ExpectExec(`^UPDATE "cctvs" SET "stream_checked_at"=\$1,"stream_live"=\$2,"stream_state"=\$3,"updated_at"=\$4 WHERE id = \$5 AND "cctvs"."deleted_at" IS NULL$`).
		WithArgs(sqlmock.AnyArg(), false, models.StreamStateIdle, sqlmock.AnyArg(), camID.String()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	runRTSPProbeCycle(db)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckMetroEReachabilityAtStartup_ReachableKeepsDirectProbe(t *testing.T) {
	t.Setenv("CCTV_METRO_E_DIRECT_PROBE", "true")
	t.Setenv("CCTV_METRO_E_PROBE_TARGET", "10.20.105.11")

	logs := captureZapLogs(t)
	injectMetroEProbeResult(t, func(host string) bool {
		return host == "10.20.105.11"
	})

	snap := CheckMetroEReachabilityAtStartup(nil)
	assert.True(t, snap.ConfiguredDirectProbe)
	assert.True(t, snap.EffectiveDirectProbe)
	assert.False(t, snap.PassiveFallbackActive)
	assert.Equal(t, metroEModeDirectProbe, snap.Mode)
	assert.Equal(t, 1, snap.ReachableTargets)
	assert.Equal(t, 0, snap.UnreachableTargets)
	assert.True(t, effectiveMetroEDirectProbe())
	assert.Empty(t, logs.FilterLevelExact(zap.WarnLevel).All())
}

func TestCheckMetroEReachabilityAtStartup_DisabledByConfig(t *testing.T) {
	t.Setenv("CCTV_METRO_E_DIRECT_PROBE", "false")
	injectMetroEProbeResult(t, func(host string) bool {
		t.Fatal("probe jaringan tidak boleh dipanggil ketika CCTV_METRO_E_DIRECT_PROBE=false")
		return false
	})

	snap := CheckMetroEReachabilityAtStartup(nil)
	assert.False(t, snap.ConfiguredDirectProbe)
	assert.False(t, snap.EffectiveDirectProbe)
	assert.False(t, snap.PassiveFallbackActive)
	assert.Equal(t, metroEModePassiveConfig, snap.Mode)
	assert.False(t, effectiveMetroEDirectProbe())
}

func TestGetStreamDiagnostics_AdminAndNonAdmin(t *testing.T) {
	t.Setenv("CCTV_METRO_E_DIRECT_PROBE", "true")
	t.Setenv("CCTV_METRO_E_PROBE_TARGET", "10.20.105.62")

	db, mock := setupMockGORM(t)
	injectMetroEProbeResult(t, func(host string) bool { return false })
	CheckMetroEReachabilityAtStartup(nil)

	camMetroID := uuid.New()
	camGSMID := uuid.New()
	checkedAt := time.Now().Add(-5 * time.Minute)
	client, closeSrv := startMTXStateServer(t, []utils.PathInfo{{Name: utils.BuildStreamPath(camGSMID.String()), Ready: true}}, nil)
	defer closeSrv()
	injectStreamClient(t, client)

	rows := sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"nama", "kode_cctv", "lokasi_area", "ip_camera", "ip_nvr",
		"kategori", "payload", "health_score", "channel_port", "rtsp_password",
		"stream_enabled", "stream_live", "stream_checked_at",
		"ping_latency_ms", "stream_last_active_at", "stream_state",
	}).
		// Kamera Metro-E sempat berstatus offline lama tanpa verifikasi segar:
		// saat passive fallback aktif, diagnostik tidak boleh melabelinya offline palsu.
		AddRow(camMetroID, time.Now(), time.Now(), nil, "Kamera Metro", "MTR-01", "Gate", "10.20.105.62", "10.20.105.10", "Aman", "payload", 100, 1, "", true, false, &checkedAt, nil, nil, models.StreamStateOffline).
		AddRow(camGSMID, time.Now(), time.Now(), nil, "Kamera GSM", "GSM-01", "Toll", "10.99.99.10", "", "Aman", "payload", 100, 2, "", true, true, &checkedAt, nil, &checkedAt, models.StreamStateLive)

	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).WillReturnRows(rows)

	app := newTestApp()
	app.Get("/streams/diagnostics", withUserSiteLocals(uuid.New().String(), "admin", ""), GetStreamDiagnostics)
	app.Get("/streams/diagnostics-forbidden", withUserSiteLocals(uuid.New().String(), "teknisi", ""), GetStreamDiagnostics)

	respForbidden, err := app.Test(httptest.NewRequest("GET", "/streams/diagnostics-forbidden", nil))
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, respForbidden.StatusCode)

	resp, err := app.Test(httptest.NewRequest("GET", "/streams/diagnostics", nil))
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "success", body["status"])

	data := body["data"].(map[string]interface{})
	metro := data["metro_e"].(map[string]interface{})
	assert.Equal(t, true, metro["passive_fallback_active"])
	assert.Equal(t, metroEModePassiveFallback, metro["mode"])
	assert.Equal(t, false, metro["effective_direct_probe"])
	assert.Equal(t, float64(1), metro["camera_count"])

	summary := data["summary"].(map[string]interface{})
	assert.Equal(t, true, summary["false_offline_protected"])
	assert.Equal(t, float64(1), summary["false_offline_suppressed"])
	assert.Equal(t, float64(0), summary["offline"])
	assert.Equal(t, float64(1), summary["online"])
	assert.Equal(t, float64(1), summary["unknown"])

	mediamtx := data["mediamtx"].(map[string]interface{})
	assert.Equal(t, true, mediamtx["reachable"])
	assert.Equal(t, float64(1), mediamtx["operational_paths"])

	_ = db
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetLiveStatus_IncludesMetroEDiagnosticsAndSuppressesFalseOffline(t *testing.T) {
	t.Setenv("CCTV_METRO_E_DIRECT_PROBE", "true")
	t.Setenv("CCTV_METRO_E_PROBE_TARGET", "10.20.105.62")

	_, mock := setupMockGORM(t)
	injectMetroEProbeResult(t, func(host string) bool { return false })
	CheckMetroEReachabilityAtStartup(nil)

	camID := uuid.New()
	checkedAt := time.Now().Add(-3 * time.Minute)
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(probeCCTVRow(camID, "10.20.105.62", false, checkedAt, nil, nil, models.StreamStateOffline))

	app := newTestApp()
	app.Get("/streams/live-status", GetLiveStatus)

	resp, err := app.Test(httptest.NewRequest("GET", "/streams/live-status", nil))
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	summary := body["summary"].(map[string]interface{})
	assert.Equal(t, float64(0), summary["offline"], "offline palsu Metro-E tidak boleh terhitung saat fallback pasif aktif")
	assert.Equal(t, float64(1), summary["unchecked"])

	diag := body["diagnostics"].(map[string]interface{})
	metro := diag["metro_e"].(map[string]interface{})
	assert.Equal(t, true, metro["passive_fallback_active"])
	assert.Equal(t, metroEModePassiveFallback, metro["mode"])
	assert.NoError(t, mock.ExpectationsWereMet())
}
