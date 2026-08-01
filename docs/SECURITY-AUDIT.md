# SECURITY-AUDIT — SIGAP

> Audit keamanan, pemahaman sistem, dan rekomendasi pengembangan.
> Tanggal audit: 2026-08-01 · Cakupan: seluruh monorepo (`backend/`, `frontend/`, infra).

---

## 1. Ringkasan Eksekutif

SIGAP (Sistem Identifikasi Gangguan Aset Pusat) adalah sistem internal berbasis web untuk
mengelola, memantau, menginventarisasi, dan mencatat pemeliharaan CCTV kawasan IKN.

**Arsitektur:** Monorepo 3 layanan — Go Fiber API (`:3000`), Next.js 16 App Router (`:3001`),
PostgreSQL 15 — diorkestrasi dengan Docker Compose + Cloudflare quick tunnel.

**Kekuatan keamanan yang sudah ada:**

- Autentikasi JWT (HMAC-SHA256) + RBAC (Admin/Teknisi) + blacklist token + logout server-side.
- Enkripsi QR AES-256-GCM (tanpa menyimpan kredensial perangkat).
- Rate limiting global (100 req/menit) & khusus login (5 req/menit).
- Upload foto dengan deteksi MIME via magic bytes (tanpa fallback ekstensi), akses file dilindungi JWT.
- Validasi input terpusat (`go-playground/validator`), whitelist kolom sorting (anti SQL-injection).
- Soft-delete, graceful shutdown, security headers, audit trail, Swagger off di production.
- Auto-logout inaktivitas (15 menit) di frontend.

**Hasil audit:** ditemukan **5 temuan Kritis**, **7 temuan Tinggi**, dan **8 catatan Sedang/Rendah**.

**Status perbaikan (diperbarui):**
- 🔴 Kritis: **5/5 ✅ Selesai** (Fase 1 roadmap).
- 🟠 Tinggi: **7/7 ✅ Selesai** (Fase 2 roadmap).
- 🟡 Sedang/Rendah: **5/8 ✅ Selesai** (temuan 13, 14, 15, 18, 20); tersisa 16, 17, 19.
- Rencana bertahap & to-do per item: lihat `ROADMAP.md` dan bagian **3.4**.

---

## 2. Pemahaman Sistem & Alur

### 2.1 Struktur Folder

