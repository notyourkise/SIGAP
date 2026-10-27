package config

import (
	"os"
	"strconv"
	"strings"
)

// DefaultCCTVStreamLogRetentionDays adalah masa retensi default log event CCTV.
// Nilai dapat dioverride lewat CCTV_STREAM_LOG_RETENTION_DAYS.
const DefaultCCTVStreamLogRetentionDays = 90

// CCTVStreamLogRetentionDays mengembalikan masa retensi log event CCTV dalam
// hari. Nilai kosong, bukan angka, atau <= 0 memakai default 90 hari agar tabel
// log tidak tumbuh tanpa batas.
func CCTVStreamLogRetentionDays() int {
	raw := strings.TrimSpace(os.Getenv("CCTV_STREAM_LOG_RETENTION_DAYS"))
	if raw == "" {
		return DefaultCCTVStreamLogRetentionDays
	}

	days, err := strconv.Atoi(raw)
	if err != nil || days <= 0 {
		return DefaultCCTVStreamLogRetentionDays
	}
	return days
}
