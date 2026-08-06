# ROADMAP — SIGAP

> Rencana pengembangan SIGAP — dari perbaikan keamanan hingga kecerdasan operasional.
> Disusun: 2026-08-01 · Diperbarui: 2026-08-06

---

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
| 6 | Kecerdasan Operasional | 1/4 | 🚧 Berjalan |
| 7 | Kesiapan Produksi | 0/6 | ⬜ Belum |
| 8 | Pengalaman Lapangan | 0/4 | ⬜ Belum |

> **Urutan pengerjaan:**
> ```
> Fase 1 ✅ → Fase 2 ✅ → Fase 3 ✅ → Fase 5 (5.2 → 5.5, aktif) → Fase 6 ──┐
>                                                                              ├──→ Fase 8
>                                                                 Fase 7 ────┘
> ```
> - **Fase 5** (alur kerja operasional): dikerjakan berurutan 5.2 → 5.5, fondasi untuk semua fase berikutnya.
> - **Fase 6** (kecerdasan) dan **Fase 7** (produksi) bisa dikerjakan **paralel** — Fase 6 murni fitur, Fase 7 murni infrastruktur. Jika tim terbatas dan go-live dekat, prioritaskan Fase 7.
> - **Fase 8** (pengalaman lapangan) dikerjakan setelah Fase 6 dan/atau 7 tuntas.
> - Sisa Fase 4 (4.3–4.7) kini **dilebur ke Fase 7** dengan scope yang disesuaikan.
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
> **Status: 🚧 BERJALAN** — Task 6.1 selesai; task berikutnya dikerjakan setelah fondasi skor kesehatan tersedia.

### Task (berurutan — kerjakan dari atas ke bawah)

| # | Task | Keterangan | Skala | Status |
|---|------|-----------|-------|--------|
| 6.1 | Skor Kesehatan CCTV | Skor 0–100 per CCTV dihitung otomatis dari 5 faktor: jumlah pemeliharaan gagal (bobot 30%), frekuensi pengecekan `attention` (25%), keterlambatan tugas (20%), umur sejak pemeliharaan terakhir berhasil (15%), jumlah item checklist `not_ok` (10%). Endpoint `GET /api/cctv/:id/health-score` + kolom `health_score` di list CCTV + badge visual (hijau ≥70 / kuning ≥40 / merah <40) di tabel & peta sebaran. Goroutine penghitungan ulang berkala (tiap 6 jam) atau saat event complete/field-check selesai. | Besar | ✅ Selesai (2026-08-06) — test kalkulasi/endpoint + backend build/test/vet dan frontend lint/build PASS |
| 6.2 | Dashboard Kinerja Teknisi | Endpoint `GET /api/dashboard/technician-performance` (admin): completion rate, rata-rata waktu penyelesaian (assign → complete), jumlah tugas overdue, jumlah pengecekan lapangan, kualitas (rasio 9/9 ok vs total). Halaman admin `/kinerja-teknisi` + menu sidebar, tabel ranking + bar chart per teknisi. Filter rentang waktu (7 hari / 30 hari / kustom). | Sedang | ⬜ Belum |
| 6.3 | Laporan PDF Pengecekan Lapangan | Endpoint `GET /api/field-maintenance/:id/report-pdf`: laporan per-record dengan kop surat IKN (pola sama dengan laporan pemeliharaan), detail 9 item checklist (nama, hasil, catatan), foto bukti, identitas teknisi & CCTV, waktu pengecekan. Tombol "Cetak Laporan" di modal detail pengecekan lapangan (admin). | Sedang | ⬜ Belum |
| 6.4 | Ringkasan Operasional Otomatis | Goroutine harian (06:00 WITA): hitung ringkasan 24 jam terakhir — tugas baru/selesai/gagal/overdue, pengecekan lapangan, CCTV berubah status — simpan di model `DailyDigest`. Endpoint `GET /api/dashboard/digest` (admin): daftar ringkasan harian. Halaman `/ringkasan` dengan timeline harian. Notifikasi in-app ke admin saat ringkasan tersedia. | Sedang | ⬜ Belum |

### Alasan Urutan

- 6.1 paling berdampak — skor kesehatan langsung menjawab "CCTV mana yang perlu prioritas?"
- 6.2 membantu evaluasi beban kerja & redistribusi tugas (melengkapi 5.3 re-assign).
- 6.3 melengkapi siklus dokumentasi (pemeliharaan sudah ada PDF, pengecekan lapangan belum).
- 6.4 mengurangi kebutuhan admin login harian — informasi datang ke mereka.

