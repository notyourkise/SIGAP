package handlers

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"sigap-backend/config"
	"sigap-backend/models"
	"sigap-backend/utils"

	"github.com/go-pdf/fpdf"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// maxFieldChecks membatasi jumlah item checklist agar mencegah penyalahgunaan payload.
const maxFieldChecks = 30

// maxFieldImages membatasi jumlah foto bukti per catatan pengecekan.
const maxFieldImages = 5

type FieldCheckItem struct {
	Name   string `json:"name" validate:"required,min=3,max=150"`
	Result string `json:"result" validate:"required,oneof=ok not_ok"`
	Note   string `json:"note" validate:"omitempty,max=500"`
}

type FieldImageItem struct {
	ImageUrl string `json:"image_url" validate:"required,startswith=/api/uploads/"`
}

type CreateFieldMaintenanceRequest struct {
	CCTVID         string           `json:"cctv_id" validate:"required,uuid4"`
	Notes          string           `json:"notes" validate:"omitempty,max=1000"`
	CheckLatitude  string           `json:"check_latitude"`
	CheckLongitude string           `json:"check_longitude"`
	Checks         []FieldCheckItem `json:"checks" validate:"required,min=1,dive"`
	Images         []FieldImageItem `json:"images" validate:"required,min=1,max=5,dive"`
}

// validateCoordinates memvalidasi pasangan koordinat geotag (opsional).
// Bila keduanya kosong maka geotag diabaikan (diizinkan). Bila salah satu
// atau keduanya diisi, keduanya harus berupa angka valid: latitude dalam
// rentang [-90, 90] dan longitude dalam rentang [-180, 180].
func validateCoordinates(lat, lng string) error {
	lat = strings.TrimSpace(lat)
	lng = strings.TrimSpace(lng)

	// Keduanya kosong → geotag opsional, tidak perlu divalidasi.
	if lat == "" && lng == "" {
		return nil
	}

	// Salah satu diisi tetapi yang lain kosong → tidak lengkap.
	if lat == "" || lng == "" {
		return errors.New("check_latitude dan check_longitude harus diisi bersamaan (keduanya wajib atau keduanya kosong)")
	}

	latVal, err := strconv.ParseFloat(lat, 64)
	if err != nil {
		return errors.New("check_latitude harus berupa angka yang valid")
	}
	if latVal < -90 || latVal > 90 {
		return errors.New("check_latitude harus berada dalam rentang -90 sampai 90")
	}

	lngVal, err := strconv.ParseFloat(lng, 64)
	if err != nil {
		return errors.New("check_longitude harus berupa angka yang valid")
	}
	if lngVal < -180 || lngVal > 180 {
		return errors.New("check_longitude harus berada dalam rentang -180 sampai 180")
	}

	return nil
}

