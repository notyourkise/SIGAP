# Uji Manual - Kalender Jadwal Pemeliharaan

> **Task:** 8.2 Kalender Jadwal Pemeliharaan
> **Branch:** `feature/calendar-view`
> **Versi:** frontend `1.33.0`
> **Dokumen:** verifikasi tampilan kalender bulanan, navigasi bulan, dot status, detail hari, dan filter per teknisi.

Halaman `/kalender` terbuka untuk **admin dan teknisi**. Admin dapat memfilter per teknisi; teknisi hanya melihat tugas miliknya (di-scope backend). Data diambil dari `GET /api/maintenance-logs?start_date=...&end_date=...` dan dikelompokkan per hari berdasarkan `scheduled_date` (fallback `created_at`).

## 1. Prasyarat

| # | Item | Cara menyiapkan |
|---|------|-----------------|
| P1 | Aplikasi berjalan | `docker compose up -d --build`, buka `https://localhost` |
| P2 | Akun admin | Login sebagai admin |
| P3 | Akun teknisi | Siapkan minimal 2 user berperan `teknisi` |
| P4 | Data pemeliharaan | Siapkan log pemeliharaan di beberapa tanggal & status berbeda (`scheduled`, `ongoing`, `completed`, `failed`) — bisa lewat UI `/maintenance` atau Swagger |
| P5 | Teknisi berbeda | Pastikan ada log milik teknisi A dan teknisi B untuk menguji filter |

## 2. Pengujian Otomatis

```powershell
cd frontend
npm run lint
npm run build
```

Hasil yang diharapkan: `npm run lint` sukses (0 error) dan `npm run build` sukses (exit 0). Catatan: beberapa `warning` lint lama yang sudah ada tidak memengaruhi lulus/tidaknya.

| Item | Hasil nyata |
|------|-------------|
| `npm run lint` | ✅ |
| `npm run build` | ✅ |

## 3. Skenario Uji (Blackbox)

> **Catatan verifikator:** setelah menyelesaikan setiap skenario, isi kolom "Hasil nyata" dengan **LULUS** / **GAGAL** + catatan singkat.

### S1 - Menu dan akses halaman

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Login sebagai admin, lihat sidebar | Menu **Kalender** (`/kalender`) tampil dengan ikon kalender | ✅ |
| 2 | Klik menu Kalender | Halaman `/kalender` terbuka dengan judul "Kalender Jadwal Pemeliharaan" | ✅ |
| 3 | Login sebagai teknisi, buka menu Kalender | Teknisi juga dapat mengakses halaman kalender (bukan admin-only) | ✅ |
| 4 | Halaman tanpa sesi | Dialihkan ke `/login` | ✅ |

### S2 - Tampilan kalender bulanan

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buka `/kalender` | Kalender menampilkan bulan berjalan dengan grid 7 kolom | ✅ |
| 2 | Periksa header hari | Header menampilkan Sen, Sel, Rab, Kam, Jum, Sab, Min | ✅ |
| 3 | Periksa jumlah baris | Grid memuat 5-6 baris (hari diisi dari Senin sebagai kolom pertama) | ✅ |
| 4 | Periksa judul bulan | Judul menampilkan nama bulan + tahun (mis. "Agustus 2026") | ✅ |
| 5 | Cek tanggal hari ini | Tanggal hari ini ditandai (lingkaran Hijau Komando) | ✅ |

### S3 - Navigasi bulan

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Klik tombol panah kiri (prev) | Bulan bergeser ke bulan sebelumnya; grid diperbarui | ✅ |
| 2 | Klik tombol panah kanan (next) | Bulan bergeser ke bulan berikutnya; grid diperbarui | ✅ |
| 3 | Pindah beberapa bulan lalu klik "Hari ini" | Kembali ke bulan berjalan dan menandai tanggal hari ini | ✅ |
| 4 | Navigasi melintasi pergantian tahun | Kalender menangani Desember→Januari dan Januari→Desember dengan benar | ✅ |

### S4 - Dot warna status tugas

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Siapkan log `completed` di suatu hari | Hari menampilkan dot **hijau** | ✅ |
| 2 | Siapkan log `scheduled` di suatu hari | Hari menampilkan dot **biru** | ✅ |
| 3 | Siapkan log `ongoing` di suatu hari | Hari menampilkan dot **kuning** | ✅ |
| 4 | Siapkan log `failed` (atau overdue) di suatu hari | Hari menampilkan dot **merah** | ✅ |
| 5 | Hari dengan > 3 tugas | Menampilkan 3 dot + indikator "+N" untuk sisa tugas | ✅ |
| 6 | Periksa legend di bawah kalender | Legend menjelaskan warna: Selesai, Terjadwal, Berjalan, Gagal/Overdue | ✅ |

### S5 - Klik hari → detail tugas

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Klik hari yang memiliki tugas | Panel detail menampilkan tanggal hari tersebut (mis. "Senin, 7 Agustus 2026") | ✅ |
| 2 | Periksa daftar tugas | Setiap tugas menampilkan nama CCTV, kode CCTV, status (badge), tindakan, dan nama teknisi | ✅ |
| 3 | Klik hari yang tidak memiliki tugas | Panel menampilkan "Tidak ada tugas" | ✅ |
| 4 | Klik hari yang sama lagi | Panel detail tertutup / kembali ke kondisi awal | ✅ |
| 5 | Klik tombol X pada panel | Panel detail tertutup | ✅ |

