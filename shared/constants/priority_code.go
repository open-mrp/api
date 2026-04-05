package constants

// PriorityCode represents the code of a priority level.
type PriorityCode string

const (
	// PriorityCodeLow indicates a low priority.
	PriorityCodeLow PriorityCode = "low"
	// PriorityCodeNormal indicates a normal priority.
	PriorityCodeNormal PriorityCode = "normal"
	// PriorityCodeHigh indicates a high priority.
	PriorityCodeHigh PriorityCode = "high"
)

func (m PriorityCode) IsValid() bool {
	switch m {
	case PriorityCodeLow, PriorityCodeNormal, PriorityCodeHigh:
		return true
	default:
		return false
	}
}

func (m PriorityCode) EnumValues() []string {
	return []string{
		string(PriorityCodeLow),
		string(PriorityCodeNormal),
		string(PriorityCodeHigh),
	}
}
