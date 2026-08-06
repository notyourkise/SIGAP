package handlers

import (
	"database/sql"
	"math"
	"time"

	"sigap-backend/models"
	"sigap-backend/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	healthScoreLookback       = 90 * 24 * time.Hour
	healthScoreCalculatorTick = 6 * time.Hour
)

// HealthScoreBreakdown berisi metrik mentah yang dipakai dalam perhitungan.
// Field age berisi jumlah hari sejak MaintenanceLog completed terakhir; 91
// berarti CCTV belum memiliki riwayat pemeliharaan yang berhasil.
type HealthScoreBreakdown struct {
	Failed     int `json:"failed"`
	Attention  int `json:"attention"`
	Overdue    int `json:"overdue"`
	Age        int `json:"age"`
	NotOKItems int `json:"not_ok_items"`
}

// HealthScoreResult adalah hasil kalkulasi yang dipakai endpoint dan scheduler.
type HealthScoreResult struct {
	HealthScore int                  `json:"health_score"`
	Breakdown   HealthScoreBreakdown `json:"breakdown"`
	UpdatedAt   time.Time            `json:"updated_at"`
}

type healthScoreMetrics struct {
	Failed          int64
	Attention       int64
	Overdue         int64
	NotOKItems      int64
	LastCompletedAt sql.NullTime
}

// RunHealthScoreCalculator menghitung ulang seluruh skor saat startup lalu
// mengulanginya setiap enam jam agar data yang menua tetap tercermin.
func RunHealthScoreCalculator(db *gorm.DB) {
	if db == nil {
		return
	}

	CalculateAllCCTVHealthScoresAt(db, time.Now())

	ticker := time.NewTicker(healthScoreCalculatorTick)
	defer ticker.Stop()
	for range ticker.C {
		CalculateAllCCTVHealthScoresAt(db, time.Now())
	}
}

// CalculateAllCCTVHealthScores menghitung ulang skor semua CCTV menggunakan
// waktu sistem. Varian dengan waktu eksplisit dipakai test agar deterministik.
func CalculateAllCCTVHealthScores(db *gorm.DB) {
	CalculateAllCCTVHealthScoresAt(db, time.Now())
}

// CalculateAllCCTVHealthScoresAt memproses setiap CCTV secara terpisah agar
// satu data bermasalah tidak menghentikan pembaruan skor perangkat lain.
func CalculateAllCCTVHealthScoresAt(db *gorm.DB, now time.Time) {
	if db == nil {
		return
	}

	var cctvs []models.CCTV
	if err := db.Select("id").Find(&cctvs).Error; err != nil {
		utils.Log.Errorf("Gagal memuat CCTV untuk kalkulasi skor kesehatan: %v", err)
		return
	}

	for _, cctv := range cctvs {
		if _, err := recalculateCCTVHealthScoreAt(db, cctv.ID, now); err != nil {
			utils.Log.Errorf("Gagal menghitung skor kesehatan CCTV %s: %v", cctv.ID.String(), err)
		}
	}
}

// CalculateCCTVHealthScoreAt mengambil metrik lima faktor dari 90 hari terakhir
// lalu mengubahnya menjadi skor. Fungsi ini tidak menulis database sehingga
// dapat dipakai endpoint, scheduler, dan unit test secara deterministik.
func CalculateCCTVHealthScoreAt(db *gorm.DB, cctvID uuid.UUID, now time.Time) (HealthScoreResult, error) {
	metrics, err := loadHealthScoreMetrics(db, cctvID, now)
	if err != nil {
		return HealthScoreResult{}, err
	}
	return calculateHealthScore(metrics, now), nil
}

// RecalculateCCTVHealthScore menghitung dan menyimpan skor satu CCTV, kemudian
// memberitahu klien agar daftar CCTV/peta mengambil data terbaru.
func RecalculateCCTVHealthScore(db *gorm.DB, cctvID uuid.UUID) error {
	if db == nil {
		return gorm.ErrInvalidDB
	}
	_, err := recalculateCCTVHealthScoreAt(db, cctvID, time.Now())
	return err
}

