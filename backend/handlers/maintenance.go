package handlers

import (
	"time"

	"sigap-backend/config"
	"sigap-backend/models"
	"sigap-backend/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type CreateMaintenanceLogRequest struct {
	AssetID       string `json:"asset_id" validate:"required,uuid4"`
	ActionTaken   string `json:"action_taken" validate:"required,min=5"`
	Notes         string `json:"notes" validate:"omitempty"`
	Status        string `json:"status" validate:"required,oneof=scheduled ongoing completed failed"`
	ScheduledDate string `json:"scheduled_date" validate:"omitempty"` // YYYY-MM-DD HH:MM:SS or RFC3339
	CompletedAt   string `json:"completed_at" validate:"omitempty"`   // YYYY-MM-DD HH:MM:SS or RFC3339
}

type UpdateMaintenanceLogRequest struct {
	AssetID       string `json:"asset_id" validate:"omitempty,uuid4"`
	ActionTaken   string `json:"action_taken" validate:"omitempty,min=5"`
	Notes         string `json:"notes" validate:"omitempty"`
	Status        string `json:"status" validate:"omitempty,oneof=scheduled ongoing completed failed"`
	ScheduledDate string `json:"scheduled_date" validate:"omitempty"` // YYYY-MM-DD HH:MM:SS or RFC3339
	CompletedAt   string `json:"completed_at" validate:"omitempty"`   // YYYY-MM-DD HH:MM:SS or RFC3339
}

// CreateMaintenanceLog membuat log pemeliharaan baru (Admin & Teknisi)
// @Summary      Catat Pemeliharaan CCTV
// @Description  Membuat catatan pemeliharaan baru untuk aset CCTV tertentu. ID Teknisi dideteksi otomatis dari token JWT.
// @Tags         Log Pemeliharaan
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body CreateMaintenanceLogRequest true "Detail Catatan Pemeliharaan"
// @Success      201 {object} map[string]interface{} "Catatan berhasil dibuat"
// @Failure      400 {object} map[string]interface{} "Format tidak valid, aset tidak ditemukan, data tidak lengkap"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Router       /maintenance-logs [post]
func CreateMaintenanceLog(c *fiber.Ctx) error {
	var req CreateMaintenanceLogRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Format input tidak valid",
		})
	}

	if errs := utils.ValidateStruct(&req); len(errs) > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Validasi input gagal",
			"errors":  errs,
		})
	}

	// Cek apakah asset_id valid
	assetUUID, err := uuid.Parse(req.AssetID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Asset ID tidak valid",
		})
	}

	var asset models.Asset
	if err := config.DB.First(&asset, "id = ?", assetUUID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Aset CCTV tidak ditemukan",
		})
	}

	// Ambil ID Teknisi dari Token JWT (c.Locals)
	techIDStr, ok := c.Locals("user_id").(string)
	if !ok || techIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Sesi tidak valid, Teknisi tidak ditemukan",
		})
	}

	techUUID, err := uuid.Parse(techIDStr)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal membaca ID teknisi",
		})
	}

	// Parsing Waktu
	var schedDate *time.Time
	if req.ScheduledDate != "" {
		t, err := time.Parse("2006-01-02 15:04:05", req.ScheduledDate)
		if err != nil {
			t, err = time.Parse(time.RFC3339, req.ScheduledDate)
			if err != nil {
				t, err = time.Parse("2006-01-02", req.ScheduledDate)
				if err != nil {
					return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
						"status":  "error",
						"message": "Format scheduled_date tidak valid (gunakan YYYY-MM-DD HH:MM:SS)",
					})
				}
			}
		}
		schedDate = &t
	}

	var compDate *time.Time
	if req.CompletedAt != "" {
		t, err := time.Parse("2006-01-02 15:04:05", req.CompletedAt)
		if err != nil {
			t, err = time.Parse(time.RFC3339, req.CompletedAt)
			if err != nil {
				t, err = time.Parse("2006-01-02", req.CompletedAt)
				if err != nil {
					return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
						"status":  "error",
						"message": "Format completed_at tidak valid (gunakan YYYY-MM-DD HH:MM:SS)",
					})
				}
			}
		}
		compDate = &t
	}

	logEntry := models.MaintenanceLog{
		AssetID:       assetUUID,
		TechnicianID:  techUUID,
		ActionTaken:   req.ActionTaken,
		Notes:         req.Notes,
		Status:        req.Status,
		ScheduledDate: schedDate,
		CompletedAt:   compDate,
	}

	if err := config.DB.Create(&logEntry).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menyimpan log pemeliharaan",
		})
	}

	// Muat ulang dengan data relasi (Asset & User) untuk respon JSON
	config.DB.Preload("Asset").Preload("User").First(&logEntry, logEntry.ID)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status": "success",
		"data":   logEntry,
	})
}

