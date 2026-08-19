# Panduan Agent AI untuk Proyek SIGAP (Sistem Informasi Gangguan & Aset Pusat)

Selamat datang, Agent AI. Dokumen ini adalah panduan komprehensif mengenai codebase SIGAP. Bacalah dokumen ini secara saksama agar Anda dapat memahami arsitektur, konvensi, dan aturan proyek ini sebelum mulai berkontribusi.

> [!NOTE]
> Dokumen ini diperbarui menyusul Fase 2 (hardening keamanan: migrasi token ke cookie httpOnly + CSRF), Fase 3 (fitur: unifikasi model `assets` → `cctv`, dashboard, profil, laporan PDF, dll), Fase 9 (streaming CCTV real-time: MediaMTX service, API stream admin-only, komponen WHEPPlayer/StreamModal/StreamGridView), dan Sprint 40 (AI Feature Takedown: fitur AI detection dihapus — sistem kembali ke pure CCTV management). Status terbaru setiap item perbaikan & pengembangan ada di `docs/ROADMAP.md` dan `docs/SECURITY-AUDIT.md`.

## 1. Gambaran Umum Sistem
- **Nama**: SIGAP (Sistem Informasi Gangguan & Aset Pusat)
- **Tujuan**: Platform web untuk manajemen, inventarisasi, dan pemeliharaan perangkat CCTV secara terpusat.
- **Arsitektur**: Monorepo dengan 3 komponen utama:
  - **Backend**: REST API yang dibangun menggunakan Go Fiber, berlokasi di direktori `backend/`.
  - **Frontend**: Aplikasi web yang dibangun dengan Next.js 16 dan React 19, berlokasi di direktori `frontend/`.
  - **Database**: PostgreSQL 15, dijalankan melalui Docker.
- **Cache/Shared Store**: Redis 7 Alpine — dipakai untuk rate-limit counter & token blacklist (shared multi-instance, persisten lintas restart); sejak Task 7.2.
- **Deployment**: Menggunakan Docker Compose untuk orkestrasi container.
- **Akses web resmi**: Pada deployment Docker, aplikasi hanya diakses melalui `https://localhost`
  dengan Nginx sebagai reverse proxy. Port internal API/frontend tidak boleh diakses langsung;
  detail alur ada di `docs/ACCESS.md`.

## 2. Aturan Git yang WAJIB Dipatuhi (SANGAT PENTING)
> [!CAUTION]
> Branch `main` dilindungi. JANGAN pernah melakukan commit atau push langsung ke branch `main`.

Setiap pengembangan fitur atau perbaikan _bug_ HARUS menggunakan branch baru.

> [!IMPORTANT]
> **Aturan Deployment Sub-Agent:** Setiap sprint HANYA boleh men-deploy **maksimal 3 sub-agent** secara bersamaan. Rencanakan pembagian jobdesk agar tidak melebihi batas ini. Sub-agent ke-4 dan seterusnya TIDAK akan diizinkan.

**Workflow yang harus Anda ikuti:**
1. Pastikan selalu _up-to-date_ dengan main: `git checkout main && git pull origin main`
2. Buat branch baru: `git checkout -b feature/<nama-fitur>` atau `git checkout -b bugfix/<nama-bug>`
3. Lakukan perubahan pada kode.
4. Lakukan commit dengan standar _Conventional Commits_: `feat(scope): deskripsi` atau `fix(scope): deskripsi`
5. Push ke repositori remote: `git push -u origin feature/<nama-fitur>`
6. Buat Pull Request (PR) di GitHub.
7. User akan me-merge PR tersebut secara mandiri.
8. Setelah PR di-merge dan Anda melakukan pull di lokal, hapus branch lokal: `git branch -D feature/<nama-fitur>`

## 3. Struktur Folder

Berikut adalah penjelasan mengenai struktur direktori dan file penting dalam repositori ini:

### Root `/`
- `docker-compose.yml` — Orkestrasi 7 _service_ (db, redis, api, frontend, nginx, backup, mediamtx) di dalam _network_ `sigap_network`. Sejak Task 7.3 ada service `backup` (cron otomatis) + volume `backups_data`; sejak Task 9.2 (Fase 9) ada service `mediamtx` (streaming CCTV).
- `mediamtx.yml` (Task 9.2) — Konfigurasi MediaMTX: RTSP→WebRTC/WHEP relay, `sourceOnDemand: yes`, `protocols: [tcp]`, REST API `:9997`. Dipakai service `mediamtx` untuk proxy streaming CCTV real-time.
- `docker-compose.override.yml` — **Opsional** (Task 7.4): menambah Prometheus (port 9090) & Grafana (port 3000) untuk visualisasi metrik `/api/metrics`. Di-merge otomatis oleh `docker compose`; lepas/tidak perlu bila monitoring tidak dibutuhkan.
- `scripts/backup.sh` & `scripts/restore.sh` — Backup & restore otomatis database + uploads (Task 7.3). Prosedur lengkap di `docs/BACKUP.md`.
- `backup/Dockerfile` & `backup/entrypoint.sh` — Container `backup` (image `postgres:15-alpine` + `dcron`) yang menjalankan `backup.sh` via cron.
- `makefile` — Kumpulan _script_ untuk menjalankan backend dan frontend secara bersamaan di lingkungan lokal.
- `.gitignore` — Mengabaikan direktori seperti `node_modules`, file `.env`, hasil _build_ binary Go, folder _logs_, dan folder _backups_.

