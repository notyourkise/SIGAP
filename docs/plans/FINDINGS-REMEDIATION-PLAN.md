# Findings Remediation Implementation Plan

> **Status:** sedang dieksekusi — Gelombang 1, Task 1A implementasi dan runtime gate QR selesai; Task 1B berikutnya.
>
> **Tanggal rencana:** 19 Agustus 2026. **Mulai eksekusi:** 20 Agustus 2026.
> **Sumber:** `docs/audits/BLACKBOX-FINDINGS-2026-08-19.md` dan `docs/audits/MANUAL-TEST-AUDIT-SUMMARY.md`.

## Status Eksekusi

- [x] Baseline lokal pada `5394ea0c543766aa9979115f899307076152fd12`: backend build/test/vet lulus; frontend lint 0 error (77 warning baseline) dan build lulus.
- [x] Task 0.1 — artifact production, topologi edge, checkout SHA, image digest/ID, waktu runtime, port, dan konfigurasi non-secret telah dicatat.
- [x] Task 0.2 — governance manual/black-box formal, metadata contract, historical-status handling, dan safety guard selesai; tidak ada runtime PASS yang diciptakan.
- [x] Task 1A — lifecycle QR diperbaiki pada source: rotasi memakai ID CCTV yang sama, scanner menolak payload lama, cache label memakai versi payload, dan regression test lulus.
- [x] Task 1A runtime gate — label QR baru berubah tanpa hard refresh, payload hasil decode diterima, QR lama ditolak, RBAC/audit tervalidasi, dan staging dibersihkan.
- [ ] Task 1B dan Gelombang 2–7 — belum dimulai.

## Tujuan

Menutup seluruh temuan black-box dan audit manual test melalui perubahan kecil yang dapat ditinjau, diuji, dan dirilis bertahap. Urutan mengutamakan pemulihan fungsi kritis, keamanan/isolasi data, integritas workflow, lalu kualitas UI dan bukti pengujian.

## Aturan Eksekusi

1. Gunakan branch/worktree terpisah per workstream; jangan menggabungkan seluruh perbaikan dalam satu PR.
2. Tulis regression test yang gagal terlebih dahulu untuk setiap bug yang dapat direproduksi.
3. Jangan menguji perubahan destruktif pada production. Gunakan fixture berawalan `E2E-`, environment marker, backup, dan cleanup terverifikasi.
4. Backend authorization adalah batas keamanan utama; guard frontend hanya defense-in-depth.
5. Setiap PR memperbarui changelog terkait, manual test terkait, dan traceability ID `BB-*`/`MT-*`.
6. Temuan deployment tidak boleh ditutup hanya karena source terlihat benar; bukti harus berasal dari artifact yang sedang aktif.

## Decision Gate Sebelum Coding

Keputusan berikut harus disetujui pemilik produk pada awal eksekusi:

| ID | Keputusan | Keputusan pemilik produk | Status |
|---|---|---|---|
| DG-01 | Arti jadwal tanpa teknisi | Tetap `scheduled` dan tampil “Belum ada teknisi”; admin tidak otomatis menjadi pelaksana. | ✅ Disetujui 2026-08-20 |
| DG-02 | Waktu notifikasi jadwal | Kirim “Jadwal dibuat” saat create dan “Tugas tersedia” saat occurrence jatuh tempo. | ✅ Disetujui 2026-08-20 |
| DG-03 | Model inventaris kawasan | Master barang global + tabel stok per site `spare_part_stocks`; stok/minimum order disimpan per site. | ✅ Disetujui 2026-08-20 |
| DG-04 | Akses teknisi ke inventaris | Halaman dan API admin ditutup; teknisi hanya memakai endpoint opsi minimal yang di-scope ke site tugas maintenance. | ✅ Disetujui 2026-08-20 |
| DG-05 | Navigasi forced password | Tidak ada kembali ke aplikasi; sediakan logout/kembali login. Setelah sukses menuju `returnTo` aman atau dashboard. Mode biasa kembali ke halaman sebelumnya. | ✅ Disetujui 2026-08-20 |
| DG-06 | Zona unassignment notification | Kirim `ZONE_ASSIGNED` dan `ZONE_UNASSIGNED` setelah transaksi berhasil; perubahan tanpa delta tidak mengirim duplikat. | ✅ Disetujui 2026-08-20 |

