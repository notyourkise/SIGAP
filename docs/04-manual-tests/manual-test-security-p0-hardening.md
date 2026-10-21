# Manual Test - Hardening Keamanan P0

| Item | Detail |
|------|--------|
| **Task ID** | Security P0 Hardening |
| **Branch** | `bugfix/security-p0-hardening` |
| **Versi Backend** | 0.64.6 (setelah fix) |
| **Versi Frontend** | 1.55.5 (setelah fix) |
| **Tanggal** | 2026-10-18 |

> Tambahan 2026-10-21: Test Case 7 mencakup isolasi site pada dashboard, field maintenance, dan daftar site.

---

## 1. Prasyarat

- [ ] Docker-compose running (`db`, `redis`, `api`, `mediamtx`, `nginx`, `frontend`)
- [ ] File `.env` dan `secrets/` tersedia di host (nilai produksi)
- [ ] Login sebagai **admin**; tersedia satu akun **teknisi** yang hanya ditugaskan ke site A
- [ ] Minimal tersedia **2 site** berbeda (site A dan site B)

---

## 2. Pengujian Otomatis (Build & Test)

### Backend
```bash
cd backend
go build ./...            # Harus lolos tanpa error
go vet ./...              # Harus bersih
go test ./... -count=1    # Semua paket harus ok
```

### Frontend
```bash
cd frontend
npm run lint              # 0 error (warning lama diabaikan)
npm run build             # Production build harus sukses
```

**Kriteria Lulus:** Semua perintah di atas lolos.

---

## 3. Skenario Uji Manual

### **Test Case 1: Secret DB tidak lagi di docker-compose**

| Step | Langkah | Hasil Yang Diharapkan | Hasil Nyata |
|------|---------|---------------------|-------------|
| 1 | Jalankan `Select-String -Path docker-compose.yml -Pattern 'DB_PASSWORD'` | Tidak ada nilai password literal (hanya referensi env/secret) | ✅ / ❌ |
| 2 | `docker compose -f docker-compose.yml config` | `POSTGRES_PASSWORD` terisi dari `.env`/secret, bukan literal di file | ✅ / ❌ |
| 3 | Restart stack, login ke aplikasi | Aplikasi tetap konek DB (tidak ada error auth DB) | ✅ / ❌ |

**Kriteria Lulus:** Tidak ada kredensial hardcoded; aplikasi tetap berfungsi.

---

### **Test Case 2: File sensitif tidak masuk build context Docker**

| Step | Langkah | Hasil Yang Diharapkan | Hasil Nyata |
|------|---------|---------------------|-------------|
| 1 | Cek `backend/.dockerignore` memuat `.env` dan `.env.*` | Kedua pola ada | ✅ / ❌ |
| 2 | Cek `.dockerignore` root memuat `secrets/`, `**/.env*`, `*.key`, `*.pem` | Semua pola ada | ✅ / ❌ |
| 3 | `docker build` image backend, lalu telusuri image | Tidak ditemukan `.env`/secret di dalam image | ✅ / ❌ |

**Kriteria Lulus:** Tidak ada file rahasia yang ikut ter-build.

---

### **Test Case 3: `.env.example` bebas kredensial asli**

| Step | Langkah | Hasil Yang Diharapkan | Hasil Nyata |
|------|---------|---------------------|-------------|
| 1 | Buka `backend/.env.example` | `VAPID_PUBLIC_KEY`/`VAPID_PRIVATE_KEY` kosong | ✅ / ❌ |
| 2 | Cek `RTSP_NVR_HOST`/`RTSP_NVR_USER` | Placeholder (`192.168.1.100` / `GANTI_NVR_USER`), bukan nilai asli | ✅ / ❌ |
| 3 | Konfirmasi VAPID & kredensial NVR lama **sudah dirotasi** | Kunci baru aktif di produksi | ✅ / ❌ |

**Kriteria Lulus:** Contoh env tidak membocorkan kredensial; kredensial lama dirotasi.

---

### **Test Case 4: Service worker tidak menyimpan respons API**

| Step | Langkah | Hasil Yang Diharapkan | Hasil Nyata |
|------|---------|---------------------|-------------|
| 1 | Buka DevTools → Application → Service Workers, pastikan SW v4 aktif | SW `sigap-shell-v4` running | ✅ / ❌ |
| 2 | Buka Cache Storage, periksa isi cache | Hanya manifest & ikon shell; tidak ada entri `/api/...` atau HTML dashboard | ✅ / ❌ |
| 3 | Muat dashboard, lalu logout; periksa Cache Storage lagi | Tidak ada HTML/data terautentikasi tersimpan | ✅ / ❌ |
| 4 | Matikan network, muat ulang halaman | Muncul halaman offline generik (bukan data lama/user lain) | ✅ / ❌ |

**Kriteria Lulus:** Tidak ada data terautentikasi tersimpan di Cache Storage; offline fallback generik.

---

### **Test Case 5: SSRF diblokir pada pembuatan webhook**

| Step | Langkah | Hasil Yang Diharapkan | Hasil Nyata |
|------|---------|---------------------|-------------|
| 1 | Admin buat webhook dengan URL `http://169.254.169.254/latest/meta-data/` | Ditolak: validasi URL gagal (bukan 2xx) | ✅ / ❌ |
| 2 | Coba `http://127.0.0.1:5432`, `http://10.0.0.5/`, `http://[::1]/` | Semua ditolak | ✅ / ❌ |
| 3 | Coba skema non-http: `file:///etc/passwd` | Ditolak (hanya http/https) | ✅ / ❌ |
| 4 | Buat webhook URL publik valid (mis. endpoint uji) | Berhasil dibuat; Test Webhook mengirim request | ✅ / ❌ |

