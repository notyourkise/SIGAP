# Uji Manual - Eskalasi Overdue Otomatis

> **Kontrak uji formal — MT-13**
>
> Dokumen ini mendefinisikan prosedur; hasil sebelum **2026-08-20** adalah catatan historis/non-authoritative dan harus diuji ulang dengan execution record formal. Inspeksi kode, review unit test, atau checkmark lama bukan `PASS` black-box.
>
> **Skenario audit reproduktif:** Dengan fake clock atau test trigger dan fixture overdue, jalankan matriks eskalasi 24/48/72 jam, reschedule, anti-duplikasi, dan seluruh kanal penerima.
>
> **Status suite:** `NOT RUN`

### Formal execution metadata

| Field | Value |
|---|---|
| Test ID / Audit ID | MT-13 |
| Capability owner | QA Workflow + Backend |
| Status suite | NOT RUN |
| Build SHA | NOT RECORDED |
| Artifact digest/ID | NOT RECORDED |
| Tester | NOT RECORDED |
| Timestamp WITA | NOT RECORDED |
| Environment | NOT RECORDED |
| Fixture prefix/data | `E2E-MT13-` / NOT RECORDED |
| Cleanup result | NOT RUN |
| Expected | Level 1→2→3, anti-duplikasi, reset reschedule, penerima, email/webhook, dan status terminal terbukti end-to-end. |
| Actual | NOT RECORDED |
| Evidence path/link | NOT RECORDED |
| Mandatory flag | YES |
| Scenario status | NOT RUN |

### Formal scenario record

| ID | Mandatory | Expected | Actual | Evidence | Status |
|---|---|---|---|---|---|
| MT-13-S1 | YES | Level 1→2→3, anti-duplikasi, reset reschedule, penerima, email/webhook, dan status terminal terbukti end-to-end. | NOT RECORDED | NOT RECORDED | NOT RUN |

> **Cakupan aggregate Wave 0:** Baris formal aggregate di atas adalah **sole atomic suite-level acceptance gate** untuk baseline Wave 0. Skenario legacy terperinci di bawah ini tetap non-formal dan historical/non-authoritative sampai diperluas selama Wave 7.

**Aturan kesimpulan suite:** sebelum eksekusi suite `NOT RUN`; setelah run dimulai, setiap mandatory scenario harus `PASS`. Satu mandatory scenario berstatus `FAIL`, `BLOCKED`, atau `NOT RUN` membuat kesimpulan suite `FAIL`.

> **Guard keselamatan wajib:** Untuk setiap SQL mutasi, `DELETE`/`TRUNCATE`, rotasi/regenerasi key atau QR, penghapusan, reset kredensial, atau mutasi credential, lanjut hanya bila environment bukan production dan target bukan `nexaspace.site`, fixture prefix `E2E-MT13-` serta exact targets sudah dicatat, backup/recovery terverifikasi untuk perubahan data/key state, dan cleanup/rollback didefinisikan serta dicatat setelah run. Jika salah satu syarat tidak tersedia, hentikan prosedur dan set scenario status ke `BLOCKED`; jangan jalankan pada production. Jangan menyalin credential, token, private key, RTSP password, atau payload production yang belum disanitasi.


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

> **Guard P6 — wajib sebelum query mutasi:** Jalankan query berikut hanya pada environment non-production dan target yang bukan `nexaspace.site`, dengan fixture prefix `E2E-MT13-` serta exact `<LOG_UUID>` yang sudah dicatat. Verifikasi backup/recovery untuk perubahan log dan definisikan cleanup/rollback sebelum eksekusi; jika salah satu syarat tidak tersedia, hentikan dan set scenario status ke `BLOCKED`.

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

| Item | Observasi historis (non-authoritative) |
|------|-------------|
| `go build ./...` | NOT RUN |
| `go test ./... -count=1` (subset `-run "Overdue"`) | NOT RUN |
| `go vet ./...` | NOT RUN |
| `swag init` | NOT RUN |

## 3. Skenario Uji

### S1 - Log `scheduled` overdue mengirim notifikasi ke semua admin

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Admin membuka halaman Pemeliharaan CCTV dan membuat log baru berstatus Terjadwal dengan `scheduled_date` sementara di masa depan | Log tersimpan dengan status `scheduled` dan `overdue_notified_at` kosong | NOT RUN |
| 2 | Ubah `scheduled_date` log ke masa lalu memakai cara percepatan P6, lalu restart API | Checker memilih log tersebut pada proses startup | NOT RUN |
| 3 | Buka lonceng/halaman Pemberitahuan pada setiap akun admin | Setiap admin menerima tipe `MAINTENANCE_OVERDUE` dengan pesan `Tugas pemeliharaan [nama CCTV] (ID: [kode]) sudah melewati jadwal [tanggal]` | NOT RUN |
| 4 | Periksa data log melalui halaman atau SQL | `overdue_notified_at` terisi waktu proses checker | NOT RUN |

