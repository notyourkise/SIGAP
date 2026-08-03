# SECURITY-AUDIT — SIGAP

> Audit keamanan, pemahaman sistem, dan rekomendasi pengembangan.
> Audit awal: 2026-08-01 · Status terakhir: 2026-08-03 · Cakupan: seluruh monorepo (`backend/`, `frontend/`, infra).

---

## 1. Ringkasan Eksekutif

SIGAP (Sistem Identifikasi Gangguan Aset Pusat) adalah sistem internal berbasis web untuk
mengelola, memantau, menginventarisasi, dan mencatat pemeliharaan CCTV kawasan IKN.

**Arsitektur:** Monorepo dengan 4 service Docker — Go Fiber API (`:3000` internal), Next.js 16
App Router (`:3000` internal), PostgreSQL 15 (`:5432` internal), dan Nginx reverse proxy
(`:80`/`:443`) — diorkestrasi dengan Docker Compose. Akses browser resmi adalah
`https://localhost`; Cloudflare quick tunnel sudah dihapus.

**Kekuatan keamanan yang sudah ada:**

- Autentikasi JWT (HMAC-SHA256) + RBAC (Admin/Teknisi) + blacklist token + logout server-side.
- Enkripsi QR AES-256-GCM (tanpa menyimpan kredensial perangkat).
- Rate limiting global (100 req/menit) & khusus login (5 req/menit).
- Upload foto dengan deteksi MIME via magic bytes (tanpa fallback ekstensi), akses file dilindungi JWT.
- Validasi input terpusat (`go-playground/validator`), whitelist kolom sorting (anti SQL-injection).
- Soft-delete, graceful shutdown, security headers, audit trail, Swagger off di production.
- Single-origin melalui Nginx dengan TLS, routing API/WebSocket, dan cookie sesi httpOnly yang
  konsisten di `https://localhost`.

**Hasil audit:** ditemukan **5 temuan Kritis**, **7 temuan Tinggi**, dan **8 catatan Sedang/Rendah**.

**Status perbaikan (diperbarui):**
- 🔴 Kritis: **5/5 ✅ Selesai** (Fase 1 roadmap).
- 🟠 Tinggi: **7/7 ✅ Selesai** (Fase 2 roadmap).
- 🟡 Sedang/Rendah: **8/8 ✅ Selesai** (temuan 13, 14, 15, 16, 17, 18, 19, 20).
- Rencana bertahap & to-do per item: lihat `ROADMAP.md` dan bagian **3.4**.

---

## 2. Pemahaman Sistem & Alur

### 2.1 Struktur Folder

```text
SIGAP/
├── docker-compose.yml      # Orkestrasi db, api, frontend, nginx (TLS)
├── makefile                # Task runner (BE + FE)
├── README.md
├── docs/                   # Dokumentasi: AGENTS, ACCESS, ROADMAP, SECURITY-AUDIT, TESTING, PRODUCT, DESIGN, changelog, plan
├── backend/
│   ├── main.go             # Entry point, middleware stack, graceful shutdown
│   ├── config/database.go  # Koneksi PostgreSQL + connection pooling
│   ├── models/             # User, MaintenanceLog, CCTV, AuditLog + seeder
│   ├── handlers/           # auth, cctv, maintenance, user, audit, dashboard, upload, report, health
│   ├── middleware/         # auth (JWT), limiter, security, logger, recovery
│   ├── routes/             # Registrasi rute per modul + websocket
│   ├── utils/              # encryption, jwt, blacklist, audit, websocket, pagination, validator
│   ├── docs/               # Swagger/OpenAPI (generated)
│   └── uploads/            # Foto bukti pemeliharaan
└── frontend/
    └── src/
        ├── app/            # dashboard, cctv, maintenance, users, audit-logs, login
        ├── components/     # DashboardLayout, ProtectedRoute, AuthenticatedImage, CCTVMap, Pagination
        ├── context/        # AuthContext, ThemeContext, WebSocketContext
        └── lib/api.ts      # Fetch wrapper (Bearer token + auto 401)
```

### 2.2 Alur Utama

