-- Migration 000021 (rollback): hapus tabel event deteksi AI.
DROP TABLE IF EXISTS ai_detection_events;
