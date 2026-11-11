# Panduan Pengujian Manual — Monitoring CCTV, Tiket Gangguan Otomatis & Telegram (Revisi 3)

Dokumen ini merinci langkah pengujian manual end-to-end untuk memverifikasi seluruh aturan bisnis dan koreksi teknis pada Desain Teknis Revisi 3 tanpa menyentuh perangkat produksi atau grup Telegram operasional nyata sebelum tahap staging terkendali.

---

## 1. Mode Uji Cepat Tanpa Perangkat Fisik (`cmd/incident-demo`)

Untuk menguji alur siklus insiden, jendela pemeliharaan terjadwal, pulih mandiri, penggabungan tiket, eskalasi per kamera, pengingat/eskalasi, dan simulasi Telegram lokal (termasuk injeksi hasil ambigu/gagal):

1. Jalankan server demo terisolasi dari direktori `backend`:
   ```powershell
   go run ./cmd/incident-demo
   ```
2. Buka `http://127.0.0.1:8095` di browser.
3. Gunakan panel kontrol di halaman demo untuk mensimulasikan observasi `offline`, `live`, `unknown`, jendela pemeliharaan terjadwal, penugasan teknisi, penggabungan tiket, eskalasi penggantian perangkat, dan status pengiriman Telegram Mock tanpa memerlukan koneksi RTSP maupun token bot nyata.

---

## 2. Prasyarat Pengujian Terintegrasi Aplikasi Penuh (Lokal / Staging Terisolasi)

Pastikan variabel lingkungan berikut disetel pada `backend/.env` (native Go) atau `backend/.env.docker` (Docker Compose) pada environment uji terisolasi (bukan produksi):

```env
CCTV_INCIDENTS_ENABLED=true
CCTV_INCIDENT_INTERVAL_SECONDS=60
CCTV_MAX_UPSTREAM_SLOTS=10
CCTV_GSM_MAX_CONCURRENT=2
CCTV_VERIFY_PACE_MS=1250
CCTV_MANUAL_VERIFY_WAIT_TIMEOUT_MS=15000
CCTV_INCIDENT_REMINDER_MINUTES=30
CCTV_INCIDENT_ESCALATION_MINUTES=240
TELEGRAM_BOT_TOKEN=<token_bot_sandbox_uji>
TELEGRAM_SITE_CHATS_JSON={"<site_uuid_uji>":"<chat_id_grup_teknisi_uji>"}
TELEGRAM_SITE_ADMIN_CHATS_JSON={"<site_uuid_uji>":"<chat_id_grup_admin_uji>"}
TELEGRAM_ADMIN_CHAT_ID=<chat_id_grup_admin_fallback>
```

Catatan rute tujuan Telegram:
- Routing grup dibaca backend dari variabel env di atas (harus ID grup Telegram negatif, mis. `-100...`).
- Menu **Tiket Gangguan -> Tab Outbox Telegram, Pemantau & GSM** menampilkan ringkasan telemetri GSM, episode gangguan pemantau, dan antrean outbox Telegram (`ops_group` dan `admin_group`).

---

## 3. Skenario Uji Fungsional Wajib

### Skenario A — Deteksi Offline, Tiket Otomatis, & Penggabungan Pesan Telegram
1. Pastikan kamera uji berstatus **Aktif** (`aktif`) dan checkbox **Masuk Cakupan Monitoring Otomatis** (`monitoring_enabled`) tercentang di halaman **Aset CCTV (`/cctv`)**.
2. Putuskan sumber stream RTSP kamera uji (atau matikan simulator RTSP).
3. Verifikasi bahwa setelah kegagalan terkonfirmasi (2 pemeriksaan gagal berturut-turut melalui `ConfirmQueue` dengan jeda ulang 10 detik):
   - Satu tiket baru (`INC-YYYYMMDD-XXXX`) terbuka di `/incidents`.
   - Waktu **Gagal Pertama Teramati** (`first_failed_observed_at`) dan **Terkonfirmasi Offline** (`confirmed_offline_at`) tercatat terpisah; sumber waktu berlabel `live_observation`.
   - Satu baris outbox `CCTV_OFFLINE` masuk ke grup `ops_group` site terkait; bila beberapa kamera gagal bersamaan pada grup yang sama dalam jendela penggabungan 8 detik (`coalesce_until`), notifikasi digabung ke dalam satu pesan ringkas tetapi tiket tetap terpisah per kamera.
   - Jika `>= 2` kamera pada `ip_nvr` yang sama gagal bersamaan **dan** probe host RTSP NVR (`ip_nvr:554`) terbukti tidak merespons, sistem membuat 1 tiket bersama (`scope_type = shared_nvr`).

