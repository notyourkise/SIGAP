# TESTING — Backend SIGAP

> Status pengujian backend dan cakupan lanjutan Fase 4.1.
> Diperbarui: 2026-08-07 (Task 7.1: test migrasi DB versioned; Task 7.2: blacklist & limiter Redis via interface mockable; Task 9.1: test `utils/rtsp_test.go` + field streaming pada `cctv_test.go`; Task 9.3: test `utils/mediamtx_test.go` + `handlers/stream_test.go` + `routes/stream_test.go`; Task 10.1+10.2: test `handlers/snapshot_test.go` — snapshot & timeline; Task 10.4: test `handlers/stream_health_test.go` — stream health & deteksi down).

---

## 1. Cara Menjalankan

```bash
cd backend
go build ./...        # build
go test ./... -count=1        # semua test (tanpa cache)
go test ./... -v              # verbose
go test ./handlers/ -run TestCCTV -v   # subset berdasarkan pola
go vet ./handlers/ ./middleware/ ./utils/
```

Semua test berjalan **tanpa PostgreSQL fisik** — memakai `github.com/DATA-DOG/go-sqlmock`.
Jalankan di direktori `backend/` (modul `sigap-backend`).

---

## 2. Matriks Cakupan Test

| Package | File Test | Yang Diuji |
|---------|-----------|-----------|
| `utils` | `pagination_test.go` | default limit/page/sort, cap limit 100 & page 1.000.000, whitelist sort (anti SQL injection), `CalculatePagination`, `ParseDateRangeParams` (valid, one-sided, invalid/missing) |
| `utils` | `encryption_test.go` | round-trip AES-256-GCM, nonce unik tiap enkripsi, hex invalid, input kosong |
| `utils` | `blacklist_test.go` | kontrak `TokenBlacklist` (add/check/remove) untuk varian Redis (mock `redisCommander`) & in-memory, TTL mengikuti token terpanjang, facade `RevokeToken`/`IsTokenRevoked`, cleanup expired, `SetTokenBlacklist` |
| `utils` | `csrf_test.go` | `GenerateCSRFToken` (64 hex, unik), `ExtractTokenString` (prioritas cookie → Bearer → kosong) |
| `utils` | `jwt_test.go` (lama + uniqueness) | generate & validate token, expiry, signature salah, klaim `jti`/`iat`, token selalu unik |
| `utils` | `validator_test.go` (lama) | validator custom (`strongpassword`, dll) |
| `utils` | `audit_test.go` | retry penyimpanan audit log setelah kegagalan sementara dan batas percobaan maksimum |
| `utils` | `notify_test.go` (Fase 5.1) | `persistNotification` (sukses, retry setelah kegagalan sementara, error setelah batas percobaan), guard clause `BroadcastToUser`/`BroadcastToRole` tanpa koneksi (tidak panic) |
| `middleware` | `auth_test.go` | `Protected`: tanpa token 401, revoked 401, invalid, valid, token-version mismatch 401 → `Sesi telah dicabut...`; `RequireRole`: none/wrong/allowed |
| `middleware` | `csrf_test.go` | metode aman dilewati, Bearer dilewati, tanpa cookie sesi dilewati, login/logout dengan sesi stale dilewati, mismatch/missing 403, matching lolos |
| `middleware` | `limiter_test.go` | batas bawah lolos, > 100/menit → 429, format key `limiter:{ip}:{endpoint}`, reset counter memory setelah window, isolasi counter per key |
| `middleware` | `security_test.go` | header dasar selalu ada; CSP/HSTS hanya saat `APP_ENV=production` |
| `handlers` | `auth_test.go` (lama + fix) | `Login` (sukses, password salah, user tidak ditemukan, validasi) + `SetupTestDB` dengan `SkipDefaultTransaction: true` |
| `handlers` | `auth_extra_test.go` | ChangePassword (sukses, password lama salah, validasi), GetMyProfile (sukses, not found, tanpa identitas), UpdateMyProfile, Logout idempotent (tanpa/invalid token), CreateUser (validasi, duplikat, sukses) |
| `handlers` | `user_test.go` | GetAllUsers, GetUserByID, UpdateUser (validasi role + token-version branch), DeleteUser (+ self-delete forbidden), ResetUserPassword |
| `handlers` | `cctv_test.go` | GetCCTVByID, CreateCCTV (validasi, duplikat kode, sukses + `qr_payload` terisi), ScanCCTV (invalid, sukses), DownloadTemplateCCTV, GetCCTVQRCode, ImportCCTV batas baris, Update/Delete sukses, not-found & invalid UUID, `cctvMockRowWithStatus` (varian status untuk test sinkronisasi), field `health_score` |
| `handlers` | `maintenance_test.go` | CreateMaintenanceLog (validasi, CCTV not found, sukses + reload), GetById, GetAll, Update/Delete sukses, not-found & invalid UUID, kepemilikan teknisi → 403, sinkronisasi status otomatis (Fase 5.2): complete 9/9 ok → CCTV `aktif`, complete gagal → CCTV `pemeliharaan`, complete pada CCTV `rusak`/`nonaktif` → status tidak berubah; assign/re-assign (Fase 5.3): assign normal → 200, re-assign log `ongoing` ke teknisi baru → 200, assign ke teknisi yang sama → 400, assign log `completed` → 400, re-assign oleh role `teknisi` → 403 (RequireRole admin) |
| `handlers` | `field_maintenance_test.go` | CreateFieldMaintenance (validasi, tanpa foto, CCTV not found, sukses), GetAll scope teknisi, GetById (invalid UUID, not found), Delete not found, sinkronisasi status otomatis (Fase 5.2): field-check `attention` → CCTV `pemeliharaan`, field-check `ok` → CCTV `aktif`; GenerateFieldMaintenanceReportPDF (content-type/prefix PDF, teknisi pemilik/non-pemilik, not found) |
| `handlers` | `dashboard_test.go` | DashboardTasks (tanpa identitas 401, teknisi ter-scope, admin), DashboardSummary (distribusi status, tren 6 bulan, overdue kosong), GetDailyDigests (admin paginasi/urutan terbaru) |
| `handlers` | `digest_generator_test.go` | GenerateDailyDigestAt (agregasi dan pembuatan digest), idempotensi tanggal, jadwal 06:00 WITA, notifikasi admin, dan role non-admin endpoint → 403 |
| `handlers` | `technician_performance_test.go` | Dashboard kinerja teknisi: struktur respons, filter rentang tanggal, kalkulasi rate/quality, dan role non-admin → 403 |
| `handlers` | `audit_test.go` | GetAuditLogs (sukses, filter modul), ExportAuditLogs CSV |
| `handlers` | `upload_test.go` | tanpa file 400, tipe invalid, JPEG valid (magic bytes), path traversal 400, ekstensi salah 400, not found 404 |
| `handlers` | `report_test.go` | PDF valid prefix `%PDF`, scope teknisi |
| `handlers` | `health_test.go` | HealthCheck Connected |
| `handlers` | `notification_test.go` (Fase 5.1) | GetNotifications (sukses + paginasi, filter unread, tanpa identitas 401), GetUnreadNotificationCount (sukses, tanpa identitas 401), MarkNotificationRead (sukses, milik orang lain 403, not found 404, UUID invalid 400), MarkAllNotificationsRead (sukses, tanpa identitas 401) |
| `handlers` | `overdue_checker_test.go` (Fase 5.5) | log overdue belum dinotifikasi → notifikasi admin + flag terisi, log ongoing → notifikasi teknisi, log sudah dinotifikasi → skip, status completed/failed → skip, reschedule ke masa depan → reset `overdue_notified_at` |
| `handlers` | `health_score_test.go` (Fase 6.1) | skor default tanpa riwayat, kombinasi lima faktor, ambang umur pemeliharaan, clamp 0, endpoint invalid UUID/sukses, persistence skor, dan kalkulator seluruh CCTV |
| `models` | `setup_test.go`, `seeder_test.go` | UUID `BeforeCreate`, `SeedAdmin` tidak mengubah akun admin yang sudah ada |
| `config` | `database_test.go`, `migrate_test.go` | DSN PostgreSQL dan resolusi legacy asset → CCTV |
| `config` | `versioned_migration_test.go` (Task 7.1) | embed FS migration (konvensi nama golang-migrate), kelengkapan 10 tabel di up/down, kolom kunci tiap tabel, CASCADE pada rollback FK sirkular, propagasi error driver (tanpa PostgreSQL) |
| `routes` | `routes_test.go`, `websocket_test.go` | Bootstrap route terlindungi, logout tanpa sesi tetap dapat membersihkan cookie, route CCTV/maintenance/field-maintenance/dashboard digest wajib autentikasi, handshake WebSocket tanpa token dan token valid |
| `handlers` | `helpers_test.go` | helper bersama: `withUserLocals`, `setupMockGORM`, `newUserMockRows`, `modelsUserRow`, `cctvMockRow`, `maintenanceMockRow` |
| `monitoring` | `metrics_test.go` (Task 7.4) | counter `sigap_http_requests_total{method,path,status}`, histogram kumulatif (bucket le + _sum/_count), HELP/TYPE untuk semua metrik, gauge (DB pool, WS aktif, upload storage), empty state, `Reset`, escape label, format angka tetap (tanpa eksponensial) |
| `middleware` | `metrics_test.go` (Task 7.4) | observasi request (counter & count), exclude endpoint `/api/metrics`, pencatatan status non-200, Content-Type `text/plain` |
| `handlers` | `health_test.go` (Task 7.4, diperluas) | HealthCheck Connected, field baru `db_response_time_ms`, `redis_status=Disabled`, `uploads{status,path,size_bytes}`, `dirSize` (folder tak ada → 0, jumlah ukuran file) |
| `utils` | `secrets_test.go` (Task 7.5) | `GetSecret`: file `/run/secrets` menang atas env var, fallback ke env bila file tak ada, kosong bila keduanya hilang, trim whitespace file |
| `utils` | `logger_test.go` (Task 7.6) | `parseLogLevel` (valid, invalid/empty → `info`, case-insensitive), `InitLogger` menghasilkan `Log` non-nil |
| `utils` | `push_test.go` (Task 8.3) | `SendPushToSubscription` meneruskan subscription/payload ke sender mock (VAPID), `SendPush` no-op bila VAPID kosong, kirim ke semua subscription user, format payload JSON |
| `handlers` | `push_test.go` (Task 8.3) | `SubscribePush` (create, upsert idempotent, validasi 400, unauthorized 401), `UnsubscribePush` (success, missing endpoint 400) |
| `handlers` | `field_maintenance_test.go` (Task 8.4, diperluas) | `CreateFieldMaintenance` geotag: `check_latitude` di luar rentang → 400, `check_longitude` bukan angka → 400, geotag valid → 201 + koordinat tersimpan |
| `utils` | `rtsp_test.go` (Task 9.1) | `BuildRTSPURL` (sub/main stream — channel id `channel_port × 100 + stream_type`), `BuildStreamPath`, `GetRTSPPort` default, edge cases (channel 0/negatif, stream type tidak dikenal) |
| `handlers` | `cctv_test.go` (Task 9.1, diperluas) | `CreateCCTV`/`UpdateCCTV` menerima field streaming baru (`channel_port`, `rtsp_password`, `stream_enabled`); `rtsp_password` tidak muncul di response JSON (excluded) |
| `utils` | `mediamtx_test.go` (Task 9.3a) | `MediaMTXClient`: `AddPath`/`RemovePath`/`GetPath`/`ListPaths` (sukses 2xx, parsing body JSON, mapping path info), error/edge cases via `httptest` mock (status 404, respons non-2xx, body tidak valid, server error) |
| `handlers` | `stream_test.go` (Task 9.3b) | `StartStream` (sukses → return `whep_url`; CCTV tidak ditemukan 404; `stream_enabled=false` → 400; tanpa `channel_port` → 400), `StopStream` (sukses → path dihapus), `GetStreamStatus` (list dengan `stream_active`/`stream_ready`/`viewers`), `GetStreamURL` (return `whep_url` + `hls_url`), enforcement role (`RequireRole("admin")` — teknisi → 403) |
| `routes` | `stream_test.go` (Task 9.3b) | registrasi route `/api/streams` (grup admin-only: `Protected()` + `RequireRole("admin")`) — rute statis (`/status`, `/start-all`, `/stop-all`) tersedia sebelum dinamis (`/:id/start`, `/:id/stop`, `/:id/url`); endpoint stream wajib autentikasi |
| `handlers` | `snapshot_test.go` (Task 10.1+10.2, Fase 10) | `GetCCTVSnapshot` (sukses → JPEG, belum ada snapshot → 404, CCTV not found → 404, tanpa identitas → 401), `GetCCTVTimeline` (sukses + paginasi, CCTV not found → 404, tanpa identitas → 401), rotasi snapshot (hanya menyimpan 10 terbaru) |
| `handlers` | `stream_health_test.go` (Task 10.4, Fase 10) | `GetStreamHealth` (tanpa stream, unauthorized 401/403), `RunStreamHealthMonitor` (no MediaMTX, down > 15 menit → `STREAM_DOWN` + health -10, pulih → `STREAM_RESTORED`) |

