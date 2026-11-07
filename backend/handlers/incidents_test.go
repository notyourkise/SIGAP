package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
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

func TestIncidentEndpointsRequireExplicitSite(t *testing.T) {
	app := fiber.New()
	app.Get("/incidents", GetIncidents)
	app.Post("/incidents/:id/merge", MergeIncidentsHandler)
	app.Post("/incidents/:id/escalate-replacement", EscalateIncidentReplacement)
	app.Post("/incidents/planned-outages", CreatePlannedOutageHandler)

	for _, tc := range []struct {
		method string
		path   string
	}{
		{"GET", "/incidents"},
		{"POST", "/incidents/" + uuid.NewString() + "/merge"},
		{"POST", "/incidents/" + uuid.NewString() + "/escalate-replacement"},
		{"POST", "/incidents/planned-outages"},
	} {
		resp, err := app.Test(httptest.NewRequest(tc.method, tc.path, nil))
		require.NoError(t, err)
		assert.Equal(t, 400, resp.StatusCode, "endpoint %s %s must require explicit site", tc.method, tc.path)
	}
}

func TestStreamMonitorHeartbeatAuthAndPayload(t *testing.T) {
	app := fiber.New()
	app.Get("/heartbeat-key", func(c *fiber.Ctx) error {
		c.Locals("auth_method", "api_key")
		return GetStreamMonitorHeartbeat(c)
	})
	app.Get("/heartbeat-admin", func(c *fiber.Ctx) error {
		c.Locals("role", "admin")
		return GetStreamMonitorHeartbeat(c)
	})
	app.Get("/heartbeat-tech", func(c *fiber.Ctx) error {
		c.Locals("role", "teknisi")
		return GetStreamMonitorHeartbeat(c)
	})

	respKey, err := app.Test(httptest.NewRequest("GET", "/heartbeat-key", nil))
	require.NoError(t, err)
	assert.Equal(t, 200, respKey.StatusCode)

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(respKey.Body).Decode(&body))
	data := body["data"].(map[string]interface{})
	assert.Contains(t, data, "monitor_status")
	assert.Contains(t, data, "session_gate")

	respAdmin, err := app.Test(httptest.NewRequest("GET", "/heartbeat-admin", nil))
	require.NoError(t, err)
	assert.Equal(t, 200, respAdmin.StatusCode)

	respTech, err := app.Test(httptest.NewRequest("GET", "/heartbeat-tech", nil))
	require.NoError(t, err)
	assert.Equal(t, 403, respTech.StatusCode)
}

func TestVerifyCCTVStreamDetectsStalledZeroByteDeltaWhenRequired(t *testing.T) {
	_, mock := setupMockGORM(t)
	cctvID := uuid.New()
	mock.ExpectExec(`UPDATE "cctvs" SET`).WillReturnResult(sqlmock.NewResult(0, 1))
	expectNoPreviousCCTVStreamLog(mock, cctvID)
	expectCCTVStreamLogInsert(mock)

	// Server mengembalikan available=true dan tracks=["H264"], tetapi bytesReceived BEKU di 4096 (0 delta).
	var bytesVal atomic.Int64
	bytesVal.Store(4096)

	mux := http.NewServeMux()
	mux.HandleFunc("/v3/paths/get/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"name":          r.URL.Path[len("/v3/paths/get/"):],
			"available":     true,
			"ready":         true,
			"tracks":        []string{"H264"},
			"bytesReceived": bytesVal.Load(),
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	oldClient := streamClient
	streamClient = &utils.MediaMTXClient{BaseURL: srv.URL, HLSURL: srv.URL, HTTPClient: srv.Client()}
	defer func() { streamClient = oldClient }()

	oldByteWin, oldReqBytes := streamVerifyByteWindow, streamVerifyRequireBytes
	streamVerifyByteWindow, streamVerifyRequireBytes = 15*time.Millisecond, true
	defer func() { streamVerifyByteWindow, streamVerifyRequireBytes = oldByteWin, oldReqBytes }()

	cctv := models.CCTV{
		Base:                 models.Base{ID: cctvID},
		KodeCCTV:             "STALL-01",
		Nama:                 "Kamera Beku",
		StreamEnabled:        true,
		MonitoringEnabled:    true,
		StreamVerifiedState:  models.StreamStateUnknown,
		StreamVerifyFailures: 1,
	}

	res := verifyCCTVStream(configDBForTest(t), cctv)
	assert.Equal(t, models.StreamStateOffline, res.State, "frozen RTP stream with 0 byte delta on 2nd failure must be marked offline")
	assert.Equal(t, 2, res.Failures)
	assert.Contains(t, res.Error, "beku")
	assert.NoError(t, mock.ExpectationsWereMet())

	// Sekarang uji ketika bytesReceived bertambah aktif -> terverifikasi live!
	_, mock2 := setupMockGORM(t)
	mock2.ExpectExec(`UPDATE "cctvs" SET`).WillReturnResult(sqlmock.NewResult(0, 1))
	expectNoPreviousCCTVStreamLog(mock2, cctvID)
	expectCCTVStreamLogInsert(mock2)

	var callCount atomic.Int64
	muxGrow := http.NewServeMux()
	muxGrow.HandleFunc("/v3/paths/get/", func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"name":          r.URL.Path[len("/v3/paths/get/"):],
			"available":     true,
			"ready":         true,
			"tracks":        []string{"H264"},
			"bytesReceived": n * 8192,
		})
	})
	srvGrow := httptest.NewServer(muxGrow)
	defer srvGrow.Close()
	streamClient = &utils.MediaMTXClient{BaseURL: srvGrow.URL, HLSURL: srvGrow.URL, HTTPClient: srvGrow.Client()}

	resLive := verifyCCTVStream(configDBForTest(t), cctv)
	assert.Equal(t, models.StreamStateLive, resLive.State)
	assert.Greater(t, resLive.ByteDelta, int64(0))
	assert.NoError(t, mock2.ExpectationsWereMet())
}

func TestPlannedOutageHandlerValidationRejectsInvalidPayload(t *testing.T) {
	app := fiber.New()
	siteID := uuid.New()
	adminID := uuid.New()
	app.Post("/planned", func(c *fiber.Ctx) error {
		c.Locals("site_id", siteID.String())
		c.Locals("user_id", adminID.String())
		return CreatePlannedOutageHandler(c)
	})

	payload, _ := json.Marshal(map[string]interface{}{
		"reason":    "Tes",
		"starts_at": "bukan-tanggal",
		"ends_at":   "bukan-tanggal",
		"cctv_ids":  []string{uuid.NewString()},
	})
	req := httptest.NewRequest("POST", "/planned", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}
