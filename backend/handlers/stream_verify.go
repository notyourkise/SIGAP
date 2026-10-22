package handlers

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"sigap-backend/config"
	"sigap-backend/models"
	"sigap-backend/utils"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Verifikasi aktif stream membedakan status pasif (path on-demand terdaftar)
// dari bukti bahwa sumber benar-benar mengirim track. Probe pasif tetap
// dipertahankan untuk kompatibilitas; hasil verifikasi disimpan terpisah.
const (
	defaultStreamVerifyInterval    = 15 * time.Minute
	defaultStreamVerifyConcurrency = 1
	streamVerifyFailureThreshold   = 2
	streamVerifyFreshness          = 30 * time.Minute
	streamVerifyMethodHLS          = "mediamtx-hls"
	streamVerifyMethodLive         = "mediamtx-live"
)

// Durasi ini variabel agar pengujian dapat mempercepat polling tanpa membuat
// test menunggu timeout produksi 8 detik.
var (
	streamVerifyTimeout      = 8 * time.Second
	streamVerifyPollInterval = 500 * time.Millisecond
)

// streamVerificationResult adalah hasil satu percobaan verifikasi. Durasi
// dihitung sejak percobaan dimulai agar endpoint manual dapat menampilkan
// berapa lama verifikasi berjalan.
type streamVerificationResult struct {
	CCTV       models.CCTV
	State      string
	VerifiedAt time.Time
	Method     string
	Failures   int
	Tracks     int
	Duration   time.Duration
	Error      string
}

