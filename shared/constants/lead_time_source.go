package constants

// LeadTimeSource names which rule produced an order's ship-by commitment. Stored on the order alongside the date so the commitment can always explain itself, rather than requiring the settings to be reconstructed as they stood when the order was issued.
type LeadTimeSource string

const (
	// LeadTimeSourceCustomer is the customer's own default lead time.
	LeadTimeSourceCustomer LeadTimeSource = "customer"
	// LeadTimeSourceParentCustomer is the lead time inherited from the customer's parent account.
	LeadTimeSourceParentCustomer LeadTimeSource = "parent_customer"
	// LeadTimeSourceAccountGroup is the lead time inherited from the customer's account group.
	LeadTimeSourceAccountGroup LeadTimeSource = "account_group"
	// LeadTimeSourceAccount is the account-wide default, the last fallback in the chain.
	LeadTimeSourceAccount LeadTimeSource = "account"
	// LeadTimeSourceManual is an explicitly promised delivery date, which overrides every rule. Named before the other two per-order bases existed; it means the promised date specifically, not "somebody set this by hand".
	LeadTimeSourceManual LeadTimeSource = "manual"
	// LeadTimeSourceOrderLeadTime is a lead time set on one order, replacing the standing customer chain.
	LeadTimeSourceOrderLeadTime LeadTimeSource = "order_lead_time"
	// LeadTimeSourceOrderShipBy is a ship date pinned on one order, bypassing transit and the receiving calendar.
	LeadTimeSourceOrderShipBy LeadTimeSource = "order_ship_by"
)

func (m LeadTimeSource) IsValid() bool {
	switch m {
	case LeadTimeSourceCustomer, LeadTimeSourceParentCustomer, LeadTimeSourceAccountGroup, LeadTimeSourceAccount, LeadTimeSourceManual, LeadTimeSourceOrderLeadTime, LeadTimeSourceOrderShipBy:
		return true
	default:
		return false
	}
}

func (m LeadTimeSource) EnumValues() []string {
	return []string{
		string(LeadTimeSourceCustomer),
		string(LeadTimeSourceParentCustomer),
		string(LeadTimeSourceAccountGroup),
		string(LeadTimeSourceAccount),
		string(LeadTimeSourceManual),
		string(LeadTimeSourceOrderLeadTime),
		string(LeadTimeSourceOrderShipBy),
	}
}

func (m *LeadTimeSource) StringPtr() *string {
	if m == nil {
		return nil
	}
	s := string(*m)
	return &s
}
