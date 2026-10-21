# SECURITY-AUDIT — SIGAP

> Audit keamanan, pemahaman sistem, dan rekomendasi pengembangan.
> Audit awal: 2026-08-01 · Diperbarui: 2026-10-21 · Cakupan: seluruh monorepo (`backend/`, `frontend/`, `mobile/`, dan infrastruktur).
>
> Temuan 1–20 di bagian 3 adalah rekam audit awal. Ringkasan arsitektur dan catatan risiko terbuka telah diselaraskan dengan kondisi proyek setelah Sprint 40.

---

## 1. Ringkasan Eksekutif

SIGAP (Sistem Informasi Gangguan & Aset Pusat) adalah sistem internal berbasis web dan mobile untuk
mengelola, memantau, menginventarisasi, dan mencatat pemeliharaan CCTV kawasan IKN.

**Arsitektur saat ini:** Monorepo dengan 7 service Docker: PostgreSQL 15, Redis 7, Go Fiber API,
Next.js 16, Nginx TLS, backup cron, dan MediaMTX. Nginx memublikasikan HTTP/TLS (`80`/`443`)
dan MediaMTX memublikasikan UDP `8189` untuk media ICE; database, Redis, API, frontend, backup,
serta endpoint HTTP/API MediaMTX tetap berada di network internal.
Akses browser resmi staging adalah `https://localhost`. Fitur AI detection dan service `sigap_ai`
telah dihapus pada Sprint 40.

**Kekuatan keamanan yang sudah ada:**

- Autentikasi JWT HMAC-SHA256 dalam cookie httpOnly, CSRF double-submit, RBAC Admin/Teknisi,
  token-version, blacklist Redis, dan logout server-side.
- Enkripsi QR dan kredensial RTSP per-CCTV memakai AES-256-GCM; nilai sensitif tidak boleh muncul
  pada respons API atau dokumentasi.
- Lifecycle QR memvalidasi ciphertext aktif selain ID hasil dekripsi; rotasi mengenkripsi ulang ID CCTV
  yang sama dengan nonce baru sehingga payload lama ditolak tanpa memutus identitas aset.
- Rate limiting global (100 req/menit), autentikasi (5 req/menit), dan API key (100 req/menit per key).
- Upload foto dengan deteksi MIME via magic bytes (tanpa fallback ekstensi), akses file dilindungi JWT.
- Validasi input terpusat (`go-playground/validator`), whitelist kolom sorting (anti SQL-injection).
- API key disimpan sebagai hash dan hanya ditampilkan sekali; webhook ditandatangani HMAC-SHA256.
- Soft-delete, graceful shutdown, security headers, audit trail, Swagger nonaktif di production.
- Container API berjalan non-root, root filesystem read-only, `cap_drop: [ALL]`, resource limit,
  serta Docker secrets dengan fallback environment untuk development.
- Single-origin melalui Nginx dengan TLS, routing API/WebSocket, dan cookie sesi httpOnly yang
  konsisten di `https://localhost`.

**Hasil audit:** ditemukan **5 temuan Kritis**, **7 temuan Tinggi**, dan **8 catatan Sedang/Rendah**.

**Status perbaikan (diperbarui):**
- 🔴 Kritis: **5/5 ✅ Selesai** (Fase 1 roadmap).
- 🟠 Tinggi: **7/7 ✅ Selesai** (Fase 2 roadmap).
- 🟡 Sedang/Rendah: **8/8 ✅ Selesai** (temuan 13, 14, 15, 16, 17, 18, 19, 20).
- Rencana bertahap & to-do per item: lihat `ROADMAP.md` dan bagian **3.4**.

### 1.1 Risiko Terbuka pada Pembaruan 2026-08-19

Pembaruan ini adalah sinkronisasi dokumentasi dan bukan penetration test baru. Risiko berikut
teridentifikasi dari dokumentasi dan inspeksi statis terbatas:

| Severity | Temuan | Dampak | Tindak lanjut |
|----------|--------|--------|---------------|
| **Tinggi** | Contoh kredensial NVR nyata masih muncul di komentar backend dan placeholder form frontend. Dokumentasi telah memakai placeholder. | Siapa pun yang memperoleh source code atau UI dapat mengetahui kredensial perangkat. | Hapus nilai nyata dari source/UI, gunakan placeholder, rotasi kredensial NVR, dan perlakukan riwayat Git sebagai pernah terekspos. |
| **Sedang** | Endpoint Prometheus `/api/metrics` didokumentasikan publik. | Detail traffic, error, koneksi, dan penggunaan storage dapat membantu reconnaissance bila terekspos internet. | **Selesai 2026-10-20:** Nginx memblokir `/api/metrics` dari publik; Prometheus scrape langsung ke `api:3000`. |
| **Sedang** | Backup default disimpan pada volume lokal dengan retensi 7 hari; enkripsi at-rest dan salinan off-site belum didokumentasikan. | Kegagalan host atau akses volume dapat menyebabkan kehilangan atau kebocoran backup. | Tambahkan backup terenkripsi, off-site, verifikasi restore berkala, serta kebijakan akses dan retensi produksi. |

