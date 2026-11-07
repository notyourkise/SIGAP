package incidents

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"sigap-backend/models"
)

const PostMaintenanceFreshnessWindow = 75 * time.Second

var postMaintenanceUnknownRetrySteps = []time.Duration{
	15 * time.Second,
	30 * time.Second,
	60 * time.Second,
}

// PostMaintenanceUnknownRetryDelay menghitung jeda retry terkendali ketika evaluasi
// pasca-pemeliharaan menghasilkan status ambigu (unknown) sehingga tidak melakukan tight-loop
// dan tidak menandai evaluasi selesai sebelum ada bukti konklusif (live atau offline).
func PostMaintenanceUnknownRetryDelay(retryCount int) time.Duration {
	if retryCount < 0 {
		retryCount = 0
	}
	if retryCount >= len(postMaintenanceUnknownRetrySteps) {
		return postMaintenanceUnknownRetrySteps[len(postMaintenanceUnknownRetrySteps)-1]
	}
	return postMaintenanceUnknownRetrySteps[retryCount]
}

// PlannedOutageRefreshRequest dikembalikan untuk kamera yang sampel terakhirnya selama
// jendela pemeliharaan berusia > 75 detik sehingga perlu 1 pemeriksaan penyegaran prioritas
// melalui Unified NVR Session Gate sebelum tiket pasca-maintenance dibuka.
type PlannedOutageRefreshRequest struct {
	PlannedOutageCameraID        uuid.UUID
	Camera                       models.CCTV
	EffectiveEndsAt              time.Time
	SuppressedFirstFailedAt      time.Time
	SuppressedConfirmedOfflineAt time.Time
}

type ActivePlannedWindowInfo struct {
	OutageID uuid.UUID
	Reason   string
	EndsAt   time.Time
}

// recordPlannedOutageObservationTx mencatat hasil pemeriksaan selama kamera berada
// pada jendela pemeliharaan terjadwal aktif tanpa membuka tiket gangguan baru.
func recordPlannedOutageObservationTx(tx *gorm.DB, cctvID uuid.UUID, state string, firstFailed, at time.Time) (bool, error) {
	type activeLink struct {
		ID                           uuid.UUID
		SuppressedFirstFailedAt      *time.Time
		SuppressedConfirmedOfflineAt *time.Time
	}
	var links []activeLink
	err := tx.Table("cctv_planned_outage_cameras poc").
		Select("poc.id, poc.suppressed_first_failed_at, poc.suppressed_confirmed_offline_at").
		Joins("JOIN cctv_planned_outages po ON po.id = poc.planned_outage_id").
		Where("poc.cctv_id = ? AND po.starts_at <= ? AND COALESCE(po.ended_early_at, po.ends_at) > ?", cctvID, at, at).
		Scan(&links).Error
	if err != nil {
		return false, err
	}
	if len(links) == 0 {
		return false, nil
	}

	for _, l := range links {
		updates := map[string]interface{}{
			"last_verified_state": state,
			"last_verified_at":    at,
		}
		switch state {
		case StateOffline:
			if l.SuppressedFirstFailedAt == nil {
				updates["suppressed_first_failed_at"] = firstFailed
			}
			if l.SuppressedConfirmedOfflineAt == nil {
				updates["suppressed_confirmed_offline_at"] = at
			}
		case "live":
			updates["suppressed_first_failed_at"] = gorm.Expr("NULL")
			updates["suppressed_confirmed_offline_at"] = gorm.Expr("NULL")
		}
		if err := tx.Model(&PlannedOutageCamera{}).Where("id = ?", l.ID).Updates(updates).Error; err != nil {
			return false, err
		}
	}
	return true, nil
}

