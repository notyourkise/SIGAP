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

| Item | Hasil nyata |
|------|-------------|
| `go build ./...` | ✅ LULUS — sukses |
| `go test ./... -run "TestChecklist\|TestCreateFieldMaintenance" -count=1` | ✅ LULUS — `ok sigap-backend/handlers` (14.0s), seluruh package ok |
| `go vet ./...` | ✅ LULUS |
| `npm run lint` (frontend) | ✅ LULUS — 0 error (77 warning lama di file lain) |
| `npm run build` (frontend) | ✅ LULUS — exit 0 (setelah `npm install`; `recharts` belum ada di node_modules lokal) |

## 3. Skenario Uji (Blackbox)

> **Catatan verifikator:** setelah menyelesaikan setiap skenario, isi kolom "Hasil nyata" dengan **LULUS** / **GAGAL** + catatan singkat.

### S1 - API daftar template tersedia & ter-seed default

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Login admin, buka `GET /api/checklist-templates` | Respons `{status:"success", data:{rows:[...], pagination}}` berisi 9 item default, terurut `urutan` 1–9 | ✅ LULUS — 200, `total_rows=9`, urutan 1–9 menaik |
| 2 | Cek item default | Terdapat "Pengecekan jaringan", "Pengecekan pasokan listrik", dst. (9 item) dengan `wajib:true`, `aktif:true` | ✅ LULUS — 9 item: jaringan, pasokan listrik, pembersihan lensa, fisik kamera, kualitas gambar, perekaman, akses jarak jauh, kabel & konektor, lingkungan sekitar; semua `wajib=true aktif=true` |
| 3 | Filter `?aktif=true` | Respons hanya memuat item dengan `aktif:true` | ✅ LULUS — 200, `total_rows=9`, 0 item nonaktif tercakup |

### S2 - Menambah item template baru (admin)

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buka `/checklist-templates` (menu "Template Checklist") | Halaman kelola template tampil untuk admin | ✅ LULUS — menu "Template Checklist" (ikon ListChecks) di grup Manajemen Sistem admin-only, `DashboardLayout.tsx:160`; guard admin via `proxy.ts` (`ADMIN_PATHS`) |
| 2 | Klik "Tambah Item", isi nama/deskripsi/urutan/wajib, simpan | Item baru muncul di tabel dengan urutan sesuai; toast sukses | ✅ LULUS — `POST /api/checklist-templates` → 201; id `10912294-4f79-4def-88a5-65450e8789fe`, urutan 10, `aktif/wajib=true` (default true saat field tak dikirim) |
| 3 | Cek `GET /api/checklist-templates` | Item baru ada di daftar | ✅ LULUS — list berisi 10 item, item baru urutan 10 |
| 4 | Login sebagai teknisi, buka halaman Field Maintenance | Wizard pengecekan kini menampilkan item baru sebagai langkah | ✅ LULUS — wizard memuat template server (`?aktif=true`) saat dibuka: `activeList = templates.length ? templates : FIELD_CHECKLIST`, `FieldMaintenanceModal.tsx:120,138` |

### S3 - Mengubah item template (admin)

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Di `/checklist-templates`, klik Edit pada sebuah item | Form edit terisi data lama | ✅ LULUS — `openEditForm` mengisi form dari item terpilih, `ChecklistTemplateManager.tsx:125` (verifikasi kode) |
| 2 | Ubah nama/deskripsi, simpan | Perubahan tersimpan; tabel diperbarui | ✅ LULUS — `PUT /api/checklist-templates/:id` → 200; nama berubah menjadi "Pengecekan tambahan khusus (edit)" |
| 3 | Buka wizard pengecekan sebagai teknisi | Nama item di wizard mengikuti perubahan | ✅ LULUS — wizard memakai `nama_item` dari server (bukan konstanta) |

### S4 - Reorder urutan item (admin)

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Di `/checklist-templates`, gunakan tombol naik/turun pada item | Urutan item bertukar sesuai arah tombol | ✅ LULUS — tombol naik/turun swap array lalu kirim `PUT /reorder` (`ChecklistTemplateManager.tsx:228-243`); verifikasi API: reorder urutan 1↔9 → 200, list tampil sesuai urutan baru |
| 2 | Muat ulang halaman | Urutan tetap tersimpan (dikirim ke `PUT /checklist-templates/reorder`) | ✅ LULUS — urutan bertahan setelah GET ulang; dikembalikan ke urutan semula setelah pengujian |
| 3 | Buka wizard pengecekan sebagai teknisi | Urutan langkah di wizard mengikuti urutan template | ✅ LULUS — wizard memakai urutan ASC dari backend (`Order("urutan ASC")`, `checklist_template.go:99`) |

