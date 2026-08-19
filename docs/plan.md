# Dokumen Perencanaan Proyek SIGAP

## 1. Identitas Sistem
- **Nama Proyek:** SIGAP
- **Kepanjangan:** Sistem Informasi Gangguan & Aset Pusat
- **Tujuan Utama:** Mengelola, memantau, dan mencatat riwayat pemeliharaan aset CCTV kawasan IKN secara terpusat.
- **Arsitektur Repositori:** Monorepo (menggabungkan Frontend dan Backend dalam satu wadah).

## 2. Tech Stack (Tumpukan Teknologi)
- **Backend (API & Logika Server):** Go (Golang) menggunakan framework Fiber (Layered Architecture).
- **Frontend (Antarmuka Pengguna):** Next.js (App Router) menggunakan direktori `src/`.
- **Styling UI:** Tailwind CSS.
- **Database (Rencana):** PostgreSQL.

## 3. Ekosistem & Alat Pendukung
- **AI Code Assistant:** Codex.
- **Version Control:** Git & GitHub.
- **Automation / Task Runner:** `Makefile` di root direktori untuk mengeksekusi perintah terminal secara cepat (misal: menjalankan server BE dan FE secara bersamaan).

## 4. Strategi Dokumentasi & Prompting AI
Proyek ini menggunakan strategi distribusi file dokumentasi untuk memisahkan konteks antara pekerjaan Frontend dan Backend, serta menjaga fokus Codex AI:

- **Aturan AI (`AGENTS.md`):**
  - `/docs/AGENTS.md` -> Aturan kerja agent tingkat monorepo.
  - `/backend/agent.md` -> Berisi aturan spesifik untuk Go Fiber, struktur database, dan logika backend.
  - `/frontend/agent.md` -> Berisi aturan spesifik untuk Next.js, komponen UI, dan Tailwind CSS.
- **Akses aplikasi (`ACCESS.md`):**
  - `/docs/ACCESS.md` -> Satu-satunya entrypoint Docker `https://localhost`, routing Nginx,
    cookie httpOnly, CSRF, WebSocket, kamera, dan Swagger.
- **Buku Harian Proyek (`changelog.md`):**
  - `/docs/changelog.md` (Utama) -> Mencatat pencapaian besar ekosistem monorepo.
  - `/backend/changelog.md` -> Mencatat pembaruan teknis terkait API dan database.
  - `/frontend/changelog.md` -> Mencatat pembaruan teknis terkait antarmuka dan komponen UI.

