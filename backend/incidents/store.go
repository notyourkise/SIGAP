package incidents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"sigap-backend/models"
)

var (
	ErrNotReady          = errors.New("kamera belum pulih stabil atau bukti sudah kedaluwarsa")
	ErrCrossSite         = errors.New("operasi lintas site ditolak")
	ErrInvalidMerge      = errors.New("tiket tidak memenuhi syarat penggabungan")
	ErrInvalidEscalation = errors.New("data eskalasi penggantian perangkat belum lengkap")
)

type Service struct {
	DB             *gorm.DB
	CoalesceWindow *time.Duration
}

func (s Service) coalesceDuration(disable bool) time.Duration {
	if disable {
		return 0
	}
	if s.CoalesceWindow != nil {
		return *s.CoalesceWindow
	}
	return DefaultCoalesceWindow
}

// lockSite serializes incident mutations within a site using PostgreSQL transaction-scoped advisory locks.
func lockSite(tx *gorm.DB, site uuid.UUID) error {
	return tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", "sigap-incidents:"+site.String()).Error
}

func nextTicketNumber(tx *gorm.DB, at time.Time) (string, error) {
	wita := time.FixedZone("WITA", 8*3600)
	dateKey := at.In(wita).Format("20060102")
	if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", "sigap-ticket-seq:"+dateKey).Error; err != nil {
		return "", err
	}
	prefix := "INC-" + dateKey + "-"
	var maxExisting string
	if err := tx.Raw(
		`SELECT COALESCE(MAX(ticket_number), '') FROM cctv_incidents WHERE ticket_number LIKE ?`,
		prefix+"%",
	).Scan(&maxExisting).Error; err != nil {
		return "", err
	}
	seq := 1
	if strings.HasPrefix(maxExisting, prefix) {
		if parsed, err := strconv.Atoi(strings.TrimPrefix(maxExisting, prefix)); err == nil && parsed >= 1 {
			seq = parsed + 1
		}
	}
	return fmt.Sprintf("%s%04d", prefix, seq), nil
}

func marshalJSONString(v interface{}) string {
	if v == nil {
		return "{}"
	}
	b, err := json.Marshal(v)
	if err != nil || len(b) == 0 {
		return "{}"
	}
	return string(b)
}

func recordIncidentEvent(tx *gorm.DB, incidentID uuid.UUID, cctvID, actorID *uuid.UUID, eventType, summary string, payload map[string]interface{}, at time.Time) error {
	ev := IncidentEvent{
		ID:         uuid.New(),
		IncidentID: incidentID,
		CCTVID:     cctvID,
		ActorID:    actorID,
		EventType:  eventType,
		Summary:    summary,
		Payload:    marshalJSONString(payload),
		CreatedAt:  at,
	}
	return tx.Create(&ev).Error
}

type enqueueOpts struct {
	TargetChannel  string
	CoalesceWindow time.Duration
	SLADeadlineAt  *time.Time
}

func enqueueWithOpts(tx *gorm.DB, i *Incident, siteID uuid.UUID, episodeID *uuid.UUID, kind, body string, now time.Time, opts enqueueOpts) (*Outbox, error) {
	ch := opts.TargetChannel
	if ch == "" {
		ch = ChannelOpsGroup
	}
	var incID *uuid.UUID
	text := fmt.Sprintf("[SIGAP] %s\n%s", kind, body)
	if i != nil {
		idCopy := i.ID
		incID = &idCopy
		ref := i.TicketNumber
		if ref == "" {
			ref = i.ID.String()
		}
		text = fmt.Sprintf("[SIGAP] %s\n%s\nTiket: %s", kind, body, ref)
	}
	job := &Outbox{
		ID:               uuid.New(),
		IncidentID:       incID,
		MonitorEpisodeID: episodeID,
		SiteID:           siteID,
		TargetChannel:    ch,
		EventType:        kind,
		Text:             text,
		Status:           "pending",
		NextAttemptAt:    now,
		CoalesceUntil:    now.Add(opts.CoalesceWindow),
		SLADeadlineAt:    opts.SLADeadlineAt,
		CreatedAt:        now,
	}
	if err := tx.Create(job).Error; err != nil {
		return nil, err
	}
	return job, nil
}

// Observe preserves the simple call signature while delegating to ObserveDetailed.
func (s Service) Observe(ctx context.Context, camera models.CCTV, state string, at time.Time) error {
	return s.ObserveDetailed(ctx, CameraObservation{
		Camera:     camera,
		State:      state,
		ObservedAt: at,
	})
}

