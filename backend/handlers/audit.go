package handlers

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"time"

	"sigap-backend/config"
	"sigap-backend/models"
	"sigap-backend/utils"

	"github.com/gofiber/fiber/v2"
)

// GetAuditLogs mengembalikan daftar log audit sistem terpaginasi (Khusus Admin)
// @Summary      Mendapatkan Daftar Audit Log
// @Description  Mengembalikan daftar riwayat aktivitas sistem berdasarkan filter modul, rentang tanggal, pencarian, pengurutan, dan paginasi. Hanya bisa diakses oleh Admin.
// @Tags         Audit Log
// @Produce      json
// @Security     BearerAuth
// @Param        page query int false "Nomor Halaman (default: 1)"
// @Param        limit query int false "Jumlah Data per Halaman (default: 10)"
// @Param        sort_by query string false "Kolom Pengurutan (created_at, action, module, username)"
// @Param        order query string false "Arah Pengurutan (asc, desc)"
// @Param        module query string false "Filter berdasarkan nama modul (AUTH, CCTV, ASSET, MAINTENANCE, USER)"
// @Param        start_date query string false "Filter rentang awal (YYYY-MM-DD)"
// @Param        end_date query string false "Filter rentang akhir (YYYY-MM-DD)"
// @Param        search query string false "Cari berdasarkan username, action, description, atau ip_address"
// @Success      200 {object} map[string]interface{} "Sukses mendapatkan daftar audit log"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid"
// @Failure      403 {object} map[string]interface{} "Akses ditolak"
// @Failure      500 {object} map[string]interface{} "Gagal mengambil data"
// @Router       /audit-logs [get]
func GetAuditLogs(c *fiber.Ctx) error {
	var auditLogs []models.AuditLog
	var totalRows int64

	allowedSorts := []string{"created_at", "action", "module", "username", "user_role", "ip_address"}
	limit, page, sort := utils.GetPaginationParams(c, allowedSorts)
	offset := (page - 1) * limit

	query := config.DB.Model(&models.AuditLog{})

	// Filter berdasarkan modul
	if module := c.Query("module"); module != "" {
		query = query.Where("module = ?", module)
	}

	// Filter rentang tanggal (created_at)
	if start, end := utils.ParseDateRangeParams(c); start != nil || end != nil {
		if start != nil {
			query = query.Where("created_at >= ?", *start)
		}
		if end != nil {
			query = query.Where("created_at < ?", *end)
		}
	}

	// Pencarian teks
	if search := c.Query("search"); search != "" {
		pattern := "%" + search + "%"
		query = query.Where("username ILIKE ? OR action ILIKE ? OR description ILIKE ? OR ip_address ILIKE ?", pattern, pattern, pattern, pattern)
	}

	// Hitung total data terfilter
	if err := query.Count(&totalRows).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghitung total data audit log",
		})
	}

	// Ambil data terpaginasi
	if err := query.Order(sort).Limit(limit).Offset(offset).Find(&auditLogs).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal mengambil data audit log dari database",
		})
	}

	pagination := utils.CalculatePagination(totalRows, page, limit)

	return c.JSON(fiber.Map{
		"status":     "success",
		"message":    "Berhasil mengambil data audit log",
		"data":       auditLogs,
		"pagination": pagination,
	})
}

// ExportAuditLogs memproduksi file CSV audit log (Khusus Admin)
// @Summary      Ekspor Audit Log ke CSV
// @Description  Mengunduh seluruh riwayat aktivitas sistem terfilter (tanpa limit halaman) dalam format file CSV. Hanya bisa diakses oleh Admin.
// @Tags         Audit Log
// @Produce      text/csv
// @Security     BearerAuth
// @Param        module query string false "Filter berdasarkan nama modul"
// @Param        start_date query string false "Filter rentang awal (YYYY-MM-DD)"
// @Param        end_date query string false "Filter rentang akhir (YYYY-MM-DD)"
// @Param        search query string false "Cari berdasarkan username, action, description, atau ip_address"
// @Success      200 {file} string "File CSV Audit Log"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid"
// @Failure      403 {object} map[string]interface{} "Akses ditolak"
// @Failure      500 {object} map[string]interface{} "Gagal mengekspor data"
// @Router       /audit-logs/export [get]
func ExportAuditLogs(c *fiber.Ctx) error {
	var auditLogs []models.AuditLog

	query := config.DB.Model(&models.AuditLog{})

	// Filter berdasarkan modul
	if module := c.Query("module"); module != "" {
		query = query.Where("module = ?", module)
	}

	// Filter rentang tanggal (created_at)
	if start, end := utils.ParseDateRangeParams(c); start != nil || end != nil {
		if start != nil {
			query = query.Where("created_at >= ?", *start)
		}
		if end != nil {
			query = query.Where("created_at < ?", *end)
		}
	}

	// Pencarian teks
	if search := c.Query("search"); search != "" {
		pattern := "%" + search + "%"
		query = query.Where("username ILIKE ? OR action ILIKE ? OR description ILIKE ? OR ip_address ILIKE ?", pattern, pattern, pattern, pattern)
	}

	// Urutkan dari yang terbaru
	if err := query.Order("created_at desc").Find(&auditLogs).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal mengambil data audit log untuk diekspor",
		})
	}

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	writer.Comma = ','

	header := []string{"ID", "Waktu", "Username", "Role", "Modul", "Aksi", "Deskripsi", "Client IP", "User Agent"}
	if err := writer.Write(header); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menulis header CSV",
		})
	}

	for _, logEntry := range auditLogs {
		row := []string{
			logEntry.ID.String(),
			logEntry.CreatedAt.Format("2006-01-02 15:04:05"),
			logEntry.Username,
			logEntry.UserRole,
			logEntry.Module,
			logEntry.Action,
			logEntry.Description,
			logEntry.IPAddress,
			logEntry.UserAgent,
		}
		if err := writer.Write(row); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "Gagal menulis baris data CSV",
			})
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal merampungkan penulisan CSV",
		})
	}

	filename := fmt.Sprintf("audit_logs_export_%s.csv", time.Now().Format("20060102_150405"))
	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	return c.Send(buf.Bytes())
}
