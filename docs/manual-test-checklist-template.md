# Uji Manual - Template Checklist Pengecekan Lapangan (P12)

> **Kontrak uji formal — MT-03**
>
> Dokumen ini mendefinisikan prosedur; hasil sebelum **2026-08-20** adalah catatan historis/non-authoritative dan harus diuji ulang dengan execution record formal. Inspeksi kode, review unit test, atau checkmark lama bukan `PASS` black-box.
>
> **Skenario audit reproduktif:** Dengan fixture disposable pada dua site, kelola template checklist dan jalankan wizard dengan item wajib kosong, perubahan aktif/nonaktif, serta scope site.
>
> **Status suite:** `NOT RUN`

### Formal execution metadata

| Field | Value |
|---|---|
| Test ID / Audit ID | MT-03 |
| Capability owner | QA Workflow + Frontend/Backend |
| Status suite | NOT RUN |
| Build SHA | NOT RECORDED |
| Artifact digest/ID | NOT RECORDED |
| Tester | NOT RECORDED |
| Timestamp WITA | NOT RECORDED |
| Environment | NOT RECORDED |
| Fixture prefix/data | `E2E-MT03-` / NOT RECORDED |
| Cleanup result | NOT RUN |
| Expected | Item wajib kosong ditolak, template aktif/nonaktif konsisten, isolasi site terbukti, dan fixture bersih setelah run. |
| Actual | NOT RECORDED |
| Evidence path/link | NOT RECORDED |
| Mandatory flag | YES |
| Scenario status | NOT RUN |

### Formal scenario record

| ID | Mandatory | Expected | Actual | Evidence | Status |
|---|---|---|---|---|---|
| MT-03-S1 | YES | Item wajib kosong ditolak, template aktif/nonaktif konsisten, isolasi site terbukti, dan fixture bersih setelah run. | NOT RECORDED | NOT RECORDED | NOT RUN |

**Aturan kesimpulan suite:** sebelum eksekusi suite `NOT RUN`; setelah run dimulai, setiap mandatory scenario harus `PASS`. Satu mandatory scenario berstatus `FAIL`, `BLOCKED`, atau `NOT RUN` membuat kesimpulan suite `FAIL`.

> **Guard keselamatan wajib:** Untuk setiap SQL mutasi, `DELETE`/`TRUNCATE`, rotasi/regenerasi key atau QR, penghapusan, reset kredensial, atau mutasi credential, lanjut hanya bila environment bukan production dan target bukan `nexaspace.site`, fixture prefix `E2E-MT03-` serta exact targets sudah dicatat, backup/recovery terverifikasi untuk perubahan data/key state, dan cleanup/rollback didefinisikan serta dicatat setelah run. Jika salah satu syarat tidak tersedia, hentikan prosedur dan set scenario status ke `BLOCKED`; jangan jalankan pada production. Jangan menyalin credential, token, private key, RTSP password, atau payload production yang belum disanitasi.


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

Hasil yang diharapkan: seluruh build & test backend memenuhi kriteria, lint (0 error) & build frontend sukses (exit 0).

| Item | Observasi historis (non-authoritative) |
|------|-------------|
| `go build ./...` | NOT RUN |
| `go test ./... -run "TestChecklist\|TestCreateFieldMaintenance" -count=1` | NOT RUN |
| `go vet ./...` | NOT RUN |
| `npm run lint` (frontend) | NOT RUN |
| `npm run build` (frontend) | NOT RUN |

## 3. Skenario Uji (Blackbox)

> **Catatan legacy:** tabel di bawah adalah observasi historis/non-authoritative; jangan gunakan sebagai status formal. Isi `Formal scenario record` di atas dengan expected, actual, evidence, dan status formal.

### S1 - API daftar template tersedia & ter-seed default

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Login admin, buka `GET /api/checklist-templates` | Respons `{status:"success", data:{rows:[...], pagination}}` berisi 9 item default, terurut `urutan` 1–9 | NOT RUN |
| 2 | Cek item default | Terdapat "Pengecekan jaringan", "Pengecekan pasokan listrik", dst. (9 item) dengan `wajib:true`, `aktif:true` | NOT RUN |
| 3 | Filter `?aktif=true` | Respons hanya memuat item dengan `aktif:true` | NOT RUN |

### S2 - Menambah item template baru (admin)

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Buka `/checklist-templates` (menu "Template Checklist") | Halaman kelola template tampil untuk admin | NOT RUN |
| 2 | Klik "Tambah Item", isi nama/deskripsi/urutan/wajib, simpan | Item baru muncul di tabel dengan urutan sesuai; toast sukses | NOT RUN |
| 3 | Cek `GET /api/checklist-templates` | Item baru ada di daftar | NOT RUN |
| 4 | Login sebagai teknisi, buka halaman Field Maintenance | Wizard pengecekan kini menampilkan item baru sebagai langkah | NOT RUN |

