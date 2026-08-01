package handlers

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

// makeJPEG menghasilkan byte gambar JPEG kecil yang valid.
func makeJPEG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for x := 0; x < 8; x++ {
		for y := 0; y < 8; y++ {
			img.Set(x, y, color.RGBA{R: 200, G: 100, B: 50, A: 255})
		}
	}
	var buf bytes.Buffer
	assert.NoError(t, jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80}))
	return buf.Bytes()
}

func multipartRequest(t *testing.T, field, filename string, content []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile(field, filename)
	assert.NoError(t, err)
	_, err = part.Write(content)
	assert.NoError(t, err)
	assert.NoError(t, writer.Close())

	req := httptest.NewRequest("POST", "/api/upload", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func TestUploadFile_NoFile(t *testing.T) {
	app := newTestApp()
	app.Post("/api/upload", UploadFile)

	req := httptest.NewRequest("POST", "/api/upload", nil)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "error", result["status"])
}

func TestUploadFile_InvalidType(t *testing.T) {
	app := newTestApp()
	app.Post("/api/upload", UploadFile)

	// File teks biasa → magic bytes bukan gambar → ditolak
	req := multipartRequest(t, "file", "script.php", []byte("<?php echo 'not an image'; ?>"))

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Contains(t, result["message"].(string), "Tipe file tidak diizinkan")
}

func TestUploadFile_ValidJPEG(t *testing.T) {
	// Pastikan direktori uploads tersedia
	_ = os.MkdirAll(uploadsDir, 0o755)

	app := newTestApp()
	app.Post("/api/upload", UploadFile)

	req := multipartRequest(t, "file", "foto.png", makeJPEG(t)) // ekstensi menipu, tapi magic bytes JPEG valid

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])

	data := result["data"].(map[string]interface{})
	url := data["url"].(string)
	assert.Contains(t, url, "/api/uploads/")
	assert.Equal(t, "image/jpeg", data["type"])

	// Bersihkan file hasil upload
	filename := filepath.Base(url)
	_ = os.Remove(filepath.Join(uploadsDir, filename))
}

func TestServeUploadedFile_InvalidName(t *testing.T) {
	app := newTestApp()
	app.Get("/uploads/:filename", ServeUploadedFile)

	// Nama mengandung ".." (path traversal) → ditolak
	resp, err := app.Test(httptest.NewRequest("GET", "/uploads/abc..def.jpg", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestServeUploadedFile_BadExtension(t *testing.T) {
	app := newTestApp()
	app.Get("/uploads/:filename", ServeUploadedFile)

	resp, err := app.Test(httptest.NewRequest("GET", "/uploads/script.exe", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestServeUploadedFile_NotFound(t *testing.T) {
	app := newTestApp()
	app.Get("/uploads/:filename", ServeUploadedFile)

	resp, err := app.Test(httptest.NewRequest("GET", "/uploads/00000000-0000-0000-0000-000000000000.jpg", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "File tidak ditemukan", result["message"])
}
