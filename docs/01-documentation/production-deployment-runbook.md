# Production Deployment & Smoke Test Runbook

| Item | Detail |
|------|--------|
| **Issue** | #148 (P1 - Deployment dan smoke test production) |
| **Branch** | `docs/release-gate-runbooks` |
| **Versi acuan** | Backend `0.64.38`, Frontend `1.55.37`, migration terakhir `000029` |
| **Tanggal dokumen** | 2026-10-28 |
| **Status** | **BELUM DIEKSEKUSI** — runbook ini harus dijalankan dan diisi buktinya |

---

## 1. Tujuan, Ruang Lingkup, dan Peran

Runbook ini mencakup deployment production SIGAP setelah merge backlog
#139-#146, konfigurasi environment wajib, smoke test, dan rollback plan.

| Peran | Tanggung jawab |
|-------|----------------|
| Release manager | Menyetujui jendela deploy, memutuskan go/no-go, menandatangani hasil |
| Operator infra | Backup, build image, restart container, rollback |
| Penguji QA | Menjalankan smoke test dan menyimpan bukti |
| DBA/on-call | Memantau migration, query lambat, dan pemulihan database |

> **Gate:** jangan menyatakan "production ready" sebelum smoke test §6 lulus
> dan manual test #147 selesai (25/25 atau risiko diterima eksplisit).

---

## 2. Ringkasan Perubahan yang Akan Deploy

| Komponen | Perubahan | Dampak |
|----------|-----------|--------|
| Migration `000027_stream_verification` | Kolom `stream_verified_state`, `stream_verified_at`, `stream_verified_method`, `stream_verify_failures`, `stream_verify_error` pada `cctvs` | Aditif; aplikasi lama tetap dapat berjalan (kolom baru diabaikan) |
| Migration `000028_cctv_stream_logs` | Tabel `cctv_stream_logs` + index | Aditif; rollback menghapus seluruh riwayat log (destruktif) |
| Migration `000029_cctv_stream_logs_indexes` | Index komposit `(cctv_id, created_at)` dan `(event_type, created_at)`; index kolom tunggal dihapus | Rollback mengembalikan index kolom tunggal; tidak menghapus data |
| Backend `0.64.34`-`0.64.38` | Regression test keamanan, timezone WITA, retensi log, validasi event, kebijakan standby/dedup | Perlu rebuild image API |
| Frontend `1.55.36`-`1.55.37` | Filter/pencarian Log CCTV, URL state, retry, empty state | Perlu rebuild image frontend |

Catatan: issue #148 menyebut migrasi 27 dan 28. Sejak #144, migration terakhir
adalah **29**; deployment harus menerapkan sampai versi terbaru yang ada di
branch `main`.

---

## 3. Preflight Checklist

Jalankan minimal 24 jam sebelum jendela deploy:

- [ ] Branch `main` sudah berisi commit rilis; catat SHA:
  `git rev-parse HEAD`.
- [ ] CI Backend, Frontend, dan E2E pada commit tersebut PASS.
- [ ] Manual test #147 sudah lulus atau risiko diterima tertulis.
- [ ] Backup database + uploads terbaru tersedia dan **diuji restore** di
  staging.
- [ ] Salinan off-site backup tersedia (lihat `docs/BACKUP.md`).
- [ ] Kapasitas disk host ≥ 20% dan kapasitas volume `postgres_data` memadai.
- [ ] Secret tersedia: `DB_PASSWORD`, `JWT_SECRET`, `AES_SECRET_KEY`,
  kredensial NVR; tidak ada nilai nyata di source/chat.
- [ ] Sertifikat TLS domain valid dan tidak kedaluwarsa ≤30 hari.
- [ ] `MEDIAMTX_WEBRTC_ADDITIONAL_HOSTS` berisi domain/IP publik yang benar.
- [ ] Jendela deploy dan rollback disepakati; on-call tersedia.
- [ ] Pengguna diberi tahu potensi downtime singkat (restart API/frontend).

### 3.1 Baseline sebelum deploy

Simpan output berikut ke folder bukti:

```bash
git rev-parse HEAD
docker compose ps
docker compose images
curl -sk https://<domain>/api/health | jq
```

