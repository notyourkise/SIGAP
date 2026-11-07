package routes

import (
	"github.com/gofiber/fiber/v2"
	"sigap-backend/handlers"
	"sigap-backend/middleware"
)

func SetupIncidentRoutes(router fiber.Router) {
	group := router.Group("/incidents", middleware.Protected(), middleware.RequireRole("admin"), middleware.SiteContext())
	group.Get("/", handlers.GetIncidents)
	group.Get("/planned-outages", handlers.ListPlannedOutagesHandler)
	group.Post("/planned-outages", handlers.CreatePlannedOutageHandler)
	group.Post("/planned-outages/:outage_id/end", handlers.EndPlannedOutageHandler)
	group.Post("/outbox/:outbox_id/retry", handlers.RetryIncidentOutbox)

	group.Post("/:id/assign", handlers.AssignIncident)
	group.Post("/:id/camera-tasks", handlers.CreateIncidentCameraFieldTask)
	group.Post("/:id/resolve", handlers.ResolveIncident)
	group.Post("/:id/merge", handlers.MergeIncidentsHandler)
	group.Post("/:id/escalate-replacement", handlers.EscalateIncidentReplacement)
}
