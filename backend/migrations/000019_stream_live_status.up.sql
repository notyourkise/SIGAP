-- Migration 000019: status live stream (Task 14A.1, Sprint 37).
-- Menambahkan kolom hasil probe RTSP live (stream_live) beserta waktu
-- pemeriksaan terakhir (stream_checked_at) pada tabel cctvs.
ALTER TABLE cctvs
  ADD COLUMN IF NOT EXISTS stream_live BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS stream_checked_at TIMESTAMPTZ;
