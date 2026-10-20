package handlers

import (
	"time"

	"sigap-backend/config"
	"sigap-backend/models"
	"sigap-backend/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// CreateMaintenanceScheduleRequest adalah body pembuatan jadwal pemeliharaan
// berulang. TechnicianID opsional — bila diisi wajib berperan "teknisi".
type CreateMaintenanceScheduleRequest struct {
	CCTVID       string `json:"cctv_id" validate:"required,uuid4"`
	TechnicianID string `json:"technician_id" validate:"omitempty,uuid4"`
	ActionTaken  string `json:"action_taken" validate:"required,min=5"`
	Frequency    string `json:"frequency" validate:"required,oneof=daily weekly monthly"`
}

// UpdateMaintenanceScheduleRequest adalah body pembaruan jadwal. TechnicianID
// memakai pointer agar bisa dibedakan: nilai pointer non-nil "" berarti admin
// ingin melepas teknisi (di-set nil di database), non-nil UUID berarti mengganti
// teknisi, dan nil (field tidak dikirim) berarti tidak ada perubahan.
type UpdateMaintenanceScheduleRequest struct {
	CCTVID       string  `json:"cctv_id" validate:"omitempty,uuid4"`
	TechnicianID *string `json:"technician_id" validate:"omitempty"`
	ActionTaken  string  `json:"action_taken" validate:"omitempty,min=5"`
	Frequency    string  `json:"frequency" validate:"omitempty,oneof=daily weekly monthly"`
}

// nextRunForFrequency menghitung NextRunAt berikutnya dari titik waktu tertentu
// sesuai frekuensi jadwal. Dipakai saat pembuatan jadwal, saat frekuensi diubah
// (reset jadwal berikutnya), dan oleh generator untuk menggeser jadwal.
func nextRunForFrequency(freq string, from time.Time) time.Time {
	switch freq {
	case models.FrequencyWeekly:
		return from.AddDate(0, 0, 7)
	case models.FrequencyMonthly:
		return from.AddDate(0, 1, 0)
	case models.FrequencyDaily:
		fallthrough
	default:
		// Fallback aman: jika frekuensi tidak dikenal (seharusnya sudah tertahan
		// validasi oneof), perlakukan sebagai harian.
		return from.AddDate(0, 0, 1)
	}
}

// isValidFrequency memeriksa apakah frekuensi termasuk enum yang diizinkan.
func isValidFrequency(freq string) bool {
	return freq == models.FrequencyDaily || freq == models.FrequencyWeekly || freq == models.FrequencyMonthly
}

// CreateMaintenanceSchedule membuat jadwal pemeliharaan berulang baru (Admin saja)
// @Summary      Tambah Jadwal Pemeliharaan Berulang
// @Description  Membuat jadwal pemeliharaan berulang (harian/mingguan/bulanan) untuk sebuah CCTV. NextRunAt dihitung otomatis = waktu sekarang + frekuensi. Teknisi opsional; bila disertakan wajib berperan teknisi dan log hasil generator akan langsung ditugaskan ke teknisi tersebut.
// @Tags         Jadwal Pemeliharaan
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body CreateMaintenanceScheduleRequest true "Detail Jadwal"
// @Success      201 {object} map[string]interface{} "Jadwal berhasil dibuat"
// @Failure      400 {object} map[string]interface{} "Format tidak valid, CCTV/teknisi tidak valid, atau frekuensi tidak dikenal"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Failure      404 {object} map[string]interface{} "CCTV atau teknisi tidak ditemukan"
// @Failure      500 {object} map[string]interface{} "Gagal menyimpan jadwal"
// @Router       /maintenance-schedules [post]
func CreateMaintenanceSchedule(c *fiber.Ctx) error {
	var req CreateMaintenanceScheduleRequest
	if err := c.BodyParser(&req); err != nil {
		return sendBodyParseError(c, &req, err)
	}

	if errs := utils.ValidateStruct(&req); len(errs) > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Validasi input gagal",
			"errors":  errs,
		})
	}

	// Validasi manual tambahan di luar tag validator (opsional karena oneof
	// sudah menahan nilai tak dikenal, namun dijaga untuk keamanan berlapis).
	if !isValidFrequency(req.Frequency) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Frekuensi tidak valid. Gunakan: daily, weekly, atau monthly",
		})
	}

	// Pastikan CCTV yang dijadwalkan benar-benar ada — CCTV satu-satunya
	// sumber kebenaran aset.
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

	// Teknisi opsional: bila disertakan, wajib user berperan "teknisi".
	var technician *uuid.UUID
	if req.TechnicianID != "" {
		techUUID, err := uuid.Parse(req.TechnicianID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "error",
				"message": "ID teknisi tidak valid",
			})
		}
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
		technician = &techUUID
	}

	// Jadwal baru langsung aktif; NextRunAt dihitung dari waktu sekarang
	// + interval frekuensi (generator mulai boleh membuat log setelah itu).
	schedule := models.MaintenanceSchedule{
		CCTVID:       cctvUUID,
		TechnicianID: technician,
		ActionTaken:  req.ActionTaken,
		Frequency:    req.Frequency,
		NextRunAt:    nextRunForFrequency(req.Frequency, time.Now()),
		IsActive:     true,
	}

	if err := config.DB.Create(&schedule).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menyimpan jadwal pemeliharaan",
		})
	}

	utils.LogAudit(config.DB, c, "CREATE_SCHEDULE", "MAINTENANCE",
		"Membuat jadwal pemeliharaan berulang: "+schedule.ActionTaken+" (Frekuensi: "+schedule.Frequency+")", nil, "", "")

	// Muat ulang dengan relasi (CCTV & teknisi) untuk respon JSON
	config.DB.Preload("CCTV").Preload("Technician").First(&schedule, schedule.ID)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status": "success",
		"data":   schedule,
	})
}