## Gelombang 0 — Baseline, Provenance, dan Reproduksi

### Task 0.1 — Catat artifact production

Status: **✅ SELESAI — evidence diambil read-only dari host production pada 2026-08-20**.

- [x] Catat source branch/SHA pembanding dan konfigurasi non-secret di repository.
- [x] Catat response publik frontend, API health, edge Nginx, dan route MediaMTX.
- [x] Dokumentasikan perbedaan edge production dengan container Nginx di source.
- [x] Catat checkout/deploy SHA aktual pada host production.
- [x] Catat image digest atau immutable image ID, created/start time, dan versi aktual untuk frontend, API, Nginx, serta MediaMTX.
- [x] Catat nama variable runtime yang tersedia tanpa menyalin nilainya, port binding, dan konfigurasi ICE aktual.
- [x] Cocokkan seluruh artifact host dengan source branch; gap revision label dan dirty worktree dicatat sebagai temuan provenance.

Target operasional:

- Catat commit SHA, image digest, waktu deploy, dan environment name untuk frontend, API, Nginx, serta MediaMTX.
- Catat konfigurasi non-secret: nama variable tersedia/tidak, port, hostname ICE, dan versi image.
- Jangan menyalin token, private key, password RTSP, atau payload sensitif.

Output:

- Evidence untuk BB-08, BB-13, BB-16, BB-20, dan perbedaan BB-25 dengan changelog.
- Matriks “production artifact vs source branch”.
- Dokumen kerja: `docs/audits/PRODUCTION-ARTIFACT-PROVENANCE-2026-08-20.md`.

### Task 0.2 — Tetapkan status test formal

Status: **SELESAI — 2026-08-20**.

Files:

- Modify: `docs/TESTING.md`
- Modify bertahap: seluruh `docs/manual-test-*.md`

Perubahan:

- Gunakan hanya `PASS`, `FAIL`, `BLOCKED`, `NOT RUN`.
- Tambahkan build SHA, tester, timestamp WITA, environment, fixture, cleanup, expected, actual, evidence path/link.
- Larang “verifikasi kode” sebagai `PASS` black-box.
- Kesimpulan suite gagal bila mandatory scenario bukan `PASS`.

Checkpoint:

- [x] Seluruh temuan MT-01–MT-25 memiliki owner dan skenario reproduksi.
- [x] Semua prosedur SQL/rotasi/key memiliki guard non-production.

## Gelombang 1 — P0 Dikerjakan Paralel

### Task 1A — Perbaiki lifecycle rotasi QR

Status: **✅ SELESAI — implementasi dan runtime black-box QR non-production `PASS` pada 2026-08-20**.

Temuan: BB-10, MT-11.

Files utama:

- Modify: `backend/handlers/cctv.go`
- Modify: `backend/handlers/cctv_test.go`
- Modify: `frontend/src/app/cctv/page.tsx`
- Modify: `docs/manual-test-infra-integration.md`

Langkah:

- [x] Tambah test bahwa rotate menghasilkan ciphertext baru yang mendekripsi ke ID CCTV yang sama.
- [x] Tambah test bahwa scanner menerima payload aktif dan menolak payload sebelumnya.
- [x] Ubah rotasi agar mengenkripsi `cctv.ID.String()` dengan nonce baru, bukan UUID baru.
- [x] Saat scan, cocokkan ID hasil dekripsi dan `qr_payload` aktif.
- [x] Gunakan cache key frontend berbasis `id + qr_payload`, HTTP/fetch `no-store`, dan bypass service-worker cache agar payload baru tidak memakai gambar lama; naikkan versi shell cache untuk membuang cache lama.
- [x] Uji cetak label langsung setelah rotasi tanpa hard refresh pada fixture non-production `E2E-MT11-`.

