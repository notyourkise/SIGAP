# Pemetaan Teknis Monitoring dan Telegram untuk Antigravity

Tanggal: 1 November 2026 (WITA)

Acuan: [kebutuhan yang disetujui pengguna](KEBUTUHAN-MONITORING-CCTV-TELEGRAM-2026-11-01.md).

Status: pemetaan kode dan kesenjangan kebutuhan oleh Anto, untuk membantu Antigravity menyusun desain. Bukan desain final atau instruksi menjalankan perubahan. Implementasi tetap menjadi tanggung jawab Antigravity.

## 1. Lingkup dan batas bukti

- Checkout yang diperiksa: HEAD `6a612131478ef10353aeb40b483e06b3b4e75bc9`, beserta working tree lokal pada tanggal di atas. Checkout harus diperiksa ulang saat implementasi karena dapat berubah.
- Tool knowledge graph tidak tersedia pada sesi ini. Pemeriksaan memakai pencarian dan pembacaan sumber langsung; tidak ada klaim cakupan graph lengkap.
- Lingkup: prober, verifier stream, model CCTV/maintenance, pembuatan dan penugasan tugas, penyelesaian checklist, utilitas notifikasi, rute maintenance, dan bagian terkait di halaman maintenance.
- Tidak menjalankan server aplikasi, migrasi, perubahan database operasional, probe kamera nyata, atau pengiriman Telegram.
- Tidak memeriksa runtime environment aktif. Nilai konfigurasi di bawah adalah default kode/compose, bukan klaim nilai yang sedang dipakai deployment.
- Ini pemetaan kebutuhan baru, bukan audit menyeluruh kedua atau penerbitan issue GitHub tambahan.

## 2. Bagian yang sudah tersedia

| Bagian | Bukti sumber | Implikasi |
|---|---|---|
| Dua proses monitoring | [main.go](../../backend/main.go), `RunRTSPProber` baris 174 dan `RunStreamVerifier` baris 180 | Verifikasi aktif sudah ada; jangan membangun engine baru tanpa menilai pemakaian ulang. |
| Prober berkala 2 menit | [stream_probe.go](../../backend/handlers/stream_probe.go), baris 15, 81–168 | Memuat CCTV dengan `stream_enabled=true`. Metro-E dapat memakai probe langsung; GSM memakai status MediaMTX di jalur ini. |
| Verifikasi aktif seluruh kamera yang streaming-nya diaktifkan | [stream_verify.go](../../backend/handlers/stream_verify.go), `runStreamVerifyCycle` baris 109, `verifyCCTVStream` baris 172 | Dapat memicu HLS on-demand dan memeriksa kesiapan sumber; jalur ini tidak membatasi verifikasi hanya pada Metro-E. |
| Bukti status dan riwayat | [cctv.go](../../backend/models/cctv.go), baris 75–112; `persistStreamVerification` baris 277 pada verifier | Sudah ada status pasif, status terverifikasi, waktu, metode, jumlah kegagalan, error, event log, dan broadcast. Belum merupakan model siklus insiden. |
| Tugas tanpa teknisi | [maintenance.go](../../backend/handlers/maintenance.go), `CreateMaintenanceLog` baris 63, khususnya 127–138 | Tugas baru berstatus `scheduled` tanpa teknisi, sesuai kebutuhan penugasan dilakukan kemudian. |
| Penugasan admin melalui website | [routes/maintenance.go](../../backend/routes/maintenance.go), baris 30; `AssignMaintenanceLog` baris 472 pada handler; [halaman maintenance](../../frontend/src/app/maintenance/page.tsx), baris 560–587 | Rute assign sudah admin-only dan frontend sudah memanggilnya. Ini titik integrasi yang dapat dipertahankan. |
| Notifikasi internal | [notify.go](../../backend/utils/notify.go), `NotifyUser` baris 25 dan `NotifyRole` baris 81 | Sudah menyimpan notifikasi internal, broadcast, dan memanggil Web Push. Fungsi tersebut bukan pengirim Telegram. |

## 3. Kesenjangan yang perlu diselesaikan desain

### A. Target offline maksimal 5 menit belum didukung jadwal verifier bawaan

`defaultStreamVerifyInterval` adalah 15 menit, ambang offline dua kegagalan, dan freshness 30 menit (`stream_verify.go:23–26`). Interval environment hanya menerima 5–30 menit (`streamVerifyInterval`, baris 81–90). Compose juga memakai default 15 menit dan satu worker (`docker-compose.yml:107–108`).

