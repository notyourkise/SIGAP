package handlers

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"sigap-backend/config"
	"sigap-backend/incidents"
	"sigap-backend/models"
	"sigap-backend/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Verifikasi aktif stream membedakan status pasif (path on-demand terdaftar)
// dari bukti bahwa sumber benar-benar mengirim track & byte RTP aktif.
const (
	defaultStreamVerifyInterval        = 15 * time.Minute
	defaultStreamVerifyConcurrency     = 1
	streamVerifyFailureThreshold       = 2
	streamVerifyFreshness              = 30 * time.Minute
	streamVerifyMethodRTSP             = utils.TriggerMethodRTSP
	streamVerifyMethodHLS              = utils.TriggerMethodHLS
	streamVerifyMethodLive             = "mediamtx-live"
	defaultManualVerifyGateWaitTimeout = 15 * time.Second

	RemovePathStatusNotNeeded = "not_needed"
	RemovePathStatusSucceeded = "succeeded"
	RemovePathStatusFailed    = "failed"

	ReregisterStatusNotNeeded = "not_needed"
	ReregisterStatusSucceeded = "succeeded"
	ReregisterStatusFailed    = "failed"
	ReregisterStatusSkipped   = "skipped"

	CloseVerifyClosedNaturally         = "closed_naturally"
	CloseVerifyClosedAfterForcedRemove = "closed_after_forced_remove"
	CloseVerifyPathAbsentConfirmed     = "path_absent_confirmed"
	CloseVerifyStillActive             = "still_active"
	CloseVerifyActiveWithReaders       = "active_with_readers"
	CloseVerifyCheckFailed             = "check_failed"
	CloseVerifyNotOpenedOnDemand       = "not_opened_on_demand"
)

// Durasi ini variabel agar pengujian dapat mempercepat polling tanpa membuat
// test menunggu timeout produksi 8 detik.
var (
	streamVerifyTimeout         = 8 * time.Second
	streamVerifyPollInterval    = 500 * time.Millisecond
	streamVerifyByteWindow      = 500 * time.Millisecond
	streamVerifyRequireBytes    = false
	confirmRetryDelay           = defaultConfirmRetryDelay
	manualVerifyGateWaitTimeout = defaultManualVerifyGateWaitTimeout
	sourceCloseConfirmDelay     = 500 * time.Millisecond
	probeNVRRTSPHost            = func(ip string) bool { return utils.ProbeRTSPHost(ip, 554) }
)

var (
	verifierGateMu sync.Mutex
	verifierGate   *UnifiedNVRSessionGate

	firstFailureMu  sync.Mutex
	firstFailureMap = map[uuid.UUID]time.Time{}

	heartbeatMu                  sync.Mutex
	lastVerifierCycleStartedAt   *time.Time
	lastVerifierCycleCompletedAt *time.Time
	lastVerifierCameraCount      int
	lastMediaMTXHealthy          = true
	lastMediaMTXError            string
)

// streamVerificationResult adalah hasil satu percobaan verifikasi.
type streamVerificationResult struct {
	CCTV           models.CCTV
	State          string
	VerifiedAt     time.Time
	Method         string
	Failures       int
	Tracks         int
	Duration       time.Duration
	Error          string
	BytesVerify    int64
	ByteDelta      int64
	OpenedOnDemand bool
	SuppressObserve bool
	FirstFailedAt   *time.Time
}

func resolveGateDB(fallback *gorm.DB) *gorm.DB {
	if config.DB != nil {
		return config.DB
	}
	return fallback
}

func resolveManualVerifyWaitTimeout() time.Duration {
	if raw := strings.TrimSpace(os.Getenv("CCTV_MANUAL_VERIFY_WAIT_TIMEOUT_MS")); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v >= 50 && v <= 60000 {
			return time.Duration(v) * time.Millisecond
		}
	}
	return manualVerifyGateWaitTimeout
}

func startVerifierGateDispatcher(gate *UnifiedNVRSessionGate, fallbackDB *gorm.DB) {
	go gate.RunDispatcher(
		context.Background(),
		func(_ context.Context, job GateJob) (streamVerificationResult, bool) {
			activeDB := resolveGateDB(fallbackDB)
			suppress := job.PostMaintRefresh != nil
			res := verifyCCTVStreamInternal(activeDB, job.CCTV, suppress, job.FirstFailedAt)
			return res, res.OpenedOnDemand
		},
		func(job GateJob, res streamVerificationResult) {
			handleGateJobCompletion(resolveGateDB(fallbackDB), job, res)
		},
		func(job GateJob, res streamVerificationResult) {
			confirmSourceClosedAndRecordTelemetry(resolveGateDB(fallbackDB), job, res)
		},
	)
}

func getOrInitVerifierGate(db *gorm.DB) *UnifiedNVRSessionGate {
	verifierGateMu.Lock()
	defer verifierGateMu.Unlock()
	if verifierGate != nil {
		return verifierGate
	}
	gate := NewConfiguredSessionGate()
	verifierGate = gate
	startVerifierGateDispatcher(gate, db)
	return gate
}

// replaceVerifierGateForTest mengganti gate global selama pengujian dan menutup gate lama.
func replaceVerifierGateForTest(gate *UnifiedNVRSessionGate, startDefaultDispatcher bool) *UnifiedNVRSessionGate {
	verifierGateMu.Lock()
	prev := verifierGate
	verifierGate = gate
	verifierGateMu.Unlock()
	if prev != nil {
		prev.Close()
	}
	if gate != nil && startDefaultDispatcher {
		startVerifierGateDispatcher(gate, config.DB)
	}
	return gate
}

func rememberFirstFailure(cctvID uuid.UUID, at time.Time) {
	firstFailureMu.Lock()
	if _, exists := firstFailureMap[cctvID]; !exists {
		firstFailureMap[cctvID] = at
	}
	firstFailureMu.Unlock()
}

func consumeFirstFailure(cctvID uuid.UUID, fallback time.Time) time.Time {
	firstFailureMu.Lock()
	defer firstFailureMu.Unlock()
	if ts, ok := firstFailureMap[cctvID]; ok && !ts.IsZero() {
		delete(firstFailureMap, cctvID)
		return ts
	}
	return fallback
}

