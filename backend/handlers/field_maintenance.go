package handlers

import (
	"strconv"
	"strings"
	"time"

	"sigap-backend/config"
	"sigap-backend/models"
	"sigap-backend/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// maxFieldChecks membatasi jumlah item checklist agar mencegah penyalahgunaan payload.
const maxFieldChecks = 30

// maxFieldImages membatasi jumlah foto bukti per catatan pengecekan.
const maxFieldImages = 5

type FieldCheckItem struct {
	Name   string `json:"name" validate:"required,min=3,max=150"`
	Result string `json:"result" validate:"required,oneof=ok not_ok"`
	Note   string `json:"note" validate:"omitempty,max=500"`
}

type FieldImageItem struct {
	ImageUrl string `json:"image_url" validate:"required,startswith=/api/uploads/"`
}

type CreateFieldMaintenanceRequest struct {
	CCTVID string           `json:"cctv_id" validate:"required,uuid4"`
	Notes  string           `json:"notes" validate:"omitempty,max=1000"`
	Checks []FieldCheckItem `json:"checks" validate:"required,min=1,dive"`
	Images []FieldImageItem `json:"images" validate:"required,min=1,max=5,dive"`
}

// CreateFieldMaintenance mencatat hasil pengecekan lapangan teknisi (Admin & Teknisi)
// @Summary      Catat Pengecekan Lapangan
// @Description  Membuat catatan pengecekan lapangan untuk aset CCTV dengan daftar checklist item yang wajib diisi dan minimal satu foto bukti. ID Teknisi dideteksi otomatis dari token JWT. Status CCTV ikut disinkronkan otomatis (konservatif): hasil "attention" → "pemeliharaan", hasil "ok" → "aktif"; status "rusak"/"nonaktif" tidak pernah ditimpa otomatis.
// @Tags         Pengecekan Lapangan
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body CreateFieldMaintenanceRequest true "Detail Pengecekan Lapangan"
// @Success      201 {object} map[string]interface{} "Catatan berhasil dibuat"
// @Failure      400 {object} map[string]interface{} "Format tidak valid, aset tidak ditemukan, data tidak lengkap"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Router       /field-maintenance [post]
func CreateFieldMaintenance(c *fiber.Ctx) error {
	var req CreateFieldMaintenanceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Format input tidak valid",
		})
	}

	// Batasan struktural tambahan (maks item & foto) di luar validator struct
	if len(req.Checks) == 0 || len(req.Checks) > maxFieldChecks {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Checklist wajib diisi 1–" + strconv.Itoa(maxFieldChecks) + " item",
		})
	}
	if len(req.Images) == 0 || len(req.Images) > maxFieldImages {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Minimal satu foto bukti wajib dilampirkan (maksimal " + strconv.Itoa(maxFieldImages) + " foto)",
		})
	}

	if errs := utils.ValidateStruct(&req); len(errs) > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Validasi input gagal",
			"errors":  errs,
		})
	}

	cctvUUID, err := uuid.Parse(req.CCTVID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "CCTV ID tidak valid",
		})
	}

	var cctv models.CCTV
	if err := config.DB.First(&cctv, "id = ?", cctvUUID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "CCTV tidak ditemukan",
		})
	}

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

	// Ringkasan: seluruh item ok → "ok", ada satu saja tidak ok → "attention"
	overall := "ok"
	for _, check := range req.Checks {
		if check.Result == "not_ok" {
			overall = "attention"
			break
		}
	}

	now := time.Now()
	entry := models.FieldMaintenance{
		CCTVID:        cctvUUID,
		TechnicianID:  techUUID,
		OverallStatus: overall,
		Notes:         strings.TrimSpace(req.Notes),
		CompletedAt:   &now,
	}

	// Simpan header + checklist + gambar dalam satu transaksi agar konsisten
	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		if createErr := tx.Create(&entry).Error; createErr != nil {
			return createErr
		}
		for _, check := range req.Checks {
			item := models.FieldMaintenanceCheck{
				FieldMaintenanceID: entry.ID,
				Name:               strings.TrimSpace(check.Name),
				Result:             check.Result,
				Note:               strings.TrimSpace(check.Note),
			}
			if createErr := tx.Create(&item).Error; createErr != nil {
				return createErr
			}
		}
		for _, img := range req.Images {
			image := models.FieldMaintenanceImage{
				FieldMaintenanceID: entry.ID,
				ImageUrl:           img.ImageUrl,
			}
			if createErr := tx.Create(&image).Error; createErr != nil {
				return createErr
			}
		}

		// Fase 5.2: sinkronisasi status CCTV otomatis (konservatif) dalam
		// transaksi yang sama — hasil "attention" memindahkan CCTV ke
		// "pemeliharaan"; hasil "ok" mengembalikannya ke "aktif".
		targetStatus := "aktif"
		syncReason := "pengecekan lapangan menunjukkan semua item OK"
		if overall == "attention" {
			targetStatus = "pemeliharaan"
			syncReason = "hasil pengecekan lapangan memerlukan perhatian"
		}
		if err := syncCCTVStatus(tx, c, cctvUUID, targetStatus, syncReason); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menyimpan catatan pengecekan lapangan",
		})
	}

	utils.LogAudit(config.DB, c, "CREATE_FIELD_MAINTENANCE", "MAINTENANCE", "Mencatat pengecekan lapangan CCTV "+cctv.KodeCCTV+" (Hasil: "+overall+")", nil, "", "")

	// Notifikasi in-app ke seluruh admin bila hasil pengecekan perlu perhatian
	// (ada item tidak ok) — Fase 5.1. Hasil "ok" tidak memicu notifikasi.
	if overall == "attention" {
		refID := entry.ID
		utils.NotifyRole(config.DB, "admin", models.NotificationTypeFieldCheck,
			"Pengecekan lapangan butuh perhatian",
			"Pengecekan lapangan CCTV "+cctv.KodeCCTV+" menunjukkan ada item yang tidak ok ("+cctv.LokasiArea+").",
			"field-maintenance", &refID)
	}

	// Muat ulang dengan relasi lengkap untuk respon JSON
	config.DB.Preload("CCTV").Preload("User").Preload("Checks").Preload("Images").First(&entry, entry.ID)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status": "success",
		"data":   entry,
	})
}

