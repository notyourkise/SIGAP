package incidents

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"sigap-backend/models"
	"sigap-backend/utils"
)

func AssignmentReminderThreshold() time.Duration {
	raw := strings.TrimSpace(os.Getenv("CCTV_INCIDENT_REMINDER_MINUTES"))
	if raw != "" {
		if mins, err := strconv.Atoi(raw); err == nil && mins >= 1 && mins <= 1440 {
			return time.Duration(mins) * time.Minute
		}
	}
	return AssignmentReminderDelay
}

func formatDurationMinutesLabel(d time.Duration) string {
	mins := int(d.Round(time.Minute) / time.Minute)
	if mins < 1 {
		mins = 1
	}
	return fmt.Sprintf("%d menit", mins)
}

func EscalationThreshold() time.Duration {
	raw := strings.TrimSpace(os.Getenv("CCTV_INCIDENT_ESCALATION_MINUTES"))
	if raw != "" {
		if mins, err := strconv.Atoi(raw); err == nil && mins >= 1 && mins <= 10080 {
			return time.Duration(mins) * time.Minute
		}
	}
	return DefaultEscalationMinutes * time.Minute
}

// ReportMonitorFailure mencatat gangguan pemantau (unknown) dalam monitor_health_episodes
// sehingga hanya menerbitkan TEPAT 1 notifikasi per kejadian (bukan 1 per kamera) ke:
//   1. Notifikasi In-App Admin,
//   2. Web Push Admin (melalui NotifyRole),
//   3. Grup Telegram Khusus Admin (target_channel = 'admin_group').
func (s Service) ReportMonitorFailure(ctx context.Context, siteID *uuid.UUID, cause, detail string, affectedCameras int, now time.Time) (openedNew bool, episode MonitorHealthEpisode, err error) {
	if siteID == nil || *siteID == uuid.Nil {
		return false, MonitorHealthEpisode{}, errors.New("site_id diperlukan untuk mencatat episode pemantau")
	}
	cause = strings.TrimSpace(cause)
	if cause == "" {
		cause = "mediamtx_unreachable"
	}

	err = s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockSite(tx, *siteID); err != nil {
			return err
		}

		findErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("site_id = ? AND status = 'open'", *siteID).
			First(&episode).Error
		if findErr == nil {
			episode.LastSeenAt = now
			episode.ConsecutiveHealthyCycles = 0
			episode.Detail = detail
			if affectedCameras > episode.AffectedCameraCount {
				episode.AffectedCameraCount = affectedCameras
			}
			return tx.Save(&episode).Error
		}
		if !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return findErr
		}

		siteCopy := *siteID
		episode = MonitorHealthEpisode{
			ID:                  uuid.New(),
			SiteID:              &siteCopy,
			Cause:               cause,
			Detail:              detail,
			AffectedCameraCount: affectedCameras,
			Status:              "open",
			StartedAt:           now,
			LastSeenAt:          now,
		}
		if err := tx.Create(&episode).Error; err != nil {
			return err
		}

		epID := episode.ID
		body := fmt.Sprintf("Gangguan Pemantau CCTV (%s)\nDetail: %s\nKamera belum dapat dipastikan (unknown): %d unit.\nSistem menahan pembuatan tiket offline kamera palsu sampai pemantau kembali sehat.",
			cause, detail, affectedCameras)
		outbox, err := enqueueWithOpts(tx, nil, siteCopy, &epID, "MONITOR_DEGRADED", body, now, enqueueOpts{
			TargetChannel: ChannelAdminGroup,
		})
		if err != nil {
			return err
		}

		episode.AlertOutboxID = &outbox.ID
		if err := tx.Save(&episode).Error; err != nil {
			return err
		}
		openedNew = true
		return nil
	})
	if err != nil {
		return false, episode, err
	}

	if openedNew {
		epRef := episode.ID
		utils.NotifyRole(s.DB, "admin", models.NotificationTypeIncidentMonitorDegraded,
			"Gangguan Pemantau CCTV (Unknown)",
			fmt.Sprintf("Pemantau mengalami kendala (%s: %s) yang memengaruhi %d kamera. Tidak ada tiket offline palsu yang dibuat.", cause, detail, affectedCameras),
			"monitor-episode", &epRef)
	}
	return openedNew, episode, nil
}

