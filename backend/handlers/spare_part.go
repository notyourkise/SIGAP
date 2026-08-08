package handlers

import (
	"fmt"
	"strings"

	"sigap-backend/config"
	"sigap-backend/models"
	"sigap-backend/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// EventStockLow adalah nama event WebSocket yang dipancarkan ketika stok suku
// cadang jatuh di bawah ambang minimum_order. Dikonsumsi dashboard widget
// "Suku Cadang Stok Rendah" di frontend.
const EventStockLow = "STOCK_LOW"

// CreateSparePartRequest adalah body pembuatan suku cadang baru (Admin saja).
type CreateSparePartRequest struct {
	Nama         string  `json:"nama" validate:"required,min=2,max=100"`
	SKU          string  `json:"sku" validate:"required,min=2,max=50"`
	Kategori     string  `json:"kategori" validate:"omitempty,max=50"`
	Stok         int     `json:"stok" validate:"gte=0"`
	MinimumOrder int     `json:"minimum_order" validate:"gte=0"`
	Satuan       string  `json:"satuan" validate:"omitempty,max=20"`
	HargaSatuan  float64 `json:"harga_satuan" validate:"gte=0"`
	Keterangan   string  `json:"keterangan" validate:"omitempty"`
}

// UpdateSparePartRequest adalah body pembaruan suku cadang (Admin saja).
type UpdateSparePartRequest struct {
	Nama         string   `json:"nama" validate:"omitempty,min=2,max=100"`
	SKU          string   `json:"sku" validate:"omitempty,min=2,max=50"`
	Kategori     string   `json:"kategori" validate:"omitempty,max=50"`
	Stok         *int     `json:"stok" validate:"omitempty,gte=0"`
	MinimumOrder *int     `json:"minimum_order" validate:"omitempty,gte=0"`
	Satuan       string   `json:"satuan" validate:"omitempty,max=20"`
	HargaSatuan  *float64 `json:"harga_satuan" validate:"omitempty,gte=0"`
	Keterangan   string   `json:"keterangan" validate:"omitempty"`
}

// RecordSparePartUsageRequest adalah body pencatatan penggunaan suku cadang
// pada maintenance log. Stok akan dikurangi otomatis sebesar Jumlah.
type RecordSparePartUsageRequest struct {
	SparePartID string `json:"spare_part_id" validate:"required,uuid4"`
	Jumlah      int    `json:"jumlah" validate:"required,gt=0"`
	Catatan     string `json:"catatan" validate:"omitempty"`
}

// GetAllSpareParts mengambil daftar suku cadang terpaginasi dengan filter
// kategori dan pencarian nama/SKU (Admin & Teknisi yang sudah login).
// @Summary      Ambil Semua Suku Cadang
// @Description  Mendapatkan daftar suku cadang terpaginasi. Mendukung filter kategori (kategori), pencarian nama/SKU (search), dan sorting.
// @Tags         Suku Cadang
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        kategori query string false "Filter berdasarkan kategori (Kabel, Konektor, Power Supply, Lensa, Storage, Lainnya)"
// @Param        search query string false "Pencarian berdasarkan nama atau SKU"
// @Param        page query integer false "Nomor Halaman (default: 1)"
// @Param        limit query integer false "Jumlah data per halaman (default: 10)"
// @Param        sort_by query string false "Kolom pengurutan (default: created_at)"
// @Param        order query string false "Arah pengurutan: asc, desc (default: desc)"
// @Success      200 {object} map[string]interface{} "Daftar suku cadang terpaginasi"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      500 {object} map[string]interface{} "Gagal mengambil daftar suku cadang"
// @Router       /spare-parts [get]
func GetAllSpareParts(c *fiber.Ctx) error {
	var items []models.SparePart
	query := config.DB.Model(&models.SparePart{})

	// Filter kategori (opsional)
	if kategori := strings.TrimSpace(c.Query("kategori")); kategori != "" {
		query = query.Where("kategori = ?", kategori)
	}

	// Pencarian nama atau SKU (opsional)
	if search := strings.TrimSpace(c.Query("search")); search != "" {
		like := "%" + search + "%"
		query = query.Where("nama ILIKE ? OR sku ILIKE ?", like, like)
	}

	allowedSort := []string{"id", "nama", "sku", "kategori", "stok", "minimum_order", "satuan", "harga_satuan", "created_at", "updated_at"}
	limit, page, sort := utils.GetPaginationParams(c, allowedSort)

	var totalRows int64
	if err := query.Count(&totalRows).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghitung total suku cadang",
		})
	}

	offset := (page - 1) * limit
	if err := query.Offset(offset).Limit(limit).Order(sort).Find(&items).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal mengambil daftar suku cadang",
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"rows":       items,
			"pagination": utils.CalculatePagination(totalRows, page, limit),
		},
	})
}

