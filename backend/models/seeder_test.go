package models

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestSeedAdminLeavesExistingAccountUnchanged(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		t.Fatal(err)
	}

	adminID := "11111111-1111-1111-1111-111111111111"
	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WithArgs("admin", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "deleted_at", "username", "password_hash", "full_name", "email", "role", "token_version", "must_change_password"}).
			AddRow(adminID, time.Now(), time.Now(), nil, "admin", "existing-hash", "Admin", "admin@sigap.local", "admin", 2, false))

	SeedAdmin(gormDB)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
