# Indeks Konsolidasi Seluruh Temuan

Tanggal: 19 Agustus 2026

Cakupan sumber:

- `BLACKBOX-FINDINGS-2026-08-19.md`: 25 temuan black-box production.
- `MANUAL-TEST-AUDIT-SUMMARY.md`: audit kualitas dan isi 25 dokumen manual test.

Dokumen sumber tetap dipisahkan. Indeks ini tidak melebur nomor temuan; fungsinya menunjukkan irisan, dependensi, dan urutan penanganan agar satu perbaikan tidak dikerjakan berulang kali.

## Ringkasan Eksekutif

Terdapat empat release blocker utama:

1. Rotasi QR menghasilkan identitas yang tidak dapat dipakai secara konsisten.
2. MediaMTX/streaming production belum menghasilkan playback nyata.
3. Isolasi kawasan belum diterapkan konsisten pada CCTV, maintenance, inventaris, laporan, dan pengujian.
4. Bukti manual test belum dapat dipercaya sebagai release acceptance karena banyak status lulus tidak berasal dari eksekusi runtime.

Selain itu, akses admin-only, penyelesaian status maintenance, integritas stok, konfigurasi push, dan notifikasi jadwal/zona harus diselesaikan sebelum rilis dinyatakan stabil.

## Kelompok Pekerjaan dan Deduplicasi

| Workstream | Temuan black-box | Temuan manual-test | Akar kebutuhan bersama | Prioritas |
|---|---|---|---|---|
| WS-01 QR lifecycle | BB-10 | MT-11 | Satu kontrak rotasi, validasi payload aktif, cache invalidation, dan recovery label. | P0 |
| WS-02 Streaming/MediaMTX | BB-08 | MT-01, MT-05, MT-19, MT-21, MT-24, SYS-MT-01 | Environment reachable, konfigurasi ICE, simulator, serta health berlapis dari RTSP sampai frame. | P0 |
| WS-03 Isolasi multi-site | BB-03, BB-04, BB-19, BB-23 | MT-03, MT-12, MT-18, MT-20, MT-25, SYS-MT-04 | Site context terpusat dan invariant pada list/detail/write/export/inventory/map. | P1 blocker |
| WS-04 RBAC dan route guard | BB-16, BB-20 | MT-11, MT-18, MT-20, MT-24, SYS-MT-03 | Backend authorization sebagai batas utama, frontend guard sebagai defense-in-depth, serta verifikasi deployment. | P1 |
| WS-05 Integritas maintenance | BB-01, BB-13, BB-14, BB-21 | MT-03, MT-16, MT-17, MT-22, SYS-MT-05 | DTO konsisten, template dinamis, state machine tugas, dan semantik assignment yang eksplisit. | P1 |
| WS-06 Notifikasi multikanal | BB-11, BB-12, BB-15, BB-22 | MT-01, MT-05, MT-13, MT-14, MT-16, MT-17, MT-20, MT-25, SYS-MT-02 | Jalur event tunggal, outbox/idempotensi, konfigurasi VAPID, dan observability per kanal. | P1 |
| WS-07 Integritas stok | BB-19, BB-20 | MT-20 | Model stok per site, ownership tugas, transaksi/locking, dan idempotency key. | P1 |
| WS-08 PDF dan report builder | BB-02, BB-17, BB-18 | MT-06, MT-18, SYS-MT-06 | Layout engine berbobot, wrapping, page header, schema template, visual regression PDF. | P2 |
| WS-09 UX teknisi dan navigasi | BB-05, BB-06, BB-09 | MT-07, MT-08, MT-16 | Informasi tugas lengkap, scanner reusable, forced-password guard sebelum render. | P1/P2 |
| WS-10 Filter dan aksesibilitas UI | BB-07, BB-23, BB-24, BB-25 | MT-07, MT-10, MT-25 | Resize proporsional, filter zona konsisten, token tema valid, keputusan UI sinkron dengan source/changelog. | P2 |
| WS-11 Tata kelola pengujian | seluruh BB | seluruh MT, SYS-MT-07–10 | Status test formal, fixture non-production, evidence, cleanup, traceability, dan release gate otomatis. | P1 proses |

