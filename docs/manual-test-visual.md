# Manual Test — Snapshot & Timeline (Fase 10)

> **Kontrak uji formal — MT-24**
>
> Dokumen ini mendefinisikan prosedur; hasil sebelum **2026-08-20** adalah catatan historis/non-authoritative dan harus diuji ulang dengan execution record formal. Inspeksi kode, review unit test, atau checkmark lama bukan `PASS` black-box.
>
> **Skenario audit reproduktif:** Dengan fixture JPEG/timeline dan simulator stream, jalankan suite timeline, rotasi 10 snapshot, monitor, fullscreen/pop-out, dan matriks role.
>
> **Status suite:** `NOT RUN`

### Formal execution metadata

| Field | Value |
|---|---|
| Test ID / Audit ID | MT-24 |
| Capability owner | QA Visual/Streaming + Frontend/Backend/Infra |
| Status suite | NOT RUN |
| Build SHA | NOT RECORDED |
| Artifact digest/ID | NOT RECORDED |
| Tester | NOT RECORDED |
| Timestamp WITA | NOT RECORDED |
| Environment | NOT RECORDED |
| Fixture prefix/data | `E2E-MT24-` / NOT RECORDED |
| Cleanup result | NOT RUN |
| Expected | Empat tipe event timeline muncul, rotasi 10 snapshot terbukti, role konsisten, dan monitor/fullscreen/pop-out diuji nyata. |
| Actual | NOT RECORDED |
| Evidence path/link | NOT RECORDED |
| Mandatory flag | YES |
| Scenario status | NOT RUN |

### Formal scenario record

| ID | Mandatory | Expected | Actual | Evidence | Status |
|---|---|---|---|---|---|
| MT-24-S1 | YES | Empat tipe event timeline muncul, rotasi 10 snapshot terbukti, role konsisten, dan monitor/fullscreen/pop-out diuji nyata. | NOT RECORDED | NOT RECORDED | NOT RUN |

> **Cakupan aggregate Wave 0:** Baris formal aggregate di atas adalah **sole atomic suite-level acceptance gate** untuk baseline Wave 0. Skenario legacy terperinci di bawah ini tetap non-formal dan historical/non-authoritative sampai diperluas selama Wave 7.

**Aturan kesimpulan suite:** sebelum eksekusi suite `NOT RUN`; setelah run dimulai, setiap mandatory scenario harus `PASS`. Satu mandatory scenario berstatus `FAIL`, `BLOCKED`, atau `NOT RUN` membuat kesimpulan suite `FAIL`.

> **Guard keselamatan wajib:** Untuk setiap SQL mutasi, `DELETE`/`TRUNCATE`, rotasi/regenerasi key atau QR, penghapusan, reset kredensial, atau mutasi credential, lanjut hanya bila environment bukan production dan target bukan `nexaspace.site`, fixture prefix `E2E-MT24-` serta exact targets sudah dicatat, backup/recovery terverifikasi untuk perubahan data/key state, dan cleanup/rollback didefinisikan serta dicatat setelah run. Jika salah satu syarat tidak tersedia, hentikan prosedur dan set scenario status ke `BLOCKED`; jangan jalankan pada production. Jangan menyalin credential, token, private key, RTSP password, atau payload production yang belum disanitasi.


Dokumen ini memverifikasi fitur **Snapshot Otomatis & Timeline Event Visual** secara end-to-end (Fase 10), mencakup: snapshot otomatis berkala (goroutine 15 menit via MediaMTX `GetSnapshot`), snapshot on-demand/endpoint `GET /api/cctv/:id/snapshot` (JPEG), rotasi maksimal 10 snapshot per CCTV, field computed `snapshot_url` pada respons CCTV, timeline kronologis terpaginasi `GET /api/cctv/:id/timeline`, event hook (`status_changed`, `maintenance_completed`, `field_check_attention`, `periodic_snapshot`), model `CCTVSnapshot` + migrasi `000005`, thumbnail snapshot di tabel Master CCTV & tooltip marker peta, timeline di detail CCTV, serta role enforcement. Branch yang dipakai: `feature/cctv-snapshot`. Backend versi `0.46.0` · frontend `1.38.0`.

> **Sprint 25 — Monitor Grid & Stream Health (Task 10.3 + 10.4)** — Fase 10 selesai (4/4 NOT RUN). Backend versi `0.47.0` · frontend `1.39.0`.

## 1. Prasyarat

