package constants

// CustomerStanding is a customer's credit standing. Holds flag the customer's orders; they never reject them.
type CustomerStanding string

const (
	// CustomerStandingNormal has no restrictions.
	CustomerStandingNormal CustomerStanding = "normal"
	// CustomerStandingHoldShipment holds the customer's shipments; orders can still be placed.
	CustomerStandingHoldShipment CustomerStanding = "hold_shipment"
	// CustomerStandingHoldAll holds all of the customer's activity.
	CustomerStandingHoldAll CustomerStanding = "hold_all"
)

func (s CustomerStanding) IsValid() bool {
	switch s {
	case CustomerStandingNormal, CustomerStandingHoldShipment, CustomerStandingHoldAll:
		return true
	default:
		return false
	}
}

func (s CustomerStanding) EnumValues() []string {
	return []string{string(CustomerStandingNormal), string(CustomerStandingHoldShipment), string(CustomerStandingHoldAll)}
}

func (s *CustomerStanding) StringPtr() *string {
	if s == nil {
		return nil
	}
	v := string(*s)
	return &v
}

// CustomerStandingFromStatusCode is the standing a stored account status code holds. The retired `preferred` code
// has no restrictions, so it reads as normal.
func CustomerStandingFromStatusCode(code string) CustomerStanding {
	if s := CustomerStanding(code); s.IsValid() {
		return s
	}
	return CustomerStandingNormal
}
