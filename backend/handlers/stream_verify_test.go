package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"sigap-backend/config"
	"sigap-backend/models"
	"sigap-backend/utils"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestEffectiveStreamStatePrefersFreshVerification(t *testing.T) {
	now := time.Now()
	fresh := now.Add(-5 * time.Minute)
	stale := now.Add(-streamVerifyFreshness - time.Minute)
	future := now.Add(5 * time.Minute)

	tests := []struct {
		name     string
		cctv     models.CCTV
		expected string
	}{
		{
			name: "verified live fresh overrides idle",
			cctv: models.CCTV{
				StreamState:         models.StreamStateIdle,
				StreamVerifiedState: models.StreamStateLive,
				StreamVerifiedAt:    &fresh,
			},
			expected: models.StreamStateLive,
		},
		{
			name: "verified offline fresh overrides idle",
			cctv: models.CCTV{
				StreamState:         models.StreamStateIdle,
				StreamVerifiedState: models.StreamStateOffline,
				StreamVerifiedAt:    &fresh,
			},
			expected: models.StreamStateOffline,
		},
		{
			name: "stale verification falls back to passive idle",
			cctv: models.CCTV{
				StreamState:         models.StreamStateIdle,
				StreamVerifiedState: models.StreamStateLive,
				StreamVerifiedAt:    &stale,
			},
			expected: models.StreamStateIdle,
		},
		{
			name: "unknown verification does not override passive",
			cctv: models.CCTV{
				StreamState:         models.StreamStateIdle,
				StreamVerifiedState: models.StreamStateUnknown,
				StreamVerifiedAt:    &fresh,
			},
			expected: models.StreamStateIdle,
		},
		{
			name: "future verification timestamp is ignored",
			cctv: models.CCTV{
				StreamState:         models.StreamStateIdle,
				StreamVerifiedState: models.StreamStateLive,
				StreamVerifiedAt:    &future,
			},
			expected: models.StreamStateIdle,
		},
		{
			name:     "empty passive state falls back to unknown",
			cctv:     models.CCTV{},
			expected: models.StreamStateUnknown,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, effectiveStreamState(tc.cctv, now))
		})
	}
}

func TestIsMediaMTXNotFound(t *testing.T) {
	assert.False(t, isMediaMTXNotFound(assert.AnError))
	assert.False(t, isMediaMTXNotFound(nil))
	assert.True(t, isMediaMTXNotFound(errString("MediaMTX GET /v3/paths/get/x mengembalikan status 404: path not found")))
}

type errString string

func (e errString) Error() string { return string(e) }

func newVerificationTestClient(t *testing.T, ready *atomic.Bool) (*utils.MediaMTXClient, func()) {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/v3/paths/get/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"name":      r.URL.Path[len("/v3/paths/get/"):],
			"available": ready.Load(),
			"ready":     ready.Load(),
			"tracks":    []string{"H264"},
		})
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// HLS request memicu sumber. Status 200 cukup; verifikasi membaca
		// kesiapan dari endpoint path API, bukan dari body HLS.
		ready.Store(true)
		w.WriteHeader(http.StatusOK)
	})

	srv := httptest.NewServer(mux)
	client := &utils.MediaMTXClient{
		BaseURL:    srv.URL,
		HLSURL:     srv.URL,
		HTTPClient: srv.Client(),
	}
	return client, srv.Close
}

