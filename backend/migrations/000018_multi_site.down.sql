-- Migration 000018 (rollback): hapus kolom site_id lalu tabel user_sites & sites.
ALTER TABLE zones DROP COLUMN IF EXISTS site_id;
ALTER TABLE maintenance_logs DROP COLUMN IF EXISTS site_id;
ALTER TABLE cctvs DROP COLUMN IF EXISTS site_id;
DROP TABLE IF EXISTS user_sites;
DROP TABLE IF EXISTS sites;
