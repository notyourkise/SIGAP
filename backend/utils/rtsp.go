package utils

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var rtspCredentialURLPattern = regexp.MustCompile(`(?i)rtsps?://[^\s"'<>\]\)]+`)

// StreamType menunjukkan jenis aliran (stream) yang diminta dari NVR.
// NVR Hikvision membedakan channel ID berdasarkan kombinasi channel_port
// dan stream type (1 = MainStream, 2 = SubStream).
type StreamType int

const (
	// MainStream adalah aliran utama (resolusi tinggi) — channel id berakhiran 1.
	MainStream StreamType = 1
	// SubStream adalah aliran sekunder (resolusi rendah) — channel id berakhiran 2.
	SubStream StreamType = 2
)

// getenvOr membaca environment variable dan mengembalikan nilai default bila
// variabel kosong atau tidak di-set.
func getenvOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// BuildRTSPURL menyusun URI RTSP untuk streaming kamera dari NVR.
//
// Format URI:
//
//	rtsp://{user}:{password}@{host}:{port}/Streaming/channels/{channelID}
//
// dengan channelID = channelPort*100 + int(streamType). Misalnya:
//   - channel_port 19, SubStream  → 19*100 + 2 = 1902
//   - channel_port 1,  MainStream → 1*100 + 1  = 101
//
// Host, port, dan user dibaca dari environment RTSP_NVR_HOST, RTSP_NVR_PORT,
// dan RTSP_NVR_USER (fallback kosong — wajib dikonfigurasi di production). Password diterima dalam
// bentuk terdekripsi (dekripsi dilakukan pada Task 9.3). Baik password maupun
// user di-escape dengan url.QueryEscape agar aman saat berisi karakter khusus.
func BuildRTSPURL(channelPort int, decryptedPassword string, streamType StreamType) string {
	host := getenvOr("RTSP_NVR_HOST", "")
	port := getenvOr("RTSP_NVR_PORT", "554")
	user := getenvOr("RTSP_NVR_USER", "admin")

	channelID := channelPort*100 + int(streamType)
	return fmt.Sprintf("rtsp://%s:%s@%s:%s/Streaming/channels/%d",
		url.QueryEscape(user),
		url.QueryEscape(decryptedPassword),
		host,
		port,
		channelID,
	)
}

// rtspURLEncryptionPrefix menandai nilai rtsp_url yang sudah dienkripsi.
// Nilai lama tanpa prefix tetap dibaca sebagai plaintext agar data existing
// tidak rusak, lalu akan dienkripsi saat migration/update berikutnya.
const rtspURLEncryptionPrefix = "enc:v1:"

// EncryptRTSPURL mengenkripsi URL RTSP penuh (termasuk kredensial userinfo)
// sebelum disimpan ke database. String kosong dikembalikan apa adanya.
func EncryptRTSPURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	encrypted, err := EncryptAES256(raw)
	if err != nil {
		return "", err
	}
	return rtspURLEncryptionPrefix + encrypted, nil
}

// DecryptRTSPURL mendekripsi nilai rtsp_url dari database. Nilai legacy yang
// belum berprefix dianggap plaintext dan dikembalikan tanpa perubahan.
func DecryptRTSPURL(stored string) (string, error) {
	stored = strings.TrimSpace(stored)
	if stored == "" {
		return "", nil
	}
	if !strings.HasPrefix(stored, rtspURLEncryptionPrefix) {
		return stored, nil
	}
	return DecryptAES256(strings.TrimPrefix(stored, rtspURLEncryptionPrefix))
}

// IsEncryptedRTSPURL memeriksa apakah nilai rtsp_url sudah memakai format
// enkripsi saat ini. Dipakai migration data legacy agar idempoten.
func IsEncryptedRTSPURL(stored string) bool {
	return strings.HasPrefix(strings.TrimSpace(stored), rtspURLEncryptionPrefix)
}

// MaskRTSPURL menyamarkan password pada userinfo URL RTSP agar aman
// ditampilkan di UI/log. Bila URL tidak dapat di-parse, fallback memask
// bagian userinfo secara manual agar password tidak ikut bocor.
func MaskRTSPURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if !strings.Contains(raw, "://") {
		return redactUserInfoFallback(raw)
	}
	u, err := url.Parse(raw)
	if err == nil {
		return u.Redacted()
	}
	return redactUserInfoFallback(raw)
}

