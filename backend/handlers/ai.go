package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"sigap-backend/config"
	"sigap-backend/models"
	"sigap-backend/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// defaultAISecret adalah nilai fallback shared secret antara backend dan
// service sigap_ai (Task 14B, Sprint 38). Pada deployment produksi, nilai ini
// WAJIB diganti via environment variable AI_SECRET (sama di docker-compose:
// AI_SECRET=${AI_SECRET:-sigap-ai-secret-change-me}).
const defaultAISecret = "sigap-ai-secret-change-me"

// aiServiceURL membaca base URL service sigap_ai dari environment variable
// AI_SERVICE_URL (default: http://sigap_ai:8010 — nama service di Docker).
func aiServiceURL() string {
	if v := os.Getenv("AI_SERVICE_URL"); v != "" {
		return v
	}
	return "http://sigap_ai:8010"
}

// aiSecret membaca shared secret AI dari environment variable AI_SECRET.
func aiSecret() string {
	if v := os.Getenv("AI_SECRET"); v != "" {
		return v
	}
	return defaultAISecret
}

// DetectionEventRequest adalah payload hasil deteksi yang dikirim service
// sigap_ai ke endpoint internal /api/ai/detection-event.
type DetectionEventRequest struct {
	CCTVID        string      `json:"cctv_id"`
	Bboxes        interface{} `json:"bboxes"`
	PersonCount   int         `json:"person_count"`
	UniqueCount   int         `json:"unique_count"`
	ConfidenceAvg float64     `json:"confidence_avg"`
}

// ReceiveDetectionEvent menerima hasil deteksi orang dari service sigap_ai.
// Endpoint INTERNAL — diproteksi dengan header X-AI-Secret (bukan JWT), karena
// dipanggil langsung oleh container sigap_ai di dalam Docker network.
//
//	@Summary      Terima Event Deteksi AI
//	@Description  Menerima hasil deteksi orang dari service sigap_ai, meng-update sesi harian (upsert), mencatat event, dan mem-broadcast AI_DETECTION_UPDATE via WebSocket. Endpoint internal diproteksi X-AI-Secret.
//	@Tags         AI Detection
//	@Accept       json
//	@Produce      json
//	@Param        X-AI-Secret header string true "Shared secret antara backend & sigap_ai"
//	@Param        request body DetectionEventRequest true "Payload hasil deteksi"
//	@Success      200 {object} map[string]interface{} "Event diterima"
//	@Failure      400 {object} map[string]interface{} "Body tidak valid / CCTV ID tidak valid"
//	@Failure      401 {object} map[string]interface{} "X-AI-Secret tidak cocok"
//	@Failure      500 {object} map[string]interface{} "Gagal menyimpan sesi/event"
//	@Router       /ai/detection-event [post]
func ReceiveDetectionEvent(c *fiber.Ctx) error {
	// Validasi shared secret.
	if c.Get("X-AI-Secret") != aiSecret() {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Unauthorized",
		})
	}

	var req DetectionEventRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Body tidak valid",
		})
	}

	cctvID, err := uuid.Parse(req.CCTVID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "CCTV ID tidak valid",
		})
	}

	// Tanggal sesi memakai hari server (WITA = Asia/Makassar, UTC+8). Konsisten
	// dengan daily reset counter di sigap_ai (00:00 WITA).
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Upsert ai_detection_sessions (satu baris per cctv_id + session_date).
	// Catatan: GORM Save() selalu UPDATE bila ID non-zero — karena ID baris baru
	// kita isi dengan uuid.New(), sesi baru harus dibuat dengan Create() agar
	// tidak menjadi UPDATE dengan created_at kosong.
	var session models.AIDetectionSession
	err = config.DB.Where("cctv_id = ? AND session_date = ?", cctvID, today).First(&session).Error
	if err != nil {
		// Belum ada sesi hari ini — buat baru.
		session = models.AIDetectionSession{
			ID:          uuid.New(),
			CCTVID:      cctvID,
			SessionDate: today,
		}
	}

	session.UniquePersonCount = req.UniqueCount
	// Simpan daftar tracking id terakhir (opsional; dikirim AI sebagai bboxes
	// — tidak diwajibkan mengisi kolom ini bila payload tidak membawanya).
	var saveErr error
	if session.CreatedAt.IsZero() {
		// Baris baru → INSERT.
		saveErr = config.DB.Create(&session).Error
	} else {
		// Baris existing → UPDATE (Save).
		saveErr = config.DB.Save(&session).Error
	}
	if saveErr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menyimpan sesi deteksi",
		})
	}

	// Simpan event hanya bila ada orang terdeteksi (throttle pengisian tabel).
	if req.PersonCount > 0 {
		event := models.AIDetectionEvent{
			ID:            uuid.New(),
			CCTVID:        cctvID,
			EventType:     "person_detected",
			PersonCount:   req.PersonCount,
			UniqueCount:   req.UniqueCount,
			ConfidenceAvg: req.ConfidenceAvg,
			DetectedAt:    now,
		}
		if err := config.DB.Create(&event).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "Gagal menyimpan event deteksi",
			})
		}
	}

	// Broadcast ke browser /monitor agar bounding box & counter tampil real-time.
	utils.BroadcastEvent("AI_DETECTION_UPDATE", fiber.Map{
		"cctv_id":          req.CCTVID,
		"bboxes":           req.Bboxes,
		"unique_today":     req.UniqueCount,
		"current_in_frame": req.PersonCount,
	})

	return c.JSON(fiber.Map{"status": "success"})
}

