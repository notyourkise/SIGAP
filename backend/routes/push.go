package routes

import (
	"sigap-backend/handlers"
	"sigap-backend/middleware"

	"github.com/gofiber/fiber/v2"
)

// SetupPushRoutes mendaftarkan rute notifikasi push browser (Task 8.3).
// Akses untuk Admin & Teknisi yang telah login; scope per-user dari klaim JWT.
func SetupPushRoutes(router fiber.Router) {
	pushGroup := router.Group("/push", middleware.Protected())

	// Daftarkan langganan Web Push untuk user yang login.
	pushGroup.Post("/subscribe", handlers.SubscribePush)
	// Hapus langganan (opt-out) berdasarkan endpoint.
	pushGroup.Delete("/unsubscribe", handlers.UnsubscribePush)
}
