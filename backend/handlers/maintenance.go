package handlers

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os"
	"strings"
	"time"

	"sigap-backend/config"
	"sigap-backend/models"
	"sigap-backend/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CreateMaintenanceLogRequest struct {
	CCTVID        string `json:"cctv_id" validate:"required,uuid4"`
	ActionTaken   string `json:"action_taken" validate:"required,min=5"`
	Notes         string `json:"notes" validate:"omitempty"`
	// Status hanya boleh bernilai "scheduled" saat pembuatan — alur status
	// selanjutnya dikelola sistem: assign teknisi (-> ongoing) dan penyelesaian
	// checklist (-> completed/failed), tidak bisa diubah manual lewat API.
	Status        string `json:"status" validate:"omitempty,oneof=scheduled"`
	ScheduledDate string `json:"scheduled_date" validate:"omitempty"` // YYYY-MM-DD HH:MM:SS or RFC3339
	ImageUrl      string `json:"image_url" validate:"omitempty"`      // URL dari endpoint /api/upload
}

type UpdateMaintenanceLogRequest struct {
	CCTVID        string `json:"cctv_id" validate:"omitempty,uuid4"`
	ActionTaken   string `json:"action_taken" validate:"omitempty,min=5"`
	Notes         string `json:"notes" validate:"omitempty"`
	ScheduledDate string `json:"scheduled_date" validate:"omitempty"` // YYYY-MM-DD HH:MM:SS or RFC3339
	ImageUrl      string `json:"image_url" validate:"omitempty"`      // URL baru dari /api/upload (opsional)
}

// AssignMaintenanceLogRequest menugaskan log terjadwal ke seorang teknisi.
type AssignMaintenanceLogRequest struct {
	TechnicianID string `json:"technician_id" validate:"required,uuid4"`
}

// CompleteMaintenanceLogRequest berisi hasil penyelesaian tugas terjadwal.
// FieldCheckItem & FieldImageItem dipakai ulang dari modul field-maintenance.
type CompleteMaintenanceLogRequest struct {
	Notes  string           `json:"notes" validate:"omitempty,max=1000"`
	Checks []FieldCheckItem `json:"checks" validate:"required,min=1,dive"`
	Images []FieldImageItem `json:"images" validate:"required,min=1,max=5,dive"`
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

	// Cek apakah cctv_id valid — CCTV adalah satu-satunya sumber kebenaran aset
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

	// Log baru selalu dibuat tanpa penugasan (TechnicianID nil) dengan status
	// "scheduled". Teknisi ditautkan kemudian melalui endpoint assign.
	logEntry := models.MaintenanceLog{
		CCTVID:        cctvUUID,
		ActionTaken:   req.ActionTaken,
		Notes:         req.Notes,
		Status:        "scheduled",
		ScheduledDate: schedDate,
		ImageUrl:      req.ImageUrl,
	}

	if err := config.DB.Create(&logEntry).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menyimpan log pemeliharaan",
		})
	}

	utils.LogAudit(config.DB, c, "CREATE_MAINTENANCE", "MAINTENANCE", "Mencatat pemeliharaan baru: "+logEntry.ActionTaken+" (Status: "+logEntry.Status+")", nil, "", "")

	// Muat ulang dengan data relasi (CCTV & User) untuk respon JSON
	config.DB.Preload("CCTV").Preload("User").First(&logEntry, logEntry.ID)
	utils.BroadcastEvent("MAINTENANCE_CREATED", logEntry)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status": "success",
		"data":   logEntry,
	})
}

