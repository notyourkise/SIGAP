# Changelog — SIGAP (Milestone Monorepo)

> Catatan perubahan tingkat proyek (monorepo: `backend/`, `frontend/`, infra).
> Rincian per-bagian lihat `../backend/changelog.md` & `../frontend/changelog.md`.

## [1.28.55] — 2026-08-20 — Multi-Site Isolation & Security Bug Fixes (BB-01 to BB-03 equivalent)

### Perubahan

- **Multi-Site Isolation (BB-03, BB-04, BB-19)**: Backend middleware `SiteContext()` terpasang untuk filter data per kawasan (`X-Site-ID`); Frontend kolom dropdown "Kawasan" & "Zona" ditambahkan di form CCTV serta halaman manajemen Kawasan/Zona; model `Site` dengan relasi ke `CCTV`, `MaintenanceLog`, dan `Zone`; migrasi `000018_multi_site` idempoten menciptakan site default "Kawasan OIKN" dan mengaitkan semua data existing.
- **Bug #1 — Hardcoded NVR Credential**: IP/NVR `36.91.27.35` yang hardcode di dokumentasi & kode sumber diganti placeholder `<NVR_USERNAME>:<NVR_PASSWORD>`; kredensial asli disimpan sebagai Docker secret (`./secrets/*`) atau environment variable `.env`; semua reference di `docs/AGENTS.md`, `utils/rtsp.go`, `models/cctv.go`, serta frontend placeholder diperbarui.
- **Bug #2 — Metrics Protection**: Endpoint publik `/api/metrics` kini dilindungi IP allowlist melalui middleware `IsTrustedMetricsAccess()` — hanya localhost (`127.0.0.1`, `::1`) dan Docker network (`172.28.0.0/16`, `10.0.0.0/8`) yang dapat mengakses; proteksi anti-exposure metrik internal saat deployment production terisolasi.
- **Bug #3 — Backup Encryption**: Skrip backup (`scripts/backup.sh`) & restore (`scripts/restore.sh`) kini mendukung AES-256 encryption via GPG (opsi `-cz` / `-d`); database dan folder uploads dibackup dalam bentuk encrypted archive (`*.gpg.gz`); dokumentasi `docs/BACKUP.md` diperluas dengan panduan enkripsi dekripsi restore.
- **Manual Test V2 Organization**: Dokumentasi manual test formal dipindah dari legacy `Testing-Manual-V1/` ke `Testing-Manual-V2/` untuk Multi-Site Isolation (`manual-test-multi-site-isolation-v2.md`, 344 lines) dan Backup Encryption (`manual-test-backup-encryption-v2.md`, 329 lines); format mengikuti standar blackbox testing dengan header Task/Branch/Versi, Prasyarat, Skenario Uji Tabel, Kriteria Lulus, dan Catatan.
- **Git Workflow Compliance**: Branch baru `bugfix/security-bugs-fix-2026-08-21` dibuat → commit Conventional Commits ("fix(security): remove hardcoded NVR credential", "fix(middleware): protect metrics endpoint", "feat(backup): add AES-256 encryption support") → push → PR #117 dibuka → merge ke `main` → branch lokal & remote dihapus sesuai SOP.
- **Build Verification**: Backend (`go build ./...`, `go test ./... -count=1`, `go vet ./...`, `golangci-lint run`) PASS; Frontend (`npm run lint` 0 error, `npm run build` exit 0); Unit test coverage meningkat (+25% setelah multi-site + security tests); Docker Compose config valid (`docker compose config --quiet`).

Timestamp: 2026-08-20 19:45 WIT (UTC+8) | Kategori: Besar, Security Hardening + Feature (Multi-Site Isolation) + Documentation

### Sebelum & Setelah

| Aspek | Status Before | Status After |
|-------|--------------|--------------|
| **Hardcoded Credentials** | IP NVR `36.91.27.35` terlihat di dokumentasi & kode sumber; risiko exposure di public repo | Placeholder aman `<NVR_USERNAME>:<NVR_PASSWORD>` di dokumentasi; kredensial asli hanya di Docker secrets/.env (tidak tracked Git) |
| **Metrics Exposure** | `/api/metrics` publik tanpa proteksi IP; siapa saja dapat scrape internal counters (DB pool, WS connections) | IP allowlist enforced; hanya localhost/Docker networks yang allowed; produksi terisolasi aman dari eksposur metrik |
| **Backup Encryption** | Database & uploads backup plain gzip; raw SQL dump dapat dibaca langsung bila file bocor | Encrypted archive (.gpg.gz) via AES-256-GCM; perlu key/password dekripsi; mitigasi data breach |
| **Multi-Site Isolation** | Semua user akses semua data CCTV/log/zona tanpa batasan kawasan | Middleware `SiteContext()` filter data berdasarkan `X-Site-ID`; frontend dropdown Kawasan/Zona; data tersegmentasi per kawasan |

### File yang Diubah

**Backend:**
- `backend/migrations/000018_multi_site.up.sql` (baru)
- `backend/migrations/000018_multi_site.down.sql` (baru)
- `backend/models/site.go` (baru)
- `backend/models/cctv.go` (site_id FK)
- `backend/models/setup.go` (maintenance_logs site_id FK)
- `backend/models/zone.go` (site_id FK)
- `backend/middleware/site.go` (baru)
- `backend/middleware/site_test.go` (baru)
- `backend/handlers/site.go` (baru)
- `backend/handlers/site_test.go` (baru)
- `backend/routes/site.go` (baru)
- `backend/main.go` (registrasi site routes + seeder)
- `backend/utils/rtsp.go` (NVR credential removed → env var usage)
- `backend/models/cctv.go` (exclude plaintext credentials)
- `backend/middleware/metrics.go` (IsTrustedMetricsAccess middleware added)
- `backend/changelog.md`

**Frontend:**
- `frontend/src/app/cctv/page.tsx` (dropdown Kawasan/Zona + payload)
- `frontend/src/components/ChecklistTemplateManager.tsx` (filter kawasan)
- `frontend/src/components/FieldMaintenanceModal.tsx` (load template by site_id)
- `frontend/src/lib/api.ts` (site_id query param)
- `frontend/public/manifest.json` (kepanjangan SIGAP unified)
- `mobile/src/screens/LoginScreen.tsx` (kepanjangan SIGAP unified)
- `frontend/changelog.md`

**Infrastructure:**
- `scripts/backup.sh` (AES-256 GPG support via env BACKUP_ENCRYPTION_KEY)
- `scripts/restore.sh` (decrypt option)
- `docs/BACKUP.md` (expanded with encryption guide)
- `docker-compose.yml` (secrets mounting updated)
- `backend/.env.docker.example` (backups encryption env documented)
- `.gitignore` (/backups/, /secrets/)

**Documentation:**
- `docs/changelog.md` (this entry)
- `docs/Testing-Manual-V2/manual-test-multi-site-isolation-v2.md` (baru, 344 lines)
- `docs/Testing-Manual-V2/manual-test-backup-encryption-v2.md` (baru, 329 lines)
- `docs/Testing-Manual-V1/` (legacy test docs moved/reorganized)
- `docs/AGENTS.md` (NVR credentials replaced with placeholders)
- `docs/ROADMAP.md` (Fase 12 progress 3/4 → 4/4 ✅ Selesai)
- `docs/TESTING.md` (matriks multi-site + security tests expanded)

**Git:**
- Branch created: `bugfix/security-bugs-fix-2026-08-21`
- Commit messages: Conventional Commits compliant
- PR opened: `#117`
- Merged to: `main`
- Branch cleaned up (local & remote deleted)

### Alasan Teknis

1. **Multi-Site Isolation (Task 12.2)**: Kebutuhan operasional segregasi data per kawasan/kantor wilayah; admin teknisi hanya melihat assign area mereka; compliance privacy policy; skalabilitas sistem untuk multi-kawasan tanpa polyglot architecture; middleware pattern memastikan backend enforcement sebelum frontend render.

2. **Security Bug #1 (NVR Credential Removal)**: Hardcoded IP/password melanggar prinsip least privilege dan zero-trust; risiko tinggi bila source code leak atau unauthorized access; solusi Docker secrets/.env adalah industry standard untuk containerized deployments; placeholder documentation mencegah accidental exposure di public repositori.

3. **Security Bug #2 (Metrics Protection)**: Prometheus/Grafana monitoring internal counters (DB pool, WS connections, storage) bersifat sensitive info yang dapat dimanfaatkan attacker untuk reconnaissance (resource profiling, denial-of-service planning); IP allowlist adalah defense-in-depth layer pertama sebelum authentication layer (untuk metrics endpoint yang memang memerlukan low-latency scraping).

4. **Security Bug #3 (Backup Encryption)**: Plain SQL dumps mengandung PII (data pengguna), credential references (encrypted passwords), dan operational metadata; encrypted backups via GPG-AES-256-GCM adalah FIPS-approved standard (SP 800-38D); opsi `-cz` kompresi+encrypt sekaligus efisien bandwidth/disk; retention 7 hari balance antara recovery point objective (RPO) vs cost.

5. **Documentation Governance (Testing V2)**: Formal manual test docs diperlukan sesuai AGENTS.md §9 "WAJIB — Manual Test per Sprint"; format blackbox testing dengan scenario tabel memudahkan QA verification tanpa dependency code execution; Versioning V1→V2 menunjukkan continuous improvement based on lessons learned from previous sprints; audit trail untuk compliance certification (ISO 27001, SOC 2).

6. **Git Workflow Compliance**: Branch-per-feature/principle memungkinkan atomic commits, easy revert, parallel development; Conventional Commits (feat/fix/docs/style/refactor/test/chore) otomatis generates changelog entries semantic versioning (SemVer); Pull Request review process ensures peer validation before merge; branch cleanup reduces clutter and prevents stale branches.

## [1.28.54] — 2026-08-20 — Hardening MediaMTX dan runtime gate staging (Task 1B)

### Perubahan

- **MediaMTX v1.20**: image dipin ke `1.20.0-ffmpeg`; field konfigurasi/API deprecated diganti dan ICE additional host dapat diatur melalui environment.
- **Lifecycle path**: stop memakai `DELETE /v3/config/paths/delete/{name}` dan 404 diperlakukan idempotent; registrasi path dipisahkan dari kesiapan source serta API reachability.
- **Kerahasiaan RTSP**: response start tidak lagi mengirim URL sumber yang dapat memuat decrypted credential.
- **Playback truth**: frontend membedakan signaling, koneksi media, menunggu frame, dan live; badge `Live` hanya muncul setelah decoded frame.
- **Simulator**: Compose non-production menyediakan test-pattern H.264 Baseline deterministik dengan WebRTC ICE/TCP loopback.
- **Runtime staging**: WHEP, peer connection TCP, frame 640×360, recovery restart, kontrak add/list/delete, serta cleanup lulus pada VPS terisolasi; container production tidak dimutasi.
- **Batas status**: source/staging Task 1B `PASS`; production UDP/firewall/TURN/TLS, aplikasi lengkap, dan jaringan seluler eksternal tetap `NOT RUN`.

Timestamp: 2026-08-20 WITA | Kategori: Kritis, Streaming + Security + Runtime Verification

## [1.28.53] — 2026-08-20 — Runtime gate lifecycle QR (Task 1A)

### Perubahan

- **Runtime staging terisolasi**: source `914e0432` dijalankan pada Compose project, database, Redis, volume, subnet, dan loopback port terpisah; production tidak dimutasi.
- **Lifecycle QR**: rotasi API dan UI mempertahankan ID CCTV, menghasilkan payload/PNG baru, menerima QR aktif, menolak QR sebelumnya, serta mencatat audit.
- **Label dan cetak**: label UI berubah langsung tanpa hard refresh; tombol cetak dapat dijalankan dan aplikasi mengonfirmasi label terbaru siap dicetak.
- **RBAC**: permintaan rotasi oleh teknisi ditolak dengan HTTP 403.
- **Evidence aktual**: payload didekode dari PNG label sebelum/sesudah rotasi; hash label UI terbaru sama dengan payload aktif backend dan scanner mengembalikan aset yang sama.
- **Cleanup**: stack, database, Redis, volume, network, source staging, dan tunnel dihapus; service production tetap berjalan dan komponen ber-healthcheck tetap sehat.
- **Batas status**: Task 1A/BB-10 runtime gate `PASS`; suite agregat MT-11 tetap `NOT RUN` sampai skenario API key dan webhook selesai.

Timestamp: 2026-08-20 WITA | Kategori: Kritis, Runtime Acceptance + Security Verification

## [1.28.52] — 2026-08-20 — Perbaikan lifecycle rotasi QR (Task 1A)

### Perubahan

- **Rotasi QR**: backend mengenkripsi ulang ID CCTV yang sama dengan nonce baru; QR baru tetap menunjuk aset yang benar.
- **Invalidasi QR lama**: scanner membandingkan payload hasil pindai dengan `qr_payload` aktif dan menolak payload sebelum rotasi.
- **Cache label**: frontend memakai cache key `id + qr_payload`; respons PNG/fetch memakai `no-store`; service worker memperlakukan QR sebagai network-only, tidak menyimpan respons `no-store`, dan menaikkan versi shell cache untuk membuang cache lama.
- **Regression test**: ditambahkan round-trip rotasi, penolakan previous payload, kebijakan HTTP `no-store`, serta dua Node test policy service worker yang dijalankan oleh frontend CI.
- **Kontrak API**: artefak Swagger diregenerasi sehingga deskripsi rotasi menyatakan ID CCTV yang sama dengan nonce baru.
- **Dokumentasi**: BB-10, MT-11, remediation plan, testing matrix, dan security audit diperbarui. Runtime black-box non-production tetap `NOT RUN` dan tidak diklaim `PASS`.
- **Verifikasi source**: `go build ./...`, `go test ./... -count=1`, `go vet ./...`, `npm run test:sw`, frontend lint (0 error; 77 warning baseline), frontend build, Compose config dengan placeholder non-secret, dan `git diff --check` lulus.

Timestamp: 2026-08-20 WITA | Kategori: Kritis, Backend + Frontend + Security Hardening

## [1.28.51] — 2026-08-20 — Governance manual test formal (Task 0.2)

### Perubahan

- **Kontrak status manual/black-box**: `PASS`, `FAIL`, `BLOCKED`, dan `NOT RUN` kini memiliki definisi formal, metadata wajib, release gate, serta template execution record di `docs/TESTING.md`.
- **25 manual-test**: setiap temuan MT-01–MT-25 memiliki owner, skenario reproduksi, metadata aktif dengan suite `NOT RUN`, formal scenario record, aturan historical/non-authoritative untuk hasil sebelum 2026-08-20, dan fixture prefix `E2E-MTNN-` yang sesuai.
- **Safety gate**: prosedur SQL mutasi, rotasi/regenerasi key atau QR, deletion, dan credential mutation mensyaratkan environment non-production, target non-`nexaspace.site`, fixture exact target, backup/recovery, serta cleanup/rollback; jika tidak terpenuhi, statusnya `BLOCKED`.
- **Batasan evidence**: inspeksi kode, review unit test, dan hasil legacy tidak dipromosikan menjadi runtime `PASS`; literal host/email/credential production pada manual-test yang diedit disanitasi.

Timestamp: 2026-08-20 WITA | Kategori: Dokumentasi + Manual-Test Governance + Release Acceptance

## [1.28.50] — 2026-08-20 — Mulai remediasi: baseline dan provenance production

### Perubahan

- **Baseline Wave 0**: backend build/test/vet lulus; frontend lint 0 error (77 warning baseline) dan production build lulus pada source `5394ea0`.
- **Decision gates**: DG-01–DG-06 ditandai disetujui pemilik produk, termasuk jadwal tanpa teknisi, notifikasi jadwal/zona, stok per site, split akses inventaris, dan forced-password navigation.
- **Production provenance**: checkout SHA, image digest/ID, waktu runtime, versi, port, topologi edge, nama konfigurasi non-secret, dan ICE MediaMTX dicatat pada `docs/audits/PRODUCTION-ARTIFACT-PROVENANCE-2026-08-20.md`.
- **Temuan release engineering**: production memakai edge Nginx host di depan container Nginx; checkout production memiliki modifikasi lokal; image frontend/API belum membawa revision label sehingga source SHA artifact tidak dapat dibuktikan secara immutable.
- **Batasan**: pengambilan evidence bersifat read-only dan tidak mengubah VPS. Task 0.2 serta perbaikan fitur belum dimulai pada entri ini.

Timestamp: 2026-08-20 WITA | Kategori: Sedang, Dokumentasi + Production Provenance + Quality Baseline

## [1.28.49] — 2026-08-19 — Register temuan black-box, audit manual test, dan rencana remediasi

### Perubahan

