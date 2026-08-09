-- Migration 000009: Zona & SLA (Task 11.4 + 11.5, Sprint 28).
-- Menambah master zona, join table user_zones, kebijakan SLA, dan kolom FK
-- zone_id pada cctvs serta sla_policy_id pada maintenance_schedules.

-- Master zona operasional (wilayah penempatan CCTV & teknisi).
CREATE TABLE IF NOT EXISTS zones (
    id          uuid          PRIMARY KEY,
    created_at  timestamptz   NOT NULL,
    updated_at  timestamptz   NOT NULL,
    deleted_at  timestamptz   NULL,
    nama        varchar(100)  NOT NULL,
    deskripsi   text          NULL
);
CREATE INDEX IF NOT EXISTS idx_zones_deleted_at ON zones (deleted_at);

-- Join table many-to-many User ↔ Zone (penugasan teknisi ke zona).
CREATE TABLE IF NOT EXISTS user_zones (
    zone_id  uuid  NOT NULL,
    user_id  uuid  NOT NULL,
    PRIMARY KEY (zone_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_user_zones_user ON user_zones (user_id);
CREATE INDEX IF NOT EXISTS idx_user_zones_zone ON user_zones (zone_id);

-- Master kebijakan SLA (target waktu respons & penyelesaian per jenis tugas).
CREATE TABLE IF NOT EXISTS sla_policies (
    id                 uuid          PRIMARY KEY,
    created_at         timestamptz   NOT NULL,
    updated_at         timestamptz   NOT NULL,
    deleted_at         timestamptz   NULL,
    nama               varchar(100)  NOT NULL,
    jenis_tugas        varchar(30)   NOT NULL,
    target_jam_respon  integer       NOT NULL DEFAULT 1,
    target_jam_selesai integer       NOT NULL DEFAULT 8
);
CREATE INDEX IF NOT EXISTS idx_sla_policies_deleted_at ON sla_policies (deleted_at);

-- Kolom FK zone_id pada cctvs (one-to-many CCTV → Zone).
ALTER TABLE cctvs ADD COLUMN IF NOT EXISTS zone_id uuid NULL;
CREATE INDEX IF NOT EXISTS idx_cctvs_zone_id ON cctvs (zone_id);

-- Kolom FK sla_policy_id pada maintenance_schedules.
ALTER TABLE maintenance_schedules ADD COLUMN IF NOT EXISTS sla_policy_id uuid NULL;
CREATE INDEX IF NOT EXISTS idx_maintenance_schedules_sla_policy_id ON maintenance_schedules (sla_policy_id);

DO $$
BEGIN
    -- FK user_zones → zones
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_user_zones_zone') THEN
        ALTER TABLE user_zones ADD CONSTRAINT fk_user_zones_zone
            FOREIGN KEY (zone_id) REFERENCES zones (id) ON DELETE CASCADE;
    END IF;
    -- FK user_zones → users
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_user_zones_user') THEN
        ALTER TABLE user_zones ADD CONSTRAINT fk_user_zones_user
            FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE;
    END IF;
    -- FK cctvs.zone_id → zones
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_cctvs_zone') THEN
        ALTER TABLE cctvs ADD CONSTRAINT fk_cctvs_zone
            FOREIGN KEY (zone_id) REFERENCES zones (id) ON DELETE SET NULL;
    END IF;
    -- FK maintenance_schedules.sla_policy_id → sla_policies
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_maintenance_schedules_sla_policy') THEN
        ALTER TABLE maintenance_schedules ADD CONSTRAINT fk_maintenance_schedules_sla_policy
            FOREIGN KEY (sla_policy_id) REFERENCES sla_policies (id) ON DELETE SET NULL;
    END IF;
END $$;
