package handlers

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"sigap-backend/incidents"
	"sigap-backend/models"
)

type GatePriority int

const (
	GatePriorityConfirm GatePriority = iota
	GatePriorityInteractive
	GatePriorityPostMaintenance
	GatePriorityMain
)

var (
	ErrGateWaitTimeout = errors.New("waktu tunggu slot sesi verifikasi NVR/GSM habis")
	ErrGateClosed      = errors.New("gate sesi verifikasi sedang ditutup")
)

const (
	defaultGateMaxUpstreamSlots = 10
	defaultGateMaxGSMConcurrent = 2
	defaultGatePaceInterval     = 1250 * time.Millisecond
	defaultGateSourceCloseTail  = 5 * time.Second
	defaultConfirmRetryDelay    = 10 * time.Second
)

type GateJob struct {
	jobID                 uuid.UUID
	Priority              GatePriority
	CCTV                  models.CCTV
	EnqueuedAt            time.Time
	ReadyAt               time.Time
	FirstFailedAt         *time.Time
	PostMaintRefresh      *incidents.PlannedOutageRefreshRequest
	DeferOfflineObserve   bool
	ConfirmTracker        *cycleConfirmTracker
	JobProbe              GateProbeFunc
	SkipCompletionHandler bool
	slotAcquiredChan      chan struct{}
	CompletionNotifyChan  chan streamVerificationResult
}

type GateSnapshot struct {
	ActiveVerifications int `json:"active_verifications"`
	ClosingTailLeases   int `json:"closing_tail_leases"`
	TotalOccupiedSlots  int `json:"total_occupied_slots"`
	ActiveGSMSlots      int `json:"active_gsm_slots"`
	PeakOccupiedSlots   int `json:"peak_occupied_slots"`
	QueuedConfirm       int `json:"queued_confirm"`
	QueuedInteractive   int `json:"queued_interactive"`
	QueuedPostMaint     int `json:"queued_post_maintenance"`
	QueuedMain          int `json:"queued_main"`
}

// GateProbeFunc menjalankan 1 pemeriksaan kamera dan mengembalikan hasil verifikasi
// beserta penanda apakah sesi on-demand dibuka (sehingga slot wajib menahan tail lease
// selama SourceCloseTail agar sesi NVR gabungan tidak melampaui MaxUpstreamSlots).
type GateProbeFunc func(ctx context.Context, job GateJob) (res streamVerificationResult, openedOnDemand bool)

// GateTailReleaseHook dipanggil opsional saat tail lease berakhir (mis. untuk konfirmasi
// penutupan sumber MediaMTX & pencatatan telemetri GSM).
type GateTailReleaseHook func(job GateJob, res streamVerificationResult)

// UnifiedNVRSessionGate menyatukan ConfirmQueue, PostMaintenanceQueue, dan MainQueue
// di bawah satu pacer tunggal dan satu semaphore lease sesi upstream gabungan
// (active_verifications + closing_tail_leases <= MaxUpstreamSlots) sesuai Koreksi #2.
type UnifiedNVRSessionGate struct {
	MaxUpstreamSlots int
	MaxGSMConcurrent int
	PaceInterval     time.Duration
	SourceCloseTail  time.Duration

	mu                  sync.Mutex
	cond                *sync.Cond
	confirmQueue        []GateJob
	interactiveQueue    []GateJob
	postMaintQueue      []GateJob
	mainQueue           []GateJob
	inFlight            map[uuid.UUID]bool
	queuedPriority      map[uuid.UUID]GatePriority
	activeVerifications int
	closingTailLeases   int
	activeGSMSlots      int
	peakOccupiedSlots   int
	lastStartAt         time.Time
	closed              bool
	closedChan          chan struct{}
}

func NewUnifiedNVRSessionGate(maxSlots, maxGSM int, pace, tail time.Duration) *UnifiedNVRSessionGate {
	if maxSlots <= 0 {
		maxSlots = defaultGateMaxUpstreamSlots
	}
	if maxGSM <= 0 {
		maxGSM = defaultGateMaxGSMConcurrent
	}
	if maxGSM > maxSlots {
		maxGSM = maxSlots
	}
	if pace < 0 {
		pace = defaultGatePaceInterval
	}
	if tail < 0 {
		tail = defaultGateSourceCloseTail
	}
	g := &UnifiedNVRSessionGate{
		MaxUpstreamSlots: maxSlots,
		MaxGSMConcurrent: maxGSM,
		PaceInterval:     pace,
		SourceCloseTail:  tail,
		inFlight:         make(map[uuid.UUID]bool),
		queuedPriority:   make(map[uuid.UUID]GatePriority),
		closedChan:       make(chan struct{}),
	}
	g.cond = sync.NewCond(&g.mu)
	return g
}

