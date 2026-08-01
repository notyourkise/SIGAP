package utils

import (
	"testing"

	"sigap-backend/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func newAuditTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{SkipDefaultTransaction: true})
	assert.NoError(t, err)
	return gormDB, mock
}

func TestPersistAuditLogRetriesAfterTransientFailure(t *testing.T) {
	db, mock := newAuditTestDB(t)
	logData := &models.AuditLog{Action: "LOGIN", Module: "AUTH", Username: "tester", UserRole: "admin"}

	mock.ExpectExec(`^INSERT INTO "audit_logs"`).WillReturnError(assert.AnError)
	mock.ExpectExec(`^INSERT INTO "audit_logs"`).WillReturnResult(sqlmock.NewResult(1, 1))

	assert.NoError(t, persistAuditLog(db, logData))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPersistAuditLogReturnsErrorAfterRetryLimit(t *testing.T) {
	db, mock := newAuditTestDB(t)
	logData := &models.AuditLog{Action: "LOGIN", Module: "AUTH", Username: "tester", UserRole: "admin"}

	for i := 0; i < auditWriteAttempts; i++ {
		mock.ExpectExec(`^INSERT INTO "audit_logs"`).WillReturnError(assert.AnError)
	}

	assert.Error(t, persistAuditLog(db, logData))
	assert.NoError(t, mock.ExpectationsWereMet())
}