### Verifikasi Fase 6 (per task)

- `cd backend && go build ./... && go test ./... -count=1 && go vet ./...` — PASS.
- `cd frontend && npm run lint && npm run build` — PASS.
- `swag init` setelah anotasi endpoint baru.
- Uji manual: 6.1 badge warna skor di tabel CCTV & peta; 6.2 tabel kinerja teknisi tampil; 6.3 PDF pengecekan sesuai kop surat; 6.4 ringkasan terkirim ke admin setiap pagi.

---

## Fase 7 — Kesiapan Produksi

> Memastikan sistem siap untuk **operasi nyata (go-live)**. Menyerap sisa Fase 4 (4.3–4.7) dengan
> scope yang disesuaikan, ditambah pengerasan baru yang kritis untuk deployment produksi.
> **Status: ⬜ BELUM** — bisa dikerjakan **paralel** dengan Fase 6 (infrastruktur vs fitur).

### Task (berurutan — kerjakan dari atas ke bawah)

| # | Task | Keterangan | Skala | Asal | Status |
|---|------|-----------|-------|------|--------|
| 7.1 | Migrasi DB Versioned | Adopsi `golang-migrate` atau Atlas; konversi `AutoMigrate` → migration files terurut; skrip rollback; integrasi CI. Fase transisi: migration files + `AutoMigrate` berdampingan, lalu `AutoMigrate` dihapus setelah stabil. | Besar | Fase 4.3 | ⬜ Belum |
| 7.2 | Redis untuk Rate-Limit & Blacklist | Tambah container Redis di `docker-compose.yml`; `RateLimiter` dan `TokenBlacklist` memakai Redis; blacklist token tidak hilang saat restart; mendukung multi-instance API. Scope awal: hanya rate-limit + blacklist, belum caching data. | Sedang | Fase 4.4 | ⬜ Belum |
| 7.3 | Backup Otomatis & Restore | Skrip `scripts/backup.sh`: `pg_dump` database + tar uploads → simpan dengan rotasi (7 hari); skrip `scripts/restore.sh` untuk pemulihan. Cron job di container atau host. Dokumentasi prosedur backup & disaster recovery di `docs/BACKUP.md`. | Sedang | Fase 4.5 | ⬜ Belum |
| 7.4 | Monitoring & Alerting | Endpoint `/api/metrics` (Prometheus format): request count, latency histogram, error rate, DB pool stats, WebSocket connections, upload storage usage. `docker-compose.yml` tambah Prometheus + Grafana (opsional). Health check diperluas: cek Redis, disk space upload. | Sedang | Fase 4.6 | ⬜ Belum |
| 7.5 | Container Hardening | Dockerfile non-root user; `docker-compose.yml` resource limits (CPU/memory); read-only rootfs; drop capabilities; secret management via Docker secrets (bukan env file). | Kecil | Fase 4.7 | ⬜ Belum |
| 7.6 | Logging Terstruktur & Rotasi | Zap logger memakai `lumberjack` untuk rotasi file log otomatis (maks 100 MB/file, 30 hari retensi, 5 backup); format JSON untuk kemudahan parsing; log level konfigurasi via env (`LOG_LEVEL`). | Kecil | Baru | ⬜ Belum |

### Alasan Urutan

- 7.1 paling mendesak — tanpa versioned migration, perubahan skema produksi berisiko kehilangan data.
- 7.2 memperbaiki kelemahan arsitektur (in-memory blacklist hilang saat restart, rate-limit tidak berlaku multi-instance).
- 7.3 wajib sebelum data nyata masuk — disaster recovery harus tersedia di hari pertama go-live.
- 7.4 agar masalah terdeteksi sebelum user lapor (proaktif, bukan reaktif).
- 7.5 & 7.6 pengerasan akhir dan observabilitas sebelum go-live.

### Verifikasi Fase 7

- Semua test & build backend/frontend lulus.
- `docker compose up -d --build` berjalan lancar dengan Redis dan konfigurasi baru.
- Skrip backup berjalan dan restore berhasil di environment bersih.
- Endpoint `/api/metrics` mengembalikan data Prometheus valid.
- Container berjalan sebagai non-root; secret tidak terexpose di `docker inspect`.
- Log file ter-rotate dan terbaca di format JSON.

---

## Fase 8 — Pengalaman Lapangan

> Meningkatkan **produktivitas teknisi** yang bekerja di lapangan kawasan IKN dengan perangkat mobile.
> **Status: ⬜ BELUM** — dikerjakan setelah Fase 6 dan/atau Fase 7 tuntas.

