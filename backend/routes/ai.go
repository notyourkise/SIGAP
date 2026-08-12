package routes

import (
	"sigap-backend/handlers"
	"sigap-backend/middleware"

	"github.com/gofiber/fiber/v2"
)

// SetupAIRoutes mendefinisikan rute modul AI Detection (Task 14B, Sprint 38).
//
// Pembagian akses:
//   - POST /ai/detection-event : INTERNAL — hanya dipanggil service sigap_ai
//     (diproteksi header X-AI-Secret, bukan JWT).
//   - GET  /ai/counts, /ai/counts/:id, GET /ai/sessions : khusus Admin.
//   - POST /ai/sessions & DELETE /ai/sessions/:id : semua role login (proxy ke
//     service AI); frontend /monitor memanggilnya untuk mengatur session.
//
// Rute statis didaftarkan sebelum rute dinamis (gotcha Fiber route ordering).
func SetupAIRoutes(router fiber.Router) {
	ai := router.Group("/ai")

	// Endpoint internal dari service sigap_ai (tanpa JWT — hanya X-AI-Secret).
	ai.Post("/detection-event", handlers.ReceiveDetectionEvent)

	// Ringkasan & detail hitungan — khusus Admin.
	ai.Get("/counts", middleware.RequireRole("admin"), handlers.GetAICounts)
	ai.Get("/counts/:id", middleware.RequireRole("admin"), handlers.GetAICountByCCTV)
	// Tren hitungan harian per kamera (1–90 hari) — khusus Admin.
	ai.Get("/counts/:id/trend", middleware.RequireRole("admin"), handlers.GetAICountTrend)

	// Proxy ke service sigap_ai (frontend memanggil ini).
	ai.Post("/sessions", middleware.Protected(), handlers.ProxyActivateAISession)
	ai.Delete("/sessions/:id", middleware.Protected(), handlers.ProxyDeactivateAISession)
	ai.Get("/sessions", middleware.RequireRole("admin"), handlers.ProxyListAISessions)
}
