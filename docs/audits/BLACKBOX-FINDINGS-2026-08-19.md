# Temuan Black-box SIGAP — 19 Agustus 2026

Produk: Sistem Informasi Gangguan & Aset Pusat (SIGAP)

Environment yang dilaporkan: `https://nexaspace.site`
Metode: black-box testing oleh pengguna, kemudian pemetaan read-only terhadap source repository.

## Cara Membaca

- **Terkonfirmasi:** penyebab atau ketidaksesuaian terlihat langsung pada source saat ini.
- **Sebagian terkonfirmasi:** sebagian rantai penyebab terlihat, tetapi kejadian production masih membutuhkan log/request/database.
- **Perlu reproduksi deployment:** source saat ini seharusnya sudah mencegah kejadian; perlu membandingkan commit/image/config yang aktif.
- **P0:** memblokir akses atau fungsi operasional kritis; **P1:** keamanan/integritas/fungsi utama; **P2:** fungsi penting/UX; **P3:** kosmetik atau kebingungan minor.

Dokumen ini berisi rekomendasi solusi, bukan bukti bahwa perbaikan telah diimplementasikan.

## Register Temuan

### BB-01 — Teknisi aktif masih tersedia pada pilihan pengalihan

- URL: `/maintenance`
- Prioritas: **P2**
- Status: **Terkonfirmasi**
- Temuan: dropdown merender seluruh daftar teknisi tanpa mengecualikan `technician_id` yang sedang ditugaskan. Backend sudah menolak pemilihan teknisi yang sama, sehingga validasi UI dan API tidak selaras.
- Solusi: bentuk `eligibleTechnicians` dengan menghapus teknisi aktif, reset pilihan setiap modal dibuka, dan pertahankan guard backend.
- Acceptance: teknisi aktif tidak muncul saat reassign; assign pertama tetap menampilkan semua teknisi; request manual ke teknisi yang sama tetap ditolak `400`.

### BB-02 — Header dan isi PDF laporan pemeliharaan tidak rapi

- URL: `/maintenance`
- Prioritas: **P2**
- Status: **Terkonfirmasi**
- Temuan: total lebar kolom 190 mm melebihi area konten A4 182 mm. Baris kop setelah baris pertama kembali ke margin kiri. Tinggi baris hanya memperhitungkan kolom tindakan.
- Solusi: normalisasi total lebar ke area konten, buat helper kop yang selalu mengatur posisi X, hitung tinggi dari seluruh kolom panjang, dan ulangi header tabel setelah page break.
- Acceptance: kop sejajar, tabel berada di margin, nama/teknisi/tindakan panjang tidak bertumpuk, dan header berulang pada halaman berikutnya.

### BB-03 — Pemilihan/sortir kawasan tidak bekerja pada maintenance dan sites

- URL: `/maintenance`, `/sites`
- Prioritas: **P1**
- Status: **Terkonfirmasi**
- Temuan: frontend mengirim `X-Site-ID`, tetapi route maintenance tidak memakai `SiteContext` dan handler tidak memfilter melalui site CCTV. Pada daftar site, handler mengabaikan sort yang diparsing dan selalu memakai `nama ASC`; konteks site global juga rancu bila dianggap sebagai filter daftar master site.
- Solusi: pasang site context pada route yang membutuhkan scope, filter maintenance melalui relasi CCTV, dan bedakan jelas site aktif global dari filter/sort lokal halaman `/sites`.
- Acceptance: memilih Site A hanya menampilkan maintenance Site A; reload mempertahankan konteks; invalid site ditolak; sortir daftar site mengikuti pilihan pengguna.

### BB-04 — CCTV tidak dapat difilter per kawasan dan tidak menampilkan kolom kawasan

