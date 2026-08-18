# Uji Manual - Skor Kesehatan CCTV

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

**Hasil nyata (18 Agu 2026):** `go build ./...` ✅, `go test ./... -count=1` ⚠️ 2 test flaky `TestCreateAPIKey_Success` & `TestAPIKeyAuth_ValidKey` — "test: timeout error 1000ms" (batas default 1 detik fiber `app.Test()` saat beban CPU; keduanya **PASS saat dijalankan sendiri** — bukan kegagalan kode; test health-score semua PASS), `go vet ./...` ✅. Frontend: `npm run lint` ✅ 0 error (77 warning lama), `npm run build` ✅ exit 0. `swag init` tidak dieksekusi (tool tidak terpasang; tidak ada perubahan anotasi).

## 3. Skenario Uji

### S1 - Skor dasar dan endpoint semua role

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buka Master CCTV dan catat UUID CCTV uji | Data CCTV tampil dengan skor kesehatan dari kolom `health_score` | ✅ LULUS — CCTV uji `63c6e479-5a8a-4f16-ad4c-254a16ecc380` (MT-CCTV-001) tampil dengan `health_score=75` di list CCTV |
| 2 | Panggil `GET /api/cctv/<UUID>/health-score` melalui Swagger sebagai admin | Respons 200 dengan `health_score`, `breakdown.failed`, `attention`, `overdue`, `age`, `not_ok_items`, dan `updated_at` | ✅ LULUS — 200: `{"health_score":75,"breakdown":{"failed":3,"attention":3,"overdue":4,"age":0,"not_ok_items":3},"updated_at":"2026-08-18T12:49:38+08:00"}` |
| 3 | Ulangi sebagai teknisi | Respons tetap 200; endpoint tidak admin-only | ✅ LULUS — teknisi.test → 200 dengan payload identik |
| 4 | Gunakan UUID tidak valid | Respons 400; tidak ada query kalkulasi | ✅ LULUS — `not-a-uuid` → 400 "ID CCTV tidak valid" |

### S2 - Faktor failed dan attention

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buat satu log pemeliharaan CCTV uji berstatus `failed` dalam 90 hari terakhir | `breakdown.failed` bertambah satu | ✅ LULUS — ada 3 log failed dalam 90 hari (data uji Aug 2026: `68e97653`, `1f2039a2`, dll) → `failed=3` |
| 2 | Buat satu field-check dengan `overall_status = attention` | `breakdown.attention` bertambah satu | ✅ LULUS — `attention=3` (record attention uji: `a5d73d43`, `48d5bf9c`, `7a8705df`) |
| 3 | Panggil endpoint skor | Skor turun sesuai bobot: failed 3 poin dan attention 2,5 poin sebelum pembulatan akhir | ✅ LULUS — kalkulasi nyata: penalti = 3×10×30% + 3×10×25% + 4×10×20% + 0 + (3/3×5)×10% = 9+7.5+8+0.5 = **25** → 100−25 = **75** (persis respons) |
| 4 | Hapus/pisahkan data uji lalu panggil ulang | Faktor kembali tidak menghitung data di luar jendela/yang sudah dihapus | ✅ LULUS (verifikasi kode) — query hanya menghitung `completed_at/created_at` dalam 90 hari terakhir & `deleted_at IS NULL` (`health_score.go:172-199`); soft-deleted log diabaikan |

### S3 - Faktor overdue dan not_ok items

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buat log `scheduled` dengan `scheduled_date` satu hari lalu | `breakdown.overdue` bertambah satu | ✅ LULUS — `overdue=4` (log scheduled Aug 2026 yang jadwalnya terlewat: `d5046e42`, `dde660ab`, dll) |
| 2 | Buat field-check berisi 3 item `not_ok` dalam 90 hari terakhir | `breakdown.not_ok_items` bertambah tiga | ✅ LULUS — `not_ok_items=3` |
| 3 | Panggil endpoint skor | Overdue mengurangi 2 poin; tiga `not_ok` mengurangi 0,5 poin sebelum pembulatan | ✅ LULUS — 4 overdue = 8 poin; 1 kelompok (3/3) = 0,5 poin → total 25 → skor 75 (lihat S2.3) |
| 4 | Ubah log menjadi `completed` atau pindahkan jadwal ke masa depan | Log tidak lagi dihitung sebagai overdue pada kalkulasi berikutnya | ✅ LULUS (verifikasi kode) — query overdue hanya `status IN ('scheduled','ongoing')` dengan `scheduled_date < now` (`health_score.go:184-187`); status terminal tidak dihitung |

### S4 - Faktor umur pemeliharaan berhasil

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Siapkan log `completed` terakhir 30 hari lalu | `breakdown.age` = 30; belum ada penalti umur | ✅ LULUS (verifikasi kode + data) — CCTV uji `age=0` karena punya completed < 30 hari (log `c5a1a67d` 12 Agu dll); penalti umur hanya bila `ageDays > 30` (`health_score.go:213-221`) |
| 2 | Ubah tanggal menjadi 31 hari lalu | Penalti umur = 5 x 15%; skor turun setelah pembulatan | ✅ LULUS (verifikasi kode + unit test) — `ageDays > 30 → agePenalty=5`; bobot 5×15%=0,75 poin; tidak diubah data prod (butuh akses DB prod yang tidak tersedia non-interaktif) |
| 3 | Uji tanggal 61 dan 91 hari lalu | Penalti berubah ke 10 x 15% dan 15 x 15% | ✅ LULUS (verifikasi kode) — `>60 → 10`, `>90 → 15` (`:215-220`) |
| 4 | Gunakan CCTV tanpa riwayat completed | `breakdown.age` = 91; kalkulator menerapkan kategori >90 hari | ✅ LULUS (verifikasi kode) — `ageDays := 91` default bila `LastCompletedAt` tidak valid (`:205-211`); unit test menutup jalur ini |

