# Uji Manual - Notifikasi Email SMTP

> **Kontrak uji formal — MT-05**
>
> Dokumen ini mendefinisikan prosedur; hasil sebelum **2026-08-20** adalah catatan historis/non-authoritative dan harus diuji ulang dengan execution record formal. Inspeksi kode, review unit test, atau checkmark lama bukan `PASS` black-box.
>
> **Skenario audit reproduktif:** Dengan SMTP sandbox dan fixture penerima, kirim digest, overdue, dan stream-down untuk preferensi opt-in/opt-out serta simulasi kegagalan SMTP.
>
> **Status suite:** `NOT RUN`

### Formal execution metadata

| Field | Value |
|---|---|
| Test ID / Audit ID | MT-05 |
| Capability owner | QA Integrasi + Backend/Infra |
| Status suite | NOT RUN |
| Build SHA | NOT RECORDED |
| Artifact digest/ID | NOT RECORDED |
| Tester | NOT RECORDED |
| Timestamp WITA | NOT RECORDED |
| Environment | NOT RECORDED |
| Fixture prefix/data | `E2E-MT05-` / NOT RECORDED |
| Cleanup result | NOT RUN |
| Expected | Pesan diterima penerima yang tepat, opt-out tidak menerima, template valid, dan kegagalan SMTP tidak menggagalkan operasi utama. |
| Actual | NOT RECORDED |
| Evidence path/link | NOT RECORDED |
| Mandatory flag | YES |
| Scenario status | NOT RUN |

### Formal scenario record

| ID | Mandatory | Expected | Actual | Evidence | Status |
|---|---|---|---|---|---|
| MT-05-S1 | YES | Pesan diterima penerima yang tepat, opt-out tidak menerima, template valid, dan kegagalan SMTP tidak menggagalkan operasi utama. | NOT RECORDED | NOT RECORDED | NOT RUN |

**Aturan kesimpulan suite:** sebelum eksekusi suite `NOT RUN`; setelah run dimulai, setiap mandatory scenario harus `PASS`. Satu mandatory scenario berstatus `FAIL`, `BLOCKED`, atau `NOT RUN` membuat kesimpulan suite `FAIL`.

> **Guard keselamatan wajib:** Untuk setiap SQL mutasi, `DELETE`/`TRUNCATE`, rotasi/regenerasi key atau QR, penghapusan, reset kredensial, atau mutasi credential, lanjut hanya bila environment bukan production dan target bukan `nexaspace.site`, fixture prefix `E2E-MT05-` serta exact targets sudah dicatat, backup/recovery terverifikasi untuk perubahan data/key state, dan cleanup/rollback didefinisikan serta dicatat setelah run. Jika salah satu syarat tidak tersedia, hentikan prosedur dan set scenario status ke `BLOCKED`; jangan jalankan pada production. Jangan menyalin credential, token, private key, RTSP password, atau payload production yang belum disanitasi.


> **Task:** 11.3 Notifikasi Email SMTP (Sprint 27)
> **Branch:** `feature/email-reports`
> **Versi:** backend `0.49.0` - frontend `1.41.0`
> **Dokumen:** verifikasi pengiriman email notifikasi (ringkasan harian, eskalasi overdue, stok rendah), preferensi per pengguna, dan fallback bila SMTP tidak dikonfigurasi.

Modul ini menambah notifikasi email berbasis SMTP sebagai saluran tambahan selain notifikasi in-app dan push. Pengguna dapat mematikan/menghidupkan notifikasi email di halaman profil.

## 1. Prasyarat

| # | Item | Cara menyiapkan |
|---|------|-----------------|
| P1 | Aplikasi berjalan | `docker compose up -d --build`, buka `https://localhost` (atau `npm run dev` → `http://localhost:3001`) |
| P2 | Akun admin & teknisi | Siapkan user berperan `admin` dan `teknisi` dengan alamat email valid (bisa menerima email) |
| P3 | SMTP server | Konfigurasi `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASSWORD`, `SMTP_FROM`, `SMTP_TLS` di `backend/.env.docker` (dan/atau `.env`) |
| P4 | Kotak masuk email | Pastikan email admin/teknisi dapat menerima email (cek spam bila perlu) |
| P5 | Data pemeliharaan | Siapkan maintenance log yang melewati jadwal (untuk uji email overdue) |
| P6 | Data suku cadang | Siapkan suku cadang dengan stok mendekati minimum_order (untuk uji email stok rendah) |

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

| Item | Observasi historis (non-authoritative) |
|------|-------------|
| `go build ./...` | NOT RUN |
| `go test ./... -count=1` | NOT RUN |
| `go vet ./...` | NOT RUN |
| `npm run lint` (frontend) | NOT RUN |
| `npm run build` (frontend) | NOT RUN |

