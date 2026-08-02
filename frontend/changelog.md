# Changelog - Frontend SIGAP

## [1.4.0] - 2026-08-02

### Hapus Cookie CSRF Saat Logout

perubahan yang dilakukan: **Memanggil clearAuthCookies setelah proses logout**
before:
- Logout hanya mengandalkan backend menghapus cookie via Set-Cookie; cookie CSRF (non-httpOnly) dapat tertinggal di klien.
after:
- Setelah permintaan logout, frontend juga memanggil `clearAuthCookies` untuk membersihkan cookie CSRF di sisi klien (cookie sesi httpOnly tetap hanya dapat dihapus server).
timestamp: 2026-08-02 WITA
kategori: Bugfix
file yang diubah:
- frontend/src/context/AuthContext.tsx
- frontend/changelog.md
alasan teknis: Memastikan state autentikasi benar-benar bersih setelah logout dan mencegah cookie CSRF sisa mengganggu login berikutnya.

## [1.3.0] - 2026-08-02

### Hapus Auto-Logout Idle

perubahan yang dilakukan: **Menghapus timer auto-logout 15 menit dan modal peringatan inaktivitas**
before:
- Pengguna otomatis keluar setelah 15 menit tanpa aktivitas (idle timer) dan melihat modal peringatan "Sesi Hampir Berakhir".
- Sesi dapat terputus mendadak meski token masih valid.
after:
- Auto-logout idle dihapus total: pengguna tetap login sampai menekan logout atau token kedaluwarsa (5 jam).
- `AuthContext` tidak lagi memakai idle timer, listener aktivitas, `extendSession`, maupun modal peringatan inaktivitas.
- Logout manual tetap merevoke token & mengarahkan ke halaman login; sesi mati saat browser ditutup (session cookie).
timestamp: 2026-08-02 WITA
kategori: UI/UX, Refactor
file yang diubah:
- frontend/src/context/AuthContext.tsx
- frontend/changelog.md
alasan teknis: Menghilangkan logout mendadak yang mengganggu; model sesi disederhanakan sesuai permintaan pengguna (login → token, logout → langsung mati).

## [1.2.0] - 2026-08-02

### Redesign Halaman Login Split-Screen

perubahan yang dilakukan: **Mengganti layout login glassmorphism menjadi split-screen 50:50 dengan foto malam IKN dan panel krem**
before:
- Halaman login memakai kartu sentral glassmorphism di atas latar abstrak (blur elemen dekoratif) tanpa foto kawasan.
- Branding hanya berupa teks "SIGAP" tanpa logo Pohon Hayat NUSANTARA.
after:
- Layout responsif split 50:50: kiri adalah foto full-bleed malam Kompleks Kemenko I (Garuda Palace) IKN dari `public/assets/foto/ikn-kemenko.webp` (mobile: banner atas, desktop: setengah layar penuh); kanan adalah panel krem lembut.
- Panel kanan menampilkan logo Pohon Hayat NUSANTARA (`public/nusantara-logo.png`) dengan teks "NUSANTARA Ibu Kota Negara Indonesia", judul "SIGAP" font serif hijau tua, dan deskripsi "Sistem internal pengelolaan CCTV kawasan IKN."
- Form login dipertahankan lengkap: label & input "Username atau email" (ikon user) dan "Kata sandi" (ikon gembok + toggle mata), tombol "Masuk" hijau tua dengan teks putih.
- Ikon status (lonceng + toggle tema matahari/bulan) di pojok kanan atas panel krem.
- Seluruh warna memakai token tema (`bg-background`, `text-foreground`, `bg-primary`, `border-input-border`, dsb.) sehingga toggle mode gelap/terang berfungsi: mode terang = krem + hijau tua, mode gelap = hijau tua + krem.
timestamp: 2026-08-02 WITA
kategori: UI/UX, Feature
file yang diubah:
- frontend/src/app/login/page.tsx
- frontend/changelog.md
alasan teknis: Meningkatkan identitas visual kawasan IKN pada halaman masuk serta menghadirkan tipografi & palet hijau tua-krem yang modern dan tegas.