- URL: `/cctv`
- Prioritas: **P1**
- Status: **Terkonfirmasi**
- Temuan: model/form telah memiliki `site_id`, tetapi route CCTV tidak memasang `SiteContext`; modal filter dan tabel juga tidak memiliki kawasan.
- Solusi: pasang site context, preload relasi Site, tambah filter dan kolom kawasan, serta whitelist sorting berdasarkan nama/kode site.
- Acceptance: CCTV tidak bocor lintas site, kolom menampilkan kawasan atau “Belum ditentukan”, dan filter konsisten pada pagination serta ekspor/cetak.

### BB-05 — Kartu “Tugas Saya” kurang informatif

- URL: `/dashboard`
- Prioritas: **P2**
- Status: **Terkonfirmasi**
- Temuan: backend telah mengirim `action_taken`, tetapi frontend hanya menampilkannya untuk admin. Teknisi hanya melihat nama CCTV dan status.
- Solusi: tampilkan nama pekerjaan/tindakan untuk teknisi, dengan nama CCTV, lokasi, jadwal, status, dan overdue sebagai metadata.
- Acceptance: teknisi dapat mengenali pekerjaan tanpa membuka detail; teks panjang dapat dibaca; data teknisi lain tidak tampil.

### BB-06 — Dashboard teknisi tidak memiliki pintasan pindai QR

- URL: `/dashboard`, sumber fitur `/cctv`
- Prioritas: **P2**
- Status: **Terkonfirmasi**
- Temuan: scanner, state, dan modal hanya diimplementasikan di halaman CCTV.
- Solusi: ekstrak scanner menjadi komponen reusable, tampilkan CTA “Pindai QR Pengecekan Lapangan” pada dashboard teknisi, lalu arahkan hasil scan ke wizard pengecekan.
- Acceptance: kamera aktif hanya saat modal terbuka; QR valid membuka CCTV yang benar; permission denied dan perangkat tanpa kamera mempunyai pesan/fallback yang jelas.

### BB-07 — Rasio foto profil tidak fit

- URL: `/profile`
- Prioritas: **P2**
- Status: **Terkonfirmasi**
- Temuan: avatar memakai kotak 64×64 dengan `object-cover`, sementara upload besar di-resize paksa menjadi 1920×1920. Sumber distorsi utama berada pada proses resize backend.
- Solusi: pertahankan aspect ratio dengan batas sisi terpanjang 1920; untuk avatar gunakan crop 1:1 yang dapat diposisikan atau fallback `object-contain`.
- Acceptance: portrait/landscape tidak terdistorsi, crop avatar konsisten di profil/header, dan output besar tidak selalu 1920×1920.

### BB-08 — MediaMTX tidak terhubung sehingga stream deployment tidak berjalan

- URL: `/monitor`
- Prioritas: **P0**
- Status: **Perlu reproduksi runtime/deployment**
- Temuan: compose hanya memublikasikan UDP 8189; konfigurasi belum menetapkan host publik WebRTC; image memakai tag `latest-ffmpeg`. WHEP signaling yang berhasil belum membuktikan jalur ICE/media dapat dicapai.
- Solusi: pin versi MediaMTX, set `webrtcAdditionalHosts`, buka UDP 8189 pada firewall, sediakan TURN/TLS bila perlu, dan pecah health diagnostic menjadi API, path registration, source RTSP, WHEP, ICE, dan frame playback.
- Acceptance: container API dapat mengakses MediaMTX, path `ready=true`, WHEP memberi SDP answer, dan frame video diterima dari desktop serta jaringan seluler eksternal.

### BB-09 — Navigasi forced change-password membingungkan dan terjadi redirect flash

