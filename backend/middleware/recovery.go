package middleware

import (
	"runtime/debug"

	"sigap-backend/utils"

	"github.com/gofiber/fiber/v2"
)

// Recovery adalah middleware Fiber untuk memulihkan panic runtime dan mencatat log stack trace menggunakan Zap
func Recovery() fiber.Handler {
	return func(c *fiber.Ctx) (err error) {
		defer func() {
			if r := recover(); r != nil {
				utils.Log.Errorw("PANIC RECOVERED",
					"error", r,
					"stack", string(debug.Stack()),
				)
				err = c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"status":  "error",
					"message": "Terjadi kesalahan internal pada server",
				})
			}
		}()
		return c.Next()
	}
}
