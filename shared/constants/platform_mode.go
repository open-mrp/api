package constants

// PlatformMode represents the mode the application is running in
type PlatformMode string

const (
	PlatformModeProduction  PlatformMode = "production"
	PlatformModeDevelopment PlatformMode = "development"
)

func (m PlatformMode) IsValid() bool {
	switch m {
	case PlatformModeProduction, PlatformModeDevelopment:
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
