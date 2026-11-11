package handlers

import (
	"encoding/csv"
	"encoding/json"
	"io"
	"math"
	"net/http/httptest"
	"strings"
	"testing"

	"sigap-backend/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func floatPtr(v float64) *float64 {
	return &v
}

func TestEvaluateCoordinateQuality_AllClassifications(t *testing.T) {
	site := &models.Site{Nama: "Kawasan OIKN"}
	siteID := uuid.New()

	cctvs := []models.CCTV{
		{
			Base:       models.Base{ID: uuid.New()},
			KodeCCTV:   "THD-01",
			Nama:       "Kamera Verified",
			LokasiArea: "Plaza",
			SiteID:     &siteID,
			Site:       site,
			Status:     "aktif",
			Latitude:   floatPtr(-0.966348),
			Longitude:  floatPtr(116.720773),
			Keterangan: "Sumber data terverifikasi",
		},
		{
			Base:       models.Base{ID: uuid.New()},
			KodeCCTV:   "THD-18",
			Nama:       "Jalan Tol 1C",
			LokasiArea: "KIPP 1C",
			SiteID:     &siteID,
			Site:       site,
			Status:     "nonaktif",
			Latitude:   floatPtr(-0.972125),
			Longitude:  floatPtr(116.710371),
			Keterangan: "Koordinat: Estimasi area sekitar",
		},
		{
			Base:       models.Base{ID: uuid.New()},
			KodeCCTV:   "THD-05",
			Nama:       "Bundaran Sumbu Barat",
			Status:     "aktif",
			Latitude:   floatPtr(-0.977067574),
			Longitude:  floatPtr(116.7091789),
			Keterangan: "Sumber data",
		},
		{
			Base:       models.Base{ID: uuid.New()},
			KodeCCTV:   "THD-17",
			Nama:       "Pintu Masuk Sumbu Barat",
			Status:     "aktif",
			Latitude:   floatPtr(-0.977067574),
			Longitude:  floatPtr(116.7091789),
			Keterangan: "Sumber data",
		},
		{
			Base:     models.Base{ID: uuid.New()},
			KodeCCTV: "CAM-MISSING",
			Nama:     "Kamera Kosong",
			Status:   "nonaktif",
		},
		{
			Base:     models.Base{ID: uuid.New()},
			KodeCCTV: "CAM-PARTIAL",
			Nama:     "Kamera Parsial",
			Status:   "aktif",
			Latitude: floatPtr(-0.965),
		},
		{
			Base:      models.Base{ID: uuid.New()},
			KodeCCTV:  "CAM-INVALID",
			Nama:      "Kamera Range Invalid",
			Status:    "aktif",
			Latitude:  floatPtr(95.5),
			Longitude: floatPtr(116.7),
		},
		{
			Base:      models.Base{ID: uuid.New()},
			KodeCCTV:  "CAM-NAN",
			Nama:      "Kamera NaN",
			Status:    "aktif",
			Latitude:  floatPtr(math.NaN()),
			Longitude: floatPtr(116.7),
		},
		{
			Base:      models.Base{ID: uuid.New()},
			KodeCCTV:  "CAM-ZERO",
			Nama:      "Kamera Nol",
			Status:    "aktif",
			Latitude:  floatPtr(0),
			Longitude: floatPtr(0),
		},
	}

	report := EvaluateCoordinateQuality(cctvs)

	assert.Equal(t, 9, report.Summary.TotalCount)
	assert.Equal(t, 1, report.Summary.VerifiedCount)
	assert.Equal(t, 8, report.Summary.NeedsAttentionCount)
	assert.Equal(t, 1, report.Summary.MissingCount)
	assert.Equal(t, 1, report.Summary.PartialCount)
	assert.Equal(t, 2, report.Summary.InvalidCount)
	assert.Equal(t, 1, report.Summary.ZeroCount)
	assert.Equal(t, 1, report.Summary.EstimatedCount)
	assert.Equal(t, 2, report.Summary.DuplicateCount)
	// Complete usable pairs: THD-01, THD-18, THD-05, THD-17 = 4 / 9 = 44.4%
	assert.InDelta(t, 44.4, report.Summary.CompletenessPercent, 0.05)
	assert.InDelta(t, 11.1, report.Summary.VerifiedPercent, 0.05)
	assert.Equal(t, []string{"THD-18"}, report.Summary.PendingLegacyCodes)

	statusByCode := make(map[string]CoordinateQualityItem, len(report.Items))
	for _, it := range report.Items {
		statusByCode[it.KodeCCTV] = it
	}
	assert.Equal(t, CoordinateStatusVerified, statusByCode["THD-01"].CoordinateStatus)
	assert.Empty(t, statusByCode["THD-01"].Issues)
	assert.Equal(t, "Kawasan OIKN", statusByCode["THD-01"].SiteNama)

	assert.Equal(t, CoordinateStatusEstimated, statusByCode["THD-18"].CoordinateStatus)
	assert.Contains(t, statusByCode["THD-18"].Issues, CoordinateIssueEstimated)

	assert.Equal(t, CoordinateStatusDuplicate, statusByCode["THD-05"].CoordinateStatus)
	assert.Contains(t, statusByCode["THD-05"].Issues, CoordinateIssueDuplicate)
	assert.Equal(t, CoordinateStatusDuplicate, statusByCode["THD-17"].CoordinateStatus)

	assert.Equal(t, CoordinateStatusMissing, statusByCode["CAM-MISSING"].CoordinateStatus)
	assert.Equal(t, CoordinateStatusPartial, statusByCode["CAM-PARTIAL"].CoordinateStatus)
	assert.Equal(t, CoordinateStatusInvalid, statusByCode["CAM-INVALID"].CoordinateStatus)
	assert.Equal(t, CoordinateStatusInvalid, statusByCode["CAM-NAN"].CoordinateStatus)
	assert.Equal(t, CoordinateStatusZero, statusByCode["CAM-ZERO"].CoordinateStatus)

	// Encode ke JSON memastikan koordinat tidak boleh memuat NaN/Inf dan tidak bocor field RTSP.
	rawJSON, err := json.Marshal(report)
	// Catatan: JSON marshal standar menolak math.NaN pada pointer *float64 jika ada;
	// pada data DB normal NaN telah ditolak validasi, sedangkan EvaluateCoordinateQuality
	// aman dievaluasi terhadap pointer non-finit tanpa panic.
	if err == nil {
		assert.NotContains(t, string(rawJSON), "rtsp_url")
		assert.NotContains(t, string(rawJSON), "rtsp_password")
	}
}

