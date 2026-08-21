# 📊 Final Testing Guide - Consolidated Bug Fixes

**PR Branch:** `bugfix/bb-13-dynamic-checklist-status`  
**Date:** 2026-08-21  
**Tester:** ___________________  
**Test Status:** ⬜ Pending / ✅ Pass / ❌ Fail  

---

## 🎯 Quick Start

### System Requirements
```bash
Docker: Running version 20+
Git: Version 2.30+
Browser: Chrome/Firefox/Edge (latest)
```

### Prerequisites
- [ ] Docker containers running (`docker ps`)
- [ ] All services healthy (9 containers)
- [ ] Database accessible
- [ ] Fresh browser session ready

---

## 🧪 Test Suite A: Dynamic Checklist (BB-13)

### Pre-test Setup
1. **Create Custom Template** (Admin access required):
   ```
   URL: https://localhost/schedules
   Action: Create new maintenance schedule
   Template: Edit to have FEWER than 9 items (e.g., 5 items)
   Frequency: Weekly
   Save
   ```

2. **Generate Task**:
   ```
   Wait for automatic generation OR manually create task
   Verify status: scheduled → ongoing
   ```

### Test Steps

| # | Action | Expected Result | Actual Result | ✓/✗ |
|---|--------|-----------------|---------------|-----|
| 1 | Login as assigned technician | Redirected to `/maintenance` page | | |
| 2 | Click "Selesaikan" on active task | Modal opens showing 5 checklist items | | |
| 3 | Complete ALL items with "Normal / OK" | Each item accepts selection, no errors | | |
| 4 | Upload at least 1 photo | File uploads successfully | | |
| 5 | Click "Selesaikan Tugas" | Loading spinner appears briefly | | |
| 6 | Observe result notification | Message: "Tugas pemeliharaan... berhasil diselesaikan" | | |
| 7 | Check task list | Status = `completed` (NOT failed) | | |
| 8 | Check CCTV status | Status = `aktif` (if was maintenance) | | |

### Variants (Optional Testing)

#### Variant A: Mixed Results
- [ ] Fill 3 items as "OK", 2 items as "Bermasalah"
- **Expected**: Status = `failed`, CCTV → `pemeliharaan`

#### Variant B: Different Count
- [ ] Try with 12-item template instead of 5
- **Expected**: Works same way, completes if all OK

---

## 🧪 Test Suite B: Change Password Flow (BB-CW-01)

### Pre-test Setup
1. **Use Fresh Admin Account** (or reset one):
   ```
   Username: admin
   Password: admin123 (default)
   MustChangePassword: true (system forces change)
   ```

2. **Open Incognito/Private Browser Window**
   - Prevents cached session interference

### Test Steps

| # | Action | Expected Result | Actual Result | ✓/✗ |
|---|--------|-----------------|---------------|-----|
| 1 | Navigate to `https://localhost/login` | Login page loads correctly | | |
| 2 | Enter credentials: admin/admin123 | Submit succeeds | | |
| 3 | Observe after authentication | Redirects to `/change-password` | | |
| 4 | Monitor page behavior | NO infinite refresh loop | | |
| 5 | Wait 5 seconds | Page remains stable, form loaded | | |

#### Form Validation Tests

| # | Input Values | Click Button | Expected Error | Actual Error | ✓/✗ |
|---|-------------|--------------|----------------|--------------|-----|
| 6 | Empty both fields | Change Password | "Mohon isi semua field" | | |
| 7 | Password: `abc`, Confirm: `abc123` | Change Password | "Passwords do not match" | | |
| 8 | Password: `short`, Confirm: `short` | Change Password | "at least 8 characters" | | |
| 9 | Password: `ValidPass123!`, Confirm: `ValidPass123!` | Change Password | Success + redirect | | |
| 10 | At new login with NEW password | Login | Session created successfully | | |

### Post-Test Verification
- [ ] Original password `admin123` NO LONGER works
- [ ] New password used in step 9 IS accepted
- [ ] All other sessions invalidated (force logout)
- [ ] CSRF token regenerated in cookies

---

## 🧪 Test Suite C: Excel Import with Nullable Site ID (BB-IM-01)

### Pre-test Setup
1. **Prepare Test Data**:
   ```
   Create Excel file (template_import_cctv.xlsx):
   - Column A (Kode CCTV): CCTV001, CCTV002, CCTV003
   - Column B (ID Channel): CH-001, CH-002, CH-003
   - Column C (Nama CCTV): Camera Lobby, Camera Parking, Camera Gate
   - Column D (Lokasi Area): Building A, Building B, Building C
   - Remaining columns: Fill as needed
   - Column P (Site ID): LEAVE EMPTY for all rows
   ```

2. **Save as `.xlsx` format** (not .csv or .xls)

### Test Steps

