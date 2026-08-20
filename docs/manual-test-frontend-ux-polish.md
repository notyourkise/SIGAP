# Uji Manual - Polish Frontend & UX (Sprint 36)

> **Kontrak uji formal — MT-07**
>
> Dokumen ini mendefinisikan prosedur; hasil sebelum **2026-08-20** adalah catatan historis/non-authoritative dan harus diuji ulang dengan execution record formal. Inspeksi kode, review unit test, atau checkmark lama bukan `PASS` black-box.
>
> **Skenario audit reproduktif:** Pisahkan suite monitor, kalender, dan kinerja teknisi, lalu jalankan browser test dengan evidence DevTools/screenshot untuk lazy loading, modal, dan request periode.
>
> **Status suite:** `NOT RUN`

### Formal execution metadata

| Field | Value |
|---|---|
| Test ID / Audit ID | MT-07 |
| Capability owner | QA Web + Frontend |
| Status suite | NOT RUN |
| Build SHA | NOT RECORDED |
| Artifact digest/ID | NOT RECORDED |
| Tester | NOT RECORDED |
| Timestamp WITA | NOT RECORDED |
| Environment | NOT RECORDED |
| Fixture prefix/data | `E2E-MT07-` / NOT RECORDED |
| Cleanup result | NOT RUN |
| Expected | Setiap suite memiliki hasil runtime independen, evidence browser, request periode yang benar, dan tidak tertukar dengan inspeksi kode. |
| Actual | NOT RECORDED |
| Evidence path/link | NOT RECORDED |
| Mandatory flag | YES |
| Scenario status | NOT RUN |

### Formal scenario record

| ID | Mandatory | Expected | Actual | Evidence | Status |
|---|---|---|---|---|---|
| MT-07-S1 | YES | Setiap suite memiliki hasil runtime independen, evidence browser, request periode yang benar, dan tidak tertukar dengan inspeksi kode. | NOT RECORDED | NOT RECORDED | NOT RUN |

**Aturan kesimpulan suite:** sebelum eksekusi suite `NOT RUN`; setelah run dimulai, setiap mandatory scenario harus `PASS`. Satu mandatory scenario berstatus `FAIL`, `BLOCKED`, atau `NOT RUN` membuat kesimpulan suite `FAIL`.

> **Guard keselamatan wajib:** Untuk setiap SQL mutasi, `DELETE`/`TRUNCATE`, rotasi/regenerasi key atau QR, penghapusan, reset kredensial, atau mutasi credential, lanjut hanya bila environment bukan production dan target bukan `nexaspace.site`, fixture prefix `E2E-MT07-` serta exact targets sudah dicatat, backup/recovery terverifikasi untuk perubahan data/key state, dan cleanup/rollback didefinisikan serta dicatat setelah run. Jika salah satu syarat tidak tersedia, hentikan prosedur dan set scenario status ke `BLOCKED`; jangan jalankan pada production. Jangan menyalin credential, token, private key, RTSP password, atau payload production yang belum disanitasi.


> **Task:** 13.15 Lazy Loading StreamGridView, 13.17 Buat Tugas dari Kalender, 13.18 Filter Periode Kinerja Teknisi
> **Branch:** `fix/frontend-ux-polish`
> **Versi:** backend `0.58.0` / frontend `1.49.0`
> **Dokumen:** verifikasi performa grid streaming (lazy load, pause/resume, pagination), pembuatan tugas dari kalender, dan filter periode dashboard kinerja teknisi.

## 1. Prasyarat

| # | Item | Cara menyiapkan |
|---|------|-----------------|
| P1 | Aplikasi berjalan | `docker compose up -d --build`, buka `https://localhost` (atau `go run main.go` di `backend/` + `npm run dev` di `frontend/` → `http://localhost:3001`) |
| P2 | Akun Admin | Login sebagai admin (untuk `/monitor`, `/kalender`, `/kinerja-teknisi`) |
| P3 | CCTV dengan streaming aktif | Setidaknya 13+ CCTV `stream_enabled=true` untuk menguji pagination & lazy load (jika kurang, atur pageSize 4/8) |
| P4 | Data pemeliharaan & teknisi | Pastikan ada beberapa teknisi dan log pemeliharaan untuk menguji filter kinerja & create dari kalender |

