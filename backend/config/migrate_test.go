package config

import (
	"testing"

	"sigap-backend/migrations"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// TestCCTVStreamLogIndexMigration memastikan migration #144 menambahkan index
// komposit untuk filter cctv_id/event_type + created_at, serta mengembalikan
// index kolom tunggal saat rollback.
func TestCCTVStreamLogIndexMigration(t *testing.T) {
	up, err := migrations.FS.ReadFile("000029_cctv_stream_logs_indexes.up.sql")
	assert.NoError(t, err)
	down, err := migrations.FS.ReadFile("000029_cctv_stream_logs_indexes.down.sql")
	assert.NoError(t, err)

	upSQL := string(up)
	assert.Contains(t, upSQL, "ON cctv_stream_logs(cctv_id, created_at)")
	assert.Contains(t, upSQL, "ON cctv_stream_logs(event_type, created_at)")
	assert.Contains(t, upSQL, "DROP INDEX IF EXISTS idx_cctv_stream_logs_cctv_id")
	assert.Contains(t, upSQL, "DROP INDEX IF EXISTS idx_cctv_stream_logs_event_type")

	downSQL := string(down)
	assert.Contains(t, downSQL, "DROP INDEX IF EXISTS idx_cctv_stream_logs_cctv_created")
	assert.Contains(t, downSQL, "DROP INDEX IF EXISTS idx_cctv_stream_logs_event_created")
	assert.Contains(t, downSQL, "ON cctv_stream_logs(cctv_id)")
	assert.Contains(t, downSQL, "ON cctv_stream_logs(event_type)")
}

func TestResolveLegacyAssetToCCTVByDirectID(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{SkipDefaultTransaction: true})
	assert.NoError(t, err)

	assetID := uuid.New()
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WithArgs(assetID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kode_cctv"}).AddRow(assetID, "CCTV-LEGACY"))

	got, ok := resolveLegacyAssetToCCTV(gormDB, assetID, false)
	assert.True(t, ok)
	assert.Equal(t, assetID, got)
	assert.NoError(t, mock.ExpectationsWereMet())
}