// redactUserInfoFallback mengganti password pada userinfo tanpa bergantung
// pada parser URL. Dipakai hanya saat url.Parse gagal.
func redactUserInfoFallback(raw string) string {
	at := strings.LastIndex(raw, "@")
	if at < 0 {
		return raw
	}
	start := 0
	if schemeEnd := strings.Index(raw, "://"); schemeEnd >= 0 {
		start = schemeEnd + 3
	}
	if at <= start {
		return raw
	}
	userinfo := raw[start:at]
	if colon := strings.Index(userinfo, ":"); colon >= 0 {
		userinfo = userinfo[:colon] + ":xxxxx"
	}
	return raw[:start] + userinfo + raw[at:]
}

// RedactRTSPCredentialsInText menyamarkan seluruh URI RTSP/RTSPS berkredensial
// yang mungkin terbawa di pesan error, respons MediaMTX, maupun output log
// agar password RTSP tidak pernah bocor ke file log atau respons API.
func RedactRTSPCredentialsInText(text string) string {
	if text == "" || !strings.Contains(strings.ToLower(text), "rtsp") {
		return text
	}
	return rtspCredentialURLPattern.ReplaceAllStringFunc(text, func(match string) string {
		return MaskRTSPURL(match)
	})
}

// BuildStreamPath membangun path stream yang dipakai untuk mengidentifikasi
// kamera tertentu pada MediaMTX, dengan format "cam-<cctvID>".
func BuildStreamPath(cctvID string) string {
	return "cam-" + cctvID
}

// ResolveRTSPURL menentukan URL RTSP final untuk sebuah CCTV.
//
// Bila `rtspURL` (field rtsp_url pada master CCTV) diisi — misalnya
// rtsp://USER:PASSWORD@NVR_HOST:554/Streaming/channels/1901 — URL tersebut
// dipakai sebagai sumber stream MediaMTX. Bila URL memakai pola Hikvision
// /Streaming/channels/<channelID>, dua digit terakhir channelID disesuaikan
// dengan streamType (01 = main, 02 = sub) agar tombol kualitas SD/HD benar-
// benar memilih stream yang berbeda. URL dengan pola lain dikembalikan apa
// adanya supaya konfigurasi vendor non-Hikvision tidak rusak.
//
// Jika kosong, URL disusun otomatis dari channel port + kredensial NVR global
// melalui BuildRTSPURL (perilaku lama, hanya cocok untuk satu NVR bersama).
func ResolveRTSPURL(rtspURL string, channelPort int, decryptedPassword string, streamType StreamType) string {
	if rtspURL != "" {
		return applyStreamTypeToRTSPURL(rtspURL, streamType)
	}
	return BuildRTSPURL(channelPort, decryptedPassword, streamType)
}

// applyStreamTypeToRTSPURL mengganti suffix main/sub pada channel Hikvision.
// Mengembalikan URL asli bila pola tidak dikenali atau suffix sudah sesuai,
// sehingga encoding userinfo/query tetap dipertahankan.
func applyStreamTypeToRTSPURL(raw string, streamType StreamType) string {
	if streamType != MainStream && streamType != SubStream {
		return raw
	}

	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return raw
	}

	segments := strings.Split(u.Path, "/")
	idx := len(segments) - 1
	for idx >= 0 && segments[idx] == "" {
		idx--
	}
	if idx < 1 || !strings.EqualFold(segments[idx-1], "channels") {
		return raw
	}

	channelID := segments[idx]
	if len(channelID) < 3 {
		return raw
	}
	prefix, suffix := channelID[:len(channelID)-2], channelID[len(channelID)-2:]
	if suffix != "01" && suffix != "02" {
		return raw
	}
	if _, err := strconv.Atoi(prefix); err != nil {
		return raw
	}

	desiredSuffix := "01"
	if streamType == SubStream {
		desiredSuffix = "02"
	}
	if suffix == desiredSuffix {
		return raw
	}

	segments[idx] = prefix + desiredSuffix
	u.Path = strings.Join(segments, "/")
	u.RawPath = ""
	return u.String()
}

// GetRTSPHost membaca host NVR dari environment RTSP_NVR_HOST (fallback kosong — wajib dikonfigurasi).
func GetRTSPHost() string {
	return getenvOr("RTSP_NVR_HOST", "")
}

// GetRTSPPort membaca port NVR dari environment RTSP_NVR_PORT dengan fallback
// default 554. Mengembalikan 554 bila nilai tidak dapat di-parse sebagai integer.
func GetRTSPPort() int {
	port := getenvOr("RTSP_NVR_PORT", "554")
	if p, err := strconv.Atoi(port); err == nil {
		return p
	}
	return 554
}
