# Manual Test - Kompatibilitas Codec H.265 dan Snapshot CCTV

| Item | Detail |
|------|--------|
| **Issue** | #149 (P2 - Kompatibilitas H.265 dan snapshot) |
| **Branch** | `fix/cctv-codec-snapshot` |
| **Versi Backend** | 0.64.39 (setelah fix) |
| **Versi Frontend** | 1.55.38 (setelah fix) |
| **Tanggal** | 2026-10-28 |

---

## 1. Prasyarat

- [ ] Stack berjalan: `db`, `redis`, `api`, `mediamtx`, `nginx`, `frontend`.
- [ ] Tersedia akun **admin** atau **teknisi** dengan akses ke minimal satu CCTV.
- [ ] Minimal satu CCTV `stream_enabled = true` yang dapat diputar dari aplikasi.
- [ ] Minimal satu kamera dengan **sub-stream H.264** (untuk skenario fallback)
  dan satu kamera dengan **H.265/HEVC** (untuk skenario codec tidak didukung),
  atau NVR yang encoding-nya dapat diubah sementara.
- [ ] Browser uji:
  - Chrome/Edge desktop versi terbaru (umumnya tidak mengiklankan HEVC di
    WebRTC) — untuk skenario HEVC tidak didukung.
  - Safari macOS/iOS (opsional) — untuk skenario browser yang mendukung HEVC.
- [ ] Log container API dapat dilihat: `docker compose logs -f api`.
- [ ] Akses database untuk verifikasi snapshot:
  `docker compose exec db psql -U sigap -d sigap`.
- [ ] Variabel shell disiapkan:

  ```bash
  export BASE_URL="http://localhost/api"
  export TOKEN="<TOKEN_AKSES>"
  export AUTH="Authorization: Bearer $TOKEN"
  export CCTV_ID="<UUID_CCTV>"
  ```

> **Catatan:** MediaMTX 1.20 **tidak** memiliki endpoint snapshot. Bila stack
> memakai versi tersebut, skenario snapshot harus berstatus `BLOCKED` dengan
> bukti log, bukan `FAIL`. Skenario codec tidak bergantung pada versi MediaMTX
> selama stream dapat diputar.

---

## 2. Pengujian Otomatis

Dijalankan dari `backend/`:

```bash
gofmt -l handlers/snapshot.go handlers/snapshot_test.go routes/cctv.go routes/routes_test.go
go build ./...
go vet ./...
go test ./handlers ./routes \
  -run 'TestGetCCTVSnapshotStatus|TestCCTVRoutesRegisterProtectedPaths' \
  -count=1 -v
go test ./... -count=1
golangci-lint run ./...
```

Dijalankan dari `frontend/`:

```bash
npm run test:hevc
npm run test:whep
npm run lint
npm run build
```

**Kriteria lulus:**

- `gofmt -l` tidak menampilkan file.
- `go build`, `go vet`, `go test ./...`, dan `golangci-lint` PASS (0 issues).
- `test:hevc` **10 PASS**, `test:whep` **5 PASS**.
- `npm run lint` 0 error; warning yang tersisa adalah warning lama di luar
  perubahan #149.
- `npm run build` PASS.

Test yang wajib PASS:

- `TestGetCCTVSnapshotStatus_EndpointUnavailable` — endpoint snapshot tidak
  didukung → `endpoint_supported=false`, `available=false`.
- `TestGetCCTVSnapshotStatus_NoSnapshot` — belum ada metadata → pesan jelas.
- `TestGetCCTVSnapshotStatus_Available` — metadata + file ada → `available=true`,
  `captured_at` dan `event_type` terisi.
- `TestGetCCTVSnapshotStatus_FileMissing` — file hilang → `available=false`.
- `TestGetCCTVSnapshotStatus_InvalidID` — UUID invalid → HTTP 400.
- `TestCCTVRoutesRegisterProtectedPaths` — rute `/snapshot-status` terdaftar
  dan tetap memerlukan autentikasi.
- `tests/hevc-support.test.mts` — 10 test deteksi HEVC.
- `tests/whep-status.test.mts` — 5 test transisi status WHEP.

