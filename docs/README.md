# Checklist 25 Bug Manual Test SIGAP

Dokumen ini menjadi daftar utama untuk melanjutkan pemeriksaan dan perbaikan 25 bug hasil pengujian manual SIGAP.

## Lingkungan Pengujian

- Lokasi proyek: `d:\laragon\www\SIGAP`
- Branch: `main`
- Lingkungan: **lokal Windows + Docker Desktop & Production VPS (`https://nexaspace.site`)**
- Entrypoint resmi Docker: `https://localhost` (lokal) & `https://nexaspace.site` (production, backend/frontend internal `:3000`)
- Verifikasi terakhir baseline: 28 September 2026 (WITA); pembaruan sinkronisasi status: 13 November 2026 (WITA)
- Target production (`nexaspace.site`): **lulus verifikasi** (gate manual test `#147` dan runbook production `#148` selesai pada commit `1f984bb` / migrasi v35)

## Arti Status

- `[x]` = sudah diuji di lokal dan hasil utama sesuai harapan.
- `[ ]` = belum diuji ulang, belum selesai, hanya selesai sebagian, atau masih gagal.

## Checklist Pengujian

- [x] **Bug 01 — Pilihan alihkan teknisi:** **lulus pengujian lokal**; teknisi yang sedang ditugaskan dikeluarkan dari pilihan pengalihan dan backend tetap menolak pengalihan ke teknisi yang sama.
- [x] **Bug 02 — PDF laporan pemeliharaan:** **lulus pengujian lokal dan inspeksi visual 4 halaman**; kop, tabel, pembungkusan teks, header halaman lanjutan, serta nomor halaman tampil rapi tanpa tumpang tindih.
- [x] **Bug 03 — Filter kawasan maintenance dan sortir sites:** **lulus pengujian lokal**; daftar, ekspor CSV, dan PDF maintenance mengikuti kawasan CCTV, sedangkan sortir tabel site diproses server-side untuk seluruh halaman data.
- [x] **Bug 04 — Filter kawasan aset CCTV:** **lulus pengujian lokal dan production (`https://nexaspace.site`)**; pengalihan ke `Kawasan OIKN` menampilkan 35 CCTV dan Select-All memilih 35 item, pengalihan ke `Daerah KIPP` menampilkan 0 unit `THD-*`, dan akun `teknisi` tanpa akses ditolak HTTP 403 lintas site.
- [x] **Bug 05 — Informasi “Tugas Saya”:** **lulus verifikasi lokal**; setiap tugas kini menampilkan nama pekerjaan sebagai judul utama, identitas CCTV, lokasi, jadwal, status, serta penanda keterlambatan secara terstruktur.
- [x] **Bug 06 — Pemindai QR pada dashboard teknisi:** **lulus verifikasi lokal**; pojok kanan atas dashboard teknisi menyediakan tombol Pindai dengan ikon scanner yang jelas dan langsung membuka pemindai tanpa klik kedua.
- [x] **Bug 07 — Rasio foto profil:** **lulus verifikasi lokal**; foto disinkronkan setelah sesi selesai dimuat dan ditampilkan pada bingkai persegi menggunakan `object-cover` tanpa mengubah proporsi asli.
- [x] **Bug 08 — MediaMTX deployment:** **lulus verifikasi lokal, production desktop publik (`https://nexaspace.site`), dan jaringan seluler**; WHEP mengembalikan HTTP 201 dengan pasangan ICE `76.13.19.249:8189` (UDP, tanpa kandidat internal `172.28.x.x`), stream simultan `THD-05` & `THD-15` aktif (`readyState=4`), toggle SD (`640×480`) → HD (`2560×1440`) lulus, dan playback pada data seluler tampil normal.
- [x] **Bug 09 — Alur wajib ubah kata sandi:** **lulus verifikasi lokal**; akun baru tetap wajib membuat kata sandi pribadi, redirect tidak lagi saling berebut, sesi dilanjutkan ke dashboard setelah berhasil, dan pilihan keluar kembali ke login tidak melewati kewajiban tersebut.
- [x] **Bug 10 — Rotasi QR CCTV:** **lulus pengujian lokal**; QR baru berbeda tetapi tetap mengarah ke CCTV yang sama, QR lama ditolak, dan cache QR dinonaktifkan.
- [x] **Bug 11 — VAPID push notification:** **lulus pengujian manual lokal**; subscription memakai public key P-256 valid dari backend, aktivasi tidak lagi menghasilkan error `applicationServerKey`, dan service worker berhasil menampilkan notifikasi Windows melalui Microsoft Edge.
- [x] **Bug 12 — Push dan notifikasi in-app:** **lulus pengujian manual end-to-end lokal**; pemicu dari backend menambah notifikasi in-app sekaligus menghasilkan push Windows ketika tab teknisi tidak sedang aktif.
- [x] **Bug 13 — Status hasil pemeliharaan:** **lulus pengujian lokal**; checklist dinamis 8/8 OK menghasilkan `completed/Selesai`, sedangkan adanya item `not_ok` menghasilkan `failed`.
- [x] **Bug 14 — Jadwal tanpa teknisi:** **lulus pengujian manual lokal**; jadwal tanpa teknisi tetap menghasilkan tugas `Terjadwal`, diberi label `Belum ditugaskan`, dan menunggu admin memilih teknisi—bukan otomatis dikerjakan admin.
- [x] **Bug 15 — Notifikasi jadwal harian:** **lulus pengujian lokal**; jadwal harian yang menunjuk teknisi langsung menyimpan notifikasi sebelum respons sukses, perubahan teknisi hanya memberi tahu penerima baru, dan saat generator membuat tugas jalur in-app, WebSocket, serta Web Push dipicu melalui helper yang sama.
- [x] **Bug 16 — Akses teknisi ke laporan kustom:** **lulus pengujian lokal**; halaman memiliki guard klien tanpa flash konten admin, proxy tetap memeriksa peran, dan route API menolak token teknisi dengan HTTP 403.
- [x] **Bug 17 — PDF laporan kustom:** **lulus pengujian lokal dan inspeksi visual 3 halaman**; kolom Nama dan IP NVR memakai lebar adaptif, teks panjang dibungkus, serta header dan nomor halaman tampil konsisten.
- [x] **Bug 18 — Membuka template laporan:** **lulus verifikasi lokal**; setiap template tersimpan menampilkan ringkasan konfigurasi dan tombol `Gunakan Template` yang memuat ulang entitas, kolom, filter, serta format ke builder.
- [x] **Bug 19 — Filter kawasan suku cadang:** **lulus pengujian lokal**; suku cadang kini memiliki relasi kawasan, filter `site_id` diterapkan pada query dan pagination backend, perubahan dropdown langsung memuat ulang daftar, serta formulir tambah/edit menyediakan pilihan kawasan.
- [x] **Bug 20 — Akses teknisi ke suku cadang:** **lulus pengujian manual lokal**; teknisi diarahkan ke dashboard tanpa memuat inventaris, menu Suku Cadang tidak tersedia, dan permintaan langsung ke `/spare-parts` ditolak HTTP 403.
- [x] **Bug 21 — Edit CCTV dan URL RTSP:** **lulus pengujian lokal dan production (`https://nexaspace.site`)**; payload `site_id: null` serta `zone_id: null` bersama URL RTSP diterima HTTP 200 di konteks `Semua Site`, `GET /api/cctv/:id` dan modal Edit menyamarkan kredensial dengan `xxxxx` tanpa membocorkan password mentah, dan penghapusan aset uji berjalan bersih (`DELETE 200`).
- [x] **Bug 22 — Notifikasi penugasan zona:** **lulus pengujian lokal**; teknisi yang baru ditambahkan ke zona menerima notifikasi in-app sebelum respons sukses, event WebSocket/push tetap dikirim, dan klik notifikasi membuka Peta Sebaran CCTV.
- [x] **Bug 23 — Filter zona Sebaran CCTV:** **lulus pengujian lokal**; pilihan zona dimuat dari endpoint zona, pemilihan Zona B mengirim `zone_id=zone-b`, dan ringkasan serta peta hanya menampilkan CCTV pada zona terpilih.
- [x] **Bug 24 — Mode gelap halaman SLA:** **lulus verifikasi lokal**; teks halaman memakai warna foreground tema, sedangkan sumbu, grid, tooltip, dan badge grafik memakai token warna yang valid dan tetap terbaca pada mode gelap.
- [x] **Bug 25 — Tombol Muat ulang SLA:** **lulus verifikasi lokal**; tombol manual `Muat ulang` telah dihapus dan data tetap dimuat otomatis saat halaman dibuka.

## Ringkasan Kemajuan

- Total bug: **25**
- Sudah lulus verifikasi (lokal & gate production): **25** — Bug 1 sampai dengan Bug 25
- Masih gagal setelah diuji: **0**
- Belum selesai atau belum diuji ulang: **0**
- Progress: **25/25 (100%)**

Seluruh skenario checklist 25 bug beserta gate release `#147` dan `#148` telah dijalankan serta diverifikasi lulus hingga production (`https://nexaspace.site`).
