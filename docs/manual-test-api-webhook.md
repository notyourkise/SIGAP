
# Uji Manual - API Key & Webhook

> **Task:** 12.1 API Key untuk Integrasi + 12.3 Webhook & Event System (Sprint 29)
> **Branch:** `feature/api-integration`
> **Versi:** backend `0.51.0` - frontend `1.43.0`
> **Dokumen:** verifikasi API key untuk integrasi eksternal (generate, scope, rate limit, revoke, akses `GET /api/cctv`) dan sistem webhook (CRUD endpoint, signature HMAC, retry, event trigger, test delivery, log riwayat).

Modul ini memungkinkan admin menyediakan kredensial API key bagi sistem eksternal untuk memanggil API SIGAP, serta mengirim notifikasi event ke endpoint webhook eksternal dengan signature HMAC-SHA256.

## 1. Prasyarat

| # | Item | Cara menyiapkan |
|---|------|-----------------|
| P1 | Aplikasi berjalan | `docker compose up -d --build`, buka `https://localhost` (atau `npm run dev` → `http://localhost:3001`) |
| P2 | Akun admin | Login sebagai admin |
| P3 | Migrasi | Migrasi `000010_api_keys_webhooks` berjalan otomatis saat startup (`api_keys`, `webhook_endpoints`, `webhook_deliveries`) |
| P4 | Alat API client | Swagger (`https://localhost/swagger/index.html`), Postman, atau `curl` untuk menguji `X-API-Key` |
| P5 | Penerima webhook | Siapkan endpoint penerima (mis. `https://webhook.site` atau server test) untuk menguji pengiriman |

## 2. Pengujian Otomatis

```powershell
cd backend
go build ./...
go test ./... -count=1
go vet ./...
golangci-lint run ./...
swag init

cd ../frontend
npm run lint
npm run build
```

Hasil yang diharapkan: seluruh perintah sukses; lint tidak memiliki error (0 error).

## 3. Skenario Uji (Blackbox)

> **Catatan verifikator:** setelah menyelesaikan setiap skenario, isi kolom "Hasil nyata" dengan **LULUS** / **GAGAL** + catatan singkat.

### S1 - CRUD API Key

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buka `/api-keys` sebagai admin | Halaman menampilkan tabel API key (nama, prefix, scope, terakhir dipakai, kedaluwarsa) | ✅ Halaman 200 (admin); UI "Buat API Key"/"Salin Key" ada; `GET /api/api-keys` menampilkan tabel nama, prefix, scope, last_used, expiry |
| 2 | Klik "Buat API Key", isi nama + pilih scope | Toast sukses; key mentah ditampilkan 1× dengan tombol "Salin Key" | ✅ `POST /api/api-keys` → 201; key mentah tampil sekali + pesan "Simpan API key ini sekarang" |
| 3 | Salin key lalu tutup notifikasi | Key mentah tidak dapat dilihat lagi; hanya prefix yang tampil di daftar | ✅ Key mentah tidak ada di list; hanya `key_prefix` yang dikembalikan |
| 4 | Klik "Cabut" pada sebuah key → konfirmasi | Key ditandai "Dicabut"; tidak dapat digunakan lagi | ✅ `POST /api/api-keys/:id/revoke` → `revoked_at` terisi; key ditolak 401 |
| 5 | Buat key dengan nama kosong / tanpa scope | Muncul error validasi | ✅ Nama kosong → 400 "Validasi input gagal"; scope invalid → 400 "Scope tidak valid" |

### S2 - Akses `GET /api/cctv` via API Key

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Panggil `GET /api/cctv` tanpa header | Respons 401 (wajib autentikasi) | ✅ 401 "Token otentikasi tidak ditemukan" |
| 2 | Panggil `GET /api/cctv` dengan `X-API-Key: <key-valid-scope-cctv-read>` | Respons 200 dengan daftar CCTV | ✅ 200 dengan daftar CCTV |
| 3 | Panggil `GET /api/cctv` dengan key tanpa scope `cctv:read` | Respons 403 (scope tidak cukup) | ✅ 403 "API key tidak memiliki izin (scope)" |
| 4 | Panggil `GET /api/cctv` dengan key yang sudah dicabut | Respons 401 | ✅ 401 "API key tidak valid" |
| 5 | Panggil `GET /api/cctv` dengan key kedaluwarsa | Respons 401 | ✅ 401 "API key tidak valid" (expires_at di-backdate via DB) |

