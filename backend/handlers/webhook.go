package handlers

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"sigap-backend/config"
	"sigap-backend/models"
	"sigap-backend/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// WebhookRetryMax adalah jumlah maksimal percobaan pengiriman.
const WebhookRetryMax = 3

// WebhookRetryDelays adalah jeda exponential backoff antar percobaan.
var WebhookRetryDelays = []time.Duration{1 * time.Second, 5 * time.Second, 25 * time.Second}

// CreateWebhookRequest adalah body pembuatan endpoint webhook (Admin saja).
type CreateWebhookRequest struct {
	Name   string   `json:"name" validate:"required,min=2,max=100"`
	URL    string   `json:"url" validate:"required,url"`
	Events []string `json:"events" validate:"required"`
	Secret string   `json:"secret" validate:"required,min=8"`
}

// UpdateWebhookRequest adalah body pembaruan endpoint webhook (Admin saja).
type UpdateWebhookRequest struct {
	Name     *string  `json:"name" validate:"omitempty,min=2,max=100"`
	URL      *string  `json:"url" validate:"omitempty,url"`
	Events   []string `json:"events"`
	Secret   *string  `json:"secret" validate:"omitempty,min=8"`
	IsActive *bool    `json:"is_active"`
}

// validWebhookEvent memeriksa apakah nama event webhook terdaftar.
func validWebhookEvent(event string) bool {
	switch event {
	case models.EventWebhookCCTVStatusChanged,
		models.EventWebhookMaintenanceCompleted,
		models.EventWebhookMaintenanceOverdue,
		models.EventWebhookHealthScoreChanged,
		models.EventWebhookStreamDown:
		return true
	}
	return false
}

// signPayload menghasilkan HMAC-SHA256 signature untuk header X-Signature-256.
func signPayload(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// webhookQueue adalah antrean event webhook. Handler bisnis hanya melempar
// event ke antrean ini (tanpa query DB) sehingga tidak memperlambat request.
// Worker (StartWebhookWorker) memproses antrean dan menulis ke DB. Dipakai
// pola decoupled: handler tidak bergantung pada ketersediaan endpoint webhook.
var webhookQueue = make(chan webhookQueueItem, 1024)

// webhookQueueItem adalah satu event yang diantrekan untuk dikirim.
type webhookQueueItem struct {
	event   string
	payload interface{}
}

// StartWebhookWorker menjalankan goroutine yang mengonsumsi antrean event
// webhook dan memprosesnya (pencarian endpoint + pembuatan delivery + kirim).
// Dipanggil sekali dari main.go. Bila tidak dipanggil (mis. unit test), event
// tetap diantrekan namun tidak diproses — tidak menimbulkan query DB di handler.
func StartWebhookWorker(db *gorm.DB) {
	if db == nil {
		return
	}
	go func() {
		for item := range webhookQueue {
			dispatchWebhookSync(db, item.event, item.payload)
		}
	}()
}

// DispatchWebhook mengantrekan event webhook untuk diproses asinkron oleh
// worker. Dipanggil setelah event bisnis terjadi (hook). Tidak melakukan query
// DB — hanya menempatkan event ke antrean buffer.
func DispatchWebhook(db *gorm.DB, event string, payload interface{}) {
	if db == nil {
		return
	}
	select {
	case webhookQueue <- webhookQueueItem{event: event, payload: payload}:
	default:
		// Antrean penuh — lewati untuk menghindari pemblokiran request.
		utils.Log.Warnf("⚠️ Antrean webhook penuh, event %s dilewati", event)
	}
}

// dispatchWebhookSync melakukan pekerjaan pencarian endpoint, pembuatan record
// delivery, dan pengiriman — dijalankan oleh worker webhook.
func dispatchWebhookSync(db *gorm.DB, event string, payload interface{}) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		utils.Log.Errorf("❌ Gagal marshal payload webhook %s: %v", event, err)
		return
	}

	var endpoints []models.WebhookEndpoint
	if err := db.Where("is_active = ? AND deleted_at IS NULL", true).Find(&endpoints).Error; err != nil {
		utils.Log.Errorf("❌ Gagal memuat endpoint webhook untuk event %s: %v", event, err)
		return
	}

	for i := range endpoints {
		// Cek apakah endpoint berlangganan event ini.
		subscribed := false
		for _, e := range endpoints[i].Events {
			if e == event {
				subscribed = true
				break
			}
		}
		if !subscribed {
			continue
		}

		// Buat record delivery (pending) lalu kirim async.
		delivery := models.WebhookDelivery{
			WebhookEndpointID: endpoints[i].ID,
			Event:             event,
			Payload:           string(payloadBytes),
			Status:            models.WebhookDeliveryPending,
			Attempt:           0,
		}
		if err := db.Create(&delivery).Error; err != nil {
			utils.Log.Errorf("❌ Gagal menyimpan delivery webhook %s: %v", event, err)
			continue
		}
		go deliverWebhook(db, delivery.ID, endpoints[i], event, payloadBytes)
	}
}

