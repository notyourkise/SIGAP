-- Migration 000024 (rollback): hapus kolom site_id dari checklist_templates.
ALTER TABLE checklist_templates
  DROP COLUMN IF EXISTS site_id;