> Catatan: `golangci-lint` & `swag init` tidak dieksekusi (tool golangci-lint tidak terpasang di lingkungan verifikasi; `swag init` hanya regenerasi dokumentasi — tidak ada anotasi yang diubah selama verifikasi).

## 3. Skenario Uji (Blackbox)

> **Catatan legacy:** tabel di bawah adalah observasi historis/non-authoritative; jangan gunakan sebagai status formal. Isi `Formal scenario record` di atas dengan expected, actual, evidence, dan status formal.

### S1 - SMTP Tidak Dikonfigurasi (Fallback)

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Jalankan backend tanpa `SMTP_HOST` | Log saat startup menampilkan peringatan "Email SMTP tidak dikonfigurasi — notifikasi email nonaktif" | NOT RUN (verifikasi kode) — tidak dapat diuji nyata di prod karena SMTP aktif (lihat S2) |
| 2 | Aplikasi tetap berjalan normal | Seluruh fitur (in-app + push) berfungsi tanpa error | NOT RUN |
| 3 | Cek log tidak ada panic | Tidak ada crash terkait email | NOT RUN |

### S2 - SMTP Terkonfigurasi

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Isi `SMTP_HOST` dkk, restart backend | Log startup menampilkan "📧 Notifikasi email SMTP aktif" | NOT RUN |
| 2 | Aplikasi berjalan normal | Tidak ada error koneksi SMTP pada startup (koneksi dibuat saat kirim) | NOT RUN |

### S3 - Preferensi Email di Profil

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Buka halaman `/profile` | Terdapat section "Notifikasi Email" dengan toggle dan alamat email terdaftar | NOT RUN |
| 2 | Toggle aktif → nonaktif | Toast sukses "Notifikasi email dinonaktifkan"; state tersimpan | NOT RUN |
| 3 | Muat ulang halaman | Toggle tetap nonaktif (tersimpan di database) | NOT RUN |
| 4 | Toggle kembali ke aktif | Toast sukses "Notifikasi email diaktifkan" | NOT RUN |
| 5 | Cek database `users.email_notifications` | Nilai berubah sesuai toggle | NOT RUN |

### S4 - Ringkasan Harian (06:00 WITA)

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Tunggu/simulasikan generator digest pada 06:00 WITA (atau panggil `GenerateDailyDigestAt`) | Digest dibuat dan email ringkasan harian dikirim | NOT RUN sebagian — generator 06:00 WITA terverifikasi (digest 15–18 Agu dibuat 1×/hari) & pengiriman email dijalankan async, tapi isi kotak masuk tidak dapat diverifikasi (tanpa akses email) |
| 2 | Cek email admin yang opt-in | Email berisi tabel ringkasan: tugas baru/selesai/gagal/overdue, pengecekan lapangan, perubahan status | NOT RUN |
| 3 | Periksa formatting email | Template responsif, nama admin benar, footer SIGAP ada | NOT RUN |
| 4 | Admin dengan `email_notifications=false` | Tidak menerima email ringkasan | NOT RUN |

### S5 - Eskalasi Email Overdue

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Buat maintenance log terjadwal di masa lalu (overdue) | Overdue checker mengeskalasi | NOT RUN sebagian — checker berjalan tiap jam (startup + cron); log overdue 18 Agu (scheduled pagi) ada di DB |
| 2 | Jalankan `ProcessOverdueMaintenance` (atau tunggu cron 1 jam) | Email eskalasi overdue dikirim ke admin | NOT RUN |
| 3 | Cek email admin | Email berisi CCTV, kode, tindakan, jadwal, status | NOT RUN |
| 4 | Bila log `ongoing` dengan teknisi | Email juga dikirim ke teknisi yang ditugaskan | NOT RUN |
| 5 | Jalankan checker ulang untuk log yang sama | Email tidak terkirim duplikat (flag `overdue_notified_at` sudah terisi) | NOT RUN |

### S6 - Email Stok Rendah (STOCK_LOW)

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Catat penggunaan suku cadang hingga stok < minimum_order | Event `STOCK_LOW` dipicu | NOT RUN |
| 2 | Cek email admin | Email berisi nama suku cadang, SKU, kategori, stok tersisa, minimum order | NOT RUN |
| 3 | Periksa fallback in-app/push | Notifikasi in-app & push juga tetap terkirim (bukan pengganti) | NOT RUN |

### S7 - Role & Keamanan Endpoint

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Teknisi memanggil `PUT /api/profile/email-preferences` | Diperbolehkan (200) — preferensi milik sendiri | NOT RUN |
| 2 | Teknisi memanggil endpoint dengan `user_id` milik orang lain | Tetap ter-scope ke user sendiri (token JWT) | NOT RUN |
| 3 | Kirim body tanpa `email_notifications` | Respons 400 | NOT RUN |

