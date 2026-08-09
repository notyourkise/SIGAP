# ROADMAP — SIGAP

> Rencana pengembangan SIGAP — dari perbaikan keamanan hingga streaming CCTV real-time.
> Disusun: 2026-08-01 · Diperbarui: 2026-08-07

## Visi & Prinsip Prioritas

Pengembangan SIGAP diurutkan dari yang **paling berdampak pada keamanan** menuju **peningkatan
kapabilitas** dan **kesiapan produksi**. Prinsip:

1. **Keamanan dulu** — perbaikan kritis tidak boleh ditunda.
2. **Setiap fase dapat dirilis** — jangan menumpuk perubahan raksasa.
3. **Verifikasi wajib** — setiap fase lulus test, lint, build, dan uji manual.
4. **Konvensi tetap** — kerja di branch baru (`feature/…` / `bugfix/…`), Conventional Commits,
   dan catat perubahan di `../backend/changelog.md` / `../frontend/changelog.md`.
5. **Data menjadi insight** — fitur baru harus mengubah data yang sudah terkumpul menjadi
   informasi yang actionable bagi admin dan pimpinan.

---

## Status Pengerjaan

| Fase | Fokus | Progress | Status |
|------|-------|----------|--------|
| 1 | Kritis (Prioritas Tertinggi) | 6/6 | ✅ Selesai |
| 2 | Penguatan (Hardening) | 9/9 | ✅ Selesai |
| 3 | Fitur (Kapabilitas) | 9/9 | ✅ Selesai |
| 4 | Infrastruktur & Kualitas | 2/8 + 4.1 sebagian | ⏸️ Sisa dilebur ke Fase 7 |
| 5 | Alur Kerja Operasional (Jalur A) | 5/5 | ✅ Selesai |
| 6 | Kecerdasan Operasional | 4/4 | ✅ Selesai |
| 7 | Kesiapan Produksi | 6/6 | ✅ Selesai |
| 8 | Pengalaman Lapangan | 4/4 | ✅ Selesai |
| **9** | **Streaming CCTV Real-Time** | **6/6** | **✅ Selesai** |
| **10** | **Kecerdasan Visual & Snapshot** | **4/4** | **✅ Selesai** |
| **11** | **Manajemen Operasional Lanjut** | **5/5** | **✅ Selesai** |
| **12** | **Integrasi & Skalabilitas** | **0/4** | **🔲 Belum dimulai** |

> **Urutan pengerjaan:**
> ```
> Fase 1–8 ✅ (semua selesai)
>     │
>     ▼
> Fase 9 (Streaming CCTV) ──→ Task 11.1 (Suku Cadang)
>     │                              │
>     ▼                              ▼
> Fase 10 (Visual)            sisa Fase 11 (Operasional)
>     │                              │
>     └──────────┬───────────────────┘
>                ▼
>          Fase 12 (Integrasi)
> ```
> - **Fase 9** (streaming): prioritas utama — mengubah peta sebaran dari view-only menjadi live monitoring.
> - **Task 11.1** (suku cadang): dikerjakan langsung setelah Fase 9 stabil (kebutuhan vital operasional).
> - **Fase 10** (visual) dan **sisa Fase 11** bisa dikerjakan paralel setelahnya.
> - **Fase 12** (integrasi): dievaluasi berdasarkan kebutuhan pasca go-live.
> - Di setiap tabel, task yang berada di posisi paling atas adalah yang **harus dikerjakan lebih dulu**.

---

## Fase 1 — Kritis (Prioritas Tertinggi)

> Menutup celah keamanan paling berbahaya. **Status: ✅ SELESAI** — semua task selesai & terverifikasi
> (`go test ./...` PASS + `npm run build` sukses). Catatan: `../backend/changelog.md` 0.8.0 & `../frontend/changelog.md` 0.7.0.

### Task

| # | Task | File Terkait | Skala | Status |
|---|------|--------------|-------|--------|
| 1.1 | Hapus JWT secret fallback & seeder password fallback dari source; fail-fast penuh | `backend/utils/jwt.go`, `backend/models/seeder.go`, `backend/main.go` | Kecil | ✅ Selesai |
| 1.2 | Ubah `SeedAdmin`: hanya buat admin jika belum ada (jangan reset password tiap startup) | `backend/models/seeder.go` | Kecil | ✅ Selesai |
| 1.3 | Tambahkan autentikasi JWT pada handshake WebSocket; validasi token client | `backend/routes/websocket.go`, `backend/utils/websocket.go`, `frontend/src/context/WebSocketContext.tsx` | Sedang | ✅ Selesai |
| 1.4 | Perbaiki stored-XSS pada print window QR (render label via React/DOM API, escape semua field) | `frontend/src/app/cctv/page.tsx` | Sedang | ✅ Selesai |
| 1.5 | Generate QR server-side (endpoint single + batch) dan ganti `api.qrserver.com` | `backend/handlers/cctv.go`, `backend/routes/cctv.go`, `frontend/src/app/cctv/page.tsx` | Besar | ✅ Selesai |
| 1.6 | Batasi `limit` & `offset` pagination (cap maks, mis. 100) | `backend/utils/pagination.go` | Kecil | ✅ Selesai |

### Verifikasi Fase 1

- `cd backend && go test ./...` — semua PASS.
- `cd frontend && npm run build` — sukses tanpa error.
- Manual: login → scan QR → buka cetak QR (tidak ada teks mentah yang tampil); coba koneksi WebSocket
  tanpa token (harus ditolak); restart server → password admin tidak berubah.

---

## Fase 2 — Penguatan (Hardening)

> Memperkuat pertahanan menyeluruh dan menyiapkan basis produksi.
> **Status: ✅ SELESAI** — semua task selesai & terverifikasi
> (`go test ./...` PASS + `npm run build` sukses). Catatan: `../backend/changelog.md` 0.9.0 &
> `../frontend/changelog.md` 0.8.0.

### Task

| # | Task | File Terkait | Skala | Status |
|---|------|--------------|-------|--------|
| 2.1 | Migrasi token ke cookie httpOnly + CSRF protection | `frontend/src/lib/api.ts`, `frontend/src/context/AuthContext.tsx`, `backend/middleware/auth.go` | Besar | ✅ Selesai |
| 2.2 | Revoke token lama saat reset password / role diubah (token version `tv`) | `backend/utils/jwt.go`, `backend/handlers/user.go`, `backend/utils/blacklist.go` | Sedang | ✅ Selesai |
| 2.3 | Rate limiter & audit hanya percaya header proxy dari trusted proxy | `backend/middleware/limiter.go`, `backend/utils/audit.go`, `backend/main.go` | Kecil | ✅ Selesai |
| 2.4 | Batasi ukuran file & jumlah baris import Excel | `backend/handlers/cctv.go` | Kecil | ✅ Selesai |
| 2.5 | CORS: fail-fast jika env kosong; daftar origin eksplisit di produksi | `backend/main.go` | Kecil | ✅ Selesai |
| 2.6 | Docker: hapus quick tunnel, reverse proxy + TLS, secret management, jangan publish 5432 | `docker-compose.yml`, file infra baru | Sedang | ✅ Selesai |
| 2.7 | Aktifkan CSP & HSTS di `SecurityHeaders` (produksi) | `backend/middleware/security.go` | Kecil | ✅ Selesai |
| 2.8 | Ownership check update maintenance log (teknisi hanya log miliknya) | `backend/handlers/maintenance.go` | Sedang | ✅ Selesai |
| 2.9 | Naikkan kebijakan password (min 8 + kompleksitas) | `backend/handlers/auth.go`, `backend/handlers/user.go`, `frontend/src/app/users/page.tsx` | Kecil | ✅ Selesai |

### Verifikasi Fase 2

- `cd backend && go build ./... && go test ./...` — semua PASS (termasuk test baru: klaim `tv`,
  validator `strongpassword`, login → csrf_token + cookie httpOnly).
- `cd frontend && npm run build` — sukses tanpa error (TypeScript bersih).
- `docker compose config --quiet` — konfigurasi compose valid (secret via root `.env`).
- Manual: login di browser biasa (token tidak terlihat di localStorage/DevTools); reset password user
  → token lama langsung ditolak; upload file > batas ditolak; import Excel besar ditolak; teknisi
  mencoba mengubah log milik teknisi lain → 403.

---

## Fase 3 — Fitur (Kapabilitas)

> Meningkatkan nilai operasional sistem untuk admin & teknisi di lapangan IKN.
> **Status: ✅ SELESAI** (3.1–3.9, diimplementasikan 2026-08-01).

### Task

| # | Task | File Terkait | Skala | Status |
|---|------|--------------|-------|--------|
| 3.1 | Self-service change password + force-change saat login pertama | `backend/handlers/auth.go`, `backend/routes/auth.go`, `frontend/src/app/login/page.tsx`, komponen profil | Sedang | ✅ Selesai |
| 3.2 | Model sesi JWT 5 jam tanpa sliding session + revoke token | `backend/utils/jwt.go`, `backend/utils/session.go`, `backend/handlers/user.go`, `frontend/src/context/AuthContext.tsx` | Sedang | ✅ Selesai |
| 3.3 | Dashboard analitik: tren pemeliharaan, distribusi status, peringatan overdue | `backend/handlers/dashboard.go` (baru), `frontend/src/app/dashboard/page.tsx` | Besar | ✅ Selesai |
| 3.4 | Notifikasi tugas untuk teknisi | backend + `frontend/src/app/dashboard/page.tsx` | Besar | ✅ Selesai |
| 3.5 | Halaman profil pengguna | `frontend/src/app/profile/page.tsx` (baru), `backend/handlers/auth.go` | Kecil | ✅ Selesai |
| 3.6 | Unifikasi model `assets` ↔ `cctv` (satu sumber kebenaran) | `backend/models/`, `backend/handlers/maintenance.go`, migrasi DB | Besar | ✅ Selesai |
| 3.7 | Laporan PDF + export audit-log + filter rentang tanggal | `backend/handlers/audit.go`, `backend/handlers/report.go` (baru), frontend | Sedang | ✅ Selesai |
| 3.8 | Cleanup file yatim upload + strip EXIF GPS | `backend/handlers/upload.go`, `backend/main.go` | Kecil | ✅ Selesai |
| 3.9 | Batasi ekspor per role | `backend/routes/maintenance.go`, `backend/handlers/maintenance.go`, `backend/handlers/report.go` | Kecil | ✅ Selesai |