// ObserveDetailed memproses bukti verifikasi kamera dengan memisahkan:
//   - first_failed_observed_at (kegagalan pertama teramati),
//   - confirmed_offline_at (kegagalan terkonfirmasi),
//   - physical_fault_at (khusus uji terkendali, nil di produksi),
//   - serta penahanan alarm selama jendela pemeliharaan terjadwal (cctv_planned_outages).
func (s Service) ObserveDetailed(ctx context.Context, obs CameraObservation) error {
	camera := obs.Camera
	at := obs.ObservedAt
	state := obs.State

	if camera.SiteID == nil || *camera.SiteID == uuid.Nil {
		return errors.New("kamera belum memiliki site")
	}
	if state != "live" && state != StateOffline && state != StateUnknown {
		state = StateUnknown
	}

	firstFailed := at
	if obs.FirstFailedObservedAt != nil && !obs.FirstFailedObservedAt.IsZero() && !obs.FirstFailedObservedAt.After(at) {
		firstFailed = *obs.FirstFailedObservedAt
	}

	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockSite(tx, *camera.SiteID); err != nil {
			return err
		}

		var member Member
		memberErr := tx.Where("cctv_id = ? AND is_active_member = true", camera.ID).First(&member).Error
		if memberErr != nil && !errors.Is(memberErr, gorm.ErrRecordNotFound) {
			return memberErr
		}

		// Cek apakah kamera sedang berada dalam jendela penghentian terencana (Keputusan #1).
		suppressed, err := recordPlannedOutageObservationTx(tx, camera.ID, state, firstFailed, at)
		if err != nil {
			return err
		}

		if errors.Is(memberErr, gorm.ErrRecordNotFound) {
			// Selama jendela penghentian terencana aktif, tahan pembukaan tiket & Telegram,
			// namun waktu observasi kegagalan tetap tersimpan di cctv_planned_outage_cameras.
			if suppressed {
				return nil
			}
			// Jangan biarkan jalur monitoring biasa mendahului evaluasi pasca-maintenance yang masih tertunda
			// (mis. akibat sampel `unknown`) dengan membuat tiket gangguan di luar jadwal (`unplanned_outage`).
			handledPending, err := s.resolvePendingPostMaintenanceObservationTx(tx, obs, state, at)
			if err != nil {
				return err
			}
			if handledPending || state != StateOffline {
				return nil
			}
			return s.openSingleIncidentTx(tx, obs, firstFailed, at)
		}

		if member.LastObservedAt != nil && !at.After(*member.LastObservedAt) {
			return nil
		}

		var i Incident
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&i, "id = ? AND site_id = ?", member.IncidentID, *camera.SiteID).Error; err != nil {
			return err
		}

		old := member.ConnectionState
		recovered := Advance(&member, state, at)
		if obs.Evidence != nil {
			member.LastEvidence = marshalJSONString(obs.Evidence)
		}
		if err := tx.Save(&member).Error; err != nil {
			return err
		}

		allReady, totalActive, recoveredCount, remainingNames, err := evaluateIncidentActiveMembersTx(tx, i.ID, at)
		if err != nil {
			return err
		}

		if allReady && totalActive > 0 {
			i.Status = StatusPendingVerification
		} else if i.AssignedTo != nil {
			i.Status = StatusInProgress
		} else {
			i.Status = StatusOpen
		}
		i.UpdatedAt = at
		if err := tx.Omit("Members", "Events").Save(&i).Error; err != nil {
			return err
		}

		if recovered {
			_ = recordIncidentEvent(tx, i.ID, &camera.ID, nil, "MEMBER_RECOVERED",
				fmt.Sprintf("Kamera %s stabil online 5 menit (%d sampel berturut-turut)", camera.Nama, member.ConsecutiveLiveCount),
				map[string]interface{}{"recovered_count": recoveredCount, "total_active": totalActive}, at)

			if allReady {
				_, err := enqueueWithOpts(tx, &i, i.SiteID, nil, "CCTV_RECOVERED",
					fmt.Sprintf("%s stabil online selama 5 menit berdasarkan pemeriksaan berkala (%d/%d kamera aktif pulih). Menunggu verifikasi Admin; tiket belum ditutup.",
						camera.Nama, recoveredCount, totalActive),
					at, enqueueOpts{TargetChannel: ChannelOpsGroup})
				return err
			}

			// Pemulihan sebagian pada tiket bersama: debounce 30 detik agar grup tidak banjir.
			var recentPartial int64
			_ = tx.Model(&Outbox{}).
				Where("incident_id = ? AND event_type = 'CCTV_PARTIAL_RECOVERY' AND created_at >= ?", i.ID, at.Add(-PartialRecoveryDebounceGap)).
				Count(&recentPartial).Error
			if recentPartial == 0 {
				_, err := enqueueWithOpts(tx, &i, i.SiteID, nil, "CCTV_PARTIAL_RECOVERY",
					fmt.Sprintf("Pemulihan sebagian (%d/%d kamera pulih; baru pulih: %s). Masih menunggu pemulihan: %s. Tiket tetap terbuka.",
						recoveredCount, totalActive, camera.Nama, strings.Join(remainingNames, ", ")),
					at, enqueueOpts{TargetChannel: ChannelOpsGroup})
				return err
			}
			return nil
		}

		if state == StateOffline && old != StateOffline {
			_ = recordIncidentEvent(tx, i.ID, &camera.ID, nil, "MEMBER_OFFLINE_AGAIN",
				fmt.Sprintf("Kamera %s kembali terkonfirmasi offline; hitungan pemulihan dimulai ulang", camera.Nama), nil, at)
			_, err := enqueueWithOpts(tx, &i, i.SiteID, nil, "CCTV_OFFLINE_AGAIN",
				camera.Nama+" kembali offline. Tiket yang sama tetap digunakan.",
				at, enqueueOpts{TargetChannel: ChannelOpsGroup})
			return err
		}
		return nil
	})
}