## [1.1.0] - 2026-08-02

### Halaman 404 Kustom dengan Animasi Lottie

perubahan yang dilakukan: **Menambahkan halaman `not-found` kustom yang dirender untuk seluruh URL yang tidak cocok**
before:
- URL yang tidak dikenal menampilkan halaman 404 bawaan Next.js tanpa identitas SIGAP.
after:
- Halaman `/not-found` baru menampilkan animasi Lottie T-rex "PAGE NOT FOUND" berukuran besar (hingga 560px) dari `public/assets/animasi/404 Error.json` di atas latar putih polos.
- Animasi dimuat saat runtime via fetch (aset statis publik) melalui komponen client `Lottie404`; ada placeholder netral saat memuat agar layout tidak melompat.
- Desain minimalis: background putih murni tanpa elemen warna, heading, tombol, maupun footer — hanya animasi yang tampil.
timestamp: 2026-08-02 WITA
kategori: UI/UX, Feature
file yang diubah:
- frontend/src/app/not-found.tsx (baru)
- frontend/src/components/Lottie404.tsx (baru)
- frontend/package.json (tambah dependency `lottie-react`)
- frontend/changelog.md
alasan teknis: Memberikan pengalaman 404 yang konsisten dengan identitas produk SIGAP dan memanfaatkan aset animasi yang telah tersedia.

## [1.0.0] - 2026-08-02

### Server-side Route Guard dan Audit Log

perubahan yang dilakukan: **Middleware route guard dan pencegahan request audit yang tidak perlu**
before:
- Akses `/audit-logs` dan `/users` hanya mengandalkan pemeriksaan role di client.
- Halaman audit dapat memulai request sebelum status user admin diketahui.
after:
- Proxy Next.js memeriksa cookie sesi dan role admin melalui `/api/auth/me` sebelum route admin diproses.
- Halaman audit hanya mengambil data setelah user terautentikasi sebagai admin.
timestamp: 2026-08-02 00:00 WITA
kategori: Security, Quality
file yang diubah:
- frontend/src/proxy.ts
- frontend/src/app/audit-logs/page.tsx
- frontend/eslint.config.mjs
alasan teknis: Mengurangi ketergantungan pada guard client-side dan menghindari request API yang tidak perlu dari role teknisi.

## [0.9.0] - 2026-08-01

### Fitur Kapabilitas Fase 3 (ROADMAP 3.1 - 3.9)

perubahan yang dilakukan: **Dashboard analitik, notifikasi tugas teknisi, halaman profil, filter rentang tanggal, ekspor CSV/PDF**
before:
- Belum ada dashboard analitik; halaman utama setelah login belum menampilkan statistik/tren/peringatan.
- Teknisi tidak mendapat ringkasan tugas; belum ada halaman profil.
- List & ekspor maintenance/audit-log tanpa filter rentang tanggal.
- Belum ada tombol unduh laporan PDF maupun ekspor audit-log.
after:
- **Dashboard analitik (3.3):** halaman `/dashboard` menampilkan statistik (total CCTV, log, selesai, terlambat), tren pemeliharaan, distribusi status, dan daftar CCTV overdue via `DashboardStats`/`DashboardAnalytics`.
- **Notifikasi tugas (3.4):** section "Tugas Anda" di dashboard dengan badge tugas aktif/terlambat, memakai `GET /dashboard/tasks`.
- **Halaman profil (3.5):** `/profile` untuk mengubah nama tampilan (`PUT /auth/profile`); kartu profil di sidebar menjadi link.
- **Unifikasi CCTV (3.6):** seluruh tampilan maintenance memakai `cctv_id`/`cctv` (nama, kode, lokasi) menggantikan `asset`; filter CCTV pada form & list.
- **Filter rentang tanggal & ekspor (3.7):** input `start_date`/`end_date` pada halaman maintenance & audit-logs; tombol "Laporan PDF" dan "Ekspor CSV" (via `downloadApiFile`).
- **Upload foto (3.8):** server kini menormalkan semua upload ke JPEG dan menghapus metadata GPS; UX tampilan lampiran tetap memakai `AuthenticatedImage`.
- **Ekspor per role (3.9):** tombol ekspor/laporan pada maintenance mengikuti scope data backend sesuai role (teknisi = data miliknya).
timestamp: 2026-08-01 14:30 WITA
kategori: Feature, Dashboard, UI/UX, PDF, Export
file yang diubah:
- frontend/src/app/dashboard/page.tsx (3.3, 3.4)
- frontend/src/components/DashboardAnalytics.tsx (3.3)
- frontend/src/components/DashboardStats.tsx (3.3)
- frontend/src/app/profile/page.tsx (baru, 3.5)
- frontend/src/context/AuthContext.tsx (3.5)
- frontend/src/components/DashboardLayout.tsx (3.5)
- frontend/src/app/maintenance/page.tsx (3.6, 3.7, 3.8)
- frontend/src/app/audit-logs/page.tsx (3.7)
- frontend/changelog.md
alasan perubahan: Menyelesaikan ROADMAP Fase 3 (fitur kapabilitas) dan menyesuaikan tampilan dengan unifikasi model CCTV.