### Verifikasi Fase 3

- Semua test & build lulus.
- Manual: user baru dipaksa ganti password; notifikasi muncul saat tugas terjadwal; laporan PDF
  terunduh; ekspor dibatasi sesuai role.

> Catatan task 3.2: rencana awal menggunakan refresh token/sliding session. Setelah evaluasi
> stabilitas autentikasi, implementasi final memakai JWT statis 5 jam, cookie sesi/CSRF dengan
> `MaxAge` 18000 detik, serta revoke berbasis blacklist dan token version.

---

## Fase 4 — Infrastruktur & Kualitas (Sebagian Selesai)

> **Status: ⏸️ SISA DILEBUR KE FASE 7** — 4.1 (sebagian), 4.2, dan 4.8 selesai;
> task 4.3–4.7 kini menjadi bagian dari Fase 7 (Kesiapan Produksi) dengan scope yang disesuaikan.
> Referensi lengkap cakupan test & gotchas: `docs/TESTING.md`.

| # | Task | Keterangan | Status |
|---|------|-----------|--------|
| 4.1 | Perluas unit test backend (cctv, upload, websocket, pagination, rate limit) | `backend/**/*_test.go` | 🚧 Sebagian besar (jalur CRUD, import boundary, WebSocket valid/tanpa token, routes, models, DSN, dan resolusi migration sudah diuji; valid import transaksi penuh dan migration update penuh ditunda) |
| 4.2 | Tambah `npm run lint` + `golangci-lint` di CI | `.github/workflows/ci.yml` | ✅ Selesai |
| 4.3 | Migrasi DB versioned (golang-migrate/Atlas) | → Dilebur ke **Fase 7.1** | ➡️ Pindah |
| 4.4 | Redis untuk rate-limit & blacklist (multi-instance) | → Dilebur ke **Fase 7.2** | ➡️ Pindah |
| 4.5 | Backup DB (`pg_dump` cron) + backup uploads + restore script | → Dilebur ke **Fase 7.3** | ➡️ Pindah |
| 4.6 | Monitoring: endpoint metrics (Prometheus) | → Dilebur ke **Fase 7.4** | ➡️ Pindah |
| 4.7 | Container non-root, resource limits, secret management | → Dilebur ke **Fase 7.5** | ➡️ Pindah |
| 4.8 | Isi `docs/changelog.md` (milestone monorepo) | `docs/changelog.md` | ✅ Selesai |

---

## Fase 5 — Alur Kerja Operasional (Jalur A)

> Menutup gap alur kerja harian admin & teknisi: notifikasi penugasan, sinkronisasi status
> perangkat, otomasi jadwal preventif, pemindahan tugas, dan eskalasi keterlambatan.
> **Status: ✅ SELESAI** — disepakati 2026-08-04 dan seluruh task 5.1–5.5 telah diimplementasikan serta diverifikasi. Sistem masih pilot/UAT; notifikasi **in-app saja**.

### Keputusan Desain (disepakati 2026-08-04)

- **Notifikasi in-app saja** — bell icon + daftar notifikasi + toast real-time via WebSocket; tanpa layanan eksternal (email/SMTP tidak dipakai).
- **Status CCTV otomatis bersifat konservatif** — hanya transisi `aktif ↔ pemeliharaan` yang dilakukan sistem; status `rusak`/`nonaktif` tetap keputusan manual admin dan tidak pernah ditimpa otomatis.
- **Prasyarat arsitektur** — `WSHub` diperluas agar menyimpan identitas koneksi (`user_id`, `role`) sehingga mendukung `BroadcastToUser`/`BroadcastToRole`; event broadcast global yang ada tetap dipertahankan (backward-compatible).

### Task (berurutan — kerjakan dari atas ke bawah)

| # | Task | Keterangan | Status |
|---|------|-----------|--------|
| 5.1 | Notifikasi in-app penugasan real-time | Model `Notification`; endpoint `GET /api/notifications`, `GET /api/notifications/unread-count`, `PUT /api/notifications/:id/read`, `PUT /api/notifications/read-all`; util `NotifyUser`/`NotifyRole`; perluasan `WSHub` (targeted per-user/per-role); komponen `NotificationBell` + integrasi `DashboardLayout`; hook: assign → teknisi, complete → admin, field-check `attention` → admin; event WS `NOTIFICATION_CREATED`. Branch: `feature/notification-in-app` | ✅ Kode selesai (2026-08-04) — menunggu uji runtime Docker di `https://localhost` |
| 5.2 | Sinkronisasi status CCTV otomatis (konservatif) | Helper dalam transaksi complete/field-check; `aktif→pemeliharaan` saat tugas gagal / hasil `attention`; `pemeliharaan→aktif` saat checklist 9/9 ok; `nonaktif` tidak disentuh (status `rusak` digabung ke `nonaktif` per 2026-08-05 — enum resmi: `aktif`/`pemeliharaan`/`nonaktif`); audit `UPDATE_CCTV` (keterangan otomatis) + broadcast `CCTV_UPDATED`. Branch: `feature/sync-cctv-status` | ✅ Selesai (2026-08-05) — uji manual S1–S8 `docs/manual-test-sync-cctv.md` + 4 perbaikan lanjutan (validasi jelas, normalisasi Inggris, guard legacy uppercase, dropdown status) |
| 5.3 | Re-assign tugas oleh admin | `PUT /maintenance-logs/:id/assign` diperluas ke log `ongoing`; audit `REASSIGN_MAINTENANCE`; notif `MAINTENANCE_ASSIGNED` ke teknisi baru & `MAINTENANCE_UNASSIGNED` ke teknisi lama; tombol "Alihkan" di UI admin; tolak assign ke teknisi yang sama (400). Branch: `feature/reassign-maintenance` | ✅ Selesai (2026-08-05) — build/test/vet + lint/build PASS; menunggu uji manual runtime Docker (teknisi lama dapat notif "dialihkan", teknisi baru dapat notif "tugas baru") |
| 5.4 | Penjadwalan pemeliharaan berulang (recurring) | Model `MaintenanceSchedule` (frequency `daily`/`weekly`/`monthly`, `next_run_at`, `is_active`); kolom `MaintenanceScheduleID` di `MaintenanceLog` (link + idempotensi); goroutine generator harian (jalan juga saat startup, idempotent); CRUD admin `/api/maintenance-schedules`; halaman admin `/schedules` + menu sidebar | ✅ Selesai (2026-08-06) — build/test/vet + lint/build PASS; generator idempotent per hari per jadwal, log hasil generator mengikuti alur yang sama (assign → 5.3) |
| 5.5 | Eskalasi overdue otomatis | Kolom `OverdueNotifiedAt` di `MaintenanceLog`; goroutine saat startup lalu tiap jam: log `scheduled`/`ongoing` lewat jadwal → notif `MAINTENANCE_OVERDUE` ke semua admin (+ teknisi bila ongoing), terkirim sekali saja; flag di-reset saat reschedule | ✅ Selesai (2026-08-06) — build/test/vet + `swag init` PASS; test sqlmock mencakup notifikasi admin/teknisi, skip flag, status terminal, dan reset reschedule |

### Ketergantungan Antar Task

- 5.3 & 5.5 memakai sistem notifikasi dari 5.1 — **wajib setelah 5.1**.
- 5.2 berdiri di atas transaksi existing (bisa paralel dengan 5.1, tetapi notifikasi hasil bermasalah memakai 5.1).
- 5.4 berdiri sendiri; log hasil generator mengikuti alur yang sama (assign → 5.3, overdue → 5.5).

### Verifikasi Fase 5 (per task)

- `cd backend && go build ./... && go test ./... -count=1 && go vet ./...` — PASS (test sqlmock baru untuk tiap handler/job baru).
- `cd frontend && npm run lint && npm run build` — PASS.
- `swag init` setelah anotasi endpoint baru; uji runtime Docker di `https://localhost`.
- Uji manual per task: 5.1 lonceng notifikasi muncul real-time saat assign; 5.2 status CCTV berubah sesuai tabel keputusan (dan `nonaktif` — termasuk legacy `rusak` — tidak berubah); 5.3 alihkan tugas ke teknisi lain; 5.4 jadwal berulang membuat log otomatis tanpa duplikat; 5.5 notif overdue terkirim sekali saja dan flag kembali kosong setelah reschedule ke masa depan.

---

## Fase 6 — Kecerdasan Operasional

> Mengubah data yang sudah terkumpul menjadi **insight yang actionable** bagi admin dan pimpinan.
> Data pemeliharaan, pengecekan lapangan, dan tugas sudah tersedia — fase ini menarik nilai dari data tersebut.
> **Status: ✅ SELESAI** — seluruh Task 6.1–6.4 selesai dan terverifikasi.

### Task (berurutan — kerjakan dari atas ke bawah)

