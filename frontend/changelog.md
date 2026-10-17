# Changelog - Frontend SIGAP

## [1.55.4] - 2026-10-18

### Perbaikan loop redirect halaman ganti sandi (alur wajib ganti sandi)

**Bug**: Halaman `/change-password` mengalami reload tanpa henti saat pengguna wajib mengganti sandi (akun baru / hasil reset). `AuthContext` memanggil `window.location.replace('/change-password')` tanpa memeriksa path aktif, sehingga provider ter-mount ulang terus-menerus. Akibatnya `/auth/me` dipanggil ratusan kali sampai kena rate limit (429); state pengguna gagal dimuat dan elemen mode wajib (banner peringatan + tombol "Exit / Back to Login") tidak muncul.

**Perbaikan**: Pengalihan dipusatkan ke satu effect idempoten yang hanya menavigasi bila path saat ini berbeda dari tujuan, memakai navigasi halus `router.replace` (provider tidak ter-mount ulang). Guard keliru di halaman `/change-password` yang membuang pengguna non-wajib ke `/dashboard` dihapus sehingga ganti sandi mandiri dari halaman Profil berfungsi kembali; `handleSuccess` memakai `updateUser` agar tujuan navigasi tepat (wajib -> `/dashboard`, mandiri -> `/profile`).

timestamp: 2026-10-18 02:12 UTC+8
kategori: Tinggi, Frontend (Bug Fix - Autentikasi)
file yang diubah:
- frontend/src/context/AuthContext.tsx (redirect wajib ganti sandi idempoten)
- frontend/src/app/change-password/page.tsx (hapus guard salah + handleSuccess)
- e2e/tests/change-password.spec.ts (regression test Playwright)
- e2e/playwright.config.ts, e2e/package.json (setup Playwright)
- frontend/changelog.md
alasan perubahan: Mencegah loop reload tanpa henti + rate limit pada alur wajib ganti sandi, sekaligus memulihkan fungsi ganti sandi mandiri.

## [1.55.3] - 2026-08-20

### Multi-Site Isolation UI & Security Hardening

**Multi-Site Isolation**: Dropdown "Kawasan" dan "Zona" ditambahkan di halaman `/cctv` form (data dari `/sites/all` dan `/zones/all`); payload create/update CCTV mengirim `zone_id` dan `site_id`; halaman `/checklist-templates` filter per kawasan + field kawasan di form (kosong = global template); wizard FieldMaintenanceModal load template sesuai kawasan CCTV yang dipilih; cookie `sigap_site_id` untuk persistensi site aktif across pages.

**Security Bug #1 Credential Cleanup**: Unified kepanjangan SIGAP di manifest PWA & mobile ke "Sistem Informasi Gangguan & Aset Pusat"; semua placeholder NVR credential diganti `<NVR_USERNAME>:<NVR_PASSWORD>`; dokumentasi frontend konsisten dengan security policy.

**Manual Test V2**: Dokumentasi formal testing Multi-Site Isolation & Backup Encryption diorganisir ke `docs/Testing-Manual-V2/` mengikuti blackbox format standar AGENTS.md §9.

timestamp: 2026-08-20 19:45 UTC+8
kategori: Sedang, Frontend (UI Enhancement + Documentation)
file yang diubah:
- frontend/src/app/cctv/page.tsx (dropdown Kawasan/Zona)
- frontend/src/components/ChecklistTemplateManager.tsx (filter kawasan)
- frontend/src/components/FieldMaintenanceModal.tsx (template by site_id)
- frontend/src/lib/api.ts (site_id query param support)
- frontend/public/manifest.json (kepanjangan unified)
- mobile/src/screens/LoginScreen.tsx (kepanjangan unified)
- docs/changelog.md
- frontend/changelog.md
alasan perubahan: Completing frontend UI for multi-site isolation feature (backend already deployed); consistency across web/mobile branding; formal documentation governance.

## [1.55.2] - 2026-08-20

### Status playback WHEP berbasis decoded frame

- Player membedakan signaling, koneksi media, menunggu frame, live, dan reconnecting.
- Event track tidak lagi langsung menampilkan badge `Live`; status live menunggu frame video yang benar-benar didekode.
- Timeout frame memberi pesan pemulihan yang spesifik dan memicu reconnect.
- Toast start menjelaskan bahwa jalur baru terdaftar dan video masih ditunggu.
- Node regression test menjaga urutan state dan race antara track dengan penyelesaian SDP.

timestamp: 2026-08-20 WITA
kategori: Kritis, Frontend (Streaming UX + Reliability)

## [1.55.1] - 2026-08-20

### Perbaikan cache label setelah rotasi QR CCTV

perubahan yang dilakukan: **Cache QR kini dibedakan berdasarkan ID CCTV dan payload aktif agar label hasil rotasi langsung memakai gambar baru.**

before:
- Cache data URL hanya memakai ID CCTV; rotasi mempertahankan ID sehingga gambar label lama dapat dipakai ulang.

after:
- Cache key memakai kombinasi `cctv.id` dan `cctv.qr_payload`; fetch gambar memakai mode `no-store`.
- Respons rotasi dengan payload baru otomatis memicu fetch gambar QR baru sebelum dialog cetak.
- Service worker memperlakukan endpoint QR sebagai network-only, tidak menyimpan respons `no-store`, dan memakai cache shell v3 agar cache lama dibuang saat aktivasi.
- Dua Node regression test memverifikasi bypass QR dan larangan menyimpan respons `no-store`.

timestamp: 2026-08-20 WITA
kategori: Kritis, Frontend (Perbaikan)
file yang diubah:
- frontend/src/app/cctv/page.tsx
- frontend/public/sw.js
- frontend/tests/service-worker-cache.test.mjs
- frontend/package.json

## [1.55.0] - 2026-08-13

### Perbaikan UI & fitur kecil (Sprint 40 cleanup)

perubahan yang dilakukan: **Membersihkan UI (hapus lonceng login, badge payload, tombol muat ulang, info streaming aktif, filter zona, badge Halaman Admin, panel toggle koordinat di peta), memperbaiki dropdown aksi yang tertutup, sort default A-Z, menambah input kawasan/zona di form CCTV, checklist per kawasan, dan memperbaiki kepanjangan SIGAP.**

before:
- Halaman login menampilkan lonceng notifikasi yang tidak berfungsi.
- Scan QR punya toggle "Masukkan payload" (input manual) yang tidak diperlukan.
- Dropdown aksi per baris di tabel CCTV tertutup elemen lain saat baris terbawah.
- Daftar CCTV default urut created_at (terbaru dulu), bukan A-Z.
- Halaman sebaran punya tombol "Muat ulang", info "X kamera streaming aktif", dropdown "Semua zona", dan panel toggle CCTV tanpa koordinat di dalam peta.
- Form CCTV tidak bisa memilih kawasan/zona (padahal master data ada).
- Template checklist sama untuk semua kawasan.
- Halaman checklist-templates menampilkan badge "Halaman Admin".
- Kepanjangan SIGAP di manifest PWA & mobile masih belum konsisten dengan nama resmi proyek.

after:
- **`login/page.tsx`**: hapus ikon lonceng notifikasi (hanya toggle tema).
- **`cctv/page.tsx`**: hapus toggle "Masukkan payload" + input manual (kamera saja); dropdown aksi per baris z-50 + wrapper tabel overflow-visible saat menu terbuka; default sort `nama` asc (A-Z).
- **`sebaran-cctv/page.tsx`**: hapus tombol "Muat ulang", info "X kamera streaming aktif", dan dropdown filter zona (pemilihan kawasan tetap via SiteSwitcher header).
- **`CCTVMap.tsx`**: hapus panel toggle "N CCTV belum memiliki koordinat" di dalam peta; daftar + tombol Edit dipindah ke bawah status bar.
- **`cctv/page.tsx`**: form CCTV + dropdown "Kawasan" & "Zona" (data dari `/zones/all` & `/sites/all`); payload create/update mengirim `zone_id`/`site_id`.
- **`checklist-templates/page.tsx`**: hapus badge "Halaman Admin".
- **`ChecklistTemplateManager.tsx`**: dropdown filter kawasan + field kawasan di form (kosong = global); fetch template per `site_id`.
- **`FieldMaintenanceModal.tsx`**: wizard memuat template sesuai kawasan CCTV yang dipilih.
- **`lib/api.ts`**: `ChecklistTemplate.site_id` + query `site_id` pada `getChecklistTemplates`.
- **`manifest.json` & `mobile/LoginScreen.tsx`**: kepanjangan SIGAP → "Sistem Informasi Gangguan & Aset Pusat".

timestamp: 2026-08-13 19:45 WITA
kategori: Sedang, Frontend (Perbaikan + Fitur kecil)
file yang diubah:
- frontend/src/app/login/page.tsx
- frontend/src/app/cctv/page.tsx
- frontend/src/app/sebaran-cctv/page.tsx
- frontend/src/components/CCTVMap.tsx
- frontend/src/components/ChecklistTemplateManager.tsx
- frontend/src/components/FieldMaintenanceModal.tsx
- frontend/src/app/checklist-templates/page.tsx
- frontend/src/lib/api.ts
- frontend/public/manifest.json
- mobile/src/screens/LoginScreen.tsx
alasan perubahan: Perbaikan UX & kelengkapan data operasional (kawasan/zona & checklist per kawasan) sesuai masukan user.

## [1.54.0] - 2026-08-13

### Sprint 40 — AI Feature Takedown (hapus fitur AI detection)

perubahan yang dilakukan: **Menghapus seluruh fitur AI detection dari frontend: halaman admin `/ai-analytics`, canvas bounding box overlay + toggle AI di `WHEPPlayer.tsx`/`StreamGridView.tsx`, badge counter orang per sel, dan menu sidebar "AI Analytics". Sistem kembali ke pure CCTV management tanpa AI inference.**

before:
- Frontend memiliki halaman `/ai-analytics` (admin) dengan ringkasan hitungan orang + tren 7 hari, overlay bounding box hijau di atas video streaming, tombol toggle "🤖 AI" + badge "👥 N orang hari ini" di `/monitor`, dan item menu sidebar "AI Analytics".

after:
- Halaman `/ai-analytics` dihapus (404); canvas overlay & toggle AI dihapus dari `WHEPPlayer.tsx`/`StreamGridView.tsx`; badge counter dihapus; item menu sidebar "AI Analytics" dihapus dari `DashboardLayout.tsx`; guard proxy `/ai-analytics` dihapus dari `proxy.ts`.

timestamp: 2026-08-13 20:30 WITA
kategori: Besar, Frontend (Takedown)
file yang diubah:
- frontend/src/app/ai-analytics/page.tsx (dihapus)
- frontend/src/components/WHEPPlayer.tsx
- frontend/src/components/StreamGridView.tsx
- frontend/src/app/monitor/page.tsx
- frontend/src/components/DashboardLayout.tsx
- frontend/src/proxy.ts
- frontend/changelog.md
alasan perubahan: Fitur AI detection (person counting, bounding box, YOLOv8) dihapus pada Sprint 40 (AI Feature Takedown) — SIGAP kembali menjadi pure CCTV management tanpa AI inference.

## [1.53.0] - 2026-08-13

### Fitur: Badge Probe Strategy Metro-E vs GSM (Sprint 37-PATCH)

perubahan yang dilakukan: **Membedakan badge status live kamera berdasarkan jenis jaringan: Metro-E (serat optik, di-probe ICMP tiap 2 menit — status binary online/offline) vs GSM (tidak pernah di-probe — status berdasar state lokal MediaMTX + riwayat aktif dengan state "Pernah aktif hari ini").**

before:
- Semua kamera memakai badge 3-state sederhana berbasis `stream_live` (🟢 Online / 🔴 Offline / ⚪ Belum dicek) tanpa membedakan cara deteksi per jaringan. Kamera GSM yang baru mati tampak "Offline" merah meski masih ada aktivitas hari ini, dan tidak ada informasi latensi ping untuk Metro-E.

after:
- **`src/components/CCTVMap.tsx`**: `CCTVMapItem` tambah `network_type` ('metro-e'|'gsm'|null), `ping_latency_ms` (null|number), `stream_last_active_at` (null|string ISO). Helper baru `getLiveBadge(cctv)` (diexport) — Metro-E binary (🟢 Online (Live) / 🔴 Offline / ⚪ Tidak diketahui) vs GSM 4-state (🟢 Online (Live) / 🟡 "Pernah aktif hari ini" bila stream_last_active_at < 24 jam / 🔴 Offline bila ada riwayat ≥ 24 jam / ⚪ Tidak diketahui) dengan fallback perilaku lama untuk data tanpa network_type. Helper baru `formatLatency(ms)` → `Ping: Xms` + 🟢 (<50ms) / 🟡 (<100ms) / 🔴 (≥100ms). `createCustomMarkerIcon` kini menerima objek `cctv` (bukan `streamLive` mentah) — warna badge marker mengikuti `getLiveBadge(cctv).color`. Popup marker menampilkan `{dot} {label}`, untuk Metro-E `· Ping: Xms` bila ada latensi, untuk GSM `· Aktif HH:mm` dari stream_last_active_at, dan selalu `· Dicek HH:mm`.
- **`src/components/StreamGridView.tsx`**: indikator emoji + tooltip sel memakai `getLiveBadge`/`formatLatency` yang diimpor dari CCTVMap (tanpa duplikasi logika); tooltip Metro-E menyertakan `Ping: Xms`, GSM memakai label badge (mis. "Pernah aktif hari ini"), selalu disertai waktu dicek. Ring/border sel tetap berbasis `stream_live` (hijau/merah/abu).
- **`src/app/sebaran-cctv/page.tsx`**: merge `/streams/live-status` kini membawa `network_type`, `ping_latency_ms`, `stream_last_active_at` (fallback null bila absen); handler WebSocket `CCTV_LIVE_STATUS_UPDATED` memperbarui `ping_latency_ms`/`network_type` hanya bila payload menyediakannya (mempertahankan nilai lama), dan `stream_last_active_at` hanya diperbarui saat kamera live (stream_checked_at sebagai waktu aktif) — saat offline riwayat dipertahankan.

timestamp: 2026-08-13 19:00 WITA
kategori: Sedang, Frontend
file yang diubah:
- frontend/src/components/CCTVMap.tsx
- frontend/src/components/StreamGridView.tsx
- frontend/src/app/sebaran-cctv/page.tsx
- frontend/changelog.md
alasan perubahan: Kamera GSM tidak pernah di-probe ICMP sehingga badge binary lama menyesatkan (menampilkan Offline padahal masih ada aktivitas hari ini); Metro-E perlu menampilkan latensi ping hasil probe sebagai indikator kualitas koneksi. Membedakan strategi badge per jenis jaringan membuat status live akurat sesuai mekanisme deteksi backend.

## [1.52.0] - 2026-08-13

### Fitur: AI Overlay & E2E Integration (Sprint 39 - Task 14C.1, 14C.2 & 14C.3)

perubahan yang dilakukan: **Menambahkan overlay bounding box AI real-time di atas video streaming, tombol toggle AI session di grid monitor, counter orang unik per kamera, dan halaman admin AI Analytics.**

before:
- /monitor dan grid peta hanya menampilkan video tanpa indikator deteksi orang; tidak ada cara mengaktifkan/menghentikan session AI dari UI.
- Tidak ada halaman ringkasan hitungan orang per kamera.

