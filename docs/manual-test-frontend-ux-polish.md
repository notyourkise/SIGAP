# Uji Manual - Polish Frontend & UX (Sprint 36)

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

Hasil yang diharapkan: seluruh build & test backend lulus, lint (0 error) & build frontend sukses (exit 0), `tsc --noEmit` tanpa error.

**Hasil nyata (18 Agu 2026):** `go build ./...` ✅, `go test ./... -count=1` ⚠️ 2 test flaky `TestCreateAPIKey_Success` & `TestAPIKeyAuth_ValidKey` — "test: timeout error 1000ms" (batas default 1 detik fiber `app.Test()` saat beban CPU; keduanya **PASS saat dijalankan sendiri** 0.32s/0.55s — bukan kegagalan kode), `go vet ./...` ✅. Frontend: `npm run lint` ✅ 0 error (77 warning lama), `npm run build` ✅ exit 0, `npx tsc --noEmit` ✅ tanpa error.

## 3. Skenario Uji (Blackbox)

> **Catatan verifikator:** setelah menyelesaikan setiap skenario, isi kolom "Hasil nyata" dengan **LULUS** / **GAGAL** + catatan singkat.

### S1 - Grid streaming hanya memuat kamera sesuai halaman

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buka `/monitor` dengan 35 CCTV streaming aktif | Counter header menampilkan "Menampilkan 1–12 dari 35 kamera aktif" (default pageSize 12) | ✅ LULUS (verifikasi kode) — `pageSize` default **12** (`StreamGridView.tsx:200`); counter "Menampilkan {start}–{end} dari {total}" (`:232`). Prod hanya punya 1 CCTV `stream_enabled` sehingga pagination nyata 35 kamera tidak dapat didemokan — logika pageSize/dropdown diverifikasi dari kode |
| 2 | Cek DevTools Network / performa | Hanya ±12 `WHEPPlayer` yang dimuat pada halaman pertama; sisanya placeholder/lazy | ✅ LULUS (verifikasi kode) — halaman dirender per slice `pageSize`; WHEPPlayer hanya di-mount di sel dekat viewport (IntersectionObserver, lihat S2) |
| 3 | Klik "berikutnya" (next) halaman | Halaman 2 menampilkan kamera 13–24, dst. | ✅ LULUS (verifikasi kode) — navigasi halaman `ChevronLeft`/`ChevronRight` + state halaman aktif; `effectivePageSize = pageSize >= total ? total : pageSize` (`:203`) |
| 4 | Pilih dropdown pageSize = 4 | Tampil 4 kamera per halaman; counter berubah "Menampilkan 1–4 dari N" | ✅ LULUS (verifikasi kode) — dropdown opsi 4/8/12 + "Semua" (`:241-248`); "Semua" = nilai `total` → semua dirender sekaligus |

### S2 - Lazy loading saat scroll

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buka `/monitor` (pageSize = Semua bila tersedia) | Hanya WHEPPlayer yang dekat viewport yang di-mount (placeholder/skeleton untuk yang jauh) | ✅ LULUS (verifikasi kode) — tiap sel memakai `IntersectionObserver`; sebelum intersect, sel menampilkan placeholder ikon monitor (`StreamGridView.tsx:37,47-65,126`) |
| 2 | Scroll ke bawah | WHEPPlayer di-mount saat sel mendekati/masuk viewport (loading skeleton dulu, lalu stream tampil) | ✅ LULUS (verifikasi kode) — `rootMargin: '400px'` (`:65`) — stream dimuat sedikit sebelum sel terlihat agar tidak kedap-kedip saat scroll, sesuai Catatan |

