package handlers

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"sigap-backend/config"
	"sigap-backend/incidents"
	"sigap-backend/models"
	"sigap-backend/utils"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// MetroEDiagnosticSnapshot menyimpan status aktual diagnostik konektivitas Metro-E
// (jaringan dedicated 10.20.105.x) serta apakah mode fallback pasif sedang aktif.
type MetroEDiagnosticSnapshot struct {
	ConfiguredDirectProbe bool       `json:"configured_direct_probe"`
	EffectiveDirectProbe  bool       `json:"effective_direct_probe"`
	PassiveFallbackActive bool       `json:"passive_fallback_active"`
	Mode                  string     `json:"mode"`
	Reason                string     `json:"reason"`
	ProbeTargetsTested    []string   `json:"probe_targets_tested"`
	ReachableTargets      int        `json:"reachable_targets"`
	UnreachableTargets    int        `json:"unreachable_targets"`
	LastCheckedAt         *time.Time `json:"last_checked_at"`
}

const (
	metroEModeDirectProbe      = "direct_probe"
	metroEModePassiveFallback  = "passive_fallback"
	metroEModePassiveConfig    = "passive_configured"
	maxMetroEStartupSamples    = 5
	metroEStartupProbeTimeout  = 1500 * time.Millisecond
)

var (
	metroEDiagnosticMu sync.RWMutex
	metroEDiagnostic   MetroEDiagnosticSnapshot

	// metroEReachabilityProbeFunc dapat diganti pada pengujian unit untuk
	// mensimulasikan jaringan 10.20.105.x yang reachable atau unreachable.
	metroEReachabilityProbeFunc = func(host string) bool {
		_, ok := utils.ProbeICMP(host, metroEStartupProbeTimeout)
		return ok
	}
)

func resetMetroEDiagnosticStateForTest() {
	metroEDiagnosticMu.Lock()
	defer metroEDiagnosticMu.Unlock()
	metroEDiagnostic = MetroEDiagnosticSnapshot{}
}

// effectiveMetroEDirectProbe mengembalikan true hanya jika direct probe aktif
// di konfigurasi DAN tidak sedang mengalami auto-fallback pasif akibat 10.20.105.x
// tidak dapat dijangkau saat startup.
func effectiveMetroEDirectProbe() bool {
	if !utils.MetroEDirectProbeEnabled() {
		return false
	}
	metroEDiagnosticMu.RLock()
	fallback := metroEDiagnostic.PassiveFallbackActive
	metroEDiagnosticMu.RUnlock()
	return !fallback
}

func isMetroEPassiveFallbackActive() bool {
	metroEDiagnosticMu.RLock()
	fallback := metroEDiagnostic.PassiveFallbackActive
	metroEDiagnosticMu.RUnlock()
	return fallback
}

// GetMetroEDiagnosticSnapshot mengembalikan salinan status diagnostik Metro-E
// yang aman dibaca oleh handler admin.
func GetMetroEDiagnosticSnapshot() MetroEDiagnosticSnapshot {
	configured := utils.MetroEDirectProbeEnabled()
	metroEDiagnosticMu.RLock()
	snap := metroEDiagnostic
	metroEDiagnosticMu.RUnlock()

	if !configured {
		snap.ConfiguredDirectProbe = false
		snap.EffectiveDirectProbe = false
		snap.PassiveFallbackActive = false
		snap.Mode = metroEModePassiveConfig
		if snap.Reason == "" {
			snap.Reason = "CCTV_METRO_E_DIRECT_PROBE dinonaktifkan secara eksplisit; status Metro-E memakai mode pasif MediaMTX"
		}
		if snap.ProbeTargetsTested == nil {
			snap.ProbeTargetsTested = []string{}
		}
		return snap
	}

	snap.ConfiguredDirectProbe = true
	snap.EffectiveDirectProbe = !snap.PassiveFallbackActive
	if snap.Mode == "" || snap.Mode == metroEModePassiveConfig {
		if snap.PassiveFallbackActive {
			snap.Mode = metroEModePassiveFallback
		} else {
			snap.Mode = metroEModeDirectProbe
		}
	}
	if snap.Reason == "" {
		snap.Reason = "Probe langsung Metro-E (10.20.105.x) aktif"
	}
	if snap.ProbeTargetsTested == nil {
		snap.ProbeTargetsTested = []string{}
	}
	return snap
}

