package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CCTVStreamLog mencatat riwayat event perubahan status stream/koneksi CCTV (Sprint 42).
type CCTVStreamLog struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	CreatedAt     time.Time `gorm:"not null;default:now()" json:"created_at"`
	CCTVID        uuid.UUID `gorm:"type:uuid;not null;index" json:"cctv_id"`
	CCTV          *CCTV     `gorm:"foreignKey:CCTVID;constraint:OnDelete:CASCADE" json:"cctv,omitempty"`
	EventType     string    `gorm:"type:varchar(32);not null;index" json:"event_type"`
	PreviousState string    `gorm:"type:varchar(32);not null;default:''" json:"previous_state"`
	CurrentState  string    `gorm:"type:varchar(32);not null" json:"current_state"`
	TriggerSource string    `gorm:"type:varchar(32);not null;default:'probe'" json:"trigger_source"`
	NetworkType   string    `gorm:"type:varchar(20);not null;default:'gsm'" json:"network_type"`
	LatencyMs     *int      `gorm:"type:integer" json:"latency_ms,omitempty"`
	ErrorDetail   string    `gorm:"type:text;not null;default:''" json:"error_detail"`
	Message       string    `gorm:"type:text;not null;default:''" json:"message"`
}

// TableName menentukan nama tabel untuk CCTVStreamLog.
func (CCTVStreamLog) TableName() string {
	return "cctv_stream_logs"
}

// BeforeCreate menghasilkan UUID secara otomatis bila belum diisi.
func (l *CCTVStreamLog) BeforeCreate(tx *gorm.DB) (err error) {
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	return
}
