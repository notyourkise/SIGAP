package config

import (
	"time"

	"sigap-backend/utils"
)

// DefaultAppTimezone adalah zona waktu operasional default SIGAP (WITA).
const DefaultAppTimezone = utils.DefaultAppTimezone

// AppLocation mengembalikan zona waktu operasional aplikasi dari APP_TIMEZONE.
// Nilai kosong memakai Asia/Makassar. Nama zona yang tidak valid atau tidak
// tersedia di image runtime akan jatuh ke fixed zone WITA (UTC+8) agar endpoint
// yang menghitung batas hari tetap berjalan.
func AppLocation() *time.Location {
	return utils.AppLocation()
}