- **Autentikasi:** Login (username/email + bcrypt) → JWT 5 jam → cookie `sigap_session` httpOnly
  dengan CSRF double-submit → `Protected()` + `RequireRole("admin")` di backend. Sliding session
  dan idle-timer frontend tidak digunakan; logout manual atau kedaluwarsa token mengakhiri sesi.
- **Master CCTV:** UUID perangkat dienkripsi AES-256-GCM menjadi `QrPayload`. Scan QR → dekripsi →
  cocokkan ID → tampilkan detail. Dukung import Excel, template, bulk-delete, cetak label QR.
- **Pemeliharaan:** Log CRUD + upload foto bukti yang langsung merujuk ke CCTV melalui `cctv_id`.
- **Realtime:** WSHub broadcast event (`CCTV_UPDATED`, `MAINTENANCE_CREATED/UPDATED`, `AUDIT_LOG_ADDED`) → frontend auto-refetch.
- **Audit trail:** `LogAudit` async mencatat seluruh aksi penting dengan retry terbatas dan broadcast setelah tersimpan.

---

## 3. Temuan Keamanan

### 3.1 🔴 Kritis — ✅ 5/5 Selesai

| # | Temuan | Lokasi | Dampak | Perbaikan | Status |
|---|--------|--------|--------|-----------|--------|
| 1 | JWT secret fallback hardcoded & password seeder fallback ter-commit | `backend/utils/jwt.go:14`, `backend/models/seeder.go:16` | Secret publik diketahui siapa pun dengan akses repo → token palsu | Hapus seluruh fallback; fail-fast jika env kosong (sudah ada `requireEnv`, lengkapi juga di `getJWTSecret`) | ✅ Selesai (1.1) |
| 2 | `SeedAdmin` me-reset password admin ke env **setiap startup** | `backend/models/seeder.go:42-49` | Admin tidak bisa memiliki password permanen; reset password oleh admin hilang saat restart | Ubah seeder: hanya buat akun jika belum ada; jangan sinkronkan password ulang | ✅ Selesai (1.2) |
| 3 | WebSocket `/api/ws` tanpa autentikasi | `backend/routes/websocket.go:18-46`, `frontend/src/context/WebSocketContext.tsx:29-35` | Siapa pun yang terhubung menerima broadcast data CCTV lengkap (IP NVR/kamera, koordinat) + log maintenance (relasi Asset/User) | Tambahkan autentikasi JWT pada handshake WebSocket (token via query/subprotocol); hanya terima dari client terverifikasi | ✅ Selesai (1.3) |
| 4 | Stored-XSS pada print window | `frontend/src/app/cctv/page.tsx:571, 700, 814` | `document.write` menginterpolasi `nama`/`lokasi_area` mentah ke dokumen same-origin → eksekusi script | Buat komponen React untuk label QR (render via `dangerouslySetInnerHTML` dihindari); escape semua field atau gunakan DOM API | ✅ Selesai (1.4) |
| 5 | `qr_payload` dikirim ke pihak ketiga `api.qrserver.com` | `cctv/page.tsx:568, 814, 1090, 1554` | Bocor data payload ke layanan eksternal; gagal di jaringan internal/offline IKN | Generate QR server-side memakai `skip2/go-qrcode` (sudah jadi dependency backend) atau render QR client-side (canvas) | ✅ Selesai (1.5) |

### 3.2 🟠 Tinggi — 7/7 Selesai

