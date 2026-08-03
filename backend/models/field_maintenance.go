package models

import (
	"time"

	"github.com/google/uuid"
)

// FieldMaintenance adalah catatan pengecekan lapangan (field check) yang
// dilakukan teknisi langsung di lokasi (mis. hasil pindai QR CCTV saat tiba
// di lapangan). Berbeda dengan MaintenanceLog (jadwal rutin dari admin),
// catatan ini dibuat secara spontan oleh teknisi dengan mengisi checklist
// item satu per satu dan melampirkan foto bukti di tahap akhir.
type FieldMaintenance struct {
	Base
	CCTVID       uuid.UUID `gorm:"type:uuid;index" json:"cctv_id"`
	CCTV         CCTV      `gorm:"foreignKey:CCTVID" json:"cctv"`
	TechnicianID uuid.UUID `gorm:"type:uuid;not null" json:"technician_id"`
	User         User      `gorm:"foreignKey:TechnicianID" json:"technician"`
	// OverallStatus merupakan hasil ringkasan seluruh checklist: "ok" bila
	// semua item berstatus ok, selain itu "attention" (perlu tindak lanjut).
	OverallStatus string `gorm:"type:varchar(20);not null" json:"overall_status"`
	Notes         string `gorm:"type:text" json:"notes"`
	CompletedAt   *time.Time `json:"completed_at"`
	Checks        []FieldMaintenanceCheck `gorm:"foreignKey:FieldMaintenanceID" json:"checks"`
	Images        []FieldMaintenanceImage `gorm:"foreignKey:FieldMaintenanceID" json:"images"`
}

// FieldMaintenanceCheck adalah satu item checklist yang wajib diisi teknisi.
// Result bernilai "ok" atau "not_ok".
type FieldMaintenanceCheck struct {
	Base
	FieldMaintenanceID uuid.UUID `gorm:"type:uuid;index" json:"field_maintenance_id"`
	Name               string    `gorm:"type:varchar(150);not null" json:"name"`
	Result             string    `gorm:"type:varchar(20);not null" json:"result"`
	Note               string    `gorm:"type:text" json:"note"`
}

// FieldMaintenanceImage adalah foto bukti pengecekan yang diunggah teknisi
// di tahap akhir sebelum menyelesaikan maintenance.
type FieldMaintenanceImage struct {
	Base
	FieldMaintenanceID uuid.UUID `gorm:"type:uuid;index" json:"field_maintenance_id"`
	ImageUrl           string    `gorm:"type:varchar(255);not null" json:"image_url"`
}
