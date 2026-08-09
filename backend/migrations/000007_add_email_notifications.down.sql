-- Rollback Migration 000007: hapus kolom email_notifications pada users.
ALTER TABLE users DROP COLUMN IF EXISTS email_notifications;
