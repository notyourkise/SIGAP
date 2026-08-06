package routes

import (
	"sigap-backend/handlers"
	"sigap-backend/middleware"

	"github.com/gofiber/fiber/v2"
)

// SetupMaintenanceScheduleRoutes mendefinisikan rute modul jadwal pemeliharaan
// berulang (Fase 5.4). Seluruh rute khusus admin: Protected + RequireRole.
func SetupMaintenanceScheduleRoutes(router fiber.Router) {
	scheduleGroup := router.Group("/maintenance-schedules", middleware.Protected(), middleware.RequireRole("admin"))

	// Rute statis didaftarkan sebelum rute dinamis (gotcha Fiber route ordering).
	// Catatan: tidak ada rute statis di level grup ini selain pola "/:id",
	// namun urutan tetap dijaga: list & create dulu, lalu detail/dinamis.
	scheduleGroup.Get("/", handlers.GetAllMaintenanceSchedules)
	scheduleGroup.Post("/", handlers.CreateMaintenanceSchedule)
	scheduleGroup.Get("/:id", handlers.GetMaintenanceScheduleByID)
	scheduleGroup.Put("/:id", handlers.UpdateMaintenanceSchedule)
	scheduleGroup.Put("/:id/toggle", handlers.ToggleMaintenanceSchedule)
	scheduleGroup.Delete("/:id", handlers.DeleteMaintenanceSchedule)
}
