# Manual Test - Release Gate Bug 4, 8, dan 21

| Item | Detail |
|------|--------|
| **Issue** | #147 (P1 - Selesaikan manual test Bug 4, 8, dan 21) |
| **Branch** | `main` (`1f984bb` saat eksekusi production) |
| **Versi Backend** | 0.64.38 |
| **Versi Frontend** | 1.55.37 |
| **Tanggal eksekusi** | 2026-11-13 (WITA) |
| **Status** | **LULUS (Production `https://nexaspace.site`)** — Bug 4, Bug 8 (publik desktop & seluler), dan Bug 21 lulus serta `docs/README.md` telah diperbarui (`25/25`) |

---

## 1. Tujuan dan Batas Pengujian

Dokumen ini adalah prosedur blackbox untuk menutup tiga bug yang masih terbuka di
`docs/README.md`:

| Bug | Ringkasan | Kode terkait | Gate |
|-----|-----------|--------------|------|
| **Bug 4** | Filter kawasan aset CCTV melalui UI | Halaman `/cctv`, query `site_id` | Wajib lulus |
| **Bug 8** | MediaMTX pada production, jaringan publik, dan seluler | Streaming WHEP/ICE, `MEDIAMTX_WEBRTC_ADDITIONAL_HOSTS` | Wajib lulus |
| **Bug 21** | Edit CCTV + URL RTSP melalui UI | Modal edit CCTV, parser `site_id`/`zone_id` null, `channel_port`/`rtsp_password` | Wajib lulus |

Pengujian dilakukan pada **production** atau **staging yang identik dengan
production** (domain HTTPS, Nginx, MediaMTX, dan jaringan publik). Hasil lokal
tanpa TLS/jaringan publik tidak cukup untuk menutup Bug 8.

> **Aturan bukti:** setiap skenario harus memiliki minimal satu bukti
> screenshot/video/log. Jangan mencentang `docs/README.md` sebelum ketiga bug
> lulus dan bukti dilampirkan.

---

## 2. Prasyarat

- [ ] Akses ke `https://<domain-production>` melalui Nginx (bukan port internal).
- [ ] Akun **admin** aktif dan akun **teknisi** dengan site assignment.
- [ ] Minimal dua site dengan CCTV berbeda (mis. Site A dan Site B).
- [ ] Minimal satu CCTV aktif dengan stream H.264 yang dapat dibuka.
- [ ] Perangkat uji:
  - [ ] Desktop Chrome/Edge (jaringan kantor/publik).
  - [ ] Android Chrome dengan **data seluler** (Wi-Fi dimatikan).
  - [ ] iPhone Safari dengan **data seluler** (opsional, bila tersedia).
- [ ] DevTools browser dapat dibuka (tab Network/Console).
- [ ] `psql` atau akses database read-only untuk verifikasi.
- [ ] Folder bukti dibuat:
  `docs/05-reports/evidence/2026-10-28-release-gate/` (atau folder terpisah
  yang disepakati; jangan commit data sensitif).

### 2.1 Format bukti

Gunakan nama file yang menyebut bug, skenario, dan lingkungan:

```text
bug4-site-filter-desktop-20261028.png
bug4-select-all-print-qr-20261028.pdf
bug8-live-desktop-public-20261028.mp4
bug8-ice-candidates-public-20261028.png
bug8-mobile-cellular-20261028.mp4
bug21-edit-cctv-rtsp-desktop-20261028.png
bug21-network-response-20261028.har
bug21-db-verification-20261028.txt
```

Jangan menyertakan cookie sesi, token, password RTSP, atau isi `.env` pada
screenshot/HAR. Redaksi nilai sensitif sebelum diunggah.

---

## 3. Pengujian Otomatis Pendukung

Dijalankan sebelum pengujian manual (dari root repository):

```bash
cd backend
go build ./...
go vet ./...
go test ./... -count=1
golangci-lint run ./...

cd ../frontend
npm run lint
npm run build
```

**Kriteria lulus:** seluruh perintah PASS. Pengujian otomatis ini hanya
prasyarat; tidak menggantikan pengujian manual.

---

## 4. Bug 4 - Filter Kawasan Aset CCTV

**Tujuan:** memastikan filter kawasan pada halaman CCTV memengaruhi total,
paginasi, pemilihan semua, cetak QR, dan sortir kawasan secara konsisten.

### 4.1 Skenario

