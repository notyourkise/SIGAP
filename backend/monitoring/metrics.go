package monitoring

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v2"
)

// ==========================================
// SIGAP — Collector Metrik Prometheus (Task 7.4)
// ==========================================
// Implementasi ringan (tanpa library pihak ketiga, sesuai konvensi proyek) untuk
// mengumpulkan & menyajikan metrik aplikasi dalam format text exposition
// Prometheus (text/plain; version=0.0.4). Metrik yang disediakan:
//
//	# HELP/# TYPE sigap_http_requests_total{method,path,status}
//	# HELP/# TYPE sigap_http_request_duration_seconds{method,path}  (histogram)
//	# HELP/# TYPE sigap_db_pool_open_connections                  (gauge)
//	# HELP/# TYPE sigap_db_pool_idle_connections                  (gauge)
//	# HELP/# TYPE sigap_ws_active_connections                     (gauge)
//	# HELP/# TYPE sigap_upload_storage_bytes                      (gauge)
//
// Semua akses dilakukan thread-safe (sync.RWMutex) karena middleware metrik
// dipanggil dari banyak goroutine bersamaan.

// httpBucketBounds adalah batas atas (le) untuk histogram durasi request (detik).
var httpBucketBounds = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

// counterKey adalah kunci pencacah HTTP unik per (method, path, status).
type counterKey struct {
	method string
	path   string
	status int
}

// histogramKey adalah kunci histogram HTTP unik per (method, path).
type histogramKey struct {
	method string
	path   string
}

// histogramData menyimpan kumulatif bucket serta jumlah & total durasi.
type histogramData struct {
	buckets []uint64 // len == len(httpBucketBounds)
	count   uint64
	sum     float64
}

// Collector menyimpan seluruh metrik aplikasi secara thread-safe.
type Collector struct {
	mu sync.RWMutex

	// httpRequestsTotal: counter jumlah request per (method, path, status).
	httpRequests map[counterKey]uint64
	// httpRequestDuration: histogram durasi request per (method, path).
	httpDuration map[histogramKey]*histogramData

	// Gauge database connection pool (dari sql.DB.Stats()).
	dbOpenConnections  float64
	dbIdleConnections  float64
	dbMaxOpenConnections float64

	// Gauge WebSocket aktif (dari utils.Hub).
	wsActiveConnections float64

	// Gauge ukuran folder uploads (bytes).
	uploadStorageBytes float64
}

// metrics adalah instance collector global.
var (
	metrics     *Collector
	metricsOnce sync.Once
)

// GetCollector mengembalikan instance collector global (singleton).
func GetCollector() *Collector {
	metricsOnce.Do(func() {
		metrics = &Collector{
			httpRequests: make(map[counterKey]uint64),
			httpDuration: make(map[histogramKey]*histogramData),
		}
	})
	return metrics
}

// Reset mengosongkan seluruh metrik. Dipakai dalam pengujian agar state
// antar-test tidak saling mencemari.
func (c *Collector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.httpRequests = make(map[counterKey]uint64)
	c.httpDuration = make(map[histogramKey]*histogramData)
	c.dbOpenConnections = 0
	c.dbIdleConnections = 0
	c.dbMaxOpenConnections = 0
	c.wsActiveConnections = 0
	c.uploadStorageBytes = 0
}

// ObserveRequest mencatat sebuah request HTTP: menaikkan counter
// (method, path, status) dan mengamati durasinya ke histogram (method, path).
func (c *Collector) ObserveRequest(method, path string, status int, durationSeconds float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := counterKey{method: method, path: path, status: status}
	c.httpRequests[key]++

	hk := histogramKey{method: method, path: path}
	data, ok := c.httpDuration[hk]
	if !ok {
		data = &histogramData{buckets: make([]uint64, len(httpBucketBounds))}
		c.httpDuration[hk] = data
	}
	data.count++
	data.sum += durationSeconds
	for i, bound := range httpBucketBounds {
		if durationSeconds <= bound {
			data.buckets[i]++
		}
	}
}

// SetDBPool mencatat statistik connection pool database (gauge).
func (c *Collector) SetDBPool(open, idle, maxOpen float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.dbOpenConnections = open
	c.dbIdleConnections = idle
	c.dbMaxOpenConnections = maxOpen
}

// SetWSActive mencatat jumlah koneksi WebSocket aktif (gauge).
func (c *Collector) SetWSActive(n float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.wsActiveConnections = n
}

// SetUploadStorageBytes mencatat total ukuran folder uploads dalam bytes (gauge).
func (c *Collector) SetUploadStorageBytes(n float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.uploadStorageBytes = n
}

