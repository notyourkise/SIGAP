package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CCTVHealthHistory merekam histori skor kesehatan CCTV (Task 13.8). Satu baris
// per perhitungan skor, diurutkan menurun per (cctv_id, calculated_at).
type CCTVHealthHistory struct {
	ID           uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	CCTVID       uuid.UUID `gorm:"type:uuid;not null;index:idx_health_histories_cctv_date,priority:1" json:"cctv_id"`
	HealthScore  int       `gorm:"type:integer;not null" json:"health_score"`
	CalculatedAt time.Time `gorm:"type:timestamptz;not null;default:now();index:idx_health_histories_cctv_date,priority:2" json:"calculated_at"`
}

// BeforeCreate memastikan setiap histori mendapat UUID unik. Model ini tidak
// memakai Base, jadi tanpa hook ini GORM mengirim UUID nol dan insert kedua
// selalu gagal karena primary key duplikat.
func (h *CCTVHealthHistory) BeforeCreate(tx *gorm.DB) error {
	if h.ID == uuid.Nil {
		h.ID = uuid.New()
	}
	return nil
}
