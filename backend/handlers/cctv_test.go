package handlers

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"sigap-backend/utils"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

// cctvMockRow membangun sqlmock rows dengan seluruh kolom tabel cctvs
// berstatus "aktif" (varian status berbeda via cctvMockRowWithStatus).
func cctvMockRow() (*sqlmock.Rows, uuid.UUID) {
	return cctvMockRowWithStatus("aktif")
}

// cctvMockRowWithStatus membangun sqlmock rows CCTV dengan status tertentu —
// dipakai untuk menguji sinkronisasi status otomatis (Fase 5.2).
func cctvMockRowWithStatus(status string) (*sqlmock.Rows, uuid.UUID) {
	id := uuid.New()
	rows := sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"kode_cctv", "id_channel", "nama", "lokasi_area",
		"longitude", "latitude", "jenis_cctv", "sumber_listrik",
		"tipe_koneksi", "indoor_outdoor", "pemilik", "status",
		"ip_nvr", "ip_camera", "keterangan", "qr_payload", "health_score",
		"channel_port", "rtsp_password", "stream_enabled",
	}).AddRow(id, time.Now(), time.Now(), nil, "CCTV-001", "CH-01", "Kamera Depan", "Area Utama",
		117.1, -0.5, "Dome", "PLN", "Fiber", "outdoor", "Pemkot", status,
		"192.168.1.10", "192.168.1.11", "Aman", "encrypted-payload", 100, 19, "encrypted-rtsp", true)
	return rows, id
}

