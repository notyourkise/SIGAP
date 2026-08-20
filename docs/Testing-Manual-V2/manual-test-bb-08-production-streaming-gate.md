# Manual Test — Production Streaming Gate (BB-08)

## Informasi Umum

| Field | Value |
|-------|-------|
| **Issue** | BB-08: MediaMTX streaming production gate |
| **Branch** | fix/bb-08-media-streaming-production-gate |
| **Version** | 0.64.5 (production verification) |
| **Timestamp** | 2026-08-20 21:00 UTC+8 |
| **Tester** | QA Team / User Acceptance |
| **Environment** | Production (`https://nexaspace.site`) |

---

## Deskripsi Masalah

**Status Saat Ini**: 
- ✅ Source/staging environment tests: PASS (9 gates verified)
- ❌ Production/public/mobile tests: NOT RUN (blocking release)

**Root Cause Analysis**:
1. Missing `webrtcAdditionalHosts` in production configuration
2. No evidence of WebRTC connection from external networks
3. No TURN/TLS fallback testing completed

**Critical Questions to Answer**:
1. Can users access streams from outside corporate network?
2. What happens if UDP port 8189 is blocked by firewall?
3. Does WebRTC successfully establish connections via mobile/cellular data?

---

## Prasyarat

1. **Production Access**: URL `https://nexaspace.site` accessible
2. **Test Accounts**: 
   - Admin account with full permissions
   - Teknisi account with limited permissions
3. **Network Preparation**: 
   - Mobile device with cellular data active (Wi-Fi OFF)
   - External network available (not VPN)
   - Firewall allows outbound UDP/HTTPS
4. **NVR/Kamera Status**: At least one camera online and configured for streaming

---

## Skenario Pengujian

### S1: Login & Monitor Page Access (Internal Network)

**Step-by-Step:**

1. Navigate to `https://nexaspace.site` on desktop browser
2. Login with admin credentials
3. Click "Monitor CCTV" di sidebar
4. Wait for grid view to load

**Expected Result:**
- ✅ Redirect `/login` → authentication succeeds
- ✅ Dashboard loads without error
- ✅ Monitor page renders grid layout
- ✅ All registered CCTV appear with status indicators

**Actual Result:** [Tulis hasil nyata]

**Evidence Collection:**
- [ ] Screenshot dashboard loading sequence
- [ ] Screenshot monitor grid
- [ ] Browser console log (no errors)

**Status:** [ ] PASS / [ ] FAIL / [ ] BLOCKED

---

### S2: Live Stream Test via Browser (Internal Network)

**Step-by-Step:**

1. Dari Monitor page, click salah satu kamera aktif
2. Klik "Start Stream" tombol
3. Tunggu video player initialize
4. Verify frame playback berjalan

**Expected Result:**
- ✅ WHEP signaling returns valid SDP answer
- ✅ Peer connection state changes: connecting → connected
- ✅ Video element `readyState = 4` (have enough data)
- ✅ Frame plays at expected resolution (640x360 atau lebih)
- ✅ Badge "🟢 LIVE" appears after decoded frame
- ✅ No JavaScript errors in console

**Actual Result:** [Tulis hasil nyata]

**Evidence Collection:**
- [ ] Console logs showing peer connection states
- [ ] Video properties inspection (width/height/currentTime)
- [ ] Network tab showing WHEP request/response
- [ ] Screenshot of playing video with live badge

**Status:** [ ] PASS / [ ] FAIL / [ ] BLOCKED

---

### S3: External Network Test via Cellular Data (Mobile Device)

**Step-by-Step:**

1. Use smartphone with Wi-Fi turned OFF (cellular only)
2. Open mobile browser, navigate to `https://nexaspace.site`
3. Login with same admin credentials
4. Click Monitor → Select camera → Start Stream

**Expected Result:**
- ✅ Authentication works on mobile network
- ✅ Grid view loads properly on mobile
- ✅ Stream starts successfully
- ✅ ICE connection established via UDP or fallback TCP
- ✅ Video plays smoothly on mobile screen
- ✅ No timeout errors (>30 seconds)

**Actual Result:** [Tulis hasil nyata]

**Evidence Collection:**
- [ ] Screenshot login success on cellular
- [ ] Screenshot video playing on mobile
- [ ] Network type indicator (4G/LTE/5G visible)
- [ ] Performance metrics (buffer underruns, rebuffering events)

**Status:** [ ] PASS / [ ] FAIL / [ ] BLOCKED

**Notes:** If this fails, likely due to missing `webrtcAdditionalHosts` config or blocked UDP ports.

---

### S4: UDP Blocked Fallback Test (TURN/TLS Required)

**Step-by-Step:**

1. Configure local firewall to block UDP outbound on port 8189
2. Attempt stream playback on same device
3. Observe fallback behavior

**Expected Result:**
- ✅ WebRTC falls back to TCP connection automatically
- ⚠️ OR requires explicit TURN server configuration if both UDP/TCP blocked
- ✅ Stream continues playing with potentially higher latency

**Actual Result:** [Tulis hasil nyata]

**Evidence Collection:**
- [ ] Network analyzer capturing connection attempts
- [ ] Console logs showing UDP failure and TCP fallback
- [ ] Latency comparison (UDP vs TCP fallback)

