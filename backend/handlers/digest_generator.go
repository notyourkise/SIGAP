package handlers

import (
	"errors"
	"fmt"
	"time"

	"sigap-backend/models"
	"sigap-backend/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	dailyDigestHour     = 6
	dailyDigestLocation = "Asia/Makassar"
)

// DailyDigestMaintenanceStats adalah hasil agregasi tugas pemeliharaan dalam
// satu jendela ringkasan.
type DailyDigestMaintenanceStats struct {
	NewTasks       int64 `gorm:"column:new_tasks"`
	CompletedTasks int64 `gorm:"column:completed_tasks"`
	FailedTasks    int64 `gorm:"column:failed_tasks"`
	OverdueTasks   int64 `gorm:"column:overdue_tasks"`
}

// DailyDigestFieldStats adalah hasil agregasi pengecekan lapangan dalam satu
// jendela ringkasan.
type DailyDigestFieldStats struct {
	FieldChecks     int64 `gorm:"column:field_checks"`
	AttentionChecks int64 `gorm:"column:attention_checks"`
}

// RunDailyDigestGenerator menunggu jadwal 06:00 WITA berikutnya lalu membuat
// satu digest setiap hari. Timer dihitung ulang setiap siklus agar waktu tetap
// mengikuti kalender WITA saat server berjalan lama.
func RunDailyDigestGenerator(db *gorm.DB) {
	if db == nil {
		return
	}

	for {
		now := time.Now().In(dailyDigestWitaLocation())
		nextRun := nextDailyDigestRun(now)
		timer := time.NewTimer(time.Until(nextRun))
		<-timer.C

		if _, _, err := GenerateDailyDigestAt(db, time.Now().In(dailyDigestWitaLocation())); err != nil {
			utils.Log.Errorf("Gagal membuat ringkasan operasional harian: %v", err)
		}
	}
}

// GenerateDailyDigest membuat digest untuk tanggal WITA pada waktu yang
// diberikan. Fungsi ini dipisahkan dari goroutine agar dapat diuji deterministik
// dan dapat dipanggil satu siklusnya oleh tooling operasional.
// Return kedua bernilai true hanya bila record baru berhasil dibuat.
func GenerateDailyDigestAt(db *gorm.DB, now time.Time) (*models.DailyDigest, bool, error) {
	if db == nil {
		return nil, false, errors.New("database digest nil")
	}

	localNow := now.In(dailyDigestWitaLocation())
	digestDate := dateOnlyInLocation(localNow)

	// IDEMPOTENSI: satu tanggal hanya boleh memiliki satu digest. Unique index
	// pada model tetap menjadi perlindungan terakhir terhadap race condition.
	var existing models.DailyDigest
	if err := db.Where("digest_date = ?", digestDate).First(&existing).Error; err == nil {
		return &existing, false, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}

	periodStart := localNow.Add(-24 * time.Hour)
	periodEnd := localNow

	maintenanceStats, err := aggregateDailyDigestMaintenance(db, periodStart, periodEnd)
	if err != nil {
		return nil, false, err
	}
	fieldStats, err := aggregateDailyDigestFieldChecks(db, periodStart, periodEnd)
	if err != nil {
		return nil, false, err
	}
	statusChanges, err := countDailyDigestStatusChanges(db, periodStart, periodEnd)
	if err != nil {
		return nil, false, err
	}

	digest := models.DailyDigest{
		DigestDate:      digestDate,
		NewTasks:        int(maintenanceStats.NewTasks),
		CompletedTasks:  int(maintenanceStats.CompletedTasks),
		FailedTasks:     int(maintenanceStats.FailedTasks),
		OverdueTasks:    int(maintenanceStats.OverdueTasks),
		FieldChecks:     int(fieldStats.FieldChecks),
		AttentionChecks: int(fieldStats.AttentionChecks),
		StatusChanges:   int(statusChanges),
		Summary:         buildDailyDigestSummary(digestDate, maintenanceStats, fieldStats, statusChanges),
	}

	if err := db.Create(&digest).Error; err != nil {
		// Server restart atau dua worker yang berjalan bersamaan dapat masuk ke
		// blok ini setelah pemeriksaan awal. Bila record sudah dibuat proses lain,
		// perlakukan sebagai skip idempoten, bukan sebagai kegagalan baru.
		var concurrent models.DailyDigest
		if findErr := db.Where("digest_date = ?", digestDate).First(&concurrent).Error; findErr == nil {
			return &concurrent, false, nil
		}
		return nil, false, err
	}

	// Notifikasi dibuat sinkron agar digest yang baru dibuat langsung terlihat
	// oleh seluruh admin melalui bell/WebSocket sebelum siklus selesai.
	if err := notifyDailyDigestAdmins(db, &digest); err != nil {
		utils.Log.Errorf("Gagal mengirim notifikasi digest %s: %v", digestDate.Format("2006-01-02"), err)
	}

	return &digest, true, nil
}

