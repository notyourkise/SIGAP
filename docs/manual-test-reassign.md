# Uji Manual — Re-assign Tugas oleh Admin

> **Kontrak uji formal — MT-16**
>
> Dokumen ini mendefinisikan prosedur; hasil sebelum **2026-08-20** adalah catatan historis/non-authoritative dan harus diuji ulang dengan execution record formal. Inspeksi kode, review unit test, atau checkmark lama bukan `PASS` black-box.
>
> **Skenario audit reproduktif:** Dengan dua teknisi fixture dan satu tugas ongoing, reassign melalui UI/API lalu verifikasi kandidat, atomicity, audit, akses teknisi lama, dan notifikasi.
>
> **Status suite:** `NOT RUN`

### Formal execution metadata

| Field | Value |
|---|---|
| Test ID / Audit ID | MT-16 |
| Capability owner | QA Workflow + Frontend/Backend |
| Status suite | NOT RUN |
| Build SHA | NOT RECORDED |
| Artifact digest/ID | NOT RECORDED |
| Tester | NOT RECORDED |
| Timestamp WITA | NOT RECORDED |
| Environment | NOT RECORDED |
| Fixture prefix/data | `E2E-MT16-` / NOT RECORDED |
| Cleanup result | NOT RUN |
| Expected | Teknisi aktif tidak menjadi kandidat, reassign atomik, teknisi lama tidak dapat menyelesaikan tugas, dan notifikasi tidak ganda. |
| Actual | NOT RECORDED |
| Evidence path/link | NOT RECORDED |
| Mandatory flag | YES |
| Scenario status | NOT RUN |

### Formal scenario record

| ID | Mandatory | Expected | Actual | Evidence | Status |
|---|---|---|---|---|---|
| MT-16-S1 | YES | Teknisi aktif tidak menjadi kandidat, reassign atomik, teknisi lama tidak dapat menyelesaikan tugas, dan notifikasi tidak ganda. | NOT RECORDED | NOT RECORDED | NOT RUN |

**Aturan kesimpulan suite:** sebelum eksekusi suite `NOT RUN`; setelah run dimulai, setiap mandatory scenario harus `PASS`. Satu mandatory scenario berstatus `FAIL`, `BLOCKED`, atau `NOT RUN` membuat kesimpulan suite `FAIL`.

> **Guard keselamatan wajib:** Untuk setiap SQL mutasi, `DELETE`/`TRUNCATE`, rotasi/regenerasi key atau QR, penghapusan, reset kredensial, atau mutasi credential, lanjut hanya bila environment bukan production dan target bukan `nexaspace.site`, fixture prefix `E2E-MT16-` serta exact targets sudah dicatat, backup/recovery terverifikasi untuk perubahan data/key state, dan cleanup/rollback didefinisikan serta dicatat setelah run. Jika salah satu syarat tidak tersedia, hentikan prosedur dan set scenario status ke `BLOCKED`; jangan jalankan pada production. Jangan menyalin credential, token, private key, RTSP password, atau payload production yang belum disanitasi.


> **Task:** 5.3 Re-assign Tugas oleh Admin
> **Branch:** `feature/reassign-maintenance` · **Merge PR:** `#61` (commit `9c058f7`)
> **Versi:** backend `0.29.0` · frontend `1.26.0`
> **Dokumen ini:** alur uji manual klik-per-klik beserta hasil yang diharapkan (expected results).

Perilaku yang diuji: admin dapat **mengalihkan** tugas pemeliharaan yang sedang
dikerjakan (`ongoing`) ke teknisi lain. Saat dialihkan:
- teknisi lama dilepas dan mendapat notifikasi **"Tugas dialihkan"** (`MAINTENANCE_UNASSIGNED`),
- teknisi baru mendapat notifikasi **"Tugas baru ditugaskan"** (`MAINTENANCE_ASSIGNED`),
- status tugas **tetap "Dalam proses"** (tidak kembali ke "Terjadwal"),
- assign ke teknisi yang **sama** ditolak dengan pesan "Teknisi sudah ditugaskan pada tugas ini",
- semua aktivitas tercatat di audit log dengan aksi `REASSIGN_MAINTENANCE`.

---

## 1. Prasyarat

