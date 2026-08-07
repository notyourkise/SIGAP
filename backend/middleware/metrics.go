package middleware

import (
	"time"

	"sigap-backend/monitoring"

	"github.com/gofiber/fiber/v2"
)

// Metrics adalah middleware Fiber yang mencatat setiap request HTTP ke collector
// metrik (Task 7.4):
//   - menaikkan counter `sigap_http_requests_total{method,path,status}`
//   - mengamati durasi ke histogram `sigap_http_request_duration_seconds{method,path}`
//
// Path memakai pola route (mis. /api/cctv/:id) — bukan URL aktual — agar kardinalitas
// label tetap rendah dan metrik stabil meskipun banyak id berbeda. Endpoint metrik
// itu sendiri (`/api/metrics`) di-EXCLUDE agar tidak mencemari statistik.
func Metrics() fiber.Handler {
	collector := monitoring.GetCollector()
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Lanjutkan ke handler berikutnya.
		err := c.Next()

		// Jangan hitung request ke endpoint metrik itu sendiri (hindari loop
		// & menjaga metrik mencerminkan trafik aplikasi, bukan scraper).
		if c.Route() != nil && c.Route().Path == "/api/metrics" {
			return err
		}

		duration := time.Since(start).Seconds()
		status := c.Response().StatusCode()
		method := c.Method()

		// Ambil pola route bila tersedia, fallback ke path aktual.
		path := c.Path()
		if r := c.Route(); r != nil && r.Path != "" {
			path = r.Path
		}

		collector.ObserveRequest(method, path, status, duration)
		return err
	}
}
