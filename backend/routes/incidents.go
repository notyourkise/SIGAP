package routes

import (
	"github.com/gofiber/fiber/v2"
	"sigap-backend/handlers"
	"sigap-backend/middleware"
)

func SetupIncidentRoutes(router fiber.Router) {
	group := router.Group("/incidents", middleware.Protected(), middleware.RequireRole("admin"), middleware.SiteContext())
	group.Get("/", handlers.GetIncidents)
	group.Post("/:id/assign", handlers.AssignIncident)
	group.Post("/:id/resolve", handlers.ResolveIncident)
}