// CreatePlannedOutage membuat jendela penghentian kamera yang direncanakan Admin (Keputusan #1).
func (s Service) CreatePlannedOutage(ctx context.Context, site, admin uuid.UUID, reason string, startsAt, endsAt time.Time, cctvIDs []uuid.UUID, now time.Time) (*PlannedOutage, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, errors.New("alasan penghentian kamera wajib diisi")
	}
	if !endsAt.After(startsAt) {
		return nil, errors.New("waktu berakhir harus lebih besar dari waktu mulai")
	}
	uniqueCCTVs := make([]uuid.UUID, 0, len(cctvIDs))
	seen := map[uuid.UUID]bool{}
	for _, id := range cctvIDs {
		if id == uuid.Nil || seen[id] {
			continue
		}
		seen[id] = true
		uniqueCCTVs = append(uniqueCCTVs, id)
	}
	if len(uniqueCCTVs) == 0 {
		return nil, errors.New("pilih minimal satu kamera yang akan dimatikan terjadwal")
	}

	var created PlannedOutage
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var validCount int64
		if err := tx.Model(&models.CCTV{}).
			Where("id IN ? AND site_id = ?", uniqueCCTVs, site).
			Count(&validCount).Error; err != nil {
			return err
		}
		if int(validCount) != len(uniqueCCTVs) {
			return ErrCrossSite
		}

		created = PlannedOutage{
			ID:        uuid.New(),
			SiteID:    site,
			Reason:    reason,
			StartsAt:  startsAt,
			EndsAt:    endsAt,
			CreatedBy: admin,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := tx.Create(&created).Error; err != nil {
			return err
		}

		cams := make([]PlannedOutageCamera, 0, len(uniqueCCTVs))
		for _, cid := range uniqueCCTVs {
			poc := PlannedOutageCamera{
				ID:                uuid.New(),
				PlannedOutageID:   created.ID,
				CCTVID:            cid,
				LastVerifiedState: StateUnknown,
			}
			if err := tx.Create(&poc).Error; err != nil {
				return err
			}
			cams = append(cams, poc)
		}
		created.Cameras = cams
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &created, nil
}

// EndPlannedOutageEarly mengakhiri jendela pemeliharaan terjadwal lebih awal.
func (s Service) EndPlannedOutageEarly(ctx context.Context, outageID, site, admin uuid.UUID, now time.Time) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var po PlannedOutage
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&po, "id = ? AND site_id = ?", outageID, site).Error; err != nil {
			return err
		}
		if po.EndedEarlyAt != nil || !now.Before(po.EndsAt) {
			return errors.New("jendela pemeliharaan terjadwal sudah berakhir")
		}
		if now.Before(po.StartsAt) {
			// Bila dihentikan sebelum mulai, tandai langsung diproses agar tidak memicu evaluasi pasca-jendela.
			if err := tx.Model(&PlannedOutageCamera{}).
				Where("planned_outage_id = ?", po.ID).
				Update("post_window_processed_at", now).Error; err != nil {
				return err
			}
			po.StartsAt = now.Add(-time.Millisecond)
		}
		po.EndedEarlyAt = &now
		po.EndedEarlyBy = &admin
		po.UpdatedAt = now
		return tx.Omit("Cameras").Save(&po).Error
	})
}

func (s Service) ListPlannedOutages(ctx context.Context, site uuid.UUID) ([]PlannedOutage, error) {
	var list []PlannedOutage
	err := s.DB.WithContext(ctx).
		Preload("Cameras").
		Where("site_id = ?", site).
		Order("starts_at DESC").
		Limit(50).
		Find(&list).Error
	return list, err
}

// ActivePlannedOutageMap mengembalikan peta cctv_id -> info jendela aktif pada saat 'at'.
func (s Service) ActivePlannedOutageMap(ctx context.Context, cctvIDs []uuid.UUID, at time.Time) (map[uuid.UUID]ActivePlannedWindowInfo, error) {
	out := make(map[uuid.UUID]ActivePlannedWindowInfo)
	if s.DB == nil || len(cctvIDs) == 0 {
		return out, nil
	}
	type row struct {
		CCTVID       uuid.UUID
		OutageID     uuid.UUID
		Reason       string
		EffectiveEnd time.Time
	}
	var rows []row
	err := s.DB.WithContext(ctx).
		Table("cctv_planned_outage_cameras poc").
		Select("poc.cctv_id, po.id AS outage_id, po.reason, COALESCE(po.ended_early_at, po.ends_at) AS effective_end").
		Joins("JOIN cctv_planned_outages po ON po.id = poc.planned_outage_id").
		Where("poc.cctv_id IN ? AND po.starts_at <= ? AND COALESCE(po.ended_early_at, po.ends_at) > ?", cctvIDs, at, at).
		Order("effective_end DESC").
		Scan(&rows).Error
	if err != nil {
		return out, err
	}
	for _, r := range rows {
		if _, exists := out[r.CCTVID]; !exists {
			out[r.CCTVID] = ActivePlannedWindowInfo{
				OutageID: r.OutageID,
				Reason:   r.Reason,
				EndsAt:   r.EffectiveEnd,
			}
		}
	}
	return out, nil
}

