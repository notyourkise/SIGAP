#!/usr/bin/env bash
# ==========================================
# SIGAP — Restore Backup (Task 7.3)
# ==========================================
# Memulihkan database dan/atau uploads dari hasil backup scripts/backup.sh.
# Mendukung backup terenkripsi (.gpg) maupun raw (.gz / .tar.gz).
#
# Cara pakai (host / di dalam container):
#   ./scripts/restore.sh <path_to_sql_gz> [path_to_uploads_tar_gz]
#
# Contoh:
#   # Restore database saja (raw gzip)
#   ./scripts/restore.sh backups/db/sigap_20260806_060000.sql.gz
#
#   # Restore database terenkripsi
#   ./scripts/restore.sh backups/db/sigap_20260806_060000.sql.gz.gpg
#
#   # Restore database + uploads (keduanya terenkripsi)
#   ./scripts/restore.sh backups/db/sigap_20260806_060000.sql.gz.gpg \
#                       backups/uploads/uploads_20260806_060000.tar.gz.gpg
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
#   ENCRYPTION_KEY     (wajib jika file backup terenkripsi .gpg)
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

# Kunci enkripsi (wajib untuk file .gpg).
ENCRYPTION_KEY="${ENCRYPTION_KEY:-}"

usage() {
  echo "Penggunaan: $0 <sql_dump_path> [uploads_tar_gz_path]" >&2
  echo "  <sql_dump_path>         file backup database (*.sql.gz atau *.sql.gz.gpg)" >&2
  echo "  [uploads_tar_gz_path]   (opsional) file backup uploads (*.tar.gz atau *.tar.gz.gpg)" >&2
  echo "" >&2
  echo "  Untuk file .gpg, set env ENCRYPTION_KEY" >&2
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

# --- Helper: Deteksi & decrypt file GPG ---
# Jika file berakhiran .gpg, decrypt ke temp file dan return path temp.
# Jika tidak, return path asli.
decrypt_if_needed() {
  local file_path="$1"
  local temp_file=""

  if [[ "${file_path}" == *.gpg ]]; then
    if [ -z "${ENCRYPTION_KEY}" ]; then
      echo "ERROR: File '${file_path}' terenkripsi tapi ENCRYPTION_KEY tidak diset." >&2
      exit 1
    fi
    if ! command -v gpg >/dev/null 2>&1; then
      echo "ERROR: 'gpg' tidak ditemukan. Diperlukan untuk mendekripsi file .gpg." >&2
      exit 1
    fi
    # Buat temp file (hapus ekstensi .gpg).
    temp_file="${file_path%.gpg}"
    log "Mendekripsi '${file_path}' -> '${temp_file}' ..."
    echo "${ENCRYPTION_KEY}" | gpg --batch --yes --passphrase-fd 0 --decrypt \
      --output "${temp_file}" "${file_path}"
    if [ $? -ne 0 ]; then
      echo "ERROR: Dekripsi gagal untuk '${file_path}'. Periksa ENCRYPTION_KEY." >&2
      rm -f "${temp_file}"
      exit 1
    fi
    echo "${temp_file}"
  else
    echo "${file_path}"
  fi
}

# --- 1. Restore Database ---
log "=== RESTORE DATABASE ==="
log "Mengembalikan '${DB_NAME}' dari '${SQL_DUMP}' @ ${DB_HOST}:${DB_PORT} ..."

# Deteksi file GPG & decrypt jika diperlukan.
TEMP_SQL=$(decrypt_if_needed "${SQL_DUMP}")

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
# File TEMP_SQL bisa berupa .gz atau hasil decrypt (.sql.gz).
if [[ "${TEMP_SQL}" == *.gz ]]; then
  if command -v gunzip >/dev/null 2>&1; then
    gunzip -c "${TEMP_SQL}" | PGPASSWORD="${DB_PASSWORD}" psql -h "${DB_HOST}" -p "${DB_PORT}" \
      -U "${DB_USER}" -d "${DB_NAME}" -v ON_ERROR_STOP=1 >/dev/null
  else
    echo "ERROR: 'gunzip' tidak ditemukan. Diperlukan untuk membaca dump gzip." >&2
    exit 1
  fi
else
  # Langsung cat untuk file plain SQL.
  cat "${TEMP_SQL}" | PGPASSWORD="${DB_PASSWORD}" psql -h "${DB_HOST}" -p "${DB_PORT}" \
    -U "${DB_USER}" -d "${DB_NAME}" -v ON_ERROR_STOP=1 >/dev/null
fi

log "Database restored dari '${SQL_DUMP}'."

# --- 2. Restore Uploads ---
if [ -n "${UPLOADS_ARCHIVE}" ]; then
  log "=== RESTORE UPLOADS ==="
  log "Mengembalikan '${UPLOADS_DIR}' dari '${UPLOADS_ARCHIVE}' ..."

  # Decrypt jika perlu.
  TEMP_UPLOADS=$(decrypt_if_needed "${UPLOADS_ARCHIVE}")

  # Bersihkan folder uploads lama (buat kembali) agar tidak ada file yatim.
  rm -rf "${UPLOADS_DIR}"
  mkdir -p "${UPLOADS_DIR}"

  # Ekstrak archive. Skrip backup membuat struktur relatif '<nama uploads>'.
  tar -xzf "${TEMP_UPLOADS}" -C "$(dirname "${UPLOADS_DIR}")"
  log "Uploads restored dari '${UPLOADS_ARCHIVE}' ke '${UPLOADS_DIR}'."
else
  log "Argumen uploads tidak diberikan — melewati restore uploads."
fi

log "=== RESTORE SELESAI ==="
