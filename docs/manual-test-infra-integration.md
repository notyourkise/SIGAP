# Uji Manual - Infrastruktur & Integrasi

> **Kontrak uji formal — MT-11**
>
> Dokumen ini mendefinisikan prosedur; hasil sebelum **2026-08-20** adalah catatan historis/non-authoritative dan harus diuji ulang dengan execution record formal. Inspeksi kode, review unit test, atau checkmark lama bukan `PASS` black-box.
>
> **Skenario audit reproduktif:** Dengan fixture non-production, uji retensi webhook, rotasi QR, dan regenerasi API key memakai backup/recovery serta target cleanup yang terdaftar.
>
> **Status suite:** `NOT RUN`

### Formal execution metadata

| Field | Value |
|---|---|
| Test ID / Audit ID | MT-11 |
| Capability owner | QA Security/Infra + Infra/Backend |
| Status suite | NOT RUN |
| Build SHA | NOT RECORDED |
| Artifact digest/ID | NOT RECORDED |
| Tester | NOT RECORDED |
| Timestamp WITA | NOT RECORDED |
| Environment | NOT RECORDED |
| Fixture prefix/data | `E2E-MT11-` / NOT RECORDED |
| Cleanup result | NOT RUN |
| Expected | Credential/QR lama ditolak, pengganti berfungsi, cleanup hanya menyentuh fixture, audit tercatat, dan seluruh role negatif mendapat response yang benar. |
| Actual | NOT RECORDED |
| Evidence path/link | NOT RECORDED |
| Mandatory flag | YES |
| Scenario status | NOT RUN |

### Formal scenario record

| ID | Mandatory | Expected | Actual | Evidence | Status |
|---|---|---|---|---|---|
| MT-11-S1 | YES | Credential/QR lama ditolak, pengganti berfungsi, cleanup hanya menyentuh fixture, audit tercatat, dan seluruh role negatif mendapat response yang benar. | NOT RECORDED | NOT RECORDED | NOT RUN |

> **Cakupan aggregate Wave 0:** Baris formal aggregate di atas adalah **sole atomic suite-level acceptance gate** untuk baseline Wave 0. Skenario legacy terperinci di bawah ini tetap non-formal dan historical/non-authoritative sampai diperluas selama Wave 7.

**Aturan kesimpulan suite:** sebelum eksekusi suite `NOT RUN`; setelah run dimulai, setiap mandatory scenario harus `PASS`. Satu mandatory scenario berstatus `FAIL`, `BLOCKED`, atau `NOT RUN` membuat kesimpulan suite `FAIL`.

> **Guard keselamatan wajib:** Untuk setiap SQL mutasi, `DELETE`/`TRUNCATE`, rotasi/regenerasi key atau QR, penghapusan, reset kredensial, atau mutasi credential, lanjut hanya bila environment bukan production dan target bukan `nexaspace.site`, fixture prefix `E2E-MT11-` serta exact targets sudah dicatat, backup/recovery terverifikasi untuk perubahan data/key state, dan cleanup/rollback didefinisikan serta dicatat setelah run. Jika salah satu syarat tidak tersedia, hentikan prosedur dan set scenario status ke `BLOCKED`; jangan jalankan pada production. Jangan menyalin credential, token, private key, RTSP password, atau payload production yang belum disanitasi.


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
| P4 | Data CCTV | Buat minimal 1 fixture CCTV non-production berawalan `E2E-MT11-` dengan QR Code; catat exact ID, backup/recovery, dan target cleanup |
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

### Supporting evidence remediasi QR — 2026-08-20

Evidence berikut membuktikan perubahan source dan regression automation, bukan runtime black-box `PASS`:

- [x] Rotasi menghasilkan ciphertext baru yang mendekripsi ke ID CCTV yang sama.
- [x] Scanner menerima payload aktif dan menolak payload sebelumnya.
- [x] Cache gambar frontend memakai `id + qr_payload`, HTTP/fetch `no-store`, dan jalur QR bypass cache service worker; versi shell cache dinaikkan untuk membuang cache lama.
- [x] Targeted backend regression `TestRotateQRCode|TestScanCCTV|TestGetCCTVQRCode_DisablesHTTPCache` dan `npm run test:sw` lulus pada branch `codex/qr-lifecycle`.
- [ ] Jalankan MT-11-S1 dengan fixture non-production `E2E-MT11-`, bukti cetak/pindai, dan cleanup terverifikasi.

| Item | Observasi historis (non-authoritative) |
|------|-------------|
| `go build ./...` | NOT RUN |
| `go test ./... -count=1` (subset relevan: Overdue/Webhook/APIKey/Site/Push + utils) | NOT RUN |
| `go vet ./...` | NOT RUN |
| `npm run lint` (frontend) | NOT RUN |
| `npm run build` (frontend) | NOT RUN |
| `npm run test:sw` (cache policy QR/no-store) | NOT RUN |

