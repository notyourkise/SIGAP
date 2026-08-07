# Uji Manual - Streaming CCTV Real-Time

> **Task:** 9.3 Backend API & MediaMTX Client (Sprint 21, Fase 9 — Streaming CCTV)
> **Branch:** `feature/stream-api`
> **Versi:** backend `0.44.0`
> **Dokumen:** verifikasi backend streaming CCTV — client MediaMTX (`utils/mediamtx.go`) dan handler/rute stream (`handlers/stream.go`, `routes/stream.go`) — mulai/menghentikan stream, status & URL stream, serta enforcement akses **admin-only**.

Fokus uji ini adalah **backend API stream via Swagger/curl + otorisasi admin**. Player/UI penuh (komponen `WHEPPlayer`, modal stream, peta & grid view) hadir di **Sprint 22** (`feature/stream-frontend`); dokumen ini tidak menguji tampilan player tersebut.

## 1. Prasyarat

| # | Item | Cara menyiapkan |
|---|------|-----------------|
| P1 | Aplikasi berjalan | `docker compose up -d --build`, buka `https://localhost` |
| P2 | Konfigurasi MediaMTX | VAPID/RTSP tidak dibutuhkan untuk uji API ini, tetapi pastikan env `MEDIAMTX_API_URL` diset (mis. `http://mediamtx:9997/v3/`) dan service `mediamtx` **healthy** |
| P3 | Akun admin | Login sebagai **admin** untuk mengakses endpoint `/streams` |
| P4 | Akun teknisi | Login sebagai **teknisi** untuk menguji enforcement role (harus ditolak 403) |
| P5 | CCTV dengan konfigurasi streaming | Minimal 1 CCTV ber-`stream_enabled=true` lengkap dengan `channel_port` & `rtsp_password` (diset via konfigurasi streaming Master CCTV dari Sprint 20) |
| P6 | Akses Swagger | Buka `https://localhost/swagger/index.html` (atau `http://localhost:3000/swagger/index.html` pada development tanpa Docker) |

## 2. Pengujian Otomatis

```powershell
cd backend
go build ./...
go test ./... -count=1
go vet ./...
golangci-lint run ./...
swag init
```

Hasil yang diharapkan: `go build` sukses, `go test` seluruh package PASS, `go vet` bersih, `golangci-lint` 0 issue, dan `swag init` meregenerasi dokumentasi (endpoint `/streams/*` muncul).

## 3. Skenario Uji (Blackbox)

> **Catatan verifikator:** setelah menyelesaikan setiap skenario, isi kolom "Hasil nyata" dengan **LULUS** / **GAGAL** + catatan singkat.
>
> Semua endpoint di bawah menggunakan grup `/api/streams` (prefix `/api`) dan **wajib admin** (`RequireRole("admin")`). Jalankan via Swagger atau curl dengan `Authorization: Bearer <token>`.

### S1 - Konfigurasi streaming di Master CCTV & `rtsp_password` tidak bocor

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Edit CCTV di Master CCTV, isi `channel_port` (1–35), `rtsp_password`, dan centang `stream_enabled=true`, lalu simpan | Simpan sukses (200) | |
| 2 | Panggil `GET /api/cctv/:id` (atau `GET /api/cctv`) | Respons **TIDAK** memuat field `rtsp_password` (tidak bocor ke response) | |
| 3 | Buka detail CCTV yang sama di UI | Field `rtsp_password` tidak tampil; `channel_port` & `stream_enabled` tampil sesuai | |

### S2 - `StartStream` sukses → return `whep_url`

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Login sebagai admin, dapatkan token | Token admin valid | |
| 2 | Panggil `POST /api/streams/:id/start` untuk CCTV dengan `stream_enabled=true` | Respons 200 berisi `whep_url` (mis. `/stream/{path}/whep`) | |
| 3 | Cek service MediaMTX (path list) | Path stream CCTV terdaftar di MediaMTX | |

### S3 - `StartStream` CCTV nonaktif streaming → 400

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Pilih CCTV dengan `stream_enabled=false` | | |
| 2 | Panggil `POST /api/streams/:id/start` | Respons **400** (CCTV tidak mengaktifkan streaming) | |

