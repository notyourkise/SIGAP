# Semantik Log Event Stream CCTV

Dokumen ini menjelaskan kapan `cctv_stream_logs` diisi, perbedaan sumber
`probe` dan `verify`, kebijakan event `standby`, serta deduplikasi yang berlaku
sejak backend `0.64.38` (#146).

| Item | Detail |
|------|--------|
| **Issue** | #146 (P1 - Kebijakan event standby dan deduplikasi log) |
| **Berlaku sejak** | Backend `0.64.38` |
| **Tabel** | `cctv_stream_logs` |
| **Terkait** | #142 (timezone), #143 (UX filter), #144 (retensi), #145 (validasi) |

---

## 1. Ringkasan Keputusan Produk

1. `standby` **tetap menjadi status stream** yang sah, tetapi **tidak dicatat
   sebagai event secara default**. Alasannya, `standby` hanya berarti kamera
   on-demand sedang idle karena tidak ada penonton; mencatatnya setiap siklus
   probe akan membanjiri log tanpa informasi gangguan.
2. Pencatatan `standby` dapat diaktifkan secara eksplisit dengan
   `CCTV_LOG_STANDBY_EVENTS=true` untuk kebutuhan audit atau debugging.
3. Event dengan tipe yang sama untuk CCTV yang sama **diduplikasi dalam jendela
   60 detik**. Ini mencegah probe dan verifier menulis dua event untuk satu
   transisi yang sama.
4. Event `online`, `offline`, dan `unknown` tetap dicatat karena merepresentasikan
   perubahan kondisi yang penting. Kegagalan query deduplikasi bersifat
   **fail-open**: event tetap ditulis agar kejadian penting tidak hilang.

---

## 2. Sumber Event: `probe` vs `verify`

| Aspek | `probe` | `verify` |
|-------|---------|----------|
| **Trigger** | Periodik setiap **2 menit** (`RunRTSPProber`) | Periodik setiap **15 menit** (dapat diatur `CCTV_VERIFY_INTERVAL_MINUTES`, 5–30) dan manual lewat endpoint verifikasi |
| **Tujuan** | Menjaga status live/streamState tetap segar dengan biaya rendah | Membuktikan sumber benar-benar mengirim track (end-to-end), bukan sekadar path terdaftar |
| **Cara kerja** | Metro-E: ICMP ping (bila `CCTV_METRO_E_DIRECT_PROBE=true`) atau state MediaMTX pasif. GSM: **selalu pasif** dari daftar path MediaMTX — tidak menyentuh kamera/NVR | Membuka stream sementara ke MediaMTX (HLS/live) dengan timeout 8 detik, lalu memeriksa `available`/`ready`/`tracks` |
| **Kolom status** | `stream_state`, `stream_live`, `stream_checked_at`, `ping_latency_ms`, `stream_last_active_at` | `stream_verified_state`, `stream_verified_at`, `stream_verified_method`, `stream_verify_failures`, `stream_verify_error` |
| **Ambang offline** | Langsung dari hasil probe/path | Dua kegagalan berturut-turut (`streamVerifyFailureThreshold = 2`); kegagalan pertama → `unknown` |
| **Nilai `trigger_source`** | `probe` | `verify` |
| **Konsumsi data GSM** | Nol (hanya membaca state MediaMTX lokal) | Membuka stream saat verifikasi; dibatasi concurrency 1–4 dan interval minimal 5 menit |

Karena kedua sumber dapat mengamati transisi yang sama dalam waktu berdekatan,
tanpa deduplikasi tabel log dapat berisi dua baris `online` (satu `probe`, satu
`verify`) untuk satu kejadian. Deduplikasi 60 detik menutup celah ini.

### 2.1 Status stream dan artinya

| Status | Arti | Event yang dihasilkan |
|--------|------|-----------------------|
| `live` | Sumber benar-benar siap/merespons | `online` |
| `offline` | Sumber tidak merespons, atau path tidak siap tetapi ada penonton | `offline` |
| `idle` | Path on-demand terdaftar/instantiated tetapi belum ada penonton atau sumber sedang menutup | `standby` (hanya bila diaktifkan) |
| `unknown` | Status tidak dapat disimpulkan (mis. MediaMTX tidak dapat dihubungi) | `unknown` |
| `standby` (event) | Nama event untuk status `idle` | — |

Catatan: pada verifier, `unknown` juga dipakai sebagai hasil antara setelah satu
kegagalan verifikasi. Event `unknown` dari verifier tetap dicatat (bila berubah
dari state terverifikasi sebelumnya) karena menandakan kualitas pemantauan
sedang tidak pasti.

---

## 3. Kapan Event Ditulis

### 3.1 Probe

Event ditulis hanya ketika status berubah:

```go
stateChanged := streamState != cctv.StreamState
liveChanged := live != cctv.StreamLive
if liveChanged || stateChanged { ... }
```

Status dan broadcast `CCTV_LIVE_STATUS_UPDATED` tetap diperbarui walaupun
event-nya tidak dicatat (mis. `standby` saat kebijakan nonaktif). Dengan
demikian UI tetap menerima status terbaru tanpa membanjiri log.

### 3.2 Verifier

Event ditulis hanya ketika state terverifikasi berubah:

```go
stateChanged := result.State != result.CCTV.StreamVerifiedState
```

Hasil verifikasi tetap disimpan ke kolom `stream_verified_*` dan disiarkan
melalui WebSocket walaupun event log-nya dideduplikasi atau tidak dicatat.

### 3.3 Deduplikasi 60 detik

`shouldLogCCTVStreamEvent` mengambil **satu log terakhir** untuk CCTV tersebut
(`ORDER BY created_at DESC LIMIT 1`). Event dianggap duplikat bila:

- `event_type` log terakhir sama dengan event baru, dan
- waktu event baru masih di dalam **60 detik** setelah `created_at` log terakhir.

Contoh:

| Log terakhir | Event baru | Waktu | Ditulis? |
|--------------|------------|-------|----------|
| `online` | `online` | +30 detik | Tidak (duplikat) |
| `online` | `online` | +61 detik | Ya |
| `offline` | `online` | +30 detik | Ya (tipe berbeda) |
| tidak ada | `online` | — | Ya |
| query error | `offline` | — | Ya (fail-open) |

Deduplikasi bersifat **best-effort per instance**: bila API dijalankan
multi-instance, dua instance dapat lolos pemeriksaan secara bersamaan. Jendela
60 detik tetap menyerap hampir semua duplikasi probe/verifier karena keduanya
berjalan di proses yang sama dan verifier membatasi satu worker per kamera.

---

## 4. Konfigurasi

| Variabel | Default | Deskripsi |
|----------|---------|-----------|
| `CCTV_LOG_STANDBY_EVENTS` | `false` | Catat event `standby` saat kamera idle. Nilai `true`/`1`/`t` mengaktifkan; nilai kosong/tidak valid kembali ke `false` |
| `CCTV_STREAM_LOG_RETENTION_DAYS` | `90` | Retensi log dalam hari; cleanup berjalan saat startup lalu setiap 24 jam (batch 1.000) |
| `CCTV_VERIFY_INTERVAL_MINUTES` | `15` | Interval verifier; nilai di luar 5–30 diabaikan |
| `CCTV_VERIFY_CONCURRENCY` | `1` | Jumlah worker verifier; nilai di luar 1–4 diabaikan |
| `CCTV_METRO_E_DIRECT_PROBE` | `false` (Compose) | Aktifkan ICMP langsung ke kamera Metro-E; hanya bila host punya rute ke `10.20.105.x` |

Perubahan env memerlukan restart API. Nilai `CCTV_LOG_STANDBY_EVENTS`
dievaluasi setiap kali event akan dicatat, sehingga perilakunya konsisten
setelah restart.

---

## 5. Konsekuensi Operasional

- **Volume log**: transisi `live`/`offline` tetap tercatat; event rutin `idle`
  tidak lagi memenuhi tabel. Estimasi volume turun signifikan pada kamera yang
  sering dibuka-tutup penonton.
- **Audit gangguan**: `offline` yang berlangsung lama tidak menghasilkan event
  berulang setiap 2 menit karena status tidak berubah dan deduplikasi menahan
  duplikat. Untuk menghitung durasi gangguan, gunakan pasangan event
  `offline` → `online` terakhir.
- **Debugging standby**: aktifkan `CCTV_LOG_STANDBY_EVENTS=true` sementara,
  reproduksi perilaku, lalu kembalikan ke `false`. Jangan biarkan aktif permanen
  di produksi tanpa kebutuhan audit yang jelas.
- **Retensi**: log lama dihapus otomatis setelah 90 hari. Arsipkan lebih dulu
  bila diperlukan untuk audit jangka panjang
  (`docs/04-manual-tests/manual-test-cctv-log-retention.md`).

### 5.1 Query diagnosis

Log terakhir per kamera:

```sql
SELECT DISTINCT ON (cctv_id)
       cctv_id, event_type, previous_state, current_state,
       trigger_source, created_at
FROM cctv_stream_logs
ORDER BY cctv_id, created_at DESC;
```

Deteksi duplikat tipe sama dalam 60 detik:

```sql
SELECT a.cctv_id, a.event_type, a.created_at AS pertama, b.created_at AS kedua,
       EXTRACT(EPOCH FROM (b.created_at - a.created_at)) AS selisih_detik
FROM cctv_stream_logs a
JOIN cctv_stream_logs b
  ON b.cctv_id = a.cctv_id
 AND b.event_type = a.event_type
 AND b.created_at > a.created_at
 AND b.created_at < a.created_at + INTERVAL '60 seconds'
ORDER BY a.created_at DESC
LIMIT 50;
```

Hasil query di atas seharusnya kosong pada kondisi normal. Kemunculan baris
menandakan deduplikasi tidak berjalan (mis. multi-instance race atau kode lama).

Distribusi event per hari:

```sql
SELECT date_trunc('day', created_at AT TIME ZONE 'Asia/Makassar') AS hari,
       event_type, trigger_source, count(*)
FROM cctv_stream_logs
GROUP BY 1, 2, 3
ORDER BY 1 DESC, 2, 3;
```

---

## 6. Batasan dan Catatan Implementasi

- Deduplikasi menambah satu query `SELECT` setiap kali event akan ditulis,
  bukan pada setiap siklus probe. Biaya ini kecil dibandingkan penulisan log.
- Deduplikasi membandingkan **tipe event**, bukan pasangan
  `previous_state` → `current_state`. Transisi `unknown → offline` dan
  `live → offline` dalam 60 detik tetap dianggap duplikat `offline`; ini
  disengaja untuk mencegah banjir event saat status berfluktuasi cepat.
- Tidak ada unique index pada tabel; deduplikasi dilakukan di application layer
  agar kegagalan query tidak memblokir pencatatan event.
- Event `standby` yang dinonaktifkan tidak memengaruhi broadcast WebSocket
  `CCTV_STREAM_LOG_CREATED` (tidak dikirim) maupun
  `CCTV_LIVE_STATUS_UPDATED` (tetap dikirim).
- Perubahan pada kebijakan ini harus disertai pembaruan dokumen, test, dan
  changelog.

---

## 7. Referensi

- Implementasi: `backend/handlers/cctv_stream_log_events.go`
- Integrasi probe: `backend/handlers/stream_probe.go`
- Integrasi verifier: `backend/handlers/stream_verify.go`
- Test: `backend/handlers/cctv_stream_log_events_test.go`,
  `backend/config/cctv_log_test.go`
- Manual test: `docs/04-manual-tests/manual-test-cctv-log-dedup.md`
- Backlog induk: `docs/05-reports/TODO-PERBAIKAN-2026-10-27.md`
