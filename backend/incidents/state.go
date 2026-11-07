package incidents

import "time"

const (
	RecoveryWindow     = 5 * time.Minute
	MaxEvidenceGap     = 105 * time.Second
	MinRecoverySamples = 6
)

// Advance consumes fresh samples. It never turns missing evidence into offline.
// The return value means recovery was newly confirmed (stabil >= 5 menit dengan
// minimal 6 sampel berturut-turut dan celah antar-sampel <= 105 detik), not that
// a ticket closed.
func Advance(m *Member, state string, at time.Time) bool {
	if m.LastObservedAt != nil && !at.After(*m.LastObservedAt) {
		return false
	}
	if m.ConnectionState == StateEscalated {
		return false
	}
	previous := m.LastObservedAt
	m.LastObservedAt = &at
	m.IsActiveMember = true
	switch state {
	case StateOffline:
		m.ConnectionState = StateOffline
		m.RecoveryStartedAt = nil
		m.RecoveredAt = nil
		m.ConsecutiveLiveCount = 0
	case "live":
		if previous == nil || at.Sub(*previous) > MaxEvidenceGap || m.RecoveryStartedAt == nil {
			m.RecoveryStartedAt = &at
			m.ConsecutiveLiveCount = 1
			m.ConnectionState = StateRecovering
			return false
		}
		if m.ConnectionState == StateRecovered {
			m.ConsecutiveLiveCount++
			return false
		}
		m.ConsecutiveLiveCount++
		if at.Sub(*m.RecoveryStartedAt) >= RecoveryWindow && m.ConsecutiveLiveCount >= MinRecoverySamples {
			m.ConnectionState = StateRecovered
			m.RecoveredAt = &at
			return true
		}
		m.ConnectionState = StateRecovering
	default:
		m.ConnectionState = StateUnknown
		m.RecoveryStartedAt = nil
		m.RecoveredAt = nil
		m.ConsecutiveLiveCount = 0
	}
	return false
}
