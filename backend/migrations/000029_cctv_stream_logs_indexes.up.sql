-- Index komposit untuk query daftar dan ringkasan Log CCTV (P1 #144).
-- Index kolom tunggal cctv_id/event_type sudah tercakup sebagai prefix index
-- komposit, sehingga dihapus agar tidak menambah biaya tulis.
DROP INDEX IF EXISTS idx_cctv_stream_logs_cctv_id;
DROP INDEX IF EXISTS idx_cctv_stream_logs_event_type;

CREATE INDEX IF NOT EXISTS idx_cctv_stream_logs_cctv_created
    ON cctv_stream_logs(cctv_id, created_at);

CREATE INDEX IF NOT EXISTS idx_cctv_stream_logs_event_created
    ON cctv_stream_logs(event_type, created_at);