func aggregateDailyDigestMaintenance(db *gorm.DB, start, end time.Time) (DailyDigestMaintenanceStats, error) {
	var stats DailyDigestMaintenanceStats
	query := `
		SELECT
			COUNT(*) FILTER (WHERE created_at >= ? AND created_at < ?) AS new_tasks,
			COUNT(*) FILTER (
				WHERE status = 'completed'
				  AND COALESCE(completed_at, updated_at) >= ?
				  AND COALESCE(completed_at, updated_at) < ?
			) AS completed_tasks,
			COUNT(*) FILTER (
				WHERE status = 'failed'
				  AND COALESCE(completed_at, updated_at) >= ?
				  AND COALESCE(completed_at, updated_at) < ?
			) AS failed_tasks,
			COUNT(*) FILTER (
				WHERE status IN ('scheduled', 'ongoing')
				  AND scheduled_date IS NOT NULL
				  AND scheduled_date < ?
			) AS overdue_tasks
		FROM maintenance_logs
		WHERE deleted_at IS NULL`

	if err := db.Raw(query, start, end, start, end, start, end, end).Scan(&stats).Error; err != nil {
		return stats, err
	}
	return stats, nil
}

func aggregateDailyDigestFieldChecks(db *gorm.DB, start, end time.Time) (DailyDigestFieldStats, error) {
	var stats DailyDigestFieldStats
	query := `
		SELECT
			COUNT(*) AS field_checks,
			COUNT(*) FILTER (WHERE overall_status = 'attention') AS attention_checks
		FROM field_maintenances
		WHERE created_at >= ? AND created_at < ? AND deleted_at IS NULL`

	if err := db.Raw(query, start, end).Scan(&stats).Error; err != nil {
		return stats, err
	}
	return stats, nil
}

func countDailyDigestStatusChanges(db *gorm.DB, start, end time.Time) (int64, error) {
	var count int64
	err := db.Model(&models.AuditLog{}).
		Where("module = ? AND action = ? AND created_at >= ? AND created_at < ?", "CCTV", "UPDATE_CCTV", start, end).
		Count(&count).Error
	return count, err
}

func buildDailyDigestSummary(date time.Time, maintenance DailyDigestMaintenanceStats, field DailyDigestFieldStats, statusChanges int64) string {
	return fmt.Sprintf(
		"Ringkasan %s: %d tugas baru, %d selesai, %d gagal, %d overdue, %d pengecekan lapangan (%d attention), dan %d perubahan status CCTV.",
		date.Format("2006-01-02"),
		maintenance.NewTasks,
		maintenance.CompletedTasks,
		maintenance.FailedTasks,
		maintenance.OverdueTasks,
		field.FieldChecks,
		field.AttentionChecks,
		statusChanges,
	)
}

func notifyDailyDigestAdmins(db *gorm.DB, digest *models.DailyDigest) error {
	message := fmt.Sprintf(
		"Ringkasan operasional tanggal %s berisi %d tugas baru, %d pengecekan lapangan, dan %d perubahan status CCTV.",
		digest.DigestDate.Format("2006-01-02"),
		digest.NewTasks,
		digest.FieldChecks,
		digest.StatusChanges,
	)
	return utils.NotifyRoleSync(
		db,
		"admin",
		models.NotificationTypeDailyDigest,
		"Ringkasan operasional "+digest.DigestDate.Format("2006-01-02")+" tersedia",
		message,
		"daily-digest",
		uuidPtr(digest.ID),
	)
}

func uuidPtr(id uuid.UUID) *uuid.UUID {
	return &id
}

func nextDailyDigestRun(now time.Time) time.Time {
	localNow := now.In(dailyDigestWitaLocation())
	next := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), dailyDigestHour, 0, 0, 0, dailyDigestWitaLocation())
	if localNow.After(next) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}

func dateOnlyInLocation(value time.Time) time.Time {
	local := value.In(dailyDigestWitaLocation())
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, dailyDigestWitaLocation())
}

func dailyDigestWitaLocation() *time.Location {
	location, err := time.LoadLocation(dailyDigestLocation)
	if err == nil {
		return location
	}
	return time.FixedZone("WITA", 8*60*60)
}
