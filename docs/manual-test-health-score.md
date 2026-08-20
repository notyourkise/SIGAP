# Uji Manual - Skor Kesehatan CCTV

> **Kontrak uji formal — MT-09**
>
> Dokumen ini mendefinisikan prosedur; hasil sebelum **2026-08-20** adalah catatan historis/non-authoritative dan harus diuji ulang dengan execution record formal. Inspeksi kode, review unit test, atau checkmark lama bukan `PASS` black-box.
>
> **Skenario audit reproduktif:** Dengan fixture deterministik, clock terkontrol, dan boundary 30/60/90 hari, hitung skor dari setiap faktor lalu uji scheduler dan cleanup.
>
> **Status suite:** `NOT RUN`

### Formal execution metadata

| Field | Value |
|---|---|
| Test ID / Audit ID | MT-09 |
| Capability owner | QA Backend + Backend |
| Status suite | NOT RUN |
| Build SHA | NOT RECORDED |
| Artifact digest/ID | NOT RECORDED |
| Tester | NOT RECORDED |
| Timestamp WITA | NOT RECORDED |
| Environment | NOT RECORDED |
| Fixture prefix/data | `E2E-MT09-` / NOT RECORDED |
| Cleanup result | NOT RUN |
| Expected | Skor API cocok dengan kalkulasi independen, clamp 0–100 dan idempotensi scheduler terbukti, lalu fixture dibersihkan. |
| Actual | NOT RECORDED |
| Evidence path/link | NOT RECORDED |
| Mandatory flag | YES |
| Scenario status | NOT RUN |

### Formal scenario record

| ID | Mandatory | Expected | Actual | Evidence | Status |
|---|---|---|---|---|---|
| MT-09-S1 | YES | Skor API cocok dengan kalkulasi independen, clamp 0–100 dan idempotensi scheduler terbukti, lalu fixture dibersihkan. | NOT RECORDED | NOT RECORDED | NOT RUN |

**Aturan kesimpulan suite:** sebelum eksekusi suite `NOT RUN`; setelah run dimulai, setiap mandatory scenario harus `PASS`. Satu mandatory scenario berstatus `FAIL`, `BLOCKED`, atau `NOT RUN` membuat kesimpulan suite `FAIL`.

> **Guard keselamatan wajib:** Untuk setiap SQL mutasi, `DELETE`/`TRUNCATE`, rotasi/regenerasi key atau QR, penghapusan, reset kredensial, atau mutasi credential, lanjut hanya bila environment bukan production dan target bukan `nexaspace.site`, fixture prefix `E2E-MT09-` serta exact targets sudah dicatat, backup/recovery terverifikasi untuk perubahan data/key state, dan cleanup/rollback didefinisikan serta dicatat setelah run. Jika salah satu syarat tidak tersedia, hentikan prosedur dan set scenario status ke `BLOCKED`; jangan jalankan pada production. Jangan menyalin credential, token, private key, RTSP password, atau payload production yang belum disanitasi.


> **Task:** 6.1 Skor Kesehatan CCTV
> **Branch:** `feature/health-score`
> **Versi:** backend `0.32.0` - frontend `1.28.0`
> **Dokumen:** verifikasi kalkulasi, persistence, recalculation, badge tabel, dan marker peta.

Skor berada pada rentang 0-100. Kalkulator memakai data 90 hari terakhir:

- failed: setiap log `failed` mengurangi `10 x 30%`;
- attention: setiap field-check `attention` mengurangi `10 x 25%`;
- overdue: setiap log `scheduled`/`ongoing` dengan `scheduled_date` terlewat mengurangi `10 x 20%`;
- age: lebih dari 30/60/90 hari sejak `MaintenanceLog completed` terakhir mengurangi 5/10/15 poin sebelum bobot 15%;
- not_ok_items: setiap kelompok 3 item `not_ok` mengurangi 5 poin sebelum bobot 10%.

Penalti dijumlahkan dahulu lalu skor akhir dibulatkan ke integer dan di-clamp ke 0-100.
Jika belum ada pemeliharaan berhasil, umur dianggap 91 hari.

