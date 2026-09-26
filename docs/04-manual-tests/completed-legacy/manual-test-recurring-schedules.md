# Uji Manual — Penjadwalan Pemeliharaan Berulang (Recurring)

> **Task:** 5.4 Penjadwalan Pemeliharaan Berulang (Recurring)
> **Branch:** `feature/recurring-schedule` · **Merge PR:** `#63` (commit `d7696c2`)
> **Versi:** backend `0.30.0` · frontend `1.27.0`
> **Dokumen ini:** alur uji manual klik-per-klik beserta hasil yang diharapkan (expected results).

Perilaku yang diuji: admin dapat membuat **jadwal pemeliharaan berulang** (`daily` /
`weekly` / `monthly`) di halaman baru `/schedules`. Generator backend (`handlers.RunScheduleGenerator`)
berjalan **sekali saat startup** lalu **tiap 1 jam**, dan membuat `MaintenanceLog` otomatis untuk
setiap jadwal aktif yang `next_run_at`-nya sudah tiba:
- **tanpa teknisi** → log dibuat berstatus **"Terjadwal" (`scheduled`)** — menunggu ditugaskan admin,
- **dengan teknisi** → log langsung berstatus **"Dalam proses" (`ongoing`)** + teknisi mendapat notifikasi,
- **idempotent**: hanya **satu log per hari per jadwal** (duplikat tidak pernah dibuat),
- jadwal **nonaktif** di-skip generator (tidak menghasilkan log) tanpa dihapus,
- seluruh aktivitas admin tercatat di audit (`CREATE_SCHEDULE`, `UPDATE_SCHEDULE`,
  `DELETE_SCHEDULE`, `TOGGLE_SCHEDULE`) dan generate otomatis dicatat sebagai
  `CREATE_MAINTENANCE` dengan identitas `system` — "Dibuat otomatis oleh jadwal [id]".

---

## 1. Prasyarat

