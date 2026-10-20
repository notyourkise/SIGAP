package monitoring

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// newTestCollector mengembalikan collector kosong yang terisolasi untuk pengujian.
func newTestCollector() *Collector {
	c := GetCollector()
	c.Reset()
	return c
}

func TestObserveRequest_CounterAndHistogram(t *testing.T) {
	c := newTestCollector()

	c.ObserveRequest("GET", "/api/cctv/:id", 200, 0.05)
	c.ObserveRequest("GET", "/api/cctv/:id", 200, 0.02)
	c.ObserveRequest("GET", "/api/cctv/:id", 404, 0.1)
	c.ObserveRequest("POST", "/api/auth/login", 401, 0.5)

	out := c.Render()

	// Counter: total 2 untuk (GET, :id, 200), 1 untuk (GET, :id, 404), 1 untuk (POST, login, 401).
	assert.Contains(t, out, `sigap_http_requests_total{method="GET",path="/api/cctv/:id",status="200"} 2`)
	assert.Contains(t, out, `sigap_http_requests_total{method="GET",path="/api/cctv/:id",status="404"} 1`)
	assert.Contains(t, out, `sigap_http_requests_total{method="POST",path="/api/auth/login",status="401"} 1`)

	// Histogram kumulatif: durasi untuk GET /:id adalah 0.05, 0.02, 0.1.
	//   le=0.005 -> 0; le=0.025 -> 1 (0.02); le=0.05 -> 2 (0.02,0.05); le=0.1 -> 3.
	hist := out
	assert.Contains(t, hist, `sigap_http_request_duration_seconds{method="GET",path="/api/cctv/:id",le="0.005"} 0`)
	assert.Contains(t, hist, `sigap_http_request_duration_seconds{method="GET",path="/api/cctv/:id",le="0.025"} 1`)
	assert.Contains(t, hist, `sigap_http_request_duration_seconds{method="GET",path="/api/cctv/:id",le="0.05"} 2`)
	assert.Contains(t, hist, `sigap_http_request_duration_seconds{method="GET",path="/api/cctv/:id",le="0.1"} 3`)
	assert.Contains(t, hist, `sigap_http_request_duration_seconds{method="GET",path="/api/cctv/:id",le="+Inf"} 3`)
	assert.Contains(t, hist, `sigap_http_request_duration_seconds_count{method="GET",path="/api/cctv/:id"} 3`)
}

func TestRender_IncludesHelpTypeAndGauges(t *testing.T) {
	c := newTestCollector()
	c.SetDBPool(5, 3, 50)
	c.SetWSActive(7)
	c.SetUploadStorageBytes(12345)

	out := c.Render()

	// HELP & TYPE untuk setiap metrik yang diminta.
	for _, m := range []string{
		"sigap_http_requests_total",
		"sigap_http_request_duration_seconds",
		"sigap_db_pool_open_connections",
		"sigap_db_pool_idle_connections",
		"sigap_ws_active_connections",
		"sigap_upload_storage_bytes",
	} {
		assert.Contains(t, out, "# HELP "+m)
		assert.Contains(t, out, "# TYPE "+m)
	}

	// Nilai gauge.
	assert.Contains(t, out, "sigap_db_pool_open_connections 5")
	assert.Contains(t, out, "sigap_db_pool_idle_connections 3")
	assert.Contains(t, out, "sigap_ws_active_connections 7")
	assert.Contains(t, out, "sigap_upload_storage_bytes 12345")
}

func TestRender_EmptyState(t *testing.T) {
	c := newTestCollector()

	out := c.Render()

	// Collector baru tanpa request tetap menghasilkan output valid Prometheus.
	assert.Contains(t, out, "sigap_http_requests_total 0")
	assert.Contains(t, out, "sigap_http_request_duration_seconds_count 0")
	assert.Contains(t, out, "sigap_ws_active_connections 0")
	assert.Contains(t, out, "sigap_upload_storage_bytes 0")
	// Harus dimulai dengan baris HELP (format valid).
	assert.True(t, strings.HasPrefix(out, "# HELP"))
}

func TestRender_ToleratesNilAndShortHistogramEntries(t *testing.T) {
	c := newTestCollector()

	c.mu.Lock()
	c.httpDuration[histogramKey{method: "GET", path: "/nil"}] = nil
	c.httpDuration[histogramKey{method: "GET", path: "/short"}] = &histogramData{
		buckets: []uint64{1},
		count:   1,
		sum:     0.01,
	}
	c.mu.Unlock()

	var out string
	assert.NotPanics(t, func() {
		out = c.Render()
	})

	// Entri nil dirender sebagai nol; entri bucket pendek tidak boleh
	// menyebabkan index-out-of-range.
	assert.Contains(t, out, `sigap_http_request_duration_seconds{method="GET",path="/nil",le="0.005"} 0`)
	assert.Contains(t, out, `sigap_http_request_duration_seconds{method="GET",path="/short",le="0.005"} 1`)
	assert.Contains(t, out, `sigap_http_request_duration_seconds{method="GET",path="/short",le="10"} 0`)
	assert.Contains(t, out, `sigap_http_request_duration_seconds_count{method="GET",path="/nil"} 0`)
}

func TestObserveRequest_RepairsMalformedHistogramEntry(t *testing.T) {
	c := newTestCollector()
	key := histogramKey{method: "POST", path: "/repair"}

	c.mu.Lock()
	c.httpDuration[key] = nil
	c.mu.Unlock()

	assert.NotPanics(t, func() {
		c.ObserveRequest("POST", "/repair", 201, 0.02)
	})

	c.mu.RLock()
	data := c.httpDuration[key]
	c.mu.RUnlock()
	assert.NotNil(t, data)
	assert.Len(t, data.buckets, len(httpBucketBounds))
	assert.Equal(t, uint64(1), data.count)

	out := c.Render()
	assert.Contains(t, out, `sigap_http_request_duration_seconds{method="POST",path="/repair",le="0.025"} 1`)
}

func TestReset_ClearsAll(t *testing.T) {
	c := newTestCollector()
	c.ObserveRequest("GET", "/x", 200, 0.1)
	c.SetWSActive(3)

	c.Reset()

	out := c.Render()
	assert.NotContains(t, out, `sigap_http_requests_total{method="GET",path="/x",status="200"}`)
	assert.Contains(t, out, "sigap_ws_active_connections 0")
}

func TestEscapeLabel(t *testing.T) {
	assert.Equal(t, `a\\b`, escapeLabel(`a\b`))
	assert.Equal(t, `a\"b`, escapeLabel(`a"b`))
	assert.Equal(t, `a\nb`, escapeLabel("a\nb"))
}

func TestFormatFloat_NoExponent(t *testing.T) {
	// Nilai 0.000001 tidak boleh memakai notasi eksponensial (1e-06).
	assert.Equal(t, "0.000001", formatFloat(0.000001))
	assert.Equal(t, "0", formatFloat(0))
	assert.Equal(t, "5", formatFloat(5))
}