### Backend `backend/`
- `main.go` — Titik masuk (entry point) aplikasi Go. Berisi inisialisasi _logger_, validasi secret wajib (fail-fast), koneksi database, _migrations_ versioned + migrasi data legacy, _seeders_, instansiasi Fiber app (termasuk _Trusted Proxies_ dan CORS), _middleware stack_, pendaftaran _routes_, dan _graceful shutdown_.
- `config/database.go` — Konfigurasi koneksi ke PostgreSQL menggunakan GORM dengan pengaturan _connection pooling_.
- `config/redis.go` — Koneksi Redis (`github.com/redis/go-redis/v9`) untuk rate-limit & blacklist (Task 7.2). `ConnectRedis()` membaca `REDIS_HOST`/`REDIS_PORT`/`REDIS_PASSWORD` (default `redis:6379`); bila Redis tidak tersedia hanya mengeluarkan LOG WARNING + fallback in-memory (bukan fail-fast — aman untuk development tanpa Docker).
- `config/migrate.go` — Dua peran sejak Task 7.1: (1) **migrasi DB versioned** dengan golang-migrate v4 — `RunMigrations(db)` dijalankan otomatis saat startup, plus `MigrationUp/MigrationDown/MigrationVersion` untuk CLI; (2) **migrasi data satu kali** (_one-time_) yang meresolusi _maintenance log_ legacy `asset_id` → `cctv_id` (unifikasi model, Fase 3.6).
- `migrations/` — Folder berisi **file migration SQL versioned** (embed FS via `migrations.go`, dikompilasi ke binary): `000001_init_schema.up.sql`/`.down.sql` membentuk skema awal (10 tabel + index + FK, idempoten). Skema kini dikelola migration files; `AutoMigrate` GORM hanya fase transisi (dikomentari di `main.go`, akan dihapus setelah stabil). Prosedur lengkap di `docs/MIGRATION.md`.
- `cmd/migrate/main.go` — CLI helper manual: `go run cmd/migrate/main.go up|down|version` (env `DB_*` sama dengan aplikasi).
- `models/setup.go` — Definisi struktur `Base` (berisi UUID, timestamps, dan _soft delete_) yang di-_embed_ pada semua model, termasuk _hook_ `BeforeCreate` yang hanya men-generate UUID baru jika ID kosong (`uuid.Nil`). File ini juga mendefinisikan model `User` dan `MaintenanceLog`, termasuk flag nullable `OverdueNotifiedAt` untuk eskalasi overdue satu kali.
- `models/maintenance_schedule.go` — Definisi struktur `MaintenanceSchedule` (Fase 5.4, jadwal pemeliharaan berulang): `CCTVID`, `TechnicianID` (opsional), `ActionTaken`, `Frequency` (`daily`/`weekly`/`monthly`), `NextRunAt`, `IsActive`, `LastGeneratedAt` + konstanta `FrequencyDaily/Weekly/Monthly`.
- `models/daily_digest.go` — Definisi struktur `DailyDigest` (Fase 6.4): tanggal ringkasan WITA yang unik, jumlah tugas/overdue/field-check/perubahan status CCTV, dan summary naratif.
- `models/notification.go` — Definisi struktur `Notification` (notifikasi in-app per-user, Fase 5.1): `UserID`, `Type`, `Title`, `Message`, `ReferenceID`, `ReferenceType`, `IsRead` + konstanta tipe (`MAINTENANCE_ASSIGNED`, `MAINTENANCE_REASSIGNED`, `MAINTENANCE_UNASSIGNED`, `MAINTENANCE_COMPLETED`, `MAINTENANCE_OVERDUE`, `FIELD_CHECK_ATTENTION`); index komposit `(user_id, is_read)`.
- `models/push_subscription.go` (Task 8.3) — Definisi struktur `PushSubscription` (langganan Web Push per user): `UserID` (FK), `Endpoint`, `KeyP256DH`, `KeyAuth`; index `(user_id)`. Dipakai util `SendPush` untuk notifikasi push browser.
- `models/cctv.go` — Definisi struktur `CCTV`, termasuk `QrPayload` yang dienkripsi menggunakan AES-256 dan `HealthScore` (skor kesehatan 0–100, default 100).
- `models/audit_log.go` — Definisi struktur `AuditLog` (riwayat aktivitas penting pengguna).
- `models/seeder.go` — _Script_ untuk me-seed admin _default_ (hanya jika belum ada, tidak me-reset password tiap startup).
- `handlers/` — Berisi _handler_ untuk setiap modul: `auth`, `cctv` (termasuk endpoint skor kesehatan), `maintenance`, `field_maintenance` (termasuk laporan PDF per pengecekan), `user`, `dashboard` (termasuk endpoint kinerja teknisi dan daftar digest admin), `digest_generator` (goroutine ringkasan operasional harian pukul 06:00 WITA), `upload`, `audit`, `report`, `notification`, `health` (diperluas sejak Task 7.4: response time DB, cek Redis, ukuran folder uploads), `push` (Task 8.3: `POST /api/push/subscribe` & `DELETE /api/push/unsubscribe` untuk Web Push), `maintenance_schedule` (CRUD jadwal berulang, Fase 5.4), `schedule_runner` (goroutine generator jadwal — berjalan sekali saat startup lalu tiap jam membuat `MaintenanceLog` otomatis, idempotent per hari per jadwal), `overdue_checker` (goroutine eskalasi tugas overdue — berjalan saat startup lalu tiap jam, satu kali per log sampai reschedule), `health_score` (kalkulator skor kesehatan CCTV saat startup lalu tiap 6 jam), `stream` (Task 9.3b — `StartStream`/`StopStream`/`StartAllStreams`/`StopAllStreams`/`GetStreamStatus`/`GetStreamURL` untuk streaming CCTV real-time via MediaMTX; `rtsp_password` terenkripsi & tidak pernah dikembalikan ke response), `zone` (Task 11.4 — CRUD `/api/zones`, `GET /api/zones/all`, `POST /api/zones/:id/assign-cctvs`, `POST /api/zones/:id/assign-users`; admin-only untuk tulis), `sla` (Task 11.5 — CRUD `/api/sla-policies` admin-only, `GET /api/dashboard/sla` admin, serta goroutine `RunSLADeadlineChecker` yang memicu event `SLA_WARNING` ke admin saat tugas melewati 75% target penyelesaian), `api_key` (Task 12.1 — `POST /api/api-keys` generate key tampil 1×, `GET /api/api-keys`, `POST /api/api-keys/:id/revoke`; admin-only), dan `webhook` (Task 12.3 — CRUD `/api/webhooks` admin-only, `POST /api/webhooks/:id/test`, `GET /api/webhooks/:id/deliveries`; antrean channel + worker `StartWebhookWorker`; dispatch event via `DispatchWebhook` dengan header `X-Signature-256` HMAC-SHA256 dan retry 3×).
- `monitoring/` (Task 7.4) — Collector metrik Prometheus ringan **tanpa library pihak ketiga**: `metrics.go` (counter request, histogram durasi, gauge DB pool/WS aktif/ukuran uploads + `Handler` untuk `GET /api/metrics` dalam format text exposition), `collector.go` (`StartCollector` goroutine memperbarui gauge tiap interval).
- `middleware/auth.go` — Fungsi `Protected()` untuk memvalidasi JWT token, pengecekan blacklist,
  validasi versi token, dan `RequireRole()` untuk pengecekan _Role-Based Access Control_ (RBAC).
- `middleware/limiter.go` — Pembatasan laju _request_ (Rate Limiter). Sejak Task 7.2 memakai counter fixed-window kustom berbasis Redis `INCR`/`EXPIRE` (key `limiter:{ip}:{endpoint}`) dengan fallback in-memory. Terdapat `GlobalLimiter` (100 req/menit) dan `AuthLimiter` (khusus _endpoint_ autentikasi, 5 req/menit).
- `middleware/csrf.go` — Perlindungan CSRF pola _double-submit_ untuk permintaan mutasi yang terautentikasi via cookie.
- `middleware/security.go` — _Security headers_ (CSP & HSTS aktif hanya di produksi).
- `middleware/logger.go` & `middleware/recovery.go` — Logging terstruktur (Zap) dan pemulihan dari _panic_.
- `middleware/metrics.go` (Task 7.4) — Mencatat setiap request: increment counter `sigap_http_requests_total{method,path,status}` + observe durasi ke histogram `sigap_http_request_duration_seconds`; endpoint `/api/metrics` dikecualikan.
- `routes/` — Registrasi rute untuk masing-masing modul (auth, cctv, maintenance, field maintenance, user, upload, audit, dashboard, notification, maintenance_schedule, stream, websocket).
  > `routes/stream.go` (Task 9.3b) — grup `/streams` admin-only (`Protected()` + `RequireRole("admin")`): `GET /status`, `POST /start-all`, `POST /stop-all`, `POST /:id/start`, `POST /:id/stop`, `GET /:id/url` (rute statis didaftarkan sebelum dinamis). Registrasi di `main.go` + goroutine auto-start stream aktif saat boot.
  > [!IMPORTANT]
  > **PENTING:** Rute statis (seperti `/template`, `/import`, `/bulk-delete`) HARUS dideklarasikan SEBELUM rute dinamis (seperti `/:id`) untuk menghindari _collision_ atau tertimpa oleh parameter Fiber.
- `utils/encryption.go` — Fungsi `EncryptAES256` dan `DecryptAES256` berbasis AES-256-GCM. Kunci diderivasi menggunakan _hash_ SHA-256 dari _environment variable_ `AES_SECRET_KEY`.
- `utils/jwt.go` — Fungsi untuk me-generate, memvalidasi JWT token (HMAC-SHA256, kedaluwarsa 5 jam, klaim unik `jti`/`iat`), dan memvalidasi versi token (`tv`) terhadap database.
- `utils/session.go` — Pengaturan cookie sesi httpOnly (`sigap_session`) & cookie CSRF (`sigap_csrf`), termasuk pembersihan path legacy `/api`.
- `utils/blacklist.go` — Penyimpanan token yang di-_revoke_. Sejak Task 7.2 memakai Redis SET (`SADD`/`SISMEMBER`/`SREM` + TTL otomatis) via interface `TokenBlacklist` (persisten lintas restart & multi-instance), dengan fallback in-memory per-proses bila Redis tidak tersedia.
- `utils/csrf.go` — Generate token CSRF dan ekstraksi token sesi (prioritas cookie → Bearer).
- `utils/audit.go` — Pencatatan audit log secara _async_ dengan retry terbatas.
- `utils/notify.go` — Util notifikasi in-app (Fase 5.1): `NotifyUser`/`NotifyRole` — simpan `Notification` secara _async_ (pola `LogAudit`, retry terbatas) lalu broadcast event `NOTIFICATION_CREATED` secara terarah. Sejak Task 8.3 juga memicu `SendPush` (push browser) setelah notifikasi in-app berhasil.
- `utils/push.go` (Task 8.3) — Util Web Push (VAPID): `SendPush` mengirim notifikasi push ke semua subscription user secara async (fallback in-app bila VAPID kosong); `SendPushToSubscription` memakai `github.com/SherClockHolmes/webpush-go`. Env `VAPID_PUBLIC_KEY`/`VAPID_PRIVATE_KEY`/`VAPID_EMAIL`.
- `utils/websocket.go` — Hub manajemen koneksi WebSocket + broadcast event real-time. Sejak Fase 5.1 setiap koneksi menyimpan `WSClientInfo{UserID, Role}`; tersedia `BroadcastToUser` (per-user) & `BroadcastToRole` (per-role) selain `BroadcastEvent` global.
- `utils/pagination.go` — Helper utilitas untuk fitur paginasi (dengan cap maksimum `MAX_PAGE_SIZE`) dan proteksi _SQL injection_ pada kolom _sorting_.
- `utils/validator.go` — Validator kustom (mis. kebijakan password kuat).
- `utils/logger.go` — Sistem logging ganda menggunakan Zap (menyimpan log ke file JSON dan menampilkannya di konsol dengan warna). Sejak Task 7.6 memakai `lumberjack` untuk rotasi otomatis (100 MB/file, 5 backup, 30 hari) dan level log dari env `LOG_LEVEL` (default `info`).
- `utils/secrets.go` (Task 7.5) — Helper `GetSecret`/`RequireSecret`: membaca Docker secret sebagai FILE (`/run/secrets/<nama>`) dengan fallback ke environment variable; dipakai untuk `DB_PASSWORD`, `JWT_SECRET`, `AES_SECRET_KEY`.
- `utils/rtsp.go` (Task 9.1) — Tipe `StreamType` (`MainStream=1`/`SubStream=2`); fungsi `BuildRTSPURL` (URL Hikvision ISAPI dengan `ChannelID = channel_port × 100 + stream_type`), `BuildStreamPath`, `GetRTSPPort` (default sub-stream); menangani edge case channel ≤ 0. Env NVR global: `RTSP_NVR_HOST`/`RTSP_NVR_PORT`/`RTSP_NVR_USER`.
- `utils/mediamtx.go` (Task 9.3a) — Struct `MediaMTXClient`: HTTP client ke `MEDIAMTX_API_URL` (`http://mediamtx:9997/v3/`) dengan method `AddPath`, `RemovePath`, `GetPath`, `ListPaths` untuk mengelola path streaming RTSP→WHEP/HLS di MediaMTX.
- `uploads/` — Direktori penyimpanan foto bukti pemeliharaan (hasil normalisasi JPEG, tanpa EXIF). Akses via endpoint yang dilindungi JWT. Di-backup otomatis oleh `scripts/backup.sh` (Task 7.3); prosedur backup & restore di `docs/BACKUP.md`.
- `docs/` — Direktori yang berisi dokumentasi Swagger yang di-generate secara otomatis. Jalankan `swag init` setelah Anda memodifikasi anotasi pada _handlers_.

