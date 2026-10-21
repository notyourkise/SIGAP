# Diagnosis Streaming CCTV — 2026-10-20

**Status:** pilot THD-15 SD berhasil tampil di browser pengguna. Keputusan operasional: tetap memakai H.265 dan menjadikan sub-stream (SD) sebagai default.

**Lingkungan:** SIGAP production lokal, Nginx → MediaMTX → WHEP → browser.

**Data:** 35 record THD non-deleted, 20 kamera aktif, seluruh URL RTSP tersimpan terenkripsi.

> **Keamanan:** laporan ini tidak memuat password RTSP. URL lengkap hanya ada di database terenkripsi dan tidak ditampilkan di UI.

## Ringkasan

1. Pipeline browser **bekerja untuk H.264**. Uji WHEP headless pada sumber H.264 menghasilkan HTTP `201`, codec `H264/90000`, resolusi `1920x1080`, dan **121 frame** dalam 12 detik.
2. Pipeline **bekerja untuk H.265 pada browser yang mendukung HEVC**. Sesi WHEP THD-15 membaca satu track H.265 dan video tampil di browser pengguna.
3. Uji WHEP **headless tanpa dukungan HEVC** pada sumber H.265 menghasilkan HTTP `400` dengan body `codecs not supported by client`. Ini menjelaskan perbedaan hasil antar-klien.
4. `THD-15` memakai H.265 pada **main dan sub-stream**; `THD-06` memakai H.264 pada main dan H.265 pada sub.
5. Web/ISAPI NVR mengarah ke port `5001`, tetapi port tersebut tidak terbuka dari host ini. Port `554` (RTSP) terbuka; port `443` hanya mengembalikan halaman redirect ke `5001`.
6. Tidak ada perubahan encoding batch pada NVR. Aplikasi sekarang memakai sub-stream sebagai default; main stream tetap dapat dipilih eksplisit.
7. Status probe on-demand diperbaiki: path yang terdaftar tetapi belum berjalan karena tidak ada penonton kini berstatus **idle/Siaga**, bukan `offline`.
8. Probe awal setelah restart distabilkan: probe menunggu MediaMTX siap dan path aktif terdaftar sebelum menulis status, sehingga tidak muncul `unknown=20` sementara.

## Bukti Teknis

### Codec kamera

| Kamera | Stream | Codec | Resolusi | Catatan |
|---|---|---|---|---|
| THD-15 | main | H.265/HEVC | 2560x1440 | Butuh klien HEVC |
| THD-15 | sub | H.265/HEVC | 640x480 | Default SD; tampil di browser pengguna |
| THD-06 | main | H.264 | 1920x1080 | Kompatibel dengan WebRTC standar |
| THD-06 | sub | H.265/HEVC | 640x480 | Butuh klien HEVC |

### Uji WHEP

| Sumber | Klien | HTTP | Codec sesi | Hasil |
|---|---|---:|---|---|
| H.264 (THD-06 main) | Chrome headless | 201 | `H264/90000` | Video tampil; 121 frame/12 detik |
| H.265 (THD-15 sub) | Browser pengguna (HEVC) | — | `H265` | Sesi WHEP terbentuk; video tampil |
| H.265 (THD-15 sub) | Chrome headless (tanpa HEVC) | 400 | — | `codecs not supported by client` |

### Port NVR

| Port | Hasil |
|---:|---|
| 554 | Terbuka (RTSP) |
| 443 | Terbuka, tetapi hanya redirect ke port 5001 |
| 5001 | Timeout dari host dan container API |
| 80 / 8000 / 8080 | Tertutup |

## Perbaikan yang Sudah Diterapkan

- `ResolveRTSPURL` menyesuaikan suffix channel Hikvision `01` (main) / `02` (sub) pada URL penuh, sehingga tombol SD/HD benar-benar memilih stream yang berbeda.
- `PatchPath` + `ensureStreamPath` memperbarui konfigurasi path MediaMTX yang sudah ada, bukan hanya menganggap `already exists` sebagai sukses.
- `ReRegisterFailedStreams` memakai daftar **config path**, sehingga path on-demand yang idle tidak dianggap hilang.
- Frontend mendeteksi respons codec tidak didukung, menghentikan retry tanpa akhir, dan menampilkan detail penyebab/tindakan.
- Default kualitas diubah ke **sub/SD** setelah pilot THD-15 berhasil; pengguna tetap bisa memilih HD/main.
- Migration `000025_stream_state` menambahkan status empat keadaan: `live`, `idle`, `offline`, `unknown`.
- Probe GSM membedakan path on-demand yang belum berjalan (`idle`) dari kegagalan terkonfirmasi (`offline`); kegagalan MediaMTX tidak lagi memaksa semua kamera menjadi offline.
- Stream health monitor tidak memulai countdown `STREAM_DOWN` untuk status `idle`/`unknown`, sehingga tidak ada alarm palsu setelah 15 menit.
- Data master 35 CCTV THD disinkronkan dari daftar terbaru: nama/area, koordinat, `sumber_listrik` (PLTS/PLN), `tipe_koneksi` (GSM/Metro-E/Tidak Ada Layanan), `ip_camera`, status aktif/nonaktif, dan keterangan perangkat.
- `ip_camera` kini terisi untuk seluruh 35 record; 17 record bertipe Metro-E (16 aktif, 1 nonaktif), 9 GSM, dan 9 Tidak Ada Layanan. Sebanyak 18 record memiliki IP `10.20.105.x` (termasuk THD-01 yang bertipe Tidak Ada Layanan).
- Karena host/container API tidak memiliki rute ke `10.20.105.x`, probe langsung Metro-E dinonaktifkan melalui `CCTV_METRO_E_DIRECT_PROBE=false`. Status Metro-E memakai MediaMTX pasif (live/idle/offline/unknown) agar tidak menampilkan offline palsu.
- `THD-01` kini memakai koordinat dari sumber data; estimasi tersisa 9 kamera dan tetap diberi label `Estimasi`.

