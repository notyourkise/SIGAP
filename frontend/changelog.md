# Changelog - Frontend SIGAP

## [1.25.7] - 2026-08-04

### Peningkatan: Sortir & Filter Daftar CCTV dalam Modal Popup

perubahan yang dilakukan: **Select filter status, filter indoor/outdoor, dan select sortir di halaman `/cctv` disembunyikan dari tampilan utama dan dipindah ke modal "Sortir & Filter" (pola sama dengan `/maintenance`, `/field-maintenance`, `/audit-logs`); kotak pencarian tetap tampil; opsi pengurutan diperluas (sebelumnya hanya nama A-Z/Z-A, kini 5 kolom)**
before: 3 select (status, indoor/outdoor, urutan) tampil di grid baris dengan kotak pencarian; opsi urutan hanya "Terbaru / Nama A-Z / Nama Z-A" (mapping `sort_by=nama`).
after:
- Tombol "Sortir" (ikon `SlidersHorizontal`) di header bagian "Daftar CCTV" + badge jumlah filter/sortir aktif.
- Modal "Sortir & Filter": kolom pengurutan (Waktu ditambahkan/Nama/Kode CCTV/Status/Lokasi area — whitelist backend: id, kode_cctv, nama, lokasi_area, status, created_at, updated_at) + arah (terbaru→terlama / terlama→terbaru) + filter status (Aktif/Nonaktif/Pemeliharaan) + lokasi pemasangan (Indoor/Outdoor); pola draft → Terapkan/Batal/Reset; Escape/klik luar menutup + scroll-lock body.
- State `sortMode` (`newest|name-asc|name-desc`) diganti `sortBy` + `sortOrder`; `fetchCCTVs` mengirim `sort_by` & `order`; default `created_at desc` setara perilaku lama "Terbaru".
- Kotak pencarian tetap tampil; seleksi massal & fitur lain tidak berubah.
timestamp: 2026-08-04 WITA
kategori: Sedang, Frontend
file yang diubah:
- frontend/src/app/cctv/page.tsx
- frontend/changelog.md



### Peningkatan: Sortir & Filter Audit Log dalam Modal Popup + Posisi Menu Pemberitahuan

perubahan yang dilakukan: **Filter modul & rentang tanggal di halaman `/audit-logs` disembunyikan dari tampilan utama dan dipindah ke modal "Sortir & Filter" (pola sama dengan `/maintenance` & `/field-maintenance`); kotak pencarian tetap tampil; menu sidebar "Pemberitahuan" dipindah ke posisi setelah "Audit Log" dan tepat di atas tombol "Keluar"**
before: filter bar 3 kontrol (modul, dari tanggal, sampai tanggal) tampil di bawah kotak pencarian; menu Pemberitahuan berada setelah "Pemeliharaan CCTV" (di tengah daftar menu). Selain itu halaman audit-logs tidak pernah menampilkan baris data karena backend merespons `data: [array]` sementara frontend membaca `data.rows` (bug tersembunyi — diperbaiki lewat backend changelog [0.26.2]).
after:
- Tombol "Sortir" (ikon `SlidersHorizontal`) di header bagian "Riwayat Aktivitas" + badge jumlah filter/sortir aktif.
- Modal "Sortir & Filter": kolom pengurutan (Waktu kejadian/Aksi/Modul/Pengguna/Peran — whitelist backend) + arah + filter modul (AUTH/CCTV/MAINTENANCE/USER) + rentang tanggal; pola draft → Terapkan/Batal/Reset; Escape/klik luar menutup + scroll-lock.
- Kotak pencarian teks tetap tampil langsung; tombol ekspor CSV tetap di baris terpisah.
- `fetchAuditLogs` kini mengirim `sort_by` & `order`.
- Sidebar: "Pemberitahuan" dipindah ke akhir daftar menu utama (bawah "Audit Log" untuk admin), tepat di atas tombol "Keluar" — untuk semua role.
timestamp: 2026-08-04 WITA
kategori: Sedang, Frontend
file yang diubah:
- frontend/src/app/audit-logs/page.tsx
- frontend/src/components/DashboardLayout.tsx
- frontend/changelog.md



### Redesign: Modal Detail Pengguna di Halaman `/users`

perubahan yang dilakukan: **Modal detail pengguna didesain ulang dari daftar field polos (ikon + 4 baris grid) menjadi kartu identitas bertingkat: header avatar inisial (warna sesuai peran), nama + badge peran, username mono; panel rincian akun (email, tanggal akun dibuat) dalam satu grid berlapis; dan footer aksi cepat (Ubah data, Atur ulang kata sandi, Hapus pengguna — merah terpisah)**
before: `Modal` generik dengan judul "Detail Pengguna", satu baris berisi ikon `UserCog` + grid 2 kolom berisi 4 pasangan label-nilai (Nama lengkap, Username, Email, Peran); tanpa tanggal dibuat, tanpa aksi, tanpa hierarki visual.
after:
- Header identitas: avatar inisial 56px (rounded-xl) — `bg-warning/10 text-warning` untuk admin, `bg-primary/10 text-primary` untuk teknisi; nama (text-xl bold) + chip pill peran (kuning/hijau sesuai tabel); username `@username` mono; tombol tutup.
- Panel rincian akun: satu `dl` grid 2 kolom dengan pemisah border (pola sama dengan modal detail pemeliharaan) — Email (ikon `Mail`, `break-all`) & Akun dibuat (ikon `CalendarDays`, format `id-ID` panjang).
- Footer aksi cepat setelah separator: "Hapus pengguna" (bordered merah + hover `bg-error/10`) di kiri; "Ubah data" (secondary) & "Atur ulang kata sandi" (primary) di kanan — langsung berpindah ke modal edit/password yang sudah ada (tanpa fetch ulang), hapus memakai confirm dialog lalu menutup modal & memuat ulang daftar.
- Overlay kini `backdrop-blur-sm` + shadow besar (pola modal terbaru); layout responsif (footer menumpuk di mobile).
timestamp: 2026-08-04 WITA
kategori: Sedang, Frontend
file yang diubah:
- frontend/src/app/users/page.tsx
- frontend/changelog.md



### Peningkatan: Sortir & Filter Pengecekan Lapangan Dipindah ke Modal Popup