### Frontend `frontend/`
- `src/app/layout.tsx` — _Root layout_ yang mencakup `ThemeProvider`, `AuthProvider`, dan registrasi PWA (`ServiceWorkerRegister`). Sejak Task 8.1 mengekspor `metadata.manifest`, ikon, `appleWebApp`, dan `viewport` (theme-color).
- `public/manifest.json`, `public/sw.js`, `public/icons/` (Task 8.1 PWA + Task 8.3 push) — web manifest (SIGAP, standalone, warna tema, ikon 192/512) dan service worker (cache-first shell, network-first API, serta handler `push`/`notificationclick` untuk notifikasi push browser).
- `src/components/ServiceWorkerRegister.tsx` (Task 8.1) — komponen client yang mendaftarkan service worker (`/sw.js`) di `useEffect` (aman SSR).
- `src/app/globals.css` — Konfigurasi Tailwind CSS v4 (melalui directive `@theme`), custom CSS variables (untuk _dark/light mode_), animasi UI saat scan, dan _style override_ khusus untuk `html5-qrcode`.
- `src/app/page.tsx` — Halaman akar yang hanya me-_redirect_ pengguna ke `/dashboard`.
- `src/app/dashboard/page.tsx` — Halaman Dashboard utama: manajemen aset CCTV (tabel data, filter pencarian, CRUD modal, tombol quick maintenance) + komponen `DashboardAnalytics`.
- `src/app/sebaran-cctv/page.tsx` — Halaman Peta Sebaran CCTV: visualisasi lokasi perangkat CCTV di kawasan IKN beserta ringkasan status (total/aktif/pemeliharaan), diperbarui real-time via WebSocket. Sejak Task 9.5 (Fase 9) mendukung toggle view Peta/Grid, filter "Streaming Aktif", status bar "N/35 kamera streaming aktif", dan membuka `StreamModal` saat marker diklik (admin).
- `src/app/cctv/page.tsx` — Halaman Master Data CCTV. Berisi tabel detail lengkap termasuk badge skor kesehatan, _checkbox_ untuk seleksi massal (bulk select), modal penampil QR Code, fitur cetak QR, pemindai QR berbasis kamera, fitur _import_ via Excel, dan _download template_. Sejak Task 9.6 (Fase 9) mendukung section form "Konfigurasi Streaming" di modal edit CCTV (`channel_port` 1–35, `rtsp_password` dengan toggle visibility, `stream_enabled` checkbox, preview URI RTSP) + kolom tabel master ikon streaming aktif.
- `src/app/maintenance/page.tsx` — Halaman Log Pemeliharaan. Berisi tabel data historis, filter, CRUD modal, fitur unggah bukti foto pemeliharaan, penugasan ke teknisi (tombol "Tugaskan"), dan pengalihan tugas (tombol "Alihkan" untuk log `ongoing`, Fase 5.3 — menampilkan peringatan nama teknisi saat ini sebelum submit).
- `src/app/field-maintenance/page.tsx` — Halaman Pengecekan Lapangan: daftar dan modal detail checklist/foto bukti, tombol "Cetak Laporan" untuk Admin yang mengunduh PDF per-record, serta (Task 8.4) bagian "Lokasi Pengecekan" berupa mini-map bila koordinat tersedia.
- `src/components/FieldCheckLocationMap.tsx` (Task 8.4) — mini-map react-leaflet yang menampilkan marker CCTV (biru) + marker "Lokasi Pengecekan" (oranye) untuk membandingkan posisi teknisi dengan CCTV.
- `src/components/FieldMaintenanceModal.tsx` — wizard pengecekan lapangan; sejak Task 8.4 menangkap GPS (`navigator.geolocation`, opsional) dengan indikator "Lokasi tercatat" dan mengirim `check_latitude`/`check_longitude`.
- `src/app/schedules/page.tsx` — Halaman Jadwal Pemeliharaan (Fase 5.4, khusus admin): CRUD jadwal berulang, toggle aktif/nonaktif, filter status, paginasi; menu sidebar "Jadwal".
- `src/app/kinerja-teknisi/page.tsx` — Dashboard Kinerja Teknisi (Fase 6.2, khusus admin): filter rentang tanggal, chart completion rate SVG, tabel ranking sortable, dan metrik overdue/field-check/quality.
- `src/app/ringkasan/page.tsx` — Timeline Ringkasan Operasional (Fase 6.4, khusus admin): kartu per tanggal, metrik tugas/field-check/perubahan status, summary naratif, dan paginasi.
- `src/app/kalender/page.tsx` — Kalender Jadwal Pemeliharaan (Task 8.2, admin + teknisi): kalender bulanan (grid 7 kolom), dot warna status per hari, panel detail tugas, filter per teknisi (admin). Data dari `/maintenance-logs?start_date=&end_date=` dikelompokkan per `scheduled_date`.
- `src/components/CalendarGrid.tsx` (Task 8.2) — komponen grid kalender bulanan CSS mandiri (tanpa dependency chart eksternal): navigasi bulan, dot status, legend; helper `toDayKey`/`formatLongDate`/`STATUS_DOT_CLASS`/`STATUS_META`.
- `src/app/users/page.tsx` — Halaman manajemen pengguna (khusus admin): CRUD user, reset password.
- `src/app/audit-logs/page.tsx` — Halaman audit trail (khusus admin): daftar aktivitas + ekspor CSV.
- `src/app/profile/page.tsx` — Halaman profil pengguna (data diri + tautan ganti password + bagian "Notifikasi Push" / opt-in push browser, Task 8.3).
- `src/components/PushNotificationOptIn.tsx` (Task 8.3) — komponen opt-in notifikasi push browser: minta izin, subscribe (`pushManager.subscribe`), kirim ke `POST /api/push/subscribe`, dan nonaktifkan (`DELETE /api/push/unsubscribe`).
- `src/app/pemberitahuan/page.tsx` — Halaman Pemberitahuan (Fase 5.1): riwayat lengkap notifikasi milik user yang login (paginasi, filter belum dibaca, tandai semua dibaca, klik item → navigasi ke `/maintenance`/`/field-maintenance`). Scope per-user dijamin backend dari klaim JWT.
- `src/app/change-password/page.tsx` — Halaman self-service ganti password (wajib saat login pertama).
- `src/app/login/page.tsx` — Halaman antarmuka login dengan layout split-screen yang responsif.
- `src/proxy.ts` — _Middleware_ server-side Next.js: validasi sesi & role sebelum halaman admin diproses (guard tambahan di sisi server).
- `src/components/DashboardLayout.tsx` — Komponen _layout sidebar_ responsif dengan navigasi menu (termasuk "Kinerja Teknisi" dan "Ringkasan" untuk admin), _toggle dark mode_, info profil user, dan tombol _logout_.
- `src/components/NotificationBell.tsx` — Komponen lonceng notifikasi in-app (Fase 5.1): **hanya dirender di halaman Dashboard**; badge jumlah belum dibaca, dropdown 3 notifikasi teratas, tombol "Lihat lainnya" → `/pemberitahuan`, tandai semua dibaca, toast real-time via WebSocket `NOTIFICATION_CREATED`, fallback polling 60 detik.
- `src/components/ProtectedRoute.tsx` — Komponen _High-Order Guard_ yang melakukan otomatisasi _redirect_ ke `/login` jika pengguna belum terautentikasi.
- `src/components/DashboardAnalytics.tsx` — Komponen grafik/statistik dashboard.
- `src/components/CCTVMap.tsx` — Visualisasi lokasi CCTV pada peta, dirender di halaman `/sebaran-cctv`; warna marker dan badge popup mengikuti skor kesehatan; sejak Task 9.5 (Fase 9) mendukung prop `onMarkerClick` + indikator streaming aktif pada marker.
- `src/components/WHEPPlayer.tsx` (Task 9.4) — player WebRTC/WHEP untuk video live CCTV (browser-native `RTCPeerConnection`, zero SDK); WHEP signaling via `POST /stream/{path}/whep`; auto-reconnect exponential backoff; status overlay (connecting/live/error/reconnecting); toggle fullscreen; toggle kualitas SD sub-stream ↔ HD main-stream; indikator 🔴 LIVE; SSR-safe (dynamic import).
- `src/components/StreamModal.tsx` (Task 9.4b) — modal overlay saat marker CCTV diklik: header nama/kode/badge status + skor kesehatan, body `WHEPPlayer` embed, footer lokasi/IP/channel port, tombol Start/Stop stream, link ke Master CCTV.
- `src/components/StreamGridView.tsx` (Task 9.5) — grid view multi-stream responsif; setiap sel berisi `WHEPPlayer` mini; sel tanpa streaming menampilkan placeholder.
- `src/components/AuthenticatedImage.tsx` — Komponen `<img>` yang memuat file gambar melalui endpoint yang dilindungi JWT.
- `src/components/Pagination.tsx` — Komponen navigasi halaman reusable.
- `src/context/AuthContext.tsx` — Manajemen state sesi dan pengambilan profil `/auth/me`.
  **Sesi disimpan dalam cookie httpOnly `sigap_session` (tidak dapat dibaca JavaScript).**