// CreateFieldMaintenance mencatat hasil pengecekan lapangan teknisi (Admin & Teknisi)
// @Summary      Catat Pengecekan Lapangan
// @Description  Membuat catatan pengecekan lapangan untuk aset CCTV dengan daftar checklist item yang wajib diisi dan minimal satu foto bukti. ID Teknisi dideteksi otomatis dari token JWT. Status CCTV ikut disinkronkan otomatis (konservatif): hasil "attention" → "pemeliharaan", hasil "ok" → "aktif"; status "nonaktif" tidak pernah ditimpa otomatis.
// @Tags         Pengecekan Lapangan
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body CreateFieldMaintenanceRequest true "Detail Pengecekan Lapangan"
// @Success      201 {object} map[string]interface{} "Catatan berhasil dibuat"
// @Failure      400 {object} map[string]interface{} "Format tidak valid, aset tidak ditemukan, data tidak lengkap"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Router       /field-maintenance [post]
func CreateFieldMaintenance(c *fiber.Ctx) error {
	var req CreateFieldMaintenanceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Format input tidak valid",
		})
	}

	// Batasan struktural tambahan (maks item & foto) di luar validator struct
	if len(req.Checks) == 0 || len(req.Checks) > maxFieldChecks {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Checklist wajib diisi 1–" + strconv.Itoa(maxFieldChecks) + " item",
		})
	}
	if len(req.Images) == 0 || len(req.Images) > maxFieldImages {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Minimal satu foto bukti wajib dilampirkan (maksimal " + strconv.Itoa(maxFieldImages) + " foto)",
		})
	}

	if errs := utils.ValidateStruct(&req); len(errs) > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Validasi input gagal",
			"errors":  errs,
		})
	}

	// Validasi koordinat geotag (opsional). Menghasilkan 400 bila koordinat
	// diisi tetapi tidak valid secara format maupun rentang.
	if err := validateCoordinates(req.CheckLatitude, req.CheckLongitude); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	cctvUUID, err := uuid.Parse(req.CCTVID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "CCTV ID tidak valid",
		})
	}

	var cctv models.CCTV
	if err := config.DB.First(&cctv, "id = ?", cctvUUID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "CCTV tidak ditemukan",
		})
	}

	techIDStr, ok := c.Locals("user_id").(string)
	if !ok || techIDStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Sesi tidak valid, Teknisi tidak ditemukan",
		})
	}

	techUUID, err := uuid.Parse(techIDStr)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal membaca ID teknisi",
		})
	}

	// Ringkasan: seluruh item ok → "ok", ada satu saja tidak ok → "attention"
	overall := "ok"
	for _, check := range req.Checks {
		if check.Result == "not_ok" {
			overall = "attention"
			break
		}
	}

	now := time.Now()
	entry := models.FieldMaintenance{
		CCTVID:         cctvUUID,
		TechnicianID:   techUUID,
		OverallStatus:  overall,
		Notes:          strings.TrimSpace(req.Notes),
		CheckLatitude:  strings.TrimSpace(req.CheckLatitude),
		CheckLongitude: strings.TrimSpace(req.CheckLongitude),
		CompletedAt:    &now,
	}

	// Simpan header + checklist + gambar dalam satu transaksi agar konsisten
	if err := config.DB.Transaction(func(tx *gorm.DB) error {
		if createErr := tx.Create(&entry).Error; createErr != nil {
			return createErr
		}
		for _, check := range req.Checks {
			item := models.FieldMaintenanceCheck{
				FieldMaintenanceID: entry.ID,
				Name:               strings.TrimSpace(check.Name),
				Result:             check.Result,
				Note:               strings.TrimSpace(check.Note),
			}
			if createErr := tx.Create(&item).Error; createErr != nil {
				return createErr
			}
		}
		for _, img := range req.Images {
			image := models.FieldMaintenanceImage{
				FieldMaintenanceID: entry.ID,
				ImageUrl:           img.ImageUrl,
			}
			if createErr := tx.Create(&image).Error; createErr != nil {
				return createErr
			}
		}

		// Fase 5.2: sinkronisasi status CCTV otomatis (konservatif) dalam
		// transaksi yang sama — hasil "attention" memindahkan CCTV ke
		// "pemeliharaan"; hasil "ok" mengembalikannya ke "aktif".
		targetStatus := "aktif"
		syncReason := "pengecekan lapangan menunjukkan semua item OK"
		if overall == "attention" {
			targetStatus = "pemeliharaan"
			syncReason = "hasil pengecekan lapangan memerlukan perhatian"
		}
		if err := syncCCTVStatus(tx, c, cctvUUID, targetStatus, syncReason); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menyimpan catatan pengecekan lapangan",
		})
	}

	utils.LogAudit(config.DB, c, "CREATE_FIELD_MAINTENANCE", "MAINTENANCE", "Mencatat pengecekan lapangan CCTV "+cctv.KodeCCTV+" (Hasil: "+overall+")", nil, "", "")

	// Notifikasi in-app ke seluruh admin bila hasil pengecekan perlu perhatian
	// (ada item tidak ok) — Fase 5.1. Hasil "ok" tidak memicu notifikasi.
	if overall == "attention" {
		refID := entry.ID
		utils.NotifyRole(config.DB, "admin", models.NotificationTypeFieldCheck,
			"Pengecekan lapangan butuh perhatian",
			"Pengecekan lapangan CCTV "+cctv.KodeCCTV+" menunjukkan ada item yang tidak ok ("+cctv.LokasiArea+").",
			"field-maintenance", &refID)
	}

	// Muat ulang dengan relasi lengkap untuk respon JSON
	config.DB.Preload("CCTV").Preload("User").Preload("Checks").Preload("Images").First(&entry, entry.ID)

	// Hasil field-check mengubah salah satu faktor skor, jadi hitung ulang
	// setelah seluruh header dan checklist berhasil disimpan.
	if err := RecalculateCCTVHealthScore(config.DB, cctvUUID); err != nil {
		utils.Log.Errorf("Gagal memperbarui skor kesehatan CCTV %s: %v", cctvUUID.String(), err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status": "success",
		"data":   entry,
	})
}

