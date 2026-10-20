package handlers

import (
	"time"

	"sigap-backend/config"
	"sigap-backend/models"
	"sigap-backend/utils"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// rtspProbeInterval adalah interval antar siklus probe status live RTSP.
const rtspProbeInterval = 2 * time.Minute

// RunRTSPProber menjalankan engine probe status live RTSP (Task 14A.1,
// Sprint 37-PATCH: strategi Metro-E vs GSM). Berjalan sekali segera saat
// startup lalu berulang setiap 2 menit. Setiap siklus memeriksa ketersediaan
// kamera sesuai kebijakan jaringan (Metro-E diprobe langsung, GSM hanya
// melalui status path MediaMTX lokal — TIDAK menyentuh kamera), lalu
// memperbarui kolom stream_live, stream_state, stream_checked_at,
// ping_latency_ms & stream_last_active_at.
func RunRTSPProber(db *gorm.DB) {
	if db == nil {
		return
	}

	runRTSPProbeCycle(db)

	ticker := time.NewTicker(rtspProbeInterval)
	defer ticker.Stop()
	for range ticker.C {
		runRTSPProbeCycle(db)
	}
}

// networkType mengembalikan tipe jaringan kamera: "metro-e" untuk kamera pada
// jaringan dedicated OIKN (prefix IP 10.20.105) atau "gsm" untuk semua lainnya.
// Dipakai pada payload event & respons GetLiveStatus.
func networkType(cctv models.CCTV) string {
	if utils.IsMetroE(cctv) {
		return "metro-e"
	}
	return "gsm"
}

// runRTSPProbeCycle menjalankan satu siklus probe status live terhadap seluruh
// CCTV dengan stream_enabled=true, dengan strategi berbeda per tipe jaringan:
//   - Metro-E: probe langsung ke IP kamera (TCP dial port 554/80, ukur latensi)
//     → status live/offline.
//   - GSM: TIDAK menyentuh kamera sama sekali. Status diambil pasif dari
//     MediaMTX: path ready → live; path terdaftar tetapi idle karena on-demand
//     → idle (bukan offline); path tidak dapat disimpulkan → unknown.
//
// Kolom stream_live tetap dipertahankan untuk kompatibilitas, sedangkan
// stream_state menyimpan status empat keadaan yang lebih akurat.
func runRTSPProbeCycle(db *gorm.DB) {
	var cctvs []models.CCTV
	if err := db.Where("stream_enabled = ?", true).Find(&cctvs).Error; err != nil {
		utils.Log.Errorf("❌ Gagal memuat CCTV untuk probe RTSP: %v", err)
		return
	}

	// Baca path operasional (instantiated) dan path terdaftar (config) sekali
	// per siklus. Path on-demand yang belum ada pembaca hanya ada di config,
	// bukan di daftar operasional — kondisi itu berarti idle, bukan offline.
	operationalPaths := map[string]utils.PathInfo{}
	if paths, err := streamClient.ListPaths(); err != nil {
		utils.Log.Warnf("⚠️ Gagal mengambil daftar path operasional MediaMTX: %v — status GSM yang tidak dapat disimpulkan akan menjadi unknown", err)
	} else {
		for _, p := range paths {
			operationalPaths[p.Name] = p
		}
	}

	registeredPaths := map[string]bool{}
	configAvailable := true
	if configs, err := streamClient.ListPathConfigs(); err != nil {
		configAvailable = false
		utils.Log.Warnf("⚠️ Gagal mengambil daftar path terdaftar MediaMTX: %v — path on-demand idle tidak dapat dikonfirmasi", err)
	} else {
		for _, p := range configs {
			registeredPaths[p.Name] = true
		}
	}

	now := time.Now()
	online, idle, offline, unchecked := 0, 0, 0, 0

	for _, cctv := range cctvs {
		path := utils.BuildStreamPath(cctv.ID.String())
		pathInfo, operational := operationalPaths[path]
		registered := registeredPaths[path]

		streamState := models.StreamStateUnknown
		var latencyMs *int

		isMetroE := utils.IsMetroE(cctv)
		if isMetroE && utils.MetroEDirectProbeEnabled() {
			// Metro-E: jaringan dedicated — aman diprobe langsung tiap 2 menit.
			ms, ok := utils.ProbeICMP(cctv.IPCamera, 3*time.Second)
			if ok {
				msInt := int(ms)
				latencyMs = &msInt
				streamState = models.StreamStateLive
			} else if operational && pathInfo.SourceReady() {
				// Probe TCP gagal — tetapi stream aktif di MediaMTX (mis. sedang
				// ditonton atau AI pull) → tetap live.
				streamState = models.StreamStateLive
				utils.Log.Debugf("📡 Metro-E %s (%s) probe gagal tetapi path MediaMTX ready → live", cctv.Nama, cctv.KodeCCTV)
			} else {
				// Probe langsung gagal dan tidak ada path ready → offline terkonfirmasi.
				streamState = models.StreamStateOffline
			}
			utils.Log.Debugf("📡 Metro-E %s (%s) state=%s latency=%dms", cctv.Nama, cctv.KodeCCTV, streamState, ms)
		} else {
			// GSM — atau Metro-E saat probe langsung dinonaktifkan karena host
			// tidak punya rute ke 10.20.105.x. Klasifikasi memakai state MediaMTX
			// lokal agar tidak menghasilkan offline palsu.
			probeLabel := "GSM"
			if isMetroE {
				probeLabel = "Metro-E (pasif)"
			}
			switch {
			case operational && pathInfo.SourceReady():
				streamState = models.StreamStateLive
			case operational && len(pathInfo.Readers) > 0:
				// Ada penonton tetapi sumber tidak siap → kegagalan stream nyata.
				streamState = models.StreamStateOffline
				utils.Log.Warnf("⚠️ %s %s (%s) path tidak siap dengan %d penonton → offline", probeLabel, cctv.Nama, cctv.KodeCCTV, len(pathInfo.Readers))
			case operational:
				// Path instantiated tetapi belum ready dan tidak ada penonton:
				// sumber on-demand sedang idle/menutup, bukan bukti kamera mati.
				streamState = models.StreamStateIdle
			case configAvailable && registered:
				// Path terdaftar di config tetapi belum dijalankan → idle on-demand.
				streamState = models.StreamStateIdle
			default:
				// Tidak ada path operasional maupun config yang dapat dikonfirmasi.
				streamState = models.StreamStateUnknown
			}
			utils.Log.Debugf("📡 %s %s (%s) path=%s operational=%v ready=%v readers=%d registered=%v state=%s",
				probeLabel, cctv.Nama, cctv.KodeCCTV, path, operational, pathInfo.SourceReady(), len(pathInfo.Readers), registered, streamState)
		}

		live := streamState == models.StreamStateLive

		// Siapkan peta update: status dan waktu pemeriksaan selalu diset.
		updates := map[string]interface{}{
			"stream_checked_at": now,
			"stream_live":       live,
			"stream_state":      streamState,
		}
		if latencyMs != nil {
			updates["ping_latency_ms"] = *latencyMs
		}
		if live {
			updates["stream_last_active_at"] = now
		}

		stateChanged := streamState != cctv.StreamState
		liveChanged := live != cctv.StreamLive

		if liveChanged || stateChanged {
			// Status berubah → simpan perubahan + broadcast event + log.
			if err := db.Model(&models.CCTV{}).Where("id = ?", cctv.ID).
				Updates(updates).Error; err != nil {
				utils.Log.Errorf("❌ Gagal memperbarui status live CCTV %s: %v", cctv.ID.String(), err)
				continue
			}
			payload := fiber.Map{
				"cctv_id":           cctv.ID.String(),
				"stream_live":       live,
				"stream_state":      streamState,
				"stream_checked_at": now.Format(time.RFC3339),
				"network_type":      networkType(cctv),
				"nama":              cctv.Nama,
				"kode_cctv":         cctv.KodeCCTV,
			}
			if latencyMs != nil {
				payload["ping_latency_ms"] = *latencyMs
			}
			if live {
				payload["stream_last_active_at"] = now.Format(time.RFC3339)
			}
			utils.BroadcastEvent("CCTV_LIVE_STATUS_UPDATED", payload)
			switch streamState {
			case models.StreamStateLive:
				utils.Log.Infof("🟢 CCTV %s (%s) kini live [%s]", cctv.Nama, cctv.KodeCCTV, networkType(cctv))
			case models.StreamStateIdle:
				utils.Log.Infof("🔵 CCTV %s (%s) siaga on-demand (tidak ada penonton) [%s]", cctv.Nama, cctv.KodeCCTV, networkType(cctv))
			case models.StreamStateOffline:
				utils.Log.Warnf("🔴 CCTV %s (%s) kini offline [%s]", cctv.Nama, cctv.KodeCCTV, networkType(cctv))
			default:
				utils.Log.Warnf("⚪ CCTV %s (%s) status stream tidak diketahui [%s]", cctv.Nama, cctv.KodeCCTV, networkType(cctv))
			}
		} else if live {
			// Status tetap live → tetap update (menyegarkan stream_last_active_at
			// & stream_checked_at) agar badge GSM "pernah aktif" tidak kedaluwarsa.
			if err := db.Model(&models.CCTV{}).Where("id = ?", cctv.ID).
				Updates(updates).Error; err != nil {
				utils.Log.Errorf("❌ Gagal memperbarui status live CCTV %s: %v", cctv.ID.String(), err)
			}
		} else {
			// Status tetap offline/idle/unknown → cukup perbarui waktu pemeriksaan.
			if err := db.Model(&models.CCTV{}).Where("id = ?", cctv.ID).
				Update("stream_checked_at", now).Error; err != nil {
				utils.Log.Errorf("❌ Gagal memperbarui stream_checked_at CCTV %s: %v", cctv.ID.String(), err)
			}
		}

		// Hitung ringkasan berdasarkan state saat ini.
		switch streamState {
		case models.StreamStateLive:
			online++
		case models.StreamStateIdle:
			idle++
		case models.StreamStateOffline:
			offline++
		default:
			unchecked++
		}
	}

	utils.Log.Infof("✅ Probe cycle: %d kamera dicek | online=%d idle=%d offline=%d unknown=%d",
		len(cctvs), online, idle, offline, unchecked)
}

// GetLiveStatus menampilkan status live stream seluruh CCTV berdasarkan hasil
// probe RTSP terakhir.
// @Summary      Status Live Stream CCTV
// @Description  Menampilkan ringkasan status live (online/idle/offline/unchecked) seluruh CCTV berdasarkan hasil probe RTSP terakhir. Idle berarti path on-demand terdaftar tetapi tidak ada penonton, bukan offline.
// @Tags         Streaming
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "Ringkasan & daftar status live stream"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Failure      500 {object} map[string]interface{} "Gagal memuat data CCTV"
// @Router       /streams/live-status [get]
func GetLiveStatus(c *fiber.Ctx) error {
	if config.DB == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Database belum tersedia",
		})
	}
	var cctvs []models.CCTV
	if err := config.DB.Find(&cctvs).Error; err != nil {
		utils.Log.Errorf("❌ Gagal memuat CCTV untuk status live: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal memuat data CCTV",
		})
	}

	online, idle, offline, unchecked := 0, 0, 0, 0
	data := make([]fiber.Map, 0, len(cctvs))
	for _, cctv := range cctvs {
		streamState := cctv.StreamState
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

		switch streamState {
		case models.StreamStateLive:
			online++
		case models.StreamStateIdle:
			idle++
		case models.StreamStateOffline:
			offline++
		default:
			unchecked++
		}
		data = append(data, fiber.Map{
			"cctv_id":               cctv.ID.String(),
			"nama":                  cctv.Nama,
			"kode_cctv":             cctv.KodeCCTV,
			"stream_enabled":        cctv.StreamEnabled,
			"stream_live":           cctv.StreamLive,
			"stream_state":          streamState,
			"stream_checked_at":     cctv.StreamCheckedAt,
			"stream_last_active_at": cctv.StreamLastActiveAt,
			"ping_latency_ms":       cctv.PingLatencyMs,
			"network_type":          networkType(cctv),
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"summary": fiber.Map{
			"online":    online,
			"idle":      idle,
			"offline":   offline,
			"unchecked": unchecked,
		},
		"data": data,
	})
}
