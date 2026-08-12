# Uji Manual - Multi-Site (Task 12.2)

> **Task:** 12.2 Multi-Site
> **Branch:** `feature/multi-site`
> **Versi:** monorepo `1.28.41`
> **Dokumen:** verifikasi migrasi default site OIKN, CRUD site, assign teknisi ke site, site switcher di UI, isolasi data per site, endpoint daftar site, dan role access.

Scope implementasi: **multi-site** (beberapa kawasan operasional) dengan **site default "Kawasan OIKN"**. Data CCTV, maintenance log, dan zona existing dimigrasikan ke site OIKN saat startup. Pengguna dapat berpindah site via dropdown di header; seluruh data difilter sesuai site aktif (disimpan di cookie `sigap_site_id`).

## 1. Prasyarat

| # | Item | Cara menyiapkan |
|---|------|-----------------|
| P1 | Aplikasi berjalan | `docker compose up -d --build`, buka `https://localhost` (atau `npm run dev` di `frontend/` → `http://localhost:3001`) |
| P2 | Database ter-migrasi | Pastikan migrasi multi-site (kolom `sites`, FK `site_id`) sudah diterapkan saat startup; semua data existing menunjuk ke site "Kawasan OIKN" |
| P3 | Akun admin | Login sebagai admin (untuk CRUD site, assign teknisi, dan verifikasi role access) |
| P4 | Akun teknisi | Login sebagai teknisi (untuk verifikasi role access baca-only) |
| P5 | Data CCTV/log/zona existing | Minimal ada beberapa CCTV, maintenance log, dan zona agar verifikasi migrasi & isolasi data bisa dilakukan |

## 2. Pengujian Otomatis

```powershell
cd backend
go build ./... && go test ./... -count=1 && go vet ./...
```

Hasil yang diharapkan: `go build` sukses (exit 0), `go test ./... -count=1` semua package hijau (termasuk `handlers/site_test.go` & `middleware/site_test.go` untuk Task 12.2), dan `go vet` tidak melaporkan error. Catatan: beberapa `warning` lint lama yang sudah ada tidak memengaruhi lulus/tidaknya.

Verifikasi tambahan (backend & frontend bila tersedia):

```powershell
cd frontend
npm run lint && npm run build
```

## 3. Skenario Uji (Blackbox)

> **Catatan verifikator:** setelah menyelesaikan setiap skenario, isi kolom "Hasil nyata" dengan **LULUS** / **GAGAL** + catatan singkat.

### S1 - Migrasi default site OIKN

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Jalankan aplikasi & biarkan migrasi berjalan saat startup | Migrasi sukses; tabel `sites` dibuat berisi satu baris "Kawasan OIKN" dengan kode `OIKN` dan `is_default=true` | |
| 2 | Cek tabel `sites` di database (via `psql`/adminer) | Terdapat baris `Kawasan OIKN` (kode `OIKN`, `is_default=true`) | |
| 3 | Cek seluruh CCTV existing | Semua CCTV yang ada sebelum migrasi memiliki `site_id` = ID site OIKN | |
| 4 | Cek seluruh maintenance log existing | Semua maintenance log yang ada sebelum migrasi memiliki `site_id` = ID site OIKN | |
| 5 | Cek seluruh zona existing | Semua zona yang ada sebelum migrasi memiliki `site_id` = ID site OIKN | |
| 6 | Buka UI dan login | Halaman termuat normal; site aktif default adalah "Kawasan OIKN" | |

### S2 - CRUD site (admin)

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Login sebagai admin, buka halaman manajemen site | Daftar site tampil (termasuk "Kawasan OIKN") | |
| 2 | Buat site baru (mis. "Kawasan Balikpapan", kode `BPN`) via form/API `POST /api/sites` | Site baru tersimpan; muncul di daftar site; `is_default=false` | |
| 3 | Edit site baru (ubah nama/alamat/koordinat) via `PUT /api/sites/:id` | Perubahan tersimpan; daftar site ter-refresh | |
| 4 | Soft-delete site baru via `DELETE /api/sites/:id` | Site berpindah status soft-delete (tidak tampil di list normal; `deleted_at` terisi di DB) | |
| 5 | Coba hapus site default "Kawasan OIKN" | Permintaan ditolak (4xx) — site default **tidak bisa dihapus**; pesan error jelas | |

### S3 - Assign teknisi ke site

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Pastikan ada user ber-role `teknisi` | Teknisi tersedia di daftar user | |
| 2 | Panggil `POST /api/sites/:id/assign-users` dengan body berisi ID teknisi | Teknisi berhasil di-assign ke site; relasi user↔site tersimpan | |
| 3 | Coba assign user ber-role `admin` (bukan teknisi) | Permintaan ditolak (400) — hanya role `teknisi` yang bisa di-assign | |
| 4 | Login sebagai teknisi yang telah di-assign | Teknisi dapat mengakses site tersebut; data site tampil sesuai relasi | |

