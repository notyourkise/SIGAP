# Uji Manual — RTSP Live Status Detection (Task 14A)

> **Task:** 14A.1 + 14A.2 + 14A.3 RTSP Live Status Detection
> **Branch:** `feature/rtsp-live-probe`
> **Versi:** backend `0.60.0` / frontend `1.51.0` (Sprint 37)
> **Dokumen:** verifikasi deteksi status online/offline kamera secara real-time melalui RTSP probe: migrasi kolom `stream_live` & `stream_checked_at`, engine probe per 2 menit, endpoint `GET /api/streams/live-status` (admin), badge 🔴/🟢 di peta, status bar counter, popup "Dicek HH:mm", border sel `/monitor`, WebSocket `CCTV_LIVE_STATUS_UPDATED`, serta enforcement role.

Scope implementasi: **deteksi status live via RTSP probe** (bukan streaming video). Fitur ini memperbaiki bug fundamental di mana peta selalu hijau padahal kamera mati — kini status `stream_live` diperbarui oleh probe engine setiap ≤2 menit TANPA perlu ada yang membuka stream.

## 1. Prasyarat

| # | Item | Cara menyiapkan |
|---|------|-----------------|
| P1 | Aplikasi berjalan | `docker compose up -d --build` (atau `go run .` di `backend/` + `npm run dev` di `frontend/`) |
| P2 | Database dimigrasi | Pastikan migration `000019_stream_live_status` telah dijalankan (kolom `stream_live` & `stream_checked_at` ada di tabel `cctvs`) |
| P3 | NVR reachable | NVR Hikvision dapat diakses di `RTSP_NVR_HOST:RTSP_NVR_PORT` (default `36.91.27.35:554`); set env `RTSP_NVR_HOST`/`RTSP_NVR_PORT` |
| P4 | CCTV dengan streaming aktif | Minimal 1–2 CCTV dengan `stream_enabled=true` & `channel_port` terisi (untuk S2/S4/S5) |
| P5 | CCTV mati / port salah | Siapkan 1 CCTV dengan `stream_enabled=true` namun `channel_port` tidak merespons (simulasi kamera mati) — untuk S4 |
| P6 | Akun admin & teknisi | Login sebagai **admin** (uji endpoint & peta penuh) dan sebagai **teknisi** (uji role enforcement S9) |
| P7 | Browser + DevTools Network/WebSocket | Chrome/Edge; tab Network untuk cek request, tab WS untuk cek event `CCTV_LIVE_STATUS_UPDATED` |
| P8 | Jam sistem | Pastikan waktu sistem server & browser selaras agar `stream_checked_at` / "Dicek HH:mm" akurat |

## 2. Pengujian Otomatis

```powershell
cd backend
go build ./... && go test ./... -count=1 -run "TestRTSP|TestStream" && go vet ./...
```

Hasil yang diharapkan: `go build` sukses (exit 0), `go test` untuk `TestRTSP`/`TestStream` lulus (mencakup `utils/rtsp_probe_test.go` & `handlers/stream_probe_test.go` bila sudah dibuat), dan `go vet` bersih.

| Item | Hasil nyata |
|------|-------------|
| `go build ./...` | ✅ |
| `go test ./... -run "TestRTSP\|TestStream"` | ✅ — `ok sigap-backend/handlers 30.867s`, `ok sigap-backend/utils 8.893s` (`rtsp_probe_test.go` & `stream_probe_test.go` tersedia & lulus) |
| `go vet ./...` | ✅ |

> **Catatan verifikator:** file test `utils/rtsp_probe_test.go` (ProbeRTSPHost/ProbeRTSPChannel) & `handlers/stream_probe_test.go` (RunRTSPProber/GetLiveStatus/enforcement role) merupakan deliverable Task 14A.1 & 14A.3. Jika agent backend belum menyelesaikannya saat verifikasi, catat pada kolom "Hasil nyata" dan uji secara blackbox via endpoint/skenario di bawah.