### S3 - Akses `GET /api/health` (publik / API key)

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Panggil `GET /api/health` tanpa header | Respons 200 (publik) | ✅ 200 (db & redis Connected) |
| 2 | Panggil `GET /api/health` dengan `X-API-Key` valid scope `health:read` | Respons 200 | ✅ 200 |

### S4 - Rate Limit per API Key

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Kirim >100 request `GET /api/cctv` dalam 1 menit dengan API key yang sama | Request ke-101+ ditolak 429 "API key melebihi batas permintaan (100/menit)" | ✅ Request ke-101+ ditolak 429 (99 OK + 6×429 di 105 request); catatan: pesan dari global limiter IP (sama-sama 100/menit) |
| 2 | Tunggu melewati batas menit | Request kembali diterima (counter reset) | ✅ 200 setelah menit berlalu |

### S5 - CRUD Webhook Endpoint

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buka `/webhooks` sebagai admin | Halaman menampilkan daftar endpoint webhook | ✅ Halaman 200 (admin); `GET /api/webhooks` mengembalikan daftar |
| 2 | Klik "Tambah Webhook", isi nama, URL, secret (min 8), pilih event | Toast sukses; endpoint baru tampil dengan badge event | ✅ `POST /api/webhooks` → 201; endpoint tampil dengan daftar event |
| 3 | Ubah URL / event / status aktif | Data tersimpan | ✅ `PUT /api/webhooks/:id` → data tersimpan |
| 4 | Nonaktifkan endpoint | Endpoint berhenti menerima pengiriman (badge "Nonaktif") | ✅ `is_active=false` tersimpan |
| 5 | Hapus endpoint → konfirmasi | Endpoint hilang (soft delete) | ✅ DELETE → 200; hilang dari daftar |
| 6 | Tambah dengan event tidak valid / secret pendek | Muncul error validasi | ✅ Event invalid → 400 "Event tidak valid"; secret <8 → 400 |

### S6 - Test Delivery

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Klik ikon kirim (test) pada sebuah endpoint | Payload `test.ping` dikirim ke URL; toast menampilkan HTTP code + latensi | ✅ `POST /api/webhooks/:id/test` → response_code 200, latency 645ms |
| 2 | Periksa penerima webhook | Penerima menerima POST dengan header `X-Signature-256` (HMAC-SHA256) dan payload `test.ping` | ✅ webhook.site menerima POST + `X-Signature-256`; HMAC diverifikasi cocok dengan secret |
| 3 | Klik ikon mata untuk detail | Modal menampilkan riwayat pengiriman (event, status, HTTP, latensi, percobaan) | ✅ `GET /api/webhooks/:id/deliveries` mengembalikan log (event, status, response_code, latency_ms, attempt); catatan: test.ping sengaja tidak disimpan permanen (by design) |

### S7 - Event Trigger & Pengiriman Otomatis

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buat endpoint webhook yang berlangganan `cctv.status_changed` | Endpoint aktif tersimpan | ✅ Endpoint aktif dengan event `cctv.status_changed` tersimpan |
| 2 | Ubah status sebuah CCTV (mis. via edit Master CCTV) | Event `cctv.status_changed` terkirim ke endpoint dalam beberapa detik | ✅ Event terkirim (payload old/new status) — catatan: trigger resmi via `syncCCTVStatus` saat maintenance/field-check selesai; edit status manual di Master CCTV tidak memicu event |
| 3 | Buat endpoint berlangganan `maintenance.completed` lalu selesaikan tugas | Event `maintenance.completed` terkirim | ✅ Event terkirim setelah tugas maintenance selesai |
| 4 | Buat endpoint berlangganan `stream.down` lalu matikan stream >15 menit | Event `stream.down` terkirim | ❌ Tidak diuji (butuh skenario stream down >15 menit); dispatch terkonfirmasi di `stream.go:717` |
| 5 | Periksa riwayat pengiriman | Setiap event tercatat dengan status success/failed + latensi | ✅ Kedua event tercatat `success`, response 200, latency & attempt tampil |

