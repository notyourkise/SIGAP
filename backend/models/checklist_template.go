package models

import (
	"github.com/google/uuid"
)

// ChecklistTemplate adalah item template checklist pengecekan lapangan
// (Task 13.12). Dapat dikustomisasi admin (nama, deskripsi, urutan, wajib,
// aktif) tanpa perlu deploy ulang. ID di-generate otomatis via hook GORM
// BeforeCreate pada Base (models/setup.go).
type ChecklistTemplate struct {
	Base
	NamaItem  string `gorm:"type:varchar(255);not null" json:"nama_item"`
	Deskripsi string `gorm:"type:text" json:"deskripsi"`
	Urutan    int    `gorm:"type:integer;not null;default:0" json:"urutan"`
	Wajib     bool   `gorm:"type:boolean;not null;default:true" json:"wajib"`
	Aktif     bool   `gorm:"type:boolean;not null;default:true" json:"aktif"`
	// SiteID menunjuk kawasan operasional (multi-site, Task 12.2). Nilai NULL
	// berarti template GLOBAL (berlaku untuk semua kawasan). Dengan ini daftar
	// checklist dapat berbeda per kawasan (perbaikan Sprint 40).
	SiteID *uuid.UUID `gorm:"type:uuid;index" json:"site_id,omitempty"`
}
