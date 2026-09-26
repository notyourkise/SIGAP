# Uji Manual - Laporan Kustom (Report Builder)

> **Task:** 11.2 Laporan Kustom (Report Builder) (Sprint 27)
> **Branch:** `feature/email-reports`
> **Versi:** backend `0.49.0` - frontend `1.41.0`
> **Dokumen:** verifikasi report builder — generate laporan (CSV/Excel/PDF) dari entitas CCTV/Pemeliharaan/Pengecekan/Suku Cadang, simpan/muat/hapus template, dan role enforcement.

Modul ini memungkinkan admin membuat laporan kustom sesuai kebutuhan: memilih entitas, kolom, filter, dan format lalu mengunduh file. Template yang disimpan dapat dipakai ulang.

## 1. Prasyarat

| # | Item | Cara menyiapkan |
|---|------|-----------------|
| P1 | Aplikasi berjalan | `docker compose up -d --build`, buka `https://localhost` (atau `npm run dev` → `http://localhost:3001`) |
| P2 | Akun admin | Login sebagai admin |
| P3 | Akun teknisi | Siapkan user berperan `teknisi` (untuk uji role enforcement) |
| P4 | Data | Pastikan ada data CCTV, maintenance log, field maintenance, dan suku cadang |

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

| Item | Hasil nyata |
|------|-------------|
| `go build ./...` | ✅ |
| `go test ./... -count=1` | ✅ (termasuk `report_builder_test.go`: generate CSV/XLSX/PDF, simpan/muat/hapus template) |
| `go vet ./...` | ✅ |
| `npm run lint` | ✅ 0 error |
| `npm run build` | ✅ |

## 3. Skenario Uji (Blackbox)

> **Catatan verifikator:** setelah menyelesaikan setiap skenario, isi kolom "Hasil nyata" dengan **LULUS** / **GAGAL** + catatan singkat.

### S1 - Halaman & Navigasi

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buka `/laporan-kustom` sebagai admin | Halaman report builder tampil dengan 4 langkah (entitas, kolom, filter, format) | ✅ |
| 2 | Cek sidebar | Menu "Laporan" (`FileSpreadsheet`) tampil di section admin | ✅ |
| 3 | Login sebagai teknisi, akses `/laporan-kustom` langsung | Diarahkan kembali ke `/dashboard` (proxy middleware) | ❌ Masih bisa terbuka |

### S2 - Pilih Entitas & Kolom

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Klik entitas "CCTV" | Kolom CCTV (Kode, Nama, Lokasi, dll.) tercentang semua | ✅ |
| 2 | Ganti ke entitas "Suku Cadang" | Kolom ter-reset ke semua kolom suku cadang (Nama, SKU, Stok, dll.) | ✅ |
| 3 | Hapus beberapa centang kolom | Kolom yang dipilih berkurang | ✅ |
| 4 | Klik "Kosongkan" / "Pilih semua" | Seleksi kolom berubah sesuai | ✅ |
| 5 | Hapus semua centang lalu klik "Unduh" | Muncul error "Pilih minimal satu kolom" | ✅ |

### S3 - Filter

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Pilih entitas "Pemeliharaan" | Filter status & rentang tanggal tampil | ✅ |
| 2 | Pilih status "Selesai" | Laporan hanya berisi log selesai | ✅ |
| 3 | Pilih rentang tanggal | Laporan terfilter sesuai rentang | ✅ |
| 4 | Pilih entitas "Suku Cadang" | Filter kategori & "Stok rendah saja" tampil | ✅ |
| 5 | Pilih "Stok rendah saja" | Laporan hanya berisi suku cadang `stok < minimum_order` | ✅ |

### S4 - Unduh Format CSV / Excel / PDF

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Pilih format CSV, klik "Unduh Laporan" | File `.csv` terunduh; dapat dibuka di spreadsheet; header = label kolom | ✅ |
| 2 | Pilih format Excel, unduh | File `.xlsx` terunduh dan valid | ✅ |
| 3 | Pilih format PDF, unduh | File `.pdf` terunduh dan valid (mulai `%PDF`) | ✅ |
| 4 | Periksa isi file | Data sesuai kolom & filter yang dipilih | ✅ |

### S5 - Simpan Template

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Atur entitas/kolom/filter/format, isi nama template, klik "Simpan Template" | Template tersimpan; muncul di daftar "Template Tersimpan" | ✅ |
| 2 | Simpan template tanpa nama | Muncul error "Tuliskan nama template" | ✅ |
| 3 | Cek database `report_templates` | Baris baru dengan `config` JSON valid dan `created_by` = admin | ✅ |

### S6 - Muat & Hapus Template

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Klik nama template di daftar | Konfigurasi template dimuat ke form (entitas, kolom, filter, format) | ✅ |
| 2 | Ubah pengaturan lalu unduh | Laporan sesuai konfigurasi termuat (yang diubah) | ✅ |
| 3 | Klik ikon hapus template → konfirmasi | Template terhapus (soft-delete) dan hilang dari daftar | ✅ |

### S7 - Role Enforcement & Validasi

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Teknisi memanggil `POST /api/reports/generate` | Respons 403 (Hanya Admin) | ✅ |
| 2 | Teknisi memanggil `POST /api/reports/templates` | Respons 403 | ✅ |
| 3 | Admin kirim `entity` tidak dikenal (mis. "assets") | Respons 400 | ✅ |
| 4 | Admin kirim `format` tidak valid | Respons 400 | ✅ |
| 5 | Admin simpan template dengan `config` bukan JSON | Respons 400 | ✅ |

### S8 - API Konsistensi

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | `GET /api/reports/templates` | Daftar template terpaginasi-sederhana (semua template) | ✅ |
| 2 | `DELETE /api/reports/templates/:id` untuk id tak ada | Respons 404 | ✅ |

## 4. Kriteria Lulus

1. Model `ReportTemplate` (nama, config jsonb, created_by) + migrasi `000008`.
2. `POST /api/reports/generate` menghasilkan CSV/XLSX/PDF dari 4 entitas (cctv, maintenance_log, field_maintenance, spare_part) dengan filter.
3. `GET/POST /api/reports/templates` dan `DELETE /api/reports/templates/:id` berfungsi (admin only).
4. Halaman `/laporan-kustom` (admin only) memiliki wizard 4 langkah, simpan/muat/hapus template, dan tombol unduh.
5. Menu sidebar "Laporan" muncul untuk admin; proxy melindungi `/laporan-kustom`.
6. `go build`/`go test`/`go vet`/`golangci-lint`/`swag init` dan `npm run lint`/`npm run build` lulus.

## Catatan

- Report builder hanya untuk admin (seluruh endpoint `RequireRole("admin")`).
- Format PDF memakai `github.com/go-pdf/fpdf` (fork `jung-kurt/gofpdf` yang sudah dipakai proyek); Excel memakai `github.com/xuri/excelize/v2` (sudah ada).
- Laporan mengunduh file (download) melalui `POST` dengan autentikasi cookie + CSRF header.
