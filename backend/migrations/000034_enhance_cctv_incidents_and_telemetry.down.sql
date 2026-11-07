-- 000034_enhance_cctv_incidents_and_telemetry.down.sql
-- Rollback bersih & simetris ke skema 000033 tanpa error FK sirkular.

-- 1. Lepas FK silang pada monitor_health_episodes & cctv_incidents terhadap telegram_outbox
ALTER TABLE monitor_health_episodes
    DROP CONSTRAINT IF EXISTS fk_monitor_episode_alert_outbox,
    DROP CONSTRAINT IF EXISTS fk_monitor_episode_recovery_outbox;

ALTER TABLE cctv_incidents
    DROP COLUMN IF EXISTS assignment_reminder_outbox_id,
    DROP COLUMN IF EXISTS unrecovered_escalation_outbox_id;

-- 2. Kembalikan telegram_outbox ke skema 000031
DROP INDEX IF EXISTS idx_telegram_outbox_sla_check;
DROP INDEX IF EXISTS idx_uq_outbox_once_per_incident;

ALTER TABLE telegram_outbox
    DROP CONSTRAINT IF EXISTS chk_telegram_outbox_channel;

ALTER TABLE telegram_outbox
    DROP COLUMN IF EXISTS sla_breached_at,
    DROP COLUMN IF EXISTS sla_breached,
    DROP COLUMN IF EXISTS sla_deadline_at,
    DROP COLUMN IF EXISTS failed_at,
    DROP COLUMN IF EXISTS last_attempt_at,
    DROP COLUMN IF EXISTS batch_id,
    DROP COLUMN IF EXISTS coalesce_until,
    DROP COLUMN IF EXISTS target_channel,
    DROP COLUMN IF EXISTS monitor_episode_id;

DELETE FROM telegram_outbox WHERE incident_id IS NULL;

ALTER TABLE telegram_outbox
    ALTER COLUMN incident_id SET NOT NULL;

-- 3. Hapus tabel baru
DROP TABLE IF EXISTS cctv_probe_telemetry;
DROP TABLE IF EXISTS monitor_health_episodes;
DROP TABLE IF EXISTS cctv_incident_events;
DROP TABLE IF EXISTS cctv_planned_outage_cameras;
DROP TABLE IF EXISTS cctv_planned_outages;

-- 4. Kembalikan cctv_incident_cameras ke skema 000031
UPDATE cctv_incident_cameras
SET connection_state = 'offline'
WHERE connection_state = 'escalated';

ALTER TABLE cctv_incident_cameras
    DROP CONSTRAINT IF EXISTS chk_incident_member_outcome,
    DROP CONSTRAINT IF EXISTS chk_cctv_incident_cameras_state;

ALTER TABLE cctv_incident_cameras
    DROP COLUMN IF EXISTS last_evidence,
    DROP COLUMN IF EXISTS outcome_recorded_by,
    DROP COLUMN IF EXISTS outcome_recorded_at,
    DROP COLUMN IF EXISTS replacement_maintenance_log_id,
    DROP COLUMN IF EXISTS field_maintenance_log_id,
    DROP COLUMN IF EXISTS damage_findings,
    DROP COLUMN IF EXISTS member_outcome,
    DROP COLUMN IF EXISTS recovered_at,
    DROP COLUMN IF EXISTS confirmed_offline_at,
    DROP COLUMN IF EXISTS first_failed_observed_at,
    DROP COLUMN IF EXISTS source_ticket_number,
    DROP COLUMN IF EXISTS location_area,
    DROP COLUMN IF EXISTS camera_code;

ALTER TABLE cctv_incident_cameras
    ADD CONSTRAINT cctv_incident_cameras_connection_state_check
        CHECK (connection_state IN ('offline', 'recovering', 'recovered', 'unknown'));

-- 5. Kembalikan maintenance_logs ke skema 000033
DROP INDEX IF EXISTS idx_maintenance_logs_incident;

ALTER TABLE maintenance_logs
    DROP CONSTRAINT IF EXISTS chk_maintenance_logs_work_scope;

ALTER TABLE maintenance_logs
    DROP COLUMN IF EXISTS work_scope,
    DROP COLUMN IF EXISTS incident_id;

-- 6. Kembalikan cctv_incidents ke skema 000031
UPDATE cctv_incidents
SET
    status = 'resolved',
    resolved_at = COALESCE(resolved_at, merged_at, updated_at)
WHERE status = 'merged';

DROP INDEX IF EXISTS idx_uq_open_shared_incident;
DROP INDEX IF EXISTS idx_uq_cctv_incidents_ticket_number;

ALTER TABLE cctv_incidents
    DROP CONSTRAINT IF EXISTS chk_cctv_incidents_ts_source,
    DROP CONSTRAINT IF EXISTS chk_cctv_incidents_sla_basis,
    DROP CONSTRAINT IF EXISTS chk_cctv_incidents_resolution,
    DROP CONSTRAINT IF EXISTS chk_cctv_incidents_origin,
    DROP CONSTRAINT IF EXISTS chk_cctv_incidents_scope,
    DROP CONSTRAINT IF EXISTS chk_cctv_incidents_status;

ALTER TABLE cctv_incidents
    DROP COLUMN IF EXISTS unrecovered_escalation_queued_at,
    DROP COLUMN IF EXISTS assignment_reminder_queued_at,
    DROP COLUMN IF EXISTS merge_reason,
    DROP COLUMN IF EXISTS merged_by,
    DROP COLUMN IF EXISTS merged_at,
    DROP COLUMN IF EXISTS merged_into_incident_id,
    DROP COLUMN IF EXISTS resolution_type,
    DROP COLUMN IF EXISTS correlation_key,
    DROP COLUMN IF EXISTS origin_type,
    DROP COLUMN IF EXISTS scope_type,
    DROP COLUMN IF EXISTS timestamp_source,
    DROP COLUMN IF EXISTS sla_clock_basis,
    DROP COLUMN IF EXISTS sla_clock_start_at,
    DROP COLUMN IF EXISTS physical_fault_at,
    DROP COLUMN IF EXISTS confirmed_offline_at,
    DROP COLUMN IF EXISTS first_failed_observed_at,
    DROP COLUMN IF EXISTS ticket_number;

ALTER TABLE cctv_incidents
    ADD CONSTRAINT cctv_incidents_status_check
        CHECK (status IN ('open', 'in_progress', 'pending_verification', 'resolved'));

-- 7. Kembalikan cctvs ke skema sebelumnya
ALTER TABLE cctvs
    DROP COLUMN IF EXISTS monitoring_enabled;