### Skenario B — Jendela Pemeliharaan Terjadwal & Evaluasi Akhir Jendela (Keputusan #1 & Koreksi #1)
1. Buat tugas maintenance rutin biasa berstatus `ongoing` pada kamera A **tanpa** membuat jendela pemeliharaan terjadwal -> putuskan kamera A -> pastikan tiket gangguan dan Telegram **tetap dibuat** (tugas rutin saja tidak menahan alarm).
2. Buka `/incidents` -> Tab **Pemeliharaan Terjadwal** -> buat jendela pemeliharaan untuk kamera B dengan alasan (misal *"Upgrade firmware switch"*) dan durasi 10 menit.
3. Buka `/cctv` -> pastikan baris dan detail kamera B menampilkan badge **Sedang pemeliharaan terjadwal** dan peringatan anomali *"Perlu cek fisik"* ditekan selama jendela aktif.
4. Putuskan kamera B selama jendela aktif -> pastikan **tidak ada** tiket baru maupun pesan Telegram yang dikirim, tetapi hasil pemeriksaan tetap dicatat pada jendela pemeliharaan (`suppressed_first_failed_at` & `suppressed_confirmed_offline_at`).
5. Akhiri jendela lebih awal (atau tunggu waktu berakhir) saat kamera B masih mati -> pastikan tiket langsung terbuka **tanpa menunggu 2 pemeriksaan ulang dari awal**, dengan basis jam SLA `planned_window_end` (`effective_ends_at`) sementara waktu gagal asli selama pemeliharaan tetap terlihat di detail tiket.

### Skenario C — Pemulihan Stabil 5 Menit, Flapping, & Pulih Mandiri (Keputusan #2)
1. Nyalakan kembali kamera pada tiket terbuka yang **belum ditugaskan** ke teknisi.
2. Putuskan kembali stream pada menit ke-3 -> pastikan status tiket kembali ke *Gangguan Berlangsung* (`open`) dan penghitung pemulihan direset ke 0.
3. Nyalakan kembali dan biarkan stabil minimal 5 menit (`>= 6` sampel `live` berturut-turut tanpa celah `> 105` detik) -> pastikan tiket berpindah ke **Menunggu Verifikasi Admin (`pending_verification`)** dan pesan `CCTV_RECOVERED` (atau `CCTV_PARTIAL_RECOVERY` pada tiket bersama jika baru sebagian kamera pulih) terkirim.
4. Klik aksi **Verifikasi & Tutup** pada tiket tersebut -> pilih hasil **Pulih Mandiri (`self_recovered`)** -> simpan.
5. Pastikan tiket tertutup (`resolved`), tidak ada tugas teknisi (`MaintenanceLog`) atau checklist yang dipaksa dibuat, dan laporan mencatat hasil `self_recovered`.

### Skenario D — Gabungkan Manual ke Tiket Bersama (Keputusan #3)
1. Siapkan dua tiket aktif terpisah pada site yang sama (Tiket Induk & Tiket Anak).
2. Pada Tiket Anak, klik **Gabungkan ke Tiket Bersama**, pilih Tiket Induk, dan isi alasan teknis korelasi gangguan.
3. Pastikan:
   - Status Tiket Anak berubah menjadi **Digabung (`merged`)** dan menunjuk ke nomor Tiket Induk.
   - Kamera dari Tiket Anak berpindah ke daftar kamera Tiket Induk lengkap dengan waktu observasi awal dan referensi nomor tiket asal (`source_ticket_number`).
   - Tidak terjadi benturan constraint kamera aktif ganda, dan Tiket Induk tetap terbuka sampai seluruh kamera gabungan memenuhi syarat pemulihan/eskalasi.

