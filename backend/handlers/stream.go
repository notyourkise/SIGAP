package handlers

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"sigap-backend/config"
	"sigap-backend/models"
	"sigap-backend/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// streamClient adalah klien global untuk berkomunikasi dengan MediaMTX.
// Dapat diganti (di-inject) pada pengujian melalui setStreamClient.
var streamClient = utils.NewMediaMTXClient()

// setStreamClient mengganti klien MediaMTX global — khusus dipakai pada
// pengujian untuk menyuntikkan mock yang tidak benar-benar memanggil server.
func setStreamClient(c *utils.MediaMTXClient) {
	streamClient = c
}

// streamHealthMu melindungi akses ke streamLastReady, streamDownSince, dan
// streamDownNotified agar aman dipakai lintas goroutine (monitor & handler).
var (
	streamHealthMu sync.Mutex
	// streamLastReady memetakan stream path → waktu terakhir diketahui Ready=true.
	streamLastReady = map[string]time.Time{}
	// streamDownSince memetakan stream path → waktu mulai dianggap down.
	streamDownSince = map[string]time.Time{}
	// streamDownNotified memetakan stream path → event STREAM_DOWN sudah dipublikasikan.
	streamDownNotified = map[string]bool{}
)

// broadcastStreamEvent mempublikasikan event stream via WebSocket. Dipisahkan
// lewat variabel fungsi agar pengujian dapat menangkap event tanpa koneksi
// WebSocket nyata (di-override pada stream_health_test.go).
var broadcastStreamEvent = func(event string, payload interface{}) {
	utils.BroadcastEvent(event, payload)
}

// streamHealthMonitorInterval adalah interval iterasi monitor status stream.
const streamHealthMonitorInterval = 5 * time.Minute

// streamDownThreshold adalah durasi down (tanpa ready) sebelum dianggap
// gangguan permanen yang memicu penalti health_score & event STREAM_DOWN.
const streamDownThreshold = 15 * time.Minute

// resetStreamHealthState membersihkan seluruh state kesehatan stream — dipakai
// pengujian agar tidak terjadi polusi antar-test.
func resetStreamHealthState() {
	streamHealthMu.Lock()
	defer streamHealthMu.Unlock()
	streamLastReady = map[string]time.Time{}
	streamDownSince = map[string]time.Time{}
	streamDownNotified = map[string]bool{}
}

// loadCCTVByID memuat data CCTV dari database berdasarkan UUID. Mengembalikan
// model CCTV bila ditemukan, atau error bila ID tidak valid / data tidak ada.
func loadCCTVByID(c *fiber.Ctx) (models.CCTV, bool) {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		_ = c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID CCTV tidak valid",
		})
		return models.CCTV{}, false
	}

	var cctv models.CCTV
	if err := config.DB.First(&cctv, "id = ?", id).Error; err != nil {
		_ = c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Data CCTV tidak ditemukan",
		})
		return models.CCTV{}, false
	}
	return cctv, true
}