## 1. Prasyarat

| # | Item | Cara menyiapkan |
|---|------|-----------------|
| P1 | Aplikasi berjalan | `docker compose up -d --build`, buka `https://localhost` |
| P2 | Akun admin | Login sebagai admin |
| P3 | Akun teknisi | Siapkan user berperan `teknisi` |
| P4 | CCTV uji | Siapkan satu CCTV dengan nama/kode mudah dikenali |
| P5 | Data histori | Siapkan log pemeliharaan, field-check, dan tanggal jadwal untuk menguji faktor; data uji boleh dibuat melalui UI/Swagger |
| P6 | Dua sesi browser | Admin untuk Master CCTV/peta; teknisi untuk membuat field-check |

## 2. Pengujian Otomatis

```powershell
cd backend
go build ./...
go test ./... -count=1
go vet ./...
swag init

cd ../frontend
npm run lint
npm run build
```

Hasil yang diharapkan: seluruh perintah sukses; lint tidak memiliki error.

**Observasi historis (non-authoritative) (18 Agu 2026):** `go build ./...` NOT RUN, `go test ./... -count=1` NOT RUN 2 test flaky `TestCreateAPIKey_Success` & `TestAPIKeyAuth_ValidKey` — "test: timeout error 1000ms" (batas default 1 detik fiber `app.Test()` saat beban CPU; keduanya **NOT RUN saat dijalankan sendiri** — bukan kegagalan kode; test health-score semua NOT RUN), `go vet ./...` NOT RUN. Frontend: `npm run lint` NOT RUN 0 error (77 warning lama), `npm run build` NOT RUN exit 0. `swag init` tidak dieksekusi (tool tidak terpasang; tidak ada perubahan anotasi).

## 3. Skenario Uji

### S1 - Skor dasar dan endpoint semua role

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Buka Master CCTV dan catat UUID CCTV uji | Data CCTV tampil dengan skor kesehatan dari kolom `health_score` | NOT RUN |
| 2 | Panggil `GET /api/cctv/<UUID>/health-score` melalui Swagger sebagai admin | Respons 200 dengan `health_score`, `breakdown.failed`, `attention`, `overdue`, `age`, `not_ok_items`, dan `updated_at` | NOT RUN |
| 3 | Ulangi sebagai teknisi | Respons tetap 200; endpoint tidak admin-only | NOT RUN |
| 4 | Gunakan UUID tidak valid | Respons 400; tidak ada query kalkulasi | NOT RUN |

### S2 - Faktor failed dan attention

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Buat satu log pemeliharaan CCTV uji berstatus `failed` dalam 90 hari terakhir | `breakdown.failed` bertambah satu | NOT RUN |
| 2 | Buat satu field-check dengan `overall_status = attention` | `breakdown.attention` bertambah satu | NOT RUN |
| 3 | Panggil endpoint skor | Skor turun sesuai bobot: failed 3 poin dan attention 2,5 poin sebelum pembulatan akhir | NOT RUN |
| 4 | Hapus/pisahkan data uji lalu panggil ulang | Faktor kembali tidak menghitung data di luar jendela/yang sudah dihapus | NOT RUN |

### S3 - Faktor overdue dan not_ok items

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Buat log `scheduled` dengan `scheduled_date` satu hari lalu | `breakdown.overdue` bertambah satu | NOT RUN |
| 2 | Buat field-check berisi 3 item `not_ok` dalam 90 hari terakhir | `breakdown.not_ok_items` bertambah tiga | NOT RUN |
| 3 | Panggil endpoint skor | Overdue mengurangi 2 poin; tiga `not_ok` mengurangi 0,5 poin sebelum pembulatan | NOT RUN |
| 4 | Ubah log menjadi `completed` atau pindahkan jadwal ke masa depan | Log tidak lagi dihitung sebagai overdue pada kalkulasi berikutnya | NOT RUN |