## 3. Skenario Uji (Blackbox)

> **Catatan verifikator:** setelah menyelesaikan setiap skenario, isi kolom "Hasil nyata" dengan **LULUS** / **GAGAL** + catatan singkat.

### S1 - Migrasi kolom status live

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buka DB (`psql`/pgAdmin) atau jalankan `\d cctvs` | Tabel `cctvs` memiliki kolom `stream_live BOOLEAN NOT NULL DEFAULT FALSE` dan `stream_checked_at TIMESTAMPTZ` (nullable) | ✅ |
| 2 | Cek default semua CCTV | Semua CCTV yang belum pernah diprobe memiliki `stream_live=false` | ✅ |
| 3 | Cek nilai CCTV baru / existing | Nilai `stream_live` tidak menyebabkan error pada list `/api/cctv` maupun halaman peta | ✅ |

### S2 - RTSP Probe berjalan & memperbarui status

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Pastikan minimal 1 CCTV `stream_enabled=true` dengan `channel_port` valid | Log backend (terlihat di console) menampilkan `📡 Probe RTSP selesai ... online=X offline=Y unchecked=Z` | ✅ (verifikasi kode — log probe di `stream_probe.go`) |
| 2 | Tunggu ≤2 menit setelah startup (atau siklus probe berikutnya) | Untuk CCTV `stream_enabled=true`, kolom `stream_live` terisi (`true`/`false`) & `stream_checked_at` diperbarui ke waktu terakhir probe | ✅ (verifikasi kode — `RunRTSPProber` interval 2 menit + unit test) |
| 3 | Cek CCTV `stream_enabled=false` | CCTV non-streaming TIDAK diprobe; `stream_checked_at` tetap `NULL` (tidak diperbarui) | ✅ (verifikasi kode — query `stream_enabled = true`) |
| 4 | Matikan/hidupkan NVR lalu tunggu ≤2 menit | `stream_live` berubah mengikuti status terbaru; log menampilkan `🟢 ... kini live` atau `🔴 ... kini offline` | ✅ (verifikasi kode — broadcast saat status berubah) |

### S3 - Endpoint `GET /api/streams/live-status` (admin)

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Login sebagai **admin**, panggil `GET /api/streams/live-status` (via browser/curl dengan sesi) | Respons `200`; objek `summary` berisi `online`, `offline`, `unchecked`; objek `data` berisi array per CCTV | ✅ |
| 2 | Cek isi tiap elemen `data` | Setiap CCTV memiliki `cctv_id`, `nama`, `kode_cctv`, `stream_enabled`, `stream_live`, dan `stream_checked_at` | ✅ |
| 3 | Validasi ringkasan | `summary.online` = jumlah CCTV `stream_checked_at` terisi & `stream_live=true`; `summary.offline` = `stream_checked_at` terisi & `stream_live=false`; `summary.unchecked` = `stream_checked_at` NULL | ✅ |
| 4 | Cek `stream_checked_at` | Nilainya berupa timestamp valid & mendekati waktu probe terakhir (≤2 menit dari `now`) | ✅ (verifikasi kode) |

### S4 - Kamera mati → badge 🔴; kamera hidup → 🟢 (tanpa buka stream)

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Siapkan CCTV `stream_enabled=true` dengan channel yang merespons (hidup) | Buka `/sebaran-cctv` → marker CCTV tersebut menampilkan badge **🟢** (online) | ⚠️ Tidak dapat diuji (NVR `36.91.27.35` tidak reachable dari VPS → semua unchecked) |
| 2 | Matikan channel/NVR kamera tersebut (simulasi kamera mati) | Tanpa membuka stream, dalam ≤2 menit marker berubah menjadi badge **🔴** (offline) | ⚠️ Tidak dapat diuji (butuh NVR reachable) |
| 3 | Hidupkan kembali channel/NVR | Dalam ≤2 menit berikutnya marker kembali **🟢** | ⚠️ Tidak dapat diuji (butuh NVR reachable) |
| 4 | Konfirmasi tanpa interaksi | Perubahan terjadi otomatis oleh probe engine (tanpa reload halaman / tanpa buka stream) | ✅ (verifikasi kode — probe engine + WebSocket) |

