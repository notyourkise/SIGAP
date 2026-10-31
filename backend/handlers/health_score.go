package handlers

import (
	"database/sql"
	"errors"
	"math"
	"time"

	"sigap-backend/config"
	"sigap-backend/models"
	"sigap-backend/utils"

	"github.com/gofiber/fiber/v2"
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
// mengulanginya setiap enam jam agar data yang menua tetap tercermin. Sekaligus
// membersihkan histori skor kesehatan lama agar tabel tidak membengkak.
func RunHealthScoreCalculator(db *gorm.DB) {
	if db == nil {
		return
	}

	CalculateAllCCTVHealthScoresAt(db, time.Now())
	cleanupOldHealthHistories(db)

	ticker := time.NewTicker(healthScoreCalculatorTick)
	defer ticker.Stop()
	for range ticker.C {
		CalculateAllCCTVHealthScoresAt(db, time.Now())
		cleanupOldHealthHistories(db)
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

	// Ambil skor lama untuk mendeteksi perubahan (hook webhook health_score.changed).
	var cctv models.CCTV
	if err := db.Select("id", "kode_cctv", "nama", "health_score").First(&cctv, "id = ?", cctvID).Error; err != nil {
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

	// Task 13.8: rekam histori skor kesehatan. Kegagalan insert tidak membatalkan
	// proses utama — cukup log peringatan.
	if err := db.Create(&models.CCTVHealthHistory{
		CCTVID:       cctvID,
		HealthScore:  result.HealthScore,
		CalculatedAt: now,
	}).Error; err != nil {
		utils.Log.Warnf("⚠️ Gagal merekam histori skor kesehatan CCTV %s: %v", cctvID, err)
	}

	siteID := cctvSiteIDStr(cctv)
	utils.BroadcastToSite(siteID, "CCTV_UPDATED", map[string]interface{}{
		"id":           cctvID.String(),
		"health_score": result.HealthScore,
		"site_id":      siteID,
	})

	// Task 12.3: dispatch event webhook health_score.changed bila skor berubah.
	if cctv.HealthScore != result.HealthScore {
		DispatchWebhook(db, models.EventWebhookHealthScoreChanged, map[string]interface{}{
			"cctv_id":   cctvID.String(),
			"kode_cctv": cctv.KodeCCTV,
			"nama":      cctv.Nama,
			"old_score": cctv.HealthScore,
			"new_score": result.HealthScore,
			"timestamp": now.Format(time.RFC3339),
		})
	}
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

// GetCCTVHealthHistory mengembalikan histori skor kesehatan CCTV untuk rentang
// waktu tertentu (Task 13.8). Query param ?range=7d|30d|90d (default 30d).
// @Summary      Histori Skor Kesehatan CCTV
// @Description  Mengembalikan histori skor kesehatan (0-100) suatu CCTV dalam rentang waktu (7/30/90 hari).
// @Tags         Master CCTV
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "CCTV UUID"
// @Param        range query string false "Rentang waktu: 7d|30d|90d (default 30d)"
// @Success      200 {object} map[string]interface{} "Sukses mendapatkan histori skor kesehatan"
// @Failure      400 {object} map[string]interface{} "ID CCTV tidak valid"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid"
// @Router       /cctv/{id}/health-history [get]
func GetCCTVHealthHistory(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID CCTV tidak valid",
		})
	}

	if siteID, ok := c.Locals("site_id").(string); ok && siteID != "" {
		var cctv models.CCTV
		if err := config.DB.First(&cctv, "id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"status":  "error",
					"message": "Data CCTV tidak ditemukan",
				})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "Gagal mengambil data CCTV",
			})
		}
		if !cctvMatchesActiveSite(c, cctv) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"status":  "error",
				"message": "Akses ditolak: CCTV berada di site lain",
			})
		}
	}

	// Tentukan rentang hari dari query param.
	days := 30
	switch c.Query("range") {
	case "7d":
		days = 7
	case "90d":
		days = 90
	}
	cutoff := time.Now().AddDate(0, 0, -days)

	var history []models.CCTVHealthHistory
	if err := config.DB.Where("cctv_id = ? AND calculated_at > ?", id, cutoff).
		Order("calculated_at ASC").Find(&history).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal mengambil histori skor kesehatan",
		})
	}

	// Format respons ringkas untuk grafik frontend.
	type point struct {
		HealthScore  int       `json:"health_score"`
		CalculatedAt time.Time `json:"calculated_at"`
	}
	points := make([]point, 0, len(history))
	for _, h := range history {
		points = append(points, point{HealthScore: h.HealthScore, CalculatedAt: h.CalculatedAt})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   points,
		"meta": fiber.Map{
			"range_days": days,
		},
	})
}

// cleanupOldHealthHistories menghapus histori skor kesehatan lebih dari 1 tahun
// agar tabel tidak membengkak (Task 13.8).
func cleanupOldHealthHistories(db *gorm.DB) {
	if db == nil {
		return
	}
	if err := db.Where("calculated_at < ?", time.Now().AddDate(-1, 0, 0)).
		Delete(&models.CCTVHealthHistory{}).Error; err != nil {
		utils.Log.Warnf("⚠️ Gagal cleanup histori skor kesehatan: %v", err)
	}
}
