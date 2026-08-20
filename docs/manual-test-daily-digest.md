# Uji Manual — Ringkasan Operasional Otomatis (Task 6.4)

> **Kontrak uji formal — MT-04**
>
> Dokumen ini mendefinisikan prosedur; hasil sebelum **2026-08-20** adalah catatan historis/non-authoritative dan harus diuji ulang dengan execution record formal. Inspeksi kode, review unit test, atau checkmark lama bukan `PASS` black-box.
>
> **Skenario audit reproduktif:** Dengan lebih dari 10 digest fixture dan perhitungan expected independen, jalankan generator pada waktu WITA, ulangi proses, uji RBAC, dan baca halaman kedua.
>
> **Status suite:** `NOT RUN`

### Formal execution metadata

| Field | Value |
|---|---|
| Test ID / Audit ID | MT-04 |
| Capability owner | QA Backend + Backend |
| Status suite | NOT RUN |
| Build SHA | NOT RECORDED |
| Artifact digest/ID | NOT RECORDED |
| Tester | NOT RECORDED |
| Timestamp WITA | NOT RECORDED |
| Environment | NOT RECORDED |
| Fixture prefix/data | `E2E-MT04-` / NOT RECORDED |
| Cleanup result | NOT RUN |
| Expected | Tepat satu digest per tanggal WITA dibuat, catch-up tidak menggandakan data, angka cocok dengan sumber, dan akses/page 2 terbukti. |
| Actual | NOT RECORDED |
| Evidence path/link | NOT RECORDED |
| Mandatory flag | YES |
| Scenario status | NOT RUN |

### Formal scenario record

| ID | Mandatory | Expected | Actual | Evidence | Status |
|---|---|---|---|---|---|
| MT-04-S1 | YES | Tepat satu digest per tanggal WITA dibuat, catch-up tidak menggandakan data, angka cocok dengan sumber, dan akses/page 2 terbukti. | NOT RECORDED | NOT RECORDED | NOT RUN |

**Aturan kesimpulan suite:** sebelum eksekusi suite `NOT RUN`; setelah run dimulai, setiap mandatory scenario harus `PASS`. Satu mandatory scenario berstatus `FAIL`, `BLOCKED`, atau `NOT RUN` membuat kesimpulan suite `FAIL`.

> **Guard keselamatan wajib:** Untuk setiap SQL mutasi, `DELETE`/`TRUNCATE`, rotasi/regenerasi key atau QR, penghapusan, reset kredensial, atau mutasi credential, lanjut hanya bila environment bukan production dan target bukan `nexaspace.site`, fixture prefix `E2E-MT04-` serta exact targets sudah dicatat, backup/recovery terverifikasi untuk perubahan data/key state, dan cleanup/rollback didefinisikan serta dicatat setelah run. Jika salah satu syarat tidak tersedia, hentikan prosedur dan set scenario status ke `BLOCKED`; jangan jalankan pada production. Jangan menyalin credential, token, private key, RTSP password, atau payload production yang belum disanitasi.


## Prasyarat

- Backend dan frontend berjalan melalui konfigurasi development atau Docker.
- Tersedia akun Admin dan data tugas, pengecekan lapangan, serta perubahan status CCTV.
- Browser sudah login ke SIGAP.

## Skenario

1. Login sebagai Admin lalu pastikan menu **Ringkasan** dengan ikon `ClipboardList` tampil di sidebar. NOT RUN
2. Buka `/ringkasan` dan pastikan timeline menampilkan kartu per tanggal jika digest sudah tersedia. NOT RUN
3. Periksa setiap kartu: tanggal, summary naratif, tugas baru, selesai, gagal, overdue, pengecekan lapangan, jumlah attention, dan perubahan status CCTV. NOT RUN
4. Jika tersedia lebih dari 10 digest, gunakan paginasi dan pastikan halaman berikutnya memuat tanggal yang lebih lama. NOT RUN sebagian — komponen & API paginasi tersedia, tapi data nyata baru 4 digest sehingga halaman ke-2 tidak dapat diuji (terbatas data).
5. Jalankan atau tunggu generator pada pukul 06:00 WITA. Pastikan digest tanggal tersebut dibuat satu kali. NOT RUN
6. Pastikan Admin menerima notifikasi **Ringkasan operasional [tanggal] tersedia** pada menu Pemberitahuan. NOT RUN
7. Login sebagai Teknisi. Pastikan menu Ringkasan tidak tampil dan membuka `/ringkasan` mengarahkan kembali ke `/dashboard`. NOT RUN

## Hasil yang Diharapkan

- Generator hanya membuat satu digest untuk setiap `digest_date`. NOT RUN
- Ringkasan memakai data 24 jam terakhir dan menampilkan angka yang konsisten dengan aktivitas operasional. NOT RUN
- Endpoint digest hanya dapat diakses Admin. NOT RUN
