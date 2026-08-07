-- Migration 000002: Tabel push_subscriptions (Task 8.3 — Notifikasi Push Browser).
-- Menyimpan langganan Web Push per user (endpoint + kunci enkripsi VAPID).

CREATE TABLE IF NOT EXISTS push_subscriptions (
    id            uuid         PRIMARY KEY,
    created_at    timestamptz  NOT NULL,
    updated_at    timestamptz  NOT NULL,
    deleted_at    timestamptz  NULL,
    user_id       uuid         NOT NULL,
    endpoint      text         NOT NULL,
    key_p256dh    text         NOT NULL,
    key_auth      text         NOT NULL
);

-- Index untuk lookup cepat subscription milik user.
CREATE INDEX IF NOT EXISTS idx_push_subscriptions_user_id ON push_subscriptions (user_id);
CREATE INDEX IF NOT EXISTS idx_push_subscriptions_deleted_at ON push_subscriptions (deleted_at);

-- Foreign key ke users. Idempoten: gunakan blok DO $$ pengecekan pg_constraint
-- karena PostgreSQL tidak mendukung ADD CONSTRAINT IF NOT EXISTS.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_push_subscriptions_user'
    ) THEN
        ALTER TABLE push_subscriptions
            ADD CONSTRAINT fk_push_subscriptions_user
            FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE;
    END IF;
END $$;