---

## 3. Skenario Uji Black-box

### 3.1 Deteksi HEVC dan pesan yang jelas

| No | Langkah | Hasil yang diharapkan | Hasil nyata |
|----|---------|-----------------------|-------------|
| 1 | Buka CCTV dengan encoding **H.265/HEVC** dari Chrome/Edge desktop, lalu tekan **Mulai Stream**. | Pemutar berhenti melakukan retry setelah penolakan codec. Muncul overlay **"Codec video tidak didukung"** berisi panduan perangkat/H.264, bukan spinner tanpa akhir. | |
| 2 | Periksa toast error yang muncul bersamaan. | Toast memuat penyebab (codec H.265/HEVC), lokasi (negosiasi WebRTC/WHEP), dan tindakan (browser HEVC, ubah encoding ke H.264, atau coba kualitas lain). | |
| 3 | Di DevTools → Console, periksa tidak ada loop `POST /stream/.../whep` berulang setiap beberapa detik. | Maksimal satu percobaan untuk sub-stream dan satu untuk main stream; tidak ada retry eksponensial untuk error codec. | |
| 4 | Buka CCTV yang sama dari Safari macOS/iOS (bila tersedia). | Stream H.265 dapat diputar (badge **Live**), membuktikan masalah bersifat dukungan browser, bukan konfigurasi path MediaMTX. | |
| 5 | Di browser yang tidak mendukung HEVC, buka DevTools → Console lalu jalankan:<br>`RTCRtpReceiver.getCapabilities('video').codecs.map(c => c.mimeType)` | Daftar tidak memuat `video/H265`/`video/HEVC`, sesuai pesan yang ditampilkan. | |

### 3.2 Fallback sub-stream → main stream

| No | Langkah | Hasil yang diharapkan | Hasil nyata |
|----|---------|-----------------------|-------------|
| 6 | Siapkan kamera dengan **sub-stream H.265** dan **main stream H.264**. Buka dari Chrome/Edge dan mulai stream pada kualitas **SD**. | Pemutar mendeteksi penolakan codec, otomatis beralih ke **HD (main stream)**, lalu menampilkan badge **Live**. Tidak muncul overlay error final bila main stream berhasil. | |
| 7 | Amati indikator kualitas di modal. | Kualitas berubah menjadi **HD** dan tetap konsisten setelah stream stabil. | |
| 8 | Siapkan kamera dengan **kedua stream H.265**, lalu ulangi langkah 6. | Percobaan fallback hanya **satu kali**. Setelah main stream juga ditolak, overlay error final muncul; tidak ada loop sub → main → sub. | |
| 9 | Pada overlay error, tekan **Muat ulang**. | Pemutar mencoba koneksi sekali lagi dan kembali menampilkan overlay bila codec masih ditolak. Tidak terjadi retry tanpa batas. | |
| 10 | Pada overlay error, tekan **Coba sub-stream (SD)** / **Coba main stream (HD)**. | Kualitas berganti dan koneksi dibangun ulang. Bila berhasil, overlay hilang dan badge **Live** muncul. | |
| 11 | Setelah stream berhasil, tutup lalu buka kembali modal untuk kamera yang sama. | Fallback diizinkan lagi pada sesi baru (flag percobaan tidak terbawa antar-modal). | |

### 3.3 Snapshot saat endpoint MediaMTX tidak tersedia

| No | Langkah | Hasil yang diharapkan | Hasil nyata |
|----|---------|-----------------------|-------------|
| 12 | Pastikan stack memakai MediaMTX yang tidak memiliki endpoint snapshot (mis. 1.20), lalu restart API agar scheduler berjalan. | Log API menampilkan **satu** peringatan bahwa endpoint snapshot tidak tersedia; tidak ada spam error setiap 15 menit. | |
| 13 | Buka modal streaming CCTV mana pun. | Tombol **Snapshot** berstatus `disabled`, ikon kamera tetap tampil, dan alasan di bawah tombol menyebut MediaMTX tidak menyediakan endpoint snapshot. Tidak ada gambar rusak atau tab baru kosong. | |
| 14 | Panggil endpoint status:<br>`curl -s -H "$AUTH" "$BASE_URL/cctv/$CCTV_ID/snapshot-status" \| jq` | `endpoint_supported=false`, `available=false`, `message` menjelaskan endpoint tidak tersedia. | |
| 15 | Panggil endpoint gambar:<br>`curl -s -o /dev/null -w '%{http_code}\n' -H "$AUTH" "$BASE_URL/cctv/$CCTV_ID/snapshot"` | HTTP 404 dengan pesan JSON, bukan error 500 atau body kosong. | |

