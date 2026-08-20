# Uji Manual - Laporan Kustom (Report Builder)

> **Kontrak uji formal — MT-18**
>
> Dokumen ini mendefinisikan prosedur; hasil sebelum **2026-08-20** adalah catatan historis/non-authoritative dan harus diuji ulang dengan execution record formal. Inspeksi kode, review unit test, atau checkmark lama bukan `PASS` black-box.
>
> **Skenario audit reproduktif:** Dengan fixture empat entitas dan template per owner/site, uji navigasi langsung, API role enforcement, filter/export CSV/Excel/PDF, dan QA layout.
>
> **Status suite:** `NOT RUN`

### Formal execution metadata

| Field | Value |
|---|---|
| Test ID / Audit ID | MT-18 |
| Capability owner | QA Dokumen/Security + Frontend/Backend |
| Status suite | NOT RUN |
| Build SHA | NOT RECORDED |
| Artifact digest/ID | NOT RECORDED |
| Tester | NOT RECORDED |
| Timestamp WITA | NOT RECORDED |
| Environment | NOT RECORDED |
| Fixture prefix/data | `E2E-MT18-` / NOT RECORDED |
| Cleanup result | NOT RUN |
| Expected | Teknisi diarahkan keluar dan API 403, format valid, layout rapi, filter/kolom benar, serta template/data tidak bocor lintas user/site. |
| Actual | NOT RECORDED |
| Evidence path/link | NOT RECORDED |
| Mandatory flag | YES |
| Scenario status | NOT RUN |

### Formal scenario record

| ID | Mandatory | Expected | Actual | Evidence | Status |
|---|---|---|---|---|---|
| MT-18-S1 | YES | Teknisi diarahkan keluar dan API 403, format valid, layout rapi, filter/kolom benar, serta template/data tidak bocor lintas user/site. | NOT RECORDED | NOT RECORDED | NOT RUN |

**Aturan kesimpulan suite:** sebelum eksekusi suite `NOT RUN`; setelah run dimulai, setiap mandatory scenario harus `PASS`. Satu mandatory scenario berstatus `FAIL`, `BLOCKED`, atau `NOT RUN` membuat kesimpulan suite `FAIL`.

> **Guard keselamatan wajib:** Untuk setiap SQL mutasi, `DELETE`/`TRUNCATE`, rotasi/regenerasi key atau QR, penghapusan, reset kredensial, atau mutasi credential, lanjut hanya bila environment bukan production dan target bukan `nexaspace.site`, fixture prefix `E2E-MT18-` serta exact targets sudah dicatat, backup/recovery terverifikasi untuk perubahan data/key state, dan cleanup/rollback didefinisikan serta dicatat setelah run. Jika salah satu syarat tidak tersedia, hentikan prosedur dan set scenario status ke `BLOCKED`; jangan jalankan pada production. Jangan menyalin credential, token, private key, RTSP password, atau payload production yang belum disanitasi.


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

| Item | Observasi historis (non-authoritative) |
|------|-------------|
| `go build ./...` | NOT RUN |
| `go test ./... -count=1` | NOT RUN (termasuk `report_builder_test.go`: generate CSV/XLSX/PDF, simpan/muat/hapus template) |
| `go vet ./...` | NOT RUN |
| `npm run lint` | NOT RUN 0 error |
| `npm run build` | NOT RUN |

## 3. Skenario Uji (Blackbox)

> **Catatan legacy:** tabel di bawah adalah observasi historis/non-authoritative; jangan gunakan sebagai status formal. Isi `Formal scenario record` di atas dengan expected, actual, evidence, dan status formal.

### S1 - Halaman & Navigasi

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Buka `/laporan-kustom` sebagai admin | Halaman report builder tampil dengan 4 langkah (entitas, kolom, filter, format) | NOT RUN |
| 2 | Cek sidebar | Menu "Laporan" (`FileSpreadsheet`) tampil di section admin | NOT RUN |
| 3 | Login sebagai teknisi, akses `/laporan-kustom` langsung | Diarahkan kembali ke `/dashboard` (proxy middleware) | NOT RUN Masih bisa terbuka |

### S2 - Pilih Entitas & Kolom

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Klik entitas "CCTV" | Kolom CCTV (Kode, Nama, Lokasi, dll.) tercentang semua | NOT RUN |
| 2 | Ganti ke entitas "Suku Cadang" | Kolom ter-reset ke semua kolom suku cadang (Nama, SKU, Stok, dll.) | NOT RUN |
| 3 | Hapus beberapa centang kolom | Kolom yang dipilih berkurang | NOT RUN |
| 4 | Klik "Kosongkan" / "Pilih semua" | Seleksi kolom berubah sesuai | NOT RUN |
| 5 | Hapus semua centang lalu klik "Unduh" | Muncul error "Pilih minimal satu kolom" | NOT RUN |

