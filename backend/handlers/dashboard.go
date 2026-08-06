package handlers

import (
	"math"
	"time"

	"sigap-backend/config"
	"sigap-backend/models"
	"sigap-backend/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DashboardStatusCount mewakili jumlah CCTV per status.
type DashboardStatusCount struct {
	Status string `json:"status"`
	Count  int64  `json:"count"`
}

// DashboardTrendPoint mewakili jumlah log pemeliharaan per status pada satu bulan.
type DashboardTrendPoint struct {
	Month     string `json:"month"`
	Scheduled int64  `json:"scheduled"`
	Ongoing   int64  `json:"ongoing"`
	Completed int64  `json:"completed"`
	Failed    int64  `json:"failed"`
}

// DashboardOverdueItem mewakili satu pekerjaan pemeliharaan yang melewati jadwal.
type DashboardOverdueItem struct {
	ID             string  `json:"id"`
	CCTVName       string  `json:"cctv_name"`
	CCTVLocation   string  `json:"cctv_location"`
	Status         string  `json:"status"`
	ActionTaken    string  `json:"action_taken"`
	ScheduledDate  string  `json:"scheduled_date"`
	DaysOverdue    float64 `json:"days_overdue"`
}

// DashboardSummary menyediakan ringkasan analitik untuk halaman dashboard
// (Admin & Teknisi).
// @Summary      Ringkasan Analitik Dashboard
// @Description  Mengembalikan distribusi status CCTV, tren pemeliharaan 6 bulan terakhir,
//
//	dan daftar pekerjaan pemeliharaan yang melewati jadwal (overdue).
//
// @Tags         Dashboard
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "Ringkasan analitik dashboard"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid"
// @Failure      500 {object} map[string]interface{} "Gagal memuat data dashboard"
// @Router       /dashboard/summary [get]
func DashboardSummary(c *fiber.Ctx) error {
	now := time.Now()

	// 1. Distribusi status CCTV
	var statusCounts []DashboardStatusCount
	if err := config.DB.Model(&models.CCTV{}).
		Select("status, count(*) as count").
		Group("status").
		Order("count desc").
		Scan(&statusCounts).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghitung distribusi status CCTV",
		})
	}

	// 2. Tren pemeliharaan 6 bulan terakhir (dari bulan 5 bulan lalu s/d bulan ini)
	trendSlice := maintenanceTrend(config.DB, now)

	// 3. Pekerjaan pemeliharaan yang melewati jadwal (overdue)
	var overdueLogs []models.MaintenanceLog
	if err := config.DB.Preload("CCTV").
		Where("status IN ? AND scheduled_date IS NOT NULL AND scheduled_date < ?", []string{"scheduled", "ongoing"}, now).
		Order("scheduled_date asc").
		Limit(10).
		Find(&overdueLogs).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal memuat pekerjaan pemeliharaan yang overdue",
		})
	}

	overdue := make([]DashboardOverdueItem, 0, len(overdueLogs))
	for _, log := range overdueLogs {
		daysOverdue := float64(0)
		if log.ScheduledDate != nil {
			daysOverdue = now.Sub(*log.ScheduledDate).Hours() / 24
		}
		overdue = append(overdue, DashboardOverdueItem{
			ID:             log.ID.String(),
			CCTVName:       log.CCTV.Nama,
			CCTVLocation:   log.CCTV.LokasiArea,
			Status:         log.Status,
			ActionTaken:    log.ActionTaken,
			ScheduledDate:  log.ScheduledDate.Format("2006-01-02 15:04:05"),
			DaysOverdue:    math.Round(daysOverdue*10) / 10,
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"cctv_status":       statusCounts,
			"maintenance_trend": trendSlice,
			"overdue":           overdue,
		},
	})
}

