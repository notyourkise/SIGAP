#!/usr/bin/env bash
# ==========================================
# SIGAP — Backup Otomatis (Task 7.3)
# ==========================================
# Membuat backup database PostgreSQL (pg_dump) dan folder uploads, menyimpannya
# dengan rotasi otomatis (hapus backup lebih dari RETENTION_DAYS hari).
#
# Output:
#   backups/db/sigap_YYYYMMDD_HHMMSS.sql.gz            (dump DB + gzip)
#   backups/uploads/uploads_YYYYMMDD_HHMMSS.tar.gz     (tar + gzip uploads)
#
# Cara pakai (host):
#   chmod +x scripts/backup.sh
#   ./scripts/backup.sh
#
# Di dalam container backup (alpine + postgres-client), variabel sudah di-set
# dari docker-compose sehingga skrip bisa langsung dijalankan oleh cron.
#
# Konfigurasi via environment variable (dengan nilai default aman):
#   DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME
#   UPLOADS_DIR, BACKUP_DIR, RETENTION_DAYS
#
# Semua variabel bisa di-override; default menyesuaikan deployment Docker SIGAP.
# ==========================================
set -euo pipefail

# --- Konfigurasi (dapat di-override via env) ---
DB_HOST="${DB_HOST:-db}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-sigap_user}"
DB_PASSWORD="${DB_PASSWORD:-}"
DB_NAME="${DB_NAME:-sigap_db}"

# Lokasi folder uploads & tempat backup disimpan.
UPLOADS_DIR="${UPLOADS_DIR:-/app/uploads}"
BACKUP_DIR="${BACKUP_DIR:-/backups}"

# Retensi backup dalam hari (default 7).
RETENTION_DAYS="${RETENTION_DAYS:-7}"

# Timestamp WITA (UTC+8) untuk penamaan file backup.
TIMESTAMP="$(TZ=Asia/Makassar date +%Y%m%d_%H%M%S)"
DB_DIR="${BACKUP_DIR}/db"
UPLOADS_BACKUP_DIR="${BACKUP_DIR}/uploads"

log() { echo "[backup $(date -u +%H:%M:%S)] $*"; }

# --- Siapkan direktori tujuan ---
mkdir -p "${DB_DIR}" "${UPLOADS_BACKUP_DIR}"

DB_FILE="${DB_DIR}/sigap_${TIMESTAMP}.sql.gz"
UPLOADS_FILE="${UPLOADS_BACKUP_DIR}/uploads_${TIMESTAMP}.tar.gz"

# --- 1. Backup Database ---
log "Memulai backup database '${DB_NAME}' @ ${DB_HOST}:${DB_PORT} ..."
if [ -z "${DB_PASSWORD}" ]; then
  PGPASSWORD="" pg_dump -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" \
    -d "${DB_NAME}" --no-owner --no-privileges | gzip > "${DB_FILE}"
else
  PGPASSWORD="${DB_PASSWORD}" pg_dump -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" \
    -d "${DB_NAME}" --no-owner --no-privileges | gzip > "${DB_FILE}"
fi
log "Database backup selesai -> ${DB_FILE} ($(du -h "${DB_FILE}" | cut -f1))"

# --- 2. Backup Uploads ---
log "Memulai backup folder uploads '${UPLOADS_DIR}' ..."
if [ -d "${UPLOADS_DIR}" ] && [ -n "$(ls -A "${UPLOADS_DIR}" 2>/dev/null)" ]; then
  # -C: masuk ke folder sebelum tar agar struktur relatif (isi uploads/ saja).
  tar -czf "${UPLOADS_FILE}" -C "$(dirname "${UPLOADS_DIR}")" "$(basename "${UPLOADS_DIR}")"
  log "Uploads backup selesai -> ${UPLOADS_FILE} ($(du -h "${UPLOADS_FILE}" | cut -f1))"
else
  # Folder uploads kosong / belum ada: simpan archive kosong agar restore konsisten.
  tar -czf "${UPLOADS_FILE}" --files-from /dev/null
  log "Folder uploads kosong/ tidak ada; archive kosong dibuat -> ${UPLOADS_FILE}"
fi

# --- 3. Rotasi: hapus backup lebih dari RETENTION_DAYS ---
log "Rotasi: menghapus backup lebih dari ${RETENTION_DAYS} hari ..."
find "${DB_DIR}" -name 'sigap_*.sql.gz' -mtime "+${RETENTION_DAYS}" -delete 2>/dev/null || true
find "${UPLOADS_BACKUP_DIR}" -name 'uploads_*.tar.gz' -mtime "+${RETENTION_DAYS}" -delete 2>/dev/null || true
log "Rotasi selesai."

log "Backup selesai. Ringkasan direktori '${BACKUP_DIR}':"
ls -lhR "${BACKUP_DIR}"
