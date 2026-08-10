-- Migration 000011: tambah kolom rtsp_url (URL RTSP penuh per-CCTV).
-- Bila diisi, URL ini dipakai langsung sebagai sumber stream MediaMTX,
-- menggantikan penyusunan otomatis dari channel_port + kredensial NVR global.
ALTER TABLE cctvs ADD COLUMN IF NOT EXISTS rtsp_url text NULL;
