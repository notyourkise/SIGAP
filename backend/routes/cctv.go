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

	// SiteContext WAJIB dipasang agar filter kawasan berfungsi (Fix BB-04).
	// Middleware ini membaca X-Site-ID header, fallback cookie/query, dan
	// memberi scope default untuk non-admin. Pemasangannya harus SETELAH
	// autentikasi agar identitas pengguna tersedia saat membership dicek.
	siteMiddleware := middleware.SiteContext()

	// Rute list dapat diakses via sesi JWT (web) ATAU API key integrasi eksternal
	// (Task 12.1, Sprint 29) dengan scope cctv:read.
	cctvGroup.Get("/", middleware.APIKeyOrProtected(models.ScopeCCTVRead), siteMiddleware, handlers.GetAllCCTV)

	// Rute yang dapat diakses oleh Admin & Teknisi (via sesi JWT).
	protectedCCTV := cctvGroup.Group("", middleware.Protected(), siteMiddleware)
	protectedCCTV.Get("/logs", handlers.GetCCTVStreamLogs)
	protectedCCTV.Get("/template", handlers.DownloadTemplateCCTV)
	protectedCCTV.Post("/scan", handlers.ScanCCTV)
	protectedCCTV.Get("/:id/qrcode", handlers.GetCCTVQRCode)
	protectedCCTV.Get("/:id/health-score", handlers.GetCCTVHealthScore)
	protectedCCTV.Get("/:id/snapshot", handlers.GetCCTVSnapshot)
	protectedCCTV.Get("/:id/timeline", handlers.GetCCTVTimeline)
	protectedCCTV.Get("/:id/health-history", handlers.GetCCTVHealthHistory)
	protectedCCTV.Get("/:id", handlers.GetCCTVByID)

	// Rute yang hanya dapat diakses oleh Admin.
	adminCCTV := protectedCCTV.Group("", middleware.RequireRole("admin"))
	adminCCTV.Post("/", handlers.CreateCCTV)
	adminCCTV.Post("/import", handlers.ImportCCTV)
	adminCCTV.Post("/bulk-delete", handlers.BulkDeleteCCTV)
	adminCCTV.Post("/:id/rotate-qr", handlers.RotateQRCode)
	adminCCTV.Put("/:id", handlers.UpdateCCTV)
	adminCCTV.Delete("/:id", handlers.DeleteCCTV)

	// Alias route /api/cctv-logs
	router.Get("/cctv-logs", middleware.Protected(), siteMiddleware, handlers.GetCCTVStreamLogs)
}