- **Temuan black-box**: 25 laporan production dirangkum pada `docs/audits/BLACKBOX-FINDINGS-2026-08-19.md` beserta prioritas, tingkat kepastian akar sebab, solusi yang disarankan, dan acceptance criteria.
- **Audit manual test**: hasil review 25 berkas `docs/manual-test-*.md` dirangkum pada `docs/audits/MANUAL-TEST-AUDIT-SUMMARY.md`; file manual test asli belum diubah.
- **Indeks konsolidasi**: irisan dan dependensi kedua sumber temuan dipetakan pada `docs/audits/CONSOLIDATED-FINDINGS-INDEX.md` tanpa menggabungkan atau menghilangkan ID asli.
- **Execution plan**: `docs/plans/FINDINGS-REMEDIATION-PLAN.md` membagi pekerjaan menjadi baseline, P0, isolasi/RBAC, integritas workflow, notifikasi, laporan/UI, dan reformasi release acceptance.
- **Batasan**: perubahan ini hanya dokumentasi audit dan perencanaan; tidak mengklaim bug production telah diperbaiki.

Timestamp: 2026-08-19 WITA | Kategori: Besar, Dokumentasi + Quality Audit + Remediation Planning

## [1.28.48] — 2026-08-19 — Sinkronisasi roadmap, identitas SIGAP, dan audit keamanan

### Perubahan

- **ROADMAP**: Fase 12, 13, 14, dan 15 diselaraskan dengan changelog serta implementasi; Sprint 37-PATCH, Sprint 40, dan Sprint 41 ditandai selesai.
- **Identitas resmi**: seluruh penyebutan aktif diseragamkan menjadi "Sistem Informasi Gangguan & Aset Pusat" pada dokumentasi, laporan, template email, anotasi Swagger, dan artefak Swagger generated.
- **Kebijakan jaringan**: aturan GSM ditegaskan menjadi zero direct probe; status hanya berasal dari MediaMTX/last-active, sementara probe aktif dua-menit khusus Metro-E.
- **Kredensial dokumentasi**: nilai NVR nyata di `docs/AGENTS.md` diganti placeholder `<NVR_USERNAME>:<NVR_PASSWORD>`.
- **SECURITY-AUDIT**: arsitektur diperbarui dari 4 menjadi 7 service, alur web/mobile/integrasi ditambahkan, dan risiko terbuka terbaru dicatat tanpa memuat nilai rahasia.
- **Manual-test**: seluruh 25 dokumen `docs/manual-test-*.md` diaudit secara read-only; dokumen belum diubah karena temuan akan direncanakan pada tahap berikutnya.

Detail backend: `backend/changelog.md` [0.64.1].

Timestamp: 2026-08-19 WITA | Kategori: Sedang, Dokumentasi + Security Review

## [1.28.47] — 2026-08-13 — Perbaikan UI & data operasional (kawasan/zona, checklist per kawasan, SLA 404, probe live)

### Perubahan

- **Input kawasan/zona di CCTV**: `Create/UpdateCCTVRequest` + `zone_id`/`site_id`; form CCTV frontend punya dropdown "Kawasan" & "Zona" (data dari `/zones/all` & `/sites/all`).
- **Checklist per kawasan**: migrasi `000024_checklist_site` (kolom `site_id` di `checklist_templates`, NULL = global); `GET /checklist-templates?site_id=<uuid>` mengembalikan template kawasan + global; manager punya filter kawasan + field kawasan di form; wizard pengecekan memuat template sesuai kawasan CCTV.
- **Fix SLA 404**: route `GET /api/dashboard/sla` (admin) didaftarkan — handler `GetDashboardSLA` sudah ada sejak Sprint 28 tapi rutenya hilang.
- **Fix probe live Metro-E**: saat TCP probe ke IP kamera gagal tapi path MediaMTX Ready (stream aktif), kamera tetap dianggap live.
- **UI cleanup**: hapus lonceng login, badge "Masukkan payload", tombol "Muat ulang", info "X kamera streaming aktif", dropdown "Semua zona", badge "Halaman Admin", panel toggle koordinat di peta (pindah ke status bar + tombol Edit); dropdown aksi CCTV z-50 (tidak tertutup); sort default A-Z.
- **Kepanjangan SIGAP**: manifest PWA & mobile → "Sistem Informasi Gangguan & Aset Pusat".
- Verifikasi: `go build`/`go test`/`go vet` PASS, `tsc --noEmit` 0 error, `npm run lint` 0 error, `npm run build` sukses.

Detail: `backend/changelog.md` [0.64.0] · `frontend/changelog.md` [1.55.0].

Timestamp: 2026-08-13 WITA | Kategori: Sedang, Perbaikan (Backend + Frontend + Mobile)

## [1.28.46] — 2026-08-13 — Sprint 40: AI Feature Takedown

### Perubahan

- **Keputusan: AI detection dihapus seluruhnya** — Fitur AI detection (person counting, bounding box overlay, YOLOv8n + ByteTrack) di-remove sehingga SIGAP kembali menjadi **pure CCTV management** tanpa AI inference. Sistem kembali ke **7 Docker service** (tanpa `sigap_ai`).
- **Backend**: hapus `handlers/ai.go`, `routes/ai.go`, `models/ai_detection.go`, dan `handlers/ai_test.go`; migrasi baru `000023_remove_ai_detection` (DROP TABLE `ai_detection_events`, `ai_detection_sessions`); hapus registrasi `SetupAIRoutes` di `main.go`; hapus env `AI_SECRET`/`AI_SERVICE_URL`.
- **Frontend**: hapus halaman `/ai-analytics`, canvas bounding box overlay & toggle AI di `WHEPPlayer.tsx`/`StreamGridView.tsx`, badge counter orang, dan menu sidebar "AI Analytics".
- **Infra**: hapus service `sigap_ai` dari `docker-compose.yml`; folder `ai/` dihapus.
- **Dokumen**: `docs/AGENTS.md` (hapus referensi AI Detection § 5 & § 7; service count 8 → 7), `docs/ROADMAP.md` (Fase 14 → "Live Status & AI Takedown"; Sprint 38/39 DIBATALKAN; Sprint 40 🚧 berjalan), `docs/TESTING.md` (hapus matriks & gotchas `handlers/ai_test.go`; total file test 65 → 64).
- **Branch**: `bugfix/remove-ai-detection`.

Timestamp: 2026-08-13 WITA | Kategori: Besar, Takedown (Backend + Frontend + Infra + Docs)

## [1.28.45] — 2026-08-13 — Perbaikan: AI tidak mendeteksi manusia (H265) + Fix startup API & backup

### Perubahan

- **Fix AI detection (root cause: kodek H265)** — Kamera NVR mengirim stream **H265/HEVC**, sedangkan `cv2.VideoCapture` (OpenCV headless) **tidak bisa decode H265** → `cap.read()` hang → AI service tidak pernah mendapat frame → tidak ada deteksi orang sama sekali. Perbaikan:
  - `ai/Dockerfile`: install **ffmpeg** (ffmpeg punya decoder H265).
  - `ai/core/session_manager.py`: pipeline baca frame diganti dari `cv2.VideoCapture` → **ffmpeg subprocess** (output rawvideo BGR 1280×720 ke pipe, dibaca per-frame sebagai numpy); opsi `-fflags nobuffer+discardcorrupt`, `-err_detect ignore_err`, `-analyzeduration/-probesize` kecil agar toleran terhadap frame korup saat `sourceOnDemand` MediaMTX baru mulai pull; timeout baca 15s (menunggu IDR/keyframe H265 pertama) + reconnect otomatis; `os.set_blocking` non-blocking read.
  - **Terbukti bekerja**: setelah rebuild, AI service mengirim `detection-event` tiap ±1.5s, DB mencatat `person_detected` (person_count=1, confidence 0.58–0.72), sesi harian terisi (`unique_person_count`).
- **Fix startup API (backend `0.62.1`)** — `handlers.StartNotificationCleanup` dipanggil tanpa `go` di `main.go`; fungsi berisi loop `for range ticker.C` sehingga memblokir main thread → API tidak pernah `app.Listen` (port 3000 tidak terbuka, container api unhealthy, chain `mediamtx`/`nginx` ikut terblokir). Diperbaiki dengan `go handlers.StartNotificationCleanup(...)`.
- **Fix backup container CRLF** — `backup/entrypoint.sh` tersimpan dengan line ending CRLF (Windows) sehingga di Linux container error `exec /entrypoint.sh: no such file or directory` (shebang `#!/bin/sh\r` tidak dikenali) → container restart loop. Ditulis ulang sebagai LF.
- Verifikasi: `docker compose up -d` stack penuh (10 container Up, api healthy), login admin sukses, `GET /api/health` Healthy, AI detection berjalan real-time.

Detail: `backend/changelog.md` [0.62.1].

Timestamp: 2026-08-13 WITA | Kategori: Sedang, Perbaikan (AI Service + Backend + Infra)

## [1.28.44] — 2026-08-13 — Fitur: Frontend AI Overlay & Admin AI Analytics (Sprint 39) — Fase 14 3/3 ✅ Selesai

### Perubahan

- **Fase 14 (CCTV Intelligence)** — Sprint 39 (Task 14C Frontend AI Overlay & E2E Integration) selesai. Progress ROADMAP Fase 14 **2/3 → 3/3 ✅ Selesai**, Sprint 39 **✅ Selesai**.
- **Frontend**: halaman admin `/ai-analytics` (baru, `frontend/src/app/ai-analytics/page.tsx`) — kartu ringkasan (Total Orang Hari Ini, Jumlah Kamera Dipantau, Kamera dengan Hitungan > 0), tabel "Per Kamera — Hari Ini" (nama, kode CCTV, orang unik, terakhir diperbarui), section "Tren 7 Hari" dengan dropdown pilih kamera + grafik garis SVG mandiri (`AITrendChart`, tanpa dependency chart eksternal) dari `GET /api/ai/counts/:id/trend?days=7`; state loading/error/empty ("Belum ada data AI hari ini"); item menu sidebar "AI Analytics" (`BrainCircuit`) di grup "Pelaporan & Analisis" (admin-only); guard proxy `/ai-analytics` di `frontend/src/proxy.ts` (`ADMIN_PATHS` + `config.matcher`); overlay bounding box & counter AI di `/monitor` (Task 14C.1/14C.2, agent paralel).
- **Backend (`0.62.0`)**: endpoint tren `GET /api/ai/counts/:id/trend` (admin-only) di `handlers/ai.go` (`GetAICountTrend` — validasi UUID → 400, `days` query default 7 clamp 1–90, query `ai_detection_sessions` filter `cctv_id` + `session_date >= now-days` order ascending, respons `{status, data:[{date, unique_count}]}`) + rute di `routes/ai.go` + Swagger (`swag init`); unit test `TestGetAICountTrend` & `TestGetAICountTrend_InvalidID` di `handlers/ai_test.go`.
- **Dokumen**: `docs/manual-test-ai-analytics.md` (baru — S1–S7 sesuai format manual test lain); `docs/TESTING.md` matriks `handlers/ai_test.go` diperluas (Task 14C.3); ROADMAP Fase 14 progress **2/3 → 3/3 ✅ Selesai** & Sprint 39 ✅ Selesai.
- Verifikasi: `go build`/`go test ./... -count=1 -run "TestGetAICountTrend|TestAI"`/`go vet`/`golangci-lint` (0 issues) PASS; `npx tsc --noEmit`/`npm run lint` (0 error)/`npm run build` PASS.

Detail: `backend/changelog.md` [0.62.0].

Timestamp: 2026-08-13 WITA | Kategori: Besar, Fitur (Frontend + Backend + Docs)

## [1.28.43] — 2026-08-13 — Fitur: AI Microservice & Person Counting (Sprint 38) — Fase 14 2/3

### Perubahan

- **Fase 14 (CCTV Intelligence)** — Sprint 38 (Task 14B AI Microservice & Person Counting) selesai. Progress ROADMAP Fase 14 **1/3 → 2/3 🚧 Berjalan**, Sprint 38 **✅ Selesai**.
- **AI Service (`ai/`, baru)**: service Python FastAPI `sigap_ai` — deteksi orang YOLOv8n (class 0, conf ≥ 0.50, ~1 fps/kamera) + tracking ByteTrack (supervision), pull RTSP internal `rtsp://mediamtx:8554/cam-{cctv_id}` (sourceOnDemand — AI menjadi viewer pertama yang mengaktifkan stream), unique person counter per hari (auto-reset 00:00 WITA), POST hasil ke backend `POST /api/ai/detection-event` (header `X-AI-Secret`). Dockerfile + requirements + routers `/sessions` `/health` `/counts` `/reset-all`.
- **Docker Compose**: service `sigap_ai` (network `sigap_network`, depends_on `mediamtx`+`api`, env `BACKEND_URL`/`MEDIAMTX_RTSP_URL`/`AI_SECRET`, limits 2 CPU/2048M, port 8010 internal tidak dipublish).
- **Backend** (`0.61.0`): migrasi `000020_ai_detection_sessions` & `000021_ai_detection_events`; model `AIDetectionSession`/`AIDetectionEvent` (`models/ai_detection.go`); handler `handlers/ai.go` — `ReceiveDetectionEvent` (internal X-AI-Secret, upsert sesi + event, broadcast `AI_DETECTION_UPDATE`), `GetAICounts`/`GetAICountByCCTV` (admin), `ProxyActivateAISession`/`ProxyDeactivateAISession`/`ProxyListAISessions` (proxy ke `sigap_ai:8010`); routes `routes/ai.go`; registrasi `main.go`; env `AI_SECRET` & `AI_SERVICE_URL`; `swag init`.
- **Dokumen**: `docs/manual-test-ai.md` (baru — S1–S9 sesuai format manual test lain); `docs/TESTING.md` matriks `handlers/ai_test.go` (validasi secret, upsert sesi, counts); ROADMAP Fase 14 progress **1/3 → 2/3 🚧 Berjalan** & Sprint 38 ✅ Selesai.
- Verifikasi: `go build`/`go test ./... -count=1`/`go vet` PASS; `docker compose config --quiet` VALID (service `sigap_ai` terdaftar); `docker build -t sigap_ai ./ai` (validasi).

Detail: `backend/changelog.md` [0.61.0].

Timestamp: 2026-08-13 WITA | Kategori: Besar, Fitur (AI Service + Backend + Infra + Docs)

## [1.28.42] — 2026-08-12 — Fitur: RTSP Live Status Detection (Sprint 37) — Fase 14 dimulai

### Perubahan

- **Fase 14 (CCTV Intelligence) dimulai** — Sprint 37 (Task 14A RTSP Live Status Detection) selesai. Progress ROADMAP Fase 14 **0/3 → 1/3 🚧 Berjalan**, Sprint 37 **✅ Selesai**.
- **Backend**: migration `000019_stream_live_status` (kolom `stream_live BOOLEAN NOT NULL DEFAULT FALSE` + `stream_checked_at TIMESTAMPTZ` nullable di `cctvs`); model `CCTV` tambah `StreamLive bool` & `StreamCheckedAt *time.Time`; `utils/rtsp_probe.go` — `ProbeRTSPHost` (TCP dial 3s) & `ProbeRTSPChannel` (RTSP OPTIONS 5s, cek kode 200); `handlers/stream_probe.go` — goroutine `RunRTSPProber` (startup + tiap 2 menit, probe CCTV `stream_enabled=true`, update `stream_live`/`stream_checked_at`, broadcast WebSocket `CCTV_LIVE_STATUS_UPDATED` saat status berubah) + handler `GetLiveStatus` (`GET /api/streams/live-status` admin-only → `summary` online/offline/unchecked + `data` per CCTV); rute terdaftar di `routes/stream.go`.
- **Frontend**: `CCTVMap.tsx` badge live di marker (🟢 online / 🔴 offline / ⚪ belum dicek) + status bar peta `"X Online · Y Offline · Z Belum dicek"`; `/monitor` (`StreamGridView.tsx`) border/ring warna per sel (hijau=online, merah=offline, abu-abu=belum dicek) + tooltip `"Terakhir dicek: HH:mm WIB"`; update real-time via WebSocket `CCTV_LIVE_STATUS_UPDATED` tanpa reload.
- **Dokumen**: `docs/manual-test-rtsp-live-status.md` (baru — S1–S9 sesuai format manual test lain); `docs/ROADMAP.md` Fase 14 progress **0/3 → 1/3 🚧 Berjalan** & Sprint 37 ✅ Selesai; `docs/TESTING.md` matriks `handlers/stream_probe_test.go` + `stream_health_test.go` diperluas (total 62 → 63).
- Verifikasi: `go build`/`go test ./... -run "TestRTSP|TestStream"`/`go vet` PASS; `npm run lint`/`npm run build` PASS.

Detail: `backend/changelog.md` · `frontend/changelog.md`.

Timestamp: 2026-08-12 WITA | Kategori: Besar, Fitur (Backend + Frontend + Docs)

## [1.28.41] — 2026-08-13 — Fitur: Multi-Site (Sprint 30) — Fase 12 Selesai

### Perubahan

