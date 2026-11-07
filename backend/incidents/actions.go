package incidents

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"sigap-backend/models"
	"sigap-backend/utils"
)

// Assign menugaskan atau mengalihkan teknisi pada tiket tunggal maupun tiket bersama.
// Untuk tiket bersama (>=2 kamera aktif atau shared_nvr/manual_merged), dibuat 1
// MaintenanceLog perbaikan bersama (work_scope = 'shared_infrastructure') sesuai Keputusan #4.
func (s Service) Assign(ctx context.Context, id, site, technician uuid.UUID, now time.Time) error {
	var notifyNewTech *uuid.UUID
	var notifyOldTech *uuid.UUID
	var notifyRefID uuid.UUID
	var notifyTitle string

	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockSite(tx, site); err != nil {
			return err
		}
		var i Incident
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Preload("Members").
			First(&i, "id = ? AND site_id = ?", id, site).Error; err != nil {
			return err
		}
		if i.Status == StatusResolved || i.Status == StatusMerged {
			return errors.New("insiden sudah ditutup atau digabung")
		}

		activeMembers := make([]Member, 0, len(i.Members))
		for _, m := range i.Members {
			if m.IsActiveMember {
				activeMembers = append(activeMembers, m)
			}
		}
		if len(activeMembers) == 0 {
			return errors.New("insiden tidak memiliki kamera aktif")
		}

		var tech models.User
		if err := tx.First(&tech, "id = ? AND role = ? AND deleted_at IS NULL", technician, "teknisi").Error; err != nil {
			return errors.New("teknisi tidak ditemukan")
		}
		var membership int64
		if err := tx.Table("user_sites").Where("user_id = ? AND site_id = ?", technician, site).Count(&membership).Error; err != nil {
			return err
		}
		if membership == 0 {
			return errors.New("teknisi tidak ditugaskan pada site ini")
		}

		if i.AssignedTo != nil && *i.AssignedTo == technician && i.MaintenanceLogID != nil {
			return errors.New("teknisi sudah ditugaskan pada insiden ini")
		}

		var prevTech *uuid.UUID
		if i.AssignedTo != nil {
			oldCopy := *i.AssignedTo
			prevTech = &oldCopy
		}

		workScope := WorkScopeIncidentCamera
		actionTaken := i.Title
		if len(activeMembers) > 1 || i.ScopeType == ScopeSharedNVR || i.ScopeType == ScopeManualMerged {
			workScope = WorkScopeSharedInfrastructure
			actionTaken = fmt.Sprintf("[Perbaikan Bersama NVR/Jaringan] %s (%d kamera terdampak)", i.TicketNumber, len(activeMembers))
		}

		var task models.MaintenanceLog
		if i.MaintenanceLogID != nil {
			if err := tx.First(&task, "id = ?", *i.MaintenanceLogID).Error; err == nil && (task.Status == "ongoing" || task.Status == "scheduled") {
				task.TechnicianID = &technician
				task.AssignedAt = &now
				task.Status = "ongoing"
				task.WorkScope = workScope
				task.UpdatedAt = now
				if err := tx.Save(&task).Error; err != nil {
					return err
				}
			} else {
				i.MaintenanceLogID = nil
			}
		}
		if i.MaintenanceLogID == nil {
			incID := i.ID
			task = models.MaintenanceLog{
				CCTVID:       activeMembers[0].CCTVID,
				SiteID:       &site,
				TechnicianID: &technician,
				IncidentID:   &incID,
				WorkScope:    workScope,
				AssignedAt:   &now,
				ActionTaken:  actionTaken,
				Notes:        "Tiket insiden " + i.TicketNumber,
				Status:       "ongoing",
			}
			if err := tx.Create(&task).Error; err != nil {
				return err
			}
			i.MaintenanceLogID = &task.ID
		}

		i.AssignedTo = &technician
		i.UpdatedAt = now
		if i.Status != StatusPendingVerification {
			i.Status = StatusInProgress
		}
		if err := tx.Omit("Members", "Events").Save(&i).Error; err != nil {
			return err
		}

		notifyNewTech = &technician
		notifyOldTech = prevTech
		notifyRefID = task.ID
		notifyTitle = i.TicketNumber + " - " + i.Title

		eventLabel := "INCIDENT_ASSIGNED"
		msg := fmt.Sprintf("Ditugaskan kepada %s melalui website SIGAP (%s).", tech.FullName, workScope)
		if prevTech != nil {
			eventLabel = "INCIDENT_REASSIGNED"
			msg = fmt.Sprintf("Penugasan dialihkan kepada %s melalui website SIGAP.", tech.FullName)
		}
		_ = recordIncidentEvent(tx, i.ID, nil, nil, eventLabel, msg,
			map[string]interface{}{"technician_id": technician.String(), "work_scope": workScope, "maintenance_log_id": task.ID.String()}, now)

		_, err := enqueueWithOpts(tx, &i, i.SiteID, nil, "INCIDENT_ASSIGNED", msg, now, enqueueOpts{TargetChannel: ChannelOpsGroup})
		return err
	})
	if err != nil {
		return err
	}

	if notifyNewTech != nil {
		utils.NotifyUser(s.DB, *notifyNewTech, models.NotificationTypeAssigned,
			"Penugasan Insiden CCTV",
			"Anda ditugaskan menangani "+notifyTitle+".",
			"maintenance-log", &notifyRefID)
	}
	if notifyOldTech != nil {
		utils.NotifyUser(s.DB, *notifyOldTech, models.NotificationTypeUnassigned,
			"Tugas Insiden Dialihkan",
			"Penugasan "+notifyTitle+" telah dialihkan ke teknisi lain.",
			"maintenance-log", &notifyRefID)
	}
	return nil
}