Verifikasi:

```powershell
cd backend
go test ./handlers -run 'TestRotateQRCode|TestScanCCTV' -count=1
go test ./... -count=1
```

Done: QR baru bekerja untuk aset yang sama, QR lama ditolak, dan label baru tampil.

Evidence 2026-08-20: targeted backend test dan Node service-worker cache test lulus. Runtime staging terisolasi juga membuktikan rotasi UI/API mempertahankan ID, PNG/payload berubah, label terbaru tampil dan siap dicetak tanpa hard refresh, payload hasil decode diterima, QR sebelumnya ditolak, teknisi mendapat 403, audit tercatat, dan cleanup tidak menyentuh production. Lihat `docs/audits/MT-11-QR-RUNTIME-EVIDENCE-2026-08-20.md`. Suite agregat MT-11 tetap `NOT RUN` sampai API key dan webhook diuji.

### Task 1B — Pulihkan MediaMTX dan playback eksternal

Temuan: BB-08, MT-19, MT-21, MT-24, SYS-MT-01.

Files utama:

- Modify: `docker-compose.yml`
- Modify: `mediamtx.yml`
- Modify: `.env.example`
- Modify: `backend/utils/mediamtx.go`
- Modify: `backend/handlers/stream.go`
- Modify: `frontend/src/app/monitor/page.tsx`
- Modify: manual test streaming/RTSP/visual terkait

Langkah:

1. Pin image MediaMTX ke versi eksplisit.
2. Konfigurasikan hostname/IP publik WebRTC melalui environment dan `webrtcAdditionalHosts`.
3. Verifikasi UDP 8189/firewall; dokumentasikan TURN/TLS bila jaringan klien memblokir UDP.
4. Pisahkan health state: API reachable, path registered, source ready, WHEP signaling, ICE connected, frame received.
5. Sediakan RTSP simulator/non-production source untuk pengujian deterministik.
6. Perbaiki UI agar tidak menyamakan registrasi path dengan stream siap diputar.

Verifikasi:

```powershell
docker compose config --quiet
cd backend
go test ./utils ./handlers -run 'Test.*MediaMTX|Test.*Stream' -count=1
```

Runtime gate: playback terbukti dari desktop dan jaringan seluler eksternal; tidak cukup hanya `ready=true`.

## Gelombang 2 — Otorisasi dan Isolasi Kawasan

### Task 2.1 — Terapkan site scope sebagai invariant

Temuan: BB-03, BB-04, BB-19, BB-23, MT-03, MT-12, MT-18, MT-20, MT-25.

Files utama:

- Modify: `backend/middleware/*site*`
- Modify: `backend/routes/cctv.go`
- Modify: `backend/routes/maintenance.go`
- Modify: `backend/routes/spare_part.go`
- Modify: handlers CCTV, maintenance, report, zone, schedule, spare part
- Modify: frontend pages CCTV, maintenance, spare parts, sebaran CCTV, report builder

Langkah:

1. Tulis contract test dua site × dua user untuk list/detail/write/export.
2. Pasang `SiteContext()` secara konsisten pada route yang site-aware.
3. Filter maintenance melalui relasi `maintenance_logs.cctv_id → cctvs.site_id`.
4. Preload nama/kode Site untuk kolom/filter CCTV.
5. Terapkan site scope pada report/export/template dan zona.
6. Implementasikan model stok per site pada Task 3.4 sebelum menutup scope inventaris.
7. Tambah filter zona pada peta dengan statistik/grid yang memakai dataset sama.

Done:

- Site A tidak dapat membaca atau mengubah data Site B melalui list, detail, write, export, map, atau stok.
- Invalid/unassigned site mengikuti kebijakan eksplisit dan diuji.

### Task 2.2 — Tutup akses admin-only secara berlapis

Temuan: BB-16, BB-20, MT-18, MT-20, SYS-MT-03.

Files utama:

- Modify: `frontend/src/proxy.ts`
- Modify: `frontend/src/app/laporan-kustom/page.tsx`
- Modify: `frontend/src/app/suku-cadang/page.tsx`
- Modify: `backend/routes/report.go`
- Modify: `backend/routes/spare_part.go`
- Add: endpoint opsi suku cadang teknisi pada route/handler terkait

Langkah:

1. Reproduksi direct URL memakai artifact production.
2. Tambah E2E test anonymous/admin/teknisi untuk shell halaman dan API.
3. Pertahankan `RequireRole("admin")` pada report; tambah guard komponen agar tidak ada flash.
4. Pisahkan API inventaris admin dari API opsi teknisi dengan field minimum dan site scope.
5. Verifikasi path turunan ikut terlindungi.

Done: teknisi selalu 403 pada admin API dan tidak melihat konten halaman admin, namun workflow pemakaian suku cadang tetap berfungsi.

## Gelombang 3 — Integritas Data dan Workflow

### Task 3.1 — Normalisasi DTO edit CCTV/RTSP

Temuan: BB-21.

Files:

- Modify: `backend/handlers/cctv.go`
- Modify: `backend/handlers/cctv_test.go`
- Modify: `frontend/src/app/cctv/page.tsx`

Langkah:

1. Tambah test empty/null/valid/invalid untuk `site_id` dan `zone_id`.
2. Gunakan DTO tiga keadaan agar null dapat melepas relasi.
3. Normalisasi payload frontend; jangan kirim string kosong sebagai UUID.
4. Tambah validasi RTSP field-specific.
5. Pastikan password kosong saat edit berarti “tidak berubah”.

Done: edit biasa, RTSP, assignment, dan pelepasan site/zona berhasil dengan error yang presisi.

### Task 3.2 — Hapus asumsi checklist selalu sembilan

Temuan: BB-13, MT-03, MT-22.

Files:

- Modify: `backend/handlers/maintenance.go`
- Modify: `backend/handlers/maintenance_test.go`
- Modify: `frontend/src/components/FieldMaintenanceModal.tsx`
- Modify: model/request checklist terkait

Langkah:

1. Capture payload production untuk kasus tepat 9/9 tanpa memuat data sensitif.
2. Tambah test template 8, 9, 10 item; item wajib hilang/duplikat; satu item tidak OK.
3. Kirim identitas item/template pada payload.
4. Backend memuat template aktif untuk site CCTV dan memvalidasi kelengkapan.
5. Hitung `completed/failed` dari item wajib, bukan konstanta jumlah.
6. Pastikan respons, DB, dashboard, audit, dan notification memakai status sama.

Done: semua konfigurasi template valid dapat selesai; payload cacat ditolak `400`.

### Task 3.3 — Tegaskan assignment dan recurring schedule

Temuan: BB-01, BB-14, BB-15, MT-16, MT-17.

Files:

- Modify: `frontend/src/app/maintenance/page.tsx`
- Modify: `frontend/src/app/schedules/page.tsx`
- Modify: `backend/handlers/maintenance.go`
- Modify: `backend/handlers/schedule_runner.go`
- Modify: tests maintenance/schedule

Langkah:

1. Filter teknisi aktif dari dropdown reassign, tetapi pertahankan backend rejection.
2. Terapkan keputusan DG-01 pada status dan copy jadwal tanpa teknisi.
3. Tambah fake clock/test trigger untuk occurrence daily/weekly/monthly.
4. Pastikan generator idempoten dan tidak membuat log ganda.
5. Pindahkan pengiriman notifikasi occurrence ke pipeline standar pada Gelombang 4.

Done: assignment atomik, teknisi lama kehilangan akses, schedule occurrence tepat satu, dan label UI sesuai backend.

### Task 3.4 — Amankan stok suku cadang per kawasan

Temuan: BB-19, BB-20, MT-20.

Files:

