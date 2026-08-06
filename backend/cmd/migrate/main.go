// Command migrate adalah CLI helper untuk menjalankan migrasi DB versioned
// (golang-migrate v4) secara manual:
//
//	go run cmd/migrate/main.go up       # terapkan semua migration yang belum berjalan
//	go run cmd/migrate/main.go down     # rollback semua migration (kembali ke versi 0)
//	go run cmd/migrate/main.go version  # tampilkan versi migration aktif + status dirty
//
// Koneksi database memakai environment variable yang sama dengan aplikasi
// utama (DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME — lihat config/database.go).
package main

import (
	"fmt"
	"os"

	"sigap-backend/config"
	"sigap-backend/utils"

	"github.com/joho/godotenv"
)

func main() {
	// Muat file .env relatif terhadap direktori kerja (sama dengan main.go).
	_ = godotenv.Load(".env", "../.env")

	// Inisialisasi logger agar output konsisten dengan aplikasi utama.
	utils.InitLogger()

	// Hubungkan ke PostgreSQL menggunakan konfigurasi env yang sama.
	config.ConnectDB()

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "up":
		if err := config.MigrationUp(config.DB); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Migration up gagal: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ Migration up selesai.")
	case "down":
		if err := config.MigrationDown(config.DB); err != nil {
			fmt.Fprintf(os.Stderr, "❌ Migration down gagal: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ Migration down selesai (semua migration di-rollback).")
	case "version":
		version, dirty, err := config.MigrationVersion(config.DB)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Membaca versi migration gagal: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("📋 Versi migration aktif: %d (dirty: %v)\n", version, dirty)
	default:
		printUsage()
		os.Exit(1)
	}
}

// printUsage menampilkan petunjuk penggunaan CLI.
func printUsage() {
	fmt.Println("Penggunaan: go run cmd/migrate/main.go <up|down|version>")
	fmt.Println("  up       - Terapkan semua migration yang belum berjalan")
	fmt.Println("  down     - Rollback semua migration (kembali ke versi 0)")
	fmt.Println("  version  - Tampilkan versi migration aktif + status dirty")
}