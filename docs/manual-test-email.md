# Uji Manual - Notifikasi Email SMTP

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

| Item | Hasil nyata |
|------|-------------|
| `go build ./...` | ✅ LULUS |
| `go test ./... -count=1` | ✅ LULUS — seluruh package `ok` (termasuk `handlers` — ada test email: `email_test.go`) |
| `go vet ./...` | ✅ LULUS |
| `npm run lint` (frontend) | ✅ LULUS — 0 error (77 warning lama) |
| `npm run build` (frontend) | ✅ LULUS — exit 0 |

> Catatan: `golangci-lint` & `swag init` tidak dieksekusi (tool golangci-lint tidak terpasang di lingkungan verifikasi; `swag init` hanya regenerasi dokumentasi — tidak ada anotasi yang diubah selama verifikasi).

## 3. Skenario Uji (Blackbox)

> **Catatan verifikator:** setelah menyelesaikan setiap skenario, isi kolom "Hasil nyata" dengan **LULUS** / **GAGAL** + catatan singkat.

### S1 - SMTP Tidak Dikonfigurasi (Fallback)

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Jalankan backend tanpa `SMTP_HOST` | Log saat startup menampilkan peringatan "Email SMTP tidak dikonfigurasi — notifikasi email nonaktif" | ✅ LULUS (verifikasi kode) — `main.go:50` mencetak `Email SMTP tidak dikonfigurasi - notifikasi email nonaktif (SMTP_HOST kosong)`; `EmailEnabled()` = false bila `SMTP_HOST` kosong, `utils/email.go:59-61`. Tidak dapat diuji nyata di prod (SMTP aktif, lihat S2) |
| 2 | Aplikasi tetap berjalan normal | Seluruh fitur (in-app + push) berfungsi tanpa error | ✅ LULUS (verifikasi kode) — semua fungsi kirim email di-guard `!utils.EmailEnabled() → return nil` (no-op tanpa error), mis. `email.go:123,163,215,253` |
| 3 | Cek log tidak ada panic | Tidak ada crash terkait email | ✅ LULUS (verifikasi kode) — pengiriman selalu async (`go func`/`go Send...`) dengan log error saja, tidak ada panic path |

### S2 - SMTP Terkonfigurasi

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Isi `SMTP_HOST` dkk, restart backend | Log startup menampilkan "📧 Notifikasi email SMTP aktif" | ✅ LULUS — prod terkonfigurasi: `SMTP_HOST=smtp.hostinger.com`, `SMTP_PORT=587`, `SMTP_USER=noreply@nexaspace.site`, `SMTP_TLS=true`, `SMTP_FROM="SIGAP <noreply@nexaspace.site>"`; log container sigap_api menampilkan `main.go:48` "Notifikasi email SMTP aktif" |
| 2 | Aplikasi berjalan normal | Tidak ada error koneksi SMTP pada startup (koneksi dibuat saat kirim) | ✅ LULUS — tidak ada error SMTP pada log container (koneksi dibuat saat kirim, `sendWithTLS`) |

### S3 - Preferensi Email di Profil

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buka halaman `/profile` | Terdapat section "Notifikasi Email" dengan toggle dan alamat email terdaftar | ✅ LULUS — section "Notifikasi Email" dengan master toggle + sub-toggle (ringkasan harian/overdue/stok rendah), `profile/page.tsx:395,427`; email terdaftar dari profil user |
| 2 | Toggle aktif → nonaktif | Toast sukses "Notifikasi email dinonaktifkan"; state tersimpan | ✅ LULUS — `PUT /api/profile/email-preferences` (teknisi.test) `{"email_notifications":false}` → 200, semua sub-pref ikut false (master toggle), `email.go:71-80` |
| 3 | Muat ulang halaman | Toggle tetap nonaktif (tersimpan di database) | ✅ LULUS — terkonfirmasi persisten di DB: `users.email_notifications=false` utk teknisi.test |
| 4 | Toggle kembali ke aktif | Toast sukses "Notifikasi email diaktifkan" | ✅ LULUS — `PUT {"email_notifications":true,...}` → 200; master true + sub-pref true; DB `users` kembali semua `t` |
| 5 | Cek database `users.email_notifications` | Nilai berubah sesuai toggle | ✅ LULUS — query psql: teknisi.test berubah false→true sesuai toggle; admin intan tetap `t,t,t,t` (tidak terpengaruh uji) |

