package utils

import (
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestMediaMTXPathLifecycleIntegration(t *testing.T) {
	baseURL := os.Getenv("MEDIAMTX_TEST_API_URL")
	if baseURL == "" {
		t.Skip("MEDIAMTX_TEST_API_URL tidak diisi")
	}

	client := &MediaMTXClient{
		BaseURL:    baseURL,
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
	}
	path := "sigap-contract-test"
	_ = client.RemovePath(path)
	t.Cleanup(func() { _ = client.RemovePath(path) })

	if err := client.Ping(); err != nil {
		t.Fatalf("Ping() failed: %v", err)
	}
	if err := client.AddPath(path, PathConfig{
		Source:         "rtsp://127.0.0.1:8554/test-pattern",
		SourceOnDemand: true,
		RTSPTransport:  "tcp",
	}); err != nil {
		t.Fatalf("AddPath() failed: %v", err)
	}

	configs, err := client.ListPathConfigs()
	if err != nil {
		t.Fatalf("ListPathConfigs() failed: %v", err)
	}
	found := false
	for _, config := range configs {
		if config.Name == path {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("registered path %q tidak ditemukan", path)
	}

	if err := client.RemovePath(path); err != nil {
		t.Fatalf("RemovePath() failed: %v", err)
	}
	if err := client.RemovePath(path); err != nil {
		t.Fatalf("RemovePath() second call must be idempotent: %v", err)
	}
}

// TestMediaMTXOnDemandRTSPTriggerAndNaturalCloseIntegration membuktikan secara empiris
// terhadap server MediaMTX nyata bahwa TriggerRTSP memicu path on-demand tanpa menyisakan
// reader/hlsMuxer dan sumber menutup alami sesuai sourceOnDemandCloseAfter.
func TestMediaMTXOnDemandRTSPTriggerAndNaturalCloseIntegration(t *testing.T) {
	baseURL := os.Getenv("MEDIAMTX_TEST_API_URL")
	rtspURL := os.Getenv("MEDIAMTX_TEST_RTSP_URL")
	if baseURL == "" || rtspURL == "" {
		t.Skip("MEDIAMTX_TEST_API_URL atau MEDIAMTX_TEST_RTSP_URL tidak diisi")
	}

	client := &MediaMTXClient{
		BaseURL:    baseURL,
		RTSPURL:    rtspURL,
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
	}

	// Pastikan publisher test-pattern sudah aktif di server MediaMTX uji.
	deadline := time.Now().Add(10 * time.Second)
	for {
		info, err := client.GetPath("test-pattern")
		if err == nil && info.SourceReady() {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("path test-pattern belum siap pada MediaMTX uji: err=%v info=%+v", err, info)
		}
		time.Sleep(250 * time.Millisecond)
	}

	camPath := "cam-00000000-0000-4000-8000-000000000077"
	_ = client.RemovePath(camPath)
	t.Cleanup(func() { _ = client.RemovePath(camPath) })

	if err := client.AddPath(camPath, PathConfig{
		Source:                   "rtsp://127.0.0.1:8554/test-pattern",
		SourceOnDemand:           true,
		SourceOnDemandCloseAfter: "3s",
		RTSPTransport:            "tcp",
	}); err != nil {
		t.Fatalf("AddPath() failed: %v", err)
	}

	method, err := client.TriggerOnDemand(camPath, 2*time.Second)
	if err != nil {
		t.Fatalf("TriggerOnDemand() failed: %v", err)
	}
	if method != TriggerMethodRTSP {
		t.Fatalf("TriggerOnDemand() method = %q, want %q", method, TriggerMethodRTSP)
	}

	var firstBytes int64
	readyDeadline := time.Now().Add(4 * time.Second)
	for {
		info, gErr := client.GetPath(camPath)
		if gErr == nil && info.SourceReady() && len(info.Tracks) > 0 {
			if len(info.Readers) != 0 {
				t.Fatalf("TriggerRTSP tidak boleh menyisakan reader aktif, dapat %d", len(info.Readers))
			}
			firstBytes = info.BytesReceived
			break
		}
		if time.Now().After(readyDeadline) {
			t.Fatalf("path on-demand tidak menjadi siap setelah TriggerRTSP: err=%v", gErr)
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Pastikan ada pertumbuhan byte (menunggu hingga ~1.8s untuk menampung interval GOP 1s).
	growthDeadline := time.Now().Add(1800 * time.Millisecond)
	for {
		growthInfo, gErr := client.GetPath(camPath)
		if gErr != nil {
			t.Fatalf("GetPath() growth check failed: %v", gErr)
		}
		if growthInfo.BytesReceived > firstBytes {
			break
		}
		if time.Now().After(growthDeadline) {
			t.Fatalf("BytesReceived tidak bertambah setelah 1.8s: before=%d after=%d", firstBytes, growthInfo.BytesReceived)
		}
		time.Sleep(150 * time.Millisecond)
	}

	// Pastikan tidak ada hlsMuxer tertinggal untuk camPath di MediaMTX.
	muxResp, err := client.HTTPClient.Get(baseURL + "/v3/hlsmuxers/list")
	if err != nil {
		t.Fatalf("GET /v3/hlsmuxers/list failed: %v", err)
	}
	defer func() { _ = muxResp.Body.Close() }()
	var muxList struct {
		Items []struct {
			Path string `json:"path"`
		} `json:"items"`
	}
	if decErr := json.NewDecoder(muxResp.Body).Decode(&muxList); decErr != nil {
		t.Fatalf("decode /v3/hlsmuxers/list failed: %v", decErr)
	}
	for _, item := range muxList.Items {
		if item.Path == camPath {
			t.Fatalf("TriggerRTSP tidak boleh membuat hlsMuxer untuk %q", camPath)
		}
	}

	// Tunggu sourceOnDemandCloseAfter ("3s") tuntas lalu pastikan sumber berhenti alami.
	closeDeadline := time.Now().Add(5 * time.Second)
	for {
		closedInfo, cErr := client.GetPath(camPath)
		if cErr == nil && !closedInfo.SourceReady() && len(closedInfo.Readers) == 0 {
			return
		}
		if time.Now().After(closeDeadline) {
			t.Fatalf("sumber on-demand belum menutup alami setelah sourceOnDemandCloseAfter=3s: err=%v info=%+v", cErr, closedInfo)
		}
		time.Sleep(250 * time.Millisecond)
	}
}
