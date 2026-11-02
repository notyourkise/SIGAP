package handlers

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"sigap-backend/config"
	"sigap-backend/models"
	"sigap-backend/utils"

	"github.com/go-pdf/fpdf"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
)

// Entity supported oleh report builder.
const (
	ReportEntityCCTV             = "cctv"
	ReportEntityMaintenanceLog   = "maintenance_log"
	ReportEntityFieldMaintenance = "field_maintenance"
	ReportEntitySparePart        = "spare_part"
)

// ReportFormat supported output.
const (
	ReportFormatCSV  = "csv"
	ReportFormatXLSX = "xlsx"
	ReportFormatPDF  = "pdf"
)

// reportColumn mendeskripsikan satu kolom yang dapat diekspor.
type reportColumn struct {
	Key   string
	Label string
}

// reportEntityMeta mendeskripsikan entitas laporan beserta kolom yang tersedia.
type reportEntityMeta struct {
	Entity  string
	Columns []reportColumn
}

// reportEntityMetas mendefinisikan kolom yang tersedia per entitas.
var reportEntityMetas = map[string]reportEntityMeta{
	ReportEntityCCTV: {
		Entity: ReportEntityCCTV,
		Columns: []reportColumn{
			{"kode_cctv", "Kode CCTV"},
			{"nama", "Nama"},
			{"lokasi_area", "Lokasi"},
			{"jenis_cctv", "Jenis"},
			{"status", "Status"},
			{"ip_camera", "IP Kamera"},
			{"ip_nvr", "IP NVR"},
			{"pemilik", "Pemilik"},
			{"created_at", "Dibuat"},
		},
	},
	ReportEntityMaintenanceLog: {
		Entity: ReportEntityMaintenanceLog,
		Columns: []reportColumn{
			{"cctv_nama", "CCTV"},
			{"cctv_kode", "Kode CCTV"},
			{"action_taken", "Tindakan"},
			{"status", "Status"},
			{"technician", "Teknisi"},
			{"scheduled_date", "Terjadwal"},
			{"completed_at", "Selesai"},
		},
	},
	ReportEntityFieldMaintenance: {
		Entity: ReportEntityFieldMaintenance,
		Columns: []reportColumn{
			{"cctv_nama", "CCTV"},
			{"cctv_kode", "Kode CCTV"},
			{"technician", "Teknisi"},
			{"overall_status", "Hasil"},
			{"notes", "Catatan"},
			{"completed_at", "Waktu"},
		},
	},
	ReportEntitySparePart: {
		Entity: ReportEntitySparePart,
		Columns: []reportColumn{
			{"nama", "Nama"},
			{"sku", "SKU"},
			{"kategori", "Kategori"},
			{"stok", "Stok"},
			{"minimum_order", "Min. Order"},
			{"satuan", "Satuan"},
			{"harga_satuan", "Harga"},
		},
	},
}

// GenerateReportRequest adalah body pembuatan laporan kustom.
type GenerateReportRequest struct {
	Entity  string                 `json:"entity" validate:"required,oneof=cctv maintenance_log field_maintenance spare_part"`
	Columns []string               `json:"columns" validate:"required,min=1"`
	Filters map[string]interface{} `json:"filters"`
	Format  string                 `json:"format" validate:"required,oneof=csv xlsx pdf"`
}

// SaveReportTemplateRequest adalah body penyimpanan template laporan.
type SaveReportTemplateRequest struct {
	Nama   string `json:"nama" validate:"required,min=2,max=100"`
	Config string `json:"config" validate:"required"`
}

