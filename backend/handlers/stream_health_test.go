package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"sigap-backend/middleware"
	"sigap-backend/utils"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// capturedStreamEvent merekam satu event broadcast untuk verifikasi di test.
type capturedStreamEvent struct {
	event   string
	payload interface{}
}

// captureStreamEvents mengganti broadcastStreamEvent agar event stream dapat
// ditangkap dalam slice, lalu memulihkannya setelah test selesai.
func captureStreamEvents(t *testing.T) *[]capturedStreamEvent {
	t.Helper()
	events := &[]capturedStreamEvent{}
	original := broadcastStreamEvent
	broadcastStreamEvent = func(event string, payload interface{}) {
		*events = append(*events, capturedStreamEvent{event: event, payload: payload})
	}
	t.Cleanup(func() { broadcastStreamEvent = original })
	return events
}

// seedStreamDownState menyemai state streamLastReady & streamDownSince untuk
// mensimulasikan stream yang pernah up lalu down selama durasi tertentu.
func seedStreamDownState(path string, lastReady time.Time, downSince time.Time, notified bool) {
	streamHealthMu.Lock()
	defer streamHealthMu.Unlock()
	streamLastReady[path] = lastReady
	streamDownSince[path] = downSince
	if notified {
		streamDownNotified[path] = true
	}
}

func TestGetStreamHealth_NoStreams(t *testing.T) {
	resetStreamHealthState()
	_, mock := setupMockGORM(t)
	// ListPaths mengembalikan daftar kosong (tidak ada path aktif).
	injectMockMTX(t, mockMTXConfig{})

	id := uuid.New()
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(streamCCTVRow(id, false, 0, ""))

	app := newTestApp()
	app.Get("/streams/health", GetStreamHealth)

	resp, err := app.Test(httptest.NewRequest("GET", "/streams/health", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])

	items := result["data"].([]interface{})
	assert.Len(t, items, 1)
	item := items[0].(map[string]interface{})
	assert.Equal(t, id.String(), item["cctv_id"])
	assert.Equal(t, false, item["stream_enabled"])
	assert.Equal(t, false, item["stream_active"])
	assert.Equal(t, false, item["stream_ready"])
	assert.Equal(t, float64(0), item["viewers"])
	assert.Equal(t, false, item["down"])
	assert.Equal(t, float64(0), item["down_duration_seconds"])
	assert.Contains(t, item["whep_url"].(string), "/stream/cam-"+id.String()+"/whep")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetStreamHealth_Unauthorized(t *testing.T) {
	// Simulasikan rute dengan middleware RequireRole("admin") seperti pada
	// routes/stream.go. Peran teknisi harus ditolak dengan 403 sebelum handler.
	app := newTestApp()
	app.Get("/streams/health",
		withUserLocals(uuid.New().String(), "teknisi"),
		middleware.RequireRole("admin"),
		GetStreamHealth,
	)

	resp, err := app.Test(httptest.NewRequest("GET", "/streams/health", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}

func TestRunStreamHealthMonitor_NoMediaMTX(t *testing.T) {
	resetStreamHealthState()
	db, mock := setupMockGORM(t)
	// MediaMTX tidak dapat dihubungi → ListPaths error.
	injectMockMTX(t, mockMTXConfig{listErr: http.ErrHandlerTimeout})

	id := uuid.New()
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(streamCCTVRow(id, true, 19, "encrypted-rtsp"))

	// Monitor tidak boleh crash walau MediaMTX gagal; hanya log peringatan.
	assert.NotPanics(t, func() {
		runStreamHealthCycle(db)
	})
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRunStreamHealthOnce_DownAndRestored(t *testing.T) {
	resetStreamHealthState()
	db, mock := setupMockGORM(t)
	events := captureStreamEvents(t)

	id := uuid.New()
	path := utils.BuildStreamPath(id.String())

	// 1. Siklus DOWN: stream pernah up (streamLastReady di masa lalu) lalu down
	//    sudah melewati ambang 15 menit → STREAM_DOWN + health_score -10.
	now := time.Now()
	seedStreamDownState(path, now.Add(-1*time.Hour), now.Add(-16*time.Minute), false)

	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(streamCCTVRow(id, true, 19, "encrypted-rtsp", false, now.Add(-16*time.Minute)))
	// Decrement health_score 100 → 90.
	mock.ExpectExec(`^UPDATE "cctvs" SET "health_score"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	runStreamHealthOnce(db, map[string]bool{path: false})

	assert.NoError(t, mock.ExpectationsWereMet())
	assert.Len(t, *events, 1)
	assert.Equal(t, "STREAM_DOWN", (*events)[0].event)

	// Verifikasi state: streamDownSince tercatat & notified ditandai.
	streamHealthMu.Lock()
	assert.True(t, streamDownNotified[path])
	streamHealthMu.Unlock()

	// 2. Siklus RESTORE: path kini siap → STREAM_RESTORED tanpa update health.
	//    Setup mock baru untuk siklus kedua.
	db2, mock2 := setupMockGORM(t)
	mock2.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(streamCCTVRow(id, true, 19, "encrypted-rtsp", true, now))

	runStreamHealthOnce(db2, map[string]bool{path: true})

	assert.NoError(t, mock2.ExpectationsWereMet())
	assert.Len(t, *events, 2)
	assert.Equal(t, "STREAM_RESTORED", (*events)[1].event)

	// Verifikasi state dibersihkan setelah pulih.
	streamHealthMu.Lock()
	_, stillDown := streamDownSince[path]
	assert.False(t, stillDown)
	assert.False(t, streamDownNotified[path])
	_, hasLastReady := streamLastReady[path]
	assert.True(t, hasLastReady)
	streamHealthMu.Unlock()
}
