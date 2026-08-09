package handlers

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"sigap-backend/models"
	"sigap-backend/utils"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// userWithEmailRow membangun baris users berisi email_notifications.
func userWithEmailRow(id uuid.UUID, username, role, email string, emailNotif bool) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"username", "password_hash", "full_name", "email", "role",
		"token_version", "must_change_password", "email_notifications",
	}).AddRow(id, time.Now(), time.Now(), nil, username, "hash", "User Satu", email, role, 1, false, emailNotif)
}

// TestUpdateEmailPreferences_Success menguji memperbarui preferensi email.
func TestUpdateEmailPreferences_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	id := uuid.New()

	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnRows(userWithEmailRow(id, "admin1", "admin", "admin@sigap.local", true))
	mock.ExpectExec(`^UPDATE "users" SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	app := newTestApp()
	app.Put("/profile/email-preferences", withUserLocals(id.String(), "admin"), UpdateEmailPreferences)

	req := httptest.NewRequest("PUT", "/profile/email-preferences",
		strings.NewReader(`{"email_notifications":false}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	data := result["data"].(map[string]interface{})
	assert.Equal(t, false, data["email_notifications"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestUpdateEmailPreferences_MissingField memastikan field wajib.
func TestUpdateEmailPreferences_MissingField(t *testing.T) {
	_, mock := setupMockGORM(t)

	app := newTestApp()
	app.Put("/profile/email-preferences", withUserLocals(uuid.New().String(), "admin"), UpdateEmailPreferences)

	req := httptest.NewRequest("PUT", "/profile/email-preferences",
		strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestUpdateEmailPreferences_UserNotFound memastikan 404 untuk user tak ada.
func TestUpdateEmailPreferences_UserNotFound(t *testing.T) {
	_, mock := setupMockGORM(t)

	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnError(gorm.ErrRecordNotFound)

	app := newTestApp()
	app.Put("/profile/email-preferences", withUserLocals(uuid.New().String(), "admin"), UpdateEmailPreferences)

	req := httptest.NewRequest("PUT", "/profile/email-preferences",
		strings.NewReader(`{"email_notifications":true}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestSendCCTVStreamDownEmail_NoPanic memastikan fungsi email CCTV offline
// tidak panic bila SMTP nonaktif (no-op) maupun saat query admin berhasil.
func TestSendCCTVStreamDownEmail_NoPanic(t *testing.T) {
	// SMTP tidak dikonfigurasi → fungsi langsung no-op tanpa query DB.
	t.Setenv("SMTP_HOST", "")
	utils.InitEmailConfig()

	// Dengan db nil, fungsi tetap aman (db == nil → return).
	SendCCTVStreamDownEmail(nil, &models.CCTV{Nama: "Kamera Gerbang", KodeCCTV: "CCTV-001"})

	// Dengan DB tersedia tapi SMTP nonaktif, tidak terjadi query → tetap aman.
	gormDB, mock := setupMockGORM(t)
	cctv := models.CCTV{
		Nama:        "Kamera Gerbang",
		KodeCCTV:    "CCTV-001",
		LokasiArea:  "Area Utama",
		JenisCCTV:   "Dome",
		IPCamera:    "192.168.1.11",
		HealthScore: 80,
	}
	SendCCTVStreamDownEmail(gormDB, &cctv)
	assert.NoError(t, mock.ExpectationsWereMet())
}
