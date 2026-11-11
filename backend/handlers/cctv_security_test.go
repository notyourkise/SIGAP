package handlers

import (
	"encoding/json"
	"math"
	"strings"
	"testing"

	"sigap-backend/models"
	"sigap-backend/utils"
)

func TestValidateRTSPURL(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{name: "kosong diperbolehkan untuk legacy", raw: "", wantErr: false},
		{name: "rtsp valid", raw: "rtsp://admin:secret@10.0.0.5:554/Streaming/channels/1902", wantErr: false},
		{name: "rtsps valid", raw: "rtsps://admin:secret@nvr.local:554/path", wantErr: false},
		{name: "skema http ditolak", raw: "http://10.0.0.5/stream", wantErr: true},
		{name: "host kosong ditolak", raw: "rtsp://", wantErr: true},
		{name: "bukan URL ditolak", raw: "10.0.0.5:554", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := validateRTSPURL(tc.raw)
			if tc.wantErr && got == "" {
				t.Fatalf("validateRTSPURL(%q) = \"\", want error", tc.raw)
			}
			if !tc.wantErr && got != "" {
				t.Fatalf("validateRTSPURL(%q) = %q, want valid", tc.raw, got)
			}
		})
	}
}

func TestSanitizeCCTVForResponse_EncryptedURLIsMasked(t *testing.T) {
	t.Setenv("AES_SECRET_KEY", "sigap_aes_test_secret_2026")

	raw := "rtsp://admin:secret123@10.0.0.5:554/Streaming/channels/1902"
	encrypted, err := utils.EncryptRTSPURL(raw)
	if err != nil {
		t.Fatalf("EncryptRTSPURL() error: %v", err)
	}

	cctv := models.CCTV{
		RTSPURL:      encrypted,
		RTSPPassword: "encrypted-password-should-not-leak",
	}
	sanitizeCCTVForResponse(&cctv)

	if cctv.RTSPURL != "" {
		t.Fatalf("RTSPURL masih terisi setelah sanitize: %q", cctv.RTSPURL)
	}
	if cctv.RTSPPassword != "" {
		t.Fatalf("RTSPPassword masih terisi setelah sanitize: %q", cctv.RTSPPassword)
	}
	if !cctv.RTSPURLConfigured {
		t.Fatal("RTSPURLConfigured = false, want true")
	}
	if strings.Contains(cctv.RTSPURLMasked, "secret123") {
		t.Fatalf("RTSPURLMasked membocorkan password: %q", cctv.RTSPURLMasked)
	}
	if !strings.Contains(cctv.RTSPURLMasked, "admin:xxxxx@") {
		t.Fatalf("RTSPURLMasked = %q, want userinfo tersamarkan", cctv.RTSPURLMasked)
	}

	payload, err := json.Marshal(cctv)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}
	body := string(payload)
	if strings.Contains(body, "secret123") {
		t.Fatalf("response JSON membocorkan password: %s", body)
	}
	if strings.Contains(body, `"rtsp_url"`) {
		t.Fatalf("response JSON masih memuat rtsp_url mentah: %s", body)
	}
	if !strings.Contains(body, `"rtsp_url_masked"`) || !strings.Contains(body, `"rtsp_url_configured":true`) {
		t.Fatalf("response JSON tidak memuat metadata RTSP yang aman: %s", body)
	}
}

func TestSanitizeCCTVForResponse_LegacyPlaintextIsMasked(t *testing.T) {
	raw := "rtsp://operator:legacy-secret@nvr.local:554/Streaming/channels/101"
	cctv := models.CCTV{RTSPURL: raw}

	sanitizeCCTVForResponse(&cctv)

	if cctv.RTSPURL != "" {
		t.Fatalf("RTSPURL legacy masih terisi setelah sanitize: %q", cctv.RTSPURL)
	}
	if strings.Contains(cctv.RTSPURLMasked, "legacy-secret") {
		t.Fatalf("RTSPURLMasked membocorkan password legacy: %q", cctv.RTSPURLMasked)
	}
	if !cctv.RTSPURLConfigured {
		t.Fatal("RTSPURLConfigured = false, want true")
	}
}

func TestSanitizeCCTVForResponse_WithoutURL(t *testing.T) {
	cctv := models.CCTV{}
	sanitizeCCTVForResponse(&cctv)

	if cctv.RTSPURL != "" || cctv.RTSPURLMasked != "" || cctv.RTSPURLConfigured {
		t.Fatalf("CCTV tanpa URL tidak boleh punya metadata RTSP: %+v", cctv)
	}
}

func TestAudit_CCTVCoordinatesRejectNaNAndInf(t *testing.T) {
	nanLat := math.NaN()
	infPos := math.Inf(1)
	infNeg := math.Inf(-1)
	validLat := -0.9721
	validLng := 116.7092

	if msg := validateCoordinateRange(&nanLat, &validLng); msg == "" {
		t.Fatal("validateCoordinateRange menerima latitude NaN")
	}
	if msg := validateCoordinateRange(&validLat, &infPos); msg == "" {
		t.Fatal("validateCoordinateRange menerima longitude +Inf")
	}
	if msg := validateCoordinateRange(&infNeg, &validLng); msg == "" {
		t.Fatal("validateCoordinateRange menerima latitude -Inf")
	}
	if msg := validateCoordinateRange(&validLat, &validLng); msg != "" {
		t.Fatalf("validateCoordinateRange menolak koordinat valid: %s", msg)
	}

	for _, bad := range []string{"NaN", "nan", "Inf", "+Inf", "-Inf", "Infinity"} {
		if got := parseCoordinateCell(bad, 2, "Latitude"); got != nil {
			t.Fatalf("parseCoordinateCell(%q) = %v, want nil", bad, *got)
		}
	}
	got := parseCoordinateCell(" -0.9723 ", 2, "Latitude")
	if got == nil || *got != -0.9723 {
		t.Fatalf("parseCoordinateCell gagal membaca koordinat valid: got=%v", got)
	}
}