```text
SIGAP/
├── docker-compose.yml      # Orkestrasi db, api, frontend, nginx (TLS)
├── makefile                # Task runner (BE + FE)
├── README.md
├── docs/                   # Dokumentasi: ROADMAP, SECURITY-AUDIT, PRODUCT, DESIGN, agent, changelog, plan
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

- **Autentikasi:** Login (username/email + bcrypt) → JWT 24 jam → tersimpan di `localStorage` →
  `Protected()` + `RequireRole("admin")` di backend. Idle-timer memicu logout + revoke token.
- **Master CCTV:** UUID perangkat dienkripsi AES-256-GCM menjadi `QrPayload`. Scan QR → dekripsi →
  cocokkan ID → tampilkan detail. Dukung import Excel, template, bulk-delete, cetak label QR.
- **Pemeliharaan:** Log CRUD + upload foto bukti. `resolveMaintenanceAsset` mematerialisasi CCTV ke
  tabel `assets` saat pertama kali dipakai untuk maintenance.
- **Realtime:** WSHub broadcast event (`CCTV_UPDATED`, `MAINTENANCE_CREATED/UPDATED`) → frontend auto-refetch.
- **Audit trail:** `LogAudit` async mencatat seluruh aksi penting (AUTH/CCTV/ASSET/MAINTENANCE/USER).

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
| 8 | Reset password / hapus user tidak mencabut token lama | `backend/handlers/user.go` | Token user tetap valid hingga 24 jam | Implementasi token version (klaim `jti`) atau revoke semua token saat reset | ✅ Selesai (2.2) |
| 9 | Rate limiter & audit percaya `X-Real-IP`/`X-Forwarded-For` dari client | `backend/middleware/limiter.go:13-30`, `utils/audit.go:13-24` | Spoof header → bypass rate limit jika backend terekspos langsung | Hanya percaya header proxy jika di belakang trusted proxy (konfigurasi trust/proxy header Fiber) | ✅ Selesai (2.3) |
| 10 | Import Excel tanpa batas ukuran/baris | `backend/handlers/cctv.go:592-643` | File dibaca penuh ke memori (zip bomb) → DoS | Batasi ukuran file (mis. 5 MB) dan jumlah baris (mis. 5000); stream parsing | ✅ Selesai (2.4) |
| 11 | CORS default `*` jika env tidak diset | `backend/main.go:82-85` | Akses lintas-origin terbuka | Fail-fast atau default ke daftar origin eksplisit; wajib set `CORS_ALLOWED_ORIGINS` di produksi | ✅ Selesai (2.5) |
| 12 | Docker Compose: password DB hardcoded + port DB publik + quick tunnel | `docker-compose.yml:13,19,77` | Kredensial bocor di repo; DB terekspos ke host; layanan terekspos ke URL publik tanpa auth | Hapus quick tunnel untuk produksi; pakai reverse proxy + TLS; secret management; jangan publish 5432 ke host | ✅ Selesai (2.6) |

### 3.3 🟡 Sedang / Catatan — 5/8 Selesai

| # | Temuan | Lokasi | Perbaikan | Status |
|---|--------|--------|-----------|--------|
| 13 | Password minimum 6 karakter | `handlers/auth.go:26`, `handlers/user.go:22` | Naikkan ke 8+ dengan kebijakan kompleksitas | ✅ Selesai (2.9) |
| 14 | Teknisi bisa update maintenance log milik orang lain | `handlers/maintenance.go:317` | Tambahkan ownership check (teknisi hanya log miliknya) | ✅ Selesai (2.8) |
| 15 | CSP & HSTS masih dikomentari | `backend/middleware/security.go:22-32` | Aktifkan untuk produksi (CSP ketat + HSTS) | ✅ Selesai (2.7) |
| 16 | Role check frontend murni client-side; `/audit-logs` tetap memicu request saat teknisi buka halaman | `frontend/src/app/audit-logs/page.tsx:83-85` | Tambah Next.js middleware (server-side route guard) + hindari request tak perlu | ⬜ Belum (Fase 3/berkelanjutan) |
| 17 | Audit log async tanpa retry | `backend/utils/audit.go:65-86` | Tambah retry/queue atau minimal log error lebih jelas | ⬜ Belum (Fase 4.1/berkelanjutan) |
| 18 | Export CSV terbuka untuk semua user | `handlers/maintenance.go`, `handlers/report.go` | Batasi role (teknisi hanya data relevan) | ✅ Selesai (3.9) |
| 19 | Frontend subscribe `AUDIT_LOG_ADDED` tapi backend tidak pernah broadcast | `frontend/src/app/audit-logs/page.tsx:90` | Sinkronkan: broadcast event dari `LogAudit` atau hapus subskripsi | ⬜ Belum (Fase 3/berkelanjutan) |
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

**🔜 Nanti (Fase 3/4 — berkelanjutan):**
- [ ] Temuan 16 — Next.js middleware server-side route guard (Fase 3/berkelanjutan)
- [ ] Temuan 19 — Sinkronkan broadcast `AUDIT_LOG_ADDED` (Fase 3/berkelanjutan)
- [ ] Temuan 17 — Audit log async retry/queue (Fase 4.1 / berkelanjutan)

---

## 4. Rekomendasi Fitur

1. **Generate QR server-side** — endpoint `GET /api/cctv/:id/qrcode` + batch PDF/PNG (pakai `skip2/go-qrcode`); solusi offline untuk lapangan IKN.
2. **Ubah password mandiri (self-service)** + **force-change password saat login pertama** (user baru / hasil reset).
3. **Refresh token / sliding session** dan **revoke token saat role diubah / password di-reset** (klaim `jti` + versi token).
4. **Dashboard analitik** — grafik tren pemeliharaan, distribusi status, peringatan pemeliharaan **overdue**, notifikasi tugas untuk teknisi.
5. **Halaman profil** (data diri + ubah password) terpisah dari manajemen user admin.
6. **Unifikasi model** `assets` (legacy) vs `cctv` — hilangkan duplikasi lewat `resolveMaintenanceAsset`.
7. **Laporan PDF / export audit-log** + filter rentang tanggal.
8. **Cleanup file yatim** (upload sukses tapi penyimpanan log gagal) + strip EXIF GPS pada foto.
9. **Batas ekspor per role** — teknisi hanya data yang relevan, admin penuh.

---

## 5. Rekomendasi Kualitas & Infrastruktur

- **CI/CD** (`.github/workflows/ci.yml` sudah ada): tambah `npm run lint` dan `golangci-lint`; perluas unit test (saat ini hanya auth & JWT) ke cctv/upload/websocket/pagination.
- **Migrasi DB versioned** (golang-migrate/Atlas) — `AutoMigrate` tidak cukup untuk produksi.
- **Redis** untuk rate-limit & blacklist token agar mendukung multi-instance.
- **Backup**: cron `pg_dump` + backup volume uploads + restore script.
- **Monitoring**: endpoint metrics (Prometheus) di samping health check; log Zap sudah baik.
- **Docker hardening**: container non-root, resource limits, jangan publish 5432 ke host, secret via secrets/env.
- **Reverse proxy** (Nginx/Traefik) + TLS + security headers di level proxy; ganti quick tunnel.
- **Isi `docs/changelog.md`** sesuai rencana `plan.md` (sudah diisi Fase 1–3, 2026-08-01).
