-- Migration 000025: status stream empat keadaan (P1 on-demand probe).
-- stream_live tetap dipertahankan untuk kompatibilitas klien lama; stream_state
-- membedakan live, idle (on-demand tanpa penonton), offline, dan unknown.
ALTER TABLE cctvs
  ADD COLUMN IF NOT EXISTS stream_state VARCHAR(20) NOT NULL DEFAULT 'unknown';
