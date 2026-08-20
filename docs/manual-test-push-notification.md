# Uji Manual - Notifikasi Push Browser

> **Kontrak uji formal — MT-14**
>
> Dokumen ini mendefinisikan prosedur; hasil sebelum **2026-08-20** adalah catatan historis/non-authoritative dan harus diuji ulang dengan execution record formal. Inspeksi kode, review unit test, atau checkmark lama bukan `PASS` black-box.
>
> **Skenario audit reproduktif:** Dengan pasangan VAPID valid pada browser nyata dan fixture user, uji subscribe, push, click-through, scope, unsubscribe, serta fallback in-app.
>
> **Status suite:** `NOT RUN`

### Formal execution metadata

| Field | Value |
|---|---|
| Test ID / Audit ID | MT-14 |
| Capability owner | QA Web + Frontend/Backend/Infra |
| Status suite | NOT RUN |
| Build SHA | NOT RECORDED |
| Artifact digest/ID | NOT RECORDED |
| Tester | NOT RECORDED |
| Timestamp WITA | NOT RECORDED |
| Environment | NOT RECORDED |
| Fixture prefix/data | `E2E-MT14-` / NOT RECORDED |
| Cleanup result | NOT RUN |
| Expected | Subscribe idempoten, push diterima, klik mengarah benar, scope user benar, unsubscribe menghentikan push, dan in-app tetap berfungsi. |
| Actual | NOT RECORDED |
| Evidence path/link | NOT RECORDED |
| Mandatory flag | YES |
| Scenario status | NOT RUN |

### Formal scenario record

| ID | Mandatory | Expected | Actual | Evidence | Status |
|---|---|---|---|---|---|
| MT-14-S1 | YES | Subscribe idempoten, push diterima, klik mengarah benar, scope user benar, unsubscribe menghentikan push, dan in-app tetap berfungsi. | NOT RECORDED | NOT RECORDED | NOT RUN |

**Aturan kesimpulan suite:** sebelum eksekusi suite `NOT RUN`; setelah run dimulai, setiap mandatory scenario harus `PASS`. Satu mandatory scenario berstatus `FAIL`, `BLOCKED`, atau `NOT RUN` membuat kesimpulan suite `FAIL`.

> **Guard keselamatan wajib:** Untuk setiap SQL mutasi, `DELETE`/`TRUNCATE`, rotasi/regenerasi key atau QR, penghapusan, reset kredensial, atau mutasi credential, lanjut hanya bila environment bukan production dan target bukan `nexaspace.site`, fixture prefix `E2E-MT14-` serta exact targets sudah dicatat, backup/recovery terverifikasi untuk perubahan data/key state, dan cleanup/rollback didefinisikan serta dicatat setelah run. Jika salah satu syarat tidak tersedia, hentikan prosedur dan set scenario status ke `BLOCKED`; jangan jalankan pada production. Jangan menyalin credential, token, private key, RTSP password, atau payload production yang belum disanitasi.


> **Task:** 8.3 Notifikasi Push Browser
> **Branch:** `feature/push-notification`
> **Versi:** backend `0.41.0` - frontend `1.34.0`
> **Dokumen:** verifikasi opt-in push, subscribe/unsubscribe, pengiriman push (tugas baru, eskalasi overdue, hasil pengecekan attention), dan fallback ke notifikasi in-app.

Notifikasi push memakai **Web Push Protocol (VAPID)**. Backend mengirim push setiap kali `NotifyUser` dipanggil (tugas ditugaskan, eskalasi overdue, hasil pengecekan attention) ke semua subscription milik user. Jika VAPID tidak dikonfigurasi, push dilewati dan notifikasi in-app tetap berjalan (fallback).

## 1. Prasyarat

| # | Item | Cara menyiapkan |
|---|------|-----------------|
| P1 | Aplikasi berjalan | `docker compose up -d --build`, buka `https://localhost` |
| P2 | VAPID keys | Set `VAPID_PUBLIC_KEY`, `VAPID_PRIVATE_KEY`, `VAPID_EMAIL` di `backend/.env.docker`; juga `NEXT_PUBLIC_VAPID_PUBLIC_KEY` di frontend build env. Generate VAPID keys sesuai dokumentasi webpush-go |
| P3 | Browser Chrome/Edge (desktop) | Untuk menguji notifikasi OS; wajib HTTPS/localhost (Secure Context) |
| P4 | Akun admin & teknisi | Login sebagai admin dan siapkan akun teknisi |
| P5 | Data CCTV | Siapkan CCTV uji |
| P6 | Izinkan notifikasi | Pastikan notifikasi situs diizinkan (muncul prompt izin saat opt-in) |

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