## Ketergantungan Utama

```text
Baseline deployment + evidence
├── QR lifecycle (dapat berjalan paralel)
├── MediaMTX/streaming (dapat berjalan paralel)
└── Kontrak otorisasi dan multi-site
    ├── CCTV/maintenance filtering
    ├── Inventaris per site + ownership
    └── Report/export per site

Kontrak maintenance/template
├── Status completed/failed
├── Jadwal tanpa teknisi
└── Notifikasi jadwal/reassign

Konfigurasi VAPID + event pipeline
├── Push opt-in
├── Notifikasi jadwal
├── Notifikasi zona
└── Matriks in-app/WebSocket/push/email/webhook

Semua workstream
└── Manual test berbukti + regression/release gate
```

## Temuan yang Menunjukkan Drift Source–Deployment

| Temuan | Kondisi source saat audit | Tindakan wajib |
|---|---|---|
| BB-16 laporan kustom | Proxy dan API sudah admin-only. | Cocokkan commit SHA/image digest, pastikan proxy aktif, lalu uji direct URL dan API. |
| BB-20 halaman suku cadang | Proxy sudah admin-only; API read masih sengaja terbuka. | Bedakan drift frontend dari keputusan kontrak API teknisi. |
| BB-07 foto profil | UI sudah `object-cover`, tetapi backend masih merusak rasio saat resize. | Uji asset baru vs asset lama; jangan menganggap perubahan class saja menyelesaikan masalah. |
| BB-25 tombol SLA | Changelog menyatakan tombol dihapus, source saat audit masih merender tombol. | Sinkronkan source, build artifact, changelog, dan acceptance UI. |
| MT-18 report builder | Hasil manual production menyatakan teknisi dapat masuk, berlawanan dengan source. | Jadikan build provenance bagian wajib evidence. |

## Temuan yang Tidak Boleh Digabung Menjadi Satu Bug

- BB-11 adalah kegagalan konfigurasi/validasi push; BB-12 menegaskan in-app harus diuji terpisah.
- BB-08 membedakan API MediaMTX, path registration, source ready, WHEP signaling, ICE, dan playback frame. Status “path registered” bukan berarti video berjalan.
- Evidence Task 1B 2026-08-20 menutup sub-gate source dan staging terisolasi untuk WS-02; gate production/public/mobile masih `NOT RUN` dan WS-02 belum ditutup penuh.
- BB-03 mencakup site context/filter maintenance dan masalah sorting `/sites`; keduanya boleh satu workstream tetapi harus memiliki acceptance terpisah.
- BB-13 mencakup cacat konstanta sembilan dan kemungkinan masalah payload production tepat 9/9; capture production tetap diperlukan.
- BB-16/BB-20 membedakan akses shell halaman, guard frontend, dan authorization API. Redirect UI tidak membuktikan API aman.

## Release Gate yang Disarankan

Rilis remediasi tidak boleh dinyatakan selesai sebelum:

- P0 dan P1 wajib berstatus `PASS`; tidak ada mandatory scenario `FAIL`, `BLOCKED`, atau `NOT RUN`.
- Isolasi dua site terbukti pada list, detail, mutation, export, map, dan inventaris.
- Admin/teknisi/anonymous diuji melalui sidebar, direct URL, dan API langsung.
- QR baru berfungsi dan QR lama ditolak setelah rotasi.
- Playback MediaMTX terbukti menerima frame dari jaringan eksternal.
- Status maintenance, stok, dan notifikasi terbukti idempoten.
- PDF dirender ke gambar dan lolos visual QA.
- Seluruh manual-test menyimpan environment, build SHA, waktu, evidence, fixture, dan cleanup.

Rencana implementasi terurut terdapat pada `docs/plans/FINDINGS-REMEDIATION-PLAN.md`.
