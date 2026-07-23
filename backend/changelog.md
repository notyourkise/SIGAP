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