// ProcessExpiredPlannedOutages memproses seluruh jendela pemeliharaan yang sudah berakhir:
//   - Jika kamera tidak pernah terkonfirmasi offline selama jendela (atau sudah terbukti live), tandai selesai tanpa tiket.
//   - Jika kamera pernah terkonfirmasi offline selama jendela namun hasil terakhir masih `unknown`, evaluasi TETAP TERTUNDA
//     (`post_window_processed_at IS NULL`) dan dijadwalkan pemeriksaan ulang dengan jeda terkendali (`post_window_next_retry_at`).
//   - Jika kamera masih memegang bukti offline terkonfirmasi yang segar (<= 75s), langsung membuka tiket
//     dan mengantrekan Telegram TANPA mengulang 2 pemeriksaan dari awal, dengan sla_clock_start_at = effective_ends_at
//     sementara first_failed_observed_at dan confirmed_offline_at asli selama jendela tetap dicatat terpisah.
//   - Jika bukti offline terkonfirmasi sudah berusia > 75s, mengembalikan PlannedOutageRefreshRequest agar
//     Unified NVR Session Gate menjalankan 1 pemeriksaan penyegaran cepat dengan backoff terkendali.
func (s Service) ProcessExpiredPlannedOutages(ctx context.Context, now time.Time) (openedCount int, needRefresh []PlannedOutageRefreshRequest, err error) {
	type expiredRow struct {
		PocID                        uuid.UUID
		OutageID                     uuid.UUID
		SiteID                       uuid.UUID
		CCTVID                       uuid.UUID
		EffectiveEnd                 time.Time
		SuppressedFirstFailedAt      *time.Time
		SuppressedConfirmedOfflineAt *time.Time
		LastVerifiedState            string
		LastVerifiedAt               *time.Time
		PostWindowRetryCount         int
		PostWindowNextRetryAt        *time.Time
	}

	var rows []expiredRow
	err = s.DB.WithContext(ctx).
		Table("cctv_planned_outage_cameras poc").
		Select(`
			poc.id AS poc_id,
			po.id AS outage_id,
			po.site_id,
			poc.cctv_id,
			COALESCE(po.ended_early_at, po.ends_at) AS effective_end,
			poc.suppressed_first_failed_at,
			poc.suppressed_confirmed_offline_at,
			poc.last_verified_state,
			poc.last_verified_at,
			poc.post_window_retry_count,
			poc.post_window_next_retry_at
		`).
		Joins("JOIN cctv_planned_outages po ON po.id = poc.planned_outage_id").
		Where("poc.post_window_processed_at IS NULL AND COALESCE(po.ended_early_at, po.ends_at) <= ?", now).
		Order("effective_end ASC").
		Scan(&rows).Error
	if err != nil {
		return 0, nil, err
	}

	for _, r := range rows {
		var camera models.CCTV
		if err := s.DB.WithContext(ctx).First(&camera, "id = ?", r.CCTVID).Error; err != nil {
			_ = s.DB.WithContext(ctx).Model(&PlannedOutageCamera{}).Where("id = ?", r.PocID).Update("post_window_processed_at", now).Error
			continue
		}

		// Jika ada jendela aktif lain yang masih menaungi kamera ini, tandai baris lama selesai diproses.
		var stillActive int64
		_ = s.DB.WithContext(ctx).
			Table("cctv_planned_outage_cameras poc").
			Joins("JOIN cctv_planned_outages po ON po.id = poc.planned_outage_id").
			Where("poc.cctv_id = ? AND po.starts_at <= ? AND COALESCE(po.ended_early_at, po.ends_at) > ?", r.CCTVID, now, now).
			Count(&stillActive).Error
		if stillActive > 0 || normalizeStatusLower(camera.Status) == "nonaktif" || !camera.MonitoringEnabled {
			_ = s.DB.WithContext(ctx).Model(&PlannedOutageCamera{}).Where("id = ?", r.PocID).Updates(map[string]interface{}{
				"post_window_processed_at":  now,
				"post_window_next_retry_at": gorm.Expr("NULL"),
			}).Error
			continue
		}

		// Kamera tidak pernah terkonfirmasi offline selama jendela, atau sudah terbukti live -> selesai tanpa tiket.
		if r.SuppressedConfirmedOfflineAt == nil || r.LastVerifiedState == "live" {
			_ = s.DB.WithContext(ctx).Model(&PlannedOutageCamera{}).Where("id = ?", r.PocID).Updates(map[string]interface{}{
				"post_window_processed_at":  now,
				"post_window_next_retry_at": gorm.Expr("NULL"),
			}).Error
			continue
		}

		ff := *r.SuppressedConfirmedOfflineAt
		if r.SuppressedFirstFailedAt != nil && !r.SuppressedFirstFailedAt.IsZero() {
			ff = *r.SuppressedFirstFailedAt
		}
		conf := *r.SuppressedConfirmedOfflineAt

		// Bukti terakhir adalah offline yang masih segar (<= 75s) -> buka tiket pasca-maintenance sekarang.
		if r.LastVerifiedState == StateOffline && r.LastVerifiedAt != nil && now.Sub(*r.LastVerifiedAt) <= PostMaintenanceFreshnessWindow {
			opened, txErr := s.openPostMaintenanceIncidentAtomically(ctx, r.PocID, camera, ff, conf, r.EffectiveEnd, now)
			if txErr != nil {
				return openedCount, needRefresh, fmt.Errorf("gagal memproses akhir jendela pemeliharaan kamera %s: %w", camera.KodeCCTV, txErr)
			}
			if opened {
				openedCount++
			}
			continue
		}

		// Sampel terakhir `unknown` atau sudah kedaluwarsa (> 75s): pertahankan status tertunda
		// dan jadwalkan pemeriksaan ulang dengan jeda retry terkendali.
		if r.PostWindowNextRetryAt != nil && now.Before(*r.PostWindowNextRetryAt) {
			continue
		}

		nextRetry := now.Add(PostMaintenanceUnknownRetryDelay(r.PostWindowRetryCount))
		res := s.DB.WithContext(ctx).Model(&PlannedOutageCamera{}).
			Where("id = ? AND post_window_processed_at IS NULL", r.PocID).
			Update("post_window_next_retry_at", nextRetry)
		if res.Error != nil {
			return openedCount, needRefresh, res.Error
		}
		if res.RowsAffected == 0 {
			continue
		}

		needRefresh = append(needRefresh, PlannedOutageRefreshRequest{
			PlannedOutageCameraID:        r.PocID,
			Camera:                       camera,
			EffectiveEndsAt:              r.EffectiveEnd,
			SuppressedFirstFailedAt:      ff,
			SuppressedConfirmedOfflineAt: conf,
		})
	}

	return openedCount, needRefresh, nil
}