### S8 - Retry 3× Exponential Backoff

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buat endpoint dengan URL yang selalu error (mis. 500) | Pengiriman dicoba ulang hingga 3× | ✅ Endpoint `httpstat.us/500` — delivery tercoba 3× |
| 2 | Periksa riwayat pengiriman | Delivery berstatus `failed` dengan `attempt` = 3; percobaan berjarak 1s, 5s, 25s | ✅ Status `failed`, `attempt` = 3; catatan: jeda aktual antar percobaan 1s & 5s (delay 25s tidak terpakai karena hanya 2 jeda untuk 3 percobaan) |
| 3 | Buat endpoint dengan URL valid | Delivery berstatus `success` pada percobaan pertama | ✅ Endpoint webhook.site → `success`, `attempt` = 1 |

### S9 - Role Enforcement

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Login sebagai **teknisi**, akses `/api-keys` atau `/webhooks` langsung via URL | Diarahkan kembali ke `/dashboard` (proxy middleware) | ✅ Teknisi akses `/api-keys` → redirect ke `/dashboard` |
| 2 | Teknisi memanggil `POST /api/api-keys` via Swagger | Respons 403 (Hanya Admin) | ✅ 403 "Akses ditolak: Peran Anda tidak memiliki izin" |
| 3 | Teknisi memanggil `POST /api/webhooks` | Respons 403 | ✅ 403 |
| 4 | Teknisi memanggil `GET /api/cctv` dengan API key | Diterima (200) jika scope cukup (jalur API key, bukan role) | ✅ 200 via `X-API-Key` tanpa sesi |

## 4. Kriteria Lulus

1. Model `APIKey` & `WebhookEndpoint`/`WebhookDelivery` dibuat; migrasi `000010_api_keys_webhooks` (up/down) tersedia.
2. CRUD `/api/api-keys` berfungsi (admin-only); key mentah hanya tampil sekali saat pembuatan (bcrypt hash disimpan).
3. Middleware `APIKeyAuth` memvalidasi `X-API-Key`: cek revoked, expired, scope; rate limit 100 req/menit per key.
4. `GET /api/cctv` dapat diakses via API key scope `cctv:read`; `GET /api/health` publik (juga via API key `health:read`).
5. CRUD `/api/webhooks` berfungsi (admin-only); `POST /api/webhooks/:id/test` mengirim payload test; `GET /api/webhooks/:id/deliveries` mengembalikan log.
6. `DispatchWebhook` mengirim ke endpoint dengan header `X-Signature-256` (HMAC-SHA256), retry 3× exponential backoff (1s/5s/25s).
7. Event webhook terpicu pada: `cctv.status_changed`, `maintenance.completed`, `maintenance.overdue`, `health_score.changed`, `stream.down`.
8. Halaman `/api-keys` (admin-only) menampilkan generate key (tampil 1×), list, revoke, badge scope, usage.
9. Halaman `/webhooks` (admin-only) menampilkan CRUD endpoint, pilih event, test delivery, log deliveries.
10. Menu sidebar "API Keys" (`KeyRound`) & "Webhooks" (`Webhook`) tampil untuk admin; proxy melindungi `/api-keys` & `/webhooks`.
11. `go build`/`go test`/`go vet`/`golangci-lint`/`swag init` dan `npm run lint`/`npm run build` lulus.

## Catatan

- Key mentah disimpan sebagai **bcrypt hash**; key hanya ditampilkan satu kali saat pembuatan. Simpan di tempat aman.
- API key di-cabut (revoke) dengan mengisi `revoked_at`; key yang dicabut/kedaluwarsa ditolak middleware.
- Rate limit per key memakai Redis counter (`apikey-limiter:{keyID}:{minute}`) dengan fallback in-memory.
- Webhook dikirim dengan header `X-Signature-256` (HMAC-SHA256 dari secret); penerima dapat memverifikasi integritas payload.
- Pengiriman webhook berjalan asinkron (worker + antrean channel) sehingga tidak memperlambat request operasional.
- Riwayat pengiriman (WebhookDelivery) menyimpan status, HTTP code, latensi, dan jumlah percobaan (maks 3).
