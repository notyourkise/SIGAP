package middleware

import (
	"time"

	"sigap-backend/utils"

	"github.com/gofiber/fiber/v2"
)

// ZapLogger adalah middleware Fiber untuk mencatat detail request HTTP menggunakan Zap
func ZapLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Lanjutkan ke handler selanjutnya
		err := c.Next()

		latency := time.Since(start)
		status := c.Response().StatusCode()
		method := c.Method()
		path := c.Path()
		ip := c.IP()

		if err != nil {
			utils.Log.Errorw("HTTP Request Error",
				"method", method,
				"path", path,
				"status", status,
				"latency", latency.String(),
				"ip", ip,
				"error", err.Error(),
			)
			return err
		}

		utils.Log.Infow("HTTP Request",
			"method", method,
			"path", path,
			"status", status,
			"latency", latency.String(),
			"ip", ip,
		)

		return nil
	}
}
