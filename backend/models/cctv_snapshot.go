package models

import "github.com/google/uuid"

// CCTVSnapshot merekam snapshot frame / event visual pada sebuah CCTV.
// Setiap baris menyimpan path gambar JPEG yang diambil dari MediaMTX beserta
// tipe event (status berubah, pemeliharaan selesai, perhatian lapangan, atau
// snapshot berkala) dan catatan opsional.
type CCTVSnapshot struct {
	Base
	// CCTVID menunjuk ke CCTV pemilik snapshot (foreign key ke tabel cctvs).
	CCTVID uuid.UUID `gorm:"type:uuid;not null;index" json:"cctv_id"`
	// EventType adalah tipe event yang memicu pengambilan snapshot, misalnya
	// status_changed / maintenance_completed / field_check_attention /
	// periodic_snapshot (lihat konstanta di handlers/snapshot.go).
	EventType string `gorm:"type:varchar(50);not null" json:"event_type"`
	// ImagePath adalah path relatif file gambar JPEG, misalnya
	// "uploads/snapshots/{cctv_id}/{timestamp}.jpg".
	ImagePath string `gorm:"type:varchar(255)" json:"image_path"`
	// Notes adalah catatan opsional yang menyertai snapshot (mis. alasan status berubah).
	Notes string `gorm:"type:text" json:"notes"`
}
