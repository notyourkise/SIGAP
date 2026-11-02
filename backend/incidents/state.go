package incidents

import "time"

const RecoveryWindow = 5 * time.Minute
const MaxEvidenceGap = 150 * time.Second

// Advance consumes fresh samples. It never turns missing evidence into offline.
// The return value means recovery was newly confirmed, not that a ticket closed.
func Advance(m *Member, state string, at time.Time) bool {
	if m.LastObservedAt != nil && !at.After(*m.LastObservedAt) {
		return false
	}
	previous := m.LastObservedAt
	m.LastObservedAt = &at
	m.IsActiveMember = true
	switch state {
	case "offline":
		m.ConnectionState = "offline"
		m.RecoveryStartedAt = nil
		m.ConsecutiveLiveCount = 0
	case "live":
		if previous == nil || at.Sub(*previous) > MaxEvidenceGap || m.RecoveryStartedAt == nil {
			m.RecoveryStartedAt = &at
			m.ConsecutiveLiveCount = 1
			m.ConnectionState = "recovering"
			return false
		}
		if m.ConnectionState == "recovered" {
			return false
		}
		m.ConsecutiveLiveCount++
		if at.Sub(*m.RecoveryStartedAt) >= RecoveryWindow && m.ConsecutiveLiveCount >= 4 {
			m.ConnectionState = "recovered"
			return true
		}
		m.ConnectionState = "recovering"
	default:
		m.ConnectionState = "unknown"
		m.RecoveryStartedAt = nil
		m.ConsecutiveLiveCount = 0
	}
	return false
}