- Add: migration `backend/migrations/<next>_spare_part_stocks.up.sql/.down.sql`
- Modify: `backend/models/spare_part.go`
- Modify: `backend/handlers/spare_part.go`
- Modify: `backend/handlers/spare_part_test.go`
- Modify: frontend spare parts dan maintenance usage

Langkah:

1. Implementasikan keputusan DG-03 dan migrasikan stok lama ke site default.
2. Enforce bahwa teknisi hanya mencatat penggunaan pada maintenance log miliknya dan site yang sama.
3. Gunakan transaksi dan row locking agar stok tidak negatif.
4. Tambah idempotency key pada pencatatan penggunaan.
5. Kirim low-stock event hanya saat crossing threshold.

Done: non-owner/cross-site ditolak, retry tidak mengurangi stok dua kali, dan transaksi paralel aman.

## Gelombang 4 — Notifikasi dan Push

### Task 4.1 — Perbaiki konfigurasi VAPID runtime

Temuan: BB-11, BB-12, MT-14.

Files:

- Add/modify: `backend/routes/push.go`, `backend/handlers/push.go`
- Modify: `backend/utils/push.go`
- Modify: `frontend/src/components/PushNotificationOptIn.tsx`
- Modify: `.env.example`, `backend/.env.docker.example`

Langkah:

1. Tambah endpoint public-key runtime yang hanya mengembalikan public key.
2. Validasi key base64url dan panjang sebelum `PushManager.subscribe`.
3. Tampilkan status tidak dikonfigurasi tanpa memanggil browser API.
4. Tambah startup/health check pasangan key.
5. Uji subscription idempoten, unsubscribe, user scope, click navigation, dan fallback in-app.

Done: push dapat diaktifkan di deployment dan kegagalannya tidak memengaruhi in-app.

### Task 4.2 — Satukan event delivery jadwal dan zona

Temuan: BB-15, BB-22, MT-01, MT-05, MT-13, MT-16, MT-17, MT-25.

Files:

- Modify: `backend/utils/notify.go`
- Modify: `backend/handlers/schedule_runner.go`
- Modify: `backend/handlers/zone.go`
- Modify: notification models/tests
- Optional add: transactional outbox migration/model/worker

Langkah:

1. Terapkan DG-02 dan DG-06.
2. Hitung delta assignment zona setelah commit.
3. Gunakan satu pipeline untuk persistensi, WebSocket, push, email, dan webhook.
4. Gunakan outbox/idempotency key agar retry tidak menggandakan notifikasi.
5. Tambah correlation ID dan delivery status per kanal.

Done: setiap event tepat satu kali secara logis, tersimpan saat user offline, dan kanal dapat diuji independen.

## Gelombang 5 — Laporan dan PDF

### Task 5.1 — Rapikan PDF maintenance

Temuan: BB-02, MT-06.

Files:

- Modify: `backend/handlers/report.go`
- Modify/add: report PDF tests dan fixture

Langkah:

1. Pastikan total lebar kolom sama dengan area konten.
2. Buat helper letterhead dengan koordinat eksplisit per baris.
3. Hitung tinggi dari semua cell dan ulangi header saat page break.
4. Tambah fixture teks panjang, tabel multi-page, dan karakter non-ASCII.
5. Render PDF menjadi gambar untuk visual regression.

### Task 5.2 — Rapikan report builder dan template

Temuan: BB-17, BB-18, MT-18.

Files:

- Modify: `backend/handlers/report_builder.go`
- Modify: `backend/handlers/report_builder_test.go`
- Modify: `frontend/src/app/laporan-kustom/page.tsx`

Langkah:

1. Beri bobot lebar kolom, wrapping, tinggi baris, dan orientasi dinamis.
2. Validasi schema template dan versioning di backend.
3. Tambah CTA eksplisit “Gunakan template”, preview, dan active state.
4. Reset state sebelum load template.
5. Uji PDF, Excel, CSV, site scope, ownership, dan admin-only.

Done Gelombang 5: kedua jenis PDF terbaca pada data panjang dan template menghasilkan konfigurasi identik setelah reload.

