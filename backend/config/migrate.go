package config

import (
	"log"

	"sigap-backend/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// legacyAssetRef mewakili baris maintenance_logs lama yang masih memakai asset_id.
type legacyAssetRef struct {
	ID      uuid.UUID
	AssetID uuid.UUID
}

// MigrateMaintenanceToCCTV memigrasi kolom asset_id lama pada maintenance_logs
// menuju cctv_id — menjadikan tabel cctvs satu-satunya sumber kebenaran aset
// (Fase 3.6). Data lama dipetakan ke CCTV berdasarkan urutan: ID langsung,
// IP kamera/NVR, kode CCTV, lalu nama. Aman dijalankan berulang kali (idempotent).
func MigrateMaintenanceToCCTV(db *gorm.DB) {
	// Pastikan tabel dan kolom cctv_id sudah ada (dibuat AutoMigrate).
	if !db.Migrator().HasTable("maintenance_logs") || !db.Migrator().HasColumn(&models.MaintenanceLog{}, "CCTVID") {
		return
	}
	// Kolom asset_id lama sudah dihapus — tidak ada yang perlu dimigrasi.
	if !db.Migrator().HasColumn("maintenance_logs", "asset_id") {
		return
	}

	var refs []legacyAssetRef
	if err := db.Raw(`SELECT id, asset_id FROM maintenance_logs WHERE cctv_id IS NULL`).Scan(&refs).Error; err != nil {
		log.Printf("⚠️ Migrasi maintenance→cctv: gagal membaca data lama: %v", err)
	} else if len(refs) > 0 {
		hasAssetsTable := db.Migrator().HasTable("assets")
		updated := 0
		for _, ref := range refs {
			if ref.AssetID == uuid.Nil {
				continue
			}
			cctvID, ok := resolveLegacyAssetToCCTV(db, ref.AssetID, hasAssetsTable)
			if !ok {
				log.Printf("⚠️ Migrasi maintenance→cctv: log %s tidak dapat dipetakan ke CCTV (asset_id %s). Periksa secara manual.", ref.ID, ref.AssetID)
				continue
			}
			if err := db.Model(&models.MaintenanceLog{}).Unscoped().Where("id = ?", ref.ID).Update("cctv_id", cctvID).Error; err != nil {
				log.Printf("⚠️ Migrasi maintenance→cctv: gagal memperbarui log %s: %v", ref.ID, err)
				continue
			}
			updated++
		}
		log.Printf("✅ Migrasi maintenance→cctv selesai: %d log dipetakan ke CCTV", updated)
	}

	// Hapus kolom legacy asset_id beserta foreign key-nya. Tanpa ini kolom lama yang masih
	// NOT NULL membuat INSERT baru gagal (model sudah tidak mengirim asset_id lagi).
	if db.Migrator().HasColumn("maintenance_logs", "asset_id") {
		if err := db.Migrator().DropColumn("maintenance_logs", "asset_id"); err != nil {
			log.Printf("⚠️ Migrasi maintenance→cctv: gagal menghapus kolom legacy asset_id: %v", err)
		} else {
			log.Printf("✅ Migrasi maintenance→cctv: kolom legacy asset_id dihapus dari maintenance_logs")
		}
	}
}

// resolveLegacyAssetToCCTV mencari CCTV yang sesuai dengan referensi asset lama.
func resolveLegacyAssetToCCTV(db *gorm.DB, assetID uuid.UUID, hasAssetsTable bool) (uuid.UUID, bool) {
	// 1. Apakah asset_id sudah berupa ID CCTV yang valid?
	var cctv models.CCTV
	if err := db.Unscoped().First(&cctv, "id = ?", assetID).Error; err == nil {
		return cctv.ID, true
	}

	if !hasAssetsTable {
		return uuid.Nil, false
	}

	// 2. Petakan lewat tabel assets lama (ip / kode / nama).
	var legacy struct {
		IPAddress    string
		SerialNumber string
		Name         string
	}
	if err := db.Raw(`SELECT ip_address, serial_number, name FROM assets WHERE id = ?`, assetID).Scan(&legacy).Error; err != nil {
		return uuid.Nil, false
	}

	if legacy.IPAddress != "" {
		if err := db.Unscoped().First(&cctv, "ip_camera = ? OR ip_nvr = ?", legacy.IPAddress, legacy.IPAddress).Error; err == nil {
			return cctv.ID, true
		}
	}
	if legacy.SerialNumber != "" {
		if err := db.Unscoped().First(&cctv, "kode_cctv = ?", legacy.SerialNumber).Error; err == nil {
			return cctv.ID, true
		}
	}
	if legacy.Name != "" {
		if err := db.Unscoped().First(&cctv, "nama = ?", legacy.Name).Error; err == nil {
			return cctv.ID, true
		}
	}
	return uuid.Nil, false
}