// CreateCameraFieldTask membuat tugas pemeriksaan fisik kamera individual (camera_field_check)
// hanya ketika kamera tertentu pada tiket bersama memang perlu diperiksa di lapangan (Keputusan #4).
func (s Service) CreateCameraFieldTask(ctx context.Context, incidentID, cctvID, site, technician uuid.UUID, notes string, now time.Time) (*models.MaintenanceLog, error) {
	var created models.MaintenanceLog
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockSite(tx, site); err != nil {
			return err
		}
		var i Incident
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&i, "id = ? AND site_id = ?", incidentID, site).Error; err != nil {
			return err
		}
		if i.Status == StatusResolved || i.Status == StatusMerged {
			return errors.New("insiden sudah ditutup atau digabung")
		}
		var member Member
		if err := tx.Where("incident_id = ? AND cctv_id = ? AND is_active_member = true", incidentID, cctvID).First(&member).Error; err != nil {
			return errors.New("kamera bukan anggota aktif insiden ini")
		}
		var tech models.User
		if err := tx.First(&tech, "id = ? AND role = ? AND deleted_at IS NULL", technician, "teknisi").Error; err != nil {
			return errors.New("teknisi tidak ditemukan")
		}
		var membership int64
		if err := tx.Table("user_sites").Where("user_id = ? AND site_id = ?", technician, site).Count(&membership).Error; err != nil {
			return err
		}
		if membership == 0 {
			return errors.New("teknisi tidak ditugaskan pada site ini")
		}

		incID := i.ID
		created = models.MaintenanceLog{
			CCTVID:       cctvID,
			SiteID:       &site,
			TechnicianID: &technician,
			IncidentID:   &incID,
			WorkScope:    WorkScopeCameraFieldCheck,
			AssignedAt:   &now,
			ActionTaken:  fmt.Sprintf("[Pemeriksaan Lapangan Kamera] %s - %s (%s)", member.CameraCode, member.CameraName, i.TicketNumber),
			Notes:        strings.TrimSpace(notes),
			Status:       "ongoing",
		}
		if err := tx.Create(&created).Error; err != nil {
			return err
		}
		member.FieldMaintenanceLogID = &created.ID
		if err := tx.Save(&member).Error; err != nil {
			return err
		}
		return recordIncidentEvent(tx, i.ID, &cctvID, nil, "CAMERA_FIELD_TASK_CREATED",
			fmt.Sprintf("Tugas pemeriksaan lapangan individual dibuat untuk %s (%s)", member.CameraName, tech.FullName),
			map[string]interface{}{"maintenance_log_id": created.ID.String(), "technician_id": technician.String()}, now)
	})
	if err != nil {
		return nil, err
	}
	refID := created.ID
	utils.NotifyUser(s.DB, technician, models.NotificationTypeAssigned,
		"Tugas Pemeriksaan Kamera Lapangan",
		created.ActionTaken,
		"maintenance-log", &refID)
	return &created, nil
}

