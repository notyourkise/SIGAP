# Uji Manual - Kalender Jadwal Pemeliharaan

> **Kontrak uji formal — MT-02**
>
> Dokumen ini mendefinisikan prosedur; hasil sebelum **2026-08-20** adalah catatan historis/non-authoritative dan harus diuji ulang dengan execution record formal. Inspeksi kode, review unit test, atau checkmark lama bukan `PASS` black-box.
>
> **Skenario audit reproduktif:** Dengan fixture tanggal lintas bulan/tahun dan lebih dari 100 tugas, buka kalender dan bandingkan pengelompokan berdasarkan `scheduled_date`, filter teknisi, serta paginasi.
>
> **Status suite:** `NOT RUN`

### Formal execution metadata

| Field | Value |
|---|---|
| Test ID / Audit ID | MT-02 |
| Capability owner | QA Web + Frontend/Backend |
| Status suite | NOT RUN |
| Build SHA | NOT RECORDED |
| Artifact digest/ID | NOT RECORDED |
| Tester | NOT RECORDED |
| Timestamp WITA | NOT RECORDED |
| Environment | NOT RECORDED |
| Fixture prefix/data | `E2E-MT02-` / NOT RECORDED |
| Cleanup result | NOT RUN |
| Expected | Semua tugas fixture tampil pada bulan `scheduled_date` yang benar tanpa terpotong paginasi; scope teknisi dan boundary tanggal konsisten. |
| Actual | NOT RECORDED |
| Evidence path/link | NOT RECORDED |
| Mandatory flag | YES |
| Scenario status | NOT RUN |

### Formal scenario record

| ID | Mandatory | Expected | Actual | Evidence | Status |
|---|---|---|---|---|---|
| MT-02-S1 | YES | Semua tugas fixture tampil pada bulan `scheduled_date` yang benar tanpa terpotong paginasi; scope teknisi dan boundary tanggal konsisten. | NOT RECORDED | NOT RECORDED | NOT RUN |

> **Cakupan aggregate Wave 0:** Baris formal aggregate di atas adalah **sole atomic suite-level acceptance gate** untuk baseline Wave 0. Skenario legacy terperinci di bawah ini tetap non-formal dan historical/non-authoritative sampai diperluas selama Wave 7.

**Aturan kesimpulan suite:** sebelum eksekusi suite `NOT RUN`; setelah run dimulai, setiap mandatory scenario harus `PASS`. Satu mandatory scenario berstatus `FAIL`, `BLOCKED`, atau `NOT RUN` membuat kesimpulan suite `FAIL`.

> **Guard keselamatan wajib:** Untuk setiap SQL mutasi, `DELETE`/`TRUNCATE`, rotasi/regenerasi key atau QR, penghapusan, reset kredensial, atau mutasi credential, lanjut hanya bila environment bukan production dan target bukan `nexaspace.site`, fixture prefix `E2E-MT02-` serta exact targets sudah dicatat, backup/recovery terverifikasi untuk perubahan data/key state, dan cleanup/rollback didefinisikan serta dicatat setelah run. Jika salah satu syarat tidak tersedia, hentikan prosedur dan set scenario status ke `BLOCKED`; jangan jalankan pada production. Jangan menyalin credential, token, private key, RTSP password, atau payload production yang belum disanitasi.


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

Hasil yang diharapkan: `npm run lint` sukses (0 error) dan `npm run build` sukses (exit 0). Catatan: beberapa `warning` lint lama yang sudah ada tidak memengaruhi memenuhi kriteria/tidaknya.

| Item | Observasi historis (non-authoritative) |
|------|-------------|
| `npm run lint` | NOT RUN |
| `npm run build` | NOT RUN |

## 3. Skenario Uji (Blackbox)

> **Catatan legacy:** tabel di bawah adalah observasi historis/non-authoritative; jangan gunakan sebagai status formal. Isi `Formal scenario record` di atas dengan expected, actual, evidence, dan status formal.

