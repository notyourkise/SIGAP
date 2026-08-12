package handlers

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"sigap-backend/config"
	"sigap-backend/models"
	"sigap-backend/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/skip2/go-qrcode"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

// normalizeCCTVStatus mengubah input status menjadi enum huruf kecil standar
// (aktif/pemeliharaan/nonaktif) agar penyimpanan & filter konsisten.
// Input huruf besar (mis. "ACTIVE" dari antarmuka/impor lama) maupun padanan
// bahasa Inggris ("active", "broken", "maintenance", "inactive") diterima dan
// dinormalisasi sebelum validasi.
// Sejak 2026-08-05 status "rusak" digabung ke "nonaktif" — keduanya tidak
// dibedakan oleh logika sistem maupun tampilan, sehingga masukan "rusak"/"broken"
// dipetakan menjadi "nonaktif".
func normalizeCCTVStatus(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "aktif", "active":
		return "aktif"
	case "rusak", "broken":
		return "nonaktif"
	case "pemeliharaan", "maintenance":
		return "pemeliharaan"
	case "nonaktif", "inactive":
		return "nonaktif"
	default:
		return s
	}
}

// Request body untuk pembuatan CCTV baru
type CreateCCTVRequest struct {
	KodeCCTV      string `json:"kode_cctv" validate:"required,min=3,max=50"`
	IDChannel     string `json:"id_channel" validate:"omitempty,max=50"`
	Nama          string `json:"nama" validate:"required,min=3,max=100"`
	LokasiArea    string   `json:"lokasi_area" validate:"omitempty,max=255"`
	Longitude     *float64 `json:"longitude"`
	Latitude      *float64 `json:"latitude"`
	JenisCCTV     string   `json:"jenis_cctv" validate:"omitempty,max=50"`
	SumberListrik string   `json:"sumber_listrik" validate:"omitempty,max=50"`
	TipeKoneksi   string   `json:"tipe_koneksi" validate:"omitempty,max=50"`
	IndoorOutdoor string   `json:"indoor_outdoor" validate:"omitempty,oneof=indoor outdoor INDOOR OUTDOOR"`
	Pemilik       string   `json:"pemilik" validate:"omitempty,max=100"`
	Status        string   `json:"status" validate:"required,oneof=aktif pemeliharaan nonaktif"`
	IPNVR         string   `json:"ip_nvr" validate:"omitempty,ip"`
	IPCamera      string   `json:"ip_camera" validate:"omitempty,ip"`
	Keterangan    string   `json:"keterangan" validate:"omitempty"`
	ChannelPort   int      `json:"channel_port"`
	RTSPPassword  string   `json:"rtsp_password"`
	StreamEnabled bool     `json:"stream_enabled"`
	RTSPURL       string   `json:"rtsp_url" validate:"omitempty,max=500"`
}

// Request body untuk update data CCTV
type UpdateCCTVRequest struct {
	KodeCCTV      string `json:"kode_cctv" validate:"omitempty,min=3,max=50"`
	IDChannel     string `json:"id_channel" validate:"omitempty,max=50"`
	Nama          string `json:"nama" validate:"omitempty,min=3,max=100"`
	LokasiArea    string   `json:"lokasi_area" validate:"omitempty,max=255"`
	Longitude     *float64 `json:"longitude"`
	Latitude      *float64 `json:"latitude"`
	JenisCCTV     string   `json:"jenis_cctv" validate:"omitempty,max=50"`
	SumberListrik string   `json:"sumber_listrik" validate:"omitempty,max=50"`
	TipeKoneksi   string   `json:"tipe_koneksi" validate:"omitempty,max=50"`
	IndoorOutdoor string   `json:"indoor_outdoor" validate:"omitempty,oneof=indoor outdoor INDOOR OUTDOOR"`
	Pemilik       string   `json:"pemilik" validate:"omitempty,max=100"`
	Status        string   `json:"status" validate:"omitempty,oneof=aktif pemeliharaan nonaktif"`
	IPNVR         string   `json:"ip_nvr" validate:"omitempty,ip"`
	IPCamera      string   `json:"ip_camera" validate:"omitempty,ip"`
	Keterangan    string   `json:"keterangan" validate:"omitempty"`
	ChannelPort   *int     `json:"channel_port"`
	RTSPPassword  string   `json:"rtsp_password"`
	StreamEnabled *bool    `json:"stream_enabled"`
	RTSPURL       *string  `json:"rtsp_url" validate:"omitempty,max=500"`
}