// StartStream mengaktifkan (mendaftarkan) stream RTSP suatu CCTV ke MediaMTX.
// @Summary      Mulai Stream CCTV
// @Description  Mendaftarkan aliran RTSP kamera ke MediaMTX agar dapat diakses melalui WHEP/HLS. Kualitas default sub, dapat diubah menjadi main via query ?quality=main.
// @Tags         Streaming
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "CCTV UUID"
// @Param        quality query string false "Kualitas stream: main atau sub (default sub)"
// @Success      200 {object} map[string]interface{} "Stream berhasil dimulai"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid / streaming belum diaktifkan / konfigurasi belum lengkap"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Failure      404 {object} map[string]interface{} "CCTV tidak ditemukan"
// @Failure      500 {object} map[string]interface{} "Gagal mendekripsi password RTSP"
// @Failure      502 {object} map[string]interface{} "Gagal menghubungi server streaming"
// @Router       /streams/{id}/start [post]
func StartStream(c *fiber.Ctx) error {
	cctv, ok := loadCCTVByID(c)
	if !ok {
		return nil
	}

	// Validasi: streaming harus sudah diaktifkan pada master data CCTV.
	if !cctv.StreamEnabled {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Streaming belum diaktifkan",
		})
	}

	// Validasi konfigurasi sumber stream.
	// Bila rtsp_url penuh tersedia, tidak wajib channel port / password.
	if cctv.RTSPURL == "" {
		// Validasi: channel port wajib dikonfigurasi.
		if cctv.ChannelPort <= 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "error",
				"message": "Channel port belum dikonfigurasi",
			})
		}

		// Validasi: password RTSP wajib terisi (tersimpan terenkripsi di DB).
		if cctv.RTSPPassword == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "error",
				"message": "Password RTSP belum dikonfigurasi",
			})
		}
	}

	// Dekripsi password RTSP AES-256 (tidak diperlukan bila rtsp_url penuh
	// sudah memuat kredensial).
	decrypted := ""
	if cctv.RTSPURL == "" {
		var err error
		decrypted, err = utils.DecryptAES256(cctv.RTSPPassword)
		if err != nil {
			utils.Log.Errorf("❌ Gagal mendekripsi password RTSP untuk CCTV %s: %v", cctv.ID, err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "Gagal mendekripsi password RTSP",
			})
		}
	}

	// Tentukan tipe stream berdasarkan query ?quality= (default sub).
	streamType := utils.SubStream
	if c.Query("quality") == "main" {
		streamType = utils.MainStream
	}

	// Susun URL RTSP dan path MediaMTX.
	rtspURL := utils.ResolveRTSPURL(cctv.RTSPURL, cctv.ChannelPort, decrypted, streamType)
	path := utils.BuildStreamPath(cctv.ID.String())

	// Daftarkan path ke MediaMTX (on-demand, ditutup otomatis setelah 30 detik).
	pathCfg := utils.PathConfig{
		Source:                   rtspURL,
		SourceOnDemand:           true,
		SourceOnDemandCloseAfter: "30s",
		RTSPTransport:            "tcp",
	}
	pathAlreadyRegistered := false
	if err := streamClient.AddPath(path, pathCfg); err != nil {
		// Path mungkin sudah terdaftar dari pemanggilan sebelumnya (restart modal
		// atau auto-register). Bila sudah ada, anggap berhasil agar tidak 502.
		if strings.Contains(err.Error(), "already exists") {
			pathAlreadyRegistered = true
			utils.Log.Infof("ℹ️ Path streaming %s sudah terdaftar — dianggap berhasil", path)
		} else {
			utils.Log.Errorf("❌ Gagal menambahkan path streaming %s: %v", path, err)
			return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
				"status":  "error",
				"message": "Gagal menghubungi server streaming",
			})
		}
	}
	sourceReady := false
	if pathAlreadyRegistered {
		if info, err := streamClient.GetPath(path); err == nil {
			sourceReady = info.SourceReady()
		}
	}

	// Catat aktivitas ke audit log dan sebarkan event ke client WebSocket.
	utils.LogAudit(config.DB, c, "START_STREAM", "STREAMING", "Memulai stream CCTV: "+cctv.Nama+" ("+cctv.KodeCCTV+")", nil, "", "")
	utils.BroadcastEvent("STREAM_UPDATED", fiber.Map{
		"cctv_id": cctv.ID.String(),
		"action":  "start",
		"path":    path,
	})

	return c.JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"whep_url":           "/stream/" + path + "/whep",
			"path":               path,
			"quality":            c.Query("quality", "sub"),
			"mediamtx_reachable": true,
			"path_registered":    true,
			"source_ready":       sourceReady,
		},
	})
}

// StopStream menonaktifkan (menghapus) stream suatu CCTV dari MediaMTX.
// @Summary      Hentikan Stream CCTV
// @Description  Menghapus path stream suatu CCTV dari MediaMTX sehingga aliran tidak lagi aktif.
// @Tags         Streaming
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "CCTV UUID"
// @Success      200 {object} map[string]interface{} "Stream berhasil dihentikan"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Failure      404 {object} map[string]interface{} "CCTV tidak ditemukan"
// @Failure      502 {object} map[string]interface{} "Gagal menghubungi server streaming"
// @Router       /streams/{id}/stop [post]
func StopStream(c *fiber.Ctx) error {
	cctv, ok := loadCCTVByID(c)
	if !ok {
		return nil
	}

	path := utils.BuildStreamPath(cctv.ID.String())
	if err := streamClient.RemovePath(path); err != nil {
		utils.Log.Errorf("❌ Gagal menghapus path streaming %s: %v", path, err)
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghubungi server streaming",
		})
	}

	utils.LogAudit(config.DB, c, "STOP_STREAM", "STREAMING", "Menghentikan stream CCTV: "+cctv.Nama+" ("+cctv.KodeCCTV+")", nil, "", "")
	utils.BroadcastEvent("STREAM_UPDATED", fiber.Map{
		"cctv_id": cctv.ID.String(),
		"action":  "stop",
		"path":    path,
	})

	return c.JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"path": path,
		},
	})
}

