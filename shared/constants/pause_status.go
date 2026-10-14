package constants

// PauseStatus is the `status` of a resource that is routinely paused and resumed rather than retired, such as an
// integration, agent or account user. Tables store it as `is_active`.
type PauseStatus string

const (
	// PauseStatusActive is in use.
	PauseStatusActive PauseStatus = "active"
	// PauseStatusDisabled is paused until it is set back to active.
	PauseStatusDisabled PauseStatus = "disabled"
)

func (s PauseStatus) IsValid() bool {
	switch s {
	case PauseStatusActive, PauseStatusDisabled:
		return true
	default:
		return false
	}
}

func (s PauseStatus) EnumValues() []string {
	return []string{string(PauseStatusActive), string(PauseStatusDisabled)}
}

// IsActive is the status as the `is_active` column stores it.
func (s PauseStatus) IsActive() bool {
	return s == PauseStatusActive
}

// PauseStatusFromIsActive is the status an `is_active` column holds.
func PauseStatusFromIsActive(isActive bool) PauseStatus {
	if isActive {
		return PauseStatusActive
	}
	return PauseStatusDisabled
}

func (s *PauseStatus) StringPtr() *string {
	if s == nil {
		return nil
	}
	v := string(*s)
	return &v
}