perubahan yang dilakukan: **Filter hasil pengecekan (status) & rentang tanggal di halaman `/field-maintenance` disembunyikan dari tampilan utama dan dipindah ke dalam modal "Sortir & Filter" (pola sama dengan halaman `/maintenance`); modal kini juga berisi pengaturan pengurutan (kolom + arah) yang dikirim ke backend lewat `sort_by` & `order`; kotak pencarian teks tetap tampil di atas tabel**
before: status select + dua input tanggal tampil langsung di atas tabel (grid 4 kolom); tidak ada kontrol pengurutan (list selalu `sort_by=created_at&order=desc` dari default backend).
after:
- Tombol "Sortir" (ikon `SlidersHorizontal`) di header bagian "Daftar Hasil Pengecekan" + badge jumlah filter/sortir aktif.
- Modal "Sortir & Filter": kolom pengurutan (Waktu pengecekan/Waktu selesai/Hasil keseluruhan — whitelist backend: id, cctv_id, technician_id, overall_status, created_at, completed_at) + arah (terbaru→terlama / terlama→terbaru) + filter lengkap (hasil pengecekan, rentang tanggal) dengan pola draft → Terapkan/Batal/Reset.
- Kotak pencarian (CCTV/kode/catatan) tetap terlihat langsung untuk alur pencarian cepat.
- `fetchEntries` kini menyertakan `sort_by` & `order`; modal menutup via Escape/klik luar dengan scroll-lock body.
timestamp: 2026-08-04 WITA
kategori: Sedang, Frontend
file yang diubah:
- frontend/src/app/field-maintenance/page.tsx
- frontend/changelog.md



### Peningkatan: Sortir & Filter Pemeliharaan Dipindah ke Modal Popup

perubahan yang dilakukan: **Grid filter (CCTV, Status, rentang tanggal) di halaman `/maintenance` disembunyikan dari tampilan utama dan dipindah ke dalam modal "Sortir & Filter" yang terbuka lewat tombol Sortir di header tabel; modal kini juga berisi pengaturan pengurutan (kolom + arah) yang dikirim ke backend lewat `sort_by` & `order`**
before: filter tampil langsung di atas tabel; tidak ada kontrol pengurutan sama sekali (list selalu `sort_by=created_at&order=desc` dari default backend); filter bar memakan ruang vertikal halaman.
after:
- Tombol "Sortir" (ikon `SlidersHorizontal`) di header bagian "Riwayat Pemeliharaan"; badge angka menunjukkan jumlah filter/sortir yang aktif; tombol dinonaktifkan saat data sedang dimuat.
- Modal "Sortir & Filter": kolom pengurutan (Waktu pencatatan/Waktu terjadwal/Waktu selesai/Status/Pekerjaan A-Z) + arah (terbaru→terlama / terlama→terbaru) + filter lengkap (CCTV, Status, rentang tanggal) dengan pola draft → tombol Terapkan (perubahan diterapkan sekaligus, halaman kembali ke halaman 1) / Batal (draft dibuang) / Reset (kembali ke `created_at desc`, semua filter kosong).
- `fetchLogs` kini menyertakan `sort_by` & `order` (whitelist backend: id, cctv_id, technician_id, action_taken, notes, status, scheduled_date, completed_at, created_at, updated_at).
- Modal memakai pola yang sama dengan modal lain di halaman (backdrop blur, tutup via Escape/klik luar, scroll-lock body).
timestamp: 2026-08-04 WITA
kategori: Sedang, Frontend
file yang diubah:
- frontend/src/app/maintenance/page.tsx
- frontend/changelog.md



### Bugfix: Pemberitahuan Tidak Pernah Tampil (Salah Bungkus Respons API)

perubahan yang dilakukan: **`NotificationBell` dan halaman `/pemberitahuan` membaca daftar dari `res.data?.data`, padahal respons `GET /api/notifications` menempatkan array di `data` (kini `data.rows` sesuai konvensi list SIGAP) — hasilnya daftar selalu kosong ("Belum ada pemberitahuan") meskipun notifikasi tersimpan di database; pembacaan diubah menjadi `res.data?.rows` (dan pagination dari `res.data.pagination`)**
before:
- `NotificationBell.tsx`: `apiRequest<{ data: NotificationItem[] }>` lalu `setNotifications(res.data?.data ?? [])` — `res.data` adalah array, sehingga `.data` selalu `undefined` → dropdown kosong.
- `app/pemberitahuan/page.tsx`: membaca `res.data?.data` & `res.data?.pagination` dengan cara yang sama — halaman selalu kosong & total halaman 0.
- Akibat: badge unread-count tetap benar (bentuk `data: { unread_count }` cocok), tapi daftar & riwayat tidak pernah tampil untuk admin maupun teknisi.
after:
- `NotificationBell.tsx`: `apiRequest<{ rows: NotificationItem[] }>` → `setNotifications(res.data?.rows ?? [])`.
- `app/pemberitahuan/page.tsx`: `NotificationsResponse` kini `{ rows, pagination }`; `setNotifications(res.data?.rows ?? [])`, total rows/pages dari `res.data?.pagination`.
- Backend `GET /api/notifications` diselaraskan ke konvensi `data: { rows, pagination }` (lihat backend changelog [0.26.1]).
timestamp: 2026-08-04 WITA
kategori: Bugfix, Frontend
file yang diubah:
- frontend/src/components/NotificationBell.tsx
- frontend/src/app/pemberitahuan/page.tsx
- frontend/changelog.md



### Pemberitahuan: Lonceng di Dashboard, Menu Sidebar & Halaman `/pemberitahuan` Terpisah

perubahan yang dilakukan: **Posisi lonceng notifikasi dipindah dari header/sidebar global ke halaman Dashboard (di samping indikator layanan); dropdown lonceng kini hanya menampilkan 3 pemberitahuan teratas + tombol "Lihat lainnya" yang mengarah ke halaman Pemberitahuan baru (`/pemberitahuan`, ada di menu sidebar) berisi riwayat lengkap milik user yang login saja — teknisi tidak bisa melihat pemberitahuan teknisi lain (scope tetap dari klaim JWT di backend)**
before:
- `NotificationBell` dirender di header mobile & sidebar desktop (`DashboardLayout`) untuk semua halaman; dropdown menampilkan 10 notifikasi dan tidak ada halaman daftar lengkap.
- Tidak ada menu/halaman khusus pemberitahuan.
after:
- `src/components/DashboardLayout.tsx`: `NotificationBell` dihapus dari header mobile & sidebar; item navigasi baru "Pemberitahuan" (`/pemberitahuan`, ikon Bell) ditambahkan untuk admin & teknisi.
- `src/components/NotificationBell.tsx`: preview dropdown dibatasi 3 notifikasi terbaru (`limit=3`); tombol "Lihat lainnya" di footer dropdown → `router.push('/pemberitahuan')`; badge unread, real-time `NOTIFICATION_CREATED`, dan polling 60 detik tetap.
- `src/app/dashboard/page.tsx`: `NotificationBell` dirender di header halaman Dashboard (sebelum indikator "Layanan normal").
- `src/app/pemberitahuan/page.tsx` (baru): halaman daftar lengkap pemberitahuan milik user — paginasi (10/halaman), filter "Belum dibaca saja", tombol "Tandai semua dibaca", klik item → tandai dibaca + navigasi ke `/maintenance` atau `/field-maintenance`, refresh real-time, empty state.
timestamp: 2026-08-04 WITA
kategori: Sedang, Frontend
file yang diubah:
- frontend/src/components/DashboardLayout.tsx
- frontend/src/components/NotificationBell.tsx
- frontend/src/app/dashboard/page.tsx
- frontend/src/app/pemberitahuan/page.tsx (baru)
- frontend/changelog.md



