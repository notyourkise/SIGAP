package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newTestMediaMTXClient membuat MediaMTXClient dengan BaseURL dari server mock.
func newTestMediaMTXClient(server *httptest.Server) *MediaMTXClient {
	return &MediaMTXClient{
		BaseURL:    server.URL,
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
	}
}

// TestAddPath memverifikasi bahwa AddPath mengirim POST ke endpoint yang benar
// dengan body JSON yang benar, dan tidak error saat server membalas 2xx.
func TestAddPath(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
		gotBody   []byte
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		gotBody = body
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestMediaMTXClient(server)

	config := PathConfig{
		Source:                   "rtsp://admin:pw@host:554/Streaming/channels/101",
		SourceOnDemand:           true,
		SourceOnDemandCloseAfter: "10s",
		SourceProtocol:           "udp",
	}
	if err := client.AddPath("cam-123", config); err != nil {
		t.Fatalf("AddPath() unexpected error: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("AddPath() method = %q, want %q", gotMethod, http.MethodPost)
	}
	if want := "/v3/config/paths/add/cam-123"; gotPath != want {
		t.Errorf("AddPath() path = %q, want %q", gotPath, want)
	}

	// Verifikasi body JSON sesuai dengan konfigurasi yang dikirim.
	var decoded PathConfig
	if err := json.Unmarshal(gotBody, &decoded); err != nil {
		t.Fatalf("AddPath() body bukan JSON valid: %v (body=%s)", err, gotBody)
	}
	if decoded.Source != config.Source {
		t.Errorf("AddPath() body Source = %q, want %q", decoded.Source, config.Source)
	}
	if decoded.SourceOnDemand != config.SourceOnDemand {
		t.Errorf("AddPath() body SourceOnDemand = %v, want %v", decoded.SourceOnDemand, config.SourceOnDemand)
	}
	if decoded.SourceOnDemandCloseAfter != config.SourceOnDemandCloseAfter {
		t.Errorf("AddPath() body SourceOnDemandCloseAfter = %q, want %q", decoded.SourceOnDemandCloseAfter, config.SourceOnDemandCloseAfter)
	}
	if decoded.SourceProtocol != config.SourceProtocol {
		t.Errorf("AddPath() body SourceProtocol = %q, want %q", decoded.SourceProtocol, config.SourceProtocol)
	}
}

// TestAddPathStatusNoContent memverifikasi bahwa server yang membalas 200
// (bukan 204) juga tidak menghasilkan error.
func TestAddPathStatusNoContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newTestMediaMTXClient(server)
	if err := client.AddPath("cam-1", PathConfig{Source: "rtsp://x"}); err != nil {
		t.Fatalf("AddPath() unexpected error on 200: %v", err)
	}
}

// TestAddPathURLEncodesName memverifikasi bahwa nama path di-escape agar aman
// untuk digunakan pada segmen URL.
func TestAddPathURLEncodesName(t *testing.T) {
	var gotRawPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Gunakan RawPath (bukan Path) karena server Go otomatis men-decode
		// segmen URL sehingga r.URL.Path tidak lagi memuat karakter ter-escape.
		gotRawPath = r.URL.RawPath
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestMediaMTXClient(server)
	if err := client.AddPath("cam a/b", PathConfig{Source: "rtsp://x"}); err != nil {
		t.Fatalf("AddPath() unexpected error: %v", err)
	}
	if !strings.Contains(gotRawPath, "cam%20a%2Fb") {
		t.Errorf("AddPath() path tidak ter-escape dengan benar: %q", gotRawPath)
	}
}

