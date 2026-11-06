package handlers

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"sigap-backend/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAudit_MobileProfileContractMustMatchBackend mereproduksi dan memvalidasi
// skenario audit M13: endpoint /auth/me harus menyediakan properti data.user
// yang sesuai dengan konsumsi klien mobile saat bootstrap dan refresh sesi.
func TestAudit_MobileProfileContractMustMatchBackend(t *testing.T) {
	_, mock := setupMockGORM(t)
	uid := uuid.New()

	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WithArgs(uid.String(), 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "role", "username"}).AddRow(uid, "teknisi", "audit-tech"))

	app := newTestApp()
	app.Get("/auth/me", withUserLocals(uid.String(), "teknisi"), GetMyProfile)

	req := httptest.NewRequest("GET", "/auth/me", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var parsed struct {
		Data struct {
			User *models.User `json:"user"`
		} `json:"data"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&parsed))

	if parsed.Data.User == nil {
		t.Fatal("mobile AuthContext reads data.user; actual /auth/me response stores user directly in data, so data.user is undefined")
	}

	assert.Equal(t, uid, parsed.Data.User.ID)
	assert.Equal(t, "audit-tech", parsed.Data.User.Username)
	assert.Equal(t, "teknisi", parsed.Data.User.Role)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestM13_GetMyProfile_DualContract memastikan respons /auth/me menyajikan
// data profil lengkap baik secara nested di data.user (klien mobile) maupun
// secara flat di data (klien web/Next.js) demi backward compatibility.
func TestM13_GetMyProfile_DualContract(t *testing.T) {
	_, mock := setupMockGORM(t)
	uid := uuid.New()
	now := time.Now()

	userRow := sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"username", "password_hash", "full_name", "email",
		"role", "profile_photo_url", "token_version",
		"must_change_password", "email_notifications",
		"email_daily_digest", "email_overdue", "email_stock_low",
	}).AddRow(
		uid, now, now, nil,
		"teknisi_lapangan", "$2a$10$xyz", "Teknisi Lapangan", "teknisi@sigap.id",
		"teknisi", "/api/uploads/profile.jpg", 1,
		false, true,
		true, true, true,
	)

	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WithArgs(uid.String(), 1).
		WillReturnRows(userRow)

	app := newTestApp()
	app.Get("/auth/me", withUserLocals(uid.String(), "teknisi"), GetMyProfile)

	req := httptest.NewRequest("GET", "/auth/me", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "success", body["status"])

	data, ok := body["data"].(map[string]interface{})
	require.True(t, ok, "body.data harus bertipe map")

	// 1. Verifikasi kontrak klien web (flat fields pada data)
	assert.Equal(t, uid.String(), data["id"])
	assert.Equal(t, "teknisi_lapangan", data["username"])
	assert.Equal(t, "Teknisi Lapangan", data["full_name"])
	assert.Equal(t, "teknisi@sigap.id", data["email"])
	assert.Equal(t, "teknisi", data["role"])
	assert.Equal(t, false, data["must_change_password"])
	assert.Equal(t, "/api/uploads/profile.jpg", data["profile_photo_url"])

	// 2. Verifikasi kontrak klien mobile (nested user pada data.user)
	userNested, ok := data["user"].(map[string]interface{})
	require.True(t, ok, "data.user harus bertipe map")
	assert.Equal(t, uid.String(), userNested["id"])
	assert.Equal(t, "teknisi_lapangan", userNested["username"])
	assert.Equal(t, "Teknisi Lapangan", userNested["full_name"])
	assert.Equal(t, "teknisi@sigap.id", userNested["email"])
	assert.Equal(t, "teknisi", userNested["role"])
	assert.Equal(t, false, userNested["must_change_password"])

	assert.NoError(t, mock.ExpectationsWereMet())
}