| # | Temuan | Lokasi | Dampak | Perbaikan | Status |
|---|--------|--------|--------|-----------|--------|
| 6 | Pagination `limit` tanpa cap maks | `backend/utils/pagination.go:20-28` | `limit=9999999999` → dump seluruh tabel / DoS memori | Beri batas maksimal (mis. 100) dan validasi `offset` | ✅ Selesai (1.6) |
| 7 | JWT & data user di `localStorage` | `frontend/src/lib/api.ts:13`, `AuthContext.tsx:47` | Dapat dicuri oleh XSS; tidak ada httpOnly cookie | Migrasi ke cookie httpOnly + CSRF protection, atau minimal kurangi paparan | ✅ Selesai (2.1) |
| 8 | Reset password / hapus user tidak mencabut token lama | `backend/handlers/user.go` | Pada saat audit, token user tetap valid hingga 24 jam | Implementasi token version (`tv`) atau revoke semua token saat reset | ✅ Selesai (2.2); token kini berlaku 5 jam |
| 9 | Rate limiter & audit percaya `X-Real-IP`/`X-Forwarded-For` dari client | `backend/middleware/limiter.go:13-30`, `utils/audit.go:13-24` | Spoof header → bypass rate limit jika backend terekspos langsung | Hanya percaya header proxy jika di belakang trusted proxy (konfigurasi trust/proxy header Fiber) | ✅ Selesai (2.3) |
| 10 | Import Excel tanpa batas ukuran/baris | `backend/handlers/cctv.go:592-643` | File dibaca penuh ke memori (zip bomb) → DoS | Batasi ukuran file (mis. 5 MB) dan jumlah baris (mis. 5000); stream parsing | ✅ Selesai (2.4) |
| 11 | CORS default `*` jika env tidak diset | `backend/main.go:82-85` | Akses lintas-origin terbuka | Fail-fast atau default ke daftar origin eksplisit; wajib set `CORS_ALLOWED_ORIGINS` di produksi | ✅ Selesai (2.5) |
| 12 | Docker Compose: password DB hardcoded + port DB publik + quick tunnel | `docker-compose.yml:13,19,77` | Kredensial bocor di repo; DB terekspos ke host; layanan terekspos ke URL publik tanpa auth | Hapus quick tunnel untuk produksi; pakai reverse proxy + TLS; secret management; jangan publish 5432 ke host | ✅ Selesai (2.6) |

### 3.3 🟡 Sedang / Catatan — 8/8 Selesai

| # | Temuan | Lokasi | Perbaikan | Status |
|---|--------|--------|-----------|--------|
| 13 | Password minimum 6 karakter | `handlers/auth.go:26`, `handlers/user.go:22` | Naikkan ke 8+ dengan kebijakan kompleksitas | ✅ Selesai (2.9) |
| 14 | Teknisi bisa update maintenance log milik orang lain | `handlers/maintenance.go:317` | Tambahkan ownership check (teknisi hanya log miliknya) | ✅ Selesai (2.8) |
| 15 | CSP & HSTS masih dikomentari | `backend/middleware/security.go:22-32` | Aktifkan untuk produksi (CSP ketat + HSTS) | ✅ Selesai (2.7) |
| 16 | Role check frontend murni client-side; `/audit-logs` tetap memicu request saat teknisi buka halaman | `frontend/src/proxy.ts`, `frontend/src/app/audit-logs/page.tsx` | Proxy memvalidasi sesi dan role melalui endpoint backend; halaman tidak request sebelum admin terkonfirmasi | ✅ Selesai (Fase 4 quick wins) |
| 17 | Audit log async tanpa retry | `backend/utils/audit.go` | Retry terbatas 3 kali dengan backoff 100 ms dan pencatatan error akhir | ✅ Selesai (Fase 4 quick wins) |
| 18 | Export CSV terbuka untuk semua user | `handlers/maintenance.go`, `handlers/report.go` | Batasi role (teknisi hanya data relevan) | ✅ Selesai (3.9) |
| 19 | Frontend subscribe `AUDIT_LOG_ADDED` tapi backend tidak pernah broadcast | `backend/utils/audit.go`, `frontend/src/app/audit-logs/page.tsx` | Broadcast event setelah audit log berhasil disimpan | ✅ Selesai (Fase 4 quick wins) |
| 20 | Fetch raw di `cctv/page.tsx` (bulk-delete, template, import) tidak menangani 401 | `cctv/page.tsx:318, 351, 394` | Gunakan `apiRequest` atau tambahkan penanganan 401 | ✅ Selesai (2.1) |

---

### 3.4 To-do List — Urutan Pengerjaan & Status

Prioritas kerja dari yang **harus dikerjakan lebih dulu** menuju yang bisa ditunda.
Tandai `[x]` saat selesai; nomor di dalam kurung menunjuk task pada `ROADMAP.md`.

