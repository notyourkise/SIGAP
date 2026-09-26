package routes

import (
	"sigap-backend/handlers"
	"sigap-backend/middleware"

	"github.com/gofiber/fiber/v2"
)

// SetupMaintenanceRoutes mendefinisikan rute untuk modul log pemeliharaan CCTV
func SetupMaintenanceRoutes(router fiber.Router) {
	logGroup := router.Group("/maintenance-logs", middleware.Protected())

	// SiteContext WAJIB dipasang agar filter kawasan berfungsi (Fix BB-03)
	// Middleware ini membaca X-Site-ID header dan validasi site di database
	siteMiddleware := middleware.SiteContext()
	logGroup.Use(siteMiddleware)

	// Rute yang dapat diakses oleh Admin & Teknisi
	logGroup.Get("/", handlers.GetAllMaintenanceLogs)
	logGroup.Get("/export", handlers.ExportMaintenanceLogs)
	logGroup.Get("/report-pdf", handlers.GenerateMaintenanceReportPDF)
	logGroup.Get("/:id", handlers.GetMaintenanceLogByID)
	logGroup.Post("/", handlers.CreateMaintenanceLog)
	logGroup.Put("/:id", handlers.UpdateMaintenanceLog)
	// Penyelesaian tugas (9 tahap checklist) oleh teknisi yang ditugaskan
	logGroup.Post("/:id/complete", handlers.CompleteMaintenanceLog)

	// Rute yang hanya dapat diakses oleh Admin
	logGroup.Put("/:id/assign", middleware.RequireRole("admin"), handlers.AssignMaintenanceLog)
	logGroup.Delete("/:id", middleware.RequireRole("admin"), handlers.DeleteMaintenanceLog)
}
