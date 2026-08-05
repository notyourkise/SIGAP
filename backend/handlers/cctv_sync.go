package handlers

import (
	"fmt"

	"sigap-backend/config"
	"sigap-backend/models"
	"sigap-backend/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// syncCCTVStatus memperbarui status CCTV secara otomatis dan KONSERVATIF di
// dalam transaksi yang sama dengan operasi complete/field-check (Fase 5.2).
//
// Aturan transisi (keputusan desain 2026-08-04):
//   - Hanya perpindahan aktif ↔ pemeliharaan yang boleh dilakukan otomatis.
//   - Status "nonaktif" TIDAK PERNAH ditimpa otomatis — keputusan manual admin.
//     (Sejak 2026-08-05 "rusak" digabung ke "nonaktif".)
//   - Jika status saat ini sudah sama dengan target, tidak ada perubahan.
//
// Saat status benar-benar berubah, helper mencatat audit log UPDATE_CCTV
// (keterangan otomatis) dan menyiarkan event WebSocket CCTV_UPDATED agar
// seluruh klien (tabel & peta sebaran) ter-refresh real-time.
func syncCCTVStatus(tx *gorm.DB, c *fiber.Ctx, cctvID uuid.UUID, newStatus string, reason string) error {
	var cctv models.CCTV
	if err := tx.First(&cctv, "id = ?", cctvID).Error; err != nil {
		return err
	}

	// Konservatif: status nonaktif tidak pernah ditimpa otomatis.
	// Bandingkan nilai ternormalisasi agar baris legacy berhuruf besar/Inggris
	// ('INACTIVE', 'RUSAK', 'BROKEN', ...) tetap dilindungi (regresi 2026-08-05).
	current := normalizeCCTVStatus(cctv.Status)
	if current == "nonaktif" {
		return nil
	}

	// Status sudah sesuai target — tidak perlu mengubah apa pun
	if current == newStatus {
		return nil
	}

	oldStatus := cctv.Status
	if err := tx.Model(&models.CCTV{}).Where("id = ?", cctvID).
		Update("status", newStatus).Error; err != nil {
		return err
	}
	cctv.Status = newStatus

	// Audit log dengan keterangan otomatis (actor diambil dari konteks request)
	desc := fmt.Sprintf("Status CCTV diubah otomatis dari '%s' ke '%s' karena %s", oldStatus, newStatus, reason)
	utils.LogAudit(config.DB, c, "UPDATE_CCTV", "CCTV", desc, nil, "", "")
	utils.BroadcastEvent("CCTV_UPDATED", cctv)
	return nil
}
