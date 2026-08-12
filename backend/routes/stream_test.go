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

// TestStreamRoutesRequireAuth memastikan endpoint streaming TIDAK lagi publik:
// tanpa sesi (tanpa identitas di Locals) harus ditolak dengan 401 oleh
// middleware Protected. Sebelumnya modul streaming dibuat publik; sesuai
// keputusan arsitektur (Akses Viewer: Admin Only) kini seluruh endpoint API
// kontrol stream wajib terautentikasi.
func TestStreamRoutesRequireAuth(t *testing.T) {
	app := fiber.New()

	// Simulasikan middleware Protected() yang MENOLAK akses tanpa identitas.
	app.Get("/streams/status",
		middleware.Protected(),
		handlersStubGetStreamStatus,
	)

	resp, err := app.Test(httptest.NewRequest("GET", "/streams/status", nil))
	assert.NoError(t, err)
	// Tanpa sesi: Protected() harus menolak dengan 401 (bukan 200/404/500).
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

// handlersStubGetStreamStatus adalah stub handler agar rute terdaftar dengan
// benar di atas; tidak akan pernah dipanggil karena Protected() menolak dulu.
func handlersStubGetStreamStatus(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusOK)
}
