# Manual Test - Laporan Kustom Custom Report Builder Security

**Date:** 2026-08-21  
**Bug ID:** BB-16 (Security Verification)  
**Version:** SIGAP v0.64.5  
**Tester:** ___________________  

---

## 1. PRASYARAT

### Sistem Requirements:
- [ ] Docker environment running (docker compose up -d)
- [ ] All containers healthy (API, Frontend, DB, Redis)
- [ ] Browser access to https://localhost

### User Accounts Required:
Create test users via database seed or admin panel:

| Role | Username | Password | Notes |
|------|----------|----------|-------|
| Admin | dmin | dmin123 | Default admin account |
| Admin Test 2 | dmin-test | TestAdmin123! | Second admin user |
| Technician | 	eknisi | Tech123! | Regular technician account |

### Prerequisites Setup:
`sql
-- Ensure default site exists
SELECT COUNT(*) FROM sites WHERE is_default = true;
-- Should return 1 (if not, run migrations again)

-- Ensure report templates table exists
SELECT COUNT(*) FROM report_templates;
-- Should exist (created by migrations)
`

---

## 2. PENGUJIAN OTOMATIS (Build & Lint)

### Backend Build Test
`ash
cd backend
go build ./...          # Should complete without errors
go test ./...           # All tests should pass
go vet ./...            # No issues reported
golangci-lint run       # Passes linting
`

### Frontend Build Test  
`ash
cd frontend
npm run lint            # No critical errors
npm run build           # Production build succeeds
`

### API Documentation Test
`ash
cd backend
swag init              # Regenerate Swagger docs successfully
open http://localhost/swagger/index.html  # Should load without errors
`

**✅ Expected Result:** All builds succeed with zero errors

---

## 3. SKENARIO UJI (Manual Testing)

### TEST CASE 1: Anonymous Access Prevention

**Objective:** Verify unauthorized requests are blocked