func collectMetroEProbeTargets(db *gorm.DB) []string {
	seen := make(map[string]struct{})
	targets := make([]string, 0, maxMetroEStartupSamples)
	addTarget := func(raw string) {
		ip := strings.TrimSpace(raw)
		if ip == "" || len(targets) >= maxMetroEStartupSamples {
			return
		}
		if _, exists := seen[ip]; exists {
			return
		}
		seen[ip] = struct{}{}
		targets = append(targets, ip)
	}

	if envTargets := strings.TrimSpace(os.Getenv("CCTV_METRO_E_PROBE_TARGET")); envTargets != "" {
		for _, part := range strings.Split(envTargets, ",") {
			addTarget(part)
		}
	}

	if db != nil && len(targets) < maxMetroEStartupSamples {
		var dbIPs []string
		err := db.Table("cctvs").
			Where("deleted_at IS NULL AND LOWER(COALESCE(status, '')) <> 'nonaktif' AND (stream_enabled = ? OR monitoring_enabled = ?) AND ip_camera LIKE ?",
				true, true, "10.20.105.%").
			Order("ip_camera ASC").
			Limit(maxMetroEStartupSamples).
			Pluck("ip_camera", &dbIPs).Error
		if err != nil {
			utils.Log.Debugf("Pemeriksaan target Metro-E startup gagal memuat IP kamera: %v", err)
		} else {
			for _, ip := range dbIPs {
				addTarget(ip)
			}
		}
	}

	return targets
}

// CheckMetroEReachabilityAtStartup memeriksa keterjangkauan host Metro-E (10.20.105.x)
// ketika CCTV_METRO_E_DIRECT_PROBE=true. Bila seluruh host uji tidak dapat dijangkau
// dari host/container saat startup, dicatat warning eksplisit dan engine probe otomatis
// beralih ke mode pasif MediaMTX agar kamera Metro-E tidak menghasilkan offline palsu.
func CheckMetroEReachabilityAtStartup(db *gorm.DB) MetroEDiagnosticSnapshot {
	now := time.Now().UTC()
	configured := utils.MetroEDirectProbeEnabled()

	if !configured {
		snap := MetroEDiagnosticSnapshot{
			ConfiguredDirectProbe: false,
			EffectiveDirectProbe:  false,
			PassiveFallbackActive: false,
			Mode:                  metroEModePassiveConfig,
			Reason:                "CCTV_METRO_E_DIRECT_PROBE=false; seluruh kamera Metro-E memakai status pasif MediaMTX",
			ProbeTargetsTested:    []string{},
			LastCheckedAt:         &now,
		}
		metroEDiagnosticMu.Lock()
		metroEDiagnostic = snap
		metroEDiagnosticMu.Unlock()
		utils.Log.Info("ℹ️ Metro-E direct probe dinonaktifkan via env — memakai status pasif MediaMTX")
		return snap
	}

	targets := collectMetroEProbeTargets(db)
	if len(targets) == 0 {
		snap := MetroEDiagnosticSnapshot{
			ConfiguredDirectProbe: true,
			EffectiveDirectProbe:  true,
			PassiveFallbackActive: false,
			Mode:                  metroEModeDirectProbe,
			Reason:                "Belum ada target host 10.20.105.x aktif untuk diuji saat startup; direct probe tetap siap",
			ProbeTargetsTested:    []string{},
			LastCheckedAt:         &now,
		}
		metroEDiagnosticMu.Lock()
		metroEDiagnostic = snap
		metroEDiagnosticMu.Unlock()
		return snap
	}

	results := make([]bool, len(targets))
	var wg sync.WaitGroup
	for i, target := range targets {
		wg.Add(1)
		go func(idx int, host string) {
			defer wg.Done()
			results[idx] = metroEReachabilityProbeFunc(host)
		}(i, target)
	}
	wg.Wait()

	reachable, unreachable := 0, 0
	for _, ok := range results {
		if ok {
			reachable++
		} else {
			unreachable++
		}
	}

	snap := MetroEDiagnosticSnapshot{
		ConfiguredDirectProbe: true,
		ProbeTargetsTested:    targets,
		ReachableTargets:      reachable,
		UnreachableTargets:    unreachable,
		LastCheckedAt:         &now,
	}

	if reachable == 0 {
		snap.EffectiveDirectProbe = false
		snap.PassiveFallbackActive = true
		snap.Mode = metroEModePassiveFallback
		snap.Reason = fmt.Sprintf(
			"CCTV_METRO_E_DIRECT_PROBE=true tetapi %d/%d target 10.20.105.x tidak dapat dijangkau saat startup; fallback otomatis ke status pasif MediaMTX",
			unreachable, len(targets),
		)
		utils.Log.Warnf("⚠️ %s (targets: %v)", snap.Reason, targets)
	} else {
		snap.EffectiveDirectProbe = true
		snap.PassiveFallbackActive = false
		snap.Mode = metroEModeDirectProbe
		snap.Reason = fmt.Sprintf(
			"Jaringan Metro-E (10.20.105.x) dapat dijangkau saat startup (%d/%d host merespons)",
			reachable, len(targets),
		)
		utils.Log.Infof("✅ %s", snap.Reason)
	}

	metroEDiagnosticMu.Lock()
	metroEDiagnostic = snap
	metroEDiagnosticMu.Unlock()

	return snap
}

