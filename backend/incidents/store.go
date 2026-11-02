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

var ErrNotReady = errors.New("kamera belum pulih stabil atau bukti sudah kedaluwarsa")

type Service struct{ DB *gorm.DB }

// Every incident mutation uses the same site-scoped transaction lock. This
// deliberately simple first version serializes creation, recovery and closure.
func lockSite(tx *gorm.DB, site uuid.UUID) error {
	return tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", "sigap-incidents:"+site.String()).Error
}

func enqueue(tx *gorm.DB, i Incident, kind, body string, now time.Time) error {
	return tx.Create(&Outbox{ID: uuid.New(), IncidentID: i.ID, SiteID: i.SiteID, EventType: kind,
		Text: fmt.Sprintf("[SIGAP] %s\n%s\nTiket: %s", kind, body, i.ID), Status: "pending", NextAttemptAt: now, CreatedAt: now}).Error
}

// Observe only accepts verified camera evidence. "offline" must have already
// passed the verifier's confirmation threshold; unknown never opens an incident.
func (s Service) Observe(ctx context.Context, camera models.CCTV, state string, at time.Time) error {
	if camera.SiteID == nil || *camera.SiteID == uuid.Nil {
		return errors.New("kamera belum memiliki site")
	}
	if state != "live" && state != "offline" && state != "unknown" {
		state = "unknown"
	}
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockSite(tx, *camera.SiteID); err != nil {
			return err
		}
		var member Member
		err := tx.Where("cctv_id = ? AND is_active_member = true", camera.ID).First(&member).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if state != "offline" {
				return nil
			}
			i := Incident{ID: uuid.New(), SiteID: *camera.SiteID, Status: "open", Title: "CCTV tidak dapat diakses: " + camera.Nama, OpenedAt: at, UpdatedAt: at}
			if err := tx.Create(&i).Error; err != nil {
				return err
			}
			member = Member{ID: uuid.New(), IncidentID: i.ID, CCTVID: camera.ID, CameraName: camera.Nama, IsActiveMember: true, ConnectionState: "offline", LastObservedAt: &at}
			if err := tx.Create(&member).Error; err != nil {
				return err
			}
			return enqueue(tx, i, "CCTV_OFFLINE", camera.KodeCCTV+" - "+camera.Nama+"\nLokasi: "+camera.LokasiArea+"\nGangguan terkonfirmasi; menunggu penugasan admin.", at)
		}
		if err != nil {
			return err
		}
		if member.LastObservedAt != nil && !at.After(*member.LastObservedAt) {
			return nil
		}
		var i Incident
		if err := tx.First(&i, "id = ? AND site_id = ?", member.IncidentID, *camera.SiteID).Error; err != nil {
			return err
		}
		old := member.ConnectionState
		recovered := Advance(&member, state, at)
		if err := tx.Save(&member).Error; err != nil {
			return err
		}
		var remaining int64
		if err := tx.Model(&Member{}).Where("incident_id = ? AND is_active_member = true AND (connection_state <> 'recovered' OR last_observed_at IS NULL OR last_observed_at < ?)", i.ID, at.Add(-MaxEvidenceGap)).Count(&remaining).Error; err != nil {
			return err
		}
		if remaining == 0 {
			i.Status = "pending_verification"
		} else if i.AssignedTo != nil {
			i.Status = "in_progress"
		} else {
			i.Status = "open"
		}
		i.UpdatedAt = at
		if err := tx.Omit("Members").Save(&i).Error; err != nil {
			return err
		}
		if recovered {
			return enqueue(tx, i, "CCTV_RECOVERED", camera.Nama+" stabil online selama 5 menit berdasarkan pemeriksaan berkala. Menunggu verifikasi admin; tiket belum ditutup.", at)
		}
		if state == "offline" && old != "offline" {
			return enqueue(tx, i, "CCTV_OFFLINE_AGAIN", camera.Nama+" kembali offline. Tiket yang sama tetap digunakan.", at)
		}
		return nil
	})
}

