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

// RunRTSPProber menjalankan engine probe status live RTSP (Task 14A.1).
// Berjalan sekali segera saat startup lalu berulang setiap 2 menit. Setiap
// siklus memeriksa ketersediaan host NVR dan channel tiap CCTV stream_enabled,
// lalu memperbarui kolom stream_live & stream_checked_at.
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

// runRTSPProbeCycle menjalankan satu siklus probe status live RTSP terhadap
// seluruh CCTV dengan stream_enabled=true. Meng-update kolom stream_live &
// stream_checked_at serta mempublikasikan event bila status berubah.
func runRTSPProbeCycle(db *gorm.DB) {
	var cctvs []models.CCTV
	if err := db.Where("stream_enabled = ?", true).Find(&cctvs).Error; err != nil {
		utils.Log.Errorf("❌ Gagal memuat CCTV untuk probe RTSP: %v", err)
		return
	}

	host := utils.GetRTSPHost()
	port := utils.GetRTSPPort()
	hostReachable := utils.ProbeRTSPHost(host, port)

	online, offline, unchecked := 0, 0, 0
	now := time.Now()

	for _, cctv := range cctvs {
		live := false
		switch {
		case !hostReachable:
			// Host NVR tidak dapat dijangkau → seluruh channel dianggap tidak live.
			live = false
		case cctv.ChannelPort > 0:
			// Probe channel sub-stream: channelPort*100 + 2.
			channelID := cctv.ChannelPort*100 + 2
			live = utils.ProbeRTSPChannel(host, port, channelID)
		default:
			// Tanpa channel port → tidak ada channel spesifik yang bisa diprobe,
			// anggap live bila host itu sendiri dapat dijangkau.
			live = true
		}

		if live != cctv.StreamLive {
			// Status berubah → simpan perubahan + broadcast event + log.
			if err := db.Model(&models.CCTV{}).Where("id = ?", cctv.ID).
				Updates(map[string]interface{}{
					"stream_live":       live,
					"stream_checked_at": now,
				}).Error; err != nil {
				utils.Log.Errorf("❌ Gagal memperbarui status live CCTV %s: %v", cctv.ID.String(), err)
				continue
			}
			utils.BroadcastEvent("CCTV_LIVE_STATUS_UPDATED", fiber.Map{
				"cctv_id":           cctv.ID.String(),
				"stream_live":       live,
				"stream_checked_at": now.Format(time.RFC3339),
				"nama":              cctv.Nama,
				"kode_cctv":         cctv.KodeCCTV,
			})
			if live {
				utils.Log.Infof("🟢 CCTV %s (%s) kini live — path channel %d", cctv.Nama, cctv.KodeCCTV, cctv.ChannelPort)
			} else {
				utils.Log.Warnf("🔴 CCTV %s (%s) kini offline", cctv.Nama, cctv.KodeCCTV)
			}
		} else {
			// Status tidak berubah → cukup perbarui waktu pemeriksaan.
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

	utils.Log.Infof("📡 Probe RTSP selesai — host %s:%d reachable=%v | online=%d offline=%d unchecked=%d",
		host, port, hostReachable, online, offline, unchecked)
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
			"cctv_id":           cctv.ID.String(),
			"nama":              cctv.Nama,
			"kode_cctv":         cctv.KodeCCTV,
			"stream_enabled":    cctv.StreamEnabled,
			"stream_live":       cctv.StreamLive,
			"stream_checked_at": cctv.StreamCheckedAt,
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
