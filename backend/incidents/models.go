package incidents

import (
	"time"

	"github.com/google/uuid"
	"sigap-backend/models"
)

const (
	StatusOpen                = "open"
	StatusInProgress          = "in_progress"
	StatusPendingVerification = "pending_verification"
	StatusResolved            = "resolved"
	StatusMerged              = "merged"

	StateOffline    = "offline"
	StateRecovering = "recovering"
	StateRecovered  = "recovered"
	StateUnknown    = "unknown"
	StateEscalated  = "escalated"

	OutcomePending              = "pending"
	OutcomeSelfRecovered        = "self_recovered"
	OutcomeRecoveredVerified    = "recovered_verified"
	OutcomeEscalatedReplacement = "escalated_replacement"
	OutcomeMergedTransferred    = "merged_transferred"

	ResolutionSelfRecovered                = "self_recovered"
	ResolutionRepairedVerified             = "repaired_verified"
	ResolutionEscalatedReplacement         = "escalated_replacement"
	ResolutionPartialRecoveryAndEscalation = "partial_recovery_and_escalation"

	ScopeSingleCamera = "single_camera"
	ScopeSharedNVR    = "shared_nvr"
	ScopeManualMerged = "manual_merged"

	OriginUnplannedOutage        = "unplanned_outage"
	OriginPostPlannedMaintenance = "post_planned_maintenance"

	SLABasisFirstFailedObservation = "first_failed_observation"
	SLABasisPlannedWindowEnd       = "planned_window_end"
	SLABasisLegacyOpenedAt         = "legacy_opened_at"

	TimestampSourceLiveObservation = "live_observation"
	TimestampSourceLegacyBackfill  = "legacy_backfill_opened_at"

	ChannelOpsGroup   = "ops_group"
	ChannelAdminGroup = "admin_group"

	WorkScopeRoutine              = "routine"
	WorkScopeIncidentCamera       = "incident_camera"
	WorkScopeSharedInfrastructure = "shared_infrastructure"
	WorkScopeCameraFieldCheck     = "camera_field_check"
	WorkScopeUnitReplacement      = "unit_replacement"

	SLATargetDuration          = 5 * time.Minute
	AssignmentReminderDelay    = 30 * time.Minute
	DefaultEscalationMinutes   = 240
	DefaultCoalesceWindow      = 8 * time.Second
	PartialRecoveryDebounceGap = 30 * time.Second
)

type Incident struct {
	ID                            uuid.UUID       `gorm:"type:uuid;primaryKey" json:"id"`
	TicketNumber                  string          `gorm:"column:ticket_number" json:"ticket_number"`
	SiteID                        uuid.UUID       `gorm:"type:uuid" json:"site_id"`
	Status                        string          `json:"status"`
	ScopeType                     string          `gorm:"column:scope_type" json:"scope_type"`
	OriginType                    string          `gorm:"column:origin_type" json:"origin_type"`
	CorrelationKey                string          `gorm:"column:correlation_key" json:"correlation_key"`
	Title                         string          `json:"title"`
	MaintenanceLogID              *uuid.UUID      `gorm:"type:uuid" json:"maintenance_log_id"`
	AssignedTo                    *uuid.UUID      `gorm:"type:uuid" json:"assigned_to"`
	FirstFailedObservedAt         time.Time       `gorm:"column:first_failed_observed_at" json:"first_failed_observed_at"`
	ConfirmedOfflineAt            time.Time       `gorm:"column:confirmed_offline_at" json:"confirmed_offline_at"`
	PhysicalFaultAt               *time.Time      `gorm:"column:physical_fault_at" json:"physical_fault_at,omitempty"`
	SLAClockStartAt               time.Time       `gorm:"column:sla_clock_start_at" json:"sla_clock_start_at"`
	SLAClockBasis                 string          `gorm:"column:sla_clock_basis" json:"sla_clock_basis"`
	TimestampSource               string          `gorm:"column:timestamp_source" json:"timestamp_source"`
	OpenedAt                      time.Time       `json:"opened_at"`
	UpdatedAt                     time.Time       `json:"updated_at"`
	ResolvedAt                    *time.Time      `json:"resolved_at"`
	ResolvedBy                    *uuid.UUID      `gorm:"type:uuid" json:"resolved_by"`
	ResolutionType                *string         `gorm:"column:resolution_type" json:"resolution_type"`
	ResolutionNote                string          `json:"resolution_note"`
	MergedIntoIncidentID          *uuid.UUID      `gorm:"column:merged_into_incident_id;type:uuid" json:"merged_into_incident_id,omitempty"`
	MergedAt                      *time.Time      `gorm:"column:merged_at" json:"merged_at,omitempty"`
	MergedBy                      *uuid.UUID      `gorm:"column:merged_by;type:uuid" json:"merged_by,omitempty"`
	MergeReason                   string          `gorm:"column:merge_reason" json:"merge_reason,omitempty"`
	AssignmentReminderQueuedAt    *time.Time      `gorm:"column:assignment_reminder_queued_at" json:"assignment_reminder_queued_at,omitempty"`
	AssignmentReminderOutboxID    *uuid.UUID      `gorm:"column:assignment_reminder_outbox_id;type:uuid" json:"assignment_reminder_outbox_id,omitempty"`
	UnrecoveredEscalationQueuedAt *time.Time      `gorm:"column:unrecovered_escalation_queued_at" json:"unrecovered_escalation_queued_at,omitempty"`
	UnrecoveredEscalationOutboxID *uuid.UUID      `gorm:"column:unrecovered_escalation_outbox_id;type:uuid" json:"unrecovered_escalation_outbox_id,omitempty"`
	Members                       []Member        `gorm:"foreignKey:IncidentID" json:"members"`
	Events                        []IncidentEvent `gorm:"foreignKey:IncidentID" json:"events,omitempty"`
}