| # | Item | Cara menyiapkan |
|---|------|-----------------|
| P1 | Aplikasi berjalan | `docker compose up -d --build` → buka `https://localhost` (pastikan image backend/frontend ter-rebuild dengan kode terbaru hasil merge) |
| P2 | Akun admin | Login dengan kredensial dari `.env` / `.env.docker` |
| P3 | **Dua** akun teknisi | **Manajemen Pengguna** (sidebar) → **"Tambah Pengguna"** → buat `teknisi_a` dan `teknisi_b` (username, password, nama, email → Peran **"Teknisi"**). Catat nama lengkapnya untuk memudahkan identifikasi |
| P4 | CCTV `UJI-01` (aktif) | **Aset CCTV** → **"Tambah CCTV"** → kode `UJI-01`, nama bebas, Status **"Aktif"** → **"Tambah CCTV"** |
| P5 | Tiga sesi browser | Tab A: **admin** di **Pemeliharaan CCTV** (`/maintenance`) · Tab B: **incognito, login sebagai `teknisi_a`** → halaman Pemeliharaan · Tab C: **incognito, login sebagai `teknisi_b`** → halaman Pemeliharaan. *(Tiga profil browser berbeda juga bisa.)* |

---

## 2. Pengujian Otomatis (lapisan pertama)

Sebelum uji manual, jalankan test otomatis di `backend/`:

```powershell
go build ./...                # harus sukses
go test ./... -count=1        # semua test NOT RUN, termasuk:
                              #   TestAssignMaintenanceLog_ReassignSuccess
                              #   TestAssignMaintenanceLog_ReassignSameTechnician
                              #   TestAssignMaintenanceLog_CompletedRejected
                              #   TestAssignMaintenanceLog_ReassignRoleForbidden
go vet ./...
```

Lalu di `frontend/`:

```powershell
npm run lint   # 0 error (warning lama boleh)
npm run build  # harus sukses
```

Hasil yang diharapkan: seluruh package/test hijau dan build frontend sukses.

| Item | Observasi historis (non-authoritative) |
|------|-------------|
| `go build ./...` | NOT RUN |
| `go test ./... -count=1` | NOT RUN (termasuk `TestAssignMaintenanceLog_*`) |
| `go vet ./...` | NOT RUN |
| `npm run lint` | NOT RUN 0 error |
| `npm run build` | NOT RUN |

---

## 3. Skenario Uji Manual

Cara membaca: ikuti **Langkah** berurutan, lalu cocokkan dengan **Hasil yang diharapkan**.
Tabel ini hanya observasi historis; jangan gunakan sebagai status formal. Isi Formal scenario record di atas dengan expected, actual, evidence, dan status yang diizinkan.

### S1 — Persiapan: buat tugas terjadwal lalu tugaskan ke Teknisi A (regresi alur lama)

**Tujuan:** memastikan tombol "Tugaskan" untuk log `scheduled` masih berfungsi seperti sebelum fitur re-assign.

| # | Langkah (Tab A = Pemeliharaan CCTV, admin) | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|--------------------------------------------|------------------------|-------------|
| 1 | **"Catat Pemeliharaan"** → pilih `UJI-01` → isi Tindakan (mis. "Pembersihan lensa berkala") → **"Simpan Catatan"** | Baris baru muncul dengan status **"Terjadwal"**, teknisi "Belum ditugaskan" | NOT RUN |
| 2 | Klik ikon **tambah-user** (title "Tugaskan ke teknisi") pada baris → modal **"Tugaskan ke Teknisi"** → pilih `teknisi_a` → **"Tugaskan"** | Toast sukses "Tugas berhasil ditugaskan ke teknisi."; status baris → **"Dalam proses"**; kolom Teknisi = nama `teknisi_a` | NOT RUN |
| 3 | Cek Tab B (login `teknisi_a`) | Baris tugas tampil; ikon **centang** (selesaikan tugas) muncul | NOT RUN |
| 4 | Cek lonceng **Pemberitahuan** `teknisi_a` | Ada notifikasi **"Tugas baru ditugaskan"** dengan isi "Anda mendapat tugas pemeliharaan baru untuk CCTV ..." | NOT RUN |

### S2 — Re-assign: alihkan tugas `ongoing` dari Teknisi A ke Teknisi B

**Tujuan:** membuktikan fitur inti — admin mengalihkan tugas yang sedang dikerjakan.

