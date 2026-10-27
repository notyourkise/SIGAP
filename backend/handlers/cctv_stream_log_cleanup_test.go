package handlers

import (
	"errors"
	"testing"
	"time"

	"sigap-backend/config"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const deleteCCTVStreamLogsBatchRegex = `^DELETE FROM cctv_stream_logs WHERE id IN \(SELECT id FROM cctv_stream_logs WHERE created_at < \$1 ORDER BY created_at ASC LIMIT \$2\)$`

func TestCleanupCCTVStreamLogs_DeletesInBatches(t *testing.T) {
	db, mock := setupMockGORM(t)

	for _, affected := range []int64{1000, 1000, 250} {
		mock.ExpectExec(deleteCCTVStreamLogsBatchRegex).
			WithArgs(sqlmock.AnyArg(), cctvStreamLogCleanupBatchSize).
			WillReturnResult(sqlmock.NewResult(0, affected))
	}

	deleted, err := CleanupCCTVStreamLogs(db, 90)
	require.NoError(t, err)
	assert.Equal(t, int64(2250), deleted)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCleanupCCTVStreamLogs_IsIdempotent(t *testing.T) {
	db, mock := setupMockGORM(t)

	mock.ExpectExec(deleteCCTVStreamLogsBatchRegex).
		WithArgs(sqlmock.AnyArg(), cctvStreamLogCleanupBatchSize).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(deleteCCTVStreamLogsBatchRegex).
		WithArgs(sqlmock.AnyArg(), cctvStreamLogCleanupBatchSize).
		WillReturnResult(sqlmock.NewResult(0, 0))

	firstDeleted, err := CleanupCCTVStreamLogs(db, 90)
	require.NoError(t, err)
	secondDeleted, err := CleanupCCTVStreamLogs(db, 90)
	require.NoError(t, err)

	assert.Equal(t, int64(2), firstDeleted)
	assert.Zero(t, secondDeleted, "pemanggilan kedua tidak boleh menghapus data yang sama")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCleanupCCTVStreamLogs_ReturnsDatabaseError(t *testing.T) {
	db, mock := setupMockGORM(t)

	mock.ExpectExec(deleteCCTVStreamLogsBatchRegex).
		WithArgs(sqlmock.AnyArg(), cctvStreamLogCleanupBatchSize).
		WillReturnError(errors.New("database tidak tersedia"))

	deleted, err := CleanupCCTVStreamLogs(db, 90)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database tidak tersedia")
	assert.Zero(t, deleted)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCCTVStreamLogRetentionCutoff(t *testing.T) {
	now := time.Date(2026, 10, 28, 10, 0, 0, 0, time.UTC)
	cutoff := cctvStreamLogRetentionCutoff(now, 90)

	assert.Equal(t, time.Date(2026, 7, 30, 10, 0, 0, 0, time.UTC), cutoff)
	assert.True(t, now.AddDate(0, 0, -89).After(cutoff), "log 89 hari harus aman")
	assert.True(t, now.AddDate(0, 0, -91).Before(cutoff), "log 91 hari harus terhapus")
	assert.Equal(t,
		now.AddDate(0, 0, -config.DefaultCCTVStreamLogRetentionDays),
		cctvStreamLogRetentionCutoff(now, 0),
		"retensi <= 0 harus memakai default",
	)
}