// GetAllFieldMaintenances mengambil daftar pengecekan lapangan (Admin: semua, Teknisi: miliknya)
// @Summary      Ambil Semua Pengecekan Lapangan
// @Description  Mendapatkan daftar catatan pengecekan lapangan terpaginasi. Admin melihat semua catatan, Teknisi hanya melihat miliknya sendiri.
// @Tags         Pengecekan Lapangan
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        cctv_id query string false "Filter berdasarkan UUID CCTV"
// @Param        status query string false "Filter berdasarkan hasil (ok / attention)"
// @Param        start_date query string false "Filter rentang awal (YYYY-MM-DD)"
// @Param        end_date query string false "Filter rentang akhir (YYYY-MM-DD)"
// @Param        search query string false "Cari catatan atau nama CCTV"
// @Param        page query integer false "Nomor Halaman (default: 1)"
// @Param        limit query integer false "Jumlah data per halaman (default: 10)"
// @Param        sort_by query string false "Kolom pengurutan (default: created_at)"
// @Param        order query string false "Arah pengurutan: asc, desc (default: desc)"
// @Success      200 {object} map[string]interface{} "Daftar pengecekan terpaginasi"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Router       /field-maintenance [get]
func GetAllFieldMaintenances(c *fiber.Ctx) error {
	var entries []models.FieldMaintenance
	query := config.DB.Model(&models.FieldMaintenance{}).
		Preload("CCTV").Preload("User").Preload("Checks").Preload("Images")

	// Role scope: teknisi hanya melihat catatan miliknya sendiri
	userRole, _ := c.Locals("role").(string)
	if userRole != "admin" {
		techIDStr, ok := c.Locals("user_id").(string)
		if !ok || techIDStr == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "Sesi tidak valid, Teknisi tidak ditemukan",
			})
		}
		query = query.Where("technician_id = ?", techIDStr)
	}

	if cctvID := c.Query("cctv_id"); cctvID != "" {
		cctvUUID, err := uuid.Parse(cctvID)
		if err == nil {
			query = query.Where("cctv_id = ?", cctvUUID)
		}
	}

	if status := c.Query("status"); status != "" {
		query = query.Where("overall_status = ?", status)
	}

	if start, end := utils.ParseDateRangeParams(c); start != nil || end != nil {
		if start != nil {
			query = query.Where("created_at >= ?", *start)
		}
		if end != nil {
			query = query.Where("created_at < ?", *end)
		}
	}

	if search := c.Query("search"); search != "" {
		query = query.Where("notes ILIKE ? OR cctv_id IN (SELECT id FROM cctvs WHERE nama ILIKE ? OR kode_cctv ILIKE ?)",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	allowedSort := []string{"id", "cctv_id", "technician_id", "overall_status", "created_at", "completed_at"}
	limit, page, sort := utils.GetPaginationParams(c, allowedSort)

	var totalRows int64
	if err := query.Count(&totalRows).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghitung total pengecekan lapangan",
		})
	}

	offset := (page - 1) * limit
	if err := query.Offset(offset).Limit(limit).Order(sort).Find(&entries).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal mengambil daftar pengecekan lapangan",
		})
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"rows":       entries,
			"pagination": utils.CalculatePagination(totalRows, page, limit),
		},
	})
}

// GetFieldMaintenanceByID mengambil satu catatan pengecekan lapangan (Admin & Teknisi)
// @Summary      Ambil Pengecekan Lapangan Detail
// @Description  Mendapatkan detail satu catatan pengecekan lapangan berdasarkan UUID-nya.
// @Tags         Pengecekan Lapangan
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Catatan UUID"
// @Success      200 {object} map[string]interface{} "Data catatan ditemukan"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      404 {object} map[string]interface{} "Catatan tidak ditemukan"
// @Router       /field-maintenance/{id} [get]
func GetFieldMaintenanceByID(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID pengecekan lapangan tidak valid",
		})
	}

	var entry models.FieldMaintenance
	if err := config.DB.Preload("CCTV").Preload("User").Preload("Checks").Preload("Images").
		First(&entry, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Catatan pengecekan lapangan tidak ditemukan",
		})
	}

	// SECURITY: Teknisi hanya boleh membuka catatan miliknya sendiri
	userRole, _ := c.Locals("role").(string)
	if userRole != "admin" {
		techIDStr, ok := c.Locals("user_id").(string)
		if !ok || techIDStr != entry.TechnicianID.String() {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"status":  "error",
				"message": "Akses ditolak: Anda hanya dapat melihat catatan pengecekan milik Anda sendiri",
			})
		}
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   entry,
	})
}

