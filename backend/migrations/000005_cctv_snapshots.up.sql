-- Migration 000005: tabel cctv_snapshots (Task 10.1/10.2).
CREATE TABLE IF NOT EXISTS cctv_snapshots (
    id            uuid         PRIMARY KEY,
    created_at    timestamptz  NOT NULL,
    updated_at    timestamptz  NOT NULL,
    deleted_at    timestamptz  NULL,
    cctv_id       uuid         NOT NULL,
    event_type    varchar(50)  NOT NULL,
    image_path    varchar(255) NULL,
    notes         text         NULL
);
CREATE INDEX IF NOT EXISTS idx_cctv_snapshots_cctv_id ON cctv_snapshots (cctv_id);
CREATE INDEX IF NOT EXISTS idx_cctv_snapshots_deleted_at ON cctv_snapshots (deleted_at);
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_cctv_snapshots_cctv') THEN
        ALTER TABLE cctv_snapshots ADD CONSTRAINT fk_cctv_snapshots_cctv
            FOREIGN KEY (cctv_id) REFERENCES cctvs (id) ON DELETE CASCADE;
    END IF;
END $$;
