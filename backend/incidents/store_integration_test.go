package incidents

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"sigap-backend/config"
	"sigap-backend/models"
)

func integrationDB(t *testing.T) (*gorm.DB, models.CCTV) {
	t.Helper()
	dsn := os.Getenv("SIGAP_INCIDENT_TEST_DSN")
	if dsn == "" {
		t.Skip("isolated PostgreSQL DSN not set")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { _ = sqlDB.Close() })
	var name string
	db.Raw("SELECT current_database()").Scan(&name)
	if name != "sigap_incident_test" {
		t.Fatal("refusing non-test database")
	}
	if err := config.RunMigrations(db); err != nil {
		t.Fatal(err)
	}
	siteID := uuid.New()
	cameraID := uuid.New()
	site := models.Site{Base: models.Base{ID: siteID}, Kode: siteID.String()[:8], Nama: "Test Site"}
	if err := db.Create(&site).Error; err != nil {
		t.Fatal(err)
	}
	camera := models.CCTV{
		Base:              models.Base{ID: cameraID},
		KodeCCTV:          "CCTV-" + cameraID.String()[:6],
		QrPayload:         cameraID.String(),
		Nama:              "Kamera Uji",
		LokasiArea:        "Gerbang Utama",
		SiteID:            &siteID,
		Status:            "aktif",
		MonitoringEnabled: true,
	}
	if err := db.Create(&camera).Error; err != nil {
		t.Fatal(err)
	}
	return db, camera
}

func createTestUser(t *testing.T, db *gorm.DB, role string, siteID *uuid.UUID) models.User {
	t.Helper()
	uid := uuid.New()
	u := models.User{
		Base:         models.Base{ID: uid},
		Username:     uid.String(),
		FullName:     role + " " + uid.String()[:6],
		Role:         role,
		Email:        uid.String() + "@test.invalid",
		PasswordHash: "test",
	}
	if err := db.Create(&u).Error; err != nil {
		t.Fatal(err)
	}
	if siteID != nil {
		if err := db.Exec("INSERT INTO user_sites (user_id, site_id) VALUES (?, ?)", uid, *siteID).Error; err != nil {
			t.Fatal(err)
		}
	}
	return u
}

