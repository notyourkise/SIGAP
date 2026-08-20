# Ringkasan Audit Dokumen Manual Test

Tanggal audit: 19 Agustus 2026

Cakupan: 25 berkas `docs/manual-test-*.md`

Jenis audit: konsistensi skenario, kecukupan bukti, keamanan eksekusi, kesesuaian dengan implementasi, dan kelayakan sebagai release acceptance.

## Kesimpulan

Dokumentasi manual test saat ini belum cukup kuat dijadikan bukti bahwa rilis layak dipromosikan. Banyak hasil `✅` berasal dari inspeksi kode, bukan eksekusi black-box; beberapa kesimpulan akhir tetap menyatakan lulus walaupun skenario wajib gagal, belum dijalankan, atau terblokir. Sejumlah prosedur juga menggunakan data produksi atau memutasi database tanpa guard, backup, dan cleanup.

Prioritas tertinggi adalah isolasi data multi-site, otorisasi laporan dan suku cadang, konfigurasi VAPID, integritas stok, sinkronisasi status pemeliharaan, distorsi kompresi gambar, dan verifikasi MediaMTX/streaming secara nyata.

## Definisi Severity

- **P1 — kritis/tinggi:** klaim kelulusan tidak sah atau terdapat risiko keamanan, kebocoran lintas kawasan, integritas data, dan kegagalan fungsi operasional utama.
- **P2 — sedang:** bukti atau cakupan material belum lengkap, tetapi belum menunjukkan dampak keamanan langsung.
- **P3 — rendah:** masalah dokumentasi minor yang tidak mengubah kesimpulan fungsional.

## Register Temuan per File

### MT-01 — API Key dan Webhook

- File: `manual-test-api-webhook.md`
- Severity: **P1**
- Temuan: penggunaan `webhook.site` berisiko mengirim payload produksi ke pihak ketiga; event stream-down belum diuji; kriteria retry 1/5/25 detik tidak selaras dengan bukti percobaan.
- Solusi: gunakan receiver lokal/sandbox, sanitasi payload, catat timestamp setiap attempt, verifikasi HMAC, dan buat fixture untuk semua event.
- Exit criteria: seluruh event terkirim ke receiver internal, signature valid, retry sesuai kebijakan, dan tidak ada data produksi keluar.

### MT-02 — Kalender

- File: `manual-test-calendar.md`
- Severity: **P2**
- Temuan: klaim 37/37 mencampur inspeksi kode dengan uji browser; filter API memakai `created_at` sementara kalender mengelompokkan `scheduled_date`; limit 100 berpotensi menghilangkan tugas.
- Solusi: jalankan browser test nyata, uji boundary bulan/tahun, paginasi lebih dari 100 tugas, serta beda bulan antara tanggal dibuat dan tanggal jadwal.
- Exit criteria: screenshot/network trace tersedia dan seluruh tugas tampil pada bulan `scheduled_date` tanpa terpotong paginasi.

### MT-03 — Template Checklist

- File: `manual-test-checklist-template.md`
- Severity: **P2**
- Temuan: klaim 21/21 menggunakan API produksi dan inspeksi kode; fixture/UUID tertinggal; atribut `wajib` belum terbukti dipaksakan; scope kawasan tidak diuji.
- Solusi: gunakan fixture disposable, cleanup terotomasi, enforcement item wajib, dan matriks dua site.
- Exit criteria: item wajib yang kosong ditolak, template aktif/nonaktif konsisten, isolasi site terbukti, dan fixture bersih.

### MT-04 — Daily Digest

- File: `manual-test-daily-digest.md`
- Severity: **P2**
- Temuan: hasil hanya berupa centang tanpa request, response, timestamp, dan rekonsiliasi angka; catch-up, idempotensi, timezone, RBAC, serta pagination belum diuji.
- Solusi: gunakan format skenario standar dan fixture lebih dari 10 digest dengan expected calculation independen.
- Exit criteria: tepat satu digest per tanggal WITA, catch-up tidak menggandakan data, angka cocok dengan sumber, 401/403 dan page 2 terbukti.

