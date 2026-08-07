package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// MediaMTXClient mengelola komunikasi dengan MediaMTX REST API v3.
// MediaMTX adalah media server (proxy RTSP/HLS/WebRTC) yang dipakai SIGAP
// untuk me-relay aliran kamera dari NVR ke client.
type MediaMTXClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// PathConfig — konfigurasi yang dikirim ke MediaMTX saat mendaftarkan path
type PathConfig struct {
	Source                   string `json:"source"`
	SourceOnDemand           bool   `json:"sourceOnDemand"`
	SourceOnDemandCloseAfter string `json:"sourceOnDemandCloseAfter,omitempty"`
	SourceProtocol           string `json:"sourceProtocol,omitempty"`
}

// PathInfo — informasi path dari MediaMTX
type PathInfo struct {
	Name    string        `json:"name"`
	Source  interface{}   `json:"source"`
	Ready   bool          `json:"ready"`
	Readers []interface{} `json:"readers"`
}

// defaultMediaMTXURL adalah URL default MediaMTX REST API bila env tidak di-set.
const defaultMediaMTXURL = "http://mediamtx:9997"

// NewMediaMTXClient membuat MediaMTXClient baru dengan membaca URL API dari
// env MEDIAMTX_API_URL. Bila env kosong, gunakan default http://mediamtx:9997.
// Trailing slash pada URL dibuang agar penyusunan endpoint konsisten. HTTP
// client diberi timeout 10 detik agar panggilan tidak menggantung selamanya.
func NewMediaMTXClient() *MediaMTXClient {
	baseURL := getenvOr("MEDIAMTX_API_URL", defaultMediaMTXURL)
	return &MediaMTXClient{
		BaseURL:    strings.TrimSuffix(baseURL, "/"),
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// doRequest menjalankan permintaan HTTP ke endpoint MediaMTX dan mengembalikan
// status code serta isi body (sebagai []byte). Helper ini mengurangi duplikasi
// pada seluruh method CRUD path. Bila client atau HTTPClient nil, dikembalikan
// error. Error non-2xx diterjemahkan menjadi pesan deskriptif yang memuat
// status code.
func (c *MediaMTXClient) doRequest(method, endpoint string, body []byte) (int, []byte, error) {
	if c == nil {
		return 0, nil, fmt.Errorf("MediaMTXClient nil")
	}
	if c.HTTPClient == nil {
		return 0, nil, fmt.Errorf("MediaMTX HTTPClient nil")
	}

	req, err := http.NewRequest(method, c.BaseURL+endpoint, bytes.NewReader(body))
	if err != nil {
		Log.Errorf("❌ Gagal membuat request MediaMTX %s %s: %v", method, endpoint, err)
		return 0, nil, fmt.Errorf("gagal membuat request MediaMTX: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		Log.Errorf("❌ Gagal melakukan request MediaMTX %s %s: %v", method, endpoint, err)
		return 0, nil, fmt.Errorf("gagal melakukan request MediaMTX: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		Log.Errorf("❌ Gagal membaca body respon MediaMTX %s %s: %v", method, endpoint, err)
		return 0, nil, fmt.Errorf("gagal membaca body respon MediaMTX: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		Log.Errorf("❌ MediaMTX %s %s mengembalikan status %d: %s", method, endpoint, resp.StatusCode, string(respBody))
		return resp.StatusCode, respBody, fmt.Errorf("MediaMTX %s %s gagal dengan status %d", method, endpoint, resp.StatusCode)
	}

	return resp.StatusCode, respBody, nil
}

// AddPath mendaftarkan (menambahkan) sebuah path baru ke MediaMTX. Konfigurasi
// path dikirim sebagai JSON body. Mengembalikan error bila status respon bukan
// 2xx atau bila client tidak valid.
func (c *MediaMTXClient) AddPath(name string, config PathConfig) error {
	jsonBody, err := json.Marshal(config)
	if err != nil {
		Log.Errorf("❌ Gagal encode konfigurasi path %s: %v", name, err)
		return fmt.Errorf("gagal encode konfigurasi path: %w", err)
	}

	endpoint := "/v3/config/paths/add/" + url.PathEscape(name)
	_, _, err = c.doRequest(http.MethodPost, endpoint, jsonBody)
	return err
}

// RemovePath menghapus sebuah path dari MediaMTX. Mengembalikan error bila
// status respon bukan 2xx atau bila client tidak valid.
func (c *MediaMTXClient) RemovePath(name string) error {
	endpoint := "/v3/config/paths/remove/" + url.PathEscape(name)
	_, _, err := c.doRequest(http.MethodPost, endpoint, nil)
	return err
}

// GetPath mengambil informasi detail sebuah path dari MediaMTX dan
// mendekode-nya ke dalam PathInfo. Mengembalikan error pada status non-2xx,
// kegagalan decode JSON, atau client tidak valid.
func (c *MediaMTXClient) GetPath(name string) (*PathInfo, error) {
	endpoint := "/v3/paths/get/" + url.PathEscape(name)
	_, respBody, err := c.doRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var info PathInfo
	if err := json.Unmarshal(respBody, &info); err != nil {
		Log.Errorf("❌ Gagal decode info path %s: %v", name, err)
		return nil, fmt.Errorf("gagal decode info path MediaMTX: %w", err)
	}
	return &info, nil
}

// ListPaths mengambil daftar seluruh path yang terdaftar di MediaMTX dan
// mendekode-nya ke dalam slice PathInfo. Mengembalikan error pada status
// non-2xx, kegagalan decode JSON, atau client tidak valid.
func (c *MediaMTXClient) ListPaths() ([]PathInfo, error) {
	endpoint := "/v3/paths/list"
	_, respBody, err := c.doRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var paths []PathInfo
	if err := json.Unmarshal(respBody, &paths); err != nil {
		Log.Errorf("❌ Gagal decode daftar path MediaMTX: %v", err)
		return nil, fmt.Errorf("gagal decode daftar path MediaMTX: %w", err)
	}
	return paths, nil
}
