# Uji Manual — Sinkronisasi Status CCTV Otomatis

> **Kontrak uji formal — MT-22**
>
> Dokumen ini mendefinisikan prosedur; hasil sebelum **2026-08-20** adalah catatan historis/non-authoritative dan harus diuji ulang dengan execution record formal. Inspeksi kode, review unit test, atau checkmark lama bukan `PASS` black-box.
>
> **Skenario audit reproduktif:** Dengan state-transition matrix status awal dan hasil checklist, jalankan jalur complete 9/9, satu masalah, nonaktif, UI/API, audit, dan WebSocket.
>
> **Status suite:** `NOT RUN`

### Formal execution metadata

| Field | Value |
|---|---|
| Test ID / Audit ID | MT-22 |
| Capability owner | QA Workflow + Backend |
| Status suite | NOT RUN |
| Build SHA | NOT RECORDED |
| Artifact digest/ID | NOT RECORDED |
| Tester | NOT RECORDED |
| Timestamp WITA | NOT RECORDED |
| Environment | NOT RECORDED |
| Fixture prefix/data | `E2E-MT22-` / NOT RECORDED |
| Cleanup result | NOT RUN |
| Expected | 9/9 OK menghasilkan `completed`, satu masalah menghasilkan tindak lanjut, `nonaktif` tidak ditimpa, dan UI/API/audit konsisten. |
| Actual | NOT RECORDED |
| Evidence path/link | NOT RECORDED |
| Mandatory flag | YES |
| Scenario status | NOT RUN |

### Formal scenario record

| ID | Mandatory | Expected | Actual | Evidence | Status |
|---|---|---|---|---|---|
| MT-22-S1 | YES | 9/9 OK menghasilkan `completed`, satu masalah menghasilkan tindak lanjut, `nonaktif` tidak ditimpa, dan UI/API/audit konsisten. | NOT RECORDED | NOT RECORDED | NOT RUN |

**Aturan kesimpulan suite:** sebelum eksekusi suite `NOT RUN`; setelah run dimulai, setiap mandatory scenario harus `PASS`. Satu mandatory scenario berstatus `FAIL`, `BLOCKED`, atau `NOT RUN` membuat kesimpulan suite `FAIL`.

> **Guard keselamatan wajib:** Untuk setiap SQL mutasi, `DELETE`/`TRUNCATE`, rotasi/regenerasi key atau QR, penghapusan, reset kredensial, atau mutasi credential, lanjut hanya bila environment bukan production dan target bukan `nexaspace.site`, fixture prefix `E2E-MT22-` serta exact targets sudah dicatat, backup/recovery terverifikasi untuk perubahan data/key state, dan cleanup/rollback didefinisikan serta dicatat setelah run. Jika salah satu syarat tidak tersedia, hentikan prosedur dan set scenario status ke `BLOCKED`; jangan jalankan pada production. Jangan menyalin credential, token, private key, RTSP password, atau payload production yang belum disanitasi.


> **Task:** 5.2 Sinkronisasi Status CCTV Otomatis (Konservatif)
> **Branch:** `feature/sync-cctv-status` · **Commit:** `5228266` · **Versi backend:** `0.28.0`
> **Dokumen ini:** alur uji manual klik-per-klik beserta hasil yang diharapkan (expected results).

Perilaku yang diuji: status CCTV disinkronkan otomatis saat tugas pemeliharaan atau
pengecekan lapangan selesai. Sifat **konservatif**: hanya transisi `aktif ↔ pemeliharaan`
yang dilakukan otomatis; status `nonaktif` **tidak pernah ditimpa** oleh sinkronisasi
(status `rusak` telah digabung ke `nonaktif` per 2026-08-05).

---

## 1. Prasyarat