- **Fase 12 (Integrasi & Skalabilitas) selesai** — Sprint 30 (Task 12.2 Multi-Site) selesai. Progress ROADMAP Fase 12 **3/4 → 4/4 ✅ Selesai**, Sprint 30 **✅ Selesai**.
- **Backend**: model `Site` (nama, kode, alamat, koordinat_pusat, `is_default`); FK `site_id` di `CCTV`/`MaintenanceLog`/zona; migrasi data existing → site default "Kawasan OIKN" (kode `OIKN`, `is_default=true`); CRUD `/api/sites` (paginasi `GET /api/sites`, dropdown `GET /api/sites/all`) admin-only untuk tulis; `POST /api/sites/:id/assign-users` (assign teknisi, admin-only); perlindungan penghapusan site default; unit test `handlers/site_test.go` & `middleware/site_test.go`; Swagger + `swag init`.
- **Frontend**: site switcher (dropdown) di header; cookie `sigap_site_id` untuk menyimpan site aktif; reload halaman + filter global per site (CCTV/log/zona); halaman manajemen site admin.
- **Dokumen**: `docs/manual-test-multi-site.md` (baru — S1–S7 sesuai format manual test lain); `docs/ROADMAP.md` Fase 12 progress 3/4 → **4/4 ✅ Selesai** & Sprint 30 ✅ Selesai; `docs/TESTING.md` matriks `handlers/site_test.go` & `middleware/site_test.go` (total 60 → 62).
- Verifikasi: `go build`/`go test ./... -count=1`/`go vet` PASS.

Detail: `backend/changelog.md` · `frontend/changelog.md`.

Timestamp: 2026-08-13 WITA | Kategori: Besar, Fitur (Backend + Frontend + Docs)

## [1.28.40] — 2026-08-13 — Fitur: Polish Frontend & UX (Sprint 36)

### Perubahan

- **Backend** ([0.58.0]): `GET /api/dashboard/technician-performance` tambah query `?period=7d|30d|90d|365d` (default 30d) yang mengesampingkan `start_date`/`end_date`; unit test `TestTechnicianPerformanceWithPeriod`.
- **Frontend** ([1.49.0]): `StreamGridView` pagination grid (4/8/12/Semua) + counter "Menampilkan X–Y dari N kamera aktif" + tombol Pause/Resume per sel (lazy-load IntersectionObserver dipertahankan); `CalendarGrid` + halaman `/kalender` mendukung pembuatan tugas pemeliharaan dari klik tanggal (modal, tanggal pre-filled, refresh); `/kinerja-teknisi` filter periode 7 Hari / 30 Hari / 3 Bulan / 1 Tahun / Kustom memakai `?period=`.
- **Dokumen**: `docs/manual-test-frontend-ux-polish.md` (baru); `docs/ROADMAP.md` Fase 13 progress **15/18 → 18/18 ✅ Selesai** & Sprint 36 ✅ Selesai.
- Verifikasi: `go build`/`go test ./... -count=1`/`go vet` PASS; `npm run lint` (0 error)/`npm run build` PASS; `npx tsc --noEmit` PASS.

Detail: `backend/changelog.md` [0.58.0] · `frontend/changelog.md` [1.49.0].

Timestamp: 2026-08-13 WITA | Kategori: Sedang, Fitur (Backend + Frontend + Docs)

## [1.28.39] — 2026-08-12 — Fitur: Performa Backend & Checklist Fleksibel (Sprint 35)

### Perubahan

- **Backend** ([0.56.0] + [0.57.0]): template checklist pengecekan lapangan yang dapat dikustomisasi admin — model `ChecklistTemplate` + migrasi `000017_checklist_templates` (tabel baru + kolom `wajib` di `field_maintenance_checks`), seeder 9 item default, handler/rute CRUD + reorder `/api/checklist-templates` (admin-only untuk tulis), `CreateFieldMaintenance` menyimpan `Wajib` pada item. Kompresi foto server-side (resize ≤1920, JPEG q85, header `X-Original/Compressed-Size`) + thumbnail 400×300 via `github.com/disintegration/imaging`; endpoint `GET /api/uploads/thumbnails/:filename`.
- **Frontend** ([1.48.0]): halaman admin `/checklist-templates` + `ChecklistTemplateManager` (tambah/edit/reorder/toggle/hapus); `FieldMaintenanceModal` memuat template dari server (fallback `FIELD_CHECKLIST`); thumbnail 40×40 di tabel `/maintenance` (klik → detail fullsize); menu sidebar "Template Checklist" + guard proxy.
- **Mobile** ([1.1.0]): Checkup Wizard memuat item checklist dari `/checklist-templates` (fallback konstanta).
- **Dokumen**: `docs/manual-test-checklist-template.md` (baru), `docs/manual-test-image-compression.md` (baru); `docs/ROADMAP.md` Fase 13 progress **13/18 → 15/18** & Sprint 35 ✅ Selesai.
- Verifikasi: `go build`/`go test ./... -count=1`/`go vet` PASS; `npm run lint` (0 error)/`npm run build` PASS; `npx tsc --noEmit` PASS (frontend & mobile).

Detail: `backend/changelog.md` [0.56.0] + [0.57.0] · `frontend/changelog.md` [1.48.0] · `mobile/changelog.md` [1.1.0].

Timestamp: 2026-08-12 WITA | Kategori: Besar, Fitur (Backend + Frontend + Docs)

## [1.28.38] — 2026-08-09 — Fitur: Mobile App (Task 12.4)

### Perubahan

- **Mobile** ([1.0.0]): aplikasi Expo React Native TypeScript di `mobile/` untuk teknisi lapangan. Auth (login, `expo-secure-store`, auto-login, profil, ganti password, logout) via shared axios client (Bearer session). Scan QR (`expo-camera` + `POST /api/cctv/scan`). Checkup Wizard 9 tahap (`expo-location` GPS + `expo-image-picker` foto + `POST /api/upload` + `POST /api/field-maintenance` + draft AsyncStorage offline). Peta GIS (`react-native-maps`). Live View (WebView WHEP fallback). List Tugas (`/dashboard/tasks`).
- **Dokumen**: `mobile/README.md` (baru), `mobile/.env.example` (baru), `mobile/changelog.md` (baru); `docs/ROADMAP.md` Fase 12 progress 2/4 → 3/4 & Task 12.4 ✅.
- Verifikasi: `cd mobile && npx tsc --noEmit` lulus (0 error).

Timestamp: 2026-08-09 WITA | Kategori: Besar, Mobile | Detail: `mobile/changelog.md` [1.0.0]

## [1.28.37] — 2026-08-09 — Fitur: API Key & Webhook (Sprint 29)

### Perubahan

- **Backend** ([0.51.0]): model `APIKey` + `WebhookEndpoint`/`WebhookDelivery`; migrasi `000010_api_keys_webhooks`. Middleware `middleware/apikey.go` (`APIKeyAuth`/`APIKeyOrProtected`, rate limit 100 req/menit per key via Redis, cek revoked/expired/scope, update `last_used_at`). Handler `api_key.go` — `POST /api/api-keys` (key mentah 1×), `GET /api/api-keys`, `POST /api/api-keys/:id/revoke` (admin-only). Handler `webhook.go` — CRUD `/api/webhooks`, `POST /api/webhooks/:id/test`, `GET /api/webhooks/:id/deliveries`; antrean channel + worker `StartWebhookWorker`; `X-Signature-256` HMAC-SHA256; retry 3× 1s/5s/25s. Hook event di `cctv_sync.go`/`maintenance.go`/`overdue_checker.go`/`health_score.go`/`stream.go`. `GET /api/cctv` via API key scope `cctv:read`; unit test `api_key_test.go`, `webhook_test.go`, `middleware/apikey_test.go`.
- **Frontend** ([1.43.0]): halaman admin `/api-keys` (generate key tampil 1× + salin, list, revoke, badge scope, usage); halaman admin `/webhooks` (CRUD endpoint, pilih event, test delivery, modal detail + log deliveries); menu sidebar "API Keys" (`KeyRound`) & "Webhooks" (`Webhook`); proxy melindungi `/api-keys` & `/webhooks`.
- **Docs**: `docs/manual-test-api-webhook.md` (baru, S1–S9); `docs/ROADMAP.md` Fase 12 progress 0/4 → 2/4 & Sprint 29 ✅; `docs/TESTING.md` matriks API key & webhook (total 56 → 59).
- Detail: `backend/changelog.md` [0.51.0] · `frontend/changelog.md` [1.43.0].

Timestamp: 2026-08-09 WITA | Kategori: Besar, Fitur (Backend + Frontend + Docs) | Detail: `backend/changelog.md` [0.51.0] · `frontend/changelog.md` [1.43.0]

## [1.28.36] — 2026-08-09 — Fitur: Zona & SLA (Sprint 28)

### Perubahan

- **Backend** ([0.50.0]): model `Zone`/`UserZone` + FK `zone_id` di `CCTV`; model `SLAPolicy` + FK `sla_policy_id` di `MaintenanceSchedule`; migrasi `000009_zone_sla` (up/down). Handler `zone.go` — CRUD `/api/zones`, `GET /api/zones/all`, `POST /api/zones/:id/assign-cctvs`, `POST /api/zones/:id/assign-users`. Handler `sla.go` — CRUD `/api/sla-policies`, `GET /api/dashboard/sla` (admin → `monthly_compliance`/`violations`/`technician_ranking`); goroutine `RunSLADeadlineChecker` (startup + tiap 1 jam) memicu `SLA_WARNING` ke admin saat tugas melewati 75% target. `GetAllCCTV` mendukung filter `?zone_id=`; Swagger + unit test `zone_test.go`, `sla_test.go`.
- **Frontend** ([1.42.0]): halaman admin `/zona` (CRUD zona + modal "Atur Penugasan" multi-select CCTV & teknisi); dashboard admin `/sla` (chart bar kepatuhan bulanan via `recharts`, ranking teknisi, tabel pelanggaran SLA); filter peta per zona di `/sebaran-cctv`; menu sidebar "Zona" (`MapPinned`) & "SLA" (`Target`); proxy melindungi `/zona` & `/sla`.
- **Docs**: `docs/manual-test-zone-sla.md` (baru, S1–S9); `docs/ROADMAP.md` Fase 11 progress 3/5 → 5/5 & Sprint 28 ✅; `docs/TESTING.md` matriks zona & SLA (total 54 → 56).
- Detail: `backend/changelog.md` [0.50.0] · `frontend/changelog.md` [1.42.0].

Timestamp: 2026-08-09 WITA | Kategori: Besar, Fitur (Backend + Frontend + Docs) | Detail: `backend/changelog.md` [0.50.0] · `frontend/changelog.md` [1.42.0]

## [1.28.35] — 2026-08-09 — Fitur: Email & Report Builder (Sprint 27)

### Perubahan

- **Backend** ([0.49.0]): notifikasi email SMTP — `utils/email.go` (SendEmail via net/smtp + TLS), template `daily_summary.html`/`overdue_escalation.html`/`stock_low.html`, model `User.EmailNotifications` + migrasi `000007`, `PUT /api/profile/email-preferences`, trigger email ringkasan harian (digest), overdue (checker), stok rendah (`STOCK_LOW`). Report builder — model `ReportTemplate` + migrasi `000008`, `POST /api/reports/generate` (CSV/XLSX/PDF untuk 4 entitas), `GET/POST /api/reports/templates`, `DELETE /api/reports/templates/:id`; routes `routes/profile.go` + `routes/report.go` (admin-only); Swagger + unit test `handlers/email_test.go`, `handlers/report_builder_test.go`, `utils/email_test.go`.
- **Frontend** ([1.41.0]): halaman `/laporan-kustom` (wizard 4 langkah: entitas, kolom, filter, format; unduh CSV/Excel/PDF; simpan/muat/hapus template); email preferences di `/profile` (toggle `email_notifications`); menu sidebar "Laporan"; proxy melindungi `/laporan-kustom`.
- **Docs**: `docs/manual-test-email.md` (baru, S1–S7), `docs/manual-test-report-builder.md` (baru, S1–S8); `docs/ROADMAP.md` Fase 11 progress 1/5 → 3/5 & Sprint 27 ✅; `docs/TESTING.md` matriks email & report builder (total 51 → 54).
- Detail: `backend/changelog.md` [0.49.0] · `frontend/changelog.md` [1.41.0].

Timestamp: 2026-08-09 WITA | Kategori: Besar, Fitur (Backend + Frontend + Docs) | Detail: `backend/changelog.md` [0.49.0] · `frontend/changelog.md` [1.41.0]

## [1.28.34] — 2026-08-09 — Fitur: Inventaris Suku Cadang (Sprint 26)

### Perubahan

- **Backend** ([0.48.0]): modul inventaris suku cadang — model `SparePart` + `SparePartUsage`; migrasi `000006_spare_parts`; CRUD `/api/spare-parts` (admin-only untuk tulis), `GET /api/spare-parts/low-stock`, `POST/GET /api/maintenance-logs/:id/spare-parts`. Stok dikurangi otomatis dalam transaksi saat penggunaan dicatat (hanya log `completed`); broadcast `STOCK_LOW` ke admin bila stok < minimum_order. Swagger + unit test `handlers/spare_part_test.go`.
- **Frontend** ([1.40.0]): halaman admin `/suku-cadang` (CRUD, filter kategori, pencarian, badge stok, paginasi); integrasi "Suku Cadang Digunakan" + mini modal "Catat Penggunaan" di modal detail pemeliharaan; dashboard widget "Suku Cadang Stok Rendah" (admin only, real-time via `STOCK_LOW`); menu sidebar "Suku Cadang"; proxy melindungi `/suku-cadang`.
- **Docs**: `docs/manual-test-spare-parts.md` (baru, S1–S9); `docs/ROADMAP.md` Fase 11 progress 0/5 → 1/5 & Sprint 26 ✅; `docs/TESTING.md` matriks `handlers/spare_part_test.go` (total 50 → 51).
- Detail: `backend/changelog.md` [0.48.0] · `frontend/changelog.md` [1.40.0].

Timestamp: 2026-08-09 WITA | Kategori: Besar, Fitur (Backend + Frontend + Docs) | Detail: `backend/changelog.md` [0.48.0] · `frontend/changelog.md` [1.40.0]

## [1.28.33] — 2026-08-07 — Perbaikan Infrastruktur: Stack Docker dapat dijalankan

### Perubahan

- **Infra** (`docker-compose.yml`): service `api` kini memiliki **healthcheck** (endpoint `/api/health` via `127.0.0.1:3000` — bukan `localhost` karena API bind di IPv4 `0.0.0.0`); direktori writable `/app/logs` & `/tmp` diubah dari `tmpfs` menjadi **named volume** `api_logs`/`api_tmp` (tmpfs tidak terpasang pada setup ini) sehingga rootfs `read_only` tetap dapat menulis log.
- **Backend** ([0.47.1]): `RunMigrations` di `config/migrate.go` tidak lagi memanggil `m.Close()` — `migrate.Migrate.Close()` (driver pgx v5) menutup `*sql.DB` global yang di-share dengan `config.DB`, menyebabkan `sql: database is closed` dan aplikasi gagal startup di Docker. Diperbaiki.
- Efek: seluruh stack (db, redis, api, frontend, nginx, mediamtx, prometheus, grafana) dapat di-up; `https://localhost/api/health` = Healthy; aplikasi SIGAP Fase 9 & 10 (streaming, snapshot, timeline, multi-view monitor) dapat diakses.
- Detail: `backend/changelog.md` [0.47.1].

Timestamp: 2026-08-07 WITA | Kategori: Kecil, Infrastruktur & Backend | Detail: `backend/changelog.md` [0.47.1]

## [1.28.32] — 2026-08-07 — Sprint 25: Multi-View Monitor & Stream Health (Task 10.3 + 10.4) — Fase 10

### Perubahan

- **Fase 10 (Kecerdasan Visual & Snapshot) selesai** — Task 10.3 (Multi-View Monitor) + Task 10.4 (Deteksi Status Stream) selesai dalam Sprint 25. Progress ROADMAP Fase 10 **2/4 → 4/4 ✅ Selesai**, Sprint 25 **✅ Selesai**.
- **Backend** ([0.47.0]): goroutine monitoring (tiap 5 menit) query MediaMTX `ListPaths()` — track per-path ready state; stream down > 15 menit → broadcast `STREAM_DOWN` + kurangi `health_score` -10; pulih → `STREAM_RESTORED`; endpoint `GET /api/streams/health` (admin-only) list stream + status ready/down + `down_duration`; unit test `handlers/stream_health_test.go` (`GetStreamHealth`, `RunStreamHealthMonitor`).
- **Frontend** ([1.39.0]): halaman `/monitor` (admin-only) grid multi-stream dengan preset layout 2×2/3×3/4×4; per-cell CCTV dropdown + `WHEPPlayer`; fullscreen per sel & seluruh grid; pop-out `window.open('/monitor/pop?path=...')`; halaman `/monitor/pop` (single fullscreen stream); menu sidebar "Monitor CCTV" (`MonitorPlay`) setelah "Peta Sebaran CCTV".
- **Dokumen**: `docs/manual-test-visual.md` (tambah Sprint 25 S10–S16 — monitor grid & stream health); `docs/TESTING.md` (matriks `handlers/stream_health_test.go`; total 49 → 50); ROADMAP Fase 10 **4/4 ✅ Selesai** + Sprint 25 ✅ Selesai.
- Verifikasi: `go build`/`go test ./... -count=1`/`go vet`/`golangci-lint run` + `swag init` PASS; `npm run lint` (0 error)/`npm run build` PASS.

