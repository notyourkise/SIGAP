-- Migration 000030: pengaman CHECK stok suku cadang non-negatif & jumlah pemakaian positif (H05 #171).
UPDATE spare_parts SET stok = 0 WHERE stok < 0;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_spare_parts_stok_nonnegative') THEN
        ALTER TABLE spare_parts
            ADD CONSTRAINT chk_spare_parts_stok_nonnegative CHECK (stok >= 0);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_spare_part_usages_jumlah_positive') THEN
        ALTER TABLE spare_part_usages
            ADD CONSTRAINT chk_spare_part_usages_jumlah_positive CHECK (jumlah > 0);
    END IF;
END $$;
