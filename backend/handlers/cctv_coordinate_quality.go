package handlers

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"sigap-backend/config"
	"sigap-backend/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// Issue codes & status kualitas koordinat CCTV (#151).
const (
	CoordinateStatusVerified  = "verified"
	CoordinateStatusEstimated = "estimated"
	CoordinateStatusMissing   = "missing"
	CoordinateStatusPartial   = "partial"
	CoordinateStatusInvalid   = "invalid"
	CoordinateStatusZero      = "zero"
	CoordinateStatusDuplicate = "duplicate"

	CoordinateIssueMissing   = "missing_coordinate"
	CoordinateIssuePartial   = "partial_coordinate"
	CoordinateIssueInvalid   = "invalid_coordinate"
	CoordinateIssueZero      = "zero_coordinate"
	CoordinateIssueEstimated = "estimated_coordinate"
	CoordinateIssueDuplicate = "duplicate_coordinate"
)

// pendingSurveyLegacyTHDCodes memuat daftar 9 kode CCTV THD legacy yang
// tercatat memakai koordinat estimasi pada arsip inventaris 2026-10-20 (#151).
var pendingSurveyLegacyTHDCodes = []string{
	"THD-18",
	"THD-20",
	"THD-25",
	"THD-27",
	"THD-29",
	"THD-30",
	"THD-31",
	"THD-34",
	"THD-35",
}

var pendingSurveyLegacyTHDSet = func() map[string]struct{} {
	set := make(map[string]struct{}, len(pendingSurveyLegacyTHDCodes))
	for _, code := range pendingSurveyLegacyTHDCodes {
		set[code] = struct{}{}
	}
	return set
}()

// CoordinateQualityItem merepresentasikan ringkasan kualitas koordinat satu
// unit CCTV tanpa mengekspos kredensial RTSP ataupun payload QR.
type CoordinateQualityItem struct {
	ID               uuid.UUID  `json:"id"`
	KodeCCTV         string     `json:"kode_cctv"`
	Nama             string     `json:"nama"`
	LokasiArea       string     `json:"lokasi_area"`
	SiteID           *uuid.UUID `json:"site_id,omitempty"`
	SiteNama         string     `json:"site_nama,omitempty"`
	Status           string     `json:"status"`
	Latitude         *float64   `json:"latitude"`
	Longitude        *float64   `json:"longitude"`
	Keterangan       string     `json:"keterangan"`
	CoordinateStatus string     `json:"coordinate_status"`
	Issues           []string   `json:"issues"`
}

// CoordinateQualitySummary merangkum metrik kelengkapan dan anomali koordinat.
type CoordinateQualitySummary struct {
	TotalCount          int      `json:"total_count"`
	VerifiedCount       int      `json:"verified_count"`
	NeedsAttentionCount int      `json:"needs_attention_count"`
	MissingCount        int      `json:"missing_count"`
	PartialCount        int      `json:"partial_count"`
	InvalidCount        int      `json:"invalid_count"`
	ZeroCount           int      `json:"zero_count"`
	EstimatedCount      int      `json:"estimated_count"`
	DuplicateCount      int      `json:"duplicate_count"`
	CompletenessPercent float64  `json:"completeness_percent"`
	VerifiedPercent     float64  `json:"verified_percent"`
	PendingLegacyCodes  []string `json:"pending_legacy_codes"`
}

// CoordinateQualityReport adalah payload JSON respons GET /api/cctv/coordinate-quality.
type CoordinateQualityReport struct {
	Summary CoordinateQualitySummary `json:"summary"`
	Items   []CoordinateQualityItem  `json:"items"`
}

func isValidCoordinatePair(lat, lng float64) bool {
	if math.IsNaN(lat) || math.IsInf(lat, 0) || math.IsNaN(lng) || math.IsInf(lng, 0) {
		return false
	}
	return lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180
}

func isZeroCoordinate(lat, lng float64) bool {
	return math.Abs(lat) < 1e-9 && math.Abs(lng) < 1e-9
}

func hasEstimatedMarker(c models.CCTV) bool {
	haystack := strings.ToLower(strings.TrimSpace(c.Keterangan + " " + c.LokasiArea + " " + c.Nama))
	if haystack == "" {
		return false
	}
	return strings.Contains(haystack, "estimasi") ||
		strings.Contains(haystack, "perkiraan") ||
		strings.Contains(haystack, "belum terverifikasi")
}

func coordinatePairKey(lat, lng float64) string {
	return fmt.Sprintf("%.6f,%.6f", lat, lng)
}

func sanitizeCSVCell(value string) string {
	if value == "" {
		return ""
	}
	trimmedLeft := strings.TrimLeft(value, " ")
	if trimmedLeft != "" {
		switch trimmedLeft[0] {
		case '=', '+', '-', '@', '\t', '\r':
			return "'" + value
		}
	}
	return value
}