- `src/context/ThemeContext.tsx` — Manajemen state untuk tema UI (disimpan dalam localStorage dengan key `sigap_theme`).
- `src/context/WebSocketContext.tsx` — Koneksi WebSocket ke `/api/ws` untuk menerima broadcast event real-time.
- `src/lib/api.ts` — _Wrapper_ untuk fungsi `fetch`: mengirim cookie sesi (`credentials: 'include'`), menyematkan header `X-CSRF-Token` untuk permintaan mutasi, serta menangani _redirect_ ke `/login` ketika server mengembalikan status HTTP 401.

## 4. Model Database

Semua tabel database dikelola melalui migration files (golang-migrate v4, sejak Task 7.1) yang
bersumber dari definisi struct GORM di folder `models/`. Untuk perubahan skema, **jangan ubah
AutoMigrate** (sudah dikomentari) — buat migration baru di `backend/migrations/` (panduan:
`docs/MIGRATION.md`). Berikut detail masing-masing struktur entitas:

### Base (Embedded di semua model)
Struktur bawaan yang ada pada semua model:
- `ID`: Bertipe `uuid.UUID` (Berlaku sebagai primary key).
  > [!IMPORTANT]
  > ID ini di-generate otomatis via _hook_ `BeforeCreate` **HANYA** jika nilainya adalah `uuid.Nil`.
- `CreatedAt`, `UpdatedAt`: Standar _timestamp_.
- `DeletedAt`: Menggunakan `gorm.DeletedAt` untuk mengimplementasikan fitur _soft delete_.

### User
Menyimpan data pengguna aplikasi:
- `Username` (unique)
- `PasswordHash` (disimpan dalam bentuk bcrypt, selalu disembunyikan/diabaikan saat konversi ke JSON)
- `FullName`
- `Email` (unique)
- `Role` (enum: `admin` / `teknisi`)
- `TokenVersion` (versi token JWT; dinaikkan saat password di-reset / role diubah untuk mencabut token lama — disematkan dalam klaim JWT `tv`)
- `MustChangePassword` (memaksa ganti password saat login pertama, mis. akun baru / password di-reset Admin)

### CCTV
> [!IMPORTANT]
> Sejak Fase 3.6, CCTV adalah **satu-satunya sumber kebenaran (single source of truth)** untuk aset. Model `assets` (legacy) beserta handler/rutenya telah **dihapus**; `MaintenanceLog` kini merujuk langsung ke `cctv_id`. Migrasi data `asset_id` → `cctv_id` terjadi otomatis sekali saat startup (`config/migrate.go`).
> Jangan membuat/menghidupkan kembali model `assets` — semua relasi harus melalui `CCTV`.

Menyimpan data spesifik perangkat CCTV yang dienkripsi ID-nya untuk keperluan QR:
- `KodeCCTV` (unique)
- `IDChannel`
- `Nama`
- `LokasiArea`
- `Longitude`, `Latitude`
- `JenisCCTV`
- `SumberListrik`
- `TipeKoneksi`
- `IndoorOutdoor`
- `Pemilik`
- `Status`
- `IPNVR`, `IPCamera`
- `Keterangan`
- `QrPayload` (unique, berisi format string dari UUID perangkat yang **dienkripsi menggunakan algoritma AES-256**).
- `HealthScore` (integer 0–100, default 100; diperbarui kalkulator skor kesehatan Fase 6.1).
- `ChannelPort` (integer, JSON `channel_port`) — port channel NVR (1–35) untuk streaming (Task 9.1).
- `RTSPPassword` (text, JSON `-` / tidak dikembalikan) — password RTSP **terenkripsi AES-256** (Task 9.1).
- `StreamEnabled` (boolean, JSON `stream_enabled`, default `false`) — tanda CCTV aktif streaming (Task 9.1).

### MaintenanceLog
Menyimpan data catatan pemeliharaan:
- `CCTVID` (Foreign Key → `cctvs`, serialized sebagai `cctv_id`)
- `TechnicianID` (Foreign Key → `users`, relasi `technician`)
- `AssignedAt` (nullable; waktu mulai penugasan, dipakai untuk durasi kinerja teknisi; log lama fallback ke `created_at`)
- `ActionTaken`
- `Notes`
- `Status` (enum: `scheduled` / `ongoing` / `completed` / `failed`)
- `ScheduledDate`
- `OverdueNotifiedAt` (nullable; waktu terakhir notifikasi `MAINTENANCE_OVERDUE` dikirim)
- `CompletedAt`
- `ImageUrl` (jalur URL gambar bukti pemeliharaan)
- `MaintenanceScheduleID` (Foreign Key → `maintenance_schedules`, nullable — menunjuk jadwal berulang yang menghasilkan log ini; dipakai generator untuk idempotensi per hari)

### MaintenanceSchedule (Fase 5.4)
Menyimpan jadwal pemeliharaan berulang:
- `CCTVID` (Foreign Key → `cctvs`)
- `TechnicianID` (Foreign Key → `users`, nullable — bila diisi, log hasil generator langsung berstatus `ongoing` + notifikasi `MAINTENANCE_ASSIGNED`)
- `ActionTaken`
- `Frequency` (enum: `daily` / `weekly` / `monthly`)
- `NextRunAt` (waktu eksekusi berikutnya; diatur ulang otomatis bila frekuensi diubah)
- `IsActive` (default `true`; jadwal nonaktif tidak diproses generator)
- `LastGeneratedAt` (waktu terakhir generator membuat log)
- `SLAPolicyID` (Foreign Key → `sla_policies`, nullable — kebijakan SLA yang mengatur target waktu tugas hasil jadwal ini)

### Zone (Task 11.4)
Menyimpan wilayah operasional penempatan CCTV & teknisi:
- `Nama`, `Deskripsi`
- Relasi one-to-many `CCTVs` (via kolom `cctvs.zone_id`)
- Relasi many-to-many `Users` (via join table `user_zones`, model `UserZone`)