### Notifikasi In-App Real-Time: NotificationBell di Header & Sidebar (Fase 5.1)

perubahan yang dilakukan: **Komponen `NotificationBell` baru — ikon lonceng dengan badge jumlah belum dibaca, dropdown daftar notifikasi (10 terbaru), tombol "Tandai semua dibaca", klik item → tandai dibaca + navigasi ke halaman terkait (`/maintenance` atau `/field-maintenance`); terima notifikasi baru real-time via WebSocket (`NOTIFICATION_CREATED`) dengan toast + refresh badge, fallback polling 60 detik saat koneksi terputus**
before: Tidak ada akses notifikasi di antarmuka; penugasan/penyelesaian tugas hanya terlihat saat membuka halaman terkait secara manual.
after:
- `src/components/NotificationBell.tsx` (baru): bell + badge `99+`; dropdown (border/rounded konsisten desain SIGAP, `line-clamp-2`, waktu `id-ID`); subscribe `NOTIFICATION_CREATED` → `toast.info(title)` + muat ulang; polling `unread-count` tiap 60 detik; tutup saat klik luar/Escape; peta tipe → rute tujuan (`field-maintenance` → `/field-maintenance`, lainnya → `/maintenance`); mark-read fire-and-forget agar navigasi tidak tertahan.
- `src/components/DashboardLayout.tsx`: `NotificationBell` dirender di header mobile (sebelum tombol tema) dan area kontrol sidebar desktop (sebelum tombol tema/collapse) — tersedia untuk admin & teknisi.
timestamp: 2026-08-04 WITA
kategori: Besar, Frontend
file yang diubah:
- frontend/src/components/NotificationBell.tsx (baru)
- frontend/src/components/DashboardLayout.tsx
- frontend/changelog.md



### Label Status "Gagal" Menjadi "Perlu Tindak Lanjut" (Hanya Tampilan)

perubahan yang dilakukan: **Label tampilan status `failed` diubah dari "Gagal" menjadi "Perlu Tindak Lanjut" di seluruh antarmuka — hanya teks yang berubah, nilai status di API/database tetap `failed` sehingga filter, URL query, dan logika aplikasi tidak berubah**
before: status bernilai `failed` tampil sebagai "Gagal" di badge tabel, filter dropdown, modal detail, kartu statistik dashboard, legenda analitik, dan tooltip grafik tren.
after:
- `src/app/maintenance/page.tsx`: `getStatusLabel` & opsi filter dropdown memakai "Perlu Tindak Lanjut" (badge tabel, modal catat/ubah, dan detail otomatis ikut karena memakai `getStatusLabel`).
- `src/app/dashboard/page.tsx`: map label status & kartu statistik "Perlu Tindak Lanjut".
- `src/components/DashboardAnalytics.tsx`: legenda "Perlu Tindak Lanjut".
- `src/components/TrendChart.tsx`: tooltip "Perlu Tindak Lanjut".
timestamp: 2026-08-03 WITA
kategori: Kecil, Frontend
file yang diubah:
- frontend/src/app/maintenance/page.tsx
- frontend/src/app/dashboard/page.tsx
- frontend/src/components/DashboardAnalytics.tsx
- frontend/src/components/TrendChart.tsx
- backend/handlers/report.go
- backend/changelog.md
- frontend/changelog.md
- docs/changelog.md

## [1.24.1] - 2026-08-03

### Bugfix: Header CSRF Mengambil Cookie Stale (Path /api) Setelah Login Ulang

perubahan yang dilakukan: **`getCookie()` di `src/lib/api.ts` kini mengambil kemunculan cookie TERAKHIR, bukan yang pertama — bila browser masih menyimpan `sigap_csrf` stale warisan versi lama di path `/api`, `document.cookie` mengurutkannya lebih dulu (RFC 6265, path terpanjang pertama) sehingga header `X-CSRF-Token` memakai nilai stale sementara server membandingkan dengan cookie aktif di "/" → mutasi ditolak 403**
before: `getCookie()` memakai `document.cookie.match(...)` yang mengembalikan kecocokan pertama — cookie stale di path lebih panjang menang.
after: regex dengan flag `g` mengambil nilai kecocokan terakhir — cookie aktif di path "/" selalu dipakai untuk header `X-CSRF-Token`, sinkron dengan `ExtractLastCookieValue` di backend.
timestamp: 2026-08-03 WITA
kategori: Bugfix, Frontend
file yang diubah:
- frontend/src/lib/api.ts
- frontend/changelog.md
- backend/changelog.md
- docs/changelog.md

## [1.24.0] - 2026-08-03

### Alur Penugasan Pemeliharaan: Assign Teknisi & Selesaikan Tugas via Checklist 9 Tahap

perubahan yang dilakukan: **Di halaman Pemeliharaan CCTV, sub-menu status "Dalam Proses", "Selesai", "Gagal" pada input status dihapus (status kini tampil read-only & dikelola sistem), log terjadwal dapat ditugaskan ke teknisi lewat dropdown, dan tugas muncul di dashboard teknisi dengan tombol "Selesaikan" membuka wizard checklist 9 tahap**
before:
- Form catat/ubah pemeliharaan menampilkan select status 4 opsi (Selesai/Terjadwal/Dalam proses/Gagal) + input "waktu pencatatan" yang diisi manual — status bisa diubah sesuka hati.
- Log terjadwal tidak punya mekanisme penugasan ke teknisi tertentu; teknisi hanya bisa melihat tugas via dashboard tanpa bisa mengerjakan/menyelesaikannya.
- Nama teknisi pada baris log memakai label "Teknisi Sistem" bila kosong, tidak mencerminkan belum ada penugasan.
after:
- **Halaman `src/app/maintenance/page.tsx`:**
  - Input status diganti tampilan read-only: badge status saat ini (untuk catatan baru selalu "Terjadwal") + keterangan "Status diperbarui otomatis oleh sistem"; field `completed_at` dihapus dari form; input hanya "Waktu terjadwal" (datetime-local, default waktu sekarang; bug lama yang hanya menyimpan tanggal saat edit diperbaiki — kini memakai konversi `toDatetimeLocalInput`).
  - Daftar teknisi (role "teknisi") diambil dari `GET /users?role=teknisi` (admin) untuk dropdown penugasan.
  - Tombol aksi baru **Tugaskan** (ikon UserPlus) pada baris log berstatus "Terjadwal" (admin): modal pilih teknisi → `PUT /maintenance-logs/:id/assign`; setelah ditugaskan status otomatis "Dalam proses" dan nama teknisi tampil pada baris.
  - Tombol aksi baru **Selesaikan** (ikon CheckCheck) pada baris log berstatus "Dalam proses" milik teknisi yang ditugaskan (atau admin): membuka wizard checklist 9 tahap dalam mode "complete".
  - Teknisi tidak lagi melihat tombol "Catat Pemeliharaan" dan "Ubah" — alur teknisi: menerima tugas (dashboard) → mengerjakan → selesaikan; admin tetap bisa membuat/mengubah/menghapus.
  - Baris & detail log menampilkan "Belum ditugaskan" saat `technician` nil.
