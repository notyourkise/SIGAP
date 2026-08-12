package utils

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
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
