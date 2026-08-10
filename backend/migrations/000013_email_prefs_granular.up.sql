-- Migration 000013: preferensi email granular per-tipe (Task 13.7, Sprint 32).
ALTER TABLE users
  ADD COLUMN IF NOT EXISTS email_daily_digest BOOLEAN NOT NULL DEFAULT TRUE,
  ADD COLUMN IF NOT EXISTS email_overdue BOOLEAN NOT NULL DEFAULT TRUE,
  ADD COLUMN IF NOT EXISTS email_stock_low BOOLEAN NOT NULL DEFAULT TRUE;

-- Sync nilai awal dengan email_notifications existing.
UPDATE users SET
  email_daily_digest = email_notifications,
  email_overdue = email_notifications,
  email_stock_low = email_notifications;
