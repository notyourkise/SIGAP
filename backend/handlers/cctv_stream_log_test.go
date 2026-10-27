package handlers

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"sigap-backend/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// expectCCTVStreamLogRows menyiapkan ekspektasi query utama + preload CCTV.
func expectCCTVStreamLogRows(mock sqlmock.Sqlmock, mainQueryRegex string) {
	cctvID := uuid.New()
	now := time.Now()

	// 1. Count query
	mock.ExpectQuery(`^SELECT count\(\*\) FROM "cctv_stream_logs"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	// 2. Main data query
	rows := sqlmock.NewRows([]string{
		"id", "created_at", "cctv_id", "event_type", "previous_state",
		"current_state", "trigger_source", "network_type", "latency_ms", "error_detail", "message",
	}).AddRow(uuid.New(), now, cctvID, "online", "offline", "live", "verify", "gsm", nil, "", "Verifikasi berhasil")

	mock.ExpectQuery(mainQueryRegex).
		WillReturnRows(rows)

	// Preload CCTV query
	cctvRows := sqlmock.NewRows([]string{
		"id", "nama", "kode_cctv", "lokasi_area", "ip_camera", "status",
	}).AddRow(cctvID, "Rest Area 1", "THD-01", "Rest Area", "10.20.105.1", "aktif")

	mock.ExpectQuery(`^SELECT id, nama, kode_cctv, lokasi_area, ip_camera, status FROM "cctvs" WHERE "cctvs"\."id" = \$1`).
		WithArgs(cctvID).
		WillReturnRows(cctvRows)
}

func TestGetCCTVStreamLogs_Success(t *testing.T) {
	_, mock := setupMockGORM(t)

	expectCCTVStreamLogRows(mock, `^SELECT \* FROM "cctv_stream_logs"`)

	// Summary count queries (online and offline) tanpa filter site untuk admin.
	mock.ExpectQuery(`^SELECT count\(\*\) FROM "cctv_stream_logs" WHERE cctv_stream_logs\.created_at >= \$1 AND cctv_stream_logs\.created_at < \$2 AND event_type = \$3`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "online").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery(`^SELECT count\(\*\) FROM "cctv_stream_logs" WHERE cctv_stream_logs\.created_at >= \$1 AND cctv_stream_logs\.created_at < \$2 AND event_type = \$3`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "offline").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	app := newTestApp()
	app.Get("/cctv-logs", withUserSiteLocals(uuid.New().String(), "admin", ""), GetCCTVStreamLogs)

	resp, err := app.Test(httptest.NewRequest("GET", "/cctv-logs", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])

	data := result["data"].(map[string]interface{})
	rowsData := data["rows"].([]interface{})
	assert.Len(t, rowsData, 1)

	summary := data["summary"].(map[string]interface{})
	assert.Equal(t, float64(1), summary["today_online_events"])
	assert.Equal(t, float64(0), summary["today_offline_events"])
}

// TestGetCCTVStreamLogs_TechnicianSiteScope memastikan query count, list, dan
// ringkasan memakai EXISTS ke site CCTV milik teknisi.
func TestGetCCTVStreamLogs_TechnicianSiteScope(t *testing.T) {
	_, mock := setupMockGORM(t)
	siteID := uuid.New().String()

	// Count query dengan filter site.
	mock.ExpectQuery(`^SELECT count\(\*\) FROM "cctv_stream_logs" WHERE .*EXISTS .*cctvs\.site_id = \$1`).
		WithArgs(siteID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	cctvID := uuid.New()
	now := time.Now()
	mock.ExpectQuery(`^SELECT \* FROM "cctv_stream_logs" WHERE .*EXISTS .*cctvs\.site_id = \$1`).
		WithArgs(siteID, 20).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "created_at", "cctv_id", "event_type", "previous_state",
			"current_state", "trigger_source", "network_type", "latency_ms", "error_detail", "message",
		}).AddRow(uuid.New(), now, cctvID, "online", "offline", "live", "probe", "gsm", nil, "", "Online"))

	mock.ExpectQuery(`^SELECT id, nama, kode_cctv, lokasi_area, ip_camera, status FROM "cctvs" WHERE "cctvs"\."id" = \$1`).
		WithArgs(cctvID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "nama", "kode_cctv", "lokasi_area", "ip_camera", "status",
		}).AddRow(cctvID, "Rest Area 1", "THD-01", "Rest Area", "10.20.105.1", "aktif"))

	mock.ExpectQuery(`^SELECT count\(\*\) FROM "cctv_stream_logs" WHERE .*EXISTS .*cctvs\.site_id = \$1.* AND \(cctv_stream_logs\.created_at >= \$2 AND cctv_stream_logs\.created_at < \$3 AND event_type = \$4\)`).
		WithArgs(siteID, sqlmock.AnyArg(), sqlmock.AnyArg(), "online").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery(`^SELECT count\(\*\) FROM "cctv_stream_logs" WHERE .*EXISTS .*cctvs\.site_id = \$1.* AND \(cctv_stream_logs\.created_at >= \$2 AND cctv_stream_logs\.created_at < \$3 AND event_type = \$4\)`).
		WithArgs(siteID, sqlmock.AnyArg(), sqlmock.AnyArg(), "offline").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	app := newTestApp()
	app.Get("/cctv-logs", withUserSiteLocals(uuid.New().String(), "teknisi", siteID), GetCCTVStreamLogs)

	resp, err := app.Test(httptest.NewRequest("GET", "/cctv-logs", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetCCTVStreamLogs_NonAdminWithoutSiteForbidden memastikan teknisi tanpa
// site assignment tidak dapat melihat data apa pun (fail closed).
func TestGetCCTVStreamLogs_NonAdminWithoutSiteForbidden(t *testing.T) {
	_, mock := setupMockGORM(t)

	app := newTestApp()
	app.Get("/cctv-logs", withUserSiteLocals(uuid.New().String(), "teknisi", ""), GetCCTVStreamLogs)

	resp, err := app.Test(httptest.NewRequest("GET", "/cctv-logs", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "error", result["status"])
	assert.Equal(t, "Anda belum ditugaskan ke site mana pun", result["message"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetCCTVStreamLogs_AdminSelectedSite memastikan admin yang memilih satu
// site juga memakai filter site yang sama.
func TestGetCCTVStreamLogs_AdminSelectedSite(t *testing.T) {
	_, mock := setupMockGORM(t)
	siteID := uuid.New().String()

	mock.ExpectQuery(`^SELECT count\(\*\) FROM "cctv_stream_logs" WHERE .*EXISTS .*cctvs\.site_id = \$1`).
		WithArgs(siteID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectQuery(`^SELECT \* FROM "cctv_stream_logs" WHERE .*EXISTS .*cctvs\.site_id = \$1`).
		WithArgs(siteID, 20).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "created_at", "cctv_id", "event_type", "previous_state",
			"current_state", "trigger_source", "network_type", "latency_ms", "error_detail", "message",
		}))

	mock.ExpectQuery(`^SELECT count\(\*\) FROM "cctv_stream_logs" WHERE .*EXISTS .*cctvs\.site_id = \$1.* AND \(cctv_stream_logs\.created_at >= \$2 AND cctv_stream_logs\.created_at < \$3 AND event_type = \$4\)`).
		WithArgs(siteID, sqlmock.AnyArg(), sqlmock.AnyArg(), "online").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectQuery(`^SELECT count\(\*\) FROM "cctv_stream_logs" WHERE .*EXISTS .*cctvs\.site_id = \$1.* AND \(cctv_stream_logs\.created_at >= \$2 AND cctv_stream_logs\.created_at < \$3 AND event_type = \$4\)`).
		WithArgs(siteID, sqlmock.AnyArg(), sqlmock.AnyArg(), "offline").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	app := newTestApp()
	app.Get("/cctv-logs", withUserSiteLocals(uuid.New().String(), "admin", siteID), GetCCTVStreamLogs)

	resp, err := app.Test(httptest.NewRequest("GET", "/cctv-logs", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetCCTVStreamLogs_InvalidEventType memastikan nilai event_type di luar
// daftar resmi ditolak dengan HTTP 400 dan pesan yang jelas, tanpa query DB.
func TestGetCCTVStreamLogs_InvalidEventType(t *testing.T) {
	for _, eventType := range []string{"live", "all", "ONLINE", "onlne", "unknown;drop"} {
		t.Run(eventType, func(t *testing.T) {
			_, mock := setupMockGORM(t)

			app := newTestApp()
			app.Get("/cctv-logs", withUserSiteLocals(uuid.New().String(), "admin", ""), GetCCTVStreamLogs)

			resp, err := app.Test(httptest.NewRequest("GET", "/cctv-logs?event_type="+eventType, nil))
			assert.NoError(t, err)
			assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

			var result map[string]interface{}
			assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
			assert.Equal(t, "error", result["status"])
			assert.Contains(t, result["message"], "online, offline, standby, unknown")
			assert.NoError(t, mock.ExpectationsWereMet(), "event_type invalid tidak boleh menyentuh database")
		})
	}
}

// TestGetCCTVStreamLogs_ValidEventTypeFilter memastikan event_type resmi
// diteruskan sebagai filter ke query daftar dan ringkasan.
func TestGetCCTVStreamLogs_ValidEventTypeFilter(t *testing.T) {
	_, mock := setupMockGORM(t)
	cctvID := uuid.New()
	now := time.Now()

	mock.ExpectQuery(`^SELECT count\(\*\) FROM "cctv_stream_logs" WHERE event_type = \$1$`).
		WithArgs(models.CCTVEventUnknown).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery(`^SELECT \* FROM "cctv_stream_logs" WHERE event_type = \$1`).
		WithArgs(models.CCTVEventUnknown, 20).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "created_at", "cctv_id", "event_type", "previous_state",
			"current_state", "trigger_source", "network_type", "latency_ms", "error_detail", "message",
		}).AddRow(uuid.New(), now, cctvID, models.CCTVEventUnknown, "", "unknown", "probe", "gsm", nil, "", "Status tidak diketahui"))

	mock.ExpectQuery(`^SELECT id, nama, kode_cctv, lokasi_area, ip_camera, status FROM "cctvs" WHERE "cctvs"\."id" = \$1`).
		WithArgs(cctvID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "nama", "kode_cctv", "lokasi_area", "ip_camera", "status",
		}).AddRow(cctvID, "Rest Area 1", "THD-01", "Rest Area", "10.20.105.1", "aktif"))

	mock.ExpectQuery(`^SELECT count\(\*\) FROM "cctv_stream_logs" WHERE event_type = \$1 AND \(cctv_stream_logs\.created_at >= \$2 AND cctv_stream_logs\.created_at < \$3 AND event_type = \$4\)$`).
		WithArgs(models.CCTVEventUnknown, sqlmock.AnyArg(), sqlmock.AnyArg(), models.CCTVEventOnline).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectQuery(`^SELECT count\(\*\) FROM "cctv_stream_logs" WHERE event_type = \$1 AND \(cctv_stream_logs\.created_at >= \$2 AND cctv_stream_logs\.created_at < \$3 AND event_type = \$4\)$`).
		WithArgs(models.CCTVEventUnknown, sqlmock.AnyArg(), sqlmock.AnyArg(), models.CCTVEventOffline).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	app := newTestApp()
	app.Get("/cctv-logs", withUserSiteLocals(uuid.New().String(), "admin", ""), GetCCTVStreamLogs)

	resp, err := app.Test(httptest.NewRequest("GET", "/cctv-logs?event_type=unknown", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetCCTVStreamLogs_SummaryQueryError memastikan kegagalan query ringkasan
// tidak diam-diam ditampilkan sebagai angka 0, melainkan HTTP 500.
func TestGetCCTVStreamLogs_SummaryQueryError(t *testing.T) {
	for _, failEvent := range []string{models.CCTVEventOnline, models.CCTVEventOffline} {
		t.Run(failEvent, func(t *testing.T) {
			_, mock := setupMockGORM(t)
			expectCCTVStreamLogRows(mock, `^SELECT \* FROM "cctv_stream_logs"`)

			summaryRegex := `^SELECT count\(\*\) FROM "cctv_stream_logs" WHERE cctv_stream_logs\.created_at >= \$1 AND cctv_stream_logs\.created_at < \$2 AND event_type = \$3$`
			if failEvent == models.CCTVEventOffline {
				mock.ExpectQuery(summaryRegex).
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), models.CCTVEventOnline).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			}
			mock.ExpectQuery(summaryRegex).
				WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), failEvent).
				WillReturnError(errors.New("ringkasan gagal"))

			app := newTestApp()
			app.Get("/cctv-logs", withUserSiteLocals(uuid.New().String(), "admin", ""), GetCCTVStreamLogs)

			resp, err := app.Test(httptest.NewRequest("GET", "/cctv-logs", nil))
			assert.NoError(t, err)
			assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)

			var result map[string]interface{}
			assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
			assert.Equal(t, "error", result["status"])
			assert.Equal(t, "Gagal menghitung ringkasan log CCTV", result["message"])
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
