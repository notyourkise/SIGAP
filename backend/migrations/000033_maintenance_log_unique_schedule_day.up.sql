-- M09: Cegah duplikasi pembuatan log pemeliharaan oleh generator/worker bersamaan.
-- Satu jadwal pemeliharaan (maintenance_schedule_id) hanya boleh menghasilkan maksimal
-- satu log pemeliharaan per tanggal (UTC date).
CREATE UNIQUE INDEX IF NOT EXISTS uq_maintenance_logs_schedule_day
ON maintenance_logs (maintenance_schedule_id, CAST(scheduled_date AT TIME ZONE 'UTC' AS date))
WHERE maintenance_schedule_id IS NOT NULL;
