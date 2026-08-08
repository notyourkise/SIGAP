-- Rollback Migration 000006: hapus tabel suku cadang (urutan kebalikan dari up).
DROP TABLE IF EXISTS spare_part_usages;
DROP TABLE IF EXISTS spare_parts;