// formatFloat menulis angka dalam notasi desimal tetap (tanpa eksponensial)
// untuk kompatibilitas dan keterbacaan Prometheus text format.
func formatFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// Render menghasilkan representasi teks Prometheus dari seluruh metrik.
// Dipanggil oleh handler GET /api/metrics.
func (c *Collector) Render() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var b strings.Builder

	// ---- sigap_http_requests_total (counter) ----
	b.WriteString("# HELP sigap_http_requests_total Total jumlah request HTTP yang diterima server.\n")
	b.WriteString("# TYPE sigap_http_requests_total counter\n")
	if len(c.httpRequests) > 0 {
		keys := make([]counterKey, 0, len(c.httpRequests))
		for k := range c.httpRequests {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			if keys[i].method != keys[j].method {
				return keys[i].method < keys[j].method
			}
			if keys[i].path != keys[j].path {
				return keys[i].path < keys[j].path
			}
			return keys[i].status < keys[j].status
		})
		for _, k := range keys {
			fmt.Fprintf(&b, "sigap_http_requests_total{method=\"%s\",path=\"%s\",status=\"%d\"} %d\n",
				escapeLabel(k.method), escapeLabel(k.path), k.status, c.httpRequests[k])
		}
	} else {
		b.WriteString("sigap_http_requests_total 0\n")
	}

	// ---- sigap_http_request_duration_seconds (histogram) ----
	b.WriteString("# HELP sigap_http_request_duration_seconds Durasi pemrosesan request HTTP dalam detik.\n")
	b.WriteString("# TYPE sigap_http_request_duration_seconds histogram\n")
	if len(c.httpDuration) > 0 {
		keys := make([]histogramKey, 0, len(c.httpDuration))
		for k := range c.httpDuration {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			if keys[i].method != keys[j].method {
				return keys[i].method < keys[j].method
			}
			return keys[i].path < keys[j].path
		})
		for _, k := range keys {
			data := c.httpDuration[k]
			// Nama metrik tanpa suffix (bucket) + label method/path.
			labels := fmt.Sprintf("{method=\"%s\",path=\"%s\"",
				escapeLabel(k.method), escapeLabel(k.path))
			for i, bound := range httpBucketBounds {
				fmt.Fprintf(&b, "sigap_http_request_duration_seconds%s,le=\"%s\"} %d\n",
					labels, formatFloat(bound), data.buckets[i])
			}
			// Bucket +Inf
			fmt.Fprintf(&b, "sigap_http_request_duration_seconds%s,le=\"+Inf\"} %d\n", labels, data.count)
			// Sum & count: suffix _sum/_count adalah bagian dari NAMA metrik.
			fmt.Fprintf(&b, "sigap_http_request_duration_seconds_sum%s} %s\n", labels, formatFloat(data.sum))
			fmt.Fprintf(&b, "sigap_http_request_duration_seconds_count%s} %d\n", labels, data.count)
		}
	} else {
		for _, bound := range httpBucketBounds {
			fmt.Fprintf(&b, "sigap_http_request_duration_seconds_bucket{le=\"%s\"} 0\n", formatFloat(bound))
		}
		b.WriteString("sigap_http_request_duration_seconds_bucket{le=\"+Inf\"} 0\n")
		b.WriteString("sigap_http_request_duration_seconds_sum 0\n")
		b.WriteString("sigap_http_request_duration_seconds_count 0\n")
	}

	// ---- sigap_db_pool_open_connections (gauge) ----
	b.WriteString("# HELP sigap_db_pool_open_connections Jumlah koneksi database yang terbuka saat ini.\n")
	b.WriteString("# TYPE sigap_db_pool_open_connections gauge\n")
	fmt.Fprintf(&b, "sigap_db_pool_open_connections %s\n", formatFloat(c.dbOpenConnections))

	// ---- sigap_db_pool_idle_connections (gauge) ----
	b.WriteString("# HELP sigap_db_pool_idle_connections Jumlah koneksi database yang idle saat ini.\n")
	b.WriteString("# TYPE sigap_db_pool_idle_connections gauge\n")
	fmt.Fprintf(&b, "sigap_db_pool_idle_connections %s\n", formatFloat(c.dbIdleConnections))

	// ---- sigap_ws_active_connections (gauge) ----
	b.WriteString("# HELP sigap_ws_active_connections Jumlah koneksi WebSocket aktif saat ini.\n")
	b.WriteString("# TYPE sigap_ws_active_connections gauge\n")
	fmt.Fprintf(&b, "sigap_ws_active_connections %s\n", formatFloat(c.wsActiveConnections))

	// ---- sigap_upload_storage_bytes (gauge) ----
	b.WriteString("# HELP sigap_upload_storage_bytes Total ukuran folder uploads dalam bytes.\n")
	b.WriteString("# TYPE sigap_upload_storage_bytes gauge\n")
	fmt.Fprintf(&b, "sigap_upload_storage_bytes %s\n", formatFloat(c.uploadStorageBytes))

	return b.String()
}

// Handler adalah handler Fiber untuk endpoint GET /api/metrics yang mengembalikan
// metrik dalam format text exposition Prometheus (text/plain; version=0.0.4).
// @Summary      Metrik Prometheus
// @Description  Mengembalikan metrik aplikasi (request count, durasi histogram, DB pool, WebSocket aktif, ukuran uploads) dalam format text exposition Prometheus untuk di-scrape oleh Prometheus.
// @Tags         Monitoring
// @Produce      plain
// @Success      200 {string} string "Metrik Prometheus text format"
// @Router       /metrics [get]
func Handler(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	return c.SendString(GetCollector().Render())
}

// escapeLabel mengganti karakter yang tidak diizinkan dalam label Prometheus
// (\ " \n) agar nilai label selalu valid.
func escapeLabel(s string) string {
	r := strings.NewReplacer("\\", "\\\\", "\"", "\\\"", "\n", "\\n")
	return r.Replace(s)
}