- **Wizard checklist `src/components/FieldMaintenanceModal.tsx`:** properti baru `mode="complete"` + `maintenanceLogId` — judul "Selesaikan Tugas Pemeliharaan", submit mengirim `POST /maintenance-logs/:id/complete` (tanpa `cctv_id`), toast sukses disesuaikan; seluruh 9 item + foto bukti (min 1, maks 5) tetap sama dengan mode field.
- **Dashboard `src/app/dashboard/page.tsx`:** tipe tugas menambahkan `cctv_id`/`cctv_kode`; untuk teknisi, baris tugas berstatus "Dalam proses" menampilkan tombol **Selesaikan** yang membuka wizard yang sama; setelah disimpan dashboard di-refresh (status tugas berubah menjadi selesai/gagal otomatis).
timestamp: 2026-08-03 WITA
kategori: Feature, UI, Frontend
file yang diubah:
- frontend/src/app/maintenance/page.tsx
- frontend/src/components/FieldMaintenanceModal.tsx
- frontend/src/app/dashboard/page.tsx
- frontend/changelog.md
verifikasi:
- `npm run lint` PASS (0 errors, 46 warnings existing).
- `npm run build` PASS.
alasan teknis: Alur status dikelola sistem (dibuat → terjadwal → ditugaskan → dalam proses → selesai/gagal) sehingga formulir tidak perlu menawarkan pilihan status manual yang rawan salah; penugasan menghubungkan tugas ke teknisi spesifik sehingga dashboard teknisi berisi tugas yang benar-benar menjadi tanggung jawabnya; wizard checklist 9 tahap dipakai ulang dari modul pengecekan lapangan agar standar pelaporan seragam.

---

## [1.23.0] - 2026-08-03

### Pengecekan Lapangan Teknisi via Scan QR + Halaman Admin

perubahan yang dilakukan: **Setelah scan QR CCTV, teknisi bisa langsung mengisi checklist pengecekan lapangan (jaringan, listrik, lensa, dll) satu per satu lengkap dengan foto bukti di tahap akhir; admin melihat semua hasilnya di halaman baru**
before:
- Hasil scan QR hanya menampilkan informasi CCTV (nama, kode, lokasi, IP, dll) tanpa sarana mencatat hasil pengecekan spontan di lapangan.
- Catatan pemeliharaan hanya bisa dibuat lewat form di halaman Pemeliharaan CCTV (jadwal dari admin); tidak ada alur pengecekan langsung di lokasi.
- Admin tidak memiliki tampilan khusus untuk melihat hasil pengecekan lapangan teknisi.
after:
- Tombol **"Maintenance"** (ikon ClipboardCheck) ditambahkan di hasil scan QR CCTV; mengkliknya membuka wizard "Pengecekan Lapangan" (komponen baru `src/components/FieldMaintenanceModal.tsx`).
- Wizard memandu teknisi **satu item per satu**: 9 item checklist wajib (jaringan, listrik, pembersihan lensa, fisik kamera, kualitas gambar, perekaman, akses jarak jauh, kabel & konektor, lingkungan sekitar), masing-masing dengan pilihan "Normal/OK" atau "Bermasalah" + catatan (diwajibkan bila bermasalah); tombol Lanjut nonaktif sampai item dijawab; progress bar & nomor langkah.
- **Tahap akhir wajib foto bukti** (minimal 1, maksimal 5): unggah langsung ke `/api/upload` dengan pratinjau thumbnail & hapus per foto; ditutup ringkasan hasil checklist dan catatan umum sebelum menyelesaikan.
- Halaman baru **`/field-maintenance`** (menu sidebar admin "Pengecekan Lapangan", ikon ClipboardCheck): tabel CCTV, teknisi, tanggal, ringkasan x/y normal, status Normal / Perlu Tindak Lanjut, filter pencarian/status/rentang tanggal, pagination, modal detail berisi seluruh checklist + foto bukti, serta hapus catatan (dengan dialog konfirmasi kustom).
- Halaman dilindungi admin-only via `proxy.ts` (ditambahkan ke `ADMIN_PATHS` + matcher).
timestamp: 2026-08-03 WITA
kategori: Feature, UI, Frontend
file yang diubah:
- frontend/src/components/FieldMaintenanceModal.tsx (baru)
- frontend/src/app/field-maintenance/page.tsx (baru)
- frontend/src/app/cctv/page.tsx (tombol Maintenance di hasil scan + render modal)
- frontend/src/components/DashboardLayout.tsx (menu "Pengecekan Lapangan" admin)
- frontend/src/proxy.ts (guard admin)
- frontend/changelog.md
verifikasi:
- `npm run lint` PASS (0 errors, 46 warnings existing).
- `npm run build` PASS.
alasan teknis: Pengecekan di lapangan bersifat spontan (tidak selalu sesuai jadwal admin) sehingga membutuhkan alur terpisah dari MaintenanceLog; model checklist per-item memungkinkan admin melihat detail apa yang bermasalah, bukan sekadar status. Foto bukti diunggah melalui endpoint upload yang sudah ada (normalisasi JPEG + tanpa EXIF) dan ditampilkan memakai `AuthenticatedImage` (akses via JWT).

---

## [1.22.0] - 2026-08-03

### Notifikasi & Konfirmasi Kustom dengan Animasi