**Status:** [ ] PASS / [ ] FAIL / [ ] BLOCKED

**If Failed:** Need to configure `turnServers` in MediateMX config with TURN server credentials.

---

### S5: Complete App Flow Verification

**Step-by-Step:**

1. From scratch login as teknisi (non-admin)
2. Check restricted permissions (can't see all cameras)
3. Try accessing restricted features
4. Start stream from permitted camera

**Expected Result:**
- ✅ Role-based filtering works correctly
- ✅ Restricted cameras not shown to teknisi
- ✅ Allowed cameras display correctly
- ✅ Streams respect authorization boundaries

**Actual Result:** [Tulis hasil nyata]

**Evidence Collection:**
- [ ] Screenshot technician view
- [ ] Comparison of admin vs teknisi views
- [ ] API call logs showing filtered queries

**Status:** [ ] PASS / [ ] FAIL / [ ] BLOCKED

---

### S6: Concurrent Stream Load Test

**Step-by-Step:**

1. Open multiple browser tabs simultaneously
2. In each tab, start different camera streams
3. Monitor system performance

**Expected Result:**
- ✅ Each stream plays independently
- ✅ MediaMTX handles multiple concurrent viewers
- ✅ No significant memory leak or degradation
- ✅ CPU usage remains reasonable (<80%)

**Actual Result:** [Tulis hasil nyata]

**Evidence Collection:**
- [ ] Performance metrics (memory/CPU before/after)
- [ ] Number of concurrent viewers supported
- [ ] Quality degradation threshold observation

**Status:** [ ] PASS / [ ] FAIL / [ ] BLOCKED

---

## Kriteria Lulus

Suite dianggap **PASS** bila semua mandatory scenarios achieved:

✅ **Mandatory Gates (ALL required):**
1. [ ] S1: Login & internal network access
2. [ ] S2: Live stream internal network successful
3. [ ] S3: Live stream external/cellular network successful ← **CRITICAL FOR RELEASE**
4. [ ] S4: UDP fallback or TURN configured documented
5. [ ] S5: Authorization boundaries enforced
6. [ ] S6: Concurrent load handling acceptable

**Overall Suite Status:** [ ] PASS / [ ] FAIL / [ ] BLOCKED

---

## Fix Implementation Checklist

Jika tests fail, implement fixes dalam prioritas ini:

### Priority 1: Critical Configuration Fix
- [ ] Update production MediateMX config with `webrtcAdditionalHosts`
- [ ] Include public IP address or domain reachable externally
- [ ] Restart MediateMX service
- [ ] Verify configuration applied via `GET /api/health` or YAML check

### Priority 2: TURN Server Setup (if needed)
- [ ] Choose TURN provider (coturn, Xirsys, etc.)
- [ ] Generate credentials for TURN authentication
- [ ] Add `turnServers` array to MediateMX config
- [ ] Test connectivity with TURN enabled

### Priority 3: Release Engineering Fix
- [ ] Implement reproducible Docker build process
- [ ] Tag images with commit SHA instead of mutable tags
- [ ] Add OCI labels for provenance tracking
- [ ] Document deployment procedure with version control

---

## Evidence Requirements

Setiap scenario yang PASS wajib menyertakan:

1. **Screenshot Evidence**: Visual proof of passing tests
2. **Console Log Export**: Browser developer tools captured output
3. **Network Trace**: Request/Response details for streaming endpoints
4. **Performance Metrics**: Memory/CPU/network utilization data
5. **Timestamp WITA**: Time-stamped execution record
6. **Environment Details**: Device type, network type, browser version

---

## Troubleshooting Guide

### Issue: UDP Port Blocked
**Symptoms**: WebRTC connects slowly or fails entirely  
**Solution**: Enable TURN fallback or reconfigure firewall rules  
**Verification**: Check ICE candidate lists in browser DevTools  

### Issue: Certificate Errors
**Symptoms**: Mixed content warnings, HTTPS errors  
**Solution**: Ensure valid SSL certificate on domain  
**Verification**: Run SSL Labs test on `nexaspace.site`  

### Issue: High Latency
**Symptoms**: Video lags significantly behind real-time  
**Solution**: Reduce resolution/bitrate, optimize codec parameters  
**Verification**: Compare frame timestamps vs wall clock time  

### Issue: Audio Issues (if applicable)
**Symptoms**: No audio, synchronized poorly with video  
**Solution**: Check audio codec compatibility, adjust buffering  
**Verification**: Use media inspector to verify audio track present  

---

## References

- **Original Bug**: BLACKBOX-FINDINGS-2026-08-19.md - BB-08
- **Staging Evidence**: TASK-1B-STREAMING-RUNTIME-EVIDENCE-2026-08-20.md
- **Production Provenance**: PRODUCTION-ARTIFACT-PROVENANCE-2026-08-20.md
- **Analysis Summary**: BB-08-ANALYSIS-2026-08-20.md
- **MediaMTX Config**: `mediamtx.yml` current development version

---

_Ditulis: 2026-08-20 21:00 UTC+8_  
_Diperbarui: Saat eksekusi manual_
