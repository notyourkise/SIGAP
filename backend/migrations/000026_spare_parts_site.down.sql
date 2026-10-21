DROP INDEX IF EXISTS idx_spare_parts_site_id;
ALTER TABLE spare_parts DROP COLUMN IF EXISTS site_id;
