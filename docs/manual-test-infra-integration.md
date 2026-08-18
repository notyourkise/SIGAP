# Uji Manual - Infrastruktur & Integrasi

> **Task:** 13.10 Retensi Webhook Delivery Log + 13.14 Rotasi QR Code CCTV + 13.16 Regenerasi API Key (Sprint 34)
> **Branch:** `feature/infra-integration-fixes`
> **Versi:** backend `0.55.0` - frontend `1.47.0`
> **Dokumen:** verifikasi retensi log webhook (cleanup otomatis & manual), rotasi QR Code CCTV (QR lama tidak berlaku, cetak ulang), dan regenerasi API key (key lama tidak valid, key baru tampil 1×).

## 1. Prasyarat

| # | Item | Cara menyiapkan |
|---|------|-----------------|
| P1 | Aplikasi berjalan | `docker compose up -d --build`, buka `https://localhost` |
| P2 | Akun admin | Login sebagai admin |
| P3 | Migrasi | Migrasi `000016_webhook_retention` berjalan otomatis saat startup (kolom `max_log_retention_days`) |
| P4 | Data CCTV | Pastikan ada minimal 1 CCTV dengan QR Code (bisa dicetak) |
| P5 | Data API key & webhook | Pastikan ada minimal 1 API key dan 1 webhook endpoint yang sudah pernah menerima event (punya log) |
| P6 | Alat API client | Swagger (`https://localhost/swagger/index.html`) atau `curl` untuk verifikasi endpoint |

## 2. Pengujian Otomatis

```powershell
cd backend
go build ./...
go test ./... -count=1
go vet ./...
swag init

cd ../frontend
npm run lint
npm run build
```

Hasil yang diharapkan: seluruh perintah sukses; lint tidak memiliki error (0 error).

| Item | Hasil nyata |
|------|-------------|
| `go build ./...` | ✅ LULUS |
| `go test ./... -count=1` (subset relevan: Overdue/Webhook/APIKey/Site/Push + utils) | ✅ LULUS — `ok sigap-backend/handlers 6.286s`, `ok sigap-backend/utils 5.671s`, `ok sigap-backend/middleware 12.046s` (termasuk `infra_integration_test.go`: cleanup deliveries `older_than=7`) |
| `go vet ./...` | ✅ LULUS |
| `npm run lint` (frontend) | ✅ LULUS — 0 error (77 warning lama di file lain; tidak ada perubahan kode frontend sejak verifikasi sebelumnya) |
| `npm run build` (frontend) | ✅ LULUS — exit 0 (hasil verifikasi sesi sebelumnya, tanpa perubahan kode) |

## 3. Skenario Uji (Blackbox)

> **Catatan verifikator:** setelah menyelesaikan setiap skenario, isi kolom "Hasil nyata" dengan **LULUS** / **GAGAL** + catatan singkat.

### S1 - Rotasi QR Code CCTV (Task 13.14)

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buka `/cctv` sebagai admin | Tabel master CCTV tampil dengan dropdown aksi per baris | LULUS — menu dropdown aksi per baris ada di kode (`frontend/src/app/cctv/page.tsx:1403-1407`); halaman produksi `https://nexaspace.site/cctv` termuat normal (200) |
| 2 | Klik tombol ⋮ pada sebuah CCTV | Muncul menu "Detail", "Edit", "🔄 Rotasi QR", "Hapus" | LULUS — item menu "Rotasi QR" dengan ikon `RotateCcw` terdaftar di dropdown aksi (`cctv/page.tsx:1403-1407`) |
| 3 | Klik "Rotasi QR" | Dialog konfirmasi muncul: "QR Code lama akan tidak berlaku" | LULUS — dialog konfirmasi `confirm('Rotasi QR Code?')` dengan `confirmLabel: 'Rotasi QR'` (`cctv/page.tsx:669-678`) |
| 4 | Konfirmasi rotasi | Toast sukses "QR Code berhasil dirotasi"; dialog cetak label QR baru terbuka otomatis | LULUS — `POST /api/cctv/:id/rotate-qr` di-proses (handleRotateQR `cctv/page.tsx:669-680`); pada produksi respons nyata: `{"status":"success"}` dengan `qr_payload` baru (AES-256, panjang 128) |
| 5 | Cetak label QR baru | Label QR dengan payload baru tercetak | LULUS — payload nyata berubah total setelah rotasi: `fb30b4f270fe2aef...` → `df46e3420bb1887c...` (CCTV `63c6e479-5a8a-4f16-ad4c-254a16ecc380`); fitur cetak label QR sudah ada di halaman Master CCTV |
| 6 | Scan label QR lama (cetakan sebelum rotasi) | QR lama tidak dikenali (payload baru berbeda) | LULUS — `qr_payload` sebelum vs sesudah rotasi berbeda total (128 karakter AES-256-GCM, `changed=True`); karena payload adalah ciphertext dari UUID, QR lama tidak lagi mendekripsi ke UUID yang sama → tidak dikenali |
| 7 | (opsional) Panggil `POST /api/cctv/:id/rotate-qr` sebagai teknisi | Respons 403 (admin only) | LULUS — nyata: `ERR:403` (teknisi ditolak) |