Nilai kredensial nyata sengaja tidak dicantumkan dalam dokumen ini.

### 1.2 Hardening lifecycle QR — 2026-08-20

- [x] Rotasi QR mempertahankan ID CCTV dan menghasilkan ciphertext baru melalui nonce acak.
- [x] Scanner menolak payload valid secara kriptografis yang bukan lagi payload aktif di database.
- [x] Cache gambar label dipisahkan berdasarkan ID dan payload aktif; respons PNG/fetch memakai `no-store`, service worker melewati cache QR, dan versi shell cache dinaikkan untuk menghapus cache lama.
- [x] Regression test backend membuktikan round-trip QR baru dan penolakan QR lama; Node test membuktikan policy cache service worker.
- [x] Runtime sub-gate QR non-production membuktikan label berubah tanpa hard refresh, payload aktual dari PNG baru diterima, payload lama ditolak, teknisi mendapat 403, audit tercatat, dan cleanup staging tidak menyentuh production.
- [ ] Acceptance agregat MT-11 masih menunggu skenario API key dan webhook; pengujian rotasi pada production tetap dilarang.

Bukti runtime: `docs/audits/MT-11-QR-RUNTIME-EVIDENCE-2026-08-20.md`.

### 1.3 Hardening streaming MediaMTX — 2026-08-20

- [x] Image MediaMTX dipin ke `1.20.0-ffmpeg`; konfigurasi memakai field v1.20 `rtspTransports`, `rtspTransport`, `available`, dan `online`.
- [x] Kandidat ICE `webrtcAdditionalHosts` dikonfigurasi melalui environment deployment; nilainya tidak di-hardcode pada source.
- [x] Respons start tidak lagi mengembalikan URL sumber RTSP/decrypted credential ke browser.
- [x] Stop path memakai endpoint v1.20 dan bersifat idempotent ketika path sudah tidak ada.
- [x] Simulator non-production bind loopback membuktikan WHEP, ICE/TCP, decoded frame, reconnect, dan cleanup tanpa membuka service staging ke internet.
- [ ] Firewall UDP 8189, kandidat public, dan kebutuhan TURN/TLS pada production masih harus diverifikasi setelah deployment; jaringan seluler eksternal tetap `NOT RUN`.

Bukti runtime: `docs/audits/TASK-1B-STREAMING-RUNTIME-EVIDENCE-2026-08-20.md`.

### 1.4 Dependency security update — 2026-10-21

- [x] `next` dan `eslint-config-next` dinaikkan dari `16.2.11` ke `16.4.2`; perbaikan mencakup advisory RCE/SSRF/cache poisoning Next.js serta `sharp` dan `postcss`.
- [x] `npm audit --omit=dev` menghasilkan **0 kerentanan** pada dependency produksi.
- [x] `npm audit` penuh menyisakan **5 high** yang seluruhnya berada di rantai dev-only `eslint-config-next -> @next/eslint-plugin-next -> fast-glob -> micromatch -> braces`; `braces` 3.0.3 adalah rilis terbaru dan belum memiliki patch.
- [ ] `npm audit fix --force` tidak dijalankan karena solusinya menurunkan `eslint-config-next` ke `14.2.35` (tidak kompatibel dengan Next 16). Risiko diterima sampai upstream merilis patch; dampaknya hanya pada tooling lint, bukan runtime aplikasi.

### 1.5 Isolasi site dashboard dan field maintenance — 2026-10-21

