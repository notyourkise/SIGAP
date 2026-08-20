
# Uji Manual - API Key & Webhook

> **Kontrak uji formal — MT-01**
>
> Dokumen ini mendefinisikan prosedur; hasil sebelum **2026-08-20** adalah catatan historis/non-authoritative dan harus diuji ulang dengan execution record formal. Inspeksi kode, review unit test, atau checkmark lama bukan `PASS` black-box.
>
> **Skenario audit reproduktif:** Dengan receiver internal/sandbox dan fixture tersanitasi berawalan `E2E-MT01-`, jalankan seluruh event webhook, verifikasi HMAC/retry, dan pastikan tidak ada payload produksi yang keluar.
>
> **Status suite:** `NOT RUN`

### Formal execution metadata

| Field | Value |
|---|---|
| Test ID / Audit ID | MT-01 |
| Capability owner | QA Integrasi + Backend |
| Status suite | NOT RUN |
| Build SHA | NOT RECORDED |
| Artifact digest/ID | NOT RECORDED |
| Tester | NOT RECORDED |
| Timestamp WITA | NOT RECORDED |
| Environment | NOT RECORDED |
| Fixture prefix/data | `E2E-MT01-` / NOT RECORDED |
| Cleanup result | NOT RUN |
| Expected | Semua event fixture diterima receiver internal, signature HMAC valid, retry 1/5/25 detik, dan payload tidak keluar dari environment non-production. |
| Actual | NOT RECORDED |
| Evidence path/link | NOT RECORDED |
| Mandatory flag | YES |
| Scenario status | NOT RUN |

### Formal scenario record

| ID | Mandatory | Expected | Actual | Evidence | Status |
|---|---|---|---|---|---|
| MT-01-S1 | YES | Semua event fixture diterima receiver internal, signature HMAC valid, retry 1/5/25 detik, dan payload tidak keluar dari environment non-production. | NOT RECORDED | NOT RECORDED | NOT RUN |

> **Cakupan aggregate Wave 0:** Baris formal aggregate di atas adalah **sole atomic suite-level acceptance gate** untuk baseline Wave 0. Skenario legacy terperinci di bawah ini tetap non-formal dan historical/non-authoritative sampai diperluas selama Wave 7.

**Aturan kesimpulan suite:** sebelum eksekusi suite `NOT RUN`; setelah run dimulai, setiap mandatory scenario harus `PASS`. Satu mandatory scenario berstatus `FAIL`, `BLOCKED`, atau `NOT RUN` membuat kesimpulan suite `FAIL`.

> **Guard keselamatan wajib:** Untuk setiap SQL mutasi, `DELETE`/`TRUNCATE`, rotasi/regenerasi key atau QR, penghapusan, reset kredensial, atau mutasi credential, lanjut hanya bila environment bukan production dan target bukan `nexaspace.site`, fixture prefix `E2E-MT01-` serta exact targets sudah dicatat, backup/recovery terverifikasi untuk perubahan data/key state, dan cleanup/rollback didefinisikan serta dicatat setelah run. Jika salah satu syarat tidak tersedia, hentikan prosedur dan set scenario status ke `BLOCKED`; jangan jalankan pada production. Jangan menyalin credential, token, private key, RTSP password, atau payload production yang belum disanitasi.

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

> **Catatan legacy:** tabel di bawah adalah observasi historis/non-authoritative; jangan gunakan sebagai status formal. Isi `Formal scenario record` di atas dengan expected, actual, evidence, dan status formal.

### S1 - CRUD API Key

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Buka `/api-keys` sebagai admin | Halaman menampilkan tabel API key (nama, prefix, scope, terakhir dipakai, kedaluwarsa) | NOT RUN |
| 2 | Klik "Buat API Key", isi nama + pilih scope | Toast sukses; key mentah ditampilkan 1× dengan tombol "Salin Key" | NOT RUN |
| 3 | Salin key lalu tutup notifikasi | Key mentah tidak dapat dilihat lagi; hanya prefix yang tampil di daftar | NOT RUN |
| 4 | Klik "Cabut" pada sebuah key → konfirmasi | Key ditandai "Dicabut"; tidak dapat digunakan lagi | NOT RUN |
| 5 | Buat key dengan nama kosong / tanpa scope | Muncul error validasi | NOT RUN |

### S2 - Akses `GET /api/cctv` via API Key

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Panggil `GET /api/cctv` tanpa header | Respons 401 (wajib autentikasi) | NOT RUN |
| 2 | Panggil `GET /api/cctv` dengan `X-API-Key: <key-valid-scope-cctv-read>` | Respons 200 dengan daftar CCTV | NOT RUN |
| 3 | Panggil `GET /api/cctv` dengan key tanpa scope `cctv:read` | Respons 403 (scope tidak cukup) | NOT RUN |
| 4 | Panggil `GET /api/cctv` dengan key yang sudah dicabut | Respons 401 | NOT RUN |
| 5 | Panggil `GET /api/cctv` dengan key kedaluwarsa | Respons 401 | NOT RUN |

### S3 - Akses `GET /api/health` (publik / API key)

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Panggil `GET /api/health` tanpa header | Respons 200 (publik) | NOT RUN |
| 2 | Panggil `GET /api/health` dengan `X-API-Key` valid scope `health:read` | Respons 200 | NOT RUN |

### S4 - Rate Limit per API Key

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Kirim >100 request `GET /api/cctv` dalam 1 menit dengan API key yang sama | Request ke-101+ ditolak 429 "API key melebihi batas permintaan (100/menit)" | NOT RUN |
| 2 | Tunggu melewati batas menit | Request kembali diterima (counter reset) | NOT RUN |