| # | Langkah (Tab A = admin) | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|--------------------------|------------------------|-------------|
| 1 | Cari baris tugas dari S1 (status **"Dalam proses"**) | Ikon **panah kiri-kanan** (title "Alihkan ke teknisi lain") muncul di kolom Tindakan — berbeda dengan ikon tambah-user untuk log terjadwal | NOT RUN |
| 2 | Klik ikon **panah kiri-kanan** | Modal terbuka dengan judul **"Alihkan ke Teknisi Lain"** | NOT RUN |
| 3 | Perhatikan kotak peringatan di modal | Peringatan kuning bertuliskan: "Tugas ini sedang dikerjakan oleh **[nama `teknisi_a`]**. Yakin ingin mengalihkan ke teknisi lain?" | NOT RUN |
| 4 | Pilih `teknisi_b` di dropdown → **"Alihkan"** | Toast sukses "Tugas berhasil dialihkan ke teknisi lain."; modal tertutup; kolom Teknisi = nama `teknisi_b`; status tetap **"Dalam proses"** (bukan kembali ke "Terjadwal") | NOT RUN |
| 5 | Cek Tab C (login `teknisi_b`) | Baris tugas tampil dengan ikon **centang** (selesaikan tugas) | NOT RUN |

### S3 — Notifikasi dua pihak (WebSocket real-time + daftar notifikasi)

**Tujuan:** membuktikan teknisi lama mendapat notifikasi "dialihkan" dan teknisi baru mendapat notifikasi "tugas baru", tanpa refresh.

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Sebelum langkah S2-4, biarkan Tab B (`teknisi_a`) dan Tab C (`teknisi_b`) terbuka di halaman mana pun (mis. Dashboard) | Lonceng pemberitahuan tampil normal | NOT RUN |
| 2 | Lakukan S2 langkah 4 (alihkan ke `teknisi_b`) | **Tanpa refresh**: Tab C (`teknisi_b`) memunculkan toast real-time notifikasi baru; badge lonceng bertambah | NOT RUN |
| 3 | Klik lonceng `teknisi_b` → daftar notifikasi | Item **"Tugas baru ditugaskan"** — isi: "Anda mendapat tugas pemeliharaan baru untuk CCTV **[nama CCTV]**." | NOT RUN |
| 4 | Klik item notifikasi `teknisi_b` | Notifikasi terbaca (badge berkurang) dan navigasi ke halaman Pemeliharaan | NOT RUN |
| 5 | Klik lonceng `teknisi_a` | Item **"Tugas dialihkan"** — isi: "Tugas pemeliharaan untuk CCTV **[nama CCTV]** telah dialihkan ke teknisi lain." | NOT RUN |
| 6 | Buka **Pemberitahuan** (`/pemberitahuan`) sebagai `teknisi_a` | Daftar riwayat memuat notifikasi "Tugas dialihkan" (tipe `MAINTENANCE_UNASSIGNED`) | NOT RUN |

### S4 — Tolak assign ke teknisi yang SAMA

**Tujuan:** membuktikan validasi backend — assign ke teknisi yang sedang ditugaskan ditolak 400.

| # | Langkah (Tab A = admin) | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|--------------------------|------------------------|-------------|
| 1 | Baris tugas (status "Dalam proses", teknisi `teknisi_b`) → klik **panah kiri-kanan** | Modal "Alihkan ke Teknisi Lain" terbuka dengan nama `teknisi_b` di peringatan | NOT RUN |
| 2 | Pilih `teknisi_b` (yang sama) di dropdown → **"Alihkan"** | Toast error: **"Teknisi sudah ditugaskan pada tugas ini"**; modal tetap terbuka; tidak ada perubahan data | NOT RUN |
| 3 | (Opsional) Cek Swagger: `PUT /api/maintenance-logs/{id}/assign` body `{"technician_id":"<id teknisi_b>"}` dengan token admin | Respons `400` + `message: "Teknisi sudah ditugaskan pada tugas ini"` | NOT RUN |

### S5 — Log `completed` / `failed` tidak bisa dialihkan

**Tujuan:** membuktikan hanya log `scheduled` dan `ongoing` yang menerima penugasan.

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Selesaikan tugas S2 oleh `teknisi_b` (Tab C: ikon **centang** → 9 item **"Normal / OK"** → foto → **"Selesaikan Tugas"**) | Log berstatus **"Selesai"** | NOT RUN |
| 2 | Cek Tab A (admin) pada baris "Selesai" tersebut | Ikon **panah kiri-kanan / tambah-user** **tidak muncul** untuk baris `completed` (dan `failed`) | NOT RUN |
| 3 | (Opsional) Swagger: `PUT /api/maintenance-logs/{id}/assign` pada log `completed` | Respons `400` + `message: "Hanya log berstatus terjadwal atau dalam proses yang dapat ditugaskan ke teknisi"` | NOT RUN |

### S6 — Teknisi tidak bisa mengalihkan (hanya admin)

