# Uji Manual — Laporan PDF Pengecekan Lapangan (Task 6.3)

> **Kontrak uji formal — MT-06**
>
> Dokumen ini mendefinisikan prosedur; hasil sebelum **2026-08-20** adalah catatan historis/non-authoritative dan harus diuji ulang dengan execution record formal. Inspeksi kode, review unit test, atau checkmark lama bukan `PASS` black-box.
>
> **Skenario audit reproduktif:** Dengan fixture pengecekan pendek/panjang, foto, tabel panjang, dan role berbeda, unduh PDF lalu render setiap halaman dan uji akses UUID.
>
> **Status suite:** `NOT RUN`

### Formal execution metadata

| Field | Value |
|---|---|
| Test ID / Audit ID | MT-06 |
| Capability owner | QA Dokumen + Backend |
| Status suite | NOT RUN |
| Build SHA | NOT RECORDED |
| Artifact digest/ID | NOT RECORDED |
| Tester | NOT RECORDED |
| Timestamp WITA | NOT RECORDED |
| Environment | NOT RECORDED |
| Fixture prefix/data | `E2E-MT06-` / NOT RECORDED |
| Cleanup result | NOT RUN |
| Expected | Tidak ada overlap/cutoff, foto dan tabel terbaca, MIME/filename benar, dan kontrol akses terbukti dari response nyata. |
| Actual | NOT RECORDED |
| Evidence path/link | NOT RECORDED |
| Mandatory flag | YES |
| Scenario status | NOT RUN |

### Formal scenario record

| ID | Mandatory | Expected | Actual | Evidence | Status |
|---|---|---|---|---|---|
| MT-06-S1 | YES | Tidak ada overlap/cutoff, foto dan tabel terbaca, MIME/filename benar, dan kontrol akses terbukti dari response nyata. | NOT RECORDED | NOT RECORDED | NOT RUN |

> **Cakupan aggregate Wave 0:** Baris formal aggregate di atas adalah **sole atomic suite-level acceptance gate** untuk baseline Wave 0. Skenario legacy terperinci di bawah ini tetap non-formal dan historical/non-authoritative sampai diperluas selama Wave 7.

**Aturan kesimpulan suite:** sebelum eksekusi suite `NOT RUN`; setelah run dimulai, setiap mandatory scenario harus `PASS`. Satu mandatory scenario berstatus `FAIL`, `BLOCKED`, atau `NOT RUN` membuat kesimpulan suite `FAIL`.

> **Guard keselamatan wajib:** Untuk setiap SQL mutasi, `DELETE`/`TRUNCATE`, rotasi/regenerasi key atau QR, penghapusan, reset kredensial, atau mutasi credential, lanjut hanya bila environment bukan production dan target bukan `nexaspace.site`, fixture prefix `E2E-MT06-` serta exact targets sudah dicatat, backup/recovery terverifikasi untuk perubahan data/key state, dan cleanup/rollback didefinisikan serta dicatat setelah run. Jika salah satu syarat tidak tersedia, hentikan prosedur dan set scenario status ke `BLOCKED`; jangan jalankan pada production. Jangan menyalin credential, token, private key, RTSP password, atau payload production yang belum disanitasi.


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

## Observasi Historis (Non-authoritative) (18 Agu 2026, produksi `<PRODUCTION_TARGET>`)

| # | Skenario | Observasi historis (non-authoritative) |
|---|----------|-------------|
| 1 | Login Admin, buka `/field-maintenance` | NOT RUN |
| 2 | Buka detail catatan pengecekan | NOT RUN |
| 3 | Tombol **Cetak Laporan** (ikon `FileDown`) di footer modal detail | NOT RUN |
| 4 | Klik **Cetak Laporan** | NOT RUN |
| 5 | Buka PDF: kop surat IKN, kode/nama/lokasi/status CCTV, teknisi, tanggal/waktu, status keseluruhan, tabel checklist, foto bukti/URL, catatan umum | NOT RUN |
| 6 | Teknisi buka detail catatan milik sendiri, tombol **Cetak Laporan** tidak tampil | NOT RUN |
| 7 | Teknisi bukan pemilik record | NOT RUN |
| 8 | Endpoint report dengan UUID tidak ada | NOT RUN |
| — | Tanpa sesi valid | NOT RUN |

## Catatan

- Seluruh 8 skenario memenuhi kriteria; 4 unit test backend menutup endpoint ini (`field_maintenance_test.go:461-524` — Success, TechnicianOwner, TechnicianOwnershipDenied, NotFound).
- Teknisi hanya diizinkan mengunduh record miliknya; Admin diizinkan semua record.
- Nama file unduhan mengikuti pola `laporan_pengecekan_lapangan_<uuid>.pdf` sesuai spesifikasi.
