# ROADMAP — SIGAP

> Rencana pengembangan SIGAP — dari perbaikan keamanan hingga streaming CCTV real-time.
> Disusun: 2026-08-01 · Diperbarui: 2026-11-11 (sinkronisasi status rilis pasca Sprint hardening #139–#155 & PR audit #151/#152/#153)

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
| 4 | Infrastruktur & Kualitas | Ditutup; sisa task dipindahkan ke Fase 7 | ✅ Ditutup |
| 5 | Alur Kerja Operasional (Jalur A) | 5/5 | ✅ Selesai |
| 6 | Kecerdasan Operasional | 4/4 | ✅ Selesai |
| 7 | Kesiapan Produksi | 6/6 | ✅ Selesai |
| 8 | Pengalaman Lapangan | 4/4 | ✅ Selesai |
| **9** | **Streaming CCTV Real-Time** | **6/6** | **✅ Selesai** |
| **10** | **Kecerdasan Visual & Snapshot** | **4/4** | **✅ Selesai** |
| **11** | **Manajemen Operasional Lanjut** | **5/5** | **✅ Selesai** |
| **12** | **Integrasi & Skalabilitas** | **4/4** | **✅ Selesai** |
| **13** | **Perbaikan & Penguatan Fitur Existing** | **18/18 (Sprint 31–36)** | **✅ Selesai** |
| **14** | **CCTV Intelligence — Live Status & AI Takedown** | **Sprint 37 ✅ · Sprint 37-PATCH ✅ · Sprint 40 ✅** | **✅ Selesai** |
| **15** | **Branding & Dokumentasi** | **1/1** | **✅ Selesai** |

> **Update Terbaru 2026-11-11 (Hardening, Observabilitas, & Release Gate):** 
> - **Fase 12–15**: Implementasi fitur inti selesai ✅, dilanjutkan sprint hardening backlog **#139–#155** dan PR **#151/#152/#153** (isolasi site Log CCTV & WebSocket, trusted proxy `X-Forwarded-For`, timezone `Asia/Makassar`, retensi log 90 hari + deduplikasi, diagnostik Metro-E, metrik Prometheus + alert, CI/E2E Playwright, allowlist CIDR `/api/metrics`, enkripsi backup GPG headless, serta audit kualitas koordinat CCTV).
> - **Status Release Gate Saat Ini**: **Staging / Pre-Release Pilot** (belum dinyatakan production ready sampai gate manual test `#147` [Bug 4, 8, 21 — `docs/README.md` saat ini `22/25`], deployment & smoke test VPS `#148`, serta survei koordinat fisik 9 titik `#151` selesai atau risikonya disetujui tertulis).
> - **Manual Test Governance**: Dokumen testing formal terkelola di `docs/04-manual-tests/`.
> 
> ```
> Fase 1–15 + Sprint Hardening (#139–#155, #151–#153) ✅ (implementasi kode & uji otomatis selesai)
>     │
>     ▼
> Status saat ini: Staging / Pre-Release Pilot 🚧
>     │
>     ├── Multi-Site Isolation & WebSocket Scoping aktif ✅
>     ├── Trusted Proxy (Nginx -> API) & /api/metrics CIDR Allowlist ✅
>     ├── Backup Encryption GPG AES-256 (ENCRYPT_BACKUPS=true) ✅
>     ├── Coordinate Quality Audit Report & CSV Export (#151) ✅ (9 titik THD fisik pending)
>     ├── Gate Manual Test Bug 4, 8, 21 (#147: 22/25) ⏳ Menunggu verifikasi
>     └── Production VPS Deployment & Smoke Test (#148) ⏳ Menunggu jendela rilis
> ```

> **Urutan pengerjaan:**
> ```
> Fase 1–11 ✅ (semua selesai)
>     │
>     ▼
> Fase 12 (Integrasi) ✅
>     │
>     ▼
> Fase 13 (Perbaikan & Penguatan) ← kerjakan paralel dengan Fase 14
>     │
>     ├── Sprint 31–36 (P1–P18)
>     │
>     ▼
> Fase 14 (CCTV Intelligence) ← dapat dimulai setelah Sprint 31 selesai
>     │
>     ├── Sprint 37: RTSP Live Status Detection (14A) ✅
>     ├── Sprint 38: AI Microservice & Person Counting (14B) ❌ DIBATALKAN
>     ├── Sprint 39: Frontend AI Overlay & E2E Integration (14C) ❌ DIBATALKAN
>     └── Sprint 40: AI Feature Takedown (rollback) ✅ Selesai
> ```
> - **Fase 9** (streaming): prioritas utama — mengubah peta sebaran dari view-only menjadi live monitoring.
> - **Task 11.1** (suku cadang): dikerjakan langsung setelah Fase 9 stabil (kebutuhan vital operasional).
> - **Fase 10** (visual) dan **sisa Fase 11** bisa dikerjakan paralel setelahnya.
> - **Fase 12** (integrasi): selesai 4/4, termasuk API key, webhook, mobile app, dan multi-site.
> - **Fase 13** (perbaikan): penguatan kualitas & keamanan fitur existing berdasarkan analisis teknis 2026-08-10.
> - **Fase 14** (CCTV intelligence): deteksi online/offline real-time (Live Status, Sprint 37) + **Sprint 40 (AI Takedown)** — fitur AI detection dihapus; sistem kembali ke pure CCTV management.
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
| 1 | Network Access | VPS Hostinger (cloud). Port RTSP `554` pada IP Publik NVR (`<NVR_IP>`) dibuka oleh tim jaringan OIKN. |
| 2 | Bandwidth | **Sub Stream (02)** default (~0.5–1 Mbps/stream). Toggle opsional ke Main Stream (01). |
| 3 | Kredensial NVR | 1 NVR (`<NVR_IP>:554`), 35 kamera, credential tunggal. Password AES-256 encrypted di DB. |
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
<NVR_IP>:554     ◄── RTSP pull on-demand ──  :8889 HTTP  ──►  /stream/*  ──►  WebRTC/WHEP
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
> **Status: ✅ Selesai** — 4/4 task selesai (Sprint 29 ✅, Task 12.4 ✅, Sprint 30 ✅).

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
> **Status: ✅ Selesai**

| Jobdesk | Sub-Agent | Task | Deliverable | Status |
|---------|-----------|------|-------------|--------|
| **Backend** | Agent-BE | **12.2a** Model & Migrasi Multi-Site | Model `Site` (nama, kode, alamat, koordinat_pusat). FK `site_id` di CCTV, MaintenanceLog, dll. Migrasi: semua data existing → Site "OIKN" (default). CRUD `/api/sites`. | ✅ Selesai |
| **Frontend** | Agent-FE | **12.2b** UI Multi-Site | Switch site di navigasi (dropdown header). Dashboard per site. Filter global per site. User assignment ke multiple sites. | ✅ Selesai |
| **QA/Docs** | Agent-QA | Manual Test | `docs/manual-test-multi-site.md`. | ✅ Selesai |

**✅ Selesai — Ringkasan Sprint 30:** Multi-Site (Task 12.2) selesai.
- **Backend**: model `Site` (nama, kode, alamat, koordinat_pusat, `is_default`); FK `site_id` di `CCTV`/`MaintenanceLog`/zona; migrasi data existing → site default "Kawasan OIKN" (kode `OIKN`, `is_default=true`); CRUD `/api/sites` (paginasi `GET /api/sites`, dropdown `GET /api/sites/all`) admin-only untuk tulis; `POST /api/sites/:id/assign-users` (assign teknisi, admin-only); perlindungan penghapusan site default; unit test `handlers/site_test.go` & `middleware/site_test.go`; Swagger + `swag init`.
- **Frontend**: site switcher (dropdown) di header; cookie `sigap_site_id` untuk menyimpan site aktif; reload halaman + filter global per site (CCTV/log/zona); halaman manajemen site admin.
- **Dokumen**: `docs/manual-test-multi-site.md` (baru — S1–S7 sesuai format manual test lain).

Detail: `docs/changelog.md` [1.28.41].

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

## Fase 13 — Perbaikan & Penguatan Fitur Existing

> Fase ini **tidak menambah fitur baru**, melainkan memperbaiki kualitas, keamanan, dan ketangguhan
> dari seluruh fitur yang sudah terimplementasi (Fase 1–12). Berdasarkan **analisis teknis 2026-08-10**.
> **Status: ✅ Selesai** — 18/18 task, 6 sprint (Sprint 31–36).

### Ringkasan 18 Perbaikan

| ID | Prioritas | Masalah | Upaya |
|----|-----------|---------|-------|
| P1 | 🔴 Tinggi | Koordinat CCTV disimpan sebagai `varchar` — tidak ada validasi, rentan data kacau | Sedang |
| P2 | 🔴 Tinggi | Fallback koordinat menyebar CCTV tanpa GPS di sekitar pusat IKN — menyesatkan admin | Rendah |
| P3 | 🔴 Tinggi | `RTSPPassword` berpotensi bocor ke respons API jika field terisi | Rendah |
| P4 | 🔴 Tinggi | `ChannelPort` tidak ada constraint unik — dua CCTV bisa konflik di NVR | Rendah |
| P5 | 🔴 Tinggi | Stream auto-registration berjalan sekali tanpa retry — gagal diam-diam | Rendah |
| P6 | 🟡 Menengah | Tabel notifikasi tumbuh tanpa batas — tidak ada kebijakan cleanup | Rendah |
| P7 | 🟡 Menengah | Email opt-in/out all-or-nothing — tidak ada granularitas per tipe notifikasi | Rendah |
| P8 | 🟡 Menengah | Health score hanya nilai terkini — tidak ada histori / tren | Sedang |
| P9 | 🟡 Menengah | Daily digest terlewat jika server mati saat 06:00 WITA | Rendah |
| P10 | 🟡 Menengah | Webhook delivery log tumbuh tanpa batas retensi | Rendah |
| P11 | 🟡 Menengah | Eskalasi overdue hanya satu kali — tidak ada tingkatan urgensi | Sedang |
| P12 | 🟡 Menengah | Checklist 9 tahap pengecekan lapangan hardcoded di kode | Tinggi |
| P13 | 🟢 Rendah | Foto upload teknisi disimpan tanpa kompresi — ukuran file besar | Sedang |
| P14 | 🟢 Rendah | QR Code tidak ada mekanisme rotasi untuk keamanan | Rendah |
| P15 | 🟢 Rendah | StreamGridView memuat semua WHEPPlayer sekaligus — boros resource | Rendah |
| P16 | 🟢 Rendah | API Key tidak bisa diregenerasi tanpa menghapus — merepotkan integrasi | Rendah |
| P17 | 🟢 Rendah | Kalender tidak bisa membuat tugas baru langsung dari klik tanggal | Rendah |
| P18 | 🟢 Rendah | Dashboard kinerja teknisi tidak ada filter periode (7 hari / 30 hari / dll.) | Rendah |

---

### Sprint 31 — Keamanan & Integritas Data (P1, P2, P3, P4, P5)

> **Branch:** `fix/data-integrity-security`
> **Prasyarat:** Tidak ada — bisa langsung dikerjakan setelah Fase 12 Task 12.1+12.3 selesai.
> **Estimasi:** 3–4 hari kerja

| ID | Jobdesk | Sub-Agent | Task | Deliverable |
|----|---------|-----------|------|-------------|
| **P1** | **Backend** | Agent-BE | **13.1** Migrasi Koordinat varchar → float | Buat `000012_cctv_coords_to_float.up.sql`: `ALTER TABLE cctvs ALTER COLUMN latitude TYPE double precision USING NULLIF(latitude, '')::double precision`, idem longitude. Update model GORM: `Latitude float64`, `Longitude float64`. Update semua handler yang menerima/mereturn koordinat — validasi range lat (-90..90), lng (-180..180) di `CreateCCTV` dan `UpdateCCTV`. Update seed data. Update Excel import handler (`ImportCCTV`) agar parse float bukan string. Update unit test `cctv_test.go`. |
| **P2** | **Frontend** | Agent-FE | **13.2** Perbaikan Visual Marker Koordinat Kosong | Di `CCTVMap.tsx`: ganti fallback scatter → marker abu-abu bertanda tanya (`?`) dengan icon berbeda dari `createCustomMarkerIcon`. Popup: tambah banner peringatan merah "⚠️ Koordinat belum diisi — klik untuk edit". Link dari popup ke form edit CCTV (`/cctv?edit={id}`). Tambah counter di status bar peta: "X dari N CCTV belum memiliki koordinat". |
| **P3** | **Backend** | Agent-BE | **13.3** Hardening RTSPPassword di Semua Handler | Audit seluruh handler yang mereturn data CCTV: `GetAllCCTV`, `GetCCTVByID`, `ScanCCTV`, `BulkDeleteCCTV`, `ImportCCTV`, `GetStreamStatus`, `GetStreamURL`. Di setiap titik return, pastikan `cctv.RTSPPassword = ""`. Tambah unit test khusus: `TestRTSPPasswordNotInResponse` yang memverifikasi field tidak ada atau kosong di respons JSON untuk semua endpoint tersebut. Pertimbangkan membuat DTO `CCTVResponse` terpisah yang tidak memiliki field `RTSPPassword`. |
| **P4** | **Backend** | Agent-BE | **13.4** Validasi Keunikan ChannelPort | Di `CreateCCTV`: sebelum `config.DB.Create`, cek `config.DB.Where("channel_port = ? AND channel_port != 0", req.ChannelPort).First(&existing)`. Return 400 jika duplikat. Di `UpdateCCTV`: cek `channel_port != cctv.ChannelPort && duplikat`. Tambah validasi range: `channel_port` harus 1–35 (sesuai jumlah kamera). Update unit test. Catatan: TIDAK tambah DB unique constraint karena `channel_port=0` (belum diisi) valid untuk banyak CCTV. |
| **P5** | **Backend** | Agent-BE | **13.5** Retry Logic Stream Auto-Registration | Di `handlers/stream.go`, modifikasi `RegisterActiveStreams`: tambah retry loop maksimal 5× dengan backoff (2s, 4s, 8s, 16s, 30s). Sebelum iterasi CCTV, cek terlebih dahulu apakah MediaMTX API reachable via `mediamtxClient.GetPath("__ping__")` atau `GET /v3/config/global/get`. Log berapa stream berhasil / gagal. Tambah scheduler periodik (tiap 10 menit) yang mengecek stream `stream_enabled=true` yang belum terdaftar di MediaMTX dan mendaftarkannya kembali. Unit test `TestRegisterActiveStreams_RetryOnFailure`. |

#### Verifikasi Sprint 31

```bash
cd backend && go build ./... && go test ./... -count=1 && go vet ./...
cd frontend && npm run lint && npm run build
```

> **Acceptance Criteria:**
> - `RTSPPassword` tidak muncul di output apapun dari `curl /api/cctv`
> - Dua CCTV dengan channel_port yang sama ditolak oleh API dengan pesan error yang jelas
> - Server restart tidak menyebabkan stream gagal register (retry berhasil)
> - CCTV tanpa koordinat ditampilkan dengan marker abu-abu di peta, bukan marker warna random

---

### Sprint 32 — Notifikasi & Preferensi Email (P6, P7)

> **Branch:** `feature/notification-improvements`
> **Prasyarat:** Sprint 31 selesai (tidak wajib, bisa paralel)
> **Estimasi:** 2–3 hari kerja

| ID | Jobdesk | Sub-Agent | Task | Deliverable |
|----|---------|-----------|------|-------------|
| **P6** | **Backend** | Agent-BE | **13.6** Cleanup Notifikasi Otomatis | Tambah goroutine `StartNotificationCleanup` di `main.go` (interval: tiap 7 hari). Logic: hapus notifikasi `read_at IS NOT NULL AND created_at < NOW() - INTERVAL '30 days'`, dan `read_at IS NULL AND created_at < NOW() - INTERVAL '90 days'`. Tambah endpoint admin `DELETE /api/notifications/cleanup?older_than=30d` untuk trigger manual. Update `GetAllNotifications` handler: tambah query `COUNT(*)` untuk jumlah total notifikasi user — tampilkan di response. Unit test cleanup logic. |
| **P7a** | **Backend** | Agent-BE | **13.7** Preferensi Email Per-Tipe | Buat migrasi `000013_email_prefs_granular.up.sql`: tambah kolom `email_daily_digest BOOLEAN NOT NULL DEFAULT TRUE`, `email_overdue BOOLEAN NOT NULL DEFAULT TRUE`, `email_stock_low BOOLEAN NOT NULL DEFAULT TRUE` ke tabel `users`. Hapus kolom `email_notifications` (atau set sebagai alias master). Update model `User` di `models/setup.go`. Update endpoint `PUT /api/profile/email-preferences` untuk menerima ketiga field baru. Update `SendDailyDigestEmail`, `SendOverdueEmail`, `SendStockLowEmail` di `utils/email.go` agar cek field masing-masing sebelum kirim. |
| **P7b** | **Frontend** | Agent-FE | **13.7b** UI Preferensi Email Granular | Di halaman profil (`/profile`): ganti toggle tunggal "Terima notifikasi email" dengan 3 toggle terpisah: "📅 Ringkasan Harian", "⚠️ Eskalasi Overdue", "📦 Stok Suku Cadang Rendah". Masing-masing toggle memanggil `PUT /api/profile/email-preferences`. Simpan state ke API — bukan localStorage. |

#### Verifikasi Sprint 32

```bash
cd backend && go build ./... && go test ./... -count=1 && go vet ./...
cd frontend && npm run lint && npm run build
```

> **Acceptance Criteria:**
> - User yang opt-out "Ringkasan Harian" tidak menerima email digest, tapi masih menerima email overdue
> - Notifikasi lama (>30 hari dibaca) terhapus otomatis setiap minggu
> - Admin bisa trigger cleanup manual via endpoint

---

### Sprint 33 — Kecerdasan Data & Eskalasi (P8, P9, P11)

> **Branch:** `feature/data-intelligence`
> **Prasyarat:** Sprint 31 selesai
> **Estimasi:** 3–4 hari kerja

| ID | Jobdesk | Sub-Agent | Task | Deliverable |
|----|---------|-----------|------|-------------|
| **P8a** | **Backend** | Agent-BE | **13.8** Histori Health Score CCTV | Buat migrasi `000014_cctv_health_histories.up.sql`: `CREATE TABLE cctv_health_histories (id uuid PRIMARY KEY, cctv_id uuid NOT NULL REFERENCES cctvs(id), health_score integer NOT NULL, calculated_at timestamptz NOT NULL DEFAULT NOW())`. Index: `(cctv_id, calculated_at)`. Buat model `CCTVHealthHistory` di `models/`. Di `RunHealthScoreCalculator`: setelah update `health_score` di tabel `cctvs`, insert baris baru ke `cctv_health_histories`. Tambah endpoint `GET /api/cctv/:id/health-history?range=7d|30d|90d` yang mereturn array `{health_score, calculated_at}`. |
| **P8b** | **Frontend** | Agent-FE | **13.8b** Grafik Tren Skor Kesehatan | Di modal detail CCTV (atau halaman detail jika ada): tambah tab/section "Tren Kesehatan". Fetch dari `GET /api/cctv/:id/health-history?range=30d`. Render line chart menggunakan komponen `TrendChart` yang sudah ada. Axis X: tanggal, axis Y: 0–100. Highlight titik nilai minimum (kondisi terburuk). |
| **P9** | **Backend** | Agent-BE | **13.9** Daily Digest Catch-Up Logic | Di `RunDailyDigestGenerator` (`handlers/digest_generator.go`): tambah fungsi `checkAndGenerateMissedDigest(db)` yang dipanggil saat startup. Logic: query apakah `daily_digests` memiliki record untuk rentang hari ini (00:00–23:59 WITA). Jika tidak ada → generate digest untuk jendela 24 jam terakhir. Gunakan transaksi idempoten (sudah ada pola di kode). Log: "Digest hari ini tidak ditemukan, generate catch-up..." |
| **P11a** | **Backend** | Agent-BE | **13.11** Eskalasi Overdue Bertingkat | Di `models/setup.go` atau `MaintenanceLog`: tambah kolom `overdue_level INTEGER NOT NULL DEFAULT 0` via migrasi `000015_overdue_level.up.sql`. Di `RunOverdueChecker`: tambah logic tingkatan — level 1 (>24h): notifikasi biasa (sudah ada), level 2 (>48h): notifikasi "Perhatian Tinggi" + broadcast `OVERDUE_HIGH`, level 3 (>72h): notifikasi "Kritis" + broadcast `OVERDUE_CRITICAL` + kirim email ke admin yang opt-in overdue. Cek `overdue_level < target_level` sebelum kirim — jangan kirim ulang yang sudah dikirim. |
| **P11b** | **Frontend** | Agent-FE | **13.11b** UI Eskalasi Overdue Visual | Di tabel maintenance (`/maintenance`): tambah kolom "Status Overdue" dengan badge warna — abu-abu (belum overdue), kuning (level 1: >24h), oranye (level 2: >48h), merah berkedip (level 3: >72h). Di notifikasi bell: tampilkan ikon darurat 🚨 untuk notifikasi `OVERDUE_CRITICAL`. |

#### Verifikasi Sprint 33

```bash
cd backend && go build ./... && go test ./... -count=1 && go vet ./... && swag init
cd frontend && npm run lint && npm run build
```

> **Acceptance Criteria:**
> - Health score history tersedia untuk 30 hari terakhir dan tampil sebagai grafik
> - Restart server tidak menyebabkan digest hari ini terlewat
> - Tugas yang overdue 72+ jam menghasilkan notifikasi kritis dan email (jika opt-in)

---

### Sprint 34 — Infrastruktur & Integrasi (P10, P14, P16)

> **Branch:** `feature/infra-integration-fixes`
> **Prasyarat:** Sprint 31 selesai
> **Estimasi:** 2–3 hari kerja
> **Status: ✅ Selesai** (2026-08-10)

| ID | Jobdesk | Sub-Agent | Task | Deliverable | Status |
|----|---------|-----------|------|-------------|--------|
| **P10** | **Backend** | Agent-BE | **13.10** Retensi Webhook Delivery Log | Goroutine `StartWebhookLogCleanup` (interval 7 hari); kolom `max_log_retention_days` di `WebhookEndpoint`; cleanup periodik menghormati retensi per endpoint; endpoint `DELETE /api/webhooks/:id/deliveries?older_than=7d`; UI menampilkan jumlah log per endpoint + tombol "Bersihkan Log". | ✅ Selesai |
| **P14a** | **Backend** | Agent-BE | **13.14** QR Code Rotation Endpoint | Handler `RotateQRCode` (`POST /api/cctv/:id/rotate-qr`, admin only): UUID baru → enkripsi AES-256 → update `QrPayload` → audit + broadcast. | ✅ Selesai |
| **P14b** | **Frontend** | Agent-FE | **13.14b** UI Cetak Ulang QR Setelah Rotasi | Tombol "🔄 Rotasi QR" di dropdown aksi per baris CCTV; konfirmasi; setelah sukses langsung siapkan cetak label QR baru. | ✅ Selesai |
| **P16a** | **Backend** | Agent-BE | **13.16** API Key Regeneration Endpoint | Handler `RegenerateAPIKey` (`POST /api/api-keys/:id/regenerate`, admin only): key mentah baru → hash → update `key_hash`/`key_prefix` → reset `last_used_at` → audit + broadcast → return `{ api_key }` 1×. | ✅ Selesai |
| **P16b** | **Frontend** | Agent-FE | **13.16b** UI Regenerasi API Key | Tombol "🔄 Regenerasi" per baris; dialog konfirmasi; key mentah baru ditampilkan modal one-time + tombol salin. | ✅ Selesai |

#### Verifikasi Sprint 34

```bash
cd backend && go build ./... && go test ./... -count=1 && go vet ./... && swag init
cd frontend && npm run lint && npm run build
```

> **Acceptance Criteria:**
> - Log webhook lebih dari 30 hari terhapus otomatis
> - QR Code lama tidak bisa dipakai scan setelah rotasi
> - API Key yang diregenerasi mengembalikan raw key baru dan key lama tidak lagi valid

---

### Sprint 35 — Performa Backend & Checklist Fleksibel (P12, P13)

> **Branch:** `feature/performance-checklist`
> **Prasyarat:** Sprint 31 selesai
> **Estimasi:** 4–5 hari kerja
> **Status: ✅ Selesai** (2026-08-12)

| ID | Jobdesk | Sub-Agent | Task | Deliverable | Status |
|----|---------|-----------|------|-------------|--------|
| **P12a** | **Backend** | Agent-BE | **13.12** Checklist Template yang Dapat Dikustomisasi | Buat migrasi `000016_checklist_templates.up.sql`: `CREATE TABLE checklist_templates (id uuid PRIMARY KEY, nama_item varchar(255) NOT NULL, deskripsi text, urutan integer NOT NULL DEFAULT 0, wajib boolean NOT NULL DEFAULT TRUE, aktif boolean NOT NULL DEFAULT TRUE, created_at timestamptz, updated_at timestamptz)`. Seed data: 9 item checklist default (sama dengan yang saat ini hardcoded). CRUD handler `/api/checklist-templates` (admin only): GetAll, Create, Update, Delete (soft: set `aktif=false`). Update `CreateFieldMaintenance` handler: ambil item dari DB (filter `aktif=true`, order by `urutan`) bukan dari konstanta hardcoded. Pertahankan backward compat: `FieldMaintenanceCheck` existing tidak terpengaruh. | ✅ Selesai |
| **P12b** | **Frontend** | Agent-FE | **13.12b** UI Manajemen Template Checklist | Di halaman `/field-maintenance` (atau halaman pengaturan admin baru): tambah tab/section "Template Checklist". Tabel: nomor urut, nama item, wajib/opsional, aktif/nonaktif. Aksi: edit nama, drag-reorder urutan, toggle aktif. Buat komponen `ChecklistTemplateManager`. | ✅ Selesai |
| **P13a** | **Backend** | Agent-BE | **13.13** Kompresi Foto Upload Server-Side | Di `handlers/upload.go`, setelah file diterima dan sebelum disimpan ke disk, tambah kompresi gambar: gunakan library `golang.org/x/image` atau `github.com/disintegration/imaging`. Logic: 1) Decode JPEG/PNG, 2) Resize ke max 1920×1080 (pertahankan aspect ratio), 3) Encode JPEG quality 85. Simpan file terkompresi. Generate juga thumbnail 400×300 JPEG dengan suffix `_thumb.jpg` → simpan di `uploads/thumbnails/`. Skip kompresi jika ukuran file sudah < 300KB. Tambah header `X-Original-Size` dan `X-Compressed-Size` di response untuk transparency. | ✅ Selesai |
| **P13b** | **Frontend** | Agent-FE | **13.13b** Tampilkan Thumbnail di Daftar Maintenance | Di tabel maintenance (`/maintenance`): ganti placeholder foto dengan `<img src="/api/uploads/{id}_thumb.jpg">` menggunakan `AuthenticatedImage` component yang sudah ada. Di modal detail: tampilkan foto fullsize saat diklik thumbnail. Fallback ke ikon kamera jika thumbnail tidak ada. | ✅ Selesai |

**✅ Selesai — Ringkasan Sprint 35:** Performa Backend & Checklist Fleksibel (P12, P13) selesai.
- **Backend (`0.56.0` + `0.57.0`)**: model `ChecklistTemplate` + migrasi `000017_checklist_templates` (tabel + kolom `wajib` di `field_maintenance_checks`); seeder 9 item default; handler/rute CRUD + reorder `/api/checklist-templates` (admin-only untuk tulis); `CreateFieldMaintenance` menyimpan `Wajib` pada item. Kompresi foto server-side (resize ≤1920, JPEG q85, header `X-Original/Compressed-Size`) + thumbnail 400×300 via `github.com/disintegration/imaging`; endpoint `GET /api/uploads/thumbnails/:filename`.
- **Frontend (`1.48.0`)**: halaman admin `/checklist-templates` + `ChecklistTemplateManager` (tambah/edit/reorder/toggle/hapus); `FieldMaintenanceModal` memuat template dari server (fallback `FIELD_CHECKLIST`); thumbnail 40×40 di tabel `/maintenance` (klik → detail fullsize); menu sidebar "Template Checklist" + guard proxy.
- **Mobile (`1.1.0`)**: Checkup Wizard memuat item checklist dari `/checklist-templates` (fallback konstanta).
- **Dokumen**: `docs/manual-test-checklist-template.md` (baru), `docs/manual-test-image-compression.md` (baru); ROADMAP Fase 13 progress **13/18 → 15/18** & Sprint 35 ✅ Selesai.

Detail: `backend/changelog.md` [0.56.0] + [0.57.0] · `frontend/changelog.md` [1.48.0] · `mobile/changelog.md` [1.1.0].

#### Verifikasi Sprint 35

```bash
cd backend && go build ./... && go test ./... -count=1 && go vet ./...
cd frontend && npm run lint && npm run build
```

> **Acceptance Criteria:**
> - Admin bisa menambah/mengubah urutan item checklist dari UI tanpa deploy ulang
> - Foto yang diupload > 300KB otomatis dikompresi ke < 300KB
> - Thumbnail tersedia di tabel maintenance

---

### Sprint 36 — Polish Frontend & UX (P15, P17, P18)

> **Branch:** `fix/frontend-ux-polish`
> **Prasyarat:** Tidak ada — bisa paralel dengan sprint lain
> **Estimasi:** 2–3 hari kerja
> **Status: ✅ Selesai** (2026-08-13)

| ID | Jobdesk | Sub-Agent | Task | Deliverable | Status |
|----|---------|-----------|------|-------------|--------|
| **P15** | **Frontend** | Agent-FE | **13.15** Lazy Loading StreamGridView | Di `StreamGridView.tsx`: implementasi `IntersectionObserver` — `WHEPPlayer` hanya di-mount saat sel memasuki viewport. Tambah tombol "⏸ Pause" per sel yang menutup `RTCPeerConnection` sementara (bukan unmount komponen — pertahankan placeholder). Tambah pagination grid: tampilkan 12 kamera per halaman (configurable via dropdown: 4/8/12/all). Tambah counter "Menampilkan X dari Y kamera aktif" di header grid. | ✅ Selesai |
| **P17** | **Frontend** | Agent-FE | **13.17** Buat Tugas dari Kalender | Di `CalendarGrid.tsx`: tambah handler `onDayClick(date: Date)` — klik pada sel tanggal membuka modal create maintenance baru dengan field `scheduled_date` sudah terisi tanggal yang diklik. Reuse komponen modal form yang sudah ada di `/maintenance` (lift state atau gunakan portal). Setelah submit berhasil: refresh kalender via `loadMap()` / WebSocket `MAINTENANCE_UPDATED` event. | ✅ Selesai |
| **P18a** | **Backend** | Agent-BE | **13.18** Filter Periode Kinerja Teknisi | Di `handlers/dashboard.go`, update handler `GetTechnicianPerformance`: tambah query parameter `?period=7d|30d|90d|365d` (default: `30d`). Konversi ke `start_date = NOW() - INTERVAL 'X days'`. Terapkan filter ke semua query (completion rate, avg response time, task count). | ✅ Selesai |
| **P18b** | **Frontend** | Agent-FE | **13.18b** UI Filter Periode Kinerja Teknisi | Di halaman `/kinerja-teknisi`: tambah dropdown filter periode di header ("7 Hari", "30 Hari", "3 Bulan", "1 Tahun"). State `period` memicu fetch ulang ke API dengan query param. Update komponen `TechnicianPerformanceChart` untuk menerima data baru. Label axis X di chart menyesuaikan granularitas periode (harian/mingguan/bulanan). | ✅ Selesai |

**✅ Selesai — Ringkasan Sprint 36:** Polish Frontend & UX (P15, P17, P18) selesai — **Fase 13 kini 18/18 ✅ Selesai**.
- **Backend (`0.58.0`)**: `GET /dashboard/technician-performance` tambah query `?period=7d|30d|90d|365d` (default 30d) yang mengesampingkan `start_date`/`end_date`; unit test `TestTechnicianPerformanceWithPeriod`.
- **Frontend (`1.49.0`)**: `StreamGridView` pagination (4/8/12/Semua) + counter "Menampilkan X–Y dari N" + pause/resume per sel (lazy-load IntersectionObserver dipertahankan); `CalendarGrid` + halaman `/kalender` mendukung create maintenance dari klik tanggal (modal, pre-filled, refresh); `/kinerja-teknisi` filter periode 7/30/90/365 hari/kustom memakai `?period=`.
- **Dokumen**: `docs/manual-test-frontend-ux-polish.md` (baru); ROADMAP Fase 13 progress **15/18 → 18/18** & Sprint 36 ✅ Selesai.

Detail: `backend/changelog.md` [0.58.0] · `frontend/changelog.md` [1.49.0].

#### Verifikasi Sprint 36

```bash
cd frontend && npm run lint && npm run build
cd backend && go build ./... && go test ./... -count=1 && go vet ./...
```

> **Acceptance Criteria:**
> - Membuka StreamGridView dengan 35 kamera tidak memuat lebih dari 12 WHEPPlayer sekaligus
> - Klik tanggal di kalender membuka modal create maintenance dengan tanggal pre-filled
> - Dashboard kinerja teknisi menampilkan data sesuai periode yang dipilih

---

### Ringkasan Estimasi Sprint Fase 13

```
═══════════════════════════════════════════════════════════════
FASE 13 — Perbaikan & Penguatan Fitur Existing
───────────────────────────────────────────────────────────────
Sprint 31: Keamanan & Integritas Data (P1-P5) ......... 3-4 hari
Sprint 32: Notifikasi & Email (P6-P7) ................. 2-3 hari
Sprint 33: Kecerdasan Data & Eskalasi (P8,P9,P11) .... 3-4 hari
Sprint 34: Infrastruktur & Integrasi (P10,P14,P16) ... 2-3 hari
Sprint 35: Performa & Checklist (P12-P13) ............. 4-5 hari
Sprint 36: Polish Frontend UX (P15,P17,P18) ........... 2-3 hari
───────────────────────────────────────────────────────────────
Total Estimasi: 16-22 hari kerja
Prioritas SEGERA: Sprint 31 (P1-P5 semuanya high-priority)
═══════════════════════════════════════════════════════════════
```

---

## Fase 14 — CCTV Intelligence: Live Status & AI Takedown

> Fase ini awalnya menambahkan **dua kemampuan baru** yang saling melengkapi, namun per 2026-08-13 fitur AI detection **dihapus** (Sprint 40):
> - **14A**: Deteksi status online/offline kamera secara real-time (RTSP Probe) — memperbaiki bug fundamental di mana peta tetap hijau meski kamera mati. ✅ Selesai (Sprint 37).
> - **14B/14C**: AI object detection berbasis YOLOv8 (person counting + bounding box) — **DIBATALKAN**; seluruh artefak dihapus via **Sprint 40 (AI Feature Takedown)**. Sistem kembali ke pure CCTV management tanpa AI inference.
>
> **Status: ✅ Selesai** — Sprint 37 (14A Live Status), Sprint 37-PATCH (strategi Metro-E vs GSM), dan Sprint 40 (AI Takedown) selesai; Sprint 38 & 39 dibatalkan.

### Latar Belakang & Root Cause

#### Mengapa peta selalu hijau padahal kamera mati?

Terdapat **3 lapisan bug** yang menyebabkan status kamera selalu tampak "online":

```
BUG 1: sourceOnDemand: yes (mediamtx.yml)
─────────────────────────────────────────
MediaMTX hanya connect ke RTSP kamera saat ada yang menonton stream.
Jika tidak ada viewer → path tidak pernah Ready=true
→ streamLastReady[path] tidak pernah diset

BUG 2: evaluateStreamState() — kondisi guardian (stream.go:694)
─────────────────────────────────────────
if _, wasUp := streamLastReady[path]; !wasUp {
    return  ← SKIP jika kamera belum pernah ditonton
}
Kamera yang tidak pernah dibuka = tidak pernah dianggap down

BUG 3: CCTVMap menggunakan field "status" bukan "stream_live"
─────────────────────────────────────────
Warna marker = health_score / status GORM (aktif/nonaktif/pemeliharaan)
Bukan status koneksi RTSP real-time kamera
Walau STREAM_DOWN di-broadcast, marker tidak berubah warna
```

#### Solusi Arsitektur Fase 14

```
┌──────────────────────────────────────────────────────────────────────┐
│                         FASE 14 ARCHITECTURE                         │
│                                                                      │
│  NVR Hikvision (<NVR_IP>:554)                                        │
│         │                                                            │
│         │ RTSP pull (per channel)                                    │
│         ▼                                                            │
│  ┌──────────────────┐     ┌────────────────────────────────────┐     │
│  │  sigap_api (Go)  │────▶│  RTSP Probe Engine (NEW - 14A)     │     │
│  │  RunRTSPProber   │     │  tiap 2 menit: TCP dial port 554   │     │
│  │  + DB update     │     │  + RTSP OPTIONS per channel        │     │
│  │  + WS broadcast  │     │  → update stream_live di DB        │     │
│  └──────────────────┘     │  → broadcast CCTV_LIVE_STATUS_UPDATED│  │
│         │                 └────────────────────────────────────┘     │
│         │ HTTP internal                                              │
│         ▼                                                            │
│  ┌──────────────────────────────────────┐                            │
│  │  sigap_ai (Python NEW - 14B)         │                            │
│  │  FastAPI + YOLOv8n + ByteTrack       │                            │
│  │  Pull RTSP: hanya kamera di /monitor │                            │
│  │  Inference: 1 frame/detik per kamera │                            │
│  │  Output: koordinat box + tracking ID │                            │
│  └──────────────────────────────────────┘                            │
│         │ WebSocket push                                             │
│         ▼                                                            │
│  ┌──────────────────────────────────────────────────────────┐        │
│  │  Browser (/monitor)                                       │        │
│  │  <canvas> overlay bounding box di atas <video> WebRTC    │        │
│  │  Badge: 🟢 Online / 🔴 Offline per kamera               │        │
│  │  Counter: "X orang unik terdeteksi hari ini"             │        │
│  └──────────────────────────────────────────────────────────┘        │
└──────────────────────────────────────────────────────────────────────┘
```

### Skema Database Baru (Fase 14)

```sql
-- Migration 000018: RTSP Live Status
ALTER TABLE cctvs
  ADD COLUMN stream_live       BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN stream_checked_at TIMESTAMPTZ;

-- Migration 000019: AI Detection Sessions (person counting)
CREATE TABLE ai_detection_sessions (
  id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  cctv_id          UUID NOT NULL REFERENCES cctvs(id) ON DELETE CASCADE,
  session_date     DATE NOT NULL,           -- tanggal untuk reset harian
  unique_person_count INT NOT NULL DEFAULT 0,
  last_tracking_ids   JSONB,               -- set ID ByteTrack yg sudah dihitung
  created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (cctv_id, session_date)
);
CREATE INDEX idx_ai_sessions_cctv_date ON ai_detection_sessions (cctv_id, session_date);

-- Migration 000020: AI Detection Events log
CREATE TABLE ai_detection_events (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  cctv_id         UUID NOT NULL REFERENCES cctvs(id) ON DELETE CASCADE,
  event_type      VARCHAR(50) NOT NULL,    -- 'person_detected', 'person_new_unique'
  person_count    INT NOT NULL DEFAULT 0,  -- jumlah person dalam frame saat ini
  unique_count    INT NOT NULL DEFAULT 0,  -- total unik hari ini
  confidence_avg  FLOAT,
  detected_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_ai_events_cctv_date ON ai_detection_events (cctv_id, detected_at DESC);
```

### Struktur Folder AI Service (Baru)

```
ai/
├── Dockerfile                  # python:3.11-slim, install ultralytics, fastapi, uvicorn
├── requirements.txt
├── main.py                     # FastAPI app entry point
├── routers/
│   ├── session.py              # POST /sessions, DELETE /sessions/{cctv_id}
│   └── health.py               # GET /health
├── core/
│   ├── detector.py             # YOLOv8n inference wrapper
│   ├── tracker.py              # ByteTrack person tracker
│   ├── person_counter.py       # unique ID counting + daily reset logic
│   └── session_manager.py      # manage active RTSP stream per session
├── models/
│   └── yolov8n.pt              # model nano (6MB) — auto-download saat build
└── tests/
    └── test_detector.py
```

### Spesifikasi Teknis AI

| Parameter | Nilai | Catatan |
|-----------|-------|---------|
| **Model** | YOLOv8n (nano) | 6MB, CPU-friendly, ~30ms/frame |
| **Class yang dideteksi** | Class 0 (person) saja | Filter output YOLOv8 |
| **Confidence threshold** | ≥ 0.50 | Kurangi false positive |
| **Inference interval** | 1 frame/detik per kamera | Balance CPU vs akurasi |
| **Max concurrent kamera** | Sesuai grid aktif (max 16 untuk 4×4) | Tidak proses kamera idle |
| **Tracker** | ByteTrack | Open-source, akurat, no GPU required |
| **RTSP source** | Sub-stream via MediaMTX | `rtsp://mediamtx:8554/cam-{id}` |
| **Reset counter** | 00:00 WITA setiap hari | Sesi baru per tanggal |
| **Min RAM AI service** | 1.5 GB | YOLOv8n + 16 stream buffer |
| **Min CPU AI service** | 2 vCPU | ~0.1–0.3 core/kamera aktif |

---

### Sprint 37 — RTSP Live Status Detection (Task 14A)

> **Branch:** `feature/rtsp-live-probe`
> **Prasyarat:** Sprint 31 selesai (koordinat float + hardening)
> **Estimasi:** 2–3 hari kerja
> **Status: ✅ Selesai** (2026-08-12)

| Jobdesk | Sub-Agent | Task | Deliverable | Status |
|---------|-----------|------|-------------|--------|
| **Backend** | Agent-BE | **14A.1** Migration + RTSP Probe Engine | Migration `000019_stream_live_status`. Model `CCTV` + `StreamLive bool`, `StreamCheckedAt *time.Time`. Fungsi `ProbeRTSPHost`/`ProbeRTSPChannel` via TCP dial + RTSP OPTIONS. Goroutine `RunRTSPProber(db)`: probe semua CCTV `stream_enabled=true` tiap 2 menit, update `stream_live` + `stream_checked_at`, broadcast `CCTV_LIVE_STATUS_UPDATED` via WebSocket. Endpoint `GET /api/streams/live-status` (admin) — return `summary` (online/offline/unchecked) + `data` seluruh CCTV dengan `stream_live` + `stream_checked_at`. | ✅ Selesai |
| **Frontend** | Agent-FE | **14A.2** Map Badge + Monitor Border | `CCTVMap.tsx`: tambah field `stream_live` ke interface `CCTVMapItem`. Tambah badge kecil di pojok kanan atas marker SVG: 🟢 (online) / 🔴 (offline) / ⚪ (belum dicek). Listen WebSocket event `CCTV_LIVE_STATUS_UPDATED` → update badge tanpa reload. Status bar peta: `"X Online · Y Offline · Z Belum dicek"`. Di `/monitor` (`StreamGridView.tsx`): border hijau tipis = online, border merah = offline, border abu-abu = belum dicek. Tooltip per sel: `"Terakhir dicek: HH:mm WIB"`. | ✅ Selesai |
| **Backend** | Agent-BE | **14A.3** Fix StreamHealthMonitor | Hapus kondisi guardian `if _, wasUp := streamLastReady[path]; !wasUp { return }` di `evaluateStreamState()`. Ganti logika: jika `stream_live=false` setelah probe, langsung set `streamDownSince` tanpa perlu menunggu ada viewer dulu. Update unit test `stream_health_test.go`. | ✅ Selesai |

**✅ Selesai — Ringkasan Sprint 37:** RTSP Live Status Detection (14A.1, 14A.2, 14A.3) selesai — **Fase 14 kini 1/3 🚧 Berjalan**.
- **Backend**: migration `000019_stream_live_status` (kolom `stream_live` default `false` + `stream_checked_at` nullable di `cctvs`); model `CCTV` tambah `StreamLive bool` & `StreamCheckedAt *time.Time`; `utils/rtsp_probe.go` (`ProbeRTSPHost` TCP dial 3s, `ProbeRTSPChannel` RTSP OPTIONS 5s); `handlers/stream_probe.go` (`RunRTSPProber` startup + tiap 2 menit, update `stream_live`/`stream_checked_at`, broadcast `CCTV_LIVE_STATUS_UPDATED` saat berubah, `GetLiveStatus` → `summary` + `data`); rute `GET /api/streams/live-status` admin-only.
- **Frontend**: badge live di marker `CCTVMap.tsx` (🟢/🔴/⚪), status bar peta `"X Online · Y Offline · Z Belum dicek"`, border/ring per sel di `/monitor`, tooltip `"Terakhir dicek: HH:mm WIB"`, update real-time via WebSocket tanpa reload.
- **Dokumen**: `docs/manual-test-rtsp-live-status.md` (baru — S1–S9 sesuai format manual test lain); `docs/ROADMAP.md` Fase 14 progress **0/3 → 1/3 🚧 Berjalan** & Sprint 37 ✅ Selesai; `docs/TESTING.md` matriks `handlers/stream_probe_test.go` + `stream_health_test.go` diperluas (total 62 → 63).

Detail: `backend/changelog.md` · `frontend/changelog.md`.

#### Verifikasi Sprint 37

```bash
cd backend && go build ./... && go test ./... -count=1 -run "TestRTSP|TestStream" && go vet ./...
cd frontend && npm run lint && npm run build
```

> **Acceptance Criteria Sprint 37:**
> - Kamera yang mati (RTSP tidak bisa diakses) → badge merah di peta dalam ≤ 2 menit tanpa perlu ada yang membuka stream-nya
> - Kamera hidup kembali → badge berubah hijau dalam ≤ 2 menit berikutnya
> - `GET /api/streams/live-status` mereturn `stream_live: false` untuk kamera yang offline
> - Counter status bar peta akurat (X Online, Y Offline)
> - Border merah di /monitor untuk kamera offline

> [!IMPORTANT]
> **Addendum Kebijakan Metro-E vs GSM — Probe Strategy Final (dikonfirmasi 2026-08-13):**
> Implementasi `RunRTSPProber` telah di-patch agar tidak menyentuh kamera GSM sama sekali:
>
> | Tipe | Identifikasi | Probe Method | Data GSM |
> |------|-------------|-------------|----------|
> | **Metro-E** | `ip_camera` prefix `10.20.105.` | ICMP Ping langsung ke IP kamera, tiap 2 menit | ❌ Tidak ada |
> | **GSM** | IP selain `10.20.105.x` | Cek MediaMTX path list (lokal VPS) + last-active tracking | ✅ Nol |
>
> **Strategi GSM — Zero Data Consumption:**
> ```
> JANGAN probe ke NVR/kamera GSM dari backend.
> Gunakan data yang sudah tersedia di VPS:
>
> 1. MediaMTX path list (GET http://mediamtx:9997/v3/paths/list) — lokal, gratis
>    → path Ready=true → kamera ONLINE (ada viewer aktif)
>    → path tidak ada / Ready=false → tidak bisa disimpulkan
>
> 2. Last-active tracking (field baru: stream_last_active_at)
>    → diupdate saat stream berhasil dibuka (event Ready=true dari MediaMTX)
>    → < 24 jam: "Pernah Online Hari Ini" 🟡
>    → null / > 24 jam: "Tidak Diketahui" ⚪
>    → pernah aktif lalu gagal: "Offline" 🔴
> ```
>
> **Badge UI untuk kamera GSM:**
> 🟢 Online — path MediaMTX Ready=true saat ini
> 🟡 Pernah aktif — `stream_last_active_at` < 24 jam, tidak ada viewer sekarang
> ⚪ Tidak diketahui — belum pernah dibuka atau > 24 jam
> 🔴 Offline — stream pernah aktif tapi sekarang error
>
> **Metro-E latensi:** `<50ms` 🟢 Good | `50–100ms` 🟡 Lagging | `>100ms` 🔴 Critical | Timeout ⚫ Offline
>
> **Butuh patch:** lihat **Sprint 37-PATCH** di bawah. Referensi lengkap: `docs/AGENTS.md` § 11.2.

#### Sprint 37-PATCH — Update Probe Strategy Metro-E vs GSM

> **Branch:** `bugfix/rtsp-probe-metro-gsm-strategy`
> **Prasyarat:** Sprint 37 sudah selesai (file `stream_probe.go` sudah ada)
> **Estimasi:** 1 hari kerja
> **Status: ✅ Selesai** (2026-08-13, backend `0.63.0`)

| Task | Deliverable |
|------|-------------|
| **Migration baru** | `000022_stream_ping_fields.up.sql`: `ALTER TABLE cctvs ADD COLUMN ping_latency_ms INTEGER, ADD COLUMN stream_last_active_at TIMESTAMPTZ`. Down: DROP kedua kolom. |
| **Update model** | `models/cctv.go`: tambah `PingLatencyMs *int \`gorm:"column:ping_latency_ms" json:"ping_latency_ms,omitempty"\`` dan `StreamLastActiveAt *time.Time \`gorm:"column:stream_last_active_at" json:"stream_last_active_at,omitempty"\``. |
| **Patch `utils/rtsp_probe.go`** | Tambah `ProbeICMP(ip string, timeout time.Duration) (latencyMs int64, ok bool)` via `net.DialTimeout("ip4:icmp", ...)` + ICMP echo request. Tambah `isMetroE(cctv models.CCTV) bool` — `strings.HasPrefix(cctv.IPCamera, "10.20.105.")`. |
| **Patch `handlers/stream_probe.go`** | Update `runRTSPProbeCycle()`: pisahkan Metro-E vs GSM. **Metro-E:** `isMetroE=true` → `ProbeICMP(cctv.IPCamera)` → update `stream_live` + `ping_latency_ms` + `stream_checked_at`. **GSM:** `isMetroE=false` → cek `streamClient.ListPaths()` → jika path Ready=true: update `stream_live=true` + `stream_last_active_at=now`; jika tidak: update `stream_live=false` (tanpa sentuh kamera). |
| **Update `GetLiveStatus`** | Response tambah field `ping_latency_ms`, `stream_last_active_at`, `network_type` (`"metro-e"` atau `"gsm"`). |
| **Update `CCTVMap.tsx`** | Interface `CCTVMapItem`: tambah `ping_latency_ms?: number`, `stream_last_active_at?: string`, `network_type?: string`. Marker badge GSM: 4 state 🟢/🟡/⚪/🔴. Popup Metro-E: tampilkan latensi ping (`"Ping: 12ms"`). |
| **Test** | Update `utils/rtsp_probe_test.go` + `handlers/stream_probe_test.go` dengan test case Metro-E vs GSM. |


---

### Sprint 38 — AI Microservice & Person Counting (Task 14B)

> **Branch:** `feature/ai-detection-service`
> **Status: ❌ DIBATALKAN** — Digantikan oleh Sprint 40 (AI Takedown)

> [!CAUTION]
> **Sprint ini dibatalkan pada 2026-08-13.** Kode yang sempat diimplementasikan (handlers/ai.go, routes/ai.go, models/ai_detection.go, service `ai/`, migration 000020 & 000021) telah dihapus via **Sprint 40**. Sistem kembali ke pure CCTV management tanpa AI detection.


| Jobdesk | Sub-Agent | Task | Deliverable | Status |
|---------|-----------|------|-------------|--------|
| **AI Service** | Agent-AI | **14B.1** Setup AI Microservice | Buat folder `ai/` di root repo. `Dockerfile` (python:3.11-slim). `requirements.txt`: `fastapi==0.111`, `uvicorn`, `ultralytics==8.2`, `opencv-python-headless`, `numpy`, `httpx`. `main.py`: FastAPI app dengan router `session` dan `health`. `core/detector.py`: load YOLOv8n, fungsi `detect_persons(frame: np.ndarray) → List[Detection]` — filter class=0, confidence≥0.5. `core/tracker.py`: ByteTrack wrapper (gunakan `supervision` library dari Roboflow). `core/session_manager.py`: class `SessionManager` — `activate(cctv_id, rtsp_url)`, `deactivate(cctv_id)`, `get_active()`. RTSP pull via OpenCV `cv2.VideoCapture`. Update `docker-compose.yml`: tambah service `sigap_ai` (network `sigap_network`, port 8010 internal, depends_on: `mediamtx`). | ✅ Selesai |
| **AI Service** | Agent-AI | **14B.2** Person Counting + API Endpoints | `core/person_counter.py`: class `PersonCounter` — method `process(tracking_ids: Set[int], cctv_id: str, db_session_date: date)` → update `seen_ids` set, hitung `new_unique = seen_ids - previously_seen`. Endpoint `POST /sessions`: `{ "cctv_id": "uuid", "rtsp_url": "rtsp://..." }` → aktifkan stream + tracking. Endpoint `DELETE /sessions/{cctv_id}` → nonaktifkan. Endpoint `GET /sessions` → list aktif + count per kamera. Endpoint `GET /counts/{cctv_id}` → `{ "unique_today": 42, "current_in_frame": 2 }`. Kirim deteksi ke backend Go via `POST /api/ai/detection-event` tiap ada deteksi baru. Reset `seen_ids` otomatis tiap 00:00 WITA. | ✅ Selesai |
| **Backend** | Agent-BE | **14B.3** Backend AI Integration + DB | Migration `000019_ai_detection_sessions` + `000020_ai_detection_events`. Handler `handlers/ai.go` (file baru): `POST /api/ai/detection-event` (internal only — auth via shared secret header `X-AI-Secret`). Terima payload `{ cctv_id, person_count, unique_count, tracking_ids, confidence_avg }`. Update `ai_detection_sessions` (upsert per `cctv_id + session_date`). Insert ke `ai_detection_events`. Broadcast WebSocket `AI_DETECTION_UPDATE { cctv_id, bboxes: [...], unique_today, current_in_frame }`. Endpoint `GET /api/ai/counts` (admin) → ringkasan semua kamera hari ini. Endpoint `GET /api/ai/counts/:cctv_id` → detail per kamera. Route di `routes/ai.go` (file baru). Register goroutine `RunAIDailyReset(db)` di `main.go`: tiap 00:00 WITA, trigger `POST /sessions/reset-all` ke AI service. | ✅ Selesai |

**✅ Selesai — Ringkasan Sprint 38:** AI microservice & person counting (Task 14B) selesai — **Fase 14 kini 2/3 🚧 Berjalan**.
- **AI Service (`ai/`, baru)**: `Dockerfile` (python:3.11-slim + lib OpenCV, pre-download `yolov8n.pt`), `requirements.txt` (fastapi 0.111.0, uvicorn 0.29.0, ultralytics 8.2.0, opencv-python-headless, numpy, httpx, supervision 0.20.0), `core/` — `detector.py` (YOLOv8n class 0, conf ≥ 0.50), `tracker.py` (ByteTrack via supervision), `person_counter.py` (unique person/hari, thread-safe, auto-reset 00:00 WITA), `session_manager.py` (`CameraSession` async — pull RTSP `rtsp://mediamtx:8554/cam-{cctv_id}`, ~1 fps, bbox ternormalisasi 0–1, POST ke backend + `X-AI-Secret`); `routers/session.py` & `health.py`; `main.py` (FastAPI + daily reset); `README.md`.
- **Docker Compose**: service `sigap_ai` (network `sigap_network`, depends_on `mediamtx`+`api`, env `BACKEND_URL`/`MEDIAMTX_RTSP_URL`/`AI_SECRET`, limits 2 CPU/2048M, port 8010 internal TIDAK dipublish).
- **Backend (`0.61.0`)**: migrasi `000020_ai_detection_sessions` & `000021_ai_detection_events`; model `AIDetectionSession`/`AIDetectionEvent`; `handlers/ai.go` — `ReceiveDetectionEvent` (internal X-AI-Secret, upsert sesi + event, broadcast `AI_DETECTION_UPDATE`), `GetAICounts`/`GetAICountByCCTV` (admin), proxy session (`POST/DELETE/GET /api/ai/sessions` ke `http://sigap_ai:8010`); `routes/ai.go`; registrasi `main.go`; env `AI_SECRET` & `AI_SERVICE_URL`; `swag init`.
- **Test**: `handlers/ai_test.go` (baru — 7 kasus: validasi secret, UUID, upsert sesi baru/existing, counts).
- **Dokumen**: `docs/manual-test-ai.md` (baru — S1–S9); `docs/TESTING.md` matriks AI.

Detail: `backend/changelog.md` [0.61.0] · `docs/changelog.md` [1.28.43].

#### Verifikasi Sprint 38

```bash
# AI Service
cd ai && docker build -t sigap_ai . && docker run --rm sigap_ai python -c "from core.detector import Detector; print('OK')"

# Backend
cd backend && go build ./... && go test ./... -count=1 -run "TestAI" && go vet ./...

# Integration test
docker compose up -d sigap_ai sigap_api
curl -X POST http://localhost:8010/sessions -d '{"cctv_id":"test","rtsp_url":"rtsp://..."}'
curl http://localhost:8010/counts/test
```

> **Acceptance Criteria Sprint 38:**
> - AI service berhasil start tanpa error via `docker compose up sigap_ai`
> - `POST /sessions` mengaktifkan RTSP pull dan mulai inferensi
> - Person terdeteksi → `unique_today` bertambah, orang yang sama tidak dihitung dua kali
> - `DELETE /sessions/{id}` menghentikan RTSP pull dan membebaskan CPU
> - Backend menerima event deteksi dan menyimpan ke DB
> - Counter reset ke 0 pada 00:00 WITA

---

### Sprint 39 — Frontend AI Overlay & E2E Integration (Task 14C)

> **Branch:** `feature/ai-frontend-overlay`
> **Status: ❌ DIBATALKAN** — Digantikan oleh Sprint 40 (AI Takedown)

> [!CAUTION]
> **Sprint ini dibatalkan pada 2026-08-13.** Kode frontend yang sempat diimplementasikan (canvas overlay WHEPPlayer, tombol AI StreamGridView, halaman /ai-analytics) telah dihapus via **Sprint 40**.


| Jobdesk | Sub-Agent | Task | Deliverable | Status |
|---------|-----------|------|-------------|--------|
| **Frontend** | Agent-FE | **14C.1** Canvas Overlay Bounding Box | Di `StreamGridView.tsx` atau `WHEPPlayer.tsx`: tambah `<canvas>` yang overlay tepat di atas `<video>` (CSS `position: absolute, top: 0, left: 0, pointer-events: none`). Listen WebSocket event `AI_DETECTION_UPDATE { cctv_id, bboxes }`. Saat event diterima: `ctx.clearRect(...)` → draw box per `bbox` (warna hijau `#22C55E`, line width 2). Label per box: `"Person"`. Box otomatis fade dalam 1.5 detik jika tidak ada update baru. Aktifkan canvas hanya jika AI session aktif untuk kamera tersebut. | ✅ Selesai |
| **Frontend** | Agent-FE | **14C.2** AI Session Control + Counter UI | Di `StreamGridView.tsx`: tambah tombol toggle "🤖 AI" di header grid. Klik ON: kirim `POST /api/ai/sessions` untuk setiap kamera yang aktif di grid. Klik OFF: kirim `DELETE /api/ai/sessions/{id}` untuk semua. Per sel kamera: tampilkan badge kecil `"👥 42 orang hari ini"` di pojok kiri bawah video. Badge update via WebSocket tanpa reload. Saat ganti layout grid (2×2 → 3×3): otomatis hentikan session kamera yang tidak lagi tampil, aktifkan session kamera baru. State AI on/off tersimpan di localStorage per sesi browser. | ✅ Selesai |
| **Frontend** | Agent-FE | **14C.3** Dashboard AI Analytics | Halaman `/ai-analytics` (admin only) — kartu ringkasan (total orang unik hari ini, jumlah kamera dipantau, kamera dengan hitungan > 0), tabel per kamera (nama, kode CCTV, `unique_today`, terakhir diperbarui), grafik tren 7 hari per kamera (`GET /api/ai/counts/:id/trend?days=7`, SVG mandiri tanpa library chart eksternal), dropdown pilih kamera; item menu sidebar (admin only) `{ name: "AI Analytics", icon: BrainCircuit, href: "/ai-analytics" }` di grup "Pelaporan & Analisis"; proteksi route di `frontend/src/proxy.ts` (`ADMIN_PATHS` + `config.matcher`). | ✅ Selesai |
| **Backend** | Agent-BE | **14C.3b** Endpoint Tren AI | `GET /api/ai/counts/:id/trend` (admin-only) di `handlers/ai.go` (`GetAICountTrend`) + rute di `routes/ai.go` + Swagger; unit test `TestGetAICountTrend` & `TestGetAICountTrend_InvalidID` di `handlers/ai_test.go`. | ✅ Selesai |

**✅ Selesai — Ringkasan Sprint 39:** Frontend AI overlay & E2E integration (Task 14C) selesai — **Fase 14 kini 3/3 ✅ Selesai**.
- **Frontend**: halaman admin `/ai-analytics` (baru) — ringkasan hari ini + tren 7 hari per kamera dengan grafik garis SVG mandiri; menu sidebar "AI Analytics" (`BrainCircuit`) di grup "Pelaporan & Analisis"; guard proxy `/ai-analytics`; overlay bounding box hijau `#22C55E` di atas video + counter "👥 X orang hari ini" per sel + toggle "🤖 AI" di `/monitor` (14C.1/14C.2, agent paralel).
- **Backend (`0.62.0`)**: endpoint tren `GET /api/ai/counts/:id/trend` (admin-only) — validasi UUID → 400, `days` default 7 clamp 1–90, query `ai_detection_sessions` ascending by date, respons `{date, unique_count}`; rute + Swagger; unit test tren (sukses + invalid UUID).
- **Dokumen**: `docs/manual-test-ai-analytics.md` (baru — S1–S7); `docs/TESTING.md` matriks `handlers/ai_test.go` diperluas (Task 14C.3); ROADMAP Fase 14 progress **2/3 → 3/3 ✅ Selesai** & Sprint 39 ✅ Selesai.

Detail: `backend/changelog.md` [0.62.0] · `docs/changelog.md` [1.28.44].

#### Verifikasi Sprint 39

```bash
cd frontend && npm run lint && npm run build && npx tsc --noEmit
cd backend && go build ./... && go test ./... -count=1 -run "TestGetAICountTrend|TestAI" && go vet ./... && golangci-lint run ./...
```

> **Acceptance Criteria Sprint 39:**
> - Bounding box hijau muncul real-time di atas video saat person terdeteksi
> - Box fade setelah 1.5 detik jika tidak ada person di frame
> - Counter "X orang hari ini" per sel kamera akurat dan tidak double-count
> - Ganti layout grid → AI session otomatis menyesuaikan (aktif hanya untuk kamera yang tampil)
> - Halaman `/ai-analytics` menampilkan data hitungan orang per kamera per hari + tren 7 hari (admin only; teknisi → redirect dashboard)
> - Sprint lengkap tidak menyebabkan error di halaman lain

---

### Ringkasan Sprint Fase 14

```
═══════════════════════════════════════════════════════════════
FASE 14 — CCTV Intelligence (REVISI 2026-08-13)
───────────────────────────────────────────────────────────────
Sprint 37: RTSP Live Status Detection (14A)     ✅ Selesai
Sprint 37-PATCH: Metro-E vs GSM Probe Strategy  ✅ Selesai
Sprint 38: AI Microservice & Person Counting    ❌ DIBATALKAN
Sprint 39: Frontend AI Overlay & E2E            ❌ DIBATALKAN
Sprint 40: AI Feature Takedown (rollback)       ✅ Selesai
───────────────────────────────────────────────────────────────
Keputusan: Fitur AI detection dihapus seluruhnya (2026-08-13)
Sistem: Pure CCTV Management — tanpa AI inference
───────────────────────────────────────────────────────────────
Requirement Server (revised, tanpa AI):
  CPU   : 2 vCPU (cukup untuk app + MediaMTX)
  RAM   : 4 GB total
  Port  : 554 (RTSP NVR) harus bisa dicapai dari VPS
═══════════════════════════════════════════════════════════════
```

---

### Sprint 40 — AI Feature Takedown

> **Branch:** `bugfix/remove-ai-detection`
> **Prasyarat:** Sprint 38 & 39 sudah ada di codebase (sudah diimplementasikan)
> **Estimasi:** 1–2 hari kerja
> **Status: ✅ Selesai (2026-08-13)**

Hapus **seluruh** artefak AI detection dari codebase. Sistem kembali ke 7 Docker service (tanpa `sigap_ai`).

| Komponen | Aksi | Detail |
|----------|------|--------|
| **`ai/` folder** | 🗑️ Hapus seluruh folder | Dockerfile, main.py, requirements.txt, core/, routers/ |
| **`backend/handlers/ai.go`** | 🗑️ Hapus | Handler ReceiveDetectionEvent, GetAICounts, ProxySession, dll. |
| **`backend/routes/ai.go`** | 🗑️ Hapus | Registrasi route `/ai/*` |
| **`backend/models/ai_detection.go`** | 🗑️ Hapus | Model AIDetectionSession & AIDetectionEvent |
| **`backend/main.go`** | ✏️ Edit | Hapus `routes.SetupAIRoutes(api)` + log baris AI |
| **Migration `000023`** | ➕ Buat | `000023_remove_ai_detection.up.sql`: DROP TABLE ai_detection_events, ai_detection_sessions |
| **`docker-compose.yml`** | ✏️ Edit | Hapus blok service `sigap_ai` (baris 276–305) |
| **`frontend/.../ai-analytics/page.tsx`** | 🗑️ Hapus | Halaman analytics AI (465 baris) |
| **`frontend/.../WHEPPlayer.tsx`** | ✏️ Edit | Hapus: canvas overlay, drawBBoxes, aiEnabled prop, WS AI listener |
| **`frontend/.../StreamGridView.tsx`** | ✏️ Edit | Hapus: tombol 🤖 AI, aiCounts, badge counter, WS AI subscription |
| **`frontend/.../DashboardLayout.tsx`** | ✏️ Edit | Hapus menu item "AI Analytics" jika ada |
| **`backend/.env.example`** | ✏️ Edit | Hapus `AI_SECRET`, `AI_SERVICE_URL` |
| **`docs/AGENTS.md`** | ✏️ Edit | Hapus referensi AI Detection dari § 5 & § 7 |
| **`docs/manual-test-ai.md`** | 🗑️ Hapus (jika ada) | Dokumen test AI tidak relevan |
| **`docs/manual-test-ai-analytics.md`** | 🗑️ Hapus (jika ada) | Dokumen test AI analytics tidak relevan |

#### Verifikasi Sprint 40

```bash
# Backend — tidak boleh ada referensi ke AI routes/handlers
cd backend
grep -r "SetupAIRoutes\|ai_detection\|AIDetection\|sigap_ai\|X-AI-Secret" . --include="*.go"
# Expected: no output

go build ./...        # wajib 0 error
go test ./... -count=1
go vet ./...

# Frontend — tidak boleh ada referensi ke AI
cd ../frontend
grep -r "aiEnabled\|aiCounts\|AI_DETECTION_UPDATE\|ai-analytics\|drawBBoxes" src/ --include="*.tsx"
# Expected: no output

npm run lint
npm run build
npx tsc --noEmit

# Docker
docker compose config     # validasi YAML
grep "sigap_ai" docker-compose.yml
# Expected: no output
```

> **Acceptance Criteria Sprint 40:**
> - [x] `docker ps` setelah `docker compose up -d` hanya menampilkan 7 container (tanpa sigap_ai)
> - [x] `GET /api/ai/counts` → 404 Not Found
> - [x] `GET /api/ai/sessions` → 404 Not Found
> - [x] `/ai-analytics` → 404 atau redirect (halaman tidak ada)
> - [x] `/monitor` → tidak ada tombol 🤖 AI, tidak ada badge "👥 X orang"
> - [x] Video player di `/monitor` tidak punya canvas overlay
> - [x] `go build ./...` → 0 error, 0 warning
> - [x] `npm run build` → sukses
> - [x] Tabel `ai_detection_sessions` & `ai_detection_events` → DROP via migration 000023


## Pemetaan 35 Kamera (Referensi Konfigurasi Streaming)

Semua kamera: `rtsp://admin:***@<NVR_IP>:554/Streaming/channels/{ChannelID}`

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
SELESAI (Fase 9–11, Sprint 20–28)
───────────────────────────────────────────────────────────────
Sprint 20: Fondasi Streaming (9.1 + 9.2)  ✅ ........  2 hari
Sprint 21: Backend API Stream (9.3)  ✅ ..............  2–3 hari
Sprint 22: Frontend Player + UI (9.4–9.6)  ✅ .......  3–4 hari
Sprint 23: Integrasi E2E + Manual Test  ✅ ...........  1–2 hari
Sprint 24: Snapshot & Timeline (10.1 + 10.2)  ✅ ....  3–4 hari
Sprint 25: Monitor & Stream Health (10.3 + 10.4)  ✅   3–4 hari
Sprint 26: Inventaris Suku Cadang (11.1)  ✅ ........  3–4 hari
Sprint 27: Email & Report Builder (11.2 + 11.3)  ✅ .  4–5 hari
Sprint 28: Zona & SLA (11.4 + 11.5)  ✅ .............  4–5 hari
───────────────────────────────────────────────────────────────

SELESAI (Fase 12)
───────────────────────────────────────────────────────────────
Sprint 29: API Key & Webhook (12.1 + 12.3)  ✅ ......  3–4 hari
Task 12.4: Mobile App (React Native)  ✅ ............  10–15 hari
Sprint 30: Multi-Site (12.2)  ✅ ...................  4–5 hari
───────────────────────────────────────────────────────────────

SELESAI — PERBAIKAN FITUR EXISTING (Fase 13)
───────────────────────────────────────────────────────────────
Sprint 31: Keamanan & Integritas Data (P1–P5)  ✅ ..  3–4 hari
Sprint 32: Notifikasi & Email (P6–P7)  ✅ ..........  2–3 hari
Sprint 33: Kecerdasan Data & Eskalasi (P8,P9,P11) ✅  3–4 hari
Sprint 34: Infrastruktur & Integrasi (P10,P14,P16) ✅ 2–3 hari
Sprint 35: Performa & Checklist (P12–P13)  ✅ ......  4–5 hari
Sprint 36: Polish Frontend UX (P15,P17,P18)  ✅ ....  2–3 hari
───────────────────────────────────────────────────────────────

CCTV INTELLIGENCE (Fase 14 — Live Status ✅ + AI Takedown ✅)
──────────────────────────────────────────────────────────────
Sprint 37: RTSP Live Status Detection (14A)  ✅ ....  2–3 hari
Sprint 38: AI Microservice & Person Counting (14B) ❌ DIBATALKAN
Sprint 39: Frontend AI Overlay & E2E (14C)  ❌ DIBATALKAN
Sprint 40: AI Feature Takedown (rollback)  ✅ Selesai ..  1–2 hari
──────────────────────────────────────────────────────────────
Total Fase 13: 16–22 hari kerja
Total Fase 14: Live Status (Sprint 37) ✅ + Metro-E/GSM Patch ✅ + AI Takedown (Sprint 40) ✅ — AI detection dihapus (2026-08-13)
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

---

## Fase 15 — Branding & Dokumentasi

### Sprint 41 — README & Branding Overhaul

> **Branch:** `fix/readme-branding-update`
> **Estimasi:** 0.5–1 hari kerja
> **Status: ✅ Selesai**

Nama resmi, README, metadata aplikasi, halaman login, dan manifest telah diselaraskan dengan kondisi sistem terkini.

#### Temuan yang Telah Diselesaikan

| Lokasi | Kondisi Sebelum | Kondisi Benar |
|--------|-----------|------------|
| `README.md` baris 1 | Kepanjangan SIGAP belum konsisten | `Sistem Informasi Gangguan & Aset Pusat` |
| `README.md` | Deskripsi dan fitur belum lengkap | Sudah diperbarui sesuai kondisi existing |
| `frontend/src/app/layout.tsx` baris 22 | Kepanjangan SIGAP belum konsisten | `Sistem Informasi Gangguan & Aset Pusat` |
| `frontend/src/app/login/page.tsx` baris 114 | Sub-judul `Sistem internal pengelolaan CCTV kawasan IKN.` | `Sistem Informasi Gangguan & Aset Pusat — Kawasan IKN Nusantara` |

#### Scope Perubahan

| Komponen | File | Aksi |
|----------|------|------|
| **README.md** | `README.md` | ✅ Judul, logo OIKN, nama resmi, fitur, struktur proyek, dan arsitektur 7 service diperbarui |
| **App title** | `frontend/src/app/layout.tsx` | ✅ Metadata memakai `"SIGAP - Sistem Informasi Gangguan & Aset Pusat"` |
| **Login subtitle** | `frontend/src/app/login/page.tsx` | ✅ Subtitle memakai nama resmi SIGAP |
| **Manifest** | `frontend/public/manifest.json` | ✅ Nama dan deskripsi aplikasi telah diselaraskan |

#### Spesifikasi README Baru

**Header:**
```markdown
<div align="center">
  <img src="docs/assets/logo-oikn.png" alt="Logo OIKN" width="120" />
  
  # SISTEM INFORMASI GANGGUAN & ASET PUSAT
  ## **SIGAP**
  
  *Platform manajemen CCTV terpadu Kawasan Ibu Kota Nusantara*
  
  [badges...]
</div>
```

**Konten yang harus diperbarui:**
- Fitur: tambah Live Streaming RTSP→WebRTC, Live Status Detection Metro-E, Peta Sebaran CCTV, Inventaris Suku Cadang, SLA, Zona, API Key & Webhook, Multi-Site, Jadwal Pemeliharaan, PWA/Push Notification
- Tech Stack: tambah MediaMTX, Redis, Streaming WebRTC
- Arsitektur: 7 service (db, redis, api, frontend, nginx, backup, mediamtx) — tanpa sigap_ai
- Struktur proyek: sesuaikan dengan folder yang benar-benar ada saat ini
- API Endpoints: update ke kondisi terkini (tambah endpoint streaming, live-status, suku-cadang, zona, SLA, dll)
- ERD: update model (tambah field streaming, zone_id, sla_policy, dll)

**Logo OIKN:**
Gunakan URL publik logo OIKN Nusantara dari internet (jangan hardcode file lokal):
`https://upload.wikimedia.org/wikipedia/commons/thumb/...` atau CDN resmi OIKN.
Atau simpan sebagai `docs/assets/logo-oikn.png` jika ingin bundled.



