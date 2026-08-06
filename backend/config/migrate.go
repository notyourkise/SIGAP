package config

import (
	"errors"
	"fmt"
	"log"

	"sigap-backend/migrations"
	"sigap-backend/models"
	"sigap-backend/utils"

	"github.com/golang-migrate/migrate/v4"
	pgxmigrate "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// legacyAssetRef mewakili baris maintenance_logs lama yang masih memakai asset_id.
type legacyAssetRef struct {
	ID      uuid.UUID
	AssetID uuid.UUID
}

// RunMigrations menerapkan seluruh migration DB versioned (golang-migrate v4)
// yang belum berjalan pada database (m.Up()). Dipanggil otomatis saat startup
// aplikasi di main.go — sejak Task 7.1 migration versioned menjadi sumber
// skema resmi, sedangkan AutoMigrate GORM hanya fase transisi (dikomentari).
func RunMigrations(db *gorm.DB) error {
	m, err := newMigrateInstance(db)
	if err != nil {
		return err
	}
	defer func() { _, _ = m.Close() }()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("menerapkan migration: %w", err)
	}
	logMigrationVersion(m)
	return nil
}

// MigrationUp menerapkan semua migration yang belum berjalan. Dipakai CLI
// manual: go run cmd/migrate/main.go up.
func MigrationUp(db *gorm.DB) error {
	m, err := newMigrateInstance(db)
	if err != nil {
		return err
	}
	defer func() { _, _ = m.Close() }()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migration up: %w", err)
	}
	logMigrationVersion(m)
	return nil
}

// MigrationDown melakukan rollback seluruh migration (kembali ke versi 0).
// Dipakai CLI manual: go run cmd/migrate/main.go down.
func MigrationDown(db *gorm.DB) error {
	m, err := newMigrateInstance(db)
	if err != nil {
		return err
	}
	defer func() { _, _ = m.Close() }()

	if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migration down: %w", err)
	}
	logMigrationVersion(m)
	return nil
}

// MigrationVersion mengembalikan versi migration aktif dan status dirty.
// Dipakai CLI manual: go run cmd/migrate/main.go version.
func MigrationVersion(db *gorm.DB) (uint, bool, error) {
	m, err := newMigrateInstance(db)
	if err != nil {
		return 0, false, err
	}
	defer func() { _, _ = m.Close() }()

	version, dirty, err := m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("membaca versi migration: %w", err)
	}
	return version, dirty, nil
}

// newMigrateInstance membangun instance golang-migrate dari koneksi GORM yang
// sudah ada: sumber migration dari file SQL ter-embed (backend/migrations)
// dan driver database PostgreSQL pgx v5. Riwayat versi disimpan di tabel
// schema_migrations pada skema database yang sama.
func newMigrateInstance(db *gorm.DB) (*migrate.Migrate, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("mengambil sql.DB dari GORM: %w", err)
	}

	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return nil, fmt.Errorf("memuat sumber migration: %w", err)
	}

	driver, err := pgxmigrate.WithInstance(sqlDB, &pgxmigrate.Config{})
	if err != nil {
		return nil, fmt.Errorf("inisialisasi driver migration pgx: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", src, "pgx/v5", driver)
	if err != nil {
		return nil, fmt.Errorf("inisialisasi migrate: %w", err)
	}
	return m, nil
}

// logMigrationVersion mencatat versi migration aktif pada logger aplikasi.
func logMigrationVersion(m *migrate.Migrate) {
	version, dirty, err := m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		utils.Log.Info("🗄️ Migration DB versioned: belum ada migration diterapkan (versi 0)")
		return
	}
	if err != nil {
		utils.Log.Warnf("⚠️ Tidak dapat membaca versi migration: %v", err)
		return
	}
	utils.Log.Infof("🗄️ Migration DB versioned aktif pada versi %d (dirty=%v)", version, dirty)
}