### S4 - Faktor umur pemeliharaan berhasil

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Siapkan log `completed` terakhir 30 hari lalu | `breakdown.age` = 30; belum ada penalti umur | NOT RUN |
| 2 | Ubah tanggal menjadi 31 hari lalu | Penalti umur = 5 x 15%; skor turun setelah pembulatan | NOT RUN |
| 3 | Uji tanggal 61 dan 91 hari lalu | Penalti berubah ke 10 x 15% dan 15 x 15% | NOT RUN |
| 4 | Gunakan CCTV tanpa riwayat completed | `breakdown.age` = 91; kalkulator menerapkan kategori >90 hari | NOT RUN |

### S5 - Batas skor 0-100

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Siapkan banyak log failed/overdue, field-check attention, dan item not_ok | Nilai penalti melebihi 100 | NOT RUN |
| 2 | Panggil endpoint | `health_score` = 0, tidak boleh negatif | NOT RUN |
| 3 | Gunakan CCTV baru tanpa faktor negatif selain umur default | Skor tetap dalam rentang 0-100 dan nilai awal database adalah 100 sebelum kalkulator berjalan | NOT RUN |

### S6 - Recalculation on-demand dan scheduler 6 jam

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Catat skor CCTV, lalu selesaikan maintenance melalui checklist 9 item | Setelah transaksi selesai, kalkulator dipanggil; `health_score` diperbarui | NOT RUN |
| 2 | Buat field-check baru dengan hasil `attention` atau `ok` | Skor dihitung ulang setelah record tersimpan | NOT RUN |
| 3 | Restart API | Kalkulator menjalankan perhitungan seluruh CCTV saat startup | NOT RUN |
| 4 | Periksa log API | Terdapat proses kalkulator tanpa menghalangi server Fiber; ticker berikutnya berjalan 6 jam kemudian | NOT RUN |

### S7 - Badge skor pada tabel Master CCTV

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Buka `/cctv` sebagai admin atau teknisi | Kolom **Kesehatan** tampil di tabel | NOT RUN |
| 2 | Amati tiga CCTV dengan skor 70+, 40-69, dan <40 | Badge angka memakai hijau, kuning, dan merah; angka tetap terbaca di samping warna | NOT RUN |
| 3 | Buka Sortir & Filter -> pilih **Skor kesehatan** | Backend menerima `sort_by=health_score`; baris terurut sesuai skor | NOT RUN |
| 4 | Buka layar mobile dan geser tabel horizontal | Kolom Kesehatan tetap dapat dibaca tanpa merusak layout | NOT RUN |

### S8 - Warna marker dan detail skor di peta

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Buka `/sebaran-cctv` | Marker CCTV mengambil warna dari skor, bukan hanya status operasional | NOT RUN |
| 2 | Klik marker hijau/kuning/merah | Popup menampilkan status operasional dan badge `Skor <angka>` | NOT RUN |
| 3 | Lihat tooltip marker | Nama, kode CCTV, dan `Kesehatan: <angka>/100` tampil | NOT RUN |
| 4 | Periksa legend peta | Legend menjelaskan `>=70 Sehat`, `40-69 Perhatian`, `<40 Kritis` | NOT RUN |

## 4. Kriteria Penerimaan

1. Model memiliki `health_score` default 100 dan list CCTV mengirim field tersebut.
2. Endpoint protected tersedia untuk admin dan teknisi dengan breakdown lima faktor.
3. Semua faktor hanya menghitung data 90 hari sesuai definisi dan overdue hanya status aktif.
4. Umur dihitung dari MaintenanceLog completed terakhir; tanpa histori memakai 91 hari.
5. Skor akhir dibulatkan dan selalu berada pada 0-100.
6. Scheduler berjalan saat startup dan tiap 6 jam; complete/field-check memicu recalculation.
7. Badge tabel dan marker peta mengikuti ambang 70/40, dengan angka/label sebagai cue tambahan.

## Catatan

- `updated_at` pada respons adalah waktu kalkulasi, bukan waktu record histori terakhir.
- Marker peta tetap menampilkan status operasional pada popup; warna marker khusus untuk kesehatan.
- Data uji sebaiknya dipisahkan atau dihapus setelah verifikasi agar skor operasional tidak berubah permanen.
