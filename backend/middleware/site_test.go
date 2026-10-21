package middleware

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"sigap-backend/config"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// siteRow membangun satu baris tabel sites untuk mock sqlmock.
func siteRow(id uuid.UUID, nama, kode string) *sqlmock.Rows {
	return siteRowWithDefault(id, nama, kode, false)
}

// siteRowWithDefault membangun baris sites dengan status is_default tertentu.
func siteRowWithDefault(id uuid.UUID, nama, kode string, isDefault bool) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at", "nama", "kode", "alamat",
		"koordinat_lat", "koordinat_lng", "is_default",
	}).AddRow(id, time.Now(), time.Now(), nil, nama, kode, "", nil, nil, isDefault)
}

// TestSiteContext_NoHeader memastikan tanpa X-Site-ID → next dan locals site_id kosong.
func TestSiteContext_NoHeader(t *testing.T) {
	app := fiber.New()
	app.Get("/protected", SiteContext(), func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"site_id": c.Locals("site_id")})
	})

	resp, err := app.Test(httptest.NewRequest("GET", "/protected", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Nil(t, result["site_id"])
}

// TestSiteContext_NoHeader_NonAdminDefaultsToAssignedSite memastikan non-admin
// tanpa X-Site-ID / cookie mendapat scope site penugasannya, bukan seluruh data.
func TestSiteContext_NoHeader_NonAdminDefaultsToAssignedSite(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	config.DB = gormDB
	defer func() { config.DB = nil }()

	siteID := uuid.New()
	userID := uuid.New()

	mock.ExpectQuery(`FROM "sites" JOIN user_sites`).
		WithArgs(userID.String(), 1).
		WillReturnRows(siteRowWithDefault(siteID, "Kawasan OIKN", "OIKN", true))

	app := fiber.New()
	app.Get("/protected", withAuthLocals(userID, "teknisi"), SiteContext(), func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"site_id": c.Locals("site_id")})
	})

	resp, err := app.Test(httptest.NewRequest("GET", "/protected", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, siteID.String(), result["site_id"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestSiteContext_NoHeader_NonAdminWithoutAnySiteForbidden memastikan non-admin
// tanpa penugasan ditolak (fail closed), bukan melihat semua data.
func TestSiteContext_NoHeader_NonAdminWithoutAnySiteForbidden(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	config.DB = gormDB
	defer func() { config.DB = nil }()

	userID := uuid.New()
	emptyColumns := []string{
		"id", "created_at", "updated_at", "deleted_at", "nama", "kode", "alamat",
		"koordinat_lat", "koordinat_lng", "is_default",
	}
	mock.ExpectQuery(`FROM "sites" JOIN user_sites`).
		WithArgs(userID.String(), 1).
		WillReturnRows(sqlmock.NewRows(emptyColumns))

	app := fiber.New()
	app.Get("/protected", withAuthLocals(userID, "teknisi"), SiteContext(), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	resp, err := app.Test(httptest.NewRequest("GET", "/protected", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "Anda belum ditugaskan ke site mana pun", result["message"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestSiteContext_NoHeader_AdminRemainsGlobal memastikan admin tanpa site tetap
// dapat melihat seluruh kawasan (locals kosong).
func TestSiteContext_NoHeader_AdminRemainsGlobal(t *testing.T) {
	app := fiber.New()
	app.Get("/protected", withAuthLocals(uuid.New(), "admin"), SiteContext(), func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"site_id": c.Locals("site_id")})
	})

	resp, err := app.Test(httptest.NewRequest("GET", "/protected", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Nil(t, result["site_id"])
}

// TestSiteContext_HeaderValid memastikan X-Site-ID valid → locals terisi.
func TestSiteContext_HeaderValid(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	config.DB = gormDB
	defer func() { config.DB = nil }()

	siteID := uuid.New()
	mock.ExpectQuery(`^SELECT \* FROM "sites"`).
		WillReturnRows(siteRow(siteID, "Kawasan OIKN", "OIKN"))

	app := fiber.New()
	app.Get("/protected", SiteContext(), func(c *fiber.Ctx) error {
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

// TestSiteContext_QueryValid memastikan fallback ke ?site_id= berfungsi.
func TestSiteContext_QueryValid(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	config.DB = gormDB
	defer func() { config.DB = nil }()

	siteID := uuid.New()
	mock.ExpectQuery(`^SELECT \* FROM "sites"`).
		WillReturnRows(siteRow(siteID, "Kawasan OIKN", "OIKN"))

	app := fiber.New()
	app.Get("/protected", SiteContext(), func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"site_id": c.Locals("site_id")})
	})

	req := httptest.NewRequest("GET", "/protected?site_id="+siteID.String(), nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, siteID.String(), result["site_id"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestSiteContext_CookieValid memastikan fallback cookie sigap_site_id
// berfungsi bila header X-Site-ID tidak dikirim.
func TestSiteContext_CookieValid(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	config.DB = gormDB
	defer func() { config.DB = nil }()

	siteID := uuid.New()
	mock.ExpectQuery(`^SELECT \* FROM "sites"`).
		WillReturnRows(siteRow(siteID, "Kawasan OIKN", "OIKN"))

	app := fiber.New()
	app.Get("/protected", SiteContext(), func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"site_id": c.Locals("site_id")})
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Cookie", "sigap_site_id="+siteID.String())
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, siteID.String(), result["site_id"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestSiteContext_HeaderInvalid memastikan X-Site-ID bukan UUID → 400.
func TestSiteContext_HeaderInvalid(t *testing.T) {
	app := fiber.New()
	app.Get("/protected", SiteContext(), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("X-Site-ID", "bukan-uuid")
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "Format X-Site-ID tidak valid", result["message"])
}

// TestSiteContext_SiteNotFound memastikan site valid UUID tapi tidak ada di DB → 400.
func TestSiteContext_SiteNotFound(t *testing.T) {
	gormDB, mock := setupMockDB(t)
	config.DB = gormDB
	defer func() { config.DB = nil }()

	mock.ExpectQuery(`^SELECT \* FROM "sites"`).
		WillReturnError(gorm.ErrRecordNotFound)

	app := fiber.New()
	app.Get("/protected", SiteContext(), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("X-Site-ID", uuid.New().String())
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "Site tidak ditemukan", result["message"])
	assert.NoError(t, mock.ExpectationsWereMet())
}
