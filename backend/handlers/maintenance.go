package handlers

import (
	"time"

	"sigap-backend/config"
	"sigap-backend/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type MaintenanceLogRequest struct {
	AssetID       string `json:"asset_id"`
	ActionTaken   string `json:"action_taken"`
	Notes         string `json:"notes"`
	Status        string `json:"status"` // e.g. scheduled, ongoing, completed, failed
	ScheduledDate string `json:"scheduled_date"` // YYYY-MM-DD HH:MM:SS or RFC3339
	CompletedAt   string `json:"completed_at"`   // YYYY-MM-DD HH:MM:SS or RFC3339
}

// CreateMaintenanceLog membuat log pemeliharaan baru (Admin & Teknisi)
func CreateMaintenanceLog(c *fiber.Ctx) error {
	var req MaintenanceLogRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Format input tidak valid",
		})
	}

	if req.AssetID == "" || req.ActionTaken == "" || req.Status == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Asset ID, Tindakan (action_taken), dan Status wajib diisi",
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
func GetAllMaintenanceLogs(c *fiber.Ctx) error {
	var logs []models.MaintenanceLog
	query := config.DB.Preload("Asset").Preload("User")

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

	if err := query.Find(&logs).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal mengambil daftar log pemeliharaan",
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   logs,
	})
}

// GetMaintenanceLogByID mengambil satu log pemeliharaan secara detail (Admin & Teknisi)
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

	var req MaintenanceLogRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Format input tidak valid",
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
