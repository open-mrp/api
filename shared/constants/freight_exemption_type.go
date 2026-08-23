package constants

// FreightExemptionType names the special freight outcome applied to a set of rate options.
type FreightExemptionType string

const (
	// FreightExemptionTypeFreightExempt means the order is exempt from freight, so no options are returned.
	FreightExemptionTypeFreightExempt FreightExemptionType = "freight_exempt"
	// FreightExemptionTypeMinimumOrderMet means the order cleared the shipping term's free-shipping minimum, so options are rated at zero.
	FreightExemptionTypeMinimumOrderMet FreightExemptionType = "minimum_order_met"
	// FreightExemptionTypeFlatRate means the shipping term's flat rate replaced every option's carrier rate.
	FreightExemptionTypeFlatRate FreightExemptionType = "flat_rate"
	// FreightExemptionTypeNone means standard carrier rates apply.
	FreightExemptionTypeNone FreightExemptionType = "none"
)

func (t FreightExemptionType) IsValid() bool {
	switch t {
	case FreightExemptionTypeFreightExempt, FreightExemptionTypeMinimumOrderMet, FreightExemptionTypeFlatRate, FreightExemptionTypeNone:
		return true
	default:
		return false
	}
}

func (t FreightExemptionType) EnumValues() []string {
	return []string{
		string(FreightExemptionTypeFreightExempt),
		string(FreightExemptionTypeMinimumOrderMet),
		string(FreightExemptionTypeFlatRate),
		string(FreightExemptionTypeNone),
	}
}