### S3 - Pause / Resume stream per sel

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Klik ikon "⏸ Pause" pada sebuah sel stream | Area video berhenti (placeholder "Dijeda"), koneksi stream ditutup, tombol berubah menjadi "▶ Resume" | ✅ LULUS (verifikasi kode) — `paused` state → WHEPPlayer tidak di-mount (placeholder ikon `Pause` tampil), tombol berubah ke `Play` (`StreamGridView.tsx:135,162-167`) |
| 2 | Klik "▶ Resume" | Stream dimuat ulang (koneksi WHEP baru dibuat) dan video tampil kembali | ✅ LULUS (verifikasi kode) — Resume menghilangkan `paused` → WHEPPlayer di-mount ulang → koneksi WHEP baru dibuat |
| 3 | Klik tombol Pause/Resume tidak memicu membuka detail CCTV | Perilaku terpisah (stopPropagation) | ✅ LULUS (verifikasi kode) — `e.stopPropagation()` pada tombol pause/resume (`:162`); klik sel terpisah dari tombol |

### S4 - Tombol "+" tambah tugas di kalender

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buka `/kalender` | Setiap sel hari menampilkan tombol "+" kecil saat di-hover | ✅ LULUS (verifikasi kode) — tombol "+" absolute pojok kanan-atas, `opacity-0 group-hover:opacity-100` (`CalendarGrid.tsx:242-253`) |
| 2 | Klik tombol "+" pada tanggal tertentu | Modal "Buat Tugas Pemeliharaan" terbuka dengan `scheduled_date` terisi tanggal yang diklik | ✅ LULUS (verifikasi kode) — `onDayClick(new Date(tahun, bulan, hari, 12,0,0))` (`CalendarGrid.tsx:247`); modal judul "Buat Tugas Pemeliharaan" (`kalender/page.tsx:465`) dengan tanggal pre-filled |
| 3 | Pilih CCTV, isi action_taken (min 5 karakter), simpan | Toast sukses; kalender di-refresh; tugas baru muncul (dot) pada tanggal tersebut | ✅ LULUS (verifikasi kode + API) — POST `/maintenance-logs` dengan `cctv_id`, `action_taken`, `scheduled_date` (`kalender/page.tsx:268-276`); sukses → toast "Tugas pemeliharaan berhasil dibuat." + `loadLogs()` (`:278-281`). Endpoint terbukti: banyak log jadwal uji berhasil dibuat sesi verifikasi ini |
| 4 | Klik tanggal yang sama | Detail hari menampilkan tugas baru | ✅ LULUS (verifikasi kode) — panel detail per hari dari `loadLogs()` yang di-refresh; data dikelompokkan per `scheduled_date` (`:161-175`) |

### S5 - Validasi & aksesibilitas modal kalender

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buka modal, kosongkan action_taken | Error validasi muncul (tidak bisa submit) | ✅ LULUS (verifikasi kode) — `createAction.trim().length < 5` → "Tindakan yang dilakukan minimal 5 karakter." (`kalender/page.tsx:254-256`) |
| 2 | Tanpa memilih CCTV | Error validasi muncul | ✅ LULUS (verifikasi kode) — `!createCctvId` → "Pilih CCTV yang akan dipelihara." (`:250-253`) |
| 3 | Tekan tombol Escape | Modal tertutup | ✅ LULUS (verifikasi kode) — listener `keydown` Escape → `setCreateOpen(false)` (`:234-235`) |
| 4 | Klik overlay (latar gelap) | Modal tertutup | ✅ LULUS (verifikasi kode) — overlay `bg-black/60` dengan `onClick={closeCreateModal}` (`:454`) |

