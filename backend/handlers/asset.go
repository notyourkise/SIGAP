package handlers

import (
	"time"

	"sigap-backend/config"
	"sigap-backend/models"
	"sigap-backend/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type CreateAssetRequest struct {
	Name             string  `json:"name" validate:"required,min=3,max=100"`
	SerialNumber     string  `json:"serial_number" validate:"required,max=100"`
	IPAddress        string  `json:"ip_address" validate:"omitempty,ip"`
	LocationName     string  `json:"location_name" validate:"omitempty,max=255"`
	Latitude         float64 `json:"latitude" validate:"omitempty,latitude"`
	Longitude        float64 `json:"longitude" validate:"omitempty,longitude"`
	Status           string  `json:"status" validate:"required,oneof=aktif rusak pemeliharaan nonaktif"`
	Vendor           string  `json:"vendor" validate:"omitempty,max=100"`
	InstallationDate string  `json:"installation_date" validate:"omitempty"` // format: YYYY-MM-DD
}

type UpdateAssetRequest struct {
	Name             string  `json:"name" validate:"omitempty,min=3,max=100"`
	SerialNumber     string  `json:"serial_number" validate:"omitempty,max=100"`
	IPAddress        string  `json:"ip_address" validate:"omitempty,ip"`
	LocationName     string  `json:"location_name" validate:"omitempty,max=255"`
	Latitude         float64 `json:"latitude" validate:"omitempty,latitude"`
	Longitude        float64 `json:"longitude" validate:"omitempty,longitude"`
	Status           string  `json:"status" validate:"omitempty,oneof=aktif rusak pemeliharaan nonaktif"`
	Vendor           string  `json:"vendor" validate:"omitempty,max=100"`
	InstallationDate string  `json:"installation_date" validate:"omitempty"` // format: YYYY-MM-DD
}

// CreateAsset menambahkan aset CCTV baru (Admin saja)
// @Summary      Tambah Aset CCTV
// @Description  Membuat record aset CCTV baru. Hanya dapat diakses oleh Admin.
// @Tags         Aset CCTV
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body CreateAssetRequest true "Data Aset Baru"
// @Success      201 {object} map[string]interface{} "Aset berhasil dibuat"
// @Failure      400 {object} map[string]interface{} "Data kurang, serial number duplikat, status tidak valid"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Router       /assets [post]
func CreateAsset(c *fiber.Ctx) error {
	var req CreateAssetRequest
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
// @Summary      Ambil Semua Aset CCTV
// @Description  Mendapatkan list seluruh aset CCTV terpaginasi dengan filter status, vendor, pencarian, dan sorting.
// @Tags         Aset CCTV
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        status query string false "Filter Status (aktif, rusak, pemeliharaan, nonaktif)"
// @Param        vendor query string false "Filter Vendor"
// @Param        search query string false "Cari nama, serial number, atau lokasi"
// @Param        page query integer false "Nomor Halaman (default: 1)"
// @Param        limit query integer false "Jumlah data per halaman (default: 10)"
// @Param        sort_by query string false "Kolom pengurutan (default: created_at)"
// @Param        order query string false "Arah pengurutan: asc, desc (default: desc)"
// @Success      200 {object} map[string]interface{} "Daftar aset terpaginasi"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Router       /assets [get]
func GetAllAssets(c *fiber.Ctx) error {
	var assets []models.Asset
	query := config.DB.Model(&models.Asset{})

	// Filter berdasarkan status
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	// Filter berdasarkan vendor
	if vendor := c.Query("vendor"); vendor != "" {
		query = query.Where("vendor = ?", vendor)
	}

	// Pencarian teks (berdasarkan nama, serial number, atau lokasi)
	if search := c.Query("search"); search != "" {
		query = query.Where("name ILIKE ? OR serial_number ILIKE ? OR location_name ILIKE ?", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	// Ambil parameter paginasi & sorting (dengan whitelist keamanan)
	allowedSort := []string{"id", "name", "serial_number", "ip_address", "location_name", "status", "vendor", "installation_date", "created_at", "updated_at"}
	limit, page, sort := utils.GetPaginationParams(c, allowedSort)

	var totalRows int64
	if err := query.Count(&totalRows).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghitung total aset",
		})
	}

	offset := (page - 1) * limit
	if err := query.Offset(offset).Limit(limit).Order(sort).Find(&assets).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal mengambil daftar aset",
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"rows":       assets,
			"pagination": utils.CalculatePagination(totalRows, page, limit),
		},
	})
}

// GetAssetByID mengambil detail satu aset CCTV (Admin & Teknisi)
// @Summary      Ambil Aset CCTV Detail
// @Description  Mendapatkan detail satu aset CCTV berdasarkan UUID-nya.
// @Tags         Aset CCTV
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Asset UUID"
// @Success      200 {object} map[string]interface{} "Data aset ditemukan"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      404 {object} map[string]interface{} "Aset tidak ditemukan"
// @Router       /assets/{id} [get]
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
// @Summary      Update Aset CCTV
// @Description  Memperbarui data aset CCTV tertentu. Hanya dapat diakses oleh Admin.
// @Tags         Aset CCTV
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Asset UUID"
// @Param        request body UpdateAssetRequest true "Data Update Aset"
// @Success      200 {object} map[string]interface{} "Aset berhasil diperbarui"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid, format body salah, atau serial number duplikat"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Failure      404 {object} map[string]interface{} "Aset tidak ditemukan"
// @Router       /assets/{id} [put]
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

	var req UpdateAssetRequest
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
// @Summary      Hapus Aset CCTV
// @Description  Menghapus aset CCTV (soft-delete). Hanya dapat diakses oleh Admin.
// @Tags         Aset CCTV
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Asset UUID"
// @Success      200 {object} map[string]interface{} "Aset berhasil dihapus secara soft-delete"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Failure      404 {object} map[string]interface{} "Aset tidak ditemukan"
// @Router       /assets/{id} [delete]
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