| # | Task | Keterangan | Skala | Status |
|---|------|-----------|-------|--------|
| 6.1 | Skor Kesehatan CCTV | Skor 0–100 per CCTV dihitung otomatis dari 5 faktor: jumlah pemeliharaan gagal (bobot 30%), frekuensi pengecekan `attention` (25%), keterlambatan tugas (20%), umur sejak pemeliharaan terakhir berhasil (15%), jumlah item checklist `not_ok` (10%). Endpoint `GET /api/cctv/:id/health-score` + kolom `health_score` di list CCTV + badge visual (hijau ≥70 / kuning ≥40 / merah <40) di tabel & peta sebaran. Goroutine penghitungan ulang berkala (tiap 6 jam) atau saat event complete/field-check selesai. | Besar | ✅ Selesai (2026-08-06) — test kalkulasi/endpoint + backend build/test/vet dan frontend lint/build PASS |
| 6.2 | Dashboard Kinerja Teknisi | Endpoint `GET /api/dashboard/technician-performance` (admin): completion rate, rata-rata waktu penyelesaian (assign → complete), jumlah tugas overdue, jumlah pengecekan lapangan, kualitas (rasio 9/9 ok vs total). Halaman admin `/kinerja-teknisi` + menu sidebar, tabel ranking + bar chart per teknisi. Filter rentang waktu (7 hari / 30 hari / kustom). | Sedang | ✅ Selesai (2026-08-06) — test struktur/filter/role + backend/frontend build dan lint PASS |
| 6.3 | Laporan PDF Pengecekan Lapangan | Endpoint `GET /api/field-maintenance/:id/report-pdf`: laporan per-record dengan kop surat IKN (pola sama dengan laporan pemeliharaan), detail 9 item checklist (nama, hasil, catatan), foto bukti, identitas teknisi & CCTV, waktu pengecekan. Tombol "Cetak Laporan" di modal detail pengecekan lapangan (admin). | Sedang | ✅ Selesai (2026-08-06) — test PDF/404/ownership, `swag init`, backend/frontend build dan lint PASS |
| 6.4 | Ringkasan Operasional Otomatis | Goroutine harian (06:00 WITA): hitung ringkasan 24 jam terakhir — tugas baru/selesai/gagal/overdue, pengecekan lapangan, CCTV berubah status — simpan di model `DailyDigest`. Endpoint `GET /api/dashboard/digest` (admin): daftar ringkasan harian. Halaman `/ringkasan` dengan timeline harian. Notifikasi in-app ke admin saat ringkasan tersedia. | Sedang | ✅ Selesai (2026-08-06) — test generator/idempotensi/endpoint, `swag init`, backend/frontend build dan lint PASS |

### Alasan Urutan

- 6.1 paling berdampak — skor kesehatan langsung menjawab "CCTV mana yang perlu prioritas?"
- 6.2 membantu evaluasi beban kerja & redistribusi tugas (melengkapi 5.3 re-assign).
- 6.3 melengkapi siklus dokumentasi (pemeliharaan sudah ada PDF, pengecekan lapangan belum).
- 6.4 mengurangi kebutuhan admin login harian — informasi datang ke mereka.

### Verifikasi Fase 6 (per task)

- `cd backend && go build ./... && go test ./... -count=1 && go vet ./...` — PASS.
- `cd frontend && npm run lint && npm run build` — PASS.
- `swag init` setelah anotasi endpoint baru.
- Uji manual: 6.1 badge warna skor di tabel CCTV & peta; 6.2 filter, tabel ranking, dan chart kinerja teknisi tampil untuk admin; 6.3 PDF pengecekan sesuai kop surat dan memuat checklist/foto; 6.4 ringkasan tampil di timeline admin, paginasi berjalan, dan notifikasi tersedia setiap pagi.

---

## Fase 7 — Kesiapan Produksi

> Memastikan sistem siap untuk **operasi nyata (go-live)**. Menyerap sisa Fase 4 (4.3–4.7) dengan
> scope yang disesuaikan, ditambah pengerasan baru yang kritis untuk deployment produksi.
> **Status: ✅ SELESAI** — seluruh Task 7.1–7.6 selesai (2026-08-06/07).
> Fase 7 bisa dikerjakan **paralel** dengan Fase 6 (infrastruktur vs fitur).

### Task (berurutan — kerjakan dari atas ke bawah)

| # | Task | Keterangan | Skala | Asal | Status |
|---|------|-----------|-------|------|--------|
| 7.1 | Migrasi DB Versioned | Adopsi `golang-migrate` v4 (driver pgx v5); skema dikelola file SQL terurut di `backend/migrations/` (000001 init schema: 10 tabel + index + FK, idempoten terhadap DB eksisting); `config.RunMigrations` otomatis saat startup; CLI `go run cmd/migrate/main.go up/down/version`; AutoMigrate dikomentari (fase transisi, akan dihapus setelah stabil); verifikasi `up→down→up` di CI (service PostgreSQL); prosedur di `docs/MIGRATION.md`. Branch: `feature/versioned-migration` | Besar | Fase 4.3 | ✅ Selesai (2026-08-06) — build/test/vet PASS; uji live PostgreSQL 18 lokal up/down/up + skenario DB warisan AutoMigrate PASS |
| 7.2 | Redis untuk Rate-Limit & Blacklist | Container Redis 7 Alpine di `docker-compose.yml` (volume `redis_data`, health check adaptif, port tidak dipublish); `config.ConnectRedis()` dari env `REDIS_HOST`/`REDIS_PORT`/`REDIS_PASSWORD` (default `redis:6379`); blacklist via interface `TokenBlacklist` → Redis SET (`SADD`/`SISMEMBER`/`SREM`) + TTL sesuai sisa masa hidup token (persisten lintas restart, tanpa cleanup goroutine); rate limiter fixed-window kustom memakai Redis `INCR`/`EXPIRE` key `limiter:{ip}:{endpoint}`; graceful fallback in-memory saat Redis tidak tersedia (LOG WARNING, bukan fail-fast); unit test kontrak interface (mock `redisCommander` + in-memory) & limiter (key format, window reset, isolasi key). Branch: `feature/redis-integration` | Sedang | Fase 4.4 | ✅ Selesai (2026-08-06) — build/test/vet PASS; uji live Redis 7: 12 cek PASS termasuk blacklist persisten setelah restart (client baru); `docker compose config` valid |
| 7.3 | Backup Otomatis & Restore | Skrip `scripts/backup.sh`: `pg_dump` database + tar uploads → simpan dengan rotasi (7 hari); skrip `scripts/restore.sh` untuk pemulihan. Cron job di container atau host. Dokumentasi prosedur backup & disaster recovery di `docs/BACKUP.md`. | Sedang | Fase 4.5 | ✅ Selesai (2026-08-07, branch `feature/backup-restore`) — `backup.sh` (pg_dump→gzip + tar uploads + rotasi 7 hari), `restore.sh` (drop/recreate DB + uploads), container `backup` (cron dcron, `postgres:15-alpine`, volume `backups_data`), opsi cron host di `docs/BACKUP.md`, `docker compose config` valid & `bash -n` lulus |
| 7.4 | Monitoring & Alerting | Endpoint `/api/metrics` (Prometheus format): request count, latency histogram, error rate, DB pool stats, WebSocket connections, upload storage usage. `docker-compose.yml` tambah Prometheus + Grafana (opsional). Health check diperluas: cek Redis, disk space upload. | Sedang | Fase 4.6 | ✅ Selesai (2026-08-07, branch `feature/monitoring`) — collector Prometheus tanpa library pihak ketiga (`monitoring`): counter `sigap_http_requests_total{method,path,status}`, histogram `sigap_http_request_duration_seconds`, gauge `sigap_db_pool_open/idle_connections`, `sigap_ws_active_connections`, `sigap_upload_storage_bytes`; middleware `metrics.go`; `GET /api/metrics` publik text/plain; health diperluas (response time DB, Redis, ukuran uploads); `monitoring/prometheus` + `docker-compose.override.yml` (Prometheus & Grafana opsional); build/test/vet + `docker compose config` valid |
| 7.5 | Container Hardening | Dockerfile non-root user; `docker-compose.yml` resource limits (CPU/memory); read-only rootfs; drop capabilities; secret management via Docker secrets (bukan env file). | Kecil | Fase 4.7 | ✅ Selesai (2026-08-07, branch `feature/production-hardening`) — `backend/Dockerfile` non-root `USER sigap` + kepemilikan dir uploads/logs; `api` `read_only: true` + tmpfs `/tmp` & `/app/logs`, `cap_drop: [ALL]`, resource limits (`cpus`/`mem_limit`), Docker secrets `DB_PASSWORD`/`JWT_SECRET`/`AES_SECRET_KEY` di `/run/secrets`; Go membaca secret via file dengan fallback env (`utils/secrets.go`); `docker compose config` valid & build/test/vet PASS |
| 7.6 | Logging Terstruktur & Rotasi | Zap logger memakai `lumberjack` untuk rotasi file log otomatis (maks 100 MB/file, 30 hari retensi, 5 backup); format JSON untuk kemudahan parsing; log level konfigurasi via env (`LOG_LEVEL`). | Kecil | Baru | ✅ Selesai (2026-08-07, branch `feature/production-hardening`) — `lumberjack.v2 v2.2.1` di `go.mod`; `utils/logger.go` rotasi (100 MB/file, 5 backup, 30 hari, gzip) + level dari `LOG_LEVEL` (default info) diterapkan konsisten di file & console; log JSON; test `parseLogLevel` & `InitLogger`; build/test/vet PASS |

### Alasan Urutan

- 7.1 paling mendesak — tanpa versioned migration, perubahan skema produksi berisiko kehilangan data.
- 7.2 memperbaiki kelemahan arsitektur (in-memory blacklist hilang saat restart, rate-limit tidak berlaku multi-instance).
- 7.3 wajib sebelum data nyata masuk — disaster recovery harus tersedia di hari pertama go-live.
- 7.4 agar masalah terdeteksi sebelum user lapor (proaktif, bukan reaktif).
- 7.5 & 7.6 pengerasan akhir dan observabilitas sebelum go-live.

### Verifikasi Fase 7

- Semua test & build backend/frontend lulus.
- `docker compose up -d --build` berjalan lancar dengan Redis dan konfigurasi baru.
- Skrip backup berjalan (`scripts/backup.sh`) dan restore berhasil di environment bersih
  (`scripts/restore.sh`); verifikasi end-to-end di `docs/BACKUP.md` bagian 7.