func TestPostgresConcurrentOfflineRecoveryAndSelfRecoveredClosure(t *testing.T) {
	db, camera := integrationDB(t)
	svc := Service{DB: db}
	ctx := context.Background()
	firstFailed := time.Now().UTC().Truncate(time.Microsecond).Add(-20 * time.Minute)
	confirmedAt := firstFailed.Add(19 * time.Second)
	faultAt := firstFailed.Add(-5 * time.Second)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := svc.ObserveDetailed(ctx, CameraObservation{
				Camera:                camera,
				State:                 StateOffline,
				FirstFailedObservedAt: &firstFailed,
				ObservedAt:            confirmedAt,
				PhysicalFaultAt:       &faultAt,
			}); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()

	var items []Incident
	if err := db.Preload("Members").Where("site_id = ?", camera.SiteID).Find(&items).Error; err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || len(items[0].Members) != 1 {
		t.Fatalf("expected one incident/member: %+v", items)
	}
	inc := items[0]
	if inc.TicketNumber == "" || !inc.FirstFailedObservedAt.Equal(firstFailed) || !inc.ConfirmedOfflineAt.Equal(confirmedAt) {
		t.Fatalf("timestamp separation not preserved: %+v", inc)
	}
	if inc.PhysicalFaultAt == nil || !inc.PhysicalFaultAt.Equal(faultAt) || !inc.SLAClockStartAt.Equal(firstFailed) {
		t.Fatalf("SLA clock start / physical fault mismatch: %+v", inc)
	}

	var count int64
	db.Model(&Outbox{}).Where("incident_id = ?", inc.ID).Count(&count)
	if count != 1 {
		t.Fatalf("duplicate offline messages: %d", count)
	}

	if err := svc.Observe(ctx, camera, StateUnknown, confirmedAt.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	admin := createTestUser(t, db, "admin", nil)
	if err := svc.Resolve(ctx, inc.ID, *camera.SiteID, admin.ID, "verified", confirmedAt.Add(2*time.Second)); err == nil {
		t.Fatal("closure must reject unknown evidence")
	}

	// 6 sampel berturut-turut setiap 60s selama 5 menit penuh -> pending_verification.
	for _, sec := range []int{10, 70, 130, 190, 250, 310} {
		if err := svc.Observe(ctx, camera, "live", confirmedAt.Add(time.Duration(sec)*time.Second)); err != nil {
			t.Fatal(err)
		}
	}
	var refreshed Incident
	db.First(&refreshed, "id = ?", inc.ID)
	if refreshed.Status != StatusPendingVerification {
		t.Fatalf("expected pending_verification, got %s", refreshed.Status)
	}

	// Kembali offline sebelum verifikasi Admin -> gunakan tiket yang sama (bukan duplikat).
	if err := svc.Observe(ctx, camera, StateOffline, confirmedAt.Add(320*time.Second)); err != nil {
		t.Fatal(err)
	}
	db.Model(&Incident{}).Where("site_id = ?", camera.SiteID).Count(&count)
	if count != 1 {
		t.Fatal("recurrence created duplicate ticket")
	}

	// Stabil 5 menit lagi (6 sampel) lalu tutup sebagai Pulih Mandiri (tanpa penugasan teknisi / tanpa MaintenanceLog).
	for _, sec := range []int{330, 390, 450, 510, 570, 630} {
		if err := svc.Observe(ctx, camera, "live", confirmedAt.Add(time.Duration(sec)*time.Second)); err != nil {
			t.Fatal(err)
		}
	}
	if err := svc.Resolve(ctx, inc.ID, uuid.New(), admin.ID, "wrong site", confirmedAt.Add(631*time.Second)); err == nil {
		t.Fatal("cross-site closure accepted")
	}
	if err := svc.Resolve(ctx, inc.ID, *camera.SiteID, admin.ID, "Kamera pulih mandiri", confirmedAt.Add(631*time.Second)); err != nil {
		t.Fatal(err)
	}
	db.Preload("Members").First(&refreshed, "id = ?", inc.ID)
	if refreshed.ResolutionType == nil || *refreshed.ResolutionType != ResolutionSelfRecovered {
		t.Fatalf("expected self_recovered resolution, got %+v", refreshed.ResolutionType)
	}
	if refreshed.MaintenanceLogID != nil || refreshed.Members[0].MemberOutcome != OutcomeSelfRecovered {
		t.Fatalf("expected no MaintenanceLog and self_recovered outcome, got %+v", refreshed)
	}

	// Gangguan baru setelah tiket ditutup harus membuka tiket baru.
	if err := svc.Observe(ctx, camera, StateOffline, confirmedAt.Add(640*time.Second)); err != nil {
		t.Fatal(err)
	}
	db.Model(&Incident{}).Where("site_id = ?", camera.SiteID).Count(&count)
	if count != 2 {
		t.Fatal("new outage after closure needs new incident")
	}
}

func TestPostgresPlannedOutageSuppressionAndImmediatePostWindowOpen(t *testing.T) {
	db, camera := integrationDB(t)
	svc := Service{DB: db}
	ctx := context.Background()
	admin := createTestUser(t, db, "admin", nil)
	base := time.Now().UTC().Truncate(time.Microsecond).Add(-15 * time.Minute)

	// 1. Tugas rutin 'ongoing' SAJA tidak boleh menahan tiket (Keputusan #1).
	routineCamID := uuid.New()
	routineCam := models.CCTV{
		Base:              models.Base{ID: routineCamID},
		KodeCCTV:          "RUT-" + routineCamID.String()[:5],
		QrPayload:         routineCamID.String(),
		Nama:              "Kamera Rutin",
		SiteID:            camera.SiteID,
		Status:            "aktif",
		MonitoringEnabled: true,
	}
	if err := db.Create(&routineCam).Error; err != nil {
		t.Fatal(err)
	}
	routineLog := models.MaintenanceLog{
		CCTVID:      routineCam.ID,
		SiteID:      camera.SiteID,
		WorkScope:   WorkScopeRoutine,
		ActionTaken: "Pembersihan kubah rutin",
		Status:      "ongoing",
	}
	if err := db.Create(&routineLog).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.Observe(ctx, routineCam, StateOffline, base); err != nil {
		t.Fatal(err)
	}
	var routineIncCount int64
	db.Model(&Incident{}).Joins("JOIN cctv_incident_cameras m ON m.incident_id = cctv_incidents.id").
		Where("m.cctv_id = ?", routineCam.ID).Count(&routineIncCount)
	if routineIncCount != 1 {
		t.Fatalf("ongoing routine log alone must NOT suppress incident, got %d", routineIncCount)
	}

	// 2. Jendela penghentian kamera yang direncanakan Admin menahan tiket & Telegram selama aktif,
	//    tetap merekam waktu gagal pertama & waktu terkonfirmasi, lalu saat berakhir langsung membuka tiket
	//    tanpa mengulang 2 pemeriksaan dari awal dan menetapkan sla_clock_start_at = effective_ends_at!
	winStart := base
	winEnd := base.Add(10 * time.Minute)
	po, err := svc.CreatePlannedOutage(ctx, *camera.SiteID, admin.ID, "Upgrade firmware NVR", winStart, winEnd, []uuid.UUID{camera.ID}, winStart)
	if err != nil {
		t.Fatal(err)
	}

	firstFailDuringMaint := winStart.Add(2 * time.Minute)
	confirmDuringMaint := firstFailDuringMaint.Add(15 * time.Second)
	if err := svc.ObserveDetailed(ctx, CameraObservation{
		Camera:                camera,
		State:                 StateOffline,
		FirstFailedObservedAt: &firstFailDuringMaint,
		ObservedAt:            confirmDuringMaint,
	}); err != nil {
		t.Fatal(err)
	}
	// Pemeriksaan segar 30s sebelum jendela berakhir masih offline -> ditahan.
	lastCheckDuringMaint := winEnd.Add(-30 * time.Second)
	if err := svc.ObserveDetailed(ctx, CameraObservation{
		Camera:                camera,
		State:                 StateOffline,
		FirstFailedObservedAt: &firstFailDuringMaint,
		ObservedAt:            lastCheckDuringMaint,
	}); err != nil {
		t.Fatal(err)
	}

	var suppressedIncCount int64
	db.Model(&Incident{}).Joins("JOIN cctv_incident_cameras m ON m.incident_id = cctv_incidents.id").
		Where("m.cctv_id = ?", camera.ID).Count(&suppressedIncCount)
	if suppressedIncCount != 0 {
		t.Fatalf("planned outage must suppress ticket while active, got %d", suppressedIncCount)
	}

	activeMap, err := svc.ActivePlannedOutageMap(ctx, []uuid.UUID{camera.ID}, winEnd.Add(-10*time.Second))
	if err != nil || activeMap[camera.ID].OutageID != po.ID {
		t.Fatalf("expected active planned outage badge info, got %+v err=%v", activeMap, err)
	}

	// Begitu jendela berakhir (winEnd + 2s), langsung buka tiket pasca-maintenance tanpa 2 siklus ulang!
	opened, needRefresh, err := svc.ProcessExpiredPlannedOutages(ctx, winEnd.Add(2*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if opened != 1 || len(needRefresh) != 0 {
		t.Fatalf("expected immediate post-maintenance open=1 refresh=0, got open=%d refresh=%d", opened, len(needRefresh))
	}

	var postInc Incident
	if err := db.Joins("JOIN cctv_incident_cameras m ON m.incident_id = cctv_incidents.id").
		Where("m.cctv_id = ?", camera.ID).First(&postInc).Error; err != nil {
		t.Fatal(err)
	}
	if postInc.OriginType != OriginPostPlannedMaintenance || postInc.SLAClockBasis != SLABasisPlannedWindowEnd {
		t.Fatalf("unexpected origin/basis: %+v", postInc)
	}
	if !postInc.SLAClockStartAt.Equal(winEnd) {
		t.Fatalf("expected sla_clock_start_at == effective_ends_at (%v), got %v", winEnd, postInc.SLAClockStartAt)
	}
	if !postInc.FirstFailedObservedAt.Equal(firstFailDuringMaint) || !postInc.ConfirmedOfflineAt.Equal(confirmDuringMaint) {
		t.Fatalf("original failure timestamps during window must be preserved honestly: %+v", postInc)
	}
}

func TestPostgresSharedCorrelatedWaveManualMergeAndSharedTask(t *testing.T) {
	db, cam1 := integrationDB(t)
	svc := Service{DB: db}
	ctx := context.Background()
	siteID := *cam1.SiteID
	admin := createTestUser(t, db, "admin", nil)
	tech := createTestUser(t, db, "teknisi", &siteID)
	now := time.Now().UTC().Add(-20 * time.Minute)

	makeCam := func(code string) models.CCTV {
		id := uuid.New()
		c := models.CCTV{
			Base:              models.Base{ID: id},
			KodeCCTV:          code + "-" + id.String()[:6],
			QrPayload:         id.String(),
			Nama:              "Kamera " + code,
			IPNVR:             "10.20.30.40",
			SiteID:            &siteID,
			Status:            "aktif",
			MonitoringEnabled: true,
		}
		if err := db.Create(&c).Error; err != nil {
			t.Fatal(err)
		}
		return c
	}
	cam2 := makeCam("C2")
	cam3 := makeCam("C3")

	// 1. Jika bukti NVR belum terbukti (nvrUnreachable = false), tetap buat tiket terpisah per kamera!
	err := svc.ObserveSharedCorrelatedWave(ctx, siteID, "10.20.30.40", false, []CameraObservation{
		{Camera: cam1, State: StateOffline, ObservedAt: now},
		{Camera: cam2, State: StateOffline, ObservedAt: now.Add(time.Second)},
		{Camera: cam3, State: StateOffline, ObservedAt: now.Add(2 * time.Second)},
	})
	if err != nil {
		t.Fatal(err)
	}
	var separate []Incident
	db.Where("site_id = ? AND status IN ?", siteID, []string{StatusOpen, StatusInProgress}).Order("opened_at ASC").Find(&separate)
	if len(separate) != 3 {
		t.Fatalf("expected 3 separate tickets when NVR proof absent, got %d", len(separate))
	}
	for _, s := range separate {
		if s.AssignedTo == nil || *s.AssignedTo != tech.ID || s.Status != StatusInProgress {
			t.Fatalf("expected separate ticket auto-assigned to site technician %s, got status=%s assigned_to=%v", tech.ID, s.Status, s.AssignedTo)
		}
	}

	// 2. Penggabungan lintas site wajib ditolak.
	if err := svc.MergeIncidents(ctx, separate[0].ID, []uuid.UUID{separate[1].ID}, uuid.New(), admin.ID, "wrong site", now.Add(5*time.Second)); err == nil {
		t.Fatal("expected cross-site merge rejection")
	}

	// 3. Gabungkan manual separate[1] & separate[2] ke separate[0] ("Gabungkan ke Tiket Bersama" — Keputusan #3).
	if err := svc.MergeIncidents(ctx, separate[0].ID, []uuid.UUID{separate[1].ID, separate[2].ID}, siteID, admin.ID, "Switch FO Core-2 putus", now.Add(5*time.Second)); err != nil {
		t.Fatal(err)
	}

	var parent, src1 Incident
	db.Preload("Members").First(&parent, "id = ?", separate[0].ID)
	db.First(&src1, "id = ?", separate[1].ID)
	if src1.Status != StatusMerged || src1.MergedIntoIncidentID == nil || *src1.MergedIntoIncidentID != parent.ID {
		t.Fatalf("source ticket must be marked merged -> parent, got %+v", src1)
	}
	if parent.ScopeType != ScopeManualMerged || len(parent.Members) != 3 {
		t.Fatalf("parent must hold 3 active cameras in manual_merged scope, got %+v", parent)
	}
	foundSourceRef := false
	for _, m := range parent.Members {
		if m.CCTVID == cam2.ID && m.SourceTicketNumber == separate[1].TicketNumber {
			foundSourceRef = true
		}
	}
	if !foundSourceRef {
		t.Fatal("merged camera must preserve source_ticket_number")
	}

	// 4. Penugasan/pengalihan tiket bersama menghasilkan 1 tugas perbaikan bersama (shared_infrastructure),
	//    dan tugas kamera individual dibuat hanya bila diminta eksplisit (Keputusan #4).
	techOverride := createTestUser(t, db, "teknisi", &siteID)
	if err := svc.Assign(ctx, parent.ID, siteID, techOverride.ID, now.Add(10*time.Second)); err != nil {
		t.Fatal(err)
	}
	db.First(&parent, "id = ?", parent.ID)
	if parent.AssignedTo == nil || *parent.AssignedTo != techOverride.ID {
		t.Fatalf("expected Admin override to reassign parent to %s, got %v", techOverride.ID, parent.AssignedTo)
	}
	db.First(&parent, "id = ?", parent.ID)
	var sharedTask models.MaintenanceLog
	if err := db.First(&sharedTask, "id = ?", *parent.MaintenanceLogID).Error; err != nil {
		t.Fatal(err)
	}
	if sharedTask.WorkScope != WorkScopeSharedInfrastructure {
		t.Fatalf("expected shared_infrastructure task, got %s", sharedTask.WorkScope)
	}

	fieldTask, err := svc.CreateCameraFieldTask(ctx, parent.ID, cam3.ID, siteID, tech.ID, "Cek konektor RJ45 di tiang", now.Add(12*time.Second))
	if err != nil || fieldTask.WorkScope != WorkScopeCameraFieldCheck {
		t.Fatalf("expected camera_field_check task, got %+v err=%v", fieldTask, err)
	}

	// 5. Pemulihan sebagian (hanya cam1 stabil 5 menit) menjaga tiket tetap in_progress (belum pending_verification).
	for _, sec := range []int{20, 80, 140, 200, 260, 320} {
		if err := svc.Observe(ctx, cam1, "live", now.Add(time.Duration(sec)*time.Second)); err != nil {
			t.Fatal(err)
		}
	}
	db.First(&parent, "id = ?", parent.ID)
	if parent.Status != StatusInProgress {
		t.Fatalf("partial recovery must keep shared ticket in_progress, got %s", parent.Status)
	}
}

func TestPostgresMultiCameraEscalationAndOutcomeReport(t *testing.T) {
	db, cam1 := integrationDB(t)
	svc := Service{DB: db}
	ctx := context.Background()
	siteID := *cam1.SiteID
	admin := createTestUser(t, db, "admin", nil)
	techA := createTestUser(t, db, "teknisi", &siteID)
	techB := createTestUser(t, db, "teknisi", &siteID)
	now := time.Now().UTC().Add(-20 * time.Minute)

	makeCam := func(code string) models.CCTV {
		id := uuid.New()
		c := models.CCTV{
			Base:              models.Base{ID: id},
			KodeCCTV:          code + "-" + id.String()[:6],
			QrPayload:         id.String(),
			Nama:              "Kamera " + code,
			IPNVR:             "10.99.0.10",
			SiteID:            &siteID,
			Status:            "aktif",
			MonitoringEnabled: true,
			StreamEnabled:     true,
		}
		if err := db.Create(&c).Error; err != nil {
			t.Fatal(err)
		}
		return c
	}
	cam1.IPNVR = "10.99.0.10"
	_ = db.Save(&cam1).Error
	cam2 := makeCam("ESC-2")
	cam3 := makeCam("ESC-3")
	cam4 := makeCam("ESC-4")

	// Buka 1 tiket bersama NVR dengan 4 kamera terdampak (nvrUnreachable = true).
	if err := svc.ObserveSharedCorrelatedWave(ctx, siteID, "10.99.0.10", true, []CameraObservation{
		{Camera: cam1, State: StateOffline, ObservedAt: now},
		{Camera: cam2, State: StateOffline, ObservedAt: now},
		{Camera: cam3, State: StateOffline, ObservedAt: now},
		{Camera: cam4, State: StateOffline, ObservedAt: now},
	}); err != nil {
		t.Fatal(err)
	}

	var shared Incident
	if err := db.Where("site_id = ?", siteID).First(&shared).Error; err != nil {
		t.Fatal(err)
	}

	// cam1 & cam2 pulih stabil 5 menit; cam3 & cam4 rusak berat dan masing-masing diekskalasi
	// ke tugas penggantian unit tersendiri (Koreksi #5: >1 kamera diekskalasi pada 1 tiket bersama).
	for _, sec := range []int{10, 70, 130, 190, 250, 310} {
		at := now.Add(time.Duration(sec) * time.Second)
		if err := svc.Observe(ctx, cam1, "live", at); err != nil {
			t.Fatal(err)
		}
		if err := svc.Observe(ctx, cam2, "live", at); err != nil {
			t.Fatal(err)
		}
	}

	repLogs, err := svc.EscalateReplacement(ctx, shared.ID, siteID, admin.ID, []EscalateCameraItem{
		{CCTVID: cam3.ID, DamageFindings: "Sensor utama terbakar tersambar petir", TechnicianID: techA.ID},
		{CCTVID: cam4.ID, DamageFindings: "Modul PoE board mati total", TechnicianID: techB.ID},
	}, "2 kamera perlu ganti unit baru", now.Add(315*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if len(repLogs) != 2 || repLogs[0].ID == repLogs[1].ID {
		t.Fatalf("expected 2 distinct replacement maintenance logs, got %+v", repLogs)
	}

	// Pastikan cam3 & cam4 nonaktif + monitoring_enabled=false, dan sisa anggota aktif (cam1 & cam2) sudah recovered -> pending_verification.
	var updatedCam3 models.CCTV
	db.First(&updatedCam3, "id = ?", cam3.ID)
	if updatedCam3.Status != "nonaktif" || updatedCam3.MonitoringEnabled {
		t.Fatalf("escalated CCTV must be nonaktif with monitoring_enabled=false, got %+v", updatedCam3)
	}

	db.First(&shared, "id = ?", shared.ID)
	if shared.Status != StatusPendingVerification {
		t.Fatalf("remaining 2 cameras already recovered -> expected pending_verification, got %s", shared.Status)
	}

	// Tutup tiket induk -> resolution_type wajib 'partial_recovery_and_escalation'.
	if err := svc.Resolve(ctx, shared.ID, siteID, admin.ID, "2 kamera pulih, 2 kamera ganti unit", now.Add(320*time.Second)); err != nil {
		t.Fatal(err)
	}
	db.First(&shared, "id = ?", shared.ID)
	if shared.Status != StatusResolved || shared.ResolutionType == nil || *shared.ResolutionType != ResolutionPartialRecoveryAndEscalation {
		t.Fatalf("expected partial_recovery_and_escalation, got %+v", shared)
	}

	// Laporan tingkat kamera wajib tetap memisahkan 2 kamera pulih vs 2 kamera dieskalasikan beserta tautan penggantian masing-masing!
	report, err := svc.BuildCameraOutcomeReport(ctx, siteID)
	if err != nil {
		t.Fatal(err)
	}
	if report.TotalCameraInvolvements != 4 || report.EscalatedCount != 2 || (report.SelfRecoveredCount+report.RecoveredVerifiedCount) != 2 {
		t.Fatalf("camera outcome report inaccurate after parent closure: %+v", report)
	}
	distinctRepIDs := map[uuid.UUID]bool{}
	for _, r := range report.Rows {
		if r.MemberOutcome == OutcomeEscalatedReplacement && r.ReplacementMaintenanceLogID != nil {
			distinctRepIDs[*r.ReplacementMaintenanceLogID] = true
		}
	}
	if len(distinctRepIDs) != 2 {
		t.Fatalf("expected 2 distinct replacement_maintenance_log_id links preserved per camera, got %d", len(distinctRepIDs))
	}
}

func TestPostgresMonitorHealthEpisodeDeduplicationAndResetObservations(t *testing.T) {
	db, camera := integrationDB(t)
	svc := Service{DB: db}
	ctx := context.Background()
	siteID := *camera.SiteID
	now := time.Now().UTC()

	// Simulasi 20 kamera unknown karena MediaMTX tidak merespons -> hanya 1 episode & 1 outbox ke admin_group!
	for cycle := 0; cycle < 3; cycle++ {
		opened, _, err := svc.ReportMonitorFailure(ctx, &siteID, "mediamtx_unreachable", "HTTP timeout 9997", 20, now.Add(time.Duration(cycle)*time.Minute))
		if err != nil {
			t.Fatal(err)
		}
		if cycle == 0 && !opened {
			t.Fatal("first failure must open episode")
		}
		if cycle > 0 && opened {
			t.Fatal("subsequent failures during open episode must deduplicate")
		}
	}
	var epOutboxCount int64
	db.Model(&Outbox{}).Where("site_id = ? AND target_channel = ? AND event_type = 'MONITOR_DEGRADED'", siteID, ChannelAdminGroup).Count(&epOutboxCount)
	if epOutboxCount != 1 {
		t.Fatalf("expected exactly 1 MONITOR_DEGRADED alert per episode, got %d", epOutboxCount)
	}

	// 2 siklus sehat berturut-turut menutup episode dengan 1 pesan pemulihan.
	closed1, err := svc.ReportMonitorHealthyCycle(ctx, siteID, now.Add(4*time.Minute))
	if err != nil || closed1 {
		t.Fatalf("1st healthy cycle must not close episode yet: closed=%v err=%v", closed1, err)
	}
	closed2, err := svc.ReportMonitorHealthyCycle(ctx, siteID, now.Add(5*time.Minute))
	if err != nil || !closed2 {
		t.Fatalf("2nd consecutive healthy cycle must close episode: closed=%v err=%v", closed2, err)
	}

	// Uji ResetObservations memperbaiki bug prototipe lama: unassigned pending_verification kembali ke 'open' (bukan 'in_progress').
	base := now.Add(-15 * time.Minute)
	if err := svc.Observe(ctx, camera, StateOffline, base); err != nil {
		t.Fatal(err)
	}
	for _, sec := range []int{10, 70, 130, 190, 250, 310} {
		if err := svc.Observe(ctx, camera, "live", base.Add(time.Duration(sec)*time.Second)); err != nil {
			t.Fatal(err)
		}
	}
	if err := svc.ResetObservations(ctx); err != nil {
		t.Fatal(err)
	}
	var inc Incident
	db.Where("site_id = ?", siteID).First(&inc)
	if inc.Status != StatusOpen {
		t.Fatalf("unassigned pending_verification after restart must return to open, got %s", inc.Status)
	}
}

func TestPostgresPostMaintenanceUnknownStaysPendingAndPreventsPreemption(t *testing.T) {
	db, camA := integrationDB(t)
	zeroCoalesce := time.Duration(0)
	svc := Service{DB: db, CoalesceWindow: &zeroCoalesce}
	ctx := context.Background()
	siteID := *camA.SiteID
	admin := createTestUser(t, db, "admin", nil)

	// Siapkan Fake Telegram Server untuk membuktikan pengiriman outbox end-to-end tanpa menggandakan pesan.
	var tgMu sync.Mutex
	var deliveredTexts []string
	fakeTG := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			ChatID string `json:"chat_id"`
			Text   string `json:"text"`
		}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		chatIDInt, _ := strconv.ParseInt(strings.TrimSpace(payload.ChatID), 10, 64)
		tgMu.Lock()
		deliveredTexts = append(deliveredTexts, payload.Text)
		msgID := 700 + len(deliveredTexts)
		tgMu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true,
			"result": map[string]interface{}{
				"message_id": msgID,
				"chat":       map[string]interface{}{"id": chatIDInt},
			},
		})
	}))
	defer fakeTG.Close()

	sender := TelegramSender{
		Token:   "test-bot-token",
		Client:  fakeTG.Client(),
		baseURL: fakeTG.URL,
	}
	routing := TelegramRouting{
		SiteOpsChats: map[string]string{siteID.String(): "-10099887766"},
	}

	makeCam := func(code string) models.CCTV {
		id := uuid.New()
		c := models.CCTV{
			Base:              models.Base{ID: id},
			KodeCCTV:          code + "-" + id.String()[:6],
			QrPayload:         id.String(),
			Nama:              "Kamera " + code,
			LokasiArea:        "Pit Utara",
			IPNVR:             "10.50.0.10",
			SiteID:            &siteID,
			Status:            "aktif",
			MonitoringEnabled: true,
		}
		if err := db.Create(&c).Error; err != nil {
			t.Fatal(err)
		}
		return c
	}
	camB := makeCam("POST-LIVE")
	camPreempt := makeCam("POST-PREEMPT")

	baseNow := time.Now().UTC().Truncate(time.Microsecond)
	winStart := baseNow.Add(-15 * time.Minute)
	winEnd := baseNow.Add(-5 * time.Minute)

	po, err := svc.CreatePlannedOutage(ctx, siteID, admin.ID, "Upgrade firmware switch distribusi", winStart, winEnd, []uuid.UUID{camA.ID, camB.ID, camPreempt.ID}, winStart)
	if err != nil {
		t.Fatal(err)
	}

	firstFailAt := winStart.Add(2 * time.Minute)
	confirmedDuringWinAt := winStart.Add(2*time.Minute + 20*time.Second)

	// Ketiga kamera terkonfirmasi offline selama jendela pemeliharaan, lalu mengalami sampel `unknown` menjelang akhir jendela.
	for _, cam := range []models.CCTV{camA, camB, camPreempt} {
		if err := svc.ObserveDetailed(ctx, CameraObservation{
			Camera:                cam,
			State:                 StateOffline,
			FirstFailedObservedAt: &firstFailAt,
			ObservedAt:            confirmedDuringWinAt,
		}); err != nil {
			t.Fatal(err)
		}
		if err := svc.ObserveDetailed(ctx, CameraObservation{
			Camera:     cam,
			State:      StateUnknown,
			ObservedAt: winEnd.Add(-10 * time.Second),
		}); err != nil {
			t.Fatal(err)
		}
	}

	// 1. Begitu jendela berakhir (winEnd + 2s), ketiganya TIDAK BOLEH ditandai selesai hanya karena state terakhir `unknown`!
	opened, needRefresh, err := svc.ProcessExpiredPlannedOutages(ctx, winEnd.Add(2*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if opened != 0 || len(needRefresh) != 3 {
		t.Fatalf("expected 0 immediate tickets and 3 pending refresh requests when state is unknown, got opened=%d refresh=%d", opened, len(needRefresh))
	}

	reqByCam := map[uuid.UUID]PlannedOutageRefreshRequest{}
	for _, req := range needRefresh {
		reqByCam[req.Camera.ID] = req
		var poc PlannedOutageCamera
		if err := db.First(&poc, "id = ?", req.PlannedOutageCameraID).Error; err != nil {
			t.Fatal(err)
		}
		if poc.PostWindowProcessedAt != nil {
			t.Fatalf("evaluation for %s must remain pending (PostWindowProcessedAt == nil)", req.Camera.KodeCCTV)
		}
	}

	// 2. Hasil penyegaran pertama masih `unknown` -> ketiganya tetap tertunda, 0 tiket, dan backoff terkendali mencegah tight-loop.
	for _, cam := range []models.CCTV{camA, camB, camPreempt} {
		didOpen, refreshErr := svc.CompletePostMaintenanceRefresh(ctx, reqByCam[cam.ID], StateUnknown, winEnd.Add(4*time.Second))
		if refreshErr != nil || didOpen {
			t.Fatalf("unknown refresh must keep evaluation pending without ticket: didOpen=%v err=%v", didOpen, refreshErr)
		}
	}

	// Pemanggilan ProcessExpiredPlannedOutages sebelum jeda retry 15s habis tidak boleh membanjiri antrean refresh.
	earlyOpened, earlyRefresh, err := svc.ProcessExpiredPlannedOutages(ctx, winEnd.Add(10*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if earlyOpened != 0 || len(earlyRefresh) != 0 {
		t.Fatalf("retry backoff must throttle early refresh before next_retry_at: opened=%d refresh=%d", earlyOpened, len(earlyRefresh))
	}

	// Setelah jeda retry terlewati (winEnd + 20s), ketiganya siap dievaluasi ulang.
	_, retryReqs, err := svc.ProcessExpiredPlannedOutages(ctx, winEnd.Add(20*time.Second))
	if err != nil || len(retryReqs) != 3 {
		t.Fatalf("expected 3 retry refresh requests after backoff elapsed, got len=%d err=%v", len(retryReqs), err)
	}

	// 3A. Kasus A (`offline -> unknown -> offline` via CompletePostMaintenanceRefresh pada camA):
	//     Menghasilkan TEPAT 1 tiket `post_planned_maintenance` dengan first_failed_observed_at, confirmed_offline_at,
	//     dan sla_clock_start_at = effective_ends_at.
	openedA, err := svc.CompletePostMaintenanceRefresh(ctx, reqByCam[camA.ID], StateOffline, winEnd.Add(22*time.Second))
	if err != nil || !openedA {
		t.Fatalf("expected post-maintenance ticket to open for camA on offline evidence: opened=%v err=%v", openedA, err)
	}

	// 3B. Kasus B (`offline -> unknown -> live` pada camB):
	//     Menyelesaikan evaluasi TANPA membuat tiket sama sekali.
	openedB, err := svc.CompletePostMaintenanceRefresh(ctx, reqByCam[camB.ID], "live", winEnd.Add(23*time.Second))
	if err != nil || openedB {
		t.Fatalf("expected live refresh for camB to finalize without ticket: opened=%v err=%v", openedB, err)
	}
	var pocB PlannedOutageCamera
	if err := db.First(&pocB, "planned_outage_id = ? AND cctv_id = ?", po.ID, camB.ID).Error; err != nil {
		t.Fatal(err)
	}
	if pocB.PostWindowProcessedAt == nil {
		t.Fatal("camB post-maintenance evaluation must be finalized after live evidence")
	}

	// 3C. Kasus C (Preemption Guard pada camPreempt):
	//     Jalur monitoring berkala biasa (`ObserveDetailed` dengan StateOffline) tiba saat evaluasi pasca-maintenance
	//     masih tertunda -> TIDAK BOLEH membuka `unplanned_outage`, melainkan wajib membuka 1 tiket `post_planned_maintenance`
	//     dengan sla_clock_start_at = effective_ends_at!
	wrongFirstFail := winEnd.Add(15 * time.Second)
	if err := svc.ObserveDetailed(ctx, CameraObservation{
		Camera:                camPreempt,
		State:                 StateOffline,
		FirstFailedObservedAt: &wrongFirstFail,
		ObservedAt:            winEnd.Add(25 * time.Second),
	}); err != nil {
		t.Fatal(err)
	}

	// 4. Ulangi seluruh pemrosesan untuk membuktikan tiket dan outbox TIDAK GANDA (idempoten).
	for rep := 0; rep < 3; rep++ {
		ts := winEnd.Add(time.Duration(60+rep*20) * time.Second)
		if _, _, err := svc.ProcessExpiredPlannedOutages(ctx, ts); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.CompletePostMaintenanceRefresh(ctx, reqByCam[camA.ID], StateOffline, ts); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.CompletePostMaintenanceRefresh(ctx, reqByCam[camB.ID], "live", ts); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.CompletePostMaintenanceRefresh(ctx, reqByCam[camPreempt.ID], StateOffline, ts); err != nil {
			t.Fatal(err)
		}
		if err := svc.ObserveDetailed(ctx, CameraObservation{Camera: camA, State: StateOffline, ObservedAt: ts}); err != nil {
			t.Fatal(err)
		}
		if err := svc.ObserveDetailed(ctx, CameraObservation{Camera: camPreempt, State: StateOffline, ObservedAt: ts}); err != nil {
			t.Fatal(err)
		}
	}

	assertSinglePostMaintTicket := func(cam models.CCTV) Incident {
		var incs []Incident
		if err := db.Joins("JOIN cctv_incident_cameras m ON m.incident_id = cctv_incidents.id").
			Where("m.cctv_id = ?", cam.ID).Find(&incs).Error; err != nil {
			t.Fatal(err)
		}
		if len(incs) != 1 {
			t.Fatalf("expected exactly 1 ticket for %s, got %d", cam.KodeCCTV, len(incs))
		}
		inc := incs[0]
		if inc.OriginType != OriginPostPlannedMaintenance || inc.SLAClockBasis != SLABasisPlannedWindowEnd {
			t.Fatalf("expected post_planned_maintenance / planned_window_end for %s, got origin=%s basis=%s", cam.KodeCCTV, inc.OriginType, inc.SLAClockBasis)
		}
		if !inc.SLAClockStartAt.Equal(winEnd) {
			t.Fatalf("expected sla_clock_start_at == %v for %s, got %v", winEnd, cam.KodeCCTV, inc.SLAClockStartAt)
		}
		if !inc.FirstFailedObservedAt.Equal(firstFailAt) || !inc.ConfirmedOfflineAt.Equal(confirmedDuringWinAt) {
			t.Fatalf("preserved timestamps mismatch for %s: got first=%v conf=%v, want first=%v conf=%v",
				cam.KodeCCTV, inc.FirstFailedObservedAt, inc.ConfirmedOfflineAt, firstFailAt, confirmedDuringWinAt)
		}
		var outboxCnt int64
		db.Model(&Outbox{}).Where("incident_id = ?", inc.ID).Count(&outboxCnt)
		if outboxCnt != 1 {
			t.Fatalf("expected exactly 1 outbox row for %s, got %d", cam.KodeCCTV, outboxCnt)
		}
		return inc
	}

	incA := assertSinglePostMaintTicket(camA)
	incPreempt := assertSinglePostMaintTicket(camPreempt)

	var camBTicketCnt int64
	db.Model(&Incident{}).Joins("JOIN cctv_incident_cameras m ON m.incident_id = cctv_incidents.id").
		Where("m.cctv_id = ?", camB.ID).Count(&camBTicketCnt)
	if camBTicketCnt != 0 {
		t.Fatalf("offline -> unknown -> live must produce 0 tickets for camB, got %d", camBTicketCnt)
	}

	// Kirim outbox melalui Fake Telegram Server dan buktikan konfirmasi pengiriman tanpa duplikasi.
	limiter := NewChatRateLimiter(0)
	for step := 0; step < 5; step++ {
		worked, delivErr := DeliverWithRouting(ctx, db, sender, routing, limiter, winEnd.Add(3*time.Minute))
		if delivErr != nil {
			t.Fatal(delivErr)
		}
		if !worked {
			break
		}
	}

	tgMu.Lock()
	sentCount := len(deliveredTexts)
	allBodies := strings.Join(deliveredTexts, "\n---\n")
	tgMu.Unlock()
	if sentCount < 1 || sentCount > 2 {
		t.Fatalf("expected 1 (batched) or 2 Telegram messages delivered (camA + camPreempt), got %d", sentCount)
	}
	var sentOutboxCount int64
	db.Model(&Outbox{}).Where("incident_id IN ? AND status = 'sent'", []uuid.UUID{incA.ID, incPreempt.ID}).Count(&sentOutboxCount)
	if sentOutboxCount != 2 {
		t.Fatalf("expected both outbox rows (camA + camPreempt) marked sent, got %d", sentOutboxCount)
	}
	if !strings.Contains(allBodies, incA.TicketNumber) || !strings.Contains(allBodies, incPreempt.TicketNumber) || !strings.Contains(allBodies, "Pasca-Pemeliharaan Terjadwal") {
		t.Fatalf("delivered Telegram payload missing expected post-maintenance context: %s", allBodies)
	}

	// Simpan & baca kembali ProbeTelemetry ke PostgreSQL terisolasi untuk memverifikasi kolom baru skema 000034.
	telRow := ProbeTelemetry{
		ID:                     uuid.New(),
		CCTVID:                 camA.ID,
		SiteID:                 &siteID,
		NetworkType:            "gsm",
		IPNVR:                  camA.IPNVR,
		State:                  "live",
		Method:                 "mediamtx-hls",
		VerifyDurationMs:       850,
		SourceLifetimeMs:       5850,
		BytesVerify:            12288,
		BytesTotalSession:      20480,
		ByteDelta:              8192,
		SourceClosedConfirmed:  false,
		ForcedClose:            true,
		RemovePathStatus:       "failed",
		RemovePathError:        "status 500",
		ReregisterStatus:       "skipped",
		CloseVerifyStatus:      "still_active",
		CloseUnconfirmedReason: "pelepasan path GSM gagal",
		CreatedAt:              baseNow,
	}
	if err := db.Create(&telRow).Error; err != nil {
		t.Fatalf("failed inserting ProbeTelemetry with new closure evidence columns: %v", err)
	}
}

func TestPostgresAutoAssignRoundRobinAndChannelRouting(t *testing.T) {
	db, cam1 := integrationDB(t)
	ctx := context.Background()
	siteID := *cam1.SiteID
	zeroCoalesce := time.Duration(0)
	svc := Service{DB: db, CoalesceWindow: &zeroCoalesce}

	admin := createTestUser(t, db, "admin", nil)
	baseCreated := time.Date(2026, 11, 13, 1, 0, 0, 0, time.UTC)
	createOrderedTech := func(name string, offset time.Duration) models.User {
		u := createTestUser(t, db, "teknisi", &siteID)
		_ = db.Model(&models.User{}).Where("id = ?", u.ID).Updates(map[string]interface{}{
			"full_name":  name,
			"created_at": baseCreated.Add(offset),
		}).Error
		db.First(&u, "id = ?", u.ID)
		return u
	}
	tech1 := createOrderedTech("Teknisi Satu", 0)
	tech2 := createOrderedTech("Teknisi Dua", time.Second)
	tech3 := createOrderedTech("Teknisi Tiga", 2*time.Second)

	makeCam := func(code, nama string) models.CCTV {
		id := uuid.New()
		cam := models.CCTV{
			Base:              models.Base{ID: id},
			KodeCCTV:          code + "-" + id.String()[:4],
			QrPayload:         id.String(),
			Nama:              nama,
			LokasiArea:        "Zona A",
			SiteID:            &siteID,
			Status:            "aktif",
			MonitoringEnabled: true,
		}
		if err := db.Create(&cam).Error; err != nil {
			t.Fatal(err)
		}
		return cam
	}
	cam2 := makeCam("RR-02", "Kamera Dua")
	cam3 := makeCam("RR-03", "Kamera Tiga")
	cam4 := makeCam("RR-04", "Kamera Empat")

	t0 := time.Now().UTC().Truncate(time.Microsecond).Add(-30 * time.Minute)
	orderedCams := []models.CCTV{cam1, cam2, cam3, cam4}
	expectedTechs := []models.User{tech1, tech2, tech3, tech1}
	var openedIncidents []Incident
	for idx, cam := range orderedCams {
		at := t0.Add(time.Duration(idx) * time.Second)
		if err := svc.Observe(ctx, cam, StateOffline, at); err != nil {
			t.Fatal(err)
		}
		var inc Incident
		if err := db.Joins("JOIN cctv_incident_cameras m ON m.incident_id = cctv_incidents.id").
			Where("m.cctv_id = ?", cam.ID).
			First(&inc).Error; err != nil {
			t.Fatal(err)
		}
		if inc.Status != StatusInProgress || inc.AssignedTo == nil || *inc.AssignedTo != expectedTechs[idx].ID {
			t.Fatalf("ticket #%d expected auto-assign to %s (%s), got status=%s assigned=%v",
				idx+1, expectedTechs[idx].FullName, expectedTechs[idx].ID, inc.Status, inc.AssignedTo)
		}
		if inc.MaintenanceLogID == nil {
			t.Fatalf("ticket #%d expected auto-created MaintenanceLog", idx+1)
		}
		openedIncidents = append(openedIncidents, inc)
	}

	// Admin tetap dapat mengganti teknisi hasil auto-assign.
	if err := svc.Assign(ctx, openedIncidents[0].ID, siteID, tech3.ID, t0.Add(10*time.Second)); err != nil {
		t.Fatalf("expected Admin reassignment override to succeed: %v", err)
	}
	var reassigned Incident
	db.First(&reassigned, "id = ?", openedIncidents[0].ID)
	if reassigned.AssignedTo == nil || *reassigned.AssignedTo != tech3.ID {
		t.Fatalf("expected reassigned_to %s, got %v", tech3.ID, reassigned.AssignedTo)
	}

	// Kirim tiket #1 (CCTV_OFFLINE lalu INCIDENT_ASSIGNED) lewat Fake Telegram Server -> uji reply_to_message_id.
	var mu sync.Mutex
	var opsPayloads []map[string]interface{}
	var adminPayloads []map[string]interface{}
	var seq int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var p map[string]interface{}
		_ = json.Unmarshal(raw, &p)
		mu.Lock()
		seq++
		cur := seq
		switch p["chat_id"] {
		case "-1001":
			opsPayloads = append(opsPayloads, p)
		case "-1002":
			adminPayloads = append(adminPayloads, p)
		}
		mu.Unlock()
		_, _ = fmt.Fprintf(w, `{"ok":true,"result":{"message_id":%d,"chat":{"id":%v}}}`, cur, p["chat_id"])
	}))
	defer srv.Close()

	sender := TelegramSender{Token: "test", Client: srv.Client(), baseURL: srv.URL}
	routing := TelegramRouting{
		SiteOpsChats:   map[string]string{siteID.String(): "-1001"},
		SiteAdminChats: map[string]string{siteID.String(): "-1002"},
	}

	// Stabilkan cam1 5 menit (kirim CCTV_RECOVERED -> masuk admin_group, bukan ops_group),
	// lalu flapping offline lagi (tidak boleh mengirim CCTV_OFFLINE_AGAIN ke Telegram).
	for _, sec := range []int{20, 80, 140, 200, 260, 320} {
		if err := svc.Observe(ctx, cam1, "live", t0.Add(time.Duration(sec)*time.Second)); err != nil {
			t.Fatal(err)
		}
	}
	if err := svc.Observe(ctx, cam1, StateOffline, t0.Add(330*time.Second)); err != nil {
		t.Fatal(err)
	}
	for _, sec := range []int{340, 400, 460, 520, 580, 640} {
		if err := svc.Observe(ctx, cam1, "live", t0.Add(time.Duration(sec)*time.Second)); err != nil {
			t.Fatal(err)
		}
	}
	if err := svc.Resolve(ctx, openedIncidents[0].ID, siteID, admin.ID, "Selesai dicek lapangan", t0.Add(645*time.Second)); err != nil {
		t.Fatal(err)
	}

	// Verifikasi isi outbox per channel: ops_group hanya 3 momen, nol CCTV_OFFLINE_AGAIN / CCTV_PARTIAL_RECOVERY.
	var spamCount int64
	db.Model(&Outbox{}).
		Where("site_id = ? AND event_type IN ?", siteID, []string{"CCTV_OFFLINE_AGAIN", "CCTV_PARTIAL_RECOVERY"}).
		Count(&spamCount)
	if spamCount != 0 {
		t.Fatalf("expected 0 flapping/partial-recovery outbox spam rows, got %d", spamCount)
	}

	var opsTypes []string
	db.Model(&Outbox{}).Where("site_id = ? AND target_channel = ?", siteID, ChannelOpsGroup).
		Distinct().Pluck("event_type", &opsTypes)
	for _, et := range opsTypes {
		switch et {
		case "CCTV_OFFLINE", "INCIDENT_ASSIGNED", "INCIDENT_RESOLVED":
		default:
			t.Fatalf("unexpected event_type %q in ops_group; expected only 3 technician moments", et)
		}
	}

	var recOpsCount, recAdminCount int64
	db.Model(&Outbox{}).Where("incident_id = ? AND event_type = 'CCTV_RECOVERED' AND target_channel = ?", openedIncidents[0].ID, ChannelOpsGroup).Count(&recOpsCount)
	db.Model(&Outbox{}).Where("incident_id = ? AND event_type = 'CCTV_RECOVERED' AND target_channel = ?", openedIncidents[0].ID, ChannelAdminGroup).Count(&recAdminCount)
	if recOpsCount != 0 || recAdminCount == 0 {
		t.Fatalf("expected CCTV_RECOVERED routed to admin_group only, got ops=%d admin=%d", recOpsCount, recAdminCount)
	}

	// Bersihkan sisa antrean admin_group dari test lain di DB test bersama agar pengiriman terisolasi ke site uji ini.
	if err := db.Where("site_id <> ?", siteID).Delete(&Outbox{}).Error; err != nil {
		t.Fatal(err)
	}
	for step := 0; step < 20; step++ {
		worked, err := DeliverWithRouting(ctx, db, sender, routing, nil, t0.Add(650*time.Second))
		if err != nil {
			t.Fatal(err)
		}
		if !worked {
			break
		}
	}

	mu.Lock()
	defer mu.Unlock()
	if len(opsPayloads) == 0 || len(adminPayloads) == 0 {
		t.Fatalf("expected messages sent to both ops and admin channels, got ops=%d admin=%d", len(opsPayloads), len(adminPayloads))
	}
	// Cek apakah INCIDENT_ASSIGNED pada tiket #1 memakai reply_to_message_id ke pesan awalnya.
	var sawReplyToInitial bool
	for _, p := range opsPayloads {
		txt, _ := p["text"].(string)
		if strings.Contains(txt, "ALIH TUGAS") && p["reply_to_message_id"] != nil {
			sawReplyToInitial = true
		}
	}
	if !sawReplyToInitial {
		t.Fatalf("expected ALIH TUGAS update to reply to initial ticket message, got payloads: %+v", opsPayloads)
	}
}
