package handlers

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
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

// makeLargeJPEG menghasilkan byte gambar JPEG berukuran besar dengan dimensi
// melebihi maxDimension sehingga memicu kompresi/resize pada saveCompressed.
// Rentang warna dibatasi agar ukuran file tetap di bawah batas upload 2MB namun
// masih lebih besar dari compressThreshold (300KB).
func makeLargeJPEG(t *testing.T) []byte {
	t.Helper()
	// Gambar noise acak 2200x2200 (dimensi > maxDimension 1920) dengan rentang
	// warna terbatas sehingga ukuran asli > 300KB namun < 2MB.
	width, height := 2200, 2200
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	rng := rand.New(rand.NewSource(42))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8(50 + rng.Intn(50)),
				G: uint8(100 + rng.Intn(50)),
				B: uint8(150 + rng.Intn(50)),
				A: 255,
			})
		}
	}
	var buf bytes.Buffer
	assert.NoError(t, jpeg.Encode(&buf, img, &jpeg.Options{Quality: 75}))
	return buf.Bytes()
}

func TestUploadCompress(t *testing.T) {
	// Pastikan direktori uploads & thumbnails tersedia
	_ = os.MkdirAll(uploadsDir, 0o755)
	_ = os.MkdirAll(thumbDir, 0o755)
	// Jamin kebersihan: hapus seluruh direktori uploads relatif CWD package test
	// setelah test selesai (termasuk bila assertion gagal di tengah) agar tidak
	// meninggalkan file artifact di backend/handlers/uploads/.
	defer func() { _ = os.RemoveAll(uploadsDir) }()

	app := newTestApp()
	app.Post("/api/upload", UploadFile)

	large := makeLargeJPEG(t)
	// Pastikan gambar sintetis memang berukuran cukup besar (> compressThreshold).
	assert.Greater(t, int64(len(large)), int64(compressThreshold), "gambar sintetis harus >300KB")

	req := multipartRequest(t, "file", "foto-besar.jpg", large)

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	// Pastikan header ukuran asli & hasil kompresi tersedia
	assert.NotEmpty(t, resp.Header.Get("X-Original-Size"))
	compHeader := resp.Header.Get("X-Compressed-Size")
	assert.NotEmpty(t, compHeader, "header X-Compressed-Size harus ada")
	compSize, convErr := strconv.ParseInt(compHeader, 10, 64)
	assert.NoError(t, convErr)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])

	data := result["data"].(map[string]interface{})
	assert.Equal(t, "image/jpeg", data["type"])

	// Respons harus berisi URL thumbnail
	thumbURL, ok := data["thumb_url"].(string)
	assert.True(t, ok, "data harus memuat thumb_url")
	assert.Contains(t, thumbURL, "/api/uploads/thumbnails/")

	// File fullsize tersimpan & berukuran terkompresi (dimensi diturunkan)
	url := data["url"].(string)
	filename := filepath.Base(url)
	savePath := filepath.Join(uploadsDir, filename)
	thumbPath := filepath.Join(thumbDir, filename)

	storedInfo, statErr := os.Stat(savePath)
	assert.NoError(t, statErr)
	// Ukuran hasil kompresi harus sama dengan yang dilaporkan di header & respons.
	assert.Equal(t, compSize, storedInfo.Size())
	assert.Equal(t, compSize, int64(data["size"].(float64)))

	// Thumbnail harus terbuat
	thumbInfo, thumbErr := os.Stat(thumbPath)
	assert.NoError(t, thumbErr)
	assert.True(t, thumbInfo.Size() > 0, "thumbnail harus berisi data")

	// Bersihkan file hasil upload
	_ = os.Remove(savePath)
	_ = os.Remove(thumbPath)
}
