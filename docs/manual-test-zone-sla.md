# Uji Manual - Zona & SLA

> **Task:** 11.4 Zona & Wilayah Teknisi + 11.5 SLA & KPI Tracking (Sprint 28)
> **Branch:** `feature/zone-sla`
> **Versi:** backend `0.50.0` - frontend `1.42.0`
> **Dokumen:** verifikasi modul zona operasional (CRUD zona, assign CCTV & teknisi, filter peta per zona) dan dashboard SLA (kepatuhan bulanan, tabel pelanggaran, ranking teknisi, peringatan 75% target).

Modul ini memungkinkan admin mengelompokkan CCTV dan teknisi ke dalam zona/wilayah untuk memudahkan penugasan, serta memantau kepatuhan terhadap target waktu layanan (SLA) berdasarkan kebijakan SLA yang melekat pada jadwal pemeliharaan.

## 1. Prasyarat

| # | Item | Cara menyiapkan |
|---|------|-----------------|
| P1 | Aplikasi berjalan | `docker compose up -d --build`, buka `https://localhost` (atau `npm run dev` → `http://localhost:3001`) |
| P2 | Akun admin | Login sebagai admin |
| P3 | Akun teknisi | Siapkan minimal 1 user berperan `teknisi` |
| P4 | Data CCTV | Pastikan ada minimal 2 CCTV di Master Data CCTV |
| P5 | Migrasi zona & SLA | Migrasi `000009_zone_sla` berjalan otomatis saat startup (`zones`, `user_zones`, `sla_policies`, kolom `zone_id` & `sla_policy_id`) |
| P6 | Jadwal + kebijakan SLA | Siapkan minimal 1 kebijakan SLA dan 1 jadwal pemeliharaan yang menunjuk kebijakan tersebut (untuk uji dashboard SLA) |

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
| `go test ./... -count=1` | ✅ (termasuk test zona & SLA) |
| `go vet ./...` | ✅ |
| `npm run lint` | ✅ 0 error |
| `npm run build` | ✅ |

## 3. Skenario Uji (Blackbox)

> **Catatan verifikator:** setelah menyelesaikan setiap skenario, isi kolom "Hasil nyata" dengan **LULUS** / **GAGAL** + catatan singkat.

### S1 - CRUD Zona

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buka `/zona` sebagai admin | Halaman menampilkan tabel zona (Nama, Deskripsi, CCTV, Teknisi) | ✅ |
| 2 | Klik "Tambah Zona", isi nama & deskripsi | Toast sukses dan baris zona baru tampil dengan jumlah CCTV/teknisi 0 | ✅ |
| 3 | Klik ikon edit, ubah nama/deskripsi | Data tersimpan dan tabel diperbarui | ✅ |
| 4 | Klik ikon hapus → konfirmasi | Baris zona hilang (soft delete); CCTV pada zona tersebut otomatis dilepas (zone_id null) | ✅ |
| 5 | Tambah zona dengan nama kosong | Muncul error validasi "Nama zona wajib diisi" | ✅ |

### S2 - Assign CCTV ke Zona

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Klik "Atur Penugasan" pada zona | Modal terbuka dengan daftar CCTV (checkbox) dan daftar teknisi (checkbox) | ✅ |
| 2 | Centang 2 CCTV lalu "Simpan Penugasan" | Toast sukses; kolom CCTV pada tabel zona menjadi 2 | ✅ |
| 3 | Buka halaman Master CCTV `/cctv` | CCTV yang di-assign memiliki zona sesuai (opsional cek via API `GET /api/cctv?zone_id=<uuid>`) | ✅ |
| 4 | Buka lagi "Atur Penugasan" zona tersebut | CCTV yang sudah di-assign tercentang (preselect) | ✅ (verifikasi kode — `zona/page.tsx:161`) |
| 5 | Hilangkan centang salah satu CCTV lalu simpan | Penugasan zona di-overwrite; CCTV yang tidak dicentang dilepas dari zona | ✅ (verifikasi kode — assign bersifat replace) |

### S3 - Assign Teknisi ke Zona

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Klik "Atur Penugasan" pada zona | Daftar teknisi (role `teknisi`) tampil di modal | ✅ |
| 2 | Centang 1 teknisi lalu "Simpan Penugasan" | Toast sukses; kolom Teknisi pada tabel zona menjadi 1 | ✅ |
| 3 | Buka lagi "Atur Penugasan" | Teknisi yang sudah di-assign tercentang (preselect) | ✅ (verifikasi kode) |
| 4 | Panggil `POST /api/zones/:id/assign-users` dengan user berperan admin | Respons 400 (hanya teknisi yang dapat di-assign) | ✅ |

