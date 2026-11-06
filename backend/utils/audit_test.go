package utils

import (
	"net/http/httptest"
	"testing"

	"sigap-backend/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
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

func TestLogAudit_AutoExtractUserAndRoleFromLocals(t *testing.T) {
	db, _ := newAuditTestDB(t)
	app := fiber.New()

	expectedActorID := uuid.New()
	expectedRole := "admin"

	var capturedLog models.AuditLog
	cleanup := SetAuditHookForTest(func(l models.AuditLog) {
		capturedLog = l
	})
	defer cleanup()

	app.Get("/test-audit", func(c *fiber.Ctx) error {
		c.Locals("user_id", expectedActorID.String())
		c.Locals("role", expectedRole)

		LogAudit(db, c, "TEST_ACTION", "TEST_MODULE", "Testing auto extract", nil, "", "")
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test-audit", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	assert.NotNil(t, capturedLog.UserID, "UserID harus diekstrak dari c.Locals saat bernilai nil")
	assert.Equal(t, expectedActorID, *capturedLog.UserID)
	assert.Equal(t, expectedRole, capturedLog.UserRole)
	assert.Equal(t, "TEST_ACTION", capturedLog.Action)
	assert.Equal(t, "TEST_MODULE", capturedLog.Module)
}

func TestLogAudit_NilContextSafe(t *testing.T) {
	db, _ := newAuditTestDB(t)

	var capturedLog models.AuditLog
	cleanup := SetAuditHookForTest(func(l models.AuditLog) {
		capturedLog = l
	})
	defer cleanup()

	customID := uuid.New()
	assert.NotPanics(t, func() {
		LogAudit(db, nil, "SYSTEM_ACTION", "SYSTEM", "Nil context call", &customID, "sysadmin", "admin")
	})

	assert.NotNil(t, capturedLog.UserID)
	assert.Equal(t, customID, *capturedLog.UserID)
	assert.Equal(t, "sysadmin", capturedLog.Username)
	assert.Equal(t, "admin", capturedLog.UserRole)
}
