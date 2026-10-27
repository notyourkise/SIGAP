package config

import (
	"os"
	"strings"
	"sync"
	"time"
)

// DefaultAppTimezone adalah zona waktu operasional default SIGAP (WITA).
const DefaultAppTimezone = "Asia/Makassar"

var appLocationCache sync.Map

// AppLocation mengembalikan zona waktu operasional aplikasi dari APP_TIMEZONE.
// Nilai kosong memakai Asia/Makassar. Nama zona yang tidak valid atau tidak
// tersedia di image runtime akan jatuh ke fixed zone WITA (UTC+8) agar endpoint
// yang menghitung batas hari tetap berjalan.
func AppLocation() *time.Location {
	name := strings.TrimSpace(os.Getenv("APP_TIMEZONE"))
	if name == "" {
		name = DefaultAppTimezone
	}

	if cached, ok := appLocationCache.Load(name); ok {
		return cached.(*time.Location)
	}

	location, err := time.LoadLocation(name)
	if err != nil {
		location = time.FixedZone("WITA", 8*60*60)
	}
	appLocationCache.Store(name, location)
	return location
}