func clearFirstFailure(cctvID uuid.UUID) {
	firstFailureMu.Lock()
	delete(firstFailureMap, cctvID)
	firstFailureMu.Unlock()
}

func recordHeartbeatStart(camCount int, at time.Time) {
	heartbeatMu.Lock()
	tCopy := at
	lastVerifierCycleStartedAt = &tCopy
	lastVerifierCameraCount = camCount
	heartbeatMu.Unlock()
}

func recordHeartbeatComplete(at time.Time, mediaMTXHealthy bool, mediaMTXErr string) {
	heartbeatMu.Lock()
	tCopy := at
	lastVerifierCycleCompletedAt = &tCopy
	lastMediaMTXHealthy = mediaMTXHealthy
	lastMediaMTXError = mediaMTXErr
	heartbeatMu.Unlock()
}

// DispatchPostMaintenanceRefreshes memasukkan kamera yang jendela pemeliharaannya baru berakhir
// namun sampel terakhirnya berusia >75s atau masih unknown ke PostMaintenanceQueue pada Unified NVR Session Gate.
func DispatchPostMaintenanceRefreshes(db *gorm.DB, reqs []incidents.PlannedOutageRefreshRequest) {
	if db == nil || len(reqs) == 0 {
		return
	}
	gate := getOrInitVerifierGate(db)
	now := time.Now().UTC()
	for idx := range reqs {
		rCopy := reqs[idx]
		// Kamera ini sudah terkonfirmasi offline selama jendela; 1 kegagalan kamera nyata pasca-jendela
		// langsung bernilai offline, sementara kegagalan infrastruktur MediaMTX tetap bernilai unknown.
		if rCopy.Camera.StreamVerifyFailures < streamVerifyFailureThreshold-1 {
			rCopy.Camera.StreamVerifyFailures = streamVerifyFailureThreshold - 1
		}
		gate.Enqueue(GateJob{
			Priority:         GatePriorityPostMaintenance,
			CCTV:             rCopy.Camera,
			EnqueuedAt:       now,
			ReadyAt:          now,
			FirstFailedAt:    &rCopy.SuppressedFirstFailedAt,
			PostMaintRefresh: &rCopy,
		})
	}
}

func handleGateJobCompletion(db *gorm.DB, job GateJob, res streamVerificationResult) {
	if db == nil || !IncidentMonitoringEnabled() {
		return
	}
	ctx := context.Background()
	svc := incidents.Service{DB: db}

	// 1. Jika ini adalah pemeriksaan penyegaran 1x pasca-jendela pemeliharaan terjadwal:
	if job.PostMaintRefresh != nil {
		if _, err := svc.CompletePostMaintenanceRefresh(ctx, *job.PostMaintRefresh, res.State, res.VerifiedAt.UTC()); err != nil {
			utils.Log.Errorf("Gagal menyelesaikan evaluasi pasca-pemeliharaan kamera %s: %v", res.CCTV.KodeCCTV, err)
		}
		if !res.OpenedOnDemand {
			persistImmediateProbeTelemetry(db, res)
		}
		return
	}

	// 2. Jika baru kegagalan pertama (0 -> 1), jadwalkan konfirmasi cepat ke-2 setelah backoff 10s di ConfirmQueue!
	if res.Failures == 1 && res.State == models.StreamStateUnknown && !isMonitorInfraError(res.Error) {
		ff := res.VerifiedAt.UTC()
		rememberFirstFailure(res.CCTV.ID, ff)
		updatedCam := res.CCTV
		updatedCam.StreamVerifyFailures = 1
		updatedCam.StreamVerifiedState = res.State
		getOrInitVerifierGate(db).Enqueue(GateJob{
			Priority:      GatePriorityConfirm,
			CCTV:          updatedCam,
			EnqueuedAt:    ff,
			ReadyAt:       ff.Add(confirmRetryDelay),
			FirstFailedAt: &ff,
		})
	}

	if !res.OpenedOnDemand {
		persistImmediateProbeTelemetry(db, res)
	}
}

func confirmSourceClosedAndRecordTelemetry(db *gorm.DB, job GateJob, res streamVerificationResult) {
	if !IncidentMonitoringEnabled() {
		return
	}
	_ = evaluateAndPersistSourceClosureTelemetry(db, job, res)
}

