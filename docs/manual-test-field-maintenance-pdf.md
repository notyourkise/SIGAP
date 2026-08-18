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

## Hasil Nyata (18 Agu 2026, produksi `https://nexaspace.site`)

| # | Skenario | Hasil nyata |
|---|----------|-------------|
| 1 | Login Admin, buka `/field-maintenance` | ✅ LULUS — halaman 200 untuk admin (dan teknisi); record uji `c06cb6f1-030c-4fd3-93e9-9e12a5a67625` tampil di list (7 catatan total) |
| 2 | Buka detail catatan pengecekan | ✅ LULUS — `GET /api/field-maintenance/c06cb6f1...` → detail lengkap: CCTV "Manual Test CCTV 001", teknisi "Teknisi Manual Test", overall `ok`, 10 checklist, 1 foto bukti, catatan "Pengecekan E2E template checklist - data uji manual", `completed_at 2026-08-18T12:26:37+08:00` |
| 3 | Tombol **Cetak Laporan** (ikon `FileDown`) di footer modal detail | ✅ LULUS — tombol dirender hanya bila `isAdmin` (`field-maintenance/page.tsx:792-802`); untuk teknisi tombol tidak ditampilkan |
| 4 | Klik **Cetak Laporan** | ✅ LULUS — `GET /api/field-maintenance/{id}/report-pdf` → HTTP 200, `Content-Type: application/pdf`, `Content-Disposition: attachment; filename=laporan_pengecekan_lapangan_c06cb6f1-030c-4fd3-93e9-9e12a5a67625.pdf`; file 573 KB terunduh (foto bukti ter-embed) |
| 5 | Buka PDF: kop surat IKN, kode/nama/lokasi/status CCTV, teknisi, tanggal/waktu, status keseluruhan, tabel checklist, foto bukti/URL, catatan umum | ✅ LULUS (verifikasi kode + signature) — PDF dimulai `%PDF-` (byte 37,80,68,70,45). Isi laporan dikonfirmasi dari generator: kop surat IKN (`drawLetterhead` — logo IKN kiri + identitas instansi + alamat "Kompleks Perkantoran Pemerintah IKN, Kabupaten Penajam Paser Utara", `report.go:173-180`), identitas CCTV + teknisi, checklist + foto bukti + catatan umum (`field_maintenance.go:423`). Unit test `TestGenerateFieldMaintenanceReportPDF_Success` memastikan body `%PDF` (`field_maintenance_test.go:461-479`). Isi teks tidak dapat di-string-scan karena terkompres FlateDecode — normal pada PDF |
| 6 | Teknisi buka detail catatan milik sendiri, tombol **Cetak Laporan** tidak tampil | ✅ LULUS — tombol diguard `isAdmin` (S3); API tetap melayani teknisi pemilik: teknisi.test GET report-pdf record miliknya → 200 `application/pdf` (skenario unduh via URL langsung) |
| 7 | Teknisi bukan pemilik record | ✅ LULUS — teknisi.dua GET report-pdf record milik teknisi.test → **HTTP 403** (ownership check); unit test `TestGenerateFieldMaintenanceReportPDF_TechnicianOwnershipDenied` juga menutup kasus ini |
| 8 | Endpoint report dengan UUID tidak ada | ✅ LULUS — `GET /api/field-maintenance/00000000-0000-0000-0000-000000000000/report-pdf` → **HTTP 404** |
| — | Tanpa sesi valid | ✅ LULUS — tanpa token → **HTTP 401** |

## Catatan

- Seluruh 8 skenario lulus; 4 unit test backend menutup endpoint ini (`field_maintenance_test.go:461-524` — Success, TechnicianOwner, TechnicianOwnershipDenied, NotFound).
- Teknisi hanya diizinkan mengunduh record miliknya; Admin diizinkan semua record.
- Nama file unduhan mengikuti pola `laporan_pengecekan_lapangan_<uuid>.pdf` sesuai spesifikasi.
