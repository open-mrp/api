package domain

import (
	"encoding/json"
	"time"

	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/pagination"
)

type AccountIntegration struct {
	ID              string
	AccountID       string
	IntegrationCode constants.IntegrationCode `audit:"integration_code"`
	Name            string                    `audit:"name"`
	IsActive        bool                      `audit:"is_active"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type ListAccountIntegrationsParams struct {
	AccountID string
	Cursor    *string
	Limit     int32
	Query     *string
}

type ListAccountIntegrationsResult struct {
	AccountIntegrations []*AccountIntegration
	PageInfo            pagination.PageInfo
}

type CreateAccountIntegrationParams struct {
	AccountID       string
	IntegrationCode constants.IntegrationCode
	Name            string
	Credentials     string // raw JSON credentials before encryption
}

type UpdateAccountIntegrationParams struct {
	AccountID string
	ID        string
	Name      *string
	IsActive  *bool
}

type DeleteAccountIntegrationParams struct {
	AccountID string
	ID        string
}

// StripeCredentials holds the parsed Stripe credential fields used for validation.
type StripeCredentials struct {
	PrivateKey     string `json:"private_key"` // #nosec G117 -- field carries encrypted credentials, not a hardcoded secret
	PublishableKey string `json:"publishable_key"`
	WebhookSecret  string `json:"webhook_secret"`
}

// UnmarshalJSON accepts both the canonical snake_case keys and the legacy camelCase keys (privateKey/publishableKey/webhookSecret) so credentials stored by the legacy dashboard and the v2 Go service can be read interchangeably while both write paths coexist.
func (c *StripeCredentials) UnmarshalJSON(data []byte) error {
	var raw struct {
		PrivateKey          string `json:"private_key"` // #nosec G117 -- transient JSON decode field, not a hardcoded secret
		PrivateKeyLegacy    string `json:"privateKey"`  // #nosec G117 -- transient JSON decode field, not a hardcoded secret
		PublishableKey      string `json:"publishable_key"`
		PublishableLegacy   string `json:"publishableKey"`
		WebhookSecret       string `json:"webhook_secret"`
		WebhookSecretLegacy string `json:"webhookSecret"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	c.PrivateKey = firstNonEmpty(raw.PrivateKey, raw.PrivateKeyLegacy)
	c.PublishableKey = firstNonEmpty(raw.PublishableKey, raw.PublishableLegacy)
	c.WebhookSecret = firstNonEmpty(raw.WebhookSecret, raw.WebhookSecretLegacy)
	return nil
}

// ShippoCredentials holds the parsed Shippo credential fields used for validation.
type ShippoCredentials struct {
	APIKey string `json:"api_key"` // #nosec G117 -- field carries encrypted credentials, not a hardcoded secret
}

// UnmarshalJSON accepts both the canonical snake_case key and the legacy camelCase key (apiKey) so legacy and v2 stored credentials can be read interchangeably.
func (c *ShippoCredentials) UnmarshalJSON(data []byte) error {
	var raw struct {
		APIKey       string `json:"api_key"` // #nosec G117 -- transient JSON decode field, not a hardcoded secret
		APIKeyLegacy string `json:"apiKey"`  // #nosec G117 -- transient JSON decode field, not a hardcoded secret
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	c.APIKey = firstNonEmpty(raw.APIKey, raw.APIKeyLegacy)
	return nil
}

// firstNonEmpty returns the first non-empty string from values, or "" if none.
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// HubspotCredentials holds the parsed HubSpot credential fields used for validation.
type HubspotCredentials struct {
	AccessToken string `json:"access_token"` // #nosec G117 -- field carries encrypted credentials, not a hardcoded secret
}
