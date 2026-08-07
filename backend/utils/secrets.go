package utils

import (
	"os"
	"strings"
)

// GetSecret membaca nilai secret dari file Docker Secret (bila tersedia),
// atau kembali (fallback) ke environment variable bila file tidak ada.
//
// Docker menyuntikkan secret sebagai file di dalam container pada path
// /run/secrets/<nama>. Dengan memprioritaskan file secret ini, nilai rahasia
// tidak perlu lagi dirender ke dalam env var (lebih aman terhadap pemeriksaan
// `docker inspect` / lingkungan proses). Bila file tidak ditemukan (mis. saat
// development di luar Docker), fungsi ini membaca env var sebagai fallback.
//
// Parameter:
//   - envName:       nama environment variable yang dipakai sebagai fallback.
//   - secretFilePath: path file secret di dalam container (mis. /run/secrets/...).
//
// Mengembalikan string secret dengan whitespace (newline/spasi) di ujung yang
// di-trim. Mengembalikan string kosong bila keduanya tidak tersedia.
func GetSecret(envName, secretFilePath string) string {
	if data, err := os.ReadFile(secretFilePath); err == nil {
		return strings.TrimSpace(string(data))
	}
	return os.Getenv(envName)
}

// RequireSecret adalah versi fail-fast dari GetSecret.
// Ia mengembalikan secret yang sudah terselesaikan (dari file atau env), dan
// memanggil Log.Fatalf bila hasilnya kosong. Tidak ada fallback hardcoded:
// secret wajib disediakan (melalui Docker secret atau env) sebelum server berjalan.
func RequireSecret(envName, secretFilePath string) string {
	value := GetSecret(envName, secretFilePath)
	if value == "" {
		Log.Fatalf("❌ Secret wajib tidak terisi (env '%s' atau file '%s')", envName, secretFilePath)
	}
	return value
}