### SLAPolicy (Task 11.5)
Kebijakan Service Level Agreement untuk target waktu layanan:
- `Nama`, `JenisTugas` (enum: `corrective` / `preventive` / `field_check`)
- `TargetJamRespon` (int, jam), `TargetJamSelesai` (int, jam)

### APIKey (Task 12.1)
Kredensial integrasi eksternal untuk memanggil API tanpa sesi web:
- `Name`, `KeyHash` (bcrypt, tidak pernah dikembalikan), `KeyPrefix`
- `Scopes` (jsonb array: `cctv:read` / `maintenance:read` / `health:read`)
- `ExpiresAt`, `CreatedBy`, `LastUsedAt`, `RevokedAt`

### WebhookEndpoint & WebhookDelivery (Task 12.3)
Pengiriman event ke sistem eksternal:
- `WebhookEndpoint`: `Name`, `URL`, `Events` (jsonb array), `Secret` (HMAC), `IsActive`, `CreatedBy`
- `WebhookDelivery`: `WebhookEndpointID`, `Event`, `Payload`, `ResponseCode`, `ResponseBody`, `Latency`, `Status` (pending/success/failed), `Attempt` (maks 3)

### FieldMaintenance (Pengecekan Lapangan)
Menyimpan satu catatan pengecekan CCTV oleh teknisi di lapangan:
- `CCTVID` (Foreign Key → `cctvs`), `TechnicianID` (Foreign Key → `users`)
- `OverallStatus` (`ok` / `attention`), `Notes`, `CompletedAt`
- `CheckLatitude` / `CheckLongitude` (Task 8.4, nullable string) — koordinat GPS lokasi pengecekan (opsional; `check_latitude`/`check_longitude` dalam JSON)
- `MaintenanceLogID` (nullable; penghubung bila pengecekan berasal dari tugas terjadwal)
- `FieldMaintenanceCheck`: nama item, hasil (`ok` / `not_ok`), dan catatan per item
- `FieldMaintenanceImage`: URL foto bukti yang dilindungi endpoint upload

### AuditLog
Menyimpan riwayat aktivitas penting pengguna (dipakai pada halaman audit trail):
- `UserID` (Foreign Key → `users`, nullable), `Username`, `UserRole`
- `Action`, `Module`, `Description`
- `IPAddress`, `UserAgent`

### Notification (Fase 5.1)
Notifikasi in-app per-pengguna (bell icon). Satu baris = satu penerima:
- `UserID` (Foreign Key → `users`, penerima)
- `Type` (konstanta `NotificationType*`: `MAINTENANCE_ASSIGNED`, `MAINTENANCE_REASSIGNED`, `MAINTENANCE_UNASSIGNED`, `MAINTENANCE_COMPLETED`, `MAINTENANCE_OVERDUE`, `FIELD_CHECK_ATTENTION`, `DAILY_DIGEST_AVAILABLE`)
- `Title`, `Message` (teks untuk dropdown bell)
- `ReferenceID` (UUID record terkait) + `ReferenceType` (`"maintenance-log"` / `"field-maintenance"` / `"daily-digest"`) — penentu rute navigasi frontend
- `IsRead` (default `false`)
- Index komposit `(user_id, is_read)`

## 5. Pola API & Konvensi

Pastikan Anda mengikuti konvensi berikut saat memodifikasi atau membuat API baru:
- **Response Format Standar**:
  ```json
  { 
    "status": "success" | "error",
    "data": any,
    "message": "string",
    "errors": [] 
  }
  ```
- **Paginasi**: Menggunakan _query parameters_ `page`, `limit`, `sort_by`, dan `order`. Respons berhasil harus menyertakan properti meta `pagination: { page, limit, total_rows, total_pages }`.
- **Filter Pencarian**: Tiap modul mendefinisikan _query parameters_ spesifiknya sendiri (seperti `status`, `search`, `vendor`, `jenis_cctv`, dll).
- **Autentikasi**: Sesi utama disimpan dalam **cookie httpOnly `sigap_session`** yang dikirim otomatis oleh browser (mode web). Untuk klien API / Swagger, header `Authorization: Bearer <token>` tetap didukung sebagai _fallback_ (`utils.ExtractTokenString` memprioritaskan cookie, lalu Bearer).
- **CSRF**: Semua permintaan mutasi (POST/PUT/PATCH/DELETE) yang terautentikasi via cookie **wajib** menyertakan header `X-CSRF-Token` yang bernilai sama dengan cookie non-httpOnly `sigap_csrf` (pola _double-submit_). Middleware CSRF melewati metode aman (GET/HEAD/OPTIONS), login publik, logout idempotent, dan permintaan yang memakai Bearer. Token CSRF diperoleh dari respons `POST /api/auth/login`.
- **Otorisasi (Role Check)**: Gunakan _middleware_ `RequireRole("admin")` pada rute yang melibatkan operasi _write_ atau _delete_ (hanya admin yang dapat memodifikasi data inti). Ekspor/laporan untuk teknisi dibatasi ke data miliknya (Fase 3.9).
- **WebSocket**: Koneksi `/api/ws` **wajib terautentikasi** — token diambil dari cookie `sigap_session` (dikirim otomatis browser same-origin) dengan fallback query parameter `token`. Hub mem-broadcast event `CCTV_UPDATED`, `MAINTENANCE_CREATED`, `MAINTENANCE_UPDATED`, `AUDIT_LOG_ADDED` (global), serta `NOTIFICATION_CREATED` (terarah per-user/per-role via `BroadcastToUser`/`BroadcastToRole` — Fase 5.1).
- **Notifikasi In-App** (Fase 5.1): Endpoint `/api/notifications` — `GET /` (list + filter `unread` + paginasi), `GET /unread-count`, `PUT /:id/read` (ownership check → 403), `PUT /read-all`. Semua scope per-user dari klaim JWT; rute statis didaftarkan sebelum dinamis. Pembuatan notifikasi tidak dilakukan handler secara manual — gunakan `utils.NotifyUser`/`utils.NotifyRole`.
- **Skor Kesehatan CCTV** (Fase 6.1): `GET /api/cctv/:id/health-score` protected untuk semua role; respons berisi `health_score`, `breakdown` lima metrik, dan `updated_at`. Kolom `health_score` juga tersedia pada list CCTV dan dapat dipakai sebagai `sort_by`.
- **Kinerja Teknisi** (Fase 6.2): `GET /api/dashboard/technician-performance` hanya untuk admin; parameter `start_date`/`end_date` default 30 hari terakhir; respons berupa array metrik tugas, durasi assign → complete, overdue, field check, dan quality score.
- **Laporan Pengecekan Lapangan** (Fase 6.3): `GET /api/field-maintenance/:id/report-pdf` protected untuk Admin dan teknisi pemilik record; mengembalikan PDF berkepala surat IKN dengan detail CCTV, teknisi, waktu, checklist, foto bukti, dan catatan umum.
- **Ringkasan Operasional** (Fase 6.4): `GET /api/dashboard/digest` hanya untuk admin; respons paginasi `DailyDigest` terbaru dahulu. Generator berjalan setiap pukul 06:00 WITA untuk jendela 24 jam dan mengirim notifikasi `DAILY_DIGEST_AVAILABLE` ke admin.
- **Zona** (Task 11.4): `/api/zones` — CRUD admin-only untuk tulis, `GET /api/zones` & `GET /api/zones/all` (dropdown) untuk semua role login; `POST /api/zones/:id/assign-cctvs` (bulk-assign CCTV, menimpa penugasan zona) & `POST /api/zones/:id/assign-users` (assign teknisi, hanya role `teknisi`, menimpa penugasan zona) admin-only. `GET /api/cctv` mendukung filter `?zone_id=<uuid>`.
- **SLA** (Task 11.5): `/api/sla-policies` — CRUD admin-only untuk tulis, baca untuk semua role login. `GET /api/dashboard/sla` hanya untuk admin → `monthly_compliance` (bar chart), `violations` (tabel pelanggaran), `technician_ranking`. Goroutine `RunSLADeadlineChecker` (startup + tiap 1 jam) mem-broadcast `SLA_WARNING` ke admin saat tugas melewati 75% target penyelesaian.
- **API Key** (Task 12.1): `/api/api-keys` — admin-only (generate, list, revoke). Konsumen eksternal memanggil endpoint dengan header `X-API-Key: <key>`. Middleware `APIKeyAuth`/`APIKeyOrProtected` memvalidasi hash bcrypt, cek revoked/expired/scope, rate limit 100 req/menit per key. `GET /api/cctv` dapat diakses via API key scope `cctv:read`; `GET /api/health` publik (juga via scope `health:read`).
- **Webhook** (Task 12.3): `/api/webhooks` — admin-only CRUD, `POST /api/webhooks/:id/test`, `GET /api/webhooks/:id/deliveries`. Event yang didukung: `cctv.status_changed`, `maintenance.completed`, `maintenance.overdue`, `health_score.changed`, `stream.down`. Pengiriman asinkron via antrean channel + worker `StartWebhookWorker` (dipanggil di `main.go`); header `X-Signature-256` (HMAC-SHA256 dari secret), retry 3× exponential backoff (1s/5s/25s).
- **Streaming CCTV** (Fase 9, Task 9.3): seluruh endpoint streaming **admin-only** (`Protected()` + `RequireRole("admin")`) di bawah grup `/api/streams` — `POST /api/streams/:id/start` (decrypt `rtsp_password` → build URL RTSP → register path MediaMTX → return WHEP URL), `POST /api/streams/:id/stop`, `GET /api/streams/status` (gabung data DB + info path MediaMTX), `GET /api/streams/:id/url` (WHEP + HLS), `POST /api/streams/start-all`, `POST /api/streams/stop-all`. `rtsp_password` disimpan terenkripsi (AES-256) dan **tidak pernah** dikembalikan di respons.
- **Metrik Prometheus** (Task 7.4): `GET /api/metrics` **publik** mengembalikan metrik dalam format text exposition Prometheus (`text/plain; version=0.0.4`) — counter `sigap_http_requests_total`, histogram `sigap_http_request_duration_seconds`, gauge `sigap_db_pool_open/idle_connections`, `sigap_ws_active_connections`, `sigap_upload_storage_bytes`. Implementasi di `monitoring/` (tanpa library pihak ketiga); jangan menambah library prometheus kecuali disetujui.
- **Soft Delete**: Semua proses penghapusan data secara standar menggunakan mekanisme _soft delete_ dari GORM yang akan mengisi kolom `deleted_at`.