// RunStreamVerifier menjalankan verifikasi end-to-end secara periodik.
// Verifikasi pertama ditunda agar registrasi path ke MediaMTX selesai lebih
// dahulu; selanjutnya berjalan sesuai CCTV_VERIFY_INTERVAL_MINUTES.
func RunStreamVerifier(db *gorm.DB) {
	if db == nil {
		return
	}

	// Beri waktu RegisterActiveStreams/RunStreamReRegistrar menyiapkan path.
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

// streamVerifyInterval membaca interval dari environment. Nilai di luar
// 5–30 menit diabaikan agar status terverifikasi tidak lebih lama dari masa
// berlakunya (30 menit) dan probe tidak membebani NVR/kamera.
func streamVerifyInterval() time.Duration {
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

// streamVerifyConcurrency membaca jumlah worker verifikasi (1–4).
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

// runStreamVerifyCycle memverifikasi seluruh CCTV aktif dengan worker pool
// terbatas. Pembatasan ini penting karena setiap verifikasi membuka stream
// sementara ke NVR; menjalankan 20 kamera sekaligus dapat membebani jaringan.
func runStreamVerifyCycle(db *gorm.DB) {
	if db == nil {
		return
	}

	var cctvs []models.CCTV
	if err := db.Where("stream_enabled = ?", true).Find(&cctvs).Error; err != nil {
		utils.Log.Errorf("❌ Gagal memuat CCTV untuk verifikasi stream: %v", err)
		return
	}
	if len(cctvs) == 0 {
		return
	}

	started := time.Now()
	jobs := make(chan models.CCTV)
	results := make(chan streamVerificationResult, len(cctvs))

	var wg sync.WaitGroup
	for i := 0; i < streamVerifyConcurrency(); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for cctv := range jobs {
				results <- verifyCCTVStream(db, cctv)
				// Jeda kecil antar kamera pada worker yang sama agar pembukaan
				// stream tidak menumpuk pada NVR.
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
	for result := range results {
		switch result.State {
		case models.StreamStateLive:
			live++
		case models.StreamStateOffline:
			offline++
		default:
			unknown++
		}
	}

	utils.Log.Infof("🔎 Verifikasi stream selesai: %d kamera | live=%d offline=%d unknown=%d | durasi=%s",
		len(cctvs), live, offline, unknown, time.Since(started).Round(time.Millisecond))
}

// verifyCCTVStream melakukan verifikasi satu CCTV:
//  1. Bila path sudah live (ada penonton/sumber aktif), langsung live.
//  2. Bila idle, picu path on-demand lewat HLS, lalu poll status path.
//  3. Path yang tetap tidak siap setelah timeout dihitung gagal; offline baru
//     ditetapkan setelah dua kegagalan berturut-turut.
//
// MediaMTX yang tidak dapat dihubungi atau path yang belum terdaftar tidak
// dianggap sebagai kamera offline — hasilnya unknown agar tidak false alarm.
func verifyCCTVStream(db *gorm.DB, cctv models.CCTV) streamVerificationResult {
	started := time.Now()
	path := utils.BuildStreamPath(cctv.ID.String())
	result := streamVerificationResult{
		CCTV:       cctv,
		State:      models.StreamStateUnknown,
		VerifiedAt: started,
		Method:     streamVerifyMethodHLS,
		Failures:   cctv.StreamVerifyFailures,
	}

	info, err := streamClient.GetPath(path)
	if err != nil {
		result.Error = "path stream belum terdaftar di MediaMTX"
		if !isMediaMTXNotFound(err) {
			result.Error = "MediaMTX tidak dapat dihubungi"
		}
		result.Duration = time.Since(started)
		return persistStreamVerification(db, result)
	}

	// Path yang sudah menyala tidak perlu dipicu ulang; ini juga menghindari
	// gangguan pada penonton yang sedang aktif.
	if info.SourceReady() {
		result.State = models.StreamStateLive
		result.Method = streamVerifyMethodLive
		result.Tracks = len(info.Tracks)
		result.Failures = 0
		result.Duration = time.Since(started)
		return persistStreamVerification(db, result)
	}

	if err := streamClient.TriggerHLS(path, 2*time.Second); err != nil {
		// Kegagalan koneksi ke HLS bukan bukti kamera mati. Jangan menaikkan
		// failure counter; status tetap unknown sampai verifikasi berikutnya.
		result.Error = "gagal memicu HLS MediaMTX"
		result.Duration = time.Since(started)
		return persistStreamVerification(db, result)
	}

	deadline := time.Now().Add(streamVerifyTimeout)
	for {
		info, err = streamClient.GetPath(path)
		if err == nil && info.SourceReady() {
			result.State = models.StreamStateLive
			result.Tracks = len(info.Tracks)
			result.Failures = 0
			result.Duration = time.Since(started)
			return persistStreamVerification(db, result)
		}
		if time.Now().After(deadline) {
			break
		}
		time.Sleep(streamVerifyPollInterval)
	}

	// Query terakhir memisahkan "path hilang/API error" dari "sumber tidak siap".
	info, err = streamClient.GetPath(path)
	if err != nil {
		result.Error = "MediaMTX tidak dapat dihubungi saat verifikasi"
		result.Duration = time.Since(started)
		return persistStreamVerification(db, result)
	}
	if info.SourceReady() {
		result.State = models.StreamStateLive
		result.Tracks = len(info.Tracks)
		result.Failures = 0
		result.Duration = time.Since(started)
		return persistStreamVerification(db, result)
	}

	result.Failures = cctv.StreamVerifyFailures + 1
	// Satu kegagalan belum cukup untuk menyatakan offline, tetapi status live
	// lama juga tidak boleh dipertahankan — hasil terakhir adalah belum pasti.
	result.State = models.StreamStateUnknown
	if result.Failures >= streamVerifyFailureThreshold {
		result.State = models.StreamStateOffline
	}
	result.Error = fmt.Sprintf("sumber stream tidak siap setelah %s", streamVerifyTimeout)
	result.Duration = time.Since(started)
	return persistStreamVerification(db, result)
}

// persistStreamVerification menyimpan hasil verifikasi dan menyiarkan event
// WebSocket. Kolom pasif (stream_state/stream_checked_at) tidak diubah di sini
// agar probe pasif tetap menjadi sumber status real-time saat ada penonton.
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
	}

	payload := fiber.Map{
		"cctv_id":                result.CCTV.ID.String(),
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
	utils.BroadcastEvent("CCTV_LIVE_STATUS_UPDATED", payload)

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

// isMediaMTXNotFound mendeteksi respons 404 dari MediaMTX. Error yang bukan
// 404 diperlakukan sebagai MediaMTX tidak dapat dihubungi, bukan kamera offline.
func isMediaMTXNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "status 404") || strings.Contains(msg, "path not found")
}

// VerifyStream menjalankan verifikasi end-to-end satu CCTV secara manual.
// Endpoint ini admin-only dan mengembalikan hasil verifikasi tanpa menunggu
// siklus periodik berikutnya.
// @Summary      Verifikasi Status Stream CCTV
// @Description  Memicu pembacaan singkat melalui MediaMTX lalu memeriksa apakah sumber benar-benar siap. Hasil live/offline/unknown disimpan beserta waktu verifikasi.
// @Tags         Streaming
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "CCTV UUID"
// @Success      200 {object} map[string]interface{} "Hasil verifikasi stream"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid / streaming belum diaktifkan"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Failure      404 {object} map[string]interface{} "CCTV tidak ditemukan"
// @Failure      500 {object} map[string]interface{} "Database belum tersedia"
// @Router       /streams/{id}/verify [post]
func VerifyStream(c *fiber.Ctx) error {
	if config.DB == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Database belum tersedia",
		})
	}

	cctv, ok := loadCCTVByID(c)
	if !ok {
		return nil
	}
	if !cctv.StreamEnabled {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Streaming belum diaktifkan untuk CCTV ini",
		})
	}

	result := verifyCCTVStream(config.DB, cctv)

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
			"duration_ms":            result.Duration.Milliseconds(),
		},
	})
}

// isStreamVerificationFresh memastikan hasil verifikasi masih dalam masa
// berlaku dan bukan timestamp dari masa depan (mis. karena clock skew).
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

// effectiveStreamState memilih status terverifikasi bila masih segar, lalu
// jatuh kembali ke status pasif. Dipakai endpoint live-status dan dapat dipakai
// konsumen API yang membutuhkan satu nilai status ringkas.
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