// CompletePostMaintenanceRefresh menyelesaikan hasil pemeriksaan penyegaran pasca-maintenance:
//   - `live`    : menandai evaluasi selesai tanpa membuat tiket.
//   - `offline` : membuka tepat satu tiket `post_planned_maintenance` dengan SLA dimulai pada `effective_ends_at`.
//   - `unknown` : TIDAK menyelesaikan evaluasi (`post_window_processed_at` tetap NULL), menaikkan hitungan retry,
//                 dan menjadwalkan pemeriksaan ulang berikutnya sesuai backoff terkendali.
func (s Service) CompletePostMaintenanceRefresh(ctx context.Context, req PlannedOutageRefreshRequest, refreshedState string, refreshedAt time.Time) (bool, error) {
	switch refreshedState {
	case "live":
		err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if req.Camera.SiteID != nil && *req.Camera.SiteID != uuid.Nil {
				if err := lockSite(tx, *req.Camera.SiteID); err != nil {
					return err
				}
			}
			return finalizePendingPostMaintenanceLiveTx(tx, req.Camera.ID, req.PlannedOutageCameraID, refreshedAt)
		})
		return false, err

	case StateOffline:
		return s.openPostMaintenanceIncidentAtomically(ctx, req.PlannedOutageCameraID, req.Camera, req.SuppressedFirstFailedAt, req.SuppressedConfirmedOfflineAt, req.EffectiveEndsAt, refreshedAt)

	default:
		err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var poc PlannedOutageCamera
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				First(&poc, "id = ? AND post_window_processed_at IS NULL", req.PlannedOutageCameraID).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil
				}
				return err
			}
			nextCount := poc.PostWindowRetryCount + 1
			nextRetry := refreshedAt.Add(PostMaintenanceUnknownRetryDelay(poc.PostWindowRetryCount))
			return tx.Model(&PlannedOutageCamera{}).
				Where("id = ? AND post_window_processed_at IS NULL", poc.ID).
				Updates(map[string]interface{}{
					"last_verified_state":       StateUnknown,
					"last_verified_at":          refreshedAt,
					"post_window_retry_count":   nextCount,
					"post_window_next_retry_at": nextRetry,
				}).Error
		})
		return false, err
	}
}

