package utils

import (
	"strings"
	"testing"
)

func TestRTSPURLEncryptDecrypt_RoundTrip(t *testing.T) {
	t.Setenv("AES_SECRET_KEY", "sigap_aes_test_secret_2026")

	raw := "rtsp://admin:secret123@10.0.0.5:554/Streaming/channels/1902"
	encrypted, err := EncryptRTSPURL(raw)
	if err != nil {
		t.Fatalf("EncryptRTSPURL() error: %v", err)
	}
	if !IsEncryptedRTSPURL(encrypted) {
		t.Fatalf("EncryptRTSPURL() = %q, want prefix %q", encrypted, rtspURLEncryptionPrefix)
	}
	if encrypted == raw || strings.Contains(encrypted, "secret123") {
		t.Fatalf("EncryptRTSPURL() masih memuat plaintext: %q", encrypted)
	}

	decrypted, err := DecryptRTSPURL(encrypted)
	if err != nil {
		t.Fatalf("DecryptRTSPURL() error: %v", err)
	}
	if decrypted != raw {
		t.Fatalf("DecryptRTSPURL() = %q, want %q", decrypted, raw)
	}
}

func TestDecryptRTSPURL_LegacyPlaintext(t *testing.T) {
	raw := "rtsp://admin:secret123@10.0.0.5:554/Streaming/channels/1902"
	got, err := DecryptRTSPURL(raw)
	if err != nil {
		t.Fatalf("DecryptRTSPURL() error: %v", err)
	}
	if got != raw {
		t.Fatalf("DecryptRTSPURL(legacy) = %q, want %q", got, raw)
	}
}

func TestMaskRTSPURL(t *testing.T) {
	got := MaskRTSPURL("rtsp://admin:secret123@10.0.0.5:554/Streaming/channels/1902")
	if strings.Contains(got, "secret123") {
		t.Fatalf("MaskRTSPURL() membocorkan password: %q", got)
	}
	if !strings.Contains(got, "admin:xxxxx@") {
		t.Fatalf("MaskRTSPURL() = %q, want userinfo tersamarkan", got)
	}
	if got := MaskRTSPURL("rtsp://10.0.0.5:554/Streaming/channels/1902"); strings.Contains(got, "xxxxx") {
		t.Fatalf("MaskRTSPURL() tanpa kredensial tidak boleh berubah: %q", got)
	}
	if got := MaskRTSPURL("admin:secret123@host"); strings.Contains(got, "secret123") {
		t.Fatalf("MaskRTSPURL() fallback membocorkan password: %q", got)
	}
}

func TestBuildRTSPURL_SubStream(t *testing.T) {
	t.Setenv("RTSP_NVR_HOST", "10.0.0.5")
	t.Setenv("RTSP_NVR_PORT", "554")
	t.Setenv("RTSP_NVR_USER", "admin")

	got := BuildRTSPURL(19, "secret123", SubStream)
	// channelPort 19 + SubStream(2) → channelID 1902
	want := "rtsp://admin:secret123@10.0.0.5:554/Streaming/channels/1902"
	if got != want {
		t.Fatalf("BuildRTSPURL(19, sub) = %q, want %q", got, want)
	}
}

func TestBuildRTSPURL_MainStream(t *testing.T) {
	t.Setenv("RTSP_NVR_HOST", "10.0.0.5")
	t.Setenv("RTSP_NVR_PORT", "554")
	t.Setenv("RTSP_NVR_USER", "admin")

	got := BuildRTSPURL(1, "secret123", MainStream)
	// channelPort 1 + MainStream(1) → channelID 101
	want := "rtsp://admin:secret123@10.0.0.5:554/Streaming/channels/101"
	if got != want {
		t.Fatalf("BuildRTSPURL(1, main) = %q, want %q", got, want)
	}
}

func TestBuildRTSPURL_EscapeSpecialChars(t *testing.T) {
	t.Setenv("RTSP_NVR_HOST", "10.0.0.5")
	t.Setenv("RTSP_NVR_PORT", "554")
	t.Setenv("RTSP_NVR_USER", "admin")

	// Password dengan karakter khusus harus di-escape agar URI tetap valid.
	got := BuildRTSPURL(2, "p@ss:w/rd!", SubStream)
	want := "rtsp://admin:p%40ss%3Aw%2Frd%21@10.0.0.5:554/Streaming/channels/202"
	if got != want {
		t.Fatalf("BuildRTSPURL(escape) = %q, want %q", got, want)
	}
}

func TestBuildRTSPURL_EdgeCase_ChannelZero(t *testing.T) {
	t.Setenv("RTSP_NVR_HOST", "10.0.0.5")
	t.Setenv("RTSP_NVR_PORT", "554")
	t.Setenv("RTSP_NVR_USER", "admin")

	// channel_port 0 + SubStream(2) → channelID 2 (tidak panic, URI deterministik)
	got := BuildRTSPURL(0, "pw", SubStream)
	want := "rtsp://admin:pw@10.0.0.5:554/Streaming/channels/2"
	if got != want {
		t.Fatalf("BuildRTSPURL(0, sub) = %q, want %q", got, want)
	}
}