// Resolve menutup insiden setelah verifikasi Admin atas bukti pemulihan stabil 5 menit.
// Mendukung:
//   - self_recovered ("Pulih Mandiri" tanpa penugasan teknisi / tanpa checklist — Keputusan #2),
//   - repaired_verified (seluruh anggota pulih setelah penanganan),
//   - partial_recovery_and_escalation (sebagian kamera diekskalasi ganti unit dan sisanya pulih — Koreksi #5).
func (s Service) Resolve(ctx context.Context, id, site, admin uuid.UUID, note string, now time.Time) error {
	if strings.TrimSpace(note) == "" {
		return errors.New("catatan verifikasi admin wajib diisi")
	}
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockSite(tx, site); err != nil {
			return err
		}
		var i Incident
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&i, "id = ? AND site_id = ?", id, site).Error; err != nil {
			return err
		}
		if i.Status != StatusPendingVerification {
			return ErrNotReady
		}
		var activeMembers []Member
		if err := tx.Where("incident_id = ? AND is_active_member = true", id).Find(&activeMembers).Error; err != nil {
			return err
		}
		if len(activeMembers) == 0 {
			return ErrNotReady
		}
		for _, m := range activeMembers {
			if m.ConnectionState != StateRecovered || m.LastObservedAt == nil || now.Sub(*m.LastObservedAt) > MaxEvidenceGap || m.LastObservedAt.After(now) {
				return ErrNotReady
			}
		}

		var escalatedCount int64
		if err := tx.Model(&Member{}).
			Where("incident_id = ? AND member_outcome = ?", id, OutcomeEscalatedReplacement).
			Count(&escalatedCount).Error; err != nil {
			return err
		}

		isSelfRecovered := i.AssignedTo == nil && i.MaintenanceLogID == nil
		memberOutcome := OutcomeRecoveredVerified
		if isSelfRecovered {
			memberOutcome = OutcomeSelfRecovered
		}

		resType := ResolutionRepairedVerified
		if escalatedCount > 0 {
			resType = ResolutionPartialRecoveryAndEscalation
		} else if isSelfRecovered {
			resType = ResolutionSelfRecovered
		}

		i.Status = StatusResolved
		i.ResolvedAt = &now
		i.ResolvedBy = &admin
		i.ResolutionType = &resType
		i.ResolutionNote = strings.TrimSpace(note)
		i.UpdatedAt = now
		if err := tx.Omit("Members", "Events").Save(&i).Error; err != nil {
			return err
		}

		if err := tx.Model(&Member{}).
			Where("incident_id = ? AND is_active_member = true", id).
			Updates(map[string]interface{}{
				"is_active_member":    false,
				"member_outcome":      memberOutcome,
				"outcome_recorded_at": now,
				"outcome_recorded_by": admin,
			}).Error; err != nil {
			return err
		}

		_ = recordIncidentEvent(tx, i.ID, nil, &admin, "INCIDENT_RESOLVED",
			fmt.Sprintf("Insiden ditutup oleh Admin (%s): %s", resType, strings.TrimSpace(note)),
			map[string]interface{}{"resolution_type": resType, "escalated_members": escalatedCount}, now)

		label := "Pulih Setelah Penanganan"
		switch resType {
		case ResolutionSelfRecovered:
			label = "Pulih Mandiri (tanpa penugasan teknisi)"
		case ResolutionPartialRecoveryAndEscalation:
			label = "Campuran Pulih & Eskalasi Penggantian Perangkat"
		}
		_, err := enqueueWithOpts(tx, &i, i.SiteID, nil, "INCIDENT_RESOLVED",
			fmt.Sprintf("Admin telah memverifikasi pemulihan dan menutup insiden (%s).", label),
			now, enqueueOpts{TargetChannel: ChannelOpsGroup})
		return err
	})
}

