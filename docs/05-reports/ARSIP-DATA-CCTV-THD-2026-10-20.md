# Arsip Data CCTV THD

**Tanggal arsip:** 2026-10-20  
**Sumber:** daftar data CCTV THD terbaru (koordinat, atribut perangkat, jaringan, dan status), diberikan 2026-10-20  
**Site:** Kawasan OIKN (`f490773e-204f-47ed-b3f2-4db2503e9a5b`)  
**NVR:** `<NVR_IP>:554`  
**Jumlah record:** 35 CCTV  
**Status:** sudah disinkronkan ke database SIGAP pada 2026-10-20 (35 record THD; dua record non-THD lama tidak diubah)

> **Keamanan:** file ini tidak memuat password, user RTSP, atau URL RTSP lengkap. URL lengkap tersimpan terenkripsi (`enc:v1:`) di database dan hanya ditampilkan dalam bentuk tersamarkan oleh aplikasi. Rotasi password NVR secara berkala dan simpan salinan kredensial di password manager, bukan di repository.

## Ringkasan

- Total CCTV: **35**
- Status aktif: **20**
- Status nonaktif: **15**
- Streaming diaktifkan: **20** (hanya kamera berstatus aktif)
- Koordinat terisi: **35/35** (26 dari sumber, 9 estimasi)
- Atribut perangkat terisi: **35/35**
- Sumber listrik: **23 PLTS**, **12 PLN**
- Tipe koneksi: **17 Metro-E**, **9 GSM**, **9 Tidak Ada Layanan**
- IP Camera unik: **35/35**
- Probe Metro-E: **pasif** (`CCTV_METRO_E_DIRECT_PROBE=false`) karena host saat ini tidak dapat menjangkau `10.20.105.x`; status memakai MediaMTX (live/idle/offline/unknown) agar tidak muncul offline palsu.

## Ringkasan per Area

| Area | Jumlah | Aktif | Nonaktif |
|---|---:|---:|---:|
| 1-B, 1-C | 1 | 1 | 0 |
| Glamping | 1 | 0 | 1 |
| HPK 1B | 3 | 0 | 3 |
| HPK 1C | 2 | 0 | 2 |
| Jl. Negara | 5 | 5 | 0 |
| KIPP 1B | 2 | 0 | 2 |
| KIPP 1C | 1 | 0 | 1 |
| Kantor Otorita | 1 | 1 | 0 |
| Masjid Negara | 2 | 2 | 0 |
| PSSI | 1 | 0 | 1 |
| Plasa Seremoni | 1 | 1 | 0 |
| RTJM | 2 | 0 | 2 |
| Rest Area | 5 | 4 | 1 |
| Simpang BIN/Polri | 2 | 2 | 0 |
| Sumbu Barat | 2 | 2 | 0 |
| Sumbu Timur | 2 | 1 | 1 |
| TPST | 1 | 0 | 1 |
| Taman Kusuma Bangsa | 1 | 1 | 0 |

## Daftar CCTV

