-- Migration 000021: tabel ai_detection_events (Task 14B, Sprint 38).
-- Mencatat setiap event deteksi AI yang dikirim service sigap_ai (satu baris
-- per frame/kejadian dengan jumlah orang di frame dan rata-rata confidence).
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