// GetAllMaintenanceLogs mengambil semua log pemeliharaan (Admin & Teknisi)
// @Summary      Ambil Semua Log Pemeliharaan
// @Description  Mendapatkan list riwayat pemeliharaan CCTV terpaginasi dengan filter asset_id, status, pencarian, dan sorting.
// @Tags         Log Pemeliharaan
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        asset_id query string false "Filter berdasarkan UUID Aset"
// @Param        status query string false "Filter berdasarkan status log"
// @Param        search query string false "Cari tindakan (action_taken) atau catatan (notes)"
// @Param        page query integer false "Nomor Halaman (default: 1)"
// @Param        limit query integer false "Jumlah data per halaman (default: 10)"
// @Param        sort_by query string false "Kolom pengurutan (default: created_at)"
// @Param        order query string false "Arah pengurutan: asc, desc (default: desc)"
// @Success      200 {object} map[string]interface{} "Daftar log terpaginasi"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Router       /maintenance-logs [get]
func GetAllMaintenanceLogs(c *fiber.Ctx) error {
	var logs []models.MaintenanceLog
	query := config.DB.Model(&models.MaintenanceLog{}).Preload("Asset").Preload("User")

	// Filter berdasarkan asset_id
	if assetID := c.Query("asset_id"); assetID != "" {
		assetUUID, err := uuid.Parse(assetID)
		if err == nil {
			query = query.Where("asset_id = ?", assetUUID)
		}
	}

	// Filter berdasarkan status
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	// Pencarian teks (berdasarkan tindakan atau catatan)
	if search := c.Query("search"); search != "" {
		query = query.Where("action_taken ILIKE ? OR notes ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// Ambil parameter paginasi & sorting (dengan whitelist keamanan)
	allowedSort := []string{"id", "asset_id", "technician_id", "action_taken", "notes", "status", "scheduled_date", "completed_at", "created_at", "updated_at"}
	limit, page, sort := utils.GetPaginationParams(c, allowedSort)

	var totalRows int64
	if err := query.Count(&totalRows).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghitung total log pemeliharaan",
		})
	}

	offset := (page - 1) * limit
	if err := query.Offset(offset).Limit(limit).Order(sort).Find(&logs).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal mengambil daftar log pemeliharaan",
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"rows":       logs,
			"pagination": utils.CalculatePagination(totalRows, page, limit),
		},
	})
}

// GetMaintenanceLogByID mengambil satu log pemeliharaan secara detail (Admin & Teknisi)
// @Summary      Ambil Log Pemeliharaan Detail
// @Description  Mendapatkan detail satu riwayat pemeliharaan berdasarkan UUID-nya.
// @Tags         Log Pemeliharaan
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Log UUID"
// @Success      200 {object} map[string]interface{} "Data log ditemukan"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      404 {object} map[string]interface{} "Log tidak ditemukan"
// @Router       /maintenance-logs/{id} [get]
func GetMaintenanceLogByID(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID log pemeliharaan tidak valid",
		})
	}

	var logEntry models.MaintenanceLog
	if err := config.DB.Preload("Asset").Preload("User").First(&logEntry, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Log pemeliharaan tidak ditemukan",
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   logEntry,
	})
}

