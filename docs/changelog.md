# Changelog — SIGAP (Milestone Monorepo)

> Catatan perubahan tingkat proyek (monorepo: `backend/`, `frontend/`, infra).
> Rincian per-bagian lihat `../backend/changelog.md` & `../frontend/changelog.md`.

## [1.28.8] — 2026-08-04 — Dashboard: Card Profil & Sidebar: Pemberitahuan

### Perubahan

- **Frontend** ([1.25.8]): card profil dipindah dari sidebar ke header Dashboard
  (di sebelah indikator "Layanan normal") — tanpa border, berbentuk badge profil
  (avatar inisial + nama + badge peran, klik → `/profile`); tombol "Muat ulang"
  dihapus. Menu "Pemberitahuan" di sidebar kini tombol dengan badge jumlah belum
  dibaca (sinkron via polling 60 dtk + WebSocket) yang membuka modal daftar
  notifikasi; item dapat diklik → modal informasi lengkap dengan tombol "Lihat
  detail" (tandai dibaca + navigasi); ada "Tandai semua dibaca" & "Lihat riwayat
  lengkap" → `/pemberitahuan`. Modal dirender via `createPortal`.
- `npm run lint` (0 error) & `npm run build` PASS.

Timestamp: 2026-08-04 WITA | Kategori: Sedang | Detail: `frontend/changelog.md` [1.25.8]

## [1.28.7] — 2026-08-04 — CCTV: Sortir & Filter dalam Modal Popup

### Perubahan

- **Frontend** ([1.25.7]): select filter status, indoor/outdoor, dan sortir di
  `/cctv` disembunyikan dari tampilan utama — digantikan tombol "Sortir" (badge
  jumlah filter aktif) yang membuka modal "Sortir & Filter" (5 kolom pengurutan +
  arah + filter status & lokasi, pola draft + Terapkan/Batal/Reset). Kotak pencarian
  tetap tampil. State `sortMode` diganti `sortBy`/`sortOrder`; backend tidak berubah.
- `npm run lint` (0 error) & `npm run build` PASS.

Timestamp: 2026-08-04 WITA | Kategori: Kecil | Detail: `frontend/changelog.md` [1.25.7]

## [1.28.6] — 2026-08-04 — Audit Log: Sortir & Filter dalam Modal + Posisi Menu Pemberitahuan

### Perubahan

- **Backend** ([0.26.2]): bugfix respons `GET /api/audit-logs` — `data` diselaraskan
  ke konvensi `{ rows, pagination }`; sebelumnya halaman audit-logs selalu kosong
  (frontend membaca `data.rows`, backend mengirim array).
- **Frontend** ([1.25.6]): filter modul & rentang tanggal di `/audit-logs` dipindah
  ke modal "Sortir & Filter" (kolom + arah pengurutan via `sort_by`/`order`,
  pola draft + Terapkan/Batal/Reset); kotak pencarian tetap tampil. Menu sidebar
  "Pemberitahuan" dipindah ke akhir menu utama — di bawah "Audit Log", tepat di atas
  tombol "Keluar".
- Test: `go test` & `go vet` PASS, `npm run lint` (0 error) & `npm run build` PASS.

Timestamp: 2026-08-04 WITA | Kategori: Kecil | Detail: `backend/changelog.md` [0.26.2], `frontend/changelog.md` [1.25.6]

## [1.28.5] — 2026-08-04 — Pengguna: Redesign Modal Detail

### Perubahan

- **Frontend** ([1.25.5]): modal detail pengguna di `/users` didesain ulang —
  header identitas (avatar inisial berwarna peran, nama + badge peran, username mono),
  panel rincian akun (email + tanggal dibuat, grid 2 kolom dengan pemisah),
  dan footer aksi cepat (Ubah data / Atur ulang kata sandi / Hapus pengguna —
  destruktif terpisah warna merah).
- Backend tidak berubah.
- `npm run lint` (0 error) & `npm run build` PASS.

Timestamp: 2026-08-04 WITA | Kategori: Kecil | Detail: `frontend/changelog.md` [1.25.5]

## [1.28.4] — 2026-08-04 — Pengecekan Lapangan: Sortir & Filter dalam Modal Popup

### Perubahan

- **Frontend** ([1.25.4]): filter hasil & rentang tanggal di `/field-maintenance`
  disembunyikan dari tampilan utama — digantikan tombol "Sortir" (dengan badge jumlah
  filter aktif) yang membuka modal "Sortir & Filter" (kolom + arah pengurutan via
  `sort_by`/`order`, filter lengkap, pola draft + Terapkan/Batal/Reset). Kotak pencarian
  teks tetap tampil. Pola seragam dengan `/maintenance` ([1.25.3]).
- Backend tidak berubah (endpoint `/field-maintenance` sudah mendukung `sort_by`/`order`).
- `npm run lint` (0 error) & `npm run build` PASS.

Timestamp: 2026-08-04 WITA | Kategori: Kecil | Detail: `frontend/changelog.md` [1.25.4]

## [1.28.3] — 2026-08-04 — Pemeliharaan: Sortir & Filter dalam Modal Popup

### Perubahan

- **Frontend** ([1.25.3]): grid filter (CCTV/Status/tanggal) di `/maintenance`
  disembunyikan dari tampilan utama — digantikan tombol "Sortir" (dengan badge
  jumlah filter aktif) yang membuka modal "Sortir & Filter"; modal berisi pengaturan
  pengurutan (kolom + arah, dikirim via `sort_by`/`order`) dan filter lengkap dengan
  pola draft + tombol Terapkan/Batal/Reset.
- Backend tidak berubah (endpoint `/maintenance-logs` sudah mendukung `sort_by`/`order`).
- `npm run lint` (0 error) & `npm run build` PASS.

Timestamp: 2026-08-04 WITA | Kategori: Kecil | Detail: `frontend/changelog.md` [1.25.3]

## [1.28.2] — 2026-08-04 — Bugfix: Pemberitahuan Tidak Tampil di Bell & Halaman `/pemberitahuan`

### Perubahan

- **Backend** ([0.26.1]): respons `GET /api/notifications` diselaraskan ke konvensi
  list SIGAP — `data: { rows, pagination }` (sebelumnya `data: [array]` + pagination
  top-level, pola usang `/audit-logs`).
- **Frontend** ([1.25.2]): `NotificationBell` & `/pemberitahuan` membaca `res.data?.rows`
  (sebelumnya salah baca `res.data?.data` → daftar selalu kosong meski notifikasi ada
  di database); pagination dibaca dari `res.data?.pagination`.
