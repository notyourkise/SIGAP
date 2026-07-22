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
