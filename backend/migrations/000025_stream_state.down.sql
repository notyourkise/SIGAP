-- Rollback migration 000025: hapus status stream empat keadaan.
ALTER TABLE cctvs
  DROP COLUMN IF EXISTS stream_state;
