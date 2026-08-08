-- Migration 000006: tabel spare_parts & spare_part_usages (Task 11.1, Sprint 26).
-- Inventaris suku cadang untuk perbaikan CCTV, terintegrasi dengan maintenance log.

-- Master suku cadang / komponen.
CREATE TABLE IF NOT EXISTS spare_parts (
    id             uuid          PRIMARY KEY,
    created_at     timestamptz   NOT NULL,
    updated_at     timestamptz   NOT NULL,
    deleted_at     timestamptz   NULL,
    nama           varchar(100)  NOT NULL,
    sku            varchar(50)   NOT NULL,
    kategori       varchar(50)   NULL,
    stok           integer       NOT NULL DEFAULT 0,
    minimum_order  integer       NOT NULL DEFAULT 5,
    satuan         varchar(20)   NOT NULL DEFAULT 'pcs',
    harga_satuan   numeric(12,2) NULL DEFAULT 0,
    keterangan     text          NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_spare_parts_sku ON spare_parts (sku);
CREATE INDEX IF NOT EXISTS idx_spare_parts_deleted_at ON spare_parts (deleted_at);

-- Pencatatan penggunaan suku cadang pada maintenance log.
CREATE TABLE IF NOT EXISTS spare_part_usages (
    id                   uuid        PRIMARY KEY,
    created_at           timestamptz NOT NULL,
    updated_at           timestamptz NOT NULL,
    deleted_at           timestamptz NULL,
    maintenance_log_id   uuid        NOT NULL,
    spare_part_id        uuid        NOT NULL,
    jumlah               integer     NOT NULL,
    catatan              text        NULL
);
CREATE INDEX IF NOT EXISTS idx_spare_part_usages_maintenance_log_id ON spare_part_usages (maintenance_log_id);
CREATE INDEX IF NOT EXISTS idx_spare_part_usages_spare_part_id ON spare_part_usages (spare_part_id);
CREATE INDEX IF NOT EXISTS idx_spare_part_usages_deleted_at ON spare_part_usages (deleted_at);

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_spare_part_usages_maintenance_log') THEN
        ALTER TABLE spare_part_usages ADD CONSTRAINT fk_spare_part_usages_maintenance_log
            FOREIGN KEY (maintenance_log_id) REFERENCES maintenance_logs (id) ON DELETE CASCADE;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_spare_part_usages_spare_part') THEN
        ALTER TABLE spare_part_usages ADD CONSTRAINT fk_spare_part_usages_spare_part
            FOREIGN KEY (spare_part_id) REFERENCES spare_parts (id) ON DELETE CASCADE;
    END IF;
END $$;