// deliverWebhook mengirim payload ke URL endpoint dengan retry exponential
// backoff dan mencatat hasil ke WebhookDelivery.
func deliverWebhook(db *gorm.DB, deliveryID uuid.UUID, endpoint models.WebhookEndpoint, event string, payload []byte) {
	client := &http.Client{Timeout: 10 * time.Second}

	attempts := len(WebhookRetryDelays)
	for attempt := 1; attempt <= attempts; attempt++ {
		start := time.Now()
		statusCode, respBody, err := sendWebhookRequest(client, endpoint.URL, payload, endpoint.Secret)

		// Update delivery record (async lanjutan dalam goroutine ini).
		if err == nil && statusCode >= 200 && statusCode < 300 {
			updateWebhookDelivery(db, deliveryID, attempt, statusCode, respBody, time.Since(start), models.WebhookDeliverySuccess)
			return
		}

		// Simpan kemajuan attempt (tetap failed jika ini attempt terakhir).
		status := models.WebhookDeliveryFailed
		if attempt < attempts {
			status = models.WebhookDeliveryPending
		}
		updateWebhookDelivery(db, deliveryID, attempt, statusCode, respBody, time.Since(start), status)

		if attempt < attempts {
			time.Sleep(WebhookRetryDelays[attempt-1])
		}
	}
}

// sendWebhookRequest melakukan satu POST ke URL dengan header signature.
func sendWebhookRequest(client *http.Client, url string, payload []byte, secret string) (int, string, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "SIGAP-Webhook/1.0")
	req.Header.Set("X-Event", "")
	if secret != "" {
		req.Header.Set("X-Signature-256", signPayload(payload, secret))
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	// Batasi panjang body yang disimpan.
	if len(body) > 2000 {
		body = body[:2000]
	}
	return resp.StatusCode, string(body), nil
}

// updateWebhookDelivery memperbarui record pengiriman.
func updateWebhookDelivery(db *gorm.DB, id uuid.UUID, attempt, code int, body string, latency time.Duration, status string) {
	if err := db.Model(&models.WebhookDelivery{}).Where("id = ?", id).Updates(map[string]interface{}{
		"attempt":        attempt,
		"response_code":  code,
		"response_body":  body,
		"latency":        latency.Milliseconds(),
		"status":         status,
	}).Error; err != nil {
		utils.Log.Errorf("❌ Gagal memperbarui delivery webhook %s: %v", id.String(), err)
	}
}

// ---- CRUD Handlers ----

// CreateWebhookEndpoint membuat endpoint webhook baru (Admin saja).
// @Summary      Buat Webhook Endpoint
// @Description  Membuat endpoint webhook baru dengan daftar event yang dilanggan dan secret untuk signature HMAC.
// @Tags         Webhook
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body CreateWebhookRequest true "Data Webhook Endpoint Baru"
// @Success      201 {object} map[string]interface{} "Webhook endpoint berhasil dibuat"
// @Failure      400 {object} map[string]interface{} "Validasi gagal atau event tidak valid"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Failure      500 {object} map[string]interface{} "Gagal menyimpan webhook endpoint"
// @Router       /webhooks [post]
func CreateWebhookEndpoint(c *fiber.Ctx) error {
	userIDStr, ok := c.Locals("user_id").(string)
	if !ok || userIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Identitas pengguna tidak ditemukan",
		})
	}
	createdBy, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Identitas pengguna tidak valid",
		})
	}

	var req CreateWebhookRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Format input tidak valid",
		})
	}
	if errs := utils.ValidateStruct(&req); len(errs) > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Validasi input gagal",
			"errors":  errs,
		})
	}

	for _, e := range req.Events {
		if !validWebhookEvent(e) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "error",
				"message": "Event tidak valid. Gunakan: cctv.status_changed, maintenance.completed, maintenance.overdue, health_score.changed, stream.down",
			})
		}
	}

	item := models.WebhookEndpoint{
		Name:      req.Name,
		URL:       req.URL,
		Events:    req.Events,
		Secret:    req.Secret,
		IsActive:  true,
		CreatedBy: createdBy,
	}
	if err := config.DB.Create(&item).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menyimpan webhook endpoint",
		})
	}

	utils.LogAudit(config.DB, c, "CREATE_WEBHOOK", "WEBHOOK",
		"Membuat webhook endpoint: "+item.Name, nil, "", "")

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status": "success",
		"data":   item,
	})
}

