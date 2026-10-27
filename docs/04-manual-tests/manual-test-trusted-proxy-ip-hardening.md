# Manual Test - Hardening Trusted Proxy & Anti-Spoofing IP

| Item | Detail |
|------|--------|
| **Task ID** | P0 #140 |
| **Branch** | `bugfix/trusted-proxy-ip-hardening` |
| **Versi Backend** | 0.64.33 (setelah fix) |
| **Versi Frontend** | 1.55.35 (tidak berubah) |
| **Tanggal** | 2026-10-27 |

---

## 1. Prasyarat

- [ ] Docker Compose berjalan (`db`, `redis`, `api`, `nginx`, `frontend`)
- [ ] `backend/.env.docker` berisi `TRUSTED_PROXY_IPS=172.28.0.10`
- [ ] Akses ke shell container (`docker compose exec`) dan database
- [ ] Tersedia akun uji; **jangan** memakai kredensial production untuk percobaan login gagal
- [ ] Dua sumber koneksi berbeda bila ingin menguji bucket rate limit per IP (mis. host dan container lain)

> Catatan: API tidak mem-publish port ke host. Pengujian "klien langsung" dilakukan
> dari dalam network Docker ke `api:3000`, sedangkan pengujian "melalui proxy"
> dilakukan ke `https://localhost/api/...` (Nginx).

---

## 2. Pengujian Otomatis (Build & Test)

```bash
cd backend
go build ./...            # Harus lolos tanpa error
go vet ./...              # Harus bersih
go test ./config/ ./middleware/ ./utils/ -count=1
go test ./... -count=1    # Seluruh paket harus ok
```

**Kriteria Lulus:** Semua perintah di atas lolos, termasuk test baru
`config/fiber_test.go` dan `middleware/limiter_test.go`.

---

## 3. Skenario Uji Manual

### **Test Case 1: Nginx menimpa X-Forwarded-For dari klien**

| Step | Langkah | Hasil Yang Diharapkan | Hasil Nyata |
|------|---------|---------------------|-------------|
| 1 | Kirim login gagal melalui Nginx dengan header palsu: `curl -k -H "Content-Type: application/json" -H "X-Forwarded-For: 203.0.113.99" -d '{"username":"tidak-ada","password":"salah"}' https://localhost/api/auth/login` | Respons `401` (bukan 429 pada percobaan pertama) | ✅ / ❌ |
| 2 | Tunggu ±1 detik, lalu query audit log terbaru | `ip_address` berisi IP koneksi yang dilihat Nginx, **bukan** `203.0.113.99` | ✅ / ❌ |
| 3 | Cek konfigurasi Nginx | Ketiga lokasi (`/api/`, `/stream/`, `/`) memakai `X-Forwarded-For $remote_addr`, bukan `$proxy_add_x_forwarded_for` | ✅ / ❌ |

**Kriteria Lulus:** Nilai XFF dari klien tidak pernah diteruskan ke backend/frontend/MediaMTX.

---

### **Test Case 2: Klien langsung tidak bisa memalsukan IP**

| Step | Langkah | Hasil Yang Diharapkan | Hasil Nyata |
|------|---------|---------------------|-------------|
| 1 | Dari container non-Nginx, kirim login gagal langsung ke API dengan XFF palsu: `docker compose exec -T api wget -qO- --header "Content-Type: application/json" --header "X-Forwarded-For: 203.0.113.200" --post-data '{"username":"tidak-ada","password":"salah"}' http://127.0.0.1:3000/api/auth/login` | Respons `401` | ✅ / ❌ |
| 2 | Ulangi dengan XFF berbeda (`203.0.113.201`, `203.0.113.202`, dst.) hingga 6 kali | Percobaan ke-6 menghasilkan `429` "Terlalu banyak percobaan login" | ✅ / ❌ |
| 3 | Query audit log untuk percobaan di atas | `ip_address` berisi IP peer langsung (mis. `127.0.0.1`), bukan IP palsu | ✅ / ❌ |

**Kriteria Lulus:** Pergantian XFF tidak membuat bucket rate limit baru dan tidak mengubah audit log.

---

### **Test Case 3: Trusted proxy memakai IP klien yang tervalidasi**

| Step | Langkah | Hasil Yang Diharapkan | Hasil Nyata |
|------|---------|---------------------|-------------|
| 1 | Pastikan `TRUSTED_PROXY_IPS=172.28.0.10` dan container Nginx memakai IP tersebut (`docker inspect sigap_nginx`) | IP Nginx = `172.28.0.10` | ✅ / ❌ |
| 2 | Login berhasil melalui Nginx | `200` + cookie sesi; audit `LOGIN_SUCCESS` mencatat IP koneksi klien | ✅ / ❌ |
| 3 | Kirim header `X-Forwarded-For` berisi nilai tidak valid (`bukan-ip`) melalui Nginx | Backend tidak error; XFF ditimpa Nginx; audit tetap berisi IP koneksi | ✅ / ❌ |

