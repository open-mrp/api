package constants

// ScanningStepType is how many distinct part items a production step draws on, which decides how an operator scans into it.
type ScanningStepType string

const (
	// ScanningStepTypeSingle indicates that the step consumes one part item, so a single scan advances a batch into it.
	ScanningStepTypeSingle ScanningStepType = "single"
	// ScanningStepTypeMultiPart indicates that the step consumes several distinct part items, so one batch per part must be scanned before merging or splitting into it.
	ScanningStepTypeMultiPart ScanningStepType = "multi_part"
)

// IsValid reports whether the value is a known scanning step type.
func (s ScanningStepType) IsValid() bool {
	switch s {
	case ScanningStepTypeSingle, ScanningStepTypeMultiPart:
		return true
	default:
		return false
	}
}

// EnumValues lists the scanning step types for schema generation.
func (s ScanningStepType) EnumValues() []string {
	return []string{string(ScanningStepTypeSingle), string(ScanningStepTypeMultiPart)}
}
