# Panduan Agent AI untuk Proyek SIGAP (Sistem Informasi & Inventaris CCTV Terpadu)

Selamat datang, Agent AI. Dokumen ini adalah panduan komprehensif mengenai codebase SIGAP. Bacalah dokumen ini secara saksama agar Anda dapat memahami arsitektur, konvensi, dan aturan proyek ini sebelum mulai berkontribusi.

> [!NOTE]
> Dokumen ini diperbarui menyusul Fase 2 (hardening keamanan: migrasi token ke cookie httpOnly + CSRF) dan Fase 3 (fitur: unifikasi model `assets` → `cctv`, dashboard, profil, laporan PDF, dll). Status terbaru setiap item perbaikan & pengembangan ada di `docs/ROADMAP.md` dan `docs/SECURITY-AUDIT.md`.

## 1. Gambaran Umum Sistem
- **Nama**: SIGAP (Sistem Informasi & Inventaris CCTV Terpadu)
- **Tujuan**: Platform web untuk manajemen, inventarisasi, dan pemeliharaan perangkat CCTV secara terpusat.
- **Arsitektur**: Monorepo dengan 3 komponen utama:
  - **Backend**: REST API yang dibangun menggunakan Go Fiber, berlokasi di direktori `backend/`.
  - **Frontend**: Aplikasi web yang dibangun dengan Next.js 16 dan React 19, berlokasi di direktori `frontend/`.
  - **Database**: PostgreSQL 15, dijalankan melalui Docker.
- **Deployment**: Menggunakan Docker Compose untuk orkestrasi container.
- **Akses web resmi**: Pada deployment Docker, aplikasi hanya diakses melalui `https://localhost`
  dengan Nginx sebagai reverse proxy. Port internal API/frontend tidak boleh diakses langsung;
  detail alur ada di `docs/ACCESS.md`.

## 2. Aturan Git yang WAJIB Dipatuhi (SANGAT PENTING)
> [!CAUTION]
> Branch `main` dilindungi. JANGAN pernah melakukan commit atau push langsung ke branch `main`.

Setiap pengembangan fitur atau perbaikan _bug_ HARUS menggunakan branch baru.

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
- `docker-compose.yml` — Orkestrasi 4 _service_ (db, api, frontend, nginx) di dalam _network_ `sigap_network`.
- `makefile` — Kumpulan _script_ untuk menjalankan backend dan frontend secara bersamaan di lingkungan lokal.
- `.gitignore` — Mengabaikan direktori seperti `node_modules`, file `.env`, hasil _build_ binary Go, dan folder _logs_.

### Backend `backend/`
- `main.go` — Titik masuk (entry point) aplikasi Go. Berisi inisialisasi _logger_, validasi secret wajib (fail-fast), koneksi database, _migrations_ + migrasi data legacy, _seeders_, instansiasi Fiber app (termasuk _Trusted Proxies_ dan CORS), _middleware stack_, pendaftaran _routes_, dan _graceful shutdown_.
- `config/database.go` — Konfigurasi koneksi ke PostgreSQL menggunakan GORM dengan pengaturan _connection pooling_.
- `config/migrate.go` — Migrasi data satu kali (_one-time_) yang meresolusi _maintenance log_ legacy `asset_id` → `cctv_id` (unifikasi model, Fase 3.6).
- `models/setup.go` — Definisi struktur `Base` (berisi UUID, timestamps, dan _soft delete_) yang di-_embed_ pada semua model, termasuk _hook_ `BeforeCreate` yang hanya men-generate UUID baru jika ID kosong (`uuid.Nil`). File ini juga mendefinisikan model `User` dan `MaintenanceLog`, termasuk flag nullable `OverdueNotifiedAt` untuk eskalasi overdue satu kali.
- `models/maintenance_schedule.go` — Definisi struktur `MaintenanceSchedule` (Fase 5.4, jadwal pemeliharaan berulang): `CCTVID`, `TechnicianID` (opsional), `ActionTaken`, `Frequency` (`daily`/`weekly`/`monthly`), `NextRunAt`, `IsActive`, `LastGeneratedAt` + konstanta `FrequencyDaily/Weekly/Monthly`.
- `models/notification.go` — Definisi struktur `Notification` (notifikasi in-app per-user, Fase 5.1): `UserID`, `Type`, `Title`, `Message`, `ReferenceID`, `ReferenceType`, `IsRead` + konstanta tipe (`MAINTENANCE_ASSIGNED`, `MAINTENANCE_REASSIGNED`, `MAINTENANCE_UNASSIGNED`, `MAINTENANCE_COMPLETED`, `MAINTENANCE_OVERDUE`, `FIELD_CHECK_ATTENTION`); index komposit `(user_id, is_read)`.
- `models/cctv.go` — Definisi struktur `CCTV`, termasuk `QrPayload` yang dienkripsi menggunakan AES-256 dan `HealthScore` (skor kesehatan 0–100, default 100).
- `models/audit_log.go` — Definisi struktur `AuditLog` (riwayat aktivitas penting pengguna).
- `models/seeder.go` — _Script_ untuk me-seed admin _default_ (hanya jika belum ada, tidak me-reset password tiap startup).
- `handlers/` — Berisi _handler_ untuk setiap modul: `auth`, `cctv` (termasuk endpoint skor kesehatan), `maintenance`, `user`, `dashboard`, `upload`, `audit`, `report`, `notification`, `health`, `maintenance_schedule` (CRUD jadwal berulang, Fase 5.4), `schedule_runner` (goroutine generator jadwal — berjalan sekali saat startup lalu tiap jam membuat `MaintenanceLog` otomatis, idempotent per hari per jadwal), `overdue_checker` (goroutine eskalasi tugas overdue — berjalan saat startup lalu tiap jam, satu kali per log sampai reschedule), dan `health_score` (kalkulator skor kesehatan CCTV saat startup lalu tiap 6 jam).
- `middleware/auth.go` — Fungsi `Protected()` untuk memvalidasi JWT token, pengecekan blacklist,
  validasi versi token, dan `RequireRole()` untuk pengecekan _Role-Based Access Control_ (RBAC).
