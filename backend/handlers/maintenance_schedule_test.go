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

// TestAudit_MonthlyScheduleMustNotSkipFebruary memverifikasi perbaikan M07:
// tanggal bulanan 31 Januari tidak boleh melompat ke 3 Maret, melainkan di-clamp
// secara eksplisit ke akhir Februari (28 atau 29 Februari).
func TestAudit_MonthlyScheduleMustNotSkipFebruary(t *testing.T) {
	from := time.Date(2027, 1, 31, 9, 0, 0, 0, time.UTC)
	actual := nextRunForFrequency("monthly", from)
	assert.Equal(t, time.February, actual.Month(), "monthly from %s skips February: actual %s", from.Format(time.RFC3339), actual.Format(time.RFC3339))
	assert.Equal(t, 28, actual.Day(), "non-leap year Feb should clamp to day 28")
	assert.Equal(t, 9, actual.Hour())
	assert.Equal(t, 0, actual.Minute())
}

// TestNextRunForFrequency_EndOfMonthClamping menguji kebijakan akhir bulan
// di berbagai bulan dan tahun (termasuk tahun kabisat dan pergantian tahun).
func TestNextRunForFrequency_EndOfMonthClamping(t *testing.T) {
	tests := []struct {
		name      string
		freq      string
		from      time.Time
		wantYear  int
		wantMonth time.Month
		wantDay   int
		wantHour  int
		wantMin   int
	}{
		{
			name:      "Jan 31 non-kabisat (2027) -> Feb 28",
			freq:      "monthly",
			from:      time.Date(2027, 1, 31, 14, 30, 0, 0, time.UTC),
			wantYear:  2027,
			wantMonth: time.February,
			wantDay:   28,
			wantHour:  14,
			wantMin:   30,
		},
		{
			name:      "Jan 31 kabisat (2028) -> Feb 29",
			freq:      "monthly",
			from:      time.Date(2028, 1, 31, 10, 0, 0, 0, time.UTC),
			wantYear:  2028,
			wantMonth: time.February,
			wantDay:   29,
			wantHour:  10,
			wantMin:   0,
		},
		{
			name:      "Mar 31 -> Apr 30",
			freq:      "monthly",
			from:      time.Date(2027, 3, 31, 8, 15, 0, 0, time.UTC),
			wantYear:  2027,
			wantMonth: time.April,
			wantDay:   30,
			wantHour:  8,
			wantMin:   15,
		},
		{
			name:      "May 31 -> Jun 30",
			freq:      "monthly",
			from:      time.Date(2027, 5, 31, 8, 0, 0, 0, time.UTC),
			wantYear:  2027,
			wantMonth: time.June,
			wantDay:   30,
			wantHour:  8,
			wantMin:   0,
		},
		{
			name:      "Aug 31 -> Sep 30",
			freq:      "monthly",
			from:      time.Date(2027, 8, 31, 11, 0, 0, 0, time.UTC),
			wantYear:  2027,
			wantMonth: time.September,
			wantDay:   30,
			wantHour:  11,
			wantMin:   0,
		},
		{
			name:      "Oct 31 -> Nov 30",
			freq:      "monthly",
			from:      time.Date(2027, 10, 31, 16, 45, 0, 0, time.UTC),
			wantYear:  2027,
			wantMonth: time.November,
			wantDay:   30,
			wantHour:  16,
			wantMin:   45,
		},
		{
			name:      "Dec 31 pergantian tahun -> Jan 31 tahun berikutnya",
			freq:      "monthly",
			from:      time.Date(2027, 12, 31, 23, 59, 0, 0, time.UTC),
			wantYear:  2028,
			wantMonth: time.January,
			wantDay:   31,
			wantHour:  23,
			wantMin:   59,
		},
		{
			name:      "Tanggal tengah bulan (Jan 15 -> Feb 15)",
			freq:      "monthly",
			from:      time.Date(2027, 1, 15, 9, 0, 0, 0, time.UTC),
			wantYear:  2027,
			wantMonth: time.February,
			wantDay:   15,
			wantHour:  9,
			wantMin:   0,
		},
		{
			name:      "Frekuensi mingguan (tambah 7 hari)",
			freq:      "weekly",
			from:      time.Date(2027, 1, 25, 9, 0, 0, 0, time.UTC),
			wantYear:  2027,
			wantMonth: time.February,
			wantDay:   1,
			wantHour:  9,
			wantMin:   0,
		},
		{
			name:      "Frekuensi harian (tambah 1 hari)",
			freq:      "daily",
			from:      time.Date(2027, 1, 31, 9, 0, 0, 0, time.UTC),
			wantYear:  2027,
			wantMonth: time.February,
			wantDay:   1,
			wantHour:  9,
			wantMin:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nextRunForFrequency(tt.freq, tt.from)
			assert.Equal(t, tt.wantYear, got.Year())
			assert.Equal(t, tt.wantMonth, got.Month())
			assert.Equal(t, tt.wantDay, got.Day())
			assert.Equal(t, tt.wantHour, got.Hour())
			assert.Equal(t, tt.wantMin, got.Minute())
		})
	}
}

