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

## 3. Skenario Uji (Blackbox)

> **Catatan verifikator:** setelah menyelesaikan setiap skenario, isi kolom "Hasil nyata" dengan **LULUS** / **GAGAL** + catatan singkat.

### S1 - SMTP Tidak Dikonfigurasi (Fallback)

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Jalankan backend tanpa `SMTP_HOST` | Log saat startup menampilkan peringatan "Email SMTP tidak dikonfigurasi — notifikasi email nonaktif" | |
| 2 | Aplikasi tetap berjalan normal | Seluruh fitur (in-app + push) berfungsi tanpa error | |
| 3 | Cek log tidak ada panic | Tidak ada crash terkait email | |

### S2 - SMTP Terkonfigurasi

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Isi `SMTP_HOST` dkk, restart backend | Log startup menampilkan "📧 Notifikasi email SMTP aktif" | |
| 2 | Aplikasi berjalan normal | Tidak ada error koneksi SMTP pada startup (koneksi dibuat saat kirim) | |

### S3 - Preferensi Email di Profil

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buka halaman `/profile` | Terdapat section "Notifikasi Email" dengan toggle dan alamat email terdaftar | |
| 2 | Toggle aktif → nonaktif | Toast sukses "Notifikasi email dinonaktifkan"; state tersimpan | |
| 3 | Muat ulang halaman | Toggle tetap nonaktif (tersimpan di database) | |
| 4 | Toggle kembali ke aktif | Toast sukses "Notifikasi email diaktifkan" | |
| 5 | Cek database `users.email_notifications` | Nilai berubah sesuai toggle | |

### S4 - Ringkasan Harian (06:00 WITA)

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Tunggu/simulasikan generator digest pada 06:00 WITA (atau panggil `GenerateDailyDigestAt`) | Digest dibuat dan email ringkasan harian dikirim | |
| 2 | Cek email admin yang opt-in | Email berisi tabel ringkasan: tugas baru/selesai/gagal/overdue, pengecekan lapangan, perubahan status | |
| 3 | Periksa formatting email | Template responsif, nama admin benar, footer SIGAP ada | |
| 4 | Admin dengan `email_notifications=false` | Tidak menerima email ringkasan | |

### S5 - Eskalasi Email Overdue

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buat maintenance log terjadwal di masa lalu (overdue) | Overdue checker mengeskalasi | |
| 2 | Jalankan `ProcessOverdueMaintenance` (atau tunggu cron 1 jam) | Email eskalasi overdue dikirim ke admin | |
| 3 | Cek email admin | Email berisi CCTV, kode, tindakan, jadwal, status | |
| 4 | Bila log `ongoing` dengan teknisi | Email juga dikirim ke teknisi yang ditugaskan | |
| 5 | Jalankan checker ulang untuk log yang sama | Email tidak terkirim duplikat (flag `overdue_notified_at` sudah terisi) | |

### S6 - Email Stok Rendah (STOCK_LOW)

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Catat penggunaan suku cadang hingga stok < minimum_order | Event `STOCK_LOW` dipicu | |
| 2 | Cek email admin | Email berisi nama suku cadang, SKU, kategori, stok tersisa, minimum order | |
| 3 | Periksa fallback in-app/push | Notifikasi in-app & push juga tetap terkirim (bukan pengganti) | |

### S7 - Role & Keamanan Endpoint

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Teknisi memanggil `PUT /api/profile/email-preferences` | Diperbolehkan (200) — preferensi milik sendiri | |
| 2 | Teknisi memanggil endpoint dengan `user_id` milik orang lain | Tetap ter-scope ke user sendiri (token JWT) | |
| 3 | Kirim body tanpa `email_notifications` | Respons 400 | |

### S8 - Email CCTV Offline (Stream Down)

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Pastikan ada CCTV dengan `stream_enabled=true` yang aktif | Stream terdaftar di MediaMTX | |
| 2 | Matikan stream/CCTV (mis. hentikan MediaMTX path atau matikan sumber NVR) | Monitor stream mendeteksi down | |
| 3 | Tunggu hingga down > 15 menit (atau jalankan monitor) | Event `STREAM_DOWN` dipicu + penalti health_score -10 | |
| 4 | Cek email seluruh akun admin | Email "CCTV Offline" terkirim ke semua admin yang opt-in | |
| 5 | Periksa isi email | Email berisi nama CCTV, kode, lokasi, jenis, IP kamera, waktu, skor kesehatan, dan tindakan yang disarankan | |
| 6 | Admin dengan `email_notifications=false` | Tidak menerima email CCTV offline | |
| 7 | Pulihkan stream | Event `STREAM_RESTORED` dipicu; tidak ada email baru (email hanya saat down) | |

> Catatan: pastikan `SMTP_HOST` terisi (mis. `smtp.hostinger.com` untuk email `noreply@nexaspace.site`) agar email benar-benar terkirim. Bila SMTP nonaktif, email di-skip (no-op).

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

- Email adalah saluran tambahan; notifikasi in-app + push tetap berjalan (tidak digantikan).
- Bila `SMTP_HOST` kosong, seluruh fungsi email di-skip (no-op) tanpa error.
- Kirim email dilakukan async (goroutine) agar tidak memperlambat request/checker.
- Gunakan akun email uji atau layanan SMTP testing (mis. Mailtrap) untuk verifikasi tanpa spam.
