# Security Regression Report — Multi-Site & Proxy Spoofing

| Item | Detail |
|------|--------|
| **Issue** | P0 #141 (lanjutan #139 dan #140) |
| **Branch** | `test/security-regression-multi-site-proxy` |
| **Tanggal** | 2026-10-27 |
| **Environment** | Windows 11, Go 1.26.5, `golangci-lint` v2.12.2 |
| **Status** | ✅ PASS |

---

## 1. Scope

Regression test untuk memastikan dua perbaikan keamanan P0 tidak mengalami
regresi:

- **#139** — isolasi log CCTV dan event WebSocket per site.
- **#140** — hardening trusted proxy, validasi IP, dan anti-spoofing
  `X-Forwarded-For`.

Issue #141 meminta empat area pengujian: API log CCTV (admin/teknisi A/B),
WebSocket lintas site, trusted vs untrusted proxy, serta rate limit per endpoint
dan per IP.

---

## 2. Matriks Test

| Area | File Test | Skenario | Hasil |
|------|-----------|----------|-------|
| API log CCTV | `handlers/cctv_stream_log_isolation_test.go` (baru) | Teknisi site A dan site B mengakses endpoint yang sama; masing-masing hanya menerima log/CCTV milik site-nya. Query count, list, preload, dan ringkasan memakai parameter site yang benar. | ✅ PASS |
| API log CCTV | `handlers/cctv_stream_log_test.go` (existing) | Admin global, admin dengan site terpilih, teknisi dengan site, dan non-admin tanpa assignment (403 fail closed). | ✅ PASS |
| WebSocket | `utils/websocket_test.go` (diperkuat) | Event `CCTV_STREAM_LOG_CREATED` site A hanya diterima teknisi site A dan admin; teknisi site B tidak menerima. Payload `site_id`/`nama` diverifikasi. | ✅ PASS |
| WebSocket | `routes/websocket_test.go` (existing) | Handshake teknisi memuat `user_sites` sebelum registrasi Hub. | ✅ PASS |
| Trusted proxy | `config/fiber_test.go` (existing #140) | XFF dari klien langsung diabaikan; XFF dari trusted proxy dipakai; entri invalid dilewati; XFF seluruhnya invalid jatuh ke IP peer. | ✅ PASS |
| Rate limit | `middleware/limiter_test.go` (diperluas) | XFF palsu dari klien langsung tidak memecah bucket; trusted proxy memakai bucket per IP; IP yang sama di endpoint berbeda punya bucket terpisah; peer yang tidak ada di trusted list tidak boleh memakai XFF; trusted proxy CIDR bekerja. | ✅ PASS |

---

## 3. Perintah dan Hasil

Dijalankan dari `backend/`:

```bash
go build ./...                 # PASS
go vet ./...                   # PASS
go test ./... -count=1         # PASS — seluruh paket ok
golangci-lint run ./...        # PASS — 0 issues
```

Catatan:

- Docker tidak tersedia di environment lokal, sehingga verifikasi black-box
  stack penuh tidak dijalankan di sini. Prosedur manual tersedia di
  `docs/04-manual-tests/manual-test-cctv-log-site-isolation.md` dan
  `docs/04-manual-tests/manual-test-trusted-proxy-ip-hardening.md`.
- CI tetap menjalankan `go test -v ./...` dan `golangci-lint` pada PR.

---

## 4. Kesimpulan

Seluruh area yang diminta #141 tercakup oleh test otomatis dan lulus tanpa
regresi. Tidak ada perubahan kode produksi pada issue ini; perubahan hanya
berupa test, dokumentasi hasil pengujian, dan changelog.
