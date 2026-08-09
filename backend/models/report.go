package models

import "github.com/google/uuid"

// ReportTemplate menyimpan template laporan kustom yang dapat disimpan dan
// dipakai ulang oleh admin (Task 11.2, Sprint 27). Konfigurasi laporan
// (entitas, kolom, filter, format) disimpan sebagai JSON string.
type ReportTemplate struct {
	Base
	// Nama adalah label template yang mudah dikenali oleh admin.
	Nama string `gorm:"type:varchar(100);not null" json:"nama"`
	// Config berisi JSON: {"entity": "...", "columns": [...], "filters": {...}, "format": "..."}.
	Config string `gorm:"type:jsonb;not null" json:"config"`
	// CreatedBy menunjuk ke user admin yang membuat template.
	CreatedBy uuid.UUID `gorm:"type:uuid;not null;index" json:"created_by"`
	User      User      `gorm:"foreignKey:CreatedBy" json:"user,omitempty"`
}
