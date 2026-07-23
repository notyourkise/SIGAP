package config

import (
	"fmt"
	"os"
	"sigap-backend/utils"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDB() {
	// Memuat file .env
	err := godotenv.Load()
	if err != nil {
		utils.Log.Warn("⚠️ Peringatan: File .env tidak ditemukan!")
	}

	// Merangkai koneksi dari environment variables
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Makassar",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
	)

	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		utils.Log.Fatal("❌ Gagal koneksi ke database! ", err)
	}

	utils.Log.Info("🚀 Koneksi Database PostgreSQL Berhasil!")
}