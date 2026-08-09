package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// webhookEndpointRows membangun baris tabel webhook_endpoints untuk mock.
func webhookEndpointRows(id uuid.UUID, name, url string, events []byte, createdBy uuid.UUID) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"name", "url", "events", "secret", "is_active", "created_by",
	}).AddRow(id, time.Now(), time.Now(), nil, name, url, string(events), "secret123", true, createdBy)
}

// TestCreateWebhookEndpoint_Success menguji pembuatan endpoint webhook.
func TestCreateWebhookEndpoint_Success(t *testing.T) {
	_, mock := setupMockGORM(t)

	// Events []string disimpan sebagai jsonb → GORM memakai INSERT ... RETURNING
	// "events". Kembalikan []byte agar GORM dapat mem-parse JSON ke *[]string.
	mock.ExpectQuery(`^INSERT INTO "webhook_endpoints"`).
		WillReturnRows(sqlmock.NewRows([]string{"events"}).AddRow([]byte(`["cctv.status_changed","maintenance.completed"]`)))

	app := newTestApp()
	app.Post("/webhooks", withUserLocals(uuid.New().String(), "admin"), CreateWebhookEndpoint)

	body := `{"name":"Endpoint A","url":"https://example.com/hook","events":["cctv.status_changed","maintenance.completed"],"secret":"rahasia123"}`
	req := httptest.NewRequest("POST", "/webhooks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCreateWebhookEndpoint_InvalidEvent memastikan event tidak valid ditolak.
func TestCreateWebhookEndpoint_InvalidEvent(t *testing.T) {
	_, mock := setupMockGORM(t)

	app := newTestApp()
	app.Post("/webhooks", withUserLocals(uuid.New().String(), "admin"), CreateWebhookEndpoint)

	body := `{"name":"Endpoint A","url":"https://example.com/hook","events":["bogus.event"],"secret":"rahasia123"}`
	req := httptest.NewRequest("POST", "/webhooks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCreateWebhookEndpoint_ValidationError memastikan URL/secret wajib.
func TestCreateWebhookEndpoint_ValidationError(t *testing.T) {
	_, mock := setupMockGORM(t)

	app := newTestApp()
	app.Post("/webhooks", withUserLocals(uuid.New().String(), "admin"), CreateWebhookEndpoint)

	req := httptest.NewRequest("POST", "/webhooks",
		strings.NewReader(`{"name":"X","url":"not-a-url","events":["cctv.status_changed"],"secret":"short"}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetAllWebhookEndpoints_Success menguji daftar endpoint webhook.
func TestGetAllWebhookEndpoints_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	id, createdBy := uuid.New(), uuid.New()

	mock.ExpectQuery(`^SELECT count\(\*\) FROM "webhook_endpoints"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`SELECT \* FROM "webhook_endpoints"`).
		WillReturnRows(webhookEndpointRows(id, "Endpoint A", "https://example.com/hook", []byte(`["cctv.status_changed"]`), createdBy))
	mock.ExpectQuery(`SELECT \* FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "full_name", "username"}).AddRow(createdBy, "Admin", "admin"))

	app := newTestApp()
	app.Get("/webhooks", withUserLocals(uuid.New().String(), "admin"), GetAllWebhookEndpoints)

	req := httptest.NewRequest("GET", "/webhooks?page=1&limit=10", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestUpdateWebhookEndpoint_Success menguji pembaruan endpoint.
func TestUpdateWebhookEndpoint_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	id, createdBy := uuid.New(), uuid.New()

	mock.ExpectQuery(`SELECT \* FROM "webhook_endpoints"`).
		WillReturnRows(webhookEndpointRows(id, "Endpoint A", "https://example.com/hook", []byte(`["cctv.status_changed"]`), createdBy))
	mock.ExpectExec(`^UPDATE "webhook_endpoints" SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	app := newTestApp()
	app.Put("/webhooks/:id", withUserLocals(uuid.New().String(), "admin"), UpdateWebhookEndpoint)

	body := `{"is_active":false,"events":["stream.down"]}`
	req := httptest.NewRequest("PUT", "/webhooks/"+id.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestDeleteWebhookEndpoint_Success menguji soft delete endpoint.
func TestDeleteWebhookEndpoint_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	id, createdBy := uuid.New(), uuid.New()

	mock.ExpectQuery(`SELECT \* FROM "webhook_endpoints"`).
		WillReturnRows(webhookEndpointRows(id, "Endpoint A", "https://example.com/hook", []byte(`["cctv.status_changed"]`), createdBy))
	mock.ExpectExec(`^UPDATE "webhook_endpoints" SET "deleted_at"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	app := newTestApp()
	app.Delete("/webhooks/:id", withUserLocals(uuid.New().String(), "admin"), DeleteWebhookEndpoint)

	req := httptest.NewRequest("DELETE", "/webhooks/"+id.String(), nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetWebhookDeliveries_Success menguji riwayat pengiriman.
func TestGetWebhookDeliveries_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	endpointID, deliveryID := uuid.New(), uuid.New()

	mock.ExpectQuery(`^SELECT count\(\*\) FROM "webhook_deliveries"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`SELECT \* FROM "webhook_deliveries"`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "created_at", "updated_at", "deleted_at",
			"webhook_endpoint_id", "event", "payload", "response_code", "response_body", "latency", "status", "attempt",
		}).AddRow(deliveryID, time.Now(), time.Now(), nil, endpointID, "cctv.status_changed", `{"cctv_id":"x"}`, 200, "ok", 15, "success", 1))

	app := newTestApp()
	app.Get("/webhooks/:id/deliveries", withUserLocals(uuid.New().String(), "admin"), GetWebhookDeliveries)

	req := httptest.NewRequest("GET", "/webhooks/"+endpointID.String()+"/deliveries", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestSignPayload memastikan HMAC-SHA256 signature sesuai.
func TestSignPayload(t *testing.T) {
	secret := "rahasia123"
	payload := []byte(`{"event":"test.ping"}`)

	sig := signPayload(payload, secret)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	assert.Equal(t, expected, sig)
	assert.Len(t, sig, 64)
}

// TestValidWebhookEvent memastikan hanya event terdaftar yang valid.
func TestValidWebhookEvent(t *testing.T) {
	assert.True(t, validWebhookEvent("cctv.status_changed"))
	assert.True(t, validWebhookEvent("maintenance.completed"))
	assert.True(t, validWebhookEvent("maintenance.overdue"))
	assert.True(t, validWebhookEvent("health_score.changed"))
	assert.True(t, validWebhookEvent("stream.down"))
	assert.False(t, validWebhookEvent("bogus.event"))
}
