package constants

// Account Mode is the intended mode of operation for a request. This must be
// either "production" or "sandbox". The mode of the request is a useful way
// to ensure that a given request stays within the defined boundaries of it's
// intended mode.
type AccountMode string

const (
	// AccountModeProduction indicates that the request is targeting production
	// resources and integrations. This mode has real-world consequences and
	// should be used with care.
	AccountModeProduction AccountMode = "prod"
	// AccountModeSandbox indicates that the request is targeting sandbox
	// resources and integrations. This mode is useful for testing and development
	// and can be used more dangerously.
	AccountModeSandbox AccountMode = "test"
)

func (m AccountMode) IsValid() bool {
	switch m {
	case AccountModeProduction, AccountModeSandbox:
		return true
	default:
		return false
	}
}

func (m AccountMode) EnumValues() []string {
	return []string{string(AccountModeProduction), string(AccountModeSandbox)}
}
