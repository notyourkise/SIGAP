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
# CATATAN: tidak memakai `exec` — saat crond menjadi PID 1 di dalam container,
# pemanggilan `setpgid(0,0)`-nya gagal dengan "Operation not permitted" (EPERM)
# karena PID 1 sudah menjadi session/process group leader. Dengan menjalankan
# crond sebagai child lalu `wait`, crond bukan PID 1 sehingga setpgid berhasil
# dan container tetap hidup selama crond berjalan.
echo "Memulai crond (foreground)..."
crond -f -l 2 &
CRON_PID=$!
# Tunggu crond; bila berhenti, container ikut berhenti (restart policy menangani).
wait "$CRON_PID"
