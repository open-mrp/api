package constants

type SandboxMode string

const (
	SandboxModeBlank  SandboxMode = "blank"
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
