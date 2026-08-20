# TESTING — Backend SIGAP

> Status pengujian backend dan cakupan lanjutan Fase 4.1.
> Diperbarui: 2026-08-20 (Task 1A: regression lifecycle QR memastikan rotasi tetap menunjuk ID CCTV yang sama, scanner menolak payload lama, dan service worker tidak menyimpan/menyajikan QR atau respons `no-store`). Sebelumnya diperbarui 2026-08-13 (Sprint 40: AI Feature Takedown — matriks & gotchas `handlers/ai_test.go` dihapus karena test AI detection di-remove dari codebase; total file test 65 → 64).

## Tata Kelola Manual/Black-box

Bagian ini adalah kontrak formal untuk seluruh hasil pada `docs/manual-test-*.md`. Hasil sebelum **2026-08-20** adalah catatan historis/non-authoritative: pertahankan observasi yang berguna, tetapi jangan menganggapnya sebagai active run sampai direkam ulang dengan metadata dan evidence lengkap.

### Vocabulary dan aturan status

Hanya nilai status yang case-sensitive berikut yang boleh dipakai sebagai status test atau suite:

- `PASS`: scenario executed against the recorded artifact/environment, actual equals expected, and an evidence path/link is recorded.
- `FAIL`: scenario executed and actual differs from expected.
- `BLOCKED`: execution was attempted but an external prerequisite prevented reaching the assertion; blocker evidence is recorded.
- `NOT RUN`: no conforming runtime execution exists. Code inspection, unit-test review, or an old checkmark without complete metadata maps here, not to `PASS`.

Sebelum eksekusi, suite berstatus `NOT RUN`. Setelah run dimulai, satu mandatory scenario yang bukan `PASS` membuat kesimpulan suite `FAIL`, termasuk bila scenario tersebut `BLOCKED` atau `NOT RUN`. Inspeksi kode dan unit test hanya boleh dicatat sebagai supporting evidence; keduanya bukan bukti black-box untuk `PASS`.

Untuk catatan historis yang dipertahankan: failure runtime yang eksplisit dipetakan menjadi `FAIL` saat direkam ulang dengan kontrak formal; kegagalan prerequisite eksternal dipetakan menjadi `BLOCKED` bila blocker evidence tersedia; code inspection, unit-test review, checkmark lama, atau metadata/evidence yang hilang dipetakan menjadi `NOT RUN`.

### Aggregate formal row untuk baseline Wave 0

Untuk baseline Wave 0, suite yang saat ini memiliki satu aggregate formal row memakai baris tunggal tersebut sebagai **sole atomic suite-level acceptance gate**. Skenario legacy terperinci di bawahnya tetap non-formal dan historical/non-authoritative sampai diperluas selama Wave 7. Aggregate formal row tetap wajib memiliki expected, actual, evidence, mandatory flag, dan status persis dari vocabulary di atas.

### Metadata eksekusi wajib

Setiap execution record wajib memuat seluruh field berikut: test ID, owner, build SHA, artifact digest/ID, tester, timestamp WITA, environment, fixture prefix/data, cleanup result, expected, actual, evidence path/link, mandatory flag, scenario status, dan suite status. Nilai yang belum tersedia ditulis `NOT RECORDED`; status yang belum dijalankan ditulis `NOT RUN`.

### Release gate dan production safety gate

Release hanya boleh menyimpulkan suite berhasil bila setiap mandatory scenario berstatus `PASS` dengan expected, actual, dan evidence yang dapat ditelusuri.

Destructive SQL, key regeneration/rotation, QR rotation, deletion, dan credential mutation **dilarang pada production**. Prosedur hanya boleh dijalankan bila environment bukan production dan target bukan `nexaspace.site`, fixture prefix `E2E-MTNN-` serta exact targets sudah dicatat, backup/recovery terverifikasi untuk perubahan data/key state, dan cleanup/rollback sudah didefinisikan serta akan dicatat. Bila salah satu prasyarat itu tidak tersedia, hentikan prosedur dan catat scenario sebagai `BLOCKED` beserta blocker evidence. Jangan menyalin credential, token, private key, RTSP password, atau payload production yang belum disanitasi.

