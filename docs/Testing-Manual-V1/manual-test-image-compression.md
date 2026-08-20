# Uji Manual - Kompresi Foto Upload & Thumbnail (P13)

> **Kontrak uji formal — MT-10**
>
> Dokumen ini mendefinisikan prosedur; hasil sebelum **2026-08-20** adalah catatan historis/non-authoritative dan harus diuji ulang dengan execution record formal. Inspeksi kode, review unit test, atau checkmark lama bukan `PASS` black-box.
>
> **Skenario audit reproduktif:** Dengan fixture foto landscape, portrait, square, EXIF rotation, dan batas ukuran, unggah lalu bandingkan rasio, dimensi, ukuran, serta thumbnail.
>
> **Status suite:** `NOT RUN`

### Formal execution metadata

| Field | Value |
|---|---|
| Test ID / Audit ID | MT-10 |
| Capability owner | QA Media + Backend |
| Status suite | NOT RUN |
| Build SHA | NOT RECORDED |
| Artifact digest/ID | NOT RECORDED |
| Tester | NOT RECORDED |
| Timestamp WITA | NOT RECORDED |
| Environment | NOT RECORDED |
| Fixture prefix/data | `E2E-MT10-` / NOT RECORDED |
| Cleanup result | NOT RUN |
| Expected | Rasio asli dipertahankan, sisi terpanjang maksimal 1920 px, thumbnail sesuai spesifikasi, dan hasil dapat dibuktikan dari artifact runtime. |
| Actual | NOT RECORDED |
| Evidence path/link | NOT RECORDED |
| Mandatory flag | YES |
| Scenario status | NOT RUN |

### Formal scenario record

| ID | Mandatory | Expected | Actual | Evidence | Status |
|---|---|---|---|---|---|
| MT-10-S1 | YES | Rasio asli dipertahankan, sisi terpanjang maksimal 1920 px, thumbnail sesuai spesifikasi, dan hasil dapat dibuktikan dari artifact runtime. | NOT RECORDED | NOT RECORDED | NOT RUN |

> **Cakupan aggregate Wave 0:** Baris formal aggregate di atas adalah **sole atomic suite-level acceptance gate** untuk baseline Wave 0. Skenario legacy terperinci di bawah ini tetap non-formal dan historical/non-authoritative sampai diperluas selama Wave 7.

**Aturan kesimpulan suite:** sebelum eksekusi suite `NOT RUN`; setelah run dimulai, setiap mandatory scenario harus `PASS`. Satu mandatory scenario berstatus `FAIL`, `BLOCKED`, atau `NOT RUN` membuat kesimpulan suite `FAIL`.

> **Guard keselamatan wajib:** Untuk setiap SQL mutasi, `DELETE`/`TRUNCATE`, rotasi/regenerasi key atau QR, penghapusan, reset kredensial, atau mutasi credential, lanjut hanya bila environment bukan production dan target bukan `nexaspace.site`, fixture prefix `E2E-MT10-` serta exact targets sudah dicatat, backup/recovery terverifikasi untuk perubahan data/key state, dan cleanup/rollback didefinisikan serta dicatat setelah run. Jika salah satu syarat tidak tersedia, hentikan prosedur dan set scenario status ke `BLOCKED`; jangan jalankan pada production. Jangan menyalin credential, token, private key, RTSP password, atau payload production yang belum disanitasi.


> **Task:** 13.13 Kompresi Foto Upload Server-Side (Sprint 35 - P13)
> **Branch:** `feature/performance-checklist`
> **Versi:** backend `0.57.0` / frontend `1.48.0`
> **Dokumen:** verifikasi foto yang diunggah dikompresi otomatis (resize + re-encode JPEG) serta thumbnail 400×300 dibuat dan ditampilkan di daftar pemeliharaan.

## 1. Prasyarat

| # | Item | Cara menyiapkan |
|---|------|-----------------|
| P1 | Aplikasi berjalan | `docker compose up -d --build`, buka `https://localhost` (atau `go run main.go` + `npm run dev`) |
| P2 | Foto uji berukuran besar | Siapkan foto JPEG > 300KB dengan resolusi tinggi (mis. 4000×3000 dari HP). Pastikan < 2MB (batas upload) |
| P3 | Foto uji kecil | Siapkan foto < 300KB untuk memverifikasi thumbnail tetap dibuat walau tanpa resize |
| P4 | Akun pengguna | Login sebagai admin atau teknisi |

## 2. Pengujian Otomatis

```powershell
cd backend
go build ./...
go test ./... -run "TestUpload" -count=1
go vet ./...

cd ../frontend
npm run lint
npm run build
```

Hasil yang diharapkan: seluruh build & test backend memenuhi kriteria (termasuk `TestUploadCompress`), lint (0 error) & build frontend sukses (exit 0). Pastikan setelah test tidak ada folder artifact `backend/handlers/uploads/` yang tersisa.

