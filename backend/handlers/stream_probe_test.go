package handlers

import (
	"bufio"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"sigap-backend/models"
	"sigap-backend/utils"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// startTestRTSPListener membuka listener TCP pada 127.0.0.1:0 dan mengembalikan
// alamat serta fungsi cleanup. Opsi `respondRTSP` mengendalikan apakah server
// tiruan membalas baris RTSP 200 OK atau hanya membuka koneksi lalu menutupnya.
func startTestRTSPListener(t *testing.T, respondRTSP bool) (addr string, closeFn func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return // listener ditutup
			}
			go func(c net.Conn) {
				defer func() { _ = c.Close() }()
				if respondRTSP {
					// Baca request dari klien (OPTIONS) lalu balas 200 OK.
					scanner := bufio.NewScanner(c)
					for scanner.Scan() {
						line := scanner.Text()
						if line == "" {
							break
						}
					}
					_, _ = c.Write([]byte("RTSP/1.0 200 OK\r\nCSeq: 1\r\n\r\n"))
				}
			}(conn)
		}
	}()

	closeFn = func() { _ = ln.Close() }
	return ln.Addr().String(), closeFn
}

func TestProbeRTSPHost_Success(t *testing.T) {
	addr, closeFn := startTestRTSPListener(t, false)
	defer closeFn()

	host, portStr, ok := strings.Cut(addr, ":")
	assert.True(t, ok)
	port, err := strconv.Atoi(portStr)
	assert.NoError(t, err)

	assert.True(t, utils.ProbeRTSPHost(host, port))
}

func TestProbeRTSPHost_Unreachable(t *testing.T) {
	// Port 1 (tidak terpakai) pada localhost — koneksi harus gagal.
	assert.False(t, utils.ProbeRTSPHost("127.0.0.1", 1))
}

func TestProbeRTSPChannel_Success(t *testing.T) {
	addr, closeFn := startTestRTSPListener(t, true)
	defer closeFn()

	host, portStr, ok := strings.Cut(addr, ":")
	assert.True(t, ok)
	port, err := strconv.Atoi(portStr)
	assert.NoError(t, err)

	assert.True(t, utils.ProbeRTSPChannel(host, port, 1902))
}

func TestProbeRTSPChannel_Unreachable(t *testing.T) {
	assert.False(t, utils.ProbeRTSPChannel("127.0.0.1", 1, 1902))
}

// probeCCTVRow membangun sqlmock rows untuk satu CCTV stream_enabled dengan
// seluruh kolom tabel cctvs (termasuk field baru ping_latency_ms &
// stream_last_active_at). Parameter ipCamera, streamLive, streamCheckedAt
// dikendalikan agar mudah menguji skenario Metro-E vs GSM.
func probeCCTVRow(id uuid.UUID, ipCamera string, streamLive bool, streamCheckedAt interface{}, pingLatencyMs interface{}, streamLastActiveAt interface{}) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"kode_cctv", "id_channel", "nama", "lokasi_area",
		"longitude", "latitude", "jenis_cctv", "sumber_listrik",
		"tipe_koneksi", "indoor_outdoor", "pemilik", "status",
		"ip_nvr", "ip_camera", "keterangan", "qr_payload", "health_score",
		"channel_port", "rtsp_password", "stream_enabled", "stream_live", "stream_checked_at",
		"ping_latency_ms", "stream_last_active_at",
	}).AddRow(id, time.Now(), time.Now(), nil, "CCTV-PROBE", "CH-01", "Kamera Probe", "Area Utama",
		117.1, -0.5, "Dome", "PLN", "Fiber", "outdoor", "Pemkot", "aktif",
		"192.168.1.10", ipCamera, "Aman", "payload", 100, 19, "encrypted-rtsp", true,
		streamLive, streamCheckedAt, pingLatencyMs, streamLastActiveAt)
}

// startMTXListCounter membuat server MediaMTX tiruan yang hanya melayani
// GET /v3/paths/list dan menghitung berapa kali endpoint tersebut dipanggil.
// Mengembalikan klien MediaMTX, counter, dan fungsi cleanup.
func startMTXListCounter(t *testing.T, listPaths []utils.PathInfo) (*utils.MediaMTXClient, *int, func()) {
	t.Helper()
	calls := 0
	srv := newMockMediaMTXServer(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/v3/paths/list") {
			calls++
			_ = json.NewEncoder(w).Encode(listPaths)
			return
		}
		http.Error(w, "endpoint tidak dikenal", http.StatusNotFound)
	})
	client := &utils.MediaMTXClient{BaseURL: srv.URL, HTTPClient: srv.Client()}
	return client, &calls, srv.Close
}

// newMockMediaMTXServer membuka httptest server MediaMTX tiruan dengan handler
// khusus (dipakai pengujian probe cycle agar perilaku endpoint terkontrol).
func newMockMediaMTXServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

// injectStreamClient menyuntikkan klien MediaMTX ke streamClient global dan
// menjamin restore ke klien baru setelah pengujian selesai.
func injectStreamClient(t *testing.T, c *utils.MediaMTXClient) {
	t.Helper()
	setStreamClient(c)
	t.Cleanup(func() { setStreamClient(utils.NewMediaMTXClient()) })
}

