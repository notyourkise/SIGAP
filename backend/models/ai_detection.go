package models

import (
	"time"

	"github.com/google/uuid"
)

// AIDetectionSession mewakili ringkasan sesi deteksi AI harian per CCTV
// (Task 14B, Sprint 38). Satu baris per (cctv_id, session_date), di-upsert
// oleh backend setiap kali event deteksi diterima dari service sigap_ai.
//
// CATATAN: model ini TIDAK me-embed Base (tidak memakai soft delete / hook
// BeforeCreate). ID diisi eksplisit dengan uuid.New() di handler, mengikuti
// skema migrasi 000020 yang memakai DEFAULT gen_random_uuid().
type AIDetectionSession struct {
	// ID adalah primary key UUID (diisi eksplisit oleh handler).
	ID uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	// CCTVID menunjuk CCTV yang diamati (foreign key ke cctvs.id).
	CCTVID uuid.UUID `gorm:"type:uuid;not null;index" json:"cctv_id"`
	// SessionDate adalah tanggal sesi (tengah malam WITA/local server).
	SessionDate time.Time `gorm:"type:date;not null" json:"session_date"`
	// UniquePersonCount adalah jumlah orang unik terdeteksi sepanjang hari.
	UniquePersonCount int `gorm:"type:integer;not null;default:0" json:"unique_person_count"`
	// LastTrackingIDs adalah daftar tracking id unik terakhir (jsonb).
	LastTrackingIDs []byte `gorm:"type:jsonb" json:"last_tracking_ids"`
	// CreatedAt mencatat waktu pembuatan baris sesi.
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt mencatat waktu pembaruan terakhir (diubah GORM otomatis).
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName menetapkan nama tabel eksplisit agar konsisten dengan migrasi.
func (AIDetectionSession) TableName() string {
	return "ai_detection_sessions"
}

// AIDetectionEvent mewakili satu event deteksi AI yang diterima dari service
// sigap_ai (Task 14B, Sprint 38). Satu baris per frame/kejadian deteksi.
//
// CATATAN: model ini TIDAK me-embed Base (tidak memakai soft delete / hook
// BeforeCreate). ID diisi eksplisit dengan uuid.New() di handler, mengikuti
// skema migrasi 000021 yang memakai DEFAULT gen_random_uuid().
type AIDetectionEvent struct {
	// ID adalah primary key UUID (diisi eksplisit oleh handler).
	ID uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	// CCTVID menunjuk CCTV tempat deteksi terjadi.
	CCTVID uuid.UUID `gorm:"type:uuid;not null;index" json:"cctv_id"`
	// EventType adalah jenis event deteksi (default "person_detected").
	EventType string `gorm:"type:varchar(50);not null;default:'person_detected'" json:"event_type"`
	// PersonCount adalah jumlah orang di frame saat deteksi.
	PersonCount int `gorm:"type:integer;not null;default:0" json:"person_count"`
	// UniqueCount adalah jumlah orang unik di frame saat deteksi.
	UniqueCount int `gorm:"type:integer;not null;default:0" json:"unique_count"`
	// ConfidenceAvg adalah rata-rata confidence deteksi (0-1).
	ConfidenceAvg float64 `gorm:"type:double precision" json:"confidence_avg"`
	// DetectedAt mencatat waktu deteksi terjadi.
	DetectedAt time.Time `json:"detected_at"`
}

// TableName menetapkan nama tabel eksplisit agar konsisten dengan migrasi.
func (AIDetectionEvent) TableName() string {
	return "ai_detection_events"
}