| # | Prasyarat | Keterangan |
|---|-----------|------------|
| P1 | Aplikasi berjalan | `docker compose up -d --build` — seluruh service (db, redis, api, frontend, nginx, backup, mediamtx) healthy |
| P2 | MediaMTX running + stream CCTV aktif | Service `mediamtx` berjalan; minimal 1 CCTV dengan `stream_enabled=true` & `channel_port` terisi agar path RTSP tersedia untuk snapshot |
| P3 | Login admin | Login sebagai **admin** di `https://localhost` (Swagger memakai `Authorization: Bearer`) |
| P4 | CCTV dengan `stream_enabled=true` | Data CCTV di Master `/cctv` yang diaktifkan streaming-nya (agar scheduler dapat menangkap frame) |
| P5 | Akses Swagger | Swagger tersedia di `https://localhost/swagger/index.html` (atau `http://localhost:3000/swagger/index.html` tanpa Docker) |

## 2. Pengujian Otomatis

```
cd backend && go build ./... && go test ./... -count=1 && go vet ./...
cd backend && golangci-lint run ./...
cd backend && swag init
cd frontend && npm run lint && npm run build
```

| Item | Observasi historis (non-authoritative) |
|------|-------------|
| `go build ./...` | NOT RUN |
| `go test ./... -count=1` | NOT RUN (termasuk `snapshot_test.go`) |
| `go vet ./...` | NOT RUN |
| `npm run lint` | NOT RUN 0 error |
| `npm run build` | NOT RUN |

## 3. Skenario Uji

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|-----------------------|-------------|
| S1 | Snapshot otomatis — setelah scheduler berjalan 15 menit, periksa folder `uploads/snapshots/{cctv_id}/` pada backend, lalu panggil `GET /api/cctv/:id/snapshot` | Folder `uploads/snapshots/{cctv_id}/` berisi file JPEG; `GET /api/cctv/:id/snapshot` mengembalikan gambar **terbaru** (Content-Type `image/jpeg`) | NOT RUN — Tidak dapat diuji (NVR `<NVR_HOST_NON_PROD>` unreachable → scheduler 15 menit tidak berhasil menangkap frame; mekanisme terverifikasi dari kode `RunSnapshotScheduler` + `CaptureSnapshot`) |
| S2 | Snapshot on-demand/endpoint — panggil `GET /api/cctv/:id/snapshot` untuk CCTV yang **belum** memiliki snapshot (folder kosong / snapshot belum diambil) | **404 Not Found** (tidak ada snapshot tersimpan) | NOT RUN |
| S3 | Rotasi snapshot — biarkan beberapa siklus 15 menit berjalan atau buat banyak snapshot pada satu CCTV | Direktori `uploads/snapshots/{cctv_id}/` hanya menyimpan **10 snapshot terbaru** (snapshot lama dihapus) | NOT RUN (verifikasi kode + unit test rotasi) |
| S4 | Field `snapshot_url` — panggil `GET /api/cctv` dan `GET /api/cctv/:id` | Respons memuat field `snapshot_url` yang menunjuk ke endpoint `GET /api/cctv/:id/snapshot` (nilai null/omitempty bila belum ada snapshot) | NOT RUN (verifikasi kode — `models/cctv.go:44` `snapshot_url,omitempty`) |
| S5 | Timeline display — panggil `GET /api/cctv/:id/timeline?page=1&limit=20` | Respons mengembalikan daftar event **kronologis** (urutan waktu, terbaru dahulu) dengan paginasi (`page`, `limit`, `total_rows`, `total_pages`) | NOT RUN |
| S6 | Thumbnail di tabel Master CCTV — buka halaman `/cctv` | Kolom snapshot menampilkan **gambar kecil** (via `AuthenticatedImage` menggunakan `snapshot_url`); menampilkan **placeholder** (ikon `ImageOff`) bila belum ada snapshot | NOT RUN (verifikasi kode — `snapshot_url` di `cctv/page.tsx:64`) |
| S7 | Thumbnail di tooltip marker peta — buka `/sebaran-cctv`, arahkan kursor ke marker CCTV | Tooltip marker menampilkan **gambar snapshot kecil** (64×48); placeholder ikon `Camera` bila belum ada snapshot | NOT RUN (verifikasi kode — `snapshot_url` di `CCTVMap.tsx:23`) |
| S8 | Hook event — ubah status CCTV / field-check `attention` / maintenance selesai, lalu cek timeline | Event **baru muncul** di `GET /api/cctv/:id/timeline` dengan `event_type` sesuai: `status_changed`, `maintenance_completed`, `field_check_attention`, atau `periodic_snapshot`; tampil juga di timeline detail CCTV | NOT RUN Sebagian — 3 dari 4 event terimplementasi (`status_changed` di `cctv.go:663`, `field_check_attention` di `field_maintenance.go:260`, `periodic_snapshot` di `snapshot.go:269`); **`maintenance_completed` tidak terimplementasi** (tidak ada pemanggilan `CaptureSnapshot` saat maintenance selesai) |
| S9 | Role enforcement — akses `GET /api/cctv/:id/snapshot` dan `GET /api/cctv/:id/timeline` tanpa login / dengan role teknisi di luar izin | **401 Unauthorized** tanpa login; **403 Forbidden** bila role tidak diizinkan | NOT RUN (endpoint `Protected()`) |