// evaluateAndPersistSourceClosureTelemetry memverifikasi penutupan sumber MediaMTX berbasis bukti,
// mencatat hasil RemovePath, registrasi ulang, dan verifikasi penutupan secara terpisah, serta
// hanya menyetel SourceClosedConfirmed = true apabila pemeriksaan MediaMTX membuktikan sumber berhenti.
func evaluateAndPersistSourceClosureTelemetry(db *gorm.DB, job GateJob, res streamVerificationResult) incidents.ProbeTelemetry {
	path := utils.BuildStreamPath(job.CCTV.ID.String())
	closedConfirmed := false
	forcedClose := false
	totalBytes := res.BytesVerify
	removeStatus := RemovePathStatusNotNeeded
	removeErrText := ""
	reregisterStatus := ReregisterStatusNotNeeded
	reregisterErrText := ""
	var verifyStatus string
	unconfirmedReason := ""

	if sourceCloseConfirmDelay > 0 {
		// Beri margin kecil pasca-tail 5s agar timer sourceOnDemandCloseAfter MediaMTX tuntas menutup sesi RTSP.
		time.Sleep(sourceCloseConfirmDelay)
	}

	info, err := streamClient.GetPath(path)
	if err != nil {
		if isMediaMTXNotFound(err) {
			closedConfirmed = true
			verifyStatus = CloseVerifyPathAbsentConfirmed
		} else {
			closedConfirmed = false
			verifyStatus = CloseVerifyCheckFailed
			unconfirmedReason = "pemeriksaan awal penutupan sumber di MediaMTX gagal: " + err.Error()
		}
	} else {
		if info.BytesReceived > totalBytes {
			totalBytes = info.BytesReceived
		}
		if !info.SourceReady() {
			closedConfirmed = true
			verifyStatus = CloseVerifyClosedNaturally
		} else if len(info.Readers) > 0 {
			closedConfirmed = false
			verifyStatus = CloseVerifyActiveWithReaders
			unconfirmedReason = fmt.Sprintf("sumber masih terbuka karena terdapat %d pembaca aktif", len(info.Readers))
		} else if networkType(job.CCTV) != "gsm" {
			closedConfirmed = false
			verifyStatus = CloseVerifyStillActive
			unconfirmedReason = "sumber non-GSM masih aktif di MediaMTX setelah jendela penutupan"
		} else {
			// Guardrail kebocoran kuota GSM: bila path GSM masih menarik stream tanpa penonton,
			// paksa lepas path, daftar ulang on-demand bila kamera aktif, lalu verifikasi ulang ke MediaMTX.
			forcedClose = true
			if rmErr := streamClient.RemovePath(path); rmErr != nil {
				removeStatus = RemovePathStatusFailed
				removeErrText = rmErr.Error()
				reregisterStatus = ReregisterStatusSkipped

				if recheckInfo, recheckErr := streamClient.GetPath(path); recheckErr != nil {
					closedConfirmed = false
					verifyStatus = CloseVerifyCheckFailed
					unconfirmedReason = fmt.Sprintf("pelepasan path GSM gagal (%s) dan pemeriksaan ulang MediaMTX gagal: %s", rmErr.Error(), recheckErr.Error())
				} else {
					if recheckInfo.BytesReceived > totalBytes {
						totalBytes = recheckInfo.BytesReceived
					}
					if recheckInfo.SourceReady() {
						closedConfirmed = false
						verifyStatus = CloseVerifyStillActive
						unconfirmedReason = fmt.Sprintf("pelepasan path GSM gagal (%s) dan sumber masih aktif di MediaMTX", rmErr.Error())
					} else {
						closedConfirmed = true
						verifyStatus = CloseVerifyClosedNaturally
					}
				}
			} else {
				removeStatus = RemovePathStatusSucceeded
				shouldKeepRegistered := job.CCTV.StreamEnabled ||
					(IncidentMonitoringEnabled() && job.CCTV.MonitoringEnabled && strings.ToLower(strings.TrimSpace(job.CCTV.Status)) != "nonaktif")
				if shouldKeepRegistered {
					if regErr := registerStreamWithRetry(job.CCTV, 1); regErr != nil {
						reregisterStatus = ReregisterStatusFailed
						reregisterErrText = regErr.Error()
					} else {
						reregisterStatus = ReregisterStatusSucceeded
					}
				} else {
					reregisterStatus = ReregisterStatusNotNeeded
				}

				// Verifikasi wajib ke MediaMTX pasca-pelepasan untuk membuktikan sumber benar-benar berhenti.
				recheckInfo, recheckErr := streamClient.GetPath(path)
				if recheckErr != nil {
					if isMediaMTXNotFound(recheckErr) && (!shouldKeepRegistered || reregisterStatus == ReregisterStatusFailed) {
						closedConfirmed = true
						verifyStatus = CloseVerifyPathAbsentConfirmed
					} else {
						closedConfirmed = false
						verifyStatus = CloseVerifyCheckFailed
						unconfirmedReason = "pemeriksaan ulang penutupan sumber di MediaMTX gagal: " + recheckErr.Error()
					}
				} else {
					if recheckInfo.BytesReceived > totalBytes {
						totalBytes = recheckInfo.BytesReceived
					}
					if !recheckInfo.SourceReady() {
						closedConfirmed = true
						verifyStatus = CloseVerifyClosedAfterForcedRemove
					} else {
						closedConfirmed = false
						verifyStatus = CloseVerifyStillActive
						unconfirmedReason = "sumber GSM masih aktif (SourceReady=true) setelah pelepasan paksa"
					}
				}
			}
		}
	}

	lifetimeMs := res.Duration.Milliseconds() + defaultGateSourceCloseTail.Milliseconds()
	tel := incidents.ProbeTelemetry{
		ID:                     uuid.New(),
		CCTVID:                 job.CCTV.ID,
		SiteID:                 job.CCTV.SiteID,
		NetworkType:            networkType(job.CCTV),
		IPNVR:                  job.CCTV.IPNVR,
		State:                  res.State,
		Method:                 res.Method,
		VerifyDurationMs:       res.Duration.Milliseconds(),
		SourceLifetimeMs:       lifetimeMs,
		BytesVerify:            res.BytesVerify,
		BytesTotalSession:      totalBytes,
		ByteDelta:              res.ByteDelta,
		SourceClosedConfirmed:  closedConfirmed,
		ForcedClose:            forcedClose,
		RemovePathStatus:       removeStatus,
		RemovePathError:        removeErrText,
		ReregisterStatus:       reregisterStatus,
		ReregisterError:        reregisterErrText,
		CloseVerifyStatus:      verifyStatus,
		CloseUnconfirmedReason: unconfirmedReason,
		CreatedAt:              res.VerifiedAt.UTC(),
	}
	if db != nil {
		if err := db.Create(&tel).Error; err != nil {
			utils.Log.Debugf("Gagal menyimpan telemetri probe CCTV %s: %v", job.CCTV.KodeCCTV, err)
		}
	}
	return tel
}

func persistImmediateProbeTelemetry(db *gorm.DB, res streamVerificationResult) {
	if db == nil || !IncidentMonitoringEnabled() {
		return
	}
	verifyStatus := CloseVerifyNotOpenedOnDemand
	unconfirmedReason := ""
	if res.OpenedOnDemand {
		verifyStatus = CloseVerifyCheckFailed
		unconfirmedReason = "sesi on-demand dibuka tanpa konfirmasi penutupan tail"
	}
	tel := incidents.ProbeTelemetry{
		ID:                     uuid.New(),
		CCTVID:                 res.CCTV.ID,
		SiteID:                 res.CCTV.SiteID,
		NetworkType:            networkType(res.CCTV),
		IPNVR:                  res.CCTV.IPNVR,
		State:                  res.State,
		Method:                 res.Method,
		VerifyDurationMs:       res.Duration.Milliseconds(),
		SourceLifetimeMs:       res.Duration.Milliseconds(),
		BytesVerify:            res.BytesVerify,
		BytesTotalSession:      res.BytesVerify,
		ByteDelta:              res.ByteDelta,
		SourceClosedConfirmed:  !res.OpenedOnDemand,
		RemovePathStatus:       RemovePathStatusNotNeeded,
		ReregisterStatus:       ReregisterStatusNotNeeded,
		CloseVerifyStatus:      verifyStatus,
		CloseUnconfirmedReason: unconfirmedReason,
		CreatedAt:              res.VerifiedAt.UTC(),
	}
	_ = db.Create(&tel).Error
}