// Batas keamanan impor Excel: ukuran file maksimum (mengikuti BodyLimit Fiber
// default 4 MB) dan jumlah baris data maksimum per file agar proses impor
// massal tidak membebani memori server & database.
const (
	maxImportFileSize = 4 << 20 // 4 MB
	maxImportRows     = 1000    // maksimal 1000 baris data per file
)

// Request body untuk scanner QR Payload
type ScanCCTVRequest struct {
	QrPayload string `json:"qr_payload" validate:"required"`
}

// validateCoordinateRange memvalidasi rentang koordinat (Task 13.1).
// Latitude harus -90..90, Longitude -180..180. Nilai nil dianggap valid (opsional).
// Mengembalikan pesan error, atau "" bila valid.
func validateCoordinateRange(lat, lng *float64) string {
	if lat != nil && (*lat < -90 || *lat > 90) {
		return "Latitude tidak valid (rentang -90 s.d. 90)"
	}
	if lng != nil && (*lng < -180 || *lng > 180) {
		return "Longitude tidak valid (rentang -180 s.d. 180)"
	}
	return ""
}

// validateChannelPortForCreate memvalidasi range & keunikan ChannelPort (Task 13.4).
// Mengembalikan pesan error, atau "" bila valid.
func validateChannelPortForCreate(db *gorm.DB, port int) string {
	if port == 0 {
		return ""
	}
	if port < 1 || port > 35 {
		return "ChannelPort harus antara 1-35"
	}
	var existing models.CCTV
	if err := db.Where("channel_port = ?", port).First(&existing).Error; err == nil {
		return "ChannelPort sudah digunakan oleh CCTV: " + existing.Nama
	}
	return ""
}

// validateChannelPortForUpdate memvalidasi range & keunikan ChannelPort, mengecualikan
// CCTV itu sendiri (Task 13.4).
func validateChannelPortForUpdate(db *gorm.DB, port *int, excludeID uuid.UUID) string {
	if port == nil || *port == 0 {
		return ""
	}
	if *port < 1 || *port > 35 {
		return "ChannelPort harus antara 1-35"
	}
	var existing models.CCTV
	if err := db.Where("channel_port = ? AND id <> ?", *port, excludeID).First(&existing).Error; err == nil {
		return "ChannelPort sudah digunakan oleh CCTV: " + existing.Nama
	}
	return ""
}