// GetSparePartByID mengambil detail satu suku cadang berdasarkan UUID-nya.
// @Summary      Ambil Suku Cadang Detail
// @Description  Mendapatkan detail satu suku cadang berdasarkan UUID-nya.
// @Tags         Suku Cadang
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Suku Cadang UUID"
// @Success      200 {object} map[string]interface{} "Data suku cadang ditemukan"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      404 {object} map[string]interface{} "Suku cadang tidak ditemukan"
// @Router       /spare-parts/{id} [get]
func GetSparePartByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID suku cadang tidak valid",
		})
	}

	var item models.SparePart
	if err := config.DB.First(&item, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Suku cadang tidak ditemukan",
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   item,
	})
}

// CreateSparePart membuat data suku cadang baru (Admin saja).
// @Summary      Tambah Suku Cadang
// @Description  Membuat record master suku cadang baru. SKU wajib unik.
// @Tags         Suku Cadang
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body CreateSparePartRequest true "Data Suku Cadang Baru"
// @Success      201 {object} map[string]interface{} "Suku cadang berhasil disimpan"
// @Failure      400 {object} map[string]interface{} "Format input salah, validasi gagal, atau SKU duplikat"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Failure      500 {object} map[string]interface{} "Gagal menyimpan suku cadang ke database"
// @Router       /spare-parts [post]
func CreateSparePart(c *fiber.Ctx) error {
	var req CreateSparePartRequest
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

	// Cek keunikan SKU (termasuk yang di-soft-delete).
	var existing models.SparePart
	if err := config.DB.Unscoped().Where("sku = ?", req.SKU).First(&existing).Error; err == nil {
		if existing.DeletedAt.Valid {
			// Hapus permanen data soft-deleted agar tidak memicu error unique constraint.
			if err := config.DB.Unscoped().Delete(&existing).Error; err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"status":  "error",
					"message": "Gagal membersihkan data suku cadang usang di database",
				})
			}
		} else {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "error",
				"message": "SKU sudah digunakan oleh suku cadang lain",
			})
		}
	}

	satuan := req.Satuan
	if satuan == "" {
		satuan = "pcs"
	}

	item := models.SparePart{
		Nama:         req.Nama,
		SKU:          req.SKU,
		Kategori:     req.Kategori,
		Stok:         req.Stok,
		MinimumOrder: req.MinimumOrder,
		Satuan:       satuan,
		HargaSatuan:  req.HargaSatuan,
		Keterangan:   req.Keterangan,
	}

	if err := config.DB.Create(&item).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menyimpan suku cadang ke database",
		})
	}

	utils.LogAudit(config.DB, c, "CREATE_SPARE_PART", "INVENTORY",
		"Membuat suku cadang baru: "+item.Nama+" (SKU: "+item.SKU+")", nil, "", "")

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status": "success",
		"data":   item,
	})
}

// UpdateSparePart memperbarui data suku cadang (Admin saja).
// @Summary      Update Suku Cadang
// @Description  Memperbarui data master suku cadang. SKU bila diubah wajib tetap unik.
// @Tags         Suku Cadang
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Suku Cadang UUID"
// @Param        request body UpdateSparePartRequest true "Data Update Suku Cadang"
// @Success      200 {object} map[string]interface{} "Suku cadang berhasil diperbarui"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid, format body salah, atau SKU duplikat"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Failure      404 {object} map[string]interface{} "Suku cadang tidak ditemukan"
// @Failure      500 {object} map[string]interface{} "Gagal memperbarui suku cadang"
// @Router       /spare-parts/{id} [put]
func UpdateSparePart(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID suku cadang tidak valid",
		})
	}

	var item models.SparePart
	if err := config.DB.First(&item, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Suku cadang tidak ditemukan",
		})
	}

	var req UpdateSparePartRequest
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

	// Jika SKU diubah, pastikan tidak bertabrakan dengan suku cadang lain
	// (termasuk yang di-soft-delete).
	if req.SKU != "" && req.SKU != item.SKU {
		var dup models.SparePart
		if err := config.DB.Unscoped().Where("sku = ?", req.SKU).First(&dup).Error; err == nil {
			if dup.DeletedAt.Valid {
				if err := config.DB.Unscoped().Delete(&dup).Error; err != nil {
					return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
						"status":  "error",
						"message": "Gagal membersihkan data suku cadang usang di database",
					})
				}
			} else {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"status":  "error",
					"message": "SKU sudah digunakan oleh suku cadang lain",
				})
			}
		}
	}

	if req.Nama != "" {
		item.Nama = req.Nama
	}
	if req.SKU != "" {
		item.SKU = req.SKU
	}
	if req.Kategori != "" {
		item.Kategori = req.Kategori
	}
	if req.Stok != nil {
		item.Stok = *req.Stok
	}
	if req.MinimumOrder != nil {
		item.MinimumOrder = *req.MinimumOrder
	}
	if req.Satuan != "" {
		item.Satuan = req.Satuan
	}
	if req.HargaSatuan != nil {
		item.HargaSatuan = *req.HargaSatuan
	}
	if req.Keterangan != "" {
		item.Keterangan = req.Keterangan
	}

	if err := config.DB.Save(&item).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal memperbarui suku cadang",
		})
	}

	utils.LogAudit(config.DB, c, "UPDATE_SPARE_PART", "INVENTORY",
		"Memperbarui suku cadang: "+item.Nama+" (SKU: "+item.SKU+")", nil, "", "")

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   item,
	})
}

