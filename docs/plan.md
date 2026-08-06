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
├── Makefile
├── README.md
├── .gitignore
│
├── docs/
│   ├── AGENTS.md
│   ├── ACCESS.md
│   ├── changelog.md
│   ├── plan.md
│   ├── ROADMAP.md
│   ├── SECURITY-AUDIT.md
│   ├── TESTING.md
│   ├── PRODUCT.md
│   └── DESIGN.md
│
├── backend/
│   ├── agent.md
│   ├── changelog.md
│   ├── main.go
│   ├── go.mod
│   ├── go.sum
│   ├── config/
│   ├── routes/
│   ├── handlers/
│   └── models/
│
└── frontend/
    ├── agent.md
    ├── changelog.md
    ├── package.json
    ├── tailwind.config.ts
    └── src/
        ├── app/
        ├── components/
        └── lib/

## 6. Rencana Pengembangan Aktif (per 2026-08-06)

> Sumber kebenaran status & urutan pengerjaan: `docs/ROADMAP.md` (Fase 5–8).
> Bagian ini mencatat konteks keputusan agar tidak hilang antar sesi pengembangan.

### 6.1 Konteks Keputusan

**Diskusi 2026-08-04:**
- Sistem **masih pilot/UAT** — belum menampung data produksi nyata, sehingga fitur alur
  kerja (Jalur A) didahulukan; pengerasan infrastruktur (sisa Fase 4: backup, versioned
  migration, container hardening, Redis, metrics) menyusul menjelang go-live.
- Prioritas: **Jalur A — Alur Logika**, 5 task berurutan 5.1 → 5.5 (lihat `ROADMAP.md` Fase 5).
- Notifikasi: **in-app saja** (bell + daftar notifikasi + toast via WebSocket); email tidak dipakai.
- Status CCTV otomatis: **konservatif** — hanya `aktif ↔ pemeliharaan`; `rusak`/`nonaktif`
  tetap keputusan manual admin. **Per 2026-08-05 status `rusak` digabung ke `nonaktif`**
  (fungsional identik: keduanya dilindungi sync dan tampil sama "Nonaktif"); enum resmi
  kini `aktif` / `pemeliharaan` / `nonaktif`.

**Brainstorming 2026-08-05:**
- Setelah analisis gap menyeluruh, disepakati **3 fase baru** (Fase 6–8) setelah Fase 5 tuntas:
  - **Fase 6 — Kecerdasan Operasional**: skor kesehatan CCTV, kinerja teknisi, laporan PDF pengecekan lapangan, ringkasan harian otomatis.
  - **Fase 7 — Kesiapan Produksi**: menyerap sisa Fase 4 (4.3–4.7) + logging terstruktur. Wajib sebelum go-live.
  - **Fase 8 — Pengalaman Lapangan**: PWA, kalender jadwal, push notification, geotag pengecekan.
- Fase 6 dan 7 bisa dikerjakan **paralel** (fitur vs infrastruktur). Jika tim terbatas dan go-live dekat, Fase 7 diprioritaskan.
- Sisa Fase 4 (4.3–4.7) resmi **dilebur ke Fase 7** — tidak lagi menjadi backlog terpisah.

### 6.2 Ringkasan Sprint

| Sprint | Fase | Task ROADMAP | Cakupan | Status |
|--------|------|--------------|---------|--------|
| 1 | 5 | 5.1 Notifikasi in-app | Model `Notification`, endpoint `/api/notifications`, `WSHub` targeted, `NotificationBell`, hook assign/complete/field-check | ✅ Kode selesai (2026-08-04, branch `feature/notification-in-app`) — menunggu uji runtime Docker |
| 2 | 5 | 5.2 Sinkron status CCTV | Helper dalam transaksi complete/field-check; tabel transisi konservatif | ✅ Selesai (2026-08-05, branch `feature/sync-cctv-status`) — build/test/vet PASS, uji manual S1–S8 di `docs/manual-test-sync-cctv.md` (3 bug ditemukan & ditindaklanjuti: pesan validasi [FE 1.25.11], normalisasi Inggris [BE 0.28.1], guard legacy uppercase [BE 0.28.2], dropdown status [FE 1.25.12]) + penggabungan `rusak`→`nonaktif` [BE 0.28.3 / FE 1.25.13] |
| 3 | 5 | 5.3 Re-assign tugas | Perluas endpoint assign ke log `ongoing`, notif dua pihak, tombol "Alihkan" | ✅ Selesai (2026-08-05, branch `feature/reassign-maintenance`) — build/test/vet PASS + lint/build PASS + swag init; menunggu uji runtime Docker (teknisi lama dapat notif "dialihkan", teknisi baru dapat notif "tugas baru") |
| 4 | 5 | 5.4 Jadwal berulang | Model `MaintenanceSchedule`, goroutine generator harian, CRUD + halaman admin `/schedules` | ✅ Selesai (2026-08-06, branch `feature/recurring-schedule`) — build/test/vet PASS + lint/build PASS + swag init; generator idempotent per hari per jadwal, tanpa teknisi → log `scheduled`, dengan teknisi → `ongoing` + notifikasi |
| 5 | 5 | 5.5 Eskalasi overdue | Kolom `OverdueNotifiedAt`, goroutine saat startup + tiap jam, notif admin/teknisi, reset saat reschedule | ✅ Selesai (2026-08-06, branch `feature/overdue-escalation`) — test checker + reset reschedule PASS |
| 6 | 6 | 6.1 Skor Kesehatan CCTV | Skor 0–100 dari 5 faktor, endpoint breakdown, badge tabel/peta, goroutine 6 jam, recalculation saat complete/field-check | ✅ Selesai (2026-08-06, branch `feature/health-score`) — test kalkulasi + endpoint, backend/frontend build PASS |
| 7 | 6 | 6.2 Dashboard Kinerja Teknisi | Endpoint statistik per teknisi, halaman `/kinerja-teknisi` | ✅ Selesai (2026-08-06, branch `feature/technician-performance`) — test struktur/filter/role + backend/frontend build dan lint PASS |
| 8 | 6 | 6.3 Laporan PDF Pengecekan | PDF per-record field-maintenance dengan kop surat IKN | ✅ Selesai (2026-08-06, branch `feature/field-maintenance-pdf`) |
| 9 | 6 | 6.4 Ringkasan Operasional | Goroutine harian, model `DailyDigest`, halaman `/ringkasan` | ⬜ Belum |
| 10–15 | 7 | 7.1–7.6 Kesiapan Produksi | Versioned migration, Redis, backup, monitoring, hardening, logging | ⬜ Belum |
| 16–19 | 8 | 8.1–8.4 Pengalaman Lapangan | PWA, kalender, push notification, geotag | ⬜ Belum |