func NewConfiguredSessionGate() *UnifiedNVRSessionGate {
	maxSlots := defaultGateMaxUpstreamSlots
	if raw := strings.TrimSpace(os.Getenv("CCTV_MAX_UPSTREAM_SLOTS")); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v >= 1 && v <= 64 {
			maxSlots = v
		}
	}
	maxGSM := defaultGateMaxGSMConcurrent
	if raw := strings.TrimSpace(os.Getenv("CCTV_GSM_MAX_CONCURRENT")); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v >= 1 && v <= 10 {
			maxGSM = v
		}
	}
	pace := defaultGatePaceInterval
	if raw := strings.TrimSpace(os.Getenv("CCTV_VERIFY_PACE_MS")); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v >= 100 && v <= 10000 {
			pace = time.Duration(v) * time.Millisecond
		}
	}
	return NewUnifiedNVRSessionGate(maxSlots, maxGSM, pace, defaultGateSourceCloseTail)
}

func (g *UnifiedNVRSessionGate) Snapshot() GateSnapshot {
	g.mu.Lock()
	defer g.mu.Unlock()
	return GateSnapshot{
		ActiveVerifications: g.activeVerifications,
		ClosingTailLeases:   g.closingTailLeases,
		TotalOccupiedSlots:  g.activeVerifications + g.closingTailLeases,
		ActiveGSMSlots:      g.activeGSMSlots,
		PeakOccupiedSlots:   g.peakOccupiedSlots,
		QueuedConfirm:       len(g.confirmQueue),
		QueuedInteractive:   len(g.interactiveQueue),
		QueuedPostMaint:     len(g.postMaintQueue),
		QueuedMain:          len(g.mainQueue),
	}
}

// ExecuteInteractive mengantrekan permintaan interaktif Admin (VerifyStream / ReactivateAndVerifyCCTV)
// ke UnifiedNVRSessionGate yang sama dengan pemeriksaan terjadwal, sehingga batas sesi gabungan
// (MaxUpstreamSlots) dan batas GSM (MaxGSMConcurrent) tidak pernah dilampaui.
// Jika hingga batas waitCtx berakhir slot belum tersedia, permintaan dibatalkan dari antrean
// tanpa pernah membuka stream dan mengembalikan ErrGateWaitTimeout.
func (g *UnifiedNVRSessionGate) ExecuteInteractive(waitCtx context.Context, job GateJob) (streamVerificationResult, error) {
	if job.CCTV.ID == uuid.Nil {
		return streamVerificationResult{}, errors.New("ID CCTV tidak valid")
	}
	now := time.Now()
	slotCh := make(chan struct{})
	resCh := make(chan streamVerificationResult, 1)

	job.jobID = uuid.New()
	job.Priority = GatePriorityInteractive
	job.EnqueuedAt = now
	job.ReadyAt = now
	job.slotAcquiredChan = slotCh
	job.CompletionNotifyChan = resCh

	g.mu.Lock()
	if g.closed {
		g.mu.Unlock()
		return streamVerificationResult{}, ErrGateClosed
	}
	// Bila kamera sedang mengantre di MainQueue latar belakang, hapus dari MainQueue agar tidak ganda.
	if existingPrio, ok := g.queuedPriority[job.CCTV.ID]; ok && existingPrio == GatePriorityMain {
		g.removeMainQueueCameraLocked(job.CCTV.ID)
	}
	g.interactiveQueue = append(g.interactiveQueue, job)
	g.cond.Broadcast()
	g.mu.Unlock()

	select {
	case <-slotCh:
		select {
		case res := <-resCh:
			return res, nil
		case <-g.closedChan:
			return streamVerificationResult{}, ErrGateClosed
		}
	case <-g.closedChan:
		return streamVerificationResult{}, ErrGateClosed
	case <-waitCtx.Done():
		g.mu.Lock()
		removed := g.removeInteractiveJobByIDLocked(job.jobID)
		closed := g.closed
		g.mu.Unlock()
		if removed {
			return streamVerificationResult{}, ErrGateWaitTimeout
		}
		if closed {
			return streamVerificationResult{}, ErrGateClosed
		}
		// Bila tepat sebelum Lock job sudah memperoleh slot di dispatcher, tunggu hasil probe selesai.
		select {
		case res := <-resCh:
			return res, nil
		case <-g.closedChan:
			return streamVerificationResult{}, ErrGateClosed
		}
	}
}

func (g *UnifiedNVRSessionGate) removeMainQueueCameraLocked(cid uuid.UUID) {
	out := g.mainQueue[:0]
	for _, item := range g.mainQueue {
		if item.CCTV.ID != cid {
			out = append(out, item)
		} else if item.CompletionNotifyChan != nil {
			close(item.CompletionNotifyChan)
		}
	}
	g.mainQueue = out
	delete(g.queuedPriority, cid)
}