// DeleteSparePart menghapus suku cadang secara soft-delete (Admin saja).
// @Summary      Hapus Suku Cadang
// @Description  Menghapus suku cadang (soft-delete). Riwayat penggunaan pada maintenance log tetap tersimpan.
// @Tags         Suku Cadang
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Suku Cadang UUID"
// @Success      200 {object} map[string]interface{} "Suku cadang berhasil dihapus"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Failure      404 {object} map[string]interface{} "Suku cadang tidak ditemukan"
// @Failure      500 {object} map[string]interface{} "Gagal menghapus suku cadang"
// @Router       /spare-parts/{id} [delete]
func DeleteSparePart(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID suku cadang tidak valid",
		})
	}

	var item models.SparePart
	if err := config.DB.First(&item, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Suku cadang tidak ditemukan",
		})
	}

	if err := config.DB.Delete(&item).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghapus suku cadang",
		})
	}

	utils.LogAudit(config.DB, c, "DELETE_SPARE_PART", "INVENTORY",
		"Menghapus suku cadang: "+item.Nama+" (SKU: "+item.SKU+")", nil, "", "")

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Suku cadang berhasil dihapus",
	})
}

// RecordSparePartUsage mencatat penggunaan suku cadang pada maintenance log
// dan mengurangi stok otomatis. Maintenance log wajib berstatus "completed".
// Bila stok setelah pengurangan jatuh di bawah minimum_order, event STOCK_LOW
// di-broadcast ke seluruh admin.
// @Summary      Catat Penggunaan Suku Cadang
// @Description  Mencatat penggunaan suku cadang pada maintenance log berstatus completed. Stok suku cadang dikurangi otomatis dalam satu transaksi. Bila stok < minimum_order setelah pengurangan, notifikasi/event STOCK_LOW dikirim ke admin.
// @Tags         Suku Cadang
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Maintenance Log UUID"
// @Param        request body RecordSparePartUsageRequest true "Data Penggunaan Suku Cadang"
// @Success      201 {object} map[string]interface{} "Penggunaan suku cadang berhasil dicatat"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid, validasi gagal, atau stok tidak mencukupi"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      404 {object} map[string]interface{} "Maintenance log atau suku cadang tidak ditemukan"
// @Failure      409 {object} map[string]interface{} "Maintenance log belum berstatus completed"
// @Failure      500 {object} map[string]interface{} "Gagal menyimpan penggunaan suku cadang"
// @Router       /maintenance-logs/{id}/spare-parts [post]
func RecordSparePartUsage(c *fiber.Ctx) error {
	logID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID log pemeliharaan tidak valid",
		})
	}

	var req RecordSparePartUsageRequest
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

	sparePartID, err := uuid.Parse(req.SparePartID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID suku cadang tidak valid",
		})
	}

	// Verifikasi maintenance log ada & berstatus completed.
	var logEntry models.MaintenanceLog
	if err := config.DB.First(&logEntry, "id = ?", logID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Log pemeliharaan tidak ditemukan",
		})
	}
	if logEntry.Status != "completed" {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"status":  "error",
			"message": "Penggunaan suku cadang hanya dapat dicatat pada log pemeliharaan berstatus selesai (completed)",
		})
	}

	// Kurangi stok + buat catatan penggunaan dalam satu transaksi.
	var usage models.SparePartUsage
	var sparePart models.SparePart
	stockLow := false

	err = config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&sparePart, "id = ?", sparePartID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fiber.NewError(fiber.StatusNotFound, "Suku cadang tidak ditemukan")
			}
			return err
		}

		if req.Jumlah > sparePart.Stok {
			return fiber.NewError(fiber.StatusBadRequest,
				fmt.Sprintf("Stok tidak mencukupi. Tersedia %d %s", sparePart.Stok, sparePart.Satuan))
		}

		// Kurangi stok secara atomik.
		newStok := sparePart.Stok - req.Jumlah
		if err := tx.Model(&models.SparePart{}).Where("id = ?", sparePart.ID).
			Update("stok", newStok).Error; err != nil {
			return err
		}

		// Buat catatan penggunaan.
		usage = models.SparePartUsage{
			MaintenanceLogID: logID,
			SparePartID:      sparePartID,
			Jumlah:           req.Jumlah,
			Catatan:          req.Catatan,
		}
		if err := tx.Create(&usage).Error; err != nil {
			return err
		}

		// Deteksi stok rendah setelah pengurangan.
		if newStok < sparePart.MinimumOrder {
			stockLow = true
		}
		return nil
	})

	if err != nil {
		// Konversi fiber.Error ke status & pesan yang sesuai.
		if fe, ok := err.(*fiber.Error); ok {
			return c.Status(fe.Code).JSON(fiber.Map{
				"status":  "error",
				"message": fe.Message,
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menyimpan penggunaan suku cadang",
		})
	}

	utils.LogAudit(config.DB, c, "RECORD_SPARE_PART_USAGE", "INVENTORY",
		fmt.Sprintf("Mencatat penggunaan suku cadang %s (%s) sebanyak %d pada maintenance log", sparePart.Nama, sparePart.SKU, req.Jumlah),
		&logID, "", "")

	// Broadcast event stok rendah ke seluruh admin bila ambang minimum terlampaui.
	if stockLow {
		utils.BroadcastToRole("admin", EventStockLow, fiber.Map{
			"spare_part_id": sparePart.ID,
			"nama":          sparePart.Nama,
			"sku":           sparePart.SKU,
			"stok":          sparePart.Stok - req.Jumlah,
			"minimum_order": sparePart.MinimumOrder,
		})
	}

	// Muat ulang dengan relasi suku cadang untuk respon.
	config.DB.Preload("SparePart").First(&usage, usage.ID)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status": "success",
		"data":   usage,
	})
}

