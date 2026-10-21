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
	Name                     string `json:"name,omitempty"`
	Source                   string `json:"source"`
	SourceOnDemand           bool   `json:"sourceOnDemand"`
	SourceOnDemandCloseAfter string `json:"sourceOnDemandCloseAfter,omitempty"`
	RTSPTransport            string `json:"rtspTransport,omitempty"`
}

// PathInfo — informasi path dari MediaMTX
type PathInfo struct {
	Name      string        `json:"name"`
	Source    interface{}   `json:"source"`
	Available bool          `json:"available"`
	Online    bool          `json:"online"`
	Ready     bool          `json:"ready"` // kompatibilitas MediaMTX sebelum v1.20
	Readers   []interface{} `json:"readers"`
}

// SourceReady memakai field available MediaMTX v1.20. Field ready hanya
// dipakai sebagai fallback agar rolling upgrade dari versi lama tetap aman.
func (p PathInfo) SourceReady() bool {
	return p.Available || p.Ready
}

// defaultMediaMTXURL adalah URL default MediaMTX REST API untuk proses backend
// yang dijalankan langsung di host (tanpa Docker). Deployment Docker selalu
// mengirim MEDIAMTX_API_URL=http://mediamtx:9997 melalui compose.
const defaultMediaMTXURL = "http://127.0.0.1:9997"

// NewMediaMTXClient membuat MediaMTXClient baru dengan membaca URL API dari
// env MEDIAMTX_API_URL. Bila env kosong, gunakan MediaMTX lokal pada
// http://127.0.0.1:9997.
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
		return resp.StatusCode, respBody, fmt.Errorf("MediaMTX %s %s gagal dengan status %d: %s", method, endpoint, resp.StatusCode, string(respBody))
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
	endpoint := "/v3/config/paths/delete/" + url.PathEscape(name)
	status, _, err := c.doRequest(http.MethodDelete, endpoint, nil)
	if status == http.StatusNotFound {
		return nil
	}
	return err
}

// ListPathConfigs mengambil daftar path yang terdaftar, terpisah dari
// ListPaths yang hanya merepresentasikan kondisi operasional sumber.
func (c *MediaMTXClient) ListPathConfigs() ([]PathConfig, error) {
	_, respBody, err := c.doRequest(http.MethodGet, "/v3/config/paths/list?itemsPerPage=10000", nil)
	if err != nil {
		return nil, err
	}

	var paged struct {
		Items []PathConfig `json:"items"`
	}
	if err := json.Unmarshal(respBody, &paged); err == nil && paged.Items != nil {
		return paged.Items, nil
	}

	var configs []PathConfig
	if err := json.Unmarshal(respBody, &configs); err != nil {
		return nil, fmt.Errorf("gagal decode daftar konfigurasi path MediaMTX: %w", err)
	}
	return configs, nil
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
	endpoint := "/v3/paths/list?itemsPerPage=10000"
	_, respBody, err := c.doRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	// MediaMTX v1.20+ mengembalikan objek terpaginasi {itemCount,pageCount,items}.
	// Beberapa versi mengembalikan array langsung. Tangani kedua format tersebut.
	var paged struct {
		Items []PathInfo `json:"items"`
	}
	if err := json.Unmarshal(respBody, &paged); err == nil && paged.Items != nil {
		return paged.Items, nil
	}

	var paths []PathInfo
	if err := json.Unmarshal(respBody, &paths); err != nil {
		Log.Errorf("❌ Gagal decode daftar path MediaMTX: %v", err)
		return nil, fmt.Errorf("gagal decode daftar path MediaMTX: %w", err)
	}
	return paths, nil
}

// Ping memeriksa apakah MediaMTX siap menerima permintaan dengan memanggil
// endpoint global config. Mengembalikan nil bila server merespons 2xx.
func (c *MediaMTXClient) Ping() error {
	_, _, err := c.doRequest(http.MethodGet, "/v3/config/global/get", nil)
	return err
}

// GetSnapshot mengambil snapshot JPEG sebuah path dari MediaMTX. Endpoint
// snapshot mengembalikan gambar berformat JPEG (Content-Type image/jpeg).
// Mengembalikan error pada status non-2xx atau bila client tidak valid.
func (c *MediaMTXClient) GetSnapshot(name string) ([]byte, error) {
	endpoint := "/v3/paths/get/" + url.PathEscape(name) + "/snapshot"
	_, body, err := c.doRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	return body, nil
}