| No | Langkah | Hasil yang diharapkan | Hasil nyata |
|----|---------|-----------------------|-------------|
| 4.1 | Login sebagai admin, buka `/cctv`, dan catat jumlah total data tanpa filter. | Tabel menampilkan seluruh CCTV yang boleh diakses admin; total sesuai data database. | |
| 4.2 | Pilih **Site A** pada dropdown kawasan. Catat total dan jumlah baris per halaman. | Seluruh baris hanya milik Site A; total berubah sesuai jumlah CCTV Site A. Tidak ada baris Site B. | |
| 4.3 | Pindah ke halaman 2 (dan 3 bila ada) pada filter Site A. | Paginasi tetap membawa filter Site A (query `site_id`), tidak kembali ke data semua site. | |
| 4.4 | Klik **Pilih semua** pada halaman yang sedang difilter, lalu periksa jumlah item terpilih. | Hanya baris hasil filter Site A yang terpilih; jumlahnya tidak melebihi total Site A. | |
| 4.5 | Dengan pilihan tersebut, klik **Cetak QR** dan simpan PDF. | PDF hanya memuat CCTV Site A yang dipilih; tidak ada CCTV Site B. | |
| 4.6 | Ubah sortir tabel ke **nama kawasan** (ascending/descending). | Urutan berubah konsisten di seluruh halaman; sortir diproses server-side dan filter site tetap aktif. | |
| 4.7 | Ganti filter ke **Site B**, lalu ulangi langkah 4.2-4.4. | Hasil hanya Site B; tidak ada kebocoran data Site A. | |
| 4.8 | Hapus filter kawasan (pilih "Semua"). | Total kembali ke jumlah seluruh site; paginasi dan pilihan sebelumnya direset. | |
| 4.9 | Login sebagai **teknisi** yang ditugaskan ke Site A, buka `/cctv`. | Hanya CCTV Site A yang tampil; dropdown/filter tidak membuka akses ke Site B. | |
| 4.10 | Cross-check API:<br>`GET /api/cctv?site_id=<SITE_A>&page=1&limit=10` | `data.pagination.total_rows` sama dengan total UI; seluruh `site_id` baris = Site A. | |

### 4.11 Bukti Bug 4

- [ ] Screenshot filter Site A + total/paginasi.
- [ ] Screenshot pilihan semua + jumlah terpilih.
- [ ] PDF hasil cetak QR (hanya Site A).
- [ ] Screenshot/JSON respons API `site_id`.
- [ ] Screenshot tampilan teknisi (scope site).

**Status Bug 4:** [ ] Lulus [ ] Gagal — catatan: ______________________

---

## 5. Bug 8 - MediaMTX Production, Jaringan Publik, dan Seluler

**Tujuan:** membuktikan frame live benar-benar tampil di production melalui
domain publik dan jaringan seluler, bukan hanya di localhost/LAN.

### 5.1 Pemeriksaan konfigurasi sebelum uji

| No | Langkah | Hasil yang diharapkan | Hasil nyata |
|----|---------|-----------------------|-------------|
| 8.1 | Periksa root `.env` production: `MEDIAMTX_WEBRTC_ADDITIONAL_HOSTS`. | Berisi domain publik atau IP publik yang dapat dijangkau browser; **bukan** `172.28.x.x` atau `localhost`. | |
| 8.2 | Periksa container MediaMTX: `docker compose ps mediamtx` dan log startup. | Container `healthy`; tidak ada error konfigurasi `MTX_WEBRTCADDITIONALHOSTS`. | |
| 8.3 | Periksa `mediamtx.yml`: `webrtcIPsFromInterfaces: false` dan `webrtcAdditionalHosts: []` (nilai dari env). | Tidak ada kandidat ICE bridge Docker yang diiklankan ke browser. | |
| 8.4 | Pastikan port WebRTC dapat diakses dari luar: TCP/UDP `8189` (atau sesuai konfigurasi ICE/TURN). | Port terbuka di firewall/security group; tidak diblokir ISP. | |
| 8.5 | Periksa sertifikat/TLS domain: `curl -I https://<domain>`. | HTTP 200/302 dengan sertifikat valid; tidak ada peringatan mixed content. | |

### 5.2 Desktop - jaringan publik