func isMonitorInfraError(errMsg string) bool {
	if errMsg == "" {
		return false
	}
	low := strings.ToLower(errMsg)
	return strings.Contains(low, "mediamtx") ||
		strings.Contains(low, "pendaftaran ulang") ||
		strings.Contains(low, "gagal memicu hls") ||
		strings.Contains(low, "gagal memicu rtsp")
}

// RunStreamVerifier menjalankan verifikasi end-to-end secara periodik.
func RunStreamVerifier(db *gorm.DB) {
	if db == nil {
		return
	}

	time.Sleep(20 * time.Second)
	if waitForMediaMTX(2 * time.Minute) {
		ReRegisterFailedStreams(db)
	} else {
		utils.Log.Warn("⚠️ MediaMTX belum siap; verifikasi stream tetap dijalankan dan akan melaporkan unknown bila perlu")
	}

	runStreamVerifyCycle(db)

	interval := streamVerifyInterval()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		runStreamVerifyCycle(db)
	}
}

func streamVerifyInterval() time.Duration {
	if IncidentMonitoringEnabled() {
		seconds, err := strconv.Atoi(os.Getenv("CCTV_INCIDENT_INTERVAL_SECONDS"))
		if err != nil || seconds < 30 || seconds > 120 {
			seconds = 60
		}
		return time.Duration(seconds) * time.Second
	}
	raw := strings.TrimSpace(os.Getenv("CCTV_VERIFY_INTERVAL_MINUTES"))
	if raw == "" {
		return defaultStreamVerifyInterval
	}
	minutes, err := strconv.Atoi(raw)
	if err != nil || minutes < 5 || minutes > 30 {
		return defaultStreamVerifyInterval
	}
	return time.Duration(minutes) * time.Minute
}

func streamVerifyConcurrency() int {
	raw := strings.TrimSpace(os.Getenv("CCTV_VERIFY_CONCURRENCY"))
	if raw == "" {
		return defaultStreamVerifyConcurrency
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 || n > 4 {
		return defaultStreamVerifyConcurrency
	}
	return n
}

// runStreamVerifyCycle memverifikasi seluruh CCTV aktif.
// Saat IncidentMonitoringEnabled() aktif, seluruh pemeriksaan melewati Unified NVR Session Gate,
// mendukung korelasi gangguan bersama NVR (ObserveSharedCorrelatedWave), serta deduplikasi episode
// gangguan pemantau (ReportMonitorFailure / ReportMonitorHealthyCycle).
func runStreamVerifyCycle(db *gorm.DB) {
	if db == nil {
		return
	}

	var cctvs []models.CCTV
	q := db.Where("stream_enabled = ?", true)
	if IncidentMonitoringEnabled() {
		q = db.Where("LOWER(status) <> 'nonaktif' AND (stream_enabled = ? OR monitoring_enabled = ?)", true, true)
	}
	if err := q.Find(&cctvs).Error; err != nil {
		utils.Log.Errorf("❌ Gagal memuat CCTV untuk verifikasi stream: %v", err)
		return
	}
	if len(cctvs) == 0 {
		return
	}

	started := time.Now().UTC()
	recordHeartbeatStart(len(cctvs), started)

	if !IncidentMonitoringEnabled() {
		jobs := make(chan models.CCTV)
		results := make(chan streamVerificationResult, len(cctvs))
		var wg sync.WaitGroup
		for i := 0; i < streamVerifyConcurrency(); i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for cctv := range jobs {
					results <- verifyCCTVStream(db, cctv)
					time.Sleep(250 * time.Millisecond)
				}
			}()
		}
		for _, cctv := range cctvs {
			jobs <- cctv
		}
		close(jobs)
		wg.Wait()
		close(results)

		live, offline, unknown := 0, 0, 0
		for r := range results {
			switch r.State {
			case models.StreamStateLive:
				live++
			case models.StreamStateOffline:
				offline++
			default:
				unknown++
			}
		}
		recordHeartbeatComplete(time.Now().UTC(), true, "")
		utils.Log.Infof("🔎 Verifikasi stream selesai: %d kamera | live=%d offline=%d unknown=%d | durasi=%s",
			len(cctvs), live, offline, unknown, time.Since(started).Round(time.Millisecond))
		return
	}

	gate := getOrInitVerifierGate(db)
	notifyChans := make([]chan streamVerificationResult, 0, len(cctvs))
	for _, cctv := range cctvs {
		ch := make(chan streamVerificationResult, 1)
		var ffPtr *time.Time
		firstFailureMu.Lock()
		if ts, ok := firstFailureMap[cctv.ID]; ok {
			tCopy := ts
			ffPtr = &tCopy
		}
		firstFailureMu.Unlock()

		enqueued := gate.Enqueue(GateJob{
			Priority:             GatePriorityMain,
			CCTV:                 cctv,
			EnqueuedAt:           started,
			ReadyAt:              started,
			FirstFailedAt:        ffPtr,
			CompletionNotifyChan: ch,
		})
		if enqueued {
			notifyChans = append(notifyChans, ch)
		}
	}

	collected := make([]streamVerificationResult, 0, len(notifyChans))
	for _, ch := range notifyChans {
		res := <-ch
		collected = append(collected, res)
	}

	processMonitoringCycleSummary(db, collected)
}