// ReportMonitorHealthyCycle menaikkan penghitung siklus sehat berturut-turut dan menutup
// episode gangguan pemantau setelah 2 siklus sehat berturut-turut dengan 1 pesan pemulihan.
func (s Service) ReportMonitorHealthyCycle(ctx context.Context, siteID uuid.UUID, now time.Time) (closed bool, err error) {
	var closedEpisodeID uuid.UUID
	var cause string

	err = s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockSite(tx, siteID); err != nil {
			return err
		}
		var episode MonitorHealthEpisode
		findErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("site_id = ? AND status = 'open'", siteID).
			First(&episode).Error
		if errors.Is(findErr, gorm.ErrRecordNotFound) {
			return nil
		}
		if findErr != nil {
			return findErr
		}

		episode.ConsecutiveHealthyCycles++
		episode.LastSeenAt = now
		if episode.ConsecutiveHealthyCycles < 2 {
			return tx.Save(&episode).Error
		}

		episode.Status = "resolved"
		episode.ResolvedAt = &now
		epID := episode.ID
		outbox, err := enqueueWithOpts(tx, nil, siteID, &epID, "MONITOR_RECOVERED",
			fmt.Sprintf("Pemantau CCTV kembali sehat setelah 2 siklus berturut-turut (sebab sebelumnya: %s). Pemeriksaan kamera normal dilanjutkan.", episode.Cause),
			now, enqueueOpts{TargetChannel: ChannelAdminGroup})
		if err != nil {
			return err
		}
		episode.RecoveryOutboxID = &outbox.ID
		if err := tx.Save(&episode).Error; err != nil {
			return err
		}
		closed = true
		closedEpisodeID = episode.ID
		cause = episode.Cause
		return nil
	})
	if err != nil {
		return false, err
	}
	if closed {
		utils.NotifyRole(s.DB, "admin", models.NotificationTypeIncidentMonitorRecovered,
			"Pemantau CCTV Kembali Normal",
			fmt.Sprintf("Gangguan pemantau (%s) telah pulih stabil.", cause),
			"monitor-episode", &closedEpisodeID)
	}
	return closed, nil
}

type PeriodicMaintenanceTickResult struct {
	SLABreachesMarked     int
	RemindersQueued       int
	EscalationsQueued     int
	PlannedOutagesOpened  int
	PlannedOutagesRefresh []PlannedOutageRefreshRequest
}

