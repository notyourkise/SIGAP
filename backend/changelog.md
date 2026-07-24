# Changelog - Backend SIGAP

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
