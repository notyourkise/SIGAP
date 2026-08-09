package utils

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestEmailDisabledWhenNoSMTP memastikan email nonaktif bila SMTP_HOST kosong.
func TestEmailDisabledWhenNoSMTP(t *testing.T) {
	t.Setenv("SMTP_HOST", "")
	InitEmailConfig()
	assert.False(t, EmailEnabled())

	err := SendEmail("a@b.c", "sub", "body")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tidak dikonfigurasi")
}

// TestEmailEnabledWithSMTP memastikan email aktif bila SMTP_HOST terisi.
func TestEmailEnabledWithSMTP(t *testing.T) {
	t.Setenv("SMTP_HOST", "smtp.example.com")
	t.Setenv("SMTP_PORT", "587")
	t.Setenv("SMTP_USER", "user")
	t.Setenv("SMTP_PASSWORD", "pass")
	t.Setenv("SMTP_FROM", "SIGAP <noreply@sigap.id>")
	t.Setenv("SMTP_TLS", "true")
	InitEmailConfig()
	assert.True(t, EmailEnabled())
}

// TestEncodeHeader memastikan subject non-ASCII di-encode RFC 2047.
func TestEncodeHeader(t *testing.T) {
	// Subject ASCII dikembalikan apa adanya.
	assert.Equal(t, "Daily Report", encodeHeader("Daily Report"))
	// Subject dengan karakter non-ASCII (mis. em-dash) di-encode UTF-8 quoted-printable.
	encoded := encodeHeader("Ringkasan Harian – SIGAP")
	assert.True(t, strings.HasPrefix(encoded, "=?UTF-8?Q?"))
	assert.True(t, strings.HasSuffix(encoded, "?="))
}

// TestRenderEmailTemplate memastikan template dirender bila berkas tersedia.
func TestRenderEmailTemplate(t *testing.T) {
	// Template ada di backend/templates saat test dijalankan dari folder backend.
	body, err := RenderEmailTemplate("stock_low.html", map[string]interface{}{
		"Nama":           "Admin",
		"NamaSukuCadang": "Kabel UTP",
		"SKU":            "KBL-001",
		"Stok":           3,
		"Satuan":         "pcs",
		"MinimumOrder":   5,
	})
	if os.IsNotExist(err) {
		// Folder templates tidak tersedia saat test dijalankan dari path lain;
		// skip saja, bukan kegagalan.
		t.Skip("folder templates tidak ditemukan")
	}
	assert.NoError(t, err)
	assert.Contains(t, body, "Kabel UTP")
	assert.Contains(t, body, "KBL-001")
}

// TestRenderStreamDownTemplate memastikan template email CCTV offline dirender
// dengan data CCTV yang benar.
func TestRenderStreamDownTemplate(t *testing.T) {
	body, err := RenderEmailTemplate("stream_down.html", map[string]interface{}{
		"Nama":         "Admin IKN",
		"CCTVNama":     "Kamera Gerbang Utama",
		"CCTVCCTVKode": "CCTV-001",
		"Lokasi":       "Area Utama",
		"Jenis":        "Dome",
		"IPKamera":     "192.168.1.11",
		"Waktu":        "2026-08-09 02:00:00 WITA",
		"HealthScore":  80,
	})
	if os.IsNotExist(err) {
		t.Skip("folder templates tidak ditemukan")
	}
	assert.NoError(t, err)
	assert.Contains(t, body, "Kamera Gerbang Utama")
	assert.Contains(t, body, "CCTV-001")
	assert.Contains(t, body, "OFFLINE")
}