func (g *UnifiedNVRSessionGate) removeInteractiveJobByIDLocked(jobID uuid.UUID) bool {
	for idx, item := range g.interactiveQueue {
		if item.jobID == jobID {
			g.interactiveQueue = append(g.interactiveQueue[:idx], g.interactiveQueue[idx+1:]...)
			return true
		}
	}
	return false
}

// Enqueue mendaftarkan job ke antrean prioritas dengan Per-Camera In-Flight & Pending Guard.
// Mengembalikan true bila diterima atau ditingkatkan prioritasnya.
func (g *UnifiedNVRSessionGate) Enqueue(job GateJob) bool {
	if job.CCTV.ID == uuid.Nil {
		return false
	}
	now := time.Now()
	if job.EnqueuedAt.IsZero() {
		job.EnqueuedAt = now
	}
	if job.ReadyAt.IsZero() {
		job.ReadyAt = job.EnqueuedAt
	}

	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed {
		return false
	}

	cid := job.CCTV.ID
	// Jika kamera sedang aktif diperiksa, cegah duplikasi pemeriksaan serentak pada Main/PostMaint;
	// job Confirm yang terjadwal di masa depan (ReadyAt > now) tetap boleh masuk antrean tunggu.
	if g.inFlight[cid] && job.Priority == GatePriorityMain {
		return false
	}

	if existingPrio, alreadyQueued := g.queuedPriority[cid]; alreadyQueued {
		if existingPrio <= job.Priority {
			return false
		}
		// Hapus dari antrean berprioritas lebih rendah sebelum mempromosikan ke antrean lebih tinggi.
		g.removeQueuedCameraLocked(cid)
	}

	switch job.Priority {
	case GatePriorityConfirm:
		g.confirmQueue = append(g.confirmQueue, job)
	case GatePriorityPostMaintenance:
		g.postMaintQueue = append(g.postMaintQueue, job)
	default:
		job.Priority = GatePriorityMain
		g.mainQueue = append(g.mainQueue, job)
	}
	g.queuedPriority[cid] = job.Priority
	g.cond.Broadcast()
	return true
}

func (g *UnifiedNVRSessionGate) removeQueuedCameraLocked(cid uuid.UUID) {
	filter := func(slice []GateJob) []GateJob {
		out := slice[:0]
		for _, item := range slice {
			if item.CCTV.ID != cid {
				out = append(out, item)
			} else if item.CompletionNotifyChan != nil {
				close(item.CompletionNotifyChan)
			}
		}
		return out
	}
	g.confirmQueue = filter(g.confirmQueue)
	g.postMaintQueue = filter(g.postMaintQueue)
	g.mainQueue = filter(g.mainQueue)
	delete(g.queuedPriority, cid)
}

func (g *UnifiedNVRSessionGate) Close() {
	g.mu.Lock()
	if !g.closed {
		g.closed = true
		close(g.closedChan)
	}
	g.cond.Broadcast()
	g.mu.Unlock()
}

func (g *UnifiedNVRSessionGate) ClosedChan() <-chan struct{} {
	return g.closedChan
}

// RunDispatcher menjalankan loop penjadwalan tunggal yang mematuhi:
//  1. Urutan prioritas ketat: ConfirmQueue > InteractiveQueue > PostMaintenanceQueue > MainQueue,
//  2. Pacer tunggal lintas antrean (PaceInterval),
//  3. Batas sesi upstream gabungan (activeVerifications + closingTailLeases <= MaxUpstreamSlots),
//  4. Batas pemeriksaan GSM bersamaan (activeGSMSlots <= MaxGSMConcurrent).
func (g *UnifiedNVRSessionGate) RunDispatcher(ctx context.Context, probe GateProbeFunc, onComplete func(GateJob, streamVerificationResult), tailHook GateTailReleaseHook) {
	go func() {
		<-ctx.Done()
		g.Close()
	}()

	for {
		job, isGSM, waitDur, ok := g.tryPopNextReadyJob()
		if !ok {
			if ctx.Err() != nil {
				return
			}
			if waitDur > 0 {
				timer := time.NewTimer(waitDur)
				select {
				case <-ctx.Done():
					timer.Stop()
					return
				case <-g.closedChan:
					timer.Stop()
					return
				case <-timer.C:
				}
				continue
			}
			g.mu.Lock()
			if !g.closed && len(g.confirmQueue) == 0 && len(g.interactiveQueue) == 0 && len(g.postMaintQueue) == 0 && len(g.mainQueue) == 0 {
				g.cond.Wait()
			}
			closed := g.closed
			g.mu.Unlock()
			if closed {
				return
			}
			continue
		}

		go g.executeLeasedJob(ctx, job, isGSM, probe, onComplete, tailHook)
	}
}

