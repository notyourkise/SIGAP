-- Migration 000012 (rollback): kembalikan koordinat CCTV ke varchar(50).
ALTER TABLE cctvs
  ALTER COLUMN latitude TYPE varchar(50) USING latitude::varchar,
  ALTER COLUMN longitude TYPE varchar(50) USING longitude::varchar;
