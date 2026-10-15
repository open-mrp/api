package constants

// ShippingTermType is how a shipping term charges freight.
type ShippingTermType string

const (
	// ShippingTermTypeFree never charges freight.
	ShippingTermTypeFree ShippingTermType = "free"
	// ShippingTermTypeFlatRate charges the term's flat rate on every order.
	ShippingTermTypeFlatRate ShippingTermType = "flat_rate"
	// ShippingTermTypeCarrierRate charges the carrier's quoted rate for the order's service level.
	ShippingTermTypeCarrierRate ShippingTermType = "carrier_rate"
)

func (m ShippingTermType) IsValid() bool {
	switch m {
	case ShippingTermTypeFree, ShippingTermTypeFlatRate, ShippingTermTypeCarrierRate:
		return true
	default:
		return false
	}
}

func (m ShippingTermType) EnumValues() []string {
	return []string{string(ShippingTermTypeFree), string(ShippingTermTypeFlatRate), string(ShippingTermTypeCarrierRate)}
}
