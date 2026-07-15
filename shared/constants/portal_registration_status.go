package constants

import "time"

// PortalRegistrationSessionTTL bounds how long an incomplete buyer registration session can be resumed before it reads as expired. Applied logically at read time (no cleanup job).
const PortalRegistrationSessionTTL = 7 * 24 * time.Hour

// PortalRegistrationStatus is the lifecycle state of a buyer's customer-portal registration session, derived from its completion/abandonment timestamps and the resume TTL. It lets customer service see which registrations stalled so they can follow up.
type PortalRegistrationStatus string

const (
	// PortalRegistrationStatusInProgress indicates an incomplete session still within its resume window.
	PortalRegistrationStatusInProgress PortalRegistrationStatus = "in_progress"
	// PortalRegistrationStatusCompleted indicates the buyer finished registering.
	PortalRegistrationStatusCompleted PortalRegistrationStatus = "completed"
	// PortalRegistrationStatusAbandoned indicates the buyer explicitly abandoned the session.
	PortalRegistrationStatusAbandoned PortalRegistrationStatus = "abandoned"
	// PortalRegistrationStatusExpired indicates an incomplete session whose resume window has elapsed.
	PortalRegistrationStatusExpired PortalRegistrationStatus = "expired"
)

func (s PortalRegistrationStatus) IsValid() bool {
	switch s {
	case PortalRegistrationStatusInProgress,
		PortalRegistrationStatusCompleted,
		PortalRegistrationStatusAbandoned,
		PortalRegistrationStatusExpired:
		return true
	default:
		return false
	}
}

func (s PortalRegistrationStatus) EnumValues() []string {
	return []string{
		string(PortalRegistrationStatusInProgress),
		string(PortalRegistrationStatusCompleted),
		string(PortalRegistrationStatusAbandoned),
		string(PortalRegistrationStatusExpired),
	}
}

func (s *PortalRegistrationStatus) StringPtr() *string {
	if s == nil {
		return nil
	}
	v := string(*s)
	return &v
}
