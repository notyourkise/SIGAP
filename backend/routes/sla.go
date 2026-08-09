package routes

import (
	"sigap-backend/handlers"
	"sigap-backend/middleware"

	"github.com/gofiber/fiber/v2"
)

// SetupSLARoutes mendefinisikan rute modul SLA (Task 11.5, Sprint 28).
// CRUD kebijakan SLA khusus admin; daftar kebijakan dapat diakses
// admin/teknisi yang sudah login.
func SetupSLARoutes(router fiber.Router) {
	policyGroup := router.Group("/sla-policies", middleware.Protected())

	// Rute statis didaftarkan sebelum rute dinamis (gotcha Fiber route ordering).
	policyGroup.Get("/", handlers.GetSLAPolicies)
	policyGroup.Get("/:id", handlers.GetSLAPolicyByID)

	// Operasi tulis (CRUD) khusus admin.
	policyGroup.Post("/", middleware.RequireRole("admin"), handlers.CreateSLAPolicy)
	policyGroup.Put("/:id", middleware.RequireRole("admin"), handlers.UpdateSLAPolicy)
	policyGroup.Delete("/:id", middleware.RequireRole("admin"), handlers.DeleteSLAPolicy)
}