// GenerateReport membuat laporan kustom (CSV/XLSX/PDF) berdasarkan entitas,
// kolom, filter, dan format yang diminta (Admin saja).
// @Summary      Buat Laporan Kustom
// @Description  Menghasilkan laporan dari entitas (cctv, maintenance_log, field_maintenance, spare_part) dalam format CSV/Excel/PDF, dengan pilihan kolom dan filter.
// @Tags         Laporan
// @Accept       json
// @Produce      application/octet-stream
// @Security     BearerAuth
// @Param        request body GenerateReportRequest true "Konfigurasi Laporan"
// @Success      200 {file} string "File laporan diunduh"
// @Failure      400 {object} map[string]interface{} "Validasi/format tidak valid"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin"
// @Failure      500 {object} map[string]interface{} "Gagal membuat laporan"
// @Router       /reports/generate [post]
func GenerateReport(c *fiber.Ctx) error {
	var req GenerateReportRequest
	if err := c.BodyParser(&req); err != nil {
		return sendBodyParseError(c, &req, err)
	}
	if errs := utils.ValidateStruct(&req); len(errs) > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Validasi input gagal",
			"errors":  errs,
		})
	}

	// Validasi kolom yang diminta ada di metadata entitas.
	meta, ok := reportEntityMetas[req.Entity]
	if !ok {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Entitas laporan tidak dikenal",
		})
	}
	colKeySet := make(map[string]bool)
	for _, col := range meta.Columns {
		colKeySet[col.Key] = true
	}
	var selectedCols []reportColumn
	for _, key := range req.Columns {
		if colKeySet[key] {
			selectedCols = append(selectedCols, reportColumn{Key: key, Label: columnLabel(meta, key)})
		}
	}
	if len(selectedCols) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Tidak ada kolom valid yang dipilih",
		})
	}

	// Validasi filter tanggal sebelum eksekusi query
	if _, _, err := parseReportDateRange(req.Filters); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": err.Error(),
		})
	}

	rows, err := fetchReportRows(req.Entity, req.Filters)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal mengambil data laporan: " + err.Error(),
		})
	}

	// Proyeksi baris ke kolom terpilih.
	data := make([][]interface{}, 0, len(rows))
	for _, row := range rows {
		var line []interface{}
		for _, col := range selectedCols {
			line = append(line, row[col.Key])
		}
		data = append(data, line)
	}

	labels := make([]string, len(selectedCols))
	for i, col := range selectedCols {
		labels[i] = col.Label
	}

	switch req.Format {
	case ReportFormatCSV:
		return serveCSV(c, labels, data)
	case ReportFormatXLSX:
		return serveXLSX(c, labels, data)
	case ReportFormatPDF:
		return servePDF(c, req.Entity, labels, data)
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Format laporan tidak didukung",
		})
	}
}

// ListReportTemplates mengambil daftar template laporan tersimpan (Admin saja).
// @Summary      Ambil Semua Template Laporan
// @Description  Mendapatkan daftar template laporan kustom yang disimpan oleh admin.
// @Tags         Laporan
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]interface{} "Daftar template laporan"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin"
// @Failure      500 {object} map[string]interface{} "Gagal mengambil template"
// @Router       /reports/templates [get]
func ListReportTemplates(c *fiber.Ctx) error {
	var templates []models.ReportTemplate
	if err := config.DB.Order("created_at desc").Find(&templates).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal mengambil daftar template laporan",
		})
	}
	return c.JSON(fiber.Map{
		"status": "success",
		"data":   templates,
	})
}

// SaveReportTemplate menyimpan template laporan kustom (Admin saja).
// @Summary      Simpan Template Laporan
// @Description  Menyimpan template laporan kustom agar dapat dipakai ulang.
// @Tags         Laporan
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body SaveReportTemplateRequest true "Data Template"
// @Success      201 {object} map[string]interface{} "Template berhasil disimpan"
// @Failure      400 {object} map[string]interface{} "Validasi gagal"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin"
// @Failure      500 {object} map[string]interface{} "Gagal menyimpan template"
// @Router       /reports/templates [post]
func SaveReportTemplate(c *fiber.Ctx) error {
	var req SaveReportTemplateRequest
	if err := c.BodyParser(&req); err != nil {
		return sendBodyParseError(c, &req, err)
	}
	if errs := utils.ValidateStruct(&req); len(errs) > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Validasi input gagal",
			"errors":  errs,
		})
	}

	// Validasi config adalah JSON valid.
	if !json.Valid([]byte(req.Config)) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "Config template harus berupa JSON valid",
		})
	}

	userIDStr, _ := c.Locals("user_id").(string)
	createdBy, _ := uuid.Parse(userIDStr)

	template := models.ReportTemplate{
		Nama:      req.Nama,
		Config:    req.Config,
		CreatedBy: createdBy,
	}
	if err := config.DB.Create(&template).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menyimpan template laporan",
		})
	}

	utils.LogAudit(config.DB, c, "CREATE_REPORT_TEMPLATE", "REPORT",
		"Menyimpan template laporan: "+template.Nama, &createdBy, "", "")

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status": "success",
		"data":   template,
	})
}