### Task (berurutan — kerjakan dari atas ke bawah)

| # | Task | Keterangan | Skala | Status |
|---|------|-----------|-------|--------|
| 8.1 | Progressive Web App (PWA) | Service worker + web manifest → SIGAP bisa di-install di home screen HP teknisi; cache halaman shell agar loading cepat; badge ikon app. Scope: installability + offline shell (bukan full offline dengan IndexedDB). | Sedang | ⬜ Belum |
| 8.2 | Kalender Jadwal Pemeliharaan | Halaman `/kalender` (admin + teknisi): tampilan kalender bulanan (implementasi SVG/CSS mandiri, tanpa dependency chart eksternal) dengan dot/bar per hari menunjukkan jadwal; klik hari → daftar tugas; warna sesuai status; filter per teknisi. Data dari endpoint maintenance-logs + maintenance-schedules (jika 5.4 sudah ada). | Sedang | ⬜ Belum |
| 8.3 | Notifikasi Push Browser | Web Push API (VAPID keys) → notifikasi OS saat tugas baru ditugaskan, eskalasi overdue, atau hasil pengecekan `attention`; opt-in per user; fallback ke in-app jika push ditolak. Endpoint `POST /api/push/subscribe`, `DELETE /api/push/unsubscribe`; model `PushSubscription`. | Sedang | ⬜ Belum |
| 8.4 | Geotag Pengecekan Lapangan | Saat teknisi mengisi wizard pengecekan, posisi GPS (`navigator.geolocation`) dicatat otomatis; koordinat disimpan di `FieldMaintenance` (`check_latitude`, `check_longitude`); admin bisa melihat lokasi pengecekan di peta (verifikasi: teknisi benar-benar di lokasi CCTV). Persetujuan lokasi bersifat opsional — wizard tetap bisa diselesaikan tanpa GPS. | Kecil | ⬜ Belum |

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
- PWA: install di Chrome Android/desktop, ikon muncul, halaman shell termuat saat offline.
- Kalender: tampilan bulan benar, klik hari menampilkan tugas, warna status sesuai.
- Push: notifikasi OS muncul saat tugas baru ditugaskan; opt-out berfungsi.
- Geotag: koordinat tercatat di database; admin melihat pin lokasi pengecekan di peta.

---

## Kandidat Lanjutan (Dievaluasi Setelah Fase 8)

> Item-item berikut belum dijadwalkan tetapi berpotensi tinggi. Dievaluasi berdasarkan
> kebutuhan nyata setelah go-live dan feedback pengguna.

| Kandidat | Deskripsi | Prasyarat |
|----------|-----------|-----------|
| Integrasi Snapshot CCTV | Endpoint yang menangkap thumbnail/snapshot dari kamera CCTV (RTSP → JPEG) untuk ditampilkan di dashboard; membutuhkan akses jaringan ke NVR/kamera. | Akses jaringan CCTV |
| Inventaris Suku Cadang | Model `SparePart` + `SparePartUsage`: catat penggunaan suku cadang per tugas pemeliharaan; stok, minimum order, notifikasi stok rendah. | Fase 5 selesai |
| Multi-Zona / Multi-Site | Mendukung beberapa kawasan (bukan hanya IKN); model `Zone` sebagai parent `CCTV`; filter + dashboard per zona. | Kebutuhan ekspansi |
| Notifikasi Email (SMTP) | Kirim ringkasan harian, eskalasi overdue, dan pengumuman via email; konfigurasi SMTP; opt-in per user. | SMTP server tersedia |
| API Key untuk Integrasi | Model `APIKey` dengan scope terbatas; memungkinkan sistem lain membaca data CCTV/pemeliharaan tanpa login browser. | Kebutuhan integrasi |
| Laporan Kustom (Report Builder) | Admin memilih kolom, filter, dan format (PDF/CSV/Excel) → sistem generate laporan sesuai permintaan. | Fase 6 selesai |

---

## Konvensi Pengerjaan

1. `git checkout main && git pull origin main`
2. `git checkout -b feature/…` atau `bugfix/…`
3. Commit dengan Conventional Commits: `feat(scope): …` / `fix(scope): …`
4. Push → buat PR → tunggu merge oleh user → hapus branch lokal
5. Catat perubahan di `../backend/changelog.md` / `../frontend/changelog.md` (format: before/after,
   timestamp WITA, kategori, file, alasan)
6. Perbarui dokumen terdampak: `docs/AGENTS.md`, `docs/ROADMAP.md`, `docs/TESTING.md`, `docs/plan.md`
