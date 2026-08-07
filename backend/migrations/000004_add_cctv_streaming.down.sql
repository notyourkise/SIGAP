-- Migration 000004 (rollback): hapus kolom streaming RTSP pada tabel cctvs (Task 9.1).
ALTER TABLE cctvs DROP COLUMN IF EXISTS channel_port;
ALTER TABLE cctvs DROP COLUMN IF EXISTS rtsp_password;
ALTER TABLE cctvs DROP COLUMN IF EXISTS stream_enabled;
