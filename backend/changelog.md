# Changelog - Backend SIGAP

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
- Kop surat resmi: logo IKN (`assets/nusantara-logo.png`, 691×996 px) di kiri, nama instansi "OTORITA IBU KOTA NUSANTARA" + sub "Sistem Identifikasi Gangguan Aset Pusat (SIGAP)", alamat lengkap (Kompleks Perkantoran Pemerintah IKN, Kab. Penajam Paser Utara, Provinsi Kalimantan Timur), website www.ikn.go.id (telp/email sebagai konstanta kosong yang siap diisi), ditutup garis ganda (tebal + tipis).
- Judul laporan diubah "LAPORAN PEMELIHARAAN CCTV"; baris "Sistem Identifikasi Gangguan Aset Pusat (SIGAP) IKN" yang lama dihapus karena sudah ada di kop.
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