// StartAllStreams mengaktifkan stream untuk seluruh CCTV yang stream_enabled.
// @Summary      Mulai Semua Stream
// @Description  Mendaftarkan stream RTSP seluruh CCTV dengan stream_enabled=true ke MediaMTX sekaligus.
// @Tags         Streaming
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "Ringkasan total/sukses/gagal"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Router       /streams/start-all [post]
func StartAllStreams(c *fiber.Ctx) error {
	var cctvs []models.CCTV
	if err := config.DB.Where("stream_enabled = ?", true).Find(&cctvs).Error; err != nil {
		utils.Log.Errorf("❌ Gagal memuat CCTV untuk start-all: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal memuat data CCTV",
		})
	}

	total := len(cctvs)
	success := 0
	failed := 0
	if err := streamClient.Ping(); err != nil {
		utils.Log.Warnf("⚠️ MediaMTX tidak dapat dijangkau saat start-all: %v", err)
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"status":  "error",
			"message": "MediaMTX tidak dapat dijangkau dari backend. Pastikan layanan streaming berjalan dan konfigurasi MEDIAMTX_API_URL benar.",
			"data": fiber.Map{
				"total":              total,
				"success":            0,
				"failed":             total,
				"mediamtx_reachable": false,
			},
		})
	}

	for _, cctv := range cctvs {
		decrypted := ""
		if cctv.RTSPURL == "" {
			if cctv.ChannelPort <= 0 {
				utils.Log.Warnf("⚠️ CCTV %s (%s) channel port belum dikonfigurasi — dilewati", cctv.Nama, cctv.KodeCCTV)
				failed++
				continue
			}
			// Lewati dengan peringatan bila password kosong atau gagal didekripsi.
			if cctv.RTSPPassword == "" {
				utils.Log.Warnf("⚠️ CCTV %s (%s) password RTSP kosong — dilewati", cctv.Nama, cctv.KodeCCTV)
				failed++
				continue
			}
			var err error
			decrypted, err = utils.DecryptAES256(cctv.RTSPPassword)
			if err != nil {
				utils.Log.Warnf("⚠️ Gagal mendekripsi password RTSP CCTV %s (%s): %v", cctv.Nama, cctv.KodeCCTV, err)
				failed++
				continue
			}
		}

		rtspURL := utils.ResolveRTSPURL(cctv.RTSPURL, cctv.ChannelPort, decrypted, utils.SubStream)
		path := utils.BuildStreamPath(cctv.ID.String())
		pathCfg := utils.PathConfig{
			Source:                   rtspURL,
			SourceOnDemand:           true,
			SourceOnDemandCloseAfter: "30s",
			RTSPTransport:            "tcp",
		}
		if err := streamClient.AddPath(path, pathCfg); err != nil {
			// Path mungkin sudah terdaftar dari auto-register atau panggilan
			// sebelumnya; bila sudah ada, anggap berhasil.
			if strings.Contains(err.Error(), "already exists") {
				success++
			} else {
				utils.Log.Errorf("❌ Gagal menambahkan path streaming %s: %v", path, err)
				failed++
			}
			continue
		}
		success++
	}

	utils.LogAudit(config.DB, c, "START_ALL_STREAMS", "STREAMING", "Memulai semua stream CCTV", nil, "", "")
	utils.BroadcastEvent("STREAM_UPDATED", fiber.Map{"action": "start_all", "success": success, "failed": failed})

	return c.JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"total":              total,
			"success":            success,
			"failed":             failed,
			"mediamtx_reachable": true,
		},
	})
}

