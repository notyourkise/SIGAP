package routes

import (
	"sigap-backend/handlers"
	"sigap-backend/middleware"

	"github.com/gofiber/fiber/v2"
)

// SetupStreamRoutes mendefinisikan rute modul Streaming CCTV (Task 9.3).
// Seluruh rute khusus admin: Protected + RequireRole("admin").
func SetupStreamRoutes(router fiber.Router) {
	streamGroup := router.Group("/streams", middleware.Protected(), middleware.RequireRole("admin"))

	// Rute STATIS didaftarkan sebelum rute dinamis (gotcha Fiber route ordering).
	streamGroup.Get("/status", handlers.GetStreamStatus)
	streamGroup.Post("/start-all", handlers.StartAllStreams)
	streamGroup.Post("/stop-all", handlers.StopAllStreams)

	// Rute DINAMIS dengan parameter :id.
	streamGroup.Post("/:id/start", handlers.StartStream)
	streamGroup.Post("/:id/stop", handlers.StopStream)
	streamGroup.Get("/:id/url", handlers.GetStreamURL)
}
