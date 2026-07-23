package main

import (
	"log"
	"sigap-backend/config"
	_ "sigap-backend/docs" // Dokumen yang dihasilkan oleh swag
	"sigap-backend/models"
	"sigap-backend/routes"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/swagger"
)

// @title                      SIGAP API
// @version                    1.0
// @description                Dokumentasi API Sistem Identifikasi Gangguan Aset Pusat (SIGAP) IKN.
// @host                       localhost:3000
// @BasePath                   /api
// @securityDefinitions.apikey  BearerAuth
// @in                         header
// @name                       Authorization
// @description                Format token JWT: Bearer <token_anda>
func main() {
	// 1. Eksekusi Koneksi Database
	config.ConnectDB()

	// 2. Jalankan Auto-Migrate (Mencetak tabel otomatis ke PostgreSQL)
	err := config.DB.AutoMigrate(
		&models.User{},
		&models.Asset{},
		&models.MaintenanceLog{},
	)
	if err != nil {
		log.Fatal("❌ Gagal melakukan migrasi database: ", err)
	}
	log.Println("✅ Migrasi tabel database berhasil!")

	// Jalankan Seeder untuk Admin Default pertama kali
	models.SeedAdmin(config.DB)

	// 3. Inisialisasi Aplikasi Fiber
	app := fiber.New()

	// Pasang Middleware CORS untuk membolehkan request dari Frontend (misal port 3001 atau domain local)
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	// Rute Dokumentasi Swagger UI
	app.Get("/swagger/*", swagger.HandlerDefault)

	// 4. Setup Routes
	api := app.Group("/api")
	routes.SetupAuthRoutes(api)
	routes.SetupAssetRoutes(api)
	routes.SetupMaintenanceRoutes(api)

	// Rute Tes API Perdana
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "success",
			"message": "Sistem SIGAP API Berjalan Lancar!",
		})
	})

	// 5. Nyalakan Server di Port 3000
	log.Println("⚡ Server menyala di http://localhost:3000")
	log.Fatal(app.Listen(":3000"))
}