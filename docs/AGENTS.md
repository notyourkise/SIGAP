# Panduan Agent AI untuk Proyek SIGAP (Sistem Informasi & Inventaris CCTV Terpadu)

Selamat datang, Agent AI. Dokumen ini adalah panduan komprehensif mengenai codebase SIGAP. Bacalah dokumen ini secara saksama agar Anda dapat memahami arsitektur, konvensi, dan aturan proyek ini sebelum mulai berkontribusi.

## 1. Gambaran Umum Sistem
- **Nama**: SIGAP (Sistem Informasi & Inventaris CCTV Terpadu)
- **Tujuan**: Platform web untuk manajemen, inventarisasi, dan pemeliharaan perangkat CCTV secara terpusat.
- **Arsitektur**: Monorepo dengan 3 komponen utama:
  - **Backend**: REST API yang dibangun menggunakan Go Fiber, berlokasi di direktori `backend/`.
  - **Frontend**: Aplikasi web yang dibangun dengan Next.js 16 dan React 19, berlokasi di direktori `frontend/`.
  - **Database**: PostgreSQL 15, dijalankan melalui Docker.
- **Deployment**: Menggunakan Docker Compose untuk orkestrasi container.

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
- `docker-compose.yml` — Orkestrasi 3 _service_ utama (db, api, frontend) di dalam _network_ `sigap_network`.
- `makefile` — Kumpulan _script_ untuk menjalankan backend dan frontend secara bersamaan di lingkungan lokal.
- `.gitignore` — Mengabaikan direktori seperti `node_modules`, file `.env`, hasil _build_ binary Go, dan folder _logs_.

### Backend `backend/`
- `main.go` — Titik masuk (entry point) aplikasi Go. Berisi inisialisasi _logger_, koneksi database, _migrations_, _seeders_, instansiasi Fiber app, _middleware stack_, pendaftaran _routes_, dan _graceful shutdown_.
- `config/database.go` — Konfigurasi koneksi ke PostgreSQL menggunakan GORM dengan pengaturan _connection pooling_.
- `models/setup.go` — Definisi struktur `Base` (berisi UUID, timestamps, dan _soft delete_) yang di-_embed_ pada semua model. Termasuk _hook_ `BeforeCreate` yang hanya men-generate UUID baru jika ID kosong (`uuid.Nil`).
- `models/cctv.go` — Definisi struktur `CCTV` dengan 17 field, termasuk `QrPayload` yang dienkripsi menggunakan AES-256.
- `models/seeder.go` — _Script_ untuk me-seed admin _default_ dan 4 data aset CCTV _dummy_.
- `handlers/` — Berisi _handler_ untuk setiap modul: `auth`, `asset`, `cctv`, `maintenance`, `user`, `dashboard`, `upload`, dan `health`.
- `middleware/auth.go` — Fungsi `Protected()` untuk memvalidasi JWT token dan `RequireRole()` untuk pengecekan _Role-Based Access Control_ (RBAC).
- `middleware/limiter.go` — Pembatasan laju _request_ (Rate Limiter). Terdapat `GlobalLimiter` (100 req/menit) dan `AuthLimiter` (khusus _endpoint_ autentikasi, 5 req/menit).
- `routes/` — Registrasi rute untuk masing-masing modul. 
  > [!IMPORTANT]
  > **PENTING:** Rute statis (seperti `/template`, `/import`, `/bulk-delete`) HARUS dideklarasikan SEBELUM rute dinamis (seperti `/:id`) untuk menghindari _collision_ atau tertimpa oleh parameter Fiber.
- `utils/encryption.go` — Fungsi `EncryptAES256` dan `DecryptAES256` berbasis AES-256-GCM. Kunci diderivasi menggunakan _hash_ SHA-256 dari _environment variable_ `AES_SECRET_KEY`.
- `utils/jwt.go` — Fungsi untuk me-generate dan memvalidasi JWT token dengan durasi kedaluwarsa 24 jam menggunakan algoritma HMAC-SHA256.
- `utils/pagination.go` — Helper utilitas untuk fitur paginasi yang dilengkapi dengan proteksi _SQL injection_ pada kolom _sorting_.
- `utils/logger.go` — Sistem logging ganda menggunakan Zap (menyimpan log ke file JSON dan menampilkannya di konsol dengan warna).
- `docs/` — Direktori yang berisi dokumentasi Swagger yang di-generate secara otomatis. Jalankan `swag init` setelah Anda memodifikasi anotasi pada _handlers_.

