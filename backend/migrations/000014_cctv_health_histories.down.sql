-- Migration 000014 (rollback): hapus tabel histori skor kesehatan CCTV.
DROP TABLE IF EXISTS cctv_health_histories;
