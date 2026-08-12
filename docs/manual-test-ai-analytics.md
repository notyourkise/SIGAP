# Uji Manual — Admin AI Analytics (Task 14C.3)

> **Task:** 14C.3 Dashboard AI Analytics
> **Branch:** `feature/ai-frontend-overlay`
> **Versi:** backend `0.62.0` (Sprint 39)
> **Dokumen:** verifikasi halaman admin `/ai-analytics` — ringkasan hitungan orang unik hari ini per kamera (`GET /api/ai/counts`), tabel per kamera, dan grafik tren 7 hari (`GET /api/ai/counts/:id/trend?days=7`) yang dirender sebagai SVG mandiri (tanpa dependency chart eksternal).

## 1. Prasyarat

| # | Item | Cara menyiapkan |
|---|------|-----------------|
| P1 | Stack berjalan | `docker compose up -d --build` — pastikan container `sigap_ai`, `sigap_api`, `sigap_frontend` berstatus Up/healthy |
| P2 | Backend versi 0.62.0 | Endpoint tren `GET /api/ai/counts/:id/trend` tersedia (ditambahkan di `handlers/ai.go` & `routes/ai.go`) |
| P3 | Frontend versi Sprint 39 | Halaman `/ai-analytics` ada di `frontend/src/app/ai-analytics/page.tsx`; item menu sidebar "AI Analytics" (ikon `BrainCircuit`) tampil di grup "Pelaporan & Analisis" |
| P4 | Data deteksi AI | Minimal 1–2 CCTV memiliki sesi di `ai_detection_sessions` hari ini (dari deteksi AI di `/monitor` atau injeksi manual ke DB) |
| P5 | Data tren | Beberapa baris `ai_detection_sessions` dengan `session_date` berbeda (mis. 2–7 hari terakhir) agar grafik tren terisi |
| P6 | Akun admin & teknisi | Login sebagai **admin** untuk S1–S4/S6/S7; login sebagai **teknisi** untuk S5 |
| P7 | Browser + DevTools | Chrome/Edge; tab Network untuk cek request `/api/ai/counts` & `/api/ai/counts/:id/trend` |

## 2. Pengujian Otomatis

```bash
# Backend — build, test (termasuk tren), vet, lint
cd backend
go build ./... && go test ./... -count=1 -run "TestGetAICountTrend|TestAI" && go vet ./...
golangci-lint run ./...   # 0 issues

# Frontend — tipe, lint, build
cd ../frontend
npx tsc --noEmit && npm run lint && npm run build
```

Hasil yang diharapkan: `go build`/`go test`/`go vet`/`golangci-lint` lulus (0 issues), `tsc` bersih (0 error), `npm run lint` 0 error, `npm run build` sukses dan route `/ai-analytics` terdaftar.

## 3. Skenario Uji (Blackbox)

> **Catatan verifikator:** setelah menyelesaikan setiap skenario, isi kolom "Hasil nyata" dengan **LULUS** / **GAGAL** + catatan singkat.

### S1 - Halaman /ai-analytics hanya untuk admin

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Login sebagai **admin**, buka `/ai-analytics` | Halaman tampil normal (HTTP 200) dengan judul "AI Analytics", kartu ringkasan, tabel per kamera, dan section tren | |
| 2 | Item menu sidebar "AI Analytics" (ikon `BrainCircuit`) di grup "Pelaporan & Analisis" | Item menu tampil hanya untuk admin | |
| 3 | Login sebagai **teknisi**, buka `/ai-analytics` | Redirect ke `/dashboard` (proxy admin path memblokir non-admin; tanpa sesi → redirect ke `/login`) | |

### S2 - Kartu ringkasan (Total / Kamera / Aktif)

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Dengan data sesi hari ini tersedia, buka `/ai-analytics` | Kartu "Total Orang Hari Ini" = jumlah `unique_today` seluruh kamera | |
| 2 | Cek kartu "Jumlah Kamera Dipantau" | = jumlah baris data `/api/ai/counts` (kamera dengan sesi hari ini) | |
| 3 | Cek kartu "Kamera dengan Hitungan > 0" | = jumlah kamera dengan `unique_today > 0` | |
| 4 | Bandingkan dengan DB: `SELECT cctv_id, unique_person_count FROM ai_detection_sessions WHERE session_date = CURRENT_DATE` | Nilai kartu konsisten dengan DB | |

### S3 - Tabel per kamera — hari ini

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Periksa tabel "Per Kamera — Hari Ini" | Kolom Nama, Kode CCTV, Orang Unik, Terakhir Diperbarui terisi sesuai data | |
| 2 | Kamera dengan `unique_today > 0` | Badge hijau "terdeteksi" muncul di kolom Orang Unik | |
| 3 | Kolom Terakhir Diperbarui | Menampilkan tanggal `session_date` (atau placeholder "Hari ini" bila tidak ada) | |