// GetSparePartUsageByLog mengambil daftar penggunaan suku cadang pada sebuah
// maintenance log tertentu.
// @Summary      Ambil Penggunaan Suku Cadang per Log
// @Description  Mendapatkan daftar suku cadang yang digunakan pada maintenance log tertentu beserta jumlah dan catatan.
// @Tags         Suku Cadang
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Maintenance Log UUID"
// @Success      200 {object} map[string]interface{} "Daftar penggunaan suku cadang"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      500 {object} map[string]interface{} "Gagal mengambil daftar penggunaan suku cadang"
// @Router       /maintenance-logs/{id}/spare-parts [get]
func GetSparePartUsageByLog(c *fiber.Ctx) error {
	logID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID log pemeliharaan tidak valid",
		})
	}

	var usages []models.SparePartUsage
	if err := config.DB.Preload("SparePart").
		Where("maintenance_log_id = ?", logID).
		Order("created_at asc").
		Find(&usages).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal mengambil daftar penggunaan suku cadang",
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   usages,
	})
}

// GetLowStockSpareParts mengambil daftar suku cadang yang stoknya berada di
// bawah minimum_order (untuk widget dashboard).
// @Summary      Ambil Suku Cadang Stok Rendah
// @Description  Mendapatkan daftar suku cadang yang stoknya lebih kecil dari minimum_order, diurutkan dari stok paling sedikit.
// @Tags         Suku Cadang
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "Daftar suku cadang stok rendah"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      500 {object} map[string]interface{} "Gagal mengambil daftar suku cadang stok rendah"
// @Router       /spare-parts/low-stock [get]
func GetLowStockSpareParts(c *fiber.Ctx) error {
	var items []models.SparePart
	// Stok di bawah minimum_order; urutkan dari stok paling sedikit.
	if err := config.DB.Where("stok < minimum_order").
		Order("stok asc").
		Limit(100).
		Find(&items).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal mengambil daftar suku cadang stok rendah",
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   items,
	})
}