- `middleware/limiter.go` — Pembatasan laju _request_ (Rate Limiter). Terdapat `GlobalLimiter` (100 req/menit) dan `AuthLimiter` (khusus _endpoint_ autentikasi, 5 req/menit).
- `middleware/csrf.go` — Perlindungan CSRF pola _double-submit_ untuk permintaan mutasi yang terautentikasi via cookie.
- `middleware/security.go` — _Security headers_ (CSP & HSTS aktif hanya di produksi).
- `middleware/logger.go` & `middleware/recovery.go` — Logging terstruktur (Zap) dan pemulihan dari _panic_.
- `routes/` — Registrasi rute untuk masing-masing modul (auth, cctv, maintenance, user, upload, audit, dashboard, notification, maintenance_schedule, websocket).
  > [!IMPORTANT]
  > **PENTING:** Rute statis (seperti `/template`, `/import`, `/bulk-delete`) HARUS dideklarasikan SEBELUM rute dinamis (seperti `/:id`) untuk menghindari _collision_ atau tertimpa oleh parameter Fiber.
- `utils/encryption.go` — Fungsi `EncryptAES256` dan `DecryptAES256` berbasis AES-256-GCM. Kunci diderivasi menggunakan _hash_ SHA-256 dari _environment variable_ `AES_SECRET_KEY`.
- `utils/jwt.go` — Fungsi untuk me-generate, memvalidasi JWT token (HMAC-SHA256, kedaluwarsa 5 jam, klaim unik `jti`/`iat`), dan memvalidasi versi token (`tv`) terhadap database.
- `utils/session.go` — Pengaturan cookie sesi httpOnly (`sigap_session`) & cookie CSRF (`sigap_csrf`), termasuk pembersihan path legacy `/api`.
- `utils/blacklist.go` — Penyimpanan token yang di-_revoke_ (_in-memory_, bersifat sementara sampai restart server).
- `utils/csrf.go` — Generate token CSRF dan ekstraksi token sesi (prioritas cookie → Bearer).
- `utils/audit.go` — Pencatatan audit log secara _async_ dengan retry terbatas.
- `utils/notify.go` — Util notifikasi in-app (Fase 5.1): `NotifyUser`/`NotifyRole` — simpan `Notification` secara _async_ (pola `LogAudit`, retry terbatas) lalu broadcast event `NOTIFICATION_CREATED` secara terarah.
- `utils/websocket.go` — Hub manajemen koneksi WebSocket + broadcast event real-time. Sejak Fase 5.1 setiap koneksi menyimpan `WSClientInfo{UserID, Role}`; tersedia `BroadcastToUser` (per-user) & `BroadcastToRole` (per-role) selain `BroadcastEvent` global.
- `utils/pagination.go` — Helper utilitas untuk fitur paginasi (dengan cap maksimum `MAX_PAGE_SIZE`) dan proteksi _SQL injection_ pada kolom _sorting_.
- `utils/validator.go` — Validator kustom (mis. kebijakan password kuat).
- `utils/logger.go` — Sistem logging ganda menggunakan Zap (menyimpan log ke file JSON dan menampilkannya di konsol dengan warna).
- `uploads/` — Direktori penyimpanan foto bukti pemeliharaan (hasil normalisasi JPEG, tanpa EXIF). Akses via endpoint yang dilindungi JWT.
- `docs/` — Direktori yang berisi dokumentasi Swagger yang di-generate secara otomatis. Jalankan `swag init` setelah Anda memodifikasi anotasi pada _handlers_.