// parseCoordinateCell mengurai nilai koordinat dari sel Excel menjadi *float64.
// Nilai kosong dikembalikan nil. Bila format tidak valid, dicatat peringatan dan
// mengembalikan nil (koordinat di-skip) agar seluruh baris impor tidak gagal.
func parseCoordinateCell(value string, rowNum int, field string) *float64 {
	v := strings.TrimSpace(value)
	if v == "" {
		return nil
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		utils.Log.Warnf("⚠️ Impor CCTV baris %d: %s '%s' bukan angka valid — koordinat dilewati", rowNum, field, v)
		return nil
	}
	return &f
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

	// Normalisasi status ke enum huruf kecil sebelum validasi & penyimpanan
	req.Status = normalizeCCTVStatus(req.Status)

	if errs := utils.ValidateStruct(&req); len(errs) > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Validasi input gagal",
			"errors":  errs,
		})
	}

	// Validasi rentang koordinat (Task 13.1)
	if msg := validateCoordinateRange(req.Latitude, req.Longitude); msg != "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": msg,
		})
	}

	// Validasi range & keunikan ChannelPort (Task 13.4)
	if msg := validateChannelPortForCreate(config.DB, req.ChannelPort); msg != "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": msg,
		})
	}

	// Cek keunikan kode_cctv (termasuk yang di-soft-delete)
	var existing models.CCTV
	if err := config.DB.Unscoped().Where("kode_cctv = ?", req.KodeCCTV).First(&existing).Error; err == nil {
		if existing.DeletedAt.Valid {
			// Hapus secara permanen data soft-deleted agar tidak memicu error unique constraint di PostgreSQL
			if err := config.DB.Unscoped().Delete(&existing).Error; err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"status":  "error",
					"message": "Gagal membersihkan data CCTV usang di database",
				})
			}
		} else {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "error",
				"message": "Kode CCTV sudah digunakan oleh perangkat lain",
			})
		}
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
		HealthScore:   100,
	}
	cctv.ID = newID // Set ID yang sudah dienkripsi tadi

	cctv.ChannelPort = req.ChannelPort
	cctv.StreamEnabled = req.StreamEnabled
	cctv.RTSPURL = req.RTSPURL
	if req.RTSPPassword != "" {
		encrypted, err := utils.EncryptAES256(req.RTSPPassword)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "Gagal mengenkripsi password RTSP",
			})
		}
		cctv.RTSPPassword = encrypted
	}

	if err := config.DB.Create(&cctv).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menyimpan data CCTV ke database",
		})
	}

	utils.LogAudit(config.DB, c, "CREATE_CCTV", "CCTV", "Membuat data CCTV baru: "+cctv.Nama+" ("+cctv.KodeCCTV+")", nil, "", "")
	utils.BroadcastEvent("CCTV_UPDATED", cctv)

	// Password RTSP tidak boleh bocor ke klien pada respons.
	cctv.RTSPPassword = ""

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
// @Param        zone_id query string false "Filter berdasarkan zona operasional (UUID zona)"
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

	// Filter site (multi-site, Task 12.2): hanya aktif bila X-Site-ID header
	// valid disediakan dan diset oleh middleware SiteContext. Opsional.
	if siteID, ok := c.Locals("site_id").(string); ok && siteID != "" {
		query = query.Where("site_id = ?", siteID)
	}

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
	// Filter zona operasional (Task 11.4, Sprint 28): ?zone_id=<uuid>.
	// Opsional — bila diisi hanya CCTV milik zona tersebut yang dikembalikan.
	if zoneID := c.Query("zone_id"); zoneID != "" {
		zid, err := uuid.Parse(zoneID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "error",
				"message": "Filter zone_id tidak valid",
			})
		}
		query = query.Where("zone_id = ?", zid)
	}

	// Pencarian
	if search := c.Query("search"); search != "" {
		query = query.Where("nama ILIKE ? OR kode_cctv ILIKE ? OR id_channel ILIKE ? OR lokasi_area ILIKE ?", "%"+search+"%", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	// Paginasi & Sorting
	allowedSort := []string{"id", "kode_cctv", "nama", "lokasi_area", "status", "health_score", "created_at", "updated_at"}
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

	// Password RTSP tidak boleh bocor ke klien pada respons.
	for i := range cctvs {
		cctvs[i].RTSPPassword = ""
		// SnapshotURL adalah field computed — sertakan path snapshot terbaru.
		cctvs[i].SnapshotURL = snapshotURLFor(cctvs[i].ID.String())
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

	// Password RTSP tidak boleh bocor ke klien pada respons.
	cctv.RTSPPassword = ""
	// SnapshotURL adalah field computed — sertakan path snapshot terbaru.
	cctv.SnapshotURL = snapshotURLFor(cctv.ID.String())

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   cctv,
	})
}

