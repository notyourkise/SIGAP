package utils

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
)

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
// dan RTSP_NVR_USER (dengan fallback nilai default). Password diterima dalam
// bentuk terdekripsi (dekripsi dilakukan pada Task 9.3). Baik password maupun
// user di-escape dengan url.QueryEscape agar aman saat berisi karakter khusus.
func BuildRTSPURL(channelPort int, decryptedPassword string, streamType StreamType) string {
	host := getenvOr("RTSP_NVR_HOST", "36.91.27.35")
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

// BuildStreamPath membangun path stream yang dipakai untuk mengidentifikasi
// kamera tertentu pada MediaMTX, dengan format "cam-<cctvID>".
func BuildStreamPath(cctvID string) string {
	return "cam-" + cctvID
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
