package incidents

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"sigap-backend/models"
)

func TestTelegramSenderChecksAPIResultAndChat(t *testing.T) {
	for _, tc := range []struct {
		name, body    string
		status        int
		wantError     bool
		wantPermanent bool
		wantAmbiguous bool
	}{
		{"success", `{"ok":true,"result":{"message_id":42,"chat":{"id":-100}}}`, 200, false, false, false},
		{"api rejection 200 ok=false", `{"ok":false,"description":"bad"}`, 200, true, false, false},
		{"unauthorized 401 permanent", `{"ok":false,"description":"unauthorized"}`, 401, true, true, false},
		{"rate limit 429", `{"ok":false,"parameters":{"retry_after":60}}`, 429, true, false, false},
		{"wrong destination", `{"ok":true,"result":{"message_id":42,"chat":{"id":-200}}}`, 200, true, false, true},
		{"corrupt json", `{"ok":true,"result":`, 200, true, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Error("expected POST")
				}
				w.WriteHeader(tc.status)
				_, _ = fmt.Fprint(w, tc.body)
			}))
			defer server.Close()
			sender := TelegramSender{Token: "test", Client: server.Client(), baseURL: server.URL}
			id, err := sender.Send(context.Background(), "-100", "camera offline")
			if (err != nil) != tc.wantError {
				t.Fatalf("id=%d err=%v", id, err)
			}
			if err == nil && id != 42 {
				t.Fatal(id)
			}
			if err != nil {
				var de *DeliveryError
				if !errors.As(err, &de) {
					t.Fatalf("expected DeliveryError, got %T", err)
				}
				if de.Permanent != tc.wantPermanent || de.Ambiguous != tc.wantAmbiguous {
					t.Fatalf("got permanent=%v ambiguous=%v, want permanent=%v ambiguous=%v",
						de.Permanent, de.Ambiguous, tc.wantPermanent, tc.wantAmbiguous)
				}
			}
		})
	}
}

func TestTelegramDistinguishesPreSendVsPostSendAndNeverExposesToken(t *testing.T) {
	const secret = "secret-test-token-998877"

	// 1. Kegagalan pra-kirim (dial ditolak sebelum byte request dikirim) -> PreSendFailure=true, Ambiguous=false.
	closedLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	closedAddr := closedLn.Addr().String()
	_ = closedLn.Close()

	preSender := TelegramSender{
		Token:   secret,
		Client:  &http.Client{Timeout: 2 * time.Second},
		baseURL: "http://" + closedAddr,
	}
	_, preErr := preSender.Send(context.Background(), "-100", "test")
	if preErr == nil {
		t.Fatal("expected pre-send transport error")
	}
	if strings.Contains(preErr.Error(), secret) {
		t.Fatalf("error leaked secret token: %s", preErr.Error())
	}
	var preDE *DeliveryError
	if !errors.As(preErr, &preDE) || !preDE.PreSendFailure || preDE.Ambiguous {
		t.Fatalf("expected PreSendFailure=true Ambiguous=false, got %+v", preDE)
	}

	// 2. Kegagalan pasca-kirim (server membaca request HTTP lalu memutus koneksi TCP tanpa respons) -> Ambiguous=true.
	dropLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = dropLn.Close() }()
	go func() {
		conn, acceptErr := dropLn.Accept()
		if acceptErr != nil {
			return
		}
		_, _ = http.ReadRequest(bufio.NewReader(conn))
		_ = conn.Close()
	}()

	postSender := TelegramSender{
		Token:   secret,
		Client:  &http.Client{Timeout: 2 * time.Second},
		baseURL: "http://" + dropLn.Addr().String(),
	}
	_, postErr := postSender.Send(context.Background(), "-100", "test")
	if postErr == nil {
		t.Fatal("expected post-send transport error")
	}
	if strings.Contains(postErr.Error(), secret) {
		t.Fatalf("error leaked secret token: %s", postErr.Error())
	}
	var postDE *DeliveryError
	if !errors.As(postErr, &postDE) || !postDE.Ambiguous || postDE.PreSendFailure {
		t.Fatalf("expected Ambiguous=true PreSendFailure=false, got %+v", postDE)
	}
}