// GenerateFieldMaintenanceReportPDF membuat laporan PDF satu catatan pengecekan
// lapangan. Admin dapat membuka semua catatan, sedangkan teknisi hanya dapat
// mengunduh catatan yang menjadi miliknya sendiri.
// @Summary      Laporan Pengecekan Lapangan (PDF)
// @Description  Mengunduh laporan PDF satu catatan pengecekan lapangan lengkap dengan kop surat IKN, identitas CCTV dan teknisi, checklist, foto bukti, serta catatan umum.
// @Tags         Pengecekan Lapangan
// @Produce      application/pdf
// @Security     BearerAuth
// @Param        id path string true "Catatan UUID"
// @Success      200 {file} string "File PDF laporan pengecekan lapangan"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      403 {object} map[string]interface{} "Akses ditolak"
// @Failure      404 {object} map[string]interface{} "Catatan tidak ditemukan"
// @Failure      500 {object} map[string]interface{} "Gagal membuat laporan"
// @Router       /field-maintenance/{id}/report-pdf [get]
func GenerateFieldMaintenanceReportPDF(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID pengecekan lapangan tidak valid",
		})
	}

	var entry models.FieldMaintenance
	if err := config.DB.Preload("CCTV").Preload("User").Preload("Checks").Preload("Images").
		First(&entry, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"status":  "error",
				"message": "Catatan pengecekan lapangan tidak ditemukan",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal mengambil data laporan pengecekan lapangan",
		})
	}

	// SECURITY: scope teknisi tetap ditegakkan di backend walaupun tombol
	// laporan pada frontend hanya ditampilkan untuk Admin.
	userRole, _ := c.Locals("role").(string)
	if userRole != "admin" {
		userID, _ := c.Locals("user_id").(string)
		if userID == "" || userID != entry.TechnicianID.String() {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"status":  "error",
				"message": "Akses ditolak: Anda hanya dapat mengunduh laporan pengecekan milik Anda sendiri",
			})
		}
	}

	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(pdfMarginX, 12, pdfMarginX)
	pdf.SetAutoPageBreak(true, 16)
	pdf.AddPage()
	drawFieldMaintenanceReport(pdf, entry)

	filename := fmt.Sprintf("laporan_pengecekan_lapangan_%s.pdf", entry.ID.String())
	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	var out bytes.Buffer
	if err := pdf.Output(&out); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghasilkan berkas laporan PDF",
		})
	}
	return c.Send(out.Bytes())
}

