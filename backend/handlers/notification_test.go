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

// notificationRows membuat baris mock tabel notifications sesuai urutan kolom GORM.
func notificationRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at", "user_id",
		"type", "title", "message", "reference_id", "reference_type", "is_read",
	})
}

func TestGetNotifications_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	userID := uuid.New()

	mock.ExpectQuery(`^SELECT count\(\*\) FROM "notifications"`).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	rows := notificationRows().
		AddRow(uuid.New(), time.Now(), time.Now(), nil, userID,
			"MAINTENANCE_ASSIGNED", "Tugas baru ditugaskan", "Silakan kerjakan.", nil, "maintenance-log", false)
	mock.ExpectQuery(`^SELECT \* FROM "notifications"`).
		WithArgs(userID, 10).
		WillReturnRows(rows)

	app := newTestApp()
	app.Get("/notifications", withUserLocals(userID.String(), "teknisi"), GetNotifications)

	resp, err := app.Test(httptest.NewRequest("GET", "/notifications", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	data := result["data"].(map[string]interface{})
	rowsData := data["rows"].([]interface{})
	assert.Len(t, rowsData, 1)
	pagination := data["pagination"].(map[string]interface{})
	assert.Equal(t, float64(1), pagination["total_rows"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetNotifications_UnreadFilter(t *testing.T) {
	_, mock := setupMockGORM(t)
	userID := uuid.New()

	// Filter unread=true menambah kondisi is_read = false pada kedua query
	mock.ExpectQuery(`^SELECT count\(\*\) FROM "notifications"`).
		WithArgs(userID, false).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`^SELECT \* FROM "notifications"`).
		WithArgs(userID, false, 10).
		WillReturnRows(notificationRows())

	app := newTestApp()
	app.Get("/notifications", withUserLocals(userID.String(), "teknisi"), GetNotifications)

	resp, err := app.Test(httptest.NewRequest("GET", "/notifications?unread=true", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetNotifications_NoIdentity(t *testing.T) {
	_, mock := setupMockGORM(t)

	app := newTestApp()
	app.Get("/notifications", GetNotifications)

	resp, err := app.Test(httptest.NewRequest("GET", "/notifications", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "error", result["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUnreadNotificationCount_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	userID := uuid.New()

	mock.ExpectQuery(`^SELECT count\(\*\) FROM "notifications"`).
		WithArgs(userID, false).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

	app := newTestApp()
	app.Get("/notifications/unread-count", withUserLocals(userID.String(), "admin"), GetUnreadNotificationCount)

	resp, err := app.Test(httptest.NewRequest("GET", "/notifications/unread-count", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, float64(3), result["data"].(map[string]interface{})["unread_count"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUnreadNotificationCount_NoIdentity(t *testing.T) {
	_, mock := setupMockGORM(t)

	app := newTestApp()
	app.Get("/notifications/unread-count", GetUnreadNotificationCount)

	resp, err := app.Test(httptest.NewRequest("GET", "/notifications/unread-count", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMarkNotificationRead_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	userID := uuid.New()
	notifID := uuid.New()

	rows := notificationRows().
		AddRow(notifID, time.Now(), time.Now(), nil, userID,
			"MAINTENANCE_ASSIGNED", "Tugas baru ditugaskan", "Silakan kerjakan.", nil, "maintenance-log", false)
	mock.ExpectQuery(`^SELECT \* FROM "notifications"`).
		WithArgs(notifID, 1).
		WillReturnRows(rows)

	mock.ExpectExec(`^UPDATE "notifications" SET "is_read"=\$1,"updated_at"=\$2`).
		WithArgs(true, sqlmock.AnyArg(), notifID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	app := newTestApp()
	app.Put("/notifications/:id/read", withUserLocals(userID.String(), "teknisi"), MarkNotificationRead)

	resp, err := app.Test(httptest.NewRequest("PUT", "/notifications/"+notifID.String()+"/read", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMarkNotificationRead_NotOwner(t *testing.T) {
	_, mock := setupMockGORM(t)
	callerID := uuid.New()
	ownerID := uuid.New()
	notifID := uuid.New()

	rows := notificationRows().
		AddRow(notifID, time.Now(), time.Now(), nil, ownerID,
			"MAINTENANCE_ASSIGNED", "Tugas baru ditugaskan", "Silakan kerjakan.", nil, "maintenance-log", false)
	mock.ExpectQuery(`^SELECT \* FROM "notifications"`).
		WithArgs(notifID, 1).
		WillReturnRows(rows)

	app := newTestApp()
	app.Put("/notifications/:id/read", withUserLocals(callerID.String(), "teknisi"), MarkNotificationRead)

	resp, err := app.Test(httptest.NewRequest("PUT", "/notifications/"+notifID.String()+"/read", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "error", result["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMarkNotificationRead_NotFound(t *testing.T) {
	_, mock := setupMockGORM(t)
	userID := uuid.New()
	notifID := uuid.New()

	mock.ExpectQuery(`^SELECT \* FROM "notifications"`).
		WithArgs(notifID, 1).
		WillReturnRows(notificationRows())

	app := newTestApp()
	app.Put("/notifications/:id/read", withUserLocals(userID.String(), "teknisi"), MarkNotificationRead)

	resp, err := app.Test(httptest.NewRequest("PUT", "/notifications/"+notifID.String()+"/read", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMarkNotificationRead_InvalidUUID(t *testing.T) {
	_, mock := setupMockGORM(t)

	app := newTestApp()
	app.Put("/notifications/:id/read", withUserLocals(uuid.NewString(), "teknisi"), MarkNotificationRead)

	resp, err := app.Test(httptest.NewRequest("PUT", "/notifications/not-a-uuid/read", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "error", result["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMarkAllNotificationsRead_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	userID := uuid.New()

	mock.ExpectExec(`^UPDATE "notifications" SET "is_read"=\$1,"updated_at"=\$2`).
		WithArgs(true, sqlmock.AnyArg(), userID, false).
		WillReturnResult(sqlmock.NewResult(1, 5))

	app := newTestApp()
	app.Put("/notifications/read-all", withUserLocals(userID.String(), "teknisi"), MarkAllNotificationsRead)

	resp, err := app.Test(httptest.NewRequest("PUT", "/notifications/read-all", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMarkAllNotificationsRead_NoIdentity(t *testing.T) {
	_, mock := setupMockGORM(t)

	app := newTestApp()
	app.Put("/notifications/read-all", MarkAllNotificationsRead)

	resp, err := app.Test(httptest.NewRequest("PUT", "/notifications/read-all", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