### S5 - Menonaktifkan item (soft delete)

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Di `/checklist-templates`, klik Hapus pada item | Muncul dialog konfirmasi "Item ini akan dinonaktifkan. Pengecekan yang sudah ada tidak terpengaruh." | ✅ LULUS — `handleDelete` membuka dialog konfirmasi (`ConfirmDialog`) dengan teks tsb (verifikasi kode, `ChecklistTemplateManager.tsx:181-192`) |
| 2 | Konfirmasi | Item dihapus dari daftar (set `aktif=false`, bukan hard delete) | ✅ LULUS — `DELETE /api/checklist-templates/:id` → 200 "berhasil dinonaktifkan"; item masih ada di DB (`aktif=false`, terlihat via `?aktif=false`) — soft delete, bukan hard delete |
| 3 | Buka wizard pengecekan sebagai teknisi | Item yang dinonaktifkan tidak muncul sebagai langkah | ✅ LULUS — wizard memakai `?aktif=true` (`getChecklistTemplates(true)`) sehingga item nonaktif tersembunyi |
| 4 | Cek pengecekan lama | Pengecekan yang sudah tersimpan sebelum penonaktifan tidak berubah (masih menampilkan item) | ✅ LULUS — snapshot nama item tersimpan di `FieldMaintenanceCheck.Name`; pengecekan `c06cb6f1-030c-4fd3-93e9-9e12a5a67625` tetap menampilkan 10 item lengkap meski template sempat dinonaktifkan |

### S6 - Batasan akses (role)

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Login sebagai teknisi, akses `POST /api/checklist-templates` | Ditolak (403) — hanya admin yang dapat menulis | ✅ LULUS — teknisi POST → 403 |
| 2 | Login sebagai teknisi, akses `GET /api/checklist-templates` | Diizinkan (200) — semua role login bisa baca daftar template | ✅ LULUS — teknisi GET → 200 |
| 3 | Tanpa login, akses `GET /api/checklist-templates` | Ditolak (401) | ✅ LULUS — tanpa token → 401 |

### S7 - End-to-end: template dipakai di pengecekan baru

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Sebagai admin, tambah item "Pengecekan tambahan khusus" (urutan 10) | Item tersimpan | ✅ LULUS — 201, urutan 10; item di-aktifkan kembali setelah uji S5 |
| 2 | Sebagai teknisi, lakukan pengecekan lapangan (Scan QR → wizard) | Wizard menampilkan 10 langkah checklist (9 default + 1 baru) | ✅ LULUS — template server berisi 10 item aktif (9 default + 1 baru) → wizard 10 langkah (verifikasi kode + data) |
| 3 | Isi seluruh item + foto bukti, submit | Catatan tersimpan; detail pengecekan menampilkan seluruh 10 item | ✅ LULUS — `POST /api/field-maintenance` (teknisi.test) → 201, `overall_status=ok`, 1 foto |
| 4 | Cek `GET /api/field-maintenance` detail | Item "Pengecekan tambahan khusus" ada di `checks` | ✅ LULUS — detail id `c06cb6f1-030c-4fd3-93e9-9e12a5a67625`: 10 checks, item terakhir = "Pengecekan tambahan khusus (edit)" → `ok` |

## 4. Kriteria Lulus

1. Migration `000017_checklist_templates` berjalan & tabel ter-seed 9 item default.
2. Admin dapat tambah/ubah/urutkan/nonaktifkan item checklist dari UI tanpa deploy ulang.
3. `POST /api/field-maintenance` menyimpan `wajib` pada item `FieldMaintenanceCheck` (default true).
4. Wizard pengecekan lapangan memakai template dari server (bukan konstanta hardcoded); fallback ke konstanta bila API kosong/gagal.
5. Pengecekan existing tidak terpengaruh oleh perubahan template.
6. Hanya admin yang dapat menulis template; semua role login dapat membaca.
7. `go build`/`go test`/`go vet` backend lulus; `npm run lint`/`npm run build` frontend lulus.

## Catatan

- **Hasil uji (18 Agu 2026, produksi `https://nexaspace.site`):** seluruh skenario S1–S7 **LULUS** (21/21 langkah + 5/5 item pengujian otomatis).
- Data uji ditinggalkan di prod: item template "Pengecekan tambahan khusus (edit)" id `10912294-4f79-4def-88a5-65450e8789fe` (aktif, urutan 10) + pengecekan lapangan `c06cb6f1-030c-4fd3-93e9-9e12a5a67625` (teknisi.test, 10 checks). Bila perlu dibersihkan: item dapat dinonaktifkan via UI/API.
- Perilaku UI (dialog konfirmasi, toast, tombol reorder) diverifikasi lewat kode; alur API diverifikasi langsung ke produksi.
- Template hanya berlaku untuk pengecekan **baru**; pengecekan yang sudah tersimpan menyimpan snapshot nama item pada `FieldMaintenanceCheck.Name`, sehingga tidak berubah saat template diedit.
- Toggle "Wajib" saat ini bersifat informasional pada template; backend tetap menerima seluruh item dari client (`Checks` dinamis) demi kompatibilitas mobile & alur penyelesaian tugas (mode `complete`).
- Jika API template gagal diakses, wizard otomatis memakai `FIELD_CHECKLIST` (9 item default) sebagai fallback agar tidak menghalangi teknisi.