## 4. Kriteria Penerimaan

- Snapshot tersimpan di `uploads/snapshots/{cctv_id}/` dan **rotasi maksimal 10** snapshot per CCTV.
- Endpoint `GET /api/cctv/:id/snapshot` & `GET /api/cctv/:id/timeline` berfungsi (404 bila tidak ada snapshot; paginasi benar).
- Field `snapshot_url` muncul pada respons `GET /api/cctv` / `GET /api/cctv/:id`.
- Thumbnail snapshot tampil di tabel Master CCTV & tooltip marker peta; timeline tampil di detail CCTV.
- Build/test/lint NOT RUN: `go build`, `go test ./... -count=1`, `go vet`, `golangci-lint run`, `swag init`; `npm run lint` & `npm run build`.

## Catatan & Gotchas

- Snapshot berupa **JPEG** (`Content-Type: image/jpeg`), diambil via `utils/mediamtx.go` `GetSnapshot` dari endpoint MediaMTX `/v3/paths/get/{path}/snapshot`.
- **MediaMTX harus reachable** agar scheduler snapshot dapat menangkap frame; bila path tidak `ready` / stream mati, snapshot di-skip.
- Snapshot hanya untuk stream ber-**H.264** (sesuai keputusan arsitektur streaming Fase 9 — zero transcoding).
- Snapshot disimpan di folder **`uploads/snapshots/{cctv_id}/`** (di-backup bersama folder `uploads` oleh `scripts/backup.sh`); gambar diakses via endpoint protected `AuthenticatedImage`.
- Event types & label UI: `status_changed` ("Status berubah"), `maintenance_completed` ("Pemeliharaan selesai"), `field_check_attention` ("Pengecekan butuh perhatian"), `periodic_snapshot` ("Snapshot berkala").
- Endpoint terkait: `GET /api/cctv/:id/snapshot`, `GET /api/cctv/:id/timeline?page=&limit=`.
- Endpoint backend terkait: `GET /api/cctv/:id/snapshot` (JPEG), `GET /api/cctv/:id/timeline` (paginasi), model `CCTVSnapshot` + migrasi `000005_cctv_snapshots`.

---

# Manual Test — Monitor Grid & Stream Health (Sprint 25, Fase 10)

Dokumen ini memverifikasi fitur **Multi-View Monitor & Deteksi Status Stream** (Fase 10) secara end-to-end, mencakup: halaman `/monitor` (admin-only) grid multi-stream dengan preset layout 2×2/3×3/4×4, per-cell CCTV dropdown + WHEPPlayer, fullscreen per sel & seluruh grid, pop-out `window.open('/monitor/pop?path=...')` & halaman `/monitor/pop`, menu sidebar "Monitor CCTV", serta deteksi status stream: goroutine monitoring (tiap 5 menit) query MediaMTX `ListPaths()` — track per-path ready state, stream down > 15 menit → broadcast `STREAM_DOWN` + kurangi health_score -10, pulih → `STREAM_RESTORED`; endpoint `GET /api/streams/health` (admin-only). Branch yang dipakai: `feature/monitor-view`. Backend versi `0.47.0` · frontend `1.39.0`.

## 1. Prasyarat

| # | Prasyarat | Keterangan |
|---|-----------|------------|
| P1 | Aplikasi berjalan | `docker compose up -d --build` — seluruh service (db, redis, api, frontend, nginx, backup, mediamtx) healthy |
| P2 | MediaMTX running + stream CCTV aktif | Service `mediamtx` berjalan; minimal beberapa CCTV dengan `stream_enabled=true` & `channel_port` terisi agar path RTSP tersedia untuk monitor grid |
| P3 | Login admin | Login sebagai **admin** di `https://localhost` (Swagger memakai `Authorization: Bearer`) |
| P4 | Akses Swagger | Swagger tersedia di `https://localhost/swagger/index.html` (atau `http://localhost:3000/swagger/index.html` tanpa Docker) |
| P5 | Akun teknisi (uji role) | Akun **teknisi** tersedia untuk memverifikasi pembatasan akses admin-only pada `/monitor` & `/api/streams/health` |

## 2. Pengujian Otomatis

```
cd backend && go build ./... && go test ./... -count=1 && go vet ./...
cd backend && golangci-lint run ./...
cd backend && swag init
cd frontend && npm run lint && npm run build
```

