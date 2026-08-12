package constants

// TransitSource names where an order's transit time came from. Stored on the order beside the day count for the same reason the lead-time source is: a commitment has to be able to explain itself later, and "3 days" reads very differently depending on whether the carrier quoted that lane or someone typed a default into the service level.
type TransitSource string

const (
	// TransitSourceCarrierLane is a cached carrier estimate for this order's exact lane, the most specific answer available.
	TransitSourceCarrierLane TransitSource = "carrier_lane"
	// TransitSourceServiceLevel is the service level's default, used when no lane estimate has been cached and for carriers that cannot be rated.
	TransitSourceServiceLevel TransitSource = "service_level"
)

func (m TransitSource) IsValid() bool {
	switch m {
	case TransitSourceCarrierLane, TransitSourceServiceLevel:
		return true
	default:
		return false
	}
}

func (m TransitSource) EnumValues() []string {
	return []string{
		string(TransitSourceCarrierLane),
		string(TransitSourceServiceLevel),
	}
}

func (m *TransitSource) StringPtr() *string {
	if m == nil {
		return nil
	}
	s := string(*m)
	return &s
}
