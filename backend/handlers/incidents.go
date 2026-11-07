package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"sigap-backend/config"
	"sigap-backend/incidents"
	"sigap-backend/utils"
)

func IncidentMonitoringEnabled() bool { return os.Getenv("CCTV_INCIDENTS_ENABLED") == "true" }

func incidentSite(c *fiber.Ctx) (uuid.UUID, bool) {
	value, _ := c.Locals("site_id").(string)
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		_ = c.Status(400).JSON(fiber.Map{"status": "error", "message": "Pilih site untuk mengelola insiden CCTV"})
		return uuid.Nil, false
	}
	return id, true
}

func incidentAdminID(c *fiber.Ctx) (uuid.UUID, bool) {
	user, _ := c.Locals("user_id").(string)
	admin, err := uuid.Parse(user)
	if err != nil || admin == uuid.Nil {
		_ = c.SendStatus(fiber.StatusUnauthorized)
		return uuid.Nil, false
	}
	return admin, true
}

type ProbeTelemetrySummary struct {
	WindowHours              int   `json:"window_hours"`
	GSMProbeCount            int64 `json:"gsm_probe_count"`
	GSMAvgBytesSession       int64 `json:"gsm_avg_bytes_session"`
	GSMMaxBytesSession       int64 `json:"gsm_max_bytes_session"`
	GSMTotalBytesObserved    int64 `json:"gsm_total_bytes_observed"`
	GSMAvgDurationMs         int64 `json:"gsm_avg_duration_ms"`
	GSMForcedCloseCount      int64 `json:"gsm_forced_close_count"`
	GSMUnconfirmedCloseCount int64 `json:"gsm_unconfirmed_close_count"`
	MetroProbeCount          int64 `json:"metro_probe_count"`
}

func buildProbeTelemetrySummary(db *gorm.DB, site uuid.UUID) ProbeTelemetrySummary {
	summary := ProbeTelemetrySummary{WindowHours: 24}
	if db == nil {
		return summary
	}
	since := time.Now().UTC().Add(-24 * time.Hour)
	type agg struct {
		Cnt            int64
		AvgBytes       int64
		MaxBytes       int64
		TotalBytes     int64
		AvgDuration    int64
		ForcedCnt      int64
		UnconfirmedCnt int64
	}
	var gsmAgg agg
	_ = db.Table("cctv_probe_telemetry").
		Select(`
			COUNT(*) AS cnt,
			COALESCE(AVG(bytes_total_session), 0)::bigint AS avg_bytes,
			COALESCE(MAX(bytes_total_session), 0)::bigint AS max_bytes,
			COALESCE(SUM(bytes_total_session), 0)::bigint AS total_bytes,
			COALESCE(AVG(verify_duration_ms), 0)::bigint AS avg_duration,
			COALESCE(SUM(CASE WHEN forced_close THEN 1 ELSE 0 END), 0)::bigint AS forced_cnt,
			COALESCE(SUM(CASE WHEN NOT source_closed_confirmed THEN 1 ELSE 0 END), 0)::bigint AS unconfirmed_cnt
		`).
		Where("site_id = ? AND network_type = 'gsm' AND created_at >= ?", site, since).
		Scan(&gsmAgg).Error
	summary.GSMProbeCount = gsmAgg.Cnt
	summary.GSMAvgBytesSession = gsmAgg.AvgBytes
	summary.GSMMaxBytesSession = gsmAgg.MaxBytes
	summary.GSMTotalBytesObserved = gsmAgg.TotalBytes
	summary.GSMAvgDurationMs = gsmAgg.AvgDuration
	summary.GSMForcedCloseCount = gsmAgg.ForcedCnt
	summary.GSMUnconfirmedCloseCount = gsmAgg.UnconfirmedCnt

	_ = db.Model(&incidents.ProbeTelemetry{}).
		Where("site_id = ? AND network_type <> 'gsm' AND created_at >= ?", site, since).
		Count(&summary.MetroProbeCount).Error
	return summary
}