func (Incident) TableName() string { return "cctv_incidents" }

type Member struct {
	ID                          uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	IncidentID                  uuid.UUID  `gorm:"type:uuid" json:"incident_id"`
	CCTVID                      uuid.UUID  `gorm:"type:uuid" json:"cctv_id"`
	CameraCode                  string     `gorm:"column:camera_code" json:"camera_code"`
	CameraName                  string     `json:"camera_name"`
	LocationArea                string     `gorm:"column:location_area" json:"location_area"`
	SourceTicketNumber          string     `gorm:"column:source_ticket_number" json:"source_ticket_number,omitempty"`
	IsActiveMember              bool       `json:"is_active_member"`
	ConnectionState             string     `json:"connection_state"`
	FirstFailedObservedAt       *time.Time `gorm:"column:first_failed_observed_at" json:"first_failed_observed_at,omitempty"`
	ConfirmedOfflineAt          *time.Time `gorm:"column:confirmed_offline_at" json:"confirmed_offline_at,omitempty"`
	LastObservedAt              *time.Time `json:"last_observed_at"`
	RecoveryStartedAt           *time.Time `json:"recovery_started_at"`
	RecoveredAt                 *time.Time `gorm:"column:recovered_at" json:"recovered_at,omitempty"`
	ConsecutiveLiveCount        int        `json:"consecutive_live_count"`
	MemberOutcome               string     `gorm:"column:member_outcome" json:"member_outcome"`
	DamageFindings              string     `gorm:"column:damage_findings" json:"damage_findings,omitempty"`
	FieldMaintenanceLogID       *uuid.UUID `gorm:"column:field_maintenance_log_id;type:uuid" json:"field_maintenance_log_id,omitempty"`
	ReplacementMaintenanceLogID *uuid.UUID `gorm:"column:replacement_maintenance_log_id;type:uuid" json:"replacement_maintenance_log_id,omitempty"`
	OutcomeRecordedAt           *time.Time `gorm:"column:outcome_recorded_at" json:"outcome_recorded_at,omitempty"`
	OutcomeRecordedBy           *uuid.UUID `gorm:"column:outcome_recorded_by;type:uuid" json:"outcome_recorded_by,omitempty"`
	LastEvidence                string     `gorm:"column:last_evidence;type:jsonb;default:'{}'" json:"last_evidence,omitempty"`
}

func (Member) TableName() string { return "cctv_incident_cameras" }