### Frontend `frontend/`
- `src/app/layout.tsx` — _Root layout_ yang mencakup `ThemeProvider` dan `AuthProvider`.
- `src/app/globals.css` — Konfigurasi Tailwind CSS v4 (melalui directive `@theme`), custom CSS variables (untuk _dark/light mode_), animasi UI saat scan, dan _style override_ khusus untuk `html5-qrcode`.
- `src/app/page.tsx` — Halaman akar yang hanya me-_redirect_ pengguna ke `/dashboard`.
- `src/app/dashboard/page.tsx` — Halaman Dashboard utama: manajemen aset CCTV (tabel data, filter pencarian, CRUD modal, tombol quick maintenance) + komponen `DashboardAnalytics`.
- `src/app/sebaran-cctv/page.tsx` — Halaman Peta Sebaran CCTV: visualisasi lokasi perangkat CCTV di kawasan IKN beserta ringkasan status (total/aktif/pemeliharaan), diperbarui real-time via WebSocket.
- `src/app/cctv/page.tsx` — Halaman Master Data CCTV. Berisi tabel detail lengkap termasuk badge skor kesehatan, _checkbox_ untuk seleksi massal (bulk select), modal penampil QR Code, fitur cetak QR, pemindai QR berbasis kamera, fitur _import_ via Excel, dan _download template_.
- `src/app/maintenance/page.tsx` — Halaman Log Pemeliharaan. Berisi tabel data historis, filter, CRUD modal, fitur unggah bukti foto pemeliharaan, penugasan ke teknisi (tombol "Tugaskan"), dan pengalihan tugas (tombol "Alihkan" untuk log `ongoing`, Fase 5.3 — menampilkan peringatan nama teknisi saat ini sebelum submit).
- `src/app/schedules/page.tsx` — Halaman Jadwal Pemeliharaan (Fase 5.4, khusus admin): CRUD jadwal berulang, toggle aktif/nonaktif, filter status, paginasi; menu sidebar "Jadwal".
- `src/app/users/page.tsx` — Halaman manajemen pengguna (khusus admin): CRUD user, reset password.
- `src/app/audit-logs/page.tsx` — Halaman audit trail (khusus admin): daftar aktivitas + ekspor CSV.
- `src/app/profile/page.tsx` — Halaman profil pengguna (data diri + tautan ganti password).
- `src/app/pemberitahuan/page.tsx` — Halaman Pemberitahuan (Fase 5.1): riwayat lengkap notifikasi milik user yang login (paginasi, filter belum dibaca, tandai semua dibaca, klik item → navigasi ke `/maintenance`/`/field-maintenance`). Scope per-user dijamin backend dari klaim JWT.
- `src/app/change-password/page.tsx` — Halaman self-service ganti password (wajib saat login pertama).
- `src/app/login/page.tsx` — Halaman antarmuka login dengan layout split-screen yang responsif.
- `src/proxy.ts` — _Middleware_ server-side Next.js: validasi sesi & role sebelum halaman admin diproses (guard tambahan di sisi server).
- `src/components/DashboardLayout.tsx` — Komponen _layout sidebar_ responsif dengan navigasi menu, _toggle dark mode_, info profil user, dan tombol _logout_.
- `src/components/NotificationBell.tsx` — Komponen lonceng notifikasi in-app (Fase 5.1): **hanya dirender di halaman Dashboard**; badge jumlah belum dibaca, dropdown 3 notifikasi teratas, tombol "Lihat lainnya" → `/pemberitahuan`, tandai semua dibaca, toast real-time via WebSocket `NOTIFICATION_CREATED`, fallback polling 60 detik.
- `src/components/ProtectedRoute.tsx` — Komponen _High-Order Guard_ yang melakukan otomatisasi _redirect_ ke `/login` jika pengguna belum terautentikasi.
- `src/components/DashboardAnalytics.tsx` — Komponen grafik/statistik dashboard.
- `src/components/CCTVMap.tsx` — Visualisasi lokasi CCTV pada peta, dirender di halaman `/sebaran-cctv`; warna marker dan badge popup mengikuti skor kesehatan.
- `src/components/AuthenticatedImage.tsx` — Komponen `<img>` yang memuat file gambar melalui endpoint yang dilindungi JWT.
- `src/components/Pagination.tsx` — Komponen navigasi halaman reusable.
- `src/context/AuthContext.tsx` — Manajemen state sesi dan pengambilan profil `/auth/me`.
  **Sesi disimpan dalam cookie httpOnly `sigap_session` (tidak dapat dibaca JavaScript).**