func TestClassifyRetryOutcomeConsistentSchedule(t *testing.T) {
	now := time.Date(2026, 11, 7, 10, 0, 0, 0, time.UTC)

	// Ambiguous retry ke-1 wajib tepat 10s, retry ke-2 tepat 30s, lalu berhenti retry otomatis.
	ambErr := &DeliveryError{Ambiguous: true, Reason: "ambiguous"}
	st1, next1, failed1 := classifyRetryOutcome(Outbox{Attempts: 1}, false, ambErr, now)
	if st1 != "ambiguous" || next1.Sub(now) != 10*time.Second || failed1 {
		t.Fatalf("ambiguous #1 mismatch: status=%s delay=%s failed=%v", st1, next1.Sub(now), failed1)
	}
	st2, next2, failed2 := classifyRetryOutcome(Outbox{Attempts: 2}, true, ambErr, now)
	if st2 != "ambiguous" || next2.Sub(now) != 30*time.Second || failed2 {
		t.Fatalf("ambiguous #2 mismatch: status=%s delay=%s failed=%v", st2, next2.Sub(now), failed2)
	}
	st3, next3, failed3 := classifyRetryOutcome(Outbox{Attempts: 3}, true, ambErr, now)
	if st3 != "ambiguous" || next3.Sub(now) < 24*time.Hour || failed3 {
		t.Fatalf("ambiguous #3 must park auto-retry: status=%s delay=%s failed=%v", st3, next3.Sub(now), failed3)
	}

	// Pre-send failure memakai backoff pending (5s pertama) dan gagal setelah MaxPendingAttempts.
	preErr := &DeliveryError{PreSendFailure: true, Reason: "dial failed"}
	stP1, nextP1, _ := classifyRetryOutcome(Outbox{Attempts: 1}, false, preErr, now)
	if stP1 != "pending" || nextP1.Sub(now) != 5*time.Second {
		t.Fatalf("pre-send #1 mismatch: status=%s delay=%s", stP1, nextP1.Sub(now))
	}
	stP6, _, markFailed := classifyRetryOutcome(Outbox{Attempts: MaxPendingAttempts}, false, preErr, now)
	if stP6 != "failed" || !markFailed {
		t.Fatalf("expected failed after max attempts, got %s markFailed=%v", stP6, markFailed)
	}
}

func TestChatRateLimiterRespects20PerMinute(t *testing.T) {
	lim := NewChatRateLimiter(MinGroupChatInterval)
	base := time.Date(2026, 11, 7, 10, 0, 0, 0, time.UTC)
	sent := 0
	for ms := 0; ms < 60_000; ms += 200 {
		ok, _ := lim.AllowAt("-100123", base.Add(time.Duration(ms)*time.Millisecond))
		if ok {
			sent++
		}
	}
	if sent >= 20 || sent < 18 {
		t.Fatalf("expected 18..19 messages per minute (<20 limit), got %d", sent)
	}
}

func TestTelegramSendWithOptionsAndBadgeFormatters(t *testing.T) {
	var captured map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &captured)
		_, _ = fmt.Fprint(w, `{"ok":true,"result":{"message_id":99,"chat":{"id":-100}}}`)
	}))
	defer server.Close()

	sender := TelegramSender{Token: "test", Client: server.Client(), baseURL: server.URL}
	id, err := sender.SendWithOptions(context.Background(), "-100", "<b>[TIKET BARU]</b>", SendOptions{
		ParseMode:        "HTML",
		ReplyToMessageID: 55,
	})
	if err != nil || id != 99 {
		t.Fatalf("expected message_id 99 without error, got id=%d err=%v", id, err)
	}
	if captured["parse_mode"] != "HTML" {
		t.Fatalf("expected parse_mode HTML, got %v", captured["parse_mode"])
	}
	if captured["reply_to_message_id"] != float64(55) || captured["allow_sending_without_reply"] != true {
		t.Fatalf("expected reply_to_message_id=55 and allow_sending_without_reply=true, got %+v", captured)
	}

	t.Setenv("CCTV_TELEGRAM_COALESCE_SECONDS", "45")
	if got := TelegramCoalesceWindow(); got != 45*time.Second {
		t.Fatalf("expected 45s coalesce window from env, got %v", got)
	}
	t.Setenv("CCTV_TELEGRAM_COALESCE_SECONDS", "invalid")
	if got := TelegramCoalesceWindow(); got != DefaultCoalesceWindow {
		t.Fatalf("expected fallback to DefaultCoalesceWindow, got %v", got)
	}

	cam := models.CCTV{
		KodeCCTV:   "CAM-01",
		Nama:       "Gerbang <Timur> & A",
		LokasiArea: "Zona A",
	}
	tech := &models.User{FullName: "Budi & Rekan"}
	msg := formatTicketNewSingleMessage("INC-20261113-0001", cam, OriginUnplannedOutage, time.Date(2026, 11, 13, 15, 0, 0, 0, time.UTC), tech)
	if !strings.HasPrefix(msg, "TIKET BARU - INC-20261113-0001\n") {
		t.Fatalf("expected plain text ticket header at top of message, got:\n%s", msg)
	}
	if !strings.Contains(msg, "Gerbang <Timur> & A") || !strings.Contains(msg, "Teknisi: Budi & Rekan (Otomatis)") {
		t.Fatalf("expected plain text camera & auto technician in message, got:\n%s", msg)
	}

	batchResolved := formatBatchText(&claimedBatch{
		Jobs: []Outbox{
			{EventType: "INCIDENT_RESOLVED", Text: "TIKET SELESAI - INC-001\nKamera: CAM-01"},
			{EventType: "INCIDENT_RESOLVED", Text: "TIKET SELESAI - INC-002\nKamera: CAM-02"},
		},
	})
	if !strings.HasPrefix(batchResolved, "2 TIKET SELESAI\n\n1) INC-001") {
		t.Fatalf("expected plain text batch resolved header, got:\n%s", batchResolved)
	}
}

