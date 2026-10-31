package handlers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"sigap-backend/middleware"
	"sigap-backend/utils"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func streamMediaCCTVRow(id uuid.UUID, siteID uuid.UUID, streamEnabled bool) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"site_id", "kode_cctv", "id_channel", "nama", "lokasi_area",
		"longitude", "latitude", "jenis_cctv", "sumber_listrik",
		"tipe_koneksi", "indoor_outdoor", "pemilik", "status",
		"ip_nvr", "ip_camera", "keterangan", "qr_payload", "health_score",
		"channel_port", "rtsp_password", "stream_enabled",
	}).AddRow(
		id, time.Now(), time.Now(), nil,
		siteID, "CCTV-H01", "CH-01", "Kamera H01", "Area Inti",
		117.1, -0.5, "Dome", "PLN", "Fiber", "outdoor", "OIKN", "aktif",
		"10.20.105.2", "10.20.105.11", "Aman", "payload", 100,
		1, "enc-pwd", streamEnabled,
	)
}

func setupStreamMediaProtectedApp() *fiber.App {
	app := newTestApp()
	mediaChain := []fiber.Handler{
		middleware.Protected(),
		middleware.RequireRole("admin"),
	}
	app.Get("/api/streams/auth", append(mediaChain, AuthorizeStreamAccess)...)
	app.All("/stream/*", append(mediaChain, ProxyWHEPStream)...)
	app.All("/hls/*", append(mediaChain, ProxyHLSStream)...)
	return app
}

// TestStreamMedia_RejectsAnonymousAndTechnician memastikan request video
// anonim (401) dan teknisi (403) ditolak pada jalur WHEP (/stream/*),
// HLS (/hls/*), maupun subrequest autentikasi Nginx (/api/streams/auth).
func TestStreamMedia_RejectsAnonymousAndTechnician(t *testing.T) {
	_, mock := setupMockGORM(t)
	t.Setenv("JWT_SECRET", "h01-stream-auth-test-secret-key-32bytes!")

	camID := uuid.New()
	whepPath := "/stream/cam-" + camID.String() + "/whep"
	hlsPath := "/hls/cam-" + camID.String() + "/index.m3u8"

	upstreamCalled := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	setStreamClient(&utils.MediaMTXClient{
		BaseURL:    srv.URL,
		HLSURL:     srv.URL,
		WebRTCURL:  srv.URL,
		HTTPClient: srv.Client(),
	})
	t.Cleanup(func() { setStreamClient(utils.NewMediaMTXClient()) })

	app := setupStreamMediaProtectedApp()

	// 1. Anonim ditolak 401 pada WHEP, HLS, dan /api/streams/auth
	for _, tc := range []struct {
		method string
		path   string
		header map[string]string
	}{
		{method: "POST", path: whepPath},
		{method: "GET", path: hlsPath},
		{method: "GET", path: "/api/streams/auth", header: map[string]string{"X-Original-URI": whepPath}},
	} {
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader("v=0"))
		for k, v := range tc.header {
			req.Header.Set(k, v)
		}
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode, "anonim harus ditolak 401 pada %s %s", tc.method, tc.path)
	}

	// 2. Teknisi dengan token valid ditolak 403 pada WHEP, HLS, dan /api/streams/auth
	techID := uuid.New()
	techToken, err := utils.GenerateToken(techID.String(), "teknisi", 1)
	require.NoError(t, err)

	for _, tc := range []struct {
		method string
		path   string
		header map[string]string
	}{
		{method: "POST", path: whepPath},
		{method: "GET", path: hlsPath},
		{method: "GET", path: "/api/streams/auth", header: map[string]string{"X-Original-URI": hlsPath}},
	} {
		mock.ExpectQuery(`^SELECT "token_version" FROM "users"`).
			WillReturnRows(sqlmock.NewRows([]string{"token_version"}).AddRow(1))

		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader("v=0"))
		req.Header.Set("Authorization", "Bearer "+techToken)
		for k, v := range tc.header {
			req.Header.Set(k, v)
		}
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusForbidden, resp.StatusCode, "teknisi harus ditolak 403 pada %s %s", tc.method, tc.path)
	}

	assert.False(t, upstreamCalled, "MediaMTX tidak boleh dipanggil ketika otorisasi gagal")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestStreamMedia_AllowsValidAdminAndRejectsAfterLogout memastikan Admin
