package handlers

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestGetAuditLogs_Success(t *testing.T) {
	_, mock := setupMockGORM(t)

	mock.ExpectQuery(`^SELECT count\(\*\) FROM "audit_logs"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	rows := sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at", "user_id",
		"username", "user_role", "action", "module", "description", "ip_address", "user_agent",
	}).AddRow(uuid.New(), time.Now(), time.Now(), nil, nil, "admin", "admin", "LOGIN_SUCCESS", "AUTH", "Pengguna berhasil login", "127.0.0.1", "test-agent")
	mock.ExpectQuery(`^SELECT \* FROM "audit_logs"`).
		WillReturnRows(rows)

	app := newTestApp()
	app.Get("/audit-logs", GetAuditLogs)

	resp, err := app.Test(httptest.NewRequest("GET", "/audit-logs", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	pagination := result["pagination"].(map[string]interface{})
	assert.Equal(t, float64(1), pagination["total_rows"])
	data := result["data"].([]interface{})
	assert.Len(t, data, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAuditLogs_ModuleFilter(t *testing.T) {
	_, mock := setupMockGORM(t)

	mock.ExpectQuery(`^SELECT count\(\*\) FROM "audit_logs"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`^SELECT \* FROM "audit_logs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "action", "module", "description"}))

	app := newTestApp()
	app.Get("/audit-logs", GetAuditLogs)

	resp, err := app.Test(httptest.NewRequest("GET", "/audit-logs?module=CCTV", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	pagination := result["pagination"].(map[string]interface{})
	assert.Equal(t, float64(0), pagination["total_rows"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestExportAuditLogs_Success(t *testing.T) {
	_, mock := setupMockGORM(t)

	mock.ExpectQuery(`^SELECT \* FROM "audit_logs"`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "created_at", "updated_at", "deleted_at", "user_id",
			"username", "user_role", "action", "module", "description", "ip_address", "user_agent",
		}))

	app := newTestApp()
	app.Get("/audit-logs/export", ExportAuditLogs)

	resp, err := app.Test(httptest.NewRequest("GET", "/audit-logs/export", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/csv", resp.Header.Get("Content-Type"))

	buf := make([]byte, 512)
	n, _ := resp.Body.Read(buf)
	// Header CSV selalu tertulis
	assert.Contains(t, string(buf[:n]), "ID,Waktu,Username")
	assert.NoError(t, mock.ExpectationsWereMet())
}
