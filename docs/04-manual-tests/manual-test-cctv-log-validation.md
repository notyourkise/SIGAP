# Manual Test - Validasi dan Error Handling Log CCTV

| Item | Detail |
|------|--------|
| **Issue** | #145 (P1 - Validasi dan error handling endpoint Log CCTV) |
| **Branch** | `fix/cctv-log-validation` |
| **Versi Backend** | 0.64.37 (setelah fix) |
| **Versi Frontend** | 1.55.37 (tidak berubah) |
| **Tanggal** | 2026-10-28 |

---

## 1. Prasyarat

- [ ] Stack berjalan: `db`, `redis`, `api`, `mediamtx`, `nginx`, `frontend`.
- [ ] Tersedia akun **admin** dan token akses yang valid.
- [ ] Site aktif memiliki data `cctv_stream_logs` (minimal beberapa baris).
- [ ] `curl` dan `jq` tersedia di mesin penguji.
- [ ] Variabel shell disiapkan:

  ```bash
  export BASE_URL="http://localhost/api"
  export TOKEN="<TOKEN_AKSES>"
  export AUTH="Authorization: Bearer $TOKEN"
  ```

- [ ] Untuk skenario ringkasan, catat data mentah di database:

  ```sql
  SELECT event_type, count(*)
  FROM cctv_stream_logs
  WHERE created_at >= date_trunc('day', NOW() AT TIME ZONE 'Asia/Makassar') AT TIME ZONE 'Asia/Makassar'
  GROUP BY event_type;
  ```

---

## 2. Pengujian Otomatis

Dijalankan dari `backend/`:

```bash
gofmt -l handlers/cctv_stream_log.go handlers/cctv_stream_log_test.go \
  handlers/stream_probe.go handlers/stream_verify.go models/cctv_stream_log.go
go build ./...
go vet ./...
go test ./handlers/ -run 'TestGetCCTVStreamLogs' -count=1 -v
go test ./... -count=1
golangci-lint run ./...
```

**Kriteria lulus:** tidak ada file yang diubah perlu diformat, semua perintah
PASS, dan `golangci-lint` 0 issues.

Test yang wajib PASS:

- `TestGetCCTVStreamLogs_InvalidEventType` — `live`, `all`, `ONLINE`, typo, dan
  input berbahaya → **400** tanpa query database.
- `TestGetCCTVStreamLogs_ValidEventTypeFilter` — `event_type=unknown` → **200**
  serta filter diteruskan ke query daftar dan ringkasan.
- `TestGetCCTVStreamLogs_SummaryQueryError` — kegagalan query ringkasan
  online/offline → **500**, bukan ringkasan `0`.
- `TestGetCCTVStreamLogs_Success`, `..._TechnicianSiteScope`,
  `..._AdminSelectedSite`, `..._NonAdminWithoutSiteForbidden` — perilaku lama
  tetap berjalan.
- `TestGetCCTVStreamLogs_SummaryAppliesFilters` — ringkasan tetap mengikuti
  filter daftar.

---

## 3. Skenario Uji

### 3.1 Event type valid

| No | Langkah | Hasil yang diharapkan | Hasil nyata |
|----|---------|-----------------------|-------------|
| 1 | Minta `GET /cctv-logs?event_type=online`:<br>`curl -s -H "$AUTH" "$BASE_URL/cctv-logs?event_type=online" \| jq '{status, total: .data.pagination.total_rows, types: [.data.rows[].event_type] \| unique}'` | HTTP 200, `status="success"`, `types` hanya berisi `["online"]`, dan `total` konsisten dengan jumlah data online. | |
| 2 | Ulangi untuk `event_type=offline`, `standby`, dan `unknown`. | Masing-masing HTTP 200; seluruh baris pada `rows` memiliki `event_type` sesuai filter. Bila belum ada data untuk tipe tersebut, respons tetap 200 dengan `rows=[]` dan `total_rows=0`. | |
| 3 | Minta tanpa parameter `event_type`:<br>`curl -s -H "$AUTH" "$BASE_URL/cctv-logs"` | HTTP 200 dan daftar memuat semua tipe event (tanpa filter). | |
| 4 | Kirim nilai dengan spasi di awal/akhir:<br>`curl -s -H "$AUTH" "$BASE_URL/cctv-logs?event_type=%20online%20"` | HTTP 200; nilai di-trim dan diperlakukan sebagai `online`. | |

### 3.2 Event type invalid

