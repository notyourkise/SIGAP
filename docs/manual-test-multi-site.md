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

| Item | Hasil nyata |
|------|-------------|
| `go build ./...` | ✅ LULUS |
| `go test ./... -count=1` (subset relevan) | ✅ LULUS — `ok sigap-backend/handlers 6.286s` (termasuk `site_test.go`), `ok sigap-backend/utils`, `ok sigap-backend/middleware 12.046s` (termasuk `site_test.go`: header valid/invalid, query fallback, site not found) |
| `go vet ./...` | ✅ LULUS |
| `npm run lint` / `npm run build` | ✅ LULUS — 0 error lint; build exit 0 (hasil sesi sebelumnya, tanpa perubahan kode frontend) |

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
| 1 | Jalankan aplikasi & biarkan migrasi berjalan saat startup | Migrasi sukses; tabel `sites` dibuat berisi satu baris "Kawasan OIKN" dengan kode `OIKN` dan `is_default=true` | LULUS — `GET /api/sites` produksi: total=1, `Kawasan OIKN (OIKN) default=True`; migrasi `000018_multi_site.up.sql` menyisipkan site default OIKN (kode `OIKN`, `is_default=true`) |
| 2 | Cek tabel `sites` di database (via `psql`/adminer) | Terdapat baris `Kawasan OIKN` (kode `OIKN`, `is_default=true`) | LULUS — terverifikasi via API (`/api/sites` → id `5759393a-c61d-49f9-8d58-f1ebc44b8bb8`, `is_default=true`); cek psql langsung tidak tersedia (VPS tanpa SSH non-interaktif) |
| 3 | Cek seluruh CCTV existing | Semua CCTV yang ada sebelum migrasi memiliki `site_id` = ID site OIKN | LULUS — migrasi `000018` menetapkan site default ke semua data existing (idempoten, per changelog Task 12.2); ⚠️ field `site_id` tidak diekspos di respons `GET /api/cctv` (JSON `-`), sehingga verifikasi langsung via API tidak mungkin — dikonfirmasi dari kode migrasi & model |
| 4 | Cek seluruh maintenance log existing | Semua maintenance log yang ada sebelum migrasi memiliki `site_id` = ID site OIKN | LULUS — sama seperti #3: kolom `site_id` ditambahkan ke `maintenance_logs` + diisi site default saat migrasi (changelog Task 12.2); tidak diekspos di respons API |
| 5 | Cek seluruh zona existing | Semua zona yang ada sebelum migrasi memiliki `site_id` = ID site OIKN | LULUS — kolom `site_id` ditambahkan ke `zones` + diisi site default saat migrasi (changelog Task 12.2) |
| 6 | Buka UI dan login | Halaman termuat normal; site aktif default adalah "Kawasan OIKN" | LULUS — halaman produksi termuat (200); `SiteSwitcher` fallback ke site default bila cookie `sigap_site_id` kosong (`frontend/src/components/SiteSwitcher.tsx:7-15`) |

### S2 - CRUD site (admin)

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Login sebagai admin, buka halaman manajemen site | Daftar site tampil (termasuk "Kawasan OIKN") | LULUS — `GET /api/sites` produksi: 1 site "Kawasan OIKN" |
| 2 | Buat site baru (mis. "Kawasan Balikpapan", kode `BPN`) via form/API `POST /api/sites` | Site baru tersimpan; muncul di daftar site; `is_default=false` | LULUS — nyata: `POST /api/sites {"nama":"Kawasan Balikpapan","kode":"BPN",...}` → `[201] success`, id `0157e5fa...`, `is_default=false` |
| 3 | Edit site baru (ubah nama/alamat/koordinat) via `PUT /api/sites/:id` | Perubahan tersimpan; daftar site ter-refresh | LULUS — nyata: `PUT` → `[200]`, nama berubah menjadi "Kawasan Balikpapan Raya" |
| 4 | Soft-delete site baru via `DELETE /api/sites/:id` | Site berpindah status soft-delete (tidak tampil di list normal; `deleted_at` terisi di DB) | LULUS — nyata: `DELETE` → `[200] "Site berhasil dihapus"`; total site kembali 1 (soft-delete, model `Site` memakai `Base`/`DeletedAt`) |
| 5 | Coba hapus site default "Kawasan OIKN" | Permintaan ditolak (4xx) — site default **tidak bisa dihapus**; pesan error jelas | LULUS — nyata: `DELETE /api/sites/5759393a-...` (OIKN) → `ERR:400` (ditolak) |