### S3 - Filter

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Pilih entitas "Pemeliharaan" | Filter status & rentang tanggal tampil | NOT RUN |
| 2 | Pilih status "Selesai" | Laporan hanya berisi log selesai | NOT RUN |
| 3 | Pilih rentang tanggal | Laporan terfilter sesuai rentang | NOT RUN |
| 4 | Pilih entitas "Suku Cadang" | Filter kategori & "Stok rendah saja" tampil | NOT RUN |
| 5 | Pilih "Stok rendah saja" | Laporan hanya berisi suku cadang `stok < minimum_order` | NOT RUN |

### S4 - Unduh Format CSV / Excel / PDF

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Pilih format CSV, klik "Unduh Laporan" | File `.csv` terunduh; dapat dibuka di spreadsheet; header = label kolom | NOT RUN |
| 2 | Pilih format Excel, unduh | File `.xlsx` terunduh dan valid | NOT RUN |
| 3 | Pilih format PDF, unduh | File `.pdf` terunduh dan valid (mulai `%PDF`) | NOT RUN |
| 4 | Periksa isi file | Data sesuai kolom & filter yang dipilih | NOT RUN |

### S5 - Simpan Template

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Atur entitas/kolom/filter/format, isi nama template, klik "Simpan Template" | Template tersimpan; muncul di daftar "Template Tersimpan" | NOT RUN |
| 2 | Simpan template tanpa nama | Muncul error "Tuliskan nama template" | NOT RUN |
| 3 | Cek database `report_templates` | Baris baru dengan `config` JSON valid dan `created_by` = admin | NOT RUN |

### S6 - Muat & Hapus Template

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Klik nama template di daftar | Konfigurasi template dimuat ke form (entitas, kolom, filter, format) | NOT RUN |
| 2 | Ubah pengaturan lalu unduh | Laporan sesuai konfigurasi termuat (yang diubah) | NOT RUN |
| 3 | Klik ikon hapus template → konfirmasi | Template terhapus (soft-delete) dan hilang dari daftar | NOT RUN |

### S7 - Role Enforcement & Validasi

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Teknisi memanggil `POST /api/reports/generate` | Respons 403 (Hanya Admin) | NOT RUN |
| 2 | Teknisi memanggil `POST /api/reports/templates` | Respons 403 | NOT RUN |
| 3 | Admin kirim `entity` tidak dikenal (mis. "assets") | Respons 400 | NOT RUN |
| 4 | Admin kirim `format` tidak valid | Respons 400 | NOT RUN |
| 5 | Admin simpan template dengan `config` bukan JSON | Respons 400 | NOT RUN |

### S8 - API Konsistensi

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | `GET /api/reports/templates` | Daftar template terpaginasi-sederhana (semua template) | NOT RUN |
| 2 | `DELETE /api/reports/templates/:id` untuk id tak ada | Respons 404 | NOT RUN |

## 4. Kriteria Penerimaan

1. Model `ReportTemplate` (nama, config jsonb, created_by) + migrasi `000008`.
2. `POST /api/reports/generate` menghasilkan CSV/XLSX/PDF dari 4 entitas (cctv, maintenance_log, field_maintenance, spare_part) dengan filter.
3. `GET/POST /api/reports/templates` dan `DELETE /api/reports/templates/:id` berfungsi (admin only).
4. Halaman `/laporan-kustom` (admin only) memiliki wizard 4 langkah, simpan/muat/hapus template, dan tombol unduh.
5. Menu sidebar "Laporan" muncul untuk admin; proxy melindungi `/laporan-kustom`.
6. `go build`/`go test`/`go vet`/`golangci-lint`/`swag init` dan `npm run lint`/`npm run build` memenuhi kriteria.

## Catatan

- Report builder hanya untuk admin (seluruh endpoint `RequireRole("admin")`).
- Format PDF memakai `github.com/go-pdf/fpdf` (fork `jung-kurt/gofpdf` yang sudah dipakai proyek); Excel memakai `github.com/xuri/excelize/v2` (sudah ada).
- Laporan mengunduh file (download) melalui `POST` dengan autentikasi cookie + CSRF header.