type IncidentEvent struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	IncidentID uuid.UUID  `gorm:"type:uuid" json:"incident_id"`
	CCTVID     *uuid.UUID `gorm:"type:uuid" json:"cctv_id,omitempty"`
	ActorID    *uuid.UUID `gorm:"type:uuid" json:"actor_id,omitempty"`
	EventType  string     `json:"event_type"`
	Summary    string     `json:"summary"`
	Payload    string     `gorm:"type:jsonb;default:'{}'" json:"payload,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

func (IncidentEvent) TableName() string { return "cctv_incident_events" }

type PlannedOutage struct {
	ID           uuid.UUID             `gorm:"type:uuid;primaryKey" json:"id"`
	SiteID       uuid.UUID             `gorm:"type:uuid" json:"site_id"`
	Reason       string                `json:"reason"`
	StartsAt     time.Time             `json:"starts_at"`
	EndsAt       time.Time             `json:"ends_at"`
	EndedEarlyAt *time.Time            `json:"ended_early_at,omitempty"`
	EndedEarlyBy *uuid.UUID            `gorm:"type:uuid" json:"ended_early_by,omitempty"`
	CreatedBy    uuid.UUID             `gorm:"type:uuid" json:"created_by"`
	CreatedAt    time.Time             `json:"created_at"`
	UpdatedAt    time.Time             `json:"updated_at"`
	Cameras      []PlannedOutageCamera `gorm:"foreignKey:PlannedOutageID" json:"cameras"`
}

func (PlannedOutage) TableName() string { return "cctv_planned_outages" }

func (p PlannedOutage) EffectiveEndsAt() time.Time {
	if p.EndedEarlyAt != nil && p.EndedEarlyAt.Before(p.EndsAt) {
		return *p.EndedEarlyAt
	}
	return p.EndsAt
}

type PlannedOutageCamera struct {
	ID                           uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	PlannedOutageID              uuid.UUID  `gorm:"type:uuid" json:"planned_outage_id"`
	CCTVID                       uuid.UUID  `gorm:"type:uuid" json:"cctv_id"`
	SuppressedFirstFailedAt      *time.Time `json:"suppressed_first_failed_at,omitempty"`
	SuppressedConfirmedOfflineAt *time.Time `json:"suppressed_confirmed_offline_at,omitempty"`
	LastVerifiedState            string     `json:"last_verified_state"`
	LastVerifiedAt               *time.Time `json:"last_verified_at,omitempty"`
	PostWindowRetryCount         int        `json:"post_window_retry_count"`
	PostWindowNextRetryAt        *time.Time `json:"post_window_next_retry_at,omitempty"`
	PostWindowProcessedAt        *time.Time `json:"post_window_processed_at,omitempty"`
}

func (PlannedOutageCamera) TableName() string { return "cctv_planned_outage_cameras" }

type MonitorHealthEpisode struct {
	ID                       uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	SiteID                   *uuid.UUID `gorm:"type:uuid" json:"site_id,omitempty"`
	Cause                    string     `json:"cause"`
	Detail                   string     `json:"detail"`
	AffectedCameraCount      int        `json:"affected_camera_count"`
	Status                   string     `json:"status"`
	ConsecutiveHealthyCycles int        `json:"consecutive_healthy_cycles"`
	StartedAt                time.Time  `json:"started_at"`
	LastSeenAt               time.Time  `json:"last_seen_at"`
	ResolvedAt               *time.Time `json:"resolved_at,omitempty"`
	AlertOutboxID            *uuid.UUID `gorm:"type:uuid" json:"alert_outbox_id,omitempty"`
	RecoveryOutboxID         *uuid.UUID `gorm:"type:uuid" json:"recovery_outbox_id,omitempty"`
}

func (MonitorHealthEpisode) TableName() string { return "monitor_health_episodes" }

type ProbeTelemetry struct {
	ID                     uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	CCTVID                 uuid.UUID  `gorm:"type:uuid" json:"cctv_id"`
	SiteID                 *uuid.UUID `gorm:"type:uuid" json:"site_id,omitempty"`
	NetworkType            string     `json:"network_type"`
	IPNVR                  string     `json:"ip_nvr"`
	State                  string     `json:"state"`
	Method                 string     `json:"method"`
	VerifyDurationMs       int64      `json:"verify_duration_ms"`
	SourceLifetimeMs       int64      `json:"source_lifetime_ms"`
	BytesVerify            int64      `json:"bytes_verify"`
	BytesTotalSession      int64      `json:"bytes_total_session"`
	ByteDelta              int64      `json:"byte_delta"`
	SourceClosedConfirmed  bool       `json:"source_closed_confirmed"`
	ForcedClose            bool       `json:"forced_close"`
	RemovePathStatus       string     `json:"remove_path_status"`
	RemovePathError        string     `json:"remove_path_error,omitempty"`
	ReregisterStatus       string     `json:"reregister_status"`
	ReregisterError        string     `json:"reregister_error,omitempty"`
	CloseVerifyStatus      string     `json:"close_verify_status"`
	CloseUnconfirmedReason string     `json:"close_unconfirmed_reason,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
}

func (ProbeTelemetry) TableName() string { return "cctv_probe_telemetry" }

