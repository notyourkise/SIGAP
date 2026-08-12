-- Migration 000020: tabel ai_detection_sessions (Task 14B, Sprint 38).
-- Menyimpan ringkasan sesi deteksi AI harian per CCTV: jumlah orang unik
-- yang terdeteksi sepanjang hari (unique_person_count) beserta daftar
-- tracking id terakhir (last_tracking_ids, jsonb). Satu baris per
-- (cctv_id, session_date) — di-upsert oleh backend setiap ada event deteksi.
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
