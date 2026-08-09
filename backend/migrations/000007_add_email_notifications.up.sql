-- Migration 000007: kolom email_notifications pada users (Task 11.3, Sprint 27).
-- Menambahkan preferensi notifikasi email per pengguna (default aktif).
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'users' AND column_name = 'email_notifications'
    ) THEN
        ALTER TABLE users
            ADD COLUMN email_notifications boolean NOT NULL DEFAULT true;
    END IF;
END $$;