func (s Service) openSingleIncidentTx(tx *gorm.DB, obs CameraObservation, firstFailed, confirmedAt time.Time) error {
	camera := obs.Camera
	ticketNo, err := nextTicketNumber(tx, confirmedAt)
	if err != nil {
		return err
	}

	origin := obs.OriginType
	if origin != OriginPostPlannedMaintenance {
		origin = OriginUnplannedOutage
	}
	slaStart := firstFailed
	slaBasis := SLABasisFirstFailedObservation
	if origin == OriginPostPlannedMaintenance && obs.SLAClockStartOverride != nil {
		slaStart = *obs.SLAClockStartOverride
		slaBasis = SLABasisPlannedWindowEnd
	}
	slaDeadline := slaStart.Add(SLATargetDuration)

	title := "CCTV tidak dapat diakses: " + camera.Nama
	if origin == OriginPostPlannedMaintenance {
		title = "CCTV masih offline pasca-pemeliharaan terjadwal: " + camera.Nama
	}

	i := Incident{
		ID:                    uuid.New(),
		TicketNumber:          ticketNo,
		SiteID:                *camera.SiteID,
		Status:                StatusOpen,
		ScopeType:             ScopeSingleCamera,
		OriginType:            origin,
		Title:                 title,
		FirstFailedObservedAt: firstFailed,
		ConfirmedOfflineAt:    confirmedAt,
		PhysicalFaultAt:       obs.PhysicalFaultAt,
		SLAClockStartAt:       slaStart,
		SLAClockBasis:         slaBasis,
		TimestampSource:       TimestampSourceLiveObservation,
		OpenedAt:              confirmedAt,
		UpdatedAt:             confirmedAt,
	}
	if err := tx.Create(&i).Error; err != nil {
		return err
	}

	member := Member{
		ID:                    uuid.New(),
		IncidentID:            i.ID,
		CCTVID:                camera.ID,
		CameraCode:            camera.KodeCCTV,
		CameraName:            camera.Nama,
		LocationArea:          camera.LokasiArea,
		IsActiveMember:        true,
		ConnectionState:       StateOffline,
		FirstFailedObservedAt: &firstFailed,
		ConfirmedOfflineAt:    &confirmedAt,
		LastObservedAt:        &confirmedAt,
		MemberOutcome:         OutcomePending,
		LastEvidence:          marshalJSONString(obs.Evidence),
	}
	if err := tx.Create(&member).Error; err != nil {
		return err
	}

	_ = recordIncidentEvent(tx, i.ID, &camera.ID, nil, "INCIDENT_OPENED",
		fmt.Sprintf("Tiket %s dibuka (%s)", ticketNo, origin),
		map[string]interface{}{
			"first_failed_observed_at": firstFailed,
			"confirmed_offline_at":     confirmedAt,
			"sla_clock_start_at":       slaStart,
			"sla_clock_basis":          slaBasis,
		}, confirmedAt)

	body := fmt.Sprintf("%s - %s\nLokasi: %s\nGagal teramati pertama: %s WITA\nOffline terkonfirmasi: %s WITA\nMenunggu penugasan Admin melalui website SIGAP.",
		camera.KodeCCTV, camera.Nama, camera.LokasiArea,
		firstFailed.In(time.FixedZone("WITA", 8*3600)).Format("15:04:05"),
		confirmedAt.In(time.FixedZone("WITA", 8*3600)).Format("15:04:05"))
	if origin == OriginPostPlannedMaintenance {
		body = fmt.Sprintf("[Pasca-Pemeliharaan Terjadwal]\n%s", body)
	}

	coalesce := s.coalesceDuration(obs.DisableBatching)
	_, err = enqueueWithOpts(tx, &i, i.SiteID, nil, "CCTV_OFFLINE", body, confirmedAt, enqueueOpts{
		TargetChannel:  ChannelOpsGroup,
		CoalesceWindow: coalesce,
		SLADeadlineAt:  &slaDeadline,
	})
	return err
}