### 3.4 Snapshot tersedia

| No | Langkah | Hasil yang diharapkan | Hasil nyata |
|----|---------|-----------------------|-------------|
| 16 | Pada MediaMTX yang mendukung snapshot, tunggu scheduler berjalan (atau picu perubahan status CCTV) hingga ada baris di `cctv_snapshots`:<br>`SELECT id, event_type, image_path, created_at FROM cctv_snapshots WHERE cctv_id = '$CCTV_ID' ORDER BY created_at DESC LIMIT 1;` | Ada minimal satu snapshot dengan `image_path` yang menunjuk file di `uploads/snapshots/<cctv_id>/`. | |
| 17 | Panggil endpoint status. | `endpoint_supported=true`, `available=true`, `captured_at` terisi, `event_type` sesuai, `snapshot_url` = `/api/cctv/<uuid>/snapshot`. | |
| 18 | Buka modal streaming. | Tombol **Snapshot** aktif dengan tooltip berisi waktu snapshot terakhir. | |
| 19 | Klik **Snapshot**. | Tab baru membuka gambar JPEG terbaru. Tidak ada prompt login ulang karena cookie sesi dipakai. | |
| 20 | Bandingkan gambar dengan tampilan live kamera. | Gambar sesuai lokasi kamera dan waktu pengambilan; bukan placeholder atau gambar kamera lain. | |
| 21 | Hapus file snapshot dari disk (non-production), lalu muat ulang modal. | Status menjadi `available=false` dengan pesan "File snapshot tidak ditemukan di penyimpanan."; tombol nonaktif; tidak ada gambar rusak. | |

### 3.5 Snapshot belum pernah ada dan akses kontrol

| No | Langkah | Hasil yang diharapkan | Hasil nyata |
|----|---------|-----------------------|-------------|
| 22 | Pilih CCTV yang belum memiliki baris `cctv_snapshots`. Buka modal. | Tombol **Snapshot** nonaktif dengan pesan "Belum ada snapshot tersimpan untuk CCTV ini." | |
| 23 | Panggil endpoint status tanpa token. | HTTP 401; endpoint tidak membocorkan metadata snapshot. | |
| 24 | Panggil endpoint status dengan UUID yang tidak valid. | HTTP 400 "ID CCTV tidak valid". | |
| 25 | Panggil endpoint status dengan UUID valid tetapi tidak ada di database. | HTTP 404 "Data CCTV tidak ditemukan". | |
| 26 | Periksa isi respons endpoint status. | Respons hanya memuat `available`, `endpoint_supported`, `snapshot_url`, `captured_at`, `event_type`, dan `message`; tidak memuat `image_path` absolut, kredensial RTSP, atau path filesystem server. | |

### 3.6 Regresi timeline dan pemutar biasa

| No | Langkah | Hasil yang diharapkan | Hasil nyata |
|----|---------|-----------------------|-------------|
| 27 | Buka timeline CCTV yang memiliki snapshot. | Thumbnail tampil; event tanpa file menampilkan placeholder "Foto tidak dapat ditampilkan", bukan error halaman. | |
| 28 | Putar kamera H.264 pada kualitas SD dan HD. | Kedua kualitas berfungsi seperti sebelumnya; tidak ada overlay codec palsu. | |
| 29 | Jeda dan lanjutkan pemutaran, lalu masuk/keluar fullscreen. | Kontrol tetap berfungsi; status **Live** tidak hilang karena perubahan kualitas. | |
| 30 | Tutup modal saat stream sedang menghubungkan. | Tidak ada error console akibat `setState` setelah unmount; koneksi WHEP ditutup. | |

