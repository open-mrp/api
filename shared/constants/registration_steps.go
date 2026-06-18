package constants

// RegistrationStep represents the step of the registration process.
type RegistrationStep string

const (
	// RegistrationStepVerification indicates that the user is verifying their email address.
	RegistrationStepVerification RegistrationStep = "verification"
	// RegistrationStepUserDetails indicates that the user is providing their user details.
	RegistrationStepUserDetails RegistrationStep = "user_details"
	// RegistrationStepAccountDetails indicates that the user is providing their account details.
	RegistrationStepAccountDetails RegistrationStep = "account_details"
	// RegistrationStepReview indicates that the user is reviewing their registration details.
	RegistrationStepReview RegistrationStep = "review"
	// RegistrationStepPayment indicates that the user is providing their payment details.
	RegistrationStepPayment RegistrationStep = "payment"
	// RegistrationStepCompleted indicates that the user has completed the registration process.
	RegistrationStepCompleted RegistrationStep = "completed"
)

func (m RegistrationStep) IsValid() bool {
	switch m {
	case RegistrationStepVerification,
		RegistrationStepUserDetails,
		RegistrationStepAccountDetails,
		RegistrationStepReview,
		RegistrationStepPayment,
		RegistrationStepCompleted:
		return true
	default:
		return false
	}
}

// Ordinal returns the numeric ordering of a registration step. Higher values indicate later steps in the flow.
func (m RegistrationStep) Ordinal() int {
	switch m {
	case RegistrationStepVerification:
		return 0
	case RegistrationStepUserDetails:
		return 1
	case RegistrationStepAccountDetails:
		return 2
	case RegistrationStepReview:
		return 3
	case RegistrationStepPayment:
		return 4
	case RegistrationStepCompleted:
		return 5
	default:
		return -1
	}
}

// IsAfter returns true if this step comes after the other step in the flow.
func (m RegistrationStep) IsAfter(other RegistrationStep) bool {
	return m.Ordinal() > other.Ordinal()
}

func (m RegistrationStep) EnumValues() []string {
	return []string{string(RegistrationStepVerification), string(RegistrationStepUserDetails), string(RegistrationStepAccountDetails), string(RegistrationStepReview), string(RegistrationStepPayment), string(RegistrationStepCompleted)}
}