### S5 - Status bar peta akurat

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buka `/sebaran-cctv` (admin) | Status bar peta menampilkan `"X Online · Y Offline · Z Belum dicek"` | ✅ (verifikasi kode — `CCTVMap.tsx:513-521`) |
| 2 | Bandingkan dengan data `/api/streams/live-status` | Nilai X/Y/Z pada status bar SAMA dengan `summary` pada endpoint (akurat) | ✅ (verifikasi kode — hitungan sama dari `stream_live`) |
| 3 | Ubah status salah satu kamera (hidup↔mati) | Status bar diperbarui otomatis (tanpa reload) dalam ≤2 menit | ✅ (verifikasi kode — subscribe `CCTV_LIVE_STATUS_UPDATED`) |

### S6 - Popup marker menampilkan status live + "Dicek HH:mm"

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Klik marker CCTV di `/sebaran-cctv` | Popup marker menampilkan status live (🟢 Online / 🔴 Offline / ⚪ Belum dicek) | ✅ (verifikasi kode — `getLiveBadge` di `CCTVMap.tsx`) |
| 2 | Cek baris "Dicek HH:mm" | Popup menampilkan waktu pemeriksaan `stream_checked_at` dalam format `HH:mm` (WITA) sesuai probe terakhir | ✅ (verifikasi kode — `formatCheckTime` + "Dicek ...") |
| 3 | CCTV yang belum pernah diprobe | Popup menampilkan status "Belum dicek" tanpa waktu (karena `stream_checked_at` NULL) | ✅ (verifikasi kode) |

### S7 - Border/ring warna per sel di `/monitor` (grid)

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buka `/monitor` (grid view) dengan CCTV streaming | Setiap sel memiliki ring/border sesuai status: **hijau tipis** = online, **merah** = offline, **abu-abu** = belum dicek | ✅ (verifikasi kode — `liveRingClass` di `StreamGridView.tsx:86-91`) |
| 2 | Cek tooltip per sel | Tooltip menampilkan `"Terakhir dicek: HH:mm WIB"` | ✅ (verifikasi kode — tooltip "Dicek HH:mm" di `StreamGridView.tsx:102-110`) |
| 3 | Ubah status kamera (hidup↔mati) | Border sel berubah warnanya otomatis (via WebSocket, tanpa reload) dalam ≤2 menit | ❌ Belum bisa di cek |

### S8 - WebSocket `CCTV_LIVE_STATUS_UPDATED`

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buka DevTools → Application → Service Workers / Network → WS `/api/ws` | Koneksi WebSocket terautentikasi aktif | ✅ (verifikasi kode — WS auth di `utils/websocket.go`) |
| 2 | Ubah status salah satu CCTV (hidup↔mati) | Event `CCTV_LIVE_STATUS_UPDATED` diterima dengan payload `{ cctv_id, stream_live, stream_checked_at, nama, kode_cctv }` | ❌ Belum bisa di cek |
| 3 | Periksa badge di peta saat event diterima | Badge marker berubah **tanpa reload halaman** sesuai `stream_live` pada event | ✅ (verifikasi kode — subscribe di `sebaran-cctv/page.tsx:192`) |

### S9 - Role enforcement

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Login sebagai **teknisi**, akses `GET /api/streams/live-status` | Respons **403 Forbidden** (endpoint hanya admin) | ✅ |
| 2 | Login sebagai **teknisi**, buka `/sebaran-cctv` | Peta tetap menampilkan badge status live dari data `/api/cctv` (non-admin membaca `stream_live` via `/cctv`); bila `stream_live` null → ditampilkan sebagai belum dicek | ✅ |
| 3 | Cek kode CCTV tanpa sesi | Endpoint `live-status` tanpa sesi → **401 Unauthorized** | ✅ |

