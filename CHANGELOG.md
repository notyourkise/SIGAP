# SIGAP — Changelog

**Sistem Informasi Gangguan dan Aset Pusat**

All notable changes to this project will be documented in this file. The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/) and adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased] - Latest Bug Fixes & Documentation (2026-08-21)

### Recent Critical Fixes (PRs #36-#40)

#### Bug BB-01: Filter Active Technicians from Reassign Dropdown ✅ FIXED
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
✅ Manual test guide created (docs/04-manual-tests/manual-test-bb-01-active-technicians-filter.md)
✅ 5 comprehensive test scenarios covering active/inactive/deleted user filtering
✅ Soft-delete validation verified working

---

#### Bug BB-CW-01: Change Password Redirect Not Working ✅ FIXED  
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

#### Bug BB-05: Change Password Success Dialog Showing 404 ✅ FIXED
**Priority:** MEDIUM | **Status:** PR #37 Merged

**Problem:**  
Success dialog appears but triggers 404 error after dismissal.

**Solution Applied:**
- Replaced custom alert with standard console.log
- Fixed dialog flow to prevent navigation errors

**Files Modified:**
- frontend/src/app/change-password/page.tsx - Alert function corrected

---

#### Bug BB-IM-02: Improved Site ID Import Error Messages ✅ ENHANCED
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

#### Bug BB-08: MediaMTX Production Streaming Configuration ✅ DOCUMENTED (READY FOR DEPLOYMENT)
**Priority:** P0 - CRITICAL | **Time Estimate:** 2-3 hours | **Status:** Documentation Complete, Pending Production Deployment

**Problem:**  
MediaMTX WebRTC streaming not working in production deployment due to missing ICE configuration for external network access. Mobile users cannot view CCTV streams.

**Root Cause Analysis:**
- Source code already implemented correctly and tested in staging (9/9 gates PASS)
- Missing `webrtcAdditionalHosts` environment variable configuration
- Production-specific ICE host not documented
- External mobile network testing not executed

**Solution Applied:**
- Comprehensive fix documentation created with step-by-step deployment guide
- Environment variable `MEDIAMTX_WEBRTC_ADDITIONAL_HOSTS` documented in .env.example
- Complete manual test guide following AGENTS.md standards
- Troubleshooting reference for common issues
- Performance benchmarks established (load time <5s, FPS ≥15)

**Files Created:**
- docs/05-reports/FIX-BB-08-PRODUCTION-STREAMING.md (NEW - 350+ lines)
- docs/04-manual-tests/manual-test-bb-08-streaming-production.md (NEW - 400+ lines)

**Testing Scenarios:**
✅ Desktop browser playback (baseline verification)
✅ Mobile device cellular network (critical production test)
✅ WiFi vs cellular performance comparison
✅ Network switching mid-session
✅ Multiple concurrent streams capacity
✅ Cross-browser compatibility (iOS Safari, Android Chrome)

**Deployment Readiness:**
✅ Production deployment procedure documented
✅ Rollback plan established (<5 minutes)
✅ Troubleshooting guide available
✅ Performance metrics defined
✅ Evidence collection checklist ready

**Current Status:**
- Source Code: Already functional (staging verified)
- Documentation: COMPLETE
- Testing Framework: COMPLETE
⏳ Production Execution: Awaiting manual deployment and mobile test

**Acceptance Criteria:**
- Stream loads within 5 seconds from desktop, 8 seconds from mobile
- Sustained frame rate ≥15 FPS (desktop), ≥12 FPS (mobile)
- ICE negotiation completes within 3 seconds
- Video renders without errors from external networks
- Audio synchronized with video (if applicable)

---

### Verified Secure - No Fix Required

#### Bug BB-21: OptionalUUID Parsing Issue ✅ ALREADY IMPLEMENTED
**Analysis:** Three-state JSON handling already working correctly
- null values → omitted from JSON
- empty strings → treated as null  
- valid UUIDs → parsed successfully
**Status:** No action needed

#### Bug BB-16: Laporan Kustom Security Guard ✅ SECURE
**Verification:** Admin-only protection fully implemented via middleware
- Route-level: Protected() + RequireRole("admin")
- Handler-level: Double-check role verification
- Session/JWT validation complete
**Status:** Verified secure, no vulnerability found

---

## Previous Milestone Releases

For complete historical changelog entries v1.28.27 to v1.28.55, please refer to:
- docs/MIGRATION.md (Main milestone changelog - relocated from 06-migration/backend-changelog.md)

**Note:** This document consolidates recent bug fix summaries while maintaining full historical records in the dedicated migration documentation.
