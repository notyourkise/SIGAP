package utils

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

// paginationTestHandler mengekspos parameter pagination sebagai JSON agar mudah diuji.
func paginationTestHandler(c *fiber.Ctx) error {
	limit, page, sort := GetPaginationParams(c, []string{"created_at", "username", "role"})
	return c.JSON(fiber.Map{"limit": limit, "page": page, "sort": sort})
}

// dateRangeTestHandler mengekspos hasil ParseDateRangeParams sebagai JSON agar mudah diuji.
func dateRangeTestHandler(c *fiber.Ctx) error {
	start, end := ParseDateRangeParams(c)
	var startStr, endStr string
	if start != nil {
		startStr = start.Format(time.RFC3339)
	}
	if end != nil {
		endStr = end.Format(time.RFC3339)
	}
	return c.JSON(fiber.Map{"start": startStr, "end": endStr})
}

func doPaginationRequest(t *testing.T, url string) (map[string]interface{}, int) {
	t.Helper()
	app := fiber.New()
	app.Get("/", paginationTestHandler)

	req := httptest.NewRequest("GET", url, nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	return result, resp.StatusCode
}

func TestGetPaginationParams_Defaults(t *testing.T) {
	result, status := doPaginationRequest(t, "/")
	assert.Equal(t, fiber.StatusOK, status)
	assert.Equal(t, float64(10), result["limit"])
	assert.Equal(t, float64(1), result["page"])
	assert.Equal(t, "created_at desc", result["sort"])
}

func TestGetPaginationParams_LimitCaps(t *testing.T) {
	// Batas atas: maksimal 100 agar tidak bisa dump seluruh tabel
	result, _ := doPaginationRequest(t, "/?limit=1000")
	assert.Equal(t, float64(100), result["limit"])

	// Nilai invalid / nol / negatif: fallback ke default 10
	for _, bad := range []string{"0", "-5", "abc"} {
		result, _ = doPaginationRequest(t, "/?limit="+bad)
		assert.Equal(t, float64(10), result["limit"], "limit=%s harus fallback ke 10", bad)
	}
}

func TestGetPaginationParams_PageCaps(t *testing.T) {
	// Batas atas: 1.000.000 agar offset tidak membesar tak terkendali
	result, _ := doPaginationRequest(t, "/?page=99999999")
	assert.Equal(t, float64(1000000), result["page"])

	// Nilai invalid: fallback ke 1
	for _, bad := range []string{"0", "-1", "xyz"} {
		result, _ = doPaginationRequest(t, "/?page="+bad)
		assert.Equal(t, float64(1), result["page"], "page=%s harus fallback ke 1", bad)
	}
}

func TestGetPaginationParams_SortWhitelist(t *testing.T) {
	// Kolom yang diizinkan
	result, _ := doPaginationRequest(t, "/?sort_by=username")
	assert.Equal(t, "username desc", result["sort"])

	result, _ = doPaginationRequest(t, "/?sort_by=role&order=asc")
	assert.Equal(t, "role asc", result["sort"])

	// Kolom yang tidak ada di whitelist → fallback created_at (cegah SQL injection di Order)
	result, _ = doPaginationRequest(t, "/?sort_by=id;DROP%20TABLE%20users")
	assert.Equal(t, "created_at desc", result["sort"])
}

func TestGetPaginationParams_InvalidOrder(t *testing.T) {
	result, _ := doPaginationRequest(t, "/?order=sideways")
	assert.Equal(t, "created_at desc", result["sort"])
}

func TestCalculatePagination(t *testing.T) {
	// Tidak ada data → minimal 1 halaman
	meta := CalculatePagination(0, 1, 10)
	assert.Equal(t, int64(0), meta.TotalRows)
	assert.Equal(t, 1, meta.TotalPages)
	assert.Equal(t, 1, meta.Page)
	assert.Equal(t, 10, meta.Limit)

	// 25 baris / limit 10 → 3 halaman
	meta = CalculatePagination(25, 2, 10)
	assert.Equal(t, 3, meta.TotalPages)
	assert.Equal(t, 2, meta.Page)

	// Kelipatan tepat → 10 baris / limit 5 → 2 halaman
	meta = CalculatePagination(10, 1, 5)
	assert.Equal(t, 2, meta.TotalPages)
}

func TestParseDateRangeParams_Valid(t *testing.T) {
	t.Setenv("APP_TIMEZONE", "Asia/Makassar")
	app := fiber.New()
	app.Get("/", dateRangeTestHandler)
	req := httptest.NewRequest("GET", "/?start_date=2026-01-01&end_date=2026-01-15", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	// Mulai dari tengah malam WITA (+08:00) tanggal awal
	assert.Equal(t, "2026-01-01T00:00:00+08:00", result["start"])
	// Akhir di-increment satu hari agar mencakup seluruh tanggal 15 Januari WITA
	assert.Equal(t, "2026-01-16T00:00:00+08:00", result["end"])
}

func TestParseDateRangeParams_InvalidAndMissing(t *testing.T) {
	t.Setenv("APP_TIMEZONE", "Asia/Makassar")
	app := fiber.New()
	app.Get("/", dateRangeTestHandler)

	// Tanpa parameter sama sekali → keduanya nil
	for _, url := range []string{"/", "/?end_date=2026-13-99"} {
		req := httptest.NewRequest("GET", url, nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)

		var result map[string]interface{}
		assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		assert.Equal(t, "", result["start"])
		assert.Equal(t, "", result["end"])
	}

	// start_date invalid → diabaikan, namun end_date valid tetap diproses
	req := httptest.NewRequest("GET", "/?start_date=not-a-date&end_date=2026-01-01", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "", result["start"])
	assert.Equal(t, "2026-01-02T00:00:00+08:00", result["end"])
}

func TestParseDateRangeParams_OneSided(t *testing.T) {
	t.Setenv("APP_TIMEZONE", "Asia/Makassar")
	app := fiber.New()
	app.Get("/", dateRangeTestHandler)
	req := httptest.NewRequest("GET", "/?start_date=2026-05-20", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "2026-05-20T00:00:00+08:00", result["start"])
	assert.Equal(t, "", result["end"])
}
