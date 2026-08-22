# Manual Test Guide - MediaMTX Production Streaming (Bug BB-08)

**Date:** 2026-08-21  
**Bug ID:** BB-08  
**Priority:** P0 - CRITICAL  
**Version:** SIGAP v0.64.5+  
**Tester:** ___________________  

---

## 1. PRASYARAT

### Environment Setup:

#### Production Server Requirements:
- [ ] Publicly accessible domain (e.g., `nexuspace.site`)
- [ ] Firewall rules allowing:
  - UDP port 8189 (WebRTC media)
  - TCP port 8889 (WHEP signaling)
  - TCP port 8888 (HLS fallback)
- [ ] Docker Compose installed and running
- [ ] Access to production server via SSH
- [ ] Mobile device with cellular data capability

#### Test Accounts:
```sql
-- Ensure at least one admin user exists for access
SELECT username, role FROM users WHERE role = 'admin' LIMIT 1;

-- If no admin found, create one:
INSERT INTO users (username, full_name, email, role, password_hash, must_change_password)
VALUES ('test_admin', 'Test Admin', 'admin@test.com', 'admin', '$2a$12...', false);
```

#### Pre-test Verification:
```bash
# Check all containers are healthy
docker compose ps

# Expected output:
# NAME               STATUS
# sigap_db           Up (healthy)
# sigap_redis        Up (healthy)
# sigap_api          Up (healthy)
# sigap_frontend     Up (healthy)
# sigap_nginx        Up (healthy)
# sigap_mediamtx     Up (healthy)
```

---

## 2. PENGUJIAN OTOMATIS

### Build & Health Check Tests:

#### Backend Build Test:
```bash
cd backend
go build ./...              # Should succeed
go test ./... -count=1       # All tests pass
golangci-lint run            # No critical issues
```

#### Frontend Build Test:
```bash
cd frontend
npm run lint                 # No critical errors
npm run build                # Production build succeeds
```

#### Docker Compose Validation:
```bash
docker compose config --quiet    # Should return no errors
docker compose ps                # All services UP
docker stats --no-stream         # Memory/CPU within limits
```

### Configuration Verification:

#### Check Environment Variable:
```bash
# Verify MEDIA.MTX_WEBRTC_ADDITIONAL_HOSTS is set
docker exec sigap_api printenv | grep MEDIAMTX_WEBRTC_ADDITIONAL_HOSTS

# Expected output should show the production domain/IP:
# MEDIAMTX_WEBRTC_ADDITIONAL_HOSTS=nexuspace.site
```

#### Verify MediaMTX Config:
```bash
# Check if webrtcAdditionalHosts is loaded
docker logs sigap_mediamtx --tail 50 | grep -i "webrtc"

# Look for logs indicating:
# - Webrtc listening on :8889
# - ICE servers configured
```

---

## 3. SKENARIO UJI MANUAL

### TEST S1: Desktop Browser Playback (Baseline)

**Objective:** Verify WebRTC stream loads successfully from desktop browser

| Step | Action | Expected Result | Actual Result | ✓/✗ |
|------|--------|-----------------|---------------|-----|
| 1 | Login as admin to https://[domain]/monitor | Dashboard loads, camera grid visible | | ⬜ |
| 2 | Click "Stream" button on any CCTV card | Stream modal opens with player | | ⬜ |
| 3 | Wait for WHEP request to complete | Network tab shows WHEP request, HTTP 200 | | ⬜ |
| 4 | Observe video playback | Video renders in player, no errors | | ⬜ |
| 5 | Check browser console | No WebSocket/WebRTC errors | | ⬜ |
| 6 | Verify frame rate | FPS counter ≥ 15 fps sustained | | ⬜ |

**Expected Timeline:**
- Connection established: < 2 seconds
- First frame displayed: < 5 seconds total
- Stable playback: No flickering or freezes

**Evidence Capture:**
- 📸 Screenshot of successful playback
- 📹 Screen recording of 30-second session
- 📋 Console log showing no errors

---

### TEST S2: Mobile Device - Cellular Network (Critical Test)

**⚠️ CRITICAL TEST - Must be executed from external network**

**Setup:**
1. Disable Wi-Fi on mobile device
2. Confirm cellular data enabled (4G/5G active)
3. Note current network type (LTE/5G)