// ObserveSharedCorrelatedWave membuat atau memperbarui 1 tiket bersama (shared_nvr)
// hanya apabila pemantau sehat, >=2 kamera pada site & ip_nvr yang sama gagal terkonfirmasi,
// dan bukti ProbeRTSPHost membuktikan NVR tidak merespons (nvrUnreachable == true).
// Jika bukti penyebab bersama belum terpenuhi, otomatis jatuh ke tiket per kamera.
func (s Service) ObserveSharedCorrelatedWave(ctx context.Context, siteID uuid.UUID, ipNVR string, nvrUnreachable bool, observations []CameraObservation) error {
	ipNVR = strings.TrimSpace(ipNVR)
	if !nvrUnreachable || ipNVR == "" || len(observations) < 2 {
		for _, obs := range observations {
			if err := s.ObserveDetailed(ctx, obs); err != nil {
				return err
			}
		}
		return nil
	}

	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockSite(tx, siteID); err != nil {
			return err
		}

		// Pisahkan kamera yang belum memiliki tiket aktif dan tidak sedang dalam planned outage.
		eligible := make([]CameraObservation, 0, len(observations))
		for _, obs := range observations {
			if obs.Camera.SiteID == nil || *obs.Camera.SiteID != siteID || obs.State != StateOffline {
				continue
			}
			firstFailed := obs.ObservedAt
			if obs.FirstFailedObservedAt != nil && !obs.FirstFailedObservedAt.IsZero() {
				firstFailed = *obs.FirstFailedObservedAt
			}
			suppressed, err := recordPlannedOutageObservationTx(tx, obs.Camera.ID, StateOffline, firstFailed, obs.ObservedAt)
			if err != nil {
				return err
			}
			if suppressed {
				continue
			}
			handledPending, err := s.resolvePendingPostMaintenanceObservationTx(tx, obs, StateOffline, obs.ObservedAt)
			if err != nil {
				return err
			}
			if handledPending {
				continue
			}
			var existing Member
			err = tx.Where("cctv_id = ? AND is_active_member = true", obs.Camera.ID).First(&existing).Error
			if err == nil {
				// Kamera sudah terikat tiket aktif -> perbarui observasi pada tiketnya.
				old := existing.ConnectionState
				Advance(&existing, StateOffline, obs.ObservedAt)
				_ = tx.Save(&existing).Error
				if old != StateOffline {
					var existingInc Incident
					if tx.First(&existingInc, "id = ?", existing.IncidentID).Error == nil {
						if existingInc.AssignedTo != nil {
							existingInc.Status = StatusInProgress
						} else {
							existingInc.Status = StatusOpen
						}
						existingInc.UpdatedAt = obs.ObservedAt
						_ = tx.Omit("Members", "Events").Save(&existingInc).Error
					}
				}
				continue
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			eligible = append(eligible, obs)
		}

		if len(eligible) == 0 {
			return nil
		}

		corrKey := "nvr:" + ipNVR
		var shared Incident
		findErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("site_id = ? AND correlation_key = ? AND status NOT IN ('resolved', 'merged')", siteID, corrKey).
			First(&shared).Error

		if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return findErr
		}

		// Bila belum ada tiket bersama aktif dan kamera baru yang eligible hanya 1, jangan paksakan shared_nvr baru.
		if errors.Is(findErr, gorm.ErrRecordNotFound) && len(eligible) < 2 {
			firstFailed := eligible[0].ObservedAt
			if eligible[0].FirstFailedObservedAt != nil && !eligible[0].FirstFailedObservedAt.IsZero() {
				firstFailed = *eligible[0].FirstFailedObservedAt
			}
			return s.openSingleIncidentTx(tx, eligible[0], firstFailed, eligible[0].ObservedAt)
		}

		earliestFirstFailed := eligible[0].ObservedAt
		latestConfirmed := eligible[0].ObservedAt
		for _, obs := range eligible {
			ff := obs.ObservedAt
			if obs.FirstFailedObservedAt != nil && !obs.FirstFailedObservedAt.IsZero() && obs.FirstFailedObservedAt.Before(ff) {
				ff = *obs.FirstFailedObservedAt
			}
			if ff.Before(earliestFirstFailed) {
				earliestFirstFailed = ff
			}
			if obs.ObservedAt.After(latestConfirmed) {
				latestConfirmed = obs.ObservedAt
			}
		}

		isNewIncident := errors.Is(findErr, gorm.ErrRecordNotFound)
		if isNewIncident {
			ticketNo, err := nextTicketNumber(tx, latestConfirmed)
			if err != nil {
				return err
			}
			shared = Incident{
				ID:                    uuid.New(),
				TicketNumber:          ticketNo,
				SiteID:                siteID,
				Status:                StatusOpen,
				ScopeType:             ScopeSharedNVR,
				OriginType:            OriginUnplannedOutage,
				CorrelationKey:        corrKey,
				Title:                 fmt.Sprintf("Gangguan Bersama NVR %s (%d CCTV terdampak)", ipNVR, len(eligible)),
				FirstFailedObservedAt: earliestFirstFailed,
				ConfirmedOfflineAt:    latestConfirmed,
				SLAClockStartAt:       earliestFirstFailed,
				SLAClockBasis:         SLABasisFirstFailedObservation,
				TimestampSource:       TimestampSourceLiveObservation,
				OpenedAt:              latestConfirmed,
				UpdatedAt:             latestConfirmed,
			}
			if err := tx.Create(&shared).Error; err != nil {
				return err
			}
		}

		names := make([]string, 0, len(eligible))
		for _, obs := range eligible {
			ff := obs.ObservedAt
			if obs.FirstFailedObservedAt != nil && !obs.FirstFailedObservedAt.IsZero() {
				ff = *obs.FirstFailedObservedAt
			}
			conf := obs.ObservedAt
			m := Member{
				ID:                    uuid.New(),
				IncidentID:            shared.ID,
				CCTVID:                obs.Camera.ID,
				CameraCode:            obs.Camera.KodeCCTV,
				CameraName:            obs.Camera.Nama,
				LocationArea:          obs.Camera.LokasiArea,
				IsActiveMember:        true,
				ConnectionState:       StateOffline,
				FirstFailedObservedAt: &ff,
				ConfirmedOfflineAt:    &conf,
				LastObservedAt:        &conf,
				MemberOutcome:         OutcomePending,
				LastEvidence:          marshalJSONString(obs.Evidence),
			}
			if err := tx.Create(&m).Error; err != nil {
				return err
			}
			names = append(names, obs.Camera.KodeCCTV+" ("+obs.Camera.Nama+")")
		}

		if !isNewIncident {
			var activeCount int64
			_ = tx.Model(&Member{}).Where("incident_id = ? AND is_active_member = true", shared.ID).Count(&activeCount).Error
			shared.Title = fmt.Sprintf("Gangguan Bersama NVR %s (%d CCTV terdampak)", ipNVR, activeCount)
			if shared.AssignedTo != nil {
				shared.Status = StatusInProgress
			} else {
				shared.Status = StatusOpen
			}
			shared.UpdatedAt = latestConfirmed
			if err := tx.Omit("Members", "Events").Save(&shared).Error; err != nil {
				return err
			}
		}

		_ = recordIncidentEvent(tx, shared.ID, nil, nil, "SHARED_CORRELATED_OFFLINE",
			fmt.Sprintf("Korelasi gangguan bersama NVR %s: %d kamera terkonfirmasi offline", ipNVR, len(eligible)),
			map[string]interface{}{"ip_nvr": ipNVR, "cameras": names}, latestConfirmed)

		slaDeadline := shared.SLAClockStartAt.Add(SLATargetDuration)
		var deadlinePtr *time.Time
		if isNewIncident {
			deadlinePtr = &slaDeadline
		}
		_, err := enqueueWithOpts(tx, &shared, siteID, nil, "CCTV_OFFLINE",
			fmt.Sprintf("Bukti gangguan bersama pada NVR %s (host tidak merespons).\nKamera terdampak (%d): %s\nMenunggu penugasan Admin.",
				ipNVR, len(names), strings.Join(names, ", ")),
			latestConfirmed, enqueueOpts{
				TargetChannel:  ChannelOpsGroup,
				CoalesceWindow: s.coalesceDuration(false),
				SLADeadlineAt:  deadlinePtr,
			})
		return err
	})
}