// GetAllMaintenanceSchedules mengambil daftar jadwal pemeliharaan berulang (Admin saja)
// @Summary      Ambil Semua Jadwal Pemeliharaan
// @Description  Mendapatkan daftar jadwal pemeliharaan berulang terpaginasi dengan preload CCTV & teknisi. Mendukung filter is_active dan sorting.
// @Tags         Jadwal Pemeliharaan
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        is_active query boolean false "Filter berdasarkan status aktif jadwal (true/false)"
// @Param        page query integer false "Nomor Halaman (default: 1)"
// @Param        limit query integer false "Jumlah data per halaman (default: 10)"
// @Param        sort_by query string false "Kolom pengurutan (default: created_at)"
// @Param        order query string false "Arah pengurutan: asc, desc (default: desc)"
// @Success      200 {object} map[string]interface{} "Daftar jadwal terpaginasi"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Failure      500 {object} map[string]interface{} "Gagal mengambil daftar jadwal"
// @Router       /maintenance-schedules [get]
func GetAllMaintenanceSchedules(c *fiber.Ctx) error {
	var schedules []models.MaintenanceSchedule
	query := config.DB.Model(&models.MaintenanceSchedule{}).Preload("CCTV").Preload("Technician")

	// Filter status aktif/nonaktif jadwal (opsional)
	if isActive := c.Query("is_active"); isActive != "" {
		query = query.Where("is_active = ?", isActive == "true")
	}

	allowedSort := []string{"id", "cctv_id", "technician_id", "action_taken", "frequency", "next_run_at", "is_active", "last_generated_at", "created_at", "updated_at"}
	limit, page, sort := utils.GetPaginationParams(c, allowedSort)

	var totalRows int64
	if err := query.Count(&totalRows).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghitung total jadwal pemeliharaan",
		})
	}

	offset := (page - 1) * limit
	if err := query.Offset(offset).Limit(limit).Order(sort).Find(&schedules).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal mengambil daftar jadwal pemeliharaan",
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"rows":       schedules,
			"pagination": utils.CalculatePagination(totalRows, page, limit),
		},
	})
}