## 3. Skenario Uji (Blackbox)

> **Catatan legacy:** tabel di bawah adalah observasi historis/non-authoritative; jangan gunakan sebagai status formal. Isi `Formal scenario record` di atas dengan expected, actual, evidence, dan status formal.

### S1 - Rotasi QR Code CCTV (Task 13.14)

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Buka `/cctv` sebagai admin | Tabel master CCTV tampil dengan dropdown aksi per baris | NOT RUN |
| 2 | Klik tombol ⋮ pada sebuah CCTV | Muncul menu "Detail", "Edit", "🔄 Rotasi QR", "Hapus" | NOT RUN |
| 3 | Klik "Rotasi QR" | Dialog konfirmasi muncul: "QR Code lama akan tidak berlaku" | NOT RUN |
| 4 | Konfirmasi rotasi | Toast sukses "QR Code berhasil dirotasi"; dialog cetak label QR baru terbuka otomatis | NOT RUN |
| 5 | Cetak label QR baru tanpa hard refresh | Label menampilkan QR dari payload baru, bukan gambar cache lama | NOT RUN |
| 6 | Scan label QR baru | CCTV fixture yang sama berhasil diidentifikasi | NOT RUN |
| 7 | Scan label QR lama (cetakan sebelum rotasi) | Respons menolak QR dengan pesan bahwa QR tidak aktif/sudah dirotasi | NOT RUN |
| 8 | (opsional) Panggil `POST /api/cctv/:id/rotate-qr` sebagai teknisi | Respons 403 (admin only) | NOT RUN |

### S2 - Regenerasi API Key (Task 13.16)

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Buka `/api-keys` sebagai admin | Tabel API key tampil; setiap baris aktif punya tombol "🔄 Regenerasi" dan "Cabut" | NOT RUN |
| 2 | Klik "Regenerasi" pada sebuah key | Dialog konfirmasi: "Integrasi yang memakai key ini akan berhenti bekerja sampai diupdate" | NOT RUN |
| 3 | Konfirmasi regenerasi | Toast sukses; modal one-time menampilkan key mentah baru + tombol "Salin Key" | NOT RUN |
| 4 | Salin key baru lalu tutup | Key mentah tidak dapat dilihat lagi; prefix baru tampil di daftar | NOT RUN |
| 5 | (opsional) Panggil `GET /api/cctv` dengan key lama | Respons 401 (key lama tidak lagi valid) | NOT RUN |
| 6 | (opsional) Panggil `GET /api/cctv` dengan key baru | Respons 200 (key baru valid) | NOT RUN |

### S3 - Retensi & Cleanup Log Webhook (Task 13.10)

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Buka `/webhooks` sebagai admin | Tabel webhook memiliki kolom "Log" yang menampilkan jumlah log delivery per endpoint | NOT RUN |
| 2 | Klik ikon 👁 pada sebuah endpoint | Modal detail terbuka; header menampilkan "N log tersimpan" + tombol "Bersihkan Log" | NOT RUN |
| 3 | Klik "Bersihkan Log" → konfirmasi | Toast sukses dengan jumlah log yang dihapus; jumlah log di daftar menurun | NOT RUN |
| 4 | (opsional) Panggil `DELETE /api/webhooks/:id/deliveries?older_than=7` | Log lebih tua dari 7 hari terhapus; respons berisi `deleted` | NOT RUN |
| 5 | (opsional) Buka form edit webhook | Field "Retensi Log (hari)" tampil; ubah nilai lalu simpan | NOT RUN |
| 6 | (opsional) Restart container api | Cleanup otomatis (`StartWebhookLogCleanup`) berjalan saat startup tanpa error | NOT RUN |

## 4. Kriteria Penerimaan

- **QR rotasi:** QR Code lama tidak bisa dipakai scan setelah rotasi; QR baru bisa dicetak & dipindai.
- **API key regenerasi:** Key lama tidak valid; key baru dikembalikan 1× dan dapat dipakai.
- **Webhook cleanup:** Log lebih tua dari retensi terhapus otomatis (atau manual); jumlah log tampil per endpoint; field retensi dapat diubah.
- **Role enforcement:** Endpoint rotasi QR, regenerasi API key, dan cleanup log webhook hanya dapat diakses admin (403 untuk teknisi).

## Catatan

- Endpoint rotasi QR & regenerasi API key menghasilkan audit log (`ROTATE_QR_CODE`, `REGENERATE_API_KEY`) dan broadcast WebSocket (`CCTV_UPDATED`, `API_KEY_REGENERATED`).
- Cleanup otomatis log webhook berjalan saat startup dan setiap 7 hari, menghormati `max_log_retention_days` per endpoint.
