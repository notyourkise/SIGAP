package main

import (
	"os"
	"os/signal"
	"sigap-backend/config"
	_ "sigap-backend/docs" // Dokumen yang dihasilkan oleh swag
	"sigap-backend/handlers"
	"sigap-backend/middleware"
	"sigap-backend/models"
	"sigap-backend/routes"
	"sigap-backend/utils"
	"syscall"
	"time"

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
	// Inisialisasi Logger Terstruktur
	utils.InitLogger()
	defer func() {
		_ = utils.Log.Sync() // Flush buffer log ke file
	}()

	// 1. Eksekusi Koneksi Database
	config.ConnectDB()

	// 1.5. Pastikan direktori uploads tersedia untuk lampiran foto pemeliharaan
	if err := os.MkdirAll("./uploads", os.ModePerm); err != nil {
		utils.Log.Fatalf("❌ Gagal membuat direktori uploads: %v", err)
	}

	// 2. Jalankan Auto-Migrate (Mencetak tabel otomatis ke PostgreSQL)
	err := config.DB.AutoMigrate(
		&models.User{},
		&models.Asset{},
		&models.MaintenanceLog{},
	)
	if err != nil {
		utils.Log.Fatal("❌ Gagal melakukan migrasi database: ", err)
	}
	utils.Log.Info("✅ Migrasi tabel database berhasil!")

	// Jalankan Seeder untuk Admin Default pertama kali
	models.SeedAdmin(config.DB)

	// 3. Inisialisasi Aplikasi Fiber
	app := fiber.New()

	// Pasang Custom Recovery Middleware paling awal untuk melindung server dari crash/panic
	app.Use(middleware.Recovery())

	// Pasang Middleware CORS untuk membolehkan request dari Frontend (misal port 3001 atau domain local)
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	// Pasang Custom Structured Logger menggunakan Zap
	app.Use(middleware.ZapLogger())

	// Rute Dokumentasi Swagger UI
	app.Get("/swagger/*", swagger.HandlerDefault)

	// Sajikan folder uploads sebagai direktori statis (untuk akses gambar lampiran foto)
	app.Static("/uploads", "./uploads")

	// 4. Setup Routes dengan Global Rate Limiting
	api := app.Group("/api", middleware.GlobalLimiter())
	routes.SetupAuthRoutes(api)
	routes.SetupAssetRoutes(api)
	routes.SetupMaintenanceRoutes(api)
	routes.SetupUserRoutes(api)
	routes.SetupUploadRoutes(api)

	// Rute Health Check
	api.Get("/health", handlers.HealthCheck)

	// Rute Statistik Dashboard (membutuhkan autentikasi)
	api.Get("/dashboard/stats", middleware.Protected(), handlers.GetDashboardStats)

	// Rute Tes Panic Sementara untuk memvalidasi Auto-Recovery
	api.Get("/test-panic", TestPanic)

	// Rute Tes API Perdana
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "success",
			"message": "Sistem SIGAP API Berjalan Lancar!",
		})
	})

	// 5. Nyalakan Server di Goroutine agar tidak memblokir main thread
	go func() {
		utils.Log.Info("⚡ Server menyala di http://localhost:3000")
		if err := app.Listen(":3000"); err != nil {
			utils.Log.Fatal("❌ Gagal menyalakan server: ", err)
		}
	}()

	// Menangkap Signal OS untuk Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// Blokir program hingga sinyal penghentian diterima
	<-quit
	utils.Log.Info("⚠️ Menutup server SIGAP secara rapi (Graceful Shutdown)...")

	// Shutdown Fiber dengan batas toleransi waktu 10 detik
	if err := app.ShutdownWithTimeout(10 * time.Second); err != nil {
		utils.Log.Errorf("❌ Terjadi kesalahan saat menutup server: %v", err)
	}

	// Menutup Koneksi Database PostgreSQL secara aman
	sqlDB, err := config.DB.DB()
	if err == nil {
		if err := sqlDB.Close(); err != nil {
			utils.Log.Errorf("❌ Gagal menutup koneksi database: %v", err)
		} else {
			utils.Log.Info("✅ Koneksi database PostgreSQL ditutup secara aman.")
		}
	}

	utils.Log.Info("👋 Server SIGAP berhasil dimatikan secara bersih.")
}

// TestPanic pemicu panic buatan untuk pengujian
// @Summary      Pemicu Panic Buatan
// @Description  Meminta server untuk melakukan panic sengaja demi menguji kinerja Auto-Recovery.
// @Tags         Pengujian
// @Produce      json
// @Success      500 {object} map[string]interface{} "Kembalian error internal seragam"
// @Router       /test-panic [get]
func TestPanic(c *fiber.Ctx) error {
	panic("⚠️ Pemicu Panic Sengaja untuk Pengujian Auto-Recovery!")
}