// TestRemovePath memverifikasi bahwa RemovePath mengirim POST ke endpoint yang
// benar dan tidak error saat server membalas 2xx.
func TestRemovePath(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newTestMediaMTXClient(server)
	if err := client.RemovePath("cam-456"); err != nil {
		t.Fatalf("RemovePath() unexpected error: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("RemovePath() method = %q, want %q", gotMethod, http.MethodPost)
	}
	if want := "/v3/config/paths/remove/cam-456"; gotPath != want {
		t.Errorf("RemovePath() path = %q, want %q", gotPath, want)
	}
}

// TestGetPath memverifikasi bahwa GetPath mem-parsing JSON respon menjadi
// PathInfo dengan field Ready dan Name yang benar, untuk kasus ready=true dan
// ready=false.
func TestGetPath(t *testing.T) {
	cases := []struct {
		name      string
		ready     bool
		pathName  string
		sourceStr string
		readerCnt int
	}{
		{name: "ready true", ready: true, pathName: "cam-ready", sourceStr: "rtsp://src", readerCnt: 1},
		{name: "ready false", ready: false, pathName: "cam-notready", sourceStr: "", readerCnt: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotPath string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				// Bangun payload PathInfo sebagai JSON mentah agar readers dapat
				// berisi beberapa item.
				readers := "[]"
				if tc.readerCnt > 0 {
					readers = `[{"id":"r1"},{"id":"r2"}]`
				}
				payload := fmt.Sprintf(`{"name":%q,"source":%q,"ready":%v,"readers":%s}`,
					tc.pathName, tc.sourceStr, tc.ready, readers)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(payload))
			}))
			defer server.Close()

			client := newTestMediaMTXClient(server)
			info, err := client.GetPath(tc.pathName)
			if err != nil {
				t.Fatalf("GetPath() unexpected error: %v", err)
			}
			if info == nil {
				t.Fatal("GetPath() returned nil info")
			}
			if gotPath != "/v3/paths/get/"+tc.pathName {
				t.Errorf("GetPath() path = %q, want /v3/paths/get/%s", gotPath, tc.pathName)
			}
			if info.Name != tc.pathName {
				t.Errorf("GetPath() Name = %q, want %q", info.Name, tc.pathName)
			}
			if info.Ready != tc.ready {
				t.Errorf("GetPath() Ready = %v, want %v", info.Ready, tc.ready)
			}
			if tc.readerCnt > 0 && len(info.Readers) != 2 {
				t.Errorf("GetPath() len(Readers) = %d, want 2", len(info.Readers))
			}
		})
	}
}

// TestListPaths memverifikasi bahwa ListPaths mem-parsing array JSON menjadi
// slice PathInfo dengan panjang dan nama yang benar.
func TestListPaths(t *testing.T) {
	var gotMethod string
	var gotPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		payload := `[
			{"name":"cam-1","source":"rtsp://a","ready":true,"readers":[]},
			{"name":"cam-2","source":"rtsp://b","ready":false,"readers":[]},
			{"name":"cam-3","source":"rtsp://c","ready":true,"readers":[]}
		]`
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(payload))
	}))
	defer server.Close()

	client := newTestMediaMTXClient(server)
	paths, err := client.ListPaths()
	if err != nil {
		t.Fatalf("ListPaths() unexpected error: %v", err)
	}

	if gotMethod != http.MethodGet {
		t.Errorf("ListPaths() method = %q, want %q", gotMethod, http.MethodGet)
	}
	if want := "/v3/paths/list"; gotPath != want {
		t.Errorf("ListPaths() path = %q, want %q", gotPath, want)
	}
	if len(paths) != 3 {
		t.Fatalf("ListPaths() len = %d, want 3", len(paths))
	}
	if paths[0].Name != "cam-1" || paths[1].Name != "cam-2" || paths[2].Name != "cam-3" {
		t.Errorf("ListPaths() names = %v, %v, %v; want cam-1, cam-2, cam-3",
			paths[0].Name, paths[1].Name, paths[2].Name)
	}
}

// TestAddPathStatusError memverifikasi bahwa status non-2xx (404/500) pada
// AddPath menghasilkan error yang memuat kode status.
func TestAddPathStatusError(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusInternalServerError} {
		t.Run(fmt.Sprintf("status %d", status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				_, _ = w.Write([]byte("some error body"))
			}))
			defer server.Close()

			client := newTestMediaMTXClient(server)
			err := client.AddPath("cam-err", PathConfig{Source: "rtsp://x"})
			if err == nil {
				t.Fatalf("AddPath() expected error for status %d, got nil", status)
			}
			if !strings.Contains(err.Error(), fmt.Sprintf("%d", status)) {
				t.Errorf("AddPath() error = %q, want contains status %d", err.Error(), status)
			}
		})
	}
}

// TestRemovePathStatusError memverifikasi bahwa status non-2xx pada RemovePath
// menghasilkan error yang memuat kode status.
func TestRemovePathStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := newTestMediaMTXClient(server)
	err := client.RemovePath("cam-x")
	if err == nil {
		t.Fatal("RemovePath() expected error for status 404, got nil")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("RemovePath() error = %q, want contains 404", err.Error())
	}
}

