package utils

import (
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
)

// cookieSecure menentukan apakah cookie hanya dikirim lewat HTTPS.
func cookieSecure() bool {
	return os.Getenv("APP_ENV") == "production"
}

// SetSessionCookie memasang cookie httpOnly yang menyimpan token JWT sesi.
// MaxAge diselaraskan dengan durasi kedaluwarsa JWT (5 jam) agar cookie
// bertahan selama token masih valid, termasuk saat refresh halaman.
func SetSessionCookie(c *fiber.Ctx, token string) {
	c.Cookie(&fiber.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		HTTPOnly: true,
		Secure:   cookieSecure(),
		SameSite: "Lax",
		Path:     "/",
		MaxAge:   18000, // 5 jam, selaras dengan exp JWT di jwt.go
	})
}

// SetCSRFCookie memasang cookie CSRF (non-httpOnly) untuk pola double-submit.
// MaxAge diselaraskan dengan cookie sesi agar keduanya tetap sinkron.
func SetCSRFCookie(c *fiber.Ctx, token string) {
	c.Cookie(&fiber.Cookie{
		Name:     CSRFCookieName,
		Value:    token,
		HTTPOnly: false,
		Secure:   cookieSecure(),
		SameSite: "Lax",
		Path:     "/",
		MaxAge:   18000, // 5 jam, selaras dengan cookie sesi
	})
}

// expireCookie menghapus satu cookie dengan cara menuliskan nilai kosong
// bertanggal kedaluwarsa di masa lalu. Atribut Path/SameSite/Secure HARUS
// sama persis dengan saat cookie dibuat, karena browser mencocokkan cookie
// berdasarkan nama + domain + path — bila berbeda, cookie asli tidak terhapus.
func expireCookie(c *fiber.Ctx, name string, httpOnly bool) {
	c.Cookie(&fiber.Cookie{
		Name:     name,
		Value:    "",
		HTTPOnly: httpOnly,
		Secure:   cookieSecure(),
		SameSite: "Lax",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
}

// ClearSessionCookies menghapus cookie sesi & CSRF dari browser klien.
func ClearSessionCookies(c *fiber.Ctx) {
	expireCookie(c, SessionCookieName, true)
	expireCookie(c, CSRFCookieName, false)
}
