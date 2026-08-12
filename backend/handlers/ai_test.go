package handlers

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// setAISecretEnv mengatur environment variable AI_SECRET untuk pengujian dan
// mengembalikannya ke nilai semula setelah test selesai (t.Setenv otomatis).
func setAISecretEnv(t *testing.T) {
	t.Helper()
	t.Setenv("AI_SECRET", "test-ai-secret")
}

// aiSessionRows membangun baris tabel ai_detection_sessions untuk mock sqlmock.
func aiSessionRows(id, cctvID uuid.UUID, sessionDate time.Time, unique int, tracking []byte) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "cctv_id", "session_date", "unique_person_count", "last_tracking_ids",
		"created_at", "updated_at",
	}).AddRow(id, cctvID, sessionDate, unique, tracking, time.Now(), time.Now())
}

// TestReceiveDetectionEvent_ValidSecret menguji penerimaan event deteksi dengan
// X-AI-Secret yang benar: sesi baru di-upsert + event disimpan → 200.
func TestReceiveDetectionEvent_ValidSecret(t *testing.T) {
	_, mock := setupMockGORM(t)
	setAISecretEnv(t)
	cctvID := uuid.New()

	// Query sesi → tidak ditemukan (gorm.ErrRecordNotFound).
	mock.ExpectQuery(`SELECT \* FROM "ai_detection_sessions"`).
		WillReturnError(gorm.ErrRecordNotFound)
	// Insert sesi baru.
	mock.ExpectExec(`^INSERT INTO "ai_detection_sessions"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	// Insert event deteksi (person_count > 0).
	mock.ExpectExec(`^INSERT INTO "ai_detection_events"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	app := newTestApp()
	app.Post("/ai/detection-event", ReceiveDetectionEvent)

	body := `{"cctv_id":"` + cctvID.String() + `","bboxes":[{"x1":0.1,"y1":0.2,"x2":0.3,"y2":0.4}],"person_count":1,"unique_count":1,"confidence_avg":0.85}`
	req := httptest.NewRequest("POST", "/ai/detection-event", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-AI-Secret", "test-ai-secret")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestReceiveDetectionEvent_InvalidSecret memastikan secret salah/kosong → 401.
func TestReceiveDetectionEvent_InvalidSecret(t *testing.T) {
	_, mock := setupMockGORM(t)
	setAISecretEnv(t)

	app := newTestApp()
	app.Post("/ai/detection-event", ReceiveDetectionEvent)

	req := httptest.NewRequest("POST", "/ai/detection-event",
		strings.NewReader(`{"cctv_id":"00000000-0000-0000-0000-000000000000","person_count":1}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-AI-Secret", "salah-secret")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestReceiveDetectionEvent_InvalidUUID memastikan cctv_id bukan UUID → 400.
func TestReceiveDetectionEvent_InvalidUUID(t *testing.T) {
	_, mock := setupMockGORM(t)
	setAISecretEnv(t)

	app := newTestApp()
	app.Post("/ai/detection-event", ReceiveDetectionEvent)

	req := httptest.NewRequest("POST", "/ai/detection-event",
		strings.NewReader(`{"cctv_id":"bukan-uuid","person_count":1}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-AI-Secret", "test-ai-secret")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestReceiveDetectionEvent_ExistingSession menguji sesi yang sudah ada:
// hanya Update (Save) + insert event → 200.
func TestReceiveDetectionEvent_ExistingSession(t *testing.T) {
	_, mock := setupMockGORM(t)
	setAISecretEnv(t)
	cctvID := uuid.New()
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Query sesi → ditemukan.
	mock.ExpectQuery(`SELECT \* FROM "ai_detection_sessions"`).
		WillReturnRows(aiSessionRows(uuid.New(), cctvID, today, 2, []byte(`["10"]`)))
	// GORM Save pada baris existing → UPDATE ... (tanpa RETURNING untuk model
	// tanpa kolom dengan nilai default yang perlu dibaca kembali).
	mock.ExpectExec(`^UPDATE "ai_detection_sessions" SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	// Insert event deteksi.
	mock.ExpectExec(`^INSERT INTO "ai_detection_events"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	app := newTestApp()
	app.Post("/ai/detection-event", ReceiveDetectionEvent)

	body := `{"cctv_id":"` + cctvID.String() + `","person_count":1,"unique_count":3,"confidence_avg":0.7}`
	req := httptest.NewRequest("POST", "/ai/detection-event", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-AI-Secret", "test-ai-secret")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetAICounts_Empty menguji ringkasan saat belum ada sesi hari ini → data [].
func TestGetAICounts_Empty(t *testing.T) {
	_, mock := setupMockGORM(t)

	mock.ExpectQuery(`SELECT \* FROM "ai_detection_sessions"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	app := newTestApp()
	app.Get("/ai/counts", withUserLocals(uuid.New().String(), "admin"), GetAICounts)

	req := httptest.NewRequest("GET", "/ai/counts", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result struct {
		Status string        `json:"status"`
		Data   []interface{} `json:"data"`
	}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result.Status)
	assert.Empty(t, result.Data)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetAICounts_WithData menguji ringkasan dengan 1 sesi + data CCTV.
func TestGetAICounts_WithData(t *testing.T) {
	_, mock := setupMockGORM(t)
	cctvID := uuid.New()
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	mock.ExpectQuery(`SELECT \* FROM "ai_detection_sessions"`).
		WillReturnRows(aiSessionRows(uuid.New(), cctvID, today, 5, nil))
	// Query CCTV untuk melengkapi nama & kode. GORM menambahkan klausa
	// WHERE id = $1 AND deleted_at IS NULL ORDER BY id LIMIT 1.
	mock.ExpectQuery(`SELECT id,nama,kode_cctv FROM "cctvs" WHERE`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "nama", "kode_cctv"}).
			AddRow(cctvID, "Kamera Gerbang", "CCTV-001"))

	app := newTestApp()
	app.Get("/ai/counts", withUserLocals(uuid.New().String(), "admin"), GetAICounts)

	req := httptest.NewRequest("GET", "/ai/counts", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result struct {
		Status string `json:"status"`
		Data   []struct {
			CCTVID       string `json:"cctv_id"`
			Nama         string `json:"nama"`
			KodeCCTV     string `json:"kode_cctv"`
			UniqueToday  int    `json:"unique_today"`
		} `json:"data"`
	}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result.Status)
	assert.Len(t, result.Data, 1)
	assert.Equal(t, "Kamera Gerbang", result.Data[0].Nama)
	assert.Equal(t, "CCTV-001", result.Data[0].KodeCCTV)
	assert.Equal(t, 5, result.Data[0].UniqueToday)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetAICountByCCTV menguji detail hitungan per kamera (sesi ada → 200).
func TestGetAICountByCCTV(t *testing.T) {
	_, mock := setupMockGORM(t)
	cctvID := uuid.New()
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	mock.ExpectQuery(`SELECT \* FROM "ai_detection_sessions"`).
		WillReturnRows(aiSessionRows(uuid.New(), cctvID, today, 7, nil))

	app := newTestApp()
	app.Get("/ai/counts/:id", withUserLocals(uuid.New().String(), "admin"), GetAICountByCCTV)

	req := httptest.NewRequest("GET", "/ai/counts/"+cctvID.String(), nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result struct {
		Status string `json:"status"`
		Data   struct {
			UniqueToday int `json:"unique_today"`
		} `json:"data"`
	}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result.Status)
	assert.Equal(t, 7, result.Data.UniqueToday)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetAICountByCCTV_InvalidID memastikan ID bukan UUID → 400.
func TestGetAICountByCCTV_InvalidID(t *testing.T) {
	_, mock := setupMockGORM(t)

	app := newTestApp()
	app.Get("/ai/counts/:id", withUserLocals(uuid.New().String(), "admin"), GetAICountByCCTV)

	req := httptest.NewRequest("GET", "/ai/counts/bukan-uuid", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetAICountTrend menguji tren hitungan harian per kamera (days=7):
// query sesi dengan filter cctv_id + session_date >= (N hari lalu) diurutkan
// ascending → 200 dengan data berisi {date, unique_count}.
func TestGetAICountTrend(t *testing.T) {
	_, mock := setupMockGORM(t)
	cctvID := uuid.New()
	now := time.Now()

	// GORM: SELECT * FROM "ai_detection_sessions" WHERE cctv_id = $1 AND
	// session_date >= $2 ORDER BY session_date ASC (tanpa soft delete — model
	// AI tidak me-embed Base).
	mock.ExpectQuery(`SELECT \* FROM "ai_detection_sessions" WHERE cctv_id = \$1 AND session_date >= \$2 ORDER BY session_date ASC`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "cctv_id", "session_date", "unique_person_count", "last_tracking_ids",
			"created_at", "updated_at",
		}).
			AddRow(uuid.New(), cctvID, now.AddDate(0, 0, -2), 3, nil, now, now).
			AddRow(uuid.New(), cctvID, now.AddDate(0, 0, -1), 5, nil, now, now).
			AddRow(uuid.New(), cctvID, now, 7, nil, now, now))

	app := newTestApp()
	app.Get("/ai/counts/:id/trend", withUserLocals(uuid.New().String(), "admin"), GetAICountTrend)

	req := httptest.NewRequest("GET", "/ai/counts/"+cctvID.String()+"/trend?days=7", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result struct {
		Status string `json:"status"`
		Data   []struct {
			Date        string `json:"date"`
			UniqueCount int    `json:"unique_count"`
		} `json:"data"`
	}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result.Status)
	assert.Len(t, result.Data, 3)
	// Urutan ascending berdasarkan session_date.
	assert.Equal(t, 3, result.Data[0].UniqueCount)
	assert.Equal(t, 5, result.Data[1].UniqueCount)
	assert.Equal(t, 7, result.Data[2].UniqueCount)
	assert.NotEmpty(t, result.Data[0].Date)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetAICountTrend_InvalidID memastikan ID bukan UUID → 400 (tanpa query DB).
func TestGetAICountTrend_InvalidID(t *testing.T) {
	_, mock := setupMockGORM(t)

	app := newTestApp()
	app.Get("/ai/counts/:id/trend", withUserLocals(uuid.New().String(), "admin"), GetAICountTrend)

	req := httptest.NewRequest("GET", "/ai/counts/bukan-uuid/trend?days=7", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}