### MT-05 — Email

- File: `manual-test-email.md`
- Severity: **P1**
- Temuan: penerimaan dan format email dinyatakan lulus tanpa akses inbox; stream-down belum diuji; SMTP produksi dipakai.
- Solusi: gunakan SMTP sandbox, simpan message ID/body, uji preferensi per jenis, penerima, retry, dan kegagalan SMTP.
- Exit criteria: pesan diterima penerima tepat, opt-out tidak menerima, template valid, dan kegagalan SMTP tidak menggagalkan operasi utama.

### MT-06 — PDF Pengecekan Lapangan

- File: `manual-test-field-maintenance-pdf.md`
- Severity: **P2**
- Temuan: tidak ada bukti layout, overflow, foto, font, tabel panjang, filename, MIME, RBAC, dan inspeksi visual PDF.
- Solusi: buat fixture pendek/panjang, render setiap halaman menjadi gambar, validasi `%PDF`, MIME, filename, owner/non-owner, dan UUID invalid.
- Exit criteria: tidak ada overlap/cutoff, foto dan tabel terbaca, serta kontrol akses terbukti dari response nyata.

### MT-07 — Frontend UX Polish

- File: `manual-test-frontend-ux-polish.md`
- Severity: **P2**
- Temuan: monitor, kalender, dan kinerja teknisi digabung; semua UI dicentang lulus tanpa bukti browser; full test suite sempat timeout.
- Solusi: pecah menjadi tiga suite dan simpan evidence DevTools/screenshot untuk lazy loading, modal, serta request periode.
- Exit criteria: tiap suite mempunyai environment, hasil independen, evidence runtime, dan test suite stabil.

### MT-08 — Geotag

- File: `manual-test-geotag.md`
- Severity: **P2**
- Temuan: tidak ada bukti koordinat, permission state, atau peta; ownership record dan manipulasi koordinat non-owner belum diuji.
- Solusi: gunakan koordinat simulasi, uji denied/timeout, pasangan koordinat invalid, detail milik sendiri dan milik pengguna lain.
- Exit criteria: koordinat tersimpan presisi, izin ditolak tetap memiliki fallback, input invalid ditolak, dan non-owner tidak dapat membaca record.

### MT-09 — Health Score

- File: `manual-test-health-score.md`
- Severity: **P2**
- Temuan: perhitungan kompleks tidak dilengkapi tabel input-output; manipulasi histori berisiko mengubah produksi; boundary 30/60/90 hari dan cleanup tidak terbukti.
- Solusi: gunakan fixture deterministik, clock terkontrol, dan expected calculation independen per faktor/boundary.
- Exit criteria: skor API cocok dengan kalkulasi, clamp 0–100 dan idempotensi scheduler terbukti, lalu fixture dibersihkan.

### MT-10 — Kompresi Gambar

- File: `manual-test-image-compression.md`
- Severity: **P1**
- Temuan: foto 4000×3000 menjadi 1920×1920 sehingga rasio rusak; unit test timeout, tetapi hasil tetap dianggap bukan kegagalan fungsional.
- Solusi: gunakan fit-resize proporsional dan uji landscape, portrait, square, EXIF rotation, serta batas ukuran.
- Exit criteria: rasio asli dipertahankan, sisi terpanjang maksimal 1920 px, thumbnail sesuai spesifikasi, dan test stabil.

### MT-11 — Integrasi Infrastruktur