---

## [0.8.0] - 2026-08-01

### Hardening Keamanan Fase 2 (Security Audit & Roadmap)

perubahan yang dilakukan: **Migrasi Auth ke Cookie httpOnly + CSRF, Pengetatan Kebijakan Password**
before:
- Token JWT disimpan di `localStorage` (`sigap_token`/`sigap_user`) dan dikirim via header `Authorization` sehingga rentan terhadap XSS token theft.
- Mutasi terautentikasi (POST/PUT/DELETE) tidak mengirimkan token CSRF, dan beberapa fetch mentah (QR, import, download template) memakai `localStorage` token secara langsung.
- Form kata sandi (buat/reset user) hanya membatasi `minLength=6`.
after:
- **Auth berbasis cookie httpOnly:** seluruh request API memakai `credentials:'include'`; token JWT otomatis dikirim via cookie `sigap_session` oleh browser (tidak lagi di `localStorage`).
- **CSRF double-submit:** `apiRequest` otomatis membaca cookie `sigap_csrf` dan mengirim header `X-CSRF-Token` untuk semua mutasi (`POST/PUT/PATCH/DELETE`).
- **Sesi terkelola via backend:** bootstrap sesi lewat `GET /auth/me`; event `sigap:auth-expired` di-dispatch saat 401 untuk reset state & redirect `/login`; helper `getCookie`, `clearAuthCookies`, `downloadApiFile` (dengan credentials) ditambahkan.
- `AuthContextType` tidak lagi mengekspos `token`; `WebSocketContext` tidak lagi mengirim `?token=` (browser mengirim cookie), `AuthenticatedImage` & seluruh fetch mentah di `cctv/page.tsx` memakai `credentials:'include'`.
- **Kebijakan password:** form buat/reset pengguna kini `minLength=8` (backend menerapkan kompleksitas minimal).
timestamp: 2026-08-01 13:45 WITA
kategori: Security, Auth, Hardening, API
file yang diubah:
- frontend/src/lib/api.ts
- frontend/src/context/AuthContext.tsx
- frontend/src/context/WebSocketContext.tsx
- frontend/src/app/login/page.tsx
- frontend/src/app/users/page.tsx
- frontend/src/app/cctv/page.tsx
- frontend/src/components/AuthenticatedImage.tsx
- frontend/changelog.md
alasan perubahan: Menutup temuan Fase 2 (docs/SECURITY-AUDIT.md) terkait penyimpanan token di localStorage, ketiadaan CSRF protection, dan kebijakan password lemah.

---

## [0.7.0] - 2026-08-01

### Hardening Keamanan Fase 1 (Security Audit & Roadmap)