```sql
SELECT version, dirty FROM schema_migrations;
SELECT count(*) AS total_cctv FROM cctvs WHERE deleted_at IS NULL;
SELECT count(*) AS total_log FROM cctv_stream_logs;
SELECT event_type, count(*) FROM cctv_stream_logs GROUP BY event_type;
```

Catat juga jumlah kamera online/offline di UI sebagai pembanding setelah deploy.

---

## 4. Konfigurasi Environment

Periksa `backend/.env.docker` dan root `.env` (jangan commit nilainya):

| Variabel | Nilai production | Catatan |
|----------|------------------|---------|
| `APP_ENV` | `production` | Mengaktifkan security headers/HSTS; Swagger nonaktif |
| `APP_TIMEZONE` | `Asia/Makassar` | Batas "hari ini" ringkasan Log CCTV |
| `TRUSTED_PROXY_IPS` | `172.28.0.10` | Hanya Nginx yang boleh mengirim XFF; jangan longgarkan |
| `CCTV_VERIFY_INTERVAL_MINUTES` | `15` | Nilai valid 5-30; di luar itu fallback 15 |
| `CCTV_VERIFY_CONCURRENCY` | `1` | Nilai valid 1-4; naikkan hanya bila NVR kuat |
| `CCTV_METRO_E_DIRECT_PROBE` | `false` | `true` hanya bila host benar-benar punya rute ke `10.20.105.x` |
| `MEDIAMTX_WEBRTC_ADDITIONAL_HOSTS` | `<domain-publik>` | Wajib; jangan `172.28.x.x`/`localhost` |
| `CCTV_STREAM_LOG_RETENTION_DAYS` | `90` | Kosong/invalid/`<=0` fallback 90 |
| `CCTV_LOG_STANDBY_EVENTS` | `false` | Aktifkan hanya untuk audit sementara |
| `CCTV_METRO_E_DIRECT_PROBE` | `false` | Menghindari offline palsu bila tidak ada rute |
| `LOG_LEVEL` | `info` | Jangan `debug` permanen di production |
| `REDIS_PASSWORD` | secret | Harus sama di root `.env` dan backend |
| `RTSP_NVR_HOST/PORT/USER` | secret/placeholder | Jangan tulis kredensial di dokumentasi |

Validasi compose tanpa menjalankan container:

```bash
docker compose config >/dev/null && echo "COMPOSE_OK"
```

**Kriteria lulus:** tidak ada error variabel wajib kosong
(`MEDIAMTX_WEBRTC_ADDITIONAL_HOSTS` memakai `:?` di compose).

---

## 5. Prosedur Deployment

> Lakukan pada jendela yang disetujui. Jangan menjalankan `docker compose down -v`
> di production — volume dapat terhapus.

### 5.1 Backup wajib

```bash
# Dari root repository; sesuaikan dengan prosedur docs/BACKUP.md
docker compose exec backup /scripts/backup.sh
ls -lh backups/
```

Verifikasi file backup terbaru ada dan ukurannya wajar. Untuk backup manual
database:

```bash
docker compose exec -T db pg_dump -U "$DB_USER" -d "$DB_NAME" -Fc \
  > "backups/predeploy-$(date +%Y%m%d-%H%M).dump"
```

### 5.2 Tarik kode dan build image

```bash
git fetch origin
git checkout main
git pull --ff-only origin main
git rev-parse HEAD   # samakan dengan commit yang lolos CI

docker compose build api frontend
```

**Kriteria lulus:** build selesai tanpa error; image baru memiliki tag/digest
yang tercatat.

### 5.3 Terapkan migration dan restart

API menjalankan migration otomatis saat startup (`RunMigrations`). Urutan aman:

```bash
docker compose up -d db redis
docker compose up -d api
docker compose logs -f api
```

Tunggu log:

```text
✅ Migrasi DB versioned berhasil dijalankan!
```

Lalu:

```bash
docker compose up -d frontend nginx mediamtx
docker compose ps
```

Verifikasi versi migration:

```sql
SELECT version, dirty FROM schema_migrations;
```

**Kriteria lulus:** `version = 29`, `dirty = false`. Bila `dirty = true`, hentikan
deploy dan ikuti §8.

Alternatif CLI (dari host yang memiliki Go dan akses DB):

```bash
cd backend
go run cmd/migrate/main.go version
go run cmd/migrate/main.go up
go run cmd/migrate/main.go version
```