// DashboardTaskItem mewakili satu tugas pemeliharaan (notifikasi untuk teknisi).
type DashboardTaskItem struct {
	ID             string  `json:"id"`
	CCTVID         string  `json:"cctv_id"`
	CCTVName       string  `json:"cctv_name"`
	CCTVCode       string  `json:"cctv_kode"`
	CCTVLocation   string  `json:"cctv_location"`
	Status         string  `json:"status"`
	ActionTaken    string  `json:"action_taken"`
	ScheduledDate  string  `json:"scheduled_date"`
	DaysOverdue    float64 `json:"days_overdue"`
}

// DashboardTasks mengembalikan daftar tugas pemeliharaan aktif untuk pengguna
// saat ini — Teknisi hanya melihat tugas miliknya, Admin melihat semua.
// @Summary      Tugas & Notifikasi Teknisi
// @Description  Mengembalikan tugas pemeliharaan berstatus terjadwal/berjalan milik teknisi
//               (atau seluruh tugas bila Admin) beserta jumlah tugas yang melewati jadwal.
//
// @Tags         Dashboard
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "Daftar tugas aktif + ringkasan"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid"
// @Failure      500 {object} map[string]interface{} "Gagal memuat tugas"
// @Router       /dashboard/tasks [get]
func DashboardTasks(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(string)
	role, _ := c.Locals("role").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Identitas pengguna tidak ditemukan dalam token",
		})
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Identitas pengguna tidak valid",
		})
	}

	now := time.Now()
	query := config.DB.Preload("CCTV").
		Where("status IN ?", []string{"scheduled", "ongoing"})
	if role != "admin" {
		// Teknisi hanya melihat tugas yang ditugaskan kepadanya.
		query = query.Where("technician_id = ?", uid)
	}

	var logs []models.MaintenanceLog
	if err := query.Order("scheduled_date asc").Limit(20).Find(&logs).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal memuat daftar tugas pemeliharaan",
		})
	}

	tasks := make([]DashboardTaskItem, 0, len(logs))
	overdueCount := 0
	for _, log := range logs {
		daysOverdue := float64(0)
		if log.ScheduledDate != nil && log.ScheduledDate.Before(now) {
			daysOverdue = math.Round(now.Sub(*log.ScheduledDate).Hours()/24*10) / 10
			overdueCount++
		}
		sched := ""
		if log.ScheduledDate != nil {
			sched = log.ScheduledDate.Format("2006-01-02 15:04:05")
		}
		tasks = append(tasks, DashboardTaskItem{
			ID:            log.ID.String(),
			CCTVID:        log.CCTVID.String(),
			CCTVName:      log.CCTV.Nama,
			CCTVCode:      log.CCTV.KodeCCTV,
			CCTVLocation:  log.CCTV.LokasiArea,
			Status:        log.Status,
			ActionTaken:   log.ActionTaken,
			ScheduledDate: sched,
			DaysOverdue:   daysOverdue,
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"tasks":         tasks,
			"pending_count": len(tasks),
			"overdue_count": overdueCount,
		},
	})
}

// TechnicianPerformance adalah ringkasan kinerja satu teknisi pada rentang
// waktu yang diminta admin.
type TechnicianPerformance struct {
	UserID             string  `json:"user_id"`
	FullName           string  `json:"full_name"`
	Username           string  `json:"username"`
	TotalTasks         int64   `json:"total_tasks"`
	Completed          int64   `json:"completed"`
	Failed             int64   `json:"failed"`
	Ongoing            int64   `json:"ongoing"`
	CompletionRate     float64 `json:"completion_rate"`
	AvgCompletionHours float64 `json:"avg_completion_hours"`
	OverdueCount       int64   `json:"overdue_count"`
	FieldCheckCount    int64   `json:"field_check_count"`
	QualityScore       float64 `json:"quality_score"`
}

type technicianPerformanceRow struct {
	UserID             string
	FullName           string
	Username           string
	TotalTasks         int64
	Completed          int64
	Failed             int64
	Ongoing            int64
	AvgCompletionHours float64
	OverdueCount       int64
	FieldCheckCount    int64
	OKCount            int64
}

