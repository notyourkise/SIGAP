package handlers

import (
	"testing"
	"time"

	"sigap-backend/config"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SetupTestDB disediakan oleh auth_test.go (package handlers).
// withUserLocals membuat middleware yang mengisi identitas user ke Locals,
// mensimulasikan hasil middleware Protected().
func withUserLocals(userID, role string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Locals("user_id", userID)
		c.Locals("role", role)
		return c.Next()
	}
}

// newTestApp membuat aplikasi Fiber untuk pengujian handler.
func newTestApp() *fiber.App {
	return fiber.New()
}

type modelsUserRow struct {
	ID        uuid.UUID
	Username  string
	Hash      string
	FullName  string
	Email     string
	Role      string
	TokenVer  int64
	MustChg   bool
	CreatedAt time.Time
}

func newUserMockRows(rows []modelsUserRow) *sqlmock.Rows {
	out := sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"username", "password_hash", "full_name", "email", "role",
		"token_version", "must_change_password",
	})
	for _, r := range rows {
		out.AddRow(r.ID, r.CreatedAt, r.CreatedAt, nil, r.Username, r.Hash, r.FullName, r.Email, r.Role, r.TokenVer, r.MustChg)
	}
	return out
}

// setupMockGORM menyiapkan GORM + sqlmock dan meng-inject ke config.DB.
func setupMockGORM(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	gormDB, mock := SetupTestDB(t)
	config.DB = gormDB
	return gormDB, mock
}

// expectHealthScoreRecalculation menyiapkan ekspektasi query kalkulator yang
// dipanggil setelah complete atau field-check berhasil.
func expectHealthScoreRecalculation(mock sqlmock.Sqlmock) {
	mock.ExpectQuery(`^SELECT\s+\(SELECT COUNT\(\*\) FROM maintenance_logs`).
		WillReturnRows(sqlmock.NewRows([]string{
			"failed", "attention", "overdue", "not_ok_items", "last_completed_at",
		}).AddRow(0, 0, 0, 0, nil))
	mock.ExpectExec(`^UPDATE "cctvs" SET "health_score"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
}
