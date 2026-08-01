package routes

import (
	"sigap-backend/handlers"
	"sigap-backend/middleware"

	"github.com/gofiber/fiber/v2"
)

// SetupUploadRoutes mendaftarkan rute unggah dan pengambilan file gambar pemeliharaan
func SetupUploadRoutes(api fiber.Router) {
	// POST /api/upload — Upload foto (dilindungi JWT, Admin & Teknisi)
	api.Post("/upload", middleware.Protected(), handlers.UploadFile)

	// GET /api/uploads/:filename — Ambil foto (dilindungi JWT, mencegah akses publik ke file)
	// SECURITY: File tidak lagi disajikan via app.Static() yang publik.
	// Semua request ke file gambar harus melalui endpoint ini yang memverifikasi token.
	api.Get("/uploads/:filename", middleware.Protected(), handlers.ServeUploadedFile)
}
