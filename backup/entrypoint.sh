#!/bin/sh
# ==========================================
# SIGAP — Entrypoint Container Backup (Task 7.3)
# ==========================================
# 1. Menulis crontab untuk menjalankan /usr/local/bin/backup.sh sesuai jadwal.
#    Default: setiap hari pukul 06:00 WITA (CRON_SCHEDULE bisa di-override).
# 2. Menjalankan crond (dcron) di foreground agar container tetap hidup.
#
# Variabel environment yang dipakai:
#   CRON_SCHEDULE  -> jadwal cron (default "0 6 * * *" = tiap hari 06:00).
#                     Catatan: crond memakai zona waktu UTC; untuk 06:00 WITA
#                     gunakan "0 22 * * *" (UTC) atau set TZ container.
#   TZ             -> zona waktu container (default Asia/Makassar). Dengan TZ
#                     Asia/Makassar, CRON_SCHEDULE "0 6 * * *" berarti 06:00 WITA.
# ==========================================
set -e

CRON_SCHEDULE="${CRON_SCHEDULE:-0 6 * * *}"

# Pastikan TZ di-set agar jadwal cron mengikuti zona WITA bila diinginkan.
: "${TZ:=Asia/Makassar}"
export TZ

echo "Menulis crontab dengan jadwal: '${CRON_SCHEDULE}' (TZ=${TZ})"
# Setiap baris cron ditambah redirect log ke stdout agar terlihat di `docker logs`.
echo "${CRON_SCHEDULE} /usr/local/bin/backup.sh >> /proc/1/fd/1 2>&1" > /etc/crontabs/root

# Jalankan crond (dcron) di foreground.
echo "Memulai crond (foreground)..."
exec crond -f -l 2
