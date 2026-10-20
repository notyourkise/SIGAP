package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"sigap-backend/models"
	"sigap-backend/utils"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// streamCCTVRow membangun sqlmock rows CCTV dengan seluruh kolom tabel cctvs.
// Kolom streaming (stream_enabled, channel_port, rtsp_password) dikendalikan
// parameter agar mudah diuji berbagai kondisi.
//
// Parameter variadic `extras` (opsional) dapat berisi:
//   - streamLive bool          → nilai kolom stream_live (default false)
//   - streamCheckedAt interface{} → nilai kolom stream_checked_at (default nil)
//   - streamState string       → nilai kolom stream_state (default "" untuk fallback data lama)
//
// Pemanggilan lama tanpa extras tetap kompatibel (default false/nil/"").
func streamCCTVRow(id uuid.UUID, streamEnabled bool, channelPort int, rtspPassword string, extras ...interface{}) *sqlmock.Rows {
	streamLive := false
	var streamCheckedAt interface{} = nil
	streamState := ""
	if len(extras) > 0 {
		if v, ok := extras[0].(bool); ok {
			streamLive = v
		}
	}
	if len(extras) > 1 {
		streamCheckedAt = extras[1]
	}
	if len(extras) > 2 {
		if v, ok := extras[2].(string); ok {
			streamState = v
		}
	}
	return sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"kode_cctv", "id_channel", "nama", "lokasi_area",
		"longitude", "latitude", "jenis_cctv", "sumber_listrik",
		"tipe_koneksi", "indoor_outdoor", "pemilik", "status",
		"ip_nvr", "ip_camera", "keterangan", "qr_payload", "health_score",
		"channel_port", "rtsp_password", "stream_enabled", "stream_live", "stream_state", "stream_checked_at",
	}).AddRow(id, time.Now(), time.Now(), nil, "CCTV-STREAM", "CH-01", "Kamera Streaming", "Area Utama",
		117.1, -0.5, "Dome", "PLN", "Fiber", "outdoor", "Pemkot", "aktif",
		"192.168.1.10", "192.168.1.11", "Aman", "payload", 100, channelPort, rtspPassword, streamEnabled,
		streamLive, streamState, streamCheckedAt)
}

// encryptedTestPassword menghasilkan ciphertext AES-256 dari password polos
// memakai AES_SECRET_KEY yang sama dengan yang dipakai handler untuk dekripsi.
func encryptedTestPassword(t *testing.T, plaintext string) string {
	t.Helper()
	t.Setenv("AES_SECRET_KEY", "stream-test-aes-secret-key-1234567890")
	enc, err := utils.EncryptAES256(plaintext)
	assert.NoError(t, err)
	return enc
}

// mockMTXConfig mengendalikan perilaku server MediaMTX tiruan.
type mockMTXConfig struct {
	addErr               error
	patchErr             error
	removeErr            error
	getErr               error
	listErr              error
	pingErr              error
	getInfo              *utils.PathInfo
	listPaths            []utils.PathInfo
	listConfigs          []utils.PathConfig
	addFailBeforeSuccess int
	addAlreadyExists     bool
	onPatch              func()
}

