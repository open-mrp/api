package constants

// CarrierBillingType represents the carrier billing type for an account relation.
type CarrierBillingType string

const (
	// CarrierBillingTypeSender indicates the sender pays for shipping.
	CarrierBillingTypeSender CarrierBillingType = "sender"
	// CarrierBillingTypeThirdParty indicates a third party pays for shipping.
	CarrierBillingTypeThirdParty CarrierBillingType = "third_party"
)

func (m CarrierBillingType) IsValid() bool {
	switch m {
	case CarrierBillingTypeSender, CarrierBillingTypeThirdParty:
		return true
	default:
		return false
	}
}

func (m CarrierBillingType) EnumValues() []string {
	return []string{string(CarrierBillingTypeSender), string(CarrierBillingTypeThirdParty)}
}
