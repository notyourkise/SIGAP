package incidents

import (
	"github.com/google/uuid"
	"time"
)

type Incident struct {
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	SiteID           uuid.UUID  `gorm:"type:uuid" json:"site_id"`
	Status           string     `json:"status"`
	Title            string     `json:"title"`
	MaintenanceLogID *uuid.UUID `gorm:"type:uuid" json:"maintenance_log_id"`
	AssignedTo       *uuid.UUID `gorm:"type:uuid" json:"assigned_to"`
	OpenedAt         time.Time  `json:"opened_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	ResolvedAt       *time.Time `json:"resolved_at"`
	ResolvedBy       *uuid.UUID `gorm:"type:uuid" json:"resolved_by"`
	ResolutionNote   string     `json:"resolution_note"`
	Members          []Member   `gorm:"foreignKey:IncidentID" json:"members"`
}

func (Incident) TableName() string { return "cctv_incidents" }

type Member struct {
	ID                   uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	IncidentID           uuid.UUID  `gorm:"type:uuid" json:"incident_id"`
	CCTVID               uuid.UUID  `gorm:"type:uuid" json:"cctv_id"`
	CameraName           string     `json:"camera_name"`
	IsActiveMember       bool       `json:"is_active_member"`
	ConnectionState      string     `json:"connection_state"`
	LastObservedAt       *time.Time `json:"last_observed_at"`
	RecoveryStartedAt    *time.Time `json:"recovery_started_at"`
	ConsecutiveLiveCount int        `json:"consecutive_live_count"`
}

func (Member) TableName() string { return "cctv_incident_cameras" }

type Outbox struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	IncidentID    uuid.UUID  `gorm:"type:uuid" json:"incident_id"`
	SiteID        uuid.UUID  `gorm:"type:uuid" json:"site_id"`
	EventType     string     `json:"event_type"`
	Text          string     `json:"text"`
	Status        string     `json:"status"`
	Attempts      int        `json:"attempts"`
	NextAttemptAt time.Time  `json:"next_attempt_at"`
	LeaseUntil    *time.Time `json:"lease_until"`
	MessageID     int64      `json:"message_id"`
	CreatedAt     time.Time  `json:"created_at"`
	SentAt        *time.Time `json:"sent_at"`
	LastError     string     `json:"last_error"`
}

func (Outbox) TableName() string { return "telegram_outbox" }
