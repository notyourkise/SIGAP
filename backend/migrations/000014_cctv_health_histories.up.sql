-- Migration 000014: tabel histori skor kesehatan CCTV (Task 13.8).
CREATE TABLE IF NOT EXISTS cctv_health_histories (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  cctv_id uuid NOT NULL REFERENCES cctvs(id) ON DELETE CASCADE,
  health_score integer NOT NULL CHECK (health_score BETWEEN 0 AND 100),
  calculated_at timestamptz NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_health_histories_cctv_date
  ON cctv_health_histories (cctv_id, calculated_at DESC);
