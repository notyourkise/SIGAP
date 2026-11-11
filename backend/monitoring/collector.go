package monitoring

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"sigap-backend/config"
	"sigap-backend/utils"
)

const streamVerifyFreshnessWindow = 30 * time.Minute

type cctvStateProjection struct {
	StreamLive          bool
	StreamState         string
	StreamCheckedAt     *time.Time
	StreamVerifiedState string
	StreamVerifiedAt    *time.Time
}

// updateDBPool membaca statistik connection pool dari sql.DB dan mencatatnya
// sebagai gauge. Bila gagal, gauge dibiarkan bernilai sebelumnya (tidak fatal).
func (c *Collector) updateDBPool() {
	if config.DB == nil {
		return
	}
	sqlDB, err := config.DB.DB()
	if err != nil {
		utils.Log.Warnf("⚠️ Metrics: gagal mengambil sql.DB untuk pool stats: %v", err)
		return
	}
	stats := sqlDB.Stats()
	c.SetDBPool(
		float64(stats.OpenConnections),
		float64(stats.Idle),
		float64(stats.MaxOpenConnections),
	)
}

// updateCCTVCameraStates membaca status efektif kamera aktif dari database untuk
// memperbarui gauge online, idle, offline, dan unknown pada Prometheus.
func (c *Collector) updateCCTVCameraStates() {
	if config.DB == nil {
		return
	}

	var rows []cctvStateProjection
	err := config.DB.Table("cctvs").
		Select("stream_live, stream_state, stream_checked_at, stream_verified_state, stream_verified_at").
		Where("deleted_at IS NULL AND (stream_enabled = ? OR monitoring_enabled = ?) AND LOWER(COALESCE(status, '')) <> 'nonaktif'", true, true).
		Find(&rows).Error
	if err != nil {
		utils.Log.Warnf("⚠️ Metrics: gagal memuat status kamera CCTV: %v", err)
		return
	}

	now := time.Now().UTC()
	var online, idle, offline, unknown float64
	for _, row := range rows {
		switch effectiveCCTVCameraState(row, now) {
		case "live":
			online++
		case "idle":
			idle++
		case "offline":
			offline++
		default:
			unknown++
		}
	}
	c.SetCCTVCameraStates(online, idle, offline, unknown)
}

func effectiveCCTVCameraState(row cctvStateProjection, now time.Time) string {
	if row.StreamVerifiedAt != nil {
		age := now.Sub(*row.StreamVerifiedAt)
		if age >= 0 && age <= streamVerifyFreshnessWindow &&
			(row.StreamVerifiedState == "live" || row.StreamVerifiedState == "offline") {
			return row.StreamVerifiedState
		}
	}
	state := strings.TrimSpace(row.StreamState)
	if state != "" {
		return state
	}
	if row.StreamLive {
		return "live"
	}
	if row.StreamCheckedAt != nil {
		return "offline"
	}
	return "unknown"
}

// updateWSActive membaca jumlah koneksi WebSocket aktif dan mencatatnya sebagai gauge.
func (c *Collector) updateWSActive() {
	c.SetWSActive(float64(utils.Hub.ActiveConnections()))
}

// updateUploadStorage menghitung total ukuran folder uploads dan mencatatnya
// sebagai gauge (bytes). Folder yang tidak ada dianggap 0 bytes.
func (c *Collector) updateUploadStorage(dir string) {
	size, err := dirSizeBytes(dir)
	if err != nil {
		utils.Log.Warnf("⚠️ Metrics: gagal menghitung ukuran uploads '%s': %v", dir, err)
		return
	}
	c.SetUploadStorageBytes(float64(size))
}

// dirSizeBytes menghitung ukuran total folder (rekursif) dalam bytes.
func dirSizeBytes(path string) (int64, error) {
	var size int64
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	if !info.IsDir() {
		return info.Size(), nil
	}
	err = filepath.Walk(path, func(_ string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !fi.IsDir() {
			size += fi.Size()
		}
		return nil
	})
	return size, err
}

// StartCollector menjalankan goroutine yang memperbarui gauge metrik
// (DB pool, WS aktif, ukuran uploads) secara berkala. Pertama dijalankan
// segera, lalu berulang tiap interval. Dipanggil dari main.go saat startup.
func StartCollector(uploadsDir string, interval time.Duration) {
	collector := GetCollector()

	refresh := func() {
		collector.updateDBPool()
		collector.updateWSActive()
		collector.updateUploadStorage(uploadsDir)
		collector.updateCCTVCameraStates()
	}

	// Perbarui langsung sekali saat startup agar metrik gauge langsung terisi.
	refresh()

	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			refresh()
		}
	}()
	utils.Log.Infof("📊 Metrics collector diaktifkan (interval: %v, uploads: %s)", interval, uploadsDir)
}