// GetMediaMTXReadiness memeriksa apakah backend dapat menjangkau REST API
// MediaMTX. Endpoint ini tidak membuka URL internal atau kredensial RTSP.
// @Summary      Kesiapan MediaMTX
// @Description  Memeriksa koneksi backend ke MediaMTX sebelum monitor memulai stream.
// @Tags         Streaming
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "MediaMTX dapat dijangkau"
// @Failure      503 {object} map[string]interface{} "MediaMTX tidak dapat dijangkau"
// @Router       /streams/readiness [get]
func GetMediaMTXReadiness(c *fiber.Ctx) error {
	if err := streamClient.Ping(); err != nil {
		utils.Log.Warnf("⚠️ Pemeriksaan kesiapan MediaMTX gagal: %v", err)
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"status":  "error",
			"message": "MediaMTX belum dapat dijangkau. Pastikan layanan streaming berjalan dan konfigurasi koneksi benar.",
			"data": fiber.Map{
				"reachable": false,
			},
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"reachable": true,
		},
	})
}

// StopAllStreams menonaktifkan stream untuk seluruh CCTV yang stream_enabled.
// @Summary      Hentikan Semua Stream
// @Description  Menghapus path stream seluruh CCTV dengan stream_enabled=true dari MediaMTX sekaligus.
// @Tags         Streaming
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "Ringkasan total/sukses/gagal"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Router       /streams/stop-all [post]
func StopAllStreams(c *fiber.Ctx) error {
	var cctvs []models.CCTV
	if err := config.DB.Where("stream_enabled = ?", true).Find(&cctvs).Error; err != nil {
		utils.Log.Errorf("❌ Gagal memuat CCTV untuk stop-all: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal memuat data CCTV",
		})
	}

	total := len(cctvs)
	success := 0
	failed := 0

	for _, cctv := range cctvs {
		path := utils.BuildStreamPath(cctv.ID.String())
		if err := streamClient.RemovePath(path); err != nil {
			utils.Log.Errorf("❌ Gagal menghapus path streaming %s: %v", path, err)
			failed++
			continue
		}
		success++
	}

	utils.LogAudit(config.DB, c, "STOP_ALL_STREAMS", "STREAMING", "Menghentikan semua stream CCTV", nil, "", "")
	utils.BroadcastEvent("STREAM_UPDATED", fiber.Map{"action": "stop_all", "success": success, "failed": failed})

	return c.JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"total":   total,
			"success": success,
			"failed":  failed,
		},
	})
}

// GetStreamStatus menampilkan status stream seluruh CCTV.
// @Summary      Status Seluruh Stream
// @Description  Menampilkan daftar status streaming setiap CCTV termasuk ketersediaan path dan jumlah penonton (viewers). Bila MediaMTX tidak dapat dihubungi, semua dianggap tidak siap.
// @Tags         Streaming
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "Daftar status stream"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Failure      500 {object} map[string]interface{} "Gagal memuat data CCTV"
// @Router       /streams/status [get]
func GetStreamStatus(c *fiber.Ctx) error {
	if config.DB == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Database belum tersedia",
		})
	}
	var cctvs []models.CCTV
	if err := config.DB.Find(&cctvs).Error; err != nil {
		utils.Log.Errorf("❌ Gagal memuat CCTV untuk status stream: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal memuat data CCTV",
		})
	}

	// Registrasi path dan kesiapan sumber adalah dua status berbeda. Path dapat
	// terdaftar dengan benar sementara sumber on-demand belum mengirim frame.
	registered := map[string]bool{}
	configs, configErr := streamClient.ListPathConfigs()
	if configErr != nil {
		utils.Log.Warnf("⚠️ Gagal mengambil konfigurasi path dari MediaMTX: %v", configErr)
	} else {
		for _, cfg := range configs {
			registered[cfg.Name] = true
		}
	}

	pathMap := map[string]utils.PathInfo{}
	paths, pathErr := streamClient.ListPaths()
	if pathErr != nil {
		utils.Log.Warnf("⚠️ Gagal mengambil status path dari MediaMTX: %v", pathErr)
	} else {
		for _, p := range paths {
			pathMap[p.Name] = p
		}
	}
	mediamtxReachable := configErr == nil || pathErr == nil

	result := make([]fiber.Map, 0, len(cctvs))
	for _, cctv := range cctvs {
		path := utils.BuildStreamPath(cctv.ID.String())
		info, exists := pathMap[path]
		ready := exists && info.SourceReady()
		pathRegistered := registered[path] || exists
		viewers := 0
		if exists {
			viewers = len(info.Readers)
		}
		result = append(result, fiber.Map{
			"cctv_id":            cctv.ID.String(),
			"nama":               cctv.Nama,
			"kode_cctv":          cctv.KodeCCTV,
			"stream_enabled":     cctv.StreamEnabled,
			"stream_active":      cctv.StreamEnabled && ready,
			"stream_ready":       ready,
			"mediamtx_reachable": mediamtxReachable,
			"path_registered":    pathRegistered,
			"source_ready":       ready,
			"viewers":            viewers,
			"whep_url":           "/stream/" + path + "/whep",
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   result,
	})
}