// newMockMediaMTXClient membuat klien MediaMTX yang menunjuk ke server tiruan
// (httptest). Digunakan untuk menguji handler tanpa memanggil MediaMTX asli.
// Mengembalikan klien dan fungsi cleanup untuk menutup server.
func newMockMediaMTXClient(t *testing.T, cfg mockMTXConfig) (*utils.MediaMTXClient, func()) {
	t.Helper()
	addAttempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/config/paths/add/"):
			addAttempts++
			if cfg.addAlreadyExists {
				http.Error(w, `{"error":"path already exists"}`, http.StatusBadRequest)
				return
			}
			if cfg.addErr != nil || (cfg.addFailBeforeSuccess > 0 && addAttempts <= cfg.addFailBeforeSuccess) {
				http.Error(w, "gagal menambah path", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
		case strings.Contains(r.URL.Path, "/config/paths/patch/") && r.Method == http.MethodPatch:
			if cfg.onPatch != nil {
				cfg.onPatch()
			}
			if cfg.patchErr != nil {
				http.Error(w, "gagal patch path", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
		case strings.Contains(r.URL.Path, "/config/paths/delete/") && r.Method == http.MethodDelete:
			if cfg.removeErr != nil {
				http.Error(w, "gagal menghapus path", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
		case strings.Contains(r.URL.Path, "/v3/paths/get/"):
			if cfg.getErr != nil {
				http.Error(w, "gagal ambil path", http.StatusInternalServerError)
				return
			}
			if cfg.getInfo != nil {
				_ = json.NewEncoder(w).Encode(cfg.getInfo)
				return
			}
			http.Error(w, "path tidak ditemukan", http.StatusNotFound)
		case strings.Contains(r.URL.Path, "/v3/paths/list"):
			if cfg.listErr != nil {
				http.Error(w, "gagal daftar path", http.StatusInternalServerError)
				return
			}
			_ = json.NewEncoder(w).Encode(cfg.listPaths)
		case strings.Contains(r.URL.Path, "/v3/config/paths/list"):
			_ = json.NewEncoder(w).Encode(fiber.Map{"items": cfg.listConfigs})
		case strings.Contains(r.URL.Path, "/v3/config/global/get"):
			if cfg.pingErr != nil {
				http.Error(w, "belum siap", http.StatusServiceUnavailable)
				return
			}
			w.WriteHeader(http.StatusOK)
		default:
			http.Error(w, "endpoint tidak dikenal", http.StatusNotFound)
		}
	}))

	client := &utils.MediaMTXClient{BaseURL: srv.URL, HTTPClient: srv.Client()}
	return client, srv.Close
}

// injectMockMTX menyuntikkan klien MediaMTX tiruan ke streamClient global dan
// menjamin restore ke klien baru setelah pengujian selesai.
func injectMockMTX(t *testing.T, cfg mockMTXConfig) {
	t.Helper()
	client, close := newMockMediaMTXClient(t, cfg)
	setStreamClient(client)
	t.Cleanup(func() {
		close()
		setStreamClient(utils.NewMediaMTXClient())
	})
}

func TestStartStream_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	injectMockMTX(t, mockMTXConfig{}) // AddPath berhasil

	id := uuid.New()
	encPwd := encryptedTestPassword(t, "rtsp-secret")
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(streamCCTVRow(id, true, 19, encPwd))

	app := newTestApp()
	app.Post("/streams/:id/start", StartStream)

	resp, err := app.Test(httptest.NewRequest("POST", "/streams/"+id.String()+"/start", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	data := result["data"].(map[string]interface{})
	assert.Contains(t, data["whep_url"].(string), "/stream/cam-"+id.String()+"/whep")
	assert.Equal(t, "cam-"+id.String(), data["path"])
	assert.Equal(t, "sub", data["quality"])
	assert.Equal(t, true, data["mediamtx_reachable"])
	assert.Equal(t, true, data["path_registered"])
	assert.Equal(t, false, data["source_ready"])
	assert.NotContains(t, data, "source", "respons start tidak boleh membocorkan URL/kredensial RTSP")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStartStream_AlreadyRegistered_PatchesPath(t *testing.T) {
	_, mock := setupMockGORM(t)
	patched := false
	injectMockMTX(t, mockMTXConfig{
		addAlreadyExists: true,
		getInfo:          &utils.PathInfo{Name: "cam-ready", Available: true},
		onPatch:          func() { patched = true },
	})

	id := uuid.New()
	encPwd := encryptedTestPassword(t, "rtsp-secret")
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(streamCCTVRow(id, true, 15, encPwd))

	app := newTestApp()
	app.Post("/streams/:id/start", StartStream)

	resp, err := app.Test(httptest.NewRequest("POST", "/streams/"+id.String()+"/start?quality=main", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.True(t, patched, "path yang sudah ada harus di-patch agar kualitas main/sub benar-benar berubah")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStartStream_NotFound(t *testing.T) {
	_, mock := setupMockGORM(t)
	injectMockMTX(t, mockMTXConfig{})

	id := uuid.New()
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnError(gorm.ErrRecordNotFound)

	app := newTestApp()
	app.Post("/streams/:id/start", StartStream)

	resp, err := app.Test(httptest.NewRequest("POST", "/streams/"+id.String()+"/start", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStartStream_StreamDisabled(t *testing.T) {
	_, mock := setupMockGORM(t)
	injectMockMTX(t, mockMTXConfig{})

	id := uuid.New()
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(streamCCTVRow(id, false, 19, "encrypted-rtsp"))

	app := newTestApp()
	app.Post("/streams/:id/start", StartStream)

	resp, err := app.Test(httptest.NewRequest("POST", "/streams/"+id.String()+"/start", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "Streaming belum diaktifkan", result["message"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStartStream_NoChannelPort(t *testing.T) {
	_, mock := setupMockGORM(t)
	injectMockMTX(t, mockMTXConfig{})

	id := uuid.New()
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(streamCCTVRow(id, true, 0, "encrypted-rtsp"))

	app := newTestApp()
	app.Post("/streams/:id/start", StartStream)

	resp, err := app.Test(httptest.NewRequest("POST", "/streams/"+id.String()+"/start", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "Channel port belum dikonfigurasi", result["message"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStopStream_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	injectMockMTX(t, mockMTXConfig{}) // RemovePath berhasil

	id := uuid.New()
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(streamCCTVRow(id, true, 19, "encrypted-rtsp"))

	app := newTestApp()
	app.Post("/streams/:id/stop", StopStream)

	resp, err := app.Test(httptest.NewRequest("POST", "/streams/"+id.String()+"/stop", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	data := result["data"].(map[string]interface{})
	assert.Equal(t, "cam-"+id.String(), data["path"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetStreamStatus(t *testing.T) {
	_, mock := setupMockGORM(t)

	id := uuid.New()
	// ListPaths mengembalikan satu path siap (ready=true) dengan 2 penonton.
	injectMockMTX(t, mockMTXConfig{
		listConfigs: []utils.PathConfig{{Name: "cam-" + id.String()}},
		listPaths: []utils.PathInfo{
			{
				Name:      "cam-" + id.String(),
				Source:    "rtsp://...",
				Available: true,
				Online:    true,
				Readers:   []interface{}{"r1", "r2"},
			},
		},
	})

	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(streamCCTVRow(id, true, 19, "encrypted-rtsp"))

	app := newTestApp()
	app.Get("/streams/status", GetStreamStatus)

	resp, err := app.Test(httptest.NewRequest("GET", "/streams/status", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	items := result["data"].([]interface{})
	item := items[0].(map[string]interface{})
	assert.Equal(t, id.String(), item["cctv_id"])
	assert.Equal(t, true, item["stream_active"])
	assert.Equal(t, true, item["stream_ready"])
	assert.Equal(t, true, item["mediamtx_reachable"])
	assert.Equal(t, true, item["path_registered"])
	assert.Equal(t, true, item["source_ready"])
	assert.Equal(t, float64(2), item["viewers"])
	assert.Contains(t, item["whep_url"].(string), "/stream/cam-"+id.String()+"/whep")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetStreamURL(t *testing.T) {
	_, mock := setupMockGORM(t)

	id := uuid.New()
	injectMockMTX(t, mockMTXConfig{
		getInfo: &utils.PathInfo{
			Name:      "cam-" + id.String(),
			Source:    "rtsp://...",
			Available: true,
			Online:    true,
			Readers:   []interface{}{"r1"},
		},
		listConfigs: []utils.PathConfig{{Name: "cam-" + id.String()}},
	})

	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(streamCCTVRow(id, true, 19, "encrypted-rtsp"))

	app := newTestApp()
	app.Get("/streams/:id/url", GetStreamURL)

	resp, err := app.Test(httptest.NewRequest("GET", "/streams/"+id.String()+"/url", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	data := result["data"].(map[string]interface{})
	assert.Contains(t, data["whep_url"].(string), "/stream/cam-"+id.String()+"/whep")
	assert.Contains(t, data["hls_url"].(string), "/hls/cam-"+id.String()+"/index.m3u8")
	assert.Equal(t, true, data["stream_ready"])
	assert.Equal(t, true, data["mediamtx_reachable"])
	assert.Equal(t, true, data["path_registered"])
	assert.Equal(t, true, data["source_ready"])
	assert.Equal(t, float64(1), data["viewers"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestRegisterActiveStreams_RetryOnFailure memastikan registerStreamWithRetry
// mencoba ulang (retry) saat AddPath gagal sementara, lalu berhasil (Task 13.5).
func TestRegisterActiveStreams_RetryOnFailure(t *testing.T) {
	t.Setenv("AES_SECRET_KEY", "stream-test-aes-secret-key-1234567890")
	// Gagal 2 kali lalu berhasil.
	injectMockMTX(t, mockMTXConfig{addFailBeforeSuccess: 2})

	enc := encryptedTestPassword(t, "secret123")
	id := uuid.New()
	cctv := models.CCTV{
		Base:          models.Base{ID: id},
		KodeCCTV:      "CCTV-RETRY",
		Nama:          "Kamera Retry",
		ChannelPort:   19,
		RTSPPassword:  enc,
		StreamEnabled: true,
	}

	err := registerStreamWithRetry(cctv, 5)
	assert.NoError(t, err, "registerStreamWithRetry harus berhasil setelah retry")
}

// TestRegisterActiveStreams_GiveUpAfterMaxRetry memastikan registerStreamWithRetry
// menyerah (mengembalikan error) setelah seluruh percobaan gagal (Task 13.5).
func TestRegisterActiveStreams_GiveUpAfterMaxRetry(t *testing.T) {
	t.Setenv("AES_SECRET_KEY", "stream-test-aes-secret-key-1234567890")
	// Selalu gagal.
	injectMockMTX(t, mockMTXConfig{addErr: errors.New("boom")})

	enc := encryptedTestPassword(t, "secret123")
	id := uuid.New()
	cctv := models.CCTV{
		Base:          models.Base{ID: id},
		KodeCCTV:      "CCTV-GIVEUP",
		Nama:          "Kamera GiveUp",
		ChannelPort:   19,
		RTSPPassword:  enc,
		StreamEnabled: true,
	}

	err := registerStreamWithRetry(cctv, 2)
	assert.Error(t, err, "registerStreamWithRetry harus gagal setelah max retry")
}
