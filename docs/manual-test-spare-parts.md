# Uji Manual - Inventaris Suku Cadang

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

## 3. Skenario Uji (Blackbox)

> **Catatan verifikator:** setelah menyelesaikan setiap skenario, isi kolom "Hasil nyata" dengan **LULUS** / **GAGAL** + catatan singkat.

### S1 - CRUD Suku Cadang

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buka `/suku-cadang` sebagai admin | Halaman menampilkan tabel suku cadang dengan kolom SKU, Nama, Kategori, Stok, Min Order, Satuan, Harga | |
| 2 | Klik "Tambah Suku Cadang", isi form (nama, SKU unik, kategori, stok, minimum order, satuan, harga) | Muncul toast sukses dan baris baru tampil di tabel | |
| 3 | Klik ikon edit pada baris, ubah stok/harga | Data tersimpan dan tabel diperbarui | |
| 4 | Klik ikon hapus → konfirmasi | Baris hilang (soft delete); SKU dapat dipakai ulang | |
| 5 | Tambah suku cadang dengan SKU duplikat | Ditampilkan error "SKU sudah digunakan" | |

### S2 - Pencarian & Filter Kategori

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Ketik kata kunci di kotak pencarian (nama atau SKU) | Daftar terfilter sesuai kata kunci | |
| 2 | Pilih kategori "Kabel" pada dropdown filter | Hanya suku cadang kategori Kabel yang tampil | |
| 3 | Kombinasikan pencarian + filter | Hasil sesuai kedua kondisi | |
| 4 | Navigasi paginasi | Pindah halaman saat data > limit | |

### S3 - Badge Indikator Stok

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Suku cadang dengan `stok > 2 × minimum_order` | Badge **hijau** "Cukup" | |
| 2 | Suku cadang dengan `minimum_order < stok ≤ 2 × minimum_order` | Badge **kuning** "Menipis" | |
| 3 | Suku cadang dengan `stok < minimum_order` | Badge **merah** "Stok rendah" | |

### S4 - Pencatatan Penggunaan & Stok Otomatis Berkurang

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buka maintenance log berstatus `completed` (ikon mata) | Modal detail menampilkan section **"Suku Cadang Digunakan"** (kosong) | |
| 2 | Klik "Catat Penggunaan", pilih suku cadang, isi jumlah & catatan | Penggunaan tercatat; stok suku cadang berkurang otomatis | |
| 3 | Periksa tabel pada section "Suku Cadang Digunakan" | Baris menampilkan Nama, SKU, Jumlah, Catatan | |
| 4 | Buka halaman `/suku-cadang` | Stok suku cadang tersebut berkurang sesuai jumlah | |
| 5 | Buka maintenance log **bukan** completed (scheduled/ongoing) | Tombol "Catat Penggunaan" tidak tampil (atau backend menolak 409) | |

### S5 - Stok Tidak Cukup

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Catat penggunaan dengan jumlah lebih besar dari stok tersedia | Muncul error "Stok tidak mencukupi" (400) | |
| 2 | Periksa stok suku cadang | Stok **tidak berubah** (transaksi di-rollback) | |
| 3 | Catat penggunaan dengan jumlah 0 atau negatif | Ditolak (validasi) | |

### S6 - Notifikasi / Event Stok Rendah (STOCK_LOW)

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Catat penggunaan hingga stok suku cadang jatuh di bawah `minimum_order` | Event `STOCK_LOW` di-broadcast ke admin | |
| 2 | Dashboard admin terbuka | Widget "Suku Cadang Stok Rendah" ter-refresh real-time dan menampilkan suku cadang tersebut | |
| 3 | Periksa halaman `/suku-cadang` | Suku cadang menampilkan badge merah "Stok rendah" | |

### S7 - Role Enforcement

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Login sebagai **teknisi**, akses `/suku-cadang` langsung via URL | Diarahkan kembali ke `/dashboard` (proxy middleware) | |
| 2 | Teknisi memanggil `POST /api/spare-parts` via Swagger | Respons 403 (Hanya Admin) | |
| 3 | Teknisi memanggil `DELETE /api/spare-parts/:id` | Respons 403 | |
| 4 | Teknisi memanggil `GET /api/spare-parts` | Diperbolehkan (200) — daftar suku cadang dapat dilihat | |

### S8 - Dashboard Widget "Suku Cadang Stok Rendah"

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Login sebagai admin, buka `/dashboard` | Widget "Suku Cadang Stok Rendah" tampil (maks 5 item dengan badge merah) | |
| 2 | Klik "Kelola Suku Cadang" | Berpindah ke halaman `/suku-cadang` | |
| 3 | Login sebagai teknisi, buka `/dashboard` | Widget suku cadang stok rendah **tidak** tampil | |

### S9 - API Low-Stock & Endpoint Konsistensi

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Panggil `GET /api/spare-parts/low-stock` (admin) | Mengembalikan suku cadang dengan `stok < minimum_order`, diurutkan stok terkecil | |
| 2 | Panggil `GET /api/spare-parts/:id` | Detail satu suku cadang | |
| 3 | Panggil `GET /api/maintenance-logs/:id/spare-parts` | Daftar penggunaan suku cadang pada log tersebut | |

## 4. Kriteria Lulus

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
11. `go build`/`go test`/`go vet`/`golangci-lint`/`swag init` dan `npm run lint`/`npm run build` lulus.

## Catatan

- Pencatatan penggunaan hanya diperbolehkan pada maintenance log berstatus `completed` (sudah selesai).
- Pengurangan stok dilakukan atomik dalam transaksi database; kegagalan apa pun mengembalikan stok ke nilai semula.
- Soft delete diterapkan pada suku cadang; riwayat penggunaan pada maintenance log tetap tersimpan.
- Widget dashboard hanya muncul untuk admin dan menampilkan maksimal 5 suku cadang stok terendah.
