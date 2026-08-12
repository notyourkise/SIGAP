-- Migration 000017: tabel checklist_templates (Task 13.12, Sprint 35).
-- Menyimpan template checklist pengecekan lapangan yang dapat dikustomisasi
-- admin (nama, deskripsi, urutan, wajib, aktif) tanpa perlu deploy ulang.
-- ID di-generate di aplikasi Go via hook GORM (BeforeCreate), sehingga
-- migration ini cukup membuat tabel. Data default (9 item) di-seed melalui
-- models.SeedChecklistTemplates pada saat startup (models/seeder.go).
CREATE TABLE IF NOT EXISTS checklist_templates (
    id          uuid         PRIMARY KEY,
    created_at  timestamptz  NOT NULL,
    updated_at  timestamptz  NOT NULL,
    deleted_at  timestamptz  NULL,
    nama_item   varchar(255) NOT NULL,
    deskripsi   text         NULL,
    urutan      integer      NOT NULL DEFAULT 0,
    wajib       boolean      NOT NULL DEFAULT TRUE,
    aktif       boolean      NOT NULL DEFAULT TRUE
);
CREATE INDEX IF NOT EXISTS idx_checklist_templates_urutan ON checklist_templates (urutan);
CREATE INDEX IF NOT EXISTS idx_checklist_templates_deleted_at ON checklist_templates (deleted_at);

-- Tambahkan kolom Wajib pada tabel field_maintenance_checks (Task 13.12).
-- Menandai apakah sebuah item checklist hasil pengecekan lapangan bersifat
-- wajib. Default TRUE agar data lama tetap dianggap wajib.
ALTER TABLE field_maintenance_checks
    ADD COLUMN IF NOT EXISTS wajib boolean NOT NULL DEFAULT TRUE;