func GetIncidents(c *fiber.Ctx) error {
	site, ok := incidentSite(c)
	if !ok {
		return nil
	}
	var items []incidents.Incident
	if err := config.DB.
		Preload("Members", func(db *gorm.DB) *gorm.DB { return db.Order("camera_name ASC") }).
		Preload("Events", func(db *gorm.DB) *gorm.DB { return db.Order("created_at DESC") }).
		Where("site_id = ?", site).
		Order("opened_at DESC").
		Limit(100).
		Find(&items).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Tidak dapat memuat tiket insiden"})
	}

	var jobs []incidents.Outbox
	if err := config.DB.Where("site_id = ?", site).Order("created_at DESC").Limit(80).Find(&jobs).Error; err != nil {
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Tidak dapat memuat status pemberitahuan"})
	}

	svc := incidents.Service{DB: config.DB}
	plannedOutages, _ := svc.ListPlannedOutages(c.UserContext(), site)

	var episodes []incidents.MonitorHealthEpisode
	_ = config.DB.Where("site_id = ?", site).Order("started_at DESC").Limit(20).Find(&episodes).Error

	outcomeSummary, _ := svc.BuildCameraOutcomeReport(c.UserContext(), site)
	telemetrySummary := buildProbeTelemetrySummary(config.DB, site)

	return c.JSON(fiber.Map{
		"status": "success",
		"data": fiber.Map{
			"incidents":             items,
			"notifications":         jobs,
			"planned_outages":       plannedOutages,
			"monitor_episodes":      episodes,
			"outcome_summary":       outcomeSummary,
			"gsm_telemetry_summary": telemetrySummary,
			"monitoring_enabled":    IncidentMonitoringEnabled(),
		},
	})
}

func AssignIncident(c *fiber.Ctx) error {
	site, ok := incidentSite(c)
	if !ok {
		return nil
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "ID insiden tidak valid"})
	}
	var req struct {
		TechnicianID string `json:"technician_id"`
	}
	if c.BodyParser(&req) != nil {
		return c.SendStatus(400)
	}
	tech, err := uuid.Parse(req.TechnicianID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "ID teknisi tidak valid"})
	}
	if err := (incidents.Service{DB: config.DB}).Assign(c.UserContext(), id, site, tech, time.Now().UTC()); err != nil {
		return incidentActionError(c, err)
	}
	utils.LogAudit(config.DB, c, "ASSIGN_INCIDENT", "INCIDENT", "Menugaskan teknisi pada insiden CCTV "+id.String(), nil, "", "")
	return c.JSON(fiber.Map{"status": "success", "message": "Teknisi ditugaskan; tugas tersedia di menu pemeliharaan"})
}

func CreateIncidentCameraFieldTask(c *fiber.Ctx) error {
	site, ok := incidentSite(c)
	if !ok {
		return nil
	}
	incidentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "ID insiden tidak valid"})
	}
	var req struct {
		CCTVID       string `json:"cctv_id"`
		TechnicianID string `json:"technician_id"`
		Notes        string `json:"notes"`
	}
	if c.BodyParser(&req) != nil || len(req.Notes) > 2000 {
		return c.SendStatus(400)
	}
	cctvID, err := uuid.Parse(req.CCTVID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "ID CCTV tidak valid"})
	}
	techID, err := uuid.Parse(req.TechnicianID)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "ID teknisi tidak valid"})
	}
	created, err := (incidents.Service{DB: config.DB}).CreateCameraFieldTask(c.UserContext(), incidentID, cctvID, site, techID, req.Notes, time.Now().UTC())
	if err != nil {
		return incidentActionError(c, err)
	}
	utils.LogAudit(config.DB, c, "CREATE_INCIDENT_CAMERA_TASK", "INCIDENT", "Membuat tugas pemeriksaan lapangan individual pada insiden "+incidentID.String(), nil, "", "")
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status":  "success",
		"message": "Tugas pemeriksaan lapangan kamera berhasil dibuat",
		"data":    created,
	})
}

func ResolveIncident(c *fiber.Ctx) error {
	site, ok := incidentSite(c)
	if !ok {
		return nil
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.SendStatus(400)
	}
	admin, ok := incidentAdminID(c)
	if !ok {
		return nil
	}
	var req struct {
		Note string `json:"note"`
	}
	if c.BodyParser(&req) != nil || len(req.Note) > 2000 || strings.TrimSpace(req.Note) == "" {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Catatan verifikasi Admin wajib diisi (maksimal 2000 karakter)"})
	}
	if err := (incidents.Service{DB: config.DB}).Resolve(c.UserContext(), id, site, admin, req.Note, time.Now().UTC()); err != nil {
		return incidentActionError(c, err)
	}
	utils.LogAudit(config.DB, c, "RESOLVE_INCIDENT", "INCIDENT", "Memverifikasi dan menutup insiden CCTV "+id.String(), nil, "", "")
	return c.JSON(fiber.Map{"status": "success", "message": "Insiden telah diverifikasi dan ditutup"})
}

