package constants

// AccountGroupType represents the type of an account group.
type AccountGroupType string

const (
	// AccountGroupTypePricingGroup indicates a pricing-based account group.
	AccountGroupTypePricingGroup AccountGroupType = "pricing_group"
	// AccountGroupTypeTypeGroup indicates a type-based account group.
	AccountGroupTypeTypeGroup AccountGroupType = "type_group"
)

func (m AccountGroupType) IsValid() bool {
	switch m {
	case AccountGroupTypePricingGroup, AccountGroupTypeTypeGroup:
		return true
	default:
		return false
	}
}

func (m AccountGroupType) EnumValues() []string {
	return []string{string(AccountGroupTypePricingGroup), string(AccountGroupTypeTypeGroup)}
}
