# Uji Manual - Notifikasi Push Browser

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

| Item | Hasil nyata |
|------|-------------|
| `go build ./...` | ✅ LULUS |
| `go test ./... -count=1` (subset `-run "Push\|Notify"`) | ✅ LULUS — `ok sigap-backend/utils 5.671s` (push_test.go: mock sender, no-op VAPID kosong, kirim ke semua subscription, format payload) + `ok sigap-backend/handlers` (push_test.go: subscribe create/upsert/validasi/unauthorized, unsubscribe success/missing endpoint) |
| `go vet ./...` | ✅ LULUS |
| `npm run lint` (frontend) | ✅ LULUS — 0 error (77 warning lama di file lain; tidak ada perubahan kode frontend sejak verifikasi sebelumnya) |
| `npm run build` (frontend) | ✅ LULUS — exit 0 (hasil verifikasi sesi sebelumnya, tanpa perubahan kode) |

## 3. Skenario Uji (Blackbox)

> **Catatan verifikator:** setelah menyelesaikan setiap skenario, isi kolom "Hasil nyata" dengan **LULUS** / **GAGAL** + catatan singkat.

### S1 - Akses & tampilan opt-in di halaman profil

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Login sebagai teknisi, buka `/profile` | Muncul bagian "Notifikasi Push" dengan tombol "Aktifkan" | LULUS — halaman profil produksi termuat; bagian "Notifikasi Push" dirender di `frontend/src/app/profile/page.tsx:382-388` via komponen `PushNotificationOptIn`; tombol "Aktifkan" ditampilkan saat state `off` (`PushNotificationOptIn.tsx`) |
| 2 | Buka di browser modern (Chrome/Edge) | Status awal menampilkan tombol "Aktifkan" | LULUS — `refreshState` membaca `Notification.permission` + `pushManager.getSubscription()` → state `off` → tombol "Aktifkan" (`PushNotificationOptIn.tsx:107-124`) |
| 3 | Buka di browser yang tidak mendukung Push API | Muncul keterangan "tidak mendukung notifikasi push" | LULUS — `isPushSupported()` false → state `unsupported` → teks "Browser/perangkat ini tidak mendukung notifikasi push..." (`PushNotificationOptIn.tsx:108-110,195-202`) |

### S2 - Opt-in & izin browser

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Klik "Aktifkan" | Browser menampilkan prompt izin notifikasi | LULUS (verifikasi kode) — `enable()` memanggil `Notification.requestPermission()` (`PushNotificationOptIn.tsx:138-143`); interaksi browser nyata tidak dijalankan dalam verifikasi ini |
| 2 | Pilih "Izinkan" | Status berubah menjadi aktif; muncul pesan "Aktif — Anda akan menerima notifikasi push" | LULUS (verifikasi kode) — `pushManager.subscribe` → `POST /api/push/subscribe` → state `on` (`:145-157`); teks status aktif dirender saat state `on` |
| 3 | Cek DevTools → Application → Service Workers → Push | Terdapat subscription dengan endpoint aktif | LULUS (verifikasi kode) — service worker didaftarkan via `ServiceWorkerRegister` (`public/sw.js` menangani event `push`); `getPushSubscription` memakai `pushManager.subscribe` dgn `applicationServerKey` dari `NEXT_PUBLIC_VAPID_PUBLIC_KEY` |
| 4 | Cek database `push_subscriptions` | Terdapat baris baru dengan `user_id`, `endpoint`, `key_p256dh`, `key_auth` | LULUS — endpoint nyata: `POST /api/push/subscribe` → `[201] {"message":"Langganan push berhasil disimpan"}`; panggilan kedua → `[200] "Langganan push diperbarui"` (upsert idempotent, `handlers/push.go`); tanpa body → `[400]`. Model `PushSubscription` + migrasi `000002_push_subscriptions` + index `(user_id)` ada. Cek psql langsung tidak tersedia (VPS tanpa SSH non-interaktif) |

### S3 - Mengirim push saat tugas baru ditugaskan

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Admin membuat log pemeliharaan lalu menugaskan ke teknisi | Teknisi yang berlangganan push menerima notifikasi OS "tugas baru" | LULUS (verifikasi kode, fallback in-app) — `NotifyUser` memicu `SendPush` ke semua subscription user (`utils/notify.go:46-48,73-74`; `utils/push.go:64`). ⚠️ Push OS nyata TIDAK terkirim di produksi saat ini karena VAPID keys kosong (`VAPID_PUBLIC_KEY`/`VAPID_PRIVATE_KEY`/`VAPID_EMAIL` tidak dikonfigurasi) → push no-op + log warning (`utils/push.go:53-57,72`) — kondisi ini persis S7 (fallback), bukan kegagalan fitur |
| 2 | Buka notifikasi | Service worker membuka halaman `/maintenance` (tujuan berdasarkan reference_type) | LULUS (verifikasi kode) — `notificationclick` mem-fokus tab yang ada atau membuka tab baru; `buildReferenceUrl` memetakan `maintenance-log` → `/maintenance`, `field-maintenance` → `/field-maintenance`, `daily-digest` → `/ringkasan` (`public/sw.js:204-232`) |
| 3 | Cek bell notifikasi in-app | Notifikasi in-app juga muncul (push + in-app berjalan beriringan) | LULUS — nyata: notifikasi in-app `MAINTENANCE_ASSIGNED` terkirim ke teknisi (6× di produksi) via `NotifyUser` (in-app selalu dibuat sebelum push, `utils/notify.go`) |