// GetAICounts menampilkan ringkasan hitungan orang unik hari ini per kamera.
// Hanya Admin yang dapat mengakses.
//
//	@Summary      Ringkasan Hitungan AI (Admin)
//	@Description  Menampilkan jumlah orang unik terdeteksi hari ini untuk seluruh CCTV yang memiliki sesi deteksi.
//	@Tags         AI Detection
//	@Produce      json
//	@Security     BearerAuth
//	@Success      200 {object} map[string]interface{} "Daftar ringkasan per kamera"
//	@Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
//	@Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
//	@Failure      500 {object} map[string]interface{} "Gagal mengambil data"
//	@Router       /ai/counts [get]
func GetAICounts(c *fiber.Ctx) error {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	var sessions []models.AIDetectionSession
	if err := config.DB.Where("session_date = ?", today).Find(&sessions).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal mengambil data hitungan AI",
		})
	}

	result := make([]fiber.Map, 0)
	for _, s := range sessions {
		var cctv models.CCTV
		if err := config.DB.Select("id,nama,kode_cctv").First(&cctv, "id = ?", s.CCTVID).Error; err != nil {
			// CCTV mungkin sudah dihapus — lewati baris ini.
			continue
		}
		result = append(result, fiber.Map{
			"cctv_id":     s.CCTVID,
			"nama":        cctv.Nama,
			"kode_cctv":   cctv.KodeCCTV,
			"unique_today": s.UniquePersonCount,
			"session_date": s.SessionDate,
		})
	}

	return c.JSON(fiber.Map{"status": "success", "data": result})
}

// GetAICountByCCTV menampilkan detail hitungan orang unik per kamera.
// Hanya Admin yang dapat mengakses.
//
//	@Summary      Detail Hitungan AI per Kamera (Admin)
//	@Description  Menampilkan jumlah orang unik terdeteksi hari ini untuk satu CCTV.
//	@Tags         AI Detection
//	@Produce      json
//	@Security     BearerAuth
//	@Param        id path string true "CCTV UUID"
//	@Success      200 {object} map[string]interface{} "Detail hitungan per kamera"
//	@Failure      400 {object} map[string]interface{} "ID tidak valid"
//	@Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
//	@Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
//	@Router       /ai/counts/{id} [get]
func GetAICountByCCTV(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID tidak valid",
		})
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	var session models.AIDetectionSession
	_ = config.DB.Where("cctv_id = ? AND session_date = ?", id, today).First(&session).Error

	return c.JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"cctv_id":      id,
			"unique_today": session.UniquePersonCount,
			"session_date": today,
		},
	})
}

