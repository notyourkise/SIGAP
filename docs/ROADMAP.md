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
| 4 | Infrastruktur & Kualitas | 0/8 | ⬜ Belum |

> **Urutan pengerjaan:** Fase 1 ✅ → **Fase 2** ✅ → **Fase 3** ✅ → **Fase 4** (prioritas berikutnya, mulai task 4.1).
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
| 2.2 | Revoke token lama saat reset password / role diubah (token version `jti`) | `backend/utils/jwt.go`, `backend/handlers/user.go`, `backend/utils/blacklist.go` | Sedang | ✅ Selesai |
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
| 3.2 | Refresh token / sliding session | `backend/utils/jwt.go`, `backend/handlers/auth.go`, `frontend/src/context/AuthContext.tsx` | Sedang | ✅ Selesai |
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

---

## Fase 4 — Infrastruktur & Kualitas (Berkelanjutan)

> **Status: 🚧 BERJALAN.** Fase 4.1 sebagian selesai; lanjut dari task berikutnya.
> Referensi lengkap cakupan test & gotchas: `docs/TESTING.md`.

| # | Task | Keterangan | Status |
|---|------|-----------|--------|
| 4.1 | Perluas unit test backend (cctv, upload, websocket, pagination, rate limit) | `backend/**/*_test.go` | 🚧 Sebagian (utils, middleware, handlers inti selesai; tersisa import/export CCTV, update/delete maintenance, websocket, routes, models, config) |
| 4.2 | Tambah `npm run lint` + `golangci-lint` di CI | `.github/workflows/ci.yml` | ⬜ Belum |
| 4.3 | Migrasi DB versioned (golang-migrate/Atlas) | ganti `AutoMigrate` bertahap | ⬜ Belum |
| 4.4 | Redis untuk rate-limit & blacklist (multi-instance) | `backend/config`, `backend/middleware/limiter.go`, `backend/utils/blacklist.go` | ⬜ Belum |
| 4.5 | Backup DB (`pg_dump` cron) + backup uploads + restore script | `scripts/` baru | ⬜ Belum |
| 4.6 | Monitoring: endpoint metrics (Prometheus) | `backend/handlers/metrics.go` (baru) | ⬜ Belum |
| 4.7 | Container non-root, resource limits, secret management | `backend/Dockerfile`, `docker-compose.yml` | ⬜ Belum |
| 4.8 | Isi `docs/changelog.md` (milestone monorepo) | `docs/changelog.md` | ⬜ Belum |

---

## Konvensi Pengerjaan

1. `git checkout main && git pull origin main`
2. `git checkout -b feature/…` atau `bugfix/…`
3. Commit dengan Conventional Commits: `feat(scope): …` / `fix(scope): …`
4. Push → buat PR → tunggu merge oleh user → hapus branch lokal
5. Catat perubahan di `../backend/changelog.md` / `../frontend/changelog.md` (format: before/after,
   timestamp WITA, kategori, file, alasan)
