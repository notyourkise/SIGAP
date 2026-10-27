# Manual Test - Kebijakan Event Standby dan Deduplikasi Log CCTV

| Item | Detail |
|------|--------|
| **Issue** | #146 (P1 - Kebijakan event standby dan deduplikasi log) |
| **Branch** | `fix/cctv-log-standby-dedup` |
| **Versi Backend** | 0.64.38 (setelah fix) |
| **Versi Frontend** | 1.55.37 (tidak berubah) |
| **Tanggal** | 2026-10-28 |

---

## 1. Prasyarat

- [ ] Stack berjalan: `db`, `redis`, `api`, `mediamtx`, `nginx`, `frontend`.
- [ ] Akses `psql` ke database SIGAP:
  `docker compose exec db psql -U sigap -d sigap`.
- [ ] Tersedia akun **admin** dan token akses yang valid.
- [ ] Minimal satu CCTV aktif dengan `stream_enabled = true`.
- [ ] Untuk skenario GSM idle: kamera GSM yang path MediaMTX-nya terdaftar
  tetapi **tidak sedang ditonton** (tidak ada viewer).
- [ ] `CCTV_LOG_STANDBY_EVENTS` dan `CCTV_VERIFY_INTERVAL_MINUTES` dapat diubah
  pada `docker-compose.yml`/`.env`, lalu API dapat direstart.
- [ ] Log container API dapat dilihat: `docker compose logs -f api`.
- [ ] Variabel shell disiapkan:

  ```bash
  export BASE_URL="http://localhost/api"
  export TOKEN="<TOKEN_AKSES>"
  export AUTH="Authorization: Bearer $TOKEN"
  ```

- [ ] Catat baseline log per tipe:

  ```sql
  SELECT event_type, count(*)
  FROM cctv_stream_logs
  GROUP BY event_type
  ORDER BY event_type;
  ```

> **Catatan:** probe berjalan setiap **2 menit** dan verifier setiap
> **15 menit** (default). Skenario yang menunggu siklus probe membutuhkan waktu
> sampai ±2 menit per siklus; gunakan endpoint verifikasi manual untuk skenario
> yang lebih cepat.

---

## 2. Pengujian Otomatis

Dijalankan dari `backend/`:

```bash
gofmt -l config/cctv_log.go config/cctv_log_test.go \
  handlers/cctv_stream_log_events.go handlers/cctv_stream_log_events_test.go \
  handlers/stream_probe.go handlers/stream_probe_test.go \
  handlers/stream_verify.go handlers/stream_verify_test.go
go build ./...
go vet ./...
go test ./config/ ./handlers/ \
  -run 'TestCCTVLogStandbyEvents|TestShouldLogCCTVStreamEvent|TestRunProbeCycle|TestVerifyCCTVStream' \
  -count=1 -v
go test ./... -count=1
golangci-lint run ./...
```

**Kriteria lulus:** tidak ada file yang diubah perlu diformat, semua perintah
PASS, dan `golangci-lint` 0 issues.

Test yang wajib PASS:

- `TestCCTVLogStandbyEvents_DefaultFalse` — env kosong → `false`.
- `TestCCTVLogStandbyEvents_Configured` — `true`/`1`/`t` → aktif;
  `false`/`0`/`f` → nonaktif; `yes` tidak diterima.
- `TestCCTVLogStandbyEvents_InvalidFallsBackToFalse` — nilai tidak valid →
  `false`.
- `TestShouldLogCCTVStreamEvent_StandbyDisabledByDefault` — standby tidak
  dicatat dan tidak ada query database.
- `TestShouldLogCCTVStreamEvent_StandbyEnabled` — standby dicatat saat
  diaktifkan.
- `TestShouldLogCCTVStreamEvent_DuplicateWithinWindow` — tipe sama dalam 60
  detik → tidak dicatat.
