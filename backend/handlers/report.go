package handlers

import (
	"bytes"
	"fmt"
	"os"
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
	pdfBreakY     = 297.0 - 16.0 // batas auto page break (297 - bottom margin)
	statusSelesai = "completed"
	statusGagal   = "failed"

	// Kop surat
	kopNamaInstansi = "OTORITA IBU KOTA NUSANTARA"
	kopSubInstansi  = "Sistem Identifikasi Gangguan Aset Pusat (SIGAP)"
	kopAlamat1      = "Kompleks Perkantoran Pemerintah IKN, Kabupaten Penajam Paser Utara,"
	kopAlamat2      = "Provinsi Kalimantan Timur"
	kopTelepon      = ""          // isi jika diperlukan
	kopEmail        = ""          // isi jika diperlukan
	kopWebsite      = "www.ikn.go.id"
	kopLogoPath     = "assets/nusantara-logo.png"
	kopLogoWidth    = 15.3 // mm (691x996 px, rasio ~0.694)
	kopLogoHeight   = 22.0
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

	// Kop surat resmi
	drawLetterhead(pdf)

	// Judul laporan
	pdf.SetY(pdf.GetY() + 3)
	pdf.SetFont("Helvetica", "B", 14)
	pdf.SetTextColor(30, 30, 30)
	pdf.CellFormat(pdfContentW, 8, "LAPORAN PEMELIHARAAN CCTV", "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(110, 110, 110)
	pdf.CellFormat(pdfContentW, 5, "Periode: "+reportPeriod(c)+"   |   Dihasilkan: "+time.Now().Format("2006-01-02 15:04:05"), "", 1, "C", false, 0, "")
	pdf.SetTextColor(30, 30, 30)

	// Ringkasan
	summary := map[string]int{
		"Terjadwal":   0,
		"Dalam proses": 0,
		"Selesai":              0,
		"Perlu Tindak Lanjut": 0,
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
			summary["Perlu Tindak Lanjut"]++
		}
	}

	pdf.SetY(pdf.GetY() + 3)
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(pdfContentW, 6, fmt.Sprintf("Ringkasan: %d catatan pemeliharaan", len(logs)), "", 1, "", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(pdfContentW, 5, fmt.Sprintf(
		"Terjadwal: %d   |   Dalam proses: %d   |   Selesai: %d   |   Perlu Tindak Lanjut: %d",
		summary["Terjadwal"], summary["Dalam proses"], summary["Selesai"], summary["Perlu Tindak Lanjut"],
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

// drawLetterhead menggambar kop surat: logo IKN kiri, identitas instansi + alamat
// di tengah, diakhiri garis ganda (tebal + tipis).
func drawLetterhead(pdf *fpdf.Fpdf) {
	topY := 10.0
	leftX := pdfMarginX
	rightX := pdfMarginX + pdfContentW

	// Logo IKN di kiri (dilewati bila berkas tidak tersedia, mis. saat unit test)
	if _, err := os.Stat(kopLogoPath); err == nil {
		pdf.RegisterImage(kopLogoPath, "png")
		pdf.Image(kopLogoPath, leftX, topY, kopLogoWidth, kopLogoHeight, false, "", 0, "")
	}

	// Blok teks identitas di sebelah logo
	textX := leftX + kopLogoWidth + 5
	textW := rightX - textX

	pdf.SetFont("Helvetica", "B", 13)
	pdf.SetTextColor(30, 30, 30)
	pdf.SetXY(textX, topY+1.5)
	pdf.CellFormat(textW, 7, kopNamaInstansi, "", 1, "C", false, 0, "")

	pdf.SetFont("Helvetica", "", 8.5)
	pdf.SetTextColor(60, 60, 60)
	pdf.CellFormat(textW, 5, kopSubInstansi, "", 1, "C", false, 0, "")

	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(90, 90, 90)
	pdf.CellFormat(textW, 4.5, kopAlamat1, "", 1, "C", false, 0, "")
	pdf.CellFormat(textW, 4.5, kopAlamat2, "", 1, "C", false, 0, "")

	contact := "Website: " + kopWebsite
	if kopTelepon != "" {
		contact += "   |   Telp: " + kopTelepon
	}
	if kopEmail != "" {
		contact += "   |   Email: " + kopEmail
	}
	pdf.CellFormat(textW, 4.5, contact, "", 1, "C", false, 0, "")

	// Garis kop: tebal lalu tipis
	lineY := topY + kopLogoHeight + 2.5
	pdf.SetDrawColor(20, 20, 20)
	pdf.SetLineWidth(0.9)
	pdf.Line(leftX, lineY, rightX, lineY)
	pdf.SetLineWidth(0.25)
	pdf.Line(leftX, lineY+1.6, rightX, lineY+1.6)
	pdf.SetLineWidth(0.2)
	pdf.SetTextColor(30, 30, 30)
	pdf.SetY(lineY + 5)
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

	pdf.SetFont("Helvetica", "", 8)
	action := strings.TrimSpace(l.ActionTaken)
	if action == "" {
		action = "—"
	}

	// Hitung tinggi baris dari jumlah baris terlipat yang akurat (SplitLines)
	// agar sel tidak menimpa baris berikutnya dan tidak ada potongan di tengah sel.
	actionLines := pdf.SplitLines([]byte(action), colWidths[3])
	cellHeight := float64(len(actionLines))*5 + 3
	if cellHeight < 8 {
		cellHeight = 8
	}

	// Pindah halaman bila baris tidak muat; mencegah MultiCell terpotong tengah baris
	yStart := pdf.GetY()
	if yStart+cellHeight > pdfBreakY {
		pdf.AddPage()
		pdfTableHeader(pdf)
		yStart = pdf.GetY()
	}

	textY := yStart + 1.5

	// Border grid penuh agar tabel simetris
	pdf.SetDrawColor(205, 205, 205)
	x := xStart
	for _, w := range colWidths {
		pdf.Rect(x, yStart, w, cellHeight, "D")
		x += w
	}

	// No
	pdf.SetXY(xStart, textY)
	pdf.CellFormat(colWidths[0], 5, fmt.Sprintf("%d", no), "", 0, "C", false, 0, "")
	// Nama CCTV
	pdf.SetX(xStart + colWidths[0])
	pdf.CellFormat(colWidths[1], 5, pdfSafe(l.CCTV.Nama, 18), "", 0, "L", false, 0, "")
	// Kode CCTV
	pdf.SetX(xStart + colWidths[0] + colWidths[1])
	pdf.CellFormat(colWidths[2], 5, pdfSafe(l.CCTV.KodeCCTV, 12), "", 0, "L", false, 0, "")
	// Tindakan (multiline)
	pdf.SetXY(xStart+colWidths[0]+colWidths[1]+colWidths[2], textY)
	pdf.MultiCell(colWidths[3], 5, action, "", "L", false)
	// Status
	pdf.SetXY(xStart+colWidths[0]+colWidths[1]+colWidths[2]+colWidths[3], textY)
	pdf.CellFormat(colWidths[4], 5, pdfStatusLabel(l.Status), "", 0, "C", false, 0, "")
	// Teknisi
	pdf.SetXY(xStart+colWidths[0]+colWidths[1]+colWidths[2]+colWidths[3]+colWidths[4], textY)
	techName := ""
	if l.User != nil {
		techName = l.User.FullName
	}
	pdf.CellFormat(colWidths[5], 5, pdfSafe(techName, 16), "", 0, "L", false, 0, "")
	// Tanggal
	pdf.SetXY(xStart+colWidths[0]+colWidths[1]+colWidths[2]+colWidths[3]+colWidths[4]+colWidths[5], textY)
	dateStr := ""
	if l.CompletedAt != nil {
		dateStr = l.CompletedAt.Format("2006-01-02")
	} else if l.ScheduledDate != nil {
		dateStr = l.ScheduledDate.Format("2006-01-02")
	}
	pdf.CellFormat(colWidths[6], 5, dateStr, "", 0, "C", false, 0, "")

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
		return "Perlu Tindak Lanjut"
	default:
		return status
	}
}