**Kriteria Lulus:** Target internal/non-http diblokir; endpoint publik valid tetap bisa.

---

### **Test Case 6: Isolasi site ditegakkan (user non-admin)**

| Step | Langkah | Hasil Yang Diharapkan | Hasil Nyata |
|------|---------|---------------------|-------------|
| 1 | Login teknisi (hanya ditugaskan site A) | Berhasil login | ✅ / ❌ |
| 2 | Request `GET /api/cctv` dengan header `X-Site-ID: <site A>` | 200 OK, data site A | ✅ / ❌ |
| 3 | Request `GET /api/cctv` dengan header `X-Site-ID: <site B>` | 403 "Akses ditolak: Anda tidak ditugaskan ke site ini" | ✅ / ❌ |
| 4 | Login admin, request site B yang sama | 200 OK (admin akses semua site) | ✅ / ❌ |
| 5 | Hapus cookie `sigap_site_id`, request `GET /api/cctv` sebagai teknisi yang ditugaskan | 200 OK, hanya data site penugasan default (bukan seluruh site) | ✅ / ❌ |
| 6 | Request `GET /api/cctv` sebagai non-admin tanpa penugasan site | 403 "Anda belum ditugaskan ke site mana pun" | ✅ / ❌ |

**Kriteria Lulus:** Non-admin hanya bisa mengakses site yang ditugaskan; tanpa cookie tetap mendapat scope site default; tanpa penugasan ditolak (fail closed); admin bebas.

---

### **Test Case 7: Scope site dashboard, field maintenance, dan daftar site**

| Step | Langkah | Hasil Yang Diharapkan | Hasil Nyata |
|------|---------|---------------------|-------------|
| 1 | Login teknisi site A, buka `GET /api/dashboard/summary` | 200; distribusi status, tren, dan overdue hanya menghitung CCTV/log site A | ✅ / ❌ |
| 2 | Buka `GET /api/dashboard/tasks` | 200; hanya tugas site A milik teknisi tersebut | ✅ / ❌ |
| 3 | Buka `GET /api/field-maintenance` | 200; hanya catatan yang CCTV-nya berada di site A | ✅ / ❌ |
| 4 | `POST /api/field-maintenance` dengan `cctv_id` milik site B | 403 "Akses ditolak: CCTV berada di site lain" | ✅ / ❌ |
| 5 | Buka `GET /api/sites` dan `GET /api/sites/all` | 200; hanya site yang ditugaskan (site A), bukan seluruh site | ✅ / ❌ |
| 6 | Buka `GET /api/sites/<site B>` dengan konteks site A | 404 (site di luar penugasan tidak dapat dibuka); 403 bila memaksa header `X-Site-ID: <site B>` | ✅ / ❌ |
| 7 | Login admin tanpa site aktif, ulangi langkah 1 dan 3 | 200; data global seluruh site | ✅ / ❌ |
| 8 | Buka WebSocket sebagai teknisi, lalu lakukan aksi yang menulis audit log | Tidak menerima event `AUDIT_LOG_ADDED`; koneksi admin tetap menerima | ✅ / ❌ |

**Kriteria Lulus:** Dashboard, field maintenance, dan daftar site mengikuti site aktif/penugasan; pembuatan lintas site ditolak; admin tetap global; event audit tidak bocor ke non-admin.

---

## 4. Kriteria Lulus Umum

✅ **PASS** jika:
- Tidak ada kredensial hardcoded (compose/env.example/secret yang ikut ter-commit)
- File sensitif tidak masuk image/build context
- Service worker tidak menyimpan respons API/HTML terautentikasi
- URL webhook internal/non-http ditolak
- Non-admin tidak bisa mengakses site di luar penugasannya, termasuk dashboard, field maintenance, dan daftar site
- `go test ./...` dan `npm run build` hijau; tidak ada error di log backend

❌ **FAIL** jika:
- Masih ada secret/kredensial ter-baca dari file yang ter-commit
- Data terautentikasi tersimpan di Cache Storage
- Webhook internal/publik tak terduga lolos validasi
- Non-admin bisa mengakses site di luar penugasannya (200, bukan 403)
- Ada regresi pada paket test/atau build

---

## 5. Catatan Tambahan

### Verifikasi Database (site membership)
```sql
-- Keanggotaan user pada site
SELECT u.username, s.kode, s.nama
FROM user_sites us
JOIN users u ON u.id = us.user_id
JOIN sites s ON s.id = us.site_id
ORDER BY u.username;
```

### Catatan Rotasi Kredensial
Kunci VAPID dan kredensial NVR yang pernah ter-commit pada history git harus dianggap **kompromis**. Rotasi wajib dilakukan di sisi layanan (generate VAPID baru; ubah user/password NVR), lalu perbarui `.env`/`secrets/` di server. Tindakan ini di luar scope PR dan menjadi tanggung jawab operasional.

---

## 6. Rollback Plan

Jika ditemukan regresi kritis:

```bash
git checkout main
git revert --no-edit <merge-commit-hash>
git push origin main
```

---

**Tester**: ________________  
**Tanggal Testing**: ________________  
**Status**: ⬜ Pass / ⬜ Fail / ⬜ Blocked  
**Catatan**: _______________________________________________________