### S5 - Batas skor 0-100

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Siapkan banyak log failed/overdue, field-check attention, dan item not_ok | Nilai penalti melebihi 100 | ✅ LULUS (verifikasi kode) — penalti dijumlahkan bebas tanpa batas atas (`:225-229`) |
| 2 | Panggil endpoint | `health_score` = 0, tidak boleh negatif | ✅ LULUS (verifikasi kode) — `if score < 0 { score = 0 }` (`:231-233`) + `>100 → 100` (`:234-236`); unit test `TestCalculateHealthScore_*` menutup clamp |
| 3 | Gunakan CCTV baru tanpa faktor negatif selain umur default | Skor tetap dalam rentang 0-100 dan nilai awal database adalah 100 sebelum kalkulator berjalan | ✅ LULUS (verifikasi kode) — kolom `health_score` default **100** (migrasi skema); skor akhir selalu di-round & di-clamp |

### S6 - Recalculation on-demand dan scheduler 6 jam

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Catat skor CCTV, lalu selesaikan maintenance melalui checklist 9 item | Setelah transaksi selesai, kalkulator dipanggil; `health_score` diperbarui | ✅ LULUS (verifikasi kode) — `CompleteMaintenanceLog` memanggil `RecalculateCCTVHealthScore` setelah transaksi penyelesaian (`maintenance.go:789-793`) |
| 2 | Buat field-check baru dengan hasil `attention` atau `ok` | Skor dihitung ulang setelah record tersimpan | ✅ LULUS — **teruji nyata**: sebelum 75 (attention=3); buat field-check attention baru (3× not_ok) → skor langsung **72** = 100 − (9 + 10 + 8 + 2×5×10%) persis sesuai bobot; pemicu `field_maintenance.go:269` |
| 3 | Restart API | Kalkulator menjalankan perhitungan seluruh CCTV saat startup | ✅ LULUS (verifikasi kode) — `RunHealthScoreCalculator` menjalankan `CalculateAllCCTVHealthScoresAt` saat startup (`health_score.go:51-57`), dipanggil di `main.go` |
| 4 | Periksa log API | Terdapat proses kalkulator tanpa menghalangi server Fiber; ticker berikutnya berjalan 6 jam kemudian | ✅ LULUS (verifikasi kode) — goroutine terpisah dengan `ticker := time.NewTicker(6 * time.Hour)` (`:59-64`); tiap siklus juga cleanup histori > 1 tahun (`:314-322`) |

### S7 - Badge skor pada tabel Master CCTV

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buka `/cctv` sebagai admin atau teknisi | Kolom **Kesehatan** tampil di tabel | ✅ LULUS (verifikasi kode) — kolom "Kesehatan" dengan `SortableHeader sortKey="health_score"` (`cctv/page.tsx:1265-1267`) |
| 2 | Amati tiga CCTV dengan skor 70+, 40-69, dan <40 | Badge angka memakai hijau, kuning, dan merah; angka tetap terbaca di samping warna | ✅ LULUS (verifikasi kode) — `healthScoreClasses`: ≥70 emerald, 40-69 amber, <40 merah (`cctv/page.tsx:427-431`); badge menampilkan **angka** (`:1339-1343`) + `title`/`aria-label` |
| 3 | Buka Sortir & Filter -> pilih **Skor kesehatan** | Backend menerima `sort_by=health_score`; baris terurut sesuai skor | ✅ LULUS — `GET /api/cctv?sort_by=health_score&order=asc` → 200, MT-CCTV-001 (75) terurut; dropdown filter "Skor kesehatan" ada (`cctv/page.tsx:1487`), sort default `desc` untuk health (`:996`) |
| 4 | Buka layar mobile dan geser tabel horizontal | Kolom Kesehatan tetap dapat dibaca tanpa merusak layout | ✅ LULUS (verifikasi kode) — tabel dalam kontainer scroll horizontal (`overflow-x`); badge kompak `min-w-14` tetap terbaca |

### S8 - Warna marker dan detail skor di peta

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buka `/sebaran-cctv` | Marker CCTV mengambil warna dari skor, bukan hanya status operasional | ✅ LULUS (verifikasi kode) — `createCustomMarkerIcon(item.health_score, ...)` (`CCTVMap.tsx:366`); warna marker `≥70 → #10B981`, `40-69 → amber`, `<40 → merah` (`:61-66`) |
| 2 | Klik marker hijau/kuning/merah | Popup menampilkan status operasional dan badge `Skor <angka>` | ✅ LULUS (verifikasi kode) — popup menampilkan status + badge "Skor {n}" (`:402-403`) |
| 3 | Lihat tooltip marker | Nama, kode CCTV, dan `Kesehatan: <angka>/100` tampil | ✅ LULUS (verifikasi kode) — tooltip "Kesehatan: {n}/100" (`:379`) |
| 4 | Periksa legend peta | Legend menjelaskan `>=70 Sehat`, `40-69 Perhatian`, `<40 Kritis` | ✅ LULUS (verifikasi kode) — legend "≥70 Sehat / 40–69 Perhatian / <40 Kritis" (`:438-453`) |

## 4. Kriteria Lulus

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
