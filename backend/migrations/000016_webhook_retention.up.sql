-- Migration 000016: retensi log webhook delivery (Task 13.10, Sprint 34).
-- Menambah kolom max_log_retention_days pada webhook_endpoints agar tiap
-- endpoint dapat mengatur berapa lama log pengiriman disimpan (default 30 hari).
ALTER TABLE webhook_endpoints
  ADD COLUMN IF NOT EXISTS max_log_retention_days INTEGER NOT NULL DEFAULT 30;