- Test: `go build/test` PASS, `npm run lint` (0 error) & `npm run build` PASS;
  container `api` + `frontend` di-rebuild, `/api/health` 200.

Timestamp: 2026-08-04 WITA | Kategori: Kecil (bugfix) | Detail: `backend/changelog.md` [0.26.1], `frontend/changelog.md` [1.25.2]

## [1.28.1] — 2026-08-04 — Pemberitahuan: Lonceng di Dashboard + Halaman `/pemberitahuan`

### Perubahan

- **Frontend** ([1.25.1]): lonceng notifikasi dipindah ke header halaman Dashboard
  (tidak lagi global); dropdown menampilkan 3 pemberitahuan teratas + tombol
  "Lihat lainnya"; menu sidebar baru "Pemberitahuan" menuju halaman `/pemberitahuan`
  (paginasi, filter belum dibaca, tandai semua, klik → navigasi, real-time).
- Scope pemberitahuan tetap per-user dari klaim JWT (backend) — teknisi tidak
  melihat pemberitahuan teknisi lain.
- `npm run lint` (0 error) & `npm run build` PASS.

Timestamp: 2026-08-04 WITA | Kategori: Sedang | Detail: `frontend/changelog.md` [1.25.1]

### Perubahan

- **Fase 5 (Jalur A) dimulai** — `docs/ROADMAP.md` & `docs/plan.md` diperbarui: status
  Fase 4 "⏸️ Sisa task ditunda (menjelang go-live, sistem masih pilot/UAT)", bagian baru
  Fase 5 dengan 5 task berurutan (5.1 notifikasi → 5.2 sinkron status CCTV → 5.3 re-assign →
  5.4 jadwal berulang → 5.5 eskalasi overdue) + keputusan desain (notifikasi in-app saja,
  sinkron status konservatif) + kandidat lanjutan Jalur B. Keputusan lengkap dicatat di
  `docs/plan.md` Bagian 6.
- **Backend** ([0.26.0]): model `Notification`, endpoint `/api/notifications`
  (list/unread-count/read/read-all, scope per-user), `WSHub` diperluas dengan
  `WSClientInfo{UserID, Role}` + `BroadcastToUser`/`BroadcastToRole` (global broadcast
  tetap), util `NotifyUser`/`NotifyRole` (async + retry), hook notifikasi pada
  assign/complete/field-check (attention), `swag init`.
- **Frontend** ([1.25.0]): `NotificationBell` (badge unread, dropdown, tandai semua dibaca,
  klik → navigasi, toast real-time `NOTIFICATION_CREATED`, polling 60 detik) di header
  mobile & sidebar desktop.
- Test: +13 kasus backend (handlers/notification_test.go, utils/notify_test.go);
  `go build/test/vet` PASS, `npm run lint && npm run build` PASS.

Timestamp: 2026-08-04 WITA | Kategori: Besar | Detail: `backend/changelog.md` [0.26.0], `frontend/changelog.md` [1.25.0]



### Perubahan

- **Frontend** (`maintenance/page.tsx`, `dashboard/page.tsx`, `DashboardAnalytics.tsx`, `TrendChart.tsx`):
  label tampilan status `failed` diubah dari "Gagal" menjadi "Perlu Tindak Lanjut" (badge tabel,
  filter dropdown, modal detail, kartu statistik, legenda analitik, tooltip grafik).
- **Backend** (`handlers/report.go`): `pdfStatusLabel` & ringkasan laporan PDF memakai
  "Perlu Tindak Lanjut".
- Nilai internal `failed` di database/API **tidak berubah** — tanpa migrasi data; filter,
  query parameter, dan seluruh klien API tetap kompatibel. CSV export tetap menulis nilai mentah.
- Test backend & lint/build frontend lulus; container `sigap-api` & `sigap-frontend` di-rebuild.

Timestamp: 2026-08-03 WITA | Kategori: Kecil | Detail: `backend/changelog.md` [0.25.2], `frontend/changelog.md` [1.24.2]

## [1.27.1] — 2026-08-03 — Bugfix Sesi: Cookie Duplikat (Stale di Path /api) Menyebabkan Login "Langasung Ter-logout"

### Penyebab

Setelah logout, browser yang masih menyimpan cookie `sigap_session`/`sigap_csrf` stale
(warisan versi lama di path `/api` atau `/api/`, token sudah direvoke) akan mengirimkan
dua cookie bernama sama pada setiap request. Sesuai RFC 6265 §5.4 cookie ber-Path
terpanjang dikirim lebih dulu, dan backend (`c.Cookies()`) selama ini mengambil nilai
PERTAMA — cookie stale — sehingga `GET /auth/me` (dan seluruh endpoint terlindungi)
menjawab 401 "Token telah dicabut" tepat setelah login sukses; frontend lalu mengarahkan
kembali ke halaman login (gejala: "login langsung ter-logout", tidak bisa login setelah logout).

### Perbaikan

- **Backend** (`utils/csrf.go`, `middleware/csrf.go`): nilai cookie sesi & CSRF diambil dari
  kemunculan TERAKHIR (sesi aktif di path "/" selalu menang) lewat helper baru
  `ExtractCookieValues`/`ExtractLastCookieValue`; berlaku untuk `Protected`, WebSocket, dan Logout.
- **Frontend** (`src/lib/api.ts`): `getCookie()` mengambil nilai terakhir agar header
  `X-CSRF-Token` selalu sinkron dengan cookie aktif.
- Cleanup cookie legacy saat login/logout tetap dipertahankan (defense in depth).
- Test baru untuk skenario cookie duplikat (utils & middleware) — seluruh suite lulus.

Timestamp: 2026-08-03 WITA | Kategori: Bugfix | Detail: `backend/changelog.md` [0.25.1], `frontend/changelog.md` [1.24.1]

## [1.27.0] — 2026-08-03 — Alur Penugasan Pemeliharaan (Assign Teknisi & Checklist 9 Tahap)

### Frontend

- Halaman Pemeliharaan CCTV: sub-menu status "Dalam Proses", "Selesai", "Gagal" pada
  input status dihapus — status kini tampil read-only ("Terjadwal" untuk catatan baru)
  dengan keterangan status diperbarui otomatis oleh sistem; field waktu pencatatan
  (`completed_at`) dihapus dari form, hanya "Waktu terjadwal".
- Log terjadwal dapat **ditugaskan ke teknisi** (admin): tombol ikon UserPlus pada baris
  status Terjadwal → modal pilih teknisi (diambil dari `GET /users?role=teknisi`) →
  `PUT /maintenance-logs/:id/assign`; setelah ditugaskan status otomatis "Dalam proses".