## Bukti Status Probe On-Demand

Setelah deploy dan siklus probe pertama:

```
Probe cycle: 20 kamera dicek | online=0 idle=20 offline=0 unknown=0
```

Database: 20/20 kamera aktif memiliki `stream_state=idle`, `stream_live=false`, dan `stream_checked_at` terisi. Tidak ada log `STREAM_DOWN` palsu. Status idle berarti path on-demand terdaftar dan siap dibuka saat pengguna menonton, bukan kamera mati.

Catatan data: 16 kamera aktif adalah Metro-E dan 4 adalah GSM. Seluruh IP sudah tersimpan, tetapi probe langsung Metro-E belum diaktifkan karena `10.20.105.x` timeout dari container API. Set `CCTV_METRO_E_DIRECT_PROBE=true` hanya setelah host benar-benar berada di jaringan OIKN; setelah itu status Metro-E menjadi live/offline biner.

## Stabilisasi Probe Awal Setelah Restart

Sebelum perbaikan, API bisa healthy lebih dulu daripada MediaMTX. Probe pertama langsung berjalan, tidak menemukan path on-demand, lalu menulis `unknown` untuk semua kamera aktif sampai siklus 2 menit berikutnya.

Perbaikan di `RunRTSPProber`:
- menunggu MediaMTX siap maksimal 2 menit sebelum probe pertama;
- menjalankan `ReRegisterFailedStreams` agar path kamera aktif terdaftar lebih dulu;
- melewati probe awal bila MediaMTX belum siap, sehingga status terakhir di database tidak ditimpa `unknown`.

Bukti uji restart terkontrol:

| Waktu | Peristiwa |
|---|---|
| 20:05:31 | Container API mulai |
| 20:05:37 | Container MediaMTX mulai |
| 20:05:39 | MediaMTX belum dapat di-resolve; tidak ada probe yang menulis status |
| 20:05:42 | Registrasi 20 path selesai; probe pertama: `online=0 idle=20 offline=0 unknown=0` |

Hasil database setelah probe pertama: `idle|20`, tanpa `unknown`. Path MediaMTX berjumlah 21 setelah path `dawda` dipulihkan secara runtime.

Catatan: `dawda` adalah baris soft-deleted non-THD dengan `stream_enabled=true`. Path ini tidak ikut auto-register karena barisnya soft-deleted; setelah MediaMTX restart path tersebut hilang dan dipulihkan manual dari URL terenkripsi tanpa menampilkan kredensial. Keputusan apakah path ini perlu dipersistenkan masih pending.

## Catatan Kompatibilitas

H.265/HEVC tidak dijamin tersedia di semua browser/perangkat. Klien tanpa dukungan HEVC akan menerima HTTP `400` dari MediaMTX. Untuk klien seperti itu:

- gunakan browser/perangkat yang mendukung HEVC, atau
- ubah encoding stream kamera (main/sub) di NVR ke H.264, lalu mulai ulang stream.

Tidak ada transcoding FFmpeg yang diaktifkan.

## Yang Belum Selesai

- Rollout H.264 sub-stream ke 20 kamera aktif **tidak dilakukan** karena keputusan operasional saat ini adalah mempertahankan H.265.
- Kompatibilitas perangkat pengguna (HP, Edge, Chrome tanpa HEVC extension) belum diuji menyeluruh.
- Rotasi password NVR masih pending dan menunggu koordinasi agar seluruh URL tersimpan diperbarui.
- Probe langsung Metro-E belum bisa diaktifkan dari host ini karena `10.20.105.x` tidak terjangkau; status masih pasif.
- Sembilan koordinat masih estimasi dan perlu verifikasi lapangan.
- Perlu pemantauan beberapa siklus untuk memastikan status idle tetap stabil dan tidak ada alarm palsu.
- Path `dawda` (soft-deleted non-THD) hanya ada secara runtime dan hilang saat MediaMTX restart; perlu diputuskan apakah akan dipersistenkan atau dihentikan.

## Langkah Berikutnya

1. Uji playback SD di perangkat yang benar-benar akan dipakai pengguna (minimal satu HP dan satu browser desktop selain mesin uji).
2. Pantau laporan pengguna; bila ada perangkat tanpa HEVC, ubah sub-stream kamera terkait ke H.264 atau arahkan pengguna memilih HD/main yang kompatibel.
3. Bila kompatibilitas luas menjadi prioritas, lanjutkan rollout H.264 sub-stream secara bertahap dan verifikasi tiap kamera.
4. Pantau log probe/health monitor setelah beberapa siklus; pastikan kamera idle tetap berstatus Siaga dan tidak memicu `STREAM_DOWN`.
5. Set `CCTV_METRO_E_DIRECT_PROBE=true` setelah API ditempatkan di jaringan yang dapat menjangkau `10.20.105.x`, lalu verifikasi 16 kamera Metro-E aktif.
6. Verifikasi lapangan untuk 9 koordinat estimasi: THD-18, THD-20, THD-25, THD-27, THD-29, THD-30, THD-31, THD-34, THD-35.