// GetAllMaintenanceLogs mengambil semua log pemeliharaan (Admin & Teknisi)
// @Summary      Ambil Semua Log Pemeliharaan
// @Description  Mendapatkan list riwayat pemeliharaan CCTV terpaginasi dengan filter cctv_id, status, pencarian, dan sorting.
// @Tags         Log Pemeliharaan
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        cctv_id query string false "Filter berdasarkan UUID CCTV"
// @Param        status query string false "Filter berdasarkan status log"
// @Param        start_date query string false "Filter rentang awal (YYYY-MM-DD)"
// @Param        end_date query string false "Filter rentang akhir (YYYY-MM-DD)"
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
	query := config.DB.Model(&models.MaintenanceLog{}).Preload("CCTV").Preload("User")

	// SECURITY: Teknisi hanya dapat melihat log miliknya (lewat penugasan)
	query = applyRoleScope(c, query)

	// Filter berdasarkan cctv_id (dukung alias legacy asset_id dari QR lama)
	if cctvID := c.Query("cctv_id"); cctvID != "" {
		cctvUUID, err := uuid.Parse(cctvID)
		if err == nil {
			query = query.Where("cctv_id = ?", cctvUUID)
		}
	} else if assetID := c.Query("asset_id"); assetID != "" {
		assetUUID, err := uuid.Parse(assetID)
		if err == nil {
			query = query.Where("cctv_id = ?", assetUUID)
		}
	}

	// Filter berdasarkan status
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
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

	// Pencarian teks (berdasarkan tindakan atau catatan)
	if search := c.Query("search"); search != "" {
		query = query.Where("action_taken ILIKE ? OR notes ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// Ambil parameter paginasi & sorting (dengan whitelist keamanan)
	allowedSort := []string{"id", "cctv_id", "technician_id", "action_taken", "notes", "status", "scheduled_date", "completed_at", "created_at", "updated_at"}
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
	if err := config.DB.Preload("CCTV").Preload("User").First(&logEntry, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Log pemeliharaan tidak ditemukan",
		})
	}

	// SECURITY: Teknisi hanya boleh melihat detail log miliknya sendiri
	userRole, _ := c.Locals("role").(string)
	if userRole != "admin" {
		techIDStr, ok := c.Locals("user_id").(string)
		if !ok || techIDStr == "" || logEntry.TechnicianID == nil || techIDStr != logEntry.TechnicianID.String() {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"status":  "error",
				"message": "Akses ditolak: Anda hanya dapat melihat log pemeliharaan milik Anda sendiri",
			})
		}
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

	// SECURITY: Ownership check — Teknisi hanya boleh mengubah log miliknya
	// sendiri. Admin diperbolehkan mengubah semua log.
	userRole, _ := c.Locals("role").(string)
	if userRole != "admin" {
		techIDStr, ok := c.Locals("user_id").(string)
		if !ok || techIDStr == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "Sesi tidak valid, Teknisi tidak ditemukan",
			})
		}
		if logEntry.TechnicianID == nil || techIDStr != logEntry.TechnicianID.String() {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"status":  "error",
				"message": "Akses ditolak: Anda hanya dapat mengubah log pemeliharaan milik Anda sendiri",
			})
		}
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
	if req.CCTVID != "" {
		cctvUUID, err := uuid.Parse(req.CCTVID)
		if err == nil {
			var cctv models.CCTV
			if err := config.DB.First(&cctv, "id = ?", cctvUUID).Error; err == nil {
				logEntry.CCTVID = cctvUUID
			}
		}
	}
	if req.ActionTaken != "" {
		logEntry.ActionTaken = req.ActionTaken
	}
	if req.Notes != "" {
		logEntry.Notes = req.Notes
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

	// Jika image_url baru diberikan, hapus file lama sebelum mengganti
	if req.ImageUrl != "" && req.ImageUrl != logEntry.ImageUrl {
		if logEntry.ImageUrl != "" {
			oldPath := "." + logEntry.ImageUrl // /uploads/xxx.jpg -> ./uploads/xxx.jpg
			_ = os.Remove(oldPath)             // Abaikan error jika file sudah tidak ada
		}
		logEntry.ImageUrl = req.ImageUrl
	}

	if err := config.DB.Save(&logEntry).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal memperbarui log pemeliharaan",
		})
	}

	utils.LogAudit(config.DB, c, "UPDATE_MAINTENANCE", "MAINTENANCE", "Memperbarui log pemeliharaan: "+logEntry.ActionTaken+" (Status: "+logEntry.Status+")", nil, "", "")

	config.DB.Preload("CCTV").Preload("User").First(&logEntry, logEntry.ID)
	utils.BroadcastEvent("MAINTENANCE_UPDATED", logEntry)

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   logEntry,
	})
}

// maintenanceCheckTotal adalah jumlah item checklist wajib (9 tahap) yang harus
// dilaporkan teknisi untuk menyelesaikan tugas. Status menjadi "completed"
// hanya bila 9/9 item berstatus ok; selain itu "failed".
const maintenanceCheckTotal = 9

