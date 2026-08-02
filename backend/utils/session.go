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
// Cookie bersifat session cookie (tanpa MaxAge) sehingga otomatis terhapus
// saat browser ditutup — tidak ada sesi yang tersimpan permanen.
func SetSessionCookie(c *fiber.Ctx, token string) {
	c.Cookie(&fiber.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		HTTPOnly: true,
		Secure:   cookieSecure(),
		SameSite: "Lax",
		Path:     "/",
	})
}

// SetCSRFCookie memasang cookie CSRF (non-httpOnly) untuk pola double-submit.
// Cookie bersifat session cookie sehingga ikut terhapus saat browser ditutup.
func SetCSRFCookie(c *fiber.Ctx, token string) {
	c.Cookie(&fiber.Cookie{
		Name:     CSRFCookieName,
		Value:    token,
		HTTPOnly: false,
		Secure:   cookieSecure(),
		SameSite: "Lax",
		Path:     "/",
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
