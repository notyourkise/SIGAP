# Verifikasi Aktif Status Streaming CCTV

**Tanggal:** 2026-10-22  
**Branch:** `improvement/cctv-active-verification`  
**Status:** Implementasi dan validasi lokal selesai; deployment belum dilakukan.

## Masalah

Status pasif MediaMTX untuk path on-demand adalah `idle` ketika tidak ada penonton.
Nilai tersebut sebelumnya dapat tampil sebagai "Siaga" dan tidak membuktikan apakah
kamera benar-benar mengirim video. Akibatnya badge online/offline tidak cukup dapat
dipercaya untuk operasional.

## Perubahan

- Migrasi `000027_stream_verification` menambahkan kolom terpisah:
  `stream_verified_state`, `stream_verified_at`, `stream_verified_method`,
  `stream_verify_failures`, dan `stream_verify_error`.
- Verifier periodik end-to-end memicu HLS MediaMTX, lalu memeriksa
  `SourceReady()` dan jumlah track. Status `offline` baru ditetapkan setelah dua
  kegagalan berturut-turut; kegagalan koneksi MediaMTX menghasilkan `unknown`,
  bukan offline.
- Endpoint admin `POST /api/streams/:id/verify` menjalankan verifikasi manual dan
  mengembalikan hasil beserta `tracks`, `duration_ms`, dan error diagnosis.
- Bila path tidak ditemukan (404), verifier mendaftarkan ulang path satu kali
  sebelum menyimpulkan `unknown`. Ini menangani path yang hilang setelah
  **Stop Semua Stream** atau restart MediaMTX. Kegagalan pendaftaran ulang tidak
  menaikkan failure counter karena bukan bukti kamera offline.
- Status terverifikasi yang masih segar (30 menit) menang atas status pasif pada
  `/api/streams/live-status`, badge peta, grid monitor, detail CCTV, statistik peta,
  dan health monitor.
- UI menampilkan label "Online terverifikasi"/"Offline terkonfirmasi", waktu
  verifikasi, serta tombol admin **Cek Status Live**.
- Verifikasi memakai concurrency default 1 dan `SourceOnDemandCloseAfter: 10s`
  agar NVR tidak dibuka serentak. Interval default 15 menit (rentang 5-30 menit).
- Konfigurasi HLS MediaMTX memakai `hlsSegmentCount: 7` agar playlist low-latency
  valid.

## Bukti Runtime Lokal

- Database lokal berada pada migrasi versi 27, `dirty=false`.
- 20 kamera dengan `stream_enabled=true` diverifikasi.
- Siklus verifikasi berulang konsisten: **17 live, 3 offline, 0 unknown**,
  durasi sekitar 1 menit 6 detik.
- `GET /api/streams/live-status` mengembalikan total 35 kamera dengan
  `online=17`, `offline=3`, `unchecked=15`, `verified_live=17`,
  `verified_offline=3`.
- Tiga kamera offline terkonfirmasi: `THD-04`, `THD-10`, dan `THD-12`.
  Ketiganya menghasilkan NVR `453 (Not Enough Bandwidth)` saat diuji terpisah,
  sehingga hasil verifier bukan artefak beban verifikasi.
- `POST /api/streams/:id/verify` telah diuji dengan sesi admin lokal:
  - kamera live menghasilkan `live`, `tracks=1`, sekitar 1,3 detik;
  - kamera offline menghasilkan `offline`, `failures` bertambah, sekitar 10 detik.
- Tanpa autentikasi, endpoint manual dan live-status mengembalikan HTTP 401.

## Keterbatasan

- Snapshot periodik tidak dapat memakai MediaMTX 1.20 karena endpoint
  `/v3/paths/get/{path}/snapshot` tidak tersedia (selalu 404). Scheduler sekarang
  berhenti setelah satu peringatan agar tidak membanjiri log. Opsi FFmpeg
  transcoding tidak dipakai sesuai batasan proyek.
- Tiga kamera THD tetap offline selama NVR mengembalikan `453`; port ISAPI NVR
  tidak dapat dijangkau dari host/API untuk diagnosis lebih lanjut.
- Browser/perangkat tanpa dukungan H.265 tidak dapat memutar stream H.265 tanpa
  transcoding.
- **Stop Semua Stream** hanya menghapus path dari MediaMTX; path dapat
  didaftarkan ulang oleh re-registrar periodik atau saat verifikasi. Untuk
  mematikan kamera secara permanen, nonaktifkan `stream_enabled` pada kamera
  tersebut.

## Checklist Deployment

1. Set `MEDIAMTX_WEBRTC_ADDITIONAL_HOSTS` ke host/IP yang dapat dijangkau browser.
2. Terapkan dan verifikasi migrasi 26 dan 27 pada database target.
3. Pastikan `MEDIAMTX_HLS_URL`, `CCTV_VERIFY_INTERVAL_MINUTES`, dan
   `CCTV_VERIFY_CONCURRENCY` terdefinisi sesuai kapasitas NVR.
4. Jalankan verifikasi manual pada satu kamera live dan satu kamera offline.
5. Pastikan tidak ada kredensial RTSP yang masuk ke log atau dokumen.
6. Deployment belum dilakukan pada pekerjaan ini.

## Validasi Otomatis

- Backend: `go test ./... -count=1`, `go vet ./...`, dan `golangci-lint run ./...`
  lulus (0 issue).
- Frontend: unit test service worker, WHEP, dan live-status lulus; `npm run build`
  lulus; lint 0 error (warning yang sudah ada tetap).
- Test baru mencakup prioritas status terverifikasi, timestamp kedaluwarsa/masa
  depan, ambang dua kegagalan, MediaMTX tidak terjangkau, pemicu HLS, registrasi
  route verifikasi manual, serta pendaftaran ulang otomatis path yang hilang
  (berhasil, gagal, dan tetap 404 setelah pendaftaran ulang).
