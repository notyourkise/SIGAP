package routes

import (
	"sigap-backend/handlers"
	"sigap-backend/middleware"

	"github.com/gofiber/fiber/v2"
)

// SetupProfileRoutes mendefinisikan rute profil pengguna (Task 11.3).
// Endpoint preferensi email dapat diakses oleh seluruh user yang login.
func SetupProfileRoutes(router fiber.Router) {
	profileGroup := router.Group("/profile", middleware.Protected())
	profileGroup.Put("/email-preferences", handlers.UpdateEmailPreferences)
}
