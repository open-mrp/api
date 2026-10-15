package constants

// FreightPolicy says whether freight is billed to the buyer.
type FreightPolicy string

const (
	// FreightPolicyFree indicates no shipping cost to the buyer.
	FreightPolicyFree FreightPolicy = "free"
	// FreightPolicyBilled indicates that freight is billed to the buyer.
	FreightPolicyBilled FreightPolicy = "billed"
)

// Stored freight status codes. Customer and account-group rows keep these codes; only the API values changed.
const (
	freightStatusCodeFree   = "free_freight"
	freightStatusCodeBilled = "billed_freight"
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

// FreightPolicyFromStatusCode reads a stored freight_status_code. Values that are already a policy pass through.
func FreightPolicyFromStatusCode(code string) FreightPolicy {
	switch code {
	case freightStatusCodeFree:
		return FreightPolicyFree
	case freightStatusCodeBilled:
		return FreightPolicyBilled
	default:
		return FreightPolicy(code)
	}
}

// StatusCode is the freight_status_code stored for the policy.
func (m FreightPolicy) StatusCode() string {
	switch m {
	case FreightPolicyFree:
		return freightStatusCodeFree
	case FreightPolicyBilled:
		return freightStatusCodeBilled
	default:
		return string(m)
	}
}
