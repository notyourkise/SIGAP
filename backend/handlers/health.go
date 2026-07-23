package handlers

import (
	"time"

	"sigap-backend/config"

	"github.com/gofiber/fiber/v2"
)

// HealthCheck memeriksa status API dan Database PostgreSQL
// @Summary      Periksa Kesehatan Layanan
// @Description  Memeriksa status keaktifan server API dan koneksi database PostgreSQL.
// @Tags         Health Check
// @Produce      json
// @Success      200 {object} map[string]interface{} "Status Sehat"
// @Failure      500 {object} map[string]interface{} "Database terputus atau bermasalah"
// @Router       /health [get]
func HealthCheck(c *fiber.Ctx) error {
	dbStatus := "Connected"
	statusCode := fiber.StatusOK

	sqlDB, err := config.DB.DB()
	if err != nil {
		dbStatus = "Disconnected: " + err.Error()
		statusCode = fiber.StatusInternalServerError
	} else {
		err = sqlDB.Ping()
		if err != nil {
			dbStatus = "Disconnected: " + err.Error()
			statusCode = fiber.StatusInternalServerError
		}
	}

	return c.Status(statusCode).JSON(fiber.Map{
		"api_status": "Healthy",
		"db_status":  dbStatus,
		"timestamp":  time.Now().Format(time.RFC3339),
	})
}