### 6.3 Rancangan Teknis Sprint 1 (Notifikasi In-App)

**Backend**
- `models/notification.go`: `Notification{UserID, Type, Title, Message, ReferenceID, ReferenceType, IsRead}` + konstanta tipe (`MAINTENANCE_ASSIGNED`, `MAINTENANCE_REASSIGNED`, `MAINTENANCE_UNASSIGNED`, `MAINTENANCE_COMPLETED`, `MAINTENANCE_OVERDUE`, `FIELD_CHECK_ATTENTION`); index komposit `(user_id, is_read)`; didaftarkan ke `AutoMigrate` di `main.go`.
- `utils/websocket.go`: `WSHubStore.clients` menyimpan `WSClientInfo{UserID, Role}`; `Register(conn, userID, role)`; tambah `BroadcastToUser(userID, event, payload)` & `BroadcastToRole(role, event, payload)`; `BroadcastEvent` global tetap dipertahankan.
- `routes/websocket.go`: meneruskan `user_id`/`role` dari claims JWT saat `Register`.
- `utils/notify.go`: `NotifyUser(db, userID, type, title, msg, refType, refID)` (async, pola `LogAudit`) → simpan DB → `BroadcastToUser` event `NOTIFICATION_CREATED`; `NotifyRole(db, role, ...)` → buat notif per user berperan → `BroadcastToRole`.
- `handlers/notification.go`: `GetNotifications` (scope user dari JWT, paginasi, filter `unread`), `GetUnreadNotificationCount`, `MarkNotificationRead` (ownership check), `MarkAllNotificationsRead`.
- `routes/notification.go`: grup `/notifications` + `Protected()`; **rute statis (`/unread-count`, `/read-all`) didaftarkan SEBELUM rute dinamis (`/:id/read`)**; registrasi di `main.go`.
- Hook: `AssignMaintenanceLog` → notif ke teknisi; `CompleteMaintenanceLog` → notif ke semua admin (label "Selesai"/"Perlu Tindak Lanjut"); `CreateFieldMaintenance` bila `attention` → notif ke admin.

**Frontend**
- `components/NotificationBell.tsx`: ikon Bell + badge unread; dropdown 3 notifikasi teratas + tombol "Lihat lainnya" → `/pemberitahuan`; subscribe `NOTIFICATION_CREATED` → `toast.info` + refresh badge; fallback polling 60 detik; dirender di header halaman Dashboard.
- `components/DashboardLayout.tsx`: menu sidebar "Pemberitahuan" (`/pemberitahuan`).
- `app/pemberitahuan/page.tsx`: daftar lengkap notifikasi milik user (paginasi, filter belum dibaca, tandai semua, klik item → mark-read + navigasi ke `/maintenance` atau `/field-maintenance`).

**Test & Verifikasi Sprint 1**
- `handlers/notification_test.go` (sqlmock, `SkipDefaultTransaction`): list scope user, filter unread, unread-count, read milik orang lain → 403/404, read-all, UUID invalid → 400, tanpa identitas → 401.
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
- `handlers/dashboard.go`: `GET /dashboard/technician-performance` memakai satu agregasi SQL; default rentang 30 hari, filter `start_date`/`end_date`, metrik tugas, completion rate, durasi assign → complete, overdue, field check, dan quality score.
- `routes/dashboard.go`: endpoint berada di bawah `Protected()` dan `RequireRole("admin")`.

**Frontend**
- `app/kinerja-teknisi/page.tsx`: filter 7 hari/30 hari/kustom, tabel ranking sortable, state loading/error/empty, dan badge ambang completion/quality.
- `components/TechnicianPerformanceChart.tsx`: chart batang horizontal SVG responsif tanpa dependency chart eksternal.
- `components/DashboardLayout.tsx` dan `proxy.ts`: menu serta guard admin untuk `/kinerja-teknisi`.

**Test & Verifikasi**
- `handlers/technician_performance_test.go`: struktur respons, filter tanggal, dan role non-admin → 403.
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

### 6.7 Konvensi Eksekusi Tiap Sprint

1. Branch `feature/...` dari `main` terbaru → Conventional Commits → push → PR (user merge) → hapus branch lokal.
2. Changelog: `backend/changelog.md`, `frontend/changelog.md`, `docs/changelog.md` (format before/after, timestamp WITA, kategori, daftar file, alasan teknis).
3. Perbarui dokumen yang terdampak: `docs/AGENTS.md` (model/endpoint/menu baru), `docs/ROADMAP.md` (status task), `docs/TESTING.md` (matriks test), `docs/plan.md` (status sprint).