## 6. Gotchas & Catatan Teknis Penting

Perhatikan dengan sangat saksama _edge cases_ dan potensi _pitfall_ berikut:

### GORM BeforeCreate Hook
- _Hook_ `BeforeCreate` yang berada di `models/setup.go` hanya boleh melakukan _generate_ UUID baru jika parameter `base.ID == uuid.Nil`.
- **JANGAN ubah logika ini!** Jika logika dihapus atau diganti dengan cara paksa (_force overwrite_) `base.ID = uuid.New()` tanpa pengecekan terlebih dahulu, seluruh data `CCTV` akan mengalami _mismatch_ (ketidakcocokan) antara `ID` dan nilai enkripsi `QrPayload`. Hal ini akan menyebabkan _QR scanner_ gagal mendekripsi dan mengenali perangkat CCTV.

### Fiber Route Ordering
- Rute statis HARUS selalu didaftarkan **SEBELUM** rute dinamis (contoh `/:id`). 
- Fiber Router memproses secara linier berdasarkan urutan registrasi. Jika rute dinamis diletakkan di atas, maka rute statis akan tertangkap oleh parameter `id`.
- **Contoh Benar:**
  ```go
  cctvGroup.Get("/template", handlers.DownloadTemplateCCTV)  // statis - DIDAFTARKAN DULUAN
  cctvGroup.Get("/:id", handlers.GetCCTVByID)                // dinamis - DIDAFTARKAN SETELAHNYA
  ```

### Soft Delete vs Unique Constraint
- PostgreSQL secara _default_ tetap memberlakukan _unique constraint_ (seperti pada kolom `KodeCCTV`, `QrPayload`, `Username`, atau `Email`) meskipun sebuah _row_ telah berstatus _soft-delete_.
- Pada _handler_ `CreateCCTV` dan `ImportCCTV`, digunakan perintah `.Unscoped()` untuk secara eksplisit mencari data yang pernah di-_soft-delete_. Jika ditemukan bentrokan (data yang telah terhapus tapi konfigurasinya sama), sistem akan melakukan _hard delete_ (_force delete_) terlebih dahulu sebelum melakukan insert data baru.

### AES_SECRET_KEY Consistency
- Kunci (_key_) enkripsi **HARUS konsisten** antara saat sistem me-generate QR code dan saat memindai (_scan_) QR code tersebut.
- Jika nilai ini berubah (misalnya pada saat Anda berpindah menggunakan `.env` ke `.env.docker`), QR Code yang telah dicetak lama **tidak akan bisa didekripsi** lagi.
- Pastikan bahwa `AES_SECRET_KEY` memiliki nilai yang identik baik di `.env` lokal Anda maupun `.env.docker`.

### Sesi Cookie & CSRF (Pasca-Fase 2)
- JWT **tidak lagi** disimpan di `localStorage`. Token sesi ada di cookie httpOnly `sigap_session` (atribut `Secure` hanya aktif di `APP_ENV=production`, `SameSite=Lax`).
- Jangan kembali menyimpan token di `localStorage`/`sessionStorage`; gunakan cookie + `credentials: 'include'` pada fetch.
- Setiap perubahan password/reset/role oleh Admin menaikkan `TokenVersion` pengguna; seluruh JWT lama langsung dicabut (klaim `tv`). Middleware `Protected()` memvalidasi versi ini terhadap database pada **setiap** request.
- JWT berlaku 5 jam dan tidak menggunakan sliding session. Cookie `sigap_session` dan `sigap_csrf`
  memiliki `MaxAge: 18000` detik (5 jam) agar durasinya selaras dengan JWT.

### WebSocket Autentikasi
- Browser tidak dapat mengirim header `Authorization` saat handshake WebSocket, sehingga token diambil dari cookie httpOnly `sigap_session` (same-origin) atau query `?token=`. Jangan melepas validasi token pada route `/api/ws`.

### Next.js SSR & Browser APIs
- Komponen atau _library_ yang bergantung pada objek global di browser seperti `window` atau `navigator` (contohnya `html5-qrcode`) **HARUS** di-import secara asinkron atau dimuat secara dinamis melalui fungsi `import()` di dalam blok `useEffect`.
- **Dilarang** melakukan impor _library_ ini secara _top-level scope_, karena _Server-Side Rendering_ (SSR) bawaan Next.js akan memproses kodenya di Node.js (dimana API DOM tidak tersedia) yang mengakibatkan aplikasi _crash_.

### Kamera Scanner & Secure Context
- Browser modern memiliki batasan privasi yang memblokir akses ke fungsi WebRTC `getUserMedia()` (akses kamera). Fitur ini hanya bisa dipanggil dalam ekosistem _Secure Context_: yakni melalui protokol `https://` atau jika di-host langsung di `http://localhost`.
- **Catatan**: Akses resmi aplikasi adalah `https://localhost`. Akses melalui IP Address lokal
  seperti `http://192.168.x.x:3001` dianggap sebagai _insecure context_ dan tidak didukung untuk scanner.

## 7. Konfigurasi Docker
Proyek menggunakan pendekatan *Dockerized Environment* untuk *development* dan *deployment*:
- Terdiri dari **7 services**:
  - `db` (berbasis `postgres:15-alpine`) — port `5432` **tidak dipublish** ke host; hanya dapat diakses dari dalam network.
  - `redis` (berbasis `redis:7-alpine`, sejak Task 7.2) — port `6379` **tidak dipublish**; shared store untuk rate-limit counter & token blacklist (persisten lintas restart, multi-instance). Volume `redis_data` (AOF) untuk persistensi.
  - `api` (merupakan _executable binary_ aplikasi Go) — port `3000` **tidak dipublish**; hanya diakses via reverse proxy.
  - `frontend` (menjalankan aplikasi Next.js) — port `3000` internal, **tidak dipublish**.
  - `nginx` (reverse proxy + TLS) — satu-satunya service yang mem-publish port `80` & `443`. Menangani terminasi TLS, HTTP→HTTPS redirect, security headers, routing `/api/*` (termasuk WebSocket) dan `/swagger/*` ke `api`, routing `/stream/*` & `/hls/*` ke `mediamtx`, serta routing sisanya ke `frontend`.
  - `backup` (Task 7.3) — image `postgres:15-alpine` + `dcron`; menjalankan `scripts/backup.sh` via cron (default tiap hari 06:00 WITA) untuk mem-backup database + uploads ke volume `backups_data`. Port **tidak dipublish**; hanya berkomunikasi dengan `db` di dalam network.
  - `mediamtx` (Task 9.2, Fase 9) — image `bluenviron/mediamtx:latest-ffmpeg`, IP statis `172.28.0.11`; proxy RTSP→WebRTC/WHEP streaming CCTV real-time. Port internal `8554` (RTSP), `8889` (HTTP/WebRTC), `8888` (HLS), `9997` (REST API) + UDP `8189`; resource limits 2 CPU / 1024M; health check; port **tidak dipublish** ke host — diakses hanya via Nginx (`/stream/*` → `:8889`, `/hls/*` → `:8888`) dan API (`http://mediamtx:9997/v3/`).