// AssignMaintenanceLog menugaskan log terjadwal ke seorang teknisi (Admin saja)
// @Summary      Tugaskan Log ke Teknisi
// @Description  Menugaskan log pemeliharaan berstatus "scheduled" ke teknisi berperan "teknisi". Status log otomatis berubah menjadi "ongoing".
// @Tags         Log Pemeliharaan
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Log UUID"
// @Param        request body AssignMaintenanceLogRequest true "ID Teknisi Tujuan"
// @Success      200 {object} map[string]interface{} "Log berhasil ditugaskan"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid, log tidak terjadwal, atau format body salah"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Failure      404 {object} map[string]interface{} "Log atau teknisi tidak ditemukan"
// @Router       /maintenance-logs/{id}/assign [put]
func AssignMaintenanceLog(c *fiber.Ctx) error {
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

	// Hanya log terjadwal (belum ditugaskan) yang dapat di-assign
	if logEntry.Status != "scheduled" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Hanya log berstatus terjadwal yang dapat ditugaskan ke teknisi",
		})
	}

	var req AssignMaintenanceLogRequest
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

	techUUID, err := uuid.Parse(req.TechnicianID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID teknisi tidak valid",
		})
	}

	// Pastikan user tujuan benar-benar berperan teknisi
	var techUser models.User
	if err := config.DB.First(&techUser, "id = ?", techUUID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Teknisi tidak ditemukan",
		})
	}
	if techUser.Role != "teknisi" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "User tujuan bukan berperan teknisi",
		})
	}

	now := time.Now()
	logEntry.TechnicianID = &techUUID
	logEntry.Status = "ongoing"
	logEntry.UpdatedAt = now

	if err := config.DB.Save(&logEntry).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menugaskan log pemeliharaan",
		})
	}

	utils.LogAudit(config.DB, c, "ASSIGN_MAINTENANCE", "MAINTENANCE", "Menugaskan log pemeliharaan ke teknisi: "+techUser.FullName+" ("+logEntry.ActionTaken+")", nil, "", "")

	// Notifikasi in-app ke teknisi yang ditugaskan (Fase 5.1)
	refID := logEntry.ID
	utils.NotifyUser(config.DB, techUUID, models.NotificationTypeAssigned,
		"Tugas baru ditugaskan",
		"Tugas \""+logEntry.ActionTaken+"\" ditugaskan kepada Anda. Silakan kerjakan sesuai jadwal.",
		"maintenance-log", &refID)

	config.DB.Preload("CCTV").Preload("User").First(&logEntry, logEntry.ID)
	utils.BroadcastEvent("MAINTENANCE_UPDATED", logEntry)

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   logEntry,
	})
}