### S8 - Email CCTV Offline (Stream Down)

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Pastikan ada CCTV dengan `stream_enabled=true` yang aktif | Stream terdaftar di MediaMTX | NOT RUN — butuh CCTV aktif + stream terdaftar di MediaMTX (skenario live production) |
| 2 | Matikan stream/CCTV (mis. hentikan MediaMTX path atau matikan sumber NVR) | Monitor stream mendeteksi down | NOT RUN — dapat dicek via monitor live-status (`/api/streams/live-status`) |
| 3 | Tunggu hingga down > 15 menit (atau jalankan monitor) | Event `STREAM_DOWN` dipicu + penalti health_score -10 | NOT RUN — ambang `streamDownThreshold = 15 menit` terverifikasi di kode (butuh waktu down nyata > 15 menit) |
| 4 | Cek email seluruh akun admin | Email "CCTV Offline" terkirim ke semua admin yang opt-in | NOT RUN — trigger kirim email saat stream down terverifikasi di kode |
| 5 | Periksa isi email | Email berisi nama CCTV, kode, lokasi, jenis, IP kamera, waktu, skor kesehatan, dan tindakan yang disarankan | NOT RUN |
| 6 | Admin dengan `email_notifications=false` | Tidak menerima email CCTV offline | NOT RUN |
| 7 | Pulihkan stream | Event `STREAM_RESTORED` dipicu; tidak ada email baru (email hanya saat down) | NOT RUN — email hanya dipicu dari kondisi down; pemulihan tidak memanggil fungsi email (verifikasi kode) |

> Catatan: pastikan `SMTP_HOST` terisi (mis. `<SMTP_SANDBOX_HOST>` untuk email `<REDACTED_PROD_SENDER>`) agar email benar-benar terkirim. Bila SMTP nonaktif, email di-skip (no-op). — **Prod terisi** (S2.1), email benar-benar terkirim ke `<REDACTED_TEST_RECIPIENT_1>` & `<REDACTED_TEST_RECIPIENT_2>`; isi kotak masuk tidak diverifikasi (tanpa akses email).

## 4. Kriteria Penerimaan

1. Env var SMTP (`SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASSWORD`, `SMTP_FROM`, `SMTP_TLS`) terdokumentasi dan dibaca aplikasi.
2. `utils/email.go` menyediakan `SendEmail` dengan `net/smtp` + TLS; email nonaktif bila `SMTP_HOST` kosong.
3. Template HTML `daily_summary.html`, `overdue_escalation.html`, `stock_low.html` tersedia & responsif (anti XSS via `html/template`).
4. Model `User` memiliki `EmailNotifications` + migrasi `000007`.
5. `PUT /api/profile/email-preferences` memperbarui preferensi per pengguna.
6. Goroutine/trigger: ringkasan harian saat digest 06:00 WITA, email overdue saat checker, email stok rendah saat `STOCK_LOW`.
7. Email CCTV offline terkirim ke seluruh admin yang opt-in saat stream down > 15 menit (event `STREAM_DOWN`), dengan template `stream_down.html`.
8. `go build`/`go test`/`go vet`/`golangci-lint`/`swag init` dan `npm run lint`/`npm run build` memenuhi kriteria.

## Catatan

- **Hasil uji (18 Agu 2026, produksi `<PRODUCTION_TARGET>`):** S1–S7 **NOT RUN**; S8 **NOT RUN** (butuh skenario stream down nyata > 15 menit — trigger & template terverifikasi di kode). Pengujian otomatis 5/5 memenuhi kriteria (golangci-lint & swag init tidak dieksekusi — lihat tabel Bagian 2).
- SMTP **aktif di produksi** (`<SMTP_SANDBOX_HOST>`, `<REDACTED_PROD_SENDER>`), sehingga S1 (fallback SMTP kosong) hanya bisa diverifikasi lewat kode, dan isi email di kotak masuk tidak dapat diperiksa (tanpa akses email). Data preferensi user terverifikasi persisten di DB.
- Preferensi granular per tipe (digest/overdue/stock) tersedia di UI profil & API (Task 13.7) — master toggle off mematikan semua sub-preferensi.
- Email adalah saluran tambahan; notifikasi in-app + push tetap berjalan (tidak digantikan).
- Bila `SMTP_HOST` kosong, seluruh fungsi email di-skip (no-op) tanpa error.
- Kirim email dilakukan async (goroutine) agar tidak memperlambat request/checker.
- Gunakan akun email uji atau layanan SMTP testing (mis. Mailtrap) untuk verifikasi tanpa spam.
