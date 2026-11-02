-- M08: Cegah duplikasi penyelesaian tugas pemeliharaan akibat request bersamaan / double-submit.
-- Satu maintenance_log hanya boleh memiliki satu hasil field_maintenance aktif.
CREATE UNIQUE INDEX IF NOT EXISTS uq_field_maintenances_maintenance_log_id
ON field_maintenances (maintenance_log_id)
WHERE maintenance_log_id IS NOT NULL AND deleted_at IS NULL;