perubahan yang dilakukan: **Semua `alert()`/`confirm()` bawaan browser diganti sistem toast & dialog konfirmasi kustom beranimasi**
before:
- 24 titik memakai `alert()` (peringatan/pesan sukses) dan `confirm()` (konfirmasi hapus) bawaan browser yang polos, tidak konsisten dengan desain, dan tanpa animasi.
after:
- Komponen baru `src/components/Toast.tsx`: provider `ToastProvider` + hook `useToast()` dengan `toast.success()` / `toast.error()` / `toast.info()`. Toast tampil di kanan atas: kartu dengan ikon lingkaran, **animasi check hijau (stroke-draw SVG)** untuk sukses, ikon X merah untuk error, info biru; slide-in elastis (`toastIn`), pop ikon (`checkPop`), progress bar yang menyusut, auto-dismiss 4,2 detik, maksimal 4 toast.
- Komponen baru `src/components/ConfirmDialog.tsx`: provider `ConfirmProvider` + hook `useConfirm()` (Promise<boolean>). Dialog kustom dengan ikon peringatan beranimasi goyang halus, fade + pop in/out, tombol Batal/Konfirmasi (merah untuk aksi destruktif), tutup via X/Escape/klik latar.
- Kedua provider dipasang di `src/app/layout.tsx` (z-index di atas modal lain: toast z-80, dialog z-70).
- Semua halaman diganti: users (simpan, reset password, hapus), maintenance (simpan, ekspor CSV, laporan PDF, hapus), cctv (simpan, hapus, bulk delete, cetak QR, template, impor). Toast sukses ditambahkan di semua aksi yang berhasil (sebelumnya beberapa hanya menutup modal tanpa umpan balik).
- Animasi baru ditambahkan di `globals.css`: `toastIn`, `toastProgress`, `checkPop`, `checkCircle`, `checkPath`, `fadeIn/Out`, `popIn/Out`, `wiggleSlow`.
timestamp: 2026-08-03 WITA
kategori: Feature, UI, Frontend
file yang diubah:
- frontend/src/components/Toast.tsx (baru)
- frontend/src/components/ConfirmDialog.tsx (baru)
- frontend/src/app/layout.tsx
- frontend/src/app/globals.css
- frontend/src/app/users/page.tsx
- frontend/src/app/maintenance/page.tsx
- frontend/src/app/cctv/page.tsx
- frontend/changelog.md
verifikasi:
- `npm run lint` PASS (0 errors, 38 warnings existing).
- `npm run build` PASS.
alasan teknis: Menghilangkan seluruh UI native browser demi konsistensi bahasa desain, aksesibilitas (role/aria), dan umpan balik visual yang jelas (check animasi untuk sukses) serta mencegah logout/hapus tidak sengaja via dialog konfirmasi yang dapat dikustomisasi.

---

### Redesain Filter Audit Log (Input Tidak Lagi Terpotong)

perubahan yang dilakukan: **Tata letak filter & search bar audit log dirombak agar setiap input mendapat ruang cukup**
before:
- Grid filter memakai 4 kolom (`lg:grid-cols-4`); dua input tanggal + teks "s.d." dipaksakan masuk dalam satu sel kolom sehingga input terpotong/tertekan di layar laptop.
after:
- Grid diubah menjadi `lg:grid-cols-3`: baris pencarian full-width di atas, lalu kolom "Modul", "Dari tanggal", dan "Sampai tanggal" masing-masing satu kolom penuh dengan label (`<label>` + `htmlFor`) sehingga jelas dan lega.
- Teks "s.d." di tengah input tanggal digantikan label "Dari tanggal"/"Sampai tanggal".
timestamp: 2026-08-03 WITA
kategori: UI, Frontend
file yang diubah:
- frontend/src/app/audit-logs/page.tsx
- frontend/changelog.md
verifikasi:
- `npm run lint` PASS (0 errors, 38 warnings existing).
- `npm run build` PASS.
alasan teknis: Dua input `type="date"` (masing-masing butuh ~150px) tidak muat dalam satu sel grid 4 kolom (~230px); memberi satu kolom penuh per input dan label di atasnya menghilangkan pemotongan dan memperjelas fungsi tiap field.

---

### Perbaikan Cetak Semua QR Code Hanya 10 CCTV

perubahan yang dilakukan: **"Cetak Semua QR Code" kini mengambil seluruh data CCTV dari backend, bukan hanya halaman aktif**
before:
- Tombol "Cetak Semua QR Code" (menu aksi di halaman Aset CCTV) memakai state `cctvs` yang hanya berisi data halaman aktif (10 baris per halaman), sehingga hanya 10 label QR yang tercetak padahal total perangkat 35.
after:
- Ketika dipanggil tanpa ID target, halaman meminta seluruh data dari backend dengan `GET /cctv?page=1&limit=100` (batas maksimum backend); hasilnya difilter dan digunakan sebagai antrian cetak penuh.
- Fallback ke data halaman aktif hanya jika permintaan gagal.
timestamp: 2026-08-03 WITA
kategori: Bugfix, Frontend
file yang diubah:
- frontend/src/app/cctv/page.tsx
- frontend/changelog.md
verifikasi:
- `npm run lint` PASS (0 errors, 38 warnings existing).
- `npm run build` PASS.
alasan teknis: State `cctvs` dibatasi paginasi (10 per halaman); cetak semua memerlukan dataset lengkap sehingga diambil ulang dari API dengan limit maksimum yang diizinkan server (100).

---

### Konfirmasi Logout dengan Dialog Kustom

perubahan yang dilakukan: **Klik "Keluar" kini membuka dialog konfirmasi kustom (bukan logout langsung)**
before:
- Item menu "Keluar" di sidebar langsung menjalankan logout tanpa konfirmasi; risiko keluar tidak sengaja.
after:
- Klik "Keluar" membuka dialog konfirmasi kustom (modal) dengan: ikon `LogOut` dalam lingkaran merah, judul "Keluar dari SIGAP?", deskripsi penjelas, tombol "Batal" (netral) dan "Keluar" (merah, autoFocus).
- Modal dapat ditutup via tombol X, klik area latar, atau tekan `Escape`; dialog memakai pola `role="dialog"` + `aria-modal` untuk aksesibilitas.
- Logout hanya dieksekusi setelah konfirmasi; drawer mobile ikut ditutup saat logout disetujui.
timestamp: 2026-08-03 WITA
kategori: Feature, UI, Frontend
file yang diubah:
- frontend/src/components/DashboardLayout.tsx
- frontend/changelog.md
verifikasi:
- `npm run lint` PASS (0 errors, 38 warnings existing).
- `npm run build` PASS.
alasan teknis: Konfirmasi dialog kustom mencegah logout tidak disengaja dan konsisten dengan bahasa desain aplikasi (modal, backdrop blur, tombol danger merah).

---

### Card Tugas Menggantikan Distribusi Status CCTV

perubahan yang dilakukan: **Card "Distribusi Status CCTV" dihapus, card tugas naik ke posisinya di samping "Tren Pemeliharaan"**
before:
- Dashboard menampilkan card "Distribusi Status CCTV" (bar persentase Aktif/Nonaktif/Pemeliharaan) di kolom kanan grid analitik, sedangkan card "Tugas Berjalan Semua Teknisi"/"Tugas Saya" berada sendiri di bawah grid.
after:
- Card "Distribusi Status CCTV" dihapus total dari `DashboardAnalytics`.
- Card tugas kini dirender sebagai slot `children` di dalam grid analitik, menggantikan posisi card distribusi status — berdampingan dengan "Tren Pemeliharaan"; card "Peringatan Pemeliharaan Terlambat" tetap full-width di bawahnya.
- Interface `DashboardSummary` dirapikan (field `cctv_status` dan helper `cctvStatusLabel` yang tidak terpakai dihapus).
timestamp: 2026-08-03 WITA
kategori: UI, Frontend
file yang diubah:
- frontend/src/components/DashboardAnalytics.tsx
- frontend/src/app/dashboard/page.tsx
- frontend/changelog.md
verifikasi:
- `npm run lint` PASS (0 errors, 38 warnings existing).
- `npm run build` PASS.
alasan teknis: Mengurangi redundansi (distribusi status sudah terwakili ringkasan di atas), dan menaikkan card tugas yang lebih relevan untuk pemantauan pekerjaan ke area utama dashboard.

