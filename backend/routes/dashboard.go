package routes

import (
	"sigap-backend/handlers"
	"sigap-backend/middleware"

	"github.com/gofiber/fiber/v2"
)

// SetupDashboardRoutes mendefinisikan rute ringkasan analitik dashboard
// (dapat diakses oleh Admin & Teknisi yang sudah login).
func SetupDashboardRoutes(router fiber.Router) {
	dashboardGroup := router.Group("/dashboard", middleware.Protected())
	dashboardGroup.Get("/summary", handlers.DashboardSummary)
	dashboardGroup.Get("/tasks", handlers.DashboardTasks)
	dashboardGroup.Get("/technician-performance", middleware.RequireRole("admin"), handlers.GetTechnicianPerformance)
	dashboardGroup.Get("/digest", middleware.RequireRole("admin"), handlers.GetDailyDigests)
	// Dashboard SLA compliance (Task 11.5) — khusus admin. Handler sudah ada
	// sejak Sprint 28 namun rutenya belum terdaftar (menyebabkan 404 di
	// halaman /sla). Perbaikan Sprint 40.
	dashboardGroup.Get("/sla", middleware.RequireRole("admin"), handlers.GetDashboardSLA)
}