// drawFieldMaintenanceReport menyusun seluruh bagian laporan pengecekan
// lapangan menggunakan kop surat yang sama dengan laporan pemeliharaan.
func drawFieldMaintenanceReport(pdf *fpdf.Fpdf, entry models.FieldMaintenance) {
	drawLetterhead(pdf)

	pdf.SetY(pdf.GetY() + 3)
	pdf.SetFont("Helvetica", "B", 14)
	pdf.SetTextColor(30, 30, 30)
	pdf.CellFormat(pdfContentW, 8, "LAPORAN PENGECEKAN LAPANGAN", "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(110, 110, 110)
	pdf.CellFormat(pdfContentW, 5, "Dihasilkan: "+time.Now().Format("2006-01-02 15:04:05"), "", 1, "C", false, 0, "")
	pdf.SetTextColor(30, 30, 30)

	drawFieldReportSectionTitle(pdf, "INFORMASI CCTV")
	drawFieldReportInfoGrid(pdf, [][2]string{
		{"Kode CCTV", pdfSafe(entry.CCTV.KodeCCTV, 30)},
		{"Nama CCTV", pdfSafe(entry.CCTV.Nama, 42)},
		{"Lokasi", pdfSafe(entry.CCTV.LokasiArea, 42)},
		{"Status CCTV", fieldReportCCTVStatusLabel(entry.CCTV.Status)},
	})

	drawFieldReportSectionTitle(pdf, "INFORMASI PENGECEKAN")
	technician := entry.User.FullName
	if strings.TrimSpace(technician) == "" {
		technician = entry.User.Username
	}
	checkTime := entry.CreatedAt
	if entry.CompletedAt != nil {
		checkTime = *entry.CompletedAt
	}
	drawFieldReportInfoGrid(pdf, [][2]string{
		{"Teknisi", pdfSafe(technician, 42)},
		{"Tanggal / waktu", checkTime.Format("2006-01-02 15:04:05")},
		{"Status keseluruhan", fieldReportOverallStatusLabel(entry.OverallStatus)},
		{"Jumlah item", fmt.Sprintf("%d item checklist", len(entry.Checks))},
	})

	drawFieldReportSectionTitle(pdf, "TABEL CHECKLIST")
	drawFieldReportChecklist(pdf, entry.Checks)

	drawFieldReportSectionTitle(pdf, "FOTO BUKTI")
	drawFieldReportPhotos(pdf, entry.Images)

	if strings.TrimSpace(entry.Notes) != "" {
		drawFieldReportSectionTitle(pdf, "CATATAN UMUM")
		ensureFieldReportSpace(pdf, 14)
		pdf.SetFont("Helvetica", "", 9)
		pdf.SetTextColor(45, 45, 45)
		pdf.MultiCell(pdfContentW, 5, pdfSafe(entry.Notes, 2000), "1", "L", false)
	}
}

func drawFieldReportSectionTitle(pdf *fpdf.Fpdf, title string) {
	ensureFieldReportSpace(pdf, 12)
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetTextColor(64, 112, 56)
	pdf.CellFormat(pdfContentW, 6, title, "", 1, "L", false, 0, "")
	pdf.SetDrawColor(210, 210, 210)
	pdf.SetLineWidth(0.2)
	pdf.Line(pdfMarginX, pdf.GetY(), pdfMarginX+pdfContentW, pdf.GetY())
	pdf.SetY(pdf.GetY() + 2)
	pdf.SetTextColor(30, 30, 30)
}

func drawFieldReportInfoGrid(pdf *fpdf.Fpdf, rows [][2]string) {
	const rowHeight = 8.0
	colWidth := pdfContentW / 2
	for i := 0; i < len(rows); i += 2 {
		ensureFieldReportSpace(pdf, rowHeight)
		y := pdf.GetY()
		for col := 0; col < 2 && i+col < len(rows); col++ {
			x := pdfMarginX + float64(col)*colWidth
			pdf.SetFillColor(248, 248, 244)
			pdf.SetDrawColor(220, 220, 210)
			pdf.Rect(x, y, colWidth, rowHeight, "DF")
			pdf.SetXY(x+3, y+1.2)
			pdf.SetFont("Helvetica", "B", 7.5)
			pdf.SetTextColor(105, 105, 105)
			pdf.CellFormat(colWidth-6, 3.5, rows[i+col][0], "", 1, "L", false, 0, "")
			pdf.SetFont("Helvetica", "", 8.5)
			pdf.SetTextColor(35, 35, 35)
			pdf.CellFormat(colWidth-6, 3.5, rows[i+col][1], "", 0, "L", false, 0, "")
		}
		pdf.SetY(y + rowHeight)
	}
	pdf.SetY(pdf.GetY() + 2)
}

func drawFieldReportChecklist(pdf *fpdf.Fpdf, checks []models.FieldMaintenanceCheck) {
	colWidths := []float64{10, 57, 27, 88}
	headers := []string{"No", "Nama Item", "Hasil", "Catatan"}

	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetFillColor(64, 112, 56)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetX(pdfMarginX)
	for i, header := range headers {
		pdf.CellFormat(colWidths[i], 7, header, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)
	pdf.SetTextColor(30, 30, 30)

	for i, check := range checks {
		name := pdfSafe(check.Name, 90)
		note := pdfSafe(check.Note, 140)
		nameLines := pdf.SplitLines([]byte(name), colWidths[1]-4)
		noteLines := pdf.SplitLines([]byte(note), colWidths[3]-4)
		lineCount := len(nameLines)
		if len(noteLines) > lineCount {
			lineCount = len(noteLines)
		}
		cellHeight := float64(lineCount)*4.5 + 3
		if cellHeight < 8 {
			cellHeight = 8
		}

		if pdf.GetY()+cellHeight > pdfBreakY {
			pdf.AddPage()
			drawLetterhead(pdf)
			drawFieldReportSectionTitle(pdf, "TABEL CHECKLIST (LANJUTAN)")
			pdf.SetFont("Helvetica", "B", 8)
			pdf.SetFillColor(64, 112, 56)
			pdf.SetTextColor(255, 255, 255)
			pdf.SetX(pdfMarginX)
			for j, header := range headers {
				pdf.CellFormat(colWidths[j], 7, header, "1", 0, "C", true, 0, "")
			}
			pdf.Ln(-1)
			pdf.SetTextColor(30, 30, 30)
		}

		y := pdf.GetY()
		x := pdfMarginX
		pdf.SetDrawColor(205, 205, 205)
		for _, width := range colWidths {
			pdf.Rect(x, y, width, cellHeight, "D")
			x += width
		}

		pdf.SetFont("Helvetica", "", 8)
		pdf.SetXY(pdfMarginX, y+1.5)
		pdf.CellFormat(colWidths[0], 5, fmt.Sprintf("%d", i+1), "", 0, "C", false, 0, "")
		pdf.SetXY(pdfMarginX+colWidths[0]+2, y+1.5)
		pdf.MultiCell(colWidths[1]-4, 4.5, name, "", "L", false)
		pdf.SetXY(pdfMarginX+colWidths[0]+colWidths[1], y+1.5)
		pdf.SetFont("Helvetica", "B", 8)
		pdf.CellFormat(colWidths[2], 5, fieldReportCheckResultLabel(check.Result), "", 0, "C", false, 0, "")
		pdf.SetXY(pdfMarginX+colWidths[0]+colWidths[1]+colWidths[2]+2, y+1.5)
		pdf.SetFont("Helvetica", "", 8)
		pdf.MultiCell(colWidths[3]-4, 4.5, note, "", "L", false)
		pdf.SetY(y + cellHeight)
	}
	pdf.SetY(pdf.GetY() + 2)
}

func drawFieldReportPhotos(pdf *fpdf.Fpdf, images []models.FieldMaintenanceImage) {
	if len(images) == 0 {
		pdf.SetFont("Helvetica", "I", 9)
		pdf.SetTextColor(110, 110, 110)
		pdf.CellFormat(pdfContentW, 7, "Tidak ada foto bukti.", "", 1, "L", false, 0, "")
		pdf.SetTextColor(30, 30, 30)
		return
	}

	const (
		photoWidth  = 87.0
		photoHeight = 48.0
		photoGap    = 8.0
	)
	for i := 0; i < len(images); i += 2 {
		ensureFieldReportSpace(pdf, photoHeight+8)
		y := pdf.GetY()
		for col := 0; col < 2 && i+col < len(images); col++ {
			x := pdfMarginX + float64(col)*(photoWidth+photoGap)
			drawFieldReportPhoto(pdf, images[i+col], i+col+1, x, y, photoWidth, photoHeight)
		}
		pdf.SetY(y + photoHeight + 7)
	}
}

func drawFieldReportPhoto(pdf *fpdf.Fpdf, imageRecord models.FieldMaintenanceImage, number int, x, y, width, height float64) {
	pdf.SetDrawColor(205, 205, 205)
	pdf.Rect(x, y, width, height, "D")

	imagePath, imageType := fieldReportImagePath(imageRecord.ImageUrl)
	if imagePath != "" {
		imageWidth, imageHeight, ok := fieldReportImageDimensions(imagePath)
		if ok {
			pdf.RegisterImage(imagePath, imageType)
			if !pdf.Err() {
				const padding = 2.0
				maxWidth := width - padding*2
				maxHeight := height - 8
				ratio := float64(imageWidth) / float64(imageHeight)
				drawWidth := maxWidth
				drawHeight := drawWidth / ratio
				if drawHeight > maxHeight {
					drawHeight = maxHeight
					drawWidth = drawHeight * ratio
				}
				pdf.Image(imagePath, x+(width-drawWidth)/2, y+2+(maxHeight-drawHeight)/2, drawWidth, drawHeight, false, "", 0, "")
				if !pdf.Err() {
					pdf.SetFont("Helvetica", "", 7)
					pdf.SetTextColor(110, 110, 110)
					pdf.SetXY(x+2, y+height-5.5)
					pdf.CellFormat(width-4, 4, fmt.Sprintf("Foto bukti %d", number), "", 0, "C", false, 0, "")
					pdf.SetTextColor(30, 30, 30)
					return
				}
			}
		}
	}

	// Bila file lokal tidak tersedia, tetap sertakan URL agar bukti dapat
	// ditelusuri dari laporan tanpa membuat proses generate PDF gagal.
	pdf.SetFont("Helvetica", "", 7.5)
	pdf.SetTextColor(90, 90, 90)
	pdf.SetXY(x+3, y+8)
	pdf.MultiCell(width-6, 4, "Foto bukti "+fmt.Sprintf("%d", number)+"\nURL: "+pdfSafe(imageRecord.ImageUrl, 42), "", "L", false)
	pdf.SetTextColor(30, 30, 30)
}

func fieldReportImagePath(imageURL string) (string, string) {
	path := strings.TrimSpace(imageURL)
	var filename string
	switch {
	case strings.HasPrefix(path, "/api/uploads/"):
		filename = strings.TrimPrefix(path, "/api/uploads/")
	case strings.HasPrefix(path, "/uploads/"):
		filename = strings.TrimPrefix(path, "/uploads/")
	default:
		return "", ""
	}

	if filename == "" || filepath.Base(filename) != filename || strings.Contains(filename, "..") {
		return "", ""
	}
	ext := strings.ToLower(filepath.Ext(filename))
	imageType := strings.TrimPrefix(ext, ".")
	if imageType == "jpeg" {
		imageType = "jpg"
	}
	if imageType != "jpg" && imageType != "png" {
		return "", ""
	}

	filePath := filepath.Join("uploads", filename)
	info, err := os.Stat(filePath)
	if err != nil || !info.Mode().IsRegular() {
		return "", ""
	}
	return filePath, imageType
}

func fieldReportImageDimensions(filePath string) (int, int, bool) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, 0, false
	}
	defer func() { _ = file.Close() }()

	config, _, err := image.DecodeConfig(file)
	if err != nil || config.Width <= 0 || config.Height <= 0 {
		return 0, 0, false
	}
	return config.Width, config.Height, true
}

