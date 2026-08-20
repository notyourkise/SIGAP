# Uji Manual - Inventaris Suku Cadang

> **Kontrak uji formal — MT-20**
>
> Dokumen ini mendefinisikan prosedur; hasil sebelum **2026-08-20** adalah catatan historis/non-authoritative dan harus diuji ulang dengan execution record formal. Inspeksi kode, review unit test, atau checkmark lama bukan `PASS` black-box.
>
> **Skenario audit reproduktif:** Dengan fixture stok per site dan maintenance owner, uji non-owner, retry idempotent, concurrent usage, rollback, batas stok, dan site scope.
>
> **Status suite:** `NOT RUN`

### Formal execution metadata

| Field | Value |
|---|---|
| Test ID / Audit ID | MT-20 |
| Capability owner | QA Inventory + Frontend/Backend |
| Status suite | NOT RUN |
| Build SHA | NOT RECORDED |
| Artifact digest/ID | NOT RECORDED |
| Tester | NOT RECORDED |
| Timestamp WITA | NOT RECORDED |
| Environment | NOT RECORDED |
| Fixture prefix/data | `E2E-MT20-` / NOT RECORDED |
| Cleanup result | NOT RUN |
| Expected | Non-owner ditolak, retry tidak mengurangi stok lagi, stok tidak negatif, rollback memulihkan nilai, dan event low-stock hanya satu kali saat melewati ambang. |
| Actual | NOT RECORDED |
| Evidence path/link | NOT RECORDED |
| Mandatory flag | YES |
| Scenario status | NOT RUN |

### Formal scenario record

| ID | Mandatory | Expected | Actual | Evidence | Status |
|---|---|---|---|---|---|
| MT-20-S1 | YES | Non-owner ditolak, retry tidak mengurangi stok lagi, stok tidak negatif, rollback memulihkan nilai, dan event low-stock hanya satu kali saat melewati ambang. | NOT RECORDED | NOT RECORDED | NOT RUN |

> **Cakupan aggregate Wave 0:** Baris formal aggregate di atas adalah **sole atomic suite-level acceptance gate** untuk baseline Wave 0. Skenario legacy terperinci di bawah ini tetap non-formal dan historical/non-authoritative sampai diperluas selama Wave 7.

**Aturan kesimpulan suite:** sebelum eksekusi suite `NOT RUN`; setelah run dimulai, setiap mandatory scenario harus `PASS`. Satu mandatory scenario berstatus `FAIL`, `BLOCKED`, atau `NOT RUN` membuat kesimpulan suite `FAIL`.

> **Guard keselamatan wajib:** Untuk setiap SQL mutasi, `DELETE`/`TRUNCATE`, rotasi/regenerasi key atau QR, penghapusan, reset kredensial, atau mutasi credential, lanjut hanya bila environment bukan production dan target bukan `nexaspace.site`, fixture prefix `E2E-MT20-` serta exact targets sudah dicatat, backup/recovery terverifikasi untuk perubahan data/key state, dan cleanup/rollback didefinisikan serta dicatat setelah run. Jika salah satu syarat tidak tersedia, hentikan prosedur dan set scenario status ke `BLOCKED`; jangan jalankan pada production. Jangan menyalin credential, token, private key, RTSP password, atau payload production yang belum disanitasi.


> **Task:** 11.1 Inventaris Suku Cadang (Sprint 26)
> **Branch:** `feature/spare-parts`
> **Versi:** backend `0.48.0` - frontend `1.40.0`
> **Dokumen:** verifikasi modul inventaris suku cadang yang terintegrasi dengan alur pemeliharaan: CRUD stok, pencatatan penggunaan yang mengurangi stok otomatis, notifikasi stok rendah, role enforcement, dan widget dashboard.

Modul ini memungkinkan teknisi/admin mencatat suku cadang yang dipakai saat perbaikan CCTV dan memantau stok komponen (kabel, konektor, power supply, dll.). Stok berkurang otomatis saat penggunaan dicatat pada maintenance log berstatus selesai.

## 1. Prasyarat

| # | Item | Cara menyiapkan |
|---|------|-----------------|
| P1 | Aplikasi berjalan | `docker compose up -d --build`, buka `https://localhost` (atau `npm run dev` → `http://localhost:3001`) |
| P2 | Akun admin | Login sebagai admin |
| P3 | Akun teknisi | Siapkan user berperan `teknisi` |
| P4 | Data suku cadang | Tambahkan minimal 2 suku cadang (satu dengan stok di bawah `minimum_order`) |
| P5 | Maintenance log selesai | Siapkan minimal 1 maintenance log berstatus `completed` (untuk uji pencatatan penggunaan) |
| P6 | Suku cadang stok rendah | Set salah satu suku cadang agar `stok < minimum_order` untuk uji widget dashboard |