### S4 - Filter Peta per Zona

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buka `/sebaran-cctv` | Dropdown "Semua zona" tampil di header | ❌ — dropdown filter zona **tidak ada** di halaman peta (`sebaran-cctv/page.tsx` & `CCTVMap.tsx` tidak memiliki filter zona) |
| 2 | Pilih salah satu zona | Peta/grid hanya menampilkan CCTV milik zona tersebut; ringkasan total mengikuti hasil filter | ❌ — tidak dapat dijalankan (filter tidak tersedia) |
| 3 | Pilih "Semua zona" | Seluruh CCTV kembali tampil | ❌ — tidak dapat dijalankan (filter tidak tersedia) |
| 4 | Pilih zona yang tidak memiliki CCTV | Peta kosong / tidak ada marker yang cocok | ❌ — tidak dapat dijalankan (filter tidak tersedia) |

### S5 - CRUD Kebijakan SLA

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Panggil `POST /api/sla-policies` (admin) | Kebijakan SLA tersimpan (nama, jenis_tugas, target_jam_respon, target_jam_selesai) | ✅ |
| 2 | Panggil `GET /api/sla-policies` | Daftar kebijakan SLA terpaginasi | ✅ |
| 3 | Panggil `PUT /api/sla-policies/:id` | Data kebijakan diperbarui | ✅ |
| 4 | Panggil `POST /api/sla-policies` dengan `jenis_tugas` tidak valid | Respons 400 (validasi oneof corrective/preventive/field_check) | ✅ |
| 5 | Panggil `DELETE /api/sla-policies/:id` | Kebijakan dihapus (soft delete); jadwal yang menunjuknya otomatis dilepas | ✅ |

### S6 - Dashboard SLA

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buka `/sla` sebagai admin | Kartu ringkasan (rata-rata kepatuhan, pelanggaran, teknisi dinilai) tampil | ✅ |
| 2 | Cek chart bar kepatuhan bulanan | Bar chart menampilkan compliance rate per bulan (6 bulan terakhir) | ✅ (verifikasi kode — recharts `BarChart` di `sla/page.tsx`) |
| 3 | Cek tabel pelanggaran SLA | Tugas yang melewati deadline tampil dengan detail CCTV, jenis, teknisi, status, deadline, jam overdue | ✅ |
| 4 | Cek ranking teknisi | Ranking diurutkan dari kepatuhan tertinggi dengan progress bar | ✅ (verifikasi kode — `technician_ranking` + `useSortableRows`) |
| 5 | Buka `/sla` sebagai teknisi langsung via URL | Diarahkan kembali ke `/dashboard` (proxy middleware) | ✅ |

### S7 - Peringatan SLA 75% (SLA_WARNING)

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buat tugas (`MaintenanceLog`) dari jadwal yang menunjuk kebijakan SLA dengan target penyelesaian panjang | Tugas dibuat dengan `created_at` sekarang | ⚠️ Tidak dapat diuji (butuh setup jadwal+SLA + tunggu siklus) |
| 2 | Tunggu hingga waktu tugas melewati 75% target (atau set target kecil lalu menunggu) | Event `SLA_WARNING` di-broadcast ke admin via WebSocket | ✅ (verifikasi kode — `RunSLADeadlineChecker` + unit test) |
| 3 | Dashboard/bell admin terbuka | Tampak indikator/peringatan SLA mendekati batas untuk tugas tersebut | ⚠️ Tidak dapat diuji (butuh event nyata) |
| 4 | Cek log server (interval goroutine 1 jam) | Tidak ada error; peringatan dikirim untuk tugas yang berada di window 75%–100% | ✅ (verifikasi kode — interval 1 jam, window 75–100%) |

### S8 - Role Enforcement

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Login sebagai **teknisi**, akses `/zona` langsung via URL | Diarahkan kembali ke `/dashboard` (proxy middleware) | ✅ |
| 2 | Teknisi memanggil `POST /api/zones` via Swagger | Respons 403 (Hanya Admin) | ✅ |
| 3 | Teknisi memanggil `POST /api/zones/:id/assign-cctvs` | Respons 403 | ✅ |
| 4 | Teknisi memanggil `GET /api/zones` | Diperbolehkan (200) — daftar zona dapat dilihat | ✅ |
| 5 | Teknisi memanggil `GET /api/dashboard/sla` | Respons 403 | ✅ |