**Kriteria Lulus:** Hanya koneksi dari IP trusted proxy yang memakai XFF; nilai invalid tidak pernah menjadi identitas klien.

---

### **Test Case 4: Rate limit per IP tidak bisa di-bypass**

| Step | Langkah | Hasil Yang Diharapkan | Hasil Nyata |
|------|---------|---------------------|-------------|
| 1 | Dari host, lakukan 6 login gagal melalui Nginx sambil mengganti-ganti header XFF palsu | Percobaan ke-6 → `429` | ✅ / ❌ |
| 2 | Ulangi dari sumber koneksi berbeda (mis. container lain di network) | Bucket terpisah; percobaan pertama sumber kedua tetap diproses | ✅ / ❌ |
| 3 | Tunggu window 1 menit, ulangi | Rate limit kembali normal | ✅ / ❌ |

**Kriteria Lulus:** Batas dihitung per IP klien asli, bukan per nilai header yang bisa dipalsukan.

---

### **Test Case 5: Deployment tanpa proxy mengabaikan XFF**

| Step | Langkah | Hasil Yang Diharapkan | Hasil Nyata |
|------|---------|---------------------|-------------|
| 1 | Set `TRUSTED_PROXY_IPS=` (kosong), restart `api` | Startup mencatat warning bahwa XFF tidak dipercaya | ✅ / ❌ |
| 2 | Kirim request langsung ke API dengan XFF palsu | IP yang dipakai audit/rate limit adalah IP peer, bukan XFF | ✅ / ❌ |
| 3 | Set `TRUSTED_PROXY_IPS=172.28.0.10`, restart `api` | Startup mencatat "Trusted proxy aktif: [172.28.0.10]" | ✅ / ❌ |

**Kriteria Lulus:** Tanpa proxy, `ProxyHeader` tidak aktif sama sekali (default aman).

---

### **Test Case 6: Entri konfigurasi tidak valid dibuang**

| Step | Langkah | Hasil Yang Diharapkan | Hasil Nyata |
|------|---------|---------------------|-------------|
| 1 | Set `TRUSTED_PROXY_IPS=172.28.0.10, bukan-ip, 10.0.0.0/33`, restart `api` | API tetap start; warning menyebut entri `bukan-ip` dan `10.0.0.0/33` diabaikan | ✅ / ❌ |
| 2 | Cek log startup | Hanya `172.28.0.10` yang aktif sebagai trusted proxy | ✅ / ❌ |
| 3 | Set `TRUSTED_PROXY_IPS=bukan-ip` saja, restart `api` | API tetap start dengan warning; seluruh XFF diabaikan | ✅ / ❌ |

**Kriteria Lulus:** Salah konfigurasi tidak membuat API crash dan tidak mempercayai nilai yang tidak valid.

---

### **Test Case 7: Validasi konfigurasi Nginx**

| Step | Langkah | Hasil Yang Diharapkan | Hasil Nyata |
|------|---------|---------------------|-------------|
| 1 | `docker compose exec nginx nginx -t` | `syntax is ok` dan `test is successful` | ✅ / ❌ |
| 2 | `docker compose restart nginx`, lalu buka aplikasi | Halaman dan API tetap berfungsi melalui HTTPS | ✅ / ❌ |
| 3 | Cek log backend saat request melalui Nginx | Tidak ada error parsing IP/header | ✅ / ❌ |

**Kriteria Lulus:** Konfigurasi Nginx valid dan tidak menimbulkan regresi routing.

---

## 4. Kriteria Lulus Umum

✅ **PASS** jika:

- Klien langsung tidak dapat memengaruhi IP melalui `X-Forwarded-For`
- Nginx selalu menimpa `X-Forwarded-For` dengan `$remote_addr`
- Rate limit memakai IP klien asli dan tidak bisa di-bypass dengan header palsu
- Audit log mencatat IP yang benar untuk request langsung maupun melalui proxy
- Deployment tanpa proxy tidak membaca header proxy sama sekali
- Entri `TRUSTED_PROXY_IPS` tidak valid dibuang dengan warning tanpa membuat API gagal start
- Seluruh test otomatis dan `nginx -t` lolos

❌ **FAIL** jika:

- IP palsu dari klien muncul di audit log atau memecah bucket rate limit
- Nginx masih memakai `$proxy_add_x_forwarded_for`
- API gagal start hanya karena ada entri env tidak valid
- Ada regresi pada rate limit, audit log, atau routing Nginx

---

## 5. Rollback Plan

Jika ditemukan regresi kritis:

```bash
git checkout main
git revert --no-edit <merge-commit-hash>
git push origin main
```

Lalu rebuild `api` dan restart `nginx`:

```bash
docker compose up -d --build api nginx
```

---

**Tester**: ________________
**Tanggal Testing**: ________________
**Status**: ⬜ Pass / ⬜ Fail / ⬜ Blocked
**Catatan**: _______________________________________________________