- `TestShouldLogCCTVStreamEvent_DifferentEventWithinWindow` — tipe berbeda
  dalam 60 detik → dicatat.
- `TestShouldLogCCTVStreamEvent_SameEventOutsideWindow` — tipe sama setelah 60
  detik → dicatat.
- `TestShouldLogCCTVStreamEvent_NoPreviousLog` — belum ada log → dicatat.
- `TestShouldLogCCTVStreamEvent_DatabaseErrorFailsOpen` — query error → tetap
  dicatat.
- `TestRunProbeCycle_GSM` dan `TestVerifyCCTVStreamMarksLive...` — ekspektasi
  sqlmock mencakup query dedup + INSERT log secara eksplisit.

---

## 3. Skenario Uji

### 3.1 Standby nonaktif (default)

| No | Langkah | Hasil yang diharapkan | Hasil nyata |
|----|---------|-----------------------|-------------|
| 1 | Pastikan `CCTV_LOG_STANDBY_EVENTS=false` (atau tidak diset), lalu restart API. | API berjalan normal; log startup tidak menampilkan error konfigurasi. | |
| 2 | Pilih satu CCTV GSM yang path-nya terdaftar tetapi tidak sedang ditonton. Catat `stream_state`, `stream_checked_at`, dan jumlah log `standby`:<br>`SELECT id, nama, stream_state, stream_checked_at FROM cctvs WHERE id = '<UUID>';`<br>`SELECT count(*) FROM cctv_stream_logs WHERE cctv_id = '<UUID>' AND event_type = 'standby';` | `stream_state` = `idle`. Simpan nilai `stream_checked_at` dan jumlah log. | |
| 3 | Tunggu dua siklus probe (±4–5 menit) tanpa membuka stream. Periksa kembali state dan jumlah log standby. | `stream_checked_at` **diperbarui** setiap siklus (probe tetap berjalan), `stream_state` tetap `idle`, dan jumlah log `standby` **tidak bertambah**. Tidak ada event `standby` baru. | |
| 4 | Periksa daftar log terbaru kamera tersebut:<br>`SELECT event_type, trigger_source, current_state, created_at FROM cctv_stream_logs WHERE cctv_id = '<UUID>' ORDER BY created_at DESC LIMIT 10;` | Tidak ada baris baru bertipe `standby` dari siklus probe; event lama (bila ada) tetap utuh. | |
| 5 | Buka halaman Log CCTV dan amati daftar. | Tidak ada entri `standby` baru; entri `online`/`offline` lama tetap tampil dan dapat difilter. | |

### 3.2 Standby diaktifkan

| No | Langkah | Hasil yang diharapkan | Hasil nyata |
|----|---------|-----------------------|-------------|
| 6 | Set `CCTV_LOG_STANDBY_EVENTS=true`, restart API, dan picu transisi ke `idle` (mis. tutup semua viewer stream on-demand). | Ketika state berubah menjadi `idle`, muncul **satu** log `standby` dengan `trigger_source = probe` dan `message` "Kamera siaga on-demand...". | |
| 7 | Tunggu dua siklus probe berikutnya tanpa perubahan state. | Tidak ada log `standby` tambahan karena status tidak berubah (event hanya ditulis saat transisi) dan deduplikasi 60 detik berlaku. | |
| 8 | Buka lalu tutup viewer dalam waktu <60 detik sehingga state bergerak `live` → `idle` (atau sebaliknya). | Event `standby` kedua dalam 60 detik **tidak** ditulis bila log terakhir juga `standby`; transisi `online` tetap dicatat karena tipe berbeda. | |
| 9 | Set `CCTV_LOG_STANDBY_EVENTS=1`, restart, ulangi langkah 6. | Perilaku sama seperti `true` (ParseBool menerima `1`). | |
| 10 | Set `CCTV_LOG_STANDBY_EVENTS=yes` (tidak valid), restart, ulangi langkah 6. | Standby kembali **tidak** dicatat; nilai tidak valid jatuh ke default `false`. | |