func TestBuildRTSPURL_EdgeCase_NegativeChannel(t *testing.T) {
	t.Setenv("RTSP_NVR_HOST", "10.0.0.5")
	t.Setenv("RTSP_NVR_PORT", "554")
	t.Setenv("RTSP_NVR_USER", "admin")

	// channel_port negatif tetap menghasilkan URI deterministik tanpa panic.
	got := BuildRTSPURL(-5, "pw", MainStream)
	want := "rtsp://admin:pw@10.0.0.5:554/Streaming/channels/-499"
	if got != want {
		t.Fatalf("BuildRTSPURL(-5, main) = %q, want %q", got, want)
	}
}

func TestBuildRTSPURL_Defaults(t *testing.T) {
	// Tanpa env: fallback host kosong, port 554, user admin.
	got := BuildRTSPURL(19, "secret123", SubStream)
	want := "rtsp://admin:secret123@:554/Streaming/channels/1902"
	if got != want {
		t.Fatalf("BuildRTSPURL(default) = %q, want %q", got, want)
	}
}

func TestBuildStreamPath(t *testing.T) {
	got := BuildStreamPath("9f2c1a3e-0000-4000-8000-000000000001")
	want := "cam-9f2c1a3e-0000-4000-8000-000000000001"
	if got != want {
		t.Fatalf("BuildStreamPath() = %q, want %q", got, want)
	}
}

func TestGetRTSPPort_Default(t *testing.T) {
	t.Setenv("RTSP_NVR_PORT", "")
	if got := GetRTSPPort(); got != 554 {
		t.Fatalf("GetRTSPPort() default = %d, want 554", got)
	}
}

func TestGetRTSPPort_Custom(t *testing.T) {
	t.Setenv("RTSP_NVR_PORT", "8554")
	if got := GetRTSPPort(); got != 8554 {
		t.Fatalf("GetRTSPPort() custom = %d, want 8554", got)
	}
}

func TestGetRTSPPort_Invalid(t *testing.T) {
	t.Setenv("RTSP_NVR_PORT", "not-a-number")
	if got := GetRTSPPort(); got != 554 {
		t.Fatalf("GetRTSPPort() invalid = %d, want 554", got)
	}
}

func TestResolveRTSPURL_SelectsHikvisionStreamType(t *testing.T) {
	cases := []struct {
		name       string
		raw        string
		streamType StreamType
		want       string
	}{
		{
			name:       "main URL to sub",
			raw:        "rtsp://admin:secret@36.91.27.35:554/Streaming/channels/1501",
			streamType: SubStream,
			want:       "rtsp://admin:secret@36.91.27.35:554/Streaming/channels/1502",
		},
		{
			name:       "sub URL to main",
			raw:        "rtsp://admin:secret@36.91.27.35:554/Streaming/channels/602",
			streamType: MainStream,
			want:       "rtsp://admin:secret@36.91.27.35:554/Streaming/channels/601",
		},
		{
			name:       "suffix already matches",
			raw:        "rtsp://admin:secret@host:554/Streaming/channels/1902",
			streamType: SubStream,
			want:       "rtsp://admin:secret@host:554/Streaming/channels/1902",
		},
		{
			name:       "case insensitive channels segment",
			raw:        "rtsp://admin:secret@host:554/Streaming/Channels/301",
			streamType: SubStream,
			want:       "rtsp://admin:secret@host:554/Streaming/Channels/302",
		},
		{
			name:       "query is preserved",
			raw:        "rtsp://admin:secret@host:554/Streaming/channels/1501?transportmode=unicast",
			streamType: SubStream,
			want:       "rtsp://admin:secret@host:554/Streaming/channels/1502?transportmode=unicast",
		},
		{
			name:       "non hikvision path is unchanged",
			raw:        "rtsp://admin:secret@host:554/live/ch0",
			streamType: SubStream,
			want:       "rtsp://admin:secret@host:554/live/ch0",
		},
		{
			name:       "invalid channel suffix is unchanged",
			raw:        "rtsp://admin:secret@host:554/Streaming/channels/1503",
			streamType: SubStream,
			want:       "rtsp://admin:secret@host:554/Streaming/channels/1503",
		},
		{
			name:       "invalid stream type is unchanged",
			raw:        "rtsp://admin:secret@host:554/Streaming/channels/1501",
			streamType: StreamType(0),
			want:       "rtsp://admin:secret@host:554/Streaming/channels/1501",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ResolveRTSPURL(tc.raw, 15, "unused", tc.streamType); got != tc.want {
				t.Fatalf("ResolveRTSPURL(%q, %v) = %q, want %q", tc.raw, tc.streamType, got, tc.want)
			}
		})
	}
}

func TestResolveRTSPURL_EmptyUsesBuildFallback(t *testing.T) {
	t.Setenv("RTSP_NVR_HOST", "10.0.0.5")
	t.Setenv("RTSP_NVR_PORT", "554")
	t.Setenv("RTSP_NVR_USER", "admin")

	got := ResolveRTSPURL("", 15, "secret123", SubStream)
	want := "rtsp://admin:secret123@10.0.0.5:554/Streaming/channels/1502"
	if got != want {
		t.Fatalf("ResolveRTSPURL(empty) = %q, want %q", got, want)
	}
}