- Endpoint `/api/metrics` mengembalikan data Prometheus valid (format text exposition, `text/plain; version=0.0.4`), terverifikasi unit test di `backend/monitoring/metrics_test.go` & `backend/middleware/metrics_test.go`.
- Container berjalan sebagai non-root (`USER sigap`); rootfs read-only dengan tmpfs; secret tidak terexpose di `docker inspect` (dibaca dari `/run/secrets/...` via `utils/secrets.go`).
- Log file ter-rotate (lumberjack 100 MB/file, 5 backup, 30 hari) dan terbaca di format JSON; level via `LOG_LEVEL`.

---

## Fase 8 — Pengalaman Lapangan

> Meningkatkan **produktivitas teknisi** yang bekerja di lapangan kawasan IKN dengan perangkat mobile.
> **Status: ✅ SELESAI** — seluruh Task 8.1–8.4 selesai (2026-08-07).

### Task (berurutan — kerjakan dari atas ke bawah)

| # | Task | Keterangan | Skala | Status |
|---|------|-----------|-------|--------|
| 8.1 | Progressive Web App (PWA) | Service worker + web manifest → SIGAP bisa di-install di home screen HP teknisi; cache halaman shell agar loading cepat; badge ikon app. Scope: installability + offline shell (bukan full offline dengan IndexedDB). | Sedang | ✅ Selesai (2026-08-07, branch `feature/pwa`) — `public/manifest.json` (SIGAP, standalone, `#162218`/`#467235`, ikon 192/512), `public/sw.js` (cache-first shell, network-first API, versi cache `sigap-shell-v1`), ikon PNG 192/512 dari logo, registrasi SW + manifest/theme-color/apple-touch-icon di `layout.tsx`, header cache `sw.js`/`manifest.json` di `next.config.ts`; lint & build PASS; manual test `docs/manual-test-pwa.md` |
| 8.2 | Kalender Jadwal Pemeliharaan | Halaman `/kalender` (admin + teknisi): tampilan kalender bulanan (implementasi SVG/CSS mandiri, tanpa dependency chart eksternal) dengan dot/bar per hari menunjukkan jadwal; klik hari → daftar tugas; warna sesuai status; filter per teknisi. Data dari endpoint maintenance-logs + maintenance-schedules (jika 5.4 sudah ada). | Sedang | ✅ Selesai (2026-08-07, branch `feature/calendar-view`) — `src/app/kalender/page.tsx` + `components/CalendarGrid.tsx` (grid 7 kolom, navigasi bulan, dot status, panel detail hari, filter teknisi admin), menu sidebar "Kalender" (`Calendar`), data dari `/maintenance-logs?start_date=&end_date=` dikelompokkan per `scheduled_date`; lint & build PASS; manual test `docs/manual-test-calendar.md` |
| 8.3 | Notifikasi Push Browser | Web Push API (VAPID keys) → notifikasi OS saat tugas baru ditugaskan, eskalasi overdue, atau hasil pengecekan `attention`; opt-in per user; fallback ke in-app jika push ditolak. Endpoint `POST /api/push/subscribe`, `DELETE /api/push/unsubscribe`; model `PushSubscription`. | Sedang | ✅ Selesai (2026-08-07, branch `feature/push-notification`) — model `PushSubscription` + migrasi `000002`; `handlers/push.go` (subscribe/unsubscribe), `routes/push.go`; `utils/push.go` `SendPush` (webpush-go VAPID, async, fallback in-app); integrasi ke `NotifyUser`/`NotifyUserSync`; SW `push`/`notificationclick` + komponen opt-in `PushNotificationOptIn` di `/profile`; build/test/vet + lint/build PASS; manual test `docs/manual-test-push-notification.md` |
| 8.4 | Geotag Pengecekan Lapangan | Saat teknisi mengisi wizard pengecekan, posisi GPS (`navigator.geolocation`) dicatat otomatis; koordinat disimpan di `FieldMaintenance` (`check_latitude`, `check_longitude`); admin bisa melihat lokasi pengecekan di peta (verifikasi: teknisi benar-benar di lokasi CCTV). Persetujuan lokasi bersifat opsional — wizard tetap bisa diselesaikan tanpa GPS. | Kecil | ✅ Selesai (2026-08-07, branch `feature/geotag-field-check`) — kolom `check_latitude`/`check_longitude` + migrasi `000003`; `CreateFieldMaintenance` menerima & memvalidasi koordinat (lat -90..90, lng -180..180, keduanya wajib bersamaan); wizard `FieldMaintenanceModal` menangkap GPS (indikator "Lokasi tercatat", opsional), kirim `check_latitude`/`check_longitude`; mini-map admin `FieldCheckLocationMap` (marker CCTV + Lokasi Pengecekan); build/test/vet + lint/build PASS; manual test `docs/manual-test-geotag.md` |

### Alasan Urutan

- 8.1 paling berdampak luas — semua teknisi langsung merasakan manfaat (akses cepat dari home screen, loading lebih cepat).
- 8.2 melengkapi fitur jadwal berulang (5.4) dengan visualisasi yang lebih intuitif daripada tabel.
- 8.3 memastikan teknisi tidak melewatkan tugas penting (menggantikan keharusan buka app terus-menerus).
- 8.4 menambah akuntabilitas pengecekan lapangan tanpa menghambat alur kerja.

### Ketergantungan

- 8.2 bergantung pada 5.4 (penjadwalan berulang) untuk data jadwal lengkap; bisa diimplementasikan tanpa 5.4 tapi hanya menampilkan tugas manual.
- 8.3 bergantung pada 5.1 (notifikasi in-app) untuk data notifikasi; push sebagai channel tambahan.
- 8.1 dan 8.4 berdiri sendiri.

### Verifikasi Fase 8

- `cd backend && go build ./... && go test ./... -count=1 && go vet ./...` — PASS.
- `cd frontend && npm run lint && npm run build` — PASS.
- PWA: install di Chrome Android/desktop, ikon muncul, halaman shell termuat saat offline (panduan di `docs/manual-test-pwa.md`); Lighthouse PWA score ≥ 80.
- Kalender: tampilan bulan benar, klik hari menampilkan tugas, warna status sesuai (panduan di `docs/manual-test-calendar.md`).
- Push: notifikasi OS muncul saat tugas baru ditugaskan; opt-out berfungsi (panduan di `docs/manual-test-push-notification.md`).
- Geotag: koordinat tercatat di database; admin melihat pin lokasi pengecekan di peta (panduan di `docs/manual-test-geotag.md`).

---

## Keputusan Arsitektur Streaming (Disepakati 2026-08-07)

| # | Pertanyaan | Keputusan |
|---|-----------|----------|
| 1 | Network Access | VPS Hostinger (cloud). Port RTSP `554` pada IP Publik `36.91.27.35` dibuka oleh tim jaringan OIKN. |
| 2 | Bandwidth | **Sub Stream (02)** default (~0.5–1 Mbps/stream). Toggle opsional ke Main Stream (01). |
| 3 | Kredensial NVR | 1 NVR (`36.91.27.35:554`), 35 kamera, credential tunggal. Password AES-256 encrypted di DB. |
| 4 | Encoding | **H.264 wajib**. MediaMTX relay only — **zero transcoding** (tanpa FFmpeg). |
| 5 | Akses Viewer | **Admin Only** — `middleware.RequireRole("admin")`. |
| 6 | Retention | **Live View saja** — rekaman tetap di NVR Hikvision lokal. |
| 7 | Prioritas pasca-Fase 9 | Langsung ke Task 11.1 (Inventaris Suku Cadang). |

---

## Fase 9 — Streaming CCTV Real-Time

> Mengubah peta sebaran CCTV dari **view-only** menjadi **interactive live monitoring center**.
> Teknologi: **MediaMTX** (RTSP-to-WebRTC proxy) + **WHEP** (browser playback) + Nginx reverse proxy.
> **Status: ✅ Selesai** — seluruh 6 task (9.1–9.6) dalam 4 sprint (20–23) **selesai** (2026-08-07).

### Arsitektur Streaming

```
Hikvision NVR          MediaMTX (Docker)       Nginx             Browser
36.91.27.35:554   ◄── RTSP pull on-demand ──  :8889 HTTP  ──►  /stream/*  ──►  WebRTC/WHEP
35 kamera (H.264)      :8554 RTSP internal      :443 TLS         <video> element
CH 1–35, Sub Stream    :9997 REST API           reverse proxy     Admin Only
```

### Logika RTSP Channel ID (Hikvision ISAPI)

```
ChannelID = Channel_Port × 100 + Stream_Type
  Stream_Type: 01 = Main Stream, 02 = Sub Stream (default)

Contoh: CH 19 Sub → 19×100 + 2 = 1902 → /Streaming/channels/1902
        CH 1  Main → 1×100 + 1 = 101  → /Streaming/channels/101
```

### Sprint 20 — Fondasi Streaming (Task 9.1 + 9.2)

> **Branch:** `feature/cctv-streaming-foundation`
> **Status: ✅ Selesai** (2026-08-07)