func processMonitoringCycleSummary(db *gorm.DB, results []streamVerificationResult) {
	now := time.Now().UTC()
	svc := incidents.Service{DB: db}

	live, offline, unknown := 0, 0, 0
	siteInfraFailures := map[uuid.UUID]struct {
		count  int
		detail string
	}{}
	sitesSeen := map[uuid.UUID]bool{}

	type nvrKey struct {
		SiteID uuid.UUID
		IPNVR  string
	}
	nvrOfflineGroups := map[nvrKey][]incidents.CameraObservation{}

	for _, r := range results {
		switch r.State {
		case models.StreamStateLive:
			live++
		case models.StreamStateOffline:
			offline++
		default:
			unknown++
		}
		if r.CCTV.SiteID != nil && *r.CCTV.SiteID != uuid.Nil {
			sid := *r.CCTV.SiteID
			sitesSeen[sid] = true
			if isMonitorInfraError(r.Error) {
				cur := siteInfraFailures[sid]
				cur.count++
				if cur.detail == "" {
					cur.detail = r.Error
				}
				siteInfraFailures[sid] = cur
			}
			if r.State == models.StreamStateOffline && strings.TrimSpace(r.CCTV.IPNVR) != "" {
				ff := r.VerifiedAt.UTC()
				if r.FirstFailedAt != nil && !r.FirstFailedAt.IsZero() {
					ff = *r.FirstFailedAt
				}
				key := nvrKey{SiteID: sid, IPNVR: strings.TrimSpace(r.CCTV.IPNVR)}
				nvrOfflineGroups[key] = append(nvrOfflineGroups[key], incidents.CameraObservation{
					Camera:                r.CCTV,
					State:                 models.StreamStateOffline,
					ObservedAt:            r.VerifiedAt.UTC(),
					FirstFailedObservedAt: &ff,
					Evidence: map[string]interface{}{
						"method": r.Method,
						"error":  r.Error,
					},
				})
			}
		}
	}

	ctx := context.Background()
	anyInfraErr := ""
	for sid := range sitesSeen {
		if failInfo, hasFail := siteInfraFailures[sid]; hasFail {
			if anyInfraErr == "" {
				anyInfraErr = failInfo.detail
			}
			siteCopy := sid
			_, _, _ = svc.ReportMonitorFailure(ctx, &siteCopy, "mediamtx_unreachable", failInfo.detail, failInfo.count, now)
		} else {
			_, _ = svc.ReportMonitorHealthyCycle(ctx, sid, now)
		}
	}

	// Korelasi gelombang NVR bila >= 2 kamera pada site & IP NVR yang sama gagal bersamaan dan NVR tidak merespons RTSP TCP:554.
	for key, obsList := range nvrOfflineGroups {
		if len(obsList) < 2 {
			continue
		}
		if _, infraBroken := siteInfraFailures[key.SiteID]; infraBroken {
			continue
		}
		nvrDead := !probeNVRRTSPHost(key.IPNVR)
		if nvrDead {
			_ = svc.ObserveSharedCorrelatedWave(ctx, key.SiteID, key.IPNVR, true, obsList)
		}
	}

	recordHeartbeatComplete(now, anyInfraErr == "", anyInfraErr)
	utils.Log.Infof("🔎 Verifikasi stream terpadu selesai: %d kamera | live=%d offline=%d unknown=%d",
		len(results), live, offline, unknown)
}

// verifyCCTVStream mempertahankan tanda tangan fungsi yang dipakai pemanggil & unit test eksisting.
func verifyCCTVStream(db *gorm.DB, cctv models.CCTV) streamVerificationResult {
	return verifyCCTVStreamInternal(db, cctv, false, nil)
}

func verifyCCTVStreamInternal(db *gorm.DB, cctv models.CCTV, suppressObserve bool, priorFirstFailed *time.Time) streamVerificationResult {
	return verifyCCTVStreamWithOptions(db, cctv, suppressObserve, IncidentMonitoringEnabled() || streamVerifyRequireBytes, priorFirstFailed)
}