// GetStreamURL menampilkan URL streaming (WHEP & HLS) serta status siap suatu CCTV.
// @Summary      Ambil URL Stream CCTV
// @Description  Mengembalikan URL WHEP dan HLS serta status kesiapan stream suatu CCTV.
// @Tags         Streaming
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "CCTV UUID"
// @Success      200 {object} map[string]interface{} "Detail URL stream"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Failure      404 {object} map[string]interface{} "CCTV tidak ditemukan"
// @Router       /streams/{id}/url [get]
func GetStreamURL(c *fiber.Ctx) error {
	cctv, ok := loadCCTVByID(c)
	if !ok {
		return nil
	}

	path := utils.BuildStreamPath(cctv.ID.String())
	ready := false
	viewers := 0
	mediamtxReachable := false
	if info, err := streamClient.GetPath(path); err == nil {
		mediamtxReachable = true
		ready = info.SourceReady()
		viewers = len(info.Readers)
	} else {
		utils.Log.Warnf("⚠️ Gagal mengambil info path %s dari MediaMTX: %v", path, err)
	}

	pathRegistered := false
	if configs, err := streamClient.ListPathConfigs(); err == nil {
		mediamtxReachable = true
		for _, cfg := range configs {
			if cfg.Name == path {
				pathRegistered = true
				break
			}
		}
	} else if ready {
		mediamtxReachable = true
	}
	if ready {
		pathRegistered = true
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"whep_url":           "/stream/" + path + "/whep",
			"hls_url":            "/hls/" + path + "/index.m3u8",
			"stream_active":      cctv.StreamEnabled && ready,
			"stream_ready":       ready,
			"mediamtx_reachable": mediamtxReachable,
			"path_registered":    pathRegistered,
			"source_ready":       ready,
			"viewers":            viewers,
		},
	})
}

// RegisterActiveStreams mendaftarkan seluruh CCTV stream_enabled ke MediaMTX
// saat aplikasi mulai berjalan (dipanggil sebagai goroutine). Sebelum iterasi,
// menunggu MediaMTX siap; tiap CCTV didaftarkan dengan retry terbatas.
func RegisterActiveStreams(db *gorm.DB) {
	if db == nil {
		return
	}
	if !waitForMediaMTX(2 * time.Minute) {
		utils.Log.Error("MediaMTX tidak siap setelah 2 menit, stream registration dibatalkan")
		return
	}

	var cctvs []models.CCTV
	if err := db.Where("stream_enabled = ?", true).Find(&cctvs).Error; err != nil {
		utils.Log.Errorf("❌ Gagal memuat CCTV aktif untuk auto-register: %v", err)
		return
	}

	successCount, failCount := 0, 0
	for _, cctv := range cctvs {
		if err := registerStreamWithRetry(cctv, 5); err != nil {
			utils.Log.Warnf("⚠️ Gagal register stream CCTV %s (%s): %v", cctv.Nama, cctv.KodeCCTV, err)
			failCount++
		} else {
			successCount++
		}
	}
	utils.Log.Infof("Stream registration selesai: %d berhasil, %d gagal", successCount, failCount)
}

