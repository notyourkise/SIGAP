package routes

import (
	"sigap-backend/handlers"
	"sigap-backend/middleware"

	"github.com/gofiber/fiber/v2"
)

// SetupSiteRoutes mendefinisikan rute modul kawasan operasional multi-site
// (Task 12.2, Sprint 30). CRUD & assign khusus admin; daftar site dapat
// diakses semua role yang sudah login.
func SetupSiteRoutes(router fiber.Router) {
	group := router.Group("/sites", middleware.Protected(), middleware.SiteContext())

	// Rute statis didaftarkan sebelum rute dinamis (gotcha Fiber route ordering):
	// /all harus sebelum /:id.
	group.Get("/", handlers.GetAllSites)
	group.Get("/all", handlers.GetAllSitesForSelect)
	group.Get("/:id", handlers.GetSiteByID)

	// Operasi tulis (CRUD + assign) khusus admin.
	group.Post("/", middleware.RequireRole("admin"), handlers.CreateSite)
	group.Post("/:id/assign-users", middleware.RequireRole("admin"), handlers.AssignUsersToSite)
	group.Put("/:id", middleware.RequireRole("admin"), handlers.UpdateSite)
	group.Delete("/:id", middleware.RequireRole("admin"), handlers.DeleteSite)
}
