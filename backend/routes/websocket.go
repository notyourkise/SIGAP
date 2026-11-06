package routes

import (
	"sigap-backend/config"
	"sigap-backend/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// SetupWebSocketRoutes mendaftarkan rute WebSocket real-time broadcast
// @Summary      Koneksi Real-time WebSocket Broadcast
// @Description  Membuka koneksi bi-directional WebSocket real-time untuk menerima siaran langsung event sistem (CCTV_UPDATED, MAINTENANCE_CREATED, MAINTENANCE_UPDATED, AUDIT_LOG_ADDED). Event AUDIT_LOG_ADDED hanya dikirim ke koneksi admin. Autentikasi wajib: sertakan token JWT sebagai query parameter `token`.
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
		mustChangePassword, err := utils.ValidateTokenVersionAndMustChange(config.DB, token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "Sesi telah dicabut. Silakan login kembali.",
			})
		}

		// Pengguna yang wajib ganti sandi tidak boleh mengakses stream WebSocket operasional
		if mustChangePassword {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"status":  "error",
				"message": "Anda wajib mengganti password sebelum dapat mengakses resource ini",
			})
		}

		c.Locals("user_id", claims["user_id"])
		c.Locals("role", claims["role"])
		c.Locals("allowed", true)
		return c.Next()
	})

	// GET /api/ws — Handshake & Listening Loop WebSocket
	api.Get("/ws", websocket.New(func(c *websocket.Conn) {
		// Identitas pengguna (user_id & role) sudah divalidasi middleware
		// upgrade dan disalin oleh library websocket ke locals koneksi.
		userID, _ := c.Locals("user_id").(string)
		role, _ := c.Locals("role").(string)

		utils.Hub.Register(c, userID, role, loadUserSiteIDs(userID, role))
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

// loadUserSiteIDs memuat daftar site yang ditugaskan ke pengguna untuk
// keperluan filter broadcast WebSocket. Admin tidak dibatasi site sehingga
// tidak perlu query. Kegagalan query diperlakukan fail closed (tanpa site)
// agar koneksi non-admin tidak menerima event lintas site.
func loadUserSiteIDs(userID string, role string) []string {
	if role == "admin" || userID == "" || config.DB == nil {
		return nil
	}

	var rows []uuid.UUID
	if err := config.DB.Table("user_sites").
		Where("user_id = ?", userID).
		Pluck("site_id", &rows).Error; err != nil {
		utils.Log.Warnf("⚠️ Gagal memuat penugasan site WebSocket user %s: %v", userID, err)
		return nil
	}

	siteIDs := make([]string, 0, len(rows))
	for _, siteID := range rows {
		siteIDs = append(siteIDs, siteID.String())
	}
	return siteIDs
}
