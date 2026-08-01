# TESTING — Backend SIGAP

> Status pengujian backend dan cakupan lanjutan Fase 4.1.
> Diperbarui: 2026-08-02 (penambahan test config, models, routes, dan edge case handler).

---

## 1. Cara Menjalankan

```bash
cd backend
go build ./...        # build
go test ./... -count=1        # semua test (tanpa cache)
go test ./... -v              # verbose
go test ./handlers/ -run TestCCTV -v   # subset berdasarkan pola
go vet ./handlers/ ./middleware/ ./utils/
```

Semua test berjalan **tanpa PostgreSQL fisik** — memakai `github.com/DATA-DOG/go-sqlmock`.
Jalankan di direktori `backend/` (modul `sigap-backend`).

---

## 2. Matriks Cakupan Test

| Package | File Test | Yang Diuji |
|---------|-----------|-----------|
| `utils` | `pagination_test.go` | default limit/page/sort, cap limit 100 & page 1.000.000, whitelist sort (anti SQL injection), `CalculatePagination`, `ParseDateRangeParams` (valid, one-sided, invalid/missing) |
| `utils` | `encryption_test.go` | round-trip AES-256-GCM, nonce unik tiap enkripsi, hex invalid, input kosong |
| `utils` | `blacklist_test.go` | `RevokeToken`/`IsTokenRevoked`, cleanup token kadaluarsa |
| `utils` | `csrf_test.go` | `GenerateCSRFToken` (64 hex, unik), `ExtractTokenString` (prioritas cookie → Bearer → kosong) |
| `utils` | `jwt_test.go` (lama) | generate & validate token, expiry, signature salah |
| `utils` | `validator_test.go` (lama) | validator custom (`strongpassword`, dll) |
| `middleware` | `auth_test.go` | `Protected`: tanpa token 401, revoked 401, invalid, valid, token-version mismatch 401 → `Sesi telah dicabut...`; `RequireRole`: none/wrong/allowed |
| `middleware` | `csrf_test.go` | metode aman dilewati, Bearer dilewati, tanpa cookie sesi dilewati, mismatch/missing 403, matching lolos |
| `middleware` | `limiter_test.go` | batas bawah lolos, > 100/menit → 429 |
| `middleware` | `security_test.go` | header dasar selalu ada; CSP/HSTS hanya saat `APP_ENV=production` |
| `handlers` | `auth_test.go` (lama + fix) | `Login` (sukses, password salah, user tidak ditemukan, validasi) + `SetupTestDB` dengan `SkipDefaultTransaction: true` |
| `handlers` | `auth_extra_test.go` | ChangePassword (sukses, password lama salah, validasi), GetMyProfile (sukses, not found, tanpa identitas), UpdateMyProfile, Logout, CreateUser (validasi, duplikat, sukses) |
| `handlers` | `user_test.go` | GetAllUsers, GetUserByID, UpdateUser (validasi role + token-version branch), DeleteUser (+ self-delete forbidden), ResetUserPassword |
| `handlers` | `cctv_test.go` | GetCCTVByID, CreateCCTV (validasi, duplikat kode, sukses + `qr_payload` terisi), ScanCCTV (invalid, sukses), DownloadTemplateCCTV, GetCCTVQRCode, ImportCCTV batas baris, Update/Delete not-found & invalid UUID |
| `handlers` | `maintenance_test.go` | CreateMaintenanceLog (validasi, CCTV not found, sukses + reload), GetById, GetAll, Update/Delete not-found & invalid UUID, kepemilikan teknisi → 403 |
| `handlers` | `dashboard_test.go` | DashboardTasks (tanpa identitas 401, teknisi ter-scope, admin), DashboardSummary (distribusi status, tren 6 bulan, overdue kosong) |
| `handlers` | `audit_test.go` | GetAuditLogs (sukses, filter modul), ExportAuditLogs CSV |
| `handlers` | `upload_test.go` | tanpa file 400, tipe invalid, JPEG valid (magic bytes), path traversal 400, ekstensi salah 400, not found 404 |
| `handlers` | `report_test.go` | PDF valid prefix `%PDF`, scope teknisi |
| `handlers` | `health_test.go` | HealthCheck Connected |
| `models` | `setup_test.go`, `seeder_test.go` | UUID `BeforeCreate`, `SeedAdmin` tidak mengubah akun admin yang sudah ada |
| `config` | `database_test.go` | DSN PostgreSQL dibentuk dari environment variable |
| `routes` | `routes_test.go`, `websocket_test.go` | Route CCTV/maintenance wajib autentikasi, handshake WebSocket tanpa token ditolak |
| `handlers` | `helpers_test.go` | helper bersama: `withUserLocals`, `setupMockGORM`, `newUserMockRows`, `modelsUserRow`, `cctvMockRow`, `maintenanceMockRow` |