// GetTechnicianPerformance mengembalikan statistik teknisi untuk dashboard admin.
// Waktu penugasan memakai AssignedAt; log lama yang belum memiliki kolom itu
// memakai CreatedAt sebagai fallback agar histori tetap dapat dianalisis.
// @Summary      Kinerja Teknisi
// @Description  Mengembalikan ringkasan tugas pemeliharaan dan pengecekan lapangan per teknisi dalam rentang tanggal tertentu.
// @Tags         Dashboard
// @Produce      json
// @Security     BearerAuth
// @Param        start_date query string false "Tanggal mulai (YYYY-MM-DD, default 30 hari terakhir)"
// @Param        end_date query string false "Tanggal akhir (YYYY-MM-DD, inklusif, default hari ini)"
// @Success      200 {object} map[string]interface{} "Daftar kinerja teknisi"
// @Failure      400 {object} map[string]interface{} "Rentang tanggal tidak valid"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid"
// @Failure      403 {object} map[string]interface{} "Hanya admin yang dapat mengakses dashboard kinerja"
// @Failure      500 {object} map[string]interface{} "Gagal memuat kinerja teknisi"
// @Router       /dashboard/technician-performance [get]
func GetTechnicianPerformance(c *fiber.Ctx) error {
	if role, _ := c.Locals("role").(string); role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "error",
			"message": "Akses ditolak: dashboard kinerja teknisi hanya untuk admin",
		})
	}

	now := time.Now()
	start, end := utils.ParseDateRangeParams(c)
	startDate := now.AddDate(0, 0, -30)
	endDate := now
	if start != nil {
		startDate = *start
	}
	if end != nil {
		endDate = *end
	}
	if !endDate.After(startDate) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Rentang tanggal tidak valid: tanggal akhir harus setelah tanggal mulai",
		})
	}

	// Satu query agregat menjaga jumlah round-trip database tetap konstan.
	// Filter tugas memakai waktu assign; COALESCE menjaga kompatibilitas dengan
	// log lama yang dibuat sebelum kolom assigned_at tersedia.
	query := `
		SELECT
			u.id AS user_id,
			u.full_name,
			u.username,
			COALESCE(ml.total_tasks, 0) AS total_tasks,
			COALESCE(ml.completed, 0) AS completed,
			COALESCE(ml.failed, 0) AS failed,
			COALESCE(ml.ongoing, 0) AS ongoing,
			COALESCE(ml.avg_completion_hours, 0)::double precision AS avg_completion_hours,
			COALESCE(ml.overdue_count, 0) AS overdue_count,
			COALESCE(fm.field_check_count, 0) AS field_check_count,
			COALESCE(fm.ok_count, 0) AS ok_count
		FROM users AS u
		LEFT JOIN (
			SELECT
				technician_id,
				COUNT(*) AS total_tasks,
				COUNT(*) FILTER (WHERE status = 'completed') AS completed,
				COUNT(*) FILTER (WHERE status = 'failed') AS failed,
				COUNT(*) FILTER (WHERE status = 'ongoing') AS ongoing,
				ROUND((AVG(EXTRACT(EPOCH FROM (completed_at - COALESCE(assigned_at, created_at))) / 3600.0)
					FILTER (WHERE status = 'completed' AND completed_at IS NOT NULL))::numeric, 1) AS avg_completion_hours,
				COUNT(*) FILTER (
					WHERE (status IN ('scheduled', 'ongoing') AND scheduled_date IS NOT NULL AND scheduled_date < ?)
					   OR (status = 'completed' AND scheduled_date IS NOT NULL AND completed_at IS NOT NULL AND scheduled_date < completed_at)
				) AS overdue_count
			FROM maintenance_logs
			WHERE technician_id IS NOT NULL
			  AND COALESCE(assigned_at, created_at) >= ?
			  AND COALESCE(assigned_at, created_at) < ?
			  AND deleted_at IS NULL
			GROUP BY technician_id
		) AS ml ON ml.technician_id = u.id
		LEFT JOIN (
			SELECT
				technician_id,
				COUNT(*) AS field_check_count,
				COUNT(*) FILTER (WHERE overall_status = 'ok') AS ok_count
			FROM field_maintenances
			WHERE created_at >= ?
			  AND created_at < ?
			  AND deleted_at IS NULL
			GROUP BY technician_id
		) AS fm ON fm.technician_id = u.id
		WHERE u.role = 'teknisi'
		  AND u.deleted_at IS NULL
		  AND (ml.technician_id IS NOT NULL OR fm.technician_id IS NOT NULL)
		ORDER BY total_tasks DESC, field_check_count DESC, u.full_name ASC
	`

	var rows []technicianPerformanceRow
	if err := config.DB.Raw(query, now, startDate, endDate, startDate, endDate).Scan(&rows).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal memuat kinerja teknisi",
		})
	}

	performance := make([]TechnicianPerformance, 0, len(rows))
	for _, row := range rows {
		completionRate := float64(0)
		if row.TotalTasks > 0 {
			completionRate = roundDashboardMetric(float64(row.Completed) / float64(row.TotalTasks) * 100)
		}
		qualityScore := float64(0)
		if row.FieldCheckCount > 0 {
			qualityScore = roundDashboardMetric(float64(row.OKCount) / float64(row.FieldCheckCount) * 100)
		}

		performance = append(performance, TechnicianPerformance{
			UserID:             row.UserID,
			FullName:           row.FullName,
			Username:           row.Username,
			TotalTasks:         row.TotalTasks,
			Completed:          row.Completed,
			Failed:             row.Failed,
			Ongoing:            row.Ongoing,
			CompletionRate:     completionRate,
			AvgCompletionHours: roundDashboardMetric(row.AvgCompletionHours),
			OverdueCount:       row.OverdueCount,
			FieldCheckCount:    row.FieldCheckCount,
			QualityScore:       qualityScore,
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   performance,
	})
}

