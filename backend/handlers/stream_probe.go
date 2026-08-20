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
// memperbarui kolom stream_live, stream_checked_at, ping_latency_ms &
// stream_last_active_at.
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
//     → update stream_live & ping_latency_ms.
//   - GSM: TIDAK menyentuh kamera sama sekali. Status diambil pasif dari daftar
//     path MediaMTX (lokal VPS): Ready=true → live + update stream_last_active_at.
//
// Meng-update kolom stream_live/stream_checked_at (dan kolom spesifik tipe)
// serta mempublikasikan event bila status berubah.
func runRTSPProbeCycle(db *gorm.DB) {
	var cctvs []models.CCTV
	if err := db.Where("stream_enabled = ?", true).Find(&cctvs).Error; err != nil {
		utils.Log.Errorf("❌ Gagal memuat CCTV untuk probe RTSP: %v", err)
		return
	}

	// Baca status path MediaMTX sekali per siklus (sumber status kamera GSM).
	// Bila gagal, lanjutkan dengan peta kosong — kamera GSM dianggap offline
	// pada siklus ini (tidak pernah menyentuh kamera secara langsung).
	mediamtxPaths := map[string]bool{}
	if paths, err := streamClient.ListPaths(); err != nil {
		utils.Log.Warnf("⚠️ Gagal mengambil daftar path MediaMTX: %v — kamera GSM dianggap offline", err)
	} else {
		for _, p := range paths {
			mediamtxPaths[p.Name] = p.SourceReady()
		}
	}

	now := time.Now()
	online, offline, unchecked := 0, 0, 0

	for _, cctv := range cctvs {
		live := false
		var latencyMs *int

		if utils.IsMetroE(cctv) {
			// Metro-E: jaringan dedicated — aman diprobe langsung tiap 2 menit.
			ms, ok := utils.ProbeICMP(cctv.IPCamera, 3*time.Second)
			live = ok
			if ok {
				msInt := int(ms)
				latencyMs = &msInt
			} else {
				// Probe TCP gagal — tapi kamera bisa saja tetap ONLINE karena
				// stream-nya aktif di MediaMTX (mis. sedang ditonton di /monitor
				// atau AI pull). Cek path MediaMTX sebagai sumber kebenaran kedua
				// agar status tidak salah "offline" saat stream benar-benar jalan.
				path := utils.BuildStreamPath(cctv.ID.String())
				if ready, exists := mediamtxPaths[path]; exists && ready {
					live = true
					utils.Log.Debugf("📡 Metro-E %s (%s) probe gagal tetapi path MediaMTX ready → live", cctv.Nama, cctv.KodeCCTV)
				}
			}
			utils.Log.Debugf("📡 Metro-E %s (%s) probe=%v latency=%dms", cctv.Nama, cctv.KodeCCTV, live, ms)
		} else {
			// GSM: TIDAK pernah diprobe langsung (biaya kuota data seluler).
			// Status hanya dari path MediaMTX lokal VPS.
			path := utils.BuildStreamPath(cctv.ID.String())
			ready, exists := mediamtxPaths[path]
			live = exists && ready
			utils.Log.Debugf("📡 GSM %s (%s) path=%s exists=%v ready=%v", cctv.Nama, cctv.KodeCCTV, path, exists, ready)
		}

		// Siapkan peta update: stream_checked_at & stream_live selalu diset.
		updates := map[string]interface{}{
			"stream_checked_at": now,
			"stream_live":       live,
		}
		if latencyMs != nil {
			updates["ping_latency_ms"] = *latencyMs
		}
		if live {
			updates["stream_last_active_at"] = now
		}

		if live != cctv.StreamLive {
			// Status berubah → simpan perubahan + broadcast event + log.
			if err := db.Model(&models.CCTV{}).Where("id = ?", cctv.ID).
				Updates(updates).Error; err != nil {
				utils.Log.Errorf("❌ Gagal memperbarui status live CCTV %s: %v", cctv.ID.String(), err)
				continue
			}
			payload := fiber.Map{
				"cctv_id":           cctv.ID.String(),
				"stream_live":       live,
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
			if live {
				utils.Log.Infof("🟢 CCTV %s (%s) kini live [%s]", cctv.Nama, cctv.KodeCCTV, networkType(cctv))
			} else {
				utils.Log.Warnf("🔴 CCTV %s (%s) kini offline [%s]", cctv.Nama, cctv.KodeCCTV, networkType(cctv))
			}
		} else if live {
			// Status tetap live → tetap update (menyegarkan stream_last_active_at
			// & stream_checked_at) agar badge GSM "pernah aktif" tidak kedaluwarsa.
			if err := db.Model(&models.CCTV{}).Where("id = ?", cctv.ID).
				Updates(updates).Error; err != nil {
				utils.Log.Errorf("❌ Gagal memperbarui status live CCTV %s: %v", cctv.ID.String(), err)
			}
		} else {
			// Status tetap offline → cukup perbarui waktu pemeriksaan.
			if err := db.Model(&models.CCTV{}).Where("id = ?", cctv.ID).
				Update("stream_checked_at", now).Error; err != nil {
				utils.Log.Errorf("❌ Gagal memperbarui stream_checked_at CCTV %s: %v", cctv.ID.String(), err)
			}
		}

		// Hitung ringkasan.
		if cctv.StreamCheckedAt == nil {
			unchecked++
		} else if live {
			online++
		} else {
			offline++
		}
	}

	utils.Log.Infof("✅ Probe cycle: %d kamera dicek | online=%d offline=%d unchecked=%d",
		len(cctvs), online, offline, unchecked)
}

// GetLiveStatus menampilkan status live stream seluruh CCTV berdasarkan hasil
// probe RTSP terakhir.
// @Summary      Status Live Stream CCTV
// @Description  Menampilkan ringkasan status live (online/offline/unchecked) seluruh CCTV berdasarkan hasil probe RTSP terakhir.
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

	online, offline, unchecked := 0, 0, 0
	data := make([]fiber.Map, 0, len(cctvs))
	for _, cctv := range cctvs {
		checked := cctv.StreamCheckedAt != nil
		switch {
		case !checked:
			unchecked++
		case cctv.StreamLive:
			online++
		default:
			offline++
		}
		data = append(data, fiber.Map{
			"cctv_id":               cctv.ID.String(),
			"nama":                  cctv.Nama,
			"kode_cctv":             cctv.KodeCCTV,
			"stream_enabled":        cctv.StreamEnabled,
			"stream_live":           cctv.StreamLive,
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
			"offline":   offline,
			"unchecked": unchecked,
		},
		"data": data,
	})
}
