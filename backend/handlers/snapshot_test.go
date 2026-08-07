package handlers

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// TestGetCCTVTimeline_CCTVNotFound memastikan endpoint timeline mengembalikan
// 404 ketika CCTV tidak ditemukan di database.
func TestGetCCTVTimeline_CCTVNotFound(t *testing.T) {
	_, mock := setupMockGORM(t)
	id := uuid.New()

	// Query pertama memuat CCTV (SELECT id ...).
	mock.ExpectQuery(`^SELECT .* FROM "cctvs"`).
		WillReturnError(gorm.ErrRecordNotFound)

	app := newTestApp()
	app.Get("/cctv/:id/timeline", GetCCTVTimeline)

	resp, err := app.Test(httptest.NewRequest("GET", "/cctv/"+id.String()+"/timeline", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "Data CCTV tidak ditemukan", result["message"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetCCTVTimeline_Success memastikan endpoint timeline mengembalikan daftar
// snapshot terpaginasi beserta snapshot_url untuk setiap baris.
func TestGetCCTVTimeline_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	cctvID := uuid.New()
	snapID := uuid.New()
	now := time.Now()

	// Query memuat CCTV.
	cctvCols := []string{"id"}
	cctvRows := sqlmock.NewRows(cctvCols).AddRow(cctvID)
	mock.ExpectQuery(`^SELECT "id" FROM "cctvs"`).
		WillReturnRows(cctvRows)

	// Query count snapshot.
	mock.ExpectQuery(`^SELECT count\(\*\) FROM "cctv_snapshots"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	// Query data snapshot.
	snapRows := sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"cctv_id", "event_type", "image_path", "notes",
	}).AddRow(snapID, now, now, nil, cctvID, "periodic_snapshot", "uploads/snapshots/"+cctvID.String()+"/20260101_000000.jpg", "")
	mock.ExpectQuery(`^SELECT \* FROM "cctv_snapshots"`).
		WillReturnRows(snapRows)

	app := newTestApp()
	app.Get("/cctv/:id/timeline", GetCCTVTimeline)

	resp, err := app.Test(httptest.NewRequest("GET", "/cctv/"+cctvID.String()+"/timeline", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])

	data := result["data"].(map[string]interface{})
	pagination := data["pagination"].(map[string]interface{})
	assert.Equal(t, float64(1), pagination["total_rows"])

	rows := data["rows"].([]interface{})
	assert.Len(t, rows, 1)
	row := rows[0].(map[string]interface{})
	assert.Equal(t, "periodic_snapshot", row["event_type"])
	assert.Equal(t, "/api/cctv/"+cctvID.String()+"/snapshot", row["snapshot_url"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetCCTVSnapshot_NoSnapshot memastikan endpoint snapshot mengembalikan
// 404 ketika tidak ada snapshot tersimpan untuk CCTV tersebut.
func TestGetCCTVSnapshot_NoSnapshot(t *testing.T) {
	_, mock := setupMockGORM(t)
	cctvID := uuid.New()

	// Query memuat CCTV.
	rows, _ := cctvMockRowWithStatus("aktif")
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(rows)

	// Query snapshot terbaru — tidak ada data.
	mock.ExpectQuery(`^SELECT \* FROM "cctv_snapshots"`).
		WillReturnError(gorm.ErrRecordNotFound)

	app := newTestApp()
	app.Get("/cctv/:id/snapshot", GetCCTVSnapshot)

	resp, err := app.Test(httptest.NewRequest("GET", "/cctv/"+cctvID.String()+"/snapshot", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "Snapshot CCTV tidak ditemukan", result["message"])
	assert.NoError(t, mock.ExpectationsWereMet())
}