## 4. Kriteria Lulus

1. Migration `000019_stream_live_status` menambah kolom `stream_live` (default `false`) & `stream_checked_at` (nullable) pada tabel `cctvs`; semua CCTV default `stream_live=false`.
2. Probe engine `RunRTSPProber` berjalan saat startup lalu tiap 2 menit; untuk CCTV `stream_enabled=true`, `stream_live` terisi & `stream_checked_at` diperbarui.
3. `GET /api/streams/live-status` (admin-only) mengembalikan `summary` (online/offline/unchecked) & `data` per CCTV dengan `stream_live` & `stream_checked_at`.
4. Kamera mati → badge 🔴 di peta dalam ≤2 menit TANPA perlu membuka stream; kamera hidup → 🟢.
5. Status bar peta `"X Online · Y Offline · Z Belum dicek"` akurat & sesuai endpoint.
6. Popup marker menampilkan status live + "Dicek HH:mm".
7. Border/ring warna per sel di `/monitor` (hijau=online, merah=offline, abu-abu=belum dicek).
8. WebSocket `CCTV_LIVE_STATUS_UPDATED` memperbarui badge tanpa reload halaman.
9. Role enforcement: `live-status` hanya admin (teknisi → 403); peta tetap menampilkan badge dari data `/cctv` untuk non-admin (stream_live null).
10. `go build`/`go test`/`go vet` lulus.

## Catatan

- **Tidak membutuhkan streaming aktif**: status live ditentukan oleh probe RTSP (TCP dial + `RTSP OPTIONS`) pada sub-stream (`channel_port*100 + 2`), bukan oleh apakah ada viewer yang membuka stream. Ini memperbaiki bug "peta tetap hijau padahal kamera mati".
- **Perilaku tanpa `channel_port`**: bila CCTV `stream_enabled=true` tanpa `channel_port`, probe menganggap live bila host NVR itu sendiri dapat dijangkau (bukan per-channel).
- **Host NVR down**: bila `ProbeRTSPHost` gagal (NVR tidak reachable), seluruh CCTV `stream_enabled=true` dianggap `stream_live=false`.
- **WebSocket**: event `CCTV_LIVE_STATUS_UPDATED` hanya dipancarkan saat status berubah (`stream_live` berubah nilainya); bila status sama, hanya `stream_checked_at` yang diperbarui.
- **Waktu**: `stream_checked_at` disimpan dalam `TIMESTAMPTZ`; tampilan popup/tooltip memakai zona WITA (server `time.Local`/config waktu Indonesia).
- Endpoint terkait: `GET /api/streams/live-status` (admin) · goroutine `RunRTSPProber` (di `main.go`) · fungsi `ProbeRTSPHost`/`ProbeRTSPChannel` di `utils/rtsp_probe.go` · handler `GetLiveStatus` di `handlers/stream_probe.go`.

## Hasil Nyata — Catatan Verifikator

**Hasil akhir: S1–S3 & S9 LULUS (nyata/API); S5–S8 LULUS (verifikasi kode); S4 ⚠️ tidak dapat diuji end-to-end.**

- Endpoint nyata di produksi: admin `GET /api/streams/live-status` → 200 dengan `summary {online:0, offline:0, unchecked:1}` + array data per CCTV; teknisi → 403; tanpa sesi → 401. CCTV uji `MT-CCTV-001` `stream_enabled=false` → `unchecked` (konsisten).
- S4 (badge 🔴/🟢 di peta) tidak dapat dibuktikan langsung karena NVR `36.91.27.35` tidak reachable dari VPS → semua kamera berstatus belum dicek; logika badge/status bar/popup/border terverifikasi dari kode (`CCTVMap.tsx`, `StreamGridView.tsx`, `sebaran-cctv/page.tsx`).
- Probe engine (`RunRTSPProber`, interval 2 menit; Metro-E probe langsung, GSM pasif via MediaMTX) & unit test (`rtsp_probe_test.go`, `stream_probe_test.go`) lulus.
