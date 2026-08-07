# Uji Manual - Streaming CCTV Real-Time

> **Task:** 9.3 (Sprint 21) & 9.4/9.5/9.6 (Sprint 22, Fase 9 — Streaming CCTV)
> **Branch:** `feature/stream-api` (backend) → `feature/stream-frontend` (frontend)
> **Versi:** backend `0.44.0` · frontend `1.36.0`
> **Dokumen:** verifikasi streaming CCTV end-to-end — (a) backend client MediaMTX (`utils/mediamtx.go`) & handler/rute stream (`handlers/stream.go`, `routes/stream.go`), serta (b) UI frontend (player `WHEPPlayer`, modal stream, grid view, peta sebaran, konfigurasi streaming Master CCTV).

Skenario **S1–S9** menguji **backend API stream via Swagger/curl + otorisasi admin** (Sprint 21). Skenario **S10–S15** menguji **UI frontend streaming** (Sprint 22) yang memakai komponen `WHEPPlayer`, `StreamModal`, `StreamGridView`, dan integrasi peta.

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

---

## 3.1 Skenario Uji Frontend (Sprint 22 — UI Streaming)

> **Catatan verifikator:** skenario di bawah menguji **UI frontend** (`feature/stream-frontend`). Prasyarat: login sebagai **admin**, minimal 1 CCTV ber-`stream_enabled=true` lengkap dengan `channel_port` & `rtsp_password`, dan **MediaMTX reachable** dari browser (lihat Catatan). Isi kolom "Hasil nyata" dengan **LULUS** / **GAGAL** + catatan singkat.

### S10 - Admin mengonfigurasi streaming di Master CCTV

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buka **Master CCTV** (`/cctv`) → klik ikon edit pada sebuah CCTV | Form "Ubah Data CCTV" terbuka, berisi bagian **Konfigurasi Streaming** | |
| 2 | Isi **Channel Port NVR** (1–35) | Field menerima angka; **Preview URI** diperbarui mengikuti `channel_port*100+2` (mis. port 2 → `.../channels/202`) | |
| 3 | Isi **Password RTSP**; uji tombol toggle (ikon mata) untuk menampilkan/menyembunyikan | Password dapat ditampilkan/disembunyikan | |
| 4 | Centang **Aktifkan Streaming**, lalu simpan | Simpan sukses; `channel_port` & `stream_enabled` tersimpan | |
| 5 | Buka kembali form edit CCTV yang sama | `channel_port` & `stream_enabled` terisi sesuai; **`rtsp_password` tampil kosong** (tidak pernah dikembalikan API) | |

### S11 - Klik marker CCTV streaming di Peta Sebaran → video live

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buka **Peta Sebaran** (`/sebaran-cctv`) | Marker CCTV tampil di peta | |
| 2 | Klik marker CCTV dengan `stream_enabled=true` | `StreamModal` terbuka | |
| 3 | Tekan tombol **Start** | Video **live** tampil via `WHEPPlayer` dengan indikator **🔴 LIVE** | |
| 4 | (Opsi) tekan **Stop** | Stream berhenti; tombol kembali ke **Start** | |

### S12 - Toggle kualitas SD/HD pada player

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Mulai streaming pada suatu CCTV (lihat S11) | Video live tampil | |
| 2 | Ganti kualitas ke **HD** | Stream di-restart dengan kualitas baru (URL channel utama) | |
| 3 | Ganti kualitas kembali ke **SD** | Stream di-restart ke sub stream (kualitas default) | |

### S13 - Toggle view Peta / Grid

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Di halaman Peta Sebaran, alihkan ke **Grid View** | `StreamGridView` menampilkan multi-stream CCTV yang aktif | |
| 2 | Periksa sel CCTV tanpa streaming | Sel menampilkan **placeholder** (bukan video) | |
| 3 | Alihkan kembali ke **Peta View** | Peta tampil kembali dengan marker | |

### S14 - Status bar & kolom ikon Stream

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Di Peta/Grid Sebaran, periksa status bar | Tampil teks "**N kamera streaming aktif**" sesuai jumlah CCTV streaming | |
| 2 | Buka **Master CCTV** (`/cctv`) | Kolom **Stream** pada tabel menampilkan ikon **MonitorPlay** | |
| 3 | Bandingkan ikon per baris | CCTV `stream_enabled=true` berikon **hijau**; yang `false` berikon **abu-abu** | |

### S15 - Guard role — teknisi tidak melihat UI streaming

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Login sebagai **teknisi** | Teknisi masuk ke dashboard (tanpa akses admin) | |
| 2 | Akses halaman streaming/peta sebaran yang memuat UI stream | UI streaming **tidak tampil** untuk teknisi (guard role); hanya admin yang dapat melihat/memutar stream | |
| 3 | Coba akses konfigurasi streaming di Master CCTV | Teknisi tidak dapat membuka/menyimpan konfigurasi streaming (guard role admin) | |

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
- MediaMTX harus **reachable** dari backend (`MEDIAMTX_API_URL` benar dan service `mediamtx` healthy) agar `StartStream`/`StopStream`/status bekerja. Untuk uji **end-to-end frontend (S10–S15)**, pastikan juga **MediaMTX dapat dijangkau dari browser** melalui Nginx reverse proxy (`/stream/*` → `mediamtx:8889`, `/hls/*` → `mediamtx:8888`) agar video live dapat ditampilkan pada player.
- Seluruh operasi stream bersifat **admin-only** sesuai keputusan arsitektur streaming (akses viewer dibatasi admin).
- UI player/streaming penuh (klik marker → video live, toggle SD/HD, grid view) kini tercakup pada skenario **S10–S15** di bawah (Sprint 22).
