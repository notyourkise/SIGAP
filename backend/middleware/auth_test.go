package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"sigap-backend/config"
	"sigap-backend/utils"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("gagal membuat mock DB: %v", err)
	}
	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
	if err != nil {
		t.Fatalf("gagal membuka GORM: %v", err)
	}
	return gormDB, mock
}

func buildTokenRequest(t *testing.T, token string) *http.Request {
	t.Helper()
	req := httptest.NewRequest("GET", "/protected", nil)
	req.AddCookie(&http.Cookie{Name: utils.SessionCookieName, Value: token})
	return req
}

func TestProtected_NoToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "sigap_test_jwt_secret_key_2026")
	app := fiber.New()
	app.Get("/protected", Protected())

	resp, err := app.Test(httptest.NewRequest("GET", "/protected", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "error", result["status"])
}

func TestProtected_RevokedToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "sigap_test_jwt_secret_key_2026")
	app := fiber.New()
	app.Get("/protected", Protected())

	token, err := utils.GenerateToken("user-1", "admin")
	assert.NoError(t, err)
	utils.RevokeToken(token, time.Now().Add(time.Hour))

	resp, err := app.Test(buildTokenRequest(t, token))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "Token telah dicabut. Silakan login kembali.", result["message"])
}

func TestProtected_InvalidToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "sigap_test_jwt_secret_key_2026")
	app := fiber.New()
	app.Get("/protected", Protected())

	// Token ditandatangani dengan secret yang salah
	claims := jwt.MapClaims{
		"user_id": "user-1",
		"role":    "admin",
		"exp":     time.Now().Add(time.Hour).Unix(),
	}
	bad := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	badString, err := bad.SignedString([]byte("wrong_secret"))
	assert.NoError(t, err)

	resp, err := app.Test(buildTokenRequest(t, badString))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestProtected_ValidToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "sigap_test_jwt_secret_key_2026")
	gormDB, mock := setupMockDB(t)
	config.DB = gormDB

	userID := "user-123"

	// Ekspektasi query validasi versi token di database
	mock.ExpectQuery(`^SELECT "token_version",\s*"must_change_password" FROM "users"|^SELECT "token_version".*FROM "users"`).
		WithArgs(userID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"token_version", "must_change_password"}).AddRow(1, false))

	token, err := utils.GenerateToken(userID, "admin")
	assert.NoError(t, err)

	app := fiber.New()
	app.Get("/protected", Protected(), func(c *fiber.Ctx) error {
		uid, _ := c.Locals("user_id").(string)
		role, _ := c.Locals("role").(string)
		return c.JSON(fiber.Map{"uid": uid, "role": role})
	})

	resp, err := app.Test(buildTokenRequest(t, token))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, userID, result["uid"])
	assert.Equal(t, "admin", result["role"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestProtected_TokenVersionMismatch(t *testing.T) {
	t.Setenv("JWT_SECRET", "sigap_test_jwt_secret_key_2026")
	gormDB, mock := setupMockDB(t)
	config.DB = gormDB

	userID := "user-456"

	// Token lama tv=1, namun versi terbaru di DB adalah 2 (password sudah di-reset)
	mock.ExpectQuery(`^SELECT "token_version",\s*"must_change_password" FROM "users"|^SELECT "token_version".*FROM "users"`).
		WithArgs(userID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"token_version", "must_change_password"}).AddRow(2, false))

	token, err := utils.GenerateToken(userID, "admin")
	assert.NoError(t, err)

	app := fiber.New()
	app.Get("/protected", Protected(), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	resp, err := app.Test(buildTokenRequest(t, token))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "Sesi telah dicabut. Silakan login kembali.", result["message"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestProtected_MustChangePassword_RestrictsOperationalRoute(t *testing.T) {
	t.Setenv("JWT_SECRET", "sigap_test_jwt_secret_key_2026")
	gormDB, mock := setupMockDB(t)
	config.DB = gormDB

	userID := "user-must-change"

	mock.ExpectQuery(`^SELECT "token_version",\s*"must_change_password" FROM "users"|^SELECT "token_version".*FROM "users"`).
		WithArgs(userID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"token_version", "must_change_password"}).AddRow(1, true))

	token, err := utils.GenerateToken(userID, "admin")
	assert.NoError(t, err)

	app := fiber.New()
	app.Get("/operational", Protected(), RequireRole("admin"), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest("GET", "/operational", nil)
	req.AddCookie(&http.Cookie{Name: utils.SessionCookieName, Value: token})

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "error", result["status"])
	assert.Equal(t, "Anda wajib mengganti password sebelum dapat mengakses resource ini", result["message"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestProtected_MustChangePassword_AllowsProfileAndChangePassword(t *testing.T) {
	t.Setenv("JWT_SECRET", "sigap_test_jwt_secret_key_2026")
	gormDB, mock := setupMockDB(t)
	config.DB = gormDB

	userID := "user-must-change-allowed"

	// Mock 3 kali panggilan untuk 3 endpoint yang diizinkan
	for i := 0; i < 3; i++ {
		mock.ExpectQuery(`^SELECT "token_version",\s*"must_change_password" FROM "users"|^SELECT "token_version".*FROM "users"`).
			WithArgs(userID, 1).
			WillReturnRows(sqlmock.NewRows([]string{"token_version", "must_change_password"}).AddRow(1, true))
	}

	token, err := utils.GenerateToken(userID, "admin")
	assert.NoError(t, err)

	app := fiber.New()
	app.Get("/api/auth/me", Protected(), func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "profile_ok"})
	})
	app.Post("/api/auth/change-password", Protected(), func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "password_changed"})
	})
	app.Put("/api/auth/profile", Protected(), func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "profile_updated"})
	})

	// 1. Tes GET /api/auth/me
	req1 := httptest.NewRequest("GET", "/api/auth/me", nil)
	req1.AddCookie(&http.Cookie{Name: utils.SessionCookieName, Value: token})
	resp1, err := app.Test(req1)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp1.StatusCode)

	// 2. Tes POST /api/auth/change-password
	req2 := httptest.NewRequest("POST", "/api/auth/change-password", nil)
	req2.AddCookie(&http.Cookie{Name: utils.SessionCookieName, Value: token})
	resp2, err := app.Test(req2)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp2.StatusCode)

	// 3. Tes PUT /api/auth/profile
	req3 := httptest.NewRequest("PUT", "/api/auth/profile", nil)
	req3.AddCookie(&http.Cookie{Name: utils.SessionCookieName, Value: token})
	resp3, err := app.Test(req3)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp3.StatusCode)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRequireRole_NoRole(t *testing.T) {
	app := fiber.New()
	app.Get("/admin", RequireRole("admin"), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	resp, err := app.Test(httptest.NewRequest("GET", "/admin", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}

func TestRequireRole_WrongRole(t *testing.T) {
	app := fiber.New()
	app.Get("/admin", func(c *fiber.Ctx) error {
		c.Locals("role", "teknisi")
		return c.Next()
	}, RequireRole("admin"), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	resp, err := app.Test(httptest.NewRequest("GET", "/admin", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}

func TestRequireRole_Allowed(t *testing.T) {
	app := fiber.New()
	app.Get("/teknis", func(c *fiber.Ctx) error {
		c.Locals("role", "teknisi")
		return c.Next()
	}, RequireRole("admin", "teknisi"), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	resp, err := app.Test(httptest.NewRequest("GET", "/teknis", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}
