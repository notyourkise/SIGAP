-- Migration 000008: tabel report_templates (Task 11.2, Sprint 27).
-- Menyimpan template laporan kustom yang dapat disimpan & dipakai ulang admin.
CREATE TABLE IF NOT EXISTS report_templates (
    id          uuid         PRIMARY KEY,
    created_at  timestamptz  NOT NULL,
    updated_at  timestamptz  NOT NULL,
    deleted_at  timestamptz  NULL,
    nama        varchar(100) NOT NULL,
    config      jsonb        NOT NULL,
    created_by  uuid         NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_report_templates_created_by ON report_templates (created_by);
CREATE INDEX IF NOT EXISTS idx_report_templates_deleted_at ON report_templates (deleted_at);

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_report_templates_user') THEN
        ALTER TABLE report_templates ADD CONSTRAINT fk_report_templates_user
            FOREIGN KEY (created_by) REFERENCES users (id) ON DELETE CASCADE;
    END IF;
END $$;