**Observasi historis (non-authoritative) (18 Agu 2026):** `go build ./...` NOT RUN. `go test -run "TestUpload"` NOT RUN **TestUploadCompress NOT RUN di mesin lokal** — "test: timeout error 1000ms" (batas default 1 detik fiber `app.Test()` di `upload_test.go:186`; kompresi gambar noise sintetis 2200×2200 + resize Lanczos + encode JPEG di mesin lokal melewati 1 detik → `resp` nil → panic di assertion. **Bukan kegagalan fungsional**: kompresi teruji nyata di produksi (S1). `TestUploadFile_NoFile`/`InvalidType`/`ValidJPEG` NOT RUN; `defer RemoveAll(uploadsDir)` tetap membersihkan artifact. `go vet ./...` NOT RUN. Frontend: `npm run lint` NOT RUN 0 error (77 warning lama), `npm run build` NOT RUN exit 0.

## 3. Skenario Uji (Blackbox)

> **Catatan legacy:** tabel di bawah adalah observasi historis/non-authoritative; jangan gunakan sebagai status formal. Isi `Formal scenario record` di atas dengan expected, actual, evidence, dan status formal.

### S1 - Upload foto besar terkompresi

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Buka halaman Maintenance / Field Maintenance, unggah foto besar (>300KB) sebagai bukti | Upload sukses | NOT RUN |
| 2 | Amati respons / ukuran file tersimpan | File fullsize terkompresi: dimensi diturunkan ke ≤1920 (jika sumber lebih besar), ukuran < ukuran asli | NOT RUN **historical partial observation + TEMUAN BUG** — ukuran turun 490.141 → **67.111 byte (87% lebih kecil)** NOT RUN; dimensi tersimpan **1920×1920** dari sumber 4000×3000 NOT RUN — **aspect ratio TIDAK dipertahankan** (foto menjadi kotak/distorsi). Perbaikan disarankan memakai fit-resize agar rasio terjaga |
| 3 | Cek header respons `X-Original-Size` & `X-Compressed-Size` | Kedua header ada; `X-Compressed-Size` ≤ `X-Original-Size` | NOT RUN |

### S2 - Thumbnail dibuat

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Setelah upload, cek respons `data.thumb_url` | `thumb_url` berisi path `/api/uploads/thumbnails/<uuid>.jpg` | NOT RUN |
| 2 | Buka `GET /api/uploads/thumbnails/<uuid>.jpg` (login) | Respons 200, berupa gambar JPEG kecil (thumbnail ~400×300) | NOT RUN |
| 3 | Buka tanpa login | Ditolak (401) — thumbnail juga dilindungi JWT | NOT RUN |

### S3 - Upload foto kecil (tanpa resize)

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Unggah foto < 300KB | Upload sukses; file disimpan tanpa penurunan resolusi (tidak perlu resize) | NOT RUN |
| 2 | Cek `thumb_url` | Thumbnail tetap dibuat dari gambar asli | NOT RUN |
| 3 | Cek ukuran tersimpan | Ukuran tetap kecil (≤ ukuran asli) | NOT RUN |

### S4 - Thumbnail tampil di tabel pemeliharaan

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Buka halaman `/maintenance` yang memiliki log dengan `image_url` | Thumbnail kecil (≈40×40) tampil di kolom CCTV (samping nama) | NOT RUN |
| 2 | Cek bahwa thumbnail memuat gambar | Gambar kecil tampil (diambil dari `/api/uploads/thumbnails/...`) | NOT RUN |
| 3 | Klik thumbnail | Modal detail log terbuka menampilkan foto fullsize | NOT RUN |
| 4 | Log tanpa `image_url` | Menampilkan ikon kamera abu-abu (placeholder) | NOT RUN |

### S5 - Fallback foto lama (tanpa thumbnail)

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Siapkan log pemeliharaan lama yang diunggah sebelum fitur ini (hanya punya `/api/uploads/<file>.jpg`, tidak ada thumbnail) | Thumbnail di tabel menampilkan placeholder/ikon kamera (gambar gagal dimuat di `AuthenticatedImage`) | NOT RUN |
| 2 | Buka detail log | Foto fullsize masih bisa dibuka dari `/api/uploads/<file>.jpg` | NOT RUN |

## 4. Kriteria Penerimaan

1. Foto > 300KB (resolusi besar) otomatis di-resize ke ≤1920px dan di-re-encode JPEG quality 85 → ukuran tersimpan berkurang.
2. Header respons `X-Original-Size` & `X-Compressed-Size` tersedia.
3. Thumbnail 400×300 dibuat untuk setiap upload dan dapat diakses via `GET /api/uploads/thumbnails/:filename` (protected JWT).
4. Tabel pemeliharaan menampilkan thumbnail foto (klik → detail fullsize); log tanpa foto menampilkan placeholder ikon kamera.
5. Foto lama tanpa thumbnail tetap berfungsi (fallback placeholder di tabel, fullsize di detail).
6. `go build`/`go test`/`go vet` backend memenuhi kriteria; `npm run lint`/`npm run build` frontend memenuhi kriteria.
7. Tidak ada file artifact test yang tertinggal di `backend/handlers/uploads/`.

## Catatan

- Kompresi hanya menurunkan ukuran bila menguntungkan; gambar kecil (< 300KB) disimpan tanpa resize agar kualitas tidak berkurang, namun thumbnail tetap dibuat.
- Seluruh file tetap dinormalisasi ke JPEG (metadata EXIF/GPS dihapus) seperti perilaku sebelumnya.
- Thumbnail dibuat di `./uploads/thumbnails/`; direktori dibuat otomatis saat upload (tidak bergantung pada startup manual).
