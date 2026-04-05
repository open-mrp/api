package constants

// CarrierCode identifies a shipping carrier.
type CarrierCode string

const (
	// CarrierCodeFedEx identifies the FedEx carrier.
	CarrierCodeFedEx CarrierCode = "fedex"
	// CarrierCodeUPS identifies the UPS carrier.
	CarrierCodeUPS CarrierCode = "ups"
	// CarrierCodeUSPS identifies the USPS carrier.
	CarrierCodeUSPS CarrierCode = "usps"
	// CarrierCodeWillCall identifies the will-call carrier.
	CarrierCodeWillCall CarrierCode = "will_call"
	// CarrierCodeDelivery identifies the delivery carrier.
	CarrierCodeDelivery CarrierCode = "delivery"
	// CarrierCodeLTL identifies the LTL carrier.
	CarrierCodeLTL CarrierCode = "ltl"
	// CarrierCodeLTL1 identifies the LTL1 carrier.
	CarrierCodeLTL1 CarrierCode = "ltl1"
	// CarrierCodeFreightCollect identifies the freight collect carrier.
	CarrierCodeFreightCollect CarrierCode = "freight_collect"
)

// IsShippoCarrier returns true if the given code corresponds to a carrier
// managed through the Shippo API (FedEx, UPS, USPS).
func IsShippoCarrier(code *string) bool {
	if code == nil {
		return false
	}
	switch CarrierCode(*code) {
	case CarrierCodeFedEx, CarrierCodeUPS, CarrierCodeUSPS:
		return true
	default:
		return false
	}
}

func (m CarrierCode) IsValid() bool {
	switch m {
	case CarrierCodeFedEx, CarrierCodeUPS, CarrierCodeUSPS, CarrierCodeWillCall, CarrierCodeDelivery, CarrierCodeLTL, CarrierCodeLTL1, CarrierCodeFreightCollect:
		return true
	default:
		return false
	}
}

func (m CarrierCode) EnumValues() []string {
	return []string{string(CarrierCodeFedEx), string(CarrierCodeUPS), string(CarrierCodeUSPS), string(CarrierCodeWillCall), string(CarrierCodeDelivery), string(CarrierCodeLTL), string(CarrierCodeLTL1), string(CarrierCodeFreightCollect)}
}
