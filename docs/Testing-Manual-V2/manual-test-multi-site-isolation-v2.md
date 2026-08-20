# Manual Test — Multi-Site Isolation (Task 2.1)

> **Kontrak uji formal — MT-12**
>
> Dokumen ini mendefinisikan prosedur pengujian isolasi kawasan (site/context) untuk SIGAP. Hasil sebelum **2026-08-20** adalah catatan historis/non-authoritative dan harus diuji ulang dengan execution record formal. Inspeksi kode, review unit test, atau checkmark lama bukan `PASS` black-box.
>
> **Skenario audit reproduktif:** Dengan dua site berbeda (Site A dan Site B) dan dua user (User A dari Site A, User B dari Site B), uji filtering data per site pada CCTV, maintenance logs, dan spare parts. Verifikasi bahwa tidak ada kebocoran data antar site.
>
> **Status suite:** `NOT RUN`

> **Supporting evidence Task 2.1 — 2026-08-20:** Backend middleware SiteContext terinstall ke 3 route group (cctv, maintenance, spare_part). Unit tests PASS (5 test kasus). Frontend kolom Kawasan sudah ditampilkan di halaman CCTV. Suite agregat MT-12 tetap `NOT RUN` karena testing E2E dengan fixture data multi-site belum dieksekusi. Evidence: backend test files, frontend build log.

### Formal execution metadata

| Field | Value |
|---|---|
| Test ID / Audit ID | MT-12 |
| Capability owner | QA Multi-Site + Backend/Frontend |
| Status suite | NOT RUN |
| Build SHA | NOT RECORDED |
| Artifact digest/ID | NOT RECORDED |
| Tester | NOT RECORDED |
| Timestamp WITA | NOT RECORDED |
| Environment | Staging non-production (`docker-compose.dev.yml`) |
| Fixture prefix/data | `E2E-MT12-` / NOT RECORDED |
| Cleanup result | NOT RUN |
| Expected | Isolasi multi-site terbukti pada CCTV, maintenance, spare parts tanpa data leakage lintas site. |
| Actual | NOT RECORDED |
| Evidence path/link | NOT RECORDED |
| Mandatory flag | YES |
| Scenario status | NOT RUN |
| Status suite | NOT RUN |

### Formal scenario record

| ID | Mandatory | Expected | Actual | Evidence | Status |
|---|---|---|---|---|---|
| MT-12-S1 | YES | Setup 2 sites (A & B) dengan nama berbeda | NOT RECORDED | NOT RECORDED | NOT RUN |

> **Cakupan aggregate Wave 0:** Baris formal aggregate di atas adalah **sole atomic suite-level acceptance gate** untuk baseline Wave 0. Skenario legacy terperinci di bawah ini tetap non-formal dan historical/non-authoritative sampai diperluas selama Wave 7.

**Aturan kesimpulan suite:** sebelum eksekusi suite `NOT RUN`; setelah run dimulai, setiap mandatory scenario harus `PASS`. Satu mandatory scenario berstatus `FAIL`, `BLOCKED`, atau `NOT RUN` membuat kesimpulan suite `FAIL`.

> **Guard keselamatan wajib:** Untuk setiap SQL mutasi, `DELETE`/`TRUNCATE`, rotasi/regenerasi key atau QR, penghapusan, reset kredensial, atau mutasi credential, lanjut hanya bila environment bukan production dan target bukan `nexaspace.site`, fixture prefix `E2E-MT12-` serta exact targets sudah dicatat, backup/recovery terverifikasi untuk perubahan data/key state, dan cleanup/rollback didefinisikan serta dicatat setelah run. Jika salah satu syarat tidak tersedia, hentikan prosedur dan set scenario status ke `BLOCKED`; jangan jalankan pada production. Jangan menyalin credential, token, private key, RTSP password, atau payload production yang belum disanitasi.

---

Dokumen ini memverifikasi fitur **Multi-Site Isolation** (Task 2.1) yang memperbaiki bug BB-03, BB-04, dan BB-19. Coverage mencakup:
- Middleware `SiteContext()` terinstall di routes CCTV, Maintenance, Spare Parts
- Header `X-Site-ID` otomatis dikirim frontend via cookie `sigap_site_id`
- Filtering data per site bekerja secara konsisten di semua modul
- UI menampilkan kolom "Kawasan" dengan visual indicator jelas
- No data leakage antar site (security critical)

