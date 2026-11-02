package handlers

import (
	"strings"
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
	// IDEMPOTENSI: lewati bila sudah ada log dari jadwal ini pada tanggal yang
	// sama. Unscoped digunakan agar log yang di-soft-delete tetap dihormati dan
	// tidak membangkitkan tugas duplikat (M09).
	var existing models.MaintenanceLog
	err := db.Unscoped().Where("maintenance_schedule_id = ? AND scheduled_date >= ? AND scheduled_date < ?",
		schedule.ID, startOfDay, endOfDay).First(&existing).Error
	if err == nil {
		// Log untuk hari ini sudah ada — jangan buat duplikat.
		utils.Log.Infof("⏰ Jadwal %s: log hari ini sudah ada, lewati (idempotent)", schedule.ID.String())
		// PEMULIHAN (Recovery): Bila log hari ini sudah ada tetapi jadwal belum
		// sempat digeser ke masa depan (misal kegagalan/crash antara pembuatan log
		// dan pembaruan jadwal pada siklus sebelumnya), pulihkan NextRunAt sekarang.
		if existing.ScheduledDate != nil && !schedule.NextRunAt.After(*existing.ScheduledDate) && schedule.NextRunAt.Before(now) {
			nextRun := calculateNextRun(schedule.Frequency, schedule.NextRunAt, now)
			schedule.NextRunAt = nextRun
			schedule.LastGeneratedAt = &existing.CreatedAt
			if saveErr := db.Save(schedule).Error; saveErr != nil {
				utils.Log.Errorf("⚠️ Gagal memulihkan jadwal %s: %v", schedule.ID.String(), saveErr)
			} else {
				utils.Log.Infof("🔄 Jadwal %s: berhasil dipulihkan (next_run_at: %s)",
					schedule.ID.String(), nextRun.Format(time.RFC3339))
			}
		}
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
	var assignedAt *time.Time
	if schedule.TechnicianID != nil {
		status = "ongoing"
		assignedAt = &now
	}

	logEntry := models.MaintenanceLog{
		CCTVID:                schedule.CCTVID,
		TechnicianID:          schedule.TechnicianID,
		AssignedAt:            assignedAt,
		ActionTaken:           schedule.ActionTaken,
		Status:                status,
		ScheduledDate:         &now,
		MaintenanceScheduleID: &schedule.ID,
	}
	if err := db.Create(&logEntry).Error; err != nil {
		// Bila dua worker memproses jadwal bersamaan dan keduanya lolos pengecekan awal,
		// unique constraint uq_maintenance_logs_schedule_day pada database menolak insert kedua.
		errLower := strings.ToLower(err.Error())
		if strings.Contains(errLower, "unique") || strings.Contains(errLower, "duplicate") || strings.Contains(errLower, "23505") {
			utils.Log.Infof("⏰ Jadwal %s: log hari ini sudah dibuat oleh worker lain (unique constraint), lewati secara aman", schedule.ID.String())
			return
		}
		utils.Log.Errorf("⚠️ Gagal membuat log otomatis untuk jadwal %s: %v", schedule.ID.String(), err)
		return
	}

	// Geser jadwal berikutnya sesuai frekuensi & catat waktu generate terakhir.
	// Gunakan waktu rencana (schedule.NextRunAt) sebagai anchor, bukan waktu aktual,
	// agar keterlambatan worker tidak menimbulkan drift kumulatif.
	nextRun := calculateNextRun(schedule.Frequency, schedule.NextRunAt, now)
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
	broadcastMaintenanceLogEvent(db, "MAINTENANCE_CREATED", logEntry)

	// Bila jadwal langsung menugaskan teknisi, tulis notifikasi secara sinkron.
	// Helper ini juga memicu WebSocket dan Web Push sehingga jalur in-app dan
	// push tidak lagi berbeda perilaku.
	if schedule.TechnicianID != nil {
		if err := utils.NotifyUserSync(
			db,
			*schedule.TechnicianID,
			models.NotificationTypeAssigned,
			"Tugas baru dari jadwal",
			"Anda mendapat tugas pemeliharaan terjadwal otomatis: "+schedule.ActionTaken+".",
			"maintenance-log",
			&logEntry.ID,
		); err != nil {
			utils.Log.Errorf("⚠️ Gagal mengirim notifikasi jadwal %s: %v", schedule.ID.String(), err)
		}
	}

	utils.Log.Infof("✅ Generator jadwal %s: log pemeliharaan dibuat (status: %s, next_run_at: %s)",
		schedule.ID.String(), status, nextRun.Format(time.RFC3339))
}

// calculateNextRun menghitung waktu eksekusi berikutnya berdasarkan anchor waktu
// rencana sebelumnya (anchor/planned time), bukan waktu eksekusi aktual (now),
// sehingga keterlambatan worker/scheduler tidak menggeser jam maupun interval jadwal.
//
// Kebijakan Catch-up (setelah downtime):
// Bila server sempat mati atau worker tertunda lebih lama dari satu interval frekuensi
// sehingga target berikutnya masih berada di masa lalu (<= now), fungsi ini akan
// memajukan jadwal langkah demi langkah sesuai frekuensinya hingga jadwal berikutnya
// berada di masa depan (> now), sambil tetap mempertahankan jam dan menit anchor aslinya.
// Hal ini mencegah jadwal tertahan di masa lalu atau menghasilkan rentetan tugas duplikat.
func calculateNextRun(freq string, anchor time.Time, now time.Time) time.Time {
	if anchor.IsZero() {
		anchor = now
	}
	next := nextRunForFrequency(freq, anchor)
	// Jika anchor sudah sangat lampau (misal server down lama),
	// majukan terus dengan interval jadwalnya sampai berada di masa depan (> now).
	for !next.After(now) {
		next = nextRunForFrequency(freq, next)
	}
	return next
}
