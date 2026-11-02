package incidents

import (
	"testing"
	"time"
)

func TestRecoveryRequiresFreshEvidenceForFiveMinutes(t *testing.T) {
	start := time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)
	m := Member{ConnectionState: "offline"}
	for _, sec := range []int{0, 120, 240} {
		if Advance(&m, "live", start.Add(time.Duration(sec)*time.Second)) {
			t.Fatal("recovery confirmed before five minutes")
		}
	}
	if !Advance(&m, "live", start.Add(300*time.Second)) {
		t.Fatal("fresh final evidence should confirm recovery")
	}
	if m.ConnectionState != "recovered" {
		t.Fatal(m.ConnectionState)
	}
}

func TestUnknownInterruptsRecoveryWithoutInventingOffline(t *testing.T) {
	start := time.Now().UTC()
	m := Member{ConnectionState: "offline"}
	Advance(&m, "live", start)
	Advance(&m, "unknown", start.Add(time.Minute))
	if m.ConnectionState != "unknown" || m.RecoveryStartedAt != nil {
		t.Fatal("unknown must invalidate observation without claiming offline")
	}
	if Advance(&m, "live", start.Add(6*time.Minute)) {
		t.Fatal("elapsed time without evidence is not recovery")
	}
}

func TestFlappingAndStaleObservationsCannotConfirmRecovery(t *testing.T) {
	start := time.Now().UTC()
	m := Member{ConnectionState: "offline"}
	Advance(&m, "live", start)
	Advance(&m, "live", start.Add(120*time.Second))
	Advance(&m, "offline", start.Add(180*time.Second))
	Advance(&m, "live", start.Add(240*time.Second))
	if Advance(&m, "live", start.Add(300*time.Second)) {
		t.Fatal("flapping must restart the recovery window")
	}
	Advance(&m, "offline", start.Add(100*time.Second))
	if m.ConnectionState != "recovering" {
		t.Fatal("out of order observations must be ignored")
	}
}

func TestGapAndRepeatedRecovery(t *testing.T) {
	start := time.Now().UTC()
	m := Member{ConnectionState: "offline"}
	Advance(&m, "live", start)
	if Advance(&m, "live", start.Add(301*time.Second)) {
		t.Fatal("gap must restart observation")
	}
	for _, sec := range []int{421, 541, 601} {
		Advance(&m, "live", start.Add(time.Duration(sec)*time.Second))
	}
	if m.ConnectionState != "recovered" {
		t.Fatal("expected recovery")
	}
	if Advance(&m, "live", start.Add(610*time.Second)) {
		t.Fatal("repeated live must not repeat recovery notification")
	}
	Advance(&m, "offline", start.Add(620*time.Second))
	if m.ConnectionState != "offline" || !m.IsActiveMember {
		t.Fatal("offline recurrence must retain active membership")
	}
}