type Outbox struct {
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	IncidentID       *uuid.UUID `gorm:"type:uuid" json:"incident_id,omitempty"`
	MonitorEpisodeID *uuid.UUID `gorm:"type:uuid" json:"monitor_episode_id,omitempty"`
	SiteID           uuid.UUID  `gorm:"type:uuid" json:"site_id"`
	TargetChannel    string     `gorm:"column:target_channel" json:"target_channel"`
	EventType        string     `json:"event_type"`
	Text             string     `json:"text"`
	Status           string     `json:"status"`
	Attempts         int        `json:"attempts"`
	NextAttemptAt    time.Time  `json:"next_attempt_at"`
	CoalesceUntil    time.Time  `gorm:"column:coalesce_until" json:"coalesce_until"`
	BatchID          *uuid.UUID `gorm:"column:batch_id;type:uuid" json:"batch_id,omitempty"`
	LeaseUntil       *time.Time `json:"lease_until"`
	MessageID        int64      `json:"message_id"`
	CreatedAt        time.Time  `json:"created_at"`
	LastAttemptAt    *time.Time `gorm:"column:last_attempt_at" json:"last_attempt_at,omitempty"`
	SentAt           *time.Time `json:"sent_at"`
	FailedAt         *time.Time `gorm:"column:failed_at" json:"failed_at,omitempty"`
	SLADeadlineAt    *time.Time `gorm:"column:sla_deadline_at" json:"sla_deadline_at,omitempty"`
	SLABreached      bool       `gorm:"column:sla_breached" json:"sla_breached"`
	SLABreachedAt    *time.Time `gorm:"column:sla_breached_at" json:"sla_breached_at,omitempty"`
	LastError        string     `json:"last_error"`
}

func (Outbox) TableName() string { return "telegram_outbox" }

// CameraObservation membawa bukti verifikasi kamera dengan pemisahan tegas antara
// waktu gagal pertama teramati, waktu konfirmasi offline, dan waktu gangguan uji
// terkendali (physical_fault_at hanya diisi pada harness staging/pengujian).
type CameraObservation struct {
	Camera                models.CCTV
	State                 string
	ObservedAt            time.Time
	FirstFailedObservedAt *time.Time
	PhysicalFaultAt       *time.Time
	OriginType            string
	SLAClockStartOverride *time.Time
	Evidence              map[string]interface{}
	DisableBatching       bool
}

type EscalateCameraItem struct {
	CCTVID                   uuid.UUID  `json:"cctv_id"`
	DamageFindings           string     `json:"damage_findings"`
	TechnicianID             uuid.UUID  `json:"technician_id"`
	ExistingReplacementLogID *uuid.UUID `json:"existing_replacement_log_id,omitempty"`
	ScheduledDate            *time.Time `json:"scheduled_date,omitempty"`
}

type CameraOutcomeReportRow struct {
	IncidentID                  uuid.UUID  `json:"incident_id"`
	TicketNumber                string     `json:"ticket_number"`
	IncidentStatus              string     `json:"incident_status"`
	ResolutionType              *string    `json:"resolution_type"`
	OriginType                  string     `json:"origin_type"`
	TimestampSource             string     `json:"timestamp_source"`
	SLAClockStartAt             time.Time  `json:"sla_clock_start_at"`
	SLAClockBasis               string     `json:"sla_clock_basis"`
	CCTVID                      uuid.UUID  `json:"cctv_id"`
	CameraCode                  string     `json:"camera_code"`
	CameraName                  string     `json:"camera_name"`
	SourceTicketNumber          string     `json:"source_ticket_number,omitempty"`
	FirstFailedObservedAt       *time.Time `json:"first_failed_observed_at"`
	ConfirmedOfflineAt          *time.Time `json:"confirmed_offline_at"`
	RecoveredAt                 *time.Time `json:"recovered_at,omitempty"`
	MemberOutcome               string     `json:"member_outcome"`
	DamageFindings              string     `json:"damage_findings,omitempty"`
	ReplacementMaintenanceLogID *uuid.UUID `json:"replacement_maintenance_log_id,omitempty"`
	OutcomeRecordedAt           *time.Time `json:"outcome_recorded_at,omitempty"`
}

type IncidentOutcomeSummary struct {
	TotalCameraInvolvements int                      `json:"total_camera_involvements"`
	PendingCount            int                      `json:"pending_count"`
	SelfRecoveredCount      int                      `json:"self_recovered_count"`
	RecoveredVerifiedCount  int                      `json:"recovered_verified_count"`
	EscalatedCount          int                      `json:"escalated_count"`
	SLABreachedOutboxCount  int64                    `json:"sla_breached_outbox_count"`
	Rows                    []CameraOutcomeReportRow `json:"rows"`
}
