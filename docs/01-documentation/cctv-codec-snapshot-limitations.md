# Batasan Codec Streaming dan Snapshot CCTV

| Item | Detail |
|------|--------|
| **Issue** | #149 (P2 - Kompatibilitas H.265 dan snapshot) |
| **Berlaku sejak** | Backend `0.64.39`, Frontend `1.55.38` |
| **Terkait** | #146 (semantik log), #147 (gate streaming produksi), `docs/01-documentation/production-deployment-runbook.md` |

Dokumen ini menjelaskan mengapa sebagian browser tidak dapat memutar stream
CCTV, bagaimana aplikasi menangani kegagalan codec, serta kapan tombol snapshot
aktif. Prinsip utama SIGAP tetap **zero-transcoding**: MediaMTX hanya
meneruskan RTSP ke WebRTC/WHEP; server tidak mengubah codec video.

---

## 1. Ringkasan Perilaku

1. Pemutar WHEP mendeteksi penolakan codec dari MediaMTX (HTTP 400/415 dengan
   pesan `codec`) dan **berhenti melakukan retry tanpa henti**.
2. Jika sub-stream memakai H.265/HEVC dan browser tidak mendukungnya, pemutar
   mencoba **main stream (HD) satu kali**. Bila main stream juga H.265,
   pengguna melihat panduan permanen: gunakan perangkat yang mendukung HEVC
   atau ubah encoding kamera ke H.264 di NVR.
3. Tombol **Snapshot** hanya aktif bila backend memastikan endpoint snapshot
   tersedia dan file snapshot terbaru benar-benar ada. Bila MediaMTX 1.20 tidak
   memiliki endpoint snapshot, tombol dinonaktifkan dengan alasan yang jelas.
4. UI tidak pernah membuka URL snapshot yang diketahui 404. Endpoint status
   menjadi sumber kebenaran untuk tombol.

---

## 2. Codec H.265/HEVC

### 2.1 Mengapa tidak semua browser bisa memutar

WebRTC tidak memakai decoder video umum seperti pemutaran file MP4. Browser
harus mengiklankan dukungan codec pada `RTCRtpReceiver.getCapabilities('video')`
agar MediaMTX dapat mengirim H.265. Dukungan HEVC sangat bergantung pada
platform:

| Platform | Kecenderungan dukungan HEVC/WebRTC |
|----------|-----------------------------------|
| Safari macOS/iOS | Umumnya mendukung HEVC |
| Chrome/Edge desktop | Sering **tidak** mengiklankan H.265, tergantung OS/ekstensi codec |
| Android Chrome | Bervariasi menurut perangkat dan versi OS |
| WebView/aplikasi embedded | Bervariasi; sering tidak mendukung |

Tabel ini bukan jaminan. Aplikasi memakai deteksi runtime, bukan daftar
perangkat statis.

### 2.2 Deteksi di frontend

`frontend/src/lib/hevc-support.ts` memeriksa secara berurutan:

1. `RTCRtpReceiver.getCapabilities('video')` → sinyal paling akurat untuk
   WebRTC. Bila daftar codec tersedia dan tidak memuat `video/H265` atau
   `video/HEVC`, HEVC dinyatakan **tidak didukung**.
2. `MediaSource.isTypeSupported()` dan `HTMLVideoElement.canPlayType()` sebagai
   sinyal tambahan untuk browser/WebView yang tidak menyediakan capabilities.
   Hasil negatif dari jalur ini tidak dipakai sebagai bukti mutlak karena
   dukungan MSE tidak identik dengan dukungan WebRTC.

Jika hasilnya tidak pasti, pesan memakai kata "tampaknya tidak mendukung" agar
tidak menyesatkan pengguna.

### 2.3 Alur penanganan kegagalan