| Item | Observasi historis (non-authoritative) |
|------|-------------|
| `go build ./...` | NOT RUN |
| `go test ./... -count=1` | NOT RUN (termasuk `stream_health_test.go`) |
| `go vet ./...` | NOT RUN |
| `npm run lint` | NOT RUN 0 error |
| `npm run build` | NOT RUN |

## 3. Skenario Uji

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|-----------------------|-------------|
| S10 | Halaman `/monitor` admin-only — login sebagai **teknisi**, buka `/monitor` | Akses **ditolak/di-redirect** (guard proxy + middleware `RequireRole("admin")`); halaman tidak dirender untuk teknisi | NOT RUN |
| S11 | Grid multi-view — login sebagai admin, buka `/monitor`, pilih CCTV pada beberapa sel via dropdown, pilih preset layout 2×2/3×3/4×4 | Grid menampilkan preset yang dipilih; tiap sel menampilkan **video live** CCTV yang dipilih via `WHEPPlayer` | NOT RUN — Tidak dapat diuji (butuh browser + NVR reachable); struktur grid/`WHEPPlayer` terverifikasi dari kode |
| S12 | Fullscreen per sel & seluruh grid — klik tombol fullscreen pada satu sel, lalu tombol fullscreen seluruh grid | Sel yang dipilih menjadi **fullscreen** (Fullscreen API); seluruh grid juga bisa masuk **fullscreen** dan kembali normal (Esc/tombol) | NOT RUN — Tidak dapat diuji (butuh browser) |
| S13 | Pop-out — klik tombol pop pada sebuah sel | Window baru terbuka via `window.open('/monitor/pop?path=...')` menampilkan stream CCTV tersebut dalam **fullscreen** | NOT RUN — Tidak dapat diuji (butuh browser) |
| S14 | Stream health — panggil `GET /api/streams/health` sebagai admin | Respons menampilkan daftar stream CCTV dengan status **ready/down** beserta **down_duration** per stream | NOT RUN |
| S15 | Deteksi down — simulasi NVR/stream mati > 15 menit, lalu pulihkan | Saat stream down > 15 menit: notifikasi **`STREAM_DOWN`** ter-broadcast ke admin & **health_score turun 10**; saat stream pulih: notifikasi **`STREAM_RESTORED`** | NOT RUN — Tidak dapat diuji (butuh NVR reachable + down >15 menit); unit test `stream_health_test.go` memenuhi kriteria |
| S16 | Menu sidebar "Monitor CCTV" — buka dashboard sebagai admin, periksa sidebar | Menu sidebar **"Monitor CCTV"** (ikon `MonitorPlay`) tampil untuk admin, diposisikan **setelah "Peta Sebaran CCTV"** | NOT RUN (verifikasi kode — `DashboardLayout.tsx:120`) |

## 4. Kriteria Penerimaan

- Halaman `/monitor` hanya dapat diakses admin (teknisi di-redirect/403); menu sidebar "Monitor CCTV" tampil untuk admin setelah "Peta Sebaran CCTV".
- Grid multi-view mendukung preset layout 2×2/3×3/4×4 dengan video live per sel via `WHEPPlayer`.
- Fullscreen per sel & seluruh grid berfungsi; pop-out membuka `/monitor/pop?path=...` menampilkan stream fullscreen.
- `GET /api/streams/health` (admin-only) mengembalikan status ready/down + `down_duration`.
- Stream down > 15 menit → notifikasi `STREAM_DOWN` + health_score -10; pulih → `STREAM_RESTORED`.
- Build/test/lint NOT RUN: `go build`, `go test ./... -count=1`, `go vet`, `golangci-lint run`, `swag init`; `npm run lint` & `npm run build`.

## Catatan & Gotchas

- **Goroutine monitoring** berjalan tiap 5 menit memakai `utils/mediamtx.go` `ListPaths()` untuk men-track ready state per path.
- Ambang deteksi down adalah **> 15 menit** — setelan internal, bukan 5 menit siklus polling.
- Notifikasi `STREAM_DOWN`/`STREAM_RESTORED` di-broadcast (in-app/admin) dan **mengurangi `health_score` sebesar 10** per kejadian down.
- Endpoint `GET /api/streams/health` **admin-only** (`Protected()` + `RequireRole("admin")`).
- Pop-out memakai `window.open('/monitor/pop?path=...')` — pastikan popup blocker tidak memblokir window baru.
- Halaman `/monitor/pop` adalah single fullscreen stream yang membaca `path` dari query parameter.
- Unit test backend: `handlers/stream_health_test.go` (Task 10.4).
