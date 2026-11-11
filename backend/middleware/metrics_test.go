package middleware

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"sigap-backend/config"
	"sigap-backend/monitoring"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestIsTrustedMetricsIP_Matrix(t *testing.T) {
	cases := []struct {
		name string
		ip   string
		want bool
	}{
		{name: "empty IP rejected fail-closed", ip: "", want: false},
		{name: "whitespace rejected", ip: "   ", want: false},
		{name: "invalid IP string rejected", ip: "not-an-ip", want: false},
		{name: "unspecified 0.0.0.0 rejected", ip: "0.0.0.0", want: false},
		{name: "unspecified :: rejected", ip: "::", want: false},
		{name: "IPv4 loopback allowed", ip: "127.0.0.1", want: true},
		{name: "IPv6 loopback allowed", ip: "::1", want: true},
		{name: "IPv4-mapped loopback allowed", ip: "::ffff:127.0.0.1", want: true},
		{name: "Docker bridge 172.16.0.1 allowed", ip: "172.16.0.1", want: true},
		{name: "Docker sigap_network 172.28.0.5 allowed", ip: "172.28.0.5", want: true},
		{name: "Docker subnet upper bound 172.31.255.254 allowed", ip: "172.31.255.254", want: true},
		{name: "IPv4-mapped Docker IP allowed", ip: "::ffff:172.28.0.12", want: true},
		{name: "Outside Docker /12 lower bound 172.15.255.254 rejected", ip: "172.15.255.254", want: false},
		{name: "Outside Docker /12 upper bound 172.32.0.1 rejected", ip: "172.32.0.1", want: false},
		{name: "Public IPv4 rejected", ip: "203.0.113.10", want: false},
		{name: "Public IPv6 rejected", ip: "2001:db8::1", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isTrustedMetricsIP(tc.ip))
		})
	}
}

func TestIsTrustedMetricsAccess_Middleware(t *testing.T) {
	t.Run("allows real loopback TCP connection", func(t *testing.T) {
		app := fiber.New(config.NewFiberConfig(nil))
		app.Get("/api/metrics", IsTrustedMetricsAccess(), monitoring.Handler)

		ln, err := net.Listen("tcp4", "127.0.0.1:0")
		require.NoError(t, err)
		defer func() { _ = ln.Close() }()

		errCh := make(chan error, 1)
		go func() {
			errCh <- app.Listener(ln)
		}()
		defer func() { _ = app.Shutdown() }()

		resp, err := http.Get("http://" + ln.Addr().String() + "/api/metrics")
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	})

	t.Run("rejects public and unspecified peer IPs and ignores spoofed headers without trusted proxy", func(t *testing.T) {
		app := fiber.New(config.NewFiberConfig(nil))
		app.Get("/api/metrics", IsTrustedMetricsAccess(), monitoring.Handler)

		req := httptest.NewRequest("GET", "/api/metrics", nil)
		// Spoof XFF seolah berasal dari loopback — wajib diabaikan karena tanpa trusted proxy.
		req.Header.Set(fiber.HeaderXForwardedFor, "127.0.0.1")
		req.Header.Set("X-Real-IP", "172.28.0.5")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)

		var payload map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&payload))
		assert.Equal(t, "error", payload["status"])
	})

	t.Run("enforces allowlist on forwarded client IP from trusted proxy", func(t *testing.T) {
		app := fiber.New(config.NewFiberConfig([]string{"0.0.0.0"}))
		app.Get("/api/metrics", IsTrustedMetricsAccess(), monitoring.Handler)

		reqPublic := httptest.NewRequest("GET", "/api/metrics", nil)
		reqPublic.Header.Set(fiber.HeaderXForwardedFor, "198.51.100.77")
		respPublic, err := app.Test(reqPublic)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusForbidden, respPublic.StatusCode)

		reqInternal := httptest.NewRequest("GET", "/api/metrics", nil)
		reqInternal.Header.Set(fiber.HeaderXForwardedFor, "172.28.0.15")
		respInternal, err := app.Test(reqInternal)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, respInternal.StatusCode)
	})
}
