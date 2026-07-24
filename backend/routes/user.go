package routes

import (
	"sigap-backend/handlers"
	"sigap-backend/middleware"

	"github.com/gofiber/fiber/v2"
)

// SetupUserRoutes mendefinisikan rute-rute untuk pengelolaan pengguna oleh Admin
func SetupUserRoutes(router fiber.Router) {
	// Semua rute user dilindungi oleh middleware Protected dan RequireRole("admin")
	userGroup := router.Group("/users", middleware.Protected(), middleware.RequireRole("admin"))

	userGroup.Get("/", handlers.GetAllUsers)
	userGroup.Get("/:id", handlers.GetUserByID)
	userGroup.Put("/:id", handlers.UpdateUser)
	userGroup.Delete("/:id", handlers.DeleteUser)
	userGroup.Post("/:id/reset-password", handlers.ResetUserPassword)
}