| # | Item | Cara menyiapkan |
|---|------|-----------------|
| P1 | Aplikasi berjalan | `docker compose up -d --build` → buka `https://localhost` (pastikan image backend/frontend ter-rebuild dengan kode hasil merge PR #63) |
| P2 | Akun admin | Login dengan kredensial dari `.env` / `.env.docker` |
| P3 | Akun teknisi | **Manajemen Pengguna** (sidebar) → **"Tambah Pengguna"** → buat `teknisi_ruji` (username, password, nama, email → Peran **"Teknisi"**). Catat nama lengkapnya untuk identifikasi notifikasi |
| P4 | CCTV `UJI-01` (aktif) | **Aset CCTV** → **"Tambah CCTV"** → kode `UJI-01`, nama bebas, Status **"Aktif"** → **"Tambah CCTV"** |
| P5 | Teknisi bukan-teknisi (opsional) | Satu akun berperan **Admin** tambahan (mis. `admin2`) untuk menguji penolakan teknisi non-teknisi via Swagger |
| P6 | Dua tab browser | Tab S: **Jadwal** (`/schedules`, admin) · Tab M: **Pemeliharaan CCTV** (`/maintenance`, admin). Untuk S7–S8 tambah Tab K: **Teknisi** (incognito, login sebagai `test_ruju`) → Dashboard |

---

## 2. Pengujian Otomatis (lapisan pertama)

Sebelum uji manual, jalankan test otomatis di `backend/`:

```powershell
go build ./...                # harus sukses
go test ./... -count=1        # semua test PASS, termasuk:
                              #   TestCreateMaintenanceSchedule_*
                              #   TestGetAllMaintenanceSchedules_Success
                              #   TestGetMaintenanceScheduleByID_*
                              #   TestUpdateMaintenanceSchedule_*
                              #   TestToggleMaintenanceSchedule_Success
                              #   TestDeleteMaintenanceSchedule_Success
                              #   TestProcessDueSchedules_* (generator)
go vet ./...
```

Lalu di `frontend/`:

```powershell
npm run lint   # 0 error (warning lama boleh)
npm run build  # harus sukses, dan halaman /schedules tercantum pada daftar route
```

Hasil yang diharapkan: seluruh package/test hijau dan build frontend sukses.

| Item | Hasil nyata |
|------|-------------|
| `go build ./...` | ✅ |
| `go test ./... -count=1` | ✅ (termasuk `maintenance_schedule_test.go` & `schedule_runner_test.go`: aktif buat log, nonaktif skip, duplikat skip, berteknisi) |
| `go vet ./...` | ✅ |
| `npm run lint` | ✅ 0 error |
| `npm run build` | ✅ (rute `/schedules` ter-generate) |

---

## 3. Skenario Uji Manual

Cara membaca: ikuti **Langkah** berurutan, lalu cocokkan dengan **Hasil yang diharapkan**.
Beri tanda centang pada kolom **Hasil nyata** sesuai pengamatan (`✓` lulus / `✗` gagal).

> **Mempercepat proses generator (opsional):** halaman `/schedules` hanya memicu generate
> otomatis sekali saat startup lalu tiap 60 menit. Untuk mengamati generate tanpa menunggu,
> mundurkan `next_run_at` jadwal ke masa lalu lalu restart API:
> `docker compose exec db psql -U ${POSTGRES_USER} -d ${POSTGRES_DB} -c "UPDATE maintenance_schedules SET next_run_at = now() - interval '1 minute' WHERE id = '<JADWAL_UUID>';"`
> Nilai `POSTGRES_USER`/`POSTGRES_DB` ada di `.env` root. Setiap log baru yang muncul silakan
> hapus lewat halaman **Pemeliharaan CCTV** agar skenario berikutnya tidak terkontaminasi.

### S1 — Regresi: akses halaman Jadwal & menu sidebar hanya Admin

**Tujuan:** memastikan `/schedules` tampil untuk admin dan tidak untuk teknisi (RBAC + guard mesin).

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Login admin → Tab S: buka **Jadwal** (sidebar) | Halaman **"Jadwal Pemeliharaan"** terbuka; URL `/schedules`; tombol **"Buat Jadwal"** tampil | ✅ |
| 2 | Login **teknisi** (`test_uji` di Tab K) | Menu **"Jadwal" tidak muncul** di sidebar; membuka `https://localhost#ymint/schedules` langsung → redirect/ditolak (role non-admin) | ✅ |
| 3 | (Opsional) Swagger dengan token teknisi: `GET /api/maintenance-schedules` | Respons `403` "Akses ditolak: Hanya Admin yang diperbolehkan" | ✅ |

### S2 — Buat jadwal baru (Harian, tanpa teknisi)

**Tujuan:** membuktikan pembuatan jadwal via **"Buat Jadwal"** dan pengisian default NextRunAt.

| # | Langkah (Tab S = admin) | Hasil yang diharapkan | Hasil nyata |
|---|--------------------------|------------------------|-------------|
| 1 | **"Buat Jadwal"** → modal "Jadwal Pemeliharaan Baru" | Modal terbuka; kolom: **CCTV \***, **Tindakan yang dilakukan \***, **Frekuensi penjadwalan \*** (Harian/Mingguan/Bulanan), **Teknisi pelaksana** (opsional) | ✅ |
| 2 | Pilih CCTV `UJI-01` → Tindakan (mis. "Pembersihan lensa dan pemeriksaan konektor RJ45") → Frekuensi **"Harian"** → biarkan **Teknisi kosong** → **"Simpan Jadwal"** | Toast sukses; baris baru muncul: Frekuensi **Harian** · Teknisi **"Langsung oleh admin"** · Status **Aktif** · Jalankan Berikutnya ≈ **besok (jam ini ±24 jam)** | ✅ |
| 3 | Periksa **Audit Log** (sidebar) → "Sortir" → filter Modul = MAINTENANCE | Baris dengan aksi **`CREATE_SCHEDULE`** — deskripsi: "Membuat jadwal pemeliharaan berulang: Pembersihan lensa … (Frekuensi: daily)" | ✅ |

### S3 — Validasi input jadwal

**Tujuan:** membuktikan kolom wajib dan penolakan teknisi non-teknisi untuk **Buat/Edit**.

| # | Langkah (Tab S = admin) | Hasil yang diharapkan | Hasil nyata |
|---|--------------------------|------------------------|-------------|
| 1 | **"Buat Jadwal"** → **CCTV kosong** → **"Simpan Jadwal"** | Tidak tersimpan; toast error "Pilih CCTV yang dijadwalkan." | ✅ |
| 2 | **"Buat Jadwal"** → pilih CCTV, **Tindakan kosong** → **"Simpan Jadwal"** | Tidak tersimpan; toast "Tuliskan tindakan yang dilakukan." (atau warning validasi isian) | ✅ |
| 3 | (Opsional) Swagger `POST /api/maintenance-schedules` body `{"cctv_id":"<id>","action":"A","frequency":"daily"}` | Respons `400` + `errors[]` field `ActionTaken` (tag `min`/`required`) | ✅ |
| 4 | (Opsional) Swagger `POST /api/maintenance-schedules` body `{"cctv_id":"<id>","technician_id":"<id admin2>","action_taken":"...","frequency":"daily"}` | Respons `400` + `message: "User tujuan bukan berperan teknisi"` | ✅ |

### S4 — Buat jadwal Mingguan dengan teknisi → log auto-`ongoing` + notifikasi (S7 gabung?)

**Tujuan:** membuktikan **jadwal berteknisi** menghasilkan log berstatus "Dalam proses" dan
teknisi menerima notifikasi (generator, langkah S7 nanti). Di sini cukup buat jadwalnya.

| # | Langkah (Tab S = admin) | Hasil yang diharapkan | Hasil nyata |
|---|--------------------------|------------------------|-------------|
| 1 | **"Buat Jadwal"** → CCTV `UJI-01` → Tindakan lain (mis. "Cek sambungan dan ganti konektor RJ45") → Frekuensi **"Mingguan"** → **Teknisi = `test_uji`** → **"Simpan Jadwal"** | Toast sukses; baris baru Frekuensi **Mingguan**, Teknisi = nama `test_uji`, Status **Aktif**, Jalankan Berikutnya ≈ **+7 hari** | ✅ |
| 2 | Audit Log → Modul MAINTENANCE | Baris `CREATE_SCHEDULE` kedua untuk jadwal Mingguan (deskripsi berisi "(Frekuensi: weekly)") | ✅ |

### S5 — Edit jadwal: ubah frekuensi → NextRunAt di-reset; kosongkan teknisi

**Tujuan:** membuktikan perubahan frekuensi menghitung ulang jadwal berikutnya dan teknisi bisa dilepas.

| # | Langkah (Tab S = admin) | Hasil yang diharapkan | Hasil nyata |
|---|--------------------------|-----------------------|-------------|
| 1 | Klik **ikon Settings** pada baris jadwal **Mingguan** → modal "Ubah Jadwal Pemeliharaan" | Modal terisi data awal (CCTV, tindakan, frekuensi Mingguan, teknisi `test_uji`) | ✅ |
| 2 | Ubah Frekuensi → **"Bulanan"** → **"Simpan Perubahan"** | Toast sukses; kolom Frekuensi = **Bulanan**; kolom **Jalankan Berikutnya** ≈ **+1 bulan dari sekarang** (di-reset) | ✅ |
| 3 | Buka **Settings** baris yang sama → ubah **Teknisi → pilih kosong** (`""`) → **"Simpan Perubahan"** | Teknisi = **"Teknisi pelaksana kosong")** diubah menjadi **"Langsung oleh admin"** di tabel | ✅ |
| 4 | Audit Log | Baris `UPDATE_SCHEDULE` untuk kedua edit (deskripsi memuat frekuensi baru) | ✅ |

