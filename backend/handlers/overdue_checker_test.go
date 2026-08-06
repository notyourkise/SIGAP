package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func overdueMaintenanceRows(id, cctvID uuid.UUID, technicianID *uuid.UUID, status string, notifiedAt *time.Time, scheduledAt time.Time) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"cctv_id", "technician_id", "maintenance_result_id", "maintenance_schedule_id",
		"action_taken", "notes", "status", "overdue_notified_at", "scheduled_date",
		"completed_at", "image_url",
	}).AddRow(
		id, scheduledAt, scheduledAt, nil,
		cctvID, technicianID, nil, nil,
		"Bersihkan lensa kamera", "", status, notifiedAt, scheduledAt,
		nil, "",
	)
}

func overdueCCTVRows(id uuid.UUID) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "kode_cctv", "nama", "status"}).
		AddRow(id, "CCTV-OVERDUE-01", "Kamera Gerbang Utama", "aktif")
}

// TestProcessOverdueMaintenance_OverdueNotified memastikan tugas overdue yang
// belum memiliki flag mengirim notifikasi dan menulis waktu pengiriman.
func TestProcessOverdueMaintenance_OverdueNotified(t *testing.T) {
	db, mock := setupMockGORM(t)
	logID, cctvID := uuid.New(), uuid.New()
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	scheduledAt := now.Add(-2 * time.Hour)
	adminID := uuid.New()

	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(overdueMaintenanceRows(logID, cctvID, nil, "scheduled", nil, scheduledAt))
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(overdueCCTVRows(cctvID))
	mock.ExpectQuery(`^SELECT "id" FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(adminID))
	mock.ExpectExec(`^INSERT INTO "notifications"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`^UPDATE "maintenance_logs" SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	ProcessOverdueMaintenanceAt(db, now)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestProcessOverdueMaintenance_OngoingNotifiesTechnician memastikan log
// ongoing mengirim eskalasi tambahan kepada teknisi yang ditugaskan.
func TestProcessOverdueMaintenance_OngoingNotifiesTechnician(t *testing.T) {
	db, mock := setupMockGORM(t)
	logID, cctvID, technicianID := uuid.New(), uuid.New(), uuid.New()
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	scheduledAt := now.Add(-2 * time.Hour)
	adminID := uuid.New()

	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(overdueMaintenanceRows(logID, cctvID, &technicianID, "ongoing", nil, scheduledAt))
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(overdueCCTVRows(cctvID))
	mock.ExpectQuery(`^SELECT "id" FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(adminID))
	mock.ExpectExec(`^INSERT INTO "notifications"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`^INSERT INTO "notifications"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`^UPDATE "maintenance_logs" SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	ProcessOverdueMaintenanceAt(db, now)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestProcessOverdueMaintenance_AlreadyNotifiedSkipped memastikan query checker
// melewati log yang sudah memiliki overdue_notified_at.
func TestProcessOverdueMaintenance_AlreadyNotifiedSkipped(t *testing.T) {
	db, mock := setupMockGORM(t)
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	notifiedAt := now.Add(-time.Hour)

	// Kondisi overdue_notified_at IS NULL membuat log ini tidak dipilih.
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "cctv_id", "status", "scheduled_date", "overdue_notified_at",
		}).AddRow(uuid.New(), uuid.New(), "scheduled", now.Add(-2*time.Hour), notifiedAt))

	ProcessOverdueMaintenanceAt(db, now)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestProcessOverdueMaintenance_CompletedAndFailedSkipped memastikan status
// terminal tidak dieskalasi walaupun ada di hasil query yang dimock.
func TestProcessOverdueMaintenance_CompletedAndFailedSkipped(t *testing.T) {
	db, mock := setupMockGORM(t)
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	completedID, failedID := uuid.New(), uuid.New()
	completedCCTV, failedCCTV := uuid.New(), uuid.New()

	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "created_at", "updated_at", "deleted_at", "cctv_id", "technician_id",
			"maintenance_result_id", "maintenance_schedule_id", "action_taken", "notes",
			"status", "overdue_notified_at", "scheduled_date", "completed_at", "image_url",
		}).AddRow(
			completedID, now, now, nil, completedCCTV, nil, nil, nil,
			"Selesai", "", "completed", nil, now.Add(-time.Hour), now, "",
		).AddRow(
			failedID, now, now, nil, failedCCTV, nil, nil, nil,
			"Perlu tindak lanjut", "", "failed", nil, now.Add(-time.Hour), nil, "",
		))
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kode_cctv", "nama", "status"}).
			AddRow(completedCCTV, "CCTV-COMPLETED", "Kamera Selesai", "aktif").
			AddRow(failedCCTV, "CCTV-FAILED", "Kamera Gagal", "aktif"))

	ProcessOverdueMaintenanceAt(db, now)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestUpdateMaintenanceLog_RescheduleResetsOverdueNotification memastikan
// reschedule ke masa depan mengosongkan flag agar checker dapat mengirim ulang
// bila tugas tersebut kembali overdue.
func TestUpdateMaintenanceLog_RescheduleResetsOverdueNotification(t *testing.T) {
	_, mock := setupMockGORM(t)
	logID, cctvID, technicianID := uuid.New(), uuid.New(), uuid.New()
	now := time.Now()
	notifiedAt := now.Add(-time.Hour)

	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(overdueMaintenanceRows(logID, cctvID, &technicianID, "ongoing", &notifiedAt, now.Add(-time.Hour)))
	mock.ExpectExec(`^UPDATE "maintenance_logs" SET .*overdue_notified_at`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	future := now.Add(24 * time.Hour).Format(time.RFC3339)
	app := newTestApp()
	app.Put("/maintenance-logs/:id", withUserLocals(uuid.New().String(), "admin"), UpdateMaintenanceLog)
	request := httptest.NewRequest(
		"PUT",
		"/maintenance-logs/"+logID.String(),
		strings.NewReader(`{"scheduled_date":"`+future+`"}`),
	)
	request.Header.Set("Content-Type", "application/json")

	response, err := app.Test(request)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, response.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}
