package handlers

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

// apiKeyRows membangun baris tabel api_keys untuk mock sqlmock.
func apiKeyRows(id uuid.UUID, name, prefix string, scopes []byte, createdBy uuid.UUID) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"name", "key_hash", "key_prefix", "scopes", "expires_at", "created_by", "last_used_at", "revoked_at",
	}).AddRow(id, time.Now(), time.Now(), nil, name, "hash", prefix, string(scopes), nil, createdBy, nil, nil)
}

// TestCreateAPIKey_Success menguji pembuatan API key (key ditampilkan 1×).
func TestCreateAPIKey_Success(t *testing.T) {
	_, mock := setupMockGORM(t)

	// Scopes []string disimpan sebagai jsonb → GORM memakai INSERT ... RETURNING
	// "scopes". Kembalikan []byte agar GORM dapat mem-parse JSON ke *[]string.
	mock.ExpectQuery(`^INSERT INTO "api_keys"`).
		WillReturnRows(sqlmock.NewRows([]string{"scopes"}).AddRow([]byte(`["cctv:read","health:read"]`)))

	app := newTestApp()
	app.Post("/api-keys", withUserLocals(uuid.New().String(), "admin"), CreateAPIKey)

	body := `{"name":"Integrasi Dashboard","scopes":["cctv:read","health:read"],"expires_in_days":30}`
	req := httptest.NewRequest("POST", "/api-keys", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	data := result["data"].(map[string]interface{})
	assert.NotEmpty(t, data["api_key"], "key mentah harus dikembalikan sekali")
	assert.NotEmpty(t, data["key_prefix"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCreateAPIKey_InvalidScope memastikan scope tidak valid ditolak (400).
func TestCreateAPIKey_InvalidScope(t *testing.T) {
	_, mock := setupMockGORM(t)

	app := newTestApp()
	app.Post("/api-keys", withUserLocals(uuid.New().String(), "admin"), CreateAPIKey)

	body := `{"name":"Integrasi","scopes":["invalid:scope"]}`
	req := httptest.NewRequest("POST", "/api-keys", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCreateAPIKey_ValidationError memastikan nama kosong ditolak.
func TestCreateAPIKey_ValidationError(t *testing.T) {
	_, mock := setupMockGORM(t)

	app := newTestApp()
	app.Post("/api-keys", withUserLocals(uuid.New().String(), "admin"), CreateAPIKey)

	req := httptest.NewRequest("POST", "/api-keys", strings.NewReader(`{"name":"","scopes":["cctv:read"]}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetAllAPIKeys_Success menguji daftar API key terpaginasi.
func TestGetAllAPIKeys_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	id, createdBy := uuid.New(), uuid.New()

	mock.ExpectQuery(`^SELECT count\(\*\) FROM "api_keys"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`SELECT \* FROM "api_keys"`).
		WillReturnRows(apiKeyRows(id, "Integrasi", "abc12345", []byte(`["cctv:read"]`), createdBy))
	mock.ExpectQuery(`SELECT \* FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "full_name", "username"}).AddRow(createdBy, "Admin", "admin"))

	app := newTestApp()
	app.Get("/api-keys", withUserLocals(uuid.New().String(), "admin"), GetAllAPIKeys)

	req := httptest.NewRequest("GET", "/api-keys?page=1&limit=10", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestRevokeAPIKey_Success menguji pencabutan API key.
func TestRevokeAPIKey_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	id, createdBy := uuid.New(), uuid.New()

	mock.ExpectQuery(`SELECT \* FROM "api_keys"`).
		WillReturnRows(apiKeyRows(id, "Integrasi", "abc12345", []byte(`["cctv:read"]`), createdBy))
	mock.ExpectExec(`^UPDATE "api_keys" SET "revoked_at"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	app := newTestApp()
	app.Post("/api-keys/:id/revoke", withUserLocals(uuid.New().String(), "admin"), RevokeAPIKey)

	req := httptest.NewRequest("POST", "/api-keys/"+id.String()+"/revoke", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestRevokeAPIKey_NotFound memastikan revoke key tidak ada → 404.
func TestRevokeAPIKey_NotFound(t *testing.T) {
	_, mock := setupMockGORM(t)

	mock.ExpectQuery(`SELECT \* FROM "api_keys"`).
		WillReturnError(assert.AnError)

	app := newTestApp()
	app.Post("/api-keys/:id/revoke", withUserLocals(uuid.New().String(), "admin"), RevokeAPIKey)

	req := httptest.NewRequest("POST", "/api-keys/"+uuid.New().String()+"/revoke", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGenerateAPIKeyFormat memastikan key & prefix format benar.
func TestGenerateAPIKeyFormat(t *testing.T) {
	key, prefix := generateAPIKey()
	assert.Len(t, key, 64)       // 32 byte hex
	assert.Len(t, prefix, 8)     // 8 karakter pertama
	assert.Equal(t, key[:8], prefix)

	// Hash bcrypt dari key dapat diverifikasi.
	hash, err := bcrypt.GenerateFromPassword([]byte(key), bcrypt.DefaultCost)
	assert.NoError(t, err)
	assert.NoError(t, bcrypt.CompareHashAndPassword(hash, []byte(key)))
}
