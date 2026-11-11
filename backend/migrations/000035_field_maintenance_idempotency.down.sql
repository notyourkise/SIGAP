DROP INDEX IF EXISTS uq_field_maintenances_tech_client_req;
ALTER TABLE field_maintenances
    DROP COLUMN IF EXISTS client_request_id;
