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

	// Rute logout — merevoke token ke blacklist (dilindungi JWT)
	authGroup.Post("/logout", middleware.Protected(), handlers.Logout)

	// Rute profil pengguna saat ini berdasarkan token (dilindungi JWT)
	authGroup.Get("/me", middleware.Protected(), handlers.GetMyProfile)

	// Rute perbarui profil sendiri (nama tampilan)
	authGroup.Put("/profile", middleware.Protected(), handlers.UpdateMyProfile)

	// Rute ganti password sendiri (self-service, dilindungi JWT)
	authGroup.Post("/change-password", middleware.Protected(), handlers.ChangePassword)

	// Rute pembuatan akun (Hanya dapat diakses oleh Admin yang sudah login)
	authGroup.Post("/users", middleware.Protected(), middleware.RequireRole("admin"), handlers.CreateUser)
}