| No | Kode | Channel | Nama CCTV | Lokasi Area | Status | Tipe Koneksi | IP Camera | Stream |
|---:|---|---|---|---|---|---|---|---|
| 1 | THD-01 | D1 | Rest Area 1 | Rest Area | Nonaktif | Tidak Ada Layanan | `10.20.105.51` | Tidak |
| 2 | THD-02 | D2 | Depan Masjid Negara | Masjid Negara | Aktif | GSM | `20.20.21.26` | Ya |
| 3 | THD-03 | D3 | Parkiran TKB | Taman Kusuma Bangsa | Aktif | Metro-E | `10.20.105.62` | Ya |
| 4 | THD-04 | D4 | Jalan Masjid Negara 1 | Masjid Negara | Aktif | GSM | `20.20.20.226` | Ya |
| 5 | THD-05 | D5 | Bundaran Sumbu Barat | Sumbu Barat | Aktif | Metro-E | `10.20.105.66` | Ya |
| 6 | THD-06 | D6 | Simpang BIN/Polri Luar | Simpang BIN/Polri | Aktif | Metro-E | `10.20.105.59` | Ya |
| 7 | THD-07 | D7 | REST AREA 3 | Rest Area | Aktif | Metro-E | `10.20.105.53` | Ya |
| 8 | THD-08 | D8 | REST AREA 2 | Rest Area | Aktif | Metro-E | `10.20.105.52` | Ya |
| 9 | THD-09 | D9 | Rest Area 4 | Rest Area | Aktif | Metro-E | `10.20.105.54` | Ya |
| 10 | THD-10 | D10 | Sentra Massa | Plasa Seremoni | Aktif | GSM | `20.20.20.34` | Ya |
| 11 | THD-11 | D11 | REST AREA PARKIRAN | Rest Area | Aktif | Metro-E | `10.20.105.55` | Ya |
| 12 | THD-12 | D12 | Simpang BIN/Polri Dalam | Simpang BIN/Polri | Aktif | Metro-E | `10.20.105.58` | Ya |
| 13 | THD-13 | D13 | Simpang ASN 3 (Astra Biz) | Jl. Negara | Aktif | Metro-E | `10.20.105.67` | Ya |
| 14 | THD-14 | D14 | HPK 1C Security | HPK 1C | Nonaktif | Tidak Ada Layanan | `20.20.20.202` | Tidak |
| 15 | THD-15 | D15 | BELAKANG ISTANA | Sumbu Timur | Aktif | Metro-E | `10.20.105.61` | Ya |
| 16 | THD-16 | D16 | Pintu Masuk Rest Area | Jl. Negara | Aktif | Metro-E | `10.20.105.56` | Ya |
| 17 | THD-17 | D17 | PINTU MASUK SUMBU BARAT | Sumbu Barat | Aktif | Metro-E | `10.20.105.63` | Ya |
| 18 | THD-18 | D18 | Jalan Tol 1C | KIPP 1C | Nonaktif | GSM | `20.20.20.90` | Tidak |
| 19 | THD-19 | D19 | Sumbu Timur Luar | Jl. Negara | Aktif | Metro-E | `10.20.105.60` | Ya |
| 20 | THD-20 | D22 | Simpang Hauling (Belakang Waluyo) | KIPP 1B | Nonaktif | GSM | `20.20.20.122` | Tidak |
| 21 | THD-21 | D23 | Bundaran Sumbu Timur | Sumbu Timur | Nonaktif | Metro-E | `10.20.105.65` | Tidak |
| 22 | THD-22 | D24 | Pintu Masuk PSSI | PSSI | Nonaktif | Tidak Ada Layanan | `20.20.21.2` | Tidak |
| 23 | THD-23 | D25 | RTJM Pertigaan | RTJM | Nonaktif | Tidak Ada Layanan | `20.20.20.178` | Tidak |
| 24 | THD-24 | D26 | RTJM Jembatan | RTJM | Nonaktif | Tidak Ada Layanan | `20.20.20.130` | Tidak |
| 25 | THD-25 | D27 | Kantin Mandiri 1B | HPK 1B | Nonaktif | Tidak Ada Layanan | `20.20.20.242` | Tidak |
| 26 | THD-26 | D28 | Depan RS Hermina | Jl. Negara | Aktif | Metro-E | `10.20.105.70` | Ya |
| 27 | THD-27 | D29 | Simpang 4 Hauling (Starbright) | KIPP 1B | Nonaktif | GSM | `20.20.20.194` | Tidak |
| 28 | THD-28 | D30 | Jalan Masuk Kantor Otorita | Kantor Otorita | Aktif | Metro-E | `10.20.105.68` | Ya |
| 29 | THD-29 | D31 | Kantin BNI 1B | HPK 1B | Nonaktif | Tidak Ada Layanan | `20.20.21.50` | Tidak |
| 30 | THD-30 | D32 | Kantin BNI 2 1B | HPK 1B | Nonaktif | Tidak Ada Layanan | `20.20.21.51` | Tidak |
| 31 | THD-31 | D33 | Kantin HPK 1C | HPK 1C | Nonaktif | Tidak Ada Layanan | `20.20.21.34` | Tidak |
| 32 | THD-32 | D37 | TPST Menara Sanitasi | TPST | Nonaktif | GSM | `20.20.20.114` | Tidak |
| 33 | THD-33 | D38 | Gerbang Rumah Teknologi | Jl. Negara | Aktif | Metro-E | `10.20.105.57` | Ya |
| 34 | THD-34 | D39 | Simpang 4 Hauling (Pentol) | 1-B, 1-C | Aktif | GSM | `20.20.20.186` | Ya |
| 35 | THD-35 | D43 | Jalan Menuju Glamping | Glamping | Nonaktif | GSM | `20.20.20.234` | Tidak |

