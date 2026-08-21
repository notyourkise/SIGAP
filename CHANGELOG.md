# SIGAP â€” Changelog

**Sistem Informasi Gangguan dan Aset Pusat**

All notable changes to this project will be documented in this file. The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/) and adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased] - Latest Bug Fixes & Documentation Consolidation (2026-08-21)

### Recent Critical Fixes (PRs #36-#39)

#### Bug BB-01: Filter Active Technicians from Reassign Dropdown âœ… FIXED
**Priority:** MEDIUM | **Time Estimate:** ~1 hour | **Status:** PR #39 Merged

**Problem:**  
Dropdown reassignment in maintenance log shows all users including inactive/deleted ones.

**Solution Applied:**
- Enhanced GetAllUsers handler with explicit soft-delete filter
- Added validation in AssignMaintenanceLog to reject deleted technicians
- Better error messages for invalid assignment attempts

**Files Modified:**
- backend/handlers/user.go - GetAllUsers function (+1 active filter)
- backend/handlers/maintenance.go - AssignMaintenanceLog function (+5 lines validation)

**Testing:**
âœ… Manual test guide created (docs/04-manual-tests/manual-test-bb-01-active-technicians-filter.md)
âœ… 5 comprehensive test scenarios covering active/inactive/deleted user filtering
âœ… Soft-delete validation verified working

---

#### Bug BB-CW-01: Change Password Redirect Not Working âœ… FIXED  
**Priority:** HIGH | **Status:** PR #36 Merged

**Problem:**  
After changing password, user stays on same page instead of redirecting to home.

**Root Cause:**  
Using window.location.href for navigation instead of Next.js router pattern.

**Solution Applied:**
```typescript
// Changed from imperative window.location to useRouter hook
const router = useRouter();
router.push('/'); // Clean redirect to home
```

**Files Modified:**
- frontend/src/app/change-password/page.tsx - Router navigation fixed

---

#### Bug BB-05: Change Password Success Dialog Showing 404 âœ… FIXED
**Priority:** MEDIUM | **Status:** PR #37 Merged

**Problem:**  
Success dialog appears but triggers 404 error after dismissal.

**Solution Applied:**
- Replaced custom alert with standard console.log
- Fixed dialog flow to prevent navigation errors

**Files Modified:**
- frontend/src/app/change-password/page.tsx - Alert function corrected

---

#### Bug BB-IM-02: Improved Site ID Import Error Messages âœ… ENHANCED
**Priority:** LOW | **Status:** PR #38 Merged

**Problem:**  
Error messages during bulk site import are unclear and don't explain optional fields.

**Solution Applied:**
- Enhanced error text clarity in backend/handlers/cctv.go
- Added guidance about optional field behavior
- Better user feedback during imports

**Files Modified:**
- backend/handlers/cctv.go - Error handling improved

---

### Verified Secure - No Fix Required

#### Bug BB-21: OptionalUUID Parsing Issue âœ… ALREADY IMPLEMENTED
**Analysis:** Three-state JSON handling already working correctly
- null values â†’ omitted from JSON
- empty strings â†’ treated as null  
- valid UUIDs â†’ parsed successfully
**Status:** No action needed

#### Bug BB-16: Laporan Kustom Security Guard âœ… SECURE
**Verification:** Admin-only protection fully implemented via middleware
- Route-level: Protected() + RequireRole("admin")
- Handler-level: Double-check role verification
- Session/JWT validation complete
**Status:** Verified secure, no vulnerability found

---

## Previous Milestone Releases

For complete historical changelog entries v1.28.27 to v1.28.55, please refer to the original source files:
- docs/06-migration/backend-changelog.md (Main milestone changelog)
- docs/05-reports/CHANGELOG-BB-08.md (CI Build Fix and Production Streaming Analysis)

---

# Complete Historical Milestone Releases

For the complete detailed milestone history (v1.28.27 to v1.28.55), please refer to the original ackend-changelog.md which is now available at: docs/06-migration/backend-changelog.md (restored for reference)

**Note:** This document consolidates recent bug fix summaries while maintaining full historical records in the dedicated backend changelog.
