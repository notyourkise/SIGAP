-- Migration 000012: migrasi koordinat CCTV dari varchar ke double precision (Task 13.1).
-- Nilai kosong ('' / spasi) dikonversi ke NULL; nilai tidak valid akan gagal dan
-- migration perlu dievaluasi data-nya terlebih dahulu.
ALTER TABLE cctvs
  ALTER COLUMN latitude TYPE double precision
    USING NULLIF(TRIM(latitude), '')::double precision,
  ALTER COLUMN longitude TYPE double precision
    USING NULLIF(TRIM(longitude), '')::double precision;
