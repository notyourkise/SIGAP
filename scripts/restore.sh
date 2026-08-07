#!/usr/bin/env bash
# ==========================================
# SIGAP — Restore Backup (Task 7.3)
# ==========================================
# Memulihkan database dan/atau uploads dari hasil backup scripts/backup.sh.
#
# Cara pakai (host / di dalam container):
#   ./scripts/restore.sh <path_to_sql_gz> [path_to_uploads_tar_gz]
#
# Contoh:
#   # Restore database saja
#   ./scripts/restore.sh backups/db/sigap_20260806_060000.sql.gz
#
#   # Restore database + uploads
#   ./scripts/restore.sh backups/db/sigap_20260806_060000.sql.gz \
#                       backups/uploads/uploads_20260806_060000.tar.gz
#
# PERINGATAN:
#   - Restore database akan MENIMPA seluruh isi database saat ini (drop & recreate).
#   - Pastikan service API dihentikan selama proses restore untuk menghindari
#     inkonsistensi / tulis bersamaan.
#   - Pastikan file .sql.gz dibuat dengan pg_dump versi yang kompatibel.
#
# Konfigurasi via environment variable (sama dengan backup.sh):
#   DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME
#   UPLOADS_DIR
# ==========================================
set -euo pipefail

# --- Konfigurasi (dapat di-override via env) ---
DB_HOST="${DB_HOST:-db}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-sigap_user}"
DB_PASSWORD="${DB_PASSWORD:-}"
DB_NAME="${DB_NAME:-sigap_db}"

# Lokasi folder uploads tujuan.
UPLOADS_DIR="${UPLOADS_DIR:-/app/uploads}"

usage() {
  echo "Penggunaan: $0 <sql_dump_path> [uploads_tar_gz_path]" >&2
  echo "  <sql_dump_path>         file backup database (*.sql.gz) dari backup.sh" >&2
  echo "  [uploads_tar_gz_path]   (opsional) file backup uploads (*.tar.gz)" >&2
  exit 1
}

# --- Validasi argumen ---
if [ "$#" -lt 1 ]; then
  usage
fi

SQL_DUMP="${1:-}"
UPLOADS_ARCHIVE="${2:-}"

if [ ! -f "${SQL_DUMP}" ]; then
  echo "ERROR: File SQL dump tidak ditemukan: ${SQL_DUMP}" >&2
  exit 1
fi

if [ -n "${UPLOADS_ARCHIVE}" ] && [ ! -f "${UPLOADS_ARCHIVE}" ]; then
  echo "ERROR: File uploads archive tidak ditemukan: ${UPLOADS_ARCHIVE}" >&2
  exit 1
fi

log() { echo "[restore $(date -u +%H:%M:%S)] $*"; }

# --- 1. Restore Database ---
log "=== RESTORE DATABASE ==="
log "Mengembalikan '${DB_NAME}' dari '${SQL_DUMP}' @ ${DB_HOST}:${DB_PORT} ..."

# Gunakan psql untuk drop/recreate agar restore bersih (tanpa data lama).
# Bila psql tersedia (postgres-client), lakukan drop & create; jika hanya
# pg_restore yang dipakai untuk format plain dump (gzip), cukup feed ke psql.
if command -v psql >/dev/null 2>&1; then
  # Hentikan koneksi aktif lain, drop & recreate database bersih.
  PGPASSWORD="${DB_PASSWORD}" psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" \
    -d "postgres" -v ON_ERROR_STOP=1 \
    -c "DROP DATABASE IF EXISTS \"${DB_NAME}\";" \
    -c "CREATE DATABASE \"${DB_NAME}\";" \
    >/dev/null
  log "Database '${DB_NAME}' di-drop & dibuat ulang (kosong)."
else
  log "WARNING: 'psql' tidak ditemukan — melewati drop/recreate. Asumsikan database kosong."
fi

# Restore isi dump (gzip) langsung ke database.
if command -v gunzip >/dev/null 2>&1; then
  gunzip -c "${SQL_DUMP}" | PGPASSWORD="${DB_PASSWORD}" psql -h "${DB_HOST}" -p "${DB_PORT}" \
    -U "${DB_USER}" -d "${DB_NAME}" -v ON_ERROR_STOP=1 >/dev/null
else
  # Fallback: pg_restore dapat membaca gzip untuk format custom/archive; untuk
  # plain SQL gzip perlu di-decompress terlebih dahulu.
  echo "ERROR: 'gunzip' tidak ditemukan. Diperlukan untuk membaca dump gzip." >&2
  exit 1
fi
log "Database restored dari '${SQL_DUMP}'."

# --- 2. Restore Uploads ---
if [ -n "${UPLOADS_ARCHIVE}" ]; then
  log "=== RESTORE UPLOADS ==="
  log "Mengembalikan '${UPLOADS_DIR}' dari '${UPLOADS_ARCHIVE}' ..."

  # Bersihkan folder uploads lama (buat kembali) agar tidak ada file yatim.
  rm -rf "${UPLOADS_DIR}"
  mkdir -p "${UPLOADS_DIR}"

  # Ekstrak archive. Skrip backup membuat struktur relatif '<nama uploads>'.
  tar -xzf "${UPLOADS_ARCHIVE}" -C "$(dirname "${UPLOADS_DIR}")"
  log "Uploads restored dari '${UPLOADS_ARCHIVE}' ke '${UPLOADS_DIR}'."
else
  log "Argumen uploads tidak diberikan — melewati restore uploads."
fi

log "=== RESTORE SELESAI ==="
