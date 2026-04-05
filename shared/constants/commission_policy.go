package constants

// CommissionPolicy represents the commission status of an account group.
type CommissionPolicy string

const (
	// CommissionPolicyApplied indicates that commission is applied.
	CommissionPolicyApplied CommissionPolicy = "commission_applied"
	// CommissionPolicyExempt indicates that the account group is exempt from commission.
	CommissionPolicyExempt CommissionPolicy = "commission_exempt"
)

func (m CommissionPolicy) IsValid() bool {
	switch m {
	case CommissionPolicyApplied, CommissionPolicyExempt:
		return true
	default:
		return false
	}
}

func (m CommissionPolicy) EnumValues() []string {
	return []string{string(CommissionPolicyApplied), string(CommissionPolicyExempt)}
}

// CommissionPolicyFromBool converts a boolean is_commission_exempt flag to a CommissionPolicy.
func CommissionPolicyFromBool(isExempt bool) CommissionPolicy {
	if isExempt {
		return CommissionPolicyExempt
	}
	return CommissionPolicyApplied
}

// ToBool converts a CommissionPolicy to a boolean is_commission_exempt flag.
func (m CommissionPolicy) ToBool() bool {
	return m == CommissionPolicyExempt
}