func roundDashboardMetric(value float64) float64 {
	return math.Round(value*10) / 10
}

// maintenanceTrend menghitung jumlah log pemeliharaan per status untuk 6 bulan
// terakhir (bulan berjalan + 5 bulan sebelumnya), diurutkan dari bulan tertua.
func maintenanceTrend(db *gorm.DB, now time.Time) []DashboardTrendPoint {
	start := time.Date(now.Year(), now.Month()-5, 1, 0, 0, 0, 0, now.Location())

	// Inisialisasi 6 slot bulan secara berurutan (order stabil).
	trend := map[string]*DashboardTrendPoint{}
	months := make([]string, 0, 6)
	for i := 0; i < 6; i++ {
		m := time.Date(start.Year(), start.Month()+time.Month(i), 1, 0, 0, 0, 0, now.Location())
		key := m.Format("2006-01")
		months = append(months, key)
		trend[key] = &DashboardTrendPoint{Month: key}
	}

	// Ambil hanya kolom yang dibutuhkan dalam 6 bulan terakhir.
	var logs []struct {
		Status    string
		CreatedAt time.Time
	}
	if err := db.Model(&models.MaintenanceLog{}).
		Select("status, created_at").
		Where("created_at >= ?", start).
		Find(&logs).Error; err != nil {
		return []DashboardTrendPoint{}
	}

	for _, log := range logs {
		point, ok := trend[log.CreatedAt.Format("2006-01")]
		if !ok {
			continue
		}
		switch log.Status {
		case "scheduled":
			point.Scheduled++
		case "ongoing":
			point.Ongoing++
		case "completed":
			point.Completed++
		case "failed":
			point.Failed++
		}
	}

	result := make([]DashboardTrendPoint, 0, len(months))
	for _, key := range months {
		result = append(result, *trend[key])
	}
	return result
}
