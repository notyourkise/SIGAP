package config

import (
	"os"
	"strconv"
	"strings"
)

// DefaultCCTVStreamLogRetentionDays adalah masa retensi default log event CCTV.
// Nilai dapat dioverride lewat CCTV_STREAM_LOG_RETENTION_DAYS.
const DefaultCCTVStreamLogRetentionDays = 90

// DefaultCCTVLogStandbyEvents menentukan apakah event standby dicatat secara
// default. Standby hanya menunjukkan kamera idle menunggu penonton, sehingga
// secara default tidak dicatat agar tidak membanjiri log.
const DefaultCCTVLogStandbyEvents = false

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

// CCTVLogStandbyEvents mengembalikan true bila event standby harus dicatat.
// Nilai kosong atau tidak valid memakai default false.
func CCTVLogStandbyEvents() bool {
	raw := strings.TrimSpace(os.Getenv("CCTV_LOG_STANDBY_EVENTS"))
	if raw == "" {
		return DefaultCCTVLogStandbyEvents
	}

	enabled, err := strconv.ParseBool(raw)
	if err != nil {
		return DefaultCCTVLogStandbyEvents
	}
	return enabled
}
