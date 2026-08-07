package utils

import (
	"context"
	"encoding/json"
	"os"

	"sigap-backend/models"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ==========================================
// SIGAP — Web Push (Task 8.3)
// ==========================================
// Mengirim notifikasi push (OS/browser) ke langganan Web Push milik user
// menggunakan Web Push Protocol dengan VAPID. Kunci VAPID dibaca dari env:
//   - VAPID_PUBLIC_KEY
//   - VAPID_PRIVATE_KEY
//   - VAPID_EMAIL (kontak operator)
//
// Bila VAPID tidak dikonfigurasi, push dilewati dengan LOG WARNING (tidak
// menggagalkan operasi utama) — notifikasi in-app tetap berjalan sebagai
// fallback.

// sendNotificationFn adalah kontrak pengiriman push aktual. Di-inject untuk
// memudahkan unit test (mock). Default: webpush-go.
var sendNotificationFn = func(sub *webpush.Subscription, message []byte, options *webpush.Options) error {
	_, err := webpush.SendNotification(message, sub, options)
	return err
}

// SetSendNotificationFn mengganti implementasi pengiriman push (untuk test).
func SetSendNotificationFn(fn func(*webpush.Subscription, []byte, *webpush.Options) error) {
	sendNotificationFn = fn
}

// PushNotificationPayload adalah struktur pesan yang dikirim ke service worker
// (field data.push). Service worker membaca title/body untuk menampilkan
// notifikasi OS.
type PushNotificationPayload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	// NotificationType & ReferenceID dipakai service worker untuk membuka
	// halaman yang sesuai saat notifikasi diklik (opsional).
	NotificationType string `json:"notification_type,omitempty"`
	ReferenceID      string `json:"reference_id,omitempty"`
	ReferenceType    string `json:"reference_type,omitempty"`
}

// vapidConfigured mengembalikan true bila seluruh kunci VAPID tersedia di env.
func vapidConfigured() bool {
	return os.Getenv("VAPID_PUBLIC_KEY") != "" &&
		os.Getenv("VAPID_PRIVATE_KEY") != "" &&
		os.Getenv("VAPID_EMAIL") != ""
}

// SendPush mengirim notifikasi push ke semua langganan aktif milik user.
// Berjalan asynchronous (goroutine) dan TIDAK menggagalkan operasi utama bila
// gagal — kegagalan hanya dicatat sebagai warning. Dipanggil dari NotifyUser /
// NotifyUserSync setelah notifikasi in-app berhasil.
func SendPush(db *gorm.DB, userID uuid.UUID, notifType string, title string, message string, refType string, refID *uuid.UUID) {
	if db == nil || !vapidConfigured() {
		return
	}

	go func(database *gorm.DB, uid uuid.UUID) {
		var subs []models.PushSubscription
		if err := database.WithContext(context.Background()).
			Where("user_id = ? AND deleted_at IS NULL", uid).
			Find(&subs).Error; err != nil {
			Log.Warnf("⚠️ Push: gagal memuat subscription user %s: %v", uid, err)
			return
		}
		if len(subs) == 0 {
			return
		}

		var refIDStr string
		if refID != nil {
			refIDStr = refID.String()
		}
		payload := PushNotificationPayload{
			Title:            title,
			Body:             message,
			NotificationType: notifType,
			ReferenceID:      refIDStr,
			ReferenceType:    refType,
		}
		raw, err := json.Marshal(payload)
		if err != nil {
			Log.Warnf("⚠️ Push: gagal encode payload untuk user %s: %v", uid, err)
			return
		}

		for _, sub := range subs {
			if err := SendPushToSubscription(sub, raw); err != nil {
				// Gagal mengirim (mungkin endpoint kedaluwarsa). Jangan gagalkan
				// operasi utama — cukup catat warning.
				Log.Warnf("⚠️ Push gagal ke user %s: %v", uid, err)
			}
		}
	}(db, userID)
}

// SendPushToSubscription mengirim payload mentah (JSON) ke satu subscription
// menggunakan webpush-go dengan opsi VAPID. Mengembalikan error bila gagal.
func SendPushToSubscription(sub models.PushSubscription, raw []byte) error {
	subscription := &webpush.Subscription{
		Endpoint: sub.Endpoint,
		Keys: webpush.Keys{
			P256dh: sub.KeyP256DH,
			Auth:   sub.KeyAuth,
		},
	}
	options := &webpush.Options{
		Subscriber:      os.Getenv("VAPID_EMAIL"),
		VAPIDPublicKey:  os.Getenv("VAPID_PUBLIC_KEY"),
		VAPIDPrivateKey: os.Getenv("VAPID_PRIVATE_KEY"),
		TTL:             60,
	}
	return sendNotificationFn(subscription, raw, options)
}