### S4 - Switch site di UI

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Login sebagai admin/teknisi | Terdapat dropdown **site switcher** di header (menampilkan site aktif saat ini) | |
| 2 | Klik dropdown, pilih site lain (mis. site yang baru dibuat) | Cookie `sigap_site_id` tersimpan di browser dengan ID site terpilih | |
| 3 | Amati perilaku setelah pilihan site | Halaman di-`reload`; site aktif di header berubah | |
| 4 | Amati data yang ditampilkan | Data (CCTV/log/zona) difilter sesuai site aktif terpilih | |
| 5 | Muat ulang halaman manual (F5) | Site aktif tetap dipertahankan (dibaca dari cookie `sigap_site_id`) | |

### S5 - Isolasi data per site

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Pastikan ada ≥ 2 site dengan data berbeda (mis. site OIKN & site BPN) | Kedua site memiliki CCTV/log sendiri-sendiri | |
| 2 | Pilih site A di dropdown, buka daftar CCTV | CCTV site A tampil | |
| 3 | Catat kode CCTV yang terlihat | Ada minimal 1 CCTV milik site A | |
| 4 | Pilih site B di dropdown, buka daftar CCTV | CCTV site B tampil; **CCTV site A TIDAK tampil** | |
| 5 | Ulangi untuk maintenance log & zona | Log/zona site A tidak tampil saat memilih site B (isolasi data berjalan) | |

### S6 - Endpoint daftar site

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Panggil `GET /api/sites` (login) | Respons berisi daftar site **terpaginasi** (`{ rows, pagination }`); site default tampil | |
| 2 | Uji paginasi (`?page=2&limit=1`) | `total_rows`/`total_pages` sesuai; page 2 kosong bila hanya 1 site | |
| 3 | Panggil `GET /api/sites/all` (login) | Respons berisi seluruh site (tanpa paginasi) — cocok untuk dropdown site switcher | |
| 4 | Cek respons `GET /api/sites/all` | Site default "Kawasan OIKN" ada di daftar; field `is_default` tersedia | |

### S7 - Role access

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Login sebagai admin | Dapat **menulis** site (CRUD, assign-users) — sukses | |
| 2 | Login sebagai teknisi | Dapat **membaca** daftar site (`GET /api/sites`, `GET /api/sites/all`) — sukses | |
| 3 | Login sebagai teknisi, coba buat/ubah/hapus site | Permintaan ditolak (403) — hanya admin yang bisa menulis site | |
| 4 | Login sebagai teknisi, coba `POST /api/sites/:id/assign-users` | Permintaan ditolak (403) — assign teknisi hanya untuk admin | |
| 5 | Akses endpoint site tanpa login | Permintaan ditolak (401) — endpoint site wajib autentikasi | |

## 4. Kriteria Lulus

1. Migrasi berhasil: tabel `sites` berisi "Kawasan OIKN" (kode `OIKN`, `is_default=true`); semua CCTV/log/zona existing memiliki `site_id` = site OIKN.
2. CRUD site admin berfungsi (create/edit/soft-delete), dan site default **tidak dapat dihapus**.
3. `POST /api/sites/:id/assign-users` berhasil menugaskan teknisi; user non-teknisi ditolak (400).
4. Site switcher di header menyimpan cookie `sigap_site_id`, me-reload halaman, dan memfilter data sesuai site aktif.
5. Isolasi data per site berjalan: data site A tidak tampil saat memilih site B.
6. `GET /api/sites` terpaginasi dan `GET /api/sites/all` (untuk dropdown) tersedia.
7. Role access benar: hanya admin yang bisa menulis site; semua role login bisa membaca.
8. `go build`/`go test ./... -count=1`/`go vet` lulus; `npm run lint` & `npm run build` lulus (bila frontend tersedia).

## Catatan

- Site default "Kawasan OIKN" bersifat read-protected untuk penghapusan (tidak bisa dihapus) demi menjaga integritas data warisan.
- Cookie `sigap_site_id` dipakai untuk menyimpan site aktif lintas halaman; pastikan cookie dikirim bersama permintaan (same-origin) agar filter global konsisten.
- Endpoint `GET /api/sites` (paginasi) digunakan untuk halaman manajemen; `GET /api/sites/all` (tanpa paginasi) untuk dropdown site switcher di header.
- Scope penulisan site (CRUD & assign-users) dibatasi admin via `RequireRole("admin")`; pembacaan tersedia untuk semua role login.
- Catatan gotchas test `site_test.go` (handlers) & `middleware/site_test.go` (bila tersedia): pola sqlmock sama dengan test CRUD lain — `SetupTestDB` memakai `SkipDefaultTransaction: true`; `LogAudit`/broadcast ditulis async di goroutine sehingga jangan mem-mock query user/audit pada test success; operasi transaksi (assign-users) memerlukan `mock.ExpectBegin()`/`mock.ExpectCommit()`.
