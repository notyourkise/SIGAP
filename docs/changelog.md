# Changelog — SIGAP (Milestone Monorepo)

> Catatan perubahan tingkat proyek (monorepo: `backend/`, `frontend/`, infra).
> Rincian per-bagian lihat `../backend/changelog.md` & `../frontend/changelog.md`.

## [1.4.1] — 2026-08-02 — Fase 4.1: Penyelesaian Coverage Test

### Testing Backend

- Test jalur sukses update/delete CCTV dan maintenance.
- Test handshake WebSocket dengan token valid dan bootstrap route terlindungi.
- Test resolusi legacy asset ke CCTV dan pembaruan matriks coverage.
- Sisa yang ditunda: import transaksi penuh, token version mismatch WebSocket, dan migration update penuh.

kategori: Testing, Quality, Migration
file terkait (root/infra): `docs/ROADMAP.md`, `docs/TESTING.md`, `docs/changelog.md`
verifikasi: `go test ./... -count=1`, `go vet ./...`, `golangci-lint v2.12.2`, `npm run lint`, `npm run build`.
alasan perubahan: Menyelaraskan dokumentasi dengan hasil implementasi Fase 4.1 sebelum melanjutkan ke versioned database migration.

---

## [1.4.0] — 2026-08-02 — Fase 4: Quick Wins Keamanan & Kualitas

### Keamanan, Pengujian, dan CI

- Proxy server-side Next.js untuk route `/audit-logs` dan `/users`, termasuk validasi sesi melalui backend dan pengecekan role admin.
- Audit log memiliki retry terbatas dan broadcast `AUDIT_LOG_ADDED` setelah penyimpanan berhasil.
- Unit test tambahan untuk edge case CCTV, maintenance, user, audit retry, dan UUID model hook.
- CI menjalankan `npm run lint` dan `golangci-lint` selain build serta unit test.

kategori: Security, Testing, CI, Quality
file terkait (root/infra): `docs/ROADMAP.md`, `docs/SECURITY-AUDIT.md`, `.github/workflows/ci.yml`
verifikasi: `go test ./... -count=1`, `go build ./...`, `npm run lint`, `npm run build`.
alasan perubahan: Menutup temuan keamanan tersisa dan menjadikan pemeriksaan kualitas otomatis pada setiap pull request.

---

## [1.3.0] — 2026-08-01 — Fase 3: Fitur (Kapabilitas)

### Fitur Kapabilitas (ROADMAP 3.1 – 3.9)

- **3.1** Self-service change password (`POST /api/auth/change-password`) + paksa ganti password saat login pertama (`must_change_password`).
- **3.2** Sliding session — masa berlaku JWT diperpanjang saat akses aktif; blacklist & revoke token lama tetap berjalan.
- **3.3** Dashboard analitik: `GET /api/dashboard/summary` (statistik, tren 6 bulan, distribusi status, overdue) + komponen `DashboardAnalytics`.
- **3.4** Notifikasi tugas teknisi: `GET /api/dashboard/tasks` (teknisi = tugas miliknya, admin = semua; `pending_count`/`overdue_count`).
- **3.5** Halaman profil (`/profile`) + `PUT /api/auth/profile`; kartu profil di sidebar jadi tautan.
- **3.6** Unifikasi model `assets` ↔ `cctv`: CCTV jadi satu-satunya sumber kebenaran. Model & handler/rute `assets` dihapus; `MaintenanceLog` kini `cctv_id` → `cctvs`; migrasi data legacy `asset_id` otomatis sekali saat startup (`config/migrate.go`).
- **3.7** Laporan PDF (`GET /api/maintenance-logs/report-pdf`, via `go-pdf/fpdf`), ekspor audit-log CSV (`GET /api/audit-logs/export`), dan filter rentang tanggal `start_date`/`end_date` pada list & ekspor.
- **3.8** Privasi & kebersihan upload: seluruh foto dinormalisasi ke JPEG (metadata EXIF/GPS terhapus); goroutine cleanup menghapus file yatim berumur >24 jam.
- **3.9** Pembatasan ekspor per role: teknisi hanya mengekspor/mengunduh laporan data miliknya; admin penuh.

kategori: Feature, Security, Dashboard, PDF, Upload
file terkait (root/infra): `ROADMAP.md`, `SECURITY-AUDIT.md`, `../README.md`, `changelog.md` (docs, baru)
verifikasi: `go build/vet/test` PASS (backend), `npm run build` PASS (frontend), Swagger regenerated.
alasan perubahan: Meningkatkan nilai operasional sistem dan menutup temuan keamanan 18 (ekspor per role).

---

## [1.2.0] — 2026-08-01 — Fase 2: Penguatan (Hardening)

### Hardening Keamanan (SECURITY-AUDIT temuan 7–15, 20)

- Migrasi token JWT ke cookie httpOnly + pola CSRF double-submit (`sigap_session`, `sigap_csrf`, header `X-CSRF-Token`).
- Revoke token lama saat reset password / role diubah (klaim `tv` / token version).
- Rate limiter & audit log hanya percaya header proxy dari IP terdaftar (`TRUSTED_PROXY_IPS`).
- Batas ukuran (4 MB) & baris (1000) import Excel.
- CORS tanpa wildcard; default aman `http://localhost:3001`.
- Docker hardening: quick tunnel dihapus, reverse proxy Nginx + TLS, port DB/API/frontend tidak dipublish, secret via root `.env`.
- CSP & HSTS aktif di produksi; kebijakan password min 8 + kompleksitas; ownership check maintenance log; penanganan 401 pada fetch raw frontend.

kategori: Security, Hardening, Docker, Infra
file terkait (root/infra): `docker-compose.yml`, `nginx/` (baru), `.env.example` (baru, root), `.gitignore`
verifikasi: `go build/vet/test` PASS (backend), `npm run build` PASS (frontend).
alasan perubahan: Menutup temuan Fase 2 (SECURITY-AUDIT.md) untuk mencegah pencurian token, CSRF, spoofing IP, overload impor, eksposur layanan, dan password lemah.

---

## [1.1.0] — 2026-08-01 — Fase 1: Kritis (Prioritas Tertinggi)

### Perbaikan Kritis (SECURITY-AUDIT temuan 1–6)

- JWT: hapus fallback secret hardcoded (wajib env `JWT_SECRET`).
- Seeder: admin default tidak lagi di-reset tiap startup; password wajib dari env `ADMIN_DEFAULT_PASSWORD`.
- WebSocket: wajib autentikasi (handshake token).
- Stored XSS pada cetak label QR diperbaiki (render React, bukan `document.write`).
- QR Code dipindah ke server-side (`skip2/go-qrcode`), lepas dari API pihak ketiga.
- Pagination dibatasi cap maksimum (`MAX_PAGE_SIZE`).

kategori: Security, Kritis
file terkait (root/infra): `README.md`, `docker-compose.yml`
verifikasi: `go build/vet/test` PASS (backend), `npm run build` PASS (frontend).
alasan perubahan: Menutup celah keamanan paling berbahaya sebelum pengembangan lanjutan.

---

## [1.0.0] — 2026-08-01 — Rilis Awal

- Sistem identifikasi & inventaris CCTV (SIGAP) berbasis Go Fiber + Next.js + PostgreSQL.
- Fitur inti: manajemen master CCTV, QR Code terenkripsi AES-256, scanner QR, import Excel massal, log pemeliharaan dengan unggah foto, manajemen pengguna & RBAC (Admin/Teknisi), autentikasi JWT, Swagger UI.
- Deployment Docker Compose (db, api, frontend).