## Gelombang 6 — UX Operasional dan Aksesibilitas

### Task 6.1 — Dashboard teknisi dan scanner reusable

Temuan: BB-05, BB-06.

Files:

- Add: komponen scanner reusable di `frontend/src/components/`
- Modify: `frontend/src/app/cctv/page.tsx`
- Modify: `frontend/src/app/dashboard/page.tsx`

Perubahan: tampilkan nama pekerjaan, aset, lokasi, jadwal, status; tambah CTA scan yang menuju field-maintenance.

### Task 6.2 — Navigasi change-password tanpa flash

Temuan: BB-09.

Files:

- Modify: `frontend/src/proxy.ts`
- Modify: `frontend/src/context/AuthContext.tsx`
- Modify: `frontend/src/app/change-password/page.tsx`

Perubahan: guard sebelum render, logout pada forced mode, back/fallback untuk mode biasa, dan safe `returnTo`.

### Task 6.3 — Foto proporsional

Temuan: BB-07, MT-10.

Files:

- Modify: `backend/handlers/upload.go`
- Modify/add: upload/image tests
- Modify: avatar UI bila crop interaktif disetujui

Perubahan: fit-resize proporsional, EXIF orientation, uji portrait/landscape/square dan batas ukuran.

### Task 6.4 — Zona map, dark mode SLA, dan cleanup UI

Temuan: BB-23, BB-24, BB-25, MT-25.

Files:

- Modify: `frontend/src/app/sebaran-cctv/page.tsx`
- Modify: `frontend/src/app/sla/page.tsx`
- Modify: `frontend/src/app/globals.css` bila token tambahan diperlukan

Perubahan: dropdown zona memengaruhi seluruh derived state; gunakan token tema valid; hapus reload normal dan pertahankan retry error.

Done Gelombang 6: workflow teknisi lebih singkat, tidak ada flash/loop, gambar tidak terdistorsi, dan SLA memenuhi kontras.

## Gelombang 7 — Perbaiki Seluruh Manual Test dan Release Gate

Temuan: MT-01–MT-25 dan SYS-MT-01–10.

### Task 7.1 — Pecah suite yang tumpang tindih

- Pecah `manual-test-visual.md` menjadi snapshot/timeline dan monitor/stream health.
- Pecah `manual-test-frontend-ux-polish.md` menjadi monitor, kalender, dan performance.
- Pecah `manual-test-zone-sla.md` menjadi zona dan SLA.
- Tetapkan satu file owner per capability dan gunakan referensi silang.

### Task 7.2 — Lengkapi skenario yang hilang

- Calendar: boundary/pagination/scheduled date.
- Daily digest: catch-up/timezone/idempotensi/RBAC.
- Email/webhook/push: receiver sandbox dan delivery matrix.
- Field PDF/report: visual QA dan security.
- Geotag/health/PWA: evidence runtime dan negative case.
- Reassign/schedules/overdue: fake clock, ownership, cleanup.
- Multi-site/spare parts: dua site, concurrent/idempotency.
- Streaming/RTSP/visual: simulator dan playback nyata.
- Performance/SLA: fixture matematis dan visual light/dark.

### Task 7.3 — Tambahkan automation layer frontend

Rekomendasi:

- Tambah Playwright untuk direct URL RBAC, site switching, scanner fallback, template load, filter zona, dan tema SLA.
- Tambah test komponen/unit hanya bila logika state kompleks; jangan menggantikan E2E release acceptance.

### Task 7.4 — Full verification

```powershell
cd backend
go build ./...
go test ./... -count=1
go vet ./...

cd ../frontend
npm run lint
npm run build

cd ..
docker compose config --quiet
git diff --check
```

Runtime verification tambahan:

- E2E role matrix dan two-site matrix.
- QR rotation round-trip.
- MediaMTX external playback.
- Notification event-delivery matrix.
- PDF render-to-image review.
- Cleanup fixture dan audit log.

## Urutan PR yang Disarankan