// verifyCCTVStreamWithOptions melakukan verifikasi 4-lapis:
//  1. MediaMTX terjangkau & path terdaftar (atau diregistrasi ulang otomatis 1x).
//  2. Sumber RTSP siap (SourceReady) dan memiliki >= 1 track media.
//  3. Saat requireByteProgress bernilai true, memverifikasi kemajuan paket nyata
//     (Delta BytesReceived > 0 dalam streamVerifyByteWindow) untuk menangkap stream beku (stalled RTP stream).
//  4. Mencatat pemisahan kegagalan pertama (FirstFailedAt) dari konfirmasi ke-2 (VerifiedAt).
func verifyCCTVStreamWithOptions(db *gorm.DB, cctv models.CCTV, suppressObserve, requireByteProgress bool, priorFirstFailed *time.Time) streamVerificationResult {
	started := time.Now()
	path := utils.BuildStreamPath(cctv.ID.String())
	initialMethod := streamVerifyMethodHLS
	if streamClient != nil && streamClient.RTSPBaseURL() != "" {
		initialMethod = streamVerifyMethodRTSP
	}
	result := streamVerificationResult{
		CCTV:            cctv,
		State:           models.StreamStateUnknown,
		VerifiedAt:      started,
		Method:          initialMethod,
		Failures:        cctv.StreamVerifyFailures,
		SuppressObserve: suppressObserve,
		FirstFailedAt:   priorFirstFailed,
	}

	info, err := streamClient.GetPath(path)
	if err != nil {
		if !isMediaMTXNotFound(err) {
			result.Error = "MediaMTX tidak dapat dihubungi"
			result.Duration = time.Since(started)
			return persistStreamVerification(db, result)
		}

		utils.Log.Infof("🔁 Path streaming %s belum terdaftar; mendaftarkan ulang sebelum verifikasi", path)
		if regErr := registerStreamWithRetry(cctv, 1); regErr != nil {
			result.Error = "path stream belum terdaftar di MediaMTX; pendaftaran ulang gagal"
			result.Duration = time.Since(started)
			return persistStreamVerification(db, result)
		}

		info, err = streamClient.GetPath(path)
		if err != nil {
			result.Error = "path stream belum terdaftar di MediaMTX setelah pendaftaran ulang"
			if !isMediaMTXNotFound(err) {
				result.Error = "MediaMTX tidak dapat dihubungi setelah pendaftaran ulang"
			}
			result.Duration = time.Since(started)
			return persistStreamVerification(db, result)
		}
	}

	checkByteGrowth := func(initial *utils.PathInfo) (ok bool, totalBytes, delta int64, mediaMTXErr bool) {
		if !requireByteProgress {
			return true, initial.BytesReceived, 0, false
		}
		b1 := initial.BytesReceived
		lastBytes := b1
		maxAttempts := 1
		if b1 == 0 {
			// Kamera RTSP yang baru dibuka on-demand dapat menunda IDR pertama hingga ~1s GOP setelah handshake PLAY.
			maxAttempts = 3
		}
		for attempt := 0; attempt < maxAttempts; attempt++ {
			time.Sleep(streamVerifyByteWindow)
			after, gErr := streamClient.GetPath(path)
			if gErr != nil {
				return false, lastBytes, 0, true
			}
			lastBytes = after.BytesReceived
			if !after.SourceReady() || len(after.Tracks) == 0 {
				return false, lastBytes, 0, false
			}
			d := after.BytesReceived - b1
			if d > 0 {
				return true, after.BytesReceived, d, false
			}
		}
		return false, lastBytes, 0, false
	}

	// Path yang sudah menyala tidak perlu dipicu ulang.
	if info.SourceReady() && (!requireByteProgress || len(info.Tracks) > 0) {
		okBytes, totalB, deltaB, mtxErr := checkByteGrowth(info)
		if mtxErr {
			result.Error = "MediaMTX tidak dapat dihubungi saat verifikasi byte"
			result.Duration = time.Since(started)
			return persistStreamVerification(db, result)
		}
		if okBytes {
			clearFirstFailure(cctv.ID)
			result.State = models.StreamStateLive
			result.Method = streamVerifyMethodLive
			result.Tracks = len(info.Tracks)
			result.Failures = 0
			result.BytesVerify = totalB
			result.ByteDelta = deltaB
			result.Duration = time.Since(started)
			return persistStreamVerification(db, result)
		}
		// Sumber tampak aktif namun 0 paket baru (stalled stream) -> perlakukan sebagai kegagalan kamera!
		result.Failures = cctv.StreamVerifyFailures + 1
		result.State = models.StreamStateUnknown
		if result.Failures >= streamVerifyFailureThreshold {
			result.State = models.StreamStateOffline
		}
		result.Error = "aliran RTP beku (0 byte baru selama jendela verifikasi)"
		result.BytesVerify = totalB
		result.Duration = time.Since(started)
		return persistStreamVerification(db, result)
	}

	method, triggerErr := streamClient.TriggerOnDemand(path, 2*time.Second)
	result.Method = method
	if triggerErr != nil {
		if method == streamVerifyMethodRTSP {
			result.Error = fmt.Sprintf("gagal memicu RTSP MediaMTX: %v", triggerErr)
		} else {
			result.Error = "gagal memicu HLS MediaMTX"
		}
		result.Duration = time.Since(started)
		return persistStreamVerification(db, result)
	}

	deadline := time.Now().Add(streamVerifyTimeout)
	var readyInfo *utils.PathInfo
	for {
		info, err = streamClient.GetPath(path)
		if err == nil && info.SourceReady() && (!requireByteProgress || len(info.Tracks) > 0) {
			readyInfo = info
			break
		}
		if time.Now().After(deadline) {
			break
		}
		time.Sleep(streamVerifyPollInterval)
	}

	if readyInfo == nil {
		info, err = streamClient.GetPath(path)
		if err != nil {
			result.Error = "MediaMTX tidak dapat dihubungi saat verifikasi"
			result.Duration = time.Since(started)
			return persistStreamVerification(db, result)
		}
		if info.SourceReady() && (!requireByteProgress || len(info.Tracks) > 0) {
			readyInfo = info
		}
	}

	if readyInfo != nil {
		result.OpenedOnDemand = true
		okBytes, totalB, deltaB, mtxErr := checkByteGrowth(readyInfo)
		if mtxErr {
			result.Error = "MediaMTX tidak dapat dihubungi saat verifikasi byte"
			result.Duration = time.Since(started)
			return persistStreamVerification(db, result)
		}
		if okBytes {
			clearFirstFailure(cctv.ID)
			result.State = models.StreamStateLive
			result.Tracks = len(readyInfo.Tracks)
			result.Failures = 0
			result.BytesVerify = totalB
			result.ByteDelta = deltaB
			result.Duration = time.Since(started)
			return persistStreamVerification(db, result)
		}
		result.Failures = cctv.StreamVerifyFailures + 1
		result.State = models.StreamStateUnknown
		if result.Failures >= streamVerifyFailureThreshold {
			result.State = models.StreamStateOffline
		}
		result.Error = "aliran RTP beku (0 byte baru setelah sumber terbuka)"
		result.BytesVerify = totalB
		result.Duration = time.Since(started)
		return persistStreamVerification(db, result)
	}

	result.Failures = cctv.StreamVerifyFailures + 1
	result.State = models.StreamStateUnknown
	if result.Failures >= streamVerifyFailureThreshold {
		result.State = models.StreamStateOffline
	}
	result.Error = fmt.Sprintf("sumber stream tidak siap setelah %s", streamVerifyTimeout)
	result.Duration = time.Since(started)
	return persistStreamVerification(db, result)
}

