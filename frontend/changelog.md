# Changelog - Frontend SIGAP

## [0.1.0] - 2026-07-22

### Added New
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

perubahan yang dilakukan:
Menyempurnakan dashboard per role, identitas sidebar, loading state, dan halaman login.
before:
Dashboard masih menampilkan shortcut pengelolaan, rincian status pemeliharaan belum lengkap, status API hanya berupa label singkat, logo menghilang saat sidebar diciutkan, dan sebagian aksi belum memiliki indikator proses. Halaman login masih memuat terlalu banyak teks dan efek visual.
after:
Dashboard admin menampilkan rincian pemeliharaan terjadwal, berjalan, selesai, dan gagal tanpa bagian Akses Pengelolaan Admin. Dashboard teknisi menampilkan prioritas CCTV dan status pekerjaan tanpa bagian Akses Operasional Teknisi. Kesehatan API dilengkapi keterangan dan waktu pemeriksaan, logo aplikasi tetap terlihat saat sidebar kecil, aksi utama memiliki loading state, dan login dibuat lebih ringkas dengan palet hijau-kuning yang konsisten.
timestamp:
2026-07-29 10:45:02 WITA
kategori:
UI/UX, RBAC, Loading State
file yang diubah:
src/app/dashboard/page.tsx, src/app/login/page.tsx, src/app/cctv/page.tsx, src/app/maintenance/page.tsx, src/components/DashboardLayout.tsx, changelog.md
alasan perubahan:
Meningkatkan keterbacaan informasi operasional, memberikan umpan balik yang jelas saat proses berjalan, dan menjaga konsistensi identitas visual pada seluruh role.

perubahan yang dilakukan:
Mengurangi ikon dekoratif pada login, sidebar, dan dashboard serta mempertahankan akses keluar saat sidebar kecil.
before:
Login memiliki ikon kunci dekoratif, identitas SIGAP di sidebar didampingi ikon database, tombol keluar menghilang ketika sidebar diciutkan, dan dashboard menggunakan ikon pada hampir setiap kartu.
after:
Ikon dekoratif login dan identitas sidebar dihapus. Tulisan SIGAP tetap tampil ringkas saat sidebar kecil, tombol keluar tetap terlihat sebagai tombol ikon, dan dashboard menggunakan tipografi, garis warna, serta titik status sebagai penanda utama.
timestamp:
2026-07-29 10:56:08 WITA
kategori:
UI/UX
file yang diubah:
src/app/login/page.tsx, src/app/dashboard/page.tsx, src/components/DashboardLayout.tsx, changelog.md
alasan perubahan:
Membuat antarmuka lebih tenang, klasik, dan tidak terlihat dipenuhi ikon dekoratif.

perubahan yang dilakukan:
Menyeimbangkan kartu ringkasan, memperjelas prioritas teknisi, dan memperbaiki sumber total pemeliharaan.
before:
Kartu dashboard dan Aset CCTV memakai susunan serta definisi angka yang berbeda. Nilai Pemeliharaan dihitung dari status perangkat sehingga tetap nol walaupun sudah ada catatan pemeliharaan. Prioritas teknisi mengulang informasi CCTV dan kartu kesehatan API belum cukup terstruktur.
after:
Kartu ringkasan pada dashboard dan Aset CCTV memakai urutan yang konsisten: Total CCTV, Aktif, Nonaktif, dan Pemeliharaan. Pemeliharaan mengambil total catatan dari endpoint log pemeliharaan. Prioritas teknisi hanya menampilkan status pekerjaan Terjadwal, Dalam Proses, Selesai, dan Gagal. Kesehatan API menampilkan status online, ketersediaan backend, sinkronisasi data, serta waktu pemeriksaan.
timestamp:
2026-07-29 11:06:30 WITA
kategori:
UI/UX, Data Integration
file yang diubah:
src/app/dashboard/page.tsx, src/app/cctv/page.tsx, changelog.md
alasan perubahan:
Memastikan angka mudah dibandingkan, tidak membingungkan pengguna, dan berasal dari sumber data yang sesuai.

perubahan yang dilakukan:
Mengatasi rate limit dashboard teknisi dan menambahkan detail interaktif aktivitas pemeliharaan.
before:
Dashboard mengirim delapan request bersamaan sehingga backend dapat membalas 429 dan seluruh data ikut kosong. Kartu status pemeliharaan tidak dapat dibuka, serta kartu ringkasan masih memiliki teks keterangan kecil.
after:
Dashboard hanya mengirim tiga request dan memproses hasil CCTV, pemeliharaan, serta health check secara terpisah. Kegagalan health check tidak lagi mengosongkan data CCTV. Kartu Terjadwal, Dalam Proses, Selesai, dan Gagal dapat ditekan untuk menampilkan nama CCTV, lokasi, serta tindakan pemeliharaan. Teks keterangan kecil pada kartu ringkasan dihapus.
timestamp:
2026-07-29 11:14:19 WITA
kategori:
Bug Fix, UI/UX, Data Integration
file yang diubah:
src/app/dashboard/page.tsx, src/app/cctv/page.tsx, changelog.md
alasan perubahan:
Menghindari pembatasan request backend, menjaga data dashboard tetap tampil secara parsial, dan memudahkan pengguna melihat detail pekerjaan langsung dari ringkasan status.