// waitForMediaMTX mem-poll endpoint config MediaMTX hingga siap atau timeout.
func waitForMediaMTX(maxWait time.Duration) bool {
	deadline := time.Now().Add(maxWait)
	for {
		if err := streamClient.Ping(); err == nil {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(2 * time.Second)
	}
}

// registerStreamWithRetry mendaftarkan stream satu CCTV ke MediaMTX dengan
// exponential backoff (2s, 4s, 8s, 16s, 30s) hingga maxRetry percobaan.
func registerStreamWithRetry(cctv models.CCTV, maxRetry int) error {
	decrypted := ""
	if cctv.RTSPURL == "" {
		if cctv.RTSPPassword == "" {
			return fmt.Errorf("password RTSP kosong")
		}
		var err error
		decrypted, err = utils.DecryptAES256(cctv.RTSPPassword)
		if err != nil {
			return fmt.Errorf("gagal mendekripsi password RTSP: %w", err)
		}
	}

	rtspURL := utils.ResolveRTSPURL(cctv.RTSPURL, cctv.ChannelPort, decrypted, utils.SubStream)
	path := utils.BuildStreamPath(cctv.ID.String())
	pathCfg := utils.PathConfig{
		Source:                   rtspURL,
		SourceOnDemand:           true,
		SourceOnDemandCloseAfter: "30s",
		RTSPTransport:            "tcp",
	}

	backoff := []time.Duration{2 * time.Second, 4 * time.Second, 8 * time.Second, 16 * time.Second, 30 * time.Second}
	for attempt := 0; attempt < maxRetry; attempt++ {
		err := streamClient.AddPath(path, pathCfg)
		if err == nil {
			utils.Log.Infof("✅ Stream CCTV %s (%s) terdaftar ke MediaMTX pada path %s", cctv.Nama, cctv.KodeCCTV, path)
			return nil
		}
		// Path sudah ada → dianggap berhasil.
		if strings.Contains(err.Error(), "already exists") {
			utils.Log.Infof("ℹ️ Path streaming %s sudah terdaftar — dianggap berhasil", path)
			return nil
		}
		utils.Log.Warnf("⚠️ Percobaan %d/%d register stream %s gagal: %v", attempt+1, maxRetry, path, err)
		if attempt < maxRetry-1 {
			time.Sleep(backoff[attempt])
		}
	}
	return fmt.Errorf("gagal setelah %d percobaan", maxRetry)
}

// ReRegisterFailedStreams memeriksa CCTV stream_enabled yang belum terdaftar di
// MediaMTX lalu mencoba mendaftarkan ulang. Dipanggil berkala oleh scheduler.
func ReRegisterFailedStreams(db *gorm.DB) {
	if db == nil {
		return
	}
	var cctvs []models.CCTV
	if err := db.Where("stream_enabled = ?", true).Find(&cctvs).Error; err != nil {
		utils.Log.Errorf("❌ Gagal memuat CCTV untuk re-register: %v", err)
		return
	}

	// Bangun set path yang sudah ada di MediaMTX.
	existing := map[string]bool{}
	if paths, err := streamClient.ListPaths(); err == nil {
		for _, p := range paths {
			existing[p.Name] = true
		}
	} else {
		utils.Log.Warnf("⚠️ Gagal mengambil daftar path MediaMTX untuk re-register: %v", err)
		return
	}

	for _, cctv := range cctvs {
		path := utils.BuildStreamPath(cctv.ID.String())
		if existing[path] {
			continue
		}
		if err := registerStreamWithRetry(cctv, 3); err != nil {
			utils.Log.Warnf("⚠️ Re-register stream CCTV %s (%s) gagal: %v", cctv.Nama, cctv.KodeCCTV, err)
		}
	}
}

// RunStreamReRegistrar menjalankan ReRegisterFailedStreams sekali segera lalu
// berulang setiap 10 menit untuk menutup celah stream yang belum terdaftar.
func RunStreamReRegistrar(db *gorm.DB) {
	if db == nil {
		return
	}
	ReRegisterFailedStreams(db)
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		ReRegisterFailedStreams(db)
	}
}

// RunStreamHealthMonitor memantau status kesiapan stream tiap CCTV setiap 5
// menit (Task 10.4). Berjalan sekali segera saat startup, lalu berulang. Tidak
// crash bila MediaMTX tidak dapat dihubungi — cukup log peringatan & lanjut.
func RunStreamHealthMonitor(db *gorm.DB) {
	if db == nil {
		return
	}

	runStreamHealthCycle(db)

	ticker := time.NewTicker(streamHealthMonitorInterval)
	defer ticker.Stop()
	for range ticker.C {
		runStreamHealthCycle(db)
	}
}

