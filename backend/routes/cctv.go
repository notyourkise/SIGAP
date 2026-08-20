package routes

import (
	"sigap-backend/handlers"
	"sigap-backend/middleware"
	"sigap-backend/models"

	"github.com/gofiber/fiber/v2"
)

// SetupCCTVRoutes mendaftarkan rute untuk modul Master Data CCTV terenkripsi
func SetupCCTVRoutes(router fiber.Router) {
	cctvGroup := router.Group("/cctv")

	// SiteContext WAJIB dipasang agar filter kawasan berfungsi (Fix BB-04)
	// Middleware ini membaca X-Site-ID header dan validasi site di database
	siteMiddleware := middleware.SiteContext()

	// Rute list dapat diakses via sesi JWT (web) ATAU API key integrasi eksternal
	// (Task 12.1, Sprint 29) dengan scope cctv:read.
	cctvGroup.Use(siteMiddleware)
	cctvGroup.Get("/", middleware.APIKeyOrProtected(models.ScopeCCTVRead), handlers.GetAllCCTV)

	// Rute yang dapat diakses oleh Admin & Teknisi (via sesi JWT)
	cctvGroup.Get("/template", middleware.Protected(), handlers.DownloadTemplateCCTV)
	cctvGroup.Post("/scan", middleware.Protected(), handlers.ScanCCTV)
	cctvGroup.Get("/:id/qrcode", middleware.Protected(), handlers.GetCCTVQRCode)
	cctvGroup.Get("/:id/health-score", middleware.Protected(), handlers.GetCCTVHealthScore)
	cctvGroup.Get("/:id/snapshot", middleware.Protected(), handlers.GetCCTVSnapshot)
	cctvGroup.Get("/:id/timeline", middleware.Protected(), handlers.GetCCTVTimeline)
	cctvGroup.Get("/:id/health-history", middleware.Protected(), handlers.GetCCTVHealthHistory)
	cctvGroup.Get("/:id", middleware.Protected(), handlers.GetCCTVByID)

	// Rute yang hanya dapat diakses oleh Admin
	cctvGroup.Post("/", middleware.Protected(), middleware.RequireRole("admin"), handlers.CreateCCTV)
	cctvGroup.Post("/import", middleware.Protected(), middleware.RequireRole("admin"), handlers.ImportCCTV)
	cctvGroup.Post("/bulk-delete", middleware.Protected(), middleware.RequireRole("admin"), handlers.BulkDeleteCCTV)
	cctvGroup.Post("/:id/rotate-qr", middleware.Protected(), middleware.RequireRole("admin"), handlers.RotateQRCode)
	cctvGroup.Put("/:id", middleware.Protected(), middleware.RequireRole("admin"), handlers.UpdateCCTV)
	cctvGroup.Delete("/:id", middleware.Protected(), middleware.RequireRole("admin"), handlers.DeleteCCTV)
}