- URL: `/change-password`
- Prioritas: **P1**
- Status: **Terkonfirmasi**
- Temuan: forced mode sengaja menghilangkan tombol batal; mode biasa selalu kembali ke dashboard. Guard forced password baru berjalan setelah bootstrap client sehingga dashboard dapat muncul sesaat sebelum dialihkan lagi.
- Solusi: forced mode tetap tidak boleh masuk aplikasi tetapi harus menyediakan “Keluar/Kembali ke login”; mode biasa gunakan `router.back()` dengan fallback `/profile`; pindahkan guard sebelum render melalui proxy/server-side auth; gunakan `returnTo` yang tervalidasi setelah sukses.
- Acceptance: tidak ada flash dashboard, forced user tidak dapat melewati halaman, logout tetap tersedia, dan perubahan password biasa kembali ke konteks asal.

### BB-10 — QR baru setelah rotasi tidak berfungsi dan QR lama tidak valid secara tidak konsisten

- URL terkait: `/cctv`; laporan menyebut alur forced `/change-password`
- Prioritas: **P0**
- Status: **✅ Remediasi terverifikasi 2026-08-20**
- Status remediasi source/runtime: **PASS** untuk acceptance lifecycle QR BB-10 pada staging terisolasi; suite agregat MT-11 tetap `NOT RUN` untuk cakupan API key/webhook.
- Temuan: rotasi mengenkripsi UUID acak baru, bukan ID CCTV; scanner lalu mencari CCTV berdasarkan UUID yang tidak ada. Scanner tidak membandingkan payload dengan `qr_payload` aktif, dan cache gambar QR hanya menggunakan ID CCTV sehingga gambar lama dapat tetap dipakai.
- Solusi: enkripsi ulang ID CCTV yang sama dengan nonce baru, validasi ID sekaligus payload aktif saat scan, invalidasi cache setelah rotasi atau gunakan key berbasis versi/hash payload, dan tambah round-trip test.
- Acceptance: QR baru berbeda dan memetakan CCTV yang sama; QR lama ditolak; cetak sesudah rotasi memakai gambar baru tanpa hard refresh.
- Checklist: [x] backend round-trip dan active-payload guard; [x] frontend cache version + HTTP/service-worker `no-store`; [x] automated regression backend + service worker; [x] label terbaru tampil/siap dicetak tanpa hard refresh dan payload hasil decode dipindai pada staging non-production; [x] QR lama ditolak, teknisi 403, audit serta cleanup terverifikasi. Evidence: `MT-11-QR-RUNTIME-EVIDENCE-2026-08-20.md`.

### BB-11 — PushManager menolak `applicationServerKey`

- URL: `/profile`
- Prioritas: **P1**
- Status: **Terkonfirmasi**
- Temuan: frontend selalu meneruskan `NEXT_PUBLIC_VAPID_PUBLIC_KEY`; bila kosong, hasilnya `Uint8Array` kosong. Docker frontend/compose tidak menyediakan public key pada build, sedangkan VAPID hanya disiapkan untuk backend.
- Solusi: rekomendasi utama adalah endpoint runtime `GET /api/push/public-key`; validasi base64url dan panjang key sebelum subscribe; pastikan pasangan key frontend/backend sama; disable CTA dengan pesan “Push belum dikonfigurasi” bila tidak tersedia.
- Acceptance: key kosong/invalid tidak diteruskan, subscription tersimpan idempoten, push percobaan diterima service worker, dan mismatch key terdeteksi oleh health/config test.

### BB-12 — Pengujian push dan in-app dianggap sama-sama terblokir

- URL: `/profile`, `/pemberitahuan`
- Prioritas: **P1**
- Status: **Push terblokir oleh BB-11; in-app merupakan jalur independen**
- Temuan: `NotifyUser` menyimpan notifikasi dan menyiarkan WebSocket sebelum mencoba push. VAPID kosong seharusnya tidak menggagalkan database, bell, atau halaman pemberitahuan.
- Solusi: pisahkan indikator health dan skenario test setiap kanal; buat test-notification admin dengan correlation ID untuk database, WebSocket, push, email, dan webhook.
- Acceptance: tanpa VAPID, in-app tetap masuk; dengan VAPID valid, event yang sama juga menjadi OS push; kegagalan push tidak menghapus atau menunda in-app.