func TestGetCCTVByID_InvalidUUID(t *testing.T) {
	app := newTestApp()
	app.Get("/cctv/:id", GetCCTVByID)

	resp, err := app.Test(httptest.NewRequest("GET", "/cctv/not-a-uuid", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestGetCCTVByID_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	rows, id := cctvMockRow()

	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(rows)

	app := newTestApp()
	app.Get("/cctv/:id", GetCCTVByID)

	resp, err := app.Test(httptest.NewRequest("GET", "/cctv/"+id.String(), nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	data := result["data"].(map[string]interface{})
	assert.Equal(t, "Kamera Depan", data["nama"])
	assert.Equal(t, "aktif", data["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetCCTVByID_NotFound(t *testing.T) {
	_, mock := setupMockGORM(t)
	id := uuid.New()

	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnError(gorm.ErrRecordNotFound)

	app := newTestApp()
	app.Get("/cctv/:id", GetCCTVByID)

	resp, err := app.Test(httptest.NewRequest("GET", "/cctv/"+id.String(), nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestCreateCCTV_ValidationError(t *testing.T) {
	app := newTestApp()
	app.Post("/cctv", CreateCCTV)

	// Kode CCTV kurang dari 3 karakter
	payload := `{"kode_cctv":"a","nama":"Kamera","status":"aktif"}`
	req := httptest.NewRequest("POST", "/cctv", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "Validasi input gagal", result["message"])
}

func TestCreateCCTV_DuplicateKode(t *testing.T) {
	_, mock := setupMockGORM(t)
	rows, _ := cctvMockRow()

	// Kode sudah dipakai oleh data yang tidak di-soft-delete → 400
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(rows)

	app := newTestApp()
	app.Post("/cctv", CreateCCTV)

	payload := `{"kode_cctv":"CCTV-001","nama":"Kamera Baru","status":"aktif"}`
	req := httptest.NewRequest("POST", "/cctv", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "Kode CCTV sudah digunakan oleh perangkat lain", result["message"])
}

func TestCreateCCTV_Success(t *testing.T) {
	t.Setenv("AES_SECRET_KEY", "sigap_aes_test_secret_2026")
	_, mock := setupMockGORM(t)

	// Tidak ada kode duplikat
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnError(gorm.ErrRecordNotFound)
	mock.ExpectExec(`^INSERT INTO "cctvs"`).
		WillReturnResult(sqlmock.NewResult(1, 1))

	app := newTestApp()
	app.Post("/cctv", CreateCCTV)

	payload := `{"kode_cctv":"CCTV-009","nama":"Kamera Baru","lokasi_area":"Area Selatan","status":"aktif"}`
	req := httptest.NewRequest("POST", "/cctv", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	data := result["data"].(map[string]interface{})
	assert.Equal(t, "CCTV-009", data["kode_cctv"])
	// QrPayload harus terisi (hasil enkripsi AES dari UUID)
	assert.NotEmpty(t, data["qr_payload"])
	assert.Equal(t, float64(100), data["health_score"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestScanCCTV_InvalidPayload(t *testing.T) {
	t.Setenv("AES_SECRET_KEY", "sigap_aes_test_secret_2026")
	app := newTestApp()
	app.Post("/cctv/scan", ScanCCTV)

	payload := `{"qr_payload":"bukan-hex-terenkripsi"}`
	req := httptest.NewRequest("POST", "/cctv/scan", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "error", result["status"])
}

func TestScanCCTV_Success(t *testing.T) {
	t.Setenv("AES_SECRET_KEY", "sigap_aes_test_secret_2026")
	_, mock := setupMockGORM(t)
	rows, _ := cctvMockRow()

	cctvID := uuid.New()
	encrypted, err := utils.EncryptAES256(cctvID.String())
	assert.NoError(t, err)

	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).
		WillReturnRows(rows)

	app := newTestApp()
	app.Post("/cctv/scan", ScanCCTV)

	payload := `{"qr_payload":"` + encrypted + `"}`
	req := httptest.NewRequest("POST", "/cctv/scan", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "success", result["status"])
	data := result["data"].(map[string]interface{})
	assert.Equal(t, "Kamera Depan", data["nama"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDownloadTemplateCCTV(t *testing.T) {
	app := newTestApp()
	app.Get("/cctv/template", DownloadTemplateCCTV)

	resp, err := app.Test(httptest.NewRequest("GET", "/cctv/template", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", resp.Header.Get("Content-Type"))

	// File harus valid sebagai .xlsx dan memuat header kolom yang benar
	f, err := excelize.OpenReader(resp.Body)
	assert.NoError(t, err)
	defer func() { _ = f.Close() }()

	sheets := f.GetSheetList()
	assert.NotEmpty(t, sheets)
	cell, err := f.GetCellValue(sheets[0], "A1")
	assert.NoError(t, err)
	assert.Equal(t, "Kode CCTV", cell)
	cell, err = f.GetCellValue(sheets[0], "C1")
	assert.NoError(t, err)
	assert.Equal(t, "Nama CCTV", cell)
}

func TestGetCCTVQRCode_InvalidUUID(t *testing.T) {
	app := newTestApp()
	app.Get("/cctv/:id/qrcode", GetCCTVQRCode)

	resp, err := app.Test(httptest.NewRequest("GET", "/cctv/not-a-uuid/qrcode", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestUpdateCCTV_InvalidUUID(t *testing.T) {
	app := newTestApp()
	app.Put("/cctv/:id", UpdateCCTV)

	resp, err := app.Test(httptest.NewRequest("PUT", "/cctv/not-a-uuid", strings.NewReader(`{"nama":"Kamera"}`)))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestDeleteCCTV_InvalidUUID(t *testing.T) {
	app := newTestApp()
	app.Delete("/cctv/:id", DeleteCCTV)

	resp, err := app.Test(httptest.NewRequest("DELETE", "/cctv/not-a-uuid", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestImportCCTV_NoFile(t *testing.T) {
	app := newTestApp()
	app.Post("/cctv/import", ImportCCTV)

	resp, err := app.Test(httptest.NewRequest("POST", "/cctv/import", nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestImportCCTV_RejectsTooManyRows(t *testing.T) {
	// Keep the multipart construction close to the handler contract while
	// avoiding a database transaction because the row limit is checked first.
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()
	for i := 1; i <= maxImportRows+1; i++ {
		row := i + 1
		assert.NoError(t, f.SetCellValue("Sheet1", "A"+strconv.Itoa(row), "CCTV-"+strconv.Itoa(i)))
		assert.NoError(t, f.SetCellValue("Sheet1", "C"+strconv.Itoa(row), "Kamera"))
		assert.NoError(t, f.SetCellValue("Sheet1", "L"+strconv.Itoa(row), "aktif"))
	}
	var file bytes.Buffer
	assert.NoError(t, f.Write(&file))
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "import.xlsx")
	assert.NoError(t, err)
	_, err = part.Write(file.Bytes())
	assert.NoError(t, err)
	assert.NoError(t, writer.Close())

	app := newTestApp()
	app.Post("/cctv/import", ImportCCTV)
	req := httptest.NewRequest("POST", "/cctv/import", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestUpdateCCTV_NotFound(t *testing.T) {
	_, mock := setupMockGORM(t)
	id := uuid.New()
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).WillReturnError(gorm.ErrRecordNotFound)

	app := newTestApp()
	app.Put("/cctv/:id", UpdateCCTV)
	resp, err := app.Test(httptest.NewRequest("PUT", "/cctv/"+id.String(), strings.NewReader(`{"nama":"Kamera"}`)))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteCCTV_NotFound(t *testing.T) {
	_, mock := setupMockGORM(t)
	id := uuid.New()
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).WillReturnError(gorm.ErrRecordNotFound)

	app := newTestApp()
	app.Delete("/cctv/:id", DeleteCCTV)
	resp, err := app.Test(httptest.NewRequest("DELETE", "/cctv/"+id.String(), nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateCCTV_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	rows, id := cctvMockRow()
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).WillReturnRows(rows)
	mock.ExpectExec(`^UPDATE "cctvs" SET`).WillReturnResult(sqlmock.NewResult(1, 1))

	app := newTestApp()
	app.Put("/cctv/:id", UpdateCCTV)
	// "rusak" (status lama) dinormalisasi menjadi "nonaktif" sejak penggabungan status
	req := httptest.NewRequest("PUT", "/cctv/"+id.String(), strings.NewReader(`{"nama":"Kamera Diperbarui","status":"rusak"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	data := result["data"].(map[string]interface{})
	assert.Equal(t, "Kamera Diperbarui", data["nama"])
	assert.Equal(t, "nonaktif", data["status"])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteCCTV_Success(t *testing.T) {
	_, mock := setupMockGORM(t)
	rows, id := cctvMockRow()
	mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).WillReturnRows(rows)
	mock.ExpectExec(`^UPDATE "cctvs" SET "deleted_at"`).WillReturnResult(sqlmock.NewResult(1, 1))

	app := newTestApp()
	app.Delete("/cctv/:id", DeleteCCTV)
	resp, err := app.Test(httptest.NewRequest("DELETE", "/cctv/"+id.String(), nil))
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestNormalizeCCTVStatus memastikan padanan bahasa Indonesia & Inggris
// (huruf besar/kecil, dengan spasi di sekitar) dinormalisasi ke enum standar,
// sementara nilai tak dikenal dibiarkan apa adanya agar validasi oneof menolaknya.
func TestNormalizeCCTVStatus(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"indonesia aktif", "aktif", "aktif"},
		{"indonesia rusak → nonaktif", "rusak", "nonaktif"},
		{"indonesia pemeliharaan", "pemeliharaan", "pemeliharaan"},
		{"indonesia nonaktif", "nonaktif", "nonaktif"},
		{"inggris active", "active", "aktif"},
		{"inggris broken → nonaktif", "broken", "nonaktif"},
		{"inggris maintenance", "maintenance", "pemeliharaan"},
		{"inggris inactive", "inactive", "nonaktif"},
		{"huruf besar dari UI", "ACTIVE", "aktif"},
		{"huruf besar dari UI 2", "MAINTENANCE", "pemeliharaan"},
		{"huruf besar dari UI 3", "INACTIVE", "nonaktif"},
		{"campuran huruf", "MainTenance", "pemeliharaan"},
		{"spasi di sekitar", "  aktif ", "aktif"},
		{"nilai tidak dikenal dibiarkan", "matang", "matang"},
		{"string kosong", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, normalizeCCTVStatus(tt.in))
		})
	}
}

// TestRTSPPasswordNotInResponse memastikan password RTSP tidak bocor pada
// respons JSON (Task 13.3 / P3). Diuji lewat GET /cctv/:id dan POST /cctv/scan.
func TestRTSPPasswordNotInResponse(t *testing.T) {
	t.Setenv("AES_SECRET_KEY", "sigap_aes_test_secret_2026")

	t.Run("GetByID", func(t *testing.T) {
		_, mock := setupMockGORM(t)
		rows, id := cctvMockRow()
		mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).WillReturnRows(rows)

		app := newTestApp()
		app.Get("/cctv/:id", GetCCTVByID)
		resp, err := app.Test(httptest.NewRequest("GET", "/cctv/"+id.String(), nil))
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		data := result["data"].(map[string]interface{})
		val, exists := data["rtsp_password"]
		if exists {
			assert.Equal(t, "", val, "rtsp_password tidak boleh berisi nilai")
		}
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Scan", func(t *testing.T) {
		_, mock := setupMockGORM(t)
		rows, _ := cctvMockRow()
		mock.ExpectQuery(`^SELECT \* FROM "cctvs"`).WillReturnRows(rows)

		cctvID := uuid.New()
		encrypted, err := utils.EncryptAES256(cctvID.String())
		assert.NoError(t, err)

		app := newTestApp()
		app.Post("/cctv/scan", ScanCCTV)
		payload := `{"qr_payload":"` + encrypted + `"}`
		req := httptest.NewRequest("POST", "/cctv/scan", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, fiber.StatusOK, resp.StatusCode)

		var result map[string]interface{}
		assert.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		data := result["data"].(map[string]interface{})
		val, exists := data["rtsp_password"]
		if exists {
			assert.Equal(t, "", val, "rtsp_password tidak boleh berisi nilai")
		}
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestCreateCCTV_LatitudeOutOfRange memastikan latitude di luar rentang -90..90
// ditolak dengan 400 (Task 13.1).
func TestCreateCCTV_LatitudeOutOfRange(t *testing.T) {
	t.Setenv("AES_SECRET_KEY", "sigap_aes_test_secret_2026")
	_, mock := setupMockGORM(t)

	app := newTestApp()
	app.Post("/cctv", CreateCCTV)

	payload := `{"kode_cctv":"CCTV-LAT","nama":"Kamera LAT","status":"aktif","latitude":200}`
	req := httptest.NewRequest("POST", "/cctv", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCreateCCTV_ChannelPortRange memastikan channel_port di luar 1-35 ditolak
// dengan 400 (Task 13.4).
func TestCreateCCTV_ChannelPortRange(t *testing.T) {
	t.Setenv("AES_SECRET_KEY", "sigap_aes_test_secret_2026")
	_, mock := setupMockGORM(t)

	app := newTestApp()
	app.Post("/cctv", CreateCCTV)

	payload := `{"kode_cctv":"CCTV-PORT","nama":"Kamera PORT","status":"aktif","channel_port":40}`
	req := httptest.NewRequest("POST", "/cctv", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCreateCCTV_ChannelPortDuplicate memastikan channel_port yang sudah dipakai
// CCTV lain ditolak dengan 400 (Task 13.4).
func TestCreateCCTV_ChannelPortDuplicate(t *testing.T) {
	t.Setenv("AES_SECRET_KEY", "sigap_aes_test_secret_2026")
	_, mock := setupMockGORM(t)
	existingID := uuid.New()

	// Query keunikan channel_port menemukan CCTV lain.
	mock.ExpectQuery(`^SELECT \* FROM "cctvs" WHERE channel_port =`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "created_at", "updated_at", "deleted_at", "kode_cctv", "id_channel",
			"nama", "lokasi_area", "longitude", "latitude", "jenis_cctv", "sumber_listrik",
			"tipe_koneksi", "indoor_outdoor", "pemilik", "status", "ip_nvr", "ip_camera",
			"keterangan", "qr_payload", "health_score", "channel_port", "rtsp_password", "stream_enabled",
		}).AddRow(existingID, time.Now(), time.Now(), nil, "CCTV-010", "", "Kamera Lama", "",
			nil, nil, "", "", "", "", "", "aktif", "", "", "", "payload", 100, 19, "", false))

	app := newTestApp()
	app.Post("/cctv", CreateCCTV)

	payload := `{"kode_cctv":"CCTV-011","nama":"Kamera Baru","status":"aktif","channel_port":19}`
	req := httptest.NewRequest("POST", "/cctv", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
	assert.NoError(t, mock.ExpectationsWereMet())
}