| No | Langkah | Hasil yang diharapkan | Hasil nyata |
|----|---------|-----------------------|-------------|
| 5 | Kirim typo:<br>`curl -s -o /tmp/r.json -w "%{http_code}\n" -H "$AUTH" "$BASE_URL/cctv-logs?event_type=onlne"; cat /tmp/r.json` | Status **400**, `status="error"`, pesan memuat `event_type tidak valid. Nilai yang diizinkan: online, offline, standby, unknown`, dan tidak ada field `data`. | |
| 6 | Kirim `event_type=all`. | Status **400** dengan pesan yang sama. Nilai `all` tidak lagi diam-diam diabaikan; frontend mengirim tanpa parameter untuk "semua". | |
| 7 | Kirim `event_type=ONLINE` (huruf besar) dan `event_type=live`. | Keduanya **400**; validasi bersifat case-sensitive dan hanya menerima empat nilai resmi. | |
| 8 | Kirim `event_type=unknown;drop` (uji injection). | **400** sebelum query database; tidak ada error SQL di log API dan tidak ada perubahan data. | |

### 3.3 Konsistensi ringkasan

| No | Langkah | Hasil yang diharapkan | Hasil nyata |
|----|---------|-----------------------|-------------|
| 9 | Minta `event_type=online`, lalu bandingkan:<br>`jq '.data.summary'` | `summary.total_events` = `pagination.total_rows`; `today_online_events` mengikuti data hari ini; `today_offline_events` = 0 karena filter bertentangan dengan event offline. | |
| 10 | Minta tanpa filter, lalu bandingkan dengan hasil SQL pada prasyarat. | `today_online_events` dan `today_offline_events` sama dengan hitungan SQL untuk hari berjalan WITA (toleransi baris baru selama pengujian). `summary.timezone` bernilai `Asia/Makassar`. | |
| 11 | Terapkan `cctv_id` dan `search` bersama `event_type=online`. | Ringkasan dan daftar memakai filter yang sama; angka tidak melebihi daftar. | |

### 3.4 Kegagalan database

| No | Langkah | Hasil yang diharapkan | Hasil nyata |
|----|---------|-----------------------|-------------|
| 12 | Jalankan test otomatis `TestGetCCTVStreamLogs_SummaryQueryError`. | Kedua subtest (`online`, `offline`) PASS dan handler mengembalikan **500** `"Gagal menghitung ringkasan log CCTV"`, bukan HTTP 200 dengan angka 0. | |
| 13 | (Opsional) Hentikan database saat API melayani request, lalu minta `/cctv-logs`. | Request gagal dengan **500** dan pesan error yang jelas; tidak ada respons `success` dengan ringkasan palsu. Setelah database hidup, endpoint pulih tanpa restart API. | |

### 3.5 Konstanta dan produsen event

| No | Langkah | Hasil yang diharapkan | Hasil nyata |
|----|---------|-----------------------|-------------|
| 14 | Periksa penggunaan konstanta:<br>`grep -R "CCTVEvent" backend/handlers backend/models` | `stream_probe.go` dan `stream_verify.go` memakai `models.CCTVEvent*`, bukan literal `"online"`/`"offline"`/`"standby"`/`"unknown"`. | |
| 15 | Tunggu satu siklus probe/verifier, lalu periksa event baru. | Event baru hanya memiliki `event_type` di empat nilai resmi dan dapat difilter melalui endpoint. | |

---

## 4. Kriteria Lulus

- [ ] `event_type` kosong → tanpa filter; empat nilai resmi → HTTP 200.
- [ ] Nilai invalid (termasuk `all`, huruf besar, dan typo) → HTTP 400 dengan pesan jelas.
- [ ] Input invalid tidak menyentuh database dan tidak menghasilkan daftar kosong palsu.
- [ ] Kegagalan query ringkasan → HTTP 500, bukan angka `0`.
- [ ] Ringkasan tetap konsisten dengan daftar untuk filter yang sama.
- [ ] Produsen event memakai konstanta `models.CCTVEvent*`.
- [ ] `go build`, `go vet`, `go test ./...`, dan `golangci-lint` PASS.
- [ ] CI Backend PASS.

---

## Catatan

- Endpoint belum memiliki parameter rentang tanggal. Bila nanti ditambahkan,
  terapkan batas maksimum rentang (mis. 366 hari) dan validasi `start <= end`
  sebelum query.
- Validasi `event_type` bersifat case-sensitive dan tidak menerima alias `all`.
  Frontend versi 1.55.37 mengirim tanpa parameter untuk pilihan "Semua Event".
- Nilai `cctv_id` yang bukan UUID masih diabaikan (tanpa filter) sesuai perilaku
  lama; issue ini hanya mencakup validasi `event_type` dan error ringkasan.
- Kegagalan database pada langkah 13 dapat diuji dengan menghentikan container
  `db`; pastikan database dinyalakan kembali sebelum melanjutkan.
