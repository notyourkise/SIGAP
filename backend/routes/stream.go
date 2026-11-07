package routes

import (
	"sigap-backend/handlers"
	"sigap-backend/middleware"
	"sigap-backend/models"

	"github.com/gofiber/fiber/v2"
)

// SetupStreamRoutes mendefinisikan rute modul Streaming CCTV (Task 9.3).
// Seluruh endpoint streaming bersifat ADMIN-ONLY (Protected + RequireRole + SiteContext),
// kecuali /streams/monitor-heartbeat yang juga menerima API key eksternal ber-scope health:read
// untuk pemantauan liveness di luar server SIGAP (Keputusan #6).
func SetupStreamRoutes(router fiber.Router) {
	router.Get("/streams/monitor-heartbeat", middleware.APIKeyOrProtected(models.ScopeHealthRead), handlers.GetStreamMonitorHeartbeat)

	streamGroup := router.Group("/streams", middleware.Protected(), middleware.RequireRole("admin"), middleware.SiteContext())

	// Rute validasi akses media (dipakai Nginx auth_request pada /stream/* dan /hls/*).
	streamGroup.Get("/auth", handlers.AuthorizeStreamAccess)
	streamGroup.Head("/auth", handlers.AuthorizeStreamAccess)
	streamGroup.Post("/auth", handlers.AuthorizeStreamAccess)

	// Rute STATIS didaftarkan sebelum rute dinamis (gotcha Fiber route ordering).
	streamGroup.Get("/status", handlers.GetStreamStatus)
	streamGroup.Get("/readiness", handlers.GetMediaMTXReadiness)
	streamGroup.Get("/health", handlers.GetStreamHealth)
	streamGroup.Get("/live-status", handlers.GetLiveStatus)
	streamGroup.Post("/start-all", handlers.StartAllStreams)
	streamGroup.Post("/stop-all", handlers.StopAllStreams)

	// Rute DINAMIS dengan parameter :id.
	streamGroup.Post("/:id/start", handlers.StartStream)
	streamGroup.Post("/:id/verify", handlers.VerifyStream)
	streamGroup.Post("/:id/stop", handlers.StopStream)
	streamGroup.Get("/:id/url", handlers.GetStreamURL)
}

// SetupStreamMediaRoutes mendaftarkan jalur media langsung (/stream/* untuk WHEP
// dan /hls/* untuk HLS) dengan proteksi autentikasi Admin + isolasi Site.
func SetupStreamMediaRoutes(router fiber.Router) {
	mediaChain := []fiber.Handler{
		middleware.GlobalLimiter(),
		middleware.Protected(),
		middleware.RequireRole("admin"),
		middleware.SiteContext(),
	}
	router.All("/stream/*", append(mediaChain, handlers.ProxyWHEPStream)...)
	router.All("/hls/*", append(mediaChain, handlers.ProxyHLSStream)...)
}