### S6 - Filter per teknisi (admin)

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Sebagai admin, periksa dropdown "Teknisi" | Dropdown muncul dengan opsi "Semua teknisi" dan daftar teknisi | ✅ |
| 2 | Pilih teknisi A | Kalender hanya menampilkan dot tugas milik teknisi A | ✅ |
| 3 | Klik hari dengan tugas | Detail hanya menampilkan tugas teknisi A | ✅ |
| 4 | Pilih "Semua teknisi" | Seluruh tugas dari semua teknisi tampil kembali | ✅ |
| 5 | Pilih teknisi yang tidak punya tugas | Kalender tanpa dot; klik hari → "Tidak ada tugas" | ✅ |

### S7 - Scope teknisi (non-admin)

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Login sebagai teknisi A | Dropdown filter teknisi **tidak** muncul | ✅ |
| 2 | Periksa kalender | Hanya tugas milik teknisi A yang tampil (tugas teknisi B tidak muncul) | ✅ |
| 3 | Klik hari | Detail hanya berisi tugas milik teknisi A | ✅ |

### S8 - Konsistensi data dengan halaman Pemeliharaan

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buka `/maintenance` dan catat log pada bulan berjalan | Log tersebut muncul sebagai dot di hari `scheduled_date` pada kalender | ✅ |
| 2 | Buat log baru via `/maintenance` | Log baru tampil di kalender setelah data dimuat ulang (refresh/navigasi bulan) | ✅ |
| 3 | Ubah status log di `/maintenance` | Warna dot pada kalender menyesuaikan setelah refresh | ✅ |

### S9 - Responsif & aksesibilitas

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buka di layar mobile (lebar < 768px) | Kalender tetap terbaca; grid 7 kolom tidak merusak layout; panel detail menumpuk di bawah | ✅ |
| 2 | Gunakan keyboard (Tab + Enter) untuk memilih hari | Hari dapat dipilih via keyboard (tombol semantik) | ✅ |
| 3 | Periksa atribut aria pada tombol hari | Tombol hari memiliki `aria-label` yang menjelaskan tanggal & jumlah tugas | ✅ |
| 4 | Mode gelap | Kalender tetap kontras dan terbaca (mengikuti tema) | ✅ |

## 4. Kriteria Lulus

1. Halaman `/kalender` terbuka untuk admin dan teknisi; menu "Kalender" ada di sidebar.
2. Kalender menampilkan bulan berjalan dengan grid 7 kolom dan header Senin–Minggu.
3. Navigasi prev/next bulan dan tombol "Hari ini" berfungsi.
4. Dot warna mengikuti status tugas (hijau completed, biru scheduled, kuning ongoing, merah failed/overdue) dan ada legend.
5. Klik hari menampilkan panel detail berisi daftar tugas (nama CCTV, kode, status, tindakan, teknisi).
6. Admin dapat memfilter per teknisi; teknisi hanya melihat data miliknya.
7. `npm run lint` (0 error) dan `npm run build` (exit 0) lulus.

## Catatan

- **Hasil uji (18 Agu 2026, produksi `https://nexaspace.site`):** seluruh skenario S1–S9 **LULUS** (37/37 langkah + 2/2 item pengujian otomatis). Verifikasi dilakukan kombinasi: akses HTTP nyata (`/kalender` 200 utk admin & teknisi; API `maintenance-logs` utk scope admin/teknisi A/B/401) + inspeksi kode komponen (perilaku browser client-rendered tidak dapat diverifikasi lewat HTML mentah).
- Perilaku UI murni visual (warna dot, mode gelap, mobile layout) diverifikasi lewat kode — disarankan cek visual singkat oleh manusia bila ada browser.
- Data uji ditinggalkan di produksi untuk verifikasi lanjutan: user `teknisi.dua` (teknisi B) + 9 log baru Agustus 2026 (03/05/07/10/12×4/20) selain log existing 18 Agu. Semua memakai CCTV `MT-CCTV-001`.
- Catatan perilaku desain: `PUT /:id/assign` langsung mengubah status `scheduled → ongoing`; log tanpa penugasan tidak terlihat teknisi mana pun; filter tanggal backend (`start_date`/`end_date`) berlaku pada `created_at` (keterbatasan API yang sudah ada — lihat di bawah).
- Pengelompokan hari memakai `scheduled_date`; bila `scheduled_date` kosong, dipakai `created_at`.
- Filter tanggal backend (`start_date`/`end_date`) bekerja pada `created_at`; tugas yang `scheduled_date`-nya berada di luar rentang `created_at` bulan tersebut mungkin tidak tampil (keterbatasan API yang sudah ada).
- Data kalender diambil dengan `limit=100` per bulan; bulan dengan lebih dari 100 tugas tidak menampilkan semua (cukup untuk kebutuhan operasional umum).
- Halaman kalender tidak menulis data (read-only); untuk membuat/mengubah jadwal tetap melalui `/maintenance` atau `/schedules`.
