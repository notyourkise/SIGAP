package handlers

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/SherClockHolmes/webpush-go"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestGetPushPublicKey_Success(t *testing.T) {
	setupMockGORM(t)
	privateKey, publicKey, err := webpush.GenerateVAPIDKeys()
	assert.NoError(t, err)
	t.Setenv("VAPID_PUBLIC_KEY", publicKey)
	t.Setenv("VAPID_PRIVATE_KEY", privateKey)
	t.Setenv("VAPID_EMAIL", "admin@sigap.local")

	app := newTestApp()
	app.Get("/push/public-key", withUserLocals(uuid.New().String(), "teknisi"), GetPushPublicKey)
	resp, err := app.Test(httptest.NewRequest("GET", "/push/public-key", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, publicKey, result["data"].(map[string]interface{})["public_key"])
}

func TestGetPushPublicKey_InvalidConfiguration(t *testing.T) {
	setupMockGORM(t)
	t.Setenv("VAPID_PUBLIC_KEY", "invalid")

	app := newTestApp()
	app.Get("/push/public-key", withUserLocals(uuid.New().String(), "teknisi"), GetPushPublicKey)
	resp, err := app.Test(httptest.NewRequest("GET", "/push/public-key", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)
}

func TestSubscribePush_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	userID := uuid.New()

	// Tidak ada subscription existing (First -> ErrRecordNotFound).
	mock.ExpectQuery(`^SELECT \* FROM "push_subscriptions"`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "created_at", "updated_at", "deleted_at", "user_id",
			"endpoint", "key_p256dh", "key_auth",
		}))
	// Create subscription baru (SkipDefaultTransaction => tanpa BEGIN/COMMIT).
	mock.ExpectExec(`^INSERT INTO "push_subscriptions"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	app := newTestApp()
	app.Post("/push/subscribe", withUserLocals(userID.String(), "teknisi"), SubscribePush)

	body := `{"endpoint":"https://push.example.com/sub/abc","key_p256dh":"BASE64KEY","key_auth":"BASE64AUTH"}`
	req := httptest.NewRequest("POST", "/push/subscribe", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSubscribePush_UpsertExisting(t *testing.T) {
	_, mock := setupMockGORM(t)
	userID := uuid.New()
	subID := uuid.New()

	// Subscription existing ditemukan (Unscoped First).
	rows := sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at", "user_id",
		"endpoint", "key_p256dh", "key_auth",
	}).
		AddRow(subID, time.Now(), time.Now(), nil, userID, "https://push.example.com/sub/abc", "OLDKEY", "OLDAUTH")
	mock.ExpectQuery(`^SELECT \* FROM "push_subscriptions"`).
		WillReturnRows(rows)

	// Update kunci via Unscoped().Model().Updates() (tanpa BEGIN/COMMIT).
	mock.ExpectExec(`^UPDATE "push_subscriptions" SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	app := newTestApp()
	app.Post("/push/subscribe", withUserLocals(userID.String(), "teknisi"), SubscribePush)

	body := `{"endpoint":"https://push.example.com/sub/abc","key_p256dh":"NEWKEY","key_auth":"NEWAUTH"}`
	req := httptest.NewRequest("POST", "/push/subscribe", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSubscribePush_Validation(t *testing.T) {
	_, mock := setupMockGORM(t)
	userID := uuid.New()

	app := newTestApp()
	app.Post("/push/subscribe", withUserLocals(userID.String(), "teknisi"), SubscribePush)

	// Payload tidak lengkap -> 400, tanpa query DB.
	body := `{"endpoint":"","key_p256dh":"","key_auth":""}`
	resp, err := app.Test(httptest.NewRequest("POST", "/push/subscribe", bytes.NewBufferString(body)))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSubscribePush_Unauthorized(t *testing.T) {
	setupMockGORM(t)

	app := newTestApp()
	// Tanpa middleware withUserLocals -> user_id kosong.
	app.Post("/push/subscribe", SubscribePush)

	body := `{"endpoint":"https://push.example.com/sub/abc","key_p256dh":"K","key_auth":"A"}`
	req := httptest.NewRequest("POST", "/push/subscribe", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestUnsubscribePush_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	userID := uuid.New()

	// Delete subscription (soft delete) — tanpa BEGIN/COMMIT.
	mock.ExpectExec(`^UPDATE "push_subscriptions" SET "deleted_at"`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	app := newTestApp()
	app.Delete("/push/unsubscribe", withUserLocals(userID.String(), "teknisi"), UnsubscribePush)

	body := `{"endpoint":"https://push.example.com/sub/abc"}`
	req := httptest.NewRequest("DELETE", "/push/unsubscribe", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUnsubscribePush_MissingEndpoint(t *testing.T) {
	_, mock := setupMockGORM(t)
	userID := uuid.New()

	app := newTestApp()
	app.Delete("/push/unsubscribe", withUserLocals(userID.String(), "teknisi"), UnsubscribePush)

	body := `{}`
	resp, err := app.Test(httptest.NewRequest("DELETE", "/push/unsubscribe", bytes.NewBufferString(body)))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}
