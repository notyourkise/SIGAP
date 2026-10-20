package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func postJSON(t *testing.T, app *fiber.App, path, body string) (*http.Response, map[string]interface{}) {
	t.Helper()
	req := httptest.NewRequest("POST", path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)

	var payload map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&payload))
	return resp, payload
}

func TestBodyParseError_InvalidUUIDIncludesFieldAndReason(t *testing.T) {
	app := newTestApp()
	app.Post("/cctv", CreateCCTV)

	resp, payload := postJSON(t, app, "/cctv", `{"zone_id":"not-a-uuid"}`)

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	message, _ := payload["message"].(string)
	assert.Contains(t, message, `field "zone_id"`)
	assert.Contains(t, message, "bukan UUID yang valid")
	assert.Contains(t, message, "not-a-uuid")

	detail, ok := payload["detail"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "zone_id", detail["field"])
	assert.Contains(t, detail["reason"], "bukan UUID yang valid")
	assert.Equal(t, "not-a-uuid", detail["value"])
}

func TestBodyParseError_WrongTypeIncludesExpectedAndReceived(t *testing.T) {
	app := newTestApp()
	app.Post("/cctv", CreateCCTV)

	resp, payload := postJSON(t, app, "/cctv", `{"nama":123}`)

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	message, _ := payload["message"].(string)
	assert.Contains(t, message, `field "nama"`)
	assert.Contains(t, message, "diharapkan string")
	assert.Contains(t, message, "diterima number")
	assert.Contains(t, message, "123")
}

func TestBodyParseError_MalformedJSONIncludesLineAndColumn(t *testing.T) {
	app := newTestApp()
	app.Post("/cctv", CreateCCTV)

	resp, payload := postJSON(t, app, "/cctv", `{bad}`)

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	message, _ := payload["message"].(string)
	assert.Contains(t, message, "JSON tidak dapat dibaca")
	assert.Contains(t, message, "baris 1")
	assert.Contains(t, message, "kolom")
}

func TestBodyParseError_EmptyBody(t *testing.T) {
	app := newTestApp()
	app.Post("/cctv", CreateCCTV)

	resp, payload := postJSON(t, app, "/cctv", "")

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	message, _ := payload["message"].(string)
	assert.Contains(t, message, "body request kosong")
}

func TestBodyParseError_RedactsSensitiveValue(t *testing.T) {
	app := newTestApp()
	app.Post("/cctv", CreateCCTV)

	resp, payload := postJSON(t, app, "/cctv", `{"rtsp_password":12345}`)

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	message, _ := payload["message"].(string)
	assert.Contains(t, message, `field "rtsp_password"`)
	assert.NotContains(t, message, "12345")
	assert.Contains(t, message, "disembunyikan")
}