**Total: 50 file test** (13 utils, 5 middleware, 23 handlers, 2 models, 3 config, 3 routes, 1 monitoring, termasuk helper).

### Matriks Streaming (Fase 9)

Ringkasan cakupan test streaming CCTV real-time:

| File Test | Task | Yang Diuji |
|-----------|------|-----------|
| `utils/rtsp_test.go` | 9.1 | `BuildRTSPURL` (sub/main stream — channel id `channel_port × 100 + stream_type`), `BuildStreamPath`, `GetRTSPPort` default, edge cases (channel 0/negatif, stream type tidak dikenal) |
| `utils/mediamtx_test.go` | 9.3a | `MediaMTXClient`: `AddPath`/`RemovePath`/`GetPath`/`ListPaths` (sukses 2xx, parsing body JSON, mapping path info), error/edge cases via `httptest` mock (status 404, respons non-2xx, body tidak valid, server error) |
| `handlers/stream_test.go` | 9.3b | `StartStream` (sukses/404/disabled/no-channel-port), `StopStream`, `GetStreamStatus`, `GetStreamURL`, enforcement `RequireRole("admin")` — teknisi → 403 |
| `routes/stream_test.go` | 9.3b | registrasi route `/api/streams` admin-only; rute statis sebelum dinamis; endpoint stream wajib autentikasi |
| `handlers/cctv_test.go` (diperluas) | 9.1 | `CreateCCTV`/`UpdateCCTV` menerima field streaming baru (`channel_port`, `rtsp_password`, `stream_enabled`); `rtsp_password` tidak muncul di response JSON (excluded) |

