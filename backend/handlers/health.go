package handlers

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"sigap-backend/config"

	"github.com/gofiber/fiber/v2"
)

// healthUploadsDir adalah lokasi fisik folder foto bukti (sama dengan var
// uploadsDir di handlers/upload.go). Dipakai untuk cek ukuran folder uploads.
const healthUploadsDir = "./uploads"

// HealthCheck memeriksa status API, Database PostgreSQL, Redis (bila ada), dan
// folder uploads (Task 7.4).
// @Summary      Periksa Kesehatan Layanan
// @Description  Memeriksa status keaktifan server API, koneksi database PostgreSQL (beserta response time), Redis (bila aktif), dan penggunaan folder uploads.
// @Tags         Health Check
// @Produce      json
// @Success      200 {object} map[string]interface{} "Status Sehat"
// @Failure      500 {object} map[string]interface{} "Database terputus atau bermasalah"
// @Router       /health [get]
func HealthCheck(c *fiber.Ctx) error {
	dbStatus := "Connected"
	statusCode := fiber.StatusOK
	dbResponseMs := 0.0

	// --- Cek Database + response time ---
	sqlDB, err := config.DB.DB()
	if err != nil {
		dbStatus = "Disconnected: " + err.Error()
		statusCode = fiber.StatusInternalServerError
	} else {
		start := time.Now()
		err = sqlDB.Ping()
		dbResponseMs = float64(time.Since(start).Microseconds()) / 1000.0 // ms
		if err != nil {
			dbStatus = "Disconnected: " + err.Error()
			statusCode = fiber.StatusInternalServerError
		}
	}

	// --- Cek Redis (bila aktif) ---
	redisStatus := "Disabled"
	redisResponseMs := 0.0
	if config.RedisEnabled && config.RedisClient != nil {
		ctx, cancel := context.WithTimeout(c.Context(), 2*time.Second)
		start := time.Now()
		err = config.RedisClient.Ping(ctx).Err()
		redisResponseMs = float64(time.Since(start).Microseconds()) / 1000.0
		cancel()
		if err != nil {
			redisStatus = "Disconnected: " + err.Error()
		} else {
			redisStatus = "Connected"
		}
	}

	// --- Cek folder uploads: ukuran total (bytes) & status ---
	uploadStatus := "OK"
	uploadSizeBytes, err := dirSize(healthUploadsDir)
	if err != nil {
		uploadStatus = "Unavailable: " + err.Error()
	}

	return c.Status(statusCode).JSON(fiber.Map{
		"api_status":        "Healthy",
		"db_status":         dbStatus,
		"db_response_time_ms": dbResponseMs,
		"redis_status":      redisStatus,
		"redis_response_time_ms": redisResponseMs,
		"uploads": fiber.Map{
			"status":     uploadStatus,
			"path":       healthUploadsDir,
			"size_bytes": uploadSizeBytes,
		},
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// dirSize menghitung total ukuran file dalam sebuah direktori secara rekursif.
// Mengembalikan 0 tanpa error bila direktori tidak ada / kosong.
func dirSize(path string) (int64, error) {
	var size int64
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	if !info.IsDir() {
		return info.Size(), nil
	}

	err = filepath.Walk(path, func(_ string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !fi.IsDir() {
			size += fi.Size()
		}
		return nil
	})
	return size, err
}