- [x] `SiteContext` dipasang setelah autentikasi pada `GET /dashboard/summary`, `GET /dashboard/tasks`, dan seluruh grup `/field-maintenance`.
- [x] Ringkasan dashboard (status, tren 6 bulan, overdue) serta daftar tugas difilter ke site aktif; teknisi tetap dibatasi pada `technician_id` miliknya.
- [x] Field maintenance list/detail/laporan PDF dan pembuatan catatan memakai scope site melalui relasi CCTV; CCTV di luar site aktif ditolak `403`.
- [x] Daftar `/sites` dan `/sites/all` memakai scope keanggotaan `user_sites` untuk non-admin; admin tetap melihat seluruh site. `/sites/all` tetap memuat semua site penugasan agar pemilih site multi-site berfungsi.
- [x] Event WebSocket `AUDIT_LOG_ADDED` dibatasi ke koneksi admin; non-admin tidak menerima metadata audit (IP, user-agent, deskripsi aktivitas).
- [x] Regression test mencakup dashboard, field maintenance, daftar site, detail lintas site, dan penolakan non-admin tanpa penugasan. `go test ./...`, `go vet ./...`, dan `go build ./...` lulus.
- [x] Pemeriksaan read-only database deployment: 35/35 CCTV aktif memiliki `site_id` OIKN; tidak ada `site_id` NULL atau baris soft-deleted. Tidak ada data yang diubah.
- [ ] Verifikasi runtime dengan dua site dan akun non-admin belum dijalankan karena database deployment masih memiliki 0 baris `user_sites`; penugasan harus dilakukan admin sebelum uji manual.

---

## 2. Pemahaman Sistem & Alur

### 2.1 Struktur Folder

```text
SIGAP/
├── docker-compose.yml      # 7 service: db, redis, api, frontend, nginx, backup, mediamtx
├── docker-compose.override.yml # Prometheus + Grafana opsional
├── mediamtx.yml            # Relay RTSP → WebRTC/WHEP/HLS, source-on-demand
├── nginx/                  # TLS, security headers, reverse proxy single-origin
├── scripts/                # Backup/restore database dan uploads
├── backend/
│   ├── main.go             # Startup, migration, middleware, routes, worker, shutdown
│   ├── migrations/         # Migration SQL versioned yang di-embed ke binary
│   ├── config/             # PostgreSQL, Redis, migration
│   ├── models/             # CCTV, user, maintenance, site/zone/SLA, integrasi, notifikasi
│   ├── handlers/           # REST API dan background worker
│   ├── middleware/         # JWT/RBAC, CSRF, API key, limiter, site context, security
│   ├── monitoring/         # Metrik Prometheus
│   ├── routes/             # Registrasi rute termasuk WebSocket dan streaming
│   ├── utils/              # Enkripsi, token, notifikasi, MediaMTX, email, audit
│   ├── docs/               # Swagger/OpenAPI generated
│   └── uploads/            # Bukti foto, thumbnail, dan snapshot
├── frontend/
│   ├── public/             # Manifest PWA, service worker, ikon
│   └── src/                # App Router, komponen, context, API client
├── mobile/                 # Expo React Native untuk teknisi lapangan
└── docs/                   # Arsitektur, roadmap, audit, testing, dan manual-test
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
- **Streaming:** API admin-only mengelola path MediaMTX; password RTSP didekripsi hanya saat
  membangun source URL dan tidak boleh dikembalikan pada respons JSON.
- **Integrasi:** API key memakai scope, expiry, revoke, dan rate limit per key. Webhook dikirim
  asinkron dengan signature HMAC-SHA256, retry terbatas, serta log delivery ber-retensi.
- **Multi-site:** Site aktif diteruskan melalui context/filter backend; non-admin dibatasi ke site
  penugasan (default site, `403` bila belum ditugaskan) pada CCTV, zona, dashboard, field maintenance,
  dan daftar site. Admin tetap global. Write operation site, zona, SLA, API key, dan webhook tetap
  dilindungi role admin.
- **Mobile:** Klien Expo menggunakan Bearer token yang disimpan melalui secure storage;
  kebijakan cookie/CSRF web tidak diterapkan pada request Bearer.

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
- **Migrasi DB versioned** — ✅ golang-migrate v4 dengan migration SQL berpasangan dan verifikasi CI.
- **Redis** — ✅ rate-limit dan blacklist token mendukung multi-instance dengan fallback development.
- **Backup** — ✅ cron `pg_dump`, archive uploads, rotasi 7 hari, dan restore script; enkripsi/off-site masih menjadi risiko terbuka.
- **Monitoring** — ✅ endpoint metrics Prometheus dan health check; pembatasan akses metrics di production masih perlu ditetapkan.
- **Docker hardening** — ✅ container API non-root, resource limits, rootfs read-only, capabilities di-drop, port internal tidak dipublish, dan Docker secrets.
- **Reverse proxy** Nginx + TLS + security headers sudah diterapkan; akses resmi kini melalui `https://localhost`.
- **Isi `docs/changelog.md`** sesuai rencana `plan.md` sudah diterapkan untuk milestone proyek.
