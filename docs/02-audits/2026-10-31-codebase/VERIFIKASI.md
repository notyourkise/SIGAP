# Bukti dan batas verifikasi audit SIGAP

Tanggal: 31 Oktober 2026, WITA. Checkout HEAD `edadd8b1cea2eb316b0f26bcb08c9067eb07be13`; working tree memiliki perubahan pengguna pada CCTVMap.tsx yang sudah ada sebelum audit.

## Baseline yang benar-benar dijalankan

1. Dari `backend`: `go test ./... -count=1 -timeout=180s`, exit 0.

```text
?       sigap-backend                      [no test files]
?       sigap-backend/cmd/migrate          [no test files]
?       sigap-backend/cmd/vapid-keygen     [no test files]
ok      sigap-backend/config              0.425s
?       sigap-backend/docs                [no test files]
ok      sigap-backend/handlers            16.656s
ok      sigap-backend/middleware          0.820s
?       sigap-backend/migrations          [no test files]
ok      sigap-backend/models              0.361s
ok      sigap-backend/monitoring          0.244s
ok      sigap-backend/routes              0.521s
ok      sigap-backend/utils               3.829s
```

2. Dari `backend`: `go vet ./...`, exit 0, tanpa diagnostik.
3. Dari `frontend`: `node --test tests/service-worker-cache.test.mjs tests/*.test.mts`, exit 0: 37 test, 37 pass, 0 fail, 0 skipped. Mencakup HEVC, live-status, monitor, service worker, encoding, dan WHEP status. Node mengeluarkan warning module type saat membaca beberapa file TypeScript; tidak menyebabkan test gagal.
4. Dari `frontend`: `npx tsc --noEmit --incremental false`, tanpa diagnostik. Sesudahnya `npm run lint`, exit 0: 79 warning, 0 error. Warning tidak dianggap bukti bug runtime secara otomatis.
5. `docker compose ps --format json`, exit 1: pipe Docker Desktop Linux tidak tersedia. Tidak ada koneksi ke DB operasional melalui perintah ini.

Baseline di atas diringkas dari keluaran command pada sesi audit; log reproduksi khusus disimpan lengkap di direktori ini. Build produksi frontend dan aplikasi mobile tidak dijalankan. Test integrasi MediaMTX yang memerlukan MEDIAMTX_TEST_API_URL tidak diaktifkan; variabel itu tidak terisi pada sesi ini.

## Pengujian audit tambahan

Dari direktori `backend`, jalankan:

```powershell
go test -overlay 'D:\laragon\www\SIGAP\docs\02-audits\2026-10-31-codebase\overlay.json' ./handlers -run '^TestAudit_' -count=1 -v -timeout=90s
go test -overlay 'D:\laragon\www\SIGAP\docs\02-audits\2026-10-31-codebase\overlay.json' ./utils -run '^TestAudit_' -count=1 -v -timeout=30s
```

**Hasil yang tercatat sebelum perbaikan: exit 1 pada kedua command.** Handler: 20 skenario, 19 FAIL dan 1 PASS. Utils: 1 skenario, 1 FAIL. FAIL berarti perilaku kode melanggar ekspektasi test audit. Tidak ada patch perbaikan yang diterapkan untuk membuat test hijau.

Overlay memetakan file test virtual `zz_audit_regression_test.go` dan `zz_audit_websocket_test.go` ke sumber `.go.txt` pada folder audit. Tidak ada file test virtual yang ditulis ke direktori backend. Path overlay bersifat lokal mesin ini; sesuaikan bila checkout dipindahkan.

SQL pada test memakai sqlmock, bukan PostgreSQL. Beberapa reload kosong dan query sampingan async tidak diberi fixture lengkap; mock GORM memakai logger silent. Kesimpulan memakai status/field/operasi yang ditargetkan serta ekspektasi SQL penting yang terpenuhi, bukan kegagalan query sampingan. Test kontrak mobile hanya memeriksa bentuk JSON backend terhadap pemakaian di mobile, bukan mengeksekusi React Native.

Test concurrent stock mempertemukan dua pembacaan stok awal 10. Kedua request sukses memakai 4 dan kedua UPDATE menulis 6. Ini membuktikan jalur absolute-write dapat kehilangan pengurangan pada interleaving tersebut; bukan simulasi mesin PostgreSQL yang lengkap.

Test file hanya membuat satu file `audit-owned-<UUID>.txt` dengan O_EXCL di working directory test, lalu memeriksa bahwa handler menghapusnya. Cleanup juga hanya menunjuk nama file tersebut. Tidak ada file aplikasi/unggahan pengguna yang dijadikan target. Sebelum menjalankan suite bawaan, `backend/handlers/uploads` dipastikan belum ada karena satu test bawaan menghapus direktori uploads relatif working directory package.

## Matriks role dan modul

| Area | Admin | Teknisi | Batas bukti |
|---|---|---|---|
| Login/password/profile/pengguna | Kode + suite bawaan + test forced password | Kode + suite bawaan + kontrak mobile | Tanpa kredensial nyata |
| CCTV/QR/site/zona | Kode + suite bawaan + create zona | Kode + suite bawaan + akses CCTV lintas site | Endpoint lain dalam H02 ditelusuri dari kode |
| Pemeliharaan/checklist | Kode + suite bawaan | Kode + test lintas site, ownership, completion palsu | Tanpa DB nyata |
| Stok | Kode + suite bawaan + interleaving concurrent | Test write/read usage tugas lain | SQL simulasi, tidak mengurangi stok operasional |
| Jadwal/SLA/kalender | Kode + suite bawaan + tanggal/SLA/generator | Kode konsumsi tugas/notifikasi | Kalender tanpa browser; race generator belum diuji DB |
| Dashboard/kinerja/digest/laporan | Kode + suite bawaan + filter laporan/SLA | Scope laporan pada suite bawaan | Tidak ada penerimaan black-box |
| WebSocket/notifikasi/email/push | Kode + suite bawaan | Kode + penerima lintas site pada socket lokal | Email/push nyata tidak dikirim |
| API key/webhook | Kode + suite bawaan | Batas role dari route/middleware | Tidak memakai key/integrasi nyata |
| Streaming/snapshot | Kode/konfigurasi + suite bawaan | Batas role dari route dan akses media | Kamera/video aktif tidak diuji |
| Mobile | Kontrak API terkait | Sumber auth, scan, checkup, task, map, offline | Tanpa emulator/build perangkat |

Jalur anonim ditinjau pada middleware/route dan konfigurasi media H01. API key ditinjau sebagai mekanisme integrasi eksternal. Pembahasan dua role tidak berarti semua kombinasi endpoint, data, dan state sudah diuji.

## Perubahan artefak

Audit hanya menambah direktori laporan ini. Tidak ada perubahan kode produksi, migrasi, konfigurasi aktif atau database. Tidak ada commit/push/deploy. Perubahan pengguna yang sudah ada tidak ditimpa.

## Verifikasi database lanjutan

Setelah Docker tersedia, database existing diperiksa read-only pada 31 Oktober 2026 pukul 23.00 WITA. Hasil rinci dan batas terbaru ada di [VERIFIKASI-DATABASE.md](VERIFIKASI-DATABASE.md). Ringkasan prioritas diperbarui menjadi 7 High, 15 Medium, 3 Low; jumlah temuan tetap 25. Tidak ada test mutasi yang dijalankan pada database existing.

