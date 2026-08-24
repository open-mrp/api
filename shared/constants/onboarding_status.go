package constants

// OnboardingStatus is how far an account has progressed through onboarding, and whether it is usable.
type OnboardingStatus string

const (
	// OnboardingStatusUnclaimed means the account exists but nobody has taken ownership of it yet.
	OnboardingStatusUnclaimed OnboardingStatus = "unclaimed"
	// OnboardingStatusActive means the account is fully set up and usable.
	OnboardingStatusActive OnboardingStatus = "active"
	// OnboardingStatusSuspended means access is temporarily withdrawn, typically for non-payment.
	OnboardingStatusSuspended OnboardingStatus = "suspended"
	// OnboardingStatusDeactivated means the account has been shut down.
	OnboardingStatusDeactivated OnboardingStatus = "deactivated"
)

func (s OnboardingStatus) IsValid() bool {
	switch s {
	case OnboardingStatusUnclaimed, OnboardingStatusActive, OnboardingStatusSuspended, OnboardingStatusDeactivated:
		return true
	default:
		return false
	}
}

func (s OnboardingStatus) EnumValues() []string {
	return []string{
		string(OnboardingStatusUnclaimed),
		string(OnboardingStatusActive),
		string(OnboardingStatusSuspended),
		string(OnboardingStatusDeactivated),
	}
}
