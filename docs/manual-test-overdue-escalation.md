# Uji Manual - Eskalasi Overdue Otomatis

> **Task:** 5.5 Eskalasi Overdue Otomatis
> **Branch:** `feature/overdue-escalation`
> **Versi backend:** `0.31.0`
> **Dokumen:** verifikasi alur overdue, notifikasi admin/teknisi, idempotensi, dan reset saat reschedule.

Checker overdue berjalan sekali saat startup API lalu setiap 1 jam. Checker memilih
log berstatus `scheduled` atau `ongoing` yang `scheduled_date`-nya sudah lewat dan
`overdue_notified_at` masih `NULL`. Setelah notifikasi berhasil dibuat, timestamp
tersebut diisi sehingga checker tidak mengirim ulang. Reschedule ke masa depan
mengosongkan timestamp tersebut.

## 1. Prasyarat

| # | Item | Cara menyiapkan |
|---|------|-----------------|
| P1 | Aplikasi berjalan | `docker compose up -d --build`, lalu buka `https://localhost` |
| P2 | Akun admin | Login dengan akun admin yang tersedia |
| P3 | Akun teknisi | Siapkan satu user berperan `teknisi` untuk skenario log `ongoing` |
| P4 | CCTV aktif | Siapkan satu CCTV dengan nama dan kode yang mudah dikenali |
| P5 | Dua sesi browser | Sesi admin untuk memeriksa notifikasi admin; sesi teknisi untuk skenario `ongoing` |
| P6 | Akses SQL opsional | Untuk mempercepat uji, gunakan `psql` di container database lalu restart API |

Checker tidak perlu menunggu satu jam saat memakai cara percepatan berikut. Ganti
`<LOG_UUID>` dan nilai environment database sesuai deployment:

```powershell
docker compose exec db psql -U ${POSTGRES_USER} -d ${POSTGRES_DB} -c "UPDATE maintenance_logs SET scheduled_date = now() - interval '1 hour', overdue_notified_at = NULL WHERE id = '<LOG_UUID>';"
```

Setelah itu restart API agar checker berjalan pada startup:

```powershell
docker compose restart api
```

## 2. Pengujian Otomatis

```powershell
cd backend
go build ./...
go test ./... -count=1
go vet ./...
swag init
```

Hasil yang diharapkan: semua perintah sukses; test `overdue_checker_test.go`
mencakup notifikasi admin, notifikasi teknisi, skip flag, status terminal, dan
reset saat reschedule.

## 3. Skenario Uji

### S1 - Log `scheduled` overdue mengirim notifikasi ke semua admin

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Admin membuka halaman Pemeliharaan CCTV dan membuat log baru berstatus Terjadwal dengan `scheduled_date` sementara di masa depan | Log tersimpan dengan status `scheduled` dan `overdue_notified_at` kosong | |
| 2 | Ubah `scheduled_date` log ke masa lalu memakai cara percepatan P6, lalu restart API | Checker memilih log tersebut pada proses startup | |
| 3 | Buka lonceng/halaman Pemberitahuan pada setiap akun admin | Setiap admin menerima tipe `MAINTENANCE_OVERDUE` dengan pesan `Tugas pemeliharaan [nama CCTV] (ID: [kode]) sudah melewati jadwal [tanggal]` | |
| 4 | Periksa data log melalui halaman atau SQL | `overdue_notified_at` terisi waktu proses checker | |

### S2 - Log `ongoing` juga mengirim notifikasi ke teknisi

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buat log Terjadwal, lalu tugaskan ke user teknisi | Status berubah menjadi `ongoing` dan `technician_id` terisi | |
| 2 | Mundurkan `scheduled_date` ke masa lalu dan restart API | Checker memproses log overdue | |
| 3 | Periksa akun admin | Admin menerima notifikasi overdue dengan nama dan kode CCTV | |
| 4 | Periksa akun teknisi yang ditugaskan | Teknisi menerima `MAINTENANCE_OVERDUE` dengan pesan `Tugas pemeliharaan Anda untuk CCTV [nama] sudah melewati jadwal` | |

