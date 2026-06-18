package constants

// StripeConnectionStatus represents whether an account has a Stripe integration configured.
type StripeConnectionStatus string

const (
	// StripeConnectionStatusConnected indicates the account has a Stripe integration on file.
	StripeConnectionStatusConnected StripeConnectionStatus = "connected"
	// StripeConnectionStatusNotConnected indicates the account has no Stripe integration.
	StripeConnectionStatusNotConnected StripeConnectionStatus = "not_connected"
)

func (m StripeConnectionStatus) IsValid() bool {
	switch m {
	case StripeConnectionStatusConnected, StripeConnectionStatusNotConnected:
		return true
	default:
		return false
	}
}

func (m StripeConnectionStatus) EnumValues() []string {
	return []string{string(StripeConnectionStatusConnected), string(StripeConnectionStatusNotConnected)}
}

// StripeConnectionStatusFromExists maps the existence boolean to its public status value.
func StripeConnectionStatusFromExists(exists bool) StripeConnectionStatus {
	if exists {
		return StripeConnectionStatusConnected
	}
	return StripeConnectionStatusNotConnected
}
