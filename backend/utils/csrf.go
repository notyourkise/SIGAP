package utils

import (
	"crypto/rand"
	"encoding/hex"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// ExtractCookieValues mengembalikan seluruh nilai cookie dengan nama tertentu
// sesuai urutan kemunculannya di header Cookie.
//
// Browser dapat mengirimkan BEBERAPA cookie dengan nama sama tetapi Path
// berbeda — misalnya cookie stale warisan versi lama di path /api atau /api/
// yang gagal dihapus, berdampingan dengan cookie sesi aktif di path /. Sesuai
// RFC 6265 §5.4, cookie dengan Path terpanjang dikirim lebih dulu, sehingga
// nilai pada path terpendek ("/") selalu berada paling akhir.
func ExtractCookieValues(c *fiber.Ctx, name string) []string {
	raw := string(c.Request().Header.Peek(fiber.HeaderCookie))
	if raw == "" {
		return nil
	}

	var values []string
	for _, part := range strings.Split(raw, ";") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 || kv[0] != name {
			continue
		}
		values = append(values, kv[1])
	}
	return values
}

// ExtractLastCookieValue mengambil nilai cookie TERAKHIR dengan nama tertentu.
// Cookie stale ber-Path panjang dikirim lebih dulu oleh browser, jadi nilai
// pada path terpendek (sesi aktif di "/") selalu menang. Ini membuat server
// kebal terhadap cookie duplikat warisan versi lama yang tidak terhapus.
func ExtractLastCookieValue(c *fiber.Ctx, name string) string {
	values := ExtractCookieValues(c, name)
	if len(values) == 0 {
		return ""
	}
	return values[len(values)-1]
}

const (
	// SessionCookieName nama cookie httpOnly yang menyimpan JWT sesi.
	// Token tidak lagi diekspos ke JavaScript (migrasi dari localStorage).
	SessionCookieName = "sigap_session"
	// CSRFCookieName nama cookie CSRF (non-httpOnly) untuk pola double-submit.
	CSRFCookieName = "sigap_csrf"
	// CSRFHeaderName nama header untuk mengirim token CSRF.
	CSRFHeaderName = "X-CSRF-Token"
)

// GenerateCSRFToken menghasilkan token CSRF acak 32-byte (hex).
func GenerateCSRFToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

// ExtractTokenString mengambil token JWT dari cookie sesi terlebih dahulu,
// lalu fallback ke header Authorization Bearer (untuk client API/Swagger).
func ExtractTokenString(c *fiber.Ctx) string {
	// Pakai nilai cookie TERAKHIR: bila browser mengirim duplikat sigap_session
	// (stale lama di path /api + aktif di /), nilai aktif di "/" selalu paling
	// akhir menurut urutan RFC 6265 §5.4 dan harus diprioritaskan.
	if cookieToken := ExtractLastCookieValue(c, SessionCookieName); cookieToken != "" {
		return cookieToken
	}
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return ""
	}
	parts := strings.Split(authHeader, " ")
	if len(parts) == 2 && parts[0] == "Bearer" {
		return parts[1]
	}
	return ""
}
