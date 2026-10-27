# Manual Test - Isolasi Site Log Stream CCTV & WebSocket

| Item | Detail |
|------|--------|
| **Issue** | #139 (P0 — Isolasi log/event CCTV per site) |
| **Branch** | `bugfix/cctv-log-site-isolation` |
| **Versi Backend** | 0.64.32 (setelah fix) |
| **Versi Frontend** | 1.55.35 (setelah fix) |
| **Tanggal** | 2026-10-27 |

---

## 1. Prasyarat

- [ ] Docker Compose berjalan: `db`, `redis`, `api`, `mediamtx`, `nginx`, `frontend`.
- [ ] Tersedia minimal **2 site** (site A dan site B) dengan masing-masing minimal 1 CCTV.
- [ ] Tersedia akun **admin**.
- [ ] Tersedia **teknisi A** yang hanya ditugaskan ke site A dan **teknisi B** yang hanya ditugaskan ke site B.
- [ ] Tersedia akun teknisi tanpa penugasan site (opsional, untuk uji fail closed).
- [ ] Minimal ada beberapa baris `cctv_stream_logs` di kedua site (bisa dihasilkan dari probe/verifikasi stream).
- [ ] Catat pemetaan CCTV → site agar hasil filter dapat diverifikasi.

Verifikasi pemetaan dan jumlah log:

```sql
-- Pemetaan kamera ke site
SELECT c.id, c.kode_cctv, c.nama, s.kode AS site_kode, s.nama AS site_nama
FROM cctvs c
JOIN sites s ON s.id = c.site_id
WHERE c.deleted_at IS NULL
ORDER BY s.kode, c.kode_cctv;

-- Jumlah log per site
SELECT s.kode, COUNT(l.id) AS total_log
FROM cctv_stream_logs l
JOIN cctvs c ON c.id = l.cctv_id
JOIN sites s ON s.id = c.site_id
GROUP BY s.kode
ORDER BY s.kode;
```

---

## 2. Pengujian Otomatis

### Backend

```bash
cd backend
go build ./...
go test ./handlers -run 'TestGetCCTVStreamLogs' -count=1 -v
go test ./utils -run 'TestWSClientInfo_CanReceiveSiteEvent|TestBroadcastToSite_FiltersRecipients' -count=1 -v
go test ./routes -run 'TestWebSocketHandshake' -count=1 -v
go test ./... -count=1
```

**Kriteria lulus:** seluruh perintah lolos. Test `BroadcastToSite_FiltersRecipients`
membuka tiga koneksi WebSocket nyata (teknisi A, teknisi B, admin) dan memastikan
event site A hanya diterima teknisi A + admin.

### Frontend

```bash
cd frontend
npx eslint src/app/cctv/logs/page.tsx
npx tsc --noEmit
```

**Kriteria lulus:** tidak ada error. Warning lama `react-hooks/set-state-in-effect`
pada baris 140 bukan berasal dari perubahan ini dan boleh diabaikan.

---

## 3. Skenario Uji Manual REST API

Gunakan header `X-Site-ID` atau cookie `sigap_site_id` untuk memilih site aktif.
Endpoint yang diuji: `GET /api/cctv/logs` (alias `GET /api/cctv-logs`).

### Test Case 1 — Teknisi site A hanya melihat log site A

| Step | Langkah | Hasil Yang Diharapkan | Hasil Nyata |
|------|---------|----------------------|-------------|
| 1 | Login sebagai teknisi A, panggil `GET /api/cctv/logs?limit=100` tanpa header site | 200; site default = site A; seluruh baris hanya milik CCTV site A | ✅ / ❌ |
| 2 | Panggil dengan header `X-Site-ID: <site A>` | 200; hanya log CCTV site A | ✅ / ❌ |
| 3 | Bandingkan `data.pagination.total_rows` dengan query SQL jumlah log site A | Nilai sama | ✅ / ❌ |
| 4 | Periksa `data.summary.today_online_events`, `today_offline_events`, dan `total_events` | Hanya menghitung event site A; `total_events` = `total_rows` | ✅ / ❌ |
| 5 | Pastikan tidak ada `cctv_id`/nama kamera milik site B pada respons | Tidak ada kebocoran site B | ✅ / ❌ |

**Kriteria lulus:** count, list, dan ringkasan konsisten hanya untuk site A.

---

### Test Case 2 — Teknisi tidak dapat memaksa site lain

| Step | Langkah | Hasil Yang Diharapkan | Hasil Nyata |
|------|---------|----------------------|-------------|
| 1 | Login teknisi A, panggil `GET /api/cctv/logs` dengan `X-Site-ID: <site B>` | 403 `"Akses ditolak: Anda tidak ditugaskan ke site ini"` | ✅ / ❌ |
| 2 | Ulangi memakai query `?site_id=<site B>` | 403 (fallback query juga divalidasi) | ✅ / ❌ |
| 3 | Ulangi memakai cookie `sigap_site_id=<site B>` | 403 | ✅ / ❌ |

