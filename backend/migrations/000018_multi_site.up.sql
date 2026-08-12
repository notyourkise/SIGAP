-- Migration 000018: Multi-site (Task 12.2, Sprint 30).
-- Menambahkan kawasan operasional (site) sebagai kelompok data CCTV/log/zona.
-- Data existing otomatis dimasukkan ke site default "OIKN". ID site default
-- memakai gen_random_uuid() karena ini data SQL murni (bukan dibuat aplikasi).
CREATE TABLE IF NOT EXISTS sites (
  id uuid PRIMARY KEY,
  nama varchar(100) NOT NULL,
  kode varchar(20) NOT NULL UNIQUE,
  alamat text,
  koordinat_lat double precision,
  koordinat_lng double precision,
  is_default boolean NOT NULL DEFAULT FALSE,
  created_at timestamptz NOT NULL DEFAULT NOW(),
  updated_at timestamptz NOT NULL DEFAULT NOW(),
  deleted_at timestamptz
);

CREATE TABLE IF NOT EXISTS user_sites (
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  site_id uuid NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
  PRIMARY KEY (user_id, site_id)
);
CREATE INDEX IF NOT EXISTS idx_user_sites_user ON user_sites (user_id);
CREATE INDEX IF NOT EXISTS idx_user_sites_site ON user_sites (site_id);

-- Site default OIKN (idempoten)
INSERT INTO sites (id, nama, kode, alamat, is_default)
SELECT gen_random_uuid(), 'Kawasan OIKN', 'OIKN', 'Ibu Kota Nusantara, Kalimantan Timur', TRUE
WHERE NOT EXISTS (SELECT 1 FROM sites WHERE kode = 'OIKN');

-- Tambah kolom site_id ke tabel terkait (idempoten)
ALTER TABLE cctvs ADD COLUMN IF NOT EXISTS site_id uuid REFERENCES sites(id) ON DELETE SET NULL;
ALTER TABLE maintenance_logs ADD COLUMN IF NOT EXISTS site_id uuid REFERENCES sites(id) ON DELETE SET NULL;
ALTER TABLE zones ADD COLUMN IF NOT EXISTS site_id uuid REFERENCES sites(id) ON DELETE SET NULL;

-- Set default site OIKN ke semua data existing (idempoten, hanya yang masih NULL)
UPDATE cctvs SET site_id = (SELECT id FROM sites WHERE kode = 'OIKN') WHERE site_id IS NULL;
UPDATE maintenance_logs SET site_id = (SELECT id FROM sites WHERE kode = 'OIKN') WHERE site_id IS NULL;
UPDATE zones SET site_id = (SELECT id FROM sites WHERE kode = 'OIKN') WHERE site_id IS NULL;