### BB-13 — Hasil pemeliharaan seluruhnya OK masih “Perlu Tindak Lanjut”

- URL: `/maintenance`
- Prioritas: **P1**
- Status: **Cacat desain terkonfirmasi; kasus tepat 9/9 perlu capture production**
- Temuan: backend hanya menetapkan `completed` bila semua jawaban `ok` dan jumlah persis sembilan, sedangkan checklist frontend bersifat dinamis. Delapan atau sepuluh item seluruhnya OK akan selalu gagal. Unit test sembilan item OK saat ini lulus, sehingga laporan produksi tepat 9/9 perlu request/response dan record database.
- Solusi: hilangkan konstanta sembilan; kirim identitas template/item; backend memuat template aktif yang berlaku pada site, memastikan seluruh item wajib hadir tepat sekali, lalu menghitung status. Payload kurang/duplikat harus `400`, bukan diam-diam `failed`.
- Acceptance: 8/8 atau 9/9 sesuai template menghasilkan `completed`; satu item wajib bermasalah menghasilkan `failed`; item wajib hilang/duplikat ditolak; API, DB, dashboard, dan notifikasi konsisten.

### BB-14 — Makna jadwal tanpa teknisi tidak jelas

- URL: `/schedules`
- Prioritas: **P2**
- Status: **Terkonfirmasi**
- Temuan: backend membuat log `scheduled` dengan `technician_id = null`; admin tidak otomatis menjadi pelaksana. UI justru memberi label “Langsung oleh admin”.
- Solusi: pertahankan sebagai antrean “Belum ada teknisi” dan perbaiki copy. Bila admin perlu menjadi pelaksana, lakukan keputusan produk serta perubahan model pelaksana, bukan hanya teks.
- Acceptance: tanpa teknisi menghasilkan `scheduled/null`; dengan teknisi menghasilkan assignment sesuai kontrak; tidak ada teks yang menyatakan admin otomatis menangani.

### BB-15 — Notifikasi teknisi dari jadwal harian tidak masuk

- URL: `/schedules`, `/pemberitahuan`
- Prioritas: **P1**
- Status: **Sebagian terkonfirmasi**
- Temuan: `next_run_at` pertama berada satu frekuensi setelah pembuatan; generator berjalan per jam; notifikasi baru dibuat saat jatuh tempo. Generator menulis langsung ke DB dan broadcast, melewati helper notifikasi standar/push.
- Solusi: bedakan event “jadwal dibuat” dan “tugas jatuh tempo”; sepakati waktu pengiriman; gunakan `NotifyUserSync` atau outbox setelah log commit; tambahkan idempotensi log/notifikasi.
- Acceptance: notifikasi muncul pada waktu yang disepakati, hanya satu log/notifikasi per occurrence, tetap tercatat ketika user offline, dan push dipanggil bila konfigurasi valid.

### BB-16 — Teknisi dapat membuka halaman laporan kustom

- URL: `/laporan-kustom`
- Prioritas: **P1**
- Status: **Perlu reproduksi deployment**
- Temuan: source saat ini memasukkan route dalam admin paths dan seluruh API report memakai `RequireRole("admin")`. Kandidatnya adalah image frontend lama, proxy tidak aktif, atau shell sempat dirender sementara API tetap 403.
- Solusi: cocokkan commit SHA/image production, tambah guard komponen sebagai defense-in-depth, pertahankan backend authorization, dan tambah E2E direct URL.
- Acceptance: teknisi diarahkan ke 403/dashboard sebelum konten dirender dan seluruh `/api/reports/*` mengembalikan 403; admin tetap berfungsi.

### BB-17 — Kolom nama dan IP NVR pada PDF laporan kustom tidak rapi