| Jobdesk | Sub-Agent | Task | Deliverable | Status |
|---------|-----------|------|-------------|--------|
| **Backend** | Agent-BE | **9.1** Model & Migrasi: Data Streaming CCTV | Tambah field `channel_port` (int), `rtsp_password` (text, AES-256 encrypted), `stream_enabled` (boolean) ke model `CCTV`. Migrasi `000004_add_cctv_streaming.up/down.sql`. Util `utils/rtsp.go` (BuildRTSPURL, BuildStreamPath). Konfigurasi NVR global di `.env` (`RTSP_NVR_HOST`, `RTSP_NVR_PORT`, `RTSP_NVR_USER`, `MEDIAMTX_API_URL`). Enkripsi password via `utils.EncryptAES256()`. Update `CreateCCTVRequest`/`UpdateCCTVRequest` di `handlers/cctv.go`. Exclude `rtsp_password` dari response JSON. Unit test `utils/rtsp_test.go`. | ✅ Selesai |
| **Infra** | Agent-Infra | **9.2** Infrastruktur MediaMTX Docker | File `mediamtx.yml` (RTSP→WebRTC relay, `sourceOnDemand: yes`, `protocols: [tcp]`, REST API `:9997`). Service `mediamtx` di `docker-compose.yml` (image `bluenviron/mediamtx:latest-ffmpeg`, IP `172.28.0.11`, resource limits 2 CPU / 1024M, health check, port UDP `8189`). Nginx location block `/stream/*` → `mediamtx:8889`, `/hls/*` → `mediamtx:8888`. Upstream `sigap_mediamtx`. Dependency nginx→mediamtx. `docker compose config --quiet` PASS. | ✅ Selesai |

**✅ Selesai — Ringkasan Sprint 20:** Fondasi streaming CCTV selesai.
- **Backend (Task 9.1, `0.43.0`)**: model `CCTV` bertambah field `channel_port` (int), `rtsp_password` (text, AES-256 encrypted), `stream_enabled` (bool, default false); migrasi `000004_add_cctv_streaming.up/down.sql`; util `utils/rtsp.go` (tipe `StreamType` `MainStream=1`/`SubStream=2`, fungsi `BuildRTSPURL`, `BuildStreamPath`, `GetRTSPPort`) + unit test `utils/rtsp_test.go`; `CreateCCTVRequest`/`UpdateCCTVRequest` diperbarui; `rtsp_password` di-exclude dari response JSON; konfigurasi NVR global di `.env` (`RTSP_NVR_HOST`, `RTSP_NVR_PORT`, `RTSP_NVR_USER`, `MEDIAMTX_API_URL`).
- **Infra (Task 9.2)**: file `mediamtx.yml`; service `mediamtx` di `docker-compose.yml` (image `bluenviron/mediamtx:latest-ffmpeg`, IP `172.28.0.11`, resource limits 2 CPU/1024M, health check, UDP `8189`); `nginx/nginx.conf` upstream `sigap_mediamtx` + location `/stream/*` → `mediamtx:8889` dan `/hls/*` → `mediamtx:8888`.

Detail: `backend/changelog.md` [0.43.0] · `docs/changelog.md` [1.28.28].

#### Verifikasi Sprint 20

```bash
cd backend && go build ./... && go test ./... -count=1 && go vet ./...
cd .. && docker compose config --quiet
```

---

### Sprint 21 — Backend API & MediaMTX Client (Task 9.3)

> **Branch:** `feature/stream-api`
> **Prasyarat:** Sprint 20 selesai (model + infrastruktur tersedia)
> **Status: ✅ Selesai** (2026-08-07)

| Jobdesk | Sub-Agent | Task | Deliverable | Status |
|---------|-----------|------|-------------|--------|
| **Backend** | Agent-BE | **9.3a** MediaMTX REST Client | `utils/mediamtx.go`: struct `MediaMTXClient` dengan method `AddPath`, `RemovePath`, `GetPath`, `ListPaths`. HTTP client ke `http://mediamtx:9997/v3/`. Unit test `utils/mediamtx_test.go` dengan HTTP mock. | ✅ Selesai |
| **Backend** | Agent-BE | **9.3b** Stream Handler & Routes | `handlers/stream.go`: `StartStream` (decrypt password → build RTSP URL → register path di MediaMTX → return WHEP URL), `StopStream`, `StartAllStreams`, `StopAllStreams`, `GetStreamStatus` (gabung data DB + MediaMTX path info), `GetStreamURL`. `routes/stream.go`: registrasi route dengan `Protected()` + `RequireRole("admin")`. Rute statis sebelum dinamis. Registrasi di `main.go`. Goroutine auto-start saat boot. Unit test `handlers/stream_test.go`. Swagger annotations + `swag init`. | ✅ Selesai |

**✅ Selesai — Ringkasan Sprint 21:** Backend streaming CCTV selesai.
- **Task 9.3a (MediaMTX Client)**: `utils/mediamtx.go` — struct `MediaMTXClient` (HTTP client ke `http://mediamtx:9997/v3/`) dengan method `AddPath`, `RemovePath`, `GetPath`, `ListPaths` untuk mengelola path streaming RTSP→WHEP/HLS di MediaMTX; unit test `utils/mediamtx_test.go` memakai HTTP mock (sukses + kasus error/404/response tidak valid).
- **Task 9.3b (Stream Handler & Routes)**: `handlers/stream.go` — `StartStream` (decrypt `rtsp_password` → build URL RTSP → register path di MediaMTX → return WHEP URL), `StopStream`, `StartAllStreams`, `StopAllStreams`, `GetStreamStatus` (gabung data DB + info path MediaMTX), `GetStreamURL` (WHEP + HLS); `routes/stream.go` — grup `/streams` admin-only (`Protected()` + `RequireRole("admin")`): `/status`, `/start-all`, `/stop-all`, `/:id/start`, `/:id/stop`, `/:id/url` (rute statis sebelum dinamis); registrasi di `main.go` + goroutine auto-start stream aktif saat boot; Swagger annotations + `swag init`; unit test `handlers/stream_test.go` (StartStream sukses/404/disabled/no-channel-port, StopStream, GetStreamStatus, GetStreamURL, enforcement admin → 403 untuk teknisi).

Detail: `backend/changelog.md` [0.44.0] · `docs/changelog.md` [1.28.29].

#### Verifikasi Sprint 21

```bash
cd backend && go build ./... && go test ./... -count=1 && go vet ./... && golangci-lint run ./...
swag init
```

---

### Sprint 22 — Frontend: Player & Streaming UI (Task 9.4 + 9.5 + 9.6)

> **Branch:** `feature/stream-frontend`
> **Prasyarat:** Sprint 21 selesai (API stream tersedia)
> **Status: ✅ Selesai** (2026-08-07)

| Jobdesk | Sub-Agent | Task | Deliverable | Status |
|---------|-----------|------|-------------|--------|
| **Frontend** | Agent-FE-1 | **9.4** Komponen WHEPPlayer | `components/WHEPPlayer.tsx`: WebRTC player menggunakan browser-native `RTCPeerConnection` (zero SDK). WHEP signaling via `POST /stream/{path}/whep`. Auto-reconnect exponential backoff. Status overlay (connecting/live/error/reconnecting). Toggle fullscreen. Toggle kualitas (SD sub-stream ↔ HD main-stream). Indikator 🔴 LIVE. Cleanup saat unmount. SSR-safe (dynamic import). | ✅ Selesai |
| **Frontend** | Agent-FE-1 | **9.4b** Komponen StreamModal | `components/StreamModal.tsx`: modal overlay saat marker diklik. Header: nama CCTV, kode, badge status + skor kesehatan. Body: `WHEPPlayer` embed. Footer: lokasi, IP, channel port. Tombol Start/Stop stream (panggil API). Link ke Master CCTV. | ✅ Selesai |
| **Frontend** | Agent-FE-2 | **9.5** Integrasi Peta Sebaran + Grid View | Modifikasi `CCTVMap.tsx`: tambah prop `onMarkerClick`, indikator streaming aktif pada marker. Interface `CCTVMapItem` tambah `stream_enabled`, `channel_port`. Modifikasi `sebaran-cctv/page.tsx`: state `selectedCctv` + `StreamModal`. Toggle view Peta/Grid. Filter "Streaming Aktif". Status bar "N/35 kamera streaming aktif". `components/StreamGridView.tsx`: grid layout responsif, setiap sel berisi `WHEPPlayer` mini. | ✅ Selesai |
| **Frontend** | Agent-FE-2 | **9.6** Admin Config UI Streaming | Modifikasi `cctv/page.tsx`: section form "Konfigurasi Streaming" di modal edit CCTV. Field: `channel_port` (number 1–35), `rtsp_password` (password + toggle visibility), `stream_enabled` (checkbox). Preview URI RTSP. Kolom tabel master: ikon streaming aktif. | ✅ Selesai |

**✅ Selesai — Ringkasan Sprint 22:** UI streaming CCTV real-time selesai (Task 9.4 + 9.5 + 9.6).
- **Task 9.4 / 9.4b (Player & Modal)**: `WHEPPlayer.tsx` (WebRTC player browser-native `RTCPeerConnection`, WHEP signaling, auto-reconnect exponential backoff, status overlay, toggle SD/HD, indikator 🔴 LIVE, SSR-safe dynamic import) + `StreamModal.tsx` (modal overlay saat marker diklik; header nama/kode/badge status + skor kesehatan; body `WHEPPlayer`; footer lokasi/IP/channel port; tombol Start/Stop; link Master CCTV).
- **Task 9.5 (Peta + Grid)**: `CCTVMap.tsx` prop `onMarkerClick` + indikator streaming aktif pada marker; `sebaran-cctv/page.tsx` state `selectedCctv` + `StreamModal`, toggle view Peta/Grid, filter "Streaming Aktif", status bar "N/35 kamera streaming aktif"; `StreamGridView.tsx` (grid responsif, tiap sel berisi `WHEPPlayer` mini).
- **Task 9.6 (Admin Config)**: `cctv/page.tsx` section "Konfigurasi Streaming" di modal edit — `channel_port` (1–35), `rtsp_password` (toggle visibility), `stream_enabled` (checkbox), preview URI RTSP; kolom tabel master ikon streaming aktif.

Detail: `frontend/changelog.md` [1.36.0].

#### Verifikasi Sprint 22

```bash
cd frontend && npm run lint && npm run build
```

---

### Sprint 23 — Integrasi End-to-End & Manual Test Streaming

> **Branch:** sama dengan Sprint 22 (merge semua ke satu PR)
> **Prasyarat:** Sprint 20–22 selesai
> **Status: ✅ Selesai** (2026-08-07)