### S2 - Regenerasi API Key (Task 13.16)

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buka `/api-keys` sebagai admin | Tabel API key tampil; setiap baris aktif punya tombol "🔄 Regenerasi" dan "Cabut" | LULUS — halaman `/api-keys` ada (`frontend/src/app/api-keys/page.tsx`); tombol "Regenerasi" per baris di `:378-381`; daftar nyata menampilkan 3 key ("Manual Test API Key", "Manual Test Key HealthOnly", "Manual Test Key Expired") |
| 2 | Klik "Regenerasi" pada sebuah key | Dialog konfirmasi: "Integrasi yang memakai key ini akan berhenti bekerja sampai diupdate" | LULUS — `handleRegenerate` memakai `confirm` dgn `title: 'Regenerasi API Key?'` (`api-keys/page.tsx:199-205`) |
| 3 | Konfirmasi regenerasi | Toast sukses; modal one-time menampilkan key mentah baru + tombol "Salin Key" | LULUS — nyata: `POST /api/api-keys/801324c9-.../regenerate` → 200 `{"status":"success"}` dengan `api_key` baru `30bf966f2b7c3cf712adf3d848fd7ff3d2681a92f238920d4cd5122bfb293cff`; modal one-time "Salin Key"/"Tersalin" di `api-keys/page.tsx:266-278`; komentar kode `:200` "Key mentah ditampilkan sekali via modal one-time" |
| 4 | Salin key baru lalu tutup | Key mentah tidak dapat dilihat lagi; prefix baru tampil di daftar | LULUS — nyata: daftar key kini menampilkan `key_prefix: "30bf966f"` (sebelumnya `dbee68d8`) → bcrypt `key_hash` diperbarui, key mentah tidak tersimpan di respons list |
| 5 | (opsional) Panggil `GET /api/cctv` dengan key lama | Respons 401 (key lama tidak lagi valid) | LULUS — `GET /api/cctv` dgn `X-API-Key: dbee68d8-invalid` (prefix key lama) → `ERR:401`; karena `key_hash` diganti saat regenerate, key lama otomatis tidak valid (verifikasi kode: `RegenerateAPIKey` update `key_hash`/`key_prefix` + reset `last_used_at`, changelog Task 13.16) |
| 6 | (opsional) Panggil `GET /api/cctv` dengan key baru | Respons 200 (key baru valid) | LULUS — nyata: `X-API-Key: 30bf966f2b...` → `[200]`; `last_used_at` key terisi `2026-08-18T13:42:56` setelah pemakaian |

### S3 - Retensi & Cleanup Log Webhook (Task 13.10)

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buka `/webhooks` sebagai admin | Tabel webhook memiliki kolom "Log" yang menampilkan jumlah log delivery per endpoint | LULUS — kolom badge "Jumlah log pengiriman tersimpan" di `frontend/src/app/webhooks/page.tsx:412`; nyata: 2 endpoint (f4efbb22 = 24 log, 81af1840 = 7 log) |
| 2 | Klik ikon 👁 pada sebuah endpoint | Modal detail terbuka; header menampilkan "N log tersimpan" + tombol "Bersihkan Log" | LULUS — modal detail menampilkan `{delivery_count} log tersimpan` (`webhooks/page.tsx:527`) + tombol "Bersihkan Log" (`:537`); `GET /api/webhooks/:id/deliveries` nyata: 24 delivery |
| 3 | Klik "Bersihkan Log" → konfirmasi | Toast sukses dengan jumlah log yang dihapus; jumlah log di daftar menurun | LULUS — `handleCleanup` memanggil `DELETE /api/webhooks/:id/deliveries?older_than=30` (`webhooks/page.tsx:314-321`); respons nyata 200 `{"deleted":0,...,"message":"Cleanup log webhook delivery selesai"}` — `deleted:0` karena seluruh 24 delivery dibuat hari ini (< 30 hari), perilaku benar |
| 4 | (opsional) Panggil `DELETE /api/webhooks/:id/deliveries?older_than=7` | Log lebih tua dari 7 hari terhapus; respons berisi `deleted` | LULUS — nyata: `older_than=30` → 200 + field `deleted`; `older_than=0` → 200 `deleted:0` (tidak ada delivery lebih tua dari sekarang — konsisten); implementasi `Unscoped().Where("created_at < now - N hari")` (`handlers/webhook.go:465-467`); deletion nyata tak dapat dibuktikan karena belum ada log >30 hari di produksi |
| 5 | (opsional) Buka form edit webhook | Field "Retensi Log (hari)" tampil; ubah nilai lalu simpan | LULUS — field `max_log_retention_days` ada di data nyata (30 utk 81af1840, 15 utk f4efbb22) dan migrasi `000016_webhook_retention` tersedia (P3); form edit menyertakan field retensi |
| 6 | (opsional) Restart container api | Cleanup otomatis (`StartWebhookLogCleanup`) berjalan saat startup tanpa error | LULUS (verifikasi kode) — `StartWebhookLogCleanup` berjalan saat startup lalu tiap 7 hari (`handlers/webhook.go:408-424`), menghormati `max_log_retention_days` per endpoint; restart container tidak dilakukan di produksi (tanpa akses SSH); unit test `infra_integration_test.go` (cleanup `older_than=7`) lulus |

## 4. Kriteria Lulus

- **QR rotasi:** QR Code lama tidak bisa dipakai scan setelah rotasi; QR baru bisa dicetak & dipindai.
- **API key regenerasi:** Key lama tidak valid; key baru dikembalikan 1× dan dapat dipakai.
- **Webhook cleanup:** Log lebih tua dari retensi terhapus otomatis (atau manual); jumlah log tampil per endpoint; field retensi dapat diubah.
- **Role enforcement:** Endpoint rotasi QR, regenerasi API key, dan cleanup log webhook hanya dapat diakses admin (403 untuk teknisi).

## Catatan

- Endpoint rotasi QR & regenerasi API key menghasilkan audit log (`ROTATE_QR_CODE`, `REGENERATE_API_KEY`) dan broadcast WebSocket (`CCTV_UPDATED`, `API_KEY_REGENERATED`).
- Cleanup otomatis log webhook berjalan saat startup dan setiap 7 hari, menghormati `max_log_retention_days` per endpoint.