### S1 - Menu dan akses halaman

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Login sebagai admin, lihat sidebar | Menu **Kalender** (`/kalender`) tampil dengan ikon kalender | NOT RUN |
| 2 | Klik menu Kalender | Halaman `/kalender` terbuka dengan judul "Kalender Jadwal Pemeliharaan" | NOT RUN |
| 3 | Login sebagai teknisi, buka menu Kalender | Teknisi juga dapat mengakses halaman kalender (bukan admin-only) | NOT RUN |
| 4 | Halaman tanpa sesi | Dialihkan ke `/login` | NOT RUN |

### S2 - Tampilan kalender bulanan

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Buka `/kalender` | Kalender menampilkan bulan berjalan dengan grid 7 kolom | NOT RUN |
| 2 | Periksa header hari | Header menampilkan Sen, Sel, Rab, Kam, Jum, Sab, Min | NOT RUN |
| 3 | Periksa jumlah baris | Grid memuat 5-6 baris (hari diisi dari Senin sebagai kolom pertama) | NOT RUN |
| 4 | Periksa judul bulan | Judul menampilkan nama bulan + tahun (mis. "Agustus 2026") | NOT RUN |
| 5 | Cek tanggal hari ini | Tanggal hari ini ditandai (lingkaran Hijau Komando) | NOT RUN |

### S3 - Navigasi bulan

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Klik tombol panah kiri (prev) | Bulan bergeser ke bulan sebelumnya; grid diperbarui | NOT RUN |
| 2 | Klik tombol panah kanan (next) | Bulan bergeser ke bulan berikutnya; grid diperbarui | NOT RUN |
| 3 | Pindah beberapa bulan lalu klik "Hari ini" | Kembali ke bulan berjalan dan menandai tanggal hari ini | NOT RUN |
| 4 | Navigasi melintasi pergantian tahun | Kalender menangani Desember→Januari dan Januari→Desember dengan benar | NOT RUN |

### S4 - Dot warna status tugas

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Siapkan log `completed` di suatu hari | Hari menampilkan dot **hijau** | NOT RUN |
| 2 | Siapkan log `scheduled` di suatu hari | Hari menampilkan dot **biru** | NOT RUN |
| 3 | Siapkan log `ongoing` di suatu hari | Hari menampilkan dot **kuning** | NOT RUN |
| 4 | Siapkan log `failed` (atau overdue) di suatu hari | Hari menampilkan dot **merah** | NOT RUN |
| 5 | Hari dengan > 3 tugas | Menampilkan 3 dot + indikator "+N" untuk sisa tugas | NOT RUN |
| 6 | Periksa legend di bawah kalender | Legend menjelaskan warna: Selesai, Terjadwal, Berjalan, Gagal/Overdue | NOT RUN |

### S5 - Klik hari → detail tugas

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Klik hari yang memiliki tugas | Panel detail menampilkan tanggal hari tersebut (mis. "Senin, 7 Agustus 2026") | NOT RUN |
| 2 | Periksa daftar tugas | Setiap tugas menampilkan nama CCTV, kode CCTV, status (badge), tindakan, dan nama teknisi | NOT RUN |
| 3 | Klik hari yang tidak memiliki tugas | Panel menampilkan "Tidak ada tugas" | NOT RUN |
| 4 | Klik hari yang sama lagi | Panel detail tertutup / kembali ke kondisi awal | NOT RUN |
| 5 | Klik tombol X pada panel | Panel detail tertutup | NOT RUN |

### S6 - Filter per teknisi (admin)

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Sebagai admin, periksa dropdown "Teknisi" | Dropdown muncul dengan opsi "Semua teknisi" dan daftar teknisi | NOT RUN |
| 2 | Pilih teknisi A | Kalender hanya menampilkan dot tugas milik teknisi A | NOT RUN |
| 3 | Klik hari dengan tugas | Detail hanya menampilkan tugas teknisi A | NOT RUN |
| 4 | Pilih "Semua teknisi" | Seluruh tugas dari semua teknisi tampil kembali | NOT RUN |
| 5 | Pilih teknisi yang tidak punya tugas | Kalender tanpa dot; klik hari → "Tidak ada tugas" | NOT RUN |