### Reusable Markdown execution-record template

```markdown
### Formal execution metadata

| Field | Value |
|---|---|
| Test ID / Audit ID | MT-NN |
| Capability owner | NOT RECORDED |
| Build SHA | NOT RECORDED |
| Artifact digest/ID | NOT RECORDED |
| Tester | NOT RECORDED |
| Timestamp WITA | NOT RECORDED |
| Environment | NOT RECORDED |
| Fixture prefix/data | E2E-MTNN- / NOT RECORDED |
| Cleanup result | NOT RUN |
| Expected | NOT RECORDED |
| Actual | NOT RECORDED |
| Evidence path/link | NOT RECORDED |
| Mandatory flag | YES |
| Scenario status | NOT RUN |
| Status suite | NOT RUN |

### Formal scenario record

| ID | Mandatory | Expected | Actual | Evidence | Status |
|---|---|---|---|---|---|
| MT-NN-S1 | YES | NOT RECORDED | NOT RECORDED | NOT RECORDED | NOT RUN |

Suite conclusion: sebelum eksekusi `NOT RUN`; setelah run dimulai, setiap mandatory scenario harus `PASS`, atau suite `FAIL`.
```

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

Regression test service worker dijalankan terpisah dari direktori frontend:

```bash
cd frontend
npm run test:sw
```

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
| `handlers` | `cctv_test.go` | GetCCTVByID, CreateCCTV (validasi, duplikat kode, sukses + `qr_payload` terisi), ScanCCTV (invalid, payload aktif sukses, payload lama setelah rotasi ditolak), RotateQRCode (ciphertext baru tetap mendekripsi ke ID CCTV yang sama), GetCCTVQRCode (`Cache-Control: no-store`), DownloadTemplateCCTV, ImportCCTV batas baris, Update/Delete sukses, not-found & invalid UUID, `cctvMockRowWithStatus` (varian status untuk test sinkronisasi), field `health_score` |
| `frontend` | `tests/service-worker-cache.test.mjs` | Request gambar QR selalu network-only tanpa cache fallback; respons API dengan `Cache-Control: no-store` tidak ditulis ke runtime cache service worker |
| `handlers` | `maintenance_test.go` | CreateMaintenanceLog (validasi, CCTV not found, sukses + reload), GetById, GetAll, Update/Delete sukses, not-found & invalid UUID, kepemilikan teknisi → 403, sinkronisasi status otomatis (Fase 5.2): complete 9/9 ok → CCTV `aktif`, complete gagal → CCTV `pemeliharaan`, complete pada CCTV `rusak`/`nonaktif` → status tidak berubah; assign/re-assign (Fase 5.3): assign normal → 200, re-assign log `ongoing` ke teknisi baru → 200, assign ke teknisi yang sama → 400, assign log `completed` → 400, re-assign oleh role `teknisi` → 403 (RequireRole admin) |
| `handlers` | `field_maintenance_test.go` | CreateFieldMaintenance (validasi, tanpa foto, CCTV not found, sukses), GetAll scope teknisi, GetById (invalid UUID, not found), Delete not found, sinkronisasi status otomatis (Fase 5.2): field-check `attention` → CCTV `pemeliharaan`, field-check `ok` → CCTV `aktif`; GenerateFieldMaintenanceReportPDF (content-type/prefix PDF, teknisi pemilik/non-pemilik, not found) |
| `handlers` | `dashboard_test.go` | DashboardTasks (tanpa identitas 401, teknisi ter-scope, admin), DashboardSummary (distribusi status, tren 6 bulan, overdue kosong), GetDailyDigests (admin paginasi/urutan terbaru) |
| `handlers` | `digest_generator_test.go` | GenerateDailyDigestAt (agregasi dan pembuatan digest), idempotensi tanggal, jadwal 06:00 WITA, notifikasi admin, dan role non-admin endpoint → 403 |
| `handlers` | `technician_performance_test.go` | Dashboard kinerja teknisi: struktur respons, filter rentang tanggal, kalkulasi rate/quality, role non-admin → 403; `TestTechnicianPerformanceWithPeriod` (Task 13.18, Sprint 36): `?period=7d` → query 5 argumen + sukses 200 |
| `handlers` | `audit_test.go` | GetAuditLogs (sukses, filter modul), ExportAuditLogs CSV |
| `handlers` | `upload_test.go` | tanpa file 400, tipe invalid, JPEG valid (magic bytes), path traversal 400, ekstensi salah 400, not found 404; `TestUploadCompress` (Task 13.13, Sprint 35): gambar besar → terkompresi (header `X-Original-Size`/`X-Compressed-Size`), thumbnail terbuat, cleanup `defer os.RemoveAll` |
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
| `handlers` | `stream_health_test.go` (Task 10.4 + 14A.3, Fase 10/14) | `GetStreamHealth` (tanpa stream, unauthorized 401/403), `RunStreamHealthMonitor` (no MediaMTX, down > 15 menit → `STREAM_DOWN` + health -10, pulih → `STREAM_RESTORED`); Task 14A.3: deteksi down langsung saat `stream_live=false` hasil probe (tanpa menunggu viewer) — penghapusan guardian `streamLastReady` |
| `handlers` | `spare_part_test.go` (Task 11.1, Sprint 26) | `CreateSparePart` (sukses, SKU duplikat → 400), `GetAllSpareParts` (paginasi), `GetSparePartByID` (sukses, not found → 404), `UpdateSparePart` (sukses), `DeleteSparePart` (soft delete), `RecordSparePartUsage` (stok berkurang, stok tidak cukup → 400, log bukan completed → 409), `GetSparePartUsageByLog`, `GetLowStockSpareParts` |
| `handlers` | `email_test.go` (Task 11.3, Sprint 27) | `UpdateEmailPreferences` (sukses → simpan, field wajib hilang → 400, user not found → 404) |
| `handlers` | `report_builder_test.go` (Task 11.2, Sprint 27) | `GenerateReport` (CSV sukses, entitas tak dikenal → 400, tanpa kolom valid → 400), `SaveReportTemplate` (sukses, config bukan JSON → 400), `ListReportTemplates` (sukses), `DeleteReportTemplate` (sukses, not found → 404) |
| `utils` | `email_test.go` (Task 11.3, Sprint 27) | `EmailEnabled` (false bila SMTP kosong, true bila terisi), `SendEmail` no-op error bila nonaktif, `EncodeHeader` (RFC 2047 untuk non-ASCII), `RenderEmailTemplate` (merender template bila tersedia), `RenderStreamDownTemplate` (template CCTV offline) |
| `handlers` | `email_test.go` (Task 11.3, diperluas) | `SendCCTVStreamDownEmail` (no-op tanpa panic bila SMTP nonaktif / db nil) |
| `handlers` | `zone_test.go` (Task 11.4, Sprint 28) | `CreateZone` (sukses → 201, validasi nama kosong → 400), `GetAllZones` (paginasi + `cctv_count`/`user_count`), `GetZoneByID` (sukses, not found → 404), `UpdateZone` (sukses), `DeleteZone` (soft delete), `AssignCCTVToZone` (bulk-assign sukses), `AssignUsersToZone` (sukses, user bukan teknisi → 400), `GetAllZonesForSelect` |
| `handlers` | `sla_test.go` (Task 11.5, Sprint 28) | `CreateSLAPolicy` (sukses → 201, jenis_tugas tidak valid → 400), `GetSLAPolicies` (paginasi), `GetSLAPolicyByID` (sukses, not found → 404), `UpdateSLAPolicy` (sukses), `DeleteSLAPolicy` (soft delete), `GetDashboardSLA` (sukses → `monthly_compliance`/`violations`/`technician_ranking`, teknisi → 403), `CheckSLADeadlines` (tanpa panic tanpa tugas) |
| `routes` | `routes_test.go` (Sprint 28, diperluas) | `TestZoneRoutesRequireAuthentication`, `TestSLAPoliciesRoutesRequireAuthentication`, registrasi `SetupZoneRoutes`/`SetupSLARoutes` pada `TestSetupAllRoutesRegistersProtectedEndpoints` |
| `handlers` | `api_key_test.go` (Task 12.1, Sprint 29) | `CreateAPIKey` (sukses → 201 + key mentah 1×, scope invalid → 400, validasi → 400), `GetAllAPIKeys` (paginasi), `RevokeAPIKey` (sukses, not found → 404), `GenerateAPIKeyFormat` (key 64 hex + prefix 8 + bcrypt verify) |
| `handlers` | `webhook_test.go` (Task 12.3, Sprint 29) | `CreateWebhookEndpoint` (sukses → 201, event invalid → 400, validasi → 400), `GetAllWebhookEndpoints` (paginasi), `UpdateWebhookEndpoint` (sukses), `DeleteWebhookEndpoint` (soft delete), `GetWebhookDeliveries` (paginasi), `SignPayload` (HMAC-SHA256 hex 64), `ValidWebhookEvent` |
| `middleware` | `apikey_test.go` (Task 12.1, Sprint 29) | `APIKeyAuth` (tanpa header → 401, key tidak valid → 401, key valid + scope → 200 + locals terisi, scope kurang → 403) |
| `routes` | `routes_test.go` (Sprint 29, diperluas) | `TestAPIKeyRoutesRequireAuthentication`, `TestWebhookRoutesRequireAuthentication`, registrasi `SetupAPIKeyRoutes`/`SetupWebhookRoutes` pada `TestSetupAllRoutesRegistersProtectedEndpoints` |
| `handlers` | `site_test.go` (Task 12.2, Sprint 30) | `CreateSite` (sukses → 201, validasi kode/nama kosong → 400), `GetAllSites` (paginasi + `is_default`), `GetAllSitesForSelect` (dropdown `GET /api/sites/all`), `UpdateSite` (sukses), `DeleteSite` (soft delete), `DeleteSite` site default → ditolak, `AssignUsersToSite` (sukses → relasi dalam transaksi, user bukan teknisi → 400) |
| `middleware` | `site_test.go` (Task 12.2, Sprint 30) | guard penulisan site via `RequireRole("admin")` — teknisi → 403; pembacaan site tersedia untuk semua role login; endpoint site wajib autentikasi (401 tanpa sesi) |
| `handlers` | `checklist_template_test.go` (Task 13.12, Sprint 35) | `CreateChecklistTemplate` (sukses → 201, validasi nama kosong → 400, default wajib/aktif), `GetAllChecklistTemplates` (paginasi + filter `?aktif=`), `UpdateChecklistTemplate` (sukses), `DeleteChecklistTemplate` (soft delete → `aktif=false`), `ReorderChecklistTemplates` (batch update urutan), guard role (teknisi → 403) |
| `handlers` | `stream_probe_test.go` (Task 14A, Sprint 37) | `ProbeRTSPHost` (host reachable via listener TCP → true, port tak terpakai → false), `ProbeRTSPChannel` (response RTSP/1.0 200 → true, koneksi gagal → false) — via `startTestRTSPListener` mock; (catatan: `utils/rtsp_probe_test.go` untuk `RunRTSPProber`/`GetLiveStatus` dijadwalkan pada Task 14A) |

