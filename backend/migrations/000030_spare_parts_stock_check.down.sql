-- Rollback Migration 000030: hapus constraint CHECK stok suku cadang & jumlah pemakaian.
ALTER TABLE spare_part_usages DROP CONSTRAINT IF EXISTS chk_spare_part_usages_jumlah_positive;
ALTER TABLE spare_parts DROP CONSTRAINT IF EXISTS chk_spare_parts_stok_nonnegative;