---

### Logout Hanya di Sidebar & Transisi Drawer Lebih Halus

perubahan yang dilakukan: **Tombol logout di header mobile dihapus + animasi buka/tutup sidebar diperhalus**
before:
- Header mobile (dekat tombol hamburger) masih memiliki tombol logout ikon `LogOut` merah, sehingga ada dua jalur logout (header + item menu sidebar).
- Drawer sidebar membuka/menutup dengan transisi 300ms linear dan overlay hanya muncul setelah kondisi berubah (tanpa animasi keluar).
after:
- Tombol logout di header mobile dihapus total; logout hanya tersedia melalui item menu "Keluar" di dalam sidebar.
- Transisi sidebar memakai `duration-500 ease-in-out` untuk sliding yang lebih halus; overlay kini selalu ter-mount (`md:hidden`) dengan fade `transition-opacity duration-500` + `opacity-0`/`pointer-events-none` saat tertutup, sehingga efek buka dan tutup sama-sama beranimasi.
timestamp: 2026-08-03 WITA
kategori: UI, Frontend
file yang diubah:
- frontend/src/components/DashboardLayout.tsx
- frontend/changelog.md
verifikasi:
- `npm run lint` PASS (0 errors, 38 warnings existing).
- `npm run build` PASS.
alasan teknis: Menghindari duplikasi aksi logout yang membingungkan dan memperhalus gerakan drawer dengan easing yang lebih natural serta fade overlay dua arah.

---

### Tombol Keluar Sebagai Item Menu di Sidebar

perubahan yang dilakukan: **Tombol logout dipindah ke dalam daftar navigasi, di bawah menu Audit Log, dengan gaya yang sama seperti item menu lain**
before:
- Tombol logout terpisah di bagian bawah sidebar (di bawah garis pembatas `border-t`), dengan gaya khusus merah dan label "Keluar dari SIGAP".
after:
- Tombol logout kini menjadi item terakhir di daftar navigasi sidebar — tepat di bawah "Audit Log" untuk admin (dan di bawah "Pemeliharaan CCTV" untuk teknisi) — dengan class yang identik dengan item menu lain (`min-h-11`, `rounded-xl`, `text-muted hover:bg-background hover:text-foreground`), ikon `LogOut`, dan label singkat "Keluar".
- Item "Keluar" menutup drawer mobile saat diklik dan tetap kompatibel dengan mode sidebar diciutkan (ikon saja).
timestamp: 2026-08-03 WITA
kategori: UI, Frontend
file yang diubah:
- frontend/src/components/DashboardLayout.tsx
- frontend/changelog.md
verifikasi:
- `npm run lint` PASS (0 errors, 38 warnings existing).
- `npm run build` PASS.
alasan teknis: Meletakkan logout sebagai bagian dari menu navigasi membuatnya konsisten dengan pola item menu lain dan selalu terlihat tanpa menggantung di bagian bawah, sekaligus tetap tersedia di drawer mobile.

---

### Penyesuaian Antarmuka untuk Peran Teknisi

perubahan yang dilakukan: **Tombol keluar di mobile, tombol PDF/CSV jadi ikon, payload QR disembunyikan untuk teknisi, dan ringkasan tugas teknisi**
before:
- Di layar kecil (HP) tidak ada tombol keluar pada sidebar; logout hanya tersedia di sidebar
  desktop, dan pada perangkat pendek tombol logout bisa tergeser keluar layar karena nav tidak bisa digulir.
- Tombol "Laporan PDF" dan "Ekspor CSV" di halaman Pemeliharaan (dan Audit Log) berupa tombol teks lebar.
- Modal Pindai QR menampilkan tombol "Masukkan payload" (input manual payload terenkripsi) untuk semua peran, termasuk teknisi.
- Card "Tugas Saya" teknisi menampilkan teks berlebihan: lokasi CCTV, tindakan (action_taken), dan baris "Jadwal: ...".
after:
- Header mobile (di samping toggle tema) kini memiliki tombol keluar berikon `LogOut` berwarna merah; sidebar mendapat `flex-1 min-h-0 overflow-y-auto` sehingga tombol logout tetap terjangkau pada layar pendek.
- Tombol Laporan PDF kini ikon `FileText` merah dan Ekspor CSV ikon `FileSpreadsheet` hijau (konsisten di halaman Pemeliharaan dan Audit Log), keduanya kotak ikon-only dengan `title`/`aria-label`.
- Toggle "Masukkan payload" dan view input manual hanya dirender untuk admin; teknisi hanya melihat pemindai kamera (`!isAdmin || !showManualInput`).
- "Tugas Saya" teknisi dirampingkan: hanya nama CCTV + badge jadwal (mono) + badge status + badge terlambat; lokasi dan action_taken tetap tampil untuk admin. Subtitle diperpendek.
timestamp: 2026-08-03 WITA
kategori: Feature, UI, Frontend
file yang diubah:
- frontend/src/components/DashboardLayout.tsx
- frontend/src/app/maintenance/page.tsx
- frontend/src/app/audit-logs/page.tsx
- frontend/src/app/cctv/page.tsx
- frontend/src/app/dashboard/page.tsx
- frontend/changelog.md
verifikasi:
- `npm run lint` PASS (0 errors, 38 warnings existing).
- `npm run build` PASS.
alasan teknis: Akses payload QR terenkripsi dibatasi ke admin demi prinsip least privilege; tombol ikon-only menghemat ruang header pada layar kecil; sidebar scrollable menjamin aksi logout selalu tersedia; pengurangan teks menurunkan beban kognitif peran teknisi di lapangan.

---

### Waktu Pencatatan dengan Picker Kalender

perubahan yang dilakukan: **Input teks diganti `datetime-local` + perbaikan zona waktu default**
before:
- Field "Waktu pencatatan"/"Waktu terjadwal" berupa input teks bebas dengan placeholder
  `YYYY-MM-DD HH:MM:SS`; tidak ada kalender, mudah salah format.
- Nilai default memakai `new Date().toISOString()` yang menghasilkan waktu **UTC** (bukan waktu
  lokal Indonesia), sehingga tampilan bisa meleset beberapa jam.
after:
- Field memakai `type="datetime-local"`: kalender bawaan browser untuk memilih hari + picker jam,
  nilai default otomatis waktu saat ini (format disimpan tetap `YYYY-MM-DD HH:MM:SS`).
