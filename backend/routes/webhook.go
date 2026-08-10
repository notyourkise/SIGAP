package routes

import (
	"sigap-backend/handlers"
	"sigap-backend/middleware"

	"github.com/gofiber/fiber/v2"
)

// SetupWebhookRoutes mendefinisikan rute modul Webhook (Task 12.3, Sprint 29).
// Seluruh operasi khusus admin: Protected + RequireRole("admin").
func SetupWebhookRoutes(router fiber.Router) {
	group := router.Group("/webhooks", middleware.Protected(), middleware.RequireRole("admin"))

	// Rute statis didaftarkan sebelum rute dinamis (gotcha Fiber route ordering):
	// /:id/test & /:id/deliveries harus sebelum pola dinamis lain yang bentrok.
	group.Get("/", handlers.GetAllWebhookEndpoints)
	group.Post("/", handlers.CreateWebhookEndpoint)
	group.Get("/:id", handlers.GetWebhookEndpointByID)
	group.Put("/:id", handlers.UpdateWebhookEndpoint)
	group.Delete("/:id", handlers.DeleteWebhookEndpoint)
	group.Post("/:id/test", handlers.TestWebhook)
	group.Get("/:id/deliveries", handlers.GetWebhookDeliveries)
	group.Delete("/:id/deliveries", handlers.AdminCleanupWebhookDeliveries)
}
