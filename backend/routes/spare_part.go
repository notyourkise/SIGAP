package routes

import (
	"sigap-backend/handlers"
	"sigap-backend/middleware"

	"github.com/gofiber/fiber/v2"
)

// SetupSparePartRoutes mendefinisikan rute modul inventaris suku cadang
// (Task 11.1, Sprint 26). Seluruh inventaris suku cadang khusus admin;
// teknisi hanya dapat mencatat penggunaan melalui pekerjaan pemeliharaan.
func SetupSparePartRoutes(router fiber.Router) {
	// SiteContext WAJIB dipasang agar filter kawasan berfungsi (Fix BB-19)
	// Middleware ini membaca X-Site-ID header dan validasi site di database
	siteMiddleware := middleware.SiteContext()

	// Rute inventaris suku cadang (daftar, detail, dan CRUD) khusus admin.
	spGroup := router.Group("/spare-parts", middleware.Protected(), middleware.RequireRole("admin"))
	spGroup.Use(siteMiddleware)

	// Rute statis didaftarkan sebelum rute dinamis (gotcha Fiber route ordering):
	// /low-stock harus sebelum /:id.
	spGroup.Get("/", handlers.GetAllSpareParts)
	spGroup.Get("/low-stock", handlers.GetLowStockSpareParts)
	spGroup.Get("/:id", handlers.GetSparePartByID)
	spGroup.Post("/", handlers.CreateSparePart)
	spGroup.Put("/:id", handlers.UpdateSparePart)
	spGroup.Delete("/:id", handlers.DeleteSparePart)

	// Penggunaan suku cadang — terikat maintenance log (admin & teknisi).
	mlGroup := router.Group("/maintenance-logs", middleware.Protected())
	mlGroup.Use(middleware.SiteContext()) // Add site context here too
	mlGroup.Post("/:id/spare-parts", handlers.RecordSparePartUsage)
	mlGroup.Get("/:id/spare-parts", handlers.GetSparePartUsageByLog)
}
