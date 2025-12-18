package constants

// Represents the mode the user intends to target with their request
type AccountMode string

const (
	AccountModeProduction AccountMode = "prod"
	AccountModeSandbox    AccountMode = "test"
)

func (m AccountMode) IsValid() bool {
	switch m {
	case AccountModeProduction, AccountModeSandbox:
		return true
	default:
		return false
	}
}