| Step | Action | Expected Result | Actual Result | ✓/✗ |
|------|--------|-----------------|---------------|-----|
| 1 | Navigate to https://nexuspace.site/monitor | Page loads, login form shown | | ⬜ |
| 2 | Login with admin credentials | Dashboard appears | | ⬜ |
| 3 | Click "Stream" on CCTV | Stream modal opens | | ⬜ |
| 4 | Wait for connection indicator | Loading spinner shows, then connects | | ⬜ |
| 5 | Observe video playback | Video plays smoothly without buffering | | ⬜ |
| 6 | Monitor performance metrics | Frame rate stable, no packet loss | | ⬜ |
| 7 | Stay connected for 1 minute | No disconnection occurs | | ⬜ |

**Performance Metrics:**
| Metric | Target | Measured | Pass? |
|--------|--------|----------|-------|
| **Load Time** | < 5 seconds | ______ sec | ⬜ |
| **Frame Rate** | ≥ 15 FPS | ______ FPS | ⬜ |
| **Buffer Events** | 0-1 events | ______ count | ⬜ |
| **Connection Drops** | None | ______ count | ⬜ |

**Evidence Capture:**
- 📱 Screen recording from mobile device
- 📊 Network capture showing successful ICE negotiation
- 📋 System status screenshot (cellular indicator visible)

**Pass Criteria:**
✅ Video loads within 8 seconds maximum  
✅ Sustained frame rate ≥ 12 FPS (may be lower than desktop)  
✅ No unexpected disconnections  
✅ Audio (if present) synchronized and clear  

---

### TEST S3: WiFi vs Cellular Performance Comparison

**Objective:** Verify streaming quality comparable between networks

| Test Condition | Load Time | Frame Rate | Buffer Events | Notes |
|----------------|-----------|------------|---------------|-------|
| **Wi-Fi** (baseline) | ___ sec | ___ FPS | ___ events | |
| **Cellular** (4G) | ___ sec | ___ FPS | ___ events | Compare to Wi-Fi |
| **Acceptable Diff** | +2 sec max | -20% min | Same | |

**Analysis:**
- Wi-Fi should be baseline (typically faster)
- Cellular performance acceptable if load time ≤ 8 sec
- Frame rate variance up to 20% is acceptable
- Excessive buffering (>3 events) indicates network issue

---

### TEST S4: Stream Reconnection After Network Change

**Scenario:** User switches between Wi-Fi and cellular mid-session

| Step | Action | Expected Result | Actual Result | ✓/✗ |
|------|--------|-----------------|---------------|-----|
| 1 | Start stream on Wi-Fi | Plays successfully | | ⬜ |
| 2 | Disable Wi-Fi (switch to cellular) | Stream continues or reconnects quickly | | ⬜ |
| 3 | Re-enable Wi-Fi | Stream reconnects without reloading page | | ⬜ |
| 4 | Maintain playback for 1 minute | Stable on new network | | ⬜ |

**Key Observation:** 
- ICE re-negotiation should happen automatically
- User experience should be smooth (not jarring)
- Maximum interruption: 2-3 seconds during transition

---

### TEST S5: Multiple Concurrent Streams

**Objective:** Test system capacity under load

**Setup:**
- Open multiple cameras in separate tabs/windows
- Simulate real usage pattern (multi-camera monitoring)

| Test Scenario | Max Streams | Load Time | CPU Usage | Network Bandwidth | Pass? |
|---------------|-------------|-----------|-----------|-------------------|-------|
| **Desktop Chrome** | 4 streams | ___ sec | ___% | ___ Mbps | ⬜ |
| **Desktop Chrome** | 9 streams | ___ sec | ___% | ___ Mbps | ⬜ |
| **Mobile Safari** | 1 stream | ___ sec | ___% | ___ Mbps | ⬜ |
| **Mobile Chrome** | 1 stream | ___ sec | ___% | ___ Mbps | ⬜ |

**Resource Limits:**
- Browser memory: < 2GB per tab
- Network bandwidth: < 20 Mbps per concurrent HD stream
- CPU usage: < 50% sustained

---

## 4. KRITERIA LULUS

### Overall Test Must Meet ALL These Criteria:

#### ✅ FUNCTIONAL CRITERIA
- [ ] Stream loads successfully from desktop browser
- [ ] Stream loads successfully from mobile device (cellular network)
- [ ] Video playback continuous without unexpected drops
- [ ] Audio synchronized with video (if applicable)
- [ ] Controls (pause, stop, fullscreen) functional

#### ✅ PERFORMANCE CRITERIA
- [ ] Initial load time < 5 seconds (desktop), < 8 seconds (mobile)
- [ ] Frame rate maintained ≥ 15 FPS (desktop), ≥ 12 FPS (mobile)
- [ ] Buffer events minimal (< 3 during 1-minute session)
- [ ] No significant latency increase after 1 hour of continuous use