## Titik Koordinat dan Atribut Perangkat

Koordinat menggunakan format **Longitude, Latitude** (WGS84). Kolom **Sumber Koordinat** menunjukkan `Sumber` (dari daftar terbaru) atau `Estimasi` (area/fasilitas terdekat). `Sumber Listrik` sekarang berisi PLTS/PLN; tipe jaringan ada di kolom `Tipe Koneksi`.

<!-- CCTV-COORDINATES:START -->
| Kode | Longitude | Latitude | Sumber Koordinat | Jenis CCTV | Sumber Listrik | Tipe Koneksi | Indoor/Outdoor | Pemilik |
|---|---:|---:|---|---|---|---|---|---|
| THD-01 | 116.7207729 | -0.9663481 | Sumber data | PTZ | PLN | Tidak Ada Layanan | INDOOR | THD-OIKN |
| THD-02 | 116.7202849 | -0.9769854 | Sumber data | PTZ | PLTS | GSM | OUTDOOR | THD-OIKN |
| THD-03 | 116.7030018 | -0.9653249 | Sumber data | PTZ | PLTS | Metro-E | OUTDOOR | THD-OIKN |
| THD-04 | 116.714884 | -0.976202 | Sumber data | PTZ | PLTS | GSM | OUTDOOR | THD-OIKN |
| THD-05 | 116.7091789 | -0.977067574 | Sumber data | PTZ | PLTS | Metro-E | OUTDOOR | THD-OIKN |
| THD-06 | 116.7236283 | -0.9644617 | Sumber data | PTZ | PLTS | Metro-E | OUTDOOR | THD-OIKN |
| THD-07 | 116.7206501 | -0.966057418 | Sumber data | PTZ | PLN | Metro-E | INDOOR | THD-OIKN |
| THD-08 | 116.7205618 | -0.965843849 | Sumber data | PTZ | PLN | Metro-E | INDOOR | THD-OIKN |
| THD-09 | 116.7204397 | -0.965544996 | Sumber data | PTZ | PLN | Metro-E | INDOOR | THD-OIKN |
| THD-10 | 116.699964 | -0.962474 | Sumber data | PTZ | PLN | GSM | INDOOR | THD-OIKN |
| THD-11 | 116.7202569 | -0.9654453 | Sumber data | PTZ | PLN | Metro-E | OUTDOOR | THD-OIKN |
| THD-12 | 116.7231967 | -0.9644933 | Sumber data | PTZ | PLTS | Metro-E | OUTDOOR | THD-OIKN |
| THD-13 | 116.7152495 | -0.9685501 | Sumber data | PTZ | PLTS | Metro-E | OUTDOOR | THD-OIKN |
| THD-14 | 116.718156 | -0.954387 | Sumber data | PTZ | PLN | Tidak Ada Layanan | INDOOR | THD-OIKN |
| THD-15 | 116.695591 | -0.954508 | Sumber data | PTZ | PLTS | Metro-E | OUTDOOR | THD-OIKN |
| THD-16 | 116.7199752 | -0.9647164 | Sumber data | PTZ | PLTS | Metro-E | OUTDOOR | THD-OIKN |
| THD-17 | 116.7091789 | -0.977067574 | Sumber data | PTZ | PLTS | Metro-E | OUTDOOR | THD-OIKN |
| THD-18 | 116.7103708 | -0.972125 | Estimasi | PTZ | PLTS | GSM | OUTDOOR | THD-OIKN |
| THD-19 | 116.7119641 | -0.9743412 | Sumber data | PTZ | PLTS | Metro-E | OUTDOOR | THD-OIKN |
| THD-20 | 116.6739831 | -0.9894029 | Estimasi | PTZ | PLTS | GSM | OUTDOOR | THD-OIKN |
| THD-21 | 116.7110136 | -0.9711968 | Sumber data | PTZ | PLTS | Metro-E | OUTDOOR | THD-OIKN |
| THD-22 | 116.677241 | -0.982165 | Sumber data | PTZ | PLN | Tidak Ada Layanan | OUTDOOR | THD-OIKN |
| THD-23 | 116.6999483 | -0.9760081 | Sumber data | PTZ | PLTS | Tidak Ada Layanan | OUTDOOR | THD-OIKN |
| THD-24 | 116.70154 | -0.9765133 | Sumber data | PTZ | PLTS | Tidak Ada Layanan | OUTDOOR | THD-OIKN |
| THD-25 | 116.7119828 | -0.9501451 | Estimasi | PTZ | PLN | Tidak Ada Layanan | INDOOR | THD-OIKN |
| THD-26 | 116.713154 | -0.9718523 | Sumber data | PTZ | PLTS | Metro-E | OUTDOOR | THD-OIKN |
| THD-27 | 116.68574 | -0.9800554 | Estimasi | PTZ | PLTS | GSM | OUTDOOR | THD-OIKN |
| THD-28 | 116.709153 | -0.973653 | Sumber data | PTZ | PLTS | Metro-E | OUTDOOR | THD-OIKN |
| THD-29 | 116.7124279 | -0.9502616 | Estimasi | PTZ | PLN | Tidak Ada Layanan | INDOOR | THD-OIKN |
| THD-30 | 116.7125 | -0.9505 | Estimasi | PTZ | PLN | Tidak Ada Layanan | INDOOR | THD-OIKN |
| THD-31 | 116.7184 | -0.9546 | Estimasi | PTZ | PLN | Tidak Ada Layanan | INDOOR | THD-OIKN |
| THD-32 | 116.7076016 | -0.9856458 | Sumber data | PTZ | PLTS | GSM | OUTDOOR | THD-OIKN |
| THD-33 | 116.7192392 | -0.9650503 | Sumber data | PTZ | PLTS | Metro-E | OUTDOOR | THD-OIKN |
| THD-34 | 116.6997416 | -0.9678361 | Estimasi | PTZ | PLTS | GSM | OUTDOOR | THD-OIKN |
| THD-35 | 116.6906816 | -0.9509035 | Estimasi | PTZ | PLTS | GSM | OUTDOOR | THD-OIKN |
<!-- CCTV-COORDINATES:END -->