---

## 4. Verifikasi Database dan Log

```sql
-- Snapshot terbaru per CCTV
SELECT c.id, c.nama, s.event_type, s.image_path, s.created_at
FROM cctvs c
LEFT JOIN LATERAL (
  SELECT event_type, image_path, created_at
  FROM cctv_snapshots
  WHERE cctv_id = c.id
  ORDER BY created_at DESC
  LIMIT 1
) s ON true
WHERE c.id = '<UUID_CCTV>';
```

```bash
# Satu peringatan endpoint snapshot, bukan berulang
docker compose logs api | grep -i 'snapshot'

# Tidak ada loop WHEP pada kamera H.265
docker compose logs nginx | grep -i 'whep' | tail -20
```

**Kriteria lulus:**

- Tidak ada error 500 pada endpoint snapshot/status.
- Tidak ada pertumbuhan file snapshot melebihi 10 per CCTV.
- Log tidak dibanjiri percobaan WHEP berulang untuk kamera H.265.
- Endpoint status tidak mengembalikan path filesystem absolut, kredensial RTSP,
  atau data internal lain.

---

## 5. Pembersihan

- [ ] Kembalikan encoding kamera/NVR ke konfigurasi semula bila diubah untuk uji.
- [ ] Hapus snapshot uji yang ditandai (non-production):
  `DELETE FROM cctv_snapshots WHERE notes LIKE 'UJI CODEC%';`
- [ ] Hapus file uji di `uploads/snapshots/<cctv_id>/` bila metadata sudah dihapus.
- [ ] Tutup semua tab snapshot dan stream.
- [ ] Pastikan tidak ada perubahan konfigurasi MediaMTX yang tertinggal.

---

## 6. Catatan

- Deteksi HEVC bersifat **best-effort**. Browser yang mengiklankan HEVC tetapi
  gagal mendekode frame akan tetap masuk ke overlay error setelah timeout frame.
- SIGAP tidak melakukan transcoding. Perubahan codec harus dilakukan di
  kamera/NVR; lihat `docs/01-documentation/cctv-codec-snapshot-limitations.md`.
- Bila MediaMTX 1.20 tidak memiliki endpoint snapshot, snapshot baru tidak dapat
  diambil. Snapshot lama tetap dapat dilihat; tombol dinonaktifkan agar UI tidak
  rusak.
- Hasil manual test ini harus dicatat dengan metadata lengkap sesuai kontrak di
  `docs/04-manual-tests/TESTING.md` (build SHA, tester, waktu WITA, environment,
  evidence).

---

## 7. Catatan Eksekusi Formal (Kontrak `TESTING.md`)

### Formal execution metadata

| Field | Value |
|---|---|
| Test ID / Audit ID | MT-149 |
| Capability owner | NOT RECORDED |
| Build SHA | NOT RECORDED |
| Artifact digest/ID | NOT RECORDED |
| Tester | NOT RECORDED |
| Timestamp WITA | NOT RECORDED |
| Environment | NOT RECORDED |
| Fixture prefix/data | E2E-MT149- / NOT RECORDED |
| Cleanup result | NOT RUN |
| Expected | Overlay codec H.265 menghentikan retry, fallback sub -> main 1x berjalan tanpa loop, dan tombol snapshot mengikuti `/cctv/:id/snapshot-status` tanpa membuka URL 404 |
| Actual | NOT RECORDED |
| Evidence path/link | NOT RECORDED |
| Mandatory flag | YES |
| Scenario status | NOT RUN |
| Status suite | NOT RUN |

### Formal scenario record

| ID | Mandatory | Expected | Actual | Evidence | Status |
|---|---|---|---|---|---|
| MT-149-S1 | YES | Seluruh skenario 1–30 (codec HEVC, fallback sub -> main, snapshot status, dan regresi) berjalan sesuai hasil yang diharapkan | NOT RECORDED | NOT RECORDED | NOT RUN |

Suite conclusion: sebelum eksekusi `NOT RUN`; setelah run dimulai, setiap mandatory scenario harus `PASS`, atau suite `FAIL`.