func ensureFieldReportSpace(pdf *fpdf.Fpdf, height float64) {
	if pdf.GetY()+height <= pdfBreakY {
		return
	}
	pdf.AddPage()
	drawLetterhead(pdf)
}

func fieldReportCheckResultLabel(result string) string {
	if strings.EqualFold(strings.TrimSpace(result), "ok") {
		return "OK"
	}
	return "Tidak OK"
}

func fieldReportOverallStatusLabel(status string) string {
	if strings.EqualFold(strings.TrimSpace(status), "ok") {
		return "OK"
	}
	return "ATTENTION"
}

func fieldReportCCTVStatusLabel(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "aktif", "active":
		return "Aktif"
	case "pemeliharaan", "maintenance":
		return "Pemeliharaan"
	case "nonaktif", "inactive", "rusak", "broken":
		return "Nonaktif"
	default:
		return pdfSafe(status, 24)
	}
}

// DeleteFieldMaintenance menghapus catatan pengecekan lapangan (Admin saja)
// @Summary      Hapus Pengecekan Lapangan
// @Description  Menghapus catatan pengecekan lapangan beserta checklist dan gambar buktinya (soft delete).
// @Tags         Pengecekan Lapangan
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Catatan UUID"
// @Success      200 {object} map[string]interface{} "Catatan berhasil dihapus"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      404 {object} map[string]interface{} "Catatan tidak ditemukan"
// @Router       /field-maintenance/{id} [delete]
func DeleteFieldMaintenance(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID pengecekan lapangan tidak valid",
		})
	}

	var entry models.FieldMaintenance
	if err := config.DB.First(&entry, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Catatan pengecekan lapangan tidak ditemukan",
		})
	}

	if err := config.DB.Delete(&entry).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghapus catatan pengecekan lapangan",
		})
	}

	utils.LogAudit(config.DB, c, "DELETE_FIELD_MAINTENANCE", "MAINTENANCE", "Menghapus catatan pengecekan lapangan CCTV "+entry.CCTVID.String(), nil, "", "")

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Catatan pengecekan lapangan berhasil dihapus",
	})
}
