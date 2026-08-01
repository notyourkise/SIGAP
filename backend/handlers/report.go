package handlers

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"sigap-backend/config"
	"sigap-backend/models"
	"sigap-backend/utils"

	"github.com/go-pdf/fpdf"
	"github.com/gofiber/fiber/v2"
)

const (
	pdfPageWidth  = 210.0 // A4 (mm)
	pdfMarginX    = 14.0
	pdfContentW   = pdfPageWidth - 2*pdfMarginX
	statusSelesai = "completed"
	statusGagal   = "failed"
)

// GenerateMaintenanceReportPDF membuat laporan pemeliharaan CCTV dalam format PDF
// (Admin & Teknisi), dengan opsi filter rentang tanggal & status.
// @Summary      Laporan Pemeliharaan CCTV (PDF)
// @Description  Mengunduh laporan riwayat pemeliharaan CCTV dalam format PDF, terfilter status dan rentang tanggal.
// @Tags         Log Pemeliharaan
// @Produce      application/pdf
// @Security     BearerAuth
// @Param        status query string false "Filter berdasarkan status log"
// @Param        start_date query string false "Filter rentang awal (YYYY-MM-DD)"
// @Param        end_date query string false "Filter rentang akhir (YYYY-MM-DD)"
// @Success      200 {file} string "File PDF Laporan Pemeliharaan"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid / Token tidak terdaftar"
// @Failure      500 {object} map[string]interface{} "Gagal membuat laporan"
// @Router       /maintenance-logs/report-pdf [get]
func GenerateMaintenanceReportPDF(c *fiber.Ctx) error {
	var logs []models.MaintenanceLog
	query := config.DB.Model(&models.MaintenanceLog{}).Preload("CCTV").Preload("User")

	// SECURITY (Fase 3.9): Teknisi hanya boleh mengunduh laporan data miliknya;
	// admin berhak atas seluruh data.
	query = applyRoleScope(c, query)

	// Filter berdasarkan status
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	// Filter rentang tanggal (created_at)
	if start, end := utils.ParseDateRangeParams(c); start != nil || end != nil {
		if start != nil {
			query = query.Where("created_at >= ?", *start)
		}
		if end != nil {
			query = query.Where("created_at < ?", *end)
		}
	}

	if err := query.Order("created_at desc").Find(&logs).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal mengambil data untuk laporan",
		})
	}

	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(pdfMarginX, 12, pdfMarginX)
	pdf.SetAutoPageBreak(true, 16)
	pdf.AddPage()

	// Kop laporan
	pdf.SetFont("Helvetica", "B", 15)
	pdf.CellFormat(pdfContentW, 8, "Laporan Pemeliharaan CCTV", "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(110, 110, 110)
	pdf.CellFormat(pdfContentW, 5, "Sistem Identifikasi Gangguan Aset Pusat (SIGAP) IKN", "", 1, "C", false, 0, "")
	pdf.CellFormat(pdfContentW, 5, "Periode: "+reportPeriod(c)+"   |   Dihasilkan: "+time.Now().Format("2006-01-02 15:04:05"), "", 1, "C", false, 0, "")
	pdf.SetTextColor(30, 30, 30)

	// Ringkasan
	summary := map[string]int{
		"Terjadwal":   0,
		"Dalam proses": 0,
		"Selesai":     0,
		"Gagal":       0,
	}
	for _, l := range logs {
		switch l.Status {
		case "scheduled":
			summary["Terjadwal"]++
		case "ongoing":
			summary["Dalam proses"]++
		case statusSelesai:
			summary["Selesai"]++
		case statusGagal:
			summary["Gagal"]++
		}
	}

	pdf.SetY(pdf.GetY() + 3)
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(pdfContentW, 6, fmt.Sprintf("Ringkasan: %d catatan pemeliharaan", len(logs)), "", 1, "", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(pdfContentW, 5, fmt.Sprintf(
		"Terjadwal: %d   |   Dalam proses: %d   |   Selesai: %d   |   Gagal: %d",
		summary["Terjadwal"], summary["Dalam proses"], summary["Selesai"], summary["Gagal"],
	), "", 1, "", false, 0, "")
	pdf.SetY(pdf.GetY() + 2)

	if len(logs) == 0 {
		pdf.SetFont("Helvetica", "", 10)
		pdf.SetTextColor(150, 150, 150)
		pdf.CellFormat(pdfContentW, 8, "Tidak ada catatan pemeliharaan pada periode tersebut.", "", 1, "C", false, 0, "")
		pdf.SetTextColor(30, 30, 30)
	} else {
		pdfTableHeader(pdf)
		for i, l := range logs {
			if pdf.GetY() > 260 {
				pdf.AddPage()
				pdfTableHeader(pdf)
			}
			pdfTableRow(pdf, i+1, l)
		}
	}

	filename := fmt.Sprintf("laporan_pemeliharaan_%s.pdf", time.Now().Format("20060102_150405"))
	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	var outBuf bytes.Buffer
	if err := pdf.Output(&outBuf); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal menghasilkan berkas laporan PDF",
		})
	}
	return c.Send(outBuf.Bytes())
}

