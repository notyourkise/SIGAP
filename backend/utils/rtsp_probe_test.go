package utils

import (
	"net"
	"testing"
	"time"

	"sigap-backend/models"

	"github.com/stretchr/testify/assert"
)

func TestIsMetroE_True(t *testing.T) {
	cctv := models.CCTV{IPCamera: "10.20.105.62"}
	assert.True(t, IsMetroE(cctv))
}

func TestIsMetroE_False(t *testing.T) {
	cctv := models.CCTV{IPCamera: "10.99.99.99"}
	assert.False(t, IsMetroE(cctv))
}

func TestIsMetroE_Empty(t *testing.T) {
	cctv := models.CCTV{IPCamera: ""}
	assert.False(t, IsMetroE(cctv))
}

func TestProbeICMP_Success(t *testing.T) {
	// Buka listener TCP lokal pada 127.0.0.1:0 — ProbeICMP mencoba port 554
	// lalu fallback 80, sehingga listener port 80 harus tersedia (atau kita
	// dial ke port lain bila ProbeICMP dapat menerima port).
	ln, err := net.Listen("tcp", "127.0.0.1:80")
	if err != nil {
		t.Skipf("port 80 tidak tersedia di environment pengujian: %v", err)
	}
	defer func() { _ = ln.Close() }()

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()

	latencyMs, ok := ProbeICMP("127.0.0.1", 3*time.Second)
	assert.True(t, ok)
	assert.GreaterOrEqual(t, latencyMs, int64(0))
}

func TestProbeICMP_Fail(t *testing.T) {
	// 192.0.2.1 adalah alamat TEST-NET-1 (RFC 5737) yang dijamin tidak routable
	// dan tidak ada host — dial port 554 dan fallback port 80 harus gagal
	// (connection refused/timeout) → ok=false.
	latencyMs, ok := ProbeICMP("192.0.2.1", time.Second)
	assert.False(t, ok)
	assert.Equal(t, int64(0), latencyMs)
}