- Helper baru: `nowLocalDateTime()` (waktu lokal WITA/browser), `toDatetimeLocalInput()`, dan
  `fromDatetimeLocalInput()` untuk konversi format dua arah.
- Teks bantu "Terisi otomatis waktu saat ini — buka kalender untuk memilih hari."
timestamp: 2026-08-03 WITA
kategori: Feature, Bugfix, Frontend, UI
file yang diubah:
- frontend/src/app/maintenance/page.tsx
- frontend/changelog.md
verifikasi:
- `npm run lint` PASS dengan warning existing.
- `npm run build` PASS.
alasan teknis: Input native datetime-local menghilangkan kesalahan format manual dan menjaga
konsistensi nilai yang dikirim ke backend (`YYYY-MM-DD HH:MM:SS`), sekaligus memperbaiki default
waktu yang sebelumnya berada di zona UTC.

---

## [1.13.0] - 2026-08-03

### Tren Pemeliharaan Menjadi Spline Area Chart

perubahan yang dilakukan: **Bar chart CSS diganti grafik area melengkung halus dengan gradasi biru**
before:
- Card "Tren Pemeliharaan" menampilkan bar chart bertumpuk per status (CSS div) dengan tinggi
  proporsional dan tooltip native.
after:
- Komponen baru `TrendChart` (`frontend/src/components/TrendChart.tsx`): Spline Area Chart berbasis
  SVG murni tanpa dependency baru.
- Garis tren total pekerjaan per bulan digambar melengkung halus (Catmull-Rom spline → cubic
  Bézier), area di bawah garis diisi gradasi biru transparan.
- Responsif via `ResizeObserver`, grid sumbu Y, label bulan, titik data, penanda hover dengan
  garis putus-putus, dan tooltip rincian per status (Terjadwal/Dalam proses/Selesai/Gagal).
- Legend bar chart per status diganti legend "Total pekerjaan" + petunjuk hover.
- Deskripsi card disesuaikan menjadi "Total pekerjaan per bulan — 6 bulan terakhir."
timestamp: 2026-08-03 WITA
kategori: Feature, Frontend, UI
file yang diubah:
- frontend/src/components/TrendChart.tsx (baru)
- frontend/src/components/DashboardAnalytics.tsx
- frontend/changelog.md
verifikasi:
- `npm run lint` PASS dengan warning existing.
- `npm run build` PASS.
alasan teknis: Visualisasi tren berbentuk area dengan garis halus lebih mudah dibaca untuk melihat
pergerakan total pekerjaan antar bulan; implementasi SVG mandiri menghindari dependency chart
eksternal yang menambah ukuran bundle dan risiko keamanan.

---

## [1.12.0] - 2026-08-03

### Posisi Rincian Ringkasan CCTV & Warna Bar Distribusi Status

perubahan yang dilakukan: **Detail respon kartu ringkasan dipindah ke bawah kartu; warna bar status dibuat cerah**
before:
- Klik kartu Total CCTV/Aktif/Nonaktif menampilkan rincian (`cctv-summary-details`) di bagian
  paling bawah halaman — setelah analitik, tugas, dan pemeliharaan — sehingga respon jauh dari
  kartu yang diklik.
- Bar progress "Distribusi Status CCTV" memakai warna tema yang redup (`bg-success`/`bg-error`).
after:
- Section rincian kini dirender tepat setelah kartu ringkasan CCTV, sebelum
  `DashboardAnalytics`, sehingga respon tampil langsung di bawah kartu yang diklik.
- Bar progress aktif menjadi hijau terang (`bg-emerald-500`), nonaktif/rusak merah terang
  (`bg-red-500`), pemeliharaan kuning cerah (`bg-amber-500`) — konsisten dengan marker peta.
- Pencocokan status dibuat case-insensitive: backend mengirim `ACTIVE`/`INACTIVE` (kapital),
  sedangkan pengecekan sebelumnya memakai `aktif` (huruf kecil) sehingga bar selalu jatuh ke
  warna default. Label status juga dinormalisasi menjadi "Aktif"/"Nonaktif"/"Pemeliharaan".
timestamp: 2026-08-03 WITA
kategori: Feature, Frontend, UI
file yang diubah:
- frontend/src/app/dashboard/page.tsx
- frontend/src/components/DashboardAnalytics.tsx
- frontend/changelog.md
verifikasi:
- `npm run lint` PASS dengan warning existing.
- `npm run build` PASS.
alasan teknis: Memperpendek jarak umpan balik (feedback loop) saat pengguna mengeksplorasi
ringkasan CCTV, serta meningkatkan keterbacaan proporsi status pada bar chart.

---

## [1.11.0] - 2026-08-03

### Mode Layar Penuh untuk Peta Sebaran CCTV

perubahan yang dilakukan: **Tombol fullscreen pada peta sebaran CCTV menggunakan Fullscreen API**
before:
- Peta sebaran CCTV hanya tampil di dalam kartu (tinggi 460px) tanpa opsi memperbesar tampilan.
after:
- Tombol fullscreen (ikon Maximize/Minimize) ditambahkan di pojok kanan atas peta.
- Mode layar penuh memakai Fullscreen API browser: peta mengisi seluruh layar (100vw x 100vh),
  border/radius dihilangkan, dan dapat ditutup lewat tombol atau tombol Esc.
- Ikon tombol ikut berubah mengikuti status fullscreen via event `fullscreenchange`.
- Import ikon `Camera` yang tidak terpakai dihapus.
timestamp: 2026-08-03 WITA
kategori: Feature, Frontend, UI
file yang diubah:
- frontend/src/components/CCTVMap.tsx
- frontend/changelog.md
verifikasi:
- `npm run lint` PASS dengan warning existing.
- `npm run build` PASS.
alasan teknis: Memberikan fleksibilitas memeriksa sebaran CCTV pada layar besar tanpa mengubah
tampilan peta di dalam halaman.

---

## [1.10.0] - 2026-08-03

### Marker CCTV Bertumpuk pada Peta Sebaran

perubahan yang dilakukan: **Marker dengan koordinat identik digeser agar tidak saling menutupi**
before:
- CCTV dengan koordinat latitude/longitude yang sama (contoh: THD-05 "Bundaran Sumbu Barat" dan
  THD-17 "Pintu Masuk Sumbu Barat" keduanya di -0.977067574, 116.7091789) dirender di titik yang
  persis sama, sehingga satu marker menutupi marker lainnya dan jumlah pin di peta lebih sedikit
  dari jumlah CCTV sebenarnya (20 aktif di info, tapi hanya 19 pin terlihat).
after:
- `CCTVMap` menghitung kemunculan tiap pasangan koordinat dan menggeser marker ke-2, ke-3, dst.
  secara deterministik (jarak 0.005° dengan sudut 137.5° per kemunculan) sehingga semua marker
  tetap terlihat meskipun datanya berbagi koordinat yang sama.
