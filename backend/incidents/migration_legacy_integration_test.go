package incidents

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	pgxmigrate "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"sigap-backend/migrations"
)

// TestPostgresLegacy000031MigrationBackfillAndRollback menguji Koreksi #4 secara nyata
// pada database PostgreSQL terisolasi:
//  1. Menempatkan skema tepat pada versi 000033 (sebelum 000034).
//  2. Menyisipkan baris tiket lama skema 000031 (open, pending_verification, resolved unassigned,
//     resolved assigned + maintenance_log_id, serta baris telegram_outbox lama).
//  3. Menjalankan migrasi 000034 up -> memverifikasi nomor tiket unik INC-LEGACY-YYYYMMDD-XXXX,
//     timestamp_source = 'legacy_backfill_opened_at', sla_clock_basis = 'legacy_opened_at',
//     serta seluruh kolom NOT NULL terisi valid.
//  4. Menjalankan rollback 000034 down -> 000033 lalu 000034 up kembali tanpa error.
func TestPostgresLegacy000031MigrationBackfillAndRollback(t *testing.T) {
	dsn := os.Getenv("SIGAP_INCIDENT_TEST_DSN")
	if dsn == "" {
		t.Skip("isolated PostgreSQL DSN not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	var dbName string
	db.Raw("SELECT current_database()").Scan(&dbName)
	if dbName != "sigap_incident_test" {
		t.Fatal("refusing non-test database")
	}

	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		t.Fatal(err)
	}
	driver, err := pgxmigrate.WithInstance(sqlDB, &pgxmigrate.Config{})
	if err != nil {
		t.Fatal(err)
	}
	m, err := migrate.NewWithInstance("iofs", src, "pgx/v5", driver)
	if err != nil {
		t.Fatal(err)
	}
	// Pastikan di akhir test skema dikembalikan ke versi terbaru untuk test lain.
	t.Cleanup(func() {
		_ = m.Up()
	})

	// 1. Tempatkan database tepat di versi 33 (skema insiden 000031 aktif, sebelum 000034).
	if err := m.Migrate(33); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		t.Fatalf("migrate to v33 failed: %v", err)
	}

	siteID := uuid.New()
	adminID := uuid.New()
	techID := uuid.New()
	camOpenID := uuid.New()
	camResolvedSelfID := uuid.New()
	camResolvedTechID := uuid.New()
	maintLogID := uuid.New()
	incOpenID := uuid.New()
	incSelfID := uuid.New()
	incTechID := uuid.New()

	if err := db.Exec(`INSERT INTO sites (id, kode, nama, created_at, updated_at) VALUES (?, ?, 'Legacy Site', NOW(), NOW())`,
		siteID, "LG-"+siteID.String()[:5]).Error; err != nil {
		t.Fatal(err)
	}
	for _, u := range []struct {
		id   uuid.UUID
		role string
	}{{adminID, "admin"}, {techID, "teknisi"}} {
		if err := db.Exec(`INSERT INTO users (id, username, full_name, email, password_hash, role, created_at, updated_at) VALUES (?, ?, ?, ?, 'x', ?, NOW(), NOW())`,
			u.id, u.id.String(), u.role, u.id.String()+"@legacy.test", u.role).Error; err != nil {
			t.Fatal(err)
		}
	}
	for idx, cid := range []uuid.UUID{camOpenID, camResolvedSelfID, camResolvedTechID} {
		status := "aktif"
		if idx == 2 {
			status = "nonaktif"
		}
		if err := db.Exec(`INSERT INTO cctvs (id, kode_cctv, qr_payload, nama, lokasi_area, site_id, status, created_at, updated_at) VALUES (?, ?, ?, ?, 'Area Lama', ?, ?, NOW(), NOW())`,
			cid, "LGC-"+cid.String()[:5], cid.String(), "Kamera Legacy", siteID, status).Error; err != nil {
			t.Fatal(err)
		}
	}

	if err := db.Exec(`INSERT INTO maintenance_logs (id, cctv_id, site_id, technician_id, action_taken, notes, status, created_at, updated_at) VALUES (?, ?, ?, ?, 'Perbaikan lama', 'Catatan lama', 'completed', NOW(), NOW())`,
		maintLogID, camResolvedTechID, siteID, techID).Error; err != nil {
		t.Fatal(err)
	}

	openedBase := time.Date(2026, 10, 15, 2, 30, 0, 0, time.UTC)
	resolvedTime := openedBase.Add(45 * time.Minute)

	// 2. Sisipkan 3 baris tiket skema 000031 lama (tanpa ticket_number, tanpa sla_clock_start_at, dll.)
	if err := db.Exec(`INSERT INTO cctv_incidents (id, site_id, status, title, opened_at, updated_at) VALUES (?, ?, 'open', 'Tiket Lama Open', ?, ?)`,
		incOpenID, siteID, openedBase, openedBase).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO cctv_incidents (id, site_id, status, title, opened_at, updated_at, resolved_at, resolved_by, resolution_note) VALUES (?, ?, 'resolved', 'Tiket Lama Self', ?, ?, ?, ?, 'Pulih sendiri')`,
		incSelfID, siteID, openedBase.Add(time.Minute), resolvedTime, resolvedTime, adminID).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO cctv_incidents (id, site_id, status, title, maintenance_log_id, assigned_to, opened_at, updated_at, resolved_at, resolved_by, resolution_note) VALUES (?, ?, 'resolved', 'Tiket Lama Tech', ?, ?, ?, ?, ?, ?, 'Sudah diperbaiki')`,
		incTechID, siteID, maintLogID, techID, openedBase.Add(2*time.Minute), resolvedTime, resolvedTime, adminID).Error; err != nil {
		t.Fatal(err)
	}

	for _, pair := range []struct {
		incID  uuid.UUID
		camID  uuid.UUID
		active bool
		state  string
	}{
		{incOpenID, camOpenID, true, "offline"},
		{incSelfID, camResolvedSelfID, false, "recovered"},
		{incTechID, camResolvedTechID, false, "recovered"},
	} {
		if err := db.Exec(`INSERT INTO cctv_incident_cameras (id, incident_id, cctv_id, camera_name, is_active_member, connection_state, last_observed_at, consecutive_live_count) VALUES (?, ?, ?, 'Kamera Legacy', ?, ?, ?, 0)`,
			uuid.New(), pair.incID, pair.camID, pair.active, pair.state, openedBase).Error; err != nil {
			t.Fatal(err)
		}
	}

	outboxFailedID := uuid.New()
	if err := db.Exec(`INSERT INTO telegram_outbox (id, incident_id, site_id, event_type, text, status, attempts, next_attempt_at, created_at, last_error) VALUES (?, ?, ?, 'CCTV_OFFLINE', 'legacy msg', 'failed', 5, ?, ?, 'old err')`,
		outboxFailedID, incOpenID, siteID, openedBase, openedBase).Error; err != nil {
		t.Fatal(err)
	}

	// 3. Jalankan migrasi 000034 up!
	if err := m.Migrate(34); err != nil {
		t.Fatalf("000034 up failed on legacy 000031 rows: %v", err)
	}

	var migrated []Incident
	if err := db.Preload("Members").Where("site_id = ?", siteID).Order("opened_at ASC").Find(&migrated).Error; err != nil {
		t.Fatal(err)
	}
	if len(migrated) != 3 {
		t.Fatalf("expected 3 migrated incidents, got %d", len(migrated))
	}

	seenTickets := map[string]bool{}
	for _, inc := range migrated {
		if !strings.HasPrefix(inc.TicketNumber, "INC-LEGACY-20261015-") || seenTickets[inc.TicketNumber] {
			t.Fatalf("invalid or duplicate backfilled ticket_number: %s", inc.TicketNumber)
		}
		seenTickets[inc.TicketNumber] = true
		if inc.TimestampSource != TimestampSourceLegacyBackfill || inc.SLAClockBasis != SLABasisLegacyOpenedAt {
			t.Fatalf("legacy row must be explicitly tagged as backfill: %+v", inc)
		}
		if inc.FirstFailedObservedAt.IsZero() || inc.ConfirmedOfflineAt.IsZero() || inc.SLAClockStartAt.IsZero() {
			t.Fatalf("NOT NULL timestamps missing after backfill: %+v", inc)
		}
	}

	if migrated[1].ResolutionType == nil || *migrated[1].ResolutionType != ResolutionSelfRecovered || migrated[1].Members[0].MemberOutcome != OutcomeSelfRecovered {
		t.Fatalf("unassigned resolved legacy row should backfill self_recovered: %+v", migrated[1])
	}
	if migrated[2].ResolutionType == nil || *migrated[2].ResolutionType != ResolutionRepairedVerified || migrated[2].Members[0].MemberOutcome != OutcomeRecoveredVerified {
		t.Fatalf("assigned resolved legacy row should backfill repaired_verified: %+v", migrated[2])
	}

	var nonaktifMonitoring bool
	db.Raw("SELECT monitoring_enabled FROM cctvs WHERE id = ?", camResolvedTechID).Scan(&nonaktifMonitoring)
	if nonaktifMonitoring {
		t.Fatal("nonaktif legacy CCTV must have monitoring_enabled=false after migration")
	}

	var migratedOutbox Outbox
	if err := db.First(&migratedOutbox, "id = ?", outboxFailedID).Error; err != nil {
		t.Fatal(err)
	}
	if migratedOutbox.CoalesceUntil.IsZero() || migratedOutbox.FailedAt == nil {
		t.Fatalf("legacy failed outbox must backfill coalesce_until and failed_at: %+v", migratedOutbox)
	}

	// 4. Uji rollback 000034 down -> 000033 lalu 000034 up kembali secara bersih!
	if err := m.Migrate(33); err != nil {
		t.Fatalf("000034 down failed: %v", err)
	}
	if err := m.Migrate(34); err != nil {
		t.Fatalf("000034 re-up after down failed: %v", err)
	}
}