| PR | Scope | Depends on |
|---|---|---|
| 1 | Test governance baseline + production provenance | — |
| 2 | QR lifecycle | PR 1 |
| 3 | MediaMTX production hardening | PR 1; paralel dengan PR 2 |
| 4 | Multi-site invariant CCTV/maintenance/report/map | PR 1 |
| 5 | RBAC report + split inventory API | PR 1 |
| 6 | CCTV DTO + dynamic checklist status | PR 4 |
| 7 | Spare stock per site + ownership/idempotency | PR 4–5 |
| 8 | Schedule/reassign semantics | PR 6 |
| 9 | VAPID runtime + notification pipeline/outbox | PR 8 |
| 10 | PDF maintenance/report builder/template UX | PR 4–5 |
| 11 | Dashboard/QR shortcut/password/photo/SLA UI | PR 2 dan PR 5 bila relevan |
| 12 | Rewrite 25 manual tests + full regression | seluruh PR fitur |

## Traceability Manual-test

| ID | Gelombang penutupan utama |
|---|---|
| MT-01 | 1B, 4.2, dan 7 — webhook sandbox serta event streaming. |
| MT-02 | 7 — kalender boundary, pagination, dan browser evidence. |
| MT-03 | 2.1, 3.2, dan 7 — site scope serta enforcement checklist. |
| MT-04 | 7 — struktur daily digest, timezone, catch-up, dan idempotensi. |
| MT-05 | 1B, 4.2, dan 7 — SMTP sandbox serta stream-down delivery. |
| MT-06 | 5.1 — security dan visual QA PDF lapangan. |
| MT-07 | 6 dan 7 — pecah suite UX serta browser evidence. |
| MT-08 | 7 — geotag permission, ownership, negative case, dan evidence. |
| MT-09 | 7 — fixture matematis health score dan boundary 30/60/90 hari. |
| MT-10 | 6.3 — resize proporsional dan regression image. |
| MT-11 | 1A, 2.2, dan 7 — QR/key lifecycle, RBAC, guard non-production. |
| MT-12 | 2.1 — invariant isolasi dua site. |
| MT-13 | 4.2 dan 7 — escalation 24/48/72, fake clock, anti-duplikasi. |
| MT-14 | 4.1 — VAPID runtime dan push E2E. |
| MT-15 | 7 — install/offline PWA pada perangkat dan URL yang benar. |
| MT-16 | 3.3 dan 4.2 — reassign, ownership, audit, dan notifikasi. |
| MT-17 | 3.3 dan 4.2 — recurring occurrence dan notifikasi. |
| MT-18 | 2.1, 2.2, dan 5.2 — site scope, RBAC, format, dan template. |
| MT-19 | 1B — kebijakan probe dan simulator RTSP/MediaMTX. |
| MT-20 | 2.1, 2.2, dan 3.4 — site stock, API split, ownership, idempotensi. |
| MT-21 | 1B — lifecycle streaming dan playback nyata. |
| MT-22 | 3.2 — state-transition matrix maintenance/CCTV. |
| MT-23 | 7 — fixture kinerja teknisi, formula, periode, dan browser evidence. |
| MT-24 | 1B dan 7 — pecah suite visual, snapshot, timeline, dan monitor. |
| MT-25 | 2.1, 4.2, 6.4, dan 7 — filter zona, SLA event, dark mode, dan suite split. |

## Definition of Done Keseluruhan

- Seluruh BB-01–BB-25 mempunyai regression evidence dan status akhir.
- Seluruh MT-01–MT-25 diperbarui; mandatory scenario tidak lagi memakai code inspection sebagai PASS.
- P0/P1 lulus pada artifact yang benar-benar dideploy.
- Tidak ada kebocoran lintas site atau bypass admin API.
- QR, streaming, maintenance, stok, dan notifikasi memenuhi acceptance idempotensi/integritas.
- PDF lolos visual QA; UI kritis lolos desktop/mobile dan light/dark.
- Build, test, vet, lint, compose validation, dan diff check lulus.
