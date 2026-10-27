package handlers

import (
	"errors"
	"testing"
	"time"

	"sigap-backend/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

const lastCCTVStreamLogQuery = `^SELECT "event_type","created_at" FROM "cctv_stream_logs" WHERE cctv_id = \$1 ORDER BY created_at DESC LIMIT \$2$`

func expectLastCCTVStreamLog(mock sqlmock.Sqlmock, cctvID uuid.UUID, eventType string, createdAt time.Time) {
	mock.ExpectQuery(lastCCTVStreamLogQuery).
		WithArgs(cctvID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"event_type", "created_at"}).
			AddRow(eventType, createdAt))
}

func expectNoPreviousCCTVStreamLog(mock sqlmock.Sqlmock, cctvID uuid.UUID) {
	mock.ExpectQuery(lastCCTVStreamLogQuery).
		WithArgs(cctvID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"event_type", "created_at"}))
}

func expectCCTVStreamLogInsert(mock sqlmock.Sqlmock) {
	mock.ExpectQuery(`^INSERT INTO "cctv_stream_logs"`).
		WillReturnRows(sqlmock.NewRows([]string{"created_at"}).AddRow(time.Now()))
}

func TestShouldLogCCTVStreamEvent_StandbyDisabledByDefault(t *testing.T) {
	t.Setenv("CCTV_LOG_STANDBY_EVENTS", "")

	db, mock := setupMockGORM(t)
	now := time.Now()

	assert.False(t, shouldLogCCTVStreamEvent(db, uuid.New(), models.CCTVEventStandby, now),
		"standby tidak boleh dicatat secara default")
	assert.NoError(t, mock.ExpectationsWereMet(), "kebijakan standby harus dicek sebelum query database")
}

func TestShouldLogCCTVStreamEvent_StandbyEnabled(t *testing.T) {
	t.Setenv("CCTV_LOG_STANDBY_EVENTS", "true")

	db, mock := setupMockGORM(t)
	cctvID := uuid.New()
	now := time.Now()
	expectLastCCTVStreamLog(mock, cctvID, models.CCTVEventOnline, now.Add(-2*time.Minute))

	assert.True(t, shouldLogCCTVStreamEvent(db, cctvID, models.CCTVEventStandby, now))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestShouldLogCCTVStreamEvent_DuplicateWithinWindow(t *testing.T) {
	db, mock := setupMockGORM(t)
	cctvID := uuid.New()
	now := time.Now()
	expectLastCCTVStreamLog(mock, cctvID, models.CCTVEventOnline, now.Add(-30*time.Second))

	assert.False(t, shouldLogCCTVStreamEvent(db, cctvID, models.CCTVEventOnline, now),
		"event tipe sama dalam 60 detik harus dideduplikasi")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestShouldLogCCTVStreamEvent_DifferentEventWithinWindow(t *testing.T) {
	db, mock := setupMockGORM(t)
	cctvID := uuid.New()
	now := time.Now()
	expectLastCCTVStreamLog(mock, cctvID, models.CCTVEventOffline, now.Add(-30*time.Second))

	assert.True(t, shouldLogCCTVStreamEvent(db, cctvID, models.CCTVEventOnline, now),
		"transisi ke tipe berbeda harus tetap dicatat")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestShouldLogCCTVStreamEvent_SameEventOutsideWindow(t *testing.T) {
	db, mock := setupMockGORM(t)
	cctvID := uuid.New()
	now := time.Now()
	expectLastCCTVStreamLog(mock, cctvID, models.CCTVEventOnline, now.Add(-61*time.Second))

	assert.True(t, shouldLogCCTVStreamEvent(db, cctvID, models.CCTVEventOnline, now),
		"event tipe sama setelah jendela dedup harus dicatat lagi")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestShouldLogCCTVStreamEvent_NoPreviousLog(t *testing.T) {
	db, mock := setupMockGORM(t)
	cctvID := uuid.New()

	mock.ExpectQuery(lastCCTVStreamLogQuery).
		WithArgs(cctvID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"event_type", "created_at"}))

	assert.True(t, shouldLogCCTVStreamEvent(db, cctvID, models.CCTVEventOnline, time.Now()))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestShouldLogCCTVStreamEvent_DatabaseErrorFailsOpen(t *testing.T) {
	db, mock := setupMockGORM(t)
	cctvID := uuid.New()

	mock.ExpectQuery(lastCCTVStreamLogQuery).
		WithArgs(cctvID, 1).
		WillReturnError(errors.New("database tidak tersedia"))

	assert.True(t, shouldLogCCTVStreamEvent(db, cctvID, models.CCTVEventOffline, time.Now()),
		"kegagalan query dedup tidak boleh menghilangkan event penting")
	assert.NoError(t, mock.ExpectationsWereMet())
}