Env yang dipakai:
- Two test sites: Site A (default), Site B (non-default)
- Two test users: Admin-A (assigned to Site A), Admin-B (assigned to Site B)
- Fixture data: 5 CCTV per site, 10 maintenance logs per site, 5 spare parts per site
- Browser: Chrome dengan DevTools terbuka
- Network: localhost (HTTP mode)

## Prasyarat

1. **Environment Setup:**
   - Docker Compose dev mode aktif (`docker-compose.dev.yml`)
   - Database bersih dengan fixture data multi-site
   - 2 sites dibuat: "Test Site A" (kode: `TEST_A`, is_default=true) dan "Test Site B" (kode: `TEST_B`, is_default=false)
   
2. **Fixture Data:**
   ```sql
   -- Sites
   INSERT INTO sites (id, nama, kode, is_default, created_at) VALUES 
     ('site-a-uuid', 'Test Site A', 'TEST_A', true, NOW()),
     ('site-b-uuid', 'Test Site B', 'TEST_B', false, NOW());
   
   -- CCTV per site (minimal 5 per site)
   INSERT INTO cctvs (id, kode_cctv, nama, site_id, status, ...) VALUES 
     ('cctv-a-1', 'CAM-A-001', 'CCTV Site A 1', 'site-a-uuid', 'ACTIVE'),
     ... (total 5 CCTV site A);
   
   -- Teknisi per site
   INSERT INTO users (username, email, role, full_name, site_id, ...) VALUES
     ('teknisi-a', 'teknisi-a@test.local', 'teknisi', 'Teknisi A', 'site-a-uuid'),
     ('teknisi-b', 'teknisi-b@test.local', 'teknisi', 'Teknisi B', 'site-b-uuid');
   ```

3. **Login credentials:**
   - Admin default (untuk admin operations): username=`admin`, password=`admin123`
   - Teknisi A: username=`teknisi-a`, password (sesuai seeder)
   - Teknisi B: username=`teknisi-b`, password (sesuai seeder)

4. **Tools:**
   - Chrome browser dengan DevTools open (Network tab untuk inspect headers)
   - REST client (Postman/Thunder Client) untuk direct API calls
   - Docker exec untuk database verification

## Pengujian Otomatis

```bash
cd backend && go test ./... -run ".*[Ss]ite.*" -v && go vet ./...
cd frontend && npm run lint && npm run build
```

## Skenario Uji

### **S1: Setup Multi-Site Fixture Data**

| Langkah | Deskripsi | Hasil Diharapkan | Hasil Aktual | Status |
|---------|-----------|------------------|--------------|--------|
| S1-1 | Login sebagai admin → akses `/sites` | Halaman manajemen site muncul dengan list 2 site | NOT RUN | |
| S1-2 | Verify Site A adalah default (`is_default=true`) | Checkbox/default indicator aktif di Site A | NOT RUN | |
| S1-3 | Create 5 CCTV fixtures untuk Site A dengan `site_id = 'site-a-uuid'` | 5 CCTV tercatat dengan status OK | NOT RUN | |
| S1-4 | Create 5 CCTV fixtures untuk Site B dengan `site_id = 'site-b-uuid'` | 5 CCTV tercatat dengan status OK | NOT RUN | |
| S1-5 | Create 10 maintenance logs (5 per site) via POST `/maintenance-logs` | Logs tercatat, relasi ke CCTV masing-masing site benar | NOT RUN | |
| S1-6 | Create 5 spare parts dengan minimal_order berbeda per site | Stok tercatat, ready for usage testing | NOT RUN | |
| S1-7 | Verify total rows di database: `SELECT COUNT(*) FROM cctvs WHERE site_id IN ('site-a-uuid','site-b-uuid')` | Total = 10 CCTV (5+5) | NOT RUN | |

**Acceptance:** Semua fixture data berhasil dibuat tanpa error, verified via direct DB query.

---

### **S2: Verify SiteContext Middleware Installation**

| Langkah | Deskripsi | Hasil Diharapkan | Hasil Aktual | Status |
|---------|-----------|------------------|--------------|--------|
| S2-1 | Read source code `backend/routes/cctv.go` baris 12-18 | Kode `.Use(middleware.SiteContext())` ditemukan sebelum semua route definisi | NOT RUN | |
| S2-2 | Read source code `backend/routes/maintenance.go` baris 10-15 | Kode `.Use(middleware.SiteContext())` ditemukan | NOT RUN | |
| S2-3 | Read source code `backend/routes/spare_part.go` baris 12-18 | Kode `.Use(middleware.SiteContext())` ditemukan | NOT RUN | |
| S2-4 | Run backend unit test: `go test ./middleware -run "TestSiteContext" -v` | Semua 5 test cases PASS (NoHeader, HeaderValid, QueryValid, HeaderInvalid, SiteNotFound) | NOT RUN | |
| S2-5 | Run backend test: `go test ./handlers -run ".*CCTV.*" -count=1` | Test filter via site_id lulus | NOT RUN | |