// GetCCTVHealthScore menghitung skor kesehatan satu CCTV dari lima faktor
// operasional dalam 90 hari terakhir. Endpoint dapat diakses semua role.
// @Summary      Ambil Skor Kesehatan CCTV
// @Description  Menghitung skor kesehatan 0-100 berdasarkan kegagalan pemeliharaan, field-check attention, overdue, umur pemeliharaan berhasil, dan item checklist not_ok.
// @Tags         Master CCTV
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "CCTV UUID"
// @Success      200 {object} map[string]interface{} "Skor kesehatan CCTV"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      404 {object} map[string]interface{} "CCTV tidak ditemukan"
// @Failure      500 {object} map[string]interface{} "Gagal menghitung skor kesehatan"
// @Router       /cctv/{id}/health-score [get]
func GetCCTVHealthScore(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID CCTV tidak valid",
		})
	}

	var cctv models.CCTV
	if err := config.DB.Select("id").First(&cctv, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Data CCTV tidak ditemukan",
		})
	}

	result, err := CalculateCCTVHealthScoreAt(config.DB, id, time.Now())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghitung skor kesehatan CCTV",
		})
	}
	if err := config.DB.Model(&models.CCTV{}).Where("id = ?", id).Update("health_score", result.HealthScore).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menyimpan skor kesehatan CCTV",
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   result,
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

	// Catat status awal sebelum update untuk mendeteksi perubahan status.
	oldStatus := normalizeCCTVStatus(cctv.Status)

	var req UpdateCCTVRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Format input tidak valid",
		})
	}

	// Normalisasi status ke enum huruf kecil sebelum validasi & penyimpanan
	req.Status = normalizeCCTVStatus(req.Status)

	if errs := utils.ValidateStruct(&req); len(errs) > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Validasi input gagal",
			"errors":  errs,
		})
	}

	// Validasi rentang koordinat (Task 13.1)
	if msg := validateCoordinateRange(req.Latitude, req.Longitude); msg != "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": msg,
		})
	}

	// Validasi range & keunikan ChannelPort, mengecualikan CCTV ini (Task 13.4)
	if msg := validateChannelPortForUpdate(config.DB, req.ChannelPort, id); msg != "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": msg,
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
	if req.Longitude != nil {
		cctv.Longitude = req.Longitude
	}
	if req.Latitude != nil {
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
	if req.ChannelPort != nil {
		cctv.ChannelPort = *req.ChannelPort
	}
	if req.RTSPPassword != "" {
		encrypted, err := utils.EncryptAES256(req.RTSPPassword)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "Gagal mengenkripsi password RTSP",
			})
		}
		cctv.RTSPPassword = encrypted
	}
	if req.StreamEnabled != nil {
		cctv.StreamEnabled = *req.StreamEnabled
	}
	if req.RTSPURL != nil {
		cctv.RTSPURL = *req.RTSPURL
	}

	if err := config.DB.Save(&cctv).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal memperbarui data CCTV",
		})
	}

	utils.LogAudit(config.DB, c, "UPDATE_CCTV", "CCTV", "Memperbarui data CCTV: "+cctv.Nama+" ("+cctv.KodeCCTV+")", nil, "", "")
	utils.BroadcastEvent("CCTV_UPDATED", cctv)

	// Hook Task 10.1: bila status benar-benar berubah, ambil snapshot otomatis
	// secara asinkron agar tidak memblokir respons API.
	if oldStatus != normalizeCCTVStatus(cctv.Status) {
		reason := "status CCTV diubah ke " + cctv.Status
		go func(cctv models.CCTV, reason string) {
			_, _ = CaptureSnapshot(cctv, EventStatusChanged, reason)
		}(cctv, reason)
	}

	// Password RTSP tidak boleh bocor ke klien pada respons.
	cctv.RTSPPassword = ""

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

	utils.LogAudit(config.DB, c, "DELETE_CCTV", "CCTV", "Menghapus data CCTV: "+cctv.Nama+" ("+cctv.KodeCCTV+")", nil, "", "")
	utils.BroadcastEvent("CCTV_UPDATED", cctv)

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

	// Password RTSP tidak boleh bocor ke klien pada respons.
	cctv.RTSPPassword = ""

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   cctv,
	})
}

