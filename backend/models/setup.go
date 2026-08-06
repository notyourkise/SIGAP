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
	// ProfilePhotoURL menyimpan path file foto profil (mis. "/api/uploads/<uuid>.jpg").
	// Kosong = belum mengunggah foto; file disimpan di direktori uploads seperti
	// foto pemeliharaan dan disajikan via endpoint yang dilindungi JWT.
	ProfilePhotoURL string `gorm:"type:varchar(255)" json:"profile_photo_url"`
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
	// TechnicianID bersifat opsional (nil) karena log baru dibuat berstatus
	// "scheduled" tanpa penugasan; teknisi baru ditautkan saat admin menugaskan
	// lewat endpoint assign (status berubah menjadi "ongoing").
	TechnicianID *uuid.UUID `gorm:"type:uuid;index" json:"technician_id"`
	User         *User      `gorm:"foreignKey:TechnicianID" json:"technician"`
	// MaintenanceResultID menunjuk ke catatan FieldMaintenance hasil pengecekan
	// 9 tahap checklist yang diselesaikan teknisi (nil sebelum tugas selesai).
	MaintenanceResultID *uuid.UUID `gorm:"type:uuid;index" json:"maintenance_result_id"`
	// MaintenanceScheduleID menautkan log yang dibuat otomatis oleh generator
	// jadwal pemeliharaan berulang (Fase 5.4) ke MaintenanceSchedule. Nil saat
	// log dibuat manual. Kolom ini dipakai generator untuk menjamin idempotensi
	// (satu log per hari per jadwal) dan untuk audit/penelusuran asal log.
	MaintenanceScheduleID *uuid.UUID `gorm:"type:uuid;index" json:"maintenance_schedule_id"`
	ActionTaken           string     `gorm:"type:text;not null" json:"action_taken"`
	Notes                 string     `gorm:"type:text" json:"notes"`
	Status                string     `gorm:"type:varchar(20);not null" json:"status"`
	// OverdueNotifiedAt menyimpan waktu terakhir eskalasi overdue dikirim.
	// Nil berarti log belum pernah menerima notifikasi overdue, atau flag telah
	// di-reset karena jadwalnya dipindahkan ke masa depan.
	OverdueNotifiedAt *time.Time `json:"overdue_notified_at,omitempty"`
	ScheduledDate     *time.Time `json:"scheduled_date"`
	CompletedAt       *time.Time `json:"completed_at"`
	ImageUrl          string     `gorm:"type:varchar(255)" json:"image_url,omitempty"`
}
