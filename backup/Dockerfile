# ==========================================
# SIGAP — Container Backup Otomatis (Task 7.3)
# ==========================================
# Menjalankan cron job yang memanggil scripts/backup.sh secara berkala.
# Memakai postgres:15-alpine karena sudah menyertakan klien PostgreSQL
# (pg_dump, psql, pg_restore, gunzip) dan basis Alpine yang ringan.
#
# Struktur file yang disalin:
#   backup/entrypoint.sh  -> menyiapkan crontab & menjalankan crond di foreground
#   scripts/backup.sh     -> skrip backup itu sendiri
#   scripts/restore.sh    -> skrip restore (dipakai manual / DR)
# ==========================================
FROM postgres:15-alpine

# dcron (cron untuk Alpine) — pengganti busybox crond yang lebih stabil di foreground.
RUN apk add --no-cache dcron bash

# Salin skrip backup/restore & entrypoint.
COPY scripts/backup.sh /usr/local/bin/backup.sh
COPY scripts/restore.sh /usr/local/bin/restore.sh
COPY backup/entrypoint.sh /entrypoint.sh

RUN chmod +x /usr/local/bin/backup.sh /usr/local/bin/restore.sh /entrypoint.sh

# Jalankan entrypoint sebagai init (menyiapkan cron lalu crond di foreground).
ENTRYPOINT ["/entrypoint.sh"]
