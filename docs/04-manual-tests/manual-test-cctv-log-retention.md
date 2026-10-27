# Manual Test - Retensi dan Index Log Event CCTV

| Item | Detail |
|------|--------|
| **Issue** | #144 (P1 - Retensi dan index tabel `cctv_stream_logs`) |
| **Branch** | `feat/cctv-log-retention` |
| **Versi Backend** | 0.64.36 (setelah fix) |
| **Versi Frontend** | 1.55.37 (tidak berubah) |
| **Tanggal** | 2026-10-28 |

---

## 1. Prasyarat

- [ ] Stack berjalan: `db`, `redis`, `api`, `mediamtx`, `nginx`, `frontend`.
- [ ] Akses `psql` ke database SIGAP:
  `docker compose exec db psql -U sigap -d sigap`.
- [ ] Minimal satu CCTV aktif (`cctvs.deleted_at IS NULL`) untuk referensi
  `cctv_id`.
- [ ] Environment `CCTV_STREAM_LOG_RETENTION_DAYS` dapat diubah pada
  `docker-compose.yml`/`.env`, lalu API dapat direstart.
- [ ] Log container API dapat dilihat: `docker compose logs -f api`.
- [ ] Catat jumlah baris awal:
  ```sql
  SELECT count(*) AS total,
         min(created_at) AS tertua,
         max(created_at) AS terbaru
  FROM cctv_stream_logs;
  ```

> **Penting untuk audit:** cleanup menghapus permanen. Bila log lama masih
> dibutuhkan untuk audit, arsipkan lebih dulu:
>
> ```sql
> \copy (SELECT * FROM cctv_stream_logs WHERE created_at < NOW() - INTERVAL '90 days' ORDER BY created_at) TO '/tmp/cctv_stream_logs_archive.csv' CSV HEADER
> ```
>
> Salin file keluar container dengan
> `docker compose cp db:/tmp/cctv_stream_logs_archive.csv ./cctv_stream_logs_archive.csv`,
> lalu simpan sesuai kebijakan retensi arsip organisasi.

---

## 2. Pengujian Otomatis

Dijalankan dari `backend/`:

```bash
gofmt -l config/cctv_log.go config/cctv_log_test.go config/migrate_test.go \
  handlers/cctv_stream_log_cleanup.go handlers/cctv_stream_log_cleanup_test.go main.go
go build ./...
go vet ./...
go test ./config/ ./handlers/ -run 'TestCCTVStreamLog|TestCleanupCCTVStreamLogs' -count=1 -v
go test ./... -count=1
golangci-lint run ./...
```

**Kriteria lulus:** tidak ada file yang diubah yang perlu diformat, `go build`,
`go vet`, `go test`, dan `golangci-lint` PASS (0 issues). Test terarah harus
memuat:

- `TestCCTVStreamLogRetentionDays_Default` → default 90 hari.
- `TestCCTVStreamLogRetentionDays_Configured` → override `30` dan `" 45 "`.
- `TestCCTVStreamLogRetentionDays_InvalidFallsBackToDefault` → `0`, negatif,
  bukan angka, desimal, dan overflow jatuh ke 90.
- `TestCleanupCCTVStreamLogs_DeletesInBatches` → 1.000 + 1.000 + 250 = 2.250.
- `TestCleanupCCTVStreamLogs_IsIdempotent` → pemanggilan kedua menghapus 0.
- `TestCleanupCCTVStreamLogs_ReturnsDatabaseError` → error diteruskan.
- `TestCCTVStreamLogRetentionCutoff` → cutoff 90 hari, log 89 hari aman,
  log 91 hari terhapus, retensi `<= 0` memakai default.
- `TestCCTVStreamLogIndexMigration` → migration up/down berisi index komposit.

Verifikasi migration secara penuh dijalankan CI Backend dengan PostgreSQL
(`up → down → up`). Untuk lokal (bila Docker tersedia):

```bash
cd backend
go run cmd/migrate/main.go version
go run cmd/migrate/main.go up
go run cmd/migrate/main.go down
go run cmd/migrate/main.go up
go run cmd/migrate/main.go version
```

**Kriteria lulus:** versi akhir `29`, tidak dirty, dan tidak ada error.

---

## 3. Skenario Uji

### 3.1 Konfigurasi dan cleanup dasar

