package models

// Konstanta jenis tugas (JenisTugas) yang dipakai pada kebijakan SLA.
// Menentukan aturan target waktu untuk tiap kategori pekerjaan pemeliharaan.
const (
	// SLATypeCorrective adalah perbaikan darurat/korektif pada CCTV bermasalah.
	SLATypeCorrective = "corrective"
	// SLATypePreventive adalah pemeliharaan rutin preventif terjadwal.
	SLATypePreventive = "preventive"
	// SLATypeFieldCheck adalah pengecekan lapangan (field check) berkala.
	SLATypeFieldCheck = "field_check"
)

// SLAPolicy adalah kebijakan Service Level Agreement (Task 11.5, Sprint 28).
// Mendefinisikan target waktu respons (TargetJamRespon) dan waktu penyelesaian
// (TargetJamSelesai) untuk tiap jenis tugas. MaintenanceSchedule menunjuk ke
// kebijakan ini melalui SLAPolicyID agar jadwal otomatis mengikuti target SLA.
type SLAPolicy struct {
	Base
	// Nama adalah label kebijakan SLA (mis. "SLA Korektif — Prioritas Tinggi").
	Nama string `gorm:"type:varchar(100);not null" json:"nama"`
	// JenisTugas adalah enum jenis tugas: corrective / preventive / field_check.
	JenisTugas string `gorm:"type:varchar(30);not null" json:"jenis_tugas"`
	// TargetJamRespon adalah target waktu respons dalam jam (sejak tugas dibuat).
	TargetJamRespon int `gorm:"type:integer;not null;default:1" json:"target_jam_respon"`
	// TargetJamSelesai adalah target waktu penyelesaian dalam jam (sejak tugas dibuat).
	TargetJamSelesai int `gorm:"type:integer;not null;default:8" json:"target_jam_selesai"`
}
