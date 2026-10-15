package constants

// CommissionPolicy says whether sales earn commission.
type CommissionPolicy string

const (
	// CommissionPolicyApplied indicates that commission is applied.
	CommissionPolicyApplied CommissionPolicy = "applied"
	// CommissionPolicyExempt indicates that sales are exempt from commission.
	CommissionPolicyExempt CommissionPolicy = "exempt"
)

// Stored commission status codes. Customer and account-group rows keep these codes; only the API values changed.
const (
	commissionStatusCodeApplied = "commission_applied"
	commissionStatusCodeExempt  = "commission_exempt"
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

// CommissionPolicyFromStatusCode reads a stored commission_status_code. Values that are already a policy pass through.
func CommissionPolicyFromStatusCode(code string) CommissionPolicy {
	switch code {
	case commissionStatusCodeApplied:
		return CommissionPolicyApplied
	case commissionStatusCodeExempt:
		return CommissionPolicyExempt
	default:
		return CommissionPolicy(code)
	}
}

// StatusCode is the commission_status_code stored for the policy.
func (m CommissionPolicy) StatusCode() string {
	switch m {
	case CommissionPolicyApplied:
		return commissionStatusCodeApplied
	case CommissionPolicyExempt:
		return commissionStatusCodeExempt
	default:
		return string(m)
	}
}
