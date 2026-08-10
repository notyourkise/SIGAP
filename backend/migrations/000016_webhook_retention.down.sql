-- Migration 000016 (rollback): hapus kolom max_log_retention_days.
ALTER TABLE webhook_endpoints DROP COLUMN IF EXISTS max_log_retention_days;