after:
- **`components/WHEPPlayer.tsx`** (14C.1): props baru `cctvId` & `aiEnabled`; `<canvas>` overlay (z-[5], pointer-events-none) yang menggambar bounding box hijau (#22C55E) + label "Person {track_id}" dengan koordinat relatif 0–1 di-scale ke ukuran video; auto-fade 1.5 detik; subscribe WebSocket `AI_DETECTION_UPDATE` via `useWebSocket()` (pola resmi codebase).
- **`app/monitor/page.tsx`** (14C.2): state `aiEnabled` (persist localStorage `sigap_ai_enabled`) + `aiCounts`; tombol toggle "🤖 AI" / "AI Aktif" hijau; sync session AI saat ganti layout/selection (POST/DELETE `/ai/sessions`); badge counter "👥 N orang hari ini" per sel.
- **`components/StreamGridView.tsx`** (14C.2): props opsional `aiEnabled`/`onAiToggle`/`aiCounts` + fallback state lokal (self-sufficient); tombol AI di header grid; badge counter per sel; `cctvId`/`aiEnabled` diteruskan ke WHEPPlayer.
- **`app/ai-analytics/page.tsx`** (14C.3, baru): halaman admin — kartu ringkasan (Total Orang, Kamera Dipantau, Aktif), tabel "Per Kamera — Hari Ini" (`GET /api/ai/counts`), dropdown kamera + grafik tren SVG mandiri 7 hari (`GET /api/ai/counts/:id/trend?days=7`); loading/error/empty state.
- **`components/DashboardLayout.tsx`**: menu sidebar "AI Analytics" (ikon `BrainCircuit`) di grup "Pelaporan & Analisis" (admin).
- **`proxy.ts`**: `/ai-analytics` ditambahkan ke `ADMIN_PATHS` & `config.matcher` (proteksi admin server-side).

timestamp: 2026-08-13 18:00 WITA
kategori: Besar, Frontend
file yang diubah:
- frontend/src/components/WHEPPlayer.tsx
- frontend/src/app/monitor/page.tsx
- frontend/src/components/StreamGridView.tsx
- frontend/src/app/ai-analytics/page.tsx (baru)
- frontend/src/components/DashboardLayout.tsx
- frontend/src/proxy.ts
- frontend/changelog.md

## [1.51.0] - 2026-08-13

### Fitur: RTSP Live Status Detection (Sprint 37 - Task 14A.2)

perubahan yang dilakukan: **Menambahkan badge online/offline real-time pada marker peta, status bar live counter, popup status live, border status per sel grid, dan WebSocket listener untuk update live status tanpa reload.**

before:
- Peta CCTV memakai field `status` (aktif/nonaktif/pemeliharaan) untuk warna marker — bukan status koneksi RTSP real-time. Kamera mati tetap tampak hijau meski offline.
- Tidak ada indikator online/offline real-time di peta maupun grid monitor.

after:
- **`src/components/CCTVMap.tsx`**: `CCTVMapItem` tambah `stream_live?` & `stream_checked_at?`; `createCustomMarkerIcon` tambah param `streamLive` → badge bulat di pojok marker (🟢 online / 🔴 offline / ⚪ belum dicek); status bar "X Online · Y Offline · Z Belum dicek"; popup menampilkan status live + waktu terakhir dicek.
- **`src/components/StreamGridView.tsx`**: border/ring warna per sel berdasarkan `stream_live` (hijau/merah/abu); indikator emoji kecil di pojok sel dengan tooltip waktu cek.
- **`src/app/sebaran-cctv/page.tsx`**: enrich data CCTV dengan `GET /streams/live-status` (admin-only, fallback null untuk non-admin); subscribe WebSocket `CCTV_LIVE_STATUS_UPDATED` → update badge tanpa reload.

timestamp: 2026-08-13 14:00 WITA
kategori: Sedang, Frontend
file yang diubah:
- frontend/src/components/CCTVMap.tsx
- frontend/src/components/StreamGridView.tsx
- frontend/src/app/sebaran-cctv/page.tsx

## [1.50.0] - 2026-08-13

### Fitur: Multi-Site (Sprint 30 - Task 12.2b)

perubahan yang dilakukan: **Menambahkan site switcher di header, header `X-Site-ID` pada seluruh request API, halaman admin kawasan (CRUD + assign teknisi), dan pemusatan peta sebaran ke koordinat site aktif.**

before:
- SIGAP hanya melayani satu kawasan (OIKN); tidak ada cara memilih kawasan maupun melihat data per kawasan.

after:
- **`src/lib/api.ts`**: helper `setCookie`/`clearCookie`, interface `Site`, `apiRequest` kini menyisipkan header `X-Site-ID` dari cookie `sigap_site_id`; fungsi `getSites`, `getAllSitesForSelect`, `createSite`, `updateSite`, `deleteSite`, `assignUsersToSite`.
- **`src/components/SiteSwitcher.tsx`** (baru): dropdown pemilih kawasan di header; pilih → simpan cookie `sigap_site_id` → reload agar seluruh data difilter.
- **`src/app/sites/page.tsx`** (baru): halaman admin CRUD kawasan (nama, kode, alamat, koordinat, default), modal assign teknisi multi-select, hapus dengan konfirmasi (site default ditolak backend).
- **`src/components/DashboardLayout.tsx`**: `SiteSwitcher` di header (mobile & desktop); menu sidebar "Kawasan" (`Building2`) → `/sites`.
- **`src/proxy.ts`**: `/sites` ditambahkan ke `ADMIN_PATHS` & `config.matcher`.
- **`src/components/CCTVMap.tsx`**: tambah prop `center` untuk memusatkan peta.
- **`src/app/sebaran-cctv/page.tsx`**: memusatkan peta ke koordinat site aktif (dari cookie `sigap_site_id`); data `/cctv` otomatis difilter via header `X-Site-ID`.

timestamp: 2026-08-13 13:30 WITA
kategori: Besar, Frontend
file yang diubah:
- frontend/src/lib/api.ts
- frontend/src/components/SiteSwitcher.tsx
- frontend/src/app/sites/page.tsx
- frontend/src/components/DashboardLayout.tsx
- frontend/src/proxy.ts
- frontend/src/components/CCTVMap.tsx
- frontend/src/app/sebaran-cctv/page.tsx

## [1.49.0] - 2026-08-13

### Fitur: Polish Frontend & UX (Sprint 36 - P15, P17, P18b)

perubahan yang dilakukan: **Menambahkan pagination, pause/resume, dan lazy-load pada grid streaming; pembuatan tugas pemeliharaan langsung dari kalender; serta filter periode (7 hari / 30 hari / 3 bulan / 1 tahun) pada dashboard kinerja teknisi.**

before:
- `/monitor` memuat seluruh `WHEPPlayer` sekaligus; tidak ada pagination, pause, maupun counter jumlah kamera.
- Kalender hanya menampilkan tugas existing — tidak bisa membuat tugas dari klik tanggal.
- `/kinerja-teknisi` hanya punya preset 7/30 hari + kustom; belum ada preset 3 bulan / 1 tahun.

after:
- **P15 — `StreamGridView`**:
  - Pagination grid: state `page`/`pageSize` (default 12), dropdown "4 / 8 / 12 / Semua", kontrol prev/next, guard halaman.
  - Counter header "Menampilkan {start}–{end} dari {total} kamera aktif".
  - Tombol Pause/Resume per sel: saat pause WHEPPlayer di-unmount (koneksi terputus) & placeholder "Dijeda"; saat resume di-mount ulang.
  - Lazy-load IntersectionObserver yang sudah ada dipertahankan (`isNear && !paused` → WHEPPlayer).
- **P17 — Kalender**:
  - `CalendarGrid` tambah prop `onDayClick` + tombol "+" per sel (muncul saat hover, stopPropagation).
  - Halaman `/kalender` tambah modal "Buat Tugas Pemeliharaan" (pilih CCTV, action_taken, scheduled_date pre-filled, notes) via `useToast`; submit `POST /maintenance-logs` lalu `loadLogs()` refresh; tutup via Escape/overlay/X.
- **P18b — `/kinerja-teknisi`**:
  - `RangePreset` menjadi `'7d'|'30d'|'90d'|'365d'|'custom'` (default `'30d'`); tombol filter "7 Hari / 30 Hari / 3 Bulan / 1 Tahun / Kustom".
  - Preset memakai `?period=...`; custom tetap `start_date`/`end_date`.
  - Subtitle "Data kinerja dalam {N} terakhir" sesuai periode aktif.

timestamp: 2026-08-13 09:00 WITA
kategori: Sedang, Frontend
file yang diubah:
- frontend/src/components/StreamGridView.tsx
- frontend/src/components/CalendarGrid.tsx
- frontend/src/app/kalender/page.tsx
- frontend/src/app/kinerja-teknisi/page.tsx

## [1.48.0] - 2026-08-12

### Fitur: Manajemen Template Checklist & Thumbnail Daftar Pemeliharaan (Sprint 35 - P12b, P13b)

perubahan yang dilakukan: **Menambahkan UI kelola template checklist pengecekan lapangan (admin) serta thumbnail foto CCTV di tabel pemeliharaan.**

before:
- Daftar item pengecekan lapangan terhardcode di konstanta `FIELD_CHECKLIST` (tidak bisa dikelola admin).
- Tabel pemeliharaan tidak menampilkan thumbnail foto CCTV (foto hanya tampil di modal detail).

after:
- **P12b — UI Manajemen Template Checklist**:
  - Fungsi & tipe API di `src/lib/api.ts`: `ChecklistTemplate`, `getChecklistTemplates`, `createChecklistTemplate`, `updateChecklistTemplate`, `deleteChecklistTemplate`, `reorderChecklistTemplates`.
  - Komponen baru `ChecklistTemplateManager` (mode `page` & `modal`): tabel No Urut/Nama Item/Deskripsi/Wajib/Aktif/Aksi, tombol tambah/edit inline, reorder naik-turun (tanpa dependency DnD eksternal), toggle wajib/aktif, hapus soft-delete dengan konfirmasi, serta loading/error/empty state.
  - Tombol "⚙️ Kelola Template" (khusus admin) di halaman `field-maintenance` yang membuka manager dalam modal.
  - Halaman baru `src/app/checklist-templates/page.tsx` (admin) yang merender manager full-page.
  - `FieldMaintenanceModal` kini membaca template aktif dari server (`getChecklistTemplates`) sebagai `activeList`, dengan `FIELD_CHECKLIST` sebagai fallback; `answers`, `totalSteps`, dan nama item saat submit mengikuti `activeList`.
  - Item menu sidebar "Template Checklist" (ikon `ListChecks`) + guard proxy (`ADMIN_PATHS` & `config.matcher`).
- **P13b — Thumbnail di Daftar Pemeliharaan**:
  - Helper `toThumbUrl` yang mengubah `/api/uploads/` atau `/uploads/` menjadi `/api/uploads/thumbnails/`.
  - Thumbnail 40x40 di sel CCTV (kolom pertama tabel maintenance) memakai `AuthenticatedImage`; klik thumbnail membuka detail log. Fallback ikon `Camera` abu-abu bila tidak ada `image_url`.

timestamp: 2026-08-12 10:35 WITA
kategori: Sedang, Frontend
file yang diubah:
- frontend/src/lib/api.ts
- frontend/src/components/ChecklistTemplateManager.tsx
- frontend/src/components/FieldMaintenanceModal.tsx
- frontend/src/components/DashboardLayout.tsx
- frontend/src/app/checklist-templates/page.tsx
- frontend/src/app/field-maintenance/page.tsx
- frontend/src/app/maintenance/page.tsx
- frontend/src/proxy.ts
- frontend/changelog.md

## [1.47.0] - 2026-08-10

### Fitur: UI Infrastruktur & Integrasi (Sprint 34 - P14b, P16b, P10)

perubahan yang dilakukan: **Menambahkan tombol rotasi QR Code di master CCTV, tombol regenerasi API key, serta tampilan jumlah log & tombol bersihkan log di halaman webhook.**

before:
- Master CCTV tidak bisa merotasi QR Code (label lama tetap berlaku).
- API key tidak bisa diregenerasi dari UI (harus hapus & buat baru).
- Halaman webhook tidak menampilkan jumlah log tersimpan & tidak ada cara membersihkan log.

after:
- **P14b — Rotasi QR (Task 13.14b)**:
  - Item menu "🔄 Rotasi QR" di dropdown aksi per baris CCTV (samping Edit/Hapus).
  - Dialog konfirmasi "QR Code lama akan tidak berlaku"; setelah sukses langsung siapkan cetak label QR baru.
- **P16b — Regenerasi API key (Task 13.16b)**:
  - Tombol "🔄 Regenerasi" per baris di halaman `/api-keys` (samping "Cabut").
  - Dialog konfirmasi peringatan integrasi; key mentah baru tampil di modal one-time dengan tombol salin.
- **P10 — Webhook log (Task 13.10)**:
  - Kolom "Log" di tabel webhook menampilkan jumlah log delivery per endpoint (`delivery_counts`).
  - Tombol "Bersihkan Log" di modal detail (hapus log > 30 hari, panggil `DELETE /api/webhooks/:id/deliveries`).
  - Field "Retensi Log (hari)" di form tambah/ubah webhook (`max_log_retention_days`).

timestamp: 2026-08-10 18:35 WITA
kategori: Sedang, Frontend
file yang diubah:
- frontend/src/app/cctv/page.tsx
- frontend/src/app/api-keys/page.tsx
- frontend/src/app/webhooks/page.tsx

## [1.46.0] - 2026-08-10

### Fitur: Grafik Tren Kesehatan & UI Eskalasi Overdue (Sprint 33 - P8b, P11b)

perubahan yang dilakukan: **Menambahkan grafik tren skor kesehatan di detail CCTV dan badge eskalasi overdue di tabel pemeliharaan.**

before: Detail CCTV tidak menampilkan tren skor; tabel pemeliharaan tidak menunjukkan level overdue.

after:
- **P8b — Grafik tren (Task 13.8b)**:
  - Section "Tren Kesehatan" di modal detail CCTV dengan tab range 7/30/90 hari.
  - Komponen `HealthTrendChart` (SVG mandiri): zona warna merah/kuning/hijau, highlight titik minimum, tooltip, loading skeleton, empty state.
  - Fetch `GET /cctv/:id/health-history`.
- **P11b — UI eskalasi overdue (Task 13.11b)**:
  - Badge status overdue di tabel pemeliharaan: 🟡 Overdue (level 1), 🟠 Overdue Tinggi (level 2), 🔴 KRITIS berkedip (level 3).
  - Notifikasi `OVERDUE_CRITICAL` ditampilkan dengan latar merah + ikon 🚨 di bell & halaman pemberitahuan.

timestamp: 2026-08-10 16:50 WITA
kategori: Sedang, Frontend
file yang diubah:
- frontend/src/app/cctv/page.tsx
- frontend/src/app/maintenance/page.tsx
- frontend/src/app/pemberitahuan/page.tsx
- frontend/src/components/NotificationBell.tsx

## [1.45.0] - 2026-08-10

### Fitur: UI Preferensi Email Granular (Sprint 32 - P7b, Task 13.7b)

perubahan yang dilakukan: **Mengganti toggle email tunggal dengan card berisi master toggle + 3 sub-toggle per-tipe, dengan auto-save.**

before: Halaman profil hanya punya satu toggle "Terima notifikasi email".

after:
- Card "Notifikasi Email" berisi:
  - Master toggle "Aktifkan notifikasi email" — bila mati, ketiga sub-toggle tampil disabled (grayed out).
  - Sub-toggle: Ringkasan Harian (📅), Eskalasi Overdue (⚠️), Stok Suku Cadang Rendah (📦).
- Auto-save: setiap toggle change langsung hit `PUT /api/profile/email-preferences` (debounce 500ms), tampilkan toast sukses/error.
- Inisialisasi nilai dari `GET /api/profile` (user context).
- Interface `User` & `ProfileResponse` tambah `email_daily_digest`, `email_overdue`, `email_stock_low`.

timestamp: 2026-08-10 14:35 WITA
kategori: Sedang, Frontend
file yang diubah:
- frontend/src/app/profile/page.tsx
- frontend/src/context/AuthContext.tsx

## [1.44.0] - 2026-08-10

### Fitur: Perbaikan Marker Peta Koordinat Kosong (Sprint 31 - P2, Task 13.2)

perubahan yang dilakukan: **CCTV tanpa koordinat valid tidak lagi dirender sebagai marker palsu di peta sebaran; ditampilkan di panel terpisah dengan tombol Edit.**

before: CCTV tanpa koordinat valid (atau 0/0) disebar secara acak di sekitar pusat IKN, menyesatkan admin tentang lokasi sebenarnya.

after:
- `CCTVMapItem` (interface): `latitude`/`longitude` → `number | null`.
- `CCTVMap.tsx`: hapus scatter acak; hanya CCTV dengan koordinat valid dirender sebagai marker; panel "X CCTV belum memiliki koordinat" dengan daftar + tombol Edit (`/cctv?edit={id}`); status bar "Menampilkan X dari N CCTV (Y belum memiliki koordinat)".
- `cctv/page.tsx`: interface CCTV koordinat → `number | null`; konversi string→number|null saat submit (`toCoordinateNumber`).
- `field-maintenance/page.tsx` & `FieldCheckLocationMap.tsx`: interface koordinat disesuaikan.

timestamp: 2026-08-10 13:35 WITA
kategori: Sedang, Frontend
file yang diubah:
- frontend/src/components/CCTVMap.tsx
- frontend/src/app/cctv/page.tsx
- frontend/src/app/field-maintenance/page.tsx
- frontend/src/components/FieldCheckLocationMap.tsx

## [1.43.0] - 2026-08-09

### Fitur: UI API Key & Webhook (Task 12.1b + 12.3b - Sprint 29)

perubahan yang dilakukan: **Menambahkan halaman admin API Key dan Webhook untuk integrasi eksternal. Frontend naik `1.42.0` → `1.43.0`.**

before: Belum ada antarmuka untuk mengelola kredensial API key integrasi eksternal maupun endpoint webhook untuk mengirim event ke sistem lain.

after:
- `src/app/api-keys/page.tsx` (baru): halaman admin `/api-keys` — daftar API key (nama, prefix, badge scope, terakhir dipakai, kedaluwarsa, status dicabut), modal "Buat API Key" (nama, scope multi-select, kedaluwarsa), menampilkan key mentah 1× dengan tombol salin, cabut key.
- `src/app/webhooks/page.tsx` (baru): halaman admin `/webhooks` — CRUD endpoint (nama, URL, secret, event multi-select), toggle aktif/nonaktif, tombol test delivery, modal detail + riwayat pengiriman (status, HTTP, latensi, percobaan).
- `src/components/DashboardLayout.tsx`: menu sidebar admin "API Keys" (`KeyRound`) & "Webhooks" (`Webhook`).
- `src/proxy.ts`: rute `/api-keys` & `/webhooks` ditambahkan ke guard admin server-side.

timestamp: 2026-08-09 11:55 WITA
kategori: Besar, Frontend
file yang diubah:
- frontend/src/app/api-keys/page.tsx (baru)
- frontend/src/app/webhooks/page.tsx (baru)
- frontend/src/components/DashboardLayout.tsx
- frontend/src/proxy.ts
- frontend/changelog.md
alasan perubahan: Kebutuhan integrasi & skalabilitas (ROADMAP Task 12.1b + 12.3b) — admin dapat mengelola API key integrasi eksternal dan endpoint webhook beserta riwayat pengiriman.

## [1.42.0] - 2026-08-09

### Fitur: UI Zona & SLA (Task 11.4b + 11.5b - Sprint 28)

perubahan yang dilakukan: **Menambahkan halaman admin zona operasional dan dashboard SLA compliance, plus filter peta per zona. Frontend naik `1.41.0` → `1.42.0`.**

before: Belum ada cara mengelompokkan CCTV & teknisi ke zona/wilayah, tidak ada dashboard kepatuhan SLA, dan peta sebaran tidak dapat difilter per zona.

after:
- `src/app/zona/page.tsx` (baru): halaman admin `/zona` — CRUD zona (tabel nama, deskripsi, jumlah CCTV & teknisi), modal "Atur Penugasan" dengan multi-select CCTV (`/cctv`) dan teknisi (`/users?role=teknisi`), kirim via `POST /zones/:id/assign-cctvs` & `POST /zones/:id/assign-users` (menimpa penugasan zona).
- `src/app/sla/page.tsx` (baru): dashboard admin `/sla` — kartu ringkasan (rata-rata kepatuhan, pelanggaran, teknisi dinilai), chart bar kepatuhan bulanan (recharts), ranking teknisi (progress bar), dan tabel pelanggaran SLA (CCTV, jenis, teknisi, status, deadline, overdue).
- `src/app/sebaran-cctv/page.tsx`: filter dropdown "Semua zona" (dari `/zones/all`) yang mengirim `?zone_id=<uuid>` ke `/cctv` untuk menyaring peta/grid per zona.
- `src/components/DashboardLayout.tsx`: menu sidebar admin "Zona" (`MapPinned`) dan "SLA" (`Target`) setelah "Laporan".
- `src/proxy.ts`: rute `/zona` & `/sla` ditambahkan ke guard admin server-side.
- Dependency `recharts` ditambahkan untuk chart SLA.

timestamp: 2026-08-09 09:50 WITA
kategori: Besar, Frontend
file yang diubah:
- frontend/src/app/zona/page.tsx (baru)
- frontend/src/app/sla/page.tsx (baru)
- frontend/src/app/sebaran-cctv/page.tsx
- frontend/src/components/DashboardLayout.tsx
- frontend/src/proxy.ts
- frontend/package.json (recharts)
- frontend/changelog.md
alasan perubahan: Kebutuhan operasional lanjut (ROADMAP Task 11.4b + 11.5b) — admin dapat mengelompokkan CCTV/teknisi per zona dan memantau kepatuhan target waktu layanan (SLA).

## [1.41.0] - 2026-08-09

### Fitur: UI Email Preferences & Report Builder (Task 11.2b + 11.3b - Sprint 27)

perubahan yang dilakukan: **Menambahkan halaman report builder laporan kustom dan preferensi notifikasi email di profil. Frontend naik `1.40.0` → `1.41.0`.**

before: Belum ada cara membuat laporan kustom yang dapat dikonfigurasi, dan pengguna tidak bisa mengatur apakah menerima notifikasi email.

after:
- `src/app/laporan-kustom/page.tsx` (baru): halaman admin `/laporan-kustom` — wizard 4 langkah (pilih entitas CCTV/Pemeliharaan/Pengecekan/Suku Cadang, pilih kolom, filter status/tanggal/kategori, pilih format CSV/Excel/PDF), tombol "Unduh Laporan", simpan/muat/hapus template, daftar template tersimpan.
- `src/app/profile/page.tsx`: section "Notifikasi Email" — toggle `email_notifications` yang memanggil `PUT /api/profile/email-preferences` (optimistic update + rollback), menampilkan alamat email terdaftar.
- `src/context/AuthContext.tsx`: interface `User` bertambah `email_notifications`.
- `src/components/DashboardLayout.tsx`: menu sidebar "Laporan" (ikon `FileSpreadsheet`) dalam section admin-only setelah "Suku Cadang".
- `src/proxy.ts`: route `/laporan-kustom` ditambahkan ke guard admin server-side.
- `src/lib/api.ts` dipakai untuk template CRUD; unduhan laporan memakai fetch POST dengan cookie + CSRF header (helper lokal di halaman).

timestamp: 2026-08-09 02:40 WITA
kategori: Besar, Frontend
file yang diubah:
- frontend/src/app/laporan-kustom/page.tsx (baru)
- frontend/src/app/profile/page.tsx
- frontend/src/context/AuthContext.tsx
- frontend/src/components/DashboardLayout.tsx
- frontend/src/proxy.ts
- frontend/changelog.md
alasan perubahan: Kebutuhan operasional lanjut (ROADMAP Task 11.2b + 11.3b) — admin dapat membuat laporan kustom dan pengguna dapat mengatur preferensi notifikasi email.

## [1.40.0] - 2026-08-09

### Fitur: UI Inventaris Suku Cadang (Task 11.1 - Sprint 26)

perubahan yang dilakukan: **Menambahkan halaman admin suku cadang + integrasi pencatatan penggunaan di modal pemeliharaan + widget dashboard stok rendah. Frontend naik `1.39.0` → `1.40.0`.**

before: Belum ada antarmuka untuk mengelola stok suku cadang; teknisi tidak bisa mencatat suku cadang yang dipakai saat perbaikan, dan admin tidak bisa melihat komponen yang perlu di-restock.

after:
- `src/app/suku-cadang/page.tsx` (baru): halaman admin `/suku-cadang` — tabel (SKU, Nama, Kategori, Stok, Min Order, Satuan, Harga, Aksi), badge stok (hijau > 2×min / kuning ≤ 2×min / merah < min), pencarian nama/SKU, filter kategori (Kabel, Konektor, Power Supply, Lensa, Storage, Lainnya), modal form tambah/edit, hapus konfirmasi, paginasi; subscribe event `STOCK_LOW` untuk refresh real-time.
- `src/app/maintenance/page.tsx`: section "Suku Cadang Digunakan" di modal detail + mini modal "Catat Penggunaan" (pilih suku cadang, jumlah, catatan); stok otomatis berkurang; peringatan stok rendah setelah pencatatan.
- `src/app/dashboard/page.tsx`: widget "Suku Cadang Stok Rendah" (admin only) — fetch `/spare-parts/low-stock`, daftar 5 terendah dengan badge merah + link ke `/suku-cadang`, refresh real-time via `STOCK_LOW`.
- `src/components/DashboardLayout.tsx`: menu sidebar "Suku Cadang" (ikon `Package`) dalam section admin-only setelah "Audit Log".
- `src/proxy.ts`: route `/suku-cadang` ditambahkan ke guard admin server-side.

timestamp: 2026-08-09 00:50 WITA
kategori: Besar, Frontend
file yang diubah:
- frontend/src/app/suku-cadang/page.tsx (baru)
- frontend/src/app/maintenance/page.tsx
- frontend/src/app/dashboard/page.tsx
- frontend/src/components/DashboardLayout.tsx
- frontend/src/proxy.ts
- frontend/changelog.md
alasan perubahan: Kebutuhan vital operasional (keputusan prioritas ROADMAP Task 11.1) agar admin dapat mengelola stok komponen dan teknisi dapat mencatat suku cadang yang dipakai saat perbaikan CCTV.

## [1.39.0] - 2026-08-07

### Fitur: Multi-View Monitor & Pop-out (Task 10.3 — Fase 10)

perubahan yang dilakukan: **Menambahkan halaman Monitor CCTV grid multi-stream untuk Fase 10 (Kecerdasan Visual & Snapshot). Frontend naik `1.38.0` → `1.39.0`.**

before: Streaming CCTV hanya dapat dilihat per-stream (StreamModal/grid di peta sebaran) tanpa monitor grid multi-view terpusat maupun mode fullscreen/pop-out khusus.

after:
- `src/app/monitor/page.tsx` (baru): halaman `/monitor` (admin-only) — grid multi-stream dengan preset layout 2×2/3×3/4×4; pilih CCTV per sel via dropdown; tiap sel memuat `WHEPPlayer`; fullscreen per sel & seluruh grid; pop-out `window.open('/monitor/pop?path=...')`.
- `src/app/monitor/pop/page.tsx` (baru): halaman `/monitor/pop` — single fullscreen stream dari query parameter `path`.
- `src/components/DashboardLayout.tsx`: menu sidebar "Monitor CCTV" (ikon `MonitorPlay`) setelah "Peta Sebaran CCTV".
- `src/proxy.ts`: route `/monitor` ditambahkan ke guard admin server-side.

timestamp: 2026-08-07 WITA
kategori: Besar, Frontend
file yang diubah:
- frontend/src/app/monitor/page.tsx
- frontend/src/app/monitor/pop/page.tsx
- frontend/src/components/DashboardLayout.tsx
- frontend/src/proxy.ts
- frontend/changelog.md
alasan perubahan: Mewujudkan monitor CCTV multi-view (grid 4/9/16 stream, preset layout 2×2/3×3/4×4, fullscreen, pop-out) sebagai pusat pemantauan live bagi admin — Task 10.3, Fase 10.

## [1.38.0] - 2026-08-07

### Fitur: Snapshot & Timeline Visual (Task 10.1b + 10.2b — Fase 10)

perubahan yang dilakukan: **Menambahkan thumbnail snapshot CCTV dan timeline event visual untuk Fase 10 (Kecerdasan Visual & Snapshot). Frontend naik `1.37.0` → `1.38.0`.**

before: Master CCTV & peta sebaran belum menampilkan snapshot frame; detail CCTV belum memiliki riwayat event visual.

after:
- `src/app/cctv/page.tsx`: kolom/thumbnail snapshot di tabel Master CCTV memakai `snapshot_url` via `AuthenticatedImage`, placeholder ikon `ImageOff` bila belum ada; timeline kronologis di detail CCTV (state `timeline`, `EVENT_TYPE_LABELS` untuk label event — `status_changed`, `maintenance_completed`, `field_check_attention`, `periodic_snapshot`); interface CCTV bertambah `snapshot_url?`.
- `src/components/CCTVMap.tsx`: tooltip & popup marker peta menampilkan gambar snapshot kecil (64×48) via `AuthenticatedImage`, placeholder ikon `Camera` bila belum ada; interface `CCTVMapItem` bertambah `snapshot_url?`.

timestamp: 2026-08-07 WITA
kategori: Sedang, Frontend
file yang diubah:
- frontend/src/app/cctv/page.tsx
- frontend/src/components/CCTVMap.tsx
- frontend/changelog.md
alasan perubahan: Mewujudkan antarmuka visual snapshot (thumbnail di tabel master & tooltip marker peta) dan timeline kronologis di detail CCTV — Task 10.1b + 10.2b, Fase 10.

## [1.37.0] - 2026-08-07

### Fitur: Fase 9 Streaming CCTV — Selesai (Frontend Task 9.4 / 9.5 / 9.6)

perubahan yang dilakukan: **Menutup milestone Fase 9 streaming CCTV real-time pada sisi frontend. Menyimpulkan hasil Sprint 22 (Task 9.4/9.5/9.6): player WHEP, modal stream, grid multi-stream, integrasi marker CCTV di peta sebaran, dan konfigurasi streaming pada Master CCTV. Frontend naik `1.36.0` → `1.37.0` sebagai penutup Fase 9.**

before: UI streaming (Sprint 22) sudah menyediakan `WHEPPlayer`, `StreamModal`, `StreamGridView`, integrasi peta/guard admin, serta konfigurasi streaming di Master CCTV; namun Fase 9 belum ditandai selesai dan versi rilis belum menutup milestone.

after:
- `src/components/WHEPPlayer.tsx` (baru): player WebRTC/WHEP video live (indikator 🔴 LIVE, toggle kualitas SD/HD).
- `src/components/StreamModal.tsx` (baru): modal pemutaran stream saat marker CCTV diklik di peta.
- `src/components/StreamGridView.tsx` (baru): grid view multi-stream; sel tanpa streaming menampilkan placeholder.
- `src/components/CCTVMap.tsx`: integrasi marker CCTV dengan aksi klik → buka `StreamModal`.
- `src/app/sebaran-cctv/page.tsx`: toggle view Peta/Grid, status bar "N kamera streaming aktif", guard role admin.
- `src/app/cctv/page.tsx`: konfigurasi streaming (channel port 1–35, password RTSP toggle visibility, preview URI, checkbox aktifkan) + kolom ikon "Stream" pada tabel master.
- Fase 9 ditandai **6/6 ✅ Selesai** di ROADMAP.

timestamp: 2026-08-07 20:00 WITA
kategori: Besar, Frontend
file yang diubah:
- frontend/src/components/WHEPPlayer.tsx
- frontend/src/components/StreamModal.tsx
- frontend/src/components/StreamGridView.tsx
- frontend/src/components/CCTVMap.tsx
- frontend/src/app/sebaran-cctv/page.tsx
- frontend/src/app/cctv/page.tsx
- frontend/src/lib/api.ts
- frontend/changelog.md
alasan perubahan: Menutup Fase 9 (Streaming CCTV Real-Time) dari sisi frontend — mewujudkan streaming live yang dapat diakses langsung dari peta sebaran & grid view bagi admin, dengan dukungan konfigurasi streaming terpadu pada Master CCTV; versi rilis naik sebagai penutup milestone.

## [1.36.0] - 2026-08-07

### Fitur: Streaming CCTV Real-Time (Task 9.4 / 9.5 / 9.6)

perubahan yang dilakukan: **Menambahkan UI streaming CCTV real-time end-to-end: player WHEP (`WHEPPlayer`) untuk menonton video live, modal stream (`StreamModal`), grid multi-stream (`StreamGridView`), integrasi marker CCTV di peta sebaran, serta konfigurasi streaming (channel port, password RTSP, toggle aktifkan) pada Master CCTV untuk admin.**

before: CCTV yang telah diset `stream_enabled`/`channel_port` di backend belum dapat ditonton langsung dari UI; admin tidak dapat melihat/mengonfigurasi parameter streaming pada formulir Master CCTV.

after:
- `src/components/WHEPPlayer.tsx` (baru): player WHEP untuk video live (indikator 🔴 LIVE, toggle kualitas SD/HD yang me-restart stream).
- `src/components/StreamModal.tsx` (baru): modal pemutaran stream saat marker CCTV diklik di peta.
- `src/components/StreamGridView.tsx` (baru): grid view multi-stream; sel tanpa streaming menampilkan placeholder.
- `src/components/CCTVMap.tsx`: integrasi marker CCTV dengan aksi klik → buka StreamModal untuk CCTV ber-streaming.
- `src/app/sebaran-cctv/page.tsx`: toggle view Peta/Grid, status bar "N kamera streaming aktif", guard role admin.
- `src/app/cctv/page.tsx`: interface CCTV + `channel_port`/`stream_enabled`; formulir "Konfigurasi Streaming" (channel port NVR 1–35, password RTSP dengan toggle visibilitas, preview URI sub stream, checkbox aktifkan streaming); kolom ikon "Stream" (MonitorPlay hijau/abu) pada tabel Master CCTV; kirim `channel_port`/`rtsp_password`/`stream_enabled` saat simpan; edit mengisi `channel_port`/`stream_enabled` dari data dan membiarkan `rtsp_password` kosong.
- `src/lib/api.ts`: tambah label validasi `channel_port`/`rtsp_password`/`stream_enabled` pada `FIELD_LABELS`.

timestamp: 2026-08-07 19:30 WITA
kategori: Besar, Frontend
file yang diubah:
- frontend/src/components/WHEPPlayer.tsx
- frontend/src/components/StreamModal.tsx
- frontend/src/components/StreamGridView.tsx
- frontend/src/components/CCTVMap.tsx
- frontend/src/app/sebaran-cctv/page.tsx
- frontend/src/app/cctv/page.tsx
- frontend/src/lib/api.ts
- frontend/changelog.md
alasan perubahan: Task 9.4/9.5/9.6 ROADMAP (Fase 9) — mewujudkan streaming CCTV real-time yang dapat diakses langsung dari peta sebaran dan grid view bagi admin, dengan dukungan konfigurasi streaming terpadu pada Master CCTV.

## [1.35.0] - 2026-08-07

### Fitur: Geotag Pengecekan Lapangan (Task 8.4)

perubahan yang dilakukan: **Menambahkan pencatatan posisi GPS pada wizard pengecekan lapangan (indikator "Lokasi tercatat", opsional) dan mini-map lokasi pengecekan untuk admin (marker CCTV + marker Lokasi Pengecekan).**

before: Teknisi tidak mengirim koordinat saat pengecekan; admin tidak bisa memverifikasi lokasi pengecekan di peta.

after:
- `src/components/FieldMaintenanceModal.tsx`: saat wizard mode `field` dibuka, `navigator.geolocation.getCurrentPosition` (guard `'geolocation' in navigator`, timeout 10s); simpan `coords`; indikator "Lokasi tercatat" / "Lokasi belum tercatat (opsional)"; kirim `check_latitude`/`check_longitude` pada body `field` (opsional, di-omit bila null); mode `complete` tidak menangkap lokasi.
- `src/components/FieldCheckLocationMap.tsx` (baru): mini-map react-leaflet — marker CCTV (biru) + marker "Lokasi Pengecekan" (oranye), legend, fallback pusat IKN.
- `src/app/field-maintenance/page.tsx`: interface `FieldMaintenance` + `check_latitude`/`check_longitude` & `cctv.latitude|longitude`; bagian "Lokasi Pengecekan" di modal detail (render mini-map via `dynamic(..., { ssr: false })` bila koordinat ada, else "Tidak ada data lokasi").

timestamp: 2026-08-07 18:35 WITA
kategori: Kecil, Frontend
file yang diubah:
- frontend/src/components/FieldMaintenanceModal.tsx
- frontend/src/components/FieldCheckLocationMap.tsx
- frontend/src/app/field-maintenance/page.tsx
- frontend/changelog.md
alasan perubahan: Task 8.4 ROADMAP (Fase 8) — menambah akuntabilitas pengecekan lapangan dengan mencatat posisi GPS teknisi (opsional) dan menampilkan lokasi pengecekan di peta untuk admin.

## [1.34.0] - 2026-08-07

### Fitur: Notifikasi Push Browser (Task 8.3)

perubahan yang dilakukan: **Menambahkan dukungan notifikasi push browser: service worker menangani event `push` dan `notificationclick`, serta komponen opt-in di halaman profil untuk mengaktifkan/menonaktifkan push (subscribe/unsubscribe ke backend).**

before: SIGAP hanya menampilkan notifikasi in-app (bell); belum ada notifikasi OS saat aplikasi tidak dibuka.

after:
- `public/sw.js`: handler event `push` (membaca payload `data.push` → `showNotification`) dan `notificationclick` (fokus/buka tab ke halaman sesuai `reference_type`); `CACHE_VERSION` dinaikkan ke `sigap-shell-v2`.
- `src/components/PushNotificationOptIn.tsx` (baru): cek dukungan (Service Worker + Push API), minta izin, subscribe (`pushManager.subscribe` + `applicationServerKey` dari `NEXT_PUBLIC_VAPID_PUBLIC_KEY`), kirim ke `POST /api/push/subscribe`; nonaktif → `DELETE /api/push/unsubscribe`; status unsupported/denied/on/off.
- `src/app/profile/page.tsx`: bagian "Notifikasi Push" yang merender `PushNotificationOptIn`.

timestamp: 2026-08-07 17:35 WITA
kategori: Sedang, Frontend
file yang diubah:
- frontend/public/sw.js
- frontend/src/components/PushNotificationOptIn.tsx
- frontend/src/app/profile/page.tsx
- frontend/changelog.md
alasan perubahan: Task 8.3 ROADMAP (Fase 8) — memastikan teknisi tidak melewatkan tugas penting via notifikasi OS; opt-in per user dengan fallback ke notifikasi in-app.

## [1.33.0] - 2026-08-07

### Fitur: Kalender Jadwal Pemeliharaan (Task 8.2)

perubahan yang dilakukan: **Menambahkan halaman `/kalender` (admin & teknisi) yang menampilkan jadwal pemeliharaan CCTV sebagai kalender bulanan (implementasi CSS mandiri, tanpa dependency chart eksternal) dengan grid 7 kolom, navigasi bulan, dot warna status per hari, panel detail tugas per hari, dan filter per teknisi untuk admin. Menambah menu sidebar "Kalender".**

before: Teknisi dan admin hanya melihat jadwal pemeliharaan dalam bentuk tabel di halaman `/maintenance` / `/schedules`; belum ada visualisasi kalender bulanan yang intuitif.

after:
- `src/components/CalendarGrid.tsx` (baru): grid kalender bulanan (header Senin–Minggu, 7 kolom × 5-6 baris), navigasi prev/next + "Hari ini", dot status (hijau completed, biru scheduled, kuning ongoing, merah failed/overdue), max 3 dot + "+N", legend; helper `toDayKey`/`formatLongDate`/`formatMonthTitle`/`STATUS_DOT_CLASS`/`STATUS_META`.
- `src/app/kalender/page.tsx` (baru): halaman `/kalender` — ambil data `GET /api/maintenance-logs?start_date=&end_date=&page=1&limit=100` untuk bulan berjalan, kelompokkan per `scheduled_date` (fallback `created_at`); filter teknisi (dropdown admin dari `/users?role=teknisi`; teknisi di-scope backend); panel detail hari (nama CCTV, kode, status badge, tindakan, teknisi); loading/error/empty state.
- `src/components/DashboardLayout.tsx`: menu sidebar "Kalender" (`/kalender`) dengan ikon `Calendar` untuk admin & teknisi.

timestamp: 2026-08-07 16:30 WITA
kategori: Sedang, Frontend
file yang diubah:
- frontend/src/components/CalendarGrid.tsx
- frontend/src/app/kalender/page.tsx
- frontend/src/components/DashboardLayout.tsx
- frontend/changelog.md
alasan perubahan: Task 8.2 ROADMAP (Fase 8) — melengkapi fitur jadwal berulang (5.4) dengan visualisasi kalender yang lebih intuitif daripada tabel, memudahkan admin/teknisi melihat jadwal per hari dan per teknisi.

## [1.32.0] - 2026-08-07

### Fitur: Progressive Web App (PWA) (Task 8.1)

perubahan yang dilakukan: **Menjadikan SIGAP installable dan menyediakan offline shell via PWA: menambah web manifest (`public/manifest.json`), service worker (`public/sw.js`) dengan strategi cache-first untuk shell dan network-first untuk API, ikon aplikasi 192×192 & 512×512 dari logo, registrasi service worker + meta tags PWA di `layout.tsx`, serta header cache untuk `sw.js`/`manifest.json` di `next.config.ts`.**

before: SIGAP hanya aplikasi web browser; tidak bisa di-install ke home screen HP teknisi dan tidak ada cache shell untuk mempercepat loading / akses offline dasar.

after:
- `public/manifest.json` (baru): `name`/`short_name` "SIGAP", `start_url "/dashboard"`, `display "standalone"`, `background_color "#162218"` (Kanopi Malam), `theme_color "#467235"` (Hijau Komando), ikon 192/512 (purpose `any` + `maskable`), `lang "id"`.
- `public/icons/icon-192x192.png` & `public/icons/icon-512x512.png` (baru): ikon PNG di-generate dari `public/nusantara-logo.png` (canvas persegi, latar Kanopi Malam).
- `public/sw.js` (baru): service worker — pre-cache shell (`/`, `/dashboard`, `/login`, manifest, ikon); navigasi HTML network-first (fallback cache offline), aset statis cache-first (pembaruan latar belakang), API (`/api/*`) network-first (fallback cache, else 503 JSON "offline"); `CACHE_VERSION = sigap-shell-v1`; `skipWaiting` + `clients.claim`.
- `src/components/ServiceWorkerRegister.tsx` (baru): client component yang mendaftarkan service worker di `useEffect` (aman SSR).
- `src/app/layout.tsx`: `metadata.manifest`, ikon (incl. apple-touch-icon), `appleWebApp`; export `viewport` (theme-color); render `<ServiceWorkerRegister />`.
- `next.config.ts`: header `Cache-Control: no-cache` + `Service-Worker-Allowed: /` untuk `/sw.js`, dan `public, max-age=3600` untuk `/manifest.json`.

timestamp: 2026-08-07 15:30 WITA
kategori: Sedang, Frontend
file yang diubah:
- frontend/public/manifest.json
- frontend/public/sw.js
- frontend/public/icons/icon-192x192.png
- frontend/public/icons/icon-512x512.png
- frontend/src/components/ServiceWorkerRegister.tsx
- frontend/src/app/layout.tsx
- frontend/next.config.ts
- frontend/changelog.md
alasan perubahan: Task 8.1 ROADMAP (Fase 8) — memberi teknisi akses cepat dari home screen dan loading lebih cepat via cache shell; scope installability + offline shell (bukan full offline dengan IndexedDB).

## [1.31.0] - 2026-08-06

### Fitur: Timeline Ringkasan Operasional (Task 6.4)

perubahan yang dilakukan: **Menambahkan halaman Admin `/ringkasan` yang menampilkan timeline kartu digest harian, summary naratif, metrik tugas baru/selesai/gagal/overdue, pengecekan lapangan beserta attention, perubahan status CCTV, serta paginasi. Menu sidebar `Ringkasan` memakai ikon `ClipboardList` dan route dijaga oleh proxy admin.**

before: Admin belum memiliki halaman timeline untuk membaca ringkasan operasional harian dari satu tempat.

after:
- `src/app/ringkasan/page.tsx`: halaman timeline dengan loading/error/empty state, metrik ringkas, dan paginasi melalui `/dashboard/digest`.
- `src/components/DashboardLayout.tsx`: menu `Ringkasan` khusus Admin dengan ikon `ClipboardList`.
- `src/proxy.ts`: route `/ringkasan` ditambahkan ke guard admin server-side.
- `src/components/NotificationBell.tsx` dan `src/app/pemberitahuan/page.tsx`: tipe `DAILY_DIGEST_AVAILABLE` diberi aksen dan diarahkan ke `/ringkasan` saat dibuka.
- `frontend/changelog.md`: catatan versi fitur baru.

timestamp: 2026-08-06 15:35 WITA
kategori: Sedang, Frontend
file yang diubah:
- frontend/src/app/ringkasan/page.tsx
- frontend/src/components/DashboardLayout.tsx
- frontend/src/proxy.ts
- frontend/src/components/NotificationBell.tsx
- frontend/src/app/pemberitahuan/page.tsx
- frontend/changelog.md
alasan perubahan: Memberi Admin tampilan kronologis yang cepat dipindai untuk memahami kondisi operasional harian tanpa membuka banyak modul.

## [1.30.0] - 2026-08-06

### Fitur: Cetak Laporan Pengecekan Lapangan (Task 6.3)

perubahan yang dilakukan: **Menambahkan tombol "Cetak Laporan" dengan ikon `FileDown` pada modal detail halaman `/field-maintenance`. Tombol hanya terlihat untuk Admin dan mengunduh PDF melalui helper API dengan cookie sesi, respons blob, nama file dari header server, serta state loading/error.**

before: Admin belum dapat mengunduh laporan PDF per-record dari modal detail pengecekan lapangan.

after:
- `src/app/field-maintenance/page.tsx`: integrasi `useAuth`, `downloadApiFile`, tombol `FileDown`, dan feedback toast saat download.
- `frontend/changelog.md`: catatan versi fitur baru.

timestamp: 2026-08-06 15:05 WITA
kategori: Kecil, Frontend
file yang diubah:
- frontend/src/app/field-maintenance/page.tsx
- frontend/changelog.md
alasan perubahan: Memberi Admin cara cepat untuk mengarsipkan hasil pengecekan lapangan tanpa meninggalkan modal detail dan tetap mengikuti pola autentikasi frontend yang sudah ada.

## [1.29.0] - 2026-08-06

### Fitur: Dashboard Kinerja Teknisi (Task 6.2)

perubahan yang dilakukan: **Menambahkan halaman admin `/kinerja-teknisi` dengan filter 7 hari, 30 hari, dan kustom; chart SVG completion rate per teknisi; tabel ranking yang dapat diurutkan pada setiap kolom; serta badge persentase berwarna sesuai ambang kinerja.**

before: Admin belum memiliki halaman khusus untuk membandingkan beban tugas, kecepatan penyelesaian, overdue, dan kualitas pengecekan teknisi.

after:
- `src/app/kinerja-teknisi/page.tsx`: halaman admin dengan filter rentang, state loading/error/empty, tabel ranking sortable, dan badge completion/quality.
- `src/components/TechnicianPerformanceChart.tsx`: chart batang horizontal SVG responsif tanpa dependency chart baru.
- `src/components/DashboardLayout.tsx`: menu `Kinerja Teknisi` dengan ikon `TrendingUp` di bawah Dashboard untuk admin.
- `src/proxy.ts`: route `/kinerja-teknisi` ditambahkan ke guard admin server-side.
- `frontend/changelog.md`: catatan versi fitur baru.

timestamp: 2026-08-06 14:18 WITA
kategori: Sedang, Frontend
file yang diubah:
- frontend/src/app/kinerja-teknisi/page.tsx
- frontend/src/components/TechnicianPerformanceChart.tsx
- frontend/src/components/DashboardLayout.tsx
- frontend/src/proxy.ts
- frontend/changelog.md
alasan perubahan: Menjadikan insight kinerja teknisi dapat dipindai dan dibandingkan admin tanpa mengorbankan akses mobile atau keterbacaan tabel operasional.

## [1.28.0] - 2026-08-06

### Fitur: Badge Skor Kesehatan CCTV (Task 6.1)

perubahan yang dilakukan: **Menambahkan kolom Kesehatan pada tabel Master CCTV dengan badge angka 0–100 dan warna semantik (hijau ≥70, kuning ≥40, merah <40). Marker pada peta sebaran kini memakai skor kesehatan sebagai warna utama; tooltip, popup, dan legend tetap menampilkan angka/rentang agar informasi tidak hanya bergantung pada warna. Sorting tabel mendukung Skor kesehatan**.

before: Tabel Master CCTV dan marker peta hanya menampilkan status operasional (`aktif`, `pemeliharaan`, `nonaktif`), sehingga kondisi kesehatan berbasis riwayat pemeliharaan belum terlihat secara langsung.

after:
- `src/app/cctv/page.tsx`: field `health_score`, badge Kesehatan, helper normalisasi rentang, dan opsi sortir `health_score`.
- `src/components/CCTVMap.tsx`: marker berdasarkan skor, badge skor pada popup, skor pada tooltip, dan legend rentang kesehatan.

timestamp: 2026-08-06 13:22 WITA
kategori: Sedang, Frontend
file yang diubah:
- frontend/src/app/cctv/page.tsx
- frontend/src/components/CCTVMap.tsx
- frontend/changelog.md
alasan perubahan: Membuat indikator kesehatan CCTV dapat dipindai cepat pada inventaris dan peta tanpa menghilangkan status operasional perangkat.

## [1.27.0] - 2026-08-06

### Fitur: Halaman Jadwal Pemeliharaan Berulang (Recurring) - Task 5.4

perubahan yang dilakukan: **Halaman baru `/schedules` khusus admin untuk mengelola jadwal pemeliharaan berulang: tabel jadwal (CCTV, pekerjaan, teknisi, frekuensi Harian/Mingguan/Bulanan, jadwal berikutnya, status Aktif/Nonaktif) + tindakan toggle aktif/nonaktif (ikon `Power`), ubah (`Settings`), dan hapus (`Trash2`) dengan konfirmasi; tombol "Buat Jadwal" membuka modal form (pilih CCTV, tindakan, frekuensi, teknisi opsional — teknisi kosong berarti penugasan oleh admin). Filter status aktif/nonaktif dan paginasi disediakan. Menu sidebar "Jadwal" (ikon `CalendarClock`) ditambahkan khusus admin, dan `/schedules` terdaftar sebagai path admin di proxy middleware**.

before: Tidak ada cara untuk mengelola pemeliharaan berulang dari antarmuka — pekerjaan pemeliharaan hanya dicatat manual per kejadian di halaman Pemeliharaan.

after:
- `app/schedules/page.tsx` (baru): `useState` untuk data, filter `is_active`, pagination, modal form tambah/ubah (select CCTV, textarea tindakan, select frekuensi dengan keterangan, select teknisi opsional), aksi toggle/ubah/hapus dengan `useConfirm`, toast umpan balik, dan refresh real-time via `MAINTENANCE_CREATED` websocket.
- `components/DashboardLayout.tsx`: menu "Jadwal" (`CalendarClock`) ditambahkan di bagian admin.

timestamp: 2026-08-06 10:50 WITA
kategori: Kecil, Frontend
file yang diubah:
- frontend/src/app/schedules/page.tsx
- frontend/src/components/DashboardLayout.tsx
- frontend/src/proxy.ts
- frontend/changelog.md

## [1.26.0] - 2026-08-05

### Fitur: Tombol "Alihkan" untuk Re-assign Tugas Pemeliharaan — Task 5.3

perubahan yang dilakukan: **Halaman Pemeliharaan kini menampilkan tombol "Alihkan" (ikon `ArrowLeftRight`) pada baris tugas admin berstatus `ongoing` — di samping tombol "Tugaskan" (`UserPlus`) yang tetap untuk log `scheduled`. Klik membuka modal penugasan dengan dropdown teknisi (`GET /api/users?role=teknisi`) yang sudah ada; tugas `ongoing` menampilkan peringatan "Tugas ini sedang dikerjakan oleh [nama]. Yakin ingin mengalihkan?" dan tombol submit berlabel "Alihkan". Keduanya memanggil `PUT /api/maintenance-logs/:id/assign` yang sama, lalu toast sukses/error dan refresh tabel (error 400 "Teknisi sudah ditugaskan pada tugas ini" tampil via toast)**.
before: Tombol aksi hanya memunculkan "Tugaskan" (`UserPlus`) untuk log `scheduled`; tugas `ongoing` tidak punya aksi pengalihan sehingga admin tidak bisa memindahkan tugas yang sedang dikerjakan ke teknisi lain.
after:
- `app/maintenance/page.tsx`: kondisi tombol assign diperluas `log.status === 'scheduled' || log.status === 'ongoing'` dengan ikon & label berbeda per status (`ArrowLeftRight` / "Alihkan ke teknisi lain" untuk ongoing); impor ikon `ArrowLeftRight` ditambahkan.
- Modal assign: judul dinamis ("Tugaskan ke Teknisi" / "Alihkan ke Teknisi Lain"); kotak info berubah menjadi peringatan kuning bertuliskan nama teknisi saat ini untuk status `ongoing`; tombol submit berlabel "Tugaskan"/"Alihkan" sesuai status.
- `handleAssignLog`: toast sukses disesuaikan ("Tugas berhasil dialihkan ke teknisi lain." saat re-assign); error dari backend ditampilkan via toast.
timestamp: 2026-08-05 20:33 WITA
kategori: Kecil, Frontend
file yang diubah:
- frontend/src/app/maintenance/page.tsx
- frontend/changelog.md

## [1.25.13] - 2026-08-05

### Perubahan: Opsi Status "Rusak" Dihapus — Digabung ke "Nonaktif"

perubahan yang dilakukan: **Status `rusak` digabung ke `nonaktif` (keputusan backend [0.28.3]) karena keduanya selalu ditampilkan sebagai "Nonaktif" dan dilindungi sinkronisasi otomatis. Opsi "Rusak" dihapus dari dropdown form dan filter, `toStatusValue` memetakan `RUSAK`/`BROKEN` → `nonaktif` sehingga baris legacy rusak tetap tampil benar, dan enum status pada komentar diperbarui menjadi `aktif` / `pemeliharaan` / `nonaktif`**
before: Dropdown form & filter menyediakan 4 opsi (`Aktif/Nonaktif/Rusak/Pemeliharaan`); `toStatusValue` memetakan `RUSAK`/`BROKEN` → `rusak`; komentar enum masih menyebut 4 status.
after:
- `app/cctv/page.tsx`: opsi `<option value="rusak">Rusak</option>` dihapus dari form Tambah/Ubah dan modal "Sortir & Filter" — tersisa `aktif` / `nonaktif` / `pemeliharaan`.
- `app/cctv/page.tsx` (`toStatusValue`): `RUSAK`/`BROKEN` kini dipetakan ke `nonaktif`; komentar enum backend diperbarui.
- `lib/api.ts`: contoh kalimat terjemahan validasi & komentar format `errors[]` disesuaikan ("aktif, pemeliharaan, nonaktif").
- Halaman dashboard/sebaran tidak perlu berubah — badge legacy `rusak` sudah tampil "Nonaktif" sejak sebelumnya.
timestamp: 2026-08-05 WITA
kategori: Kecil, Frontend
file yang diubah:
- frontend/src/app/cctv/page.tsx
- frontend/src/lib/api.ts
- frontend/changelog.md

## [1.25.12] - 2026-08-05

### Perbaikan: Dropdown Status Form & Filter CCTV Selaras dengan Enum Backend (lowercase Indonesia)

perubahan yang dilakukan: **Opsi `<select>` status di form Tambah/Ubah CCTV dan filter Sortir memakai nilai Inggris huruf besar (`ACTIVE`/`INACTIVE`/`MAINTENANCE`), sementara database (dan enum backend `oneof=aktif rusak pemeliharaan nonaktif`) memakai huruf kecil Indonesia. Saat mengedit CCTV berstatus `pemeliharaan`, controlled select tidak cocok dengan opsi mana pun sehingga browser menampilkan "Aktif" seolah terpilih padahal nilai tersimpan tetap `pemeliharaan` → perubahan status tidak tersimpan walau toast sukses. Nilai opsi kini memakai enum backend dan status diisi normalisasi saat form dibuka**
before: `<select value={formData.status}>` dengan opsi `ACTIVE/INACTIVE/MAINTENANCE`, sedangkan `formData.status` diisi nilai mentah DB (`pemeliharaan`) — select tampil fallback ke opsi pertama ("Aktif") dan nilai sebenarnya tidak berubah; setelan status manual pemeliharaan→aktif seolah "berhasil" padahal tidak.
after:
- `app/cctv/page.tsx`: helper `toStatusValue` menormalisasi nilai status (huruf kecil Indonesia & legacy huruf besar Inggris → enum `aktif/nonaktif/pemeliharaan/rusak`); dipakai saat mengisi form edit (`handleOpenForm`) sehingga dropdown menampilkan status yang sebenarnya.
- Opsi status di form (Tambah/Ubah) diganti: `aktif`/`nonaktif`/`rusak`/`pemeliharaan` (termasuk opsi baru "Rusak" agar baris berstatus `rusak` tidak mengalami fallback display yang sama); default form baru juga `aktif`.
- Opsi status di modal "Sortir & Filter" diganti ke lowercase — sinkron dengan query `status=` backend yang mencocokkan persis (`status = ?`) agar filter benar-benar bekerja untuk baris lowercase.
timestamp: 2026-08-05 WITA
kategori: Kecil, Frontend
file yang diubah:
- frontend/src/app/cctv/page.tsx
- frontend/changelog.md

## [1.25.11] - 2026-08-05

### Peningkatan: Pesan Error Validasi yang Jelas (rincian per field) di Form CCTV

perubahan yang dilakukan: **Sebelumnya saat penyimpanan form CCTV ditolak backend, toast hanya menampilkan "Validasi input gagal" tanpa menjelaskan isian mana yang bermasalah — padahal backend sudah mengirim rincian per field (`errors: [{field, tag, value}]`) namun dibuang oleh lapisan API. Kini `apiRequest` menangkap rincian tersebut dan menerjemahkannya menjadi kalimat jelas per field (mis. "Status harus berupa salah satu dari: aktif, rusak, pemeliharaan, nonaktif"); pada form Tambah/Ubah CCTV, daftar isian yang bermasalah ditampilkan langsung di dalam modal (kotak merah dengan bullet) sehingga pengguna langsung tahu apa yang harus diperbaiki — tidak hanya lewat toast**
before: `apiRequest` hanya melempar `Error(data.message)` sehingga rincian `errors[]` dari backend hilang; toast form CCTV menampilkan pesan generik "Validasi input gagal" (via `toast.error(err.message)`).
after:
- `lib/api.ts`: tipe `ApiResponse` bertambah `errors?: ApiFieldError[]`; kelas baru `ApiValidationError` (membawa `.errors` terstruktur); fungsi `formatValidationMessages(errors)` menerjemahkan `{field, tag, value}` → kalimat Bahasa Indonesia per field (label field via `FIELD_LABELS`, fallback transformasi CamelCase; tag didukung: `required`, `min`, `max`, `oneof`, `ip`, `ipv4`, `email`, `alpha`, `alphanum`, `uuid`/`uuid4`, `strongpassword`, `datetime`, `eqfield`).
- Saat respons error memuat `errors[]`, `apiRequest` melempar `ApiValidationError` dengan `.message` yang sudah berisi ringkasan per field — otomatis membuat toast halaman lain ikut informatif tanpa perubahan di tiap halaman.
- `app/cctv/page.tsx`: state `formErrors`; saat `ApiValidationError` diterima, daftar isian yang bermasalah dirender di dalam modal form (kotak `border-red-500/20 bg-red-500/10` berisi bullet "Periksa isian berikut sebelum menyimpan:"), state di-reset saat form dibuka/disimpan sukses; error non-validasi tetap via toast.
timestamp: 2026-08-05 WITA
kategori: Kecil, Frontend
file yang diubah:
- frontend/src/lib/api.ts
- frontend/src/app/cctv/page.tsx
- frontend/changelog.md



## [1.25.10] - 2026-08-04

### Peningkatan: Badge Profil Sejajar dengan Status Layanan + Tampilan Nama & Peran Monokrom

perubahan yang dilakukan: **Tombol profil di header Dashboard diposisikan sejajar (satu baris) dengan lonceng notifikasi dan indikator "Layanan normal"; di samping badge (foto) kini ditampilkan nama akun dan peran — teks monokrom (putih/hitam sesuai tema, tanpa warna-warni) yang dipisahkan garis pembatas (`border-l`) dari badge; avatar fallback (inisial) juga dibuat netral; dialog konfirmasi keluar dipindah ke dalam komponen `ProfileMenu` (diportal ke `document.body`) sehingga badge bisa dipakai langsung di halaman Dashboard tanpa prop khusus**
before: badge profil (hanya foto) berada di baris terpisah di pojok kanan atas area konten (semua halaman); fallback avatar & teks peran memakai warna aksen berwarna; dialog konfirmasi keluar dikelola di `DashboardLayout` dengan prop `onLogoutRequest`.
after:
- Header Dashboard (desktop): `[Lonceng] [● Layanan normal] [Foto | Nama + Peran]` — satu baris sejajar (`hidden md:block`; di mobile tetap lewat header mobile).
- Teks nama `text-foreground` & peran `text-muted` (monokrom mengikuti tema terang/gelap), peran ditampilkan kapitalisasi ("Admin"/"Teknisi"), garis pembatas `border-l border-border` di antara badge dan teks.
- Fallback avatar (belum ada foto): latar `bg-muted/60` + inisial `text-foreground` — netral, tidak berwarna.
- Dropdown tetap: identitas + "Profil" → `/profile` + "Keluar" (merah kontras) — kini memunculkan dialog konfirmasi yang dikelola sendiri oleh `ProfileMenu` via `createPortal`; `DashboardLayout` dibersihkan dari state/modal konfirmasi keluar (tombol Keluar lama di sidebar sudah dihapus di [1.25.9]).
- Halaman selain Dashboard: badge tetap di pojok kanan atas area konten (desktop) dengan tampilan yang sama; header mobile tetap badge saja (`showDetails={false}`).
timestamp: 2026-08-04 WITA
kategori: Sedang, Frontend
file yang diubah:
- frontend/src/components/ProfileMenu.tsx
- frontend/src/components/DashboardLayout.tsx
- frontend/src/app/dashboard/page.tsx
- frontend/changelog.md



## [1.25.9] - 2026-08-04

### Peningkatan: Badge Profil Global (foto + dropdown) & Foto Profil di Halaman Profil

perubahan yang dilakukan: **Badge profil kini hanya menampilkan foto (avatar bulat; fallback inisial bila belum ada foto) dan dipindah menjadi elemen global di komponen layout — pojok kanan atas area konten (desktop) dan header mobile — di semua halaman. Klik badge membuka dropdown menu: "Profil" (→ `/profile`) dan "Keluar" (warna kontras merah). Tombol "Keluar" di sidebar dihapus dan dipindah ke dropdown ini. Halaman `/profile` mendapat fitur unggah/ubah/hapus foto profil**
before: badge profil (avatar inisial + nama + badge peran) hanya ada di header Dashboard; tombol "Keluar" ada di sidebar; halaman profil hanya bisa mengubah nama tampilan.
after:
- `ProfileMenu.tsx` (baru): badge bulat berisi foto profil (diambil via endpoint JWT; jika gagal/tidak ada → inisial), dropdown dengan identitas pengguna, link "Profil", dan tombol "Keluar" (teks merah, hover merah muda). Dropdown ditutup oleh klik luar/Escape; item Keluar tetap memicu dialog konfirmasi keluar yang sama seperti sebelumnya.
- `DashboardLayout.tsx`: tombol "Keluar" dihapus dari sidebar; `ProfileMenu` dirender di baris atas area konten (desktop) dan di header mobile (bersebelahan dengan tombol tema); dialog konfirmasi keluar tetap dipertahankan.
- `dashboard/page.tsx`: card profil di header dashboard dihapus (digantikan badge global); indikator "Layanan normal" + lonceng tetap.
- `profile/page.tsx`: section identitas menampilkan foto profil (atau ikon), tombol "Tambah/ubah Foto" (unggah JPEG/PNG/WebP maks 2MB → `POST /api/upload` → `PUT /api/auth/profile` dengan `profile_photo_url`), dan tombol "Hapus Foto". Simpan nama tampilan kini selalu mengirim `profile_photo_url` terkini sehingga foto tidak hilang.
- `AuthContext.tsx`: tipe `User` ditambah `profile_photo_url`.
- Backend pendukung: changelog [0.27.0] (kolom `users.profile_photo_url` + validasi path + pengecualian pembersihan upload yatim).
timestamp: 2026-08-04 WITA
kategori: Sedang, Frontend
file yang diubah:
- frontend/src/components/ProfileMenu.tsx (baru)
- frontend/src/components/DashboardLayout.tsx
- frontend/src/app/dashboard/page.tsx
- frontend/src/app/profile/page.tsx
- frontend/src/context/AuthContext.tsx
- frontend/changelog.md



## [1.25.8] - 2026-08-04

### Peningkatan: Card Profil di Dashboard + Pemberitahuan di Sidebar

perubahan yang dilakukan: **Card profil pengguna dipindah dari sidebar ke header halaman Dashboard (di sebelah indikator "Layanan normal"), tampilan diubah tanpa border dan berbentuk badge profil (avatar inisial + nama + badge peran, mengarah ke halaman `/profile`); tombol "Muat ulang" dihapus; menu "Pemberitahuan" di sidebar kini menampilkan badge jumlah notifikasi belum dibaca (hilang saat sudah dibaca) dan membuka modal daftar pemberitahuan — item notifikasi bisa diklik untuk membuka modal informasi lengkap (sebelumnya hanya navigasi langsung)**
before: card profil (avatar ikon + nama + badge peran) tampil di sidebar dengan border; indikator status "Layanan normal" berdampingan dengan tombol "Muat ulang"; menu "Pemberitahuan" di sidebar berisi link biasa tanpa indikator; item notifikasi pada dropdown lonceng langsung menavigasi ke halaman.
after:
- Header dashboard: `[Lonceng] [● Layanan normal] [Card profil tanpa border — avatar inisial (badge) + nama + badge peran, klik → /profile]`; tombol "Muat ulang" dihapus (refresh tetap otomatis via websocket + polling).
- Sidebar: item "Pemberitahuan" adalah tombol yang menampilkan badge merah berisi angka unread (sinkron via `/notifications/unread-count`, polling 60 detik + event websocket `NOTIFICATION_CREATED`; ter-update juga saat notifikasi dibaca/ditandai semua).
- Klik item "Pemberitahuan" di sidebar → modal daftar pemberitahuan (20 terbaru, filter preview): tiap item bisa diklik → modal informasi lengkap (judul, pesan, waktu lengkap) dengan tombol "Lihat detail" (menandai dibaca + navigasi ke halaman terkait); tersedia "Tandai semua dibaca" dan "Lihat riwayat lengkap" → `/pemberitahuan`.
- Helper notifikasi (`NotificationItem`, `typeStyles`, `getTargetRoute`, `formatNotifTime`) di-export dari `NotificationBell.tsx` agar dipakai bersama komponen baru `NotificationsMenu.tsx`; modal dirender via `createPortal` (aman dari transform sidebar).
- Card profil lama di sidebar dihapus (posisi baru di dashboard; halaman `/profile` tidak berubah).
timestamp: 2026-08-04 WITA
kategori: Sedang, Frontend
file yang diubah:
- frontend/src/app/dashboard/page.tsx
- frontend/src/components/DashboardLayout.tsx
- frontend/src/components/NotificationBell.tsx
- frontend/src/components/NotificationsMenu.tsx (baru)
- frontend/changelog.md



## [1.25.7] - 2026-08-04

### Peningkatan: Sortir & Filter Daftar CCTV dalam Modal Popup

perubahan yang dilakukan: **Select filter status, filter indoor/outdoor, dan select sortir di halaman `/cctv` disembunyikan dari tampilan utama dan dipindah ke modal "Sortir & Filter" (pola sama dengan `/maintenance`, `/field-maintenance`, `/audit-logs`); kotak pencarian tetap tampil; opsi pengurutan diperluas (sebelumnya hanya nama A-Z/Z-A, kini 5 kolom)**
before: 3 select (status, indoor/outdoor, urutan) tampil di grid baris dengan kotak pencarian; opsi urutan hanya "Terbaru / Nama A-Z / Nama Z-A" (mapping `sort_by=nama`).
after:
- Tombol "Sortir" (ikon `SlidersHorizontal`) di header bagian "Daftar CCTV" + badge jumlah filter/sortir aktif.
- Modal "Sortir & Filter": kolom pengurutan (Waktu ditambahkan/Nama/Kode CCTV/Status/Lokasi area — whitelist backend: id, kode_cctv, nama, lokasi_area, status, created_at, updated_at) + arah (terbaru→terlama / terlama→terbaru) + filter status (Aktif/Nonaktif/Pemeliharaan) + lokasi pemasangan (Indoor/Outdoor); pola draft → Terapkan/Batal/Reset; Escape/klik luar menutup + scroll-lock body.
- State `sortMode` (`newest|name-asc|name-desc`) diganti `sortBy` + `sortOrder`; `fetchCCTVs` mengirim `sort_by` & `order`; default `created_at desc` setara perilaku lama "Terbaru".
- Kotak pencarian tetap tampil; seleksi massal & fitur lain tidak berubah.
timestamp: 2026-08-04 WITA
kategori: Sedang, Frontend
file yang diubah:
- frontend/src/app/cctv/page.tsx
- frontend/changelog.md



## [1.25.6] - 2026-08-04

### Peningkatan: Sortir & Filter Audit Log dalam Modal Popup + Posisi Menu Pemberitahuan

perubahan yang dilakukan: **Filter modul & rentang tanggal di halaman `/audit-logs` disembunyikan dari tampilan utama dan dipindah ke modal "Sortir & Filter" (pola sama dengan `/maintenance` & `/field-maintenance`); kotak pencarian tetap tampil; menu sidebar "Pemberitahuan" dipindah ke posisi setelah "Audit Log" dan tepat di atas tombol "Keluar"**
before: filter bar 3 kontrol (modul, dari tanggal, sampai tanggal) tampil di bawah kotak pencarian; menu Pemberitahuan berada setelah "Pemeliharaan CCTV" (di tengah daftar menu). Selain itu halaman audit-logs tidak pernah menampilkan baris data karena backend merespons `data: [array]` sementara frontend membaca `data.rows` (bug tersembunyi — diperbaiki lewat backend changelog [0.26.2]).
after:
- Tombol "Sortir" (ikon `SlidersHorizontal`) di header bagian "Riwayat Aktivitas" + badge jumlah filter/sortir aktif.
- Modal "Sortir & Filter": kolom pengurutan (Waktu kejadian/Aksi/Modul/Pengguna/Peran — whitelist backend) + arah + filter modul (AUTH/CCTV/MAINTENANCE/USER) + rentang tanggal; pola draft → Terapkan/Batal/Reset; Escape/klik luar menutup + scroll-lock.
- Kotak pencarian teks tetap tampil langsung; tombol ekspor CSV tetap di baris terpisah.
- `fetchAuditLogs` kini mengirim `sort_by` & `order`.
- Sidebar: "Pemberitahuan" dipindah ke akhir daftar menu utama (bawah "Audit Log" untuk admin), tepat di atas tombol "Keluar" — untuk semua role.
timestamp: 2026-08-04 WITA
kategori: Sedang, Frontend
file yang diubah:
- frontend/src/app/audit-logs/page.tsx
- frontend/src/components/DashboardLayout.tsx
- frontend/changelog.md



## [1.25.5] - 2026-08-04

### Redesign: Modal Detail Pengguna di Halaman `/users`

perubahan yang dilakukan: **Modal detail pengguna didesain ulang dari daftar field polos (ikon + 4 baris grid) menjadi kartu identitas bertingkat: header avatar inisial (warna sesuai peran), nama + badge peran, username mono; panel rincian akun (email, tanggal akun dibuat) dalam satu grid berlapis; dan footer aksi cepat (Ubah data, Atur ulang kata sandi, Hapus pengguna — merah terpisah)**
before: `Modal` generik dengan judul "Detail Pengguna", satu baris berisi ikon `UserCog` + grid 2 kolom berisi 4 pasangan label-nilai (Nama lengkap, Username, Email, Peran); tanpa tanggal dibuat, tanpa aksi, tanpa hierarki visual.
after:
- Header identitas: avatar inisial 56px (rounded-xl) — `bg-warning/10 text-warning` untuk admin, `bg-primary/10 text-primary` untuk teknisi; nama (text-xl bold) + chip pill peran (kuning/hijau sesuai tabel); username `@username` mono; tombol tutup.
- Panel rincian akun: satu `dl` grid 2 kolom dengan pemisah border (pola sama dengan modal detail pemeliharaan) — Email (ikon `Mail`, `break-all`) & Akun dibuat (ikon `CalendarDays`, format `id-ID` panjang).
- Footer aksi cepat setelah separator: "Hapus pengguna" (bordered merah + hover `bg-error/10`) di kiri; "Ubah data" (secondary) & "Atur ulang kata sandi" (primary) di kanan — langsung berpindah ke modal edit/password yang sudah ada (tanpa fetch ulang), hapus memakai confirm dialog lalu menutup modal & memuat ulang daftar.
- Overlay kini `backdrop-blur-sm` + shadow besar (pola modal terbaru); layout responsif (footer menumpuk di mobile).
timestamp: 2026-08-04 WITA
kategori: Sedang, Frontend
file yang diubah:
- frontend/src/app/users/page.tsx
- frontend/changelog.md



## [1.25.4] - 2026-08-04

### Peningkatan: Sortir & Filter Pengecekan Lapangan Dipindah ke Modal Popup

perubahan yang dilakukan: **Filter hasil pengecekan (status) & rentang tanggal di halaman `/field-maintenance` disembunyikan dari tampilan utama dan dipindah ke dalam modal "Sortir & Filter" (pola sama dengan halaman `/maintenance`); modal kini juga berisi pengaturan pengurutan (kolom + arah) yang dikirim ke backend lewat `sort_by` & `order`; kotak pencarian teks tetap tampil di atas tabel**
before: status select + dua input tanggal tampil langsung di atas tabel (grid 4 kolom); tidak ada kontrol pengurutan (list selalu `sort_by=created_at&order=desc` dari default backend).
after:
- Tombol "Sortir" (ikon `SlidersHorizontal`) di header bagian "Daftar Hasil Pengecekan" + badge jumlah filter/sortir aktif.
- Modal "Sortir & Filter": kolom pengurutan (Waktu pengecekan/Waktu selesai/Hasil keseluruhan — whitelist backend: id, cctv_id, technician_id, overall_status, created_at, completed_at) + arah (terbaru→terlama / terlama→terbaru) + filter lengkap (hasil pengecekan, rentang tanggal) dengan pola draft → Terapkan/Batal/Reset.
- Kotak pencarian (CCTV/kode/catatan) tetap terlihat langsung untuk alur pencarian cepat.
- `fetchEntries` kini menyertakan `sort_by` & `order`; modal menutup via Escape/klik luar dengan scroll-lock body.
timestamp: 2026-08-04 WITA
kategori: Sedang, Frontend
file yang diubah:
- frontend/src/app/field-maintenance/page.tsx
- frontend/changelog.md



## [1.25.3] - 2026-08-04

### Peningkatan: Sortir & Filter Pemeliharaan Dipindah ke Modal Popup

perubahan yang dilakukan: **Grid filter (CCTV, Status, rentang tanggal) di halaman `/maintenance` disembunyikan dari tampilan utama dan dipindah ke dalam modal "Sortir & Filter" yang terbuka lewat tombol Sortir di header tabel; modal kini juga berisi pengaturan pengurutan (kolom + arah) yang dikirim ke backend lewat `sort_by` & `order`**
before: filter tampil langsung di atas tabel; tidak ada kontrol pengurutan sama sekali (list selalu `sort_by=created_at&order=desc` dari default backend); filter bar memakan ruang vertikal halaman.
after:
- Tombol "Sortir" (ikon `SlidersHorizontal`) di header bagian "Riwayat Pemeliharaan"; badge angka menunjukkan jumlah filter/sortir yang aktif; tombol dinonaktifkan saat data sedang dimuat.
- Modal "Sortir & Filter": kolom pengurutan (Waktu pencatatan/Waktu terjadwal/Waktu selesai/Status/Pekerjaan A-Z) + arah (terbaru→terlama / terlama→terbaru) + filter lengkap (CCTV, Status, rentang tanggal) dengan pola draft → tombol Terapkan (perubahan diterapkan sekaligus, halaman kembali ke halaman 1) / Batal (draft dibuang) / Reset (kembali ke `created_at desc`, semua filter kosong).
- `fetchLogs` kini menyertakan `sort_by` & `order` (whitelist backend: id, cctv_id, technician_id, action_taken, notes, status, scheduled_date, completed_at, created_at, updated_at).
- Modal memakai pola yang sama dengan modal lain di halaman (backdrop blur, tutup via Escape/klik luar, scroll-lock body).
timestamp: 2026-08-04 WITA
kategori: Sedang, Frontend
file yang diubah:
- frontend/src/app/maintenance/page.tsx
- frontend/changelog.md



## [1.25.2] - 2026-08-04

### Bugfix: Pemberitahuan Tidak Pernah Tampil (Salah Bungkus Respons API)

perubahan yang dilakukan: **`NotificationBell` dan halaman `/pemberitahuan` membaca daftar dari `res.data?.data`, padahal respons `GET /api/notifications` menempatkan array di `data` (kini `data.rows` sesuai konvensi list SIGAP) — hasilnya daftar selalu kosong ("Belum ada pemberitahuan") meskipun notifikasi tersimpan di database; pembacaan diubah menjadi `res.data?.rows` (dan pagination dari `res.data.pagination`)**
before:
- `NotificationBell.tsx`: `apiRequest<{ data: NotificationItem[] }>` lalu `setNotifications(res.data?.data ?? [])` — `res.data` adalah array, sehingga `.data` selalu `undefined` → dropdown kosong.
- `app/pemberitahuan/page.tsx`: membaca `res.data?.data` & `res.data?.pagination` dengan cara yang sama — halaman selalu kosong & total halaman 0.
- Akibat: badge unread-count tetap benar (bentuk `data: { unread_count }` cocok), tapi daftar & riwayat tidak pernah tampil untuk admin maupun teknisi.
after:
- `NotificationBell.tsx`: `apiRequest<{ rows: NotificationItem[] }>` → `setNotifications(res.data?.rows ?? [])`.
- `app/pemberitahuan/page.tsx`: `NotificationsResponse` kini `{ rows, pagination }`; `setNotifications(res.data?.rows ?? [])`, total rows/pages dari `res.data?.pagination`.
- Backend `GET /api/notifications` diselaraskan ke konvensi `data: { rows, pagination }` (lihat backend changelog [0.26.1]).
timestamp: 2026-08-04 WITA
kategori: Bugfix, Frontend
file yang diubah:
- frontend/src/components/NotificationBell.tsx
- frontend/src/app/pemberitahuan/page.tsx
- frontend/changelog.md



## [1.25.1] - 2026-08-04

### Pemberitahuan: Lonceng di Dashboard, Menu Sidebar & Halaman `/pemberitahuan` Terpisah

perubahan yang dilakukan: **Posisi lonceng notifikasi dipindah dari header/sidebar global ke halaman Dashboard (di samping indikator layanan); dropdown lonceng kini hanya menampilkan 3 pemberitahuan teratas + tombol "Lihat lainnya" yang mengarah ke halaman Pemberitahuan baru (`/pemberitahuan`, ada di menu sidebar) berisi riwayat lengkap milik user yang login saja — teknisi tidak bisa melihat pemberitahuan teknisi lain (scope tetap dari klaim JWT di backend)**
before:
- `NotificationBell` dirender di header mobile & sidebar desktop (`DashboardLayout`) untuk semua halaman; dropdown menampilkan 10 notifikasi dan tidak ada halaman daftar lengkap.
- Tidak ada menu/halaman khusus pemberitahuan.
after:
- `src/components/DashboardLayout.tsx`: `NotificationBell` dihapus dari header mobile & sidebar; item navigasi baru "Pemberitahuan" (`/pemberitahuan`, ikon Bell) ditambahkan untuk admin & teknisi.
- `src/components/NotificationBell.tsx`: preview dropdown dibatasi 3 notifikasi terbaru (`limit=3`); tombol "Lihat lainnya" di footer dropdown → `router.push('/pemberitahuan')`; badge unread, real-time `NOTIFICATION_CREATED`, dan polling 60 detik tetap.
- `src/app/dashboard/page.tsx`: `NotificationBell` dirender di header halaman Dashboard (sebelum indikator "Layanan normal").
- `src/app/pemberitahuan/page.tsx` (baru): halaman daftar lengkap pemberitahuan milik user — paginasi (10/halaman), filter "Belum dibaca saja", tombol "Tandai semua dibaca", klik item → tandai dibaca + navigasi ke `/maintenance` atau `/field-maintenance`, refresh real-time, empty state.
timestamp: 2026-08-04 WITA
kategori: Sedang, Frontend
file yang diubah:
- frontend/src/components/DashboardLayout.tsx
- frontend/src/components/NotificationBell.tsx
- frontend/src/app/dashboard/page.tsx
- frontend/src/app/pemberitahuan/page.tsx (baru)
- frontend/changelog.md



## [1.25.0] - 2026-08-04

### Notifikasi In-App Real-Time: NotificationBell di Header & Sidebar (Fase 5.1)

perubahan yang dilakukan: **Komponen `NotificationBell` baru — ikon lonceng dengan badge jumlah belum dibaca, dropdown daftar notifikasi (10 terbaru), tombol "Tandai semua dibaca", klik item → tandai dibaca + navigasi ke halaman terkait (`/maintenance` atau `/field-maintenance`); terima notifikasi baru real-time via WebSocket (`NOTIFICATION_CREATED`) dengan toast + refresh badge, fallback polling 60 detik saat koneksi terputus**
before: Tidak ada akses notifikasi di antarmuka; penugasan/penyelesaian tugas hanya terlihat saat membuka halaman terkait secara manual.
after:
- `src/components/NotificationBell.tsx` (baru): bell + badge `99+`; dropdown (border/rounded konsisten desain SIGAP, `line-clamp-2`, waktu `id-ID`); subscribe `NOTIFICATION_CREATED` → `toast.info(title)` + muat ulang; polling `unread-count` tiap 60 detik; tutup saat klik luar/Escape; peta tipe → rute tujuan (`field-maintenance` → `/field-maintenance`, lainnya → `/maintenance`); mark-read fire-and-forget agar navigasi tidak tertahan.
- `src/components/DashboardLayout.tsx`: `NotificationBell` dirender di header mobile (sebelum tombol tema) dan area kontrol sidebar desktop (sebelum tombol tema/collapse) — tersedia untuk admin & teknisi.
timestamp: 2026-08-04 WITA
kategori: Besar, Frontend
file yang diubah:
- frontend/src/components/NotificationBell.tsx (baru)
- frontend/src/components/DashboardLayout.tsx
- frontend/changelog.md



## [1.24.2] - 2026-08-03

### Label Status "Gagal" Menjadi "Perlu Tindak Lanjut" (Hanya Tampilan)

perubahan yang dilakukan: **Label tampilan status `failed` diubah dari "Gagal" menjadi "Perlu Tindak Lanjut" di seluruh antarmuka — hanya teks yang berubah, nilai status di API/database tetap `failed` sehingga filter, URL query, dan logika aplikasi tidak berubah**
before: status bernilai `failed` tampil sebagai "Gagal" di badge tabel, filter dropdown, modal detail, kartu statistik dashboard, legenda analitik, dan tooltip grafik tren.
after:
- `src/app/maintenance/page.tsx`: `getStatusLabel` & opsi filter dropdown memakai "Perlu Tindak Lanjut" (badge tabel, modal catat/ubah, dan detail otomatis ikut karena memakai `getStatusLabel`).
- `src/app/dashboard/page.tsx`: map label status & kartu statistik "Perlu Tindak Lanjut".
- `src/components/DashboardAnalytics.tsx`: legenda "Perlu Tindak Lanjut".
- `src/components/TrendChart.tsx`: tooltip "Perlu Tindak Lanjut".
timestamp: 2026-08-03 WITA
kategori: Kecil, Frontend
file yang diubah:
- frontend/src/app/maintenance/page.tsx
- frontend/src/app/dashboard/page.tsx
- frontend/src/components/DashboardAnalytics.tsx
- frontend/src/components/TrendChart.tsx
- backend/handlers/report.go
- backend/changelog.md
- frontend/changelog.md
- docs/changelog.md

## [1.24.1] - 2026-08-03

### Bugfix: Header CSRF Mengambil Cookie Stale (Path /api) Setelah Login Ulang

perubahan yang dilakukan: **`getCookie()` di `src/lib/api.ts` kini mengambil kemunculan cookie TERAKHIR, bukan yang pertama — bila browser masih menyimpan `sigap_csrf` stale warisan versi lama di path `/api`, `document.cookie` mengurutkannya lebih dulu (RFC 6265, path terpanjang pertama) sehingga header `X-CSRF-Token` memakai nilai stale sementara server membandingkan dengan cookie aktif di "/" → mutasi ditolak 403**
before: `getCookie()` memakai `document.cookie.match(...)` yang mengembalikan kecocokan pertama — cookie stale di path lebih panjang menang.
after: regex dengan flag `g` mengambil nilai kecocokan terakhir — cookie aktif di path "/" selalu dipakai untuk header `X-CSRF-Token`, sinkron dengan `ExtractLastCookieValue` di backend.
timestamp: 2026-08-03 WITA
kategori: Bugfix, Frontend
file yang diubah:
- frontend/src/lib/api.ts
- frontend/changelog.md
- backend/changelog.md
- docs/changelog.md

## [1.24.0] - 2026-08-03

### Alur Penugasan Pemeliharaan: Assign Teknisi & Selesaikan Tugas via Checklist 9 Tahap

perubahan yang dilakukan: **Di halaman Pemeliharaan CCTV, sub-menu status "Dalam Proses", "Selesai", "Gagal" pada input status dihapus (status kini tampil read-only & dikelola sistem), log terjadwal dapat ditugaskan ke teknisi lewat dropdown, dan tugas muncul di dashboard teknisi dengan tombol "Selesaikan" membuka wizard checklist 9 tahap**
before:
- Form catat/ubah pemeliharaan menampilkan select status 4 opsi (Selesai/Terjadwal/Dalam proses/Gagal) + input "waktu pencatatan" yang diisi manual — status bisa diubah sesuka hati.
- Log terjadwal tidak punya mekanisme penugasan ke teknisi tertentu; teknisi hanya bisa melihat tugas via dashboard tanpa bisa mengerjakan/menyelesaikannya.
- Nama teknisi pada baris log memakai label "Teknisi Sistem" bila kosong, tidak mencerminkan belum ada penugasan.
after:
- **Halaman `src/app/maintenance/page.tsx`:**
  - Input status diganti tampilan read-only: badge status saat ini (untuk catatan baru selalu "Terjadwal") + keterangan "Status diperbarui otomatis oleh sistem"; field `completed_at` dihapus dari form; input hanya "Waktu terjadwal" (datetime-local, default waktu sekarang; bug lama yang hanya menyimpan tanggal saat edit diperbaiki — kini memakai konversi `toDatetimeLocalInput`).
  - Daftar teknisi (role "teknisi") diambil dari `GET /users?role=teknisi` (admin) untuk dropdown penugasan.
  - Tombol aksi baru **Tugaskan** (ikon UserPlus) pada baris log berstatus "Terjadwal" (admin): modal pilih teknisi → `PUT /maintenance-logs/:id/assign`; setelah ditugaskan status otomatis "Dalam proses" dan nama teknisi tampil pada baris.
  - Tombol aksi baru **Selesaikan** (ikon CheckCheck) pada baris log berstatus "Dalam proses" milik teknisi yang ditugaskan (atau admin): membuka wizard checklist 9 tahap dalam mode "complete".
  - Teknisi tidak lagi melihat tombol "Catat Pemeliharaan" dan "Ubah" — alur teknisi: menerima tugas (dashboard) → mengerjakan → selesaikan; admin tetap bisa membuat/mengubah/menghapus.
  - Baris & detail log menampilkan "Belum ditugaskan" saat `technician` nil.
- **Wizard checklist `src/components/FieldMaintenanceModal.tsx`:** properti baru `mode="complete"` + `maintenanceLogId` — judul "Selesaikan Tugas Pemeliharaan", submit mengirim `POST /maintenance-logs/:id/complete` (tanpa `cctv_id`), toast sukses disesuaikan; seluruh 9 item + foto bukti (min 1, maks 5) tetap sama dengan mode field.
- **Dashboard `src/app/dashboard/page.tsx`:** tipe tugas menambahkan `cctv_id`/`cctv_kode`; untuk teknisi, baris tugas berstatus "Dalam proses" menampilkan tombol **Selesaikan** yang membuka wizard yang sama; setelah disimpan dashboard di-refresh (status tugas berubah menjadi selesai/gagal otomatis).
timestamp: 2026-08-03 WITA
kategori: Feature, UI, Frontend
file yang diubah:
- frontend/src/app/maintenance/page.tsx
- frontend/src/components/FieldMaintenanceModal.tsx
- frontend/src/app/dashboard/page.tsx
- frontend/changelog.md
verifikasi:
- `npm run lint` PASS (0 errors, 46 warnings existing).
- `npm run build` PASS.
alasan teknis: Alur status dikelola sistem (dibuat → terjadwal → ditugaskan → dalam proses → selesai/gagal) sehingga formulir tidak perlu menawarkan pilihan status manual yang rawan salah; penugasan menghubungkan tugas ke teknisi spesifik sehingga dashboard teknisi berisi tugas yang benar-benar menjadi tanggung jawabnya; wizard checklist 9 tahap dipakai ulang dari modul pengecekan lapangan agar standar pelaporan seragam.

---

## [1.23.0] - 2026-08-03

### Pengecekan Lapangan Teknisi via Scan QR + Halaman Admin

perubahan yang dilakukan: **Setelah scan QR CCTV, teknisi bisa langsung mengisi checklist pengecekan lapangan (jaringan, listrik, lensa, dll) satu per satu lengkap dengan foto bukti di tahap akhir; admin melihat semua hasilnya di halaman baru**
before:
- Hasil scan QR hanya menampilkan informasi CCTV (nama, kode, lokasi, IP, dll) tanpa sarana mencatat hasil pengecekan spontan di lapangan.
- Catatan pemeliharaan hanya bisa dibuat lewat form di halaman Pemeliharaan CCTV (jadwal dari admin); tidak ada alur pengecekan langsung di lokasi.
- Admin tidak memiliki tampilan khusus untuk melihat hasil pengecekan lapangan teknisi.
after:
- Tombol **"Maintenance"** (ikon ClipboardCheck) ditambahkan di hasil scan QR CCTV; mengkliknya membuka wizard "Pengecekan Lapangan" (komponen baru `src/components/FieldMaintenanceModal.tsx`).
- Wizard memandu teknisi **satu item per satu**: 9 item checklist wajib (jaringan, listrik, pembersihan lensa, fisik kamera, kualitas gambar, perekaman, akses jarak jauh, kabel & konektor, lingkungan sekitar), masing-masing dengan pilihan "Normal/OK" atau "Bermasalah" + catatan (diwajibkan bila bermasalah); tombol Lanjut nonaktif sampai item dijawab; progress bar & nomor langkah.
- **Tahap akhir wajib foto bukti** (minimal 1, maksimal 5): unggah langsung ke `/api/upload` dengan pratinjau thumbnail & hapus per foto; ditutup ringkasan hasil checklist dan catatan umum sebelum menyelesaikan.
- Halaman baru **`/field-maintenance`** (menu sidebar admin "Pengecekan Lapangan", ikon ClipboardCheck): tabel CCTV, teknisi, tanggal, ringkasan x/y normal, status Normal / Perlu Tindak Lanjut, filter pencarian/status/rentang tanggal, pagination, modal detail berisi seluruh checklist + foto bukti, serta hapus catatan (dengan dialog konfirmasi kustom).
- Halaman dilindungi admin-only via `proxy.ts` (ditambahkan ke `ADMIN_PATHS` + matcher).
timestamp: 2026-08-03 WITA
kategori: Feature, UI, Frontend
file yang diubah:
- frontend/src/components/FieldMaintenanceModal.tsx (baru)
- frontend/src/app/field-maintenance/page.tsx (baru)
- frontend/src/app/cctv/page.tsx (tombol Maintenance di hasil scan + render modal)
- frontend/src/components/DashboardLayout.tsx (menu "Pengecekan Lapangan" admin)
- frontend/src/proxy.ts (guard admin)
- frontend/changelog.md
verifikasi:
- `npm run lint` PASS (0 errors, 46 warnings existing).
- `npm run build` PASS.
alasan teknis: Pengecekan di lapangan bersifat spontan (tidak selalu sesuai jadwal admin) sehingga membutuhkan alur terpisah dari MaintenanceLog; model checklist per-item memungkinkan admin melihat detail apa yang bermasalah, bukan sekadar status. Foto bukti diunggah melalui endpoint upload yang sudah ada (normalisasi JPEG + tanpa EXIF) dan ditampilkan memakai `AuthenticatedImage` (akses via JWT).

---

## [1.22.0] - 2026-08-03

### Notifikasi & Konfirmasi Kustom dengan Animasi

perubahan yang dilakukan: **Semua `alert()`/`confirm()` bawaan browser diganti sistem toast & dialog konfirmasi kustom beranimasi**
before:
- 24 titik memakai `alert()` (peringatan/pesan sukses) dan `confirm()` (konfirmasi hapus) bawaan browser yang polos, tidak konsisten dengan desain, dan tanpa animasi.
after:
- Komponen baru `src/components/Toast.tsx`: provider `ToastProvider` + hook `useToast()` dengan `toast.success()` / `toast.error()` / `toast.info()`. Toast tampil di kanan atas: kartu dengan ikon lingkaran, **animasi check hijau (stroke-draw SVG)** untuk sukses, ikon X merah untuk error, info biru; slide-in elastis (`toastIn`), pop ikon (`checkPop`), progress bar yang menyusut, auto-dismiss 4,2 detik, maksimal 4 toast.
- Komponen baru `src/components/ConfirmDialog.tsx`: provider `ConfirmProvider` + hook `useConfirm()` (Promise<boolean>). Dialog kustom dengan ikon peringatan beranimasi goyang halus, fade + pop in/out, tombol Batal/Konfirmasi (merah untuk aksi destruktif), tutup via X/Escape/klik latar.
- Kedua provider dipasang di `src/app/layout.tsx` (z-index di atas modal lain: toast z-80, dialog z-70).
- Semua halaman diganti: users (simpan, reset password, hapus), maintenance (simpan, ekspor CSV, laporan PDF, hapus), cctv (simpan, hapus, bulk delete, cetak QR, template, impor). Toast sukses ditambahkan di semua aksi yang berhasil (sebelumnya beberapa hanya menutup modal tanpa umpan balik).
- Animasi baru ditambahkan di `globals.css`: `toastIn`, `toastProgress`, `checkPop`, `checkCircle`, `checkPath`, `fadeIn/Out`, `popIn/Out`, `wiggleSlow`.
timestamp: 2026-08-03 WITA
kategori: Feature, UI, Frontend
file yang diubah:
- frontend/src/components/Toast.tsx (baru)
- frontend/src/components/ConfirmDialog.tsx (baru)
- frontend/src/app/layout.tsx
- frontend/src/app/globals.css
- frontend/src/app/users/page.tsx
- frontend/src/app/maintenance/page.tsx
- frontend/src/app/cctv/page.tsx
- frontend/changelog.md
verifikasi:
- `npm run lint` PASS (0 errors, 38 warnings existing).
- `npm run build` PASS.
alasan teknis: Menghilangkan seluruh UI native browser demi konsistensi bahasa desain, aksesibilitas (role/aria), dan umpan balik visual yang jelas (check animasi untuk sukses) serta mencegah logout/hapus tidak sengaja via dialog konfirmasi yang dapat dikustomisasi.

---

## [1.21.0] - 2026-08-03

### Redesain Filter Audit Log (Input Tidak Lagi Terpotong)

perubahan yang dilakukan: **Tata letak filter & search bar audit log dirombak agar setiap input mendapat ruang cukup**
before:
- Grid filter memakai 4 kolom (`lg:grid-cols-4`); dua input tanggal + teks "s.d." dipaksakan masuk dalam satu sel kolom sehingga input terpotong/tertekan di layar laptop.
after:
- Grid diubah menjadi `lg:grid-cols-3`: baris pencarian full-width di atas, lalu kolom "Modul", "Dari tanggal", dan "Sampai tanggal" masing-masing satu kolom penuh dengan label (`<label>` + `htmlFor`) sehingga jelas dan lega.
- Teks "s.d." di tengah input tanggal digantikan label "Dari tanggal"/"Sampai tanggal".
timestamp: 2026-08-03 WITA
kategori: UI, Frontend
file yang diubah:
- frontend/src/app/audit-logs/page.tsx
- frontend/changelog.md
verifikasi:
- `npm run lint` PASS (0 errors, 38 warnings existing).
- `npm run build` PASS.
alasan teknis: Dua input `type="date"` (masing-masing butuh ~150px) tidak muat dalam satu sel grid 4 kolom (~230px); memberi satu kolom penuh per input dan label di atasnya menghilangkan pemotongan dan memperjelas fungsi tiap field.

---

## [1.20.0] - 2026-08-03

### Perbaikan Cetak Semua QR Code Hanya 10 CCTV

perubahan yang dilakukan: **"Cetak Semua QR Code" kini mengambil seluruh data CCTV dari backend, bukan hanya halaman aktif**
before:
- Tombol "Cetak Semua QR Code" (menu aksi di halaman Aset CCTV) memakai state `cctvs` yang hanya berisi data halaman aktif (10 baris per halaman), sehingga hanya 10 label QR yang tercetak padahal total perangkat 35.
after:
- Ketika dipanggil tanpa ID target, halaman meminta seluruh data dari backend dengan `GET /cctv?page=1&limit=100` (batas maksimum backend); hasilnya difilter dan digunakan sebagai antrian cetak penuh.
- Fallback ke data halaman aktif hanya jika permintaan gagal.
timestamp: 2026-08-03 WITA
kategori: Bugfix, Frontend
file yang diubah:
- frontend/src/app/cctv/page.tsx
- frontend/changelog.md
verifikasi:
- `npm run lint` PASS (0 errors, 38 warnings existing).
- `npm run build` PASS.
alasan teknis: State `cctvs` dibatasi paginasi (10 per halaman); cetak semua memerlukan dataset lengkap sehingga diambil ulang dari API dengan limit maksimum yang diizinkan server (100).

---

## [1.19.0] - 2026-08-03

### Konfirmasi Logout dengan Dialog Kustom

perubahan yang dilakukan: **Klik "Keluar" kini membuka dialog konfirmasi kustom (bukan logout langsung)**
before:
- Item menu "Keluar" di sidebar langsung menjalankan logout tanpa konfirmasi; risiko keluar tidak sengaja.
after:
- Klik "Keluar" membuka dialog konfirmasi kustom (modal) dengan: ikon `LogOut` dalam lingkaran merah, judul "Keluar dari SIGAP?", deskripsi penjelas, tombol "Batal" (netral) dan "Keluar" (merah, autoFocus).
- Modal dapat ditutup via tombol X, klik area latar, atau tekan `Escape`; dialog memakai pola `role="dialog"` + `aria-modal` untuk aksesibilitas.
- Logout hanya dieksekusi setelah konfirmasi; drawer mobile ikut ditutup saat logout disetujui.
timestamp: 2026-08-03 WITA
kategori: Feature, UI, Frontend
file yang diubah:
- frontend/src/components/DashboardLayout.tsx
- frontend/changelog.md
verifikasi:
- `npm run lint` PASS (0 errors, 38 warnings existing).
- `npm run build` PASS.
alasan teknis: Konfirmasi dialog kustom mencegah logout tidak disengaja dan konsisten dengan bahasa desain aplikasi (modal, backdrop blur, tombol danger merah).

---

## [1.18.0] - 2026-08-03

### Card Tugas Menggantikan Distribusi Status CCTV

perubahan yang dilakukan: **Card "Distribusi Status CCTV" dihapus, card tugas naik ke posisinya di samping "Tren Pemeliharaan"**
before:
- Dashboard menampilkan card "Distribusi Status CCTV" (bar persentase Aktif/Nonaktif/Pemeliharaan) di kolom kanan grid analitik, sedangkan card "Tugas Berjalan Semua Teknisi"/"Tugas Saya" berada sendiri di bawah grid.
after:
- Card "Distribusi Status CCTV" dihapus total dari `DashboardAnalytics`.
- Card tugas kini dirender sebagai slot `children` di dalam grid analitik, menggantikan posisi card distribusi status — berdampingan dengan "Tren Pemeliharaan"; card "Peringatan Pemeliharaan Terlambat" tetap full-width di bawahnya.
- Interface `DashboardSummary` dirapikan (field `cctv_status` dan helper `cctvStatusLabel` yang tidak terpakai dihapus).
timestamp: 2026-08-03 WITA
kategori: UI, Frontend
file yang diubah:
- frontend/src/components/DashboardAnalytics.tsx
- frontend/src/app/dashboard/page.tsx
- frontend/changelog.md
verifikasi:
- `npm run lint` PASS (0 errors, 38 warnings existing).
- `npm run build` PASS.
alasan teknis: Mengurangi redundansi (distribusi status sudah terwakili ringkasan di atas), dan menaikkan card tugas yang lebih relevan untuk pemantauan pekerjaan ke area utama dashboard.

---

## [1.17.0] - 2026-08-03

### Logout Hanya di Sidebar & Transisi Drawer Lebih Halus

perubahan yang dilakukan: **Tombol logout di header mobile dihapus + animasi buka/tutup sidebar diperhalus**
before:
- Header mobile (dekat tombol hamburger) masih memiliki tombol logout ikon `LogOut` merah, sehingga ada dua jalur logout (header + item menu sidebar).
- Drawer sidebar membuka/menutup dengan transisi 300ms linear dan overlay hanya muncul setelah kondisi berubah (tanpa animasi keluar).
after:
- Tombol logout di header mobile dihapus total; logout hanya tersedia melalui item menu "Keluar" di dalam sidebar.
- Transisi sidebar memakai `duration-500 ease-in-out` untuk sliding yang lebih halus; overlay kini selalu ter-mount (`md:hidden`) dengan fade `transition-opacity duration-500` + `opacity-0`/`pointer-events-none` saat tertutup, sehingga efek buka dan tutup sama-sama beranimasi.
timestamp: 2026-08-03 WITA
kategori: UI, Frontend
file yang diubah:
- frontend/src/components/DashboardLayout.tsx
- frontend/changelog.md
verifikasi:
- `npm run lint` PASS (0 errors, 38 warnings existing).
- `npm run build` PASS.
alasan teknis: Menghindari duplikasi aksi logout yang membingungkan dan memperhalus gerakan drawer dengan easing yang lebih natural serta fade overlay dua arah.

---

## [1.16.0] - 2026-08-03

### Tombol Keluar Sebagai Item Menu di Sidebar

perubahan yang dilakukan: **Tombol logout dipindah ke dalam daftar navigasi, di bawah menu Audit Log, dengan gaya yang sama seperti item menu lain**
before:
- Tombol logout terpisah di bagian bawah sidebar (di bawah garis pembatas `border-t`), dengan gaya khusus merah dan label "Keluar dari SIGAP".
after:
- Tombol logout kini menjadi item terakhir di daftar navigasi sidebar — tepat di bawah "Audit Log" untuk admin (dan di bawah "Pemeliharaan CCTV" untuk teknisi) — dengan class yang identik dengan item menu lain (`min-h-11`, `rounded-xl`, `text-muted hover:bg-background hover:text-foreground`), ikon `LogOut`, dan label singkat "Keluar".
- Item "Keluar" menutup drawer mobile saat diklik dan tetap kompatibel dengan mode sidebar diciutkan (ikon saja).
timestamp: 2026-08-03 WITA
kategori: UI, Frontend
file yang diubah:
- frontend/src/components/DashboardLayout.tsx
- frontend/changelog.md
verifikasi:
- `npm run lint` PASS (0 errors, 38 warnings existing).
- `npm run build` PASS.
alasan teknis: Meletakkan logout sebagai bagian dari menu navigasi membuatnya konsisten dengan pola item menu lain dan selalu terlihat tanpa menggantung di bagian bawah, sekaligus tetap tersedia di drawer mobile.

---

## [1.15.0] - 2026-08-03

### Penyesuaian Antarmuka untuk Peran Teknisi

perubahan yang dilakukan: **Tombol keluar di mobile, tombol PDF/CSV jadi ikon, payload QR disembunyikan untuk teknisi, dan ringkasan tugas teknisi**
before:
- Di layar kecil (HP) tidak ada tombol keluar pada sidebar; logout hanya tersedia di sidebar
  desktop, dan pada perangkat pendek tombol logout bisa tergeser keluar layar karena nav tidak bisa digulir.
- Tombol "Laporan PDF" dan "Ekspor CSV" di halaman Pemeliharaan (dan Audit Log) berupa tombol teks lebar.
- Modal Pindai QR menampilkan tombol "Masukkan payload" (input manual payload terenkripsi) untuk semua peran, termasuk teknisi.
- Card "Tugas Saya" teknisi menampilkan teks berlebihan: lokasi CCTV, tindakan (action_taken), dan baris "Jadwal: ...".
after:
- Header mobile (di samping toggle tema) kini memiliki tombol keluar berikon `LogOut` berwarna merah; sidebar mendapat `flex-1 min-h-0 overflow-y-auto` sehingga tombol logout tetap terjangkau pada layar pendek.
- Tombol Laporan PDF kini ikon `FileText` merah dan Ekspor CSV ikon `FileSpreadsheet` hijau (konsisten di halaman Pemeliharaan dan Audit Log), keduanya kotak ikon-only dengan `title`/`aria-label`.
- Toggle "Masukkan payload" dan view input manual hanya dirender untuk admin; teknisi hanya melihat pemindai kamera (`!isAdmin || !showManualInput`).
- "Tugas Saya" teknisi dirampingkan: hanya nama CCTV + badge jadwal (mono) + badge status + badge terlambat; lokasi dan action_taken tetap tampil untuk admin. Subtitle diperpendek.
timestamp: 2026-08-03 WITA
kategori: Feature, UI, Frontend
file yang diubah:
- frontend/src/components/DashboardLayout.tsx
- frontend/src/app/maintenance/page.tsx
- frontend/src/app/audit-logs/page.tsx
- frontend/src/app/cctv/page.tsx
- frontend/src/app/dashboard/page.tsx
- frontend/changelog.md
verifikasi:
- `npm run lint` PASS (0 errors, 38 warnings existing).
- `npm run build` PASS.
alasan teknis: Akses payload QR terenkripsi dibatasi ke admin demi prinsip least privilege; tombol ikon-only menghemat ruang header pada layar kecil; sidebar scrollable menjamin aksi logout selalu tersedia; pengurangan teks menurunkan beban kognitif peran teknisi di lapangan.

---

## [1.14.0] - 2026-08-03

### Waktu Pencatatan dengan Picker Kalender

perubahan yang dilakukan: **Input teks diganti `datetime-local` + perbaikan zona waktu default**
before:
- Field "Waktu pencatatan"/"Waktu terjadwal" berupa input teks bebas dengan placeholder
  `YYYY-MM-DD HH:MM:SS`; tidak ada kalender, mudah salah format.
- Nilai default memakai `new Date().toISOString()` yang menghasilkan waktu **UTC** (bukan waktu
  lokal Indonesia), sehingga tampilan bisa meleset beberapa jam.
after:
- Field memakai `type="datetime-local"`: kalender bawaan browser untuk memilih hari + picker jam,
  nilai default otomatis waktu saat ini (format disimpan tetap `YYYY-MM-DD HH:MM:SS`).
- Helper baru: `nowLocalDateTime()` (waktu lokal WITA/browser), `toDatetimeLocalInput()`, dan
  `fromDatetimeLocalInput()` untuk konversi format dua arah.
- Teks bantu "Terisi otomatis waktu saat ini — buka kalender untuk memilih hari."
timestamp: 2026-08-03 WITA
kategori: Feature, Bugfix, Frontend, UI
file yang diubah:
- frontend/src/app/maintenance/page.tsx
- frontend/changelog.md
verifikasi:
- `npm run lint` PASS dengan warning existing.
- `npm run build` PASS.
alasan teknis: Input native datetime-local menghilangkan kesalahan format manual dan menjaga
konsistensi nilai yang dikirim ke backend (`YYYY-MM-DD HH:MM:SS`), sekaligus memperbaiki default
waktu yang sebelumnya berada di zona UTC.

---

## [1.13.0] - 2026-08-03

### Tren Pemeliharaan Menjadi Spline Area Chart

perubahan yang dilakukan: **Bar chart CSS diganti grafik area melengkung halus dengan gradasi biru**
before:
- Card "Tren Pemeliharaan" menampilkan bar chart bertumpuk per status (CSS div) dengan tinggi
  proporsional dan tooltip native.
after:
- Komponen baru `TrendChart` (`frontend/src/components/TrendChart.tsx`): Spline Area Chart berbasis
  SVG murni tanpa dependency baru.
- Garis tren total pekerjaan per bulan digambar melengkung halus (Catmull-Rom spline → cubic
  Bézier), area di bawah garis diisi gradasi biru transparan.
- Responsif via `ResizeObserver`, grid sumbu Y, label bulan, titik data, penanda hover dengan
  garis putus-putus, dan tooltip rincian per status (Terjadwal/Dalam proses/Selesai/Gagal).
- Legend bar chart per status diganti legend "Total pekerjaan" + petunjuk hover.
- Deskripsi card disesuaikan menjadi "Total pekerjaan per bulan — 6 bulan terakhir."
timestamp: 2026-08-03 WITA
kategori: Feature, Frontend, UI
file yang diubah:
- frontend/src/components/TrendChart.tsx (baru)
- frontend/src/components/DashboardAnalytics.tsx
- frontend/changelog.md
verifikasi:
- `npm run lint` PASS dengan warning existing.
- `npm run build` PASS.
alasan teknis: Visualisasi tren berbentuk area dengan garis halus lebih mudah dibaca untuk melihat
pergerakan total pekerjaan antar bulan; implementasi SVG mandiri menghindari dependency chart
eksternal yang menambah ukuran bundle dan risiko keamanan.

---

## [1.12.0] - 2026-08-03

### Posisi Rincian Ringkasan CCTV & Warna Bar Distribusi Status

perubahan yang dilakukan: **Detail respon kartu ringkasan dipindah ke bawah kartu; warna bar status dibuat cerah**
before:
- Klik kartu Total CCTV/Aktif/Nonaktif menampilkan rincian (`cctv-summary-details`) di bagian
  paling bawah halaman — setelah analitik, tugas, dan pemeliharaan — sehingga respon jauh dari
  kartu yang diklik.
- Bar progress "Distribusi Status CCTV" memakai warna tema yang redup (`bg-success`/`bg-error`).
after:
- Section rincian kini dirender tepat setelah kartu ringkasan CCTV, sebelum
  `DashboardAnalytics`, sehingga respon tampil langsung di bawah kartu yang diklik.
- Bar progress aktif menjadi hijau terang (`bg-emerald-500`), nonaktif/rusak merah terang
  (`bg-red-500`), pemeliharaan kuning cerah (`bg-amber-500`) — konsisten dengan marker peta.
- Pencocokan status dibuat case-insensitive: backend mengirim `ACTIVE`/`INACTIVE` (kapital),
  sedangkan pengecekan sebelumnya memakai `aktif` (huruf kecil) sehingga bar selalu jatuh ke
  warna default. Label status juga dinormalisasi menjadi "Aktif"/"Nonaktif"/"Pemeliharaan".
timestamp: 2026-08-03 WITA
kategori: Feature, Frontend, UI
file yang diubah:
- frontend/src/app/dashboard/page.tsx
- frontend/src/components/DashboardAnalytics.tsx
- frontend/changelog.md
verifikasi:
- `npm run lint` PASS dengan warning existing.
- `npm run build` PASS.
alasan teknis: Memperpendek jarak umpan balik (feedback loop) saat pengguna mengeksplorasi
ringkasan CCTV, serta meningkatkan keterbacaan proporsi status pada bar chart.

---

## [1.11.0] - 2026-08-03

### Mode Layar Penuh untuk Peta Sebaran CCTV

perubahan yang dilakukan: **Tombol fullscreen pada peta sebaran CCTV menggunakan Fullscreen API**
before:
- Peta sebaran CCTV hanya tampil di dalam kartu (tinggi 460px) tanpa opsi memperbesar tampilan.
after:
- Tombol fullscreen (ikon Maximize/Minimize) ditambahkan di pojok kanan atas peta.
- Mode layar penuh memakai Fullscreen API browser: peta mengisi seluruh layar (100vw x 100vh),
  border/radius dihilangkan, dan dapat ditutup lewat tombol atau tombol Esc.
- Ikon tombol ikut berubah mengikuti status fullscreen via event `fullscreenchange`.
- Import ikon `Camera` yang tidak terpakai dihapus.
timestamp: 2026-08-03 WITA
kategori: Feature, Frontend, UI
file yang diubah:
- frontend/src/components/CCTVMap.tsx
- frontend/changelog.md
verifikasi:
- `npm run lint` PASS dengan warning existing.
- `npm run build` PASS.
alasan teknis: Memberikan fleksibilitas memeriksa sebaran CCTV pada layar besar tanpa mengubah
tampilan peta di dalam halaman.

---

## [1.10.0] - 2026-08-03

### Marker CCTV Bertumpuk pada Peta Sebaran

perubahan yang dilakukan: **Marker dengan koordinat identik digeser agar tidak saling menutupi**
before:
- CCTV dengan koordinat latitude/longitude yang sama (contoh: THD-05 "Bundaran Sumbu Barat" dan
  THD-17 "Pintu Masuk Sumbu Barat" keduanya di -0.977067574, 116.7091789) dirender di titik yang
  persis sama, sehingga satu marker menutupi marker lainnya dan jumlah pin di peta lebih sedikit
  dari jumlah CCTV sebenarnya (20 aktif di info, tapi hanya 19 pin terlihat).
after:
- `CCTVMap` menghitung kemunculan tiap pasangan koordinat dan menggeser marker ke-2, ke-3, dst.
  secara deterministik (jarak 0.005° dengan sudut 137.5° per kemunculan) sehingga semua marker
  tetap terlihat meskipun datanya berbagi koordinat yang sama.
timestamp: 2026-08-03 WITA
kategori: Bugfix, Frontend, UI
file yang diubah:
- frontend/src/components/CCTVMap.tsx
- frontend/changelog.md
verifikasi:
- `npm run lint` PASS dengan warning existing.
- `npm run build` PASS.
- Verifikasi data: `SELECT status, COUNT(*) FROM cctvs GROUP BY status` → 20 ACTIVE, 15 INACTIVE
  (kartu info akurat; selisih hanya efek tumpukan marker).
alasan teknis: Memastikan visualisasi peta menampilkan seluruh CCTV tanpa kehilangan marker akibat
duplikasi koordinat pada data aset, tanpa mengubah data di database.

---

## [1.9.0] - 2026-08-03

### Pemisahan Peta Sebaran CCTV dari Dashboard

perubahan yang dilakukan: **Peta GIS sebaran CCTV dipindah dari Dashboard ke halaman & menu baru**
before:
- Halaman `/dashboard` merender `CCTVMap` (peta GIS sebaran CCTV) di antara kartu ringkasan dan
  analitik; navigasi sidebar hanya memiliki Dashboard, Aset CCTV, Pemeliharaan CCTV, dan menu admin.
after:
- Peta sebaran CCTV dihapus dari `/dashboard`; dashboard menjadi lebih fokus pada ringkasan,
  analitik, tugas, dan pemeliharaan.
- Halaman baru `/sebaran-cctv` dibuat: memuat data CCTV (`GET /cctv` limit 1000), menampilkan
  `CCTVMap` dengan tinggi peta 560px, kartu ringkasan status (Total/Aktif/Pemeliharaan), indikator
  kesehatan layanan, tombol muat ulang, dan refresh otomatis via event WebSocket `CCTV_UPDATED`.
- Menu baru "Peta Sebaran CCTV" (ikon `MapPinned`) ditambahkan di bawah "Dashboard" pada sidebar
  `DashboardLayout`, tersedia untuk semua peran.
timestamp: 2026-08-03 WITA
kategori: Feature, Frontend, UI
file yang diubah:
- frontend/src/app/sebaran-cctv/page.tsx (baru)
- frontend/src/app/dashboard/page.tsx
- frontend/src/components/DashboardLayout.tsx
- frontend/changelog.md
verifikasi:
- `npm run lint` PASS dengan warning existing.
- `npm run build` PASS.
alasan teknis: Memisahkan visualisasi geospasial dari halaman ringkasan agar Dashboard tetap ringan
dan fokus pada pemantauan data operasional, sekaligus memberikan akses cepat ke peta sebaran CCTV
melalui menu navigasi utama.

---

## [1.8.0] - 2026-08-03

### Navigasi Auth Atomik

perubahan yang dilakukan: **Login dan logout memakai full navigation untuk membatalkan request/state lama**
before:
- `router.push()` mempertahankan root layout dan request dashboard lama setelah logout.
- Request lama yang selesai dengan 401 dapat memicu redirect global setelah sesi baru berhasil login.
after:
- `window.location.replace()` digunakan setelah login/logout.
- AuthProvider, WebSocket, dan request halaman dimulai ulang secara bersih dengan cookie sesi terbaru.
timestamp: 2026-08-03 WITA
kategori: Bugfix, Auth, Frontend
file yang diubah:
- frontend/src/context/AuthContext.tsx
- frontend/changelog.md
verifikasi:
- `npm run lint` PASS dengan warning existing.
- `npm run build` PASS.
alasan teknis: Mencegah race condition request async lama yang menyebabkan login berhasil tetapi
langsung diarahkan kembali ke login.

---

## [1.7.0] - 2026-08-03

### Hentikan Reconnect WebSocket Setelah Logout

perubahan yang dilakukan: **Timer reconnect WebSocket sekarang mengikuti status autentikasi**
before:
- `onclose` selalu menjadwalkan koneksi ulang setiap 5 detik, termasuk setelah logout.
- Browser terus mengirim `/api/ws` dengan status 401 setelah sesi dihapus.
after:
- Reconnect hanya dijadwalkan ketika sesi masih aktif.
- Cleanup logout membatalkan timer dan menutup socket tanpa membuat koneksi baru.
timestamp: 2026-08-03 WITA
kategori: Bugfix, Auth, Realtime
file yang diubah:
- frontend/src/context/WebSocketContext.tsx
- frontend/changelog.md
verifikasi:
- `npm run lint` PASS dengan warning existing.
- `npm run build` PASS.
alasan teknis: Menghentikan request WebSocket berulang setelah logout agar tidak mengganggu
rate limiter dan tidak menimbulkan kesan sesi masih aktif.

---

## [1.6.0] - 2026-08-03

### Stabilitas Alur Logout

perubahan yang dilakukan: **Request logout tidak lagi memicu redirect 401 global**
before:
- `apiRequest('/auth/logout')` menggunakan penanganan 401 global yang tidak diperlukan untuk
  request logout dan dapat menjalankan redirect/event auth-expired tambahan.
after:
- Request logout memakai `skipAuthRedirect: true`; frontend tetap membersihkan state lokal dan
  mengarahkan ke login setelah endpoint logout idempotent dipanggil.
timestamp: 2026-08-03 WITA
kategori: Bugfix, Auth
file yang diubah:
- frontend/src/context/AuthContext.tsx
- frontend/changelog.md
verifikasi:
- `npm run lint` PASS dengan warning existing.
- `npm run build` PASS.
alasan teknis: Menjaga alur logout tetap deterministik ketika sesi server sudah kedaluwarsa atau
cookie lama tidak lagi valid.

---

## [1.5.0] - 2026-08-03

### Penghapusan Sisa Idle Logout

perubahan yang dilakukan: **Menghapus state banner inaktivitas dan mekanisme `sigap_logout_reason` yang sudah tidak digunakan**
before:
- Halaman login masih membaca `sigap_logout_reason` dan menampilkan pesan logout otomatis setelah 15 menit.
- `AuthContext.logout` masih menerima parameter alasan meskipun idle timer sudah dihapus.
after:
- State, effect, banner, ikon, dan localStorage key idle logout dihapus.
- `logout()` hanya menjalankan alur logout manual/server-side yang masih aktif.
timestamp: 2026-08-03 WITA
kategori: Bugfix, Refactor, Auth
file yang diubah:
- frontend/src/app/login/page.tsx
- frontend/src/context/AuthContext.tsx
- frontend/changelog.md
alasan teknis: Menghapus dead code dan mencegah UI menampilkan status sesi yang tidak lagi dapat dipicu.
verifikasi:
- `npm ci` berhasil memasang 370 package sesuai `package-lock.json`.
- `npm run lint` PASS dengan warning existing.
- `npm run build` PASS.
- `npm audit` melaporkan 4 vulnerability high yang belum diperbaiki.

---

## [1.4.0] - 2026-08-02

### Hapus Cookie CSRF Saat Logout

perubahan yang dilakukan: **Memanggil clearAuthCookies setelah proses logout**
before:
- Logout hanya mengandalkan backend menghapus cookie via Set-Cookie; cookie CSRF (non-httpOnly) dapat tertinggal di klien.
after:
- Setelah permintaan logout, frontend juga memanggil `clearAuthCookies` untuk membersihkan cookie CSRF di sisi klien (cookie sesi httpOnly tetap hanya dapat dihapus server).
timestamp: 2026-08-02 WITA
kategori: Bugfix
file yang diubah:
- frontend/src/context/AuthContext.tsx
- frontend/changelog.md
alasan teknis: Memastikan state autentikasi benar-benar bersih setelah logout dan mencegah cookie CSRF sisa mengganggu login berikutnya.

## [1.3.0] - 2026-08-02

### Hapus Auto-Logout Idle

perubahan yang dilakukan: **Menghapus timer auto-logout 15 menit dan modal peringatan inaktivitas**
before:
- Pengguna otomatis keluar setelah 15 menit tanpa aktivitas (idle timer) dan melihat modal peringatan "Sesi Hampir Berakhir".
- Sesi dapat terputus mendadak meski token masih valid.
after:
- Auto-logout idle dihapus total: pengguna tetap login sampai menekan logout atau token kedaluwarsa (5 jam).
- `AuthContext` tidak lagi memakai idle timer, listener aktivitas, `extendSession`, maupun modal peringatan inaktivitas.
- Logout manual tetap merevoke token & mengarahkan ke halaman login; sesi mati saat browser ditutup (session cookie).
timestamp: 2026-08-02 WITA
kategori: UI/UX, Refactor
file yang diubah:
- frontend/src/context/AuthContext.tsx
- frontend/changelog.md
alasan teknis: Menghilangkan logout mendadak yang mengganggu; model sesi disederhanakan sesuai permintaan pengguna (login → token, logout → langsung mati).

## [1.2.0] - 2026-08-02

### Redesign Halaman Login Split-Screen

perubahan yang dilakukan: **Mengganti layout login glassmorphism menjadi split-screen 50:50 dengan foto malam IKN dan panel krem**
before:
- Halaman login memakai kartu sentral glassmorphism di atas latar abstrak (blur elemen dekoratif) tanpa foto kawasan.
- Branding hanya berupa teks "SIGAP" tanpa logo Pohon Hayat NUSANTARA.
after:
- Layout responsif split 50:50: kiri adalah foto full-bleed malam Kompleks Kemenko I (Garuda Palace) IKN dari `public/assets/foto/ikn-kemenko.webp` (mobile: banner atas, desktop: setengah layar penuh); kanan adalah panel krem lembut.
- Panel kanan menampilkan logo Pohon Hayat NUSANTARA (`public/nusantara-logo.png`) dengan teks "NUSANTARA Ibu Kota Negara Indonesia", judul "SIGAP" font serif hijau tua, dan deskripsi "Sistem internal pengelolaan CCTV kawasan IKN."
- Form login dipertahankan lengkap: label & input "Username atau email" (ikon user) dan "Kata sandi" (ikon gembok + toggle mata), tombol "Masuk" hijau tua dengan teks putih.
- Ikon status (lonceng + toggle tema matahari/bulan) di pojok kanan atas panel krem.
- Seluruh warna memakai token tema (`bg-background`, `text-foreground`, `bg-primary`, `border-input-border`, dsb.) sehingga toggle mode gelap/terang berfungsi: mode terang = krem + hijau tua, mode gelap = hijau tua + krem.
timestamp: 2026-08-02 WITA
kategori: UI/UX, Feature
file yang diubah:
- frontend/src/app/login/page.tsx
- frontend/changelog.md
alasan teknis: Meningkatkan identitas visual kawasan IKN pada halaman masuk serta menghadirkan tipografi & palet hijau tua-krem yang modern dan tegas.

## [1.1.0] - 2026-08-02

### Halaman 404 Kustom dengan Animasi Lottie

perubahan yang dilakukan: **Menambahkan halaman `not-found` kustom yang dirender untuk seluruh URL yang tidak cocok**
before:
- URL yang tidak dikenal menampilkan halaman 404 bawaan Next.js tanpa identitas SIGAP.
after:
- Halaman `/not-found` baru menampilkan animasi Lottie T-rex "PAGE NOT FOUND" berukuran besar (hingga 560px) dari `public/assets/animasi/404 Error.json` di atas latar putih polos.
- Animasi dimuat saat runtime via fetch (aset statis publik) melalui komponen client `Lottie404`; ada placeholder netral saat memuat agar layout tidak melompat.
- Desain minimalis: background putih murni tanpa elemen warna, heading, tombol, maupun footer — hanya animasi yang tampil.
timestamp: 2026-08-02 WITA
kategori: UI/UX, Feature
file yang diubah:
- frontend/src/app/not-found.tsx (baru)
- frontend/src/components/Lottie404.tsx (baru)
- frontend/package.json (tambah dependency `lottie-react`)
- frontend/changelog.md
alasan teknis: Memberikan pengalaman 404 yang konsisten dengan identitas produk SIGAP dan memanfaatkan aset animasi yang telah tersedia.

## [1.0.0] - 2026-08-02

### Server-side Route Guard dan Audit Log

perubahan yang dilakukan: **Middleware route guard dan pencegahan request audit yang tidak perlu**
before:
- Akses `/audit-logs` dan `/users` hanya mengandalkan pemeriksaan role di client.
- Halaman audit dapat memulai request sebelum status user admin diketahui.
after:
- Proxy Next.js memeriksa cookie sesi dan role admin melalui `/api/auth/me` sebelum route admin diproses.
- Halaman audit hanya mengambil data setelah user terautentikasi sebagai admin.
timestamp: 2026-08-02 00:00 WITA
kategori: Security, Quality
file yang diubah:
- frontend/src/proxy.ts
- frontend/src/app/audit-logs/page.tsx
- frontend/eslint.config.mjs
alasan teknis: Mengurangi ketergantungan pada guard client-side dan menghindari request API yang tidak perlu dari role teknisi.

## [0.9.0] - 2026-08-01

### Fitur Kapabilitas Fase 3 (ROADMAP 3.1 - 3.9)

perubahan yang dilakukan: **Dashboard analitik, notifikasi tugas teknisi, halaman profil, filter rentang tanggal, ekspor CSV/PDF**
before:
- Belum ada dashboard analitik; halaman utama setelah login belum menampilkan statistik/tren/peringatan.
- Teknisi tidak mendapat ringkasan tugas; belum ada halaman profil.
- List & ekspor maintenance/audit-log tanpa filter rentang tanggal.
- Belum ada tombol unduh laporan PDF maupun ekspor audit-log.
after:
- **Dashboard analitik (3.3):** halaman `/dashboard` menampilkan statistik (total CCTV, log, selesai, terlambat), tren pemeliharaan, distribusi status, dan daftar CCTV overdue via `DashboardStats`/`DashboardAnalytics`.
- **Notifikasi tugas (3.4):** section "Tugas Anda" di dashboard dengan badge tugas aktif/terlambat, memakai `GET /dashboard/tasks`.
- **Halaman profil (3.5):** `/profile` untuk mengubah nama tampilan (`PUT /auth/profile`); kartu profil di sidebar menjadi link.
- **Unifikasi CCTV (3.6):** seluruh tampilan maintenance memakai `cctv_id`/`cctv` (nama, kode, lokasi) menggantikan `asset`; filter CCTV pada form & list.
- **Filter rentang tanggal & ekspor (3.7):** input `start_date`/`end_date` pada halaman maintenance & audit-logs; tombol "Laporan PDF" dan "Ekspor CSV" (via `downloadApiFile`).
- **Upload foto (3.8):** server kini menormalkan semua upload ke JPEG dan menghapus metadata GPS; UX tampilan lampiran tetap memakai `AuthenticatedImage`.
- **Ekspor per role (3.9):** tombol ekspor/laporan pada maintenance mengikuti scope data backend sesuai role (teknisi = data miliknya).
timestamp: 2026-08-01 14:30 WITA
kategori: Feature, Dashboard, UI/UX, PDF, Export
file yang diubah:
- frontend/src/app/dashboard/page.tsx (3.3, 3.4)
- frontend/src/components/DashboardAnalytics.tsx (3.3)
- frontend/src/components/DashboardStats.tsx (3.3)
- frontend/src/app/profile/page.tsx (baru, 3.5)
- frontend/src/context/AuthContext.tsx (3.5)
- frontend/src/components/DashboardLayout.tsx (3.5)
- frontend/src/app/maintenance/page.tsx (3.6, 3.7, 3.8)
- frontend/src/app/audit-logs/page.tsx (3.7)
- frontend/changelog.md
alasan perubahan: Menyelesaikan ROADMAP Fase 3 (fitur kapabilitas) dan menyesuaikan tampilan dengan unifikasi model CCTV.

---

## [0.8.0] - 2026-08-01

### Hardening Keamanan Fase 2 (Security Audit & Roadmap)

perubahan yang dilakukan: **Migrasi Auth ke Cookie httpOnly + CSRF, Pengetatan Kebijakan Password**
before:
- Token JWT disimpan di `localStorage` (`sigap_token`/`sigap_user`) dan dikirim via header `Authorization` sehingga rentan terhadap XSS token theft.
- Mutasi terautentikasi (POST/PUT/DELETE) tidak mengirimkan token CSRF, dan beberapa fetch mentah (QR, import, download template) memakai `localStorage` token secara langsung.
- Form kata sandi (buat/reset user) hanya membatasi `minLength=6`.
after:
- **Auth berbasis cookie httpOnly:** seluruh request API memakai `credentials:'include'`; token JWT otomatis dikirim via cookie `sigap_session` oleh browser (tidak lagi di `localStorage`).
- **CSRF double-submit:** `apiRequest` otomatis membaca cookie `sigap_csrf` dan mengirim header `X-CSRF-Token` untuk semua mutasi (`POST/PUT/PATCH/DELETE`).
- **Sesi terkelola via backend:** bootstrap sesi lewat `GET /auth/me`; event `sigap:auth-expired` di-dispatch saat 401 untuk reset state & redirect `/login`; helper `getCookie`, `clearAuthCookies`, `downloadApiFile` (dengan credentials) ditambahkan.
- `AuthContextType` tidak lagi mengekspos `token`; `WebSocketContext` tidak lagi mengirim `?token=` (browser mengirim cookie), `AuthenticatedImage` & seluruh fetch mentah di `cctv/page.tsx` memakai `credentials:'include'`.
- **Kebijakan password:** form buat/reset pengguna kini `minLength=8` (backend menerapkan kompleksitas minimal).
timestamp: 2026-08-01 13:45 WITA
kategori: Security, Auth, Hardening, API
file yang diubah:
- frontend/src/lib/api.ts
- frontend/src/context/AuthContext.tsx
- frontend/src/context/WebSocketContext.tsx
- frontend/src/app/login/page.tsx
- frontend/src/app/users/page.tsx
- frontend/src/app/cctv/page.tsx
- frontend/src/components/AuthenticatedImage.tsx
- frontend/changelog.md
alasan perubahan: Menutup temuan Fase 2 (docs/SECURITY-AUDIT.md) terkait penyimpanan token di localStorage, ketiadaan CSRF protection, dan kebijakan password lemah.

---

## [0.7.0] - 2026-08-01

### Hardening Keamanan Fase 1 (Security Audit & Roadmap)

perubahan yang dilakukan: **Perbaikan Stored XSS pada Cetak Label QR & Migrasi QR ke Server-side**
before:
- Cetak label QR (tunggal & batch) memakai `window.open().document.write()` dengan interpolasi mentah `nama`, `kode_cctv`, dan `lokasi_area` dari database (risiko Stored XSS — eksekusi skrip berbahaya saat mencetak).
- Gambar QR dibuat via API pihak ketiga `api.qrserver.com` dengan mengirim `QrPayload` terenkripsi ke server eksternal (kebocoran data & dependensi layanan luar yang tidak terautentikasi).
- Koneksi WebSocket tidak mengirimkan token sehingga backend tidak dapat mengautentikasi client.
after:
- Cetak label QR dirender ulang sebagai elemen React (bukan `document.write`); seluruh teks (nama, kode, lokasi) di-*escape* otomatis oleh React sehingga Stored XSS tertutup.
- Area cetak dirender via portal ke `document.body` (`#sigap-print-area`) dan hanya ditampilkan saat mencetak (`@media print`), dengan ukuran halaman `100mm × 38mm` (label tunggal) atau A4 (batch).
- Gambar QR ditampilkan dari endpoint server `GET /api/cctv/:id/qrcode` (dengan header `Authorization`), di-cache per perangkat, menggantikan `api.qrserver.com`.
- Koneksi WebSocket kini mengirim `?token=` dari `AuthContext` saat handshake.
timestamp: 2026-08-01 15:30 WITA
kategori: Security, Bugfix, UI/UX, API
file yang diubah:
- frontend/src/app/cctv/page.tsx
- frontend/src/app/globals.css
- frontend/src/context/WebSocketContext.tsx
- frontend/changelog.md
alasan perubahan: Menutup temuan kritis Stored XSS (docs/SECURITY-AUDIT.md) dan menghilangkan ketergantungan pada layanan QR eksternal, sekaligus mengamankan handshake WebSocket.

---

## [0.6.0] - 2026-08-01

### Dashboard Peta Interaktif GIS (Leaflet.js)

perubahan yang dilakukan: **Integrasi Komponen CCTVMap.tsx Berbasis Leaflet.js & OpenStreetMap IKN**
before:
- Halaman dashboard tidak memiliki visualisasi peta geografis sebaran perangkat CCTV di kawasan IKN.
after:
- Mengintegrasikan pustaka `leaflet` & `react-leaflet` dengan *dynamic import SSR-safe* (`next/dynamic` dengan `{ ssr: false }`).
- Dibuat komponen `CCTVMap.tsx` di `src/components/CCTVMap.tsx` yang memvisualisasikan seluruh perangkat CCTV terpusat di kawasan **IKN Nusantara** (`-0.9723, 116.7083`).
- Menyiapkan SVG DivIcon kustom ber-kode warna status: 🟢 **Aktif**, 🔴 **Nonaktif/Rusak**, 🟡 **Pemeliharaan**.
- Menambahkan **Interactive Popup Card** dan **Tooltip** yang menyajikan Nama CCTV, Kode, Lokasi, IP Kamera, Status, dan tombol pintas ke Master CCTV.
- Diintegrasikan ke halaman `src/app/dashboard/page.tsx`.
timestamp: 2026-08-01 01:44 WITA
kategori: Feature, GIS, UI/UX, Map
file yang diubah:
- frontend/package.json
- frontend/src/components/CCTVMap.tsx (baru)
- frontend/src/app/dashboard/page.tsx
- frontend/changelog.md
alasan perubahan: Menyajikan pemantauan visual spasial berbasis GIS untuk membantu pengelola sistem melihat sebaran posisi dan kondisi operasional perangkat CCTV di kawasan IKN secara cepat.

---

## [0.5.0] - 2026-08-01

### Audit Log Viewer UI (Admin Trail Exploration)

perubahan yang dilakukan: **Pembuatan Halaman Master Audit Log Trail & Navigasi Sidebar**
before:
- Tidak ada halaman antarmuka frontend untuk melihat rekam jejak audit trail aktivitas sistem yang dikumpulkan oleh backend.
after:
- Dibuat halaman master `src/app/audit-logs/page.tsx` khusus untuk peran Admin dengan proteksi RBAC.
- Diintegrasikan ke endpoint `GET /api/audit-logs` dengan dukungan paginasi, pencarian kata kunci (*search*), dan penyaringan berdasarkan modul (`AUTH`, `CCTV`, `ASSET`, `MAINTENANCE`, `USER`).
- Menambahkan **Modal Detail Audit Log** untuk menampilkan detail teknis lengkap (User ID, User Agent browser, Client IP proxy-aware, dan Timestamp presisi).
- Menambahkan menu navigasi **"Audit Log"** dengan ikon `ShieldCheck` pada sidebar `DashboardLayout.tsx` untuk Admin.
- Mengintegrasikan listener real-time WebSocket (`AUDIT_LOG_ADDED`) untuk otomatis memuat ulang log saat ada aktivitas baru.
timestamp: 2026-08-01 01:36 WITA
kategori: Feature, Admin, Security, UI/UX
file yang diubah:
- frontend/src/app/audit-logs/page.tsx (baru)
- frontend/src/components/DashboardLayout.tsx
- frontend/changelog.md
alasan perubahan: Memberikan kapabilitas auditabilitas penuh kepada Admin untuk memantau keamanan dan riwayat operasional infrastruktur IKN.

---

## [0.4.0] - 2026-08-01

### Real-time WebSocket Client Integration & Live Auto-Refetch

perubahan yang dilakukan: **Integrasi WebSocketProvider & Subskripsi Event Real-time pada Halaman Utama**
before:
- Antarmuka tidak secara otomatis memperbarui tabel atau statistik saat ada perubahan data di server oleh pengguna lain.
after:
- Dibuat `WebSocketContext.tsx` di `src/context/WebSocketContext.tsx` dengan fitur otomatisasi *auto-reconnect* (exponential backoff).
- Dibungkus `WebSocketProvider` di `src/app/layout.tsx`.
- Diintegrasikan *hook subscription* `useWebSocket` pada halaman utama:
  - `src/app/dashboard/page.tsx`: Mendengarkan `CCTV_UPDATED`, `MAINTENANCE_CREATED`, dan `MAINTENANCE_UPDATED` untuk memuat ulang statistik & daftar CCTV/log secara otomatis.
  - `src/app/cctv/page.tsx`: Mendengarkan `CCTV_UPDATED` untuk memuat ulang tabel master CCTV secara otomatis.
  - `src/app/maintenance/page.tsx`: Mendengarkan `MAINTENANCE_CREATED` dan `MAINTENANCE_UPDATED` untuk memuat ulang tabel riwayat pemeliharaan secara otomatis.
timestamp: 2026-08-01 01:35 WITA
kategori: Feature, Real-time, UI/UX
file yang diubah:
- frontend/src/context/WebSocketContext.tsx (baru)
- frontend/src/app/layout.tsx
- frontend/src/app/dashboard/page.tsx
- frontend/src/app/cctv/page.tsx
- frontend/src/app/maintenance/page.tsx
- frontend/changelog.md
alasan perubahan: Menyajikan pengalaman pengguna yang sangat responsif di mana setiap perubahan data CCTV atau log pemeliharaan langsung memicu pembaruan antarmuka secara instan tanpa perlu refresh.

---

## [0.3.0] - 2026-08-01

### Inactivity Auto-Logout & Idle Timer Security

perubahan yang dilakukan: **Integrasi Pelacak Keaktifan Pengguna, Warning Modal (14 Menit), dan Auto-Logout (15 Menit)**
before:
- Sesi pengguna di frontend tetap aktif di localStorage tanpa batas waktu pasif/inaktivitas selama browser terbuka.
- Tidak ada mekanisme otomatis untuk mencabut token JWT jika pengguna meninggalkan komputer dalam kondisi ter-login.
after:
- Diimplementasikan pelacak keaktifan pengguna global (`mousemove`, `keydown`, `click`, `scroll`, `touchstart`) pada `AuthContext.tsx`.
- Menambahkan modal peringatan `IdleWarningModal` yang muncul di menit ke-14 dengan countdown detik dan opsi **"Tetap Masuk"** untuk memperpanjang sesi.
- Otomatisasi pemanggilan `logout('inactivity')` yang mengirimkan request `POST /api/auth/logout` (mencabut token ke blacklist backend) dan mengosongkan sesi lokal setelah 15 menit pasif.
- Menambahkan alert banner pemberitahuan keamanan inaktivitas pada `src/app/login/page.tsx`.
timestamp: 2026-08-01 01:30 WITA
kategori: Feature, Security, Auth
file yang diubah:
- frontend/src/context/AuthContext.tsx
- frontend/src/app/login/page.tsx
- frontend/changelog.md
alasan perubahan: Meningkatkan standar keamanan infrastruktur kritis IKN dengan memastikan sesi pengguna pasif diakhiri secara otomatis dan token dicabut dari server.

---

## [0.2.1] - 2026-08-01

### Fix Session Storage Restoration & Direct Route Navigation

perubahan yang dilakukan: **Penguatan Sesi AuthContext & Direct Navigation /dashboard**
before:
- Pemulihan sesi pada `useEffect` `AuthContext.tsx` tidak memiliki penanganan `try...catch` dan pemeriksaan string invalid (`undefined`/`null`), sehingga potensi error `JSON.parse` membatalkan proses pembacaan sesi.
- Fungsi `login` melakukan navigasi ke `/` yang kemudian di-redirect ke `/dashboard`.
after:
- Menambahkan penanganan `try...catch...finally` dan pembersihan token invalid pada `useEffect` `AuthContext.tsx`.
- Mengubah alur navigasi `login` agar langsung mengarah ke `/dashboard`.
timestamp: 2026-08-01 01:25 WITA
kategori: Bugfix, Auth, Session
file yang diubah:
- frontend/src/context/AuthContext.tsx
- frontend/changelog.md
alasan perubahan: Memastikan sesi pengguna pulih secara stabil saat berpindah halaman dan mencegah pemutusan sesi tak terduga.

---

## [0.2.0] - 2026-08-01

### Cetak QR Code Massal (Batch PDF / Print Label Export)

perubahan yang dilakukan: **Integrasi Fitur Cetak QR Code Massal & Floating Bulk Bar Action**
before:
- Pencetakan label QR Code hanya dapat dilakukan satu per satu via modal detail CCTV (`handlePrintQR`).
- Floating Bulk Action Bar hanya menyediakan tombol hapus massal ("Hapus [N] CCTV").
- Tidak ada opsi untuk mencetak seluruh stiker QR Code sekaligus dari daftar master CCTV.
after:
- Diimplementasikan fungsi `handleBatchPrintQR(targetIDs?: string[])` di `src/app/cctv/page.tsx` yang mendukung pencetakan batch multiple CCTV terpilih maupun seluruh CCTV.
- Ditambahkan tombol **"Cetak [N] QR"** (dengan ikon Printer) pada *Floating Bulk Action Bar* ketika 1 atau lebih CCTV dicentang pada tabel.
- Ditambahkan opsi **"Cetak Semua QR Code"** pada menu dropdown tindakan di header tabel CCTV.
- Tata letak cetak (*Print Layout*) dirancang menggunakan grid stiker label berukuran **92mm x 35mm** (proporsional untuk kertas A4 / stiker label 100x38mm) lengkap dengan Logo Nusantara, QR Code terenkripsi, Kode CCTV, Nama CCTV, dan Lokasi Area dengan proteksi `@media print` & `break-inside: avoid`.
timestamp: 2026-08-01 00:44 WITA
kategori: Feature, UI/UX, Batch Processing
file yang diubah:
- frontend/src/app/cctv/page.tsx
- frontend/changelog.md
alasan perubahan: Memudahkan Admin dan Teknisi menyebarkan stiker QR Code fisik di lapangan IKN dengan mencetak puluhan label QR Code secara sekaligus atau menyimpannya langsung sebagai PDF melalui dialog print browser.

---

## [0.1.0] - 2026-07-22

### Added New
- Inisialisasi proyek Next.js menggunakan **TypeScript**, **App Router**, **ESLint**, dan **Tailwind CSS v4** di dalam direktori `src/`.
- Pemasangan pustaka **`lucide-react`** untuk kebutuhan ikon antarmuka.
- Pembuatan API client helper (`src/lib/api.ts`) untuk mempermudah pemanggilan REST API backend dengan dukungan otomatisasi otentikasi Bearer Token JWT.
- Pembuatan **`AuthContext`** (`src/context/AuthContext.tsx`) untuk memanajemen data sesi login pengguna (token JWT dan detail peran pengguna).
- Pembuatan **`ThemeContext`** (`src/context/ThemeContext.tsx`) untuk mendukung fitur **Light/Dark Toggle** yang dinamis berbasis penyimpanan `localStorage`.
- Pembuatan komponen proteksi rute **`ProtectedRoute`** (`src/components/ProtectedRoute.tsx`) guna mengamankan halaman dari akses ilegal sebelum login.
- Pembuatan layout dashboard **`DashboardLayout`** (`src/components/DashboardLayout.tsx`) yang menyediakan sidebar navigasi responsif, avatar pengguna, lencana peran, serta tombol keluar (logout).
- Pembuatan halaman **Login** (`src/app/login/page.tsx`) dengan tampilan premium berbasis *glassmorphism card* dan validasi input.
- Pembuatan halaman **Dashboard Aset CCTV** (`src/app/page.tsx`) yang menampilkan ringkasan statistik (Total, Aktif, Rusak, Pemeliharaan), tabel aset CCTV dengan pencarian & filter, serta modal form penambahan/edit aset (Admin saja).
- Pembuatan halaman **Riwayat Pemeliharaan** (`src/app/maintenance/page.tsx`) untuk mencatat log pemeliharaan preventif/korektif lengkap dengan filter CCTV, filter status, form log baru, dan fitur hapus log (Admin saja).

### Changed
- Konfigurasi file `package.json` agar server pengembangan Next.js berjalan di **port 3001** (`next dev -p 3001`) untuk menghindari tabrakan port dengan server backend Go Fiber di port 3000.
- Penyesuaian `globals.css` dengan variabel CSS khusus Tailwind v4 untuk mendukung transisi tema yang halus.

perubahan yang dilakukan:
Mengintegrasikan endpoint backend yang belum memiliki antarmuka dan menyelaraskan tampilan berdasarkan role admin/teknisi.
before:
Navigasi hanya menampilkan Aset CCTV dan Log Pemeliharaan; endpoint aset legacy, statistik, health check, upload bukti, ekspor, detail, serta manajemen pengguna belum memiliki alur UI lengkap. Aksi ubah log juga tersembunyi dari teknisi meskipun backend mengizinkannya.
after:
Menambahkan Dashboard API, Aset Legacy, dan Manajemen Pengguna; mengintegrasikan detail CCTV/log, ekspor CSV, QR aset legacy, upload foto bukti, serta seluruh operasi pengguna. Navigasi dan aksi kini mengikuti role: admin memperoleh fitur kelola penuh, sedangkan teknisi memperoleh akses operasional tanpa aksi khusus admin.
timestamp:
2026-07-28 12:15:58 WITA
kategori:
Feature, Integration, RBAC, UI/UX
file yang diubah:
src/app/dashboard/page.tsx, src/app/assets/page.tsx, src/app/users/page.tsx, src/app/cctv/page.tsx, src/app/maintenance/page.tsx, src/app/page.tsx, src/components/DashboardLayout.tsx, src/lib/api.ts, changelog.md
alasan perubahan:
Memastikan endpoint runtime SIGAP terlihat dan dapat digunakan dari sistem sesuai kewenangan dua role, tanpa menghapus endpoint aset lama yang masih tumpang tindih.

perubahan yang dilakukan:
Menghapus Aset Legacy dari antarmuka dan memisahkan dashboard berdasarkan role.
before:
Halaman dan navigasi Aset Legacy menampilkan data seeder/dummy. Admin dan teknisi menggunakan susunan dashboard yang sama, serta tombol Import Excel masih terlihat oleh teknisi.
after:
Aset Legacy tidak lagi tampil pada route maupun sidebar. Dashboard admin menampilkan ringkasan pengelolaan CCTV, pemeliharaan, dan pengguna; dashboard teknisi hanya menampilkan status CCTV, prioritas pekerjaan, scanner, serta akses pencatatan pemeliharaan. Import Excel dibatasi untuk admin.
timestamp:
2026-07-28 12:49:59 WITA
kategori:
Changed, RBAC, UI/UX
file yang diubah:
src/app/dashboard/page.tsx, src/app/assets/page.tsx, src/app/cctv/page.tsx, src/components/DashboardLayout.tsx, changelog.md
alasan perubahan:
Mencegah data dummy tampil sebagai data operasional dan memastikan setiap role hanya melihat fitur yang sesuai dengan hak akses backend.

perubahan yang dilakukan:
Menyederhanakan teks dashboard teknisi dan placeholder pencarian CCTV.
before:
Dashboard teknisi menampilkan deskripsi tambahan di bawah sapaan dan kolom pencarian menggunakan placeholder panjang yang menyebut seluruh atribut.
after:
Deskripsi tambahan dashboard teknisi dihapus dan placeholder pencarian disederhanakan menjadi "Cari CCTV".
timestamp:
2026-07-28 13:10:03 WITA
kategori:
UI/UX
file yang diubah:
src/app/dashboard/page.tsx, src/app/cctv/page.tsx, changelog.md
alasan perubahan:
Membuat antarmuka teknisi lebih ringkas dan sesuai arahan desain.

perubahan yang dilakukan:
Menyempurnakan dashboard per role, identitas sidebar, loading state, dan halaman login.
before:
Dashboard masih menampilkan shortcut pengelolaan, rincian status pemeliharaan belum lengkap, status API hanya berupa label singkat, logo menghilang saat sidebar diciutkan, dan sebagian aksi belum memiliki indikator proses. Halaman login masih memuat terlalu banyak teks dan efek visual.
after:
Dashboard admin menampilkan rincian pemeliharaan terjadwal, berjalan, selesai, dan gagal tanpa bagian Akses Pengelolaan Admin. Dashboard teknisi menampilkan prioritas CCTV dan status pekerjaan tanpa bagian Akses Operasional Teknisi. Kesehatan API dilengkapi keterangan dan waktu pemeriksaan, logo aplikasi tetap terlihat saat sidebar kecil, aksi utama memiliki loading state, dan login dibuat lebih ringkas dengan palet hijau-kuning yang konsisten.
timestamp:
2026-07-29 10:45:02 WITA
kategori:
UI/UX, RBAC, Loading State
file yang diubah:
src/app/dashboard/page.tsx, src/app/login/page.tsx, src/app/cctv/page.tsx, src/app/maintenance/page.tsx, src/components/DashboardLayout.tsx, changelog.md
alasan perubahan:
Meningkatkan keterbacaan informasi operasional, memberikan umpan balik yang jelas saat proses berjalan, dan menjaga konsistensi identitas visual pada seluruh role.

perubahan yang dilakukan:
Mengurangi ikon dekoratif pada login, sidebar, dan dashboard serta mempertahankan akses keluar saat sidebar kecil.
before:
Login memiliki ikon kunci dekoratif, identitas SIGAP di sidebar didampingi ikon database, tombol keluar menghilang ketika sidebar diciutkan, dan dashboard menggunakan ikon pada hampir setiap kartu.
after:
Ikon dekoratif login dan identitas sidebar dihapus. Tulisan SIGAP tetap tampil ringkas saat sidebar kecil, tombol keluar tetap terlihat sebagai tombol ikon, dan dashboard menggunakan tipografi, garis warna, serta titik status sebagai penanda utama.
timestamp:
2026-07-29 10:56:08 WITA
kategori:
UI/UX
file yang diubah:
src/app/login/page.tsx, src/app/dashboard/page.tsx, src/components/DashboardLayout.tsx, changelog.md
alasan perubahan:
Membuat antarmuka lebih tenang, klasik, dan tidak terlihat dipenuhi ikon dekoratif.

perubahan yang dilakukan:
Menyeimbangkan kartu ringkasan, memperjelas prioritas teknisi, dan memperbaiki sumber total pemeliharaan.
before:
Kartu dashboard dan Aset CCTV memakai susunan serta definisi angka yang berbeda. Nilai Pemeliharaan dihitung dari status perangkat sehingga tetap nol walaupun sudah ada catatan pemeliharaan. Prioritas teknisi mengulang informasi CCTV dan kartu kesehatan API belum cukup terstruktur.
after:
Kartu ringkasan pada dashboard dan Aset CCTV memakai urutan yang konsisten: Total CCTV, Aktif, Nonaktif, dan Pemeliharaan. Pemeliharaan mengambil total catatan dari endpoint log pemeliharaan. Prioritas teknisi hanya menampilkan status pekerjaan Terjadwal, Dalam Proses, Selesai, dan Gagal. Kesehatan API menampilkan status online, ketersediaan backend, sinkronisasi data, serta waktu pemeriksaan.
timestamp:
2026-07-29 11:06:30 WITA
kategori:
UI/UX, Data Integration
file yang diubah:
src/app/dashboard/page.tsx, src/app/cctv/page.tsx, changelog.md
alasan perubahan:
Memastikan angka mudah dibandingkan, tidak membingungkan pengguna, dan berasal dari sumber data yang sesuai.

perubahan yang dilakukan:
Mengatasi rate limit dashboard teknisi dan menambahkan detail interaktif aktivitas pemeliharaan.
before:
Dashboard mengirim delapan request bersamaan sehingga backend dapat membalas 429 dan seluruh data ikut kosong. Kartu status pemeliharaan tidak dapat dibuka, serta kartu ringkasan masih memiliki teks keterangan kecil.
after:
Dashboard hanya mengirim tiga request dan memproses hasil CCTV, pemeliharaan, serta health check secara terpisah. Kegagalan health check tidak lagi mengosongkan data CCTV. Kartu Terjadwal, Dalam Proses, Selesai, dan Gagal dapat ditekan untuk menampilkan nama CCTV, lokasi, serta tindakan pemeliharaan. Teks keterangan kecil pada kartu ringkasan dihapus.
timestamp:
2026-07-29 11:14:19 WITA
kategori:
Bug Fix, UI/UX, Data Integration
file yang diubah:
src/app/dashboard/page.tsx, src/app/cctv/page.tsx, changelog.md
alasan perubahan:
Menghindari pembatasan request backend, menjaga data dashboard tetap tampil secara parsial, dan memudahkan pengguna melihat detail pekerjaan langsung dari ringkasan status.
