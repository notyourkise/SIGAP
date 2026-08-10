package routes

import (
	"sigap-backend/handlers"
	"sigap-backend/middleware"

	"github.com/gofiber/fiber/v2"
)

// SetupAPIKeyRoutes mendefinisikan rute modul API Key (Task 12.1, Sprint 29).
// Seluruh operasi khusus admin: Protected + RequireRole("admin").
func SetupAPIKeyRoutes(router fiber.Router) {
	group := router.Group("/api-keys", middleware.Protected(), middleware.RequireRole("admin"))

	// Rute statis didaftarkan sebelum rute dinamis (gotcha Fiber route ordering).
	group.Get("/", handlers.GetAllAPIKeys)
	group.Post("/", handlers.CreateAPIKey)
	group.Post("/:id/revoke", handlers.RevokeAPIKey)
	group.Post("/:id/regenerate", handlers.RegenerateAPIKey)
}