timestamp: 2026-08-03 WITA
kategori: Bugfix, Frontend, UI
file yang diubah:
- frontend/src/components/CCTVMap.tsx
- frontend/changelog.md
verifikasi:
- `npm run lint` PASS dengan warning existing.
- `npm run build` PASS.
- Verifikasi data: `SELECT status, COUNT(*) FROM cctvs GROUP BY status` → 20 ACTIVE, 15 INACTIVE
  (kartu info akurat; selisih hanya efek tumpukan marker).
alasan teknis: Memastikan visualisasi peta menampilkan seluruh CCTV tanpa kehilangan marker akibat
duplikasi koordinat pada data aset, tanpa mengubah data di database.

---

## [1.9.0] - 2026-08-03

### Pemisahan Peta Sebaran CCTV dari Dashboard

perubahan yang dilakukan: **Peta GIS sebaran CCTV dipindah dari Dashboard ke halaman & menu baru**
before:
- Halaman `/dashboard` merender `CCTVMap` (peta GIS sebaran CCTV) di antara kartu ringkasan dan
  analitik; navigasi sidebar hanya memiliki Dashboard, Aset CCTV, Pemeliharaan CCTV, dan menu admin.
after:
- Peta sebaran CCTV dihapus dari `/dashboard`; dashboard menjadi lebih fokus pada ringkasan,
  analitik, tugas, dan pemeliharaan.
- Halaman baru `/sebaran-cctv` dibuat: memuat data CCTV (`GET /cctv` limit 1000), menampilkan
  `CCTVMap` dengan tinggi peta 560px, kartu ringkasan status (Total/Aktif/Pemeliharaan), indikator
  kesehatan layanan, tombol muat ulang, dan refresh otomatis via event WebSocket `CCTV_UPDATED`.
- Menu baru "Peta Sebaran CCTV" (ikon `MapPinned`) ditambahkan di bawah "Dashboard" pada sidebar
  `DashboardLayout`, tersedia untuk semua peran.
timestamp: 2026-08-03 WITA
kategori: Feature, Frontend, UI
file yang diubah:
- frontend/src/app/sebaran-cctv/page.tsx (baru)
- frontend/src/app/dashboard/page.tsx
- frontend/src/components/DashboardLayout.tsx
- frontend/changelog.md
verifikasi:
- `npm run lint` PASS dengan warning existing.
- `npm run build` PASS.
alasan teknis: Memisahkan visualisasi geospasial dari halaman ringkasan agar Dashboard tetap ringan
dan fokus pada pemantauan data operasional, sekaligus memberikan akses cepat ke peta sebaran CCTV
melalui menu navigasi utama.

---

## [1.8.0] - 2026-08-03

### Navigasi Auth Atomik

perubahan yang dilakukan: **Login dan logout memakai full navigation untuk membatalkan request/state lama**
before:
- `router.push()` mempertahankan root layout dan request dashboard lama setelah logout.
- Request lama yang selesai dengan 401 dapat memicu redirect global setelah sesi baru berhasil login.
after:
- `window.location.replace()` digunakan setelah login/logout.
- AuthProvider, WebSocket, dan request halaman dimulai ulang secara bersih dengan cookie sesi terbaru.
timestamp: 2026-08-03 WITA
kategori: Bugfix, Auth, Frontend
file yang diubah:
- frontend/src/context/AuthContext.tsx
- frontend/changelog.md
verifikasi:
- `npm run lint` PASS dengan warning existing.
- `npm run build` PASS.
alasan teknis: Mencegah race condition request async lama yang menyebabkan login berhasil tetapi
langsung diarahkan kembali ke login.

---

## [1.7.0] - 2026-08-03

### Hentikan Reconnect WebSocket Setelah Logout

perubahan yang dilakukan: **Timer reconnect WebSocket sekarang mengikuti status autentikasi**
before:
- `onclose` selalu menjadwalkan koneksi ulang setiap 5 detik, termasuk setelah logout.
- Browser terus mengirim `/api/ws` dengan status 401 setelah sesi dihapus.
after:
- Reconnect hanya dijadwalkan ketika sesi masih aktif.
- Cleanup logout membatalkan timer dan menutup socket tanpa membuat koneksi baru.
timestamp: 2026-08-03 WITA
kategori: Bugfix, Auth, Realtime
file yang diubah:
- frontend/src/context/WebSocketContext.tsx
- frontend/changelog.md
verifikasi:
- `npm run lint` PASS dengan warning existing.
- `npm run build` PASS.
alasan teknis: Menghentikan request WebSocket berulang setelah logout agar tidak mengganggu
rate limiter dan tidak menimbulkan kesan sesi masih aktif.

---

## [1.6.0] - 2026-08-03

### Stabilitas Alur Logout

perubahan yang dilakukan: **Request logout tidak lagi memicu redirect 401 global**
before:
- `apiRequest('/auth/logout')` menggunakan penanganan 401 global yang tidak diperlukan untuk
  request logout dan dapat menjalankan redirect/event auth-expired tambahan.
after:
- Request logout memakai `skipAuthRedirect: true`; frontend tetap membersihkan state lokal dan
  mengarahkan ke login setelah endpoint logout idempotent dipanggil.
timestamp: 2026-08-03 WITA
kategori: Bugfix, Auth
file yang diubah:
- frontend/src/context/AuthContext.tsx
- frontend/changelog.md
verifikasi:
- `npm run lint` PASS dengan warning existing.
- `npm run build` PASS.
alasan teknis: Menjaga alur logout tetap deterministik ketika sesi server sudah kedaluwarsa atau
cookie lama tidak lagi valid.

---

## [1.5.0] - 2026-08-03

### Penghapusan Sisa Idle Logout

perubahan yang dilakukan: **Menghapus state banner inaktivitas dan mekanisme `sigap_logout_reason` yang sudah tidak digunakan**
before:
- Halaman login masih membaca `sigap_logout_reason` dan menampilkan pesan logout otomatis setelah 15 menit.
- `AuthContext.logout` masih menerima parameter alasan meskipun idle timer sudah dihapus.
after:
- State, effect, banner, ikon, dan localStorage key idle logout dihapus.
- `logout()` hanya menjalankan alur logout manual/server-side yang masih aktif.
timestamp: 2026-08-03 WITA
kategori: Bugfix, Refactor, Auth
file yang diubah:
- frontend/src/app/login/page.tsx
- frontend/src/context/AuthContext.tsx
- frontend/changelog.md
alasan teknis: Menghapus dead code dan mencegah UI menampilkan status sesi yang tidak lagi dapat dipicu.
verifikasi:
- `npm ci` berhasil memasang 370 package sesuai `package-lock.json`.
- `npm run lint` PASS dengan warning existing.
- `npm run build` PASS.
- `npm audit` melaporkan 4 vulnerability high yang belum diperbaiki.

---

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
