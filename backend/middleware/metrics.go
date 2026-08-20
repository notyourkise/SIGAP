package middleware

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"sigap-backend/monitoring"
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

// IsTrustedMetricsAccess checks if the request comes from a trusted internal source.
// Used to restrict access to /api/metrics endpoint.
func IsTrustedMetricsAccess() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ip := c.IP() // Returns string like "127.0.0.1" or "::1"
		
		// Allow loopback addresses (::1, 127.0.0.1)
		if ip == "127.0.0.1" || ip == "::1" || ip == "" {
			return c.Next()
		}
		
		// Check if it's from internal Docker network (172.x.x.x range)
		internalPrefixes := []string{"172.16.", "172.17.", "172.18.", "172.19.", 
		                             "172.20.", "172.21.", "172.22.", "172.23.", 
		                             "172.24.", "172.25.", "172.26.", "172.27.", 
		                             "172.28.", "172.29.", "172.30.", "172.31."}
							 
		for _, prefix := range internalPrefixes {
			if strings.HasPrefix(ip, prefix) {
				return c.Next()
			}
		}
		
		// Request not from trusted source - return 403 Forbidden
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "error",
			"message": "Metrics endpoint accessible only from internal network",
		})
	}
}