### S2 - Log `ongoing` juga mengirim notifikasi ke teknisi

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Buat log Terjadwal, lalu tugaskan ke user teknisi | Status berubah menjadi `ongoing` dan `technician_id` terisi | NOT RUN |
| 2 | Mundurkan `scheduled_date` ke masa lalu dan restart API | Checker memproses log overdue | NOT RUN |
| 3 | Periksa akun admin | Admin menerima notifikasi overdue dengan nama dan kode CCTV | NOT RUN |
| 4 | Periksa akun teknisi yang ditugaskan | Teknisi menerima `MAINTENANCE_OVERDUE` dengan pesan `Tugas pemeliharaan Anda untuk CCTV [nama] sudah melewati jadwal` | NOT RUN |

### S3 - Idempotensi: notifikasi tidak dikirim dua kali

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Setelah S1 atau S2, restart API sekali lagi pada hari yang sama | Checker berjalan kembali | NOT RUN |
| 2 | Periksa lonceng dan riwayat pemberitahuan | Tidak ada notifikasi overdue duplikat untuk log yang sama | NOT RUN |
| 3 | Periksa `maintenance_logs.overdue_notified_at` | Timestamp tetap sama dan tidak diganti oleh proses kedua | NOT RUN |

### S4 - Status terminal tidak dieskalasi

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Siapkan satu log berstatus `completed` dan satu log `failed` dengan `scheduled_date` masa lalu | Kedua log terlihat sebagai riwayat terminal | NOT RUN |
| 2 | Jalankan checker melalui restart API | Checker melewati kedua log | NOT RUN |
| 3 | Periksa notifikasi admin/teknisi | Tidak ada `MAINTENANCE_OVERDUE` untuk log `completed` atau `failed` | NOT RUN |

### S5 - Reschedule ke masa depan mereset flag

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Pilih log yang sudah memiliki `overdue_notified_at` dari S1 atau S2 | Flag overdue terisi | NOT RUN |
| 2 | Ubah `scheduled_date` melalui form Edit menjadi besok | Perubahan tersimpan; `overdue_notified_at` kembali `NULL` | NOT RUN |
| 3 | Restart API saat tanggal baru belum tiba | Checker tidak mengirim notifikasi karena log belum overdue | NOT RUN |

### S6 - Setelah kembali overdue, notifikasi dapat dikirim ulang

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Mundurkan lagi `scheduled_date` log yang telah di-reschedule ke masa lalu | Log menjadi overdue dan flag tetap `NULL` | NOT RUN |
| 2 | Restart API | Checker mengirim satu notifikasi overdue baru | NOT RUN |
| 3 | Periksa database | `overdue_notified_at` terisi timestamp baru setelah pengiriman | NOT RUN |

### S7 - Tugas yang belum jatuh tempo dilewati

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Buat atau reschedule log `scheduled` ke besok | `scheduled_date` lebih besar dari waktu saat ini | NOT RUN |
| 2 | Restart API | Log tidak diproses checker | NOT RUN |
| 3 | Periksa notifikasi | Tidak ada `MAINTENANCE_OVERDUE` untuk log tersebut | NOT RUN |

### S8 - Tipe notifikasi tampil di antarmuka in-app

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Picu S1 atau S2 sampai notifikasi overdue dibuat | Event `NOTIFICATION_CREATED` diterima sesi browser yang sesuai | NOT RUN |
| 2 | Buka NotificationBell dan halaman Pemberitahuan | Item overdue tampil dengan label/ikon/warna overdue yang konsisten dengan tipe `MAINTENANCE_OVERDUE`; klik item mengarah ke `/maintenance` | NOT RUN |
| 3 | Tandai item sebagai sudah dibaca | Badge jumlah belum dibaca berkurang dan item tersimpan sebagai dibaca | NOT RUN |

## 4. Kriteria Penerimaan

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

## Observasi Historis (Non-authoritative) — Catatan Verifikator

**Evolusi implementasi (Task 13.11, eskalasi bertingkat):** checker kini menerapkan level eskalasi 0–3 (`overdue_checker.go:18-23`): level 1 (>24 jam) → `MAINTENANCE_OVERDUE`, level 2 (>48 jam) → `OVERDUE_HIGH`, level 3 (>72 jam) → `OVERDUE_CRITICAL` + email + webhook `maintenance.overdue` + broadcast WS. Notifikasi dikirim hanya saat level **naik** (anti-spam). Dokumen ini ditulis untuk Task 5.5 (versi tunggal `MAINTENANCE_OVERDUE`), sehingga hasil nyata di atas mengikuti perilaku terkini: log uji prod (>72 jam) menerima `OVERDUE_CRITICAL`, bukan `MAINTENANCE_OVERDUE`. Semantik yang diuji (notif admin+teknisi, idempotensi, terminal skip, reset saat reschedule, re-eskalasi) tetap terpenuhi.

**Catatan verifikasi:** restart API untuk memicu checker tidak dilakukan di produksi (tanpa akses SSH); checker produksi berjalan otomatis tiap 1 jam dan bukti end-to-end diperoleh dari notifikasi nyata yang terkirim ke admin (4×) & teknisi (2×). Ambang level diverifikasi via unit test `overdue_checker_test.go` (25 jam → Warning, 49 → High, 73 → Critical).
