package handlers

import (
	"time"

	"sigap-backend/config"
	"sigap-backend/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type AssetRequest struct {
	Name             string  `json:"name"`
	SerialNumber     string  `json:"serial_number"`
	IPAddress        string  `json:"ip_address"`
	LocationName     string  `json:"location_name"`
	Latitude         float64 `json:"latitude"`
	Longitude        float64 `json:"longitude"`
	Status           string  `json:"status"`
	Vendor           string  `json:"vendor"`
	InstallationDate string  `json:"installation_date"` // format: YYYY-MM-DD
}

// CreateAsset menambahkan aset CCTV baru (Admin saja)
func CreateAsset(c *fiber.Ctx) error {
	var req AssetRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Format input tidak valid",
		})
	}

	if req.Name == "" || req.SerialNumber == "" || req.Status == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Nama, Serial Number, dan Status wajib diisi",
		})
	}

	// Validasi status
	validStatuses := map[string]bool{"aktif": true, "rusak": true, "pemeliharaan": true, "nonaktif": true}
	if !validStatuses[req.Status] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Status tidak valid. Gunakan 'aktif', 'rusak', 'pemeliharaan', atau 'nonaktif'",
		})
	}

	// Cek apakah serial number unik
	var existingAsset models.Asset
	err := config.DB.Where("serial_number = ?", req.SerialNumber).First(&existingAsset).Error
	if err == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Serial Number sudah digunakan oleh aset lain",
		})
	}

	// Parse Tanggal
	var instDate time.Time
	if req.InstallationDate != "" {
		instDate, err = time.Parse("2006-01-02", req.InstallationDate)
		if err != nil {
			instDate, err = time.Parse(time.RFC3339, req.InstallationDate)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"status":  "error",
					"message": "Format tanggal installation_date tidak valid (gunakan YYYY-MM-DD)",
				})
			}
		}
	} else {
		instDate = time.Now()
	}

	asset := models.Asset{
		Name:             req.Name,
		SerialNumber:     req.SerialNumber,
		IPAddress:        req.IPAddress,
		LocationName:     req.LocationName,
		Latitude:         req.Latitude,
		Longitude:        req.Longitude,
		Status:           req.Status,
		Vendor:           req.Vendor,
		InstallationDate: instDate,
	}

	if err := config.DB.Create(&asset).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menyimpan aset baru",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status": "success",
		"data":   asset,
	})
}

// GetAllAssets mengambil daftar semua aset CCTV (Admin & Teknisi)
func GetAllAssets(c *fiber.Ctx) error {
	var assets []models.Asset
	query := config.DB

	// Filter berdasarkan status
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	// Filter berdasarkan vendor
	if vendor := c.Query("vendor"); vendor != "" {
		query = query.Where("vendor = ?", vendor)
	}

	// Pencarian teks (berdasarkan nama atau serial number)
	if search := c.Query("search"); search != "" {
		query = query.Where("name ILIKE ? OR serial_number ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	if err := query.Find(&assets).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal mengambil daftar aset",
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   assets,
	})
}

// GetAssetByID mengambil detail satu aset CCTV (Admin & Teknisi)
func GetAssetByID(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID aset tidak valid",
		})
	}

	var asset models.Asset
	if err := config.DB.First(&asset, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Aset tidak ditemukan",
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   asset,
	})
}

// UpdateAsset mengubah data aset CCTV (Admin saja)
func UpdateAsset(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID aset tidak valid",
		})
	}

	var asset models.Asset
	if err := config.DB.First(&asset, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Aset tidak ditemukan",
		})
	}

	var req AssetRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Format input tidak valid",
		})
	}

	// Update field jika disediakan
	if req.Name != "" {
		asset.Name = req.Name
	}
	if req.SerialNumber != "" {
		// Validasi apakah serial number unik jika diubah
		if req.SerialNumber != asset.SerialNumber {
			var existingAsset models.Asset
			err = config.DB.Where("serial_number = ? AND id != ?", req.SerialNumber, id).First(&existingAsset).Error
			if err == nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"status":  "error",
					"message": "Serial Number sudah digunakan oleh aset lain",
				})
			}
			asset.SerialNumber = req.SerialNumber
		}
	}
	if req.IPAddress != "" {
		asset.IPAddress = req.IPAddress
	}
	if req.LocationName != "" {
		asset.LocationName = req.LocationName
	}
	if req.Latitude != 0 {
		asset.Latitude = req.Latitude
	}
	if req.Longitude != 0 {
		asset.Longitude = req.Longitude
	}
	if req.Status != "" {
		validStatuses := map[string]bool{"aktif": true, "rusak": true, "pemeliharaan": true, "nonaktif": true}
		if !validStatuses[req.Status] {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "error",
				"message": "Status tidak valid. Gunakan 'aktif', 'rusak', 'pemeliharaan', atau 'nonaktif'",
			})
		}
		asset.Status = req.Status
	}
	if req.Vendor != "" {
		asset.Vendor = req.Vendor
	}
	if req.InstallationDate != "" {
		instDate, err := time.Parse("2006-01-02", req.InstallationDate)
		if err != nil {
			instDate, err = time.Parse(time.RFC3339, req.InstallationDate)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"status":  "error",
					"message": "Format tanggal installation_date tidak valid (gunakan YYYY-MM-DD)",
				})
			}
		}
		asset.InstallationDate = instDate
	}

	if err := config.DB.Save(&asset).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal memperbarui data aset",
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   asset,
	})
}

// DeleteAsset menghapus data aset secara soft delete (Admin saja)
func DeleteAsset(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID aset tidak valid",
		})
	}

	var asset models.Asset
	if err := config.DB.First(&asset, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Aset tidak ditemukan",
		})
	}

	if err := config.DB.Delete(&asset).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghapus aset",
		})
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Aset berhasil dihapus secara soft-delete",
	})
}