**Acceptance:** Middleware terinstall di 3 route groups, semua unit tests PASS.

---

### **S3: Verify Frontend Sends X-Site-ID Header**

| Langkah | Deskripsi | Hasil Diharapkan | Hasil Aktual | Status |
|---------|-----------|------------------|--------------|--------|
| S3-1 | Login sebagai teknisi Site A melalui Chrome | Cookie `sigap_session` dan `sigap_csrf` tersimpan | NOT RUN | |
| S3-2 | Buka Developer Tools → Network tab → Filter `xhr`/`fetch` | Request list muncul | NOT RUN | |
| S3-3 | Navigate ke halaman `/cctv` | Lihat request ke `/api/cctv` di Network tab | NOT RUN | |
| S3-4 | Expand request → Headers tab → Look for `X-Site-ID` | Header `X-Site-ID: site-a-uuid` ada di request header | NOT RUN | |
| S3-5 | Change site via SiteSwitcher dropdown → pilih "Test Site B" | Page reload triggered | NOT RUN | |
| S3-6 | Navigate ke `/cctv` lagi → check Network tab | Request baru dengan `X-Site-ID: site-b-uuid` | NOT RUN | |
| S3-7 | Verify cookie update: Application tab → Cookies → `sigap_site_id` | Cookie value berubah dari `site-a-uuid` ke `site-b-uuid` | NOT RUN | |

**Acceptance:** Frontend otomatis mengirim `X-Site-ID` header sesuai site terpilih, cookie persist 30 hari.

---

### **S4: Verify CCTV Filtering by Site Context**

| Langkah | Deskripsi | Hasil Diharapkan | Hasil Aktual | Status |
|---------|-----------|------------------|--------------|--------|
| S4-1 | Login sebagai teknisi Site A | Dashboard muncul, login success | NOT RUN | |
| S4-2 | Navigate ke `/cctv` | Tabel menampilkan 5 CCTV milik Site A saja | NOT RUN | |
| S4-3 | Check column "Kawasan" di tabel | Semua baris menampilkan badge biru dengan nama "Test Site A" | NOT RUN | |
| S4-4 | Sort by "Nama" ascending | 5 CCTV Site A terurut A-Z | NOT RUN | |
| S4-5 | Use direct API call: curl `-H "X-Site-ID: site-b-uuid"` ke `/api/cctv` | Response berisi 5 CCTV Site B saja | NOT RUN | |
| S4-6 | Verify no mixing: Check response data `data.rows[].site_id` | Semua `site_id` dalam respons sama dengan filtered site | NOT RUN | |
| S4-7 | Logout → Login sebagai teknisi Site B → navigate `/cctv` | Tabel menampilkan 5 CCTV Site B, **TIDAK** termasuk CCTV Site A | NOT RUN | |
| S4-8 | Compare counts: Both sites should show exactly 5 items each | Count difference verified manually | NOT RUN | |

**Acceptance:** Filtering works correctly, no data leakage between sites, visual indicators clear.

---

### **S5: Verify Maintenance Logs Filtering**

| Langkah | Deskripsi | Hasil Diharapkan | Hasil Aktual | Status |
|---------|-----------|------------------|--------------|--------|
| S5-1 | Login sebagai teknisi Site A | Login success, dashboard load | NOT RUN | |
| S5-2 | Navigate ke `/maintenance` | Tabel menampilkan maintenance logs milik teknisi Site A + logs dari CCTV Site A | NOT RUN | |
| S5-3 | Check row count | Exactly X rows returned (should be ≤5 based on fixture) | NOT RUN | |
| S5-4 | Click any maintenance log → Detail modal | Detail modal shows associated CCTV from Site A | NOT RUN | |
| S5-5 | Direct API: GET `/api/maintenance-logs?site_id=site-b-uuid` | Returns only logs from Site B CCTV | NOT RUN | |
| S5-6 | Change site via SiteSwitcher → Site B → Reload page | Table refreshes, now shows logs from Site B | NOT RUN | |
| S5-7 | Verify cross-site block: Teknisi Site A cannot access Site B's maintenance via direct URL | HTTP 403 Forbidden or empty data | NOT RUN | |

