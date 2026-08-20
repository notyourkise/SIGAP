# BACKUP — Backup Otomatis & Restore (Task 7.3)

> Prosedur backup database & uploads SIGAP secara otomatis beserta pemulihan
> (disaster recovery). Menggunakan skrip `scripts/backup.sh` dan `scripts/restore.sh`.
> Diperbarui: 2026-08-21 (Task 7.3 + Encryption新增, Fase 7 Kesiapan Produksi).

---

## 1. Gambaran Umum

SIGAP menyimpan data penting di dua tempat:

1. **Database PostgreSQL** — seluruh data terstruktur (user, CCTV, maintenance log,
   pengecekan lapangan, audit, notifikasi, jadwal, digest).
2. **Folder `backend/uploads/`** — foto bukti pemeliharaan & pengecekan lapangan.

Backup otomatis memastikan kedua hal tersebut bisa dipulihkan bila terjadi
kegagalan perangkat, penghapusan tak sengaja, atau korupsi data. Tersedia dua
cara menjadwalkan: **container `backup` di docker-compose (opsi utama)** dan
**cron job di host** (opsi alternatif).

> ⚠️ **Keamanan Backup:** Untuk production, **WAKTU MENGGUNAKAN ENKRIPSI** untuk
> melindungi backup dari akses tidak sah saat off-site storage atau transfer.

Hasil backup disimpan di `backups/`:

```
backups/
├── db/
│   ├── sigap_YYYYMMDD_HHMMSS.sql.gz           # raw gzip dump (tidak aman)
│   └── sigap_YYYYMMDD_HHMMSS.sql.gz.gpg       # AES-256 encrypted ← RECOMMENDED
└── uploads/
    ├── uploads_YYYYMMDD_HHMMSS.tar.gz         # raw tar+gzip (tidak aman)
    └── uploads_YYYYMMDD_HHMMSS.tar.gz.gpg     # AES-256 encrypted ← RECOMMENDED
```

> `YYYYMMDD_HHMMSS` memakai zona waktu **WITA (Asia/Makassar, UTC+8)**.

---

## 2. Prasyarat

- PostgreSQL klien (`pg_dump`, `psql`, `gunzip`, `tar`) tersedia:
  - Di dalam container `backup` → otomatis (image `postgres:15-alpine`).
  - Di host → pasang `postgresql-client` (mis. `apt install postgresql-client`).
- Skrip dapat dieksekusi: `chmod +x scripts/backup.sh scripts/restore.sh`.
- Kredensial DB dapat diakses melalui environment variable (lihat bagian 3).

---

## 3. Variabel Environment

Kedua skrip membaca konfigurasi dari environment variable (dengan nilai default
yang menyesuaikan deployment Docker SIGAP). Dapat di-override per pemanggilan.
| Variabel | Default | Deskripsi |
|----------|---------|-----------|
| `DB_HOST` | `db` | Host/name PostgreSQL |
| `DB_PORT` | `5432` | Port PostgreSQL |
| `DB_USER` | `sigap_user` | User database |
| `DB_PASSWORD` | *(kosong)* | Password database |
| `DB_NAME` | `sigap_db` | Nama database |
| `UPLOADS_DIR` | `/app/uploads` | Folder uploads sumber/tujuan |
| `BACKUP_DIR` | `/backups` | Folder induk hasil backup |
| `RETENTION_DAYS` | `7` | Retensi backup dalam hari |
| `CRON_SCHEDULE` | `0 6 * * *` | Jadwal cron (container backup saja) |
| `TZ` | `Asia/Makassar` | Zona waktu container backup |

> **Variabel Enkripsi (NEW):**  
> Untuk enkripsi backup AES-256, tambahkan:
> - `ENCRYPT_BACKUPS=true` — aktifkan enkripsi
> - `ENCRYPTION_KEY=<strong_password>` — kunci AES-256 untuk enkripsi/dekripsi
> 
> Contoh secure key generation: `openssl rand -base64 32`

---

## 4. Skrip Backup (`scripts/backup.sh`)

Skrip melakukan tiga hal:

1. **Backup database**: `pg_dump` → `gzip` → `backups/db/sigap_<TS>.sql.gz`.
2. **Backup uploads**: `tar` + `gzip` folder uploads → `backups/uploads/uploads_<TS>.tar.gz`.
   Bila folder uploads kosong/tidak ada, dibuat archive kosong agar restore konsisten.
3. **Rotasi otomatis**: hapus file backup `db`/`uploads` yang lebih tua dari
   `RETENTION_DAYS` (default 7 hari) menggunakan `find -mtime +N -delete`.