- *Network*: Layanan berkomunikasi di bawah *bridge network* `sigap_network` (subnet eksplisit `172.28.0.0/16`; Nginx memakai IP statis `172.28.0.10` dan MediaMTX memakai IP statis `172.28.0.11`; IP Nginx didaftarkan sebagai trusted proxy).
- *Volume*: Direktori `postgres_data` untuk persistensi _storage_ PostgreSQL; `redis_data` untuk persistensi Redis; `backups_data` (Task 7.3) untuk hasil backup db/uploads; folder `./backend/uploads` di-*bind mount* untuk foto bukti (di-mount read-only ke service `backup`).
- **Secret management (Task 7.5)**: Secret aplikasi (`DB_PASSWORD`, `JWT_SECRET`, `AES_SECRET_KEY`) dikirim sebagai **Docker secret** (file `./secrets/*` → `/run/secrets/<nama>`), dibaca di Go via `utils.secrets.GetSecret` dengan fallback env. `REDIS_PASSWORD` opsional — harus sama di root `.env` & `backend/.env.docker`. Jangan pernah men-commit file `.env*` atau folder `secrets/`.
- **Container hardening (Task 7.5)**: service `api` berjalan **non-root** (`USER sigap`), rootfs **read-only** (`read_only: true`) dengan tmpfs `/tmp` & `/app/logs`, `cap_drop: [ALL]`, dan resource limits (`cpus`/`mem_limit`). Jika menambah fitur yang menulis ke disk container, pastikan target writable (uploads/logs) ter-mount.
- Layanan API membutuhkan file spesifik `backend/.env.docker` (gandakan dari `backend/.env.docker.example`); sertifikat TLS di `nginx/certs/`.
- **Env NVR / MediaMTX** (Task 9.1): `RTSP_NVR_HOST`, `RTSP_NVR_PORT`, `RTSP_NVR_USER` (kredensial NVR Hikvision global) dan `MEDIAMTX_API_URL` (base URL REST MediaMTX, default `http://mediamtx:9997/v3/`) — didokumentasikan di `.env.example` & `backend/.env.docker.example`.
- Menggunakan *Health check* pada _service_ PostgreSQL dan Redis: layanan API akan menunggu (`depends_on`) sampai _service_ basis data dan Redis berstatus `healthy` sebelum memulai _boot sequence_-nya. Bila Redis tidak tersedia saat startup, API tetap berjalan dengan fallback in-memory (hanya LOG WARNING — bukan fail-fast, aman untuk development tanpa Docker).
- **Trusted proxy**: set `TRUSTED_PROXY_IPS=172.28.0.10` (IP Nginx) di `backend/.env.docker` agar header `X-Forwarded-For`/`X-Real-IP` hanya dipercaya dari proxy yang dikenal (anti-spoofing).

## 8. Swagger Documentation (API Docs)
Backend mendukung auto-generasi UI Dokumentasi Interaktif.
- Pada deployment Docker/non-production, Swagger UI dapat diakses pada
  `https://localhost/swagger/index.html` melalui Nginx. Pada development tanpa Docker,
  Swagger tersedia pada `http://localhost:3000/swagger/index.html`.
- Setiap kali Anda memodifikasi atau menambah _endpoint_ REST API yang menyertakan komentar _Swagger annotations_ di dalam file _handlers_, pastikan untuk menjalankan perintah regenerasi:
  ```bash
  cd backend
  swag init
  ```
- File yang secara otomatis di-generate akan tersimpan pada: `docs/docs.go`, `docs/swagger.json`, `docs/swagger.yaml`.

## 9. Testing
- **WAJIB — Manual Test per Sprint:** Setiap sprint/task yang dikerjakan **HARUS** menyertakan dokumen uji manual (blackbox) yang diletakkan di folder **`docs/`** dengan nama pola `manual-test-<nama-fitur>.md` (contoh: `docs/manual-test-pwa.md`, `docs/manual-test-health-score.md`). Format dokumen mengikuti file manual test lainnya: header (Task/Branch/Versi), **1. Prasyarat**, **2. Pengujian Otomatis**, **3. Skenario Uji** (tabel Langkah → Hasil yang diharapkan → Hasil nyata), **4. Kriteria Lulus**, dan **Catatan**. Buat dokumen ini untuk memudahkan user melakukan verifikasi blackbox.
- Matriks cakupan test lengkap (37 file test: `utils`, `middleware`, `handlers`, `models`, `config`, `routes`) ada di **`docs/TESTING.md`**. Bacalah sebelum menambah/memodifikasi test (ada gotchas khusus sqlmock, `SkipDefaultTransaction`, dsb.).
- Untuk mengeksekusi tes & quality check:
  ```bash
  cd backend && go build ./... && go test ./... -count=1 && go vet ./...
  cd backend && golangci-lint run        # lint backend (bila terpasang)
  cd frontend && npm run lint && npm run build   # lint & build frontend
  ```
- Test berjalan **tanpa PostgreSQL fisik** memakai `github.com/DATA-DOG/go-sqlmock`.
- Jalankan `swag init` (di `backend/`) setelah mengubah anotasi Swagger pada handler.

## 10. Changelog
- **SETIAP PERUBAHAN** pada kode backend **HARUS** dicatat dengan rapi di dalam dokumen
  `../backend/changelog.md`.
- Ikuti standar format berikut ketika mencatat _changelog_:
  - Perubahan spesifik yang dilakukan.
  - Perbandingan status _Before_ dan _After_.
  - Label Waktu (_Timestamp_) dalam zona waktu **WITA**.
  - Kategori Skala Perubahan (Kecil / Sedang / Besar).
  - Daftar file yang terpengaruh/diubah.
  - Alasan teknis atau objektif dari perubahan tersebut.

---

## 11. Keputusan Operasional & Kebijakan Sistem

> [!IMPORTANT]
> Bagian ini berisi keputusan operasional yang telah dikonfirmasi oleh user. Semua agent **WAJIB** mematuhi kebijakan ini — jangan override tanpa persetujuan eksplisit user.

### 11.1 Streaming & NVR

| Parameter | Nilai |
|-----------|-------|
| **Host NVR** | `36.91.27.35:554` (Hikvision) |
| **Kredensial NVR** | `<NVR_USERNAME>:<NVR_PASSWORD>` (nilai nyata wajib disimpan sebagai secret; password per-CCTV terenkripsi AES-256 di DB) |
| **Protocol** | RTSP over TCP |
| **Pembedaan kamera** | Channel port 1–35 → `channelID = channelPort × 100 + streamType` |
| **Zero-Transcoding** | ✅ WAJIB — MediaMTX bertindak sebagai RTSP→WebRTC relay murni. **DILARANG** mengaktifkan FFmpeg transcoding. Kamera/NVR dikonfigurasi hardcoded H.264. |
| **Default Stream** | Sub Stream (`02`) — untuk popup peta & grid view (hemat bandwidth ~0.5–1 Mbps/kamera) |
| **Main Stream** | `01` — hanya via tombol toggle manual di dalam video player |
| **Live View Access** | **Admin-only** (route `/streams/*` dilindungi `RequireRole("admin")`) |
| **Recording** | ❌ Tidak ada recording/retention di VPS SIGAP. Playback tetap di NVR lokal. |

### 11.2 Network Health Check — Kebijakan Metro-E vs GSM

> [!CAUTION]
> **JANGAN** melakukan ICMP ping atau background-stream pada kamera GSM. Ini dapat menguras kuota data seluler. Hanya 17 kamera Metro-E yang boleh di-ping secara berkala.

#### Aturan Probe per Tipe Jaringan

| Tipe | ICMP Ping Berkala | RTSP Options Probe | Keterangan |
|------|-------------------|-------------------|------------|
| **Metro-E** (`10.20.105.x`) | ✅ Ya — tiap 2 menit | ✅ Ya | Jaringan dedicated, aman di-probe |
| **GSM** (IP publik/non-Metro-E) | ❌ TIDAK | ❌ TIDAK | Wajib pasif melalui state MediaMTX lokal; jangan menyentuh kamera/NVR GSM |

#### Cara Identifikasi Tipe Jaringan