### S4 - Dropdown kamera + grafik tren 7 hari

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Di section "Tren 7 Hari", pilih kamera dari dropdown | Request `GET /api/ai/counts/{id}/trend?days=7` terkirim (cek tab Network) | |
| 2 | Grafik garis SVG tampil | Sumbu X = tanggal (7 hari terakhir), sumbu Y = jumlah orang unik; titik data & garis tren terlihat | |
| 3 | Hover pada titik data | Tooltip menampilkan tanggal + jumlah orang unik | |
| 4 | Label "Rentang 7 Hari" | Badge rentang tampil di kanan atas section tren | |
| 5 | Ganti pilihan kamera | Grafik memuat ulang dengan data kamera yang baru dipilih | |
| 6 | Kamera tanpa data tren | Tampil state kosong "Belum ada data tren untuk kamera ini." | |

### S5 - Role enforcement

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Teknisi membuka `/ai-analytics` langsung (ketik URL) | Redirect ke `/dashboard` (proxy: role != admin → dashboard) | |
| 2 | Teknisi panggil `GET /api/ai/counts` atau `GET /api/ai/counts/{id}/trend` via DevTools/curl | Respons **403 Forbidden** (endpoint admin-only di `routes/ai.go`) | |
| 3 | Tanpa sesi, buka `/ai-analytics` | Redirect ke `/login` (proxy memblokir tanpa cookie sesi) | |

### S6 - Data kosong

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Kosongkan `ai_detection_sessions` (atau set `session_date` ≠ hari ini), buka `/ai-analytics` sebagai admin | State kosong "Belum ada data AI hari ini" tampil; kartu ringkasan menampilkan 0 | |
| 2 | Periksa respons `/api/ai/counts` | HTTP 200 dengan `data: []` | |

### S7 - Verifikasi endpoint trend

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | `GET /api/ai/counts/{uuid}/trend?days=7` (admin) | HTTP 200 `{status:"success", data:[{date, unique_count}, ...]}` — diurutkan ascending berdasarkan `session_date` | |
| 2 | `GET /api/ai/counts/{uuid}/trend?days=0` | `days` di-clamp ke 1 (masih 200) | |
| 3 | `GET /api/ai/counts/{uuid}/trend?days=999` | `days` di-clamp ke 90 (masih 200) | |
| 4 | `GET /api/ai/counts/bukan-uuid/trend?days=7` | HTTP **400** `{"status":"error","message":"ID tidak valid"}` | |
| 5 | Teknisi panggil endpoint trend | HTTP **403 Forbidden** | |

## 4. Kriteria Lulus

1. Halaman `/ai-analytics` hanya dapat diakses admin (teknisi → redirect dashboard; tanpa sesi → login).
2. Kartu ringkasan (total orang, jumlah kamera, kamera aktif) akurat terhadap data `/api/ai/counts`.
3. Tabel "Per Kamera — Hari Ini" menampilkan nama, kode CCTV, orang unik, dan waktu terakhir diperbarui.
4. Dropdown kamera + grafik tren 7 hari berfungsi; data diambil dari `GET /api/ai/counts/:id/trend?days=7`.
5. State loading (spinner), error, dan kosong ("Belum ada data AI hari ini") ditangani dengan baik.
6. Endpoint tren memvalidasi UUID (400), clamp `days` 1–90, dan hanya untuk admin (403 untuk teknisi).
7. `go build`/`go test`/`go vet`/`golangci-lint` lulus; `npx tsc --noEmit`/`npm run lint`/`npm run build` lulus.

## Catatan

- **Data sumber**: ringkasan hari ini dari `ai_detection_sessions` (di-upsert oleh `ReceiveDetectionEvent` setiap deteksi); tren 7 hari juga dari tabel yang sama, di-group per `session_date`.
- **Grafik**: `AITrendChart` di dalam `frontend/src/app/ai-analytics/page.tsx` — SVG mandiri (line + area + tooltip hover) tanpa library chart eksternal, konsisten dengan `TrendChart.tsx`.
- **Keamanan berlapis**: proxy (`frontend/src/proxy.ts`) memblokir non-admin di level middleware; backend (`routes/ai.go`) menerapkan `RequireRole("admin")` pada `/ai/counts`, `/ai/counts/:id`, dan `/ai/counts/:id/trend`.
- **Endpoint terkait**: `GET /api/ai/counts` (ringkasan) · `GET /api/ai/counts/:id` (detail) · `GET /api/ai/counts/:id/trend?days=1..90` (tren, baru — Task 14C.3).
