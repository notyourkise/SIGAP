-- Migration 000017 (rollback): hapus kolom wajib pada field_maintenance_checks
-- lalu hapus tabel checklist_templates.
ALTER TABLE field_maintenance_checks DROP COLUMN IF EXISTS wajib;
DROP TABLE IF EXISTS checklist_templates;
