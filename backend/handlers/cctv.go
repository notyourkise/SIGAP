package handlers

import (
	"errors"
	"sigap-backend/config"
	"sigap-backend/models"
	"sigap-backend/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Request body untuk pembuatan CCTV baru
type CreateCCTVRequest struct {
	KodeCCTV      string `json:"kode_cctv" validate:"required,min=3,max=50"`
	IDChannel     string `json:"id_channel" validate:"omitempty,max=50"`
	Nama          string `json:"nama" validate:"required,min=3,max=100"`
	LokasiArea    string `json:"lokasi_area" validate:"omitempty,max=255"`
	Longitude     string `json:"longitude" validate:"omitempty,max=50"`
	Latitude      string `json:"latitude" validate:"omitempty,max=50"`
	JenisCCTV     string `json:"jenis_cctv" validate:"omitempty,max=50"`
	SumberListrik string `json:"sumber_listrik" validate:"omitempty,max=50"`
	TipeKoneksi   string `json:"tipe_koneksi" validate:"omitempty,max=50"`
	IndoorOutdoor string `json:"indoor_outdoor" validate:"omitempty,oneof=indoor outdoor INDOOR OUTDOOR"`
	Pemilik       string `json:"pemilik" validate:"omitempty,max=100"`
	Status        string `json:"status" validate:"required,oneof=aktif rusak pemeliharaan nonaktif"`
	IPNVR         string `json:"ip_nvr" validate:"omitempty,ip"`
	IPCamera      string `json:"ip_camera" validate:"omitempty,ip"`
	Keterangan    string `json:"keterangan" validate:"omitempty"`
}

// Request body untuk update data CCTV
type UpdateCCTVRequest struct {
	KodeCCTV      string `json:"kode_cctv" validate:"omitempty,min=3,max=50"`
	IDChannel     string `json:"id_channel" validate:"omitempty,max=50"`
	Nama          string `json:"nama" validate:"omitempty,min=3,max=100"`
	LokasiArea    string `json:"lokasi_area" validate:"omitempty,max=255"`
	Longitude     string `json:"longitude" validate:"omitempty,max=50"`
	Latitude      string `json:"latitude" validate:"omitempty,max=50"`
	JenisCCTV     string `json:"jenis_cctv" validate:"omitempty,max=50"`
	SumberListrik string `json:"sumber_listrik" validate:"omitempty,max=50"`
	TipeKoneksi   string `json:"tipe_koneksi" validate:"omitempty,max=50"`
	IndoorOutdoor string `json:"indoor_outdoor" validate:"omitempty,oneof=indoor outdoor INDOOR OUTDOOR"`
	Pemilik       string `json:"pemilik" validate:"omitempty,max=100"`
	Status        string `json:"status" validate:"omitempty,oneof=aktif rusak pemeliharaan nonaktif"`
	IPNVR         string `json:"ip_nvr" validate:"omitempty,ip"`
	IPCamera      string `json:"ip_camera" validate:"omitempty,ip"`
	Keterangan    string `json:"keterangan" validate:"omitempty"`
}

// Request body untuk scanner QR Payload
type ScanCCTVRequest struct {
	QrPayload string `json:"qr_payload" validate:"required"`
}

// CreateCCTV membuat data CCTV baru & meng-generate QR Payload AES terenkripsi (Admin saja)
// @Summary      Tambah Data CCTV
// @Description  Membuat record master data CCTV baru. Sistem otomatis membuat UUID ID baru, mengenkripsinya dengan AES-256, lalu menyimpan hasilnya sebagai QrPayload unik.
// @Tags         Master CCTV
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body CreateCCTVRequest true "Data CCTV Baru"
// @Success      201 {object} map[string]interface{} "Data CCTV berhasil disimpan"
// @Failure      400 {object} map[string]interface{} "Format input salah, validasi gagal, kode_cctv duplikat"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Failure      500 {object} map[string]interface{} "Gagal mengenkripsi atau menyimpan data ke database"
// @Router       /cctv [post]
func CreateCCTV(c *fiber.Ctx) error {
	var req CreateCCTVRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Format input tidak valid",
		})
	}

	if errs := utils.ValidateStruct(&req); len(errs) > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Validasi input gagal",
			"errors":  errs,
		})
	}

	// Cek keunikan kode_cctv
	var existing models.CCTV
	if err := config.DB.Where("kode_cctv = ?", req.KodeCCTV).First(&existing).Error; err == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Kode CCTV sudah digunakan oleh perangkat lain",
		})
	}

	// Generate UUID baru secara manual terlebih dahulu untuk kebutuhan enkripsi QrPayload
	newID := uuid.New()
	encryptedPayload, err := utils.EncryptAES256(newID.String())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal memproses pembuatan QR payload terenkripsi",
		})
	}

	cctv := models.CCTV{
		KodeCCTV:      req.KodeCCTV,
		IDChannel:     req.IDChannel,
		Nama:          req.Nama,
		LokasiArea:    req.LokasiArea,
		Longitude:     req.Longitude,
		Latitude:      req.Latitude,
		JenisCCTV:     req.JenisCCTV,
		SumberListrik: req.SumberListrik,
		TipeKoneksi:   req.TipeKoneksi,
		IndoorOutdoor: req.IndoorOutdoor,
		Pemilik:       req.Pemilik,
		Status:        req.Status,
		IPNVR:         req.IPNVR,
		IPCamera:      req.IPCamera,
		Keterangan:    req.Keterangan,
		QrPayload:     encryptedPayload,
	}
	cctv.ID = newID // Set ID yang sudah dienkripsi tadi

	if err := config.DB.Create(&cctv).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menyimpan data CCTV ke database",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status": "success",
		"data":   cctv,
	})
}