### S5 - CRUD Webhook Endpoint

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Buka `/webhooks` sebagai admin | Halaman menampilkan daftar endpoint webhook | NOT RUN |
| 2 | Klik "Tambah Webhook", isi nama, URL, secret (min 8), pilih event | Toast sukses; endpoint baru tampil dengan badge event | NOT RUN |
| 3 | Ubah URL / event / status aktif | Data tersimpan | NOT RUN |
| 4 | Nonaktifkan endpoint | Endpoint berhenti menerima pengiriman (badge "Nonaktif") | NOT RUN |
| 5 | Hapus endpoint → konfirmasi | Endpoint hilang (soft delete) | NOT RUN |
| 6 | Tambah dengan event tidak valid / secret pendek | Muncul error validasi | NOT RUN |

### S6 - Test Delivery

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Klik ikon kirim (test) pada sebuah endpoint | Payload `test.ping` dikirim ke URL; toast menampilkan HTTP code + latensi | NOT RUN |
| 2 | Periksa penerima webhook | Penerima menerima POST dengan header `X-Signature-256` (HMAC-SHA256) dan payload `test.ping` | NOT RUN |
| 3 | Klik ikon mata untuk detail | Modal menampilkan riwayat pengiriman (event, status, HTTP, latensi, percobaan) | NOT RUN |

### S7 - Event Trigger & Pengiriman Otomatis

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Buat endpoint webhook yang berlangganan `cctv.status_changed` | Endpoint aktif tersimpan | NOT RUN |
| 2 | Ubah status sebuah CCTV (mis. via edit Master CCTV) | Event `cctv.status_changed` terkirim ke endpoint dalam beberapa detik | NOT RUN |
| 3 | Buat endpoint berlangganan `maintenance.completed` lalu selesaikan tugas | Event `maintenance.completed` terkirim | NOT RUN |
| 4 | Buat endpoint berlangganan `stream.down` lalu matikan stream >15 menit | Event `stream.down` terkirim | NOT RUN — Tidak diuji — butuh skenario stream down >15 menit yang tidak tersedia (NVR tidak reachable) |
| 5 | Periksa riwayat pengiriman | Setiap event tercatat dengan status success/failed + latensi | NOT RUN |

### S8 - Retry 3× Exponential Backoff

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Buat endpoint dengan URL yang selalu error (mis. 500) | Pengiriman dicoba ulang hingga 3× | NOT RUN |
| 2 | Periksa riwayat pengiriman | Delivery berstatus `failed` dengan `attempt` = 3; percobaan berjarak 1s, 5s, 25s | NOT RUN |
| 3 | Buat endpoint dengan URL valid | Delivery berstatus `success` pada percobaan pertama | NOT RUN |

### S9 - Role Enforcement

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Login sebagai **teknisi**, akses `/api-keys` atau `/webhooks` langsung via URL | Diarahkan kembali ke `/dashboard` (proxy middleware) | NOT RUN |
| 2 | Teknisi memanggil `POST /api/api-keys` via Swagger | Respons 403 (Hanya Admin) | NOT RUN |
| 3 | Teknisi memanggil `POST /api/webhooks` | Respons 403 | NOT RUN |
| 4 | Teknisi memanggil `GET /api/cctv` dengan API key | Diterima (200) jika scope cukup (jalur API key, bukan role) | NOT RUN |

## 4. Kriteria Penerimaan

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
11. `go build`/`go test`/`go vet`/`golangci-lint`/`swag init` dan `npm run lint`/`npm run build` memenuhi kriteria.

## Catatan

- Key mentah disimpan sebagai **bcrypt hash**; key hanya ditampilkan satu kali saat pembuatan. Simpan di tempat aman.
- API key di-cabut (revoke) dengan mengisi `revoked_at`; key yang dicabut/kedaluwarsa ditolak middleware.
- Rate limit per key memakai Redis counter (`apikey-limiter:{keyID}:{minute}`) dengan fallback in-memory.
- Webhook dikirim dengan header `X-Signature-256` (HMAC-SHA256 dari secret); penerima dapat memverifikasi integritas payload.
- Pengiriman webhook berjalan asinkron (worker + antrean channel) sehingga tidak memperlambat request operasional.
- Riwayat pengiriman (WebhookDelivery) menyimpan status, HTTP code, latensi, dan jumlah percobaan (maks 3).

## Observasi Historis (Non-authoritative) — Catatan Verifikator (18 Agu 2026, produksi `<PRODUCTION_TARGET>`)

**Kesimpulan historis (non-authoritative): S1–S9 NOT RUN, kecuali S7 #4 (NOT RUN — tidak diuji — butuh stream down >15 menit; NVR tidak reachable).**

- Verifikasi nyata: CRUD API key (201 + key mentah 1×, list prefix-only, revoke → key ditolak 401, validasi 400), akses `GET /api/cctv` via `X-API-Key` (200 scope cukup, 403 scope kurang, 401 revoked/expired), `GET /api/health` publik (200), rate limit (request ke-101+ ditolak 429 lalu pulih setelah menit berlalu), CRUD webhook (201/200/soft-delete, validasi event & secret), test delivery (response 200, HMAC cocok, log latensi), event `cctv.status_changed` & `maintenance.completed` terkirim, retry 3× (endpoint error → `failed` attempt 3; endpoint valid → `success` attempt 1), dan role enforcement (teknisi redirect/403, tanpa sesi 401).
- Catatan perilaku: event `cctv.status_changed` terpicu resmi saat maintenance/field-check selesai; edit status manual di Master CCTV tidak memicu event. `test.ping` sengaja tidak disimpan permanen di log (by design). Jeda retry aktual 1s & 5s (delay 25s tidak terpakai karena hanya 2 jeda untuk 3 percobaan).
