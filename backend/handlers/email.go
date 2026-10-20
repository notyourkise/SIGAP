package handlers

import (
	"fmt"
	"time"

	"sigap-backend/config"
	"sigap-backend/models"
	"sigap-backend/utils"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// EmailPreferencesRequest adalah body untuk memperbarui preferensi notifikasi
// email pengguna yang sedang login. Mendukung preferensi granular per-tipe
// (Task 13.7): ringkasan harian, eskalasi overdue, stok suku cadang rendah.
type EmailPreferencesRequest struct {
	EmailNotifications *bool `json:"email_notifications"`
	EmailDailyDigest   *bool `json:"email_daily_digest"`
	EmailOverdue       *bool `json:"email_overdue"`
	EmailStockLow      *bool `json:"email_stock_low"`
}

// UpdateEmailPreferences memperbarui preferensi notifikasi email pengguna.
// @Summary      Perbarui Preferensi Email
// @Description  Mengaktifkan/menonaktifkan notifikasi email (ringkasan harian, eskalasi overdue, stok rendah) untuk pengguna yang sedang login.
// @Tags         Profil
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body EmailPreferencesRequest true "Preferensi Email"
// @Success      200 {object} map[string]interface{} "Preferensi email berhasil diperbarui"
// @Failure      400 {object} map[string]interface{} "Format input tidak valid / validasi gagal"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Pengguna tidak ditemukan"
// @Failure      500 {object} map[string]interface{} "Gagal menyimpan preferensi"
// @Router       /profile/email-preferences [put]
func UpdateEmailPreferences(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Identitas pengguna tidak ditemukan dalam token",
		})
	}

	var req EmailPreferencesRequest
	if err := c.BodyParser(&req); err != nil {
		return sendBodyParseError(c, &req, err)
	}

	// Harus ada minimal satu field yang dikirim.
	if req.EmailNotifications == nil && req.EmailDailyDigest == nil && req.EmailOverdue == nil && req.EmailStockLow == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Setidaknya satu preferensi email harus diisi",
		})
	}

	var user models.User
	if err := config.DB.First(&user, "id = ?", userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Pengguna tidak ditemukan",
		})
	}

	// Master toggle: bila dikirim, set email_notifications. Bila dimatikan,
	// semua email diblokir.
	if req.EmailNotifications != nil {
		user.EmailNotifications = *req.EmailNotifications
		if !*req.EmailNotifications {
			user.EmailDailyDigest = false
			user.EmailOverdue = false
			user.EmailStockLow = false
		}
	}

	// Preferensi granular per-tipe.
	if req.EmailDailyDigest != nil {
		user.EmailDailyDigest = *req.EmailDailyDigest
	}
	if req.EmailOverdue != nil {
		user.EmailOverdue = *req.EmailOverdue
	}
	if req.EmailStockLow != nil {
		user.EmailStockLow = *req.EmailStockLow
	}

	// Sinkronkan master toggle: bila setidaknya satu sub-preferensi aktif,
	// email_notifications menjadi true; bila semua nonaktif, menjadi false.
	user.EmailNotifications = user.EmailDailyDigest || user.EmailOverdue || user.EmailStockLow

	if err := config.DB.Save(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menyimpan preferensi email",
		})
	}

	utils.LogAudit(config.DB, c, "UPDATE_EMAIL_PREFERENCES", "USER",
		fmt.Sprintf("Pengguna %s mengubah preferensi email (master=%v, digest=%v, overdue=%v, stock=%v)",
			user.Username, user.EmailNotifications, user.EmailDailyDigest, user.EmailOverdue, user.EmailStockLow),
		&user.ID, user.Username, user.Role)

	return c.JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"email_notifications": user.EmailNotifications,
			"email_daily_digest":  user.EmailDailyDigest,
			"email_overdue":       user.EmailOverdue,
			"email_stock_low":     user.EmailStockLow,
		},
	})
}

// SendDailySummaryEmails mengirim email ringkasan operasional harian ke seluruh
// admin yang mengaktifkan notifikasi email. Memakai data digest terbaru.
func SendDailySummaryEmails(db *gorm.DB, digest *models.DailyDigest) error {
	if db == nil || digest == nil || !utils.EmailEnabled() {
		return nil
	}

	var admins []models.User
	if err := db.Where("role = ? AND email_notifications = ? AND email_daily_digest = ? AND deleted_at IS NULL",
		"admin", true, true).Find(&admins).Error; err != nil {
		return err
	}

	for _, admin := range admins {
		if admin.Email == "" {
			continue
		}
		subject := "Ringkasan Operasional Harian SIGAP " + digest.DigestDate.Format("2006-01-02")
		body, err := utils.RenderEmailTemplate("daily_summary.html", map[string]interface{}{
			"Nama":               admin.FullName,
			"Tanggal":            digest.DigestDate.Format("2006-01-02"),
			"TugasBaru":          digest.NewTasks,
			"TugasSelesai":       digest.CompletedTasks,
			"TugasGagal":         digest.FailedTasks,
			"TugasOverdue":       digest.OverdueTasks,
			"PengecekanLapangan": digest.FieldChecks,
			"PerubahanStatus":    digest.StatusChanges,
			"Summary":            digest.Summary,
		})
		if err != nil {
			utils.Log.Errorf("Gagal merender template email ringkasan untuk %s: %v", admin.Email, err)
			continue
		}
		if err := utils.SendEmail(admin.Email, subject, body); err != nil {
			utils.Log.Errorf("Gagal mengirim email ringkasan harian ke %s: %v", admin.Email, err)
		}
	}
	return nil
}