| Jobdesk | Sub-Agent | Task | Deliverable | Status |
|---------|-----------|------|-------------|--------|
| **QA/Docs** | Agent-QA | Manual Test Document | `docs/manual-test-streaming.md`: header, prasyarat, pengujian otomatis, skenario uji S1–S9 (edit config streaming, start stream, klik marker → video live, toggle SD/HD, stop stream, grid view, role enforcement 403, reconnect saat NVR down, start-all). Kriteria lulus. | ✅ Selesai |
| **Infra** | Agent-Infra | Docker E2E Verification | `docker compose up -d --build` → verifikasi 8 service running. Health check semua PASS. MediaMTX API reachable. | ✅ Selesai |
| **Docs** | Agent-Docs | Update Dokumentasi | Update `docs/AGENTS.md` (MediaMTX service, streaming endpoints, env vars). Update `docs/TESTING.md` (test matrix stream/rtsp/mediamtx). Update `docs/plan.md` (Sprint 20–23). | ✅ Selesai |

**✅ Selesai — Ringkasan Sprint 23:** Integrasi end-to-end & dokumentasi streaming selesai.
- **Manual Test**: `docs/manual-test-streaming.md` — dokumen uji manual blackbox S1–S9 sesuai format manual test lain.
- **Docker E2E**: `docker compose up -d --build` → verifikasi 8 service (db, redis, api, frontend, nginx, backup, mediamtx, + health) running; health check semua PASS; MediaMTX API reachable.
- **Dokumentasi**: `docs/AGENTS.md` (MediaMTX service, streaming endpoints, env vars, file baru), `docs/TESTING.md` (matriks test stream/rtsp/mediamtx), `docs/plan.md` (Sprint 20–23), `docs/changelog.md` [1.28.30].

#### Verifikasi Sprint 23 (Full)

```bash
cd backend && go build ./... && go test ./... -count=1 && go vet ./... && golangci-lint run ./... && swag init
cd ../frontend && npm run lint && npm run build
cd .. && docker compose config --quiet
```

---

## Fase 10 — Kecerdasan Visual & Snapshot

> Memperkaya data visual CCTV dengan snapshot otomatis dan monitoring stream.
> **Prasyarat:** Fase 9 selesai. **Status: ✅ Selesai** — 4 task, 2 sprint (Sprint 24 = Task 10.1 + 10.2 ✅ Selesai; Sprint 25 = Task 10.3 + 10.4 ✅ Selesai).

### Sprint 24 — Snapshot & Timeline (Task 10.1 + 10.2)

> **Branch:** `feature/cctv-snapshot`
> **Status: ✅ Selesai** (2026-08-07)

| Jobdesk | Sub-Agent | Task | Deliverable | Status |
|---------|-----------|------|-------------|--------|
| **Backend** | Agent-BE | **10.1** Snapshot Otomatis (Thumbnail) | Goroutine berkala (tiap 15 menit) mengambil snapshot frame dari stream aktif via MediaMTX API → simpan JPEG di `uploads/snapshots/{cctv_id}/` → endpoint `GET /api/cctv/:id/snapshot`. Rotasi max 10 snapshot per CCTV. Field `snapshot_url` di response `GetAllCCTV`. | ✅ Selesai |
| **Backend** | Agent-BE | **10.2** Timeline Event Visual | Model `CCTVSnapshot` (cctv_id, timestamp, event_type, image_path). Capture otomatis saat event penting (status berubah, maintenance selesai, field-check `attention`). Endpoint `GET /api/cctv/:id/timeline`. Migrasi `000005_cctv_snapshots`. | ✅ Selesai |
| **Frontend** | Agent-FE | **10.1b + 10.2b** UI Snapshot & Timeline | Thumbnail di kartu CCTV tabel master & peta marker tooltip. Timeline kronologis di detail CCTV dengan snapshot + label event. | ✅ Selesai |

**✅ Selesai — Ringkasan Sprint 24:** Snapshot & Timeline (Fase 10) selesai.
- **Backend (Task 10.1 + 10.2, `0.46.0`)**: model `CCTVSnapshot` + migrasi `000005_cctv_snapshots`; goroutine snapshot berkala (15 menit) via `utils/mediamtx.go` `GetSnapshot` (endpoint `/v3/paths/get/{path}/snapshot`); endpoint `GET /api/cctv/:id/snapshot` (JPEG, 404 bila belum ada) & `GET /api/cctv/:id/timeline` (paginasi); rotasi maksimal 10 snapshot per CCTV di `uploads/snapshots/{cctv_id}/`; event hook `status_changed` / `maintenance_completed` / `field_check_attention` / `periodic_snapshot`; field computed `snapshot_url` (`gorm:"-"`) pada respons `GetAllCCTV`/`GetCCTVByID`.
- **Frontend (Task 10.1b + 10.2b, `1.38.0`)**: thumbnail snapshot di tabel Master CCTV (`/cctv`, `AuthenticatedImage` + placeholder `ImageOff`); thumbnail di tooltip marker peta (`/sebaran-cctv`, placeholder `Camera`); timeline kronologis di detail CCTV dengan snapshot + label event (`EVENT_TYPE_LABELS`).
- **Dokumen**: `docs/manual-test-visual.md` (baru — manual test blackbox S1–S9 sesuai format manual test lain); `docs/TESTING.md` (matriks snapshot: `handlers/snapshot_test.go` — `GetCCTVTimeline` 404/success, `GetCCTVSnapshot` 404; total 48 → 49).

Detail: `backend/changelog.md` [0.46.0] · `frontend/changelog.md` [1.38.0] · `docs/changelog.md` [1.28.31].

#### Verifikasi Sprint 24

```bash
cd backend && go build ./... && go test ./... -count=1 && go vet ./... && golangci-lint run ./... && swag init
cd ../frontend && npm run lint && npm run build
```

### Sprint 25 — Multi-View Monitor & Stream Health (Task 10.3 + 10.4)

> **Branch:** `feature/monitor-view`
> **Status: ✅ Selesai** (2026-08-07)

| Jobdesk | Sub-Agent | Task | Deliverable | Status |
|---------|-----------|------|-------------|--------|
| **Frontend** | Agent-FE | **10.3** Multi-View Monitor | Halaman `/monitor` (admin only): grid 4/9/16 video stream bersamaan. Preset layout (2×2, 3×3, 4×4). Pilih CCTV per sel. Fullscreen per sel atau seluruh grid. Menu sidebar "Monitor" (`MonitorPlay`). | ✅ Selesai |
| **Backend** | Agent-BE | **10.4** Deteksi Status Stream | Goroutine monitoring (tiap 5 menit): query MediaMTX path list → cek `ready`. Stream mati > 15 menit → notifikasi admin `STREAM_DOWN` + turunkan health score (-10). Stream pulih → `STREAM_RESTORED`. Kolom "Stream" di tabel master CCTV. | ✅ Selesai |
| **QA/Docs** | Agent-QA | Manual Test | `docs/manual-test-visual.md`: skenario snapshot, timeline, monitor grid, stream health detection. | ✅ Selesai |

**✅ Selesai — Ringkasan Sprint 25:** Multi-View Monitor & Stream Health (Fase 10) selesai — Fase 10 kini **4/4 ✅ Selesai**.
- **Frontend (Task 10.3, `1.39.0`)**: halaman `/monitor` (admin-only) grid multi-stream dengan preset layout 2×2/3×3/4×4; per-cell CCTV dropdown + `WHEPPlayer`; fullscreen per sel & seluruh grid; pop-out `window.open('/monitor/pop?path=...')`; halaman `/monitor/pop` (single fullscreen stream); menu sidebar "Monitor CCTV" (`MonitorPlay`) setelah "Peta Sebaran CCTV".
- **Backend (Task 10.4, `0.47.0`)**: goroutine monitoring (tiap 5 menit) query MediaMTX `ListPaths()` — track per-path ready state; stream down > 15 menit → broadcast `STREAM_DOWN` + kurangi health_score -10; pulih → `STREAM_RESTORED`; endpoint `GET /api/streams/health` (admin-only) list stream + status ready/down + `down_duration`; unit test `handlers/stream_health_test.go`.
- **Dokumen**: `docs/manual-test-visual.md` (tambah Sprint 25 S10–S16); `docs/TESTING.md` (matriks `handlers/stream_health_test.go`; total 49 → 50).

Detail: `backend/changelog.md` [0.47.0] · `frontend/changelog.md` [1.39.0] · `docs/changelog.md` [1.28.32].

#### Verifikasi Sprint 25

```bash
cd backend && go build ./... && go test ./... -count=1 && go vet ./... && golangci-lint run ./... && swag init
cd ../frontend && npm run lint && npm run build
```

---

## Fase 11 — Manajemen Operasional Lanjut

> Fitur operasional yang memperkuat fungsi SIGAP sebagai platform pengelolaan lengkap.
> **Status: ✅ Selesai** — 5/5 task selesai (Sprint 26 ✅, Sprint 27 ✅, Sprint 28 ✅).
> **Task 11.1 dikerjakan langsung setelah Fase 9 stabil** (keputusan prioritas).

### Sprint 26 — Inventaris Suku Cadang (Task 11.1) ⭐ PRIORITAS

> **Branch:** `feature/spare-parts`
> **Status: ✅ Selesai** (2026-08-09)
> **Dikerjakan segera setelah Fase 9 stabil** — tidak menunggu Fase 10.

