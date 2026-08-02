package middleware

import (
	"strings"

	"sigap-backend/utils"

	"github.com/gofiber/fiber/v2"
)

// CSRFProtection menerapkan proteksi CSRF dengan pola double-submit cookie.
//
// Aturan:
//   - Metode aman (GET/HEAD/OPTIONS) selalu dilewati.
//   - Request dengan header Authorization Bearer (client API) dilewati —
//     token Bearer tidak dikirim otomatis oleh browser sehingga tidak rentan CSRF.
//   - Request mutasi tanpa cookie sesi dilewati (mis. login publik).
//   - Untuk request mutasi yang terautentikasi via cookie, header
//     X-CSRF-Token wajib cocok dengan nilai cookie sigap_csrf.
func CSRFProtection() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Method() == "GET" || c.Method() == "HEAD" || c.Method() == "OPTIONS" {
			return c.Next()
		}

		// Endpoint login bersifat publik — tidak ada resource terautentikasi
		// yang dilindungi. Sisa cookie sesi lama dari sesi sebelumnya TIDAK
		// boleh memicu pengecekan CSRF pada percobaan login.
		if c.Path() == "/api/auth/login" {
			return c.Next()
		}

		// Client API memakai Bearer token — tidak rentan CSRF.
		if authHeader := c.Get("Authorization"); authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			return c.Next()
		}

		// Tidak ada sesi cookie — tidak ada resource yang perlu dilindungi.
		if c.Cookies(utils.SessionCookieName) == "" {
			return c.Next()
		}

		cookieToken := c.Cookies(utils.CSRFCookieName)
		headerToken := c.Get(utils.CSRFHeaderName)
		if cookieToken == "" || headerToken == "" || cookieToken != headerToken {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"status":  "error",
				"message": "Permintaan tidak sah (CSRF token tidak cocok)",
			})
		}

		return c.Next()
	}
}
