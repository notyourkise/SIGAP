package routes

import (
	"sigap-backend/handlers"
	"sigap-backend/middleware"

	"github.com/gofiber/fiber/v2"
)

// SetupNotificationRoutes mendefinisikan rute notifikasi in-app (Fase 5.1).
// Akses untuk Admin & Teknisi; setiap user hanya melihat data miliknya sendiri.
func SetupNotificationRoutes(router fiber.Router) {
	notifGroup := router.Group("/notifications", middleware.Protected())

	// PENTING: rute statis (/unread-count, /read-all) didaftarkan SEBELUM rute
	// dinamis (/:id/read) agar tidak tertimpa oleh parameter id (gotcha Fiber).
	notifGroup.Get("/", handlers.GetNotifications)
	notifGroup.Get("/unread-count", handlers.GetUnreadNotificationCount)
	notifGroup.Put("/read-all", handlers.MarkAllNotificationsRead)
	notifGroup.Delete("/cleanup", middleware.RequireRole("admin"), handlers.AdminCleanupNotifications)
	notifGroup.Put("/:id/read", handlers.MarkNotificationRead)
}