> **Peringatan:** `go run cmd/migrate/main.go down` melakukan rollback **semua**
> migration ke versi 0. Jangan pakai perintah itu untuk rollback production.
> Lihat §8.

### 5.4 Health check awal

```bash
curl -sk https://<domain>/api/health | jq
docker compose logs --tail=100 api | grep -Ei 'error|fatal|panic' || true
docker compose logs --tail=100 mediamtx | grep -Ei 'error|fatal' || true
```

**Kriteria lulus:** health `status=ok`, database/Redis terhubung, tidak ada
panic/fatal berulang.

---

## 6. Smoke Test Production

Isi kolom bukti dengan tautan file/screenshot. Jalankan berurutan.

### 6.1 Login, sesi, dan CSRF

| No | Langkah | Hasil yang diharapkan | Bukti |
|----|---------|-----------------------|-------|
| 1 | Buka `https://<domain>`, login admin. | Redirect ke dashboard; cookie `sigap_session` httpOnly, `sigap_csrf` tersedia; tidak ada error console. | |
| 2 | Refresh halaman dan buka menu lain. | Sesi bertahan; tidak logout sendiri. | |
| 3 | Lakukan aksi mutasi (mis. edit profil/CCTV uji). | Request membawa `X-CSRF-Token`; respons sukses; tanpa CSRF → 403 (uji via DevTools). | |
| 4 | Logout lalu login ulang. | Sesi lama tidak valid; login baru berhasil. | |

### 6.2 Rate limit dan trusted proxy

| No | Langkah | Hasil yang diharapkan | Bukti |
|----|---------|-----------------------|-------|
| 5 | Dari satu IP, lakukan login gagal berulang (mis. 7 kali). | Setelah ambang (5/menit), respons **429**; tidak ada 429 pada request normal sebelumnya. | |
| 6 | Buka beberapa halaman/dashboard dengan kecepatan normal (<100 req/menit). | Tidak ada 429 palsu; aktivitas normal tetap lancar. | |
| 7 | Coba spoofing langsung ke API internal dengan header `X-Forwarded-For: 1.2.3.4` (dari host yang tidak diizinkan). | Header diabaikan; rate limit/audit memakai IP koneksi sebenarnya; akses langsung tidak bisa memalsukan IP. | |
| 8 | Periksa log audit/rate limit. | IP yang tercatat adalah IP klien melalui Nginx, bukan `172.28.x.x` atau nilai spoof. | |

### 6.3 Live status dan manual verify

| No | Langkah | Hasil yang diharapkan | Bukti |
|----|---------|-----------------------|-------|
| 9 | Buka peta/monitor; amati badge kamera. | Status live/offline/idle/unknown tampil sesuai; tidak semua kamera menjadi offline palsu. | |
| 10 | Buka live view satu kamera H.264. | Frame tampil ≤15 detik; tidak ada error WHEP/CORS. | |
| 11 | Panggil verifikasi manual (admin):<br>`POST /api/streams/<UUID>/verify` dengan CSRF. | Respons sukses berisi state/metode/durasi; kolom `stream_verified_*` terisi. | |
| 12 | Amati kamera yang baru diverifikasi. | Badge/status konsisten; tidak ada banjir event `standby` (default nonaktif). | |

### 6.4 Halaman Log CCTV

| No | Langkah | Hasil yang diharapkan | Bukti |
|----|---------|-----------------------|-------|
| 13 | Buka `/cctv/logs`. | Daftar memuat log; ringkasan hari ini tampil; timezone `Asia/Makassar`. | |
| 14 | Filter `online`, `offline`, `standby`, `unknown`, dan cari nama/kode. | Filter bekerja; URL menyimpan state; angka ringkasan konsisten dengan daftar. | |
| 15 | Kirim `event_type=invalid` lewat URL/API. | HTTP **400** dengan pesan nilai yang diizinkan; bukan daftar kosong palsu. | |
| 16 | Restart API, lalu buka halaman log lagi. | Cleanup retensi berjalan (log >90 hari terhapus); log baru tetap ada. | |
| 17 | Amati kamera idle selama 2 siklus probe. | Tidak ada deretan event `standby` baru; status tetap diperbarui. | |

### 6.5 WebSocket

