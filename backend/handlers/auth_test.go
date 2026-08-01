package handlers

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"sigap-backend/config"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// SetupTestDB menyiapkan database GORM mock terisolasi
func SetupTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Gagal membuat stub database mock: %s", err)
	}

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{
		// SkipDefaultTransaction menghindari BEGIN/COMMIT otomatis GORM pada
		// operasi tulis sehingga ekspektasi sqlmock hanya berisi query yang
		// dipanggil handler (lebih sederhana & deterministik untuk diuji).
		SkipDefaultTransaction: true,
	})
	if err != nil {
		t.Fatalf("Gagal membuka koneksi database GORM: %s", err)
	}

	return gormDB, mock
}

func TestLogin_Success(t *testing.T) {
	// 1. Setup Mock DB
	gormDB, mock := SetupTestDB(t)
	config.DB = gormDB // Inject mock DB ke global config

	userID := uuid.New()
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("adminpassword"), bcrypt.DefaultCost)

	// Persiapkan baris database mock untuk dikembalikan oleh query
	rows := sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "username", "password_hash", "full_name", "email", "role"}).
		AddRow(userID, time.Now(), time.Now(), nil, "admin", string(hashedPassword), "Admin Haikal", "admin@sigap.local", "admin")

	// Set ekspektasi query GORM
	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WithArgs("admin", "admin", 1).
		WillReturnRows(rows)

	// 2. Setup Fiber Route
	app := fiber.New()
	app.Post("/api/auth/login", Login)

	// 3. Eksekusi Request
	payload := `{"username":"admin","password":"adminpassword"}`
	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)

	// 4. Asersi Hasil
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)

	assert.Equal(t, "success", result["status"])
	data, ok := result["data"].(map[string]interface{})
	assert.True(t, ok)
	// Token JWT tidak lagi dikembalikan di body — digantikan cookie httpOnly + csrf_token
	assert.Empty(t, data["token"])
	assert.NotEmpty(t, data["csrf_token"])

	userMap, ok := data["user"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "admin", userMap["username"])
	assert.Equal(t, "admin", userMap["role"])

	// Verifikasi cookie sesi httpOnly & cookie CSRF diset pada respons login
	cookies := resp.Cookies()
	foundSession := false
	foundCSRF := false
	for _, ck := range cookies {
		if ck.Name == "sigap_session" {
			foundSession = true
			assert.True(t, ck.HttpOnly, "cookie sesi wajib HttpOnly")
		}
		if ck.Name == "sigap_csrf" {
			foundCSRF = true
		}
	}
	assert.True(t, foundSession, "cookie sesi sigap_session harus diset")
	assert.True(t, foundCSRF, "cookie CSRF sigap_csrf harus diset")

	// Memastikan semua ekspektasi query mock terpenuhi
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLogin_WrongPassword(t *testing.T) {
	// 1. Setup Mock DB
	gormDB, mock := SetupTestDB(t)
	config.DB = gormDB

	userID := uuid.New()
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("adminpassword"), bcrypt.DefaultCost)

	rows := sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "username", "password_hash", "full_name", "email", "role"}).
		AddRow(userID, time.Now(), time.Now(), nil, "admin", string(hashedPassword), "Admin Haikal", "admin@sigap.local", "admin")

	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WithArgs("admin", "admin", 1).
		WillReturnRows(rows)

	// 2. Setup Fiber Route
	app := fiber.New()
	app.Post("/api/auth/login", Login)

	// 3. Eksekusi Request dengan Password Salah
	payload := `{"username":"admin","password":"wrongpassword"}`
	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)

	// 4. Asersi Hasil
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)

	assert.Equal(t, "error", result["status"])
	assert.Equal(t, "Username atau password salah", result["message"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLogin_UserNotFound(t *testing.T) {
	// 1. Setup Mock DB
	gormDB, mock := SetupTestDB(t)
	config.DB = gormDB

	// Set ekspektasi query GORM mengembalikan error GORM Record Not Found
	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WithArgs("nonexistent", "nonexistent", 1).
		WillReturnError(gorm.ErrRecordNotFound)

	// 2. Setup Fiber Route
	app := fiber.New()
	app.Post("/api/auth/login", Login)

	// 3. Eksekusi Request
	payload := `{"username":"nonexistent","password":"somepassword"}`
	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)

	// 4. Asersi Hasil
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)

	assert.Equal(t, "error", result["status"])
	assert.Equal(t, "Username atau password salah", result["message"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLogin_ValidationError(t *testing.T) {
	// Setup Fiber Route (tanpa butuh mock DB karena gagal di validasi input)
	app := fiber.New()
	app.Post("/api/auth/login", Login)

	// Request dengan body kosong
	payload := `{"username":"","password":""}`
	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)

	assert.Equal(t, "error", result["status"])
	assert.Equal(t, "Validasi input gagal", result["message"])
}
