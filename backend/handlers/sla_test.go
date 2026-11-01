package handlers

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// slaPolicyRows membangun baris tabel sla_policies untuk mock sqlmock.
func slaPolicyRows(id uuid.UUID, nama, jenis string, respon, selesai int) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"nama", "jenis_tugas", "target_jam_respon", "target_jam_selesai",
	}).AddRow(id, time.Now(), time.Now(), nil, nama, jenis, respon, selesai)
}

// TestCreateSLAPolicy_Success menguji pembuatan kebijakan SLA.
func TestCreateSLAPolicy_Success(t *testing.T) {
	_, mock := setupMockGORM(t)

	mock.ExpectExec(`^INSERT INTO "sla_policies"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	app := newTestApp()
	app.Post("/sla-policies", withUserLocals(uuid.New().String(), "admin"), CreateSLAPolicy)

	body := `{"nama":"SLA Korektif Prioritas Tinggi","jenis_tugas":"corrective","target_jam_respon":1,"target_jam_selesai":8}`
	req := httptest.NewRequest("POST", "/sla-policies", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
}

// TestCreateSLAPolicy_InvalidJenis memastikan jenis tugas di luar enum ditolak.
func TestCreateSLAPolicy_InvalidJenis(t *testing.T) {
	_, mock := setupMockGORM(t)

	app := newTestApp()
	app.Post("/sla-policies", withUserLocals(uuid.New().String(), "admin"), CreateSLAPolicy)

	body := `{"nama":"SLA","jenis_tugas":"invalid","target_jam_respon":1,"target_jam_selesai":8}`
	req := httptest.NewRequest("POST", "/sla-policies", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetSLAPolicies_Success menguji daftar kebijakan SLA terpaginasi.
func TestGetSLAPolicies_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	id := uuid.New()

	mock.ExpectQuery(`^SELECT count\(\*\) FROM "sla_policies"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`SELECT \* FROM "sla_policies"`).
		WillReturnRows(slaPolicyRows(id, "SLA Korektif", "corrective", 1, 8))

	app := newTestApp()
	app.Get("/sla-policies", withUserLocals(uuid.New().String(), "admin"), GetSLAPolicies)

	req := httptest.NewRequest("GET", "/sla-policies?page=1&limit=10", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetSLAPolicyByID_Success menguji detail kebijakan SLA.
func TestGetSLAPolicyByID_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	id := uuid.New()

	mock.ExpectQuery(`SELECT \* FROM "sla_policies"`).
		WillReturnRows(slaPolicyRows(id, "SLA Korektif", "corrective", 1, 8))

	app := newTestApp()
	app.Get("/sla-policies/:id", withUserLocals(uuid.New().String(), "admin"), GetSLAPolicyByID)

	req := httptest.NewRequest("GET", "/sla-policies/"+id.String(), nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	data := result["data"].(map[string]interface{})
	assert.Equal(t, "SLA Korektif", data["nama"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetSLAPolicyByID_NotFound memastikan 404 untuk id tidak ada.
func TestGetSLAPolicyByID_NotFound(t *testing.T) {
	_, mock := setupMockGORM(t)

	mock.ExpectQuery(`SELECT \* FROM "sla_policies"`).
		WillReturnError(gorm.ErrRecordNotFound)

	app := newTestApp()
	app.Get("/sla-policies/:id", withUserLocals(uuid.New().String(), "admin"), GetSLAPolicyByID)

	req := httptest.NewRequest("GET", "/sla-policies/"+uuid.New().String(), nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestUpdateSLAPolicy_Success menguji pembaruan kebijakan SLA.
func TestUpdateSLAPolicy_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	id := uuid.New()

	mock.ExpectQuery(`SELECT \* FROM "sla_policies"`).
		WillReturnRows(slaPolicyRows(id, "SLA Korektif", "corrective", 1, 8))
	mock.ExpectExec(`^UPDATE "sla_policies" SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	app := newTestApp()
	app.Put("/sla-policies/:id", withUserLocals(uuid.New().String(), "admin"), UpdateSLAPolicy)

	body := `{"target_jam_selesai":4}`
	req := httptest.NewRequest("PUT", "/sla-policies/"+id.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
}

// TestDeleteSLAPolicy_Success menguji soft delete kebijakan SLA.
func TestDeleteSLAPolicy_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	id := uuid.New()

	mock.ExpectQuery(`SELECT \* FROM "sla_policies"`).
		WillReturnRows(slaPolicyRows(id, "SLA Korektif", "corrective", 1, 8))
	mock.ExpectExec(`^UPDATE "sla_policies" SET "deleted_at"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	app := newTestApp()
	app.Delete("/sla-policies/:id", withUserLocals(uuid.New().String(), "admin"), DeleteSLAPolicy)

	req := httptest.NewRequest("DELETE", "/sla-policies/"+id.String(), nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetDashboardSLA_Success menguji dashboard SLA admin.
func TestGetDashboardSLA_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	logID, cctvID, techID := uuid.New(), uuid.New(), uuid.New()

	// Query agregat SLA (JOIN maintenance_logs → maintenance_schedules → sla_policies).
	rows := sqlmock.NewRows([]string{
		"id", "cctv_id", "cctv_name", "cctv_code", "technician_id", "technician_name",
		"action_taken", "status", "created_at", "completed_at", "target_jam_selesai", "jenis_tugas",
	}).AddRow(
		logID.String(), cctvID.String(), "CCTV 1", "CCTV-001", techID.String(), "Teknisi A",
		"Perbaikan lensa", "completed", time.Now().Add(-2*time.Hour), time.Now(), 8, "corrective",
	)
	mock.ExpectQuery(`SELECT ml\.id, ml\.cctv_id.*FROM maintenance_logs AS ml`).
		WillReturnRows(rows)

	app := newTestApp()
	app.Get("/dashboard/sla", withUserLocals(uuid.New().String(), "admin"), GetDashboardSLA)

	req := httptest.NewRequest("GET", "/dashboard/sla", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	data := result["data"].(map[string]interface{})
	assert.NotNil(t, data["monthly_compliance"])
	assert.NotNil(t, data["violations"])
	assert.NotNil(t, data["technician_ranking"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetDashboardSLA_Forbidden memastikan teknisi ditolak (403).
func TestGetDashboardSLA_Forbidden(t *testing.T) {
	_, mock := setupMockGORM(t)

	app := newTestApp()
	app.Get("/dashboard/sla", withUserLocals(uuid.New().String(), "teknisi"), GetDashboardSLA)

	req := httptest.NewRequest("GET", "/dashboard/sla", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCheckSLADeadlines_Success menguji goroutine pemeriksa deadline SLA tanpa
// panic ketika tidak ada tugas.
func TestCheckSLADeadlines_Success(t *testing.T) {
	db, mock := setupMockGORM(t)

	mock.ExpectQuery(`SELECT ml\.id, ml\.cctv_id.*FROM maintenance_logs AS ml`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "cctv_id", "cctv_name", "action_taken", "technician_id",
			"created_at", "target_jam_selesai", "jenis_tugas",
		}))

	// Jalankan tanpa panic; tidak ada tugas → tidak ada broadcast.
	CheckSLADeadlines(db)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestAudit_SLAComplianceMustExcludeUnfinishedBeforeDeadline memastikan tugas
// aktif sebelum deadline dipisahkan sebagai pending dan tidak menurunkan
// persentase kepatuhan bulanan maupun ranking teknisi (Issue #176 / M03).
func TestAudit_SLAComplianceMustExcludeUnfinishedBeforeDeadline(t *testing.T) {
	cols := []string{
		"id", "cctv_id", "cctv_name", "cctv_code", "technician_id", "technician_name",
		"action_taken", "status", "created_at", "completed_at", "target_jam_selesai", "jenis_tugas",
	}

	t.Run("pending_only_task_is_separated_and_not_counted_as_violation", func(t *testing.T) {
		_, mock := setupMockGORM(t)
		now := time.Now()
		// Anchor pada pertengahan bulan berjalan (atau minimal beberapa jam lalu di bulan yang sama)
		createdAt := time.Date(now.Year(), now.Month(), 1, 1, 0, 0, 0, now.Location())
		if now.Sub(createdAt) > 12*time.Hour {
			createdAt = now.Add(-1 * time.Hour)
		}
		techID := uuid.New()

		rows := sqlmock.NewRows(cols).AddRow(
			uuid.New().String(), uuid.New().String(), "CCTV Gate 1", "CAM-01",
			techID.String(), "Teknisi Pending",
			"Pengecekan rutin", "ongoing", createdAt, nil, 24, "preventive",
		)
		mock.ExpectQuery(`SELECT ml\.id, ml\.cctv_id.*FROM maintenance_logs AS ml`).
			WillReturnRows(rows)

		app := newTestApp()
		app.Get("/dashboard/sla", withUserLocals(uuid.New().String(), "admin"), GetDashboardSLA)

		req := httptest.NewRequest("GET", "/dashboard/sla?months=1", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		data := result["data"].(map[string]interface{})

		monthly := data["monthly_compliance"].([]interface{})
		assert.Len(t, monthly, 1)
		currMonth := monthly[0].(map[string]interface{})
		assert.Equal(t, float64(1), currMonth["total"])
		assert.Equal(t, float64(1), currMonth["pending"])
		assert.Equal(t, float64(0), currMonth["assessed"])
		assert.Equal(t, float64(0), currMonth["completed_in_target"])
		assert.Equal(t, float64(0), currMonth["violated"])
		assert.Equal(t, float64(0), currMonth["compliance_rate"])

		ranking := data["technician_ranking"].([]interface{})
		assert.Len(t, ranking, 1)
		techRank := ranking[0].(map[string]interface{})
		assert.Equal(t, float64(1), techRank["total_tasks"])
		assert.Equal(t, float64(1), techRank["pending"])
		assert.Equal(t, float64(0), techRank["assessed"])
		assert.Equal(t, float64(0), techRank["completed_in_target"])
		assert.Equal(t, float64(0), techRank["violated"])
		assert.Equal(t, float64(0), techRank["compliance_rate"])

		violations := data["violations"].([]interface{})
		assert.Empty(t, violations)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("pending_on_time_late_and_active_violation_in_monthly_and_ranking", func(t *testing.T) {
		_, mock := setupMockGORM(t)
		now := time.Now()
		// Gunakan base waktu di bulan berjalan agar seluruh log tercatat di slot bulan terakhir.
		monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		baseCreated := monthStart.Add(1 * time.Minute)

		techAID := uuid.New()
		techBID := uuid.New()

		// 1) Teknisi A - Selesai tepat waktu (target 8 jam, selesai dalam 2 jam) -> compliant
		techAOnTimeCreated := baseCreated
		techAOnTimeDone := techAOnTimeCreated.Add(2 * time.Hour)

		// 2) Teknisi A - Ongoing baru dibuat, belum melewati deadline (target 720 jam) -> pending
		techAPendingCreated := baseCreated.Add(5 * time.Minute)

		// 3) Teknisi B - Selesai tepat waktu (target 8 jam, selesai dalam 3 jam) -> compliant
		techBOnTimeCreated := baseCreated.Add(10 * time.Minute)
		techBOnTimeDone := techBOnTimeCreated.Add(3 * time.Hour)

		// 4) Teknisi B - Selesai terlambat (target 4 jam, selesai dalam 10 jam) -> violated
		techBLateCreated := baseCreated.Add(15 * time.Minute)
		techBLateDone := techBLateCreated.Add(10 * time.Hour)

		// 5) Teknisi B - Pelanggaran aktif (ongoing melewati deadline; dibuat di awal bulan sebelumnya dgn target 1 jam)
		// Agar tetap di bulan berjalan даже jika tes berjalan tanggal 1 jam 01:00, buat created_at = now - 2 jam (jika masih di bulan yang sama)
		// atau baseCreated dengan target 1 jam jika now > baseCreated + 1 jam.
		activeViolationCreated := now.Add(-3 * time.Hour)
		if activeViolationCreated.Before(monthStart) {
			// Bila tes berjalan pada jam 00:00-02:59 tanggal 1, geser ke slot bulan lalu dan minta months=2
			activeViolationCreated = monthStart.Add(-3 * time.Hour)
		}

		// 6) Teknisi B - Scheduled sebelum deadline (target 720 jam) -> pending
		techBPendingCreated := baseCreated.Add(20 * time.Minute)

		// Pastikan kasus pelanggaran aktif diuji di bulan berjalan bila memungkinkan,
		// atau gunakan created_at = now.Add(-2*time.Hour) selama sama bulannya.
		sameMonthActiveViolation := activeViolationCreated.Month() == now.Month()
		if !sameMonthActiveViolation {
			// Tanggal 1 dini hari: gunakan created_at = monthStart dan status = "failed" atau gunakan bulan berjalan jika memungkinkan.
			// Namun dengan target_jam_selesai = 1, jika now >= monthStart + 1h01m (mis. jam 01:02 WITA), now.Add(-61*time.Minute) masih di bulan ini!
			if now.Sub(monthStart) > 61*time.Minute {
				activeViolationCreated = now.Add(-61 * time.Minute)
				sameMonthActiveViolation = true
			}
		}

		rows := sqlmock.NewRows(cols).
			AddRow(
				uuid.New().String(), uuid.New().String(), "CCTV A1", "CAM-A1",
				techAID.String(), "Teknisi A",
				"Perbaikan konektor", "completed", techAOnTimeCreated, techAOnTimeDone, 8, "corrective",
			).
			AddRow(
				uuid.New().String(), uuid.New().String(), "CCTV A2", "CAM-A2",
				techAID.String(), "Teknisi A",
				"Pembersihan rutin", "ongoing", techAPendingCreated, nil, 720, "preventive",
			).
			AddRow(
				uuid.New().String(), uuid.New().String(), "CCTV B1", "CAM-B1",
				techBID.String(), "Teknisi B",
				"Penggantian adaptor", "completed", techBOnTimeCreated, techBOnTimeDone, 8, "corrective",
			).
			AddRow(
				uuid.New().String(), uuid.New().String(), "CCTV B2", "CAM-B2",
				techBID.String(), "Teknisi B",
				"Perbaikan fokus", "completed", techBLateCreated, techBLateDone, 4, "corrective",
			).
			AddRow(
				uuid.New().String(), uuid.New().String(), "CCTV B3", "CAM-B3",
				techBID.String(), "Teknisi B",
				"Cek jaringan darurat", "ongoing", activeViolationCreated, nil, 1, "corrective",
			).
			AddRow(
				uuid.New().String(), uuid.New().String(), "CCTV B4", "CAM-B4",
				techBID.String(), "Teknisi B",
				"Inspeksi terjadwal", "scheduled", techBPendingCreated, nil, 720, "preventive",
			)

		mock.ExpectQuery(`SELECT ml\.id, ml\.cctv_id.*FROM maintenance_logs AS ml`).
			WillReturnRows(rows)

		app := newTestApp()
		app.Get("/dashboard/sla", withUserLocals(uuid.New().String(), "admin"), GetDashboardSLA)

		req := httptest.NewRequest("GET", "/dashboard/sla?months=2", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		data := result["data"].(map[string]interface{})

		// Verifikasi ranking teknisi:
		// Teknisi A: 1 compliant, 1 pending -> assessed = 1, compliance_rate = 100% (tidak turun menjadi 50%!)
		// Teknisi B: 1 compliant, 2 violated (1 terlambat + 1 aktif overdue), 1 pending -> assessed = 3, compliance_rate = 33.3%
		ranking := data["technician_ranking"].([]interface{})
		assert.Len(t, ranking, 2)

		rank0 := ranking[0].(map[string]interface{})
		assert.Equal(t, techAID.String(), rank0["user_id"])
		assert.Equal(t, float64(2), rank0["total_tasks"])
		assert.Equal(t, float64(1), rank0["pending"])
		assert.Equal(t, float64(1), rank0["assessed"])
		assert.Equal(t, float64(1), rank0["completed_in_target"])
		assert.Equal(t, float64(0), rank0["violated"])
		assert.Equal(t, float64(100), rank0["compliance_rate"])

		rank1 := ranking[1].(map[string]interface{})
		assert.Equal(t, techBID.String(), rank1["user_id"])
		assert.Equal(t, float64(4), rank1["total_tasks"])
		assert.Equal(t, float64(1), rank1["pending"])
		assert.Equal(t, float64(3), rank1["assessed"])
		assert.Equal(t, float64(1), rank1["completed_in_target"])
		assert.Equal(t, float64(2), rank1["violated"])
		assert.Equal(t, float64(33.3), rank1["compliance_rate"])

		// Verifikasi rekap bulanan bulan berjalan:
		monthly := data["monthly_compliance"].([]interface{})
		assert.Len(t, monthly, 2)
		currMonth := monthly[1].(map[string]interface{})
		if sameMonthActiveViolation {
			assert.Equal(t, float64(6), currMonth["total"])
			assert.Equal(t, float64(2), currMonth["pending"])
			assert.Equal(t, float64(4), currMonth["assessed"])
			assert.Equal(t, float64(2), currMonth["completed_in_target"])
			assert.Equal(t, float64(2), currMonth["violated"])
			assert.Equal(t, float64(50), currMonth["compliance_rate"])
		}

		// Verifikasi daftar pelanggaran mencakup tugas selesai terlambat dan tugas aktif melewati deadline
		violations := data["violations"].([]interface{})
		assert.Len(t, violations, 2)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