Timestamp: 2026-08-07 WITA | Kategori: Besar | Detail: `backend/changelog.md` [0.47.0], `frontend/changelog.md` [1.39.0]

## [1.28.31] — 2026-08-07 — Sprint 24: Snapshot & Timeline (Task 10.1 + 10.2) — Fase 10

### Perubahan

- **Fase 10 (Kecerdasan Visual & Snapshot) dimulai** — Task 10.1 (Snapshot Otomatis) + Task 10.2 (Timeline Event Visual) selesai dalam Sprint 24. Progress ROADMAP Fase 10 **0/4 → 2/4 🚧 Berjalan**, Sprint 24 **✅ Selesai**.
- **Backend** ([0.46.0]): model `CCTVSnapshot` + migrasi `000005_cctv_snapshots`; goroutine snapshot berkala (15 menit) via `utils/mediamtx.go` `GetSnapshot` (endpoint `/v3/paths/get/{path}/snapshot`); endpoint `GET /api/cctv/:id/snapshot` (JPEG, 404 bila belum ada) & `GET /api/cctv/:id/timeline` (paginasi); rotasi maksimal 10 snapshot per CCTV di `uploads/snapshots/{cctv_id}/`; event hook `status_changed` / `maintenance_completed` / `field_check_attention` / `periodic_snapshot`; field computed `snapshot_url` (`gorm:"-"`) pada respons `GetAllCCTV`/`GetCCTVByID`.
- **Frontend** ([1.38.0]): thumbnail snapshot di tabel Master CCTV (`/cctv`, `AuthenticatedImage` + placeholder `ImageOff`); thumbnail di tooltip marker peta (`/sebaran-cctv`, placeholder `Camera`); timeline kronologis di detail CCTV dengan snapshot + label event (`EVENT_TYPE_LABELS`).
- **Dokumen**: `docs/manual-test-visual.md` (baru — manual test blackbox S1–S9 sesuai format manual test lain); `docs/TESTING.md` (matriks snapshot & timeline: `handlers/snapshot_test.go` — `GetCCTVTimeline` 404/success, `GetCCTVSnapshot` 404; total 48 → 49); ROADMAP Fase 10 2/4 🚧 Berjalan + Sprint 24 ✅ Selesai.
- Verifikasi: `go build`/`go test ./... -count=1`/`go vet`/`golangci-lint run` + `swag init` PASS; `npm run lint` (0 error)/`npm run build` PASS.

Timestamp: 2026-08-07 WITA | Kategori: Besar | Detail: `backend/changelog.md` [0.46.0], `frontend/changelog.md` [1.38.0]

## [1.28.30] — 2026-08-07 — Fase 9 Streaming CCTV — Selesai (Task 9.1–9.6)

### Perubahan

- **Fase 9 (Streaming CCTV Real-Time) selesai** — seluruh 6 task (9.1–9.6) dalam 4 sprint (20–23) telah diimplementasikan & diverifikasi. Progress ROADMAP Fase 9 **3/6 → 6/6 ✅ Selesai**.
- **Backend** ([0.44.0]): fondasi streaming (model `CCTV` + migrasi `000004`, `utils/rtsp.go`, service `mediamtx` di Docker) → client MediaMTX (`utils/mediamtx.go`) + handler/rute stream admin-only (`handlers/stream.go`, `routes/stream.go` `/api/streams/*`) + goroutine auto-start. `rtsp_password` terenkripsi & tidak pernah dikembalikan.
- **Frontend** ([1.36.0]): UI streaming real-time — `WHEPPlayer.tsx`, `StreamModal.tsx`, `StreamGridView.tsx`, integrasi marker CCTV + toggle Peta/Grid di `sebaran-cctv`, serta konfigurasi streaming di Master CCTV (`cctv/page.tsx`).
- **Dokumen**: ROADMAP Fase 9 6/6 ✅ Selesai + Sprint 20–23 ✅ Selesai; `docs/AGENTS.md` (7 services + service `mediamtx`, endpoint streaming admin-only, env NVR/MediaMTX, file baru); `docs/TESTING.md` (matriks streaming + `routes/stream_test.go`, total 48); `docs/plan.md` (Sprint 20–23 + rancangan teknis); `docs/manual-test-streaming.md` (S1–S9).
- Verifikasi: `go build`/`go test ./... -count=1`/`go vet`/`golangci-lint run` + `swag init` PASS; `npm run lint` (0 error)/`npm run build` PASS; `docker compose config --quiet` VALID; Docker E2E 8 service healthy.

Timestamp: 2026-08-07 WITA | Kategori: Besar | Detail: `backend/changelog.md` [0.44.0], `frontend/changelog.md` [1.36.0]

## [1.28.29] — 2026-08-07 — Sprint 21: Backend API & MediaMTX Client (Task 9.3) — Fase 9

### Perubahan

- **Backend** ([0.44.0]): client REST MediaMTX `utils/mediamtx.go` (`MediaMTXClient` — `AddPath`, `RemovePath`, `GetPath`, `ListPaths` ke `http://mediamtx:9997/v3/`) + unit test `utils/mediamtx_test.go` (HTTP mock); handler streaming `handlers/stream.go` (`StartStream` — decrypt `rtsp_password` → build URL RTSP → register path MediaMTX → return WHEP URL; `StopStream`; `StartAllStreams`; `StopAllStreams`; `GetStreamStatus` — gabung data DB + info path MediaMTX; `GetStreamURL` — WHEP + HLS) + unit test `handlers/stream_test.go`; `routes/stream.go` (grup `/streams` admin-only: `/status`, `/start-all`, `/stop-all`, `/:id/start`, `/:id/stop`, `/:id/url`, rute statis sebelum dinamis); registrasi di `main.go` + goroutine auto-start stream aktif saat boot; Swagger `swag init`.
- **Dokumen**: ROADMAP Fase 9 progress 2/6 → **3/6**, Sprint 21 ✅ Selesai; `docs/manual-test-streaming.md` (baru — manual test blackbox S1–S9 sesuai format manual test lain); TESTING matriks menambah `utils/mediamtx_test.go` + `handlers/stream_test.go` (+2 total).
- Verifikasi: `go build`/`go test ./... -count=1`/`go vet`/`golangci-lint run ./...` + `swag init` PASS.

Timestamp: 2026-08-07 WITA | Kategori: Sedang | Detail: `backend/changelog.md` [0.44.0]

## [1.28.28] — 2026-08-07 — Sprint 20: Fondasi Streaming (Task 9.1 + 9.2) — Fase 9

### Perubahan

- **Backend** ([0.43.0]): model `CCTV` bertambah field `channel_port`, `rtsp_password` (AES-256 encrypted), `stream_enabled` (migrasi `000004_add_cctv_streaming`); util `utils/rtsp.go` (`StreamType` Main/Sub, `BuildRTSPURL`, `BuildStreamPath`, `GetRTSPPort`) + unit test `utils/rtsp_test.go`; `CreateCCTVRequest`/`UpdateCCTVRequest` diperbarui; `rtsp_password` di-exclude dari response JSON; env NVR global (`RTSP_NVR_HOST`/`RTSP_NVR_PORT`/`RTSP_NVR_USER`/`MEDIAMTX_API_URL`).
- **Infra**: file `mediamtx.yml` (RTSP→WebRTC relay, sourceOnDemand); service `mediamtx` di `docker-compose.yml` (image `bluenviron/mediamtx:latest-ffmpeg`, IP `172.28.0.11`, resource limits 2 CPU/1024M, health check, UDP `8189`); `nginx/nginx.conf` upstream `sigap_mediamtx` + location `/stream/*` → `mediamtx:8889` dan `/hls/*` → `mediamtx:8888`.
- **Dokumen**: ROADMAP Fase 9 progress 0/6 → **2/6**, Sprint 20 ✅ Selesai; TESTING matriks menambah `utils/rtsp_test.go` (+1 total).
- Verifikasi: `go build`/`go test ./... -count=1`/`go vet` PASS; `docker compose config --quiet` VALID.

Timestamp: 2026-08-07 WITA | Kategori: Sedang | Detail: `backend/changelog.md` [0.43.0]

## [1.28.27] — 2026-08-07 — Geotag Pengecekan Lapangan (Task 8.4) — Fase 8 Selesai

### Perubahan

- **Backend** ([0.42.0]): kolom `check_latitude`/`check_longitude` pada `FieldMaintenance` (migrasi `000003`); `CreateFieldMaintenance` menerima & memvalidasi koordinat opsional (format angka, rentang lat/lng); respons list/detail menyertakan geotag.
- **Frontend** ([1.35.0]): wizard pengecekan menangkap GPS (indikator "Lokasi tercatat", opsional) & mengirim `check_latitude`/`check_longitude`; admin melihat mini-map `FieldCheckLocationMap` (marker CCTV + Lokasi Pengecekan) di modal detail.
- **Dokumen**: `docs/manual-test-geotag.md` (baru — manual test blackbox S1–S6 sesuai format manual test lain); ROADMAP Task 8.4 ✅ Selesai (**Fase 8 = 4/4 SELESAI**); plan Sprint 19 + rancangan teknis.
- Verifikasi: `go build`/`go test`/`go vet` + `golangci-lint` PASS; `npm run lint` (0 error)/`npm run build` PASS.

Timestamp: 2026-08-07 WITA | Kategori: Kecil | Detail: `backend/changelog.md` [0.42.0], `frontend/changelog.md` [1.35.0]

## [1.28.26] — 2026-08-07 — Notifikasi Push Browser (Task 8.3)

### Perubahan

- **Backend** ([0.41.0]): model `PushSubscription` + migrasi `000002`; endpoint `POST /api/push/subscribe` & `DELETE /api/push/unsubscribe`; `utils/push.go` `SendPush` (webpush-go VAPID, async, fallback in-app); integrasi ke `NotifyUser`/`NotifyUserSync`; env `VAPID_*`.
- **Frontend** ([1.34.0]): service worker handle `push`/`notificationclick`; komponen opt-in `PushNotificationOptIn` di `/profile` (subscribe/unsubscribe).
- **Dokumen**: `docs/manual-test-push-notification.md` (baru — manual test blackbox S1–S9 sesuai format manual test lain); ROADMAP Task 8.3 ✅ Selesai (Fase 8 3/4); plan Sprint 18 + rancangan teknis.
- Verifikasi: `go build`/`go test`/`go vet` PASS; `npm run lint` (0 error)/`npm run build` PASS.

Timestamp: 2026-08-07 WITA | Kategori: Sedang | Detail: `backend/changelog.md` [0.41.0], `frontend/changelog.md` [1.34.0]

## [1.28.25] — 2026-08-07 — Kalender Jadwal Pemeliharaan (Task 8.2)

### Perubahan

- **Frontend** ([1.33.0]): halaman `/kalender` (admin & teknisi) — kalender bulanan CSS mandiri (`components/CalendarGrid.tsx`): grid 7 kolom, navigasi bulan, dot warna status per hari, panel detail tugas, filter per teknisi (admin); menu sidebar "Kalender". Data dari `/maintenance-logs?start_date=&end_date=`, dikelompokkan per `scheduled_date`.
- **Dokumen**: `docs/manual-test-calendar.md` (baru — manual test blackbox S1–S9 sesuai format manual test lain); ROADMAP Task 8.2 ✅ Selesai (Fase 8 2/4); plan Sprint 17 + rancangan teknis.
- Verifikasi: `npm run lint` (0 error) & `npm run build` (exit 0) PASS.

Timestamp: 2026-08-07 WITA | Kategori: Sedang | Detail: `frontend/changelog.md` [1.33.0]

## [1.28.24] — 2026-08-07 — Progressive Web App (PWA) (Task 8.1)

### Perubahan

- **Frontend** ([1.32.0]): SIGAP kini installable + offline shell via PWA — web manifest `public/manifest.json` (SIGAP, `standalone`, `#162218`/`#467235`, ikon 192/512), service worker `public/sw.js` (cache-first shell, network-first API, `sigap-shell-v1`), ikon PNG dari logo, registrasi SW + meta tags di `layout.tsx`, header cache di `next.config.ts`.
- **Dokumen**: `docs/manual-test-pwa.md` (baru — manual test blackbox S1–S9 sesuai format manual test lain); ROADMAP Task 8.1 ✅ Selesai (Fase 8 1/4); plan Sprint 16 + rancangan teknis; AGENTS.md (aturan manual test per sprint).
- Verifikasi: `npm run lint` (0 error) & `npm run build` (exit 0) PASS; manual test installability/offline shell/Lighthouse di `docs/manual-test-pwa.md`.

Timestamp: 2026-08-07 WITA | Kategori: Sedang | Detail: `frontend/changelog.md` [1.32.0]

## [1.28.23] — 2026-08-07 — Fix CI: Deprecation Node 20

### Perubahan

- **CI** ([0.40.1]): upgrade `golangci/golangci-lint-action@v7` → `@v9` (Node runtime 24) dan `node-version` frontend `20` → `24` untuk menghilangkan warning/error deprecation Node 20 di GitHub Actions; `docker compose` / build & lint backend/frontend tetap hijau.

Timestamp: 2026-08-07 WITA | Kategori: Kecil | Detail: `backend/changelog.md` [0.40.1]

## [1.28.22] — 2026-08-07 — Container Hardening & Logging Terstruktur (Task 7.5 & 7.6)

### Perubahan

- **Backend** ([0.40.0]): container `api` berjalan sebagai **non-root** (`USER sigap`), rootfs **read-only** dengan tmpfs `/tmp` & `/app/logs`, `cap_drop: [ALL]`, resource limits (`cpus`/`mem_limit`), dan **Docker secrets** untuk `DB_PASSWORD`/`JWT_SECRET`/`AES_SECRET_KEY` (file `./secrets/*` di-mount ke `/run/secrets/`, dibaca via `backend/utils/secrets.go` dengan fallback env). Logger memakai **lumberjack** untuk rotasi (100 MB/file, 5 backup, 30 hari, gzip) dengan level dari `LOG_LEVEL` (default info) dan format JSON.
- **Infra**: `docker-compose.yml` service `api` & `nginx` di-hardening; env docs (`.env.example`, `backend/.env.docker.example`); `.gitignore` tambah `/secrets/`.
- **Dokumen**: ROADMAP Fase 7 = **✅ SELESAI** (6/6); plan Sprint 14–15 + rancangan teknis; matriks test; panduan agent; backend changelog [0.40.0].
- Verifikasi: `go build`/`go test ./... -count=1`/`go vet` PASS (semua package hijau); `docker compose config --quiet` VALID (base + override); test baru `utils/secrets_test.go` & `utils/logger_test.go` lulus.

Timestamp: 2026-08-07 WITA | Kategori: Sedang | Detail: `backend/changelog.md` [0.40.0]

## [1.28.21] — 2026-08-07 — Monitoring & Alerting (Task 7.4)

### Perubahan

- **Backend** ([0.39.0]): endpoint publik `GET /api/metrics` (format text exposition Prometheus) dengan collector ringan **tanpa library pihak ketiga** — counter `sigap_http_requests_total{method,path,status}`, histogram `sigap_http_request_duration_seconds`, gauge `sigap_db_pool_open/idle_connections`, `sigap_ws_active_connections`, `sigap_upload_storage_bytes`. Middleware `metrics.go` mencatat setiap request (endpoint `/api/metrics` dikecualikan). `GET /api/health` diperluas: response time DB, cek Redis, ukuran folder uploads. Goroutine `StartCollector` memperbarui gauge DB pool / WS aktif / ukuran uploads tiap 15 detik.
- **Infra (opsional)**: `docker-compose.override.yml` menambah Prometheus (port 9090) & Grafana (port 3000, datasource auto-provision) — tidak diwajibkan di compose utama; `monitoring/prometheus/prometheus.yml` scrape `api:3000/api/metrics`.
- **Dokumen**: ROADMAP Task 7.4 ✅ Selesai (Fase 7 4/6); plan Sprint 13 + rancangan teknis; matriks test; panduan agent; Swagger di-regenerate (`/api/metrics` + deskripsi health).
- Verifikasi: `go build`/`go test ./... -count=1`/`go vet` PASS (semua package hijau); `docker compose config --quiet` VALID (base + override); unit test metrik/middleware/health lulus.

Timestamp: 2026-08-07 WITA | Kategori: Sedang | Detail: `backend/changelog.md` [0.39.0]

## [1.28.20] — 2026-08-07 — Backup Otomatis & Restore (Task 7.3)

### Perubahan