// GetAllWebhookEndpoints mengambil daftar endpoint webhook terpaginasi.
// @Summary      Ambil Semua Webhook Endpoint
// @Description  Mendapatkan daftar endpoint webhook terpaginasi. Secret tidak dikembalikan.
// @Tags         Webhook
// @Produce      json
// @Security     BearerAuth
// @Param        page query integer false "Nomor Halaman (default: 1)"
// @Param        limit query integer false "Jumlah data per halaman (default: 10)"
// @Success      200 {object} map[string]interface{} "Daftar webhook endpoint terpaginasi"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      500 {object} map[string]interface{} "Gagal mengambil daftar webhook endpoint"
// @Router       /webhooks [get]
func GetAllWebhookEndpoints(c *fiber.Ctx) error {
	var items []models.WebhookEndpoint
	allowedSort := []string{"id", "name", "url", "is_active", "created_at"}
	limit, page, sort := utils.GetPaginationParams(c, allowedSort)

	var totalRows int64
	if err := config.DB.Model(&models.WebhookEndpoint{}).Count(&totalRows).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghitung total webhook endpoint",
		})
	}

	offset := (page - 1) * limit
	if err := config.DB.Preload("User").Offset(offset).Limit(limit).Order(sort).Find(&items).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal mengambil daftar webhook endpoint",
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"rows":       items,
			"pagination": utils.CalculatePagination(totalRows, page, limit),
		},
	})
}

// GetWebhookEndpointByID mengambil detail satu endpoint webhook.
// @Summary      Ambil Webhook Endpoint Detail
// @Description  Mendapatkan detail satu endpoint webhook beserta riwayat pengiriman terbaru.
// @Tags         Webhook
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Webhook Endpoint UUID"
// @Success      200 {object} map[string]interface{} "Data webhook endpoint"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      404 {object} map[string]interface{} "Webhook endpoint tidak ditemukan"
// @Router       /webhooks/{id} [get]
func GetWebhookEndpointByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID webhook endpoint tidak valid",
		})
	}

	var item models.WebhookEndpoint
	if err := config.DB.First(&item, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Webhook endpoint tidak ditemukan",
		})
	}

	// Ambil delivery terbaru.
	var deliveries []models.WebhookDelivery
	config.DB.Where("webhook_endpoint_id = ?", id).Order("created_at desc").Limit(20).Find(&deliveries)

	return c.JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"endpoint":   item,
			"deliveries": deliveries,
		},
	})
}

// UpdateWebhookEndpoint memperbarui endpoint webhook (Admin saja).
// @Summary      Update Webhook Endpoint
// @Description  Memperbarui nama, URL, event, secret, atau status aktif endpoint webhook.
// @Tags         Webhook
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Webhook Endpoint UUID"
// @Param        request body UpdateWebhookRequest true "Data Update Webhook Endpoint"
// @Success      200 {object} map[string]interface{} "Webhook endpoint berhasil diperbarui"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid, format body salah, atau validasi gagal"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Failure      404 {object} map[string]interface{} "Webhook endpoint tidak ditemukan"
// @Failure      500 {object} map[string]interface{} "Gagal memperbarui webhook endpoint"
// @Router       /webhooks/{id} [put]
func UpdateWebhookEndpoint(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID webhook endpoint tidak valid",
		})
	}

	var item models.WebhookEndpoint
	if err := config.DB.First(&item, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Webhook endpoint tidak ditemukan",
		})
	}

	var req UpdateWebhookRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Format input tidak valid",
		})
	}
	if errs := utils.ValidateStruct(&req); len(errs) > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Validasi input gagal",
			"errors":  errs,
		})
	}

	if req.Name != nil {
		item.Name = *req.Name
	}
	if req.URL != nil {
		item.URL = *req.URL
	}
	if req.Events != nil {
		for _, e := range req.Events {
			if !validWebhookEvent(e) {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"status":  "error",
					"message": "Event tidak valid: " + e,
				})
			}
		}
		item.Events = req.Events
	}
	if req.Secret != nil {
		item.Secret = *req.Secret
	}
	if req.IsActive != nil {
		item.IsActive = *req.IsActive
	}

	if err := config.DB.Save(&item).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal memperbarui webhook endpoint",
		})
	}

	utils.LogAudit(config.DB, c, "UPDATE_WEBHOOK", "WEBHOOK",
		"Memperbarui webhook endpoint: "+item.Name, nil, "", "")

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   item,
	})
}

