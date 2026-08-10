package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func sqlNullTime(value time.Time) (result sql.NullTime) {
	result.Time = value
	result.Valid = true
	return result
}

func TestCalculateHealthScore_DefaultWithoutHistory(t *testing.T) {
	now := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	result := calculateHealthScore(healthScoreMetrics{}, now)

	assert.Equal(t, 98, result.HealthScore)
	assert.Equal(t, 91, result.Breakdown.Age)
	assert.Equal(t, now, result.UpdatedAt)
}

func TestCalculateHealthScore_AllFactors(t *testing.T) {
	now := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	lastCompleted := now.Add(-31 * 24 * time.Hour)
	metrics := healthScoreMetrics{
		Failed:          1,
		Attention:       1,
		Overdue:         1,
		NotOKItems:      3,
		LastCompletedAt: sqlNullTime(lastCompleted),
	}

	result := calculateHealthScore(metrics, now)

	// Penalti = 3 + 2,5 + 2 + 0,75 + 0,5 = 8,75; dibulatkan menjadi 91.
	assert.Equal(t, 91, result.HealthScore)
	assert.Equal(t, HealthScoreBreakdown{Failed: 1, Attention: 1, Overdue: 1, Age: 31, NotOKItems: 3}, result.Breakdown)
}

func TestCalculateHealthScore_AgeThresholds(t *testing.T) {
	now := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		age  int
		want int
	}{
		{name: "30 hari belum kena penalti", age: 30, want: 100},
		{name: "31 hari", age: 31, want: 99},
		{name: "61 hari", age: 61, want: 99},
		{name: "91 hari", age: 91, want: 98},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lastCompleted := now.Add(-time.Duration(tt.age) * 24 * time.Hour)
			result := calculateHealthScore(healthScoreMetrics{LastCompletedAt: sqlNullTime(lastCompleted)}, now)
			assert.Equal(t, tt.want, result.HealthScore)
		})
	}
}

func TestCalculateHealthScore_ClampedToZero(t *testing.T) {
	now := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	result := calculateHealthScore(healthScoreMetrics{
		Failed:          100,
		Attention:       100,
		Overdue:         100,
		NotOKItems:      300,
		LastCompletedAt: sqlNullTime(now.Add(-91 * 24 * time.Hour)),
	}, now)

	assert.Equal(t, 0, result.HealthScore)
}

func TestGetCCTVHealthScore_InvalidUUID(t *testing.T) {
	app := newTestApp()
	app.Get("/cctv/:id/health-score", GetCCTVHealthScore)

	response, err := app.Test(httptest.NewRequest("GET", "/cctv/not-a-uuid/health-score", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, response.StatusCode)
}

func TestGetCCTVHealthScore_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	cctvID := uuid.New()

	mock.ExpectQuery(`^SELECT "id" FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(cctvID))
	mock.ExpectQuery(`^SELECT\s+\(SELECT COUNT\(\*\) FROM maintenance_logs`).
		WillReturnRows(sqlmock.NewRows([]string{
			"failed", "attention", "overdue", "not_ok_items", "last_completed_at",
		}).AddRow(1, 0, 0, 0, nil))
	mock.ExpectExec(`^UPDATE "cctvs" SET "health_score"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	app := newTestApp()
	app.Get("/cctv/:id/health-score", withUserLocals(uuid.New().String(), "teknisi"), GetCCTVHealthScore)

	response, err := app.Test(httptest.NewRequest("GET", "/cctv/"+cctvID.String()+"/health-score", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, response.StatusCode)

	var payload map[string]interface{}
	assert.NoError(t, json.NewDecoder(response.Body).Decode(&payload))
	assert.Equal(t, "success", payload["status"])
	data := payload["data"].(map[string]interface{})
	assert.Equal(t, float64(95), data["health_score"])
	assert.NotNil(t, data["breakdown"])
	assert.NotEmpty(t, data["updated_at"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCalculateAllCCTVHealthScoresAt_UpdatesEveryCCTV(t *testing.T) {
	db, mock := setupMockGORM(t)
	cctvID := uuid.New()

	mock.ExpectQuery(`^SELECT "id" FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(cctvID))
	expectHealthScoreRecalculation(mock)

	CalculateAllCCTVHealthScoresAt(db, time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC))

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetCCTVHealthHistory memastikan endpoint histori skor kesehatan mengembalikan
// data terurut naik (Task 13.8).
func TestGetCCTVHealthHistory(t *testing.T) {
	_, mock := setupMockGORM(t)
	cctvID := uuid.New()

	rows := sqlmock.NewRows([]string{"id", "cctv_id", "health_score", "calculated_at"}).
		AddRow(uuid.New(), cctvID, 90, time.Now().AddDate(0, 0, -3)).
		AddRow(uuid.New(), cctvID, 85, time.Now().AddDate(0, 0, -2)).
		AddRow(uuid.New(), cctvID, 78, time.Now().AddDate(0, 0, -1))
	mock.ExpectQuery(`SELECT \* FROM "cctv_health_histories"`).
		WillReturnRows(rows)

	app := newTestApp()
	app.Get("/cctv/:id/health-history", withUserLocals(uuid.New().String(), "teknisi"), GetCCTVHealthHistory)

	resp, err := app.Test(httptest.NewRequest("GET", "/cctv/"+cctvID.String()+"/health-history?range=30d", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var payload map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&payload))
	assert.Equal(t, "success", payload["status"])
	data := payload["data"].([]interface{})
	assert.Len(t, data, 3)
	first := data[0].(map[string]interface{})
	assert.Equal(t, float64(90), first["health_score"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetCCTVHealthHistory_InvalidUUID memastikan 400 untuk UUID tidak valid.
func TestGetCCTVHealthHistory_InvalidUUID(t *testing.T) {
	_, mock := setupMockGORM(t)
	app := newTestApp()
	app.Get("/cctv/:id/health-history", withUserLocals(uuid.New().String(), "teknisi"), GetCCTVHealthHistory)

	resp, err := app.Test(httptest.NewRequest("GET", "/cctv/not-a-uuid/health-history", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}
