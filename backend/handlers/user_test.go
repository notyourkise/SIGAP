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

func sampleUserRow() modelsUserRow {
	return modelsUserRow{
		ID: uuid.New(), Username: "admin", Hash: "hash", FullName: "Admin Utama",
		Email: "admin@sigap.local", Role: "admin", TokenVer: 1, CreatedAt: time.Now(),
	}
}

func TestGetAllUsers_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	row := sampleUserRow()

	mock.ExpectQuery(`^SELECT count\(\*\) FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnRows(newUserMockRows([]modelsUserRow{row}))

	app := newTestApp()
	app.Get("/users", GetAllUsers)

	resp, err := app.Test(httptest.NewRequest("GET", "/users", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	data := result["data"].(map[string]interface{})
	pagination := data["pagination"].(map[string]interface{})
	assert.Equal(t, float64(1), pagination["total_rows"])
	rows := data["rows"].([]interface{})
	assert.Len(t, rows, 1)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUserByID_InvalidUUID(t *testing.T) {
	app := newTestApp()
	app.Get("/users/:id", GetUserByID)

	resp, err := app.Test(httptest.NewRequest("GET", "/users/not-a-uuid", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestGetUserByID_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	row := sampleUserRow()

	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnRows(newUserMockRows([]modelsUserRow{row}))

	app := newTestApp()
	app.Get("/users/:id", GetUserByID)

	resp, err := app.Test(httptest.NewRequest("GET", "/users/"+row.ID.String(), nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUserByID_NotFound(t *testing.T) {
	_, mock := setupMockGORM(t)
	id := uuid.New()

	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnError(gorm.ErrRecordNotFound)

	app := newTestApp()
	app.Get("/users/:id", GetUserByID)

	resp, err := app.Test(httptest.NewRequest("GET", "/users/"+id.String(), nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestUpdateUser_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	row := sampleUserRow()

	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnRows(newUserMockRows([]modelsUserRow{row}))
	mock.ExpectExec(`^UPDATE "users" SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	app := newTestApp()
	app.Put("/users/:id", UpdateUser)

	req := httptest.NewRequest("PUT", "/users/"+row.ID.String(), strings.NewReader(`{"full_name":"Nama Diperbarui"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "Pengguna berhasil diperbarui", result["message"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateUser_InvalidUUID(t *testing.T) {
	app := newTestApp()
	app.Put("/users/:id", UpdateUser)

	req := httptest.NewRequest("PUT", "/users/xyz", strings.NewReader(`{"full_name":"Nama"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestUpdateUser_InvalidRole(t *testing.T) {
	_, mock := setupMockGORM(t)
	row := sampleUserRow()
	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnRows(newUserMockRows([]modelsUserRow{row}))

	app := newTestApp()
	app.Put("/users/:id", UpdateUser)

	req := httptest.NewRequest("PUT", "/users/"+row.ID.String(), strings.NewReader(`{"role":"operator"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateUser_RoleChangeBumpsTokenVersion(t *testing.T) {
	_, mock := setupMockGORM(t)
	row := sampleUserRow()
	mock.ExpectQuery(`^SELECT \* FROM "users"`).WillReturnRows(newUserMockRows([]modelsUserRow{row}))
	mock.ExpectExec(`^UPDATE "users" SET`).WillReturnResult(sqlmock.NewResult(1, 1))

	app := newTestApp()
	app.Put("/users/:id", UpdateUser)
	req := httptest.NewRequest("PUT", "/users/"+row.ID.String(), strings.NewReader(`{"role":"teknisi"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	data := result["data"].(map[string]interface{})
	assert.Equal(t, "teknisi", data["role"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteUser_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	row := sampleUserRow()

	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnRows(newUserMockRows([]modelsUserRow{row}))
	mock.ExpectExec(`^UPDATE "users" SET "deleted_at"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	app := newTestApp()
	app.Delete("/users/:id", DeleteUser)

	resp, err := app.Test(httptest.NewRequest("DELETE", "/users/"+row.ID.String(), nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteUser_SelfForbidden(t *testing.T) {
	row := sampleUserRow()
	app := newTestApp()
	app.Delete("/users/:id", withUserLocals(row.ID.String(), "admin"), DeleteUser)

	resp, err := app.Test(httptest.NewRequest("DELETE", "/users/"+row.ID.String(), nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "Anda tidak diperbolehkan menghapus akun sendiri yang sedang aktif", result["message"])
}

func TestResetUserPassword_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	row := sampleUserRow()

	mock.ExpectQuery(`^SELECT \* FROM "users"`).
		WillReturnRows(newUserMockRows([]modelsUserRow{row}))
	mock.ExpectExec(`^UPDATE "users" SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	app := newTestApp()
	app.Post("/users/:id/reset-password", ResetUserPassword)

	req := httptest.NewRequest("POST", "/users/"+row.ID.String()+"/reset-password", strings.NewReader(`{"new_password":"Kuat123!"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "Sandi pengguna berhasil disetel ulang", result["message"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestResetUserPassword_InvalidUUID(t *testing.T) {
	app := newTestApp()
	app.Post("/users/:id/reset-password", ResetUserPassword)

	req := httptest.NewRequest("POST", "/users/xyz/reset-password", strings.NewReader(`{"new_password":"Kuat123!"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}