func (s Service) Resolve(ctx context.Context, id, site, admin uuid.UUID, note string, now time.Time) error {
	if strings.TrimSpace(note) == "" {
		return errors.New("catatan verifikasi admin wajib diisi")
	}
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockSite(tx, site); err != nil {
			return err
		}
		var i Incident
		if err := tx.First(&i, "id = ? AND site_id = ?", id, site).Error; err != nil {
			return err
		}
		if i.Status != "pending_verification" {
			return ErrNotReady
		}
		var members []Member
		if err := tx.Where("incident_id = ? AND is_active_member = true", id).Find(&members).Error; err != nil {
			return err
		}
		if len(members) == 0 {
			return ErrNotReady
		}
		for _, m := range members {
			if m.ConnectionState != "recovered" || m.LastObservedAt == nil || now.Sub(*m.LastObservedAt) > MaxEvidenceGap || m.LastObservedAt.After(now) {
				return ErrNotReady
			}
		}
		i.Status = "resolved"
		i.ResolvedAt = &now
		i.ResolvedBy = &admin
		i.ResolutionNote = strings.TrimSpace(note)
		i.UpdatedAt = now
		if err := tx.Omit("Members").Save(&i).Error; err != nil {
			return err
		}
		if err := tx.Model(&Member{}).Where("incident_id = ?", id).Update("is_active_member", false).Error; err != nil {
			return err
		}
		return enqueue(tx, i, "INCIDENT_RESOLVED", "Admin telah memverifikasi pemulihan dan menutup insiden.", now)
	})
}

// Assign creates the field task only when the admin assigns work. This first
// release creates individual-camera incidents; it never fabricates batch checks.
func (s Service) Assign(ctx context.Context, id, site, technician uuid.UUID, now time.Time) error {
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockSite(tx, site); err != nil {
			return err
		}
		var i Incident
		if err := tx.Preload("Members").First(&i, "id = ? AND site_id = ?", id, site).Error; err != nil {
			return err
		}
		if i.Status == "resolved" || i.MaintenanceLogID != nil {
			return errors.New("insiden sudah ditutup atau ditugaskan")
		}
		if len(i.Members) != 1 {
			return errors.New("penugasan bersama belum didukung pada versi awal")
		}
		var tech models.User
		if err := tx.First(&tech, "id = ? AND role = ?", technician, "teknisi").Error; err != nil {
			return errors.New("teknisi tidak ditemukan")
		}
		var membership int64
		if err := tx.Table("user_sites").Where("user_id = ? AND site_id = ?", technician, site).Count(&membership).Error; err != nil {
			return err
		}
		if membership == 0 {
			return errors.New("teknisi tidak ditugaskan pada site ini")
		}
		task := models.MaintenanceLog{CCTVID: i.Members[0].CCTVID, SiteID: &site, TechnicianID: &technician, AssignedAt: &now, ActionTaken: i.Title, Notes: "Tiket insiden " + i.ID.String(), Status: "ongoing"}
		if err := tx.Create(&task).Error; err != nil {
			return err
		}
		i.MaintenanceLogID = &task.ID
		i.AssignedTo = &technician
		i.UpdatedAt = now
		if i.Status != "pending_verification" {
			i.Status = "in_progress"
		}
		if err := tx.Omit("Members").Save(&i).Error; err != nil {
			return err
		}
		return enqueue(tx, i, "INCIDENT_ASSIGNED", "Ditugaskan kepada "+tech.FullName+" melalui website SIGAP.", now)
	})
}

// ResetObservations invalidates recovery evidence after a monitoring restart.
// It does not announce a new outage or release active incident membership.
func (s Service) ResetObservations(ctx context.Context) error {
	var sites []uuid.UUID
	if err := s.DB.WithContext(ctx).Model(&Incident{}).Where("status <> 'resolved'").Distinct().Pluck("site_id", &sites).Error; err != nil {
		return err
	}
	for _, site := range sites {
		if err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := lockSite(tx, site); err != nil {
				return err
			}
			ids := tx.Model(&Incident{}).Select("id").Where("site_id = ? AND status <> 'resolved'", site)
			if err := tx.Model(&Member{}).Where("is_active_member = true AND incident_id IN (?)", ids).Updates(map[string]interface{}{"connection_state": "unknown", "recovery_started_at": nil, "consecutive_live_count": 0}).Error; err != nil {
				return err
			}
			return tx.Model(&Incident{}).Where("site_id = ? AND status = 'pending_verification'", site).Update("status", "in_progress").Error
		}); err != nil {
			return err
		}
	}
	return nil
}

// claim locks only while claiming work; no database transaction spans HTTP I/O.
func claim(db *gorm.DB, now time.Time, sites []string) (*Outbox, error) {
	var job Outbox
	err := db.Transaction(func(tx *gorm.DB) error {
		err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).Where("site_id IN ?", sites).Where("(status IN ('pending','ambiguous') AND next_attempt_at <= ?) OR (status = 'sending' AND lease_until < ?)", now, now).Order("created_at, id").First(&job).Error
		if err != nil {
			return err
		}
		lease := now.Add(30 * time.Second)
		job.LeaseUntil = &lease
		job.Status = "sending"
		job.Attempts++
		return tx.Save(&job).Error
	})
	if err != nil {
		return nil, err
	}
	return &job, nil
}