## 2. Pengujian Otomatis

```powershell
cd backend
go build ./...
go test ./... -count=1
go vet ./...
golangci-lint run ./...
swag init

cd ../frontend
npm run lint
npm run build
```

Hasil yang diharapkan: seluruh perintah sukses; lint tidak memiliki error (0 error).

| Item | Observasi historis (non-authoritative) |
|------|-------------|
| `go build ./...` | NOT RUN |
| `go test ./... -count=1` | NOT RUN (termasuk test spare part & usage) |
| `go vet ./...` | NOT RUN |
| `npm run lint` | NOT RUN 0 error |
| `npm run build` | NOT RUN |

## 3. Skenario Uji (Blackbox)

> **Catatan legacy:** tabel di bawah adalah observasi historis/non-authoritative; jangan gunakan sebagai status formal. Isi `Formal scenario record` di atas dengan expected, actual, evidence, dan status formal.

### S1 - CRUD Suku Cadang

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Buka `/suku-cadang` sebagai admin | Halaman menampilkan tabel suku cadang dengan kolom SKU, Nama, Kategori, Stok, Min Order, Satuan, Harga | NOT RUN |
| 2 | Klik "Tambah Suku Cadang", isi form (nama, SKU unik, kategori, stok, minimum order, satuan, harga) | Muncul toast sukses dan baris baru tampil di tabel | NOT RUN |
| 3 | Klik ikon edit pada baris, ubah stok/harga | Data tersimpan dan tabel diperbarui | NOT RUN |
| 4 | Klik ikon hapus → konfirmasi | Baris hilang (soft delete); SKU dapat dipakai ulang | NOT RUN |
| 5 | Tambah suku cadang dengan SKU duplikat | Ditampilkan error "SKU sudah digunakan" | NOT RUN |

### S2 - Pencarian & Filter Kategori

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Ketik kata kunci di kotak pencarian (nama atau SKU) | Daftar terfilter sesuai kata kunci | NOT RUN |
| 2 | Pilih kategori "Kabel" pada dropdown filter | Hanya suku cadang kategori Kabel yang tampil | NOT RUN |
| 3 | Kombinasikan pencarian + filter | Hasil sesuai kedua kondisi | NOT RUN |
| 4 | Navigasi paginasi | Pindah halaman saat data > limit | NOT RUN |

### S3 - Badge Indikator Stok

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Suku cadang dengan `stok > 2 × minimum_order` | Badge **hijau** "Cukup" | NOT RUN |
| 2 | Suku cadang dengan `minimum_order < stok ≤ 2 × minimum_order` | Badge **kuning** "Menipis" | NOT RUN |
| 3 | Suku cadang dengan `stok < minimum_order` | Badge **merah** "Stok rendah" | NOT RUN |

### S4 - Pencatatan Penggunaan & Stok Otomatis Berkurang

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Buka maintenance log berstatus `completed` (ikon mata) | Modal detail menampilkan section **"Suku Cadang Digunakan"** (kosong) | NOT RUN |
| 2 | Klik "Catat Penggunaan", pilih suku cadang, isi jumlah & catatan | Penggunaan tercatat; stok suku cadang berkurang otomatis | NOT RUN |
| 3 | Periksa tabel pada section "Suku Cadang Digunakan" | Baris menampilkan Nama, SKU, Jumlah, Catatan | NOT RUN |
| 4 | Buka halaman `/suku-cadang` | Stok suku cadang tersebut berkurang sesuai jumlah | NOT RUN |
| 5 | Buka maintenance log **bukan** completed (scheduled/ongoing) | Tombol "Catat Penggunaan" tidak tampil (atau backend menolak 409) | NOT RUN |

### S5 - Stok Tidak Cukup

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Catat penggunaan dengan jumlah lebih besar dari stok tersedia | Muncul error "Stok tidak mencukupi" (400) | NOT RUN |
| 2 | Periksa stok suku cadang | Stok **tidak berubah** (transaksi di-rollback) | NOT RUN |
| 3 | Catat penggunaan dengan jumlah 0 atau negatif | Ditolak (validasi) | NOT RUN |

### S6 - Notifikasi / Event Stok Rendah (STOCK_LOW)

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Catat penggunaan hingga stok suku cadang jatuh di bawah `minimum_order` | Event `STOCK_LOW` di-broadcast ke admin | NOT RUN (verifikasi kode — `spare_part.go:522-523`) |
| 2 | Dashboard admin terbuka | Widget "Suku Cadang Stok Rendah" ter-refresh real-time dan menampilkan suku cadang tersebut | NOT RUN (verifikasi kode — subscribe `STOCK_LOW` di `dashboard/page.tsx:170`) |
| 3 | Periksa halaman `/suku-cadang` | Suku cadang menampilkan badge merah "Stok rendah" | NOT RUN |