// reportPeriod merangkum filter rentang tanggal dari query URL.
func reportPeriod(c *fiber.Ctx) string {
	start := strings.TrimSpace(c.Query("start_date"))
	end := strings.TrimSpace(c.Query("end_date"))
	switch {
	case start != "" && end != "":
		return start + " s.d. " + end
	case start != "":
		return "mulai " + start
	case end != "":
		return "hingga " + end
	default:
		return "Semua waktu"
	}
}

func pdfTableHeader(pdf *fpdf.Fpdf) {
	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetFillColor(64, 112, 56)
	pdf.SetTextColor(255, 255, 255)
	colWidths := []float64{9, 32, 22, 60, 20, 28, 19}
	headers := []string{"No", "Nama CCTV", "Kode CCTV", "Tindakan", "Status", "Teknisi", "Tanggal"}
	pdf.SetX(pdfMarginX)
	for i, h := range headers {
		pdf.CellFormat(colWidths[i], 7, h, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)
	pdf.SetTextColor(30, 30, 30)
}

func pdfTableRow(pdf *fpdf.Fpdf, no int, l models.MaintenanceLog) {
	colWidths := []float64{9, 32, 22, 60, 20, 28, 19}
	xStart := pdfMarginX
	yStart := pdf.GetY()

	cellHeight := 10.0
	actionLines := 1
	runes := []rune(l.ActionTaken)
	if len(runes) > 44 {
		actionLines = (len(runes)-1)/44 + 1
	}
	if actionLines > 1 {
		cellHeight = float64(actionLines) * 5
	}
	textY := yStart + 1

	pdf.SetFont("Helvetica", "", 8)
	// No
	pdf.SetXY(xStart, textY)
	pdf.CellFormat(colWidths[0], cellHeight-2, fmt.Sprintf("%d", no), "", 0, "C", false, 0, "")
	// Nama CCTV
	pdf.SetX(pdf.GetX() + colWidths[0])
	pdf.CellFormat(colWidths[1], cellHeight-2, pdfSafe(l.CCTV.Nama, 18), "", 0, "L", false, 0, "")
	// Kode CCTV
	pdf.SetX(pdf.GetX() + colWidths[1])
	pdf.CellFormat(colWidths[2], cellHeight-2, pdfSafe(l.CCTV.KodeCCTV, 12), "", 0, "L", false, 0, "")
	// Tindakan (multiline)
	xTindakan := xStart + colWidths[0] + colWidths[1] + colWidths[2]
	pdf.SetXY(xTindakan, textY)
	pdf.MultiCell(colWidths[3], 5, pdfSafe(l.ActionTaken, 44), "", "L", false)
	// Status
	xStatus := xStart + colWidths[0] + colWidths[1] + colWidths[2] + colWidths[3]
	pdf.SetXY(xStatus, textY)
	pdf.CellFormat(colWidths[4], cellHeight-2, pdfStatusLabel(l.Status), "", 0, "C", false, 0, "")
	// Teknisi
	xTeknisi := xStatus + colWidths[4]
	pdf.SetXY(xTeknisi, textY)
	pdf.CellFormat(colWidths[5], cellHeight-2, pdfSafe(l.User.FullName, 16), "", 0, "L", false, 0, "")
	// Tanggal
	xTanggal := xTeknisi + colWidths[5]
	pdf.SetXY(xTanggal, textY)
	dateStr := ""
	if l.CompletedAt != nil {
		dateStr = l.CompletedAt.Format("2006-01-02")
	} else if l.ScheduledDate != nil {
		dateStr = l.ScheduledDate.Format("2006-01-02")
	}
	pdf.CellFormat(colWidths[6], cellHeight-2, dateStr, "", 0, "C", false, 0, "")

	pdf.Line(xStart, yStart+cellHeight, xStart+pdfContentW, yStart+cellHeight)
	pdf.SetY(yStart + cellHeight)
}

func pdfSafe(s string, maxRunes int) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "—"
	}
	runes := []rune(s)
	if len(runes) > maxRunes {
		return string(runes[:maxRunes]) + "…"
	}
	return s
}

func pdfStatusLabel(status string) string {
	switch status {
	case "scheduled":
		return "Terjadwal"
	case "ongoing":
		return "Dalam proses"
	case statusSelesai:
		return "Selesai"
	case statusGagal:
		return "Gagal"
	default:
		return status
	}
}
