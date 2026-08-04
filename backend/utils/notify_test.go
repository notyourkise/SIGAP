package utils

import (
	"testing"

	"sigap-backend/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func newNotifyTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{SkipDefaultTransaction: true})
	assert.NoError(t, err)
	return gormDB, mock
}

func TestPersistNotificationSuccess(t *testing.T) {
	db, mock := newNotifyTestDB(t)
	notif := &models.Notification{
		UserID:  uuid.New(),
		Type:    models.NotificationTypeAssigned,
		Title:   "Tugas baru ditugaskan",
		Message: "Tugas dikerjakan",
	}

	mock.ExpectExec(`^INSERT INTO "notifications"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	assert.NoError(t, persistNotification(db, notif))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPersistNotificationRetriesAfterTransientFailure(t *testing.T) {
	db, mock := newNotifyTestDB(t)
	notif := &models.Notification{
		UserID:  uuid.New(),
		Type:    models.NotificationTypeCompleted,
		Title:   "Tugas selesai",
		Message: "Status Selesai",
	}

	mock.ExpectExec(`^INSERT INTO "notifications"`).WillReturnError(assert.AnError)
	mock.ExpectExec(`^INSERT INTO "notifications"`).WillReturnResult(sqlmock.NewResult(1, 1))

	assert.NoError(t, persistNotification(db, notif))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPersistNotificationReturnsErrorAfterRetryLimit(t *testing.T) {
	db, mock := newNotifyTestDB(t)
	notif := &models.Notification{
		UserID:  uuid.New(),
		Type:    models.NotificationTypeOverdue,
		Title:   "Tugas terlambat",
		Message: "Segera tindak lanjuti",
	}

	for i := 0; i < notifyWriteAttempts; i++ {
		mock.ExpectExec(`^INSERT INTO "notifications"`).WillReturnError(assert.AnError)
	}

	assert.Error(t, persistNotification(db, notif))
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestBroadcastToUserNoClients memastikan guard clause bekerja: tanpa koneksi
// WebSocket aktif, broadcast terarah tidak boleh panic ataupun error.
func TestBroadcastToUserNoClients(t *testing.T) {
	assert.NotPanics(t, func() {
		BroadcastToUser(uuid.NewString(), EventNotifCreated, models.Notification{})
	})
}

// TestBroadcastToRoleNoClients memastikan guard clause bekerja untuk broadcast
// terarah per-role tanpa koneksi aktif.
func TestBroadcastToRoleNoClients(t *testing.T) {
	assert.NotPanics(t, func() {
		BroadcastToRole("admin", EventNotifCreated, models.Notification{})
	})
}