Hasil yang diharapkan: seluruh perintah sukses; lint tidak memiliki error (0 error).

| Item | Observasi historis (non-authoritative) |
|------|-------------|
| `go build ./...` | NOT RUN |
| `go test ./... -count=1` (subset `-run "Push\|Notify"`) | NOT RUN |
| `go vet ./...` | NOT RUN |
| `npm run lint` (frontend) | NOT RUN |
| `npm run build` (frontend) | NOT RUN |

## 3. Skenario Uji (Blackbox)

> **Catatan legacy:** tabel di bawah adalah observasi historis/non-authoritative; jangan gunakan sebagai status formal. Isi `Formal scenario record` di atas dengan expected, actual, evidence, dan status formal.

### S1 - Akses & tampilan opt-in di halaman profil

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Login sebagai teknisi, buka `/profile` | Muncul bagian "Notifikasi Push" dengan tombol "Aktifkan" | NOT RUN |
| 2 | Buka di browser modern (Chrome/Edge) | Status awal menampilkan tombol "Aktifkan" | NOT RUN |
| 3 | Buka di browser yang tidak mendukung Push API | Muncul keterangan "tidak mendukung notifikasi push" | NOT RUN |

### S2 - Opt-in & izin browser

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Klik "Aktifkan" | Browser menampilkan prompt izin notifikasi | NOT RUN |
| 2 | Pilih "Izinkan" | Status berubah menjadi aktif; muncul pesan "Aktif — Anda akan menerima notifikasi push" |NOT RUN — Tidak aktif dan ada pesan "Failed to execute 'subscribe' on 'PushManager': The provided applicationServerKey is not valid."|
| 3 | Cek DevTools → Application → Service Workers → Push | Terdapat subscription dengan endpoint aktif | NOT RUN |
| 4 | Cek database `push_subscriptions` | Terdapat baris baru dengan `user_id`, `endpoint`, `key_p256dh`, `key_auth` | NOT RUN |

### S3 - Mengirim push saat tugas baru ditugaskan

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Admin membuat log pemeliharaan lalu menugaskan ke teknisi | Teknisi yang berlangganan push menerima notifikasi OS "tugas baru" | NOT RUN |
| 2 | Buka notifikasi | Service worker membuka halaman `/maintenance` (tujuan berdasarkan reference_type) | NOT RUN |
| 3 | Cek bell notifikasi in-app | Notifikasi in-app juga muncul (push + in-app berjalan beriringan) | NOT RUN Belum berhasil |

### S4 - Push saat eskalasi overdue

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Buat log `scheduled`/`ongoing` yang melewati jadwal | Setelah checker overdue berjalan, teknisi (bila ongoing) dan admin menerima notifikasi push "overdue" | NOT RUN |
| 2 | Buka notifikasi | Terarah ke halaman yang relevan | NOT RUN |

### S5 - Push saat hasil pengecekan `attention`

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Teknisi membuat pengecekan lapangan dengan `overall_status = attention` | Admin yang berlangganan push menerima notifikasi OS "perlu perhatian" | NOT RUN |
| 2 | Buka notifikasi | Terarah ke halaman field maintenance | NOT RUN |

### S6 - Opt-out / nonaktifkan

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Buka `/profile`, klik "Nonaktifkan" | Status berubah menjadi tidak aktif; subscription dihapus dari browser | NOT RUN |
| 2 | Cek database `push_subscriptions` | Baris subscription untuk endpoint tersebut dihapus (soft delete) | NOT RUN |
| 3 | Admin menugaskan tugas baru lagi | Teknisi TIDAK lagi menerima notifikasi OS (sudah opt-out) | NOT RUN |

### S7 - Fallback ke in-app saat VAPID tidak dikonfigurasi

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Kosongkan `VAPID_*` di `backend/.env.docker`, restart API | Aplikasi tetap berjalan; tidak ada error fatal | NOT RUN |
| 2 | Tugaskan tugas baru | Notifikasi in-app tetap muncul (bell); push tidak dikirim | NOT RUN |
| 3 | Periksa log API | Muncul warning ringan (bukan fatal) bahwa VAPID tidak dikonfigurasi | NOT RUN |