### S3 - Idempotensi: notifikasi tidak dikirim dua kali

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Setelah S1 atau S2, restart API sekali lagi pada hari yang sama | Checker berjalan kembali | |
| 2 | Periksa lonceng dan riwayat pemberitahuan | Tidak ada notifikasi overdue duplikat untuk log yang sama | |
| 3 | Periksa `maintenance_logs.overdue_notified_at` | Timestamp tetap sama dan tidak diganti oleh proses kedua | |

### S4 - Status terminal tidak dieskalasi

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Siapkan satu log berstatus `completed` dan satu log `failed` dengan `scheduled_date` masa lalu | Kedua log terlihat sebagai riwayat terminal | |
| 2 | Jalankan checker melalui restart API | Checker melewati kedua log | |
| 3 | Periksa notifikasi admin/teknisi | Tidak ada `MAINTENANCE_OVERDUE` untuk log `completed` atau `failed` | |

### S5 - Reschedule ke masa depan mereset flag

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Pilih log yang sudah memiliki `overdue_notified_at` dari S1 atau S2 | Flag overdue terisi | |
| 2 | Ubah `scheduled_date` melalui form Edit menjadi besok | Perubahan tersimpan; `overdue_notified_at` kembali `NULL` | |
| 3 | Restart API saat tanggal baru belum tiba | Checker tidak mengirim notifikasi karena log belum overdue | |

### S6 - Setelah kembali overdue, notifikasi dapat dikirim ulang

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Mundurkan lagi `scheduled_date` log yang telah di-reschedule ke masa lalu | Log menjadi overdue dan flag tetap `NULL` | |
| 2 | Restart API | Checker mengirim satu notifikasi overdue baru | |
| 3 | Periksa database | `overdue_notified_at` terisi timestamp baru setelah pengiriman | |

### S7 - Tugas yang belum jatuh tempo dilewati

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buat atau reschedule log `scheduled` ke besok | `scheduled_date` lebih besar dari waktu saat ini | |
| 2 | Restart API | Log tidak diproses checker | |
| 3 | Periksa notifikasi | Tidak ada `MAINTENANCE_OVERDUE` untuk log tersebut | |

### S8 - Tipe notifikasi tampil di antarmuka in-app

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Picu S1 atau S2 sampai notifikasi overdue dibuat | Event `NOTIFICATION_CREATED` diterima sesi browser yang sesuai | |
| 2 | Buka NotificationBell dan halaman Pemberitahuan | Item overdue tampil dengan label/ikon/warna overdue yang konsisten dengan tipe `MAINTENANCE_OVERDUE`; klik item mengarah ke `/maintenance` | |
| 3 | Tandai item sebagai sudah dibaca | Badge jumlah belum dibaca berkurang dan item tersimpan sebagai dibaca | |

## 4. Kriteria Lulus

1. Log `scheduled` overdue mengirim notifikasi ke seluruh admin.
2. Log `ongoing` overdue juga mengirim notifikasi ke teknisi yang ditugaskan.
3. Setiap log hanya menerima satu eskalasi sampai jadwalnya diubah.
4. `completed` dan `failed` tidak menerima eskalasi.
5. Reschedule ke masa depan mereset `overdue_notified_at` ke `NULL`.
6. Jika log kembali overdue setelah reschedule, satu notifikasi baru dapat dikirim.
7. Log yang belum jatuh tempo tidak diproses.
8. NotificationBell dan halaman Pemberitahuan menampilkan tipe `MAINTENANCE_OVERDUE` dengan benar.

## Catatan

- Checker menggunakan interval satu jam; restart API hanya shortcut untuk menjalankan
  siklus pertama tanpa menunggu ticker.
- Notifikasi dibuat per penerima dan memakai `ReferenceType = maintenance-log` serta
  `ReferenceID` UUID log agar navigasi detail tetap dapat dilakukan.
- Hapus atau pisahkan data uji setelah verifikasi agar tidak mengganggu dashboard overdue.