| # | Item | Cara menyiapkan |
|---|------|-----------------|
| P1 | Aplikasi berjalan | `docker compose up -d --build` → buka `https://localhost` |
| P2 | Akun admin | Login dengan kredensial dari `.env` / `.env.docker` |
| P3 | Akun teknisi | **Manajemen Pengguna** (sidebar) → **"Tambah Pengguna"** → isi username, password, nama, email → Peran **"Teknisi"** → **"Tambah Pengguna"** |
| P4 | CCTV `UJI-01` (aktif) | **Aset CCTV** → **"Tambah CCTV"** → kode `UJI-01`, nama bebas, Status **"Aktif"** → **"Tambah CCTV"** |
| P5 | CCTV `UJI-02` (nonaktif) | **Aset CCTV** → tombol **"…"** → **"Impor Excel"** → unduh template → isi baris `UJI-02` dengan kolom **Status = `nonaktif`** → upload → impor. *(Alternatif: Swagger `PUT /api/cctv/{id}` dengan body `{"status":"nonaktif"}`.)* Catatan: status `rusak` lama otomatis dipetakan ke `nonaktif`. |
| P6 | QR CCTV | Cetak/unduh label QR tiap CCTV: **Aset CCTV** → tombol ikon **QR** pada baris → **"Unduh"**. *(Alternatif: input manual di modal "Pindai QR".)* |
| P7 | Dua tab browser | Tab A: **Aset CCTV** (`/cctv`) · Tab B: **Pemeliharaan CCTV** (`/maintenance`). Untuk S8 tambah Tab C: **Peta Sebaran CCTV** (`/sebaran-cctv`). |

---

## 2. Pengujian Otomatis (lapisan pertama)

Sebelum uji manual, jalankan test otomatis di `backend/`:

```powershell
go build ./...      # harus sukses
go test ./... -count=1   # semua test NOT RUN, termasuk:
                          # TestCompleteMaintenanceLog_SyncCCTV_*,
                          # TestCreateFieldMaintenance_SyncCCTV_*
go vet ./...
```

Hasil yang diharapkan: seluruh package hijau; tidak ada test yang gagal.

| Item | Observasi historis (non-authoritative) |
|------|-------------|
| `go build ./...` | NOT RUN |
| `go test ./... -count=1` | NOT RUN (termasuk `TestCompleteMaintenanceLog_SyncCCTV_*` & `TestCreateFieldMaintenance_SyncCCTV_*`) |
| `go vet ./...` | NOT RUN |

---

## 3. Skenario Uji Manual

Cara membaca: ikuti **Langkah** berurutan, lalu cocokkan dengan **Hasil yang diharapkan**.
Tabel ini hanya observasi historis; jangan gunakan sebagai status formal. Isi Formal scenario record di atas dengan expected, actual, evidence, dan status yang diizinkan.

### S1 — Complete tugas pemeliharaan: semua item OK → CCTV menjadi `aktif`

**Tujuan:** membuktikan transisi otomatis `pemeliharaan → aktif` setelah tugas selesai tanpa kegagalan.

| # | Langkah (Tab B = Pemeliharaan CCTV) | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|--------------------------------------|------------------------|-------------|
| 1 | Tab A (Aset CCTV): edit `UJI-01` (ikon pensil) → Status **"Pemeliharaan"** → **"Simpan Perubahan"** | Badge `UJI-01` = **Pemeliharaan** | NOT RUN |
| 2 | Tab B → **"Catat Pemeliharaan"** → pilih `UJI-01` → isi Tindakan → **"Simpan Catatan"** | Baris baru muncul, status **"Terjadwal"** | NOT RUN |
| 3 | Klik ikon **tambah-user** (title "Tugaskan ke teknisi") pada baris → **"Tugaskan ke Teknisi"** → pilih teknisi → **"Tugaskan"** | Status baris → **"Dalam proses"** | NOT RUN |
| 4 | Login sebagai teknisi (tab incognito) → Tab B → klik ikon **centang** pada baris "Dalam proses" | Wizard **"Selesaikan Tugas Pemeliharaan"** terbuka | NOT RUN — Tidak dapat diuji (butuh browser interaktif) |
| 5 | Semua **9 item** checklist: klik **"Normal / OK"** → **"Lanjut"** | Berpindah ke item berikutnya sampai tahap foto | NOT RUN — Tidak dapat diuji (butuh browser interaktif) |
| 6 | Tahap **"Bukti Foto"** → **"Tambah foto (0/5)"** → pilih 1 gambar → **"Selesaikan Tugas"** | Log tersimpan; toast sukses | NOT RUN — Tidak dapat diuji (butuh browser interaktif) |
| 7 | Cek Tab B | Status log = **"Selesai"** | NOT RUN |
| 8 | Cek Tab A (Aset CCTV) | Badge `UJI-01` berubah otomatis **Pemeliharaan → Aktif** (tanpa refresh manual) | NOT RUN (verifikasi kode + audit nyata: "Status CCTV diubah otomatis dari 'pemeliharaan' ke 'aktif' karena semua checklist pemeliharaan berhasil") |

