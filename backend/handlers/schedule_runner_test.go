package handlers

import (
	"errors"
	"testing"

	"sigap-backend/models"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// TestProcessDueSchedules_ActiveCreatesLog memverifikasi generator membuat
// log pemeliharaan untuk jadwal aktif yang waktunya telah tiba (tanpa
// teknisi → status "scheduled", menunggu ditugaskan admin).
func TestProcessDueSchedules_ActiveCreatesLog(t *testing.T) {
	db, mock := setupMockGORM(t)
	scheduleID, cctvID := uuid.New(), uuid.New()
	now := time.Now()

	mock.ExpectQuery(`^SELECT \* FROM "maintenance_schedules"`).
		WillReturnRows(maintenanceScheduleRows(scheduleID, cctvID, nil, "daily", true))
	// Idempotensi: belum ada log untuk jadwal ini pada hari yang sama
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnError(gorm.ErrRecordNotFound)
	mock.ExpectExec(`^INSERT INTO "maintenance_logs"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`^UPDATE "maintenance_schedules" SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	// Audit log dibuat langsung (identitas "system", bukan dari request HTTP)
	mock.ExpectExec(`^INSERT INTO "audit_logs"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	ProcessDueSchedulesAt(db, now)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestProcessDueSchedules_InactiveSkipped memastikan tidak ada log yang
// dibuat bila tidak ada jadwal aktif yang jatuh tempo (filter is_active =
// true di query generator).
func TestProcessDueSchedules_InactiveSkipped(t *testing.T) {
	db, mock := setupMockGORM(t)
	now := time.Now()

	// Tidak ada jadwal aktif yang jatuh tempo → tidak boleh ada INSERT log
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_schedules"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "cctv_id", "frequency", "next_run_at", "is_active"}))

	ProcessDueSchedulesAt(db, now)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestProcessDueSchedules_DuplicateSameDaySkipped memverifikasi idempotensi:
// bila log untuk jadwal pada tanggal yang sama sudah ada, generator menolak
// membuat duplikat (tidak ada INSERT log).
func TestProcessDueSchedules_DuplicateSameDaySkipped(t *testing.T) {
	db, mock := setupMockGORM(t)
	scheduleID, cctvID, techID := uuid.New(), uuid.New(), uuid.New()
	now := time.Now()

	mock.ExpectQuery(`^SELECT \* FROM "maintenance_schedules"`).
		WillReturnRows(maintenanceScheduleRows(scheduleID, cctvID, &techID, "daily", true))
	// Log tanggal yang sama SUDAH ADA → generator harus berhenti (skip)
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "created_at", "updated_at", "deleted_at",
			"cctv_id", "technician_id", "action_taken", "status",
			"scheduled_date", "completed_at", "image_url",
		}).AddRow(uuid.New(), now, now, nil, cctvID, techID,
			"Bersihkan lensa kamera", "ongoing", now, nil, ""))

	ProcessDueSchedulesAt(db, now)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestProcessDueSchedules_ActiveWithTechnician memverifikasi jadwal berteknisi:
// log langsung berstatus "ongoing", teknisi menerima notifikasi, dan
// status pindah ke jadwal berikutnya.
func TestProcessDueSchedules_ActiveWithTechnician(t *testing.T) {
	db, mock := setupMockGORM(t)
	scheduleID, cctvID, techID := uuid.New(), uuid.New(), uuid.New()
	now := time.Now()

	mock.ExpectQuery(`^SELECT \* FROM "maintenance_schedules"`).
		WillReturnRows(maintenanceScheduleRows(scheduleID, cctvID, &techID, "weekly", true))
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnError(gorm.ErrRecordNotFound)
	mock.ExpectExec(`^INSERT INTO "maintenance_logs"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`^UPDATE "maintenance_schedules" SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`^INSERT INTO "audit_logs"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`^INSERT INTO "notifications"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	ProcessDueSchedulesAt(db, now)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestAudit_GeneratorMustNotDriftFromPlannedTime memverifikasi perbaikan M07:
// keterlambatan worker/generator (misal tertunda 45 menit) tidak boleh menggeser
// jam rencana tugas berikutnya. Anchor harus tetap berpatokan pada waktu rencana
// sebelumnya (planned nextRunAt).
func TestAudit_GeneratorMustNotDriftFromPlannedTime(t *testing.T) {
	db, mock := setupMockGORM(t)
	due := time.Date(2026, 10, 30, 9, 0, 0, 0, time.UTC)
	now := due.Add(45 * time.Minute)
	schedule := models.MaintenanceSchedule{
		Base:      models.Base{ID: uuid.New()},
		CCTVID:    uuid.New(),
		Frequency: "daily",
		NextRunAt: due,
		IsActive:  true,
	}

	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnError(gorm.ErrRecordNotFound)
	mock.ExpectExec(`^INSERT INTO "maintenance_logs"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`^UPDATE "maintenance_schedules" SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`^INSERT INTO "audit_logs"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	processDueSchedule(db, &schedule, now)

	want := due.AddDate(0, 0, 1)
	assert.True(t, schedule.NextRunAt.Equal(want), "planned next=%s actual=%s: scheduler delay shifts future schedule", want, schedule.NextRunAt)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCalculateNextRun_AntiDriftAndCatchUp menguji anti-drift dan catch-up
// setelah downtime pada berbagai frekuensi.
func TestCalculateNextRun_AntiDriftAndCatchUp(t *testing.T) {
	t.Run("worker delayed does not drift anchor time", func(t *testing.T) {
		planned := time.Date(2026, 10, 30, 9, 0, 0, 0, time.UTC)
		delayedNow := planned.Add(45 * time.Minute) // worker tertunda 45 menit
		got := calculateNextRun("daily", planned, delayedNow)

		want := time.Date(2026, 10, 31, 9, 0, 0, 0, time.UTC)
		assert.True(t, got.Equal(want), "expected %v, got %v", want, got)
	})

	t.Run("server downtime multi-day catch-up", func(t *testing.T) {
		// Server mati 4 hari: jadwal rencana 2026-10-25 09:00:00
		// Server hidup kembali pada 2026-10-29 14:00:00
		planned := time.Date(2026, 10, 25, 9, 0, 0, 0, time.UTC)
		resumedNow := time.Date(2026, 10, 29, 14, 0, 0, 0, time.UTC)

		got := calculateNextRun("daily", planned, resumedNow)
		// Jadwal berikutnya harus langsung berada di masa depan (> resumedNow)
		// yaitu 2026-10-30 09:00:00, dengan jam anchor 09:00 tetap terjaga.
		want := time.Date(2026, 10, 30, 9, 0, 0, 0, time.UTC)
		assert.True(t, got.Equal(want), "expected catch-up next run %v, got %v", want, got)
	})

	t.Run("server downtime monthly catch-up", func(t *testing.T) {
		// Server mati 2 bulan: jadwal rencana 2027-01-31 09:00:00
		// Server hidup kembali pada 2027-03-15 10:00:00
		planned := time.Date(2027, 1, 31, 9, 0, 0, 0, time.UTC)
		resumedNow := time.Date(2027, 3, 15, 10, 0, 0, 0, time.UTC)

		got := calculateNextRun("monthly", planned, resumedNow)
		// Jan 31 -> Feb 28 (<= now) -> Mar 28 (> now)
		want := time.Date(2027, 3, 28, 9, 0, 0, 0, time.UTC)
		assert.True(t, got.Equal(want), "expected monthly catch-up %v, got %v", want, got)
	})

	t.Run("zero anchor falls back to now", func(t *testing.T) {
		now := time.Date(2026, 10, 30, 9, 0, 0, 0, time.UTC)
		got := calculateNextRun("daily", time.Time{}, now)
		want := time.Date(2026, 10, 31, 9, 0, 0, 0, time.UTC)
		assert.True(t, got.Equal(want), "expected %v, got %v", want, got)
	})
}


// TestProcessDueSchedules_SoftDeletedLogPreventsDuplicate memverifikasi mitigasi M09:
// bila log pemeliharaan pada hari yang sama pernah dibuat lalu di-soft-delete,
// generator menggunakan Unscoped() sehingga log soft-deleted tetap terdeteksi
// dan generator menolak membuat tugas duplikat.
func TestProcessDueSchedules_SoftDeletedLogPreventsDuplicate(t *testing.T) {
	db, mock := setupMockGORM(t)
	scheduleID, cctvID, techID := uuid.New(), uuid.New(), uuid.New()
	now := time.Now()
	deletedAt := now.Add(-1 * time.Hour)

	mock.ExpectQuery(`^SELECT \* FROM "maintenance_schedules"`).
		WillReturnRows(maintenanceScheduleRows(scheduleID, cctvID, &techID, "daily", true))
	// Log soft-deleted SUDAH ADA (deleted_at != nil) -> generator harus tetap skip (idempotent)
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "created_at", "updated_at", "deleted_at",
			"cctv_id", "technician_id", "action_taken", "status",
			"scheduled_date", "completed_at", "image_url",
		}).AddRow(uuid.New(), now, now, deletedAt, cctvID, techID,
			"Bersihkan lensa kamera", "ongoing", now, nil, ""))

	ProcessDueSchedulesAt(db, now)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestProcessDueSchedules_ConcurrentWorkerUniqueConstraintHandled memverifikasi mitigasi M09:
// bila dua worker memproses jadwal secara bersamaan dan keduanya lolos pengecekan awal,
// unique constraint database uq_maintenance_logs_schedule_day menolak insert kedua.
// Worker kedua menangkap error ini secara aman (idempotent skip) tanpa panik atau error fatal.
func TestProcessDueSchedules_ConcurrentWorkerUniqueConstraintHandled(t *testing.T) {
	db, mock := setupMockGORM(t)
	scheduleID, cctvID, techID := uuid.New(), uuid.New(), uuid.New()
	now := time.Now()

	mock.ExpectQuery(`^SELECT \* FROM "maintenance_schedules"`).
		WillReturnRows(maintenanceScheduleRows(scheduleID, cctvID, &techID, "daily", true))
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnError(gorm.ErrRecordNotFound)
	// Worker kedua kalah race: PostgreSQL melempar unique constraint violation (SQLSTATE 23505)
	mock.ExpectExec(`^INSERT INTO "maintenance_logs"`).
		WillReturnError(errors.New("ERROR: duplicate key value violates unique constraint \"uq_maintenance_logs_schedule_day\" (SQLSTATE 23505)"))

	ProcessDueSchedulesAt(db, now)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestProcessDueSchedules_RecoveryAfterCrash memverifikasi pemulihan (recovery) M09:
// bila log pemeliharaan berhasil dibuat pada siklus sebelumnya namun server terhenti
// sebelum jadwal berhasil digeser (NextRunAt <= ScheduledDate dan NextRunAt < now),
// generator memulihkan NextRunAt ke jadwal berikutnya tanpa membuat log duplikat.
func TestProcessDueSchedules_RecoveryAfterCrash(t *testing.T) {
	db, mock := setupMockGORM(t)
	scheduleID, cctvID, techID := uuid.New(), uuid.New(), uuid.New()

	// Waktu rencana jadwal sebelumnya: jam 09.00
	plannedTime := time.Date(2026, 10, 30, 9, 0, 0, 0, time.UTC)
	// Server hidup kembali pada jam 11.00 (setelah crash sebelum update NextRunAt)
	resumedNow := time.Date(2026, 10, 30, 11, 0, 0, 0, time.UTC)

	// NextRunAt pada jadwal di database masih jam 09.00 (belum pernah bergeser)
	scheduleRows := sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"cctv_id", "technician_id", "action_taken", "frequency",
		"next_run_at", "is_active", "last_generated_at",
	}).AddRow(scheduleID, plannedTime, plannedTime, nil, cctvID, techID,
		"Bersihkan lensa dan periksa konektor RJ45", "daily", plannedTime, true, nil)

	mock.ExpectQuery(`^SELECT \* FROM "maintenance_schedules"`).
		WillReturnRows(scheduleRows)
	// Log untuk tanggal 30 Okt sudah ada di database dari eksekusi sebelum crash
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "created_at", "updated_at", "deleted_at",
			"cctv_id", "technician_id", "action_taken", "status",
			"scheduled_date", "completed_at", "image_url",
		}).AddRow(uuid.New(), plannedTime, plannedTime, nil, cctvID, techID,
			"Bersihkan lensa dan periksa konektor RJ45", "ongoing", plannedTime, nil, ""))

	// Recovery: generator TIDAK melakukan INSERT log baru, melainkan mengupdate NextRunAt jadwal
	mock.ExpectExec(`^UPDATE "maintenance_schedules" SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	ProcessDueSchedulesAt(db, resumedNow)

	assert.NoError(t, mock.ExpectationsWereMet())
}

