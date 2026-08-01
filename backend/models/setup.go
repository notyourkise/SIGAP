package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Base struct untuk otomatisasi UUID, Timestamp, dan Soft Delete
type Base struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key;" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// Hook GORM: Generate UUID secara otomatis sebelum data disimpan ke database jika belum diisi
func (base *Base) BeforeCreate(tx *gorm.DB) (err error) {
	if base.ID == uuid.Nil {
		base.ID = uuid.New()
	}
	return
}

type User struct {
	Base
	Username     string `gorm:"type:varchar(50);uniqueIndex;not null" json:"username"`
	PasswordHash string `gorm:"type:varchar(255);not null" json:"-"` // Sembunyikan dari respon JSON API
	FullName     string `gorm:"type:varchar(100);not null" json:"full_name"`
	Email        string `gorm:"type:varchar(100);uniqueIndex;not null" json:"email"`
	Role         string `gorm:"type:varchar(20);not null" json:"role"`
	// TokenVersion dipakai untuk mencabut semua token lama saat password
	// di-reset atau role diubah oleh Admin (disematkan dalam klaim JWT "tv").
	TokenVersion int64 `gorm:"type:integer;not null;default:1" json:"-"`
	// MustChangePassword memaksa pengguna mengganti password saat login pertama
	// (setelah akun dibuat / password di-reset oleh Admin). Ditandai false otomatis
	// setelah pengguna berhasil mengganti password lewat /auth/change-password.
	MustChangePassword bool `gorm:"type:boolean;not null;default:false" json:"must_change_password"`
}

type MaintenanceLog struct {
	Base
	// CCTVID menunjuk langsung ke tabel cctvs — CCTV adalah satu-satunya sumber
	// kebenaran aset (unifikasi model assets ↔ cctv, Fase 3.6).
	CCTVID       uuid.UUID  `gorm:"type:uuid;index" json:"cctv_id"`
	CCTV         CCTV       `gorm:"foreignKey:CCTVID" json:"cctv"`
	TechnicianID uuid.UUID  `gorm:"type:uuid;not null" json:"technician_id"`
	User         User       `gorm:"foreignKey:TechnicianID" json:"technician"`
	ActionTaken  string     `gorm:"type:text;not null" json:"action_taken"`
	Notes        string     `gorm:"type:text" json:"notes"`
	Status       string     `gorm:"type:varchar(20);not null" json:"status"`
	ScheduledDate *time.Time `json:"scheduled_date"`
	CompletedAt  *time.Time `json:"completed_at"`
	ImageUrl     string     `gorm:"type:varchar(255)" json:"image_url,omitempty"`
}
