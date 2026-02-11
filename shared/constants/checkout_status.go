package constants

// CheckoutStatus represents the status of a checkout session.
type CheckoutStatus string

const (
	// CheckoutStatusOpen indicates that the checkout session is open and active.
	CheckoutStatusOpen CheckoutStatus = "open"
	// CheckoutStatusComplete indicates that the checkout session is complete and successful.
	CheckoutStatusComplete CheckoutStatus = "complete"
	// CheckoutStatusExpired indicates that the checkout session is expired and has timed out.
	CheckoutStatusExpired CheckoutStatus = "expired"
)

func (s CheckoutStatus) IsValid() bool {
	switch s {
	case CheckoutStatusOpen, CheckoutStatusComplete, CheckoutStatusExpired:
		return true
	default:
		return false
	}
}

func (s CheckoutStatus) EnumValues() []string {
	return []string{string(CheckoutStatusOpen), string(CheckoutStatusComplete), string(CheckoutStatusExpired)}
}