- File: `manual-test-infra-integration.md`
- Severity: **P1**
- Status: sub-gate lifecycle QR Task 1A **PASS 2026-08-20**; acceptance agregat MT-11 tetap `NOT RUN` untuk API key, webhook, dan perluasan role-negative.
- Temuan: rotasi QR, regenerasi key, dan penghapusan webhook log bersifat destruktif tanpa guard non-production, backup, rollback, dan cleanup; beberapa RBAC hanya opsional.
- Solusi: wajibkan test environment, fixture khusus, recovery procedure, dan matriks 401/403 untuk seluruh endpoint mutasi.
- Exit criteria: kredensial/QR lama ditolak, pengganti berfungsi, cleanup hanya menyentuh fixture, audit tercatat, dan seluruh role negatif lulus.
- Evidence QR: `MT-11-QR-RUNTIME-EVIDENCE-2026-08-20.md`.

### MT-12 — Multi-site

- File: `manual-test-multi-site.md`
- Severity: **P1 — release blocker**
- Temuan: isolasi kawasan gagal pada CCTV, pemeliharaan, dan zona, tetapi banyak status dan kriteria tetap dicentang lulus.
- Solusi: terapkan site scope terpusat pada list/detail/write/export dan uji dua site × dua user × seluruh endpoint.
- Exit criteria: konteks site A tidak dapat membaca atau memutasi data site B, invalid site ditolak, dan tidak ada endpoint yang melewati scope.

### MT-13 — Overdue Escalation

- File: `manual-test-overdue-escalation.md`
- Severity: **P1**
- Temuan: skenario satu level sudah usang terhadap kebijakan 24/48/72 jam; hasil produksi dan unit test dicampur; contoh PowerShell salah dan langsung memutasi DB.
- Solusi: tulis matriks tiga level, gunakan fake clock/test trigger, serta larang SQL produksi tanpa backup dan cleanup.
- Exit criteria: level 1→2→3, anti-duplikasi, reset reschedule, penerima, email/webhook, dan status terminal terbukti end-to-end.

### MT-14 — Push Notification

- File: `manual-test-push-notification.md`
- Severity: **P1**
- Temuan: subscribe gagal karena `applicationServerKey` tidak valid, tetapi skenario dependen tetap ditandai lulus.
- Solusi: tandai skenario dependen `BLOCKED`, validasi pasangan VAPID saat build/startup, kemudian ulangi pada browser nyata.
- Exit criteria: subscribe idempoten, push diterima, klik mengarah benar, scope user benar, unsubscribe menghentikan push, dan in-app tetap berfungsi.

### MT-15 — PWA

- File: `manual-test-pwa.md`
- Severity: **P2**
- Temuan: perilaku install/offline banyak disimpulkan dari kode; Android diarahkan ke `https://localhost`; ekspektasi cache v1/v2 tidak konsisten.
- Solusi: gunakan URL HTTPS yang dapat diakses perangkat, Chrome DevTools/Lighthouse, dan pisahkan code review dari runtime test.
- Exit criteria: instalasi nyata berhasil, offline shell termuat, API offline memiliki fallback, cache lama terhapus, dan skor Lighthouse dicatat.

### MT-16 — Reassign

- File: `manual-test-reassign.md`
- Severity: **P2**
- Temuan: dokumen justru memilih teknisi aktif agar backend menolak; UX penghapusan opsi teknisi aktif, cleanup, dan hilangnya akses teknisi lama belum diuji penuh.
- Solusi: tambahkan assertion dropdown, akses API teknisi lama, atomicity, audit, dan notifikasi dua pihak.
- Exit criteria: teknisi aktif tidak menjadi kandidat, reassign atomik, teknisi lama tidak dapat menyelesaikan tugas, dan notifikasi tidak ganda.

### MT-17 — Jadwal Berulang

- File: `manual-test-recurring-schedules.md`
- Severity: **P1**
- Temuan: generator/notifikasi belum diuji end-to-end; penomoran, username, URL, dan contoh PowerShell bermasalah; makna jadwal tanpa teknisi ambigu.
- Solusi: rapikan dokumen, gunakan fake clock/test trigger, dan tetapkan semantik jadwal tanpa teknisi sebagai kontrak produk.
- Exit criteria: daily/weekly/monthly menghasilkan satu log tepat waktu, inactive skip, tidak duplikat, teknisi menerima notifikasi, dan jadwal tanpa teknisi memiliki status/label yang disetujui.