- Baris log berstatus "Dalam proses" menampilkan tombol **Selesaikan** (ikon CheckCheck)
  bagi teknisi yang ditugaskan (atau admin) → membuka wizard checklist 9 tahap dalam mode
  `complete` (`POST /maintenance-logs/:id/complete`) — checklist + foto bukti yang sama
  dengan pengecekan lapangan.
- Dashboard teknisi: tugas berstatus "Dalam proses" punya tombol **Selesaikan** yang
  membuka wizard yang sama; setelah selesai dashboard di-refresh (status jadi
  Selesai/Gagal otomatis).
- Teknisi tidak lagi melihat tombol "Catat Pemeliharaan"/"Ubah"; baris log menampilkan
  "Belum ditugaskan" saat belum ada teknisi.

### Backend

- Model `MaintenanceLog`: `TechnicianID` menjadi opsional (`*uuid.UUID`, relasi `User`
  pointer → `technician: null` saat belum ditugaskan), kolom baru `MaintenanceResultID`
  menautkan hasil pengecekan; `FieldMaintenance` mendapat `MaintenanceLogID`.
- `POST /maintenance-logs` hanya menerima status `scheduled` (tanpa teknisi,
  `completed_at` tidak lagi diterima); `PUT /:id` tidak lagi menerima `status`/`completed_at`
  — status dikelola sistem.
- Endpoint baru: `PUT /maintenance-logs/:id/assign` (admin; hanya log terjadwal; user
  tujuan wajib role teknisi; status → ongoing) dan `POST /maintenance-logs/:id/complete`
  (ownership check; hanya ongoing; checklist 1–30 item + 1–5 foto; transaksi; status
  `completed` bila 9/9 ok, selain itu `failed`; audit `ASSIGN_MAINTENANCE`/
  `COMPLETE_MAINTENANCE`).
- `GET /maintenance-logs` & `GET /maintenance-logs/:id` kini menerapkan scope peran
  (teknisi hanya melihat log miliknya) + ownership check detail; ekspor CSV & PDF
  nil-safe untuk log tanpa teknisi.
- `GET /users` mendukung filter `role`; `GET /dashboard/tasks` menambahkan
  `cctv_id`/`cctv_kode`; swagger di-regenerasi.

### Rincian

- Detail per bagian: `../frontend/changelog.md` versi [1.24.0],
  `../backend/changelog.md` versi [0.25.0].
- Uji: `go test ./...` PASS, `go vet ./...` PASS, `npm run lint` (0 errors, 46 warnings
  existing) & `npm run build` PASS.

---

## [1.26.0] — 2026-08-03 — Pengecekan Lapangan Teknisi via Scan QR

### Frontend

- Tombol **"Maintenance"** pada hasil scan QR CCTV (halaman Aset CCTV) membuka wizard
  pengecekan lapangan (`FieldMaintenanceModal`): 9 item checklist wajib diisi satu per
  satu (jaringan, listrik, pembersihan lensa, fisik kamera, kualitas gambar, perekaman,
  akses jarak jauh, kabel & konektor, lingkungan sekitar) dengan pilihan Normal/Bermasalah
  + catatan, lalu tahap akhir **unggah foto bukti (min. 1, maks. 5)** sebelum menyelesaikan.
- Halaman admin baru **`/field-maintenance`** (menu sidebar "Pengecekan Lapangan"):
  tabel hasil pengecekan (CCTV, teknisi, tanggal, ringkasan x/y normal, status
  Normal / Perlu Tindak Lanjut), filter pencarian/status/rentang tanggal, modal detail
  berisi seluruh checklist + foto bukti, dan hapus catatan. Dilindungi admin-only via proxy.

### Backend

- Model baru `FieldMaintenance` + `FieldMaintenanceCheck` + `FieldMaintenanceImage`
  (soft delete, relasi ke CCTV & User) dengan AutoMigrate.
- Endpoint baru `/api/field-maintenance` (POST create dengan transaksi + audit log,
  GET list terpaginasi — teknisi hanya miliknya sendiri, GET detail, DELETE admin).
- `CleanupOrphanUploads` kini juga menghitung foto `field_maintenance_images` sebagai
  referensi sah (tidak dihapus sebagai file yatim).
- Swagger regenerasi (`swag init`).

### Rincian

- Detail per bagian: `../frontend/changelog.md` versi [1.23.0],
  `../backend/changelog.md` versi [0.24.0].
- Uji: `go test ./...` PASS, `npm run lint` (0 errors) & `npm run build` PASS.

---

## [1.25.0] — 2026-08-03 — Notifikasi & Konfirmasi Kustom dengan Animasi

### Frontend

- Semua `alert()`/`confirm()` bawaan browser (24 titik di halaman users, maintenance,
  cctv) diganti sistem kustom:
  - **Toast** (`useToast()`): kartu notifikasi kanan atas dengan animasi check hijau
    (stroke-draw) untuk sukses, ikon merah untuk error, info biru; slide-in elastis,
    progress bar, auto-dismiss.
  - **Dialog konfirmasi** (`useConfirm()`): modal kustom dengan ikon peringatan
    bergoyang, fade/pop in-out, tombol Batal/Konfirmasi, tutup via X/Escape/klik latar.
- Toast sukses kini muncul di semua aksi berhasil (simpan, hapus, reset password,
  ekspor, cetak QR, impor, dsb.).

### Rincian

- Detail per bagian: `../frontend/changelog.md` versi [1.22.0].
- Tidak ada perubahan backend pada rilis ini.

---

### Backend

- PDF laporan pemeliharaan kini berkop surat resmi: logo IKN (dari
  `backend/assets/nusantara-logo.png`), nama instansi "Otorita Ibu Kota Nusantara",
  alamat lengkap, dan website, ditutup garis ganda.
- Perbaikan tata letak tabel: tinggi baris dihitung akurat dengan `SplitLines`,
  border grid penuh per sel (tabel simetris), dan pindah halaman per-baris agar
  tidak ada sel terpotong atau teks ketimpa.

### Rincian

- Detail per bagian: `../backend/changelog.md` versi [0.23.0].
- Tidak ada perubahan frontend pada rilis ini.

---

### Frontend

- Filter audit log dirombak agar input tidak terpotong: grid 3 kolom (bukan 4), baris
  pencarian full-width, lalu kolom "Modul", "Dari tanggal", dan "Sampai tanggal"
  masing-masing satu kolom penuh dengan label.

### Rincian

- Detail per bagian: `../frontend/changelog.md` versi [1.21.0].
- Tidak ada perubahan backend pada rilis ini.

---

### Frontend

