package routes

import (
	"sigap-backend/handlers"
	"sigap-backend/middleware"

	"github.com/gofiber/fiber/v2"
)

// SetupCCTVRoutes mendaftarkan rute untuk modul Master Data CCTV terenkripsi
func SetupCCTVRoutes(router fiber.Router) {
	cctvGroup := router.Group("/cctv", middleware.Protected())

	// Rute yang dapat diakses oleh Admin & Teknisi
	cctvGroup.Get("/", handlers.GetAllCCTV)
	cctvGroup.Get("/template", handlers.DownloadTemplateCCTV)
	cctvGroup.Post("/scan", handlers.ScanCCTV)
	cctvGroup.Get("/:id/qrcode", handlers.GetCCTVQRCode)
	cctvGroup.Get("/:id", handlers.GetCCTVByID)

	// Rute yang hanya dapat diakses oleh Admin
	cctvGroup.Post("/", middleware.RequireRole("admin"), handlers.CreateCCTV)
	cctvGroup.Post("/import", middleware.RequireRole("admin"), handlers.ImportCCTV)
	cctvGroup.Post("/bulk-delete", middleware.RequireRole("admin"), handlers.BulkDeleteCCTV)
	cctvGroup.Put("/:id", middleware.RequireRole("admin"), handlers.UpdateCCTV)
	cctvGroup.Delete("/:id", middleware.RequireRole("admin"), handlers.DeleteCCTV)
}