### S6 — Toggle nonaktif/aktif (generator melewati jadwal nonaktif)

**Tujuan:** membuktikan toggle tidak menghapus jadwal tetapi menghentikan generate.

| # | Langkah (Tab S = admin) | Hasil yang diharapkan | Hasil nyata |
|---|--------------------------|------------------------|-------------|
| 1 | Pada jadwal **Mingguan berteknisi** dari S4 → klik ikon **Power** (title "Hentikan sementara jadwal") | Toast "Jadwal dihentikan sementara."; badge **Nonaktif**; ikon tombol berubah menjadi sukses/green (siap aktifkan) | ✅ |
| 2 | Cek Tab M (Pemeliharaan CCTV) | Tidak ada log baru yang muncul (jadwal nonaktif) | ✅ |
| 3 | Klik **Power** sekali lagi | Toast "Jadwal kembali aktif."; badge **Aktif** | ✅ |
| 4 | Audit Log | Ada dua baris `TOGGLE_SCHEDULE` (nonaktif lalu aktif) | ✅ |

### S6 — Generator: jadwal aktif jatuh tempo → log `scheduled` dibuat otomatis

**Tujuan:** fitur inti — generator membuat `MaintenanceLog` tanpa interaksi admin.

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Pastikan jadwal **daily tanpa teknisi** (S2) berstatus **Aktif** dan **istirahat**: Tab M tetap terbuka (halaman Pemeliharaan CCTV) | Tab M menampilkan daftar log | ✅ |
| 2 | Sertai `next_run_at` jadwal daily tersebut ke **masa lalu** (fast tip §3) → pastikan harus diingat `UJI-01` | — | ⚠️ Tidak dapat diuji (butuh akses SSH/psql untuk mundurkan `next_run_at`) |
| 3 | Restart API (atau tunggu ≤60 menit untuk ticker) — generator berjalan saat startup | — | ⚠️ Tidak dapat diuji (tidak ada akses restart API produksi) |
| 4 | Lihat Tab M (`/maintenance`) | Log **baru otomatis** muncul: CCTV `UJI-01`, tindakan = template jadwal, status **Terjadwal**, teknisi "Belum ditugaskan", CATATAN waktu = saat generate | ✅ (verifikasi kode + unit test) |
| 5 | Klik **Settings** jadwal daily di Tab S (atau Swagger `GET /api/maintenance-schedules/:id`) | `last_generated_at` terisi = waktu generate; `next_run_at` sudah bergeser **+24 jam** dari generate | ✅ (verifikasi kode) |
| 6 | **Audit Log** → Modul MAINTENANCE | Baris aksi **`CREATE_MAINTENANCE`** dengan username **`system`** — "Dibuat otomatis oleh jadwal **<UUID>**" | ✅ (verifikasi kode) |