| Jobdesk | Sub-Agent | Task | Deliverable | Status |
|---------|-----------|------|-------------|--------|
| **Backend** | Agent-BE | **11.1a** Model & API Suku Cadang | Model `SparePart` (nama, SKU, stok, minimum_order, satuan, harga, kategori) + `SparePartUsage` (maintenance_log_id, spare_part_id, jumlah, catatan). Stok otomatis berkurang saat teknisi mencatat penggunaan. Notifikasi `STOCK_LOW` ke admin saat stok < minimum_order. CRUD endpoint `/api/spare-parts`. Endpoint `POST /api/maintenance-logs/:id/spare-parts`. Migrasi `000006_spare_parts`. | ✅ Selesai |
| **Frontend** | Agent-FE | **11.1b** UI Suku Cadang | Halaman admin `/suku-cadang` (CRUD tabel stok, filter kategori, indikator stok rendah). Form penggunaan suku cadang di modal pemeliharaan. Dashboard widget "Suku Cadang Stok Rendah". Menu sidebar "Suku Cadang" (`Package`). | ✅ Selesai |
| **QA/Docs** | Agent-QA | Manual Test | `docs/manual-test-spare-parts.md`. | ✅ Selesai |

**✅ Selesai — Ringkasan Sprint 26:** Inventaris suku cadang (Task 11.1) selesai.
- **Backend (`0.48.0`)**: model `SparePart` + `SparePartUsage`; migrasi `000006_spare_parts` (up/down); handler `spare_part.go` (CRUD `/api/spare-parts`, `GET /api/spare-parts/low-stock`, `POST/GET /api/maintenance-logs/:id/spare-parts`); stok dikurangi otomatis dalam transaksi hanya untuk log `completed`; broadcast `STOCK_LOW` ke admin saat stok < minimum_order; route di `routes/spare_part.go` + registrasi di `main.go`; Swagger annotations + `swag init`; unit test `handlers/spare_part_test.go` (CRUD, stok berkurang, stok tidak cukup 400, log bukan completed 409, low-stock).
- **Frontend (`1.40.0`)**: halaman admin `/suku-cadang` (CRUD tabel, filter kategori, pencarian, badge stok, modal form, hapus konfirmasi, paginasi); integrasi "Suku Cadang Digunakan" + mini modal "Catat Penggunaan" di modal detail pemeliharaan; dashboard widget "Suku Cadang Stok Rendah" (admin only, real-time via `STOCK_LOW`); menu sidebar "Suku Cadang" (`Package`); proxy melindungi `/suku-cadang`.
- **Dokumen**: `docs/manual-test-spare-parts.md` (baru — S1–S9 sesuai format manual test lain); `docs/TESTING.md` (matriks `handlers/spare_part_test.go`; total 50 → 51).

Detail: `backend/changelog.md` [0.48.0] · `frontend/changelog.md` [1.40.0] · `docs/changelog.md` [1.28.33].

### Sprint 27 — Email & Report Builder (Task 11.2 + 11.3)

> **Branch:** `feature/email-reports`
> **Status: ✅ Selesai** (2026-08-09)

| Jobdesk | Sub-Agent | Task | Deliverable | Status |
|---------|-----------|------|-------------|--------|
| **Backend** | Agent-BE | **11.3** Notifikasi Email SMTP | Konfigurasi SMTP di `.env`. Template email HTML (Go `html/template`). Jenis: ringkasan harian, eskalasi overdue, stock rendah. Field `Email` + `EmailNotifications` di model `User`. Endpoint `PUT /api/profile/email-preferences`. Opt-in per user. | ✅ Selesai |
| **Backend** | Agent-BE | **11.2** Laporan Kustom (Report Builder) | Model `ReportTemplate` (nama, konfigurasi JSON). Endpoint `POST /api/reports/generate` (entitas, kolom, filter, format PDF/CSV/Excel). Template yang bisa disimpan/dipakai ulang. | ✅ Selesai |
| **Frontend** | Agent-FE | **11.2b + 11.3b** UI Email & Report | Halaman `/laporan-kustom` (report builder wizard). Email preferences di halaman profil. | ✅ Selesai |

**✅ Selesai — Ringkasan Sprint 27:** Email & Report Builder (Task 11.2 + 11.3) selesai.
- **Backend (`0.49.0`)**: `utils/email.go` (SendEmail via net/smtp + TLS, render template html/template, `InitEmailConfig`/`EmailEnabled`); template `daily_summary.html`/`overdue_escalation.html`/`stock_low.html`; model `User.EmailNotifications` + migrasi `000007`; `PUT /api/profile/email-preferences`; trigger email ringkasan harian (digest 06:00 WITA), overdue (overdue checker), stock low (`STOCK_LOW`). Report builder: model `ReportTemplate` + migrasi `000008`; handler `report_builder.go` (`POST /api/reports/generate` CSV/XLSX/PDF untuk 4 entitas, `GET/POST /api/reports/templates`, `DELETE /api/reports/templates/:id`); routes `routes/profile.go` + `routes/report.go` (admin-only) + registrasi di `main.go`; Swagger + unit test `handlers/email_test.go`, `handlers/report_builder_test.go`, `utils/email_test.go`.
- **Frontend (`1.41.0`)**: halaman `/laporan-kustom` (wizard 4 langkah: entitas, kolom, filter, format; unduh CSV/Excel/PDF; simpan/muat/hapus template); email preferences di `/profile` (toggle `email_notifications`); menu sidebar "Laporan" (`FileSpreadsheet`); proxy melindungi `/laporan-kustom`.
- **Dokumen**: `docs/manual-test-email.md` (baru — S1–S7), `docs/manual-test-report-builder.md` (baru — S1–S8); `docs/TESTING.md` (matriks `email_test.go`, `report_builder_test.go`, `utils/email_test.go`; total 51 → 54).

Detail: `backend/changelog.md` [0.49.0] · `frontend/changelog.md` [1.41.0] · `docs/changelog.md` [1.28.35].

### Sprint 28 — Zona & SLA (Task 11.4 + 11.5)

> **Branch:** `feature/zone-sla`
> **Status: ✅ Selesai** (2026-08-09)

| Jobdesk | Sub-Agent | Task | Deliverable | Status |
|---------|-----------|------|-------------|--------|
| **Backend** | Agent-BE | **11.4** Zona & Wilayah Teknisi | Model `Zone` (nama, deskripsi). Relasi CCTV → Zone (FK `zone_id`). Relasi User ↔ Zone (many-to-many). Auto-assign berdasarkan zona. CRUD `/api/zones`. | ✅ Selesai |
| **Backend** | Agent-BE | **11.5** SLA & KPI Tracking | Model `SLAPolicy` (nama, jenis_tugas, target_jam). Kalkulasi SLA achievement rate. Notifikasi saat SLA mendekati batas (75%). Endpoint `GET /api/dashboard/sla`. | ✅ Selesai |
| **Frontend** | Agent-FE | **11.4b + 11.5b** UI Zona & SLA | Halaman `/zona` (CRUD zona + assign CCTV/teknisi). Filter peta per zona. Dashboard `/sla` (chart compliance, tabel pelanggaran). | ✅ Selesai |
| **QA/Docs** | Agent-QA | Manual Test | `docs/manual-test-zone-sla.md`. | ✅ Selesai |

**✅ Selesai — Ringkasan Sprint 28:** Zona & SLA (Task 11.4 + 11.5) selesai — **Fase 11 kini 5/5 ✅ Selesai**.
- **Backend (`0.50.0`)**: model `Zone`/`UserZone` (`models/zone.go`) + FK `zone_id` di `CCTV`; model `SLAPolicy` (`models/sla.go`) + FK `sla_policy_id` di `MaintenanceSchedule`; migrasi `000009_zone_sla` (up/down); handler `zone.go` (CRUD `/api/zones`, `GET /api/zones/all`, `POST /api/zones/:id/assign-cctvs`, `POST /api/zones/:id/assign-users`); handler `sla.go` (CRUD `/api/sla-policies`, `GET /api/dashboard/sla` admin → `monthly_compliance`/`violations`/`technician_ranking`); goroutine `RunSLADeadlineChecker` (startup + tiap 1 jam) memicu `SLA_WARNING` ke admin saat tugas melewati 75% target; `GetAllCCTV` mendukung filter `?zone_id=`; unit test `zone_test.go`, `sla_test.go`, rute baru di `routes_test.go`; Swagger + `swag init`.
- **Frontend (`1.42.0`)**: halaman admin `/zona` (CRUD zona + modal "Atur Penugasan" multi-select CCTV & teknisi); dashboard admin `/sla` (kartu ringkasan, chart bar kepatuhan bulanan via `recharts`, ranking teknisi, tabel pelanggaran SLA); filter peta per zona di `/sebaran-cctv`; menu sidebar "Zona" (`MapPinned`) & "SLA" (`Target`); proxy melindungi `/zona` & `/sla`.
- **Dokumen**: `docs/manual-test-zone-sla.md` (baru — S1–S9 sesuai format manual test lain).

Detail: `backend/changelog.md` [0.50.0] · `frontend/changelog.md` [1.42.0].

---

## Fase 12 — Integrasi & Skalabilitas

> Mempersiapkan SIGAP untuk integrasi dengan sistem lain.
> Fokus saat ini: **Kawasan OIKN**.
> **Status: 🔶 Berjalan** — 3/4 task selesai (Sprint 29 ✅, Task 12.4 ✅, Task 12.2 belum dimulai).

### Sprint 29 — API Key & Webhook (Task 12.1 + 12.3)

> **Branch:** `feature/api-integration`
> **Status: ✅ Selesai** (2026-08-09)

| Jobdesk | Sub-Agent | Task | Deliverable | Status |
|---------|-----------|------|-------------|--------|
| **Backend** | Agent-BE | **12.1** API Key untuk Integrasi | Model `APIKey` (key_hash, scopes, expires_at). Middleware `APIKeyAuth`. Rate limit per key. Endpoint CRUD `/api/api-keys` (admin). | ✅ Selesai |
| **Backend** | Agent-BE | **12.3** Webhook & Event System | Model `WebhookEndpoint` (url, events, secret). Events: `cctv.status_changed`, `maintenance.completed`, `stream.down`. Async delivery + retry 3x. Model `WebhookDelivery` (log). | ✅ Selesai |
| **Frontend** | Agent-FE | **12.1b + 12.3b** UI API Key & Webhook | Halaman `/api-keys` (generate, revoke, usage). Halaman `/webhooks` (CRUD, test delivery, log). | ✅ Selesai |

