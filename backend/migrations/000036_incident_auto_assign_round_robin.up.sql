ALTER TABLE sites
    ADD COLUMN IF NOT EXISTS last_auto_assigned_technician_id UUID NULL REFERENCES users(id) ON DELETE SET NULL;
