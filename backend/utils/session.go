package utils

import (
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
)

var sessionCookiePaths = []string{"/", "/api", "/api/"}

// cookieSecure menentukan apakah cookie hanya dikirim lewat HTTPS.
func cookieSecure() bool {
	return os.Getenv("APP_ENV") == "production"
}

// SetSessionCookie memasang cookie httpOnly yang menyimpan token JWT sesi.
// MaxAge diselaraskan dengan durasi kedaluwarsa JWT (5 jam) agar cookie
// bertahan selama token masih valid, termasuk saat refresh halaman.
func SetSessionCookie(c *fiber.Ctx, token string) {
	writeCookie(c, &fiber.Cookie{
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
	writeCookie(c, &fiber.Cookie{
		Name:     CSRFCookieName,
		Value:    token,
		HTTPOnly: false,
		Secure:   cookieSecure(),
		SameSite: "Lax",
		Path:     "/",
		MaxAge:   18000, // 5 jam, selaras dengan cookie sesi
	})
}

// writeCookie menambahkan header Set-Cookie tanpa mengganti cookie lain yang
// memiliki nama sama tetapi Path berbeda. Ini diperlukan saat membersihkan
// cookie legacy pada beberapa path dalam satu response.
func writeCookie(c *fiber.Ctx, cookie *fiber.Cookie) {
	fcookie := fasthttp.AcquireCookie()
	defer fasthttp.ReleaseCookie(fcookie)

	fcookie.SetKey(cookie.Name)
	fcookie.SetValue(cookie.Value)
	fcookie.SetPath(cookie.Path)
	fcookie.SetDomain(cookie.Domain)
	if !cookie.SessionOnly {
		fcookie.SetMaxAge(cookie.MaxAge)
		fcookie.SetExpire(cookie.Expires)
	}
	fcookie.SetSecure(cookie.Secure)
	fcookie.SetHTTPOnly(cookie.HTTPOnly)
	fcookie.SetSameSite(fasthttp.CookieSameSiteLaxMode)

	c.Context().Response.Header.AddBytesV(fiber.HeaderSetCookie, fcookie.Cookie())
}

// expireCookie menghapus satu cookie dengan cara menuliskan nilai kosong
// bertanggal kedaluwarsa di masa lalu. Atribut Path/SameSite/Secure HARUS
// sama persis dengan saat cookie dibuat, karena browser mencocokkan cookie
// berdasarkan nama + domain + path — bila berbeda, cookie asli tidak terhapus.
func expireCookie(c *fiber.Ctx, name string, httpOnly bool, path string) {
	writeCookie(c, &fiber.Cookie{
		Name:     name,
		Value:    "",
		HTTPOnly: httpOnly,
		Secure:   cookieSecure(),
		SameSite: "Lax",
		Path:     path,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
}

// ClearSessionCookies menghapus cookie sesi & CSRF dari browser klien.
func ClearSessionCookies(c *fiber.Ctx) {
	for _, path := range sessionCookiePaths {
		expireCookie(c, SessionCookieName, true, path)
		expireCookie(c, CSRFCookieName, false, path)
	}
}
