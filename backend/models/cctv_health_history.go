package models

import (
	"time"

	"github.com/google/uuid"
)

// CCTVHealthHistory merekam histori skor kesehatan CCTV (Task 13.8). Satu baris
// per perhitungan skor, diurutkan menurun per (cctv_id, calculated_at).
type CCTVHealthHistory struct {
	ID           uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	CCTVID       uuid.UUID `gorm:"type:uuid;not null;index:idx_health_histories_cctv_date,priority:1" json:"cctv_id"`
	HealthScore  int       `gorm:"type:integer;not null" json:"health_score"`
	CalculatedAt time.Time `gorm:"type:timestamptz;not null;default:now();index:idx_health_histories_cctv_date,priority:2" json:"calculated_at"`
}