| No | Langkah | Hasil yang diharapkan | Hasil nyata |
|----|---------|-----------------------|-------------|
| 8.6 | Dari jaringan **di luar LAN** (hotspot/data seluler atau kantor lain), login sebagai admin dan buka live view CCTV H.264. | Video bergerak tampil dalam ≤10 detik; overlay status berubah ke `live`. | |
| 8.7 | Rekam 15-30 detik video dan buka DevTools → Network. Periksa request `/stream/cam-<uuid>/whep`. | Respons sukses (200/201), `Content-Type` sesuai, dan tidak ada error CORS/TLS. | |
| 8.8 | Di DevTools → Console, periksa kandidat ICE/log WebRTC. | Kandidat yang dipakai adalah host publik/domain; **tidak ada** kandidat `172.28.x.x` yang menjadi jalur aktif. | |
| 8.9 | Tunggu 60 detik, lalu toggle kualitas SD ↔ HD. | Stream tetap berjalan atau reconnect otomatis; tidak ada frame freeze permanen. | |
| 8.10 | Putuskan jaringan (Wi-Fi off/on atau ganti jaringan), lalu amati player. | Player menampilkan status reconnect dan pulih otomatis ≤30 detik tanpa refresh halaman. | |
| 8.11 | Buka halaman `/monitor` (grid) dan ulangi langkah 8.6 pada minimal dua kamera. | Kedua sel menampilkan frame; tidak ada placeholder/error yang menetap. | |

### 5.3 Seluler - data seluler

| No | Langkah | Hasil yang diharapkan | Hasil nyata |
|----|---------|-----------------------|-------------|
| 8.12 | Di Android Chrome, matikan Wi-Fi dan gunakan data seluler. Login dan buka live view. | Frame tampil dalam ≤15 detik pada kualitas sub-stream. | |
| 8.13 | Rekam video 15-30 detik yang menunjukkan jam perangkat, indikator seluler, dan frame bergerak. | Bukti jelas menunjukkan lingkungan seluler dan video live. | |
| 8.14 | Lakukan pada iPhone Safari bila tersedia. | Perilaku sama; bila gagal, catat pesan error dan kandidat ICE. | |
| 8.15 | Pindah dari Wi-Fi ke seluler saat stream berjalan. | Player reconnect dan kembali live; tidak perlu login ulang. | |

### 5.4 Diagnosis bila gagal

Kumpulkan:

```bash
docker compose logs --tail=200 mediamtx
docker compose logs --tail=200 api
docker compose ps
```

Serta screenshot DevTools Console/Network dan kandidat ICE. Klasifikasikan
kegagalan:

- **ICE tidak punya kandidat publik** → periksa `MEDIAMTX_WEBRTC_ADDITIONAL_HOSTS`
  dan firewall UDP/TCP 8189.