// ProxyActivateAISession meneruskan permintaan aktivasi AI session dari
// frontend ke service sigap_ai. Body dikirim sebagai {active_cctv_ids:[...]}.
//
//	@Summary      Aktifkan AI Session (Proxy)
//	@Description  Meneruskan daftar CCTV aktif ke service sigap_ai agar mulai pull RTSP & deteksi orang.
//	@Tags         AI Detection
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        request body map[string]interface{} true "Body berisi active_cctv_ids"
//	@Success      200 {object} map[string]interface{} "Session diaktifkan"
//	@Failure      400 {object} map[string]interface{} "Body tidak valid"
//	@Failure      502 {object} map[string]interface{} "Gagal menghubungi service AI"
//	@Router       /ai/sessions [post]
func ProxyActivateAISession(c *fiber.Ctx) error {
	var req struct {
		ActiveCCTVIDs []string `json:"active_cctv_ids"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Body tidak valid",
		})
	}

	if len(req.ActiveCCTVIDs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "active_cctv_ids tidak boleh kosong",
		})
	}

	base := aiServiceURL()
	client := &http.Client{Timeout: 10 * time.Second}
	activated := make([]string, 0, len(req.ActiveCCTVIDs))
	failed := make([]fiber.Map, 0)

	for _, id := range req.ActiveCCTVIDs {
		if _, err := uuid.Parse(id); err != nil {
			failed = append(failed, fiber.Map{"cctv_id": id, "error": "ID tidak valid"})
			continue
		}
		body, _ := json.Marshal(map[string]interface{}{
			"cctv_id": id,
		})
		resp, err := client.Post(base+"/sessions", "application/json", bytes.NewReader(body))
		if err != nil {
			failed = append(failed, fiber.Map{"cctv_id": id, "error": "Gagal menghubungi service AI"})
			continue
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			activated = append(activated, id)
		} else {
			failed = append(failed, fiber.Map{"cctv_id": id, "error": fmt.Sprintf("Service AI menolak: %d", resp.StatusCode)})
		}
	}

	return c.JSON(fiber.Map{
		"status":    "success",
		"activated": activated,
		"failed":    failed,
	})
}

// ProxyDeactivateAISession meneruskan permintaan penonaktifan AI session ke
// service sigap_ai (DELETE /sessions/{id}).
//
//	@Summary      Nonaktifkan AI Session (Proxy)
//	@Description  Menghentikan pull RTSP & deteksi pada service sigap_ai untuk satu CCTV.
//	@Tags         AI Detection
//	@Produce      json
//	@Security     BearerAuth
//	@Param        id path string true "CCTV UUID"
//	@Success      200 {object} map[string]interface{} "Session dinonaktifkan"
//	@Failure      400 {object} map[string]interface{} "ID tidak valid"
//	@Failure      502 {object} map[string]interface{} "Gagal menghubungi service AI"
//	@Router       /ai/sessions/{id} [delete]
func ProxyDeactivateAISession(c *fiber.Ctx) error {
	id := c.Params("id")
	if _, err := uuid.Parse(id); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID tidak valid",
		})
	}

	base := aiServiceURL()
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodDelete, base+"/sessions/"+id, nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal membuat request",
		})
	}
	resp, err := client.Do(req)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghubungi service AI",
		})
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 300 {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"status":  "error",
			"message": fmt.Sprintf("Service AI menolak: %d", resp.StatusCode),
		})
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "AI session dinonaktifkan",
		"cctv_id": id,
	})
}

// ProxyListAISessions mengambil daftar session aktif dari service sigap_ai.
//
//	@Summary      Daftar AI Session Aktif (Admin)
//	@Description  Mengambil daftar session AI yang sedang aktif beserta jumlah orang unik hari ini dari service sigap_ai.
//	@Tags         AI Detection
//	@Produce      json
//	@Security     BearerAuth
//	@Success      200 {object} map[string]interface{} "Daftar session aktif"
//	@Failure      502 {object} map[string]interface{} "Gagal menghubungi service AI"
//	@Router       /ai/sessions [get]
func ProxyListAISessions(c *fiber.Ctx) error {
	base := aiServiceURL()
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(base + "/sessions")
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghubungi service AI",
		})
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal membaca respons service AI",
		})
	}

	// Teruskan respons mentah dari service AI.
	c.Set("Content-Type", "application/json")
	return c.Status(resp.StatusCode).Send(body)
}