**Total: 64 file test** (14 utils, 7 middleware, 33 handlers, 2 models, 3 config, 4 routes, 1 monitoring, termasuk helper).

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
| `handlers/stream_health_test.go` | 10.4 + 14A.3 | `GetStreamHealth` (tanpa stream → respons kosong/valid; unauthorized tanpa identitas → 401; role non-admin → 403), `RunStreamHealthMonitor` (MediaMTX tidak tersedia → tidak panic/tidak ada penalti; stream down > 15 menit → broadcast `STREAM_DOWN` + `health_score` turun 10; stream pulih → broadcast `STREAM_RESTORED`); Task 14A.3: deteksi down langsung via `stream_live=false` tanpa menunggu viewer (guardian `streamLastReady` dihapus) |

### Matriks Inventaris Suku Cadang (Fase 11, Task 11.1 / Sprint 26)

Ringkasan cakupan test inventaris suku cadang:

| File Test | Task | Yang Diuji |
|-----------|------|-----------|
| `handlers/spare_part_test.go` | 11.1 | `CreateSparePart` (sukses → 201, SKU duplikat → 400), `GetAllSpareParts` (paginasi + filter), `GetSparePartByID` (sukses, not found → 404), `UpdateSparePart` (sukses), `DeleteSparePart` (soft delete → `deleted_at`), `RecordSparePartUsage` (sukses → stok berkurang + create usage dalam transaksi, stok tidak cukup → 400 tanpa mengubah stok, log bukan completed → 409), `GetSparePartUsageByLog` (list penggunaan + preload suku cadang), `GetLowStockSpareParts` (filter `stok < minimum_order`) |

