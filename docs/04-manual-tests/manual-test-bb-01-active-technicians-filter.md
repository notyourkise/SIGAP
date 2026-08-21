# Manual Test - Active Technicians Filter (Bug BB-01)

**Date:** 2026-08-21  
**Bug ID:** BB-01  
**Version:** SIGAP v0.64.5  
**Tester:** ___________________  

---

## 1. PRASYARAT

### Sistem Requirements:
- [ ] Docker environment running (docker compose up -d)
- [ ] All containers healthy
- [ ] Browser access to https://localhost

### Test Data Setup:

#### Create Test Users:
`sql
-- Admin user (should NOT appear in tech dropdown)
INSERT INTO users (id, username, full_name, email, role, must_change_password, password_hash)
VALUES ('admin-test-id', 'admintest', 'Admin Test', 'admin@test.com', 'admin', false, '...');

-- Active technicians (SHOULD appear in dropdown)
INSERT INTO users (id, username, full_name, email, role, must_change_password, password_hash)
VALUES 
('tech-active-1', 'techactive1', 'Active Tech 1', 'tech1@test.com', 'teknisi', false, '...'),
('tech-active-2', 'techactive2', 'Active Tech 2', 'tech2@test.com', 'teknisi', false, '...');

-- Soft-deleted technician (should NOT appear)
INSERT INTO users (id, username, full_name, email, role, must_change_password, deleted_at, password_hash)
VALUES ('tech-deleted-1', 'techdeleted1', 'Deleted Tech', 'deleted@test.com', 'teknisi', false, '2026-08-21', '...');
`

**Verification Queries:**
`sql
-- Count active technicians
SELECT COUNT(*) FROM users WHERE role='teknisi' AND deleted_at IS NULL;
-- Should return 2

-- Check if soft-deleted exists
SELECT * FROM users WHERE role='teknisi' AND deleted_at IS NOT NULL;
-- Should return 1 row
`

---

## 2. PENGUJIAN OTOMATIS

### Backend Build Test
`ash
cd backend
go build ./...          # Should succeed
go test ./...           # All tests pass
golangci-lint run       # No issues
`

### Frontend Build Test
`ash
cd frontend
npm run lint            # No critical errors
npm run build           # Production build succeeds
`

---

## 3. SKENARIO UJI MANUAL

### TEST CASE 1: GET /api/users?role=teknisi Response

**Objective:** Verify only active technicians returned

| Step | Action | Expected Result | Actual Result | ✓/✗ |
|------|--------|-----------------|---------------|-----|
| 1 | Login as admin | Session cookie set | | ⬜ |
| 2 | Execute: GET https://localhost/api/users?role=teknisi | HTTP 200 OK | | ⬜ |
| 3 | Parse JSON response | Valid JSON structure | | ⬜ |
| 4 | Check data.rows.length | Should be 2 (active only) | | ⬜ |
| 5 | Verify each user.role | All equal "teknisi" | | ⬜ |
| 6 | Check no deleted_at field | All users have null deleted_at | | ⬜ |
| 7 | Verify usernames match | "techactive1", "techactive2" only | | ⬜ |

**Expected Response Structure:**
`json
{
  "status": "success",
  "data": {
    "rows": [
      {
        "id": "...",
        "username": "techactive1",
        "full_name": "Active Tech 1",
        "email": "tech1@test.com",
        "role": "teknisi"
      },
      {
        "id": "...",
        "username": "techactive2",
        "full_name": "Active Tech 2",
        "email": "tech2@test.com",
        "role": "teknisi"
      }
    ],
    "pagination": {
      "page": 1,
      "limit": 10,
      "total_rows": 2,
      "total_pages": 1
    }
  }
}
`

**Pass Criteria:** 
- ✅ Returns exactly 2 technicians
- ✅ Neither is the soft-deleted one
- ✅ Pagination present and correct
- ✅ No admin users in list

---

### TEST CASE 2: Assignment to Active Technician

**Setup:**
- Create maintenance log with status "ongoing"
- Note down UUID of active technician (e.g., "techactive1")

| Step | Action | Expected Result | Actual Result | ✓/✗ |
|------|--------|-----------------|---------------|-----|
| 1 | GET task details | Shows current assignment | | ⬜ |
| 2 | POST reassign to ACTIVE tech | Success response | | ⬜ |
| 3 | Response contains status "success" | JSON shows success | | ⬜ |
| 4 | Task updated in database | New technician assigned | | ⬜ |
| 5 | AssignAt timestamp set | Timestamp present | | ⬜ |
| 6 | Status changed to "ongoing" | Status updated | | ⬜ |
| 7 | Notification sent | Admin notified of reassign | | ⬜ |

**Request Payload:**
`json
{
  "technician_id": "tech-active-1-uuid-here"
}
`