### S7 - Scope teknisi (non-admin)

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Login sebagai teknisi A | Dropdown filter teknisi **tidak** muncul | NOT RUN |
| 2 | Periksa kalender | Hanya tugas milik teknisi A yang tampil (tugas teknisi B tidak muncul) | NOT RUN |
| 3 | Klik hari | Detail hanya berisi tugas milik teknisi A | NOT RUN |

### S8 - Konsistensi data dengan halaman Pemeliharaan

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Buka `/maintenance` dan catat log pada bulan berjalan | Log tersebut muncul sebagai dot di hari `scheduled_date` pada kalender | NOT RUN |
| 2 | Buat log baru via `/maintenance` | Log baru tampil di kalender setelah data dimuat ulang (refresh/navigasi bulan) | NOT RUN |
| 3 | Ubah status log di `/maintenance` | Warna dot pada kalender menyesuaikan setelah refresh | NOT RUN |

### S9 - Responsif & aksesibilitas

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Buka di layar mobile (lebar < 768px) | Kalender tetap terbaca; grid 7 kolom tidak merusak layout; panel detail menumpuk di bawah | NOT RUN |
| 2 | Gunakan keyboard (Tab + Enter) untuk memilih hari | Hari dapat dipilih via keyboard (tombol semantik) | NOT RUN |
| 3 | Periksa atribut aria pada tombol hari | Tombol hari memiliki `aria-label` yang menjelaskan tanggal & jumlah tugas | NOT RUN |
| 4 | Mode gelap | Kalender tetap kontras dan terbaca (mengikuti tema) | NOT RUN |

## 4. Kriteria Penerimaan

1. Halaman `/kalender` terbuka untuk admin dan teknisi; menu "Kalender" ada di sidebar.
2. Kalender menampilkan bulan berjalan dengan grid 7 kolom dan header Senin–Minggu.
3. Navigasi prev/next bulan dan tombol "Hari ini" berfungsi.
4. Dot warna mengikuti status tugas (hijau completed, biru scheduled, kuning ongoing, merah failed/overdue) dan ada legend.
5. Klik hari menampilkan panel detail berisi daftar tugas (nama CCTV, kode, status, tindakan, teknisi).
6. Admin dapat memfilter per teknisi; teknisi hanya melihat data miliknya.
7. `npm run lint` (0 error) dan `npm run build` (exit 0) memenuhi kriteria.

## Catatan

- **Hasil uji (18 Agu 2026, produksi `<PRODUCTION_TARGET>`):** seluruh skenario S1–S9 **NOT RUN** (37/37 langkah + 2/2 item pengujian otomatis). Verifikasi dilakukan kombinasi: akses HTTP nyata (`/kalender` 200 utk admin & teknisi; API `maintenance-logs` utk scope admin/teknisi A/B/401) + inspeksi kode komponen (perilaku browser client-rendered tidak dapat diverifikasi lewat HTML mentah).
- Perilaku UI murni visual (warna dot, mode gelap, mobile layout) diverifikasi lewat kode — disarankan cek visual singkat oleh manusia bila ada browser.
- Data uji ditinggalkan di produksi untuk verifikasi lanjutan: user `teknisi.dua` (teknisi B) + 9 log baru Agustus 2026 (03/05/07/10/12×4/20) selain log existing 18 Agu. Semua memakai CCTV `MT-CCTV-001`.
- Catatan perilaku desain: `PUT /:id/assign` langsung mengubah status `scheduled → ongoing`; log tanpa penugasan tidak terlihat teknisi mana pun; filter tanggal backend (`start_date`/`end_date`) berlaku pada `created_at` (keterbatasan API yang sudah ada — lihat di bawah).
- Pengelompokan hari memakai `scheduled_date`; bila `scheduled_date` kosong, dipakai `created_at`.
- Filter tanggal backend (`start_date`/`end_date`) bekerja pada `created_at`; tugas yang `scheduled_date`-nya berada di luar rentang `created_at` bulan tersebut mungkin tidak tampil (keterbatasan API yang sudah ada).
- Data kalender diambil dengan `limit=100` per bulan; bulan dengan lebih dari 100 tugas tidak menampilkan semua (cukup untuk kebutuhan operasional umum).
- Halaman kalender tidak menulis data (read-only); untuk membuat/mengubah jadwal tetap melalui `/maintenance` atau `/schedules`.
