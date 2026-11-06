package handlers

import (
	"sigap-backend/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// validateTechnicianSiteAssignment memastikan:
// Bila CCTV memiliki site (cctvSiteID != nil dan bukan nil UUID), teknisi WAJIB terdaftar
// pada site tersebut di tabel user_sites (Issue #184 / M11).
func validateTechnicianSiteAssignment(db *gorm.DB, techID uuid.UUID, cctvSiteID *uuid.UUID) (int, string) {
	if cctvSiteID == nil || *cctvSiteID == uuid.Nil {
		// CCTV tanpa site (mis. data warisan atau fixture unit test tanpa relasi site)
		return 0, ""
	}
	var count int64
	if err := db.Table("user_sites").Where("site_id = ? AND user_id = ?", *cctvSiteID, techID).Count(&count).Error; err != nil {
		return fiber.StatusInternalServerError, "Gagal memverifikasi keanggotaan site teknisi"
	}
	if count == 0 {
		return fiber.StatusBadRequest, "Teknisi tidak ditugaskan ke site tempat CCTV berada"
	}
	return 0, ""
}

// reconcileRevokedTechnicianSiteAssignments merekonsiliasi tugas aktif dan jadwal pemeliharaan
// saat teknisi dicabut dari sebuah site agar tidak menjadi tidak dapat diakses (Issue #184 / M11).
func reconcileRevokedTechnicianSiteAssignments(tx *gorm.DB, removedUserIDs []uuid.UUID, siteID uuid.UUID) error {
	if len(removedUserIDs) == 0 {
		return nil
	}

	// 1. Ambil daftar CCTV pada site ini
	var cctvIDs []uuid.UUID
	if err := tx.Model(&models.CCTV{}).Where("site_id = ?", siteID).Pluck("id", &cctvIDs).Error; err != nil {
		return err
	}
	if len(cctvIDs) == 0 {
		return nil
	}

	// 2. Unassign tugas pemeliharaan aktif (scheduled / ongoing)
	if err := tx.Model(&models.MaintenanceLog{}).
		Where("cctv_id IN ? AND technician_id IN ? AND status IN ('scheduled', 'ongoing')", cctvIDs, removedUserIDs).
		Updates(map[string]interface{}{
			"technician_id": nil,
			"status":        "scheduled",
		}).Error; err != nil {
		return err
	}

	// 3. Unassign jadwal pemeliharaan berulang
	if err := tx.Model(&models.MaintenanceSchedule{}).
		Where("cctv_id IN ? AND technician_id IN ?", cctvIDs, removedUserIDs).
		Update("technician_id", nil).Error; err != nil {
		return err
	}

	return nil
}

// reconcileRevokedTechnicianGlobalAssignments merekonsiliasi tugas aktif, jadwal berulang,
// serta keanggotaan site/zona ketika peran teknisi dicabut atau pengguna dihapus (Issue #184 / M11).
func reconcileRevokedTechnicianGlobalAssignments(tx *gorm.DB, userID uuid.UUID) error {
	// 1. Unassign semua log pemeliharaan aktif
	if err := tx.Model(&models.MaintenanceLog{}).
		Where("technician_id = ? AND status IN ('scheduled', 'ongoing')", userID).
		Updates(map[string]interface{}{
			"technician_id": nil,
			"status":        "scheduled",
		}).Error; err != nil {
		return err
	}

	// 2. Unassign jadwal pemeliharaan berulang
	if err := tx.Model(&models.MaintenanceSchedule{}).
		Where("technician_id = ?", userID).
		Update("technician_id", nil).Error; err != nil {
		return err
	}

	// 3. Bersihkan membership user_sites dan user_zones
	if err := tx.Where("user_id = ?", userID).Delete(&models.UserSite{}).Error; err != nil {
		return err
	}
	if err := tx.Where("user_id = ?", userID).Delete(&models.UserZone{}).Error; err != nil {
		return err
	}

	return nil
}