### Matriks Snapshot & Timeline (Fase 10)

Ringkasan cakupan test snapshot & timeline:

| File Test | Task | Yang Diuji |
|-----------|------|-----------|
| `handlers/snapshot_test.go` | 10.1 + 10.2 | `GetCCTVSnapshot` (sukses → JPEG, belum ada snapshot → 404, CCTV not found → 404, tanpa identitas → 401), `GetCCTVTimeline` (sukses + paginasi, CCTV not found → 404, tanpa identitas → 401), rotasi snapshot (hanya menyimpan 10 terbaru) |

### Matriks Stream Health & Deteksi Status Stream (Fase 10, Task 10.4)

Ringkasan cakupan test stream health & deteksi down:

| File Test | Task | Yang Diuji |
|-----------|------|-----------|
| `handlers/stream_health_test.go` | 10.4 | `GetStreamHealth` (tanpa stream → respons kosong/valid; unauthorized tanpa identitas → 401; role non-admin → 403), `RunStreamHealthMonitor` (MediaMTX tidak tersedia → tidak panic/tidak ada penalti; stream down > 15 menit → broadcast `STREAM_DOWN` + `health_score` turun 10; stream pulih → broadcast `STREAM_RESTORED`) |

---

## 3. Gotchas Penting (Wajib Dibaca Sebelum Menambah Test)

