ALTER TABLE cctvs
    DROP COLUMN IF EXISTS stream_verify_error,
    DROP COLUMN IF EXISTS stream_verify_failures,
    DROP COLUMN IF EXISTS stream_verified_method,
    DROP COLUMN IF EXISTS stream_verified_at,
    DROP COLUMN IF EXISTS stream_verified_state;