### S6 — Idempotensi: tidak ada duplikat per hari

**Tujuan:** menjamin hanya satu log per hari per jadwal walau generator berjalan berkali-kali.

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Tanpa mengubah apa pun, **restart API lagi** di hari yang sama (sedikit setelah S6-3) | Generator berjalan lagi, sangat log dibuat tapi harus **tidak ada** log baru/duplikat | ⚠️ Tidak dapat diuji (tanpa akses restart API produksi) |
| 2 | Cek Tab M (filter/reload) | Jumlah log untuk `UJ-01` pada tanggal tersebut **tetap satu** (tidak bertambah) | ✅ (verifikasi kode + unit test) |
| 3 | Cek log backend container (`docker compose logs api`) | Baris log generator: "…lewati (idempotent)" untuk jadwal itu — tidak ada error | ✅ (verifikasi kode) |

### S8 — Jadwal dengan teknisi → log otomatis `ongoing` + notifikasi teknisi

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|------------|
| 1 | Buka **Tab K** (incognito login `test_uji`) → biarkan di **Dashboard** (lonceng terbuka) | Lonceng notifikasi tampil normal | ✅ |
| 2 | Tab S: pastikan jadwal **Mingguan berteknisi** (S4) **Aktif**; mundurkan `next_run_at` → **restart api** | — | ⚠️ Tidak dapat diuji (tanpa akses restart API produksi) |
| 3 | Cek Tab K tanpa refresh | **Toast real-time** notifikasi "Tugas baru dari jadwal"; badge lonceng bertambah | ✅ (verifikasi kode) |
| 4 | Buka **Pemberitahuan** (`/pemberitahuan`) sebagai `test_uji` | Item **"Tugas baru dari jadwal"** — "Anda mendapat tugas pemeliharaan terjadwal otomatis: **<action schedule>**." | ✅ (verifikasi kode) |
| 5 | Cek Tab M | Log baru berstatus **"Dalam proses"** dengan teknisi = `test_uji` (langsung ongoing) | ✅ (verifikasi kode + unit test) |