// DownloadTemplateCCTV mengunduh template Excel kosong untuk import data CCTV (Admin & Teknisi)
// @Summary      Unduh Template Excel CCTV
// @Description  Membuat dan mengunduh berkas spreadsheet Excel (.xlsx) kosong dengan kolom header standar untuk mempermudah impor data CCTV secara massal.
// @Tags         Master CCTV
// @Produce      application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Security     BearerAuth
// @Success      200 {string} string "Berkas template Excel (.xlsx)"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid"
// @Failure      500 {object} map[string]interface{} "Gagal membuat file template"
// @Router       /cctv/template [get]
func DownloadTemplateCCTV(c *fiber.Ctx) error {
	f := excelize.NewFile()
	defer func() {
		_ = f.Close()
	}()

	// Set nama sheet default
	sheetName := "Sheet1"

	// Daftar header kolom template
	headers := []string{
		"Kode CCTV", "ID Channel", "Nama CCTV", "Lokasi Area",
		"Longitude", "Latitude", "Jenis CCTV", "Sumber Listrik",
		"Tipe Koneksi", "Indoor/Outdoor", "Pemilik", "Status",
		"IP NVR", "IP Camera", "Keterangan",
	}

	// Tulis header ke baris pertama (Baris 1) secara berurutan
	for i, header := range headers {
		colName, err := excelize.ColumnNumberToName(i + 1)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "Gagal memproses kolom template",
			})
		}
		cellRef := colName + "1"
		_ = f.SetCellValue(sheetName, cellRef, header)
	}

	// Buat buffer memori untuk menyimpan spreadsheet
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menulis file template",
		})
	}

	c.Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Set("Content-Disposition", "attachment; filename=template_import_cctv.xlsx")

	return c.Send(buf.Bytes())
}

