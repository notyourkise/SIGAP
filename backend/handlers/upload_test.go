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

	"strings"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
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

// H04: Mengganti image_url pada log pemeliharaan yang berisi path di luar folder
// uploads tidak boleh menghapus file di luar direktori unggahan.
func TestAudit_UpdateImageMustNotDeleteOutsideUploads(t *testing.T) {
	_, mock := setupMockGORM(t)
	id, techID, cctvID := uuid.New(), uuid.New(), uuid.New()

	sentinelName := "audit-owned-" + uuid.NewString() + ".txt"
	file, err := os.OpenFile(sentinelName, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	require.NoError(t, err)
	_, err = file.WriteString("audit sentinel only")
	require.NoError(t, err)
	require.NoError(t, file.Close())
	t.Cleanup(func() { _ = os.Remove(sentinelName) })

	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "created_at", "updated_at", "deleted_at",
			"cctv_id", "technician_id", "action_taken", "notes", "status",
			"scheduled_date", "completed_at", "image_url",
		}).AddRow(id, time.Now(), time.Now(), nil, cctvID, techID, "Bersihkan lensa", "Catatan", "ongoing", nil, nil, "/"+sentinelName))
	mock.ExpectExec(`^UPDATE "maintenance_logs" SET`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(id))

	app := newTestApp()
	app.Put("/maintenance-logs/:id", withUserLocals(techID.String(), "teknisi"), UpdateMaintenanceLog)

	req := httptest.NewRequest("PUT", "/maintenance-logs/"+id.String(), strings.NewReader(`{"image_url":"/api/uploads/audit-new.jpg"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	_, statErr := os.Stat(sentinelName)
	assert.NoError(t, statErr, "file di luar direktori uploads tidak boleh terhapus saat mengganti lampiran")
	assert.NoError(t, mock.ExpectationsWereMet())
}

// H04: Input image_url dengan path bebas atau traversal wajib ditolak dengan 400.
func TestAttachmentSecurity_RejectUnsafeUploadPaths(t *testing.T) {
	app := newTestApp()
	techID := uuid.New().String()
	app.Post("/maintenance-logs", withUserLocals(techID, "teknisi"), CreateMaintenanceLog)
	app.Put("/maintenance-logs/:id", withUserLocals(techID, "teknisi"), UpdateMaintenanceLog)
	app.Post("/field-maintenance", withUserLocals(techID, "teknisi"), CreateFieldMaintenance)

	t.Run("CreateMaintenanceLog_RejectsArbitraryAndTraversalPaths", func(t *testing.T) {
		badURLs := []string{
			"/etc/passwd",
			"/api/uploads/../../secret.jpg",
			"/api/uploads/bukan-gambar.txt",
			"/uploads/../escape.jpg",
		}
		for _, badURL := range badURLs {
			payload := `{"cctv_id":"` + uuid.NewString() + `","action_taken":"Bersihkan lensa kamera","image_url":"` + badURL + `"}`
			req := httptest.NewRequest("POST", "/maintenance-logs", strings.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode, "harus menolak image_url=%s", badURL)
		}
	})

	t.Run("UpdateMaintenanceLog_RejectsTraversalPath", func(t *testing.T) {
		_, mock := setupMockGORM(t)
		id, cctvID := uuid.New(), uuid.New()
		techUUID := uuid.MustParse(techID)

		mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
			WillReturnRows(knownMaintenanceRows(id, techUUID, cctvID))

		req := httptest.NewRequest("PUT", "/maintenance-logs/"+id.String(), strings.NewReader(`{"image_url":"/api/uploads/../outside.jpg"}`))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("CreateFieldMaintenance_RejectsTraversalInsideUploadPrefix", func(t *testing.T) {
		payload := `{"cctv_id":"` + uuid.NewString() + `","checks":[{"name":"Jaringan internet kamera","result":"ok","note":""}],"images":[{"image_url":"/api/uploads/../../outside.jpg"}]}`
		req := httptest.NewRequest("POST", "/field-maintenance", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	})
}

// H04: Lampiran lama di dalam ./uploads tidak boleh dihapus bila penyimpanan DB
// gagal atau bila file masih direferensikan oleh record lain; dan hanya dihapus
// setelah update berhasil serta tidak ada referensi lain.
func TestAttachmentSecurity_PreserveOnSaveFailureAndSharedReference(t *testing.T) {
	require.NoError(t, os.MkdirAll(uploadsDir, 0o755))
	t.Cleanup(func() { _ = os.RemoveAll(uploadsDir) })

	t.Run("PreserveOldFileWhenDBSaveFails", func(t *testing.T) {
		_, mock := setupMockGORM(t)
		id, techID, cctvID := uuid.New(), uuid.New(), uuid.New()

		oldFilename := "save-fail-" + uuid.NewString() + ".jpg"
		oldFilePath := filepath.Join(uploadsDir, oldFilename)
		require.NoError(t, os.WriteFile(oldFilePath, makeJPEG(t), 0o600))
		t.Cleanup(func() { _ = os.Remove(oldFilePath) })

		mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "created_at", "updated_at", "deleted_at",
				"cctv_id", "technician_id", "action_taken", "notes", "status",
				"scheduled_date", "completed_at", "image_url",
			}).AddRow(id, time.Now(), time.Now(), nil, cctvID, techID, "Bersihkan lensa", "Catatan", "ongoing", nil, nil, "/api/uploads/"+oldFilename))
		mock.ExpectExec(`^UPDATE "maintenance_logs" SET`).
			WillReturnError(gorm.ErrInvalidDB)

		app := newTestApp()
		app.Put("/maintenance-logs/:id", withUserLocals(techID.String(), "teknisi"), UpdateMaintenanceLog)

		req := httptest.NewRequest("PUT", "/maintenance-logs/"+id.String(), strings.NewReader(`{"image_url":"/api/uploads/new-photo.jpg"}`))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)

		_, statErr := os.Stat(oldFilePath)
		assert.NoError(t, statErr, "file lama tidak boleh dihapus bila penyimpanan DB gagal")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("PreserveOldFileWhenStillReferencedByOtherRecord", func(t *testing.T) {
		_, mock := setupMockGORM(t)
		id, techID, cctvID := uuid.New(), uuid.New(), uuid.New()

		sharedFilename := "shared-" + uuid.NewString() + ".jpg"
		sharedFilePath := filepath.Join(uploadsDir, sharedFilename)
		require.NoError(t, os.WriteFile(sharedFilePath, makeJPEG(t), 0o600))
		t.Cleanup(func() { _ = os.Remove(sharedFilePath) })

		mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "created_at", "updated_at", "deleted_at",
				"cctv_id", "technician_id", "action_taken", "notes", "status",
				"scheduled_date", "completed_at", "image_url",
			}).AddRow(id, time.Now(), time.Now(), nil, cctvID, techID, "Bersihkan lensa", "Catatan", "ongoing", nil, nil, "/api/uploads/"+sharedFilename))
		mock.ExpectExec(`^UPDATE "maintenance_logs" SET`).
			WillReturnResult(sqlmock.NewResult(1, 1))
		// Masih direferensikan oleh 1 maintenance_log lain
		mock.ExpectQuery(`^SELECT count\(\*\) FROM "maintenance_logs"`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(id))

		app := newTestApp()
		app.Put("/maintenance-logs/:id", withUserLocals(techID.String(), "teknisi"), UpdateMaintenanceLog)

		req := httptest.NewRequest("PUT", "/maintenance-logs/"+id.String(), strings.NewReader(`{"image_url":"/api/uploads/new-photo.jpg"}`))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		_, statErr := os.Stat(sharedFilePath)
		assert.NoError(t, statErr, "file yang masih direferensikan record lain tidak boleh dihapus")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("DeleteOldFileWhenUnreferencedInsideUploads", func(t *testing.T) {
		_, mock := setupMockGORM(t)
		id, techID, cctvID := uuid.New(), uuid.New(), uuid.New()

		unrefFilename := "unref-" + uuid.NewString() + ".jpg"
		unrefFilePath := filepath.Join(uploadsDir, unrefFilename)
		require.NoError(t, os.WriteFile(unrefFilePath, makeJPEG(t), 0o600))
		t.Cleanup(func() { _ = os.Remove(unrefFilePath) })

		mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "created_at", "updated_at", "deleted_at",
				"cctv_id", "technician_id", "action_taken", "notes", "status",
				"scheduled_date", "completed_at", "image_url",
			}).AddRow(id, time.Now(), time.Now(), nil, cctvID, techID, "Bersihkan lensa", "Catatan", "ongoing", nil, nil, "/api/uploads/"+unrefFilename))
		mock.ExpectExec(`^UPDATE "maintenance_logs" SET`).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectQuery(`^SELECT count\(\*\) FROM "maintenance_logs"`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectQuery(`^SELECT count\(\*\) FROM "field_maintenance_images"`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectQuery(`^SELECT count\(\*\) FROM "users"`).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectQuery(`^SELECT \* FROM "maintenance_logs"`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(id))

		app := newTestApp()
		app.Put("/maintenance-logs/:id", withUserLocals(techID.String(), "teknisi"), UpdateMaintenanceLog)

		req := httptest.NewRequest("PUT", "/maintenance-logs/"+id.String(), strings.NewReader(`{"image_url":"/api/uploads/new-photo.jpg"}`))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		_, statErr := os.Stat(unrefFilePath)
		assert.True(t, os.IsNotExist(statErr), "file di dalam uploads yang sudah tidak direferensikan harus dihapus setelah update berhasil")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
