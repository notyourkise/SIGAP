package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"sigap-backend/aiassistant"
)

var witaZoneSummary = time.FixedZone("WITA", 8*3600)

func isOperationalSummaryQuestion(question string) bool {
	q := strings.ToLower(strings.TrimSpace(question))
	for _, kw := range []string{
		"7 hari",
		"tujuh hari",
		"seminggu",
		"minggu ini",
		"hari terakhir",
		"cctv mati",
		"kamera mati",
		"cctv offline",
		"kamera offline",
		"sering mati",
		"sering offline",
		"gangguan cctv",
		"kamera bermasalah",
		"rangkum",
		"ringkasan",
		"rekap",
		"statistik",
		"laporan cctv",
		"penyebab",
		"kendala",
		"tiket aktif",
		"insiden terkini",
		"daftar tiket",
		"thd-",
	} {
		if strings.Contains(q, kw) {
			return true
		}
	}
	return false
}

func buildSevenDaySummaryDocument(ctx context.Context, db *gorm.DB, siteID uuid.UUID, now time.Time) (aiassistant.Document, bool, error) {
	if db == nil {
		return aiassistant.Document{}, false, nil
	}

	since := now.UTC().Add(-7 * 24 * time.Hour)

	type incidentTotals struct {
		Total       int64
		Active      int64
		PendingVer  int64
		Resolved    int64
	}
	var totals incidentTotals
	qTotals := db.WithContext(ctx).Table("cctv_incidents").Where("opened_at >= ?", since)
	if siteID != uuid.Nil {
		qTotals = qTotals.Where("site_id = ?", siteID)
	}
	if err := qTotals.Select(
		"count(*) AS total, " +
			"count(*) FILTER (WHERE status IN ('open','in_progress')) AS active, " +
			"count(*) FILTER (WHERE status = 'pending_verification') AS pending_ver, " +
			"count(*) FILTER (WHERE status IN ('resolved','merged')) AS resolved",
	).Scan(&totals).Error; err != nil {
		return aiassistant.Document{}, false, err
	}

	type cameraRow struct {
		CCTVID          uuid.UUID
		KodeCCTV        string
		Nama            string
		TipeKoneksi     string
		SumberListrik   string
		TicketCount     int64
		OfflineEvents   int64
		LatestStatus    string
		LatestTicketNo  string
		LatestFindings  string
	}
	var cameras []cameraRow
	qCam := db.WithContext(ctx).Table("cctv_incident_cameras m").
		Joins("JOIN cctv_incidents i ON i.id = m.incident_id").
		Joins("JOIN cctvs c ON c.id = m.cctv_id").
		Where("i.opened_at >= ?", since)
	if siteID != uuid.Nil {
		qCam = qCam.Where("i.site_id = ?", siteID)
	}
	if err := qCam.Select(`
		c.id AS cctv_id,
		c.kode_cctv,
		c.nama,
		coalesce(c.tipe_koneksi, '-') AS tipe_koneksi,
		coalesce(c.sumber_listrik, '-') AS sumber_listrik,
		count(DISTINCT i.id) AS ticket_count,
		coalesce((SELECT count(*) FROM cctv_incident_events e WHERE e.cctv_id = c.id AND e.event_type IN ('MEMBER_OFFLINE_AGAIN','INCIDENT_OPENED') AND e.created_at >= ?), 0) AS offline_events,
		(ARRAY_AGG(i.status ORDER BY i.opened_at DESC))[1] AS latest_status,
		(ARRAY_AGG(i.ticket_number ORDER BY i.opened_at DESC))[1] AS latest_ticket_no,
		coalesce(nullif(trim((ARRAY_AGG(m.damage_findings ORDER BY i.opened_at DESC))[1]), ''), '') AS latest_findings
	`, since).
		Group("c.id, c.kode_cctv, c.nama, c.tipe_koneksi, c.sumber_listrik").
		Order("offline_events DESC, ticket_count DESC, c.kode_cctv ASC").
		Limit(15).
		Scan(&cameras).Error; err != nil {
		return aiassistant.Document{}, false, err
	}

	type camReasonRow struct {
		CCTVID uuid.UUID
		Detail string
		Cnt    int64
	}
	var camReasons []camReasonRow
	if len(cameras) > 0 {
		ids := make([]uuid.UUID, 0, len(cameras))
		for _, cm := range cameras {
			ids = append(ids, cm.CCTVID)
		}
		_ = db.WithContext(ctx).Table("cctv_stream_logs").
			Select("cctv_id, coalesce(nullif(trim(error_detail), ''), nullif(trim(message), ''), 'gangguan stream') AS detail, count(*) AS cnt").
			Where("created_at >= ? AND cctv_id IN ? AND current_state = 'offline'", since, ids).
			Group("cctv_id, detail").
			Order("cctv_id, cnt DESC").
			Scan(&camReasons).Error
	}
	reasonByCam := map[uuid.UUID][]string{}
	for _, r := range camReasons {
		if len(reasonByCam[r.CCTVID]) >= 2 {
			continue
		}
		reasonByCam[r.CCTVID] = append(reasonByCam[r.CCTVID], fmt.Sprintf("%s (%dx)", r.Detail, r.Cnt))
	}

	type topCauseRow struct {
		Detail      string
		EventCount  int64
		CameraCount int64
	}
	var topCauses []topCauseRow
	qCauses := db.WithContext(ctx).Table("cctv_stream_logs l").
		Joins("JOIN cctvs c ON c.id = l.cctv_id").
		Where("l.created_at >= ? AND l.current_state = 'offline' AND c.deleted_at IS NULL", since)
	if siteID != uuid.Nil {
		qCauses = qCauses.Where("c.site_id = ?", siteID)
	}
	_ = qCauses.Select("coalesce(nullif(trim(l.error_detail), ''), nullif(trim(l.message), ''), 'gangguan stream') AS detail, count(*) AS event_count, count(DISTINCT l.cctv_id) AS camera_count").
		Group("detail").
		Order("event_count DESC").
		Limit(5).
		Scan(&topCauses).Error

	var lines []string
	lines = append(lines, "Ringkasan Operasional CCTV Mati dan Insiden 7 Hari Terakhir beserta Penyebab (Data Live Database SIGAP)")
	lines = append(lines, fmt.Sprintf("Periode data: %s s.d. %s WITA",
		since.In(witaZoneSummary).Format("02 Jan 2006 15:04"),
		now.In(witaZoneSummary).Format("02 Jan 2006 15:04")))
	lines = append(lines, fmt.Sprintf("Total tiket insiden 7 hari terakhir: %d tiket (aktif/penanganan: %d, siap verifikasi: %d, selesai/digabung: %d).",
		totals.Total, totals.Active, totals.PendingVer, totals.Resolved))

	if len(cameras) == 0 {
		lines = append(lines, "Tidak ada kamera CCTV yang tercatat mati atau memiliki tiket insiden dalam 7 hari terakhir.")
	} else {
		lines = append(lines, fmt.Sprintf("Daftar kamera yang sempat mati atau terdampak insiden dalam 7 hari terakhir (%d kamera teratas):", len(cameras)))
		for _, cm := range cameras {
			causeText := "belum ada detail error spesifik di log stream"
			if rs := reasonByCam[cm.CCTVID]; len(rs) > 0 {
				causeText = strings.Join(rs, "; ")
			}
			if cm.LatestFindings != "" {
				causeText += "; temuan teknisi: " + cm.LatestFindings
			}
			lines = append(lines, fmt.Sprintf("- %s (%s, %s/%s): %d tiket (%s %s), %d kejadian offline berulang; penyebab: %s.",
				cm.KodeCCTV, cm.Nama, cm.TipeKoneksi, cm.SumberListrik,
				cm.TicketCount, cm.LatestTicketNo, cm.LatestStatus, cm.OfflineEvents, causeText))
		}
	}

	if len(topCauses) > 0 {
		lines = append(lines, "Penyebab terbanyak kamera offline 7 hari terakhir:")
		for _, tc := range topCauses {
			lines = append(lines, fmt.Sprintf("- %s: %d kejadian pada %d kamera.", tc.Detail, tc.EventCount, tc.CameraCount))
		}
	}

	return aiassistant.Document{
		ID:   "live-db://ringkasan-7-hari",
		Text: strings.Join(lines, "\n"),
	}, true, nil
}
