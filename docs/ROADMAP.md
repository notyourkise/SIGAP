# ROADMAP — SIGAP

> Rencana pengembangan ke depan berdasarkan hasil `SECURITY-AUDIT.md`.
> Disusun: 2026-08-01

---

## Visi & Prinsip Prioritas

Pengembangan SIGAP diurutkan dari yang **paling berdampak pada keamanan** menuju **peningkatan
kapabilitas**. Prinsip:

1. **Keamanan dulu** — perbaikan kritis tidak boleh ditunda.
2. **Setiap fase dapat dirilis** — jangan menumpuk perubahan raksasa.
3. **Verifikasi wajib** — setiap fase lulus test, lint, build, dan uji manual.
4. **Konvensi tetap** — kerja di branch baru (`feature/…` / `bugfix/…`), Conventional Commits,
   dan catat perubahan di `../backend/changelog.md` / `../frontend/changelog.md`.

---

## Status Pengerjaan

| Fase | Fokus | Progress | Status |
|------|-------|----------|--------|
| 1 | Kritis (Prioritas Tertinggi) | 6/6 | ✅ Selesai |
| 2 | Penguatan (Hardening) | 9/9 | ✅ Selesai |
| 3 | Fitur (Kapabilitas) | 9/9 | ✅ Selesai |
| 4 | Infrastruktur & Kualitas | 2/8 + 4.1 sebagian | ⏸️ Sisa task ditunda (menjelang go-live) |
| 5 | Alur Kerja Operasional (Jalur A) | 1/5 | 🚧 Berjalan |

> **Urutan pengerjaan:** Fase 1 ✅ → Fase 2 ✅ → Fase 3 ✅ → **Fase 5 (Jalur A — aktif; kerjakan 5.1 → 5.5 berurutan)** → sisa Fase 4 (4.3–4.7, menjelang go-live produksi).
> Sisa task Fase 4 ditunda karena sistem masih pilot/UAT (keputusan 2026-08-04); pengerasan infrastruktur kembali diprioritaskan saat mendekati go-live.
> Di setiap tabel, task yang berada di posisi paling atas adalah yang **harus dikerjakan lebih dulu**.

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

## Fase 4 — Infrastruktur & Kualitas (Berkelanjutan)

> **Status: ⏸️ SISA TASK DITUNDA** (sejak 2026-08-04) — 4.2 & 4.8 selesai, 4.1 sebagian besar selesai;
> task 4.3–4.7 dikerjakan kembali menjelang go-live produksi setelah Fase 5 tuntas.
> Referensi lengkap cakupan test & gotchas: `docs/TESTING.md`.

| # | Task | Keterangan | Status |
|---|------|-----------|--------|
| 4.1 | Perluas unit test backend (cctv, upload, websocket, pagination, rate limit) | `backend/**/*_test.go` | 🚧 Sebagian besar (jalur CRUD, import boundary, WebSocket valid/tanpa token, routes, models, DSN, dan resolusi migration sudah diuji; valid import transaksi penuh dan migration update penuh ditunda) |
| 4.2 | Tambah `npm run lint` + `golangci-lint` di CI | `.github/workflows/ci.yml` | ✅ Selesai |
| 4.3 | Migrasi DB versioned (golang-migrate/Atlas) | ganti `AutoMigrate` bertahap | ⬜ Belum |
| 4.4 | Redis untuk rate-limit & blacklist (multi-instance) | `backend/config`, `backend/middleware/limiter.go`, `backend/utils/blacklist.go` | ⬜ Belum |
| 4.5 | Backup DB (`pg_dump` cron) + backup uploads + restore script | `scripts/` baru | ⬜ Belum |
| 4.6 | Monitoring: endpoint metrics (Prometheus) | `backend/handlers/metrics.go` (baru) | ⬜ Belum |
| 4.7 | Container non-root, resource limits, secret management | `backend/Dockerfile`, `docker-compose.yml` | ⬜ Belum |
| 4.8 | Isi `docs/changelog.md` (milestone monorepo) | `docs/changelog.md` | ✅ Selesai |

---

## Fase 5 — Alur Kerja Operasional (Jalur A)

> Menutup gap alur kerja harian admin & teknisi: notifikasi penugasan, sinkronisasi status
> perangkat, otomasi jadwal preventif, pemindahan tugas, dan eskalasi keterlambatan.
> **Status: 🚧 BERJALAN** — disepakati 2026-08-04. Sistem masih pilot/UAT; notifikasi **in-app saja**.

### Keputusan Desain (disepakati 2026-08-04)

- **Notifikasi in-app saja** — bell icon + daftar notifikasi + toast real-time via WebSocket; tanpa layanan eksternal (email/SMTP tidak dipakai).
- **Status CCTV otomatis bersifat konservatif** — hanya transisi `aktif ↔ pemeliharaan` yang dilakukan sistem; status `rusak`/`nonaktif` tetap keputusan manual admin dan tidak pernah ditimpa otomatis.
- **Prasyarat arsitektur** — `WSHub` diperluas agar menyimpan identitas koneksi (`user_id`, `role`) sehingga mendukung `BroadcastToUser`/`BroadcastToRole`; event broadcast global yang ada tetap dipertahankan (backward-compatible).