- "Cetak Semua QR Code" di halaman Aset CCTV sebelumnya hanya mencetak data halaman aktif
  (10 CCTV). Kini semua perangkat diambil ulang dari backend (`page=1&limit=100`) sehingga
  seluruh QR code tercetak (total 35 perangkat).

### Rincian

- Detail per bagian: `../frontend/changelog.md` versi [1.20.0].
- Tidak ada perubahan backend pada rilis ini.

---

### Frontend

- Klik item menu "Keluar" di sidebar kini membuka dialog konfirmasi kustom ("Keluar dari
  SIGAP?" dengan ikon `LogOut` merah, tombol "Batal" dan "Keluar").
- Dialog dapat ditutup via tombol X, klik latar, atau `Escape`; logout baru dieksekusi
  setelah pengguna mengonfirmasi.

### Rincian

- Detail per bagian: `../frontend/changelog.md` versi [1.19.0].
- Tidak ada perubahan backend pada rilis ini.

---

### Frontend

- Card "Distribusi Status CCTV" dihapus dari dashboard.
- Card "Tugas Berjalan Semua Teknisi" (admin) / "Tugas Saya" (teknisi) naik dan kini
  dirender berdampingan dengan card "Tren Pemeliharaan" pada grid analitik;
  "Peringatan Pemeliharaan Terlambat" tetap full-width di bawahnya.

### Rincian

- Detail per bagian: `../frontend/changelog.md` versi [1.18.0].
- Tidak ada perubahan backend pada rilis ini.

---

### Frontend

- Tombol logout di header mobile (dekat tombol hamburger) dihapus; logout kini hanya
  melalui item menu "Keluar" di dalam sidebar.
- Transisi drawer sidebar diperhalus: `duration-500 ease-in-out` untuk slide, dan overlay
  selalu ter-mount dengan fade opacity dua arah agar animasi buka dan tutup sama-sama mulus.

### Rincian

- Detail per bagian: `../frontend/changelog.md` versi [1.17.0].
- Tidak ada perubahan backend pada rilis ini.

---

### Frontend

- Tombol logout dipindah dari bagian bawah sidebar ke dalam daftar navigasi, tepat di bawah
  menu "Audit Log" (admin) / "Pemeliharaan CCTV" (teknisi), dengan gaya yang sama persis
  dengan item menu lainnya (ikon `LogOut`, label "Keluar").

### Rincian

- Detail per bagian: `../frontend/changelog.md` versi [1.16.0].
- Tidak ada perubahan backend pada rilis ini.

---

### Frontend

- **Tombol keluar di layar kecil**: header mobile (di samping toggle tema) kini menampilkan
  tombol logout berikon `LogOut`; sidebar juga dapat digulir (`flex-1 min-h-0 overflow-y-auto`)
  sehingga aksi logout selalu terjangkau pada perangkat pendek.
- **Tombol laporan jadi ikon**: "Laporan PDF" (`FileText` merah) dan "Ekspor CSV"
  (`FileSpreadsheet` hijau) berubah menjadi kotak ikon-only dengan tooltip, konsisten di
  halaman Pemeliharaan dan Audit Log.
- **Payload QR dibatasi admin**: toggle "Masukkan payload" dan view input manual hanya
  dirender untuk admin; teknisi hanya melihat pemindai kamera.
- **"Tugas Saya" teknisi dirampingkan**: hanya nama CCTV + badge jadwal + badge status +
  badge terlambat; lokasi & tindakan (`action_taken`) tetap tampil untuk admin.

### Rincian

- Detail per bagian: `../frontend/changelog.md` versi [1.15.0].
- Tidak ada perubahan backend pada rilis ini.

---

### Backend Database

- **Kolom legacy `asset_id` dihapus**: migrasi `MigrateMaintenanceToCCTV` kini menghapus kolom
  `asset_id uuid NOT NULL` (beserta FK `fk_maintenance_logs_asset`) yang tersisa dari model
  `assets` lama. Sebelumnya INSERT log baru selalu gagal dengan SQLSTATE 23502 → HTTP 500
  ("Gagal menyimpan log pemeliharaan").
- **Penghapusan tidak lagi terlewat**: drop kolom dieksekusi di luar early-return sehingga
  selalu berjalan selama kolom masih ada; migrasi tetap idempotent.

before:
- `maintenance_logs.asset_id` masih NOT NULL tapi tidak pernah diisi oleh model `cctv_id` baru.
after:
- Tabel hanya bergantung pada `cctv_id`; penyimpanan log pemeliharaan berjalan normal.
timestamp: 2026-08-03 WITA
kategori: Bugfix, Database, Migration
file terkait:
- backend/config/migrate.go
- backend/changelog.md
- docs/changelog.md
verifikasi: `go test ./... -count=1`, `go vet ./...`, log startup Docker, dan `\d maintenance_logs`.
alasan perubahan: Memulihkan fungsi simpan log pemeliharaan yang rusak oleh sisa skema era model
`assets` (Fase 3.6).

---

## [1.15.0] — 2026-08-03 — Picker Kalender untuk Waktu Pencatatan Pemeliharaan

### Frontend UI

- **Input `datetime-local`**: field "Waktu pencatatan"/"Waktu terjadwal" pada form Catat
  Pemeliharaan Baru kini memakai picker native browser — kalender untuk memilih hari dan kontrol
  jam/menit — menggantikan input teks bebas.
- **Default waktu lokal**: nilai awal memakai waktu browser (bukan UTC), sebelumnya
  `new Date().toISOString()` bisa meleset beberapa jam dari waktu Indonesia.
- **Format tetap konsisten**: nilai dikonversi dua arah ke `YYYY-MM-DD HH:MM:SS` agar payload
  ke backend tidak berubah.

before:
- Input teks tanpa kalender; default waktu salah zona (UTC).
after:
- Kalender + picker waktu; default = waktu saat ini di zona lokal browser.
timestamp: 2026-08-03 WITA
kategori: Feature, Bugfix, Frontend, UI
file terkait:
- frontend/src/app/maintenance/page.tsx
- frontend/changelog.md
- docs/changelog.md
verifikasi: `npm run lint` dan `npm run build`.
alasan perubahan: Menghilangkan kesalahan format manual dan memastikan waktu tercatat sesuai zona
waktu pengguna.

---

## [1.14.0] — 2026-08-03 — Tren Pemeliharaan Jadi Spline Area Chart

### Frontend UI

- **Grafik area baru**: card "Tren Pemeliharaan" di dashboard mengganti bar chart CSS dengan
  Spline Area Chart SVG murni (`TrendChart`) — garis melengkung halus (Catmull-Rom → Bézier),
  area di bawah garis berisi gradasi biru, responsif, dengan tooltip hover rincian per status.
- **Tanpa dependency baru**: implementasi SVG mandiri (grid, sumbu, titik data, penanda hover,
  tooltip) menjaga ukuran bundle dan permukaan serangan tetap minimal.

before:
- Tren pemeliharaan divisualkan sebagai bar chart bertumpuk per status yang kurang terbaca untuk
  melihat arah pergerakan total pekerjaan.
after:
- Total pekerjaan per bulan tampil sebagai garis halus dengan area gradasi biru; rincian status
  tetap tersedia lewat tooltip saat kursor diarahkan ke titik data.
timestamp: 2026-08-03 WITA
kategori: Feature, Frontend, UI
file terkait:
- frontend/src/components/TrendChart.tsx (baru)
- frontend/src/components/DashboardAnalytics.tsx
- frontend/changelog.md
- docs/changelog.md
verifikasi: `npm run lint` dan `npm run build`.
alasan perubahan: Grafik area dengan garis halus lebih intuitif untuk memantau tren pemeliharaan
antar bulan, tanpa menambah ketergantungan pustaka chart eksternal.

---

## [1.13.0] — 2026-08-03 — Rincian Ringkasan CCTV di Bawah Kartu & Warna Bar Status

### Frontend UI

- **Respon kartu ringkasan dipindah ke atas**: saat kartu Total CCTV / Aktif / Nonaktif diklik,
  rincian (`cctv-summary-details`) kini tampil langsung di bawah kartu tersebut — bukan di bagian
  bawah halaman setelah analitik, tugas, dan pemeliharaan.
- **Bar progress lebih cerah**: "Distribusi Status CCTV" kini memakai hijau terang
  (`bg-emerald-500`) untuk aktif, merah terang (`bg-red-500`) untuk nonaktif/rusak, dan kuning
  cerah (`bg-amber-500`) untuk pemeliharaan, selaras dengan warna marker di peta sebaran.

before:
- Rincian ringkasan tampil jauh dari kartu yang diklik (di bawah semua section lain).
- Bar distribusi status memakai warna tema yang redup sehingga proporsinya kurang terbaca.
after:
- Rincian tampil tepat di bawah kartu ringkasan sebelum section analitik.
- Proporsi status langsung terbaca dengan warna terang.
timestamp: 2026-08-03 WITA
kategori: Feature, Frontend, UI
file terkait:
- frontend/src/app/dashboard/page.tsx
- frontend/src/components/DashboardAnalytics.tsx
- frontend/changelog.md
- docs/changelog.md
verifikasi: `npm run lint` dan `npm run build`.
alasan perubahan: Memperpendek jarak umpan balik interaksi ringkasan CCTV dan meningkatkan
keterbacaan chart distribusi status.

---

## [1.12.0] — 2026-08-03 — Mode Layar Penuh Peta Sebaran CCTV

### Frontend UI

- **Tombol fullscreen di peta**: `CCTVMap` kini memiliki tombol di pojok kanan atas untuk
  menampilkan peta dalam layar penuh (Fullscreen API), termasuk ikon status Maximize/Minimize,
  dukungan keluar via tombol maupun tombol Esc, serta pembersihan border/radius saat fullscreen.
- **Pembersihan kecil**: import ikon `Camera` yang tidak terpakai dihapus.

before:
- Peta sebaran CCTV terkurung dalam kartu halaman tanpa opsi tampilan penuh.
after:
- Peta dapat diperbesar ke layar penuh dan kembali ke ukuran semula dengan satu klik.
timestamp: 2026-08-03 WITA
kategori: Feature, Frontend, UI
file terkait:
- frontend/src/components/CCTVMap.tsx
- frontend/changelog.md
- docs/changelog.md
verifikasi: `npm run lint` dan `npm run build`.
alasan perubahan: Memudahkan pengguna memeriksa sebaran CCTV secara menyeluruh pada layar besar.

---

## [1.11.0] — 2026-08-03 — Marker CCTV Bertumpuk di Peta Sebaran

### Frontend UI

- **Koordinat duplikat diurai di peta**: `CCTVMap` mendeteksi CCTV yang berbagi pasangan
  latitude/longitude identik (mis. THD-05 "Bundaran Sumbu Barat" & THD-17 "Pintu Masuk Sumbu
  Barat" di titik yang sama) dan menggeser marker berikutnya secara deterministik agar semua
  CCTV tetap tampil dan tidak saling menutupi.
- **Kartu info sudah akurat**: total 35 CCTV (20 ACTIVE, 15 INACTIVE) sesuai database; selisih
  yang diamati pengguna (19 vs 20 pin aktif) murni akibat satu pin tertutup pin lain.

before:
- Marker dengan koordinat sama dirender tepat di atas satu sama lain; pin yang terhitung manual
  tidak cocok dengan jumlah di kartu ringkasan.
after:
- Seluruh marker (35 CCTV) dapat dibedakan di peta; popup masing-masing tetap bisa dibuka.
timestamp: 2026-08-03 WITA
kategori: Bugfix, Frontend, UI
file terkait:
- frontend/src/components/CCTVMap.tsx
- frontend/changelog.md
- docs/changelog.md
verifikasi: `npm run lint`, `npm run build`, dan cek jumlah status di PostgreSQL.
alasan perubahan: Menjaga visualisasi peta tetap sesuai jumlah aset sebenarnya tanpa mengubah
data koordinat di database.

---

## [1.10.0] — 2026-08-03 — Halaman & Menu Peta Sebaran CCTV

### Frontend UI

- **Peta dipindah dari Dashboard**: komponen `CCTVMap` tidak lagi dirender di `/dashboard`;
  dashboard kini fokus pada kartu ringkasan, analitik, tugas, dan pemeliharaan.
- **Halaman baru `/sebaran-cctv`**: menampilkan peta GIS sebaran CCTV (tinggi 560px), kartu
  ringkasan status (Total CCTV, Aktif, Pemeliharaan), indikator kesehatan layanan, tombol muat
  ulang, dan pembaruan otomatis saat event WebSocket `CCTV_UPDATED`.
- **Menu baru di sidebar**: "Peta Sebaran CCTV" (ikon `MapPinned`) ditambahkan tepat di bawah
  "Dashboard" pada `DashboardLayout`, tersedia untuk semua peran.

before:
- Peta sebaran CCTV hanya tampil di dalam halaman `/dashboard` tanpa navigasi langsung dari menu.
after:
- Peta sebaran CCTV dapat diakses dari menu navigasi utama dan dashboard tidak lagi memuat peta.
timestamp: 2026-08-03 WITA
kategori: Feature, Frontend, UI
file terkait:
- frontend/src/app/sebaran-cctv/page.tsx (baru)
- frontend/src/app/dashboard/page.tsx
- frontend/src/components/DashboardLayout.tsx
- frontend/changelog.md
- docs/AGENTS.md
- docs/changelog.md
verifikasi: `npm run lint` dan `npm run build`.
alasan perubahan: Memisahkan visualisasi geospasial dari halaman ringkasan operasional agar
dashboard tetap fokus pada data pemantauan, dan memudahkan pengguna membuka peta sebaran CCTV
langsung dari menu utama.

---

## [1.9.0] — 2026-08-03 — Navigasi Auth Atomik Setelah Logout/Login

### Frontend Auth Race

- **Full navigation setelah login/logout**: mengganti transisi SPA `router.push()` dengan
  `window.location.replace()` agar request dashboard lama, state AuthProvider, dan WebSocket
  lama benar-benar dibatalkan sebelum sesi baru dimulai.
- **Pencegahan stale 401**: response 401 dari request halaman sebelum logout tidak lagi dapat
  memanggil handler global dan mencabut sesi yang baru saja dibuat.

before:
- SPA navigation mempertahankan root layout dan request async halaman lama setelah logout.
- Setelah login cepat, request lama dapat selesai dengan 401 bersamaan dengan request sesi baru
  yang 200, lalu redirect global membuat pengguna langsung keluar lagi.
after:
- Logout/login melakukan navigasi penuh setelah cookie server selesai diproses.
- AuthProvider dan seluruh request halaman dimulai ulang dengan sesi yang sama.
timestamp: 2026-08-03 WITA
kategori: Bugfix, Auth, Frontend
file terkait:
- frontend/src/context/AuthContext.tsx
- frontend/changelog.md
- docs/changelog.md
verifikasi: `npm run lint`, `npm run build`, dan observasi log Docker pada alur login/logout.
alasan perubahan: Menghilangkan race condition antara request async lama dan sesi baru saat
pengguna melakukan login kembali segera setelah logout.

---

## [1.8.0] — 2026-08-03 — Penghapusan Cookie Legacy dan Unifikasi Token Sesi

### Auth, Cookie, JWT, dan WebSocket

- **Cookie legacy lintas path dibersihkan**: response login/logout mengirim `Set-Cookie` terpisah untuk path `/`, `/api`, dan `/api/`, sehingga browser tidak memilih cookie sesi stale pada request API.
- **JWT dibuat selalu unik**: klaim `jti` UUID dan `iat` ditambahkan agar login ulang cepat tidak menghasilkan token identik dengan token yang baru direvoke.
- **Reconnect WebSocket dihentikan saat logout**: timer reconnect memeriksa status sesi dan tidak lagi membanjiri `/api/ws` dengan 401 setelah logout.
- **Testing diperluas**: validasi multi-path cookie dan keunikan token ditambahkan.

before:
- Header `Set-Cookie` Fiber menimpa cookie dengan nama sama sehingga hanya path terakhir yang terkirim.
- JWT tanpa `jti` dapat identik jika login ulang terjadi pada detik yang sama dengan token lama.
- WebSocket tetap reconnect setiap 5 detik meskipun sesi sudah tidak aktif.
after:
- Cookie root dan cookie legacy API dihapus bersamaan saat login/logout.
- Token baru selalu berbeda dan tidak ikut blacklist token lama.
- WebSocket berhenti reconnect ketika `isAuthenticated=false`.
timestamp: 2026-08-03 WITA
kategori: Bugfix, Auth, Security, Infrastructure, Testing
file terkait:
- backend/utils/session.go
- backend/utils/jwt.go
- backend/utils/jwt_test.go
- backend/handlers/auth.go
- backend/handlers/auth_test.go
- backend/handlers/auth_extra_test.go
- backend/middleware/csrf.go
- backend/routes/auth.go
- docs/AGENTS.md
- frontend/src/context/WebSocketContext.tsx
- docs/TESTING.md
- docs/changelog.md
verifikasi: `go test ./... -count=1`, `go vet ./...`, `npm run lint`, `npm run build`, dan smoke test cookie melalui `https://localhost`.
alasan perubahan: Menghilangkan token/cookie stale yang menyebabkan login berhasil tetapi request pertama langsung 401, serta menghentikan reconnect WebSocket pasca logout.

---

## [1.7.0] — 2026-08-03 — Pemulihan Logout/Login dan Proxy Runtime

### Auth, CSRF, Rate Limit, dan Konfigurasi Docker

- **Logout idempotent**: endpoint logout selalu menghapus cookie sesi/CSRF; token valid tetap direvoke, sedangkan token kosong/invalid dianggap logout berhasil.
- **Logout tidak diblokir CSRF stale**: middleware CSRF melewati endpoint logout karena logout tidak mengubah resource dan harus tetap bisa membersihkan sesi yang rusak.
- **Konfigurasi runtime Docker dilengkapi**: `APP_ENV=production`, `CORS_ALLOWED_ORIGINS=https://localhost`, dan `TRUSTED_PROXY_IPS=172.28.0.10` ditambahkan ke environment runtime/template.
- **Frontend logout tidak memicu redirect 401 global**: request logout memakai `skipAuthRedirect` dan tetap mengarahkan pengguna secara lokal setelah server merespons atau gagal.

before:
- Logout dapat berhenti di status 403 CSRF ketika cookie CSRF stale/hilang, tetapi frontend tetap mengarahkan pengguna ke login sehingga cookie httpOnly lama tertinggal.
- Login dapat merespons 200 lalu `/auth/me` menerima 401 karena sesi lama/tidak terkirim; percobaan berulang kemudian terkena 429.
- Container runtime belum memuat trusted proxy, sehingga semua request dihitung dari IP Nginx yang sama.
after:
- Logout dapat dipanggil ulang dalam kondisi apa pun, cookie selalu di-expire, dan sesi valid direvoke.
- Rate limiter membaca client IP melalui trusted proxy Nginx dan origin Docker menggunakan `https://localhost`.
- Test regresi logout/CSRF/route ditambahkan.
timestamp: 2026-08-03 WITA
kategori: Bugfix, Auth, Security, Infrastructure
file terkait:
- backend/handlers/auth.go
- backend/middleware/csrf.go
- backend/routes/auth.go
- backend/handlers/auth_extra_test.go
- backend/middleware/csrf_test.go
- backend/routes/routes_test.go
- backend/.env.docker (runtime lokal, tidak di-commit)
- backend/.env.docker.example
- frontend/src/context/AuthContext.tsx
- nginx/nginx.conf
- docker-compose.yml
- docs/changelog.md
verifikasi: `go test ./... -count=1`, `go vet ./...`, `npm run lint`, `npm run build`, `docker compose config --quiet`, dan uji runtime login -> logout -> login ulang.
alasan perubahan: Menghentikan loop logout/login yang disebabkan kombinasi CSRF stale, cookie sesi tertinggal, dan rate limiter yang salah mengidentifikasi client di belakang Nginx.

---

## [1.6.0] — 2026-08-03 — Sinkronisasi Single-Origin, Swagger Nginx, dan Dokumentasi

### Dokumentasi, Infrastruktur, dan Stabilitas Sesi

- **Dokumentasi akses single-origin**: menambahkan `docs/ACCESS.md` yang menetapkan
  `https://localhost` sebagai satu-satunya entrypoint Docker dan menjelaskan alur Nginx ke
  frontend, API, WebSocket, dan Swagger.
- **Sinkronisasi dokumentasi autentikasi**: memperbarui masa berlaku JWT menjadi 5 jam,
  menghapus referensi sliding session dan idle timer yang sudah tidak digunakan, serta
  mendokumentasikan `MaxAge: 18000` pada cookie sesi/CSRF.
- **Matriks test dikoreksi**: mencantumkan `utils/audit_test.go` dan memperbaiki jumlah aktual
  menjadi 28 file test.
- **Swagger melalui Nginx**: menambahkan proxy `/swagger/*` ke API dan mengubah metadata Swagger
  agar menggunakan host publik `localhost` dengan scheme HTTPS.
- **Dead code idle logout dihapus**: frontend tidak lagi menyimpan atau membaca
  `sigap_logout_reason` maupun menampilkan banner inaktivitas yang sudah tidak pernah terpicu.

before:
- Dokumen masih mencampur akses `localhost:3000`/`localhost:3001`, JWT 24 jam, sliding session,
  idle timer, dan URL Swagger yang tidak sesuai dengan routing Nginx.
- Swagger tidak dapat diakses melalui reverse proxy karena hanya `/api/*` yang diteruskan ke API.
- Matriks test menghilangkan `audit_test.go` dan menyatakan jumlah yang tidak sesuai.
after:
- Deployment Docker menggunakan `https://localhost` sebagai single-origin; port service tetap
  internal dan route Swagger tersedia melalui `https://localhost/swagger/index.html` pada non-production.
- Dokumentasi sesi, keamanan, roadmap, README, dan testing sesuai kondisi kode aktual.
- Sisa mekanisme idle logout di frontend dihapus.
timestamp: 2026-08-03 WITA
kategori: Documentation, Infrastructure, Security, Auth, Testing
file terkait:
- docs/ACCESS.md
- docs/AGENTS.md
- docs/TESTING.md
- docs/SECURITY-AUDIT.md
- docs/ROADMAP.md
- docs/PRODUCT.md
- docs/plan.md
- docs/changelog.md
- README.md
- nginx/nginx.conf
- docker-compose.yml
- backend/.env.docker.example
- backend/main.go
- backend/docs/docs.go
- backend/docs/swagger.json
- backend/docs/swagger.yaml
- frontend/src/context/AuthContext.tsx
- frontend/src/app/login/page.tsx
alasan perubahan: Menjaga dokumentasi, routing publik, autentikasi cookie, dan catatan coverage
tetap selaras setelah migrasi ke single-origin Nginx serta penyederhanaan sesi.
verifikasi akhir:
- `go test ./... -count=1` PASS.
- `go vet ./...` PASS.
- `npm ci` berhasil memasang 370 package sesuai lockfile.
- `npm run lint` PASS dengan warning existing dan `npm run build` PASS.
- `docker compose config --quiet` PASS.
- `nginx -t` PASS pada image Nginx 1.27 setelah upstream Docker disediakan.
- `npm audit` melaporkan 4 vulnerability high yang belum diperbaiki.

---

## [1.5.0] — 2026-08-03 — Perbaikan Stabilitas Autentikasi & Single-Origin Proxy

### Autentikasi, Sesi, dan Infra

- **Pencegahan Phantom Duplicate Cookie pada Auth Client**: `clearAuthCookies()` di `frontend/src/lib/api.ts` tidak lagi mencoba menghapus cookie `sigap_session` (httpOnly) melalui JavaScript untuk mencegah browser membuat cookie hantu non-httpOnly yang menyebabkan kegagalan login ulang.
- **Opsi `skipAuthRedirect` pada API Request**: Menambahkan opsi `skipAuthRedirect: true` pada `apiRequest()` untuk rute bootstrap (`/auth/me`) dan login (`/auth/login`) agar respons 401 wajar tidak memicu pembersihan sesi global secara prematur.
- **Penetapan MaxAge Cookie Sesi**: Menyetel `MaxAge: 18000` (5 jam) pada cookie sesi `sigap_session` dan CSRF `sigap_csrf` di `backend/utils/session.go` agar selaras dengan durasi kedaluwarsa JWT (5 jam) sehingga sesi bertahan saat browser di-refresh.
- **Penegasan Single-Origin Nginx Proxy**: Menutup port publik `3001` pada service `frontend` di `docker-compose.yml` dan mewajibkan seluruh akses melalui Reverse Proxy Nginx (`https://localhost`). Hal ini menjamin frontend (`https://localhost/`) dan backend (`https://localhost/api/`) berada pada origin yang sama persis sehingga cookie httpOnly terkirim secara 100% konsisten tanpa masalah cross-origin port.

kategori: Auth, Bugfix, Security, Infrastructure
file terkait (root/infra): `docker-compose.yml`, `frontend/src/lib/api.ts`, `frontend/src/context/AuthContext.tsx`, `frontend/src/app/login/page.tsx`, `backend/utils/session.go`, `docs/changelog.md`
verifikasi: `docker compose up -d`, pengujian alur login -> logout -> login ulang dan refresh halaman pada `https://localhost`.
alasan perubahan: Menyelesaikan masalah sesi terputus saat refresh dan kegagalan login ulang akibat bentrokan cookie httpOnly serta perbedaan origin port.

---

## [1.4.1] — 2026-08-02 — Fase 4.1: Penyelesaian Coverage Test

### Testing Backend

- Test jalur sukses update/delete CCTV dan maintenance.
- Test handshake WebSocket dengan token valid dan bootstrap route terlindungi.
- Test resolusi legacy asset ke CCTV dan pembaruan matriks coverage.
- Sisa yang ditunda: import transaksi penuh, token version mismatch WebSocket, dan migration update penuh.

kategori: Testing, Quality, Migration
file terkait (root/infra): `docs/ROADMAP.md`, `docs/TESTING.md`, `docs/changelog.md`
verifikasi: `go test ./... -count=1`, `go vet ./...`, `golangci-lint v2.12.2`, `npm run lint`, `npm run build`.
alasan perubahan: Menyelaraskan dokumentasi dengan hasil implementasi Fase 4.1 sebelum melanjutkan ke versioned database migration.

---

## [1.4.0] — 2026-08-02 — Fase 4: Quick Wins Keamanan & Kualitas

### Keamanan, Pengujian, dan CI

- Proxy server-side Next.js untuk route `/audit-logs` dan `/users`, termasuk validasi sesi melalui backend dan pengecekan role admin.
- Audit log memiliki retry terbatas dan broadcast `AUDIT_LOG_ADDED` setelah penyimpanan berhasil.
- Unit test tambahan untuk edge case CCTV, maintenance, user, audit retry, dan UUID model hook.
- CI menjalankan `npm run lint` dan `golangci-lint` selain build serta unit test.

kategori: Security, Testing, CI, Quality
file terkait (root/infra): `docs/ROADMAP.md`, `docs/SECURITY-AUDIT.md`, `.github/workflows/ci.yml`
verifikasi: `go test ./... -count=1`, `go build ./...`, `npm run lint`, `npm run build`.
alasan perubahan: Menutup temuan keamanan tersisa dan menjadikan pemeriksaan kualitas otomatis pada setiap pull request.

---

## [1.3.0] — 2026-08-01 — Fase 3: Fitur (Kapabilitas)

### Fitur Kapabilitas (ROADMAP 3.1 – 3.9)

- **3.1** Self-service change password (`POST /api/auth/change-password`) + paksa ganti password saat login pertama (`must_change_password`).
- **3.2** Sliding session — masa berlaku JWT diperpanjang saat akses aktif; blacklist & revoke token lama tetap berjalan.
- **3.3** Dashboard analitik: `GET /api/dashboard/summary` (statistik, tren 6 bulan, distribusi status, overdue) + komponen `DashboardAnalytics`.
- **3.4** Notifikasi tugas teknisi: `GET /api/dashboard/tasks` (teknisi = tugas miliknya, admin = semua; `pending_count`/`overdue_count`).
- **3.5** Halaman profil (`/profile`) + `PUT /api/auth/profile`; kartu profil di sidebar jadi tautan.
- **3.6** Unifikasi model `assets` ↔ `cctv`: CCTV jadi satu-satunya sumber kebenaran. Model & handler/rute `assets` dihapus; `MaintenanceLog` kini `cctv_id` → `cctvs`; migrasi data legacy `asset_id` otomatis sekali saat startup (`config/migrate.go`).
- **3.7** Laporan PDF (`GET /api/maintenance-logs/report-pdf`, via `go-pdf/fpdf`), ekspor audit-log CSV (`GET /api/audit-logs/export`), dan filter rentang tanggal `start_date`/`end_date` pada list & ekspor.
- **3.8** Privasi & kebersihan upload: seluruh foto dinormalisasi ke JPEG (metadata EXIF/GPS terhapus); goroutine cleanup menghapus file yatim berumur >24 jam.
- **3.9** Pembatasan ekspor per role: teknisi hanya mengekspor/mengunduh laporan data miliknya; admin penuh.

kategori: Feature, Security, Dashboard, PDF, Upload
file terkait (root/infra): `ROADMAP.md`, `SECURITY-AUDIT.md`, `../README.md`, `changelog.md` (docs, baru)
verifikasi: `go build/vet/test` PASS (backend), `npm run build` PASS (frontend), Swagger regenerated.
alasan perubahan: Meningkatkan nilai operasional sistem dan menutup temuan keamanan 18 (ekspor per role).

---

## [1.2.0] — 2026-08-01 — Fase 2: Penguatan (Hardening)

### Hardening Keamanan (SECURITY-AUDIT temuan 7–15, 20)

- Migrasi token JWT ke cookie httpOnly + pola CSRF double-submit (`sigap_session`, `sigap_csrf`, header `X-CSRF-Token`).
- Revoke token lama saat reset password / role diubah (klaim `tv` / token version).
- Rate limiter & audit log hanya percaya header proxy dari IP terdaftar (`TRUSTED_PROXY_IPS`).
- Batas ukuran (4 MB) & baris (1000) import Excel.
- CORS tanpa wildcard; default aman `http://localhost:3001`.
- Docker hardening: quick tunnel dihapus, reverse proxy Nginx + TLS, port DB/API/frontend tidak dipublish, secret via root `.env`.
- CSP & HSTS aktif di produksi; kebijakan password min 8 + kompleksitas; ownership check maintenance log; penanganan 401 pada fetch raw frontend.

kategori: Security, Hardening, Docker, Infra
file terkait (root/infra): `docker-compose.yml`, `nginx/` (baru), `.env.example` (baru, root), `.gitignore`
verifikasi: `go build/vet/test` PASS (backend), `npm run build` PASS (frontend).
alasan perubahan: Menutup temuan Fase 2 (SECURITY-AUDIT.md) untuk mencegah pencurian token, CSRF, spoofing IP, overload impor, eksposur layanan, dan password lemah.

---

## [1.1.0] — 2026-08-01 — Fase 1: Kritis (Prioritas Tertinggi)

### Perbaikan Kritis (SECURITY-AUDIT temuan 1–6)

- JWT: hapus fallback secret hardcoded (wajib env `JWT_SECRET`).
- Seeder: admin default tidak lagi di-reset tiap startup; password wajib dari env `ADMIN_DEFAULT_PASSWORD`.
- WebSocket: wajib autentikasi (handshake token).
- Stored XSS pada cetak label QR diperbaiki (render React, bukan `document.write`).
- QR Code dipindah ke server-side (`skip2/go-qrcode`), lepas dari API pihak ketiga.
- Pagination dibatasi cap maksimum (`MAX_PAGE_SIZE`).

kategori: Security, Kritis
file terkait (root/infra): `README.md`, `docker-compose.yml`
verifikasi: `go build/vet/test` PASS (backend), `npm run build` PASS (frontend).
alasan perubahan: Menutup celah keamanan paling berbahaya sebelum pengembangan lanjutan.

---

## [1.0.0] — 2026-08-01 — Rilis Awal

- Sistem identifikasi & inventaris CCTV (SIGAP) berbasis Go Fiber + Next.js + PostgreSQL.
- Fitur inti: manajemen master CCTV, QR Code terenkripsi AES-256, scanner QR, import Excel massal, log pemeliharaan dengan unggah foto, manajemen pengguna & RBAC (Admin/Teknisi), autentikasi JWT, Swagger UI.
- Deployment Docker Compose (db, api, frontend).
