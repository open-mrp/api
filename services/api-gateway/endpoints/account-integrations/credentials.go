package accountintegrationep

import (
	"encoding/json"
	"fmt"

	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/field"
)

// Credentials for an integration's provider. Which keys it takes depends on the provider:
//
// - `stripe`: `private_key` (`sk_...`), `publishable_key` (`pk_...`) and `webhook_secret` (`whsec_...`).
// - `shippo`: `api_key` (`shippo_live_...` or `shippo_test_...`).
// - `hubspot`: `access_token` (a Private App token, `pat-...`).
//
// Every key the provider takes is required, and a key it doesn't take is rejected. Credentials are encrypted at rest and never returned or logged.
type IntegrationCredentials struct {
	// Stripe secret key (`sk_live_...` or `sk_test_...`).
	PrivateKey field.Optional[string] `json:"private_key,omitzero"` // #nosec G117 -- request field carrying a caller's credential, not a hardcoded secret
	// Stripe publishable key (`pk_live_...` or `pk_test_...`).
	PublishableKey field.Optional[string] `json:"publishable_key,omitzero"`
	// Stripe webhook signing secret (`whsec_...`).
	WebhookSecret field.Optional[string] `json:"webhook_secret,omitzero"` // #nosec G117 -- request field carrying a caller's credential, not a hardcoded secret
	// Shippo API key (`shippo_live_...` or `shippo_test_...`).
	APIKey field.Optional[string] `json:"api_key,omitzero"` // #nosec G117 -- request field carrying a caller's credential, not a hardcoded secret
	// HubSpot Private App access token (`pat-...`).
	AccessToken field.Optional[string] `json:"access_token,omitzero"` // #nosec G117 -- request field carrying a caller's credential, not a hardcoded secret
}

// credentialKey is one key of IntegrationCredentials, named as the request names it.
type credentialKey struct {
	name  string
	value field.Optional[string]
}

func (c *IntegrationCredentials) keys() []credentialKey {
	return []credentialKey{
		{"private_key", c.PrivateKey},
		{"publishable_key", c.PublishableKey},
		{"webhook_secret", c.WebhookSecret},
		{"api_key", c.APIKey},
		{"access_token", c.AccessToken},
	}
}

// providerCredentialKeys are the keys each provider takes, all required.
var providerCredentialKeys = map[constants.IntegrationCode][]string{
	constants.IntegrationCodeStripe:  {"private_key", "publishable_key", "webhook_secret"},
	constants.IntegrationCodeShippo:  {"api_key"},
	constants.IntegrationCodeHubspot: {"access_token"},
}

// encode checks the credentials against what provider takes and returns them as the JSON core-service stores.
// Every failing key is reported at once.
func (c *IntegrationCredentials) encode(provider constants.IntegrationCode) (string, *apierror.APIError) {
	wanted, ok := providerCredentialKeys[provider]
	if !ok {
		return "", apierror.NewInvalidFormatError("provider", fmt.Sprintf("Unsupported provider %q.", provider))
	}
	takes := make(map[string]bool, len(wanted))
	for _, k := range wanted {
		takes[k] = true
	}

	var errs []apierror.FieldError
	out := make(map[string]string, len(wanted))
	for _, k := range c.keys() {
		v, set := k.value.Value()
		switch {
		case takes[k.name] && (!set || v == ""):
			errs = append(errs, apierror.Field("credentials."+k.name, apierror.CodeMissingField, fmt.Sprintf("A %s integration needs `%s`.", provider, k.name)))
		case !takes[k.name] && set:
			errs = append(errs, apierror.Field("credentials."+k.name, apierror.CodeInvalidFormat, fmt.Sprintf("A %s integration doesn't take `%s`.", provider, k.name)))
		case set:
			out[k.name] = v
		}
	}
	if len(errs) > 0 {
		return "", apierror.NewValidationError("The credentials don't match the provider.", errs...)
	}
	b, err := json.Marshal(out)
	if err != nil {
		return "", apierror.NewInternalError(err, "Failed to encode integration credentials.")
	}
	return string(b), nil
}