### S8 - Konsistensi payload & navigasi notifikasi

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Terima notifikasi push | Judul & isi (body) sesuai notifikasi in-app yang dibuat backend | NOT RUN Belum berhasil |
| 2 | Klik notifikasi saat aplikasi terbuka di tab lain | Tab difokuskan dan diarahkan ke halaman sesuai reference_type | NOT RUN Belum berhasil |
| 3 | Klik notifikasi saat aplikasi tertutup | Tab baru terbuka menuju halaman tujuan | NOT RUN Belum berhasil |

### S9 - Keamanan & scope per user

| # | Langkah | Hasil yang diharapkan | Observasi historis (non-authoritative) |
|---|---------|------------------------|-------------|
| 1 | Login sebagai teknisi A, aktifkan push | Hanya teknisi A yang menerima push miliknya | NOT RUN Belum bisa dicoba |
| 2 | Tugaskan tugas ke teknisi B | Teknisi A tidak menerima push tugas B (scope per user) | NOT RUNBelum bisa dicoba |
| 3 | Periksa endpoint `/api/push/subscribe` via Swagger | Endpoint protected (401 tanpa sesi); hanya user yang login dapat subscribe | NOT RUN |

## 4. Kriteria Penerimaan

1. Model `PushSubscription` + migrasi `000002_push_subscriptions` tersedia; index `(user_id)`.
2. Endpoint `POST /api/push/subscribe` & `DELETE /api/push/unsubscribe` protected dan berfungsi (idempotent).
3. Util `SendPush` mengirim push (VAPID) ke semua subscription user; bila VAPID kosong → no-op + fallback in-app (tidak fatal).
4. `NotifyUser`/`NotifyUserSync` juga memicu push (async, kegagalan hanya warning).
5. Service worker menangani event `push` → `showNotification` dan `notificationclick` → navigasi.
6. UX opt-in di halaman profil; subscribe/unsubscribe terhubung ke backend.
7. `go build`/`go test`/`go vet`, `npm run lint`/`npm run build` memenuhi kriteria.

## Catatan

- Push hanya berjalan pada **Secure Context** (https / localhost).
- VAPID keys harus cocok antara backend (`VAPID_PRIVATE_KEY`) dan frontend (`NEXT_PUBLIC_VAPID_PUBLIC_KEY`); jika tidak, subscribe dapat gagal.
- Kegagalan mengirim push tidak pernah menggagalkan operasi utama (log warning saja).
- Data push tidak disimpan di IndexedDB; notifikasi OS hanya muncul saat push diterima dari server.

## Observasi Historis (Non-authoritative) — Catatan Verifikator

**Kesimpulan historis (non-authoritative): skenario fallback/in-app NOT RUN; push OS end-to-end BELUM dapat diverifikasi** (S1, S4–S6, S7 via verifikasi kode + API nyata; pengujian push OS memerlukan VAPID valid + uji ulang browser sesuai catatan di bawah).

- **Push OS nyata tidak dapat diuji end-to-end** di produksi: VAPID keys (`VAPID_PUBLIC_KEY`/`VAPID_PRIVATE_KEY`/`VAPID_EMAIL`) tidak dikonfigurasi pada deployment prod → `SendPush` no-op + warning, sesuai S7. Fitur dalam kondisi fallback yang dijamin desain; setelah VAPID diisi, perlu pengujian browser ulang (S2 #1-3, S3, S4, S5, S8).
- **Pengujian otomatis:** `go build ./...` NOT RUN, `go vet ./...` NOT RUN, `go test` (termasuk `handlers/push_test.go` & `utils/push_test.go` — subscribe/upsert/validasi/unauthorized, no-op VAPID kosong, kirim ke semua subscription, format payload) NOT RUN; frontend `npm run lint` 0 error & `npm run build` NOT RUN.
- **API nyata yang diuji di produksi:** subscribe → 201 (create) / 200 (upsert idempoten) / 400 (body kosong) / 401 (tanpa token); unsubscribe → 200 (idempoten) / 401 (tanpa token).
- Data `push_subscriptions` tidak dicek langsung via psql (VPS tanpa akses SSH non-interaktif); verifikasi via respons API + unit test + migrasi `000002_push_subscriptions`.
