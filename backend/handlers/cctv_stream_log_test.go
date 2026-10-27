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
	mock.ExpectQuery(`^SELECT count\(\*\) FROM "cctv_stream_logs" WHERE created_at >= \$1 AND event_type = \$2`).
		WithArgs(sqlmock.AnyArg(), "online").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery(`^SELECT count\(\*\) FROM "cctv_stream_logs" WHERE created_at >= \$1 AND event_type = \$2`).
		WithArgs(sqlmock.AnyArg(), "offline").
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

	mock.ExpectQuery(`^SELECT count\(\*\) FROM "cctv_stream_logs" WHERE .*EXISTS .*cctvs\.site_id = \$1.* AND \(created_at >= \$2 AND event_type = \$3\)`).
		WithArgs(siteID, sqlmock.AnyArg(), "online").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery(`^SELECT count\(\*\) FROM "cctv_stream_logs" WHERE .*EXISTS .*cctvs\.site_id = \$1.* AND \(created_at >= \$2 AND event_type = \$3\)`).
		WithArgs(siteID, sqlmock.AnyArg(), "offline").
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

	mock.ExpectQuery(`^SELECT count\(\*\) FROM "cctv_stream_logs" WHERE .*EXISTS .*cctvs\.site_id = \$1.* AND \(created_at >= \$2 AND event_type = \$3\)`).
		WithArgs(siteID, sqlmock.AnyArg(), "online").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectQuery(`^SELECT count\(\*\) FROM "cctv_stream_logs" WHERE .*EXISTS .*cctvs\.site_id = \$1.* AND \(created_at >= \$2 AND event_type = \$3\)`).
		WithArgs(siteID, sqlmock.AnyArg(), "offline").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	app := newTestApp()
	app.Get("/cctv-logs", withUserSiteLocals(uuid.New().String(), "admin", siteID), GetCCTVStreamLogs)

	resp, err := app.Test(httptest.NewRequest("GET", "/cctv-logs", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}