// dengan sesi valid dapat menonton WHEP & HLS, namun segera ditolak 401
// begitu sesi dicabut (logout).
func TestStreamMedia_AllowsValidAdminAndRejectsAfterLogout(t *testing.T) {
	_, mock := setupMockGORM(t)
	t.Setenv("JWT_SECRET", "h01-stream-auth-test-secret-key-32bytes!")

	camID := uuid.New()
	siteID := uuid.New()
	streamName := "cam-" + camID.String()
	whepPath := "/stream/" + streamName + "/whep"
	hlsPath := "/hls/" + streamName + "/index.m3u8"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/"+streamName+"/whep":
			w.Header().Set("Content-Type", "application/sdp")
			w.Header().Set("Location", "/"+streamName+"/whep/session-abc")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte("v=0\r\ns=MediaMTX\r\n"))
		case r.Method == http.MethodGet && r.URL.Path == "/"+streamName+"/index.m3u8":
			w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("#EXTM3U\n"))
		default:
			http.Error(w, "unexpected upstream request: "+r.Method+" "+r.URL.Path, http.StatusNotFound)
		}
	}))
	defer srv.Close()
	setStreamClient(&utils.MediaMTXClient{
		BaseURL:    srv.URL,
		HLSURL:     srv.URL,
		WebRTCURL:  srv.URL,
		HTTPClient: srv.Client(),
	})
	t.Cleanup(func() { setStreamClient(utils.NewMediaMTXClient()) })

	app := setupStreamMediaProtectedApp()

	adminID := uuid.New()
	adminToken, err := utils.GenerateToken(adminID.String(), "admin", 1)
	require.NoError(t, err)

	// 1. Subrequest auth_request (/api/streams/auth) sukses untuk Admin
	mock.ExpectQuery(`^SELECT "token_version" FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"token_version"}).AddRow(1))
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(streamMediaCCTVRow(camID, siteID, true))

	authReq := httptest.NewRequest("GET", "/api/streams/auth", nil)
	authReq.Header.Set("Authorization", "Bearer "+adminToken)
	authReq.Header.Set("X-Original-URI", whepPath)
	authResp, err := app.Test(authReq)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, authResp.StatusCode)
	assert.Equal(t, streamName, authResp.Header.Get("X-Stream-Path"))

	// 2. WHEP signaling sukses untuk Admin
	mock.ExpectQuery(`^SELECT "token_version" FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"token_version"}).AddRow(1))
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(streamMediaCCTVRow(camID, siteID, true))

	whepReq := httptest.NewRequest("POST", whepPath, strings.NewReader("v=0\r\n"))
	whepReq.Header.Set("Authorization", "Bearer "+adminToken)
	whepReq.Header.Set("Content-Type", "application/sdp")
	whepResp, err := app.Test(whepReq)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, whepResp.StatusCode)
	assert.Equal(t, "/stream/"+streamName+"/whep/session-abc", whepResp.Header.Get("Location"))
	whepBody, _ := io.ReadAll(whepResp.Body)
	assert.Contains(t, string(whepBody), "s=MediaMTX")

	// 3. HLS playlist sukses untuk Admin
	mock.ExpectQuery(`^SELECT "token_version" FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"token_version"}).AddRow(1))
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(streamMediaCCTVRow(camID, siteID, true))

	hlsReq := httptest.NewRequest("GET", hlsPath, nil)
	hlsReq.Header.Set("Authorization", "Bearer "+adminToken)
	hlsResp, err := app.Test(hlsReq)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, hlsResp.StatusCode)
	hlsBody, _ := io.ReadAll(hlsResp.Body)
	assert.Contains(t, string(hlsBody), "#EXTM3U")

	// 4. Sesi dicabut (logout) -> WHEP, HLS, dan /api/streams/auth langsung ditolak 401
	utils.RevokeToken(adminToken, time.Now().Add(time.Hour))

	for _, tc := range []struct {
		method string
		path   string
		header map[string]string
	}{
		{method: "POST", path: whepPath},
		{method: "GET", path: hlsPath},
		{method: "GET", path: "/api/streams/auth", header: map[string]string{"X-Original-URI": whepPath}},
	} {
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader("v=0"))
		req.Header.Set("Authorization", "Bearer "+adminToken)
		for k, v := range tc.header {
			req.Header.Set(k, v)
		}
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode, "sesi yang sudah logout harus ditolak 401 pada %s %s", tc.method, tc.path)
	}

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestStreamMedia_EnforcesSiteAndStreamEnabled memastikan kamera yang belum
// mengaktifkan stream atau berada di luar site aktif ditolak 403, serta path
// tidak sah ditolak 400.
func TestStreamMedia_EnforcesSiteAndStreamEnabled(t *testing.T) {
	_, mock := setupMockGORM(t)

	camID := uuid.New()
	siteA := uuid.New()
	siteB := uuid.New()
	whepPath := "/stream/cam-" + camID.String() + "/whep"

	app := newTestApp()
	app.Get("/api/streams/auth", withUserSiteLocals(uuid.NewString(), "admin", siteA.String()), AuthorizeStreamAccess)

	// 1. Kamera milik siteB ditolak 403 saat konteks site aktif adalah siteA
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(streamMediaCCTVRow(camID, siteB, true))

	reqOtherSite := httptest.NewRequest("GET", "/api/streams/auth", nil)
	reqOtherSite.Header.Set("X-Original-URI", whepPath)
	respOtherSite, err := app.Test(reqOtherSite)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, respOtherSite.StatusCode)

	// 2. Kamera dengan stream_enabled=false ditolak 403
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(streamMediaCCTVRow(camID, siteA, false))

	reqDisabled := httptest.NewRequest("GET", "/api/streams/auth", nil)
	reqDisabled.Header.Set("X-Original-URI", whepPath)
	respDisabled, err := app.Test(reqDisabled)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, respDisabled.StatusCode)

	// 3. Path traversal / bukan format cam-<uuid> ditolak 400
	reqInvalid := httptest.NewRequest("GET", "/api/streams/auth", nil)
	reqInvalid.Header.Set("X-Original-URI", "/stream/../v3/config/global/get")
	respInvalid, err := app.Test(reqInvalid)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, respInvalid.StatusCode)

	assert.NoError(t, mock.ExpectationsWereMet())
}
