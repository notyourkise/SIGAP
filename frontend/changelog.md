# Changelog - Frontend SIGAP

## [0.1.0] - 2026-07-22

### Added
- Inisialisasi proyek Next.js menggunakan **TypeScript**, **App Router**, **ESLint**, dan **Tailwind CSS v4** di dalam direktori `src/`.
- Pemasangan pustaka **`lucide-react`** untuk kebutuhan ikon antarmuka.
- Pembuatan API client helper (`src/lib/api.ts`) untuk mempermudah pemanggilan REST API backend dengan dukungan otomatisasi otentikasi Bearer Token JWT.
- Pembuatan **`AuthContext`** (`src/context/AuthContext.tsx`) untuk memanajemen data sesi login pengguna (token JWT dan detail peran pengguna).
- Pembuatan **`ThemeContext`** (`src/context/ThemeContext.tsx`) untuk mendukung fitur **Light/Dark Toggle** yang dinamis berbasis penyimpanan `localStorage`.
- Pembuatan komponen proteksi rute **`ProtectedRoute`** (`src/components/ProtectedRoute.tsx`) guna mengamankan halaman dari akses ilegal sebelum login.
- Pembuatan layout dashboard **`DashboardLayout`** (`src/components/DashboardLayout.tsx`) yang menyediakan sidebar navigasi responsif, avatar pengguna, lencana peran, serta tombol keluar (logout).
- Pembuatan halaman **Login** (`src/app/login/page.tsx`) dengan tampilan premium berbasis *glassmorphism card* dan validasi input.
- Pembuatan halaman **Dashboard Aset CCTV** (`src/app/page.tsx`) yang menampilkan ringkasan statistik (Total, Aktif, Rusak, Pemeliharaan), tabel aset CCTV dengan pencarian & filter, serta modal form penambahan/edit aset (Admin saja).
- Pembuatan halaman **Riwayat Pemeliharaan** (`src/app/maintenance/page.tsx`) untuk mencatat log pemeliharaan preventif/korektif lengkap dengan filter CCTV, filter status, form log baru, dan fitur hapus log (Admin saja).

### Changed
- Konfigurasi file `package.json` agar server pengembangan Next.js berjalan di **port 3001** (`next dev -p 3001`) untuk menghindari tabrakan port dengan server backend Go Fiber di port 3000.
- Penyesuaian `globals.css` dengan variabel CSS khusus Tailwind v4 untuk mendukung transisi tema yang halus.

perubahan yang dilakukan:
Mengintegrasikan endpoint backend yang belum memiliki antarmuka dan menyelaraskan tampilan berdasarkan role admin/teknisi.
before:
Navigasi hanya menampilkan Aset CCTV dan Log Pemeliharaan; endpoint aset legacy, statistik, health check, upload bukti, ekspor, detail, serta manajemen pengguna belum memiliki alur UI lengkap. Aksi ubah log juga tersembunyi dari teknisi meskipun backend mengizinkannya.
after:
Menambahkan Dashboard API, Aset Legacy, dan Manajemen Pengguna; mengintegrasikan detail CCTV/log, ekspor CSV, QR aset legacy, upload foto bukti, serta seluruh operasi pengguna. Navigasi dan aksi kini mengikuti role: admin memperoleh fitur kelola penuh, sedangkan teknisi memperoleh akses operasional tanpa aksi khusus admin.
timestamp:
2026-07-28 12:15:58 WITA
kategori:
Feature, Integration, RBAC, UI/UX
file yang diubah:
src/app/dashboard/page.tsx, src/app/assets/page.tsx, src/app/users/page.tsx, src/app/cctv/page.tsx, src/app/maintenance/page.tsx, src/app/page.tsx, src/components/DashboardLayout.tsx, src/lib/api.ts, changelog.md
alasan perubahan:
Memastikan endpoint runtime SIGAP terlihat dan dapat digunakan dari sistem sesuai kewenangan dua role, tanpa menghapus endpoint aset lama yang masih tumpang tindih.

perubahan yang dilakukan:
Menghapus Aset Legacy dari antarmuka dan memisahkan dashboard berdasarkan role.
before:
Halaman dan navigasi Aset Legacy menampilkan data seeder/dummy. Admin dan teknisi menggunakan susunan dashboard yang sama, serta tombol Import Excel masih terlihat oleh teknisi.
after:
Aset Legacy tidak lagi tampil pada route maupun sidebar. Dashboard admin menampilkan ringkasan pengelolaan CCTV, pemeliharaan, dan pengguna; dashboard teknisi hanya menampilkan status CCTV, prioritas pekerjaan, scanner, serta akses pencatatan pemeliharaan. Import Excel dibatasi untuk admin.
timestamp:
2026-07-28 12:49:59 WITA
kategori:
Changed, RBAC, UI/UX
file yang diubah:
src/app/dashboard/page.tsx, src/app/assets/page.tsx, src/app/cctv/page.tsx, src/components/DashboardLayout.tsx, changelog.md
alasan perubahan:
Mencegah data dummy tampil sebagai data operasional dan memastikan setiap role hanya melihat fitur yang sesuai dengan hak akses backend.

perubahan yang dilakukan:
Menyederhanakan teks dashboard teknisi dan placeholder pencarian CCTV.
before:
Dashboard teknisi menampilkan deskripsi tambahan di bawah sapaan dan kolom pencarian menggunakan placeholder panjang yang menyebut seluruh atribut.
after:
Deskripsi tambahan dashboard teknisi dihapus dan placeholder pencarian disederhanakan menjadi "Cari CCTV".
timestamp:
2026-07-28 13:10:03 WITA
kategori:
UI/UX
file yang diubah:
src/app/dashboard/page.tsx, src/app/cctv/page.tsx, changelog.md
alasan perubahan:
Membuat antarmuka teknisi lebih ringkas dan sesuai arahan desain.
