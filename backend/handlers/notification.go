package handlers

import (
	"sigap-backend/config"
	"sigap-backend/models"
	"sigap-backend/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// GetNotifications mengembalikan daftar notifikasi milik user terautentikasi
// (Fase 5.1). Setiap user hanya melihat notifikasinya sendiri — scope ditentukan
// dari klaim JWT, bukan parameter URL.
// @Summary      Mendapatkan Daftar Notifikasi
// @Description  Mengembalikan daftar notifikasi in-app milik user yang sedang login, terpaginasi. Mendukung filter unread, pengurutan, dan paginasi.
// @Tags         Notifikasi
// @Produce      json
// @Security     BearerAuth
// @Param        page query int false "Nomor Halaman (default: 1)"
// @Param        limit query int false "Jumlah Data per Halaman (default: 10)"
// @Param        sort_by query string false "Kolom Pengurutan (created_at, title, type)"
// @Param        order query string false "Arah Pengurutan (asc, desc)"
// @Param        unread query boolean false "Filter hanya notifikasi belum dibaca (true)"
// @Success      200 {object} map[string]interface{} "Sukses mendapatkan daftar notifikasi"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid"
// @Failure      500 {object} map[string]interface{} "Gagal mengambil data"
// @Router       /notifications [get]
func GetNotifications(c *fiber.Ctx) error {
	userIDStr, ok := c.Locals("user_id").(string)
	if !ok || userIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Sesi tidak valid",
		})
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Sesi tidak valid",
		})
	}

	var notifications []models.Notification
	var totalRows int64

	allowedSorts := []string{"created_at", "title", "type"}
	limit, page, sort := utils.GetPaginationParams(c, allowedSorts)
	offset := (page - 1) * limit

	query := config.DB.Model(&models.Notification{}).Where("user_id = ?", userID)

	// Filter notifikasi yang belum dibaca saja
	if c.Query("unread") == "true" {
		query = query.Where("is_read = ?", false)
	}

	if err := query.Count(&totalRows).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghitung total notifikasi",
		})
	}

	if err := query.Order(sort).Limit(limit).Offset(offset).Find(&notifications).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal mengambil data notifikasi",
		})
	}

	pagination := utils.CalculatePagination(totalRows, page, limit)

	// Konvensi respons list SIGAP: data berisi objek { rows, pagination }
	// (sama dengan /cctv, /maintenance-logs, /users, /field-maintenance).
	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Berhasil mengambil data notifikasi",
		"data": fiber.Map{
			"rows":       notifications,
			"pagination": pagination,
		},
	})
}

// GetUnreadNotificationCount mengembalikan jumlah notifikasi belum dibaca milik
// user terautentikasi — dipakai badge pada NotificationBell.
// @Summary      Jumlah Notifikasi Belum Dibaca
// @Description  Mengembalikan jumlah notifikasi in-app yang belum dibaca oleh user yang sedang login.
// @Tags         Notifikasi
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "Sukses mendapatkan jumlah notifikasi belum dibaca"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid"
// @Failure      500 {object} map[string]interface{} "Gagal mengambil data"
// @Router       /notifications/unread-count [get]
func GetUnreadNotificationCount(c *fiber.Ctx) error {
	userIDStr, ok := c.Locals("user_id").(string)
	if !ok || userIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Sesi tidak valid",
		})
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Sesi tidak valid",
		})
	}

	var count int64
	if err := config.DB.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Count(&count).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghitung notifikasi belum dibaca",
		})
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Berhasil mengambil jumlah notifikasi belum dibaca",
		"data": fiber.Map{
			"unread_count": count,
		},
	})
}

// MarkNotificationRead menandai satu notifikasi sebagai sudah dibaca. Hanya
// pemilik notifikasi yang boleh mengubahnya (ownership check → 403/404).
// @Summary      Tandai Notifikasi Dibaca
// @Description  Menandai satu notifikasi milik user yang sedang login sebagai sudah dibaca.
// @Tags         Notifikasi
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Notifikasi UUID"
// @Success      200 {object} map[string]interface{} "Notifikasi berhasil ditandai dibaca"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: bukan pemilik notifikasi"
// @Failure      404 {object} map[string]interface{} "Notifikasi tidak ditemukan"
// @Failure      500 {object} map[string]interface{} "Gagal memperbarui notifikasi"
// @Router       /notifications/{id}/read [put]
func MarkNotificationRead(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID notifikasi tidak valid",
		})
	}

	userIDStr, ok := c.Locals("user_id").(string)
	if !ok || userIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Sesi tidak valid",
		})
	}

	var notif models.Notification
	if err := config.DB.First(&notif, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Notifikasi tidak ditemukan",
		})
	}

	// Ownership check: user hanya boleh menandai notifikasinya sendiri
	if notif.UserID.String() != userIDStr {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "error",
			"message": "Akses ditolak: Anda hanya dapat mengubah notifikasi milik sendiri",
		})
	}

	if !notif.IsRead {
		if err := config.DB.Model(&notif).Update("is_read", true).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "Gagal menandai notifikasi sebagai dibaca",
			})
		}
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Notifikasi berhasil ditandai sebagai dibaca",
		"data":    notif,
	})
}

// MarkAllNotificationsRead menandai seluruh notifikasi milik user terautentikasi
// sebagai sudah dibaca sekaligus (aksi tombol pada dropdown bell).
// @Summary      Tandai Semua Notifikasi Dibaca
// @Description  Menandai seluruh notifikasi milik user yang sedang login sebagai sudah dibaca.
// @Tags         Notifikasi
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "Seluruh notifikasi berhasil ditandai dibaca"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid"
// @Failure      500 {object} map[string]interface{} "Gagal memperbarui notifikasi"
// @Router       /notifications/read-all [put]
func MarkAllNotificationsRead(c *fiber.Ctx) error {
	userIDStr, ok := c.Locals("user_id").(string)
	if !ok || userIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Sesi tidak valid",
		})
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Sesi tidak valid",
		})
	}

	if err := config.DB.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Update("is_read", true).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menandai seluruh notifikasi sebagai dibaca",
		})
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Seluruh notifikasi berhasil ditandai sebagai dibaca",
	})
}
