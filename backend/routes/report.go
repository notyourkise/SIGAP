package routes

import (
	"sigap-backend/handlers"
	"sigap-backend/middleware"

	"github.com/gofiber/fiber/v2"
)

// SetupReportRoutes mendefinisikan rute report builder kustom (Task 11.2).
// Seluruh rute khusus admin: Protected + RequireRole("admin").
func SetupReportRoutes(router fiber.Router) {
	reportGroup := router.Group("/reports", middleware.Protected(), middleware.RequireRole("admin"))

	// Rute statis didaftarkan sebelum rute dinamis (gotcha Fiber route ordering):
	// /generate & /templates sebelum /templates/:id.
	reportGroup.Post("/generate", handlers.GenerateReport)
	reportGroup.Get("/templates", handlers.ListReportTemplates)
	reportGroup.Post("/templates", handlers.SaveReportTemplate)
	reportGroup.Delete("/templates/:id", handlers.DeleteReportTemplate)
}