**Expected Success Response:**
`json
{
  "status": "success",
  "data": {
    "id": "task-uuid",
    "technician_id": "tech-active-1-uuid-here",
    "assigned_at": "2026-08-21T14:30:00Z",
    "status": "ongoing"
  }
}
`

---

### TEST CASE 3: Assignment to Deleted Technician (Should FAIL)

**Setup:**
- Use soft-deleted technician UUID ("tech-deleted-1")

| Step | Action | Expected Result | Actual Result | ✓/✗ |
|------|--------|-----------------|---------------|-----|
| 1 | POST reassign with DELETED tech UUID | HTTP 400 Bad Request | | ⬜ |
| 2 | Response contains error message | Proper error text | | ⬜ |
| 3 | Error mentions non-active/removed | Specific error shown | | ⬜ |
| 4 | Task NOT reassigned | Database unchanged | | ⬜ |
| 5 | Old technician still assigned | Assignment preserved | | ⬜ |

**Request Payload:**
`json
{
  "technician_id": "tech-deleted-1-uuid-here"
}
`

**Expected Error Response:**
`json
{
  "status": "error",
  "message": "Teknisi tersebut sudah dinonaktifkan/dihapus"
}
`

**Fail Criteria:** ❌ If task gets reassigned successfully

---

### TEST CASE 4: Assignment to Admin User (Should FAIL)

**Setup:**
- Use admin user UUID ("admin-test-id")

| Step | Action | Expected Result | Actual Result | ✓/✗ |
|------|--------|-----------------|---------------|-----|
| 1 | POST reassign with ADMIN UUID | HTTP 400 Bad Request | | ⬜ |
| 2 | Error message indicates wrong role | Clear role error | | ⬜ |
| 3 | Task NOT reassigned | Database unchanged | | ⬜ |
| 4 | Admin not assigned to task | Assignment remains original | | ⬜ |

**Request Payload:**
`json
{
  "technician_id": "admin-test-id-uuid-here"
}
`

**Expected Error Response:**
`json
{
  "status": "error",
  "message": "User tujuan bukan berperan teknisi"
}
`

**Fail Criteria:** ❌ If admin user can be assigned to maintenance task

---

### TEST CASE 5: Frontend Dropdown Verification

**Objective:** Verify UI dropdown shows only active technicians

**Note:** Requires Intan's approval for frontend changes verification

| Step | Action | Expected Result | Actual Result | ✓/✗ |
|------|--------|-----------------|---------------|-----|
| 1 | Navigate to /maintenance page | Page loads | | ⬜ |
| 2 | Click "Alihkan" on ongoing task | Modal opens | | ⬜ |
| 3 | View technician dropdown | Only active techs listed | | ⬜ |
| 4 | Verify count matches backend | Same count as API | | ⬜ |
| 5 | Try to select deleted tech | Not visible in list | | ⬜ |
| 6 | Try to select admin | Not visible in list | | ⬜ |
| 7 | Reassign to visible tech | Success notification | | ⬜ |

**Dropdown Content Should Show:**
- ✅ "Active Tech 1" (techactive1)
- ✅ "Active Tech 2" (techactive2)
- ❌ "Deleted Tech" (NOT present)
- ❌ "Admin Test" (NOT present)

---

## 4. KRITERIA LULUS

### Overall Test Must Meet ALL These Criteria:

#### ✅ FUNCTIONALITY CRITERIA
- [ ] Active technicians appear in list
- [ ] Deleted technicians excluded from list
- [ ] Admin users excluded from list
- [ ] Successful assignment to active tech works
- [ ] Failed assignment prevention working correctly

#### ✅ VALIDATION CRITERIA  
- [ ] Soft-delete check prevents assignment
- [ ] Role validation prevents admin assignment
- [ ] Proper error messages displayed
- [ ] Database integrity maintained

#### ✅ PERFORMANCE CRITERIA
- [ ] API response time < 500ms
- [ ] Page load time < 2 seconds
- [ ] No memory leaks or slow queries

---

## 5. CATATAN (Notes)

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

### Recommendations for Improvement
1. _____________________________________________________________
2. _____________________________________________________________

---

## 6. VERIFICATION CHECKLIST

Before marking test as COMPLETE:

- [ ] Test plan reviewed
- [ ] Test accounts created
- [ ] All scenarios executed
- [ ] Pass criteria met
- [ ] Validation working correctly
- [ ] Error messages clear
- [ ] Documentation requirements met

### Final Decision:

**RESULT:** 
⬜ **PASS** - All tests successful, filter working properly  
⬜ **FAIL** - Critical issues found  
⬜ **PARTIAL PASS** - Minor issues only  
⬜ **BLOCKED** - Cannot complete testing  

### Sign-off:

**Tester Name:** ___________________  
**Signature:** ___________________  
**Date:** ___________________  
**Reviewer:** ___________________  
**Review Date:** ___________________  


