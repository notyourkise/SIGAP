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

// expectCCTVStreamLogForSite menyiapkan lima query yang dijalankan
// GetCCTVStreamLogs untuk satu site: count utama, list log, preload CCTV,
// ringkasan online, dan ringkasan offline. Seluruh query wajib memakai
// parameter siteID agar isolasi site terverifikasi end-to-end.
func expectCCTVStreamLogForSite(mock sqlmock.Sqlmock, siteID string, cctvID uuid.UUID, nama, kode string) {
	now := time.Now()

	mock.ExpectQuery(`^SELECT count\(\*\) FROM "cctv_stream_logs" WHERE .*EXISTS .*cctvs\.site_id = \$1`).
		WithArgs(siteID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery(`^SELECT \* FROM "cctv_stream_logs" WHERE .*EXISTS .*cctvs\.site_id = \$1`).
		WithArgs(siteID, 20).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "created_at", "cctv_id", "event_type", "previous_state",
			"current_state", "trigger_source", "network_type", "latency_ms", "error_detail", "message",
		}).AddRow(uuid.New(), now, cctvID, "online", "offline", "live", "verify", "gsm", nil, "", "Online"))

	mock.ExpectQuery(`^SELECT id, nama, kode_cctv, lokasi_area, ip_camera, status FROM "cctvs" WHERE "cctvs"\."id" = \$1`).
		WithArgs(cctvID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "nama", "kode_cctv", "lokasi_area", "ip_camera", "status",
		}).AddRow(cctvID, nama, kode, "Area Uji", "10.0.0.1", "aktif"))

	mock.ExpectQuery(`^SELECT count\(\*\) FROM "cctv_stream_logs" WHERE .*EXISTS .*cctvs\.site_id = \$1.* AND \(created_at >= \$2 AND event_type = \$3\)`).
		WithArgs(siteID, sqlmock.AnyArg(), "online").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery(`^SELECT count\(\*\) FROM "cctv_stream_logs" WHERE .*EXISTS .*cctvs\.site_id = \$1.* AND \(created_at >= \$2 AND event_type = \$3\)`).
		WithArgs(siteID, sqlmock.AnyArg(), "offline").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
}

// TestGetCCTVStreamLogs_TwoTechniciansSeeOnlyOwnSite adalah regression test P0
// #141: teknisi site A dan site B mengakses endpoint yang sama dan hanya boleh
// menerima log milik site masing-masing.
func TestGetCCTVStreamLogs_TwoTechniciansSeeOnlyOwnSite(t *testing.T) {
	_, mock := setupMockGORM(t)

	siteA := uuid.New()
	siteB := uuid.New()
	cctvA := uuid.New()
	cctvB := uuid.New()

	expectCCTVStreamLogForSite(mock, siteA.String(), cctvA, "Kamera Site A", "SITE-A-01")
	expectCCTVStreamLogForSite(mock, siteB.String(), cctvB, "Kamera Site B", "SITE-B-01")

	app := newTestApp()
	app.Get("/cctv-logs/a", withUserSiteLocals(uuid.New().String(), "teknisi", siteA.String()), GetCCTVStreamLogs)
	app.Get("/cctv-logs/b", withUserSiteLocals(uuid.New().String(), "teknisi", siteB.String()), GetCCTVStreamLogs)

	assertOnlyCCTV := func(path string, wantCCTVID uuid.UUID, wantName, wantCode string) {
		t.Helper()

		resp, err := app.Test(httptest.NewRequest("GET", path, nil))
		require.NoError(t, err)
		require.Equal(t, fiber.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		require.Equal(t, "success", result["status"])

		data, ok := result["data"].(map[string]interface{})
		require.True(t, ok)
		rows, ok := data["rows"].([]interface{})
		require.True(t, ok)
		require.Len(t, rows, 1)

		row, ok := rows[0].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, wantCCTVID.String(), row["cctv_id"])

		cctv, ok := row["cctv"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, wantName, cctv["nama"])
		assert.Equal(t, wantCode, cctv["kode_cctv"])
	}

	assertOnlyCCTV("/cctv-logs/a", cctvA, "Kamera Site A", "SITE-A-01")
	assertOnlyCCTV("/cctv-logs/b", cctvB, "Kamera Site B", "SITE-B-01")
	assert.NoError(t, mock.ExpectationsWereMet())
}
