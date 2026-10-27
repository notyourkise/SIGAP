package handlers

import (
	"time"

	"sigap-backend/config"
	"sigap-backend/utils"

	"gorm.io/gorm"
)

const (
	// cctvStreamLogCleanupBatchSize membatasi jumlah baris yang dihapus per
	// pernyataan agar lock tabel tidak panjang.
	cctvStreamLogCleanupBatchSize = 1000
	// cctvStreamLogCleanupInterval menjalankan cleanup sekali sehari.
	cctvStreamLogCleanupInterval = 24 * time.Hour
	// cctvStreamLogCleanupBatchPause memberi jeda antar-batch agar worker tidak
	// membanjiri database saat backlog besar.
	cctvStreamLogCleanupBatchPause = 100 * time.Millisecond
)

const deleteCCTVStreamLogsBatchQuery = `DELETE FROM cctv_stream_logs WHERE id IN (SELECT id FROM cctv_stream_logs WHERE created_at < $1 ORDER BY created_at ASC LIMIT $2)`

// cctvStreamLogRetentionCutoff menghitung batas waktu retensi. Nilai
// retentionDays <= 0 memakai default agar cleanup tidak pernah menghapus semua
// log karena konfigurasi yang salah.
func cctvStreamLogRetentionCutoff(now time.Time, retentionDays int) time.Time {
	if retentionDays <= 0 {
		retentionDays = config.DefaultCCTVStreamLogRetentionDays
	}
	return now.AddDate(0, 0, -retentionDays)
}

// CleanupCCTVStreamLogs menghapus log event CCTV yang lebih tua dari
// retentionDays secara bertahap (maksimal 1.000 baris per batch). Fungsi
// mengembalikan total baris yang terhapus dan bersifat idempotent: pemanggilan
// berikutnya pada data yang sama tidak menghapus apa pun.
func CleanupCCTVStreamLogs(db *gorm.DB, retentionDays int) (int64, error) {
	if db == nil {
		return 0, nil
	}

	cutoff := cctvStreamLogRetentionCutoff(time.Now(), retentionDays)
	var totalDeleted int64

	for {
		result := db.Exec(deleteCCTVStreamLogsBatchQuery, cutoff, cctvStreamLogCleanupBatchSize)
		if result.Error != nil {
			return totalDeleted, result.Error
		}

		totalDeleted += result.RowsAffected
		if result.RowsAffected < cctvStreamLogCleanupBatchSize {
			break
		}

		time.Sleep(cctvStreamLogCleanupBatchPause)
	}

	return totalDeleted, nil
}

// StartCCTVStreamLogCleanup menjalankan cleanup saat startup lalu berulang
// setiap 24 jam. Pemanggil harus menjalankannya sebagai goroutine karena fungsi
// ini memblokir pada loop ticker.
func StartCCTVStreamLogCleanup(db *gorm.DB) {
	if db == nil {
		return
	}

	retentionDays := config.CCTVStreamLogRetentionDays()
	run := func() {
		deleted, err := CleanupCCTVStreamLogs(db, retentionDays)
		if err != nil {
			utils.Log.Errorf("❌ Gagal cleanup log stream CCTV: %v", err)
			return
		}
		if deleted > 0 {
			utils.Log.Infof("🧹 Cleanup log stream CCTV: %d baris lebih tua dari %d hari dihapus", deleted, retentionDays)
		} else {
			utils.Log.Infof("🧹 Cleanup log stream CCTV: tidak ada log lebih tua dari %d hari", retentionDays)
		}
	}

	run()
	ticker := time.NewTicker(cctvStreamLogCleanupInterval)
	defer ticker.Stop()
	for range ticker.C {
		run()
	}
}
