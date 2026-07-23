package routes

import (
	"sigap-backend/handlers"
	"sigap-backend/middleware"

	"github.com/gofiber/fiber/v2"
)

// SetupAuthRoutes mendefinisikan rute-rute untuk modul autentikasi dan manajemen user
func SetupAuthRoutes(router fiber.Router) {
	authGroup := router.Group("/auth")

	// Rute publik login dengan limitasi brute force
	authGroup.Post("/login", middleware.AuthLimiter(), handlers.Login)

	// Rute pembuatan akun (Hanya dapat diakses oleh Admin yang sudah login)
	authGroup.Post("/users", middleware.Protected(), middleware.RequireRole("admin"), handlers.CreateUser)
}