**Kriteria lulus:** seluruh jalur pemilihan site (header, cookie, query) menegakkan keanggotaan.

---

### Test Case 3 — Non-admin tanpa penugasan (fail closed)

| Step | Langkah | Hasil Yang Diharapkan | Hasil Nyata |
|------|---------|----------------------|-------------|
| 1 | Login sebagai teknisi tanpa `user_sites`, panggil `GET /api/cctv/logs` tanpa header site | 403 `"Anda belum ditugaskan ke site mana pun"` | ✅ / ❌ |
| 2 | Panggil dengan header site mana pun | 403 (tidak ada site yang boleh dipilih) | ✅ / ❌ |
| 3 | Pastikan respons tidak memuat `data.rows` | Tidak ada data sama sekali | ✅ / ❌ |

**Kriteria lulus:** tidak ada fallback ke data global untuk non-admin tanpa penugasan.

---

### Test Case 4 — Admin melihat semua site atau site terpilih

| Step | Langkah | Hasil Yang Diharapkan | Hasil Nyata |
|------|---------|----------------------|-------------|
| 1 | Login admin, panggil `GET /api/cctv/logs?limit=100` tanpa site aktif | 200; log dari seluruh site | ✅ / ❌ |
| 2 | Bandingkan `total_rows` dengan total seluruh baris `cctv_stream_logs` | Nilai sama | ✅ / ❌ |
| 3 | Panggil dengan `X-Site-ID: <site A>` | 200; hanya log site A; total sama dengan query SQL site A | ✅ / ❌ |
| 4 | Ganti ke `X-Site-ID: <site B>` | 200; hanya log site B | ✅ / ❌ |
| 5 | Periksa ringkasan harian pada langkah 3 | Hanya menghitung event site A | ✅ / ❌ |

**Kriteria lulus:** admin tetap global saat tanpa site aktif, namun filter site terpilih berlaku untuk list, count, dan ringkasan.

---

## 4. Skenario Uji Manual WebSocket

Gunakan dua browser/sesi berbeda (mis. jendela normal dan incognito) agar cookie sesi
tidak saling menimpa. Buka DevTools → Network → WS → Frames untuk melihat payload.

> Catatan: koneksi WebSocket memakai cookie httpOnly `sigap_session`; tidak perlu
> menempelkan token di URL. Koneksi memuat penugasan `user_sites` saat handshake.

### Test Case 5 — Event site A tidak diterima teknisi site B

| Step | Langkah | Hasil Yang Diharapkan | Hasil Nyata |
|------|---------|----------------------|-------------|
| 1 | Buka sesi **teknisi A**, **teknisi B**, dan **admin** pada aplikasi | Ketiganya terhubung WebSocket | ✅ / ❌ |
| 2 | Sebagai admin, verifikasi kamera site A: `POST /api/streams/<id-cctv-site-A>/verify` | Teknisi A dan admin menerima `CCTV_LIVE_STATUS_UPDATED` dengan `site_id` = site A | ✅ / ❌ |
| 3 | Periksa frame pada sesi teknisi B | Tidak ada event untuk kamera site A | ✅ / ❌ |
| 4 | Verifikasi kamera site B | Teknisi B dan admin menerima event; teknisi A tidak menerima | ✅ / ❌ |
| 5 | Periksa payload event | Memuat `cctv_id` dan `site_id` yang sesuai | ✅ / ❌ |

**Kriteria lulus:** event live status hanya sampai ke teknisi pemilik site dan admin.

---

### Test Case 6 — Event log stream mengikuti isolasi site

| Step | Langkah | Hasil Yang Diharapkan | Hasil Nyata |
|------|---------|----------------------|-------------|
| 1 | Dengan tiga sesi yang sama, picu perubahan status terverifikasi pada kamera site A (dua kegagalan/keberhasilan berturut-turut hingga `CCTV_STREAM_LOG_CREATED` terbit) | Teknisi A dan admin menerima `CCTV_STREAM_LOG_CREATED` dengan `site_id` site A | ✅ / ❌ |
| 2 | Periksa frame teknisi B | Tidak menerima event log site A | ✅ / ❌ |
| 3 | Ulangi untuk kamera site B | Hanya teknisi B dan admin yang menerima | ✅ / ❌ |

**Kriteria lulus:** event `CCTV_STREAM_LOG_CREATED` tidak lagi global.