| No | Langkah | Hasil yang diharapkan | Hasil nyata |
|----|---------|-----------------------|-------------|
| 1 | Pastikan `CCTV_STREAM_LOG_RETENTION_DAYS` kosong/tidak diset, lalu restart API. Periksa log startup. | Log menampilkan `Cleanup log event CCTV diaktifkan (retensi: 90 hari, interval: 24 jam)`. | |
| 2 | Seed log lama dan log baru pada satu CCTV (sesuaikan `cctv_id`):<br>`INSERT INTO cctv_stream_logs (created_at, cctv_id, event_type, current_state, message)`<br>`SELECT NOW() - INTERVAL '100 days', id, 'offline', 'offline', 'UJI RETENSI LAMA' FROM cctvs WHERE deleted_at IS NULL LIMIT 1;`<br>`INSERT ... NOW() - INTERVAL '89 days' ... 'UJI RETENSI BARU';`<br>`INSERT ... NOW() ... 'UJI RETENSI HARI INI';` | Tiga baris uji tersimpan dengan `created_at` sesuai. | |
| 3 | Catat jumlah baris, lalu restart API (`docker compose restart api`) untuk memicu cleanup startup. | Log cleanup menampilkan jumlah baris yang dihapus. Setelah selesai:<br>`SELECT message FROM cctv_stream_logs WHERE message LIKE 'UJI RETENSI%';` hanya menyisakan `UJI RETENSI BARU` dan `UJI RETENSI HARI INI`. | |
| 4 | Hitung ulang seluruh tabel. | `SELECT count(*) FROM cctv_stream_logs WHERE created_at < NOW() - INTERVAL '90 days';` = **0**. Log baru (hari ini) tetap ada. | |
| 5 | Restart API sekali lagi tanpa menambah data. | Log menampilkan tidak ada log lebih tua dari 90 hari (atau jumlah dihapus 0); jumlah baris tidak berubah. Worker idempotent. | |

### 3.2 Batch dan beban database

| No | Langkah | Hasil yang diharapkan | Hasil nyata |
|----|---------|-----------------------|-------------|
| 6 | Seed 2.500 log berumur 100 hari dengan `generate_series`:<br>`INSERT INTO cctv_stream_logs (created_at, cctv_id, event_type, current_state, message)`<br>`SELECT NOW() - INTERVAL '100 days', (SELECT id FROM cctvs WHERE deleted_at IS NULL LIMIT 1), 'offline', 'offline', 'UJI BATCH ' || g FROM generate_series(1, 2500) g;` | 2.500 baris uji masuk. | |
| 7 | Restart API dan amati log cleanup serta `pg_stat_activity` selama proses. | Cleanup menghapus 2.500 baris dalam tiga batch (1.000 + 1.000 + 500) dengan jeda ±100 ms. Tidak ada transaksi/lock panjang yang menahan tabel; API tetap melayani request lain. | |
| 8 | Setelah cleanup, periksa sisa baris uji batch. | `SELECT count(*) FROM cctv_stream_logs WHERE message LIKE 'UJI BATCH %';` = **0**, dan log baru di luar data uji tidak terhapus. | |

### 3.3 Override retensi dan fallback konfigurasi

| No | Langkah | Hasil yang diharapkan | Hasil nyata |
|----|---------|-----------------------|-------------|
| 9 | Set `CCTV_STREAM_LOG_RETENTION_DAYS=7`, seed log berumur 8 hari dan 6 hari (masing-masing satu baris, message `UJI RETENSI 7H` dan `UJI RETENSI 6H`), lalu restart API. | Log startup menampilkan retensi 7 hari. Baris 8 hari terhapus; baris 6 hari tetap ada. | |
| 10 | Set `CCTV_STREAM_LOG_RETENTION_DAYS=0`, tambahkan satu log berumur 100 hari, lalu restart API. | Log startup tetap menampilkan retensi **90 hari** (fallback). Log 100 hari terhapus; tidak ada indikasi retensi 0 yang menghapus seluruh tabel. | |
| 11 | Set nilai tidak valid (mis. `abc`), restart API. | Retensi tetap 90 hari dan cleanup berjalan normal. | |
| 12 | Kembalikan `CCTV_STREAM_LOG_RETENTION_DAYS=90` dan restart API. | Log startup menampilkan 90 hari. | |

### 3.4 Audit log dan verifikasi index

