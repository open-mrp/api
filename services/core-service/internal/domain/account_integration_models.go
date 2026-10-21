package domain

import (
	"encoding/json"
	"time"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/pagination"
)

type AccountIntegration struct {
	ID              string
	AccountID       string
	IntegrationCode constants.IntegrationCode `audit:"integration_code"`
	Name            string                    `audit:"name"`
	IsActive        bool                      `audit:"is_active"`
	// MonthlySpendLimit is nil when the integration has no cap; only OpenRouter integrations take one.
	MonthlySpendLimit *Money `audit:"monthly_spend_limit"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// Money is a decimal amount in a lowercase ISO 4217 currency.
type Money struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
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
	// MonthlySpendLimit, when set, replaces the integration's cap; nil leaves an existing cap as it is.
	MonthlySpendLimit *Money
}

type UpdateAccountIntegrationParams struct {
	AccountID string
	ID        string
	Name      *string
	IsActive  *bool
	// SetMonthlySpendLimit replaces the cap with MonthlySpendLimit, which nil clears.
	SetMonthlySpendLimit bool
	MonthlySpendLimit    *Money
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

// StoredIntegrationCredentials is an integration's encrypted credentials with the settings read alongside them.
type StoredIntegrationCredentials struct {
	Encrypted         string
	IsActive          bool
	MonthlySpendLimit *Money
}

// OpenRouterCredentials holds the parsed OpenRouter credential fields.
type OpenRouterCredentials struct {
	APIKey string `json:"api_key"` // #nosec G117 -- field carries encrypted credentials, not a hardcoded secret
}

// OpenRouterKeyUsage is what OpenRouter reports an API key has spent, in US dollars. KeyLimit and KeyLimitRemaining are nil when the key itself has no credit limit.
type OpenRouterKeyUsage struct {
	SpentToday        string
	SpentThisWeek     string
	SpentThisMonth    string
	SpentTotal        string
	KeyLimit          *string
	KeyLimitRemaining *string
}

// OpenRouterUsage is an account's OpenRouter spend alongside its own monthly cap. MonthlySpendRemaining is nil when there is no cap.
type OpenRouterUsage struct {
	Key                   OpenRouterKeyUsage
	MonthlySpendLimit     *Money
	MonthlySpendRemaining *Money
}

// OpenRouterAccess is what an agent run needs to call OpenRouter for an account.
type OpenRouterAccess struct {
	// Configured is false when the account has no OpenRouter integration, leaving the deployment's own key, if any, to serve it.
	Configured bool
	IsActive   bool
	APIKey     string
	// MonthlySpendLimit and SpentThisMonth are set together, and only when the integration has a cap.
	MonthlySpendLimit *Money
	SpentThisMonth    *string
	LimitReached      bool
}
