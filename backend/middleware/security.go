package middleware

import (
	"os"

	"github.com/gofiber/fiber/v2"
)

// SecurityHeaders memasang header HTTP keamanan standar industri pada setiap respons.
// Header-header ini melindungi dari serangan umum seperti XSS, clickjacking,
// MIME sniffing, dan information leakage.
func SecurityHeaders() fiber.Handler {
	// CSP strict & HSTS hanya diaktifkan pada environment production:
	//   - CSP strict memblokir resource CDN Swagger UI, sehingga hanya aman
	//     dipasang saat produksi (Swagger UI dinonaktifkan di produksi).
	//   - HSTS mewajibkan browser selalu memakai HTTPS; menambahkannya saat
	//     masih HTTP polos (development) justru bisa melumpuhkan akses.
	isProduction := os.Getenv("APP_ENV") == "production"

	return func(c *fiber.Ctx) error {
		// Cegah browser menebak tipe konten (MIME sniffing attack)
		c.Set("X-Content-Type-Options", "nosniff")

		// Cegah halaman ini dimuat dalam iframe (Clickjacking protection)
		c.Set("X-Frame-Options", "DENY")

		// Aktifkan perlindungan XSS bawaan browser (browser lama)
		c.Set("X-XSS-Protection", "1; mode=block")

		// Sembunyikan informasi server untuk mengurangi information disclosure
		c.Set("Server", "SIGAP")

		// Batasi akses referrer untuk privasi
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Batasi resource lintas domain yang boleh dimuat di halaman ini
		c.Set("X-Permitted-Cross-Domain-Policies", "none")

		// Content Security Policy (CSP) & HSTS — hanya pada environment produksi:
		//   - CSP strict ini akan memblokir resource CDN Swagger UI, sehingga
		//     hanya diaktifkan saat produksi (Swagger dinonaktifkan di produksi).
		//   - HSTS memaksa browser selalu memakai HTTPS (dipasang di belakang
		//     reverse proxy Nginx yang menangani TLS).
		if isProduction {
			c.Set("Content-Security-Policy", "default-src 'none'; img-src 'self' data:; media-src 'self'; connect-src 'self'; style-src 'self' 'unsafe-inline'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
			c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		return c.Next()
	}
}