### S2 — Complete tugas pemeliharaan: ada item "Bermasalah" → CCTV menjadi `pemeliharaan`

**Tujuan:** membuktikan transisi otomatis `aktif → pemeliharaan` ketika tugas dianggap gagal.

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Tab A: edit `UJI-01` → Status **"Aktif"** → **"Simpan Perubahan"** | Badge `UJI-01` = **Aktif** |NOT RUN (catatan: ketika status badge pemeliharan ingin diubah ke aktif melalui halaman cctv assets gagal, karna ada yang dibuat gagal dari 9 task, tetapi jika diubah ke nonaktif bisa, hanya ke aktif yang tidak bisa, toast menampilkan informasi data berhasil dirubah, tetapi hasilnya masih pemeliharan)|
| 2 | Tab B: buat tugas `UJI-01` ("Catat Pemeliharaan") → tugaskan ke teknisi | Status **"Dalam proses"** | NOT RUN|
| 3 | Teknisi: buka baris → ikon **centang** → pada item ke-5 ("Pengecekan kualitas gambar") klik **"Bermasalah"** → isi Catatan (wajib) → **"Lanjut"**; item lain **"Normal / OK"** | Tombol Lanjut aktif setelah memilih hasil + catatan | NOT RUN |
| 4 | Tahap foto → 1 gambar → **"Selesaikan Tugas"** | Log tersimpan | NOT RUN |
| 5 | Cek Tab B | Status log = **"Perlu Tindak Lanjut"** | NOT RUN |
| 6 | Cek Tab A | Badge `UJI-01` berubah otomatis **Aktif → Pemeliharaan** | NOT RUN (verifikasi kode + audit nyata: "Status CCTV diubah otomatis dari 'aktif' ke 'pemeliharaan' karena tugas pemeliharaan gagal — ada item checklist tidak OK") |

### S3 — Pengecekan lapangan (scan QR): ada item "Bermasalah" → CCTV menjadi `pemeliharaan`

**Tujuan:** membuktikan sinkronisasi juga terjadi lewat alur pengecekan lapangan, dan admin menerima notifikasi.

| # | Langkah (Tab A = Aset CCTV) | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|------------------------------|------------------------|-------------|
| 1 | Tab A: edit `UJI-01` → Status **"Aktif"** → **"Simpan Perubahan"** | Badge = **Aktif** | NOT RUN|
| 2 | Tab A → **"Pindai QR"** → pindai QR `UJI-01` (atau input manual) | CCTV teridentifikasi; muncul tombol **"Maintenance"** |NOT RUN |
| 3 | Klik **"Maintenance"** → wizard **"Pengecekan Lapangan"** terbuka → item ke-2 ("Pengecekan pasokan listrik") klik **"Bermasalah"** + catatan → sisanya **"Normal / OK"** → foto → **"Selesaikan Pengecekan"** | Pengecekan tersimpan; toast sukses |NOT RUN |
| 4 | Cek Tab A | Badge `UJI-01` berubah otomatis **Aktif → Pemeliharaan** | NOT RUN (verifikasi kode + audit nyata: "Status CCTV diubah otomatis dari 'aktif' ke 'pemeliharaan' karena hasil pengecekan lapangan memerlukan perhatian") |
| 5 | Cek halaman **Pengecekan Lapangan** (`/field-maintenance`, admin) | Baris pengecekan baru muncul | NOT RUN |
| 6 | Cek lonceng **Pemberitahuan** (sidebar, admin) | Ada notifikasi hasil pengecekan "butuh perhatian" | NOT RUN (nyata — notifikasi `FIELD_CHECK_ATTENTION` diterima admin) |