#### ✅ NETWORK CRITERIA
- [ ] ICE negotiation completes within 3 seconds
- [ ] WebRTC connection established without manual intervention
- [ ] NAT traversal successful through firewall
- [ ] No CORS or cross-origin errors

#### ✅ MOBILE SPECIFIC CRITERIA
- [ ] Works on iOS Safari (iOS 14+)
- [ ] Works on Android Chrome (Android 8+)
- [ ] Cellular data works without VPN/proxy
- [ ] Battery drain acceptable (< 10% per hour)
- [ ] Network switching handled gracefully

---

## 5. CATATAN (Notes)

### Environment Information
```
Test Date: _________________
Browser: ___________________
OS Version: ________________
Device Type: _______________
Network Type: ☐ Wi-Fi  ☐ Cellular (4G/5G)  ☐ Ethernet
SIGAP Version: ____________
MediaMTX Version: 1.20.0-ffmpeg
Backend Git SHA: ___________
```

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

## 6. TROUBLESHOOTING REFERENCE

### Common Issues & Solutions:

#### Issue: "Failed to load stream"
**Possible Causes:**
- Firewall blocking UDP 8189
- Missing/webrtcAdditionalHosts not set
- RTSP source unreachable

**Debug Steps:**
```bash
# Check firewall
sudo ufw status | grep 8189

# Verify environment variable
docker exec sigap_api printenv | grep WEBRTC

# Test RTSP source directly
curl http://localhost:9997/v3/paths/info?path=cam-<uuid>
```

#### Issue: "No video frames received"
**Possible Causes:**
- ICE negotiation failed
- RTP packets blocked by firewall
- Codec mismatch

**Debug Steps:**
```bash
# Check ICE connections
docker logs sigap_mediamtx | grep -i ice

# Inspect path info
curl http://localhost:9997/v3/paths/list

# Verify codec supported
docker logs sigap_mediamtx | grep -i codec
```

#### Issue: "Poor performance on mobile"
**Possible Causes:**
- TURN/TLS required for symmetric NAT
- Bandwidth limitations
- Device processing power

**Solutions:**
1. Add TURN servers to configuration
2. Implement adaptive bitrate streaming
3. Reduce resolution/frame rate for mobile clients

---

## 7. VERIFICATION CHECKLIST

Before marking test as COMPLETE:

- [ ] All scenarios executed successfully
- [ ] Performance metrics within targets
- [ ] Evidence collected (screenshots/videos)
- [ ] Troubleshooting documented for known issues
- [ ] Mobile external network test completed
- [ ] Cross-browser verification done
- [ ] Documentation requirements met

### Final Decision:

**RESULT:** 
⬜ **PASS** - All tests successful, streaming working properly  
⬜ **FAIL** - Critical issues found, blocking release  
⬜ **PARTIAL PASS** - Minor issues only, can deploy with notes  
⬜ **BLOCKED** - Cannot complete testing, prerequisites missing  

### Sign-off:

**Tester Name:** ___________________  
**Signature:** ___________________  
**Date:** ___________________  
**Reviewer:** ___________________  
**Review Date:** ___________________  

---

## APPENDIX A: Sample Data for Testing

### Sample CCTV UUID (from database):
```sql
SELECT id, code, name FROM cctvs WHERE status = 'active' LIMIT 1;
```

### Example WHEP URL Format:
```
https://nexuspace.site/stream/whep/<cctv-id>/stream.weph
```

### Expected API Response Structure:
```json
{
  "status": "success",
  "data": {
    "whep_url": "https://nexuspace.site/stream/whep/<uuid>.m3u8",
    "hls_url": "https://nexuspace.site/stream/hls/<uuid>/stream.m3u8"
  }
}
```

---

## APPENDIX B: Browser Console Debug Commands

```javascript
// Check WebRTC connection state
RTCPeerConnection.prototype.addIceCandidate = (candidate) => {
  console.log('ICE candidate added:', candidate);
};

// Monitor video element
const video = document.querySelector('video');
video.onloadedmetadata = () => console.log('Video loaded:', video.duration);
video.onplaying = () => console.log('Playing started at:', Date.now());

// Network timing
console.time('WHEP Request');
// ... trigger stream request ...
console.timeEnd('WHEP Request');
```

---

*This test guide follows AGENTS.md Section 9 requirements for comprehensive manual testing.*