## 2. Pengujian Otomatis

```powershell
cd backend
go build ./...
go test ./... -run "TestTechnicianPerf|TestDashboard|TestTechnicianPerformance" -count=1
go vet ./...

cd ../frontend
npm run lint
npm run build
npx tsc --noEmit
```

Hasil yang diharapkan: seluruh build & test backend memenuhi kriteria, lint (0 error) & build frontend sukses (exit 0), `tsc --noEmit` tanpa error.

**Observasi historis (non-authoritative) (18 Agu 2026):** `go build ./...` NOT RUN, `go test ./... -count=1` NOT RUN 2 test flaky `TestCreateAPIKey_Success` & `TestAPIKeyAuth_ValidKey` — "test: timeout error 1000ms" (batas default 1 detik fiber `app.Test()` saat beban CPU; keduanya **NOT RUN saat dijalankan sendiri** 0.32s/0.55s — bukan kegagalan kode), `go vet ./...` NOT RUN. Frontend: `npm run lint` NOT RUN 0 error (77 warning lama), `npm run build` NOT RUN exit 0, `npx tsc --noEmit` NOT RUN tanpa error.

## 3. Skenario Uji (Blackbox)

> **Catatan legacy:** tabel di bawah adalah observasi historis/non-authoritative; jangan gunakan sebagai status formal. Isi `Formal scenario record` di atas dengan expected, actual, evidence, dan status formal.

### S1 - Grid streaming hanya memuat kamera sesuai halaman

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Buka `/monitor` dengan 35 CCTV streaming aktif | Counter header menampilkan "Menampilkan 1–12 dari 35 kamera aktif" (default pageSize 12) | NOT RUN |
| 2 | Cek DevTools Network / performa | Hanya ±12 `WHEPPlayer` yang dimuat pada halaman pertama; sisanya placeholder/lazy | NOT RUN |
| 3 | Klik "berikutnya" (next) halaman | Halaman 2 menampilkan kamera 13–24, dst. | NOT RUN |
| 4 | Pilih dropdown pageSize = 4 | Tampil 4 kamera per halaman; counter berubah "Menampilkan 1–4 dari N" | NOT RUN |

### S2 - Lazy loading saat scroll

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Buka `/monitor` (pageSize = Semua bila tersedia) | Hanya WHEPPlayer yang dekat viewport yang di-mount (placeholder/skeleton untuk yang jauh) | NOT RUN |
| 2 | Scroll ke bawah | WHEPPlayer di-mount saat sel mendekati/masuk viewport (loading skeleton dulu, lalu stream tampil) | NOT RUN |

### S3 - Pause / Resume stream per sel

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Klik ikon "⏸ Pause" pada sebuah sel stream | Area video berhenti (placeholder "Dijeda"), koneksi stream ditutup, tombol berubah menjadi "▶ Resume" | NOT RUN |
| 2 | Klik "▶ Resume" | Stream dimuat ulang (koneksi WHEP baru dibuat) dan video tampil kembali | NOT RUN |
| 3 | Klik tombol Pause/Resume tidak memicu membuka detail CCTV | Perilaku terpisah (stopPropagation) | NOT RUN |

### S4 - Tombol "+" tambah tugas di kalender

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Buka `/kalender` | Setiap sel hari menampilkan tombol "+" kecil saat di-hover | NOT RUN |
| 2 | Klik tombol "+" pada tanggal tertentu | Modal "Buat Tugas Pemeliharaan" terbuka dengan `scheduled_date` terisi tanggal yang diklik | NOT RUN |
| 3 | Pilih CCTV, isi action_taken (min 5 karakter), simpan | Toast sukses; kalender di-refresh; tugas baru muncul (dot) pada tanggal tersebut | NOT RUN |
| 4 | Klik tanggal yang sama | Detail hari menampilkan tugas baru | NOT RUN |

