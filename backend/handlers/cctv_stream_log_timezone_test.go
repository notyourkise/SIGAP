package handlers

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustWITALocation(t *testing.T) *time.Location {
	t.Helper()
	location, err := time.LoadLocation("Asia/Makassar")
	require.NoError(t, err)
	return location
}

// TestCCTVStreamLogDayRange_WITAMidnight memastikan event pukul 00:30 WITA
// masuk ke hari yang benar dan batas atas bersifat eksklusif.
func TestCCTVStreamLogDayRange_WITAMidnight(t *testing.T) {
	location := mustWITALocation(t)

	// 2026-10-26 16:30 UTC == 2026-10-27 00:30 WITA.
	now := time.Date(2026, 10, 26, 16, 30, 0, 0, time.UTC)
	start, end := cctvStreamLogDayRange(now, location)

	assert.True(t, start.Equal(time.Date(2026, 10, 26, 16, 0, 0, 0, time.UTC)),
		"batas bawah harus tengah malam WITA dalam UTC")
	assert.True(t, end.Equal(time.Date(2026, 10, 27, 16, 0, 0, 0, time.UTC)),
		"batas atas harus tengah malam WITA berikutnya dalam UTC")
	assert.Equal(t, 24*time.Hour, end.Sub(start))

	event := time.Date(2026, 10, 27, 0, 30, 0, 0, location)
	assert.False(t, event.Before(start), "event 00:30 WITA harus masuk hari ini")
	assert.True(t, event.Before(end))

	firstMoment := time.Date(2026, 10, 27, 0, 0, 0, 0, location)
	assert.False(t, firstMoment.Before(start), "event tepat 00:00 WITA harus masuk")

	lastMoment := time.Date(2026, 10, 27, 23, 59, 59, 0, location)
	assert.True(t, lastMoment.Before(end), "event 23:59:59 WITA masih hari yang sama")

	nextMidnight := time.Date(2026, 10, 28, 0, 0, 0, 0, location)
	assert.False(t, nextMidnight.Before(end), "event tengah malam berikutnya harus di luar rentang")

	previousDay := time.Date(2026, 10, 26, 23, 59, 59, 0, location)
	assert.True(t, previousDay.Before(start), "event hari sebelumnya harus di luar rentang")
}

// TestCCTVStreamLogDayRange_ServerTimezoneIndependent memastikan hasil rentang
// sama baik instan waktu direpresentasikan dalam UTC maupun WITA. Ini menutup
// perbedaan hasil antara server UTC dan server WITA.
func TestCCTVStreamLogDayRange_ServerTimezoneIndependent(t *testing.T) {
	location := mustWITALocation(t)
	instant := time.Date(2026, 10, 27, 0, 30, 0, 0, location)

	startUTC, endUTC := cctvStreamLogDayRange(instant.UTC(), location)
	startWITA, endWITA := cctvStreamLogDayRange(instant.In(location), location)

	assert.True(t, startUTC.Equal(startWITA), "batas bawah tidak boleh bergantung timezone proses")
	assert.True(t, endUTC.Equal(endWITA), "batas atas tidak boleh bergantung timezone proses")
	assert.Equal(t, time.Date(2026, 10, 26, 16, 0, 0, 0, time.UTC), startUTC)
	assert.Equal(t, time.Date(2026, 10, 27, 16, 0, 0, 0, time.UTC), endUTC)
}

// TestGetCCTVStreamLogs_SummaryAppliesFilters memastikan query ringkasan
// memakai cctv_id, event_type, search, dan rentang hari yang sama dengan daftar.
func TestGetCCTVStreamLogs_SummaryAppliesFilters(t *testing.T) {
	t.Setenv("APP_TIMEZONE", "Asia/Makassar")
	_, mock := setupMockGORM(t)

	cctvID := uuid.New()
	now := time.Now()

	// 1. Count daftar dengan seluruh filter.
	mock.ExpectQuery(`^SELECT count\(\*\) FROM "cctv_stream_logs".*cctv_id = \$1.*event_type = \$2.*cctvs\.nama ILIKE \$3.*cctvs\.kode_cctv ILIKE \$4`).
		WithArgs(cctvID, "online", "%THD%", "%THD%").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	// 2. Daftar utama dengan filter yang sama.
	mock.ExpectQuery(`^SELECT .* FROM "cctv_stream_logs".*cctv_id = \$1.*event_type = \$2.*cctvs\.nama ILIKE \$3.*cctvs\.kode_cctv ILIKE \$4.*ORDER BY cctv_stream_logs\.created_at DESC`).
		WithArgs(cctvID, "online", "%THD%", "%THD%", 20).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "created_at", "cctv_id", "event_type", "previous_state",
			"current_state", "trigger_source", "network_type", "latency_ms", "error_detail", "message",
		}).AddRow(uuid.New(), now, cctvID, "online", "offline", "live", "probe", "gsm", nil, "", "Online"))

	// 3. Preload CCTV.
	mock.ExpectQuery(`^SELECT id, nama, kode_cctv, lokasi_area, ip_camera, status FROM "cctvs" WHERE "cctvs"\."id" = \$1`).
		WithArgs(cctvID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "nama", "kode_cctv", "lokasi_area", "ip_camera", "status",
		}).AddRow(cctvID, "Kamera THD-01", "THD-01", "Rest Area", "10.20.105.1", "aktif"))

	// 4. Ringkasan online: filter daftar + rentang hari.
	mock.ExpectQuery(`^SELECT count\(\*\) FROM "cctv_stream_logs".*cctv_id = \$1.*event_type = \$2.*cctvs\.nama ILIKE \$3.*cctvs\.kode_cctv ILIKE \$4.*cctv_stream_logs\.created_at >= \$5 AND cctv_stream_logs\.created_at < \$6 AND event_type = \$7`).
		WithArgs(cctvID, "online", "%THD%", "%THD%", sqlmock.AnyArg(), sqlmock.AnyArg(), "online").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	// 5. Ringkasan offline tetap memakai filter aktif, sehingga menghasilkan 0
	// saat pengguna memilih event_type=online.
	mock.ExpectQuery(`^SELECT count\(\*\) FROM "cctv_stream_logs".*cctv_id = \$1.*event_type = \$2.*cctvs\.nama ILIKE \$3.*cctvs\.kode_cctv ILIKE \$4.*cctv_stream_logs\.created_at >= \$5 AND cctv_stream_logs\.created_at < \$6 AND event_type = \$7`).
		WithArgs(cctvID, "online", "%THD%", "%THD%", sqlmock.AnyArg(), sqlmock.AnyArg(), "offline").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	app := newTestApp()
	app.Get("/cctv-logs", withUserSiteLocals(uuid.New().String(), "admin", ""), GetCCTVStreamLogs)

	path := "/cctv-logs?event_type=online&search=THD&cctv_id=" + cctvID.String()
	resp, err := app.Test(httptest.NewRequest("GET", path, nil))
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	require.Equal(t, "success", result["status"])

	data := result["data"].(map[string]interface{})
	summary := data["summary"].(map[string]interface{})
	assert.Equal(t, float64(1), summary["today_online_events"])
	assert.Equal(t, float64(0), summary["today_offline_events"])
	assert.Equal(t, float64(1), summary["total_events"])
	assert.Equal(t, "Asia/Makassar", summary["timezone"])
	assert.NoError(t, mock.ExpectationsWereMet())
}