func TestTelegramMessagesContainNoEmojis(t *testing.T) {
	now := time.Date(2026, 11, 14, 1, 2, 43, 0, time.UTC)
	cam := models.CCTV{KodeCCTV: "THD-11", Nama: "REST AREA PARKIRAN", LokasiArea: "Rest Area"}
	tech := &models.User{FullName: "Rifqy"}
	inc := Incident{TicketNumber: "INC-20261114-0005"}

	messages := []struct {
		name string
		text string
	}{
		{"new_single", formatTicketNewSingleMessage("INC-20261114-0005", cam, OriginUnplannedOutage, now, tech)},
		{"new_shared_nvr", formatTicketNewSharedNVRMessage("INC-20261114-0005", "10.0.0.10", []string{"THD-11", "THD-12"}, now, tech)},
		{"assigned", formatIncidentAssignedMessage(inc, "THD-11 - REST AREA PARKIRAN (Rest Area)", "Rifqy", "", "remote")},
		{"reassigned", formatIncidentAssignedMessage(inc, "THD-11 - REST AREA PARKIRAN (Rest Area)", "Andi", "Rifqy", "field")},
		{"resolved", formatIncidentResolvedMessage(inc, "THD-11 - REST AREA PARKIRAN (Rest Area)", "Pulih Setelah Penanganan")},
		{"escalated_closed", formatIncidentEscalatedClosedMessage(inc, "THD-11 - REST AREA PARKIRAN (Rest Area)", []string{"THD-11"})},
		{"admin_ready_verify", formatAdminReadyVerifyMessage(inc, "THD-11 - REST AREA PARKIRAN (Rest Area)", 1, 1)},
		{"admin_reminder", formatAdminReminderMessage(inc, "1 jam", 1)},
		{"admin_escalation", formatAdminEscalationMessage(inc, 4*time.Hour, 1)},
		{"admin_merged", formatAdminMergedMessage(inc, []string{"INC-20261114-0004"}, "Shared NVR", 2)},
		{"admin_camera_escalated", formatAdminCameraEscalatedMessage(inc, []string{"THD-11"}, 0)},
		{"admin_monitor_degraded", formatAdminMonitorDegradedMessage("NVR timeout", "koneksi terputus", 3)},
		{"admin_monitor_recovered", formatAdminMonitorRecoveredMessage("NVR timeout")},
		{"batch", formatBatchText(&claimedBatch{
			Jobs: []Outbox{
				{EventType: "CCTV_OFFLINE", Text: "TIKET BARU - INC-001\nKamera: THD-11"},
				{EventType: "CCTV_OFFLINE", Text: "TIKET BARU - INC-002\nKamera: THD-12"},
			},
		})},
	}

	for _, tc := range messages {
		if strings.TrimSpace(tc.text) == "" {
			t.Fatalf("%s returned empty message", tc.name)
		}
		for _, r := range tc.text {
			if r > 127 {
				t.Fatalf("%s contains non-ASCII character %q (%U) in:\n%s", tc.name, r, r, tc.text)
			}
		}
		if strings.Contains(tc.text, "<b>") || strings.Contains(tc.text, "</b>") {
			t.Fatalf("%s unexpectedly contains HTML bold tag in:\n%s", tc.name, tc.text)
		}
	}
}
