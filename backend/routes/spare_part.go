package routes

import (
	"sigap-backend/handlers"
	"sigap-backend/middleware"

	"github.com/gofiber/fiber/v2"
)

// SetupSparePartRoutes mendefinisikan rute modul inventaris suku cadang
// (Task 11.1, Sprint 26). CRUD suku cadang khusus admin; daftar & penggunaan
// dapat diakses admin/teknisi yang sudah login.
func SetupSparePartRoutes(router fiber.Router) {
	// Rute suku cadang — daftar & detail untuk admin/teknisi, CRUD khusus admin.
	spGroup := router.Group("/spare-parts", middleware.Protected())

	// Rute statis didaftarkan sebelum rute dinamis (gotcha Fiber route ordering):
	// /low-stock harus sebelum /:id.
	spGroup.Get("/", handlers.GetAllSpareParts)
	spGroup.Get("/low-stock", handlers.GetLowStockSpareParts)
	spGroup.Get("/:id", handlers.GetSparePartByID)
	spGroup.Post("/", middleware.RequireRole("admin"), handlers.CreateSparePart)
	spGroup.Put("/:id", middleware.RequireRole("admin"), handlers.UpdateSparePart)
	spGroup.Delete("/:id", middleware.RequireRole("admin"), handlers.DeleteSparePart)

	// Penggunaan suku cadang — terikat maintenance log (admin & teknisi).
	mlGroup := router.Group("/maintenance-logs", middleware.Protected())
	mlGroup.Post("/:id/spare-parts", handlers.RecordSparePartUsage)
	mlGroup.Get("/:id/spare-parts", handlers.GetSparePartUsageByLog)
}