Catatan gotchas test `spare_part_test.go`:
- `RecordSparePartUsage` memakai `config.DB.Transaction`, sehingga test wajib menambah
  `mock.ExpectBegin()` / `mock.ExpectCommit()` (sukses) atau `mock.ExpectRollback()` (gagal) di
  sekitar query transaksi — lihat pola `field_maintenance_test.go`.
- `LogAudit` ditulis async di goroutine; jangan mem-mock query user/audit di test success (pola sama
  dengan `maintenance_schedule_test.go`).

### Matriks Email & Report Builder (Fase 11, Task 11.2 + 11.3 / Sprint 27)

Ringkasan cakupan test email & report builder:

| File Test | Task | Yang Diuji |
|-----------|------|-----------|
| `handlers/email_test.go` | 11.3 | `UpdateEmailPreferences` (sukses → simpan & return, `email_notifications` wajib → 400, user tidak ditemukan → 404) |
| `handlers/report_builder_test.go` | 11.2 | `GenerateReport` (CSV sukses → content-type + filename, entitas tak dikenal → 400, tanpa kolom valid → 400), `SaveReportTemplate` (sukses → 201, config bukan JSON → 400), `ListReportTemplates` (sukses), `DeleteReportTemplate` (sukses → soft delete, not found → 404) |
| `utils/email_test.go` | 11.3 | `EmailEnabled` (false bila SMTP_HOST kosong, true bila terisi), `SendEmail` (return error bila nonaktif), `EncodeHeader` (ASCII apa adanya, non-ASCII → RFC 2047 `=?UTF-8?Q?...?=`), `RenderEmailTemplate` (render template bila folder tersedia), `RenderStreamDownTemplate` (template CCTV offline `stream_down.html`) |

