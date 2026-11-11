package routes

import (
	"sigap-backend/handlers"
	"sigap-backend/middleware"
	"sigap-backend/models"

	"github.com/gofiber/fiber/v2"
)

// SetupCCTVRoutes mendaftarkan rute untuk modul Master Data CCTV terenkripsi
func SetupCCTVRoutes(router fiber.Router) {
	siteMiddleware := middleware.SiteContext()
	protected := middleware.Protected()
	adminOnly := middleware.RequireRole("admin")

	// Alias kompatibilitas /cctv-logs didaftarkan di luar grup /cctv serta tanpa
	// sub-grup `Group("", RequireRole("admin"))` karena di Fiber v2 middleware
	// `Use()` pada prefix `/cctv` juga cocok dengan `/cctv-logs` (`strings.HasPrefix`).
	router.Get("/cctv-logs", protected, siteMiddleware, handlers.GetCCTVStreamLogs)

	cctvGroup := router.Group("/cctv")

	cctvGroup.Get("/", middleware.APIKeyOrProtected(models.ScopeCCTVRead), siteMiddleware, handlers.GetAllCCTV)

	// Rute baca & operasional terautentikasi (admin & teknisi).
	// Rute statis didaftarkan sebelum rute dinamis /:id (gotcha Fiber route ordering).
	cctvGroup.Get("/logs", protected, siteMiddleware, handlers.GetCCTVStreamLogs)
	cctvGroup.Get("/template", protected, siteMiddleware, handlers.DownloadTemplateCCTV)
	cctvGroup.Post("/scan", protected, siteMiddleware, handlers.ScanCCTV)
	cctvGroup.Get("/:id/qrcode", protected, siteMiddleware, handlers.GetCCTVQRCode)
	cctvGroup.Get("/:id/health-score", protected, siteMiddleware, handlers.GetCCTVHealthScore)
	cctvGroup.Get("/:id/snapshot-status", protected, siteMiddleware, handlers.GetCCTVSnapshotStatus)
	cctvGroup.Get("/:id/snapshot", protected, siteMiddleware, handlers.GetCCTVSnapshot)
	cctvGroup.Get("/:id/timeline", protected, siteMiddleware, handlers.GetCCTVTimeline)
	cctvGroup.Get("/:id/health-history", protected, siteMiddleware, handlers.GetCCTVHealthHistory)
	cctvGroup.Get("/:id", protected, siteMiddleware, handlers.GetCCTVByID)

	// Rute mutasi master data CCTV khusus admin (inline middleware agar tidak
	// membayangi rute non-admin atau alias /cctv-logs).
	cctvGroup.Post("/", protected, adminOnly, siteMiddleware, handlers.CreateCCTV)
	cctvGroup.Post("/import", protected, adminOnly, siteMiddleware, handlers.ImportCCTV)
	cctvGroup.Post("/bulk-delete", protected, adminOnly, siteMiddleware, handlers.BulkDeleteCCTV)
	cctvGroup.Post("/:id/rotate-qr", protected, adminOnly, siteMiddleware, handlers.RotateQRCode)
	cctvGroup.Post("/:id/reactivate-verify", protected, adminOnly, siteMiddleware, handlers.ReactivateAndVerifyCCTV)
	cctvGroup.Put("/:id", protected, adminOnly, siteMiddleware, handlers.UpdateCCTV)
	cctvGroup.Delete("/:id", protected, adminOnly, siteMiddleware, handlers.DeleteCCTV)
}