// EvaluateRemindersAndSLABreaches menjalankan:
//  1. Deteksi proaktif pelanggaran SLA (sla_breached = true segera saat now > sla_deadline_at tanpa sent_at,
//     termasuk bila outbox masih pending/sending/ambiguous maupun sudah failed permanen — Koreksi #3).
//  2. Pengingat penugasan 30 menit (maksimal 1 event & 1 entri outbox per insiden — Keputusan #7 & Koreksi #3).
//  3. Eskalasi 4 jam ke Admin jika gangguan masih berlangsung (maksimal 1 event & 1 entri outbox per insiden).
func (s Service) EvaluateRemindersAndSLABreaches(ctx context.Context, now time.Time) (slaBreaches, reminders, escalations int, err error) {
	// 1. Tandai pelanggaran SLA segera saat tenggat terlewati tanpa konfirmasi pengiriman API (sent_at IS NULL).
	var overdueOutbox []Outbox
	if err := s.DB.WithContext(ctx).
		Where("sla_deadline_at IS NOT NULL AND sla_breached = false AND sent_at IS NULL AND sla_deadline_at < ?", now).
		Find(&overdueOutbox).Error; err != nil {
		return 0, 0, 0, err
	}
	for _, ob := range overdueOutbox {
		res := s.DB.WithContext(ctx).Model(&Outbox{}).
			Where("id = ? AND sla_breached = false AND sent_at IS NULL", ob.ID).
			Updates(map[string]interface{}{
				"sla_breached":    true,
				"sla_breached_at": now,
			})
		if res.Error != nil {
			return slaBreaches, reminders, escalations, res.Error
		}
		if res.RowsAffected > 0 {
			slaBreaches++
			ref := ob.ID
			if ob.IncidentID != nil {
				ref = *ob.IncidentID
				_ = s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
					return recordIncidentEvent(tx, *ob.IncidentID, nil, nil, "SLA_NOTIFICATION_BREACHED",
						fmt.Sprintf("Target pemberitahuan 5 menit terlewati tanpa konfirmasi API Telegram (status outbox: %s)", ob.Status),
						map[string]interface{}{"outbox_id": ob.ID.String(), "outbox_status": ob.Status}, now)
				})
			}
			utils.NotifyRole(s.DB, "admin", models.NotificationTypeIncidentSLABreached,
				"Pelanggaran Target Waktu Notifikasi 5 Menit",
				fmt.Sprintf("Pemberitahuan Telegram melewati batas 5 menit tanpa konfirmasi API (status saat ini: %s).", ob.Status),
				"incident", &ref)
		}
	}

	// 2. Pengingat penugasan (default 30 menit, konfigurabel via CCTV_INCIDENT_REMINDER_MINUTES, terbuka, belum ditugaskan, maksimal 1x).
	remThreshold := AssignmentReminderThreshold()
	remLabel := formatDurationMinutesLabel(remThreshold)
	reminderCutoff := now.Add(-remThreshold)
	var dueReminders []Incident
	if err := s.DB.WithContext(ctx).
		Where("status = ? AND assigned_to IS NULL AND assignment_reminder_queued_at IS NULL AND sla_clock_start_at <= ?", StatusOpen, reminderCutoff).
		Find(&dueReminders).Error; err != nil {
		return slaBreaches, reminders, escalations, err
	}

	for _, cand := range dueReminders {
		queued := false
		txErr := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := lockSite(tx, cand.SiteID); err != nil {
				return err
			}
			var i Incident
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				First(&i, "id = ? AND status = ? AND assigned_to IS NULL AND assignment_reminder_queued_at IS NULL", cand.ID, StatusOpen).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil
				}
				return err
			}

			var unrecovered int64
			if err := tx.Model(&Member{}).
				Where("incident_id = ? AND is_active_member = true AND connection_state <> ?", i.ID, StateRecovered).
				Count(&unrecovered).Error; err != nil {
				return err
			}
			if unrecovered == 0 {
				return nil
			}

			outbox, err := enqueueWithOpts(tx, &i, i.SiteID, nil, "INCIDENT_ASSIGN_REMINDER",
				fmt.Sprintf("Pengingat %s: Tiket %s (%s) masih terbuka, belum ditugaskan ke teknisi, dan %d kamera belum pulih.",
					remLabel, i.TicketNumber, i.Title, unrecovered),
				now, enqueueOpts{TargetChannel: ChannelOpsGroup})
			if err != nil {
				return err
			}

			i.AssignmentReminderQueuedAt = &now
			i.AssignmentReminderOutboxID = &outbox.ID
			i.UpdatedAt = now
			if err := tx.Omit("Members", "Events").Save(&i).Error; err != nil {
				return err
			}
			_ = recordIncidentEvent(tx, i.ID, nil, nil, "INCIDENT_ASSIGN_REMINDER_QUEUED",
				fmt.Sprintf("Pengingat penugasan %s diantrekan ke outbox Telegram", remLabel),
				map[string]interface{}{"outbox_id": outbox.ID.String()}, now)
			queued = true
			return nil
		})
		if txErr != nil {
			return slaBreaches, reminders, escalations, txErr
		}
		if queued {
			reminders++
			ref := cand.ID
			utils.NotifyRole(s.DB, "admin", models.NotificationTypeIncidentAssignReminder,
				fmt.Sprintf("Pengingat Penugasan Insiden (%s)", remLabel),
				fmt.Sprintf("Tiket %s belum ditugaskan setelah %s dan kamera masih mengalami gangguan.", cand.TicketNumber, remLabel),
				"incident", &ref)
		}
	}

	// 3. Eskalasi ke Admin jika belum pulih setelah 4 jam (konfigurabel via CCTV_INCIDENT_ESCALATION_MINUTES, maksimal 1x).
	escThreshold := EscalationThreshold()
	escCutoff := now.Add(-escThreshold)
	var dueEscalations []Incident
	if err := s.DB.WithContext(ctx).
		Where("status IN ? AND unrecovered_escalation_queued_at IS NULL AND opened_at <= ?",
			[]string{StatusOpen, StatusInProgress}, escCutoff).
		Find(&dueEscalations).Error; err != nil {
		return slaBreaches, reminders, escalations, err
	}

	for _, cand := range dueEscalations {
		queued := false
		txErr := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := lockSite(tx, cand.SiteID); err != nil {
				return err
			}
			var i Incident
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				First(&i, "id = ? AND status IN ? AND unrecovered_escalation_queued_at IS NULL",
					cand.ID, []string{StatusOpen, StatusInProgress}).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil
				}
				return err
			}

			var unrecovered int64
			if err := tx.Model(&Member{}).
				Where("incident_id = ? AND is_active_member = true AND connection_state <> ?", i.ID, StateRecovered).
				Count(&unrecovered).Error; err != nil {
				return err
			}
			if unrecovered == 0 {
				return nil
			}

			outbox, err := enqueueWithOpts(tx, &i, i.SiteID, nil, "INCIDENT_UNRECOVERED_ESCALATION",
				fmt.Sprintf("Eskalasi Internal Admin: Tiket %s (%s) belum pulih setelah %s sejak dibuka (%d kamera masih terganggu).",
					i.TicketNumber, i.Title, escThreshold.Round(time.Minute), unrecovered),
				now, enqueueOpts{TargetChannel: ChannelAdminGroup})
			if err != nil {
				return err
			}

			i.UnrecoveredEscalationQueuedAt = &now
			i.UnrecoveredEscalationOutboxID = &outbox.ID
			i.UpdatedAt = now
			if err := tx.Omit("Members", "Events").Save(&i).Error; err != nil {
				return err
			}
			_ = recordIncidentEvent(tx, i.ID, nil, nil, "INCIDENT_UNRECOVERED_ESCALATION_QUEUED",
				fmt.Sprintf("Eskalasi belum pulih (%s) diantrekan ke grup Admin", escThreshold.Round(time.Minute)),
				map[string]interface{}{"outbox_id": outbox.ID.String()}, now)
			queued = true
			return nil
		})
		if txErr != nil {
			return slaBreaches, reminders, escalations, txErr
		}
		if queued {
			escalations++
			ref := cand.ID
			utils.NotifyRole(s.DB, "admin", models.NotificationTypeIncidentEscalation,
				"Eskalasi Insiden Belum Pulih",
				fmt.Sprintf("Tiket %s masih belum pulih setelah %s.", cand.TicketNumber, escThreshold.Round(time.Minute)),
				"incident", &ref)
		}
	}

	return slaBreaches, reminders, escalations, nil
}
