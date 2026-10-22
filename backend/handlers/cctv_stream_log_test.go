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

func TestGetCCTVStreamLogs_Success(t *testing.T) {
	_, mock := setupMockGORM(t)

	logID := uuid.New()
	cctvID := uuid.New()
	now := time.Now()

	// 1. Count query
	mock.ExpectQuery(`^SELECT count\(\*\) FROM "cctv_stream_logs"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	// 2. Main data query
	rows := sqlmock.NewRows([]string{
		"id", "created_at", "cctv_id", "event_type", "previous_state",
		"current_state", "trigger_source", "network_type", "latency_ms", "error_detail", "message",
	}).AddRow(logID, now, cctvID, "online", "offline", "live", "verify", "gsm", nil, "", "Verifikasi berhasil")

	mock.ExpectQuery(`^SELECT \* FROM "cctv_stream_logs"`).
		WillReturnRows(rows)

	// Preload CCTV query
	cctvRows := sqlmock.NewRows([]string{
		"id", "nama", "kode_cctv", "lokasi_area", "ip_camera", "status",
	}).AddRow(cctvID, "Rest Area 1", "THD-01", "Rest Area", "10.20.105.1", "aktif")

	mock.ExpectQuery(`^SELECT id, nama, kode_cctv, lokasi_area, ip_camera, status FROM "cctvs" WHERE "cctvs"\."id" = \$1`).
		WithArgs(cctvID).
		WillReturnRows(cctvRows)

	// 3. Summary count queries (online and offline)
	mock.ExpectQuery(`^SELECT count\(\*\) FROM "cctv_stream_logs" WHERE created_at >= \$1 AND event_type = \$2`).
		WithArgs(sqlmock.AnyArg(), "online").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery(`^SELECT count\(\*\) FROM "cctv_stream_logs" WHERE created_at >= \$1 AND event_type = \$2`).
		WithArgs(sqlmock.AnyArg(), "offline").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	app := newTestApp()
	app.Get("/cctv-logs", GetCCTVStreamLogs)

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