// CompleteMaintenanceLog menyelesaikan tugas terjadwal lewat checklist (Admin & Teknisi)
// @Summary      Selesaikan Tugas Pemeliharaan
// @Description  Teknisi melaporkan hasil pengecekan 9 tahap checklist beserta foto bukti. Status menjadi "completed" bila 9/9 item "ok", selain itu "failed". Teknisi wajib menyelesaikan tugas yang ditugaskan — tidak ada mekanisme penolakan.
// @Tags         Log Pemeliharaan
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Log UUID"
// @Param        request body CompleteMaintenanceLogRequest true "Hasil Checklist & Foto Bukti"
// @Success      200 {object} map[string]interface{} "Tugas berhasil diselesaikan"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid, tugas bukan berstatus dalam proses, atau format body salah"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: bukan teknisi yang ditugaskan"
// @Failure      404 {object} map[string]interface{} "Log tidak ditemukan"
// @Failure      500 {object} map[string]interface{} "Gagal menyimpan hasil pengecekan"
// @Router       /maintenance-logs/{id}/complete [post]
func CompleteMaintenanceLog(c *fiber.Ctx) error {
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

	// SECURITY: Teknisi wajib menyelesaikan tugasnya sendiri; admin diperbolehkan
	userRole, _ := c.Locals("role").(string)
	if userRole != "admin" {
		techIDStr, ok := c.Locals("user_id").(string)
		if !ok || techIDStr == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "Sesi tidak valid, Teknisi tidak ditemukan",
			})
		}
		if logEntry.TechnicianID == nil || techIDStr != logEntry.TechnicianID.String() {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"status":  "error",
				"message": "Akses ditolak: Anda hanya dapat menyelesaikan tugas yang ditugaskan kepada Anda",
			})
		}
	}

	// Hanya tugas yang sedang dikerjakan (ongoing) yang dapat diselesaikan
	if logEntry.Status != "ongoing" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Hanya tugas berstatus dalam proses yang dapat diselesaikan",
		})
	}

	var req CompleteMaintenanceLogRequest
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

	// Status akhir: completed hanya bila 9/9 item checklist berstatus ok
	allOk := true
	for _, chk := range req.Checks {
		if chk.Result != "ok" {
			allOk = false
			break
		}
	}
	newStatus := "failed"
	if allOk && len(req.Checks) == maintenanceCheckTotal {
		newStatus = "completed"
	}

	now := time.Now()
	overallStatus := "ok"
	if !allOk {
		overallStatus = "attention"
	}

	// Teknisi pelaksana: teknisi yang ditugaskan (atau admin sebagai pengganti)
	techUUID := uuid.Nil
	if logEntry.TechnicianID != nil {
		techUUID = *logEntry.TechnicianID
	} else if tidStr, ok := c.Locals("user_id").(string); ok {
		if tid, err := uuid.Parse(tidStr); err == nil {
			techUUID = tid
		}
	}

	err = config.DB.Transaction(func(tx *gorm.DB) error {
		field := models.FieldMaintenance{
			CCTVID:          logEntry.CCTVID,
			TechnicianID:    techUUID,
			MaintenanceLogID: &id,
			OverallStatus:   overallStatus,
			Notes:           req.Notes,
			CompletedAt:     &now,
		}
		if err := tx.Create(&field).Error; err != nil {
			return err
		}

		for _, chk := range req.Checks {
			item := models.FieldMaintenanceCheck{
				FieldMaintenanceID: field.ID,
				Name:               chk.Name,
				Result:             chk.Result,
				Note:               chk.Note,
			}
			if err := tx.Create(&item).Error; err != nil {
				return err
			}
		}

		for _, img := range req.Images {
			image := models.FieldMaintenanceImage{
				FieldMaintenanceID: field.ID,
				ImageUrl:           img.ImageUrl,
			}
			if err := tx.Create(&image).Error; err != nil {
				return err
			}
		}

		// Tautkan ID hasil pengecekan ke log & perbarui status tugas
		logEntry.MaintenanceResultID = &field.ID
		logEntry.Status = newStatus
		logEntry.CompletedAt = &now
		return tx.Save(&logEntry).Error
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menyimpan hasil pengecekan",
		})
	}

	utils.LogAudit(config.DB, c, "COMPLETE_MAINTENANCE", "MAINTENANCE", "Menyelesaikan tugas pemeliharaan: "+logEntry.ActionTaken+" (Status: "+newStatus+")", nil, "", "")

	// Notifikasi in-app ke seluruh admin: hasil selesai (9/9 ok) atau perlu
	// tindak lanjut (failed karena ada item tidak ok) — Fase 5.1.
	label := "Selesai"
	if newStatus == "failed" {
		label = "Perlu Tindak Lanjut"
	}
	refID := logEntry.ID
	utils.NotifyRole(config.DB, "admin", models.NotificationTypeCompleted,
		"Tugas selesai: "+label,
		"Tugas \""+logEntry.ActionTaken+"\" selesai dikerjakan dengan status \""+label+"\".",
		"maintenance-log", &refID)

	config.DB.Preload("CCTV").Preload("User").First(&logEntry, logEntry.ID)
	utils.BroadcastEvent("MAINTENANCE_UPDATED", logEntry)

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

	// AUTO-DELETE: Hapus file fisik jika ada image_url sebelum menghapus record DB
	if logEntry.ImageUrl != "" {
		// Konversi URL path (/uploads/xxx.jpg) ke path filesystem (./uploads/xxx.jpg)
		filePath := "." + logEntry.ImageUrl
		// Keamanan: pastikan path tidak keluar dari direktori uploads
		if strings.HasPrefix(logEntry.ImageUrl, "/uploads/") {
			if removeErr := os.Remove(filePath); removeErr != nil && !os.IsNotExist(removeErr) {
				// Log error tapi lanjutkan proses penghapusan record DB
				utils.Log.Warnf("⚠️ Gagal menghapus file fisik lampiran: %v", removeErr)
			} else {
				utils.Log.Infof("🗑️ File lampiran foto berhasil dihapus: %s", filePath)
			}
		}
	}

	if err := config.DB.Delete(&logEntry).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghapus log pemeliharaan",
		})
	}

	utils.LogAudit(config.DB, c, "DELETE_MAINTENANCE", "MAINTENANCE", "Menghapus log pemeliharaan: "+logEntry.ActionTaken, nil, "", "")

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Log pemeliharaan beserta lampiran foto berhasil dihapus",
	})
}

