package domain

import (
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
	PrivateKey     string `json:"privateKey"` // #nosec G117 -- field carries encrypted credentials, not a hardcoded secret
	PublishableKey string `json:"publishableKey"`
	WebhookSecret  string `json:"webhookSecret"`
}

// ShippoCredentials holds the parsed Shippo credential fields used for validation.
type ShippoCredentials struct {
	APIKey string `json:"apiKey"` // #nosec G117 -- field carries encrypted credentials, not a hardcoded secret
}
