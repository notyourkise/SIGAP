-- Migration 000022: field ping latensi & last-active stream (Sprint 37-PATCH).
-- Menambahkan kolom hasil probe Metro-E (ping_latency_ms) dan jejak terakhir
-- stream aktif untuk kamera GSM (stream_last_active_at) pada tabel cctvs.
ALTER TABLE cctvs
  ADD COLUMN IF NOT EXISTS ping_latency_ms     INTEGER,
  ADD COLUMN IF NOT EXISTS stream_last_active_at TIMESTAMPTZ;