- URL: `/laporan-kustom`
- Prioritas: **P2**
- Status: **Terkonfirmasi**
- Temuan: semua kolom diberi lebar sama, isi dipotong berdasarkan jumlah karakter, dan tidak ada wrapping/tinggi baris dinamis.
- Solusi: gunakan bobot lebar per tipe kolom, multi-line cell, tinggi baris dinamis, header berulang, serta orientasi A4 landscape/A3 berdasarkan jumlah kolom.
- Acceptance: nama panjang dan IP tidak bertumpuk, baris tetap sejajar, header berulang, dan sembilan kolom terbaca pada zoom 100%.

### BB-18 — Template laporan tersimpan tetapi tidak jelas cara membukanya

- URL: `/laporan-kustom`
- Prioritas: **P2**
- Status: **Fungsi load tersedia; kegagalan runtime belum terkonfirmasi**
- Temuan: nama template sebenarnya clickable, tetapi tidak ada affordance “Gunakan/Buka”. Backend hanya memvalidasi JSON, bukan skema konfigurasi.
- Solusi: tambah tombol “Gunakan template”, indikator aktif dan preview; validasi schema entity/format/filter/kolom; reset state sebelum menerapkan template.
- Acceptance: konfigurasi pulih persis setelah reload; template rusak/versi lama menghasilkan error informatif; output sama dengan konfigurasi sebelum disimpan.

### BB-19 — Filter kawasan suku cadang tidak berfungsi

- URL: `/suku-cadang`
- Prioritas: **P1**
- Status: **Terkonfirmasi**
- Temuan: UI hanya memiliki kategori/search; model tidak memiliki `site_id`; handler/route tidak memakai site context.
- Solusi: putuskan model inventaris per kawasan. Untuk stok per site, gunakan tabel `spare_part_stocks(spare_part_id, site_id, stock, minimum_order)`; migrasikan data default; tambah site context, filter, kolom, dan validasi penggunaan sesuai site maintenance.
- Acceptance: stok terpisah per kawasan, penggunaan mengurangi stok kawasan yang benar, teknisi tidak dapat memakai stok site lain, dan data lama masuk site default.

### BB-20 — Teknisi dapat memaksa akses halaman suku cadang

- URL: `/suku-cadang`
- Prioritas: **P1**
- Status: **Drift deployment mungkin; API read memang terbuka bagi teknisi**
- Temuan: source frontend telah menandai halaman admin-only, tetapi backend sengaja memperbolehkan semua user terautentikasi membaca inventaris. Teknisi kemungkinan tetap membutuhkan opsi barang saat mencatat penggunaan.
- Solusi: pisahkan endpoint admin inventaris dari endpoint teknisi `options` yang hanya berisi field minimum dan stok site yang diizinkan. Audit image/proxy production.
- Acceptance: teknisi tidak dapat membuka halaman/admin API, namun tetap dapat memilih spare part sah dari alur maintenance; CRUD hanya admin.

### BB-21 — Edit CCTV dan URL RTSP gagal dengan “Format Input Tidak Valid”

- URL: `/cctv`
- Prioritas: **P1**
- Status: **Terkonfirmasi**
- Temuan: frontend mengirim `zone_id: ""` dan `site_id: ""`, sedangkan backend mem-parsing langsung sebagai `*uuid.UUID`. Body parsing gagal sebelum validasi RTSP.
- Solusi: gunakan DTO optional UUID dengan tiga keadaan—tidak dikirim, UUID valid, atau null untuk melepas relasi—dan normalisasi frontend/backend. Berikan validasi URL RTSP terpisah serta pertahankan password lama bila input password kosong.
- Acceptance: edit nama/RTSP tanpa zona/site berhasil; relasi dapat diisi, diganti, dikosongkan; UUID dan URL invalid memberi error field-specific.

### BB-22 — Notifikasi penugasan zona terlambat atau tidak muncul

