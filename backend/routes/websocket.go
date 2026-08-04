package routes

import (
	"sigap-backend/config"
	"sigap-backend/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/golang-jwt/jwt/v5"
)

// SetupWebSocketRoutes mendaftarkan rute WebSocket real-time broadcast
// @Summary      Koneksi Real-time WebSocket Broadcast
// @Description  Membuka koneksi bi-directional WebSocket real-time untuk menerima siaran langsung event sistem (CCTV_UPDATED, MAINTENANCE_CREATED, MAINTENANCE_UPDATED, AUDIT_LOG_ADDED). Autentikasi wajib: sertakan token JWT sebagai query parameter `token`.
// @Tags         WebSocket
// @Produce      json
// @Param        token query string true "Token JWT (Bearer) untuk autentikasi"
// @Success      101 {string} string "Switching Protocols ke WebSocket"
// @Failure      400 {object} map[string]interface{} "Header Upgrade: websocket wajib dikirimkan"
// @Failure      401 {object} map[string]interface{} "Token tidak valid / tidak ditemukan"
// @Router       /ws [get]
func SetupWebSocketRoutes(api fiber.Router) {
	// Middleware pengecekan upgrade protokol WebSocket + autentikasi JWT.
	// Browser tidak dapat mengirim header Authorization pada handshake WebSocket,
	// sehingga token diambil dari cookie httpOnly sigap_session (dikirim otomatis
	// oleh browser pada koneksi same-origin) dengan fallback query parameter `token`.
	api.Use("/ws", func(c *fiber.Ctx) error {
		if !websocket.IsWebSocketUpgrade(c) {
			return fiber.ErrUpgradeRequired
		}

		tokenString := c.Query("token")
		if tokenString == "" {
			tokenString = utils.ExtractTokenString(c)
		}
		if tokenString == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "Token otentikasi tidak ditemukan",
			})
		}

		// SECURITY: Periksa apakah token sudah direvoke (user sudah logout)
		if utils.IsTokenRevoked(tokenString) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "Token telah dicabut. Silakan login kembali.",
			})
		}

		token, err := utils.ValidateToken(tokenString)
		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "Token tidak valid atau telah kedaluwarsa",
			})
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "Klaim token tidak valid",
			})
		}

		// SECURITY: Pastikan versi token cocok dengan versi terbaru pengguna
		// (token dicabut saat password di-reset / role diubah).
		if err := utils.ValidateTokenVersion(config.DB, token); err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "Sesi telah dicabut. Silakan login kembali.",
			})
		}

		c.Locals("user_id", claims["user_id"])
		c.Locals("role", claims["role"])
		c.Locals("allowed", true)
		return c.Next()
	})

	// GET /api/ws — Handshake & Listening Loop WebSocket
	api.Get("/ws", websocket.New(func(c *websocket.Conn) {
		// Ambil kembali Fiber context asli dari middleware upgrade; identitas
		// pengguna (user_id & role) sudah divalidasi & disimpan di c.Locals.
		var userID, role string
		if fc, ok := c.Locals("fiberctx").(*fiber.Ctx); ok {
			userID, _ = fc.Locals("user_id").(string)
			role, _ = fc.Locals("role").(string)
		}
		utils.Hub.Register(c, userID, role)
		defer utils.Hub.Unregister(c)

		// Loop pembacaan pesan (menjaga koneksi tetap terbuka & merespons ping)
		for {
			mt, msg, err := c.ReadMessage()
			if err != nil {
				break
			}
			// Ping-pong echo sederhana jika client mengirim pesan
			if string(msg) == "ping" {
				if err := c.WriteMessage(mt, []byte("pong")); err != nil {
					break
				}
			}
		}
	}))
}