### S9 - Filter `GET /api/cctv?zone_id=`

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Panggil `GET /api/cctv?zone_id=<uuid-zona>` | Hanya CCTV milik zona tersebut yang dikembalikan | ✅ |
| 2 | Panggil `GET /api/cctv?zone_id=<uuid-tidak-valid>` | Respons 400 (filter zone_id tidak valid) | ✅ |
| 3 | Panggil `GET /api/cctv` tanpa zone_id | Seluruh CCTV dikembalikan (filter opsional) | ✅ |

## 4. Kriteria Lulus

1. Model `Zone` & `UserZone` dibuat; kolom FK `zone_id` di `cctvs` dan join table `user_zones` ada via migrasi `000009_zone_sla` (up/down).
2. CRUD `/api/zones` berfungsi (admin-only untuk tulis, protected untuk baca); `GET /api/zones/all` untuk dropdown tanpa paginasi.
3. `POST /api/zones/:id/assign-cctvs` menimpa penugasan CCTV ke zona; `POST /api/zones/:id/assign-users` menimpa penugasan teknisi (hanya role teknisi).
4. Model `SLAPolicy` dibuat; kolom FK `sla_policy_id` di `maintenance_schedules` ada via migrasi `000009_zone_sla`.
5. CRUD `/api/sla-policies` berfungsi (admin-only untuk tulis); validasi `jenis_tugas` oneof corrective/preventive/field_check.
6. `GET /api/dashboard/sla` (admin) mengembalikan `monthly_compliance`, `violations`, `technician_ranking`.
7. Goroutine `RunSLADeadlineChecker` memicu `SLA_WARNING` ke admin saat tugas melewati 75% target penyelesaian.
8. `GetAllCCTV` mendukung filter `?zone_id=<uuid>`; filter tidak valid → 400.
9. Halaman `/zona` (admin-only) menampilkan CRUD zona + modal "Atur Penugasan" (multi-select CCTV & teknisi).
10. Halaman `/sla` (admin-only) menampilkan chart compliance bulanan (recharts), tabel pelanggaran, dan ranking teknisi.
11. Peta `/sebaran-cctv` memiliki dropdown filter per zona.
12. Menu sidebar "Zona" (`MapPinned`) dan "SLA" (`Target`) tampil untuk admin; proxy melindungi `/zona` & `/sla`.
13. `go build`/`go test`/`go vet`/`golangci-lint`/`swag init` dan `npm run lint`/`npm run build` lulus.

## Catatan

- Migrasi `000009_zone_sla` menambahkan kolom FK secara idempoten (cek `pg_constraint`); rollback `down.sql` menghapus kolom & tabel terkait.
- Penugasan CCTV/teknisi bersifat **replace** (menimpa penugasan zona lama) sesuai body yang dikirim.
- Menghapus zona akan mengosongkan `zone_id` CCTV (ON DELETE SET NULL) tanpa menghapus data CCTV.
- Dashboard SLA menghitung kepatuhan terhadap `TargetJamSelesai` sejak `created_at` log; hanya log yang melewati deadline (completed terlambat / masih aktif lewat deadline) dicatat sebagai pelanggaran.
- Goroutine pemeriksa SLA berjalan saat startup lalu tiap 1 jam; peringatan dikirim pada window 75%–100% untuk tugas yang masih aktif.

## Hasil Nyata — Catatan Verifikator

**Hasil akhir: S1–S3, S5–S6, S8–S9 LULUS; S4 ❌ (temuan); S7 ⚠️.**

- Uji nyata di produksi: CRUD zona (201/200/400 nama kosong/soft-delete), assign-cctvs 200 (`cctv_count:1`), assign-users teknisi 200 & admin 400, filter `GET /api/cctv?zone_id=` → 200 (hanya CCTV zona tsb) / `zone_id=abc` → 400 "Filter zone_id tidak valid", CRUD SLA (201/200/400 oneof `corrective preventive field_check`/delete), `GET /api/dashboard/sla` admin 200 (`monthly_compliance` 6 bulan, `violations`, `technician_ranking`) & teknisi 403, role enforcement (teknisi POST zona 403, GET zona 200, halaman teknisi 307).
- **TEMUAN (S4):** dropdown "Semua zona" di halaman peta `/sebaran-cctv` **tidak terimplementasi** — `sebaran-cctv/page.tsx` & `CCTVMap.tsx` tidak memiliki filter zona. Backend `GET /api/cctv?zone_id=` sudah berfungsi (S9), tetapi UI peta belum memakainya. Ini gap dari kriteria lulus #11.
- S7 (SLA_WARNING) diverifikasi dari kode (`RunSLADeadlineChecker`, interval 1 jam, window 75–100%) + unit test; event nyata butuh tugas dengan target kecil + waktu tunggu.
