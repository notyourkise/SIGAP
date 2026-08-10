package models

import (
	"github.com/google/uuid"
)

// Konstanta tipe notifikasi in-app (Fase 5.1). Nilai disimpan apa adanya di
// kolom "type" sehingga mudah dipetakan ke ikon/label pada komponen frontend.
const (
	NotificationTypeAssigned    = "MAINTENANCE_ASSIGNED"
	NotificationTypeReassigned  = "MAINTENANCE_REASSIGNED"
	NotificationTypeUnassigned  = "MAINTENANCE_UNASSIGNED"
	NotificationTypeCompleted   = "MAINTENANCE_COMPLETED"
	NotificationTypeOverdue     = "MAINTENANCE_OVERDUE"
	NotificationTypeOverdueCritical = "OVERDUE_CRITICAL"
	NotificationTypeFieldCheck  = "FIELD_CHECK_ATTENTION"
	NotificationTypeDailyDigest = "DAILY_DIGEST_AVAILABLE"
)

// Notification adalah notifikasi in-app per-pengguna (bell icon). Satu notifikasi
// dibuat untuk satu penerima; notifikasi ke seluruh user berperan sama dibuatkan
// baris sendiri-sendiri oleh util NotifyRole (Fase 5.1).
type Notification struct {
	Base
	// UserID adalah penerima notifikasi (Foreign Key → users).
	UserID uuid.UUID `gorm:"type:uuid;index:idx_notif_user_read,priority:1" json:"user_id"`
	User   User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	// Type berisi konstanta NotificationType* di atas (mis. MAINTENANCE_ASSIGNED).
	Type string `gorm:"type:varchar(50);not null" json:"type"`
	// Title ringkas untuk dropdown bell (mis. "Tugas baru ditugaskan").
	Title string `gorm:"type:varchar(150);not null" json:"title"`
	// Message detail lengkap untuk dibaca pada dropdown.
	Message string `gorm:"type:text;not null" json:"message"`
	// ReferenceID menunjuk ke record terkait (log pemeliharaan / pengecekan lapangan);
	// ReferenceType menjelaskan tabelnya ("maintenance-log" / "field-maintenance")
	// agar frontend tahu rute navigasi saat item diklik.
	ReferenceID   *uuid.UUID `gorm:"type:uuid" json:"reference_id,omitempty"`
	ReferenceType string     `gorm:"type:varchar(30)" json:"reference_type,omitempty"`
	IsRead        bool       `gorm:"type:boolean;not null;default:false;index:idx_notif_user_read,priority:2" json:"is_read"`
}