func finalizePendingPostMaintenanceLiveTx(tx *gorm.DB, cctvID, primaryPocID uuid.UUID, at time.Time) error {
	var expiredIDs []uuid.UUID
	if err := tx.Table("cctv_planned_outage_cameras poc").
		Select("poc.id").
		Joins("JOIN cctv_planned_outages po ON po.id = poc.planned_outage_id").
		Where("poc.cctv_id = ? AND poc.post_window_processed_at IS NULL AND COALESCE(po.ended_early_at, po.ends_at) <= ?", cctvID, at).
		Pluck("poc.id", &expiredIDs).Error; err != nil {
		return err
	}
	if primaryPocID != uuid.Nil {
		found := false
		for _, id := range expiredIDs {
			if id == primaryPocID {
				found = true
				break
			}
		}
		if !found {
			expiredIDs = append(expiredIDs, primaryPocID)
		}
	}
	if len(expiredIDs) == 0 {
		return nil
	}
	return tx.Model(&PlannedOutageCamera{}).
		Where("id IN ? AND post_window_processed_at IS NULL", expiredIDs).
		Updates(map[string]interface{}{
			"last_verified_state":       "live",
			"last_verified_at":          at,
			"post_window_processed_at":  at,
			"post_window_next_retry_at": gorm.Expr("NULL"),
		}).Error
}

