package constants

// CarrierConnectionStatus is how far a carrier account has gotten through OAuth authorization.
type CarrierConnectionStatus string

const (
	// CarrierConnectionStatusConnected means the account's own carrier account is authorized, for live rating and label purchase.
	CarrierConnectionStatusConnected CarrierConnectionStatus = "connected"
	// CarrierConnectionStatusAuthorizationPending means a carrier account exists but is still the shared default one, so authorization has not been completed.
	CarrierConnectionStatusAuthorizationPending CarrierConnectionStatus = "authorization_pending"
	// CarrierConnectionStatusDisconnected means there is no carrier account to authorize, or it could not be reached. Sandbox accounts always report this.
	CarrierConnectionStatusDisconnected CarrierConnectionStatus = "disconnected"
)

func (s CarrierConnectionStatus) IsValid() bool {
	switch s {
	case CarrierConnectionStatusConnected, CarrierConnectionStatusAuthorizationPending, CarrierConnectionStatusDisconnected:
		return true
	default:
		return false
	}
}

func (s CarrierConnectionStatus) EnumValues() []string {
	return []string{
		string(CarrierConnectionStatusConnected),
		string(CarrierConnectionStatusAuthorizationPending),
		string(CarrierConnectionStatusDisconnected),
	}
}
