-- Migration 000020 (rollback): hapus tabel sesi deteksi AI.
DROP TABLE IF EXISTS ai_detection_sessions;