### S4 — Pengecekan lapangan (scan QR): semua item OK → CCTV menjadi `aktif`

**Tujuan:** membuktikan transisi `pemeliharaan → aktif` lewat alur pengecekan lapangan.

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Tab A: pastikan `UJI-01` berstatus **Pemeliharaan** (jika belum, ubah via edit) | Badge = **Pemeliharaan** | NOT RUN |
| 2 | Tab A → **"Pindai QR"** → **"Maintenance"** → semua **9 item** "Normal / OK" → foto → **"Selesaikan Pengecekan"** | Pengecekan tersimpan |NOT RUN |
| 3 | Cek Tab A | Badge `UJI-01` berubah otomatis **Pemeliharaan → Aktif** | NOT RUN (verifikasi kode + audit nyata: "Status CCTV diubah otomatis dari 'pemeliharaan' ke 'aktif' karena pengecekan lapangan menunjukkan semua item OK") |

### S5 — Uji konservatif: CCTV `nonaktif` (termasuk legacy `rusak`) TIDAK berubah walau hasil 9/9 OK

**Tujuan:** membuktikan status `nonaktif` tidak pernah ditimpa oleh sinkronisasi otomatis (fitur kunci 5.2).

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Tab A: pastikan `UJI-02` (dari P5) tetap berbadge **Nonaktif**; biarkan tab terbuka | Badge = Nonaktif | NOT RUN |
| 2 | Tab B: buat tugas `UJI-02` → tugaskan → teknisi complete dengan **SEMUA** "Normal / OK" → foto → **"Selesaikan Tugas"** | Log = **"Selesai"** | NOT RUN
| 3 | Tab B: buat tugas kedua untuk `UJI-02` → tugaskan → complete dengan item "Bermasalah" → foto → **"Selesaikan Tugas"** | Log = **"Perlu Tindak Lanjut"** | NOT RUN
| 4 | Cek Tab A setelah kedua penyelesaian | Badge `UJI-02` **tetap Nonaktif** — tidak berubah sama sekali | NOT RUN (verifikasi kode — guard `current == "nonaktif" → return nil` di `cctv_sync.go:38-40`; tidak ada audit `UPDATE_CCTV` ke nonaktif) |

### S6 — Uji konservatif: transisi ulang `nonaktif → aktif` tetap dijaga

**Tujuan:** membuktikan bahwa walau `nonaktif` sudah sempat dinonaktifkan, sinkronisasi otomatis pada hasil manapun tetap tidak mengubahnya.

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Tab A: (opsional) edit `UJI-02` ke **"Aktif"** lalu kembali ke **"Nonaktif"** → **"Simpan Perubahan"** | Badge `UJI-02` = **Nonaktif** | NOT RUN |
| 2 | Ulangi S5 langkah 2–3 (complete semua OK + complete dengan item bermasalah) | Log = "Selesai" lalu "Perlu Tindak Lanjut" | NOT RUN
| 3 | Cek Tab A | Badge `UJI-02` **tetap Nonaktif** | NOT RUN (verifikasi kode — konservatif) |

### S7 — Verifikasi Audit Log & data tersimpan

**Tujuan:** membuktikan setiap sinkronisasi tercatat di audit log modul CCTV.

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | **Audit Log** (sidebar, admin) → **"Sortir"** → filter **"Modul" → "CCTV"** → **"Terapkan"** | Daftar berisi baris aksi `UPDATE_CCTV` | NOT RUN |
| 2 | Cari baris dari skenario S1–S4 | Deskripsi memuat: "Status CCTV diubah otomatis dari 'pemeliharaan' ke 'aktif' karena semua checklist pemeliharaan berhasil" dan "...dari 'aktif' ke 'pemeliharaan' karena tugas pemeliharaan gagal — ada item checklist tidak OK" | NOT RUN |
| 3 | Cari baris dari S5–S6 | **Tidak ada** baris `UPDATE_CCTV` yang mengubah status `nonaktif` (tidak ada sinkronisasi yang terjadi) | NOT RUN |

