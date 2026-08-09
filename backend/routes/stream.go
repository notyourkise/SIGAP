package routes

import (
	"sigap-backend/handlers"

	"github.com/gofiber/fiber/v2"
)

// SetupStreamRoutes mendefinisikan rute modul Streaming CCTV (Task 9.3).
// Streaming dibuat dapat diakses tanpa login (publik) agar video CCTV dapat
// ditonton langsung tanpa proses sign-in. Endpoint lain (master data, dsb.)
// tetap dilindungi autentikasi seperti biasa.
func SetupStreamRoutes(router fiber.Router) {
	streamGroup := router.Group("/streams")

	// Rute STATIS didaftarkan sebelum rute dinamis (gotcha Fiber route ordering).
	streamGroup.Get("/status", handlers.GetStreamStatus)
	streamGroup.Get("/health", handlers.GetStreamHealth)
	streamGroup.Post("/start-all", handlers.StartAllStreams)
	streamGroup.Post("/stop-all", handlers.StopAllStreams)

	// Rute DINAMIS dengan parameter :id.
	streamGroup.Post("/:id/start", handlers.StartStream)
	streamGroup.Post("/:id/stop", handlers.StopStream)
	streamGroup.Get("/:id/url", handlers.GetStreamURL)
}