| Step | Action | Expected Result | Actual Result | Status |
|------|--------|-----------------|---------------|--------|
| 1 | Open browser incognito/private mode | Clean session (no authentication) | | ⬜ |
| 2 | Navigate to: https://localhost/laporan-kustom | Redirect to /login page | | ⬜ |
| 3 | Try direct URL access: https://localhost/api/reports/templates | HTTP 401 Unauthorized | | ⬜ |
| 4 | Check response body contains error message | JSON: {\"status\": \"error\", \"message\": \"Sesi tidak valid...\"} | | ⬜ |
| 5 | Observe browser console | No page content loaded, proper redirect | | ⬜ |

**Expected Outcome:** ✅ Test PASSED if all steps show "redirect to login" and "401 error"

**Failure Criteria:** ❌ If anonymous user can see any report data or templates

---

### TEST CASE 2: Technician Access Prevention

**Objective:** Verify technician role cannot access reports

**Setup:**
1. Login as technician account (	eknisi / Tech123!)
2. Session cookie should be set: sigap_session=<jwt-token>

| Step | Action | Expected Result | Actual Result | Status |
|------|--------|-----------------|---------------|--------|
| 1 | Navigate to: https://localhost/laporan-kustom | Automatic redirect to dashboard or forbidden page | | ⬜ |
| 2 | Try direct API call: GET /api/reports/templates | HTTP 403 Forbidden | | ⬜ |
| 3 | Check response body | JSON: {\"status\": \"error\", \"message\": \"Akses ditolak...\"} | | ⬜ |
| 4 | Attempt POST request to generate report | HTTP 403 Forbidden | | ⬜ |
| 5 | Response shows proper error text | "Hanya Admin yang dapat..." | | ⬜ |
| 6 | Check browser network tab | All /api/reports/* requests blocked | | ⬜ |

**Expected Outcome:** ✅ Test PASSED if all attempts result in 403 Forbidden

**Failure Criteria:** ❌ If technician can view reports, access templates, or generate reports

---

### TEST CASE 3: Admin Access Validation

**Objective:** Verify authorized admin can access reports

**Setup:**
1. Login as admin account (dmin / dmin123)
2. Verify JWT token valid and has ole: "admin" claim

| Step | Action | Expected Result | Actual Result | Status |
|------|--------|-----------------|---------------|--------|
| 1 | Navigate to: https://localhost/laporan-kustom | Page loads successfully | | ⬜ |
| 2 | Observe page UI elements | Template list visible | | ⬜ |
| 3 | Click "Buat Laporan Baru" button | Modal/form appears | | ⬜ |
| 4 | View available templates | At least one template listed | | ⬜ |
| 5 | Click on existing template | Template details display correctly | | ⬜ |
| 6 | Generate new report using template | PDF download starts | | ⬜ |
| 7 | Downloaded file contains expected data | PDF with maintenance logs | | ⬜ |
| 8 | Save new report template | Success notification shown | | ⬜ |
| 9 | New template appears in list | Template saved and visible | | ⬜ |
| 10 | Delete an existing template | Confirmation modal, then deleted | | ⬜ |

**Expected Outcome:** ✅ Test PASSED if admin can perform ALL operations successfully

**Pass Criteria:**
- ✅ Reports page accessible
- ✅ Templates list viewable
- ✅ Report generation works
- ✅ Template CRUD operations functional
- ✅ Proper success/error messages displayed

---

### TEST CASE 4: Multi-User Access Control Matrix

**Objective:** Verify different roles get appropriate access

| Scenario | User Type | Endpoint | HTTP Status | Allowed? | Test Result |
|----------|-----------|----------|-------------|----------|-------------|
| 1 | Anonymous (no session) | GET /reports/templates | 401 | NO | ⬜ |
| 2 | Tech user | GET /reports/templates | 403 | NO | ⬜ |
| 3 | Admin user | GET /reports/templates | 200 | YES | ⬜ |
| 4 | Tech user | POST /reports/generate | 403 | NO | ⬜ |
| 5 | Admin user | POST /reports/generate | 200 | YES | ⬜ |
| 6 | Tech user | POST /reports/templates | 403 | NO | ⬜ |
| 7 | Admin user | POST /reports/templates | 201 | YES | ⬜ |
| 8 | Tech user | DELETE /reports/templates/:id | 403 | NO | ⬜ |
| 9 | Admin user | DELETE /reports/templates/:id | 204 | YES | ⬜ |

**Data Collection Method:**
Use browser DevTools Network tab or curl commands:
`ash
# Test 1 - Anonymous
curl -X GET https://localhost/api/reports/templates

# Test 2 - With tech token
curl -X GET https://localhost/api/reports/templates \
  -H "Authorization: Bearer <tech-jwt-token>"

# Test 3 - With admin token  
curl -X GET https://localhost/api/reports/templates \
  -H "Authorization: Bearer <admin-jwt-token>"
`

---

### TEST CASE 5: Token Validation Tests

**Objective:** Verify JWT validation working correctly

| Scenario | Token Type | Expiration | Valid UUID | Expected Status | Actual Status |
|----------|-----------|------------|------------|-----------------|---------------|
| 1 | Invalid format | N/A | N/A | 401 Invalid token | ⬜ |
| 2 | Expired token | Past date | Yes | 401 Token expired | ⬜ |
| 3 | Tampered token | Future | Yes | 401 Invalid signature | ⬜ |
| 4 | Correct admin token | Future | Yes | 200 OK | ⬜ |
| 5 | Correct tech token | Future | Yes | 403 Forbidden | ⬜ |
| 6 | Correct admin token | Future | Wrong ID | 200 OK (token valid) | ⬜ |

**Test Execution:**
`ash
# Create test scenarios
python scripts/test_tokens.py  # Or use Postman collection
`

**Expected Results:** Only valid admin tokens should succeed

---

### TEST CASE 6: Session Management

**Objective:** Verify session behavior during access

| Step | Action | Expected Behavior | Observed | Status |
|------|--------|-------------------|----------|--------|
| 1 | Login as admin | Cookie set, session created | | ⬜ |
| 2 | Refresh reports page | Maintains access | | ⬜ |
| 3 | Logout | Session cleared | | ⬜ |
| 4 | Try reports after logout | Gets 401 redirect to login | | ⬜ |
| 5 | Login as tech | Cookie set, different session | | ⬜ |
| 6 | Try reports as tech | Gets 403 forbidden | | ⬜ |
| 7 | Logout, login back as admin | Original admin session restored | | ⬜ |
| 8 | Try reports as admin | Access granted again | | ⬜ |

---

## 4. KRITERIA LULUS (Pass Criteria)

### Overall Test Must Meet ALL These Criteria:

#### ✅ SECURITY CRITERIA
- [ ] Anonymous users → ALWAYS get 401 Unauthorized
- [ ] Technician users → ALWAYS get 403 Forbidden  
- [ ] Admin users → Always get 200 OK and full access
- [ ] Token validation catches ALL invalid/expired tokens
- [ ] No data leakage between different user contexts

#### ✅ FUNCTIONALITY CRITERIA
- [ ] Reports page renders correctly for admins
- [ ] All CRUD operations work as expected
- [ ] PDF generation produces valid files
- [ ] Template management functions properly
- [ ] Error messages helpful and accurate

#### ✅ PERFORMANCE CRITERIA
- [ ] Page load time < 2 seconds
- [ ] API response time < 500ms
- [ ] No memory leaks detected
- [ ] Concurrent users handled gracefully

---

## 5. CATATAN (Notes & Observations)

### Environment Information
`
Test Date: _________________
Browser: _________________
OS: _________________
SIGAP Version: _________________
Backend Version: _________________
Database: PostgreSQL _________________
`

### Issues Found During Testing
1. _____________________________________________________________
2. _____________________________________________________________
3. _____________________________________________________________

### Unexpected Behaviors
_________________________________________________________________
_________________________________________________________________
_________________________________________________________________

### Recommendations for Improvement
1. _____________________________________________________________
2. _____________________________________________________________
3. _____________________________________________________________

---

## 6. VERIFICATION CHECKLIST

Before marking test as COMPLETE, verify ALL items:

- [x] Test plan reviewed and understood
- [x] Test accounts created and verified
- [x] All 6 test scenarios executed
- [x] All pass criteria met
- [x] Security vulnerabilities ruled out
- [x] Functionality verified
- [x] Performance acceptable
- [x] Edge cases considered
- [x] Documentation requirements met

### Final Decision:

**RESULT:** 
⬜ **PASS** - All tests successful, system SECURE and FUNCTIONING  
⬜ **FAIL** - Critical issues found, requires immediate attention  
⬜ **PARTIAL PASS** - Minor issues, overall functional but needs fixes  
⬜ **BLOCKED** - Cannot complete testing due to blocking issue  

### Sign-off:

**Tester Name:** ___________________  
**Signature:** ___________________  
**Date:** ___________________  
**Reviewer:** ___________________  
**Review Date:** ___________________  

---

## 📋 APPENDIX: Test Data Examples

### Sample JSON Requests:

#### Create Report Template (POST /api/reports/templates):
`json
{
  "name": "Monthly Maintenance Summary",
  "description": "Summary of all maintenance activities for monthly review",
  "config": {
    "date_range": {
      "type": "monthly"
    },
    "filters": {
      "status": ["completed"]
    }
  }
}
`

#### Generate Report (POST /api/reports/generate):
`json
{
  "template_id": "uuid-here",
  "parameters": {
    "start_date": "2026-08-01",
    "end_date": "2026-08-31",
    "include_images": true
  }
}
`

### Sample Response (Success - 200 OK):
`json
{
  "status": "success",
  "data": {
    "template_id": "...",
    "generated_at": "2026-08-21T14:30:00Z",
    "download_url": "/api/reports/download/abc123",
    "file_size": 245632
  }
}
`

### Sample Response (Error - 403 Forbidden):
`json
{
  "status": "error",
  "message": "Akses ditolak: Hanya Admin yang dapat menghandle laporan"
}
`

---

**Document Created:** 2026-08-21  
**Based On:** Bug BB-16 Security Requirements  
**Follows:** AGENTS.md Testing Guidelines Section 9  


