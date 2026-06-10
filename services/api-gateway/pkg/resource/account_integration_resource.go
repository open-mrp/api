package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleAccountIntegrationID = "ai_0177772eae113431f64d473124"
const SampleAccountIntegrationName = "My Stripe Integration"

// Third-party integration connected to an account.
type AccountIntegration struct {
	// Account integration ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=account_integration"`
	// Display name of the integration.
	Name string `json:"name" validate:"required"`
	// Integration provider code.
	IntegrationCode constants.IntegrationCode `json:"provider" validate:"required"`
	// Whether the integration is active.
	IsActive bool `json:"is_active"`
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
	IsActive:        true,
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
	// Stripe publishable key.
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
	HasStripeIntegration bool `json:"has_stripe_integration"`
}

var SampleStripeStatus = &StripeStatus{
	Object:               constants.ObjectTypeStripeStatus,
	HasStripeIntegration: true,
}

func (*StripeStatus) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleStripeStatus)
}