- **WHEP 404** → path belum terdaftar; cek start stream/registrasi MediaMTX.
- **Video freeze di seluler** → kemungkinan NAT operator; dokumentasikan sebagai
  risiko dan pertimbangkan TURN (di luar scope #147).
- **Sertifikat/mixed content** → periksa TLS Nginx dan URL WHEP (`https/wss`).

### 5.5 Bukti Bug 8

- [ ] Video desktop publik 15-30 detik.
- [ ] Screenshot DevTools request WHEP + kandidat ICE.
- [ ] Video Android seluler 15-30 detik (indikator seluler terlihat).
- [ ] Screenshot hasil iPhone Safari (bila diuji).
- [ ] Log MediaMTX/API saat reconnect.
- [ ] Screenshot `/monitor` dua kamera.

**Status Bug 8:** [ ] Lulus [ ] Gagal — catatan: ______________________

---

## 6. Bug 21 - Edit CCTV dan URL RTSP

**Tujuan:** memastikan edit CCTV dengan `site_id: null`/`zone_id: null` dan
perubahan konfigurasi RTSP berhasil melalui UI, tersimpan, dan tidak
mengekspos password.

> Gunakan **CCTV uji** atau staging. Jangan mengubah kamera production tanpa
> persetujuan operator. Catat nilai sebelum edit.

### 6.1 Skenario

| No | Langkah | Hasil yang diharapkan | Hasil nyata |
|----|---------|-----------------------|-------------|
| 21.1 | Buka `/cctv`, pilih satu CCTV uji, klik **Edit**. Catat `site_id`, `zone_id`, `channel_port`, dan status streaming. | Modal terbuka dengan nilai saat ini; tidak ada error console. | |
| 21.2 | Kosongkan pilihan **Site** dan **Zona** (biarkan null), ubah `channel_port` ke nilai valid lain, lalu simpan. | Request `PUT /api/cctv/:id` mengembalikan **200**; tidak ada error `cannot unmarshal null` atau 400. | |
| 21.3 | Periksa tab Network: payload berisi `site_id: null` dan `zone_id: null` bersama field RTSP. | Payload valid; response `status="success"` dan data CCTV terbaru. | |
| 21.4 | Muat ulang halaman dan buka kembali modal edit. | `site_id`/`zone_id` tetap kosong (null), `channel_port` baru tersimpan, field lain tidak berubah. | |
| 21.5 | Isi `rtsp_password` baru, simpan, lalu buka kembali modal. | Password tidak pernah dikembalikan dalam respons/UI (field kosong atau masked); tidak ada plaintext di Network/console. | |
| 21.6 | Periksa preview URL RTSP di modal. | URL memakai channel yang benar (`channel_port × 100 + stream_type`) dan host/NVR yang berlaku; tidak ada password di URL yang ditampilkan ke UI. | |
| 21.7 | Pilih Site A + Zona yang valid, simpan, lalu edit lagi. | Kombinasi site/zona tersimpan dan tetap konsisten setelah reload. | |
| 21.8 | Coba simpan dengan `channel_port` di luar 1-35. | Validasi menolak dengan pesan jelas (400/422); data lama tidak berubah. | |
| 21.9 | Verifikasi database (redaksi kredensial):<br>`SELECT id, site_id, zone_id, channel_port, stream_enabled FROM cctvs WHERE id = '<UUID>';` | Nilai sesuai hasil edit; `rtsp_password` tersimpan terenkripsi (bukan plaintext). | |
| 21.10 | Periksa log API setelah edit. | Tidak ada password RTSP/plaintext di log. | |

### 6.2 Bukti Bug 21

- [ ] Screenshot modal sebelum/sesudah edit (site/zona null).
- [ ] Screenshot response 200 + payload `site_id: null`.
- [ ] Screenshot modal setelah reload (nilai tersimpan).
- [ ] Screenshot field password tidak mengembalikan plaintext.
- [ ] Output query database (redaksi).
- [ ] Potongan log API tanpa kredensial.

**Status Bug 21:** [ ] Lulus [ ] Gagal — catatan: ______________________

---

## 7. Kriteria Lulus dan Tindak Lanjut

### 7.1 Kriteria lulus

- [ ] Bug 4: filter site konsisten pada total, paginasi, pilih semua, cetak QR,
  dan sortir; teknisi tidak bocor ke site lain.
- [ ] Bug 8: frame live tampil di production melalui domain publik, desktop
  jaringan publik, dan Android seluler; reconnect berjalan; tidak ada kandidat
  ICE bridge Docker.
- [ ] Bug 21: edit CCTV dengan `site_id`/`zone_id` null dan URL RTSP berhasil
  (HTTP 200), tersimpan setelah reload, dan password tidak terekspos.
- [ ] Semua bukti tersimpan dan bebas dari kredensial sensitif.
- [ ] Tidak ada error 500/console error yang belum dijelaskan.

### 7.2 Bila lulus

1. Lampirkan tautan bukti pada issue #147.
2. Perbarui `docs/README.md`:
   - centang Bug 4, 8, dan 21;
   - ubah ringkasan menjadi **25/25 (100%)**;
   - tambahkan tanggal/environment pengujian.
3. Komentari issue #147 dengan ringkasan hasil dan tautan bukti.
4. Baru setelah itu nyatakan gate manual test selesai.

### 7.3 Bila gagal

1. Jangan centang `docs/README.md`.
2. Buat issue bug baru (atau komentar #147) berisi langkah reproduksi, hasil
   aktual, dan bukti.
3. Klasifikasikan apakah kegagalan memblokir rilis (Bug 8 selalu memblokir
   klaim production-ready).
4. Perbaiki kode, tambahkan test regresi, lalu ulangi skenario terkait.

---

## 8. Sign-off

| Peran | Nama | Tanggal | Hasil |
|-------|------|---------|-------|
| Penguji UI (Bug 4 & 21) | | | |
| Penguji streaming (Bug 8) | | | |
| Reviewer/Release manager | | | |

**Kesimpulan:**

- [ ] Ketiga bug lulus → lanjut memperbarui `docs/README.md` (25/25).
- [ ] Ada kegagalan → rilis ditahan; issue perbaikan dibuat.
- [ ] Risiko diterima secara eksplisit (jelaskan): ______________________

---

## 9. Catatan

- Dokumen ini tidak mengubah status bug. Status tetap **belum lulus** sampai
  bukti manual terlampir.
- Bug 8 tidak dapat dinyatakan lulus hanya dari localhost atau LAN; harus ada
  bukti jaringan publik/seluler.
- Untuk pengujian di staging, gunakan domain HTTPS dan MediaMTX yang
  konfigurasinya sama dengan production.
- Jangan mengunggah file `.env`, HAR lengkap yang memuat cookie, atau video yang
  memperlihatkan area sensitif tanpa persetujuan.
