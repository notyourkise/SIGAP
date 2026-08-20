# Manual Test — Streaming CCTV Real-Time (Fase 9)

> **Kontrak uji formal — MT-21**
>
> Dokumen ini mendefinisikan prosedur; hasil sebelum **2026-08-20** adalah catatan historis/non-authoritative dan harus diuji ulang dengan execution record formal. Inspeksi kode, review unit test, atau checkmark lama bukan `PASS` black-box.
>
> **Skenario audit reproduktif:** Dengan simulator atau environment streaming yang reachable dan credential placeholder non-production, uji start, readiness, playback, stop, SD/HD, reconnect, bulk action, dan role.
>
> **Status suite:** `NOT RUN`

> **Supporting evidence Task 1B — 2026-08-20:** sub-gate simulator/staging lulus untuk registrasi path, source `available/online`, WHEP, ICE/TCP, decoded frame 640×360, stop idempotent, dan recovery restart. Suite agregat MT-21 tetap `NOT RUN` karena SD/HD, grid aplikasi, bulk action, RBAC lengkap, production UDP, dan mobile eksternal belum dijalankan. Evidence: `audits/TASK-1B-STREAMING-RUNTIME-EVIDENCE-2026-08-20.md`.

### Formal execution metadata

| Field | Value |
|---|---|
| Test ID / Audit ID | MT-21 |
| Capability owner | QA Streaming + Infra/Frontend/Backend |
| Status suite | NOT RUN |
| Build SHA | NOT RECORDED |
| Artifact digest/ID | NOT RECORDED |
| Tester | NOT RECORDED |
| Timestamp WITA | NOT RECORDED |
| Environment | NOT RECORDED |
| Fixture prefix/data | `E2E-MT21-` / NOT RECORDED |
| Cleanup result | NOT RUN |
| Expected | Start, readiness, playback, stop, SD/HD, reconnect, bulk action, dan role enforcement terbukti tanpa response 5xx. |
| Actual | NOT RECORDED |
| Evidence path/link | NOT RECORDED |
| Mandatory flag | YES |
| Scenario status | NOT RUN |

### Formal scenario record

| ID | Mandatory | Expected | Actual | Evidence | Status |
|---|---|---|---|---|---|
| MT-21-S1 | YES | Start, readiness, playback, stop, SD/HD, reconnect, bulk action, dan role enforcement terbukti tanpa response 5xx. | NOT RECORDED | NOT RECORDED | NOT RUN |

> **Cakupan aggregate Wave 0:** Baris formal aggregate di atas adalah **sole atomic suite-level acceptance gate** untuk baseline Wave 0. Skenario legacy terperinci di bawah ini tetap non-formal dan historical/non-authoritative sampai diperluas selama Wave 7.

**Aturan kesimpulan suite:** sebelum eksekusi suite `NOT RUN`; setelah run dimulai, setiap mandatory scenario harus `PASS`. Satu mandatory scenario berstatus `FAIL`, `BLOCKED`, atau `NOT RUN` membuat kesimpulan suite `FAIL`.

> **Guard keselamatan wajib:** Untuk setiap SQL mutasi, `DELETE`/`TRUNCATE`, rotasi/regenerasi key atau QR, penghapusan, reset kredensial, atau mutasi credential, lanjut hanya bila environment bukan production dan target bukan `nexaspace.site`, fixture prefix `E2E-MT21-` serta exact targets sudah dicatat, backup/recovery terverifikasi untuk perubahan data/key state, dan cleanup/rollback didefinisikan serta dicatat setelah run. Jika salah satu syarat tidak tersedia, hentikan prosedur dan set scenario status ke `BLOCKED`; jangan jalankan pada production. Jangan menyalin credential, token, private key, RTSP password, atau payload production yang belum disanitasi.


Dokumen ini memverifikasi fitur **streaming CCTV real-time** secara end-to-end (Fase 9), mencakup: konfigurasi streaming pada Master CCTV (backend `utils/rtsp.go`, `utils/mediamtx.go`, `handlers/stream.go`, rute `/api/streams/*` admin-only), start/stop streaming individual, player WebRTC (`WHEPPlayer.tsx` dengan sinyal WHEP `POST /stream/{path}/whep`), toggle kualitas SD/HD, grid view, role enforcement, reconnect saat NVR down, serta start-all streams. Env yang dipakai: `RTSP_NVR_HOST=<NVR_HOST_NON_PROD>`, `RTSP_NVR_PORT=554`, `RTSP_NVR_USER=<NVR_USER_NON_PROD>`, `MEDIAMTX_API_URL=http://mediamtx:9997`. MediaMTX berjalan pada RTSP `:8554`, WebRTC `:8889`/`:8189` (UDP), HLS `:8888`, REST `:9997`; Nginx mem-proxy `/stream/*` → `mediamtx:8889` dan `/hls/*` → `mediamtx:8888`.

## Prasyarat

