package incidents

import (
	"testing"
	"time"
)

func TestRecoveryRequiresSixFreshSamplesOverFiveMinutes(t *testing.T) {
	start := time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)
	m := Member{ConnectionState: StateOffline}
	for _, sec := range []int{0, 60, 120, 180, 240} {
		if Advance(&m, "live", start.Add(time.Duration(sec)*time.Second)) {
			t.Fatalf("recovery confirmed prematurely at %ds", sec)
		}
	}
	if !Advance(&m, "live", start.Add(300*time.Second)) {
		t.Fatal("6th fresh sample at 300s should confirm recovery")
	}
	if m.ConnectionState != StateRecovered || m.RecoveredAt == nil {
		t.Fatalf("expected recovered with timestamp, got %+v", m)
	}

	// Hanya 5 sampel selama 300 detik (misal tiap 75s) belum mencapai MinRecoverySamples=6.
	mSparse := Member{ConnectionState: StateOffline}
	for _, sec := range []int{0, 75, 150, 225, 300} {
		if Advance(&mSparse, "live", start.Add(time.Duration(sec)*time.Second)) {
			t.Fatalf("5 samples must not confirm recovery at %ds", sec)
		}
	}
	if mSparse.ConnectionState != StateRecovering {
		t.Fatalf("expected recovering with 5 samples, got %s", mSparse.ConnectionState)
	}
	if !Advance(&mSparse, "live", start.Add(360*time.Second)) {
		t.Fatal("6th sample at 360s should confirm recovery")
	}
}

func TestUnknownInterruptsRecoveryWithoutInventingOffline(t *testing.T) {
	start := time.Now().UTC()
	m := Member{ConnectionState: StateOffline}
	Advance(&m, "live", start)
	Advance(&m, StateUnknown, start.Add(time.Minute))
	if m.ConnectionState != StateUnknown || m.RecoveryStartedAt != nil || m.RecoveredAt != nil {
		t.Fatal("unknown must invalidate observation without claiming offline")
	}
	if Advance(&m, "live", start.Add(6*time.Minute)) {
		t.Fatal("elapsed time without evidence is not recovery")
	}
}

func TestFlappingAndStaleObservationsCannotConfirmRecovery(t *testing.T) {
	start := time.Now().UTC()
	m := Member{ConnectionState: StateOffline}
	Advance(&m, "live", start)
	Advance(&m, "live", start.Add(60*time.Second))
	Advance(&m, StateOffline, start.Add(120*time.Second))
	Advance(&m, "live", start.Add(180*time.Second))
	for _, sec := range []int{240, 300, 360, 420} {
		if Advance(&m, "live", start.Add(time.Duration(sec)*time.Second)) {
			t.Fatalf("flapping must restart recovery window from 180s; confirmed too early at %ds", sec)
		}
	}
	Advance(&m, StateOffline, start.Add(100*time.Second))
	if m.ConnectionState != StateRecovering {
		t.Fatal("out of order observations must be ignored")
	}
	if !Advance(&m, "live", start.Add(480*time.Second)) {
		t.Fatal("expected recovery at 180s + 300s = 480s")
	}
}

func TestGapOver105SecondsAndRepeatedRecovery(t *testing.T) {
	start := time.Now().UTC()
	m := Member{ConnectionState: StateOffline}
	Advance(&m, "live", start)
	// Celah 106 detik (> MaxEvidenceGap = 105s) wajib memulai ulang jendela pemulihan.
	if Advance(&m, "live", start.Add(106*time.Second)) {
		t.Fatal("gap > 105s must restart observation")
	}
	if m.ConsecutiveLiveCount != 1 {
		t.Fatalf("expected consecutive count reset to 1, got %d", m.ConsecutiveLiveCount)
	}
	for _, sec := range []int{166, 226, 286, 346, 406} {
		Advance(&m, "live", start.Add(time.Duration(sec)*time.Second))
	}
	if m.ConnectionState != StateRecovered {
		t.Fatalf("expected recovery after 5 full minutes from 106s, got %s", m.ConnectionState)
	}
	if Advance(&m, "live", start.Add(420*time.Second)) {
		t.Fatal("repeated live must not repeat recovery notification")
	}
	Advance(&m, StateOffline, start.Add(430*time.Second))
	if m.ConnectionState != StateOffline || !m.IsActiveMember || m.RecoveredAt != nil {
		t.Fatal("offline recurrence must retain active membership and clear recovered_at")
	}

	// Anggota yang sudah diekskalasi untuk penggantian unit tidak boleh diubah oleh Advance.
	esc := Member{ConnectionState: StateEscalated}
	if Advance(&esc, "live", start.Add(500*time.Second)) || esc.ConnectionState != StateEscalated {
		t.Fatal("escalated member must ignore live/offline advance")
	}
}
