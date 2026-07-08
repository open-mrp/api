package constants

// PortalRegistrationStep represents the step of a buyer's customer-portal registration.
type PortalRegistrationStep string

const (
	// PortalRegistrationStepCustomerDetails indicates the buyer is choosing existing-vs-new and providing customer details (name, group, terms).
	PortalRegistrationStepCustomerDetails PortalRegistrationStep = "customer_details"
	// PortalRegistrationStepBillingAddress indicates the buyer is providing a billing address.
	PortalRegistrationStepBillingAddress PortalRegistrationStep = "billing_address"
	// PortalRegistrationStepContact indicates the buyer is providing contact information.
	PortalRegistrationStepContact PortalRegistrationStep = "contact"
	// PortalRegistrationStepCompleted indicates the registration has completed.
	PortalRegistrationStepCompleted PortalRegistrationStep = "completed"
)

func (s PortalRegistrationStep) IsValid() bool {
	switch s {
	case PortalRegistrationStepCustomerDetails,
		PortalRegistrationStepBillingAddress,
		PortalRegistrationStepContact,
		PortalRegistrationStepCompleted:
		return true
	default:
		return false
	}
}

// Ordinal returns the numeric ordering of a step. Higher values indicate later steps.
func (s PortalRegistrationStep) Ordinal() int {
	switch s {
	case PortalRegistrationStepCustomerDetails:
		return 0
	case PortalRegistrationStepBillingAddress:
		return 1
	case PortalRegistrationStepContact:
		return 2
	case PortalRegistrationStepCompleted:
		return 3
	default:
		return -1
	}
}

// IsAfter returns true if this step comes after the other step in the flow.
func (s PortalRegistrationStep) IsAfter(other PortalRegistrationStep) bool {
	return s.Ordinal() > other.Ordinal()
}

func (s PortalRegistrationStep) EnumValues() []string {
	return []string{
		string(PortalRegistrationStepCustomerDetails),
		string(PortalRegistrationStepBillingAddress),
		string(PortalRegistrationStepContact),
		string(PortalRegistrationStepCompleted),
	}
}

func (s *PortalRegistrationStep) StringPtr() *string {
	if s == nil {
		return nil
	}
	str := string(*s)
	return &str
}
