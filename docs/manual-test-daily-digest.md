# Uji Manual — Ringkasan Operasional Otomatis (Task 6.4)

## Prasyarat

- Backend dan frontend berjalan melalui konfigurasi development atau Docker.
- Tersedia akun Admin dan data tugas, pengecekan lapangan, serta perubahan status CCTV.
- Browser sudah login ke SIGAP.

## Skenario

1. Login sebagai Admin lalu pastikan menu **Ringkasan** dengan ikon `ClipboardList` tampil di sidebar.
2. Buka `/ringkasan` dan pastikan timeline menampilkan kartu per tanggal jika digest sudah tersedia.
3. Periksa setiap kartu: tanggal, summary naratif, tugas baru, selesai, gagal, overdue, pengecekan lapangan, jumlah attention, dan perubahan status CCTV.
4. Jika tersedia lebih dari 10 digest, gunakan paginasi dan pastikan halaman berikutnya memuat tanggal yang lebih lama.
5. Jalankan atau tunggu generator pada pukul 06:00 WITA. Pastikan digest tanggal tersebut dibuat satu kali.
6. Pastikan Admin menerima notifikasi **Ringkasan operasional [tanggal] tersedia** pada menu Pemberitahuan.
7. Login sebagai Teknisi. Pastikan menu Ringkasan tidak tampil dan membuka `/ringkasan` mengarahkan kembali ke `/dashboard`.

## Hasil yang Diharapkan

- Generator hanya membuat satu digest untuk setiap `digest_date`.
- Ringkasan memakai data 24 jam terakhir dan menampilkan angka yang konsisten dengan aktivitas operasional.
- Endpoint digest hanya dapat diakses Admin.
