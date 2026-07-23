package handlers

import (
	"time"

	"sigap-backend/config"
	"sigap-backend/models"

	"github.com/gofiber/fiber/v2"
)

// StatusCount menyimpan jumlah aset per status
type StatusCount struct {
	Status string `json:"status"`
	Count  int64  `json:"count"`
}

// DashboardStatsResponse menyimpan struktur respons statistik dashboard
type DashboardStatsResponse struct {
	TotalAssets          int64         `json:"total_assets"`
	AssetsByStatus       []StatusCount `json:"assets_by_status"`
	TotalMaintenanceLogs int64         `json:"total_maintenance_logs"`
	LogsThisMonth        int64         `json:"logs_this_month"`
	LogsCompletedMonth   int64         `json:"logs_completed_this_month"`
	LogsPendingMonth     int64         `json:"logs_pending_this_month"`
	TotalUsers           int64         `json:"total_users"`
	Timestamp            string        `json:"timestamp"`
}

// GetDashboardStats mengembalikan ringkasan statistik agregat untuk dashboard
// @Summary      Statistik Dashboard
// @Description  Mendapatkan ringkasan data agregat: total aset, jumlah per status, total log pemeliharaan, aktivitas bulan ini, dan jumlah pengguna.
// @Tags         Dashboard
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} DashboardStatsResponse "Statistik berhasil diambil"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      500 {object} map[string]interface{} "Gagal mengambil statistik"
// @Router       /dashboard/stats [get]
func GetDashboardStats(c *fiber.Ctx) error {
	var stats DashboardStatsResponse

	// 1. Total Aset
	if err := config.DB.Model(&models.Asset{}).Count(&stats.TotalAssets).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghitung total aset",
		})
	}

	// 2. Jumlah Aset per Status
	statuses := []string{"aktif", "rusak", "pemeliharaan", "nonaktif"}
	for _, s := range statuses {
		var count int64
		config.DB.Model(&models.Asset{}).Where("status = ?", s).Count(&count)
		stats.AssetsByStatus = append(stats.AssetsByStatus, StatusCount{
			Status: s,
			Count:  count,
		})
	}

	// 3. Total Log Pemeliharaan
	if err := config.DB.Model(&models.MaintenanceLog{}).Count(&stats.TotalMaintenanceLogs).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghitung total log pemeliharaan",
		})
	}

	// 4. Log Pemeliharaan Bulan Ini
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	endOfMonth := startOfMonth.AddDate(0, 1, 0)

	config.DB.Model(&models.MaintenanceLog{}).
		Where("created_at >= ? AND created_at < ?", startOfMonth, endOfMonth).
		Count(&stats.LogsThisMonth)

	// 5. Log Selesai Bulan Ini
	config.DB.Model(&models.MaintenanceLog{}).
		Where("created_at >= ? AND created_at < ? AND status = ?", startOfMonth, endOfMonth, "completed").
		Count(&stats.LogsCompletedMonth)

	// 6. Log Pending Bulan Ini
	config.DB.Model(&models.MaintenanceLog{}).
		Where("created_at >= ? AND created_at < ? AND status = ?", startOfMonth, endOfMonth, "pending").
		Count(&stats.LogsPendingMonth)

	// 7. Total Pengguna Terdaftar
	config.DB.Model(&models.User{}).Count(&stats.TotalUsers)

	stats.Timestamp = now.Format(time.RFC3339)

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   stats,
	})
}
