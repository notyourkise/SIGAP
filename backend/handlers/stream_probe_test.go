package handlers

import (
	"bufio"
	"net"
	"strconv"
	"strings"
	"testing"

	"sigap-backend/utils"

	"github.com/stretchr/testify/assert"
)

// startTestRTSPListener membuka listener TCP pada 127.0.0.1:0 dan mengembalikan
// alamat serta fungsi cleanup. Opsi `respondRTSP` mengendalikan apakah server
// tiruan membalas baris RTSP 200 OK atau hanya membuka koneksi lalu menutupnya.
func startTestRTSPListener(t *testing.T, respondRTSP bool) (addr string, closeFn func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return // listener ditutup
			}
			go func(c net.Conn) {
				defer func() { _ = c.Close() }()
				if respondRTSP {
					// Baca request dari klien (OPTIONS) lalu balas 200 OK.
					scanner := bufio.NewScanner(c)
					for scanner.Scan() {
						line := scanner.Text()
						if line == "" {
							break
						}
					}
					_, _ = c.Write([]byte("RTSP/1.0 200 OK\r\nCSeq: 1\r\n\r\n"))
				}
			}(conn)
		}
	}()

	closeFn = func() { _ = ln.Close() }
	return ln.Addr().String(), closeFn
}

func TestProbeRTSPHost_Success(t *testing.T) {
	addr, closeFn := startTestRTSPListener(t, false)
	defer closeFn()

	host, portStr, ok := strings.Cut(addr, ":")
	assert.True(t, ok)
	port, err := strconv.Atoi(portStr)
	assert.NoError(t, err)

	assert.True(t, utils.ProbeRTSPHost(host, port))
}

func TestProbeRTSPHost_Unreachable(t *testing.T) {
	// Port 1 (tidak terpakai) pada localhost — koneksi harus gagal.
	assert.False(t, utils.ProbeRTSPHost("127.0.0.1", 1))
}

func TestProbeRTSPChannel_Success(t *testing.T) {
	addr, closeFn := startTestRTSPListener(t, true)
	defer closeFn()

	host, portStr, ok := strings.Cut(addr, ":")
	assert.True(t, ok)
	port, err := strconv.Atoi(portStr)
	assert.NoError(t, err)

	assert.True(t, utils.ProbeRTSPChannel(host, port, 1902))
}

func TestProbeRTSPChannel_Unreachable(t *testing.T) {
	assert.False(t, utils.ProbeRTSPChannel("127.0.0.1", 1, 1902))
}