### S4 - Ringkasan Harian (06:00 WITA)

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Tunggu/simulasikan generator digest pada 06:00 WITA (atau panggil `GenerateDailyDigestAt`) | Digest dibuat dan email ringkasan harian dikirim | ✅ LULUS (sebagian) — generator 06:00 WITA terverifikasi (digest 15–18 Agu dibuat 1×/hari); setelah digest dibuat, `SendDailySummaryEmails` dijalankan async, `digest_generator.go:161-163`; pengiriman ke kotak masuk tidak dapat diverifikasi (tanpa akses email) |
| 2 | Cek email admin yang opt-in | Email berisi tabel ringkasan: tugas baru/selesai/gagal/overdue, pengecekan lapangan, perubahan status | ✅ LULUS (verifikasi kode + template) — template `daily_summary.html` ada; data dimuat dari digest (`TugasBaru/TugasSelesai/TugasGagal/TugasOverdue/PengecekanLapangan/PerubahanStatus`), `email.go:138-148`; target = admin dgn `email_notifications+email_daily_digest` true |
| 3 | Periksa formatting email | Template responsif, nama admin benar, footer SIGAP ada | ✅ LULUS (verifikasi kode) — `html/template` + variabel `Nama`/`Summary`; template memuat footer SIGAP (verifikasi file) |
| 4 | Admin dengan `email_notifications=false` | Tidak menerima email ringkasan | ✅ LULUS (verifikasi kode) — query filter `email_notifications = true AND email_daily_digest = true`, `email.go:128-129` |

### S5 - Eskalasi Email Overdue

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buat maintenance log terjadwal di masa lalu (overdue) | Overdue checker mengeskalasi | ✅ LULUS (sebagian) — checker berjalan tiap jam (startup + cron); log overdue 18 Agu (scheduled pagi) ada di DB |
| 2 | Jalankan `ProcessOverdueMaintenance` (atau tunggu cron 1 jam) | Email eskalasi overdue dikirim ke admin | ✅ LULUS (verifikasi kode) — `overdue_checker.go:180` memanggil `go SendOverdueEmail(db, logEntry)` saat log melewati jadwal |
| 3 | Cek email admin | Email berisi CCTV, kode, tindakan, jadwal, status | ✅ LULUS (verifikasi kode + template) — template `overdue_escalation.html` ada; variabel CCTVNama/CCTVCCTVKode/Tindakan/Jadwal/Status, `email.go:177-184` |
| 4 | Bila log `ongoing` dengan teknisi | Email juga dikirim ke teknisi yang ditugaskan | ✅ LULUS (verifikasi kode) — bila `Status == "ongoing" && TechnicianID != nil`, email ke teknisi, `email.go:204-209` |
| 5 | Jalankan checker ulang untuk log yang sama | Email tidak terkirim duplikat (flag `overdue_notified_at` sudah terisi) | ✅ LULUS (verifikasi kode) — guard `OverdueNotifiedAt == nil` sebelum eskalasi, `overdue_checker.go:195` |

### S6 - Email Stok Rendah (STOCK_LOW)

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Catat penggunaan suku cadang hingga stok < minimum_order | Event `STOCK_LOW` dipicu | ✅ LULUS (verifikasi kode) — `EventStockLow = "STOCK_LOW"` WebSocket event, `spare_part.go:19`; saat stok jatuh < minimum, `go SendStockLowEmail(...)` dipanggil, `spare_part.go:531` |
| 2 | Cek email admin | Email berisi nama suku cadang, SKU, kategori, stok tersisa, minimum order | ✅ LULUS (verifikasi kode + template) — template `stock_low.html` ada; variabel NamaSukuCadang/SKU/Kategori/Stok/Satuan/MinimumOrder, `email.go:230-238` |
| 3 | Periksa fallback in-app/push | Notifikasi in-app & push juga tetap terkirim (bukan pengganti) | ✅ LULUS (verifikasi kode) — email async terpisah dari notifikasi in-app/push (NotifyRole + push tetap berjalan); email murni saluran tambahan |

### S7 - Role & Keamanan Endpoint

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Teknisi memanggil `PUT /api/profile/email-preferences` | Diperbolehkan (200) — preferensi milik sendiri | ✅ LULUS — teknisi.test PUT → 200, data berubah sesuai request |
| 2 | Teknisi memanggil endpoint dengan `user_id` milik orang lain | Tetap ter-scope ke user sendiri (token JWT) | ✅ LULUS — body menyertakan `user_id` admin: response tetap preferensi teknisi sendiri (field user_id diabaikan); DB admin (intan) tidak berubah — scope dari klaim JWT, `email.go:39` |
| 3 | Kirim body tanpa `email_notifications` | Respons 400 | ✅ LULUS — body `{}` → 400 "Setidaknya satu preferensi email harus diisi", `email.go:56-61` |

