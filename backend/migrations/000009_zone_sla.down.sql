-- Rollback Migration 000009: hapus kolom FK, join table, dan tabel zona/SLA.
ALTER TABLE maintenance_schedules DROP COLUMN IF EXISTS sla_policy_id;
ALTER TABLE cctvs DROP COLUMN IF EXISTS zone_id;

DROP TABLE IF EXISTS user_zones;
DROP TABLE IF EXISTS sla_policies;
DROP TABLE IF EXISTS zones;
