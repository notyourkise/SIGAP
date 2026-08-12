package models

import (
	"github.com/google/uuid"
)

// Zone adalah wilayah operasional penempatan CCTV dan teknisi (Task 11.4,
// Sprint 28). Satu zona menaungi banyak CCTV (one-to-many via CCTV.ZoneID)
// dan banyak teknisi (many-to-many via join table user_zones).
type Zone struct {
	Base
	// Nama adalah label zona yang mudah dikenali (mis. "Zona Utara", "KIPP").
	Nama string `gorm:"type:varchar(100);not null" json:"nama"`
	// Deskripsi berisi keterangan tambahan mengenai cakupan zona.
	Deskripsi string `gorm:"type:text" json:"deskripsi"`
	// CCTVs adalah daftar perangkat CCTV yang berada dalam zona ini
	// (one-to-many; relasi dikelola via kolom CCTV.ZoneID).
	CCTVs []CCTV `gorm:"foreignKey:ZoneID" json:"cctvs,omitempty"`
	// Users adalah daftar teknisi yang ditugaskan ke zona ini
	// (many-to-many via join table user_zones).
	Users []User `gorm:"many2many:user_zones;" json:"users,omitempty"`
	// SiteID menunjuk ke kawasan operasional (multi-site, Task 12.2). Opsional
	// — data existing otomatis masuk ke site default "OIKN" saat migrasi.
	SiteID *uuid.UUID `gorm:"type:uuid;index" json:"site_id,omitempty"`
}

// UserZone adalah model eksplisit untuk join table many-to-many User ↔ Zone.
// Bila GORM mengelola relasi many2many secara otomatis, kolom utama yang
// digunakan adalah ZoneID dan UserID.
type UserZone struct {
	ZoneID uuid.UUID `gorm:"type:uuid;primaryKey;not null;index:idx_user_zones_zone" json:"zone_id"`
	UserID uuid.UUID `gorm:"type:uuid;primaryKey;not null;index:idx_user_zones_user" json:"user_id"`
}