- Server backend berjalan (`go run .` atau Docker / `docker compose up -d --build`)
- MediaMTX service berjalan (`docker compose up mediamtx`)
- NVR Hikvision dapat diakses di `<NVR_HOST_NON_PROD>:554`
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
| S1 | Konfigurasi streaming CCTV | Buka **Aset CCTV** (`/cctv`), klik edit pada CCTV, isi **Channel Port** 1–35, isi **Password RTSP**, centang **"Aktifkan Streaming"**, lalu simpan | Simpan sukses; kolom **Stream** di tabel menampilkan **ikon hijau** (MonitorPlay) untuk CCTV `stream_enabled=true`; `rtsp_password` **tidak bocor** di response (field tidak tampil / kosong pada re-open form) | NOT RUN — `PUT /api/cctv/:id {"stream_enabled":true,"channel_port":3,"rtsp_password":"..."}` → 200; respons CCTV tidak menyertakan `rtsp_password` | NOT RUN |
| S2 | Start streaming individual | Buka **Peta Sebaran** (`/sebaran-cctv`), klik marker CCTV dengan `stream_enabled=true`, klik **"Mulai Stream"** di `StreamModal` | Video **live** tampil via `WHEPPlayer` dalam **5–10 detik** dengan indikator **🔴 LIVE** | NOT RUN — historical start observation: start → 200: WHEP URL `/stream/cam-.../whep`, source `rtsp://<NVR_USER_NON_PROD>:<NVR_PASSWORD_NON_PROD>@<NVR_HOST_NON_PROD>:554/Streaming/channels/302` (sub-stream); NOT RUN video tidak tampil karena NVR tidak reachable dari VPS | NOT RUN |
| S3 | Stop streaming | Di `StreamModal`, klik **"Hentikan"** | Video berhenti; status menjadi **"Stream tidak aktif"** | NOT RUN — `POST /api/streams/:id/stop` → 502 "Gagal menghubungi server streaming" (konsisten 2×); `stop-all` → `failed:1` | NOT RUN |
| S4 | Toggle kualitas SD/HD | Saat streaming aktif, klik badge **SD → HD** | Stream di-restart dengan resolusi lebih tinggi (**main stream**) | NOT RUN — endpoint `GET /api/streams/:id/url` → 200 (`stream_active:true`, `stream_ready:false`); toggle SD/HD tidak dapat dibuktikan karena stream tidak pernah ready (NVR unreachable) | NOT RUN |
| S5 | Grid View | Klik toggle **"Grid"** di header | Semua CCTV streaming tampil dalam grid (`StreamGridView`); video masing-masing **playing** | NOT RUN — Tidak dapat diuji (butuh browser + NVR reachable); struktur grid & ring status terverifikasi dari kode `StreamGridView.tsx` | NOT RUN |
| S6 | Role enforcement | Login sebagai **teknisi**, akses `/api/streams/status` via browser | **403 Forbidden** (hanya admin yang dapat mengakses `/api/streams/*`) | NOT RUN — teknisi 403, tanpa sesi 401 | NOT RUN |
| S7 | Reconnect saat NVR down | Mulai stream, simulasi NVR unreachable, tunggu | Player menampilkan **"Reconnecting..."** dengan backoff; pulih otomatis saat NVR kembali | NOT RUN — Tidak dapat diuji (NVR `<NVR_HOST_NON_PROD>` unreachable sejak awal → player tidak pernah live; logika reconnect terverifikasi di `WHEPPlayer.tsx`) | NOT RUN |
| S8 | Start All Streams | Panggil `POST /api/streams/start-all` | Semua CCTV `stream_enabled=true` terdaftar di MediaMTX; response berisi jumlah berhasil | NOT RUN — `{"success":1,"failed":0,"total":1}` | NOT RUN |
| S9 | CCTV tanpa streaming diklik | Klik marker CCTV dengan `stream_enabled=false` | Popup **info statis** muncul (bukan `StreamModal`) | NOT RUN — Tidak dapat diuji (butuh browser); logika pemilihan popup/StreamModal terverifikasi dari kode | NOT RUN |

## Kriteria Penerimaan

- Semua skenario S1–S9 berstatus **NOT RUN**
- Tidak ada error di console browser
- Tidak ada goroutine leak di backend

## Catatan & Gotchas

- WebRTC memerlukan **HTTPS** atau **localhost** (HTTP biasa tidak bisa — insecure context)
- UDP port **8189** harus accessible untuk media WebRTC production; staging terisolasi dapat memakai listener ICE/TCP loopback melalui tunnel.
- Jika stream tidak muncul, cek: 1) NVR reachable, 2) encoding **H.264**, 3) MediaMTX logs
- `rtsp_password` disimpan **terenkripsi (AES-256)** di database dan **tidak pernah dikembalikan** oleh API; pada `StartStream` password didekripsi internal lalu hanya dipakai untuk membangun URL RTSP (`utils/rtsp.go` — BuildRTSPURL/BuildStreamPath/GetRTSPPort, StreamType Main/Sub)
- `/stream/*` dan `/hls/*` di-proxy oleh Nginx ke `mediamtx` (`/stream/*` → `mediamtx:8889`, `/hls/*` → `mediamtx:8888`); pastikan MediaMTX reachable baik dari backend (`MEDIAMTX_API_URL=http://mediamtx:9997`) maupun dari browser agar video live tampil
- Seluruh operasi stream bersifat **admin-only** (`RequireRole("admin")` pada rute `/api/streams/*`)
- Endpoint backend terkait: `POST /api/streams/:id/start`, `POST /api/streams/:id/stop`, `POST /api/streams/start-all`, `POST /api/streams/stop-all`, `GET /api/streams/status`, `GET /api/streams/:id/url` (mengembalikan `whep_url` `/stream/{path}/whep` dan `hls_url` `/hls/{path}/index.m3u8`)