// ExportMaintenanceLogs memproduksi file CSV log pemeliharaan
// @Summary      Ekspor Log Pemeliharaan ke CSV
// @Description  Mengunduh seluruh riwayat pemeliharaan CCTV terfilter (tanpa limit halaman) dalam format file CSV.
// @Tags         Log Pemeliharaan
// @Produce      text/csv
// @Security     BearerAuth
// @Param        cctv_id query string false "Filter berdasarkan UUID CCTV"
// @Param        status query string false "Filter berdasarkan status log"
// @Param        start_date query string false "Filter rentang awal (YYYY-MM-DD)"
// @Param        end_date query string false "Filter rentang akhir (YYYY-MM-DD)"
// @Param        search query string false "Cari tindakan (action_taken) atau catatan (notes)"
// @Success      200 {file} string "File CSV Log Pemeliharaan"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      500 {object} map[string]interface{} "Gagal mengekspor data"
// @Router       /maintenance-logs/export [get]
func ExportMaintenanceLogs(c *fiber.Ctx) error {
	var logs []models.MaintenanceLog
	query := config.DB.Model(&models.MaintenanceLog{}).Preload("CCTV").Preload("User")

	// SECURITY (Fase 3.9): Teknisi hanya boleh mengekspor data yang relevan
	// untuk dirinya sendiri; admin berhak mengekspor seluruh data.
	query = applyRoleScope(c, query)

	// Filter berdasarkan cctv_id (dukung alias legacy asset_id)
	if cctvID := c.Query("cctv_id"); cctvID != "" {
		cctvUUID, err := uuid.Parse(cctvID)
		if err == nil {
			query = query.Where("cctv_id = ?", cctvUUID)
		}
	} else if assetID := c.Query("asset_id"); assetID != "" {
		assetUUID, err := uuid.Parse(assetID)
		if err == nil {
			query = query.Where("cctv_id = ?", assetUUID)
		}
	}

	// Filter berdasarkan status
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
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
		query = query.Where("action_taken ILIKE ? OR notes ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// Urutkan berdasarkan created_at desc agar data terbaru ada di paling atas
	if err := query.Order("created_at desc").Find(&logs).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal mengambil data log pemeliharaan untuk diekspor",
		})
	}

	// Tulis ke CSV Buffer
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	writer.Comma = ','

	// Header
	header := []string{"ID", "CCTV ID", "Nama CCTV", "Kode CCTV", "Technician ID", "Nama Teknisi", "Tindakan", "Catatan", "Status", "Tanggal Dijadwalkan", "Tanggal Selesai", "Tanggal Pembuatan"}
	if err := writer.Write(header); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menulis header CSV",
		})
	}

	// Baris data
	for _, logEntry := range logs {
		schedStr := ""
		if logEntry.ScheduledDate != nil {
			schedStr = logEntry.ScheduledDate.Format("2006-01-02 15:04:05")
		}
		compStr := ""
		if logEntry.CompletedAt != nil {
			compStr = logEntry.CompletedAt.Format("2006-01-02 15:04:05")
		}
		techIDStr := ""
		techName := ""
		if logEntry.TechnicianID != nil {
			techIDStr = logEntry.TechnicianID.String()
		}
		if logEntry.User != nil {
			techName = logEntry.User.FullName
		}

		row := []string{
			logEntry.ID.String(),
			logEntry.CCTVID.String(),
			logEntry.CCTV.Nama,
			logEntry.CCTV.KodeCCTV,
			techIDStr,
			techName,
			logEntry.ActionTaken,
			logEntry.Notes,
			logEntry.Status,
			schedStr,
			compStr,
			logEntry.CreatedAt.Format("2006-01-02 15:04:05"),
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

	// Mengatur headers untuk respon download file CSV
	filename := fmt.Sprintf("maintenance_logs_export_%s.csv", time.Now().Format("20060102_150405"))
	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	return c.Send(buf.Bytes())
}

// applyRoleScope membatasi query berdasarkan peran pengguna dari JWT.
// Admin dapat melihat/mengekspor seluruh data; selain admin (teknisi) hanya
// melihat/mengekspor log pemeliharaan miliknya sendiri.
func applyRoleScope(c *fiber.Ctx, query *gorm.DB) *gorm.DB {
	if role, _ := c.Locals("role").(string); role == "admin" {
		return query
	}
	techIDStr, _ := c.Locals("user_id").(string)
	techUUID, err := uuid.Parse(techIDStr)
	if err != nil {
		return query.Where("1 = 0")
	}
	return query.Where("technician_id = ?", techUUID)
}