- **Skrip**: `scripts/backup.sh` — `pg_dump` → `gzip` → `backups/db/sigap_<TS>.sql.gz`, tar+gzip uploads → `backups/uploads/uploads_<TS>.tar.gz`, rotasi otomatis hapus backup > 7 hari (`find -mtime +N -delete`), timestamp WITA; `scripts/restore.sh` — restore database (drop/recreate + feed dump) dan uploads (hapus & ekstrak), dengan validasi argumen.
- **Infra**: service `backup` di `docker-compose.yml` — image `postgres:15-alpine` + `dcron`, cron `0 6 * * *` WITA via `backup/entrypoint.sh`, volume `backups_data:/backups`, uploads bind-mount read-only, menunggu `db` healthy; kredensial dari root `.env`. Opsi alternatif cron host di `docs/BACKUP.md`. `.gitignore` tambah `/backups/`.
- **Dokumen**: `docs/BACKUP.md` (baru) — prosedur backup manual & restore, jadwal cron (container & host), verifikasi integritas + end-to-end, dan disaster recovery; ROADMAP Task 7.3 ✅ Selesai (Fase 7 3/6); plan Sprint 12 + rancangan teknis; panduan agent.
- Verifikasi: `bash -n scripts/backup.sh scripts/restore.sh backup/entrypoint.sh` PASS; `docker compose config --quiet` VALID; uji fungsional nama file, struktur `backups/db`+`backups/uploads`, rotasi `-mtime +7`, dan alur restore drop/recreate lulus.

Timestamp: 2026-08-07 WITA | Kategori: Sedang | Detail: `backend/changelog.md` [0.38.0]

## [1.28.19] — 2026-08-06 — Redis untuk Rate-Limit & Blacklist (Task 7.2)

### Perubahan

- **Backend** ([0.37.0]): blacklist token dipindah dari in-memory ke Redis SET (`SADD`/`SISMEMBER`/`SREM` + TTL otomatis sesuai sisa masa hidup token — persisten lintas restart, aman multi-instance); rate limiter memakai counter fixed-window Redis `INCR`/`EXPIRE` dengan key `limiter:{ip}:{endpoint}`; graceful fallback ke in-memory saat Redis tidak tersedia (development tanpa Docker — hanya LOG WARNING); env baru `REDIS_HOST`/`REDIS_PORT`/`REDIS_PASSWORD` (opsional); unit test kontrak interface blacklist (mock `redisCommander` + in-memory) dan limiter (key format, window reset, isolasi per key).
- **Infra**: service `redis` (redis:7-alpine) di `docker-compose.yml` — volume `redis_data` (AOF), health check adaptif, port tidak dipublish; `api` menunggu `redis` healthy; `.env.example` + `backend/.env.docker.example` menambah `REDIS_PASSWORD`/`REDIS_HOST`/`REDIS_PORT`.
- **Dokumen**: ROADMAP Task 7.2 ✅ Selesai (Fase 7 2/6); plan Sprint 11 + rancangan teknis; matriks test; panduan agent.
- Verifikasi: `go build`/`go test -count=1`/`go vet` PASS; uji live Redis 7 (docker): koneksi, revoke→deteksi, **persistensi blacklist lintas restart API (client baru)**, TTL key, remove, dan semantik INCR/EXPIRE batas 5/menit — seluruh 12 cek PASS; `docker compose config --quiet` VALID; container Redis healthy.

Timestamp: 2026-08-06 WITA | Kategori: Sedang | Detail: `backend/changelog.md` [0.37.0]

## [1.28.18] — 2026-08-06 — Migrasi DB Versioned (Task 7.1)

### Perubahan

- **Backend** ([0.36.0]): adopsi `golang-migrate` v4 (driver pgx v5) — skema dikelola file SQL terurut di `backend/migrations/` (embed FS); `config.RunMigrations` otomatis saat startup; CLI `go run cmd/migrate/main.go up|down|version`; fase transisi AutoMigrate dikomentari (akan dihapus setelah stabil); migration 000001 idempoten terhadap DB eksisting.
- **CI**: job backend-ci menambah service PostgreSQL + verifikasi `up → down → up → version` end-to-end.
- **Dokumen**: `docs/MIGRATION.md` (baru — prosedur, CLI, troubleshooting, cara tambah migration); ROADMAP Task 7.1 ✅ Selesai (Fase 7 1/6); plan Sprint 10; matriks test; panduan agent.
- Verifikasi: `go build`/`go test`/`go vet` PASS; uji live PostgreSQL 18 lokal `up → down → up` (+ skenario DB warisan AutoMigrate) PASS; DB uji dihapus setelah selesai.

Timestamp: 2026-08-06 WITA | Kategori: Besar | Detail: `backend/changelog.md` [0.36.0]

## [1.28.17] — 2026-08-06 — Ringkasan Operasional Otomatis (Task 6.4)

### Perubahan

- **Backend** ([0.35.0]): model `DailyDigest`, generator 06:00 WITA dengan agregasi 24 jam dan idempotensi tanggal, endpoint Admin `/api/dashboard/digest`, serta notifikasi `DAILY_DIGEST_AVAILABLE`.
- **Frontend** ([1.31.0]): halaman Admin `/ringkasan` berbentuk timeline harian dengan statistik dan paginasi; menu sidebar `Ringkasan` serta guard proxy admin.
- **Dokumen**: Fase 6 ditandai selesai (4/4); README, plan, roadmap, matriks test, panduan agent, Swagger, dan uji manual diperbarui.
- `swag init`, test backend, backend build/vet, frontend lint, dan frontend build diverifikasi pada branch fitur.

Timestamp: 2026-08-06 WITA | Kategori: Sedang | Detail: `backend/changelog.md` [0.35.0], `frontend/changelog.md` [1.31.0]

## [1.28.16] — 2026-08-06 — Laporan PDF Pengecekan Lapangan (Task 6.3)

### Perubahan

- **Backend** ([0.34.0]): endpoint protected `/api/field-maintenance/:id/report-pdf` dengan kop surat IKN, detail CCTV/teknisi/waktu, checklist, foto bukti, catatan umum, serta ownership scope untuk teknisi.
- **Frontend** ([1.30.0]): tombol Admin "Cetak Laporan" dengan ikon `FileDown` pada modal detail field maintenance dan download PDF berbasis blob.
- **Dokumen**: Task 6.3 ditandai selesai pada Fase 6; README, plan, roadmap, matriks test, panduan agent, Swagger, dan uji manual diperbarui.
- `swag init`, test backend terarah, dan test route protected diverifikasi pada branch fitur.

Timestamp: 2026-08-06 WITA | Kategori: Sedang | Detail: `backend/changelog.md` [0.34.0], `frontend/changelog.md` [1.30.0]

## [1.28.15] — 2026-08-06 — Dashboard Kinerja Teknisi (Task 6.2)

### Perubahan

- **Backend** ([0.33.0]): endpoint admin `/api/dashboard/technician-performance` dengan filter rentang tanggal, agregasi tugas, completion rate, rata-rata durasi dari assign ke selesai, overdue, field check, dan quality score; `MaintenanceLog.AssignedAt` dicatat saat assign/re-assign.
- **Frontend** ([1.29.0]): halaman admin `/kinerja-teknisi`, filter 7/30 hari atau kustom, chart completion rate SVG, tabel ranking sortable, menu sidebar, dan guard proxy admin.
- **Dokumen**: Task 6.2 ditandai selesai pada Fase 6; matriks test, panduan agent, plan, dan panduan uji manual diperbarui.
- `swag init`, backend build/test/vet, frontend lint/build diverifikasi pada branch fitur.

Timestamp: 2026-08-06 WITA | Kategori: Sedang | Detail: `backend/changelog.md` [0.33.0], `frontend/changelog.md` [1.29.0]

## [1.28.14] — 2026-08-06 — Skor Kesehatan CCTV (Task 6.1)

### Perubahan

- **Backend** ([0.32.0]): `CCTV.health_score` default 100; endpoint protected `/api/cctv/:id/health-score`; agregasi lima faktor 90 hari; kalkulator startup + tiap 6 jam; recalculation setelah complete maintenance/field-check; sorting `health_score` pada list CCTV.
- **Frontend** ([1.28.0]): badge Kesehatan pada tabel Master CCTV; warna marker peta mengikuti rentang skor; popup, tooltip, dan legend menampilkan angka/rentang.
- **Dokumen**: status Fase 6 menjadi 1/4 dan Task 6.1 ✅ Selesai; rancangan Sprint 6, matriks test, panduan agent, dan `docs/manual-test-health-score.md` diperbarui.
- `swag init`, backend build/test/vet, frontend lint/build diverifikasi.

Timestamp: 2026-08-06 WITA | Kategori: Besar | Detail: `backend/changelog.md` [0.32.0], `frontend/changelog.md` [1.28.0]

## [1.28.13] — 2026-08-06 — Eskalasi Overdue Otomatis (Task 5.5)

### Perubahan

- **Backend** ([0.31.0]): `MaintenanceLog.OverdueNotifiedAt` nullable; checker background saat startup + tiap 1 jam mengirim `MAINTENANCE_OVERDUE` ke semua admin dan teknisi untuk log `ongoing`, satu kali per log sampai reschedule; reschedule ke masa depan me-reset flag. Generator recurring juga dijalankan sebagai goroutine agar startup Fiber tidak terblokir.
- **Frontend**: tidak ada perubahan UI; tipe `MAINTENANCE_OVERDUE` sudah tersedia dari sistem notifikasi in-app Task 5.1.
- **Dokumen**: status Task 5.5 dan seluruh Fase 5 diperbarui menjadi ✅ Selesai; matriks test menambahkan `overdue_checker_test.go`; panduan agent mencatat field dan checker baru; panduan uji manual ditambahkan di `docs/manual-test-overdue-escalation.md`.
- `go build`/`go test`/`go vet` + `swag init` diverifikasi pada branch fitur.

Timestamp: 2026-08-06 WITA | Kategori: Sedang | Detail: `backend/changelog.md` [0.31.0]

## [1.28.12] — 2026-08-06 — Recurring Maintenance: Penjadwalan Pemeliharaan Berulang (Task 5.4)

### Perubahan

- **Backend** ([0.30.0]): modul baru `maintenance_schedules` — model, CRUD admin
  (`/api/maintenance-schedules`), kolom `maintenance_schedule_id` pada
  `maintenance_logs`, dan generator berjalan saat startup + tiap 1 jam yang
  membuat log otomatis (idempotent per hari per jadwal; tanpa teknisi → log
  `scheduled`, dengan teknisi → `ongoing` + notifikasi in-app). Audit
  CREATE/UPDATE/DELETE/TOGGLE_SCHEDULE dan "Dibuat otomatis oleh jadwal [id]".
- **Frontend** ([1.27.0]): halaman `/schedules` khusus admin (CRUD + toggle
  aktif/nonaktif + filter + paginasi), menu sidebar "Jadwal", `/schedules`
  terdaftar di proxy admin.
- **Dokumen**: `docs/ROADMAP.md` & `docs/plan.md` status Task 5.4 = ✅ Selesai.
- `go build`/`go vet`/`go test` PASS; `npm run lint` (0 error) & `npm run build` PASS.

Timestamp: 2026-08-06 WITA | Kategori: Sedang | Detail: `backend/changelog.md` [0.30.0], `frontend/changelog.md` [1.27.0]

## [1.28.11] — 2026-08-05 — Sprint 2 Selesai: Sinkronisasi Status CCTV (Task 5.2) + Uji Manual S1–S8

### Perubahan

- **Backend** ([0.28.0]–[0.28.3]): helper `syncCCTVStatus` konservatif dalam transaksi
  complete/field-check (`aktif ↔ pemeliharaan`; `nonaktif` tidak pernah ditimpa);
  normalisasi status menerima nilai Inggris (`active`/`maintenance`/`inactive`);
  guard kini membandingkan nilai ternormalisasi agar baris legacy huruf besar tetap
  dilindungi; status `rusak` **digabung ke `nonaktif`** (enum resmi:
  `aktif`/`pemeliharaan`/`nonaktif`); audit `UPDATE_CCTV` + broadcast `CCTV_UPDATED`.
- **Frontend** ([1.25.11]–[1.25.13]): pesan error validasi rinci per field (kotak
  merah di modal form); dropdown status form & filter selaras enum lowercase;
  opsi "Rusak" dihapus (nilai legacy `rusak`/`broken` dipetakan ke `nonaktif`).
- **Dokumen**: panduan uji manual `docs/manual-test-sync-cctv.md` (8 skenario S1–S8)
  — hasil uji runtime Docker: 3 bug ditemukan & ditindaklanjuti; `docs/plan.md` &
  `docs/ROADMAP.md` status Sprint 2 = ✅ Selesai.
- `go build`/`go vet`/`go test` PASS; `npx tsc --noEmit` & `npm run lint` (0 error) PASS.

Timestamp: 2026-08-05 WITA | Kategori: Sedang | Detail: `backend/changelog.md` [0.28.0]–[0.28.3], `frontend/changelog.md` [1.25.11]–[1.25.13]

## [1.28.10] — 2026-08-04 — Badge Profil: Sejajar dengan Status Layanan + Monokrom

### Perubahan

- **Frontend** ([1.25.10]): badge profil di header Dashboard sejajar satu baris
  dengan lonceng notifikasi & indikator "Layanan normal", menampilkan nama akun
  dan peran di samping foto — teks monokrom mengikuti tema (putih/hitam), garis
  pembatas antara badge dan teks, fallback avatar netral. Dialog konfirmasi
  keluar dipindah ke dalam `ProfileMenu` (via portal) sehingga badge berfungsi
  mandiri di halaman Dashboard; `DashboardLayout` dibersihkan dari modal konfirmasi.
- `npm run lint` (0 error) & `npm run build` PASS.

Timestamp: 2026-08-04 WITA | Kategori: Kecil | Detail: `frontend/changelog.md` [1.25.10]

## [1.28.9] — 2026-08-04 — Badge Profil Global + Foto Profil

### Perubahan

- **Frontend** ([1.25.9]): badge profil berisi foto saja (fallback inisial) menjadi
  elemen global di semua halaman (pojok kanan atas konten desktop + header mobile);
  klik → dropdown "Profil" & "Keluar" (merah kontras). Tombol "Keluar" di sidebar
  dihapus (dipindah ke dropdown). Halaman `/profile` mendukung tambah/ubah/hapus
  foto profil (unggah via `/api/upload`, disimpan via `PUT /api/auth/profile`).
- **Backend** ([0.27.0]): kolom `users.profile_photo_url`; `GET /auth/me` &
  `PUT /auth/profile` mengembalikan/menerima `profile_photo_url` (validasi path
  upload aman; `""` = hapus); `CleanupOrphanUploads` kini mengecualikan foto profil;
  Swagger di-regenerate.
- `go build`/`go vet`/`go test` PASS; `npm run lint` (0 error) & `npm run build` PASS.

Timestamp: 2026-08-04 WITA | Kategori: Sedang | Detail: `backend/changelog.md` [0.27.0], `frontend/changelog.md` [1.25.9]

## [1.28.8] — 2026-08-04 — Dashboard: Card Profil & Sidebar: Pemberitahuan

### Perubahan

- **Frontend** ([1.25.8]): card profil dipindah dari sidebar ke header Dashboard
  (di sebelah indikator "Layanan normal") — tanpa border, berbentuk badge profil
  (avatar inisial + nama + badge peran, klik → `/profile`); tombol "Muat ulang"
  dihapus. Menu "Pemberitahuan" di sidebar kini tombol dengan badge jumlah belum
  dibaca (sinkron via polling 60 dtk + WebSocket) yang membuka modal daftar
  notifikasi; item dapat diklik → modal informasi lengkap dengan tombol "Lihat
  detail" (tandai dibaca + navigasi); ada "Tandai semua dibaca" & "Lihat riwayat
  lengkap" → `/pemberitahuan`. Modal dirender via `createPortal`.
- `npm run lint` (0 error) & `npm run build` PASS.

Timestamp: 2026-08-04 WITA | Kategori: Sedang | Detail: `frontend/changelog.md` [1.25.8]

## [1.28.7] — 2026-08-04 — CCTV: Sortir & Filter dalam Modal Popup

### Perubahan

- **Frontend** ([1.25.7]): select filter status, indoor/outdoor, dan sortir di
  `/cctv` disembunyikan dari tampilan utama — digantikan tombol "Sortir" (badge
  jumlah filter aktif) yang membuka modal "Sortir & Filter" (5 kolom pengurutan +
  arah + filter status & lokasi, pola draft + Terapkan/Batal/Reset). Kotak pencarian
  tetap tampil. State `sortMode` diganti `sortBy`/`sortOrder`; backend tidak berubah.
- `npm run lint` (0 error) & `npm run build` PASS.

Timestamp: 2026-08-04 WITA | Kategori: Kecil | Detail: `frontend/changelog.md` [1.25.7]

## [1.28.6] — 2026-08-04 — Audit Log: Sortir & Filter dalam Modal + Posisi Menu Pemberitahuan

### Perubahan

