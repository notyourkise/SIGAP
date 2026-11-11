package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"sigap-backend/models"
	"sigap-backend/monitoring"
	"sigap-backend/utils"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStreamLogs_NeverLeakRTSPCredentials(t *testing.T) {
	t.Setenv("AES_SECRET_KEY", "0123456789abcdef0123456789abcdef")

	secretPass := "RahasiaSuperKuat_2026!"
	secretUser := "nvrops"
	rawRTSP := "rtsp://" + secretUser + ":" + secretPass + "@10.20.105.55:554/Streaming/channels/1902"
	encURL, err := utils.EncryptRTSPURL(rawRTSP)
	require.NoError(t, err)

	logs := captureZapLogs(t)

	// Simulasikan server MediaMTX membalas error yang memantulkan kembali URI berkredensial.
	srv := newMockMediaMTXServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("upstream dial failed for source " + rawRTSP))
	})
	defer srv.Close()

	client := &utils.MediaMTXClient{BaseURL: srv.URL, HTTPClient: srv.Client()}
	injectStreamClient(t, client)

	cctv := models.CCTV{
		Base:          models.Base{ID: uuid.New()},
		Nama:          "Kamera Rahasia",
		KodeCCTV:      "SEC-01",
		IPCamera:      "10.20.105.55",
		StreamEnabled: true,
		RTSPURL:       encURL,
	}

	regErr := registerStreamWithRetry(cctv, 1)
	require.Error(t, regErr)
	assert.NotContains(t, regErr.Error(), secretPass, "error return tidak boleh membocorkan password RTSP")

	// Simulasikan TriggerRTSP ke alamat RTSP berpassword yang tidak valid agar log error MediaMTXClient juga diuji.
	badClient := &utils.MediaMTXClient{RTSPURL: "rtsp://" + secretUser + ":" + secretPass + "@invalid host"}
	_ = badClient.TriggerRTSP("cam-test", 100*time.Millisecond)

	allLogs := logs.All()
	require.NotEmpty(t, allLogs)
	for _, entry := range allLogs {
		assert.NotContains(t, entry.Message, secretPass, "log %q membocorkan password RTSP", entry.Message)
	}
}

func TestStreamVerificationAndAlertUpdateMetrics(t *testing.T) {
	monitoring.GetCollector().Reset()
	resetStreamHealthState()

	db, mock := setupMockGORM(t)
	_ = captureStreamEvents(t)

	id := uuid.New()
	cctv := models.CCTV{
		Base:                models.Base{ID: id},
		Nama:                "Kamera Obs",
		KodeCCTV:            "OBS-01",
		IPCamera:            "10.99.99.99",
		StreamEnabled:       true,
		StreamVerifiedState: models.StreamStateUnknown,
	}

	mock.ExpectExec(`^UPDATE "cctvs" SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	expectNoPreviousCCTVStreamLog(mock, id)
	expectCCTVStreamLogInsert(mock)

	persistStreamVerification(db, streamVerificationResult{
		CCTV:       cctv,
		State:      models.StreamStateLive,
		VerifiedAt: time.Now(),
		Method:     streamVerifyMethodRTSP,
		Tracks:     2,
		Duration:   350 * time.Millisecond,
	})

	// Simulasikan alert down > 15 menit.
	path := utils.BuildStreamPath(id.String())
	now := time.Now()
	seedStreamDownState(path, now.Add(-1*time.Hour), now.Add(-16*time.Minute), false)
	mock.ExpectExec(`^UPDATE "cctvs" SET "health_score"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	cctv.StreamLive = false
	cctv.StreamCheckedAt = &now
	cctv.StreamState = models.StreamStateOffline
	cctv.HealthScore = 100
	evaluateStreamState(db, cctv, path, false, now)

	metricsOut := monitoring.GetCollector().Render()
	assert.Contains(t, metricsOut, `sigap_stream_verify_total{outcome="success"} 1`)
	assert.Contains(t, metricsOut, `sigap_stream_verify_success_rate 1`)
	assert.Contains(t, metricsOut, `sigap_stream_verify_duration_seconds_count 1`)
	assert.Contains(t, metricsOut, `sigap_cctv_stream_events_total{event_type="online"} 1`)
	assert.Contains(t, metricsOut, `sigap_cctv_offline_alerts_total 1`)

	assert.NoError(t, mock.ExpectationsWereMet())

	app := newTestApp()
	app.Get("/metrics", monitoring.Handler)
	resp, err := app.Test(httptest.NewRequest("GET", "/metrics", nil))
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}
