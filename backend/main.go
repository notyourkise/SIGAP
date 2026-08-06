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
	"strings"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/swagger"
	"github.com/joho/godotenv"
)

// @title                      SIGAP API
// @version                    1.0
// @description                Dokumentasi API Sistem Identifikasi Gangguan Aset Pusat (SIGAP) IKN.
// @host                       localhost
// @schemes                    https
// @BasePath                   /api
// @securityDefinitions.apikey  BearerAuth
// @in                         header
// @name                       Authorization
// @description                Format token JWT: Bearer <token_anda>
func main() {
	// Memuat environment variables dari file .env (jika ada)
	_ = godotenv.Load(".env", "../.env")

	// Inisialisasi Logger Terstruktur
	utils.InitLogger()
	defer func() {
		_ = utils.Log.Sync() // Flush buffer log ke file
	}()

	// Validasi secret wajib sebelum server berjalan (tanpa fallback hardcoded)
	requireEnv("JWT_SECRET")
	requireEnv("AES_SECRET_KEY")

	// 1. Eksekusi Koneksi Database
	config.ConnectDB()

	// 1.5. Pastikan direktori uploads tersedia untuk lampiran foto pemeliharaan
	if err := os.MkdirAll("./uploads", os.ModePerm); err != nil {
		utils.Log.Fatalf("❌ Gagal membuat direktori uploads: %v", err)
	}

	// 2. Jalankan Auto-Migrate (Mencetak tabel otomatis ke PostgreSQL)
	err := config.DB.AutoMigrate(
		&models.User{},
		&models.MaintenanceLog{},
		&models.FieldMaintenance{},
		&models.FieldMaintenanceCheck{},
		&models.FieldMaintenanceImage{},
		&models.CCTV{},
		&models.AuditLog{},
		&models.Notification{},
		&models.MaintenanceSchedule{},
	)
	if err != nil {
		utils.Log.Fatal("❌ Gagal melakukan migrasi database: ", err)
	}
	utils.Log.Info("✅ Migrasi tabel database berhasil!")

	// Migrasi data legacy: maintenance_logs.asset_id → cctv_id (Fase 3.6)
	config.MigrateMaintenanceToCCTV(config.DB)

	// Jalankan Seeder untuk Admin Default pertama kali
	models.SeedAdmin(config.DB)

	// Jalankan goroutine cleanup JWT blacklist setiap 30 menit
	// Membersihkan token expired agar tidak terjadi memory leak
	utils.StartBlacklistCleanup(30 * time.Minute)
	utils.Log.Info("🧹 JWT Blacklist cleanup goroutine diaktifkan (interval: 30 menit)")

	// Jalankan goroutine cleanup file upload yatim (tidak direferensikan log) tiap 1 jam
	handlers.StartUploadCleanup(config.DB, 1*time.Hour)
	utils.Log.Info("🗂️ Upload orphan file cleanup goroutine diaktifkan (interval: 1 jam)")

	// Jalankan goroutine generator jadwal pemeliharaan berulang (Fase 5.4).
	// Berjalan sekali saat startup + tiap 1 jam; idempotent per hari per jadwal.
	go handlers.RunScheduleGenerator(config.DB)
	utils.Log.Info("⏰ Generator jadwal pemeliharaan berulang diaktifkan (interval: 1 jam)")

	// Jalankan goroutine eskalasi overdue (Fase 5.5). Proses pertama dilakukan
	// saat startup, lalu berulang setiap 1 jam.
	go handlers.RunOverdueChecker(config.DB)
	utils.Log.Info("🚨 Checker eskalasi overdue diaktifkan (interval: 1 jam)")

	// 3. Inisialisasi Aplikasi Fiber — konfigurasi Trusted Proxy: header
	// X-Forwarded-For / X-Real-IP hanya dipercaya bila koneksi datang dari
	// proxy yang dikenal (mis. frontend Next.js/Nginx). Default kosong =
	// header proxy diabaikan (aman dari spoofing).
	trustedProxies := []string{}
	if trustedIPs := os.Getenv("TRUSTED_PROXY_IPS"); trustedIPs != "" {
		for _, p := range strings.Split(trustedIPs, ",") {
			if t := strings.TrimSpace(p); t != "" {
				trustedProxies = append(trustedProxies, t)
			}
		}
	}
	app := fiber.New(fiber.Config{
		EnableTrustedProxyCheck: len(trustedProxies) > 0,
		TrustedProxies:          trustedProxies,
	})
	if len(trustedProxies) > 0 {
		utils.Log.Info("🛡️ Trusted proxy aktif: " + strings.Join(trustedProxies, ", "))
	} else {
		utils.Log.Warn("⚠️ TRUSTED_PROXY_IPS tidak diset — header X-Forwarded-For/X-Real-IP TIDAK dipercaya. Set variabel ini bila aplikasi di belakang proxy (frontend Next.js/Nginx).")
	}

	// Pasang Custom Recovery Middleware paling awal untuk melindung server dari crash/panic
	app.Use(middleware.Recovery())

	// Pasang Middleware CORS: whitelist origin dari env (tanpa wildcard karena
	// memakai cookie credentials). Fallback ini hanya untuk development tanpa Docker;
	// deployment Docker wajib mengatur origin publik https://localhost.
	allowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	if allowedOrigins == "" {
		allowedOrigins = "http://localhost:3001"
		utils.Log.Warn("⚠️ CORS_ALLOWED_ORIGINS tidak diset — menggunakan default development http://localhost:3001. Deployment Docker wajib memakai https://localhost.")
	}
	app.Use(cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-CSRF-Token",
		AllowMethods:     "GET, POST, PUT, DELETE, OPTIONS",
		AllowCredentials: true,
	}))

	// Pasang Custom Structured Logger menggunakan Zap
	app.Use(middleware.ZapLogger())

	// Pasang Security Headers (X-Content-Type-Options, X-Frame-Options, dll)
	app.Use(middleware.SecurityHeaders())

	// SECURITY: Swagger UI hanya tersedia di environment non-production.
	// Set APP_ENV=production di .env untuk menonaktifkan Swagger di production.
	appEnv := os.Getenv("APP_ENV")
	if appEnv != "production" {
		app.Get("/swagger/*", swagger.HandlerDefault)
		utils.Log.Info("📖 Swagger UI aktif di /swagger/index.html (environment: " + appEnv + ")")
	} else {
		utils.Log.Info("🔒 Swagger UI dinonaktifkan di environment production")
	}

	// SECURITY: File uploads tidak lagi disajikan sebagai direktori statis publik.
	// Akses ke file gambar sekarang melalui endpoint /api/uploads/:filename yang dilindungi JWT.
	// app.Static("/uploads", "./uploads") <-- DIHAPUS demi keamanan

	// 4. Setup Routes dengan Global Rate Limiting
	// CSRFProtection melindungi permintaan mutasi yang terautentikasi via cookie.
	api := app.Group("/api", middleware.GlobalLimiter(), middleware.CSRFProtection())
	routes.SetupAuthRoutes(api)
	routes.SetupMaintenanceRoutes(api)
	routes.SetupFieldMaintenanceRoutes(api)
	routes.SetupUserRoutes(api)
	routes.SetupCCTVRoutes(api)
	routes.SetupUploadRoutes(api)
	routes.SetupAuditRoutes(api)
	routes.SetupWebSocketRoutes(api)
	routes.SetupDashboardRoutes(api)
	routes.SetupNotificationRoutes(api)
	routes.SetupMaintenanceScheduleRoutes(api)

	// Rute Health Check (publik untuk monitoring)
	api.Get("/health", handlers.HealthCheck)

	// Rute root API
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "success",
			"message": "Sistem SIGAP API Berjalan Lancar!",
		})
	})

	// SECURITY: Endpoint test-panic HANYA tersedia di development environment
	// Tidak pernah aktif di production untuk mencegah penyalahgunaan
	if appEnv != "production" {
		api.Get("/test-panic", TestPanic)
		utils.Log.Warn("⚠️  Endpoint /api/test-panic aktif (hanya untuk development)")
	}

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

// requireEnv memastikan environment variable wajib tersedia; fatal jika kosong.
func requireEnv(name string) {
	if os.Getenv(name) == "" {
		utils.Log.Fatalf("❌ Environment variable wajib tidak terisi: %s", name)
	}
}

// TestPanic adalah handler pengujian sementara untuk memverifikasi fungsionalitas auto-recovery
// @Summary      Tes Panic Recovery
// @Description  Endpoint uji coba untuk memastikan Recovery Middleware bekerja (hanya tersedia di development).
// @Tags         Utilitas
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Router       /test-panic [get]
func TestPanic(c *fiber.Ctx) error {
	panic("🔥 Ini adalah panic buatan untuk pengujian recovery middleware")
}
