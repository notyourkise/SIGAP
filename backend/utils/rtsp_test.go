package utils

import (
	"testing"
)

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