```text
WHEP POST /stream/cam-<uuid>/whep
        |
        +-- SDP answer sukses --> tunggu frame --> live
        |
        +-- HTTP 400/415 + pesan codec
                |
                +-- kualitas sub dan fallback belum dicoba
                |       -> restart otomatis ke main stream (HD)
                |
                +-- kualitas main / fallback sudah dicoba
                        -> hentikan retry
                        -> overlay "Codec video tidak didukung"
                        -> tombol "Coba sub-stream (SD)" / "Muat ulang"
                        -> toast berisi penyebab dan tindakan
```

Pemutar tidak melakukan reconnect eksponensial untuk error codec karena SDP
yang sama akan ditolak berulang kali. Retry hanya dilakukan setelah pengguna
mengganti kualitas atau mengubah encoding kamera.

### 2.4 Tindakan yang dapat dilakukan operator

1. **Coba kualitas lain** dari overlay pemutar (HD ↔ SD). Kadang main stream
   H.264 sementara sub-stream H.265, atau sebaliknya.
2. **Gunakan browser/perangkat yang mendukung HEVC**, mis. Safari di
   macOS/iOS.
3. **Ubah encoding kamera di NVR ke H.264** (main dan sub). Ini solusi permanen
   dan paling kompatibel. Perubahan dilakukan di perangkat NVR, bukan di SIGAP.
4. Setelah mengubah encoding, mulai ulang stream dari modal agar MediaMTX
   membaca sumber yang baru.

> **Catatan kebijakan:** Jangan mengaktifkan transcoding FFmpeg di MediaMTX
> untuk "memaksa" H.264. Kebijakan zero-transcoding menjaga beban CPU VPS dan
> kualitas stream. Perubahan codec harus dilakukan di kamera/NVR.

---

## 3. Snapshot CCTV

### 3.1 Sumber dan jadwal

- Snapshot diambil dari endpoint MediaMTX `GET /v3/paths/get/{path}/snapshot`.
- Snapshot berkala berjalan setiap **15 menit** untuk CCTV dengan
  `stream_enabled=true`, setelah penundaan awal 10 detik.
- Snapshot juga dipicu oleh perubahan status, penyelesaian pemeliharaan, dan
  pengecekan lapangan yang membutuhkan perhatian (lihat `handlers/snapshot.go`).
- Setiap CCTV menyimpan maksimal **10 file** JPEG di
  `uploads/snapshots/{cctv_id}/`; file tertua dihapus otomatis.

### 3.2 MediaMTX 1.20 tidak menyediakan endpoint snapshot

MediaMTX 1.20 tidak memiliki endpoint snapshot. Ketika scheduler menemukan
path terdaftar tetapi endpoint snapshot membalas 404, backend:

1. menandai `snapshotEndpointUnavailable` secara atomik;
2. menulis **satu** peringatan, bukan puluhan error setiap siklus;
3. menghentikan scheduler snapshot agar tidak membebani log dan MediaMTX.

Snapshot lama yang sudah tersimpan tetap dapat dilihat, tetapi tombol snapshot
di modal dinonaktifkan karena fitur pengambilan snapshot baru tidak tersedia.

### 3.3 Endpoint status snapshot

`GET /api/cctv/{id}/snapshot-status` (semua role yang login) mengembalikan:

```json
{
  "status": "success",
  "data": {
    "available": true,
    "endpoint_supported": true,
    "snapshot_url": "/api/cctv/<uuid>/snapshot",
    "captured_at": "2026-10-28T00:45:00Z",
    "event_type": "periodic_snapshot",
    "message": "Snapshot tersedia."
  }
}
```

| Field | Arti |
|-------|------|
| `available` | `true` hanya bila endpoint didukung **dan** metadata + file snapshot ada |
| `endpoint_supported` | `false` bila backend sudah mendeteksi MediaMTX tanpa endpoint snapshot |
| `captured_at` | Waktu snapshot terbaru diambil (bila ada) |
| `event_type` | `periodic_snapshot`, `status_changed`, `maintenance_completed`, `field_check_attention` |
| `message` | Alasan yang ditampilkan ke pengguna saat tombol dinonaktifkan |

