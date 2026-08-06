package handlers

import (
	"testing"
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