### S7 - Role Enforcement

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Login sebagai **teknisi**, akses `/suku-cadang` langsung via URL | Diarahkan kembali ke `/dashboard` (proxy middleware) | NOT RUN |
| 2 | Teknisi memanggil `POST /api/spare-parts` via Swagger | Respons 403 (Hanya Admin) | NOT RUN |
| 3 | Teknisi memanggil `DELETE /api/spare-parts/:id` | Respons 403 | NOT RUN |
| 4 | Teknisi memanggil `GET /api/spare-parts` | Diperbolehkan (200) — daftar suku cadang dapat dilihat | NOT RUN |

### S8 - Dashboard Widget "Suku Cadang Stok Rendah"

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Login sebagai admin, buka `/dashboard` | Widget "Suku Cadang Stok Rendah" tampil (maks 5 item dengan badge merah) | NOT RUN (verifikasi kode — `dashboard/page.tsx:438-441`) |
| 2 | Klik "Kelola Suku Cadang" | Berpindah ke halaman `/suku-cadang` | NOT RUN (verifikasi kode) |
| 3 | Login sebagai teknisi, buka `/dashboard` | Widget suku cadang stok rendah **tidak** tampil | NOT RUN (verifikasi kode — guard `isAdmin` di `dashboard/page.tsx:119`) |

### S9 - API Low-Stock & Endpoint Konsistensi

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Panggil `GET /api/spare-parts/low-stock` (admin) | Mengembalikan suku cadang dengan `stok < minimum_order`, diurutkan stok terkecil | NOT RUN |
| 2 | Panggil `GET /api/spare-parts/:id` | Detail satu suku cadang | NOT RUN |
| 3 | Panggil `GET /api/maintenance-logs/:id/spare-parts` | Daftar penggunaan suku cadang pada log tersebut | NOT RUN |

## 4. Kriteria Penerimaan

1. Model `SparePart` & `SparePartUsage` dibuat + migrasi `000006_spare_parts` (up/down) tersedia.
2. CRUD `/api/spare-parts` berfungsi (admin-only untuk tulis, protected untuk baca).
3. `POST /api/maintenance-logs/:id/spare-parts` mengurangi stok otomatis dalam transaksi, hanya untuk log `completed`.
4. Stok tidak cukup → 400 tanpa mengubah stok (rollback).
5. Stok jatuh di bawah `minimum_order` → broadcast `STOCK_LOW` ke admin.
6. `GET /api/spare-parts/low-stock` mengembalikan suku cadang stok rendah untuk widget dashboard.
7. Halaman `/suku-cadang` (admin-only) menampilkan tabel, filter kategori, pencarian, badge stok, modal form, hapus konfirmasi, paginasi.
8. Maintenance modal menampilkan section "Suku Cadang Digunakan" + mini modal "Catat Penggunaan".
9. Dashboard admin menampilkan widget "Suku Cadang Stok Rendah".
10. Menu sidebar "Suku Cadang" muncul untuk admin; proxy melindungi `/suku-cadang`.
11. `go build`/`go test`/`go vet`/`golangci-lint`/`swag init` dan `npm run lint`/`npm run build` memenuhi kriteria.

## Catatan

- Pencatatan penggunaan hanya diperbolehkan pada maintenance log berstatus `completed` (sudah selesai).
- Pengurangan stok dilakukan atomik dalam transaksi database; kegagalan apa pun mengembalikan stok ke nilai semula.
- Soft delete diterapkan pada suku cadang; riwayat penggunaan pada maintenance log tetap tersimpan.
- Widget dashboard hanya muncul untuk admin dan menampilkan maksimal 5 suku cadang stok terendah.

## Observasi Historis (Non-authoritative) — Catatan Verifikator

**Kesimpulan historis (non-authoritative): S1–S9 NOT RUN.** Uji nyata di produksi: CRUD (201/200/soft-delete/SKU reuse), duplikat SKU → 400 "SKU sudah digunakan", low-stock (stok 5 < min 20 → tampil), pencatatan penggunaan pada log completed → 201 + stok otomatis 5→3, jumlah melebihi stok → 400 "Stok tidak mencukupi. Tersedia 3 pcs" (stok tetap, rollback), jumlah 0/-1 → 400 (validasi `gt:0`), teknisi POST/DELETE → 403 & GET → 200, `GET /api/maintenance-logs/:id/spare-parts` → 200. Event `STOCK_LOW` & widget dashboard terverifikasi dari kode.