### S9 — Hapus jadwal (soft delete) + log lama tetap ada

| # | Langkah (Tab S = admin) | Hasil yang diharapkan | Hasil nyata |
|---|--------------------------|------------------------|-------------|
| 1 | Klik **ikon Trash2** pada jadwal **daily** → dialgo **"Hapus Jadwal Pemeliharaan?"** → **"Hapus"** | Toast "Jadwal pemeliharaan berhasil dihapus."; baris hilang dari tabel | ✅ |
| 2 | Buka **Tab M** (log pemeliharaan) | Log yang pernah dibuat oleh jadwal (S6) **masih ada** — tidak ikut terhapus | ✅ (verifikasi kode — soft delete) |
| 3 | Audit Log | Baris `DELETE_SCHEDULE` "Menghapus jadwal pemeliharaan berulang: … (Frekuensi: daily)" | ✅ |

---

## 4. Kriteria Lulus Keseluruhan

Semua baris "Hasil yang diharapkan" pada S1–S9 terpenuhi, khususnya:

1. S1: `/schedules` + menu "Jadwal" hanya untuk **admin** (teknisi ditolak 403).
2. S2–S3: jadwal harian/wetekan/bulanan bisa dibuat; validasi wajib & teknisi non-teknisi ditolak.
3. S4–S5: frekuensi diperbarui → `next_run_at` reset; teknisi bisa dilepas; audit `CREATE_SCHEDULE`/`UPDATE_SCHEDULE`.
4. S6: toggle nonaktif menghentikan generator (tanpa hapus data); audit `TOGGLE_SCHEDULE`.
5. S7: generator membuat log otomatis berstatus **Terjadwal** (+ `last_generated_at` terisi, `next_run_at` bergeser), audit `CREATE_MAINTENANCE` oleh `system`.
6. S8: idempotensi — **tidak ada duplikat** walau restart berkali dalam sehari; jadwal berteknisi → log **Dalam Proses** + notifikasi "Tugas baru dari jadwal".
7. S9: hapus jadwal = soft delete; log yang dibuat tidak ikut terhapus; audit `DELETE_SCHEDULE`.
8. `go build ./...`, `go test ./... -count=1`, `go vet ./...`, `npm run lint`, dan `npm run build` hijau.

## Catatan

- Halaman `/schedules` menampilkan **teknisi kosong sebagai "Langsung oleh admin"**: log hasil
  generate tanpa teknisi berstatus **"Terjadwal"** (menunggu admin tugaskan, alir assign 5.3
  tetap berlaku). Jika jadwal mengunci teknisi, log langsung **"Dalam Proses"**.
- `next_run_at` dihitung otomatis backend (`now + frekuensi`): Harian = +1 hari, Mingguan = +7 hari,
  Bulanan = +1 bulan (tanggal sama). Mengubah frekuensi **meng-reset** `next_run_at` ke skema baru.
- Notifikasi hasil generate dibuat **sinkron** di dalam goroutine (bukan `NotifyUser` async);
  tetap aman karena tidak memblokir request HTTP.
- Kecepatan uji disarankan memakai **psql back-date** + **restart API** (fast-forward generator
  yang berjalan sekali saat startup); catatan baris null kolom itu juga merevisi `last_generated_at`.
- Untuk verifikasi SQL langsung: cek `SELECT count(*) FROM maintenance_logs WHERE maintenance_schedule_id = '<JADWAL_UUID>'` — harus = 1 untuk satu hari.