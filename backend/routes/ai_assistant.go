package routes

import (
	"github.com/gofiber/fiber/v2"
	"sigap-backend/handlers"
	"sigap-backend/middleware"
)

func SetupAIAssistantRoutes(router fiber.Router) {
	group := router.Group("/ai-assistant", middleware.Protected(), middleware.SiteContext())
	group.Post("/ask", handlers.AskAIAssistant)
}