func recalculateCCTVHealthScoreAt(db *gorm.DB, cctvID uuid.UUID, now time.Time) (HealthScoreResult, error) {
	result, err := CalculateCCTVHealthScoreAt(db, cctvID, now)
	if err != nil {
		return HealthScoreResult{}, err
	}

	update := db.Model(&models.CCTV{}).
		Where("id = ?", cctvID).
		Update("health_score", result.HealthScore)
	if update.Error != nil {
		return HealthScoreResult{}, update.Error
	}
	if update.RowsAffected == 0 {
		return HealthScoreResult{}, gorm.ErrRecordNotFound
	}

	utils.BroadcastEvent("CCTV_UPDATED", map[string]interface{}{
		"id":           cctvID.String(),
		"health_score": result.HealthScore,
	})
	return result, nil
}

func loadHealthScoreMetrics(db *gorm.DB, cctvID uuid.UUID, now time.Time) (healthScoreMetrics, error) {
	if db == nil {
		return healthScoreMetrics{}, gorm.ErrInvalidDB
	}

	since := now.Add(-healthScoreLookback)
	var metrics healthScoreMetrics
	err := db.Raw(`
		SELECT
			(SELECT COUNT(*) FROM maintenance_logs
			 WHERE cctv_id = ? AND status = 'failed'
			   AND COALESCE(completed_at, created_at) >= ?
			   AND COALESCE(completed_at, created_at) < ?
			   AND deleted_at IS NULL) AS failed,
			(SELECT COUNT(*) FROM field_maintenances
			 WHERE cctv_id = ? AND overall_status = 'attention'
			   AND COALESCE(completed_at, created_at) >= ?
			   AND COALESCE(completed_at, created_at) < ?
			   AND deleted_at IS NULL) AS attention,
			(SELECT COUNT(*) FROM maintenance_logs
			 WHERE cctv_id = ? AND status IN ('scheduled', 'ongoing')
			   AND scheduled_date IS NOT NULL AND scheduled_date >= ?
			   AND scheduled_date < ? AND deleted_at IS NULL) AS overdue,
			(SELECT COUNT(*) FROM field_maintenance_checks AS checks
			 JOIN field_maintenances AS fields
			   ON fields.id = checks.field_maintenance_id
			 WHERE fields.cctv_id = ? AND checks.result = 'not_ok'
			   AND COALESCE(fields.completed_at, fields.created_at) >= ?
			   AND COALESCE(fields.completed_at, fields.created_at) < ?
			   AND checks.deleted_at IS NULL AND fields.deleted_at IS NULL) AS not_ok_items,
			(SELECT MAX(completed_at) FROM maintenance_logs
			 WHERE cctv_id = ? AND status = 'completed'
			   AND completed_at IS NOT NULL AND completed_at < ?
			   AND deleted_at IS NULL) AS last_completed_at
	`, cctvID, since, now, cctvID, since, now, cctvID, since, now, cctvID, since, now, cctvID, now).
		Scan(&metrics).Error
	return metrics, err
}

func calculateHealthScore(metrics healthScoreMetrics, now time.Time) HealthScoreResult {
	ageDays := 91
	if metrics.LastCompletedAt.Valid {
		ageDays = int(now.Sub(metrics.LastCompletedAt.Time).Hours() / 24)
		if ageDays < 0 {
			ageDays = 0
		}
	}

	agePenalty := 0
	switch {
	case ageDays > 90:
		agePenalty = 15
	case ageDays > 60:
		agePenalty = 10
	case ageDays > 30:
		agePenalty = 5
	}

	// Bobot diterapkan pada total penalti tiap faktor, kemudian skor akhir
	// dibulatkan sekali agar penalti pecahan tidak hilang di tiap langkah.
	penalty := float64(metrics.Failed)*10*0.30 +
		float64(metrics.Attention)*10*0.25 +
		float64(metrics.Overdue)*10*0.20 +
		float64(agePenalty)*0.15 +
		float64((metrics.NotOKItems/3)*5)*0.10
	score := int(math.Round(100 - penalty))
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return HealthScoreResult{
		HealthScore: score,
		Breakdown: HealthScoreBreakdown{
			Failed:     int(metrics.Failed),
			Attention:  int(metrics.Attention),
			Overdue:    int(metrics.Overdue),
			Age:        ageDays,
			NotOKItems: int(metrics.NotOKItems),
		},
		UpdatedAt: now,
	}
}
