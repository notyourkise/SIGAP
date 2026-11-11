package utils

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	// TriggerMethodRTSP menandakan verifikasi memicu path on-demand melalui RTSP DESCRIBE.
	TriggerMethodRTSP = "mediamtx-rtsp"
	// TriggerMethodHLS menandakan verifikasi memicu path on-demand melalui HLS fallback.
	TriggerMethodHLS = "mediamtx-hls"
)

// MediaMTXClient mengelola komunikasi dengan MediaMTX REST API v3.
// MediaMTX adalah media server (proxy RTSP/HLS/WebRTC) yang dipakai SIGAP
// untuk me-relay aliran kamera dari NVR ke client.
type MediaMTXClient struct {
	BaseURL string
	// HLSURL adalah alamat HTTP HLS MediaMTX. Dipakai untuk proksi /hls/* serta
	// fallback pemicu on-demand bila RTSPURL belum tersedia.
	HLSURL string
	// WebRTCURL adalah alamat HTTP WebRTC/WHEP MediaMTX (port 8889) untuk
	// proksi jalur /stream/* yang terautentikasi. Bila kosong, diturunkan dari
	// BaseURL (port 9997 -> 8889).
	WebRTCURL string
	// RTSPURL adalah alamat RTSP internal MediaMTX (port 8554) untuk memicu path
	// on-demand via RTSP DESCRIBE saat verifikasi aktif tanpa menahan sesi/muxer HLS.
	RTSPURL    string
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
	Name          string        `json:"name"`
	Source        interface{}   `json:"source"`
	Available     bool          `json:"available"`
	Online        bool          `json:"online"`
	Ready         bool          `json:"ready"` // kompatibilitas MediaMTX sebelum v1.20
	Tracks        []interface{} `json:"tracks"`
	Readers       []interface{} `json:"readers"`
	BytesReceived int64         `json:"bytesReceived"`
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
// env MEDIAMTX_API_URL, HLS dari MEDIAMTX_HLS_URL, WebRTC dari MEDIAMTX_WEBRTC_URL,
// dan RTSP internal dari MEDIAMTX_RTSP_URL. Bila env opsional kosong, host
// diturunkan dari MEDIAMTX_API_URL (default lokal 127.0.0.1).
// Trailing slash pada URL dibuang agar penyusunan endpoint konsisten. HTTP
// client diberi timeout 10 detik agar panggilan tidak menggantung selamanya.
func NewMediaMTXClient() *MediaMTXClient {
	baseURL := getenvOr("MEDIAMTX_API_URL", defaultMediaMTXURL)
	hlsURL := getenvOr("MEDIAMTX_HLS_URL", deriveMediaMTXURL(baseURL, "", "8888", "http://127.0.0.1:8888"))
	webrtcURL := getenvOr("MEDIAMTX_WEBRTC_URL", deriveMediaMTXURL(baseURL, "", "8889", "http://127.0.0.1:8889"))
	rtspURL := getenvOr("MEDIAMTX_RTSP_URL", deriveMediaMTXURL(baseURL, "rtsp", "8554", "rtsp://127.0.0.1:8554"))
	return &MediaMTXClient{
		BaseURL:    strings.TrimSuffix(baseURL, "/"),
		HLSURL:     strings.TrimSuffix(hlsURL, "/"),
		WebRTCURL:  strings.TrimSuffix(webrtcURL, "/"),
		RTSPURL:    strings.TrimSuffix(rtspURL, "/"),
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func deriveMediaMTXURL(baseURL, targetScheme, targetPort, fallback string) string {
	u, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || u.Hostname() == "" {
		return fallback
	}
	scheme := targetScheme
	if scheme == "" {
		scheme = u.Scheme
		if scheme == "" {
			scheme = "http"
		}
	}
	return scheme + "://" + net.JoinHostPort(u.Hostname(), targetPort)
}

// doRequest menjalankan permintaan HTTP ke endpoint MediaMTX dan mengembalikan
// status code serta isi body (sebagai []byte). Helper ini mengurangi duplikasi
// pada seluruh method CRUD path. Bila client atau HTTPClient nil, dikembalikan
// error. Error non-2xx diterjemahkan menjadi pesan deskriptif yang memuat
// status code.
func (c *MediaMTXClient) doRequest(method, endpoint string, body []byte) (int, []byte, error) {
	return c.doRequestWithLog(method, endpoint, body, true)
}

// doRequestWithLog menjalankan request seperti doRequest, tetapi dapat
// menonaktifkan log error untuk respons non-2xx yang memang diharapkan.
// Dipakai AddPath: respons "path already exists" adalah jalur normal saat
// ensureStreamPath akan melakukan patch, bukan kegagalan yang perlu dicetak
// sebagai ERROR.
func (c *MediaMTXClient) doRequestWithLog(method, endpoint string, body []byte, logError bool) (int, []byte, error) {
	if c == nil {
		return 0, nil, fmt.Errorf("MediaMTXClient nil")
	}
	if c.HTTPClient == nil {
		return 0, nil, fmt.Errorf("MediaMTX HTTPClient nil")
	}

	req, err := http.NewRequest(method, c.BaseURL+endpoint, bytes.NewReader(body))
	if err != nil {
		safeErr := RedactRTSPCredentialsInText(err.Error())
		Log.Errorf("❌ Gagal membuat request MediaMTX %s %s: %s", method, endpoint, safeErr)
		return 0, nil, fmt.Errorf("gagal membuat request MediaMTX: %s", safeErr)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		safeErr := RedactRTSPCredentialsInText(err.Error())
		Log.Errorf("❌ Gagal melakukan request MediaMTX %s %s: %s", method, endpoint, safeErr)
		return 0, nil, fmt.Errorf("gagal melakukan request MediaMTX: %s", safeErr)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		safeErr := RedactRTSPCredentialsInText(err.Error())
		Log.Errorf("❌ Gagal membaca body respon MediaMTX %s %s: %s", method, endpoint, safeErr)
		return 0, nil, fmt.Errorf("gagal membaca body respon MediaMTX: %s", safeErr)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		safeBody := RedactRTSPCredentialsInText(string(respBody))
		if logError {
			Log.Errorf("❌ MediaMTX %s %s mengembalikan status %d: %s", method, endpoint, resp.StatusCode, safeBody)
		}
		return resp.StatusCode, respBody, fmt.Errorf("MediaMTX %s %s gagal dengan status %d: %s", method, endpoint, resp.StatusCode, safeBody)
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
	_, _, err = c.doRequestWithLog(http.MethodPost, endpoint, jsonBody, false)
	return err
}

// PatchPath memperbarui konfigurasi path yang sudah terdaftar. Dipakai saat
// kualitas stream berubah (main/sub) atau sumber RTSP diperbarui; MediaMTX
// akan me-restart source bila konfigurasi source berubah.
func (c *MediaMTXClient) PatchPath(name string, config PathConfig) error {
	jsonBody, err := json.Marshal(config)
	if err != nil {
		Log.Errorf("❌ Gagal encode konfigurasi patch path %s: %v", name, err)
		return fmt.Errorf("gagal encode konfigurasi path: %w", err)
	}

	endpoint := "/v3/config/paths/patch/" + url.PathEscape(name)
	_, _, err = c.doRequest(http.MethodPatch, endpoint, jsonBody)
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

// hlsBaseURL mengembalikan alamat HLS MediaMTX. Bila HLSURL belum diset,
// turunkan dari BaseURL dengan mengganti port API 9997 menjadi 8888. Ini
// menjaga kompatibilitas pemanggil lama yang hanya mengisi BaseURL.
func (c *MediaMTXClient) hlsBaseURL() string {
	if c == nil {
		return ""
	}
	if c.HLSURL != "" {
		return strings.TrimSuffix(c.HLSURL, "/")
	}
	return strings.Replace(strings.TrimSuffix(c.BaseURL, "/"), ":9997", ":8888", 1)
}

// HLSBaseURL mengembalikan alamat HLS MediaMTX untuk proksi jalur /hls/*.
func (c *MediaMTXClient) HLSBaseURL() string {
	return c.hlsBaseURL()
}

// WebRTCBaseURL mengembalikan alamat WebRTC/WHEP MediaMTX untuk proksi jalur
// /stream/*. Bila WebRTCURL belum diset, diturunkan dari BaseURL (9997 -> 8889).
func (c *MediaMTXClient) WebRTCBaseURL() string {
	if c == nil {
		return ""
	}
	if c.WebRTCURL != "" {
		return strings.TrimSuffix(c.WebRTCURL, "/")
	}
	return strings.Replace(strings.TrimSuffix(c.BaseURL, "/"), ":9997", ":8889", 1)
}

func (c *MediaMTXClient) rtspBaseURL() string {
	if c == nil {
		return ""
	}
	if trimmed := strings.TrimSpace(c.RTSPURL); trimmed != "" {
		return strings.TrimSuffix(trimmed, "/")
	}
	u, err := url.Parse(strings.TrimSpace(c.BaseURL))
	if err != nil || u.Hostname() == "" {
		return ""
	}
	// Turunkan alamat RTSP port 8554 otomatis hanya bila pemanggil mengarah ke
	// port API standar MediaMTX (9997) tanpa mengisi RTSPURL.
	if u.Port() == "9997" {
		return "rtsp://" + net.JoinHostPort(u.Hostname(), "8554")
	}
	return ""
}

// RTSPBaseURL mengembalikan alamat RTSP internal MediaMTX untuk pemicu on-demand
// verifikasi aktif, atau string kosong bila klien hanya dikonfigurasi untuk mock HTTP/HLS.
func (c *MediaMTXClient) RTSPBaseURL() string {
	return c.rtspBaseURL()
}

func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, os.ErrDeadlineExceeded) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

// TriggerRTSP memicu path on-demand MediaMTX melalui RTSP DESCRIBE singkat lalu
// langsung menutup koneksi TCP. Berbeda dengan TriggerHLS, pemicu RTSP tidak
// membuat hlsSession atau hlsMuxer internal sehingga pembaca MediaMTX langsung 0
// dan timer sourceOnDemandCloseAfter (5s) menutup sumber NVR/GSM tepat waktu.
func (c *MediaMTXClient) TriggerRTSP(name string, timeout time.Duration) error {
	if c == nil {
		return fmt.Errorf("MediaMTXClient tidak siap")
	}
	base := c.rtspBaseURL()
	if base == "" {
		return fmt.Errorf("alamat RTSP MediaMTX kosong")
	}
	u, err := url.Parse(base)
	if err != nil || u.Hostname() == "" {
		return fmt.Errorf("alamat RTSP MediaMTX tidak valid: %s", RedactRTSPCredentialsInText(base))
	}
	port := u.Port()
	if port == "" {
		port = "8554"
	}
	dialAddr := net.JoinHostPort(u.Hostname(), port)
	if timeout <= 0 {
		timeout = 2 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", dialAddr)
	if err != nil {
		return fmt.Errorf("gagal menghubungi RTSP MediaMTX (%s): %w", dialAddr, err)
	}
	defer func() { _ = conn.Close() }()

	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	targetURL := strings.TrimSuffix(base, "/") + "/" + url.PathEscape(name)
	req := fmt.Sprintf("DESCRIBE %s RTSP/1.0\r\nCSeq: 1\r\nAccept: application/sdp\r\nUser-Agent: SIGAP-StreamVerifier/1.0\r\n\r\n", targetURL)
	if _, err := io.WriteString(conn, req); err != nil {
		if isTimeoutError(err) {
			return nil
		}
		return fmt.Errorf("gagal mengirim RTSP DESCRIBE ke MediaMTX: %w", err)
	}

	reader := bufio.NewReaderSize(conn, 4096)
	statusLine, err := reader.ReadString('\n')
	if err != nil && statusLine == "" {
		if isTimeoutError(err) {
			// MediaMTX sudah menerima DESCRIBE dan sedang menunggu handshake sumber RTSP hulu.
			return nil
		}
		return fmt.Errorf("gagal membaca respon RTSP MediaMTX: %w", err)
	}

	fields := strings.Fields(strings.TrimSpace(statusLine))
	if len(fields) < 2 || !strings.HasPrefix(fields[0], "RTSP/") {
		return fmt.Errorf("respon RTSP MediaMTX tidak valid: %q", strings.TrimSpace(statusLine))
	}
	statusCode, convErr := strconv.Atoi(fields[1])
	if convErr != nil {
		return fmt.Errorf("kode status RTSP MediaMTX tidak valid: %q", fields[1])
	}

	// 400/401/403/404 pada DESCRIBE langsung dari MediaMTX menandakan path tidak terdaftar
	// atau pembatasan auth/konfigurasi — bukan bukti kamera hulu offline.
	if statusCode == http.StatusBadRequest ||
		statusCode == http.StatusUnauthorized ||
		statusCode == http.StatusForbidden ||
		statusCode == http.StatusNotFound {
		return fmt.Errorf("RTSP MediaMTX mengembalikan status %d", statusCode)
	}

	// Kuras header respon secukupnya bila sudah tersedia sebelum menutup soket TCP.
	for i := 0; i < 64; i++ {
		line, hdrErr := reader.ReadString('\n')
		if hdrErr != nil || line == "\r\n" || line == "\n" {
			break
		}
	}
	return nil
}

// TriggerOnDemand memicu path on-demand menggunakan RTSP DESCRIBE bila tersedia,
// atau fallback ke HLS ketika hanya endpoint HTTP/HLS (mis. mock server uji) yang diisi.
func (c *MediaMTXClient) TriggerOnDemand(name string, timeout time.Duration) (string, error) {
	if c == nil {
		return TriggerMethodRTSP, fmt.Errorf("MediaMTXClient tidak siap")
	}
	if c.rtspBaseURL() != "" {
		return TriggerMethodRTSP, c.TriggerRTSP(name, timeout)
	}
	return TriggerMethodHLS, c.TriggerHLS(name, timeout)
}

// TriggerHLS memicu path on-demand MediaMTX agar sumber RTSP mulai ditarik.
// Permintaan HLS normalnya menggantung sampai segmen pertama tersedia; timeout
// karena itu bukan kegagalan selama koneksi berhasil. Status 5xx/3xx dianggap
// pemicu normal, sedangkan 404/401/403 menandakan path atau konfigurasi HLS
// salah dan dikembalikan sebagai error agar tidak salah diklasifikasikan
// sebagai kamera offline. Hasil sebenarnya tetap dibaca dari endpoint path API.
func (c *MediaMTXClient) TriggerHLS(name string, timeout time.Duration) error {
	if c == nil || c.HTTPClient == nil {
		return fmt.Errorf("MediaMTXClient tidak siap")
	}
	base := c.hlsBaseURL()
	if base == "" {
		return fmt.Errorf("alamat HLS MediaMTX kosong")
	}
	if timeout <= 0 {
		timeout = 2 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	endpoint := base + "/" + url.PathEscape(name) + "/index.m3u8"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("gagal membuat request HLS: %w", err)
	}
	resp, err := c.HTTPClient.Do(req)
	if resp != nil {
		status := resp.StatusCode
		_ = resp.Body.Close()
		// 404/401/403 menandakan path atau konfigurasi HLS salah, bukan
		// sumber kamera yang belum siap. Jangan hitung sebagai kegagalan
		// kamera; biarkan verifikasi melaporkan unknown.
		if status == http.StatusNotFound || status == http.StatusUnauthorized || status == http.StatusForbidden {
			return fmt.Errorf("HLS MediaMTX mengembalikan status %d", status)
		}
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		// Timeout adalah perilaku normal HLS live; path sudah dipicu.
		return nil
	}
	return fmt.Errorf("gagal memicu HLS MediaMTX: %w", err)
}

// GetSnapshot mengambil snapshot JPEG sebuah path dari MediaMTX. Endpoint
// snapshot mengembalikan gambar berformat JPEG (Content-Type image/jpeg).
// Mengembalikan error pada status non-2xx atau bila client tidak valid.
func (c *MediaMTXClient) GetSnapshot(name string) ([]byte, error) {
	endpoint := "/v3/paths/get/" + url.PathEscape(name) + "/snapshot"
	// Pemanggil (scheduler) menangani kegagalan snapshot; jangan mencetak ERROR
	// untuk setiap path yang belum siap atau endpoint yang tidak tersedia.
	_, body, err := c.doRequestWithLog(http.MethodGet, endpoint, nil, false)
	if err != nil {
		return nil, err
	}
	return body, nil
}
