package constants

// ShippingTermType represents the freight pricing model for a shipping term.
type ShippingTermType string

const (
	// ShippingTermTypeFreeFreight indicates no shipping cost to the buyer.
	ShippingTermTypeFreeFreight ShippingTermType = "free_freight"
	// ShippingTermTypeFlatRateFreight indicates a fixed shipping cost regardless of order details.
	ShippingTermTypeFlatRateFreight ShippingTermType = "flat_rate_freight"
	// ShippingTermTypeCarrierRateFreight indicates shipping cost is determined by the carrier's rate.
	ShippingTermTypeCarrierRateFreight ShippingTermType = "carrier_rate_freight"
)

func (m ShippingTermType) IsValid() bool {
	switch m {
	case ShippingTermTypeFreeFreight, ShippingTermTypeFlatRateFreight, ShippingTermTypeCarrierRateFreight:
		return true
	default:
		return false
	}
}

func (m ShippingTermType) EnumValues() []string {
	return []string{
		string(ShippingTermTypeFreeFreight),
		string(ShippingTermTypeFlatRateFreight),
		string(ShippingTermTypeCarrierRateFreight),
	}
}
