-- 000034_enhance_cctv_incidents_and_telemetry.up.sql
-- Migrasi 3-langkah (Expand -> Deterministic Backfill -> Enforce NOT NULL & UNIQUE)
-- untuk mendukung siklus insiden CCTV lengkap, jendela pemeliharaan terjadwal,
-- penggabungan tiket, hasil penanganan & eskalasi penggantian per kamera,
-- episode gangguan pemantau, batching/SLA Telegram, serta telemetri byte GSM.

-- ============================================================================
-- LANGKAH 1: EXPAND (Tambah tabel baru & kolom nullable/default pada tabel lama)
-- ============================================================================

-- 1.1 Master CCTV: pisahkan monitoring_enabled dari stream_enabled
ALTER TABLE cctvs
    ADD COLUMN IF NOT EXISTS monitoring_enabled BOOLEAN NOT NULL DEFAULT true;

UPDATE cctvs
SET monitoring_enabled = false
WHERE LOWER(status) = 'nonaktif';

-- 1.2 Jendela penghentian kamera terencana (Keputusan #1)
CREATE TABLE IF NOT EXISTS cctv_planned_outages (
    id UUID PRIMARY KEY,
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    reason TEXT NOT NULL,
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,
    ended_early_at TIMESTAMPTZ NULL,
    ended_early_by UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_planned_outage_window CHECK (ends_at > starts_at)
);

CREATE INDEX IF NOT EXISTS idx_planned_outages_site_window
    ON cctv_planned_outages(site_id, starts_at, ends_at);

CREATE TABLE IF NOT EXISTS cctv_planned_outage_cameras (
    id UUID PRIMARY KEY,
    planned_outage_id UUID NOT NULL REFERENCES cctv_planned_outages(id) ON DELETE CASCADE,
    cctv_id UUID NOT NULL REFERENCES cctvs(id) ON DELETE CASCADE,
    suppressed_first_failed_at TIMESTAMPTZ NULL,
    suppressed_confirmed_offline_at TIMESTAMPTZ NULL,
    last_verified_state VARCHAR(24) NOT NULL DEFAULT 'unknown',
    last_verified_at TIMESTAMPTZ NULL,
    post_window_retry_count INTEGER NOT NULL DEFAULT 0,
    post_window_next_retry_at TIMESTAMPTZ NULL,
    post_window_processed_at TIMESTAMPTZ NULL,
    UNIQUE(planned_outage_id, cctv_id)
);

CREATE INDEX IF NOT EXISTS idx_planned_outage_cameras_cctv
    ON cctv_planned_outage_cameras(cctv_id);

-- 1.3 Perluas cctv_incidents (kolom wajib baru dibuat NULL dahulu untuk kompatibilitas data 000031)
DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN
        SELECT c.conname
        FROM pg_constraint c
        JOIN pg_class t ON c.conrelid = t.oid
        WHERE t.relname = 'cctv_incidents'
          AND c.contype = 'c'
          AND pg_get_constraintdef(c.oid) ILIKE '%status%'
    LOOP
        EXECUTE format('ALTER TABLE cctv_incidents DROP CONSTRAINT %I', r.conname);
    END LOOP;
END $$;

ALTER TABLE cctv_incidents
    ADD CONSTRAINT chk_cctv_incidents_status
        CHECK (status IN ('open', 'in_progress', 'pending_verification', 'resolved', 'merged'));

ALTER TABLE cctv_incidents
    ADD COLUMN IF NOT EXISTS ticket_number VARCHAR(32) NULL,
    ADD COLUMN IF NOT EXISTS first_failed_observed_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS confirmed_offline_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS physical_fault_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS sla_clock_start_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS sla_clock_basis VARCHAR(32) NULL,
    ADD COLUMN IF NOT EXISTS timestamp_source VARCHAR(32) NULL,
    ADD COLUMN IF NOT EXISTS scope_type VARCHAR(24) NOT NULL DEFAULT 'single_camera',
    ADD COLUMN IF NOT EXISTS origin_type VARCHAR(32) NOT NULL DEFAULT 'unplanned_outage',
    ADD COLUMN IF NOT EXISTS correlation_key VARCHAR(120) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS resolution_type VARCHAR(40) NULL,
    ADD COLUMN IF NOT EXISTS merged_into_incident_id UUID NULL REFERENCES cctv_incidents(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS merged_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS merged_by UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS merge_reason TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS assignment_reminder_queued_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS assignment_reminder_outbox_id UUID NULL REFERENCES telegram_outbox(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS unrecovered_escalation_queued_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS unrecovered_escalation_outbox_id UUID NULL REFERENCES telegram_outbox(id) ON DELETE SET NULL;

ALTER TABLE cctv_incidents
    ADD CONSTRAINT chk_cctv_incidents_scope
        CHECK (scope_type IN ('single_camera', 'shared_nvr', 'manual_merged')),
    ADD CONSTRAINT chk_cctv_incidents_origin
        CHECK (origin_type IN ('unplanned_outage', 'post_planned_maintenance')),
    ADD CONSTRAINT chk_cctv_incidents_resolution
        CHECK (resolution_type IS NULL OR resolution_type IN (
            'self_recovered',
            'repaired_verified',
            'escalated_replacement',
            'partial_recovery_and_escalation'
        ));

-- 1.4 Kait ringan pada maintenance_logs tanpa mengubah alur rutin (work_scope = 'routine')
ALTER TABLE maintenance_logs
    ADD COLUMN IF NOT EXISTS incident_id UUID NULL REFERENCES cctv_incidents(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS work_scope VARCHAR(32) NOT NULL DEFAULT 'routine';

ALTER TABLE maintenance_logs
    ADD CONSTRAINT chk_maintenance_logs_work_scope
        CHECK (work_scope IN (
            'routine',
            'incident_camera',
            'shared_infrastructure',
            'camera_field_check',
            'unit_replacement'
        ));

CREATE INDEX IF NOT EXISTS idx_maintenance_logs_incident
    ON maintenance_logs(incident_id)
    WHERE incident_id IS NOT NULL;

-- 1.5 Perluas cctv_incident_cameras untuk hasil penanganan & tautan penggantian per kamera (Koreksi #5)
DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN
        SELECT c.conname
        FROM pg_constraint c
        JOIN pg_class t ON c.conrelid = t.oid
        WHERE t.relname = 'cctv_incident_cameras'
          AND c.contype = 'c'
          AND pg_get_constraintdef(c.oid) ILIKE '%connection_state%'
    LOOP
        EXECUTE format('ALTER TABLE cctv_incident_cameras DROP CONSTRAINT %I', r.conname);
    END LOOP;
END $$;

ALTER TABLE cctv_incident_cameras
    ADD CONSTRAINT chk_cctv_incident_cameras_state
        CHECK (connection_state IN ('offline', 'recovering', 'recovered', 'unknown', 'escalated'));

ALTER TABLE cctv_incident_cameras
    ADD COLUMN IF NOT EXISTS camera_code VARCHAR(50) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS location_area VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS source_ticket_number VARCHAR(32) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS first_failed_observed_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS confirmed_offline_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS recovered_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS member_outcome VARCHAR(32) NOT NULL DEFAULT 'pending',
    ADD COLUMN IF NOT EXISTS damage_findings TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS field_maintenance_log_id UUID NULL REFERENCES maintenance_logs(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS replacement_maintenance_log_id UUID NULL REFERENCES maintenance_logs(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS outcome_recorded_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS outcome_recorded_by UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS last_evidence JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE cctv_incident_cameras
    ADD CONSTRAINT chk_incident_member_outcome
        CHECK (member_outcome IN (
            'pending',
            'self_recovered',
            'recovered_verified',
            'escalated_replacement',
            'merged_transferred'
        ));

-- 1.6 Riwayat event & jejak audit per insiden
CREATE TABLE IF NOT EXISTS cctv_incident_events (
    id UUID PRIMARY KEY,
    incident_id UUID NOT NULL REFERENCES cctv_incidents(id) ON DELETE CASCADE,
    cctv_id UUID NULL REFERENCES cctvs(id) ON DELETE SET NULL,
    actor_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    event_type VARCHAR(48) NOT NULL,
    summary TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_incident_events_incident
    ON cctv_incident_events(incident_id, created_at ASC);

-- 1.7 Episode gangguan pemantau (unknown) untuk deduplikasi 1 notifikasi per kejadian (Keputusan #6)
CREATE TABLE IF NOT EXISTS monitor_health_episodes (
    id UUID PRIMARY KEY,
    site_id UUID NULL REFERENCES sites(id) ON DELETE CASCADE,
    cause VARCHAR(120) NOT NULL,
    detail TEXT NOT NULL DEFAULT '',
    affected_camera_count INTEGER NOT NULL DEFAULT 0,
    status VARCHAR(24) NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'resolved')),
    consecutive_healthy_cycles INTEGER NOT NULL DEFAULT 0,
    started_at TIMESTAMPTZ NOT NULL,
    last_seen_at TIMESTAMPTZ NOT NULL,
    resolved_at TIMESTAMPTZ NULL,
    alert_outbox_id UUID NULL,
    recovery_outbox_id UUID NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_uq_open_monitor_episode
    ON monitor_health_episodes(COALESCE(site_id, '00000000-0000-0000-0000-000000000000'::uuid))
    WHERE status = 'open';

-- 1.8 Perluas telegram_outbox (dukung episode pemantau, target_channel, batching, dan SLA breach)
ALTER TABLE telegram_outbox
    ALTER COLUMN incident_id DROP NOT NULL;

ALTER TABLE telegram_outbox
    ADD COLUMN IF NOT EXISTS monitor_episode_id UUID NULL REFERENCES monitor_health_episodes(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS target_channel VARCHAR(24) NOT NULL DEFAULT 'ops_group',
    ADD COLUMN IF NOT EXISTS coalesce_until TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS batch_id UUID NULL,
    ADD COLUMN IF NOT EXISTS last_attempt_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS failed_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS sla_deadline_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS sla_breached BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS sla_breached_at TIMESTAMPTZ NULL;

ALTER TABLE telegram_outbox
    ADD CONSTRAINT chk_telegram_outbox_channel
        CHECK (target_channel IN ('ops_group', 'admin_group'));

ALTER TABLE monitor_health_episodes
    ADD CONSTRAINT fk_monitor_episode_alert_outbox
        FOREIGN KEY (alert_outbox_id) REFERENCES telegram_outbox(id) ON DELETE SET NULL,
    ADD CONSTRAINT fk_monitor_episode_recovery_outbox
        FOREIGN KEY (recovery_outbox_id) REFERENCES telegram_outbox(id) ON DELETE SET NULL;

-- 1.9 Telemetri konsumsi byte & masa hidup sesi RTSP/GSM (Bagian 2.4)
CREATE TABLE IF NOT EXISTS cctv_probe_telemetry (
    id UUID PRIMARY KEY,
    cctv_id UUID NOT NULL REFERENCES cctvs(id) ON DELETE CASCADE,
    site_id UUID NULL REFERENCES sites(id) ON DELETE SET NULL,
    network_type VARCHAR(20) NOT NULL DEFAULT 'unknown',
    ip_nvr VARCHAR(45) NOT NULL DEFAULT '',
    state VARCHAR(24) NOT NULL,
    method VARCHAR(32) NOT NULL DEFAULT '',
    verify_duration_ms BIGINT NOT NULL DEFAULT 0,
    source_lifetime_ms BIGINT NOT NULL DEFAULT 0,
    bytes_verify BIGINT NOT NULL DEFAULT 0,
    bytes_total_session BIGINT NOT NULL DEFAULT 0,
    byte_delta BIGINT NOT NULL DEFAULT 0,
    source_closed_confirmed BOOLEAN NOT NULL DEFAULT false,
    forced_close BOOLEAN NOT NULL DEFAULT false,
    remove_path_status VARCHAR(24) NOT NULL DEFAULT 'not_needed',
    remove_path_error TEXT NOT NULL DEFAULT '',
    reregister_status VARCHAR(24) NOT NULL DEFAULT 'not_needed',
    reregister_error TEXT NOT NULL DEFAULT '',
    close_verify_status VARCHAR(32) NOT NULL DEFAULT 'unverified',
    close_unconfirmed_reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_probe_telemetry_cctv_time
    ON cctv_probe_telemetry(cctv_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_probe_telemetry_site_net
    ON cctv_probe_telemetry(site_id, network_type, created_at DESC);

-- ============================================================================
-- LANGKAH 2: DETERMINISTIC BACKFILL UNTUK BARIS TIKET LAMA SKEMA 000031 (Koreksi #4)
-- ============================================================================

WITH ordered_legacy AS (
    SELECT
        i.id,
        i.opened_at,
        ROW_NUMBER() OVER (
            PARTITION BY to_char(i.opened_at AT TIME ZONE 'Asia/Makassar', 'YYYYMMDD')
            ORDER BY i.opened_at ASC, i.id ASC
        ) AS seq,
        (
            SELECT MIN(m.last_observed_at)
            FROM cctv_incident_cameras m
            WHERE m.incident_id = i.id AND m.last_observed_at IS NOT NULL
        ) AS earliest_member_observed
    FROM cctv_incidents i
    WHERE i.ticket_number IS NULL
)
UPDATE cctv_incidents tgt
SET
    ticket_number = 'INC-LEGACY-' || to_char(ol.opened_at AT TIME ZONE 'Asia/Makassar', 'YYYYMMDD') || '-' || lpad(ol.seq::text, 4, '0'),
    first_failed_observed_at = COALESCE(tgt.first_failed_observed_at, ol.opened_at),
    confirmed_offline_at = COALESCE(tgt.confirmed_offline_at, ol.earliest_member_observed, ol.opened_at),
    sla_clock_start_at = COALESCE(tgt.sla_clock_start_at, ol.opened_at),
    sla_clock_basis = COALESCE(tgt.sla_clock_basis, 'legacy_opened_at'),
    timestamp_source = COALESCE(tgt.timestamp_source, 'legacy_backfill_opened_at'),
    resolution_type = CASE
        WHEN tgt.status = 'resolved' AND tgt.resolution_type IS NULL AND tgt.assigned_to IS NULL THEN 'self_recovered'
        WHEN tgt.status = 'resolved' AND tgt.resolution_type IS NULL THEN 'repaired_verified'
        ELSE tgt.resolution_type
    END
FROM ordered_legacy ol
WHERE tgt.id = ol.id;

UPDATE cctv_incidents
SET
    first_failed_observed_at = COALESCE(first_failed_observed_at, opened_at),
    confirmed_offline_at = COALESCE(confirmed_offline_at, opened_at),
    sla_clock_start_at = COALESCE(sla_clock_start_at, opened_at),
    sla_clock_basis = COALESCE(sla_clock_basis, 'legacy_opened_at'),
    timestamp_source = COALESCE(timestamp_source, 'legacy_backfill_opened_at')
WHERE first_failed_observed_at IS NULL
   OR confirmed_offline_at IS NULL
   OR sla_clock_start_at IS NULL
   OR sla_clock_basis IS NULL
   OR timestamp_source IS NULL;

UPDATE cctv_incident_cameras m
SET
    camera_code = CASE WHEN m.camera_code = '' THEN COALESCE(c.kode_cctv, '') ELSE m.camera_code END,
    location_area = CASE WHEN m.location_area = '' THEN COALESCE(c.lokasi_area, '') ELSE m.location_area END,
    first_failed_observed_at = COALESCE(m.first_failed_observed_at, i.first_failed_observed_at),
    confirmed_offline_at = COALESCE(m.confirmed_offline_at, m.last_observed_at, i.confirmed_offline_at),
    member_outcome = CASE
        WHEN i.status = 'resolved' AND m.member_outcome = 'pending' AND i.assigned_to IS NULL THEN 'self_recovered'
        WHEN i.status = 'resolved' AND m.member_outcome = 'pending' THEN 'recovered_verified'
        ELSE m.member_outcome
    END,
    outcome_recorded_at = CASE
        WHEN i.status = 'resolved' AND m.outcome_recorded_at IS NULL THEN i.resolved_at
        ELSE m.outcome_recorded_at
    END,
    outcome_recorded_by = CASE
        WHEN i.status = 'resolved' AND m.outcome_recorded_by IS NULL THEN i.resolved_by
        ELSE m.outcome_recorded_by
    END
FROM cctv_incidents i, cctvs c
WHERE m.incident_id = i.id AND m.cctv_id = c.id;

UPDATE maintenance_logs ml
SET
    incident_id = i.id,
    work_scope = CASE WHEN ml.work_scope = 'routine' THEN 'incident_camera' ELSE ml.work_scope END
FROM cctv_incidents i
WHERE i.maintenance_log_id = ml.id AND ml.incident_id IS NULL;

UPDATE telegram_outbox
SET
    coalesce_until = COALESCE(coalesce_until, created_at),
    failed_at = CASE WHEN status = 'failed' AND failed_at IS NULL THEN updated.ref_time ELSE failed_at END
FROM (
    SELECT id, COALESCE(sent_at, next_attempt_at, created_at) AS ref_time
    FROM telegram_outbox
) AS updated
WHERE telegram_outbox.id = updated.id
  AND (telegram_outbox.coalesce_until IS NULL OR (telegram_outbox.status = 'failed' AND telegram_outbox.failed_at IS NULL));

UPDATE cctv_incidents i
SET
    assignment_reminder_queued_at = COALESCE(i.assignment_reminder_queued_at, o.created_at),
    assignment_reminder_outbox_id = COALESCE(i.assignment_reminder_outbox_id, o.id)
FROM telegram_outbox o
WHERE o.incident_id = i.id
  AND o.event_type = 'INCIDENT_ASSIGN_REMINDER'
  AND i.assignment_reminder_queued_at IS NULL;

UPDATE cctv_incidents i
SET
    unrecovered_escalation_queued_at = COALESCE(i.unrecovered_escalation_queued_at, o.created_at),
    unrecovered_escalation_outbox_id = COALESCE(i.unrecovered_escalation_outbox_id, o.id)
FROM telegram_outbox o
WHERE o.incident_id = i.id
  AND o.event_type = 'INCIDENT_UNRECOVERED_ESCALATION'
  AND i.unrecovered_escalation_queued_at IS NULL;

-- ============================================================================
-- LANGKAH 3: ENFORCE NOT NULL, CHECK, & UNIQUE CONSTRAINTS
-- ============================================================================

ALTER TABLE cctv_incidents
    ALTER COLUMN ticket_number SET NOT NULL,
    ALTER COLUMN first_failed_observed_at SET NOT NULL,
    ALTER COLUMN confirmed_offline_at SET NOT NULL,
    ALTER COLUMN sla_clock_start_at SET NOT NULL,
    ALTER COLUMN sla_clock_basis SET NOT NULL,
    ALTER COLUMN timestamp_source SET NOT NULL;

ALTER TABLE cctv_incidents
    ADD CONSTRAINT chk_cctv_incidents_sla_basis
        CHECK (sla_clock_basis IN ('first_failed_observation', 'planned_window_end', 'legacy_opened_at')),
    ADD CONSTRAINT chk_cctv_incidents_ts_source
        CHECK (timestamp_source IN ('live_observation', 'legacy_backfill_opened_at'));

CREATE UNIQUE INDEX IF NOT EXISTS idx_uq_cctv_incidents_ticket_number
    ON cctv_incidents(ticket_number);

CREATE UNIQUE INDEX IF NOT EXISTS idx_uq_open_shared_incident
    ON cctv_incidents(site_id, correlation_key)
    WHERE correlation_key <> '' AND status NOT IN ('resolved', 'merged');

ALTER TABLE telegram_outbox
    ALTER COLUMN coalesce_until SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_uq_outbox_once_per_incident
    ON telegram_outbox(incident_id, event_type)
    WHERE incident_id IS NOT NULL
      AND event_type IN ('INCIDENT_ASSIGN_REMINDER', 'INCIDENT_UNRECOVERED_ESCALATION');

CREATE INDEX IF NOT EXISTS idx_telegram_outbox_sla_check
    ON telegram_outbox(sla_deadline_at)
    WHERE sla_deadline_at IS NOT NULL AND sla_breached = false AND sent_at IS NULL;