// DeleteReportTemplate menghapus template laporan tersimpan (Admin saja).
// @Summary      Hapus Template Laporan
// @Description  Menghapus template laporan kustom (soft-delete).
// @Tags         Laporan
// @Security     BearerAuth
// @Param        id path string true "Template UUID"
// @Success      200 {object} map[string]interface{} "Template berhasil dihapus"
// @Failure      400 {object} map[string]interface{} "UUID tidak valid"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid"
// @Failure      403 {object} map[string]interface{} "Akses ditolak: Hanya Admin"
// @Failure      404 {object} map[string]interface{} "Template tidak ditemukan"
// @Failure      500 {object} map[string]interface{} "Gagal menghapus template"
// @Router       /reports/templates/{id} [delete]
func DeleteReportTemplate(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "ID template tidak valid",
		})
	}

	var template models.ReportTemplate
	if err := config.DB.First(&template, "id = ?", id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "Template laporan tidak ditemukan",
		})
	}

	if err := config.DB.Delete(&template).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghapus template laporan",
		})
	}

	utils.LogAudit(config.DB, c, "DELETE_REPORT_TEMPLATE", "REPORT",
		"Menghapus template laporan: "+template.Nama, nil, "", "")

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Template laporan berhasil dihapus",
	})
}

// --- Helpers ---

func columnLabel(meta reportEntityMeta, key string) string {
	for _, col := range meta.Columns {
		if col.Key == key {
			return col.Label
		}
	}
	return key
}

// fetchReportRows mengambil data mentah (map) dari entitas dengan filter.
func fetchReportRows(entity string, filters map[string]interface{}) ([]map[string]interface{}, error) {
	switch entity {
	case ReportEntityCCTV:
		return fetchCCTVReportRows(filters)
	case ReportEntityMaintenanceLog:
		return fetchMaintenanceReportRows(filters)
	case ReportEntityFieldMaintenance:
		return fetchFieldMaintenanceReportRows(filters)
	case ReportEntitySparePart:
		return fetchSparePartReportRows(filters)
	default:
		return nil, fmt.Errorf("entitas tidak dikenal")
	}
}

