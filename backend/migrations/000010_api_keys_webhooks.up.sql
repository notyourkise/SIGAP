-- Migration 000010: API Key & Webhook (Task 12.1 + 12.3, Sprint 29).
-- Menambah tabel api_keys (kredensial integrasi eksternal) dan tabel
-- webhook_endpoints + webhook_deliveries (pengiriman event ke sistem lain).

-- API keys untuk integrasi eksternal (konsumen memanggil API via X-API-Key).
CREATE TABLE IF NOT EXISTS api_keys (
    id            uuid          PRIMARY KEY,
    created_at    timestamptz   NOT NULL,
    updated_at    timestamptz   NOT NULL,
    deleted_at    timestamptz   NULL,
    name          varchar(100)  NOT NULL,
    key_hash      varchar(255)  NOT NULL,
    key_prefix    varchar(16)   NOT NULL,
    scopes        jsonb         NOT NULL DEFAULT '[]',
    expires_at    timestamptz   NULL,
    created_by    uuid          NOT NULL,
    last_used_at  timestamptz   NULL,
    revoked_at    timestamptz   NULL
);
CREATE INDEX IF NOT EXISTS idx_api_keys_deleted_at ON api_keys (deleted_at);
CREATE INDEX IF NOT EXISTS idx_api_keys_key_prefix ON api_keys (key_prefix);
CREATE INDEX IF NOT EXISTS idx_api_keys_created_by ON api_keys (created_by);

-- Webhook endpoints (target URL penerima event).
CREATE TABLE IF NOT EXISTS webhook_endpoints (
    id            uuid          PRIMARY KEY,
    created_at    timestamptz   NOT NULL,
    updated_at    timestamptz   NOT NULL,
    deleted_at    timestamptz   NULL,
    name          varchar(100)  NOT NULL,
    url           text          NOT NULL,
    events        jsonb         NOT NULL DEFAULT '[]',
    secret        text          NOT NULL,
    is_active     boolean       NOT NULL DEFAULT true,
    created_by    uuid          NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_webhook_endpoints_deleted_at ON webhook_endpoints (deleted_at);
CREATE INDEX IF NOT EXISTS idx_webhook_endpoints_created_by ON webhook_endpoints (created_by);

-- Log pengiriman webhook (satu baris per percobaan/delivery).
CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id                    uuid          PRIMARY KEY,
    created_at            timestamptz   NOT NULL,
    updated_at            timestamptz   NOT NULL,
    deleted_at            timestamptz   NULL,
    webhook_endpoint_id   uuid          NOT NULL,
    event                 varchar(100)  NOT NULL,
    payload               jsonb         NOT NULL,
    response_code         integer       NULL,
    response_body         text          NULL,
    latency               bigint        NULL,
    status                varchar(20)   NOT NULL DEFAULT 'pending',
    attempt               integer       NOT NULL DEFAULT 1
);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_endpoint ON webhook_deliveries (webhook_endpoint_id);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_event ON webhook_deliveries (event);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_status ON webhook_deliveries (status);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_deleted_at ON webhook_deliveries (deleted_at);

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_api_keys_created_by') THEN
        ALTER TABLE api_keys ADD CONSTRAINT fk_api_keys_created_by
            FOREIGN KEY (created_by) REFERENCES users (id) ON DELETE CASCADE;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_webhook_endpoints_created_by') THEN
        ALTER TABLE webhook_endpoints ADD CONSTRAINT fk_webhook_endpoints_created_by
            FOREIGN KEY (created_by) REFERENCES users (id) ON DELETE CASCADE;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_webhook_deliveries_endpoint') THEN
        ALTER TABLE webhook_deliveries ADD CONSTRAINT fk_webhook_deliveries_endpoint
            FOREIGN KEY (webhook_endpoint_id) REFERENCES webhook_endpoints (id) ON DELETE CASCADE;
    END IF;
END $$;
