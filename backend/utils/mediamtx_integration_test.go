package utils

import (
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
