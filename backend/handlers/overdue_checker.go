package handlers

import (
	"fmt"
	"time"

	"sigap-backend/models"
	"sigap-backend/utils"

	"gorm.io/gorm"
)

// overdueCheckerInterval adalah jeda pemeriksaan tugas yang melewati jadwal.
const overdueCheckerInterval = 1 * time.Hour

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
		Where("status IN ? AND scheduled_date IS NOT NULL AND scheduled_date < ? AND overdue_notified_at IS NULL", []string{"scheduled", "ongoing"}, now).
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
		if logs[i].OverdueNotifiedAt != nil {
			continue
		}
		processOverdueMaintenanceLog(db, &logs[i], now)
	}
}

// processOverdueMaintenanceLog mengirimkan eskalasi ke admin dan, bila ada,
// teknisi yang sedang mengerjakan tugas. Flag baru ditulis setelah seluruh
// notifikasi berhasil dibuat agar checker dapat mencoba kembali saat database
// sedang mengalami gangguan.
func processOverdueMaintenanceLog(db *gorm.DB, logEntry *models.MaintenanceLog, now time.Time) {
	if logEntry.ScheduledDate == nil {
		return
	}

	const overdueTitle = "Tugas pemeliharaan terlambat"
	const referenceType = "maintenance-log"
	scheduledDate := logEntry.ScheduledDate.Format("2006-01-02 15:04:05")
	adminMessage := fmt.Sprintf(
		"Tugas pemeliharaan %s (ID: %s) sudah melewati jadwal %s",
		logEntry.CCTV.Nama,
		logEntry.CCTV.KodeCCTV,
		scheduledDate,
	)

	if err := utils.NotifyRoleSync(
		db,
		"admin",
		models.NotificationTypeOverdue,
		overdueTitle,
		adminMessage,
		referenceType,
		&logEntry.ID,
	); err != nil {
		utils.Log.Errorf("⚠️ Gagal mengirim eskalasi overdue untuk log %s ke admin: %v", logEntry.ID.String(), err)
		return
	}

	if logEntry.Status == "ongoing" && logEntry.TechnicianID != nil {
		technicianMessage := fmt.Sprintf(
			"Tugas pemeliharaan Anda untuk CCTV %s sudah melewati jadwal",
			logEntry.CCTV.Nama,
		)
		if err := utils.NotifyUserSync(
			db,
			*logEntry.TechnicianID,
			models.NotificationTypeOverdue,
			overdueTitle,
			technicianMessage,
			referenceType,
			&logEntry.ID,
		); err != nil {
			utils.Log.Errorf("⚠️ Gagal mengirim eskalasi overdue untuk teknisi log %s: %v", logEntry.ID.String(), err)
			return
		}
	}

	// Task 11.3: kirim email eskalasi overdue ke admin/teknisi yang opt-in.
	// Berjalan async agar tidak memperlambat checker bila SMTP lambat.
	go SendOverdueEmail(db, logEntry)

	// Task 12.3: dispatch event webhook maintenance.overdue.
	DispatchWebhook(db, models.EventWebhookMaintenanceOverdue, map[string]interface{}{
		"log_id":         logEntry.ID.String(),
		"cctv_id":        logEntry.CCTVID.String(),
		"cctv_name":      logEntry.CCTV.Nama,
		"cctv_kode":      logEntry.CCTV.KodeCCTV,
		"scheduled_date": scheduledDate,
		"timestamp":      now.Format(time.RFC3339),
	})

	result := db.Model(&models.MaintenanceLog{}).
		Where("id = ? AND overdue_notified_at IS NULL", logEntry.ID).
		Update("overdue_notified_at", now)
	if result.Error != nil {
		utils.Log.Errorf("⚠️ Gagal menandai log %s sebagai sudah dinotifikasi overdue: %v", logEntry.ID.String(), result.Error)
		return
	}
	if result.RowsAffected == 0 {
		utils.Log.Infof("⏰ Log %s sudah ditandai overdue oleh proses lain", logEntry.ID.String())
	}
}