// GetAllCCTV mengambil daftar master data CCTV terpaginasi (Admin & Teknisi)
// @Summary      Ambil Semua Data CCTV
// @Description  Mendapatkan daftar data master CCTV dengan paginasi, pencarian, dan penyaringan berdasarkan status, lokasi, jenis, pemilik, dll.
// @Tags         Master CCTV
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        status query string false "Filter berdasarkan status"
// @Param        jenis_cctv query string false "Filter berdasarkan jenis CCTV"
// @Param        indoor_outdoor query string false "Filter berdasarkan indoor/outdoor"
// @Param        search query string false "Cari berdasarkan nama, kode_cctv, id_channel, lokasi_area"
// @Param        page query integer false "Nomor Halaman (default: 1)"
// @Param        limit query integer false "Jumlah data per halaman (default: 10)"
// @Param        sort_by query string false "Kolom pengurutan (default: created_at)"
// @Param        order query string false "Arah pengurutan: asc, desc (default: desc)"
// @Success      200 {object} map[string]interface{} "Daftar CCTV terpaginasi"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Router       /cctv [get]
func GetAllCCTV(c *fiber.Ctx) error {
	var cctvs []models.CCTV
	query := config.DB.Model(&models.CCTV{})

	// Filter
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if jenis := c.Query("jenis_cctv"); jenis != "" {
		query = query.Where("jenis_cctv = ?", jenis)
	}
	if io := c.Query("indoor_outdoor"); io != "" {
		query = query.Where("indoor_outdoor = ?", io)
	}

	// Pencarian
	if search := c.Query("search"); search != "" {
		query = query.Where("nama ILIKE ? OR kode_cctv ILIKE ? OR id_channel ILIKE ? OR lokasi_area ILIKE ?", "%"+search+"%", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	// Paginasi & Sorting
	allowedSort := []string{"id", "kode_cctv", "nama", "lokasi_area", "status", "created_at", "updated_at"}
	limit, page, sort := utils.GetPaginationParams(c, allowedSort)

	var totalRows int64
	if err := query.Count(&totalRows).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghitung total data CCTV",
		})
	}

	offset := (page - 1) * limit
	if err := query.Offset(offset).Limit(limit).Order(sort).Find(&cctvs).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal mengambil daftar data CCTV",
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"rows":       cctvs,
			"pagination": utils.CalculatePagination(totalRows, page, limit),
		},
	})
}

// GetCCTVByID mengambil data detail CCTV berdasarkan UUID (Admin & Teknisi)
// @Summary      Ambil CCTV Detail
// @Description  Mendapatkan detail CCTV berdasarkan UUID-nya.
// @Tags         Master CCTV
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "CCTV UUID"
// @Success      200 {object} map[string]interface{} "Detail data CCTV ditemukan"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      404 {object} map[string]interface{} "CCTV tidak ditemukan"
// @Router       /cctv/{id} [get]
func GetCCTVByID(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID CCTV tidak valid",
		})
	}

	var cctv models.CCTV
	if err := config.DB.First(&cctv, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Data CCTV tidak ditemukan",
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   cctv,
	})
}

