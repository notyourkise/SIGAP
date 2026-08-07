-- Migration 000003: tambah kolom geotag pada field_maintenances (Task 8.4).
ALTER TABLE field_maintenances ADD COLUMN IF NOT EXISTS check_latitude varchar(30) NULL;
ALTER TABLE field_maintenances ADD COLUMN IF NOT EXISTS check_longitude varchar(30) NULL;