perubahan yang dilakukan: **Perbaikan Stored XSS pada Cetak Label QR & Migrasi QR ke Server-side**
before:
- Cetak label QR (tunggal & batch) memakai `window.open().document.write()` dengan interpolasi mentah `nama`, `kode_cctv`, dan `lokasi_area` dari database (risiko Stored XSS — eksekusi skrip berbahaya saat mencetak).
- Gambar QR dibuat via API pihak ketiga `api.qrserver.com` dengan mengirim `QrPayload` terenkripsi ke server eksternal (kebocoran data & dependensi layanan luar yang tidak terautentikasi).
- Koneksi WebSocket tidak mengirimkan token sehingga backend tidak dapat mengautentikasi client.
after:
- Cetak label QR dirender ulang sebagai elemen React (bukan `document.write`); seluruh teks (nama, kode, lokasi) di-*escape* otomatis oleh React sehingga Stored XSS tertutup.
- Area cetak dirender via portal ke `document.body` (`#sigap-print-area`) dan hanya ditampilkan saat mencetak (`@media print`), dengan ukuran halaman `100mm × 38mm` (label tunggal) atau A4 (batch).
- Gambar QR ditampilkan dari endpoint server `GET /api/cctv/:id/qrcode` (dengan header `Authorization`), di-cache per perangkat, menggantikan `api.qrserver.com`.
- Koneksi WebSocket kini mengirim `?token=` dari `AuthContext` saat handshake.
timestamp: 2026-08-01 15:30 WITA
kategori: Security, Bugfix, UI/UX, API
file yang diubah:
- frontend/src/app/cctv/page.tsx
- frontend/src/app/globals.css
- frontend/src/context/WebSocketContext.tsx
- frontend/changelog.md
alasan perubahan: Menutup temuan kritis Stored XSS (docs/SECURITY-AUDIT.md) dan menghilangkan ketergantungan pada layanan QR eksternal, sekaligus mengamankan handshake WebSocket.

---

## [0.6.0] - 2026-08-01

### Dashboard Peta Interaktif GIS (Leaflet.js)

perubahan yang dilakukan: **Integrasi Komponen CCTVMap.tsx Berbasis Leaflet.js & OpenStreetMap IKN**
before:
- Halaman dashboard tidak memiliki visualisasi peta geografis sebaran perangkat CCTV di kawasan IKN.
after:
- Mengintegrasikan pustaka `leaflet` & `react-leaflet` dengan *dynamic import SSR-safe* (`next/dynamic` dengan `{ ssr: false }`).
- Dibuat komponen `CCTVMap.tsx` di `src/components/CCTVMap.tsx` yang memvisualisasikan seluruh perangkat CCTV terpusat di kawasan **IKN Nusantara** (`-0.9723, 116.7083`).
- Menyiapkan SVG DivIcon kustom ber-kode warna status: 🟢 **Aktif**, 🔴 **Nonaktif/Rusak**, 🟡 **Pemeliharaan**.
- Menambahkan **Interactive Popup Card** dan **Tooltip** yang menyajikan Nama CCTV, Kode, Lokasi, IP Kamera, Status, dan tombol pintas ke Master CCTV.
- Diintegrasikan ke halaman `src/app/dashboard/page.tsx`.
timestamp: 2026-08-01 01:44 WITA
kategori: Feature, GIS, UI/UX, Map
file yang diubah:
- frontend/package.json
- frontend/src/components/CCTVMap.tsx (baru)
- frontend/src/app/dashboard/page.tsx
- frontend/changelog.md
alasan perubahan: Menyajikan pemantauan visual spasial berbasis GIS untuk membantu pengelola sistem melihat sebaran posisi dan kondisi operasional perangkat CCTV di kawasan IKN secara cepat.

---

## [0.5.0] - 2026-08-01

### Audit Log Viewer UI (Admin Trail Exploration)