Catatan gotchas:
- Test `utils/email_test.go` memakai `t.Setenv` + `InitEmailConfig()`; pastikan tidak meninggalkan env
  SMTP yang bocor antar-test (selalu set `SMTP_HOST` eksplisit).
- `RenderEmailTemplate` mencari folder template pada beberapa kandidat path
  (`templates/`, `../templates/`, `backend/templates/`, `../backend/templates/`) agar dapat dipanggil
  dari lokasi working directory yang berbeda (lokal vs container).

### Matriks Zona & SLA (Fase 11, Task 11.4 + 11.5 / Sprint 28)

Ringkasan cakupan test zona & SLA:

| File Test | Task | Yang Diuji |
|-----------|------|-----------|
| `handlers/zone_test.go` | 11.4 | `CreateZone` (sukses → 201, validasi nama kosong → 400), `GetAllZones` (paginasi + `cctv_count`/`user_count` per zona), `GetZoneByID` (sukses → 200, not found → 404), `UpdateZone` (sukses), `DeleteZone` (soft delete → `deleted_at`), `AssignCCTVToZone` (bulk-assign sukses → update `zone_id` dalam transaksi), `AssignUsersToZone` (sukses → hapus lama + sisipkan baru dalam transaksi, user bukan teknisi → 400), `GetAllZonesForSelect` |
| `handlers/sla_test.go` | 11.5 | `CreateSLAPolicy` (sukses → 201, `jenis_tugas` tidak valid → 400), `GetSLAPolicies` (paginasi), `GetSLAPolicyByID` (sukses, not found → 404), `UpdateSLAPolicy` (sukses), `DeleteSLAPolicy` (soft delete), `GetDashboardSLA` (sukses → `monthly_compliance`/`violations`/`technician_ranking`, teknisi → 403), `CheckSLADeadlines` (tanpa tugas → tidak panic) |
| `routes/routes_test.go` (diperluas) | 11.4 + 11.5 | `TestZoneRoutesRequireAuthentication`, `TestSLAPoliciesRoutesRequireAuthentication`, `SetupZoneRoutes`/`SetupSLARoutes` dalam `TestSetupAllRoutesRegistersProtectedEndpoints` |