// GetMaintenanceScheduleByID mengambil detail satu jadwal pemeliharaan (Admin saja)
// @Summary      Ambil Jadwal Pemeliharaan Detail
// @Description  Mendapatkan detail satu jadwal pemeliharaan berulang beserta relasi CCTV & teknisi berdasarkan UUID-nya.
// @Tags         Jadwal Pemeliharaan
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Jadwal UUID"
// @Success      200 {object} map[string]interface{} "Data jadwal ditemukan"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Failure      404 {object} map[string]interface{} "Jadwal tidak ditemukan"
// @Router       /maintenance-schedules/{id} [get]
func GetMaintenanceScheduleByID(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID jadwal pemeliharaan tidak valid",
		})
	}

	var schedule models.MaintenanceSchedule
	if err := config.DB.Preload("CCTV").Preload("Technician").First(&schedule, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Jadwal pemeliharaan tidak ditemukan",
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   schedule,
	})
}

// UpdateMaintenanceSchedule memperbarui jadwal pemeliharaan berulang (Admin saja)
// @Summary      Update Jadwal Pemeliharaan
// @Description  Memperbarui data jadwal pemeliharaan berulang (CCTV, teknisi, template aksi, frekuensi). Bila frekuensi berubah, NextRunAt di-reset = waktu sekarang + frekuensi baru. Kirim technician_id kosong ("") untuk melepas penugasan teknisi.
// @Tags         Jadwal Pemeliharaan
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Jadwal UUID"
// @Param        request body UpdateMaintenanceScheduleRequest true "Data Update Jadwal"
// @Success      200 {object} map[string]interface{} "Jadwal berhasil diperbarui"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid, format body salah, atau teknisi tidak berperan teknisi"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Failure      404 {object} map[string]interface{} "Jadwal tidak ditemukan"
// @Failure      500 {object} map[string]interface{} "Gagal memperbarui jadwal"
// @Router       /maintenance-schedules/{id} [put]
func UpdateMaintenanceSchedule(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID jadwal pemeliharaan tidak valid",
		})
	}

	var schedule models.MaintenanceSchedule
	if err := config.DB.First(&schedule, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Jadwal pemeliharaan tidak ditemukan",
		})
	}

	var req UpdateMaintenanceScheduleRequest
	if err := c.BodyParser(&req); err != nil {
		return sendBodyParseError(c, &req, err)
	}

	if errs := utils.ValidateStruct(&req); len(errs) > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Validasi input gagal",
			"errors":  errs,
		})
	}

	// Tandai perubahan frekuensi untuk reset NextRunAt setelah update
	frequencyChanged := false

	if req.CCTVID != "" {
		cctvUUID, err := uuid.Parse(req.CCTVID)
		if err == nil {
			var cctv models.CCTV
			if err := config.DB.First(&cctv, "id = ?", cctvUUID).Error; err != nil {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
					"status":  "error",
					"message": "CCTV tidak ditemukan",
				})
			}
			schedule.CCTVID = cctvUUID
		}
	}

	if req.ActionTaken != "" {
		schedule.ActionTaken = req.ActionTaken
	}

	if req.Frequency != "" {
		if req.Frequency != schedule.Frequency {
			frequencyChanged = true
		}
		schedule.Frequency = req.Frequency
	}

	// TechnicianID pointer: nil = tidak diubah; "" = lepas teknisi;
	// UUID valid = ganti teknisi (wajib berperan teknisi).
	if req.TechnicianID != nil {
		if *req.TechnicianID == "" {
			schedule.TechnicianID = nil
		} else {
			techUUID, err := uuid.Parse(*req.TechnicianID)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"status":  "error",
					"message": "ID teknisi tidak valid",
				})
			}
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
			schedule.TechnicianID = &techUUID
		}
	}

	// Reset jadwal berikutnya bila frekuensi berubah — siklus baru dihitung
	// mulai dari sekarang agar generator tidak langsung mengejar jadwal lama.
	if frequencyChanged {
		schedule.NextRunAt = nextRunForFrequency(schedule.Frequency, time.Now())
	}

	if err := config.DB.Save(&schedule).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal memperbarui jadwal pemeliharaan",
		})
	}

	utils.LogAudit(config.DB, c, "UPDATE_SCHEDULE", "MAINTENANCE",
		"Memperbarui jadwal pemeliharaan berulang: "+schedule.ActionTaken+" (Frekuensi: "+schedule.Frequency+")", nil, "", "")

	// Muat ulang dengan relasi untuk respon JSON
	config.DB.Preload("CCTV").Preload("Technician").First(&schedule, schedule.ID)

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   schedule,
	})
}

