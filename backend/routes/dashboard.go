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
}