// GetAllFieldMaintenances mengambil daftar pengecekan lapangan (Admin: semua, Teknisi: miliknya)
// @Summary      Ambil Semua Pengecekan Lapangan
// @Description  Mendapatkan daftar catatan pengecekan lapangan terpaginasi. Admin melihat semua catatan, Teknisi hanya melihat miliknya sendiri.
// @Tags         Pengecekan Lapangan
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        cctv_id query string false "Filter berdasarkan UUID CCTV"
// @Param        status query string false "Filter berdasarkan hasil (ok / attention)"
// @Param        start_date query string false "Filter rentang awal (YYYY-MM-DD)"
// @Param        end_date query string false "Filter rentang akhir (YYYY-MM-DD)"
// @Param        search query string false "Cari catatan atau nama CCTV"
// @Param        page query integer false "Nomor Halaman (default: 1)"
// @Param        limit query integer false "Jumlah data per halaman (default: 10)"
// @Param        sort_by query string false "Kolom pengurutan (default: created_at)"
// @Param        order query string false "Arah pengurutan: asc, desc (default: desc)"
// @Success      200 {object} map[string]interface{} "Daftar pengecekan terpaginasi"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Router       /field-maintenance [get]
func GetAllFieldMaintenances(c *fiber.Ctx) error {
	var entries []models.FieldMaintenance
	query := config.DB.Model(&models.FieldMaintenance{}).
		Preload("CCTV").Preload("User").Preload("Checks").Preload("Images")

	// Role scope: teknisi hanya melihat catatan miliknya sendiri
	userRole, _ := c.Locals("role").(string)
	if userRole != "admin" {
		techIDStr, ok := c.Locals("user_id").(string)
		if !ok || techIDStr == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "Sesi tidak valid, Teknisi tidak ditemukan",
			})
		}
		query = query.Where("technician_id = ?", techIDStr)
	}

	if cctvID := c.Query("cctv_id"); cctvID != "" {
		cctvUUID, err := uuid.Parse(cctvID)
		if err == nil {
			query = query.Where("cctv_id = ?", cctvUUID)
		}
	}

	if status := c.Query("status"); status != "" {
		query = query.Where("overall_status = ?", status)
	}

	if start, end := utils.ParseDateRangeParams(c); start != nil || end != nil {
		if start != nil {
			query = query.Where("created_at >= ?", *start)
		}
		if end != nil {
			query = query.Where("created_at < ?", *end)
		}
	}

	if search := c.Query("search"); search != "" {
		query = query.Where("notes ILIKE ? OR cctv_id IN (SELECT id FROM cctvs WHERE nama ILIKE ? OR kode_cctv ILIKE ?)",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	allowedSort := []string{"id", "cctv_id", "technician_id", "overall_status", "created_at", "completed_at"}
	limit, page, sort := utils.GetPaginationParams(c, allowedSort)

	var totalRows int64
	if err := query.Count(&totalRows).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghitung total pengecekan lapangan",
		})
	}

	offset := (page - 1) * limit
	if err := query.Offset(offset).Limit(limit).Order(sort).Find(&entries).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal mengambil daftar pengecekan lapangan",
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"rows":       entries,
			"pagination": utils.CalculatePagination(totalRows, page, limit),
		},
	})
}

