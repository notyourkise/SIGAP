-- Kembalikan index kolom tunggal bila migration #144 di-rollback.
DROP INDEX IF EXISTS idx_cctv_stream_logs_event_created;
DROP INDEX IF EXISTS idx_cctv_stream_logs_cctv_created;

CREATE INDEX IF NOT EXISTS idx_cctv_stream_logs_cctv_id ON cctv_stream_logs(cctv_id);
CREATE INDEX IF NOT EXISTS idx_cctv_stream_logs_event_type ON cctv_stream_logs(event_type);
