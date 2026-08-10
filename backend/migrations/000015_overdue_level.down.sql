-- Migration 000015 (rollback): hapus kolom overdue_level.
ALTER TABLE maintenance_logs DROP COLUMN IF EXISTS overdue_level;
