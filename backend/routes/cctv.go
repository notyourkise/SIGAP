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
	cctvGroup.Post("/scan", handlers.ScanCCTV)
	cctvGroup.Get("/:id", handlers.GetCCTVByID)

	// Rute yang hanya dapat diakses oleh Admin
	cctvGroup.Post("/", middleware.RequireRole("admin"), handlers.CreateCCTV)
	cctvGroup.Put("/:id", middleware.RequireRole("admin"), handlers.UpdateCCTV)
	cctvGroup.Delete("/:id", middleware.RequireRole("admin"), handlers.DeleteCCTV)
}