**Cakupan koordinat:** 35 dari 35 record. 26 record memakai koordinat dari sumber; 9 record bertanda **Estimasi**: THD-18, THD-20, THD-25, THD-27, THD-29, THD-30, THD-31, THD-34, THD-35. Estimasi menggunakan fasilitas/area terdekat dan belum diverifikasi lapangan.

**Atribut perangkat:** 35 record dari daftar terbaru; THD-01 kini juga terisi (PTZ, PLN, Tidak Ada Layanan, INDOOR, THD-OIKN).

## Catatan

- Kode CCTV dan channel mengikuti data sumber; pasangan angka tidak selalu sama (misalnya `THD-20` menggunakan channel `D22`).
- Kamera berstatus `aktif` diset `stream_enabled=true`; kamera `nonaktif` diset `stream_enabled=false` agar tidak membebani NVR.
- Dua record CCTV lama di database tidak diubah atau dihapus.
- `sumber_listrik` dan `tipe_koneksi` dikoreksi dari data terbaru: sebelumnya nilai jaringan (GSM/Metro-E) tersimpan di kolom sumber listrik dan `tipe_koneksi` kosong.
- Sebanyak 18 record memiliki IP `10.20.105.x`; `THD-01` bertipe `Tidak Ada Layanan` tetapi IP-nya berada di segmen Metro-E dan tidak diprobe karena statusnya nonaktif.
- URL RTSP lengkap tidak ditulis di arsip ini; hanya tersimpan terenkripsi di database. Username dan password tidak dimuat.
- Keterangan perangkat (termasuk nomor kartu GSM dan nomor panel) tersimpan di database, tidak dimuat di arsip ini.
- Status streaming saat ini `idle` untuk 20 kamera aktif karena seluruh path on-demand belum memiliki penonton; bukan indikasi kamera mati.
- Probe langsung Metro-E dapat diaktifkan dengan `CCTV_METRO_E_DIRECT_PROBE=true` setelah API berada di jaringan yang menjangkau `10.20.105.x`.
