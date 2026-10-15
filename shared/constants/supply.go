package constants

// Supply is where an item's supply comes from: inventory, production or purchasing.
type Supply string

const (
	// SupplyStock takes the item from inventory, however it was replenished.
	SupplyStock Supply = "stock"
	// SupplyMake produces the item for the demand it is pegged to.
	SupplyMake Supply = "make"
	// SupplyBuy purchases the item for the demand it is pegged to.
	SupplyBuy Supply = "buy"
)

func (s Supply) IsValid() bool {
	switch s {
	case SupplyStock, SupplyMake, SupplyBuy:
		return true
	default:
		return false
	}
}

func (s Supply) EnumValues() []string {
	return []string{string(SupplyStock), string(SupplyMake), string(SupplyBuy)}
}

func (s *Supply) StringPtr() *string {
	if s == nil {
		return nil
	}
	v := string(*s)
	return &v
}

// FulfillmentPolicy is the planning policy the supply implies: stock is built to stock, make is built to order. Buy
// has none, since planning never produces it.
func (s Supply) FulfillmentPolicy() (FulfillmentPolicy, bool) {
	switch s {
	case SupplyStock:
		return FulfillmentPolicyMakeToStock, true
	case SupplyMake:
		return FulfillmentPolicyMakeToOrder, true
	default:
		return "", false
	}
}
