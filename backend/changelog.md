# Changelog - Backend SIGAP

## [0.64.41] - 2026-11-01

### [High][H02] Penegakan isolasi site pada akses objek, turunan CCTV, pemeliharaan, dan zona (#168)

**Masalah**: Isolasi kawasan (`site_id`) selama ini ditegakkan pada endpoint daftar (list), tetapi sejumlah endpoint akses objek berbasis ID atau QR (`GetCCTVByID`, `GetCCTVQRCode`, `ScanCCTV`, `RotateQRCode`, `UpdateCCTV`, `DeleteCCTV`, `GetCCTVSnapshot`, `GetCCTVSnapshotStatus`, `GetCCTVTimeline`, `GetCCTVHealthScore`, `GetCCTVHealthHistory`, `CreateMaintenanceLog`, `GetMaintenanceLogByID`, `UpdateMaintenanceLog`, `AssignMaintenanceLog`, `CompleteMaintenanceLog`, `DeleteMaintenanceLog`, serta `GetZoneByID`/`UpdateZone`/`DeleteZone`/`AssignCCTVToZone`/`AssignUsersToZone`) belum membandingkan `site_id` objek terhadap `site_id` aktif pengguna. Selain itu pada log pemeliharaan, pemeriksaan ownership teknisi saja tidak menutup akses apabila aset CCTV telah dipindahkan ke site lain.

**Perbaikan**:
- Menegakkan pemeriksaan `cctvMatchesActiveSite(c, cctv)` pada seluruh pembacaan detail, QR, scan, rotasi QR, pembaruan, penghapusan, snapshot, status snapshot, timeline, skor kesehatan, serta histori skor kesehatan CCTV.
- Menegakkan pemeriksaan `cctvMatchesActiveSite` pada `CreateMaintenanceLog` dan `ensureMaintenanceLogMatchesActiveSite` pada `GetMaintenanceLogByID`, `UpdateMaintenanceLog`, `AssignMaintenanceLog`, `CompleteMaintenanceLog`, serta `DeleteMaintenanceLog` sehingga ownership dan site diperiksa bersama (termasuk menolak akses setelah CCTV dipindahkan ke site lain).
- Menambahkan `zoneMatchesActiveSite(c, zone)` pada `GetZoneByID`, `UpdateZone`, `DeleteZone`, `AssignCCTVToZone`, dan `AssignUsersToZone`, serta mengaitkan `site_id` aktif saat `CreateZone`.
- Menambahkan regression test lengkap dua site di `backend/handlers/site_isolation_h02_test.go` (`TestAudit_CCTVDetailMustDenyOtherSite`, `TestAudit_MaintenanceCreateMustDenyOtherSite`, `TestAudit_FieldCheckMustDenyOtherSite_Control`, `TestSiteIsolation_CCTVObjectAccessAndDerivativesDenyOtherSite`, `TestSiteIsolation_MaintenanceDeniesAccessAfterCCTVMovedToOtherSite`, dan `TestSiteIsolation_ZoneDetailAndWriteDenyOtherSite`).

timestamp: 2026-11-01 01:11 UTC+8
kategori: Tinggi, Backend (Security / Object-Level Site Isolation)
file yang diubah:
- backend/handlers/cctv.go
- backend/handlers/snapshot.go
- backend/handlers/health_score.go
- backend/handlers/maintenance.go
- backend/handlers/zone.go
- backend/handlers/field_maintenance.go
- backend/handlers/site_isolation_h02_test.go

alasan perubahan: Menutup celah akses dan penulisan objek lintas kawasan (IDOR lintas site) pada CCTV, pemeliharaan, dan zona (Issue #168 / H02).

---
## [0.64.40] - 2026-11-01

### [High][H01] Enkapsulasi autentikasi & otorisasi Admin pada jalur media streaming (/stream/* & /hls/*) (#167)

**Masalah**: Jalur media langsung (`/stream/*` untuk WebRTC WHEP dan `/hls/*` untuk HLS) diteruskan ke MediaMTX tanpa memvalidasi sesi JWT maupun peran Admin. Selain itu `mediamtx.yml` mengizinkan `read`/`playback` dari IP mana pun pada seluruh path.

**Perbaikan**:
- Menambahkan handler `AuthorizeStreamAccess` (`GET/HEAD/POST /api/streams/auth`) untuk subrequest `auth_request` Nginx yang memverifikasi JWT aktif (termasuk pencabutan token setelah logout dan `token_version`), peran `admin`, keberadaan kamera `cam-<uuid>`, status `stream_enabled=true`, serta cakupan `site_id` aktif.
- Menambahkan `ProxyWHEPStream` (`/stream/*`) dan `ProxyHLSStream` (`/hls/*`) lewat `SetupStreamMediaRoutes(app)` agar akses melalui rewrite Next.js atau tanpa Nginx tetap melewati autentikasi/otorisasi Admin yang sama.
- Memperketat `loadCCTVByID` pada kontrol stream agar menghormati isolasi `site_id` aktif.
- Memperbarui `nginx/nginx.conf` dengan subrequest `auth_request /_stream_auth` pada `location /stream/` dan `location /hls/`.
- Memperketat `mediamtx.yml` (`webrtcTrustedProxies: []`, `ips` internal, dan pembatasan path `~^cam-[0-9a-fA-F-]{36}$` pada `read`/`playback`).
- Menambahkan regression test lengkap di `backend/handlers/stream_media_test.go` dan `backend/routes/stream_test.go`.

timestamp: 2026-11-01 00:47 UTC+8
kategori: Tinggi, Backend & Infra (Security / Streaming Auth)
file yang diubah:
- backend/handlers/stream_media.go
- backend/handlers/stream_media_test.go
- backend/handlers/stream.go
- backend/routes/stream.go
- backend/routes/stream_test.go
- backend/utils/mediamtx.go
- backend/main.go
- nginx/nginx.conf
- mediamtx.yml
- docker-compose.yml

alasan perubahan: Menutup celah bypass autentikasi pada jalur media video langsung (Issue #167 / H01).

---
## [0.64.39] - 2026-10-28

### Status snapshot CCTV dan penanganan endpoint MediaMTX yang tidak tersedia

**Masalah**: Frontend selalu menampilkan tombol snapshot yang aktif. Ketika
MediaMTX tidak menyediakan endpoint snapshot (mis. v1.20) atau file snapshot
sudah terhapus, klik tombol membuka tab baru berisi 404/JSON error. Pengguna
tidak mendapat alasan yang jelas dan UI terlihat rusak.

**Perbaikan**:
- Menambahkan endpoint `GET /api/cctv/:id/snapshot-status` (protected) yang
  melaporkan `available`, `endpoint_supported`, `snapshot_url`, `captured_at`,
  `event_type`, dan `message`.
- `available=true` hanya bila MediaMTX mendukung endpoint snapshot **dan**
  metadata snapshot terbaru **dan** file benar-benar ada di disk (`os.Stat`).
- Endpoint status **tidak** memicu capture baru; membuka modal tidak menambah
  beban MediaMTX atau kuota GSM.
- UUID invalid → HTTP 400, CCTV tidak ditemukan → HTTP 404, kegagalan query
  metadata → HTTP 500, sedangkan kondisi "belum ada snapshot"/"file hilang"
  tetap HTTP 200 dengan pesan yang dapat ditampilkan UI.
- Menambahkan 5 test handler (endpoint tidak tersedia, belum ada snapshot,
  snapshot tersedia, file hilang, UUID invalid) dan memperbarui test registrasi
  route protected.
- Menambahkan dokumentasi `docs/01-documentation/cctv-codec-snapshot-limitations.md`
  dan manual test `docs/04-manual-tests/manual-test-cctv-codec-snapshot.md`.

**Hasil verifikasi** (dijalankan dari `backend/`, Go 1.26.5):
- `gofmt -l` → tidak ada file yang perlu diformat
- `go build ./...` → PASS
- `go vet ./...` → PASS
- `go test ./... -count=1` → PASS (seluruh paket)
- `golangci-lint run ./...` (v2.12.2) → PASS, 0 issues

timestamp: 2026-10-28 12:25 UTC+8
kategori: Sedang, Backend (CCTV Snapshot / Status Endpoint / MediaMTX Compatibility)
file yang diubah:
- backend/handlers/snapshot.go
- backend/handlers/snapshot_test.go
- backend/routes/cctv.go
- backend/routes/routes_test.go
- backend/docs/docs.go
- backend/docs/swagger.json
- backend/docs/swagger.yaml
- backend/changelog.md
- docs/01-documentation/cctv-codec-snapshot-limitations.md
- docs/04-manual-tests/manual-test-cctv-codec-snapshot.md
- docs/04-manual-tests/TESTING.md

alasan perubahan: Memberikan status ketersediaan snapshot CCTV yang akurat (termasuk saat MediaMTX 1.20 tidak memiliki endpoint snapshot atau file hilang) agar tombol snapshot di UI tidak membuka halaman 404.

---

## [0.64.38] - 2026-10-28

### Kebijakan event standby dan deduplikasi Log CCTV

**Masalah**: Setiap siklus probe (2 menit) dan verifier (15 menit) dapat
menghasilkan event `standby` saat kamera on-demand idle, sehingga tabel
`cctv_stream_logs` dipenuhi event rutin yang bukan gangguan. Selain itu, probe
dan verifier dapat mengamati transisi `online`/`offline` yang sama dan menulis
dua baris log untuk satu kejadian.

**Keputusan produk**:
- `standby` tetap menjadi status stream yang sah, tetapi **tidak dicatat
  sebagai event secara default**. Pencatatan dapat diaktifkan dengan
  `CCTV_LOG_STANDBY_EVENTS=true` untuk audit/debugging.
- Event dengan tipe sama untuk CCTV yang sama diduplikasi dalam jendela
  **60 detik** berdasarkan log terakhir kamera tersebut.
- Query deduplikasi yang gagal bersifat **fail-open**: event tetap dicatat agar
  kejadian penting tidak hilang.

**Perbaikan**:
- Menambahkan `config.CCTVLogStandbyEvents()` dan
  `DefaultCCTVLogStandbyEvents=false`; nilai kosong/tidak valid kembali ke
  `false`.
- Menambahkan `handlers.shouldLogCCTVStreamEvent` yang menerapkan kebijakan
  standby sebelum query, lalu memeriksa log terakhir per CCTV untuk deduplikasi
  60 detik.
- Membungkus penulisan log di `stream_probe.go` dan `stream_verify.go` dengan
  helper tersebut. Status stream, kolom `stream_*`, dan broadcast
  `CCTV_LIVE_STATUS_UPDATED` tetap berjalan walaupun log di-skip.
- Mendokumentasikan `CCTV_LOG_STANDBY_EVENTS` di `.env.example`,
  `.env.docker.example`, dan `docker-compose.yml`.
- Menambahkan test konfigurasi default/override/fallback, test helper
  (standby default/aktif, duplikat dalam/lewat jendela, tipe berbeda, tanpa log,
  error database), serta melengkapi ekspektasi sqlmock pada test probe/verifier.
- Menambahkan dokumen semantics `docs/01-documentation/cctv-stream-log-semantics.md`
  yang menjelaskan perbedaan `probe` vs `verify`, arti status, kebijakan
  standby, deduplikasi, dan query diagnosis.

**Hasil verifikasi** (dijalankan dari `backend/`, Go 1.26.5):
- `go build ./...` → PASS
- `go vet ./...` → PASS
- `go test ./... -count=1` → PASS (seluruh paket)
- `golangci-lint run ./...` (v2.12.2) → PASS, 0 issues

timestamp: 2026-10-28 00:45 UTC+8
kategori: Sedang, Backend (CCTV Logs / Policy / Deduplication)
file yang diubah:
- backend/config/cctv_log.go
- backend/config/cctv_log_test.go
- backend/handlers/cctv_stream_log_events.go
- backend/handlers/cctv_stream_log_events_test.go
- backend/handlers/stream_probe.go
- backend/handlers/stream_probe_test.go
- backend/handlers/stream_verify.go
- backend/handlers/stream_verify_test.go
- backend/.env.example
- backend/.env.docker.example
- docker-compose.yml
- backend/changelog.md
- docs/01-documentation/cctv-stream-log-semantics.md
- docs/04-manual-tests/manual-test-cctv-log-dedup.md
- docs/04-manual-tests/TESTING.md

alasan perubahan: Menghentikan banjir log standby dan duplikasi event dari probe/verifier tanpa menghilangkan event online/offline yang penting.

---

## [0.64.37] - 2026-10-28

### Validasi event type dan error handling ringkasan Log CCTV

**Masalah**: Parameter `event_type` tidak divalidasi, sehingga nilai typo
seperti `onlne` tetap menghasilkan HTTP 200 dengan daftar kosong dan nilai
`all` diam-diam diabaikan. Kegagalan query count ringkasan juga ditelan dan
dikembalikan sebagai `0`, sehingga UI menampilkan angka palsu seolah tidak ada
event. Literal event type tersebar di beberapa handler sehingga rawan typo.

**Perbaikan**:
- Menambahkan konstanta `models.CCTVEventOnline`, `CCTVEventOffline`,
  `CCTVEventStandby`, dan `CCTVEventUnknown`, lalu memakainya di
  `stream_probe.go`, `stream_verify.go`, serta handler Log CCTV.
- Memvalidasi `event_type` sebelum menyentuh database: nilai kosong berarti
  tanpa filter, hanya empat nilai resmi yang diterima, dan nilai lain
  menghasilkan **HTTP 400** dengan pesan yang menyebut nilai yang diizinkan.
- Mengubah `countCCTVStreamLogs` agar mengembalikan error. Handler kini
  merespons **HTTP 500** `"Gagal menghitung ringkasan log CCTV"` bila query
  ringkasan gagal, bukan menampilkan `0` palsu.
- Memperbarui anotasi Swagger untuk nilai `unknown` dan respons `400`.
- Menambahkan test event type invalid (termasuk `all`, huruf besar, dan typo),
  filter `unknown` yang valid, serta kegagalan query ringkasan online/offline.
- Endpoint belum memiliki parameter rentang tanggal, sehingga batas maksimum
  rentang belum relevan untuk ditambahkan.

**Hasil verifikasi** (dijalankan dari `backend/`, Go 1.26.5):
- `go build ./...` → PASS
- `go vet ./...` → PASS
- `go test ./... -count=1` → PASS (seluruh paket)
- `golangci-lint run ./...` (v2.12.2) → PASS, 0 issues

timestamp: 2026-10-28 00:30 UTC+8
kategori: Sedang, Backend (CCTV Logs / Validation / Error Handling)
file yang diubah:
- backend/handlers/cctv_stream_log.go
- backend/handlers/cctv_stream_log_test.go
- backend/handlers/stream_probe.go
- backend/handlers/stream_verify.go
- backend/models/cctv_stream_log.go
- backend/changelog.md
- docs/04-manual-tests/manual-test-cctv-log-validation.md
- docs/04-manual-tests/TESTING.md

alasan perubahan: Menolak input filter yang tidak dikenal dengan pesan jelas dan memastikan kegagalan database tidak menampilkan ringkasan palsu.

---

## [0.64.36] - 2026-10-28

### Retensi dan index komposit log event CCTV

**Masalah**: Tabel `cctv_stream_logs` terus bertambah karena setiap perubahan
status stream menghasilkan baris baru tanpa mekanisme cleanup. Index yang ada
hanya kolom tunggal `cctv_id` dan `event_type`, sehingga query daftar yang
memfilter kamera/jenis event lalu mengurutkan `created_at` harus membaca dan
mengurutkan banyak baris. Tabel yang tumbuh tanpa batas memperlambat query serta
memperbesar biaya storage dan backup.

**Perbaikan**:
- Menambahkan `config.CCTVStreamLogRetentionDays()` yang membaca
  `CCTV_STREAM_LOG_RETENTION_DAYS` dengan default **90 hari**. Nilai kosong,
  bukan angka, atau `<= 0` jatuh ke default agar salah konfigurasi tidak
  menghapus seluruh log.
- Menambahkan `handlers.CleanupCCTVStreamLogs` yang menghapus log lebih tua
  dari cutoff secara bertahap maksimal **1.000 baris per batch** dengan jeda
  **100 ms** antar-batch, sehingga lock tabel singkat dan worker tidak
  membanjiri database saat backlog besar.
- Menambahkan `handlers.StartCCTVStreamLogCleanup` yang berjalan sekali saat
  startup lalu setiap **24 jam**. Worker bersifat idempotent: pemanggilan
  berikutnya pada data yang sama tidak menghapus apa pun.
- Menambahkan migration `000029_cctv_stream_logs_indexes` yang membuat index
  komposit `(cctv_id, created_at)` dan `(event_type, created_at)`, menghapus
  index kolom tunggal yang sudah tercakup prefix, serta mempertahankan index
  `created_at DESC` untuk query cleanup.
- Menghubungkan worker di `main.go` dan mendokumentasikan konfigurasi retensi di
  `.env.example`, `.env.docker.example`, dan `docker-compose.yml`.
- Menambahkan test konfigurasi default/override/fallback, test cleanup (batch,
  idempotensi, error database, cutoff), dan test isi migration up/down.

**Hasil verifikasi** (dijalankan dari `backend/`):
- `go build ./...` → PASS
- `go vet ./...` → PASS
- `go test ./... -count=1` → PASS (seluruh paket)
- `golangci-lint run ./...` (v2.12.2) → PASS, 0 issues
- Catatan: Docker tidak tersedia di lingkungan lokal, sehingga verifikasi
  migration `up → down → up` dan cleanup black-box dijalankan oleh CI serta
  prosedur di `docs/04-manual-tests/manual-test-cctv-log-retention.md`.

timestamp: 2026-10-28 00:20 UTC+8
kategori: Sedang, Backend (CCTV Logs / Retention / Index / Ops)
file yang diubah:
- backend/config/cctv_log.go
- backend/config/cctv_log_test.go
- backend/handlers/cctv_stream_log_cleanup.go
- backend/handlers/cctv_stream_log_cleanup_test.go
- backend/migrations/000029_cctv_stream_logs_indexes.up.sql
- backend/migrations/000029_cctv_stream_logs_indexes.down.sql
- backend/config/migrate_test.go
- backend/main.go
- backend/.env.example
- backend/.env.docker.example
- docker-compose.yml
- backend/changelog.md
- docs/04-manual-tests/manual-test-cctv-log-retention.md
- docs/04-manual-tests/TESTING.md

alasan perubahan: Membatasi pertumbuhan tabel log event CCTV, menghapus data lama secara aman dan bertahap, serta mempercepat query daftar/ringkasan melalui index komposit.

---

## [0.64.35] - 2026-10-27

### Timezone operasional dan konsistensi ringkasan Log CCTV

**Masalah**: `startOfDay` pada ringkasan Log CCTV memakai timezone proses,
sedangkan komentar menyebut UTC. Container umumnya berjalan di UTC, sementara
operasional memakai WITA, sehingga event pukul 00:00-07:59 WITA bisa masuk ke
hari yang salah. Ringkasan "hari ini" juga hanya memakai scope site dan tidak
mengikuti filter `cctv_id`, `event_type`, atau `search`, sehingga angka kartu
tidak konsisten dengan daftar.

**Perbaikan**:
- Menambahkan `config.AppLocation()` yang membaca `APP_TIMEZONE` (default
  `Asia/Makassar`), di-cache per nama zona, dan jatuh ke fixed zone WITA (UTC+8)
  bila nama zona tidak valid/tidak tersedia di runtime.
- Menambahkan `cctvStreamLogDayRange` yang menghitung rentang `[start, end)`
  satu hari kalender WITA, lalu mengembalikannya dalam UTC agar perbandingan
  `timestamptz` konsisten di server UTC maupun WITA.
- Memusatkan filter daftar pada `applyCCTVStreamLogFilters`; query ringkasan
  kini memakai scope site serta filter `cctv_id`, `event_type`, dan `search`
  yang sama dengan daftar.
- Menambahkan field `timezone` pada `summary` respons `/api/cctv-logs` agar UI
  dapat menampilkan zona waktu perhitungan hari.
- Mendokumentasikan `APP_TIMEZONE` di `.env.example`, `.env.docker.example`,
  dan `docker-compose.yml`.
- Menambahkan test `config/timezone_test.go` serta
  `handlers/cctv_stream_log_timezone_test.go` untuk batas tengah malam WITA,
  independensi timezone proses, dan propagasi filter ke ringkasan.

**Hasil verifikasi** (dijalankan dari `backend/`, Go 1.26.5):
- `go build ./...` → PASS
- `go vet ./...` → PASS
- `go test ./... -count=1` → PASS (seluruh paket)
- `golangci-lint run ./...` (v2.12.2) → PASS, 0 issues

timestamp: 2026-10-27 23:25 UTC+8
kategori: Sedang, Backend (CCTV Logs / Timezone / Query Consistency)
file yang diubah:
- backend/config/timezone.go
- backend/config/timezone_test.go
- backend/handlers/cctv_stream_log.go
- backend/handlers/cctv_stream_log_test.go
- backend/handlers/cctv_stream_log_isolation_test.go
- backend/handlers/cctv_stream_log_timezone_test.go
- backend/.env.example
- backend/.env.docker.example
- docker-compose.yml
- backend/changelog.md
- docs/04-manual-tests/TESTING.md

alasan perubahan: Menyamakan batas hari Log CCTV dengan zona waktu operasional WITA dan memastikan kartu ringkasan konsisten dengan daftar serta filter aktif.

---

## [0.64.34] - 2026-10-27

### Regression test multi-site dan proxy spoofing

**Masalah**: Perbaikan P0 #139 (isolasi log CCTV/WebSocket per site) dan #140
(hardening trusted proxy serta anti-spoofing `X-Forwarded-For`) belum memiliki
regression test yang menguji skenario dua site secara bersamaan. Tanpa test ini,
perubahan berikutnya berpotensi mengembalikan kebocoran data lintas site atau
membuka kembali celah spoofing IP tanpa terdeteksi.

**Perbaikan**:
- Menambahkan `handlers/cctv_stream_log_isolation_test.go`: teknisi site A dan
  site B mengakses endpoint log CCTV yang sama dan masing-masing hanya menerima
  log/CCTV milik site-nya. Seluruh query (count, list, preload CCTV, ringkasan
  online/offline) diverifikasi memakai parameter `site_id` yang benar.
- Memperluas `middleware/limiter_test.go`: IP yang sama pada endpoint berbeda
  memiliki bucket terpisah; peer yang tidak ada di daftar trusted proxy tidak
  dapat memakai XFF untuk memecah bucket; entri trusted proxy berbentuk CIDR
  dikenali dan IP klien di header dipakai per klien.
- Memperkuat `utils/websocket_test.go`: event nyata `CCTV_STREAM_LOG_CREATED`
  site A hanya diterima teknisi site A dan admin, tidak diterima teknisi site B;
  payload `site_id`/`nama`/`kode_cctv` ikut diverifikasi.
- Menambahkan laporan hasil `docs/05-reports/SECURITY-REGRESSION-2026-10-27.md`
  dan memperbarui matriks `docs/04-manual-tests/TESTING.md`.

**Hasil verifikasi** (dijalankan dari `backend/`, Go 1.26.5):
- `go build ./...` → PASS
- `go vet ./...` → PASS
- `go test ./... -count=1` → PASS (seluruh paket)
- `golangci-lint run ./...` (v2.12.2) → PASS, 0 issues

timestamp: 2026-10-27 23:20 UTC+8
kategori: Rendah, Backend (Regression Test / Security)
file yang diubah:
- backend/handlers/cctv_stream_log_isolation_test.go
- backend/middleware/limiter_test.go
- backend/utils/websocket_test.go
- backend/changelog.md
- docs/04-manual-tests/TESTING.md
- docs/05-reports/SECURITY-REGRESSION-2026-10-27.md

alasan perubahan: Mengunci perilaku isolasi multi-site dan hardening proxy dengan regression test agar perbaikan P0 #139/#140 tidak kembali regresi.

---

## [0.64.33] - 2026-10-27

### Hardening trusted proxy: validasi IP dan anti-spoofing X-Forwarded-For

**Masalah**: `main.go` selalu memasang `ProxyHeader: fiber.HeaderXForwardedFor` meskipun `EnableTrustedProxyCheck` nonaktif saat `TRUSTED_PROXY_IPS` kosong. Pada Fiber v2, `IsProxyTrusted()` mengembalikan `true` bila pengecekan trusted proxy nonaktif, sehingga API yang diakses langsung mempercayai `X-Forwarded-For` dari klien. Klien dapat memalsukan IP untuk memecah bucket rate limit dan mengotori audit log. Nginx juga memakai `$proxy_add_x_forwarded_for` sehingga nilai XFF dari klien ikut diteruskan ke backend.

**Perbaikan**:
- Menambahkan `config.ParseTrustedProxies` untuk memvalidasi dan menormalisasi daftar IP/CIDR dari `TRUSTED_PROXY_IPS`; entri kosong/duplikat dibuang, entri tidak valid dibuang dengan warning, dan bentuk IPv4-mapped IPv6/CIDR dinormalisasi.
- Menambahkan `config.NewFiberConfig` yang selalu mengaktifkan `EnableIPValidation`; `EnableTrustedProxyCheck`, `TrustedProxies`, dan `ProxyHeader` hanya dipasang bila ada minimal satu trusted proxy valid.
- `main.go` kini memakai helper tersebut, sehingga deployment tanpa proxy tidak membaca `X-Forwarded-For` sama sekali.
- Nginx menimpa `X-Forwarded-For` dengan `$remote_addr` (bukan append) pada lokasi `/api/`, `/stream/`, dan `/`; dokumentasi `set_real_ip_from`/`real_ip_header` ditambahkan untuk deployment dengan load balancer di depan Nginx.
- Menambahkan regression test `config/fiber_test.go` dan memperluas `middleware/limiter_test.go` untuk membuktikan spoofing XFF tidak memecah bucket rate limit dan audit log memakai IP tervalidasi.
- Memperjelas dokumentasi `TRUSTED_PROXY_IPS` di `backend/.env.example` dan `backend/.env.docker.example`, serta menambahkan manual test P0 #140.

timestamp: 2026-10-27 22:55 UTC+8
kategori: Tinggi, Backend + Infrastructure (Security / Rate Limit / Audit Log)
file yang diubah:
- backend/config/fiber.go
- backend/config/fiber_test.go
- backend/main.go
- backend/middleware/limiter.go
- backend/middleware/limiter_test.go
- backend/utils/audit.go
- backend/.env.example
- backend/.env.docker.example
- nginx/nginx.conf
- backend/changelog.md
- docs/04-manual-tests/manual-test-trusted-proxy-ip-hardening.md
- docs/04-manual-tests/TESTING.md

alasan perubahan: Menutup celah spoofing IP yang dapat mem-bypass rate limit dan mencemari audit log, serta memastikan hanya trusted proxy yang boleh menentukan IP klien.

---

## [0.64.32] - 2026-10-27

### Isolasi log stream CCTV dan event WebSocket per site

**Masalah**: `GetCCTVStreamLogs` menerima `SiteContext` tetapi tidak memakai `c.Locals("site_id")`, sehingga teknisi di site A dapat melihat log CCTV site B. Event WebSocket `CCTV_STREAM_LOG_CREATED`, `CCTV_LIVE_STATUS_UPDATED`, dan `CCTV_UPDATED` juga disiarkan global ke seluruh koneksi, membocorkan nama/kode kamera lintas kawasan. Selain itu, identitas JWT pada koneksi WebSocket tidak pernah terbaca karena library `gofiber/websocket` tidak menyediakan locals `fiberctx`, sehingga `BroadcastToUser`/`BroadcastToRole` selama ini menerima identitas kosong.

**Perbaikan**:
- Menerapkan scope site pada `GetCCTVStreamLogs` untuk query count, list, dan ringkasan harian memakai pola `EXISTS` ke `cctvs.site_id` seperti `applyMaintenanceSiteScope`.
- Menolak non-admin tanpa site assignment dengan `403 "Anda belum ditugaskan ke site mana pun"` (fail closed); admin tanpa site tetap melihat seluruh kawasan dan admin dengan site terpilih ikut terfilter.
- Menambahkan `WSClientInfo.SiteIDs`, `CanReceiveSiteEvent`, dan `BroadcastToSite` pada `utils/websocket.go`. Admin menerima semua event; non-admin hanya event dari site yang ditugaskan; event tanpa `site_id` hanya dikirim ke admin.
- Mengubah `Hub.Register` agar menerima daftar site dan memuat `user_sites` saat handshake WebSocket (`routes/websocket.go`).
- Menambahkan `site_id` pada payload `CCTV_STREAM_LOG_CREATED`, `CCTV_LIVE_STATUS_UPDATED`, dan `CCTV_UPDATED`, lalu mengarahkan broadcast-nya melalui `BroadcastToSite` pada `stream_probe.go`, `stream_verify.go`, `health_score.go`, dan `cctv.go`.
- Memperbaiki pembacaan identitas WebSocket agar memakai `c.Locals("user_id")`/`c.Locals("role")` langsung dari koneksi, bukan locals `fiberctx` yang tidak pernah diisi.
- Menambahkan regression test dua site/two user: `cctv_stream_log_test.go`, `utils/websocket_test.go`, dan `routes/websocket_test.go`.

timestamp: 2026-10-27 22:35 UTC+8
kategori: Sedang, Backend (Security / Multi-Site Isolation / WebSocket)
file yang diubah:
- backend/handlers/cctv.go
- backend/handlers/cctv_stream_log.go
- backend/handlers/cctv_stream_log_test.go
- backend/handlers/health_score.go
- backend/handlers/stream_probe.go
- backend/handlers/stream_verify.go
- backend/routes/websocket.go
- backend/routes/websocket_test.go
- backend/utils/websocket.go
- backend/utils/websocket_test.go
- backend/changelog.md

alasan perubahan: Menutup kebocoran data log/event CCTV lintas site dan memastikan broadcast real-time mengikuti hak akses site pengguna.

---

## [0.64.31] - 2026-10-22

### Perbaikan trusted proxy rate limit dan penambahan sistem log event CCTV

**Masalah**:
1. Go Fiber backend belum mengatur `ProxyHeader: fiber.HeaderXForwardedFor` pada konfigurasi Fiber, sehingga IP peer Nginx (`172.28.0.10`) dianggap sebagai IP klien bagi semua request dan grup middleware `/api` membatasi total request seluruh pengguna menjadi 100/menit (memicu error 429 "Terlalu banyak permintaan" saat login).
2. Sistem belum memiliki pencatatan riwayat event perubahan status stream/koneksi kamera (kapan sebuah CCTV online, offline, atau siaga), sehingga perubahan status di lapangan tidak dapat dilacak secara historis.

**Perbaikan**:
- Menambahkan `ProxyHeader: fiber.HeaderXForwardedFor` di `main.go` agar backend membaca IP asli pengguna dari reverse proxy Nginx.
- Memperbaiki `limiterKey` pada `middleware/limiter.go` agar membedakan kuota per endpoint spesifik (`c.Path()`) dan menaikkan batas `GlobalLimiter` menjadi 300 request/menit.
- Membuat tabel migrasi `cctv_stream_logs` (`000028_cctv_stream_logs.up.sql`), model `models.CCTVStreamLog`, dan endpoint `GET /api/cctv-logs` (alias `/api/cctv/logs`).
- Mengintegrasikan pencatatan log otomatis dan event broadcast WebSocket `CCTV_STREAM_LOG_CREATED` pada `RunRTSPProber` dan `persistStreamVerification`.

timestamp: 2026-10-22 18:18 UTC+8
kategori: Sedang, Backend (Security / Rate Limit / CCTV Stream Logs)
file yang diubah:
- backend/main.go
- backend/middleware/limiter.go
- backend/middleware/limiter_test.go
- backend/models/cctv_stream_log.go
- backend/handlers/stream_probe.go
- backend/handlers/stream_verify.go
- backend/handlers/cctv_stream_log.go
- backend/handlers/cctv_stream_log_test.go
- backend/routes/cctv.go
- backend/migrations/000028_cctv_stream_logs.up.sql
- backend/migrations/000028_cctv_stream_logs.down.sql
- backend/changelog.md

alasan perubahan: Memperbaiki pemblokiran rate limit palsu pada proxy dan menyediakan pelacakan riwayat online/offline CCTV.

---

## [0.64.30] - 2026-10-22

### Dukungan sorting parameter status live pada Master CCTV

**Masalah**: Endpoint `GET /api/cctv` belum mendukung parameter pengurutan `sort_by` berdasarkan kondisi streaming (`stream_state` dan `stream_live`).

**Perbaikan**:
- Menambahkan `stream_state` dan `stream_live` ke dalam daftar `allowedSort` pada handler `GetAllCCTV`.

timestamp: 2026-10-22 16:15 UTC+8
kategori: Kecil, Backend (CCTV/API)
file yang diubah:
- backend/handlers/cctv.go
- backend/changelog.md

alasan perubahan: Mendukung pengurutan tabel Master CCTV berdasarkan kondisi live kamera.

---

## [0.64.29] - 2026-10-22

### Pembersihan teks mojibake pada riwayat pemeliharaan

**Masalah**: Alasan sinkronisasi status CCTV menyimpan karakter `â€”` (mojibake dari em dash) ketika tugas pemeliharaan gagal. Karakter yang sama juga muncul pada deskripsi Swagger dan beberapa komentar sumber.

**Perbaikan**:
- `syncReason` kegagalan checklist sekarang memakai `—`.
- Deskripsi endpoint `CompleteMaintenanceLog` pada anotasi, `docs.go`, `swagger.json`, dan `swagger.yaml` diseragamkan.
- Komentar `auth.go` dan `push_test.go` dibersihkan tanpa mengubah perilaku.
- Audit database terhadap seluruh kolom teks tidak menemukan nilai mojibake; data CCTV tetap utuh.

timestamp: 2026-10-22 13:10 UTC+8
kategori: Kecil, Backend (Text/Encoding)
file yang diubah:
- backend/handlers/maintenance.go
- backend/handlers/auth.go
- backend/utils/push_test.go
- backend/docs/docs.go
- backend/docs/swagger.json
- backend/docs/swagger.yaml
- backend/changelog.md

alasan perubahan: Menghilangkan karakter rusak pada teks yang dapat tersimpan/ditampilkan dan menjaga dokumentasi API tetap bersih.

---

## [0.64.28] - 2026-10-22

### Verifikasi stream memulihkan path MediaMTX yang hilang

**Masalah**: Setelah **Stop Semua Stream** atau restart MediaMTX, verifikasi manual dapat langsung mengembalikan `unknown` dengan pesan "path stream belum terdaftar di MediaMTX". Path sebenarnya dapat didaftarkan ulang, tetapi verifier hanya memeriksa path yang sudah ada sehingga tombol **Cek Status Live** tidak memulihkan kondisi tersebut.

**Perbaikan**:
- Saat `GetPath` mengembalikan 404, verifier mendaftarkan ulang path satu kali melalui konfigurasi RTSP yang sama (sub-stream, on-demand, `closeAfter: 10s`) lalu memeriksa ulang.
- Kegagalan pendaftaran ulang tetap menghasilkan `unknown` dan tidak menaikkan `stream_verify_failures`, karena bukan bukti kamera offline.
- Pesan diagnosis membedakan pendaftaran ulang gagal, path tetap tidak ditemukan, dan MediaMTX tidak dapat dihubungi.
- Test baru mencakup pendaftaran ulang berhasil, gagal, dan path tetap 404 setelah pendaftaran ulang.

timestamp: 2026-10-22 11:40 UTC+8
kategori: Sedang, Backend (Streaming/Reliability)
file yang diubah:
- backend/handlers/stream_verify.go
- backend/handlers/stream_verify_test.go
- backend/changelog.md
- docs/04-manual-tests/manual-test-cctv-active-verification.md
- docs/05-reports/CCTV-ACTIVE-VERIFICATION-2026-10-22.md

alasan perubahan: Membuat verifikasi manual dan periodik tetap bekerja setelah path MediaMTX dihapus, tanpa menimbulkan false offline atau menunggu re-registrar periodik.

---

## [0.64.27] - 2026-10-22

### Verifikasi aktif status streaming CCTV

**Masalah**: Status path MediaMTX `idle` pada mode on-demand tidak membuktikan kamera benar-benar mengirim video. Badge online/offline sebelumnya dapat menampilkan status pasif yang menyesatkan ketika tidak ada penonton, dan kegagalan sumber tidak memiliki jejak diagnosis yang terpisah.

**Perbaikan**:
- Migrasi `000027_stream_verification` menambahkan `stream_verified_state`, `stream_verified_at`, `stream_verified_method`, `stream_verify_failures`, dan `stream_verify_error` tanpa mengubah kolom probe pasif.
- Verifier periodik end-to-end memicu HLS MediaMTX, memeriksa `SourceReady()` dan track, menetapkan `unknown` untuk kegagalan koneksi, serta `offline` setelah dua kegagalan berturut-turut. Interval default 15 menit (5-30) dan concurrency default 1 (1-4).
- Endpoint admin `POST /api/streams/:id/verify` menjalankan verifikasi manual dan mengembalikan `tracks`, `duration_ms`, serta error diagnosis tanpa kredensial.
- `GET /api/streams/live-status` dan health monitor memakai status terverifikasi yang masih segar (30 menit) sebagai status efektif; summary menambahkan `verified_live`/`verified_offline`.
- `SourceOnDemandCloseAfter` diturunkan ke `10s` dan konfigurasi HLS memakai `hlsSegmentCount: 7` agar playlist low-latency valid dan NVR tidak dibuka serentak.
- Scheduler snapshot berhenti dengan satu peringatan bila MediaMTX terbukti tidak menyediakan endpoint snapshot; error 404 per kamera tidak lagi membanjiri log dan tidak memakai transcoding FFmpeg.
- Test baru mencakup prioritas status terverifikasi, timestamp kedaluwarsa/masa depan, ambang kegagalan, MediaMTX tidak terjangkau, pemicu HLS, dan registrasi route admin.

timestamp: 2026-10-22 11:05 UTC+8
kategori: Besar, Backend (Streaming/Reliability)
file yang diubah:
- backend/handlers/snapshot.go
- backend/handlers/stream.go
- backend/handlers/stream_probe.go
- backend/handlers/stream_verify.go
- backend/handlers/stream_verify_test.go
- backend/main.go
- backend/migrations/000027_stream_verification.up.sql
- backend/migrations/000027_stream_verification.down.sql
- backend/models/cctv.go
- backend/routes/stream.go
- backend/routes/stream_test.go
- backend/utils/mediamtx.go
- backend/utils/mediamtx_test.go
- backend/changelog.md

alasan perubahan: Memberi bukti end-to-end yang dapat dipercaya untuk status live/offline CCTV, memisahkan status terverifikasi dari probe pasif, dan menghindari false alarm akibat path on-demand yang idle.

---

## [0.64.26] - 2026-10-21

### Isolasi site dashboard, field maintenance, dan daftar site

**Masalah**: `GET /dashboard/summary`, `GET /dashboard/tasks`, seluruh grup `/field-maintenance`, dan daftar `/sites` belum menegakkan site aktif. Non-admin masih dapat melihat agregat/tugas/catatan dari site lain, membuat pengecekan untuk CCTV di site lain, serta melihat seluruh daftar site.

**Perbaikan**:
- `SiteContext` dipasang setelah autentikasi pada route dashboard summary/tasks dan seluruh grup field-maintenance; non-admin tanpa penugasan ditolak `403`.
- Distribusi status CCTV, tren pemeliharaan 6 bulan, pekerjaan overdue, dan daftar tugas dashboard difilter berdasarkan site aktif. Teknisi tetap dibatasi `technician_id`.
- Daftar, detail, laporan PDF, dan pembuatan field maintenance memakai scope site lewat relasi CCTV. Pembuatan untuk CCTV di site lain ditolak `403`.
- `/sites`, `/sites/all`, dan `/sites/{id}` memakai scope keanggotaan `user_sites`: non-admin hanya melihat site yang ditugaskan, admin tetap global. `/sites/all` sengaja tetap memuat seluruh site penugasan agar `SiteSwitcher` multi-site tidak rusak.
- Event WebSocket `AUDIT_LOG_ADDED` dibatasi ke koneksi admin; non-admin tidak lagi menerima metadata audit (IP, user-agent, deskripsi aktivitas).
- Test regresi ditambahkan untuk dashboard site scope, field maintenance list/create mismatch, daftar site non-admin, detail lintas site, dan route non-admin tanpa penugasan.
- Swagger diregenerasi dengan respons `403` dan deskripsi scope site.

timestamp: 2026-10-21 23:20 UTC+8
kategori: Sedang, Backend (Keamanan/Multi-site)
file yang diubah:
- backend/handlers/dashboard.go
- backend/handlers/dashboard_test.go
- backend/handlers/field_maintenance.go
- backend/handlers/field_maintenance_test.go
- backend/handlers/helpers_test.go
- backend/handlers/site.go
- backend/handlers/site_test.go
- backend/routes/dashboard.go
- backend/routes/field_maintenance.go
- backend/routes/routes_test.go
- backend/routes/websocket.go
- backend/utils/audit.go
- backend/docs/docs.go
- backend/docs/swagger.json
- backend/docs/swagger.yaml
- backend/changelog.md
- docs/01-documentation/AGENTS.md
- docs/02-audits/SECURITY-AUDIT.md
- docs/04-manual-tests/manual-test-security-p0-hardening.md

alasan perubahan: Menutup sisa kebocoran data lintas site pada dashboard, pengecekan lapangan, dan daftar site tanpa mengubah fitur atau alur kerja yang sudah ada.

---

## [0.64.25] - 2026-10-21

### Scope site default non-admin dan validasi sandi lama

**Masalah**: `SiteContext` membiarkan `site_id` kosong bila header `X-Site-ID`/cookie `sigap_site_id` tidak ada, sehingga non-admin dapat menerima data seluruh site. Pada modul CCTV, middleware site juga berjalan sebelum autentikasi sehingga pemeriksaan keanggotaan site tidak pernah dievaluasi. Selain itu, endpoint ganti sandi menerima `old_password` kosong untuk perubahan mandiri biasa.

**Perbaikan**:
- `SiteContext` memilih site penugasan default non-admin (site `is_default`, lalu nama site) saat konteks site tidak dikirim; akun tanpa penugasan ditolak `403` (fail closed).
- Urutan middleware CCTV diubah: autentikasi lebih dulu, baru `SiteContext`, sehingga membership site benar-benar diperiksa. Rute admin tetap memakai `RequireRole("admin")`.
- `SiteContext` dipasang pada grup `/zones`; daftar zona untuk dropdown ikut difilter berdasarkan site aktif.
- `ChangePassword` mewajibkan `old_password` kecuali pengguna dalam mode `must_change_password`; sandi baru juga ditolak bila sama dengan sandi sementara saat ini.
- Test middleware site (default scope, cookie fallback, tanpa penugasan, admin global) dan test ganti sandi (old password kosong, forced mode, sandi sama) ditambahkan, termasuk test registrasi rute CCTV setelah perubahan urutan middleware.
- Swagger diregenerasi; deskripsi change-password, filter `site_id`, dan endpoint readiness MediaMTX yang tertunda ikut tersinkron.

timestamp: 2026-10-21 21:35 UTC+8
kategori: Sedang, Backend (Keamanan/Multi-site)
file yang diubah:
- backend/middleware/site.go
- backend/middleware/site_test.go
- backend/routes/cctv.go
- backend/routes/routes_test.go
- backend/routes/zone.go
- backend/handlers/zone.go
- backend/handlers/auth.go
- backend/handlers/auth_extra_test.go
- backend/docs/docs.go
- backend/docs/swagger.json
- backend/docs/swagger.yaml
- backend/changelog.md
- docs/01-documentation/AGENTS.md
- docs/04-manual-tests/manual-test-security-p0-hardening.md

alasan perubahan: Menutup kebocoran data lintas site saat cookie site belum ada dan mencegah pengambilalihan sesi mengganti sandi tanpa mengetahui sandi lama.

---

## [0.64.24] - 2026-10-21

### Stabilkan probe awal setelah restart

**Masalah**: Saat restart host/container, API healthy lebih dulu daripada MediaMTX. Probe pertama langsung berjalan sebelum path on-demand terdaftar, sehingga 20 kamera aktif sementara ditulis sebagai `unknown`. Siklus berikutnya baru mengembalikannya ke `idle`.

**Perbaikan**:
- `RunRTSPProber` memakai `runInitialProbe`: menunggu MediaMTX siap (maksimal 2 menit) sebelum probe pertama.
- Setelah MediaMTX siap, `ReRegisterFailedStreams` dijalankan lebih dulu agar path kamera aktif terdaftar, baru probe dijalankan.
- Bila MediaMTX belum siap sampai timeout, probe awal dilewati. Status terakhir di database tidak ditimpa `unknown` hanya karena race startup.
- Menambahkan test urutan registrasi-sebelum-probe dan test probe awal dilewati saat MediaMTX tidak siap.
- Uji restart terkontrol: API mulai 20:05:31, MediaMTX mulai 20:05:37, probe pertama 20:05:42 menghasilkan `idle=20 unknown=0` (sebelumnya sempat `unknown=20`).

timestamp: 2026-10-21 20:10 UTC+8
kategori: Sedang, Backend (Streaming/Probe)
file yang diubah:
- backend/handlers/stream_probe.go
- backend/handlers/stream_probe_test.go
- backend/changelog.md
- docs/05-reports/DIAGNOSA-STREAMING-CCTV-2026-10-20.md

alasan perubahan: Menghilangkan status `unknown` palsu pada probe pertama setelah restart dan memastikan path aktif sudah terdaftar sebelum status dievaluasi.

---

## [0.64.23] - 2026-10-20

### Probe Metro-E pasif saat host tanpa rute OIKN

**Konteks**: Data master terbaru mengisi `ip_camera` 17 kamera Metro-E. Container API tidak dapat menjangkau `10.20.105.x` (timeout), sehingga probe langsung akan menandai 16 kamera aktif sebagai offline meskipun streaming lewat NVR publik masih bekerja.

**Perubahan**:
- Menambahkan `utils.MetroEDirectProbeEnabled()` dengan env `CCTV_METRO_E_DIRECT_PROBE`.
- `docker-compose.yml` default `false` untuk deployment ini; set `true` hanya jika host berada di jaringan OIKN.
- Saat flag false, kamera Metro-E memakai klasifikasi MediaMTX pasif yang sama dengan GSM: `live` jika path ready, `idle` jika path terdaftar tetapi belum berjalan, `offline` hanya jika ada penonton tetapi sumber tidak siap, dan `unknown` jika tidak dapat disimpulkan.
- `AddPath` tidak lagi mencetak ERROR untuk respons normal `path already exists`; `ensureStreamPath` tetap melakukan patch konfigurasi.
- Menambahkan test untuk flag dan fallback pasif Metro-E.
- Sinkronisasi data master 35 CCTV THD dilakukan terpisah di database (bukan migration): `sumber_listrik` PLTS/PLN, `tipe_koneksi` GSM/Metro-E/Tidak Ada Layanan, `ip_camera`, status, koordinat, dan keterangan.

timestamp: 2026-10-20 16:45 UTC+8
kategori: Sedang, Backend (Streaming/Probe)
file yang diubah:
- backend/utils/rtsp_probe.go
- backend/utils/rtsp_probe_test.go
- backend/utils/mediamtx.go
- backend/handlers/stream_probe.go
- backend/handlers/stream_probe_test.go
- docker-compose.yml
- docs/01-documentation/AGENTS.md
- docs/05-reports/DIAGNOSA-STREAMING-CCTV-2026-10-20.md
- backend/changelog.md

alasan perubahan: Menyimpan IP Metro-E tanpa menimbulkan offline palsu pada deployment yang belum memiliki rute ke jaringan OIKN.

---

## [0.64.22] - 2026-10-20

### Status probe on-demand: idle bukan offline

**Masalah**: Probe GSM hanya membaca path operasional MediaMTX. Path on-demand yang belum berjalan karena tidak ada penonton memiliki `ready=false`, lalu ditulis sebagai `stream_live=false` dan ditampilkan sebagai offline. Stream health monitor juga dapat memicu `STREAM_DOWN` palsu setelah 15 menit.

**Perbaikan**:
- Migration `000025_stream_state` menambahkan kolom `stream_state` (`live`, `idle`, `offline`, `unknown`); `stream_live` tetap dipertahankan untuk kompatibilitas.
- Probe GSM membedakan path terdaftar yang idle dari kegagalan terkonfirmasi (ada penonton tetapi sumber tidak siap). Kegagalan MediaMTX tidak lagi memaksa semua kamera GSM menjadi offline.
- `GetLiveStatus` mengembalikan `stream_state` dan ringkasan `idle`; status unknown dihitung terpisah dari offline.
- Stream health monitor tidak memulai countdown `STREAM_DOWN` untuk status `idle`/`unknown`, dan mereset countdown lama saat stream kembali idle.
- Test probe, status live, dan health monitor diperluas.

timestamp: 2026-10-20 18:20 UTC+8
kategori: Sedang, Backend (Streaming/Status)
file yang diubah:
- backend/migrations/000025_stream_state.up.sql
- backend/migrations/000025_stream_state.down.sql
- backend/models/cctv.go
- backend/handlers/stream_probe.go
- backend/handlers/stream.go
- backend/handlers/stream_probe_test.go
- backend/handlers/stream_test.go
- backend/handlers/stream_health_test.go
- backend/changelog.md

alasan perubahan: Mencegah kamera on-demand yang hanya siaga ditampilkan offline dan mencegah alarm STREAM_DOWN palsu.

---

## [0.64.21] - 2026-10-20

### Pilot THD-15 SD berhasil: sub-stream menjadi default

**Konteks**: Pilot THD-15 SD berhasil tampil di browser pengguna. MediaMTX mencatat sesi WHEP membaca satu track H.265; uji headless tanpa dukungan HEVC tetap ditolak `400 codecs not supported by client`. Kompatibilitas H.265 bergantung pada browser/perangkat.

**Perubahan**:
- `StartStream` memakai sub-stream sebagai default; `?quality=main` tetap tersedia untuk memilih HD/main.
- `StartAllStreams` dan `registerStreamWithRetry` memakai sub-stream sebagai default.
- Respons start mengembalikan `quality=sub` bila parameter tidak diberikan.
- Regression test default kualitas diperbarui.

timestamp: 2026-10-20 18:05 UTC+8
kategori: Kecil, Backend (Streaming)
file yang diubah:
- backend/handlers/stream.go
- backend/handlers/stream_test.go
- backend/changelog.md

alasan perubahan: Menindaklanjuti pilot THD-15 SD yang berhasil dan menjadikan sub-stream sebagai default operasional.

---

## [0.64.20] - 2026-10-20

### Streaming: pemilihan sub-stream dan update path MediaMTX

**Masalah**: Saat `rtsp_url` penuh tersimpan, `ResolveRTSPURL` mengembalikan URL apa adanya sehingga parameter `quality=main/sub` tidak pernah mengubah stream. Path yang sudah terdaftar juga hanya dianggap sukses saat `AddPath` mengembalikan `already exists`, sehingga perubahan kualitas tidak diterapkan. `ReRegisterFailedStreams` memakai daftar path operasional, padahal path on-demand tetap terdaftar meski sumbernya sedang idle.

**Perbaikan**:
- `ResolveRTSPURL` menyesuaikan suffix channel Hikvision `01`/`02` pada URL penuh sesuai `quality`; URL non-Hikvision atau pola yang tidak dikenali dikembalikan tanpa perubahan.
- Default kualitas tetap **main** sampai rollout encoding H.264 sub-stream selesai; ini mencegah seluruh kamera ikut pindah ke sub-stream H.265 sebelum pilot THD-15 lolos. Pengguna tetap dapat memilih SD secara eksplisit.
- Ditambahkan `PatchPath` pada klien MediaMTX dan helper `ensureStreamPath`: path baru ditambahkan, path lama di-patch agar sumber/kualitas terbaru benar-benar dipakai.
- `StartStream`, `StartAllStreams`, dan `registerStreamWithRetry` memakai helper tersebut.
- `ReRegisterFailedStreams` memakai `ListPathConfigs` sehingga path on-demand yang idle tidak dianggap hilang.
- Regression test untuk pemilihan main/sub, patch path, dan path yang sudah terdaftar.

timestamp: 2026-10-20 17:05 UTC+8
kategori: Kecil, Backend (Streaming)
file yang diubah:
- backend/utils/rtsp.go
- backend/utils/rtsp_test.go
- backend/utils/mediamtx.go
- backend/utils/mediamtx_test.go
- backend/handlers/stream.go
- backend/handlers/stream_test.go
- backend/changelog.md

alasan perubahan: Memastikan tombol SD/HD dan konfigurasi sub-stream benar-benar diterapkan, serta mengurangi false negative pada re-register path on-demand.

---

## [0.64.19] - 2026-10-20

### Perbaikan keamanan P0: metrics, diagnostik, dan URL RTSP

**Masalah**: `/api/metrics` bisa panic (500) saat entri histogram nil/panjang bucket tidak sesuai. Swagger, `/api/test-panic`, dan metrics masih dapat dijangkau dari luar bila `APP_ENV` kosong. URL RTSP penuh—termasuk `user:password@host`—tersimpan plaintext, dikembalikan API, dan disiarkan lewat WebSocket.

**Perbaikan**:
- `ObserveRequest` menormalkan entri histogram yang nil/korup; `Render` membaca bucket secara defensif sehingga tidak panic dan tetap mengembalikan metrik.
- Regression test untuk entri nil, bucket pendek, dan self-healing `ObserveRequest`.
- `APP_ENV` default `production` di Compose; Swagger dan `/api/test-panic` nonaktif di API.
- Nginx memblokir `GET /api/metrics`, `/api/test-panic`, dan seluruh `/swagger` dengan 404. Scraping internal Prometheus ke `api:3000` tetap berjalan.
- URL RTSP disimpan terenkripsi AES-256-GCM dengan prefix `enc:v1:`; migration startup mengenkripsi data legacy secara idempoten tanpa menghapus data.
- URL didekripsi hanya saat registrasi streaming; API hanya mengirim `rtsp_url_masked` + `rtsp_url_configured`, dan event WebSocket tidak lagi memuat model CCTV penuh.
- Detail error parsing body menyamarkan field `rtsp_url` agar URL/kredensial tidak bocor lewat pesan validasi.
- Histori kesehatan CCTV mendapat UUID otomatis lewat hook `BeforeCreate`; insert tidak lagi gagal `duplicate key` karena UUID nol.
- Validasi format `rtsp://`/`rtsps://` pada create/update.
- Port Prometheus/Grafana dibatasi ke `127.0.0.1`; password Grafana dibaca dari `GRAFANA_ADMIN_PASSWORD`, bukan `admin/admin`.

timestamp: 2026-10-20 13:05 UTC+8
kategori: Kecil, Backend (Security + Monitoring + CCTV)
file yang diubah:
- backend/monitoring/metrics.go
- backend/monitoring/metrics_test.go
- backend/utils/rtsp.go
- backend/utils/rtsp_test.go
- backend/models/cctv.go
- backend/models/cctv_health_history.go
- backend/models/cctv_health_history_test.go
- backend/handlers/cctv.go
- backend/handlers/cctv_security_test.go
- backend/handlers/cctv_sync.go
- backend/handlers/stream.go
- backend/handlers/request_error.go
- backend/handlers/request_error_test.go
- backend/config/cctv_rtsp_migration.go
- backend/main.go
- nginx/nginx.conf
- docker-compose.yml
- docker-compose.override.yml
- .env.example
- README.md
- docs/01-documentation/AGENTS.md
- docs/02-audits/SECURITY-AUDIT.md
- backend/changelog.md

alasan perubahan: Menutup panic metrics, eksposur endpoint diagnostik, kebocoran kredensial RTSP, dan akses default monitoring.

---

## [0.64.18] - 2026-10-20

### Perubahan: Form CCTV memakai URL RTSP saja

**Konteks**: Form frontend sebelumnya mengirim `channel_port`, `rtsp_password`, dan menampilkan preview URI. Alur operasional yang dipakai adalah URL RTSP penuh, sehingga tiga input itu duplikatif dan membingungkan.

**Perubahan**:
- Form tambah/edit CCTV hanya menampilkan satu input **URL RTSP Kamera**.
- `channel_port` dan `rtsp_password` tidak lagi dikirim oleh form web.
- URL wajib diisi saat streaming diaktifkan dan di-trim sebelum dikirim.
- Backend tetap menerima field lama agar data/API existing tidak rusak; tidak ada perubahan skema atau penghapusan data.

timestamp: 2026-10-20 12:08 UTC+8
kategori: Kecil, Frontend (Form CCTV + Streaming)
file yang diubah:
- frontend/src/app/cctv/page.tsx
- frontend/changelog.md
- backend/changelog.md
alasan perubahan: Menyelaraskan form dengan konfigurasi streaming berbasis URL RTSP penuh.

---

## [0.64.17] - 2026-10-20

### Peningkatan: Tombol detail pada seluruh toast error frontend

**Masalah**: Toast error di frontend hanya menampilkan satu kalimat. Saat backend mengembalikan field, penyebab, atau status HTTP, pengguna tidak bisa melihat konteksnya tanpa membuka DevTools.

**Perbaikan**:
- Toast error kini selalu memiliki tombol **Detail error**, termasuk error validasi yang dibuat langsung di halaman.
- Modal detail menampilkan pesan, field, penyebab, nilai, baris/kolom JSON, status HTTP, endpoint, request ID (bila ada), halaman, dan waktu.
- `ApiError` di frontend membawa metadata `status`, `endpoint`, `method`, `requestId`, `detail`, dan `errors` dari respons backend.
- Semua pemanggil `toast.error` yang sudah memiliki objek error meneruskan error tersebut sebagai argumen kedua; error validasi halaman tetap mendapat konteks halaman dan waktu.
- Toast error bertahan 9 detik dan ditutup saat modal detail dibuka agar tidak menutupi modal.
- Modal bisa ditutup dengan Escape/klik overlay, dan detail bisa disalin sebagai JSON.

timestamp: 2026-10-20 10:58 UTC+8
kategori: Sedang, Frontend (Error UX)
file yang diubah:
- frontend/src/components/Toast.tsx
- frontend/src/lib/api.ts
- frontend/src/app/{api-keys,cctv,field-maintenance,laporan-kustom,maintenance,pemberitahuan,schedules,sites,suku-cadang,users,webhooks,zona}/page.tsx
- frontend/src/components/{ChecklistTemplateManager,FieldMaintenanceModal,NotificationBell,StreamModal}.tsx
- backend/changelog.md
alasan perubahan: Setiap error harus bisa dijelaskan lokasi dan penyebabnya langsung dari toast, tanpa membuka DevTools.

---

## [0.64.16] - 2026-10-20

### Peningkatan: Pesan error parsing body request

**Masalah**: Saat `BodyParser` gagal, handler hanya mengembalikan pesan generik `"Format input tidak valid"` atau `"Payload tidak valid"`. Pengguna tidak tahu field mana yang salah dan apa penyebabnya, sehingga kasus seperti `zone_id: null` sangat membingungkan.

**Perbaikan**: Utility baru `sendBodyParseError` di `backend/handlers/request_error.go`:
- Mengidentifikasi nama field JSON yang bermasalah, termasuk error dari custom unmarshaler seperti `OptionalUUID` yang tidak membawa nama field.
- Menjelaskan penyebabnya: tipe data tidak sesuai, UUID tidak valid, JSON rusak (dengan baris/kolom), atau body kosong.
- Menyertakan nilai yang bermasalah pada pesan, tetapi meredaksi field sensitif seperti password/secret/token.
- Mengembalikan detail terstruktur di properti `detail` (`field`, `reason`, `value`, `line`, `column`).
- Menerjemahkan error `OptionalUUID` menjadi pesan yang bisa dipahami (`bukan UUID yang valid`, `harus berupa string UUID atau null`).

Seluruh handler yang memakai `c.BodyParser` diganti memakai helper ini. Frontend tidak perlu diubah karena toast sudah menampilkan `message` dari backend.

timestamp: 2026-10-20 10:20 UTC+8
kategori: Kecil, Backend (Error Handling)
file yang diubah:
- backend/handlers/request_error.go (baru)
- backend/handlers/request_error_test.go (baru)
- backend/utils/optional_uuid.go
- backend/utils/optional_uuid_test.go
- backend/handlers/*.go (pemakaian sendBodyParseError)
- backend/changelog.md
alasan perubahan: Pesan error parsing harus menjelaskan field dan penyebabnya agar masalah input bisa langsung diperbaiki tanpa menebak-nebak.

---

## [0.64.15] - 2026-10-20

### Fix: Form Tambah CCTV gagal saat Kawasan/Zona kosong

**Masalah**: Frontend mengirim `zone_id` dan `site_id` bernilai `null` ketika dropdown Kawasan/Zona dikosongkan. `OptionalUUID.UnmarshalJSON` tidak menangani JSON `null` (hanya string kosong), sehingga `uuid.Parse("null")` gagal dan `BodyParser` mengembalikan 400 "Format input tidak valid". Akibatnya pembuatan CCTV tidak bisa dilakukan selama belum ada zona yang dipilih.

**Perbaikan**: `OptionalUUID.UnmarshalJSON` kini mendeteksi JSON `null` secara eksplisit, mengosongkan `Value`, menandai `Null=true`, dan tetap `Present=true`. Ditambahkan unit test untuk `null`, string kosong, UUID valid, UUID invalid, field tidak dikirim, dan reset nilai lama.

timestamp: 2026-10-20 10:05 UTC+8
kategori: Kecil, Backend (Bug Fix)
file yang diubah:
- backend/utils/optional_uuid.go
- backend/utils/optional_uuid_test.go (baru)
- backend/changelog.md
alasan perubahan: Field Kawasan/Zona bersifat opsional; nilai `null` harus diterima dan tidak boleh menggagalkan parsing seluruh request CCTV.

---

## [0.64.14] - 2026-10-18

### Hardening Keamanan P0 (Tindak Lanjut Audit)

**P0-1 (Secret di docker-compose)**: Menghapus password database hardcoded (`DB_PASSWORD`) dari `docker-compose.yml`. Nilai kini hanya bersumber dari `.env` dan Docker secret (`./secrets/db_password`), mengikuti presedensi `GetSecret` (file secret > env).

**P0-2 (Env bocor ke build context)**: `backend/.dockerignore` diperketat (`.env` + `.env.*`) dan ditambahkan `.dockerignore` di root untuk mengecualikan `secrets/`, `**/.env*`, `nginx/certs/`, `*.key`, `*.pem` dari konteks build Docker.

**P0-3 (Kredensial di contoh env)**: `backend/.env.example` dibersihkan — VAPID public/private key asli dikosongkan, `RTSP_NVR_HOST`/`RTSP_NVR_USER` diganti placeholder (`192.168.1.100` / `GANTI_NVR_USER`). CATATAN: kunci VAPID & kredensial NVR yang pernah ter-commit WAJIB dirotasi, karena history git tidak dapat "di-un-leak".

**P0-4 (Cache service worker)**: `frontend/public/sw.js` ditulis ulang — request `/api/` SELALU network-only (fallback JSON 503), navigasi network-only (fallback HTML offline generik), HTML terautentikasi tidak lagi di-cache, pre-cache dibatasi manifest + ikon.

**P0-5 (SSRF webhook)**: Utility baru `backend/utils/ssrf.go` — `ValidateExternalURL` (hanya http/https), `isBlockedIP` (loopback/privat/link-local/multicast/unspecified/0.0.0.0-8/100.64-10), `NewSafeHTTPClient` (resolusi DNS dicek via DialContext anti-rebinding, `CheckRedirect` maks 3 + validasi ulang). Diintegrasikan di `backend/handlers/webhook.go` pada Create/Update (validasi URL) serta `deliverWebhook`/`TestWebhook` (klien aman).

**P0-6 (Isolasi multi-site)**: `backend/middleware/site.go` menambah `enforceSiteMembership` — user non-admin terautentikasi wajib terdaftar di `user_sites` untuk site yang diminta, else 403 "Akses ditolak: Anda tidak ditugaskan ke site ini".

**Catatan teknis**: `enforceSiteMembership` mengembalikan flag `handled` (bukan `error`) karena `c.Status(...).JSON(...)` mengembalikan `nil`; tanpa flag ini `SiteContext` akan lanjut `c.Next()` sehingga respons 403 tertimpa handler berikutnya.

timestamp: 2026-10-18 10:00 UTC+8
kategori: Besar, Backend (Security Hardening P0)
file yang diubah:
- docker-compose.yml (hapus DB_PASSWORD hardcoded)
- backend/.dockerignore (perketat .env.*)
- .dockerignore (baru, root)
- backend/.env.example (bersihkan VAPID/NVR)
- backend/utils/ssrf.go (baru)
- backend/utils/ssrf_test.go (baru)
- backend/handlers/webhook.go (integrasi SSRF guard)
- backend/middleware/site.go (enforceSiteMembership)
- backend/middleware/site_membership_test.go (baru)
- frontend/public/sw.js (network-only API)
- backend/changelog.md
alasan perubahan: Menutup temuan keamanan P0 dari audit (secret hardcoded, kebocoran env/credential, cache SW berisiko, SSRF, isolasi site tidak ditegakkan).

---

## [0.64.13] - 2026-10-20

### Penghapusan kredensial database hardcoded dan pengamanan artefak lokal

perubahan yang dilakukan: **Menghapus password database yang tertulis langsung di `docker-compose.yml` dan mencegah artefak debug/profil browser lokal ikut ter-commit.**

- `DB_PASSWORD` tidak lagi di-hardcode; nilai dibaca dari `backend/.env.docker` atau Docker secret `/run/secrets/db_password`.
- Menambahkan `.codex/`, `output/`, `tmp/`, serta cache build Go lokal ke `.gitignore`.
- Password lama tetap perlu dirotasi karena masih tercatat pada riwayat Git.

before:
- `docker-compose.yml` memuat `DB_PASSWORD=...` secara hardcoded.
- Profil browser `.codex/edge-cdp-profile` (±143 MB), `.codex/edge-debug-profile`, `output/`, dan `tmp/` berstatus untracked dan berpotensi ikut ter-commit.

after:
- `docker-compose.yml` hanya mengandalkan `env_file`/Docker secret.
- Artefak lokal tersebut diabaikan Git sehingga tidak dapat ikut ter-commit secara tidak sengaja.

timestamp: 2026-10-20 13:37 WITA
kategori: Kritis, Keamanan/Infrastruktur
file yang diubah:
- docker-compose.yml
- .gitignore
- backend/changelog.md
alasan perubahan: Mencegah kebocoran kredensial database dan mencegah artefak debug berukuran besar/profil browser masuk ke repositori.

---

## [0.64.12] - 2026-09-28

### Otorisasi inventaris dan edit CCTV opsional

perubahan yang dilakukan: **Mengunci inventaris suku cadang untuk admin dan memperbaiki parsing nilai kawasan/zona kosong saat mengedit CCTV.**

- Seluruh endpoint daftar, detail, tambah, edit, dan hapus inventaris suku cadang kini mewajibkan peran admin.
- Endpoint pencatatan penggunaan suku cadang pada pekerjaan pemeliharaan tetap tersedia untuk teknisi.
- Parser UUID opsional kini menerima JSON `null` dan string kosong sebagai perintah mengosongkan relasi.
- Ditambahkan pengujian HTTP 403 bagi teknisi serta regresi penyimpanan URL RTSP bersama `site_id`/`zone_id` kosong.
- Perilaku generator jadwal tanpa teknisi diverifikasi tetap menghasilkan tugas `scheduled`, bukan otomatis dikerjakan admin.

timestamp: 2026-09-28 WITA
kategori: Kritis, Backend (Manual Test Bug 14, 20, 21)
file yang diubah:
- backend/handlers/cctv_test.go
- backend/routes/routes_test.go
- backend/routes/spare_part.go
- backend/utils/optional_uuid.go
- backend/utils/optional_uuid_test.go

---

## [0.64.11] - 2026-09-28

### Kesiapan MediaMTX dan streaming lintas jaringan

perubahan yang dilakukan: **Memisahkan alamat MediaMTX native-local dan Docker, menambahkan pemeriksaan kesiapan, serta menyediakan ICE TCP fallback untuk jaringan yang memblokir UDP.**

- Backend native-local memakai MediaMTX pada `127.0.0.1:9997`; Compose mengirim alamat internal `mediamtx:9997` secara eksplisit.
- Endpoint admin `GET /api/streams/readiness` memeriksa MediaMTX tanpa mengekspos URL internal atau kredensial RTSP.
- `start-all` berhenti dengan pesan yang spesifik ketika MediaMTX tidak terjangkau, serta melewati CCTV yang belum memiliki channel port.
- MediaMTX mendengarkan ICE pada UDP dan TCP port 8189; Compose mewajibkan kandidat domain/IP yang dapat dijangkau browser.
- Ditambahkan pengujian kesiapan dan kegagalan start-all ketika MediaMTX tidak tersedia.

timestamp: 2026-09-28 WITA
kategori: Kritis, Backend/Infrastruktur (Manual Test Bug 08)
file yang diubah:
- .env.example
- backend/handlers/stream.go
- backend/handlers/stream_test.go
- backend/routes/stream.go
- backend/utils/mediamtx.go
- docker-compose.yml
- mediamtx.yml

---

## [0.64.10] - 2026-09-28

### Filter dan sortir kawasan aset CCTV

perubahan yang dilakukan: **Menerapkan kawasan pada query daftar CCTV dan menyediakan nama kawasan yang dapat dipakai frontend.**

- Endpoint daftar CCTV menerima filter `site_id` dan menerapkannya sebelum perhitungan total serta pagination.
- Relasi kawasan dimuat bersama CCTV agar nama kawasan tampil, bukan UUID atau label sementara.
- Pengurutan `site_name` dilakukan berdasarkan nama kawasan untuk seluruh data, bukan hanya baris pada halaman aktif.
- Ditambahkan pengujian handler untuk filter, relasi nama kawasan, dan sortir kawasan.

timestamp: 2026-09-28 WITA
kategori: Sedang, Backend (Manual Test Bug 04)
file yang diubah:
- backend/handlers/cctv.go
- backend/handlers/cctv_list_test.go
- backend/models/cctv.go

---

## [0.64.9] - 2026-09-28

### Perbaikan VAPID dan notifikasi jadwal harian

perubahan yang dilakukan: **Menyatukan konfigurasi Web Push di backend dan memastikan penugasan jadwal langsung menghasilkan notifikasi teknisi.**

- Endpoint `GET /api/push/public-key` menjadi sumber public key tunggal untuk frontend.
- Pasangan VAPID divalidasi sebagai key P-256 dan subject email dinormalisasi menjadi URI `mailto:`.
- Jadwal baru atau perubahan teknisi menyimpan notifikasi penugasan secara sinkron.
- Generator jadwal memakai `NotifyUserSync` agar in-app, WebSocket, dan Web Push berjalan pada jalur yang sama.
- Ditambahkan utilitas `go run ./cmd/vapid-keygen` untuk membuat pasangan key unik per environment.

timestamp: 2026-09-28 WITA
kategori: Tinggi, Backend (Manual Test Bug 11, 12, dan 15)
file yang diubah:
- backend/.env.example
- backend/cmd/vapid-keygen/main.go
- backend/handlers/maintenance_schedule.go
- backend/handlers/maintenance_schedule_test.go
- backend/handlers/push.go
- backend/handlers/push_test.go
- backend/handlers/schedule_runner.go
- backend/models/notification.go
- backend/routes/push.go
- backend/utils/push.go
- backend/utils/push_test.go

---

## [0.64.8] - 2026-09-28

### Perbaikan filter kawasan suku cadang dan notifikasi zona

perubahan yang dilakukan: **Menambahkan relasi kawasan pada inventaris suku cadang serta notifikasi sinkron untuk teknisi yang baru ditugaskan ke zona.**

- Migration `000026` menambahkan `site_id` tanpa menghapus data dan menempatkan data lama pada kawasan default (dinomori ulang dari `000025` agar tidak bentrok dengan migrasi stream_state).
- Endpoint daftar suku cadang memvalidasi serta menerapkan filter `site_id`, termasuk pagination yang konsisten.
- Penugasan zona membandingkan anggota lama dan baru agar notifikasi tidak dikirim berulang kepada teknisi yang sama.
- Notifikasi disimpan sebelum respons sukses dan tetap diteruskan melalui WebSocket serta Web Push.

timestamp: 2026-09-28 WITA
kategori: Sedang, Backend (Manual Test Bug 19 dan 22)
file yang diubah:
- backend/migrations/000026_spare_parts_site.up.sql
- backend/migrations/000026_spare_parts_site.down.sql
- backend/models/spare_part.go
- backend/models/notification.go
- backend/handlers/spare_part.go
- backend/handlers/spare_part_test.go
- backend/handlers/zone.go
- backend/handlers/zone_test.go

---

## [0.64.7] - 2026-09-27

### Perbaikan Bug Manual 16-17: otorisasi laporan dan PDF kustom

perubahan yang dilakukan: **Menambahkan pengujian route yang membuktikan teknisi ditolak dari API laporan serta mengganti tabel PDF kustom menjadi layout adaptif dengan pembungkusan teks dan pagination yang konsisten.**

before:
- Perlindungan `RequireRole("admin")` belum memiliki pengujian pada route laporan menggunakan token teknisi.
- Semua kolom PDF dibagi sama rata dan isi dipotong berdasarkan jumlah karakter, sehingga Nama dan IP NVR dapat terpotong atau tidak rapi.

after:
- Pengujian route memakai token teknisi valid dan memastikan `/reports/generate` mengembalikan HTTP 403 sebelum handler dijalankan.
- Lebar kolom dihitung dari label dan data, seluruh teks dibungkus, tinggi baris menyesuaikan isi, header diulang, dan footer menampilkan nomor halaman.

timestamp: 2026-09-27 WITA
kategori: Sedang, Backend (Authorization Test + PDF Reporting)
file yang diubah:
- backend/handlers/report_builder.go
- backend/handlers/report_builder_test.go
- backend/routes/routes_test.go

---

## [0.64.6] - 2026-09-26

### Perbaikan Bug Manual 01-03: pengalihan teknisi, PDF, dan kawasan

perubahan yang dilakukan: **API pengalihan tetap menolak teknisi aktif, filter kawasan maintenance mengikuti relasi CCTV pada daftar/CSV/PDF, sortir site diterapkan server-side, dan layout PDF dipadatkan menjadi tabel A4 yang membungkus seluruh isi.**

before:
- Filter maintenance dapat bergantung pada kolom site log yang tidak selalu mencerminkan kawasan CCTV saat ini; ekspor dan PDF tidak menerima scope kawasan dari UI.
- Daftar site selalu diurutkan nama naik dan menghitung teknisi dengan query per baris.
- Lebar tabel PDF melampaui area A4, posisi kop bergeser, isi sel panjang terpotong, serta header halaman lanjutan dapat memengaruhi gaya baris pertama.

after:
- Scope kawasan memakai `EXISTS` terhadap `cctvs.site_id` dan diterapkan konsisten pada daftar, ekspor CSV, serta laporan PDF.
- Sortir site memakai whitelist kolom dan arah di backend; jumlah teknisi dihitung dalam satu subquery.
- Lebar tabel tepat 182 mm, semua kolom membungkus teks, baris tidak terbelah antarhalaman, header tabel diulang, dan footer memberi nomor halaman.

timestamp: 2026-09-26 WITA
kategori: Sedang, Backend (Bug Fix + Reporting)
file yang diubah:
- backend/handlers/maintenance.go
- backend/handlers/maintenance_test.go
- backend/handlers/report.go
- backend/handlers/site.go
- backend/handlers/site_test.go

---

## [0.64.5] - 2026-08-21

### Bug BB-13: Dynamic Checklist Status Calculation

**Bug Fix**: Removed hardcoded check for exactly 9 items in maintenance completion logic. Status "completed" now applies when ALL items are OK, regardless of count (dynamic based on active template).

timestamp: 2026-08-21 14:30 UTC+8
kategori: Kecil, Backend (Bug Fix)
file yang diubah:
- backend/handlers/maintenance.go (CompleteMaintenanceLog function)
- backend/docs/docs.go (regenerated)
- backend/utils/optional_uuid.go (syntax fix)

---

## [0.64.4] - 2026-08-20

### Multi-Site Isolation & Security Fixes (BB-01 to BB-03 equivalent)

**Multi-Site Isolation**: Model `Site` (nama, kode unik, alamat, koordinat, `is_default`) + join table `UserSite`; migrasi `000018_multi_site.idempoten` menciptakan site default "Kawasan OIKN" dan mengaitkan semua data existing; middleware `SiteContext()` membaca `X-Site-ID` header, validasi UUID & keberadaan site, set `c.Locals("site_id")`, filter opsional ke `GetAllCCTV`, `GetAllMaintenanceLogs`, `GetAllZones` (hanya aktif bila site_id valid); handler/rute CRUD `/api/sites` (admin-only untuk tulis, dropdown untuk semua role); endpoint `POST /api/sites/:id/assign-users` (assign teknisi ke kawasan); seeder `EnsureDefaultSite(idempoten)`.

**Security Bug #1 (NVR Credential Removal)**: Removed hardcoded NVR IP `36.91.27.35` from code comments & documentation; credential now via environment variable `RTSP_NVR_USER` fallback to `<NVR_USERNAME>` placeholder; AES-256 encrypted `rtsp_password` di-exclude from all JSON responses (`json:"-"`); consistent secret management via Docker secrets (`./secrets/*`) or `.env`.

**Security Bug #2 (Metrics Protection)**: Middleware `IsTrustedMetricsAccess()` added to exclude non-trusted IPs from `/api/metrics` access (trusted: `127.0.0.1`, `::1`, `172.28.0.0/16`, `10.0.0.0/8`); protection prevents internal metrics exposure (DB pool, WS connections, storage bytes) to unauthorized external parties.

**Security Bug #3 (Backup Encryption)**: Updated `scripts/backup.sh` & `scripts/restore.sh` to support AES-256-GCM encryption via GPG (environment variable `BACKUP_ENCRYPTION_KEY`); database dump → gzip → encrypt → `backups/db/sigap_<TS>.sql.gz.gpg`; uploads tar+gzip→encrypt → `backups/uploads/uploads_<TS>.tar.gz.gpg`; restore reverses decryption; documentation `docs/BACKUP.md` expanded with encryption guide.

**Documentation**: Manual test V2 docs created in `Testing-Manual-V2/` directory for Multi-Site Isolation & Backup Encryption formal blackbox testing.

timestamp: 2026-08-20 19:45 UTC+8
kategori: Besar, Backend (Security Hardening + Multi-Site Feature)
file yang diubah:
- backend/models/site.go (baru)
- backend/migrations/000018_multi_site.up.sql (baru)
- backend/migrations/000018_multi_site.down.sql (baru)
- backend/middleware/site.go (baru)
- backend/middleware/site_test.go (baru)
- backend/handlers/site.go (baru)
- backend/handlers/site_test.go (baru)
- backend/routes/site.go (baru)
- backend/main.go (registrasi site routes + seeder)
- backend/utils/rtsp.go (credential updated)
- backend/models/cctv.go (site_id FK)
- backend/middleware/metrics.go (IsTrustedMetricsAccess added)
- scripts/backup.sh (GPG encryption support)
- scripts/restore.sh (GPG decryption support)
- docs/BACKUP.md (encryption section expanded)
- docs/changelog.md
- backend/changelog.md
alasan perubahan: Task 12.2 Multi-Site Isolation operational requirement per user prioritization; Security hardening compliance (remove hardcoded credentials, protect metrics, encrypt backups).

## [0.64.3] - 2026-08-20

### Perbaikan kontrak MediaMTX v1.20 dan status streaming

- Stop path memakai endpoint `DELETE /v3/config/paths/delete/{name}` dan 404 diperlakukan idempotent.
- Status API membedakan MediaMTX reachable, path registered, source available, dan viewer count.
- Field API v1.20 `available/online` serta `rtspTransport` menggantikan kontrak deprecated, dengan fallback `ready` untuk rolling upgrade.
- Respons start tidak lagi mengembalikan source URL RTSP yang dapat memuat decrypted credential.
- Unit dan integration test mencakup lifecycle add/list/delete serta response credential guard.

timestamp: 2026-08-20 WITA
kategori: Kritis, Backend (Streaming + Security)

## [0.64.2] - 2026-08-20

### Perbaikan lifecycle rotasi QR CCTV

perubahan yang dilakukan: **Rotasi QR kini membuat ciphertext baru dari ID CCTV yang sama dan scanner hanya menerima payload yang masih aktif.**

before:
- Rotasi mengenkripsi UUID acak yang tidak memiliki record CCTV, sehingga QR baru gagal dipindai.
- QR lama tetap menemukan CCTV karena scanner hanya memeriksa ID hasil dekripsi.

after:
- `RotateQRCode` mengenkripsi `cctv.ID.String()` dengan nonce AES-256-GCM baru.
- `ScanCCTV` membandingkan payload hasil pindai dengan `qr_payload` aktif dan menolak QR lama dengan 404.
- Endpoint gambar QR mengirim `Cache-Control: no-store` agar URL aset yang sama tidak mempertahankan label lama.
- Regression test membuktikan ciphertext berubah, hasil dekripsi tetap ID aset yang sama, payload aktif diterima, previous payload ditolak, dan HTTP cache dinonaktifkan.
- Swagger diregenerasi agar kontrak API menjelaskan rotasi ID CCTV yang sama dengan nonce baru.

timestamp: 2026-08-20 WITA
kategori: Kritis, Backend (Perbaikan + Security Hardening)
file yang diubah:
- backend/handlers/cctv.go
- backend/handlers/cctv_test.go
- backend/docs/docs.go
- backend/docs/swagger.json
- backend/docs/swagger.yaml

## [0.64.1] - 2026-08-19

### Koreksi identitas resmi SIGAP pada keluaran backend

perubahan yang dilakukan: **Menyeragamkan kepanjangan SIGAP menjadi "Sistem Informasi Gangguan & Aset Pusat" pada laporan PDF, template email, anotasi API, dan dokumentasi Swagger generated.**

before:
- Sebagian keluaran backend masih memakai kepanjangan SIGAP lama yang tidak sesuai keputusan produk.

after:
- Kop laporan PDF, footer/header email, anotasi Swagger, serta `docs.go`/`swagger.json`/`swagger.yaml` memakai nama resmi yang sama.

timestamp: 2026-08-19 WITA
kategori: Kecil, Backend (Copy/Branding)
file yang diubah:
- backend/handlers/report.go
- backend/main.go
- backend/templates/daily_summary.html
- backend/templates/maintenance_assigned.html
- backend/templates/overdue_escalation.html
- backend/templates/stock_low.html
- backend/templates/stream_down.html
- backend/docs/docs.go
- backend/docs/swagger.json
- backend/docs/swagger.yaml

## [0.64.0] - 2026-08-13

### Perbaikan: input kawasan/zona CCTV, checklist per kawasan, fix SLA 404 & probe offline (Sprint 40 cleanup)

perubahan yang dilakukan: **Menambahkan field kawasan (site) & zona pada create/update CCTV, mendukung template checklist berbeda per kawasan, mendaftarkan route `/api/dashboard/sla` yang selama ini hilang (404), serta memperbaiki deteksi live kamera Metro-E yang salah offline saat stream aktif di MediaMTX.**

before:
- `CreateCCTVRequest`/`UpdateCCTVRequest` tidak menerima `zone_id`/`site_id` — admin tidak bisa mengelompokkan CCTV ke kawasan/zona dari form.
- `checklist_templates` tidak memiliki relasi kawasan — daftar checklist sama untuk semua kawasan.
- Handler `GetDashboardSLA` sudah ada sejak Sprint 28 namun rutenya TIDAK terdaftar → halaman `/sla` selalu 404.
- Probe Metro-E hanya mengandalkan TCP dial ke IP kamera; bila IP tidak reachable dari server (mis. stream aktif via MediaMTX tapi probe gagal) kamera dianggap offline padahal stream berjalan.

after:
- **`handlers/cctv.go`**: `CreateCCTVRequest` & `UpdateCCTVRequest` + field `ZoneID`/`SiteID` (uuid opsional); `CreateCCTV` & `UpdateCCTV` menyimpan nilai tersebut ke model CCTV.
- **Migrasi `000024_checklist_site`**: kolom `site_id` (FK → sites, ON DELETE SET NULL, NULL = global) + index di `checklist_templates`.
- **`models/checklist_template.go`**: field `SiteID *uuid.UUID` (nil = template global).
- **`handlers/checklist_template.go`**: request create/update + `SiteID`; `GetAllChecklistTemplates` mendukung filter `?site_id=<uuid>` (mengembalikan template milik kawasan + global).
- **`routes/dashboard.go`**: daftarkan `GET /dashboard/sla` (admin) → `handlers.GetDashboardSLA` (fix 404).
- **`handlers/stream_probe.go`**: saat probe TCP Metro-E gagal, cek path MediaMTX — bila Ready=true, kamera tetap dianggap live (stream aktif = online).
- **Swagger regenerated** (`swag init`).

timestamp: 2026-08-13 19:45 WITA
kategori: Sedang, Backend (Perbaikan + Fitur kecil)
file yang diubah:
- backend/handlers/cctv.go
- backend/migrations/000024_checklist_site.up.sql (baru)
- backend/migrations/000024_checklist_site.down.sql (baru)
- backend/models/checklist_template.go
- backend/handlers/checklist_template.go
- backend/routes/dashboard.go
- backend/handlers/stream_probe.go
- backend/docs/* (swag init)
alasan perubahan: Melengkapi pengelompokan CCTV per kawasan/zona, checklist spesifik per kawasan, memperbaiki halaman SLA yang 404, dan akurasi status live kamera Metro-E.

## [0.63.0] - 2026-08-13

### Fitur: Sprint 37-PATCH — Probe Strategy Metro-E vs GSM

perubahan yang dilakukan: **Mem-patch RTSP Probe Engine agar membedakan kamera Metro-E vs GSM sesuai kebijakan docs/AGENTS.md § 11.2: Metro-E diprobe langsung (TCP dial port 554/80 dengan pengukuran latensi) setiap 2 menit; kamera GSM TIDAK pernah disentuh sama sekali — statusnya hanya dibaca pasif dari daftar path MediaMTX (lokal VPS), dengan jejak aktivitas `stream_last_active_at` untuk badge "pernah aktif".**

before:
- `runRTSPProbeCycle` memprobe SEMUA CCTV `stream_enabled` via `ProbeRTSPHost`+`ProbeRTSPChannel` ke host NVR tunggal (`RTSP_NVR_HOST`) — termasuk kamera GSM yang memakai kuota data seluler, melanggar kebijakan § 11.2 (dilarang ping/background-stream pada GSM).
- Belum ada kolom latensi ping (`ping_latency_ms`) maupun jejak terakhir stream aktif (`stream_last_active_at`); `GetLiveStatus` belum menampilkan tipe jaringan.

after:
- **`migrations/000022_stream_ping_fields.up.sql`** (baru): menambah kolom `ping_latency_ms INTEGER` (nullable) & `stream_last_active_at TIMESTAMPTZ` (nullable) pada tabel `cctvs`. `.down.sql` menghapusnya.
- **`models/cctv.go`**: menambah `PingLatencyMs *int` (json `ping_latency_ms,omitempty`) & `StreamLastActiveAt *time.Time` (json `stream_last_active_at,omitempty`).
- **`utils/rtsp_probe.go`**: menambah `IsMetroE(cctv)` — deteksi Metro-E dari prefix IP `10.20.105.`; `ProbeICMP(ip, timeout)` — probe konektivitas + ukur latensi via TCP dial port 554 dengan fallback port 80 (ICMP raw socket butuh privilege NET_RAW). Import `sigap-backend/models` (tanpa import cycle: models tidak mengimpor utils).
- **`handlers/stream_probe.go`**: `runRTSPProbeCycle` di-rewrite — sekali per siklus baca `streamClient.ListPaths()` → peta path Ready (error hanya log warning, GSM dianggap offline); per CCTV: Metro-E → `ProbeICMP(IP, 3s)` update `stream_live`+`ping_latency_ms`; GSM → cek `BuildStreamPath(cctvID)` di peta MediaMTX (live = exists && ready), update `stream_last_active_at` bila live. Status berubah → broadcast `CCTV_LIVE_STATUS_UPDATED` dengan payload `{cctv_id, stream_live, stream_checked_at, ping_latency_ms, network_type, stream_last_active_at, nama, kode_cctv}` + log 🟢/🔴. Status live bertahan → update tetap (refresh last_active). Ringkasan `✅ Probe cycle: N kamera dicek`. Helper baru `networkType(cctv)` ("metro-e"/"gsm"). `GetLiveStatus` menampilkan `stream_last_active_at`, `ping_latency_ms`, `network_type` per item. Tetap memakai `streamClient`/`setStreamClient` package-level (stream.go) agar testable.
- **Test**: `utils/rtsp_probe_test.go` (baru) — `IsMetroE` true/false/empty + `ProbeICMP` sukses (listener lokal port 80) / gagal (port 1). `handlers/stream_probe_test.go` — `TestRunProbeCycle_MetroE` (UPDATE stream_checked_at; ListPaths sekali per siklus, bukan per kamera), `TestRunProbeCycle_GSM` (path Ready → UPDATE stream_live=true + stream_last_active_at), `TestRunProbeCycle_GSMNotReady` (Ready=false → hanya checked_at), `TestRunProbeCycle_GSMReadyKeepsLive` (live bertahan → refresh), `TestNetworkType`, `TestGetLiveStatus_NetworkFields` (network_type/ping_latency_ms/stream_last_active_at di respons).

timestamp: 2026-08-13 16:00 WITA
kategori: Kecil, Backend (Probe strategy + Test)
file yang diubah:
- backend/migrations/000022_stream_ping_fields.up.sql (baru)
- backend/migrations/000022_stream_ping_fields.down.sql (baru)
- backend/models/cctv.go
- backend/utils/rtsp_probe.go
- backend/utils/rtsp_probe_test.go (baru)
- backend/handlers/stream_probe.go
- backend/handlers/stream_probe_test.go
- backend/changelog.md
alasan perubahan: Kebijakan § 11.2 melarang probe berkala ke kamera GSM (biaya kuota data seluler); hanya 17 kamera Metro-E yang boleh di-ping. Strategi GSM diubah menjadi pasif (baca status path MediaMTX lokal VPS + tracking last-active) sehingga konsumsi data GSM = nol.

## [0.62.1] - 2026-08-13

### Fix: StartNotificationCleanup memblokir main thread (API tidak listen)

perubahan yang dilakukan: **Menambahkan prefix `go` pada pemanggilan `handlers.StartNotificationCleanup(config.DB)` di `main.go` agar dijalankan sebagai goroutine. Tanpa ini, fungsi yang berisi loop `for range ticker.C` memblokir main thread sehingga aplikasi tidak pernah mencapai `app.Listen(":3000")` — API tidak mendengarkan port, healthcheck gagal, dan container `api` tidak pernah healthy (memblokir `mediamtx`/`nginx` via depends_on).**

before:
- `main.go:197` memanggil `handlers.StartNotificationCleanup(config.DB)` tanpa `go`.
- `StartNotificationCleanup` (handlers/notification.go) berisi `CleanupOldNotifications(db)` lalu `for range ticker.C { ... }` — loop tak berujung yang memblokir pemanggil.
- Akibat: startup berhenti setelah log "Worker pengiriman webhook diaktifkan"; log "Trusted proxy", "Swagger", "Server menyala" tidak pernah muncul; port 3000 tidak terbuka; container api unhealthy (ditemukan saat menjalankan stack Docker penuh).

after:
- `main.go:199` kini `go handlers.StartNotificationCleanup(config.DB)` — konsisten dengan fungsi background lain (`RunScheduleGenerator`, `RunOverdueChecker`, dll.) yang semuanya dijalankan sebagai goroutine.
- API berhasil listen di :3000, healthcheck `GET /api/health` hijau, dan chain dependency (api → mediamtx → nginx) berjalan normal.
- Diverifikasi saat `docker compose up -d` stack penuh: semua container Up, api healthy, login admin sukses.

timestamp: 2026-08-13 13:30 WITA
kategori: Kecil, Backend (Fix startup/blocking)
file yang diubah:
- backend/main.go
alasan perubahan: Startup API macet karena fungsi cleanup notifikasi yang blocking dijalankan di main thread; harus dijalankan sebagai goroutine seperti pola fungsi background lainnya.

## [0.62.0] - 2026-08-13

### Fitur: Sprint 39 — Endpoint tren AI analytics + RBAC halaman (Task 14C.3)

perubahan yang dilakukan: **Menambahkan endpoint tren hitungan orang unik harian per kamera `GET /api/ai/counts/:id/trend` (admin-only) beserta unit test, untuk mendukung halaman admin AI Analytics (`/ai-analytics`) pada Task 14C.3 Sprint 39.**

before:
- Backend hanya menyediakan ringkasan hitungan hari ini (`GET /api/ai/counts`) dan detail per kamera (`GET /api/ai/counts/:id`) — belum ada data historis/tren per kamera untuk halaman analytics.
- Frontend tidak memiliki halaman admin AI Analytics (dibuat di `frontend/` — lihat docs/changelog).

after:
- **`handlers/ai.go`**: handler baru `GetAICountTrend` — validasi UUID `:id` (invalid → 400), query `days` via `c.QueryInt("days", 7)` di-clamp 1–90, query `config.DB.Where("cctv_id = ? AND session_date >= ?", id, time.Now().AddDate(0, 0, -days)).Order("session_date ASC").Find(&sessions)` pada `ai_detection_sessions`, respons `{status:"success", data:[{date, unique_count}]}` (ascending by date). Swagger annotations (Tags "AI Detection", `@Router /ai/counts/{id}/trend [get]`, admin-only) + `swag init`.
- **`routes/ai.go`**: rute baru `ai.Get("/counts/:id/trend", middleware.RequireRole("admin"), handlers.GetAICountTrend)` didaftarkan tepat setelah `/counts/:id` — admin-only, konsisten dengan endpoint AI lainnya.
- **`handlers/ai_test.go`**: `TestGetAICountTrend` (UUID valid + `days=7` → 200, mock `SELECT * FROM "ai_detection_sessions" WHERE cctv_id = $1 AND session_date >= $2 ORDER BY session_date ASC` dengan 3 baris sesi → `data` panjang 3, urutan ascending, field `date` & `unique_count` terisi) dan `TestGetAICountTrend_InvalidID` (UUID invalid → 400 tanpa query DB). Memakai helper yang ada: `setupMockGORM`, `newTestApp`, `withUserLocals("admin")`, pola kolom baris sesi `id,cctv_id,session_date,unique_person_count,last_tracking_ids,created_at,updated_at`.
- **`backend/docs/*`**: regenerated via `swag init` (endpoint `/api/ai/counts/{id}/trend` masuk Swagger).

timestamp: 2026-08-13 22:00 WITA
kategori: Kecil, Backend (Fitur + Test)
file yang diubah:
- backend/handlers/ai.go
- backend/routes/ai.go
- backend/handlers/ai_test.go
- backend/docs/docs.go (swag init)
- backend/docs/swagger.json (swag init)
- backend/docs/swagger.yaml (swag init)
alasan perubahan: Halaman admin AI Analytics (Task 14C.3) membutuhkan data tren 7 hari per kamera; endpoint baru menyediakan data tersebut dengan validasi & RBAC yang konsisten dengan endpoint AI yang ada.

## [0.61.1] - 2026-08-13

### Fix: errcheck CI & RBAC Stream Admin-Only (Sprint 38 lanjutan)

perubahan yang dilakukan: **Memperbaiki kegagalan CI golangci-lint (errcheck) pada `handlers/ai.go` dan menuntaskan RBAC streaming menjadi admin-only sesuai aturan AGENTS.md & keputusan arsitektur ROADMAP.**

before:
- `handlers/ai.go` memiliki 3 pemanggilan `resp.Body.Close()` tanpa pengecekan error → linter `errcheck` gagal (job CI `golangci-lint` red).
- `routes/stream.go` mendaftarkan seluruh endpoint `/streams/*` TANPA `Protected()`/`RequireRole("admin")` — streaming dapat diakses publik, bertentangan dengan AGENTS.md ("Streaming CCTV: seluruh endpoint streaming admin-only") dan keputusan arsitektur Fase 9 ("Akses Viewer: Admin Only"). Test `routes/stream_test.go` justru menguji perilaku publik (TestStreamRoutesPublic).

after:
- **`handlers/ai.go`**: ketiga pemanggilan `resp.Body.Close()` kini memakai pola aman `_ = resp.Body.Close()` (untuk per-id di `ProxyActivateAISession`) dan `defer func() { _ = resp.Body.Close() }()` (untuk `ProxyDeactivateAISession` & `ProxyListAISessions`) — konsisten dengan pola `handlers/webhook.go`. `golangci-lint run ./...` → **0 issues**.
- **`routes/stream.go`**: grup `/streams` kini memakai `middleware.Protected()` + `middleware.RequireRole("admin")` — seluruh endpoint (`/status`, `/health`, `/live-status`, `/start-all`, `/stop-all`, `/:id/start`, `/:id/stop`, `/:id/url`) admin-only.
- **`routes/stream_test.go`**: `TestStreamRoutesPublic` dihapus (perilaku publik tidak lagi valid); diganti `TestStreamRoutesRequireAuth` (tanpa sesi → 401 via Protected) + `TestStreamRoutesRequireAdmin` dipertahankan (teknisi → 403 via RequireRole).

timestamp: 2026-08-13 16:30 WITA
kategori: Kecil, Backend (Fix CI + Security/RBAC)
file yang diubah:
- backend/handlers/ai.go
- backend/routes/stream.go
- backend/routes/stream_test.go
alasan perubahan: CI golangci-lint failing karena errcheck; dan menutup celah keamanan akses publik ke API kontrol streaming (harus admin-only sesuai aturan proyek).

## [0.61.0] - 2026-08-13

### Fitur: AI Microservice & Person Counting (Sprint 38 - Task 14B.1, 14B.2 & 14B.3)

perubahan yang dilakukan: **Menambahkan service AI deteksi orang (YOLOv8n + ByteTrack) di folder `ai/` (FastAPI) dan integrasi backend Go: migrasi tabel `ai_detection_sessions` & `ai_detection_events`, endpoint internal `POST /api/ai/detection-event` (diproteksi `X-AI-Secret`), proxy session AI (`POST/DELETE/GET /api/ai/sessions`), serta ringkasan hitungan orang unik per hari (`GET /api/ai/counts`, `GET /api/ai/counts/:id`).**

before:
- Belum ada deteksi orang real-time; tidak ada cara menghitung orang unik per kamera per hari.
- Tidak ada endpoint untuk mengaktifkan/menghentikan session AI maupun menerima hasil deteksi dari service eksternal.

after:
- **Service `ai/`** (baru, root repo): `ai/Dockerfile` (python:3.11-slim + lib OpenCV, pre-download `yolov8n.pt`), `ai/requirements.txt` (fastapi, uvicorn, ultralytics 8.2.0, opencv-python-headless, numpy, httpx, supervision 0.20.0, python-dotenv), `ai/core/` (`detector.py` — YOLOv8n hanya class 0, conf ≥ 0.50; `tracker.py` — ByteTrack via supervision; `person_counter.py` — unique person per hari, thread-safe, auto-reset; `session_manager.py` — `CameraSession` async: pull RTSP `rtsp://mediamtx:8554/cam-{cctv_id}`, throttle ~1 fps, bbox ternormalisasi 0–1, POST hasil ke backend dengan `X-AI-Secret`), `ai/routers/` (`session.py`, `health.py`), `ai/main.py` (FastAPI + daily reset 00:00 WITA), `ai/README.md`.
- **`docker-compose.yml`**: service `sigap_ai` (build `./ai`, network `sigap_network`, depends_on `mediamtx`+`api`, env `BACKEND_URL`/`MEDIAMTX_RTSP_URL`/`AI_SECRET`, resource limits 2 CPU/2048M, port 8010 internal TIDAK dipublish).
- **`migrations/000020_ai_detection_sessions.up/down.sql`** (baru): tabel `ai_detection_sessions` (cctv_id FK, session_date, unique_person_count, last_tracking_ids jsonb, UNIQUE (cctv_id, session_date)).
- **`migrations/000021_ai_detection_events.up/down.sql`** (baru): tabel `ai_detection_events` (cctv_id FK, event_type, person_count, unique_count, confidence_avg, detected_at + index).
- **`models/ai_detection.go`** (baru): `AIDetectionSession` & `AIDetectionEvent` (tidak me-embed Base — tanpa soft delete; ID diisi `uuid.New()` di handler).
- **`handlers/ai.go`** (baru): `ReceiveDetectionEvent` (validasi `X-AI-Secret` → 401 bila salah; parse & validasi UUID; upsert sesi harian — baris baru via `Create()` karena GORM `Save()` selalu UPDATE untuk ID non-zero; insert event hanya bila `person_count > 0`; broadcast `AI_DETECTION_UPDATE` via WebSocket), `GetAICounts` (admin, join CCTV), `GetAICountByCCTV` (admin), `ProxyActivateAISession` (`POST /api/ai/sessions` body `{active_cctv_ids:[...]}` → forward per-id ke `http://sigap_ai:8010/sessions`), `ProxyDeactivateAISession`, `ProxyListAISessions`. Swagger annotations + `swag init`.
- **`routes/ai.go`** (baru): `SetupAIRoutes` — `/ai/detection-event` (internal, X-AI-Secret), `/ai/counts` & `/ai/counts/:id` (admin), `/ai/sessions` POST (protected semua role) / DELETE (protected) / GET (admin).
- **`main.go`**: registrasi `routes.SetupAIRoutes(api)` + log "🤖 Rute AI detection diaktifkan".
- **`.env.example` (root)** & **`backend/.env.docker.example`**: tambah `AI_SECRET` (shared secret, fallback `sigap-ai-secret-change-me`) & `AI_SERVICE_URL` (default `http://sigap_ai:8010`).
- **Test**: `handlers/ai_test.go` (baru) — `ReceiveDetectionEvent` (secret benar/salah 401, UUID invalid 400, sesi baru & existing, insert event), `GetAICounts` (empty & with data), `GetAICountByCCTV` (sukses & invalid).

timestamp: 2026-08-13 15:45 WITA
kategori: Besar, Backend + AI Service + Infra
file yang diubah:
- ai/Dockerfile (baru)
- ai/requirements.txt (baru)
- ai/main.py (baru)
- ai/core/detector.py (baru)
- ai/core/tracker.py (baru)
- ai/core/person_counter.py (baru)
- ai/core/session_manager.py (baru)
- ai/routers/session.py (baru)
- ai/routers/health.py (baru)
- ai/README.md (baru)
- docker-compose.yml
- backend/migrations/000020_ai_detection_sessions.up.sql (baru)
- backend/migrations/000020_ai_detection_sessions.down.sql (baru)
- backend/migrations/000021_ai_detection_events.up.sql (baru)
- backend/migrations/000021_ai_detection_events.down.sql (baru)
- backend/models/ai_detection.go (baru)
- backend/handlers/ai.go (baru)
- backend/handlers/ai_test.go (baru)
- backend/routes/ai.go (baru)
- backend/main.go
- backend/.env.docker.example
- .env.example
- backend/docs/* (swag init)

## [0.60.0] - 2026-08-13

### Fitur: RTSP Live Status Detection (Sprint 37 - Task 14A.1 & 14A.3)

perubahan yang dilakukan: **Menambahkan RTSP Probe Engine yang secara berkala (tiap 2 menit) memeriksa ketersediaan host NVR & channel kamera lalu menyimpan hasilnya pada kolom `stream_live` & `stream_checked_at` di `cctvs`. Endpoint admin `GET /streams/live-status` menampilkan ringkasan online/offline/unchecked. Memperbaiki guardian bug pada StreamHealthMonitor sehingga kamera yang tidak pernah ditonton kini tetap dapat terdeteksi down.**

before:
- Kolom `stream_live`/`stream_checked_at` belum ada; tidak ada mekanisme deteksi live kamera berbasis probe RTSP TCP.
- `evaluateStreamState` di StreamHealthMonitor hanya menghitung durasi down bila stream pernah terlihat up (`streamLastReady`); kamera yang tidak pernah ditonton tidak pernah dianggap down (guardian bug baris 694-696).

after:
- **`migrations/000019_stream_live_status.up.sql`** (baru): menambah kolom `stream_live boolean NOT NULL DEFAULT FALSE` & `stream_checked_at timestamptz` pada tabel `cctvs`. `.down.sql` menghapusnya.
- **`models/cctv.go`**: menambah `StreamLive bool` (json `stream_live`) & `StreamCheckedAt *time.Time` (json `stream_checked_at,omitempty`).
- **`utils/rtsp.go`**: menambah `GetRTSPHost()` (baca env `RTSP_NVR_HOST`, fallback `36.91.27.35`).
- **`utils/rtsp_probe.go`** (baru): `ProbeRTSPHost(host, port)` (TCP dial timeout 3 detik) dan `ProbeRTSPChannel(host, port, channelID)` (kirim `OPTIONS` RTSP, cek response berisi `200`, deadline 5 detik).
- **`handlers/stream_probe.go`** (baru): `RunRTSPProber(db)` goroutine interval 2 menit → `runRTSPProbeCycle(db)` memprobe tiap CCTV `stream_enabled`, meng-update `stream_live`/`stream_checked_at`, broadcast `CCTV_LIVE_STATUS_UPDATED`, dan log online/offline. `GetLiveStatus` endpoint admin mengembalikan `{status, summary:{online,offline,unchecked}, data}`.
- **`routes/stream.go`**: menambah rute statis `GET /streams/live-status` (didaftarkan sebelum rute dinamis `/:id/...`).
- **`main.go`**: memanggil `go handlers.RunRTSPProber(config.DB)` + log "📡 RTSP Probe Engine aktif (interval: 2 menit)" setelah `RunStreamReRegistrar`.
- **`handlers/stream.go`** (fix 14A.3): menghapus guardian `if _, wasUp := streamLastReady[path]; !wasUp { return }`; `evaluateStreamState` kini memakai `cctv.StreamLive` dari DB sebagai sumber kebenaran saat path tidak siap (down countdown tanpa menunggu viewer pernah ada), logika webhook/email/health_score dipertahankan utuh, dan `live==true` mereset state down (mirip blok `isReady`).
- **Test**: `streamCCTVRow` di `handlers/stream_test.go` menjadi variadic untuk mendukung `stream_live`/`stream_checked_at`. `handlers/stream_health_test.go` `TestRunStreamHealthOnce_DownAndRestored` diperbarui (DOWN pakai `stream_live=false`, RESTORE pakai `stream_live=true`). `handlers/stream_probe_test.go` (baru) menguji `ProbeRTSPHost` & `ProbeRTSPChannel` sukses/gagal via mock TCP listener.

timestamp: 2026-08-13 12:00 WITA
kategori: Besar, Backend
file yang diubah:
- backend/models/cctv.go
- backend/utils/rtsp.go
- backend/utils/rtsp_probe.go
- backend/handlers/stream_probe.go
- backend/handlers/stream_probe_test.go
- backend/handlers/stream.go
- backend/handlers/stream_test.go
- backend/handlers/stream_health_test.go
- backend/routes/stream.go
- backend/migrations/000019_stream_live_status.up.sql
- backend/migrations/000019_stream_live_status.down.sql
- backend/main.go

## [0.59.0] - 2026-08-13

### Fitur: Multi-Site (Sprint 30 - Task 12.2)

perubahan yang dilakukan: **Menambahkan kawasan operasional (site) sebagai kelompok data CCTV, log pemeliharaan, dan zona melalui tabel baru `sites` + join `user_sites`, lengkap dengan middleware `SiteContext`, CRUD `/api/sites`, dan filter opsional per-site.**

before:
- Seluruh data CCTV/log/zona tidak dikelompokkan per kawasan; tidak ada konsep site. Semua user dapat melihat seluruh data tanpa batasan kawasan.
- Tidak ada endpoint untuk mengelola kawasan operasional maupun menugaskan teknisi ke suatu kawasan.

after:
- **`models/site.go`** (baru): model `Site` (nama, kode unik, alamat, koordinat, `is_default`) + join table `UserSite`. Menambah kolom `SiteID *uuid.UUID` pada `models/cctv.go`, `models/setup.go` (`MaintenanceLog`), dan `models/zone.go`.
- **`migrations/000018_multi_site.up.sql`** (baru): membuat tabel `sites` & `user_sites`, menambah kolom `site_id` ke `cctvs`/`maintenance_logs`/`zones` (idempoten), menyisipkan site default `OIKN`, dan menetapkan site default ke semua data existing yang masih NULL. `.down.sql` menghapusnya.
- **`middleware/site.go`** (baru): `SiteContext()` membaca `X-Site-ID` (fallback `?site_id=`), validasi UUID & keberadaan site, lalu set `c.Locals("site_id", ...)`. Tanpa header → dibiarkan kosong (tanpa filter) agar tidak mengubah perilaku existing.
- **`handlers/site.go`** (baru): `GetAllSites` (paginasi + `user_count`), `GetAllSitesForSelect` (dropdown), `CreateSite` (validasi kode unik incl soft-delete via Unscoped), `UpdateSite`, `DeleteSite` (cegah hapus site default), `AssignUsersToSite` (validasi role teknisi, timpa assignment). Semua operasi tulis mencatat audit via `utils.LogAudit`.
- **`routes/site.go`** (baru): `SetupSiteRoutes` — `/api/sites` dengan `Protected()` + `SiteContext()`; rute statis `/all` sebelum `/:id`.
- **`main.go`**: memanggil `routes.SetupSiteRoutes(api)` dan seeder `models.EnsureDefaultSite(config.DB)`.
- **`models/seeder.go`**: `EnsureDefaultSite(db)` idempoten memastikan site default OIKN ada saat startup (guard anti-error bila tabel belum ada).
- **Filter opsional `site_id`**: ditambahkan ke `GetAllCCTV`, `GetAllMaintenanceLogs`, `GetAllZones` — hanya aktif bila `c.Locals("site_id")` terisi (X-Site-ID valid disediakan). Tanpa header, query tetap sama sehingga perilaku/test existing tidak berubah. (Tidak diterapkan ke `GetAllFieldMaintenances` karena tabel `field_maintenances` tidak memiliki kolom `site_id`.)
- **Unit test**: `handlers/site_test.go` (Create/Get/Update/Delete/Assign, termasuk kode duplikat, nama kosong, hapus site default, user bukan teknisi) dan `middleware/site_test.go` (tanpa header, header valid, query valid, header invalid, site tidak ditemukan). Menambah `routes/routes_test.go` `TestSiteRoutesRequireAuthentication`.

timestamp: 2026-08-13 10:00 WITA
kategori: Besar, Backend
file yang diubah:
- backend/models/site.go
- backend/models/cctv.go
- backend/models/setup.go
- backend/models/zone.go
- backend/models/seeder.go
- backend/migrations/000018_multi_site.up.sql
- backend/migrations/000018_multi_site.down.sql
- backend/middleware/site.go
- backend/middleware/site_test.go
- backend/handlers/site.go
- backend/handlers/site_test.go
- backend/handlers/cctv.go
- backend/handlers/maintenance.go
- backend/handlers/zone.go
- backend/routes/site.go
- backend/routes/routes_test.go
- backend/main.go

## [0.58.0] - 2026-08-13

### Fitur: Filter Periode Kinerja Teknisi (Sprint 36 - P18a, Task 13.18)

perubahan yang dilakukan: **Menambahkan query parameter `?period=7d|30d|90d|365d` pada endpoint `GET /dashboard/technician-performance` untuk memilih preset rentang berjalan (default 30 hari).**

before:
- Endpoint kinerja teknisi hanya menerima `start_date`/`end_date` dengan default 30 hari terakhir; frontend harus menghitung tanggal sendiri untuk preset.

after:
- **`handlers/dashboard.go`** (`GetTechnicianPerformance`):
  - Membaca `period := c.Query("period", "")` dan mencocokkannya ke map preset `{7d:7, 30d:30, 90d:90, 365d:365}`.
  - Bila `period` valid, `startDate = now.AddDate(0,0,-periodDays)` dan `endDate = now`, mengesampingkan `start_date`/`end_date`.
  - Bila `period` kosong/tidak valid, tetap memakai perilaku existing (`start_date`/`end_date`, default 30 hari).
  - Menambahkan anotasi Swagger `@Param period query string false "Preset periode (7d/30d/90d/365d)"`.
- **Unit test** `handlers/technician_performance_test.go`: `TestTechnicianPerformanceWithPeriod` — request `?period=7d` menghasilkan query 5 argumen (now + dua pasang tanggal) dan sukses 200.

timestamp: 2026-08-13 08:00 WITA
kategori: Kecil, Backend
file yang diubah:
- backend/handlers/dashboard.go
- backend/handlers/technician_performance_test.go

## [0.57.0] - 2026-08-12

### Fitur: Kompresi Foto Upload & Thumbnail (Sprint 35 - P13a, Task 13.13)

perubahan yang dilakukan: **Menambahkan kompresi gambar server-side (resize + re-encode JPEG) dan pembuatan thumbnail untuk foto bukti pemeliharaan.**

before:
- Foto yang diunggah hanya di-re-encode ke JPEG quality 88 tanpa resize — ukuran file bergantung resolusi sumber, berpotensi besar & boros bandwidth.
- Tidak ada thumbnail; frontend tidak dapat menampilkan gambar kecil tanpa memuat file penuh.

after:
- **Dependency**: tambah `github.com/disintegration/imaging v1.6.2` untuk resize/thumbnail (pure Go).
- **`handlers/upload.go`**:
  - `saveCompressed(img, dstPath)`: resize gambar ke maks `maxDimension=1920` (pertahankan aspect ratio, filter Lanczos) bila lebih besar, lalu encode JPEG quality 85 & simpan; mengembalikan ukuran file.
  - `generateThumbnail(img, thumbPath)`: buat thumbnail `imaging.Fit(img, 400, 300, Lanczos)`, encode JPEG quality 80, simpan ke `thumbDir`.
  - `UploadFile`: memanggil keduanya, menyimpan fullsize + thumbnail; menambah header respons `X-Original-Size` & `X-Compressed-Size`; respons `data` kini memuat `thumb_url`.
  - `jpegEncode` quality diubah 88 → 85.
  - `ServeUploadedThumbnail`: endpoint pengambilan thumbnail (anti path-traversal, hanya `.jpg`).
- **`routes/upload.go`**: tambah `GET /api/uploads/thumbnails/:filename` (protected JWT).
- **Unit test** `handlers/upload_test.go`: `TestUploadCompress` — gambar besar (>300KB) → assert terkompresi, header `X-Compressed-Size`, thumbnail terbuat, cleanup direktori dengan `defer os.RemoveAll`.
- **`main.go`**: direktori `./uploads/thumbnails` dibuat otomatis oleh `generateThumbnail` (tidak bergantung pada startup).

timestamp: 2026-08-12 11:00 WITA
kategori: Sedang, Backend
file yang diubah:
- backend/go.mod
- backend/go.sum
- backend/handlers/upload.go
- backend/handlers/upload_test.go
- backend/routes/upload.go

## [0.56.0] - 2026-08-12

### Fitur: Checklist Template yang Dapat Dikustomisasi (Sprint 35 - P12a, Task 13.12)

perubahan yang dilakukan: **Menambahkan template checklist pengecekan lapangan yang dapat dikustomisasi admin (CRUD + reorder) dan kolom wajib pada hasil pengecekan.**

before:
- Checklist 9 item pengecekan lapangan tidak dikelola admin; tidak dapat diubah tanpa deploy ulang.
- `FieldMaintenanceCheck` belum menyimpan apakah sebuah item bersifat wajib.

after:
- **Model** `ChecklistTemplate` (embed `Base`): `nama_item`, `deskripsi`, `urutan`, `wajib`, `aktif`.
- **Migration** `000017_checklist_templates`: tabel baru + kolom `wajib` di `field_maintenance_checks` (default TRUE). ID di-generate di Go via hook GORM (tanpa `gen_random_uuid()`).
- **Seeder** `SeedChecklistTemplates` (models/seeder.go): mengisi 9 item default bila tabel kosong, idempotent.
- **Handler** `handlers/checklist_template.go`: `GetAllChecklistTemplates` (paginasi + filter `?aktif=`), `CreateChecklistTemplate`, `UpdateChecklistTemplate`, `DeleteChecklistTemplate` (soft delete via `aktif=false`), `ReorderChecklistTemplates` (batch update urutan dalam transaksi). Semua operasi tulis mencatat audit.
- **Route** `routes/checklist_template.go`: `/checklist-templates` (statis `/reorder` didaftarkan sebelum `/:id`); CRUD khusus admin, daftar untuk semua role login.
- **Integrasi** `CreateFieldMaintenance`: item `FieldMaintenanceCheck` kini menyimpan `Wajib` (dari request bila ada, default true). Kontrak API `Checks` dinamis tetap dipertahankan.
- **Unit test** `handlers/checklist_template_test.go`: CRUD, reorder, validasi, dan guard admin (401/403).
- **main.go**: registrasi `routes.SetupChecklistTemplateRoutes(api)` dan panggilan `models.SeedChecklistTemplates(config.DB)`.

timestamp: 2026-08-12 10:30 WITA
kategori: Sedang, Backend
file yang diubah:
- backend/migrations/000017_checklist_templates.up.sql (baru)
- backend/migrations/000017_checklist_templates.down.sql (baru)
- backend/models/checklist_template.go (baru)
- backend/models/seeder.go
- backend/models/field_maintenance.go
- backend/handlers/checklist_template.go (baru)
- backend/handlers/checklist_template_test.go (baru)
- backend/handlers/field_maintenance.go
- backend/routes/checklist_template.go (baru)
- backend/main.go

## [0.55.0] - 2026-08-10

### Fitur: Infrastruktur & Integrasi (Sprint 34 - P10, P14, P16)

perubahan yang dilakukan: **Menambahkan retensi log webhook delivery, endpoint rotasi QR Code CCTV, dan endpoint regenerasi API key.**

before:
- Log `webhook_deliveries` tumbuh tanpa batas (tidak ada retensi).
- QR Code CCTV tidak bisa dirotasi (QrPayload tetap sama selamanya).
- API key tidak bisa diregenerasi tanpa menghapus & membuat baru (mengganggu integrasi).

after:
- **P10 — Retensi log webhook (Task 13.10)**:
  - Migration `000016_webhook_retention` + kolom `max_log_retention_days` (default 30) di `WebhookEndpoint`.
  - `StartWebhookLogCleanup` (startup + tiap 7 hari) menghapus log delivery lebih tua dari retensi per endpoint.
  - Endpoint admin `DELETE /api/webhooks/:id/deliveries?older_than=<hari>` untuk trigger manual.
  - `GetAllWebhookEndpoints` mengembalikan `delivery_counts` (jumlah log per endpoint); `GetWebhookEndpointByID` mengembalikan `delivery_count`.
  - `CreateWebhookRequest`/`UpdateWebhookRequest` mendukung `max_log_retention_days`.
  - Unit test: `TestCleanupWebhookDeliveries`, `TestAdminCleanupWebhookDeliveries(_NotFound)`.
- **P14 — Rotasi QR Code (Task 13.14)**:
  - Handler `RotateQRCode` (`POST /api/cctv/:id/rotate-qr`, admin-only): generate UUID baru → enkripsi AES-256 sebagai `QrPayload` baru → simpan → audit + broadcast `CCTV_UPDATED`.
  - Route di `routes/cctv.go` (sebelum `/:id`); unit test `TestRotateQRCode_Success(_NotFound)`.
- **P16 — Regenerasi API key (Task 13.16)**:
  - Handler `RegenerateAPIKey` (`POST /api/api-keys/:id/regenerate`, admin-only): generate key mentah baru → bcrypt hash → update `key_hash`/`key_prefix` → reset `last_used_at` → audit + broadcast `API_KEY_REGENERATED` → return `{ api_key }` (tampil 1×).
  - Route di `routes/api_key.go`; unit test `TestRegenerateAPIKey_Success(_NotFound)`.

timestamp: 2026-08-10 18:30 WITA
kategori: Sedang, Backend
file yang diubah:
- backend/migrations/000016_webhook_retention.up.sql (baru)
- backend/migrations/000016_webhook_retention.down.sql (baru)
- backend/models/webhook.go
- backend/handlers/webhook.go
- backend/handlers/webhook_test.go
- backend/handlers/api_key.go
- backend/routes/api_key.go
- backend/handlers/cctv.go
- backend/routes/cctv.go
- backend/handlers/infra_integration_test.go (baru)
- backend/main.go
- backend/docs/* (swag init)

## [0.54.0] - 2026-08-10

### Fitur: Kecerdasan Data & Eskalasi (Sprint 33 - P8, P9, P11)

perubahan yang dilakukan: **Menambahkan histori skor kesehatan, catch-up digest, dan eskalasi overdue bertingkat.**

before:
- Skor kesehatan CCTV hanya disimpan di kolom `health_score` (tanpa histori/tren).
- `RunDailyDigestGenerator` menunggu 06:00 WITA; bila server restart setelahnya, digest hari ini terlewat.
- `RunOverdueChecker` hanya mengirim notifikasi overdue sekali (tanpa eskalasi bertingkat).

after:
- **P8 — Histori skor kesehatan (Task 13.8)**:
  - Migration `000014_cctv_health_histories` + model `CCTVHealthHistory`.
  - Insert histori saat `recalculateCCTVHealthScoreAt`; endpoint `GET /api/cctv/:id/health-history?range=7d|30d|90d`.
  - Route + unit test `TestGetCCTVHealthHistory`.
- **P9 — Digest catch-up (Task 13.9)**:
  - `checkAndGenerateMissedDigest` dipanggil di awal `RunDailyDigestGenerator` (generate digest hari ini bila belum ada).
  - Unit test `TestDailyDigestCatchUp_AlreadyExists`.
- **P11 — Eskalasi overdue bertingkat (Task 13.11)**:
  - Migration `000015_overdue_level` + kolom `overdue_level` di `MaintenanceLog`.
  - Level 0/1(>24h)/2(>48h)/3(>72h); eskalasi hanya saat level naik; level 3 → notifikasi kritis `OVERDUE_CRITICAL` + email + broadcast; level 2 → `OVERDUE_HIGH`.
  - Reset `overdue_level` saat selesai/di-reschedule.
  - Unit test `TestOverdueEscalationLevels`.

timestamp: 2026-08-10 16:45 WITA
kategori: Besar, Backend
file yang diubah:
- backend/migrations/000014_cctv_health_histories.up.sql (baru)
- backend/migrations/000014_cctv_health_histories.down.sql (baru)
- backend/migrations/000015_overdue_level.up.sql (baru)
- backend/migrations/000015_overdue_level.down.sql (baru)
- backend/models/cctv_health_history.go (baru)
- backend/models/notification.go
- backend/models/setup.go
- backend/handlers/health_score.go
- backend/handlers/health_score_test.go
- backend/handlers/digest_generator.go
- backend/handlers/digest_generator_test.go
- backend/handlers/overdue_checker.go
- backend/handlers/overdue_checker_test.go
- backend/handlers/maintenance.go
- backend/routes/cctv.go

## [0.53.1] - 2026-08-10

### Fix: Race/Panic pada middleware API Key (last_used_at)

perubahan yang dilakukan: **Update `last_used_at` API key diubah dari goroutine async menjadi sinkron, menghilangkan race/panic saat `config.DB` berubah.**

before: `last_used_at` diperbarui dalam goroutine async yang memakai `config.DB` global setelah handler selesai; di lingkungan uji/CI `config.DB` bisa menjadi nil (cleanup `defer`) sehingga terjadi nil pointer dereference (flaky).

after:
- `backend/middleware/apikey.go`: update `last_used_at` dijalankan sinkron + guard `config.DB != nil`; hapus goroutine & import `uuid` yang tidak terpakai.
- `backend/middleware/apikey_test.go`: perbarui komentar (sinkron).

timestamp: 2026-08-10 14:45 WITA
kategori: Kecil, Backend
file yang diubah:
- backend/middleware/apikey.go
- backend/middleware/apikey_test.go

## [0.53.0] - 2026-08-10

### Fitur: Notifikasi & Preferensi Email (Sprint 32 - P6, P7a)

perubahan yang dilakukan: **Menambahkan cleanup otomatis notifikasi lama dan preferensi email granular per-tipe.**

before:
- Tabel `notifications` tumbuh tanpa batas (tidak ada cleanup otomatis).
- `EmailNotifications` satu toggle untuk semua jenis email; user tidak bisa pilih per-tipe (digest/overdue/stock).

after:
- **P6 — Cleanup notifikasi (Task 13.6)**:
  - `CleanupOldNotifications(db)` — hapus yang dibaca >30 hari & belum dibaca >90 hari (hard delete via Unscoped).
  - `StartNotificationCleanup(db)` — goroutine saat startup lalu tiap 7 hari.
  - Endpoint admin `DELETE /api/notifications/cleanup?older_than=30` → `{deleted_read, deleted_unread, total_deleted}`.
  - `GetNotifications` respons tambah field `total_count`.
  - Route `DELETE /cleanup` (admin-only) + registrasi di main.go.
  - Unit test `TestCleanupOldNotifications`, `TestAdminCleanupEndpoint`.
- **P7a — Preferensi email granular (Task 13.7)**:
  - Migration `000013_email_prefs_granular` (up/down): kolom `email_daily_digest`, `email_overdue`, `email_stock_low` (default true, sync dengan email_notifications).
  - Model `User` tambah 3 field; `EmailNotifications` tetap master toggle.
  - `PUT /api/profile/email-preferences` terima body granular; sinkronisasi master (false jika semua nonaktif, true jika minimal satu aktif).
  - Gate pengiriman email: digest → `email_daily_digest`, overdue → `email_overdue`, stock low → `email_stock_low`.
  - Unit test `TestUpdateEmailPreferences_Granular`, `TestUpdateEmailPreferences_AllOff`.

timestamp: 2026-08-10 14:30 WITA
kategori: Sedang, Backend
file yang diubah:
- backend/migrations/000013_email_prefs_granular.up.sql (baru)
- backend/migrations/000013_email_prefs_granular.down.sql (baru)
- backend/models/setup.go
- backend/handlers/notification.go
- backend/handlers/notification_test.go
- backend/handlers/email.go
- backend/handlers/email_test.go
- backend/handlers/auth.go
- backend/routes/notification.go
- backend/main.go

## [0.52.0] - 2026-08-10

### Fitur: Keamanan & Integritas Data (Sprint 31 - P1, P3, P4, P5)

perubahan yang dilakukan: **Memperkuat integritas & keamanan data CCTV: migrasi koordinat ke float, hardening password RTSP, validasi keunikan channel port, dan retry auto-register stream.**

before:
- Koordinat CCTV disimpan sebagai `varchar(50)` — tidak presisi & sulit dihitung.
- `rtsp_password` berpotensi bocor bila terisi (tag `omitempty`).
- `channel_port` tidak divalidasi keunikan → konflik di NVR Hikvision.
- `RegisterActiveStreams` hanya jalan sekali tanpa retry & tanpa cek kesiapan MediaMTX.
- Peta sebaran menyebar CCTV tanpa koordinat secara acak di sekitar IKN (menyesatkan).

after:
- **P1 — Koordinat float (Task 13.1)**: migrasi `000012_cctv_coords_to_float` (latitude/longitude → `double precision`, kosong → NULL); model `CCTV.Latitude/Longitude` → `*float64`; request `Create/UpdateCCTVRequest` → `*float64` + validasi rentang (lat -90..90, lng -180..180); `ImportCCTV` parse `strconv.ParseFloat` (format tidak valid → warning + dilewati); frontend konversi string→number|null saat submit.
- **P3 — Hardening RTSPPassword (Task 13.3)**: memastikan `RTSPPassword` dikosongkan pada semua titik return (Create, GetAll, GetByID, Update, Scan); test `TestRTSPPasswordNotInResponse`.
- **P4 — Validasi ChannelPort (Task 13.4)**: range 1-35 + keunikan saat create & update (exclude self); helper `validateChannelPortForCreate/ForUpdate`; test range & duplikat.
- **P5 — Retry Stream Auto-Register (Task 13.5)**: `MediaMTXClient.Ping`; `waitForMediaMTX` (2 mnt); `registerStreamWithRetry` (exponential backoff 2s/4s/8s/16s/30s); `ReRegisterFailedStreams` + scheduler `RunStreamReRegistrar` (tiap 10 mnt); test retry & give-up.

timestamp: 2026-08-10 13:30 WITA
kategori: Sedang, Backend
file yang diubah:
- backend/migrations/000012_cctv_coords_to_float.up.sql (baru)
- backend/migrations/000012_cctv_coords_to_float.down.sql (baru)
- backend/models/cctv.go
- backend/handlers/cctv.go
- backend/handlers/cctv_test.go
- backend/handlers/stream.go
- backend/handlers/stream_test.go
- backend/utils/mediamtx.go
- backend/main.go

## [0.51.0] - 2026-08-09

### Fitur: API Key & Webhook (Task 12.1 + 12.3 - Sprint 29)

perubahan yang dilakukan: **Menambahkan API key untuk integrasi eksternal dan sistem webhook untuk mengirim event ke sistem lain. Backend naik `0.50.0` → `0.51.0`.**

before: Integrasi eksternal hanya lewat sesi web (cookie JWT); belum ada kredensial API key. Belum ada cara mengirim notifikasi event ke sistem eksternal (webhook).

after:
- API Key (Task 12.1):
  - Model `APIKey` (`models/api_key.go`) — `Name`, `KeyHash` (bcrypt), `KeyPrefix`, `Scopes` (jsonb: `cctv:read`/`maintenance:read`/`health:read`), `ExpiresAt`, `CreatedBy`, `LastUsedAt`, `RevokedAt`.
  - Middleware `middleware/apikey.go` — `APIKeyAuth(scope)` (strict via `X-API-Key`) & `APIKeyOrProtected(scope)` (API key ATAU JWT untuk `GET /api/cctv`); rate limit per key 100 req/menit via Redis counter (fallback in-memory); cek revoked/expired/scope; update `last_used_at`.
  - Handler `handlers/api_key.go` — `POST /api/api-keys` (generate random key, tampil 1×), `GET /api/api-keys`, `POST /api/api-keys/:id/revoke` (admin-only).
  - `GET /api/cctv` dapat diakses via API key scope `cctv:read`; `GET /api/health` tetap publik (juga via API key `health:read`).
- Webhook (Task 12.3):
  - Model `WebhookEndpoint` & `WebhookDelivery` (`models/webhook.go`); event: `cctv.status_changed`, `maintenance.completed`, `maintenance.overdue`, `health_score.changed`, `stream.down`.
  - Migrasi `000010_api_keys_webhooks` (up/down).
  - Util `handlers/webhook.go` — antrean event (channel) + worker `StartWebhookWorker` (dipanggil di main.go); `DispatchWebhook` decoupled (tidak query DB di handler); `deliverWebhook` POST dengan header `X-Signature-256` (HMAC-SHA256), retry 3× exponential backoff (1s/5s/25s).
  - Hook di handler existing: `cctv.status_changed` (cctv_sync.go), `maintenance.completed` (maintenance.go), `maintenance.overdue` (overdue_checker.go), `health_score.changed` (health_score.go), `stream.down` (stream.go).
  - Handler `handlers/webhook.go` — CRUD `/api/webhooks` (admin-only), `POST /api/webhooks/:id/test` (test payload), `GET /api/webhooks/:id/deliveries` (log riwayat).
- Routes `routes/api_key.go`, `routes/webhook.go` + registrasi di `main.go` (termasuk `StartWebhookWorker`).
- Swagger annotations + `swag init`.
- Unit test: `handlers/api_key_test.go`, `handlers/webhook_test.go`, `middleware/apikey_test.go`, rute baru di `routes/routes_test.go`.

timestamp: 2026-08-09 11:50 WITA
kategori: Besar, Backend
file yang diubah:
- backend/models/api_key.go (baru)
- backend/models/webhook.go (baru)
- backend/middleware/apikey.go (baru)
- backend/middleware/apikey_test.go (baru)
- backend/handlers/api_key.go (baru)
- backend/handlers/api_key_test.go (baru)
- backend/handlers/webhook.go (baru)
- backend/handlers/webhook_test.go (baru)
- backend/handlers/cctv_sync.go (hook webhook)
- backend/handlers/maintenance.go (hook webhook)
- backend/handlers/overdue_checker.go (hook webhook)
- backend/handlers/health_score.go (hook webhook + muat skor lama)
- backend/handlers/stream.go (hook webhook)
- backend/handlers/helpers_test.go (expectHealthScoreRecalculation + muat skor)
- backend/migrations/000010_api_keys_webhooks.up/down.sql (baru)
- backend/routes/api_key.go (baru)
- backend/routes/webhook.go (baru)
- backend/routes/cctv.go (API key access)
- backend/routes/routes_test.go
- backend/main.go (registrasi rute + StartWebhookWorker)
- backend/docs/ (swag init)
alasan perubahan: Kebutuhan integrasi & skalabilitas (ROADMAP Task 12.1 + 12.3) — menyediakan kredensial API key bagi sistem eksternal dan mengirim event perubahan ke endpoint webhook.

## [0.50.0] - 2026-08-09

### Fitur: Zona & SLA (Task 11.4 + 11.5 - Sprint 28)

perubahan yang dilakukan: **Menambahkan modul zona operasional (wilayah CCTV & teknisi) dan kebijakan SLA beserta dashboard kepatuhan. Backend naik `0.49.0` → `0.50.0`.**

before: CCTV tidak memiliki pengelompokan wilayah; teknisi tidak ditautkan ke zona; belum ada target waktu layanan (SLA) maupun dashboard kepatuhan. Seluruh daftar CCTV belum mendukung filter berdasarkan zona.

after:
- Zona (Task 11.4):
  - Model `Zone` (`models/zone.go`) — `Nama`, `Deskripsi`, relasi one-to-many `CCTVs` (FK `zone_id`) dan many-to-many `Users` via join table `user_zones` (model `UserZone`).
  - Migrasi `000009_zone_sla` (up/down): tabel `zones`, `user_zones`, `sla_policies`; kolom FK `zone_id` di `cctvs` (ON DELETE SET NULL) dan `sla_policy_id` di `maintenance_schedules` (ON DELETE SET NULL).
  - Handler `handlers/zone.go`: CRUD `/api/zones` (admin-only untuk tulis), `GET /api/zones/all` (dropdown tanpa paginasi), `POST /api/zones/:id/assign-cctvs` (bulk assign CCTV), `POST /api/zones/:id/assign-users` (assign teknisi, menimpa penugasan zona).
  - Daftar zona mengembalikan `cctv_count` & `user_count` per zona; detail zona memuat relasi CCTV & teknisi.
- SLA (Task 11.5):
  - Model `SLAPolicy` (`models/sla.go`) — `Nama`, `JenisTugas` (corrective/preventive/field_check), `TargetJamRespon`, `TargetJamSelesai`.
  - FK `sla_policy_id` di `MaintenanceSchedule` (`models/maintenance_schedule.go`).
  - Handler `handlers/sla.go`: CRUD `/api/sla-policies` (admin-only untuk tulis) + `GET /api/dashboard/sla` (admin) — `monthly_compliance` (bar chart), `violations` (tabel pelanggaran), `technician_ranking`.
  - Goroutine `RunSLADeadlineChecker` (startup + tiap 1 jam) memicu `BroadcastToRole("admin", "SLA_WARNING", ...)` saat tugas melewati 75% target penyelesaian.
- `GetAllCCTV` mendukung filter `?zone_id=<uuid>`.
- Routes `routes/zone.go` & `routes/sla.go` + registrasi di `main.go`.
- Swagger annotations + `swag init`.
- Unit test: `handlers/zone_test.go`, `handlers/sla_test.go`, rute baru di `routes/routes_test.go`.

timestamp: 2026-08-09 09:40 WITA
kategori: Besar, Backend
file yang diubah:
- backend/models/zone.go (baru)
- backend/models/sla.go (baru)
- backend/models/cctv.go (ZoneID FK)
- backend/models/maintenance_schedule.go (SLAPolicyID FK)
- backend/migrations/000009_zone_sla.up/down.sql (baru)
- backend/handlers/zone.go (baru)
- backend/handlers/sla.go (baru)
- backend/handlers/cctv.go (filter zone_id)
- backend/routes/zone.go (baru)
- backend/routes/sla.go (baru)
- backend/routes/routes_test.go
- backend/handlers/zone_test.go (baru)
- backend/handlers/sla_test.go (baru)
- backend/main.go (registrasi rute + goroutine SLA)
- backend/docs/ (swag init)
alasan perubahan: Kebutuhan operasional lanjut (ROADMAP Task 11.4 + 11.5) — mengelompokkan CCTV & teknisi per zona dan memantau kepatuhan target waktu layanan (SLA) untuk meningkatkan kualitas pemeliharaan.

## [0.49.0] - 2026-08-09

### Fitur: Notifikasi Email SMTP & Report Builder (Task 11.2 + 11.3 - Sprint 27)

perubahan yang dilakukan: **Menambahkan notifikasi email berbasis SMTP dan report builder laporan kustom. Backend naik `0.48.0` → `0.49.0`.**

before: Notifikasi hanya in-app + push (WebSocket/browser); laporan hanya PDF pemeliharaan statis. Belum ada email otomatis maupun laporan kustom yang dapat dikonfigurasi admin.

after:
- Email (Task 11.3):
  - `utils/email.go`: `SendEmail` via `net/smtp` + TLS (STARTTLS/implicit), `InitEmailConfig`/`EmailEnabled`, `RenderEmailTemplate` (html/template, multi-path), `EncodeHeader` RFC 2047.
  - Template `backend/templates/`: `daily_summary.html`, `overdue_escalation.html`, `stock_low.html`, `stream_down.html` (responsif, anti XSS).
  - Model `User.EmailNotifications` + migrasi `000007_add_email_notifications`.
  - Handler `handlers/email.go`: `UpdateEmailPreferences` (`PUT /api/profile/email-preferences`), `SendDailySummaryEmails`, `SendOverdueEmail`, `SendStockLowEmail`, `SendCCTVStreamDownEmail`.
  - Trigger: digest 06:00 WITA (ringkasan harian), overdue checker (eskalasi), `STOCK_LOW` (stok rendah), stream down > 15 menit → email CCTV offline ke seluruh admin yang opt-in.
  - Env SMTP di `.env.example` & `backend/.env.docker.example` (contoh Hostinger `noreply@nexaspace.site`).
- Report Builder (Task 11.2):
  - Model `ReportTemplate` + migrasi `000008_report_templates`.
  - Handler `handlers/report_builder.go`: `GenerateReport` (`POST /api/reports/generate`, entitas cctv/maintenance_log/field_maintenance/spare_part, format csv/xlsx/pdf), `ListReportTemplates`/`SaveReportTemplate`/`DeleteReportTemplate`.
  - Memakai `github.com/xuri/excelize/v2` (Excel) & `github.com/go-pdf/fpdf` (PDF, fork gofpdf yang sudah ada).
- Routes `routes/profile.go` (`/profile/email-preferences`) & `routes/report.go` (admin-only) + registrasi di `main.go`; `InitEmailConfig` di startup.
- Swagger annotations + `swag init`.
- Unit test: `handlers/email_test.go`, `handlers/report_builder_test.go`, `utils/email_test.go`.

timestamp: 2026-08-09 02:40 WITA
kategori: Besar, Backend
file yang diubah:
- backend/utils/email.go (baru)
- backend/utils/email_test.go (baru)
- backend/templates/daily_summary.html (baru)
- backend/templates/overdue_escalation.html (baru)
- backend/templates/stock_low.html (baru)
- backend/models/setup.go (User.EmailNotifications)
- backend/models/report.go (baru)
- backend/migrations/000007_add_email_notifications.up/down.sql (baru)
- backend/migrations/000008_report_templates.up/down.sql (baru)
- backend/handlers/email.go (baru)
- backend/handlers/email_test.go (baru)
- backend/handlers/report_builder.go (baru)
- backend/handlers/report_builder_test.go (baru)
- backend/handlers/overdue_checker.go
- backend/handlers/digest_generator.go
- backend/handlers/spare_part.go
- backend/handlers/auth.go
- backend/routes/profile.go (baru)
- backend/routes/report.go (baru)
- backend/main.go
- backend/docs/* (regenerasi swag)
- .env.example, backend/.env.docker.example
- backend/changelog.md
alasan perubahan: Kebutuhan operasional lanjut (ROADMAP Task 11.2 & 11.3) — email otomatis untuk proaktifitas (ringkasan, eskalasi, stok rendah) dan laporan kustom yang dapat dikonfigurasi admin untuk pengambilan keputusan.

## [0.48.0] - 2026-08-09

### Fitur: Inventaris Suku Cadang (Task 11.1 - Sprint 26)

perubahan yang dilakukan: **Menambahkan modul inventaris suku cadang yang terintegrasi dengan alur pemeliharaan existing. Backend naik `0.47.1` → `0.48.0`.**

before: Teknisi tidak memiliki cara mencatat suku cadang yang dipakai saat perbaikan CCTV, dan admin tidak dapat memantau stok komponen (kabel, konektor, power supply, dll.) karena belum ada modul inventaris.

after:
- Model `models/spare_part.go`: `SparePart` (nama, sku unik, kategori, stok, minimum_order, satuan, harga_satuan, keterangan) + `SparePartUsage` (maintenance_log_id, spare_part_id, jumlah, catatan) dengan relasi GORM.
- Migrasi `000006_spare_parts.up.sql` / `.down.sql`: tabel `spare_parts` & `spare_part_usages` + index + foreign keys (idempoten).
- Handler `handlers/spare_part.go`:
  - `GetAllSpareParts` (paginasi, filter kategori, pencarian nama/SKU).
  - `GetSparePartByID`, `CreateSparePart` (admin, cek SKU duplikat incl. soft-delete), `UpdateSparePart` (admin), `DeleteSparePart` (admin, soft delete).
  - `RecordSparePartUsage` (`POST /api/maintenance-logs/:id/spare-parts`): hanya log `completed`, mengurangi stok otomatis dalam transaksi, broadcast `STOCK_LOW` ke admin bila stok < minimum_order.
  - `GetSparePartUsageByLog` (`GET /api/maintenance-logs/:id/spare-parts`).
  - `GetLowStockSpareParts` (`GET /api/spare-parts/low-stock`, filter `stok < minimum_order`, urut stok terkecil) untuk widget dashboard.
- Route `routes/spare_part.go`: `/spare-parts` (protected, CRUD admin-only; `/low-stock` statis sebelum `/:id` dinamis) + `/maintenance-logs/:id/spare-parts`; registrasi `routes.SetupSparePartRoutes(api)` di `main.go`.
- Swagger annotations di semua handler + `swag init`.
- Unit test `handlers/spare_part_test.go`: CRUD, SKU duplikat, stok berkurang (transaksi + `ExpectBegin`/`ExpectCommit`), stok tidak cukup (400, `ExpectRollback`), log bukan completed (409), usage by log, low-stock.

timestamp: 2026-08-09 00:50 WITA
kategori: Besar, Backend
file yang diubah:
- backend/models/spare_part.go (baru)
- backend/migrations/000006_spare_parts.up.sql (baru)
- backend/migrations/000006_spare_parts.down.sql (baru)
- backend/handlers/spare_part.go (baru)
- backend/handlers/spare_part_test.go (baru)
- backend/routes/spare_part.go (baru)
- backend/main.go
- backend/docs/* (regenerasi swag)
- backend/changelog.md
alasan perubahan: Kebutuhan vital operasional (keputusan prioritas ROADMAP Task 11.1) agar teknisi dapat mencatat penggunaan suku cadang dan admin dapat memantau stok. Stok berkurang otomatis pada pencatatan penggunaan; peringatan `STOCK_LOW` real-time memungkinkan restock proaktif.

## [0.47.1] - 2026-08-07

### Perbaikan: RunMigrations menutup koneksi DB global (aplikasi crash di Docker)

perubahan yang dilakukan: **Menghapus pemanggilan `m.Close()` pada `RunMigrations` di `backend/config/migrate.go` agar koneksi database global (`config.DB`) tidak ikut ditutup setelah migrasi selesai. Ini memperbaiki crash "sql: database is closed" saat aplikasi berjalan di container Docker.**

before: `RunMigrations` memakai `defer func() { _, _ = m.Close() }()`. `migrate.Migrate.Close()` pada driver pgx v5 memanggil `p.db.Close()` yang menutup `*sql.DB` global yang di-share dengan `config.DB`, sehingga setelah migrasi berhasil, seluruh operasi DB berikutnya (migrasi data legacy, seeder admin, handler) gagal dengan `sql: database is closed` dan aplikasi tidak bisa startup.

after:
- `backend/config/migrate.go`: `RunMigrations` TIDAK lagi memanggil `m.Close()` (instance dibiarkan dibersihkan GC). Komentar dijelaskan alasannya.
- Efek: aplikasi berhasil menjalankan migrasi, seeder admin, dan melayani request setelah startup di Docker.

timestamp: 2026-08-07 22:10 WITA
kategori: Kecil, Backend
file yang diubah:
- backend/config/migrate.go
- backend/changelog.md
alasan perubahan: Aplikasi tidak dapat berjalan di container Docker karena `RunMigrations` menutup koneksi DB global (golang-migrate `Close()` → `p.db.Close()` pada driver pgx v5 yang memakai `*sql.DB` yang sama dengan `config.DB`). Menghapus `m.Close()` pada `RunMigrations` memperbaiki crash "sql: database is closed" sehingga aplikasi dapat startup dan melayani request.

## [0.47.0] - 2026-08-07

### Fitur: Deteksi Status Stream & Stream Health (Task 10.4 — Fase 10)

perubahan yang dilakukan: **Menambahkan monitoring status stream & endpoint stream health untuk Fase 10 (Kecerdasan Visual & Snapshot). Backend naik `0.46.0` → `0.47.0`.**

before: Stream CCTV sudah dapat di-start/stop via MediaMTX (Fase 9) dan snapshot/timeline tersedia (Task 10.1+10.2), namun belum ada pemantauan otomatis terhadap status ready/down per path maupun cara admin melihat kondisi health tiap stream.

after:
- Goroutine monitoring (tiap 5 menit) query MediaMTX `ListPaths()` — men-track ready state per path stream CCTV.
- Stream down **> 15 menit** → broadcast notifikasi `STREAM_DOWN` ke admin + kurangi `health_score` CCTV sebesar 10.
- Stream pulih → broadcast notifikasi `STREAM_RESTORED`.
- Endpoint `GET /api/streams/health` (admin-only) — daftar stream + status `ready`/`down` + `down_duration`.
- Unit test `handlers/stream_health_test.go` — `GetStreamHealth` (tanpa stream, unauthorized 401/403), `RunStreamHealthMonitor` (no MediaMTX, down > 15 menit → `STREAM_DOWN` + health -10, pulih → `STREAM_RESTORED`).
- Swagger annotations + `swag init`.

timestamp: 2026-08-07 WITA
kategori: Besar, Backend
file yang diubah:
- backend/handlers/stream.go
- backend/handlers/stream_health_test.go
- backend/routes/stream.go
- backend/main.go
- backend/docs/docs.go
- backend/docs/swagger.json
- backend/docs/swagger.yaml
- backend/changelog.md
alasan perubahan: Memantau kesehatan stream CCTV secara otomatis dan menyediakan endpoint health (ready/down + down_duration) bagi admin — Task 10.4 (Deteksi Status Stream), Fase 10 (Kecerdasan Visual & Snapshot).

## [0.46.0] - 2026-08-07

### Fitur: Snapshot Otomatis & Timeline Event Visual (Task 10.1 + 10.2 — Fase 10)

perubahan yang dilakukan: **Menambahkan snapshot otomatis dan timeline event visual CCTV untuk Fase 10 (Kecerdasan Visual & Snapshot). Backend naik `0.45.0` → `0.46.0`.**

before: Fase 9 streaming sudah menampilkan video live, namun belum ada penyimpanan visual (snapshot frame) maupun riwayat event per CCTV; `utils/mediamtx.go` sudah memiliki `GetSnapshot` namun belum dipakai untuk snapshot terjadwal/endpoint.

after:
- Model `CCTVSnapshot` (cctv_id, timestamp, event_type, image_path, notes) + migrasi `000005_cctv_snapshots`.
- Goroutine snapshot berkala (tiap 15 menit) mengambil frame dari stream aktif via `utils/mediamtx.go` `GetSnapshot` (endpoint MediaMTX `/v3/paths/get/{path}/snapshot`) → simpan JPEG di `uploads/snapshots/{cctv_id}/`; rotasi maksimal 10 snapshot per CCTV.
- Endpoint `GET /api/cctv/:id/snapshot` (mengembalikan JPEG terbaru; 404 bila belum ada snapshot).
- Endpoint `GET /api/cctv/:id/timeline` (paginasi, event kronologis terbaru dahulu).
- Event hook `status_changed`, `maintenance_completed`, `field_check_attention`, `periodic_snapshot` (capture otomatis saat event penting).
- Field computed `snapshot_url` (`gorm:"-"`, `json:"snapshot_url,omitempty"`) pada respons `GetAllCCTV`/`GetCCTVByID` — menunjuk ke endpoint snapshot.
- Swagger annotations + `swag init`; unit test `handlers/snapshot_test.go` (GetCCTVSnapshot 404/success, GetCCTVTimeline 404/success, rotasi).

timestamp: 2026-08-07 WITA
kategori: Besar, Backend
file yang diubah:
- backend/models/cctv_snapshot.go
- backend/migrations/000005_cctv_snapshots.up.sql
- backend/migrations/000005_cctv_snapshots.down.sql
- backend/handlers/snapshot.go
- backend/handlers/snapshot_test.go
- backend/routes/cctv.go
- backend/utils/mediamtx.go
- backend/models/cctv.go
- backend/main.go
- backend/docs/docs.go
- backend/docs/swagger.json
- backend/docs/swagger.yaml
- backend/changelog.md
alasan perubahan: Memperkaya data visual CCTV dengan snapshot otomatis (thumbnail untuk tabel master & peta) dan timeline event visual (riwayat status/pemeliharaan/field-check) — membuka Fase 10 (Kecerdasan Visual & Snapshot), Task 10.1 + 10.2.

## [0.45.0] - 2026-08-07

### Fitur: Fase 9 Streaming CCTV — Selesai (Task 9.1–9.6)

perubahan yang dilakukan: **Melengkapi & menutup seluruh backend streaming CCTV real-time (Fase 9). Menyimpulkan hasil dari Sprint 20–23: model & migrasi streaming CCTV, util RTSP, service MediaMTX, client MediaMTX, serta handler & route stream admin-only — sehingga peta sebaran CCTV dapat menampilkan video live via WHEP/WebRTC. Backend naik `0.44.0` → `0.45.0` sebagai penutup Fase 9.**

before: Backend streaming (Sprint 20–21) sudah menyediakan model, migrasi `000004`, `utils/rtsp.go`, `utils/mediamtx.go`, serta `handlers/stream.go`/`routes/stream.go`; namun Fase 9 belum ditandai selesai dan dokumentasi serta versi rilis belum menutup milestone.

after:
- Fase 9 ditandai **6/6 ✅ Selesai** (Task 9.1–9.6) di ROADMAP; seluruh endpoint streaming `/api/streams/*` admin-only (`Protected()` + `RequireRole("admin")`).
- `models/cctv.go` + `migrations/000004_add_cctv_streaming`: `channel_port`, `rtsp_password` (AES-256 encrypted, tidak dikembalikan), `stream_enabled`.
- `utils/rtsp.go` (BuildRTSPURL/BuildStreamPath/GetRTSPPort), `utils/mediamtx.go` (MediaMTXClient AddPath/RemovePath/GetPath/ListPaths), `handlers/stream.go` (StartStream/StopStream/StartAllStreams/StopAllStreams/GetStreamStatus/GetStreamURL), `routes/stream.go` (grup `/streams`), registrasi `main.go` + goroutine auto-start.
- Swagger annotations + `swag init`; unit test `utils/rtsp_test.go`, `utils/mediamtx_test.go`, `handlers/stream_test.go`, `routes/stream_test.go`.

timestamp: 2026-08-07 WITA
kategori: Besar, Backend
file yang diubah:
- backend/models/cctv.go
- backend/migrations/000004_add_cctv_streaming.up.sql
- backend/migrations/000004_add_cctv_streaming.down.sql
- backend/utils/rtsp.go
- backend/utils/mediamtx.go
- backend/handlers/stream.go
- backend/routes/stream.go
- backend/main.go
- backend/utils/rtsp_test.go
- backend/utils/mediamtx_test.go
- backend/handlers/stream_test.go
- backend/routes/stream_test.go
- backend/docs/docs.go
- backend/docs/swagger.json
- backend/docs/swagger.yaml
- backend/changelog.md
alasan perubahan: Menutup Fase 9 (Streaming CCTV Real-Time) — menyatukan fondasi model/migrasi/infra (Task 9.1/9.2), API & client MediaMTX (Task 9.3) menjadi rilis backend final Fase 9; seluruh endpoint streaming bersifat admin-only sesuai keputusan arsitektur streaming dan `rtsp_password` terenkripsi tidak pernah diekspos.

## [0.44.0] - 2026-08-07

### Fitur: Backend API & MediaMTX Client (Task 9.3)

perubahan yang dilakukan: **Menambahkan client REST MediaMTX dan seluruh handler/rute streaming CCTV — `utils/mediamtx.go` (struct `MediaMTXClient` dengan `AddPath`, `RemovePath`, `GetPath`, `ListPaths` ke `http://mediamtx:9997/v3/`); `handlers/stream.go` (`StartStream` — decrypt `rtsp_password` → build URL RTSP → register path di MediaMTX → return WHEP URL; `StopStream`; `StartAllStreams`; `StopAllStreams`; `GetStreamStatus` — gabung data DB + info path MediaMTX; `GetStreamURL` — return WHEP + HLS); `routes/stream.go` (grup `/streams` admin-only); registrasi di `main.go` + goroutine auto-start stream aktif saat boot; serta unit test `utils/mediamtx_test.go` & `handlers/stream_test.go` dan Swagger annotations + `swag init`. Versi backend naik `0.43.0` → `0.44.0`.**

before: MediaMTX sudah tersedia sebagai service (Task 9.2) dan model CCTV menyimpan konfigurasi streaming (Task 9.1), tetapi belum ada cara untuk mulai/menghentikan stream, mengambil status/URL stream, maupun mengelola path RTSP di MediaMTX melalui API — peta sebaran masih view-only dan belum bisa menampilkan video live.

after:
- `utils/mediamtx.go` (baru): struct `MediaMTXClient` — HTTP client ke base URL `MEDIAMTX_API_URL` (`http://mediamtx:9997/v3/`); method `AddPath` (register path RTSP pull → WHEP/HLS), `RemovePath` (hapus path), `GetPath` (detail satu path), `ListPaths` (daftar path); parsing respons JSON MediaMTX; handle status non-2xx & error.
- `handlers/stream.go` (baru): `StartStream` (decrypt `rtsp_password` via `utils.DecryptAES256` → `utils.BuildRTSPURL` → `AddPath` di MediaMTX → return `whep_url`; validasi `stream_enabled=false` → 400, tanpa `channel_port` → 400, CCTV tidak ditemukan → 404), `StopStream` (`RemovePath` → path dihapus), `StartAllStreams`/`StopAllStreams` (loop seluruh CCTV streaming), `GetStreamStatus` (gabung data DB CCTV + info path MediaMTX — `stream_active`/`stream_ready`/`viewers`), `GetStreamURL` (return `whep_url` + `hls_url`). Swagger annotations.
- `routes/stream.go` (baru): grup `/streams` dengan `Protected()` + `RequireRole("admin")` — `GET /status`, `POST /start-all`, `POST /stop-all`, `POST /:id/start`, `POST /:id/stop`, `GET /:id/url`; rute statis didaftarkan sebelum dinamis.
- `main.go`: panggil `routes.SetupStreamRoutes(api)`; jalankan goroutine `RegisterActiveStreams`/auto-start untuk stream yang `stream_enabled` saat boot.
- Test: `utils/mediamtx_test.go` (baru) — `AddPath`/`RemovePath`/`GetPath`/`ListPaths` sukses + error/edge cases via `httptest` mock (status 404, non-2xx, body tidak valid); `handlers/stream_test.go` (baru) — `StartStream` sukses/404/disabled/no-channel-port, `StopStream`, `GetStreamStatus`, `GetStreamURL`, enforcement `RequireRole("admin")` (teknisi → 403).
- Swagger: `swag init` (endpoint baru `/streams/*`).

timestamp: 2026-08-07 WITA
kategori: Besar, Backend
file yang diubah:
- backend/utils/mediamtx.go
- backend/utils/mediamtx_test.go
- backend/handlers/stream.go
- backend/handlers/stream_test.go
- backend/routes/stream.go
- backend/main.go
- backend/docs/docs.go
- backend/docs/swagger.json
- backend/docs/swagger.yaml
- backend/changelog.md
alasan perubahan: Task 9.3 ROADMAP (Fase 9 — Sprint 21) — menghubungkan konfigurasi streaming CCTV (Task 9.1) dan infrastruktur MediaMTX (Task 9.2) menjadi API backend yang bisa memulai/menghentikan stream dan mengembalikan URL WHEP/HLS untuk dipakai frontend (Sprint 22); seluruh endpoint admin-only sesuai keputusan arsitektur streaming; `rtsp_password` tidak pernah diekspos.

## [0.43.0] - 2026-08-07

### Fitur: Model & Migrasi Streaming CCTV (Task 9.1)

perubahan yang dilakukan: **Menambahkan fondasi data streaming CCTV — field `channel_port` (int), `rtsp_password` (text, AES-256 encrypted), `stream_enabled` (bool, default false) pada model `CCTV` (migrasi `000004_add_cctv_streaming`); util `utils/rtsp.go` (tipe `StreamType` `MainStream=1`/`SubStream=2`, fungsi `BuildRTSPURL`, `BuildStreamPath`, `GetRTSPPort`) untuk membangun URL RTSP Hikvision ISAPI; update `CreateCCTVRequest`/`UpdateCCTVRequest`; `rtsp_password` di-exclude dari response JSON; konfigurasi NVR global di `.env`; serta unit test `utils/rtsp_test.go`.**

before: Model `CCTV` belum memiliki data koneksi streaming (channel port, password RTSP, status stream) sehingga peta sebaran hanya view-only; URL RTSP dibangun manual/inline tanpa helper; tidak ada konfigurasi terpusat untuk host/port/user NVR maupun MediaMTX.

after:
- `models/cctv.go`: tambah `ChannelPort int` (`json:"channel_port"`), `RTSPPassword string` (text, `json:"-"` — di-exclude dari response; AES-256 encrypted), `StreamEnabled bool` (`json:"stream_enabled"`, default `false`).
- `migrations/000004_add_cctv_streaming.up.sql`/`.down.sql`: `ALTER TABLE cctvs ADD/DROP COLUMN channel_port integer NOT NULL DEFAULT 1, rtsp_password text NOT NULL DEFAULT '', stream_enabled boolean NOT NULL DEFAULT false` (idempoten).
- `utils/rtsp.go` (baru): tipe `StreamType` (`MainStream=1`/`SubStream=2`); `BuildRTSPURL(host, port, user, pass, channelPort, streamType)` — `rtsp://user:pass@host:port/Streaming/channels/{ChannelID}` dengan `ChannelID = channelPort × 100 + streamType`; `BuildStreamPath(channelPort, streamType)`; `GetRTSPPort(streamType)` (default ke sub stream bila tidak dikenal). Edge cases channel ≤ 0 ditangani.
- `handlers/cctv.go`: `CreateCCTVRequest`/`UpdateCCTVRequest` tambah `channel_port` (validasi 1–35), `rtsp_password` (di-encrypt via `utils.EncryptAES256` saat simpan), `stream_enabled`; `rtsp_password` tidak pernah dikembalikan ke response (JSON tag `-`).
- `.env.example` & `backend/.env.docker.example`: `RTSP_NVR_HOST`, `RTSP_NVR_PORT`, `RTSP_NVR_USER`, `MEDIAMTX_API_URL`.
- Test: `utils/rtsp_test.go` (baru) — `BuildRTSPURL` sub/main stream, `BuildStreamPath`, `GetRTSPPort` default, edge cases channel 0/negatif; `handlers/cctv_test.go` diperluas untuk field streaming baru.

timestamp: 2026-08-07 WITA
kategori: Sedang, Backend
file yang diubah:
- backend/models/cctv.go
- backend/migrations/000004_add_cctv_streaming.up.sql
- backend/migrations/000004_add_cctv_streaming.down.sql
- backend/utils/rtsp.go
- backend/utils/rtsp_test.go
- backend/handlers/cctv.go
- backend/handlers/cctv_test.go
- backend/changelog.md
- .env.example
- backend/.env.docker.example
alasan perubahan: Task 9.1 ROADMAP (Fase 9 — Sprint 20) — membangun fondasi streaming CCTV real-time: model menyimpan data koneksi streaming, util RTSP membangun URL Hikvision ISAPI secara konsisten, password disimpan terenkripsi AES-256 dan di-exclude dari response, serta konfigurasi NVR dipusatkan di env sebagai prasyarat sprint berikutnya (MediaMTX client & stream handler).

## [0.42.0] - 2026-08-07

### Fitur: Geotag Pengecekan Lapangan (Task 8.4)

perubahan yang dilakukan: **Menambahkan pencatatan koordinat GPS (geotag) pada pengecekan lapangan — kolom `check_latitude`/`check_longitude` pada `FieldMaintenance` (migrasi 000003), `CreateFieldMaintenance` menerima & memvalidasi koordinat opsional, dan respons list/detail menyertakan field geotag.**

before: Catatan pengecekan lapangan tidak menyimpan posisi GPS teknisi, sehingga admin tidak bisa memverifikasi bahwa teknisi benar-benar berada di lokasi CCTV saat melakukan pengecekan.

after:
- `models/field_maintenance.go`: tambah `CheckLatitude`/`CheckLongitude` (string nullable, `json:"check_latitude,omitempty"`).
- `migrations/000003_add_field_maintenance_geotag.up.sql`/`.down.sql`: `ALTER TABLE field_maintenances ADD/DROP COLUMN check_latitude/check_longitude varchar(30) NULL`.
- `handlers/field_maintenance.go`: `CreateFieldMaintenanceRequest` + `check_latitude`/`check_longitude`; helper `validateCoordinates(lat, lng)` — keduanya wajib bersamaan, format angka valid, lat rentang -90..90, lng -180..180; bila kosong → di-skip (opsional); nilai disimpan pada record.
- Respons `GetFieldMaintenanceList` & `GetFieldMaintenanceDetail` otomatis menyertakan `check_latitude`/`check_longitude` via model.
- Test: `handlers/field_maintenance_test.go` — `TestCreateFieldMaintenance_InvalidLatitude` (400), `TestCreateFieldMaintenance_InvalidLongitude` (400), `TestCreateFieldMaintenance_WithValidGeotag` (201); update reload SELECT agar mencakup kolom baru.
- Swagger: `swag init` (request struct kini memuat field geotag).

timestamp: 2026-08-07 18:30 WITA
kategori: Kecil, Backend
file yang diubah:
- backend/models/field_maintenance.go
- backend/migrations/000003_add_field_maintenance_geotag.up.sql
- backend/migrations/000003_add_field_maintenance_geotag.down.sql
- backend/handlers/field_maintenance.go
- backend/handlers/field_maintenance_test.go
- backend/docs/docs.go
- backend/docs/swagger.json
- backend/docs/swagger.yaml
- backend/changelog.md
alasan perubahan: Task 8.4 ROADMAP (Fase 8) — menambah akuntabilitas pengecekan lapangan dengan mencatat posisi GPS teknisi (opsional), sehingga admin dapat memverifikasi teknisi berada di lokasi CCTV; persetujuan lokasi bersifat opsional sehingga tidak menghambat alur kerja.

## [0.41.1] - 2026-08-07

### Perbaikan: errcheck pada test push (CI backend)

perubahan yang dilakukan: **Mengganti `os.Setenv`/`os.Unsetenv` yang return value-nya tidak dicek dengan `t.Setenv` pada `backend/utils/push_test.go` agar linter `errcheck` lulus di CI (golangci-lint).**

before: `backend/utils/push_test.go` memakai `os.Setenv`/`os.Unsetenv` tanpa memeriksa error sehingga golangci-lint memunculkan 15 issue `errcheck` dan job backend CI gagal.

after:
- `backend/utils/push_test.go`: ganti seluruh `os.Setenv`/`os.Unsetenv` dengan `t.Setenv` (idiomatik, auto-cleanup, mem-fail test bila gagal) pada `TestSendPushToSubscriptionCallsInjectedSender`, `TestSendPushNoopWhenVapidMissing`, dan `TestSendPushSendsToAllSubscriptions`; hapus import `os` yang tak terpakai.
- Verifikasi lokal: `golangci-lint run ./...` → **0 issues**; `go build`, `go test ./... -count=1`, `go vet` → PASS.

timestamp: 2026-08-07 18:00 WITA
kategori: Kecil, Backend & CI
file yang diubah:
- backend/utils/push_test.go
- backend/changelog.md
alasan perubahan: Menjaga CI backend tetap hijau — golangci-lint (errcheck) menolak `os.Setenv`/`os.Unsetenv` yang tidak memeriksa error; `t.Setenv` adalah pola yang benar dan mem-fail test bila pengaturan env gagal.

## [0.41.0] - 2026-08-07

### Fitur: Notifikasi Push Browser (Task 8.3)

perubahan yang dilakukan: **Menambahkan notifikasi push (Web Push Protocol / VAPID) — model `PushSubscription`, endpoint `POST /api/push/subscribe` & `DELETE /api/push/unsubscribe`, util `SendPush` (webpush-go), dan integrasi ke `NotifyUser`/`NotifyUserSync` sehingga setiap notifikasi in-app juga memicu push ke subscription user (async, fallback in-app bila VAPID tidak dikonfigurasi).**

before: Notifikasi hanya in-app (bell); teknisi/admin tidak mendapat pemberitahuan OS ketika aplikasi tidak dibuka, sehingga berisiko melewatkan tugas penting.

after:
- `models/push_subscription.go` (baru): `PushSubscription` — `Base`, `UserID` (FK), `Endpoint`, `KeyP256DH` (column `key_p256dh`), `KeyAuth`; index `(user_id)`.
- `migrations/000002_push_subscriptions.up.sql`/`.down.sql`: tabel `push_subscriptions` + index + FK ke users (idempoten, blok DO $$ pg_constraint).
- `utils/push.go` (baru): `SendPush(db, userID, type, title, msg, refType, refID)` — async query subscription user lalu kirim ke semua via `webpush-go`; no-op bila VAPID kosong (fallback in-app, tidak fatal); `SendPushToSubscription` (payload JSON `PushNotificationPayload`); `SetSendNotificationFn` untuk unit test. Env `VAPID_PUBLIC_KEY`/`VAPID_PRIVATE_KEY`/`VAPID_EMAIL`.
- `handlers/push.go` (baru): `POST /push/subscribe` (upsert idempotent berdasarkan endpoint+user, restore soft-deleted), `DELETE /push/unsubscribe` (soft delete by endpoint); helper `currentUserID`.
- `routes/push.go` (baru): grup `/push` protected.
- `utils/notify.go`: `NotifyUser` & `NotifyUserSync` memanggil `SendPush` setelah notifikasi in-app berhasil.
- `main.go`: daftarkan `SetupPushRoutes`.
- `go.mod`/`go.sum`: `github.com/SherClockHolmes/webpush-go v1.4.0`.
- Test: `handlers/push_test.go` (subscribe create/upsert/validasi/unauthorized, unsubscribe success/missing endpoint), `utils/push_test.go` (SendPushToSubscription mock, no-op VAPID kosong, kirim ke semua subscription, payload format).
- Env docs: `backend/.env.docker.example` & `.env.example` tambah `VAPID_PUBLIC_KEY`/`VAPID_PRIVATE_KEY`/`VAPID_EMAIL`.

timestamp: 2026-08-07 17:30 WITA
kategori: Sedang, Backend
file yang diubah:
- backend/models/push_subscription.go
- backend/migrations/000002_push_subscriptions.up.sql
- backend/migrations/000002_push_subscriptions.down.sql
- backend/utils/push.go
- backend/utils/push_test.go
- backend/handlers/push.go
- backend/handlers/push_test.go
- backend/routes/push.go
- backend/utils/notify.go
- backend/main.go
- backend/go.mod
- backend/go.sum
- backend/docs/docs.go
- backend/docs/swagger.json
- backend/docs/swagger.yaml
- backend/changelog.md
- .env.example
- backend/.env.docker.example
alasan perubahan: Task 8.3 ROADMAP (Fase 8) — memastikan teknisi tidak melewatkan tugas penting dengan notifikasi OS (push) sebagai pelengkap notifikasi in-app; push dikirim async sehingga tidak menggagalkan operasi utama dan memiliki fallback in-app bila VAPID tidak dikonfigurasi.

## [0.40.1] - 2026-08-07

### Infrastruktur: Fix CI — Deprecation Node 20 & errcheck pada test logger

perubahan yang dilakukan: **Memperbaiki error job CI backend: (1) deprecation runtime Node 20 di GitHub Actions (upgrade `golangci/golangci-lint-action@v7` → `@v9` memakai Node runtime 24, dan `node-version` frontend 20 → 24); (2) error lint `errcheck` di `backend/utils/logger_test.go` karena `os.Setenv`/`os.Unsetenv` tidak dicek.**

before: Job CI backend (`Run Go Linter`) memakai `golangci/golangci-lint-action@v7` (runtime Node 20 — sudah di-deprecate di runner, muncul warning/error). Job frontend memakai `node-version: '20'`. Selain itu `TestInitLogger` memakai `os.Setenv`/`os.Unsetenv` tanpa memeriksa error sehingga linter `errcheck` gagal.

after:
- `.github/workflows/ci.yml`: `golangci/golangci-lint-action@v7` → `@v9` (v9.0.0+ memakai Node runtime 24; `version: v2.12.2` & `working-directory: ./backend` tetap valid). `node-version: '20'` → `'24'` pada job `frontend-ci`.
- `backend/utils/logger_test.go`: ganti `os.Setenv`/`os.Unsetenv` dengan `t.Setenv("LOG_LEVEL", "info")` (idiomatik, auto-cleanup, mem-fail test bila gagal) dan hapus import `os` yang tak terpakai.
- Verifikasi lokal: `golangci-lint run ./...` → **0 issues**; `go build`, `go test ./... -count=1`, `go vet` → PASS.

timestamp: 2026-08-07 14:20 WITA
kategori: Kecil, Infrastruktur CI
file yang diubah:
- .github/workflows/ci.yml
- backend/utils/logger_test.go
- backend/changelog.md
alasan perubahan: Node 20 sudah di-deprecate di GitHub Actions runner (github.blog/changelog/2025-09-19); gunakan action/runtime terbaru (Node 24) dan perbaiki `errcheck` pada test agar CI backend tetap hijau.

## [0.40.0] - 2026-08-07

### Infrastruktur: Container Hardening (Task 7.5) & Logging Terstruktur (Task 7.6)

perubahan yang dilakukan: **Mengeraskan container produksi (non-root user, read-only rootfs + tmpfs, drop capabilities, resource limits, Docker secrets untuk DB_PASSWORD/JWT_SECRET/AES_SECRET_KEY) dan meningkatkan observabilitas log (rotasi via lumberjack, level dari LOG_LEVEL, format JSON konsisten).**

before: Container api berjalan sebagai root tanpa batasan resource/capability dan tanpa read-only rootfs; secret (DB_PASSWORD, JWT_SECRET, AES_SECRET_KEY) dikirim sebagai env var sehingga terexpose di `docker inspect`/proses env. Logger menulis ke satu file `logs/server.log` tanpa rotasi (berisiko membengkak) dan level log tidak bisa dikonfigurasi.

after:
- `backend/Dockerfile`: user non-root `sigap` (`addgroup -S sigap && adduser -S -G sigap sigap`); kepemilikan `/app/uploads` & `/app/logs` oleh `sigap`; `USER sigap` sebelum `CMD`; `chmod +x` binary.
- `docker-compose.yml` (service `api`): `read_only: true` + tmpfs `/tmp` & `/app/logs`; `cap_drop: [ALL]`; resource limits (`deploy.resources.limits` cpus 1.0 / mem 512M + `mem_limit`/`cpus` untuk kompatibilitas lokal); Docker secrets `db_password`/`jwt_secret`/`aes_secret_key` dari `./secrets/*` di-mount ke `/run/secrets/`; env `LOG_LEVEL`; `nginx` diberi resource limits.
- `backend/utils/secrets.go` (baru): `GetSecret(env, file)` — baca file `/run/secrets/<nama>` bila ada (trim whitespace) dengan fallback ke env var; `RequireSecret` versi fail-fast (fatal bila kosong, tanpa fallback hardcoded).
- Wiring secret: `config/database.go` `databaseDSN()` → `utils.GetSecret("DB_PASSWORD", "/run/secrets/db_password")`; `utils/jwt.go` `getJWTSecret()` → `GetSecret("JWT_SECRET", "/run/secrets/jwt_secret")`; `utils/encryption.go` `deriveKey()` → `GetSecret("AES_SECRET_KEY", "/run/secrets/aes_secret_key")`; `main.go` → `utils.RequireSecret(...)` menggantikan `requireEnv`.
- `backend/go.mod`/`go.sum`: `gopkg.in/natefinch/lumberjack.v2 v2.2.1`.
- `backend/utils/logger.go`: `parseLogLevel(env LOG_LEVEL, default info)`; file core memakai `lumberjack.Logger` (`MaxSize 100 MB`, `MaxBackups 5`, `MaxAge 30 hari`, `Compress true`); level diterapkan konsisten di core file (JSON) & console; file encoder tetap JSON.
- Konsistensi logging: `middleware/logger.go`, `utils/audit.go`, `middleware/recovery.go` sudah memakai `utils.Log` terstruktur (JSON) — diverifikasi tanpa `fmt.Println`/`log.Printf`.
- Test: `utils/secrets_test.go` (baru: file menang atas env, fallback env, kosong bila keduanya hilang, trim whitespace); `utils/logger_test.go` (baru: `parseLogLevel` valid/invalid/empty/case-insensitive, `InitLogger` non-nil).
- Env docs: `.env.example`, `backend/.env.docker.example` (LOG_LEVEL + catatan secrets); `.gitignore` tambah `/secrets/`.

timestamp: 2026-08-07 13:40 WITA
kategori: Sedang, Backend & Infrastruktur
file yang diubah:
- backend/Dockerfile
- docker-compose.yml
- backend/utils/secrets.go
- backend/utils/secrets_test.go
- backend/utils/logger.go
- backend/utils/logger_test.go
- backend/config/database.go
- backend/utils/jwt.go
- backend/utils/encryption.go
- backend/main.go
- backend/go.mod
- backend/go.sum
- backend/changelog.md
- .env.example
- backend/.env.docker.example
- .gitignore
- docs/changelog.md
- docs/ROADMAP.md
- docs/plan.md
- docs/TESTING.md
- docs/AGENTS.md
alasan perubahan: Task 7.5 & 7.6 ROADMAP (asal Fase 4.7) — container hardening (non-root, read-only, cap_drop, resource limits, secrets) dan observabilitas log (rotasi + level) wajib sebelum go-live untuk memperkecil permukaan serangan dan memastikan log tidak membengkak serta mudah diparsing.

## [0.39.0] - 2026-08-07

### Infrastruktur: Monitoring & Alerting (Task 7.4)

perubahan yang dilakukan: **Menambahkan endpoint `GET /api/metrics` (format text exposition Prometheus) dan collector metrik tanpa library pihak ketiga, middleware pencatatan request, serta memperluas `GET /api/health` (response time DB, cek Redis, ukuran folder uploads). Menyediakan gauge DB pool, WebSocket aktif, dan ukuran uploads; menambah konfigurasi Prometheus & Grafana opsional di `docker-compose.override.yml`.**

before: Tidak ada observabilitas runtime — masalah (lonjakan latensi, koneksi DB/WS, pertumbuhan storage uploads) baru terdeteksi setelah user melapor. `GET /api/health` hanya mengecek koneksi database, tanpa Redis, disk/ukuran uploads, maupun response time.

after:
- `monitoring/metrics.go` (baru): collector Prometheus ringan tanpa library pihak ketiga (sesuai konvensi proyek) — counter `sigap_http_requests_total{method,path,status}`, histogram `sigap_http_request_duration_seconds{method,path}` (bucket kumulatif + _sum/_count), gauge `sigap_db_pool_open_connections`, `sigap_db_pool_idle_connections`, `sigap_ws_active_connections`, `sigap_upload_storage_bytes`. `Render()` menghasilkan text exposition Prometheus valid; `Handler` Fiber untuk `GET /api/metrics` (`text/plain; version=0.0.4`). Thread-safe (RWMutex); `Reset()` untuk pengujian.
- `monitoring/collector.go` (baru): `StartCollector(uploadsDir, interval)` goroutine memperbarui gauge DB pool (`sql.DB.Stats()`), WS aktif (`utils.Hub.ActiveConnections()`), dan ukuran uploads (rekursif).
- `middleware/metrics.go` (baru): mencatat setiap request — increment counter (method, path pola route, status) + observe durasi histogram; endpoint `/api/metrics` dikecualikan agar tidak mencemari statistik.
- `handlers/health.go`: diperluas — `db_response_time_ms`, `redis_status`/`redis_response_time_ms` (bila Redis aktif), `uploads{status,path,size_bytes}`; helper `dirSize` (ukuran folder rekursif).
- `utils/websocket.go`: tambah `ActiveConnections()` (jumlah koneksi WebSocket aktif, thread-safe).
- `main.go`: pasang `middleware.Metrics()`; rute `GET /api/metrics` (publik); jalankan `monitoring.StartCollector("./uploads", 15*time.Second)`.
- `monitoring/prometheus/prometheus.yml` (baru): scrape `api:3000/api/metrics` tiap 15 detik.
- `docker-compose.override.yml` (baru): service `prometheus` (port 9090) & `grafana` (port 3000, datasource auto-provision) + volume `prometheus_data`/`grafana_data` — opsional, tidak diwajibkan di compose utama.
- `monitoring/grafana/provisioning/datasources/prometheus.yml` (baru): auto-register datasource Prometheus.
- Test: `monitoring/metrics_test.go` (counter/histogram/HELP-TYPE/gauge/empty/reset/escape/format angka), `middleware/metrics_test.go` (observasi, exclude /api/metrics, status non-200, Content-Type), `handlers/health_test.go` diperluas (+ `dirSize`).
- Swagger: `swag init` — endpoint `/metrics` & deskripsi health baru.

timestamp: 2026-08-07 12:50 WITA
kategori: Sedang, Backend & Infrastruktur
file yang diubah:
- backend/monitoring/metrics.go
- backend/monitoring/metrics_test.go
- backend/monitoring/collector.go
- backend/middleware/metrics.go
- backend/middleware/metrics_test.go
- backend/handlers/health.go
- backend/handlers/health_test.go
- backend/utils/websocket.go
- backend/main.go
- backend/docs/docs.go
- backend/docs/swagger.json
- backend/docs/swagger.yaml
- backend/changelog.md
- docker-compose.override.yml
- monitoring/prometheus/prometheus.yml
- monitoring/grafana/provisioning/datasources/prometheus.yml
- docs/changelog.md
- docs/ROADMAP.md
- docs/plan.md
- docs/TESTING.md
- docs/AGENTS.md
alasan perubahan: Task 7.4 ROADMAP (asal Fase 4.6) — observabilitas runtime wajib sebelum go-live agar masalah (latensi, koneksi, storage) terdeteksi secara proaktif, bukan menunggu laporan user; endpoint metrik Prometheus menjadi dasar alerting & dashboard.

## [0.38.0] - 2026-08-07

### Infrastruktur: Backup Otomatis & Restore (Task 7.3)

perubahan yang dilakukan: **Menambahkan skrip backup otomatis (`scripts/backup.sh`) dan restore (`scripts/restore.sh`) untuk database PostgreSQL dan folder uploads, container `backup` di docker-compose dengan cron, volume `backups_data`, serta dokumentasi `docs/BACKUP.md`. Backup database memakai `pg_dump` (opsi `--no-owner --no-privileges`) yang dikompres gzip ke `backups/db/sigap_YYYYMMDD_HHMMSS.sql.gz`; uploads di-tar+gzip ke `backups/uploads/uploads_YYYYMMDD_HHMMSS.tar.gz`; rotasi otomatis menghapus backup lebih dari 7 hari. Restore mendukung pemulihan database (drop/recreate lalu feed dump) dan/atau uploads.**

before: Tidak ada mekanisme backup/pemulihan — database dan foto bukti uploads rentan kehilangan data permanen bila terjadi kegagalan perangkat, penghapusan tak sengaja, atau korupsi data. Tidak ada disaster recovery tersedia di hari pertama go-live.

after:
- `scripts/backup.sh` (baru): `pg_dump` → `gzip` → `backups/db/sigap_<TS>.sql.gz`; tar+gzip folder uploads → `backups/uploads/uploads_<TS>.tar.gz` (archive kosong bila folder kosong agar restore konsisten); rotasi `find -mtime +RETENTION_DAYS -delete` (default 7 hari); timestamp WITA (Asia/Makassar); konfigurasi via env `DB_HOST/DB_PORT/DB_USER/DB_PASSWORD/DB_NAME/UPLOADS_DIR/BACKUP_DIR/RETENTION_DAYS` (default menyesuaikan deployment Docker).
- `scripts/restore.sh` (baru): argumen `<sql_dump.gz>` [optional `<uploads.tar.gz>`]; validasi keberadaan file; restore DB = drop & recreate via `psql` lalu `gunzip` dump → `psql`; restore uploads = hapus & buat ulang folder lalu `tar -xzf`.
- `backup/Dockerfile` (baru): image `postgres:15-alpine` (klien PostgreSQL bawaan) + `dcron` + `bash`; menyalin `scripts/backup.sh`, `scripts/restore.sh`, dan `backup/entrypoint.sh`.
- `backup/entrypoint.sh` (baru): menulis crontab (`CRON_SCHEDULE` default `0 6 * * *`, `TZ=Asia/Makassar`) yang memanggil `/usr/local/bin/backup.sh`, lalu menjalankan `crond -f` di foreground.
- `docker-compose.yml`: service `backup` (volume `backups_data:/backups`, uploads bind-mount read-only, `depends_on.db` healthy, kredensial dari root `.env`); volume `backups_data`.
- `.gitignore`: tambah `/backups/`.

timestamp: 2026-08-07 04:00 WITA
kategori: Sedang, Backend & Infrastruktur
file yang diubah:
- scripts/backup.sh
- scripts/restore.sh
- backup/Dockerfile
- backup/entrypoint.sh
- docker-compose.yml
- .gitignore
- docs/BACKUP.md
- docs/changelog.md
- docs/ROADMAP.md
- docs/plan.md
- docs/AGENTS.md
- backend/changelog.md
alasan perubahan: Task 7.3 ROADMAP (asal Fase 4.5) — disaster recovery wajib tersedia sebelum data nyata masuk di hari pertama go-live; backup otomatis database + uploads dengan rotasi memastikan sistem dapat dipulihkan bila terjadi kegagalan.

## [0.37.0] - 2026-08-06

### Infrastruktur: Redis untuk Rate-Limit & Blacklist (Task 7.2)

perubahan yang dilakukan: **Mengadopsi Redis sebagai shared store untuk rate limiter dan token blacklist (mendukung multi-instance API; blacklist persisten lintas restart). Blacklist dipindah dari map in-memory ke Redis SET (SADD/SISMEMBER/SREM + TTL otomatis), rate limiter diganti dari storage bawaan Fiber ke counter fixed-window berbasis Redis INCR/EXPIRE dengan key `limiter:{ip}:{endpoint}`, ditambah graceful fallback ke in-memory saat Redis tidak tersedia, container Redis 7 Alpine di docker-compose, dan env baru `REDIS_HOST`/`REDIS_PORT`/`REDIS_PASSWORD` (opsional).**

before: Token blacklist dan rate-limit counter tersimpan in-memory per proses — data hilang saat server restart dan tidak berfungsi lintas instance (multi-instance API rentan bypass rate limit & token ter-revoke bisa dipakai lagi setelah restart).

after:
- `config/redis.go` (baru): `ConnectRedis()` memakai `github.com/redis/go-redis/v9` — host/port/password dari env `REDIS_HOST`/`REDIS_PORT`/`REDIS_PASSWORD` (default `redis:6379`); verifikasi `Ping` ber-timeout 2 detik; bila gagal hanya LOG WARNING + `RedisEnabled=false` (graceful fallback, tidak fail-fast).
- `utils/blacklist.go`: interface `TokenBlacklist` (`Add`/`IsRevoked`/`Remove`/`CleanupExpired`) + dua implementasi — `RedisTokenBlacklist` (SET `sigap:blacklist:jwt`, TTL mengikuti token dengan sisa masa hidup terpanjang, `CleanupExpired` no-op) dan `memoryTokenBlacklist` (fallback). Facade `RevokeToken`/`IsTokenRevoked`/`SetTokenBlacklist`/`StartBlacklistCleanup` tetap kompatibel dengan pemanggil lama.
- `middleware/limiter.go`: limiter fixed-window kustom menggantikan `fiber/middleware/limiter` — backend Redis (`INCR` + `EXPIRE` saat INCR=1) dengan fallback in-memory per-proses; key `limiter:{ip}:{endpoint}` (endpoint = pola route, fallback path aktual); `GlobalLimiter` 100/menit & `AuthLimiter` 5/menit tetap dengan signature sama.
- `main.go`: `config.ConnectRedis()` setelah `ConnectDB`; `SetTokenBlacklist` memilih Redis atau in-memory; penutupan koneksi Redis pada graceful shutdown.
- `docker-compose.yml`: service `redis` (redis:7-alpine) — volume `redis_data` (AOF), health check adaptif terhadap password, port 6379 tidak dipublish; `api` menunggu `redis` healthy; komentar service dinomori ulang.
- `.env.example` & `backend/.env.docker.example`: `REDIS_PASSWORD` (opsional) + `REDIS_HOST`/`REDIS_PORT`/`REDIS_PASSWORD`.
- `utils/blacklist_test.go`: test kontrak interface (add/check/remove) untuk varian Redis (mock `redisCommander`) & in-memory, TTL mengikuti token terpanjang, facade `RevokeToken`/`IsTokenRevoked`, cleanup expired, dan pergantian backend via `SetTokenBlacklist`.
- `middleware/limiter_test.go`: test lama tetap hijau + format key `limiter:{ip}:{endpoint}`, reset counter in-memory setelah window, isolasi counter per key.
- `go.mod`/`go.sum`: dependensi `github.com/redis/go-redis/v9 v9.22.0`.

timestamp: 2026-08-06 17:50 WITA
kategori: Sedang, Backend
file yang diubah:
- backend/config/redis.go
- backend/utils/blacklist.go
- backend/utils/blacklist_test.go
- backend/middleware/limiter.go
- backend/middleware/limiter_test.go
- backend/main.go
- backend/go.mod
- backend/go.sum
- backend/changelog.md
- docker-compose.yml
- .env.example
- backend/.env.docker.example
- docs/changelog.md
- docs/ROADMAP.md
- docs/plan.md
- docs/TESTING.md
- docs/AGENTS.md
alasan perubahan: Task 7.2 ROADMAP (asal Fase 4.4) — kelemahan arsitektur in-memory blacklist (hilang saat restart) dan rate limit per-proses (tidak efektif multi-instance) harus diperbaiki sebelum go-live; Redis juga menjadi prasyarat shared state untuk fitur produksi berikutnya.

## [0.36.0] - 2026-08-06

### Infrastruktur: Migrasi DB Versioned (Task 7.1)

perubahan yang dilakukan: **Mengadopsi golang-migrate v4 (driver pgx v5) untuk mengelola skema database dengan file migration SQL terurut di `backend/migrations/`. Menambahkan `config.RunMigrations` yang berjalan otomatis saat startup, CLI manual `go run cmd/migrate/main.go up|down|version`, fase transisi (AutoMigrate dikomentari dengan catatan akan dihapus setelah migration stabil), integrasi verifikasi migration di CI, dan dokumentasi prosedur di `docs/MIGRATION.md`.**

before: Skema database hanya dikelola GORM AutoMigrate dari definisi struct — tanpa riwayat versi, tanpa rollback andal, dan perubahan skema produksi berisiko kehilangan data.

after:
- `migrations/migrations.go`: embed FS untuk seluruh file `*.sql` (kompilasi ke binary).
- `migrations/000001_init_schema.up.sql`: skema awal — 10 tabel (users, cctvs, maintenance_logs, field_maintenances, field_maintenance_checks, field_maintenance_images, audit_logs, notifications, maintenance_schedules, daily_digests) + index + 13 foreign key. Seluruh pernyataan idempoten (IF NOT EXISTS / blok DO $$ pengecekan pg_constraint) sehingga aman dijalankan pada database eksisting hasil AutoMigrate (fase transisi).
- `migrations/000001_init_schema.down.sql`: rollback seluruh tabel (CASCADE pada maintenance_logs untuk FK sirkular).
- `config/migrate.go`: `RunMigrations(db)` (startup, `m.Up()` + toleransi ErrNoChange), `MigrationUp/MigrationDown/MigrationVersion` (CLI), `newMigrateInstance` (iofs + driver pgx/v5), `logMigrationVersion`.
- `cmd/migrate/main.go`: CLI helper `go run cmd/migrate/main.go up|down|version`.
- `main.go`: `RunMigrations` dijalankan sebelum operasi lain; blok `AutoMigrate` dikomentari dengan catatan "akan dihapus setelah migration stabil".
- `.github/workflows/ci.yml`: job backend-ci kini menyediakan service PostgreSQL dan menjalankan verifikasi `up → down → up → version` end-to-end.
- `config/versioned_migration_test.go`: test embed FS + konvensi penamaan, kelengkapan tabel up/down, kolom kunci, dan propagasi error driver tanpa PostgreSQL fisik.
- `docs/MIGRATION.md`: dokumentasi prosedur (baru).
- `go.mod`/`go.sum`: dependensi `github.com/golang-migrate/migrate/v4 v4.19.1` (+ `github.com/jackc/pgerrcode` indirect).

timestamp: 2026-08-06 16:35 WITA
kategori: Besar, Backend
file yang diubah:
- backend/migrations/migrations.go
- backend/migrations/000001_init_schema.up.sql
- backend/migrations/000001_init_schema.down.sql
- backend/config/migrate.go
- backend/config/versioned_migration_test.go
- backend/cmd/migrate/main.go
- backend/main.go
- backend/go.mod
- backend/go.sum
- backend/changelog.md
- .github/workflows/ci.yml
- docs/MIGRATION.md
alasan perubahan: Menghilangkan ketergantungan pada AutoMigrate yang tidak memiliki riwayat versi dan rollback; migration versioned menjadi prasyarat Fase 7 sebelum go-live (ROADMAP Task 7.1, asal Fase 4.3) agar perubahan skema produksi terkontrol dan dapat dipulihkan.

## [0.35.0] - 2026-08-06

### Fitur: Ringkasan Operasional Otomatis (Task 6.4)

perubahan yang dilakukan: **Menambahkan model `DailyDigest`, generator ringkasan harian yang berjalan setiap pukul 06:00 WITA, endpoint Admin `GET /api/dashboard/digest`, dan notifikasi in-app `DAILY_DIGEST_AVAILABLE` ke seluruh Admin. Ringkasan mengagregasi tugas baru/selesai/gagal/overdue, pengecekan lapangan, hasil attention, dan perubahan status CCTV pada jendela 24 jam terakhir.**

before: Admin belum memiliki arsip ringkasan harian yang menggabungkan pergerakan tugas, pengecekan lapangan, dan perubahan status CCTV secara otomatis.

after:
- `models/daily_digest.go`: model `DailyDigest` dengan `digest_date` unique, tujuh metrik, dan summary naratif.
- `models/notification.go`: konstanta `DAILY_DIGEST_AVAILABLE`.
- `handlers/digest_generator.go`: scheduler 06:00 WITA, agregasi 24 jam, idempotensi per tanggal, penyimpanan, dan notifikasi Admin.
- `handlers/dashboard.go`: endpoint `GetDailyDigests` dengan role admin, paginasi, dan urutan tanggal terbaru.
- `routes/dashboard.go`: route `/dashboard/digest` protected + `RequireRole("admin")`.
- `main.go`: AutoMigrate `DailyDigest` dan goroutine generator saat startup.
- `handlers/digest_generator_test.go`, `routes/routes_test.go`: coverage generator, idempotensi, jadwal, notifikasi, endpoint, dan proteksi route.
- `docs/docs.go`, `docs/swagger.json`, `docs/swagger.yaml`: dokumentasi Swagger diregenerasi.

timestamp: 2026-08-06 15:35 WITA
kategori: Sedang, Backend
file yang diubah:
- backend/models/daily_digest.go
- backend/models/notification.go
- backend/handlers/digest_generator.go
- backend/handlers/digest_generator_test.go
- backend/handlers/dashboard.go
- backend/routes/dashboard.go
- backend/routes/routes_test.go
- backend/main.go
- backend/docs/docs.go
- backend/docs/swagger.json
- backend/docs/swagger.yaml
- backend/changelog.md
alasan perubahan: Mengubah data operasional harian menjadi ringkasan yang dapat dipantau Admin tanpa query manual, dengan idempotensi dan notifikasi in-app sebagai pengingat.

## [0.34.0] - 2026-08-06

### Fitur: Laporan PDF Pengecekan Lapangan (Task 6.3)

perubahan yang dilakukan: **Menambahkan endpoint protected `GET /api/field-maintenance/:id/report-pdf` untuk menghasilkan laporan PDF per-record dengan kop surat IKN, informasi CCTV dan teknisi, status pengecekan, tabel checklist, foto bukti, serta catatan umum. Akses Admin mencakup seluruh record, sedangkan teknisi dibatasi pada record miliknya sendiri.**

before: Catatan pengecekan lapangan hanya dapat dilihat melalui respons JSON dan belum memiliki laporan PDF per-record yang siap diarsipkan atau dicetak.

after:
- `handlers/field_maintenance.go`: handler PDF berbasis `go-pdf/fpdf`, ownership check, tabel checklist, embed foto lokal yang valid, dan fallback URL foto.
- `routes/field_maintenance.go`: route `GET /field-maintenance/:id/report-pdf` protected didaftarkan sebelum route detail dinamis.
- `handlers/field_maintenance_test.go`: test content type/prefix PDF, akses teknisi pemilik/non-pemilik, dan record tidak ditemukan.
- `routes/routes_test.go`: test route field maintenance wajib autentikasi.
- `docs/docs.go`, `docs/swagger.json`, `docs/swagger.yaml`: dokumentasi Swagger diregenerasi.

timestamp: 2026-08-06 15:05 WITA
kategori: Sedang, Backend
file yang diubah:
- backend/handlers/field_maintenance.go
- backend/handlers/field_maintenance_test.go
- backend/routes/field_maintenance.go
- backend/routes/routes_test.go
- backend/docs/docs.go
- backend/docs/swagger.json
- backend/docs/swagger.yaml
- backend/changelog.md
alasan perubahan: Melengkapi siklus dokumentasi pengecekan lapangan dengan berkas PDF yang dapat diarsipkan, sambil mempertahankan pembatasan akses berdasarkan peran dan kepemilikan data.

## [0.33.0] - 2026-08-06

### Fitur: Dashboard Kinerja Teknisi (Task 6.2)

perubahan yang dilakukan: **Menambahkan endpoint admin `GET /api/dashboard/technician-performance` dengan filter rentang tanggal, agregasi tugas pemeliharaan, waktu penyelesaian, overdue, pengecekan lapangan, dan quality score. Menambahkan `MaintenanceLog.AssignedAt` agar durasi completion dihitung dari waktu penugasan ke `completed_at`; log lama memakai `created_at` sebagai fallback.**

before: SIGAP belum menyediakan ringkasan beban kerja dan kualitas teknisi yang dapat dibandingkan admin dalam satu rentang waktu. Waktu mulai tugas juga belum dicatat secara eksplisit.

after:
- `models/setup.go`: field nullable `MaintenanceLog.AssignedAt` untuk waktu mulai penugasan.
- `handlers/maintenance.go`: waktu assign/re-assign disimpan pada `AssignedAt`.
- `handlers/schedule_runner.go`: recurring log yang langsung memiliki teknisi menyimpan `AssignedAt` saat dibuat.
- `handlers/dashboard.go`: handler `GetTechnicianPerformance`, agregasi satu query, filter default 30 hari, completion rate, rata-rata jam, overdue, field check, dan quality score.
- `routes/dashboard.go`: endpoint kinerja teknisi dilindungi `Protected()` dan `RequireRole("admin")`.
- `handlers/technician_performance_test.go`: test struktur respons, filter tanggal, dan penolakan role non-admin.
- Swagger docs diregenerasi dengan `swag init`.

timestamp: 2026-08-06 14:18 WITA
kategori: Sedang, Backend
file yang diubah:
- backend/models/setup.go
- backend/handlers/dashboard.go
- backend/handlers/maintenance.go
- backend/handlers/schedule_runner.go
- backend/handlers/technician_performance_test.go
- backend/routes/dashboard.go
- backend/docs/docs.go
- backend/docs/swagger.json
- backend/docs/swagger.yaml
- backend/changelog.md
alasan perubahan: Memberikan dasar evaluasi beban kerja dan kualitas teknisi dari data operasional yang sudah tersimpan, tanpa query N+1 atau dependency baru.

## [0.32.0] - 2026-08-06

### Fitur: Skor Kesehatan CCTV (Task 6.1)

perubahan yang dilakukan: **Menambahkan `CCTV.HealthScore` (integer 0–100, default 100), endpoint protected `GET /api/cctv/:id/health-score`, dan kalkulator berkala yang berjalan saat startup lalu setiap 6 jam. Skor memakai lima faktor dalam 90 hari terakhir: failed, field-check attention, overdue, umur sejak MaintenanceLog completed terakhir, dan item checklist not_ok. Skor disimpan kembali ke CCTV, dipancarkan lewat event `CCTV_UPDATED`, serta dihitung ulang setelah complete maintenance atau field-check berhasil**.

before: Data CCTV hanya memiliki status operasional dan belum mempunyai indikator gabungan yang menunjukkan kesehatan perangkat berdasarkan riwayat pemeliharaan, keterlambatan tugas, dan hasil pengecekan lapangan.

after:
- `models/cctv.go`: field `HealthScore int` dengan default database 100 dan JSON `health_score`.
- `handlers/health_score.go` (baru): agregasi PostgreSQL lima faktor dari jendela 90 hari, bobot penalti, clamp 0–100, `CalculateCCTVHealthScoreAt`, `RecalculateCCTVHealthScore`, dan `RunHealthScoreCalculator` interval 6 jam.
- `handlers/cctv.go`: handler `GetCCTVHealthScore` protected semua role; list CCTV menambahkan `health_score` ke whitelist sorting.
- `routes/cctv.go`: rute `GET /cctv/:id/health-score` didaftarkan sebelum rute `/:id`.
- `handlers/maintenance.go` dan `handlers/field_maintenance.go`: recalculation setelah transaksi complete/field-check sukses.
- `handlers/health_score_test.go`: unit test default, kombinasi lima faktor, ambang umur, clamp, dan endpoint sqlmock.
- `main.go`: kalkulator skor dijalankan sebagai goroutine saat startup.
- Swagger docs diregenerasi (`swag init`) karena endpoint baru.

timestamp: 2026-08-06 13:22 WITA
kategori: Besar, Backend
file yang diubah:
- backend/models/cctv.go
- backend/handlers/health_score.go
- backend/handlers/health_score_test.go
- backend/handlers/cctv.go
- backend/handlers/cctv_test.go
- backend/handlers/helpers_test.go
- backend/handlers/maintenance.go
- backend/handlers/maintenance_test.go
- backend/handlers/field_maintenance.go
- backend/handlers/field_maintenance_test.go
- backend/routes/cctv.go
- backend/main.go
- backend/docs/docs.go
- backend/docs/swagger.json
- backend/docs/swagger.yaml
- backend/changelog.md
alasan perubahan: Memberikan indikator prioritas yang dapat dipindai admin dari data operasional yang sudah ada, tanpa menambah tabel baru atau ketergantungan pihak ketiga.

## [0.31.0] - 2026-08-06

### Fitur: Eskalasi Overdue Otomatis (Task 5.5)

perubahan yang dilakukan: **Menambahkan flag nullable `OverdueNotifiedAt` pada `MaintenanceLog` dan goroutine checker overdue yang berjalan saat startup lalu setiap 1 jam. Checker memilih log berstatus `scheduled`/`ongoing` yang `scheduled_date`-nya telah lewat dan belum pernah dieskalasikan, mengirim notifikasi `MAINTENANCE_OVERDUE` ke seluruh admin serta teknisi untuk log `ongoing`, kemudian menyimpan timestamp agar tidak mengirim duplikat. Reschedule ke masa depan mengosongkan flag tersebut. Generator recurring dan checker kini benar-benar dijalankan sebagai goroutine agar server Fiber tetap dapat melakukan bootstrap**.

before: Tidak ada timestamp eskalasi pada `MaintenanceLog` dan tidak ada proses background yang memberi tahu admin/teknisi saat tugas melewati jadwal. Perubahan tanggal jadwal tidak dapat membuka kembali siklus eskalasi. `RunScheduleGenerator` dipanggil sinkron sehingga ticker-nya memblokir inisialisasi server.

after:
- `models/setup.go`: `MaintenanceLog.OverdueNotifiedAt *time.Time` ditambahkan; `nil` berarti belum pernah dinotifikasi atau flag telah di-reset.
- `handlers/overdue_checker.go` (baru): `RunOverdueChecker`, `ProcessOverdueMaintenance`, dan `ProcessOverdueMaintenanceAt`; query hanya mengambil log aktif yang overdue dan belum ditandai; pesan admin memuat nama/kode CCTV + tanggal, pesan teknisi memuat nama CCTV; update flag memakai kondisi `IS NULL`.
- `handlers/maintenance.go`: `UpdateMaintenanceLog` me-reset `OverdueNotifiedAt` saat `scheduled_date` valid dipindahkan ke masa depan.
- `utils/notify.go`: `NotifyUserSync`/`NotifyRoleSync` untuk job background yang harus menunggu persist notifikasi sebelum menandai sumber.
- `main.go`: generator recurring dan checker overdue dijalankan dengan `go` saat startup.
- `handlers/overdue_checker_test.go`: test sqlmock untuk eskalasi admin, eskalasi teknisi pada status `ongoing`, skip flag, skip status terminal, dan reset saat reschedule.
- Swagger tidak memiliki anotasi endpoint baru sehingga tidak ada perubahan pada hasil `swag init`.

timestamp: 2026-08-06 12:06 WITA
kategori: Sedang, Backend
file yang diubah:
- backend/models/setup.go
- backend/handlers/overdue_checker.go
- backend/handlers/overdue_checker_test.go
- backend/handlers/maintenance.go
- backend/utils/notify.go
- backend/main.go
- backend/changelog.md
alasan perubahan: Mencegah tugas terjadwal terlewat tanpa eskalasi, menjaga notifikasi hanya dikirim sekali per siklus jadwal, dan memastikan reschedule membuka kembali siklus tersebut tanpa memblokir startup API.

## [0.30.0] - 2026-08-06

### Fitur: Penjadwalan Pemeliharaan Berulang (Recurring) - Task 5.4

perubahan yang dilakukan: **Modul baru `maintenance_schedules` untuk memjadwalkan pemeliharaan berulang: admin membuat jadwal (CCTV + tindakan + frekuensi daily/weekly/monthly + teknisi opsional), generator berjalan saat startup dan tiap 1 jam membuat `MaintenanceLog` otomatis untuk jadwal aktif yang `next_run_at`-nya sudah tiba — idempotent (satu log per hari per jadwal). Log tanpa teknisi dibuat berstatus `scheduled`; dengan teknisi langsung `ongoing` + notifikasi in-app. `MaintenanceLog` mendapat kolom `maintenance_schedule_id` untuk penelusuran asal log. Audit khusus: CREATE/UPDATE/DELETE/TOGGLE_SCHEDULE dari admin, CREATE_MAINTENANCE "Dibuat otomatis oleh jadwal [id]" oleh sistem**.

before: Pemeliharaan hanya bisa dicatat manual oleh admin — tidak ada mekanisme pengingat maupun pembuatan log otomatis sehingga pemeliharaan berkala berisiko terlewat.

after:
- `models/maintenance_schedule.go` (baru): `MaintenanceSchedule` (CCTVID FK, TechnicianID FK opsional ke users, ActionTaken text, Frequency varchar(20), NextRunAt timestamptz, IsActive default true, LastGeneratedAt *time.Time) + konstanta `FrequencyDaily/Weekly/Monthly`; relasi `CCTV` & `Technician` (User).
- `models/setup.go`: `MaintenanceLog` + kolom `MaintenanceScheduleID *uuid.UUID` (index) + relasi `MaintenanceSchedule`.
- `handlers/maintenance_schedule.go` (baru): CRUD lengkap `CreateMaintenanceSchedule` (validasi CCTV ada & teknisi berperan `teknisi`, NextRunAt = now + frekuensi, audit CREATE_SCHEDULE), `GetAllMaintenanceSchedules` (paginasi + filter `is_active` + preload CCTV/Teknisi), `GetMaintenanceScheduleByID`, `UpdateMaintenanceSchedule` (reset NextRunAt saat frekuensi berubah; teknisi: `""` = lepas, UUID = ganti, tanpa field = tetap; audit UPDATE_SCHEDULE), `DeleteMaintenanceSchedule` (soft delete; audit DELETE_SCHEDULE), `ToggleMaintenanceSchedule` (aktif/nonaktif; audit TOGGLE_SCHEDULE) + helper `nextRunForFrequency` & `isValidFrequency`. Swagger tag "Jadwal Pemeliharaan".
- `handlers/schedule_runner.go` (baru): `RunScheduleGenerator(db)` berjalan sekali saat startup lalu tiap 1 jam; `ProcessDueSchedulesAt(db, now)` mencari jadwal `is_active=true AND next_run_at <= now`; idempotensi per hari (maintenance_schedule_id + rentang hari); audit oleh sistem (identitas `system`, bukan HTTP); broadcast `MAINTENANCE_CREATED`; notifikasi `MAINTENANCE_ASSIGNED` ke teknisi saat jadwal berteknisi.
- `routes/maintenance_schedule.go` (baru): grup `/maintenance-schedules` (Protected + RequireRole("admin")) — GET/POST `/`, GET/PUT `/`:id, PUT `/:id/toggle`, DELETE `/:id`.
- `main.go`: `&models.MaintenanceSchedule{}` masuk AutoMigrate; `handlers.RunScheduleGenerator(config.DB)` dijalankan saat startup (interval 1 jam); `routes.SetupMaintenanceScheduleRoutes(api)`.
- `handlers/maintenance_schedule_test.go` (baru): CRUD (success, CCTV tidak ditemukan, teknisi non-teknisi ditolak, not-found, update frekuensi, toggle, delete). `handlers/schedule_runner_test.go` (baru): jadwal aktif tanpa teknisi → log `scheduled`; tidak ada jadwal → skip; duplikat hari yang sama → skip; jadwal berteknisi → log `ongoing` + notifikasi.
- Swagger docs di-regenerate (`swag init`).
timestamp: 2026-08-06 10:50 WITA
kategori: Sedang, Backend
file yang diubah:
- backend/models/maintenance_schedule.go
- backend/models/setup.go
- backend/handlers/maintenance_schedule.go
- backend/handlers/schedule_runner.go
- backend/routes/maintenance_schedule.go
- backend/main.go
- backend/handlers/maintenance_schedule_test.go
- backend/handlers/schedule_runner_test.go
- backend/docs/docs.go
- backend/docs/swagger.json
- backend/docs/swagger.yaml
- backend/changelog.md

## [0.29.0] - 2026-08-05

### Fitur: Re-assign Tugas oleh Admin — Task 5.3

perubahan yang dilakukan: **`PUT /api/maintenance-logs/:id/assign` diperluas untuk menerima log berstatus `ongoing` (re-assign) selain `scheduled`. Saat re-assign teknisi lama dilepas dan teknisi baru ditugaskan dengan status tetap `ongoing`; assign ke teknisi yang sama ditolak 400 "Teknisi sudah ditugaskan pada tugas ini". Audit log mencatat action baru `REASSIGN_MAINTENANCE` dengan deskripsi "Tugas [ID] dialihkan dari [nama lama] ke [nama baru]". Notifikasi in-app dua pihak: `MAINTENANCE_ASSIGNED` ke teknisi baru ("Anda mendapat tugas pemeliharaan baru untuk CCTV [nama]") dan `MAINTENANCE_UNASSIGNED` ke teknisi lama ("Tugas pemeliharaan untuk CCTV [nama] telah dialihkan ke teknisi lain"). Pesan notifikasi assign pertama juga diseragamkan memakai nama CCTV**.
before: Endpoint assign hanya menerima log berstatus `scheduled`; log `ongoing` ditolak 400 sehingga admin tidak bisa mengalihkan tugas yang sedang dikerjakan. Tidak ada audit khusus atau notifikasi ke teknisi lama.
after:
- `handlers/maintenance.go` (`AssignMaintenanceLog`): guard status diperluas ke `scheduled`/`ongoing` (selain itu 400); variabel `isReassign`; saat re-assign muat teknisi lama, tolak bila `*TechnicianID == techUUID` (400 khusus), dan teknisi lama/baru dipakai untuk audit & notifikasi; audit memakai action `REASSIGN_MAINTENANCE` saat re-assign; nama CCTV diambil untuk pesan notifikasi (fallback `action_taken` bila CCTV tak ditemukan). Anotasi Swagger diperbarui.
- `handlers/maintenance_test.go`: test baru — re-assign log `ongoing` ke teknisi baru → 200 (status tetap ongoing), assign ke teknisi yang sama → 400, assign log `completed` → 400, re-assign oleh role `teknisi` (lewat `RequireRole("admin")`) → 403; `TestAssignMaintenanceLog_Success` diperbarui (query nama CCTV) & `TestAssignMaintenanceLog_NotScheduled` diganti `TestAssignMaintenanceLog_CompletedRejected`.
- Swagger docs di-regenerate (`swag init`).
timestamp: 2026-08-05 20:33 WITA
kategori: Sedang, Backend
file yang diubah:
- backend/handlers/maintenance.go
- backend/handlers/maintenance_test.go
- backend/docs/docs.go
- backend/docs/swagger.json
- backend/docs/swagger.yaml
- backend/changelog.md

## [0.28.3] - 2026-08-05

### Perubahan: Penggabungan Status "rusak" → "nonaktif"

perubahan yang dilakukan: **Status `rusak` digabung ke `nonaktif` karena secara fungsional identik di SIGAP (keduanya dilindungi dari sinkronisasi otomatis dan ditampilkan sama sebagai "Nonaktif" di antarmuka). Enum status resmi kini `aktif` / `pemeliharaan` / `nonaktif`: `normalizeCCTVStatus` memetakan `rusak`/`broken` → `nonaktif` (berlaku otomatis untuk create, update, dan impor Excel), validasi `oneof` dan switch validasi impor hanya menerima tiga status resmi, dan guard konservatif sinkronisasi disederhanakan menjadi `current == "nonaktif"`**
before: Tiga status dilindungi sinkronisasi (`rusak` + `nonaktif`) dengan enum 4 nilai — memaksa frontend & dokumentasi menyediakan opsi "Rusak" yang perilakunya identik dengan "Nonaktif" (dua status = satu makna).
after:
- `handlers/cctv.go`: `normalizeCCTVStatus` — `case "rusak", "broken": return "nonaktif"` (sejak 2026-08-05); tag `oneof` pada `CreateCCTVRequest`/`UpdateCCTVRequest` → `aktif pemeliharaan nonaktif`; switch validasi impor → `case "aktif", "pemeliharaan", "nonaktif"` dengan pesan error "Gunakan: aktif, pemeliharaan, atau nonaktif".
- `handlers/cctv_sync.go`: guard konservatif `if current == "rusak" || current == "nonaktif"` → `if current == "nonaktif"` (nilai `rusak` sudah ternormalisasi menjadi `nonaktif` sebelumnya); komentar aturan transisi diperbarui.
- `handlers/cctv_test.go`: `TestNormalizeCCTVStatus` — kasus `rusak`/`broken` kini mengharapkan `nonaktif`; `TestUpdateCCTV_Success` — input `{"status":"rusak"}` disimpan sebagai `nonaktif` (bukti pemetaan pada jalur update).
- `handlers/dashboard_test.go`: agregat status dashboard `AddRow("rusak", ...)` → `AddRow("nonaktif", ...)`.
- Anotasi Swagger `field_maintenance.go` & `maintenance.go` diperbarui ("status nonaktif tidak pernah ditimpa otomatis"); docs di-regenerate (`swag init`).
timestamp: 2026-08-05 WITA
kategori: Kecil, Backend
file yang diubah:
- backend/handlers/cctv.go
- backend/handlers/cctv_sync.go
- backend/handlers/cctv_test.go
- backend/handlers/dashboard_test.go
- backend/handlers/field_maintenance.go
- backend/handlers/maintenance.go
- backend/docs/*
- backend/changelog.md



## [0.28.2] - 2026-08-05

### Perbaikan: Guard Konservatif Sinkronisasi Status Berlaku untuk Baris Legacy (huruf besar/Inggris)

perubahan yang dilakukan: **Guard "rusak/nonaktif tidak pernah ditimpa" di `syncCCTVStatus` membandingkan string secara persis terhadap huruf kecil Indonesia, sehingga baris legacy berstatus huruf besar Inggris (`'INACTIVE'`, `'RUSAK'`, `'BROKEN'`, ...) lolos guard dan bisa diubah otomatis — terbukti di audit log uji manual: `Status CCTV diubah otomatis dari 'INACTIVE' ke 'aktif'`. Kini status saat ini dinormalisasi (`normalizeCCTVStatus`) sebelum guard, jadi baris legacy tetap dilindungi**
before: `if cctv.Status == "rusak" || cctv.Status == "nonaktif" { return nil }` — nilai `'INACTIVE'`/`'RUSAK'` (huruf besar dari data lama) tidak cocok dengan perbandingan literal → sinkronisasi otomatis menimpanya (mis. `'INACTIVE'` → `'aktif'`).
after:
- `handlers/cctv_sync.go`: status saat ini dinormalisasi dulu (`current := normalizeCCTVStatus(cctv.Status)`), lalu guard membandingkan `current`; perbandingan "sudah sesuai target" juga memakai `current` sehingga baris legacy `'ACTIVE'` tidak ditulis ulang percuma saat target `aktif`.
- `handlers/maintenance_test.go`: test baru `TestCompleteMaintenanceLog_SyncCCTV_LegacyUppercaseUntouched` — CCTV legacy `'INACTIVE'` + complete 9/9 ok → tidak ada aksi UPDATE status (hanya baca).
timestamp: 2026-08-05 WITA
kategori: Kecil, Backend
file yang diubah:
- backend/handlers/cctv_sync.go
- backend/handlers/maintenance_test.go
- backend/changelog.md



## [0.28.1] - 2026-08-05

### Perbaikan: Normalisasi Status CCTV untuk Nilai Bahasa Inggris (ACTIVE/MAINTENANCE/INACTIVE)

perubahan yang dilakukan: **`normalizeCCTVStatus` tidak menormalisasi padanan bahasa Inggris dari dropdown antarmuka (`ACTIVE`/`INACTIVE`/`MAINTENANCE` — padahal komentar fungsi sudah menjanjikannya), sehingga validasi `oneof=aktif rusak pemeliharaan nonaktif` selalu menolak simpan/ubah CCTV via form UI dengan pesan "Validasi input gagal". Kini alias bahasa Inggris ditambahkan: `active`→`aktif`, `broken`→`rusak`, `maintenance`→`pemeliharaan`, `inactive`→`nonaktif` (tetap case-insensitive, spasi di sekitar ditrim). Berlaku untuk create, update, dan impor Excel karena ketiganya lewat fungsi yang sama**
before: `normalizeCCTVStatus` hanya mengenal kata Indonesia; "ACTIVE"/"MAINTENANCE"/"INACTIVE" dari dropdown form gagal validasi enum → 400 "Validasi input gagal" (kode error: `Status harus berupa salah satu dari: aktif, rusak, pemeliharaan, nonaktif`).
after:
- `handlers/cctv.go` (`normalizeCCTVStatus`): switch diperluas dengan alias Inggris (`active`, `broken`, `maintenance`, `inactive`) selain kata Indonesia yang sudah ada; nilai tak dikenal tetap dibiarkan agar validasi `oneof` menolaknya.
- `handlers/cctv_test.go`: test baru `TestNormalizeCCTVStatus` (tabel 15 kasus: Indonesia, Inggris, huruf besar UI, campuran, spasi, nilai tak dikenal, string kosong).
timestamp: 2026-08-05 WITA
kategori: Kecil, Backend
file yang diubah:
- backend/handlers/cctv.go
- backend/handlers/cctv_test.go
- backend/changelog.md



## [0.28.0] - 2026-08-05

### Fitur: Sinkronisasi Status CCTV Otomatis (Konservatif) — Task 5.2

perubahan yang dilakukan: **Menambahkan helper `syncCCTVStatus` yang menyinkronkan status CCTV secara otomatis dan konservatif di dalam transaksi yang sama dengan complete/field-check. Hanya transisi `aktif ↔ pemeliharaan` yang dilakukan sistem; status `rusak`/`nonaktif` tidak pernah ditimpa otomatis. Complete dengan 9/9 checklist ok → `aktif`, complete dengan item tidak ok → `pemeliharaan`, field-check `attention` → `pemeliharaan`, field-check `ok` → `aktif`. Setiap perubahan mencatat audit log `UPDATE_CCTV` (keterangan otomatis) dan menyiarkan event WebSocket `CCTV_UPDATED` agar tabel & peta sebaran ter-refresh real-time**
before: Status CCTV hanya bisa diubah manual oleh admin lewat halaman master CCTV; hasil complete/field-check tidak memengaruhi status perangkat sehingga CCTV yang bermasalah tetap tampil "aktif".
after:
- `handlers/cctv_sync.go` (baru): helper `syncCCTVStatus(tx, c, cctvID, newStatus, reason)` — baca status saat ini dalam transaksi `tx`; skip bila status `rusak`/`nonaktif` atau sudah sama dengan target; update status → `newStatus`; catat audit `UPDATE_CCTV` + broadcast `CCTV_UPDATED`.
- `handlers/maintenance.go` (`CompleteMaintenanceLog`): dalam transaksi yang sama — hasil `completed` → `syncCCTVStatus(..., "aktif", "semua checklist pemeliharaan berhasil")`; hasil `failed` → `syncCCTVStatus(..., "pemeliharaan", "tugas pemeliharaan gagal — ada item checklist tidak OK")`.
- `handlers/field_maintenance.go` (`CreateFieldMaintenance`): dalam transaksi yang sama — hasil `attention` → `syncCCTVStatus(..., "pemeliharaan", "hasil pengecekan lapangan memerlukan perhatian")`; hasil `ok` → `syncCCTVStatus(..., "aktif", "pengecekan lapangan menunjukkan semua item OK")`.
- `handlers/cctv_test.go`: tambah `cctvMockRowWithStatus` (varian status untuk simulasi sqlmock).
- `handlers/maintenance_test.go`: test baru — complete 9/9 ok → CCTV `aktif`; complete gagal → CCTV `pemeliharaan`; complete pada CCTV `rusak`/`nonaktif` → status tidak berubah.
- `handlers/field_maintenance_test.go`: test baru — field-check `attention` → CCTV `pemeliharaan`; field-check `ok` → CCTV `aktif`.
- Swagger docs di-regenerate (`swag init`).
timestamp: 2026-08-05 WITA
kategori: Sedang, Backend
file yang diubah:
- backend/handlers/cctv_sync.go (baru)
- backend/handlers/maintenance.go
- backend/handlers/field_maintenance.go
- backend/handlers/cctv_test.go
- backend/handlers/maintenance_test.go
- backend/handlers/field_maintenance_test.go
- backend/docs/ (swag init)
- backend/changelog.md



## [0.27.0] - 2026-08-04

### Fitur: Foto Profil Pengguna (upload, simpan, dan hapus)

perubahan yang dilakukan: **Menambahkan kolom `profile_photo_url` pada model `User` + dukungan pengaturan foto profil pada `PUT /api/auth/profile` (upload file memakai endpoint `/api/upload` yang sudah ada, path disimpan di database, file disajikan via `GET /api/uploads/:filename` yang dilindungi JWT). File foto profil tidak lagi ikut dihapus oleh pembersihan upload yatim; `GET /api/auth/me` dan `PUT /api/auth/profile` kini mengembalikan `profile_photo_url`**
before: `User` tidak memiliki field foto profil; `PUT /api/auth/profile` hanya menerima `full_name`; `CleanupOrphanUploads` hanya menganggap file yang dirujuk `maintenance_logs.image_url` & `field_maintenance_images.image_url` sebagai referensi sah (file foto profil yang baru diupload akan dihapus setelah 24 jam).
after:
- `models/setup.go` (`User`): field baru `ProfilePhotoURL` (`varchar(255)`, JSON `profile_photo_url`) — kolom ditambahkan otomatis oleh AutoMigrate saat API restart.
- `handlers/auth.go` (`UpdateMyProfileRequest`): field opsional `profile_photo_url` — `nil` = biarkan foto saat ini, `""` = hapus foto, selain itu harus path upload aman (`/api/uploads/<uuid>.jpg`, divalidasi `isSafeUploadPath` anti path-traversal).
- `handlers/auth.go` (`GetMyProfile` & `UpdateMyProfile`): respons kini menyertakan `profile_photo_url`.
- `handlers/upload.go` (`CleanupOrphanUploads`): referensi sah ditambah `users.profile_photo_url` agar foto profil tidak terhapus.
- Swagger docs di-regenerate (`swag init`).
timestamp: 2026-08-04 WITA
kategori: Sedang, Backend
file yang diubah:
- backend/models/setup.go
- backend/handlers/auth.go
- backend/handlers/upload.go
- backend/docs/ (swag init)
- backend/changelog.md



## [0.26.2] - 2026-08-04

### Bugfix: Respons `GET /api/audit-logs` Tidak Sesuai Konvensi List SIGAP

perubahan yang dilakukan: **Respons list audit log diubah dari `data: [array]` + `pagination` top-level menjadi `data: { rows, pagination }` — menyelaraskan dengan konvensi mayoritas handler list lain (`/cctv`, `/maintenance-logs`, `/users`, `/field-maintenance`, `/notifications`). Halaman audit-logs frontend membaca `data.rows`, sehingga sebelum perbaikan ini daftar selalu kosong ("Belum ada data audit log") meskipun log tersimpan di database**
before: `GET /api/audit-logs` mengembalikan `{ status, message, data: [...], pagination }` — tidak cocok dengan pembacaan frontend `response.data.rows`.
after:
- `handlers/audit.go` (GetAuditLogs): `data` kini objek `{ rows, pagination }`.
- `handlers/audit_test.go`: asersi disesuaikan (data → `data.rows` + `data.pagination`).
- Endpoint ekspor CSV tidak berubah.
timestamp: 2026-08-04 WITA
kategori: Bugfix, Backend
file yang diubah:
- backend/handlers/audit.go
- backend/handlers/audit_test.go
- backend/changelog.md



## [0.26.1] - 2026-08-04

### Bugfix: Respons `GET /api/notifications` Tidak Sesuai Konvensi List SIGAP

perubahan yang dilakukan: **Respons list notifikasi diubah dari `data: [array]` + `pagination` top-level menjadi `data: { rows, pagination }` — menyelaraskan dengan konvensi mayoritas handler list lain (`/cctv`, `/maintenance-logs`, `/users`, `/field-maintenance`) sehingga konsumen frontend memakai pola bungkus yang sama**
before: `GET /api/notifications` mengembalikan `{ status, message, data: [...], pagination }` (pola lama `/audit-logs`) — frontend yang mengikuti konvensi `data.rows` membaca daftar kosong sehingga pemberitahuan tidak pernah tampil.
after:
- `handlers/notification.go` (GetNotifications): `data` kini objek `{ rows, pagination }`.
- `handlers/notification_test.go`: asersi disesuaikan (data → `data.rows` + `data.pagination`).
- Endpoint lain (`/unread-count`, `/:id/read`, `/read-all`) tidak berubah.
timestamp: 2026-08-04 WITA
kategori: Bugfix, Backend
file yang diubah:
- backend/handlers/notification.go
- backend/handlers/notification_test.go
- backend/changelog.md



## [0.26.0] - 2026-08-04

### Notifikasi In-App Real-Time (Fase 5.1 — Jalur A)

perubahan yang dilakukan: **Sistem notifikasi in-app pertama: model `Notification`, 4 endpoint `/api/notifications`, perluasan `WSHub` untuk broadcast terarah per-user/per-role, util `NotifyUser`/`NotifyRole` (async + retry), dan hook otomatis pada alur assign/complete/field-check — setiap user melihat notifikasi miliknya saja (bell icon + badge unread), dikirim real-time via WebSocket dengan fallback polling 60 detik di frontend**
before:
- Tidak ada model/endpoint notifikasi; seluruh informasi (penugasan, penyelesaian, hasil pengecekan) hanya tampil di halaman masing-masing tanpa pemberitahuan aktif.
- `WSHub` hanya menyimpan `map[*websocket.Conn]bool`; `BroadcastEvent` global — tidak mungkin mengirim pesan ke user/role tertentu.
- WebSocket `/api/ws` tidak menyimpan identitas pengguna di balik koneksi.
after:
- `models/notification.go`: `Notification{UserID, Type, Title, Message, ReferenceID, ReferenceType, IsRead}` + 6 konstanta tipe (`MAINTENANCE_ASSIGNED`, `MAINTENANCE_REASSIGNED`, `MAINTENANCE_UNASSIGNED`, `MAINTENANCE_COMPLETED`, `MAINTENANCE_OVERDUE`, `FIELD_CHECK_ATTENTION`); index komposit `idx_notif_user_read (user_id, is_read)`; didaftarkan ke `AutoMigrate` (main.go).
- Endpoint baru (semua butuh autentikasi, scope per-user dari klaim JWT): `GET /api/notifications` (paginasi + filter `unread`), `GET /api/notifications/unread-count`, `PUT /api/notifications/:id/read` (ownership check → 403), `PUT /api/notifications/read-all`. Rute statis didaftarkan sebelum dinamis (gotcha Fiber).
- `utils/websocket.go`: `WSHubStore.clients` kini `map[*websocket.Conn]WSClientInfo{UserID, Role}`; `Register(conn, userID, role)`; `BroadcastToUser` & `BroadcastToRole` (guard clause tanpa client); `BroadcastEvent` global tetap (backward-compatible). `routes/websocket.go` meneruskan identitas JWT via `c.Locals("fiberctx")`.
- `utils/notify.go`: `NotifyUser` (simpan async + `BroadcastToUser` event `NOTIFICATION_CREATED`) & `NotifyRole` (notif per user berperan + `BroadcastToRole`); retry 3x @100ms (pola `LogAudit`).
- Hook: `AssignMaintenanceLog` → notif `MAINTENANCE_ASSIGNED` ke teknisi; `CompleteMaintenanceLog` → notif `MAINTENANCE_COMPLETED` ke seluruh admin (label "Selesai"/"Perlu Tindak Lanjut"); `CreateFieldMaintenance` hasil `attention` → notif `FIELD_CHECK_ATTENTION` ke admin.
- Test baru: `handlers/notification_test.go` (11 kasus: list scope, filter unread, unread-count, read ownership 403, read-all, UUID invalid 400, tanpa identitas 401) & `utils/notify_test.go` (persist sukses/retry/gagal, guard clause broadcast tanpa client).
- `swag init` — dokumentasi endpoint Notifikasi di Swagger.
timestamp: 2026-08-04 WITA
kategori: Besar, Backend
file yang diubah:
- backend/models/notification.go (baru)
- backend/utils/websocket.go
- backend/utils/notify.go (baru)
- backend/utils/notify_test.go (baru)
- backend/routes/websocket.go
- backend/routes/notification.go (baru)
- backend/handlers/notification.go (baru)
- backend/handlers/notification_test.go (baru)
- backend/handlers/maintenance.go
- backend/handlers/field_maintenance.go
- backend/main.go
- backend/docs/* (swag init)
- backend/changelog.md



## [0.25.2] - 2026-08-03

### Label Status "Gagal" Menjadi "Perlu Tindak Lanjut" (Hanya Tampilan, Nilai Internal Tetap `failed`)

perubahan yang dilakukan: **Label tampilan status `failed` pada laporan PDF diubah dari "Gagal" menjadi "Perlu Tindak Lanjut" — hanya perubahan teks tampilan, nilai enum `failed` di database/API tetap sama sehingga tidak diperlukan migrasi data dan seluruh klien API tidak terpengaruh**
before: `pdfStatusLabel("failed")` mengembalikan "Gagal"; ringkasan laporan PDF menampilkan "Gagal: N".
after:
- `handlers/report.go`: `pdfStatusLabel` mengembalikan "Perlu Tindak Lanjut"; key ringkasan PDF & baris statistik memakai "Perlu Tindak Lanjut".
- CSV export tetap menulis nilai mentah `failed` (tidak diubah).
timestamp: 2026-08-03 WITA
kategori: Kecil, Backend
file yang diubah:
- backend/handlers/report.go
- backend/changelog.md
- frontend/changelog.md
- docs/changelog.md

## [0.25.1] - 2026-08-03

### Bugfix: Cookie Sesi Duplikat (Stale di Path /api) Membuat Login "Langasung Ter-logout"

perubahan yang dilakukan: **`ExtractTokenString` dan middleware CSRF kini memakai nilai cookie TERAKHIR (path terpendek "/"), bukan nilai pertama — browser yang masih menyimpan cookie `sigap_session`/`sigap_csrf` stale warisan versi lama di path `/api` atau `/api/` (token sudah revoked) akan mengirim dua cookie sekaligus, dan nilai stale (path terpanjang, dikirim lebih dulu) yang selama ini dipilih membuat setiap request dianggap sesi dicabut (401) — termasuk `/auth/me` tepat setelah login sukses → user dilempar kembali ke halaman login ("percobaan login langsung ter-logout")**
before:
- `utils.ExtractTokenString` dan `middleware/csrf.go` memakai `c.Cookies(nama)` yang mengembalikan nilai cookie PERTAMA di header `Cookie`. Sesuai RFC 6265 §5.4 browser mengirim cookie ber-Path terpanjang lebih dulu, sehingga cookie stale `sigap_session` di `/api` (token yang sudah direvoke saat logout) selalu menang atas cookie sesi aktif di `/` → `Protected`/WebSocket menjawab 401 "Token telah dicabut", padahal sesi baru valid.
- `frontend/src/lib/api.ts` `getCookie()` juga mengambil kecocokan PERTAMA, sehingga header `X-CSRF-Token` bisa memakai token CSRF stale (path `/api`) sementara server membandingkannya dengan cookie aktif → mutasi berisiko ditolak 403.
after:
- `utils/csrf.go`: helper baru `ExtractCookieValues` (seluruh nilai cookie sesuai urutan header) & `ExtractLastCookieValue` (nilai terakhir); `ExtractTokenString` memakai nilai cookie sesi TERAKHIR — sesi aktif di "/" selalu menang atas stale di "/api" — berlaku otomatis untuk `Protected`, WebSocket `/api/ws`, dan `Logout` (tetap mencabut token yang benar).
- `middleware/csrf.go`: pengecekan keberadaan sesi & perbandingan token CSRF memakai `ExtractLastCookieValue` — konsisten dengan sesi aktif.
- `frontend/src/lib/api.ts`: `getCookie()` mengambil kemunculan TERAKHIR (cookie aktif di "/") agar header CSRF selalu sinkron dengan nilai yang dibandingkan server.
- Cleanup cookie legacy (`ClearSessionCookies` pada login/logout) tetap dipertahankan sebagai lapisan pertahanan; dengan perbaikan ini cookie stale yang gagal dihapus browser tidak lagi merusak sesi.
- Test baru: `TestExtractTokenString_DuplicateCookiesPrefersLast`, `TestExtractTokenString_NoSessionCookieUsesBearer` (utils/csrf_test.go), `TestCSRF_DuplicateCsrfCookiePrefersLast`, `TestCSRF_DuplicateCsrfCookieHeaderMatchesStaleOnlyForbidden` (middleware/csrf_test.go).
timestamp: 2026-08-03 WITA
kategori: Bugfix, Backend
file yang diubah:
- backend/utils/csrf.go
- backend/utils/csrf_test.go
- backend/middleware/csrf.go
- backend/middleware/csrf_test.go
- frontend/src/lib/api.ts
- backend/changelog.md
- frontend/changelog.md
- docs/changelog.md

## [0.25.0] - 2026-08-03

### Alur Penugasan Pemeliharaan: Assign Teknisi & Penyelesaian Checklist 9 Tahap

perubahan yang dilakukan: **Endpoint baru `PUT /maintenance-logs/:id/assign` (admin menugaskan log terjadwal ke teknisi) dan `POST /maintenance-logs/:id/complete` (teknisi menyelesaikan tugas lewat checklist 9 tahap), status tidak lagi bisa diubah manual — dikelola sistem mengikuti alur: dibuat → scheduled → ditugaskan → ongoing → selesai (completed 9/9 ok) / gagal (failed)**
before:
- `MaintenanceLog.TechnicianID` wajib terisi (not null) saat pembuatan; `POST /maintenance-logs` menerima status bebas (scheduled/ongoing/completed/failed) dan `PUT /:id` memungkinkan status diubah manual oleh siapa pun yang berhak — tidak ada alur penugasan ke teknisi maupun pelaporan penyelesaian checklist.
- Teknisi hanya melihat log miliknya saat ekspor/CSV; halaman list (`GET /maintenance-logs`) menampilkan seluruh data untuk semua peran.
- Log yang belum ditugaskan tidak ada; `technician_id`/`technician` selalu objek penuh dalam JSON.
after:
- Model `MaintenanceLog` (models/setup.go): `TechnicianID` menjadi opsional (`*uuid.UUID`) — log baru dibuat tanpa penugasan; relasi `User` menjadi pointer (`*User`) sehingga `"technician": null` saat belum ditugaskan; kolom baru `MaintenanceResultID *uuid.UUID` menautkan hasil pengecekan ke tugas.
- Model `FieldMaintenance` (models/field_maintenance.go): kolom baru `MaintenanceLogID *uuid.UUID` (nil saat catatan spontan dari halaman field-maintenance, terisi saat berasal dari penyelesaian tugas).
- `POST /maintenance-logs` (CreateMaintenanceLog): status hanya boleh "scheduled" (`oneof=scheduled`), `TechnicianID` selalu nil, `completed_at` tidak lagi diterima dari body; ID teknisi tidak lagi diambil dari JWT saat pembuatan.
- `PUT /maintenance-logs/:id` (UpdateMaintenanceLog): field `status` dan `completed_at` dihapus dari request — status hanya berubah lewat assign/complete; ownership check diperbaiki agar aman saat `TechnicianID` nil.
- Handler baru `AssignMaintenanceLog` (rute `PUT /:id/assign`, admin saja): hanya log berstatus `scheduled` yang bisa ditugaskan; user tujuan wajib berperan `teknisi`; status otomatis `ongoing`; audit `ASSIGN_MAINTENANCE`.
- Handler baru `CompleteMaintenanceLog` (rute `POST /:id/complete`): teknisi wajib menyelesaikan tugasnya sendiri (ownership check; tanpa mekanisme penolakan), hanya tugas `ongoing`; body berupa 1–30 item checklist (name min 3, result ok/not_ok, note maks 500) + 1–5 foto (`/api/uploads/...`); status akhir `completed` hanya bila 9/9 item ok, selain itu `failed`; seluruh header+checks+images+update status disimpan dalam satu transaksi; `completed_at` diisi otomatis; audit `COMPLETE_MAINTENANCE`; broadcast `MAINTENANCE_UPDATED`.
- `GET /maintenance-logs` & `GET /maintenance-logs/:id` kini menerapkan scope peran (teknisi hanya melihat/membuka log miliknya) + ownership check pada detail — konsisten dengan ekspor.
- `GET /users` menerima filter query `role` (mis. `?role=teknisi` untuk dropdown penugasan).
- `GET /dashboard/tasks` menambahkan `cctv_id` & `cctv_kode` pada setiap item tugas (dibutuhkan frontend untuk membuka wizard checklist).
- Ekspor CSV & laporan PDF aman saat teknisi belum ditugaskan (nil-safe, kolom teknisi kosong).
- `swag init` dijalankan ulang; test baru: assign sukses/tidak terjadwal/bukan teknisi/UUID invalid; complete sukses (completed 9/9), tidak semua ok (failed), ownership denied, tidak ongoing.
timestamp: 2026-08-03 WITA
kategori: Feature, Backend
file yang diubah:
- backend/models/setup.go
- backend/models/field_maintenance.go
- backend/handlers/maintenance.go
- backend/routes/maintenance.go
- backend/handlers/user.go
- backend/handlers/dashboard.go
- backend/handlers/report.go
- backend/handlers/maintenance_test.go
- backend/docs/* (regenerasi swag)
- backend/changelog.md
verifikasi:
- `go build ./...` PASS.
- `go vet ./...` PASS.
- `go test ./... -count=1` PASS (semua package ok).
alasan teknis: Alur bisnis menuntut status dikelola sistem (penugasan + penyelesaian), bukan diubah manual, agar riwayat terjaga dan teknisi tidak bisa menghindari tugas; checklist 9 tahap dipakai ulang dari modul field-maintenance sehingga hasil pengecekan tugas terdokumentasi lengkap dengan foto; scope list/detail mencegah teknisi melihat data pemeliharaan milik teknisi lain.

---

## [0.24.0] - 2026-08-03

### API Pengecekan Lapangan (Field Maintenance) untuk Teknisi

perubahan yang dilakukan: **Endpoint baru untuk mencatat hasil pengecekan lapangan spontan teknisi (checklist per item + foto bukti), serta penyesuaian pembersihan file upload agar foto pengecekan tidak terhapus**
before:
- Tidak ada cara mencatat pengecekan lapangan selain `maintenance-logs` (jadwal dari admin); tidak ada struktur checklist per item maupun lampiran banyak foto.
- `CleanupOrphanUploads` hanya menganggap `maintenance_logs.image_url` sebagai referensi sah, sehingga foto pengecekan lapangan akan dihapus sebagai file yatim setelah 24 jam.
after:
- Model baru di `models/field_maintenance.go`: `FieldMaintenance` (header: cctv_id, technician_id, overall_status ok/attention, notes, completed_at) + `FieldMaintenanceCheck` (name, result ok/not_ok, note) + `FieldMaintenanceImage` (image_url); semua terdaftar di `AutoMigrate` main.go.
- Handler baru `handlers/field_maintenance.go`:
  - `POST /api/field-maintenance` — wajib: cctv_id valid, 1–30 item checklist (name min 3 karakter, result ok/not_ok, note maks 500), minimal 1 maksimal 5 foto (`image_url` wajib prefix `/api/uploads/`). Teknisi diambil dari token JWT; `overall_status` dihitung otomatis (ada item not_ok → attention); seluruh header+checks+images disimpan dalam satu transaksi; audit log `CREATE_FIELD_MAINTENANCE`.
  - `GET /api/field-maintenance` — terpaginasi; admin melihat semua, teknisi hanya miliknya (scope `technician_id`); filter cctv_id, status, rentang tanggal, pencarian (notes/nama CCTV/kode); preload CCTV, teknisi, checklist, foto.
  - `GET /api/field-maintenance/:id` — detail lengkap dengan ownership check untuk teknisi.
  - `DELETE /api/field-maintenance/:id` — admin saja; audit `DELETE_FIELD_MAINTENANCE`.
- Rute baru `routes/field_maintenance.go` (group `/field-maintenance` dilindungi `Protected()`).
- `CleanupOrphanUploads` kini juga membaca `field_maintenance_images.image_url` sebagai referensi sah agar foto bukti pengecekan tidak terhapus.
- Unit test baru `handlers/field_maintenance_test.go` (validasi, CCTV tidak ditemukan, sukses bertransaksi, scope teknisi, UUID tidak valid, not found).
timestamp: 2026-08-03 WITA
kategori: Feature, Backend
file yang diubah:
- backend/models/field_maintenance.go (baru)
- backend/handlers/field_maintenance.go (baru)
- backend/routes/field_maintenance.go (baru)
- backend/handlers/field_maintenance_test.go (baru)
- backend/main.go (AutoMigrate + registrasi rute)
- backend/handlers/upload.go (cleanup menyertakan field_maintenance_images)
- backend/changelog.md
verifikasi:
- `go build ./...` PASS.
- `go vet ./...` PASS.
- `go test ./... -count=1` PASS (semua package ok).
alasan teknis: Pengecekan spontan di lapangan membutuhkan struktur data terpisah dari jadwal pemeliharaan; checklist per item memungkinkan pelaporan hasil yang detail dan dapat difilter admin; penyimpanan transaksional menjaga konsistensi header-checklist-foto; referensi foto tambahan pada cleanup mencegah file bukti terhapus.

---

## [0.23.0] - 2026-08-03

### Kop Surat & Perbaikan Tata Letak Laporan PDF Pemeliharaan

perubahan yang dilakukan: **PDF laporan pemeliharaan kini berkop surat resmi (logo IKN + identitas instansi) dan tabel baris tidak lagi timpang/ketimpa**
before:
- Kop laporan hanya berupa 2 baris teks (judul + nama sistem) tanpa logo maupun alamat instansi.
- Baris tabel memakai perkiraan jumlah baris lipat teks "Tindakan" berbasis hitungan karakter (44 karakter/baris) yang bisa meleset, sehingga teks menimpa garis/baris berikutnya; hanya ada garis bawah (tanpa border sel) sehingga tabel tampak tidak simetris; pindah halaman hanya dicek dengan ambang kasar `GetY() > 260` sehingga sel bisa terpotong di tengah saat pindah halaman.
after:
- Kop surat resmi: logo IKN (`assets/nusantara-logo.png`, 691×996 px) di kiri, nama instansi "OTORITA IBU KOTA NUSANTARA" + sub "Sistem Informasi Gangguan & Aset Pusat (SIGAP)", alamat lengkap (Kompleks Perkantoran Pemerintah IKN, Kab. Penajam Paser Utara, Provinsi Kalimantan Timur), website www.ikn.go.id (telp/email sebagai konstanta kosong yang siap diisi), ditutup garis ganda (tebal + tipis).
- Judul laporan diubah "LAPORAN PEMELIHARAAN CCTV"; baris "Sistem Informasi Gangguan & Aset Pusat (SIGAP) IKN" yang lama dihapus karena sudah ada di kop.
- Perhitungan tinggi baris tabel kini akurat memakai `pdf.SplitLines` (pembungkus teks engine yang sama dengan MultiCell), sehingga teks multi-baris tidak lagi menimpa baris berikutnya.
- Setiap sel digambar border grid penuh (`Rect`) sehingga tabel simetris dan mudah dibaca.
- Pindah halaman dilakukan per-baris dengan cek `yStart+cellHeight > pdfBreakY` sebelum menggambar, mencegah sel terpotong di tengah (aksi pindah halaman lama di loop dihapus).
- Logo dibuat opsional (`os.Stat`): jika berkas tidak ditemukan (mis. lingkungan unit test) kop tetap terbit tanpa logo tanpa membuat PDF gagal.
- Dockerfile runner kini menyalin folder `backend/assets` ke `/app/assets` agar logo tersedia saat runtime.
timestamp: 2026-08-03 WITA
kategori: Feature, Bugfix, UI, Backend
file yang diubah:
- backend/handlers/report.go
- backend/Dockerfile
- backend/assets/nusantara-logo.png (baru, disalin dari frontend/public)
- backend/changelog.md
verifikasi:
- `go build ./...` PASS.
- `go vet ./handlers/` PASS.
- `go test ./... -count=1` PASS (semua package ok).
alasan teknis: FPDF `MultiCell` membungkus teks berdasarkan lebar karakter, bukan hitungan rune, sehingga estimasi baris lama tidak akurat → tumpang tindih. `SplitLines` memakai pengukuran yang sama sehingga konsisten; border grid + cek pindah halaman per-baris menjamin simetri dan tidak ada sel terpotong. Logo dimuat dari file lokal agar hasil cetak konsisten tanpa permintaan jaringan.

---

## [0.22.0] - 2026-08-03

### Penghapusan Kolom Legacy asset_id pada maintenance_logs

perubahan yang dilakukan: **Migrasi kini menghapus kolom legacy `asset_id` yang menyebabkan INSERT gagal**
before:
- `maintenance_logs` masih memiliki kolom `asset_id uuid NOT NULL` + FK `fk_maintenance_logs_asset`
  dari era model `assets`, sementara model GORM `MaintenanceLog` sudah memakai `cctv_id`.
- INSERT baru tidak pernah mengisi `asset_id`, sehingga setiap penyimpanan log pemeliharaan gagal
  dengan `null value in column "asset_id" violates not-null constraint (SQLSTATE 23502)` → HTTP 500.
- `MigrateMaintenanceToCCTV` hanya memetakan data lama `asset_id → cctv_id` dan tidak menghapus
  kolom legacy; early-return `len(refs) == 0` juga membuat kolom tetap tertinggal walau tidak ada
  data tersisa.
after:
- Migrasi memetakan baris `cctv_id IS NULL` lalu **menghapus kolom `asset_id`** beserta FK-nya
  (`DROP COLUMN` di PostgreSQL ikut menurunkan constraint dependent).
- Penghapusan dilakukan di luar early-return sehingga selalu dieksekusi selama kolom masih ada;
  tetap idempotent (kolom sudah hilang → return awal).
- Tabel kini hanya memakai `cctv_id` sebagai referensi aset; INSERT log baru berjalan normal.
timestamp: 2026-08-03 WITA
kategori: Bugfix, Database, Migration
file yang diubah:
- backend/config/migrate.go
- backend/changelog.md
verifikasi:
- `go test ./... -count=1` PASS.
- `go vet ./...` PASS.
- `gofmt` & `git diff --check` bersih.
- Log startup Docker: "kolom legacy asset_id dihapus dari maintenance_logs".
- `\d maintenance_logs` tidak lagi menampilkan kolom `asset_id`/FK `fk_maintenance_logs_asset`.
alasan teknis: Kolom legacy dari model `assets` (yang sudah dihapus, Fase 3.6) masih mengikat
NOT NULL constraint; membersihkannya memulihkan fungsi penyimpanan log pemeliharaan tanpa
mengubah payload API.

---

## [0.21.0] - 2026-08-03

### Cookie Legacy dan JWT Unik

perubahan yang dilakukan: **Membersihkan cookie sesi pada semua path legacy dan menambahkan identitas unik JWT**
before:
- `fiber.Ctx.Cookie()` memakai Set-Cookie keyed by nama sehingga beberapa cookie `sigap_session`
  dengan path berbeda tidak dapat dihapus dalam satu response.
- JWT hanya memiliki `user_id`, `role`, `tv`, dan `exp`; login ulang pada detik yang sama dapat
  menghasilkan string token identik dengan token blacklist.
after:
- Penulisan cookie memakai multi-value `Set-Cookie` untuk path `/`, `/api`, dan `/api/`.
- Login membersihkan cookie legacy sebelum menerbitkan sesi baru; logout membersihkan seluruh path.
- JWT memiliki `jti` UUID dan `iat`, sehingga setiap token selalu unik.
- Test handler/session dan JWT uniqueness ditambahkan.
timestamp: 2026-08-03 WITA
kategori: Bugfix, Auth, Security, Testing
file yang diubah:
- backend/utils/session.go
- backend/utils/jwt.go
- backend/utils/jwt_test.go
- backend/handlers/auth.go
- backend/handlers/auth_test.go
- backend/handlers/auth_extra_test.go
- backend/changelog.md
verifikasi:
- `go test ./... -count=1` PASS.
- `go vet ./...` PASS.
- `swag init` tetap PASS setelah perubahan handler.
alasan teknis: Mencegah browser memilih cookie sesi lama untuk route `/api/*` dan mencegah token
baru ditolak karena kebetulan identik dengan token yang baru direvoke.

---

## [0.20.0] - 2026-08-03

### Logout Idempotent dan Trusted Proxy Runtime

perubahan yang dilakukan: **Logout tidak lagi gagal karena CSRF stale dan konfigurasi runtime Docker dilengkapi**
before:
- Route logout dilindungi `Protected()` dan CSRF sehingga cookie sesi yang masih ada dengan token CSRF
  stale dapat menghasilkan 403 sebelum cookie dibersihkan.
- Request logout tanpa token menghasilkan 401 dan konfigurasi container aktif belum memuat trusted proxy.
after:
- Route logout menjadi idempotent tanpa `Protected()`: token valid direvoke, token kosong/invalid tetap
  menghapus cookie dan mengembalikan 200.
- CSRF melewati `/api/auth/logout`; route tetap tidak dapat mengakses resource lain dan hanya mencabut sesi.
- Runtime Docker memakai `APP_ENV=production`, `CORS_ALLOWED_ORIGINS=https://localhost`, dan
  `TRUSTED_PROXY_IPS=172.28.0.10`.
- Test route, handler, dan middleware menutup regresi logout/CSRF.
timestamp: 2026-08-03 WITA
kategori: Bugfix, Auth, Security, Infrastructure, Testing
file yang diubah:
- backend/handlers/auth.go
- backend/middleware/csrf.go
- backend/routes/auth.go
- backend/handlers/auth_extra_test.go
- backend/middleware/csrf_test.go
- backend/routes/routes_test.go
- backend/.env.docker (runtime lokal, tidak di-commit)
- backend/.env.docker.example
- backend/docs/docs.go
- backend/docs/swagger.json
- backend/docs/swagger.yaml
- backend/changelog.md
verifikasi:
- `go test ./... -count=1` PASS.
- `go vet ./...` PASS.
- `swag init` PASS.
alasan teknis: Memastikan pengguna dapat logout dan login ulang meskipun cookie CSRF/session sudah
kedaluwarsa, serta mencegah seluruh client di belakang Nginx berbagi bucket rate limit yang sama.

---

## [0.19.0] - 2026-08-03

### Swagger Single-Origin melalui Nginx

perubahan yang dilakukan: **Metadata Swagger diarahkan ke host publik `localhost` dengan scheme HTTPS**
before:
- Metadata Swagger menggunakan `localhost:3000`, sedangkan port API tidak dipublish pada Docker.
- Swagger UI belum memiliki jalur reverse proxy `/swagger/*`.
after:
- Anotasi API menggunakan host `localhost` dan scheme `https`, kemudian file Swagger digenerate ulang.
- Route publik Swagger diteruskan oleh Nginx pada non-production melalui `https://localhost/swagger/index.html`.
- CORS example Docker menggunakan origin single-origin `https://localhost`.
timestamp: 2026-08-03 WITA
kategori: Documentation, Infrastructure, API
file yang diubah:
- backend/main.go
- backend/.env.docker.example
- backend/docs/docs.go
- backend/docs/swagger.json
- backend/docs/swagger.yaml
- nginx/nginx.conf
- backend/changelog.md
alasan teknis: Menyelaraskan dokumentasi interaktif dan origin CORS dengan deployment Nginx tanpa
membuka port API internal ke host.
verifikasi:
- `swag init` berhasil menghasilkan ulang `backend/docs/docs.go`, `swagger.json`, dan `swagger.yaml`.
- `go test ./... -count=1` PASS.
- `go vet ./...` PASS.
- `docker compose config --quiet` PASS dan syntax Nginx PASS.

---

## [0.18.0] - 2026-08-03

### Penyetelan MaxAge Cookie Sesi & CSRF ke 5 Jam

perubahan yang dilakukan: **Cookie sesi (sigap_session) dan CSRF (sigap_csrf) menyetel MaxAge=18000 (5 jam)**
before:
- Cookie sesi dan CSRF diterbitkan tanpa atribut `MaxAge` (session cookie), sehingga pada kondisi tertentu browser menghapus cookie saat tab/halaman di-refresh.
after:
- Cookie `sigap_session` (httpOnly) dan `sigap_csrf` kini menyetel `MaxAge: 18000` (5 jam), selaras dengan masa berlaku token JWT (`exp`). Sesi tetap bertahan utuh saat browser di-refresh.
timestamp: 2026-08-03 WITA
kategori: Bugfix, Auth, Security
file yang diubah:
- backend/utils/session.go
- backend/changelog.md
alasan teknis: Menyelaraskan umur cookie dengan umur token JWT agar cookie tidak hilang secara prematur saat navigasi atau refresh halaman.

---

## [0.17.0] - 2026-08-02

### Perbaikan Cookie Sesi Tidak Terhapus Saat Logout

perubahan yang dilakukan: **Penghapusan cookie sesi/CSRF kini menyetel Path, SameSite, dan Secure yang cocok**
before:
- `ClearSessionCookies` memakai `c.ClearCookie()` yang tidak menyetel atribut `Path`/`SameSite`/`Secure`; browser mencocokkan cookie berdasarkan nama+domain+path sehingga cookie asli (`Path=/`) tidak terhapus.
- Akibatnya setelah logout, cookie sesi masih dikirim saat refresh → `/auth/me` sukses → pengguna kembali ke dashboard.
after:
- `ClearSessionCookies` menuliskan cookie kosong ber-`Expires` masa lalu dan `MaxAge=-1` dengan `Path=/`, `SameSite=Lax`, `Secure` yang sama persis saat pembuatan → cookie sesi & CSRF benar-benar dihapus browser.
- Setelah logout, refresh tetap diarahkan ke halaman login.
timestamp: 2026-08-02 WITA
kategori: Bugfix, Security
file yang diubah:
- backend/utils/session.go
- backend/handlers/auth_extra_test.go
- backend/changelog.md
alasan teknis: Cookie hanya dapat dihapus bila atribut penghapusan identik dengan atribut pembuatan; ketidakcocokan path membuat sesi bocor setelah logout.

## [0.16.0] - 2026-08-02

### Penyederhanaan Sesi: Token 5 Jam, Session Cookie, Tanpa Sliding Session

perubahan yang dilakukan: **Token berlaku 5 jam, cookie sesi non-persisten, hapus perpanjangan otomatis**
before:
- Token JWT berlaku 24 jam; cookie sesi & CSRF bertahan 24 jam (`MaxAge`).
- Middleware `Protected()` memperpanjang otomatis (sliding session) token yang sisa umurnya < 1 jam → rotasi cookie sering terjadi dan refresh tampak memutus sesi.
- Auto-logout idle 15 menit di sisi frontend memutus sesi mendadak.
after:
- Token JWT berlaku **5 jam** (satu hari kerja); tidak ada perpanjangan otomatis.
- Cookie sesi & CSRF menjadi **session cookie** (tanpa `MaxAge`) → otomatis terhapus saat browser ditutup; tidak ada sesi tersimpan permanen.
- Logout tetap me-revoke token (blacklist) & menghapus cookie → token langsung mati, akses dashboard ditolak sampai login ulang.
timestamp: 2026-08-02 WITA
kategori: Security, Refactor, Bugfix
file yang diubah:
- backend/utils/jwt.go
- backend/utils/session.go
- backend/middleware/auth.go
- backend/changelog.md
alasan teknis: Menghilangkan kompleksitas sliding session dan sesi persisten yang menyebabkan logout tak terduga saat refresh, sesuai model autentikasi sederhana yang diminta pengguna.

## [0.15.0] - 2026-08-02

### Perbaikan Login Gagal Akibat Sisa Cookie Sesi

perubahan yang dilakukan: **Endpoint login publik dikecualikan dari proteksi CSRF**
before:
- `CSRFProtection` menolak `POST /api/auth/login` dengan status 403 "Permintaan tidak sah (CSRF token tidak cocok)" bila browser masih membawa cookie `sigap_session` lama dari sesi sebelumnya (mis. token kedaluwarsa namun cookie 24 jam belum dibersihkan).
after:
- `POST /api/auth/login` selalu dilewati CSRF karena bersifat publik (tanpa `Protected()`); sisa cookie sesi tidak lagi memicu pengecekan token CSRF saat login.
- CSRF tetap berlaku penuh untuk seluruh endpoint mutasi terautentikasi lainnya.
timestamp: 2026-08-02 WITA
kategori: Bugfix, Security
file yang diubah:
- backend/middleware/csrf.go
- backend/middleware/csrf_test.go
- backend/changelog.md
alasan teknis: CSRF hanya relevan untuk sesi yang valid & terautentikasi, bukan sekadar keberadaan cookie; login publik tidak memiliki resource terautentikasi yang perlu dilindungi.

## [0.14.0] - 2026-08-02

### Penyelesaian Sisa Coverage Fase 4.1

perubahan yang dilakukan: **Test jalur sukses handler, WebSocket valid, bootstrap route, dan migrasi legacy**
before:
- Jalur sukses update/delete CCTV dan maintenance, handshake WebSocket valid, serta resolusi migration legacy belum diuji.
after:
- Test mencakup jalur sukses CRUD, handshake WebSocket dengan token valid, registrasi route terlindungi, dan resolusi asset legacy melalui ID CCTV.
- Lint `golangci-lint v2.12.2` tetap menghasilkan 0 issue.
timestamp: 2026-08-02 00:00 WITA
kategori: Testing, Quality, Migration
file yang diubah:
- backend/config/migrate_test.go
- backend/handlers/cctv_test.go
- backend/handlers/maintenance_test.go
- backend/routes/routes_test.go
- backend/routes/websocket_test.go
alasan teknis: Menutup coverage boundary dan jalur sukses yang tersisa sebelum masuk ke versioned database migration.

## [0.13.0] - 2026-08-02

### Penyelesaian Coverage Fase 4.1

perubahan yang dilakukan: **Perluasan test handler, model, konfigurasi, dan route**
before:
- Coverage belum mencakup import Excel CCTV, jalur not-found update/delete, seeder admin existing, DSN database, route auth, dan handshake WebSocket tanpa token.
after:
- Test tambahan mencakup batas baris import, invalid/not-found CCTV dan maintenance, perubahan role user, UUID/config, seeder idempotent, route smoke, dan penolakan handshake WebSocket tanpa token.
- Seluruh test backend lulus dengan `go test ./... -count=1`; `golangci-lint v2.12.2` menghasilkan 0 issue.
timestamp: 2026-08-02 00:00 WITA
kategori: Testing, Quality
file yang diubah:
- backend/config/database.go
- backend/config/database_test.go
- backend/handlers/cctv_test.go
- backend/handlers/maintenance_test.go
- backend/handlers/user_test.go
- backend/models/seeder_test.go
- backend/routes/routes_test.go
- backend/routes/websocket_test.go
alasan teknis: Menutup celah coverage pada jalur validasi dan boundary yang paling berisiko mengalami regresi.

## [0.12.0] - 2026-08-02

### Audit Log Reliability dan Unit Test Tambahan

perubahan yang dilakukan: **Retry audit log dan perluasan pengujian edge case**
before:
- Audit log ditulis sekali secara asynchronous; kegagalan koneksi membuat catatan hilang.
- Belum ada test untuk retry audit, UUID hook, serta invalid UUID pada operasi update/delete utama.
after:
- Penyimpanan audit mencoba maksimal tiga kali dengan jeda 100 ms dan menyiarkan `AUDIT_LOG_ADDED` setelah sukses.
- Test tambahan mencakup retry audit, batas retry, UUID hook, dan validasi ID handler CCTV, maintenance, serta user.
timestamp: 2026-08-02 00:00 WITA
kategori: Security, Reliability, Testing
file yang diubah:
- backend/utils/audit.go
- backend/utils/audit_test.go
- backend/models/setup_test.go
- backend/handlers/cctv_test.go
- backend/handlers/maintenance_test.go
- backend/handlers/user_test.go
alasan teknis: Mencegah kehilangan jejak audit saat gangguan database sementara dan memperkecil risiko regresi pada endpoint mutasi.

## [0.11.0] - 2026-08-01

### Perluasan Unit Test Backend (ROADMAP Fase 4.1)

perubahan yang dilakukan: **Unit test komprehensif untuk `utils`, `middleware`, dan `handlers` (sqlmock tanpa PostgreSQL fisik)**
before:
- Cakupan unit test hanya ada di `utils/jwt_test.go`, `utils/validator_test.go`, dan `handlers/auth_test.go` (test Login).
- Fungsi penting (pagination, enkripsi AES, blacklist, CSRF, middleware keamanan, handler user/CCTV/maintenance/dashboard/audit/upload/report/health) tidak memiliki pengujian otomatis sehingga rawan regresi.
after:
- **utils (5 file test baru):** `pagination_test.go` (default limit/page/sort, batas atas limit 100 & page 1.000.000, whitelist sort vs SQL injection, `CalculatePagination`, `ParseDateRangeParams`), `encryption_test.go` (round-trip AES-256-GCM, nonce unik, hex invalid, input kosong), `blacklist_test.go` (revoke/check, cleanup token kadaluarsa), `csrf_test.go` (`GenerateCSRFToken` 64 hex unik, `ExtractTokenString` prioritas cookie → Bearer → kosong).
- **middleware (4 file test baru):** `auth_test.go` (`Protected` tanpa token 401, revoked 401, invalid, valid, token-version mismatch → `Sesi telah dicabut. Silakan login kembali.`; `RequireRole` none/wrong/allowed), `csrf_test.go` (metode aman & Bearer dilewati, tanpa cookie sesi dilewati, mismatch/missing 403, matching lolos), `limiter_test.go` (batas bawah lolos, > 100/menit → 429), `security_test.go` (header dasar selalu ada; CSP/HSTS hanya saat `APP_ENV=production`).
- **handlers (9 file test baru):** `auth_extra_test.go` (ChangePassword sukses/password lama salah/validasi, GetMyProfile, UpdateMyProfile, Logout, CreateUser validasi/duplikat/sukses), `user_test.go` (GetAllUsers, GetUserByID, UpdateUser, DeleteUser + self-delete forbidden, ResetUserPassword), `cctv_test.go` (GetCCTVByID, CreateCCTV + `qr_payload` terisi, ScanCCTV, DownloadTemplateCCTV via `excelize.OpenReader`, GetCCTVQRCode invalid UUID), `maintenance_test.go` (Create sukses/reload/CCTV not found, GetById, GetAll, kepemilikan teknisi → 403), `dashboard_test.go` (DashboardTasks tanpa identitas 401/teknisi ter-scope/admin, DashboardSummary distribusi status & tren 6 bulan), `audit_test.go` (GetAuditLogs sukses/filter modul, ExportAuditLogs CSV), `upload_test.go` (tanpa file 400, tipe invalid, JPEG valid magic bytes, path traversal 400, ekstensi salah 400, not found 404), `report_test.go` (PDF valid prefix `%PDF`, scope teknisi), `health_test.go` (HealthCheck Connected).
- **helper bersama:** `handlers/helpers_test.go` (`withUserLocals`, `setupMockGORM` inject ke `config.DB`, `newUserMockRows`, `modelsUserRow`, `cctvMockRow`, `maintenanceMockRow`).
- **Fix penting:** `SetupTestDB` di `auth_test.go` memakai `gorm.Config{SkipDefaultTransaction: true}` — tanpa ini GORM memanggil `BEGIN` tak terduga sehingga sqlmock gagal pada operasi tulis.
timestamp: 2026-08-01 16:00 WITA
kategori: Testing, Quality, UnitTest
file yang diubah:
- backend/utils/pagination_test.go (baru)
- backend/utils/encryption_test.go (baru)
- backend/utils/blacklist_test.go (baru)
- backend/utils/csrf_test.go (baru)
- backend/middleware/auth_test.go (baru)
- backend/middleware/csrf_test.go (baru)
- backend/middleware/limiter_test.go (baru)
- backend/middleware/security_test.go (baru)
- backend/handlers/auth_extra_test.go (baru)
- backend/handlers/user_test.go (baru)
- backend/handlers/cctv_test.go (baru)
- backend/handlers/maintenance_test.go (baru)
- backend/handlers/dashboard_test.go (baru)
- backend/handlers/audit_test.go (baru)
- backend/handlers/upload_test.go (baru)
- backend/handlers/report_test.go (baru)
- backend/handlers/health_test.go (baru)
- backend/handlers/helpers_test.go (baru)
- backend/handlers/auth_test.go
- backend/changelog.md
alasan perubahan: Menutup Fase 4.1 ROADMAP (perluas unit test backend) untuk mencegah regresi pada fungsi keamanan dan bisnis inti, serta menyediakan fondasi pengujian terisolasi (sqlmock) yang dapat diperluas ke package lain.

---

## [0.10.0] - 2026-08-01

### Fitur Kapabilitas Fase 3 (ROADMAP 3.1 - 3.9)

perubahan yang dilakukan: **Self-service password, sliding session, dashboard analitik, notifikasi tugas, profil pengguna, unifikasi aset→CCTV, laporan PDF & filter tanggal, privasi upload, pembatasan ekspor per role**
before:
- Teknisi/admin tidak bisa mengganti password sendiri; tidak ada paksa ganti password saat login pertama.
- Sesi JWT statis tanpa perpanjangan; token tidak di-refresh sehingga user logout saat masa berlaku habis.
- Tidak ada dashboard analitik, notifikasi tugas teknisi, maupun halaman profil.
- Model `assets` terpisah dari `cctv` (dua sumber data) sehingga log pemeliharaan bergantung pada tabel `assets`.
- Belum ada laporan PDF, ekspor audit-log, atau filter rentang tanggal pada list/ekspor.
- File upload menyimpan metadata EXIF (termasuk lokasi GPS) dan file yatim tidak pernah dibersihkan.
- Ekspor CSV/PDF maintenance terbuka untuk semua role.
after:
- **3.1 Change password:** `POST /auth/change-password` self-service + flag `must_change_password` saat login pertama.
- **3.2 Sliding session:** klaim `exp` diperpanjang pada akses aktif (refresh otomatis di `AuthContext`), blacklist token tetap jalan.
- **3.3 Dashboard analitik:** `GET /dashboard/overview`, `GET /dashboard/analytics`, `GET /dashboard/overdue`, `GET /dashboard/activity` + `DashboardAnalytics`/`DashboardStats` di frontend.
- **3.4 Notifikasi tugas:** `GET /dashboard/tasks` — teknisi melihat tugasnya sendiri (`technician_id`), admin semua; limit 20, `pending_count` & `overdue_count` + `days_overdue`.
- **3.5 Profil:** `PUT /auth/profile` (nama tampilan) + halaman `/profile`; kartu user di sidebar jadi link.
- **3.6 Unifikasi aset→CCTV:** `MaintenanceLog` kini memakai `cctv_id` (UUID FK ke `cctvs`, relasi `CCTV`); model `Asset`, `SeedAssets`, `handlers/asset.go`, `routes/asset.go` dihapus; `config.MigrateMaintenanceToCCTV` memetakan data legacy `asset_id` → `cctv_id` (ID/IP/kode/nama) sekali saat startup; JSON API kini `cctv`/`cctv_name`/`cctv_location`.
- **3.7 Laporan & filter tanggal:** `GET /maintenance-logs/report-pdf` (PDF via `go-pdf/fpdf`), `GET /audit-logs/export` (CSV), filter `start_date`/`end_date` (helper `ParseDateRangeParams`) pada list & ekspor maintenance dan audit-log.
- **3.8 Privasi & kebersihan upload:** seluruh upload di-decode ulang & dinormalisasi ke JPEG sehingga metadata EXIF/GPS terhapus; goroutine `StartUploadCleanup` (tiap 1 jam) menghapus file yatim berumur >24 jam yang tidak direferensikan `image_url`.
- **3.9 Pembatasan ekspor per role:** `ExportMaintenanceLogs` & `GenerateMaintenanceReportPDF` membatasi teknisi hanya pada log miliknya sendiri (`technician_id = uid`); admin penuh; ekspor audit-log sudah admin-only.
timestamp: 2026-08-01 14:30 WITA
kategori: Feature, Security, API, Dashboard, PDF, Upload
file yang diubah:
- backend/handlers/auth.go (3.1, 3.5)
- backend/handlers/dashboard.go (baru, 3.3, 3.4)
- backend/handlers/upload.go (3.8)
- backend/handlers/report.go (baru, 3.7)
- backend/handlers/audit.go (3.7)
- backend/handlers/maintenance.go (3.6, 3.7, 3.9)
- backend/handlers/asset.go (dihapus, 3.6)
- backend/routes/dashboard.go (baru, 3.3, 3.4)
- backend/routes/asset.go (dihapus, 3.6)
- backend/routes/maintenance.go (3.7, 3.9)
- backend/routes/audit.go (3.7)
- backend/routes/auth.go (3.5)
- backend/config/migrate.go (baru, 3.6)
- backend/models/setup.go (3.6)
- backend/models/seeder.go (3.6)
- backend/utils/pagination.go (3.7)
- backend/main.go (3.6, 3.8)
- backend/docs/ (regenerated via swag)
- backend/changelog.md
alasan perubahan: Meningkatkan nilai operasional sistem (ROADMAP Fase 3) dan menutup temuan keamanan 18 (ekspor terbuka semua role) dari docs/SECURITY-AUDIT.md.

---

## [0.9.0] - 2026-08-01

### Hardening Keamanan Fase 2 (Security Audit & Roadmap)

perubahan yang dilakukan: **Penerapan Perbaikan Berisiko Tinggi & Menengah dari Hasil Audit Keamanan (Fase 2)**
before:
- Token JWT disimpan di `localStorage` dan dikirim via header `Authorization` sehingga rentan terhadap XSS token theft; mutasi terautentikasi tidak dilindungi CSRF.
- Password di-reset atau role diubah tanpa mencabut token JWT lama yang masih beredar.
- Rate limiter & audit log mempercayai header `X-Forwarded-For`/`X-Real-IP` dari siapa pun (risiko spoofing IP untuk mem-bypass rate limit).
- Impor Excel CCTV tanpa batas ukuran file maupun jumlah baris (risiko memori/DB overload).
- CORS memakai wildcard `*` dengan credentials; Docker mempublish port DB 5432, memakai Cloudflare quick tunnel, dan meng-hardcode kredensial DB di `docker-compose.yml`.
- Header CSP & HSTS belum aktif di produksi; teknisi dapat mengubah log pemeliharaan milik teknisi lain; kebijakan password hanya `min=6` tanpa kompleksitas.
after:
- **Token di cookie httpOnly + CSRF:** JWT dipindah ke cookie `sigap_session` (HttpOnly, Secure di produksi, SameSite=Lax) + pola double-submit CSRF (`sigap_csrf` cookie + header `X-CSRF-Token`); backend tetap mendukung header Bearer untuk client API/Swagger. WebSocket membaca cookie (fallback `?token=`). CORS `AllowCredentials: true` tanpa wildcard, default aman `http://localhost:3001`.
- **Revoke token (token version):** klaim `tv` ditambahkan pada JWT; versi token dinaikkan saat `UpdateUser` (role berubah) dan `ResetUserPassword`, sehingga seluruh token lama langsung dicabut (`ValidateTokenVersion` di `Protected()` & handshake WebSocket).
- **Trusted proxy:** header proxy hanya dipercaya dari IP yang terdaftar di env `TRUSTED_PROXY_IPS` (konfigurasi Fiber `TrustedProxies`); `getClientIP` (limiter) & `ExtractClientIP` (audit) kini memakai `c.IP()`.
- **Batas impor Excel:** maksimum 4 MB per file dan 1000 baris data di `ImportCCTV`.
- **Docker hardening:** quick tunnel dihapus; tambah reverse proxy Nginx + TLS (`nginx/nginx.conf`, sertifikat di `nginx/certs`); port DB/API/frontend tidak lagi dipublish ke host; kredensial DB dibaca dari root `.env` (secret management); `TRUSTED_PROXY_IPS` merujuk IP statis nginx `172.28.0.10`.
- **CSP & HSTS produksi:** header `Content-Security-Policy` (strict) dan `Strict-Transport-Security` aktif hanya saat `APP_ENV=production`.
- **Ownership check:** teknisi hanya dapat mengubah log pemeliharaan miliknya sendiri (`UpdateMaintenanceLog`); admin tetap penuh.
- **Kebijakan password:** minimal 8 karakter + kompleksitas (huruf besar, huruf kecil, angka, karakter khusus) lewat validator custom `strongpassword` pada `CreateUserRequest` & `ResetUserPasswordRequest`.
timestamp: 2026-08-01 13:45 WITA
kategori: Security, Hardening, API, Docker, Infra
file yang diubah:
- backend/utils/csrf.go (baru)
- backend/middleware/csrf.go (baru)
- backend/middleware/auth.go
- backend/middleware/security.go
- backend/middleware/limiter.go
- backend/utils/jwt.go
- backend/utils/jwt_test.go
- backend/utils/validator.go
- backend/utils/validator_test.go (baru)
- backend/utils/audit.go
- backend/models/setup.go
- backend/handlers/auth.go
- backend/handlers/user.go
- backend/handlers/cctv.go
- backend/handlers/maintenance.go
- backend/handlers/auth_test.go
- backend/routes/websocket.go
- backend/main.go
- backend/.env.example
- backend/.env.docker.example
- docker-compose.yml
- nginx/nginx.conf (baru)
- nginx/certs/README.md (baru)
- .env.example (baru, root)
- frontend/src/lib/api.ts
- frontend/src/context/AuthContext.tsx
- frontend/src/context/WebSocketContext.tsx
- frontend/src/app/login/page.tsx
- frontend/src/app/users/page.tsx
- frontend/src/components/AuthenticatedImage.tsx
- frontend/src/app/cctv/page.tsx
- backend/changelog.md
alasan perubahan: Menutup temuan Fase 2 (docs/SECURITY-AUDIT.md temuan 7-20) untuk mencegah pencurian token via XSS, CSRF, token lama yang masih valid, spoofing IP pada rate limit, overload impor Excel, eksposur DB/layanan via Docker, dan pelanggaran ownership/password lemah.

---

## [0.8.0] - 2026-08-01

### Hardening Keamanan Fase 1 (Security Audit & Roadmap)

perubahan yang dilakukan: **Penerapan Perbaikan Kritis dari Hasil Audit Keamanan (Temuan Kritis/Tinggi)**
before:
- `getJWTSecret()` memiliki fallback secret hardcoded `super_secret_sigap_jwt_key_ikn_2026_secure` yang dipakai bila env `JWT_SECRET` kosong (risiko pemalsuan token JWT).
- `SeedAdmin` mereset kata sandi admin default setiap kali server dimulai (risiko pengambilalihan akun bila env default bocor).
- Endpoint `GET /api/ws` dapat diakses tanpa autentikasi (risiko injeksi/pemalsuan event ke client lain).
- Cetak QR Code CCTV mengandalkan API pihak ketiga `api.qrserver.com` dan mengirim `QrPayload` mentah ke layanan eksternal (kebocoran data terenkripsi + dependensi layanan luar).
- Pagination `limit` dan `page` tidak dibatasi sehingga dapat memicu beban berlebih pada basis data.
after:
- `getJWTSecret()` kini selalu membaca dari env `JWT_SECRET` tanpa fallback; nilai secret kosong akan memicu kegagalan cepat (fail-fast) sehingga server tidak berjalan dengan secret lemah.
- `SeedAdmin` hanya membuat akun admin jika belum ada dan tidak mengubah kata sandi pada startup berikutnya; validasi fail-fast bila `ADMIN_DEFAULT_PASSWORD` kosong/lemah (< 8 karakter).
- Rute `GET /api/ws` divalidasi melalui token JWT pada query parameter `?token=`; koneksi tanpa token valid ditolak, identitas pengguna (user id & role) dimasukkan ke konteks koneksi.
- Ditambahkan endpoint `GET /api/cctv/:id/qrcode` yang menghasilkan gambar QR PNG server-side (pustaka `github.com/skip2/go-qrcode`) berisi `QrPayload` terenkripsi; menggantikan dependensi API QR pihak ketiga.
- Pagination dibatasi: `limit` maksimum 100 dan `page` maksimum 1.000.000; nilai di luar batas dijepit ke batas aman.
timestamp: 2026-08-01 15:30 WITA
kategori: Security, Hardening, API
file yang diubah:
- backend/utils/jwt.go
- backend/utils/jwt_test.go
- backend/models/seeder.go
- backend/utils/pagination.go
- backend/routes/websocket.go
- backend/handlers/cctv.go
- backend/routes/cctv.go
- backend/.env.example
- backend/.env.docker.example
- backend/changelog.md
alasan perubahan: Menutup temuan kritis dan berisiko tinggi hasil audit keamanan (docs/SECURITY-AUDIT.md) untuk mencegah pemalsuan token, pengambilalihan akun, akses WebSocket tak terautentikasi, kebocoran data ke layanan eksternal, dan eksploitasi pagination.

---

## [0.7.0] - 2026-08-01

### Penyempurnaan Dokumentasi API Swagger UI (OpenAPI 3.0)

perubahan yang dilakukan: **Penyelarasan Seluruh Anotasi Swag & Regenerasi Swagger Docs**
before:
- Endpoint baru seperti `/api/audit-logs`, `/api/ws`, `/api/upload`, dan `/api/uploads/{filename}` belum sepenuhnya terdaftar pada skema OpenAPI Swagger UI.
after:
- Menambahkan anotasi OpenAPI 3.0 Swag lengkap pada rute `/api/audit-logs`, `/api/upload`, `/api/uploads/{filename}`, dan `/api/ws`.
- Mengonfigurasi `swag CLI` v1.16.4 dan melakukan regenerasi berkas `docs/docs.go`, `docs/swagger.json`, dan `docs/swagger.yaml`.
- Dokumentasi API interaktif kini 100% selaras dan dapat diuji langsung di `http://localhost:3000/swagger/index.html`.
timestamp: 2026-08-01 01:43 WITA
kategori: Feature, Documentation, Swagger, API
file yang diubah:
- backend/routes/websocket.go
- backend/handlers/audit.go
- backend/handlers/upload.go
- backend/docs/docs.go
- backend/docs/swagger.json
- backend/docs/swagger.yaml
- backend/changelog.md
alasan perubahan: Menyediakan dokumen API interaktif standar OpenAPI 3.0 yang lengkap bagi tim teknis dan pengembang integrasi sistem SIGAP IKN.

---

## [0.6.0] - 2026-08-01

### Real-time WebSocket Broadcast Engine

perubahan yang dilakukan: **Integrasi WSHub Terpusat, Endpoint GET /api/ws, dan Penyiaran Event Real-time**
before:
- Tidak ada dukungan WebSocket di backend; pembaruan data di frontend memerlukan refresh manual atau polling berulang.
after:
- Mengintegrasikan pustaka `github.com/gofiber/websocket/v2`.
- Dibuat utilitas `WSHub` di `utils/websocket.go` untuk mengelola koneksi client aktif secara *thread-safe* (`sync.RWMutex`) dan fungsi `BroadcastEvent`.
- Dibuat rute `GET /api/ws` di `routes/websocket.go` dengan middleware pengecekan upgrade protokol WebSocket.
- Diintegrasikan panggilan `utils.BroadcastEvent` pada event penting:
  - `cctv.go`: Event `CCTV_UPDATED` pada `CreateCCTV`, `UpdateCCTV`, dan `DeleteCCTV`.
  - `maintenance.go`: Event `MAINTENANCE_CREATED` dan `MAINTENANCE_UPDATED` pada `CreateMaintenanceLog` dan `UpdateMaintenanceLog`.
timestamp: 2026-08-01 01:35 WITA
kategori: Feature, Real-time, WebSocket
file yang diubah:
- backend/utils/websocket.go (baru)
- backend/routes/websocket.go (baru)
- backend/handlers/cctv.go
- backend/handlers/maintenance.go
- backend/main.go
- backend/changelog.md
alasan perubahan: Memungkinkan pemantauan operasional CCTV dan pekerjaan pemeliharaan di kawasan IKN secara langsung (*real-time live update*) tanpa latensi refresh.

---

## [0.5.1] - 2026-08-01

### Fix Auto-Logout Mismatch & Environment Loading Order

perubahan yang dilakukan: **Perbaikan Inisialisasi Environment godotenv & Pembaruan getJWTSecret**
before:
- `godotenv.Load()` baru dipanggil di dalam `config.ConnectDB()`, yang berjalan setelah `requireEnv("JWT_SECRET")`.
- Utilitas `getJWTSecret()` menyimpan byte slice `jwtSecret` secara permanen pada memori; jika dipanggil saat nilai environment belum siap atau berubah, token JWT yang dibuat tidak konsisten sehingga memicu error 401 (Auto-Logout).
after:
- Mengubah `godotenv.Load()` agar dieksekusi di baris pertama `main()` pada `main.go`.
- Memperbarui `getJWTSecret()` di `utils/jwt.go` agar selalu mengembalikan `[]byte(os.Getenv("JWT_SECRET"))` secara langsung tanpa menyimpan cache slice yang usang.
timestamp: 2026-08-01 01:25 WITA
kategori: Bugfix, Auth, Security
file yang diubah:
- backend/main.go
- backend/utils/jwt.go
- backend/changelog.md
alasan perubahan: Memastikan validasi token JWT di backend berjalan konsisten dan tidak memicu penolakan otentikasi (401) yang menyebabkan pengguna terkeluar (auto-logout) saat navigasi antar halaman.

---

## [0.5.0] - 2026-08-01

### Modul Audit Logging Terpusat (Audit Trail)

perubahan yang dilakukan: **Implementasi Modul Audit Logging Terpusat & Rute GET /api/audit-logs**
before:
- Tidak ada pencatatan histori aktivitas penting pengguna (siapa yang mengubah data CCTV, menambah aset, menghapus log pemeliharaan, atau mereset password).
- Tidak ada entitas `AuditLog` di database PostgreSQL.
- Tidak ada rute API untuk melihat log riwayat aktivitas operasional sistem.
after:
- Dibuat model `AuditLog` di `models/audit_log.go` dan didaftarkan ke `AutoMigrate` di `main.go`.
- Dibuat utilitas `LogAudit` (non-blocking async via Goroutine) di `utils/audit.go` dengan fitur *Smart User Resolution* dan penanganan client IP proxy-aware (`X-Real-IP`, `X-Forwarded-For`).
- Dibuat handler `GetAuditLogs` di `handlers/audit.go` dan rute `GET /api/audit-logs` terproteksi JWT Admin di `routes/audit.go`.
- Diintegrasikan pencatatan audit log pada seluruh handler penting:
  - Auth (`handlers/auth.go`): `LOGIN_SUCCESS`, `LOGIN_FAILED`, `LOGOUT`, `CREATE_USER`
  - CCTV (`handlers/cctv.go`): `CREATE_CCTV`, `UPDATE_CCTV`, `DELETE_CCTV`, `IMPORT_CCTV`, `BULK_DELETE_CCTV`
  - Aset (`handlers/asset.go`): `CREATE_ASSET`, `UPDATE_ASSET`, `DELETE_ASSET`
  - Pemeliharaan (`handlers/maintenance.go`): `CREATE_MAINTENANCE`, `UPDATE_MAINTENANCE`, `DELETE_MAINTENANCE`
  - Manajemen User (`handlers/user.go`): `UPDATE_USER`, `DELETE_USER`, `RESET_PASSWORD`
- Melepas ketergantungan import package `utils` dari `models/seeder.go` untuk memutus import cycle.
timestamp: 2026-08-01 00:38 WITA
kategori: besar
file yang diubah:
- backend/models/audit_log.go (baru)
- backend/utils/audit.go (baru)
- backend/handlers/audit.go (baru)
- backend/routes/audit.go (baru)
- backend/models/seeder.go
- backend/main.go
- backend/handlers/auth.go
- backend/handlers/cctv.go
- backend/handlers/asset.go
- backend/handlers/maintenance.go
- backend/handlers/user.go
alasan perubahan: Memenuhi kebutuhan auditability operasional infrastruktur IKN dengan mencatat seluruh aktivitas administratif penting ke dalam database secara otomatis, presisi, dan non-blocking.

---

## [0.4.0] - 2026-07-31

### Security Hardening (Critical Fixes)

perubahan yang dilakukan: **Penghapusan Fallback Ekstensi pada File Upload & Proteksi Akses File via JWT**
before:
- `handlers/upload.go` menggunakan fallback ke ekstensi nama file jika deteksi MIME magic bytes gagal, rentan terhadap upload file berbahaya.
- `handlers/upload.go` memiliki bug variabel `contentType` tidak terdefinisi (runtime error).
- File gambar yang diunggah dapat diakses publik via `app.Static("/uploads")` tanpa autentikasi apapun.
after:
- Fallback ekstensi dihapus sepenuhnya — hanya magic bytes yang dipercaya; file ditolak jika tipe tidak terdeteksi.
- Bug `contentType` diperbaiki menggunakan `detectedType` yang sudah ada.
- Ditambahkan handler `ServeUploadedFile` dan rute `GET /api/uploads/:filename` (dilindungi JWT + validasi path traversal).
- `app.Static("/uploads")` dihapus dari `main.go`.
- URL yang dikembalikan upload handler diubah dari `/uploads/...` menjadi `/api/uploads/...`.
timestamp: 2026-07-31 23:06 WITA
kategori: kritis
file yang diubah:
- backend/handlers/upload.go
- backend/routes/upload.go
- backend/main.go
alasan perubahan: Mencegah bypass validasi tipe file berbahaya dan melindungi file gambar dari akses tidak terautentikasi.

---

perubahan yang dilakukan: **Fix Rate Limiter — Proxy-Aware IP Detection**
before:
- `middleware/limiter.go` menggunakan `c.IP()` sebagai key rate limiting, yang mengembalikan IP proxy di balik Nginx/Load Balancer sehingga seluruh user terhitung sebagai satu IP.
after:
- Ditambahkan fungsi `getClientIP()` yang memprioritaskan `X-Real-IP` → `X-Forwarded-For` → `c.IP()` sebagai fallback.
- Digunakan di `GlobalLimiter()` dan `AuthLimiter()`.
timestamp: 2026-07-31 23:07 WITA
kategori: kritis
file yang diubah:
- backend/middleware/limiter.go
alasan perubahan: Memastikan rate limiting bekerja benar per IP user asli, bukan per IP proxy.

---

perubahan yang dilakukan: **Hapus Hardcoded Password Admin dari Seeder**
before:
- `models/seeder.go` meng-hardcode password `adminpassword` langsung di source code.
after:
- Password admin dibaca dari environment variable `ADMIN_DEFAULT_PASSWORD`.
- Jika tidak diset, password fallback otomatis dibuat dari prefix JWT_SECRET dengan peringatan eksplisit di log.
- Log menggunakan Zap Logger (konsisten dengan sistem).
timestamp: 2026-07-31 23:07 WITA
kategori: kritis
file yang diubah:
- backend/models/seeder.go
- backend/.env.example
- backend/.env.docker.example
alasan perubahan: Mencegah kredensial default yang mudah ditebak ada di source code yang bisa diakses publik di repositori.

---

perubahan yang dilakukan: **Implementasi JWT Token Blacklist & Endpoint Logout**
before:
- Tidak ada endpoint logout di backend; logout hanya terjadi di sisi client (hapus localStorage).
- Token JWT yang sudah di-logout masih valid selama 24 jam penuh di server.
after:
- Dibuat `utils/blacklist.go` berisi in-memory token blacklist dengan `sync.RWMutex` untuk thread safety.
- Fungsi `RevokeToken`, `IsTokenRevoked`, dan `StartBlacklistCleanup` (goroutine cleanup otomatis setiap 30 menit).
- Middleware `Protected()` di `middleware/auth.go` sekarang memeriksa blacklist sebelum memvalidasi token.
- Ditambahkan handler `Logout` di `handlers/auth.go` dan rute `POST /api/auth/logout`.
- Goroutine cleanup diaktifkan di `main.go` saat server startup.
timestamp: 2026-07-31 23:08 WITA
kategori: tinggi
file yang diubah:
- backend/utils/blacklist.go (baru)
- backend/middleware/auth.go
- backend/handlers/auth.go
- backend/routes/auth.go
- backend/main.go
alasan perubahan: Memastikan token yang sudah di-logout tidak dapat disalahgunakan; mencegah session hijacking pasca logout.

---

perubahan yang dilakukan: **Tambah Endpoint GET /api/auth/me & Handler GetMyProfile**
before:
- Tidak ada endpoint untuk mengambil profil user yang sedang login secara langsung dari server.
- Data user hanya tersedia dari respons login yang disimpan di localStorage (rawan manipulasi sisi client).
after:
- Ditambahkan handler `GetMyProfile` di `handlers/auth.go` dan rute `GET /api/auth/me`.
- Endpoint membaca `user_id` dari JWT claims dan mengembalikan data profil terkini dari database.
timestamp: 2026-07-31 23:08 WITA
kategori: sedang
file yang diubah:
- backend/handlers/auth.go
- backend/routes/auth.go
alasan perubahan: Menyediakan sumber data profil yang terpercaya dari server, bukan hanya dari localStorage client.

---

perubahan yang dilakukan: **Pembatasan Swagger UI ke Non-Production & Endpoint test-panic**
before:
- Swagger UI selalu aktif di semua environment termasuk production, mengekspos seluruh detail API.
- Endpoint `GET /api/test-panic` selalu aktif tanpa pembatasan environment.
after:
- Swagger UI hanya aktif jika `APP_ENV != "production"`.
- Endpoint `test-panic` hanya aktif jika `APP_ENV != "production"`.
- Variabel `APP_ENV` ditambahkan ke `.env.example` dan `.env.docker.example`.
timestamp: 2026-07-31 23:08 WITA
kategori: sedang
file yang diubah:
- backend/main.go
- backend/.env.example
- backend/.env.docker.example
alasan perubahan: Mencegah disclosure informasi struktur API di production dan menutup endpoint uji coba dari akses luar.

---

perubahan yang dilakukan: **Tambah Security Headers Middleware**
before:
- Tidak ada security headers HTTP pada response (X-Frame-Options, X-Content-Type-Options, dll).
after:
- Dibuat `middleware/security.go` berisi `SecurityHeaders()` yang memasang header: `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `X-XSS-Protection: 1; mode=block`, `Referrer-Policy: strict-origin-when-cross-origin`, `Server: SIGAP`.
- Dipasang setelah ZapLogger di `main.go`.
timestamp: 2026-07-31 23:09 WITA
kategori: sedang
file yang diubah:
- backend/middleware/security.go (baru)
- backend/main.go
alasan perubahan: Melindungi dari serangan MIME sniffing, clickjacking, dan mengurangi information disclosure.

---

perubahan yang dilakukan: **Cache AES Key dengan sync.Once**
before:
- `utils/encryption.go` fungsi `deriveKey()` memanggil `os.Getenv()` dan `sha256.Sum256()` setiap kali dipanggil.
after:
- Ditambahkan `aesKey` dan `aesKeyOnce` package-level variable; `deriveKey()` menggunakan `sync.Once` untuk menjamin key hanya diderivasi satu kali.
timestamp: 2026-07-31 23:09 WITA
kategori: rendah
file yang diubah:
- backend/utils/encryption.go
alasan perubahan: Efisiensi dan konsistensi — menghindari overhead SHA-256 berulang setiap enkripsi/dekripsi QR payload.

## [0.2.0] - 2026-07-22

### Added
- Pembuatan **Database Seeder** (`backend/models/seeder.go`) untuk menyuntikkan user Admin pertama secara otomatis (`admin` / `adminpassword`) ke dalam database PostgreSQL saat tabel `users` kosong.
- Pembuatan utilitas **JWT Helper** (`backend/utils/jwt.go`) untuk pembuatan (*generating*) dan verifikasi (*validating*) JSON Web Token dengan klaim `user_id` dan `role`.
- Pembuatan **Auth Middleware** (`backend/middleware/auth.go`) yang terdiri dari:
  - `Protected()`: Memvalidasi token JWT dari header Authorization Bearer.
  - `RequireRole(allowedRoles ...string)`: Membatasi hak akses rute endpoint berdasarkan peranan pengguna (`admin` dan `teknisi`).
- Pembuatan **Auth Handlers & Routes** (`backend/handlers/auth.go` & `backend/routes/auth.go`):
  - `Login`: Memproses autentikasi pengguna menggunakan username maupun email.
  - `CreateUser`: Mengizinkan pendaftaran user baru (Admin atau Teknisi), dibatasi khusus hanya dapat dipicu oleh Admin.
- Pembuatan **CRUD Asset Handlers & Routes** (`backend/handlers/asset.go` & `backend/routes/asset.go`):
  - Menyediakan endpoint lengkap untuk Create, Read (List & Detail), Update, dan Delete Aset CCTV.
  - Otorisasi penambahan, pembaruan, dan penghapusan data aset dilindungi secara ketat khusus untuk role `admin`.
- Pembuatan **CRUD Maintenance Log Handlers & Routes** (`backend/handlers/maintenance.go` & `backend/routes/maintenance.go`):
  - Menyediakan endpoint untuk Create, Read (List & Detail), Update, dan Delete Log Pemeliharaan.
  - Input `technician_id` dideteksi secara otomatis dari token JWT pengguna saat ini.
  - Aksi penghapusan log pemeliharaan dibatasi khusus untuk role `admin`.
- Pemasangan middleware **CORS** dari framework Fiber di `backend/main.go` untuk mengizinkan request lintas port/domain dari frontend.

### Changed
- Refaktorisasi berkas **`models/setup.go`** untuk **menghapus entitas `Issue`** secara menyeluruh (sesuai aturan baru sistem tanpa tiket gangguan), serta membersihkan relasi terkait pada entitas `MaintenanceLog`.
- Modifikasi berkas **`main.go`** untuk menghapus registrasi model `Issue` dari daftar target `AutoMigrate` GORM database.

## [0.3.0] - 2026-07-23

perubahan yang dilakukan: Integrasi Swagger API Documentation pada backend Go Fiber.
before:
- main.go tidak memiliki rute /swagger/*, import docs, maupun anotasi global Swagger.
- handlers (auth.go, asset.go, maintenance.go) tidak memiliki anotasi Swag.
after:
- main.go mengimpor docs, memanggil swagger middleware pada /swagger/*, dan memiliki anotasi global.
- Semua fungsi handler di auth.go, asset.go, dan maintenance.go dilengkapi anotasi Swag lengkap.
timestamp: 2026-07-23 09:00 WITA
kategori: sedang
file yang diubah:
- backend/main.go
- backend/handlers/auth.go
- backend/handlers/asset.go
- backend/handlers/maintenance.go
alasan perubahan: Memudahkan tim frontend (Intan) untuk melihat dan menguji spesifikasi request/response API SIGAP tanpa membaca kode program backend.

perubahan yang dilakukan: Mengimplementasikan sistem validasi input terpusat (Strict Input Validation) menggunakan go-playground/validator/v10.
before:
- Masukan input request API divalidasi secara manual menggunakan pengecekan string kosong (`""`) di handler masing-masing.
- Tidak ada pemeriksaan format email, IP address, coordinates (latitude/longitude), maupun format UUID untuk data asset_id di log pemeliharaan.
after:
- Dibuat helper terpusat `utils.ValidateStruct` di `utils/validator.go`.
- Structs request di handlers (auth.go, asset.go, maintenance.go) dilengkapi dengan tag validasi terperinci (`required`, `email`, `min`, `max`, `ip`, `latitude`, `longitude`, `oneof`, `uuid4`).
- Pengecekan manual di masing-masing handler digantikan oleh pemanggilan validator global.
timestamp: 2026-07-23 09:10 WITA
kategori: sedang
file yang diubah:
- backend/utils/validator.go
- backend/handlers/auth.go
- backend/handlers/asset.go
- backend/handlers/maintenance.go
alasan perubahan: Menjaga konsistensi data, validitas data masukan di tingkat aplikasi sebelum disimpan ke database PostgreSQL, serta memberikan respons error validasi yang informatif bagi klien.

perubahan yang dilakukan: Implementasi fitur paginasi (Pagination), sorting aman (Sort Whitelisting), dan pencarian data (Search) terintegrasi pada list Aset CCTV dan Log Pemeliharaan.
before:
- Endpoint GET `/assets` dan `/maintenance-logs` mengembalikan seluruh isi data tabel tanpa limitasi (tidak terpaginasi).
- Fitur pencarian tidak didukung di log pemeliharaan, dan di rute aset tidak mencakup nama lokasi (hanya nama/serial number).
- Tidak ada mekanisme keamanan whitelisting kolom pengurutan (potensi celah SQL Injection jika sorting langsung dari query client).
after:
- Dibuat utilitas paginasi `utils/pagination.go` yang memvalidasi kolom sort terhadap whitelist kolom aman dan memvalidasi arah sorting (asc/desc).
- Handler GetAllAssets dan GetAllMaintenanceLogs diubah untuk menghitung total data dan memotong data berbasis offset/limit serta mendukung pencarian teks lanjutan (`ILIKE`).
- Swagger anotasi pada handler diperbarui dengan kelima parameter paginasi (page, limit, sort_by, order, search).
timestamp: 2026-07-23 09:35 WITA
kategori: sedang
file yang diubah:
- backend/utils/pagination.go
- backend/handlers/asset.go
- backend/handlers/maintenance.go
alasan perubahan: Mencegah kegagalan memori dan kelambatan jaringan saat volume data tinggi, serta mengamankan parameter pengurutan (sort) dari ancaman celah SQL Injection di GORM.

perubahan yang dilakukan: Perbaikan tipe model request body pada anotasi Swagger (CreateAsset, UpdateAsset, CreateMaintenanceLog, UpdateMaintenanceLog).
before:
- Anotasi Swagger masih merujuk ke struct lama `AssetRequest` dan `MaintenanceLogRequest` yang sudah dihapus karena proses pemisahan struct.
after:
- Anotasi Swagger disesuaikan dengan benar untuk merujuk ke `CreateAssetRequest`, `UpdateAssetRequest`, `CreateMaintenanceLogRequest`, dan `UpdateMaintenanceLogRequest`.
timestamp: 2026-07-23 09:40 WITA
kategori: low
file yang diubah:
- backend/handlers/asset.go
- backend/handlers/maintenance.go
alasan perubahan: Memperbaiki error kompilasi swag init yang disebabkan oleh tidak ditemukannya definisi struct model lama.

perubahan yang dilakukan: Mengubah tipe parameter `page` dan `limit` dari `int` menjadi `integer` pada anotasi Swagger.
before:
- Tipe data parameter page dan limit didefinisikan sebagai `int`.
after:
- Tipe data parameter page dan limit disesuaikan menjadi `integer` untuk kompatibilitas penuh dengan Swagger generator.
timestamp: 2026-07-23 09:47 WITA
kategori: low
file yang diubah:
- backend/handlers/asset.go
- backend/handlers/maintenance.go
alasan perubahan: Memastikan parameter terdeteksi dengan benar dan kompatibel dengan standar OpenAPI/Swagger generator.

perubahan yang dilakukan: Pembuatan unit testing otomatis dan database mocking komprehensif untuk modul JWT dan Autentikasi.
before:
- Tidak ada file pengujian unit (unit testing) otomatis di proyek backend, sehingga pipa CI/CD GitHub Actions belum memverifikasi stabilitas kode.
after:
- Dibuat `utils/jwt_test.go` untuk menguji pembuatan token (`GenerateToken`) dan validasi token (`ValidateToken`) termasuk masa kadaluarsa dan validasi signature.
- Dibuat `handlers/auth_test.go` untuk mensimulasikan login pengguna (`Login`) dalam skenario sukses, password salah, user tidak ditemukan, dan error validasi menggunakan GORM mock DB (`go-sqlmock`) tanpa bergantung pada PostgreSQL fisik.
timestamp: 2026-07-23 10:02 WITA
kategori: sedang
file yang diubah:
- backend/handlers/auth_test.go
alasan perubahan: Menyediakan cakupan pengujian unit yang terisolasi dan andal untuk membuktikan stabilitas autentikasi pada pipa integrasi berkelanjutan CI/CD.

perubahan yang dilakukan: Perbaikan minor pada kode asersi unit test JWT dan pembersihan import unused di auth_test.go.
before:
- `auth_test.go` mengimpor `"sigap-backend/models"` tanpa menggunakannya (memicu error kompilasi).
- `jwt_test.go` menegaskan `parsedToken` bernilai `nil` saat signature salah, padahal pustaka jwt-go tetap mengembalikan object token non-nil dengan `.Valid = false`.
after:
- Menghapus import `"sigap-backend/models"` pada `auth_test.go`.
- Menyesuaikan asersi `jwt_test.go` menjadi `assert.False(t, parsedToken.Valid)` agar sinkron dengan perilaku pustaka jwt.
timestamp: 2026-07-23 10:07 WITA
kategori: low
file yang diubah:
- backend/handlers/auth_test.go
- backend/utils/jwt_test.go
alasan perubahan: Menyelesaikan kegagalan kompilasi dan kesalahan asersi agar seluruh unit test pass 100%.

perubahan yang dilakukan: Perbaikan salah ketik rute URL pada berkas auth_test.go.
before:
- Permintaan pengujian `net/http/httptest` mengirimkan request ke rute `"/api/api/auth/login"` (menyebabkan respon 404).
after:
- Rute pengujian disesuaikan menjadi `"/api/auth/login"` yang sesuai dengan registrasi rute Fiber.
timestamp: 2026-07-23 10:14 WITA
kategori: low
file yang diubah:
- backend/handlers/auth_test.go
alasan perubahan: Menyelesaikan error HTTP 404 saat unit test dijalankan agar login handler test bernilai PASS.

perubahan yang dilakukan: Penambahan parameter limit pada ekspektasi query sqlmock di auth_test.go.
before:
- Ekspektasi query `WithArgs` di sqlmock hanya mengharapkan 2 argumen untuk pencarian user (username, email).
after:
- Ekspektasi query `WithArgs` ditambahkan argumen ketiga bernilai 1 karena driver postgres GORM v2 mempassing limit 1 sebagai parameter query.
timestamp: 2026-07-23 10:15 WITA
kategori: low
file yang diubah:
- backend/handlers/auth_test.go
alasan perubahan: Menyelesaikan error ketidakcocokan jumlah argumen query sqlmock yang menyebabkan pengujian Login gagal.

perubahan yang dilakukan: Integrasi Logging Terstruktur (Zap Logger), Middleware HTTP Logger, dan Endpoint Health Check.
before:
- Menggunakan standar output terminal bawaan (`log.Println`, `log.Fatal`, dan `fmt.Println`) yang tidak terstruktur.
- Tidak ada file log server permanen untuk aktivitas produksi.
- Tidak ada endpoint pemantauan kesehatan layanan (health check).
after:
- Zap Logger terintegrasi di `utils/logger.go` dengan output ganda: Console dev berwarna dan file JSON di `logs/server.log`.
- Dipasang `middleware.ZapLogger()` pada main Fiber app untuk mencatat setiap request HTTP masuk.
- Dibuat endpoint `GET /api/health` di `handlers/health.go` yang memvalidasi keaktifan API dan koneksi database PostgreSQL (melalui Ping), terdaftar lengkap di Swagger UI.
timestamp: 2026-07-23 10:48 WITA
kategori: sedang
file yang diubah:
- backend/utils/logger.go
- backend/middleware/logger.go
- backend/handlers/health.go
- backend/config/database.go
- backend/main.go
- backend/go.sum
alasan perubahan: Memungkinkan audit operasional server melalui logging terstruktur yang tangguh dan menyediakan endpoint health check bagi load balancer / container orchestrator untuk memantau status kesehatan sistem.

perubahan yang dilakukan: Implementasi fitur Auto-Recovery (Proteksi Panic) dan Graceful Shutdown (Pemadaman Bersih) pada server.
before:
- Server rentan mati total (crash) jika terjadi panic tak terduga dalam penanganan rute HTTP.
- Penghentian server mematikan layanan secara kasar (abrupt), berpotensi memutus transaksi database menggantung dan request aktif.
after:
- Ditambahkan kustom `middleware.Recovery()` yang menangkap panic runtime, merekam stack trace menggunakan Zap Logger, dan membalas dengan respon JSON internal server error yang seragam.
- Menyusun alur shutdown di `main.go` yang menunggu OS Signals (SIGINT/SIGTERM), membatasi waktu penutupan request aktif (10 detik), dan memutus koneksi database PostgreSQL secara aman (`sqlDB.Close()`).
- Menyediakan endpoint pengujian sementara `GET /api/test-panic` untuk menguji fungsionalitas auto-recovery.
timestamp: 2026-07-23 11:08 WITA
kategori: sedang
file yang diubah:
- backend/main.go
alasan perubahan: Melindungi operasional server agar tetap menyala (resilient) saat terjadi runtime crash tak terduga, dan menjaga kebersihan siklus memori/database saat server dihentikan.

perubahan yang dilakukan: Pemindahan handler `test-panic` ke package-level function agar terdeteksi oleh generator Swagger UI.
before:
- Rute `test-panic` didaftarkan menggunakan anonymous inline function di dalam `main()`, sehingga diabaikan oleh swag CLI.
after:
- Rute `test-panic` menggunakan rujukan fungsi deklarasi `TestPanic` di tingkat package pada `main.go`.
timestamp: 2026-07-23 11:22 WITA
kategori: low
file yang diubah:
- backend/main.go
alasan perubahan: Memungkinkan swag parser mengenali anotasi komentar di atas fungsi handler agar rute uji coba panic tampil di antarmuka Swagger UI.

perubahan yang dilakukan: Pengamanan API menggunakan Rate Limiting (Global & Khusus Login).
before:
- Seluruh endpoint API tidak memiliki batasan pemanggilan, rentan terhadap serangan DDoS ringan.
- Rute login `/api/auth/login` tidak terproteksi dari serangan brute force (percobaan tebakan password tanpa henti).
after:
- Dibuat berkas `middleware/limiter.go` berisi `GlobalLimiter` (maksimal 100 request/menit per IP) dan `AuthLimiter` (maksimal 5 request/menit per IP).
- Menyematkan `GlobalLimiter` pada grup rute `/api` di `main.go`.
- Menyematkan `AuthLimiter` khusus pada rute POST `/auth/login` di `routes/auth.go`.
- Format respons terstandarisasi mengembalikan JSON error dan HTTP Status `429 Too Many Requests`.
timestamp: 2026-07-23 11:39 WITA
kategori: sedang
file yang diubah:
- backend/middleware/limiter.go
- backend/routes/auth.go
- backend/main.go
- backend/go.sum
alasan perubahan: Melindungi sumber daya komputasi server backend dari eksploitasi berlebihan dan memblokir upaya pembobolan kredensial pengguna secara otomatis melalui rate limiting dinamis berbasis IP.

perubahan yang dilakukan: Penyesuaian ekstraksi data list aset dan log pemeliharaan di frontend akibat perubahan format paginasi API.
before:
- Kode React langsung memasukkan `response.data` (yang bertipe object dengan properti `rows` dan `pagination`) ke state `assets` dan `logs`.
- Ini memicu error runtime `assets.filter is not a function` atau `assets.map is not a function` karena state bertipe object, bukan array.
after:
- Menambahkan pengecekan `Array.isArray(response.data) ? response.data : (response.data.rows || [])` di `src/app/page.tsx` dan `src/app/maintenance/page.tsx`.
- State terisi data array `rows` secara aman sehingga metode `.filter` dan `.map` berjalan lancar.
timestamp: 2026-07-23 11:45 WITA
kategori: sedang
file yang diubah:
- frontend/src/app/page.tsx
- frontend/src/app/maintenance/page.tsx
alasan perubahan: Menyelesaikan konflik format integrasi antara API terpaginasi baru dengan visualisasi data front-end Next.js.

perubahan yang dilakukan: Pembuatan Endpoint Statistik Dashboard untuk data agregat.
before:
- Frontend menghitung statistik (total aset, jumlah per status) secara lokal dari seluruh data array menggunakan `.filter().length`, tidak efisien untuk dataset besar.
- Tidak ada endpoint khusus yang menyediakan ringkasan data agregat langsung dari database.
after:
- Dibuat `handlers/dashboard.go` dengan endpoint `GET /api/dashboard/stats` yang mengembalikan: total aset, jumlah aset per status, total log pemeliharaan, log bulan ini (total/completed/pending), total pengguna, dan timestamp server.
- Rute dilindungi middleware `Protected()` (JWT) dan terdaftar lengkap di Swagger UI.
timestamp: 2026-07-23 14:21 WITA
kategori: sedang
file yang diubah:
- backend/handlers/dashboard.go
- backend/main.go
alasan perubahan: Menyediakan data statistik agregat dari sisi server untuk mengurangi beban komputasi frontend dan mempercepat render halaman dashboard.

perubahan yang dilakukan: Pembuatan Fitur Export Data Aset CCTV dan Log Pemeliharaan ke CSV.
before:
- Tidak ada opsi pengeksporan data tabular secara langsung dari server.
after:
- Ditambahkan fungsi `ExportAssets` di `handlers/asset.go` dan rute `GET /assets/export` yang mengunduh data seluruh aset terfilter ke dalam bentuk file CSV.
- Ditambahkan fungsi `ExportMaintenanceLogs` di `handlers/maintenance.go` dan rute `GET /maintenance-logs/export` yang mengunduh seluruh data log pemeliharaan terfilter beserta preloaded relasi data ke CSV.
- Kedua endpoint dilindungi autentikasi JWT dan terdaftar lengkap di dokumentasi Swagger UI.
timestamp: 2026-07-23 14:50 WITA
kategori: sedang
file yang diubah:
- backend/handlers/asset.go
- backend/handlers/maintenance.go
- backend/routes/maintenance.go
alasan perubahan: Menyediakan fitur pengeksporan data ke format CSV untuk kebutuhan audit data fisik dan pelaporan operasional sistem SIGAP.

perubahan yang dilakukan: Implementasi Database Connection Pooling dan Tuning di tingkat GORM/sql.DB.
before:
- Koneksi database PostgreSQL dibuka tanpa konfigurasi limitasi pool, berpotensi memicu kebocoran koneksi (connection leak) dan pemborosan sumber daya PostgreSQL.
after:
- Ditambahkan penyesuaian pool koneksi pada `config/database.go` menggunakan `SetMaxOpenConns`, `SetMaxIdleConns`, `SetConnMaxLifetime`, dan `SetConnMaxIdleTime`.
- Mendukung pemuatan parameter pooling secara dinamis melalui file konfigurasi `.env` dengan fallback nilai default aman.
- Mencetak laporan spesifikasi pool yang aktif ke dalam log terstruktur menggunakan Zap Logger pada saat inisialisasi server.
timestamp: 2026-07-23 15:46 WITA
kategori: sedang
file yang diubah:
- backend/config/database.go
alasan perubahan: Mencegah error "too many connections" dan mengoptimalkan efisiensi penanganan kueri konkuren di database PostgreSQL berskala industri.

perubahan yang dilakukan: Pembuatan Modul Manajemen Pengguna (User CRUD & Admin Controls) Terproteksi.
before:
- Hanya ada endpoint registrasi terbatas `/api/auth/users` dan tidak ada fungsi pengelolaan akun user yang komprehensif.
after:
- Dibuat `handlers/user.go` berisi handler GetAllUsers (paginasi & pencarian), GetUserByID, UpdateUser, DeleteUser (soft-delete), dan ResetUserPassword (penyetelan sandi paksa).
- Dibuat `routes/user.go` yang mendaftarkan seluruh endpoint di bawah grup rute `/api/users`.
- Rute didaftarkan di `main.go` dan dilindungi oleh middleware `Protected()` dan `RequireRole("admin")` untuk menjamin keamanan administratif.
- Seluruh endpoint terdokumentasi secara penuh di Swagger UI.
timestamp: 2026-07-24 09:10 WITA
kategori: sedang
file yang diubah:
- backend/handlers/user.go
- backend/routes/user.go
- backend/main.go
alasan perubahan: Melengkapi kebutuhan administrasi portal SIGAP agar admin dapat mengelola akun pengguna, mereset password, dan menonaktifkan pengguna dengan aman.

perubahan yang dilakukan: Konfigurasi Docker untuk Containerisasi Backend SIGAP dan PostgreSQL.
before:
- Tidak ada konfigurasi containerisasi; aplikasi hanya bisa dijalankan secara manual dengan PostgreSQL lokal.
after:
- Dibuat `Dockerfile` dengan Multi-stage Build (golang:alpine sebagai builder, alpine murni sebagai runner) untuk menghasilkan image produksi yang kecil dan aman.
- Dibuat `docker-compose.yml` dengan dua service: `db` (postgres:15-alpine dengan named volume `postgres_data`) dan `api` (build dari Dockerfile, env_file dari `.env.docker`, volume bind `./uploads:/app/uploads`).
- Dibuat `.env.docker` berisi variabel konfigurasi lingkungan Docker dengan `DB_HOST=db`.
- Dibuat `.dockerignore` untuk mengecualikan file tidak perlu dari proses build Docker.
- Service `api` menggunakan `depends_on` dengan kondisi `service_healthy` agar menunggu database siap sebelum API dimulai.
timestamp: 2026-07-24 09:26 WITA
kategori: sedang
file yang diubah:
- backend/Dockerfile
- backend/docker-compose.yml
- backend/.env.docker
- backend/.dockerignore
alasan perubahan: Memungkinkan rekan tim menjalankan seluruh stack backend SIGAP (API + PostgreSQL) dengan satu perintah `docker compose up --build` tanpa konfigurasi manual apapun.

perubahan yang dilakukan: Implementasi Lampiran Foto Pemeliharaan CCTV (File Upload & Auto-Delete).
before:
- Log pemeliharaan tidak memiliki kemampuan melampirkan bukti foto kegiatan pemeliharaan.
after:
- Ditambahkan kolom `image_url VARCHAR(255)` pada tabel `maintenance_logs` melalui penambahan field `ImageUrl` pada struct GORM `MaintenanceLog` di `models/setup.go`.
- Dibuat endpoint `POST /api/upload` (handler `handlers/upload.go`) yang menerima file multipart/form-data dengan validasi tipe konten (hanya JPEG/PNG/WebP), batas ukuran 2MB, dan penamaan berkas berbasis UUID untuk menghindari tabrakan nama file.
- Dimodifikasi `handlers/maintenance.go`: struct `CreateMaintenanceLogRequest` dan `UpdateMaintenanceLogRequest` menerima field `image_url`. Saat Update, file lama otomatis dihapus secara fisik jika diganti URL baru.
- Logika Auto-Delete diterapkan pada `DeleteMaintenanceLog`: sebelum record DB dihapus, sistem mengecek `image_url`, memvalidasi prefix path keamanan (`/uploads/`), lalu menghapus file fisik dari direktori menggunakan `os.Remove`.
- Ditambahkan `app.Static("/uploads", "./uploads")` di `main.go` agar file gambar yang diunggah dapat diakses publik via URL.
- Ditambahkan `os.MkdirAll("./uploads")` di startup `main.go` agar direktori dibuat otomatis jika belum ada.
- Diregistrasikan rute baru `POST /api/upload` di `routes/upload.go` di bawah proteksi middleware `Protected()` JWT.
timestamp: 2026-07-24 13:54 WITA
kategori: besar
file yang diubah:
- backend/models/setup.go
- backend/handlers/upload.go
- backend/handlers/maintenance.go
- backend/routes/upload.go
- backend/main.go
- backend/docs/docs.go
- backend/docs/swagger.json
- backend/docs/swagger.yaml
alasan perubahan: Memenuhi kebutuhan pencatatan bukti visual pemeliharaan perangkat CCTV SIGAP IKN, sekaligus menjaga kebersihan penyimpanan server dengan mekanisme Auto-Delete file yatim piatu.

perubahan yang dilakukan: Penyediaan Data Seeder CCTV Default dan Pembuatan QR Code Asset CCTV.
before:
- Tidak ada data CCTV default (seeder) saat inisialisasi database awal.
- Tidak ada endpoint untuk mengunduh QR Code fisik aset CCTV.
after:
- Ditambahkan fungsi `SeedAssets` di `models/seeder.go` yang otomatis menyuntikkan 4 aset CCTV dummy dengan koordinat dan IP address di area Jakarta saat pertama kali server diinisialisasi.
- Dipanggil `models.SeedAssets` di `main.go` setelah inisialisasi koneksi database.
- Ditambahkan pustaka `github.com/skip2/go-qrcode` untuk kebutuhan encoding string URL ke bentuk QR Code PNG.
- Diimplementasikan endpoint `GET /api/assets/:id/qrcode` (handler `GetAssetQRCode` di `handlers/asset.go`) yang mendeteksi UUID aset, mengecek kecocokan data di database, dan menghasilkan QR Code (PNG 256x256px) berisi URL detail/maintenance aset pada frontend (`{FRONTEND_URL}/maintenance?asset_id={id}`).
- Rute baru terproteksi JWT didaftarkan di `routes/asset.go`.
timestamp: 2026-07-24 14:46 WITA
kategori: sedang
file yang diubah:
- backend/main.go
- backend/models/seeder.go
- backend/handlers/asset.go
- backend/routes/asset.go
- backend/go.mod
- backend/go.sum
- backend/docs/docs.go
- backend/docs/swagger.json
- backend/docs/swagger.yaml
alasan perubahan: Memudahkan administrator menyebarkan kode QR fisik pada perangkat CCTV agar teknisi di lapangan tinggal memindai QR Code untuk langsung mencatat pemeliharaan aset CCTV yang bersangkutan.

perubahan yang dilakukan: Implementasi Master Data CCTV dan QR Code Terenkripsi AES-256 (GCM).
before:
- Tidak ada modul khusus untuk master data detail spesifikasi CCTV (general info).
- Scanner QR code CCTV fisik belum mendukung dekripsi payload aman berbasis enkripsi tingkat militer (AES-256).
after:
- Dibuat `backend/utils/encryption.go` berisi helper `EncryptAES256` dan `DecryptAES256` menggunakan AES-256-GCM terenkripsi dengan kunci simetris `AES_SECRET_KEY` dari `.env`.
- Dibuat model database `models.CCTV` (`backend/models/cctv.go`) yang mendata spesifikasi detail general CCTV (tanpa kredensial login) dan didaftarkan ke AutoMigrate di `main.go`.
- Diimplementasikan CRUD lengkap untuk CCTV di `backend/handlers/cctv.go`. Pada saat pembuatan (`CreateCCTV`), UUID perangkat dienkripsi menjadi string acak dan disimpan di kolom `QrPayload`.
- Dibuat endpoint scanner khusus `POST /api/cctv/scan` (`ScanCCTV` di `handlers/cctv.go`) yang mendekompresi payload QR terenkripsi, mencari kecocokan perangkat di database, dan mengembalikan data spesifikasi umum.
- Rute-rute baru dilindungi JWT dan didaftarkan di `backend/routes/cctv.go` serta dihubungkan ke `main.go`.
timestamp: 2026-07-24 15:06 WITA
kategori: besar
file yang diubah:
- backend/utils/encryption.go
- backend/models/cctv.go
- backend/handlers/cctv.go
- backend/routes/cctv.go
- backend/main.go
- backend/docs/docs.go
- backend/docs/swagger.json
- backend/docs/swagger.yaml
alasan perubahan: Memungkinkan inventarisasi detail CCTV secara mandiri tanpa membocorkan kredensial, serta meningkatkan keamanan operasional melalui pelabelan kode QR terenkripsi tingkat tinggi.

perubahan yang dilakukan: Standardisasi Keamanan Kredensial Environment (.env.example & .env.docker.example).
before:
- File `.env.docker` dilacak oleh Git (tidak di-ignore) dan berisi nilai kredensial langsung yang berpotensi bocor.
- Tidak ada file template `.env.example` untuk membantu panduan inisialisasi berkas `.env` lokal baru bagi developer.
after:
- Berkas `.env.docker` dipindahkan (`git mv`) menjadi berkas template `.env.docker.example` yang kredensialnya sudah disamarkan dengan placeholder aman.
- Dibuat berkas `.env.example` di dalam direktori `backend/` berisi panduan variabel environment lokal.
- Menambahkan aturan `.env.docker` ke dalam berkas `.gitignore` di root folder proyek agar tidak pernah ter-push secara tidak sengaja oleh developer.
timestamp: 2026-07-24 15:42 WITA
kategori: sedang
file yang diubah:
- .gitignore
- backend/.env.docker.example
- backend/.env.example
alasan perubahan: Mencegah kebocoran kredensial sensitif database dan kunci rahasia enkripsi ke dalam Git repositori dengan mengikuti standar industri.

perubahan yang dilakukan: Penyediaan Template & Fitur Impor Massal CCTV dari Excel.
before:
- Data CCTV hanya bisa ditambahkan satu per satu melalui form/API manual.
- Tidak tersedia template Excel standar untuk mempermudah migrasi data besar.
after:
- Ditambahkan library `github.com/xuri/excelize/v2` untuk mendukung pengolahan file Excel (.xlsx).
- Ditambahkan endpoint `GET /api/cctv/template` (handler `DownloadTemplateCCTV`) untuk mengunduh template kosong berisi header kolom terstandarisasi.
- Ditambahkan endpoint `POST /api/cctv/import` (handler `ImportCCTV`) yang menerima file multipart form data, membaca baris-baris data dari spreadsheet, menghasilkan UUID baru secara otomatis, mengenkripsi UUID tersebut menggunakan AES-256 menjadi `QrPayload`, dan menyimpannya secara transaksional ke database PostgreSQL.
- Rute-rute baru dilindungi JWT dan didaftarkan di `routes/cctv.go` (aman dari collision routing).
timestamp: 2026-07-24 16:33 WITA
kategori: sedang
file yang diubah:
- backend/go.mod
- backend/go.sum
- backend/handlers/cctv.go
- backend/routes/cctv.go
- backend/docs/docs.go
- backend/docs/swagger.json
- backend/docs/swagger.yaml
alasan perubahan: Memungkinkan migrasi dan inventarisasi data CCTV massal secara cepat, rapi, dan sesuai arsitektur enterprise dengan validasi data baris per baris.

perubahan yang dilakukan: Perbaikan Hook BeforeCreate GORM (Nil UUID Check).
before:
- Hook `BeforeCreate` pada model `Base` selalu menghasilkan UUID baru secara acak untuk kolom `ID` tanpa memedulikan apakah `ID` tersebut sudah diisi secara manual sebelumnya. Hal ini merusak sinkronisasi data QR Code terenkripsi (mismatch antara `ID` dan `QrPayload`).
after:
- Memodifikasi hook `BeforeCreate` agar mengecek kondisi `base.ID == uuid.Nil`. UUID baru hanya akan dihasilkan secara otomatis jika kolom `ID` kosong. ID kustom yang telah diisi sebelumnya (misalnya saat registrasi CCTV baru atau impor massal) akan dipertahankan dengan aman.
timestamp: 2026-07-24 17:34 WITA
kategori: sedang
file yang diubah:
- backend/models/setup.go
alasan perubahan: Mencegah bug mismatch ID di mana data CCTV tidak dapat diidentifikasi oleh QR scanner karena ID UUID di database ditimpa secara tidak sengaja oleh hook GORM.
