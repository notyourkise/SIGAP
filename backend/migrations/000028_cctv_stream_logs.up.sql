-- Tabel cctv_stream_logs untuk mencatat riwayat event Online/Offline CCTV (Sprint 42).
CREATE TABLE IF NOT EXISTS cctv_stream_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    cctv_id UUID NOT NULL REFERENCES cctvs(id) ON DELETE CASCADE,
    event_type VARCHAR(32) NOT NULL,
    previous_state VARCHAR(32) NOT NULL DEFAULT '',
    current_state VARCHAR(32) NOT NULL,
    trigger_source VARCHAR(32) NOT NULL DEFAULT 'probe',
    network_type VARCHAR(20) NOT NULL DEFAULT 'gsm',
    latency_ms INTEGER,
    error_detail TEXT NOT NULL DEFAULT '',
    message TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_cctv_stream_logs_cctv_id ON cctv_stream_logs(cctv_id);
CREATE INDEX IF NOT EXISTS idx_cctv_stream_logs_created_at ON cctv_stream_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_cctv_stream_logs_event_type ON cctv_stream_logs(event_type);
