package constants

// CarrierType is what kind of carrier a carrier is. It is fixed when the carrier is created.
type CarrierType string

const (
	// CarrierTypeFedEx is FedEx, connected through Shippo for live rates and labels.
	CarrierTypeFedEx CarrierType = "fedex"
	// CarrierTypeUPS is UPS, connected through Shippo for live rates and labels.
	CarrierTypeUPS CarrierType = "ups"
	// CarrierTypeUSPS is USPS, connected through Shippo for live rates and labels.
	CarrierTypeUSPS CarrierType = "usps"
	// CarrierTypeLTL is a less-than-truckload freight carrier.
	CarrierTypeLTL CarrierType = "ltl"
	// CarrierTypeWillCall is the customer picking the order up.
	CarrierTypeWillCall CarrierType = "will_call"
	// CarrierTypeLocalDelivery is delivery by your own vehicles.
	CarrierTypeLocalDelivery CarrierType = "local_delivery"
	// CarrierTypeOther is any other carrier you manage yourself.
	CarrierTypeOther CarrierType = "other"
)

// IsShippo reports whether the carrier is connected through Shippo for live rates and labels.
func (t CarrierType) IsShippo() bool {
	switch t {
	case CarrierTypeFedEx, CarrierTypeUPS, CarrierTypeUSPS:
		return true
	default:
		return false
	}
}

// CarrierTypeFromStored is the type a carrier's stored code holds. Rows written before every carrier had a type hold
// no code, which reads as other.
func CarrierTypeFromStored(code *string) CarrierType {
	if code == nil || *code == "" {
		return CarrierTypeOther
	}
	return CarrierType(*code)
}

// TrackingURL returns a carrier tracking deep-link for the given tracking number, or "" when the carrier isn't one we can build a link for (or the tracking number is empty). Mirrors the carrier deep-links used by the frontend.
func TrackingURL(t CarrierType, trackingNumber string) string {
	if trackingNumber == "" {
		return ""
	}
	switch t {
	case CarrierTypeFedEx:
		return "https://www.fedex.com/fedextrack/?trknbr=" + trackingNumber
	case CarrierTypeUPS:
		return "https://www.ups.com/track/?loc=en_US&tracknum=" + trackingNumber
	case CarrierTypeUSPS:
		return "https://tools.usps.com/go/TrackConfirmAction?tLabels=" + trackingNumber
	default:
		return ""
	}
}

func (t CarrierType) IsValid() bool {
	switch t {
	case CarrierTypeFedEx, CarrierTypeUPS, CarrierTypeUSPS, CarrierTypeLTL, CarrierTypeWillCall, CarrierTypeLocalDelivery, CarrierTypeOther:
		return true
	default:
		return false
	}
}

func (t CarrierType) EnumValues() []string {
	return []string{string(CarrierTypeFedEx), string(CarrierTypeUPS), string(CarrierTypeUSPS), string(CarrierTypeLTL), string(CarrierTypeWillCall), string(CarrierTypeLocalDelivery), string(CarrierTypeOther)}
}