// MergeIncidents menggabungkan satu atau lebih tiket sumber ke tiket induk ("Gabungkan ke Tiket Bersama" — Keputusan #3).
// Mempertahankan waktu observasi awal, nomor tiket asal (source_ticket_number), dan riwayat event tanpa melanggar idx_camera_one_active_incident.
func (s Service) MergeIncidents(ctx context.Context, targetID uuid.UUID, sourceIDs []uuid.UUID, site, admin uuid.UUID, mergeReason string, now time.Time) error {
	mergeReason = strings.TrimSpace(mergeReason)
	if mergeReason == "" {
		return fmt.Errorf("%w: alasan penggabungan wajib diisi", ErrInvalidMerge)
	}
	uniqueSources := make([]uuid.UUID, 0, len(sourceIDs))
	seen := map[uuid.UUID]bool{targetID: true}
	for _, src := range sourceIDs {
		if src == uuid.Nil || seen[src] {
			continue
		}
		seen[src] = true
		uniqueSources = append(uniqueSources, src)
	}
	if len(uniqueSources) == 0 {
		return fmt.Errorf("%w: pilih minimal satu tiket sumber yang berbeda dari tiket induk", ErrInvalidMerge)
	}

	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockSite(tx, site); err != nil {
			return err
		}

		// Urutkan seluruh UUID untuk row-level lock deterministik bebas deadlock.
		allIDs := append([]uuid.UUID{targetID}, uniqueSources...)
		sort.Slice(allIDs, func(a, b int) bool { return allIDs[a].String() < allIDs[b].String() })

		var locked []Incident
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id IN ? AND site_id = ?", allIDs, site).
			Find(&locked).Error; err != nil {
			return err
		}
		if len(locked) != len(allIDs) {
			return ErrCrossSite
		}

		byID := make(map[uuid.UUID]*Incident, len(locked))
		for idx := range locked {
			inc := &locked[idx]
			if inc.Status == StatusResolved || inc.Status == StatusMerged {
				return fmt.Errorf("%w: tiket %s sudah ditutup atau digabung", ErrInvalidMerge, inc.TicketNumber)
			}
			byID[inc.ID] = inc
		}

		target := byID[targetID]
		if target.ScopeType == ScopeSingleCamera {
			target.ScopeType = ScopeManualMerged
		}

		mergedTicketNos := make([]string, 0, len(uniqueSources))
		transferredCount := 0

		for _, srcID := range uniqueSources {
			src := byID[srcID]
			mergedTicketNos = append(mergedTicketNos, src.TicketNumber)

			if src.FirstFailedObservedAt.Before(target.FirstFailedObservedAt) {
				target.FirstFailedObservedAt = src.FirstFailedObservedAt
			}
			if src.ConfirmedOfflineAt.Before(target.ConfirmedOfflineAt) {
				target.ConfirmedOfflineAt = src.ConfirmedOfflineAt
			}
			if src.SLAClockStartAt.Before(target.SLAClockStartAt) {
				target.SLAClockStartAt = src.SLAClockStartAt
				target.SLAClockBasis = src.SLAClockBasis
			}

			var srcMembers []Member
			if err := tx.Where("incident_id = ? AND is_active_member = true", src.ID).Find(&srcMembers).Error; err != nil {
				return err
			}

			// 1. Lepas keanggotaan aktif pada tiket sumber lebih dulu agar idx_camera_one_active_incident aman.
			if err := tx.Model(&Member{}).
				Where("incident_id = ? AND is_active_member = true", src.ID).
				Updates(map[string]interface{}{
					"is_active_member":    false,
					"member_outcome":      OutcomeMergedTransferred,
					"outcome_recorded_at": now,
					"outcome_recorded_by": admin,
				}).Error; err != nil {
				return err
			}

			// 2. Pindahkan anggota kamera ke tiket induk dengan menyalin penuh jejak waktu awalnya.
			for _, sm := range srcMembers {
				originTicket := sm.SourceTicketNumber
				if originTicket == "" {
					originTicket = src.TicketNumber
				}
				var existingTargetMember Member
				findErr := tx.Where("incident_id = ? AND cctv_id = ?", target.ID, sm.CCTVID).First(&existingTargetMember).Error
				if findErr == nil {
					existingTargetMember.IsActiveMember = true
					existingTargetMember.SourceTicketNumber = originTicket
					existingTargetMember.ConnectionState = sm.ConnectionState
					existingTargetMember.FirstFailedObservedAt = sm.FirstFailedObservedAt
					existingTargetMember.ConfirmedOfflineAt = sm.ConfirmedOfflineAt
					existingTargetMember.LastObservedAt = sm.LastObservedAt
					existingTargetMember.RecoveryStartedAt = sm.RecoveryStartedAt
					existingTargetMember.RecoveredAt = sm.RecoveredAt
					existingTargetMember.ConsecutiveLiveCount = sm.ConsecutiveLiveCount
					existingTargetMember.MemberOutcome = OutcomePending
					if existingTargetMember.FieldMaintenanceLogID == nil {
						existingTargetMember.FieldMaintenanceLogID = sm.FieldMaintenanceLogID
					}
					if err := tx.Save(&existingTargetMember).Error; err != nil {
						return err
					}
				} else if errors.Is(findErr, gorm.ErrRecordNotFound) {
					newMember := Member{
						ID:                    uuid.New(),
						IncidentID:            target.ID,
						CCTVID:                sm.CCTVID,
						CameraCode:            sm.CameraCode,
						CameraName:            sm.CameraName,
						LocationArea:          sm.LocationArea,
						SourceTicketNumber:    originTicket,
						IsActiveMember:        true,
						ConnectionState:       sm.ConnectionState,
						FirstFailedObservedAt: sm.FirstFailedObservedAt,
						ConfirmedOfflineAt:    sm.ConfirmedOfflineAt,
						LastObservedAt:        sm.LastObservedAt,
						RecoveryStartedAt:     sm.RecoveryStartedAt,
						RecoveredAt:           sm.RecoveredAt,
						ConsecutiveLiveCount:  sm.ConsecutiveLiveCount,
						MemberOutcome:         OutcomePending,
						FieldMaintenanceLogID: sm.FieldMaintenanceLogID,
						LastEvidence:          sm.LastEvidence,
					}
					if err := tx.Create(&newMember).Error; err != nil {
						return err
					}
				} else {
					return findErr
				}
				transferredCount++
			}

			// 3. Tandai tiket sumber sebagai 'merged' dan batalkan outbox pending miliknya.
			targetIDCopy := target.ID
			src.Status = StatusMerged
			src.MergedIntoIncidentID = &targetIDCopy
			src.MergedAt = &now
			src.MergedBy = &admin
			src.MergeReason = mergeReason
			src.UpdatedAt = now
			if err := tx.Omit("Members", "Events").Save(src).Error; err != nil {
				return err
			}

			_ = tx.Model(&Outbox{}).
				Where("incident_id = ? AND status = 'pending'", src.ID).
				Updates(map[string]interface{}{
					"status":          "failed",
					"failed_at":       now,
					"sla_deadline_at": nil,
					"last_error":      "Dibatalkan karena tiket digabung ke " + target.TicketNumber,
				}).Error

			_ = recordIncidentEvent(tx, src.ID, nil, &admin, "INCIDENT_MERGED_OUT",
				fmt.Sprintf("Digabungkan ke tiket bersama %s: %s", target.TicketNumber, mergeReason),
				map[string]interface{}{"target_incident_id": target.ID.String(), "target_ticket_number": target.TicketNumber}, now)
		}

		// 4. Evaluasi ulang status tiket induk setelah penggabungan.
		allReady, totalActive, _, _, err := evaluateIncidentActiveMembersTx(tx, target.ID, now)
		if err != nil {
			return err
		}
		if allReady && totalActive > 0 {
			target.Status = StatusPendingVerification
		} else if target.AssignedTo != nil {
			target.Status = StatusInProgress
		} else {
			target.Status = StatusOpen
		}
		target.UpdatedAt = now
		if err := tx.Omit("Members", "Events").Save(target).Error; err != nil {
			return err
		}

		_ = recordIncidentEvent(tx, target.ID, nil, &admin, "INCIDENT_MERGED_IN",
			fmt.Sprintf("Menerima penggabungan %d kamera dari tiket %s (%s)", transferredCount, strings.Join(mergedTicketNos, ", "), mergeReason),
			map[string]interface{}{"source_tickets": mergedTicketNos, "transferred_cameras": transferredCount}, now)

		_, err = enqueueWithOpts(tx, target, target.SiteID, nil, "INCIDENT_MERGED",
			fmt.Sprintf("Admin menggabungkan tiket [%s] ke tiket bersama %s.\nAlasan: %s\nTotal kamera aktif dipantau: %d.",
				strings.Join(mergedTicketNos, ", "), target.TicketNumber, mergeReason, totalActive),
			now, enqueueOpts{TargetChannel: ChannelOpsGroup})
		return err
	})
}

