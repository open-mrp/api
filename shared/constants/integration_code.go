package constants

// IntegrationCode identifies a third-party integration provider.
type IntegrationCode string

const (
	// IntegrationCodeStripe identifies the Stripe payment integration.
	IntegrationCodeStripe IntegrationCode = "stripe"
	// IntegrationCodeShippo identifies the Shippo shipping integration.
	IntegrationCodeShippo IntegrationCode = "shippo"
)

func (m IntegrationCode) IsValid() bool {
	switch m {
	case IntegrationCodeStripe, IntegrationCodeShippo:
		return true
	default:
		return false
	}
}

func (m IntegrationCode) EnumValues() []string {
	return []string{string(IntegrationCodeStripe), string(IntegrationCodeShippo)}
}