> `pg_dump` memakai opsi `--no-owner --no-privileges` agar dump lebih portabel
> lintas environment. Jalankan skrip saat database sedang stabil (idle).

### 4.1 Menjalankan manual (host)

```bash
chmod +x scripts/backup.sh
DB_HOST=localhost DB_PORT=5432 DB_USER=sigap_user DB_PASSWORD=xxxx DB_NAME=sigap_db \
  UPLOADS_DIR=/path/backend/uploads BACKUP_DIR=/path/backups \
  ./scripts/backup.sh
```

### 4.2 Menjalankan manual (di dalam container)

```bash
docker compose exec backup backup.sh
```

Hasil backup otomatis tersimpan ke named volume `backups_data`.

---

## 5. Skrip Restore (`scripts/restore.sh`)

Memulihkan database dan/atau uploads dari hasil backup.

```bash
# Restore database saja
./scripts/restore.sh backups/db/sigap_20260806_060000.sql.gz

# Restore database + uploads
./scripts/restore.sh backups/db/sigap_20260806_060000.sql.gz \
                    backups/uploads/uploads_20260806_060000.tar.gz
```

### 5.1 Yang dilakukan skrip

1. **Restore database**: drop & recreate database bersih (via `psql`), lalu
   `gunzip` dump dan feed ke `psql`.
2. **Restore uploads** (opsional): hapus folder uploads lama, buat ulang, lalu
   ekstrak archive `tar.gz`.

### 5.2 Contoh di dalam container

```bash
# Masuk ke shell container backup
docker compose exec backup sh

# Jalankan restore dari file di /backups
/usr/local/bin/restore.sh /backups/db/sigap_20260806_060000.sql.gz \
                          /backups/uploads/uploads_20260806_060000.tar.gz
```

> [!WARNING]
> **Restore database menimpa seluruh isi database saat ini.**
> - Hentikan service `api` selama proses restore agar tidak ada tulis bersamaan:
>   `docker compose stop api`.
> - Pastikan file `.sql.gz` valid dan dibuat dengan versi `pg_dump` yang kompatibel.
> - Setelah restore, jalankan kembali: `docker compose start api`.

---

## 5.3 Backup Terenkripsi (NEW)

Untuk meningkatkan keamanan, backup dapat dienkripsi menggunakan **AES-256** sebelum disimpan.
Fitur ini sangat disarankan untuk production environment atau jika backup akan disimpan
di lokasi off-site (AWS S3, Google Cloud Storage, dll).

### 5.3.1 Cara mengaktifkan enkripsi

Set environment variable berikut saat menjalankan backup:

```bash
ENCRYPTION_KEY="your-secure-encryption-key"
ENCRYPT_BACKUPS=true
./scripts/backup.sh
```

Atau di dalam `docker-compose.yml`:

```yaml
environment:
  - ENCRYPTION_KEY=${ENCRYPTION_KEY}
  - ENCRYPT_BACKUPS=true
```

### 5.3.2 Output backup terenkripsi

Jika enkripsi aktif, skrip akan membuat dua versi file:

- `sigap_<TS>.sql.gz` — raw dump (tidak aman untuk off-site)
- `sigap_<TS>.sql.gz.gpg` — AES-256 encrypted ← gunakan ini untuk backup

File `.gpg` hanya bisa dibuka dengan kunci yang sama saat restore.

### 5.3.3 Restore backup terenkripsi

Gunakan `ENCRYPTION_KEY` yang sama dengan saat backup:

```bash
export ENCRYPTION_KEY="your-secure-encryption-key"
./scripts/restore.sh backups/db/sigap_20260806_060000.sql.gz.gpg \
                      backups/uploads/uploads_20260806_060000.tar.gz.gpg
```

Skrip otomatis mendeteksi ekstensi `.gpg` dan mendekripsi sebelum restore.

### 5.3.4 Best practices

- 🔒 **Generate strong key:** Gunakan `openssl rand -base64 32` untuk generate key baru.
- 🔐 **Jangan hardcode key di repo:** Simpan di Docker secrets atau env file terpisah (.gitignore).
- 🔄 **Rotasi key:** Ganti key secara berkala dan backup semua file lama dengan key lama.
- 📦 **Backup key separately:** Jangan simpan key di lokasi yang sama dengan backup files.
- ✅ **Verifikasi restore:** Selalu test restore dengan key baru di staging sebelum production.

---

## 6. Penjadwalan Otomatis

### Opsi 2 (utama): Container `backup` di docker-compose

Service `backup` (di `docker-compose.yml`) berjalan dengan **cron**:

- Memakai image `postgres:15-alpine` (sudah punya klien PostgreSQL) + `dcron`.
- `backup/entrypoint.sh` menulis crontab yang memanggil `/usr/local/bin/backup.sh`
  lalu menjalankan `crond` di foreground.
- Volume `backups_data:/backups` menyimpan hasil backup (persisten).
- Folder uploads di-mount read-only dari host (`./backend/uploads:/app/uploads:ro`).
- Menunggu `db` sehat (`depends_on ... condition: service_healthy`).

Jadwal default: **setiap hari 06:00 WITA** (`CRON_SCHEDULE=0 6 * * *` dengan
`TZ=Asia/Makassar`). Untuk mengubah jadwal, atur `CRON_SCHEDULE` di
`docker-compose.yml` (atau root `.env` bila Anda mengekspos variabelnya).

Contoh jadwal cron:

| Jadwal | Nilai `CRON_SCHEDULE` |
|--------|----------------------|
| Setiap hari 06:00 WITA | `0 6 * * *` |
| Setiap hari 02:00 & 14:00 WITA | `0 2,14 * * *` |
| Setiap hari Minggu 03:00 WITA | `0 3 * * 0` |

### Opsi 1 (alternatif): Cron job di host

Jika tidak memakai container backup, tambahkan baris di crontab host
(`crontab -e`). Contoh backup tiap hari 06:00 WITA pada server bertzona WITA:

```cron
0 6 * * * cd /opt/sigap && DB_HOST=localhost DB_USER=sigap_user DB_PASSWORD=xxxx \
  DB_NAME=sigap_db UPLOADS_DIR=/opt/sigap/backend/uploads BACKUP_DIR=/opt/sigap/backups \
  /opt/sigap/scripts/backup.sh >> /var/log/sigap-backup.log 2>&1
```

> [!NOTE]
> Container `backup` adalah **opsi yang disarankan** karena sudah menyatukan
> kredensial, jadwal, dan lokasi penyimpanan dalam satu tempat yang terkelola
> Compose. Pilih salah satu opsi, bukan keduanya sekaligus (hindari duplikasi).

---

## 7. Verifikasi

Setelah backup/restore, lakukan verifikasi berikut:

### Verifikasi backup berhasil

```bash
# Daftar isi volume backup
docker compose exec backup sh -c "ls -lhR /backups"

# Cek log container backup (menampilkan output backup.sh)
docker compose logs backup

# Pastikan tidak ada file lebih dari 7 hari (rotasi berjalan)
docker compose exec backup sh -c "find /backups -name '*.gz' -mtime +7 | wc -l"   # harus 0
```

### Verifikasi integritas file backup

```bash
# Cek dump SQL bisa dibaca (gzip valid + mengandung pernyataan CREATE TABLE)
docker compose exec backup sh -c "gunzip -c /backups/db/sigap_*.sql.gz | head -n 30"

# Cek archive uploads valid
docker compose exec backup sh -c "tar -tzf /backups/uploads/uploads_*.tar.gz"
```

### Uji restore end-to-end (data utuh)

Prosedur resmi verifikasi (jalankan di environment uji/DR):

1. Catat jumlah baris tabel utama sebelum restore (mis. `SELECT count(*) FROM cctvs;`).
2. Hentikan `api`: `docker compose stop api`.
3. Restore dari backup terbaru: `scripts/restore.sh <dump.sql.gz> <uploads.tar.gz>`.
4. Jalankan kembali `api`: `docker compose start api`.
5. Bandingkan jumlah baris tabel & file uploads dengan catatan langkah 1 — **harus sama**.
6. Login ke aplikasi (`https://localhost`) dan pastikan data tampil utuh.

---

## 8. Pemulihan Bencana (Disaster Recovery)

Urutan pemulihan penuh setelah kegagalan:

1. Pastikan service berhenti: `docker compose down`.
2. Bila volume DB hilang, jalankan ulang stack (membuat DB baru):
   `docker compose up -d db`, tunggu healthy.
3. Restore database dari backup terbaru: `docker compose exec backup sh -c
   "/usr/local/bin/restore.sh /backups/db/sigap_<latest>.sql.gz
   /backups/uploads/uploads_<latest>.tar.gz"`.
4. Jalankan seluruh stack: `docker compose up -d --build`.
5. Lakukan verifikasi (bagian 7).

---

## 9. Referensi

- Skrip: `scripts/backup.sh`, `scripts/restore.sh`.
- Container backup: `backup/Dockerfile`, `backup/entrypoint.sh`.
- Konfigurasi Compose: service `backup` & volume `backups_data` di `docker-compose.yml`.
- Status task: `docs/ROADMAP.md` Fase 7 (Task 7.3).
