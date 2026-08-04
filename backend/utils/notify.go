package utils

import (
	"fmt"
	"sigap-backend/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	notifyWriteAttempts = 3
	notifyRetryDelay    = 100 * time.Millisecond
)

// EventNotifCreated adalah nama event WebSocket yang dipancarkan saat notifikasi
// baru dibuat (dikonsumsi oleh NotificationBell di frontend).
const EventNotifCreated = "NOTIFICATION_CREATED"

// NotifyUser membuat notifikasi in-app untuk satu user lalu mem-broadcast event
// NOTIFICATION_CREATED secara terarah (targeted) ke koneksi WebSocket milik user
// tersebut. Penulisan database dilakukan asynchronous dengan retry terbatas —
// pola yang sama dengan LogAudit — sehingga request operasional tidak menunggu.
func NotifyUser(db *gorm.DB, userID uuid.UUID, notifType string, title string, message string, refType string, refID *uuid.UUID) {
	if db == nil {
		return
	}

	notif := models.Notification{
		UserID:        userID,
		Type:          notifType,
		Title:         title,
		Message:       message,
		ReferenceType: refType,
		ReferenceID:   refID,
	}

	go func(n models.Notification, database *gorm.DB) {
		if err := persistNotification(database, &n); err != nil {
			Log.Errorf("❌ Gagal menyimpan Notification [%s] setelah %d percobaan: %v", n.Type, notifyWriteAttempts, err)
			return
		}
		BroadcastToUser(n.UserID.String(), EventNotifCreated, n)
	}(notif, db)
}

// NotifyRole membuat notifikasi in-app untuk setiap user berperan tertentu lalu
// mem-broadcast NOTIFICATION_CREATED ke seluruh koneksi role tersebut. Query
// user dilakukan di goroutine agar tidak memblokir request pemanggil.
func NotifyRole(db *gorm.DB, role string, notifType string, title string, message string, refType string, refID *uuid.UUID) {
	if db == nil {
		return
	}

	go func(database *gorm.DB) {
		var users []models.User
		if err := database.Select("id").Where("role = ? AND deleted_at IS NULL", role).Find(&users).Error; err != nil {
			Log.Errorf("❌ Gagal memuat user role %s untuk notifikasi: %v", role, err)
			return
		}
		for _, u := range users {
			NotifyUser(database, u.ID, notifType, title, message, refType, refID)
		}
	}(db)
}

func persistNotification(database *gorm.DB, notif *models.Notification) error {
	var lastErr error
	for attempt := 1; attempt <= notifyWriteAttempts; attempt++ {
		if err := database.Create(notif).Error; err == nil {
			return nil
		} else {
			lastErr = err
		}

		if attempt < notifyWriteAttempts {
			time.Sleep(notifyRetryDelay)
		}
	}

	return fmt.Errorf("notification write failed: %w", lastErr)
}
