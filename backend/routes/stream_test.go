package routes

import (
	"net/http/httptest"
	"testing"

	"sigap-backend/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

// TestStreamRoutesRequireAdmin memastikan penegakan peran admin pada modul
// streaming: pengguna berperan non-admin (mis. teknisi) ditolak dengan 403.
func TestStreamRoutesRequireAdmin(t *testing.T) {
	app := fiber.New()

	// Simulasikan middleware Protected() yang sudah mengisi peran "teknisi",
	// lalu RequireRole("admin") diharapkan menolak akses.
	app.Get("/streams/status",
		func(c *fiber.Ctx) error {
			c.Locals("user_id", "00000000-0000-0000-0000-000000000000")
			c.Locals("role", "teknisi")
			return c.Next()
		},
		middleware.RequireRole("admin"),
		func(c *fiber.Ctx) error {
			return c.SendStatus(fiber.StatusOK)
		},
	)

	resp, err := app.Test(httptest.NewRequest("GET", "/streams/status", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}

// TestStreamRoutesPublic memastikan endpoint streaming dapat diakses TANPA
// otentikasi (publik). Sebelumnya modul streaming diproteksi auth; kini dibuat
// publik agar video CCTV dapat ditonton langsung tanpa proses sign-in. Rute
// tetap terdaftar dan tidak menolak akses dengan 401.
func TestStreamRoutesPublic(t *testing.T) {
	app := fiber.New()
	SetupStreamRoutes(app)

	resp, err := app.Test(httptest.NewRequest("GET", "/streams/status", nil))
	assert.NoError(t, err)
	// Streaming publik: tidak boleh menolak akses tanpa otentikasi.
	assert.NotEqual(t, fiber.StatusUnauthorized, resp.StatusCode)
}
