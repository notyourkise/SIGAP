package main

import (
	"os"
	"os/signal"
	"sigap-backend/config"
	_ "sigap-backend/docs" // Dokumen yang dihasilkan oleh swag
	"sigap-backend/handlers"
	"sigap-backend/middleware"
	"sigap-backend/models"
	"sigap-backend/monitoring"
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

	// Inisialisasi konfigurasi email/SMTP (Task 11.3). Bila SMTP_HOST kosong,
	// notifikasi email dinonaktifkan dan hanya memakai in-app + push.
	utils.InitEmailConfig()
	if utils.EmailEnabled() {
		utils.Log.Info("📧 Notifikasi email SMTP aktif")
	} else {
		utils.Log.Warn("⚠️ Email SMTP tidak dikonfigurasi — notifikasi email nonaktif (SMTP_HOST kosong)")
	}

	// Validasi secret wajib sebelum server berjalan (tanpa fallback hardcoded).
	// Bila Docker secret tersedia di /run/secrets/..., nilainya dipakai;
	// selain itu fallback ke environment variable. Fatal bila keduanya kosong.
	utils.RequireSecret("JWT_SECRET", "/run/secrets/jwt_secret")
	utils.RequireSecret("AES_SECRET_KEY", "/run/secrets/aes_secret_key")

	// 1. Eksekusi Koneksi Database
	config.ConnectDB()

	// 1.5. Koneksi Redis untuk rate-limit & blacklist (Task 7.2).
	// Graceful fallback: bila Redis tidak tersedia (development tanpa Docker),
	// hanya muncul LOG WARNING dan sistem memakai in-memory store.
	config.ConnectRedis()
	if config.RedisEnabled {
		utils.SetTokenBlacklist(utils.NewRedisTokenBlacklist(config.RedisClient))
		utils.Log.Info("🛡️ Token blacklist memakai Redis — persisten lintas restart & multi-instance")
	} else {
		utils.Log.Warn("⚠️ Token blacklist memakai in-memory — token tercabut akan hilang saat server restart")
	}

	// 1.5. Pastikan direktori uploads tersedia untuk lampiran foto pemeliharaan
	if err := os.MkdirAll("./uploads", os.ModePerm); err != nil {
		utils.Log.Fatalf("❌ Gagal membuat direktori uploads: %v", err)
	}

	// 2. Jalankan migrasi DB versioned (golang-migrate v4) — sumber skema resmi.
	//    File SQL terurut ada di backend/migrations/ dan riwayat versi dicatat
	//    pada tabel schema_migrations. Migration ini BERJALAN LEBIH DULU
	//    dibanding AutoMigrate (fase transisi) di bawahnya.
	if err := config.RunMigrations(config.DB); err != nil {
		utils.Log.Fatal("❌ Gagal menjalankan migrasi DB versioned: ", err)
	}
	utils.Log.Info("✅ Migrasi DB versioned berhasil dijalankan!")

	// FASE TRANSISI (Task 7.1): AutoMigrate GORM dikomentari dan AKAN DIHAPUS
	// setelah migration versioned stabil. Jangan diaktifkan kembali tanpa
	// persetujuan — skema kini dikelola sepenuhnya oleh migration file.
	/*
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
			&models.DailyDigest{},
		)
		if err != nil {
			utils.Log.Fatal("❌ Gagal melakukan migrasi database: ", err)
		}
		utils.Log.Info("✅ Migrasi tabel database berhasil!")
	*/

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

	// Jalankan kalkulator skor kesehatan CCTV (Fase 6.1). Proses pertama
	// dilakukan saat startup, lalu diulang setiap 6 jam.
	go handlers.RunHealthScoreCalculator(config.DB)
	utils.Log.Info("🩺 Kalkulator skor kesehatan CCTV diaktifkan (interval: 6 jam)")

	// Jalankan generator ringkasan operasional harian (Fase 6.4). Generator
	// menunggu jadwal 06:00 WITA berikutnya lalu memproses jendela 24 jam.
	go handlers.RunDailyDigestGenerator(config.DB)
	utils.Log.Info("🗓️ Generator ringkasan operasional harian diaktifkan (jadwal: 06:00 WITA)")

	// Jalankan collector metrik (Task 7.4): memperbarui gauge DB pool, WS aktif,
	// dan ukuran folder uploads secara berkala (interval 15 detik).
	monitoring.StartCollector("./uploads", 15*time.Second)

	// Auto-register stream CCTV yang aktif ke MediaMTX (Task 9.3). Dijalankan
	// sebagai goroutine dengan jeda awal 5 detik agar server sudah siap.
	go func() {
		time.Sleep(5 * time.Second)
		handlers.RegisterActiveStreams(config.DB)
	}()
	utils.Log.Info("📡 Auto-register stream CCTV ke MediaMTX dijadwalkan (tunda 5 detik)")

	// Jalankan scheduler snapshot otomatis (Task 10.1). Berjalan sekali segera,
	// lalu berulang setiap 15 menit untuk seluruh CCTV stream_enabled.
	go handlers.RunSnapshotScheduler(config.DB)
	utils.Log.Info("📸 Scheduler snapshot otomatis diaktifkan (interval: 15 menit)")

	// Jalankan monitor status stream (Task 10.4). Berjalan sekali segera, lalu
	// berulang setiap 5 menit; mendeteksi down >15 menit & pemulihan stream.
	go handlers.RunStreamHealthMonitor(config.DB)
	utils.Log.Info("🩺 Monitor status stream diaktifkan (interval: 5 menit)")

	// Jalankan goroutine pemeriksa deadline SLA (Task 11.5, Sprint 28).
	// Berjalan sekali saat startup, lalu berulang setiap 1 jam; memicu event
	// SLA_WARNING ke admin saat tugas melewati 75% target penyelesaian.
	go handlers.RunSLADeadlineChecker(config.DB)
	utils.Log.Info("⏱️ Pemeriksa deadline SLA diaktifkan (interval: 1 jam)")

	// Jalankan worker pengiriman webhook (Task 12.3, Sprint 29). Mengonsumsi
	// antrean event dan mengirim ke endpoint webhook secara asinkron.
	handlers.StartWebhookWorker(config.DB)
	utils.Log.Info("🔗 Worker pengiriman webhook diaktifkan")

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

	// Pasang Middleware Metrics (Task 7.4): mencatat counter request & durasi
	// histogram ke collector Prometheus. Endpoint /api/metrics dikecualikan.
	app.Use(middleware.Metrics())

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
	routes.SetupPushRoutes(api)
	routes.SetupMaintenanceScheduleRoutes(api)
	routes.SetupStreamRoutes(api)
	routes.SetupSparePartRoutes(api)
	routes.SetupProfileRoutes(api)
	routes.SetupReportRoutes(api)
	routes.SetupZoneRoutes(api)
	routes.SetupSLARoutes(api)
	routes.SetupAPIKeyRoutes(api)
	routes.SetupWebhookRoutes(api)

	// Rute Health Check (publik untuk monitoring, juga dapat diakses via API key
	// scope health:read untuk integrasi eksternal — Task 12.1).
	api.Get("/health", handlers.HealthCheck)

	// Rute Metrik Prometheus (publik untuk monitoring / scraping). Mengembalikan
	// format text exposition Prometheus (Task 7.4).
	api.Get("/metrics", monitoring.Handler)

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

	// Menutup koneksi Redis bila aktif (Task 7.2)
	if config.RedisClient != nil {
		if err := config.RedisClient.Close(); err != nil {
			utils.Log.Errorf("❌ Gagal menutup koneksi Redis: %v", err)
		} else {
			utils.Log.Info("✅ Koneksi Redis ditutup secara aman.")
		}
	}

	utils.Log.Info("👋 Server SIGAP berhasil dimatikan secara bersih.")
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
