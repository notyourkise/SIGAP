# TODO Perbaikan Fitur SIGAP

**Tanggal:** 2026-10-27  
**Basis commit:** `8819c3e` (`main`, merge PR #137)  
**Status:** Usulan backlog; belum ada item yang dikerjakan  
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

### [ ] P0-01 - Isolasi site pada Log CCTV dan WebSocket

**Masalah:** `GetCCTVStreamLogs` menerima `SiteContext`, tetapi tidak memakai `c.Locals("site_id")`. User non-admin di site A dapat melihat log site B. Event `CCTV_STREAM_LOG_CREATED` juga dikirim global ke semua koneksi WebSocket.

**Pekerjaan:**
- [ ] Tambahkan scope site pada query count, list, dan ringkasan log.
- [ ] Gunakan pola `EXISTS` ke `cctvs` seperti `applyMaintenanceSiteScope`, atau join yang setara.
- [ ] Pastikan user non-admin tanpa site assignment tidak melihat data apa pun.
- [ ] Tambahkan `site_id` pada payload event WebSocket.
- [ ] Batasi broadcast event ke site terkait. Jika site room belum tersedia, opsi minimal adalah membatasi event log ke admin terlebih dahulu.
- [ ] Audit event global lain yang membawa nama/kode CCTV, terutama `CCTV_LIVE_STATUS_UPDATED`.
- [ ] Tentukan kebijakan akses: admin-only, atau teknisi dengan scope site.
- [ ] Tambahkan regression test dua site dan dua user.

**File utama:**
`backend/handlers/cctv_stream_log.go`, `backend/routes/cctv.go`, `backend/utils/websocket.go`, `backend/handlers/stream_probe.go`, `backend/handlers/stream_verify.go`, `frontend/src/app/cctv/logs/page.tsx`.

**Acceptance criteria:**
- [ ] User site A mendapat 403/data kosong saat mengakses log site B.
- [ ] Event WebSocket site B tidak diterima user site A.
- [ ] Admin tetap dapat melihat seluruh site atau site yang dipilih.
- [ ] Test regresi lulus.

**Catatan desain:** Untuk jangka panjang, pertimbangkan kolom `site_id` pada `cctv_stream_logs` sebagai snapshot site saat event terjadi. Jika kamera dipindahkan antar-site, log lama tetap berada pada site historisnya.

---

### [ ] P0-02 - Hardening trusted proxy dan IP klien

**Masalah:** `ProxyHeader` selalu diisi, tetapi `EnableTrustedProxyCheck` hanya aktif jika `TRUSTED_PROXY_IPS` diisi. Pada Fiber v2, kondisi ini membuat `X-Forwarded-For` dari client langsung tetap dipercaya. `EnableIPValidation` juga belum aktif.

**Pekerjaan:**
- [ ] Set `ProxyHeader` hanya jika `TRUSTED_PROXY_IPS` valid dan tidak kosong.
- [ ] Aktifkan validasi IP.
- [ ] Pastikan `getClientIP` dan audit log memakai IP hasil validasi, bukan seluruh isi header XFF.
- [ ] Ubah Nginx agar menimpa `X-Forwarded-For` dengan `$remote_addr`, bukan `$proxy_add_x_forwarded_for`, atau gunakan konfigurasi real-IP yang eksplisit.
- [ ] Tambahkan test: request langsung dengan XFF palsu tidak boleh mengubah rate-limit bucket atau audit IP.
- [ ] Tambahkan test: request dari trusted proxy memakai IP client yang benar.

**File utama:**
`backend/main.go`, `backend/middleware/limiter.go`, utilitas audit IP, `nginx/nginx.conf`, `backend/middleware/limiter_test.go`.

**Acceptance criteria:**
- [ ] Client langsung tidak bisa memalsukan IP.
- [ ] Rate limit tidak bisa dilewati dengan mengganti XFF.
- [ ] Audit log mencatat IP yang benar.
- [ ] Deployment tanpa proxy tidak memakai header XFF.

---

### [ ] P0-03 - Test regresi keamanan multi-site/proxy

**Pekerjaan:**
- [ ] Test API log CCTV untuk admin, teknisi site A, dan teknisi site B.
- [ ] Test WebSocket event tidak bocor lintas site.
- [ ] Test trusted vs untrusted proxy.
- [ ] Test rate limit per endpoint dan per IP.
- [ ] Jalankan `go test ./... -count=1`, `go vet ./...`, dan `golangci-lint run ./...`.

**Acceptance criteria:**
- [ ] Semua test baru lulus.
- [ ] Tidak ada regresi pada test lama.
- [ ] Hasil test dicatat di changelog/report.

---

## P1 - Sprint Berikutnya

### [ ] P1-01 - Timezone dan konsistensi ringkasan log

**Masalah:** `startOfDay` memakai timezone proses, sedangkan komentar menyebut UTC. Container umumnya UTC, operasional memakai WITA. Ringkasan "hari ini" juga global, tidak mengikuti site/filter.

**Pekerjaan:**
- [ ] Tambahkan konfigurasi `APP_TIMEZONE=Asia/Makassar` atau helper timezone eksplisit.
- [ ] Hitung rentang hari dengan batas bawah dan batas atas.
- [ ] Terapkan site/filter yang sama pada query ringkasan.
- [ ] Selaraskan label UI: "Total hasil filter", "Event hari ini pada site/filter aktif".
- [ ] Tambahkan unit test untuk batas tengah malam WITA.

**Acceptance criteria:**
- [ ] Event pukul 00:30 WITA masuk hari yang benar.
- [ ] Angka kartu konsisten dengan daftar dan filter.
- [ ] Tidak ada perbedaan hasil antara server UTC dan WITA.

---

### [ ] P1-02 - Perbaikan UX halaman Log CCTV

**Pekerjaan:**
- [ ] Tambahkan filter `unknown`/"Tidak Diketahui".
- [ ] Debounce pencarian 300 ms agar tidak fetch setiap ketikan.
- [ ] Reset page ke 1 saat filter/pencarian berubah.
- [ ] Simpan filter di URL query agar bisa dibagikan/refresh.
- [ ] Tambahkan tombol "Coba lagi" saat error.
- [ ] Tampilkan "Terakhir diperbarui" dan indikator loading yang tidak menutupi data lama.
- [ ] Pastikan empty state membedakan "belum ada log" dan "tidak ada hasil filter".

**Acceptance criteria:**
- [ ] Mengetik pencarian tidak membanjiri API.
- [ ] Filter unknown berfungsi.
- [ ] Refresh halaman mempertahankan filter.
- [ ] Error state bisa dipulihkan tanpa reload penuh.

---

### [ ] P1-03 - Retensi dan index `cctv_stream_logs`

**Pekerjaan:**
- [ ] Tambahkan worker cleanup harian, misalnya `CCTV_STREAM_LOG_RETENTION_DAYS=90`.
- [ ] Hapus batch 1.000 baris agar tidak mengunci tabel.
- [ ] Tambahkan index komposit yang sesuai, misalnya `(cctv_id, created_at)` dan `(event_type, created_at)`.
- [ ] Tambahkan test cleanup dan konfigurasi default.
- [ ] Dokumentasikan retensi dan cara audit log lama.

**Acceptance criteria:**
- [ ] Log lebih tua dari batas terhapus.
- [ ] Log baru tetap aman.
- [ ] Worker idempotent dan tidak membebani database.
- [ ] Tidak ada pertumbuhan tabel tanpa batas.

---

### [ ] P1-04 - Validasi dan error handling endpoint log

**Pekerjaan:**
- [ ] Validasi `event_type` hanya `online`, `offline`, `standby`, dan `unknown`; nilai lain menghasilkan HTTP 400.
- [ ] Periksa error pada query count ringkasan; jangan diam-diam menampilkan 0.
- [ ] Gunakan konstanta event type agar tidak ada typo.
- [ ] Pertimbangkan batas maksimum rentang tanggal jika nanti ditambahkan.
- [ ] Tambahkan test untuk query error dan event type invalid.

**Acceptance criteria:**
- [ ] Input invalid menghasilkan pesan jelas.
- [ ] Kegagalan database tidak menampilkan angka palsu.
- [ ] Test handler mencakup skenario error.

---

### [ ] P1-05 - Kebijakan event standby dan duplikasi

**Masalah:** Probe 2 menit dan verifier 15 menit dapat menghasilkan log `standby`/`unknown` yang terlalu sering. Perlu keputusan produk.

**Pekerjaan:**
- [ ] Tentukan apakah `standby` layak dicatat sebagai event atau hanya status.
- [ ] Jika tetap dicatat, tambahkan opsi `CCTV_LOG_STANDBY_EVENTS=false` atau dedup window 60 detik.
- [ ] Cegah duplikasi event `online`/`offline` dari probe dan verifier dalam waktu singkat.
- [ ] Dokumentasikan semantics `probe` vs `verify`.

**Acceptance criteria:**
- [ ] Tidak ada banjir log saat kamera idle.
- [ ] Event penting online/offline tetap tercatat.
- [ ] Keputusan produk terdokumentasi.

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
- [ ] Terapkan migrasi 27 dan 28.
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

### [ ] P2-01 - Kompatibilitas H.265 dan snapshot

- [ ] Deteksi dukungan HEVC di browser; tampilkan pesan yang jelas jika tidak didukung.
- [ ] Sediakan fallback ke main stream/H.264 atau panduan perangkat.
- [ ] Nonaktifkan tombol snapshot jika endpoint MediaMTX 1.20 tidak tersedia; jangan tampilkan UI rusak.
- [ ] Dokumentasikan batasan codec dan snapshot.

### [ ] P2-02 - Diagnostik Metro-E dan false offline

- [ ] Tambahkan pengecekan startup: jika `CCTV_METRO_E_DIRECT_PROBE=true` tetapi `10.20.105.x` tidak reachable, log warning dan fallback ke status pasif.
- [ ] Tambahkan indikator health/diagnostic di endpoint admin.
- [ ] Pastikan hasil tidak pernah menampilkan offline palsu.

### [ ] P2-03 - Verifikasi 9 koordinat CCTV

- [ ] Verifikasi lapangan: `THD-18`, `THD-20`, `THD-25`, `THD-27`, `THD-29`, `THD-30`, `THD-31`, `THD-34`, `THD-35`.
- [ ] Update latitude/longitude.
- [ ] Hapus label "Estimasi" setelah terverifikasi.
- [ ] Tambahkan laporan data quality koordinat.

### [ ] P2-04 - Sinkronisasi dokumentasi dan status

- [ ] Ubah Next.js 15 menjadi 16.4.2.
- [ ] Ubah "22+ migration" menjadi 28.
- [ ] Perbarui tabel endpoint, termasuk `/api/cctv-logs` dan `/api/cctv/logs`.
- [ ] Selaraskan klaim `ROADMAP.md` "Production Ready" dengan status manual test.
- [ ] Perbarui path proyek lama di `docs/README.md`.
- [ ] Dokumentasikan retensi log, timezone, dan site scoping.

### [ ] P2-05 - Item keamanan residual

- [ ] Rotasi kredensial NVR dan pastikan tidak ada nilai nyata di source/UI.
- [ ] Verifikasi backup terenkripsi dan salinan off-site.
- [ ] Putuskan risiko 5 temuan `npm audit` dev-only.
- [ ] Pastikan `/api/metrics` tetap tidak dapat diakses publik.
- [ ] Review dependency update berikutnya.

### [ ] P2-06 - Observability dan alert

- [ ] Tambahkan metrik: jumlah event per tipe, durasi verifikasi, success rate, dan jumlah kamera offline.
- [ ] Alert jika kamera offline lebih dari 15 menit.
- [ ] Tambahkan panel Grafana untuk tren online/offline.
- [ ] Pastikan log tidak memuat kredensial RTSP.

### [ ] P2-07 - CI/E2E regression suite

- [ ] Tambahkan test E2E untuk site isolation, proxy spoofing, timezone, dan retensi.
- [ ] Jalankan full backend test, vet, lint, frontend build, lint, serta test live/monitor di CI.
- [ ] Tambahkan smoke test untuk halaman `/cctv/logs`.

---

## Definition of Done

Setiap item dianggap selesai jika:

- [ ] Kode dan test unit/integration tersedia.
- [ ] `go test ./...`, `go vet ./...`, dan `golangci-lint` lulus.
- [ ] `npm run build`, `npm run lint`, dan test frontend terkait lulus.
- [ ] Changelog backend/frontend diperbarui.
- [ ] Dokumentasi dan Swagger diperbarui.
- [ ] Migration memiliki file up/down dan sudah diuji.
- [ ] Tidak ada kredensial/token di log.
- [ ] Bukti manual test tersedia untuk perubahan UI.

---

## Urutan Eksekusi yang Disarankan

1. **Sprint 1 - Keamanan:** P0-01, P0-02, P0-03.
2. **Sprint 2 - Koreksi fitur log:** P1-01 sampai P1-05.
3. **Sprint 3 - Release gate:** P1-06, P1-07, P2-01, P2-02.
4. **Sprint 4 - Hardening:** P2-03 sampai P2-07.

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

- [ ] Apakah halaman Log CCTV hanya untuk admin, atau juga teknisi dengan scope site?
- [ ] Apakah event `standby` perlu dicatat sebagai riwayat, atau cukup menjadi status?
- [ ] Apakah retensi default log 90 hari, 180 hari, atau mengikuti kebijakan audit?
- [ ] Apakah site pada log mengikuti site kamera saat ini atau site historis saat event terjadi?
- [ ] Apakah target rilis berikutnya adalah pilot multi-site atau production penuh?

---

## Catatan

Dokumen ini adalah backlog usulan berdasarkan inspeksi statis dan laporan yang tersedia di repository. Sebelum eksekusi, validasi ulang temuan P0-01 dan P0-02 pada branch terbaru karena area tersebut bersinggungan dengan perubahan keamanan dan multi-site.
