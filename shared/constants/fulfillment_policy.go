package constants

// FulfillmentPolicy is how a SKU is produced: to a forecast, or only against orders already placed.
type FulfillmentPolicy string

const (
	// FulfillmentPolicyMakeToStock builds to the forecast and holds a safety stock against its variability.
	FulfillmentPolicyMakeToStock FulfillmentPolicy = "make_to_stock"
	// FulfillmentPolicyMakeToOrder contributes no forecast demand and holds no safety stock, so it is built only against the order book.
	FulfillmentPolicyMakeToOrder FulfillmentPolicy = "make_to_order"
)

func (m FulfillmentPolicy) IsValid() bool {
	switch m {
	case FulfillmentPolicyMakeToStock, FulfillmentPolicyMakeToOrder:
		return true
	default:
		return false
	}
}

func (m FulfillmentPolicy) EnumValues() []string {
	return []string{
		string(FulfillmentPolicyMakeToStock),
		string(FulfillmentPolicyMakeToOrder),
	}
}

func (m *FulfillmentPolicy) StringPtr() *string {
	if m == nil {
		return nil
	}
	s := string(*m)
	return &s
}

// FulfillmentPolicySource names which rule in the chain decided a SKU's policy.
type FulfillmentPolicySource string

const (
	// FulfillmentPolicySourceItem is an explicit per-item override.
	FulfillmentPolicySourceItem FulfillmentPolicySource = "item"
	// FulfillmentPolicySourceProductLine is the default set on the item's product line.
	FulfillmentPolicySourceProductLine FulfillmentPolicySource = "product_line"
	// FulfillmentPolicySourceAccountDefault is the account-wide fallback.
	FulfillmentPolicySourceAccountDefault FulfillmentPolicySource = "account_default"
)

func (m FulfillmentPolicySource) IsValid() bool {
	switch m {
	case FulfillmentPolicySourceItem, FulfillmentPolicySourceProductLine, FulfillmentPolicySourceAccountDefault:
		return true
	default:
		return false
	}
}

func (m FulfillmentPolicySource) EnumValues() []string {
	return []string{
		string(FulfillmentPolicySourceItem),
		string(FulfillmentPolicySourceProductLine),
		string(FulfillmentPolicySourceAccountDefault),
	}
}

func (m *FulfillmentPolicySource) StringPtr() *string {
	if m == nil {
		return nil
	}
	s := string(*m)
	return &s
}
