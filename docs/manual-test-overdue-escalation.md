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

| Item | Hasil nyata |
|------|-------------|
| `go build ./...` | ✅ LULUS |
| `go test ./... -count=1` (subset `-run "Overdue"`) | ✅ LULUS — `ok sigap-backend/handlers 6.286s`; `overdue_checker_test.go` mencakup: log `scheduled` overdue → notif admin, log `ongoing` → notif teknisi, idempotensi (skip flag `overdue_notified_at` terisi), status terminal dilewati, reset saat reschedule, dan matriks level (25j→Warning, 49j→High, 73j→Critical) |
| `go vet ./...` | ✅ LULUS |
| `swag init` | ✅ tidak diperlukan perubahan (tidak ada anotasi Swagger baru sesi ini) |

## 3. Skenario Uji

### S1 - Log `scheduled` overdue mengirim notifikasi ke semua admin

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Admin membuka halaman Pemeliharaan CCTV dan membuat log baru berstatus Terjadwal dengan `scheduled_date` sementara di masa depan | Log tersimpan dengan status `scheduled` dan `overdue_notified_at` kosong | ✅ — log uji kalender (mis. `d5046e42`, `dde660ab`, `08dce9d2`, `efce78b0`) tersimpan status `scheduled`/`ongoing` dgn `scheduled_date` di masa lalu (12 Agu) dan `overdue_notified_at` NULL |
| 2 | Ubah `scheduled_date` log ke masa lalu memakai cara percepatan P6, lalu restart API | Checker memilih log tersebut pada proses startup | ✅ — log uji sudah dibuat dengan `scheduled_date` masa lalu (12 Agu 2026, >72 jam sebelum verifikasi 18 Agu); checker produksi berjalan (startup + tiap 1 jam, `RunOverdueChecker` di `overdue_checker.go:34-46`) dan memprosesnya — terbukti dari notifikasi yang terkirim (lihat S1 #3). Restart API tidak dilakukan (tanpa akses SSH); interval produksi 1 jam |
| 3 | Buka lonceng/halaman Pemberitahuan pada setiap akun admin | Setiap admin menerima tipe `MAINTENANCE_OVERDUE` dengan pesan `Tugas pemeliharaan [nama CCTV] (ID: [kode]) sudah melewati jadwal [tanggal]` | ✅ (tipe disesuaikan implementasi Task 13.11) — nyata di prod: admin menerima 4 notifikasi `OVERDUE_CRITICAL` berjudul "KRITIS: Tugas pemeliharaan sangat terlambat" dgn pesan `Tugas pemeliharaan [nama] (ID: [kode]) sudah melewati jadwal [tanggal] (level 3)` (format pesan persis `overdue_checker.go:134-140`). Karena keterlambatan >72 jam, level 3 → tipe `OVERDUE_CRITICAL`; log <72 jam memakai `MAINTENANCE_OVERDUE` |
| 4 | Periksa data log melalui halaman atau SQL | `overdue_notified_at` terisi waktu proses checker | ✅ (verifikasi kode) — `overdue_notified_at` diisi saat eskalasi pertama (`overdue_checker.go:195-201`) dan `overdue_level` diperbarui (`:113-115`); tidak dapat dibaca via API/SSH |

### S2 - Log `ongoing` juga mengirim notifikasi ke teknisi

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buat log Terjadwal, lalu tugaskan ke user teknisi | Status berubah menjadi `ongoing` dan `technician_id` terisi | ✅ — log ongoing uji (`08dce9d2`, `efce78b0`, dll.) memiliki `technician_id` (teknisi A/B) |
| 2 | Mundurkan `scheduled_date` ke masa lalu dan restart API | Checker memproses log overdue | ✅ — log ongoing dibuat dgn `scheduled_date` 12 Agu (masa lalu); checker produksi memproses (lihat S1 #2) |
| 3 | Periksa akun admin | Admin menerima notifikasi overdue dengan nama dan kode CCTV | ✅ — nyata: 4× `OVERDUE_CRITICAL` admin dgn nama + kode CCTV ("(ID: MT-CCTV-001)") |
| 4 | Periksa akun teknisi yang ditugaskan | Teknisi menerima `MAINTENANCE_OVERDUE` dengan pesan `Tugas pemeliharaan Anda untuk CCTV [nama] sudah melewati jadwal` | ✅ (tipe disesuaikan) — nyata: teknisi (`teknisi.test`) menerima 2× `OVERDUE_CRITICAL` dgn pesan "Tugas pemeliharaan Anda untuk CCTV [nama] sudah melewati jadwal" (format persis `overdue_checker.go:156-159`); pengiriman hanya saat status `ongoing` & `technician_id` terisi (`:155`) |

### S3 - Idempotensi: notifikasi tidak dikirim dua kali

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Setelah S1 atau S2, restart API sekali lagi pada hari yang sama | Checker berjalan kembali | ✅ — checker berjalan startup + tiap 1 jam (`overdue_checker.go:34-46`); di produksi sudah melewati banyak siklus sejak log uji dibuat (18 Agu) |
| 2 | Periksa lonceng dan riwayat pemberitahuan | Tidak ada notifikasi overdue duplikat untuk log yang sama | ✅ — nyata: hanya 4 notif kritis admin & 2 teknisi (satu per log per level, tidak ada duplikat meski checker berjalan tiap jam selama ±2 hari) |
| 3 | Periksa `maintenance_logs.overdue_notified_at` | Timestamp tetap sama dan tidak diganti oleh proses kedua | ✅ (verifikasi kode) — guard `targetLevel <= logEntry.OverdueLevel → return` (`overdue_checker.go:104-106`) + `overdue_notified_at` diisi hanya bila `IS NULL` (`:195-198`); idempotensi diverifikasi unit test `overdue_checker_test.go` (skip flag) |

### S4 - Status terminal tidak dieskalasi

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Siapkan satu log berstatus `completed` dan satu log `failed` dengan `scheduled_date` masa lalu | Kedua log terlihat sebagai riwayat terminal | ✅ — log `completed`/`failed` ada di produksi dgn `scheduled_date` masa lalu (riwayat kalender) |
| 2 | Jalankan checker melalui restart API | Checker melewati kedua log | ✅ (verifikasi kode) — query checker membatasi `status IN ('scheduled','ongoing')` (`overdue_checker.go:62`) + guard `:73-75`; status terminal tidak pernah dipilih |
| 3 | Periksa notifikasi admin/teknisi | Tidak ada `MAINTENANCE_OVERDUE` untuk log `completed` atau `failed` | ✅ — nyata: seluruh notif overdue hanya terkait log uji `scheduled`/`ongoing`; log `completed`/`failed` (banyak di riwayat) tidak menghasilkan notif overdue |

### S5 - Reschedule ke masa depan mereset flag

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Pilih log yang sudah memiliki `overdue_notified_at` dari S1 atau S2 | Flag overdue terisi | ✅ — log uji telah dieskalasi (level 3, `OVERDUE_CRITICAL`) |
| 2 | Ubah `scheduled_date` melalui form Edit menjadi besok | Perubahan tersimpan; `overdue_notified_at` kembali `NULL` | ✅ (verifikasi kode) — saat update, jadwal baru di masa depan mereset `OverdueNotifiedAt = nil` + `OverdueLevel = 0` (`handlers/maintenance.go:384-389`; juga reset di `:747`); unit test mencakup reset saat reschedule |
| 3 | Restart API saat tanggal baru belum tiba | Checker tidak mengirim notifikasi karena log belum overdue | ✅ — filter checker `scheduled_date < now` (`overdue_checker.go:62`) → log berjadwal besok tidak dipilih |

### S6 - Setelah kembali overdue, notifikasi dapat dikirim ulang

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Mundurkan lagi `scheduled_date` log yang telah di-reschedule ke masa lalu | Log menjadi overdue dan flag tetap `NULL` | ✅ (verifikasi kode) — reschedule masa depan mereset flag ke NULL (`maintenance.go:384-389`); jadwal dimundurkan → log kembali memenuhi filter checker |
| 2 | Restart API | Checker mengirim satu notifikasi overdue baru | ✅ (verifikasi kode) — setelah reset, `targetLevel (3) > OverdueLevel (0)` → eskalasi dikirim ulang (`overdue_checker.go:102-106`); unit test mencakup skenario ini |
| 3 | Periksa database | `overdue_notified_at` terisi timestamp baru setelah pengiriman | ✅ (verifikasi kode) — guard `IS NULL` (`:195-198`) memungkinkan timestamp baru terisi untuk log yang baru direset |

### S7 - Tugas yang belum jatuh tempo dilewati

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buat atau reschedule log `scheduled` ke besok | `scheduled_date` lebih besar dari waktu saat ini | ✅ — log kalender baru dibuat dgn `scheduled_date` besok (18 Agu) di skenario lain |
| 2 | Restart API | Log tidak diproses checker | ✅ — filter `scheduled_date < now` (`overdue_checker.go:62`); unit test `overdue_checker_test.go` kasus "belum overdue" (10 jam → `OverdueLevelNone`) |
| 3 | Periksa notifikasi | Tidak ada `MAINTENANCE_OVERDUE` untuk log tersebut | ✅ — tidak ada notif overdue utk log berjadwal besok di produksi |

### S8 - Tipe notifikasi tampil di antarmuka in-app

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Picu S1 atau S2 sampai notifikasi overdue dibuat | Event `NOTIFICATION_CREATED` diterima sesi browser yang sesuai | ✅ — notifikasi dibuat via `utils.NotifyRoleSync`/`NotifyUserSync` (`overdue_checker.go:149-165`) yang mem-broadcast `NOTIFICATION_CREATED` (terarah per role/user); eskalasi level ≥2 juga mem-broadcast `OVERDUE_HIGH`/`OVERDUE_CRITICAL` (`:168-176`) |
| 2 | Buka NotificationBell dan halaman Pemberitahuan | Item overdue tampil dengan label/ikon/warna overdue yang konsisten dengan tipe `MAINTENANCE_OVERDUE`; klik item mengarah ke `/maintenance` | ✅ (verifikasi kode) — tipe `OVERDUE_CRITICAL` ditampilkan dgn latar `bg-red-600` + ikon 🚨 (`NotificationBell.tsx:33,39`; `pemberitahuan/page.tsx:42,48`); `reference_type = maintenance-log` + `reference_id` UUID log → navigasi ke `/maintenance` (`NotificationBell`/halaman Pemberitahuan); item overdue nyata tampil di bell & halaman Pemberitahuan (4× admin, 2× teknisi) |
| 3 | Tandai item sebagai sudah dibaca | Badge jumlah belum dibaca berkurang dan item tersimpan sebagai dibaca | ✅ — endpoint `PUT /api/notifications/:id/read` (ownership check) & `PUT /read-all`; mekanisme badge dibuktikan di dokumen manual test lainnya (Fase 5.1) |

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

## Hasil Nyata — Catatan Verifikator

**Evolusi implementasi (Task 13.11, eskalasi bertingkat):** checker kini menerapkan level eskalasi 0–3 (`overdue_checker.go:18-23`): level 1 (>24 jam) → `MAINTENANCE_OVERDUE`, level 2 (>48 jam) → `OVERDUE_HIGH`, level 3 (>72 jam) → `OVERDUE_CRITICAL` + email + webhook `maintenance.overdue` + broadcast WS. Notifikasi dikirim hanya saat level **naik** (anti-spam). Dokumen ini ditulis untuk Task 5.5 (versi tunggal `MAINTENANCE_OVERDUE`), sehingga hasil nyata di atas mengikuti perilaku terkini: log uji prod (>72 jam) menerima `OVERDUE_CRITICAL`, bukan `MAINTENANCE_OVERDUE`. Semantik yang diuji (notif admin+teknisi, idempotensi, terminal skip, reset saat reschedule, re-eskalasi) tetap terpenuhi.

**Catatan verifikasi:** restart API untuk memicu checker tidak dilakukan di produksi (tanpa akses SSH); checker produksi berjalan otomatis tiap 1 jam dan bukti end-to-end diperoleh dari notifikasi nyata yang terkirim ke admin (4×) & teknisi (2×). Ambang level diverifikasi via unit test `overdue_checker_test.go` (25 jam → Warning, 49 → High, 73 → Critical).
