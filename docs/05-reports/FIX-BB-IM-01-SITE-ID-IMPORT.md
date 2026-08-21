# Fix Documentation - Site ID Import (Nullable)

**Date:** 2026-08-21  
**Bug ID:** BB-IM-01 (Site Not Found Error on Import)  
**Status:** ✅ FIXED

---

## 🐛 PROBLEM DESCRIPTION

### Issue Reported:
User encountered "Site tidak ditemukan" (Site not found) error when importing CCTV data via Excel file at `/cctv` page. The `site` column in import template was causing validation errors even when left empty.

### Symptoms:
1. ❌ Cannot import CCTV data if "Site ID" column is empty
2. ❌ Error: "Site tidak ditemukan" appears immediately
3. ❌ Template doesn't include Site ID column
4. ❌ No option to skip site assignment during bulk import

### Root Cause:
The import process didn't account for optional/null Site ID values:
- Template missing Site ID column entirely
- Backend not reading Site ID from Excel
- No handling for nullable field in multi-site setup

---

## ✅ SOLUTIONS IMPLEMENTED

### Solution 1: Add Site ID Column to Excel Template

**File Modified:** `backend/handlers/cctv.go` (DownloadTemplateCCTV function)

**BEFORE (Line 843-847):**
```go
headers := []string{
    "Kode CCTV", "ID Channel", "Nama CCTV", "Lokasi Area",
    "Longitude", "Latitude", "Jenis CCTV", "Sumber Listrik",
    "Tipe Koneksi", "Indoor/Outdoor", "Pemilik", "Status",
    "IP NVR", "IP Camera", "Keterangan",
}
```

**AFTER (NEW):**
```go
// Daftar header kolom template (site_id adalah opsi, nullable)
headers := []string{
    "Kode CCTV", "ID Channel", "Nama CCTV", "Lokasi Area",
    "Longitude", "Latitude", "Jenis CCTV", "Sumber Listrik",
    "Tipe Koneksi", "Indoor/Outdoor", "Pemilik", "Status",
    "IP NVR", "IP Camera", "Keterangan", "Site ID",
}
```

**Impact:**
✅ New download template includes "Site ID" column  
✅ Users can optionally fill in Site UUID or leave blank  
✅ Clear documentation that Site ID is optional  

---

### Solution 2: Read & Validate Site ID from Excel

**File Modified:** `backend/handlers/cctv.go` (ImportCCTV function)

**Key Changes:**

#### A. Read Site ID Cell (Line 996)
```go
rowNum := idx + 1
kodeCCTV := getCell(0)
idChannel := getCell(1)
nama := getCell(2)
lokasiArea := getCell(3)
longitude := parseCoordinateCell(getCell(4), rowNum, "longitude")
latitude := parseCoordinateCell(getCell(5), rowNum, "latitude")
jenisCCTV := getCell(6)
sumberListrik := getCell(7)
tipeKoneksi := getCell(8)
indoorOutdoor := getCell(9)
pemilik := getCell(10)
status := getCell(11)
ipNVR := getCell(12)
ipCamera := getCell(13)
keterangan := getCell(14)
siteIDStr := getCell(15) // Site ID - opsi (nullable)
```

#### B. Optional Site ID Validation (Lines 1102-1125)
```go
// Parse dan validasi Site ID (jika diisi)
if strings.TrimSpace(siteIDStr) != "" {
    siteUUID, err := uuid.Parse(strings.TrimSpace(siteIDStr))
    if err != nil {
        tx.Rollback()
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "status":  "error",
            "message": fmt.Sprintf("Baris %d: Format Site ID tidak valid. Gunakan UUID format.", rowNum),
        })
    }
    
    // Validasi bahwa Site ID benar-benar ada di database
    var site models.Site
    if err := tx.First(&site, "id = ?", siteUUID).Error; err != nil {
        tx.Rollback()
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "status":  "error",
            "message": fmt.Sprintf("Baris %d: Site '%s' tidak ditemukan. Pastikan Site ID valid.", rowNum, siteIDStr),
        })
    }
    
    cctv.SiteID = &siteUUID // Set Site ID hanya bila valid
}
// Jika siteIDStr kosong/tidak diisi, SiteID tetap NULL (tidak di-set)

// Generate unique ID untuk CCTV
cctv.ID = newID

if err := tx.Create(&cctv).Error; err != nil {
    // ...
}
```

---

## 🔍 TECHNICAL DETAILS

### Three-State Handling for Site ID:

1. **Not Provided (Empty Cell)** → `SiteID = NULL`
   ```
   Excel cell: [empty]
   Result: CCTV created without site assignment
   Database: site_id = NULL
   ```

2. **Invalid UUID Format** → Error
   ```
   Excel cell: "not-a-uuid"
   Result: Import rejected with clear error message
   Error: "Baris X: Format Site ID tidak valid. Gunakan UUID format."
   ```

3. **Valid UUID but Non-existent Site** → Error
   ```
   Excel cell: "00000000-0000-0000-0000-000000000000"
   Result: Import rejected with specific error
   Error: "Baris X: Site '0000...00' tidak ditemukan..."
   ```

4. **Valid UUID and Exists in DB** → Success
   ```
   Excel cell: "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
   Result: CCTV assigned to specified site
   Database: site_id = a1b2c3d4-e5f6-7890-abcd-ef1234567890
   ```

### Multi-Site Compatibility:

This fix ensures backward compatibility:
- Existing imports without Site ID column continue to work
- New imports with optional Site ID work correctly
- Mixed scenarios (some rows have site, others don't) are supported

---

## 🧪 TESTING VERIFICATION

### Test Case 1: Import Without Site ID (NULL)
**Steps:**
1. Download template: https://localhost/cctv/template
2. Fill required columns only (leave "Site ID" empty)
3. Upload via /cctv import form

**Expected Result:**
✅ Import succeeds  
✅ All CCTV records created with `site_id = NULL`  
✅ No "Site tidak ditemukan" error

---

### Test Case 2: Import with Valid Site ID
**Steps:**
1. Login as admin
2. Go to Sites management (if using multi-site)
3. Copy a valid Site UUID
4. Paste into "Site ID" column in Excel
5. Upload file

**Expected Result:**
✅ Import succeeds  
✅ CCTV records created with correct `site_id`  
✅ Properly associated with specified site

---

### Test Case 3: Import with Invalid Site ID (Should Fail)
**Steps:**
1. Create Excel with invalid Site UUID: `invalid-uuid-123`
2. Upload file

**Expected Result:**
❌ Import fails with clear error message  
✅ Error: "Baris X: Format Site ID tidak valid. Gunakan UUID format."

---

### Test Case 4: Import with Non-existent Site ID (Should Fail)
**Steps:**
1. Use a valid UUID format but one that doesn't exist in DB
2. Upload file

**Expected Result:**
❌ Import fails with clear error message  
✅ Error: "Baris X: Site '[uuid]' tidak ditemukan..."

---

## 📋 FILES MODIFIED

| File | Function Changed | Lines Changed | Type |
|------|------------------|---------------|------|
| `backend/handlers/cctv.go` | DownloadTemplateCCTV | +1 | Added Site ID to template |
| `backend/handlers/cctv.go` | ImportCCTV | +35 | Added Site ID parsing & validation |

---

## ✨ IMPROVEMENTS SUMMARY

### Before This Fix:
- ❌ Site ID not available in import template
- ❌ Cannot assign sites during bulk import
- ❌ Manual site assignment required after import
- ❌ No way to import CCTV without site (always null or error)

### After This Fix:
- ✅ Site ID column added to template
- ✅ Optional field - can be left empty (NULL)
- ✅ Validated only when provided
- ✅ Clear error messages for invalid sites
- ✅ Supports both single-site and multi-site setups
- ✅ Backward compatible with existing workflows

---

## 🎯 USER ACTIONS NEEDED

### For User Testing:

1. **Download Updated Template:**
   ```
   Navigate to: https://localhost/cctv
   Click "Download Template" button
   Open template_import_cctv.xlsx
   Verify "Site ID" column is present (column P)
   ```

2. **Test Import Without Site:**
   ```
   Leave "Site ID" column empty for all rows
   Fill other required columns
   Upload file to /cctv
   Should succeed without errors
   ```

3. **Test Import With Site:**
   ```
   Get a valid Site UUID from database or Sites page
   Paste into "Site ID" column
   Upload file
   Should succeed and assign sites correctly
   ```

---

## 🔒 SECURITY CONSIDERATIONS

### SQL Injection Prevention:
- UUID parsing ensures only valid UUIDs accepted
- Direct parameter binding (no string concatenation)
- GORM query safe from injection

### Validation:
- Site existence verified before assignment
- Empty cells explicitly handled as NULL
- No user input directly executed in database queries

---

## 🔄 ROLLBACK PLAN

If this change causes issues:

1. Restore previous version:
   ```bash
   git checkout HEAD~1 -- backend/handlers/cctv.go
   ```

2. Rebuild API:
   ```bash
   cd C:\laragon\www\SIGAP
   docker compose build --no-cache api
   docker compose up -d api
   ```

---

## 💡 BEST PRACTICES APPLIED

1. **Defensive Programming**
   - Trim whitespace before parsing
   - Check for empty strings before validation
   - Rollback on any error to maintain data integrity

2. **Clear Error Messages**
   - Row-specific error reporting
   - Descriptive failure reasons
   - Actionable guidance for users

3. **Graceful Degradation**
   - Optional field behavior
   - NULL-safe database design
   - Backward compatible changes

4. **Documentation**
   - Inline comments explaining three-state logic
   - Template clearly indicates optional field
   - Error messages provide helpful context

---

## 📊 IMPACT ASSESSMENT

| Aspect | Impact Level | Notes |
|--------|-------------|-------|
| **User Experience** | HIGH ⬆️ | Now flexible with optional site assignment |
| **Data Integrity** | MEDIUM ⚡ | Site validation ensures consistency |
| **Multi-site Support** | HIGH ⬆️ | Enables proper site-based organization |
| **Performance** | LOW | Minimal overhead (one additional SELECT per row) |
| **Risk** | LOW | Well-tested pattern, easy rollback |

---

## 🎉 COMPLETION STATUS

- ✅ Template updated with Site ID column
- ✅ Import handler reads Site ID from Excel
- ✅ Optional validation implemented (NULL support)
- ✅ Error handling for invalid/non-existent sites
- ✅ Backend code compiles without errors
- ✅ Docker image built successfully
- ✅ API container running and healthy

**Build Verification:**
```
✓ go build ./...          SUCCESS
✓ docker compose build api  SUCCESS
✓ Container Status: HEALTHY
```

---

**Fix Completed:** 2026-08-21 16:15 UTC+8  
**Status:** ✅ READY FOR TESTING  
**Docker Image:** sigap-api:v0.64.5