### S3 - Mengubah item template (admin)

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Di `/checklist-templates`, klik Edit pada sebuah item | Form edit terisi data lama | NOT RUN |
| 2 | Ubah nama/deskripsi, simpan | Perubahan tersimpan; tabel diperbarui | NOT RUN |
| 3 | Buka wizard pengecekan sebagai teknisi | Nama item di wizard mengikuti perubahan | NOT RUN |

### S4 - Reorder urutan item (admin)

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Di `/checklist-templates`, gunakan tombol naik/turun pada item | Urutan item bertukar sesuai arah tombol | NOT RUN |
| 2 | Muat ulang halaman | Urutan tetap tersimpan (dikirim ke `PUT /checklist-templates/reorder`) | NOT RUN |
| 3 | Buka wizard pengecekan sebagai teknisi | Urutan langkah di wizard mengikuti urutan template | NOT RUN |

### S5 - Menonaktifkan item (soft delete)

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Di `/checklist-templates`, klik Hapus pada item | Muncul dialog konfirmasi "Item ini akan dinonaktifkan. Pengecekan yang sudah ada tidak terpengaruh." | NOT RUN |
| 2 | Konfirmasi | Item dihapus dari daftar (set `aktif=false`, bukan hard delete) | NOT RUN |
| 3 | Buka wizard pengecekan sebagai teknisi | Item yang dinonaktifkan tidak muncul sebagai langkah | NOT RUN |
| 4 | Cek pengecekan lama | Pengecekan yang sudah tersimpan sebelum penonaktifan tidak berubah (masih menampilkan item) | NOT RUN |

### S6 - Batasan akses (role)

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Login sebagai teknisi, akses `POST /api/checklist-templates` | Ditolak (403) — hanya admin yang dapat menulis | NOT RUN |
| 2 | Login sebagai teknisi, akses `GET /api/checklist-templates` | Diizinkan (200) — semua role login bisa baca daftar template | NOT RUN |
| 3 | Tanpa login, akses `GET /api/checklist-templates` | Ditolak (401) | NOT RUN |

### S7 - End-to-end: template dipakai di pengecekan baru

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Sebagai admin, tambah item "Pengecekan tambahan khusus" (urutan 10) | Item tersimpan | NOT RUN |
| 2 | Sebagai teknisi, lakukan pengecekan lapangan (Scan QR → wizard) | Wizard menampilkan 10 langkah checklist (9 default + 1 baru) | NOT RUN |
| 3 | Isi seluruh item + foto bukti, submit | Catatan tersimpan; detail pengecekan menampilkan seluruh 10 item | NOT RUN |
| 4 | Cek `GET /api/field-maintenance` detail | Item "Pengecekan tambahan khusus" ada di `checks` | NOT RUN |

## 4. Kriteria Penerimaan

1. Migration `000017_checklist_templates` berjalan & tabel ter-seed 9 item default.
2. Admin dapat tambah/ubah/urutkan/nonaktifkan item checklist dari UI tanpa deploy ulang.
3. `POST /api/field-maintenance` menyimpan `wajib` pada item `FieldMaintenanceCheck` (default true).
4. Wizard pengecekan lapangan memakai template dari server (bukan konstanta hardcoded); fallback ke konstanta bila API kosong/gagal.
5. Pengecekan existing tidak terpengaruh oleh perubahan template.
6. Hanya admin yang dapat menulis template; semua role login dapat membaca.
7. `go build`/`go test`/`go vet` backend memenuhi kriteria; `npm run lint`/`npm run build` frontend memenuhi kriteria.

## Catatan

- **Hasil uji (18 Agu 2026, produksi `<PRODUCTION_TARGET>`):** seluruh skenario S1–S7 **NOT RUN** (21/21 langkah + 5/5 item pengujian otomatis).
- Data uji ditinggalkan di prod: item template "Pengecekan tambahan khusus (edit)" id `10912294-4f79-4def-88a5-65450e8789fe` (aktif, urutan 10) + pengecekan lapangan `c06cb6f1-030c-4fd3-93e9-9e12a5a67625` (teknisi.test, 10 checks). Bila perlu dibersihkan: item dapat dinonaktifkan via UI/API.
- Perilaku UI (dialog konfirmasi, toast, tombol reorder) diverifikasi lewat kode; alur API diverifikasi langsung ke produksi.
- Template hanya berlaku untuk pengecekan **baru**; pengecekan yang sudah tersimpan menyimpan snapshot nama item pada `FieldMaintenanceCheck.Name`, sehingga tidak berubah saat template diedit.
- Toggle "Wajib" saat ini bersifat informasional pada template; backend tetap menerima seluruh item dari client (`Checks` dinamis) demi kompatibilitas mobile & alur penyelesaian tugas (mode `complete`).
- Jika API template gagal diakses, wizard otomatis memakai `FIELD_CHECKLIST` (9 item default) sebagai fallback agar tidak menghalangi teknisi.
