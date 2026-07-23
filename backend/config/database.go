package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

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

	// Konfigurasi Database Connection Pool (Tuning)
	sqlDB, err := DB.DB()
	if err != nil {
		utils.Log.Fatal("❌ Gagal mengambil instance sql.DB dari GORM: ", err)
	}

	// Ambil konfigurasi dari environment variables atau gunakan default
	maxOpenConnsStr := os.Getenv("DB_MAX_OPEN_CONNS")
	maxOpenConns := 50
	if maxOpenConnsStr != "" {
		if val, err := strconv.Atoi(maxOpenConnsStr); err == nil && val > 0 {
			maxOpenConns = val
		}
	}

	maxIdleConnsStr := os.Getenv("DB_MAX_IDLE_CONNS")
	maxIdleConns := 10
	if maxIdleConnsStr != "" {
		if val, err := strconv.Atoi(maxIdleConnsStr); err == nil && val > 0 {
			maxIdleConns = val
		}
	}

	connMaxLifetimeStr := os.Getenv("DB_CONN_MAX_LIFETIME") // dalam menit
	connMaxLifetime := 30 * time.Minute
	if connMaxLifetimeStr != "" {
		if val, err := strconv.Atoi(connMaxLifetimeStr); err == nil && val > 0 {
			connMaxLifetime = time.Duration(val) * time.Minute
		}
	}

	connMaxIdleTimeStr := os.Getenv("DB_CONN_MAX_IDLE_TIME") // dalam menit
	connMaxIdleTime := 10 * time.Minute
	if connMaxIdleTimeStr != "" {
		if val, err := strconv.Atoi(connMaxIdleTimeStr); err == nil && val > 0 {
			connMaxIdleTime = time.Duration(val) * time.Minute
		}
	}

	// Terapkan parameter pooling ke sql.DB
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetConnMaxLifetime(connMaxLifetime)
	sqlDB.SetConnMaxIdleTime(connMaxIdleTime)

	utils.Log.Infof("⚡ Database Connection Pool diatur: MaxOpen=%d, MaxIdle=%d, MaxLifetime=%v, MaxIdleTime=%v",
		maxOpenConns, maxIdleConns, connMaxLifetime, connMaxIdleTime)
}