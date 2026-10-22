# Manual Test - Verifikasi Aktif Status Streaming CCTV

| Item | Detail |
|------|--------|
| **Task ID** | CCTV Active Verification |
| **Branch** | `improvement/cctv-active-verification` |
| **Versi Backend** | 0.64.27 (setelah fix) |
| **Versi Frontend** | 1.55.26 (setelah fix) |
| **Tanggal** | 2026-10-22 |

---

## 1. Prasyarat

- [ ] Docker Compose berjalan: `db`, `redis`, `api`, `mediamtx`, `nginx`, `frontend`.
- [ ] Migrasi database sudah mencapai versi **27** (`dirty=false`).
- [ ] Login sebagai **admin**.
- [ ] Tersedia minimal satu kamera live dan satu kamera yang benar-benar tidak dapat
      membuka stream.
- [ ] `MEDIAMTX_API_URL` dan `MEDIAMTX_HLS_URL` dapat dijangkau dari container API.
- [ ] Tidak ada kredensial RTSP yang dicetak ke log atau dokumen.

---

## 2. Pengujian Otomatis

### Backend

```bash
cd backend
gofmt -l .
go test ./... -count=1
go vet ./...
golangci-lint run ./...
```

**Kriteria lulus:** semua test `ok`, vet bersih, dan lint `0 issues`.

### Frontend

```bash
cd frontend
npm run test:sw
npm run test:whep
npm run test:live
npm run lint
npm run build
```

**Kriteria lulus:** seluruh unit test lulus, lint 0 error, dan production build sukses.

---

## 3. Skenario Uji Manual

| No | Langkah | Hasil yang diharapkan | Hasil nyata |
|----|---------|----------------------|-------------|
| 1 | Buka `GET /api/streams/live-status` dengan sesi admin. | Respons memuat `effective_stream_state`, `stream_verified_state`, `stream_verified_at`, `stream_verified_method`, `stream_verify_failures`, dan `stream_verify_error`. Summary memuat `verified_live`/`verified_offline`. | **Lulus.** Total 35 kamera, `online=17`, `offline=3`, `unchecked=15`, `verified_live=17`, `verified_offline=3`. |
| 2 | Buka peta `/sebaran-cctv` dan halaman `/cctv`. | Kamera terverifikasi live berlabel "Online terverifikasi"; kamera terverifikasi offline berlabel "Offline terkonfirmasi". Status pasif `idle` tidak menimpa hasil verifikasi. | **Lulus.** Tiga kamera offline terkonfirmasi: `THD-04`, `THD-10`, `THD-12`. |
| 3 | Buka detail kamera live, klik **Cek Status Live**. | Tombol menampilkan status memverifikasi; respons sukses membawa `live`, `tracks`, `duration_ms`, dan waktu verifikasi. | **Lulus.** `live`, `tracks=1`, sekitar 1,3 detik. |
| 4 | Lakukan langkah 3 pada kamera offline. | Setelah dua kegagalan berturut-turut status menjadi `offline`; error diagnosis tampil tanpa kredensial. | **Lulus.** `offline`, `stream_verify_error` berisi "sumber stream tidak siap setelah 8s"; sekitar 10 detik. |
| 5 | Panggil `POST /api/streams/:id/verify` tanpa autentikasi. | HTTP 401. | **Lulus.** |
| 6 | Tunggu satu siklus verifikasi periodik (default 15 menit). | Log menampilkan ringkasan `live/offline/unknown`; tidak ada error kredensial. | **Lulus.** Siklus berulang: 20 kamera, `live=17 offline=3 unknown=0`, sekitar 1 menit 6 detik. |
| 7 | Periksa log snapshot setelah startup. | Tidak ada puluhan error 404 berulang. Bila endpoint snapshot tidak tersedia, scheduler berhenti dengan satu peringatan. | **Lulus.** Satu peringatan: "Snapshot berkala dihentikan: MediaMTX tidak menyediakan endpoint snapshot (404)". |
| 8 | Klik **Stop Semua Stream** di halaman Monitor, lalu buka detail kamera dan klik **Cek Status Live**. | Backend mendaftarkan ulang path yang hilang satu kali lalu menjalankan verifikasi; status tidak langsung `unknown` hanya karena path tidak terdaftar. | **Lulus (unit test backend).** Uji ulang melalui UI menunggu sesi admin valid. |

### Catatan hasil runtime

- Tiga kamera offline sudah diuji terpisah dan NVR mengembalikan `453 (Not Enough Bandwidth)`,
  sehingga hasil verifier bukan artefak beban verifikasi.
- MediaMTX 1.20 tidak memiliki endpoint snapshot pada API publiknya. Fitur snapshot periodik
  karena itu tidak dapat diandalkan dan scheduler dihentikan secara aman tanpa transcoding FFmpeg.
- Browser tanpa dukungan H.265 tetap tidak dapat memutar stream H.265 tanpa transcoding.

---

## 4. Kriteria Lulus

- [x] Status live/offline memiliki bukti verifikasi end-to-end, bukan hanya status path pasif.
- [x] Status `idle` on-demand tidak lagi membuat kamera tampak offline.
- [x] Kegagalan koneksi MediaMTX tidak langsung dinyatakan sebagai kamera offline.
- [x] Dua kegagalan berturut-turut diperlukan sebelum status offline.
- [x] Path yang hilang setelah Stop Semua Stream/restart MediaMTX didaftarkan ulang sebelum verifikasi (unit test backend).
- [x] Endpoint manual hanya dapat diakses admin.
- [x] UI peta, grid, ringkasan, dan detail menampilkan status efektif yang konsisten.
- [x] Tidak ada kredensial RTSP pada log, API, atau dokumen.
- [x] Test backend/frontend dan build lulus.

---

## 5. Catatan Deployment

- Set `MEDIAMTX_WEBRTC_ADDITIONAL_HOSTS` ke host/IP yang dapat dijangkau browser.
- Terapkan migrasi 26 dan 27 pada database target dan pastikan `dirty=false`.
- Verifikasi `MEDIAMTX_HLS_URL`, `CCTV_VERIFY_INTERVAL_MINUTES`, dan
  `CCTV_VERIFY_CONCURRENCY` sesuai kapasitas NVR.
- Deployment belum dilakukan pada pekerjaan ini.