Untuk kamera idle yang mengandalkan verifier periodik, gangguan tepat setelah verifikasi berhasil dapat menunggu hampir dua interval sebelum dinyatakan offline, ditambah waktu pemeriksaan dan antrean. Ini penalaran dari kode, bukan hasil pengukuran perangkat. Mengubah interval menjadi 5 menit saja tidak menjamin target end-to-end 5 menit, terlebih dengan dua kegagalan.

Antigravity perlu menghitung seluruh anggaran waktu: menunggu giliran, pemeriksaan, konfirmasi, antrean insiden, transaksi tiket, dan pengiriman. Nilai final belum disepakati.

### B. Status keterjangkauan belum sama dengan kesehatan video

Fungsi `ProbeICMP` sebenarnya membuka TCP port 554 lalu fallback 80 (`utils/rtsp_probe.go:77–87`), bukan mengirim ICMP. Prober dapat menetapkan Metro-E `live` saat koneksi TCP berhasil (`stream_probe.go:125–129`). Jangan menjadikan hasil ini satu-satunya bukti kanal video sehat.

Verifier memakai `PathInfo.SourceReady()`, yang membaca `Available || Ready` (`utils/mediamtx.go:55–56`). Bukti ini perlu dinilai terhadap kebutuhan kesehatan video, termasuk sumber yang masih dianggap siap tetapi gambar berhenti. Pemeriksaan frame/kemajuan data belum dibuktikan di pemetaan ini.

`effectiveStreamState` dapat memilih hasil verifikasi yang masih dianggap segar selama 30 menit sebelum status pasif (`stream_verify.go:466–495`). Konsumen insiden harus memiliki aturan kesegaran bukti sendiri yang memenuhi target, bukan memakai ringkasan tampilan tanpa penilaian.

### C. Cakupan “seluruh CCTV” perlu definisi kamera yang wajib dipantau

Prober dan verifier menyaring `stream_enabled=true`. Kebutuhan pengguna mencakup seluruh CCTV. Kamera yang streaming-nya dinonaktifkan, sengaja nonaktif, atau dalam maintenance perlu aturan eksplisit; jangan diam-diam dianggap tercakup oleh engine sekarang.

### D. Tiket bersama belum terwakili oleh model tugas yang dibaca

`MaintenanceLog` menyimpan satu `CCTVID` dan satu teknisi (`models/setup.go:57–96`). Frontend juga mendefinisikan satu `cctv_id` dan satu `cctv` (`maintenance/page.tsx:50–65`). Model yang dibaca ini belum menyediakan daftar kamera terdampak dalam satu tugas.

Pilihan untuk desain Antigravity: entitas insiden dengan anggota kamera yang ditautkan ke tugas maintenance, atau perluasan model maintenance menjadi multi-kamera. Anto merekomendasikan entitas insiden terpisah agar siklus gangguan tidak mengubah makna tugas rutin. Ini rekomendasi, belum keputusan desain pengguna. Jangan mengelompokkan hanya karena IP NVR sama tanpa bukti dan batas site yang jelas.

### E. Penutupan tiket gangguan memerlukan verifikasi admin tersendiri

`CompleteMaintenanceLog` mengizinkan admin atau teknisi pemilik tugas, menghitung hasil checklist menjadi `completed`/`failed`, lalu menyimpan status akhir dan `CompletedAt` dalam transaksi (`handlers/maintenance.go:677–699,735–825`). Frontend menampilkan aksi selesai bagi admin atau teknisi yang ditugaskan (`maintenance/page.tsx:589–590`).

Pengguna sudah menegaskan verifikasi admin hanya untuk tiket gangguan otomatis CCTV. Maintenance rutin tetap menggunakan alur sekarang. Desain harus membedakan hasil kerja teknisi dari penutupan insiden oleh admin, termasuk kasus kamera pulih sebelum ada penugasan. Jangan menerapkan verifikasi admin secara global ke seluruh tugas rutin.

### F. Persistensi insiden, kestabilan pemulihan, dan pengiriman Telegram perlu ditambahkan atau dirancang

