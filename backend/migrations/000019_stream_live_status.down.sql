-- Migration 000019 (rollback): hapus kolom status live stream.
ALTER TABLE cctvs DROP COLUMN IF EXISTS stream_live, DROP COLUMN IF EXISTS stream_checked_at;
