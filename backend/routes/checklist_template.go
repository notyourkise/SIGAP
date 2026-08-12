package routes

import (
	"sigap-backend/handlers"
	"sigap-backend/middleware"

	"github.com/gofiber/fiber/v2"
)

// SetupChecklistTemplateRoutes mendefinisikan rute modul template checklist
// pengecekan lapangan yang dapat dikustomisasi admin (Task 13.12). Daftar
// dapat diakses semua role yang login; operasi tulis (create/update/delete/
// reorder) khusus admin.
func SetupChecklistTemplateRoutes(router fiber.Router) {
	group := router.Group("/checklist-templates", middleware.Protected())

	// Rute statis didaftarkan SEBELUM rute dinamis /:id (gotcha Fiber route
	// ordering): /reorder harus didaftarkan lebih dulu.
	group.Put("/reorder", middleware.RequireRole("admin"), handlers.ReorderChecklistTemplates)
	group.Get("/", handlers.GetAllChecklistTemplates)
	group.Post("/", middleware.RequireRole("admin"), handlers.CreateChecklistTemplate)
	group.Put("/:id", middleware.RequireRole("admin"), handlers.UpdateChecklistTemplate)
	group.Delete("/:id", middleware.RequireRole("admin"), handlers.DeleteChecklistTemplate)
}
