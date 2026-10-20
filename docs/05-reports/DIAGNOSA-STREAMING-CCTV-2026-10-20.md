# Diagnosis Streaming CCTV — 2026-10-20

**Status:** akar masalah live playback sudah dikonfirmasi; perubahan encoding NVR belum dapat dilakukan dari host ini.  
**Lingkungan:** SIGAP production lokal, Nginx → MediaMTX → WHEP → Chrome/Edge.  
**Data:** 35 record THD non-deleted, 20 kamera aktif, seluruh URL RTSP tersimpan terenkripsi.

> **Keamanan:** laporan ini tidak memuat password RTSP. URL lengkap hanya ada di database terenkripsi dan tidak ditampilkan di UI.

## Ringkasan

1. Pipeline browser **bekerja untuk H.264**. Uji WHEP pada sumber H.264 menghasilkan HTTP `201`, codec `H264/90000`, resolusi `1920x1080`, dan **121 frame** dalam 12 detik.
2. Pipeline **menolak H.265/HEVC**. Uji WHEP pada sumber H.265 menghasilkan HTTP `400` dengan body `codecs not supported by client`.
3. `THD-15` memakai H.265 pada **main dan sub-stream**; `THD-06` memakai H.264 pada main dan H.265 pada sub.
4. Web/ISAPI NVR mengarah ke port `5001`, tetapi port tersebut tidak terbuka dari host ini. Port `554` (RTSP) terbuka; port `443` hanya mengembalikan halaman redirect ke `5001`.
5. Karena itu, encoding sub-stream NVR belum diubah. Tidak ada perubahan batch pada 20 kamera aktif.

## Bukti Teknis

### Codec kamera

| Kamera | Stream | Codec | Resolusi | Catatan |
|---|---|---|---|---|
| THD-15 | main | H.265/HEVC | 2560x1440 | Tidak dapat diputar WebRTC |
| THD-15 | sub | H.265/HEVC | 640x480 | Target pilot, belum diubah |
| THD-06 | main | H.264 | 1920x1080 | Dapat diputar WebRTC |
| THD-06 | sub | H.265/HEVC | 640x480 | Tidak dapat diputar WebRTC |

### Uji WHEP

| Sumber | HTTP | Codec answer | Hasil |
|---|---:|---|---|
| H.264 (THD-06 main) | 201 | `H264/90000` | Video tampil; 121 frame/12 detik |
| H.265 (THD-15 sub) | 400 | — | `codecs not supported by client` |

### Port NVR

| Port | Hasil |
|---:|---|
| 554 | Terbuka (RTSP) |
| 443 | Terbuka, tetapi hanya redirect ke port 5001 |
| 5001 | Timeout dari host ini |
| 80 / 8000 / 8080 | Tertutup |

## Perbaikan yang Sudah Diterapkan

- `ResolveRTSPURL` sekarang menyesuaikan suffix channel Hikvision `01` (main) / `02` (sub) pada URL penuh, sehingga tombol SD/HD benar-benar memilih stream yang berbeda.
- `PatchPath` + `ensureStreamPath` memperbarui konfigurasi path MediaMTX yang sudah ada, bukan hanya menganggap `already exists` sebagai sukses.
- `ReRegisterFailedStreams` memakai daftar **config path**, sehingga path on-demand yang idle tidak dianggap hilang.
- Frontend mendeteksi respons codec tidak didukung, menghentikan retry tanpa akhir, dan menampilkan detail penyebab/tindakan.
- Default kualitas dipertahankan **main** sampai rollout H.264 sub-stream selesai; pengguna tetap bisa memilih SD secara eksplisit.

## Yang Belum Selesai

- Encoding sub-stream `THD-15` belum diubah ke H.264 karena akses web/ISAPI NVR ke port `5001` tidak tersedia dari host ini.
- Rollout H.264 sub-stream ke 20 kamera aktif belum dilakukan.
- Pengujian tombol SD di aplikasi untuk `THD-15` belum bisa dilakukan sampai encoding NVR berubah.
- Status probe on-demand masih dapat menampilkan `offline` saat sumber idle; ini isu status, bukan penyebab utama kegagalan playback.

## Langkah Berikutnya

1. Dari jaringan yang diizinkan mengakses NVR, buka UI NVR dan ubah **hanya sub-stream THD-15** ke H.264.
2. Verifikasi dengan probe ke channel sub `1502`: codec harus `h264`, bukan `hevc`.
3. Buka aplikasi, pilih `THD-15`, lalu pilih **SD**. Pastikan WHEP mengembalikan `201` dan frame tampil.
4. Bila pilot lolos, ubah sub-stream kamera aktif lainnya ke H.264 secara bertahap dan jalankan **Mulai Semua Stream**.
5. Setelah rollout selesai, default kualitas dapat dikembalikan ke **sub** untuk menghemat bandwidth.
