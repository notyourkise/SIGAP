package routes

import (
	"sigap-backend/handlers"
	"sigap-backend/middleware"

	"github.com/gofiber/fiber/v2"
)

// SetupZoneRoutes mendefinisikan rute modul zona operasional (Task 11.4,
// Sprint 28). CRUD & assign khusus admin; daftar zona dapat diakses
// admin/teknisi yang sudah login.
func SetupZoneRoutes(router fiber.Router) {
	zoneGroup := router.Group("/zones", middleware.Protected())
	// SiteContext dipasang setelah Protected agar scope default non-admin
	// dapat ditentukan dari identitas pengguna.
	zoneGroup.Use(middleware.SiteContext())

	// Rute statis didaftarkan sebelum rute dinamis (gotcha Fiber route ordering):
	// /all harus sebelum /:id.
	zoneGroup.Get("/", handlers.GetAllZones)
	zoneGroup.Get("/all", handlers.GetAllZonesForSelect)
	zoneGroup.Get("/:id", handlers.GetZoneByID)

	// Operasi tulis (CRUD + assign) khusus admin.
	zoneGroup.Post("/", middleware.RequireRole("admin"), handlers.CreateZone)
	zoneGroup.Put("/:id", middleware.RequireRole("admin"), handlers.UpdateZone)
	zoneGroup.Delete("/:id", middleware.RequireRole("admin"), handlers.DeleteZone)
	zoneGroup.Post("/:id/assign-cctvs", middleware.RequireRole("admin"), handlers.AssignCCTVToZone)
	zoneGroup.Post("/:id/assign-users", middleware.RequireRole("admin"), handlers.AssignUsersToZone)
}
