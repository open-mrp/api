package apiresource

import (
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/timeutil"
)

const SampleAccountIntegrationID = "acig_5ilahyezrs63"
const SampleAccountIntegrationName = "My Stripe Integration"

// Third-party integration connected to an account.
//
// An account can have at most one integration per provider. The credentials supplied when the integration was connected are encrypted at rest and are never returned by the API.
type AccountIntegration struct {
	// Account integration ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account_integration"`
	// Display name of the integration.
	Name string `json:"name" validate:"required"`
	// Integration provider code.
	//
	// - `stripe`: Stripe payment processing.
	// - `shippo`: Shippo shipping and label generation.
	// - `hubspot`: HubSpot CRM.
	// - `openrouter`: OpenRouter, the LLM gateway that serves agent runs.
	IntegrationCode constants.IntegrationCode `json:"provider" validate:"required"`
	// Lifecycle status of the integration.
	//
	// Integrations are created `active`. Setting an integration to `inactive` keeps its stored credentials but stops it from being used (for example, the Stripe publishable key cannot be retrieved while the Stripe integration is inactive).
	Status constants.AccountIntegrationStatus `json:"status" validate:"required"`
	// Most the provider may spend for the account in a UTC calendar month, or `null` for no limit.
	//
	// Only `openrouter` integrations take a limit. Agent runs stop starting once OpenRouter reports this much spent this month; a run already underway finishes its current turn.
	MonthlySpendLimit *Money `json:"monthly_spend_limit"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleAccountIntegration = &AccountIntegration{
	ID:              SampleAccountIntegrationID,
	Object:          constants.ObjectTypeAccountIntegration,
	Name:            SampleAccountIntegrationName,
	IntegrationCode: constants.IntegrationCodeStripe,
	Status:          constants.AccountIntegrationStatusActive,
	CreatedAt:       timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:       timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*AccountIntegration) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAccountIntegration)
}

// Stripe publishable key for an account.
type StripePublishableKey struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=stripe_publishable_key"`
	// The publishable key (`pk_...`) from the account's Stripe integration, safe to use in client-side code.
	PublishableKey string `json:"publishable_key" validate:"required"`
}

var SampleStripePublishableKey = &StripePublishableKey{
	Object:         constants.ObjectTypeStripePublishableKey,
	PublishableKey: "pk_test_example123",
}

func (*StripePublishableKey) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleStripePublishableKey)
}

// Stripe integration status for an account.
type StripeStatus struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=stripe_status"`
	// Whether a Stripe integration is configured.
	//
	// `connected` if the account has a Stripe integration on file, regardless of whether the integration is currently active. The stored keys are not checked against Stripe, so `connected` does not guarantee that payments will succeed.
	Status constants.StripeConnectionStatus `json:"status" validate:"required"`
}

var SampleStripeStatus = &StripeStatus{
	Object: constants.ObjectTypeStripeStatus,
	Status: constants.StripeConnectionStatusConnected,
}

func (*StripeStatus) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleStripeStatus)
}

// What an account has spent through its OpenRouter integration.
//
// Spend comes from OpenRouter and covers every request made with the integration's API key, including any made outside OpenMRP. Days, weeks and months are UTC calendar periods.
type OpenRouterUsage struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=openrouter_usage"`
	// Spent today.
	SpentToday Money `json:"spent_today" validate:"required"`
	// Spent this week.
	SpentThisWeek Money `json:"spent_this_week" validate:"required"`
	// Spent this month, which is what `monthly_spend_limit` is measured against.
	SpentThisMonth Money `json:"spent_this_month" validate:"required"`
	// Spent since the API key was created.
	SpentTotal Money `json:"spent_total" validate:"required"`
	// The integration's monthly spend limit, or `null` for no limit.
	MonthlySpendLimit *Money `json:"monthly_spend_limit"`
	// What is left of `monthly_spend_limit` this month, never below zero, or `null` when there is no limit.
	MonthlySpendRemaining *Money `json:"monthly_spend_remaining"`
	// The credit limit set on the API key in OpenRouter itself, or `null` when the key has none.
	KeyLimit *Money `json:"key_limit"`
	// What is left of `key_limit`, or `null` when the key has no limit.
	KeyLimitRemaining *Money `json:"key_limit_remaining"`
}

var SampleOpenRouterUsage = &OpenRouterUsage{
	Object:                constants.ObjectTypeOpenRouterUsage,
	SpentToday:            Money{Amount: "1.25", Currency: "usd"},
	SpentThisWeek:         Money{Amount: "8.4", Currency: "usd"},
	SpentThisMonth:        Money{Amount: "31.9", Currency: "usd"},
	SpentTotal:            Money{Amount: "212.6", Currency: "usd"},
	MonthlySpendLimit:     &Money{Amount: "100", Currency: "usd"},
	MonthlySpendRemaining: &Money{Amount: "68.1", Currency: "usd"},
	KeyLimit:              nil,
	KeyLimitRemaining:     nil,
}

func (*OpenRouterUsage) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleOpenRouterUsage)
}