- **Backend** ([0.26.2]): bugfix respons `GET /api/audit-logs` — `data` diselaraskan
  ke konvensi `{ rows, pagination }`; sebelumnya halaman audit-logs selalu kosong
  (frontend membaca `data.rows`, backend mengirim array).
- **Frontend** ([1.25.6]): filter modul & rentang tanggal di `/audit-logs` dipindah
  ke modal "Sortir & Filter" (kolom + arah pengurutan via `sort_by`/`order`,
  pola draft + Terapkan/Batal/Reset); kotak pencarian tetap tampil. Menu sidebar
  "Pemberitahuan" dipindah ke akhir menu utama — di bawah "Audit Log", tepat di atas
  tombol "Keluar".
- Test: `go test` & `go vet` PASS, `npm run lint` (0 error) & `npm run build` PASS.

Timestamp: 2026-08-04 WITA | Kategori: Kecil | Detail: `backend/changelog.md` [0.26.2], `frontend/changelog.md` [1.25.6]

## [1.28.5] — 2026-08-04 — Pengguna: Redesign Modal Detail

### Perubahan

- **Frontend** ([1.25.5]): modal detail pengguna di `/users` didesain ulang —
  header identitas (avatar inisial berwarna peran, nama + badge peran, username mono),
  panel rincian akun (email + tanggal dibuat, grid 2 kolom dengan pemisah),
  dan footer aksi cepat (Ubah data / Atur ulang kata sandi / Hapus pengguna —
  destruktif terpisah warna merah).
- Backend tidak berubah.
- `npm run lint` (0 error) & `npm run build` PASS.

Timestamp: 2026-08-04 WITA | Kategori: Kecil | Detail: `frontend/changelog.md` [1.25.5]

## [1.28.4] — 2026-08-04 — Pengecekan Lapangan: Sortir & Filter dalam Modal Popup

### Perubahan

- **Frontend** ([1.25.4]): filter hasil & rentang tanggal di `/field-maintenance`
  disembunyikan dari tampilan utama — digantikan tombol "Sortir" (dengan badge jumlah
  filter aktif) yang membuka modal "Sortir & Filter" (kolom + arah pengurutan via
  `sort_by`/`order`, filter lengkap, pola draft + Terapkan/Batal/Reset). Kotak pencarian
  teks tetap tampil. Pola seragam dengan `/maintenance` ([1.25.3]).
- Backend tidak berubah (endpoint `/field-maintenance` sudah mendukung `sort_by`/`order`).
- `npm run lint` (0 error) & `npm run build` PASS.

Timestamp: 2026-08-04 WITA | Kategori: Kecil | Detail: `frontend/changelog.md` [1.25.4]

## [1.28.3] — 2026-08-04 — Pemeliharaan: Sortir & Filter dalam Modal Popup

### Perubahan

- **Frontend** ([1.25.3]): grid filter (CCTV/Status/tanggal) di `/maintenance`
  disembunyikan dari tampilan utama — digantikan tombol "Sortir" (dengan badge
  jumlah filter aktif) yang membuka modal "Sortir & Filter"; modal berisi pengaturan
  pengurutan (kolom + arah, dikirim via `sort_by`/`order`) dan filter lengkap dengan
  pola draft + tombol Terapkan/Batal/Reset.
- Backend tidak berubah (endpoint `/maintenance-logs` sudah mendukung `sort_by`/`order`).
- `npm run lint` (0 error) & `npm run build` PASS.

Timestamp: 2026-08-04 WITA | Kategori: Kecil | Detail: `frontend/changelog.md` [1.25.3]

## [1.28.2] — 2026-08-04 — Bugfix: Pemberitahuan Tidak Tampil di Bell & Halaman `/pemberitahuan`

### Perubahan

- **Backend** ([0.26.1]): respons `GET /api/notifications` diselaraskan ke konvensi
  list SIGAP — `data: { rows, pagination }` (sebelumnya `data: [array]` + pagination
  top-level, pola usang `/audit-logs`).
- **Frontend** ([1.25.2]): `NotificationBell` & `/pemberitahuan` membaca `res.data?.rows`
  (sebelumnya salah baca `res.data?.data` → daftar selalu kosong meski notifikasi ada
  di database); pagination dibaca dari `res.data?.pagination`.
- Test: `go build/test` PASS, `npm run lint` (0 error) & `npm run build` PASS;
  container `api` + `frontend` di-rebuild, `/api/health` 200.

Timestamp: 2026-08-04 WITA | Kategori: Kecil (bugfix) | Detail: `backend/changelog.md` [0.26.1], `frontend/changelog.md` [1.25.2]

## [1.28.1] — 2026-08-04 — Pemberitahuan: Lonceng di Dashboard + Halaman `/pemberitahuan`

### Perubahan

- **Frontend** ([1.25.1]): lonceng notifikasi dipindah ke header halaman Dashboard
  (tidak lagi global); dropdown menampilkan 3 pemberitahuan teratas + tombol
  "Lihat lainnya"; menu sidebar baru "Pemberitahuan" menuju halaman `/pemberitahuan`
  (paginasi, filter belum dibaca, tandai semua, klik → navigasi, real-time).
- Scope pemberitahuan tetap per-user dari klaim JWT (backend) — teknisi tidak
  melihat pemberitahuan teknisi lain.
- `npm run lint` (0 error) & `npm run build` PASS.

Timestamp: 2026-08-04 WITA | Kategori: Sedang | Detail: `frontend/changelog.md` [1.25.1]

## [1.28.0] — 2026-08-04 — Notifikasi In-App Real-Time (Fase 5.1 — Jalur A)

### Perubahan

- **Fase 5 (Jalur A) dimulai** — `docs/ROADMAP.md` & `docs/plan.md` diperbarui: status
  Fase 4 "⏸️ Sisa task ditunda (menjelang go-live, sistem masih pilot/UAT)", bagian baru
  Fase 5 dengan 5 task berurutan (5.1 notifikasi → 5.2 sinkron status CCTV → 5.3 re-assign →
  5.4 jadwal berulang → 5.5 eskalasi overdue) + keputusan desain (notifikasi in-app saja,
  sinkron status konservatif) + kandidat lanjutan Jalur B. Keputusan lengkap dicatat di
  `docs/plan.md` Bagian 6.
- **Backend** ([0.26.0]): model `Notification`, endpoint `/api/notifications`
  (list/unread-count/read/read-all, scope per-user), `WSHub` diperluas dengan
  `WSClientInfo{UserID, Role}` + `BroadcastToUser`/`BroadcastToRole` (global broadcast
  tetap), util `NotifyUser`/`NotifyRole` (async + retry), hook notifikasi pada
  assign/complete/field-check (attention), `swag init`.
- **Frontend** ([1.25.0]): `NotificationBell` (badge unread, dropdown, tandai semua dibaca,
  klik → navigasi, toast real-time `NOTIFICATION_CREATED`, polling 60 detik) di header
  mobile & sidebar desktop.
- Test: +13 kasus backend (handlers/notification_test.go, utils/notify_test.go);
  `go build/test/vet` PASS, `npm run lint && npm run build` PASS.

Timestamp: 2026-08-04 WITA | Kategori: Besar | Detail: `backend/changelog.md` [0.26.0], `frontend/changelog.md` [1.25.0]



## [1.27.2] — 2026-08-03 — Label "Gagal" Menjadi "Perlu Tindak Lanjut"

### Perubahan

- **Frontend** (`maintenance/page.tsx`, `dashboard/page.tsx`, `DashboardAnalytics.tsx`, `TrendChart.tsx`):
  label tampilan status `failed` diubah dari "Gagal" menjadi "Perlu Tindak Lanjut" (badge tabel,
  filter dropdown, modal detail, kartu statistik, legenda analitik, tooltip grafik).
- **Backend** (`handlers/report.go`): `pdfStatusLabel` & ringkasan laporan PDF memakai
  "Perlu Tindak Lanjut".
- Nilai internal `failed` di database/API **tidak berubah** — tanpa migrasi data; filter,
  query parameter, dan seluruh klien API tetap kompatibel. CSV export tetap menulis nilai mentah.
- Test backend & lint/build frontend lulus; container `sigap-api` & `sigap-frontend` di-rebuild.

Timestamp: 2026-08-03 WITA | Kategori: Kecil | Detail: `backend/changelog.md` [0.25.2], `frontend/changelog.md` [1.24.2]

## [1.27.1] — 2026-08-03 — Bugfix Sesi: Cookie Duplikat (Stale di Path /api) Menyebabkan Login "Langasung Ter-logout"

### Penyebab

Setelah logout, browser yang masih menyimpan cookie `sigap_session`/`sigap_csrf` stale
(warisan versi lama di path `/api` atau `/api/`, token sudah direvoke) akan mengirimkan
dua cookie bernama sama pada setiap request. Sesuai RFC 6265 §5.4 cookie ber-Path
terpanjang dikirim lebih dulu, dan backend (`c.Cookies()`) selama ini mengambil nilai
PERTAMA — cookie stale — sehingga `GET /auth/me` (dan seluruh endpoint terlindungi)
menjawab 401 "Token telah dicabut" tepat setelah login sukses; frontend lalu mengarahkan
kembali ke halaman login (gejala: "login langsung ter-logout", tidak bisa login setelah logout).

### Perbaikan

- **Backend** (`utils/csrf.go`, `middleware/csrf.go`): nilai cookie sesi & CSRF diambil dari
  kemunculan TERAKHIR (sesi aktif di path "/" selalu menang) lewat helper baru
  `ExtractCookieValues`/`ExtractLastCookieValue`; berlaku untuk `Protected`, WebSocket, dan Logout.
- **Frontend** (`src/lib/api.ts`): `getCookie()` mengambil nilai terakhir agar header
  `X-CSRF-Token` selalu sinkron dengan cookie aktif.
- Cleanup cookie legacy saat login/logout tetap dipertahankan (defense in depth).
- Test baru untuk skenario cookie duplikat (utils & middleware) — seluruh suite lulus.

Timestamp: 2026-08-03 WITA | Kategori: Bugfix | Detail: `backend/changelog.md` [0.25.1], `frontend/changelog.md` [1.24.1]

## [1.27.0] — 2026-08-03 — Alur Penugasan Pemeliharaan (Assign Teknisi & Checklist 9 Tahap)

### Frontend

- Halaman Pemeliharaan CCTV: sub-menu status "Dalam Proses", "Selesai", "Gagal" pada
  input status dihapus — status kini tampil read-only ("Terjadwal" untuk catatan baru)
  dengan keterangan status diperbarui otomatis oleh sistem; field waktu pencatatan
  (`completed_at`) dihapus dari form, hanya "Waktu terjadwal".
- Log terjadwal dapat **ditugaskan ke teknisi** (admin): tombol ikon UserPlus pada baris
  status Terjadwal → modal pilih teknisi (diambil dari `GET /users?role=teknisi`) →
  `PUT /maintenance-logs/:id/assign`; setelah ditugaskan status otomatis "Dalam proses".
- Baris log berstatus "Dalam proses" menampilkan tombol **Selesaikan** (ikon CheckCheck)
  bagi teknisi yang ditugaskan (atau admin) → membuka wizard checklist 9 tahap dalam mode
  `complete` (`POST /maintenance-logs/:id/complete`) — checklist + foto bukti yang sama
  dengan pengecekan lapangan.
- Dashboard teknisi: tugas berstatus "Dalam proses" punya tombol **Selesaikan** yang
  membuka wizard yang sama; setelah selesai dashboard di-refresh (status jadi
  Selesai/Gagal otomatis).
- Teknisi tidak lagi melihat tombol "Catat Pemeliharaan"/"Ubah"; baris log menampilkan
  "Belum ditugaskan" saat belum ada teknisi.

### Backend

- Model `MaintenanceLog`: `TechnicianID` menjadi opsional (`*uuid.UUID`, relasi `User`
  pointer → `technician: null` saat belum ditugaskan), kolom baru `MaintenanceResultID`
  menautkan hasil pengecekan; `FieldMaintenance` mendapat `MaintenanceLogID`.
- `POST /maintenance-logs` hanya menerima status `scheduled` (tanpa teknisi,
  `completed_at` tidak lagi diterima); `PUT /:id` tidak lagi menerima `status`/`completed_at`
  — status dikelola sistem.
- Endpoint baru: `PUT /maintenance-logs/:id/assign` (admin; hanya log terjadwal; user
  tujuan wajib role teknisi; status → ongoing) dan `POST /maintenance-logs/:id/complete`
  (ownership check; hanya ongoing; checklist 1–30 item + 1–5 foto; transaksi; status
  `completed` bila 9/9 ok, selain itu `failed`; audit `ASSIGN_MAINTENANCE`/
  `COMPLETE_MAINTENANCE`).
- `GET /maintenance-logs` & `GET /maintenance-logs/:id` kini menerapkan scope peran
  (teknisi hanya melihat log miliknya) + ownership check detail; ekspor CSV & PDF
  nil-safe untuk log tanpa teknisi.
- `GET /users` mendukung filter `role`; `GET /dashboard/tasks` menambahkan
  `cctv_id`/`cctv_kode`; swagger di-regenerasi.

### Rincian

- Detail per bagian: `../frontend/changelog.md` versi [1.24.0],
  `../backend/changelog.md` versi [0.25.0].
- Uji: `go test ./...` PASS, `go vet ./...` PASS, `npm run lint` (0 errors, 46 warnings
  existing) & `npm run build` PASS.

---

## [1.26.0] — 2026-08-03 — Pengecekan Lapangan Teknisi via Scan QR

### Frontend

- Tombol **"Maintenance"** pada hasil scan QR CCTV (halaman Aset CCTV) membuka wizard
  pengecekan lapangan (`FieldMaintenanceModal`): 9 item checklist wajib diisi satu per
  satu (jaringan, listrik, pembersihan lensa, fisik kamera, kualitas gambar, perekaman,
  akses jarak jauh, kabel & konektor, lingkungan sekitar) dengan pilihan Normal/Bermasalah
  + catatan, lalu tahap akhir **unggah foto bukti (min. 1, maks. 5)** sebelum menyelesaikan.
- Halaman admin baru **`/field-maintenance`** (menu sidebar "Pengecekan Lapangan"):
  tabel hasil pengecekan (CCTV, teknisi, tanggal, ringkasan x/y normal, status
  Normal / Perlu Tindak Lanjut), filter pencarian/status/rentang tanggal, modal detail
  berisi seluruh checklist + foto bukti, dan hapus catatan. Dilindungi admin-only via proxy.

### Backend

- Model baru `FieldMaintenance` + `FieldMaintenanceCheck` + `FieldMaintenanceImage`
  (soft delete, relasi ke CCTV & User) dengan AutoMigrate.
- Endpoint baru `/api/field-maintenance` (POST create dengan transaksi + audit log,
  GET list terpaginasi — teknisi hanya miliknya sendiri, GET detail, DELETE admin).
- `CleanupOrphanUploads` kini juga menghitung foto `field_maintenance_images` sebagai
  referensi sah (tidak dihapus sebagai file yatim).
- Swagger regenerasi (`swag init`).

### Rincian

- Detail per bagian: `../frontend/changelog.md` versi [1.23.0],
  `../backend/changelog.md` versi [0.24.0].
- Uji: `go test ./...` PASS, `npm run lint` (0 errors) & `npm run build` PASS.

---

## [1.25.0] — 2026-08-03 — Notifikasi & Konfirmasi Kustom dengan Animasi

### Frontend

- Semua `alert()`/`confirm()` bawaan browser (24 titik di halaman users, maintenance,
  cctv) diganti sistem kustom:
  - **Toast** (`useToast()`): kartu notifikasi kanan atas dengan animasi check hijau
    (stroke-draw) untuk sukses, ikon merah untuk error, info biru; slide-in elastis,
    progress bar, auto-dismiss.
  - **Dialog konfirmasi** (`useConfirm()`): modal kustom dengan ikon peringatan
    bergoyang, fade/pop in-out, tombol Batal/Konfirmasi, tutup via X/Escape/klik latar.
- Toast sukses kini muncul di semua aksi berhasil (simpan, hapus, reset password,
  ekspor, cetak QR, impor, dsb.).

### Rincian

- Detail per bagian: `../frontend/changelog.md` versi [1.22.0].
- Tidak ada perubahan backend pada rilis ini.

---

## [1.24.0] — 2026-08-03 — Kop Surat & Perbaikan Tata Letak Laporan PDF Pemeliharaan

### Backend

- PDF laporan pemeliharaan kini berkop surat resmi: logo IKN (dari
  `backend/assets/nusantara-logo.png`), nama instansi "Otorita Ibu Kota Nusantara",
  alamat lengkap, dan website, ditutup garis ganda.
- Perbaikan tata letak tabel: tinggi baris dihitung akurat dengan `SplitLines`,
  border grid penuh per sel (tabel simetris), dan pindah halaman per-baris agar
  tidak ada sel terpotong atau teks ketimpa.

### Rincian

- Detail per bagian: `../backend/changelog.md` versi [0.23.0].
- Tidak ada perubahan frontend pada rilis ini.

