package middleware

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"sigap-backend/config"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// withAuthLocals menyuntikkan konteks autentikasi (user_id + role) sebelum
// SiteContext, meniru perilaku middleware Protected().
func withAuthLocals(userID uuid.UUID, role string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Locals("user_id", userID.String())
		c.Locals("role", role)
		return c.Next()
	}
}

func userSiteCountRow(n int64) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"count"}).AddRow(n)
}

// TestSiteContext_TeknisiNotAssigned_Forbidden memastikan teknisi yang tidak
// ditugaskan ke site diminta ditolak (403).
func TestSiteContext_TeknisiNotAssigned_Forbidden(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	config.DB = gormDB
	defer func() { config.DB = nil }()

	siteID := uuid.New()
	userID := uuid.New()

	mock.ExpectQuery(`^SELECT \* FROM "sites"`).
		WillReturnRows(siteRow(siteID, "Kawasan Lain", "LAIN"))
	mock.ExpectQuery(`^SELECT count\(\*\) FROM "user_sites"`).
		WillReturnRows(userSiteCountRow(0))

	app := fiber.New()
	app.Get("/protected", withAuthLocals(userID, "teknisi"), SiteContext(), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("X-Site-ID", siteID.String())
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "Akses ditolak: Anda tidak ditugaskan ke site ini", result["message"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestSiteContext_TeknisiAssigned_OK memastikan teknisi yang ditugaskan ke site
// diminta diizinkan (200).
func TestSiteContext_TeknisiAssigned_OK(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	config.DB = gormDB
	defer func() { config.DB = nil }()

	siteID := uuid.New()
	userID := uuid.New()

	mock.ExpectQuery(`^SELECT \* FROM "sites"`).
		WillReturnRows(siteRow(siteID, "Kawasan OIKN", "OIKN"))
	mock.ExpectQuery(`^SELECT count\(\*\) FROM "user_sites"`).
		WillReturnRows(userSiteCountRow(1))

	app := fiber.New()
	app.Get("/protected", withAuthLocals(userID, "teknisi"), SiteContext(), func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"site_id": c.Locals("site_id")})
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("X-Site-ID", siteID.String())
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, siteID.String(), result["site_id"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestSiteContext_AdminBypass memastikan admin dapat mengakses site mana pun
// tanpa query user_sites.
func TestSiteContext_AdminBypass(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	config.DB = gormDB
	defer func() { config.DB = nil }()

	siteID := uuid.New()

	mock.ExpectQuery(`^SELECT \* FROM "sites"`).
		WillReturnRows(siteRow(siteID, "Kawasan OIKN", "OIKN"))

	app := fiber.New()
	app.Get("/protected", withAuthLocals(uuid.New(), "admin"), SiteContext(), func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"site_id": c.Locals("site_id")})
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("X-Site-ID", siteID.String())
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}
