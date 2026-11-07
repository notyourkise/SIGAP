# Desain Teknis Monitoring CCTV, Tiket Gangguan Otomatis, dan Pemberitahuan Telegram SIGAP (Revisi 3 — Final Pra-Implementasi)

**Tanggal**: 7 November 2026 (WITA)  
**Checkout Acuan**: `HEAD d1f0d65` (branch `main`)  
**Dokumen Acuan**:
1. [KEBUTUHAN-MONITORING-CCTV-TELEGRAM-2026-11-01.md](file:///d:/laragon/www/SIGAP/docs/05-reports/KEBUTUHAN-MONITORING-CCTV-TELEGRAM-2026-11-01.md) — **Acuan kebutuhan yang telah disetujui pengguna**
2. [PEMETAAN-TEKNIS-MONITORING-TELEGRAM-2026-11-01.md](file:///d:/laragon/www/SIGAP/docs/05-reports/PEMETAAN-TEKNIS-MONITORING-TELEGRAM-2026-11-01.md) — **Pemetaan awal Anto pada commit `6a61213`, diverifikasi ulang terhadap `HEAD d1f0d65`**
3. [AGENTS.md](file:///d:/laragon/www/SIGAP/docs/01-documentation/AGENTS.md) (Bagian 11.1, 11.2, dan 11.6) serta [ARSIP-DATA-CCTV-THD-2026-10-20.md](file:///d:/laragon/www/SIGAP/docs/05-reports/ARSIP-DATA-CCTV-THD-2026-10-20.md)
4. **Tujuh Keputusan Operasional Pengguna & Lima Koreksi Teknis Hasil Tinjauan Anto (7 November 2026)** — seluruhnya telah diselaraskan pada Revisi 3 ini.

**Status Dokumen**: Desain Teknis Revisi 3 — Acuan implementasi pada branch fitur terisolasi. Pembuktian target end-to-end 5 menit terhadap perangkat fisik, kapasitas NVR aktual, dan konsumsi kuota GSM nyata tetap merupakan bagian dari pengujian staging terkendali (bukan klaim sebelum diuji).

---

## Ringkasan 5 Koreksi Teknis Terfokus pada Revisi 3

| Prioritas | Temuan Tinjauan Revisi 2 | Koreksi Teknis pada Revisi 3 & Kriteria Penerimaan | Lokasi Bagian |
|---|---|---|---|
| **High** | Semantik awal SLA & waktu offline belum seragam (`actual_offline_at` memakai waktu deteksi; `suppressed_offline_since` disebut waktu offline asli padahal waktu konfirmasi). | Memisahkan tegas: (1) waktu gangguan terkendali saat uji (`physical_fault_at`, tidak pernah dikarang di produksi), (2) waktu gagal pertama teramati (`first_failed_observed_at`), dan (3) waktu gagal terkonfirmasi (`confirmed_offline_at`), beserta `timestamp_source`. Di produksi, durasi dilaporkan jujur sebagai **berdasarkan observasi** (`sla_clock_start_at = first_failed_observed_at` untuk luar jadwal, dan `effective_ends_at` untuk pasca-maintenance). Di staging, target $\le 300\text{ dtk}$ diukur dari waktu gangguan dikendalikan s.d. pesan tampil di grup. | Bagian 2.1, 4.3, 4.4, 6.1, 8.1, 8.2 |
| **High** | Estimasi waktu belum konsisten dengan aturan retry ambigu (`5s` di Bagian 4 vs `10s` di Bagian 5), belum menggabungkan batas sesi RTSP lintas antrean beserta ekor sumber terbuka (`closeAfter`), dan belum menghitung antrean saat banyak jendela maintenance berakhir bersamaan. | Menyatukan seluruh pemeriksaan (`Confirm`, `Post-Maintenance`, `Main`) di bawah **satu *Unified NVR Session Gate*** (pacer tunggal `1,25s` + *Upstream Lease Semaphore* `MaxTotalUpstreamSlots = 10` yang ikut menahan slot selama masa ekor `sourceOnDemandCloseAfter = 5s`). Menyeragamkan jeda retry `ambiguous` menjadi `10,0s` di semua bagian (`20,0s` total kirim + retry). Menghitung kasus 35 jendela maintenance berakhir bersamaan (`~102,5s` estimasi model) dan memberi label seluruh angka sebagai estimasi ber-asumsi sampai diuji beban di staging. | Bagian 2.4, 4.2, 4.3, 4.4, 5.4 |
| **High** | Waktu antre vs waktu konfirmasi API vs pelanggaran SLA tercampur (`*_sent_at` diisi saat masuk antrean; `sla_breached` hanya disebut saat terlambat terkirim). | Mengganti kolom pengingat menjadi `assignment_reminder_queued_at` & `unrecovered_escalation_queued_at` (+ unique index parsial `(incident_id, event_type)` di `telegram_outbox`). Memisahkan `created_at` (diantrekan), `sent_at` (dikonfirmasi API Telegram), dan `failed_at`/`ambiguous`. Menandai `sla_breached = true` segera saat `now > sla_deadline_at` tanpa pengiriman terkonfirmasi (`sent_at IS NULL`), termasuk bila gagal permanen. Mencatat eksplisit bahwa retry `ambiguous` tetap dapat menghasilkan pesan duplikat di grup. | Bagian 3.7, 5.4, 6.1, 8.1 |
| **High** | Migrasi belum menjelaskan pengisian kolom `NOT NULL` baru untuk baris tiket lama dari skema `000031`. | Merancang migrasi 3-langkah (*Expand nullable $\rightarrow$ Deterministic Backfill $\rightarrow$ Enforce NOT NULL & UNIQUE*): mengisi baris lama dengan `INC-LEGACY-YYYYMMDD-XXXX`, menyalin timestamp historis (`opened_at`/`last_observed_at`), dan menandainya dengan `timestamp_source = 'legacy_backfill_opened_at'`. Menambahkan uji migrasi pada DB terisolasi berisi data skema `000031`. | Bagian 6.1, 8.1 |
| **Medium** | Satu `replacement_maintenance_log_id` di tingkat tiket tidak cukup bila pada tiket bersama terdapat $>1$ kamera yang perlu diganti unit dengan tindak lanjut masing-masing. | Memindahkan hasil akhir (`member_outcome`), catatan kerusakan (`damage_findings`), dan tautan tugas penggantian (`replacement_maintenance_log_id`) ke tingkat per kamera (`cctv_incident_cameras`), serta menambah `resolution_type = 'partial_recovery_and_escalation'` di tingkat induk. Laporan membaca hasil setiap kamera meski tiket induk sudah ditutup. | Bagian 3.1, 3.6, 6.1, 8.1 |

---

## Ringkasan 7 Keputusan Operasional yang Dipertahankan

1. **Penahanan Alarm Terencana**: Tugas rutin `ongoing` saja tidak menahan tiket/Telegram. Admin harus membuat jendela penghentian terencana berbatas waktu & alasan (`cctv_planned_outages`). Selama jendela aktif, tampilkan **Sedang pemeliharaan terjadwal** dan tetap jalankan pemeriksaan. Jika kamera masih terkonfirmasi offline saat jendela berakhir, langsung buat tiket & kirim notifikasi tanpa mengulang rangkaian pemeriksaan dari awal; awal SLA 5 menit pasca-maintenance diukur dari `effective_ends_at`, sementara waktu observasi kegagalan selama maintenance tetap dicatat terpisah.
2. **Pulih Mandiri Tanpa Teknisi**: Setelah pemulihan stabil 5 menit terkonfirmasi, Admin dapat menutup tiket tanpa menugaskan teknisi dengan alasan **Pulih Mandiri (`self_recovered`)**, tanpa membuat `MaintenanceLog` atau checklist.
3. **Gabungkan Manual ke Tiket Bersama**: Tersedia tombol manual **“Gabungkan ke Tiket Bersama”** untuk Admin setelah bukti keterkaitan cukup. Tiket asal ditandai **Digabung (`merged`)** dan mengarah ke tiket induk dengan jejak waktu observasi, catatan, dan referensi tiket awal tetap utuh.
4. **Tugas Perbaikan Bersama NVR/Switch**: Penugasan tiket bersama membuat satu tugas perbaikan perangkat bersama (`shared_infrastructure`). Daftar kamera terdampak dan status pulih masing-masing tetap dipantau pada insiden; tugas lapangan individual per kamera dibuat hanya bila unit tertentu memang perlu diperiksa/diperbaiki di lapangan.
5. **Eskalasi Penggantian Perangkat**: Admin diizinkan menonaktifkan kamera dan menyelesaikan keterlibatan kamera/insiden dengan alasan **Eskalasi Penggantian Perangkat (`escalated_replacement`)**, bersyarat catatan kerusakan & tugas penggantian ber-penanggung-jawab tertaut per kamera. Hasil **dieskalasikan** dipisahkan dari **pulih** di laporan; setelah diganti, Admin mengaktifkan kembali kamera dan memverifikasi stream.
6. **Grup Khusus Admin & Gangguan Pemantau**: Menggunakan notifikasi in-app, Web Push, dan **grup Telegram khusus Admin** (1 bot dengan tujuan grup berbeda). Peringatan `unknown` dikirim sebagai gangguan **pemantau** sebanyak **1 notifikasi per kejadian (episode)**, bukan per kamera; email ditunda; deteksi server SIGAP mati total membutuhkan pemantau di luar server.
7. **Pengingat 30 Menit & Eskalasi 4 Jam**: Satu pengingat penugasan setelah 30 menit (untuk tiket terbuka, belum ditugaskan, dan belum pulih) dan satu eskalasi ke Admin setelah 4 jam sejak pembukaan tiket jika gangguan masih berlangsung (masing-masing maksimal 1 event & 1 entri antrean per insiden; angka 4 jam sebagai parameter konfigurasi awal).

---

## 1. Hasil Pemeriksaan Ulang Kode Terbaru (`HEAD d1f0d65`)

### 1.1 Perkembangan Sejak Pemetaan Anto (`6a61213`)

1. **Prototipe Awal Monitoring Insiden (`cb65dcb` & `542d18e`)**:
   Tersedia migrasi [000031_cctv_incidents.up.sql](file:///d:/laragon/www/SIGAP/backend/migrations/000031_cctv_incidents.up.sql) (`cctv_incidents`, `cctv_incident_cameras` dengan partial unique index `idx_camera_one_active_incident`, dan `telegram_outbox`), paket [backend/incidents/](file:///d:/laragon/www/SIGAP/backend/incidents/store.go), handler/rute [backend/handlers/incidents.go](file:///d:/laragon/www/SIGAP/backend/handlers/incidents.go) & [backend/routes/incidents.go](file:///d:/laragon/www/SIGAP/backend/routes/incidents.go), serta halaman [frontend/src/app/incidents/page.tsx](file:///d:/laragon/www/SIGAP/frontend/src/app/incidents/page.tsx).
2. **Hardening Audit Terkait (`#173`–`#186`)**:
   Validasi keanggotaan `user_sites` teknisi pada penugasan ([maintenance.go](file:///d:/laragon/www/SIGAP/backend/handlers/maintenance.go#L631-L637)), pembatasan sesi `must_change_password` ([auth.go](file:///d:/laragon/www/SIGAP/backend/middleware/auth.go)), proteksi konkurensi penyelesaian tugas (`3c2e285`), dan validasi template checklist + foto bukti (`f2b38b5`).

### 1.2 Strategi Penggunaan Ulang & Penyempurnaan

- **Digunakan Ulang**: Fondasi tabel `000031`, `pg_advisory_xact_lock` per site, klaim outbox `FOR UPDATE SKIP LOCKED` ber-lease di luar transaksi HTTP, pemeriksaan MediaMTX Sub-Stream (`02`) zero-transcoding, `ProbeRTSPHost` untuk korelasi NVR, `NotifyUser`/`NotifyRole` + Web Push, `LogAudit`, serta isolasi `SiteContext()` + `user_sites`.
- **Disempurnakan untuk Menutup Kesenjangan**:
  1. Mengganti worker tunggal lambat dengan *Unified NVR Session Gate* (pacer tunggal + antrean konfirmasi cepat + pelacak ekor `closeAfter` agar terhindar dari `453 Not Enough Bandwidth` sebagaimana tercatat di [stream.go](file:///d:/laragon/www/SIGAP/backend/handlers/stream.go#L22-L26)).
  2. Menambahkan verifikasi kemajuan `BytesReceived` ($\Delta > 0$) di atas `SourceReady()`.
  3. Memisahkan `monitoring_enabled` dari `stream_enabled` dan menambahkan jendela pemeliharaan terencana (`cctv_planned_outages`).
  4. Menambahkan korelasi tiket bersama `shared_nvr`, tombol manual `merge`, resolusi `self_recovered`, eskalasi penggantian per kamera (`escalated_replacement`), episode gangguan pemantau ke grup Admin, *batching* + rate-limit grup Telegram, pemisahan waktu antre vs konfirmasi API vs `sla_breached`, serta telemetri konsumsi data GSM.
  5. Memperbaiki `MaxEvidenceGap` menjadi `105s` ($\ge 6$ sampel pada siklus 60s) dan memperbaiki bug `ResetObservations()` di [store.go](file:///d:/laragon/www/SIGAP/backend/incidents/store.go#L203) agar tiket `pending_verification` tanpa teknisi kembali ke `open` saat restart.

---

## 2. Pemeriksaan Kesehatan, Semantik Waktu, Jendela Terjadwal, Pemantau Gagal, & GSM

### 2.1 Cakupan Kamera, Semantik Waktu Observasi, & Jendela Penghentian Terencana

1. **Cakupan Seluruh CCTV**:
   - Seluruh CCTV dengan `deleted_at IS NULL`, `status != 'nonaktif'`, dan `monitoring_enabled = true` (baik **Metro-E** maupun **GSM**) diperiksa secara aktif menggunakan Sub-Stream (`02`). Kamera berstatus `nonaktif` tidak diprobe dan diberi label **Nonaktif (Tidak Dipantau)**.
2. **Pemisahan Tiga Konsep Waktu Gangguan (Koreksi #1)**:
   Di lingkungan produksi, sistem pemantau berkala **tidak mengetahui** detik pasti perangkat putus secara fisik di antara dua siklus pemeriksaan. Oleh karena itu skema dan laporan membedakan secara ketat:
   - **`physical_fault_at`** *(hanya pada lingkungan uji/staging terkendali)*: waktu injeksi gangguan yang diketahui penguji untuk mengukur kriteria penerimaan produk $\le 300\text{ detik}$. Tidak pernah diisi/dikarang pada operasi produksi.
   - **`first_failed_observed_at`**: stempel waktu saat verifier pertama kali mengamati kegagalan kanal (`failures = 1`).
   - **`confirmed_offline_at`**: stempel waktu saat pemeriksaan konfirmasi independen membuktikan kamera masih gagal (`failures >= 2`).
   - **`timestamp_source`**: `'live_observation'` untuk insiden hasil pemantauan aktif, atau `'legacy_backfill_opened_at'` untuk baris historis dari skema `000031` yang hanya memiliki waktu pembuatan tiket.
   - Pada UI dan ekspor laporan produksi, waktu gangguan diberi label eksplisit **“Waktu Gagal Teramati Pertama”** dan **“Waktu Offline Terkonfirmasi”** (disertai catatan bahwa pemutusan fisik dapat terjadi paling cepat 1 interval siklus sebelum kegagalan pertama teramati).
3. **Jendela Penghentian Terencana (`cctv_planned_outages`) & Awal SLA**:
   - Tugas `MaintenanceLog` rutin `ongoing` biasa **tidak** menahan tiket gangguan.
   - Admin menandai penghentian yang direncanakan pada `cctv_planned_outages` (`site_id`, daftar `cctv_id`, `reason` wajib, `starts_at`, `ends_at`, dan opsional `ended_early_at`, sehingga `effective_ends_at = COALESCE(ended_early_at, ends_at)`).
   - Selama `starts_at <= now < effective_ends_at`:
     - Status tampilan kamera menjadi **Sedang pemeliharaan terjadwal**.
     - Pemeriksaan berkala **tetap berjalan**; bila kamera gagal, sistem mencatat `suppressed_first_failed_at` dan `suppressed_confirmed_offline_at` (sebagai waktu observasi selama jendela, bukan waktu putus fisik) tanpa membuka tiket atau mengirim Telegram. Bila kamera kembali `live` sebelum jendela usai, kedua penanda tersebut dihapus.
   - Saat `now >= effective_ends_at`:
     - Jika kamera masih memegang status gagal terkonfirmasi dari masa jendela (`suppressed_confirmed_offline_at IS NOT NULL` dan sampel terakhir masih segar $\le 75\text{s}$), sistem **langsung membuka tiket & mengantrekan Telegram tanpa mengulang 2 pemeriksaan dari awal**. Jika sampel terakhir berusia $> 75\text{s}$, kamera dimasukkan ke antrean prioritas untuk 1 pemeriksaan penyegaran tunggal sebelum membuka tiket.
   - **Penetapan `sla_clock_start_at` dan `sla_clock_basis`**:
     - Gangguan di luar jadwal (`origin_type = 'unplanned_outage'`): `sla_clock_start_at = first_failed_observed_at` (`sla_clock_basis = 'first_failed_observation'`), `sla_deadline_at = sla_clock_start_at + 5 menit`.
     - Kamera masih offline pasca-jadwal (`origin_type = 'post_planned_maintenance'`): `first_failed_observed_at` dan `confirmed_offline_at` menyimpan waktu observasi asli selama maintenance, sedangkan **`sla_clock_start_at = effective_ends_at`** (`sla_clock_basis = 'planned_window_end'`), sehingga target notifikasi 5 menit dan pengingat 30 menit dihitung secara jujur sejak akhir periode maintenance yang disetujui.

### 2.2 Empat Lapis Bukti Kesehatan Video

1. **Lapis 1 — Kesehatan Pemantau**: Kesiapan API & HLS MediaMTX lokal + keabsahan dekripsi konfigurasi RTSP. Kegagalan di lapis ini menghasilkan `unknown`.
2. **Lapis 2 — Korelasi Host NVR (`ProbeRTSPHost`)**: Diuji saat kanal gagal (dan maks 1x per NVR per gelombang korelasi) sebagai bukti pendukung keterjangkauan NVR, bukan bukti tunggal kesehatan kanal video.
3. **Lapis 3 — Kesiapan Sumber MediaMTX**: `info.SourceReady() == true` dan `len(info.Tracks) > 0`. Error kapasitas/server menghasilkan `unknown`, bukan `offline`.
4. **Lapis 4 — Kemajuan Aliran Byte Video ($\Delta\text{BytesReceived} > 0$)**: Membandingkan pembacaan `BytesReceived` berjarak `500ms` (atau memastikan byte $> 0$ pada sesi on-demand yang baru ditarik dari nol) untuk mendeteksi stream beku (*stalled stream*) tanpa transcoding FFmpeg.

### 2.3 Gangguan Pemantau (`unknown`), Grup Khusus Admin, & Pemantau Eksternal

- Kegagalan pemantau dikelompokkan dalam **`monitor_health_episodes`** sehingga transisi sehat $\rightarrow$ terganggu hanya menerbitkan **1 notifikasi per kejadian** (mencantumkan sebab dan jumlah kamera terdampak, bukan 1 pesan per kamera) ke **Notifikasi In-App Admin**, **Web Push**, dan **Grup Telegram Khusus Admin** (`target_channel = 'admin_group'`, menggunakan bot yang sama dengan chat ID grup Admin terpisah). Saat pemantau pulih stabil 2 siklus, episode ditutup dengan 1 pesan pemulihan.
- Tiket kamera yang sudah terbuka sebelum pemantau gagal tetap dipertahankan; anggota yang sedang `recovering` kembali ke `unknown` (hitungan 5 menit diulang saat bukti `live` segar kembali ada).
- Untuk mendeteksi **server SIGAP mati total**, disediakan endpoint ringan ber-autentikasi scope `health:read` (`GET /api/streams/monitor-heartbeat` yang melaporkan `last_verifier_cycle_at`) untuk dipantau oleh layanan di luar server SIGAP.

### 2.4 Pembatasan Monitoring Aktif GSM & Telemetri Konsumsi Data

- Sesi verifikasi dibatasi pada Sub-Stream (`02`), pemicu HLS `1,5s`, polling maks `6,0s`, maks 1 konfirmasi ulang, dan `sourceOnDemandCloseAfter = "5s"` (selaras antara [stream.go](file:///d:/laragon/www/SIGAP/backend/handlers/stream.go#L26) dan [mediamtx.yml](file:///d:/laragon/www/SIGAP/mediamtx.yml#L73) untuk sesi tanpa penonton WebRTC).
- Tabel `cctv_probe_telemetry` merekam `verify_duration_ms`, `source_lifetime_ms`, `bytes_verify`, `bytes_total_session` (termasuk ekor `closeAfter`), dan `source_closed_confirmed` (`Available == false`). Bila sumber belum tertutup setelah batas `closeAfter + 5s` tanpa penonton, sistem memaksa reset path dan mencatat peringatan.

---

## 3. Siklus Insiden, Tiket Bersama, Penggabungan, Eskalasi Per Kamera, & Pengingat

### 3.1 Mesin Status Insiden Induk & Hasil Per Kamera Anggota

- **Status Insiden Induk (`cctv_incidents.status`)**:
  - `open` $\rightarrow$ `in_progress` (ditugaskan teknisi) $\rightarrow$ `pending_verification` (seluruh anggota aktif berstatus `recovered` dengan bukti segar) $\rightarrow$ `resolved` (ditutup Admin).
  - Tiket aktif (`open`/`in_progress`/`pending_verification`) yang digabung manual oleh Admin ke tiket induk berubah menjadi **`merged`** (`merged_into_incident_id`).
- **Kategori Penutupan Induk (`resolution_type`)**:
  - `self_recovered` (**Pulih Mandiri** — ditutup tanpa teknisi setelah stabil $\ge 5$ menit).
  - `repaired_verified` (**Pulih Setelah Penanganan** — seluruh anggota pulih).
  - `escalated_replacement` (**Dieskalasikan — Penggantian Perangkat** — seluruh anggota dinonaktifkan untuk ganti unit).
  - `partial_recovery_and_escalation` (**Campuran Pulih & Eskalasi Penggantian** — sebagian kamera pulih terverifikasi dan $\ge 1$ kamera diekskalasi untuk penggantian unit; lihat Koreksi #5).
- **Hasil Akhir Per Kamera Anggota (`cctv_incident_cameras.member_outcome`)**:
  Setiap baris anggota kamera menyimpan hasil akhirnya sendiri (`pending`, `self_recovered`, `recovered_verified`, `escalated_replacement`, atau `merged_transferred`) sehingga laporan tingkat kamera tetap 100% presisi walaupun tiket bersama memiliki hasil berbeda antar-kamera.

### 3.2 Gangguan yang Sama & Bukti Tiket Bersama Otomatis

- Selama kamera memiliki `is_active_member = true`, gangguan berulang memperbarui tiket terbuka tersebut.
- Tiket bersama otomatis (`scope_type = 'shared_nvr'`) hanya dibuat bila pemantau sehat, $\ge 2$ kamera pada `site_id` dan `ip_nvr` yang sama terkonfirmasi `offline` dalam jendela korelasi `5s`, dan `ProbeRTSPHost` membuktikan NVR tidak merespons (tanpa ada kanal lain pada `ip_nvr` tersebut yang `live`). Jika penyebab bersama belum terbukti, dibuat tiket per kamera dan pemberitahuan digabung di outbox.

### 3.3 Penggabungan Manual “Gabungkan ke Tiket Bersama” (`POST /api/incidents/:target_id/merge`)

- Hanya untuk Admin pada `site_id` yang sama, wajib menyertakan `merge_reason`.
- Dalam satu transaksi ber-urutan lock UUID:
  1. Pada tiket asal: set anggota `is_active_member = false`, `member_outcome = 'merged_transferred'`, lalu set tiket asal `status = 'merged'` dan `merged_into_incident_id = target_id` (waktu observasi awal, nomor tiket awal, dan histori event tetap utuh).
  2. Pada tiket induk: tambahkan anggota aktif baru (`is_active_member = true`) dengan menyalin penuh `first_failed_observed_at`, `confirmed_offline_at`, `connection_state`, `last_observed_at`, `recovery_started_at`, dan `consecutive_live_count` dari tiket asal.
  3. Hitung ulang status tiket induk (`pending_verification` hanya jika seluruh anggota aktif gabungan sudah `recovered`), batalkan outbox `pending` milik tiket asal, catat audit, dan antrekan 1 pembaruan Telegram penggabungan.

### 3.4 Pemulihan Stabil 5 Menit, Pemulihan Sebagian, & Penutupan “Pulih Mandiri”

- **Syarat `recovered`**: Durasi $\ge 300\text{ detik}$ sejak `recovery_started_at`, minimal `6` sampel `live` berturut-turut, dan jarak antar-sampel berturut-turut $\le \mathbf{105\text{ detik}}$ (`MaxEvidenceGap`). Sampel `offline` atau `unknown` memulai ulang hitungan dari nol.
- **Pemulihan Sebagian**: Jika baru sebagian anggota aktif pada tiket bersama yang `recovered`, tiket tetap `open`/`in_progress`, penutupan ditolak (`409`), dan Telegram menerima pembaruan daftar kamera yang masih offline (debounce `30s`).
- **Pulih Mandiri**: Saat seluruh anggota aktif `recovered` (bukti segar $\le 105\text{s}$) dan `assigned_to IS NULL`, Admin dapat menutup tiket dengan `resolution_type = 'self_recovered'` (`member_outcome = 'self_recovered'`) tanpa membuat `MaintenanceLog` maupun checklist.

### 3.5 Tugas Perbaikan Bersama NVR/Switch vs Tugas Individual Kamera

- **Tiket Bersama**: Penugasan Admin secara standar membuat **1 `MaintenanceLog` perbaikan bersama** (`incident_id = incident.id`, `work_scope = 'shared_infrastructure'`) tempat teknisi mengunggah bukti foto & catatan pekerjaan pada NVR/switch tanpa dipaksa mengisi checklist fisik tiang per kamera. Bila kamera tertentu membutuhkan pemeriksaan fisik di lapangan, Admin dapat menambahkan tugas individual (`work_scope = 'camera_field_check'`) yang ditautkan ke `cctv_incident_cameras.field_maintenance_log_id`.
- **Maintenance Rutin (`incident_id IS NULL`, `work_scope = 'routine'`)**: Sama sekali tidak diubah dan tetap selesai langsung saat teknisi menyelesaikan checklist tanpa menunggu verifikasi Admin.

### 3.6 Eskalasi Penggantian Perangkat Per Kamera (Koreksi #5)

Endpoint `POST /api/incidents/:id/escalate-replacement` menerima daftar satu atau lebih anggota kamera yang terbukti mengalami kerusakan berat pada insiden tersebut, di mana **setiap kamera yang diekskalasi** wajib memiliki:
1. `damage_findings` (uraian temuan kerusakan kamera tersebut), dan
2. Tugas penggantian unit tertaut (`replacement_maintenance_log_id` baru atau eksisting dengan `work_scope = 'unit_replacement'` yang wajib memiliki `technician_id` penanggung jawab di site tersebut dan status tindak lanjut aktif).

Dalam 1 transaksi atomik:
- Setiap kamera yang diekskalasi diubah pada master `cctvs` menjadi `status = 'nonaktif'` dan `monitoring_enabled = false` (path stream dilepas dari MediaMTX).
- Baris kamera pada `cctv_incident_cameras` diperbarui: `connection_state = 'escalated'`, `member_outcome = 'escalated_replacement'`, `damage_findings`, `replacement_maintenance_log_id`, dan `is_active_member = false`.
- Evaluasi ulang insiden induk:
  - Jika seluruh anggota insiden telah diekskalasi (`0` anggota aktif tersisa dan tidak ada yang pulih): insiden induk ditutup otomatis dengan `status = 'resolved'` dan `resolution_type = 'escalated_replacement'`.
  - Jika masih ada anggota aktif lain (mis. 3 kamera lain pada tiket bersama): insiden induk tetap berjalan untuk memantau ke-3 kamera sisanya; begitu ke-3 kamera sisanya `recovered` dan diverifikasi Admin, insiden induk ditutup dengan `resolution_type = 'partial_recovery_and_escalation'` (sementara ke-3 kamera tersebut dicatat `member_outcome = 'recovered_verified'` dan ke-2 kamera rusak tetap tercatat `member_outcome = 'escalated_replacement'` beserta tautan tugas penggantian masing-masing).
- Setelah unit fisik diganti, Admin menjalankan `POST /api/cctv/:id/reactivate-verify` yang memverifikasi stream `live` ($\Delta\text{BytesReceived} > 0$) sebelum mengembalikan kamera ke `aktif` & `monitoring_enabled = true`.

### 3.7 Pengingat Penugasan 30 Menit & Eskalasi 4 Jam (Koreksi #3)

- **Pengingat Penugasan 30 Menit**: Berlaku untuk insiden `status = 'open'`, `assigned_to IS NULL`, masih ada anggota aktif belum `recovered`, `assignment_reminder_queued_at IS NULL`, dan `now >= sla_clock_start_at + 30 menit`. Sistem menyisipkan tepat 1 entri `INCIDENT_ASSIGN_REMINDER` ke `telegram_outbox` (ke grup operasional & grup Admin + in-app Admin) dan mengisi `assignment_reminder_queued_at = now` serta `assignment_reminder_outbox_id`.
- **Eskalasi Belum Pulih 4 Jam**: Berlaku untuk insiden `status IN ('open', 'in_progress')`, `unrecovered_escalation_queued_at IS NULL`, dan `now >= opened_at + CCTV_INCIDENT_ESCALATION_MINUTES` (default `240` menit). Sistem menyisipkan tepat 1 entri `INCIDENT_UNRECOVERED_ESCALATION` khusus Admin (`admin_group` + in-app + Web Push) dan mengisi `unrecovered_escalation_queued_at = now` serta `unrecovered_escalation_outbox_id`.
- **Penjaminan Database**: Indeks unik parsial `CREATE UNIQUE INDEX idx_uq_outbox_once_per_incident ON telegram_outbox(incident_id, event_type) WHERE event_type IN ('INCIDENT_ASSIGN_REMINDER', 'INCIDENT_UNRECOVERED_ESCALATION')` memastikan mustahil tercipta lebih dari satu entri antrean pengingat/eskalasi per insiden, sedangkan status pengiriman aktualnya (`pending`, `sent`, `ambiguous`, `failed`) dibaca dari baris outbox tertaut.

---

## 4. Model Antrean Gabungan & Perhitungan Anggaran Waktu (Koreksi #2)

### 4.1 *Unified NVR Session Gate* (Batas Sesi Gabungan Termasuk Ekor `closeAfter`)

Seluruh antrean pemeriksaan yang menuju `ip_nvr` yang sama (urutan prioritas pengambilan: **1. `Confirm Queue`** $\rightarrow$ **2. `Post-Maintenance Refresh Queue`** $\rightarrow$ **3. `Main Cycle Queue`**) wajib melewati **satu pengatur terpadu** dengan 2 kendali simultan:

1. **Global Trigger Pacer Tunggal (`T_pace = 1,25 detik`)**: Jarak minimum antar-peluncuran pemeriksaan baru lintas seluruh antrean adalah `1,25 detik` (maksimal `0,8` peluncuran/detik gabungan), didukung pool worker $W_{\text{pool}} = 10$ dan *Per-Camera In-Flight Guard*.
2. **Upstream Lease Semaphore (`MaxTotalUpstreamSlots = 10`, Mencakup Sumber yang Masih Terbuka)**:
   - Setiap pemeriksaan mengakuisisi 1 slot sebelum menyentuh MediaMTX/NVR.
   - Jika pemeriksaan gagal/timeout (`T_attempt,max = 9,0s`: pra-cek `0,5s` + `TriggerHLS` `1,5s` + poll & cek byte `6,5s` + tulis DB `0,5s`), sumber tidak pernah `Ready` sehingga slot langsung dilepas saat pemeriksaan berakhir. Puncak slot gagal simultan dengan pacer `1,25s` adalah $\lceil 9,0 / 1,25 \rceil = \mathbf{8\text{ slot}}$ ($\le 10$).
   - Jika pemeriksaan **berhasil membuka sesi baru (`SourceReady == true`)**, slot **tidak langsung dilepas** saat fungsi verifikasi selesai (`~1,5s`), melainkan dipertahankan oleh *Tail Lease Timer* selama masa **`sourceOnDemandCloseAfter = 5,0 detik`** (total masa tahan slot `~6,5s`, atau dilepas lebih cepat begitu `Available == false`). Puncak sesi RTSP hidup gabungan di NVR (pemeriksaan sedang berjalan **ditambah** sumber yang masih terbuka menunggu `closeAfter`) terikat ketat pada $\lceil 6,5 / 1,25 \rceil = \mathbf{6\text{ sub-stream}}$ (dan tidak pernah diizinkan melebihi `10` slot, bahkan jika ada beberapa kamera lambat menutup sumber).

### 4.2 Estimasi Waktu Kasus Terburuk — Gangguan di Luar Jadwal (35 Kamera Gagal Serentak)

> **Pernyataan Asumsi Model**: Perhitungan di bawah adalah **estimasi model antrean perangkat lunak** untuk kapasitas $N = 35$ kamera pada satu NVR, periode siklus tetap $T_{\text{cycle}} = 60,0\text{s}$, pacer tunggal $T_{\text{pace}} = 1,25\text{s}$, jeda konfirmasi $T_{\text{backoff}} = 10,0\text{s}$, dan 1 kali timeout pengiriman HTTP Telegram (`5,0s`) yang masuk status **`ambiguous`** dengan **jeda retry konsisten `10,0s`** (selaras Bagian 5.4) diikuti percobaan kedua sukses (`5,0s`). Klaim pemenuhan target 5 menit terhadap perangkat nyata tetap menunggu pembuktian uji staging (Bagian 8.2).

| Tahap | Komponen Waktu Kasus Terburuk | Rumus / Asumsi Batas Atas | Batas Atas Konservatif (Tanpa Overlap Fase) | Simulasi Antrean Pacer Tunggal (Dengan Overlap) |
|---|---|---|---:|---:|
| **1. Tunggu Siklus** | Kamera putus fisik tepat sesaat setelah selesai diperiksa `live` pada siklus sebelumnya | Maksimal $1 \times T_{\text{cycle}}$ sebelum observasi gagal pertama dimulai | `60,00 dtk` | `60,00 dtk` |
| **2. Pemeriksaan #1** | Antrean pacer tunggal (`1,25s`) untuk 35 kamera + timeout penuh `9,0s` | $34 \times 1,25\text{s} + 9,0\text{s}$ | `51,50 dtk` | *(tergabung di baris 4)* |
| **3. Jeda Konfirmasi** | Backoff tenang sebelum kamera boleh diambil `Confirm Queue` | $T_{\text{backoff}} = 10,0\text{s}$ | `10,00 dtk` | *(terpenuhi sebelum giliran ke-70)* |
| **4. Konfirmasi #2** | Eksekusi 35 konfirmasi di pacer tunggal yang sama (`1,25s`) + timeout `9,0s` | Konservatif: $34 \times 1,25\text{s} + 9,0\text{s}$. Overlap nyata: total $35 + 35 = 70$ peluncuran tak terputus di pacer `1,25s` sejak $t=60\text{s}$ $\rightarrow (70-1)\times 1,25\text{s} + 9,0\text{s} = 95,25\text{s}$ untuk tahap 2–4 gabungan | `51,50 dtk` | `95,25 dtk` *(Tahap 2–4 gabungan)* |
| **5. Korelasi & DB** | Jendela korelasi NVR (`5,0s`, termasuk `ProbeRTSPHost` `2,0s`) + transaksi DB (`2,0s`) | $T_{\text{corr}} + T_{\text{tx,max}}$ | `7,00 dtk` | `7,00 dtk` |
| **6. Batching Outbox** | Jendela penggabungan pesan (`8,0s`) + poll worker outbox (`2,0s`) | $T_{\text{batch}} + T_{\text{poll}}$ | `10,00 dtk` | `10,00 dtk` |
| **7. Kirim & Retry Telegram** | HTTP #1 timeout $\rightarrow$ status `ambiguous` (`5,0s`) + **jeda retry `ambiguous` konsisten Bagian 5.4 (`10,0s`)** + HTTP #2 sukses (`5,0s`) | $5,0\text{s} + 10,0\text{s} + 5,0\text{s}$ | `20,00 dtk` | `20,00 dtk` |
| **TOTAL ESTIMASI** | **Sejak Putus Fisik s.d. Konfirmasi API Telegram** | **Target Kebutuhan Produk: $\le \mathbf{300,00\text{ dtk}}$** | **`210,00 dtk` (3m 30,0d)** | **`192,25 dtk` (3m 12,25d)** |

*(Catatan: Di produksi ketika waktu putus fisik Tahap 1 tidak diketahui, tenggat `sla_deadline_at` dihitung sejak `first_failed_observed_at`, sedangkan di staging diukur sejak waktu gangguan disuntikkan termasuk Tahap 1).*

### 4.3 Estimasi Waktu Pasca-Maintenance (`sla_clock_start_at = effective_ends_at`)

1. **Kasus 1 Kamera Berakhir Jendela Maintenance**:
   - *Bila bukti gagal terkonfirmasi terakhir selama jendela masih segar ($\le 75\text{s}$)*: Poll kedaluwarsa (`5,0s`) + korelasi & transaksi DB (`7,0s`) + batching & poll outbox (`10,0s`) + pengiriman Telegram dengan 1x retry `ambiguous` (`20,0s`) = **`42,0 detik`**.
   - *Bila memerlukan 1 verifikasi penyegaran cepat (tanpa antrean)*: ditambah maks `9,0s` (dan tunggu maks 1 slot pacer `1,25s`) = **`52,25 detik`**.
2. **Kasus Beban Puncak — Seluruh 35 Kamera Berakhir Jendela Maintenance Bersamaan**:
   Apabila seluruh 35 kamera dijadwalkan selesai maintenance pada detik yang sama, semuanya masih offline, dan seluruhnya memerlukan 1 verifikasi penyegaran melalui *Unified NVR Session Gate* sementara slot sedang penuh:
   - Poll kedaluwarsa jendela: maks `5,00s`.
   - Tunggu pelepasan slot yang sedang berjalan (`maks 9,00s`) + antrean pacer tunggal `1,25s` untuk 35 kamera ($34 \times 1,25\text{s} + 9,0\text{s} = 51,50\text{s}$): maks **`60,50s`**.
   - Korelasi & transaksi DB (`7,00s`) + batching & poll outbox (`10,00s`) + kirim Telegram termasuk 1x retry `ambiguous` `10s` (`20,00s`): **`37,00s`**.
   - **Total Estimasi Beban Puncak 35 Jendela Berakhir Serentak**: **`102,50 detik`** ($\le 300,0\text{ detik}$). Seluruh angka ini diperlakukan sebagai estimasi model antrean sampai diuji pada pengujian beban.

---

## 5. Pencegahan Tiket Ganda, Restart, Batching, & Penanganan Ambiguitas / SLA

### 5.1 Pencegahan Tiket Ganda & Pemulihan Restart
- **3 Lapis Anti-Tiket Ganda**: (1) *Per-Camera In-Flight Guard*, (2) `pg_advisory_xact_lock` per site + `SELECT ... FOR UPDATE` pada insiden induk + unique index `idx_uq_open_shared_incident` pada `(site_id, correlation_key) WHERE status NOT IN ('resolved','merged')`, dan (3) partial unique index PostgreSQL `idx_camera_one_active_incident ON cctv_incident_cameras(cctv_id) WHERE is_active_member = true`.
- **Restart Recovery**: `ResetObservations()` me-reset pengamatan `recovering`/`recovered` yang belum ditutup menjadi `unknown` (`recovery_started_at = NULL`, `consecutive_live_count = 0`) dan menurunkan `pending_verification` menjadi `in_progress` (bila `assigned_to IS NOT NULL`) atau `open` (bila `assigned_to IS NULL`). Baris outbox `sending` yang `lease_until`-nya kedaluwarsa saat proses crash otomatis dipindahkan ke status **`ambiguous`**.

### 5.2 Penggabungan Pesan (*Batching*) & Rate-Limit Grup Telegram
- Pesan `CCTV_OFFLINE` pada grup yang sama digabung selama jendela `coalesce_until = created_at + 8s` (maks `4.000` karakter per pesan) dan dikirim melalui *per-chat token bucket* minimal **`3,2 detik` antar-pesan** ($\le 18\text{ pesan/menit}$), aman dari batas 20 pesan/menit Telegram.

### 5.3 Aturan Retry, Pengiriman Ambigu, & Pencatatan Pelanggaran SLA (Koreksi #2 & #3)

| Kondisi Pengiriman HTTP Telegram | Klasifikasi | Status Outbox | Jadwal Retry & Perilaku |
|---|---|---|---|
| Gagal DNS atau gagal dial TCP/TLS sebelum request terkirim | Pasti belum terkirim | `pending` | Backoff `5s`, `15s`, `30s`, `60s`, `120s` (maks 6 percobaan total); tidak membuat tiket baru. |
| Respons HTTP valid `429` (`retry_after`) atau `500/502/503/504` | Ditolak sementara oleh server Telegram | `pending` | Mengikuti `max(retry_after, backoff)` (maks 6 percobaan total). |
| Respons HTTP `400` / `401` / `403` atau konfigurasi grup/token tidak valid | Gagal permanen | `failed` | Mengisi `failed_at = now`, menghentikan retry otomatis, dan mengirim notifikasi in-app + Web Push ke Admin. |
| Request HTTP sudah dikirim ke soket lalu **read timeout (`5,0s`)**, **connection reset**, JSON rusak, atau **lease crash habis** | **Ambigu** (mungkin sudah tampil di grup, mungkin belum) | `ambiguous` | Maksimal **2 percobaan ulang terkendali** dengan jeda tetap **`10,0s`** (retry ke-1) dan **`30,0s`** (retry ke-2), menempelkan catatan eksplisit `[Pengiriman ulang karena konfirmasi jaringan sebelumnya terputus — mengacu pada Tiket INC-... yang sama]`. **Catatan kejujuran**: retry pada status `ambiguous` tetap dapat menghasilkan pesan Telegram duplikat bila percobaan pertama sebenarnya telah diterima server Telegram. Jika setelah 2 retry masih ambigu, retry otomatis dihentikan (tersedia tombol Kirim Ulang Manual bagi Admin). |

**Aturan Pencatatan Pelanggaran SLA (`sla_breached`)**:
- Baris outbox yang terikat target waktu memiliki `sla_deadline_at = sla_clock_start_at + 5 menit`.
- Worker memantau setiap baris dengan `sla_deadline_at IS NOT NULL AND sla_breached = false`: **segera setelah `now > sla_deadline_at` sementara `sent_at IS NULL`** (baik pesan masih `pending`, `sending`, `ambiguous`, maupun sudah berhenti di `failed`), sistem langsung menetapkan **`sla_breached = true` dan `sla_breached_at = now`**, serta memberi tahu Admin. Jika pesan baru berhasil terkonfirmasi sesudah tenggat (`sent_at > sla_deadline_at`), `sla_breached` juga tetap bernilai `true`.

---

## 6. Rincian Perubahan Skema Database (Migrasi 3-Langkah) & File Kode

### 6.1 Rancangan Migrasi `000034_enhance_cctv_incidents_and_telemetry.{up,down}.sql`

Untuk menjamin kompatibilitas penuh terhadap database yang sudah memiliki baris tiket dari migrasi `000031_cctv_incidents` (Koreksi #4), migrasi ditulis dalam **3 langkah transaksional**:

1. **Langkah 1 — Penambahan Tabel & Kolom Baru (Nullable untuk Kolom Wajib Baru pada Tabel Eksisting)**:
   - `cctvs`: tambah `monitoring_enabled BOOLEAN NOT NULL DEFAULT true` (di-update `false` untuk baris dengan `status = 'nonaktif'`).
   - `cctv_planned_outages` & `cctv_planned_outage_cameras`: tabel baru jendela penghentian terencana beserta kolom `suppressed_first_failed_at`, `suppressed_confirmed_offline_at`, `post_window_retry_count`, dan `post_window_next_retry_at` agar bukti `unknown` saat/pasca-jendela tetap mempertahankan evaluasi dalam status tertunda dengan retry berjeda terkendali.
   - `cctv_incidents`: perluas `CHECK(status)` mencakup `'merged'`; tambah kolom nullable sementara `ticket_number VARCHAR(32)`, `first_failed_observed_at TIMESTAMPTZ`, `confirmed_offline_at TIMESTAMPTZ`, `sla_clock_start_at TIMESTAMPTZ`, `sla_clock_basis VARCHAR(32)`, `timestamp_source VARCHAR(32)`, serta kolom `scope_type`, `origin_type`, `correlation_key`, `resolution_type` (`self_recovered`, `repaired_verified`, `escalated_replacement`, `partial_recovery_and_escalation`), `merged_into_incident_id`, `merged_at`, `merged_by`, `merge_reason`, `assignment_reminder_queued_at`, `assignment_reminder_outbox_id`, `unrecovered_escalation_queued_at`, `unrecovered_escalation_outbox_id`.
   - `cctv_incident_cameras` (Koreksi #5): perluas `CHECK(connection_state)` mencakup `'escalated'`; tambah `camera_code VARCHAR(50)`, `location_area VARCHAR(255)`, `first_failed_observed_at TIMESTAMPTZ`, `confirmed_offline_at TIMESTAMPTZ`, `recovered_at TIMESTAMPTZ`, `member_outcome VARCHAR(32) NOT NULL DEFAULT 'pending'`, `damage_findings TEXT NOT NULL DEFAULT ''`, `field_maintenance_log_id UUID NULL REFERENCES maintenance_logs(id)`, `replacement_maintenance_log_id UUID NULL REFERENCES maintenance_logs(id)`, `outcome_recorded_at TIMESTAMPTZ NULL`, `outcome_recorded_by UUID NULL REFERENCES users(id)`, dan `last_evidence JSONB`.
   - `cctv_incident_events`: tabel riwayat event & bukti transisi per insiden.
   - `maintenance_logs`: tambah `incident_id UUID NULL REFERENCES cctv_incidents(id) ON DELETE SET NULL` dan `work_scope VARCHAR(32) NOT NULL DEFAULT 'routine'`.
   - `monitor_health_episodes`: tabel episode gangguan pemantau (`unknown`) untuk deduplikasi 1 notifikasi per kejadian.
   - `telegram_outbox`: ubah `incident_id` menjadi nullable, tambah `monitor_episode_id UUID NULL`, `target_channel VARCHAR(24) NOT NULL DEFAULT 'ops_group'`, `coalesce_until TIMESTAMPTZ`, `batch_id UUID NULL`, `last_attempt_at TIMESTAMPTZ NULL`, `failed_at TIMESTAMPTZ NULL`, `sla_deadline_at TIMESTAMPTZ NULL`, `sla_breached BOOLEAN NOT NULL DEFAULT false`, `sla_breached_at TIMESTAMPTZ NULL`, serta unique index parsial `idx_uq_outbox_once_per_incident` untuk pengingat 30 menit & eskalasi 4 jam.
   - `cctv_probe_telemetry`: tabel metrik durasi sesi, `bytes_verify`, `bytes_total_session`, `source_closed_confirmed`, `forced_close`, serta kolom bukti penutupan terpisah (`remove_path_status`, `remove_path_error`, `reregister_status`, `reregister_error`, `close_verify_status`, `close_unconfirmed_reason`) sehingga `source_closed_confirmed = true` hanya disetel bila pemeriksaan ulang MediaMTX membuktikan sumber telah berhenti.
2. **Langkah 2 — *Deterministic Backfill* untuk Baris Tiket Historis `000031`**:
   - Mengisi `ticket_number` pada baris lama yang masih `NULL` secara deterministik (`ORDER BY opened_at ASC, id ASC`) dengan format unik `'INC-LEGACY-' || to_char(opened_at AT TIME ZONE 'Asia/Makassar', 'YYYYMMDD') || '-' || lpad(seq::text, 4, '0')`.
   - Mengisi `first_failed_observed_at = opened_at`, `confirmed_offline_at = COALESCE(earliest_member_observed, opened_at)`, `sla_clock_start_at = opened_at`, `sla_clock_basis = 'legacy_opened_at'`, dan **`timestamp_source = 'legacy_backfill_opened_at'`** agar laporan selalu jujur menandai bahwa timestamp tiket lama berasal dari waktu pembuatan tiket, bukan waktu gagal pertama teramati.
   - Mengisi `member_outcome = 'recovered_verified'` untuk anggota dari tiket lama yang sudah `resolved`.
   - Mengisi `coalesce_until = created_at` untuk baris `telegram_outbox` lama.
3. **Langkah 3 — Pemasangan Constraint `NOT NULL` & `UNIQUE`**:
   - Menetapkan `SET NOT NULL` pada `ticket_number`, `first_failed_observed_at`, `confirmed_offline_at`, `sla_clock_start_at`, `sla_clock_basis`, `timestamp_source`, dan `telegram_outbox.coalesce_until`, serta memasang `UNIQUE INDEX` pada `cctv_incidents(ticket_number)` dan partial unique index korelasi `idx_uq_open_shared_incident`.

### 6.2 Pemetaan File Kode Implementasi

- **Migrasi & Model**: `backend/migrations/000034_enhance_cctv_incidents_and_telemetry.{up,down}.sql`, `backend/models/{cctv.go,setup.go,notification.go}`, `backend/incidents/models.go`.
- **Verifier & Gate Terpadu**: `backend/handlers/stream_verify.go`, `backend/handlers/stream_verify_gate.go`, `backend/handlers/cctv_monitoring.go` (*Unified NVR Session Gate* mencakup pemeriksaan terjadwal, verifikasi manual `VerifyStream`, dan reaktivasi `ReactivateAndVerifyCCTV`, pacer `1,25s`, tail lease `closeAfter`, cek $\Delta\text{BytesReceived}$, episode pemantau, verifikasi penutupan sumber GSM berbasis bukti MediaMTX, endpoint heartbeat), `backend/handlers/stream.go`, `mediamtx.yml`.
- **Domain Insiden & Outbox**: `backend/incidents/{state.go,store.go,planned_outage.go,telegram.go,batcher.go}`.
- **Handler, Rute, & Frontend**: `backend/handlers/{incidents.go,planned_outage.go,maintenance.go,cctv.go}`, `backend/routes/incidents.go`, `frontend/src/app/incidents/page.tsx`, `frontend/src/app/cctv/page.tsx`, `frontend/src/components/DashboardLayout.tsx`.

---

## 7. Rencana Pengujian & Kriteria Penerimaan

### 7.1 Bagian A — Pengujian Tiruan / Simulasi Otomatis di CI (Wajib Lulus Sebelum PR)

1. **Migrasi Database Berisi Data Lama `000031` (Koreksi #4)**:
   - Menjalankan migrasi `000031` $\rightarrow$ menyisipkan sampel tiket lama (`open`, `pending_verification`, `resolved`) beserta anggota & outbox $\rightarrow$ menjalankan migrasi `000034` (`up` lalu `down` lalu `up`): wajib sukses tanpa pelanggaran `NOT NULL`/`UNIQUE`, nomor tiket `INC-LEGACY-...` terbentuk deterministik, dan `timestamp_source = 'legacy_backfill_opened_at'` tercatat akurat.
2. **Deteksi Offline, *Unified NVR Session Gate*, & Pemisahan Semantik Waktu (Koreksi #1 & #2)**:
   - Uji `first_failed_observed_at` vs `confirmed_offline_at` tercatat terpisah; uji 35 kamera gagal serentak melalui pacer tunggal `1,25s` + retry `ambiguous` `10s` selesai dalam estimasi model ($\le 210\text{s}$ simulasi); uji *Tail Lease Timer* menahan slot selama `closeAfter = 5s` sehingga total sesi aktif + ekor tidak pernah melampaui `MaxTotalUpstreamSlots = 10`; uji stream beku ($\Delta\text{BytesReceived} = 0$) terdeteksi gagal; uji kanal gagal saat `ProbeRTSPHost` sukses tetap terdeteksi offline.
3. **Jendela Pemeliharaan Terjadwal & Beban Berakhir Bersamaan (Keputusan #1 & Koreksi #1, #2)**:
   - Tugas rutin `ongoing` biasa tidak menahan tiket; kamera dalam jendela `cctv_planned_outages` menampilkan **Sedang pemeliharaan terjadwal**, menahan tiket selama aktif, dan saat berakhir (baik 1 jendela maupun 35 jendela serentak) langsung memproses pembukaan tiket tanpa mengulang 2 pemeriksaan dari awal dengan `sla_clock_start_at = effective_ends_at` serta menyimpan waktu observasi asli secara terpisah.
4. **Pemulihan Sebagian, Flapping, Restart, & “Pulih Mandiri” (Keputusan #2)**:
   - Pemulihan sebagian mempertahankan tiket tetap terbuka dan menolak `resolve`; flapping <5 menit dan celah $> 105\text{s}$ me-reset hitungan; restart mengembalikan `pending_verification` tanpa teknisi ke `open`; stabil $\ge 5$ menit ($\ge 6$ sampel) mengizinkan penutupan `self_recovered` tanpa `MaintenanceLog`.
5. **Penggabungan Manual Tiket (`merged` — Keputusan #3)**:
   - `POST /api/incidents/:id/merge` memindahkan keanggotaan aktif tanpa bentrok `idx_camera_one_active_incident`, mempertahankan waktu observasi awal & nomor tiket sumber, dan menolak penggabungan lintas site.
6. **Tugas Perbaikan Bersama & Multi-Kamera Campuran Pulih + $>1$ Eskalasi Penggantian (Keputusan #4, #5, & Koreksi #5)**:
   - Uji tiket bersama dengan 4 kamera: **2 kamera pulih stabil** dan **2 kamera berbeda diekskalasi untuk penggantian unit** masing-masing ke `replacement_maintenance_log_id` miliknya sendiri $\rightarrow$ kedua CCTV rusak menjadi `nonaktif` (`monitoring_enabled = false`), tiket induk ditutup dengan `resolution_type = 'partial_recovery_and_escalation'`, dan query laporan tetap menampilkan secara akurat 2 kamera `recovered_verified` serta 2 kamera `escalated_replacement` beserta tugas tindak lanjut masing-masing meski tiket induk sudah ditutup.
7. **Episode Gangguan Pemantau ke Grup Admin (Keputusan #6)**:
   - Saat MediaMTX gagal untuk 20 kamera, 0 tiket kamera offline dibuat dan tepat **1 notifikasi episode** masuk ke In-App, Web Push, dan `admin_group` (bukan 20 pesan).
8. **Outbox Batching, Ambiguitas Duplikat, Pelanggaran SLA, & Pengingat Sekali per Insiden (Keputusan #7 & Koreksi #3)**:
   - Uji 25 tiket serentak digabung & patuh $< 20$ pesan/menit; uji status `ambiguous` melakukan retry berjeda `10s` dengan teks peringatan duplikat tanpa menggandakan tiket; uji **`sla_breached = true` tercatat tepat saat tenggat 5 menit terlewati tanpa `sent_at`** (termasuk saat pesan gagal permanen `401/403` atau masih mengantre); uji tepat **1 event dan 1 entri outbox** pengingat 30 menit (`assignment_reminder_queued_at`) serta eskalasi 4 jam (`unrecovered_escalation_queued_at`) per insiden.
9. **Regresi Maintenance Rutin & Isolasi Multi-Site**:
   - Seluruh pengujian eksisting `maintenance_logs`, jadwal berulang, PDF/CSV, dan isolasi `SiteContext()` tetap lulus 100% tanpa perubahan perilaku pada tugas rutin.

### 7.2 Bagian B — Pengujian Staging & Perangkat Nyata Terkendali (Pasca-Implementasi)

Dilaksanakan hanya pada lingkungan uji yang diotorisasi pengguna dengan grup Telegram uji terpisah:
1. Mengukur waktu end-to-end nyata dari **`physical_fault_at` (waktu gangguan disuntikkan)** sampai **pesan tampil di grup uji** pada kamera Metro-E & GSM (kasus terburuk sesaat setelah verifikasi `live` selesai).
2. Menguji beban *Unified NVR Session Gate* pada NVR fisik untuk memastikan ketiadaan `453 Not Enough Bandwidth` dan kepastian penutupan sumber RTSP (`source_closed_confirmed = true`).
3. Mengukur konsumsi data aktual GSM (`bytes_total_session` vs penghitung byte antarmuka jaringan container/host) selama periode uji 1 jam dan 24 jam.
4. Memverifikasi pemantau eksternal di luar server terhadap endpoint `/api/health` dan `/api/streams/monitor-heartbeat`.