func (g *UnifiedNVRSessionGate) tryPopNextReadyJob() (GateJob, bool, time.Duration, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.closed {
		return GateJob{}, false, 0, false
	}

	now := time.Now()
	if !g.lastStartAt.IsZero() && g.PaceInterval > 0 {
		nextPace := g.lastStartAt.Add(g.PaceInterval)
		if now.Before(nextPace) {
			return GateJob{}, false, nextPace.Sub(now), false
		}
	}

	if g.activeVerifications+g.closingTailLeases >= g.MaxUpstreamSlots {
		return GateJob{}, false, 20 * time.Millisecond, false
	}

	var earliestFuture time.Time
	pickFrom := func(queue *[]GateJob, clearQueuedPrio bool) (GateJob, bool, bool) {
		s := *queue
		for idx := 0; idx < len(s); idx++ {
			cand := s[idx]
			if g.inFlight[cand.CCTV.ID] {
				continue
			}
			if now.Before(cand.ReadyAt) {
				if earliestFuture.IsZero() || cand.ReadyAt.Before(earliestFuture) {
					earliestFuture = cand.ReadyAt
				}
				continue
			}
			candGSM := networkType(cand.CCTV) == "gsm"
			if candGSM && g.activeGSMSlots >= g.MaxGSMConcurrent {
				continue
			}
			// Pop elemen idx
			*queue = append(s[:idx], s[idx+1:]...)
			if clearQueuedPrio {
				delete(g.queuedPriority, cand.CCTV.ID)
			}
			g.inFlight[cand.CCTV.ID] = true
			g.activeVerifications++
			if candGSM {
				g.activeGSMSlots++
			}
			total := g.activeVerifications + g.closingTailLeases
			if total > g.peakOccupiedSlots {
				g.peakOccupiedSlots = total
			}
			g.lastStartAt = now
			if cand.slotAcquiredChan != nil {
				close(cand.slotAcquiredChan)
			}
			return cand, candGSM, true
		}
		return GateJob{}, false, false
	}

	if job, isGSM, found := pickFrom(&g.confirmQueue, true); found {
		return job, isGSM, 0, true
	}
	if job, isGSM, found := pickFrom(&g.interactiveQueue, false); found {
		return job, isGSM, 0, true
	}
	if job, isGSM, found := pickFrom(&g.postMaintQueue, true); found {
		return job, isGSM, 0, true
	}
	if job, isGSM, found := pickFrom(&g.mainQueue, true); found {
		return job, isGSM, 0, true
	}

	if !earliestFuture.IsZero() {
		wait := earliestFuture.Sub(now)
		if wait > 250*time.Millisecond {
			wait = 250 * time.Millisecond
		}
		if wait < 5*time.Millisecond {
			wait = 5 * time.Millisecond
		}
		return GateJob{}, false, wait, false
	}
	if len(g.confirmQueue)+len(g.interactiveQueue)+len(g.postMaintQueue)+len(g.mainQueue) > 0 {
		return GateJob{}, false, 20 * time.Millisecond, false
	}
	return GateJob{}, false, 0, false
}

func (g *UnifiedNVRSessionGate) executeLeasedJob(
	ctx context.Context,
	job GateJob,
	isGSM bool,
	probe GateProbeFunc,
	onComplete func(GateJob, streamVerificationResult),
	tailHook GateTailReleaseHook,
) {
	probeFn := probe
	if job.JobProbe != nil {
		probeFn = job.JobProbe
	}
	res, openedOnDemand := probeFn(ctx, job)

	if onComplete != nil && !job.SkipCompletionHandler {
		onComplete(job, res)
	}
	if job.CompletionNotifyChan != nil {
		select {
		case job.CompletionNotifyChan <- res:
		default:
		}
	}

	g.mu.Lock()
	delete(g.inFlight, job.CCTV.ID)
	g.activeVerifications--
	if openedOnDemand && g.SourceCloseTail > 0 {
		// Pertahankan slot sebagai closingTailLeases selama SourceCloseTail (sourceOnDemandCloseAfter)
		// agar sesi RTSP yang masih terbuka di MediaMTX tetap terhitung dalam batas MaxUpstreamSlots.
		g.closingTailLeases++
		tailDur := g.SourceCloseTail
		g.mu.Unlock()

		time.AfterFunc(tailDur, func() {
			if tailHook != nil {
				tailHook(job, res)
			}
			g.mu.Lock()
			g.closingTailLeases--
			if isGSM {
				g.activeGSMSlots--
			}
			g.cond.Broadcast()
			g.mu.Unlock()
		})
		return
	}

	if isGSM {
		g.activeGSMSlots--
	}
	g.cond.Broadcast()
	g.mu.Unlock()
}