// TestGetPathStatusError memverifikasi bahwa status non-2xx pada GetPath
// menghasilkan error.
func TestGetPathStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := newTestMediaMTXClient(server)
	info, err := client.GetPath("cam-y")
	if err == nil {
		t.Fatal("GetPath() expected error for status 500, got nil")
	}
	if info != nil {
		t.Errorf("GetPath() info = %+v, want nil on error", info)
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("GetPath() error = %q, want contains 500", err.Error())
	}
}

// TestGetPathInvalidJSON memverifikasi bahwa body JSON yang tidak valid pada
// GetPath menghasilkan error.
func TestGetPathInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{not-valid-json"))
	}))
	defer server.Close()

	client := newTestMediaMTXClient(server)
	if _, err := client.GetPath("cam-bad"); err == nil {
		t.Fatal("GetPath() expected error for invalid JSON, got nil")
	}
}

// TestListPathsInvalidJSON memverifikasi bahwa body JSON yang tidak valid pada
// ListPaths menghasilkan error.
func TestListPathsInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not-an-array"))
	}))
	defer server.Close()

	client := newTestMediaMTXClient(server)
	if _, err := client.ListPaths(); err == nil {
		t.Fatal("ListPaths() expected error for invalid JSON, got nil")
	}
}

// TestHTTPClientTimeout memverifikasi bahwa ketika HTTP client melewati batas
// timeout (server sengaja tidur lebih lama), panggilan menghasilkan error.
func TestHTTPClientTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Tidur lebih lama dari timeout client sehingga request akan timeout.
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Client dengan timeout sangat pendek (10ms) agar cepat timeout.
	client := &MediaMTXClient{
		BaseURL:    server.URL,
		HTTPClient: &http.Client{Timeout: 10 * time.Millisecond},
	}
	if err := client.AddPath("cam-t", PathConfig{Source: "rtsp://x"}); err == nil {
		t.Fatal("AddPath() expected timeout error, got nil")
	}
}

// TestNilClient memverifikasi bahwa pemanggilan method pada MediaMTXClient nil
// menghasilkan error.
func TestNilClient(t *testing.T) {
	var client *MediaMTXClient
	if err := client.AddPath("cam-n", PathConfig{}); err == nil {
		t.Fatal("AddPath() expected error for nil client, got nil")
	}
	if err := client.RemovePath("cam-n"); err == nil {
		t.Fatal("RemovePath() expected error for nil client, got nil")
	}
	if _, err := client.GetPath("cam-n"); err == nil {
		t.Fatal("GetPath() expected error for nil client, got nil")
	}
	if _, err := client.ListPaths(); err == nil {
		t.Fatal("ListPaths() expected error for nil client, got nil")
	}
}

// TestNilHTTPClient memverifikasi bahwa pemanggilan method dengan HTTPClient
// nil menghasilkan error.
func TestNilHTTPClient(t *testing.T) {
	client := &MediaMTXClient{BaseURL: "http://localhost:9997"}
	if err := client.AddPath("cam-h", PathConfig{}); err == nil {
		t.Fatal("AddPath() expected error for nil HTTPClient, got nil")
	}
	if err := client.RemovePath("cam-h"); err == nil {
		t.Fatal("RemovePath() expected error for nil HTTPClient, got nil")
	}
	if _, err := client.GetPath("cam-h"); err == nil {
		t.Fatal("GetPath() expected error for nil HTTPClient, got nil")
	}
	if _, err := client.ListPaths(); err == nil {
		t.Fatal("ListPaths() expected error for nil HTTPClient, got nil")
	}
}

// TestNewMediaMTXClientDefault memverifikasi bahwa NewMediaMTXClient membaca
// env MEDIAMTX_API_URL dan membuang trailing slash, serta menetapkan timeout.
func TestNewMediaMTXClientDefault(t *testing.T) {
	t.Setenv("MEDIAMTX_API_URL", "http://mediamtx:9997/")
	client := NewMediaMTXClient()
	if client.BaseURL != "http://mediamtx:9997" {
		t.Errorf("NewMediaMTXClient() BaseURL = %q, want http://mediamtx:9997", client.BaseURL)
	}
	if client.HTTPClient == nil {
		t.Fatal("NewMediaMTXClient() HTTPClient nil")
	}
}
