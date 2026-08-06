-- Migration 000001 (rollback): Menghapus seluruh tabel skema awal SIGAP.
-- Urutan penghapusan terbalik dari pembuatan (anak sebelum orang tua).
-- maintenance_logs dihapus dengan CASCADE karena FK sirkular
-- (field_maintenances.maintenance_log_id → maintenance_logs.id) masih
-- menunjuk ke tabel ini; CASCADE hanya menjatuhkan constraint yang
-- bergantung, bukan tabel lain.
DROP TABLE IF EXISTS daily_digests;
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS field_maintenance_images;
DROP TABLE IF EXISTS field_maintenance_checks;
DROP TABLE IF EXISTS maintenance_logs CASCADE;
DROP TABLE IF EXISTS field_maintenances;
DROP TABLE IF EXISTS maintenance_schedules;
DROP TABLE IF EXISTS cctvs;
DROP TABLE IF EXISTS users;