func evaluateIncidentActiveMembersTx(tx *gorm.DB, incidentID uuid.UUID, refTime time.Time) (allReady bool, totalActive int, recoveredCount int, remainingNames []string, err error) {
	var members []Member
	if err = tx.Where("incident_id = ? AND is_active_member = true", incidentID).Order("camera_name ASC").Find(&members).Error; err != nil {
		return false, 0, 0, nil, err
	}
	totalActive = len(members)
	if totalActive == 0 {
		return false, 0, 0, nil, nil
	}
	allReady = true
	for _, m := range members {
		freshRecovered := m.ConnectionState == StateRecovered &&
			m.LastObservedAt != nil &&
			!m.LastObservedAt.After(refTime) &&
			refTime.Sub(*m.LastObservedAt) <= MaxEvidenceGap
		if freshRecovered {
			recoveredCount++
		} else {
			allReady = false
			remainingNames = append(remainingNames, m.CameraName)
		}
	}
	return allReady, totalActive, recoveredCount, remainingNames, nil
}

// ResetObservations invalidates recovery evidence after a monitoring restart.
// Fixes prototype bug: unassigned pending_verification incidents return to 'open'
// (not 'in_progress'), and expired 'sending' outbox leases transition to 'ambiguous'.
func (s Service) ResetObservations(ctx context.Context) error {
	now := time.Now().UTC()
	var sites []uuid.UUID
	if err := s.DB.WithContext(ctx).Model(&Incident{}).
		Where("status NOT IN ('resolved', 'merged')").
		Distinct().Pluck("site_id", &sites).Error; err != nil {
		return err
	}
	for _, site := range sites {
		if err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := lockSite(tx, site); err != nil {
				return err
			}
			ids := tx.Model(&Incident{}).Select("id").Where("site_id = ? AND status NOT IN ('resolved', 'merged')", site)
			if err := tx.Model(&Member{}).
				Where("is_active_member = true AND incident_id IN (?)", ids).
				Updates(map[string]interface{}{
					"connection_state":       StateUnknown,
					"recovery_started_at":    nil,
					"recovered_at":           nil,
					"consecutive_live_count": 0,
				}).Error; err != nil {
				return err
			}
			if err := tx.Model(&Incident{}).
				Where("site_id = ? AND status = ? AND assigned_to IS NOT NULL", site, StatusPendingVerification).
				Updates(map[string]interface{}{"status": StatusInProgress, "updated_at": now}).Error; err != nil {
				return err
			}
			if err := tx.Model(&Incident{}).
				Where("site_id = ? AND status = ? AND assigned_to IS NULL", site, StatusPendingVerification).
				Updates(map[string]interface{}{"status": StatusOpen, "updated_at": now}).Error; err != nil {
				return err
			}
			return nil
		}); err != nil {
			return err
		}
	}

	// Baris outbox yang sedang 'sending' saat proses crash dan lease-nya sudah habis
	// dipindahkan ke 'ambiguous' dengan jeda retry pertama 10 detik.
	return s.DB.WithContext(ctx).Model(&Outbox{}).
		Where("status = 'sending' AND lease_until IS NOT NULL AND lease_until < ?", now).
		Updates(map[string]interface{}{
			"status":          "ambiguous",
			"lease_until":     nil,
			"next_attempt_at": now.Add(AmbiguousRetryFirstDelay),
			"last_error":      "Telegram: hasil pengiriman belum dapat dipastikan karena proses worker terhenti sebelum konfirmasi diterima",
		}).Error
}