func TestVerifyCCTVStreamMarksLiveWhenPathBecomesReady(t *testing.T) {
	_, mock := setupMockGORM(t)
	cctvID := uuid.New()
	mock.ExpectExec(`UPDATE "cctvs" SET`).WillReturnResult(sqlmock.NewResult(0, 1))
	expectNoPreviousCCTVStreamLog(mock, cctvID)
	expectCCTVStreamLogInsert(mock)

	var ready atomic.Bool
	client, closeServer := newVerificationTestClient(t, &ready)
	defer closeServer()
	oldClient := streamClient
	streamClient = client
	defer func() { streamClient = oldClient }()

	oldTimeout, oldPoll := streamVerifyTimeout, streamVerifyPollInterval
	streamVerifyTimeout, streamVerifyPollInterval = time.Second, 10*time.Millisecond
	defer func() { streamVerifyTimeout, streamVerifyPollInterval = oldTimeout, oldPoll }()

	cctv := models.CCTV{
		Base:                models.Base{ID: cctvID},
		KodeCCTV:            "THD-TEST",
		Nama:                "Kamera Uji",
		StreamEnabled:       true,
		StreamState:         models.StreamStateIdle,
		StreamVerifiedState: models.StreamStateUnknown,
	}

	result := verifyCCTVStream(configDBForTest(t), cctv)

	assert.Equal(t, models.StreamStateLive, result.State)
	assert.Equal(t, streamVerifyMethodHLS, result.Method)
	assert.Zero(t, result.Failures)
	assert.Equal(t, 1, result.Tracks)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestVerifyCCTVStreamMarksOfflineAfterTwoFailures(t *testing.T) {
	_, mock := setupMockGORM(t)
	cctvID := uuid.New()
	mock.ExpectExec(`UPDATE "cctvs" SET`).WillReturnResult(sqlmock.NewResult(0, 1))
	expectNoPreviousCCTVStreamLog(mock, cctvID)
	expectCCTVStreamLogInsert(mock)

	// Server selalu menjawab tersedia=false dan HLS tidak pernah membuat path siap.
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"available": false,
			"ready":     false,
			"tracks":    []string{},
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	oldClient := streamClient
	streamClient = &utils.MediaMTXClient{BaseURL: srv.URL, HLSURL: srv.URL, HTTPClient: srv.Client()}
	defer func() { streamClient = oldClient }()

	oldTimeout, oldPoll := streamVerifyTimeout, streamVerifyPollInterval
	streamVerifyTimeout, streamVerifyPollInterval = 40*time.Millisecond, 10*time.Millisecond
	defer func() { streamVerifyTimeout, streamVerifyPollInterval = oldTimeout, oldPoll }()

	cctv := models.CCTV{
		Base:                 models.Base{ID: cctvID},
		KodeCCTV:             "THD-TEST",
		Nama:                 "Kamera Uji",
		StreamEnabled:        true,
		StreamState:          models.StreamStateIdle,
		StreamVerifiedState:  models.StreamStateUnknown,
		StreamVerifyFailures: 1,
	}

	result := verifyCCTVStream(configDBForTest(t), cctv)

	assert.Equal(t, models.StreamStateOffline, result.State)
	assert.Equal(t, 2, result.Failures)
	assert.NotEmpty(t, result.Error)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestVerifyCCTVStreamDoesNotIncrementFailuresWhenMediaMTXUnreachable(t *testing.T) {
	_, mock := setupMockGORM(t)
	mock.ExpectExec(`UPDATE "cctvs" SET`).WillReturnResult(sqlmock.NewResult(0, 1))

	oldClient := streamClient
	streamClient = &utils.MediaMTXClient{
		BaseURL:    "http://127.0.0.1:1",
		HLSURL:     "http://127.0.0.1:1",
		HTTPClient: &http.Client{Timeout: 100 * time.Millisecond},
	}
	defer func() { streamClient = oldClient }()

	cctv := models.CCTV{
		Base:                 models.Base{ID: uuid.New()},
		KodeCCTV:             "THD-TEST",
		Nama:                 "Kamera Uji",
		StreamEnabled:        true,
		StreamVerifiedState:  models.StreamStateUnknown,
		StreamVerifyFailures: 1,
	}

	result := verifyCCTVStream(configDBForTest(t), cctv)

	assert.Equal(t, models.StreamStateUnknown, result.State)
	assert.Equal(t, 1, result.Failures)
	assert.Contains(t, result.Error, "MediaMTX")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestVerifyCCTVStreamReRegistersMissingPath(t *testing.T) {
	_, mock := setupMockGORM(t)
	cctvID := uuid.New()
	mock.ExpectExec(`UPDATE "cctvs" SET`).WillReturnResult(sqlmock.NewResult(0, 1))
	expectNoPreviousCCTVStreamLog(mock, cctvID)
	expectCCTVStreamLogInsert(mock)

	var registered atomic.Bool
	var addCalls atomic.Int32

	mux := http.NewServeMux()
	mux.HandleFunc("/v3/paths/get/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if !registered.Load() {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"status": "error",
				"error":  "path not found",
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"name":      r.URL.Path[len("/v3/paths/get/"):],
			"available": true,
			"ready":     true,
			"tracks":    []string{"H264"},
		})
	})
	mux.HandleFunc("/v3/config/paths/add/", func(w http.ResponseWriter, r *http.Request) {
		addCalls.Add(1)
		registered.Store(true)
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	oldClient := streamClient
	streamClient = &utils.MediaMTXClient{BaseURL: srv.URL, HLSURL: srv.URL, HTTPClient: srv.Client()}
	defer func() { streamClient = oldClient }()

	cctv := models.CCTV{
		Base:                models.Base{ID: cctvID},
		KodeCCTV:            "THD-TEST",
		Nama:                "Kamera Uji",
		ChannelPort:         19,
		RTSPPassword:        encryptedTestPassword(t, "secret123"),
		StreamEnabled:       true,
		StreamState:         models.StreamStateIdle,
		StreamVerifiedState: models.StreamStateUnknown,
	}

	result := verifyCCTVStream(configDBForTest(t), cctv)

	assert.Equal(t, models.StreamStateLive, result.State)
	assert.Equal(t, streamVerifyMethodLive, result.Method)
	assert.Zero(t, result.Failures)
	assert.Equal(t, 1, result.Tracks)
	assert.Empty(t, result.Error)
	assert.EqualValues(t, 1, addCalls.Load(), "path yang hilang harus didaftarkan ulang tepat sekali")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestVerifyCCTVStreamReportsReRegistrationFailure(t *testing.T) {
	_, mock := setupMockGORM(t)
	mock.ExpectExec(`UPDATE "cctvs" SET`).WillReturnResult(sqlmock.NewResult(0, 1))

	var addCalls atomic.Int32

	mux := http.NewServeMux()
	mux.HandleFunc("/v3/paths/get/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "error",
			"error":  "path not found",
		})
	})
	mux.HandleFunc("/v3/config/paths/add/", func(w http.ResponseWriter, r *http.Request) {
		addCalls.Add(1)
		http.Error(w, `{"error":"gagal mendaftarkan path"}`, http.StatusInternalServerError)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	oldClient := streamClient
	streamClient = &utils.MediaMTXClient{BaseURL: srv.URL, HLSURL: srv.URL, HTTPClient: srv.Client()}
	defer func() { streamClient = oldClient }()

	cctv := models.CCTV{
		Base:                 models.Base{ID: uuid.New()},
		KodeCCTV:             "THD-TEST",
		Nama:                 "Kamera Uji",
		ChannelPort:          19,
		RTSPPassword:         encryptedTestPassword(t, "secret123"),
		StreamEnabled:        true,
		StreamVerifiedState:  models.StreamStateUnknown,
		StreamVerifyFailures: 1,
	}

	result := verifyCCTVStream(configDBForTest(t), cctv)

	assert.Equal(t, models.StreamStateUnknown, result.State)
	assert.Equal(t, 1, result.Failures, "kegagalan registrasi bukan bukti kamera offline")
	assert.Contains(t, result.Error, "pendaftaran ulang gagal")
	assert.EqualValues(t, 1, addCalls.Load())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestVerifyCCTVStreamReportsPathStillMissingAfterReRegistration(t *testing.T) {
	_, mock := setupMockGORM(t)
	mock.ExpectExec(`UPDATE "cctvs" SET`).WillReturnResult(sqlmock.NewResult(0, 1))

	mux := http.NewServeMux()
	mux.HandleFunc("/v3/paths/get/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "error",
			"error":  "path not found",
		})
	})
	mux.HandleFunc("/v3/config/paths/add/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	oldClient := streamClient
	streamClient = &utils.MediaMTXClient{BaseURL: srv.URL, HLSURL: srv.URL, HTTPClient: srv.Client()}
	defer func() { streamClient = oldClient }()

	cctv := models.CCTV{
		Base:                models.Base{ID: uuid.New()},
		KodeCCTV:            "THD-TEST",
		Nama:                "Kamera Uji",
		ChannelPort:         19,
		RTSPPassword:        encryptedTestPassword(t, "secret123"),
		StreamEnabled:       true,
		StreamVerifiedState: models.StreamStateUnknown,
	}

	result := verifyCCTVStream(configDBForTest(t), cctv)

	assert.Equal(t, models.StreamStateUnknown, result.State)
	assert.Zero(t, result.Failures)
	assert.Contains(t, result.Error, "setelah pendaftaran ulang")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// configDBForTest memastikan helper sqlmock sudah di-inject ke config.DB.
func configDBForTest(t *testing.T) *gorm.DB {
	t.Helper()
	require.NotNil(t, config.DB)
	return config.DB
}

func TestConfirmSourceClosedEvidenceBasedGSMTelemetry(t *testing.T) {
	origDelay := sourceCloseConfirmDelay
	sourceCloseConfirmDelay = 0
	defer func() { sourceCloseConfirmDelay = origDelay }()

	encPass := encryptedTestPassword(t, "secret123")
	siteID := uuid.New()
	gsmCam := models.CCTV{
		Base:              models.Base{ID: uuid.New()},
		KodeCCTV:          "GSM-CLOSE-01",
		Nama:              "Kamera GSM Uji Penutupan",
		IPCamera:          "", // kosong -> diklasifikasikan sebagai GSM oleh networkType()
		IPNVR:             "10.88.1.10",
		ChannelPort:       1,
		RTSPPassword:      encPass,
		SiteID:            &siteID,
		Status:            "aktif",
		StreamEnabled:     true,
		MonitoringEnabled: true,
	}

	makeRes := func() streamVerificationResult {
		return streamVerificationResult{
			CCTV:           gsmCam,
			State:          models.StreamStateLive,
			VerifiedAt:     time.Now().UTC(),
			Method:         streamVerifyMethodHLS,
			Duration:       900 * time.Millisecond,
			BytesVerify:    16384,
			ByteDelta:      8192,
			OpenedOnDemand: true,
		}
	}

	t.Run("RemovePath gagal tidak boleh ditandai terkonfirmasi", func(t *testing.T) {
		db, mock := setupMockGORM(t)
		mock.ExpectExec(`INSERT INTO "cctv_probe_telemetry"`).WillReturnResult(sqlmock.NewResult(1, 1))

		var addCalls atomic.Int32
		mux := http.NewServeMux()
		mux.HandleFunc("/v3/paths/get/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"name":          r.URL.Path[len("/v3/paths/get/"):],
				"available":     true,
				"ready":         true,
				"readers":       []interface{}{},
				"bytesReceived": 24576,
			})
		})
		mux.HandleFunc("/v3/config/paths/delete/", func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, `{"error":"delete failed"}`, http.StatusInternalServerError)
		})
		mux.HandleFunc("/v3/config/paths/add/", func(w http.ResponseWriter, _ *http.Request) {
			addCalls.Add(1)
			w.WriteHeader(http.StatusOK)
		})
		srv := httptest.NewServer(mux)
		defer srv.Close()

		oldClient := streamClient
		streamClient = &utils.MediaMTXClient{BaseURL: srv.URL, HLSURL: srv.URL, HTTPClient: srv.Client()}
		defer func() { streamClient = oldClient }()

		tel := evaluateAndPersistSourceClosureTelemetry(db, GateJob{CCTV: gsmCam}, makeRes())
		assert.False(t, tel.SourceClosedConfirmed, "RemovePath failure while source ready MUST keep SourceClosedConfirmed=false")
		assert.True(t, tel.ForcedClose)
		assert.Equal(t, RemovePathStatusFailed, tel.RemovePathStatus)
		assert.NotEmpty(t, tel.RemovePathError)
		assert.Equal(t, ReregisterStatusSkipped, tel.ReregisterStatus)
		assert.Equal(t, CloseVerifyStillActive, tel.CloseVerifyStatus)
		assert.Contains(t, tel.CloseUnconfirmedReason, "pelepasan path GSM gagal")
		assert.Zero(t, addCalls.Load(), "re-register must be skipped when RemovePath failed")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Pemeriksaan ulang gagal setelah RemovePath tidak boleh ditandai terkonfirmasi", func(t *testing.T) {
		db, mock := setupMockGORM(t)
		mock.ExpectExec(`INSERT INTO "cctv_probe_telemetry"`).WillReturnResult(sqlmock.NewResult(1, 1))

		var getCount atomic.Int32
		mux := http.NewServeMux()
		mux.HandleFunc("/v3/paths/get/", func(w http.ResponseWriter, r *http.Request) {
			call := getCount.Add(1)
			if call == 1 {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"name":          r.URL.Path[len("/v3/paths/get/"):],
					"available":     true,
					"ready":         true,
					"readers":       []interface{}{},
					"bytesReceived": 20000,
				})
				return
			}
			http.Error(w, `{"error":"mediamtx internal error"}`, http.StatusInternalServerError)
		})
		mux.HandleFunc("/v3/config/paths/delete/", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		mux.HandleFunc("/v3/config/paths/add/", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		srv := httptest.NewServer(mux)
		defer srv.Close()

		oldClient := streamClient
		streamClient = &utils.MediaMTXClient{BaseURL: srv.URL, HLSURL: srv.URL, HTTPClient: srv.Client()}
		defer func() { streamClient = oldClient }()

		tel := evaluateAndPersistSourceClosureTelemetry(db, GateJob{CCTV: gsmCam}, makeRes())
		assert.False(t, tel.SourceClosedConfirmed, "failed recheck MUST NOT set SourceClosedConfirmed=true")
		assert.True(t, tel.ForcedClose)
		assert.Equal(t, RemovePathStatusSucceeded, tel.RemovePathStatus)
		assert.Equal(t, ReregisterStatusSucceeded, tel.ReregisterStatus)
		assert.Equal(t, CloseVerifyCheckFailed, tel.CloseVerifyStatus)
		assert.Contains(t, tel.CloseUnconfirmedReason, "pemeriksaan ulang penutupan sumber di MediaMTX gagal")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Pemeriksaan ulang menunjukkan sumber masih aktif tidak boleh ditandai terkonfirmasi", func(t *testing.T) {
		db, mock := setupMockGORM(t)
		mock.ExpectExec(`INSERT INTO "cctv_probe_telemetry"`).WillReturnResult(sqlmock.NewResult(1, 1))

		mux := http.NewServeMux()
		mux.HandleFunc("/v3/paths/get/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"name":          r.URL.Path[len("/v3/paths/get/"):],
				"available":     true,
				"ready":         true,
				"readers":       []interface{}{},
				"bytesReceived": 30000,
			})
		})
		mux.HandleFunc("/v3/config/paths/delete/", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		mux.HandleFunc("/v3/config/paths/add/", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		srv := httptest.NewServer(mux)
		defer srv.Close()

		oldClient := streamClient
		streamClient = &utils.MediaMTXClient{BaseURL: srv.URL, HLSURL: srv.URL, HTTPClient: srv.Client()}
		defer func() { streamClient = oldClient }()

		tel := evaluateAndPersistSourceClosureTelemetry(db, GateJob{CCTV: gsmCam}, makeRes())
		assert.False(t, tel.SourceClosedConfirmed)
		assert.Equal(t, RemovePathStatusSucceeded, tel.RemovePathStatus)
		assert.Equal(t, ReregisterStatusSucceeded, tel.ReregisterStatus)
		assert.Equal(t, CloseVerifyStillActive, tel.CloseVerifyStatus)
		assert.NotEmpty(t, tel.CloseUnconfirmedReason)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Penutupan paksa GSM berhasil dan pemeriksaan ulang membuktikan ready=false tercatat terkonfirmasi", func(t *testing.T) {
		db, mock := setupMockGORM(t)
		mock.ExpectExec(`INSERT INTO "cctv_probe_telemetry"`).WillReturnResult(sqlmock.NewResult(1, 1))

		var sourceStopped atomic.Bool
		mux := http.NewServeMux()
		mux.HandleFunc("/v3/paths/get/", func(w http.ResponseWriter, r *http.Request) {
			active := !sourceStopped.Load()
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"name":          r.URL.Path[len("/v3/paths/get/"):],
				"available":     active,
				"ready":         active,
				"readers":       []interface{}{},
				"bytesReceived": 28672,
			})
		})
		mux.HandleFunc("/v3/config/paths/delete/", func(w http.ResponseWriter, _ *http.Request) {
			sourceStopped.Store(true)
			w.WriteHeader(http.StatusOK)
		})
		mux.HandleFunc("/v3/config/paths/add/", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		srv := httptest.NewServer(mux)
		defer srv.Close()

		oldClient := streamClient
		streamClient = &utils.MediaMTXClient{BaseURL: srv.URL, HLSURL: srv.URL, HTTPClient: srv.Client()}
		defer func() { streamClient = oldClient }()

		tel := evaluateAndPersistSourceClosureTelemetry(db, GateJob{CCTV: gsmCam}, makeRes())
		assert.True(t, tel.SourceClosedConfirmed, "positive MediaMTX evidence (!SourceReady) after forced close must confirm closure")
		assert.True(t, tel.ForcedClose)
		assert.Equal(t, RemovePathStatusSucceeded, tel.RemovePathStatus)
		assert.Empty(t, tel.RemovePathError)
		assert.Equal(t, ReregisterStatusSucceeded, tel.ReregisterStatus)
		assert.Empty(t, tel.ReregisterError)
		assert.Equal(t, CloseVerifyClosedAfterForcedRemove, tel.CloseVerifyStatus)
		assert.Empty(t, tel.CloseUnconfirmedReason)
		assert.EqualValues(t, 28672, tel.BytesTotalSession)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Penutupan alami oleh sourceOnDemandCloseAfter langsung terkonfirmasi tanpa RemovePath", func(t *testing.T) {
		db, mock := setupMockGORM(t)
		mock.ExpectExec(`INSERT INTO "cctv_probe_telemetry"`).WillReturnResult(sqlmock.NewResult(1, 1))

		mux := http.NewServeMux()
		mux.HandleFunc("/v3/paths/get/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"name":          r.URL.Path[len("/v3/paths/get/"):],
				"available":     false,
				"ready":         false,
				"readers":       []interface{}{},
				"bytesReceived": 18000,
			})
		})
		srv := httptest.NewServer(mux)
		defer srv.Close()

		oldClient := streamClient
		streamClient = &utils.MediaMTXClient{BaseURL: srv.URL, HLSURL: srv.URL, HTTPClient: srv.Client()}
		defer func() { streamClient = oldClient }()

		tel := evaluateAndPersistSourceClosureTelemetry(db, GateJob{CCTV: gsmCam}, makeRes())
		assert.True(t, tel.SourceClosedConfirmed)
		assert.False(t, tel.ForcedClose)
		assert.Equal(t, RemovePathStatusNotNeeded, tel.RemovePathStatus)
		assert.Equal(t, ReregisterStatusNotNeeded, tel.ReregisterStatus)
		assert.Equal(t, CloseVerifyClosedNaturally, tel.CloseVerifyStatus)
		assert.Empty(t, tel.CloseUnconfirmedReason)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
