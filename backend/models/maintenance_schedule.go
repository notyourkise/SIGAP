package models

import (
	"time"

	"github.com/google/uuid"
)

// Konstanta frekuensi penjadwalan pemeliharaan berulang (Fase 5.4).
const (
	// FrequencyDaily membuat log setiap hari.
	FrequencyDaily = "daily"
	// FrequencyWeekly membuat log setiap 7 hari.
	FrequencyWeekly = "weekly"
	// FrequencyMonthly membuat log setiap bulan (tanggal sama).
	FrequencyMonthly = "monthly"
)

// MaintenanceSchedule adalah jadwal pemeliharaan berulang yang otomatis
// membuat MaintenanceLog sesuai frekuensi (harian/mingguan/bulanan). Generator
// (handlers.RunScheduleGenerator) berjalan tiap 1 jam dari main.go dan:
//  1. mencari jadwal aktif yang waktunya tiba (next_run_at <= sekarang),
//  2. membuat satu log per hari jadwal (idempotent — cek log yang sudah ada
//     dengan maintenance_schedule_id yang sama pada tanggal yang sama),
//  3. menggeser NextRunAt sesuai frekuensi dan mencatat LastGeneratedAt.
type MaintenanceSchedule struct {
	Base
	// CCTVID menunjuk ke perangkat CCTV yang dipelihara berulang
	// (Foreign Key → cctvs). CCTV tetap satu-satunya sumber kebenaran aset.
	CCTVID uuid.UUID `gorm:"type:uuid;index" json:"cctv_id"`
	CCTV   CCTV      `gorm:"foreignKey:CCTVID" json:"cctv"`
	// TechnicianID opsional: bila diisi, log hasil generator langsung ditugaskan
	// (status "ongoing") dan teknisi menerima notifikasi; bila kosong, log dibuat
	// berstatus "scheduled" menunggu ditugaskan manual oleh admin.
	TechnicianID *uuid.UUID `gorm:"type:uuid;index" json:"technician_id"`
	Technician   *User      `gorm:"foreignKey:TechnicianID" json:"technician"`
	// ActionTaken adalah template tindakan yang disalin ke MaintenanceLog hasil
	// generator (mis. "Pembersihan lensa dan pemeriksaan konektor RJ45").
	ActionTaken string `gorm:"type:text;not null" json:"action_taken"`
	// Frequency menentukan interval pembuatan log: daily / weekly / monthly.
	Frequency string `gorm:"type:varchar(20);not null" json:"frequency"`
	// NextRunAt menentukan waktu terdekat generator boleh membuat log berikutnya.
	NextRunAt time.Time `gorm:"not null;index" json:"next_run_at"`
	// IsActive menandakan jadwal aktif (menghasilkan log) atau dijeda.
	// Toggle dilakukan admin lewat endpoint PUT /:id/toggle.
	IsActive bool `gorm:"type:boolean;not null;default:true" json:"is_active"`
	// LastGeneratedAt mencatat waktu terakhir generator berhasil membuat log
	// otomatis untuk jadwal ini (nil jika belum pernah di-generate).
	LastGeneratedAt *time.Time `json:"last_generated_at"`
}