// DeleteMaintenanceSchedule menghapus jadwal pemeliharaan berulang (Admin saja)
// @Summary      Hapus Jadwal Pemeliharaan
// @Description  Menghapus jadwal pemeliharaan berulang (soft-delete). Log yang sudah di-generate oleh jadwal tetap tersimpan.
// @Tags         Jadwal Pemeliharaan
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Jadwal UUID"
// @Success      200 {object} map[string]interface{} "Jadwal berhasil dihapus"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Failure      404 {object} map[string]interface{} "Jadwal tidak ditemukan"
// @Failure      500 {object} map[string]interface{} "Gagal menghapus jadwal"
// @Router       /maintenance-schedules/{id} [delete]
func DeleteMaintenanceSchedule(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID jadwal pemeliharaan tidak valid",
		})
	}

	var schedule models.MaintenanceSchedule
	if err := config.DB.First(&schedule, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Jadwal pemeliharaan tidak ditemukan",
		})
	}

	if err := config.DB.Delete(&schedule).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghapus jadwal pemeliharaan",
		})
	}

	utils.LogAudit(config.DB, c, "DELETE_SCHEDULE", "MAINTENANCE",
		"Menghapus jadwal pemeliharaan berulang: "+schedule.ActionTaken+" (Frekuensi: "+schedule.Frequency+")", nil, "", "")

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Jadwal pemeliharaan berhasil dihapus",
	})
}

// ToggleMaintenanceSchedule mengaktifkan/menonaktifkan jadwal pemeliharaan (Admin saja)
// @Summary      Aktifkan / Nonaktifkan Jadwal
// @Description  Membalik status IsActive jadwal pemeliharaan berulang. Jadwal nonaktif di-skip oleh generator sehingga tidak lagi membuat log otomatis (tanpa menghapus jadwal).
// @Tags         Jadwal Pemeliharaan
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Jadwal UUID"
// @Success      200 {object} map[string]interface{} "Status jadwal berhasil dibalik"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Failure      404 {object} map[string]interface{} "Jadwal tidak ditemukan"
// @Failure      500 {object} map[string]interface{} "Gagal mengubah status jadwal"
// @Router       /maintenance-schedules/{id}/toggle [put]
func ToggleMaintenanceSchedule(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID jadwal pemeliharaan tidak valid",
		})
	}

	var schedule models.MaintenanceSchedule
	if err := config.DB.First(&schedule, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Jadwal pemeliharaan tidak ditemukan",
		})
	}

	schedule.IsActive = !schedule.IsActive
	if err := config.DB.Save(&schedule).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal mengubah status jadwal pemeliharaan",
		})
	}

	statusLabel := "nonaktif"
	if schedule.IsActive {
		statusLabel = "aktif"
	}
	utils.LogAudit(config.DB, c, "TOGGLE_SCHEDULE", "MAINTENANCE",
		"Menjadikan jadwal pemeliharaan berulang "+statusLabel+": "+schedule.ActionTaken, nil, "", "")

	// Muat ulang dengan relasi untuk respon JSON
	config.DB.Preload("CCTV").Preload("Technician").First(&schedule, schedule.ID)

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   schedule,
	})
}