### S5 - Validasi & aksesibilitas modal kalender

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Buka modal, kosongkan action_taken | Error validasi muncul (tidak bisa submit) | NOT RUN |
| 2 | Tanpa memilih CCTV | Error validasi muncul | NOT RUN |
| 3 | Tekan tombol Escape | Modal tertutup | NOT RUN |
| 4 | Klik overlay (latar gelap) | Modal tertutup | NOT RUN |

### S6 - Filter periode kinerja teknisi

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Buka `/kinerja-teknisi` | Tombol filter: "7 Hari", "30 Hari", "3 Bulan", "1 Tahun", "Kustom"; default "30 Hari" | NOT RUN |
| 2 | Klik "7 Hari" | Data di-fetch dengan `?period=7d`; subtitle "Data kinerja dalam 7 hari terakhir" | NOT RUN |
| 3 | Klik "3 Bulan" | Data di-fetch dengan `?period=90d`; subtitle "Data kinerja dalam 90 hari terakhir" | NOT RUN |
| 4 | Klik "1 Tahun" | Data di-fetch dengan `?period=365d`; subtitle "Data kinerja dalam 365 hari terakhir" | NOT RUN |
| 5 | Klik "Kustom" | Muncul input tanggal mulai/sampai; fetch dengan `start_date`/`end_date` | NOT RUN |
| 6 | Amati loading saat ganti periode | Loading skeleton muncul saat re-fetch | NOT RUN |

### S7 - Backend filter periode

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | `GET /api/dashboard/technician-performance?period=7d` (admin) | Data hanya untuk 7 hari terakhir | NOT RUN |
| 2 | `GET /api/dashboard/technician-performance?period=365d` | Data untuk 1 tahun terakhir | NOT RUN |
| 3 | Tanpa `period` | Default 30 hari terakhir (atau start_date/end_date jika diberikan) | NOT RUN |
| 4 | Tanpa login | 401 | NOT RUN |
| 5 | Login sebagai teknisi | 403 (hanya admin) | NOT RUN |

## 4. Kriteria Penerimaan

1. Grid `/monitor` memuat maksimal `pageSize` WHEPPlayer per halaman (default 12); sisanya lazy-load saat scroll.
2. Tombol Pause menghentikan stream (koneksi ditutup) tanpa menutup halaman; Resume memuat ulang.
3. Klik "+" pada hari di kalender membuka modal create maintenance dengan tanggal pre-filled; submit sukses → tugas muncul di kalender tanpa reload penuh.
4. Filter periode 7 Hari / 30 Hari / 3 Bulan / 1 Tahun / Kustom di `/kinerja-teknisi` berfungsi; data mengikuti periode yang dipilih.
5. `GET /api/dashboard/technician-performance?period=7d` mengembalikan data 7 hari terakhir.
6. `go build`/`go test`/`go vet` backend memenuhi kriteria; `npm run lint`/`npm run build`/`npx tsc --noEmit` frontend memenuhi kriteria.

## Catatan

- Lazy loading memakai `IntersectionObserver` (rootMargin 400px) — stream dimuat sedikit sebelum sel terlihat agar tidak kedap-kedip saat scroll.
- Pause menghilangkan WHEPPlayer dari DOM (memutus koneksi); Resume me-mount ulang untuk membuat koneksi baru. Perilaku ini setara dengan menutup/mereset `RTCPeerConnection`.
- Backend `period` (7d/30d/90d/365d) mengesampingkan `start_date`/`end_date` bila diberikan; jika `period` kosong/tidak valid, perilaku existing (start_date/end_date, default 30 hari) tetap berlaku.
- Chart `TechnicianPerformanceChart` adalah bar chart per teknisi (bukan grafik waktu), sehingga tidak memiliki label axis tanggal; subtitle periode ditampilkan di judul halaman.
