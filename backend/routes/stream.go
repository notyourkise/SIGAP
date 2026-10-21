package routes

import (
	"sigap-backend/handlers"
	"sigap-backend/middleware"

	"github.com/gofiber/fiber/v2"
)

// SetupStreamRoutes mendefinisikan rute modul Streaming CCTV (Task 9.3).
// Seluruh endpoint streaming bersifat ADMIN-ONLY (Protected + RequireRole)
// sesuai keputusan arsitektur ROADMAP ("Akses Viewer: Admin Only") dan aturan
// AGENTS.md. Akses video CCTV (WHEP/HLS) tetap melalui MediaMTX/Nginx yang
// terpisah; endpoint API kontrol stream tidak boleh diakses publik.
func SetupStreamRoutes(router fiber.Router) {
	streamGroup := router.Group("/streams", middleware.Protected(), middleware.RequireRole("admin"))

	// Rute STATIS didaftarkan sebelum rute dinamis (gotcha Fiber route ordering).
	streamGroup.Get("/status", handlers.GetStreamStatus)
	streamGroup.Get("/readiness", handlers.GetMediaMTXReadiness)
	streamGroup.Get("/health", handlers.GetStreamHealth)
	streamGroup.Get("/live-status", handlers.GetLiveStatus)
	streamGroup.Post("/start-all", handlers.StartAllStreams)
	streamGroup.Post("/stop-all", handlers.StopAllStreams)

	// Rute DINAMIS dengan parameter :id.
	streamGroup.Post("/:id/start", handlers.StartStream)
	streamGroup.Post("/:id/stop", handlers.StopStream)
	streamGroup.Get("/:id/url", handlers.GetStreamURL)
}