1. **`SkipDefaultTransaction: true`** — `SetupTestDB` di `handlers/auth_test.go` memakai
   `gorm.Config{SkipDefaultTransaction: true}`. Tanpa ini GORM memanggil `BEGIN` tak terduga di
   tiap operasi tulis sehingga sqlmock gagal (`call to QueryRow 'BEGIN' was not expected`).
   Salin pola ini untuk semua handler yang menulis ke DB.

2. **Nama kolom selalu ber-quote di Postgres/GORM** — contoh: `SELECT "token_version" FROM "users"`.
   Regex sqlmock harus memakai bentuk ber-quote (`"users"`, `"token_version"`), bukan bentuk polos.

3. **Tipe file & ekstensi menipu** — upload memvalidasi magic bytes (bukan ekstensi). Test JPEG valid
   memakai byte `\xFF\xD8\xFF` dengan nama `.txt` → tetap diterima. Template xlsx diuji lewat
   `excelize.OpenReader` (file xlsx ter-zip, tidak bisa diassert byte mentah).

4. **Format keluaran yang di-assert test:**
   - CSV export audit dimulai header `ID,Waktu,Username`.
   - Template Excel: A1 `Kode CCTV`, C1 `Nama CCTV`.
   - `/cctv/scan` menerima `qr_payload` terenkripsi AES-256-GCM berisi UUID.
   - PDF dimulai prefix `%PDF`.

5. **Nama rute yang dipakai test (Fiber, tanpa prefix `/api`):** `cctv/:id`, `cctv/template`,
   `cctv/scan`, `maintenance-logs`, `auth/users`, `users/:id/reset-password`, `dashboard/tasks`,
   `dashboard/summary`, `dashboard/digest`, `audit-logs/export`.

6. **`APP_ENV`** — header CSP/HSTS hanya aktif saat `APP_ENV=production`; set `t.Setenv("APP_ENV", ...)`
   bila menguji variasi environment. Jangan sampai env mesin pengembang bocor ke assertion.

7. **UUID** — semua ID memakai UUID. `modelsUserRow`/`cctvMockRow`/`maintenanceMockRow` di
   `helpers_test.go` menyediakan row konstan; pakai `modelsUserRow(u, pwdHash)` dengan UUID valid.

---

## 4. Hal yang Ditunda (Tahap Berikutnya)

Prioritas lanjutan:

1. **Import CCTV transaksi penuh:** test file valid yang mengeksekusi insert, commit, duplicate, dan rollback dengan sqlmock.
2. **WebSocket revocation:** test token version mismatch saat handshake.
3. **Migration data penuh:** test `MigrateMaintenanceToCCTV` sampai update `maintenance_logs.cctv_id`.

Referensi perencanaan: `docs/ROADMAP.md` Fase 4 (task 4.1–4.8). Catat setiap penambahan di
`backend/changelog.md` dengan format before/after + timestamp WITA.