// UpdateMaintenanceLog mengubah log pemeliharaan (Admin & Teknisi)
// @Summary      Update Log Pemeliharaan
// @Description  Memperbarui data riwayat pemeliharaan tertentu.
// @Tags         Log Pemeliharaan
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Log UUID"
// @Param        request body UpdateMaintenanceLogRequest true "Data Update Log"
// @Success      200 {object} map[string]interface{} "Log berhasil diperbarui"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid atau format body salah"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      404 {object} map[string]interface{} "Log tidak ditemukan"
// @Router       /maintenance-logs/{id} [put]
func UpdateMaintenanceLog(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID log pemeliharaan tidak valid",
		})
	}

	var logEntry models.MaintenanceLog
	if err := config.DB.First(&logEntry, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Log pemeliharaan tidak ditemukan",
		})
	}

	var req UpdateMaintenanceLogRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Format input tidak valid",
		})
	}

	if errs := utils.ValidateStruct(&req); len(errs) > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Validasi input gagal",
			"errors":  errs,
		})
	}

	// Update field jika disediakan
	if req.AssetID != "" {
		assetUUID, err := uuid.Parse(req.AssetID)
		if err == nil {
			var asset models.Asset
			if config.DB.First(&asset, "id = ?", assetUUID).Error == nil {
				logEntry.AssetID = assetUUID
			}
		}
	}
	if req.ActionTaken != "" {
		logEntry.ActionTaken = req.ActionTaken
	}
	if req.Notes != "" {
		logEntry.Notes = req.Notes
	}
	if req.Status != "" {
		logEntry.Status = req.Status
	}
	if req.ScheduledDate != "" {
		t, err := time.Parse("2006-01-02 15:04:05", req.ScheduledDate)
		if err != nil {
			t, err = time.Parse(time.RFC3339, req.ScheduledDate)
			if err != nil {
				t, err = time.Parse("2006-01-02", req.ScheduledDate)
			}
		}
		if err == nil {
			logEntry.ScheduledDate = &t
		}
	}
	if req.CompletedAt != "" {
		t, err := time.Parse("2006-01-02 15:04:05", req.CompletedAt)
		if err != nil {
			t, err = time.Parse(time.RFC3339, req.CompletedAt)
			if err != nil {
				t, err = time.Parse("2006-01-02", req.CompletedAt)
			}
		}
		if err == nil {
			logEntry.CompletedAt = &t
		}
	}

	if err := config.DB.Save(&logEntry).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal memperbarui log pemeliharaan",
		})
	}

	config.DB.Preload("Asset").Preload("User").First(&logEntry, logEntry.ID)

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   logEntry,
	})
}

// DeleteMaintenanceLog menghapus data log secara soft delete (Admin saja)
// @Summary      Hapus Log Pemeliharaan
// @Description  Menghapus data riwayat pemeliharaan (soft-delete). Hanya dapat diakses oleh Admin.
// @Tags         Log Pemeliharaan
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Log UUID"
// @Success      200 {object} map[string]interface{} "Log berhasil dihapus secara soft-delete"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Failure      404 {object} map[string]interface{} "Log tidak ditemukan"
// @Router       /maintenance-logs/{id} [delete]
func DeleteMaintenanceLog(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID log pemeliharaan tidak valid",
		})
	}

	var logEntry models.MaintenanceLog
	if err := config.DB.First(&logEntry, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Log pemeliharaan tidak ditemukan",
		})
	}

	if err := config.DB.Delete(&logEntry).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghapus log pemeliharaan",
		})
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Log pemeliharaan berhasil dihapus secara soft-delete",
	})
}
