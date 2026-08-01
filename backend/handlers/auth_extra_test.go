package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func validUserRow() (uuid.UUID, modelsUserRow) {
	uid := uuid.New()
	hash, _ := bcrypt.GenerateFromPassword([]byte("AdminPass123!"), bcrypt.MinCost)
	return uid, modelsUserRow{
		ID:        uid,
		Username:  "admin",
		Hash:      string(hash),
		FullName:  "Admin SIGAP",
		Email:     "admin@sigap.local",
		Role:      "admin",
		TokenVer:  1,
		MustChg:   false,
		CreatedAt: time.Now(),
	}
}

func TestChangePassword_Success(t *testing.T) {
	t.Setenv("JWT_SECRET", "sigap_test_jwt_secret_key_2026")
	_, mock := setupMockGORM(t)
	uid, row := validUserRow()

	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WithArgs(uid.String(), 1).
		WillReturnRows(newUserMockRows([]modelsUserRow{row}))
	mock.ExpectExec(`^UPDATE "users" SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	app := newTestApp()
	app.Post("/auth/change-password", withUserLocals(uid.String(), "admin"), ChangePassword)

	payload := `{"old_password":"AdminPass123!","new_password":"NewPass456@"}`
	req := httptest.NewRequest("POST", "/auth/change-password", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])

	// Setelah ganti password, cookie sesi baru + CSRF diset
	assert.NotEmpty(t, resp.Cookies())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestChangePassword_WrongOldPassword(t *testing.T) {
	_, mock := setupMockGORM(t)
	uid, row := validUserRow()

	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WithArgs(uid.String(), 1).
		WillReturnRows(newUserMockRows([]modelsUserRow{row}))

	app := newTestApp()
	app.Post("/auth/change-password", withUserLocals(uid.String(), "admin"), ChangePassword)

	payload := `{"old_password":"Salah123!","new_password":"NewPass456@"}`
	req := httptest.NewRequest("POST", "/auth/change-password", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "Password lama tidak sesuai", result["message"])
}

func TestChangePassword_ValidationError(t *testing.T) {
	app := newTestApp()
	app.Post("/auth/change-password", withUserLocals(uuid.New().String(), "admin"), ChangePassword)

	// Password baru terlalu lemah (tanpa huruf besar & simbol)
	payload := `{"old_password":"AdminPass123!","new_password":"password123"}`
	req := httptest.NewRequest("POST", "/auth/change-password", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "Validasi input gagal", result["message"])
}

func TestGetMyProfile_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	uid, row := validUserRow()

	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WithArgs(uid.String(), 1).
		WillReturnRows(newUserMockRows([]modelsUserRow{row}))

	app := newTestApp()
	app.Get("/auth/me", withUserLocals(uid.String(), "admin"), GetMyProfile)

	resp, err := app.Test(httptest.NewRequest("GET", "/auth/me", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	data := result["data"].(map[string]interface{})
	assert.Equal(t, "admin", data["username"])
	assert.Equal(t, "Admin SIGAP", data["full_name"])
	assert.Equal(t, "admin", data["role"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetMyProfile_NotFound(t *testing.T) {
	_, mock := setupMockGORM(t)
	uid := uuid.New()

	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WithArgs(uid.String(), 1).
		WillReturnError(gorm.ErrRecordNotFound)

	app := newTestApp()
	app.Get("/auth/me", withUserLocals(uid.String(), "admin"), GetMyProfile)

	resp, err := app.Test(httptest.NewRequest("GET", "/auth/me", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestGetMyProfile_NoIdentity(t *testing.T) {
	app := newTestApp()
	app.Get("/auth/me", GetMyProfile)

	resp, err := app.Test(httptest.NewRequest("GET", "/auth/me", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestUpdateMyProfile_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	uid, row := validUserRow()

	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WithArgs(uid.String(), 1).
		WillReturnRows(newUserMockRows([]modelsUserRow{row}))
	mock.ExpectExec(`^UPDATE "users" SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	app := newTestApp()
	app.Put("/auth/profile", withUserLocals(uid.String(), "admin"), UpdateMyProfile)

	req := httptest.NewRequest("PUT", "/auth/profile", strings.NewReader(`{"full_name":"Nama Baru"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	data := result["data"].(map[string]interface{})
	assert.Equal(t, "Nama Baru", data["full_name"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateMyProfile_ValidationError(t *testing.T) {
	app := newTestApp()
	app.Put("/auth/profile", withUserLocals(uuid.New().String(), "admin"), UpdateMyProfile)

	req := httptest.NewRequest("PUT", "/auth/profile", strings.NewReader(`{"full_name":"x"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestLogout_NoToken(t *testing.T) {
	app := newTestApp()
	app.Post("/auth/logout", Logout)

	resp, err := app.Test(httptest.NewRequest("POST", "/auth/logout", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestLogout_InvalidToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "sigap_test_jwt_secret_key_2026")
	app := newTestApp()
	app.Post("/auth/logout", Logout)

	// Token sudah tidak valid → dianggap logout berhasil
	req := httptest.NewRequest("POST", "/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "sigap_session", Value: "garbage.token.here"})

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
}

func TestCreateUser_ValidationError(t *testing.T) {
	app := newTestApp()
	app.Post("/auth/users", withUserLocals(uuid.New().String(), "admin"), CreateUser)

	// Password tidak memenuhi kebijakan strong password
	payload := `{"username":"newtech","password":"lemah","full_name":"Teknisi Baru","email":"tek@sigap.local","role":"teknisi"}`
	req := httptest.NewRequest("POST", "/auth/users", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestCreateUser_DuplicateUsername(t *testing.T) {
	_, mock := setupMockGORM(t)
	uid := uuid.New()

	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnRows(newUserMockRows([]modelsUserRow{{
			ID: uid, Username: "existing", Hash: "x", FullName: "Ada", Email: "ada@sigap.local", Role: "teknisi", TokenVer: 1, CreatedAt: time.Now(),
		}}))

	app := newTestApp()
	app.Post("/auth/users", withUserLocals(uuid.New().String(), "admin"), CreateUser)

	payload := `{"username":"existing","password":"Kuat123!","full_name":"Duplikat","email":"baru@sigap.local","role":"teknisi"}`
	req := httptest.NewRequest("POST", "/auth/users", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "Username atau Email sudah terdaftar", result["message"])
}

func TestCreateUser_Success(t *testing.T) {
	_, mock := setupMockGORM(t)

	// Tidak ada user dengan username/email yang sama
	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnError(gorm.ErrRecordNotFound)
	mock.ExpectExec(`^INSERT INTO "users"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	app := newTestApp()
	app.Post("/auth/users", withUserLocals(uuid.New().String(), "admin"), CreateUser)

	payload := `{"username":"newtech","password":"Kuat123!","full_name":"Teknisi Baru","email":"tek@sigap.local","role":"teknisi"}`
	req := httptest.NewRequest("POST", "/auth/users", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	data := result["data"].(map[string]interface{})
	assert.Equal(t, "newtech", data["username"])
	assert.Equal(t, "teknisi", data["role"])
	assert.NoError(t, mock.ExpectationsWereMet())
}
