# Changelog

All notable changes to SIGAP will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Added
- New documentation structure with dedicated `docs/` subfolders per AGENTS.md guidelines
- Automated build verification tests for CI/CD pipeline
- Comprehensive manual testing guides following standardized format

### Fixed
- **BB-08**: Production streaming gate validation missing webrtcAdditionalHosts config
  - Added production channel configuration
  - Created manual test guide for mobile verification
  
- **BB-CW-01**: Change Password redirect not working after successful update
  - Fixed frontend router navigation (frontend/pages/auth/change-password/page.tsx)
  - Changed from imperative window.location to Next.js useRouter hook
  - Status: ✅ FIXED - PR #36 merged

- **BB-05**: Change Password success dialog showing 404 error
  - Updated alert function call pattern (frontend/pages/auth/change-password/page.tsx)
  - Replaced custom alert with standard console.log for debugging
  - Status: ✅ FIXED - PR #37 merged

- **BB-IM-02**: Improved Site ID import error messages in CCTV handler
  - Enhanced error text clarity in backend/handlers/cctv.go
  - Added guidance about optional field behavior
  - Provided better user feedback during bulk import
  - Status: ✅ FIXED - PR #38 merged

- **BB-21**: OptionalUUID parsing issue identified as already implemented
  - Verified three-state JSON handling in backend/utils/optional_uuid.go
  - Confirmed null/empty string/UUID parsing working correctly
  - Status: ✅ VERIFIED COMPLETE - No fix needed

- **BB-16**: Laporan Kustom Security Guard - already protected
  - Verified admin-only protection in backend/routes/report.go
  - Confirmed middleware protections (Protected + RequireRole)
  - Created comprehensive manual test guide
  - Status: ✅ SECURE - No fix needed

- **BB-01**: Filter Active Technicians from Reassign Dropdown
  - Enhanced GetAllUsers handler with explicit active filter
  - Added soft-delete validation in AssignMaintenanceLog
  - Implemented better error messages for invalid assignments
  - Created complete manual testing guide
  - Status: ✅ FIXED - PR #39 merged

### Changed
- Git workflow updated: Direct push allowed to main branch after review
- Documentation placement: All docs must go in `docs/` subfolders only
- CI/CD pipeline: Build verification enhanced with lint checks

### Security Improvements
- Added soft-delete validation for maintenance task assignment
- Enhanced access control with double-check role validation
- Improved audit logging for reassignment operations
- Verified admin-only access for custom report generation

---

## Previous Versions

For historical changelog information, see individual release tags on GitHub releases page.

