package constants

// FreightPolicy represents the freight status of an account group.
type FreightPolicy string

const (
	// FreightPolicyFree indicates no shipping cost to the buyer.
	FreightPolicyFree FreightPolicy = "free_freight"
	// FreightPolicyBilled indicates that freight is billed to the buyer.
	FreightPolicyBilled FreightPolicy = "billed_freight"
)

func (m FreightPolicy) IsValid() bool {
	switch m {
	case FreightPolicyFree, FreightPolicyBilled:
		return true
	default:
		return false
	}
}

func (m FreightPolicy) EnumValues() []string {
	return []string{string(FreightPolicyFree), string(FreightPolicyBilled)}
}

// FreightPolicyFromBool converts a boolean is_freight_exempt flag to a FreightPolicy.
func FreightPolicyFromBool(isExempt bool) FreightPolicy {
	if isExempt {
		return FreightPolicyFree
	}
	return FreightPolicyBilled
}

// ToBool converts a FreightPolicy to a boolean is_freight_exempt flag.
func (m FreightPolicy) ToBool() bool {
	return m == FreightPolicyFree
}
