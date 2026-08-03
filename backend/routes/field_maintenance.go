package routes

import (
	"sigap-backend/handlers"
	"sigap-backend/middleware"

	"github.com/gofiber/fiber/v2"
)

// SetupFieldMaintenanceRoutes mendefinisikan rute untuk modul pengecekan
// lapangan (field maintenance) yang dilakukan teknisi di lokasi.
func SetupFieldMaintenanceRoutes(router fiber.Router) {
	group := router.Group("/field-maintenance", middleware.Protected())

	// Rute yang dapat diakses oleh Admin & Teknisi
	group.Get("/", handlers.GetAllFieldMaintenances)
	group.Get("/:id", handlers.GetFieldMaintenanceByID)
	group.Post("/", handlers.CreateFieldMaintenance)

	// Rute yang hanya dapat diakses oleh Admin
	group.Delete("/:id", middleware.RequireRole("admin"), handlers.DeleteFieldMaintenance)
}