// EscalateReplacement menonaktifkan 1 atau lebih kamera rusak berat pada insiden dan
// menautkan tugas penggantian perangkat ber-penanggung-jawab masing-masing per kamera (Keputusan #5 & Koreksi #5).
func (s Service) EscalateReplacement(ctx context.Context, incidentID, site, admin uuid.UUID, items []EscalateCameraItem, note string, now time.Time) ([]models.MaintenanceLog, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("%w: pilih minimal satu kamera yang diekskalasi", ErrInvalidEscalation)
	}
	for _, item := range items {
		if item.CCTVID == uuid.Nil || strings.TrimSpace(item.DamageFindings) == "" {
			return nil, fmt.Errorf("%w: temuan kerusakan wajib diisi untuk setiap kamera", ErrInvalidEscalation)
		}
		if item.ExistingReplacementLogID == nil && item.TechnicianID == uuid.Nil {
			return nil, fmt.Errorf("%w: penanggung jawab tugas penggantian wajib dipilih untuk setiap kamera", ErrInvalidEscalation)
		}
	}

	replacementLogs := make([]models.MaintenanceLog, 0, len(items))

	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockSite(tx, site); err != nil {
			return err
		}

		var i Incident
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&i, "id = ? AND site_id = ?", incidentID, site).Error; err != nil {
			return err
		}
		if i.Status == StatusResolved || i.Status == StatusMerged {
			return errors.New("insiden sudah ditutup atau digabung")
		}

		escalatedNames := make([]string, 0, len(items))

		for _, item := range items {
			var member Member
			if err := tx.Where("incident_id = ? AND cctv_id = ? AND is_active_member = true", incidentID, item.CCTVID).First(&member).Error; err != nil {
				return fmt.Errorf("%w: kamera %s bukan anggota aktif pada insiden ini", ErrInvalidEscalation, item.CCTVID)
			}

			var cctv models.CCTV
			if err := tx.First(&cctv, "id = ? AND site_id = ?", item.CCTVID, site).Error; err != nil {
				return fmt.Errorf("%w: data CCTV tidak ditemukan pada site ini", ErrInvalidEscalation)
			}

			var repLog models.MaintenanceLog
			if item.ExistingReplacementLogID != nil {
				if err := tx.First(&repLog, "id = ? AND cctv_id = ?", *item.ExistingReplacementLogID, item.CCTVID).Error; err != nil {
					return fmt.Errorf("%w: tugas penggantian tertaut tidak ditemukan untuk kamera %s", ErrInvalidEscalation, cctv.KodeCCTV)
				}
				if repLog.SiteID != nil && *repLog.SiteID != site {
					return ErrCrossSite
				}
				if repLog.TechnicianID == nil || *repLog.TechnicianID == uuid.Nil {
					return fmt.Errorf("%w: tugas penggantian %s belum memiliki penanggung jawab", ErrInvalidEscalation, repLog.ID)
				}
				if repLog.Status != "scheduled" && repLog.Status != "ongoing" {
					return fmt.Errorf("%w: tugas penggantian harus berstatus tindak lanjut aktif (scheduled/ongoing)", ErrInvalidEscalation)
				}
				incID := i.ID
				repLog.IncidentID = &incID
				repLog.WorkScope = WorkScopeUnitReplacement
				if repLog.SiteID == nil {
					repLog.SiteID = &site
				}
				if err := tx.Save(&repLog).Error; err != nil {
					return err
				}
			} else {
				var tech models.User
				if err := tx.First(&tech, "id = ? AND role = ? AND deleted_at IS NULL", item.TechnicianID, "teknisi").Error; err != nil {
					return fmt.Errorf("%w: teknisi penanggung jawab tidak ditemukan", ErrInvalidEscalation)
				}
				var membership int64
				if err := tx.Table("user_sites").Where("user_id = ? AND site_id = ?", item.TechnicianID, site).Count(&membership).Error; err != nil {
					return err
				}
				if membership == 0 {
					return fmt.Errorf("%w: teknisi penanggung jawab tidak terdaftar pada site ini", ErrInvalidEscalation)
				}

				incID := i.ID
				techCopy := item.TechnicianID
				repLog = models.MaintenanceLog{
					CCTVID:        item.CCTVID,
					SiteID:        &site,
					TechnicianID:  &techCopy,
					IncidentID:    &incID,
					WorkScope:     WorkScopeUnitReplacement,
					AssignedAt:    &now,
					ScheduledDate: item.ScheduledDate,
					ActionTaken:   fmt.Sprintf("[Penggantian Unit CCTV] %s - %s (%s)", cctv.KodeCCTV, cctv.Nama, i.TicketNumber),
					Notes:         "Temuan kerusakan: " + strings.TrimSpace(item.DamageFindings),
					Status:        "ongoing",
				}
				if err := tx.Create(&repLog).Error; err != nil {
					return err
				}
			}

			// Nonaktifkan master CCTV dan matikan pemantauan aktif sampai unit baru dipasang & diverifikasi Admin.
			if err := tx.Model(&models.CCTV{}).Where("id = ?", item.CCTVID).Updates(map[string]interface{}{
				"status":             "nonaktif",
				"monitoring_enabled": false,
				"stream_enabled":     false,
				"updated_at":         now,
			}).Error; err != nil {
				return err
			}

			// Catat hasil eskalasi & tautan tugas penggantian secara spesifik di tingkat kamera (Koreksi #5).
			member.ConnectionState = StateEscalated
			member.MemberOutcome = OutcomeEscalatedReplacement
			member.DamageFindings = strings.TrimSpace(item.DamageFindings)
			member.ReplacementMaintenanceLogID = &repLog.ID
			member.IsActiveMember = false
			member.RecoveryStartedAt = nil
			member.RecoveredAt = nil
			member.ConsecutiveLiveCount = 0
			member.OutcomeRecordedAt = &now
			member.OutcomeRecordedBy = &admin
			if err := tx.Save(&member).Error; err != nil {
				return err
			}

			replacementLogs = append(replacementLogs, repLog)
			escalatedNames = append(escalatedNames, cctv.KodeCCTV+" ("+cctv.Nama+")")

			_ = recordIncidentEvent(tx, i.ID, &cctv.ID, &admin, "CAMERA_ESCALATED_REPLACEMENT",
				fmt.Sprintf("Kamera %s dinonaktifkan untuk penggantian unit: %s", cctv.Nama, strings.TrimSpace(item.DamageFindings)),
				map[string]interface{}{
					"replacement_maintenance_log_id": repLog.ID.String(),
					"damage_findings":                strings.TrimSpace(item.DamageFindings),
				}, now)
		}

		// Evaluasi sisa anggota aktif pada insiden induk.
		allReady, remainingActive, _, _, err := evaluateIncidentActiveMembersTx(tx, i.ID, now)
		if err != nil {
			return err
		}

		if remainingActive == 0 {
			var recoveredPrior int64
			if err := tx.Model(&Member{}).
				Where("incident_id = ? AND member_outcome IN ?", i.ID, []string{OutcomeRecoveredVerified, OutcomeSelfRecovered}).
				Count(&recoveredPrior).Error; err != nil {
				return err
			}
			resType := ResolutionEscalatedReplacement
			if recoveredPrior > 0 {
				resType = ResolutionPartialRecoveryAndEscalation
			}
			resNote := strings.TrimSpace(note)
			if resNote == "" {
				resNote = "Eskalasi penggantian perangkat tercatat untuk: " + strings.Join(escalatedNames, ", ")
			}
			i.Status = StatusResolved
			i.ResolvedAt = &now
			i.ResolvedBy = &admin
			i.ResolutionType = &resType
			i.ResolutionNote = resNote
			i.UpdatedAt = now
			if err := tx.Omit("Members", "Events").Save(&i).Error; err != nil {
				return err
			}
			_, err = enqueueWithOpts(tx, &i, i.SiteID, nil, "INCIDENT_ESCALATED_REPLACEMENT",
				fmt.Sprintf("Insiden ditutup dengan alasan Eskalasi Penggantian Perangkat.\nKamera dinonaktifkan sementara menunggu ganti unit: %s.",
					strings.Join(escalatedNames, ", ")),
				now, enqueueOpts{TargetChannel: ChannelOpsGroup})
			return err
		}

		if allReady {
			i.Status = StatusPendingVerification
		} else if i.AssignedTo != nil {
			i.Status = StatusInProgress
		} else {
			i.Status = StatusOpen
		}
		i.UpdatedAt = now
		if err := tx.Omit("Members", "Events").Save(&i).Error; err != nil {
			return err
		}

		_, err = enqueueWithOpts(tx, &i, i.SiteID, nil, "CAMERA_ESCALATED_REPLACEMENT",
			fmt.Sprintf("Kamera [%s] diekskalasi untuk penggantian perangkat (dinonaktifkan sementara). %d kamera lain pada tiket ini masih dipantau.",
				strings.Join(escalatedNames, ", "), remainingActive),
			now, enqueueOpts{TargetChannel: ChannelOpsGroup})
		return err
	})
	if err != nil {
		return nil, err
	}

	for idx := range replacementLogs {
		rl := replacementLogs[idx]
		if rl.TechnicianID != nil {
			refID := rl.ID
			utils.NotifyUser(s.DB, *rl.TechnicianID, models.NotificationTypeAssigned,
				"Tugas Penggantian Perangkat CCTV",
				rl.ActionTaken,
				"maintenance-log", &refID)
		}
	}
	return replacementLogs, nil
}