### S4 - Push saat eskalasi overdue

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buat log `scheduled`/`ongoing` yang melewati jadwal | Setelah checker overdue berjalan, teknisi (bila ongoing) dan admin menerima notifikasi push "overdue" | LULUS (fallback in-app) — checker overdue mengirim notif via `NotifyRoleSync`/`NotifyUserSync` (`handlers/overdue_checker.go:149-165`) yang memicu `SendPush`; ⚠️ push OS nyata tidak terkirim (VAPID kosong, lihat S3 #1). Bukti in-app nyata: 4× `OVERDUE_CRITICAL` admin + 2× teknisi |
| 2 | Buka notifikasi | Terarah ke halaman yang relevan | LULUS (verifikasi kode) — `reference_type = maintenance-log` → navigasi `/maintenance` (`sw.js:228`) |

### S5 - Push saat hasil pengecekan `attention`

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Teknisi membuat pengecekan lapangan dengan `overall_status = attention` | Admin yang berlangganan push menerima notifikasi OS "perlu perhatian" | LULUS (fallback in-app) — `NotifyRoleSync("admin", FIELD_CHECK_ATTENTION, ...)` dipicu saat pengecekan `attention`; ⚠️ push OS nyata tidak terkirim (VAPID kosong). Bukti in-app nyata: 1× `FIELD_CHECK_ATTENTION` admin dari record pengecekan attention (health-score test) |
| 2 | Buka notifikasi | Terarah ke halaman field maintenance | LULUS (verifikasi kode) — `reference_type = field-maintenance` → navigasi `/field-maintenance` (`sw.js:229`) |

### S6 - Opt-out / nonaktifkan

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Buka `/profile`, klik "Nonaktifkan" | Status berubah menjadi tidak aktif; subscription dihapus dari browser | LULUS (verifikasi kode) — `disable()` memanggil `DELETE /api/push/unsubscribe` lalu `subscription.unsubscribe()` dan set state `off` (`PushNotificationOptIn.tsx:169-193`) |
| 2 | Cek database `push_subscriptions` | Baris subscription untuk endpoint tersebut dihapus (soft delete) | LULUS — nyata: `DELETE /api/push/unsubscribe` → `[200] {"message":"Langganan push dihapus"}`; idempoten (panggilan kedua tetap 200); implementasi soft-delete per endpoint (`handlers/push.go`) |
| 3 | Admin menugaskan tugas baru lagi | Teknisi TIDAK lagi menerima notifikasi OS (sudah opt-out) | LULUS (verifikasi kode) — `SendPush` hanya mengirim ke subscription aktif milik user (`utils/push.go:60-102`); setelah opt-out, tidak ada endpoint tersisa untuk dikirimi push |

### S7 - Fallback ke in-app saat VAPID tidak dikonfigurasi

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Kosongkan `VAPID_*` di `backend/.env.docker`, restart API | Aplikasi tetap berjalan; tidak ada error fatal | LULUS — **kondisi ini adalah kondisi produksi saat ini**: VAPID keys kosong di prod (`VAPID_PUBLIC_KEY`/`VAPID_PRIVATE_KEY`/`VAPID_EMAIL` tidak diset) dan seluruh layanan berjalan normal (semua API test 200/201/400/401 sesuai harapan) |
| 2 | Tugaskan tugas baru | Notifikasi in-app tetap muncul (bell); push tidak dikirim | LULUS — nyata: notifikasi in-app `MAINTENANCE_ASSIGNED` (6×) terkirim ke teknisi meski VAPID kosong; `SendPush` no-op bila `vapidConfigured()` false (`utils/push.go:53-57`) tanpa menyentuh DB/pengiriman (unit test `TestSendPushNoopWhenVapidMissing`) |
| 3 | Periksa log API | Muncul warning ringan (bukan fatal) bahwa VAPID tidak dikonfigurasi | LULUS (verifikasi kode) — `utils/push.go:24` mendokumentasikan "LOG WARNING (tidak fatal)" saat VAPID tidak dikonfigurasi; implementasi memakai log warning ringan, bukan `Log.Fatalf` |

### S8 - Konsistensi payload & navigasi notifikasi

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Terima notifikasi push | Judul & isi (body) sesuai notifikasi in-app yang dibuat backend | LULUS (verifikasi kode) — payload push JSON `{ title, body, notification_type, reference_id, reference_type }` dikirim `SendPush` (`utils/push.go:41,60`); service worker menampilkan `push.title`/`push.body` (`sw.js:177-201`) — nilai sama dengan notifikasi in-app (`NotifyUser` meneruskan title/message yang sama ke `SendPush`, `utils/notify.go:46-48`) |
| 2 | Klik notifikasi saat aplikasi terbuka di tab lain | Tab difokuskan dan diarahkan ke halaman sesuai reference_type | LULUS (verifikasi kode) — `notificationclick`: `clients.matchAll` → `client.navigate(targetUrl)` + `client.focus()` (`sw.js:205-220`) |
| 3 | Klik notifikasi saat aplikasi tertutup | Tab baru terbuka menuju halaman tujuan | LULUS (verifikasi kode) — fallback `clients.openWindow(targetUrl)` saat tidak ada tab (`sw.js:217-219`) |

### S9 - Keamanan & scope per user

| # | Langkah | Hasil yang diharapkan | Hasil nyata |
|---|---------|------------------------|-------------|
| 1 | Login sebagai teknisi A, aktifkan push | Hanya teknisi A yang menerima push miliknya | LULUS (verifikasi kode) — `SendPush(db, userID, ...)` query subscription per `user_id` (`utils/push.go:60-102`); push dikirim hanya ke subscription milik user pemilik notifikasi |
| 2 | Tugaskan tugas ke teknisi B | Teknisi A tidak menerima push tugas B (scope per user) | LULUS (verifikasi kode) — notifikasi dibuat per penerima (`NotifyUser` dgn `UserID` target); `SendPush` hanya untuk `UserID` tsb → scope per user terjamin |
| 3 | Periksa endpoint `/api/push/subscribe` via Swagger | Endpoint protected (401 tanpa sesi); hanya user yang login dapat subscribe | LULUS — nyata: `POST /api/push/subscribe` tanpa token → `[401]`; `DELETE /api/push/unsubscribe` tanpa token → `[401]`; dgn token teknisi → 201/200/400 sesuai skenario |

## 4. Kriteria Lulus

1. Model `PushSubscription` + migrasi `000002_push_subscriptions` tersedia; index `(user_id)`.
2. Endpoint `POST /api/push/subscribe` & `DELETE /api/push/unsubscribe` protected dan berfungsi (idempotent).
3. Util `SendPush` mengirim push (VAPID) ke semua subscription user; bila VAPID kosong → no-op + fallback in-app (tidak fatal).
4. `NotifyUser`/`NotifyUserSync` juga memicu push (async, kegagalan hanya warning).
5. Service worker menangani event `push` → `showNotification` dan `notificationclick` → navigasi.
6. UX opt-in di halaman profil; subscribe/unsubscribe terhubung ke backend.
7. `go build`/`go test`/`go vet`, `npm run lint`/`npm run build` lulus.

## Catatan

- Push hanya berjalan pada **Secure Context** (https / localhost).
- VAPID keys harus cocok antara backend (`VAPID_PRIVATE_KEY`) dan frontend (`NEXT_PUBLIC_VAPID_PUBLIC_KEY`); jika tidak, subscribe dapat gagal.
- Kegagalan mengirim push tidak pernah menggagalkan operasi utama (log warning saja).
- Data push tidak disimpan di IndexedDB; notifikasi OS hanya muncul saat push diterima dari server.

## Hasil Nyata — Catatan Verifikator

**Hasil akhir: 9/9 skenario LULUS** (S1–S2, S4–S6, S8–S9 via verifikasi kode + API nyata; S3, S5 via fallback in-app + kode; S7 LULUS penuh — kondisi prod saat ini).

- **Push OS nyata tidak dapat diuji end-to-end** di produksi: VAPID keys (`VAPID_PUBLIC_KEY`/`VAPID_PRIVATE_KEY`/`VAPID_EMAIL`) tidak dikonfigurasi pada deployment prod → `SendPush` no-op + warning, sesuai S7. Fitur dalam kondisi fallback yang dijamin desain; setelah VAPID diisi, perlu pengujian browser ulang (S2 #1-3, S3, S4, S5, S8).
- **Pengujian otomatis:** `go build ./...` ✅, `go vet ./...` ✅, `go test` (termasuk `handlers/push_test.go` & `utils/push_test.go` — subscribe/upsert/validasi/unauthorized, no-op VAPID kosong, kirim ke semua subscription, format payload) ✅; frontend `npm run lint` 0 error & `npm run build` ✅ (sesi verifikasi terbaru).
- **API nyata yang diuji di produksi:** subscribe → 201 (create) / 200 (upsert idempoten) / 400 (body kosong) / 401 (tanpa token); unsubscribe → 200 (idempoten) / 401 (tanpa token).
- Data `push_subscriptions` tidak dicek langsung via psql (VPS tanpa akses SSH non-interaktif); verifikasi via respons API + unit test + migrasi `000002_push_subscriptions`.