// resolvePendingPostMaintenanceObservationTx memastikan jalur monitoring biasa (ObserveDetailed /
// ObserveSharedCorrelatedWave) tidak mendahului evaluasi pasca-maintenance yang masih tertunda
// dengan tiket `unplanned_outage`. Bila kamera memiliki evaluasi pasca-jendela yang belum selesai
// (`suppressed_confirmed_offline_at IS NOT NULL` & `post_window_processed_at IS NULL` & jendela sudah berakhir):
//   - `unknown` : pertahankan tertunda dengan backoff terkendali tanpa membuka tiket.
//   - `live`    : selesaikan evaluasi tanpa tiket.
//   - `offline` : buka tepat 1 tiket `post_planned_maintenance` dengan konteks dan SLA akhir jendela yang benar.
func (s Service) resolvePendingPostMaintenanceObservationTx(tx *gorm.DB, obs CameraObservation, state string, at time.Time) (bool, error) {
	type pendingRow struct {
		PocID                        uuid.UUID
		EffectiveEnd                 time.Time
		SuppressedFirstFailedAt      *time.Time
		SuppressedConfirmedOfflineAt *time.Time
		PostWindowRetryCount         int
	}
	var rows []pendingRow
	err := tx.Table("cctv_planned_outage_cameras poc").
		Select(`
			poc.id AS poc_id,
			COALESCE(po.ended_early_at, po.ends_at) AS effective_end,
			poc.suppressed_first_failed_at,
			poc.suppressed_confirmed_offline_at,
			poc.post_window_retry_count
		`).
		Joins("JOIN cctv_planned_outages po ON po.id = poc.planned_outage_id").
		Where("poc.cctv_id = ? AND poc.post_window_processed_at IS NULL AND poc.suppressed_confirmed_offline_at IS NOT NULL AND COALESCE(po.ended_early_at, po.ends_at) <= ?",
			obs.Camera.ID, at).
		Order("effective_end ASC").
		Scan(&rows).Error
	if err != nil {
		return false, err
	}
	if len(rows) == 0 {
		return false, nil
	}

	switch state {
	case "live":
		if err := finalizePendingPostMaintenanceLiveTx(tx, obs.Camera.ID, rows[0].PocID, at); err != nil {
			return false, err
		}
		return true, nil

	case StateOffline:
		origConfirmed := *rows[0].SuppressedConfirmedOfflineAt
		origFirstFailed := origConfirmed
		if rows[0].SuppressedFirstFailedAt != nil && !rows[0].SuppressedFirstFailedAt.IsZero() {
			origFirstFailed = *rows[0].SuppressedFirstFailedAt
		}
		effectiveEnd := rows[0].EffectiveEnd
		pocIDs := make([]uuid.UUID, 0, len(rows))
		for _, r := range rows {
			pocIDs = append(pocIDs, r.PocID)
			if r.SuppressedConfirmedOfflineAt != nil && r.SuppressedConfirmedOfflineAt.Before(origConfirmed) {
				origConfirmed = *r.SuppressedConfirmedOfflineAt
			}
			ff := origConfirmed
			if r.SuppressedFirstFailedAt != nil && !r.SuppressedFirstFailedAt.IsZero() {
				ff = *r.SuppressedFirstFailedAt
			}
			if ff.Before(origFirstFailed) {
				origFirstFailed = ff
			}
			if r.EffectiveEnd.After(effectiveEnd) {
				effectiveEnd = r.EffectiveEnd
			}
		}
		_, err := s.openPostMaintenanceIncidentTx(tx, pocIDs, obs.Camera, origFirstFailed, origConfirmed, effectiveEnd, at, obs.Evidence)
		return true, err

	default:
		for _, r := range rows {
			nextCount := r.PostWindowRetryCount + 1
			nextRetry := at.Add(PostMaintenanceUnknownRetryDelay(r.PostWindowRetryCount))
			if err := tx.Model(&PlannedOutageCamera{}).
				Where("id = ? AND post_window_processed_at IS NULL", r.PocID).
				Updates(map[string]interface{}{
					"last_verified_state":       StateUnknown,
					"last_verified_at":          at,
					"post_window_retry_count":   nextCount,
					"post_window_next_retry_at": nextRetry,
				}).Error; err != nil {
				return false, err
			}
		}
		return true, nil
	}
}

func (s Service) openPostMaintenanceIncidentAtomically(ctx context.Context, pocID uuid.UUID, camera models.CCTV, origFirstFailed, origConfirmed, effectiveEnd, now time.Time) (bool, error) {
	if camera.SiteID == nil || *camera.SiteID == uuid.Nil {
		return false, errors.New("kamera belum memiliki site")
	}
	var opened bool
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockSite(tx, *camera.SiteID); err != nil {
			return err
		}
		var pocIDs []uuid.UUID
		if err := tx.Table("cctv_planned_outage_cameras poc").
			Select("poc.id").
			Joins("JOIN cctv_planned_outages po ON po.id = poc.planned_outage_id").
			Where("poc.cctv_id = ? AND poc.post_window_processed_at IS NULL AND COALESCE(po.ended_early_at, po.ends_at) <= ?", camera.ID, now).
			Pluck("poc.id", &pocIDs).Error; err != nil {
			return err
		}
		found := false
		for _, id := range pocIDs {
			if id == pocID {
				found = true
				break
			}
		}
		if !found && pocID != uuid.Nil {
			pocIDs = append(pocIDs, pocID)
		}
		var txErr error
		opened, txErr = s.openPostMaintenanceIncidentTx(tx, pocIDs, camera, origFirstFailed, origConfirmed, effectiveEnd, now, nil)
		return txErr
	})
	return opened, err
}