perubahan yang dilakukan: **Pembuatan Halaman Master Audit Log Trail & Navigasi Sidebar**
before:
- Tidak ada halaman antarmuka frontend untuk melihat rekam jejak audit trail aktivitas sistem yang dikumpulkan oleh backend.
after:
- Dibuat halaman master `src/app/audit-logs/page.tsx` khusus untuk peran Admin dengan proteksi RBAC.
- Diintegrasikan ke endpoint `GET /api/audit-logs` dengan dukungan paginasi, pencarian kata kunci (*search*), dan penyaringan berdasarkan modul (`AUTH`, `CCTV`, `ASSET`, `MAINTENANCE`, `USER`).
- Menambahkan **Modal Detail Audit Log** untuk menampilkan detail teknis lengkap (User ID, User Agent browser, Client IP proxy-aware, dan Timestamp presisi).
- Menambahkan menu navigasi **"Audit Log"** dengan ikon `ShieldCheck` pada sidebar `DashboardLayout.tsx` untuk Admin.
- Mengintegrasikan listener real-time WebSocket (`AUDIT_LOG_ADDED`) untuk otomatis memuat ulang log saat ada aktivitas baru.
timestamp: 2026-08-01 01:36 WITA
kategori: Feature, Admin, Security, UI/UX
file yang diubah:
- frontend/src/app/audit-logs/page.tsx (baru)
- frontend/src/components/DashboardLayout.tsx
- frontend/changelog.md
alasan perubahan: Memberikan kapabilitas auditabilitas penuh kepada Admin untuk memantau keamanan dan riwayat operasional infrastruktur IKN.

---

## [0.4.0] - 2026-08-01

### Real-time WebSocket Client Integration & Live Auto-Refetch

perubahan yang dilakukan: **Integrasi WebSocketProvider & Subskripsi Event Real-time pada Halaman Utama**
before:
- Antarmuka tidak secara otomatis memperbarui tabel atau statistik saat ada perubahan data di server oleh pengguna lain.
after:
- Dibuat `WebSocketContext.tsx` di `src/context/WebSocketContext.tsx` dengan fitur otomatisasi *auto-reconnect* (exponential backoff).
- Dibungkus `WebSocketProvider` di `src/app/layout.tsx`.
- Diintegrasikan *hook subscription* `useWebSocket` pada halaman utama:
  - `src/app/dashboard/page.tsx`: Mendengarkan `CCTV_UPDATED`, `MAINTENANCE_CREATED`, dan `MAINTENANCE_UPDATED` untuk memuat ulang statistik & daftar CCTV/log secara otomatis.
  - `src/app/cctv/page.tsx`: Mendengarkan `CCTV_UPDATED` untuk memuat ulang tabel master CCTV secara otomatis.
  - `src/app/maintenance/page.tsx`: Mendengarkan `MAINTENANCE_CREATED` dan `MAINTENANCE_UPDATED` untuk memuat ulang tabel riwayat pemeliharaan secara otomatis.
timestamp: 2026-08-01 01:35 WITA
kategori: Feature, Real-time, UI/UX
file yang diubah:
- frontend/src/context/WebSocketContext.tsx (baru)
- frontend/src/app/layout.tsx
- frontend/src/app/dashboard/page.tsx
- frontend/src/app/cctv/page.tsx
- frontend/src/app/maintenance/page.tsx
- frontend/changelog.md
alasan perubahan: Menyajikan pengalaman pengguna yang sangat responsif di mana setiap perubahan data CCTV atau log pemeliharaan langsung memicu pembaruan antarmuka secara instan tanpa perlu refresh.

---

## [0.3.0] - 2026-08-01

### Inactivity Auto-Logout & Idle Timer Security