### MT-18 — Report Builder

- File: `manual-test-report-builder.md`
- Severity: **P1**
- Temuan: akses teknisi ke halaman dilaporkan terbuka; file hanya diperiksa dari ekstensi/signature, tanpa QA layout PDF, tipe Excel, encoding CSV, ownership template, atau site scope.
- Solusi: enforcement admin di frontend dan backend, golden/schema checks, visual PDF QA, ownership template, dan site-scoped export.
- Exit criteria: teknisi diarahkan keluar dan API 403, format valid, layout rapi, filter/kolom benar, template serta data tidak bocor lintas user/site.

### MT-19 — RTSP Live Status

- File: `manual-test-rtsp-live-status.md`
- Severity: **P1**
- Temuan: NVR tidak reachable sehingga UI/live behavior belum terbukti; kebijakan probe semua stream bertentangan dengan aturan Metro-E langsung dan GSM pasif.
- Solusi: samakan kebijakan jaringan, gunakan simulator RTSP/MediaMTX, dan uji event status nyata.
- Exit criteria: Metro-E dan GSM mengikuti jalur masing-masing, timestamp/status berubah dari stimulus nyata, WebSocket diterima, dan counter UI cocok.

### MT-20 — Suku Cadang

- File: `manual-test-spare-parts.md`
- Severity: **P1**
- Temuan: ownership tugas dan idempotensi retry tidak diuji; stok berpotensi dikurangi oleh non-owner atau berkurang ganda.
- Solusi: enforce ownership di backend, gunakan transaksi/locking dan idempotency key, uji concurrent usage, rollback, serta site scope.
- Exit criteria: non-owner ditolak, retry tidak mengurangi stok lagi, stok tidak negatif, dan event low-stock hanya satu kali saat melewati ambang.

### MT-21 — Streaming

- File: `manual-test-streaming.md`
- Severity: **P1**
- Temuan: stop stream gagal 502, video tidak tampil, tetapi start-all dianggap sukses hanya dari registrasi path; dokumen memuat topologi/IP dan username NVR nyata.
- Solusi: redaksi data sensitif, gunakan simulator/environment reachable, dan bedakan status `registered`, `ready`, serta `playing`.
- Exit criteria: start, readiness, playback, stop, SD/HD, reconnect, bulk action, dan role enforcement lulus tanpa 5xx.

### MT-22 — Sinkronisasi CCTV

- File: `manual-test-sync-cctv.md`
- Severity: **P1**
- Temuan: perubahan status pernah gagal dan beberapa wizard tidak dijalankan, tetapi kesimpulan tetap menyatakan seluruh skenario lulus.
- Solusi: pertahankan status `FAILED/BLOCKED` sampai pengujian ulang; gunakan state-transition matrix status awal × hasil checklist × jalur penyelesaian.
- Exit criteria: 9/9 OK menghasilkan `completed`, satu masalah menghasilkan tindak lanjut, `nonaktif` tidak ditimpa, serta UI/API/audit konsisten.

### MT-23 — Kinerja Teknisi

- File: `manual-test-technician-performance.md`
- Severity: **P2**
- Temuan: hasil berupa centang ringkas; chart/sort/empty state dinilai dari kode; formula, periode 90/365 hari, boundary tanggal, dan parameter invalid belum diuji.
- Solusi: buat fixture per teknisi dengan hasil matematis yang diketahui dan matriks semua periode.
- Exit criteria: metrik cocok dengan perhitungan independen, boundary benar, invalid range ditolak, RBAC lulus, dan UI terbukti di browser.

### MT-24 — Visual, Snapshot, dan Monitor

- File: `manual-test-visual.md`
- Severity: **P2**
- Temuan: dua suite besar digabung; hook `maintenance_completed` belum terimplementasi; banyak UI belum dijalankan; ekspektasi role ambigu.
- Solusi: pecah suite, tetapkan matriks role, gunakan fixture JPEG/timeline dan simulator stream.
- Exit criteria: empat tipe event timeline muncul, rotasi 10 snapshot terbukti, role konsisten, dan monitor/fullscreen/pop-out diuji nyata.