| No | Langkah | Hasil yang diharapkan | Hasil nyata |
|----|---------|-----------------------|-------------|
| 13 | Sebelum cleanup berikutnya, jalankan query audit:<br>`SELECT date_trunc('day', created_at) AS hari, event_type, count(*) FROM cctv_stream_logs GROUP BY 1,2 ORDER BY 1 DESC LIMIT 30;` | Hasil hanya memuat tanggal dalam rentang retensi (tidak ada hari lebih tua dari 90 hari). | |
| 14 | Periksa batas data yang tersisa:<br>`SELECT min(created_at), max(created_at), count(*) FROM cctv_stream_logs;` | `min(created_at)` >= `NOW() - INTERVAL '90 days'` (dengan toleransi waktu eksekusi cleanup). Tabel tidak lagi tumbuh tanpa batas. | |
| 15 | Periksa index:<br>`SELECT indexname, indexdef FROM pg_indexes WHERE tablename = 'cctv_stream_logs' ORDER BY indexname;` | Terdapat `idx_cctv_stream_logs_cctv_created` pada `(cctv_id, created_at)` dan `idx_cctv_stream_logs_event_created` pada `(event_type, created_at)`. Index `idx_cctv_stream_logs_cctv_id` dan `idx_cctv_stream_logs_event_type` sudah tidak ada. Index `idx_cctv_stream_logs_created_at` tetap ada. | |
| 16 | Jalankan rencana query daftar per kamera:<br>`EXPLAIN (ANALYZE, BUFFERS) SELECT * FROM cctv_stream_logs WHERE cctv_id = '<UUID>' ORDER BY created_at DESC LIMIT 15;` | Rencana memakai `idx_cctv_stream_logs_cctv_created` (Index Scan / Bitmap Index Scan), bukan sequential scan + sort untuk tabel besar. | |
| 17 | Jalankan rencana query ringkasan per event:<br>`EXPLAIN (ANALYZE, BUFFERS) SELECT count(*) FROM cctv_stream_logs WHERE event_type = 'offline' AND created_at >= NOW() - INTERVAL '1 day';` | Rencana memakai `idx_cctv_stream_logs_event_created` atau index `created_at`, tidak memindai seluruh tabel. | |
| 18 | Uji rollback migration (opsional, di lingkungan non-production): jalankan `go run cmd/migrate/main.go down` satu langkah, lalu `up`. | Index kolom tunggal kembali setelah `down`, index komposit kembali setelah `up`, dan data log tidak hilang. | |

### 3.5 Pembersihan data uji

| No | Langkah | Hasil yang diharapkan | Hasil nyata |
|----|---------|-----------------------|-------------|
| 19 | Hapus sisa data uji:<br>`DELETE FROM cctv_stream_logs WHERE message LIKE 'UJI RETENSI%' OR message LIKE 'UJI BATCH %';` | Tidak ada baris uji tersisa. | |

---

## 4. Kriteria Lulus

- [ ] Log lebih tua dari `CCTV_STREAM_LOG_RETENTION_DAYS` terhapus; log baru tetap aman.
- [ ] Cleanup berjalan saat startup dan dijadwalkan setiap 24 jam.
- [ ] Penghapusan dilakukan per batch maksimal 1.000 baris dengan jeda antar-batch.
- [ ] Worker idempotent: restart berulang tidak menghapus data yang sama.
- [ ] Nilai retensi kosong/invalid/`<= 0` memakai default 90 hari dan tidak menghapus seluruh tabel.
- [ ] Migration membuat index komposit `(cctv_id, created_at)` dan `(event_type, created_at)`; rollback mengembalikan index kolom tunggal.
- [ ] Query daftar/ringkasan memakai index dan tidak melakukan sequential scan pada tabel besar.
- [ ] `min(created_at)` tabel berada dalam rentang retensi sehingga tidak ada pertumbuhan tanpa batas.
- [ ] `go build`, `go vet`, `go test ./...`, dan `golangci-lint` PASS.
- [ ] CI Backend (termasuk migration `up → down → up`) PASS.

---

## Catatan

- Cleanup memakai cutoff `NOW() - retensi` dan query:
  ```sql
  DELETE FROM cctv_stream_logs
  WHERE id IN (
      SELECT id FROM cctv_stream_logs
      WHERE created_at < $1
      ORDER BY created_at ASC
      LIMIT 1000
  );
  ```
  Index `created_at DESC` dari migration `000028` tetap dipertahankan untuk
  mempercepat pemilihan batch.
- Tidak ada endpoint admin untuk memicu cleanup manual. Untuk pengujian,
  restart API atau panggil fungsi cleanup dari test.
- Worker hanya berjalan di proses API. Bila API dijalankan multi-instance,
  setiap instance menjalankan cleanup; query `DELETE ... WHERE id IN (...)`
  tetap aman karena baris yang sudah terhapus tidak akan terpilih lagi.
- Log audit aplikasi (`audit_logs`) tidak terpengaruh oleh retensi ini.
- Untuk kebutuhan forensik, arsipkan log lama sebelum cleanup sesuai prosedur
  pada bagian prasyarat.