### Frontend `frontend/`
- `src/app/layout.tsx` — _Root layout_ yang mencakup `ThemeProvider` dan `AuthProvider`.
- `src/app/globals.css` — Konfigurasi Tailwind CSS v4, custom CSS variables (untuk _dark/light mode_), animasi UI saat scan, dan _style override_ khusus untuk `html5-qrcode`.
- `src/app/page.tsx` — Halaman _Dashboard_ utama yang berisi manajemen Aset CCTV (tabel data, filter pencarian, CRUD modal, dan tombol quick maintenance).
- `src/app/cctv/page.tsx` — Halaman Master Data CCTV. Berisi tabel detail lengkap, _checkbox_ untuk seleksi massal (bulk select), modal penampil QR Code, fitur cetak QR, pemindai QR berbasis kamera, fitur _import_ via Excel, dan _download template_.
- `src/app/login/page.tsx` — Halaman antarmuka login yang menggunakan desain _glassmorphism_.
- `src/app/maintenance/page.tsx` — Halaman Log Pemeliharaan. Berisi tabel data historis, filter, CRUD modal, serta fitur unggah bukti foto pemeliharaan.
- `src/components/DashboardLayout.tsx` — Komponen _layout sidebar_ responsif dengan navigasi menu, _toggle dark mode_, info profil user, dan tombol _logout_.
- `src/components/ProtectedRoute.tsx` — Komponen _High-Order Guard_ yang melakukan otomatisasi _redirect_ ke `/login` jika pengguna belum terautentikasi.
- `src/context/AuthContext.tsx` — Manajemen state untuk JWT token (disimpan dalam localStorage dengan key `sigap_token`).
- `src/context/ThemeContext.tsx` — Manajemen state untuk tema UI (disimpan dalam localStorage dengan key `sigap_theme`).
- `src/lib/api.ts` — _Wrapper_ untuk fungsi `fetch`: secara otomatis menyematkan token Bearer pada _header_, serta melakukan penanganan _redirect_ ke `/login` ketika server mengembalikan status HTTP 401.

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

### Asset
Menyimpan master data infrastruktur aset umum:
- `Name`
- `SerialNumber` (unique)
- `IPAddress`
- `LocationName`
- `Latitude`, `Longitude`
- `Status` (enum: `aktif` / `rusak` / `pemeliharaan` / `nonaktif`)
- `Vendor`
- `InstallationDate`

### CCTV
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

### MaintenanceLog
Menyimpan data catatan pemeliharaan:
- `AssetID` (Foreign Key yang merujuk ke tabel Asset)
- `TechnicianID` (Foreign Key yang merujuk ke tabel User)
- `ActionTaken`
- `Notes`
- `Status` (enum: `scheduled` / `ongoing` / `completed` / `failed`)
- `ScheduledDate`
- `CompletedAt`
- `ImageUrl` (jalur URL gambar bukti pemeliharaan)

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
- **Autentikasi**: Endpoint dilindungi menggunakan Header `Authorization: Bearer <token>`.
- **Otorisasi (Role Check)**: Gunakan _middleware_ `RequireRole("admin")` pada rute yang melibatkan operasi _write_ atau _delete_ (hanya admin yang dapat memodifikasi data inti).
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
- PostgreSQL secara _default_ tetap memberlakukan _unique constraint_ (seperti pada kolom `KodeCCTV` atau `SerialNumber`) meskipun sebuah _row_ telah berstatus _soft-delete_.
- Pada _handler_ `CreateCCTV` dan `ImportCCTV`, digunakan perintah `.Unscoped()` untuk secara eksplisit mencari data yang pernah di-_soft-delete_. Jika ditemukan bentrokan (data yang telah terhapus tapi konfigurasinya sama), sistem akan melakukan _hard delete_ (_force delete_) terlebih dahulu sebelum melakukan insert data baru.