func MergeIncidentsHandler(c *fiber.Ctx) error {
	site, ok := incidentSite(c)
	if !ok {
		return nil
	}
	targetID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "ID tiket induk tidak valid"})
	}
	admin, ok := incidentAdminID(c)
	if !ok {
		return nil
	}
	var req struct {
		SourceIncidentIDs []string `json:"source_incident_ids"`
		Reason            string   `json:"reason"`
	}
	if c.BodyParser(&req) != nil || len(req.Reason) > 2000 || len(req.SourceIncidentIDs) == 0 {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Pilih minimal satu tiket sumber dan isi alasan penggabungan"})
	}
	sourceUUIDs := make([]uuid.UUID, 0, len(req.SourceIncidentIDs))
	for _, raw := range req.SourceIncidentIDs {
		uid, parseErr := uuid.Parse(strings.TrimSpace(raw))
		if parseErr != nil {
			return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Format ID tiket sumber tidak valid"})
		}
		sourceUUIDs = append(sourceUUIDs, uid)
	}
	if err := (incidents.Service{DB: config.DB}).MergeIncidents(c.UserContext(), targetID, sourceUUIDs, site, admin, req.Reason, time.Now().UTC()); err != nil {
		return incidentActionError(c, err)
	}
	utils.LogAudit(config.DB, c, "MERGE_INCIDENT", "INCIDENT", "Menggabungkan tiket insiden ke tiket induk "+targetID.String(), nil, "", "")
	return c.JSON(fiber.Map{"status": "success", "message": "Tiket berhasil digabungkan ke tiket bersama dengan mempertahankan jejak deteksi awal"})
}

func EscalateIncidentReplacement(c *fiber.Ctx) error {
	site, ok := incidentSite(c)
	if !ok {
		return nil
	}
	incidentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "ID insiden tidak valid"})
	}
	admin, ok := incidentAdminID(c)
	if !ok {
		return nil
	}
	var req struct {
		Note  string `json:"note"`
		Items []struct {
			CCTVID                   string `json:"cctv_id"`
			DamageFindings           string `json:"damage_findings"`
			TechnicianID             string `json:"technician_id"`
			ExistingReplacementLogID string `json:"existing_replacement_log_id"`
		} `json:"items"`
	}
	if c.BodyParser(&req) != nil || len(req.Items) == 0 || len(req.Note) > 2000 {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Pilih minimal satu kamera beserta temuan kerusakan dan penanggung jawab penggantian"})
	}

	items := make([]incidents.EscalateCameraItem, 0, len(req.Items))
	for _, raw := range req.Items {
		cid, cErr := uuid.Parse(strings.TrimSpace(raw.CCTVID))
		if cErr != nil {
			return c.Status(400).JSON(fiber.Map{"status": "error", "message": "ID CCTV pada item eskalasi tidak valid"})
		}
		item := incidents.EscalateCameraItem{
			CCTVID:         cid,
			DamageFindings: strings.TrimSpace(raw.DamageFindings),
		}
		if strings.TrimSpace(raw.ExistingReplacementLogID) != "" {
			lid, lErr := uuid.Parse(strings.TrimSpace(raw.ExistingReplacementLogID))
			if lErr != nil {
				return c.Status(400).JSON(fiber.Map{"status": "error", "message": "ID log penggantian tertaut tidak valid"})
			}
			item.ExistingReplacementLogID = &lid
		}
		if strings.TrimSpace(raw.TechnicianID) != "" {
			tid, tErr := uuid.Parse(strings.TrimSpace(raw.TechnicianID))
			if tErr != nil {
				return c.Status(400).JSON(fiber.Map{"status": "error", "message": "ID teknisi penanggung jawab tidak valid"})
			}
			item.TechnicianID = tid
		}
		items = append(items, item)
	}

	repLogs, err := (incidents.Service{DB: config.DB}).EscalateReplacement(c.UserContext(), incidentID, site, admin, items, req.Note, time.Now().UTC())
	if err != nil {
		return incidentActionError(c, err)
	}

	// Lepas path streaming MediaMTX bagi seluruh kamera yang dinonaktifkan agar tidak ada tarikan RTSP tersisa.
	for _, it := range items {
		_ = streamClient.RemovePath(utils.BuildStreamPath(it.CCTVID.String()))
	}

	utils.LogAudit(config.DB, c, "ESCALATE_INCIDENT_REPLACEMENT", "INCIDENT", "Eskalasi penggantian perangkat CCTV pada insiden "+incidentID.String(), nil, "", "")
	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Eskalasi penggantian perangkat dicatat per kamera dan master CCTV dinonaktifkan sementara",
		"data":    repLogs,
	})
}

