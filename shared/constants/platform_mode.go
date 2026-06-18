package constants

// PlatformMode represents the mode the server is running in.
type PlatformMode string

const (
	// PlatformModeProduction indicates that the server is running in production mode. This is the default mode and should be used for all production environments.
	PlatformModeProduction PlatformMode = "production"
	// PlatformModeDevelopment indicates that the server is running in development mode. This mode is somewhat more permissive and has guardrails to mock some application behaviors for testing and development purposes.
	PlatformModeDevelopment PlatformMode = "development"
	// PlatformModeTest indicates that the server is running in test mode. Third-party integrations (Stripe, AWS, Google Maps, etc.) are replaced with no-op stubs.
	PlatformModeTest PlatformMode = "test"
)

func (m PlatformMode) IsValid() bool {
	switch m {
	case PlatformModeProduction, PlatformModeDevelopment, PlatformModeTest:
		return true
	default:
		return false
	}
}

func (m PlatformMode) IsProduction() bool {
	return m == PlatformModeProduction
}

func (m PlatformMode) IsDevelopment() bool {
	return m == PlatformModeDevelopment
}

func (m PlatformMode) IsTest() bool {
	return m == PlatformModeTest
}

func (m PlatformMode) EnumValues() []string {
	return []string{string(PlatformModeProduction), string(PlatformModeDevelopment), string(PlatformModeTest)}
}
