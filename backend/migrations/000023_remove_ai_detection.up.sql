-- Migration 000023: hapus tabel AI detection (Sprint 40 — AI Feature Takedown).
-- Urutan: events dulu (FK ke sessions), baru sessions.
DROP TABLE IF EXISTS ai_detection_events;
DROP TABLE IF EXISTS ai_detection_sessions;