| No | Langkah | Hasil yang diharapkan | Bukti |
|----|---------|-----------------------|-------|
| 18 | Buka DevTools → Network → WS; pastikan `/api/ws` terhubung. | Handshake 101; koneksi authenticated via cookie. | |
| 19 | Picu perubahan status (probe/verify) atau event maintenance. | Event real-time diterima; UI diperbarui tanpa refresh. | |
| 20 | Putuskan jaringan sebentar lalu pulihkan. | WebSocket reconnect; tidak ada duplikasi UI parah. | |

### 6.6 Multi-site

| No | Langkah | Hasil yang diharapkan | Bukti |
|----|---------|-----------------------|-------|
| 21 | Login admin, pilih Site A pada halaman CCTV/log. | Hanya data Site A; total/paginasi konsisten. | |
| 22 | Login teknisi Site A; akses CCTV, log, maintenance, field check. | Hanya data Site A; permintaan lintas site → 403. | |
| 23 | Coba akses langsung data Site B via API dengan sesi teknisi Site A. | Ditolak 403; tidak ada kebocoran. | |
| 24 | Kembali ke admin dan pilih "Semua site". | Data semua site tampil sesuai kewenangan admin. | |

### 6.7 Streaming publik/seluler (rujukan Bug 8)

| No | Langkah | Hasil yang diharapkan | Bukti |
|----|---------|-----------------------|-------|
| 25 | Dari jaringan publik, buka live view dan rekam 15-30 detik. | Frame bergerak; WHEP sukses; kandidat ICE publik. | |
| 26 | Dari Android seluler (Wi-Fi off), buka live view. | Frame tampil ≤15 detik; reconnect saat jaringan berubah. | |
| 27 | Periksa `/monitor` dua kamera. | Kedua sel live; tidak ada placeholder permanen. | |

---

## 7. Post-deploy Monitoring (30-60 Menit)

- [ ] `docker compose ps` semua service `healthy`/`running`.
- [ ] Error rate API tidak meningkat; tidak ada panic.
- [ ] `schema_migrations.version = 29`, `dirty = false`.
- [ ] Jumlah event log per tipe wajar; tidak ada lonjakan `standby`.
- [ ] Kamera offline sesuai kondisi nyata, bukan massal.
- [ ] Latensi API dan query log dalam batas normal.
- [ ] Backup otomatis berikutnya berjalan.
- [ ] Tidak ada 429 palsu pada pengguna normal.
- [ ] WebSocket stabil; tidak ada reconnect loop.

Simpan grafik/log monitoring ke folder bukti.

---

## 8. Rollback Plan

### 8.1 Prinsip

1. **Utamakan restore backup + image sebelumnya** daripada rollback schema
   parsial. Data production lebih penting daripada waktu.
2. `go run cmd/migrate/main.go down` **tidak boleh** dipakai di production:
   perintah itu me-rollback semua migration ke versi 0.
3. Rollback migration 28 bersifat **destruktif** (menghapus tabel
   `cctv_stream_logs`). Export log lebih dulu bila masih dibutuhkan.
4. Rollback 27 menghapus kolom verifikasi; aplikasi versi sebelumnya tidak
   membutuhkannya, tetapi data verifikasi historis hilang.
5. Rollback 29 hanya mengembalikan index; aman terhadap data.

### 8.2 Kapan rollback

Putuskan rollback bila salah satu terjadi:

- API gagal start berulang / health gagal >5 menit.
- Migration `dirty=true` dan tidak dapat diperbaiki cepat.
- Login/CSRF/WebSocket lumpuh untuk sebagian besar pengguna.
- Streaming massal gagal karena konfigurasi deploy.
- Data log/history hilang atau korup.

### 8.3 Opsi A - Restore backup (disarankan)

```bash
docker compose stop api frontend
# Restore database dari dump pre-deploy
docker compose exec -T db pg_restore -U "$DB_USER" -d "$DB_NAME" --clean --if-exists \
  < "backups/predeploy-YYYYMMDD-HHMM.dump"
# Deploy image/tag sebelumnya
git checkout <commit-sebelumnya>
docker compose build api frontend
docker compose up -d api frontend nginx
```

Verifikasi health, login, dan jumlah data. Catat RPO/RTO aktual.

### 8.4 Opsi B - Rollback schema bertarget (hanya dengan DBA)

