package handlers

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"sigap-backend/config"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func dailyDigestDateForTest() time.Time {
	return time.Date(2026, 8, 7, 0, 0, 0, 0, time.FixedZone("WITA", 8*60*60))
}

func expectDailyDigestAggregates(mock sqlmock.Sqlmock) {
	mock.ExpectQuery(`^SELECT`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"new_tasks", "completed_tasks", "failed_tasks", "overdue_tasks"}).
			AddRow(4, 3, 1, 2))
	mock.ExpectQuery(`^SELECT`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"field_checks", "attention_checks"}).AddRow(5, 1))
	mock.ExpectQuery(`^SELECT count\(\*\) FROM "audit_logs"`).
		WithArgs("CCTV", "UPDATE_CCTV", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
}

func expectDailyDigestNotification(mock sqlmock.Sqlmock, adminID uuid.UUID) {
	mock.ExpectQuery(`^SELECT "id" FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(adminID))
	mock.ExpectExec(`^INSERT INTO "notifications"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
}

func TestGenerateDailyDigestAt_CreatesDigest(t *testing.T) {
	_, mock := setupMockGORM(t)
	now := time.Date(2026, 8, 7, 6, 0, 0, 0, time.FixedZone("WITA", 8*60*60))
	adminID := uuid.New()

	mock.ExpectQuery(`^SELECT \* FROM "daily_digests"`).
		WillReturnError(gorm.ErrRecordNotFound)
	expectDailyDigestAggregates(mock)
	mock.ExpectExec(`^INSERT INTO "daily_digests"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	expectDailyDigestNotification(mock, adminID)

	digest, created, err := GenerateDailyDigestAt(config.DB, now)
	assert.NoError(t, err)
	assert.True(t, created)
	if assert.NotNil(t, digest) {
		assert.Equal(t, 4, digest.NewTasks)
		assert.Equal(t, 3, digest.CompletedTasks)
		assert.Equal(t, 1, digest.FailedTasks)
		assert.Equal(t, 2, digest.OverdueTasks)
		assert.Equal(t, 5, digest.FieldChecks)
		assert.Equal(t, 1, digest.AttentionChecks)
		assert.Equal(t, 2, digest.StatusChanges)
		assert.Contains(t, digest.Summary, "4 tugas baru")
		assert.True(t, dailyDigestDateForTest().Equal(digest.DigestDate))
	}
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGenerateDailyDigestAt_Idempotent(t *testing.T) {
	_, mock := setupMockGORM(t)
	digestID := uuid.New()
	digestDate := dailyDigestDateForTest()

	mock.ExpectQuery(`^SELECT \* FROM "daily_digests"`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "created_at", "updated_at", "deleted_at", "digest_date",
			"new_tasks", "completed_tasks", "failed_tasks", "overdue_tasks",
			"field_checks", "attention_checks", "status_changes", "summary",
		}).AddRow(digestID, digestDate, digestDate, nil, digestDate, 1, 2, 0, 1, 3, 0, 2, "Digest lama"))

	digest, created, err := GenerateDailyDigestAt(config.DB, time.Date(2026, 8, 7, 9, 30, 0, 0, digestDate.Location()))
	assert.NoError(t, err)
	assert.False(t, created)
	if assert.NotNil(t, digest) {
		assert.Equal(t, digestID, digest.ID)
		assert.Equal(t, "Digest lama", digest.Summary)
	}
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNextDailyDigestRun_UsesWITA(t *testing.T) {
	location := time.FixedZone("WITA", 8*60*60)
	beforeSix := time.Date(2026, 8, 7, 5, 30, 0, 0, location)
	afterSix := time.Date(2026, 8, 7, 7, 0, 0, 0, location)

	assert.True(t, time.Date(2026, 8, 7, 6, 0, 0, 0, location).Equal(nextDailyDigestRun(beforeSix)))
	assert.True(t, time.Date(2026, 8, 8, 6, 0, 0, 0, location).Equal(nextDailyDigestRun(afterSix)))
}

func TestGetDailyDigests_AdminPaginatedNewestFirst(t *testing.T) {
	_, mock := setupMockGORM(t)
	newer := dailyDigestDateForTest()
	older := newer.AddDate(0, 0, -1)

	mock.ExpectQuery(`^SELECT count\(\*\) FROM "daily_digests"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	mock.ExpectQuery(`^SELECT \* FROM "daily_digests"`).
		WillReturnRows(dailyDigestRowsForTest(newer, older))

	app := newTestApp()
	app.Get("/dashboard/digest", withUserLocals(uuid.New().String(), "admin"), GetDailyDigests)

	resp, err := app.Test(httptest.NewRequest("GET", "/dashboard/digest?page=1&limit=10", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	data := result["data"].(map[string]interface{})
	rows := data["rows"].([]interface{})
	assert.Len(t, rows, 2)
	assert.Equal(t, float64(2), rows[0].(map[string]interface{})["new_tasks"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetDailyDigests_NonAdminForbidden(t *testing.T) {
	app := newTestApp()
	app.Get("/dashboard/digest", withUserLocals(uuid.New().String(), "teknisi"), GetDailyDigests)

	resp, err := app.Test(httptest.NewRequest("GET", "/dashboard/digest", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
}

func dailyDigestRowsForTest(newer, older time.Time) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at", "digest_date",
		"new_tasks", "completed_tasks", "failed_tasks", "overdue_tasks",
		"field_checks", "attention_checks", "status_changes", "summary",
	}).
		AddRow(uuid.New(), newer, newer, nil, newer, 2, 3, 1, 0, 4, 1, 2, "Digest terbaru").
		AddRow(uuid.New(), older, older, nil, older, 1, 1, 0, 2, 2, 0, 1, "Digest sebelumnya")
}
