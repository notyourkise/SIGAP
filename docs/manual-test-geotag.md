# Uji Manual - Geotag Pengecekan Lapangan

> **Task:** 8.4 Geotag Pengecekan Lapangan
> **Branch:** `feature/geotag-field-check`
> **Versi:** backend `0.42.0` - frontend `1.35.0`
> **Dokumen:** verifikasi pencatatan koordinat GPS saat pengecekan lapangan, validasi backend, indikator lokasi di wizard, dan tampilan mini-map lokasi pengecekan untuk admin.

Saat teknisi membuka wizard pengecekan lapangan, posisi GPS (`navigator.geolocation`) dicatat otomatis dan dikirim bersama form sebagai `check_latitude`/`check_longitude`. Persetujuan lokasi bersifat **opsional** — wizard tetap bisa diselesaikan tanpa GPS. Admin dapat melihat lokasi pengecekan di peta (mini-map) untuk membandingkan dengan lokasi CCTV.

## 1. Prasyarat

| # | Item | Cara menyiapkan |
|---|------|-----------------|
| P1 | Aplikasi berjalan | `docker compose up -d --build`, buka `https://localhost` (atau `npm run dev` → `http://localhost:3001`) |
| P2 | Akun admin | Login sebagai admin |
| P3 | Akun teknisi | Siapkan user berperan `teknisi` |
| P4 | CCTV dengan koordinat | Siapkan CCTV yang memiliki `latitude`/`longitude` terisi (untuk perbandingan jarak di peta) |
| P5 | Browser dengan GPS | Chrome/Edge; gunakan DevTools → Sensors untuk mensimulasikan lokasi bila perlu |
| P6 | Izin lokasi | Pastikan izin lokasi situs (bisa ditolak / diizinkan untuk menguji kedua jalur) |

## 2. Pengujian Otomatis

```powershell
cd backend
go build ./...
go test ./... -count=1
go vet ./...
swag init

cd ../frontend
npm run lint
npm run build
```

Hasil yang diharapkan: seluruh perintah sukses; lint tidak memiliki error (0 error).

**Hasil nyata (18 Agu 2026):** `go build ./...` ✅, `go test ./... -count=1` ⚠️ 2 test flaky `TestCreateAPIKey_Success` & `TestAPIKeyAuth_ValidKey` — "test: timeout error 1000ms" (batas default 1 detik fiber `app.Test()` saat beban CPU; keduanya **PASS saat dijalankan sendiri** 0.32s/0.55s — bukan kegagalan kode; test field-maintenance/geotag semua PASS), `go vet ./...` ✅. Frontend: `npm run lint` ✅ 0 error (77 warning lama), `npm run build` ✅ exit 0. `swag init` tidak dieksekusi (tool tidak terpasang; tidak ada perubahan anotasi).

## 3. Skenario Uji (Blackbox)

> **Catatan verifikator:** setelah menyelesaikan setiap skenario, isi kolom "Hasil nyata" dengan **LULUS** / **GAGAL** + catatan singkat.

### S1 - GPS diizinkan → koordinat tercatat

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Login sebagai teknisi, buka halaman Field Maintenance, mulai wizard pengecekan baru | Saat wizard terbuka, browser meminta izin lokasi | ✅ |
| 2 | Pilih "Izinkan" (simulasi lokasi bila perlu) | Muncul indikator **"Lokasi tercatat"** (📍) di wizard | ✅ |
| 3 | Isi checklist & foto, lalu selesaikan pengecekan | Catatan tersimpan | ✅ |
| 4 | Cek database `field_maintenances` | Kolom `check_latitude` & `check_longitude` terisi | ✅ |
| 5 | Buka detail catatan sebagai admin | Muncul bagian **"Lokasi Pengecekan"** dengan mini-map | ✅ |