#### Part 1: Template Verification
| # | Action | Expected Result | Actual Result | ✓/✗ |
|---|--------|-----------------|---------------|-----|
| 1 | Open `https://localhost/cctv` | Master data page loads | | |
| 2 | Click "Download Template" button | File downloads | | |
| 3 | Open downloaded Excel file | 16 columns visible | | |
| 4 | Check column headers | Column P shows "Site ID" | | |

#### Part 2: Import WITHOUT Site ID
| # | Action | Expected Result | Actual Result | ✓/✗ |
|---|--------|-----------------|---------------|-----|
| 5 | In browser, click "Impor Massal Data" | Form modal opens | | |
| 6 | Upload prepared Excel file (empty Site IDs) | Upload starts immediately | | |
| 7 | Wait for processing | Progress indicator shown | | |
| 8 | Observe completion message | "Berhasil mengimpor 3 data CCTV" | | |
| 9 | Check CCTV table | 3 new records visible | | |
| 10 | Query database (optional):<br>`SELECT id, kode_cctv, site_id FROM cctvs WHERE kode_cctv IN ('CCTV001','CCTV002','CCTV003');` | `site_id = NULL` for all | | |

#### Part 3: Import WITH Valid Site ID
1. **Get valid Site UUID from database**:
   ```sql
   SELECT id FROM sites LIMIT 1;
   -- Example result: a1b2c3d4-e5f6-7890-abcd-ef1234567890
   ```

2. **Update Excel file**:
   - Paste this UUID into Column P for first record only
   - Leave empty for remaining records
   - Save as `.xlsx`

3. **Import again**:
   | # | Action | Expected Result | Actual Result | ✓/✗ |
   |---|--------|-----------------|---------------|-----|
   | 11 | Upload new Excel file | Upload starts | | |
   | 12 | Process completes | Success message shown | | |
   | 13 | Check first record | `site_id = a1b2c3d4...` (assigned) | | |
   | 14 | Check other records | `site_id = NULL` (unassigned) | | |

#### Part 4: Error Handling Tests

**Test 4A: Invalid UUID Format**
| # | Action | Expected Result | Actual Result | ✓/✗ |
|---|--------|-----------------|---------------|-----|
| 15 | Create Excel with: `invalid-uuid-format` in Column P | Upload attempts | | |
| 16 | Observe error message | "Baris X: Format Site ID tidak valid..." | | |
| 17 | Check records imported | NONE imported (rollback) | | |

**Test 4B: Non-existent Site UUID**
| # | Action | Expected Result | Actual Result | ✓/✗ |
|---|--------|-----------------|---------------|-----|
| 18 | Create Excel with: `00000000-0000-0000-0000-000000000000` | Upload attempts | | |
| 19 | Observe error message | "Baris X: Site '...' tidak ditemukan..." | | |
| 20 | Check records imported | NONE imported (rollback) | | |

---

## 📝 Issue Log

Document any problems encountered during testing:

### Critical Issues (Blocker)
1. _________________________________________________________________
2. _________________________________________________________________

### Minor Issues (Non-blocker)
1. _________________________________________________________________
2. _________________________________________________________________

### Edge Cases Not Tested
1. _________________________________________________________________
2. _________________________________________________________________

---

## ✅ Test Sign-off

| Checklist Item | Status |
|---------------|--------|
| BB-13: Dynamic checklist works correctly | ⬜ PASS / ⬜ FAIL |
| BB-CW-01: No infinite loop in change password | ⬜ PASS / ⬜ FAIL |
| BB-IM-01: Nullable Site ID import works | ⬜ PASS / ⬜ FAIL |
| No regressions in existing features | ⬜ PASS / ⬜ FAIL |
| All error messages clear and accurate | ⬜ PASS / ⬜ FAIL |
| Database integrity maintained | ⬜ PASS / ⬜ FAIL |
| Backward compatibility verified | ⬜ PASS / ⬜ FAIL |

### Overall Assessment
**Result:** ⬜ APPROVED FOR MERGE / ⬜ REQUIRES FIXES / ⬜ NOT READY

### Tester Comments:
```
_________________________________________________
_________________________________________________
_________________________________________________
_________________________________________________
```

---

**Test Started:** ___________  
**Test Completed:** ___________  
**Duration:** ___________ hours  
**Tester Name:** ___________  
**Signature:** ___________  

---

## 🚀 Next Steps After Testing

### If All Tests Pass:
1. [ ] Approve PR in GitHub
2. [ ] Request review from another team member
3. [ ] Merge to main branch (admin only)
4. [ ] Deploy to production environment
5. [ ] Notify users about improvements

### If Tests Fail:
1. [ ] Document specific failure points
2. [ ] Provide feedback to developer/AI agent
3. [ ] Request fixes and retesting
4. [ ] Repeat until all tests pass

---

**Documentation Version:** 1.0  
**Last Updated:** 2026-08-21  