### S8 — Verifikasi real-time (WebSocket)

**Tujuan:** membuktikan perubahan status tersebar real-time tanpa refresh.

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Buka Tab A (**Aset CCTV**) dan Tab C (**Peta Sebaran CCTV**); jangan refresh | Daftar dan peta tampil normal | NOT RUN |
| 2 | Di Tab B lakukan S1 langkah 2–6 (complete semua OK untuk CCTV yang sedang `pemeliharaan`) | Dalam beberapa detik badge `UJI-01` di Tab A dan pin di Tab C berubah otomatis | NOT RUN (verifikasi kode — broadcast `CCTV_UPDATED` di `cctv_sync.go:57`) |
| 3 | Ulangi untuk S3 (pengecekan lapangan bermasalah) | Perubahan kembali tersebar real-time | NOT RUN (verifikasi kode) |

---

## 4. Kriteria Penerimaan Keseluruhan

Semua baris "Hasil yang diharapkan" pada S1–S8 terpenuhi, khususnya:

1. S1–S4: status berpindah `aktif ↔ pemeliharaan` secara otomatis sesuai arah yang benar.
2. S5–S6: status `nonaktif` (termasuk legacy `rusak`) **tidak pernah berubah** walau checklist 9/9 OK.
3. S7: audit log `UPDATE_CCTV` hanya muncul untuk transisi yang benar-benar terjadi.
4. S8: perubahan terlihat real-time di halaman lain tanpa refresh.
5. `go build ./...`, `go test ./... -count=1`, dan `go vet ./...` hijau.

## 5. Catatan

- Status di database disimpan huruf kecil: `aktif`, `pemeliharaan`, `nonaktif` (status `rusak` digabung ke `nonaktif` per 2026-08-05).
- Dropdown status pada form CCTV menyediakan **Aktif / Nonaktif / Pemeliharaan**; nilai `rusak`/`broken` dari impor atau API lama dipetakan otomatis ke `nonaktif`.
- Jika status CCTV sudah sama dengan target (mis. sudah `aktif`, complete semua OK), sinkronisasi **sengaja dilewati** — tidak ada perubahan maupun audit log. Ini perilaku yang benar.
- Tombol **"Selesaikan tugas lewat checklist 9 tahap"** hanya muncul untuk log berstatus **"Dalam proses"** dan hanya untuk admin atau teknisi yang ditugaskan.

## Observasi Historis (Non-authoritative) — Catatan Verifikator

**Kesimpulan historis (non-authoritative): S1–S8 NOT RUN** (mekanisme terbukti dari audit log produksi + kode + unit test; langkah wizard browser interaktif ditandai NOT RUN karena tidak dijalankan di lingkungan verifikasi).

- Bukti nyata di produksi: 8 baris audit `UPDATE_CCTV` otomatis — 6 dari complete maintenance (3× `'aktif'→'pemeliharaan'` "tugas pemeliharaan gagal — ada item checklist tidak OK", 3× `'pemeliharaan'→'aktif'` "semua checklist pemeliharaan berhasil") + 2 dari pengecekan lapangan ("hasil pengecekan lapangan memerlukan perhatian" & "pengecekan lapangan menunjukkan semua item OK"). Deskripsi persis sesuai dokumen.
- Konservatif: **tidak ada** audit `UPDATE_CCTV` yang mengubah status ke `nonaktif` — guard `current == "nonaktif" → return nil` (`handlers/cctv_sync.go:38-40`).
- Catatan lama S2-1 (ubah ke Aktif gagal via UI karena ada item gagal dari 9 task) tetap dipertahankan sebagai temuan pengujian UI sebelumnya; perilaku backend sinkronisasi sudah sesuai.
