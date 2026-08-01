package routes

import (
	"sigap-backend/handlers"
	"sigap-backend/middleware"

	"github.com/gofiber/fiber/v2"
)

// SetupAuditRoutes mendefinisikan rute-rute untuk audit log oleh Admin
func SetupAuditRoutes(router fiber.Router) {
	// Rute audit log dilindungi oleh middleware Protected dan RequireRole("admin")
	auditGroup := router.Group("/audit-logs", middleware.Protected(), middleware.RequireRole("admin"))

	auditGroup.Get("/", handlers.GetAuditLogs)
	auditGroup.Get("/export", handlers.ExportAuditLogs)
}
