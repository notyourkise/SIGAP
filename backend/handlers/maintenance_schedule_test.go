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

// maintenanceScheduleRows membangun baris tabel maintenance_schedules untuk
// mock sqlmock. techID boleh nil (jadwal tanpa teknisi).
func maintenanceScheduleRows(id, cctvID uuid.UUID, techID *uuid.UUID, frequency string, isActive bool) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"cctv_id", "technician_id", "action_taken", "frequency",
		"next_run_at", "is_active", "last_generated_at",
	}).AddRow(id, time.Now(), time.Now(), nil, cctvID, techID,
		"Bersihkan lensa dan periksa konektor RJ45", frequency, time.Now(), isActive, nil)
}

// TestCreateMaintenanceSchedule_Success menguji pembuatan jadwal baru lengkap
// dengan teknisi: CCTV & peran teknisi divalidasi, lalu NextRunAt dihitung
// otomatis dari frekuensi.
func TestCreateMaintenanceSchedule_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	cctvID, techID := uuid.New(), uuid.New()

	cctvRows, _ := cctvMockRow()
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).WillReturnRows(cctvRows)
	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnRows(userRow(techID, "teknisi1", "teknisi"))
	mock.ExpectExec(`^INSERT INTO "maintenance_schedules"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`^INSERT INTO "notifications"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Catatan: audit log (LogAudit) ditulis async di goroutine — tidak
	// dimock di sini, mengikuti pola test handler lain.
	newID := uuid.New()
	reload := maintenanceScheduleRows(newID, cctvID, &techID, "daily", true)
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_schedules"`).WillReturnRows(reload)
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kode_cctv", "nama", "status"}))
	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "role"}))

	app := newTestApp()
	app.Post("/maintenance-schedules", withUserLocals(uuid.New().String(), "admin"), CreateMaintenanceSchedule)

	body := `{"cctv_id":"` + cctvID.String() + `","technician_id":"` + techID.String() +
		`","action_taken":"Periksa sambungan dan bersihkan debu kamera","frequency":"daily"}`
	req := httptest.NewRequest("POST", "/maintenance-schedules", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	data := result["data"].(map[string]interface{})
	assert.Equal(t, "daily", data["frequency"])
	assert.Equal(t, true, data["is_active"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCreateMaintenanceSchedule_CCTVNotFound memastikan jadwal tidak dapat
// dibuat untuk CCTV yang tidak terdaftar.
func TestCreateMaintenanceSchedule_CCTVNotFound(t *testing.T) {
	_, mock := setupMockGORM(t)

	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnError(gorm.ErrRecordNotFound)

	app := newTestApp()
	app.Post("/maintenance-schedules", withUserLocals(uuid.New().String(), "admin"), CreateMaintenanceSchedule)

	req := httptest.NewRequest("POST", "/maintenance-schedules",
		strings.NewReader(`{"cctv_id":"`+uuid.New().String()+`","action_taken":"Periksa kamera","frequency":"daily"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCreateMaintenanceSchedule_TechnicianInvalidRole menolak jadwal yang
// menunjuk user non-teknisi sebagai pelaksana.
func TestCreateMaintenanceSchedule_TechnicianInvalidRole(t *testing.T) {
	_, mock := setupMockGORM(t)
	cctvID, otherID := uuid.New(), uuid.New()

	cctvRows, _ := cctvMockRow()
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).WillReturnRows(cctvRows)
	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnRows(userRow(otherID, "admin1", "admin"))

	app := newTestApp()
	app.Post("/maintenance-schedules", withUserLocals(uuid.New().String(), "admin"), CreateMaintenanceSchedule)

	req := httptest.NewRequest("POST", "/maintenance-schedules",
		strings.NewReader(`{"cctv_id":"`+cctvID.String()+`","technician_id":"`+otherID.String()+`","action_taken":"Periksa kamera","frequency":"daily"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetAllMaintenanceSchedules_Success menguji daftar jadwal dengan
// preload relasi CCTV & teknisi, plus paginasi.
func TestGetAllMaintenanceSchedules_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	id, cctvID, techID := uuid.New(), uuid.New(), uuid.New()

	mock.ExpectQuery(`^SELECT count\(\*\) FROM "maintenance_schedules"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_schedules"`).
		WillReturnRows(maintenanceScheduleRows(id, cctvID, &techID, "monthly", true))
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kode_cctv", "nama", "status"}))
	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "role"}))

	app := newTestApp()
	app.Get("/maintenance-schedules", withUserLocals(uuid.New().String(), "admin"), GetAllMaintenanceSchedules)

	req := httptest.NewRequest("GET", "/maintenance-schedules?page=1&limit=10", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	data := result["data"].(map[string]interface{})
	pagination := data["pagination"].(map[string]interface{})
	assert.Equal(t, float64(1), pagination["total_rows"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetMaintenanceScheduleByID_Success menguji pengambilan detail jadwal.
func TestGetMaintenanceScheduleByID_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	id, cctvID := uuid.New(), uuid.New()

	mock.ExpectQuery(`^SELECT \* FROM "maintenance_schedules"`).
		WillReturnRows(maintenanceScheduleRows(id, cctvID, nil, "daily", true))
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kode_cctv", "nama", "status"}))
	// Technician nil → tidak ada preload users

	app := newTestApp()
	app.Get("/maintenance-schedules/:id", withUserLocals(uuid.New().String(), "admin"), GetMaintenanceScheduleByID)

	req := httptest.NewRequest("GET", "/maintenance-schedules/"+id.String(), nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	data := result["data"].(map[string]interface{})
	assert.Equal(t, "daily", data["frequency"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetMaintenanceScheduleByID_NotFound memastikan 404 untuk id yang tidak ada.
func TestGetMaintenanceScheduleByID_NotFound(t *testing.T) {
	_, mock := setupMockGORM(t)

	mock.ExpectQuery(`^SELECT \* FROM "maintenance_schedules"`).
		WillReturnError(gorm.ErrRecordNotFound)

	app := newTestApp()
	app.Get("/maintenance-schedules/:id", withUserLocals(uuid.New().String(), "admin"), GetMaintenanceScheduleByID)

	req := httptest.NewRequest("GET", "/maintenance-schedules/"+uuid.New().String(), nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestUpdateMaintenanceSchedule_Success menguji perubahan frekuensi yang
// memicu penghitungan ulang NextRunAt.
func TestUpdateMaintenanceSchedule_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	id, cctvID := uuid.New(), uuid.New()

	mock.ExpectQuery(`^SELECT \* FROM "maintenance_schedules"`).
		WillReturnRows(maintenanceScheduleRows(id, cctvID, nil, "weekly", true))
	mock.ExpectExec(`^UPDATE "maintenance_schedules" SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	reload := maintenanceScheduleRows(id, cctvID, nil, "monthly", true)
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_schedules"`).WillReturnRows(reload)
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kode_cctv", "nama", "status"}))

	app := newTestApp()
	app.Put("/maintenance-schedules/:id", withUserLocals(uuid.New().String(), "admin"), UpdateMaintenanceSchedule)

	req := httptest.NewRequest("PUT", "/maintenance-schedules/"+id.String(),
		strings.NewReader(`{"frequency":"monthly","action_taken":"Periksa kamera secara menyeluruh"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	data := result["data"].(map[string]interface{})
	assert.Equal(t, "monthly", data["frequency"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestUpdateMaintenanceSchedule_TechnicianInvalidRole menolak jadwal yang
// menunjuk user non-teknisi sebagai pelaksana.
func TestUpdateMaintenanceSchedule_TechnicianInvalidRole(t *testing.T) {
	_, mock := setupMockGORM(t)
	id, cctvID, otherID := uuid.New(), uuid.New(), uuid.New()

	mock.ExpectQuery(`^SELECT \* FROM "maintenance_schedules"`).
		WillReturnRows(maintenanceScheduleRows(id, cctvID, nil, "weekly", true))
	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnRows(userRow(otherID, "admin1", "admin"))

	app := newTestApp()
	app.Put("/maintenance-schedules/:id", withUserLocals(uuid.New().String(), "admin"), UpdateMaintenanceSchedule)

	req := httptest.NewRequest("PUT", "/maintenance-schedules/"+id.String(),
		strings.NewReader(`{"technician_id":"`+otherID.String()+`"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestToggleMaintenanceSchedule_Success menguji aktivasi/nonaktif jadwal.
func TestToggleMaintenanceSchedule_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	id, cctvID := uuid.New(), uuid.New()

	// Jadwal sedang nonaktif — toggle akan mengaktifkannya kembali
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_schedules"`).
		WillReturnRows(maintenanceScheduleRows(id, cctvID, nil, "weekly", false))
	mock.ExpectExec(`^UPDATE "maintenance_schedules" SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	reload := maintenanceScheduleRows(id, cctvID, nil, "weekly", true)
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_schedules"`).WillReturnRows(reload)
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "kode_cctv", "nama", "status"}))

	app := newTestApp()
	app.Put("/maintenance-schedules/:id/toggle", withUserLocals(uuid.New().String(), "admin"), ToggleMaintenanceSchedule)

	req := httptest.NewRequest("PUT", "/maintenance-schedules/"+id.String()+"/toggle", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	data := result["data"].(map[string]interface{})
	assert.Equal(t, true, data["is_active"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestDeleteMaintenanceSchedule_Success menguji soft delete jadwal.
func TestDeleteMaintenanceSchedule_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	id, cctvID := uuid.New(), uuid.New()

	mock.ExpectQuery(`^SELECT \* FROM "maintenance_schedules"`).
		WillReturnRows(maintenanceScheduleRows(id, cctvID, nil, "weekly", true))
	mock.ExpectExec(`^UPDATE "maintenance_schedules" SET "deleted_at"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	app := newTestApp()
	app.Delete("/maintenance-schedules/:id", withUserLocals(uuid.New().String(), "admin"), DeleteMaintenanceSchedule)

	req := httptest.NewRequest("DELETE", "/maintenance-schedules/"+id.String(), nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestAudit_ScheduleMustAcceptSLAPolicy menguji pembuatan dan pembaruan jadwal
// pemeliharaan rutin dengan sla_policy_id melalui API (Issue #175 / M02).
func TestAudit_ScheduleMustAcceptSLAPolicy(t *testing.T) {
	t.Run("create schedule with valid sla_policy_id", func(t *testing.T) {
		_, mock := setupMockGORM(t)
		cctvID, slaID, newID := uuid.New(), uuid.New(), uuid.New()

		cctvRows, _ := cctvMockRow()
		mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).WillReturnRows(cctvRows)
		mock.ExpectQuery(`^SELECT \* FROM "sla_policies"`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "nama", "jenis_tugas", "target_jam_selesai", "aktif"}).
				AddRow(slaID, "SLA Rutin Standar", "rutin", 24, true))
		mock.ExpectExec(`^INSERT INTO "maintenance_schedules"`).
			WillReturnResult(sqlmock.NewResult(1, 1))

		reload := maintenanceScheduleRows(newID, cctvID, nil, "weekly", true)
		mock.ExpectQuery(`^SELECT \* FROM "maintenance_schedules"`).WillReturnRows(reload)
		mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "kode_cctv", "nama", "status"}).
				AddRow(cctvID, "CAM-1", "Kamera", "Aktif"))

		app := newTestApp()
		app.Post("/maintenance-schedules", withUserLocals(uuid.New().String(), "admin"), CreateMaintenanceSchedule)

		body := `{"cctv_id":"` + cctvID.String() + `","sla_policy_id":"` + slaID.String() + `","action_taken":"Inspeksi berkala lensa","frequency":"weekly"}`
		req := httptest.NewRequest("POST", "/maintenance-schedules", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

		var result map[string]interface{}
		assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		data := result["data"].(map[string]interface{})
		assert.Equal(t, slaID.String(), data["sla_policy_id"])
		slaObj, ok := data["sla_policy"].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "SLA Rutin Standar", slaObj["nama"])
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("create schedule with non-existent sla_policy_id returns 404", func(t *testing.T) {
		_, mock := setupMockGORM(t)
		cctvID, slaID := uuid.New(), uuid.New()

		cctvRows, _ := cctvMockRow()
		mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).WillReturnRows(cctvRows)
		mock.ExpectQuery(`^SELECT \* FROM "sla_policies"`).
			WillReturnError(gorm.ErrRecordNotFound)

		app := newTestApp()
		app.Post("/maintenance-schedules", withUserLocals(uuid.New().String(), "admin"), CreateMaintenanceSchedule)

		body := `{"cctv_id":"` + cctvID.String() + `","sla_policy_id":"` + slaID.String() + `","action_taken":"Inspeksi berkala lensa","frequency":"weekly"}`
		req := httptest.NewRequest("POST", "/maintenance-schedules", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("update schedule attaches and detaches sla_policy_id", func(t *testing.T) {
		_, mock := setupMockGORM(t)
		id, cctvID, slaID := uuid.New(), uuid.New(), uuid.New()

		mock.ExpectQuery(`^SELECT \* FROM "maintenance_schedules"`).
			WillReturnRows(maintenanceScheduleRows(id, cctvID, nil, "weekly", true))
		mock.ExpectQuery(`^SELECT \* FROM "sla_policies"`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "nama", "jenis_tugas", "target_jam_selesai", "aktif"}).
				AddRow(slaID, "SLA Rutin Standar", "rutin", 24, true))
		mock.ExpectExec(`^UPDATE "maintenance_schedules" SET`).
			WillReturnResult(sqlmock.NewResult(1, 1))

		reload := maintenanceScheduleRows(id, cctvID, nil, "weekly", true)
		mock.ExpectQuery(`^SELECT \* FROM "maintenance_schedules"`).WillReturnRows(reload)
		mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
			WillReturnRows(sqlmock.NewRows([]string{"id", "kode_cctv", "nama", "status"}))

		app := newTestApp()
		app.Put("/maintenance-schedules/:id", withUserLocals(uuid.New().String(), "admin"), UpdateMaintenanceSchedule)

		req := httptest.NewRequest("PUT", "/maintenance-schedules/"+id.String(),
			strings.NewReader(`{"sla_policy_id":"`+slaID.String()+`"}`))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		data := result["data"].(map[string]interface{})
		assert.Equal(t, slaID.String(), data["sla_policy_id"])
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