### S3 - Assign teknisi ke site

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Pastikan ada user ber-role `teknisi` | Teknisi tersedia di daftar user | LULUS — teknisi B (`f4456640-7869-401a-830f-5e0045f99407`, "Teknisi Dua") tersedia |
| 2 | Panggil `POST /api/sites/:id/assign-users` dengan body berisi ID teknisi | Teknisi berhasil di-assign ke site; relasi user↔site tersimpan | LULUS — nyata: `[200] {"site_id":"8b43d409-...","user_count":1,"message":"Teknisi berhasil ditugaskan ke site"}` |
| 3 | Coba assign user ber-role `admin` (bukan teknisi) | Permintaan ditolak (400) — hanya role `teknisi` yang bisa di-assign | LULUS — nyata: assign admin (`663fd649-...`) → `ERR:400` |
| 4 | Login sebagai teknisi yang telah di-assign | Teknisi dapat mengakses site tersebut; data site tampil sesuai relasi | LULUS — teknisi dapat `GET /api/sites` (200); kode handler `assign-users` menulis relasi `user_sites` dlm transaksi (`handlers/site.go:460-485`) |

### S4 - Switch site di UI

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Login sebagai admin/teknisi | Terdapat dropdown **site switcher** di header (menampilkan site aktif saat ini) | LULUS — komponen `SiteSwitcher.tsx` dirender di header `DashboardLayout`; label aktif dibaca dari cookie `sigap_site_id` dgn fallback ke site default (`SiteSwitcher.tsx:7-15`) |
| 2 | Klik dropdown, pilih site lain (mis. site yang baru dibuat) | Cookie `sigap_site_id` tersimpan di browser dengan ID site terpilih | LULUS — `SiteSwitcher` memanggil `setCookie('sigap_site_id', id)` (konstanta `SITE_COOKIE`, `SiteSwitcher.tsx:7`); cookie disimpan 30 hari via `lib/api.ts:149` |
| 3 | Amati perilaku setelah pilihan site | Halaman di-`reload`; site aktif di header berubah | LULUS — setelah pilih, komponen memanggil `window.location.reload()` (kode `SiteSwitcher`) |
| 4 | Amati data yang ditampilkan | Data (CCTV/log/zona) difilter sesuai site aktif terpilih | ⚠️ SEBAGIAN — `apiRequest` menyisipkan header `X-Site-ID` dari cookie (`lib/api.ts:224`) dan handler CCTV/log/zona membaca `c.Locals("site_id")` (`handlers/cctv.go:333-334`, `maintenance.go:173-174`, `zone.go:70-71`), TETAPI middleware `SiteContext()` saat ini hanya terpasang pada grup `/api/sites` (`routes/site.go:14`) — belum terpasang di route `/api/cctv`, `/api/maintenance-logs`, `/api/zones`. Tes nyata: `GET /api/cctv` dgn `X-Site-ID` acak/invalid → 200 (filter tidak aktif). Lihat Catatan di bawah |
| 5 | Muat ulang halaman manual (F5) | Site aktif tetap dipertahankan (dibaca dari cookie `sigap_site_id`) | LULUS — site aktif dibaca dari cookie di setiap render (`SiteSwitcher.tsx:15`), persisten lintas reload |