### Skenario E — Tugas Perbaikan Bersama vs Tugas Kamera Individual & Regresi Maintenance Rutin (Keputusan #4)
1. Tugaskan teknisi pada tiket bersama (`shared_nvr` atau hasil penggabungan `manual_merged`) -> pastikan sistem membuat **1 tugas perbaikan bersama (`work_scope = shared_infrastructure`)**.
2. Login sebagai Teknisi -> selesaikan tugas perbaikan bersama cukup dengan catatan perbaikan & foto bukti perangkat bersama (tanpa dipaksa mengisi checklist tiang fisik per kamera) -> pastikan penyelesaian tugas tidak menutup tiket insiden sebelum stream pulih 5 menit & diverifikasi Admin.
3. Dari detail tiket bersama di akun Admin, klik **Tugas Kamera** untuk 1 kamera spesifik yang membutuhkan pengecekan fisik di tiang -> pastikan tugas individual (`camera_field_check`) terbuat untuk kamera tersebut.
4. Uji regresi pemeliharaan rutin di `/schedules` dan `/field-maintenance` -> pastikan pembuatan jadwal rutin, pengisian checklist fisik wajib, foto bukti wajib, dan penutupan log rutin berjalan identik seperti sebelumnya.

### Skenario F — Eskalasi Penggantian Perangkat Multi-Kamera & Reaktivasi Terverifikasi (Keputusan #5 & Koreksi #5)
1. Buka tiket bersama yang memiliki 3 kamera: biarkan Kamera 1 pulih online stabil 5 menit, sedangkan Kamera 2 dan Kamera 3 mengalami kerusakan perangkat keras.
2. Klik **Eskalasi Penggantian**, centang Kamera 2 dan Kamera 3, isi temuan kerusakan (`damage_findings`), dan pilih teknisi penanggung jawab tindak lanjut penggantian (atau tautkan ID log penggantian).
3. Verifikasi bahwa:
   - Kamera 2 dan Kamera 3 masing-masing tercatat `member_outcome = escalated_replacement` dengan tautan tugas penggantian sendiri pada tingkat kamera (`cctv_incident_cameras`).
   - Master data Kamera 2 dan Kamera 3 di `/cctv` otomatis berubah menjadi **Nonaktif (`status = nonaktif`, `monitoring_enabled = false`, `stream_enabled = false`)** dengan label **Nonaktif (Tidak Dipantau)**.
   - Karena Kamera 1 sudah pulih 5 menit dan seluruh kamera sisanya telah diekskalasi, Admin kini dapat menutup tiket induk dengan hasil gabungan **`partial_recovery_and_escalation`**.
   - Buka tab **Laporan Hasil Per Kamera** di `/incidents` -> pastikan Kamera 1 terhitung pada kolom **Pulih**, sedangkan Kamera 2 & 3 terhitung terpisah pada kolom **Dieskalasikan (Penggantian Unit)**.
4. Uji reaktivasi di `/cctv` -> buka detail Kamera 2:
   - Saat stream pengganti masih mati, klik **Aktifkan & Verifikasi Stream** -> pastikan sistem menolak (`409 Conflict`, atau `503 Service Unavailable` bila antrean gate penuh melebihi `CCTV_MANUAL_VERIFY_WAIT_TIMEOUT_MS`) dan kamera **tetap nonaktif** sehingga tidak menimbulkan tiket gangguan palsu.
   - Nyalakan stream unit pengganti yang valid -> klik **Aktifkan & Verifikasi Stream** -> pastikan byte RTP terverifikasi live melalui `UnifiedNVRSessionGate` dan kamera kembali **Aktif** serta masuk kembali ke cakupan monitoring otomatis.

