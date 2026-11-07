package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
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

func TestUnifiedGatePriorityOrderingAndUpstreamLeaseCapWithTail(t *testing.T) {
	// Skala waktu dipercepat 50x: MaxSlots=4, MaxGSM=1, Pace=10ms, SourceCloseTail=60ms.
	gate := NewUnifiedNVRSessionGate(4, 1, 10*time.Millisecond, 60*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Tahan sementara slot agar antrean terkumpul dan urutan prioritas dapat diuji deterministik.
	gate.mu.Lock()
	mainID := uuid.New()
	postMaintID := uuid.New()
	confirmID := uuid.New()

	gate.mainQueue = append(gate.mainQueue, GateJob{
		Priority: GatePriorityMain,
		CCTV:     models.CCTV{Base: models.Base{ID: mainID}, IPCamera: "10.20.105.1", Nama: "MainCam"},
		ReadyAt:  time.Now(),
	})
	gate.queuedPriority[mainID] = GatePriorityMain

	gate.postMaintQueue = append(gate.postMaintQueue, GateJob{
		Priority: GatePriorityPostMaintenance,
		CCTV:     models.CCTV{Base: models.Base{ID: postMaintID}, IPCamera: "10.20.105.2", Nama: "PostMaintCam"},
		ReadyAt:  time.Now(),
	})
	gate.queuedPriority[postMaintID] = GatePriorityPostMaintenance

	gate.confirmQueue = append(gate.confirmQueue, GateJob{
		Priority: GatePriorityConfirm,
		CCTV:     models.CCTV{Base: models.Base{ID: confirmID}, IPCamera: "10.20.105.3", Nama: "ConfirmCam"},
		ReadyAt:  time.Now(),
	})
	gate.queuedPriority[confirmID] = GatePriorityConfirm
	gate.mu.Unlock()

	// Tambahkan 12 kamera lagi (termasuk 4 kamera GSM) yang semuanya membuka sesi on-demand (tail lease aktif).
	totalJobs := 3 + 12
	for idx := 0; idx < 12; idx++ {
		ip := "10.20.105.10"
		if idx%3 == 0 {
			ip = "" // diklasifikasikan sebagai GSM oleh networkType()
		}
		gate.Enqueue(GateJob{
			Priority: GatePriorityMain,
			CCTV:     models.CCTV{Base: models.Base{ID: uuid.New()}, IPCamera: ip, Nama: "BulkCam"},
		})
	}

	var mu sync.Mutex
	var startOrder []uuid.UUID
	var startTimes []time.Time
	var activeGSMNow atomic.Int32
	var maxGSMObserved atomic.Int32
	doneCh := make(chan struct{}, totalJobs)

	go gate.RunDispatcher(ctx, func(_ context.Context, job GateJob) (streamVerificationResult, bool) {
		isGSM := networkType(job.CCTV) == "gsm"
		if isGSM {
			cur := activeGSMNow.Add(1)
			for {
				prev := maxGSMObserved.Load()
				if cur <= prev || maxGSMObserved.CompareAndSwap(prev, cur) {
					break
				}
			}
			defer activeGSMNow.Add(-1)
		}

		mu.Lock()
		startOrder = append(startOrder, job.CCTV.ID)
		startTimes = append(startTimes, time.Now())
		mu.Unlock()

		time.Sleep(15 * time.Millisecond)
		return streamVerificationResult{CCTV: job.CCTV, State: models.StreamStateLive}, true
	}, func(_ GateJob, _ streamVerificationResult) {
		doneCh <- struct{}{}
	}, nil)

	for i := 0; i < totalJobs; i++ {
		select {
		case <-doneCh:
		case <-time.After(5 * time.Second):
			t.Fatalf("timeout waiting for job %d/%d", i+1, totalJobs)
		}
	}

	snap := gate.Snapshot()
	// 1. Batas gabungan (active + closing tail) TIDAK PERNAH melampaui MaxUpstreamSlots (4).
	assert.LessOrEqual(t, snap.PeakOccupiedSlots, 4, "active + tail leases must never exceed MaxUpstreamSlots")
	// 2. Konkurensi GSM TIDAK PERNAH melampaui MaxGSMConcurrent (1).
	assert.LessOrEqual(t, int(maxGSMObserved.Load()), 1, "concurrent GSM probes must never exceed MaxGSMConcurrent")

	mu.Lock()
	defer mu.Unlock()
	require.GreaterOrEqual(t, len(startOrder), 3)
	// 3. Urutan prioritas ketat: Confirm -> PostMaintenance -> Main.
	assert.Equal(t, confirmID, startOrder[0])
	assert.Equal(t, postMaintID, startOrder[1])
	assert.Equal(t, mainID, startOrder[2])

	// 4. Pacer tunggal lintas antrean menjaga jarak antar-start >= PaceInterval (toleransi scheduler OS 2ms).
	for i := 1; i < len(startTimes); i++ {
		assert.GreaterOrEqual(t, startTimes[i].Sub(startTimes[i-1]), 8*time.Millisecond)
	}
}

func TestUnifiedGateDeduplicatesPerCameraAndPromotesHigherPriority(t *testing.T) {
	gate := NewUnifiedNVRSessionGate(10, 2, 0, 0)
	cid := uuid.New()
	cam := models.CCTV{Base: models.Base{ID: cid}, IPCamera: "10.20.1.1"}

	assert.True(t, gate.Enqueue(GateJob{Priority: GatePriorityMain, CCTV: cam}))
	assert.False(t, gate.Enqueue(GateJob{Priority: GatePriorityMain, CCTV: cam}), "duplicate Main job must be rejected")

	// Promosi dari Main -> Confirm harus diterima dan menghapus entri lama di Main.
	assert.True(t, gate.Enqueue(GateJob{Priority: GatePriorityConfirm, CCTV: cam}))
	snap := gate.Snapshot()
	assert.Equal(t, 1, snap.QueuedConfirm)
	assert.Equal(t, 0, snap.QueuedMain)
}

// TestTimeBudgetModel35SimultaneousFailuresAndPostMaintenance memverifikasi secara matematis & simulasi
// anggaran waktu Koreksi #2 untuk gelombang 35 kamera gagal serentak dan 35 jendela pemeliharaan
// berakhir bersamaan di bawah parameter produksi (MaxSlots=10, Pace=1.25s, VerifyTimeout=8.5s,
// ConfirmBackoff=10s, CloseTail=5s) tetap berada jauh di bawah target 300 detik (5 menit).
func TestTimeBudgetModel35SimultaneousFailuresAndPostMaintenance(t *testing.T) {
	const (
		cameras        = 35
		maxSlots       = 10
		paceMs         = 1250
		offlineProbeMs = 8500 // 8s timeout + 0.5s overhead, openedOnDemand=false untuk kamera mati
		liveTailMs     = 6500 // ~1.5s live verify + 5s sourceOnDemandCloseAfter tail lease
		confirmDelayMs = 10000
	)

	// Simulasi diskrit antrean prioritas tunggal untuk 35 kamera yang putus serentak tepat setelah baru terlewat sweep (t_lag = 60s):
	type simSlot struct{ freeAtMs int }
	slots := make([]simSlot, maxSlots)
	lastPaceMs := -paceMs

	acquire := func(earliestReadyMs, holdMs int) int {
		startMs := earliestReadyMs
		if lastPaceMs+paceMs > startMs {
			startMs = lastPaceMs + paceMs
		}
		// Cari slot paling cepat bebas
		bestIdx := 0
		for i := 1; i < len(slots); i++ {
			if slots[i].freeAtMs < slots[bestIdx].freeAtMs {
				bestIdx = i
			}
		}
		if slots[bestIdx].freeAtMs > startMs {
			startMs = slots[bestIdx].freeAtMs
		}
		lastPaceMs = startMs
		slots[bestIdx].freeAtMs = startMs + holdMs
		return startMs + offlineProbeMs
	}

	// Fase 1: 35 kamera masuk MainQueue pada t = 60.000 ms (kasus terburuk tepat 1ms setelah sweep sebelumnya).
	confirmReady := make([]int, cameras)
	for i := 0; i < cameras; i++ {
		done1 := acquire(60_000, offlineProbeMs)
		confirmReady[i] = done1 + confirmDelayMs
	}
	// Fase 2: 35 kamera masuk ConfirmQueue setelah backoff 10s masing-masing.
	maxConfirmedMs := 0
	for i := 0; i < cameras; i++ {
		done2 := acquire(confirmReady[i], offlineProbeMs)
		if done2 > maxConfirmedMs {
			maxConfirmedMs = done2
		}
	}
	// Tambah Coalesce 8s + Batch Telegram + Retry Ambigu 10s (~26.5s = 26.500ms)
	worstTotalOutageSec := float64(maxConfirmedMs+26_500) / 1000.0
	assert.Less(t, worstTotalOutageSec, 300.0, "35 simultaneous camera failures must complete well under 300s SLA budget")

	// Simulasi 35 jendela pemeliharaan berakhir serentak dengan seluruh slot awalnya sedang memegang live tail lease (6.5s):
	for i := range slots {
		slots[i].freeAtMs = liveTailMs
	}
	lastPaceMs = 0
	maxPostMaintDoneMs := 0
	for i := 0; i < cameras; i++ {
		done := acquire(5_000, offlineProbeMs) // ticker 5s mendeteksi akhir jendela
		if done > maxPostMaintDoneMs {
			maxPostMaintDoneMs = done
		}
	}
	worstPostMaintSec := float64(maxPostMaintDoneMs+26_500) / 1000.0
	assert.Less(t, worstPostMaintSec, 300.0, "35 simultaneous post-maintenance expirations must complete well under 300s SLA budget")
}

func TestManualVerifyAndReactivateShareSessionGateAndRespectCapsWhenFull(t *testing.T) {
	origWait := manualVerifyGateWaitTimeout
	origTimeout := streamVerifyTimeout
	origPoll := streamVerifyPollInterval
	origByteWin := streamVerifyByteWindow
	origCloseDelay := sourceCloseConfirmDelay
	defer func() {
		manualVerifyGateWaitTimeout = origWait
		streamVerifyTimeout = origTimeout
		streamVerifyPollInterval = origPoll
		streamVerifyByteWindow = origByteWin
		sourceCloseConfirmDelay = origCloseDelay
		replaceVerifierGateForTest(nil, false)
	}()

	streamVerifyTimeout = 80 * time.Millisecond
	streamVerifyPollInterval = 10 * time.Millisecond
	streamVerifyByteWindow = 10 * time.Millisecond
	sourceCloseConfirmDelay = 5 * time.Millisecond

	siteA := uuid.New()
	siteB := uuid.New()
	verifyCamID := uuid.New()
	reactivateCamID := uuid.New()
	crossSiteCamID := uuid.New()
	encPass := encryptedTestPassword(t, "secret123")

	var mtxMu sync.Mutex
	mtxTouches := map[string]int{}
	firstTouchAt := map[string]time.Time{}
	bytesCounter := map[string]int64{}
	recordTouch := func(pathName string) {
		mtxMu.Lock()
		mtxTouches[pathName]++
		if _, ok := firstTouchAt[pathName]; !ok {
			firstTouchAt[pathName] = time.Now()
		}
		mtxMu.Unlock()
	}
	getTouches := func(camID uuid.UUID) int {
		mtxMu.Lock()
		defer mtxMu.Unlock()
		return mtxTouches[utils.BuildStreamPath(camID.String())]
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v3/config/paths/add/", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Path[len("/v3/config/paths/add/"):]
		recordTouch(name)
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/v3/config/paths/patch/", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Path[len("/v3/config/paths/patch/"):]
		recordTouch(name)
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/v3/config/paths/delete/", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Path[len("/v3/config/paths/delete/"):]
		recordTouch(name)
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/v3/paths/get/", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Path[len("/v3/paths/get/"):]
		recordTouch(name)
		mtxMu.Lock()
		bytesCounter[name] += 4096
		curBytes := bytesCounter[name]
		mtxMu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"name":          name,
			"available":     true,
			"ready":         true,
			"tracks":        []string{"H264"},
			"bytesReceived": curBytes,
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	oldClient := streamClient
	streamClient = &utils.MediaMTXClient{BaseURL: srv.URL, HLSURL: srv.URL, HTTPClient: srv.Client()}
	defer func() { streamClient = oldClient }()

	_, mock := setupMockGORM(t)
	mock.MatchExpectationsInOrder(false)

	cctvColumns := []string{
		"id", "kode_cctv", "nama", "ip_camera", "ip_nvr", "channel_port",
		"rtsp_password", "site_id", "status", "stream_enabled", "monitoring_enabled",
		"stream_verified_state", "stream_verify_failures",
	}
	expectLoadCCTV := func(id uuid.UUID, code, ipCam, status string, site uuid.UUID, streamOn, monOn bool) {
		mock.ExpectQuery(`SELECT \* FROM "cctvs"`).
			WithArgs(id, 1).
			WillReturnRows(sqlmock.NewRows(cctvColumns).AddRow(
				id, code, "Kamera "+code, ipCam, "10.10.0.1", 1,
				encPass, site, status, streamOn, monOn,
				models.StreamStateUnknown, 0,
			))
	}

	buildApp := func(role, activeSite string) *fiber.App {
		app := fiber.New()
		app.Use(func(c *fiber.Ctx) error {
			c.Locals("role", role)
			c.Locals("site_id", activeSite)
			c.Locals("user_id", uuid.New().String())
			return c.Next()
		})
		app.Post("/streams/:id/verify", VerifyStream)
		app.Post("/cctv/:id/reactivate-verify", ReactivateAndVerifyCCTV)
		return app
	}

	// 1. Otorisasi Admin & isolasi site tetap terjaga sebelum masuk gate.
	nonAdminApp := buildApp("teknisi", siteA.String())
	respForbidden1, err := nonAdminApp.Test(httptest.NewRequest(http.MethodPost, "/streams/"+verifyCamID.String()+"/verify", nil))
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, respForbidden1.StatusCode)

	respForbidden2, err := nonAdminApp.Test(httptest.NewRequest(http.MethodPost, "/cctv/"+reactivateCamID.String()+"/reactivate-verify", nil))
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, respForbidden2.StatusCode)

	adminAppSiteA := buildApp("admin", siteA.String())
	expectLoadCCTV(crossSiteCamID, "CAM-SITE-B", "10.20.105.9", "aktif", siteB, true, true)
	respCrossSite, err := adminAppSiteA.Test(httptest.NewRequest(http.MethodPost, "/streams/"+crossSiteCamID.String()+"/verify", nil))
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, respCrossSite.StatusCode)
	assert.Equal(t, 0, getTouches(crossSiteCamID))

	// 2. Gate penuh melebihi batas waktu tunggu: keduanya TIDAK membuka stream & sesi tetap dalam batas (MaxSlots=2, MaxGSM=1).
	gate := NewUnifiedNVRSessionGate(2, 1, 5*time.Millisecond, 25*time.Millisecond)
	replaceVerifierGateForTest(gate, true)

	blockHold := make(chan struct{})
	blockerStarted := make(chan struct{}, 2)
	var gsmActive atomic.Int32
	var maxGSMConcurrent atomic.Int32

	// Penuhi 2 slot gate (1 GSM + 1 Metro-E 10.20.105.x).
	gate.Enqueue(GateJob{
		Priority:              GatePriorityConfirm,
		CCTV:                  models.CCTV{Base: models.Base{ID: uuid.New()}, IPCamera: "", Nama: "BlockerGSM"},
		SkipCompletionHandler: true,
		JobProbe: func(_ context.Context, _ GateJob) (streamVerificationResult, bool) {
			cur := gsmActive.Add(1)
			for {
				prev := maxGSMConcurrent.Load()
				if cur <= prev || maxGSMConcurrent.CompareAndSwap(prev, cur) {
					break
				}
			}
			defer gsmActive.Add(-1)
			blockerStarted <- struct{}{}
			<-blockHold
			return streamVerificationResult{State: models.StreamStateLive}, false
		},
	})
	gate.Enqueue(GateJob{
		Priority:              GatePriorityConfirm,
		CCTV:                  models.CCTV{Base: models.Base{ID: uuid.New()}, IPCamera: "10.20.105.20", Nama: "BlockerMetro"},
		SkipCompletionHandler: true,
		JobProbe: func(_ context.Context, _ GateJob) (streamVerificationResult, bool) {
			blockerStarted <- struct{}{}
			<-blockHold
			return streamVerificationResult{State: models.StreamStateLive}, false
		},
	})

	<-blockerStarted
	<-blockerStarted
	snapFull := gate.Snapshot()
	require.Equal(t, 2, snapFull.ActiveVerifications)
	require.Equal(t, 1, snapFull.ActiveGSMSlots)

	manualVerifyGateWaitTimeout = 70 * time.Millisecond
	expectLoadCCTV(verifyCamID, "CAM-MAN-1", "10.20.105.50", "aktif", siteA, true, true)
	expectLoadCCTV(reactivateCamID, "CAM-GSM-REACT", "", "nonaktif", siteA, false, false)

	var wgTimeout sync.WaitGroup
	var statusVerifyTimeout, statusReactivateTimeout int
	wgTimeout.Add(2)
	go func() {
		defer wgTimeout.Done()
		r, reqErr := adminAppSiteA.Test(httptest.NewRequest(http.MethodPost, "/streams/"+verifyCamID.String()+"/verify", nil), 2000)
		require.NoError(t, reqErr)
		statusVerifyTimeout = r.StatusCode
	}()
	go func() {
		defer wgTimeout.Done()
		r, reqErr := adminAppSiteA.Test(httptest.NewRequest(http.MethodPost, "/cctv/"+reactivateCamID.String()+"/reactivate-verify", nil), 2000)
		require.NoError(t, reqErr)
		statusReactivateTimeout = r.StatusCode
	}()
	wgTimeout.Wait()

	assert.Equal(t, http.StatusServiceUnavailable, statusVerifyTimeout, "manual verify must return 503 when gate remains full past timeout")
	assert.Equal(t, http.StatusServiceUnavailable, statusReactivateTimeout, "reactivate must return 503 when gate remains full past timeout")
	assert.Equal(t, 0, getTouches(verifyCamID), "manual verify MUST NOT touch MediaMTX before acquiring gate slot")
	assert.Equal(t, 0, getTouches(reactivateCamID), "reactivate MUST NOT register or open stream before acquiring gate slot")
	assert.LessOrEqual(t, gate.Snapshot().PeakOccupiedSlots, 2, "gate session count must never exceed MaxUpstreamSlots=2")
	assert.LessOrEqual(t, gate.Snapshot().ActiveGSMSlots, 1, "gate GSM count must never exceed MaxGSMConcurrent=1")

	// 3. Gate penuh di awal lalu slot dibebaskan sebelum timeout:
	//    keduanya menunggu slot, tidak menyentuh MediaMTX sebelum slot bebas, tetap mematuhi batas sesi & GSM,
	//    dan ReactivateAndVerifyCCTV mengaktifkan kamera hanya setelah verifikasi live berhasil.
	manualVerifyGateWaitTimeout = 1500 * time.Millisecond
	expectLoadCCTV(verifyCamID, "CAM-MAN-1", "10.20.105.50", "aktif", siteA, true, true)
	expectLoadCCTV(reactivateCamID, "CAM-GSM-REACT", "", "nonaktif", siteA, false, false)
	expectNoPreviousCCTVStreamLog(mock, verifyCamID)
	expectNoPreviousCCTVStreamLog(mock, reactivateCamID)
	expectCCTVStreamLogInsert(mock)
	expectCCTVStreamLogInsert(mock)
	mock.ExpectExec(`UPDATE "cctvs" SET`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE "cctvs" SET`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE "cctvs" SET`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT \* FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows(cctvColumns).AddRow(
			reactivateCamID, "CAM-GSM-REACT", "Kamera CAM-GSM-REACT", "", "10.10.0.1", 1,
			encPass, siteA, "aktif", true, true,
			models.StreamStateLive, 0,
		))
	mock.ExpectQuery(`SELECT "username","role" FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"username", "role"}).AddRow("admin-test", "admin"))
	mock.ExpectExec(`INSERT INTO "audit_logs"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	var wgSuccess sync.WaitGroup
	var statusVerifyOK, statusReactivateOK int
	var reactivateRespBody map[string]interface{}
	wgSuccess.Add(2)
	go func() {
		defer wgSuccess.Done()
		r, reqErr := adminAppSiteA.Test(httptest.NewRequest(http.MethodPost, "/streams/"+verifyCamID.String()+"/verify", nil), 3000)
		require.NoError(t, reqErr)
		statusVerifyOK = r.StatusCode
	}()
	go func() {
		defer wgSuccess.Done()
		r, reqErr := adminAppSiteA.Test(httptest.NewRequest(http.MethodPost, "/cctv/"+reactivateCamID.String()+"/reactivate-verify", nil), 3000)
		require.NoError(t, reqErr)
		statusReactivateOK = r.StatusCode
		_ = json.NewDecoder(r.Body).Decode(&reactivateRespBody)
	}()

	// Biarkan kedua request mengantre selama 50ms saat gate masih penuh, buktikan 0 sentuhan ke MediaMTX!
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 0, getTouches(verifyCamID), "while waiting in full gate, manual verify must not open stream")
	assert.Equal(t, 0, getTouches(reactivateCamID), "while waiting in full gate, reactivate must not register/open stream")

	releasedAt := time.Now()
	close(blockHold)
	wgSuccess.Wait()

	assert.Equal(t, http.StatusOK, statusVerifyOK)
	assert.Equal(t, http.StatusOK, statusReactivateOK)
	assert.Greater(t, getTouches(verifyCamID), 0)
	assert.Greater(t, getTouches(reactivateCamID), 0)

	mtxMu.Lock()
	verifyTouchTime := firstTouchAt[utils.BuildStreamPath(verifyCamID.String())]
	reactivateTouchTime := firstTouchAt[utils.BuildStreamPath(reactivateCamID.String())]
	mtxMu.Unlock()

	assert.False(t, verifyTouchTime.Before(releasedAt), "manual verify touched MediaMTX only after slot released")
	assert.False(t, reactivateTouchTime.Before(releasedAt), "reactivate touched MediaMTX only after slot released")
	assert.LessOrEqual(t, gate.Snapshot().PeakOccupiedSlots, 2, "combined sessions never exceeded MaxUpstreamSlots=2")
	assert.LessOrEqual(t, int(maxGSMConcurrent.Load()), 1, "GSM concurrency never exceeded MaxGSMConcurrent=1")

	dataMap, _ := reactivateRespBody["data"].(map[string]interface{})
	if assert.NotNil(t, dataMap) {
		assert.Equal(t, "aktif", dataMap["status"])
	}
}

