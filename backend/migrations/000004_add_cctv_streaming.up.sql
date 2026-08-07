-- Migration 000004: tambah kolom streaming RTSP pada tabel cctvs (Task 9.1).
ALTER TABLE cctvs ADD COLUMN IF NOT EXISTS channel_port    integer NULL;
ALTER TABLE cctvs ADD COLUMN IF NOT EXISTS rtsp_password   text    NULL;
ALTER TABLE cctvs ADD COLUMN IF NOT EXISTS stream_enabled  boolean NOT NULL DEFAULT false;