- `src/context/ThemeContext.tsx` — Manajemen state untuk tema UI (disimpan dalam localStorage dengan key `sigap_theme`).
- `src/context/WebSocketContext.tsx` — Koneksi WebSocket ke `/api/ws` untuk menerima broadcast event real-time.
- `src/lib/api.ts` — _Wrapper_ untuk fungsi `fetch`: mengirim cookie sesi (`credentials: 'include'`), menyematkan header `X-CSRF-Token` untuk permintaan mutasi, serta menangani _redirect_ ke `/login` ketika server mengembalikan status HTTP 401.

## 4. Model Database

Semua tabel database dikelola melalui GORM. Berikut adalah detail masing-masing struktur entitas:

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

### MaintenanceLog
Menyimpan data catatan pemeliharaan:
- `CCTVID` (Foreign Key → `cctvs`, serialized sebagai `cctv_id`)
- `TechnicianID` (Foreign Key → `users`, relasi `technician`)
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

### AuditLog
Menyimpan riwayat aktivitas penting pengguna (dipakai pada halaman audit trail):
- `UserID` (Foreign Key → `users`, nullable), `Username`, `UserRole`
- `Action`, `Module`, `Description`
- `IPAddress`, `UserAgent`

### Notification (Fase 5.1)
Notifikasi in-app per-pengguna (bell icon). Satu baris = satu penerima:
- `UserID` (Foreign Key → `users`, penerima)
- `Type` (konstanta `NotificationType*`: `MAINTENANCE_ASSIGNED`, `MAINTENANCE_REASSIGNED`, `MAINTENANCE_UNASSIGNED`, `MAINTENANCE_COMPLETED`, `MAINTENANCE_OVERDUE`, `FIELD_CHECK_ATTENTION`)
- `Title`, `Message` (teks untuk dropdown bell)
- `ReferenceID` (UUID record terkait) + `ReferenceType` (`"maintenance-log"` / `"field-maintenance"`) — penentu rute navigasi frontend
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
- Terdiri dari **4 services**:
  - `db` (berbasis `postgres:15-alpine`) — port `5432` **tidak dipublish** ke host; hanya dapat diakses dari dalam network.
  - `api` (merupakan _executable binary_ aplikasi Go) — port `3000` **tidak dipublish**; hanya diakses via reverse proxy.
  - `frontend` (menjalankan aplikasi Next.js) — port `3000` internal, **tidak dipublish**.
  - `nginx` (reverse proxy + TLS) — satu-satunya service yang mem-publish port `80` & `443`. Menangani terminasi TLS, HTTP→HTTPS redirect, security headers, routing `/api/*` (termasuk WebSocket) dan `/swagger/*` ke `api`, serta routing sisanya ke `frontend`.
- *Network*: Layanan berkomunikasi di bawah *bridge network* `sigap_network` (subnet eksplisit `172.28.0.0/16`; Nginx memakai IP statis `172.28.0.10` yang didaftarkan sebagai trusted proxy).
- *Volume*: Direktori `postgres_data` untuk persistensi _storage_ PostgreSQL; folder `./backend/uploads` di-*bind mount* untuk foto bukti.
- **Secret management**: Kredensial DB dibaca dari root `.env` (dipaksa via `${VAR:?}`), secret aplikasi (JWT/AES) dari `backend/.env.docker`. Jangan pernah men-commit file `.env*`.
- Layanan API membutuhkan file spesifik `backend/.env.docker` (gandakan dari `backend/.env.docker.example`); sertifikat TLS di `nginx/certs/`.
- Menggunakan *Health check* pada _service_ PostgreSQL: layanan API akan menunggu (`depends_on`) sampai _service_ basis data berstatus `healthy` sebelum memulai _boot sequence_-nya.
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
- Matriks cakupan test lengkap (35 file test: `utils`, `middleware`, `handlers`, `models`, `config`, `routes`) ada di **`docs/TESTING.md`**. Bacalah sebelum menambah/memodifikasi test (ada gotchas khusus sqlmock, `SkipDefaultTransaction`, dsb.).
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
_Dokumen ini adalah referensi sentral untuk AI. Patuhi segala aturan, instruksi arsitektural, dan gotchas teknis yang tercantum di atas._