## 5. Peta Struktur Direktori Standar
```text
sigap-monorepo/
â”œâ”€â”€ Makefile
â”œâ”€â”€ README.md
â”œâ”€â”€ .gitignore
â”‚
â”œâ”€â”€ docs/
â”‚   â”œâ”€â”€ AGENTS.md
â”‚   â”œâ”€â”€ ACCESS.md
â”‚   â”œâ”€â”€ changelog.md
â”‚   â”œâ”€â”€ plan.md
â”‚   â”œâ”€â”€ ROADMAP.md
â”‚   â”œâ”€â”€ SECURITY-AUDIT.md
â”‚   â”œâ”€â”€ TESTING.md
â”‚   â”œâ”€â”€ PRODUCT.md
â”‚   â””â”€â”€ DESIGN.md
â”‚
â”œâ”€â”€ backend/
â”‚   â”œâ”€â”€ agent.md
â”‚   â”œâ”€â”€ changelog.md
â”‚   â”œâ”€â”€ main.go
â”‚   â”œâ”€â”€ go.mod
â”‚   â”œâ”€â”€ go.sum
â”‚   â”œâ”€â”€ config/
â”‚   â”œâ”€â”€ routes/
â”‚   â”œâ”€â”€ handlers/
â”‚   â””â”€â”€ models/
â”‚
â””â”€â”€ frontend/
    â”œâ”€â”€ agent.md
    â”œâ”€â”€ changelog.md
    â”œâ”€â”€ package.json
    â”œâ”€â”€ tailwind.config.ts
    â””â”€â”€ src/
        â”œâ”€â”€ app/
        â”œâ”€â”€ components/
        â””â”€â”€ lib/

## 6. Rencana Pengembangan Aktif (per 2026-08-06)

> Sumber kebenaran status & urutan pengerjaan: `docs/ROADMAP.md` (Fase 5â€“8).
> Bagian ini mencatat konteks keputusan agar tidak hilang antar sesi pengembangan.

### 6.1 Konteks Keputusan

**Diskusi 2026-08-04:**
- Sistem **masih pilot/UAT** â€” belum menampung data produksi nyata, sehingga fitur alur
  kerja (Jalur A) didahulukan; pengerasan infrastruktur (sisa Fase 4: backup, versioned
  migration, container hardening, Redis, metrics) menyusul menjelang go-live.
- Prioritas: **Jalur A â€” Alur Logika**, 5 task berurutan 5.1 â†’ 5.5 (lihat `ROADMAP.md` Fase 5).
- Notifikasi: **in-app saja** (bell + daftar notifikasi + toast via WebSocket); email tidak dipakai.
- Status CCTV otomatis: **konservatif** â€” hanya `aktif â†” pemeliharaan`; `rusak`/`nonaktif`
  tetap keputusan manual admin. **Per 2026-08-05 status `rusak` digabung ke `nonaktif`**
  (fungsional identik: keduanya dilindungi sync dan tampil sama "Nonaktif"); enum resmi
  kini `aktif` / `pemeliharaan` / `nonaktif`.

**Brainstorming 2026-08-05:**
- Setelah analisis gap menyeluruh, disepakati **3 fase baru** (Fase 6â€“8) setelah Fase 5 tuntas:
  - **Fase 6 â€” Kecerdasan Operasional**: skor kesehatan CCTV, kinerja teknisi, laporan PDF pengecekan lapangan, ringkasan harian otomatis.
  - **Fase 7 â€” Kesiapan Produksi**: menyerap sisa Fase 4 (4.3â€“4.7) + logging terstruktur. Wajib sebelum go-live.
  - **Fase 8 â€” Pengalaman Lapangan**: PWA, kalender jadwal, push notification, geotag pengecekan.
- Fase 6 dan 7 bisa dikerjakan **paralel** (fitur vs infrastruktur). Jika tim terbatas dan go-live dekat, Fase 7 diprioritaskan.
- Sisa Fase 4 (4.3â€“4.7) resmi **dilebur ke Fase 7** â€” tidak lagi menjadi backlog terpisah.

### 6.2 Ringkasan Sprint

| Sprint | Fase | Task ROADMAP | Cakupan | Status |
|--------|------|--------------|---------|--------|
| 1 | 5 | 5.1 Notifikasi in-app | Model `Notification`, endpoint `/api/notifications`, `WSHub` targeted, `NotificationBell`, hook assign/complete/field-check | âœ… Kode selesai (2026-08-04, branch `feature/notification-in-app`) â€” menunggu uji runtime Docker |
| 2 | 5 | 5.2 Sinkron status CCTV | Helper dalam transaksi complete/field-check; tabel transisi konservatif | âœ… Selesai (2026-08-05, branch `feature/sync-cctv-status`) â€” build/test/vet PASS, uji manual S1â€“S8 di `docs/manual-test-sync-cctv.md` (3 bug ditemukan & ditindaklanjuti: pesan validasi [FE 1.25.11], normalisasi Inggris [BE 0.28.1], guard legacy uppercase [BE 0.28.2], dropdown status [FE 1.25.12]) + penggabungan `rusak`â†’`nonaktif` [BE 0.28.3 / FE 1.25.13] |
| 3 | 5 | 5.3 Re-assign tugas | Perluas endpoint assign ke log `ongoing`, notif dua pihak, tombol "Alihkan" | âœ… Selesai (2026-08-05, branch `feature/reassign-maintenance`) â€” build/test/vet PASS + lint/build PASS + swag init; menunggu uji runtime Docker (teknisi lama dapat notif "dialihkan", teknisi baru dapat notif "tugas baru") |
| 4 | 5 | 5.4 Jadwal berulang | Model `MaintenanceSchedule`, goroutine generator harian, CRUD + halaman admin `/schedules` | âœ… Selesai (2026-08-06, branch `feature/recurring-schedule`) â€” build/test/vet PASS + lint/build PASS + swag init; generator idempotent per hari per jadwal, tanpa teknisi â†’ log `scheduled`, dengan teknisi â†’ `ongoing` + notifikasi |
| 5 | 5 | 5.5 Eskalasi overdue | Kolom `OverdueNotifiedAt`, goroutine saat startup + tiap jam, notif admin/teknisi, reset saat reschedule | âœ… Selesai (2026-08-06, branch `feature/overdue-escalation`) â€” test checker + reset reschedule PASS |
| 6 | 6 | 6.1 Skor Kesehatan CCTV | Skor 0â€“100 dari 5 faktor, endpoint breakdown, badge tabel/peta, goroutine 6 jam, recalculation saat complete/field-check | âœ… Selesai (2026-08-06, branch `feature/health-score`) â€” test kalkulasi + endpoint, backend/frontend build PASS |
| 7 | 6 | 6.2 Dashboard Kinerja Teknisi | Endpoint statistik per teknisi, halaman `/kinerja-teknisi` | âœ… Selesai (2026-08-06, branch `feature/technician-performance`) â€” test struktur/filter/role + backend/frontend build dan lint PASS |
| 8 | 6 | 6.3 Laporan PDF Pengecekan | PDF per-record field-maintenance dengan kop surat IKN | âœ… Selesai (2026-08-06, branch `feature/field-maintenance-pdf`) |
| 9 | 6 | 6.4 Ringkasan Operasional | Goroutine harian, model `DailyDigest`, halaman `/ringkasan` | âœ… Selesai (2026-08-06, branch `feature/daily-digest`) |
| 10 | 7 | 7.1 Migrasi DB Versioned | golang-migrate v4, file SQL terurut, `RunMigrations`, CLI up/down/version, AutoMigrate fase transisi, CI step, `docs/MIGRATION.md` | ✅ Selesai (2026-08-06, branch `feature/versioned-migration`) — build/test/vet PASS + uji live PostgreSQL up/down/up PASS |
| 11 | 7 | 7.2 Redis Rate-Limit & Blacklist | Container Redis 7 Alpine di compose, `config/redis.go`, blacklist → Redis SET (TTL), limiter → Redis INCR/EXPIRE key `limiter:{ip}:{endpoint}`, fallback in-memory, env `REDIS_*`, unit test interface | ✅ Selesai (2026-08-06, branch `feature/redis-integration`) — build/test/vet PASS + uji live Redis 12 cek PASS + `docker compose config` valid |
| 12 | 7 | 7.3 Backup Otomatis & Restore | Skrip `backup.sh`/`restore.sh`, container `backup` (cron), volume `backups_data`, `docs/BACKUP.md` | ✅ Selesai (2026-08-07, branch `feature/backup-restore`) — `bash -n` + `docker compose config` valid; uji fungsional backup/uploads/rotasi & restore lulus |
| 13 | 7 | 7.4 Monitoring & Alerting | Endpoint `/api/metrics` Prometheus (counter/histogram/gauge), middleware `metrics.go`, health diperluas, collector tanpa library pihak ketiga, `docker-compose.override.yml` (Prometheus & Grafana opsional) | ✅ Selesai (2026-08-07, branch `feature/monitoring`) — build/test/vet PASS + `docker compose config` valid; unit test metrik/middleware/health |
| 14 | 7 | 7.5 Container Hardening | Dockerfile non-root, resource limits, read-only rootfs + tmpfs, drop capabilities, Docker secrets (DB_PASSWORD/JWT_SECRET/AES_SECRET_KEY) + fallback env | ✅ Selesai (2026-08-07, branch `feature/production-hardening`) — build/test/vet PASS + `docker compose config` valid |
| 15 | 7 | 7.6 Logging Terstruktur | lumberjack rotasi (100 MB/5 backup/30 hari), `LOG_LEVEL`, log JSON | ✅ Selesai (2026-08-07, branch `feature/production-hardening`) — build/test/vet PASS |
| 16 | 8 | 8.1 Progressive Web App (PWA) | `public/manifest.json` (standalone, warna tema), `public/sw.js` (cache-first shell, network-first API), ikon 192/512, registrasi SW + meta di `layout.tsx`, header cache di `next.config.ts` | ✅ Selesai (2026-08-07, branch `feature/pwa`) — lint & build PASS; manual test `docs/manual-test-pwa.md` |
| 17 | 8 | 8.2 Kalender Jadwal Pemeliharaan | Halaman `/kalender` (admin+teknisi), `CalendarGrid` (grid bulanan, dot status, navigasi), panel detail hari, filter teknisi admin | ✅ Selesai (2026-08-07, branch `feature/calendar-view`) — lint & build PASS; manual test `docs/manual-test-calendar.md` |
| 18 | 8 | 8.3 Notifikasi Push Browser | Model `PushSubscription`+migrasi, `POST /push/subscribe` & `DELETE /push/unsubscribe`, `utils/push.go` (webpush VAPID), integrasi ke `NotifyUser`, SW push/notificationclick, opt-in `PushNotificationOptIn` di `/profile` | ✅ Selesai (2026-08-07, branch `feature/push-notification`) — build/test/vet + lint/build PASS; manual test `docs/manual-test-push-notification.md` |
| 19 | 8 | 8.4 Geotag Pengecekan Lapangan | Kolom `check_latitude`/`check_longitude` + migrasi, validasi backend, GPS wizard + indikator, mini-map admin | ✅ Selesai (2026-08-07, branch `feature/geotag-field-check`) — build/test/vet + lint/build PASS; manual test `docs/manual-test-geotag.md` |
| 20 | 9 | 9.1 Model & Migrasi Streaming + 9.2 Infra MediaMTX | Field `channel_port`/`rtsp_password`(AES-256)/`stream_enabled` + migrasi `000004`; `utils/rtsp.go`; service `mediamtx` di compose + `mediamtx.yml`; nginx `/stream/*` & `/hls/*`; env NVR/MediaMTX | ✅ Selesai (2026-08-07, branch `feature/cctv-streaming-foundation`) — build/test/vet PASS + `docker compose config --quiet` VALID; `backend/changelog.md` [0.43.0] |
| 21 | 9 | 9.3 Backend API & MediaMTX Client | `utils/mediamtx.go` (MediaMTXClient), `handlers/stream.go` + `routes/stream.go` (`/api/streams` admin-only), registrasi `main.go` + goroutine auto-start, Swagger, test | ✅ Selesai (2026-08-07, branch `feature/stream-api`) — build/test/vet/golangci-lint PASS + `swag init`; `backend/changelog.md` [0.44.0] |
| 22 | 9 | 9.4+9.5+9.6 Frontend Player & UI Streaming | `WHEPPlayer.tsx`, `StreamModal.tsx`, `StreamGridView.tsx`, integrasi `CCTVMap` + `sebaran-cctv` Peta/Grid toggle, admin config streaming di `cctv/page.tsx` | ✅ Selesai (2026-08-07, branch `feature/stream-frontend`) — lint & build PASS; `frontend/changelog.md` [1.36.0] |
| 23 | 9 | Integrasi E2E + Manual Test Streaming | `docs/manual-test-streaming.md` (S1–S9); Docker E2E 8 service; update dokumentasi (AGENTS/TESTING/plan) | ✅ Selesai (2026-08-07) — `docs/changelog.md` [1.28.30]; Fase 9 = 6/6 SELESAI |

### 6.3 Rancangan Teknis Sprint 1 (Notifikasi In-App)

**Backend**
- `models/notification.go`: `Notification{UserID, Type, Title, Message, ReferenceID, ReferenceType, IsRead}` + konstanta tipe (`MAINTENANCE_ASSIGNED`, `MAINTENANCE_REASSIGNED`, `MAINTENANCE_UNASSIGNED`, `MAINTENANCE_COMPLETED`, `MAINTENANCE_OVERDUE`, `FIELD_CHECK_ATTENTION`); index komposit `(user_id, is_read)`; didaftarkan ke `AutoMigrate` di `main.go`.
- `utils/websocket.go`: `WSHubStore.clients` menyimpan `WSClientInfo{UserID, Role}`; `Register(conn, userID, role)`; tambah `BroadcastToUser(userID, event, payload)` & `BroadcastToRole(role, event, payload)`; `BroadcastEvent` global tetap dipertahankan.
- `routes/websocket.go`: meneruskan `user_id`/`role` dari claims JWT saat `Register`.
- `utils/notify.go`: `NotifyUser(db, userID, type, title, msg, refType, refID)` (async, pola `LogAudit`) â†’ simpan DB â†’ `BroadcastToUser` event `NOTIFICATION_CREATED`; `NotifyRole(db, role, ...)` â†’ buat notif per user berperan â†’ `BroadcastToRole`.
- `handlers/notification.go`: `GetNotifications` (scope user dari JWT, paginasi, filter `unread`), `GetUnreadNotificationCount`, `MarkNotificationRead` (ownership check), `MarkAllNotificationsRead`.
- `routes/notification.go`: grup `/notifications` + `Protected()`; **rute statis (`/unread-count`, `/read-all`) didaftarkan SEBELUM rute dinamis (`/:id/read`)**; registrasi di `main.go`.
- Hook: `AssignMaintenanceLog` â†’ notif ke teknisi; `CompleteMaintenanceLog` â†’ notif ke semua admin (label "Selesai"/"Perlu Tindak Lanjut"); `CreateFieldMaintenance` bila `attention` â†’ notif ke admin.

**Frontend**
- `components/NotificationBell.tsx`: ikon Bell + badge unread; dropdown 3 notifikasi teratas + tombol "Lihat lainnya" â†’ `/pemberitahuan`; subscribe `NOTIFICATION_CREATED` â†’ `toast.info` + refresh badge; fallback polling 60 detik; dirender di header halaman Dashboard.
- `components/DashboardLayout.tsx`: menu sidebar "Pemberitahuan" (`/pemberitahuan`).
- `app/pemberitahuan/page.tsx`: daftar lengkap notifikasi milik user (paginasi, filter belum dibaca, tandai semua, klik item â†’ mark-read + navigasi ke `/maintenance` atau `/field-maintenance`).

**Test & Verifikasi Sprint 1**
- `handlers/notification_test.go` (sqlmock, `SkipDefaultTransaction`): list scope user, filter unread, unread-count, read milik orang lain â†’ 403/404, read-all, UUID invalid â†’ 400, tanpa identitas â†’ 401.
- `utils/notify_test.go`: persist sukses/gagal; `BroadcastToUser`/`BroadcastToRole` tanpa client (guard clause).
- `go build ./... && go test ./... -count=1 && go vet ./...` + `npm run lint && npm run build` + `swag init` + uji runtime Docker (`https://localhost`).

### 6.4 Rancangan Teknis Sprint 6 (Skor Kesehatan CCTV)

**Backend**
- `models/cctv.go`: `HealthScore` integer default 100.
- `handlers/health_score.go`: agregasi lima faktor dalam jendela 90 hari, pembulatan skor akhir, endpoint persistence helper, dan `RunHealthScoreCalculator` saat startup + tiap 6 jam.
- `handlers/cctv.go`/`routes/cctv.go`: `GET /api/cctv/:id/health-score` protected semua role dan whitelist sorting `health_score` pada list CCTV.
- `CompleteMaintenanceLog` dan `CreateFieldMaintenance`: recalculation setelah transaksi berhasil; error kalkulator dicatat tanpa membatalkan operasi utama.

**Frontend**
- `app/cctv/page.tsx`: kolom Kesehatan dengan angka dan badge hijau/kuning/merah.
- `components/CCTVMap.tsx`: warna marker berdasarkan rentang skor; popup dan legend menampilkan angka/rentang sebagai cue non-warna.

**Test & Verifikasi**
- `handlers/health_score_test.go`: skenario tanpa riwayat, kombinasi faktor, ambang umur, clamp minimum, dan endpoint.
- `go build ./... && go test ./... -count=1 && go vet ./...` + `npm run lint && npm run build` + `swag init`.

### 6.5 Rancangan Teknis Sprint 7 (Dashboard Kinerja Teknisi)

**Backend**
- `models/setup.go`: `MaintenanceLog.AssignedAt` nullable untuk mencatat waktu assign/re-assign; log lama memakai `created_at` sebagai fallback.
- `handlers/maintenance.go` dan `handlers/schedule_runner.go`: mengisi waktu penugasan untuk assign manual, re-assign, dan recurring log yang langsung memiliki teknisi.
- `handlers/dashboard.go`: `GET /dashboard/technician-performance` memakai satu agregasi SQL; default rentang 30 hari, filter `start_date`/`end_date`, metrik tugas, completion rate, durasi assign â†’ complete, overdue, field check, dan quality score.
- `routes/dashboard.go`: endpoint berada di bawah `Protected()` dan `RequireRole("admin")`.

**Frontend**
- `app/kinerja-teknisi/page.tsx`: filter 7 hari/30 hari/kustom, tabel ranking sortable, state loading/error/empty, dan badge ambang completion/quality.
- `components/TechnicianPerformanceChart.tsx`: chart batang horizontal SVG responsif tanpa dependency chart eksternal.
- `components/DashboardLayout.tsx` dan `proxy.ts`: menu serta guard admin untuk `/kinerja-teknisi`.

**Test & Verifikasi**
- `handlers/technician_performance_test.go`: struktur respons, filter tanggal, dan role non-admin â†’ 403.
- `go build ./... && go test ./... -count=1 && go vet ./...` + `npm run lint && npm run build` + `swag init`.
- Uji manual di `docs/manual-test-technician-performance.md`.

### 6.6 Rancangan Teknis Sprint 8 (Laporan PDF Pengecekan Lapangan)

**Backend**
- `handlers/field_maintenance.go`: handler `GenerateFieldMaintenanceReportPDF` memuat relasi CCTV, teknisi, checklist, dan foto; Admin dapat mengakses seluruh record, sedangkan teknisi dibatasi ke record miliknya.
- PDF memakai `go-pdf/fpdf` dan helper kop surat dari `handlers/report.go`; isi laporan mencakup informasi CCTV, informasi pengecekan, tabel checklist, foto bukti yang di-embed bila file lokal tersedia, fallback URL foto, dan catatan umum.
- `routes/field_maintenance.go`: rute `GET /field-maintenance/:id/report-pdf` protected dan didaftarkan sebelum rute `/:id`.
- `handlers/field_maintenance_test.go`: test content type/prefix PDF, akses teknisi pemilik/non-pemilik, dan record tidak ditemukan; `routes/routes_test.go` memastikan route protected.
- Swagger docs diregenerasi dengan `swag init`.

**Frontend**
- `app/field-maintenance/page.tsx`: hook `useAuth` membatasi tombol `FileDown`/"Cetak Laporan" untuk Admin; download memakai `downloadApiFile` dengan cookie sesi dan respons blob.

**Test & Verifikasi**
- `go build ./... && go test ./... -count=1 && go vet ./...` + `npm run lint && npm run build` + `swag init`.
- Uji manual di `docs/manual-test-field-maintenance-pdf.md`.

### 6.7 Rancangan Teknis Sprint 9 (Ringkasan Operasional Otomatis)

**Backend**
- `models/daily_digest.go`: model `DailyDigest` dengan tanggal unik WITA, tujuh metrik operasional, dan summary naratif.
- `handlers/digest_generator.go`: scheduler timer menuju 06:00 WITA, agregasi tugas/field-check/audit dalam jendela 24 jam, idempotensi tanggal, penyimpanan digest, dan notifikasi `DAILY_DIGEST_AVAILABLE` ke Admin.
- `main.go`: `DailyDigest` didaftarkan ke `AutoMigrate` dan generator dijalankan sebagai goroutine saat startup.
- `handlers/dashboard.go`/`routes/dashboard.go`: endpoint admin `GET /dashboard/digest` dengan paginasi dan urutan tanggal terbaru.
- `handlers/digest_generator_test.go`: test agregasi/pembuatan, idempotensi, jadwal WITA, notifikasi, paginasi endpoint, dan role non-admin.

**Frontend**
- `app/ringkasan/page.tsx`: timeline kartu harian dengan summary, metrik tugas, pengecekan, perhatian, perubahan status, loading/error/empty state, dan paginasi.
- `components/DashboardLayout.tsx`: menu Admin "Ringkasan" dengan ikon `ClipboardList`.
- `proxy.ts`: `/ringkasan` ditambahkan ke guard admin server-side.

**Test & Verifikasi**
- `go build ./... && go test ./... -count=1 && go vet ./...` + `npm run lint && npm run build` + `swag init`.
- Uji manual di `docs/manual-test-daily-digest.md`.

### 6.8 Konvensi Eksekusi Tiap Sprint

1. Branch `feature/...` dari `main` terbaru â†’ Conventional Commits â†’ push â†’ PR (user merge) â†’ hapus branch lokal.
2. Changelog: `backend/changelog.md`, `frontend/changelog.md`, `docs/changelog.md` (format before/after, timestamp WITA, kategori, daftar file, alasan teknis).
3. Perbarui dokumen yang terdampak: `docs/AGENTS.md` (model/endpoint/menu baru), `docs/ROADMAP.md` (status task), `docs/TESTING.md` (matriks test), `docs/plan.md` (status sprint).

### 6.9 Rancangan Teknis Sprint 10 (Migrasi DB Versioned)

**Backend**
- `migrations/migrations.go`: embed FS (`//go:embed *.sql`) untuk semua file migration.
- `migrations/000001_init_schema.up.sql`: `CREATE TABLE` untuk 10 model (users, cctvs, maintenance_logs, field_maintenances, field_maintenance_checks, field_maintenance_images, audit_logs, notifications, maintenance_schedules, daily_digests) + index + 13 FK. Seluruh pernyataan idempoten (IF NOT EXISTS / DO $$ pg_constraint) agar aman atas DB eksisting hasil AutoMigrate.
- `migrations/000001_init_schema.down.sql`: rollback seluruh tabel; `CASCADE` pada `maintenance_logs` untuk FK sirkular.
- `config/migrate.go`: `RunMigrations(db)` (startup, `m.Up()` + toleransi `ErrNoChange`), `MigrationUp/MigrationDown/MigrationVersion` (CLI), `newMigrateInstance` (iofs + driver pgx/v5).
- `cmd/migrate/main.go`: CLI helper `up|down|version` memakai env DB_* yang sama dengan aplikasi.
- `main.go`: `RunMigrations` dijalankan sebelum operasi lain; blok `AutoMigrate` dikomentari dengan catatan "akan dihapus setelah migration stabil".
- `.github/workflows/ci.yml`: job backend-ci menambah service PostgreSQL dan step `up → down → up → version`.
- `config/versioned_migration_test.go`: test embed FS + konvensi nama, kelengkapan tabel/kolom, rollback, dan propagasi error tanpa PostgreSQL (sqlmock).
- Dependensi: `github.com/golang-migrate/migrate/v4 v4.19.1`.

**Dokumen**
- `docs/MIGRATION.md` (baru): cara kerja, CLI, integrasi CI, troubleshooting (dirty state), dan panduan menambah migration.

**Verifikasi**
- `go build ./... && go test ./... -count=1 && go vet ./...` PASS.
- Uji live PostgreSQL: `up` (10 tabel + 13 FK, versi 1) → `down` (semua tabel hilang) → `up` (versi 1) → `up` idempoten → `version`; skenario DB warisan AutoMigrate (tanpa `schema_migrations`) PASS; DB uji dihapus.

### 6.10 Rancangan Teknis Sprint 11 (Redis untuk Rate-Limit & Blacklist)

**Infra**
- `docker-compose.yml`: service `redis` (redis:7-alpine) — `command` AOF (`--appendonly yes`) + `--requirepass ${REDIS_PASSWORD:-}`; volume `redis_data:/data`; health check adaptif (kosong = `redis-cli ping`, terisi = `redis-cli -a "$REDIS_PASSWORD" ping | grep -q PONG`); port 6379 **tidak dipublish**; `api.depends_on.redis` → `condition: service_healthy`; komentar service dinomori ulang (db, redis, api, frontend, nginx).
- `.env.example` & `backend/.env.docker.example`: `REDIS_PASSWORD` (opsional, harus cocok antar file bila diisi); `REDIS_HOST=redis`, `REDIS_PORT=6379`.

**Backend**
- `config/redis.go` (baru): `ConnectRedis()` — `github.com/redis/go-redis/v9`, default `redis:6379`, password dari `REDIS_PASSWORD`; `Ping` ber-timeout 2 detik; gagal → LOG WARNING + `RedisClient=nil`/`RedisEnabled=false` (graceful fallback, bukan fail-fast — development tanpa Docker).
- `utils/blacklist.go`: interface `TokenBlacklist` (`Add(token, expiry)` / `IsRevoked(token)` / `Remove(token)` / `CleanupExpired`) agar mudah di-mock;
  `RedisTokenBlacklist` (key `sigap:blacklist:jwt`, `SADD`+`SISMEMBER`+`SREM`, `TTL` key mengikuti sisa masa hidup token **terpanjang** — menambah token ber-TTL pendek tidak mempersingkat token lain; `CleanupExpired` no-op karena EXPIRE otomatis) dan `memoryTokenBlacklist` (fallback, map + RWMutex). Facade lama (`RevokeToken`/`IsTokenRevoked`) dan `SetTokenBlacklist(b)` untuk pergantian backend; `StartBlacklistCleanup` tetap ada (no-op saat Redis).
- `middleware/limiter.go`: limiter fixed-window kustom (bukan `fiber/middleware/limiter`) — `rateLimitCounter` interface (`Increment(key, window)`); implementasi Redis (`INCR` + `EXPIRE` saat INCR=1) dan in-memory per-proses (reset saat window habis); key `limiter:{ip}:{endpoint}` (endpoint = pola route `c.Route().Path`, fallback `c.Path()`); fail-open bila INCR error (jangan blokir layanan); `GlobalLimiter` (100/menit) & `AuthLimiter` (5/menit) signature sama.
- `main.go`: `config.ConnectRedis()` setelah `ConnectDB`; `SetTokenBlacklist` (Redis bila aktif, else in-memory + peringatan); tutup koneksi Redis saat graceful shutdown.

**Test & Verifikasi**
- `utils/blacklist_test.go`: kontrak interface add→check→remove untuk **dua varian** (Redis via mock `redisCommander` — fake set/TTL — dan memory), TTL mengikuti token terpanjang, facade `RevokeToken`/`IsTokenRevoked`, cleanup expired memory, `SetTokenBlacklist`.
- `middleware/limiter_test.go`: test lama (≤100 lolos, >100 → 429) tetap hijau (fallback memory) + format key `limiter:{ip}:{endpoint}`, reset counter memory setelah window, isolasi counter per key.
- `go build ./... && go test ./... -count=1 && go vet ./...` PASS.
- Uji live Redis 7 (Docker): ConnectRedis → Revoke → IsRevoked → **persistensi blacklist lintas restart API (client/instance baru)** → TTL key terpasang → Remove → INCR pertama 1 → EXPIRE window → batas 5/menit request ke-6 ditolak → key memakai TTL (12 cek PASS); `docker compose config --quiet` VALID; container Redis healthy.

### 6.11 Rancangan Teknis Sprint 12 (Backup Otomatis & Restore)

**Skrip**
- `scripts/backup.sh`: `pg_dump` (opsi `--no-owner --no-privileges`) → `gzip` → `backups/db/sigap_<TS>.sql.gz`; `tar`+`gzip` folder uploads → `backups/uploads/uploads_<TS>.tar.gz` (archive kosong bila folder kosong); rotasi `find -mtime +RETENTION_DAYS -delete` (default 7 hari); timestamp WITA (Asia/Makassar); konfigurasi via env `DB_HOST/DB_PORT/DB_USER/DB_PASSWORD/DB_NAME/UPLOADS_DIR/BACKUP_DIR/RETENTION_DAYS` (default menyesuaikan deployment Docker).
- `scripts/restore.sh`: argumen `<sql_dump.gz>` [optional `<uploads.tar.gz>`]; validasi keberadaan file; restore DB = drop & recreate (`psql`) lalu `gunzip` dump → `psql`; restore uploads = hapus & buat ulang folder lalu `tar -xzf`.

**Infra (Opsi 2 — utama)**
- `docker-compose.yml`: service `backup` — image `postgres:15-alpine` + `dcron` (sudah ada klien PostgreSQL); `backup/entrypoint.sh` menulis crontab (`CRON_SCHEDULE` default `0 6 * * *`, `TZ=Asia/Makassar`) lalu `crond -f` di foreground; volume `backups_data:/backups`; folder uploads bind-mount read-only; `depends_on.db` healthy; kredensial dari root `.env`.
- `.gitignore`: tambah `/backups/`.

**Dokumen**
- `docs/BACKUP.md` (baru): prosedur backup manual & restore, jadwal cron (opsi container & host), verifikasi (integritas + end-to-end data utuh), dan disaster recovery.

**Verifikasi**
- `bash -n scripts/backup.sh scripts/restore.sh backup/entrypoint.sh` PASS; `docker compose config --quiet` VALID.
- Uji fungsional: nama file `sigap_<TS>.sql.gz`/`uploads_<TS>.tar.gz`, struktur `backups/db`+`backups/uploads`, rotasi `-mtime +7` menghapus backup tua, restore drop/recreate berjalan.
- Uji end-to-end (environment bersih): backup → verifikasi file → restore ke DB kosong → data utuh (langkah di `docs/BACKUP.md` bagian 7).

### 6.12 Rancangan Teknis Sprint 13 (Monitoring & Alerting)

**Backend**
- `monitoring/metrics.go` (baru): collector Prometheus ringan **tanpa library pihak ketiga** (sesuai konvensi proyek). Menyediakan: counter `sigap_http_requests_total{method,path,status}`, histogram `sigap_http_request_duration_seconds{method,path}` (bucket 0.005–10 + Inf + _sum/_count), gauge `sigap_db_pool_open_connections`, `sigap_db_pool_idle_connections`, `sigap_ws_active_connections`, `sigap_upload_storage_bytes`. Semua thread-safe (RWMutex). `Handler` Fiber untuk `GET /api/metrics` (text/plain; version=0.0.4). `Render()` menghasilkan text exposition valid.
- `monitoring/collector.go` (baru): `StartCollector(uploadsDir, interval)` goroutine memperbarui gauge DB pool (`sql.DB.Stats()`), WS aktif (`utils.Hub.ActiveConnections()`), dan ukuran uploads (rekursif).
- `middleware/metrics.go` (baru): mencatat setiap request — counter `(method, path pola, status)` + durasi histogram; endpoint `/api/metrics` dikecualikan.
- `handlers/health.go`: diperluas — `db_response_time_ms`, `redis_status`/`redis_response_time_ms` (bila Redis aktif), `uploads{status,path,size_bytes}`; helper `dirSize`.
- `utils/websocket.go`: tambah `ActiveConnections()`.
- `main.go`: pasang `middleware.Metrics()`, rute `GET /api/metrics` (publik), jalankan `monitoring.StartCollector("./uploads", 15s)`; `swag init`.

**Infra (opsional)**
- `monitoring/prometheus/prometheus.yml`: scrape `api:3000/api/metrics` tiap 15 detik.
- `docker-compose.override.yml`: service `prometheus` (port 9090) & `grafana` (port 3000, datasource auto-provision), volume `prometheus_data`/`grafana_data` — tidak diwajibkan di compose utama.
- `monitoring/grafana/provisioning/datasources/prometheus.yml`: auto-register datasource.

**Test**
- `monitoring/metrics_test.go`: counter + histogram (nilai bucket kumulatif), HELP/TYPE untuk semua metrik, gauge, empty state, Reset, escape label, format angka tetap (tanpa eksponensial).
- `middleware/metrics_test.go`: observasi request (counter & count), exclude `/api/metrics`, status non-200, Content-Type text/plain.
- `handlers/health_test.go`: diperluas — `db_response_time_ms`, `redis_status=Disabled`, `uploads{status,path,size_bytes}`; `dirSize`.

**Verifikasi**
- `go build ./...`, `go test ./... -count=1`, `go vet ./...` PASS (seluruh package hijau).
- `docker compose config --quiet` VALID (base + override Prometheus/Grafana ter-resolve).
- Uji manual: `GET /api/metrics` mengembalikan text/plain Prometheus valid; `GET /api/health` memuat field baru.

### 6.13 Rancangan Teknis Sprint 14–15 (Container Hardening & Logging Terstruktur)

> Dikerjakan paralel oleh dua sub-agent pada branch `feature/production-hardening`.

**Task 7.5 — Container Hardening**
- `backend/Dockerfile`: buat user non-root `sigap` (`addgroup -S sigap && adduser -S -G sigap sigap`); kepemilikan `/app/uploads` & `/app/logs` oleh `sigap`; `USER sigap` sebelum `CMD`.
- `docker-compose.yml` (service `api`): `read_only: true` + tmpfs `/tmp` & `/app/logs`; `cap_drop: [ALL]`; resource limits (`deploy.resources.limits` + `mem_limit`/`cpus`); Docker secrets `db_password`/`jwt_secret`/`aes_secret_key` (file `./secrets/*` di-mount ke `/run/secrets/`); `LOG_LEVEL` env; `nginx` diberi resource limits.
- `backend/utils/secrets.go` (baru): `GetSecret(env, file)` baca file `/run/secrets/<name>` bila ada (trim whitespace), fallback ke env; `RequireSecret` fail-fast.
- Wiring secret: `config/database.go` (`DB_PASSWORD`), `utils/jwt.go` (`JWT_SECRET`), `utils/encryption.go` (`AES_SECRET_KEY`), `main.go` (`RequireSecret`).
- `utils/secrets_test.go` (baru): file menang atas env, fallback env, kosong bila keduanya hilang, trim whitespace.
- Env docs: `.env.example`, `backend/.env.docker.example`; `.gitignore` tambah `/secrets/`.

**Task 7.6 — Logging Terstruktur**
- `go.mod`/`go.sum`: `gopkg.in/natefinch/lumberjack.v2 v2.2.1`.
- `utils/logger.go`: `parseLogLevel(env LOG_LEVEL, default info)`; file core memakai `lumberjack.Logger` (`MaxSize 100 MB`, `MaxBackups 5`, `MaxAge 30 hari`, `Compress`); level diterapkan konsisten di file & console; file encoder JSON.
- `utils/logger_test.go` (baru): `parseLogLevel` (valid, invalid/empty → info, case-insensitive), `InitLogger` non-nil.
- Konsistensi: `middleware/logger.go`, `utils/audit.go`, `middleware/recovery.go` sudah memakai `utils.Log` (struktur JSON) — diverifikasi, tidak ada `fmt.Println`/`log.Printf`.

**Verifikasi**
- `go build ./...`, `go test ./... -count=1`, `go vet ./...` PASS (seluruh package hijau).
- `docker compose config --quiet` VALID (base + override); `api` punya `read_only`, `cap_drop`, resource limits, secrets `/run/secrets`, tmpfs `/app/logs`.
- Manual (environment Docker): `docker exec ... whoami` → non-root; `docker inspect` tidak menampilkan nilai secret; log ter-rotate di `/app/logs`.

### 6.14 Rancangan Teknis Sprint 16 (Progressive Web App)

**Frontend**
- `public/manifest.json`: web manifest — `name`/`short_name` "SIGAP", `start_url "/dashboard"`, `scope "/"`, `display "standalone"`, `orientation "portrait"`, `background_color "#162218"` (Kanopi Malam), `theme_color "#467235"` (Hijau Komando), ikon `192x192` & `512x512` (purpose `any` + `maskable`), `lang "id"`.
- `public/icons/icon-192x192.png` & `icon-512x512.png`: ikon PNG di-generate dari `public/nusantara-logo.png` (canvas persegi 192/512 dengan latar Kanopi Malam, logo ditengahkan).
- `public/sw.js`: service worker — pre-cache shell (`/`, `/dashboard`, `/login`, manifest, ikon); strategi cache: navigasi HTML **network-first** (fallback cache offline), aset statis **cache-first** (dengan pembaruan latar belakang), API (`/api/*`) **network-first** (fallback cache, else 503 JSON "offline"); `CACHE_VERSION = sigap-shell-v1`; `skipWaiting` + `clients.claim`.
- `src/components/ServiceWorkerRegister.tsx` (baru, client component): registrasi `navigator.serviceWorker.register('/sw.js', { scope: '/' })` dalam `useEffect` (aman SSR), hanya di Secure Context.
- `src/app/layout.tsx`: `metadata.manifest`, ikon (incl. apple), `appleWebApp`; `viewport` export (theme-color, device-width, viewportFit); render `<ServiceWorkerRegister />`.
- `next.config.ts`: header `Cache-Control: no-cache` + `Service-Worker-Allowed: /` untuk `/sw.js`, dan `public, max-age=3600` untuk `/manifest.json`.

**Dokumen**
- `docs/manual-test-pwa.md` (baru): manual test blackbox (P1–P9, S1–S9) sesuai format manual test lain — prasyarat, pengujian otomatis, skenario installability/offline shell/strategi cache/update SW/Lighthouse, kriteria lulus.
- ROADMAP Task 8.1 ✅ Selesai (Fase 8 1/4); plan Sprint 16; `frontend/changelog.md`; `docs/changelog.md`; AGENTS.md (aturan manual test per sprint).

**Verifikasi**
- `cd frontend && npm run lint` (0 error) dan `npm run build` (exit 0) PASS.
- Manual: install di desktop/Android, ikon muncul, shell termuat offline, API network-first, update SW membersihkan cache lama; Lighthouse PWA score ≥ 80 (panduan `docs/manual-test-pwa.md`).

### 6.15 Rancangan Teknis Sprint 17 (Kalender Jadwal Pemeliharaan)

**Frontend**
- `src/components/CalendarGrid.tsx` (baru): grid kalender bulanan CSS mandiri (tanpa dependency chart eksternal) — header Senin–Minggu, 7 kolom × 5-6 baris, navigasi prev/next bulan + "Hari ini", dot warna status per hari (hijau completed, biru scheduled, kuning ongoing, merah failed/overdue), maksimal 3 dot + "+N", legend status. Helper: `toDayKey`, `formatLongDate`, `formatMonthTitle`, `STATUS_DOT_CLASS`, `STATUS_META`, `CalendarTask`.
- `src/app/kalender/page.tsx` (baru): halaman `/kalender` (admin + teknisi). Ambil data `GET /api/maintenance-logs?start_date=&end_date=&page=1&limit=100` untuk rentang bulan, kelompokkan per `scheduled_date` (fallback `created_at`); filter per teknisi (admin, dropdown dari `/users?role=teknisi`; teknisi di-scope backend); panel detail hari (nama CCTV, kode, status, tindakan, teknisi); ikon lucide; pattern loading/error/empty.
- `src/components/DashboardLayout.tsx`: menu sidebar "Kalender" (`/kalender`) dengan ikon `Calendar` (terbuka untuk admin & teknisi).

**Dokumen**
- `docs/manual-test-calendar.md` (baru): manual test blackbox (P1–P5, S1–S9) sesuai format manual test lain — menu/akses, tampilan bulan, navigasi, dot status, detail hari, filter teknisi, scope non-admin, konsistensi data, responsif/aksesibilitas, kriteria lulus.
- ROADMAP Task 8.2 ✅ Selesai (Fase 8 2/4); plan Sprint 17; `frontend/changelog.md`; `docs/changelog.md`.

**Verifikasi**
- `cd frontend && npm run lint` (0 error) dan `npm run build` (exit 0) PASS.
- Manual: bulan benar, klik hari menampilkan tugas, warna status sesuai, filter teknisi admin berfungsi, teknisi hanya melihat miliknya (panduan `docs/manual-test-calendar.md`).

### 6.16 Rancangan Teknis Sprint 18 (Notifikasi Push Browser)

**Backend**
- `models/push_subscription.go` (baru): `PushSubscription` — `Base`, `UserID` (FK), `Endpoint`, `KeyP256DH` (column `key_p256dh`), `KeyAuth`; index `(user_id)`.
- `migrations/000002_push_subscriptions.up.sql`/`.down.sql`: tabel `push_subscriptions` + index + FK ke users (idempoten).
- `utils/push.go` (baru): `SendPush` (async, query subscription user, kirim ke semua via `webpush-go`; no-op bila VAPID kosong — fallback in-app), `SendPushToSubscription` (payload JSON `PushNotificationPayload`), `SetSendNotificationFn` untuk mock. Env: `VAPID_PUBLIC_KEY`/`VAPID_PRIVATE_KEY`/`VAPID_EMAIL`.
- `handlers/push.go` (baru): `POST /push/subscribe` (upsert idempotent), `DELETE /push/unsubscribe` (soft delete by endpoint); helper `currentUserID`.
- `routes/push.go` (baru): grup `/push` protected.
- `utils/notify.go`: `NotifyUser` & `NotifyUserSync` memanggil `SendPush` setelah notifikasi in-app berhasil (async, warning bila gagal).
- `main.go`: daftarkan `SetupPushRoutes`; `go.mod` tambah `github.com/SherClockHolmes/webpush-go v1.4.0`.
- Test: `handlers/push_test.go` (subscribe create/upsert/validasi/unauthorized, unsubscribe success/missing endpoint), `utils/push_test.go` (SendPushToSubscription mock, no-op VAPID kosong, kirim ke semua subscription, payload format).
- Env docs: `backend/.env.docker.example` & `.env.example` tambah `VAPID_*`.

**Frontend**
- `public/sw.js`: handle `push` (showNotification dari payload `data.push`) & `notificationclick` (fokus/buka halaman by reference_type); bump `CACHE_VERSION` ke `sigap-shell-v2`.
- `src/components/PushNotificationOptIn.tsx` (baru): cek dukungan, minta izin, subscribe (`pushManager.subscribe` + `applicationServerKey` dari `NEXT_PUBLIC_VAPID_PUBLIC_KEY`), kirim ke `POST /api/push/subscribe`; nonaktif → `DELETE /api/push/unsubscribe`.
- `src/app/profile/page.tsx`: bagian "Notifikasi Push" dengan `PushNotificationOptIn`.

**Dokumen**
- `docs/manual-test-push-notification.md` (baru): manual test blackbox (P1–P6, S1–S9) — opt-in/izin, push saat tugas/overdue/attention, opt-out, fallback in-app, konsistensi payload, scope per user, kriteria lulus.
- ROADMAP Task 8.3 ✅ Selesai (Fase 8 3/4); plan Sprint 18; `backend/changelog.md`; `frontend/changelog.md`; `docs/changelog.md`; TESTING.md; AGENTS.md.

**Verifikasi**
- `go build`/`go test ./... -count=1`/`go vet` PASS; `npm run lint` (0 error)/`npm run build` (exit 0) PASS.
- Manual: notifikasi OS muncul saat tugas baru ditugaskan / eskalasi overdue / hasil attention; opt-out berfungsi; fallback in-app bila VAPID tidak dikonfigurasi (panduan `docs/manual-test-push-notification.md`).

### 6.17 Rancangan Teknis Sprint 19 (Geotag Pengecekan Lapangan)

> Dikerjakan paralel oleh tiga sub-agent pada branch `feature/geotag-field-check`.

**Backend**
- `models/field_maintenance.go`: tambah `CheckLatitude`/`CheckLongitude` (string nullable, `json:"check_latitude,omitempty"`).
- `migrations/000003_add_field_maintenance_geotag.up/down.sql`: `ALTER TABLE field_maintenances ADD/DROP COLUMN check_latitude/check_longitude varchar(30) NULL`.
- `handlers/field_maintenance.go`: `CreateFieldMaintenanceRequest` + `check_latitude`/`check_longitude`; helper `validateCoordinates(lat, lng)` (keduanya wajib bersamaan, format angka valid, lat -90..90, lng -180..180); set pada record; respons list/detail otomatis menyertakan field via model.
- Test: `field_maintenance_test.go` (invalid lat 400, invalid lng 400, valid geotag 201 + create/reload kolom).

**Frontend**
- `components/FieldMaintenanceModal.tsx`: saat wizard `field` dibuka, `navigator.geolocation.getCurrentPosition` (enableHighAccuracy, timeout 10s; guard `'geolocation' in navigator`); simpan `coords`; indikator "Lokasi tercatat" / "Lokasi belum tercatat (opsional)"; kirim `check_latitude`/`check_longitude` pada body `field` (opsional); mode `complete` tidak menangkap lokasi.
- `components/FieldCheckLocationMap.tsx` (baru): mini-map react-leaflet dengan marker CCTV (biru) + marker "Lokasi Pengecekan" (oranye), legend, fallback pusat IKN.
- `app/field-maintenance/page.tsx`: interface + `check_latitude`/`check_longitude`/`cctv.latitude|longitude`; bagian "Lokasi Pengecekan" di modal detail (render mini-map bila koordinat ada, else "Tidak ada data lokasi"); `dynamic(..., { ssr: false })`.

**Dokumen**
- `docs/manual-test-geotag.md` (baru): manual test blackbox (P1–P6, S1–S6) — GPS diizinkan/ditolak, validasi koordinat, mini-map admin, mode complete, konsistensi, kriteria lulus.
- ROADMAP Task 8.4 ✅ Selesai (Fase 8 **4/4 SELESAI**); plan Sprint 19; `backend/changelog.md`; `frontend/changelog.md`; `docs/changelog.md`; TESTING.md; AGENTS.md.

**Verifikasi**
- `go build`/`go test ./... -count=1`/`go vet` + `golangci-lint run` PASS; `npm run lint` (0 error)/`npm run build` (exit 0) PASS.
- Manual: koordinat tercatat di DB saat GPS diizinkan, wizard tetap jalan saat ditolak, admin melihat marker "Lokasi Pengecekan" di peta (panduan `docs/manual-test-geotag.md`).

### 6.18 Rancangan Teknis Sprint 20–23 (Fase 9 — Streaming CCTV Real-Time)

> Mengubah peta sebaran CCTV dari view-only menjadi **interactive live monitoring center**. Teknologi: **MediaMTX** (RTSP-to-WebRTC proxy) + **WHEP** (browser playback) + Nginx reverse proxy. Keputusan arsitektur streaming disepakati 2026-08-07 (lihat ROADMAP Fase 9).

#### Sprint 20 — Fondasi Streaming (Task 9.1 + 9.2)

**Backend (Task 9.1)**
- `models/cctv.go`: tambah `ChannelPort int` (`channel_port`, validasi 1–35), `RTSPPassword string` (`rtsp_password`, JSON `-`, **terenkripsi AES-256**), `StreamEnabled bool` (`stream_enabled`, default `false`).
- `migrations/000004_add_cctv_streaming.up/down.sql`: `ALTER TABLE cctvs ADD/DROP COLUMN channel_port integer NOT NULL DEFAULT 1, rtsp_password text NOT NULL DEFAULT '', stream_enabled boolean NOT NULL DEFAULT false` (idempoten).
- `utils/rtsp.go` (baru): tipe `StreamType` (`MainStream=1`/`SubStream=2`); `BuildRTSPURL` — `rtsp://user:pass@host:port/Streaming/channels/{ChannelID}` dengan `ChannelID = channel_port × 100 + stream_type`; `BuildStreamPath`; `GetRTSPPort` (default sub-stream); edge cases channel ≤ 0.
- `handlers/cctv.go`: `CreateCCTVRequest`/`UpdateCCTVRequest` + `channel_port`/`rtsp_password`/`stream_enabled`; encrypt `rtsp_password` saat simpan; `rtsp_password` tidak pernah dikembalikan ke response.
- Env NVR global: `RTSP_NVR_HOST`, `RTSP_NVR_PORT`, `RTSP_NVR_USER`, `MEDIAMTX_API_URL` di `.env.example` & `backend/.env.docker.example`.

**Infra (Task 9.2)**
- `mediamtx.yml`: RTSP→WebRTC relay, `sourceOnDemand: yes`, `protocols: [tcp]`, REST API `:9997`.
- `docker-compose.yml`: service `mediamtx` (image `bluenviron/mediamtx:latest-ffmpeg`, IP `172.28.0.11`, resource limits 2 CPU/1024M, health check, UDP `8189`).
- `nginx/nginx.conf`: upstream `sigap_mediamtx` + location `/stream/*` → `mediamtx:8889`, `/hls/*` → `mediamtx:8888`.

#### Sprint 21 — Backend API & MediaMTX Client (Task 9.3)

- `utils/mediamtx.go` (baru): struct `MediaMTXClient` — HTTP client ke `MEDIAMTX_API_URL` (`http://mediamtx:9997/v3/`); method `AddPath`, `RemovePath`, `GetPath`, `ListPaths`.
- `handlers/stream.go` (baru): `StartStream` (decrypt `rtsp_password` → build RTSP URL → register path MediaMTX → return WHEP URL; validasi `stream_enabled=false`/tanpa `channel_port` → 400, CCTV not found → 404), `StopStream`, `StartAllStreams`, `StopAllStreams`, `GetStreamStatus` (gabung data DB + info path MediaMTX), `GetStreamURL` (WHEP + HLS). Swagger annotations.
- `routes/stream.go` (baru): grup `/streams` admin-only (`Protected()` + `RequireRole("admin")`) — `GET /status`, `POST /start-all`, `POST /stop-all`, `POST /:id/start`, `POST /:id/stop`, `GET /:id/url` (rute statis sebelum dinamis).
- `main.go`: `SetupStreamRoutes` + goroutine auto-start stream aktif saat boot.

#### Sprint 22 — Frontend: Player & Streaming UI (Task 9.4 + 9.5 + 9.6)

- `components/WHEPPlayer.tsx` (baru): WebRTC player browser-native `RTCPeerConnection` (zero SDK); WHEP signaling via `POST /stream/{path}/whep`; auto-reconnect exponential backoff; status overlay (connecting/live/error/reconnecting); toggle fullscreen; toggle SD sub-stream ↔ HD main-stream; indikator 🔴 LIVE; SSR-safe (dynamic import).
- `components/StreamModal.tsx` (baru): modal overlay saat marker diklik — header nama/kode/badge status + skor kesehatan, body `WHEPPlayer` embed, footer lokasi/IP/channel port, tombol Start/Stop, link Master CCTV.
- `components/StreamGridView.tsx` (baru): grid view multi-stream responsif (tiap sel berisi `WHEPPlayer` mini; sel tanpa streaming placeholder).
- `components/CCTVMap.tsx`: prop `onMarkerClick` + indikator streaming aktif pada marker; interface `CCTVMapItem` + `stream_enabled`/`channel_port`.
- `app/sebaran-cctv/page.tsx`: state `selectedCctv` + `StreamModal`; toggle view Peta/Grid; filter "Streaming Aktif"; status bar "N/35 kamera streaming aktif".
- `app/cctv/page.tsx`: section "Konfigurasi Streaming" di modal edit CCTV (`channel_port` 1–35, `rtsp_password` toggle visibility, `stream_enabled` checkbox, preview URI RTSP); kolom tabel master ikon streaming aktif.

#### Sprint 23 — Integrasi End-to-End & Manual Test Streaming

- `docs/manual-test-streaming.md` (baru): manual test blackbox S1–S9 sesuai format manual test lain.
- Docker E2E: `docker compose up -d --build` → 8 service running; health check semua PASS; MediaMTX API reachable.
- Dokumentasi: `docs/AGENTS.md`, `docs/TESTING.md`, `docs/plan.md`, `docs/changelog.md` [1.28.30].

**Verifikasi**
- `go build`/`go test ./... -count=1`/`go vet` + `golangci-lint run` + `swag init` PASS; `npm run lint` (0 error)/`npm run build` (exit 0) PASS; `docker compose config --quiet` VALID.
- Manual: `docs/manual-test-streaming.md`.