**✅ Selesai — Ringkasan Sprint 29:** API Key & Webhook (Task 12.1 + 12.3) selesai.
- **Backend (`0.51.0`)**: model `APIKey` (`models/api_key.go`) + `WebhookEndpoint`/`WebhookDelivery` (`models/webhook.go`); migrasi `000010_api_keys_webhooks` (up/down); middleware `middleware/apikey.go` (`APIKeyAuth`/`APIKeyOrProtected`, rate limit 100 req/menit per key via Redis, cek revoked/expired/scope, update `last_used_at`); handler `handlers/api_key.go` (`POST /api/api-keys`, `GET /api/api-keys`, `POST /api/api-keys/:id/revoke` admin-only; key mentah tampil 1×); handler `handlers/webhook.go` (CRUD `/api/webhooks`, `POST /api/webhooks/:id/test`, `GET /api/webhooks/:id/deliveries`; antrean channel + worker `StartWebhookWorker`; `X-Signature-256` HMAC-SHA256; retry 3× 1s/5s/25s); hook event di `cctv_sync.go`/`maintenance.go`/`overdue_checker.go`/`health_score.go`/`stream.go`; `GET /api/cctv` dapat diakses via API key scope `cctv:read`; unit test `api_key_test.go`, `webhook_test.go`, `middleware/apikey_test.go`; Swagger + `swag init`.
- **Frontend (`1.43.0`)**: halaman admin `/api-keys` (generate key tampil 1× + salin, list, revoke, badge scope, usage); halaman admin `/webhooks` (CRUD endpoint, pilih event, test delivery, modal detail + log deliveries); menu sidebar "API Keys" (`KeyRound`) & "Webhooks" (`Webhook`); proxy melindungi `/api-keys` & `/webhooks`.
- **Dokumen**: `docs/manual-test-api-webhook.md` (baru — S1–S9 sesuai format manual test lain).

Detail: `backend/changelog.md` [0.51.0] · `frontend/changelog.md` [1.43.0].

### Sprint 30 — Multi-Site (Task 12.2)

> **Branch:** `feature/multi-site`

| Jobdesk | Sub-Agent | Task | Deliverable |
|---------|-----------|------|-------------|
| **Backend** | Agent-BE | **12.2a** Model & Migrasi Multi-Site | Model `Site` (nama, kode, alamat, koordinat_pusat). FK `site_id` di CCTV, MaintenanceLog, dll. Migrasi: semua data existing → Site "OIKN" (default). CRUD `/api/sites`. |
| **Frontend** | Agent-FE | **12.2b** UI Multi-Site | Switch site di navigasi (dropdown header). Dashboard per site. Filter global per site. User assignment ke multiple sites. |
| **QA/Docs** | Agent-QA | Manual Test | `docs/manual-test-multi-site.md`. |

### Task 12.4 — Mobile App (React Native / Expo)

> **Branch:** `feature/mobile-app`
> **Status: ✅ Selesai** (2026-08-09)
> Aplikasi mobile untuk **teknisi di lapangan**: pengecekan CCTV, scan QR label fisik, dan pencatatan pemeliharaan tanpa laptop. Target OS **Android** (prioritas).

| Jobdesk | Sub-Agent | Task | Deliverable | Status |
|---------|-----------|------|-------------|--------|
| **Mobile-Core** | Agent-Mobile | **12.4a** Setup & Auth | Proyek Expo RN TypeScript di `mobile/`; `react-navigation` (Stack + Bottom Tabs); shared API client (axios, Bearer session); login + simpan token via `expo-secure-store`; auto-login & refresh; profil teknisi (ganti password, status koneksi, logout). | ✅ Selesai |
| **Mobile-Field** | Agent-Mobile | **12.4b** Scan QR & Checkup Wizard | Scanner QR kamera (`expo-camera`) → `POST /api/cctv/scan` → detail CCTV; wizard checklist 9 tahap + GPS (`expo-location`) + foto bukti (`expo-image-picker`) → unggah + `POST /api/field-maintenance`; draft offline ke AsyncStorage + sync. | ✅ Selesai |
| **Mobile-Stream** | Agent-Mobile | **12.4c** Live View & Maps | Peta GIS (`react-native-maps`, marker warna status); WHEP player via WebView fallback; list tugas `scheduled`/`ongoing` + petunjuk arah. | ✅ Selesai |

**✅ Selesai — Ringkasan Task 12.4:** Aplikasi mobile SIGAP selesai.
- **Mobile (`1.0.0`)**: proyek Expo RN TypeScript di `mobile/`; `react-navigation` (Native Stack + Bottom Tabs); shared axios client (`src/api/client.ts`) mendukung `Authorization: Bearer <token>` (token sesi ditangkap dari `Set-Cookie` saat login dan disimpan di `expo-secure-store`); AuthContext (login, auto-login, refresh via `/auth/me`, logout); layar Login, Profil (ganti password via `/auth/change-password`, status koneksi `/health`, logout); Scan QR (`expo-camera` + `POST /api/cctv/scan`); Checkup Wizard 9 tahap (`expo-location` GPS + `expo-image-picker` foto + `POST /api/upload` + `POST /api/field-maintenance` + draft AsyncStorage offline); Peta GIS (`react-native-maps`); Live View (WebView WHEP fallback); List Tugas (`/dashboard/tasks`).
- **Verifikasi**: `npx tsc --noEmit` lulus (0 error).
- **Dokumen**: `mobile/README.md` (baru), `mobile/.env.example` (baru).

Detail: `docs/changelog.md` [1.28.38].

---

## Pemetaan 35 Kamera (Referensi Konfigurasi Streaming)

Semua kamera: `rtsp://admin:***@36.91.27.35:554/Streaming/channels/{ChannelID}`

| # | Channel Port | Sub Stream ID | Main Stream ID |
|---|-------------|---------------|----------------|
| 1 | 1 | 102 | 101 |
| 2 | 2 | 202 | 201 |
| 3 | 3 | 302 | 301 |
| 4 | 4 | 402 | 401 |
| 5 | 5 | 502 | 501 |
| 6 | 6 | 602 | 601 |
| 7 | 7 | 702 | 701 |
| 8 | 8 | 802 | 801 |
| 9 | 9 | 902 | 901 |
| 10 | 10 | 1002 | 1001 |
| 11 | 11 | 1102 | 1101 |
| 12 | 12 | 1202 | 1201 |
| 13 | 13 | 1302 | 1301 |
| 14 | 14 | 1402 | 1401 |
| 15 | 15 | 1502 | 1501 |
| 16 | 16 | 1602 | 1601 |
| 17 | 17 | 1702 | 1701 |
| 18 | 18 | 1802 | 1801 |
| 19 | 19 | 1902 | 1901 |
| 20 | 20 | 2002 | 2001 |
| 21 | 21 | 2102 | 2101 |
| 22 | 22 | 2202 | 2201 |
| 23 | 23 | 2302 | 2301 |
| 24 | 24 | 2402 | 2401 |
| 25 | 25 | 2502 | 2501 |
| 26 | 26 | 2602 | 2601 |
| 27 | 27 | 2702 | 2701 |
| 28 | 28 | 2802 | 2801 |
| 29 | 29 | 2902 | 2901 |
| 30 | 30 | 3002 | 3001 |
| 31 | 31 | 3102 | 3101 |
| 32 | 32 | 3202 | 3201 |
| 33 | 33 | 3302 | 3301 |
| 34 | 34 | 3402 | 3401 |
| 35 | 35 | 3502 | 3501 |

---

## Ringkasan Estimasi

```
═══════════════════════════════════════════════════════════════
SEGERA (Agustus – September 2026)
───────────────────────────────────────────────────────────────
Sprint 20: Fondasi Streaming (9.1 + 9.2) ✅ ......... 2 hari
Sprint 21: Backend API Stream (9.3) ✅ .............. 2–3 hari
Sprint 22: Frontend Player + UI (9.4–9.6) ✅ ....... 3–4 hari
Sprint 23: Integrasi E2E + Manual Test ✅ ........... 1–2 hari
Sprint 24: Snapshot & Timeline (10.1 + 10.2) ✅ .... 3–4 hari
Sprint 26: Inventaris Suku Cadang (11.1) .......... 3–4 hari
───────────────────────────────────────────────────────────────

BERIKUTNYA (Oktober – November 2026)
───────────────────────────────────────────────────────────────
Sprint 25: Monitor & Stream Health (10.3 + 10.4) .. 3–4 hari
Sprint 27: Email & Report Builder (11.2 + 11.3) ... 4–5 hari
Sprint 28: Zona & SLA (11.4 + 11.5) ............... 4–5 hari
───────────────────────────────────────────────────────────────

JANGKA PANJANG (Evaluasi berdasarkan kebutuhan)
───────────────────────────────────────────────────────────────
Sprint 29: API Key & Webhook (12.1 + 12.3) ........ 3–4 hari
Sprint 30: Multi-Site (12.2) ....................... 4–5 hari
Task 12.4: Mobile App (React Native) .............. 10–15 hari
═══════════════════════════════════════════════════════════════
```

---

## Konvensi Pengerjaan

1. `git checkout main && git pull origin main`
2. `git checkout -b feature/…` atau `bugfix/…`
3. Commit dengan Conventional Commits: `feat(scope): …` / `fix(scope): …`
4. Push → buat PR → tunggu merge oleh user → hapus branch lokal
5. Catat perubahan di `../backend/changelog.md` / `../frontend/changelog.md` (format: before/after,
   timestamp WITA, kategori, file, alasan)
6. Perbarui dokumen terdampak: `docs/AGENTS.md`, `docs/ROADMAP.md`, `docs/TESTING.md`, `docs/plan.md`

