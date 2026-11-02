package handlers

import (
	"testing"
	"time"
)

func TestIncidentCadenceIsOptInAndBounded(t *testing.T) {
	t.Setenv("CCTV_INCIDENTS_ENABLED", "false")
	t.Setenv("CCTV_VERIFY_INTERVAL_MINUTES", "")
	if streamVerifyInterval() != 15*time.Minute {
		t.Fatal("legacy interval changed")
	}
	t.Setenv("CCTV_INCIDENTS_ENABLED", "true")
	for _, value := range []string{"", "0", "-1", "999", "broken"} {
		t.Setenv("CCTV_INCIDENT_INTERVAL_SECONDS", value)
		if streamVerifyInterval() != time.Minute {
			t.Fatal("invalid interval did not use bounded fallback")
		}
	}
	t.Setenv("CCTV_INCIDENT_INTERVAL_SECONDS", "30")
	if streamVerifyInterval() != 30*time.Second {
		t.Fatal("valid incident interval ignored")
	}
}