// ImportCCTV mengimpor data master CCTV secara massal menggunakan file Excel (Admin saja)
// @Summary      Impor Massal Data CCTV
// @Description  Mengunggah berkas Excel (.xlsx) yang diisi sesuai template standar untuk diimpor secara massal ke dalam database. UUID dienkripsi dengan AES-256 untuk kolom QrPayload.
// @Tags         Master CCTV
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        file formData file true "Berkas Excel Template CCTV (.xlsx)"
// @Success      200 {object} map[string]interface{} "Proses impor selesai dengan detail jumlah sukses/gagal"
// @Failure      400 {object} map[string]interface{} "Tidak ada file / format file tidak valid"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Failure      500 {object} map[string]interface{} "Gagal membaca atau memproses berkas Excel"
// @Router       /cctv/import [post]
func ImportCCTV(c *fiber.Ctx) error {
	// Ambil file dari form-data
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "File tidak ditemukan. Pastikan key form-data bernama 'file'.",
		})
	}

	// Batasi ukuran file agar tidak membebani memori server
	if fileHeader.Size > maxImportFileSize {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": fmt.Sprintf("Ukuran file terlalu besar (maksimal %d MB)", maxImportFileSize/(1<<20)),
		})
	}

	// Buka file multipart
	file, err := fileHeader.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal membuka file yang diunggah",
		})
	}
	defer func() { _ = file.Close() }()

	// Baca dengan excelize
	f, err := excelize.OpenReader(file)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Format file tidak valid. Pastikan mengunggah file Excel (.xlsx).",
		})
	}
	defer func() {
		_ = f.Close()
	}()

	// Baca sheet pertama
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "File Excel kosong atau tidak memiliki sheet",
		})
	}
	sheetName := sheets[0]

	// Dapatkan semua baris
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal membaca data dari sheet",
		})
	}

	// Validasi minimal baris (minimal ada header dan 1 data)
	if len(rows) <= 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "File Excel tidak berisi data untuk diimpor",
		})
	}

	// Batasi jumlah baris data agar proses impor massal tetap aman & cepat
	if len(rows)-1 > maxImportRows {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": fmt.Sprintf("File Excel melebihi batas maksimal %d baris data (file berisi %d baris)", maxImportRows, len(rows)-1),
		})
	}

	// Mulai GORM transaction untuk keamanan data (jika salah satu gagal, rollback semua)
	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var importedCount int

	for idx, row := range rows {
		// Lewati header (index 0 adalah Baris 1)
		if idx == 0 {
			continue
		}

		// Jika baris kosong atau tidak memiliki cell apa-apa, lewati
		if len(row) == 0 || (len(row) > 0 && row[0] == "") {
			continue
		}

		// Pastikan panjang array kolom aman diakses (excelize memotong array jika kolom kanan kosong)
		getCell := func(cellIdx int) string {
			if cellIdx < len(row) {
				return row[cellIdx]
			}
			return ""
		}

		rowNum := idx + 1
		kodeCCTV := getCell(0)
		idChannel := getCell(1)
		nama := getCell(2)
		lokasiArea := getCell(3)
		longitude := parseCoordinateCell(getCell(4), rowNum, "longitude")
		latitude := parseCoordinateCell(getCell(5), rowNum, "latitude")
		jenisCCTV := getCell(6)
		sumberListrik := getCell(7)
		tipeKoneksi := getCell(8)
		indoorOutdoor := getCell(9)
		pemilik := getCell(10)
		status := getCell(11)
		ipNVR := getCell(12)
		ipCamera := getCell(13)
		keterangan := getCell(14)

		// Validasi kolom wajib
		if kodeCCTV == "" {
			tx.Rollback()
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "error",
				"message": fmt.Sprintf("Baris %d: Kode CCTV wajib diisi", rowNum),
			})
		}
		if nama == "" {
			tx.Rollback()
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "error",
				"message": fmt.Sprintf("Baris %d: Nama CCTV wajib diisi", rowNum),
			})
		}
		if status == "" {
			tx.Rollback()
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "error",
				"message": fmt.Sprintf("Baris %d: Status wajib diisi", rowNum),
			})
		}

		// Normalisasi & validasi enum status agar data impor konsisten
		status = normalizeCCTVStatus(status)
		switch status {
		case "aktif", "pemeliharaan", "nonaktif":
		default:
			tx.Rollback()
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "error",
				"message": fmt.Sprintf("Baris %d: Status '%s' tidak valid. Gunakan: aktif, pemeliharaan, atau nonaktif", rowNum, getCell(11)),
			})
		}

		// Cek keunikan Kode CCTV di database (termasuk yang di-soft-delete)
		var existing models.CCTV
		if err := tx.Unscoped().Where("kode_cctv = ?", kodeCCTV).First(&existing).Error; err == nil {
			if existing.DeletedAt.Valid {
				// Hapus secara permanen data soft-deleted agar tidak memicu error unique constraint di PostgreSQL
				if err := tx.Unscoped().Delete(&existing).Error; err != nil {
					tx.Rollback()
					return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
						"status":  "error",
						"message": fmt.Sprintf("Baris %d: Gagal membersihkan data CCTV usang di database", rowNum),
					})
				}
			} else {
				tx.Rollback()
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"status":  "error",
					"message": fmt.Sprintf("Baris %d: Kode CCTV '%s' sudah digunakan", rowNum, kodeCCTV),
				})
			}
		}

		// Generate ID & QrPayload
		newID := uuid.New()
		encryptedPayload, err := utils.EncryptAES256(newID.String())
		if err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": fmt.Sprintf("Baris %d: Gagal memproses enkripsi QR Payload", rowNum),
			})
		}

		cctv := models.CCTV{
			KodeCCTV:      kodeCCTV,
			IDChannel:     idChannel,
			Nama:          nama,
			LokasiArea:    lokasiArea,
			Longitude:     longitude,
			Latitude:      latitude,
			JenisCCTV:     jenisCCTV,
			SumberListrik: sumberListrik,
			TipeKoneksi:   tipeKoneksi,
			IndoorOutdoor: indoorOutdoor,
			Pemilik:       pemilik,
			Status:        status,
			IPNVR:         ipNVR,
			IPCamera:      ipCamera,
			Keterangan:    keterangan,
			QrPayload:     encryptedPayload,
			HealthScore:   100,
		}
		cctv.ID = newID

		if err := tx.Create(&cctv).Error; err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": fmt.Sprintf("Baris %d: Gagal menyimpan data CCTV ke database: %v", rowNum, err),
			})
		}

		importedCount++
	}

	// Commit transaksi jika semua baris valid
	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal melakukan transaksi commit impor data",
		})
	}

	utils.LogAudit(config.DB, c, "IMPORT_CCTV", "CCTV", fmt.Sprintf("Impor massal Excel berhasil: %d data CCTV ditambahkan", importedCount), nil, "", "")

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": fmt.Sprintf("Berhasil mengimpor %d data CCTV secara massal", importedCount),
	})
}

