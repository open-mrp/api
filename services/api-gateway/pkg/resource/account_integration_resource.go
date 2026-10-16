package apiresource

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/timeutil"
)

const SampleAccountIntegrationID = "acig_5ilahyezrs63"
const SampleAccountIntegrationName = "My Stripe Integration"

// A provider connected to your account.
//
// One connection per provider. Its credentials are encrypted at rest and never returned.
type AccountIntegration struct {
	// Account integration ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account_integration"`
	// Display name of the integration.
	Name string `json:"name" validate:"required"`
	// The connected provider.
	//
	// - `stripe`: Stripe payment processing.
	// - `shippo`: Shippo shipping rates and labels.
	// - `hubspot`: HubSpot CRM.
	IntegrationCode constants.IntegrationCode `json:"provider" validate:"required"`
	// Whether OpenMRP uses this integration: `active` or `disabled`.
	//
	// A disabled integration keeps its credentials but isn't used until it is set back to `active`.
	Status constants.PauseStatus `json:"status" validate:"required"`
	// Whether the last call to the provider succeeded.
	//
	// - `connected`: the provider accepted the credentials.
	// - `invalid_credentials`: the provider rejected them; rotate the credentials.
	// - `unknown`: OpenMRP hasn't called the provider yet.
	ConnectionStatus constants.IntegrationConnectionStatus `json:"connection_status" validate:"required"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleAccountIntegration = &AccountIntegration{
	ID:               SampleAccountIntegrationID,
	Object:           constants.ObjectTypeAccountIntegration,
	Name:             SampleAccountIntegrationName,
	IntegrationCode:  constants.IntegrationCodeStripe,
	Status:           constants.PauseStatusActive,
	ConnectionStatus: constants.IntegrationConnectionStatusConnected,
	CreatedAt:        timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:        timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
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