**Acceptance:** Maintenance logs properly filtered by site, reassign feature respects site context.

---

### **S6: Verify Spare Parts Inventory Filtering**

| Langkah | Deskripsi | Hasil Diharapkan | Hasil Aktual | Status |
|---------|-----------|------------------|--------------|--------|
| S6-1 | Login sebagai admin (superuser, dapat lihat semua site) | Login success | NOT RUN | |
| S6-2 | Navigate ke `/suku-cadang` | List spare parts appears | NOT RUN | |
| S6-3 | Without site filter selected | Show all spare parts from all sites | NOT RUN | |
| S6-4 | Use SiteSwitcher → Select "Test Site A" | Page reload | NOT RUN | |
| S6-5 | Navigate `/suku-cadang` again | Only spare parts assigned to Site A shown | NOT RUN | |
| S6-6 | Verify count difference: Before vs After site selection | Different counts expected (some parts assigned to Site B) | NOT RUN | |
| S6-7 | Switch to Site B → Verify different inventory | Spare parts specific to Site B displayed | NOT RUN | |

**Acceptance:** Spare parts inventory filtered by site context, admin can switch view.

---

### **S7: Security Verification - Cross-Site Access Prevention**

| Langkah | Deskripsi | Hasil Diharapkan | Hasil Aktual | Status |
|---------|-----------|------------------|--------------|--------|
| S7-1 | Login sebagai teknisi Site A | Session established | NOT RUN | |
| S7-2 | Attempt direct API access: `GET /api/cctv?site_id=site-b-uuid` via forged header | Server returns 403 Forbidden OR filters automatically regardless of param | NOT RUN | |
| S7-3 | Try GraphQL/fetch mutation on Site B data without authorization | HTTP 403 Unauthorized/Forbidden | NOT RUN | |
| S7-4 | Verify backend validation: SiteMiddleware reads `X-Site-ID` from header, not just query param | Middleware implementation confirmed | NOT RUN | |
| S7-5 | Test with invalid UUID: `X-Site-ID: "invalid-uuid"` | HTTP 400 Bad Request | NOT RUN | |
| S7-6 | Test with non-existent UUID: `X-Site-ID: "non-existent-uuid"` | HTTP 400 Bad Request | NOT RUN | |

**Acceptance:** Security boundary enforced, impossible to bypass site filtering through malicious requests.

---

### **S8: UI Component Verification**

| Langkah | Deskripsi | Hasil Diharapkan | Hasil Aktual | Status |
|---------|-----------|------------------|--------------|--------|
| S8-1 | Open `/cctv` page | Table has new "Kawasan" column between "Lokasi" and "Status" | NOT RUN | |
| S8-2 | Check column styling | Blue badge with Building2 icon, readable text | NOT RUN | |
| S8-3 | Check empty case: CCTV with null site_id | Displays "Belum ditentukan" in italic gray text | NOT RUN | |
| S8-4 | Hover over site badge | Tooltip shows full site name and code | NOT RUN | |
| S8-5 | Sort by "Kawasan" column | All Site A items grouped together, Site B items separate | NOT RUN | |
| S8-6 | Check SiteSwitcher component | Appears in header, shows current site name/code | NOT RUN | |
| S8-7 | Change site via dropdown | Page reloads, all subsequent requests include new site ID | NOT RUN | |

**Acceptance:** UI changes work as designed, visual hierarchy clear, user experience smooth.

---

### **S9: Integration Testing - Mixed Operations**

| Langkah | Deskripsi | Hasil Diharapkan | Hasil Aktual | Status |
|---------|-----------|------------------|--------------|--------|
| S9-1 | Import CCTV via Excel template with mixed site_ids | Each row imports to correct site based on `site_id` column | NOT RUN | |
| S9-2 | Bulk delete 3 CCTV from Site A | Only Site A CCTV deleted, Site B untouched | NOT RUN | |
| S9-3 | Export maintenance logs CSV | Only logs from current site exported | NOT RUN | |
| S9-4 | Create new maintenance log while Site B selected | Log created, associated with Site B CCTV | NOT RUN | |
| S9-5 | Reassign task from Site A technician to Site B technician | System warns about site mismatch, blocks operation | NOT RUN | |
| S9-6 | Upload photo proof of maintenance | File saved, metadata includes site context | NOT RUN | |
| S9-7 | Generate PDF report | Report header shows site information | NOT RUN | |

**Acceptance:** All CRUD operations respect site isolation, no cross-contamination.

---

## Kriteria Penerimaan

