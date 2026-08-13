package utils

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"sigap-backend/models"
)

// ProbeRTSPHost mengecek apakah host NVR dapat dijangkau via TCP pada port
// tertentu. Mengembalikan true bila koneksi TCP berhasil dibuka dalam batas
// waktu 3 detik. Dipakai sebagai pemeriksaan awal (gate) sebelum probe channel.
func ProbeRTSPHost(host string, port int) bool {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, strconv.Itoa(port)), 3*time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// ProbeRTSPChannel mengecek apakah channel kamera individual aktif dengan
// mengirim request RTSP OPTIONS dan memeriksa apakah baris pertama response
// mengandung kode 200 (RTSP/1.0 200 OK). Deadline keseluruhan 5 detik.
func ProbeRTSPChannel(host string, port int, channelID int) bool {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, strconv.Itoa(port)), 3*time.Second)
	if err != nil {
		return false
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))

	path := fmt.Sprintf("/Streaming/channels/%d", channelID)
	req := fmt.Sprintf("OPTIONS rtsp://%s:%d%s RTSP/1.0\r\nCSeq: 1\r\n\r\n", host, port, path)
	if _, err := conn.Write([]byte(req)); err != nil {
		return false
	}
	scanner := bufio.NewScanner(conn)
	if scanner.Scan() {
		return strings.Contains(scanner.Text(), "200")
	}
	return false
}

// IsMetroE mengecek apakah kamera terhubung via jaringan Metro-E OIKN.
// Metro-E diidentifikasi dari prefix IP kamera 10.20.105 (jaringan dedicated).
// Referensi: docs/AGENTS.md § 11.2
func IsMetroE(cctv models.CCTV) bool {
	return strings.HasPrefix(cctv.IPCamera, "10.20.105.")
}

// ProbeICMP mengirim probe konektivitas ke IP target dan mengukur latensi.
// Digunakan HANYA untuk kamera Metro-E — TIDAK untuk kamera GSM (biaya data).
// Implementasi memakai TCP dial ke port 554 (RTSP) dengan fallback port 80 —
// ICMP raw socket butuh privilege NET_RAW yang tidak selalu tersedia.
// Mengembalikan latensi ms dan status ok.
func ProbeICMP(ip string, timeout time.Duration) (latencyMs int64, ok bool) {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:554", ip), timeout)
	if err != nil {
		conn, err = net.DialTimeout("tcp", fmt.Sprintf("%s:80", ip), timeout)
		if err != nil {
			return 0, false
		}
	}
	_ = conn.Close()
	return time.Since(start).Milliseconds(), true
}
