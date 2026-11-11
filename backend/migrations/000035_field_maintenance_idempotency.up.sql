-- M14: Idempotency key untuk pengiriman pengecekan lapangan offline agar retry
-- jaringan tidak menggandakan record ketika klien mengulang request yang sama.
ALTER TABLE field_maintenances
    ADD COLUMN IF NOT EXISTS client_request_id VARCHAR(100);

CREATE UNIQUE INDEX IF NOT EXISTS uq_field_maintenances_tech_client_req
ON field_maintenances (technician_id, client_request_id)
WHERE client_request_id IS NOT NULL AND deleted_at IS NULL;
