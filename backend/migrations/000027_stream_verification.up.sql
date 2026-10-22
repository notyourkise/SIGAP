-- Verifikasi aktif status stream CCTV (Sprint 41).
-- Dipisahkan dari stream_state (probe pasif) agar UI dapat membedakan
-- "belum diverifikasi" dari "offline terkonfirmasi".
ALTER TABLE cctvs
    ADD COLUMN IF NOT EXISTS stream_verified_state varchar(20) NOT NULL DEFAULT 'unknown',
    ADD COLUMN IF NOT EXISTS stream_verified_at timestamptz,
    ADD COLUMN IF NOT EXISTS stream_verified_method varchar(32) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS stream_verify_failures integer NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS stream_verify_error text NOT NULL DEFAULT '';
