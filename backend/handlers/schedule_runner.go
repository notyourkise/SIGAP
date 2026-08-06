package handlers

import (
	"time"

	"sigap-backend/models"
	"sigap-backend/utils"

	"gorm.io/gorm"
)

// scheduleGeneratorInterval adalah jeda pengecekan jadwal oleh generator
// (Fase 5.4). Generator berjalan tiap 1 jam dan juga sekali saat startup.
const scheduleGeneratorInterval = 1 * time.Hour

// RunScheduleGenerator menjalankan generator jadwal pemeliharaan berulang
// sebagai goroutine. Berjalan SEKALI langsung saat dipanggil (startup) lalu
// mengulang setiap 1 jam — sehingga jadwal yang lewat saat server mati
// langsung dibuatkan log tanpa menunggu ticker pertama.
func RunScheduleGenerator(db *gorm.DB) {
	if db == nil {
		return
	}

	// Eksekusi langsung saat startup agar jadwal yang sudah lewat waktunya
	// (mis. server sempat mati beberapa hari) segera diproses.
	ProcessDueSchedules(db)

	ticker := time.NewTicker(scheduleGeneratorInterval)
	defer ticker.Stop()
	for range ticker.C {
		ProcessDueSchedules(db)
	}
}

// ProcessDueSchedules mencari semua jadwal aktif yang waktunya tiba
// (next_run_at <= sekarang) lalu membuat MaintenanceLog untuk masing-masing.
// Idempotent: jadwal yang sudah punya log pada hari yang sama di-skip.
func ProcessDueSchedules(db *gorm.DB) {
	ProcessDueSchedulesAt(db, time.Now())
}

// ProcessDueSchedulesAt adalah varian ProcessDueSchedules dengan waktu eksplisit
// agar mudah diuji secara deterministik (tanpa bergantung waktu sistem).
func ProcessDueSchedulesAt(db *gorm.DB, now time.Time) {
	if db == nil {
		return
	}

	var schedules []models.MaintenanceSchedule
	// Hanya jadwal aktif (is_active = true) yang waktunya sudah tiba.
	// Jadwal nonaktif tidak pernah masuk hasil query → otomatis di-skip.
	if err := db.Where("is_active = ? AND next_run_at <= ?", true, now).
		Find(&schedules).Error; err != nil {
		utils.Log.Errorf("⚠️ Gagal memuat jadwal pemeliharaan berulang: %v", err)
		return
	}

	for i := range schedules {
		processDueSchedule(db, &schedules[i], now)
	}
}

// processDueSchedule membuat satu MaintenanceLog untuk jadwal tertentu pada
// hari `now` (idempotent), lalu menggeser NextRunAt sesuai frekuensi dan
// mencatat LastGeneratedAt. Bila jadwal memiliki teknisi, log langsung
// berstatus "ongoing" dan teknisi menerima notifikasi; tanpa teknisi log
// dibuat berstatus "scheduled" menunggu ditugaskan admin.
func processDueSchedule(db *gorm.DB, schedule *models.MaintenanceSchedule, now time.Time) {
	// IDEMPOTENSI: lewati bila sudah ada log dari jadwal ini pada tanggal yang
	// sama. Menghindari duplikat saat generator dijalankan lebih dari sekali
	// dalam sehari (mis. restart server di hari yang sama atau ticker dobel).
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.AddDate(0, 0, 1)
	var existing models.MaintenanceLog
	err := db.Where("maintenance_schedule_id = ? AND scheduled_date >= ? AND scheduled_date < ?",
		schedule.ID, startOfDay, endOfDay).First(&existing).Error
	if err == nil {
		// Log untuk hari ini sudah ada — jangan buat duplikat.
		utils.Log.Infof("⏰ Jadwal %s: log hari ini sudah ada, lewati (idempotent)", schedule.ID.String())
		return
	}
	if err != gorm.ErrRecordNotFound {
		utils.Log.Errorf("⚠️ Gagal memeriksa idempotensi jadwal %s: %v", schedule.ID.String(), err)
		return
	}

	// Status akhir ditentukan oleh ada/tidaknya teknisi pada jadwal:
	// - tanpa teknisi → "scheduled" (menunggu ditugaskan admin)
	// - dengan teknisi → "ongoing" (langsung menjadi tugas teknisi)
	status := "scheduled"
	if schedule.TechnicianID != nil {
		status = "ongoing"
	}

	logEntry := models.MaintenanceLog{
		CCTVID:                schedule.CCTVID,
		TechnicianID:          schedule.TechnicianID,
		ActionTaken:           schedule.ActionTaken,
		Status:                status,
		ScheduledDate:         &now,
		MaintenanceScheduleID: &schedule.ID,
	}
	if err := db.Create(&logEntry).Error; err != nil {
		utils.Log.Errorf("⚠️ Gagal membuat log otomatis untuk jadwal %s: %v", schedule.ID.String(), err)
		return
	}

	// Geser jadwal berikutnya sesuai frekuensi & catat waktu generate terakhir.
	nextRun := nextRunForFrequency(schedule.Frequency, now)
	schedule.NextRunAt = nextRun
	schedule.LastGeneratedAt = &now
	if err := db.Save(schedule).Error; err != nil {
		utils.Log.Errorf("⚠️ Gagal memperbarui jadwal %s setelah generate: %v", schedule.ID.String(), err)
	}

	// Audit: log dibuat otomatis oleh sistem, bukan aksi langsung admin —
	// sehingga memakai identitas "system" (bukan LogAudit yang butuh *fiber.Ctx).
	audit := models.AuditLog{
		Username:    "system",
		UserRole:    "system",
		Action:      "CREATE_MAINTENANCE",
		Module:      "MAINTENANCE",
		Description: "Dibuat otomatis oleh jadwal " + schedule.ID.String(),
		IPAddress:   "",
		UserAgent:   "",
	}
	if err := db.Create(&audit).Error; err != nil {
		utils.Log.Errorf("⚠️ Gagal mencatat audit log jadwal %s: %v", schedule.ID.String(), err)
	}

	// Broadcast agar halaman log pemeliharaan ter-refresh real-time.
	utils.BroadcastEvent("MAINTENANCE_CREATED", logEntry)

	// Bila jadwal langsung menugaskan teknisi, kirim notifikasi in-app.
	// Penulisan dilakukan sinkron (bukan via utils.NotifyUser yang async) karena
	// generator sudah berjalan di background — tidak ada request yang diblokir.
	if schedule.TechnicianID != nil {
		notif := models.Notification{
			UserID:        *schedule.TechnicianID,
			Type:          models.NotificationTypeAssigned,
			Title:         "Tugas baru dari jadwal",
			Message:       "Anda mendapat tugas pemeliharaan terjadwal otomatis: " + schedule.ActionTaken + ".",
			ReferenceType: "maintenance-log",
			ReferenceID:   &logEntry.ID,
		}
		if err := db.Create(&notif).Error; err != nil {
			utils.Log.Errorf("⚠️ Gagal mengirim notifikasi jadwal %s: %v", schedule.ID.String(), err)
		}
		utils.BroadcastToUser(schedule.TechnicianID.String(), utils.EventNotifCreated, notif)
	}

	utils.Log.Infof("✅ Generator jadwal %s: log pemeliharaan dibuat (status: %s, next_run_at: %s)",
		schedule.ID.String(), status, nextRun.Format(time.RFC3339))
}
