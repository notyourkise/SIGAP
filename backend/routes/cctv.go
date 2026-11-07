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

	siteMiddleware := middleware.SiteContext()

	cctvGroup.Get("/", middleware.APIKeyOrProtected(models.ScopeCCTVRead), siteMiddleware, handlers.GetAllCCTV)

	protectedCCTV := cctvGroup.Group("", middleware.Protected(), siteMiddleware)
	protectedCCTV.Get("/logs", handlers.GetCCTVStreamLogs)
	protectedCCTV.Get("/template", handlers.DownloadTemplateCCTV)
	protectedCCTV.Post("/scan", handlers.ScanCCTV)
	protectedCCTV.Get("/:id/qrcode", handlers.GetCCTVQRCode)
	protectedCCTV.Get("/:id/health-score", handlers.GetCCTVHealthScore)
	protectedCCTV.Get("/:id/snapshot-status", handlers.GetCCTVSnapshotStatus)
	protectedCCTV.Get("/:id/snapshot", handlers.GetCCTVSnapshot)
	protectedCCTV.Get("/:id/timeline", handlers.GetCCTVTimeline)
	protectedCCTV.Get("/:id/health-history", handlers.GetCCTVHealthHistory)
	protectedCCTV.Get("/:id", handlers.GetCCTVByID)

	adminCCTV := protectedCCTV.Group("", middleware.RequireRole("admin"))
	adminCCTV.Post("/", handlers.CreateCCTV)
	adminCCTV.Post("/import", handlers.ImportCCTV)
	adminCCTV.Post("/bulk-delete", handlers.BulkDeleteCCTV)
	adminCCTV.Post("/:id/rotate-qr", handlers.RotateQRCode)
	adminCCTV.Post("/:id/reactivate-verify", handlers.ReactivateAndVerifyCCTV)
	adminCCTV.Put("/:id", handlers.UpdateCCTV)
	adminCCTV.Delete("/:id", handlers.DeleteCCTV)

	router.Get("/cctv-logs", middleware.Protected(), siteMiddleware, handlers.GetCCTVStreamLogs)
}