// RecordTechWorkReported mencatat pelaporan penyelesaian pekerjaan teknisi pada insiden
// tanpa menutup tiket insiden (penutupan tetap wajib menunggu stabil 5 menit + verifikasi Admin).
func (s Service) RecordTechWorkReported(ctx context.Context, incidentID, maintenanceLogID uuid.UUID, actorID *uuid.UUID, workScope, status, notes string, at time.Time) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var i Incident
		if err := tx.First(&i, "id = ?", incidentID).Error; err != nil {
			return err
		}
		return recordIncidentEvent(tx, i.ID, nil, actorID, "TECH_WORK_REPORTED",
			fmt.Sprintf("Teknisi melaporkan penyelesaian tugas %s (%s); menunggu pemulihan stream 5 menit & verifikasi Admin", workScope, status),
			map[string]interface{}{
				"maintenance_log_id": maintenanceLogID.String(),
				"work_scope":         workScope,
				"task_status":        status,
				"notes":              notes,
			}, at)
	})
}

// BuildCameraOutcomeReport menghasilkan ringkasan & rincian hasil tingkat kamera
// (memisahkan tegas kamera pulih vs dieskalasikan, tetap akurat walau tiket induk sudah ditutup — Koreksi #5).
func (s Service) BuildCameraOutcomeReport(ctx context.Context, site uuid.UUID) (IncidentOutcomeSummary, error) {
	var summary IncidentOutcomeSummary
	rows := make([]CameraOutcomeReportRow, 0)

	err := s.DB.WithContext(ctx).
		Table("cctv_incident_cameras m").
		Select(`
			i.id AS incident_id,
			i.ticket_number,
			i.status AS incident_status,
			i.resolution_type,
			i.origin_type,
			i.timestamp_source,
			i.sla_clock_start_at,
			i.sla_clock_basis,
			m.cctv_id,
			m.camera_code,
			m.camera_name,
			m.source_ticket_number,
			m.first_failed_observed_at,
			m.confirmed_offline_at,
			m.recovered_at,
			m.member_outcome,
			m.damage_findings,
			m.replacement_maintenance_log_id,
			m.outcome_recorded_at
		`).
		Joins("JOIN cctv_incidents i ON i.id = m.incident_id").
		Where("i.site_id = ? AND m.member_outcome <> ?", site, OutcomeMergedTransferred).
		Order("i.opened_at DESC, m.camera_name ASC").
		Scan(&rows).Error
	if err != nil {
		return summary, err
	}

	summary.Rows = rows
	summary.TotalCameraInvolvements = len(rows)
	for _, r := range rows {
		switch r.MemberOutcome {
		case OutcomeSelfRecovered:
			summary.SelfRecoveredCount++
		case OutcomeRecoveredVerified:
			summary.RecoveredVerifiedCount++
		case OutcomeEscalatedReplacement:
			summary.EscalatedCount++
		default:
			summary.PendingCount++
		}
	}

	_ = s.DB.WithContext(ctx).Model(&Outbox{}).
		Where("site_id = ? AND sla_breached = true", site).
		Count(&summary.SLABreachedOutboxCount).Error

	return summary, nil
}
