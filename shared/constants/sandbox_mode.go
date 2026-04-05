package constants

// SandboxMode represents how a sandbox environment is initialized.
type SandboxMode string

const (
	// SandboxModeBlank creates an empty sandbox with no pre-populated data.
	SandboxModeBlank SandboxMode = "blank"
	// SandboxModeSeeded creates a sandbox pre-populated with sample data.
	SandboxModeSeeded SandboxMode = "seeded"
)

func (m SandboxMode) IsValid() bool {
	switch m {
	case SandboxModeBlank, SandboxModeSeeded:
		return true
	default:
		return false
	}
}

func (m SandboxMode) EnumValues() []string {
	return []string{string(SandboxModeBlank), string(SandboxModeSeeded)}
}