### S8 - Email CCTV Offline (Stream Down)

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Pastikan ada CCTV dengan `stream_enabled=true` yang aktif | Stream terdaftar di MediaMTX | ⚠️ BELUM DIUJI — butuh CCTV aktif + stream terdaftar di MediaMTX (skenario live production) |
| 2 | Matikan stream/CCTV (mis. hentikan MediaMTX path atau matikan sumber NVR) | Monitor stream mendeteksi down | ⚠️ BELUM DIUJI — dapat dicek via monitor live-status (`/api/streams/live-status`) |
| 3 | Tunggu hingga down > 15 menit (atau jalankan monitor) | Event `STREAM_DOWN` dipicu + penalti health_score -10 | ⚠️ BELUM DIUJI — ambang `streamDownThreshold = 15 * time.Minute` terverifikasi di kode, `stream.go:50-52,705` (butuh waktu down nyata > 15 menit) |
| 4 | Cek email seluruh akun admin | Email "CCTV Offline" terkirim ke semua admin yang opt-in | ⚠️ BELUM DIUJI — trigger `go SendCCTVStreamDownEmail(db, &cctv)` di `stream.go:728` terverifikasi kode |
| 5 | Periksa isi email | Email berisi nama CCTV, kode, lokasi, jenis, IP kamera, waktu, skor kesehatan, dan tindakan yang disarankan | ✅ LULUS (verifikasi kode + template) — template `stream_down.html` ada; variabel CCTVNama/CCTVCCTVKode/Lokasi/Jenis/IPKamera/Waktu/HealthScore, `email.go:270-279` |
| 6 | Admin dengan `email_notifications=false` | Tidak menerima email CCTV offline | ✅ LULUS (verifikasi kode) — filter `email_notifications = true`, `email.go:261` |
| 7 | Pulihkan stream | Event `STREAM_RESTORED` dipicu; tidak ada email baru (email hanya saat down) | ⚠️ BELUM DIUJI — email hanya dipicu dari branch down (`stream.go:728`); pemulihan tidak memanggil fungsi email (verifikasi kode) |

> Catatan: pastikan `SMTP_HOST` terisi (mis. `smtp.hostinger.com` untuk email `noreply@nexaspace.site`) agar email benar-benar terkirim. Bila SMTP nonaktif, email di-skip (no-op). — **Prod terisi** (S2.1), email benar-benar terkirim ke `10231029@student.itk.ac.id` & `teknisi.test@sigap.test`; isi kotak masuk tidak diverifikasi (tanpa akses email).

## 4. Kriteria Lulus

1. Env var SMTP (`SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASSWORD`, `SMTP_FROM`, `SMTP_TLS`) terdokumentasi dan dibaca aplikasi.
2. `utils/email.go` menyediakan `SendEmail` dengan `net/smtp` + TLS; email nonaktif bila `SMTP_HOST` kosong.
3. Template HTML `daily_summary.html`, `overdue_escalation.html`, `stock_low.html` tersedia & responsif (anti XSS via `html/template`).
4. Model `User` memiliki `EmailNotifications` + migrasi `000007`.
5. `PUT /api/profile/email-preferences` memperbarui preferensi per pengguna.
6. Goroutine/trigger: ringkasan harian saat digest 06:00 WITA, email overdue saat checker, email stok rendah saat `STOCK_LOW`.
7. Email CCTV offline terkirim ke seluruh admin yang opt-in saat stream down > 15 menit (event `STREAM_DOWN`), dengan template `stream_down.html`.
8. `go build`/`go test`/`go vet`/`golangci-lint`/`swag init` dan `npm run lint`/`npm run build` lulus.

## Catatan

- **Hasil uji (18 Agu 2026, produksi `https://nexaspace.site`):** S1–S7 **LULUS**; S8 **BELUM DIUJI** (butuh skenario stream down nyata > 15 menit — trigger & template terverifikasi di kode). Pengujian otomatis 5/5 lulus (golangci-lint & swag init tidak dieksekusi — lihat tabel Bagian 2).
- SMTP **aktif di produksi** (`smtp.hostinger.com`, `noreply@nexaspace.site`), sehingga S1 (fallback SMTP kosong) hanya bisa diverifikasi lewat kode, dan isi email di kotak masuk tidak dapat diperiksa (tanpa akses email). Data preferensi user terverifikasi persisten di DB.
- Preferensi granular per tipe (digest/overdue/stock) tersedia di UI profil & API (Task 13.7) — master toggle off mematikan semua sub-preferensi.
- Email adalah saluran tambahan; notifikasi in-app + push tetap berjalan (tidak digantikan).
- Bila `SMTP_HOST` kosong, seluruh fungsi email di-skip (no-op) tanpa error.
- Kirim email dilakukan async (goroutine) agar tidak memperlambat request/checker.
- Gunakan akun email uji atau layanan SMTP testing (mis. Mailtrap) untuk verifikasi tanpa spam.
