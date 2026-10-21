package handlers

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetAllCCTV_FilterAndSortBySite memastikan filter kawasan diterapkan pada
// seluruh dataset dan pengurutan kawasan memakai nama site, bukan UUID.
func TestGetAllCCTV_FilterAndSortBySite(t *testing.T) {
	_, mock := setupMockGORM(t)
	siteID, cctvID := uuid.New(), uuid.New()

	mock.ExpectQuery(`^SELECT count\(\*\) FROM "cctvs" LEFT JOIN sites`).
		WithArgs(siteID.String()).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	cctvRows := sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at", "kode_cctv", "id_channel",
		"nama", "lokasi_area", "status", "health_score", "site_id",
	}).AddRow(cctvID, time.Now(), time.Now(), nil, "CCTV-KIPP-01", "CH-01",
		"Kamera Istana", "Sumbu Kebangsaan", "aktif", 100, siteID)
	mock.ExpectQuery(`^SELECT "cctvs"\."id".* FROM "cctvs" LEFT JOIN sites`).
		WithArgs(siteID.String(), 10).
		WillReturnRows(cctvRows)
	mock.ExpectQuery(`^SELECT \* FROM "sites" WHERE "sites"\."id" =`).
		WillReturnRows(siteRows(siteID, "Kawasan Inti Pusat Pemerintahan", "KIPP", false))

	app := newTestApp()
	app.Get("/cctv", func(c *fiber.Ctx) error {
		c.Locals("site_id", siteID.String())
		return c.Next()
	}, GetAllCCTV)

	req := httptest.NewRequest("GET", "/cctv?page=1&limit=10&sort_by=site_name&order=asc", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	data := result["data"].(map[string]interface{})
	rows := data["rows"].([]interface{})
	assert.Len(t, rows, 1)
	site := rows[0].(map[string]interface{})["site"].(map[string]interface{})
	assert.Equal(t, "Kawasan Inti Pusat Pemerintahan", site["nama"])
	assert.NoError(t, mock.ExpectationsWereMet())
}
