package constants

// EDIStatus represents whether EDI is enabled for a customer.
type EDIStatus string

const (
	// EDIStatusEnabled indicates EDI is enabled.
	EDIStatusEnabled EDIStatus = "enabled"
	// EDIStatusDisabled indicates EDI is disabled.
	EDIStatusDisabled EDIStatus = "disabled"
)

func (m EDIStatus) IsValid() bool {
	switch m {
	case EDIStatusEnabled, EDIStatusDisabled:
		return true
	default:
		return false
	}
}

func (m EDIStatus) EnumValues() []string {
	return []string{
		string(EDIStatusEnabled),
		string(EDIStatusDisabled),
	}
}
