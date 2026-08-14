package constants

// PricingFindingReason names why a price was flagged. The analysis runs two independent checks and a finding is only produced when at least one fails, so the values enumerate every combination that can actually occur.
type PricingFindingReason string

const (
	// PricingFindingReasonBelowPeerMedian indicates that the price sits far enough below what comparable customers pay to be flagged, but still clears the target gross margin.
	PricingFindingReasonBelowPeerMedian PricingFindingReason = "below_peer_median"
	// PricingFindingReasonBelowTargetMargin indicates that the price fails to clear the target gross margin, but is not unusually low against comparable customers.
	PricingFindingReasonBelowTargetMargin PricingFindingReason = "below_target_margin"
	// PricingFindingReasonBelowPeerMedianAndTargetMargin indicates that the price is both unusually low against comparable customers and fails to clear the target gross margin.
	PricingFindingReasonBelowPeerMedianAndTargetMargin PricingFindingReason = "below_peer_median_and_target_margin"
)

func (m PricingFindingReason) IsValid() bool {
	switch m {
	case PricingFindingReasonBelowPeerMedian, PricingFindingReasonBelowTargetMargin, PricingFindingReasonBelowPeerMedianAndTargetMargin:
		return true
	default:
		return false
	}
}

func (m PricingFindingReason) EnumValues() []string {
	return []string{
		string(PricingFindingReasonBelowPeerMedian),
		string(PricingFindingReasonBelowTargetMargin),
		string(PricingFindingReasonBelowPeerMedianAndTargetMargin),
	}
}

// AccountPriceOrigin says how a customer comes to receive a contracted price, which decides where it has to be changed.
type AccountPriceOrigin string

const (
	// AccountPriceOriginDirect indicates that the price is recorded against this customer.
	AccountPriceOriginDirect AccountPriceOrigin = "direct"
	// AccountPriceOriginInherited indicates that the price is recorded against the customer's parent account and reaches this customer through it.
	AccountPriceOriginInherited AccountPriceOrigin = "inherited"
)

func (m AccountPriceOrigin) IsValid() bool {
	switch m {
	case AccountPriceOriginDirect, AccountPriceOriginInherited:
		return true
	default:
		return false
	}
}

func (m AccountPriceOrigin) EnumValues() []string {
	return []string{string(AccountPriceOriginDirect), string(AccountPriceOriginInherited)}
}

func (m *PricingFindingReason) StringPtr() *string {
	if m == nil {
		return nil
	}
	s := string(*m)
	return &s
}

func (m *AccountPriceOrigin) StringPtr() *string {
	if m == nil {
		return nil
	}
	s := string(*m)
	return &s
}