### Task (berurutan — kerjakan dari atas ke bawah)

| # | Task | Keterangan | Status |
|---|------|-----------|--------|
| 5.1 | Notifikasi in-app penugasan real-time | Model `Notification`; endpoint `GET /api/notifications`, `GET /api/notifications/unread-count`, `PUT /api/notifications/:id/read`, `PUT /api/notifications/read-all`; util `NotifyUser`/`NotifyRole`; perluasan `WSHub` (targeted per-user/per-role); komponen `NotificationBell` + integrasi `DashboardLayout`; hook: assign → teknisi, complete → admin, field-check `attention` → admin; event WS `NOTIFICATION_CREATED`. Branch: `feature/notification-in-app` | ✅ Kode selesai (2026-08-04) — menunggu uji runtime Docker di `https://localhost` |
| 5.2 | Sinkronisasi status CCTV otomatis (konservatif) | Helper dalam transaksi complete/field-check; `aktif→pemeliharaan` saat tugas gagal / hasil `attention`; `pemeliharaan→aktif` saat checklist 9/9 ok; `rusak`/`nonaktif` tidak disentuh; audit `UPDATE_CCTV` (keterangan otomatis) + broadcast `CCTV_UPDATED` | ⬜ Belum |
| 5.3 | Re-assign tugas oleh admin | `PUT /maintenance-logs/:id/assign` diperluas ke log `ongoing`; audit `REASSIGN_MAINTENANCE`; notif `MAINTENANCE_ASSIGNED` ke teknisi baru & `MAINTENANCE_UNASSIGNED` ke teknisi lama; tombol "Alihkan" di UI admin; tolak assign ke teknisi yang sama (400) | ⬜ Belum |
| 5.4 | Penjadwalan pemeliharaan berulang (recurring) | Model `MaintenanceSchedule` (frequency `daily`/`weekly`/`monthly`, `next_run_at`, `is_active`); kolom `MaintenanceScheduleID` di `MaintenanceLog` (link + idempotensi); goroutine generator harian (jalan juga saat startup, idempotent); CRUD admin `/api/maintenance-schedules`; halaman admin `/schedules` + menu sidebar | ⬜ Belum |
| 5.5 | Eskalasi overdue otomatis | Kolom `OverdueNotifiedAt` di `MaintenanceLog`; goroutine tiap jam: log `scheduled`/`ongoing` lewat jadwal → notif `MAINTENANCE_OVERDUE` ke semua admin (+ teknisi bila ongoing), terkirim sekali saja; flag di-reset saat reschedule | ⬜ Belum |

### Ketergantungan Antar Task

- 5.3 & 5.5 memakai sistem notifikasi dari 5.1 — **wajib setelah 5.1**.
- 5.2 berdiri di atas transaksi existing (bisa paralel dengan 5.1, tetapi notifikasi hasil bermasalah memakai 5.1).
- 5.4 berdiri sendiri; log hasil generator mengikuti alur yang sama (assign → 5.3, overdue → 5.5).

### Verifikasi Fase 5 (per task)

- `cd backend && go build ./... && go test ./... -count=1 && go vet ./...` — PASS (test sqlmock baru untuk tiap handler/job baru).
- `cd frontend && npm run lint && npm run build` — PASS.
- `swag init` setelah anotasi endpoint baru; uji runtime Docker di `https://localhost`.
- Uji manual per task: 5.1 lonceng notifikasi muncul real-time saat assign; 5.2 status CCTV berubah sesuai tabel keputusan (dan `rusak`/`nonaktif` tidak berubah); 5.3 alihkan tugas ke teknisi lain; 5.4 jadwal berulang membuat log otomatis tanpa duplikat; 5.5 notif overdue terkirim sekali saja.

---

## Kandidat Lanjutan (Jalur B — belum terjadwal, dievaluasi setelah Fase 5)

- Analitik kinerja teknisi (completion rate, rata-rata waktu penyelesaian, overdue per teknisi).
- Laporan PDF pengecekan lapangan per-record (`field-maintenance/:id/report-pdf`).
- Kalender jadwal pemeliharaan (tampilan bulanan).
- PWA / mode offline teknisi (checklist tersimpan lokal, sinkron saat online).
- Notifikasi email (butuh SMTP) — saat ini sengaja tidak dipakai.

---

## Konvensi Pengerjaan

1. `git checkout main && git pull origin main`
2. `git checkout -b feature/…` atau `bugfix/…`
3. Commit dengan Conventional Commits: `feat(scope): …` / `fix(scope): …`
4. Push → buat PR → tunggu merge oleh user → hapus branch lokal
5. Catat perubahan di `../backend/changelog.md` / `../frontend/changelog.md` (format: before/after,
   timestamp WITA, kategori, file, alasan)
