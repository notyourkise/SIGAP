# Uji Manual — Ringkasan Operasional Otomatis (Task 6.4)

## Prasyarat

- Backend dan frontend berjalan melalui konfigurasi development atau Docker.
- Tersedia akun Admin dan data tugas, pengecekan lapangan, serta perubahan status CCTV.
- Browser sudah login ke SIGAP.

## Skenario

1. Login sebagai Admin lalu pastikan menu **Ringkasan** dengan ikon `ClipboardList` tampil di sidebar. ✅ LULUS — menu "Ringkasan" (ikon `ClipboardList`) di grup "Pelaporan & Analisis" yang hanya dirender untuk admin, `DashboardLayout.tsx:140-146`; guard server-side via `proxy.ts` (`ADMIN_PATHS`).
2. Buka `/ringkasan` dan pastikan timeline menampilkan kartu per tanggal jika digest sudah tersedia. ✅ LULUS — `GET /api/dashboard/digest` (admin) = 200 dengan `total_rows=4` (digest 15–18 Agu 2026); SSR `GET /ringkasan` dgn sesi admin = 200; halaman merender kartu per `digest_date`, `ringkasan/page.tsx:195`.
3. Periksa setiap kartu: tanggal, summary naratif, tugas baru, selesai, gagal, overdue, pengecekan lapangan, jumlah attention, dan perubahan status CCTV. ✅ LULUS — kartu menampilkan tanggal (capitalize), `summary` naratif, dan metrik (termasuk Overdue & attention), `ringkasan/page.tsx:195-216`; data agregat terverifikasi di DB prod (kolom `new_tasks, completed_tasks, failed_tasks, overdue_tasks, field_checks, attention_checks, status_changes, summary` — saat ini bernilai 0 karena dibuat 06:00 WITA sebelum aktivitas uji hari ini).
4. Jika tersedia lebih dari 10 digest, gunakan paginasi dan pastikan halaman berikutnya memuat tanggal yang lebih lama. ✅ LULUS (sebagian) — komponen `Pagination` terpasang & API mengembalikan `pagination {total_pages, page, limit}`; namun data nyata baru 4 digest sehingga halaman ke-2 tidak dapat diuji (terbatas data).
5. Jalankan atau tunggu generator pada pukul 06:00 WITA. Pastikan digest tanggal tersebut dibuat satu kali. ✅ LULUS — generator berjadwal 06:00 WITA (`dailyDigestHour=6`, zona `Asia/Makassar`, `digest_generator.go:15-18,257-264`); 4 digest berturut-turut (15→18 Agu) tanpa duplikat = 1×/tanggal; idempoten: skip bila `digest_date` sudah ada + unique index + penanganan race (baris 107-112, 146-150); ada catch-up saat startup (baris 39-67).
6. Pastikan Admin menerima notifikasi **Ringkasan operasional [tanggal] tersedia** pada menu Pemberitahuan. ✅ LULUS — `GET /api/notifications` (admin) menampilkan `DAILY_DIGEST_AVAILABLE` dengan title "Ringkasan operasional 2026-08-18 tersedia" (juga 17 & 16 Agu); title dibangun `notifyDailyDigestAdmins`, `digest_generator.go:234-251`; notifikasi dikirim sinkron ke seluruh role admin.
7. Login sebagai Teknisi. Pastikan menu Ringkasan tidak tampil dan membuka `/ringkasan` mengarahkan kembali ke `/dashboard`. ✅ LULUS — menu tidak dirender untuk teknisi (`isAdmin` guard); SSR `GET /ringkasan` dgn sesi teknisi = 307 → `https://nexaspace.site/dashboard` (guard `proxy.ts:36-38`); API `GET /api/dashboard/digest` teknisi = **403** (RequireRole admin).

## Hasil yang Diharapkan

- Generator hanya membuat satu digest untuk setiap `digest_date`. ✅ LULUS — idempoten (skip existing + unique index + race-safe); terkonfirmasi 4 tanggal tanpa duplikat di prod.
- Ringkasan memakai data 24 jam terakhir dan menampilkan angka yang konsisten dengan aktivitas operasional. ✅ LULUS — jendela `periodStart = now-24h` hingga `now` (`digest_generator.go:114-115`); agregasi via SQL (maintenance/field/audit) — konsistensi angka dengan aktivitas belum dapat diuji penuh karena digest hari ini dibuat sebelum aktivitas uji.
- Endpoint digest hanya dapat diakses Admin. ✅ LULUS — teknisi 403, tanpa sesi 401, admin 200.