func persistStreamVerification(db *gorm.DB, result streamVerificationResult) streamVerificationResult {
	if db == nil {
		return result
	}

	updates := map[string]interface{}{
		"stream_verified_state":  result.State,
		"stream_verified_at":     result.VerifiedAt,
		"stream_verified_method": result.Method,
		"stream_verify_failures": result.Failures,
		"stream_verify_error":    result.Error,
	}
	if result.State == models.StreamStateLive {
		updates["stream_last_active_at"] = result.VerifiedAt
	}

	if err := db.Model(&models.CCTV{}).Where("id = ?", result.CCTV.ID).Updates(updates).Error; err != nil {
		utils.Log.Errorf("❌ Gagal menyimpan hasil verifikasi stream CCTV %s: %v", result.CCTV.ID, err)
		return result
	}

	if IncidentMonitoringEnabled() && !result.SuppressObserve {
		verifiedUTC := result.VerifiedAt.UTC()
		firstFailed := verifiedUTC
		if result.State == models.StreamStateOffline {
			if result.FirstFailedAt != nil && !result.FirstFailedAt.IsZero() {
				firstFailed = result.FirstFailedAt.UTC()
				clearFirstFailure(result.CCTV.ID)
			} else {
				firstFailed = consumeFirstFailure(result.CCTV.ID, verifiedUTC)
			}
		}
		result.FirstFailedAt = &firstFailed

		obs := incidents.CameraObservation{
			Camera:                result.CCTV,
			State:                 result.State,
			ObservedAt:            verifiedUTC,
			FirstFailedObservedAt: &firstFailed,
			Evidence: map[string]interface{}{
				"method":       result.Method,
				"tracks":       result.Tracks,
				"bytes_verify": result.BytesVerify,
				"byte_delta":   result.ByteDelta,
				"duration_ms":  result.Duration.Milliseconds(),
				"error":        result.Error,
			},
		}

		// Jika kamera terkonfirmasi offline dan sudah ada tiket bersama NVR terbuka di site & IP NVR yang sama
		// dengan NVR tidak merespons RTSP TCP:554, gabungkan otomatis ke tiket bersama tersebut.
		handledShared := false
		if result.State == models.StreamStateOffline && result.CCTV.SiteID != nil && strings.TrimSpace(result.CCTV.IPNVR) != "" {
			var openSharedCount int64
			_ = db.Model(&incidents.Incident{}).
				Where("site_id = ? AND correlation_key = ? AND status NOT IN ('resolved', 'merged')",
					*result.CCTV.SiteID, "nvr:"+strings.TrimSpace(result.CCTV.IPNVR)).
				Count(&openSharedCount).Error
			if openSharedCount > 0 && !probeNVRRTSPHost(strings.TrimSpace(result.CCTV.IPNVR)) {
				if err := (incidents.Service{DB: db}).ObserveSharedCorrelatedWave(context.Background(), *result.CCTV.SiteID, result.CCTV.IPNVR, true, []incidents.CameraObservation{obs}); err == nil {
					handledShared = true
				}
			}
		}

		if !handledShared {
			if err := (incidents.Service{DB: db}).ObserveDetailed(context.Background(), obs); err != nil {
				utils.Log.Errorf("Gagal memproses insiden CCTV %s: %v", result.CCTV.ID, err)
			}
		}
	}

	siteID := cctvSiteIDStr(result.CCTV)
	stateChanged := result.State != result.CCTV.StreamVerifiedState
	if stateChanged {
		var eventType string
		var eventMsg string
		switch result.State {
		case models.StreamStateLive:
			eventType = models.CCTVEventOnline
			eventMsg = fmt.Sprintf("Verifikasi stream berhasil (%s, track=%d)", result.Method, result.Tracks)
		case models.StreamStateOffline:
			eventType = models.CCTVEventOffline
			eventMsg = "Verifikasi stream gagal"
			if result.Error != "" {
				eventMsg += ": " + result.Error
			}
		default:
			eventType = models.CCTVEventUnknown
			eventMsg = "Hasil verifikasi stream tidak diketahui: " + result.Error
		}

		if shouldLogCCTVStreamEvent(db, result.CCTV.ID, eventType, result.VerifiedAt) {
			streamLog := models.CCTVStreamLog{
				CCTVID:        result.CCTV.ID,
				EventType:     eventType,
				PreviousState: result.CCTV.StreamVerifiedState,
				CurrentState:  result.State,
				TriggerSource: "verify",
				NetworkType:   networkType(result.CCTV),
				ErrorDetail:   result.Error,
				Message:       eventMsg,
			}
			if logErr := db.Create(&streamLog).Error; logErr != nil {
				utils.Log.Warnf("⚠️ Gagal mencatat log verifikasi stream CCTV %s: %v", result.CCTV.KodeCCTV, logErr)
			} else {
				utils.BroadcastToSite(siteID, "CCTV_STREAM_LOG_CREATED", fiber.Map{
					"id":             streamLog.ID.String(),
					"cctv_id":        result.CCTV.ID.String(),
					"site_id":        siteID,
					"nama":           result.CCTV.Nama,
					"kode_cctv":      result.CCTV.KodeCCTV,
					"event_type":     eventType,
					"previous_state": result.CCTV.StreamVerifiedState,
					"current_state":  result.State,
					"trigger_source": "verify",
					"created_at":     result.VerifiedAt.Format(time.RFC3339),
					"message":        eventMsg,
				})
			}
		}
	}

	payload := fiber.Map{
		"cctv_id":                result.CCTV.ID.String(),
		"site_id":                siteID,
		"nama":                   result.CCTV.Nama,
		"kode_cctv":              result.CCTV.KodeCCTV,
		"network_type":           networkType(result.CCTV),
		"stream_state":           result.CCTV.StreamState,
		"stream_live":            result.CCTV.StreamLive,
		"stream_verified_state":  result.State,
		"stream_verified_at":     result.VerifiedAt.Format(time.RFC3339),
		"stream_verified_method": result.Method,
		"stream_verify_failures": result.Failures,
		"stream_verify_error":    result.Error,
	}
	if result.CCTV.StreamCheckedAt != nil {
		payload["stream_checked_at"] = result.CCTV.StreamCheckedAt.Format(time.RFC3339)
	}
	if result.State == models.StreamStateLive {
		payload["stream_last_active_at"] = result.VerifiedAt.Format(time.RFC3339)
	} else if result.CCTV.StreamLastActiveAt != nil {
		payload["stream_last_active_at"] = result.CCTV.StreamLastActiveAt.Format(time.RFC3339)
	}
	utils.BroadcastToSite(siteID, "CCTV_LIVE_STATUS_UPDATED", payload)

	switch result.State {
	case models.StreamStateLive:
		utils.Log.Infof("✅ Verifikasi stream CCTV %s (%s): live (%s, track=%d, %s)",
			result.CCTV.Nama, result.CCTV.KodeCCTV, result.Method, result.Tracks, result.Duration.Round(time.Millisecond))
	case models.StreamStateOffline:
		utils.Log.Warnf("🔴 Verifikasi stream CCTV %s (%s): offline setelah %d kegagalan",
			result.CCTV.Nama, result.CCTV.KodeCCTV, result.Failures)
	default:
		utils.Log.Debugf("⚪ Verifikasi stream CCTV %s (%s): %s (%s)",
			result.CCTV.Nama, result.CCTV.KodeCCTV, result.State, result.Error)
	}

	return result
}

func isMediaMTXNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "status 404") || strings.Contains(msg, "path not found")
}