// GetCCTVQRCode menghasilkan QR Code PNG berisi QrPayload terenkripsi CCTV (Admin & Teknisi)
// @Summary      Unduh QR Code CCTV
// @Description  Membuat QR Code dalam format PNG berisi payload QR terenkripsi (AES-256) milik perangkat CCTV. Hasil pindaian QR ini dapat diidentifikasi melalui endpoint /cctv/scan.
// @Tags         Master CCTV
// @Produce      image/png
// @Security     BearerAuth
// @Param        id path string true "CCTV UUID"
// @Success      200 {string} string "Gambar QR Code dalam format raw PNG"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid"
// @Failure      404 {object} map[string]interface{} "CCTV tidak ditemukan"
// @Failure      500 {object} map[string]interface{} "Gagal memproses QR Code"
// @Router       /cctv/{id}/qrcode [get]
func GetCCTVQRCode(c *fiber.Ctx) error {
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

	// Konten QR adalah QrPayload terenkripsi AES-256 sehingga identitas perangkat
	// tidak terekspos langsung saat label dipindai.
	qrContent := cctv.QrPayload

	// Generate QR Code bytes menggunakan library skip2/go-qrcode
	// 256px x 256px, Level error correction Medium
	qrBytes, err := qrcode.Encode(qrContent, qrcode.Medium, 256)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal membuat gambar QR Code",
		})
	}

	c.Set("Content-Type", "image/png")
	c.Set("Content-Disposition", fmt.Sprintf("inline; filename=%s-qr.png", cctv.KodeCCTV))
	return c.Send(qrBytes)
}