// runStreamHealthCycle mengambil daftar path dari MediaMTX lalu mengevaluasi
// status seluruh CCTV stream_enabled pada satu siklus monitor. Bila MediaMTX
// tidak dapat dihubungi, log peringatan & lanjut dengan peta kesiapan kosong.
func runStreamHealthCycle(db *gorm.DB) {
	ready := map[string]bool{}
	if paths, err := streamClient.ListPaths(); err != nil {
		utils.Log.Warnf("⚠️ Gagal mengambil daftar path dari MediaMTX pada monitor status stream: %v", err)
	} else {
		for _, p := range paths {
			ready[p.Name] = p.SourceReady()
		}
	}
	runStreamHealthOnce(db, ready)
}

// runStreamHealthOnce memproses satu siklus evaluasi status stream terhadap peta
// kesiapan (readyMap). Dipakai oleh monitor dan langsung oleh pengujian agar
// transisi down/restored dapat disimulasikan secara deterministik.
func runStreamHealthOnce(db *gorm.DB, readyMap map[string]bool) {
	var cctvs []models.CCTV
	if err := db.Where("stream_enabled = ?", true).Find(&cctvs).Error; err != nil {
		utils.Log.Errorf("❌ Gagal memuat CCTV untuk monitor status stream: %v", err)
		return
	}

	now := time.Now()
	for _, cctv := range cctvs {
		path := utils.BuildStreamPath(cctv.ID.String())
		evaluateStreamState(db, cctv, path, readyMap[path], now)
	}
}

// evaluateStreamState memperbarui state kesehatan satu stream berdasarkan status
// kesiapan saat ini (isReady) dan waktu kini (now). Transisi down melewati ambang
// 15 menit memicu event STREAM_DOWN + penalti health_score 10 poin; pemulihan
// memicu event STREAM_RESTORED. Akses state dilindungi mutex.
func evaluateStreamState(db *gorm.DB, cctv models.CCTV, path string, isReady bool, now time.Time) {
	streamHealthMu.Lock()
	defer streamHealthMu.Unlock()

	if isReady {
		// Stream siap — catat waktu ready terakhir dan bersihkan status down.
		_, wasDown := streamDownSince[path]
		streamLastReady[path] = now
		delete(streamDownSince, path)
		if wasDown {
			delete(streamDownNotified, path)
			utils.Log.Infof("🟢 Stream CCTV %s (%s) pulih — path %s", cctv.Nama, cctv.KodeCCTV, path)
			broadcastStreamEvent("STREAM_RESTORED", fiber.Map{
				"cctv_id": cctv.ID.String(),
				"path":    path,
			})
		}
		return
	}

	// Path tidak siap (isReady=false).
	// Gunakan hasil probe RTSP (stream_live) sebagai sumber kebenaran bila
	// tersedia. Nilai ini dimuat dari DB saat runStreamHealthOnce sehingga
	// mencerminkan hasil probe terakhir dari RTSP Probe Engine (Task 14A.1).
	live := cctv.StreamLive
	if !live {
		// Kamera dianggap down oleh probe — mulai countdown tanpa perlu menunggu
		// pernah ada viewer (memperbaiki bug kamera yang tidak pernah ditonton).
		if _, alreadyDown := streamDownSince[path]; !alreadyDown {
			streamDownSince[path] = now
		}

		// Sudah dalam status down — cek apakah melewati ambang 15 menit.
		if now.Sub(streamDownSince[path]) > streamDownThreshold {
			if streamDownNotified[path] {
				return
			}
			streamDownNotified[path] = true
			utils.Log.Warnf("🔴 Stream CCTV %s (%s) down lebih dari 15 menit — path %s", cctv.Nama, cctv.KodeCCTV, path)
			broadcastStreamEvent("STREAM_DOWN", fiber.Map{
				"cctv_id": cctv.ID.String(),
				"path":    path,
			})

			// Task 12.3: dispatch event webhook stream.down.
			DispatchWebhook(db, models.EventWebhookStreamDown, fiber.Map{
				"cctv_id":    cctv.ID.String(),
				"kode_cctv":  cctv.KodeCCTV,
				"nama":       cctv.Nama,
				"path":       path,
				"down_since": streamDownSince[path].Format(time.RFC3339),
				"timestamp":  now.Format(time.RFC3339),
			})

			// Task 11.3: kirim email peringatan CCTV offline ke seluruh admin yang
			// opt-in. Async agar tidak memperlambat monitor.
			go SendCCTVStreamDownEmail(db, &cctv)

			// Kurangi skor kesehatan 10 poin (clamp min 0).
			newScore := cctv.HealthScore - 10
			if newScore < 0 {
				newScore = 0
			}
			if err := db.Model(&models.CCTV{}).Where("id = ?", cctv.ID).Update("health_score", newScore).Error; err != nil {
				utils.Log.Errorf("❌ Gagal memperbarui health_score CCTV %s: %v", cctv.ID.String(), err)
			}
		}
		return
	}

	// live=true: stream dianggap ready — reset state down (mirip blok isReady di atas).
	_, wasDown := streamDownSince[path]
	streamLastReady[path] = now
	delete(streamDownSince, path)
	if wasDown {
		delete(streamDownNotified, path)
		utils.Log.Infof("🟢 Stream CCTV %s (%s) pulih — path %s", cctv.Nama, cctv.KodeCCTV, path)
		broadcastStreamEvent("STREAM_RESTORED", fiber.Map{
			"cctv_id": cctv.ID.String(),
			"path":    path,
		})
	}
}

