package incidents

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"sigap-backend/models"
	"sigap-backend/utils"
)

type autoAssignedNotice struct {
	TechnicianID uuid.UUID
	TaskID       uuid.UUID
	TicketNumber string
	Title        string
}

// pickNextSiteTechnicianTx memilih teknisi berikutnya secara berurutan (round-robin)
// berdasarkan kursor terakhir yang tersimpan di tabel sites. Pemanggil harus sudah
// memegang lockSite(tx, siteID) agar pemilihan berurutan aman terhadap konkurensi.
func pickNextSiteTechnicianTx(tx *gorm.DB, siteID uuid.UUID) (*models.User, error) {
	var techs []models.User
	if err := tx.Table("users u").
		Select("u.*").
		Joins("JOIN user_sites us ON us.user_id = u.id").
		Where("us.site_id = ? AND u.role = ? AND u.deleted_at IS NULL", siteID, "teknisi").
		Order("u.created_at ASC, u.id ASC").
		Find(&techs).Error; err != nil {
		return nil, err
	}
	if len(techs) == 0 {
		return nil, nil
	}

	var site models.Site
	if err := tx.Select("id", "last_auto_assigned_technician_id").
		First(&site, "id = ?", siteID).Error; err != nil {
		return nil, err
	}

	nextIdx := 0
	if site.LastAutoAssignedTechnicianID != nil && *site.LastAutoAssignedTechnicianID != uuid.Nil {
		for idx, candidate := range techs {
			if candidate.ID == *site.LastAutoAssignedTechnicianID {
				nextIdx = (idx + 1) % len(techs)
				break
			}
		}
	}

	chosen := techs[nextIdx]
	if err := tx.Model(&models.Site{}).
		Where("id = ?", siteID).
		Update("last_auto_assigned_technician_id", chosen.ID).Error; err != nil {
		return nil, err
	}
	return &chosen, nil
}

// autoAssignIncidentTx menugaskan teknisi secara otomatis pada tiket insiden baru
// beserta pembuatan MaintenanceLog agar tugas muncul langsung di dashboard teknisi.
// Bila site belum memiliki user role teknisi, tiket tetap berstatus 'open'.
func autoAssignIncidentTx(tx *gorm.DB, i *Incident, primaryCCTVID uuid.UUID, workScope, actionTaken string, now time.Time) (*models.User, *autoAssignedNotice, error) {
	tech, err := pickNextSiteTechnicianTx(tx, i.SiteID)
	if err != nil || tech == nil {
		return nil, nil, err
	}

	incID := i.ID
	task := models.MaintenanceLog{
		CCTVID:       primaryCCTVID,
		SiteID:       &i.SiteID,
		TechnicianID: &tech.ID,
		IncidentID:   &incID,
		WorkScope:    workScope,
		AssignedAt:   &now,
		ActionTaken:  actionTaken,
		Notes:        "Penugasan otomatis sistem untuk tiket " + i.TicketNumber,
		Status:       "ongoing",
	}
	if err := tx.Create(&task).Error; err != nil {
		return nil, nil, err
	}

	i.AssignedTo = &tech.ID
	i.MaintenanceLogID = &task.ID
	i.Status = StatusInProgress
	i.UpdatedAt = now
	if err := tx.Model(&Incident{}).Where("id = ?", i.ID).Updates(map[string]interface{}{
		"assigned_to":        i.AssignedTo,
		"maintenance_log_id": i.MaintenanceLogID,
		"status":             i.Status,
		"updated_at":         i.UpdatedAt,
	}).Error; err != nil {
		return nil, nil, err
	}

	_ = recordIncidentEvent(tx, i.ID, nil, nil, "INCIDENT_AUTO_ASSIGNED",
		fmt.Sprintf("Teknisi %s ditugaskan otomatis oleh sistem (%s)", tech.FullName, workScope),
		map[string]interface{}{
			"technician_id":      tech.ID.String(),
			"technician_name":    tech.FullName,
			"work_scope":         workScope,
			"maintenance_log_id": task.ID.String(),
			"assignment_mode":    "round_robin",
		}, now)

	notice := &autoAssignedNotice{
		TechnicianID: tech.ID,
		TaskID:       task.ID,
		TicketNumber: i.TicketNumber,
		Title:        i.Title,
	}
	return tech, notice, nil
}

func dispatchAutoAssignedNotices(db *gorm.DB, notices []*autoAssignedNotice) {
	if db == nil {
		return
	}
	for _, n := range notices {
		if n == nil {
			continue
		}
		refID := n.TaskID
		utils.NotifyUser(db, n.TechnicianID, models.NotificationTypeAssigned,
			"Penugasan Otomatis Insiden CCTV",
			fmt.Sprintf("Sistem menugaskan Anda menangani %s - %s.", n.TicketNumber, n.Title),
			"maintenance-log", &refID)
	}
}
