-- Migration 000024: tambah kolom site_id (kawasan) pada checklist_templates
-- (Sprint 40, perbaikan: daftar checklist kini dapat berbeda per kawasan).
-- Nilai NULL = template global (berlaku untuk semua kawasan).
ALTER TABLE checklist_templates
  ADD COLUMN IF NOT EXISTS site_id uuid REFERENCES sites(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_checklist_templates_site ON checklist_templates (site_id);
