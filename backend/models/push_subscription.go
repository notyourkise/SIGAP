package models

import (
	"github.com/google/uuid"
)

// PushSubscription menyimpan langganan Web Push (browser) milik seorang user
// (Task 8.3). Satu user dapat memiliki beberapa subscription (multi perangkat/
// browser). Dipakai oleh util SendPush untuk mengirim notifikasi OS ke semua
// perangkat yang aktif.
type PushSubscription struct {
	Base
	// UserID adalah pemilik subscription (Foreign Key → users).
	UserID uuid.UUID `gorm:"type:uuid;index" json:"user_id"`
	User   User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	// Endpoint URL unik yang diberikan browser untuk menerima push.
	Endpoint string `gorm:"type:text;not null" json:"endpoint"`
	// KeyP256DH adalah kunci publik (P-256) untuk enkripsi payload.
	// Column dieksplisitkan menjadi "key_p256dh" (bukan snake_case GORM
	// "key_p256_dh") agar selaras dengan standar Web Push & nama JSON.
	KeyP256DH string `gorm:"column:key_p256dh;type:text;not null" json:"key_p256dh"`
	// KeyAuth adalah rahasia auth untuk enkripsi payload.
	KeyAuth string `gorm:"type:text;not null" json:"key_auth"`
}
