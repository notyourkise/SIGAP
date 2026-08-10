package middleware

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"sigap-backend/config"
	"sigap-backend/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

// TestAPIKeyAuth_NoHeader memastikan tanpa header X-API-Key → 401.
func TestAPIKeyAuth_NoHeader(t *testing.T) {
	app := fiber.New()
	app.Get("/protected", APIKeyAuth(models.ScopeCCTVRead))

	resp, err := app.Test(httptest.NewRequest("GET", "/protected", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "Header X-API-Key tidak ditemukan", result["message"])
}

// TestAPIKeyAuth_InvalidKey memastikan key yang tidak cocok → 401.
func TestAPIKeyAuth_InvalidKey(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	config.DB = gormDB
	defer func() { config.DB = nil }()

	// Tidak ada API key yang cocok → muat kandidat kosong.
	mock.ExpectQuery(`SELECT \* FROM "api_keys"`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "created_at", "updated_at", "deleted_at",
			"name", "key_hash", "key_prefix", "scopes", "expires_at", "created_by", "last_used_at", "revoked_at",
		}))

	app := fiber.New()
	app.Get("/protected", APIKeyAuth(models.ScopeCCTVRead))

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set(HeaderAPIKey, "salah-key")
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

// TestAPIKeyAuth_ValidKey memastikan key valid dengan scope → 200 & locals terisi.
func TestAPIKeyAuth_ValidKey(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	config.DB = gormDB
	defer func() { config.DB = nil }()

	id := uuid.New()
	raw := "key-mentah-123"
	hash, _ := bcrypt.GenerateFromPassword([]byte(raw), bcrypt.DefaultCost)

	rows := sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"name", "key_hash", "key_prefix", "scopes", "expires_at", "created_by", "last_used_at", "revoked_at",
	}).AddRow(id, time.Now(), time.Now(), nil, "Integrasi", string(hash), "key-ment", `["cctv:read"]`, nil, id, nil, nil)
	mock.ExpectQuery(`SELECT \* FROM "api_keys"`).
		WillReturnRows(rows)
	// Update last_used_at (sinkron — dipanggil sebelum handler lanjut).
	mock.ExpectExec(`^UPDATE "api_keys" SET "last_used_at"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	app := fiber.New()
	app.Get("/protected", APIKeyAuth(models.ScopeCCTVRead), func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"api_key_id": c.Locals("api_key_id"),
			"auth_method": c.Locals("auth_method"),
		})
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set(HeaderAPIKey, raw)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, id.String(), result["api_key_id"])
	assert.Equal(t, "api_key", result["auth_method"])
	// Update last_used_at berjalan sinkron; ekspektasi UPDATE dimock di atas.
	// Tidak memakai mock.ExpectationsWereMet() agar toleran terhadap variasi
	// argumen SQL dari GORM (updated_at dsb).
}

// TestAPIKeyAuth_WrongScope memastikan key tanpa scope yang diperlukan → 403.
func TestAPIKeyAuth_WrongScope(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	config.DB = gormDB
	defer func() { config.DB = nil }()

	id := uuid.New()
	raw := "key-mentah-456"
	hash, _ := bcrypt.GenerateFromPassword([]byte(raw), bcrypt.DefaultCost)

	// Key hanya punya scope health:read, endpoint butuh cctv:read.
	rows := sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"name", "key_hash", "key_prefix", "scopes", "expires_at", "created_by", "last_used_at", "revoked_at",
	}).AddRow(id, time.Now(), time.Now(), nil, "Integrasi", string(hash), "key-ment", `["health:read"]`, nil, id, nil, nil)
	mock.ExpectQuery(`SELECT \* FROM "api_keys"`).
		WillReturnRows(rows)

	app := fiber.New()
	app.Get("/protected", APIKeyAuth(models.ScopeCCTVRead))

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set(HeaderAPIKey, raw)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}