// DeleteWebhookEndpoint menghapus endpoint webhook secara soft-delete (Admin).
// @Summary      Hapus Webhook Endpoint
// @Description  Menghapus endpoint webhook (soft-delete). Riwayat pengiriman tetap tersimpan.
// @Tags         Webhook
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Webhook Endpoint UUID"
// @Success      200 {object} map[string]interface{} "Webhook endpoint berhasil dihapus"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Failure      404 {object} map[string]interface{} "Webhook endpoint tidak ditemukan"
// @Failure      500 {object} map[string]interface{} "Gagal menghapus webhook endpoint"
// @Router       /webhooks/{id} [delete]
func DeleteWebhookEndpoint(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID webhook endpoint tidak valid",
		})
	}

	var item models.WebhookEndpoint
	if err := config.DB.First(&item, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Webhook endpoint tidak ditemukan",
		})
	}

	if err := config.DB.Delete(&item).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghapus webhook endpoint",
		})
	}

	utils.LogAudit(config.DB, c, "DELETE_WEBHOOK", "WEBHOOK",
		"Menghapus webhook endpoint: "+item.Name, nil, "", "")

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Webhook endpoint berhasil dihapus",
	})
}

// TestWebhook mengirim payload uji ke endpoint dan mengembalikan hasilnya
// (tanpa menyimpan delivery permanen). Admin saja.
// @Summary      Test Webhook Delivery
// @Description  Mengirim payload uji (event test.ping) ke endpoint dan mengembalikan status respons serta latensi.
// @Tags         Webhook
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Webhook Endpoint UUID"
// @Success      200 {object} map[string]interface{} "Hasil test delivery"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Failure      404 {object} map[string]interface{} "Webhook endpoint tidak ditemukan"
// @Failure      500 {object} map[string]interface{} "Gagal mengirim test"
// @Router       /webhooks/{id}/test [post]
func TestWebhook(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID webhook endpoint tidak valid",
		})
	}

	var item models.WebhookEndpoint
	if err := config.DB.First(&item, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Webhook endpoint tidak ditemukan",
		})
	}

	payload := fiber.Map{
		"event":     "test.ping",
		"timestamp": time.Now().Format(time.RFC3339),
		"message":   "Test webhook dari SIGAP",
	}
	payloadBytes, _ := json.Marshal(payload)

	client := &http.Client{Timeout: 10 * time.Second}
	start := time.Now()
	code, body, err := sendWebhookRequest(client, item.URL, payloadBytes, item.Secret)
	latency := time.Since(start)

	result := fiber.Map{
		"status":       "success",
		"endpoint_id":  item.ID,
		"event":        "test.ping",
		"response_code": code,
		"latency_ms":   latency.Milliseconds(),
	}
	if err != nil {
		result["status"] = "error"
		result["error"] = err.Error()
		return c.JSON(result)
	}
	result["response_body"] = body
	return c.JSON(result)
}

// GetWebhookDeliveries mengambil riwayat pengiriman untuk satu endpoint.
// @Summary      Ambil Riwayat Webhook Delivery
// @Description  Mendapatkan daftar pengiriman webhook untuk sebuah endpoint (paginasi, terbaru dahulu).
// @Tags         Webhook
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Webhook Endpoint UUID"
// @Param        page query integer false "Nomor Halaman (default: 1)"
// @Param        limit query integer false "Jumlah data per halaman (default: 20)"
// @Success      200 {object} map[string]interface{} "Daftar delivery terpaginasi"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      500 {object} map[string]interface{} "Gagal mengambil riwayat pengiriman"
// @Router       /webhooks/{id}/deliveries [get]
func GetWebhookDeliveries(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID webhook endpoint tidak valid",
		})
	}

	limit, page, _ := utils.GetPaginationParams(c, []string{"created_at", "event", "status"})
	var totalRows int64
	if err := config.DB.Model(&models.WebhookDelivery{}).
		Where("webhook_endpoint_id = ?", id).Count(&totalRows).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghitung riwayat pengiriman",
		})
	}

	var items []models.WebhookDelivery
	offset := (page - 1) * limit
	if err := config.DB.Where("webhook_endpoint_id = ?", id).
		Order("created_at desc").Offset(offset).Limit(limit).Find(&items).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal mengambil riwayat pengiriman",
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"rows":       items,
			"pagination": utils.CalculatePagination(totalRows, page, limit),
		},
	})
}