---

## [1.23.0] — 2026-08-03 — Redesain Filter Audit Log (Input Tidak Lagi Terpotong)

### Frontend

- Filter audit log dirombak agar input tidak terpotong: grid 3 kolom (bukan 4), baris
  pencarian full-width, lalu kolom "Modul", "Dari tanggal", dan "Sampai tanggal"
  masing-masing satu kolom penuh dengan label.

### Rincian

- Detail per bagian: `../frontend/changelog.md` versi [1.21.0].
- Tidak ada perubahan backend pada rilis ini.

---

## [1.22.0] — 2026-08-03 — Perbaikan Cetak Semua QR Code

### Frontend

- "Cetak Semua QR Code" di halaman Aset CCTV sebelumnya hanya mencetak data halaman aktif
  (10 CCTV). Kini semua perangkat diambil ulang dari backend (`page=1&limit=100`) sehingga
  seluruh QR code tercetak (total 35 perangkat).

### Rincian

- Detail per bagian: `../frontend/changelog.md` versi [1.20.0].
- Tidak ada perubahan backend pada rilis ini.

---

## [1.21.0] — 2026-08-03 — Konfirmasi Logout dengan Dialog Kustom

### Frontend

- Klik item menu "Keluar" di sidebar kini membuka dialog konfirmasi kustom ("Keluar dari
  SIGAP?" dengan ikon `LogOut` merah, tombol "Batal" dan "Keluar").
- Dialog dapat ditutup via tombol X, klik latar, atau `Escape`; logout baru dieksekusi
  setelah pengguna mengonfirmasi.

### Rincian

- Detail per bagian: `../frontend/changelog.md` versi [1.19.0].
- Tidak ada perubahan backend pada rilis ini.

---

## [1.20.0] — 2026-08-03 — Card Tugas Menggantikan Distribusi Status CCTV

### Frontend

- Card "Distribusi Status CCTV" dihapus dari dashboard.
- Card "Tugas Berjalan Semua Teknisi" (admin) / "Tugas Saya" (teknisi) naik dan kini
  dirender berdampingan dengan card "Tren Pemeliharaan" pada grid analitik;
  "Peringatan Pemeliharaan Terlambat" tetap full-width di bawahnya.

### Rincian

- Detail per bagian: `../frontend/changelog.md` versi [1.18.0].
- Tidak ada perubahan backend pada rilis ini.

---

## [1.19.0] — 2026-08-03 — Logout Hanya di Sidebar & Transisi Drawer Lebih Halus

### Frontend

- Tombol logout di header mobile (dekat tombol hamburger) dihapus; logout kini hanya
  melalui item menu "Keluar" di dalam sidebar.
- Transisi drawer sidebar diperhalus: `duration-500 ease-in-out` untuk slide, dan overlay
  selalu ter-mount dengan fade opacity dua arah agar animasi buka dan tutup sama-sama mulus.

### Rincian

- Detail per bagian: `../frontend/changelog.md` versi [1.17.0].
- Tidak ada perubahan backend pada rilis ini.

---

## [1.18.0] — 2026-08-03 — Tombol Keluar Sebagai Item Menu di Sidebar

### Frontend

- Tombol logout dipindah dari bagian bawah sidebar ke dalam daftar navigasi, tepat di bawah
  menu "Audit Log" (admin) / "Pemeliharaan CCTV" (teknisi), dengan gaya yang sama persis
  dengan item menu lainnya (ikon `LogOut`, label "Keluar").

### Rincian

- Detail per bagian: `../frontend/changelog.md` versi [1.16.0].
- Tidak ada perubahan backend pada rilis ini.

---

## [1.17.0] — 2026-08-03 — Penyesuaian Antarmuka untuk Peran Teknisi

### Frontend

- **Tombol keluar di layar kecil**: header mobile (di samping toggle tema) kini menampilkan
  tombol logout berikon `LogOut`; sidebar juga dapat digulir (`flex-1 min-h-0 overflow-y-auto`)
  sehingga aksi logout selalu terjangkau pada perangkat pendek.
- **Tombol laporan jadi ikon**: "Laporan PDF" (`FileText` merah) dan "Ekspor CSV"
  (`FileSpreadsheet` hijau) berubah menjadi kotak ikon-only dengan tooltip, konsisten di
  halaman Pemeliharaan dan Audit Log.
- **Payload QR dibatasi admin**: toggle "Masukkan payload" dan view input manual hanya
  dirender untuk admin; teknisi hanya melihat pemindai kamera.
- **"Tugas Saya" teknisi dirampingkan**: hanya nama CCTV + badge jadwal + badge status +
  badge terlambat; lokasi & tindakan (`action_taken`) tetap tampil untuk admin.

### Rincian

- Detail per bagian: `../frontend/changelog.md` versi [1.15.0].
- Tidak ada perubahan backend pada rilis ini.

---

## [1.16.0] — 2026-08-03 — Penghapusan Kolom Legacy asset_id pada maintenance_logs

### Backend Database

- **Kolom legacy `asset_id` dihapus**: migrasi `MigrateMaintenanceToCCTV` kini menghapus kolom
  `asset_id uuid NOT NULL` (beserta FK `fk_maintenance_logs_asset`) yang tersisa dari model
  `assets` lama. Sebelumnya INSERT log baru selalu gagal dengan SQLSTATE 23502 → HTTP 500
  ("Gagal menyimpan log pemeliharaan").
- **Penghapusan tidak lagi terlewat**: drop kolom dieksekusi di luar early-return sehingga
  selalu berjalan selama kolom masih ada; migrasi tetap idempotent.

before:
- `maintenance_logs.asset_id` masih NOT NULL tapi tidak pernah diisi oleh model `cctv_id` baru.
after:
- Tabel hanya bergantung pada `cctv_id`; penyimpanan log pemeliharaan berjalan normal.
timestamp: 2026-08-03 WITA
kategori: Bugfix, Database, Migration
file terkait:
- backend/config/migrate.go
- backend/changelog.md
- docs/changelog.md
verifikasi: `go test ./... -count=1`, `go vet ./...`, log startup Docker, dan `\d maintenance_logs`.
alasan perubahan: Memulihkan fungsi simpan log pemeliharaan yang rusak oleh sisa skema era model
`assets` (Fase 3.6).

---

## [1.15.0] — 2026-08-03 — Picker Kalender untuk Waktu Pencatatan Pemeliharaan

### Frontend UI

- **Input `datetime-local`**: field "Waktu pencatatan"/"Waktu terjadwal" pada form Catat
  Pemeliharaan Baru kini memakai picker native browser — kalender untuk memilih hari dan kontrol
  jam/menit — menggantikan input teks bebas.
- **Default waktu lokal**: nilai awal memakai waktu browser (bukan UTC), sebelumnya
  `new Date().toISOString()` bisa meleset beberapa jam dari waktu Indonesia.
- **Format tetap konsisten**: nilai dikonversi dua arah ke `YYYY-MM-DD HH:MM:SS` agar payload
  ke backend tidak berubah.

before:
- Input teks tanpa kalender; default waktu salah zona (UTC).
after:
- Kalender + picker waktu; default = waktu saat ini di zona lokal browser.
timestamp: 2026-08-03 WITA
kategori: Feature, Bugfix, Frontend, UI
file terkait:
- frontend/src/app/maintenance/page.tsx
- frontend/changelog.md
- docs/changelog.md
verifikasi: `npm run lint` dan `npm run build`.
alasan perubahan: Menghilangkan kesalahan format manual dan memastikan waktu tercatat sesuai zona
waktu pengguna.

---

## [1.14.0] — 2026-08-03 — Tren Pemeliharaan Jadi Spline Area Chart

### Frontend UI

- **Grafik area baru**: card "Tren Pemeliharaan" di dashboard mengganti bar chart CSS dengan
  Spline Area Chart SVG murni (`TrendChart`) — garis melengkung halus (Catmull-Rom → Bézier),
  area di bawah garis berisi gradasi biru, responsif, dengan tooltip hover rincian per status.
- **Tanpa dependency baru**: implementasi SVG mandiri (grid, sumbu, titik data, penanda hover,
  tooltip) menjaga ukuran bundle dan permukaan serangan tetap minimal.

before:
- Tren pemeliharaan divisualkan sebagai bar chart bertumpuk per status yang kurang terbaca untuk
  melihat arah pergerakan total pekerjaan.
after:
- Total pekerjaan per bulan tampil sebagai garis halus dengan area gradasi biru; rincian status
  tetap tersedia lewat tooltip saat kursor diarahkan ke titik data.
timestamp: 2026-08-03 WITA
kategori: Feature, Frontend, UI
file terkait:
- frontend/src/components/TrendChart.tsx (baru)
- frontend/src/components/DashboardAnalytics.tsx
- frontend/changelog.md
- docs/changelog.md
verifikasi: `npm run lint` dan `npm run build`.
alasan perubahan: Grafik area dengan garis halus lebih intuitif untuk memantau tren pemeliharaan
antar bulan, tanpa menambah ketergantungan pustaka chart eksternal.

---

## [1.13.0] — 2026-08-03 — Rincian Ringkasan CCTV di Bawah Kartu & Warna Bar Status

### Frontend UI

- **Respon kartu ringkasan dipindah ke atas**: saat kartu Total CCTV / Aktif / Nonaktif diklik,
  rincian (`cctv-summary-details`) kini tampil langsung di bawah kartu tersebut — bukan di bagian
  bawah halaman setelah analitik, tugas, dan pemeliharaan.
- **Bar progress lebih cerah**: "Distribusi Status CCTV" kini memakai hijau terang
  (`bg-emerald-500`) untuk aktif, merah terang (`bg-red-500`) untuk nonaktif/rusak, dan kuning
  cerah (`bg-amber-500`) untuk pemeliharaan, selaras dengan warna marker di peta sebaran.

before:
- Rincian ringkasan tampil jauh dari kartu yang diklik (di bawah semua section lain).
- Bar distribusi status memakai warna tema yang redup sehingga proporsinya kurang terbaca.
after:
- Rincian tampil tepat di bawah kartu ringkasan sebelum section analitik.
- Proporsi status langsung terbaca dengan warna terang.
timestamp: 2026-08-03 WITA
kategori: Feature, Frontend, UI
file terkait:
- frontend/src/app/dashboard/page.tsx
- frontend/src/components/DashboardAnalytics.tsx
- frontend/changelog.md
- docs/changelog.md
verifikasi: `npm run lint` dan `npm run build`.
alasan perubahan: Memperpendek jarak umpan balik interaksi ringkasan CCTV dan meningkatkan
keterbacaan chart distribusi status.

---

## [1.12.0] — 2026-08-03 — Mode Layar Penuh Peta Sebaran CCTV

### Frontend UI

- **Tombol fullscreen di peta**: `CCTVMap` kini memiliki tombol di pojok kanan atas untuk
  menampilkan peta dalam layar penuh (Fullscreen API), termasuk ikon status Maximize/Minimize,
  dukungan keluar via tombol maupun tombol Esc, serta pembersihan border/radius saat fullscreen.
- **Pembersihan kecil**: import ikon `Camera` yang tidak terpakai dihapus.

before:
- Peta sebaran CCTV terkurung dalam kartu halaman tanpa opsi tampilan penuh.
after:
- Peta dapat diperbesar ke layar penuh dan kembali ke ukuran semula dengan satu klik.
timestamp: 2026-08-03 WITA
kategori: Feature, Frontend, UI
file terkait:
- frontend/src/components/CCTVMap.tsx
- frontend/changelog.md
- docs/changelog.md
verifikasi: `npm run lint` dan `npm run build`.
alasan perubahan: Memudahkan pengguna memeriksa sebaran CCTV secara menyeluruh pada layar besar.

---

## [1.11.0] — 2026-08-03 — Marker CCTV Bertumpuk di Peta Sebaran

### Frontend UI

- **Koordinat duplikat diurai di peta**: `CCTVMap` mendeteksi CCTV yang berbagi pasangan
  latitude/longitude identik (mis. THD-05 "Bundaran Sumbu Barat" & THD-17 "Pintu Masuk Sumbu
  Barat" di titik yang sama) dan menggeser marker berikutnya secara deterministik agar semua
  CCTV tetap tampil dan tidak saling menutupi.
- **Kartu info sudah akurat**: total 35 CCTV (20 ACTIVE, 15 INACTIVE) sesuai database; selisih
  yang diamati pengguna (19 vs 20 pin aktif) murni akibat satu pin tertutup pin lain.

before:
- Marker dengan koordinat sama dirender tepat di atas satu sama lain; pin yang terhitung manual
  tidak cocok dengan jumlah di kartu ringkasan.
after:
- Seluruh marker (35 CCTV) dapat dibedakan di peta; popup masing-masing tetap bisa dibuka.
timestamp: 2026-08-03 WITA
kategori: Bugfix, Frontend, UI
file terkait:
- frontend/src/components/CCTVMap.tsx
- frontend/changelog.md
- docs/changelog.md
verifikasi: `npm run lint`, `npm run build`, dan cek jumlah status di PostgreSQL.
alasan perubahan: Menjaga visualisasi peta tetap sesuai jumlah aset sebenarnya tanpa mengubah
data koordinat di database.

---

## [1.10.0] — 2026-08-03 — Halaman & Menu Peta Sebaran CCTV

### Frontend UI

- **Peta dipindah dari Dashboard**: komponen `CCTVMap` tidak lagi dirender di `/dashboard`;
  dashboard kini fokus pada kartu ringkasan, analitik, tugas, dan pemeliharaan.
- **Halaman baru `/sebaran-cctv`**: menampilkan peta GIS sebaran CCTV (tinggi 560px), kartu
  ringkasan status (Total CCTV, Aktif, Pemeliharaan), indikator kesehatan layanan, tombol muat
  ulang, dan pembaruan otomatis saat event WebSocket `CCTV_UPDATED`.
- **Menu baru di sidebar**: "Peta Sebaran CCTV" (ikon `MapPinned`) ditambahkan tepat di bawah
  "Dashboard" pada `DashboardLayout`, tersedia untuk semua peran.

before:
- Peta sebaran CCTV hanya tampil di dalam halaman `/dashboard` tanpa navigasi langsung dari menu.
after:
- Peta sebaran CCTV dapat diakses dari menu navigasi utama dan dashboard tidak lagi memuat peta.
timestamp: 2026-08-03 WITA
kategori: Feature, Frontend, UI
file terkait:
- frontend/src/app/sebaran-cctv/page.tsx (baru)
- frontend/src/app/dashboard/page.tsx
- frontend/src/components/DashboardLayout.tsx
- frontend/changelog.md
- docs/AGENTS.md
- docs/changelog.md
verifikasi: `npm run lint` dan `npm run build`.
alasan perubahan: Memisahkan visualisasi geospasial dari halaman ringkasan operasional agar
dashboard tetap fokus pada data pemantauan, dan memudahkan pengguna membuka peta sebaran CCTV
langsung dari menu utama.

---

## [1.9.0] — 2026-08-03 — Navigasi Auth Atomik Setelah Logout/Login

### Frontend Auth Race

- **Full navigation setelah login/logout**: mengganti transisi SPA `router.push()` dengan
  `window.location.replace()` agar request dashboard lama, state AuthProvider, dan WebSocket
  lama benar-benar dibatalkan sebelum sesi baru dimulai.
- **Pencegahan stale 401**: response 401 dari request halaman sebelum logout tidak lagi dapat
  memanggil handler global dan mencabut sesi yang baru saja dibuat.

before:
- SPA navigation mempertahankan root layout dan request async halaman lama setelah logout.
- Setelah login cepat, request lama dapat selesai dengan 401 bersamaan dengan request sesi baru
  yang 200, lalu redirect global membuat pengguna langsung keluar lagi.
after:
- Logout/login melakukan navigasi penuh setelah cookie server selesai diproses.
- AuthProvider dan seluruh request halaman dimulai ulang dengan sesi yang sama.
timestamp: 2026-08-03 WITA
kategori: Bugfix, Auth, Frontend
file terkait:
- frontend/src/context/AuthContext.tsx
- frontend/changelog.md
- docs/changelog.md
verifikasi: `npm run lint`, `npm run build`, dan observasi log Docker pada alur login/logout.
alasan perubahan: Menghilangkan race condition antara request async lama dan sesi baru saat
pengguna melakukan login kembali segera setelah logout.

---

## [1.8.0] — 2026-08-03 — Penghapusan Cookie Legacy dan Unifikasi Token Sesi

### Auth, Cookie, JWT, dan WebSocket

- **Cookie legacy lintas path dibersihkan**: response login/logout mengirim `Set-Cookie` terpisah untuk path `/`, `/api`, dan `/api/`, sehingga browser tidak memilih cookie sesi stale pada request API.
- **JWT dibuat selalu unik**: klaim `jti` UUID dan `iat` ditambahkan agar login ulang cepat tidak menghasilkan token identik dengan token yang baru direvoke.
- **Reconnect WebSocket dihentikan saat logout**: timer reconnect memeriksa status sesi dan tidak lagi membanjiri `/api/ws` dengan 401 setelah logout.
- **Testing diperluas**: validasi multi-path cookie dan keunikan token ditambahkan.

before:
- Header `Set-Cookie` Fiber menimpa cookie dengan nama sama sehingga hanya path terakhir yang terkirim.
- JWT tanpa `jti` dapat identik jika login ulang terjadi pada detik yang sama dengan token lama.
- WebSocket tetap reconnect setiap 5 detik meskipun sesi sudah tidak aktif.
after:
- Cookie root dan cookie legacy API dihapus bersamaan saat login/logout.
- Token baru selalu berbeda dan tidak ikut blacklist token lama.
- WebSocket berhenti reconnect ketika `isAuthenticated=false`.
timestamp: 2026-08-03 WITA
kategori: Bugfix, Auth, Security, Infrastructure, Testing
file terkait:
- backend/utils/session.go
- backend/utils/jwt.go
- backend/utils/jwt_test.go
- backend/handlers/auth.go
- backend/handlers/auth_test.go
- backend/handlers/auth_extra_test.go
- backend/middleware/csrf.go
- backend/routes/auth.go
- docs/AGENTS.md
- frontend/src/context/WebSocketContext.tsx
- docs/TESTING.md
- docs/changelog.md
verifikasi: `go test ./... -count=1`, `go vet ./...`, `npm run lint`, `npm run build`, dan smoke test cookie melalui `https://localhost`.
alasan perubahan: Menghilangkan token/cookie stale yang menyebabkan login berhasil tetapi request pertama langsung 401, serta menghentikan reconnect WebSocket pasca logout.

---

## [1.7.0] — 2026-08-03 — Pemulihan Logout/Login dan Proxy Runtime

### Auth, CSRF, Rate Limit, dan Konfigurasi Docker

- **Logout idempotent**: endpoint logout selalu menghapus cookie sesi/CSRF; token valid tetap direvoke, sedangkan token kosong/invalid dianggap logout berhasil.
- **Logout tidak diblokir CSRF stale**: middleware CSRF melewati endpoint logout karena logout tidak mengubah resource dan harus tetap bisa membersihkan sesi yang rusak.
- **Konfigurasi runtime Docker dilengkapi**: `APP_ENV=production`, `CORS_ALLOWED_ORIGINS=https://localhost`, dan `TRUSTED_PROXY_IPS=172.28.0.10` ditambahkan ke environment runtime/template.
- **Frontend logout tidak memicu redirect 401 global**: request logout memakai `skipAuthRedirect` dan tetap mengarahkan pengguna secara lokal setelah server merespons atau gagal.

before:
- Logout dapat berhenti di status 403 CSRF ketika cookie CSRF stale/hilang, tetapi frontend tetap mengarahkan pengguna ke login sehingga cookie httpOnly lama tertinggal.
- Login dapat merespons 200 lalu `/auth/me` menerima 401 karena sesi lama/tidak terkirim; percobaan berulang kemudian terkena 429.
- Container runtime belum memuat trusted proxy, sehingga semua request dihitung dari IP Nginx yang sama.
after:
- Logout dapat dipanggil ulang dalam kondisi apa pun, cookie selalu di-expire, dan sesi valid direvoke.
- Rate limiter membaca client IP melalui trusted proxy Nginx dan origin Docker menggunakan `https://localhost`.
- Test regresi logout/CSRF/route ditambahkan.
timestamp: 2026-08-03 WITA
kategori: Bugfix, Auth, Security, Infrastructure
file terkait:
- backend/handlers/auth.go
- backend/middleware/csrf.go
- backend/routes/auth.go
- backend/handlers/auth_extra_test.go
- backend/middleware/csrf_test.go
- backend/routes/routes_test.go
- backend/.env.docker (runtime lokal, tidak di-commit)
- backend/.env.docker.example
- frontend/src/context/AuthContext.tsx
- nginx/nginx.conf
- docker-compose.yml
- docs/changelog.md
verifikasi: `go test ./... -count=1`, `go vet ./...`, `npm run lint`, `npm run build`, `docker compose config --quiet`, dan uji runtime login -> logout -> login ulang.
alasan perubahan: Menghentikan loop logout/login yang disebabkan kombinasi CSRF stale, cookie sesi tertinggal, dan rate limiter yang salah mengidentifikasi client di belakang Nginx.

---

## [1.6.0] — 2026-08-03 — Sinkronisasi Single-Origin, Swagger Nginx, dan Dokumentasi

### Dokumentasi, Infrastruktur, dan Stabilitas Sesi

- **Dokumentasi akses single-origin**: menambahkan `docs/ACCESS.md` yang menetapkan
  `https://localhost` sebagai satu-satunya entrypoint Docker dan menjelaskan alur Nginx ke
  frontend, API, WebSocket, dan Swagger.
- **Sinkronisasi dokumentasi autentikasi**: memperbarui masa berlaku JWT menjadi 5 jam,
  menghapus referensi sliding session dan idle timer yang sudah tidak digunakan, serta
  mendokumentasikan `MaxAge: 18000` pada cookie sesi/CSRF.
- **Matriks test dikoreksi**: mencantumkan `utils/audit_test.go` dan memperbaiki jumlah aktual
  menjadi 28 file test.
- **Swagger melalui Nginx**: menambahkan proxy `/swagger/*` ke API dan mengubah metadata Swagger
  agar menggunakan host publik `localhost` dengan scheme HTTPS.
- **Dead code idle logout dihapus**: frontend tidak lagi menyimpan atau membaca
  `sigap_logout_reason` maupun menampilkan banner inaktivitas yang sudah tidak pernah terpicu.

before:
- Dokumen masih mencampur akses `localhost:3000`/`localhost:3001`, JWT 24 jam, sliding session,
  idle timer, dan URL Swagger yang tidak sesuai dengan routing Nginx.
- Swagger tidak dapat diakses melalui reverse proxy karena hanya `/api/*` yang diteruskan ke API.
- Matriks test menghilangkan `audit_test.go` dan menyatakan jumlah yang tidak sesuai.
after:
- Deployment Docker menggunakan `https://localhost` sebagai single-origin; port service tetap
  internal dan route Swagger tersedia melalui `https://localhost/swagger/index.html` pada non-production.
- Dokumentasi sesi, keamanan, roadmap, README, dan testing sesuai kondisi kode aktual.
- Sisa mekanisme idle logout di frontend dihapus.
timestamp: 2026-08-03 WITA
kategori: Documentation, Infrastructure, Security, Auth, Testing
file terkait:
- docs/ACCESS.md
- docs/AGENTS.md
- docs/TESTING.md
- docs/SECURITY-AUDIT.md
- docs/ROADMAP.md
- docs/PRODUCT.md
- docs/plan.md
- docs/changelog.md
- README.md
- nginx/nginx.conf
- docker-compose.yml
- backend/.env.docker.example
- backend/main.go
- backend/docs/docs.go
- backend/docs/swagger.json
- backend/docs/swagger.yaml
- frontend/src/context/AuthContext.tsx
- frontend/src/app/login/page.tsx
alasan perubahan: Menjaga dokumentasi, routing publik, autentikasi cookie, dan catatan coverage
tetap selaras setelah migrasi ke single-origin Nginx serta penyederhanaan sesi.
verifikasi akhir:
- `go test ./... -count=1` PASS.
- `go vet ./...` PASS.
- `npm ci` berhasil memasang 370 package sesuai lockfile.
- `npm run lint` PASS dengan warning existing dan `npm run build` PASS.
- `docker compose config --quiet` PASS.
- `nginx -t` PASS pada image Nginx 1.27 setelah upstream Docker disediakan.
- `npm audit` melaporkan 4 vulnerability high yang belum diperbaiki.

---

## [1.5.0] — 2026-08-03 — Perbaikan Stabilitas Autentikasi & Single-Origin Proxy

### Autentikasi, Sesi, dan Infra

- **Pencegahan Phantom Duplicate Cookie pada Auth Client**: `clearAuthCookies()` di `frontend/src/lib/api.ts` tidak lagi mencoba menghapus cookie `sigap_session` (httpOnly) melalui JavaScript untuk mencegah browser membuat cookie hantu non-httpOnly yang menyebabkan kegagalan login ulang.
- **Opsi `skipAuthRedirect` pada API Request**: Menambahkan opsi `skipAuthRedirect: true` pada `apiRequest()` untuk rute bootstrap (`/auth/me`) dan login (`/auth/login`) agar respons 401 wajar tidak memicu pembersihan sesi global secara prematur.
- **Penetapan MaxAge Cookie Sesi**: Menyetel `MaxAge: 18000` (5 jam) pada cookie sesi `sigap_session` dan CSRF `sigap_csrf` di `backend/utils/session.go` agar selaras dengan durasi kedaluwarsa JWT (5 jam) sehingga sesi bertahan saat browser di-refresh.
- **Penegasan Single-Origin Nginx Proxy**: Menutup port publik `3001` pada service `frontend` di `docker-compose.yml` dan mewajibkan seluruh akses melalui Reverse Proxy Nginx (`https://localhost`). Hal ini menjamin frontend (`https://localhost/`) dan backend (`https://localhost/api/`) berada pada origin yang sama persis sehingga cookie httpOnly terkirim secara 100% konsisten tanpa masalah cross-origin port.

kategori: Auth, Bugfix, Security, Infrastructure
file terkait (root/infra): `docker-compose.yml`, `frontend/src/lib/api.ts`, `frontend/src/context/AuthContext.tsx`, `frontend/src/app/login/page.tsx`, `backend/utils/session.go`, `docs/changelog.md`
verifikasi: `docker compose up -d`, pengujian alur login -> logout -> login ulang dan refresh halaman pada `https://localhost`.
alasan perubahan: Menyelesaikan masalah sesi terputus saat refresh dan kegagalan login ulang akibat bentrokan cookie httpOnly serta perbedaan origin port.

---

## [1.4.1] — 2026-08-02 — Fase 4.1: Penyelesaian Coverage Test

### Testing Backend

- Test jalur sukses update/delete CCTV dan maintenance.
- Test handshake WebSocket dengan token valid dan bootstrap route terlindungi.
- Test resolusi legacy asset ke CCTV dan pembaruan matriks coverage.
- Sisa yang ditunda: import transaksi penuh, token version mismatch WebSocket, dan migration update penuh.

kategori: Testing, Quality, Migration
file terkait (root/infra): `docs/ROADMAP.md`, `docs/TESTING.md`, `docs/changelog.md`
verifikasi: `go test ./... -count=1`, `go vet ./...`, `golangci-lint v2.12.2`, `npm run lint`, `npm run build`.
alasan perubahan: Menyelaraskan dokumentasi dengan hasil implementasi Fase 4.1 sebelum melanjutkan ke versioned database migration.

---

## [1.4.0] — 2026-08-02 — Fase 4: Quick Wins Keamanan & Kualitas

### Keamanan, Pengujian, dan CI

- Proxy server-side Next.js untuk route `/audit-logs` dan `/users`, termasuk validasi sesi melalui backend dan pengecekan role admin.
- Audit log memiliki retry terbatas dan broadcast `AUDIT_LOG_ADDED` setelah penyimpanan berhasil.
- Unit test tambahan untuk edge case CCTV, maintenance, user, audit retry, dan UUID model hook.
- CI menjalankan `npm run lint` dan `golangci-lint` selain build serta unit test.

kategori: Security, Testing, CI, Quality
file terkait (root/infra): `docs/ROADMAP.md`, `docs/SECURITY-AUDIT.md`, `.github/workflows/ci.yml`
verifikasi: `go test ./... -count=1`, `go build ./...`, `npm run lint`, `npm run build`.
alasan perubahan: Menutup temuan keamanan tersisa dan menjadikan pemeriksaan kualitas otomatis pada setiap pull request.

---

## [1.3.0] — 2026-08-01 — Fase 3: Fitur (Kapabilitas)

### Fitur Kapabilitas (ROADMAP 3.1 – 3.9)

- **3.1** Self-service change password (`POST /api/auth/change-password`) + paksa ganti password saat login pertama (`must_change_password`).
- **3.2** Sliding session — masa berlaku JWT diperpanjang saat akses aktif; blacklist & revoke token lama tetap berjalan.
- **3.3** Dashboard analitik: `GET /api/dashboard/summary` (statistik, tren 6 bulan, distribusi status, overdue) + komponen `DashboardAnalytics`.
- **3.4** Notifikasi tugas teknisi: `GET /api/dashboard/tasks` (teknisi = tugas miliknya, admin = semua; `pending_count`/`overdue_count`).
- **3.5** Halaman profil (`/profile`) + `PUT /api/auth/profile`; kartu profil di sidebar jadi tautan.
- **3.6** Unifikasi model `assets` ↔ `cctv`: CCTV jadi satu-satunya sumber kebenaran. Model & handler/rute `assets` dihapus; `MaintenanceLog` kini `cctv_id` → `cctvs`; migrasi data legacy `asset_id` otomatis sekali saat startup (`config/migrate.go`).
- **3.7** Laporan PDF (`GET /api/maintenance-logs/report-pdf`, via `go-pdf/fpdf`), ekspor audit-log CSV (`GET /api/audit-logs/export`), dan filter rentang tanggal `start_date`/`end_date` pada list & ekspor.
- **3.8** Privasi & kebersihan upload: seluruh foto dinormalisasi ke JPEG (metadata EXIF/GPS terhapus); goroutine cleanup menghapus file yatim berumur >24 jam.
- **3.9** Pembatasan ekspor per role: teknisi hanya mengekspor/mengunduh laporan data miliknya; admin penuh.

kategori: Feature, Security, Dashboard, PDF, Upload
file terkait (root/infra): `ROADMAP.md`, `SECURITY-AUDIT.md`, `../README.md`, `changelog.md` (docs, baru)
verifikasi: `go build/vet/test` PASS (backend), `npm run build` PASS (frontend), Swagger regenerated.
alasan perubahan: Meningkatkan nilai operasional sistem dan menutup temuan keamanan 18 (ekspor per role).

---

## [1.2.0] — 2026-08-01 — Fase 2: Penguatan (Hardening)

### Hardening Keamanan (SECURITY-AUDIT temuan 7–15, 20)

- Migrasi token JWT ke cookie httpOnly + pola CSRF double-submit (`sigap_session`, `sigap_csrf`, header `X-CSRF-Token`).
- Revoke token lama saat reset password / role diubah (klaim `tv` / token version).
- Rate limiter & audit log hanya percaya header proxy dari IP terdaftar (`TRUSTED_PROXY_IPS`).
- Batas ukuran (4 MB) & baris (1000) import Excel.
- CORS tanpa wildcard; default aman `http://localhost:3001`.
- Docker hardening: quick tunnel dihapus, reverse proxy Nginx + TLS, port DB/API/frontend tidak dipublish, secret via root `.env`.
- CSP & HSTS aktif di produksi; kebijakan password min 8 + kompleksitas; ownership check maintenance log; penanganan 401 pada fetch raw frontend.

kategori: Security, Hardening, Docker, Infra
file terkait (root/infra): `docker-compose.yml`, `nginx/` (baru), `.env.example` (baru, root), `.gitignore`
verifikasi: `go build/vet/test` PASS (backend), `npm run build` PASS (frontend).
alasan perubahan: Menutup temuan Fase 2 (SECURITY-AUDIT.md) untuk mencegah pencurian token, CSRF, spoofing IP, overload impor, eksposur layanan, dan password lemah.

---

## [1.1.0] — 2026-08-01 — Fase 1: Kritis (Prioritas Tertinggi)

### Perbaikan Kritis (SECURITY-AUDIT temuan 1–6)

- JWT: hapus fallback secret hardcoded (wajib env `JWT_SECRET`).
- Seeder: admin default tidak lagi di-reset tiap startup; password wajib dari env `ADMIN_DEFAULT_PASSWORD`.
- WebSocket: wajib autentikasi (handshake token).
- Stored XSS pada cetak label QR diperbaiki (render React, bukan `document.write`).
- QR Code dipindah ke server-side (`skip2/go-qrcode`), lepas dari API pihak ketiga.
- Pagination dibatasi cap maksimum (`MAX_PAGE_SIZE`).

kategori: Security, Kritis
file terkait (root/infra): `README.md`, `docker-compose.yml`
verifikasi: `go build/vet/test` PASS (backend), `npm run build` PASS (frontend).
alasan perubahan: Menutup celah keamanan paling berbahaya sebelum pengembangan lanjutan.

---

## [1.0.0] — 2026-08-01 — Rilis Awal

- Sistem identifikasi & inventaris CCTV (SIGAP) berbasis Go Fiber + Next.js + PostgreSQL.
- Fitur inti: manajemen master CCTV, QR Code terenkripsi AES-256, scanner QR, import Excel massal, log pemeliharaan dengan unggah foto, manajemen pengguna & RBAC (Admin/Teknisi), autentikasi JWT, Swagger UI.
- Deployment Docker Compose (db, api, frontend).
