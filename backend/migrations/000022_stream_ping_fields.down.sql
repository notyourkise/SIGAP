-- Migration 000022 (rollback): hapus field ping latensi & last-active stream.
ALTER TABLE cctvs
  DROP COLUMN IF EXISTS ping_latency_ms,
  DROP COLUMN IF EXISTS stream_last_active_at;