func TestReactivateAndVerifyCCTVRemainsInactiveWhenProbeNotLive(t *testing.T) {
	origTimeout := streamVerifyTimeout
	origPoll := streamVerifyPollInterval
	origByteWin := streamVerifyByteWindow
	defer func() {
		streamVerifyTimeout = origTimeout
		streamVerifyPollInterval = origPoll
		streamVerifyByteWindow = origByteWin
		replaceVerifierGateForTest(nil, false)
	}()

	streamVerifyTimeout = 40 * time.Millisecond
	streamVerifyPollInterval = 10 * time.Millisecond
	streamVerifyByteWindow = 10 * time.Millisecond

	gate := NewUnifiedNVRSessionGate(2, 1, 0, 10*time.Millisecond)
	replaceVerifierGateForTest(gate, true)

	var removed atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/v3/config/paths/add/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/v3/config/paths/delete/", func(w http.ResponseWriter, _ *http.Request) {
		removed.Add(1)
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/v3/paths/get/", func(w http.ResponseWriter, r *http.Request) {
		// Sumber tampak ready tetapi 0 pertumbuhan byte (stalled stream) -> gagal verifikasi live.
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"name":          r.URL.Path[len("/v3/paths/get/"):],
			"available":     true,
			"ready":         true,
			"tracks":        []string{"H264"},
			"bytesReceived": 1024,
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	oldClient := streamClient
	streamClient = &utils.MediaMTXClient{BaseURL: srv.URL, HLSURL: srv.URL, HTTPClient: srv.Client()}
	defer func() { streamClient = oldClient }()

	_, mock := setupMockGORM(t)
	camID := uuid.New()
	siteID := uuid.New()
	encPass := encryptedTestPassword(t, "secret123")

	mock.ExpectQuery(`SELECT \* FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "kode_cctv", "nama", "ip_camera", "ip_nvr", "channel_port",
			"rtsp_password", "site_id", "status", "stream_enabled", "monitoring_enabled",
			"stream_verified_state", "stream_verify_failures",
		}).AddRow(
			camID, "CAM-FAIL-REACT", "Kamera Gagal Reaktivasi", "", "10.10.0.1", 1,
			encPass, siteID, "nonaktif", false, false,
			models.StreamStateOffline, 2,
		))
	// Hanya boleh ada 1 UPDATE (persistStreamVerification); TIDAK BOLEH ada UPDATE kedua yang mengubah status menjadi 'aktif'.
	mock.ExpectExec(`UPDATE "cctvs" SET`).WillReturnResult(sqlmock.NewResult(0, 1))

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("role", "admin")
		c.Locals("site_id", siteID.String())
		c.Locals("user_id", uuid.New().String())
		return c.Next()
	})
	app.Post("/cctv/:id/reactivate-verify", ReactivateAndVerifyCCTV)

	resp, err := app.Test(httptest.NewRequest(http.MethodPost, "/cctv/"+camID.String()+"/reactivate-verify", nil), 2000)
	require.NoError(t, err)
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
	assert.EqualValues(t, 1, removed.Load(), "temporary path for failed reactivation must be cleaned up")
	assert.NoError(t, mock.ExpectationsWereMet())
}