**Tujuan:** membuktikan RBAC — tombol Alihkan hanya untuk admin, dan API menolak role teknisi (403).

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Cek Tab B / Tab C (login teknisi) pada baris tugas apa pun | Tidak ada ikon **panah kiri-kanan** maupun **tambah-user** di kolom Tindakan | NOT RUN |
| 2 | (Opsional) Swagger dengan token `teknisi_b`: `PUT /api/maintenance-logs/{id}/assign` | Respons `403` "Akses ditolak: Hanya Admin yang diperbolehkan" | NOT RUN |

### S7 — Audit log `REASSIGN_MAINTENANCE`

**Tujuan:** membuktikan setiap pengalihan tercatat dengan rincian teknisi lama → baru.

| # | Langkah (Tab A = admin) | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|--------------------------|------------------------|-------------|
| 1 | **Audit Log** (sidebar) → **"Sortir"** → filter **Modul = MAINTENANCE** → **"Terapkan"** | Daftar memuat baris dengan aksi **`REASSIGN_MAINTENANCE`** | NOT RUN |
| 2 | Buka baris `REASSIGN_MAINTENANCE` dari pengalihan S2 | Deskripsi berbentuk: "Tugas **[UUID tugas]** dialihkan dari **[nama `teknisi_a`]** ke **[nama `teknisi_b`]**" | NOT RUN |
| 3 | Pastikan pengalihan S1 (assign pertama) memakai aksi `ASSIGN_MAINTENANCE` | Dua baris berbeda: `ASSIGN_MAINTENANCE` (S1) dan `REASSIGN_MAINTENANCE` (S2) | NOT RUN |

### S8 — Regresi: alur tugas tetap berjalan normal setelah pengalihan

**Tujuan:** memastikan re-assign tidak merusak alur kerja (teknisi baru bisa lanjut, teknisi lama kehilangan akses).

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Buat tugas baru (`scheduled`) → klik **tambah-user** → tugaskan ke `teknisi_a` | Modal berjudul **"Tugaskan ke Teknisi"** (bukan "Alihkan"); tidak ada peringatan kuning; tombol submit **"Tugaskan"** — memastikan alur assign lama utuh | NOT RUN |
| 2 | Login `teknisi_a` → buka baris tugas yang telah dialihkan (S2) | Ikon **centang** (selesaikan) **tidak muncul** — teknisi lama sudah kehilangan akses | NOT RUN |
| 3 | Login `teknisi_b` → ikon **centang** pada tugas yang dialihkan → selesaikan dengan 9 item OK + foto | Tugas selesai; status **"Selesai"**; toast sukses | NOT RUN |

---

## 4. Kriteria Penerimaan Keseluruhan

Semua baris "Hasil yang diharapkan" pada S1–S8 terpenuhi, khususnya:

1. S2: tugas `ongoing` berhasil dialihkan — teknisi baru menggantikan teknisi lama, status tetap **"Dalam proses"**.
2. S3: `teknisi_a` menerima notifikasi **"Tugas dialihkan"** dan `teknisi_b` menerima **"Tugas baru ditugaskan"** secara real-time tanpa refresh.
3. S4: assign ke teknisi yang sama ditolak dengan pesan "Teknisi sudah ditugaskan pada tugas ini".
4. S5–S6: log `completed` tidak bisa dialihkan (400) dan role teknisi ditolak (403).
5. S7: audit log mencatat `REASSIGN_MAINTENANCE` dengan format "Tugas [ID] dialihkan dari [lama] ke [baru]".
6. S8: alur assign lama, penyelesaian tugas, dan RBAC tetap normal (tidak ada regresi).
7. `go build ./...`, `go test ./... -count=1`, `go vet ./...`, `npm run lint`, dan `npm run build` hijau.

## 5. Catatan

- Pesan notifikasi memakai **nama CCTV** dari log (contoh: "Anda mendapat tugas pemeliharaan baru untuk CCTV **Kamera Gerbang**."). Bila data CCTV dihapus, nama diganti fallback ke tindakan tugas.
- Tombol aksi pada baris: **tambah-user** (`UserPlus`) = tugaskan log terjadwal; **panah kiri-kanan** (`ArrowLeftRight`) = alihkan log yang sedang dikerjakan; keduanya memanggil endpoint yang sama `PUT /api/maintenance-logs/:id/assign`.
- Notifikasi re-assign dikirim secara asinkron (pola `NotifyUser`); jika server dimatikan segera setelah aksi, notifikasi bisa gagal tersimpan — ulangi pengalihan bila notifikasi tidak muncul.
- Uji memakai dua akun teknisi nyata (bukan dua tab dengan akun sama) karena notifikasi & scope data bersifat per-user.
