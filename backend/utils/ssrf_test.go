package utils

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestValidateExternalURL_RejectsInternal memastikan URL yang menunjuk ke
// alamat internal/privat atau skema tidak didukung selalu ditolak (anti-SSRF).
func TestValidateExternalURL_RejectsInternal(t *testing.T) {
	cases := []string{
		"http://127.0.0.1/hook",
		"http://localhost/hook",
		"http://10.0.0.5/hook",
		"http://192.168.1.10/hook",
		"http://172.16.0.1/hook",
		"http://169.254.169.254/latest/meta-data/",
		"http://0.0.0.0/hook",
		"http://[::1]/hook",
		"ftp://example.com/hook",
		"",
	}
	for _, raw := range cases {
		assert.Error(t, ValidateExternalURL(raw), "harus ditolak: %q", raw)
	}
}

// TestValidateExternalURL_AllowsPublicIP memastikan IP publik literal diizinkan
// (tanpa ketergantungan DNS).
func TestValidateExternalURL_AllowsPublicIP(t *testing.T) {
	assert.NoError(t, ValidateExternalURL("https://93.184.216.34/hook"))
}

// TestNewSafeHTTPClient_BlocksInternal memastikan client menolak koneksi ke
// alamat loopback pada level dial.
func TestNewSafeHTTPClient_BlocksInternal(t *testing.T) {
	client := NewSafeHTTPClient(2 * time.Second)
	req, err := http.NewRequest(http.MethodPost, "http://127.0.0.1:9/hook", nil)
	assert.NoError(t, err)

	_, err = client.Do(req)
	assert.Error(t, err)
}
