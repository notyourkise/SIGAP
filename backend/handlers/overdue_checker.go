package handlers

import (
	"fmt"
	"time"

	"sigap-backend/models"
	"sigap-backend/utils"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// overdueCheckerInterval adalah jeda pemeriksaan tugas yang melewati jadwal.
const overdueCheckerInterval = 1 * time.Hour

// Level eskalasi overdue bertingkat (Task 13.11).
const (
	OverdueLevelNone     = 0 // Belum / bukan overdue
	OverdueLevelWarning  = 1 // >24 jam
	OverdueLevelHigh     = 2 // >48 jam
	OverdueLevelCritical = 3 // >72 jam (kritis)
)

// Event WebSocket untuk eskalasi overdue tingkat lanjut.
const (
	EventOverdueHigh     = "OVERDUE_HIGH"
	EventOverdueCritical = "OVERDUE_CRITICAL"
)

// RunOverdueChecker menjalankan eskalasi overdue sebagai goroutine. Pemeriksaan
// dilakukan sekali saat startup agar tugas yang terlewat ketika server mati
// segera dieskalasikan, lalu diulang setiap satu jam.
func RunOverdueChecker(db *gorm.DB) {
	if db == nil {
		return
	}

	ProcessOverdueMaintenance(db)

	ticker := time.NewTicker(overdueCheckerInterval)
	defer ticker.Stop()
	for range ticker.C {
		ProcessOverdueMaintenance(db)
	}
}

// ProcessOverdueMaintenance memeriksa tugas overdue menggunakan waktu sistem.
func ProcessOverdueMaintenance(db *gorm.DB) {
	ProcessOverdueMaintenanceAt(db, time.Now())
}

// ProcessOverdueMaintenanceAt adalah varian deterministik untuk pengujian dan
// untuk menjalankan satu siklus checker secara manual.
func ProcessOverdueMaintenanceAt(db *gorm.DB, now time.Time) {
	if db == nil {
		return
	}

	var logs []models.MaintenanceLog
	if err := db.Preload("CCTV").
		Where("status IN ? AND scheduled_date IS NOT NULL AND scheduled_date < ?", []string{"scheduled", "ongoing"}, now).
		Order("scheduled_date asc").
		Find(&logs).Error; err != nil {
		utils.Log.Errorf("⚠️ Gagal memuat tugas overdue: %v", err)
		return
	}

	for i := range logs {
		// Filter SQL sudah membatasi status, tetapi guard ini menjaga agar
		// perubahan data di antara query dan proses tidak mengeskalasi tugas
		// yang sudah selesai atau gagal.
		if logs[i].Status != "scheduled" && logs[i].Status != "ongoing" {
			continue
		}
		processOverdueMaintenanceLog(db, &logs[i], now)
	}
}

// overdueLevelFor menghitung level eskalasi berdasarkan jam keterlambatan.
func overdueLevelFor(hoursSinceScheduled float64) int {
	switch {
	case hoursSinceScheduled > 72:
		return OverdueLevelCritical
	case hoursSinceScheduled > 48:
		return OverdueLevelHigh
	case hoursSinceScheduled > 24:
		return OverdueLevelWarning
	default:
		return OverdueLevelNone
	}
}

// processOverdueMaintenanceLog mengirimkan eskalasi bertingkat ke admin dan, bila
// ada, teknisi yang sedang mengerjakan tugas. Eskalasi hanya dikirim saat level
// naik (0→1→2→3) agar tidak spam.
func processOverdueMaintenanceLog(db *gorm.DB, logEntry *models.MaintenanceLog, now time.Time) {
	if logEntry.ScheduledDate == nil {
		return
	}

	hoursSinceScheduled := now.Sub(*logEntry.ScheduledDate).Hours()
	targetLevel := overdueLevelFor(hoursSinceScheduled)
	if targetLevel <= logEntry.OverdueLevel {
		return
	}

	if err := sendEscalationNotification(db, logEntry, targetLevel, now); err != nil {
		utils.Log.Errorf("⚠️ Gagal mengirim eskalasi level %d untuk log %s: %v", targetLevel, logEntry.ID.String(), err)
		return
	}

	result := db.Model(&models.MaintenanceLog{}).
		Where("id = ?", logEntry.ID).
		Update("overdue_level", targetLevel)
	if result.Error != nil {
		utils.Log.Errorf("⚠️ Gagal memperbarui overdue_level log %s: %v", logEntry.ID.String(), result.Error)
	}
}

// sendEscalationNotification mengirim notifikasi/email sesuai level eskalasi.
func sendEscalationNotification(db *gorm.DB, logEntry *models.MaintenanceLog, level int, now time.Time) error {
	const referenceType = "maintenance-log"
	scheduledDate := logEntry.ScheduledDate.Format("2006-01-02 15:04:05")

	title := "Tugas pemeliharaan terlambat"
	if level >= OverdueLevelHigh {
		title = "Perhatian Tinggi: Tugas pemeliharaan terlambat"
	}
	if level >= OverdueLevelCritical {
		title = "KRITIS: Tugas pemeliharaan sangat terlambat"
	}

	adminMessage := fmt.Sprintf(
		"Tugas pemeliharaan %s (ID: %s) sudah melewati jadwal %s (level %d)",
		logEntry.CCTV.Nama,
		logEntry.CCTV.KodeCCTV,
		scheduledDate,
		level,
	)

	// Level kritis memakai tipe notifikasi khusus agar frontend bisa menampilkan
	// ikon/background khusus.
	notifType := models.NotificationTypeOverdue
	if level >= OverdueLevelCritical {
		notifType = models.NotificationTypeOverdueCritical
	}

	if err := utils.NotifyRoleSync(
		db, "admin", notifType, title, adminMessage, referenceType, &logEntry.ID,
	); err != nil {
		return err
	}

	if logEntry.Status == "ongoing" && logEntry.TechnicianID != nil {
		technicianMessage := fmt.Sprintf(
			"Tugas pemeliharaan Anda untuk CCTV %s sudah melewati jadwal",
			logEntry.CCTV.Nama,
		)
		if err := utils.NotifyUserSync(
			db, *logEntry.TechnicianID, notifType, title, technicianMessage, referenceType, &logEntry.ID,
		); err != nil {
			return err
		}
	}

	// Broadcast event via WebSocket untuk eskalasi tingkat lanjut.
	if level >= OverdueLevelCritical {
		utils.BroadcastEvent(EventOverdueCritical, fiber.Map{
			"log_id": logEntry.ID.String(), "cctv_name": logEntry.CCTV.Nama, "level": level,
		})
	} else if level >= OverdueLevelHigh {
		utils.BroadcastEvent(EventOverdueHigh, fiber.Map{
			"log_id": logEntry.ID.String(), "cctv_name": logEntry.CCTV.Nama, "level": level,
		})
	}

	// Task 11.3: kirim email eskalasi overdue. Berjalan async bila SMTP lambat.
	if level >= OverdueLevelCritical {
		go SendOverdueEmail(db, logEntry)
	}

	// Task 12.3: dispatch event webhook maintenance.overdue.
	DispatchWebhook(db, models.EventWebhookMaintenanceOverdue, map[string]interface{}{
		"log_id":         logEntry.ID.String(),
		"cctv_id":        logEntry.CCTVID.String(),
		"cctv_name":      logEntry.CCTV.Nama,
		"cctv_kode":      logEntry.CCTV.KodeCCTV,
		"scheduled_date": scheduledDate,
		"overdue_level":  level,
		"timestamp":      now.Format(time.RFC3339),
	})

	// Tandai telah dinotifikasi (hanya pada level warning pertama).
	if logEntry.OverdueNotifiedAt == nil {
		if err := db.Model(&models.MaintenanceLog{}).
			Where("id = ? AND overdue_notified_at IS NULL", logEntry.ID).
			Update("overdue_notified_at", now).Error; err != nil {
			utils.Log.Warnf("⚠️ Gagal menandai overdue_notified_at log %s: %v", logEntry.ID.String(), err)
		}
	}

	return nil
}