// RotateQRCode merotasi QR Code CCTV (Admin saja). Menghasilkan UUID baru,
// mengenkripsinya ulang dengan AES-256 sebagai QrPayload baru, sehingga QR lama
// tidak lagi valid untuk pemindaian. (Task 13.14, Sprint 34)
// @Summary      Rotasi QR Code CCTV (Admin)
// @Description  Membuat QrPayload baru (UUID baru terenkripsi AES-256) sehingga QR Code lama tidak berlaku lagi. Khusus admin.
// @Tags         Master CCTV
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "CCTV UUID"
// @Success      200 {object} map[string]interface{} "QR Code berhasil dirotasi (data CCTV terbaru)"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin yang diperbolehkan"
// @Failure      404 {object} map[string]interface{} "CCTV tidak ditemukan"
// @Failure      500 {object} map[string]interface{} "Gagal merotasi QR Code"
// @Router       /cctv/{id}/rotate-qr [post]
func RotateQRCode(c *fiber.Ctx) error {
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

	// Generate UUID baru & enkripsi sebagai QrPayload baru.
	newID := uuid.New()
	encryptedPayload, err := utils.EncryptAES256(newID.String())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal mengenkripsi QR Code baru",
		})
	}
	cctv.QrPayload = encryptedPayload

	if err := config.DB.Save(&cctv).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menyimpan QR Code baru",
		})
	}

	utils.LogAudit(config.DB, c, "ROTATE_QR_CODE", "CCTV",
		"Merotasi QR Code CCTV: "+cctv.Nama+" ("+cctv.KodeCCTV+")", nil, "", "")
	utils.BroadcastEvent("CCTV_UPDATED", cctv)

	// Password RTSP tidak boleh bocor ke klien pada respons.
	cctv.RTSPPassword = ""
	cctv.SnapshotURL = snapshotURLFor(cctv.ID.String())

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "QR Code berhasil dirotasi",
		"data":    cctv,
	})
}

// BulkDeleteCCTV menghapus beberapa data CCTV sekaligus atau mengosongkan seluruh tabel (Admin saja)
// @Summary      Hapus Massal Data CCTV
// @Description  Menghapus beberapa perangkat CCTV berdasarkan daftar UUID yang dipilih, atau mengosongkan seluruh tabel CCTV jika opsi 'all' bernilai true.
// @Tags         Master CCTV
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body BulkDeleteRequest true "Daftar ID CCTV yang akan dihapus atau flag hapus semua"
// @Success      200 {object} map[string]interface{} "Proses hapus massal berhasil"
// @Failure      400 {object} map[string]interface{} "Format data tidak valid"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid"
// @Failure      403 {object} map[string]interface{} "Akses ditolak"
// @Failure      500 {object} map[string]interface{} "Gagal menghapus data dari database"
// @Router       /cctv/bulk-delete [post]
type BulkDeleteRequest struct {
	IDs []string `json:"ids"`
	All bool     `json:"all"`
}

func BulkDeleteCCTV(c *fiber.Ctx) error {
	var req BulkDeleteRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Format input tidak valid",
		})
	}

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if req.All {
		// Hapus semua data CCTV (soft-delete oleh GORM)
		if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.CCTV{}).Error; err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "Gagal mengosongkan seluruh data CCTV",
			})
		}
	} else {
		if len(req.IDs) == 0 {
			tx.Rollback()
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "error",
				"message": "Daftar ID CCTV tidak boleh kosong",
			})
		}

		// Konversi string IDs ke UUIDs
		var uuids []uuid.UUID
		for _, idStr := range req.IDs {
			parsedUUID, err := uuid.Parse(idStr)
			if err != nil {
				tx.Rollback()
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"status":  "error",
					"message": fmt.Sprintf("ID tidak valid: %s", idStr),
				})
			}
			uuids = append(uuids, parsedUUID)
		}

		// Hapus CCTV berdasarkan ID yang dicocokkan
		if err := tx.Where("id IN ?", uuids).Delete(&models.CCTV{}).Error; err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "Gagal menghapus data CCTV terpilih",
			})
		}
	}

	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menyimpan perubahan transaksi hapus massal",
		})
	}

	message := "Berhasil menghapus data CCTV terpilih"
	auditDesc := fmt.Sprintf("Hapus massal CCTV terpilih: %d data dihapus", len(req.IDs))
	if req.All {
		message = "Berhasil mengosongkan seluruh data CCTV"
		auditDesc = "Mengosongkan (truncate) seluruh data CCTV"
	}

	utils.LogAudit(config.DB, c, "BULK_DELETE_CCTV", "CCTV", auditDesc, nil, "", "")

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": message,
	})
}
