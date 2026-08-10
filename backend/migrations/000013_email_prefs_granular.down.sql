-- Migration 000013 (rollback): hapus kolom preferensi email granular.
ALTER TABLE users
  DROP COLUMN IF EXISTS email_daily_digest,
  DROP COLUMN IF EXISTS email_overdue,
  DROP COLUMN IF EXISTS email_stock_low;