// GetFieldMaintenanceByID mengambil satu catatan pengecekan lapangan (Admin & Teknisi)
// @Summary      Ambil Pengecekan Lapangan Detail
// @Description  Mendapatkan detail satu catatan pengecekan lapangan berdasarkan UUID-nya.
// @Tags         Pengecekan Lapangan
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Catatan UUID"
// @Success      200 {object} map[string]interface{} "Data catatan ditemukan"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      404 {object} map[string]interface{} "Catatan tidak ditemukan"
// @Router       /field-maintenance/{id} [get]
func GetFieldMaintenanceByID(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID pengecekan lapangan tidak valid",
		})
	}

	var entry models.FieldMaintenance
	if err := config.DB.Preload("CCTV").Preload("User").Preload("Checks").Preload("Images").
		First(&entry, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Catatan pengecekan lapangan tidak ditemukan",
		})
	}

	// SECURITY: Teknisi hanya boleh membuka catatan miliknya sendiri
	userRole, _ := c.Locals("role").(string)
	if userRole != "admin" {
		techIDStr, ok := c.Locals("user_id").(string)
		if !ok || techIDStr != entry.TechnicianID.String() {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"status":  "error",
				"message": "Akses ditolak: Anda hanya dapat melihat catatan pengecekan milik Anda sendiri",
			})
		}
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   entry,
	})
}

// DeleteFieldMaintenance menghapus catatan pengecekan lapangan (Admin saja)
// @Summary      Hapus Pengecekan Lapangan
// @Description  Menghapus catatan pengecekan lapangan beserta checklist dan gambar buktinya (soft delete).
// @Tags         Pengecekan Lapangan
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Catatan UUID"
// @Success      200 {object} map[string]interface{} "Catatan berhasil dihapus"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      404 {object} map[string]interface{} "Catatan tidak ditemukan"
// @Router       /field-maintenance/{id} [delete]
func DeleteFieldMaintenance(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID pengecekan lapangan tidak valid",
		})
	}

	var entry models.FieldMaintenance
	if err := config.DB.First(&entry, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Catatan pengecekan lapangan tidak ditemukan",
		})
	}

	if err := config.DB.Delete(&entry).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghapus catatan pengecekan lapangan",
		})
	}

	utils.LogAudit(config.DB, c, "DELETE_FIELD_MAINTENANCE", "MAINTENANCE", "Menghapus catatan pengecekan lapangan CCTV "+entry.CCTVID.String(), nil, "", "")

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Catatan pengecekan lapangan berhasil dihapus",
	})
}
