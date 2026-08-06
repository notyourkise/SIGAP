# Dokumen Perencanaan Proyek SIGAP

## 1. Identitas Sistem
- **Nama Proyek:** SIGAP
- **Kepanjangan:** Sistem Identifikasi Gangguan Aset Pusat
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
| 10 | 7 | 7.1 Migrasi DB Versioned | golang-migrate v4, file SQL terurut, `RunMigrations`, CLI up/down/version, AutoMigrate fase transisi, CI step, `docs/MIGRATION.md` | âœ… Selesai (2026-08-06, branch `feature/versioned-migration`) â€” build/test/vet PASS + uji live PostgreSQL up/down/up PASS |
| 11â€“15 | 7 | 7.2â€“7.6 Kesiapan Produksi | Redis, backup, monitoring, hardening, logging | â¬œ Belum |
| 16â€“19 | 8 | 8.1â€“8.4 Pengalaman Lapangan | PWA, kalender, push notification, geotag | â¬œ Belum |

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