// UpdateCCTV mengedit data CCTV (Admin saja)
// @Summary      Edit Data CCTV
// @Description  Mengubah detail data master CCTV yang sudah terdaftar. Hanya dapat diakses oleh Admin.
// @Tags         Master CCTV
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "CCTV UUID"
// @Param        request body UpdateCCTVRequest true "Data Update CCTV"
// @Success      200 {object} map[string]interface{} "Data CCTV berhasil diperbarui"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid, format body salah, atau validasi gagal"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Failure      404 {object} map[string]interface{} "CCTV tidak ditemukan"
// @Failure      500 {object} map[string]interface{} "Gagal menyimpan perubahan ke database"
// @Router       /cctv/{id} [put]
func UpdateCCTV(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID CCTV tidak valid",
		})
	}

	var cctv models.CCTV
	if err := config.DB.First(&cctv, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Data CCTV tidak ditemukan",
		})
	}

	var req UpdateCCTVRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Format input tidak valid",
		})
	}

	if errs := utils.ValidateStruct(&req); len(errs) > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Validasi input gagal",
			"errors":  errs,
		})
	}

	// Update field jika disediakan
	if req.KodeCCTV != "" && req.KodeCCTV != cctv.KodeCCTV {
		// Validasi keunikan kode baru
		var existing models.CCTV
		if err := config.DB.Where("kode_cctv = ? AND id != ?", req.KodeCCTV, id).First(&existing).Error; err == nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "error",
				"message": "Kode CCTV sudah digunakan oleh perangkat lain",
			})
		}
		cctv.KodeCCTV = req.KodeCCTV
	}

	if req.IDChannel != "" {
		cctv.IDChannel = req.IDChannel
	}
	if req.Nama != "" {
		cctv.Nama = req.Nama
	}
	if req.LokasiArea != "" {
		cctv.LokasiArea = req.LokasiArea
	}
	if req.Longitude != "" {
		cctv.Longitude = req.Longitude
	}
	if req.Latitude != "" {
		cctv.Latitude = req.Latitude
	}
	if req.JenisCCTV != "" {
		cctv.JenisCCTV = req.JenisCCTV
	}
	if req.SumberListrik != "" {
		cctv.SumberListrik = req.SumberListrik
	}
	if req.TipeKoneksi != "" {
		cctv.TipeKoneksi = req.TipeKoneksi
	}
	if req.IndoorOutdoor != "" {
		cctv.IndoorOutdoor = req.IndoorOutdoor
	}
	if req.Pemilik != "" {
		cctv.Pemilik = req.Pemilik
	}
	if req.Status != "" {
		cctv.Status = req.Status
	}
	if req.IPNVR != "" {
		cctv.IPNVR = req.IPNVR
	}
	if req.IPCamera != "" {
		cctv.IPCamera = req.IPCamera
	}
	if req.Keterangan != "" {
		cctv.Keterangan = req.Keterangan
	}

	if err := config.DB.Save(&cctv).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal memperbarui data CCTV",
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   cctv,
	})
}

// DeleteCCTV menghapus data CCTV secara soft-delete (Admin saja)
// @Summary      Hapus Data CCTV
// @Description  Menghapus data CCTV secara soft-delete berdasarkan UUID. Hanya dapat diakses oleh Admin.
// @Tags         Master CCTV
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "CCTV UUID"
// @Success      200 {object} map[string]interface{} "Data CCTV berhasil dihapus secara soft-delete"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Failure      404 {object} map[string]interface{} "CCTV tidak ditemukan"
// @Router       /cctv/{id} [delete]
func DeleteCCTV(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID CCTV tidak valid",
		})
	}

	var cctv models.CCTV
	if err := config.DB.First(&cctv, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Data CCTV tidak ditemukan",
		})
	}

	if err := config.DB.Delete(&cctv).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghapus data CCTV",
		})
	}

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Data CCTV berhasil dihapus secara soft-delete",
	})
}

// ScanCCTV mendekripsi payload QR code dan mengembalikan detail CCTV yang bersangkutan (Admin & Teknisi)
// @Summary      Scan & Identifikasi CCTV via QR Payload
// @Description  Menerima parameter string QR Payload terenkripsi, mendekripsinya menggunakan AES-256 untuk mendapatkan UUID asli, lalu mengembalikan data umum CCTV yang cocok.
// @Tags         Master CCTV
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body ScanCCTVRequest true "Payload QR Terenkripsi dari CCTV fisik"
// @Success      200 {object} map[string]interface{} "Identitas CCTV berhasil dipindai dan ditemukan"
// @Failure      400 {object} map[string]interface{} "QR Payload tidak valid atau gagal didekripsi"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid"
// @Failure      404 {object} map[string]interface{} "Data CCTV tidak ditemukan di database"
// @Router       /cctv/scan [post]
func ScanCCTV(c *fiber.Ctx) error {
	var req ScanCCTVRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Format input tidak valid",
		})
	}

	if errs := utils.ValidateStruct(&req); len(errs) > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "QR payload tidak boleh kosong",
		})
	}

	// Lakukan dekripsi payload
	decryptedUUID, err := utils.DecryptAES256(req.QrPayload)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "QR payload tidak valid atau gagal didekripsi secara aman",
		})
	}

	// Parse hasil dekripsi menjadi UUID
	cctvID, err := uuid.Parse(decryptedUUID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Kunci enkripsi QR tidak valid",
		})
	}

	var cctv models.CCTV
	if err := config.DB.First(&cctv, "id = ?", cctvID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"status":  "error",
				"message": "Perangkat CCTV tidak terdaftar dalam sistem",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Terjadi kesalahan internal saat membaca data",
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   cctv,
	})
}