func TestRunProbeCycle_MetroE(t *testing.T) {
	db, mock := setupMockGORM(t)

	// ListPaths dipanggil SEKALI per siklus (sumber status kamera GSM) —
	// tetapi keputusan live kamera Metro-E murni dari probe langsung ke IP,
	// bukan dari path MediaMTX. Server tiruan melayani list tanpa path.
	client, mtxCalls, closeSrv := startMTXListCounter(t, nil)
	defer closeSrv()
	injectStreamClient(t, client)

	// Metro-E: probe TCP ke 10.20.105.62 (jaringan OIKN) tidak terjangkau dari
	// environment pengujian → live=false, hanya stream_checked_at yang diupdate.
	id := uuid.New()
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WithArgs(true).
		WillReturnRows(probeCCTVRow(id, "10.20.105.62", false, nil, nil, nil))

	// Status tetap offline (false) → hanya UPDATE stream_checked_at.
	mock.ExpectExec(`^UPDATE "cctvs" SET "stream_checked_at"=\$1,"updated_at"=\$2 WHERE id = \$3 AND "cctvs"."deleted_at" IS NULL$`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), id.String()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	runRTSPProbeCycle(db)

	assert.NoError(t, mock.ExpectationsWereMet())
	// ListPaths hanya dipanggil satu kali di level siklus (bukan per kamera).
	assert.Equal(t, 1, *mtxCalls, "ListPaths dipanggil sekali per siklus probe")
}

func TestRunProbeCycle_GSM(t *testing.T) {
	db, mock := setupMockGORM(t)

	id := uuid.New()
	path := utils.BuildStreamPath(id.String())

	// MediaMTX melaporkan path kamera GSM Ready=true → live.
	client, _, closeSrv := startMTXListCounter(t, []utils.PathInfo{{Name: path, Ready: true}})
	defer closeSrv()
	injectStreamClient(t, client)

	// GSM: kamera tidak pernah diprobe langsung; status dari path MediaMTX.
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WithArgs(true).
		WillReturnRows(probeCCTVRow(id, "36.91.27.35", false, nil, nil, nil))

	// Status berubah false → true → UPDATE stream_live=true + last_active + checked.
	mock.ExpectExec(`^UPDATE "cctvs" SET "stream_checked_at"=\$1,"stream_last_active_at"=\$2,"stream_live"=\$3,"updated_at"=\$4 WHERE id = \$5 AND "cctvs"."deleted_at" IS NULL$`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), true, sqlmock.AnyArg(), id.String()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	runRTSPProbeCycle(db)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRunProbeCycle_GSMNotReady(t *testing.T) {
	db, mock := setupMockGORM(t)

	id := uuid.New()
	path := utils.BuildStreamPath(id.String())

	// MediaMTX melaporkan path kamera GSM Ready=false → live=false.
	client, _, closeSrv := startMTXListCounter(t, []utils.PathInfo{{Name: path, Ready: false}})
	defer closeSrv()
	injectStreamClient(t, client)

	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WithArgs(true).
		WillReturnRows(probeCCTVRow(id, "36.91.27.35", false, nil, nil, nil))

	// Status tetap offline → hanya UPDATE stream_checked_at.
	mock.ExpectExec(`^UPDATE "cctvs" SET "stream_checked_at"=\$1,"updated_at"=\$2 WHERE id = \$3 AND "cctvs"."deleted_at" IS NULL$`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), id.String()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	runRTSPProbeCycle(db)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRunProbeCycle_GSMReadyKeepsLive(t *testing.T) {
	db, mock := setupMockGORM(t)

	id := uuid.New()
	path := utils.BuildStreamPath(id.String())
	past := time.Now().Add(-30 * time.Minute)

	// Kamera GSM sudah live (stream_live=true) dan path masih Ready → siklus
	// "else if live" tetap meng-update (menyegarkan stream_last_active_at).
	client, _, closeSrv := startMTXListCounter(t, []utils.PathInfo{{Name: path, Ready: true}})
	defer closeSrv()
	injectStreamClient(t, client)

	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WithArgs(true).
		WillReturnRows(probeCCTVRow(id, "36.91.27.35", true, past, nil, past))

	mock.ExpectExec(`^UPDATE "cctvs" SET "stream_checked_at"=\$1,"stream_last_active_at"=\$2,"stream_live"=\$3,"updated_at"=\$4 WHERE id = \$5 AND "cctvs"."deleted_at" IS NULL$`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), true, sqlmock.AnyArg(), id.String()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	runRTSPProbeCycle(db)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestNetworkType memverifikasi klasifikasi tipe jaringan kamera.
func TestNetworkType(t *testing.T) {
	assert.Equal(t, "metro-e", networkType(models.CCTV{IPCamera: "10.20.105.62"}))
	assert.Equal(t, "gsm", networkType(models.CCTV{IPCamera: "36.91.27.35"}))
	assert.Equal(t, "gsm", networkType(models.CCTV{IPCamera: ""}))
}

// TestGetLiveStatus_NetworkFields memastikan respons GetLiveStatus memuat
// field baru network_type, ping_latency_ms & stream_last_active_at per CCTV.
func TestGetLiveStatus_NetworkFields(t *testing.T) {
	_, mock := setupMockGORM(t)

	id := uuid.New()
	past := time.Now().Add(-2 * time.Hour)
	latency := 42

	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(probeCCTVRow(id, "10.20.105.62", true, past, latency, past))

	app := newTestApp()
	app.Get("/streams/live-status", GetLiveStatus)

	resp, err := app.Test(httptest.NewRequest("GET", "/streams/live-status", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])

	items := result["data"].([]interface{})
	assert.Len(t, items, 1)
	item := items[0].(map[string]interface{})
	assert.Equal(t, "metro-e", item["network_type"])
	assert.Equal(t, float64(42), item["ping_latency_ms"])
	assert.NotNil(t, item["stream_last_active_at"])
	assert.NoError(t, mock.ExpectationsWereMet())
}
