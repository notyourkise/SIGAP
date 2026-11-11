# TODO Perbaikan Fitur SIGAP

**Tanggal:** 2026-10-27 *(diperbarui status: 2026-11-11)*  
**Basis commit awal:** `8819c3e` (`main`, merge PR #137)  
**Status:** Implementasi teknis backlog selesai (#139–#146, #149–#155); tersisa eksekusi rilis/lapangan eksternal (#147 Bug 8/manual UI gate, #148 VPS deploy, survei koordinat fisik 9 titik #151, dan rotasi fisik NVR/off-site #153)  
**Ruang lingkup:** Perbaikan dan hardening fitur yang sudah ada, bukan penambahan modul baru  
**Prinsip:** P0 wajib selesai sebelum multi-site/production; setiap item harus memiliki test, changelog, dan bukti verifikasi

## Legenda

| Kode | Arti |
|---|---|
| P0 | Kritis; blocker multi-site atau production |
| P1 | Tinggi; koreksi fitur dan kesiapan rilis |
| P2 | Sedang/rendah; hardening dan operasional |
| S | Effort kecil, kurang dari 1 hari |
| M | Effort sedang, 1-3 hari |
| L | Effort besar, lebih dari 3 hari |

Estimasi effort bersifat indikatif, bukan komitmen waktu.

---

## Ringkasan Prioritas

| ID | Pekerjaan | Prioritas | Effort |
|---|---|---|---|
| P0-01 | Isolasi site pada Log CCTV dan event WebSocket | Kritis | M |
| P0-02 | Hardening trusted proxy dan IP klien | Kritis | M |
| P0-03 | Test regresi keamanan multi-site/proxy | Kritis | S |
| P1-01 | Timezone dan konsistensi ringkasan log | Tinggi | S |
| P1-02 | Perbaikan UX halaman Log CCTV | Tinggi | S |
| P1-03 | Retensi dan index `cctv_stream_logs` | Tinggi | M |
| P1-04 | Validasi dan error handling endpoint log | Tinggi | S |
| P1-05 | Kebijakan event standby dan duplikasi | Sedang | S |
| P1-06 | Selesaikan manual test Bug 4, 8, 21 | Tinggi | M |
| P1-07 | Deployment dan smoke test production | Tinggi | M |
| P2-01 | Kompatibilitas H.265 dan snapshot | Sedang | M |
| P2-02 | Diagnostik Metro-E dan false offline | Sedang | S |
| P2-03 | Verifikasi 9 koordinat CCTV | Sedang | M |
| P2-04 | Sinkronisasi dokumentasi dan status | Sedang | S |
| P2-05 | Item keamanan residual | Sedang | M |
| P2-06 | Observability dan alert status kamera | Rendah | M |
| P2-07 | CI/E2E regression suite | Rendah | M |

---

## P0 - Wajib Sebelum Multi-Site / Production

### [x] P0-01 - Isolasi site pada Log CCTV dan WebSocket (#139)

**Masalah:** `GetCCTVStreamLogs` menerima `SiteContext`, tetapi tidak memakai `c.Locals("site_id")`. User non-admin di site A dapat melihat log site B. Event `CCTV_STREAM_LOG_CREATED` juga dikirim global ke semua koneksi WebSocket.

**Pekerjaan:**
- [x] Tambahkan scope site pada query count, list, dan ringkasan log.
- [x] Gunakan pola `EXISTS` ke `cctvs` seperti `applyMaintenanceSiteScope`, atau join yang setara.
- [x] Pastikan user non-admin tanpa site assignment tidak melihat data apa pun.
- [x] Tambahkan `site_id` pada payload event WebSocket.
- [x] Batasi broadcast event ke site terkait.
- [x] Audit event global lain yang membawa nama/kode CCTV, terutama `CCTV_LIVE_STATUS_UPDATED`.
- [x] Tentukan kebijakan akses: teknisi diizinkan dengan scope site (`GET /api/cctv/logs` & `GET /api/cctv-logs`).
- [x] Tambahkan regression test dua site dan dua user.

**File utama:**
`backend/handlers/cctv_stream_log.go`, `backend/routes/cctv.go`, `backend/utils/websocket.go`, `backend/handlers/stream_probe.go`, `backend/handlers/stream_verify.go`, `frontend/src/app/cctv/logs/page.tsx`.

**Acceptance criteria:**
- [x] User site A mendapat 403/data kosong saat mengakses log site B.
- [x] Event WebSocket site B tidak diterima user site A.
- [x] Admin tetap dapat melihat seluruh site atau site yang dipilih.
- [x] Test regresi lulus.

**Catatan desain:** Untuk jangka panjang, pertimbangkan kolom `site_id` pada `cctv_stream_logs` sebagai snapshot site saat event terjadi. Jika kamera dipindahkan antar-site, log lama tetap berada pada site historisnya.

---

### [x] P0-02 - Hardening trusted proxy dan IP klien (#140)

**Masalah:** `ProxyHeader` selalu diisi, tetapi `EnableTrustedProxyCheck` hanya aktif jika `TRUSTED_PROXY_IPS` diisi. Pada Fiber v2, kondisi ini membuat `X-Forwarded-For` dari client langsung tetap dipercaya. `EnableIPValidation` juga belum aktif.

**Pekerjaan:**
- [x] Set `ProxyHeader` hanya jika `TRUSTED_PROXY_IPS` valid dan tidak kosong.
- [x] Aktifkan validasi IP.
- [x] Pastikan `getClientIP` dan audit log memakai IP hasil validasi, bukan seluruh isi header XFF.
- [x] Ubah Nginx agar menimpa `X-Forwarded-For` dengan `$remote_addr`, bukan `$proxy_add_x_forwarded_for`.
- [x] Tambahkan test: request langsung dengan XFF palsu tidak boleh mengubah rate-limit bucket atau audit IP.
- [x] Tambahkan test: request dari trusted proxy memakai IP client yang benar.

**File utama:**
`backend/main.go`, `backend/config/fiber.go`, `backend/middleware/limiter.go`, `nginx/nginx.conf`, `backend/middleware/limiter_test.go`.

**Acceptance criteria:**
- [x] Client langsung tidak bisa memalsukan IP.
- [x] Rate limit tidak bisa dilewati dengan mengganti XFF.
- [x] Audit log mencatat IP yang benar.
- [x] Deployment tanpa proxy tidak memakai header XFF.

---

### [x] P0-03 - Test regresi keamanan multi-site/proxy (#141)

**Pekerjaan:**
- [x] Test API log CCTV untuk admin, teknisi site A, dan teknisi site B.
- [x] Test WebSocket event tidak bocor lintas site.
- [x] Test trusted vs untrusted proxy.
- [x] Test rate limit per endpoint dan per IP.
- [x] Jalankan `go test ./... -count=1`, `go vet ./...`, dan `golangci-lint run ./...`.

**Acceptance criteria:**
- [x] Semua test baru lulus.
- [x] Tidak ada regresi pada test lama.
- [x] Hasil test dicatat di `docs/05-reports/SECURITY-REGRESSION-2026-10-27.md`.

---

## P1 - Sprint Berikutnya

### [x] P1-01 - Timezone dan konsistensi ringkasan log (#142)

**Masalah:** `startOfDay` memakai timezone proses, sedangkan komentar menyebut UTC. Container umumnya UTC, operasional memakai WITA. Ringkasan "hari ini" juga global, tidak mengikuti site/filter.

**Pekerjaan:**
- [x] Tambahkan konfigurasi `APP_TIMEZONE=Asia/Makassar` dan helper timezone eksplisit.
- [x] Hitung rentang hari dengan batas bawah dan batas atas (`[start, end)`).
- [x] Terapkan site/filter yang sama pada query ringkasan.
- [x] Selaraskan label UI: "Total hasil filter", "Event hari ini pada site/filter aktif".
- [x] Tambahkan unit test untuk batas tengah malam WITA.

**Acceptance criteria:**
- [x] Event pukul 00:30 WITA masuk hari yang benar.
- [x] Angka kartu konsisten dengan daftar dan filter.
- [x] Tidak ada perbedaan hasil antara server UTC dan WITA.

---

### [x] P1-02 - Perbaikan UX halaman Log CCTV (#143)

**Pekerjaan:**
- [x] Tambahkan filter `unknown`/"Tidak Diketahui".
- [x] Debounce pencarian 300 ms agar tidak fetch setiap ketikan.
- [x] Reset page ke 1 saat filter/pencarian berubah.
- [x] Simpan filter di URL query agar bisa dibagikan/refresh.
- [x] Tambahkan tombol "Coba lagi" saat error.
- [x] Tampilkan "Terakhir diperbarui" dan indikator loading yang tidak menutupi data lama.
- [x] Pastikan empty state membedakan "belum ada log" dan "tidak ada hasil filter".

**Acceptance criteria:**
- [x] Mengetik pencarian tidak membanjiri API.
- [x] Filter unknown berfungsi.
- [x] Refresh halaman mempertahankan filter.
- [x] Error state bisa dipulihkan tanpa reload penuh.

---

### [x] P1-03 - Retensi dan index `cctv_stream_logs` (#144)

**Pekerjaan:**
- [x] Tambahkan worker cleanup harian `CCTV_STREAM_LOG_RETENTION_DAYS=90`.
- [x] Hapus batch 1.000 baris agar tidak mengunci tabel.
- [x] Tambahkan index komposit `(cctv_id, created_at)` dan `(event_type, created_at)` via migrasi `000029`.
- [x] Tambahkan test cleanup dan konfigurasi default.
- [x] Dokumentasikan retensi dan cara audit log lama.

**Acceptance criteria:**
- [x] Log lebih tua dari batas terhapus.
- [x] Log baru tetap aman.
- [x] Worker idempotent dan tidak membebani database.
- [x] Tidak ada pertumbuhan tabel tanpa batas.

---

### [x] P1-04 - Validasi dan error handling endpoint log (#145)

**Pekerjaan:**
- [x] Validasi `event_type` hanya `online`, `offline`, `standby`, dan `unknown`; nilai lain menghasilkan HTTP 400.
- [x] Periksa error pada query count ringkasan; jangan diam-diam menampilkan 0.
- [x] Gunakan konstanta event type (`models.CCTVEvent*`) agar tidak ada typo.
- [x] Validasi rentang tanggal dan parameter query tidak valid (`400`).
- [x] Tambahkan test untuk query error dan event type invalid.

**Acceptance criteria:**
- [x] Input invalid menghasilkan pesan jelas.
- [x] Kegagalan database tidak menampilkan angka palsu.
- [x] Test handler mencakup skenario error.

---

### [x] P1-05 - Kebijakan event standby dan duplikasi (#146)

**Masalah:** Probe 2 menit dan verifier 15 menit dapat menghasilkan log `standby`/`unknown` yang terlalu sering.

**Pekerjaan:**
- [x] Tetapkan `standby` sebagai status UI; pencatatan event dinonaktifkan default (`CCTV_LOG_STANDBY_EVENTS=false`).
- [x] Tambahkan deduplikasi window 60 detik lintas probe/verifier.
- [x] Cegah duplikasi event `online`/`offline` dari probe dan verifier dalam waktu singkat.
- [x] Dokumentasikan semantics `probe` vs `verify` di `docs/01-documentation/cctv-stream-log-semantics.md`.

**Acceptance criteria:**
- [x] Tidak ada banjir log saat kamera idle.
- [x] Event penting online/offline tetap tercatat.
- [x] Keputusan produk terdokumentasi.

---

### [ ] P1-06 - Selesaikan manual test Bug 4, 8, 21

**Pekerjaan:**
- [ ] Bug 4: verifikasi filter kawasan aset CCTV melalui UI.
- [ ] Bug 8: verifikasi MediaMTX pada production, jaringan publik, dan seluler.
- [ ] Bug 21: verifikasi edit CCTV + URL RTSP melalui UI.
- [ ] Simpan bukti screenshot/log/video.
- [ ] Perbarui `docs/README.md` hanya setelah lulus.

**Acceptance criteria:**
- [ ] Checklist menjadi 25/25, atau ada risiko yang diterima secara eksplisit.
- [ ] Bukti manual test terlampir.
- [ ] Tidak ada klaim "production ready" sebelum gate ini selesai.

---

### [ ] P1-07 - Deployment dan smoke test production

**Pekerjaan:**
- [ ] Terapkan migrasi hingga versi terbaru (`000035`, mencakup `27`, `28`, `29`, dan skema insiden/idempotensi).
- [ ] Set env: `TRUSTED_PROXY_IPS`, `CCTV_VERIFY_INTERVAL_MINUTES`, `CCTV_VERIFY_CONCURRENCY`, `MEDIAMTX_WEBRTC_ADDITIONAL_HOSTS`, `CCTV_METRO_E_DIRECT_PROBE=false`.
- [ ] Rebuild image Docker setelah merge.
- [ ] Smoke test: login, rate limit, live status, manual verify, halaman log, WebSocket, dan multi-site.
- [ ] Siapkan rollback plan untuk migrasi 28.

**Acceptance criteria:**
- [ ] Semua smoke test lulus.
- [ ] Tidak ada error 429 palsu.
- [ ] Log dan status live tampil benar di production.
- [ ] Rollback migration diuji di staging.

---

## P2 - Hardening dan Operasional

### [x] P2-01 - Kompatibilitas H.265 dan snapshot (#149)

- [x] Deteksi dukungan HEVC di browser; tampilkan pesan yang jelas jika tidak didukung.
- [x] Sediakan fallback ke main stream/H.264 atau panduan perangkat.
- [x] Nonaktifkan tombol snapshot jika endpoint MediaMTX 1.20 tidak tersedia; jangan tampilkan UI rusak.
- [x] Dokumentasikan batasan codec dan snapshot.

### [x] P2-02 - Diagnostik Metro-E dan false offline (#150)

- [x] Tambahkan pengecekan startup: jika `CCTV_METRO_E_DIRECT_PROBE=true` tetapi `10.20.105.x` tidak reachable, log warning dan fallback ke status pasif.
- [x] Tambahkan indikator health/diagnostic di endpoint admin (`GET /api/streams/diagnostics`).
- [x] Pastikan hasil tidak pernah menampilkan offline palsu (`resolveLiveDisplayState`).

### [ ] P2-03 - Verifikasi 9 koordinat CCTV (#151)

- [ ] Verifikasi lapangan: `THD-18`, `THD-20`, `THD-25`, `THD-27`, `THD-29`, `THD-30`, `THD-31`, `THD-34`, `THD-35` *(aksi survei fisik GPS eksternal)*.
- [ ] Update latitude/longitude setelah hasil ukur GPS lapangan tersedia *(aksi operator/admin pasca survei)*.
- [ ] Hapus label "Estimasi" setelah terverifikasi fisik.
- [x] Tambahkan laporan data quality koordinat (`GET /api/cctv/coordinate-quality` JSON/CSV + UI modal Kualitas Koordinat di `/cctv`).

### [x] P2-04 - Sinkronisasi dokumentasi dan status (#152)

- [x] Ubah Next.js 15 menjadi 16.4.2.
- [x] Ubah "22+ migration" menjadi jumlah aktual 35 pasangan file migration (`000001`–`000035`).
- [x] Perbarui tabel endpoint, termasuk `/api/cctv-logs`, `/api/cctv/logs`, `/api/cctv/coordinate-quality`, `/api/streams/diagnostics`, `/api/incidents/*`, `/api/ai-assistant/ask`.
- [x] Selaraskan klaim `ROADMAP.md` "Production Ready" dengan status Staging / Pre-Release Pilot dan gate manual test.
- [x] Perbarui path proyek lama di `docs/README.md`.
- [x] Dokumentasikan retensi log (90 hari), timezone (`Asia/Makassar`), dan site scoping (`X-Site-ID`).

### [ ] P2-05 - Item keamanan residual (#153)

- [ ] Rotasi kredensial NVR fisik dan verifikasi di perangkat lapangan *(nilai nyata sudah dihapus dari source/test/dokumentasi; rotasi fisik merupakan aksi tim infrastruktur)*.
- [ ] Verifikasi backup terenkripsi dan salinan off-site di VPS *(image `backup` dengan `gnupg` + skrip GPG headless & compose env sudah diperbaiki; salinan off-site VPS dijalankan saat deployment `#148`)*.
- [x] Putuskan risiko temuan `npm audit` dev-only (`zod` high sudah diperbaiki; `npm audit --omit=dev` 0 kerentanan; 7 moderate dev-only `minimatch` di `@typescript-eslint` didokumentasikan & diterima sementara di `SECURITY-AUDIT.md`).
- [x] Pastikan `/api/metrics` tetap tidak dapat diakses publik (blokir Nginx `404` + CIDR allowlist `127.0.0.0/8`, `::1`, `172.16.0.0/12` + unit/E2E test).
- [x] Review dependency update berikutnya (`npm audit fix --no-fund` dijalankan + jadwal review berkala tercakup).

### [x] P2-06 - Observability dan alert (#154)

- [x] Tambahkan metrik: jumlah event per tipe, durasi verifikasi, success rate, dan jumlah kamera offline.
- [x] Alert jika kamera offline lebih dari 15 menit (`CameraOfflineOver15Minutes` di `monitoring/prometheus/alerts.yml`).
- [x] Tambahkan panel Grafana untuk tren online/offline (`sigap-observability.json`).
- [x] Pastikan log tidak memuat kredensial RTSP (`RedactRTSPCredentialsInText` + `safeStreamErrorText`).

### [x] P2-07 - CI/E2E regression suite (#155)

- [x] Tambahkan test E2E untuk site isolation, proxy spoofing, timezone, dan retensi (`e2e/tests/ops-regression.spec.ts` + `ops_regression_integration_test.go`).
- [x] Jalankan full backend test, vet, lint, frontend build, lint, serta test live/monitor di CI (`.github/workflows/ci.yml`).
- [x] Tambahkan smoke test untuk halaman `/cctv/logs`.

---

## Definition of Done

Catatan: checklist di bawah mencerminkan status penyelesaian **implementasi & hardening di repository** (kecuali gate manual/production `#147`–`#148` serta aksi fisik lapangan `#151`/`#153`):

- [x] Kode dan test unit/integration tersedia.
- [x] `go test ./...`, `go vet ./...`, dan `golangci-lint` lulus.
- [x] `npm run build`, `npm run lint`, dan test frontend terkait lulus.
- [x] Changelog backend/frontend diperbarui.
- [x] Dokumentasi dan Swagger diperbarui.
- [x] Migration memiliki file up/down dan sudah diuji.
- [x] Tidak ada kredensial/token di log.
- [ ] Bukti manual test tersedia untuk perubahan UI (`#147` Bug 4/8/21 belum dieksekusi).

---

## Urutan Eksekusi yang Disarankan

1. **Sprint 1 - Keamanan:** P0-01, P0-02, P0-03. ✅
2. **Sprint 2 - Koreksi fitur log:** P1-01 sampai P1-05. ✅
3. **Sprint 3 - Release gate:** P2-01 ✅, P2-02 ✅; P1-06 (`#147`) & P1-07 (`#148`) ⏳ menunggu eksekusi.
4. **Sprint 4 - Hardening:** P2-04 ✅, P2-06 ✅, P2-07 ✅; P2-03 (`#151`) & P2-05 (`#153`) ✅ sisi kode/docs, ⏳ aksi fisik lapangan/VPS.

---

## Issue GitHub

17 issue telah dibuat pada 2026-10-27. Rentang issue: **#139-#155** (P0: #139-#141, P1: #142-#148, P2: #149-#155). Berikut judul dan labelnya:

| ID | Judul issue | Label |
|---|---|---|
| P0-01 | `[Security] Isolasi site pada Log CCTV dan WebSocket` | `security`, `backend`, `frontend`, `P0` |
| P0-02 | `[Security] Hardening trusted proxy dan validasi IP klien` | `security`, `backend`, `infra`, `P0` |
| P0-03 | `[Test] Regression test multi-site dan proxy spoofing` | `test`, `security`, `P0` |
| P1-01 | `[Fix] Timezone dan konsistensi ringkasan Log CCTV` | `bug`, `backend`, `P1` |
| P1-02 | `[UX] Perbaikan filter dan pencarian Log CCTV` | `frontend`, `UX`, `P1` |
| P1-03 | `[Ops] Retensi dan index tabel cctv_stream_logs` | `backend`, `database`, `P1` |
| P1-04 | `[Fix] Validasi dan error handling endpoint Log CCTV` | `backend`, `P1` |
| P1-05 | `[Decision] Kebijakan event standby dan deduplikasi log` | `product`, `backend`, `P1` |
| P1-06 | `[Test] Selesaikan manual test Bug 4, 8, dan 21` | `test`, `release`, `P1` |
| P1-07 | `[Release] Deployment dan smoke test production` | `release`, `infra`, `P1` |
| P2-01 | `[Streaming] Kompatibilitas H.265 dan snapshot` | `streaming`, `frontend`, `P2` |
| P2-02 | `[Streaming] Diagnostik Metro-E dan false offline` | `streaming`, `backend`, `P2` |
| P2-03 | `[Data] Verifikasi 9 koordinat CCTV` | `data-quality`, `field`, `P2` |
| P2-04 | `[Docs] Sinkronisasi dokumentasi dan status rilis` | `docs`, `P2` |
| P2-05 | `[Security] Item keamanan residual` | `security`, `infra`, `P2` |
| P2-06 | `[Ops] Observability dan alert status kamera` | `observability`, `backend`, `P2` |
| P2-07 | `[Test] CI/E2E regression suite` | `test`, `CI`, `P2` |

---

## Keputusan yang Dibutuhkan

- [x] Halaman Log CCTV tersedia untuk admin maupun teknisi dengan scope site aktif (`GET /api/cctv/logs` & `GET /api/cctv-logs`, fail-closed `403` tanpa penugasan site).
- [x] Event `standby` ditetapkan sebagai status UI; pencatatan event ke log dinonaktifkan default (`CCTV_LOG_STANDBY_EVENTS=false`) dengan deduplikasi 60 detik.
- [x] Retensi default log ditetapkan `90` hari via `CCTV_STREAM_LOG_RETENTION_DAYS=90`.
- [ ] Apakah site pada log mengikuti site kamera saat ini atau snapshot kolom `site_id` historis pada `cctv_stream_logs` saat event terjadi? *(Saat ini memakai join ke `cctvs.site_id`; snapshot kolom historis tetap opsional jangka panjang.)*
- [ ] Apakah target rilis berikutnya adalah pilot multi-site atau production penuh setelah gate `#147`/`#148`?

---

## Catatan

Dokumen ini adalah backlog usulan berdasarkan inspeksi statis dan laporan yang tersedia di repository. Sebelum eksekusi, validasi ulang temuan P0-01 dan P0-02 pada branch terbaru karena area tersebut bersinggungan dengan perubahan keamanan dan multi-site.