### 3.3 Deduplikasi `online`/`offline` antara probe dan verifier

| No | Langkah | Hasil yang diharapkan | Hasil nyata |
|----|---------|-----------------------|-------------|
| 11 | Pilih CCTV yang baru saja menghasilkan log `online` dari probe (dalam <60 detik). Catat `id` log terakhir. | Ada satu log `online` terbaru dengan `trigger_source = probe`. | |
| 12 | Segera panggil verifikasi manual:<br>`curl -s -X POST -H "$AUTH" -H "X-CSRF-Token: $CSRF" "$BASE_URL/streams/<CCTV_UUID>/verify" \| jq` | Verifikasi berhasil (state `live`), tetapi **tidak ada** log `online` baru karena tipe sama masih dalam jendela 60 detik. Kolom `stream_verified_*` tetap diperbarui. | |
| 13 | Periksa log:<br>`SELECT event_type, trigger_source, created_at FROM cctv_stream_logs WHERE cctv_id = '<UUID>' ORDER BY created_at DESC LIMIT 5;` | Hanya satu baris `online` untuk kejadian tersebut; tidak ada duplikat `probe` + `verify`. | |
| 14 | Mundurkan waktu log terakhir agar berada di luar jendela:<br>`UPDATE cctv_stream_logs SET created_at = NOW() - INTERVAL '61 seconds' WHERE id = '<LOG_ID>';`<br>Lalu picu transisi yang sama (mis. verifikasi manual setelah state terverifikasi direset ke `unknown`). | Event `online` baru **dicatat** karena jarak >60 detik. | |
| 15 | Dalam <60 detik setelah log `online`, hentikan sumber stream sehingga probe/verifier menyimpulkan `offline`. | Event `offline` **tetap dicatat** karena tipe berbeda; deduplikasi hanya berlaku untuk tipe yang sama. | |
| 16 | Jalankan query deteksi duplikat pada dokumen semantics (`docs/01-documentation/cctv-stream-log-semantics.md` §5.1). | Query mengembalikan **0 baris** pada kondisi normal. | |

### 3.4 Event penting tetap tercatat

| No | Langkah | Hasil yang diharapkan | Hasil nyata |
|----|---------|-----------------------|-------------|
| 17 | Matikan akses ke sumber satu kamera (mis. cabut kabel/block IP) dan tunggu probe. | Event `offline` tercatat dengan `trigger_source = probe`, status `stream_live=false`, dan badge UI berubah menjadi offline. | |
| 18 | Pulihkan akses dan tunggu probe berikutnya. | Event `online` tercatat, status `stream_live=true`, dan badge UI kembali live. | |
| 19 | Ulangi siklus mati/hidup dua kali dengan jeda >60 detik antar-transisi. | Setiap transisi menghasilkan satu event; tidak ada event yang hilang karena deduplikasi. | |
| 20 | Bandingkan jumlah `online`/`offline` sebelum dan sesudah pengujian. | Jumlah bertambah sesuai jumlah transisi nyata, bukan sesuai jumlah siklus probe. | |

### 3.5 Fail-open dan ketahanan database

| No | Langkah | Hasil yang diharapkan | Hasil nyata |
|----|---------|-----------------------|-------------|
| 21 | Jalankan `TestShouldLogCCTVStreamEvent_DatabaseErrorFailsOpen`. | Test PASS: error query dedup tidak menghilangkan event. | |
| 22 | (Opsional, non-production) Hentikan container `db` sesaat saat probe berjalan, lalu hidupkan kembali. Periksa log API. | API mencatat warning kegagalan dedup/penulisan log, tetapi proses probe tidak berhenti dan status kamera lain tetap diperbarui. Setelah DB hidup, event baru kembali tercatat. | |
| 23 | Periksa log API untuk pesan `Gagal memeriksa log terakhir ... untuk deduplikasi`. | Pesan hanya muncul saat database bermasalah; bukan pada operasi normal. | |

