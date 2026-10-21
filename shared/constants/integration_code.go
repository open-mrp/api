package constants

// IntegrationCode identifies a third-party integration provider.
type IntegrationCode string

const (
	// IntegrationCodeStripe identifies the Stripe payment integration.
	IntegrationCodeStripe IntegrationCode = "stripe"
	// IntegrationCodeShippo identifies the Shippo shipping integration.
	IntegrationCodeShippo IntegrationCode = "shippo"
	// IntegrationCodeHubspot identifies the HubSpot CRM integration.
	IntegrationCodeHubspot IntegrationCode = "hubspot"
	// IntegrationCodeOpenRouter identifies the OpenRouter LLM gateway that serves agent runs.
	IntegrationCodeOpenRouter IntegrationCode = "openrouter"
)

func (m IntegrationCode) IsValid() bool {
	switch m {
	case IntegrationCodeStripe, IntegrationCodeShippo, IntegrationCodeHubspot, IntegrationCodeOpenRouter:
		return true
	default:
		return false
	}
}

func (m IntegrationCode) EnumValues() []string {
	return []string{string(IntegrationCodeStripe), string(IntegrationCodeShippo), string(IntegrationCodeHubspot), string(IntegrationCodeOpenRouter)}
}
