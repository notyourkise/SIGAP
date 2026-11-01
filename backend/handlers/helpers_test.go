package handlers

import (
	"os"
	"path/filepath"
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

// withUserSiteLocals mensimulasikan hasil Protected() + SiteContext() dengan
// site aktif yang sudah ditentukan. Dipakai untuk menguji filter multi-site.
func withUserSiteLocals(userID, role, siteID string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Locals("user_id", userID)
		c.Locals("role", role)
		c.Locals("site_id", siteID)
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
	ensureStandardTestUploadFixtures(t)
	gormDB, mock := SetupTestDB(t)
	config.DB = gormDB
	return gormDB, mock
}

func ensureStandardTestUploadFixtures(t *testing.T) {
	t.Helper()
	_ = os.MkdirAll(uploadsDir, 0755)
	fixtures := []string{"a.jpg", "b.jpg", "bukti1.jpg", "foto.jpg", "baru.jpg", "lama.jpg"}
	var created []string
	for _, name := range fixtures {
		p := filepath.Join(uploadsDir, name)
		if _, err := os.Lstat(p); os.IsNotExist(err) {
			if writeErr := os.WriteFile(p, []byte{0xff, 0xd8, 0xff, 0xd9}, 0644); writeErr == nil {
				created = append(created, p)
			}
		}
	}
	t.Cleanup(func() {
		for _, p := range created {
			_ = os.Remove(p)
		}
	})
}

// expectHealthScoreRecalculation menyiapkan ekspektasi query kalkulator yang
// dipanggil setelah complete atau field-check berhasil, sesuai urutan eksekusi
// (ordered matching sqlmock): (1) query metrik, (2) muat skor lama untuk hook
// webhook health_score.changed, (3) update skor.
func expectHealthScoreRecalculation(mock sqlmock.Sqlmock) {
	mock.ExpectQuery(`^SELECT\s+\(SELECT COUNT\(\*\) FROM maintenance_logs`).
		WillReturnRows(sqlmock.NewRows([]string{
			"failed", "attention", "overdue", "not_ok_items", "last_completed_at",
		}).AddRow(0, 0, 0, 0, nil))
	// Muat skor lama (Task 12.3) — skor 98 sama dengan hasil hitung untuk
	// metrik (0,0,0,0,nil) sehingga event health_score.changed tidak terpicu.
	mock.ExpectQuery(`SELECT "id","kode_cctv","nama","health_score" FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kode_cctv", "nama", "health_score"}).
			AddRow("00000000-0000-0000-0000-000000000000", "CCTV-001", "Kamera", 98))
	mock.ExpectExec(`^UPDATE "cctvs" SET "health_score"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
}
