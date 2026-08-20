# Uji Manual - Geotag Pengecekan Lapangan

> **Kontrak uji formal — MT-08**
>
> Dokumen ini mendefinisikan prosedur; hasil sebelum **2026-08-20** adalah catatan historis/non-authoritative dan harus diuji ulang dengan execution record formal. Inspeksi kode, review unit test, atau checkmark lama bukan `PASS` black-box.
>
> **Skenario audit reproduktif:** Dengan koordinat simulasi dan permission state granted, denied, serta timeout, kirim geotag valid/invalid dan baca record milik/non-owner.
>
> **Status suite:** `NOT RUN`

### Formal execution metadata

| Field | Value |
|---|---|
| Test ID / Audit ID | MT-08 |
| Capability owner | QA Mobile/Web + Frontend/Backend |
| Status suite | NOT RUN |
| Build SHA | NOT RECORDED |
| Artifact digest/ID | NOT RECORDED |
| Tester | NOT RECORDED |
| Timestamp WITA | NOT RECORDED |
| Environment | NOT RECORDED |
| Fixture prefix/data | `E2E-MT08-` / NOT RECORDED |
| Cleanup result | NOT RUN |
| Expected | Koordinat valid tersimpan presisi, permission denied memiliki fallback, input invalid ditolak, dan non-owner tidak dapat membaca record. |
| Actual | NOT RECORDED |
| Evidence path/link | NOT RECORDED |
| Mandatory flag | YES |
| Scenario status | NOT RUN |

### Formal scenario record

| ID | Mandatory | Expected | Actual | Evidence | Status |
|---|---|---|---|---|---|
| MT-08-S1 | YES | Koordinat valid tersimpan presisi, permission denied memiliki fallback, input invalid ditolak, dan non-owner tidak dapat membaca record. | NOT RECORDED | NOT RECORDED | NOT RUN |

> **Cakupan aggregate Wave 0:** Baris formal aggregate di atas adalah **sole atomic suite-level acceptance gate** untuk baseline Wave 0. Skenario legacy terperinci di bawah ini tetap non-formal dan historical/non-authoritative sampai diperluas selama Wave 7.

**Aturan kesimpulan suite:** sebelum eksekusi suite `NOT RUN`; setelah run dimulai, setiap mandatory scenario harus `PASS`. Satu mandatory scenario berstatus `FAIL`, `BLOCKED`, atau `NOT RUN` membuat kesimpulan suite `FAIL`.

> **Guard keselamatan wajib:** Untuk setiap SQL mutasi, `DELETE`/`TRUNCATE`, rotasi/regenerasi key atau QR, penghapusan, reset kredensial, atau mutasi credential, lanjut hanya bila environment bukan production dan target bukan `nexaspace.site`, fixture prefix `E2E-MT08-` serta exact targets sudah dicatat, backup/recovery terverifikasi untuk perubahan data/key state, dan cleanup/rollback didefinisikan serta dicatat setelah run. Jika salah satu syarat tidak tersedia, hentikan prosedur dan set scenario status ke `BLOCKED`; jangan jalankan pada production. Jangan menyalin credential, token, private key, RTSP password, atau payload production yang belum disanitasi.


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

**Observasi historis (non-authoritative) (18 Agu 2026):** `go build ./...` NOT RUN, `go test ./... -count=1` NOT RUN 2 test flaky `TestCreateAPIKey_Success` & `TestAPIKeyAuth_ValidKey` — "test: timeout error 1000ms" (batas default 1 detik fiber `app.Test()` saat beban CPU; keduanya **NOT RUN saat dijalankan sendiri** 0.32s/0.55s — bukan kegagalan kode; test field-maintenance/geotag semua NOT RUN), `go vet ./...` NOT RUN. Frontend: `npm run lint` NOT RUN 0 error (77 warning lama), `npm run build` NOT RUN exit 0. `swag init` tidak dieksekusi (tool tidak terpasang; tidak ada perubahan anotasi).

## 3. Skenario Uji (Blackbox)

> **Catatan legacy:** tabel di bawah adalah observasi historis/non-authoritative; jangan gunakan sebagai status formal. Isi `Formal scenario record` di atas dengan expected, actual, evidence, dan status formal.

### S1 - GPS diizinkan → koordinat tercatat

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Login sebagai teknisi, buka halaman Field Maintenance, mulai wizard pengecekan baru | Saat wizard terbuka, browser meminta izin lokasi | NOT RUN |
| 2 | Pilih "Izinkan" (simulasi lokasi bila perlu) | Muncul indikator **"Lokasi tercatat"** (📍) di wizard | NOT RUN |
| 3 | Isi checklist & foto, lalu selesaikan pengecekan | Catatan tersimpan | NOT RUN |
| 4 | Cek database `field_maintenances` | Kolom `check_latitude` & `check_longitude` terisi | NOT RUN |
| 5 | Buka detail catatan sebagai admin | Muncul bagian **"Lokasi Pengecekan"** dengan mini-map | NOT RUN |