func filterString(filters map[string]interface{}, key string) string {
	if v, ok := filters[key]; ok {
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
	return ""
}

func fetchCCTVReportRows(filters map[string]interface{}) ([]map[string]interface{}, error) {
	var items []models.CCTV
	query := config.DB.Model(&models.CCTV{})
	if status := filterString(filters, "status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if search := filterString(filters, "search"); search != "" {
		like := "%" + search + "%"
		query = query.Where("nama ILIKE ? OR kode_cctv ILIKE ?", like, like)
	}
	start, end, err := parseReportDateRange(filters)
	if err != nil {
		return nil, err
	}
	if start != nil {
		query = query.Where("created_at >= ?", *start)
	}
	if end != nil {
		query = query.Where("created_at < ?", *end)
	}
	if err := query.Order("created_at desc").Find(&items).Error; err != nil {
		return nil, err
	}
	rows := make([]map[string]interface{}, 0, len(items))
	for _, it := range items {
		rows = append(rows, map[string]interface{}{
			"kode_cctv":   it.KodeCCTV,
			"nama":        it.Nama,
			"lokasi_area": it.LokasiArea,
			"jenis_cctv":  it.JenisCCTV,
			"status":      it.Status,
			"ip_camera":   it.IPCamera,
			"ip_nvr":      it.IPNVR,
			"pemilik":     it.Pemilik,
			"created_at":  it.CreatedAt.Format("2006-01-02"),
		})
	}
	return rows, nil
}

func fetchMaintenanceReportRows(filters map[string]interface{}) ([]map[string]interface{}, error) {
	var logs []models.MaintenanceLog
	query := config.DB.Model(&models.MaintenanceLog{}).Preload("CCTV").Preload("User")
	if status := filterString(filters, "status"); status != "" {
		query = query.Where("status = ?", status)
	}
	start, end, err := parseReportDateRange(filters)
	if err != nil {
		return nil, err
	}
	if start != nil {
		query = query.Where("created_at >= ?", *start)
	}
	if end != nil {
		query = query.Where("created_at < ?", *end)
	}
	if err := query.Order("created_at desc").Find(&logs).Error; err != nil {
		return nil, err
	}
	rows := make([]map[string]interface{}, 0, len(logs))
	for _, l := range logs {
		techName := ""
		if l.User != nil {
			techName = l.User.FullName
		}
		scheduled := ""
		if l.ScheduledDate != nil {
			scheduled = l.ScheduledDate.Format("2006-01-02")
		}
		completed := ""
		if l.CompletedAt != nil {
			completed = l.CompletedAt.Format("2006-01-02")
		}
		rows = append(rows, map[string]interface{}{
			"cctv_nama":      l.CCTV.Nama,
			"cctv_kode":      l.CCTV.KodeCCTV,
			"action_taken":   l.ActionTaken,
			"status":         l.Status,
			"technician":     techName,
			"scheduled_date": scheduled,
			"completed_at":   completed,
		})
	}
	return rows, nil
}

func fetchFieldMaintenanceReportRows(filters map[string]interface{}) ([]map[string]interface{}, error) {
	var items []models.FieldMaintenance
	query := config.DB.Model(&models.FieldMaintenance{}).Preload("CCTV").Preload("User")
	if result := filterString(filters, "overall_status"); result != "" {
		query = query.Where("overall_status = ?", result)
	} else if status := filterString(filters, "status"); status != "" {
		if status == "failed" || status == "attention" {
			query = query.Where("overall_status = ?", "attention")
		} else if status == "completed" || status == "ok" {
			query = query.Where("overall_status = ?", "ok")
		} else {
			query = query.Where("overall_status = ?", status)
		}
	}
	start, end, err := parseReportDateRange(filters)
	if err != nil {
		return nil, err
	}
	if start != nil {
		query = query.Where("created_at >= ?", *start)
	}
	if end != nil {
		query = query.Where("created_at < ?", *end)
	}
	if err := query.Order("created_at desc").Find(&items).Error; err != nil {
		return nil, err
	}
	rows := make([]map[string]interface{}, 0, len(items))
	for _, it := range items {
		techName := ""
		if it.User.FullName != "" {
			techName = it.User.FullName
		}
		completed := ""
		if it.CompletedAt != nil {
			completed = it.CompletedAt.Format("2006-01-02")
		}
		rows = append(rows, map[string]interface{}{
			"cctv_nama":      it.CCTV.Nama,
			"cctv_kode":      it.CCTV.KodeCCTV,
			"technician":     techName,
			"overall_status": it.OverallStatus,
			"notes":          it.Notes,
			"completed_at":   completed,
		})
	}
	return rows, nil
}

func fetchSparePartReportRows(filters map[string]interface{}) ([]map[string]interface{}, error) {
	var items []models.SparePart
	query := config.DB.Model(&models.SparePart{})
	if kategori := filterString(filters, "kategori"); kategori != "" {
		query = query.Where("kategori = ?", kategori)
	}
	if lowStock := filterString(filters, "low_stock"); lowStock == "true" {
		query = query.Where("stok < minimum_order")
	}
	if err := query.Order("created_at desc").Find(&items).Error; err != nil {
		return nil, err
	}
	rows := make([]map[string]interface{}, 0, len(items))
	for _, it := range items {
		rows = append(rows, map[string]interface{}{
			"nama":          it.Nama,
			"sku":           it.SKU,
			"kategori":      it.Kategori,
			"stok":          it.Stok,
			"minimum_order": it.MinimumOrder,
			"satuan":        it.Satuan,
			"harga_satuan":  it.HargaSatuan,
		})
	}
	return rows, nil
}

// parseReportDateRange membaca filter tanggal (start_date/end_date YYYY-MM-DD)
// menggunakan zona waktu operasional SIGAP (WITA / Asia/Makassar).
// Batas start diatur pada awal hari (00:00:00 WITA), dan end diatur pada awal
// hari berikutnya (00:00:00 WITA keesokan hari) agar komparasi `< end` mencakup seluruh hari end_date.
func parseReportDateRange(filters map[string]interface{}) (*time.Time, *time.Time, error) {
	if filters == nil {
		return nil, nil, nil
	}
	startStr := strings.TrimSpace(filterString(filters, "start_date"))
	endStr := strings.TrimSpace(filterString(filters, "end_date"))

	loc := utils.AppLocation()
	var start, end *time.Time

	if startStr != "" {
		t, err := time.ParseInLocation("2006-01-02", startStr, loc)
		if err != nil {
			return nil, nil, fmt.Errorf("format start_date tidak valid (harus YYYY-MM-DD)")
		}
		start = &t
	}
	if endStr != "" {
		t, err := time.ParseInLocation("2006-01-02", endStr, loc)
		if err != nil {
			return nil, nil, fmt.Errorf("format end_date tidak valid (harus YYYY-MM-DD)")
		}
		tEnd := t.AddDate(0, 0, 1)
		end = &tEnd
	}
	if start != nil && end != nil && end.Before(*start) {
		return nil, nil, fmt.Errorf("rentang tanggal tidak valid: start_date tidak boleh setelah end_date")
	}
	return start, end, nil
}

func cellString(v interface{}) string {
	switch val := v.(type) {
	case nil:
		return ""
	case string:
		return val
	case float64:
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(val, 'f', 2, 64)
	case int:
		return strconv.Itoa(val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

func serveCSV(c *fiber.Ctx, labels []string, data [][]interface{}) error {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	_ = writer.Write(labels)
	for _, row := range data {
		record := make([]string, len(row))
		for i, cell := range row {
			record[i] = cellString(cell)
		}
		_ = writer.Write(record)
	}
	writer.Flush()
	filename := fmt.Sprintf("laporan_%s.csv", time.Now().Format("20060102_150405"))
	c.Set("Content-Type", "text/csv; charset=UTF-8")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	return c.Send(buf.Bytes())
}

func serveXLSX(c *fiber.Ctx, labels []string, data [][]interface{}) error {
	file := excelize.NewFile()
	sheet := "Laporan"
	_, err := file.NewSheet(sheet)
	if err != nil {
		return err
	}
	if err := file.DeleteSheet("Sheet1"); err != nil {
		return err
	}

	for i, label := range labels {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = file.SetCellValue(sheet, cell, label)
	}
	for r, row := range data {
		for c, cell := range row {
			cellName, _ := excelize.CoordinatesToCellName(c+1, r+2)
			_ = file.SetCellValue(sheet, cellName, cellString(cell))
		}
	}

	var buf bytes.Buffer
	if err := file.Write(&buf); err != nil {
		return err
	}
	filename := fmt.Sprintf("laporan_%s.xlsx", time.Now().Format("20060102_150405"))
	c.Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	return c.Send(buf.Bytes())
}

func servePDF(c *fiber.Ctx, entity string, labels []string, data [][]interface{}) error {
	pdf := fpdf.New("L", "mm", "A4", "")
	pdf.SetMargins(10, 12, 10)
	pdf.SetAutoPageBreak(true, 14)
	pdf.SetFooterFunc(func() {
		pdf.SetY(-10)
		pdf.SetFont("Helvetica", "", 7)
		pdf.SetTextColor(105, 105, 105)
		pdf.CellFormat(277, 4, fmt.Sprintf("SIGAP - Laporan Kustom | Halaman %d", pdf.PageNo()), "", 0, "C", false, 0, "")
	})
	pdf.AddPage()

	pdf.SetFont("Helvetica", "B", 13)
	entityLabel := strings.ToUpper(strings.ReplaceAll(entity, "_", " "))
	pdf.CellFormat(0, 8, "LAPORAN KUSTOM - "+entityLabel, "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(110, 110, 110)
	pdf.CellFormat(0, 5, "Dihasilkan: "+time.Now().Format("2006-01-02 15:04:05"), "", 1, "C", false, 0, "")
	pdf.SetTextColor(30, 30, 30)

	pdf.Ln(2)
	columnWidths := reportPDFColumnWidths(pdf, labels, data)
	drawReportPDFHeader(pdf, labels, columnWidths)

	// Body
	for _, row := range data {
		drawReportPDFRow(pdf, labels, columnWidths, row)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return err
	}
	filename := fmt.Sprintf("laporan_%s.pdf", time.Now().Format("20060102_150405"))
	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	return c.Send(buf.Bytes())
}

// reportPDFColumnWidths memberi setiap kolom ruang minimum, lalu membagikan
// ruang tersisa berdasarkan panjang label dan data. Kolom Nama/IP tidak lagi
// dipaksa selebar kolom angka pendek.
func reportPDFColumnWidths(pdf *fpdf.Fpdf, labels []string, data [][]interface{}) []float64 {
	const (
		totalWidth = 277.0
		minWidth   = 18.0
	)
	weights := make([]float64, len(labels))
	weightTotal := 0.0
	for i, label := range labels {
		maxTextWidth := pdf.GetStringWidth(label)
		for _, row := range data {
			if i >= len(row) {
				continue
			}
			if width := pdf.GetStringWidth(cellString(row[i])); width > maxTextWidth {
				maxTextWidth = width
			}
		}
		weight := maxTextWidth / 18
		if weight < 1 {
			weight = 1
		}
		if weight > 3 {
			weight = 3
		}
		weights[i] = weight
		weightTotal += weight
	}

	widths := make([]float64, len(labels))
	remaining := totalWidth - minWidth*float64(len(labels))
	if remaining < 0 {
		remaining = 0
	}
	for i, weight := range weights {
		widths[i] = minWidth
		if weightTotal > 0 {
			widths[i] += remaining * weight / weightTotal
		}
	}
	return widths
}

func drawReportPDFHeader(pdf *fpdf.Fpdf, labels []string, widths []float64) {
	pdf.SetFont("Helvetica", "B", 7)
	const lineHeight = 4.0
	headerHeight := 8.0
	for i, label := range labels {
		if lines := len(pdf.SplitLines([]byte(label), widths[i]-3)); float64(lines)*lineHeight+2 > headerHeight {
			headerHeight = float64(lines)*lineHeight + 2
		}
	}

	x, y := 10.0, pdf.GetY()
	pdf.SetFillColor(64, 112, 56)
	pdf.SetDrawColor(42, 78, 38)
	pdf.SetTextColor(255, 255, 255)
	for i, label := range labels {
		pdf.Rect(x, y, widths[i], headerHeight, "DF")
		pdf.SetXY(x+1.5, y+1)
		pdf.MultiCell(widths[i]-3, lineHeight, label, "", "C", false)
		x += widths[i]
	}
	pdf.SetY(y + headerHeight)
	pdf.SetTextColor(30, 30, 30)
}

func drawReportPDFRow(pdf *fpdf.Fpdf, labels []string, widths []float64, row []interface{}) {
	pdf.SetFont("Helvetica", "", 7)
	values := make([]string, len(labels))
	const lineHeight = 4.0
	maxLines := 1
	for i := range labels {
		if i < len(row) {
			values[i] = strings.TrimSpace(cellString(row[i]))
		}
		if values[i] == "" {
			values[i] = "-"
		}
		if lines := len(pdf.SplitLines([]byte(values[i]), widths[i]-3)); lines > maxLines {
			maxLines = lines
		}
	}
	rowHeight := float64(maxLines)*lineHeight + 3
	if rowHeight < 8 {
		rowHeight = 8
	}

	if pdf.GetY()+rowHeight > 196 {
		pdf.AddPage()
		drawReportPDFHeader(pdf, labels, widths)
		pdf.SetFont("Helvetica", "", 7)
	}

	x, y := 10.0, pdf.GetY()
	pdf.SetDrawColor(205, 205, 205)
	for i, value := range values {
		pdf.Rect(x, y, widths[i], rowHeight, "D")
		pdf.SetXY(x+1.5, y+1.5)
		pdf.MultiCell(widths[i]-3, lineHeight, value, "", "L", false)
		x += widths[i]
	}
	pdf.SetY(y + rowHeight)
}