### S2 - GPS ditolak / timeout → wizard tetap lanjut (opsional)

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buka wizard pengecekan, pilih "Tolak" pada prompt izin lokasi | Indikator menampilkan **"Lokasi belum tercatat (opsional)"**; wizard tetap berjalan | ✅ |
| 2 | Isi checklist & foto, selesaikan pengecekan | Catatan tersimpan TANPA koordinat (tidak gagal) | ✅ |
| 3 | Cek database | `check_latitude` & `check_longitude` kosong/null | ✅ |
| 4 | Buka detail sebagai admin | Bagian "Lokasi Pengecekan" menampilkan keterangan tidak ada data lokasi (tanpa map) | ✅ |

### S3 - Validasi koordinat backend

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Kirim request `POST /api/field-maintenance` dengan `check_latitude:"999"` (di luar rentang) via Swagger | Respons 400 dengan pesan latitude di luar rentang -90..90 | ✅ |
| 2 | Kirim dengan `check_longitude:"abc"` (bukan angka) | Respons 400 dengan pesan longitude harus angka valid | ✅ |
| 3 | Kirim dengan hanya `check_latitude` (tanpa longitude) | Respons 400: keduanya wajib diisi bersamaan | ✅ |
| 4 | Kirim tanpa koordinat sama sekali | Diterima (201) — koordinat opsional | ✅ |

### S4 - Mini-map lokasi pengecekan (admin view)

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Admin buka detail pengecekan yang memiliki koordinat | Mini-map menampilkan **marker CCTV** (biru) dan **marker Lokasi Pengecekan** (oranye) | ✅ |
| 2 | Periksa legend peta | Legend menjelaskan warna: CCTV vs Lokasi Pengecekan | ✅ |
| 3 | Hover/klik marker | Tooltip/Popup menampilkan nama CCTV / keterangan lokasi pengecekan | ✅ |
| 4 | Bandingkan jarak kedua marker | Marker terlihat terpisah sesuai koordinat (jika teknisi jauh dari CCTV) | ✅ |

### S5 - Mode "Selesaikan Tugas" (complete) tidak menangkap lokasi

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Dari halaman Maintenance, buka "Selesaikan Tugas" untuk log terjadwal | Wizard mode complete TIDAK meminta izin lokasi dan tidak menampilkan indikator lokasi | ✅ |
| 2 | Isi checklist & selesaikan | Penyelesaian tugas normal tanpa koordinat geotag | ✅ |

### S6 - Konsistensi & scope

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Cek respons `GET /api/field-maintenance` | Setiap record menyertakan `check_latitude`/`check_longitude` (kosong bila tidak ada) | ✅ |
| 2 | Cek respons `GET /api/field-maintenance/:id` | Field koordinat tersedia di detail | ✅ |
| 3 | Periksa endpoint dengan teknologi tanpa GPS (mis. lokasi tidak tersedia) | Wizard tetap berjalan; koordinat kosong | ✅ |

## 4. Kriteria Lulus

1. Model `FieldMaintenance` memiliki `check_latitude` & `check_longitude` (nullable) + migrasi `000003`.
2. `CreateFieldMaintenance` menerima koordinat opsional dan memvalidasi format/rentang (lat -90..90, lng -180..180).
3. Koordinat tercatat di database bila teknisi mengizinkan GPS; opsional bila ditolak/timeout.
4. Wizard menampilkan indikator "Lokasi tercatat" / "Lokasi belum tercatat (opsional)".
5. Admin melihat marker "Lokasi Pengecekan" di mini-map beserta marker CCTV untuk perbandingan jarak.
6. `go build`/`go test`/`go vet`, `npm run lint`/`npm run build` lulus.

## Catatan

- Geotag bersifat opsional dan tidak menghambat alur kerja — teknisi yang menolak izin lokasi tetap bisa menyelesaikan pengecekan.
- Koordinat disimpan sebagai string (`varchar(30)`) untuk mempertahankan presisi dan format asli dari browser.
- Untuk uji tanpa perangkat GPS, gunakan DevTools → Sensors untuk mengatur lokasi simulasi di browser.
- Mini-map hanya tampil di modal detail admin bila koordinat pengecekan tersedia; CCTV tanpa koordinat memakai fallback pusat IKN.
