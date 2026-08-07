package constants

// LeadTimeSource names which rule produced an order's ship-by commitment. Stored on the order alongside the date so the commitment can always explain itself, rather than requiring the settings to be reconstructed as they stood when the order was issued.
type LeadTimeSource string

const (
	// LeadTimeSourceCustomer is the customer's own default lead time.
	LeadTimeSourceCustomer LeadTimeSource = "customer"
	// LeadTimeSourceAccountGroup is the lead time inherited from the customer's account group.
	LeadTimeSourceAccountGroup LeadTimeSource = "account_group"
	// LeadTimeSourceAccount is the account-wide default, the last fallback in the chain.
	LeadTimeSourceAccount LeadTimeSource = "account"
	// LeadTimeSourceManual is an explicitly promised date, which overrides every rule.
	LeadTimeSourceManual LeadTimeSource = "manual"
)

func (m LeadTimeSource) IsValid() bool {
	switch m {
	case LeadTimeSourceCustomer, LeadTimeSourceAccountGroup, LeadTimeSourceAccount, LeadTimeSourceManual:
		return true
	default:
		return false
	}
}

func (m LeadTimeSource) EnumValues() []string {
	return []string{
		string(LeadTimeSourceCustomer),
		string(LeadTimeSourceAccountGroup),
		string(LeadTimeSourceAccount),
		string(LeadTimeSourceManual),
	}
}

func (m *LeadTimeSource) StringPtr() *string {
	if m == nil {
		return nil
	}
	s := string(*m)
	return &s
}
