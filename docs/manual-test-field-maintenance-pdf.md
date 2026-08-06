# Uji Manual — Laporan PDF Pengecekan Lapangan (Task 6.3)

## Prasyarat

- Backend dan frontend berjalan melalui konfigurasi development atau Docker.
- Tersedia akun Admin dan satu catatan pengecekan lapangan dengan checklist serta foto bukti.
- Browser sudah login ke SIGAP.

## Skenario

1. Login sebagai Admin lalu buka `/field-maintenance`.
2. Buka detail salah satu catatan pengecekan.
3. Pastikan tombol **Cetak Laporan** dengan ikon `FileDown` terlihat di footer modal detail.
4. Klik **Cetak Laporan**.
5. Pastikan browser mengunduh file `laporan_pengecekan_lapangan_<uuid>.pdf`.
6. Buka PDF dan pastikan kop surat IKN, kode/nama/lokasi/status CCTV, teknisi, tanggal/waktu, status keseluruhan, tabel checklist, foto bukti atau URL foto, serta catatan umum (bila ada) tampil.
7. Login sebagai Teknisi dan buka detail catatan milik sendiri. Pastikan tombol **Cetak Laporan** tidak ditampilkan.
8. Panggil endpoint report dengan ID UUID yang tidak ada. Pastikan API mengembalikan HTTP 404.

## Hasil yang Diharapkan

- Download hanya berjalan untuk sesi yang valid.
- Data laporan hanya dapat dibaca Admin atau teknisi pemilik record.
- PDF valid dan dimulai dengan signature `%PDF`.