func RetryIncidentOutbox(c *fiber.Ctx) error {
	site, ok := incidentSite(c)
	if !ok {
		return nil
	}
	outboxID, err := uuid.Parse(c.Params("outbox_id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "ID antrean pemberitahuan tidak valid"})
	}
	if err := (incidents.Service{DB: config.DB}).ManualRetryOutbox(c.UserContext(), outboxID, site, time.Now().UTC()); err != nil {
		return incidentActionError(c, err)
	}
	utils.LogAudit(config.DB, c, "RETRY_INCIDENT_OUTBOX", "INCIDENT", "Menjadwalkan kirim ulang manual pemberitahuan Telegram "+outboxID.String(), nil, "", "")
	return c.JSON(fiber.Map{"status": "success", "message": "Pengiriman ulang pemberitahuan telah dijadwalkan"})
}

func ListPlannedOutagesHandler(c *fiber.Ctx) error {
	site, ok := incidentSite(c)
	if !ok {
		return nil
	}
	rows, err := (incidents.Service{DB: config.DB}).ListPlannedOutages(c.UserContext(), site)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"status": "error", "message": "Gagal memuat daftar jendela pemeliharaan terjadwal"})
	}
	return c.JSON(fiber.Map{"status": "success", "data": rows})
}

func CreatePlannedOutageHandler(c *fiber.Ctx) error {
	site, ok := incidentSite(c)
	if !ok {
		return nil
	}
	admin, ok := incidentAdminID(c)
	if !ok {
		return nil
	}
	var req struct {
		Reason   string   `json:"reason"`
		StartsAt string   `json:"starts_at"`
		EndsAt   string   `json:"ends_at"`
		CCTVIDs  []string `json:"cctv_ids"`
	}
	if c.BodyParser(&req) != nil || len(req.Reason) > 1000 {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Format input jendela pemeliharaan tidak valid"})
	}
	startsAt, err := time.Parse(time.RFC3339, strings.TrimSpace(req.StartsAt))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Format starts_at harus ISO-8601 / RFC3339"})
	}
	endsAt, err := time.Parse(time.RFC3339, strings.TrimSpace(req.EndsAt))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "Format ends_at harus ISO-8601 / RFC3339"})
	}
	cctvIDs := make([]uuid.UUID, 0, len(req.CCTVIDs))
	for _, raw := range req.CCTVIDs {
		cid, parseErr := uuid.Parse(strings.TrimSpace(raw))
		if parseErr != nil {
			return c.Status(400).JSON(fiber.Map{"status": "error", "message": "ID CCTV pada daftar pemeliharaan tidak valid"})
		}
		cctvIDs = append(cctvIDs, cid)
	}
	created, err := (incidents.Service{DB: config.DB}).CreatePlannedOutage(c.UserContext(), site, admin, req.Reason, startsAt.UTC(), endsAt.UTC(), cctvIDs, time.Now().UTC())
	if err != nil {
		return incidentActionError(c, err)
	}
	utils.LogAudit(config.DB, c, "CREATE_PLANNED_OUTAGE", "INCIDENT", "Membuat jendela penghentian kamera terjadwal: "+created.Reason, nil, "", "")
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status":  "success",
		"message": "Jendela pemeliharaan terjadwal berhasil dibuat",
		"data":    created,
	})
}

