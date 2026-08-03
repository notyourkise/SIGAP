package handlers

import (
	"math"
	"time"

	"sigap-backend/config"
	"sigap-backend/models"

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