func resolveLiveDisplayState(cctv models.CCTV, now time.Time) (streamState string, effectiveState string, falseOfflineSuppressed bool) {
	streamState = cctv.StreamState
	if streamState == "" {
		// Fallback data lama sebelum migrasi 000025.
		switch {
		case cctv.StreamLive:
			streamState = models.StreamStateLive
		case cctv.StreamCheckedAt != nil:
			streamState = models.StreamStateOffline
		default:
			streamState = models.StreamStateUnknown
		}
	}

	effectiveState = effectiveStreamState(cctv, now)

	// Saat auto-fallback pasif aktif karena 10.20.105.x tidak dapat dijangkau dari host,
	// jangan tampilkan status offline lama yang berasal dari probe TCP sebelumnya tanpa
	// bukti verifikasi MediaMTX segar.
	if utils.IsMetroE(cctv) && isMetroEPassiveFallbackActive() && !isStreamVerificationFresh(cctv, now) {
		if streamState == models.StreamStateOffline {
			streamState = models.StreamStateUnknown
			falseOfflineSuppressed = true
		}
		if effectiveState == models.StreamStateOffline {
			effectiveState = models.StreamStateUnknown
			falseOfflineSuppressed = true
		}
	}

	return streamState, effectiveState, falseOfflineSuppressed
}

