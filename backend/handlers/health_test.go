package handlers

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestHealthCheck_Connected(t *testing.T) {
	setupMockGORM(t)
	app := newTestApp()
	app.Get("/health", HealthCheck)

	resp, err := app.Test(httptest.NewRequest("GET", "/health", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "Healthy", result["api_status"])
	assert.Equal(t, "Connected", result["db_status"])
}

// TestHealthCheck_ExtendedFields memverifikasi perluasan GET /api/health
// (Task 7.4): response time DB, status Redis, dan cek folder uploads.
func TestHealthCheck_ExtendedFields(t *testing.T) {
	setupMockGORM(t)
	app := newTestApp()
	app.Get("/health", HealthCheck)

	resp, err := app.Test(httptest.NewRequest("GET", "/health", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))

	// DB response time harus berupa angka (>= 0).
	_, ok := result["db_response_time_ms"].(float64)
	assert.True(t, ok, "db_response_time_ms harus berupa angka")

	// Redis: dalam pengujian tanpa koneksi Redis aktif, status harus "Disabled".
	assert.Equal(t, "Disabled", result["redis_status"])
	_, ok = result["redis_response_time_ms"].(float64)
	assert.True(t, ok, "redis_response_time_ms harus berupa angka")

	// Cek folder uploads: objek berisi status, path, dan size_bytes.
	uploads, ok := result["uploads"].(map[string]interface{})
	assert.True(t, ok, "uploads harus berupa objek")
	assert.Equal(t, "OK", uploads["status"])
	assert.Equal(t, "./uploads", uploads["path"])
	_, ok = uploads["size_bytes"].(float64)
	assert.True(t, ok, "size_bytes harus berupa angka")
}

func TestDirSize(t *testing.T) {
	// Direktori tidak ada -> 0 tanpa error.
	size, err := dirSize("./tidak_ada_direktori_xyz")
	assert.NoError(t, err)
	assert.Equal(t, int64(0), size)

	// Direktori dengan file -> jumlah ukuran file.
	dir := filepath.Join(t.TempDir(), "sub")
	assert.NoError(t, os.MkdirAll(dir, 0o755))
	assert.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("12345"), 0o644)) // 5 byte
	assert.NoError(t, os.WriteFile(filepath.Join(dir, "b.txt"), []byte("123"), 0o644))   // 3 byte

	size, err = dirSize(filepath.Dir(dir))
	assert.NoError(t, err)
	assert.Equal(t, int64(8), size)
}