// EvaluateCoordinateQuality memeriksa kualitas koordinat seluruh record CCTV
// secara deterministik: koordinat kosong, parsial, invalid, nol (0,0),
// bertanda estimasi/belum terverifikasi, serta koordinat duplikat (presisi
// 6 desimal). Fungsi ini murni (tanpa I/O) agar mudah diuji unit.
func EvaluateCoordinateQuality(cctvs []models.CCTV) CoordinateQualityReport {
	coordCounts := make(map[string]int, len(cctvs))
	for _, c := range cctvs {
		if c.Latitude == nil || c.Longitude == nil {
			continue
		}
		lat, lng := *c.Latitude, *c.Longitude
		if !isValidCoordinatePair(lat, lng) || isZeroCoordinate(lat, lng) {
			continue
		}
		coordCounts[coordinatePairKey(lat, lng)]++
	}

	items := make([]CoordinateQualityItem, 0, len(cctvs))
	summary := CoordinateQualitySummary{
		TotalCount:         len(cctvs),
		PendingLegacyCodes: make([]string, 0),
	}

	completeCoordCount := 0
	pendingLegacySeen := make(map[string]struct{})

	for _, c := range cctvs {
		issues := make([]string, 0, 3)
		hasUsableCoord := false

		switch {
		case c.Latitude == nil && c.Longitude == nil:
			issues = append(issues, CoordinateIssueMissing)
			summary.MissingCount++
		case (c.Latitude == nil) != (c.Longitude == nil):
			issues = append(issues, CoordinateIssuePartial)
			summary.PartialCount++
		default:
			lat, lng := *c.Latitude, *c.Longitude
			if !isValidCoordinatePair(lat, lng) {
				issues = append(issues, CoordinateIssueInvalid)
				summary.InvalidCount++
			} else if isZeroCoordinate(lat, lng) {
				issues = append(issues, CoordinateIssueZero)
				summary.ZeroCount++
			} else {
				hasUsableCoord = true
				completeCoordCount++
				if coordCounts[coordinatePairKey(lat, lng)] > 1 {
					issues = append(issues, CoordinateIssueDuplicate)
					summary.DuplicateCount++
				}
			}
		}

		if hasEstimatedMarker(c) {
			issues = append(issues, CoordinateIssueEstimated)
			summary.EstimatedCount++
		}

		status := CoordinateStatusVerified
		switch {
		case containsIssue(issues, CoordinateIssueMissing):
			status = CoordinateStatusMissing
		case containsIssue(issues, CoordinateIssuePartial):
			status = CoordinateStatusPartial
		case containsIssue(issues, CoordinateIssueInvalid):
			status = CoordinateStatusInvalid
		case containsIssue(issues, CoordinateIssueZero):
			status = CoordinateStatusZero
		case containsIssue(issues, CoordinateIssueEstimated):
			status = CoordinateStatusEstimated
		case containsIssue(issues, CoordinateIssueDuplicate):
			status = CoordinateStatusDuplicate
		}

		if len(issues) == 0 && hasUsableCoord {
			summary.VerifiedCount++
		} else {
			summary.NeedsAttentionCount++
			normalizedCode := strings.ToUpper(strings.TrimSpace(c.KodeCCTV))
			if _, ok := pendingSurveyLegacyTHDSet[normalizedCode]; ok {
				pendingLegacySeen[normalizedCode] = struct{}{}
			}
		}

		siteNama := ""
		if c.Site != nil {
			siteNama = c.Site.Nama
		}

		items = append(items, CoordinateQualityItem{
			ID:               c.ID,
			KodeCCTV:         c.KodeCCTV,
			Nama:             c.Nama,
			LokasiArea:       c.LokasiArea,
			SiteID:           c.SiteID,
			SiteNama:         siteNama,
			Status:           c.Status,
			Latitude:         c.Latitude,
			Longitude:        c.Longitude,
			Keterangan:       c.Keterangan,
			CoordinateStatus: status,
			Issues:           issues,
		})
	}

	if summary.TotalCount > 0 {
		summary.CompletenessPercent = math.Round((float64(completeCoordCount)/float64(summary.TotalCount))*1000) / 10
		summary.VerifiedPercent = math.Round((float64(summary.VerifiedCount)/float64(summary.TotalCount))*1000) / 10
	}

	if len(pendingLegacySeen) > 0 {
		for _, code := range pendingSurveyLegacyTHDCodes {
			if _, ok := pendingLegacySeen[code]; ok {
				summary.PendingLegacyCodes = append(summary.PendingLegacyCodes, code)
			}
		}
		sort.Strings(summary.PendingLegacyCodes)
	}

	return CoordinateQualityReport{
		Summary: summary,
		Items:   items,
	}
}

func containsIssue(issues []string, target string) bool {
	for _, issue := range issues {
		if issue == target {
			return true
		}
	}
	return false
}

