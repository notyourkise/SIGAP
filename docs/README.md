# Checklist 25 Bug Manual Test SIGAP

Dokumen ini menjadi daftar utama untuk melanjutkan pemeriksaan dan perbaikan 25 bug hasil pengujian manual SIGAP.

## Lingkungan Pengujian

- Lokasi proyek: `d:\laragon\www\SIGAP`
- Branch: `main`
- Lingkungan: **lokal Windows + Docker Desktop (Nginx TLS)**
- Entrypoint resmi Docker: `https://localhost` (backend internal `:3000`, frontend internal `:3000`)
- Verifikasi terakhir baseline: 28 September 2026 (WITA); pembaruan sinkronisasi status: 11 November 2026 (WITA)
- Target production (`nexaspace.site`): **belum dieksekusi** (menunggu gate manual test `#147` dan runbook `#148`)

## Arti Status

- `[x]` = sudah diuji di lokal dan hasil utama sesuai harapan.
- `[ ]` = belum diuji ulang, belum selesai, hanya selesai sebagian, atau masih gagal.

## Checklist Pengujian

- [x] **Bug 01 — Pilihan alihkan teknisi:** **lulus pengujian lokal**; teknisi yang sedang ditugaskan dikeluarkan dari pilihan pengalihan dan backend tetap menolak pengalihan ke teknisi yang sama.
- [x] **Bug 02 — PDF laporan pemeliharaan:** **lulus pengujian lokal dan inspeksi visual 4 halaman**; kop, tabel, pembungkusan teks, header halaman lanjutan, serta nomor halaman tampil rapi tanpa tumpang tindih.
- [x] **Bug 03 — Filter kawasan maintenance dan sortir sites:** **lulus pengujian lokal**; daftar, ekspor CSV, dan PDF maintenance mengikuti kawasan CCTV, sedangkan sortir tabel site diproses server-side untuk seluruh halaman data.
- [ ] **Bug 04 — Filter kawasan aset CCTV:** perbaikan kode dan pengujian otomatis lokal sudah lulus; filter kawasan kini memengaruhi total, pagination, pemilihan semua, dan cetak QR, serta tersedia sortir alfabetis berdasarkan nama kawasan. Menunggu verifikasi manual melalui halaman CCTV sebelum dicentang.
- [x] **Bug 05 — Informasi “Tugas Saya”:** **lulus verifikasi lokal**; setiap tugas kini menampilkan nama pekerjaan sebagai judul utama, identitas CCTV, lokasi, jadwal, status, serta penanda keterlambatan secara terstruktur.
- [x] **Bug 06 — Pemindai QR pada dashboard teknisi:** **lulus verifikasi lokal**; pojok kanan atas dashboard teknisi menyediakan tombol Pindai dengan ikon scanner yang jelas dan langsung membuka pemindai tanpa klik kedua.
- [x] **Bug 07 — Rasio foto profil:** **lulus verifikasi lokal**; foto disinkronkan setelah sesi selesai dimuat dan ditampilkan pada bingkai persegi menggunakan `object-cover` tanpa mengubah proporsi asli.
- [ ] **Bug 08 — MediaMTX deployment:** konfigurasi dan diagnosis kode lokal sudah diperbaiki serta tes otomatis lulus; backend native-local/Docker kini memakai alamat MediaMTX yang sesuai, Monitor menampilkan status koneksi nyata, dan ICE menyediakan fallback UDP/TCP 8189. Tetap menunggu verifikasi frame nyata pada production, jaringan publik, dan seluler sebelum dicentang.
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
- [ ] **Bug 21 — Edit CCTV dan URL RTSP:** parser dan pengujian handler lokal sudah diperbaiki; payload `site_id: null` serta `zone_id: null` kini diterima bersama URL RTSP dan menghasilkan HTTP 200 pada mock database. Menunggu verifikasi manual edit aset CCTV sebelum dicentang.
- [x] **Bug 22 — Notifikasi penugasan zona:** **lulus pengujian lokal**; teknisi yang baru ditambahkan ke zona menerima notifikasi in-app sebelum respons sukses, event WebSocket/push tetap dikirim, dan klik notifikasi membuka Peta Sebaran CCTV.
- [x] **Bug 23 — Filter zona Sebaran CCTV:** **lulus pengujian lokal**; pilihan zona dimuat dari endpoint zona, pemilihan Zona B mengirim `zone_id=zone-b`, dan ringkasan serta peta hanya menampilkan CCTV pada zona terpilih.
- [x] **Bug 24 — Mode gelap halaman SLA:** **lulus verifikasi lokal**; teks halaman memakai warna foreground tema, sedangkan sumbu, grid, tooltip, dan badge grafik memakai token warna yang valid dan tetap terbaca pada mode gelap.
- [x] **Bug 25 — Tombol Muat ulang SLA:** **lulus verifikasi lokal**; tombol manual `Muat ulang` telah dihapus dan data tetap dimuat otomatis saat halaman dibuka.

## Ringkasan Kemajuan

- Total bug: **25**
- Sudah lulus lokal: **22** — Bug 1, Bug 2, Bug 3, Bug 5, Bug 6, Bug 7, Bug 9, Bug 10, Bug 11, Bug 12, Bug 13, Bug 14, Bug 15, Bug 16, Bug 17, Bug 18, Bug 19, Bug 20, Bug 22, Bug 23, Bug 24, dan Bug 25
- Masih gagal setelah diuji lokal: **0**
- Belum selesai atau belum diuji ulang: **3** — Bug 4, Bug 8, dan Bug 21
- Progress: **22/25 (88%)**

Checklist hanya boleh dicentang setelah skenario utama benar-benar dijalankan di lokal dan hasil aktual sesuai dengan hasil yang diharapkan. Perubahan kode saja belum dianggap lulus.