Catatan gotchas test `zone_test.go` / `sla_test.go`:
- `LogAudit` ditulis async di goroutine; jangan mem-mock query user/audit di test success (pola sama
  dengan `maintenance_schedule_test.go` dan `spare_part_test.go`).
- `AssignCCTVToZone`/`AssignUsersToZone` memakai `config.DB.Transaction`, sehingga test wajib menambah
  `mock.ExpectBegin()` / `mock.ExpectCommit()` di sekitar query transaksi.
- `GetDashboardSLA` memakai query raw dengan `JOIN`, sehingga regex sqlmock harus mencocokkan
  `FROM maintenance_logs AS ml` (tanpa quote pada tabel yang di-join).

### Matriks API Key & Webhook (Fase 12, Task 12.1 + 12.3 / Sprint 29)

Ringkasan cakupan test API key & webhook:

| File Test | Task | Yang Diuji |
|-----------|------|-----------|
| `handlers/api_key_test.go` | 12.1 | `CreateAPIKey` (sukses → 201 + key mentah 1×, scope invalid → 400, validasi nama kosong → 400), `GetAllAPIKeys` (paginasi), `RevokeAPIKey` (sukses, not found → 404), `GenerateAPIKeyFormat` (key 64 hex, prefix 8, bcrypt verify) |
| `handlers/webhook_test.go` | 12.3 | `CreateWebhookEndpoint` (sukses → 201, event invalid → 400, validasi → 400), `GetAllWebhookEndpoints` (paginasi), `UpdateWebhookEndpoint` (sukses), `DeleteWebhookEndpoint` (soft delete), `GetWebhookDeliveries` (paginasi), `SignPayload` (HMAC-SHA256 hex 64), `ValidWebhookEvent` |
| `middleware/apikey_test.go` | 12.1 | `APIKeyAuth` (tanpa header → 401, key tidak valid → 401, key valid + scope → 200 + locals `api_key_id`/`auth_method`, scope kurang → 403) |

Catatan gotchas test `api_key_test.go` / `webhook_test.go`:
- Field jsonb `[]string` (`Scopes`/`Events`) memakai `serializer:json`; INSERT GORM menghasilkan
  `RETURNING "scopes"`/`"events"`, sehingga mock memakai `ExpectQuery` dengan `[]byte` JSON pada kolom tsb.
- `CreateAPIKey`/`RevokeAPIKey` memicu `LogAudit` async di goroutine — jangan mem-mock query user/audit
  pada test success (pola sama dengan test CRUD lain).
- Middleware API key menulis `last_used_at` async di goroutine — hindari `mock.ExpectationsWereMet()`
  pada test `ValidKey` (menunggu UPDATE async).

### Matriks Multi-Site (Fase 12, Task 12.2 / Sprint 30)

Ringkasan cakupan test multi-site:

| File Test | Task | Yang Diuji |
|-----------|------|-----------|
| `handlers/site_test.go` | 12.2 | `CreateSite` (sukses → 201, validasi nama/kode kosong → 400, kode duplikat → 400), `GetAllSites` (paginasi + `is_default`), `GetAllSitesForSelect` (dropdown `GET /api/sites/all`), `UpdateSite` (sukses), `DeleteSite` (soft delete → `deleted_at`), `DeleteSite` site default → 4xx (tidak bisa dihapus), `AssignUsersToSite` (sukses → relasi user↔site dalam transaksi, user bukan teknisi → 400) |
| `middleware/site_test.go` | 12.2 | guard penulisan site (`RequireRole("admin")`) — teknisi → 403; pembacaan site (`GET /api/sites`, `GET /api/sites/all`) tersedia untuk semua role login; endpoint site wajib autentikasi (401 tanpa sesi) |