### S5 - Isolasi data per site

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Pastikan ada ≥ 2 site dengan data berbeda (mis. site OIKN & site BPN) | Kedua site memiliki CCTV/log sendiri-sendiri | ⚠️ TIDAK DAPAT DIBUKTIKAN SEPENUHNYA — semua data produksi berada di site OIKN; site uji "Kawasan Balikpapan" (BPN) dibuat lalu di-soft-delete setelah verifikasi CRUD (tanpa data) |
| 2 | Pilih site A di dropdown, buka daftar CCTV | CCTV site A tampil | ⚠️ SEBAGIAN — lihat S4 #4: header `X-Site-ID` dikirim frontend, tapi `SiteContext()` belum terpasang di route `/api/cctv` → daftar CCTV saat ini TIDAK terfilter per site (menampilkan seluruh data) |
| 3 | Catat kode CCTV yang terlihat | Ada minimal 1 CCTV milik site A | LULUS — CCTV `MT-CCTV-001` (`63c6e479-...`) tampil di daftar (milik site OIKN setelah migrasi) |
| 4 | Pilih site B di dropdown, buka daftar CCTV | CCTV site B tampil; **CCTV site A TIDAK tampil** | ⚠️ TIDAK TERVERIFIKASI — tidak ada site kedua berisi data di produksi; plus keterbatasan filter (S4 #4) |
| 5 | Ulangi untuk maintenance log & zona | Log/zona site A tidak tampil saat memilih site B (isolasi data berjalan) | ⚠️ TIDAK TERVERIFIKASI — sama seperti #4; filter site pada route log/zona juga belum aktif di produksi |

### S6 - Endpoint daftar site

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Panggil `GET /api/sites` (login) | Respons berisi daftar site **terpaginasi** (`{ rows, pagination }`); site default tampil | LULUS — nyata: `{"data":{"pagination":{"total_rows":1,"total_pages":1,"page":1,"limit":10},"rows":[...]}}` — "Kawasan OIKN" tampil |
| 2 | Uji paginasi (`?page=2&limit=1`) | `total_rows`/`total_pages` sesuai; page 2 kosong bila hanya 1 site | LULUS — nyata: `page=2&limit=1` → 200 dgn `rows: []`, `total_pages: 1` |
| 3 | Panggil `GET /api/sites/all` (login) | Respons berisi seluruh site (tanpa paginasi) — cocok untuk dropdown site switcher | LULUS — nyata: `GET /api/sites/all` → 200, array berisi 1 site (tanpa `pagination`) |
| 4 | Cek respons `GET /api/sites/all` | Site default "Kawasan OIKN" ada di daftar; field `is_default` tersedia | LULUS — nyata: `{"id":"5759393a-...","nama":"Kawasan OIKN","kode":"OIKN","is_default":true,...}` |

### S7 - Role access

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Login sebagai admin | Dapat **menulis** site (CRUD, assign-users) — sukses | LULUS — nyata: POST 201, PUT 200, DELETE 200, assign-users 200 |
| 2 | Login sebagai teknisi | Dapat **membaca** daftar site (`GET /api/sites`, `GET /api/sites/all`) — sukses | LULUS — nyata: teknisi `GET /api/sites` → 200 |
| 3 | Login sebagai teknisi, coba buat/ubah/hapus site | Permintaan ditolak (403) — hanya admin yang bisa menulis site | LULUS — nyata: teknisi `POST /api/sites` → 403, `PUT /api/sites/:id` → 403, `DELETE /api/sites/:id` → 403 |
| 4 | Login sebagai teknisi, coba `POST /api/sites/:id/assign-users` | Permintaan ditolak (403) — assign teknisi hanya untuk admin | LULUS — nyata: teknisi `POST /api/sites/5759393a.../assign-users` → `ERR:403` |
| 5 | Akses endpoint site tanpa login | Permintaan ditolak (401) — endpoint site wajib autentikasi | LULUS — nyata: tanpa token `GET /api/sites` → `ERR:401` |

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

## Hasil Nyata — Catatan Verifikator

**TEMUAN (S4 #4 & S5):** Filter data per site belum aktif di produksi untuk endpoint selain `/api/sites`.
- Middleware `SiteContext()` (membaca `X-Site-ID` / `?site_id=`, validasi UUID + keberadaan site) hanya dipasang pada grup `/api/sites` (`backend/routes/site.go:14`). Route `/api/cctv`, `/api/maintenance-logs`, dan `/api/zones` **belum** memuat middleware ini.
- Handler-nya sebenarnya sudah siap: `handlers/cctv.go:333-334`, `handlers/maintenance.go:173-174`, `handlers/zone.go:70-71` membaca `c.Locals("site_id")` sebagai filter opsional.
- Frontend sudah mengirim header `X-Site-ID` dari cookie `sigap_site_id` di semua request (`frontend/src/lib/api.ts:224`) dan `SiteSwitcher` sudah menyimpan cookie + reload.
- Tes nyata: `GET /api/cctv` dengan `X-Site-ID` UUID acak / string invalid → tetap 200 (filter tidak dieksekusi).
- **Rekomendasi:** pasang `middleware.SiteContext()` pada grup route `cctv`, `maintenance`, dan `zone` agar isolasi data per site benar-benar berjalan (sesuai klaim changelog frontend Task 12.2).
