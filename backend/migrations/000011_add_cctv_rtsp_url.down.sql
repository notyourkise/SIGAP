-- Migration 000011 (rollback): hapus kolom rtsp_url pada tabel cctvs.
ALTER TABLE cctvs DROP COLUMN IF EXISTS rtsp_url;