func filterCoordinateQualityItems(items []CoordinateQualityItem, filter string) []CoordinateQualityItem {
	switch strings.ToLower(strings.TrimSpace(filter)) {
	case "", "all":
		return items
	case "needs_attention", "attention":
		out := make([]CoordinateQualityItem, 0)
		for _, item := range items {
			if len(item.Issues) > 0 {
				out = append(out, item)
			}
		}
		return out
	case CoordinateStatusVerified:
		out := make([]CoordinateQualityItem, 0)
		for _, item := range items {
			if len(item.Issues) == 0 {
				out = append(out, item)
			}
		}
		return out
	default:
		out := make([]CoordinateQualityItem, 0)
		for _, item := range items {
			if item.CoordinateStatus == filter || containsIssue(item.Issues, filter) {
				out = append(out, item)
			}
		}
		return out
	}
}

func formatCoordinateForCSV(val *float64) string {
	if val == nil {
		return ""
	}
	if math.IsNaN(*val) || math.IsInf(*val, 0) {
		return ""
	}
	return strconv.FormatFloat(*val, 'f', -1, 64)
}

// GetCCTVCoordinateQuality menghasilkan laporan audit kualitas koordinat CCTV
// (kosong, parsial, invalid, 0/0, bertanda Estimasi, serta duplikat) yang
// terisolasi per site bila konteks site aktif (#151).
// @Summary      Audit Kualitas Koordinat CCTV
// @Description  Mengevaluasi kelengkapan, tanda estimasi, rentang valid WGS84, dan duplikasi koordinat CCTV pada scope site aktif. Mendukung ekspor CSV lewat ?format=csv.
// @Tags         Master CCTV
// @Produce      json,text/csv
// @Security     BearerAuth
// @Param        X-Site-ID query string false "Opsional override UUID site"
// @Param        issue query string false "Filter item berdasarkan status/issue (all, needs_attention, estimated, duplicate, missing, invalid)"
// @Param        format query string false "Format output: json (default) atau csv"
// @Success      200 {object} map[string]interface{} "Ringkasan dan rincian kualitas koordinat CCTV"
// @Failure      401 {object} map[string]interface{} "Sesi tidak valid"
// @Failure      403 {object} map[string]interface{} "Akses ditolak untuk site terkait"
// @Failure      500 {object} map[string]interface{} "Gagal membaca data CCTV"
// @Router       /cctv/coordinate-quality [get]
func GetCCTVCoordinateQuality(c *fiber.Ctx) error {
	var cctvs []models.CCTV
	query := config.DB.Model(&models.CCTV{}).
		Select("cctvs.id, cctvs.kode_cctv, cctvs.nama, cctvs.lokasi_area, cctvs.site_id, cctvs.status, cctvs.latitude, cctvs.longitude, cctvs.keterangan").
		Preload("Site")

	if siteID, ok := c.Locals("site_id").(string); ok && siteID != "" {
		query = query.Where("cctvs.site_id = ?", siteID)
	}

	if err := query.Order("cctvs.kode_cctv ASC, cctvs.nama ASC").Find(&cctvs).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Gagal mengevaluasi kualitas koordinat CCTV",
		})
	}

	report := EvaluateCoordinateQuality(cctvs)
	issueFilter := strings.TrimSpace(c.Query("issue"))
	if issueFilter != "" {
		report.Items = filterCoordinateQualityItems(report.Items, issueFilter)
	}

	if strings.EqualFold(strings.TrimSpace(c.Query("format")), "csv") {
		var buf bytes.Buffer
		writer := csv.NewWriter(&buf)
		header := []string{
			"Kode CCTV",
			"Nama",
			"Site",
			"Lokasi Area",
			"Status Perangkat",
			"Latitude",
			"Longitude",
			"Status Koordinat",
			"Masalah",
			"Keterangan",
		}
		if err := writer.Write(header); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "Gagal menulis header CSV kualitas koordinat",
			})
		}

		for _, item := range report.Items {
			row := []string{
				sanitizeCSVCell(item.KodeCCTV),
				sanitizeCSVCell(item.Nama),
				sanitizeCSVCell(item.SiteNama),
				sanitizeCSVCell(item.LokasiArea),
				sanitizeCSVCell(item.Status),
				formatCoordinateForCSV(item.Latitude),
				formatCoordinateForCSV(item.Longitude),
				sanitizeCSVCell(item.CoordinateStatus),
				sanitizeCSVCell(strings.Join(item.Issues, "; ")),
				sanitizeCSVCell(item.Keterangan),
			}
			if err := writer.Write(row); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"status":  "error",
					"message": "Gagal menulis baris CSV kualitas koordinat",
				})
			}
		}
		writer.Flush()
		if err := writer.Error(); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"status":  "error",
				"message": "Gagal menyelesaikan file CSV kualitas koordinat",
			})
		}

		c.Set("Content-Type", "text/csv; charset=utf-8")
		c.Set("Content-Disposition", "attachment; filename=cctv-coordinate-quality.csv")
		return c.Send(buf.Bytes())
	}

	return c.JSON(fiber.Map{
		"status": "success",
		"data":   report,
	})
}
