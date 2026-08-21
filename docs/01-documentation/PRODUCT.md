# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

SIGAP digunakan secara setara oleh dua peran internal:

- Admin memantau kondisi dan data CCTV, mengelola inventaris serta pengguna, dan mengawasi aktivitas pemeliharaan.
- Teknisi memantau CCTV yang menjadi tanggung jawabnya dan mencatat pekerjaan pemeliharaan.

Keduanya menggunakan SIGAP ketika perlu memantau aset CCTV dan menjalankan atau meninjau pemeliharaan di kawasan IKN.

## Product Purpose

SIGAP (Sistem Informasi Gangguan & Aset Pusat) adalah sistem internal untuk mengelola CCTV di kawasan IKN. Sistem ini memusatkan pemantauan, inventarisasi, identifikasi perangkat, dan riwayat pemeliharaan agar admin dan teknisi memiliki acuan operasional yang sama.

Keberhasilan produk berarti kondisi aset dapat dipantau, perangkat dapat diidentifikasi dengan tepat, dan pekerjaan pemeliharaan dapat dicatat serta ditelusuri secara konsisten.

## Positioning

SIGAP menghubungkan inventaris CCTV kawasan IKN dengan alur pemeliharaan dan identitas perangkat berbasis QR terenkripsi dalam satu sistem internal. QR terenkripsi menjadi penghubung antara perangkat fisik di lapangan dan catatan digitalnya tanpa mengekspos identitas perangkat secara langsung.

## Operating Context

- Digunakan melalui browser sebagai aplikasi operasional internal.
- Mencakup dashboard pemantauan, data aset dan CCTV, pemindaian QR melalui kamera, impor data, log pemeliharaan beserta foto bukti, serta manajemen pengguna.
- Mendukung pekerjaan administrasi terpusat dan aktivitas teknisi yang berkaitan dengan perangkat CCTV di kawasan IKN.
- Antarmuka dan komunikasi produk menggunakan bahasa Indonesia.

## Capabilities and Constraints

- Peran yang dikonfirmasi adalah `admin` dan `teknisi`.
- Admin memiliki kewenangan untuk mengelola data inti dan pengguna; teknisi dapat menjalankan alur operasional dan pemeliharaan sesuai hak aksesnya.
- Identitas CCTV pada QR harus tetap terenkripsi. Konsistensi kunci enkripsi wajib dijaga karena perubahan kunci dapat membuat QR yang telah dicetak tidak dapat dibaca kembali.
- Produk bergantung pada kamera browser untuk pemindaian QR; pada deployment Docker akses resmi
  `https://localhost` menyediakan secure context untuk pemindaian QR.
- Sistem mencakup pencarian, filter, paginasi, ekspor data, impor Excel, unggah foto bukti, dan mode terang/gelap.
- Sistem ditujukan untuk penggunaan internal, bukan produk publik atau layanan konsumen.

## Brand Commitments

- Nama resmi: SIGAP — Sistem Informasi Gangguan & Aset Pusat.
- Cakupan khusus: pengelolaan CCTV di kawasan IKN.
- Bahasa antarmuka: bahasa Indonesia.
- Logo Nusantara harus dipertahankan; aset tersedia di `frontend/public/nusantara-logo.png`.
- QR terenkripsi serta pembagian peran Admin dan Teknisi merupakan bagian tetap dari identitas dan cara kerja produk.
- Produk harus tampil dan berkomunikasi sebagai sistem internal yang tepercaya, bukan materi pemasaran publik.

## Evidence on Hand

- Logo Nusantara: `frontend/public/nusantara-logo.png`.
- Implementasi alur produk tersedia pada frontend dan REST API, termasuk dashboard, inventaris CCTV, pemindaian QR, pemeliharaan, dan manajemen pengguna.
- Dokumentasi kapabilitas dan endpoint tersedia di `README.md` dan `backend/docs/`.
- Tidak ada testimoni, tolok ukur performa operasional, klaim adopsi, atau bukti eksternal yang telah dikonfirmasi; pekerjaan berikutnya tidak boleh mengarangnya.

## Product Principles

1. Satukan pemantauan dan pemeliharaan agar admin serta teknisi bekerja dari kondisi aset yang sama.
2. Jaga keterlacakan setiap perangkat dan tindakan pemeliharaan dari lapangan hingga catatan digital.
3. Lindungi identitas perangkat melalui QR terenkripsi tanpa menghambat pekerjaan operasional.
4. Utamakan kejelasan, kecepatan, dan ketepatan dalam situasi kerja internal.
5. Pertahankan konteks Nusantara dan bahasa Indonesia di seluruh pengalaman produk.