// SendOverdueEmail mengirim email eskalasi overdue ke admin (dan teknisi bila
// ada) untuk satu maintenance log yang melewati jadwal.
func SendOverdueEmail(db *gorm.DB, logEntry *models.MaintenanceLog) {
	if db == nil || logEntry == nil || !utils.EmailEnabled() {
		return
	}

	subject := "Eskalasi: Tugas Pemeliharaan Terlambat " + logEntry.CCTV.Nama
	scheduledDate := ""
	if logEntry.ScheduledDate != nil {
		scheduledDate = logEntry.ScheduledDate.Format("2006-01-02 15:04:05")
	}

	sendToUser := func(u models.User, status string) {
		if u.Email == "" || !u.EmailNotifications || !u.EmailOverdue {
			return
		}
		body, err := utils.RenderEmailTemplate("overdue_escalation.html", map[string]interface{}{
			"Nama":         u.FullName,
			"CCTVNama":     logEntry.CCTV.Nama,
			"CCTVCCTVKode": logEntry.CCTV.KodeCCTV,
			"Tindakan":     logEntry.ActionTaken,
			"Jadwal":       scheduledDate,
			"Status":       status,
		})
		if err != nil {
			utils.Log.Errorf("Gagal merender template email overdue untuk %s: %v", u.Email, err)
			return
		}
		if err := utils.SendEmail(u.Email, subject, body); err != nil {
			utils.Log.Errorf("Gagal mengirim email overdue ke %s: %v", u.Email, err)
		}
	}

	// Kirim ke seluruh admin yang opt-in (overdue).
	var admins []models.User
	if err := db.Where("role = ? AND email_notifications = ? AND email_overdue = ? AND deleted_at IS NULL",
		"admin", true, true).Find(&admins).Error; err == nil {
		for _, a := range admins {
			sendToUser(a, "Terlambat")
		}
	}

	// Kirim ke teknisi yang sedang mengerjakan tugas bila ada.
	if logEntry.Status == "ongoing" && logEntry.TechnicianID != nil {
		var tech models.User
		if err := db.First(&tech, "id = ?", *logEntry.TechnicianID).Error; err == nil {
			sendToUser(tech, "Dalam proses")
		}
	}
}

// SendStockLowEmail mengirim email peringatan stok rendah ke seluruh admin yang
// opt-in ketika stok suku cadang jatuh di bawah minimum_order.
func SendStockLowEmail(db *gorm.DB, sparePart *models.SparePart, newStock int) {
	if db == nil || sparePart == nil || !utils.EmailEnabled() {
		return
	}

	subject := "Peringatan Stok Rendah: " + sparePart.Nama
	var admins []models.User
	if err := db.Where("role = ? AND email_notifications = ? AND email_stock_low = ? AND deleted_at IS NULL",
		"admin", true, true).Find(&admins).Error; err != nil {
		return
	}

	for _, admin := range admins {
		if admin.Email == "" {
			continue
		}
		body, err := utils.RenderEmailTemplate("stock_low.html", map[string]interface{}{
			"Nama":           admin.FullName,
			"NamaSukuCadang": sparePart.Nama,
			"SKU":            sparePart.SKU,
			"Kategori":       sparePart.Kategori,
			"Stok":           newStock,
			"Satuan":         sparePart.Satuan,
			"MinimumOrder":   sparePart.MinimumOrder,
		})
		if err != nil {
			utils.Log.Errorf("Gagal merender template email stok rendah untuk %s: %v", admin.Email, err)
			continue
		}
		if err := utils.SendEmail(admin.Email, subject, body); err != nil {
			utils.Log.Errorf("Gagal mengirim email stok rendah ke %s: %v", admin.Email, err)
		}
	}
}

// SendCCTVStreamDownEmail mengirim email peringatan CCTV offline ke seluruh admin
// yang mengaktifkan notifikasi email ketika stream CCTV terdeteksi down lebih
// dari ambang waktu (Task 11.3). Dihubungkan dari stream health monitor.
func SendCCTVStreamDownEmail(db *gorm.DB, cctv *models.CCTV) {
	if db == nil || cctv == nil || !utils.EmailEnabled() {
		return
	}

	subject := "🚨 CCTV Offline: " + cctv.Nama
	waktu := time.Now().In(dailyDigestWitaLocation()).Format("2006-01-02 15:04:05 WITA")

	var admins []models.User
	if err := db.Where("role = ? AND email_notifications = ? AND deleted_at IS NULL", "admin", true).
		Find(&admins).Error; err != nil {
		return
	}

	for _, admin := range admins {
		if admin.Email == "" {
			continue
		}
		body, err := utils.RenderEmailTemplate("stream_down.html", map[string]interface{}{
			"Nama":         admin.FullName,
			"CCTVNama":     cctv.Nama,
			"CCTVCCTVKode": cctv.KodeCCTV,
			"Lokasi":       cctv.LokasiArea,
			"Jenis":        cctv.JenisCCTV,
			"IPKamera":     cctv.IPCamera,
			"Waktu":        waktu,
			"HealthScore":  cctv.HealthScore,
		})
		if err != nil {
			utils.Log.Errorf("Gagal merender template email CCTV offline untuk %s: %v", admin.Email, err)
			continue
		}
		if err := utils.SendEmail(admin.Email, subject, body); err != nil {
			utils.Log.Errorf("Gagal mengirim email CCTV offline ke %s: %v", admin.Email, err)
		}
	}
}