### S4 - `StartStream` tanpa `channel_port` → 400

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Pilih CCTV yang `stream_enabled=true` tetapi `channel_port` kosong/0 | | |
| 2 | Panggil `POST /api/streams/:id/start` | Respons **400** (channel port tidak valid / tidak terisi) | |

### S5 - `GetStreamStatus` → list dengan `stream_active`/`stream_ready`/`viewers`

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Setelah beberapa stream berjalan, panggil `GET /api/streams/status` | Respons 200 berisi daftar stream CCTV | |
| 2 | Periksa field per CCTV | Memuat `stream_active`, `stream_ready`, dan `viewers` (jumlah penonton) | |
| 3 | Bandingkan dengan data DB + info path MediaMTX | Status aktif sesuai path yang terdaftar di MediaMTX | |

### S6 - `GetStreamURL` → `whep_url` + `hls_url`

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Panggil `GET /api/streams/:id/url` untuk CCTV stream aktif | Respons 200 berisi `whep_url` dan `hls_url` | |
| 2 | Validasi format URL | `whep_url` menunjuk `/stream/{path}/whep`; `hls_url` menunjuk `/hls/{path}/index.m3u8` | |

### S7 - `StopStream` → path dihapus

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Panggil `POST /api/streams/:id/stop` untuk stream yang berjalan | Respons 200 sukses | |
| 2 | Cek status via `GET /api/streams/status` | Stream CCTV tidak lagi `stream_active` | |
| 3 | Cek path list MediaMTX | Path stream CCTV telah dihapus dari MediaMTX | |

### S8 - `StartAllStreams` / `StopAllStreams`

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Panggil `POST /api/streams/start-all` | Respons 200; seluruh CCTV `stream_enabled=true` ter-register di MediaMTX | |
| 2 | Panggil `GET /api/streams/status` | Seluruh CCTV streaming aktif tampil | |
| 3 | Panggil `POST /api/streams/stop-all` | Respons 200; seluruh path stream dihapus dari MediaMTX | |
| 4 | Panggil `GET /api/streams/status` lagi | Tidak ada stream aktif | |

### S9 - Role enforcement — teknisi dapat 403, admin sukses

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Login sebagai **teknisi**, dapatkan token | Token teknisi valid | |
| 2 | Panggil salah satu endpoint `/streams` (mis. `GET /api/streams/status`, `POST /api/streams/:id/start`) | Respons **403** (hanya admin) | |
| 3 | Login kembali sebagai **admin** | Endpoint yang sama berhasil (200) | |

## 4. Kriteria Lulus

1. Seluruh endpoint grup `/streams` **admin-only** — akses sebagai teknisi selalu **403**.
2. `StartStream` pada CCTV `stream_enabled=true` mengembalikan `whep_url` dan mendaftarkan path di MediaMTX.
3. `StartStream` pada CCTV `stream_enabled=false` atau tanpa `channel_port` dikembalikan **400**.
4. `GetStreamStatus` mengembalikan `stream_active`/`stream_ready`/`viewers` yang sinkron dengan MediaMTX.
5. `GetStreamURL` mengembalikan `whep_url` + `hls_url` dengan format benar.
6. `StopStream` menghapus path dari MediaMTX.
7. `rtsp_password` **tidak pernah** muncul di response API manapun.
8. `go build` / `go test ./... -count=1` / `go vet` / `golangci-lint run ./...` / `swag init` semuanya PASS.

## Catatan

- `rtsp_password` disimpan **terenkripsi (AES-256)** di database dan di-exclude dari response JSON; pada `StartStream` password didekripsi internal lalu dipakai hanya untuk membangun URL RTSP.
- `whep_url`/`hls_url` memakai path stream (bukan URL absolut penuh); player (Sprint 22) akan menempelkannya pada base host via Nginx reverse proxy (`/stream/*` → `mediamtx:8889`, `/hls/*` → `mediamtx:8888`).
- MediaMTX harus **reachable** dari backend (`MEDIAMTX_API_URL` benar dan service `mediamtx` healthy) agar `StartStream`/`StopStream`/status bekerja.
- Seluruh operasi stream bersifat **admin-only** sesuai keputusan arsitektur streaming (akses viewer dibatasi admin).
- UI player/streaming penuh (klik marker → video live, toggle SD/HD, grid view) belum tercakup di sini; akan ditambahkan pada uji manual streaming tahap Sprint 22/23 setelah frontend tersedia.
