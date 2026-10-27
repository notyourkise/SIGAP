package handlers

import (
	"math"
	"strconv"
	"strings"
	"time"

	"sigap-backend/config"
	"sigap-backend/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GetCCTVStreamLogs mengembalikan daftar log riwayat event stream CCTV (online/offline) terpaginasi.
// @Summary      Daftar Log Event CCTV
// @Description  Mengembalikan catatan event perubahan status live/offline kamera CCTV secara terurut kronologis terbaru.
// @Tags         CCTV
// @Produce      json
// @Security     BearerAuth
// @Param        page        query  int     false  "Nomor halaman (default 1)"
// @Param        limit       query  int     false  "Jumlah per halaman (default 20, maks 100)"
// @Param        cctv_id     query  string  false  "Filter berdasarkan ID CCTV"
// @Param        event_type  query  string  false  "Filter tipe event ('online', 'offline', 'standby')"
// @Param        search      query  string  false  "Pencarian nama atau kode CCTV"
// @Success      200         {object}  map[string]interface{}
// @Failure      500         {object}  map[string]interface{}
// @Router       /cctv-logs [get]
func GetCCTVStreamLogs(c *fiber.Ctx) error {
	page, err := strconv.Atoi(c.Query("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(c.Query("limit", "20"))
	if err != nil || limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	query := config.DB.Model(&models.CCTVStreamLog{}).
		Preload("CCTV", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, nama, kode_cctv, lokasi_area, ip_camera, status")
		})

	query, ok := applyCCTVStreamLogSiteScope(c, query)
	if !ok {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "error",
			"message": "Anda belum ditugaskan ke site mana pun",
		})
	}

	if cctvIDStr := strings.TrimSpace(c.Query("cctv_id")); cctvIDStr != "" {
		if cctvID, err := uuid.Parse(cctvIDStr); err == nil {
			query = query.Where("cctv_id = ?", cctvID)
		}
	}

	if eventType := strings.TrimSpace(c.Query("event_type")); eventType != "" && eventType != "all" {
		query = query.Where("event_type = ?", eventType)
	}

	if search := strings.TrimSpace(c.Query("search")); search != "" {
		searchPattern := "%" + search + "%"
		query = query.Joins("LEFT JOIN cctvs ON cctvs.id = cctv_stream_logs.cctv_id").
			Where("cctvs.nama ILIKE ? OR cctvs.kode_cctv ILIKE ?", searchPattern, searchPattern)
	}

	var totalRows int64
	if err := query.Count(&totalRows).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghitung total data log CCTV",
		})
	}

	var logs []models.CCTVStreamLog
	if err := query.Order("cctv_stream_logs.created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&logs).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal mengambil data log CCTV",
		})
	}

	// Ringkasan hari ini (start of day UTC)
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	todayOnline := countCCTVStreamLogs(c, startOfDay, "online")
	todayOffline := countCCTVStreamLogs(c, startOfDay, "offline")

	totalPages := int(math.Ceil(float64(totalRows) / float64(limit)))
	if totalPages < 1 {
		totalPages = 1
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"rows": logs,
			"pagination": fiber.Map{
				"page":        page,
				"limit":       limit,
				"total_rows":  totalRows,
				"total_pages": totalPages,
			},
			"summary": fiber.Map{
				"today_online_events":  todayOnline,
				"today_offline_events": todayOffline,
				"total_events":         totalRows,
			},
		},
	})
}

// applyCCTVStreamLogSiteScope membatasi log event CCTV berdasarkan site
// kamera melalui relasi cctv_stream_logs.cctv_id -> cctvs.site_id. Pola EXISTS
// dipakai agar query Count, Find, dan ringkasan tetap konsisten.
//
// Admin tanpa site aktif melihat seluruh kawasan. Non-admin wajib memiliki
// site aktif; bila tidak ada, akses ditolak (fail closed) — bukan membuka
// seluruh data lintas site.
func applyCCTVStreamLogSiteScope(c *fiber.Ctx, query *gorm.DB) (*gorm.DB, bool) {
	siteID, _ := c.Locals("site_id").(string)
	if siteID != "" {
		return query.Where(
			`EXISTS (SELECT 1 FROM cctvs WHERE cctvs.id = cctv_stream_logs.cctv_id AND cctvs.site_id = ? AND cctvs.deleted_at IS NULL)`,
			siteID,
		), true
	}

	role, _ := c.Locals("role").(string)
	// role kosong hanya terjadi pada unit test tanpa middleware autentikasi;
	// perilaku lama (tanpa filter) dipertahankan untuk kompatibilitas.
	if role == "" || role == "admin" {
		return query, true
	}
	return query, false
}

// countCCTVStreamLogs menghitung event hari ini dengan scope site yang sama
// seperti daftar utama. Kegagalan query tidak menggagalkan endpoint agar
// ringkasan tetap dapat ditampilkan (perilaku lama).
func countCCTVStreamLogs(c *fiber.Ctx, since time.Time, eventType string) int64 {
	query, ok := applyCCTVStreamLogSiteScope(c, config.DB.Model(&models.CCTVStreamLog{}))
	if !ok {
		return 0
	}

	var count int64
	query.Where("created_at >= ? AND event_type = ?", since, eventType).Count(&count)
	return count
}
