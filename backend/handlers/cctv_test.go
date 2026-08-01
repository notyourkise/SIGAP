package handlers

import (
	"encoding/json"
	"net/http/httptest"
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

// cctvMockRow membangun sqlmock rows dengan seluruh kolom tabel cctvs.
func cctvMockRow() (*sqlmock.Rows, uuid.UUID) {
	id := uuid.New()
	rows := sqlmock.NewRows([]string{
		"id", "created_at", "updated_at", "deleted_at",
		"kode_cctv", "id_channel", "nama", "lokasi_area",
		"longitude", "latitude", "jenis_cctv", "sumber_listrik",
		"tipe_koneksi", "indoor_outdoor", "pemilik", "status",
		"ip_nvr", "ip_camera", "keterangan", "qr_payload",
	}).AddRow(id, time.Now(), time.Now(), nil, "CCTV-001", "CH-01", "Kamera Depan", "Area Utama",
		"117.1", "-0.5", "Dome", "PLN", "Fiber", "outdoor", "Pemkot", "aktif",
		"192.168.1.10", "192.168.1.11", "Aman", "encrypted-payload")
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
	defer f.Close()

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