### S2 - GPS ditolak / timeout → wizard tetap lanjut (opsional)

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Buka wizard pengecekan, pilih "Tolak" pada prompt izin lokasi | Indikator menampilkan **"Lokasi belum tercatat (opsional)"**; wizard tetap berjalan | NOT RUN |
| 2 | Isi checklist & foto, selesaikan pengecekan | Catatan tersimpan TANPA koordinat (tidak gagal) | NOT RUN |
| 3 | Cek database | `check_latitude` & `check_longitude` kosong/null | NOT RUN |
| 4 | Buka detail sebagai admin | Bagian "Lokasi Pengecekan" menampilkan keterangan tidak ada data lokasi (tanpa map) | NOT RUN |

### S3 - Validasi koordinat backend

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Kirim request `POST /api/field-maintenance` dengan `check_latitude:"999"` (di luar rentang) via Swagger | Respons 400 dengan pesan latitude di luar rentang -90..90 | NOT RUN |
| 2 | Kirim dengan `check_longitude:"abc"` (bukan angka) | Respons 400 dengan pesan longitude harus angka valid | NOT RUN |
| 3 | Kirim dengan hanya `check_latitude` (tanpa longitude) | Respons 400: keduanya wajib diisi bersamaan | NOT RUN |
| 4 | Kirim tanpa koordinat sama sekali | Diterima (201) — koordinat opsional | NOT RUN |

### S4 - Mini-map lokasi pengecekan (admin view)

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Admin buka detail pengecekan yang memiliki koordinat | Mini-map menampilkan **marker CCTV** (biru) dan **marker Lokasi Pengecekan** (oranye) | NOT RUN |
| 2 | Periksa legend peta | Legend menjelaskan warna: CCTV vs Lokasi Pengecekan | NOT RUN |
| 3 | Hover/klik marker | Tooltip/Popup menampilkan nama CCTV / keterangan lokasi pengecekan | NOT RUN |
| 4 | Bandingkan jarak kedua marker | Marker terlihat terpisah sesuai koordinat (jika teknisi jauh dari CCTV) | NOT RUN |

### S5 - Mode "Selesaikan Tugas" (complete) tidak menangkap lokasi

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Dari halaman Maintenance, buka "Selesaikan Tugas" untuk log terjadwal | Wizard mode complete TIDAK meminta izin lokasi dan tidak menampilkan indikator lokasi | NOT RUN |
| 2 | Isi checklist & selesaikan | Penyelesaian tugas normal tanpa koordinat geotag | NOT RUN |

### S6 - Konsistensi & scope

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Cek respons `GET /api/field-maintenance` | Setiap record menyertakan `check_latitude`/`check_longitude` (kosong bila tidak ada) | NOT RUN |
| 2 | Cek respons `GET /api/field-maintenance/:id` | Field koordinat tersedia di detail | NOT RUN |
| 3 | Periksa endpoint dengan teknologi tanpa GPS (mis. lokasi tidak tersedia) | Wizard tetap berjalan; koordinat kosong | NOT RUN |

## 4. Kriteria Penerimaan

1. Model `FieldMaintenance` memiliki `check_latitude` & `check_longitude` (nullable) + migrasi `000003`.
2. `CreateFieldMaintenance` menerima koordinat opsional dan memvalidasi format/rentang (lat -90..90, lng -180..180).
3. Koordinat tercatat di database bila teknisi mengizinkan GPS; opsional bila ditolak/timeout.
4. Wizard menampilkan indikator "Lokasi tercatat" / "Lokasi belum tercatat (opsional)".
5. Admin melihat marker "Lokasi Pengecekan" di mini-map beserta marker CCTV untuk perbandingan jarak.
6. `go build`/`go test`/`go vet`, `npm run lint`/`npm run build` memenuhi kriteria.

## Catatan

- Geotag bersifat opsional dan tidak menghambat alur kerja — teknisi yang menolak izin lokasi tetap bisa menyelesaikan pengecekan.
- Koordinat disimpan sebagai string (`varchar(30)`) untuk mempertahankan presisi dan format asli dari browser.
- Untuk uji tanpa perangkat GPS, gunakan DevTools → Sensors untuk mengatur lokasi simulasi di browser.
- Mini-map hanya tampil di modal detail admin bila koordinat pengecekan tersedia; CCTV tanpa koordinat memakai fallback pusat IKN.
