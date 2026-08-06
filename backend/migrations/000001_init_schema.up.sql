-- Migration 000001: Skema awal (init schema) SIGAP.
-- Membuat seluruh tabel yang sebelumnya dikelola GORM AutoMigrate.
-- Kolom & constraint mencerminkan tag GORM pada models/*.go (Task 7.1).
-- Urutan pembuatan tidak menyertakan FK inline; seluruh foreign key
-- ditambahkan di akhir melalui ALTER TABLE agar tidak bertabrakan
-- dengan dependensi sirkular (maintenance_result_id ↔ maintenance_log_id).

-- Tabel users
CREATE TABLE IF NOT EXISTS users (
    id                    uuid         PRIMARY KEY,
    created_at            timestamptz  NOT NULL,
    updated_at            timestamptz  NOT NULL,
    deleted_at            timestamptz  NULL,
    username              varchar(50)  NOT NULL,
    password_hash         varchar(255) NOT NULL,
    full_name             varchar(100) NOT NULL,
    email                 varchar(100) NOT NULL,
    role                  varchar(20)  NOT NULL,
    profile_photo_url     varchar(255) NULL,
    token_version         bigint       NOT NULL DEFAULT 1,
    must_change_password  boolean      NOT NULL DEFAULT false
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON users (username);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users (email);
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users (deleted_at);

-- Tabel cctvs
CREATE TABLE IF NOT EXISTS cctvs (
    id               uuid         PRIMARY KEY,
    created_at       timestamptz  NOT NULL,
    updated_at       timestamptz  NOT NULL,
    deleted_at       timestamptz  NULL,
    kode_cctv        varchar(50)  NOT NULL,
    id_channel       varchar(50)  NULL,
    nama             varchar(100) NOT NULL,
    lokasi_area      varchar(255) NULL,
    longitude        varchar(50)  NULL,
    latitude         varchar(50)  NULL,
    jenis_cctv       varchar(50)  NULL,
    sumber_listrik   varchar(50)  NULL,
    tipe_koneksi     varchar(50)  NULL,
    indoor_outdoor   varchar(20)  NULL,
    pemilik          varchar(100) NULL,
    status           varchar(20)  NOT NULL,
    ip_nvr           varchar(45)  NULL,
    ip_camera        varchar(45)  NULL,
    keterangan       text         NULL,
    qr_payload       text         NULL,
    health_score     integer      NOT NULL DEFAULT 100
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_cctvs_kode_cctv ON cctvs (kode_cctv);
CREATE UNIQUE INDEX IF NOT EXISTS uniq_cctvs_qr_payload ON cctvs (qr_payload);
CREATE INDEX IF NOT EXISTS idx_cctvs_deleted_at ON cctvs (deleted_at);

-- Tabel maintenance_schedules
CREATE TABLE IF NOT EXISTS maintenance_schedules (
    id                 uuid         PRIMARY KEY,
    created_at         timestamptz  NOT NULL,
    updated_at         timestamptz  NOT NULL,
    deleted_at         timestamptz  NULL,
    cctv_id            uuid         NULL,
    technician_id      uuid         NULL,
    action_taken       text         NOT NULL,
    frequency          varchar(20)  NOT NULL,
    next_run_at        timestamptz  NOT NULL,
    is_active          boolean      NOT NULL DEFAULT true,
    last_generated_at  timestamptz  NULL
);

CREATE INDEX IF NOT EXISTS idx_maintenance_schedules_cctv_id ON maintenance_schedules (cctv_id);
CREATE INDEX IF NOT EXISTS idx_maintenance_schedules_technician_id ON maintenance_schedules (technician_id);
CREATE INDEX IF NOT EXISTS idx_maintenance_schedules_next_run_at ON maintenance_schedules (next_run_at);
CREATE INDEX IF NOT EXISTS idx_maintenance_schedules_deleted_at ON maintenance_schedules (deleted_at);

-- Tabel maintenance_logs
CREATE TABLE IF NOT EXISTS maintenance_logs (
    id                        uuid         PRIMARY KEY,
    created_at                timestamptz  NOT NULL,
    updated_at                timestamptz  NOT NULL,
    deleted_at                timestamptz  NULL,
    cctv_id                   uuid         NULL,
    technician_id             uuid         NULL,
    assigned_at               timestamptz  NULL,
    maintenance_result_id     uuid         NULL,
    maintenance_schedule_id   uuid         NULL,
    action_taken              text         NOT NULL,
    notes                     text         NULL,
    status                    varchar(20)  NOT NULL,
    overdue_notified_at       timestamptz  NULL,
    scheduled_date            timestamptz  NULL,
    completed_at              timestamptz  NULL,
    image_url                 varchar(255) NULL
);

CREATE INDEX IF NOT EXISTS idx_maintenance_logs_cctv_id ON maintenance_logs (cctv_id);
CREATE INDEX IF NOT EXISTS idx_maintenance_logs_technician_id ON maintenance_logs (technician_id);
CREATE INDEX IF NOT EXISTS idx_maintenance_logs_maintenance_result_id ON maintenance_logs (maintenance_result_id);
CREATE INDEX IF NOT EXISTS idx_maintenance_logs_maintenance_schedule_id ON maintenance_logs (maintenance_schedule_id);
CREATE INDEX IF NOT EXISTS idx_maintenance_logs_deleted_at ON maintenance_logs (deleted_at);

-- Tabel field_maintenances
CREATE TABLE IF NOT EXISTS field_maintenances (
    id                uuid         PRIMARY KEY,
    created_at        timestamptz  NOT NULL,
    updated_at        timestamptz  NOT NULL,
    deleted_at        timestamptz  NULL,
    cctv_id           uuid         NULL,
    technician_id     uuid         NOT NULL,
    maintenance_log_id uuid        NULL,
    overall_status    varchar(20)  NOT NULL,
    notes             text         NULL,
    completed_at      timestamptz  NULL
);

CREATE INDEX IF NOT EXISTS idx_field_maintenances_cctv_id ON field_maintenances (cctv_id);
CREATE INDEX IF NOT EXISTS idx_field_maintenances_maintenance_log_id ON field_maintenances (maintenance_log_id);
CREATE INDEX IF NOT EXISTS idx_field_maintenances_deleted_at ON field_maintenances (deleted_at);

-- Tabel field_maintenance_checks
CREATE TABLE IF NOT EXISTS field_maintenance_checks (
    id                  uuid         PRIMARY KEY,
    created_at          timestamptz  NOT NULL,
    updated_at          timestamptz  NOT NULL,
    deleted_at          timestamptz  NULL,
    field_maintenance_id uuid        NULL,
    name                varchar(150) NOT NULL,
    result              varchar(20)  NOT NULL,
    note                text         NULL
);

CREATE INDEX IF NOT EXISTS idx_field_maintenance_checks_field_maintenance_id ON field_maintenance_checks (field_maintenance_id);
CREATE INDEX IF NOT EXISTS idx_field_maintenance_checks_deleted_at ON field_maintenance_checks (deleted_at);

-- Tabel field_maintenance_images
CREATE TABLE IF NOT EXISTS field_maintenance_images (
    id                  uuid         PRIMARY KEY,
    created_at          timestamptz  NOT NULL,
    updated_at          timestamptz  NOT NULL,
    deleted_at          timestamptz  NULL,
    field_maintenance_id uuid        NULL,
    image_url           varchar(255) NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_field_maintenance_images_field_maintenance_id ON field_maintenance_images (field_maintenance_id);
CREATE INDEX IF NOT EXISTS idx_field_maintenance_images_deleted_at ON field_maintenance_images (deleted_at);

-- Tabel audit_logs
CREATE TABLE IF NOT EXISTS audit_logs (
    id           uuid         PRIMARY KEY,
    created_at   timestamptz  NOT NULL,
    updated_at   timestamptz  NOT NULL,
    deleted_at   timestamptz  NULL,
    user_id      uuid         NULL,
    username     varchar(100) NOT NULL,
    user_role    varchar(50)  NOT NULL,
    action       varchar(100) NOT NULL,
    module       varchar(50)  NOT NULL,
    description  text         NOT NULL,
    ip_address   varchar(45)  NOT NULL,
    user_agent   varchar(255) NULL
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs (user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_username ON audit_logs (username);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs (action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_module ON audit_logs (module);
CREATE INDEX IF NOT EXISTS idx_audit_logs_deleted_at ON audit_logs (deleted_at);

-- Tabel notifications
CREATE TABLE IF NOT EXISTS notifications (
    id              uuid         PRIMARY KEY,
    created_at      timestamptz  NOT NULL,
    updated_at      timestamptz  NOT NULL,
    deleted_at      timestamptz  NULL,
    user_id         uuid         NULL,
    type            varchar(50)  NOT NULL,
    title           varchar(150) NOT NULL,
    message         text         NOT NULL,
    reference_id    uuid         NULL,
    reference_type  varchar(30)  NULL,
    is_read         boolean      NOT NULL DEFAULT false
);

CREATE INDEX IF NOT EXISTS idx_notif_user_read ON notifications (user_id, is_read);
CREATE INDEX IF NOT EXISTS idx_notifications_deleted_at ON notifications (deleted_at);

-- Tabel daily_digests
CREATE TABLE IF NOT EXISTS daily_digests (
    id               uuid        PRIMARY KEY,
    created_at       timestamptz NOT NULL,
    updated_at       timestamptz NOT NULL,
    deleted_at       timestamptz NULL,
    digest_date      date        NOT NULL,
    new_tasks        integer     NOT NULL DEFAULT 0,
    completed_tasks  integer     NOT NULL DEFAULT 0,
    failed_tasks     integer     NOT NULL DEFAULT 0,
    overdue_tasks    integer     NOT NULL DEFAULT 0,
    field_checks     integer     NOT NULL DEFAULT 0,
    attention_checks integer     NOT NULL DEFAULT 0,
    status_changes   integer     NOT NULL DEFAULT 0,
    summary          text        NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_daily_digests_digest_date ON daily_digests (digest_date);
CREATE INDEX IF NOT EXISTS idx_daily_digests_deleted_at ON daily_digests (deleted_at);

-- ============================================================================
-- Foreign Keys (ditambahkan setelah semua tabel ada agar dependensi sirkular
-- maintenance_logs ↔ field_maintenances dapat ditangani dengan aman).
-- Menggunakan blok DO $$ dengan cek pg_constraint karena PostgreSQL tidak
-- mendukung "ADD CONSTRAINT IF NOT EXISTS" — sekaligus membuat migration ini
-- aman dijalankan pada database eksisting hasil AutoMigrate GORM (fase
-- transisi Task 7.1) yang constraint-nya sudah dibuat dengan nama serupa.
-- ============================================================================

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_maintenance_logs_cctv') THEN
        ALTER TABLE maintenance_logs ADD CONSTRAINT fk_maintenance_logs_cctv FOREIGN KEY (cctv_id) REFERENCES cctvs (id);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_maintenance_logs_technician') THEN
        ALTER TABLE maintenance_logs ADD CONSTRAINT fk_maintenance_logs_technician FOREIGN KEY (technician_id) REFERENCES users (id);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_maintenance_logs_schedule') THEN
        ALTER TABLE maintenance_logs ADD CONSTRAINT fk_maintenance_logs_schedule FOREIGN KEY (maintenance_schedule_id) REFERENCES maintenance_schedules (id);
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_field_maintenances_cctv') THEN
        ALTER TABLE field_maintenances ADD CONSTRAINT fk_field_maintenances_cctv FOREIGN KEY (cctv_id) REFERENCES cctvs (id);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_field_maintenances_technician') THEN
        ALTER TABLE field_maintenances ADD CONSTRAINT fk_field_maintenances_technician FOREIGN KEY (technician_id) REFERENCES users (id);
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_field_maintenance_checks_field_maintenance') THEN
        ALTER TABLE field_maintenance_checks ADD CONSTRAINT fk_field_maintenance_checks_field_maintenance FOREIGN KEY (field_maintenance_id) REFERENCES field_maintenances (id);
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_field_maintenance_images_field_maintenance') THEN
        ALTER TABLE field_maintenance_images ADD CONSTRAINT fk_field_maintenance_images_field_maintenance FOREIGN KEY (field_maintenance_id) REFERENCES field_maintenances (id);
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_audit_logs_user') THEN
        ALTER TABLE audit_logs ADD CONSTRAINT fk_audit_logs_user FOREIGN KEY (user_id) REFERENCES users (id);
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_notifications_user') THEN
        ALTER TABLE notifications ADD CONSTRAINT fk_notifications_user FOREIGN KEY (user_id) REFERENCES users (id);
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_maintenance_schedules_cctv') THEN
        ALTER TABLE maintenance_schedules ADD CONSTRAINT fk_maintenance_schedules_cctv FOREIGN KEY (cctv_id) REFERENCES cctvs (id);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_maintenance_schedules_technician') THEN
        ALTER TABLE maintenance_schedules ADD CONSTRAINT fk_maintenance_schedules_technician FOREIGN KEY (technician_id) REFERENCES users (id);
    END IF;
END $$;

-- Foreign key bersifat sirkular (dibuat manual karena GORM tidak membuatnya
-- pada AutoMigrate): field_maintenances.maintenance_log_id →
-- maintenance_logs.id dan maintenance_logs.maintenance_result_id →
-- field_maintenances.id.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_field_maintenances_maintenance_log') THEN
        ALTER TABLE field_maintenances ADD CONSTRAINT fk_field_maintenances_maintenance_log FOREIGN KEY (maintenance_log_id) REFERENCES maintenance_logs (id);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_maintenance_logs_maintenance_result') THEN
        ALTER TABLE maintenance_logs ADD CONSTRAINT fk_maintenance_logs_maintenance_result FOREIGN KEY (maintenance_result_id) REFERENCES field_maintenances (id);
    END IF;
END $$;