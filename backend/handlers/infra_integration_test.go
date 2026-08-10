package handlers

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// ---- P10 (Task 13.10): Retensi Webhook Delivery Log ----

// TestCleanupWebhookDeliveries memverifikasi query cleanup menghormati
// max_log_retention_days per endpoint.
func TestCleanupWebhookDeliveries(t *testing.T) {
	db, mock := setupMockGORM(t)

	mock.ExpectExec(`^DELETE FROM webhook_deliveries`).
		WillReturnResult(sqlmock.NewResult(0, 3))

	deleted, err := CleanupWebhookDeliveries(db)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), deleted)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestAdminCleanupWebhookDeliveries memverifikasi endpoint manual cleanup log.
func TestAdminCleanupWebhookDeliveries(t *testing.T) {
	_, mock := setupMockGORM(t)
	id, createdBy := uuid.New(), uuid.New()

	mock.ExpectQuery(`SELECT \* FROM "webhook_endpoints"`).
		WillReturnRows(webhookEndpointRows(id, "Endpoint A", "https://example.com/hook", []byte(`["cctv.status_changed"]`), createdBy))
	mock.ExpectExec(`^DELETE FROM "webhook_deliveries"`).
		WillReturnResult(sqlmock.NewResult(0, 5))

	app := newTestApp()
	app.Delete("/webhooks/:id/deliveries", withUserLocals(uuid.New().String(), "admin"), AdminCleanupWebhookDeliveries)

	req := httptest.NewRequest("DELETE", "/webhooks/"+id.String()+"/deliveries?older_than=7", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	assert.Equal(t, float64(5), result["deleted"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestAdminCleanupWebhookDeliveries_NotFound memastikan cleanup endpoint tidak
// ada → 404.
func TestAdminCleanupWebhookDeliveries_NotFound(t *testing.T) {
	_, mock := setupMockGORM(t)

	mock.ExpectQuery(`SELECT \* FROM "webhook_endpoints"`).
		WillReturnError(gorm.ErrRecordNotFound)

	app := newTestApp()
	app.Delete("/webhooks/:id/deliveries", withUserLocals(uuid.New().String(), "admin"), AdminCleanupWebhookDeliveries)

	req := httptest.NewRequest("DELETE", "/webhooks/"+uuid.New().String()+"/deliveries", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---- P14 (Task 13.14): Rotasi QR Code CCTV ----

// TestRotateQRCode_Success memverifikasi rotasi QR mengubah QrPayload.
func TestRotateQRCode_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	rows, id := cctvMockRow()

	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).WillReturnRows(rows)
	mock.ExpectExec(`^UPDATE "cctvs" SET`).WillReturnResult(sqlmock.NewResult(1, 1))

	app := newTestApp()
	app.Post("/cctv/:id/rotate-qr", withUserLocals(uuid.New().String(), "admin"), RotateQRCode)

	req := httptest.NewRequest("POST", "/cctv/"+id.String()+"/rotate-qr", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	data := result["data"].(map[string]interface{})
	// QrPayload harus berubah dari nilai awal mock.
	assert.NotEqual(t, "encrypted-payload", data["qr_payload"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestRotateQRCode_NotFound memastikan rotasi CCTV tidak ada → 404.
func TestRotateQRCode_NotFound(t *testing.T) {
	_, mock := setupMockGORM(t)

	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnError(gorm.ErrRecordNotFound)

	app := newTestApp()
	app.Post("/cctv/:id/rotate-qr", withUserLocals(uuid.New().String(), "admin"), RotateQRCode)

	req := httptest.NewRequest("POST", "/cctv/"+uuid.New().String()+"/rotate-qr", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// ---- P16 (Task 13.16): Regenerasi API Key ----

// TestRegenerateAPIKey_Success memverifikasi regenerasi key mengembalikan key
// mentah baru dan memperbarui hash.
func TestRegenerateAPIKey_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	id, createdBy := uuid.New(), uuid.New()

	mock.ExpectQuery(`SELECT \* FROM "api_keys"`).
		WillReturnRows(apiKeyRows(id, "Integrasi", "abc12345", []byte(`["cctv:read"]`), createdBy))
	mock.ExpectExec(`^UPDATE "api_keys" SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	app := newTestApp()
	app.Post("/api-keys/:id/regenerate", withUserLocals(uuid.New().String(), "admin"), RegenerateAPIKey)

	req := httptest.NewRequest("POST", "/api-keys/"+id.String()+"/regenerate", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	data := result["data"].(map[string]interface{})
	assert.NotEmpty(t, data["api_key"], "key mentah baru harus dikembalikan")
	// Prefix baru harus berbeda dari prefix lama (abc12345).
	assert.NotEqual(t, "abc12345", data["key_prefix"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestRegenerateAPIKey_NotFound memastikan regenerasi key tidak ada → 404.
func TestRegenerateAPIKey_NotFound(t *testing.T) {
	_, mock := setupMockGORM(t)

	mock.ExpectQuery(`SELECT \* FROM "api_keys"`).
		WillReturnError(gorm.ErrRecordNotFound)

	app := newTestApp()
	app.Post("/api-keys/:id/regenerate", withUserLocals(uuid.New().String(), "admin"), RegenerateAPIKey)

	req := httptest.NewRequest("POST", "/api-keys/"+uuid.New().String()+"/regenerate", nil)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}