### Skenario G — Gangguan Pemantau (`unknown`), Heartbeat Eksternal, Outbox Ambigu/Gagal, & SLA (Keputusan #6, #7 & Koreksi #3)
1. Hentikan sementara service MediaMTX lokal saat beberapa kamera diperiksa -> pastikan:
   - Kamera berstatus verifikasi `unknown` (*Belum dapat dipastikan*), **bukan** `offline`, dan **tidak** membuat tiket gangguan kamera ke grup teknisi.
   - Hanya **1 episode gangguan pemantau** yang dibuka di `monitor_health_episodes` dan hanya **1 pemberitahuan** (`MONITOR_DEGRADED`) yang dikirim ke In-App Admin, Web Push Admin, serta grup Telegram `admin_group` (tidak banjir per kamera).
   - Hidupkan kembali MediaMTX -> episode pemantau otomatis tertutup (`MONITOR_RECOVERED`) setelah 2 siklus sehat berturut-turut.
2. Panggil endpoint `GET /api/streams/monitor-heartbeat` menggunakan API Key ber-scope `health:read` -> pastikan mengembalikan umur siklus verifier terakhir, antrean `session_gate`, dan status kesehatan pemantau.
3. Uji kegagalan Telegram (misal cabut akses jaringan keluar / token tidak valid pada sandbox):
   - Pastikan baris outbox memisahkan waktu **Diantrekan (`created_at`)**, **Terkonfirmasi API (`sent_at`)**, dan **Gagal (`failed_at`)** / **Ambigu (`ambiguous`)**.
   - Untuk timeout jaringan setelah request terkirim (`ambiguous`), pastikan retry dilakukan pada jeda `10s` lalu `30s` dengan teks catatan kemungkinan pesan duplikat.
   - Pastikan setelah lewat tenggat 5 menit tanpa `sent_at` (termasuk bila gagal permanen), insiden langsung ditandai **`sla_breached = true`** beserta alasan pelanggarannya, dan tombol **Coba Kirim Ulang** di tab Outbox dapat mengirim ulang item gagal.
4. Uji pengingat (`INCIDENT_ASSIGN_REMINDER`) dan eskalasi (`INCIDENT_UNRECOVERED_ESCALATION`) — dapat dipercepat di staging dengan menyetel `CCTV_INCIDENT_REMINDER_MINUTES=1` dan `CCTV_INCIDENT_ESCALATION_MINUTES=2`:
   - Pastikan masing-masing hanya menghasilkan tepat **1 event dan 1 baris outbox per insiden** meskipun worker evaluasi berjalan berulang kali.

---

## 4. Batasan yang Wajib Diuji pada Tahap Staging Perangkat Fisik

Pengujian unit dan integrasi database terisolasi membuktikan kebenaran mesin status, migrasi data lama, kuota antrean gate, dan aturan outbox/SLA. Sebelum mengaktifkan fitur di produksi, tim operasional wajib menjalankan uji staging terkendali dengan perangkat fisik untuk membuktikan tiga hal berikut:
1. **Target SLA 5 Menit End-to-End Aktual**: Mengukur selisih waktu sejak kabel LAN/daya kamera uji diputus secara fisik (`physical_fault_at`) hingga pesan terlihat di grup Telegram sandbox `<= 300 detik`.
2. **Batas Sesi RTSP NVR Aktual**: Memastikan model NVR lapangan stabil melayani batas gate `10` sesi gabungan (`sourceOnDemandCloseAfter: 5s`) tanpa menolak koneksi (`453 Not Enough Bandwidth` / timeout).
3. **Konsumsi Kuota Data GSM Aktual**: Membaca ringkasan aktual `bytes_in + bytes_out` pada tab **Outbox Telegram, Pemantau & GSM** (`cctv_probe_telemetry`) selama 24–72 jam pada 1–2 kamera GSM percontohan dan memastikan 0 sesi GSM bocor sebelum mengaktifkan pemantauan untuk seluruh populasi GSM.