### AES_SECRET_KEY Consistency
- Kunci (_key_) enkripsi **HARUS konsisten** antara saat sistem me-generate QR code dan saat memindai (_scan_) QR code tersebut.
- Jika nilai ini berubah (misalnya pada saat Anda berpindah menggunakan `.env` ke `.env.docker`), QR Code yang telah dicetak lama **tidak akan bisa didekripsi** lagi.
- Pastikan bahwa `AES_SECRET_KEY` memiliki nilai yang identik baik di `.env` lokal Anda maupun `.env.docker`.

### Next.js SSR & Browser APIs
- Komponen atau _library_ yang bergantung pada objek global di browser seperti `window` atau `navigator` (contohnya `html5-qrcode`) **HARUS** di-import secara asinkron atau dimuat secara dinamis melalui fungsi `import()` di dalam blok `useEffect`.
- **Dilarang** melakukan impor _library_ ini secara _top-level scope_, karena _Server-Side Rendering_ (SSR) bawaan Next.js akan memproses kodenya di Node.js (dimana API DOM tidak tersedia) yang mengakibatkan aplikasi _crash_.

### Kamera Scanner & Secure Context
- Browser modern memiliki batasan privasi yang memblokir akses ke fungsi WebRTC `getUserMedia()` (akses kamera). Fitur ini hanya bisa dipanggil dalam ekosistem _Secure Context_: yakni melalui protokol `https://` atau jika di-host langsung di `http://localhost`.
- **Catatan**: Fitur ini _tidak akan berfungsi_ jika aplikasi diakses melalui IP Address lokal (contoh: `http://192.168.x.x:3001`) karena hal itu dianggap sebagai _insecure context_.

## 7. Konfigurasi Docker
Proyek menggunakan pendekatan *Dockerized Environment* untuk *development* dan *deployment*:
- Terdiri dari 3 *services* (layanan) utama: 
  - `db` (berbasis `postgres:15-alpine`)
  - `api` (merupakan _executable binary_ aplikasi Go)
  - `frontend` (menjalankan aplikasi Next.js)
- *Network*: Layanan ini saling berkomunikasi di bawah _bridge network_ lokal bernama `sigap_network`.
- *Volume*: Direktori `postgres_data` untuk persistensi _storage_ PostgreSQL agar data tidak hilang ketika _container_ dimatikan.
- Layanan API membutuhkan file spesifik `backend/.env.docker` (silakan gandakan dari template `backend/.env.docker.example`).
- _Environment variable_ khusus frontend `NEXT_PUBLIC_API_URL` dikonfigurasi dan di-set langsung (di-_inject_) di dalam file `docker-compose.yml`.
- Menggunakan *Health check* pada _service_ PostgreSQL: layanan API akan menunggu (`depends_on`) sampai _service_ basis data berstatus `healthy` sebelum memulai _boot sequence_-nya.

## 8. Swagger Documentation (API Docs)
Backend mendukung auto-generasi UI Dokumentasi Interaktif.
- Antarmuka Swagger UI dapat diakses pada `http://localhost:3000/swagger/index.html`.
- Setiap kali Anda memodifikasi atau menambah _endpoint_ REST API yang menyertakan komentar _Swagger annotations_ di dalam file _handlers_, pastikan untuk menjalankan perintah regenerasi:
  ```bash
  cd backend
  swag init
  ```
- File yang secara otomatis di-generate akan tersimpan pada: `docs/docs.go`, `docs/swagger.json`, `docs/swagger.yaml`.

## 9. Testing
- Modul Backend telah dilengkapi dengan Unit Tests dasar yang berlokasi pada:
  - `handlers/auth_test.go`
  - `utils/jwt_test.go`
- Untuk mengeksekusi tes, gunakan perintah: 
  ```bash
  cd backend && go test ./...
  ```

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
