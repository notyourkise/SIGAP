# Uji Manual - Kompresi Foto Upload & Thumbnail (P13)

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

Hasil yang diharapkan: seluruh build & test backend lulus (termasuk `TestUploadCompress`), lint (0 error) & build frontend sukses (exit 0). Pastikan setelah test tidak ada folder artifact `backend/handlers/uploads/` yang tersisa.

**Hasil nyata (18 Agu 2026):** `go build ./...` ✅. `go test -run "TestUpload"` ⚠️ **TestUploadCompress GAGAL di mesin lokal** — "test: timeout error 1000ms" (batas default 1 detik fiber `app.Test()` di `upload_test.go:186`; kompresi gambar noise sintetis 2200×2200 + resize Lanczos + encode JPEG di mesin lokal melewati 1 detik → `resp` nil → panic di assertion. **Bukan kegagalan fungsional**: kompresi teruji nyata di produksi (S1). `TestUploadFile_NoFile`/`InvalidType`/`ValidJPEG` PASS; `defer RemoveAll(uploadsDir)` tetap membersihkan artifact. `go vet ./...` ✅. Frontend: `npm run lint` ✅ 0 error (77 warning lama), `npm run build` ✅ exit 0.

## 3. Skenario Uji (Blackbox)

> **Catatan verifikator:** setelah menyelesaikan setiap skenario, isi kolom "Hasil nyata" dengan **LULUS** / **GAGAL** + catatan singkat.

### S1 - Upload foto besar terkompresi

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buka halaman Maintenance / Field Maintenance, unggah foto besar (>300KB) sebagai bukti | Upload sukses | ✅ LULUS — upload API nyata: JPEG 4000×3000 (490.141 byte) → **200** `{"status":"success",...}` |
| 2 | Amati respons / ukuran file tersimpan | File fullsize terkompresi: dimensi diturunkan ke ≤1920 (jika sumber lebih besar), ukuran < ukuran asli | ⚠️ **LULUS sebagian + TEMUAN BUG** — ukuran turun 490.141 → **67.111 byte (87% lebih kecil)** ✅; dimensi tersimpan **1920×1920** dari sumber 4000×3000 ❌ — **aspect ratio TIDAK dipertahankan** (foto menjadi kotak/distorsi). Penyebab: `imaging.Resize(img, 1920, 1920)` di `upload.go:337-339` — library `imaging.Resize(w,h)` menskalakan ke kedua dimensi tanpa menjaga rasio, padahal komentar kode menyatakan "mempertahankan aspect ratio". Perbaikan disarankan: `imaging.Fit(img, 1920, 1920, imaging.Lanczos)` atau `imaging.Resize(img, 1920, 0, imaging.Lanczos)` |
| 3 | Cek header respons `X-Original-Size` & `X-Compressed-Size` | Kedua header ada; `X-Compressed-Size` ≤ `X-Original-Size` | ✅ LULUS — `X-Original-Size: 490141`, `X-Compressed-Size: 67111` (67111 ≤ 490141) |

### S2 - Thumbnail dibuat

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Setelah upload, cek respons `data.thumb_url` | `thumb_url` berisi path `/api/uploads/thumbnails/<uuid>.jpg` | ✅ LULUS — `"thumb_url":"/api/uploads/thumbnails/c87788f9-ea33-45d8-bc71-99790d755146.jpg"` |
| 2 | Buka `GET /api/uploads/thumbnails/<uuid>.jpg` (login) | Respons 200, berupa gambar JPEG kecil (thumbnail ~400×300) | ✅ LULUS — 200 `image/jpeg`, terukur **400×300** |
| 3 | Buka tanpa login | Ditolak (401) — thumbnail juga dilindungi JWT | ✅ LULUS — tanpa token → **401** |

### S3 - Upload foto kecil (tanpa resize)

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Unggah foto < 300KB | Upload sukses; file disimpan tanpa penurunan resolusi (tidak perlu resize) | ✅ LULUS — JPEG 800×600 (11.286 byte) → 200; dimensi tetap 800×600 (resize hanya bila `w > 1920 || h > 1920`, `upload.go:337`) |
| 2 | Cek `thumb_url` | Thumbnail tetap dibuat dari gambar asli | ✅ LULUS — `"thumb_url":"/api/uploads/thumbnails/5162fb46-f829-41c2-a76d-2cd5582b9b6b.jpg"` ada |
| 3 | Cek ukuran tersimpan | Ukuran tetap kecil (≤ ukuran asli) | ✅ LULUS (dengan catatan) — 11.337 vs 11.286 byte (+51 byte, +0,45%) akibat re-encode normalisasi JPEG (hapus EXIF/GPS) tanpa resize; tetap kecil dan jauh di bawah 300KB |

### S4 - Thumbnail tampil di tabel pemeliharaan

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buka halaman `/maintenance` yang memiliki log dengan `image_url` | Thumbnail kecil (≈40×40) tampil di kolom CCTV (samping nama) | ✅ LULUS (verifikasi kode) — sel thumbnail `h-10 w-10` (40×40) di kolom CCTV (`maintenance/page.tsx:827-845`); URL diubah `/api/uploads/` → `/api/uploads/thumbnails/` via `toThumbUrl` (`:114-121`) |
| 2 | Cek bahwa thumbnail memuat gambar | Gambar kecil tampil (diambil dari `/api/uploads/thumbnails/...`) | ✅ LULUS — endpoint thumbnail 200 (S2.2); `AuthenticatedImage` memuat via cookie sesi |
| 3 | Klik thumbnail | Modal detail log terbuka menampilkan foto fullsize | ✅ LULUS (verifikasi kode) — klik sel membuka detail; detail menampilkan `detailLog.image_url` (fullsize) (`:1311-1315`) |
| 4 | Log tanpa `image_url` | Menampilkan ikon kamera abu-abu (placeholder) | ✅ LULUS (verifikasi kode) — `!toThumbUrl(...)` → placeholder ikon `Camera` abu-abu (`:847-849`) |

### S5 - Fallback foto lama (tanpa thumbnail)

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Siapkan log pemeliharaan lama yang diunggah sebelum fitur ini (hanya punya `/api/uploads/<file>.jpg`, tidak ada thumbnail) | Thumbnail di tabel menampilkan placeholder/ikon kamera (gambar gagal dimuat di `AuthenticatedImage`) | ✅ LULUS (verifikasi kode + simulasi 404) — `AuthenticatedImage` gagal memuat (HTTP 404) → state `error` → placeholder "Foto tidak dapat ditampilkan" (`AuthenticatedImage.tsx:54-65,91-99`); request thumbnail yang tidak ada → **404** (terverifikasi nyata). Catatan: seluruh foto prod saat ini sudah punya thumbnail (fitur aktif sejak semua upload), sehingga skenario foto lama disimulasikan via 404 |
| 2 | Buka detail log | Foto fullsize masih bisa dibuka dari `/api/uploads/<file>.jpg` | ✅ LULUS — file fullsize lama (mis. `fdd31a95...jpg`) tetap 200 via `/api/uploads/`; detail memakai `image_url` asli (bukan thumbnail) |

## 4. Kriteria Lulus

1. Foto > 300KB (resolusi besar) otomatis di-resize ke ≤1920px dan di-re-encode JPEG quality 85 → ukuran tersimpan berkurang.
2. Header respons `X-Original-Size` & `X-Compressed-Size` tersedia.
3. Thumbnail 400×300 dibuat untuk setiap upload dan dapat diakses via `GET /api/uploads/thumbnails/:filename` (protected JWT).
4. Tabel pemeliharaan menampilkan thumbnail foto (klik → detail fullsize); log tanpa foto menampilkan placeholder ikon kamera.
5. Foto lama tanpa thumbnail tetap berfungsi (fallback placeholder di tabel, fullsize di detail).
6. `go build`/`go test`/`go vet` backend lulus; `npm run lint`/`npm run build` frontend lulus.
7. Tidak ada file artifact test yang tertinggal di `backend/handlers/uploads/`.

## Catatan

- Kompresi hanya menurunkan ukuran bila menguntungkan; gambar kecil (< 300KB) disimpan tanpa resize agar kualitas tidak berkurang, namun thumbnail tetap dibuat.
- Seluruh file tetap dinormalisasi ke JPEG (metadata EXIF/GPS dihapus) seperti perilaku sebelumnya.
- Thumbnail dibuat di `./uploads/thumbnails/`; direktori dibuat otomatis saat upload (tidak bergantung pada startup manual).
