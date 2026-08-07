package middleware

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"sigap-backend/monitoring"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestMetrics_ObservesRequests(t *testing.T) {
	// Reset collector agar state test terisolasi.
	monitoring.GetCollector().Reset()

	app := fiber.New()
	app.Use(Metrics())
	app.Get("/hello", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})
	app.Get("/api/metrics", monitoring.Handler)

	// Kirim beberapa request ke /hello.
	for i := 0; i < 3; i++ {
		resp, err := app.Test(httptest.NewRequest("GET", "/hello", nil))
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	}

	out := monitoring.GetCollector().Render()
	// Counter harus mencatat 3 request GET /hello status 200.
	assert.Contains(t, out, `sigap_http_requests_total{method="GET",path="/hello",status="200"} 3`)
	assert.Contains(t, out, `sigap_http_request_duration_seconds_count{method="GET",path="/hello"} 3`)
}

func TestMetrics_ExcludesMetricsEndpoint(t *testing.T) {
	monitoring.GetCollector().Reset()

	app := fiber.New()
	app.Use(Metrics())
	app.Get("/api/metrics", monitoring.Handler)

	// Request ke /api/metrics TIDAK boleh menambah counter-nya sendiri.
	_, err := app.Test(httptest.NewRequest("GET", "/api/metrics", nil))
	assert.NoError(t, err)

	out := monitoring.GetCollector().Render()
	assert.NotContains(t, out, `path="/api/metrics"`)
	// Tidak ada observasi sama sekali (endpoint dikecualikan).
	assert.Contains(t, out, "sigap_http_requests_total 0")
}

func TestMetrics_RecordsNon200Status(t *testing.T) {
	monitoring.GetCollector().Reset()

	app := fiber.New()
	app.Use(Metrics())
	app.Get("/missing", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNotFound)
	})

	resp, err := app.Test(httptest.NewRequest("GET", "/missing", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)

	out := monitoring.GetCollector().Render()
	assert.Contains(t, out, `sigap_http_requests_total{method="GET",path="/missing",status="404"} 1`)
}

func TestMetrics_HandlerReturnsPrometheusText(t *testing.T) {
	app := fiber.New()
	app.Get("/api/metrics", monitoring.Handler)

	resp, err := app.Test(httptest.NewRequest("GET", "/api/metrics", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	ct := resp.Header.Get("Content-Type")
	assert.True(t, strings.HasPrefix(ct, "text/plain"), "Content-Type harus text/plain, dapat: %s", ct)

	b, _ := io.ReadAll(resp.Body)
	text := string(b)
	assert.Contains(t, text, "# TYPE sigap_http_requests_total counter")
	assert.Contains(t, text, "# TYPE sigap_http_request_duration_seconds histogram")
	assert.Contains(t, text, "# TYPE sigap_ws_active_connections gauge")
}