func (s Service) openPostMaintenanceIncidentTx(
	tx *gorm.DB,
	pocIDs []uuid.UUID,
	camera models.CCTV,
	origFirstFailed, origConfirmed, effectiveEnd, now time.Time,
	evidence map[string]interface{},
) (bool, error) {
	if len(pocIDs) == 0 {
		return false, nil
	}
	res := tx.Model(&PlannedOutageCamera{}).
		Where("id IN ? AND post_window_processed_at IS NULL", pocIDs).
		Updates(map[string]interface{}{
			"last_verified_state":       StateOffline,
			"last_verified_at":          now,
			"post_window_processed_at":  now,
			"post_window_next_retry_at": gorm.Expr("NULL"),
		})
	if res.Error != nil {
		return false, res.Error
	}
	if res.RowsAffected == 0 {
		return false, nil
	}

	var existing Member
	findErr := tx.Where("cctv_id = ? AND is_active_member = true", camera.ID).First(&existing).Error
	if findErr == nil {
		return false, nil
	}
	if !errors.Is(findErr, gorm.ErrRecordNotFound) {
		return false, findErr
	}

	ticketNo, err := nextTicketNumber(tx, now)
	if err != nil {
		return false, err
	}
	slaDeadline := effectiveEnd.Add(SLATargetDuration)
	i := Incident{
		ID:                    uuid.New(),
		TicketNumber:          ticketNo,
		SiteID:                *camera.SiteID,
		Status:                StatusOpen,
		ScopeType:             ScopeSingleCamera,
		OriginType:            OriginPostPlannedMaintenance,
		Title:                 "CCTV masih offline pasca-pemeliharaan terjadwal: " + camera.Nama,
		FirstFailedObservedAt: origFirstFailed,
		ConfirmedOfflineAt:    origConfirmed,
		SLAClockStartAt:       effectiveEnd,
		SLAClockBasis:         SLABasisPlannedWindowEnd,
		TimestampSource:       TimestampSourceLiveObservation,
		OpenedAt:              now,
		UpdatedAt:             now,
	}
	if err := tx.Create(&i).Error; err != nil {
		return false, err
	}

	m := Member{
		ID:                    uuid.New(),
		IncidentID:            i.ID,
		CCTVID:                camera.ID,
		CameraCode:            camera.KodeCCTV,
		CameraName:            camera.Nama,
		LocationArea:          camera.LokasiArea,
		IsActiveMember:        true,
		ConnectionState:       StateOffline,
		FirstFailedObservedAt: &origFirstFailed,
		ConfirmedOfflineAt:    &origConfirmed,
		LastObservedAt:        &now,
		MemberOutcome:         OutcomePending,
		LastEvidence:          marshalJSONString(evidence),
	}
	if err := tx.Create(&m).Error; err != nil {
		return false, err
	}

	_ = recordIncidentEvent(tx, i.ID, &camera.ID, nil, "POST_MAINTENANCE_INCIDENT_OPENED",
		fmt.Sprintf("Jendela pemeliharaan berakhir dan kamera %s masih terkonfirmasi offline; SLA dihitung sejak akhir jendela", camera.Nama),
		map[string]interface{}{
			"first_failed_observed_at": origFirstFailed,
			"confirmed_offline_at":     origConfirmed,
			"sla_clock_start_at":       effectiveEnd,
			"sla_clock_basis":          SLABasisPlannedWindowEnd,
		}, now)

	wita := time.FixedZone("WITA", 8*3600)
	body := fmt.Sprintf("[Pasca-Pemeliharaan Terjadwal]\n%s - %s\nLokasi: %s\nGagal teramati saat pemeliharaan: %s WITA\nAkhir jendela pemeliharaan (awal SLA): %s WITA\nKamera masih offline; menunggu penugasan Admin.",
		camera.KodeCCTV, camera.Nama, camera.LokasiArea,
		origFirstFailed.In(wita).Format("15:04:05"),
		effectiveEnd.In(wita).Format("15:04:05"))

	_, err = enqueueWithOpts(tx, &i, i.SiteID, nil, "CCTV_OFFLINE", body, now, enqueueOpts{
		TargetChannel:  ChannelOpsGroup,
		CoalesceWindow: s.coalesceDuration(false),
		SLADeadlineAt:  &slaDeadline,
	})
	if err != nil {
		return false, err
	}
	return true, nil
}

func normalizeStatusLower(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