// VerifyStream menjalankan verifikasi end-to-end satu CCTV secara manual (Admin saja)
// melalui UnifiedNVRSessionGate yang sama dengan pemeriksaan terjadwal.
func VerifyStream(c *fiber.Ctx) error {
	if config.DB == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Database belum tersedia",
		})
	}
	if role, _ := c.Locals("role").(string); role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "error",
			"message": "Akses ditolak: hanya Admin yang diizinkan menjalankan verifikasi stream manual",
		})
	}

	cctv, ok := loadCCTVByID(c)
	if !ok {
		return nil
	}
	if !cctv.StreamEnabled && !cctv.MonitoringEnabled {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Streaming belum diaktifkan untuk CCTV ini",
		})
	}

	gate := getOrInitVerifierGate(config.DB)
	waitTimeout := resolveManualVerifyWaitTimeout()
	waitCtx, cancel := context.WithTimeout(c.UserContext(), waitTimeout)
	defer cancel()

	activeDB := config.DB
	result, err := gate.ExecuteInteractive(waitCtx, GateJob{
		CCTV:                  cctv,
		SkipCompletionHandler: true,
		JobProbe: func(_ context.Context, job GateJob) (streamVerificationResult, bool) {
			res := verifyCCTVStreamWithOptions(activeDB, job.CCTV, false, IncidentMonitoringEnabled() || streamVerifyRequireBytes, nil)
			if !res.OpenedOnDemand {
				persistImmediateProbeTelemetry(activeDB, res)
			}
			return res, res.OpenedOnDemand
		},
	})
	if err != nil {
		timedOut := errors.Is(err, ErrGateWaitTimeout)
		msg := "Layanan antrean verifikasi stream sedang tidak tersedia"
		if timedOut {
			msg = fmt.Sprintf("Batas sesi verifikasi NVR/GSM sedang penuh (menunggu > %s); verifikasi dibatalkan sebelum membuka stream baru", waitTimeout.Round(time.Millisecond))
		}
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"status":  "error",
			"message": msg,
			"data": fiber.Map{
				"cctv_id":        cctv.ID.String(),
				"network_type":   networkType(cctv),
				"wait_timed_out": timedOut,
				"session_gate":   gate.Snapshot(),
			},
		})
	}

	message := "Status stream belum dapat diverifikasi"
	switch result.State {
	case models.StreamStateLive:
		message = "Stream terverifikasi live"
	case models.StreamStateOffline:
		message = "Stream terverifikasi offline"
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": message,
		"data": fiber.Map{
			"cctv_id":                cctv.ID.String(),
			"kode_cctv":              cctv.KodeCCTV,
			"nama":                   cctv.Nama,
			"network_type":           networkType(cctv),
			"stream_state":           result.CCTV.StreamState,
			"stream_live":            result.CCTV.StreamLive,
			"stream_verified_state":  result.State,
			"stream_verified_at":     result.VerifiedAt,
			"stream_verified_method": result.Method,
			"stream_verify_failures": result.Failures,
			"stream_verify_error":    result.Error,
			"tracks":                 result.Tracks,
			"bytes_verify":           result.BytesVerify,
			"byte_delta":             result.ByteDelta,
			"duration_ms":            result.Duration.Milliseconds(),
		},
	})
}

// GetStreamMonitorHeartbeat menyediakan status kesehatan proses pemantau & gate sesi NVR
// bagi pemantau eksternal di luar server SIGAP (menggunakan API key ber-scope health:read
// maupun sesi Admin — Keputusan #6).
func GetStreamMonitorHeartbeat(c *fiber.Ctx) error {
	authMethod, _ := c.Locals("auth_method").(string)
	if authMethod != "api_key" {
		role, _ := c.Locals("role").(string)
		if role != "admin" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"status":  "error",
				"message": "Akses ditolak: hanya Admin atau API key ber-scope health:read yang diizinkan",
			})
		}
	}

	now := time.Now().UTC()
	heartbeatMu.Lock()
	startedAt := lastVerifierCycleStartedAt
	completedAt := lastVerifierCycleCompletedAt
	camCount := lastVerifierCameraCount
	mtxHealthy := lastMediaMTXHealthy
	mtxErr := lastMediaMTXError
	heartbeatMu.Unlock()

	var gateSnap GateSnapshot
	verifierGateMu.Lock()
	if verifierGate != nil {
		gateSnap = verifierGate.Snapshot()
	}
	verifierGateMu.Unlock()

	maxStale := 3 * streamVerifyInterval()
	if maxStale < 3*time.Minute {
		maxStale = 3 * time.Minute
	}
	stale := completedAt != nil && now.Sub(*completedAt) > maxStale
	status := "healthy"
	if !mtxHealthy || stale {
		status = "degraded"
	}

	var openEpisodes int64
	if config.DB != nil && IncidentMonitoringEnabled() {
		_ = config.DB.Model(&incidents.MonitorHealthEpisode{}).Where("status = 'open'").Count(&openEpisodes).Error
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"monitor_status":              status,
			"incident_monitoring_enabled": IncidentMonitoringEnabled(),
			"mediamtx_healthy":            mtxHealthy,
			"mediamtx_last_error":         mtxErr,
			"open_monitor_episodes":       openEpisodes,
			"last_cycle_started_at":       startedAt,
			"last_cycle_completed_at":     completedAt,
			"last_cycle_camera_count":     camCount,
			"verifier_stale":              stale,
			"session_gate":                gateSnap,
			"checked_at":                  now,
		},
	})
}

func isStreamVerificationFresh(cctv models.CCTV, now time.Time) bool {
	if cctv.StreamVerifiedAt == nil {
		return false
	}
	age := now.Sub(*cctv.StreamVerifiedAt)
	if age < 0 || age > streamVerifyFreshness {
		return false
	}
	return cctv.StreamVerifiedState == models.StreamStateLive || cctv.StreamVerifiedState == models.StreamStateOffline
}

func effectiveStreamState(cctv models.CCTV, now time.Time) string {
	if isStreamVerificationFresh(cctv, now) {
		return cctv.StreamVerifiedState
	}

	if cctv.StreamState != "" {
		return cctv.StreamState
	}
	switch {
	case cctv.StreamLive:
		return models.StreamStateLive
	case cctv.StreamCheckedAt != nil:
		return models.StreamStateOffline
	default:
		return models.StreamStateUnknown
	}
}