// GetStreamDiagnostics menyediakan ringkasan diagnostik streaming untuk Admin,
// meliputi status keterjangkauan Metro-E (direct probe vs passive fallback),
// kesehatan MediaMTX, antrean/heartbeat StreamVerifier, serta komposisi status CCTV.
// @Summary      Diagnostik Streaming & Metro-E
// @Description  Mengembalikan indikator diagnostik Metro-E (10.20.105.x), mode fallback pasif, keterjangkauan MediaMTX, heartbeat StreamVerifier, ringkasan status bebas false offline, serta parameter observabilitas (hanya Admin).
// @Tags         Streaming
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "Ringkasan diagnostik streaming"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: hanya Admin"
// @Failure      500 {object} map[string]interface{} "Gagal memuat data diagnostik"
// @Router       /streams/diagnostics [get]
func GetStreamDiagnostics(c *fiber.Ctx) error {
	if config.DB == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Database belum tersedia",
		})
	}
	if role, _ := c.Locals("role").(string); role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "error",
			"message": "Akses ditolak: hanya Admin yang diizinkan mengakses diagnostik streaming",
		})
	}

	query := config.DB.Model(&models.CCTV{})
	if siteID, _ := c.Locals("site_id").(string); siteID != "" {
		query = query.Where("site_id = ?", siteID)
	}

	var cctvs []models.CCTV
	if err := query.Find(&cctvs).Error; err != nil {
		utils.Log.Errorf("❌ Gagal memuat CCTV untuk diagnostik streaming: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal memuat data CCTV untuk diagnostik",
		})
	}

	now := time.Now().UTC()
	type netCounts struct {
		Total   int `json:"total"`
		Online  int `json:"online"`
		Idle    int `json:"idle"`
		Offline int `json:"offline"`
		Unknown int `json:"unknown"`
	}
	netStats := map[string]*netCounts{
		"metro-e": {},
		"gsm":     {},
	}

	totalOnline, totalIdle, totalOffline, totalUnknown := 0, 0, 0, 0
	suppressedCount := 0
	for _, cctv := range cctvs {
		netType := networkType(cctv)
		bucket := netStats[netType]
		bucket.Total++

		_, eff, suppressed := resolveLiveDisplayState(cctv, now)
		if suppressed {
			suppressedCount++
		}
		switch eff {
		case models.StreamStateLive:
			totalOnline++
			bucket.Online++
		case models.StreamStateIdle:
			totalIdle++
			bucket.Idle++
		case models.StreamStateOffline:
			totalOffline++
			bucket.Offline++
		default:
			totalUnknown++
			bucket.Unknown++
		}
	}

	mtxReachable := true
	mtxErr := ""
	operationalPaths := 0
	if paths, err := streamClient.ListPaths(); err != nil {
		mtxReachable = false
		mtxErr = safeStreamErrorText(err)
	} else {
		operationalPaths = len(paths)
	}

	heartbeatMu.Lock()
	startedAt := lastVerifierCycleStartedAt
	completedAt := lastVerifierCycleCompletedAt
	camCount := lastVerifierCameraCount
	mtxLastHealthy := lastMediaMTXHealthy
	mtxLastErr := lastMediaMTXError
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
	verifierStale := completedAt != nil && now.Sub(*completedAt) > maxStale

	var openMonitorEpisodes int64
	if IncidentMonitoringEnabled() {
		_ = config.DB.Model(&incidents.MonitorHealthEpisode{}).Where("status = 'open'").Count(&openMonitorEpisodes).Error
	}

	metroSnap := GetMetroEDiagnosticSnapshot()
	return c.JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"metro_e": fiber.Map{
				"configured_direct_probe": metroSnap.ConfiguredDirectProbe,
				"effective_direct_probe":  metroSnap.EffectiveDirectProbe,
				"passive_fallback_active": metroSnap.PassiveFallbackActive,
				"mode":                    metroSnap.Mode,
				"reason":                  metroSnap.Reason,
				"probe_targets_tested":    metroSnap.ProbeTargetsTested,
				"reachable_targets":       metroSnap.ReachableTargets,
				"unreachable_targets":     metroSnap.UnreachableTargets,
				"last_checked_at":         metroSnap.LastCheckedAt,
				"camera_count":            netStats["metro-e"].Total,
			},
			"mediamtx": fiber.Map{
				"reachable":         mtxReachable,
				"operational_paths": operationalPaths,
				"last_healthy":      mtxLastHealthy,
				"error":             mtxErr,
				"last_verify_error": mtxLastErr,
			},
			"verifier": fiber.Map{
				"last_cycle_started_at":   startedAt,
				"last_cycle_completed_at": completedAt,
				"last_cycle_camera_count": camCount,
				"stale":                   verifierStale,
				"open_monitor_episodes":   openMonitorEpisodes,
				"session_gate":            gateSnap,
			},
			"summary": fiber.Map{
				"total":                    len(cctvs),
				"online":                   totalOnline,
				"idle":                     totalIdle,
				"offline":                  totalOffline,
				"unknown":                  totalUnknown,
				"false_offline_protected":  !metroSnap.EffectiveDirectProbe,
				"false_offline_suppressed": suppressedCount,
			},
			"network_breakdown": netStats,
			"observability": fiber.Map{
				"metrics_endpoint":          "/api/metrics",
				"stream_log_retention_days": config.CCTVStreamLogRetentionDays(),
				"timezone":                  config.AppLocation().String(),
			},
			"checked_at": now,
		},
	})
}
