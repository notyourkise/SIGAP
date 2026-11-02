CREATE TABLE cctv_incidents (
 id uuid PRIMARY KEY, site_id uuid NOT NULL REFERENCES sites(id),
 status varchar(32) NOT NULL CHECK(status IN ('open','in_progress','pending_verification','resolved')),
 title text NOT NULL, maintenance_log_id uuid REFERENCES maintenance_logs(id),
 assigned_to uuid REFERENCES users(id), opened_at timestamptz NOT NULL,
 updated_at timestamptz NOT NULL, resolved_at timestamptz,
 resolved_by uuid REFERENCES users(id), resolution_note text NOT NULL DEFAULT ''
);
CREATE INDEX idx_incidents_site_open ON cctv_incidents(site_id, opened_at DESC);
CREATE TABLE cctv_incident_cameras (
 id uuid PRIMARY KEY, incident_id uuid NOT NULL REFERENCES cctv_incidents(id),
 cctv_id uuid NOT NULL REFERENCES cctvs(id), camera_name text NOT NULL,
 is_active_member boolean NOT NULL DEFAULT true,
 connection_state varchar(24) NOT NULL CHECK(connection_state IN ('offline','recovering','recovered','unknown')),
 last_observed_at timestamptz, recovery_started_at timestamptz,
 consecutive_live_count integer NOT NULL DEFAULT 0,
 UNIQUE(incident_id,cctv_id)
);
CREATE UNIQUE INDEX idx_camera_one_active_incident ON cctv_incident_cameras(cctv_id) WHERE is_active_member;
CREATE INDEX idx_incident_members ON cctv_incident_cameras(incident_id);
CREATE TABLE telegram_outbox (
 id uuid PRIMARY KEY, incident_id uuid NOT NULL REFERENCES cctv_incidents(id),
 site_id uuid NOT NULL REFERENCES sites(id), event_type varchar(40) NOT NULL,
 text text NOT NULL,
 status varchar(24) NOT NULL CHECK(status IN ('pending','sending','sent','ambiguous','failed')),
 attempts integer NOT NULL DEFAULT 0, next_attempt_at timestamptz NOT NULL,
 lease_until timestamptz, message_id bigint NOT NULL DEFAULT 0,
 created_at timestamptz NOT NULL, sent_at timestamptz, last_error text NOT NULL DEFAULT ''
);
CREATE INDEX idx_telegram_outbox_due ON telegram_outbox(next_attempt_at) WHERE status IN ('pending','ambiguous','sending');
