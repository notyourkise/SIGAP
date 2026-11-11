package middleware

import (
	"net"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"sigap-backend/monitoring"
)

var dockerInternalCIDR = func() *net.IPNet {
	_, ipNet, err := net.ParseCIDR("172.16.0.0/12")
	if err != nil {
		panic(err)
	}
	return ipNet
}()

// isTrustedMetricsIP memvalidasi IP secara ketat (fail-closed) agar hanya
// loopback lokal atau subnet internal Docker (172.16.0.0/12) yang dapat membaca
// /api/metrics. IP kosong, tidak valid, atau alamat publik selalu ditolak.
func isTrustedMetricsIP(rawIP string) bool {
	trimmed := strings.TrimSpace(rawIP)
	if trimmed == "" {
		return false
	}
	ip := net.ParseIP(trimmed)
	if ip == nil {
		return false
	}
	if ip4 := ip.To4(); ip4 != nil {
		ip = ip4
	}
	if ip.IsUnspecified() {
		return false
	}
	if ip.IsLoopback() {
		return true
	}
	return dockerInternalCIDR.Contains(ip)
}

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
// Used to restrict access to /api/metrics endpoint (fail-closed on empty/public IPs).
func IsTrustedMetricsAccess() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if isTrustedMetricsIP(c.IP()) {
			return c.Next()
		}

		// Request not from trusted source - return 403 Forbidden
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"status":  "error",
			"message": "Metrics endpoint accessible only from internal network",
		})
	}
}