// GetStreamHealth menampilkan status kesehatan stream seluruh CCTV.
// @Summary      Status Kesehatan Stream
// @Description  Menampilkan ringkasan status kesehatan streaming setiap CCTV termasuk kesiapan path, jumlah penonton, status down, dan durasi down.
// @Tags         Streaming
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "Daftar status kesehatan stream"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Failure      500 {object} map[string]interface{} "Gagal memuat data CCTV"
// @Router       /streams/health [get]
func GetStreamHealth(c *fiber.Ctx) error {
	if config.DB == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Database belum tersedia",
		})
	}
	var cctvs []models.CCTV
	if err := config.DB.Find(&cctvs).Error; err != nil {
		utils.Log.Errorf("❌ Gagal memuat CCTV untuk status kesehatan stream: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal memuat data CCTV",
		})
	}

	// Ambil daftar path dari MediaMTX untuk membangun peta path → PathInfo.
	// Bila MediaMTX tidak dapat dihubungi, log peringatan & anggap semua tidak siap.
	pathMap := map[string]utils.PathInfo{}
	if paths, err := streamClient.ListPaths(); err != nil {
		utils.Log.Warnf("⚠️ Gagal mengambil daftar path dari MediaMTX untuk status kesehatan: %v", err)
	} else {
		for _, p := range paths {
			pathMap[p.Name] = p
		}
	}

	result := make([]fiber.Map, 0, len(cctvs))
	now := time.Now()
	for _, cctv := range cctvs {
		result = append(result, streamHealthSnapshot(cctv, pathMap, now))
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   result,
	})
}

// streamHealthSnapshot membangun ringkasan status kesehatan satu CCTV untuk
// endpoint /streams/health. pathMap memetakan stream path → info dari MediaMTX.
func streamHealthSnapshot(cctv models.CCTV, pathMap map[string]utils.PathInfo, now time.Time) fiber.Map {
	path := utils.BuildStreamPath(cctv.ID.String())
	info, exists := pathMap[path]
	ready := exists && info.SourceReady()
	viewers := 0
	if exists {
		viewers = len(info.Readers)
	}

	streamHealthMu.Lock()
	downSince, isDown := streamDownSince[path]
	streamHealthMu.Unlock()

	down := isDown
	downDuration := int64(0)
	if down {
		downDuration = int64(now.Sub(downSince).Seconds())
		if downDuration < 0 {
			downDuration = 0
		}
	}

	return fiber.Map{
		"cctv_id":               cctv.ID.String(),
		"nama":                  cctv.Nama,
		"kode_cctv":             cctv.KodeCCTV,
		"stream_enabled":        cctv.StreamEnabled,
		"stream_active":         cctv.StreamEnabled && exists && ready,
		"stream_ready":          ready,
		"viewers":               viewers,
		"down":                  down,
		"down_duration_seconds": downDuration,
		"whep_url":              "/stream/" + path + "/whep",
	}
}