Endpoint ini tidak memicu pengambilan snapshot baru; ia hanya membaca metadata
dan memeriksa keberadaan file. Dengan demikian membuka modal tidak menambah
beban MediaMTX atau kuota GSM.

### 3.4 Perilaku UI

- Tombol **Snapshot** berada di footer modal streaming.
- Tombol aktif hanya bila `available=true`; klik membuka
  `/api/cctv/{id}/snapshot` di tab baru (cookie sesi tetap dipakai).
- Bila `available=false`, tombol berstatus `disabled` dan pesan alasan
  ditampilkan di bawah tombol (mis. "MediaMTX yang berjalan tidak menyediakan
  endpoint snapshot" atau "Belum ada snapshot tersimpan").
- Bila file snapshot terhapus dari disk, backend melaporkan `available=false`
  sehingga UI tidak menampilkan gambar rusak.
- Timeline CCTV tetap menampilkan placeholder "Foto tidak dapat ditampilkan"
  bila snapshot untuk event tertentu tidak tersedia; ini bukan error halaman.

---

## 4. Konfigurasi dan Verifikasi

Tidak ada environment variable baru untuk #149. Perilaku mengikuti:

| Variabel | Default | Dampak |
|----------|---------|--------|
| `MEDIAMTX_API_URL` | `http://mediamtx:9997/v3/` | Alamat REST API untuk snapshot |
| `CCTV_METRO_E_DIRECT_PROBE` | `false` | Tidak terkait codec, tetapi memengaruhi status kamera |
| `APP_ENV` | `development` | Tidak mengubah codec |

Verifikasi cepat:

```bash
# Status snapshot (ganti UUID dan cookie)
curl -s https://<domain>/api/cctv/<UUID>/snapshot-status | jq

# Snapshot terbaru (harus image/jpeg bila available=true)
curl -sk -o /tmp/snapshot.jpg -w '%{http_code} %{content_type}\n' \
  https://<domain>/api/cctv/<UUID>/snapshot
```

Cek log API:

```bash
docker compose logs api | grep -i 'snapshot'
```

Log normal hanya menampilkan satu peringatan bila endpoint tidak tersedia.
Peringatan berulang setiap 15 menit menandakan flag tidak tersimpan atau
scheduler dijalankan ulang.

---

## 5. Batasan yang Diterima

- Tidak ada transcoding server-side; browser tanpa decoder HEVC tidak dapat
  memutar kamera H.265 sampai encoding di NVR diubah.
- Deteksi HEVC bersifat best-effort. Browser yang mengiklankan HEVC tetapi
  gagal mendekode frame akan tetap masuk ke overlay error setelah timeout frame
  (12 detik) atau penolakan MediaMTX.
- Snapshot bergantung pada MediaMTX. Bila endpoint tidak tersedia, fitur
  snapshot dinonaktifkan; SIGAP tidak mengambil frame melalui RTSP langsung.
- Snapshot bukan rekaman video. Playback historis tetap berada di NVR lokal.
- Rotasi 10 file per CCTV bersifat best-effort; kegagalan menghapus file
  dicatat sebagai warning dan tidak menghentikan scheduler.

---

## 6. Referensi

- `frontend/src/lib/hevc-support.ts` — deteksi HEVC dan pesan panduan.
- `frontend/src/components/WHEPPlayer.tsx` — fallback kualitas dan overlay codec.
- `frontend/src/components/StreamModal.tsx` — tombol snapshot dan status.
- `backend/handlers/snapshot.go` — capture, rotasi, status, scheduler.
- `backend/routes/cctv.go` — rute snapshot dan snapshot-status.
- `docs/04-manual-tests/manual-test-cctv-codec-snapshot.md` — prosedur blackbox.
- `docs/01-documentation/production-deployment-runbook.md` — smoke test produksi.
