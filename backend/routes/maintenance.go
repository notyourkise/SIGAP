package routes

import (
	"sigap-backend/handlers"
	"sigap-backend/middleware"

	"github.com/gofiber/fiber/v2"
)

// SetupMaintenanceRoutes mendefinisikan rute untuk modul log pemeliharaan CCTV
func SetupMaintenanceRoutes(router fiber.Router) {
	logGroup := router.Group("/maintenance-logs", middleware.Protected())

	// Rute yang dapat diakses oleh Admin & Teknisi
	logGroup.Get("/", handlers.GetAllMaintenanceLogs)
	logGroup.Get("/export", handlers.ExportMaintenanceLogs)
	logGroup.Get("/report-pdf", handlers.GenerateMaintenanceReportPDF)
	logGroup.Get("/:id", handlers.GetMaintenanceLogByID)
	logGroup.Post("/", handlers.CreateMaintenanceLog)
	logGroup.Put("/:id", handlers.UpdateMaintenanceLog)

	// Rute yang hanya dapat diakses oleh Admin
	logGroup.Delete("/:id", middleware.RequireRole("admin"), handlers.DeleteMaintenanceLog)
}
