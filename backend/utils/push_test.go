package utils

import (
	"encoding/json"
	"testing"
	"time"

	"sigap-backend/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/SherClockHolmes/webpush-go"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func newPushTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{SkipDefaultTransaction: true})
	assert.NoError(t, err)
	return gormDB, mock
}

// TestSendPushToSubscriptionCallsInjectedSender memverifikasi bahwa
// SendPushToSubscription meneruskan subscription & payload ke sender yang
// di-inject (mock).
func TestSendPushToSubscriptionCallsInjectedSender(t *testing.T) {
	sub := models.PushSubscription{
		Endpoint:  "https://push.example.com/sub/xyz",
		KeyP256DH: "P256KEY",
		KeyAuth:   "AUTHKEY",
	}
	raw := []byte(`{"title":"T","body":"B"}`)

	var capturedSub *webpush.Subscription
	var capturedMsg []byte
	var capturedOptions *webpush.Options

	SetSendNotificationFn(func(s *webpush.Subscription, msg []byte, o *webpush.Options) error {
		capturedSub = s
		capturedMsg = msg
		capturedOptions = o
		return nil
	})
	defer SetSendNotificationFn(func(*webpush.Subscription, []byte, *webpush.Options) error { return nil })

	// Set VAPID env agar opsi terisi. t.Setenv otomatis membersihkan env var
	// setelah test & mem-fail test bila pengaturan env gagal (hindari errcheck).
	_, publicKey, err := webpush.GenerateVAPIDKeys()
	assert.NoError(t, err)
	t.Setenv("VAPID_PUBLIC_KEY", publicKey)
	t.Setenv("VAPID_PRIVATE_KEY", "PRIV")
	t.Setenv("VAPID_EMAIL", "ops@sigap.id")

	err = SendPushToSubscription(sub, raw)
	assert.NoError(t, err)

	assert.Equal(t, sub.Endpoint, capturedSub.Endpoint)
	assert.Equal(t, sub.KeyP256DH, capturedSub.Keys.P256dh)
	assert.Equal(t, sub.KeyAuth, capturedSub.Keys.Auth)
	assert.Equal(t, publicKey, capturedOptions.VAPIDPublicKey)
	assert.Equal(t, "PRIV", capturedOptions.VAPIDPrivateKey)
	assert.Equal(t, "mailto:ops@sigap.id", capturedOptions.Subscriber)
	assert.Equal(t, raw, capturedMsg)
}

// TestSendPushNoopWhenVapidMissing memastikan SendPush tidak melakukan query
// DB maupun pengiriman bila VAPID tidak dikonfigurasi (graceful no-op).
func TestSendPushNoopWhenVapidMissing(t *testing.T) {
	// Pastikan VAPID kosong (t.Setenv dengan nilai kosong, auto-cleanup).
	t.Setenv("VAPID_PUBLIC_KEY", "")
	t.Setenv("VAPID_PRIVATE_KEY", "")
	t.Setenv("VAPID_EMAIL", "")

	db, mock := newPushTestDB(t)
	// Tidak ada ekspektasi query — bila ada query, ExpectationsWereMet gagal.
	SendPush(db, uuid.New(), "MAINTENANCE_ASSIGNED", "T", "B", "maintenance-log", nil)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestSendPushSendsToAllSubscriptions memverifikasi SendPush memuat subscription
// user dan mengirim payload ke masing-masing (melalui sender mock).
func TestSendPushSendsToAllSubscriptions(t *testing.T) {
	privateKey, publicKey, err := webpush.GenerateVAPIDKeys()
	assert.NoError(t, err)
	t.Setenv("VAPID_PUBLIC_KEY", publicKey)
	t.Setenv("VAPID_PRIVATE_KEY", privateKey)
	t.Setenv("VAPID_EMAIL", "ops@sigap.id")

	db, mock := newPushTestDB(t)
	userID := uuid.New()

	// Dua subscription untuk user.
	rows := sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at", "user_id",
		"endpoint", "key_p256dh", "key_auth",
	}).
		AddRow(uuid.New(), time.Now(), time.Now(), nil, userID, "https://push.example.com/s1", "K1", "A1").
		AddRow(uuid.New(), time.Now(), time.Now(), nil, userID, "https://push.example.com/s2", "K2", "A2")
	mock.ExpectQuery(`^SELECT \* FROM "push_subscriptions"`).
		WithArgs(userID).
		WillReturnRows(rows)

	// Hitung panggilan sender mock (harus 2).
	var calls int
	done := make(chan struct{})
	SetSendNotificationFn(func(*webpush.Subscription, []byte, *webpush.Options) error {
		calls++
		done <- struct{}{}
		return nil
	})
	defer SetSendNotificationFn(func(*webpush.Subscription, []byte, *webpush.Options) error { return nil })

	SendPush(db, userID, "MAINTENANCE_ASSIGNED", "Tugas baru", "Dikerjakan", "maintenance-log", nil)

	// Tunggu kedua pengiriman selesai.
	for i := 0; i < 2; i++ {
		<-done
	}
	assert.Equal(t, 2, calls)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestVAPIDPublicKeyRejectsInvalidP256Key(t *testing.T) {
	t.Setenv("VAPID_PUBLIC_KEY", "_s8bFU0w3QkyhfaKa8fnhyqI0qPMatxYsiz_emqwvlA")
	_, err := VAPIDPublicKey()
	assert.Error(t, err)
}

// TestSendPushPayloadFormat memverifikasi payload JSON yang dikirim berisi field
// yang dibutuhkan service worker (title, body, dll).
func TestSendPushPayloadFormat(t *testing.T) {
	raw, err := json.Marshal(PushNotificationPayload{
		Title:            "Tugas baru",
		Body:             "Silakan kerjakan",
		NotificationType: "MAINTENANCE_ASSIGNED",
		ReferenceID:      "abc-123",
		ReferenceType:    "maintenance-log",
	})
	assert.NoError(t, err)

	var parsed map[string]interface{}
	assert.NoError(t, json.Unmarshal(raw, &parsed))
	assert.Equal(t, "Tugas baru", parsed["title"])
	assert.Equal(t, "Silakan kerjakan", parsed["body"])
	assert.Equal(t, "MAINTENANCE_ASSIGNED", parsed["notification_type"])
	assert.Equal(t, "abc-123", parsed["reference_id"])
}
