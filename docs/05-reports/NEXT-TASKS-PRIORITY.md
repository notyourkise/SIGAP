# 🎯 PRIORITAS TASK BERIKUTNYA - SIGAP Bug Remediation

**Tanggal:** 2026-08-21  
**Resources:** 1 developer  
**Status:** Phase 1 - Foundation Hardening (In Progress)  

---

## ✅ WHAT WE JUST COMPLETED (Multi-Site Isolation)

### Critical Finding: **BACKEND ALREADY COMPLETE!**

After deep code review,我发现 Multi-Site Isolation feature sudah **fully implemented**:

| Component | Status | Notes |
|-----------|--------|-------|
| ✅ Migration (000018_multi_site) | Complete | Creates default OIKN site + FK columns |
| ✅ SiteContext Middleware | Complete | Validates UUID + verifies site existence |
| ✅ Route Protection | Complete | Applied to CCTV, Maintenance, Spare Parts |
| ✅ Handler Filtering | Complete | WHERE site_id = ? OR site_id IS NULL |
| ✅ Frontend API Integration | Complete | Auto-sends X-Site-ID header from cookie |
| ✅ RBAC Guards | Complete | Admin-only for write operations |

**Files Already Correct:**
- `backend/middleware/site.go` - Full implementation
- `backend/routes/cctv.go` - Lines 17-21 (SiteContext applied)
- `backend/routes/maintenance.go` - Lines 14-17 (SiteContext applied)
- `backend/handlers/cctv.go` - Lines filtering by site_id
- `backend/handlers/maintenance.go` - JOIN with site filter
- `frontend/src/lib/api.ts` - Lines 224-227 (X-Site-ID injection)

---

## 🚨 IMMEDIATE NEXT STEP: E2E VERIFICATION TESTING

**Priority:** CRITICAL - Blocker for release

### Why Testing is Required NOW:
The backend logic is correct, but we need to verify **NO DATA LEAKAGE** in production scenario:

```
┌─────────────────────────────────────────────┐
│ SCENARIO: Two-site isolation verification   │
├─────────────────────────────────────────────┤
│ Step 1: Create Site A & Site B              │
│ Step 2: Assign Tech A → Site A              │
│ Step 3: Assign Tech B → Site B              │
│ Step 4: Login as Tech A, access /cctv       │
│ Expected: ONLY see Site A CCTV              │
│ FAIL CONDITION: See Site B data (security breach!) │
└─────────────────────────────────────────────┘
```

### Testing Plan (Today):

**Option A: Quick Manual Test (Recommended)**
```bash
# 1. Start Docker environment
cd C:\laragon\www\SIGAP
docker compose up -d

# 2. Access https://localhost/login
# 3. Create test sites manually via UI
# 4. Verify site switching works correctly
# 5. Confirm no cross-site data leakage
```

**Option B: Automated Integration Tests**
```bash
cd backend
go test ./middleware -run TestSiteContext -v
go test ./handlers -run "TestGetAllCCTV.*site" -v
```

**Time Estimate:** 2-3 hours total

---

## 📋 SUBSEQUENT PRIORITY TASKS

### Task Group 1: Security Hardening (Post-E2E Testing)

| Priority | Bug ID | Task | Effort | Dependencies |
|----------|--------|------|--------|--------------|
| 1️⃣ | BB-16 | Add component guard to `/laporan-kustom` | 1 hr | None |
| 2️⃣ | BB-20 | Split spare parts API (admin vs tech options) | 2 hrs | Site context verified |
| 3️⃣ | Enhancement | Add user assignment check to SiteContext | 3 hrs | Based on E2E results |

**Total Effort:** ~6 hours (1 day)

### Task Group 2: Maintenance Integrity Fixes

| Priority | Bug ID | Task | Effort | Dependencies |
|----------|--------|------|--------|--------------|
| 1️⃣ | BB-21 | Fix DTO parsing (OptionalUUID type) | 2 hrs | None |
| 2️⃣ | BB-13 | Implement dynamic checklist logic | 3 hrs | Template handler exists |
| 3️⃣ | BB-01 | Filter active techs from reassign dropdown | 1 hr | maintenance.go updated |

**Total Effort:** ~6 hours (1 day)

### Task Group 3: Notification System

| Priority | Bug ID | Task | Effort | Dependencies |
|----------|--------|------|--------|--------------|
| 1️⃣ | Generate VAPID keys | Create public/private key pair | 0.5 hr | None |
| 2️⃣ | BB-11 | Add `/api/push/public-key` endpoint | 1.5 hr | VAPID keys available |
| 3️⃣ | BB-15 | Integrate NotifyUserSync in schedule runner | 2 hrs | None |
| 4️⃣ | BB-22 | Add zone assignment notifications | 1.5 hr | Zone handler updated |

**Total Effort:** ~5.5 hours (1 day)

---

## 📊 SPRINT TIMELINE ESTIMATE

Assuming sequential execution with 1 developer:

| Day | Workstream | Deliverables | Verification |
|-----|------------|--------------|--------------|
| **Day 1** | Multi-Site E2E Testing | No data leakage confirmed | Browser tests |
| **Day 2** | Security Hardening | BB-16, BB-20 fixed + role checks | Unit tests + manual |
| **Day 3** | Maintenance Fixes | BB-21, BB-13, BB-01 resolved | Form submission tests |
| **Day 4** | Notification System | Push config + event pipeline | Notification received tests |
| **Day 5** | Documentation + Cleanup | Changelog, test cases updated | Review |

**Total:** 5 working days for all P1 bugs

---

## ❓ DECISION POINTS NEEDED

### 1. Testing Approach Preference:
- [ ] **Option A:** Manual browser testing first (faster start, immediate feedback)
- [ ] **Option B:** Write unit/integration tests first (more thorough, slower)

**Recommendation:** Start with Option A today, then create automated tests tomorrow.

### 2. VAPID Key Generation:
Generate VAPID keys now or defer?

**Command if proceeding:**
```bash
npm install -g webpush-cli
webpush generate-vapid-keys --contactEmail admin@sigap.local
```
Output will be public/private key pair. Need to add to `.env`.

### 3. Rollback Plan Prepared?
If critical regression found during testing:

```bash
# Database rollback script ready?
# Backend service restart procedure documented?
# Frontend cache cleared steps?
```

---

## 🎯 TODAY'S DELIVERABLE GOAL

**Primary Objective:** Complete E2E verification testing for Multi-Site Isolation

**Success Criteria:**
- ✅ Can create two separate sites via admin UI
- ✅ Assign users to different sites successfully  
- ✅ Tech A login → only sees Site A data
- ✅ Tech B login → only sees Site B data
- ✅ Attempting cross-site access returns error or filtered data
- ✅ Global data (site_id=NULL) visible to all

**Time Box:** 3 hours maximum

**Next Steps After Success:** Begin Task Group 1 (Security Hardening)

---

## 📝 ACTION ITEMS FOR YOU

1. **Confirm:** You agree to proceed with E2E testing today?
2. **Provide:** Any specific test scenarios you want verified?
3. **Decide:** Should I generate VAPID keys now or wait?
4. **Access:** Do you have access to run Docker commands locally?

Once you confirm, I'll:
1. Run the E2E tests systematically
2. Document findings
3. Proceed to next priority tasks immediately

---

**Current Status:** ⭐⭐⭐⭐⭐ **READY TO TEST**  
**Blocker:** None - everything is coded and deployed to localhost  
**Next Milestone:** E2E Verification Completion (Target: End of Today)
