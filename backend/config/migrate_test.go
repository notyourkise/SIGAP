package config

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

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
