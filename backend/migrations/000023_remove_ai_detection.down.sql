-- Migration 000023 (rollback): kembalikan tabel AI detection (Task 14B, Sprint 38).
-- Urutan: sessions dulu (referensi cctvs), baru events.
CREATE TABLE IF NOT EXISTS ai_detection_sessions (
    id                   uuid         PRIMARY KEY DEFAULT gen_random_uuid(),
    cctv_id              uuid         NOT NULL REFERENCES cctvs(id) ON DELETE CASCADE,
    session_date         date         NOT NULL,
    unique_person_count  integer      NOT NULL DEFAULT 0,
    last_tracking_ids    jsonb,
    created_at           timestamptz  NOT NULL DEFAULT NOW(),
    updated_at           timestamptz  NOT NULL DEFAULT NOW(),
    UNIQUE (cctv_id, session_date)
);
CREATE INDEX IF NOT EXISTS idx_ai_sessions_cctv_date ON ai_detection_sessions (cctv_id, session_date);

CREATE TABLE IF NOT EXISTS ai_detection_events (
    id              uuid         PRIMARY KEY DEFAULT gen_random_uuid(),
    cctv_id         uuid         NOT NULL REFERENCES cctvs(id) ON DELETE CASCADE,
    event_type      varchar(50)  NOT NULL DEFAULT 'person_detected',
    person_count    integer      NOT NULL DEFAULT 0,
    unique_count    integer      NOT NULL DEFAULT 0,
    confidence_avg  double precision,
    detected_at     timestamptz  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ai_events_cctv_detected ON ai_detection_events (cctv_id, detected_at DESC);
