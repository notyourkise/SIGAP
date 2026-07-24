package routes

import (
	"sigap-backend/handlers"
	"sigap-backend/middleware"

	"github.com/gofiber/fiber/v2"
)

// SetupUploadRoutes mendaftarkan rute unggah file gambar pemeliharaan
func SetupUploadRoutes(api fiber.Router) {
	// POST /api/upload — Dilindungi JWT (Admin & Teknisi)
	api.Post("/upload", middleware.Protected(), handlers.UploadFile)
}
