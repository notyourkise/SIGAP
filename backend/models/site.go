package models

import "github.com/google/uuid"

// Site adalah kawasan operasional (multi-site). Semua data CCTV/log/zona
// dikelompokkan per site; data existing otomatis masuk ke site default "OIKN".
type Site struct {
	Base
	Nama         string   `gorm:"type:varchar(100);not null" json:"nama"`
	Kode         string   `gorm:"type:varchar(20);uniqueIndex;not null" json:"kode"`
	Alamat       string   `gorm:"type:text" json:"alamat"`
	KoordinatLat *float64 `gorm:"type:double precision" json:"koordinat_lat"`
	KoordinatLng *float64 `gorm:"type:double precision" json:"koordinat_lng"`
	IsDefault    bool     `gorm:"type:boolean;not null;default:false" json:"is_default"`
	// Relasi many-to-many ke user via join table user_sites.
	Users        []User     `gorm:"many2many:user_sites;" json:"users,omitempty"`
	UserSites    []UserSite `gorm:"foreignKey:SiteID" json:"user_sites,omitempty"`
}

// UserSite adalah join table many-to-many User ↔ Site.
type UserSite struct {
	UserID uuid.UUID `gorm:"type:uuid;primaryKey;not null;index:idx_user_sites_user" json:"user_id"`
	SiteID uuid.UUID `gorm:"type:uuid;primaryKey;not null;index:idx_user_sites_site" json:"site_id"`
}