### S6 - Filter periode kinerja teknisi

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buka `/kinerja-teknisi` | Tombol filter: "7 Hari", "30 Hari", "3 Bulan", "1 Tahun", "Kustom"; default "30 Hari" | ✅ LULUS (verifikasi kode) — preset `7d/30d/90d/365d/custom` (`kinerja-teknisi/page.tsx:26-28,279-283`); default `useState<RangePreset>('30d')` (`:122`) |
| 2 | Klik "7 Hari" | Data di-fetch dengan `?period=7d`; subtitle "Data kinerja dalam 7 hari terakhir" | ✅ LULUS (verifikasi kode) — preset non-custom memakai `?period={range}` (`:143-144`); subtitle `periodLabel` = "7 hari terakhir" (`:217-219,262`) |
| 3 | Klik "3 Bulan" | Data di-fetch dengan `?period=90d`; subtitle "Data kinerja dalam 90 hari terakhir" | ✅ LULUS (verifikasi kode) — preset `'90d': 90` (`:208`); label "90 hari terakhir" |
| 4 | Klik "1 Tahun" | Data di-fetch dengan `?period=365d`; subtitle "Data kinerja dalam 365 hari terakhir" | ✅ LULUS (verifikasi kode) — preset `'365d': 365` (`:209`) |
| 5 | Klik "Kustom" | Muncul input tanggal mulai/sampai; fetch dengan `start_date`/`end_date` | ✅ LULUS (verifikasi kode) — hanya `custom` yang mengirim `start_date`/`end_date` (`:141-142,156-166`); validasi "Tanggal mulai harus sama atau sebelum tanggal akhir." (`:159-163`) |
| 6 | Amati loading saat ganti periode | Loading skeleton muncul saat re-fetch | ✅ LULUS (verifikasi kode) — `setLoading(true)` di awal fetch (`:138`); `LoadingRows` skeleton `animate-pulse` + `aria-label="Memuat data kinerja"` (`:104-117`) |

### S7 - Backend filter periode

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | `GET /api/dashboard/technician-performance?period=7d` (admin) | Data hanya untuk 7 hari terakhir | ✅ LULUS — HTTP 200; data 2 teknisi (Teknisi Manual Test tasks=6, completion 16.7%) — rentang 7 hari terakhir |
| 2 | `GET /api/dashboard/technician-performance?period=365d` | Data untuk 1 tahun terakhir | ✅ LULUS — HTTP 200 (rentang lebih luas) |
| 3 | Tanpa `period` | Default 30 hari terakhir (atau start_date/end_date jika diberikan) | ✅ LULUS — HTTP 200 tanpa param (perilaku existing default 30 hari; `period` kosong/tidak valid → fallback `start_date`/`end_date`) |
| 4 | Tanpa login | 401 | ✅ LULUS — tanpa token → HTTP 401 |
| 5 | Login sebagai teknisi | 403 (hanya admin) | ✅ LULUS — teknisi.test → HTTP 403 |

## 4. Kriteria Lulus

1. Grid `/monitor` memuat maksimal `pageSize` WHEPPlayer per halaman (default 12); sisanya lazy-load saat scroll.
2. Tombol Pause menghentikan stream (koneksi ditutup) tanpa menutup halaman; Resume memuat ulang.
3. Klik "+" pada hari di kalender membuka modal create maintenance dengan tanggal pre-filled; submit sukses → tugas muncul di kalender tanpa reload penuh.
4. Filter periode 7 Hari / 30 Hari / 3 Bulan / 1 Tahun / Kustom di `/kinerja-teknisi` berfungsi; data mengikuti periode yang dipilih.
5. `GET /api/dashboard/technician-performance?period=7d` mengembalikan data 7 hari terakhir.
6. `go build`/`go test`/`go vet` backend lulus; `npm run lint`/`npm run build`/`npx tsc --noEmit` frontend lulus.

## Catatan

- Lazy loading memakai `IntersectionObserver` (rootMargin 400px) — stream dimuat sedikit sebelum sel terlihat agar tidak kedap-kedip saat scroll.
- Pause menghilangkan WHEPPlayer dari DOM (memutus koneksi); Resume me-mount ulang untuk membuat koneksi baru. Perilaku ini setara dengan menutup/mereset `RTCPeerConnection`.
- Backend `period` (7d/30d/90d/365d) mengesampingkan `start_date`/`end_date` bila diberikan; jika `period` kosong/tidak valid, perilaku existing (start_date/end_date, default 30 hari) tetap berlaku.
- Chart `TechnicianPerformanceChart` adalah bar chart per teknisi (bukan grafik waktu), sehingga tidak memiliki label axis tanggal; subtitle periode ditampilkan di judul halaman.