### 3.6 Regresi retensi dan UI

| No | Langkah | Hasil yang diharapkan | Hasil nyata |
|----|---------|-----------------------|-------------|
| 24 | Pastikan `CCTV_STREAM_LOG_RETENTION_DAYS=90` dan worker cleanup berjalan. | Log `standby` (bila diaktifkan) dan `online`/`offline` lebih tua dari 90 hari tetap terhapus sesuai kebijakan retensi. | |
| 25 | Buka halaman Log CCTV dan gunakan filter `standby`, `online`, `offline`, `unknown`. | Filter tetap berfungsi; event yang tidak dicatat tidak muncul, dan tidak ada error 400 untuk nilai resmi. | |
| 26 | Buka Log CCTV saat kamera idle (standby nonaktif). | Halaman menampilkan event terakhir yang relevan tanpa deretan `standby` berulang; empty state tetap jelas bila belum ada event. | |

### 3.7 Pembersihan data uji

| No | Langkah | Hasil yang diharapkan | Hasil nyata |
|----|---------|-----------------------|-------------|
| 27 | Kembalikan `CCTV_LOG_STANDBY_EVENTS=false`, restart API. | Konfigurasi produksi kembali ke default. | |
| 28 | Hapus data uji yang ditandai (bila ada):<br>`DELETE FROM cctv_stream_logs WHERE message LIKE 'UJI DEDUP%';` | Tidak ada baris uji tersisa. | |

---

## 4. Kriteria Lulus

- [ ] Kamera idle tidak menghasilkan banjir log `standby` saat
  `CCTV_LOG_STANDBY_EVENTS=false`.
- [ ] Status `stream_state`/`stream_live` dan broadcast `CCTV_LIVE_STATUS_UPDATED`
  tetap diperbarui walau log standby dilewati.
- [ ] `CCTV_LOG_STANDBY_EVENTS=true`/`1` mencatat standby; nilai
  kosong/invalid/`false` tidak mencatat.
- [ ] Event tipe sama untuk CCTV yang sama dalam 60 detik tidak ditulis dua kali.
- [ ] Event tipe berbeda dalam 60 detik tetap ditulis.
- [ ] Event tipe sama setelah 60 detik tetap ditulis.
- [ ] Event `online`/`offline`/`unknown` yang penting tidak hilang.
- [ ] Query dedup yang gagal tidak memblokir pencatatan event (fail-open).
- [ ] Tidak ada duplikat `probe` + `verify` untuk satu transisi.
- [ ] Retensi 90 hari dan filter Log CCTV tetap berjalan.
- [ ] `go build`, `go vet`, `go test ./...`, dan `golangci-lint` PASS.
- [ ] CI Backend PASS.

---

## Catatan

- Probe hanya menulis event saat `stream_state`/`stream_live` berubah; kamera
  yang tetap idle tidak menghasilkan event baru meskipun `stream_checked_at`
  terus diperbarui.
- Verifier dapat menulis event `unknown` setelah satu kegagalan verifikasi.
  Event ini penting untuk membedakan "belum terverifikasi" dari "offline
  terkonfirmasi"; deduplikasi tetap berlaku bila tipe sama berulang dalam 60
  detik.
- Jendela deduplikasi 60 detik dipilih karena lebih pendek dari interval probe
  (2 menit) dan verifier (15 menit), tetapi cukup lebar untuk menyerap
  probe/verifier yang berjalan berdekatan.
- Deduplikasi bersifat best-effort. Pada deployment multi-instance API, dua
  instance dapat menulis event bersamaan; query diagnosis §5.1 dapat dipakai
  untuk memantau kejadian tersebut.
- Untuk mengaktifkan standby sementara saat investigasi, gunakan
  `CCTV_LOG_STANDBY_EVENTS=true`, lakukan reproduksi, lalu segera kembalikan
  ke `false`.
