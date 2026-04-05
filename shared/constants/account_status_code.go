package constants

// AccountStatusCode represents the status code of an account.
type AccountStatusCode string

const (
	// AccountStatusCodeNormal indicates a normal account status.
	AccountStatusCodeNormal AccountStatusCode = "normal"
	// AccountStatusCodePreferred indicates a preferred account status.
	AccountStatusCodePreferred AccountStatusCode = "preferred"
	// AccountStatusCodeHoldShipment indicates that shipments are on hold.
	AccountStatusCodeHoldShipment AccountStatusCode = "hold_shipment"
	// AccountStatusCodeHoldAll indicates that all activity is on hold.
	AccountStatusCodeHoldAll AccountStatusCode = "hold_all"
)

func (m AccountStatusCode) IsValid() bool {
	switch m {
	case AccountStatusCodeNormal, AccountStatusCodePreferred, AccountStatusCodeHoldShipment, AccountStatusCodeHoldAll:
		return true
	default:
		return false
	}
}

func (m AccountStatusCode) EnumValues() []string {
	return []string{
		string(AccountStatusCodeNormal),
		string(AccountStatusCodePreferred),
		string(AccountStatusCodeHoldShipment),
		string(AccountStatusCodeHoldAll),
	}
}
