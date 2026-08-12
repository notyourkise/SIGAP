# Uji Manual - Template Checklist Pengecekan Lapangan (P12)

> **Task:** 13.12 Checklist Template yang Dapat Dikustomisasi (Sprint 35 - P12)
> **Branch:** `feature/performance-checklist`
> **Versi:** backend `0.56.0` / frontend `1.48.0`
> **Dokumen:** verifikasi admin dapat mengelola item checklist pengecekan lapangan (tambah/ubah/urutan/wajib/nonaktifkan) tanpa deploy ulang, serta wizard pengecekan memakai template dari server.

## 1. Prasyarat

| # | Item | Cara menyiapkan |
|---|------|-----------------|
| P1 | Aplikasi berjalan | `docker compose up -d --build`, buka `https://localhost` (atau `go run main.go` di `backend/` + `npm run dev` di `frontend/` → `http://localhost:3001`) |
| P2 | Migrasi berjalan | Migration `000017_checklist_templates` otomatis dijalankan saat startup; tabel `checklist_templates` dibuat & di-seed 9 item default |
| P3 | Akun Admin | Login sebagai admin (untuk kelola template) |
| P4 | Akun Teknisi | Login sebagai teknisi (untuk verifikasi wizard memakai template) |

## 2. Pengujian Otomatis

```powershell
cd backend
go build ./...
go test ./... -run "TestChecklist|TestCreateFieldMaintenance" -count=1
go vet ./...

cd ../frontend
npm run lint
npm run build
```

Hasil yang diharapkan: seluruh build & test backend lulus, lint (0 error) & build frontend sukses (exit 0).

## 3. Skenario Uji (Blackbox)

> **Catatan verifikator:** setelah menyelesaikan setiap skenario, isi kolom "Hasil nyata" dengan **LULUS** / **GAGAL** + catatan singkat.

### S1 - API daftar template tersedia & ter-seed default

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Login admin, buka `GET /api/checklist-templates` | Respons `{status:"success", data:{rows:[...], pagination}}` berisi 9 item default, terurut `urutan` 1–9 |
| 2 | Cek item default | Terdapat "Pengecekan jaringan", "Pengecekan pasokan listrik", dst. (9 item) dengan `wajib:true`, `aktif:true` |
| 3 | Filter `?aktif=true` | Respons hanya memuat item dengan `aktif:true` |

### S2 - Menambah item template baru (admin)

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buka `/checklist-templates` (menu "Template Checklist") | Halaman kelola template tampil untuk admin |
| 2 | Klik "Tambah Item", isi nama/deskripsi/urutan/wajib, simpan | Item baru muncul di tabel dengan urutan sesuai; toast sukses |
| 3 | Cek `GET /api/checklist-templates` | Item baru ada di daftar |
| 4 | Login sebagai teknisi, buka halaman Field Maintenance | Wizard pengecekan kini menampilkan item baru sebagai langkah |

### S3 - Mengubah item template (admin)

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Di `/checklist-templates`, klik Edit pada sebuah item | Form edit terisi data lama |
| 2 | Ubah nama/deskripsi, simpan | Perubahan tersimpan; tabel diperbarui |
| 3 | Buka wizard pengecekan sebagai teknisi | Nama item di wizard mengikuti perubahan |

### S4 - Reorder urutan item (admin)

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Di `/checklist-templates`, gunakan tombol naik/turun pada item | Urutan item bertukar sesuai arah tombol |
| 2 | Muat ulang halaman | Urutan tetap tersimpan (dikirim ke `PUT /checklist-templates/reorder`) |
| 3 | Buka wizard pengecekan sebagai teknisi | Urutan langkah di wizard mengikuti urutan template |

### S5 - Menonaktifkan item (soft delete)

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Di `/checklist-templates`, klik Hapus pada item | Muncul dialog konfirmasi "Item ini akan dinonaktifkan. Pengecekan yang sudah ada tidak terpengaruh." |
| 2 | Konfirmasi | Item dihapus dari daftar (set `aktif=false`, bukan hard delete) |
| 3 | Buka wizard pengecekan sebagai teknisi | Item yang dinonaktifkan tidak muncul sebagai langkah |
| 4 | Cek pengecekan lama | Pengecekan yang sudah tersimpan sebelum penonaktifan tidak berubah (masih menampilkan item) |

### S6 - Batasan akses (role)

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Login sebagai teknisi, akses `POST /api/checklist-templates` | Ditolak (403) — hanya admin yang dapat menulis |
| 2 | Login sebagai teknisi, akses `GET /api/checklist-templates` | Diizinkan (200) — semua role login bisa baca daftar template |
| 3 | Tanpa login, akses `GET /api/checklist-templates` | Ditolak (401) |

### S7 - End-to-end: template dipakai di pengecekan baru

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Sebagai admin, tambah item "Pengecekan tambahan khusus" (urutan 10) | Item tersimpan |
| 2 | Sebagai teknisi, lakukan pengecekan lapangan (Scan QR → wizard) | Wizard menampilkan 10 langkah checklist (9 default + 1 baru) |
| 3 | Isi seluruh item + foto bukti, submit | Catatan tersimpan; detail pengecekan menampilkan seluruh 10 item |
| 4 | Cek `GET /api/field-maintenance` detail | Item "Pengecekan tambahan khusus" ada di `checks` |

## 4. Kriteria Lulus

1. Migration `000017_checklist_templates` berjalan & tabel ter-seed 9 item default.
2. Admin dapat tambah/ubah/urutkan/nonaktifkan item checklist dari UI tanpa deploy ulang.
3. `POST /api/field-maintenance` menyimpan `wajib` pada item `FieldMaintenanceCheck` (default true).
4. Wizard pengecekan lapangan memakai template dari server (bukan konstanta hardcoded); fallback ke konstanta bila API kosong/gagal.
5. Pengecekan existing tidak terpengaruh oleh perubahan template.
6. Hanya admin yang dapat menulis template; semua role login dapat membaca.
7. `go build`/`go test`/`go vet` backend lulus; `npm run lint`/`npm run build` frontend lulus.

## Catatan

- Template hanya berlaku untuk pengecekan **baru**; pengecekan yang sudah tersimpan menyimpan snapshot nama item pada `FieldMaintenanceCheck.Name`, sehingga tidak berubah saat template diedit.
- Toggle "Wajib" saat ini bersifat informasional pada template; backend tetap menerima seluruh item dari client (`Checks` dinamis) demi kompatibilitas mobile & alur penyelesaian tugas (mode `complete`).
- Jika API template gagal diakses, wizard otomatis memakai `FIELD_CHECKLIST` (9 item default) sebagai fallback agar tidak menghalangi teknisi.