perubahan yang dilakukan: **Integrasi Pelacak Keaktifan Pengguna, Warning Modal (14 Menit), dan Auto-Logout (15 Menit)**
before:
- Sesi pengguna di frontend tetap aktif di localStorage tanpa batas waktu pasif/inaktivitas selama browser terbuka.
- Tidak ada mekanisme otomatis untuk mencabut token JWT jika pengguna meninggalkan komputer dalam kondisi ter-login.
after:
- Diimplementasikan pelacak keaktifan pengguna global (`mousemove`, `keydown`, `click`, `scroll`, `touchstart`) pada `AuthContext.tsx`.
- Menambahkan modal peringatan `IdleWarningModal` yang muncul di menit ke-14 dengan countdown detik dan opsi **"Tetap Masuk"** untuk memperpanjang sesi.
- Otomatisasi pemanggilan `logout('inactivity')` yang mengirimkan request `POST /api/auth/logout` (mencabut token ke blacklist backend) dan mengosongkan sesi lokal setelah 15 menit pasif.
- Menambahkan alert banner pemberitahuan keamanan inaktivitas pada `src/app/login/page.tsx`.
timestamp: 2026-08-01 01:30 WITA
kategori: Feature, Security, Auth
file yang diubah:
- frontend/src/context/AuthContext.tsx
- frontend/src/app/login/page.tsx
- frontend/changelog.md
alasan perubahan: Meningkatkan standar keamanan infrastruktur kritis IKN dengan memastikan sesi pengguna pasif diakhiri secara otomatis dan token dicabut dari server.

---

## [0.2.1] - 2026-08-01

### Fix Session Storage Restoration & Direct Route Navigation

perubahan yang dilakukan: **Penguatan Sesi AuthContext & Direct Navigation /dashboard**
before:
- Pemulihan sesi pada `useEffect` `AuthContext.tsx` tidak memiliki penanganan `try...catch` dan pemeriksaan string invalid (`undefined`/`null`), sehingga potensi error `JSON.parse` membatalkan proses pembacaan sesi.
- Fungsi `login` melakukan navigasi ke `/` yang kemudian di-redirect ke `/dashboard`.
after:
- Menambahkan penanganan `try...catch...finally` dan pembersihan token invalid pada `useEffect` `AuthContext.tsx`.
- Mengubah alur navigasi `login` agar langsung mengarah ke `/dashboard`.
timestamp: 2026-08-01 01:25 WITA
kategori: Bugfix, Auth, Session
file yang diubah:
- frontend/src/context/AuthContext.tsx
- frontend/changelog.md
alasan perubahan: Memastikan sesi pengguna pulih secara stabil saat berpindah halaman dan mencegah pemutusan sesi tak terduga.

---

## [0.2.0] - 2026-08-01

### Cetak QR Code Massal (Batch PDF / Print Label Export)

perubahan yang dilakukan: **Integrasi Fitur Cetak QR Code Massal & Floating Bulk Bar Action**
before:
- Pencetakan label QR Code hanya dapat dilakukan satu per satu via modal detail CCTV (`handlePrintQR`).
- Floating Bulk Action Bar hanya menyediakan tombol hapus massal ("Hapus [N] CCTV").
- Tidak ada opsi untuk mencetak seluruh stiker QR Code sekaligus dari daftar master CCTV.
after:
- Diimplementasikan fungsi `handleBatchPrintQR(targetIDs?: string[])` di `src/app/cctv/page.tsx` yang mendukung pencetakan batch multiple CCTV terpilih maupun seluruh CCTV.
- Ditambahkan tombol **"Cetak [N] QR"** (dengan ikon Printer) pada *Floating Bulk Action Bar* ketika 1 atau lebih CCTV dicentang pada tabel.
- Ditambahkan opsi **"Cetak Semua QR Code"** pada menu dropdown tindakan di header tabel CCTV.
- Tata letak cetak (*Print Layout*) dirancang menggunakan grid stiker label berukuran **92mm x 35mm** (proporsional untuk kertas A4 / stiker label 100x38mm) lengkap dengan Logo Nusantara, QR Code terenkripsi, Kode CCTV, Nama CCTV, dan Lokasi Area dengan proteksi `@media print` & `break-inside: avoid`.
timestamp: 2026-08-01 00:44 WITA
kategori: Feature, UI/UX, Batch Processing
file yang diubah:
- frontend/src/app/cctv/page.tsx
- frontend/changelog.md
alasan perubahan: Memudahkan Admin dan Teknisi menyebarkan stiker QR Code fisik di lapangan IKN dengan mencetak puluhan label QR Code secara sekaligus atau menyimpannya langsung sebagai PDF melalui dialog print browser.

---

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