**Total: 28 file test** (6 utils, 4 middleware, 11 handlers, 2 models, 1 config, 3 routes, termasuk helper).

---

## 3. Gotchas Penting (Wajib Dibaca Sebelum Menambah Test)

1. **`SkipDefaultTransaction: true`** — `SetupTestDB` di `handlers/auth_test.go` memakai
   `gorm.Config{SkipDefaultTransaction: true}`. Tanpa ini GORM memanggil `BEGIN` tak terduga di
   tiap operasi tulis sehingga sqlmock gagal (`call to QueryRow 'BEGIN' was not expected`).
   Salin pola ini untuk semua handler yang menulis ke DB.

2. **Nama kolom selalu ber-quote di Postgres/GORM** — contoh: `SELECT "token_version" FROM "users"`.
   Regex sqlmock harus memakai bentuk ber-quote (`"users"`, `"token_version"`), bukan bentuk polos.

3. **Tipe file & ekstensi menipu** — upload memvalidasi magic bytes (bukan ekstensi). Test JPEG valid
   memakai byte `\xFF\xD8\xFF` dengan nama `.txt` → tetap diterima. Template xlsx diuji lewat
   `excelize.OpenReader` (file xlsx ter-zip, tidak bisa diassert byte mentah).

4. **Format keluaran yang di-assert test:**
   - CSV export audit dimulai header `ID,Waktu,Username`.
   - Template Excel: A1 `Kode CCTV`, C1 `Nama CCTV`.
   - `/cctv/scan` menerima `qr_payload` terenkripsi AES-256-GCM berisi UUID.
   - PDF dimulai prefix `%PDF`.

5. **Nama rute yang dipakai test (Fiber, tanpa prefix `/api`):** `cctv/:id`, `cctv/template`,
   `cctv/scan`, `maintenance-logs`, `auth/users`, `users/:id/reset-password`, `dashboard/tasks`,
   `dashboard/summary`, `audit-logs/export`.

6. **`APP_ENV`** — header CSP/HSTS hanya aktif saat `APP_ENV=production`; set `t.Setenv("APP_ENV", ...)`
   bila menguji variasi environment. Jangan sampai env mesin pengembang bocor ke assertion.

7. **UUID** — semua ID memakai UUID. `modelsUserRow`/`cctvMockRow`/`maintenanceMockRow` di
   `helpers_test.go` menyediakan row konstan; pakai `modelsUserRow(u, pwdHash)` dengan UUID valid.

---

## 4. Hal yang Belum Ditutup (Lanjutan Sesi Berikutnya)

Prioritas lanjutan:

1. **Cakupan lanjutan handler:** valid import/update/delete sukses dan event WebSocket dengan DB mock penuh.
2. **Routes:** smoke test jalur registrasi seluruh modul melalui bootstrap aplikasi.
3. **WebSocket:** handshake token valid dan token-version mismatch pada server WebSocket.
4. **Config migration:** test `MigrateMaintenanceToCCTV` dengan skenario legacy asset.

Referensi perencanaan: `docs/ROADMAP.md` Fase 4 (task 4.1–4.8). Catat setiap penambahan di
`backend/changelog.md` dengan format before/after + timestamp WITA.