func EndPlannedOutageHandler(c *fiber.Ctx) error {
	site, ok := incidentSite(c)
	if !ok {
		return nil
	}
	admin, ok := incidentAdminID(c)
	if !ok {
		return nil
	}
	outageID, err := uuid.Parse(c.Params("outage_id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"status": "error", "message": "ID jendela pemeliharaan tidak valid"})
	}
	now := time.Now().UTC()
	svc := incidents.Service{DB: config.DB}
	if err := svc.EndPlannedOutageEarly(c.UserContext(), outageID, site, admin, now); err != nil {
		return incidentActionError(c, err)
	}
	_, needRefresh, _ := svc.ProcessExpiredPlannedOutages(c.UserContext(), now)
	if len(needRefresh) > 0 {
		DispatchPostMaintenanceRefreshes(config.DB, needRefresh)
	}
	utils.LogAudit(config.DB, c, "END_PLANNED_OUTAGE", "INCIDENT", "Mengakhiri jendela pemeliharaan terjadwal lebih awal: "+outageID.String(), nil, "", "")
	return c.JSON(fiber.Map{"status": "success", "message": "Jendela pemeliharaan terjadwal diakhiri dan evaluasi kamera pasca-pemeliharaan dijalankan"})
}

func incidentActionError(c *fiber.Ctx, err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return c.SendStatus(404)
	}
	if errors.Is(err, incidents.ErrCrossSite) {
		return c.Status(403).JSON(fiber.Map{"status": "error", "message": "Operasi lintas site tidak diizinkan"})
	}
	utils.Log.Warnf("Tindakan insiden ditolak: %v", err)
	msg := "Tindakan belum dapat dilakukan. Pastikan site, teknisi, dan bukti pemulihan masih valid."
	if errors.Is(err, incidents.ErrNotReady) || errors.Is(err, incidents.ErrInvalidMerge) || errors.Is(err, incidents.ErrInvalidEscalation) {
		msg = err.Error()
	}
	return c.Status(409).JSON(fiber.Map{"status": "error", "message": msg})
}

func loadTelegramRoutingFromEnv() (string, incidents.TelegramRouting) {
	token := strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN"))
	opsChats := map[string]string{}
	_ = json.Unmarshal([]byte(os.Getenv("TELEGRAM_SITE_CHATS_JSON")), &opsChats)

	adminChats := map[string]string{}
	_ = json.Unmarshal([]byte(os.Getenv("TELEGRAM_SITE_ADMIN_CHATS_JSON")), &adminChats)

	defaultAdmin := strings.TrimSpace(os.Getenv("TELEGRAM_ADMIN_CHAT_ID"))
	return token, incidents.TelegramRouting{
		SiteOpsChats:   opsChats,
		SiteAdminChats: adminChats,
		DefaultAdminID: defaultAdmin,
	}
}

func RunIncidentNotifications(db *gorm.DB) {
	if !IncidentMonitoringEnabled() || db == nil {
		return
	}
	ctx := context.Background()
	svc := incidents.Service{DB: db}
	if err := svc.ResetObservations(ctx); err != nil {
		utils.Log.Errorf("Gagal membatalkan observasi lama: %v", err)
		return
	}

	token, routing := loadTelegramRoutingFromEnv()
	opsSites, hasAdmin := routing.ConfiguredSites()
	telegramReady := token != "" && (len(opsSites) > 0 || hasAdmin)
	if !telegramReady {
		utils.Log.Warn("Antrean insiden aktif; token/grup Telegram belum lengkap. Evaluasi jendela pemeliharaan, SLA breach, dan pengingat in-app tetap berjalan.")
	}

	sender := incidents.TelegramSender{Token: token}
	limiter := incidents.NewChatRateLimiter(incidents.MinGroupChatInterval)

	deliveryTicker := time.NewTicker(2 * time.Second)
	defer deliveryTicker.Stop()
	evalTicker := time.NewTicker(5 * time.Second)
	defer evalTicker.Stop()

	for {
		select {
		case now := <-evalTicker.C:
			utcNow := now.UTC()
			if _, needRefresh, err := svc.ProcessExpiredPlannedOutages(ctx, utcNow); err != nil {
				utils.Log.Errorf("Evaluasi akhir jendela pemeliharaan gagal: %v", err)
			} else if len(needRefresh) > 0 {
				DispatchPostMaintenanceRefreshes(db, needRefresh)
			}
			if _, _, _, err := svc.EvaluateRemindersAndSLABreaches(ctx, utcNow); err != nil {
				utils.Log.Errorf("Evaluasi SLA breach & pengingat insiden gagal: %v", err)
			}
		case now := <-deliveryTicker.C:
			if !telegramReady {
				continue
			}
			utcNow := now.UTC()
			for burst := 0; burst < 4; burst++ {
				worked, err := incidents.DeliverWithRouting(ctx, db, sender, routing, limiter, utcNow)
				if err != nil {
					utils.Log.Errorf("Antrean Telegram gagal diproses: %v", err)
					break
				}
				if !worked {
					break
				}
			}
		}
	}
}