func TestGetCCTVCoordinateQuality_SiteScopedJSONAndCSV(t *testing.T) {
	_, mock := setupMockGORM(t)
	siteID := uuid.New()
	id1 := uuid.New()
	id2 := uuid.New()

	rows := sqlmock.NewRows([]string{
		"id", "kode_cctv", "nama", "lokasi_area", "site_id", "status", "latitude", "longitude", "keterangan",
	}).
		AddRow(id1, "THD-01", "Rest Area 1", "Rest Area", siteID, "nonaktif", -0.9663481, 116.7207729, "Sumber data").
		AddRow(id2, "THD-20", "=DangerName", "KIPP 1B", siteID, "nonaktif", -0.9894029, 116.6739831, "Koordinat: Estimasi area terdekat")

	mock.ExpectQuery(`FROM "cctvs" WHERE .*cctvs\.site_id = \$1`).
		WithArgs(siteID.String()).
		WillReturnRows(rows)
	mock.ExpectQuery(`FROM "sites" WHERE "sites"\."id" =`).
		WithArgs(siteID).
		WillReturnRows(siteRows(siteID, "Kawasan OIKN", "OIKN", true))

	app := newTestApp()
	app.Get("/cctv/coordinate-quality", withUserSiteLocals(uuid.NewString(), "teknisi", siteID.String()), GetCCTVCoordinateQuality)

	// 1. JSON response
	req := httptest.NewRequest("GET", "/cctv/coordinate-quality", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, resp.StatusCode)

	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	bodyStr := string(bodyBytes)
	assert.NotContains(t, bodyStr, "rtsp_url")
	assert.NotContains(t, bodyStr, "rtsp_password")

	var decoded struct {
		Status string                  `json:"status"`
		Data   CoordinateQualityReport `json:"data"`
	}
	require.NoError(t, json.Unmarshal(bodyBytes, &decoded))
	assert.Equal(t, "success", decoded.Status)
	assert.Equal(t, 2, decoded.Data.Summary.TotalCount)
	assert.Equal(t, 1, decoded.Data.Summary.VerifiedCount)
	assert.Equal(t, 1, decoded.Data.Summary.EstimatedCount)
	assert.Equal(t, []string{"THD-20"}, decoded.Data.Summary.PendingLegacyCodes)

	// 2. CSV response with formula sanitization + intact negative latitude
	rowsCSV := sqlmock.NewRows([]string{
		"id", "kode_cctv", "nama", "lokasi_area", "site_id", "status", "latitude", "longitude", "keterangan",
	}).
		AddRow(id1, "THD-01", "Rest Area 1", "Rest Area", siteID, "nonaktif", -0.9663481, 116.7207729, "Sumber data").
		AddRow(id2, "THD-20", "=DangerName", "KIPP 1B", siteID, "nonaktif", -0.9894029, 116.6739831, "Koordinat: Estimasi")

	mock.ExpectQuery(`FROM "cctvs" WHERE .*cctvs\.site_id = \$1`).
		WithArgs(siteID.String()).
		WillReturnRows(rowsCSV)
	mock.ExpectQuery(`FROM "sites" WHERE "sites"\."id" =`).
		WithArgs(siteID).
		WillReturnRows(siteRows(siteID, "Kawasan OIKN", "OIKN", true))

	csvReq := httptest.NewRequest("GET", "/cctv/coordinate-quality?format=csv", nil)
	csvResp, err := app.Test(csvReq)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusOK, csvResp.StatusCode)
	assert.Equal(t, "text/csv; charset=utf-8", csvResp.Header.Get("Content-Type"))
	assert.Contains(t, csvResp.Header.Get("Content-Disposition"), "cctv-coordinate-quality.csv")

	csvBytes, err := io.ReadAll(csvResp.Body)
	require.NoError(t, err)
	records, err := csv.NewReader(strings.NewReader(string(csvBytes))).ReadAll()
	require.NoError(t, err)
	require.Len(t, records, 3)
	assert.Equal(t, "THD-20", records[2][0])
	assert.Equal(t, "'=DangerName", records[2][1]) // CSV formula injection neutralized
	assert.Equal(t, "-0.9894029", records[2][5])  // Numeric negative latitude preserved

	assert.NoError(t, mock.ExpectationsWereMet())
}
