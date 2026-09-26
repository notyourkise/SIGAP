# Manual Test — Streaming CCTV Real-Time (Fase 9)

Dokumen ini memverifikasi fitur **streaming CCTV real-time** secara end-to-end (Fase 9), mencakup: konfigurasi streaming pada Master CCTV (backend `utils/rtsp.go`, `utils/mediamtx.go`, `handlers/stream.go`, rute `/api/streams/*` admin-only), start/stop streaming individual, player WebRTC (`WHEPPlayer.tsx` dengan sinyal WHEP `POST /stream/{path}/whep`), toggle kualitas SD/HD, grid view, role enforcement, reconnect saat NVR down, serta start-all streams. Env yang dipakai: `RTSP_NVR_HOST=36.91.27.35`, `RTSP_NVR_PORT=554`, `RTSP_NVR_USER=admin`, `MEDIAMTX_API_URL=http://mediamtx:9997`. MediaMTX berjalan pada RTSP `:8554`, WebRTC `:8889`/`:8189` (UDP), HLS `:8888`, REST `:9997`; Nginx mem-proxy `/stream/*` → `mediamtx:8889` dan `/hls/*` → `mediamtx:8888`.

## Prasyarat

- Server backend berjalan (`go run .` atau Docker / `docker compose up -d --build`)
- MediaMTX service berjalan (`docker compose up mediamtx`)
- NVR Hikvision dapat diakses di `36.91.27.35:554`
- Minimal 1 CCTV di database dengan `channel_port` terisi dan `stream_enabled=true`
- Login sebagai **admin**

## Pengujian Otomatis

```
cd backend && go build ./... && go test ./... -count=1 && go vet ./...
cd frontend && npm run lint && npm run build
docker compose config --quiet
```

## Skenario Uji

| ID | Deskripsi | Langkah-Langkah | Hasil Diharapkan | Hasil Aktual | Status |
|----|-----------|-----------------|------------------|--------------|--------|
| S1 | Konfigurasi streaming CCTV | Buka **Aset CCTV** (`/cctv`), klik edit pada CCTV, isi **Channel Port** 1–35, isi **Password RTSP**, centang **"Aktifkan Streaming"**, lalu simpan | Simpan sukses; kolom **Stream** di tabel menampilkan **ikon hijau** (MonitorPlay) untuk CCTV `stream_enabled=true`; `rtsp_password` **tidak bocor** di response (field tidak tampil / kosong pada re-open form) | ✅ — `PUT /api/cctv/:id {"stream_enabled":true,"channel_port":3,"rtsp_password":"..."}` → 200; respons CCTV tidak menyertakan `rtsp_password` | ✅ |
| S2 | Start streaming individual | Buka **Peta Sebaran** (`/sebaran-cctv`), klik marker CCTV dengan `stream_enabled=true`, klik **"Mulai Stream"** di `StreamModal` | Video **live** tampil via `WHEPPlayer` dalam **5–10 detik** dengan indikator **🔴 LIVE** | ✅ start → 200: WHEP URL `/stream/cam-.../whep`, source `rtsp://admin:...@36.91.27.35:554/Streaming/channels/302` (sub-stream); ⚠️ video tidak tampil karena NVR tidak reachable dari VPS | ⚠️ |
| S3 | Stop streaming | Di `StreamModal`, klik **"Hentikan"** | Video berhenti; status menjadi **"Stream tidak aktif"** | ❌ — `POST /api/streams/:id/stop` → 502 "Gagal menghubungi server streaming" (konsisten 2×); `stop-all` → `failed:1` | ❌ |
| S4 | Toggle kualitas SD/HD | Saat streaming aktif, klik badge **SD → HD** | Stream di-restart dengan resolusi lebih tinggi (**main stream**) | ⚠️ — endpoint `GET /api/streams/:id/url` → 200 (`stream_active:true`, `stream_ready:false`); toggle SD/HD tidak dapat dibuktikan karena stream tidak pernah ready (NVR unreachable) | ⚠️ |
| S5 | Grid View | Klik toggle **"Grid"** di header | Semua CCTV streaming tampil dalam grid (`StreamGridView`); video masing-masing **playing** | ⚠️ Tidak dapat diuji (butuh browser + NVR reachable); struktur grid & ring status terverifikasi dari kode `StreamGridView.tsx` | ⚠️ |
| S6 | Role enforcement | Login sebagai **teknisi**, akses `/api/streams/status` via browser | **403 Forbidden** (hanya admin yang dapat mengakses `/api/streams/*`) | ✅ — teknisi 403, tanpa sesi 401 | ✅ |
| S7 | Reconnect saat NVR down | Mulai stream, simulasi NVR unreachable, tunggu | Player menampilkan **"Reconnecting..."** dengan backoff; pulih otomatis saat NVR kembali | ⚠️ Tidak dapat diuji (NVR `36.91.27.35` unreachable sejak awal → player tidak pernah live; logika reconnect terverifikasi di `WHEPPlayer.tsx`) | ⚠️ |
| S8 | Start All Streams | Panggil `POST /api/streams/start-all` | Semua CCTV `stream_enabled=true` terdaftar di MediaMTX; response berisi jumlah berhasil | ✅ — `{"success":1,"failed":0,"total":1}` | ✅ |
| S9 | CCTV tanpa streaming diklik | Klik marker CCTV dengan `stream_enabled=false` | Popup **info statis** muncul (bukan `StreamModal`) | ⚠️ Tidak dapat diuji (butuh browser); logika pemilihan popup/StreamModal terverifikasi dari kode | ⚠️ |

## Kriteria Lulus

- Semua skenario S1–S9 berstatus **PASS**
- Tidak ada error di console browser
- Tidak ada goroutine leak di backend

## Catatan & Gotchas

- WebRTC memerlukan **HTTPS** atau **localhost** (HTTP biasa tidak bisa — insecure context)
- UDP port **8189** harus accessible untuk media WebRTC
- Jika stream tidak muncul, cek: 1) NVR reachable, 2) encoding **H.264**, 3) MediaMTX logs
- `rtsp_password` disimpan **terenkripsi (AES-256)** di database dan **tidak pernah dikembalikan** oleh API; pada `StartStream` password didekripsi internal lalu hanya dipakai untuk membangun URL RTSP (`utils/rtsp.go` — BuildRTSPURL/BuildStreamPath/GetRTSPPort, StreamType Main/Sub)
- `/stream/*` dan `/hls/*` di-proxy oleh Nginx ke `mediamtx` (`/stream/*` → `mediamtx:8889`, `/hls/*` → `mediamtx:8888`); pastikan MediaMTX reachable baik dari backend (`MEDIAMTX_API_URL=http://mediamtx:9997`) maupun dari browser agar video live tampil
- Seluruh operasi stream bersifat **admin-only** (`RequireRole("admin")` pada rute `/api/streams/*`)
- Endpoint backend terkait: `POST /api/streams/:id/start`, `POST /api/streams/:id/stop`, `POST /api/streams/start-all`, `POST /api/streams/stop-all`, `GET /api/streams/status`, `GET /api/streams/:id/url` (mengembalikan `whep_url` `/stream/{path}/whep` dan `hls_url` `/hls/{path}/index.m3u8`)
