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

## 3. Skenario Uji (Blackbox)

> **Catatan verifikator:** setelah menyelesaikan setiap skenario, isi kolom "Hasil nyata" dengan **LULUS** / **GAGAL** + catatan singkat.

### S1 - Upload foto besar terkompresi

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buka halaman Maintenance / Field Maintenance, unggah foto besar (>300KB) sebagai bukti | Upload sukses |
| 2 | Amati respons / ukuran file tersimpan | File fullsize terkompresi: dimensi diturunkan ke ≤1920 (jika sumber lebih besar), ukuran < ukuran asli |
| 3 | Cek header respons `X-Original-Size` & `X-Compressed-Size` | Kedua header ada; `X-Compressed-Size` ≤ `X-Original-Size` |

### S2 - Thumbnail dibuat

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Setelah upload, cek respons `data.thumb_url` | `thumb_url` berisi path `/api/uploads/thumbnails/<uuid>.jpg` |
| 2 | Buka `GET /api/uploads/thumbnails/<uuid>.jpg` (login) | Respons 200, berupa gambar JPEG kecil (thumbnail ~400×300) |
| 3 | Buka tanpa login | Ditolak (401) — thumbnail juga dilindungi JWT |

### S3 - Upload foto kecil (tanpa resize)

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Unggah foto < 300KB | Upload sukses; file disimpan tanpa penurunan resolusi (tidak perlu resize) |
| 2 | Cek `thumb_url` | Thumbnail tetap dibuat dari gambar asli |
| 3 | Cek ukuran tersimpan | Ukuran tetap kecil (≤ ukuran asli) |

### S4 - Thumbnail tampil di tabel pemeliharaan

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buka halaman `/maintenance` yang memiliki log dengan `image_url` | Thumbnail kecil (≈40×40) tampil di kolom CCTV (samping nama) |
| 2 | Cek bahwa thumbnail memuat gambar | Gambar kecil tampil (diambil dari `/api/uploads/thumbnails/...`) |
| 3 | Klik thumbnail | Modal detail log terbuka menampilkan foto fullsize |
| 4 | Log tanpa `image_url` | Menampilkan ikon kamera abu-abu (placeholder) |

### S5 - Fallback foto lama (tanpa thumbnail)

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Siapkan log pemeliharaan lama yang diunggah sebelum fitur ini (hanya punya `/api/uploads/<file>.jpg`, tidak ada thumbnail) | Thumbnail di tabel menampilkan placeholder/ikon kamera (gambar gagal dimuat di `AuthenticatedImage`) |
| 2 | Buka detail log | Foto fullsize masih bisa dibuka dari `/api/uploads/<file>.jpg` |

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