**✅ Selesai (Fase 1 — Kritis):**
- [x] Temuan 1 — JWT fallback hardcoded (1.1)
- [x] Temuan 2 — Seeder reset password (1.2)
- [x] Temuan 3 — WebSocket tanpa auth (1.3)
- [x] Temuan 4 — Stored-XSS print QR (1.4)
- [x] Temuan 5 — QR pihak ketiga (1.5)
- [x] Temuan 6 — Pagination tanpa cap (1.6)

**✅ Selesai (Fase 2 — Penguatan):**
- [x] Temuan 7 — Migrasi token ke cookie httpOnly + CSRF (2.1)
- [x] Temuan 8 — Revoke token lama saat reset/role berubah (2.2)
- [x] Temuan 9 — Rate limiter & audit hanya percaya trusted proxy (2.3)
- [x] Temuan 10 — Batasi ukuran & baris import Excel (2.4)
- [x] Temuan 11 — CORS fail-fast / origin eksplisit (2.5)
- [x] Temuan 12 — Docker hardening: hapus quick tunnel, TLS, secret, jangan publish 5432 (2.6)
- [x] Temuan 15 — Aktifkan CSP & HSTS (2.7)
- [x] Temuan 14 — Ownership check maintenance log (2.8)
- [x] Temuan 13 — Kebijakan password min 8 + kompleksitas (2.9)
- [x] Temuan 20 — Penanganan 401 pada fetch raw bulk-delete/template/import (2.1)

**✅ Selesai (Fase 3 — Fitur):**
- [x] Temuan 18 — Batasi ekspor per role (3.9)

**✅ Selesai (Fase 4 quick wins):**
- [x] Temuan 16 — Next.js middleware server-side route guard
- [x] Temuan 19 — Sinkronkan broadcast `AUDIT_LOG_ADDED`
- [x] Temuan 17 — Audit log async retry terbatas

---

## 4. Rekomendasi Fitur dan Status Implementasi

Item rekomendasi berikut telah diimplementasikan pada Fase 3. Item 3 disederhanakan pada
backend 0.16.0: sliding session tidak dipertahankan dan JWT berlaku tetap selama 5 jam.

1. **Generate QR server-side** — ✅ endpoint `GET /api/cctv/:id/qrcode` menggunakan `skip2/go-qrcode`.
2. **Ubah password mandiri** dan **force-change password saat login pertama** — ✅ selesai.
3. **Revoke token saat role/password berubah** — ✅ token version `tv`; sliding session diganti JWT statis 5 jam.
4. **Dashboard analitik** — ✅ tren pemeliharaan, distribusi status, overdue, dan tugas teknisi.
5. **Halaman profil** — ✅ data diri dan ubah password.
6. **Unifikasi model** `assets` (legacy) dan `cctv` — ✅ `cctv` menjadi sumber kebenaran melalui `config/migrate.go`.
7. **Laporan PDF, export audit-log, dan filter tanggal** — ✅ selesai.
8. **Cleanup file yatim dan strip EXIF GPS** — ✅ selesai.
9. **Batas ekspor per role** — ✅ teknisi hanya dapat mengekspor data miliknya.

---

## 5. Rekomendasi Kualitas & Infrastruktur

- **CI/CD** (`.github/workflows/ci.yml`): `npm run lint`, `golangci-lint`, build, dan unit test sudah dijalankan; cakupan unit test Fase 4.1 sudah diperluas ke handler, middleware, utils, routes, models, dan config.
- **Migrasi DB versioned** (golang-migrate/Atlas) — `AutoMigrate` tidak cukup untuk produksi.
- **Redis** untuk rate-limit & blacklist token agar mendukung multi-instance.
- **Backup**: cron `pg_dump` + backup volume uploads + restore script.
- **Monitoring**: endpoint metrics (Prometheus) di samping health check; log Zap sudah baik.
- **Docker hardening**: container non-root, resource limits, jangan publish 5432 ke host, secret via secrets/env.
- **Reverse proxy** Nginx + TLS + security headers sudah diterapkan; akses resmi kini melalui `https://localhost`.
- **Isi `docs/changelog.md`** sesuai rencana `plan.md` sudah diterapkan untuk milestone proyek.