Pada alur prober/verifier yang dibaca, perubahan status disimpan dan disiarkan, tetapi tidak diteruskan menjadi tiket insiden atau pesan Telegram. Pencarian literal `telegram`, `api.telegram.org`, `sendMessage`, dan `chat_id` pada sumber Go/TS/TSX di `backend` serta `frontend/src` tidak menemukan integrasi Telegram. Ini batas hasil pencarian, bukan bukti ketiadaan integrasi eksternal di luar checkout atau dengan nama lain.

Antigravity perlu mendesain penyimpanan identitas insiden, anggota kamera, waktu kandidat pulih, riwayat transisi, hubungan tiket, serta pekerjaan pengiriman yang bertahan saat restart. Rekomendasi Anto: transaksi untuk insiden/tiket dan pencatatan pekerjaan kirim, lalu pengiriman terpisah dengan percobaan ulang. Retry jaringan tidak boleh membuat tiket baru.

Jangan menjanjikan exactly-once untuk pesan eksternal: koneksi terputus setelah Telegram menerima pesan dapat meninggalkan hasil pengiriman ambigu. Tetapkan pencatatan dan penanganan ambiguitas itu sebelum implementasi.

### G. Durasi monitoring GSM harus mencakup masa hidup sumber

Verifier memiliki timeout polling 8 detik dan pemicu HLS 2 detik, tetapi operasi HTTP lain memiliki timeout tersendiri. Jadi 8 detik bukan batas total seluruh verifikasi. `TriggerHLS` menutup response dan context (`utils/mediamtx.go:288–324`), sementara `mediamtx.yml:73` menetapkan `sourceOnDemandCloseAfter: 30s`.

Penutupan request bukan bukti sumber kamera langsung berhenti ditarik. Ukur masa hidup HLS/sumber serta data aktual dalam runtime uji; pemetaan ini belum membuktikan kapan sumber berhenti atau berapa kuota terpakai.

## 4. Urutan kerja yang direkomendasikan untuk Antigravity

1. Periksa ulang checkout dan selesaikan keputusan bisnis terbuka pada dokumen kebutuhan, terutama bukti offline dan batas kamera yang dipantau.
2. Ajukan desain penyimpanan insiden, hubungan multi-kamera dengan maintenance, dan verifikasi admin khusus insiden. Sertakan cara menjaga kompatibilitas tugas rutin.
3. Rancang jadwal dan kapasitas monitoring beserta perhitungan kasus terburuk target 5 menit. Pakai ulang verifier yang ada bila sesuai, sambil memisahkan kesehatan pemantau dari kesehatan kanal.
4. Rancang siklus pemulihan stabil 5 menit, kegagalan pemeriksaan, dan restart; jangan menganggap satu bukti online lalu menunggu timer sebagai bukti stabil.
5. Rancang pengiriman Telegram, penggabungan pesan tanpa menunda target, pembaruan assign/pemulihan, pencatatan hasil, serta eskalasi kegagalan. Teknisi tetap tidak perlu menautkan akun.
6. Setelah desain disetujui, susun rencana implementasi dan uji dalam lingkungan terisolasi. Perubahan skema, database operasional, konfigurasi perangkat, dan deployment memerlukan otorisasi sesuai lingkupnya.

Ini urutan rekomendasi, bukan rencana implementasi final dengan pilihan skema dan kontrak fungsi yang telah disetujui.

## 5. Verifikasi yang dijalankan Anto

Pada checkout ini, pengujian terpilih berikut dijalankan dari direktori `backend`:

```text
go test ./handlers -run '^(TestRunProbeCycle_|TestVerifyCCTVStream|TestEffectiveStreamState|TestIsMediaMTXNotFound)' -count=1 -timeout=90s
Hasil: ok sigap-backend/handlers 6.406s

go test ./utils -run '^(TestIsMetroE_|TestMetroEDirectProbeEnabled_|TestPersistNotification)' -count=1 -timeout=45s
Hasil: ok sigap-backend/utils 0.697s
```

Tes menggunakan mock database dan layanan uji lokal. Tes tersebut memeriksa perilaku yang sudah ada, termasuk klasifikasi idle/unknown, verifier, identifikasi Metro-E, serta persistensi notifikasi. Lulusnya tes tidak membuktikan fitur tiket otomatis, Telegram, pemulihan 5 menit, atau target end-to-end sudah tersedia dan benar.

Belum diuji: perangkat/NVR nyata, penggunaan kuota, pengiriman ke grup nyata, kegagalan lintas proses, restart saat insiden, atau tampilan baru. Tidak ada kode aplikasi yang diperbaiki pada pekerjaan ini.