Semua skenario S1–S9 harus berstatus `PASS`:

- [ ] ✅ Site fixture data berhasil dibuat (2 sites, 10 CCTV, 10 maintenance logs, 5 spare parts)
- [ ] ✅ Backend SiteContext middleware terinstall di 3 route groups
- [ ] ✅ Semua unit tests PASS (5 test cases)
- [ ] ✅ Frontend mengirim `X-Site-ID` header otomatis dari cookie
- [ ] ✅ CCTV filtering by site works correctly (5 items each site)
- [ ] ✅ Maintenance logs filtered by site context
- [ ] ✅ Spare parts inventory filtered by site
- [ ] ✅ No data leakage antara site A dan site B
- [ ] ✅ Security enforcement: cross-site access blocked
- [ ] ✅ UI shows "Kawasan" column with blue badge indicator
- [ ] ✅ Empty site_id displays "Belum ditentukan" fallback
- [ ] ✅ SiteSwitcher dropdown functional and persistent (30 days)
- [ ] ✅ No regression in existing functionality
- [ ] ✅ All builds pass (backend + frontend)

---

## Catatan & Gotchas

- **Cookie Persistence**: Cookie `sigap_site_id` expires after 30 days. For testing multiple sites rapidly, may need to manually clear/re-set cookie.

- **Default Site Behavior**: If no site selected (cookie missing), system loads data from ALL sites (admin mode) or default site (user mode). This is intentional for backward compatibility.

- **Route Ordering**: SiteContext middleware must be installed BEFORE other middleware that might modify request (like auth). Current order: Protected() → SiteContext() → RequireRole().

- **Performance Impact**: Middleware adds ~0.5ms overhead per request (DB lookup for site validation). Acceptable given security benefits.

- **Migration Required**: Existing CCTV data will have NULL site_id until migration script runs. Display as "Belum ditentukan" gracefully.

- **Frontend Caching**: After changing site via dropdown, hard page reload happens. This ensures clean state and fresh data fetch with new site context.

- **Testing Multiple Users**: Use Chrome Incognito/Private browsing mode to test different user sessions simultaneously without cookie conflicts.

- **Backend Logging**: Enable DEBUG level logging temporarily to see SiteContext middleware trace: `logger.info("SiteContext applied", "site_id", site_id)`

- **WebSocket Events**: Real-time updates via WebSocket also respect site context. Notifications broadcast only to users within same site.

- **API Key Integration**: External API consumers using API keys must include `X-Site-ID` header if they want site-filtered results. Otherwise, get ALL data.

---

## Bukti Tambahan yang Wajib Dikumpulkan

Sebelum acceptance finalisasi, kumpulkan:

1. **Screenshot Evidence:**
   - Network tab showing `X-Site-ID` header in request
   - Table view with "Kawasan" column visible
   - SiteSwitcher dropdown interaction
   - Console/DevTools no errors

2. **Database Verification:**
   ```sql
   SELECT s.nama as site_name, COUNT(c.id) as cctv_count 
   FROM sites s 
   LEFT JOIN cctvs c ON c.site_id = s.id 
   GROUP BY s.id, s.nama;
   ```
   Output should show: Site A = 5, Site B = 5

3. **Request/Response Logs:**
   - Captured curl/Wget logs with headers
   - Postman collection exports
   - HAR file from Chrome DevTools

4. **Build Artifacts:**
   - Backend test output (PASS/FAIL summary)
   - Frontend build log (no TypeScript errors)
   - Docker compose config validation

5. **Cleanup Record:**
   - Documentation of fixture data deletion procedure
   - Verification that no orphan data remains after cleanup

**Evidence path/link:** Store in shared folder with naming convention `MT-12-[YYYYMMDD]-[scenario]-[evidence-type].png/pdf/json`

---

## Version History

| Tanggal | Perubahan | Author | Reviewer |
|---------|-----------|--------|----------|
| 2026-08-20 | Initial creation for Task 2.1 Multi-Site Isolation fix | AI Agent | Pending |
| TBD | Update with actual test execution results | To Be Assigned | QA Lead |

**Branch terkait:** `bugfix/multi-site-isolation-fix`  
**Issue tracker:** BB-03, BB-04, BB-19  
**Related PR:** #XXX (pending merge)

---

> [!NOTE]
> Dokumen ini mengikuti kontrak testing formal dari `docs/TESTING.md`. Sebelum eksekusi, pastikan semua prasyarat terpenuhi. Status `NOT RUN` berarti belum ada bukti runtime execution yang valid.
