package config

import (
	"errors"
	"strings"
	"testing"

	"sigap-backend/migrations"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// daftarTabelResmi mencerminkan seluruh model yang didaftarkan GORM di
// main.go (fase transisi AutoMigrate) — migration 000001 harus memuatnya.
var daftarTabelResmi = []string{
	"users",
	"cctvs",
	"maintenance_logs",
	"field_maintenances",
	"field_maintenance_checks",
	"field_maintenance_images",
	"audit_logs",
	"notifications",
	"maintenance_schedules",
	"daily_digests",
}

// TestEmbeddedMigrationFiles memastikan migration files tersemat ke binary dan
// mengikuti konvensi penamaan golang-migrate (<urutan>_<nama>.up|down.sql).
func TestEmbeddedMigrationFiles(t *testing.T) {
	entries, err := migrations.FS.ReadDir(".")
	assert.NoError(t, err)
	assert.NotEmpty(t, entries, "folder migrations/ harus berisi file SQL")

	for _, entry := range entries {
		assert.Regexp(t, `^\d{6}_.+\.(up|down)\.sql$`, entry.Name(),
			"nama file tidak mengikuti konvensi golang-migrate: "+entry.Name())
	}

	// Migration pertama (init schema) harus lengkap dengan pasangan up/down.
	names := map[string]bool{}
	for _, entry := range entries {
		names[entry.Name()] = true
	}
	assert.True(t, names["000001_init_schema.up.sql"], "000001_init_schema.up.sql harus ada")
	assert.True(t, names["000001_init_schema.down.sql"], "000001_init_schema.down.sql harus ada")
}

// TestInitSchemaUpMemuatSemuaTabel memastikan migration up berisi CREATE TABLE
// untuk seluruh model yang sebelumnya dikelola AutoMigrate GORM.
func TestInitSchemaUpMemuatSemuaTabel(t *testing.T) {
	up, err := migrations.FS.ReadFile("000001_init_schema.up.sql")
	assert.NoError(t, err)

	content := string(up)
	for _, table := range daftarTabelResmi {
		assert.Contains(t, content, "CREATE TABLE IF NOT EXISTS "+table,
			"tabel "+table+" harus dibuat di migration up")
	}
}

// TestInitSchemaKolomKunci memastikan kolom penting tiap tabel masih ada pada
// migration up (proteksi terhadap penghapusan kolom saat refactor skema).
func TestInitSchemaKolomKunci(t *testing.T) {
	up, err := migrations.FS.ReadFile("000001_init_schema.up.sql")
	assert.NoError(t, err)
	content := string(up)

	kolomKunci := map[string][]string{
		"users":                    {"id", "username", "password_hash", "role", "token_version"},
		"cctvs":                    {"id", "kode_cctv", "nama", "qr_payload", "health_score"},
		"maintenance_logs":         {"id", "cctv_id", "technician_id", "status", "scheduled_date"},
		"maintenance_schedules":    {"id", "cctv_id", "frequency", "next_run_at", "is_active"},
		"field_maintenances":       {"id", "cctv_id", "technician_id", "overall_status"},
		"field_maintenance_checks": {"id", "field_maintenance_id", "name", "result"},
		"field_maintenance_images": {"id", "field_maintenance_id", "image_url"},
		"audit_logs":               {"id", "user_id", "action", "module", "ip_address"},
		"notifications":            {"id", "user_id", "type", "is_read"},
		"daily_digests":            {"id", "digest_date", "summary"},
	}

	for table, cols := range kolomKunci {
		block := extractCreateBlock(content, table)
		assert.NotEmpty(t, block, "blok CREATE TABLE "+table+" tidak ditemukan")
		for _, col := range cols {
			assert.Contains(t, block, "\n    "+col+" ",
				"kolom "+col+" harus ada di tabel "+table)
		}
	}
}

// TestInitSchemaDownMenghapusSemuaTabel memastikan migration down menghapus
// seluruh tabel yang dibuat di migration up.
func TestInitSchemaDownMenghapusSemuaTabel(t *testing.T) {
	down, err := migrations.FS.ReadFile("000001_init_schema.down.sql")
	assert.NoError(t, err)

	content := string(down)
	for _, table := range daftarTabelResmi {
		assert.Contains(t, content, "DROP TABLE IF EXISTS "+table,
			"tabel "+table+" harus dihapus di migration down")
	}

	// FK sirkular maintenance_logs ↔ field_maintenances wajib memakai CASCADE
	// pada salah satu sisi agar rollback berurutan tidak gagal.
	assert.Contains(t, content, "DROP TABLE IF EXISTS maintenance_logs CASCADE")
}

// TestRunMigrationsErrorPropagates memastikan error dari driver migration
// di-*bubble* ke pemanggil saat database tidak tersedia (tanpa PostgreSQL).
func TestRunMigrationsErrorPropagates(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{SkipDefaultTransaction: true})
	assert.NoError(t, err)

	// Langkah pertama driver pgx adalah deteksi database (SELECT
	// CURRENT_DATABASE()). Simulasikan database mati → WithInstance gagal.
	mock.ExpectQuery(`SELECT CURRENT_DATABASE\(\)`).
		WillReturnError(errors.New("db down"))

	err = RunMigrations(gormDB)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "db down")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// extractCreateBlock mengembalikan teks blok CREATE TABLE untuk satu tabel.
func extractCreateBlock(content, table string) string {
	start := strings.Index(content, "CREATE TABLE IF NOT EXISTS "+table+" ")
	if start == -1 {
		return ""
	}
	end := strings.Index(content[start:], "\n);")
	if end == -1 {
		return content[start:]
	}
	return content[start : start+end+2]
}