### MT-25 — Zona dan SLA

- File: `manual-test-zone-sla.md`
- Severity: **P1**
- Temuan: filter zona `/sebaran-cctv` tidak ada walau menjadi kriteria wajib; SLA warning belum diuji nyata; ranking/violation tidak memiliki fixture.
- Solusi: pisahkan suite zona dan SLA, siapkan fixture lengkap, serta uji API, UI, WebSocket, dark mode, dan empty state.
- Exit criteria: filter mengubah marker/ringkasan, warning diterima tepat sekali, ranking/violation benar, RBAC lulus, dan tema terang/gelap terbaca.

## Temuan Sistemik

| ID | Tema | File terkait | Solusi lintas modul |
|---|---|---|---|
| SYS-MT-01 | MediaMTX/streaming tidak dapat diverifikasi | Streaming, RTSP, visual, UX, email, webhook | Sediakan environment/simulator yang reachable dan event-delivery matrix dari sumber stream sampai seluruh kanal. |
| SYS-MT-02 | Notifikasi tidak diuji sebagai satu rantai | Push, email, reassign, schedules, overdue, SLA, spare parts | Buat matriks event × penerima × in-app × WebSocket × push × email × webhook beserta correlation ID. |
| SYS-MT-03 | Guard UI dianggap setara otorisasi | Report, spare parts, schedules, visual, infra, checklist | Uji navigasi, URL langsung, dan API langsung secara terpisah untuk anonymous/admin/teknisi. |
| SYS-MT-04 | Isolasi kawasan tidak menjadi invariant | Multi-site, calendar, checklist, spare parts, SLA, report | Terapkan site scope terpusat dan contract test untuk list/detail/write/export. |
| SYS-MT-05 | Transisi status tidak terpadu | Sync CCTV, reassign, schedules, health score, overdue | Tetapkan state machine tunggal dan test semua transisi serta side effect. |
| SYS-MT-06 | PDF hanya diuji dari signature | Field PDF, report builder | Tambahkan render-to-image dan visual regression untuk data panjang, foto, wrapping, header/footer. |
| SYS-MT-07 | Pengujian mengubah produksi | Calendar, checklist, health, infra, overdue, schedules, spare parts | Wajibkan environment marker, fixture prefix, backup/rollback, dan cleanup terverifikasi. |
| SYS-MT-08 | Inspeksi kode dicatat sebagai PASS | PWA, RTSP, visual, schedules, SLA, spare parts, email | Gunakan status `PASS`, `FAIL`, `BLOCKED`, `NOT RUN`; code review hanya dicatat sebagai evidence pendukung. |
| SYS-MT-09 | Kesimpulan bertentangan dengan detail | Push, sync CCTV, compression, multi-site, streaming, report, SLA | Hitung kesimpulan otomatis dari skenario wajib; satu mandatory FAIL/BLOCKED menggagalkan suite. |
| SYS-MT-10 | Suite tumpang tindih dan stale | Calendar/UX, performance/UX, streaming/RTSP/visual | Tetapkan satu owner file per capability dan gunakan referensi silang, bukan duplikasi skenario. |

## Standar Perbaikan Semua Manual Test

Setiap file manual test harus memuat metadata versi/build, environment non-production, identitas tester, waktu eksekusi, fixture dan cleanup, request/response atau screenshot, expected vs actual, serta status `PASS`, `FAIL`, `BLOCKED`, atau `NOT RUN`. Kesimpulan suite harus dihitung hanya dari eksekusi runtime; inspeksi kode dan unit test dicatat sebagai evidence tambahan.

Dokumen ini merangkum audit dan solusi. Urutan implementasi lintas temuan terdapat pada `docs/plans/FINDINGS-REMEDIATION-PLAN.md`.
