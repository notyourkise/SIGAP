-- Migration 000015: kolom overdue_level untuk eskalasi bertingkat (Task 13.11).
ALTER TABLE maintenance_logs
  ADD COLUMN IF NOT EXISTS overdue_level INTEGER NOT NULL DEFAULT 0;