```go
// Di backend Go — identifikasi Metro-E berdasarkan IP kamera
func isMetroE(cctv models.CCTV) bool {
    // Cara 1: cek prefix IP kamera
    if strings.HasPrefix(cctv.IPCamera, "10.20.105.") {
        return true
    }
    // Cara 2: cek field TipeKoneksi jika diisi di master data
    if strings.EqualFold(cctv.TipeKoneksi, "metro-e") ||
       strings.EqualFold(cctv.TipeKoneksi, "metro e") {
        return true
    }
    return false
}
```

#### 17 Kamera Metro-E — IP Resmi

| Kode | IP Camera | Kode | IP Camera |
|------|-----------|------|-----------|
| D3   | `10.20.105.62` | D11  | `10.20.105.55` |
| D5   | `10.20.105.66` | D12  | `10.20.105.58` |
| D6   | `10.20.105.59` | D13  | `10.20.105.67` |
| D7   | `10.20.105.53` | D15  | `10.20.105.61` |
| D8   | `10.20.105.52` | D16  | `10.20.105.56` |
| D9   | `10.20.105.54` | D17  | `10.20.105.63` |
| D19  | `10.20.105.60` | D28  | `10.20.105.70` |
| D23  | `10.20.105.65` | D30  | `10.20.105.68` |
| D38  | `10.20.105.57` | — | — |
> [!IMPORTANT]
> **Addendum Kebijakan Metro-E vs GSM — Probe Strategy Final (dikonfirmasi 2026-08-13):**
> Implementasi `RunRTSPProber` telah di-patch dan wajib mempertahankan strategi berikut:
>
> | Tipe | Cara Identifikasi | Probe Method | Interval | Konsumsi Data GSM |
> |------|-------------------|-------------|----------|-------------------|
> | **Metro-E** | `ip_camera` prefix `10.20.105.` | ICMP Ping → IP kamera langsung | 2 menit | ❌ Tidak ada (jaringan dedicated) |
> | **GSM** | IP selain `10.20.105.x` | Cek MediaMTX path list (lokal VPS) + last-active tracking | Pasif (event-driven) | ✅ Nol — tidak menyentuh kamera |
>
> **Strategi GSM — Zero Data Consumption:**
> ```
> JANGAN probe ke NVR/kamera GSM sama sekali.
> Gunakan data yang sudah ada di VPS:
>
> 1. Cek MediaMTX path list (GET http://mediamtx:9997/v3/paths/list)
>    → Jika path "cam-{uuid}" Ready=true → kamera ONLINE (sedang ada viewer)
>    → Jika path tidak ada/Ready=false → tidak bisa disimpulkan (mungkin tidak ada yang nonton)
>
> 2. Last-active tracking dari stream events:
>    → Field baru: stream_last_active_at TIMESTAMPTZ (nullable)
>    → Diupdate setiap stream berhasil dibuka (Ready=true di MediaMTX)
>    → Jika stream_last_active_at < 24 jam lalu → status "Pernah Online Hari Ini"
>    → Jika stream_last_active_at > 24 jam atau null → status "Tidak Diketahui"
>    → Jika stream pernah aktif lalu tiba-tiba gagal → status "Offline"
> ```
>
> **Badge UI untuk GSM:**
> - 🟢 Online — MediaMTX path Ready=true (sedang ditonton)
> - 🟡 Pernah aktif — `stream_last_active_at` < 24 jam, saat ini tidak ada viewer
> - ⚪ Tidak diketahui — belum pernah dibuka atau > 24 jam
> - 🔴 Offline — stream pernah aktif tapi sekarang error/disconnected
>
> **17 Metro-E IP Resmi:** D3:`10.20.105.62` D5:`10.20.105.66` D6:`10.20.105.59` D7:`10.20.105.53` D8:`10.20.105.52` D9:`10.20.105.54` D11:`10.20.105.55` D12:`10.20.105.58` D13:`10.20.105.67` D15:`10.20.105.61` D16:`10.20.105.56` D17:`10.20.105.63` D19:`10.20.105.60` D23:`10.20.105.65` D28:`10.20.105.70` D30:`10.20.105.68` D38:`10.20.105.57`
>
> **Latensi Metro-E:** `<50ms` 🟢 Good | `50–100ms` 🟡 Lagging | `>100ms` 🔴 Critical | Timeout ⚫ Offline
> Lihat `docs/AGENTS.md` § 11.2 untuk detail lengkap.

#### Sprint 37-PATCH — Update Probe Strategy Metro-E vs GSM

> **Branch:** `bugfix/rtsp-probe-metro-gsm-strategy`
> **Prasyarat:** Sprint 37 sudah selesai (stream_probe.go sudah ada)
> **Estimasi:** 1 hari kerja
> **Status:** ✅ Selesai (2026-08-13, backend `0.63.0`)

| Task | Deliverable |
|------|-------------|
| **Patch `utils/rtsp_probe.go`** | Tambah `ProbeRTSPHostICMP(ip string) (latencyMs int, ok bool)` via `net.Dial("ip4:icmp", ...)`. Tambah `CheckMediaMTXPath(client, path string) bool` — cek `Ready=true` di path list. |
| **Patch `handlers/stream_probe.go`** | Update `runRTSPProbeCycle()`: split Metro-E vs GSM. Metro-E: `ProbeRTSPHostICMP` + update `stream_live` + `ping_latency_ms`. GSM: cek MediaMTX path list → update `stream_live` (true hanya jika Ready) + update `stream_last_active_at` jika Ready. |
| **Migration baru** | `000022_stream_ping_fields.up.sql`: tambah `ping_latency_ms INTEGER` (nullable) + `stream_last_active_at TIMESTAMPTZ` (nullable) ke tabel `cctvs`. |
| **Update model** | `models/cctv.go`: tambah `PingLatencyMs *int`, `StreamLastActiveAt *time.Time`. |
| **Update `GetLiveStatus`** | Sertakan `ping_latency_ms`, `stream_last_active_at`, dan `network_type` (`"metro-e"` atau `"gsm"`) di response. |
| **Update `CCTVMap.tsx`** | Badge GSM: 4 state (🟢/🟡/⚪/🔴). Tooltip Metro-E: tampilkan latensi (`"Ping: 12ms"`). |
| **Test** | `utils/rtsp_probe_test.go` + `handlers/stream_probe_test.go` update untuk Metro-E vs GSM path. |

#### GSM Safety Rules

```
✅ Boleh: Stream terbuka SAAT user aktif menonton (on-demand)
❌ Dilarang: ICMP ping berkala (tiap menit/2 menit)
❌ Dilarang: RTSP OPTIONS/probe langsung ke kamera atau NVR GSM
❌ Dilarang: background-stream tanpa viewer
✅ Wajib: Status GSM berasal dari MediaMTX path list + last-active tracking di VPS
✅ Wajib: Auto-close stream GSM saat popup/modal ditutup (existing: sourceOnDemandCloseAfter: 30s)
```

### 11.3 Infrastruktur Deployment

| Environment | Detail |
|-------------|--------|
| **Staging** | Dell PC lokal — Windows 11 + Docker Desktop/WSL2. Akses remote via **Tailscale SSH**. |
| **Production** | VPS Hostinger |
| **Akses web** | `https://localhost` (staging) / domain publik (produksi), keduanya via Nginx TLS |
| **Deploy method** | `docker compose up -d` — tidak ada CI/CD otomatis saat ini |

### 11.4 Prioritas Fitur (Sudah Dikonfirmasi)

Urutan prioritas yang telah disepakati dengan user:

```
1. Spare Parts Inventory Management (Task 11.1) ← lebih prioritas dari Multi-View Monitor
2. Multi-View Monitor (Task 10.3) ← dikerjakan setelah Spare Parts
3. Fase 13 (Perbaikan teknis) — parallel Sprint 31-36
4. Fase 14 (CCTV Intelligence) — mulai setelah Sprint 31
```

> [!NOTE]
> **Sprint 40 — AI Feature Takedown (2026-08-13):** Fitur AI detection (person counting, bounding box, YOLOv8) telah **dihapus** dari SIGAP — sistem kembali ke pure CCTV management. Fase 14 selesai dengan **Live Status** (Sprint 37 ✅), **Metro-E/GSM Probe Patch** (Sprint 37-PATCH ✅), dan **AI Takedown** (Sprint 40 ✅); Sprint 38 & 39 (AI Microservice & Frontend AI Overlay) **DIBATALKAN**. Lihat `docs/ROADMAP.md` Fase 14.

---
_Dokumen ini adalah referensi sentral untuk AI. Patuhi segala aturan, instruksi arsitektural, dan gotchas teknis yang tercantum di atas._
