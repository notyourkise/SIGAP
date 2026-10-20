-- Migration 000025: kaitkan inventaris suku cadang dengan kawasan operasional.
ALTER TABLE spare_parts
  ADD COLUMN IF NOT EXISTS site_id uuid REFERENCES sites(id) ON DELETE SET NULL;

-- Data lama ditempatkan pada kawasan default agar tetap langsung terlihat dan
-- dapat difilter setelah migrasi, tanpa menghapus atau membuat ulang record.
UPDATE spare_parts
SET site_id = (
  SELECT id FROM sites
  WHERE is_default = true AND deleted_at IS NULL
  ORDER BY created_at ASC
  LIMIT 1
)
WHERE site_id IS NULL;

CREATE INDEX IF NOT EXISTS idx_spare_parts_site_id ON spare_parts(site_id);