Gunakan hanya bila backup restore tidak memungkinkan. Selalu backup ulang
sebelumnya.

**Migration 29 (index):**

```sql
DROP INDEX IF EXISTS idx_cctv_stream_logs_event_created;
DROP INDEX IF EXISTS idx_cctv_stream_logs_cctv_created;
CREATE INDEX IF NOT EXISTS idx_cctv_stream_logs_cctv_id ON cctv_stream_logs(cctv_id);
CREATE INDEX IF NOT EXISTS idx_cctv_stream_logs_event_type ON cctv_stream_logs(event_type);
UPDATE schema_migrations SET version = 28, dirty = false;
```

**Migration 28 (tabel log — destruktif):**

```sql
-- EKSPOR DULU: \copy (SELECT * FROM cctv_stream_logs) TO '/tmp/cctv_logs.csv' CSV HEADER
DROP TABLE IF EXISTS cctv_stream_logs;
UPDATE schema_migrations SET version = 27, dirty = false;
```

**Migration 27 (kolom verifikasi — destruktif):**

```sql
ALTER TABLE cctvs
    DROP COLUMN IF EXISTS stream_verify_error,
    DROP COLUMN IF EXISTS stream_verify_failures,
    DROP COLUMN IF EXISTS stream_verified_method,
    DROP COLUMN IF EXISTS stream_verified_at,
    DROP COLUMN IF EXISTS stream_verified_state;
UPDATE schema_migrations SET version = 26, dirty = false;
```

Setelah rollback schema, deploy image versi sebelumnya dan jalankan smoke test
minimum (login, CCTV list, live status, log).

### 8.5 Uji rollback di staging (wajib sebelum production)

1. Ambil snapshot/clone staging.
2. Catat `schema_migrations` dan jumlah data.
3. Simulasikan rollback 29 → 28 → 27 menggunakan SQL §8.4 pada clone.
4. Deploy image versi sebelumnya ke clone.
5. Verifikasi aplikasi lama tetap berjalan (login, CCTV, live status).
6. Restore snapshot; deploy versi baru lagi; verifikasi `version=29`.
7. Simpan log/screenshot sebagai bukti issue #148.

**Kriteria lulus:** rollback dan re-deploy berhasil di staging, data yang
dipertahankan sesuai harapan, dan RTO tercatat.

---

## 9. Go/No-Go dan Sign-off

Deploy dinyatakan **GO** hanya bila:

- [ ] Preflight §3 lengkap.
- [ ] Backup terverifikasi.
- [ ] Migration `version=29`, `dirty=false`.
- [ ] Smoke test §6 nomor 1-24 lulus; nomor 25-27 mengikuti hasil #147.
- [ ] Tidak ada 429 palsu.
- [ ] Log dan status live tampil benar.
- [ ] Rollback sudah diuji di staging.

| Peran | Nama | Tanggal | Keputusan |
|-------|------|---------|-----------|
| Release manager | | | GO / NO-GO |
| Operator infra | | | |
| QA | | | |
| DBA | | | |

**Catatan risiko yang diterima (bila ada):**

______________________________________________________________

---

## 10. Lampiran - Perintah Cepat

```bash
# Status dan log
docker compose ps
docker compose logs --tail=200 api
docker compose logs --tail=200 mediamtx
docker compose logs --tail=100 nginx

# Health
curl -sk https://<domain>/api/health | jq

# Migration (host dengan Go)
cd backend
go run cmd/migrate/main.go version

# Ukuran tabel log
docker compose exec db psql -U "$DB_USER" -d "$DB_NAME" -c \
  "SELECT pg_size_pretty(pg_total_relation_size('cctv_stream_logs')) AS size,
          count(*) AS rows FROM cctv_stream_logs;"
```

---

## 11. Referensi

- `docs/01-documentation/ACCESS.md` — akses deployment via Nginx/TLS.
- `docs/MIGRATION.md` — prosedur migration dan penanganan dirty state.
- `docs/BACKUP.md` — backup/restore dan enkripsi.
- `docs/01-documentation/cctv-stream-log-semantics.md` — semantics log CCTV.
- `docs/04-manual-tests/manual-test-release-gate-bugs-4-8-21.md` — gate manual
  test #147.
- `docs/05-reports/TODO-PERBAIKAN-2026-10-27.md` — backlog induk.