- URL: `/zona`, `/pemberitahuan`
- Prioritas: **P1**
- Status: **Terkonfirmasi: belum diimplementasikan**
- Temuan: handler hanya mengganti relasi user-zone dan audit log; tidak membuat notifikasi, push, atau broadcast.
- Solusi: bandingkan assignment lama/baru setelah transaksi; kirim `ZONE_ASSIGNED`/`ZONE_UNASSIGNED` melalui `NotifyUserSync` atau transactional outbox setelah commit; hindari duplikasi untuk user yang tidak berubah.
- Acceptance: user baru menerima tepat satu notifikasi, user tetap tidak menerima ulang, rollback tidak menimbulkan notifikasi palsu, dan notifikasi tersimpan saat offline.

### BB-23 — Sebaran CCTV tidak memiliki filter/sortir zona

- URL: `/sebaran-cctv`
- Prioritas: **P2**
- Status: **Terkonfirmasi**
- Temuan: halaman tidak memuat zona dan selalu meminta seluruh CCTV; backend sebenarnya telah mendukung `zone_id`.
- Solusi: muat daftar zona, tambah dropdown “Semua zona”, sertakan `zone_id` pada query, dan terapkan hasil yang sama pada marker, grid stream, statistik, dan selection.
- Acceptance: marker/grid/statistik mengikuti zona; semua zona memulihkan data dalam site aktif; zona kosong memberi empty state; UUID invalid tidak menyebabkan crash.

### BB-24 — Teks SLA tidak terbaca pada mode gelap

- URL: `/sla`
- Prioritas: **P2**
- Status: **Penyebab chart terkonfirmasi**
- Temuan: chart memakai CSS variable `--muted` dan `--card` yang tidak didefinisikan; token sebenarnya `--muted-color` dan `--card-bg`. Dampak pada seluruh halaman masih perlu visual audit.
- Solusi: gunakan design token valid, set warna tooltip/foreground eksplisit, audit badge dan seluruh state, serta tambahkan visual regression light/dark.
- Acceptance: axis, label, tooltip, tabel, dan badge memenuhi kontras minimum 4.5:1; perubahan tema memperbarui chart tanpa reload.

### BB-25 — Tombol “Muat ulang” masih ada pada SLA

- URL: `/sla`
- Prioritas: **P3**
- Status: **Terkonfirmasi pada source saat audit**
- Temuan: tombol masih dirender pada header normal.
- Solusi: hapus tombol normal sesuai keputusan produk; pertahankan “Coba lagi” hanya pada error state; gunakan refresh otomatis/WebSocket bila diperlukan.
- Acceptance: tidak ada “Muat ulang” pada kondisi normal, initial load tetap bekerja, dan error state tetap memiliki retry.

## Ringkasan Prioritas

| Prioritas | Temuan | Alasan |
|---|---|---|
| P0 | BB-08, BB-10 | Streaming operasional mati dan rotasi QR merusak identifikasi aset. |
| P1 | BB-03, BB-04, BB-09, BB-11–BB-13, BB-15–BB-16, BB-19–BB-22 | Isolasi site, akses, status pekerjaan, notifikasi, dan integritas data. |
| P2 | BB-01–BB-02, BB-05–BB-07, BB-14, BB-17–BB-18, BB-23–BB-24 | Efisiensi operasional, laporan, UX, dan aksesibilitas. |
| P3 | BB-25 | Konsistensi UI minor. |

## Bukti Tambahan yang Wajib Dikumpulkan

Sebelum implementasi pada temuan berstatus deployment/runtime, kumpulkan commit SHA/image digest production, konfigurasi environment yang sudah disanitasi, log backend/frontend/MediaMTX, response Network, event WebSocket, record database terkait, dan screenshot/video reproduksi. Jangan menyalin secret VAPID, RTSP, token, atau kredensial ke dokumen.

Urutan eksekusi dan hubungan dengan audit manual test dirangkum pada `docs/audits/CONSOLIDATED-FINDINGS-INDEX.md` dan `docs/plans/FINDINGS-REMEDIATION-PLAN.md`.