> Jika perubahan status sulit dipicu manual, bukti utama diambil dari test otomatis
> `TestBroadcastToSite_FiltersRecipients` dan test handler. Catat kondisi ini pada kolom
> hasil nyata.

---

### Test Case 7 — CCTV tanpa site hanya dikirim ke admin (fail closed)

| Step | Langkah | Hasil Yang Diharapkan | Hasil Nyata |
|------|---------|----------------------|-------------|
| 1 | Siapkan satu CCTV dengan `site_id = NULL` (sementara, di lingkungan uji) | Kamera tetap dapat diverifikasi admin | ✅ / ❌ |
| 2 | Verifikasi kamera tersebut sebagai admin | Admin menerima `CCTV_LIVE_STATUS_UPDATED` dengan `site_id` kosong | ✅ / ❌ |
| 3 | Periksa sesi teknisi A dan B | Tidak ada yang menerima event tersebut | ✅ / ❌ |
| 4 | Pulihkan `site_id` kamera setelah pengujian | Data kembali seperti semula | ✅ / ❌ |

**Kriteria lulus:** event tanpa klasifikasi site tidak bocor ke non-admin.

---

## 5. Skenario Uji Manual Frontend

### Test Case 8 — Halaman log hanya refetch untuk site aktif

| Step | Langkah | Hasil Yang Diharapkan | Hasil Nyata |
|------|---------|----------------------|-------------|
| 1 | Login admin, pilih **site A** pada `SiteSwitcher`, buka `/cctv/logs` | Halaman menampilkan log site A | ✅ / ❌ |
| 2 | Buka DevTools → Network, filter permintaan `logs` | Terlihat request awal halaman | ✅ / ❌ |
| 3 | Sebagai admin di sesi lain, verifikasi kamera **site B** | Tidak ada request `/cctv/logs` baru pada halaman site A | ✅ / ❌ |
| 4 | Verifikasi kamera **site A** | Halaman melakukan refetch dan menampilkan event terbaru | ✅ / ❌ |
| 5 | Ganti `SiteSwitcher` ke **Semua site**, lalu picu event site B | Halaman refetch (tidak ada site aktif untuk difilter) | ✅ / ❌ |
| 6 | Matikan WebSocket sementara, klik tombol segarkan | Data tetap dapat dimuat ulang secara manual | ✅ / ❌ |

**Kriteria lulus:** filter klien hanya mengurangi refetch lintas site; data tetap akurat
dan backend tetap menjadi penentu isolasi.

---

## 6. Kriteria Lulus Umum

✅ **PASS** jika:

- Teknisi hanya dapat membaca log CCTV dari site yang ditugaskan, termasuk count dan ringkasan.
- Teknisi tanpa penugasan mendapat 403 dan tidak menerima data apa pun.
- Admin dapat melihat seluruh site atau satu site terpilih.
- Event WebSocket `CCTV_STREAM_LOG_CREATED`, `CCTV_LIVE_STATUS_UPDATED`, dan `CCTV_UPDATED` hanya diterima koneksi site terkait + admin.
- Event tanpa `site_id` hanya diterima admin.
- Halaman `/cctv/logs` tidak refetch untuk event site lain saat satu site dipilih.
- Seluruh test backend dan pemeriksaan frontend lulus.

❌ **FAIL** jika:

- Ada baris/ringkasan site lain pada respons teknisi.
- Teknisi dapat memilih site di luar penugasan (200, bukan 403).
- Event WebSocket site A diterima teknisi site B.
- Event tanpa site diterima non-admin.
- Ada regresi pada test atau build.

---

## 7. Catatan Deployment

- Perubahan menyentuh backend **dan** frontend; rebuild kedua image diperlukan:
  ```bash
  docker compose build api frontend
  docker compose up -d api frontend
  ```
- Tidak ada migrasi database baru pada perubahan ini (`cctv_stream_logs` tetap memakai relasi `cctv_id`).
- Setelah deploy, pastikan setiap teknisi memiliki baris `user_sites`. Teknisi tanpa
  penugasan akan menerima 403 pada halaman log (perilaku yang diminta).
- Koneksi WebSocket lama akan memuat penugasan site saat handshake berikutnya;
  minta pengguna me-refresh halaman setelah deploy.
- Verifikasi log backend tidak memuat kebocoran nama/kode kamera lintas site.

---

## 8. Rollback Plan

Jika ditemukan regresi kritis:

```bash
git checkout main
git revert --no-edit <merge-commit-hash>
git push origin main
```

Lalu rebuild `api` dan `frontend` seperti pada bagian deployment.

---

**Tester**: ________________
**Tanggal Testing**: ________________
**Status**: ⬜ Pass / ⬜ Fail / ⬜ Blocked
**Catatan**: _______________________________________________________