// MigrateMaintenanceToCCTV memigrasi kolom asset_id lama pada maintenance_logs
// menuju cctv_id — menjadikan tabel cctvs satu-satunya sumber kebenaran aset
// (Fase 3.6). Data lama dipetakan ke CCTV berdasarkan urutan: ID langsung,
// IP kamera/NVR, kode CCTV, lalu nama. Aman dijalankan berulang kali (idempotent).
func MigrateMaintenanceToCCTV(db *gorm.DB) {
	// Pastikan tabel dan kolom cctv_id sudah ada (dibuat AutoMigrate).
	if !db.Migrator().HasTable("maintenance_logs") || !db.Migrator().HasColumn(&models.MaintenanceLog{}, "CCTVID") {
		return
	}
	// Kolom asset_id lama sudah dihapus — tidak ada yang perlu dimigrasi.
	if !db.Migrator().HasColumn("maintenance_logs", "asset_id") {
		return
	}

	var refs []legacyAssetRef
	if err := db.Raw(`SELECT id, asset_id FROM maintenance_logs WHERE cctv_id IS NULL`).Scan(&refs).Error; err != nil {
		log.Printf("⚠️ Migrasi maintenance→cctv: gagal membaca data lama: %v", err)
	} else if len(refs) > 0 {
		hasAssetsTable := db.Migrator().HasTable("assets")
		updated := 0
		for _, ref := range refs {
			if ref.AssetID == uuid.Nil {
				continue
			}
			cctvID, ok := resolveLegacyAssetToCCTV(db, ref.AssetID, hasAssetsTable)
			if !ok {
				log.Printf("⚠️ Migrasi maintenance→cctv: log %s tidak dapat dipetakan ke CCTV (asset_id %s). Periksa secara manual.", ref.ID, ref.AssetID)
				continue
			}
			if err := db.Model(&models.MaintenanceLog{}).Unscoped().Where("id = ?", ref.ID).Update("cctv_id", cctvID).Error; err != nil {
				log.Printf("⚠️ Migrasi maintenance→cctv: gagal memperbarui log %s: %v", ref.ID, err)
				continue
			}
			updated++
		}
		log.Printf("✅ Migrasi maintenance→cctv selesai: %d log dipetakan ke CCTV", updated)
	}

	// Hapus kolom legacy asset_id beserta foreign key-nya. Tanpa ini kolom lama yang masih
	// NOT NULL membuat INSERT baru gagal (model sudah tidak mengirim asset_id lagi).
	if db.Migrator().HasColumn("maintenance_logs", "asset_id") {
		if err := db.Migrator().DropColumn("maintenance_logs", "asset_id"); err != nil {
			log.Printf("⚠️ Migrasi maintenance→cctv: gagal menghapus kolom legacy asset_id: %v", err)
		} else {
			log.Printf("✅ Migrasi maintenance→cctv: kolom legacy asset_id dihapus dari maintenance_logs")
		}
	}
}

// resolveLegacyAssetToCCTV mencari CCTV yang sesuai dengan referensi asset lama.
func resolveLegacyAssetToCCTV(db *gorm.DB, assetID uuid.UUID, hasAssetsTable bool) (uuid.UUID, bool) {
	// 1. Apakah asset_id sudah berupa ID CCTV yang valid?
	var cctv models.CCTV
	if err := db.Unscoped().First(&cctv, "id = ?", assetID).Error; err == nil {
		return cctv.ID, true
	}

	if !hasAssetsTable {
		return uuid.Nil, false
	}

	// 2. Petakan lewat tabel assets lama (ip / kode / nama).
	var legacy struct {
		IPAddress    string
		SerialNumber string
		Name         string
	}
	if err := db.Raw(`SELECT ip_address, serial_number, name FROM assets WHERE id = ?`, assetID).Scan(&legacy).Error; err != nil {
		return uuid.Nil, false
	}

	if legacy.IPAddress != "" {
		if err := db.Unscoped().First(&cctv, "ip_camera = ? OR ip_nvr = ?", legacy.IPAddress, legacy.IPAddress).Error; err == nil {
			return cctv.ID, true
		}
	}
	if legacy.SerialNumber != "" {
		if err := db.Unscoped().First(&cctv, "kode_cctv = ?", legacy.SerialNumber).Error; err == nil {
			return cctv.ID, true
		}
	}
	if legacy.Name != "" {
		if err := db.Unscoped().First(&cctv, "nama = ?", legacy.Name).Error; err == nil {
			return cctv.ID, true
		}
	}
	return uuid.Nil, false
}