Catatan gotchas test `site_test.go`:
- `CreateSite`/`DeleteSite`/`AssignUsersToSite` memicu `LogAudit` async di goroutine — jangan mem-mock query
  user/audit pada test success (pola sama dengan test CRUD lain).
- `AssignUsersToSite` memakai `config.DB.Transaction`, sehingga test wajib menambah `mock.ExpectBegin()` /
  `mock.ExpectCommit()` (sukses) atau `mock.ExpectRollback()` (gagal) di sekitar query transaksi.
- `DeleteSite` site default memakai guard sebelum query hapus — assert respons 4xx tanpa `ExpectExec` hapus.

### Matriks Infrastruktur & Integrasi (Fase 13, Task 13.10 + 13.14 + 13.16 / Sprint 34)

Ringkasan cakupan test Sprint 34:

| File Test | Task | Yang Diuji |
|-----------|------|-----------|
| `handlers/infra_integration_test.go` | 13.10/13.14/13.16 | `CleanupWebhookDeliveries` (query cleanup per endpoint), `AdminCleanupWebhookDeliveries` (sukses → 200 + `deleted`, not found → 404), `RotateQRCode` (sukses → 200 + `qr_payload` berubah, not found → 404), `RegenerateAPIKey` (sukses → 200 + `api_key` baru + prefix baru, not found → 404) |
| `handlers/webhook_test.go` | 13.10 | `GetAllWebhookEndpoints` (kini menyertakan query `delivery_counts`) |

Catatan gotchas test Sprint 34:
- `CleanupWebhookDeliveries` memakai query mentah `DELETE ... WHERE id IN (SELECT ... JOIN ...)`;
  mock hanya perlu `ExpectExec('^DELETE FROM webhook_deliveries')`.
- `RotateQRCode` memicu `LogAudit` async di goroutine — jangan mem-mock query user/audit pada test success.
- `RegenerateAPIKey` memicu `LogAudit` + `BroadcastEvent` async; hanya mock `SELECT * FROM "api_keys"` dan `UPDATE "api_keys"`.
- `webhookEndpointRows` kini menyertakan kolom `max_log_retention_days` (baris tabel bertambah satu kolom).

### Matriks RTSP Live Status Detection (Fase 14, Task 14A / Sprint 37)

Ringkasan cakupan test status live RTSP:

| File Test | Task | Yang Diuji |
|-----------|------|-----------|
| `handlers/stream_probe_test.go` | 14A.1 | `ProbeRTSPHost` (host reachable via listener TCP tiruan → true; port tak terpakai → false), `ProbeRTSPChannel` (response `RTSP/1.0 200` → true; koneksi gagal → false) — via helper `startTestRTSPListener` |
| `handlers/stream_health_test.go` (diperluas) | 14A.3 | deteksi down langsung saat `stream_live=false` hasil probe (tanpa menunggu viewer) — guardian `streamLastReady` dihapus |

Catatan gotchas test `stream_probe_test.go`:
- `ProbeRTSPHost`/`ProbeRTSPChannel` melakukan koneksi TCP nyata; di unit test gunakan listener TCP tiruan (`startTestRTSPListener`) di `127.0.0.1:0` agar deterministik dan tidak bergantung pada NVR eksternal.
- `ProbeRTSPChannel` memakai `bufio.Scanner` untuk membaca baris pertama respons; pastikan server tiruan menulis `RTSP/1.0 200 OK` lalu menutup koneksi.
- **Perkiraan Task 14A**: `utils/rtsp_probe_test.go` (untuk `RunRTSPProber`/`runRTSPProbeCycle` dan `GetLiveStatus` — update `stream_live`/`stream_checked_at`, broadcast `CCTV_LIVE_STATUS_UPDATED`, enforcement role) direncanakan dibuat agent backend. Jika telah dibuat, tambahkan baris matriksnya dan naikkan total file test (+1 → 64).

> [!NOTE]
> **Sprint 40 — AI Feature Takedown (2026-08-13):** Matriks & gotchas `handlers/ai_test.go` (Task 14B/14C.3) telah **dihapus** dari dokumen ini karena test AI detection di-remove dari codebase. Total file test turun dari 65 → 64.

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
